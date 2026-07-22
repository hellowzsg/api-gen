package ir

import (
	"testing"

	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	apigenyaml "github.com/hellowzsg/api-gen/internal/yaml"
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

// TestBuild_HTTPStructuredAnnotations: IR annotations carry structured
// path segments (entity/keyLeaves/resource/suffix), not a pre-baked path
// string; the final path is resolved per service at render time.
func TestBuild_HTTPStructuredAnnotations(t *testing.T) {
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
			Name:       "book",
			Key:        apigenyaml.KeyDef{Type: "BookId"},
			Create:     &struct{}{},
			Delete:     &struct{}{},
			DeleteSoft: &struct{}{},
			Resources: []apigenyaml.Resource{{
				Name:    "meta",
				Type:    "BookMeta",
				Version: apigenyaml.VersionDef{Kind: "NONE"},
				Reader:  &apigenyaml.ReaderDef{Batch: true, List: true},
				Writer:  &apigenyaml.WriterDef{Update: &apigenyaml.UpdateDef{}},
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

	// Create: entity-level POST, no key/resource/suffix.
	ann := e.Create.HTTPAnnotation
	if ann == nil {
		t.Fatal("Create.HTTPAnnotation is nil")
	}
	if ann.Verb != "POST" || ann.Body != "*" {
		t.Errorf("Create = {%s %q}, want {POST \"*\"}", ann.Verb, ann.Body)
	}
	if ann.Entity != "book" || ann.Resource != "" || ann.Suffix != "" || len(ann.KeyLeaves) != 0 {
		t.Errorf("Create segments = {entity:%q resource:%q suffix:%q leaves:%d}, want {book \"\" \"\" 0}",
			ann.Entity, ann.Resource, ann.Suffix, len(ann.KeyLeaves))
	}
	if got := ann.ResolvePath("/library", "LibraryService"); got != "/library/LibraryService/book" {
		t.Errorf("Create ResolvePath = %q, want /library/LibraryService/book", got)
	}

	// Delete: entity-level with key leaves.
	ann = e.Delete.HTTPAnnotation
	if len(ann.KeyLeaves) != 1 || ann.KeyLeaves[0].DotPath != "id" {
		t.Errorf("Delete KeyLeaves = %v, want [id]", ann.KeyLeaves)
	}
	if got := ann.ResolvePath("/library", "LibraryService"); got != "/library/LibraryService/book/{key.id}" {
		t.Errorf("Delete ResolvePath = %q, want /library/LibraryService/book/{key.id}", got)
	}

	// DeleteSoft: entity-level with suffix.
	ann = e.DeleteSoft.HTTPAnnotation
	if ann.Suffix != "deleteSoft" {
		t.Errorf("DeleteSoft Suffix = %q, want deleteSoft", ann.Suffix)
	}
	if got := ann.ResolvePath("/library", "LibraryService"); got != "/library/LibraryService/book/deleteSoft" {
		t.Errorf("DeleteSoft ResolvePath = %q, want /library/LibraryService/book/deleteSoft", got)
	}

	r := e.Resources[0]
	// Get: key leaves + resource.
	if got := r.Get.HTTPAnnotation.ResolvePath("/library", "LibraryService"); got != "/library/LibraryService/book/{key.id}/meta" {
		t.Errorf("Get ResolvePath = %q, want /library/LibraryService/book/{key.id}/meta", got)
	}
	// BatchGet: resource + batchGet suffix, no key leaves.
	ann = r.BatchGet.HTTPAnnotation
	if ann.Resource != "meta" || ann.Suffix != "batchGet" || len(ann.KeyLeaves) != 0 {
		t.Errorf("BatchGet segments = {resource:%q suffix:%q leaves:%d}, want {meta batchGet 0}",
			ann.Resource, ann.Suffix, len(ann.KeyLeaves))
	}
	if got := ann.ResolvePath("/library", "LibraryService"); got != "/library/LibraryService/book/meta/batchGet" {
		t.Errorf("BatchGet ResolvePath = %q, want /library/LibraryService/book/meta/batchGet", got)
	}
	// List: resource + list suffix.
	if got := r.List.HTTPAnnotation.ResolvePath("/library", "LibraryService"); got != "/library/LibraryService/book/meta/list" {
		t.Errorf("List ResolvePath = %q, want /library/LibraryService/book/meta/list", got)
	}
	// Update: key leaves + resource.
	if got := r.Update.HTTPAnnotation.ResolvePath("/library", "LibraryService"); got != "/library/LibraryService/book/{key.id}/meta" {
		t.Errorf("Update ResolvePath = %q, want /library/LibraryService/book/{key.id}/meta", got)
	}
}

// TestHTTPAnnotation_ResolvePath covers ResolvePath composition rules:
// prefix handling, composite keys, and override paths returned verbatim.
func TestHTTPAnnotation_ResolvePath(t *testing.T) {
	// Composite key: multiple leaves become consecutive segments.
	ann := &HTTPAnnotation{
		Verb:      "GET",
		Entity:    "book",
		KeyLeaves: []KeyLeaf{{DotPath: "org.oid"}, {DotPath: "id"}},
		Resource:  "meta",
	}
	if got := ann.ResolvePath("/library", "Svc"); got != "/library/Svc/book/{key.org.oid}/{key.id}/meta" {
		t.Errorf("composite key path = %q, want /library/Svc/book/{key.org.oid}/{key.id}/meta", got)
	}
	// No prefix.
	ann = &HTTPAnnotation{Verb: "POST", Entity: "book", Body: "*"}
	if got := ann.ResolvePath("", "Svc"); got != "/Svc/book" {
		t.Errorf("no-prefix path = %q, want /Svc/book", got)
	}
	// Override (verbatim, custom method): returned as-is regardless of
	// prefix/svc — OverrideTemplateSvc is empty.
	ann = &HTTPAnnotation{
		Verb:         "GET",
		Entity:       "book",
		IsOverride:   true,
		OverridePath: "/custom/{key.id}/items",
	}
	if got := ann.ResolvePath("/library", "AdminService"); got != "/custom/{key.id}/items" {
		t.Errorf("verbatim override path = %q, want /custom/{key.id}/items", got)
	}
	// Override (template, entity-level reader/writer http): the path
	// segment equal to OverrideTemplateSvc is replaced by the rendering
	// service name so each service inheriting the entity gets an isolated
	// route.
	ann = &HTTPAnnotation{
		Verb:                "GET",
		Entity:              "book",
		IsOverride:          true,
		OverridePath:        "/library/LibraryService/book/meta/list",
		OverrideTemplateSvc: "LibraryService",
	}
	if got := ann.ResolvePath("/library", "LibraryService"); got != "/library/LibraryService/book/meta/list" {
		t.Errorf("template override (same svc) = %q, want /library/LibraryService/book/meta/list", got)
	}
	if got := ann.ResolvePath("/library", "AdminService"); got != "/library/AdminService/book/meta/list" {
		t.Errorf("template override (AdminService) = %q, want /library/AdminService/book/meta/list", got)
	}
}

// TestBuildListFilterType: list_config.filter_type sets Filter.Type.
func TestBuildListFilterType(t *testing.T) {
	t.Run("filter_type declared uses custom message type", func(t *testing.T) {
		cfg := &apigenyaml.Config{
			Syntax: "v1", Name: "test",
			Entities: []apigenyaml.Entity{{
				Name: "book",
				Key:  apigenyaml.KeyDef{Type: "BookId"},
				Resources: []apigenyaml.Resource{{
					Name:    "meta",
					Type:    "BookMeta",
					Version: apigenyaml.VersionDef{Kind: "NONE"},
					Reader: &apigenyaml.ReaderDef{
						List: true,
						ListConfig: &apigenyaml.ListConfig{
							TotalSize:  true,
							FilterType: "BookMetaFilter",
						},
					},
				}},
			}},
		}
		irData, err := Build(cfg)
		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}
		l := irData.Entities[0].Resources[0].List
		if l == nil {
			t.Fatal("List is nil")
		}
		if l.Filter.Type != "BookMetaFilter" {
			t.Errorf("Filter.Type = %q, want %q", l.Filter.Type, "BookMetaFilter")
		}
		if l.Filter.Number != 3 {
			t.Errorf("Filter.Number = %d, want 3", l.Filter.Number)
		}
	})

	t.Run("filter_type omitted defaults to string", func(t *testing.T) {
		cfg := &apigenyaml.Config{
			Syntax: "v1", Name: "test",
			Entities: []apigenyaml.Entity{{
				Name: "book",
				Key:  apigenyaml.KeyDef{Type: "BookId"},
				Resources: []apigenyaml.Resource{{
					Name:    "meta",
					Type:    "BookMeta",
					Version: apigenyaml.VersionDef{Kind: "NONE"},
					Reader: &apigenyaml.ReaderDef{
						List: true,
						ListConfig: &apigenyaml.ListConfig{
							TotalSize: true,
						},
					},
				}},
			}},
		}
		irData, err := Build(cfg)
		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}
		l := irData.Entities[0].Resources[0].List
		if l == nil {
			t.Fatal("List is nil")
		}
		if l.Filter.Type != "string" {
			t.Errorf("Filter.Type = %q, want %q", l.Filter.Type, "string")
		}
		if l.Filter.Number != 3 {
			t.Errorf("Filter.Number = %d, want 3", l.Filter.Number)
		}
	})

	t.Run("no list_config defaults to string", func(t *testing.T) {
		cfg := &apigenyaml.Config{
			Syntax: "v1", Name: "test",
			Entities: []apigenyaml.Entity{{
				Name: "book",
				Key:  apigenyaml.KeyDef{Type: "BookId"},
				Resources: []apigenyaml.Resource{{
					Name:    "meta",
					Type:    "BookMeta",
					Version: apigenyaml.VersionDef{Kind: "NONE"},
					Reader: &apigenyaml.ReaderDef{
						List: true,
					},
				}},
			}},
		}
		irData, err := Build(cfg)
		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}
		l := irData.Entities[0].Resources[0].List
		if l == nil {
			t.Fatal("List is nil")
		}
		if l.Filter.Type != "string" {
			t.Errorf("Filter.Type = %q, want %q", l.Filter.Type, "string")
		}
	})
}

