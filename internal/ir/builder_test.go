package ir

import (
	"testing"

	apigenyaml "github.com/acme/apigen/internal/yaml"
)

// TestBuildEntityLevelMethods 测试实体级方法 IR 生成。
func TestBuildEntityLevelMethods(t *testing.T) {
	cfg := &apigenyaml.Config{
		Syntax: "v1",
		Name:   "test",
		Entities: []apigenyaml.Entity{{
			Name:       "book",
			Key:        apigenyaml.KeyDef{Type: "BookId"},
			Create:     &struct{}{},
			Delete:     &struct{}{},
			DeleteSoft: &struct{}{},
			Resources: []apigenyaml.Resource{{
				Name:    "meta",
				Type:    "BookMeta",
				Version: apigenyaml.VersionDef{Kind: "NONE"},
				Reader:  &apigenyaml.ReaderDef{},
				Writer:  &apigenyaml.WriterDef{Update: &apigenyaml.UpdateDef{Mask: true}},
			}},
		}},
	}
	ir, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if len(ir.Entities) != 1 {
		t.Fatalf("len(Entities) = %d, want 1", len(ir.Entities))
	}
	e := ir.Entities[0]
	if e.PascalName != "Book" {
		t.Errorf("PascalName = %q, want %q", e.PascalName, "Book")
	}
	if e.Create == nil {
		t.Fatal("Create is nil")
	}
	if e.Create.RPCName != "CreateBook" {
		t.Errorf("Create.RPCName = %q, want %q", e.Create.RPCName, "CreateBook")
	}
	if len(e.Create.RequestFields) != 1 {
		t.Errorf("Create RequestFields = %d, want 1", len(e.Create.RequestFields))
	}
	if e.Create.RequestFields[0].Number != 1 {
		t.Errorf("Create RequestFields[0].Number = %d, want 1", e.Create.RequestFields[0].Number)
	}
	if e.Create.ResponseKeyField.Number != 1 {
		t.Errorf("Create ResponseKeyField.Number = %d, want 1", e.Create.ResponseKeyField.Number)
	}
	if e.Delete == nil {
		t.Fatal("Delete is nil")
	}
	if e.Delete.RPCName != "DeleteBook" {
		t.Errorf("Delete.RPCName = %q, want %q", e.Delete.RPCName, "DeleteBook")
	}
	if e.DeleteSoft == nil {
		t.Fatal("DeleteSoft is nil")
	}
	if e.DeleteSoft.RPCName != "DeleteBookSoft" {
		t.Errorf("DeleteSoft.RPCName = %q, want %q", e.DeleteSoft.RPCName, "DeleteBookSoft")
	}
}

// TestBuildResourceLevelMethods 测试资源级方法 IR 生成。
func TestBuildResourceLevelMethods(t *testing.T) {
	cfg := &apigenyaml.Config{
		Syntax: "v1",
		Name:   "test",
		Entities: []apigenyaml.Entity{{
			Name: "book",
			Key:  apigenyaml.KeyDef{Type: "BookId"},
			Resources: []apigenyaml.Resource{{
				Name:    "meta",
				Type:    "BookMeta",
				Version: apigenyaml.VersionDef{Kind: "STRONG", Type: "U64"},
				Reader: &apigenyaml.ReaderDef{
					Batch: true,
					List:  true,
				},
				Writer: &apigenyaml.WriterDef{Update: &apigenyaml.UpdateDef{Mask: true}},
			}},
		}},
	}
	ir, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	r := ir.Entities[0].Resources[0]
	if r.PascalName != "Meta" {
		t.Errorf("PascalName = %q, want %q", r.PascalName, "Meta")
	}
	if r.Get == nil {
		t.Fatal("Get is nil")
	}
	if r.Get.RPCName != "GetBookMeta" {
		t.Errorf("Get.RPCName = %q, want %q", r.Get.RPCName, "GetBookMeta")
	}
	if r.BatchGet == nil {
		t.Fatal("BatchGet is nil")
	}
	if r.BatchGet.RPCName != "BatchGetBookMetas" {
		t.Errorf("BatchGet.RPCName = %q, want %q", r.BatchGet.RPCName, "BatchGetBookMetas")
	}
	if r.List == nil {
		t.Fatal("List is nil")
	}
	if r.List.RPCName != "ListBookMetas" {
		t.Errorf("List.RPCName = %q, want %q", r.List.RPCName, "ListBookMetas")
	}
	if r.Update == nil {
		t.Fatal("Update is nil")
	}
	if r.Update.RPCName != "UpdateBookMeta" {
		t.Errorf("Update.RPCName = %q, want %q", r.Update.RPCName, "UpdateBookMeta")
	}
}

// TestBuildWrapperFieldNumbers 测试 wrapper 字段号从 1 连续分配。
func TestBuildWrapperFieldNumbers(t *testing.T) {
	cfg := &apigenyaml.Config{
		Syntax: "v1",
		Name:   "test",
		Entities: []apigenyaml.Entity{{
			Name: "book",
			Key:  apigenyaml.KeyDef{Type: "BookId"},
			Resources: []apigenyaml.Resource{{
				Name:    "meta",
				Type:    "BookMeta",
				Version: apigenyaml.VersionDef{Kind: "STRONG", Type: "U64"},
				Writer:  &apigenyaml.WriterDef{Update: &apigenyaml.UpdateDef{Mask: true}},
			}},
		}},
	}
	ir, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	r := ir.Entities[0].Resources[0]
	if len(r.Update.RequestFields) != 4 {
		t.Fatalf("Update RequestFields = %d, want 4", len(r.Update.RequestFields))
	}
	expected := []int{1, 2, 3, 4}
	for i, f := range r.Update.RequestFields {
		if f.Number != expected[i] {
			t.Errorf("Update RequestFields[%d].Number = %d, want %d", i, f.Number, expected[i])
		}
	}
}

// TestBuildCreateResponseOnlyKey 测试 Create 响应只返回 key。
func TestBuildCreateResponseOnlyKey(t *testing.T) {
	cfg := &apigenyaml.Config{
		Syntax: "v1",
		Name:   "test",
		Entities: []apigenyaml.Entity{{
			Name:   "book",
			Key:    apigenyaml.KeyDef{Type: "BookId"},
			Create: &struct{}{},
			Resources: []apigenyaml.Resource{{
				Name:    "meta",
				Type:    "BookMeta",
				Version: apigenyaml.VersionDef{Kind: "NONE"},
			}},
		}},
	}
	ir, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	e := ir.Entities[0]
	if e.Create.ResponseKeyField.Type != "test.BookId" {
		t.Errorf("Create ResponseKeyField.Type = %q, want %q", e.Create.ResponseKeyField.Type, "test.BookId")
	}
	if e.Create.ResponseKeyField.Number != 1 {
		t.Errorf("Create ResponseKeyField.Number = %d, want 1", e.Create.ResponseKeyField.Number)
	}
}
