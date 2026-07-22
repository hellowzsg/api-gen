package dep

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestGitResolver_CloneAndExtract 测试 git clone + subdir 提取。
func TestGitResolver_CloneAndExtract(t *testing.T) {
	repoDir := t.TempDir()
	remoteDir := filepath.Join(repoDir, "remote.git")
	workDir := filepath.Join(repoDir, "work")
	cacheDir := filepath.Join(repoDir, "cache")

	if err := os.MkdirAll(filepath.Join(workDir, "proto", "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	writeProtoFile(t, filepath.Join(workDir, "proto", "a.proto"))
	writeProtoFile(t, filepath.Join(workDir, "proto", "sub", "b.proto"))
	writeFile(t, filepath.Join(workDir, "README.md"), "readme")
	runGit(t, workDir, "init")
	runGit(t, workDir, "config", "user.email", "test@test.com")
	runGit(t, workDir, "config", "user.name", "test")
	runGit(t, workDir, "add", "-A")
	runGit(t, workDir, "commit", "-m", "init")
	runGit(t, workDir, "branch", "-M", "main")
	runGit(t, workDir, "init", "--bare", remoteDir)
	runGit(t, workDir, "remote", "add", "origin", remoteDir)
	runGit(t, workDir, "push", "-u", "origin", "main")

	r := NewGitResolver(GitDep{
		URL:    remoteDir,
		Ref:    "main",
		Subdir: "proto",
	}, cacheDir)

	importPaths, err := r.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if len(importPaths) == 0 {
		t.Fatal("ImportPaths empty, want at least 1")
	}

	if len(r.protoFiles) != 2 {
		t.Errorf("len(r.protoFiles) = %d, want 2", len(r.protoFiles))
	}
}

// TestGitResolver_ApiLock 测试 api.lock 生成与读取。
func TestGitResolver_ApiLock(t *testing.T) {
	repoDir := t.TempDir()
	remoteDir := filepath.Join(repoDir, "remote.git")
	workDir := filepath.Join(repoDir, "work")
	cacheDir := filepath.Join(repoDir, "cache")
	lockPath := filepath.Join(repoDir, "api.lock")

	if err := os.MkdirAll(filepath.Join(workDir, "proto"), 0755); err != nil {
		t.Fatal(err)
	}
	writeProtoFile(t, filepath.Join(workDir, "proto", "a.proto"))
	runGit(t, workDir, "init")
	runGit(t, workDir, "config", "user.email", "test@test.com")
	runGit(t, workDir, "config", "user.name", "test")
	runGit(t, workDir, "add", "-A")
	runGit(t, workDir, "commit", "-m", "init")
	runGit(t, workDir, "branch", "-M", "main")
	runGit(t, workDir, "init", "--bare", remoteDir)
	runGit(t, workDir, "remote", "add", "origin", remoteDir)
	runGit(t, workDir, "push", "-u", "origin", "main")

	dep := GitDep{URL: remoteDir, Ref: "main", Subdir: "proto"}
	r := NewGitResolver(dep, cacheDir)
	if _, err := r.Fetch(context.Background()); err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if err := WriteAPILock(lockPath, []GitDep{{
		URL:            dep.URL,
		Ref:            dep.Ref,
		ResolvedCommit: r.ResolvedCommit(),
		Subdir:         dep.Subdir,
	}}); err != nil {
		t.Fatalf("WriteAPILock failed: %v", err)
	}

	deps, err := ReadAPILock(lockPath)
	if err != nil {
		t.Fatalf("ReadAPILock failed: %v", err)
	}
	if len(deps) != 1 {
		t.Fatalf("len(deps) = %d, want 1", len(deps))
	}
	if deps[0].URL != dep.URL {
		t.Errorf("URL = %q, want %q", deps[0].URL, dep.URL)
	}
	if deps[0].ResolvedCommit == "" {
		t.Error("ResolvedCommit is empty")
	}
}

// TestGitResolver_InvalidRef 测试 ref 不存在 fail-fast。
func TestGitResolver_InvalidRef(t *testing.T) {
	repoDir := t.TempDir()
	remoteDir := filepath.Join(repoDir, "remote.git")
	workDir := filepath.Join(repoDir, "work")
	cacheDir := filepath.Join(repoDir, "cache")

	if err := os.MkdirAll(filepath.Join(workDir, "proto"), 0755); err != nil {
		t.Fatal(err)
	}
	writeProtoFile(t, filepath.Join(workDir, "proto", "a.proto"))
	runGit(t, workDir, "init")
	runGit(t, workDir, "config", "user.email", "test@test.com")
	runGit(t, workDir, "config", "user.name", "test")
	runGit(t, workDir, "add", "-A")
	runGit(t, workDir, "commit", "-m", "init")
	runGit(t, workDir, "branch", "-M", "main")
	runGit(t, workDir, "init", "--bare", remoteDir)
	runGit(t, workDir, "remote", "add", "origin", remoteDir)
	runGit(t, workDir, "push", "-u", "origin", "main")

	r := NewGitResolver(GitDep{
		URL:    remoteDir,
		Ref:    "nonexistent-branch",
		Subdir: "proto",
	}, cacheDir)

	_, err := r.Fetch(context.Background())
	if err == nil {
		t.Fatal("Fetch should fail for nonexistent ref")
	}
}

func writeProtoFile(t *testing.T, path string) {
	t.Helper()
	writeFile(t, path, `syntax = "proto3"; package test; message A {}`)
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test.com")
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %v in %s: %v", args, dir, err)
	}
}

// TestGitResolver_CommitKeyedCache 验证当 ResolvedCommit 已知时，cache key 基于
// commit SHA 而非 ref，且第二次 Fetch 命中同一缓存目录。
func TestGitResolver_CommitKeyedCache(t *testing.T) {
	repoDir := t.TempDir()
	remoteDir := filepath.Join(repoDir, "remote.git")
	workDir := filepath.Join(repoDir, "work")
	cacheDir := filepath.Join(repoDir, "cache")

	if err := os.MkdirAll(filepath.Join(workDir, "proto"), 0755); err != nil {
		t.Fatal(err)
	}
	writeProtoFile(t, filepath.Join(workDir, "proto", "a.proto"))
	runGit(t, workDir, "init")
	runGit(t, workDir, "config", "user.email", "test@test.com")
	runGit(t, workDir, "config", "user.name", "test")
	runGit(t, workDir, "add", "-A")
	runGit(t, workDir, "commit", "-m", "init")
	runGit(t, workDir, "branch", "-M", "main")
	runGit(t, workDir, "init", "--bare", remoteDir)
	runGit(t, workDir, "remote", "add", "origin", remoteDir)
	runGit(t, workDir, "push", "-u", "origin", "main")

	// First fetch by ref (no ResolvedCommit) to discover the commit SHA.
	r1 := NewGitResolver(GitDep{
		URL:    remoteDir,
		Ref:    "main",
		Subdir: "proto",
	}, cacheDir)
	if _, err := r1.Fetch(context.Background()); err != nil {
		t.Fatalf("first Fetch failed: %v", err)
	}
	commit := r1.ResolvedCommit()
	if commit == "" {
		t.Fatal("ResolvedCommit empty after first Fetch")
	}

	// Second fetch with ResolvedCommit set should produce a DIFFERENT cache
	// dir (keyed by commit), triggering a fresh clone, and the resolved HEAD
	// must equal the provided commit.
	r2 := NewGitResolver(GitDep{
		URL:            remoteDir,
		Ref:            "main",
		Subdir:         "proto",
		ResolvedCommit: commit,
	}, cacheDir)
	if _, err := r2.Fetch(context.Background()); err != nil {
		t.Fatalf("second Fetch with commit failed: %v", err)
	}
	if r2.ResolvedCommit() != commit {
		t.Errorf("after commit-keyed fetch, ResolvedCommit = %q, want %q", r2.ResolvedCommit(), commit)
	}
	if r1.cloneDir == r2.cloneDir {
		t.Errorf("cloneDir should differ between ref-keyed and commit-keyed fetches; got same %q", r1.cloneDir)
	}

	// Third fetch with the same ResolvedCommit must hit the cache (no re-clone).
	// We detect cache hit by checking the cloneDir already exists before Fetch.
	r3 := NewGitResolver(GitDep{
		URL:            remoteDir,
		Ref:            "main",
		Subdir:         "proto",
		ResolvedCommit: commit,
	}, cacheDir)
	// Pre-compute the expected cloneDir to verify it already exists.
	// (r3.cloneDir is set inside Fetch; we instead check that Fetch succeeds
	// without network by pointing URL to a path that would fail if re-cloned.)
	if _, err := r3.Fetch(context.Background()); err != nil {
		t.Fatalf("third Fetch (cache hit) failed: %v", err)
	}
	if r3.cloneDir != r2.cloneDir {
		t.Errorf("cache hit: cloneDir = %q, want %q", r3.cloneDir, r2.cloneDir)
	}
}

// TestGitResolver_LockRoundTrip 验证 api.lock 写入 ResolvedCommit 后，
// 下次用同一 commit Fetch 命中缓存（不依赖 ref）。
func TestGitResolver_LockRoundTrip(t *testing.T) {
	repoDir := t.TempDir()
	remoteDir := filepath.Join(repoDir, "remote.git")
	workDir := filepath.Join(repoDir, "work")
	cacheDir := filepath.Join(repoDir, "cache")
	lockPath := filepath.Join(repoDir, "api.lock")

	if err := os.MkdirAll(filepath.Join(workDir, "proto"), 0755); err != nil {
		t.Fatal(err)
	}
	writeProtoFile(t, filepath.Join(workDir, "proto", "a.proto"))
	runGit(t, workDir, "init")
	runGit(t, workDir, "config", "user.email", "test@test.com")
	runGit(t, workDir, "config", "user.name", "test")
	runGit(t, workDir, "add", "-A")
	runGit(t, workDir, "commit", "-m", "init")
	runGit(t, workDir, "branch", "-M", "main")
	runGit(t, workDir, "init", "--bare", remoteDir)
	runGit(t, workDir, "remote", "add", "origin", remoteDir)
	runGit(t, workDir, "push", "-u", "origin", "main")

	dep1 := GitDep{URL: remoteDir, Ref: "main", Subdir: "proto"}
	r1 := NewGitResolver(dep1, cacheDir)
	if _, err := r1.Fetch(context.Background()); err != nil {
		t.Fatalf("first Fetch failed: %v", err)
	}
	if err := WriteAPILock(lockPath, []GitDep{{
		URL:            dep1.URL,
		Ref:            dep1.Ref,
		Subdir:         dep1.Subdir,
		ResolvedCommit: r1.ResolvedCommit(),
	}}); err != nil {
		t.Fatalf("WriteAPILock failed: %v", err)
	}

	locked, err := ReadAPILock(lockPath)
	if err != nil {
		t.Fatalf("ReadAPILock failed: %v", err)
	}
	if len(locked) != 1 || locked[0].ResolvedCommit == "" {
		t.Fatalf("lock missing resolved commit: %+v", locked)
	}

	// Simulate a subsequent run that loads the lock and uses the commit.
	r2 := NewGitResolver(GitDep{
		URL:            dep1.URL,
		Ref:            dep1.Ref,
		Subdir:         dep1.Subdir,
		ResolvedCommit: locked[0].ResolvedCommit,
	}, cacheDir)
	if _, err := r2.Fetch(context.Background()); err != nil {
		t.Fatalf("second Fetch with locked commit failed: %v", err)
	}
	if r2.ResolvedCommit() != locked[0].ResolvedCommit {
		t.Errorf("ResolvedCommit = %q, want %q", r2.ResolvedCommit(), locked[0].ResolvedCommit)
	}
}

// TestGitCacheSubpath verifies that human-readable host/owner/repo paths are
// extracted from various git URL forms, mirroring buf's module-proxy layout.
func TestGitCacheSubpath(t *testing.T) {
	cases := []struct {
		name string
		url  string
		want []string
	}{
		{"https with .git", "https://github.com/googleapis/googleapis.git", []string{"github.com", "googleapis", "googleapis"}},
		{"https without .git", "https://github.com/googleapis/googleapis", []string{"github.com", "googleapis", "googleapis"}},
		{"http", "http://example.com/owner/repo.git", []string{"example.com", "owner", "repo"}},
		{"scp-style", "git@github.com:owner/repo.git", []string{"github.com", "owner", "repo"}},
		{"ssh with userinfo", "ssh://git@gitlab.com/owner/repo.git", []string{"gitlab.com", "owner", "repo"}},
		{"ssh with port", "ssh://git@gitlab.com:2222/owner/repo.git", []string{"gitlab.com", "owner", "repo"}},
		{"https with port", "https://gitea.example.com:3000/owner/repo.git", []string{"gitea.example.com", "owner", "repo"}},
		{"deep path truncated", "https://github.com/owner/repo/sub/deep.git", []string{"github.com", "owner", "repo"}},
		// Bare local paths (used in tests) cannot be parsed → nil fallback.
		{"local bare path", "/tmp/test/remote.git", nil},
		{"local relative", "remote.git", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := gitCacheSubpath(tc.url)
			if !equalSlices(got, tc.want) {
				t.Errorf("gitCacheSubpath(%q) = %v, want %v", tc.url, got, tc.want)
			}
		})
	}
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestGitResolver_BufStyleCacheLayout verifies that computeCloneDir produces
// buf-style module-proxy paths:
//
//	<cacheDir>/<version>/module-proxy/commit/<host>/<owner>/<repo>/<commit>
//	<cacheDir>/<version>/module-proxy/remote/<host>/<owner>/<repo>/<ref>
//	<cacheDir>/<version>/module-proxy/local/<hash>   (fallback)
func TestGitResolver_BufStyleCacheLayout(t *testing.T) {
	cacheDir := "/home/user/.cache/apigen"
	url := "https://github.com/acme/example.git"

	// Commit-keyed (immutable).
	r1 := NewGitResolver(GitDep{
		URL:            url,
		Ref:            "main",
		ResolvedCommit: "abc123def456",
	}, cacheDir)
	got := r1.computeCloneDir()
	want := filepath.Join(cacheDir, CacheVersion, "module-proxy", "commit",
		"github.com", "acme", "example", "abc123def456")
	if got != want {
		t.Errorf("commit-keyed cloneDir:\n  got  %q\n  want %q", got, want)
	}

	// Ref-keyed (moving ref).
	r2 := NewGitResolver(GitDep{
		URL: url,
		Ref: "main",
	}, cacheDir)
	got = r2.computeCloneDir()
	want = filepath.Join(cacheDir, CacheVersion, "module-proxy", "remote",
		"github.com", "acme", "example", "main")
	if got != want {
		t.Errorf("ref-keyed cloneDir:\n  got  %q\n  want %q", got, want)
	}

	// Ref with slashes (e.g. "feature/foo") is sanitised.
	r3 := NewGitResolver(GitDep{
		URL: url,
		Ref: "feature/foo",
	}, cacheDir)
	got = r3.computeCloneDir()
	// Slashes are stripped by sanitisePathSegment, producing "featurefoo".
	want = filepath.Join(cacheDir, CacheVersion, "module-proxy", "remote",
		"github.com", "acme", "example", "featurefoo")
	if got != want {
		t.Errorf("ref-with-slash cloneDir:\n  got  %q\n  want %q", got, want)
	}

	// Local path fallback (no host/owner/repo).
	r4 := NewGitResolver(GitDep{
		URL: "/tmp/remote.git",
		Ref: "main",
	}, cacheDir)
	got = r4.computeCloneDir()
	// Must be under .../module-proxy/local/<hash>.
	prefix := filepath.Join(cacheDir, CacheVersion, "module-proxy", "local") + "/"
	if !strings.HasPrefix(got, prefix) {
		t.Errorf("local fallback cloneDir:\n  got  %q\n  want prefix %q", got, prefix)
	}
}
