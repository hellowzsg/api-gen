package dep

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPathResolver_GlobMatch 测试 glob 匹配 proto 文件。
func TestPathResolver_GlobMatch(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "proto", "a.proto"), `syntax = "proto3"; package test; message A {}`)
	writeFile(t, filepath.Join(dir, "proto", "sub", "b.proto"), `syntax = "proto3"; package test; message B {}`)
	writeFile(t, filepath.Join(dir, "other.txt"), `not a proto`)

	r := NewPathResolver(filepath.Join(dir, "proto/**/*.proto"))
	files, err := r.ResolveFiles()
	if err != nil {
		t.Fatalf("ResolveFiles failed: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("len(files) = %d, want 2", len(files))
	}
	for _, f := range files {
		if filepath.Ext(f) != ".proto" {
			t.Errorf("file %q is not .proto", f)
		}
	}
}

// TestPathResolver_ImportPaths 测试 ImportPaths 返回包含 proto 文件的目录。
func TestPathResolver_ImportPaths(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "proto", "a.proto"), `syntax = "proto3"; package test; message A {}`)

	r := NewPathResolver(filepath.Join(dir, "proto/**/*.proto"))
	if err := r.Glob(); err != nil {
		t.Fatalf("Glob failed: %v", err)
	}
	paths := r.ImportPaths()
	if len(paths) == 0 {
		t.Fatal("ImportPaths returned empty, want at least 1")
	}
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			t.Errorf("import path %q stat failed: %v", p, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("import path %q is not a directory", p)
		}
	}
}

// TestPathResolver_NoMatch 测试 proto 文件不存在 fail-fast。
func TestPathResolver_NoMatch(t *testing.T) {
	dir := t.TempDir()
	r := NewPathResolver(filepath.Join(dir, "nonexistent/**/*.proto"))
	_, err := r.ResolveFiles()
	if err == nil {
		t.Fatal("ResolveFiles should fail when no proto files match")
	}
}

// TestPathResolver_RelativeGlob 测试相对路径 glob（基准 = baseDir）。
func TestPathResolver_RelativeGlob(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "proto", "a.proto"), `syntax = "proto3"; package test; message A {}`)

	r := NewPathResolverWithBase("proto/**/*.proto", dir)
	files, err := r.ResolveFiles()
	if err != nil {
		t.Fatalf("ResolveFiles failed: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("len(files) = %d, want 1", len(files))
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir %q: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}
