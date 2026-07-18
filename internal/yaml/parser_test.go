package yaml

import (
	"strings"
	"testing"
)

// TestParseValidYAML 测试合法 api.yaml 解析为内部结构。
func TestParseValidYAML(t *testing.T) {
	input := `
syntax: v1
name: demo.business.book
import_protos:
  - path: "proto/**/*.proto"
settings:
  go_repo: github.com/acme/demo-book
  js_repo: "@acme/demo-book"
  out:
    proto: generated/proto
    go: generated/go
    js: generated/js
entities:
  - name: book
    key:
      type_: BookId
    create: {}
    delete: {}
    resources:
      - name: meta
        type_: BookMeta
        version: { kind: STRONG, type: U64 }
        reader:
          batch: true
          list: true
        writer:
          update: { mask: true }
services:
  - name: LibraryService
    entities:
      - name: book
`
	cfg, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if cfg.Syntax != "v1" {
		t.Errorf("Syntax = %q, want %q", cfg.Syntax, "v1")
	}
	if cfg.Name != "demo.business.book" {
		t.Errorf("Name = %q, want %q", cfg.Name, "demo.business.book")
	}
	if len(cfg.Entities) != 1 {
		t.Fatalf("len(Entities) = %d, want 1", len(cfg.Entities))
	}
	e := cfg.Entities[0]
	if e.Name != "book" {
		t.Errorf("Entity.Name = %q, want %q", e.Name, "book")
	}
	if e.Key.Type != "BookId" {
		t.Errorf("Key.Type = %q, want %q", e.Key.Type, "BookId")
	}
	if e.Create == nil {
		t.Error("Create is nil, want non-nil")
	}
	if e.Delete == nil {
		t.Error("Delete is nil, want non-nil")
	}
	if len(e.Resources) != 1 {
		t.Fatalf("len(Resources) = %d, want 1", len(e.Resources))
	}
	r := e.Resources[0]
	if r.Name != "meta" {
		t.Errorf("Resource.Name = %q, want %q", r.Name, "meta")
	}
	if r.Type != "BookMeta" {
		t.Errorf("Resource.Type = %q, want %q", r.Type, "BookMeta")
	}
	if r.Version.Kind != "STRONG" {
		t.Errorf("Version.Kind = %q, want %q", r.Version.Kind, "STRONG")
	}
	if r.Version.Type != "U64" {
		t.Errorf("Version.Type = %q, want %q", r.Version.Type, "U64")
	}
	if r.Reader == nil || !r.Reader.Batch || !r.Reader.List {
		t.Errorf("Reader = %+v, want Batch=true List=true", r.Reader)
	}
	if r.Writer == nil || r.Writer.Update == nil || !r.Writer.Update.Mask {
		t.Errorf("Writer.Update.Mask = false, want true")
	}
	if len(cfg.Services) != 1 {
		t.Fatalf("len(Services) = %d, want 1", len(cfg.Services))
	}
	if cfg.Services[0].Name != "LibraryService" {
		t.Errorf("Service.Name = %q, want %q", cfg.Services[0].Name, "LibraryService")
	}
}

// TestParseMissingRequiredSections 测试缺少必填段 fail-fast。
func TestParseMissingRequiredSections(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"missing entities", `
syntax: v1
name: foo
`},
		{"missing name", `
syntax: v1
entities: []
`},
		{"missing syntax", `
name: foo
entities: []
`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(tt.input))
			if err == nil {
				t.Errorf("Parse should fail for %s, got nil", tt.name)
			}
		})
	}
}

// TestParseInvalidName 测试 name 字段非法 fail-fast。
func TestParseInvalidName(t *testing.T) {
	tests := []struct {
		name string
		val  string
	}{
		{"empty", ""},
		{"with spaces", "foo bar"},
		{"with special chars", "foo@bar"},
		{"starts with digit", "1foo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := "syntax: v1\nname: " + tt.val + "\nentities: []\n"
			_, err := Parse(strings.NewReader(input))
			if err == nil {
				t.Errorf("Parse should fail for name=%q, got nil", tt.val)
			}
		})
	}
}

