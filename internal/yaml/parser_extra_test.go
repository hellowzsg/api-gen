package yaml

import (
	"strings"
	"testing"
)

// TestParseEmptyEntities: entities: [] (empty list) should pass parse but
// the validator should accept it (entities is present, just empty).
func TestParseEmptyEntities(t *testing.T) {
	input := `
syntax: v1
name: foo
entities: []
`
	cfg, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse should succeed for empty entities list: %v", err)
	}
	if len(cfg.Entities) != 0 {
		t.Errorf("Entities = %d, want 0", len(cfg.Entities))
	}
}

// TestParseUnknownField: unknown top-level field should fail (KnownFields=true).
func TestParseUnknownField(t *testing.T) {
	input := `
syntax: v1
name: foo
unknown_field: bar
entities: []
`
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("Parse should fail for unknown field")
	}
}

// TestParseUnknownNestedField: unknown nested field should also fail.
func TestParseUnknownNestedField(t *testing.T) {
	input := `
syntax: v1
name: foo
settings:
  unknown_setting: bar
entities: []
`
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("Parse should fail for unknown nested field")
	}
}

// TestParseMultipleEntities: multiple entities parsed correctly.
func TestParseMultipleEntities(t *testing.T) {
	input := `
syntax: v1
name: catalog
entities:
  - name: book
    key: { type_: BookId }
    resources:
      - name: meta
        type_: BookMeta
        version: { kind: NONE }
  - name: author
    key: { type_: AuthorId }
    resources:
      - name: profile
        type_: AuthorProfile
        version: { kind: STRONG, type: U64 }
`
	cfg, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(cfg.Entities) != 2 {
		t.Fatalf("Entities = %d, want 2", len(cfg.Entities))
	}
	if cfg.Entities[0].Name != "book" {
		t.Errorf("Entity[0].Name = %q, want book", cfg.Entities[0].Name)
	}
	if cfg.Entities[1].Name != "author" {
		t.Errorf("Entity[1].Name = %q, want author", cfg.Entities[1].Name)
	}
	if cfg.Entities[1].Resources[0].Version.Kind != "STRONG" {
		t.Errorf("Entity[1] version kind = %q, want STRONG", cfg.Entities[1].Resources[0].Version.Kind)
	}
}

// TestParseMultipleResources: entity with multiple resources.
func TestParseMultipleResources(t *testing.T) {
	input := `
syntax: v1
name: foo
entities:
  - name: book
    key: { type_: BookId }
    resources:
      - name: meta
        type_: BookMeta
        version: { kind: STRONG, type: U64 }
      - name: content
        type_: BookContent
        version: { kind: NONE }
      - name: review
        type_: BookReview
        version: { kind: WEAK, type: STRING }
`
	cfg, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	e := cfg.Entities[0]
	if len(e.Resources) != 3 {
		t.Fatalf("Resources = %d, want 3", len(e.Resources))
	}
	if e.Resources[0].Name != "meta" {
		t.Errorf("Resource[0].Name = %q, want meta", e.Resources[0].Name)
	}
	if e.Resources[2].Version.Kind != "WEAK" {
		t.Errorf("Resource[2] version kind = %q, want WEAK", e.Resources[2].Version.Kind)
	}
}

// TestParseListConfig: reader.list_config.total_size parsing.
func TestParseListConfig(t *testing.T) {
	t.Run("total_size true", func(t *testing.T) {
		input := `
syntax: v1
name: foo
entities:
  - name: book
    key: { type_: BookId }
    resources:
      - name: meta
        type_: BookMeta
        version: { kind: NONE }
        reader:
          list: true
          list_config:
            total_size: true
`
		cfg, err := Parse(strings.NewReader(input))
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}
		r := cfg.Entities[0].Resources[0]
		if r.Reader == nil || r.Reader.ListConfig == nil {
			t.Fatal("ListConfig is nil")
		}
		if !r.Reader.ListConfig.TotalSize {
			t.Error("TotalSize = false, want true")
		}
	})

	t.Run("total_size false", func(t *testing.T) {
		input := `
syntax: v1
name: foo
entities:
  - name: book
    key: { type_: BookId }
    resources:
      - name: meta
        type_: BookMeta
        version: { kind: NONE }
        reader:
          list: true
          list_config:
            total_size: false
`
		cfg, err := Parse(strings.NewReader(input))
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}
		r := cfg.Entities[0].Resources[0]
		if r.Reader == nil || r.Reader.ListConfig == nil {
			t.Fatal("ListConfig is nil")
		}
		if r.Reader.ListConfig.TotalSize {
			t.Error("TotalSize = true, want false")
		}
	})

	t.Run("list_config omitted", func(t *testing.T) {
		input := `
syntax: v1
name: foo
entities:
  - name: book
    key: { type_: BookId }
    resources:
      - name: meta
        type_: BookMeta
        version: { kind: NONE }
        reader:
          list: true
`
		cfg, err := Parse(strings.NewReader(input))
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}
		r := cfg.Entities[0].Resources[0]
		if r.Reader.ListConfig != nil {
			t.Errorf("ListConfig = %+v, want nil", r.Reader.ListConfig)
		}
	})
}

