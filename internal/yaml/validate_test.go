package yaml

import (
	"strings"
	"testing"
)

// TestValidateTypeReference 测试 type_ 短名/全限定名引用解析。
func TestValidateTypeReference(t *testing.T) {
	cfg := &Config{
		Syntax: "v1",
		Name:   "demo.business.book",
		Entities: []Entity{
			{
				Name: "book",
				Key:  KeyDef{Type: "BookId"},
				Resources: []Resource{
					{Name: "meta", Type: "BookMeta", Version: VersionDef{Kind: "NONE"}},
				},
			},
		},
	}
	if err := cfg.ValidateReferences(); err != nil {
		t.Fatalf("ValidateReferences failed for short name: %v", err)
	}
	cfg.Entities[0].Key.Type = "demo.business.book.BookId"
	if err := cfg.ValidateReferences(); err != nil {
		t.Fatalf("ValidateReferences failed for FQMN: %v", err)
	}
	cfg.Entities[0].Key.Type = "common.Foo"
	if err := cfg.ValidateReferences(); err != nil {
		t.Fatalf("ValidateReferences failed for alias.Type: %v", err)
	}
}

// TestValidateServiceEntityReference 测试 service 引用实体校验。
func TestValidateServiceEntityReference(t *testing.T) {
	cfg := &Config{
		Syntax: "v1",
		Name:   "foo",
		Entities: []Entity{
			{Name: "book", Key: KeyDef{Type: "BookId"}, Resources: []Resource{{Name: "meta", Type: "BookMeta", Version: VersionDef{Kind: "NONE"}}}},
		},
		Services: []Service{
			{Name: "LibSvc", Entities: []ServiceEntity{{Name: "book"}}},
		},
	}
	if err := cfg.ValidateReferences(); err != nil {
		t.Fatalf("ValidateReferences failed: %v", err)
	}
}

// TestValidateServiceEntityNotFound 测试 service 引用不存在的实体。
func TestValidateServiceEntityNotFound(t *testing.T) {
	cfg := &Config{
		Syntax: "v1",
		Name:   "foo",
		Entities: []Entity{
			{Name: "book", Key: KeyDef{Type: "BookId"}, Resources: []Resource{{Name: "meta", Type: "BookMeta", Version: VersionDef{Kind: "NONE"}}}},
		},
		Services: []Service{
			{Name: "LibSvc", Entities: []ServiceEntity{{Name: "nonexistent"}}},
		},
	}
	err := cfg.ValidateReferences()
	if err == nil {
		t.Fatal("ValidateReferences should fail for nonexistent entity reference")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should mention 'nonexistent', got: %v", err)
	}
}

// TestValidateServiceFullInheritance 测试 service 不声明 resources 时全量继承。
func TestValidateServiceFullInheritance(t *testing.T) {
	cfg := &Config{
		Syntax: "v1",
		Name:   "foo",
		Entities: []Entity{
			{Name: "book", Key: KeyDef{Type: "BookId"}, Resources: []Resource{{Name: "meta", Type: "BookMeta", Version: VersionDef{Kind: "NONE"}}}},
		},
		Services: []Service{
			{Name: "LibSvc", Entities: []ServiceEntity{{Name: "book"}}},
		},
	}
	if err := cfg.ValidateReferences(); err != nil {
		t.Fatalf("ValidateReferences failed for full inheritance: %v", err)
	}
}

// TestValidateServiceNarrowing 测试 service 收窄声明只能在实体已声明能力内。
func TestValidateServiceNarrowing(t *testing.T) {
	cfg := &Config{
		Syntax: "v1",
		Name:   "foo",
		Entities: []Entity{
			{Name: "book", Key: KeyDef{Type: "BookId"}, Resources: []Resource{{Name: "meta", Type: "BookMeta", Version: VersionDef{Kind: "NONE"}}}},
		},
		Services: []Service{
			{Name: "LibSvc", Entities: []ServiceEntity{{Name: "book", Resources: []Resource{{Name: "meta", Type: "BookMeta", Version: VersionDef{Kind: "NONE"}}}}}},
		},
	}
	if err := cfg.ValidateReferences(); err != nil {
		t.Fatalf("ValidateReferences failed for valid narrowing: %v", err)
	}
	cfg.Services[0].Entities[0].Resources[0].Name = "nonexistent"
	err := cfg.ValidateReferences()
	if err == nil {
		t.Fatal("ValidateReferences should fail for narrowing to nonexistent resource")
	}
}