// TestBuildPerMethodHTTPOverride: reader.http overrides default verb/path.
func TestBuildPerMethodHTTPOverride(t *testing.T) {
	cfg := &apigenyaml.Config{
		Syntax: "v1",
		Name:   "test",
		Settings: apigenyaml.Settings{
			HTTP: &apigenyaml.HTTPConfig{
				Enable: true,
			},
		},
		Entities: []apigenyaml.Entity{{
			Name: "book",
			Key:  apigenyaml.KeyDef{Type: "BookId"},
			Resources: []apigenyaml.Resource{{
				Name:    "meta",
				Type:    "BookMeta",
				Version: apigenyaml.VersionDef{Kind: "NONE"},
				Reader: &apigenyaml.ReaderDef{
					List: true,
					HTTP: &apigenyaml.HTTPOverride{
						Verb: "get",
						Path: "/custom/path/{key.id}/metadata",
					},
				},
				Writer: &apigenyaml.WriterDef{
					Update: &apigenyaml.UpdateDef{
						Mask: true,
						HTTP: &apigenyaml.HTTPOverride{
							Verb:      "put",
							BodyStyle: "resource",
						},
					},
				},
			}},
		}},
	}
	keyDescs := map[string]protoreflect.MessageDescriptor{
		"test.BookId": buildTestKeyDesc(t),
	}
	irData, err := BuildWithOptions(cfg, BuildOptions{KeyDescriptors: keyDescs})
	if err != nil {
		t.Fatalf("BuildWithOptions failed: %v", err)
	}
	e := irData.Entities[0]
	r := e.Resources[0]

	// reader.http overrides List verb/path.
	if r.List == nil || r.List.HTTPAnnotation == nil {
		t.Fatal("List.HTTPAnnotation should be non-nil")
	}
	if r.List.HTTPAnnotation.Verb != "GET" {
		t.Errorf("List.Verb = %q, want GET (overridden)", r.List.HTTPAnnotation.Verb)
	}
	if !r.List.HTTPAnnotation.IsOverride {
		t.Error("List.IsOverride = false, want true (path overridden)")
	}
	if r.List.HTTPAnnotation.OverridePath != "/custom/path/{key.id}/metadata" {
		t.Errorf("List.OverridePath = %q, want /custom/path/{key.id}/metadata (overridden)", r.List.HTTPAnnotation.OverridePath)
	}
	// List with verb=GET should have no body.
	if r.List.HTTPAnnotation.Body != "" {
		t.Errorf("List.Body = %q, want empty (GET has no body)", r.List.HTTPAnnotation.Body)
	}

	// writer.update.http overrides Update verb and body_style.
	if r.Update == nil || r.Update.HTTPAnnotation == nil {
		t.Fatal("Update.HTTPAnnotation should be non-nil")
	}
	if r.Update.HTTPAnnotation.Verb != "PUT" {
		t.Errorf("Update.Verb = %q, want PUT (overridden)", r.Update.HTTPAnnotation.Verb)
	}
	// body_style: resource → body = resource field name "meta".
	if r.Update.HTTPAnnotation.Body != "meta" {
		t.Errorf("Update.Body = %q, want meta (body_style: resource)", r.Update.HTTPAnnotation.Body)
	}

	// Get (no http override) should keep default verb/path.
	if r.Get == nil || r.Get.HTTPAnnotation == nil {
		t.Fatal("Get.HTTPAnnotation should be non-nil (no override, default)")
	}
	if r.Get.HTTPAnnotation.Verb != "GET" {
		t.Errorf("Get.Verb = %q, want GET (default, not overridden)", r.Get.HTTPAnnotation.Verb)
	}
}

