package dep

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeBuf installs a fake `buf` executable that logs every invocation to
// logPath, emulates `buf dep update` by copying lockFile to ./buf.lock, and
// emulates `buf export` by creating the --output directory with a dummy
// proto file.
func fakeBuf(t *testing.T, lockYAML string) (bufPath, logPath string) {
	t.Helper()
	dir := t.TempDir()
	logPath = filepath.Join(dir, "buf.log")
	lockFile := filepath.Join(dir, "buf.lock")
	if err := os.WriteFile(lockFile, []byte(lockYAML), 0644); err != nil {
		t.Fatal(err)
	}
	script := `#!/bin/sh
echo "$@" >> "` + logPath + `"
case "$1" in
  dep)
    cp "` + lockFile + `" buf.lock
    ;;
  export)
    out=""
    prev=""
    for a in "$@"; do
      if [ "$prev" = "--output" ]; then out="$a"; fi
      prev="$a"
    done
    mkdir -p "$out"
    echo 'syntax = "proto3";' > "$out/dummy.proto"
    ;;
esac
`
	bufPath = filepath.Join(dir, "buf")
	if err := os.WriteFile(bufPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	return bufPath, logPath
}

const testBufLock = `version: v2
deps:
  - name: buf.build/acme/one
    commit: c1c1c1
    digest: b5:digest-one
  - name: buf.build/acme/two
    commit: c2c2c2
    digest: b5:digest-two
`

func readBufLog(t *testing.T, logPath string) []string {
	t.Helper()
	data, err := os.ReadFile(logPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, l := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

func countLogPrefix(lines []string, prefix string) int {
	n := 0
	for _, l := range lines {
		if strings.HasPrefix(l, prefix) {
			n++
		}
	}
	return n
}

// TestBSRResolver_BatchSingleDepUpdate verifies that multiple BSR deps are
// fetched with a single `buf dep update` and per-module exports.
func TestBSRResolver_BatchSingleDepUpdate(t *testing.T) {
	bufPath, logPath := fakeBuf(t, testBufLock)
	cacheDir := t.TempDir()
	r := NewBSRResolverWithBufCmd([]BSRDep{
		{Module: "buf.build/acme/one"},
		{Module: "buf.build/acme/two"},
	}, cacheDir, bufPath)

	paths, err := r.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("len(paths) = %d, want 2", len(paths))
	}
	lines := readBufLog(t, logPath)
	if n := countLogPrefix(lines, "dep update"); n != 1 {
		t.Errorf("buf dep update called %d times, want exactly 1 (batched)", n)
	}
	if n := countLogPrefix(lines, "export buf.build/acme/one "); n != 1 {
		t.Errorf("export one called %d times, want 1", n)
	}
	if n := countLogPrefix(lines, "export buf.build/acme/two "); n != 1 {
		t.Errorf("export two called %d times, want 1", n)
	}
	// Export dirs are content-addressed by buf.lock digest.
	for _, p := range paths {
		if !strings.Contains(p, "digest-") {
			t.Errorf("export dir %q should contain the buf.lock digest", p)
		}
		if !strings.HasPrefix(p, cacheDir) {
			t.Errorf("export dir %q should be under cacheDir %q", p, cacheDir)
		}
	}
}

// TestBSRResolver_ExportCacheHit verifies that a second Fetch with identical
// deps skips buf export entirely (buf dep update still runs to resolve the
// current digest).
func TestBSRResolver_ExportCacheHit(t *testing.T) {
	bufPath, logPath := fakeBuf(t, testBufLock)
	cacheDir := t.TempDir()
	deps := []BSRDep{{Module: "buf.build/acme/one"}, {Module: "buf.build/acme/two"}}

	r1 := NewBSRResolverWithBufCmd(deps, cacheDir, bufPath)
	paths1, err := r1.Fetch(context.Background())
	if err != nil {
		t.Fatalf("first Fetch failed: %v", err)
	}
	r2 := NewBSRResolverWithBufCmd(deps, cacheDir, bufPath)
	paths2, err := r2.Fetch(context.Background())
	if err != nil {
		t.Fatalf("second Fetch failed: %v", err)
	}
	if strings.Join(paths1, ",") != strings.Join(paths2, ",") {
		t.Errorf("cache-hit paths differ: %v vs %v", paths1, paths2)
	}
	lines := readBufLog(t, logPath)
	if n := countLogPrefix(lines, "dep update"); n != 2 {
		t.Errorf("buf dep update called %d times, want 2 (once per Fetch)", n)
	}
	if n := countLogPrefix(lines, "export "); n != 2 {
		t.Errorf("buf export called %d times total, want 2 (second Fetch fully cached)", n)
	}
}

// TestBSRResolver_NoWorkspacePollution verifies that buf.yaml/buf.lock live
// in a throwaway temp dir that is removed after Fetch — nothing is written
// to the user workspace or left in the OS temp dir.
func TestBSRResolver_NoWorkspacePollution(t *testing.T) {
	bufPath, _ := fakeBuf(t, testBufLock)
	cacheDir := t.TempDir()
	r := NewBSRResolverWithBufCmd([]BSRDep{{Module: "buf.build/acme/one"}}, cacheDir, bufPath)
	if _, err := r.Fetch(context.Background()); err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	leftovers, _ := filepath.Glob(filepath.Join(os.TempDir(), "apigen-buf-*"))
	if len(leftovers) > 0 {
		t.Errorf("temp buf workdirs left behind: %v", leftovers)
	}
	for _, name := range []string{"buf.yaml", "buf.lock"} {
		matches, _ := filepath.Glob(filepath.Join(cacheDir, "**", name))
		if len(matches) > 0 {
			t.Errorf("%s leaked into cache dir: %v", name, matches)
		}
	}
}

// TestBSRResolver_VersionRef verifies that the version field is honoured as
// a buf export ref (module:version).
func TestBSRResolver_VersionRef(t *testing.T) {
	bufPath, logPath := fakeBuf(t, testBufLock)
	cacheDir := t.TempDir()
	r := NewBSRResolverWithBufCmd([]BSRDep{{Module: "buf.build/acme/one", Version: "v1.2.3"}}, cacheDir, bufPath)
	if _, err := r.Fetch(context.Background()); err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if n := countLogPrefix(readBufLog(t, logPath), "export buf.build/acme/one:v1.2.3 "); n != 1 {
		t.Errorf("export should be called with module:version ref, log: %v", readBufLog(t, logPath))
	}
}

// TestBSRResolver_ImportPathsOnlyDeclared verifies that ImportPaths returns
// only the export dirs of this resolver's declared modules — unrelated
// modules already in the cache must not leak in.
func TestBSRResolver_ImportPathsOnlyDeclared(t *testing.T) {
	bufPath, _ := fakeBuf(t, testBufLock)
	cacheDir := t.TempDir()
	// Pre-seed an unrelated module in the bsr cache.
	unrelated := filepath.Join(cacheDir, CacheVersion, moduleProxyDir, "bsr", "buf.build_unrelated_mod", "b5_xyz")
	if err := os.MkdirAll(unrelated, 0755); err != nil {
		t.Fatal(err)
	}
	r := NewBSRResolverWithBufCmd([]BSRDep{{Module: "buf.build/acme/one"}}, cacheDir, bufPath)
	if _, err := r.Fetch(context.Background()); err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	for _, p := range r.ImportPaths() {
		if strings.Contains(p, "unrelated") {
			t.Errorf("ImportPaths leaked unrelated cached module: %v", r.ImportPaths())
		}
	}
	if len(r.ImportPaths()) != 1 {
		t.Errorf("len(ImportPaths) = %d, want 1", len(r.ImportPaths()))
	}
}

// TestBSRResolver_BufYAMLContent verifies the generated buf.yaml (v2)
// content including version-qualified refs.
func TestBSRResolver_BufYAMLContent(t *testing.T) {
	content := bufYAMLContent([]BSRDep{
		{Module: "buf.build/googleapis/googleapis"},
		{Module: "buf.build/acme/one", Version: "v1.2.3"},
	})
	for _, want := range []string{"version: v2", "buf.build/googleapis/googleapis", "buf.build/acme/one:v1.2.3"} {
		if !contains(content, want) {
			t.Errorf("buf.yaml content missing %q:\n%s", want, content)
		}
	}
}

// TestBSRResolver_BufNotInstalled 测试 buf 未安装检测。
func TestBSRResolver_BufNotInstalled(t *testing.T) {
	r := NewBSRResolverWithBufCmd([]BSRDep{{Module: "buf.build/googleapis/googleapis"}}, t.TempDir(), "/nonexistent/buf")
	_, err := r.Fetch(context.Background())
	if err == nil {
		t.Fatal("Fetch should fail when buf not installed")
	}
}

// TestBSRResolver_ValidateModule 测试 BSR module 名校验（白名单）。
func TestBSRResolver_ValidateModule(t *testing.T) {
	tests := []struct {
		module string
		valid  bool
	}{
		{"buf.build/googleapis/googleapis", true},
		{"buf.build/acme/options", true},
		{"buf.build/googleapis/googleapis;rm -rf /", false},
		{"buf.build/foo/bar extra", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.module, func(t *testing.T) {
			err := validateBSRModule(tt.module)
			if tt.valid && err != nil {
				t.Errorf("validateBSRModule(%q) = %v, want nil", tt.module, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("validateBSRModule(%q) = nil, want error", tt.module)
			}
		})
	}
}

// TestBSRResolver_NoDeps verifies that an empty dep list is a no-op.
func TestBSRResolver_NoDeps(t *testing.T) {
	bufPath, logPath := fakeBuf(t, testBufLock)
	r := NewBSRResolverWithBufCmd(nil, t.TempDir(), bufPath)
	paths, err := r.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch with no deps should not error, got: %v", err)
	}
	if paths != nil {
		t.Errorf("paths = %v, want nil", paths)
	}
	if lines := readBufLog(t, logPath); len(lines) > 0 {
		t.Errorf("no buf invocations expected with empty deps, got: %v", lines)
	}
}

// TestBSRResolver_InvalidVersion verifies version label validation.
func TestBSRResolver_InvalidVersion(t *testing.T) {
	r := NewBSRResolverWithBufCmd([]BSRDep{{Module: "buf.build/acme/one", Version: "v1\nrm -rf /"}}, t.TempDir(), "buf")
	if _, err := r.Fetch(context.Background()); err == nil {
		t.Fatal("Fetch should fail on invalid version label")
	}
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
