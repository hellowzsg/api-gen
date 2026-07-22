package render

import (
	"strings"
	"testing"

	"github.com/hellowzsg/api-gen/internal/ir"
)

// TestRenderProto_DeleteSoft: DeleteSoft RPC rendered with correct signature.
func TestRenderProto_DeleteSoft(t *testing.T) {
	irData := &ir.IR{PackageName: "test", Entities: []ir.EntityIR{{
		Name: "book", PascalName: "Book", KeyType: "test.BookId",
		DeleteSoft: &ir.DeleteIR{
			RPCName: "DeleteBookSoft", RequestName: "DeleteBookSoftRequest", ResponseName: "google.protobuf.Empty",
			KeyField: ir.FieldIR{Name: "key", Type: "test.BookId", Number: 1},
		},
	}}}
	svc := ir.ServiceIR{Name: "Svc", ProtoPackage: "test.svc", GoPackage: "svc", Entities: []ir.ServiceEntityIR{{Name: "book"}}}
	output, err := RenderServiceProto(irData, svc)
	if err != nil {
		t.Fatalf("RenderServiceProto failed: %v", err)
	}
	if !strings.Contains(output, "rpc DeleteBookSoft") {
		t.Error("missing DeleteBookSoft RPC")
	}
	if !strings.Contains(output, "DeleteBookSoftRequest") {
		t.Error("missing DeleteBookSoftRequest message")
	}
}

// TestRenderProto_BatchGet: BatchGet RPC with repeated keys/metas fields.
func TestRenderProto_BatchGet(t *testing.T) {
	irData := &ir.IR{PackageName: "test", Entities: []ir.EntityIR{{
		Name: "book", PascalName: "Book", KeyType: "test.BookId",
		Resources: []ir.ResourceIR{{
			Name: "meta", PascalName: "Meta", Type: "test.BookMeta", Version: ir.VersionIR{Kind: "NONE"},
			BatchGet: &ir.BatchGetIR{
				RPCName: "BatchGetBookMetas", RequestName: "BatchGetBookMetasRequest", ResponseName: "BatchGetBookMetasResponse",
				KeysField:      ir.FieldIR{Name: "keys", Type: "test.BookId", Number: 1, Repeated: true},
				ResourcesField: ir.FieldIR{Name: "metas", Type: "test.BookMeta", Number: 1, Repeated: true},
			},
		}},
	}}}
	svc := ir.ServiceIR{Name: "Svc", ProtoPackage: "test.svc", GoPackage: "svc", Entities: []ir.ServiceEntityIR{{Name: "book"}}}
	output, err := RenderServiceProto(irData, svc)
	if err != nil {
		t.Fatalf("RenderServiceProto failed: %v", err)
	}
	if !strings.Contains(output, "rpc BatchGetBookMetas") {
		t.Error("missing BatchGetBookMetas RPC")
	}
	if !strings.Contains(output, "repeated test.BookId keys") {
		t.Error("missing repeated keys field")
	}
	if !strings.Contains(output, "repeated test.BookMeta metas") {
		t.Error("missing repeated metas field")
	}
}

