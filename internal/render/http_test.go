package render

import (
	"strings"
	"testing"

	"github.com/hellowzsg/api-gen/internal/ir"
	apigenyaml "github.com/hellowzsg/api-gen/internal/yaml"
)

// TestRenderHTTPAnnotation_Format tests the google.api.http option format.
func TestRenderHTTPAnnotation_Format(t *testing.T) {
	tests := []struct {
		name   string
		ann    *ir.HTTPAnnotation
		prefix string
		svc    string
		want   string
	}{
		{
			name:   "POST with body star",
			ann:    &ir.HTTPAnnotation{Verb: "POST", Entity: "ent", Body: "*"},
			prefix: "/api",
			svc:    "svc",
			want:   `option (google.api.http) = { post: "/api/svc/ent" body: "*" };`,
		},
		{
			name:   "GET no body",
			ann:    &ir.HTTPAnnotation{Verb: "GET", Entity: "ent", KeyLeaves: []ir.KeyLeaf{{DotPath: "id"}}, Resource: "meta"},
			prefix: "/api",
			svc:    "svc",
			want:   `option (google.api.http) = { get: "/api/svc/ent/{key.id}/meta" };`,
		},
		{
			name:   "DELETE no body",
			ann:    &ir.HTTPAnnotation{Verb: "DELETE", Entity: "ent", KeyLeaves: []ir.KeyLeaf{{DotPath: "id"}}},
			prefix: "/api",
			svc:    "svc",
			want:   `option (google.api.http) = { delete: "/api/svc/ent/{key.id}" };`,
		},
		{
			name:   "PATCH with body star",
			ann:    &ir.HTTPAnnotation{Verb: "PATCH", Entity: "ent", KeyLeaves: []ir.KeyLeaf{{DotPath: "id"}}, Resource: "meta", Body: "*"},
			prefix: "/api",
			svc:    "svc",
			want:   `option (google.api.http) = { patch: "/api/svc/ent/{key.id}/meta" body: "*" };`,
		},
		{
			name: "verbatim override (custom method)",
			ann:  &ir.HTTPAnnotation{Verb: "GET", IsOverride: true, OverridePath: "/custom/{key.id}/items"},
			prefix: "/api",
			svc:    "svc",
			want:   `option (google.api.http) = { get: "/custom/{key.id}/items" };`,
		},
		{
			name: "template override (entity-level reader/writer http)",
			ann: &ir.HTTPAnnotation{
				Verb:                "GET",
				IsOverride:          true,
				OverridePath:        "/library/LibraryService/book/meta/list",
				OverrideTemplateSvc: "LibraryService",
			},
			prefix: "/library",
			svc:    "AdminService",
			want:   `option (google.api.http) = { get: "/library/AdminService/book/meta/list" };`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderHTTPAnnotation(tt.ann, tt.prefix, tt.svc)
			if got != tt.want {
				t.Errorf("RenderHTTPAnnotation = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRenderHTTPAnnotation_NilReturnsEmpty: nil annotation → empty string.
func TestRenderHTTPAnnotation_NilReturnsEmpty(t *testing.T) {
	if got := RenderHTTPAnnotation(nil, "/api", "svc"); got != "" {
		t.Errorf("RenderHTTPAnnotation(nil) = %q, want empty", got)
	}
}

// bookEntityIR builds the shared entity IR fixture with structured HTTP
// annotations (service-agnostic).
func bookEntityIR() ir.EntityIR {
	return ir.EntityIR{
		Name:       "book",
		PascalName: "Book",
		KeyType:    "test.BookId",
		KeyLeaves:  []ir.KeyLeaf{{DotPath: "id", FieldType: "string"}},
		Create: &ir.CreateIR{
			RPCName:          "CreateBook",
			RequestName:      "CreateBookRequest",
			ResponseName:     "CreateBookResponse",
			ResponseKeyField: ir.FieldIR{Name: "key", Type: "test.BookId", Number: 1},
			HTTPAnnotation:   &ir.HTTPAnnotation{Verb: "POST", Entity: "book", Body: "*"},
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
					HTTPAnnotation: &ir.HTTPAnnotation{
						Verb:      "GET",
						Entity:    "book",
						KeyLeaves: []ir.KeyLeaf{{DotPath: "id"}},
						Resource:  "meta",
					},
				},
			},
		},
	}
}

// TestRenderServiceProto_HTTPAnnotation: HTTP enabled → proto contains
// google.api.http option on RPCs and annotations import.
func TestRenderServiceProto_HTTPAnnotation(t *testing.T) {
	irData := &ir.IR{
		PackageName: "test",
		HTTPEnabled: true,
		HTTPPrefix:  "/library",
		Entities:    []ir.EntityIR{bookEntityIR()},
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

// TestRenderServiceProto_TwoServicesIndependentPaths: the SAME entity IR
// rendered into two services must produce service-isolated paths — with no
// string rewriting of a pre-baked path.
func TestRenderServiceProto_TwoServicesIndependentPaths(t *testing.T) {
	irData := &ir.IR{
		PackageName: "test",
		HTTPEnabled: true,
		HTTPPrefix:  "/library",
		Entities:    []ir.EntityIR{bookEntityIR()},
	}
	render := func(name string) string {
		out, err := RenderServiceProto(irData, ir.ServiceIR{
			Name:         name,
			ProtoPackage: "test." + strings.ToLower(name),
			GoPackage:    strings.ToLower(name),
			Entities:     []ir.ServiceEntityIR{{Name: "book"}},
		})
		if err != nil {
			t.Fatalf("RenderServiceProto(%s) failed: %v", name, err)
		}
		return out
	}
	lib := render("LibraryService")
	admin := render("AdminService")
	if !strings.Contains(lib, `"/library/LibraryService/book/{key.id}/meta"`) {
		t.Error("LibraryService output missing its own service segment")
	}
	if strings.Contains(lib, "AdminService") {
		t.Error("LibraryService output leaked AdminService segment")
	}
	if !strings.Contains(admin, `"/library/AdminService/book/{key.id}/meta"`) {
		t.Error("AdminService output missing its own service segment")
	}
	if strings.Contains(admin, "LibraryService") {
		t.Error("AdminService output leaked LibraryService segment")
	}
}

// TestRenderServiceProto_OverridePathVerbatim: user-declared override paths
// are emitted as-is for custom methods (verbatim) — OverrideTemplateSvc is
// empty so ResolvePath does not rewrite any segment.
func TestRenderServiceProto_OverridePathVerbatim(t *testing.T) {
	entity := bookEntityIR()
	entity.Resources[0].List = &ir.ListIR{
		RPCName:        "ListBookMetas",
		RequestName:    "ListBookMetasRequest",
		ResponseName:   "ListBookMetasResponse",
		PageSize:       ir.FieldIR{Name: "page_size", Type: "int32", Number: 1},
		PageToken:      ir.FieldIR{Name: "page_token", Type: "string", Number: 2},
		Filter:         ir.FieldIR{Name: "filter", Type: "string", Number: 3},
		OrderBy:        ir.FieldIR{Name: "order_by", Type: "string", Number: 4},
		ResourcesField: ir.FieldIR{Name: "metas", Type: "test.BookMeta", Number: 1, Repeated: true},
		NextPageToken:  ir.FieldIR{Name: "next_page_token", Type: "string", Number: 2},
		HTTPAnnotation: &ir.HTTPAnnotation{
			Verb:         "GET",
			IsOverride:   true,
			OverridePath: "/custom/{key.id}/items",
		},
	}
	irData := &ir.IR{
		PackageName: "test",
		HTTPEnabled: true,
		HTTPPrefix:  "/library",
		Entities:    []ir.EntityIR{entity},
	}
	out, err := RenderServiceProto(irData, ir.ServiceIR{
		Name:         "AdminService",
		ProtoPackage: "test.admin_service",
		GoPackage:    "admin_service",
		Entities:     []ir.ServiceEntityIR{{Name: "book"}},
	})
	if err != nil {
		t.Fatalf("RenderServiceProto failed: %v", err)
	}
	if !strings.Contains(out, `get: "/custom/{key.id}/items"`) {
		t.Errorf("verbatim override path should appear as-is, got:\n%s", out)
	}
}

// TestRenderServiceProto_OverridePathTemplate: entity-level reader/writer
// http overrides are templates — the segment equal to OverrideTemplateSvc
// is replaced by the rendering service name. Same entity rendered into two
// services yields two isolated routes (no collision).
func TestRenderServiceProto_OverridePathTemplate(t *testing.T) {
	entity := bookEntityIR()
	entity.Resources[0].List = &ir.ListIR{
		RPCName:        "ListBookMetas",
		RequestName:    "ListBookMetasRequest",
		ResponseName:   "ListBookMetasResponse",
		PageSize:       ir.FieldIR{Name: "page_size", Type: "int32", Number: 1},
		PageToken:      ir.FieldIR{Name: "page_token", Type: "string", Number: 2},
		Filter:         ir.FieldIR{Name: "filter", Type: "string", Number: 3},
		OrderBy:        ir.FieldIR{Name: "order_by", Type: "string", Number: 4},
		ResourcesField: ir.FieldIR{Name: "metas", Type: "test.BookMeta", Number: 1, Repeated: true},
		NextPageToken:  ir.FieldIR{Name: "next_page_token", Type: "string", Number: 2},
		HTTPAnnotation: &ir.HTTPAnnotation{
			Verb:                "GET",
			IsOverride:          true,
			OverridePath:        "/library/LibraryService/book/meta/list",
			OverrideTemplateSvc: "LibraryService",
		},
	}
	irData := &ir.IR{
		PackageName: "test",
		HTTPEnabled: true,
		HTTPPrefix:  "/library",
		Entities:    []ir.EntityIR{entity},
	}
	render := func(svcName string) string {
		out, err := RenderServiceProto(irData, ir.ServiceIR{
			Name:         svcName,
			ProtoPackage: "test." + strings.ToLower(svcName),
			GoPackage:    strings.ToLower(svcName),
			Entities:     []ir.ServiceEntityIR{{Name: "book"}},
		})
		if err != nil {
			t.Fatalf("RenderServiceProto(%s) failed: %v", svcName, err)
		}
		return out
	}
	lib := render("LibraryService")
	if !strings.Contains(lib, `get: "/library/LibraryService/book/meta/list"`) {
		t.Errorf("LibraryService should keep template as-is (segment matches), got:\n%s", lib)
	}
	admin := render("AdminService")
	if !strings.Contains(admin, `get: "/library/AdminService/book/meta/list"`) {
		t.Errorf("AdminService should have service segment rewritten, got:\n%s", admin)
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
				HTTPAnnotation: &ir.HTTPAnnotation{Verb: "POST", Entity: "book", Body: "*"},
			},
			Delete: &ir.DeleteIR{
				RPCName: "DeleteBook", RequestName: "DeleteBookRequest", ResponseName: "google.protobuf.Empty",
				HTTPAnnotation: &ir.HTTPAnnotation{Verb: "DELETE", Entity: "book", KeyLeaves: []ir.KeyLeaf{{DotPath: "id"}}},
			},
			DeleteSoft: &ir.DeleteIR{
				RPCName: "DeleteBookSoft", RequestName: "DeleteBookSoftRequest", ResponseName: "google.protobuf.Empty",
				HTTPAnnotation: &ir.HTTPAnnotation{Verb: "POST", Entity: "book", Suffix: "deleteSoft", Body: "*"},
			},
			Resources: []ir.ResourceIR{{
				Name: "meta", PascalName: "Meta", Type: "test.BookMeta", Version: ir.VersionIR{Kind: "NONE"},
				Get: &ir.GetIR{
					RPCName: "GetBookMeta", RequestName: "GetBookMetaRequest", ResponseName: "GetBookMetaResponse",
					HTTPAnnotation: &ir.HTTPAnnotation{Verb: "GET", Entity: "book", KeyLeaves: []ir.KeyLeaf{{DotPath: "id"}}, Resource: "meta"},
				},
				BatchGet: &ir.BatchGetIR{
					RPCName: "BatchGetBookMetas", RequestName: "BatchGetBookMetasRequest", ResponseName: "BatchGetBookMetasResponse",
					HTTPAnnotation: &ir.HTTPAnnotation{Verb: "POST", Entity: "book", Resource: "meta", Suffix: "batchGet", Body: "*"},
				},
				List: &ir.ListIR{
					RPCName: "ListBookMetas", RequestName: "ListBookMetasRequest", ResponseName: "ListBookMetasResponse",
					HTTPAnnotation: &ir.HTTPAnnotation{Verb: "POST", Entity: "book", Resource: "meta", Suffix: "list", Body: "*"},
				},
				Update: &ir.UpdateIR{
					RPCName: "UpdateBookMeta", RequestName: "UpdateBookMetaRequest", ResponseName: "google.protobuf.Empty",
					HTTPAnnotation: &ir.HTTPAnnotation{Verb: "PATCH", Entity: "book", KeyLeaves: []ir.KeyLeaf{{DotPath: "id"}}, Resource: "meta", Body: "*"},
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
