package dep

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// All three dependency kinds must satisfy the unified Resolver interface.
var (
	_ Resolver = (*PathResolver)(nil)
	_ Resolver = (*GitResolver)(nil)
	_ Resolver = (*BSRResolver)(nil)
)

// TestResolver_PathFetch verifies that a path resolver's Fetch globs files
// and returns its import paths, and that ProtoFiles reports the globbed
// files (path protos are explicitly named for compilation).
func TestResolver_PathFetch(t *testing.T) {
	dir := t.TempDir()
	protoDir := filepath.Join(dir, "proto")
	if err := os.MkdirAll(protoDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(protoDir, "a.proto"), []byte("syntax = \"proto3\";\n"), 0644); err != nil {
		t.Fatal(err)
	}
	r := NewPathResolverWithBase(filepath.Join("proto", "**", "*.proto"), dir)
	paths, err := r.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("Fetch returned no import paths")
	}
	if got := r.ProtoFiles(); len(got) != 1 {
		t.Fatalf("len(ProtoFiles) = %d, want 1", len(got))
	}
}

// TestResolver_LazyProtoFiles verifies that git and bsr resolvers report nil
// ProtoFiles: their protos compile lazily as transitive imports — explicitly
// naming them would compile entire dependency repos.
func TestResolver_LazyProtoFiles(t *testing.T) {
	g := NewGitResolver(GitDep{URL: "https://example.com/owner/repo.git"}, t.TempDir())
	if files := g.ProtoFiles(); files != nil {
		t.Errorf("git ProtoFiles = %v, want nil (lazy compile)", files)
	}
	b := NewBSRResolver([]BSRDep{{Module: "buf.build/owner/repo"}}, t.TempDir())
	if files := b.ProtoFiles(); files != nil {
		t.Errorf("bsr ProtoFiles = %v, want nil (lazy compile)", files)
	}
}