// TestRenderProto_List: List RPC with pagination fields.
func TestRenderProto_List(t *testing.T) {
	totalSize := ir.FieldIR{Name: "total_size", Type: "int32", Number: 3}
	irData := &ir.IR{PackageName: "test", Entities: []ir.EntityIR{{
		Name: "book", PascalName: "Book", KeyType: "test.BookId",
		Resources: []ir.ResourceIR{{
			Name: "meta", PascalName: "Meta", Type: "test.BookMeta", Version: ir.VersionIR{Kind: "NONE"},
			List: &ir.ListIR{
				RPCName: "ListBookMetas", RequestName: "ListBookMetasRequest", ResponseName: "ListBookMetasResponse",
				PageSize:       ir.FieldIR{Name: "page_size", Type: "int32", Number: 1},
				PageToken:      ir.FieldIR{Name: "page_token", Type: "string", Number: 2},
				Filter:         ir.FieldIR{Name: "filter", Type: "string", Number: 3},
				OrderBy:        ir.FieldIR{Name: "order_by", Type: "string", Number: 4},
				ResourcesField: ir.FieldIR{Name: "metas", Type: "test.BookMeta", Number: 1, Repeated: true},
				NextPageToken:  ir.FieldIR{Name: "next_page_token", Type: "string", Number: 2},
				TotalSize:      &totalSize,
			},
		}},
	}}}
	svc := ir.ServiceIR{Name: "Svc", ProtoPackage: "test.svc", GoPackage: "svc", Entities: []ir.ServiceEntityIR{{Name: "book"}}}
	output, err := RenderServiceProto(irData, svc)
	if err != nil {
		t.Fatalf("RenderServiceProto failed: %v", err)
	}
	if !strings.Contains(output, "rpc ListBookMetas") {
		t.Error("missing ListBookMetas RPC")
	}
	if !strings.Contains(output, "int32 page_size") {
		t.Error("missing page_size field")
	}
	if !strings.Contains(output, "string page_token") {
		t.Error("missing page_token field")
	}
	if !strings.Contains(output, "string filter") {
		t.Error("missing filter field")
	}
	if !strings.Contains(output, "string order_by") {
		t.Error("missing order_by field")
	}
	if !strings.Contains(output, "string next_page_token") {
		t.Error("missing next_page_token field")
	}
	if !strings.Contains(output, "int32 total_size") {
		t.Error("missing total_size field")
	}
}

// TestRenderProto_UpdateWithMask: Update with mask renders update_mask field.
func TestRenderProto_UpdateWithMask(t *testing.T) {
	irData := &ir.IR{PackageName: "test", Entities: []ir.EntityIR{{
		Name: "book", PascalName: "Book", KeyType: "test.BookId",
		Resources: []ir.ResourceIR{{
			Name: "meta", PascalName: "Meta", Type: "test.BookMeta", Version: ir.VersionIR{Kind: "NONE"},
			Update: &ir.UpdateIR{
				RPCName: "UpdateBookMeta", RequestName: "UpdateBookMetaRequest", ResponseName: "google.protobuf.Empty",
				Mask: true,
				RequestFields: []ir.FieldIR{
					{Name: "meta", Type: "test.BookMeta", Number: 1},
					{Name: "key", Type: "test.BookId", Number: 2},
					{Name: "update_mask", Type: "google.protobuf.FieldMask", Number: 3},
				},
			},
		}},
	}}}
	svc := ir.ServiceIR{Name: "Svc", ProtoPackage: "test.svc", GoPackage: "svc", Entities: []ir.ServiceEntityIR{{Name: "book"}}}
	output, err := RenderServiceProto(irData, svc)
	if err != nil {
		t.Fatalf("RenderServiceProto failed: %v", err)
	}
	if !strings.Contains(output, "rpc UpdateBookMeta") {
		t.Error("missing UpdateBookMeta RPC")
	}
	if !strings.Contains(output, "google.protobuf.FieldMask update_mask") {
		t.Error("missing update_mask field")
	}
}

