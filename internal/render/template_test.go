package render

import (
	"strings"
	"testing"

	"github.com/hellowzsg/api-gen/internal/ir"
)

func TestRenderProto_BasicService(t *testing.T) {
	irData := &ir.IR{
		PackageName: "test",
		Entities: []ir.EntityIR{{
			Name: "book", PascalName: "Book", KeyType: "test.BookId",
			Create: &ir.CreateIR{
				RPCName: "CreateBook", RequestName: "CreateBookRequest", ResponseName: "CreateBookResponse",
				RequestFields: []ir.FieldIR{{Name: "meta", Type: "test.BookMeta", Number: 1}},
				ResponseKeyField: ir.FieldIR{Name: "key", Type: "test.BookId", Number: 1},
			},
			Delete: &ir.DeleteIR{
				RPCName: "DeleteBook", RequestName: "DeleteBookRequest", ResponseName: "google.protobuf.Empty",
				KeyField: ir.FieldIR{Name: "key", Type: "test.BookId", Number: 1},
			},
			Resources: []ir.ResourceIR{{
				Name: "meta", PascalName: "Meta", Type: "test.BookMeta",
				Version: ir.VersionIR{Kind: "STRONG", Type: "U64", FieldType: "uint64"},
				Get: &ir.GetIR{
					RPCName: "GetBookMeta", RequestName: "GetBookMetaRequest", ResponseName: "GetBookMetaResponse",
					KeyField: ir.FieldIR{Name: "key", Type: "test.BookId", Number: 1},
					ResourceField: ir.FieldIR{Name: "book_meta", Type: "test.BookMeta", Number: 1},
					VersionField: &ir.FieldIR{Name: "version", Type: "uint64", Number: 2},
				},
			}},
		}},
	}
	svc := ir.ServiceIR{Name: "LibraryService", ProtoPackage: "test.library_service", GoPackage: "library_service", Entities: []ir.ServiceEntityIR{{Name: "book"}}}
	output, err := RenderServiceProto(irData, svc)
	if err != nil {
		t.Fatalf("RenderServiceProto failed: %v", err)
	}
	if !strings.Contains(output, "syntax = \"proto3\"") { t.Error("missing syntax") }
	if !strings.Contains(output, "package test.library_service") { t.Error("missing package") }
	if !strings.Contains(output, "service LibraryService") { t.Error("missing service") }
	if !strings.Contains(output, "rpc CreateBook") { t.Error("missing CreateBook") }
	if !strings.Contains(output, "rpc DeleteBook") { t.Error("missing DeleteBook") }
	if !strings.Contains(output, "rpc GetBookMeta") { t.Error("missing GetBookMeta") }
}

func TestRenderProto_Deterministic(t *testing.T) {
	irData := &ir.IR{PackageName: "test", Entities: []ir.EntityIR{{
		Name: "book", PascalName: "Book", KeyType: "test.BookId",
		Delete: &ir.DeleteIR{RPCName: "DeleteBook", RequestName: "DeleteBookRequest", ResponseName: "google.protobuf.Empty", KeyField: ir.FieldIR{Name: "key", Type: "test.BookId", Number: 1}},
	}}}
	svc := ir.ServiceIR{Name: "LibSvc", ProtoPackage: "test.lib_svc", GoPackage: "lib_svc", Entities: []ir.ServiceEntityIR{{Name: "book"}}}
	out1, _ := RenderServiceProto(irData, svc)
	out2, _ := RenderServiceProto(irData, svc)
	if out1 != out2 { t.Error("output is not deterministic") }
}