// TestParseDeleteSoft 测试 delete_soft 解析。
func TestParseDeleteSoft(t *testing.T) {
	input := `
syntax: v1
name: foo
entities:
  - name: book
    key: { type_: BookId }
    delete: {}
    delete_soft: {}
    resources:
      - name: meta
        type_: BookMeta
        version: { kind: NONE }
        reader: {}
        writer: { update: {} }
`
	cfg, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if cfg.Entities[0].DeleteSoft == nil {
		t.Error("DeleteSoft is nil, want non-nil")
	}
}

// TestParseHTTPOverride 测试 reader.http / writer.update.http /
// custom_methods[].http 的解析。
func TestParseHTTPOverride(t *testing.T) {
	input := `
syntax: v1
name: demo.business.book
import_protos:
  - path: "proto/**/*.proto"
settings:
  go_repo: github.com/acme/demo-book
  out:
    proto: generated/proto
    go: generated/go
entities:
  - name: book
    key:
      type_: BookId
    resources:
      - name: meta
        type_: BookMeta
        version: { kind: STRONG, type: U64 }
        reader:
          list: true
          http:
            verb: get
            path: /library/LibraryService/book/{key.id}/metadata
        writer:
          update:
            mask: true
            http:
              verb: put
              body_style: resource
services:
  - name: LibraryService
    entities:
      - name: book
    custom_methods:
      - name: ArchiveBook
        request: ArchiveBookRequest
        response: ArchiveBookResponse
        http:
          verb: post
          path: /library/LibraryService/book/{book_id}:archive
          body: "*"
`
	cfg, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// 验证 reader.http
	r := cfg.Entities[0].Resources[0]
	if r.Reader == nil || r.Reader.HTTP == nil {
		t.Fatalf("Reader.HTTP is nil, want non-nil")
	}
	if r.Reader.HTTP.Verb != "get" {
		t.Errorf("Reader.HTTP.Verb = %q, want %q", r.Reader.HTTP.Verb, "get")
	}
	if r.Reader.HTTP.Path != "/library/LibraryService/book/{key.id}/metadata" {
		t.Errorf("Reader.HTTP.Path = %q, want %q", r.Reader.HTTP.Path, "/library/LibraryService/book/{key.id}/metadata")
	}

	// 验证 writer.update.http
	if r.Writer == nil || r.Writer.Update == nil || r.Writer.Update.HTTP == nil {
		t.Fatalf("Writer.Update.HTTP is nil, want non-nil")
	}
	if r.Writer.Update.HTTP.Verb != "put" {
		t.Errorf("Writer.Update.HTTP.Verb = %q, want %q", r.Writer.Update.HTTP.Verb, "put")
	}
	if r.Writer.Update.HTTP.BodyStyle != "resource" {
		t.Errorf("Writer.Update.HTTP.BodyStyle = %q, want %q", r.Writer.Update.HTTP.BodyStyle, "resource")
	}

	// 验证 custom_methods[].http
	if len(cfg.Services) != 1 {
		t.Fatalf("len(Services) = %d, want 1", len(cfg.Services))
	}
	cms := cfg.Services[0].CustomMethods
	if len(cms) != 1 {
		t.Fatalf("len(CustomMethods) = %d, want 1", len(cms))
	}
	if cms[0].HTTP == nil {
		t.Fatalf("CustomMethods[0].HTTP is nil, want non-nil")
	}
	if cms[0].HTTP.Verb != "post" {
		t.Errorf("CustomMethods[0].HTTP.Verb = %q, want %q", cms[0].HTTP.Verb, "post")
	}
	if cms[0].HTTP.Path != "/library/LibraryService/book/{book_id}:archive" {
		t.Errorf("CustomMethods[0].HTTP.Path = %q, want %q", cms[0].HTTP.Path, "/library/LibraryService/book/{book_id}:archive")
	}
	if cms[0].HTTP.Body != "*" {
		t.Errorf("CustomMethods[0].HTTP.Body = %q, want %q", cms[0].HTTP.Body, "*")
	}
}
