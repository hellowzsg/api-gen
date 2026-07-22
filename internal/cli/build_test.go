package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hellowzsg/api-gen/internal/build"
)

// TestBuild_EndToEnd tests the build command (generate + compile).
// Requires protoc-gen-go to be installed; if missing, the test is skipped.
func TestBuild_EndToEnd(t *testing.T) {
	if _, err := build.CheckPluginInstalled("protoc-gen-go"); err != nil {
		t.Skipf("protoc-gen-go not installed, skipping: %v", err)
	}
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
option go_package = "github.com/acme/test/generated/go/book;book";
message BookId { string id = 1; }
message BookMeta { string title = 1; }
`)

	if err := runBuild(context.Background(), filepath.Join(dir, "api.yaml")); err != nil {
		t.Fatalf("runBuild failed: %v", err)
	}
	// Verify the generated proto file exists.
	genProto := filepath.Join(dir, "generated", "proto", "library_service", "library_service.proto")
	if _, err := os.Stat(genProto); err != nil {
		t.Errorf("generated proto not found: %v", err)
	}
	// Verify the generated Go stubs exist (service + user type).
	genGo := filepath.Join(dir, "generated", "go", "library_service", "library_service.pb.go")
	if _, err := os.Stat(genGo); err != nil {
		t.Errorf("generated Go stub not found: %v", err)
	}
	genGrpc := filepath.Join(dir, "generated", "go", "library_service", "library_service_grpc.pb.go")
	if _, err := os.Stat(genGrpc); err != nil {
		t.Errorf("generated gRPC stub not found: %v", err)
	}
	// User type proto also compiled (source-relative path: book.proto → book.pb.go).
	genTypeGo := filepath.Join(dir, "generated", "go", "book.pb.go")
	if _, err := os.Stat(genTypeGo); err != nil {
		t.Errorf("generated user-type Go stub not found: %v", err)
	}
}

// TestBuild_ServiceNarrowing verifies that a service with a narrowed resource
// list only exposes methods for the listed resources.
func TestBuild_ServiceNarrowing(t *testing.T) {
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
      - name: content
        type_: BookContent
        version: { kind: NONE }
        reader: {}
        writer: { update: {} }
services:
  - name: AdminService
    entities:
      - name: book
        resources:
          - name: meta
            reader: { list: true }
`
	writeTestFile(t, filepath.Join(dir, "api.yaml"), apiYAML)
	writeTestFile(t, filepath.Join(dir, "proto", "book.proto"), `
syntax = "proto3";
package test.book;
message BookId { string id = 1; }
message BookMeta { string title = 1; }
message BookContent { string text = 1; }
`)

	if err := runGenerate(context.Background(), filepath.Join(dir, "api.yaml")); err != nil {
		t.Fatalf("runGenerate failed: %v", err)
	}
	genProto := filepath.Join(dir, "generated", "proto", "admin_service", "admin_service.proto")
	data, err := os.ReadFile(genProto)
	if err != nil {
		t.Fatalf("read generated proto: %v", err)
	}
	content := string(data)
	// AdminService narrowed to meta only → no content methods.
	if findSubstring(content, "GetBookContent") {
		t.Error("AdminService should not expose GetBookContent (resource narrowed out)")
	}
	if findSubstring(content, "UpdateBookContent") {
		t.Error("AdminService should not expose UpdateBookContent (resource narrowed out)")
	}
	// meta reader: { list: true } → BatchGet narrowed off, Get + List retained.
	if findSubstring(content, "BatchGetBookMetas") {
		t.Error("AdminService should not expose BatchGetBookMetas (batch narrowed off)")
	}
	if !findSubstring(content, "rpc ListBookMetas") {
		t.Error("AdminService should expose ListBookMetas")
	}
	if !findSubstring(content, "rpc GetBookMeta") {
		t.Error("AdminService should expose GetBookMeta (base reader method retained)")
	}
}