// TestRenderProto_UpdateSTRONG: STRONG version Update renders version field
// and non-Empty response.
func TestRenderProto_UpdateSTRONG(t *testing.T) {
	irData := &ir.IR{PackageName: "test", Entities: []ir.EntityIR{{
		Name: "book", PascalName: "Book", KeyType: "test.BookId",
		Resources: []ir.ResourceIR{{
			Name: "meta", PascalName: "Meta", Type: "test.BookMeta", Version: ir.VersionIR{Kind: "STRONG", Type: "U64", FieldType: "uint64"},
			Update: &ir.UpdateIR{
				RPCName: "UpdateBookMeta", RequestName: "UpdateBookMetaRequest", ResponseName: "UpdateBookMetaResponse",
				Mask: true, HasVersion: true,
				VersionField: &ir.FieldIR{Name: "version", Type: "uint64", Number: 1},
				RequestFields: []ir.FieldIR{
					{Name: "meta", Type: "test.BookMeta", Number: 1},
					{Name: "key", Type: "test.BookId", Number: 2},
					{Name: "update_mask", Type: "google.protobuf.FieldMask", Number: 3},
					{Name: "version", Type: "uint64", Number: 4},
				},
			},
		}},
	}}}
	svc := ir.ServiceIR{Name: "Svc", ProtoPackage: "test.svc", GoPackage: "svc", Entities: []ir.ServiceEntityIR{{Name: "book"}}}
	output, err := RenderServiceProto(irData, svc)
	if err != nil {
		t.Fatalf("RenderServiceProto failed: %v", err)
	}
	if !strings.Contains(output, "uint64 version") {
		t.Error("missing version field in request")
	}
	if !strings.Contains(output, "UpdateBookMetaResponse") {
		t.Error("missing UpdateBookMetaResponse (STRONG should not use Empty)")
	}
}

// TestRenderProto_UpdateWEAK: WEAK version uses wrapper type.
func TestRenderProto_UpdateWEAK(t *testing.T) {
	irData := &ir.IR{PackageName: "test", Entities: []ir.EntityIR{{
		Name: "book", PascalName: "Book", KeyType: "test.BookId",
		Resources: []ir.ResourceIR{{
			Name: "meta", PascalName: "Meta", Type: "test.BookMeta",
			Version: ir.VersionIR{Kind: "WEAK", Type: "U64", FieldType: "google.protobuf.UInt64Value", IsWrapper: true},
			Update: &ir.UpdateIR{
				RPCName: "UpdateBookMeta", RequestName: "UpdateBookMetaRequest", ResponseName: "UpdateBookMetaResponse",
				Mask: true, HasVersion: true,
				VersionField: &ir.FieldIR{Name: "version", Type: "google.protobuf.UInt64Value", Number: 1},
				RequestFields: []ir.FieldIR{
					{Name: "meta", Type: "test.BookMeta", Number: 1},
					{Name: "key", Type: "test.BookId", Number: 2},
					{Name: "update_mask", Type: "google.protobuf.FieldMask", Number: 3},
					{Name: "version", Type: "google.protobuf.UInt64Value", Number: 4},
				},
			},
		}},
	}}}
	svc := ir.ServiceIR{Name: "Svc", ProtoPackage: "test.svc", GoPackage: "svc", Entities: []ir.ServiceEntityIR{{Name: "book"}}}
	output, err := RenderServiceProto(irData, svc)
	if err != nil {
		t.Fatalf("RenderServiceProto failed: %v", err)
	}
	if !strings.Contains(output, "google.protobuf.UInt64Value version") {
		t.Error("missing UInt64Value version field")
	}
	if !strings.Contains(output, "google/protobuf/wrappers.proto") {
		t.Error("missing wrappers.proto import for WEAK version")
	}
}

