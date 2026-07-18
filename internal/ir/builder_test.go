package ir

import (
	"testing"

	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

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

// --- HTTP IR tests (task 3.1) ---

// buildTestKeyDesc builds a simple BookId{string id=1;} message descriptor
// for testing HTTP IR construction. Returns the descriptor keyed by its
// fully-qualified name "test.BookId".
func buildTestKeyDesc(t *testing.T) protoreflect.MessageDescriptor {
	t.Helper()
	fd := &descriptorpb.FileDescriptorProto{
		Syntax:  strPtr2("proto3"),
		Name:    strPtr2("keytest.proto"),
		Package: strPtr2("test"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr2("BookId"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   strPtr2("id"),
						Number: int32Ptr2(1),
						Label:  labelPtr2(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						Type:   typePtr2(descriptorpb.FieldDescriptorProto_TYPE_STRING),
					},
				},
			},
		},
	}
	desc, err := protodesc.NewFile(fd, protoregistry.GlobalFiles)
	if err != nil {
		t.Fatalf("protodesc.NewFile: %v", err)
	}
	return desc.Messages().Get(0)
}

// strPtr2 etc. are test-local helpers (suffixed to avoid clashing with the
// ones defined in keyleaves_test.go in the same package).
func strPtr2(s string) *string                                            { return &s }
func int32Ptr2(i int32) *int32                                            { return &i }
func typePtr2(t descriptorpb.FieldDescriptorProto_Type) *descriptorpb.FieldDescriptorProto_Type { return &t }
func labelPtr2(l descriptorpb.FieldDescriptorProto_Label) *descriptorpb.FieldDescriptorProto_Label { return &l }

// TestBuild_HTTPDisabled: HTTP disabled → IR has no HTTP fields.
func TestBuild_HTTPDisabled(t *testing.T) {
	cfg := &apigenyaml.Config{
		Syntax: "v1",
		Name:   "test",
		Entities: []apigenyaml.Entity{{
			Name: "book",
			Key:  apigenyaml.KeyDef{Type: "BookId"},
			Resources: []apigenyaml.Resource{{
				Name:    "meta",
				Type:    "BookMeta",
				Version: apigenyaml.VersionDef{Kind: "NONE"},
				Reader:  &apigenyaml.ReaderDef{},
			}},
		}},
	}
	irData, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if irData.HTTPEnabled {
		t.Error("HTTPEnabled should be false when HTTP not configured")
	}
	if irData.HTTPPrefix != "" {
		t.Errorf("HTTPPrefix should be empty, got %q", irData.HTTPPrefix)
	}
	e := irData.Entities[0]
	if len(e.KeyLeaves) != 0 {
		t.Errorf("KeyLeaves should be empty when HTTP disabled, got %d", len(e.KeyLeaves))
	}
	if e.Create != nil && e.Create.HTTPAnnotation != nil {
		t.Error("Create.HTTPAnnotation should be nil when HTTP disabled")
	}
	r := e.Resources[0]
	if r.Get != nil && r.Get.HTTPAnnotation != nil {
		t.Error("Get.HTTPAnnotation should be nil when HTTP disabled")
	}
}

