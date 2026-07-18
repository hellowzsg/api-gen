package dep

import (
	"os"
	"os/exec"
	"path/filepath"
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

	importPaths, err := r.Fetch()
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if len(importPaths) == 0 {
		t.Fatal("ImportPaths empty, want at least 1")
	}

	files, err := r.ProtoFiles()
	if err != nil {
		t.Fatalf("ProtoFiles failed: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("len(ProtoFiles) = %d, want 2", len(files))
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
	if _, err := r.Fetch(); err != nil {
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

	_, err := r.Fetch()
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