// TestRenderProto_MultiService: multiple services in one proto package.
func TestRenderProto_MultiService(t *testing.T) {
	irData := &ir.IR{PackageName: "test", Entities: []ir.EntityIR{{
		Name: "book", PascalName: "Book", KeyType: "test.BookId",
		Delete: &ir.DeleteIR{
			RPCName: "DeleteBook", RequestName: "DeleteBookRequest", ResponseName: "google.protobuf.Empty",
			KeyField: ir.FieldIR{Name: "key", Type: "test.BookId", Number: 1},
		},
	}}}

	svc1 := ir.ServiceIR{Name: "LibSvc", ProtoPackage: "test.lib_svc", GoPackage: "lib_svc", Entities: []ir.ServiceEntityIR{{Name: "book"}}}
	out1, err := RenderServiceProto(irData, svc1)
	if err != nil {
		t.Fatalf("RenderServiceProto svc1 failed: %v", err)
	}
	if !strings.Contains(out1, "service LibSvc") {
		t.Error("missing LibSvc service")
	}

	svc2 := ir.ServiceIR{Name: "AdminSvc", ProtoPackage: "test.admin_svc", GoPackage: "admin_svc", Entities: []ir.ServiceEntityIR{{Name: "book"}}}
	out2, err := RenderServiceProto(irData, svc2)
	if err != nil {
		t.Fatalf("RenderServiceProto svc2 failed: %v", err)
	}
	if !strings.Contains(out2, "service AdminSvc") {
		t.Error("missing AdminSvc service")
	}
	// Different proto packages
	if strings.Contains(out1, "test.admin_svc") {
		t.Error("svc1 should not contain admin_svc package")
	}
}

// TestRenderProto_CreateMultiResource: Create with multiple resources renders
// all resource fields.
func TestRenderProto_CreateMultiResource(t *testing.T) {
	irData := &ir.IR{PackageName: "test", Entities: []ir.EntityIR{{
		Name: "book", PascalName: "Book", KeyType: "test.BookId",
		Create: &ir.CreateIR{
			RPCName: "CreateBook", RequestName: "CreateBookRequest", ResponseName: "CreateBookResponse",
			RequestFields: []ir.FieldIR{
				{Name: "meta", Type: "test.BookMeta", Number: 1},
				{Name: "content", Type: "test.BookContent", Number: 2},
			},
			ResponseKeyField: ir.FieldIR{Name: "key", Type: "test.BookId", Number: 1},
		},
	}}}
	svc := ir.ServiceIR{Name: "Svc", ProtoPackage: "test.svc", GoPackage: "svc", Entities: []ir.ServiceEntityIR{{Name: "book"}}}
	output, err := RenderServiceProto(irData, svc)
	if err != nil {
		t.Fatalf("RenderServiceProto failed: %v", err)
	}
	if !strings.Contains(output, "BookMeta meta") {
		t.Error("missing meta field in Create request")
	}
	if !strings.Contains(output, "BookContent content") {
		t.Error("missing content field in Create request")
	}
	if !strings.Contains(output, "BookId key") {
		t.Error("missing key field in Create response")
	}
}

// TestRenderProto_GoPackage: go_package option is rendered.
func TestRenderProto_GoPackage(t *testing.T) {
	irData := &ir.IR{PackageName: "test", Entities: []ir.EntityIR{{
		Name: "book", PascalName: "Book", KeyType: "test.BookId",
		Delete: &ir.DeleteIR{
			RPCName: "DeleteBook", RequestName: "DeleteBookRequest", ResponseName: "google.protobuf.Empty",
			KeyField: ir.FieldIR{Name: "key", Type: "test.BookId", Number: 1},
		},
	}}}
	svc := ir.ServiceIR{
		Name: "Svc", ProtoPackage: "test.svc", GoPackage: "svc",
		GoRepo:   "github.com/acme/test",
		OutGoDir: "generated/go",
		Entities: []ir.ServiceEntityIR{{Name: "book"}},
	}
	output, err := RenderServiceProto(irData, svc)
	if err != nil {
		t.Fatalf("RenderServiceProto failed: %v", err)
	}
	if !strings.Contains(output, `go_package = "github.com/acme/test/generated/go/svc;svc"`) {
		t.Errorf("missing or incorrect go_package option; output:\n%s", output)
	}
}