// TestValidateServiceNarrowingNotAllowed 越权声明实体未声明的能力。
func TestValidateServiceNarrowingNotAllowed(t *testing.T) {
	cfg := &Config{
		Syntax: "v1",
		Name:   "foo",
		Entities: []Entity{
			{Name: "book", Key: KeyDef{Type: "BookId"}, Resources: []Resource{{Name: "meta", Type: "BookMeta", Version: VersionDef{Kind: "NONE"}}}},
		},
		Services: []Service{
			{Name: "LibSvc", Entities: []ServiceEntity{{Name: "book", Resources: []Resource{{Name: "content", Type: "BookContent", Version: VersionDef{Kind: "NONE"}}}}}},
		},
	}
	err := cfg.ValidateReferences()
	if err == nil {
		t.Fatal("ValidateReferences should fail for narrowing to resource not in entity")
	}
}

// --- HTTP 配置校验测试（P1） ---

// TestValidateHTTP_BodyStyleResource 报错 body_style: resource 在 P1 不支持。
func TestValidateHTTP_BodyStyleResource(t *testing.T) {
	cfg := &Config{
		Syntax: "v1",
		Name:   "foo",
		ImportProtos: []Import{
			{Path: "proto/**/*.proto"},
			{Path: "third_party/google/api/annotations.proto"},
		},
		Settings: Settings{
			HTTP: &HTTPConfig{
				Enable:    true,
				BodyStyle: "resource",
			},
		},
		Entities: []Entity{
			{Name: "book", Key: KeyDef{Type: "BookId"}, Resources: []Resource{{Name: "meta", Type: "BookMeta", Version: VersionDef{Kind: "NONE"}}}},
		},
	}
	err := cfg.ValidateReferences()
	if err == nil {
		t.Fatal("ValidateReferences should fail for body_style: resource in P1")
	}
	if !strings.Contains(err.Error(), "body_style") || !strings.Contains(err.Error(), "resource") {
		t.Errorf("error should mention body_style resource, got: %v", err)
	}
}

// TestValidateHTTP_GenerateOpenAPI 报错 generate_openapi 在 P1 不支持。
func TestValidateHTTP_GenerateOpenAPI(t *testing.T) {
	cfg := &Config{
		Syntax: "v1",
		Name:   "foo",
		ImportProtos: []Import{
			{Path: "proto/**/*.proto"},
			{Path: "third_party/google/api/annotations.proto"},
		},
		Settings: Settings{
			HTTP: &HTTPConfig{
				Enable:          true,
				GenerateOpenAPI: true,
			},
		},
		Entities: []Entity{
			{Name: "book", Key: KeyDef{Type: "BookId"}, Resources: []Resource{{Name: "meta", Type: "BookMeta", Version: VersionDef{Kind: "NONE"}}}},
		},
	}
	err := cfg.ValidateReferences()
	if err == nil {
		t.Fatal("ValidateReferences should fail for generate_openapi in P1")
	}
	if !strings.Contains(err.Error(), "openapi") {
		t.Errorf("error should mention openapi, got: %v", err)
	}
}

// TestValidateHTTP_NoGoogleapis HTTP 启用但无 googleapis 依赖时报错。
// Uses a non-glob path (no "**") that doesn't reference google/api, so it
// should fail-fast.
func TestValidateHTTP_NoGoogleapis(t *testing.T) {
	cfg := &Config{
		Syntax: "v1",
		Name:   "foo",
		ImportProtos: []Import{
			{Path: "proto/book.proto"}, // explicit file, no glob, no google/api
		},
		Settings: Settings{
			HTTP: &HTTPConfig{Enable: true},
		},
		Entities: []Entity{
			{Name: "book", Key: KeyDef{Type: "BookId"}, Resources: []Resource{{Name: "meta", Type: "BookMeta", Version: VersionDef{Kind: "NONE"}}}},
		},
	}
	err := cfg.ValidateReferences()
	if err == nil {
		t.Fatal("ValidateReferences should fail when HTTP enabled but no googleapis dependency")
	}
	if !strings.Contains(err.Error(), "googleapis") && !strings.Contains(err.Error(), "annotations.proto") {
		t.Errorf("error should mention googleapis or annotations.proto, got: %v", err)
	}
}

// TestValidateHTTP_GoogleapisViaPath HTTP 启用且有 googleapis path 依赖时通过。
func TestValidateHTTP_GoogleapisViaPath(t *testing.T) {
	cfg := &Config{
		Syntax: "v1",
		Name:   "foo",
		ImportProtos: []Import{
			{Path: "proto/**/*.proto"},
			{Path: "third_party/google/api/annotations.proto"},
		},
		Settings: Settings{
			HTTP: &HTTPConfig{Enable: true, Prefix: "/api"},
		},
		Entities: []Entity{
			{Name: "book", Key: KeyDef{Type: "BookId"}, Resources: []Resource{{Name: "meta", Type: "BookMeta", Version: VersionDef{Kind: "NONE"}}}},
		},
	}
	if err := cfg.ValidateReferences(); err != nil {
		t.Fatalf("ValidateReferences should pass with googleapis path dependency: %v", err)
	}
}

