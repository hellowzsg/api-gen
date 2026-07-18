package dep

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

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
func (r *GitResolver) Fetch() ([]string, error) {
	if err := validateGitInput(r.dep.URL); err != nil {
		return nil, fmt.Errorf("git url: %w", err)
	}
	if r.dep.Ref != "" && !gitRefPattern.MatchString(r.dep.Ref) {
		return nil, fmt.Errorf("invalid git ref %q: illegal characters", r.dep.Ref)
	}

	h := sha256.Sum256([]byte(r.dep.URL + r.dep.Ref))
	r.cloneDir = filepath.Join(r.cacheDir, hex.EncodeToString(h[:])[:16])

	if _, err := os.Stat(r.cloneDir); os.IsNotExist(err) {
		if err := r.clone(); err != nil {
			return nil, err
		}
	}

	commit, err := r.resolveCommit()
	if err != nil {
		return nil, err
	}
	r.resolvedCommit = commit

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
	seen := make(map[string]bool)
	for _, f := range protoFiles {
		d := filepath.Dir(f)
		if !seen[d] {
			seen[d] = true
			r.importPaths = append(r.importPaths, d)
		}
	}
	sort.Strings(r.importPaths)
	return r.importPaths, nil
}

func (r *GitResolver) clone() error {
	if err := os.MkdirAll(r.cacheDir, 0755); err != nil {
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