// TestBuildCustomMethodHTTP: custom_methods[].http populates HTTPAnnotation.
func TestBuildCustomMethodHTTP(t *testing.T) {
	cfg := &apigenyaml.Config{
		Syntax: "v1",
		Name:   "test",
		Settings: apigenyaml.Settings{
			HTTP: &apigenyaml.HTTPConfig{Enable: true},
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
			Name: "LibraryService",
			Entities: []apigenyaml.ServiceEntity{{Name: "book"}},
			CustomMethods: []apigenyaml.CustomMethod{{
				Name:     "ArchiveBook",
				Request:  "ArchiveBookRequest",
				Response: "ArchiveBookResponse",
				HTTP: &apigenyaml.HTTPOverride{
					Verb: "post",
					Path: "/library/LibraryService/book/{book_id}:archive",
					Body: "*",
				},
			}},
		}},
	}
	keyDescs := map[string]protoreflect.MessageDescriptor{
		"test.BookId": buildTestKeyDesc(t),
	}
	irData, err := BuildWithOptions(cfg, BuildOptions{KeyDescriptors: keyDescs})
	if err != nil {
		t.Fatalf("BuildWithOptions failed: %v", err)
	}
	if len(irData.Services) != 1 {
		t.Fatalf("len(Services) = %d, want 1", len(irData.Services))
	}
	svc := irData.Services[0]
	if len(svc.CustomMethods) != 1 {
		t.Fatalf("len(CustomMethods) = %d, want 1", len(svc.CustomMethods))
	}
	cm := svc.CustomMethods[0]
	if cm.Name != "ArchiveBook" {
		t.Errorf("CustomMethod.Name = %q, want ArchiveBook", cm.Name)
	}
	if cm.HTTPAnnotation == nil {
		t.Fatal("CustomMethod.HTTPAnnotation should be non-nil")
	}
	if cm.HTTPAnnotation.Verb != "POST" {
		t.Errorf("CustomMethod.Verb = %q, want POST", cm.HTTPAnnotation.Verb)
	}
	if !cm.HTTPAnnotation.IsOverride {
		t.Error("CustomMethod.IsOverride = false, want true (user-declared path)")
	}
	if cm.HTTPAnnotation.OverrideTemplateSvc != "" {
		t.Errorf("CustomMethod.OverrideTemplateSvc = %q, want empty (custom methods are verbatim, never templated)", cm.HTTPAnnotation.OverrideTemplateSvc)
	}
	if cm.HTTPAnnotation.OverridePath != "/library/LibraryService/book/{book_id}:archive" {
		t.Errorf("CustomMethod.OverridePath = %q, want /library/LibraryService/book/{book_id}:archive", cm.HTTPAnnotation.OverridePath)
	}
	if cm.HTTPAnnotation.Body != "*" {
		t.Errorf("CustomMethod.Body = %q, want *", cm.HTTPAnnotation.Body)
	}
}

