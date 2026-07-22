package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hellowzsg/api-gen/internal/dep"
)

// initTestGitRepo creates a local git repo (bare remote + push) containing
// the given files, and returns the remote path usable as a git URL.
func initTestGitRepo(t *testing.T, baseDir, name string, files map[string]string) string {
	t.Helper()
	workDir := filepath.Join(baseDir, name+"-work")
	remoteDir := filepath.Join(baseDir, name+".git")
	for rel, content := range files {
		writeTestFile(t, filepath.Join(workDir, rel), content)
	}
	runGitCmd(t, workDir, "init")
	runGitCmd(t, workDir, "config", "user.email", "test@test.com")
	runGitCmd(t, workDir, "config", "user.name", "test")
	runGitCmd(t, workDir, "add", "-A")
	runGitCmd(t, workDir, "commit", "-m", "init")
	runGitCmd(t, workDir, "branch", "-M", "main")
	runGitCmd(t, workDir, "init", "--bare", remoteDir)
	runGitCmd(t, workDir, "remote", "add", "origin", remoteDir)
	runGitCmd(t, workDir, "push", "-u", "origin", "main")
	return remoteDir
}

func runGitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test.com")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
}

// setupPruneFixture builds a project with one path import (main.proto) and
// two git deps: "types" (imported by main.proto) and "unused" (not imported
// unless importsUnused is true). It returns the api.yaml path and both repo
// URLs.
func setupPruneFixture(t *testing.T, importsUnused bool) (apiYAMLPath, typesRepo, unusedRepo string) {
	t.Helper()
	t.Setenv("APIGEN_CACHE_DIR", t.TempDir())
	dir := t.TempDir()
	typesRepo = initTestGitRepo(t, dir, "types", map[string]string{
		"proto/types/types.proto": `syntax = "proto3";
package test.book;
message BookId { string id = 1; }
message BookMeta { string title = 1; }
`,
	})
	unusedRepo = initTestGitRepo(t, dir, "unused", map[string]string{
		"proto/unused/unused.proto": `syntax = "proto3";
package test.unused;
message Unused { string id = 1; }
`,
	})
	mainProto := `syntax = "proto3";
package test.book;
import "types/types.proto";
`
	extraResource := ""
	if importsUnused {
		mainProto += `import "unused/unused.proto";
`
		// Entity references a type from the second repo → both deps used.
		extraResource = `
      - name: tag
        type_: test.unused.Unused
        version: { kind: NONE }
        reader: {}`
	}
	writeTestFile(t, filepath.Join(dir, "proto", "main.proto"), mainProto)
	apiYAML := `
syntax: v1
name: test.book
import_protos:
  - path: "proto/**/*.proto"
  - git: ` + typesRepo + `
    ref: main
    subdir: proto
  - git: ` + unusedRepo + `
    ref: main
    subdir: proto
settings:
  go_repo: github.com/acme/test
  out:
    proto: generated/proto
    go: generated/go
entities:
  - name: book
    key: { type_: BookId }
    create: {}
    resources:
      - name: meta
        type_: BookMeta
        version: { kind: NONE }
        reader: {}` + extraResource + `
services:
  - name: LibraryService
    entities:
      - name: book
`
	apiYAMLPath = filepath.Join(dir, "api.yaml")
	writeTestFile(t, apiYAMLPath, apiYAML)
	return apiYAMLPath, typesRepo, unusedRepo
}

func readLockURLs(t *testing.T, apiYAMLPath string) []string {
	t.Helper()
	deps, err := dep.ReadAPILock(filepath.Join(filepath.Dir(apiYAMLPath), "api.lock"))
	if err != nil {
		t.Fatalf("ReadAPILock: %v", err)
	}
	var urls []string
	for _, d := range deps {
		urls = append(urls, d.URL)
	}
	return urls
}

// TestDepPrune_RemovesUnreferenced verifies that a git dependency whose
// protos are not referenced by any entity type is pruned from api.lock.
func TestDepPrune_RemovesUnreferenced(t *testing.T) {
	apiYAMLPath, typesRepo, _ := setupPruneFixture(t, false)
	ctx := context.Background()
	if err := runDepUpdate(ctx, apiYAMLPath); err != nil {
		t.Fatalf("runDepUpdate failed: %v", err)
	}
	if got := readLockURLs(t, apiYAMLPath); len(got) != 2 {
		t.Fatalf("lock should contain 2 deps before prune, got %v", got)
	}
	if err := runDepPrune(ctx, apiYAMLPath); err != nil {
		t.Fatalf("runDepPrune failed: %v", err)
	}
	got := readLockURLs(t, apiYAMLPath)
	if len(got) != 1 || got[0] != typesRepo {
		t.Errorf("after prune lock = %v, want only %q", got, typesRepo)
	}
}

// TestDepPrune_KeepsUndecidable verifies that a locked dependency whose clone
// is absent from the cache (never fetched, no longer declared) is
// conservatively kept.
func TestDepPrune_KeepsUndecidable(t *testing.T) {
	apiYAMLPath, typesRepo, unusedRepo := setupPruneFixture(t, false)
	ctx := context.Background()
	if err := runDepUpdate(ctx, apiYAMLPath); err != nil {
		t.Fatalf("runDepUpdate failed: %v", err)
	}
	// Rewrite the lock: keep the types dep, and add the unused dep with a
	// commit that was never cloned (cache lookup will miss → undecidable).
	lockPath := filepath.Join(filepath.Dir(apiYAMLPath), "api.lock")
	deps, err := dep.ReadAPILock(lockPath)
	if err != nil {
		t.Fatalf("ReadAPILock: %v", err)
	}
	var kept []dep.GitDep
	for _, d := range deps {
		if d.URL == typesRepo {
			kept = append(kept, d)
		}
	}
	kept = append(kept, dep.GitDep{
		URL:            unusedRepo,
		Ref:            "main",
		Subdir:         "proto",
		ResolvedCommit: strings.Repeat("0", 40),
	})
	if err := dep.WriteAPILock(lockPath, kept); err != nil {
		t.Fatalf("WriteAPILock: %v", err)
	}
	// Point api.yaml at ONLY the types repo so Prepare never clones the
	// unused one, leaving its clone dir absent from the cache.
	data, err := os.ReadFile(apiYAMLPath)
	if err != nil {
		t.Fatal(err)
	}
	content := strings.Replace(string(data), "  - git: "+unusedRepo+"\n    ref: main\n    subdir: proto\n", "", 1)
	if err := os.WriteFile(apiYAMLPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := runDepPrune(ctx, apiYAMLPath); err != nil {
		t.Fatalf("runDepPrune failed: %v", err)
	}
	if got := readLockURLs(t, apiYAMLPath); len(got) != 2 {
		t.Errorf("undecidable dep should be kept, lock = %v", got)
	}
}

// TestDepPrune_AllReferenced verifies the lock is unchanged when every
// dependency is referenced.
func TestDepPrune_AllReferenced(t *testing.T) {
	apiYAMLPath, _, _ := setupPruneFixture(t, true)
	ctx := context.Background()
	if err := runDepUpdate(ctx, apiYAMLPath); err != nil {
		t.Fatalf("runDepUpdate failed: %v", err)
	}
	if err := runDepPrune(ctx, apiYAMLPath); err != nil {
		t.Fatalf("runDepPrune failed: %v", err)
	}
	if got := readLockURLs(t, apiYAMLPath); len(got) != 2 {
		t.Errorf("all deps referenced, lock should be unchanged, got %v", got)
	}
}
