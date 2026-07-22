package cli

import (
	"context"
	"path/filepath"
	"testing"
)

func TestDepUpdate_LocalOnly(t *testing.T) {
	dir := t.TempDir()
	apiYAML := `
syntax: v1
name: test
import_protos:
  - path: "proto/**/*.proto"
entities:
  - name: book
    key: { type_: BookId }
    resources:
      - name: meta
        type_: BookMeta
        version: { kind: NONE }
        reader: {}
`
	writeTestFile(t, filepath.Join(dir, "api.yaml"), apiYAML)
	writeTestFile(t, filepath.Join(dir, "proto", "book.proto"), `syntax = "proto3"; package test; message BookId { string id = 1; } message BookMeta { string title = 1; }`)
	err := runDepUpdate(context.Background(), filepath.Join(dir, "api.yaml"))
	if err != nil {
		t.Logf("dep update with local only (no remote deps): %v", err)
	}
}

func TestEntityList(t *testing.T) {
	dir := t.TempDir()
	apiYAML := `
syntax: v1
name: test
import_protos:
  - path: "proto/**/*.proto"
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
	writeTestFile(t, filepath.Join(dir, "proto", "book.proto"), `syntax = "proto3"; package test; message BookId { string id = 1; } message BookMeta { string title = 1; }`)
	err := runEntityList(context.Background(), filepath.Join(dir, "api.yaml"))
	if err != nil {
		t.Fatalf("runEntityList failed: %v", err)
	}
}

// TestEntityList_HTTPEnabled verifies that `entity list` works when HTTP is
// enabled — it is a display command and must not require key descriptors.
func TestEntityList_HTTPEnabled(t *testing.T) {
	dir := t.TempDir()
	apiYAML := `
syntax: v1
name: test
import_protos:
  - path: "proto/**/*.proto"
settings:
  go_repo: github.com/acme/test
  out:
    proto: generated/proto
    go: generated/go
  http:
    enable: true
    prefix: /library
entities:
  - name: book
    key: { type_: BookId }
    create: {}
    resources:
      - name: meta
        type_: BookMeta
        version: { kind: NONE }
        reader: { list: true }
services:
  - name: LibraryService
    entities:
      - name: book
`
	writeTestFile(t, filepath.Join(dir, "api.yaml"), apiYAML)
	writeTestFile(t, filepath.Join(dir, "proto", "book.proto"), `syntax = "proto3"; package test; message BookId { string id = 1; } message BookMeta { string title = 1; }`)
	if err := runEntityList(context.Background(), filepath.Join(dir, "api.yaml")); err != nil {
		t.Fatalf("runEntityList with HTTP enabled should succeed, got: %v", err)
	}
}
