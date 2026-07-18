package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerate_EndToEnd(t *testing.T) {
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
    js: generated/js
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

	err := runGenerate(context.Background(), filepath.Join(dir, "api.yaml"))
	if err != nil {
		t.Fatalf("runGenerate failed: %v", err)
	}

	genProto := filepath.Join(dir, "generated", "proto", "library_service", "library_service.proto")
	if _, err := os.Stat(genProto); os.IsNotExist(err) {
		t.Fatalf("generated proto not found at %s", genProto)
	}

	data, err := os.ReadFile(genProto)
	if err != nil {
		t.Fatalf("read generated proto: %v", err)
	}
	content := string(data)
	if !findSubstring(content, "syntax = \"proto3\"") {
		t.Error("missing syntax declaration")
	}
	if !findSubstring(content, "package test.book.library_service") {
		t.Error("missing package")
	}
	if !findSubstring(content, "service LibraryService") {
		t.Error("missing service")
	}
	if !findSubstring(content, "rpc CreateBook") {
		t.Error("missing CreateBook")
	}
	if !findSubstring(content, "rpc GetBookMeta") {
		t.Error("missing GetBookMeta")
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir %q: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