func TestRenderProto_ApiLinterExemptions(t *testing.T) {
	irData := &ir.IR{PackageName: "test", Entities: []ir.EntityIR{{
		Name: "book", PascalName: "Book", KeyType: "test.BookId",
		Create: &ir.CreateIR{RPCName: "CreateBook", RequestName: "CreateBookRequest", ResponseName: "CreateBookResponse", RequestFields: []ir.FieldIR{{Name: "meta", Type: "test.BookMeta", Number: 1}}, ResponseKeyField: ir.FieldIR{Name: "key", Type: "test.BookId", Number: 1}},
	}}}
	svc := ir.ServiceIR{Name: "Svc", ProtoPackage: "test.svc", GoPackage: "svc", Entities: []ir.ServiceEntityIR{{Name: "book"}}}
	output, _ := RenderServiceProto(irData, svc)
	if !strings.Contains(output, "api-linter") { t.Error("missing api-linter comments") }
	if !strings.Contains(output, "core::0133::response-message-name") { t.Error("missing 0133 exemption") }
}

func TestRenderProto_ImportSorting(t *testing.T) {
	irData := &ir.IR{PackageName: "test", Entities: []ir.EntityIR{{
		Name: "book", PascalName: "Book", KeyType: "test.BookId",
		Delete: &ir.DeleteIR{RPCName: "DeleteBook", RequestName: "DeleteBookRequest", ResponseName: "google.protobuf.Empty", KeyField: ir.FieldIR{Name: "key", Type: "test.BookId", Number: 1}},
		Resources: []ir.ResourceIR{{Name: "meta", PascalName: "Meta", Type: "test.BookMeta", Version: ir.VersionIR{Kind: "NONE"}, Update: &ir.UpdateIR{RPCName: "UpdateBookMeta", RequestName: "UpdateBookMetaRequest", ResponseName: "google.protobuf.Empty", Mask: true, RequestFields: []ir.FieldIR{{Name: "meta", Type: "test.BookMeta", Number: 1}, {Name: "key", Type: "test.BookId", Number: 2}, {Name: "update_mask", Type: "google.protobuf.FieldMask", Number: 3}}}}},
	}}}
	svc := ir.ServiceIR{Name: "Svc", ProtoPackage: "test.svc", GoPackage: "svc", Entities: []ir.ServiceEntityIR{{Name: "book"}}}
	output, _ := RenderServiceProto(irData, svc)
	emptyIdx := strings.Index(output, "google/protobuf/empty.proto")
	maskIdx := strings.Index(output, "google/protobuf/field_mask.proto")
	if emptyIdx < 0 || maskIdx < 0 { t.Fatal("missing imports") }
	if emptyIdx > maskIdx { t.Error("imports not sorted") }
}

func TestGenerateImports(t *testing.T) {
	tests := []struct{ name string; needEmpty, needMask, needWrapper bool; wantImports []string }{
		{"empty only", true, false, false, []string{"google/protobuf/empty.proto"}},
		{"mask only", false, true, false, []string{"google/protobuf/field_mask.proto"}},
		{"both", true, true, false, []string{"google/protobuf/empty.proto", "google/protobuf/field_mask.proto"}},
		{"wrapper", false, false, true, []string{"google/protobuf/wrappers.proto"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imports := generateImports(tt.needEmpty, tt.needMask, tt.needWrapper, false, nil)
			for _, want := range tt.wantImports {
				found := false
				for _, got := range imports { if got == want { found = true; break } }
				if !found { t.Errorf("import %q not generated", want) }
			}
		})
	}
}

// TestRenderCustomMethodHTTP: custom method with HTTPAnnotation renders
// the google.api.http option inside the RPC body.
func TestRenderCustomMethodHTTP(t *testing.T) {
	irData := &ir.IR{PackageName: "test"}
	svc := ir.ServiceIR{
		Name:          "LibraryService",
		ProtoPackage:  "test.library_service",
		GoPackage:     "library_service",
		Entities:      []ir.ServiceEntityIR{},
		CustomMethods: []ir.CustomMethodIR{{
			Name:     "ArchiveBook",
			Request:  "ArchiveBookRequest",
			Response: "ArchiveBookResponse",
			HTTPAnnotation: &ir.HTTPAnnotation{
				Verb:         "POST",
				IsOverride:   true,
				OverridePath: "/library/LibraryService/book/{book_id}:archive",
				Body:         "*",
			},
		}},
	}
	output, err := RenderServiceProto(irData, svc)
	if err != nil {
		t.Fatalf("RenderServiceProto failed: %v", err)
	}
	// The RPC should have the HTTP annotation inside the body.
	expectedRPC := `  rpc ArchiveBook(ArchiveBookRequest) returns (ArchiveBookResponse) {
    option (google.api.http) = { post: "/library/LibraryService/book/{book_id}:archive" body: "*" };
  }`
	if !strings.Contains(output, expectedRPC) {
		t.Errorf("output does not contain expected custom method RPC with HTTP annotation.\nExpected:\n%s\nGot:\n%s", expectedRPC, output)
	}
}