// TestBodyStyleResource: global body_style:resource derives body=resourceName
// for Update; multi-resource Create + body_style:resource returns error.
func TestBodyStyleResource(t *testing.T) {
	t.Run("Update body_style resource", func(t *testing.T) {
		cfg := &apigenyaml.Config{
			Syntax: "v1",
			Name:   "test",
			Settings: apigenyaml.Settings{
				HTTP: &apigenyaml.HTTPConfig{
					Enable:    true,
					BodyStyle: "resource",
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
					Writer:  &apigenyaml.WriterDef{Update: &apigenyaml.UpdateDef{Mask: true}},
				}},
			}},
		}
		keyDescs := map[string]protoreflect.MessageDescriptor{
			"test.BookId": buildTestKeyDesc(t),
		}
		irData, err := BuildWithOptions(cfg, BuildOptions{KeyDescriptors: keyDescs})
		if err != nil {
			t.Fatalf("BuildWithOptions failed: %v", err)
		}
		r := irData.Entities[0].Resources[0]
		if r.Update == nil || r.Update.HTTPAnnotation == nil {
			t.Fatal("Update.HTTPAnnotation should be non-nil")
		}
		if r.Update.HTTPAnnotation.Body != "meta" {
			t.Errorf("Update.Body = %q, want meta (body_style: resource)", r.Update.HTTPAnnotation.Body)
		}
	})

	t.Run("multi-resource Create body_style resource errors", func(t *testing.T) {
		cfg := &apigenyaml.Config{
			Syntax: "v1",
			Name:   "test",
			Settings: apigenyaml.Settings{
				HTTP: &apigenyaml.HTTPConfig{
					Enable:    true,
					BodyStyle: "resource",
				},
			},
			Entities: []apigenyaml.Entity{{
				Name:  "book",
				Key:   apigenyaml.KeyDef{Type: "BookId"},
				Create: &struct{}{},
				Resources: []apigenyaml.Resource{
					{Name: "meta", Type: "BookMeta", Version: apigenyaml.VersionDef{Kind: "NONE"}, Reader: &apigenyaml.ReaderDef{}},
					{Name: "content", Type: "BookContent", Version: apigenyaml.VersionDef{Kind: "NONE"}, Reader: &apigenyaml.ReaderDef{}},
				},
			}},
		}
		keyDescs := map[string]protoreflect.MessageDescriptor{
			"test.BookId": buildTestKeyDesc(t),
		}
		_, err := BuildWithOptions(cfg, BuildOptions{KeyDescriptors: keyDescs})
		if err == nil {
			t.Fatal("BuildWithOptions should fail for multi-resource Create with body_style: resource")
		}
	})
}
