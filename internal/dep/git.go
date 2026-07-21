package dep

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// CacheVersion is the cache layout version. Bumping this invalidates all
// existing caches and forces a fresh fetch, mirroring buf's cache version
// directory (e.g. ~/.cache/buf/<version>/...).
const CacheVersion = "v1"

// moduleProxyDir is the sub-namespace under <cacheDir>/<version>/ that holds
// git dependency clones, mirroring buf's "module-proxy" directory.
const moduleProxyDir = "module-proxy"

// GitDep declares a git proto dependency.
type GitDep struct {
	URL            string `yaml:"url"`
	Ref            string `yaml:"ref"`
	Subdir         string `yaml:"subdir"`
	ResolvedCommit string `yaml:"resolved_commit"`
}

// GitResolver clones a git repo and extracts proto files from subdir.
type GitResolver struct {
	dep            GitDep
	cacheDir       string
	cloneDir       string
	resolvedCommit string
	protoFiles     []string
	importPaths    []string
}

// NewGitResolver creates a GitResolver.
func NewGitResolver(dep GitDep, cacheDir string) *GitResolver {
	return &GitResolver{dep: dep, cacheDir: cacheDir}
}

// gitRefPattern validates git ref names to prevent command injection.
var gitRefPattern = regexp.MustCompile(`^[a-zA-Z0-9._/:-]+$`)

// Fetch clones the repo and extracts proto files.
//
// Cache layout (mirrors buf's module-proxy structure):
//
//	<cacheDir>/<CacheVersion>/module-proxy/commit/<host>/<owner>/<repo>/<commit>
//	<cacheDir>/<CacheVersion>/module-proxy/remote/<host>/<owner>/<repo>/<sanitised-ref>
//
// The "commit" namespace holds immutable, content-addressed clones keyed by
// the full commit SHA (from api.lock). Because a commit SHA is immutable,
// this guarantees the cached clone always matches the locked content — no
// staleness even when Ref is a moving branch like "master".
//
// The "remote" namespace holds ref-keyed clones used during the first run
// (before api.lock exists). After cloning we resolve HEAD and the caller is
// expected to persist it via WriteAPILock so subsequent runs use the
// immutable "commit" namespace.
//
// If the git URL cannot be parsed into host/owner/repo (e.g. a bare local
// filesystem path used in tests), the layout falls back to:
//
//	<cacheDir>/<CacheVersion>/module-proxy/local/<short-hash>
//
// where <short-hash> = SHA256(URL+key)[:16] for disambiguation.
func (r *GitResolver) Fetch() ([]string, error) {
	if err := validateGitInput(r.dep.URL); err != nil {
		return nil, fmt.Errorf("git url: %w", err)
	}
	if r.dep.Ref != "" && !gitRefPattern.MatchString(r.dep.Ref) {
		return nil, fmt.Errorf("invalid git ref %q: illegal characters", r.dep.Ref)
	}

	r.cloneDir = r.computeCloneDir()

	if _, err := os.Stat(r.cloneDir); os.IsNotExist(err) {
		if r.dep.ResolvedCommit != "" {
			if err := r.cloneByCommit(r.dep.ResolvedCommit); err != nil {
				return nil, err
			}
		} else {
			if err := r.clone(); err != nil {
				return nil, err
			}
		}
	}

	commit, err := r.resolveCommit()
	if err != nil {
		return nil, err
	}
	r.resolvedCommit = commit

	// If we cloned by ref (no locked commit) but the resolved HEAD differs
	// from a previously recorded commit, the cache may be stale for moving
	// refs. For immutability, when ResolvedCommit was provided we additionally
	// verify the cloned HEAD matches it; mismatch indicates cache corruption.
	if r.dep.ResolvedCommit != "" && r.dep.ResolvedCommit != commit {
		// Cache corruption or race: re-clone from scratch.
		os.RemoveAll(r.cloneDir)
		if err := r.cloneByCommit(r.dep.ResolvedCommit); err != nil {
			return nil, err
		}
		commit = r.dep.ResolvedCommit
		r.resolvedCommit = commit
	}

	subdirPath := r.cloneDir
	if r.dep.Subdir != "" {
		subdirPath = filepath.Join(r.cloneDir, r.dep.Subdir)
	}
	if _, err := os.Stat(subdirPath); err != nil {
		return nil, fmt.Errorf("subdir %q not found in repo: %w", r.dep.Subdir, err)
	}

	var protoFiles []string
	err = filepath.WalkDir(subdirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".proto" {
			protoFiles = append(protoFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk subdir: %w", err)
	}
	if len(protoFiles) == 0 {
		return nil, fmt.Errorf("no .proto files found in subdir %q", r.dep.Subdir)
	}

	r.protoFiles = protoFiles
	// Register the subdir path (or clone root when subdir is empty) as the sole
	// import path. Proto files within use import paths relative to this root.
	// For example, a file at <clone>/google/api/annotations.proto is imported
	// as "google/api/annotations.proto", so the import root must be <clone>/.
	// Previously we registered each unique directory containing .proto files,
	// which broke imports when proto files had nested package paths.
	r.importPaths = []string{subdirPath}
	return r.importPaths, nil
}

// computeCloneDir builds the cache directory path for this dependency,
// following buf's module-proxy layout. See Fetch docs for the full spec.
func (r *GitResolver) computeCloneDir() string {
	base := filepath.Join(r.cacheDir, CacheVersion, moduleProxyDir)
	subpath := gitCacheSubpath(r.dep.URL)

	// Fallback for unparseable URLs (e.g. local filesystem paths in tests):
	// use a flat hash under the "local" namespace.
	if len(subpath) == 0 {
		key := r.dep.ResolvedCommit
		if key == "" {
			key = r.dep.Ref
		}
		h := sha256Short(r.dep.URL + key)
		return filepath.Join(base, "local", h)
	}

	if r.dep.ResolvedCommit != "" {
		// Immutable, content-addressed by commit SHA.
		return filepath.Join(append([]string{base, "commit"}, append(subpath, r.dep.ResolvedCommit)...)...)
	}

	// Ref-keyed (moving ref). Sanitise the ref for filesystem safety.
	ref := r.dep.Ref
	if ref == "" {
		ref = "HEAD"
	}
	sanitisedRef := sanitisePathSegment(ref)
	return filepath.Join(append([]string{base, "remote"}, append(subpath, sanitisedRef)...)...)
}

// sha256Short returns the first 16 hex characters of SHA256(s).
func sha256Short(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])[:16]
}

