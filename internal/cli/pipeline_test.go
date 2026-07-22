package cli

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bufbuild/protocompile/linker"

	"github.com/hellowzsg/api-gen/internal/build"
	apigenyaml "github.com/hellowzsg/api-gen/internal/yaml"
)

// TestPrepare_Basic verifies that Prepare returns the shared pipeline
// products (config, import paths, path resolvers, resolved protos, IR).
func TestPrepare_Basic(t *testing.T) {
	dir := t.TempDir()
	apiYAML := `
syntax: v1
name: test.book
import_protos:
  - path: "proto/**/*.proto"
settings:
  go_repo: github.com/acme/test
  out:
    proto: generated/proto
    go: generated/go
entities:
  - name: book
    key: { type_: BookId }
    create: {}
    delete: {}
    resources:
      - name: meta
        type_: BookMeta
        version: { kind: STRONG, type: U64 }
        reader: { batch: true, list: true }
        writer: { update: { mask: true } }
services:
  - name: LibraryService
    entities:
      - name: book
`
	writeTestFile(t, filepath.Join(dir, "api.yaml"), apiYAML)
	writeTestFile(t, filepath.Join(dir, "proto", "book.proto"), `
syntax = "proto3";
package test.book;
message BookId { string id = 1; }
message BookMeta { string title = 1; }
`)

	p, err := Prepare(context.Background(), filepath.Join(dir, "api.yaml"))
	if err != nil {
		t.Fatalf("Prepare failed: %v", err)
	}
	if p.Config == nil {
		t.Error("Pipeline.Config is nil")
	}
	if p.BaseDir != dir {
		t.Errorf("Pipeline.BaseDir = %q, want %q", p.BaseDir, dir)
	}
	if len(p.PathResolvers) != 1 {
		t.Errorf("len(Pipeline.PathResolvers) = %d, want 1", len(p.PathResolvers))
	}
	if p.Resolver == nil {
		t.Fatal("Pipeline.Resolver is nil")
	}
	if len(p.Resolver.Files()) == 0 {
		t.Error("Pipeline.Resolver.Files() is empty (Resolve not called?)")
	}
	if p.IR == nil {
		t.Fatal("Pipeline.IR is nil")
	}
	if len(p.IR.Services) != 1 {
		t.Errorf("len(Pipeline.IR.Services) = %d, want 1", len(p.IR.Services))
	}
	if got := p.IR.TypeImportPaths["test.book.BookId"]; got == "" {
		t.Error("Pipeline.IR.TypeImportPaths missing test.book.BookId")
	}
}

// TestPrepare_MultiPathImports verifies that ALL path imports participate in
// protocompile — previously only the first path import was compiled and types
// defined in later path imports silently failed type validation.
func TestPrepare_MultiPathImports(t *testing.T) {
	dir := t.TempDir()
	apiYAML := `
syntax: v1
name: test.book
import_protos:
  - path: "proto/**/*.proto"
  - path: "proto2/**/*.proto"
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
        reader: {}
      - name: content
        type_: BookContent
        version: { kind: NONE }
        reader: {}
services:
  - name: LibraryService
    entities:
      - name: book
`
	writeTestFile(t, filepath.Join(dir, "api.yaml"), apiYAML)
	writeTestFile(t, filepath.Join(dir, "proto", "book.proto"), `
syntax = "proto3";
package test.book;
message BookId { string id = 1; }
message BookMeta { string title = 1; }
`)
	// BookContent lives in the SECOND path import. Without the multi-path
	// fix this type is never compiled and type validation fails.
	writeTestFile(t, filepath.Join(dir, "proto2", "content.proto"), `
syntax = "proto3";
package test.book;
message BookContent { string text = 1; }
`)

	p, err := Prepare(context.Background(), filepath.Join(dir, "api.yaml"))
	if err != nil {
		t.Fatalf("Prepare failed (second path import not compiled?): %v", err)
	}
	if len(p.PathResolvers) != 2 {
		t.Errorf("len(Pipeline.PathResolvers) = %d, want 2 (all path imports)", len(p.PathResolvers))
	}
	if md := p.Resolver.FindMessageDescriptor("test.book.BookContent"); md == nil {
		t.Error("test.book.BookContent from second path import not found in resolved protos")
	}
}

// TestPrepare_APILockInvalid verifies that a corrupt api.lock is a hard error
// instead of being silently ignored (falling back to moving-ref cache).
func TestPrepare_APILockInvalid(t *testing.T) {
	dir := t.TempDir()
	apiYAML := `
syntax: v1
name: test.book
import_protos:
  - path: "proto/**/*.proto"
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
        reader: {}
services:
  - name: LibraryService
    entities:
      - name: book
`
	writeTestFile(t, filepath.Join(dir, "api.yaml"), apiYAML)
	writeTestFile(t, filepath.Join(dir, "proto", "book.proto"), `
syntax = "proto3";
package test.book;
message BookId { string id = 1; }
message BookMeta { string title = 1; }
`)
	writeTestFile(t, filepath.Join(dir, "api.lock"), "git_deps: [unclosed\n\t: : :\n")

	_, err := Prepare(context.Background(), filepath.Join(dir, "api.yaml"))
	if err == nil {
		t.Fatal("Prepare should fail on invalid api.lock, got nil error")
	}
	if !strings.Contains(err.Error(), "lock") {
		t.Errorf("error should mention api.lock, got: %v", err)
	}
}