// TestExemptionsBodyStyle: body_style:resource suppresses core::0133::http-body
// exemption for Create; body_style:wrapper emits it.
func TestExemptionsBodyStyle(t *testing.T) {
	t.Run("wrapper emits http-body exemption", func(t *testing.T) {
		entities := []ir.EntityIR{{
			Name: "book", PascalName: "Book", KeyType: "test.BookId",
			Create: &ir.CreateIR{
				RPCName: "CreateBook", RequestName: "CreateBookRequest", ResponseName: "CreateBookResponse",
				HTTPAnnotation: &ir.HTTPAnnotation{Verb: "POST", Entity: "book", Body: "*"},
			},
		}}
		exemptions := generateExemptions(entities, true)
		found := false
		for _, e := range exemptions {
			if e == "core::0133::http-body" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("core::0133::http-body exemption should be emitted for body_style: wrapper (body: *), got: %v", exemptions)
		}
	})

	t.Run("resource suppresses http-body exemption", func(t *testing.T) {
		entities := []ir.EntityIR{{
			Name: "book", PascalName: "Book", KeyType: "test.BookId",
			Create: &ir.CreateIR{
				RPCName: "CreateBook", RequestName: "CreateBookRequest", ResponseName: "CreateBookResponse",
				HTTPAnnotation: &ir.HTTPAnnotation{Verb: "POST", Entity: "book", Body: "meta"},
			},
		}}
		exemptions := generateExemptions(entities, true)
		for _, e := range exemptions {
			if e == "core::0133::http-body" {
				t.Errorf("core::0133::http-body exemption should NOT be emitted for body_style: resource (body: meta), got: %v", exemptions)
			}
		}
	})
}

// TestUserPathOverrideVerbatim: user-declared http.path (IsOverride=true) is
// emitted verbatim — the renderer never rewrites user-specified routes.
func TestUserPathOverrideVerbatim(t *testing.T) {
	irData := &ir.IR{PackageName: "test", HTTPEnabled: true}
	irData.Entities = []ir.EntityIR{{
		Name: "book", PascalName: "Book", KeyType: "test.BookId",
		Resources: []ir.ResourceIR{{
			Name: "meta", PascalName: "Meta", Type: "test.BookMeta",
			Version: ir.VersionIR{Kind: "NONE"},
			List: &ir.ListIR{
				RPCName: "ListBookMetas", RequestName: "ListBookMetasRequest", ResponseName: "ListBookMetasResponse",
				HTTPAnnotation: &ir.HTTPAnnotation{Verb: "GET", IsOverride: true, OverridePath: "/svc/LibraryService/book/meta/list"},
			},
		}},
	}}
	svc := ir.ServiceIR{
		Name:         "AdminService",
		ProtoPackage: "test.admin_service",
		GoPackage:    "admin_service",
		Entities:     []ir.ServiceEntityIR{{Name: "book"}},
	}
	output, err := RenderServiceProto(irData, svc)
	if err != nil {
		t.Fatalf("RenderServiceProto failed: %v", err)
	}
	// The user-declared path must appear verbatim — no service-segment rewrite.
	if !strings.Contains(output, `get: "/svc/LibraryService/book/meta/list"`) {
		t.Errorf("override path should be verbatim, got:\n%s", output)
	}
}