func (r *GitResolver) clone() error {
	if err := os.MkdirAll(filepath.Dir(r.cloneDir), 0755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}
	ref := r.dep.Ref
	if ref == "" {
		ref = "HEAD"
	}
	args := []string{"clone", "--depth", "1"}
	if ref != "HEAD" {
		args = append(args, "--branch", ref)
	}
	args = append(args, r.dep.URL, r.cloneDir)
	cmd := exec.Command("git", args...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if output, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(r.cloneDir)
		fullArgs := []string{"clone", r.dep.URL, r.cloneDir}
		cmd2 := exec.Command("git", fullArgs...)
		cmd2.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		if output2, err := cmd2.CombinedOutput(); err != nil {
			return fmt.Errorf("git clone failed: %w\n%s", err, string(output2))
		}
		_ = output
		if ref != "HEAD" {
			checkout := exec.Command("git", "checkout", ref)
			checkout.Dir = r.cloneDir
			if output3, err := checkout.CombinedOutput(); err != nil {
				return fmt.Errorf("git checkout %s: %w\n%s", ref, err, string(output3))
			}
		}
	}
	return nil
}

// cloneByCommit performs a full clone and checks out the specific commit SHA.
// Used when ResolvedCommit is known and the ref-based cache is absent/stale.
func (r *GitResolver) cloneByCommit(commit string) error {
	if err := os.MkdirAll(filepath.Dir(r.cloneDir), 0755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}
	args := []string{"clone", r.dep.URL, r.cloneDir}
	cmd := exec.Command("git", args...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if output, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(r.cloneDir)
		return fmt.Errorf("git clone failed: %w\n%s", err, string(output))
	}
	checkout := exec.Command("git", "checkout", commit)
	checkout.Dir = r.cloneDir
	if output, err := checkout.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout %s: %w\n%s", commit, err, string(output))
	}
	return nil
}

func (r *GitResolver) resolveCommit() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = r.cloneDir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// ResolvedCommit returns the commit SHA after Fetch.
func (r *GitResolver) ResolvedCommit() string {
	return r.resolvedCommit
}

// ProtoFiles returns the extracted .proto file paths.
func (r *GitResolver) ProtoFiles() ([]string, error) {
	if r.protoFiles == nil {
		return nil, fmt.Errorf("Fetch not called")
	}
	return r.protoFiles, nil
}