// TestParseMultipleServices: multiple services referencing same entity.
func TestParseMultipleServices(t *testing.T) {
	input := `
syntax: v1
name: foo
entities:
  - name: book
    key: { type_: BookId }
    resources:
      - name: meta
        type_: BookMeta
        version: { kind: NONE }
services:
  - name: LibraryService
    entities:
      - name: book
  - name: AdminService
    entities:
      - name: book
        resources:
          - name: meta
            reader: { list: true }
  - name: ReadOnlyService
    entities:
      - name: book
`
	cfg, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(cfg.Services) != 3 {
		t.Fatalf("Services = %d, want 3", len(cfg.Services))
	}
	if cfg.Services[0].Name != "LibraryService" {
		t.Errorf("Service[0].Name = %q, want LibraryService", cfg.Services[0].Name)
	}
	if cfg.Services[1].Name != "AdminService" {
		t.Errorf("Service[1].Name = %q, want AdminService", cfg.Services[1].Name)
	}
}

// TestParseCustomMethods: custom_methods parsing with and without http.
func TestParseCustomMethods(t *testing.T) {
	t.Run("with http", func(t *testing.T) {
		input := `
syntax: v1
name: foo
entities:
  - name: book
    key: { type_: BookId }
    resources:
      - name: meta
        type_: BookMeta
        version: { kind: NONE }
services:
  - name: Svc
    entities:
      - name: book
    custom_methods:
      - name: ArchiveBook
        request: ArchiveBookRequest
        response: ArchiveBookResponse
        http:
          verb: post
          path: /svc/book/{book_id}:archive
          body: "*"
      - name: PublishBook
        request: PublishBookRequest
        response: PublishBookResponse
`
		cfg, err := Parse(strings.NewReader(input))
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}
		cms := cfg.Services[0].CustomMethods
		if len(cms) != 2 {
			t.Fatalf("CustomMethods = %d, want 2", len(cms))
		}
		if cms[0].Name != "ArchiveBook" {
			t.Errorf("CustomMethod[0].Name = %q, want ArchiveBook", cms[0].Name)
		}
		if cms[0].HTTP == nil {
			t.Fatal("CustomMethod[0].HTTP is nil")
		}
		if cms[1].HTTP != nil {
			t.Error("CustomMethod[1].HTTP should be nil (no http block)")
		}
	})
}

// TestParseImportProtosVariants: all import_protos source types parsed.
func TestParseImportProtosVariants(t *testing.T) {
	input := `
syntax: v1
name: foo
import_protos:
  - path: "proto/**/*.proto"
  - git: https://github.com/googleapis/googleapis
    ref: main
    subdir: google
  - bsr: buf.build/googleapis/googleapis
  - path: "third_party/google/api/annotations.proto"
    alias: ga
entities:
  - name: book
    key: { type_: BookId }
    resources:
      - name: meta
        type_: BookMeta
        version: { kind: NONE }
`
	cfg, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(cfg.ImportProtos) != 4 {
		t.Fatalf("ImportProtos = %d, want 4", len(cfg.ImportProtos))
	}
	if cfg.ImportProtos[0].Path != "proto/**/*.proto" {
		t.Errorf("ImportProtos[0].Path = %q", cfg.ImportProtos[0].Path)
	}
	if cfg.ImportProtos[1].Git != "https://github.com/googleapis/googleapis" {
		t.Errorf("ImportProtos[1].Git = %q", cfg.ImportProtos[1].Git)
	}
	if cfg.ImportProtos[1].Ref != "main" {
		t.Errorf("ImportProtos[1].Ref = %q", cfg.ImportProtos[1].Ref)
	}
	if cfg.ImportProtos[1].Subdir != "google" {
		t.Errorf("ImportProtos[1].Subdir = %q", cfg.ImportProtos[1].Subdir)
	}
	if cfg.ImportProtos[2].BSR != "buf.build/googleapis/googleapis" {
		t.Errorf("ImportProtos[2].BSR = %q", cfg.ImportProtos[2].BSR)
	}
	if cfg.ImportProtos[3].Alias != "ga" {
		t.Errorf("ImportProtos[3].Alias = %q", cfg.ImportProtos[3].Alias)
	}
}

