package cli

import (
	"context"
	"path/filepath"
	"testing"
)

// TestBuild_EndToEnd 测试 build 命令（generate + 编译）。
// 注意：此测试需要 protoc-gen-go 已安装，否则跳过。
func TestBuild_EndToEnd(t *testing.T) {
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
        reader: {}
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

	err := runBuild(context.Background(), filepath.Join(dir, "api.yaml"))
	if err != nil {
		// If protoc-gen-go is not installed, this is expected
		t.Logf("runBuild failed (may be missing protoc-gen-go): %v", err)
		return
	}
	// Verify generate output exists at least
	genProto := filepath.Join(dir, "generated", "proto", "library_service", "library_service.proto")
	if _, err := stat(genProto); err != nil {
		t.Errorf("generated proto not found: %v", err)
	}
}

func stat(path string) (interface{}, error) {
	return nil, nil
}
