package render

import (
	"strings"
	"testing"

	"github.com/acme/apigen/internal/ir"
	apigenyaml "github.com/acme/apigen/internal/yaml"
)

// TestRenderHTTPAnnotation_Format tests the google.api.http option format.
func TestRenderHTTPAnnotation_Format(t *testing.T) {
	tests := []struct {
		name string
		ann  *ir.HTTPAnnotation
		want string
	}{
		{
			name: "POST with body star",
			ann:  &ir.HTTPAnnotation{Verb: "POST", Path: "/api/svc/ent", Body: "*"},
			want: `option (google.api.http) = { post: "/api/svc/ent" body: "*" };`,
		},
		{
			name: "GET no body",
			ann:  &ir.HTTPAnnotation{Verb: "GET", Path: "/api/svc/ent/{key.id}/meta", Body: ""},
			want: `option (google.api.http) = { get: "/api/svc/ent/{key.id}/meta" };`,
		},
		{
			name: "DELETE no body",
			ann:  &ir.HTTPAnnotation{Verb: "DELETE", Path: "/api/svc/ent/{key.id}", Body: ""},
			want: `option (google.api.http) = { delete: "/api/svc/ent/{key.id}" };`,
		},
		{
			name: "PATCH with body star",
			ann:  &ir.HTTPAnnotation{Verb: "PATCH", Path: "/api/svc/ent/{key.id}/meta", Body: "*"},
			want: `option (google.api.http) = { patch: "/api/svc/ent/{key.id}/meta" body: "*" };`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderHTTPAnnotation(tt.ann)
			if got != tt.want {
				t.Errorf("RenderHTTPAnnotation = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRenderHTTPAnnotation_NilReturnsEmpty: nil annotation → empty string.
func TestRenderHTTPAnnotation_NilReturnsEmpty(t *testing.T) {
	if got := RenderHTTPAnnotation(nil); got != "" {
		t.Errorf("RenderHTTPAnnotation(nil) = %q, want empty", got)
	}
}

// TestRenderServiceProto_HTTPAnnotation: HTTP enabled → proto contains
// google.api.http option on RPCs and annotations import.
func TestRenderServiceProto_HTTPAnnotation(t *testing.T) {
	irData := &ir.IR{
		PackageName: "test",
		HTTPEnabled: true,
		HTTPPrefix:  "/library",
		Entities: []ir.EntityIR{
			{
				Name:       "book",
				PascalName: "Book",
				KeyType:    "test.BookId",
				KeyLeaves:  []ir.KeyLeaf{{DotPath: "id", FieldType: "string"}},
				Create: &ir.CreateIR{
					RPCName:          "CreateBook",
					RequestName:      "CreateBookRequest",
					ResponseName:     "CreateBookResponse",
					ResponseKeyField: ir.FieldIR{Name: "key", Type: "test.BookId", Number: 1},
					HTTPAnnotation:   &ir.HTTPAnnotation{Verb: "POST", Path: "/library/LibraryService/book", Body: "*"},
				},
				Resources: []ir.ResourceIR{
					{
						Name: "meta", PascalName: "Meta", Type: "test.BookMeta",
						Version: ir.VersionIR{Kind: "NONE"},
						Get: &ir.GetIR{
							RPCName:       "GetBookMeta",
							RequestName:   "GetBookMetaRequest",
							ResponseName:  "GetBookMetaResponse",
							KeyField:      ir.FieldIR{Name: "key", Type: "test.BookId", Number: 1},
							ResourceField: ir.FieldIR{Name: "book_meta", Type: "test.BookMeta", Number: 1},
							HTTPAnnotation: &ir.HTTPAnnotation{Verb: "GET", Path: "/library/LibraryService/book/{key.id}/meta", Body: ""},
						},
					},
				},
			},
		},
	}
	svc := ir.ServiceIR{
		Name:         "LibraryService",
		ProtoPackage: "test.library_service",
		GoPackage:    "library_service",
		Entities:     []ir.ServiceEntityIR{{Name: "book"}},
	}
	output, err := RenderServiceProto(irData, svc)
	if err != nil {
		t.Fatalf("RenderServiceProto failed: %v", err)
	}
	// Verify annotations import present.
	if !strings.Contains(output, `import "google/api/annotations.proto";`) {
		t.Error("missing google/api/annotations.proto import")
	}
	// Verify Create RPC has http annotation.
	if !strings.Contains(output, `option (google.api.http) = { post: "/library/LibraryService/book" body: "*" };`) {
		t.Error("missing CreateBook http annotation")
	}
	// Verify Get RPC has http annotation.
	if !strings.Contains(output, `option (google.api.http) = { get: "/library/LibraryService/book/{key.id}/meta" };`) {
		t.Error("missing GetBookMeta http annotation")
	}
}

// TestRenderServiceProto_HTTPDisabled: HTTP disabled → no annotations import,
// no google.api.http options.
func TestRenderServiceProto_HTTPDisabled(t *testing.T) {
	irData := &ir.IR{
		PackageName: "test",
		HTTPEnabled: false,
		Entities: []ir.EntityIR{
			{
				Name:       "book",
				PascalName: "Book",
				KeyType:    "test.BookId",
				Create: &ir.CreateIR{
					RPCName:          "CreateBook",
					RequestName:      "CreateBookRequest",
					ResponseName:     "CreateBookResponse",
					ResponseKeyField: ir.FieldIR{Name: "key", Type: "test.BookId", Number: 1},
				},
			},
		},
	}
	svc := ir.ServiceIR{
		Name:         "LibraryService",
		ProtoPackage: "test.library_service",
		GoPackage:    "library_service",
		Entities:     []ir.ServiceEntityIR{{Name: "book"}},
	}
	output, err := RenderServiceProto(irData, svc)
	if err != nil {
		t.Fatalf("RenderServiceProto failed: %v", err)
	}
	if strings.Contains(output, "google/api/annotations.proto") {
		t.Error("should not contain annotations import when HTTP disabled")
	}
	if strings.Contains(output, "google.api.http") {
		t.Error("should not contain google.api.http option when HTTP disabled")
	}
}

// TestRenderServiceProto_AllMethodVerbs: verify all 7 method types get
// correct verb/path/body annotations.
func TestRenderServiceProto_AllMethodVerbs(t *testing.T) {
	irData := &ir.IR{
		PackageName: "test",
		HTTPEnabled: true,
		HTTPPrefix:  "/api",
		Entities: []ir.EntityIR{{
			Name:       "book",
			PascalName: "Book",
			KeyType:    "test.BookId",
			KeyLeaves:  []ir.KeyLeaf{{DotPath: "id", FieldType: "string"}},
			Create: &ir.CreateIR{
				RPCName: "CreateBook", RequestName: "CreateBookRequest", ResponseName: "CreateBookResponse",
				HTTPAnnotation: &ir.HTTPAnnotation{Verb: "POST", Path: "/api/Svc/book", Body: "*"},
			},
			Delete: &ir.DeleteIR{
				RPCName: "DeleteBook", RequestName: "DeleteBookRequest", ResponseName: "google.protobuf.Empty",
				HTTPAnnotation: &ir.HTTPAnnotation{Verb: "DELETE", Path: "/api/Svc/book/{key.id}", Body: ""},
			},
			DeleteSoft: &ir.DeleteIR{
				RPCName: "DeleteBookSoft", RequestName: "DeleteBookSoftRequest", ResponseName: "google.protobuf.Empty",
				HTTPAnnotation: &ir.HTTPAnnotation{Verb: "POST", Path: "/api/Svc/book/deleteSoft", Body: "*"},
			},
			Resources: []ir.ResourceIR{{
				Name: "meta", PascalName: "Meta", Type: "test.BookMeta", Version: ir.VersionIR{Kind: "NONE"},
				Get: &ir.GetIR{
					RPCName: "GetBookMeta", RequestName: "GetBookMetaRequest", ResponseName: "GetBookMetaResponse",
					HTTPAnnotation: &ir.HTTPAnnotation{Verb: "GET", Path: "/api/Svc/book/{key.id}/meta", Body: ""},
				},
				BatchGet: &ir.BatchGetIR{
					RPCName: "BatchGetBookMetas", RequestName: "BatchGetBookMetasRequest", ResponseName: "BatchGetBookMetasResponse",
					HTTPAnnotation: &ir.HTTPAnnotation{Verb: "POST", Path: "/api/Svc/book/meta/batchGet", Body: "*"},
				},
				List: &ir.ListIR{
					RPCName: "ListBookMetas", RequestName: "ListBookMetasRequest", ResponseName: "ListBookMetasResponse",
					HTTPAnnotation: &ir.HTTPAnnotation{Verb: "POST", Path: "/api/Svc/book/meta/list", Body: "*"},
				},
				Update: &ir.UpdateIR{
					RPCName: "UpdateBookMeta", RequestName: "UpdateBookMetaRequest", ResponseName: "google.protobuf.Empty",
					HTTPAnnotation: &ir.HTTPAnnotation{Verb: "PATCH", Path: "/api/Svc/book/{key.id}/meta", Body: "*"},
				},
			}},
		}},
	}
	svc := ir.ServiceIR{
		Name: "Svc", ProtoPackage: "test.svc", GoPackage: "svc",
		Entities: []ir.ServiceEntityIR{{Name: "book"}},
	}
	output, err := RenderServiceProto(irData, svc)
	if err != nil {
		t.Fatalf("RenderServiceProto failed: %v", err)
	}
	expected := []string{
		`post: "/api/Svc/book" body: "*"`,
		`delete: "/api/Svc/book/{key.id}"`,
		`post: "/api/Svc/book/deleteSoft" body: "*"`,
		`get: "/api/Svc/book/{key.id}/meta"`,
		`post: "/api/Svc/book/meta/batchGet" body: "*"`,
		`post: "/api/Svc/book/meta/list" body: "*"`,
		`patch: "/api/Svc/book/{key.id}/meta" body: "*"`,
	}
	for _, e := range expected {
		if !strings.Contains(output, e) {
			t.Errorf("output missing %q", e)
		}
	}
}

// Ensure unused import doesn't cause error (apigenyaml used for type reference).
var _ = apigenyaml.HTTPConfig{}