// TestValidateHTTP_GoogleapisViaGit HTTP 启用且有 googleapis git 依赖时通过。
func TestValidateHTTP_GoogleapisViaGit(t *testing.T) {
	cfg := &Config{
		Syntax: "v1",
		Name:   "foo",
		ImportProtos: []Import{
			{Path: "proto/**/*.proto"},
			{Git: "https://github.com/googleapis/googleapis", Ref: "main", Subdir: "google/api"},
		},
		Settings: Settings{
			HTTP: &HTTPConfig{Enable: true},
		},
		Entities: []Entity{
			{Name: "book", Key: KeyDef{Type: "BookId"}, Resources: []Resource{{Name: "meta", Type: "BookMeta", Version: VersionDef{Kind: "NONE"}}}},
		},
	}
	if err := cfg.ValidateReferences(); err != nil {
		t.Fatalf("ValidateReferences should pass with googleapis git dependency: %v", err)
	}
}

// TestValidateHTTP_GoogleapisViaBSR HTTP 启用且有 googleapis BSR 依赖时通过。
func TestValidateHTTP_GoogleapisViaBSR(t *testing.T) {
	cfg := &Config{
		Syntax: "v1",
		Name:   "foo",
		ImportProtos: []Import{
			{Path: "proto/**/*.proto"},
			{BSR: "buf.build/googleapis/googleapis"},
		},
		Settings: Settings{
			HTTP: &HTTPConfig{Enable: true},
		},
		Entities: []Entity{
			{Name: "book", Key: KeyDef{Type: "BookId"}, Resources: []Resource{{Name: "meta", Type: "BookMeta", Version: VersionDef{Kind: "NONE"}}}},
		},
	}
	if err := cfg.ValidateReferences(); err != nil {
		t.Fatalf("ValidateReferences should pass with googleapis BSR dependency: %v", err)
	}
}

// TestValidateHTTP_DisabledNoGoogleapisCheck HTTP 关闭时不校验 googleapis。
func TestValidateHTTP_DisabledNoGoogleapisCheck(t *testing.T) {
	cfg := &Config{
		Syntax: "v1",
		Name:   "foo",
		ImportProtos: []Import{
			{Path: "proto/**/*.proto"},
		},
		Settings: Settings{
			HTTP: &HTTPConfig{Enable: false},
		},
		Entities: []Entity{
			{Name: "book", Key: KeyDef{Type: "BookId"}, Resources: []Resource{{Name: "meta", Type: "BookMeta", Version: VersionDef{Kind: "NONE"}}}},
		},
	}
	if err := cfg.ValidateReferences(); err != nil {
		t.Fatalf("ValidateReferences should pass when HTTP disabled (no googleapis check): %v", err)
	}
}

// TestValidateHTTP_NilNoGoogleapisCheck HTTP 未声明（nil）时不校验 googleapis。
func TestValidateHTTP_NilNoGoogleapisCheck(t *testing.T) {
	cfg := &Config{
		Syntax: "v1",
		Name:   "foo",
		ImportProtos: []Import{
			{Path: "proto/**/*.proto"},
		},
		Settings: Settings{},
		Entities: []Entity{
			{Name: "book", Key: KeyDef{Type: "BookId"}, Resources: []Resource{{Name: "meta", Type: "BookMeta", Version: VersionDef{Kind: "NONE"}}}},
		},
	}
	if err := cfg.ValidateReferences(); err != nil {
		t.Fatalf("ValidateReferences should pass when HTTP nil (no googleapis check): %v", err)
	}
}

// TestValidateHTTP_BodyStyleWrapperDefault body_style 缺省或 wrapper 时通过。
func TestValidateHTTP_BodyStyleWrapperDefault(t *testing.T) {
	cfg := &Config{
		Syntax: "v1",
		Name:   "foo",
		ImportProtos: []Import{
			{Path: "proto/**/*.proto"},
			{Path: "third_party/google/api/annotations.proto"},
		},
		Settings: Settings{
			HTTP: &HTTPConfig{Enable: true}, // body_style 缺省
		},
		Entities: []Entity{
			{Name: "book", Key: KeyDef{Type: "BookId"}, Resources: []Resource{{Name: "meta", Type: "BookMeta", Version: VersionDef{Kind: "NONE"}}}},
		},
	}
	if err := cfg.ValidateReferences(); err != nil {
		t.Fatalf("ValidateReferences should pass with default body_style: %v", err)
	}
	cfg.Settings.HTTP.BodyStyle = "wrapper"
	if err := cfg.ValidateReferences(); err != nil {
		t.Fatalf("ValidateReferences should pass with body_style=wrapper: %v", err)
	}
}