// TestPrepare_InvalidProtoInSecondPath verifies that a broken proto in ANY
// path import fails fast with a compile error (previously files beyond the
// first path import were silently skipped).
func TestPrepare_InvalidProtoInSecondPath(t *testing.T) {
	dir := t.TempDir()
	apiYAML := `
syntax: v1
name: test.book
import_protos:
  - path: "proto/**/*.proto"
  - path: "proto2/**/*.proto"
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
        reader: {}
services:
  - name: LibraryService
    entities:
      - name: book
`
	writeTestFile(t, filepath.Join(dir, "api.yaml"), apiYAML)
	writeTestFile(t, filepath.Join(dir, "proto", "book.proto"), `
syntax = "proto3";
package test.book;
message BookId { string id = 1; }
message BookMeta { string title = 1; }
`)
	writeTestFile(t, filepath.Join(dir, "proto2", "broken.proto"), `
syntax = "proto3";
package test.book;
message Broken {{{ not valid proto
`)

	_, err := Prepare(context.Background(), filepath.Join(dir, "api.yaml"))
	if err == nil {
		t.Fatal("Prepare should fail on invalid proto in second path import")
	}
	if !strings.Contains(err.Error(), "resolve proto") {
		t.Errorf("error should be a proto compile failure, got: %v", err)
	}
}

// TestResolveDependencies_DeclarationOrder verifies that resolveDependencies
// aggregates import paths in api.yaml declaration order across all entries.
func TestResolveDependencies_DeclarationOrder(t *testing.T) {
	dir := t.TempDir()
	cfg, err := apigenyaml.Parse(strings.NewReader(`
syntax: v1
name: test.book
import_protos:
  - path: "proto2/**/*.proto"
  - path: "proto/**/*.proto"
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
        reader: {}
services:
  - name: LibraryService
    entities:
      - name: book
`))
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	writeTestFile(t, filepath.Join(dir, "proto", "a.proto"), "syntax = \"proto3\";\npackage test.book;\n")
	writeTestFile(t, filepath.Join(dir, "proto2", "b.proto"), "syntax = \"proto3\";\npackage test.book;\n")

	importPaths, pathResolvers, err := resolveDependencies(context.Background(), cfg, dir, t.TempDir())
	if err != nil {
		t.Fatalf("resolveDependencies failed: %v", err)
	}
	if len(pathResolvers) != 2 {
		t.Fatalf("len(pathResolvers) = %d, want 2", len(pathResolvers))
	}
	// Declaration order is proto2 first, proto second.
	idx2, idx1 := -1, -1
	for i, p := range importPaths {
		switch filepath.Clean(p) {
		case filepath.Join(dir, "proto2"):
			idx2 = i
		case filepath.Join(dir, "proto"):
			idx1 = i
		}
	}
	if idx2 < 0 || idx1 < 0 {
		t.Fatalf("importPaths missing proto roots: %v", importPaths)
	}
	if idx2 > idx1 {
		t.Errorf("importPaths not in declaration order: proto2 (idx %d) should precede proto (idx %d): %v", idx2, idx1, importPaths)
	}
}

// TestRunBuild_SinglePrepare verifies that runBuild executes the shared
// Prepare pipeline exactly once (no duplicate parse/resolve) and that the
// compiler receives the generated service protos plus the local protos from
// ALL path imports.
func TestRunBuild_SinglePrepare(t *testing.T) {
	dir := t.TempDir()
	apiYAML := `
syntax: v1
name: test.book
import_protos:
  - path: "proto/**/*.proto"
  - path: "proto2/**/*.proto"
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
        reader: {}
      - name: content
        type_: BookContent
        version: { kind: NONE }
        reader: {}
services:
  - name: LibraryService
    entities:
      - name: book
`
	writeTestFile(t, filepath.Join(dir, "api.yaml"), apiYAML)
	writeTestFile(t, filepath.Join(dir, "proto", "book.proto"), `
syntax = "proto3";
package test.book;
message BookId { string id = 1; }
message BookMeta { string title = 1; }
`)
	writeTestFile(t, filepath.Join(dir, "proto2", "content.proto"), `
syntax = "proto3";
package test.book;
message BookContent { string text = 1; }
`)

	prepareCalls := 0
	origPrepare := prepareFn
	prepareFn = func(ctx context.Context, path string) (*Pipeline, error) {
		prepareCalls++
		return origPrepare(ctx, path)
	}
	defer func() { prepareFn = origPrepare }()

	var gotFiles linker.Files
	var gotFileToGenerate []string
	origCompile := compileFn
	compileFn = func(ctx context.Context, files linker.Files, fileToGenerate []string, specs []build.PluginSpec) error {
		gotFiles = files
		gotFileToGenerate = fileToGenerate
		return nil
	}
	defer func() { compileFn = origCompile }()

	if err := runBuild(context.Background(), filepath.Join(dir, "api.yaml")); err != nil {
		t.Fatalf("runBuild failed: %v", err)
	}
	if prepareCalls != 1 {
		t.Errorf("Prepare called %d times, want exactly 1", prepareCalls)
	}
	// fileToGenerate: generated service proto + both path imports' files.
	for _, want := range []string{filepath.Join("library_service", "library_service.proto"), "book.proto", "content.proto"} {
		found := false
		for _, f := range gotFileToGenerate {
			if f == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("fileToGenerate missing %q (got %v)", want, gotFileToGenerate)
		}
	}
	// Compiled files must include the generated service proto.
	foundGen := false
	for _, f := range gotFiles {
		if f.Path() == "library_service/library_service.proto" {
			foundGen = true
		}
	}
	if !foundGen {
		t.Error("compiled files missing generated service proto library_service/library_service.proto")
	}
}
