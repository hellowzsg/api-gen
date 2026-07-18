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