// TestRenderProto_EmptyIR: empty entities should still produce valid proto.
func TestRenderProto_EmptyIR(t *testing.T) {
	irData := &ir.IR{PackageName: "test", Entities: []ir.EntityIR{}}
	svc := ir.ServiceIR{Name: "EmptySvc", ProtoPackage: "test.empty_svc", GoPackage: "empty_svc"}
	output, err := RenderServiceProto(irData, svc)
	if err != nil {
		t.Fatalf("RenderServiceProto failed: %v", err)
	}
	if !strings.Contains(output, `syntax = "proto3"`) {
		t.Error("missing syntax declaration")
	}
	if !strings.Contains(output, "package test.empty_svc") {
		t.Error("missing package")
	}
	if !strings.Contains(output, "service EmptySvc") {
		t.Error("missing service")
	}
}

// TestRenderProto_DeleteSoftHTTP: DeleteSoft with HTTP annotation renders
// the POST /deleteSoft route.
func TestRenderProto_DeleteSoftHTTP(t *testing.T) {
	irData := &ir.IR{PackageName: "test", HTTPEnabled: true, HTTPPrefix: "/api", Entities: []ir.EntityIR{{
		Name: "book", PascalName: "Book", KeyType: "test.BookId",
		KeyLeaves: []ir.KeyLeaf{{DotPath: "id", FieldType: "string"}},
		Delete: &ir.DeleteIR{
			RPCName: "DeleteBook", RequestName: "DeleteBookRequest", ResponseName: "google.protobuf.Empty",
			KeyField:       ir.FieldIR{Name: "key", Type: "test.BookId", Number: 1},
			HTTPAnnotation: &ir.HTTPAnnotation{Verb: "DELETE", Entity: "book", KeyLeaves: []ir.KeyLeaf{{DotPath: "id"}}},
		},
		DeleteSoft: &ir.DeleteIR{
			RPCName: "DeleteBookSoft", RequestName: "DeleteBookSoftRequest", ResponseName: "google.protobuf.Empty",
			KeyField:       ir.FieldIR{Name: "key", Type: "test.BookId", Number: 1},
			HTTPAnnotation: &ir.HTTPAnnotation{Verb: "POST", Entity: "book", Suffix: "deleteSoft", Body: "*"},
		},
	}}}
	svc := ir.ServiceIR{Name: "Svc", ProtoPackage: "test.svc", GoPackage: "svc", Entities: []ir.ServiceEntityIR{{Name: "book"}}}
	output, err := RenderServiceProto(irData, svc)
	if err != nil {
		t.Fatalf("RenderServiceProto failed: %v", err)
	}
	if !strings.Contains(output, `delete: "/api/Svc/book/{key.id}"`) {
		t.Error("missing DELETE annotation")
	}
	if !strings.Contains(output, `post: "/api/Svc/book/deleteSoft" body: "*"`) {
		t.Error("missing DeleteSoft POST annotation")
	}
}

// TestRenderHTTPAnnotation_AllVerbs: verify PUT annotation format.
func TestRenderHTTPAnnotation_AllVerbs(t *testing.T) {
	ann := &ir.HTTPAnnotation{Verb: "PUT", Entity: "ent", KeyLeaves: []ir.KeyLeaf{{DotPath: "id"}}, Body: "meta"}
	got := RenderHTTPAnnotation(ann, "/api", "svc")
	want := `option (google.api.http) = { put: "/api/svc/ent/{key.id}" body: "meta" };`
	if got != want {
		t.Errorf("RenderHTTPAnnotation(PUT) = %q, want %q", got, want)
}
}

// TestRenderHTTPAnnotation_BodyResource: body field name (not "*") renders correctly.
func TestRenderHTTPAnnotation_BodyResource(t *testing.T) {
	ann := &ir.HTTPAnnotation{Verb: "PATCH", Entity: "ent", KeyLeaves: []ir.KeyLeaf{{DotPath: "id"}}, Body: "meta"}
	got := RenderHTTPAnnotation(ann, "/api", "svc")
	want := `option (google.api.http) = { patch: "/api/svc/ent/{key.id}" body: "meta" };`
	if got != want {
		t.Errorf("RenderHTTPAnnotation(body=meta) = %q, want %q", got, want)
	}
}
