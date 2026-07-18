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