// validateGitInput validates a git URL for subprocess safety.
func validateGitInput(url string) error {
	for _, ch := range url {
		if !(ch == '.' || ch == '/' || ch == ':' || ch == '-' || ch == '_' ||
			(ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') ||
			ch == '@') {
			return fmt.Errorf("illegal character %q in git URL", ch)
		}
	}
	return nil
}

// sanitisePathSegment keeps only filesystem-safe characters ([a-zA-Z0-9._-])
// and strips a trailing ".git". This is used for both URL-derived cache path
// segments and git refs, preventing path traversal or illegal characters
// from reaching the filesystem.
func sanitisePathSegment(s string) string {
	s = strings.TrimSuffix(s, ".git")
	var b strings.Builder
	for _, ch := range s {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '.' || ch == '-' || ch == '_' {
			b.WriteRune(ch)
		}
	}
	return b.String()
}

// gitCacheSubpath parses a git URL and returns a human-readable relative path
// slice (e.g. ["github.com", "googleapis", "googleapis"]) suitable for nesting
// under the git cache root. This mirrors buf's module-proxy layout where the
// cache directory preserves host/owner/repo for debuggability.
//
// Supported URL forms:
//   - https://github.com/owner/repo.git         → ["github.com", "owner", "repo"]
//   - http://github.com/owner/repo              → ["github.com", "owner", "repo"]
//   - git@github.com:owner/repo.git             → ["github.com", "owner", "repo"]
//   - ssh://git@gitlab.com:2222/owner/repo.git  → ["gitlab.com", "owner", "repo"]
//
// If the URL cannot be parsed into at least host/owner/repo (e.g. a bare local
// filesystem path like /tmp/remote.git used in tests), an empty slice is
// returned and the caller falls back to a flat <short-hash> cache entry.
func gitCacheSubpath(gitURL string) []string {
	var host, path string

	// Detect the scheme to decide how to split host from path.
	switch {
	case strings.HasPrefix(gitURL, "https://"),
		strings.HasPrefix(gitURL, "http://"),
		strings.HasPrefix(gitURL, "ssh://"),
		strings.HasPrefix(gitURL, "git://"),
		strings.HasPrefix(gitURL, "ftp://"):
		// Scheme-based URL: use net/url for robust parsing.
		// We avoid url.Parse directly because git URLs with ports/userinfo
		// are handled correctly by it.
		u, err := url.Parse(gitURL)
		if err != nil || u.Host == "" {
			return nil
		}
		host = u.Hostname() // strips port and userinfo
		path = u.Path
	case strings.Contains(gitURL, "@") && strings.Contains(gitURL, ":"):
		// SCP-style: user@host:owner/repo.git
		// Strip userinfo.
		rest := gitURL
		if _, after, found := strings.Cut(rest, "@"); found {
			rest = after
		}
		// First ':' separates host from path.
		if h, p, found := strings.Cut(rest, ":"); found {
			host = h
			path = p
		} else {
			return nil
		}
	default:
		// Bare local path — not a remote git URL we can decompose.
		return nil
	}

	// Strip .git suffix and query/fragment from path.
	path = strings.TrimSuffix(path, ".git")
	if i := strings.IndexAny(path, "?#"); i >= 0 {
		path = path[:i]
	}

	host = sanitisePathSegment(host)
	if host == "" {
		return nil
	}

	var result []string
	result = append(result, host)
	for p := range strings.SplitSeq(strings.Trim(path, "/"), "/") {
		p = sanitisePathSegment(p)
		if p != "" {
			result = append(result, p)
		}
	}

	// Need at least host/owner/repo (3 segments) for a meaningful layout.
	if len(result) < 3 {
		return nil
	}
	// Keep only the first 3 segments (host, owner, repo); deeper paths are
	// collapsed into the hash suffix to avoid excessively long cache paths.
	if len(result) > 3 {
		result = result[:3]
	}
	return result
}

// WriteAPILock writes git dependencies to api.lock file.
func WriteAPILock(path string, deps []GitDep) error {
	data := struct {
		GitDeps []GitDep `yaml:"git_deps"`
	}{GitDeps: deps}
	out, err := yaml.Marshal(&data)
	if err != nil {
		return fmt.Errorf("marshal api.lock: %w", err)
	}
	header := []byte("# api.lock — 由 apigen 自动生成，请勿手改\n")
	return os.WriteFile(path, append(header, out...), 0644)
}

// ReadAPILock reads git dependencies from api.lock file.
func ReadAPILock(path string) ([]GitDep, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read api.lock: %w", err)
	}
	var parsed struct {
		GitDeps []GitDep `yaml:"git_deps"`
	}
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("unmarshal api.lock: %w", err)
	}
	return parsed.GitDeps, nil
}