// TestParseOptions: resource-level options parsing.
func TestParseOptions(t *testing.T) {
	input := `
syntax: v1
name: foo
entities:
  - name: book
    key: { type_: BookId }
    resources:
      - name: meta
        type_: BookMeta
        version: { kind: NONE }
        options:
          - target: field
            path: meta
            option: acme.cache
            value: true
          - target: message
            option: acme.deprecated
            value: false
`
	cfg, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	r := cfg.Entities[0].Resources[0]
	if len(r.Options) != 2 {
		t.Fatalf("Options = %d, want 2", len(r.Options))
	}
	if r.Options[0].Target != "field" {
		t.Errorf("Options[0].Target = %q, want field", r.Options[0].Target)
	}
	if r.Options[1].Target != "message" {
		t.Errorf("Options[1].Target = %q, want message", r.Options[1].Target)
	}
}

// TestParseVersionKinds: all version kinds and types.
func TestParseVersionKinds(t *testing.T) {
	tests := []struct {
		name      string
		kind      string
		typeName  string
	}{
		{"STRONG_U64", "STRONG", "U64"},
		{"STRONG_U32", "STRONG", "U32"},
		{"STRONG_STRING", "STRONG", "STRING"},
		{"WEAK_U64", "WEAK", "U64"},
		{"WEAK_U32", "WEAK", "U32"},
		{"WEAK_STRING", "WEAK", "STRING"},
		{"NONE", "NONE", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			versionYAML := "{ kind: " + tt.kind
			if tt.typeName != "" {
				versionYAML += ", type: " + tt.typeName
			}
			versionYAML += " }"
			input := `
syntax: v1
name: foo
entities:
  - name: book
    key: { type_: BookId }
    resources:
      - name: meta
        type_: BookMeta
        version: ` + versionYAML + `
`
			cfg, err := Parse(strings.NewReader(input))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			v := cfg.Entities[0].Resources[0].Version
			if v.Kind != tt.kind {
				t.Errorf("Kind = %q, want %q", v.Kind, tt.kind)
			}
			if tt.typeName != "" && v.Type != tt.typeName {
				t.Errorf("Type = %q, want %q", v.Type, tt.typeName)
			}
		})
	}
}

// TestParseEntityNameValidation: entity name required, resource name required.
func TestParseEntityNameValidation(t *testing.T) {
	t.Run("entity missing name", func(t *testing.T) {
		input := `
syntax: v1
name: foo
entities:
  - key: { type_: BookId }
    resources:
      - name: meta
        type_: BookMeta
        version: { kind: NONE }
`
		_, err := Parse(strings.NewReader(input))
		if err == nil {
			t.Fatal("Parse should fail for entity without name")
		}
	})

	t.Run("resource missing name", func(t *testing.T) {
		input := `
syntax: v1
name: foo
entities:
  - name: book
    key: { type_: BookId }
    resources:
      - type_: BookMeta
        version: { kind: NONE }
`
		_, err := Parse(strings.NewReader(input))
		if err == nil {
			t.Fatal("Parse should fail for resource without name")
		}
	})

	t.Run("resource missing type_", func(t *testing.T) {
		input := `
syntax: v1
name: foo
entities:
  - name: book
    key: { type_: BookId }
    resources:
      - name: meta
        version: { kind: NONE }
`
		_, err := Parse(strings.NewReader(input))
		if err == nil {
			t.Fatal("Parse should fail for resource without type_")
		}
	})

	t.Run("resource missing version kind", func(t *testing.T) {
		input := `
syntax: v1
name: foo
entities:
  - name: book
    key: { type_: BookId }
    resources:
      - name: meta
        type_: BookMeta
`
		_, err := Parse(strings.NewReader(input))
		if err == nil {
			t.Fatal("Parse should fail for resource without version.kind")
		}
	})

	t.Run("entity missing key type_", func(t *testing.T) {
		input := `
syntax: v1
name: foo
entities:
  - name: book
    resources:
      - name: meta
        type_: BookMeta
        version: { kind: NONE }
`
		_, err := Parse(strings.NewReader(input))
		if err == nil {
			t.Fatal("Parse should fail for entity without key.type_")
		}
	})

	t.Run("entity missing resources", func(t *testing.T) {
		input := `
syntax: v1
name: foo
entities:
  - name: book
    key: { type_: BookId }
`
		_, err := Parse(strings.NewReader(input))
		if err == nil {
			t.Fatal("Parse should fail for entity without resources")
		}
	})
}

