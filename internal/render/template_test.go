package render

import (
	"strings"
	"testing"

	"github.com/acme/apigen/internal/ir"
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