// TestBuild_HTTPEnabled: HTTP enabled → IR fills HTTPEnabled/HTTPPrefix/KeyLeaves/HTTPAnnotation.
func TestBuild_HTTPEnabled(t *testing.T) {
	keyDesc := buildTestKeyDesc(t)
	cfg := &apigenyaml.Config{
		Syntax: "v1",
		Name:   "test",
		Settings: apigenyaml.Settings{
			HTTP: &apigenyaml.HTTPConfig{
				Enable: true,
				Prefix: "/api",
			},
		},
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
				Reader: &apigenyaml.ReaderDef{
					Batch: true,
					List:  true,
				},
				Writer: &apigenyaml.WriterDef{Update: &apigenyaml.UpdateDef{}},
			}},
		}},
	}
	keyDescs := map[string]protoreflect.MessageDescriptor{
		"test.BookId": keyDesc,
	}
	irData, err := BuildWithOptions(cfg, BuildOptions{
		KeyDescriptors: keyDescs,
	})
	if err != nil {
		t.Fatalf("BuildWithOptions failed: %v", err)
	}
	if !irData.HTTPEnabled {
		t.Error("HTTPEnabled should be true")
	}
	if irData.HTTPPrefix != "/api" {
		t.Errorf("HTTPPrefix = %q, want %q", irData.HTTPPrefix, "/api")
	}
	e := irData.Entities[0]
	if len(e.KeyLeaves) != 1 {
		t.Fatalf("KeyLeaves = %d, want 1", len(e.KeyLeaves))
	}
	if e.KeyLeaves[0].DotPath != "id" {
		t.Errorf("KeyLeaves[0].DotPath = %q, want %q", e.KeyLeaves[0].DotPath, "id")
	}
	// Verify each method has HTTPAnnotation.
	if e.Create == nil || e.Create.HTTPAnnotation == nil {
		t.Error("Create.HTTPAnnotation should be non-nil when HTTP enabled")
	} else {
		if e.Create.HTTPAnnotation.Verb != "POST" {
			t.Errorf("Create.Verb = %q, want POST", e.Create.HTTPAnnotation.Verb)
		}
		if e.Create.HTTPAnnotation.Body != "*" {
			t.Errorf("Create.Body = %q, want *", e.Create.HTTPAnnotation.Body)
		}
	}
	if e.Delete == nil || e.Delete.HTTPAnnotation == nil {
		t.Error("Delete.HTTPAnnotation should be non-nil when HTTP enabled")
	} else {
		if e.Delete.HTTPAnnotation.Verb != "DELETE" {
			t.Errorf("Delete.Verb = %q, want DELETE", e.Delete.HTTPAnnotation.Verb)
		}
		if e.Delete.HTTPAnnotation.Body != "" {
			t.Errorf("Delete.Body = %q, want empty", e.Delete.HTTPAnnotation.Body)
		}
	}
	if e.DeleteSoft == nil || e.DeleteSoft.HTTPAnnotation == nil {
		t.Error("DeleteSoft.HTTPAnnotation should be non-nil when HTTP enabled")
	} else {
		if e.DeleteSoft.HTTPAnnotation.Verb != "POST" {
			t.Errorf("DeleteSoft.Verb = %q, want POST", e.DeleteSoft.HTTPAnnotation.Verb)
		}
		if e.DeleteSoft.HTTPAnnotation.Body != "*" {
			t.Errorf("DeleteSoft.Body = %q, want *", e.DeleteSoft.HTTPAnnotation.Body)
		}
	}
	r := e.Resources[0]
	if r.Get == nil || r.Get.HTTPAnnotation == nil {
		t.Error("Get.HTTPAnnotation should be non-nil when HTTP enabled")
	} else {
		if r.Get.HTTPAnnotation.Verb != "GET" {
			t.Errorf("Get.Verb = %q, want GET", r.Get.HTTPAnnotation.Verb)
		}
	}
	if r.BatchGet == nil || r.BatchGet.HTTPAnnotation == nil {
		t.Error("BatchGet.HTTPAnnotation should be non-nil when HTTP enabled")
	} else {
		if r.BatchGet.HTTPAnnotation.Verb != "POST" {
			t.Errorf("BatchGet.Verb = %q, want POST", r.BatchGet.HTTPAnnotation.Verb)
		}
		if r.BatchGet.HTTPAnnotation.Body != "*" {
			t.Errorf("BatchGet.Body = %q, want *", r.BatchGet.HTTPAnnotation.Body)
		}
	}
	if r.List == nil || r.List.HTTPAnnotation == nil {
		t.Error("List.HTTPAnnotation should be non-nil when HTTP enabled")
	} else {
		if r.List.HTTPAnnotation.Verb != "POST" {
			t.Errorf("List.Verb = %q, want POST", r.List.HTTPAnnotation.Verb)
		}
		if r.List.HTTPAnnotation.Body != "*" {
			t.Errorf("List.Body = %q, want *", r.List.HTTPAnnotation.Body)
		}
	}
	if r.Update == nil || r.Update.HTTPAnnotation == nil {
		t.Error("Update.HTTPAnnotation should be non-nil when HTTP enabled")
	} else {
		if r.Update.HTTPAnnotation.Verb != "PATCH" {
			t.Errorf("Update.Verb = %q, want PATCH", r.Update.HTTPAnnotation.Verb)
		}
		if r.Update.HTTPAnnotation.Body != "*" {
			t.Errorf("Update.Body = %q, want *", r.Update.HTTPAnnotation.Body)
		}
	}
}

// TestBuild_HTTPPathGeneration: verify path contains prefix/service/collection/key/resource.
func TestBuild_HTTPPathGeneration(t *testing.T) {
	keyDesc := buildTestKeyDesc(t)
	cfg := &apigenyaml.Config{
		Syntax: "v1",
		Name:   "test",
		Settings: apigenyaml.Settings{
			HTTP: &apigenyaml.HTTPConfig{
				Enable: true,
				Prefix: "/library",
			},
		},
		Entities: []apigenyaml.Entity{{
			Name: "book",
			Key:  apigenyaml.KeyDef{Type: "BookId"},
			Resources: []apigenyaml.Resource{{
				Name:    "meta",
				Type:    "BookMeta",
				Version: apigenyaml.VersionDef{Kind: "NONE"},
				Reader:  &apigenyaml.ReaderDef{},
			}},
		}},
		Services: []apigenyaml.Service{{
			Name:     "LibraryService",
			Entities: []apigenyaml.ServiceEntity{{Name: "book"}},
		}},
	}
	keyDescs := map[string]protoreflect.MessageDescriptor{
		"test.BookId": keyDesc,
	}
	irData, err := BuildWithOptions(cfg, BuildOptions{
		KeyDescriptors: keyDescs,
	})
	if err != nil {
		t.Fatalf("BuildWithOptions failed: %v", err)
	}
	e := irData.Entities[0]
	r := e.Resources[0]
	if r.Get == nil || r.Get.HTTPAnnotation == nil {
		t.Fatal("Get.HTTPAnnotation should be non-nil")
	}
	// Expected: /library/LibraryService/book/{key.id}/meta
	expectedPath := "/library/LibraryService/book/{key.id}/meta"
	if r.Get.HTTPAnnotation.Path != expectedPath {
		t.Errorf("Get.Path = %q, want %q", r.Get.HTTPAnnotation.Path, expectedPath)
	}
}