// TestParseHTTPConfigFull: full HTTP config block parsing.
func TestParseHTTPConfigFull(t *testing.T) {
	input := `
syntax: v1
name: foo
import_protos:
  - path: "proto/**/*.proto"
settings:
  go_repo: github.com/acme/test
  js_repo: "@acme/test"
  out:
    proto: generated/proto
    go: generated/go
    js: generated/js
    openapi: generated/openapi
  http:
    enable: true
    prefix: /api/v1
    body_style: wrapper
    generate_openapi: true
  plugins:
    js: [es]
entities:
  - name: book
    key: { type_: BookId }
    resources:
      - name: meta
        type_: BookMeta
        version: { kind: NONE }
`
	cfg, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	hc := cfg.Settings.HTTP
	if hc == nil {
		t.Fatal("HTTP config is nil")
	}
	if !hc.Enable {
		t.Error("Enable = false, want true")
	}
	if hc.Prefix != "/api/v1" {
		t.Errorf("Prefix = %q, want /api/v1", hc.Prefix)
	}
	if hc.BodyStyle != "wrapper" {
		t.Errorf("BodyStyle = %q, want wrapper", hc.BodyStyle)
	}
	if !hc.GenerateOpenAPI {
		t.Error("GenerateOpenAPI = false, want true")
	}
	if cfg.Settings.GoRepo != "github.com/acme/test" {
		t.Errorf("GoRepo = %q", cfg.Settings.GoRepo)
	}
	if cfg.Settings.JsRepo != "@acme/test" {
		t.Errorf("JsRepo = %q", cfg.Settings.JsRepo)
	}
	if cfg.Settings.Out.OpenAPI != "generated/openapi" {
		t.Errorf("Out.OpenAPI = %q", cfg.Settings.Out.OpenAPI)
	}
}

// TestParseReaderOnly/WriterOnly: reader-only and writer-only resources.
func TestParseReaderOnly(t *testing.T) {
	input := `
syntax: v1
name: foo
entities:
  - name: book
    key: { type_: BookId }
    resources:
      - name: meta
        type_: BookMeta
        version: { kind: NONE }
        reader: { batch: true, list: true }
`
	cfg, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	r := cfg.Entities[0].Resources[0]
	if r.Reader == nil {
		t.Fatal("Reader is nil")
	}
	if !r.Reader.Batch || !r.Reader.List {
		t.Errorf("Reader: Batch=%v List=%v, want both true", r.Reader.Batch, r.Reader.List)
	}
	if r.Writer != nil {
		t.Error("Writer should be nil for reader-only resource")
	}
}

func TestParseWriterOnly(t *testing.T) {
	input := `
syntax: v1
name: foo
entities:
  - name: book
    key: { type_: BookId }
    resources:
      - name: meta
        type_: BookMeta
        version: { kind: NONE }
        writer:
          update: { mask: true }
`
	cfg, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	r := cfg.Entities[0].Resources[0]
	if r.Writer == nil || r.Writer.Update == nil {
		t.Fatal("Writer.Update is nil")
	}
	if !r.Writer.Update.Mask {
		t.Error("Mask = false, want true")
	}
	if r.Reader != nil {
		t.Error("Reader should be nil for writer-only resource")
	}
}
