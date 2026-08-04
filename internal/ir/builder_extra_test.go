package ir

import (
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"

	apigenyaml "github.com/hellowzsg/api-gen/internal/yaml"
)

// TestBuildMultiEntity: multiple entities in a single config, each with
// distinct methods and key types.
func TestBuildMultiEntity(t *testing.T) {
	cfg := &apigenyaml.Config{
		Syntax: "v1",
		Name:   "catalog",
		Entities: []apigenyaml.Entity{
			{
				Name:   "book",
				Key:    apigenyaml.KeyDef{Type: "BookId"},
				Create: &apigenyaml.CreateDef{},
				Delete: &struct{}{},
				List:   &apigenyaml.EntityListDef{Resources: []string{"meta"}},
				Resources: []apigenyaml.Resource{{
					Name:    "meta",
					Type:    "BookMeta",
					Version: apigenyaml.VersionDef{Kind: "STRONG", Type: "U64"},
					Reader:  &apigenyaml.ReaderDef{Batch: true},
					Writer:  &apigenyaml.WriterDef{Update: &apigenyaml.UpdateDef{Mask: true}},
				}},
			},
			{
				Name: "author",
				Key:  apigenyaml.KeyDef{Type: "AuthorId"},
				Resources: []apigenyaml.Resource{{
					Name:    "profile",
					Type:    "AuthorProfile",
					Version: apigenyaml.VersionDef{Kind: "NONE"},
					Reader:  &apigenyaml.ReaderDef{},
				}},
			},
		},
	}
	ir, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if len(ir.Entities) != 2 {
		t.Fatalf("Entities = %d, want 2", len(ir.Entities))
	}

	// Entity 0: book — should have Create, Delete, Get, BatchGet, List, Update
	e0 := ir.Entities[0]
	if e0.PascalName != "Book" {
		t.Errorf("Entity[0].PascalName = %q, want Book", e0.PascalName)
	}
	if e0.Create == nil {
		t.Error("Entity[0].Create is nil")
	}
	if e0.Delete == nil {
		t.Error("Entity[0].Delete is nil")
	}
	if e0.DeleteSoft != nil {
		t.Error("Entity[0].DeleteSoft should be nil")
	}
	if e0.Resources[0].Get == nil {
		t.Error("Entity[0].Get is nil")
	}
	if e0.Resources[0].BatchGet == nil {
		t.Error("Entity[0].BatchGet is nil")
	}
	if e0.List == nil {
		t.Error("Entity[0].List is nil")
	}
	if e0.Resources[0].Update == nil {
		t.Error("Entity[0].Update is nil")
	}

	// Entity 1: author — should have only Get (reader: {}, no create/delete)
	e1 := ir.Entities[1]
	if e1.PascalName != "Author" {
		t.Errorf("Entity[1].PascalName = %q, want Author", e1.PascalName)
	}
	if e1.Create != nil {
		t.Error("Entity[1].Create should be nil")
	}
	if e1.Delete != nil {
		t.Error("Entity[1].Delete should be nil")
	}
	if e1.Resources[0].Get == nil {
		t.Error("Entity[1].Get is nil")
	}
	if e1.Resources[0].BatchGet != nil {
		t.Error("Entity[1].BatchGet should be nil (batch not enabled)")
	}
	if e1.List != nil {
		t.Error("Entity[1].List should be nil (list not enabled)")
	}
	if e1.Resources[0].Update != nil {
		t.Error("Entity[1].Update should be nil (no writer)")
	}
}

// TestBuildMultiResource: single entity with multiple resources, verifying
// each resource gets independent methods.
func TestBuildMultiResource(t *testing.T) {
	cfg := &apigenyaml.Config{
		Syntax: "v1",
		Name:   "test",
		Entities: []apigenyaml.Entity{{
			Name: "book",
			Key:  apigenyaml.KeyDef{Type: "BookId"},
			List: &apigenyaml.EntityListDef{Resources: []string{"meta", "content", "review"}},
			Resources: []apigenyaml.Resource{
				{Name: "meta", Type: "BookMeta", Version: apigenyaml.VersionDef{Kind: "STRONG", Type: "U64"},
					Reader: &apigenyaml.ReaderDef{Batch: true}, Writer: &apigenyaml.WriterDef{Update: &apigenyaml.UpdateDef{Mask: true}}},
				{Name: "content", Type: "BookContent", Version: apigenyaml.VersionDef{Kind: "NONE"},
					Reader: &apigenyaml.ReaderDef{}, Writer: &apigenyaml.WriterDef{Update: &apigenyaml.UpdateDef{}}},
				{Name: "review", Type: "BookReview", Version: apigenyaml.VersionDef{Kind: "WEAK", Type: "STRING"},
					Reader: &apigenyaml.ReaderDef{}},
			},
		}},
	}
	ir, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	e := ir.Entities[0]
	if len(e.Resources) != 3 {
		t.Fatalf("Resources = %d, want 3", len(e.Resources))
	}

	// Resource 0: meta — STRONG, batch+update(mask)
	r0 := e.Resources[0]
	if r0.Get == nil || r0.BatchGet == nil || r0.Update == nil {
		t.Error("Resource[0] missing methods")
	}
	if !r0.Update.HasVersion {
		t.Error("Resource[0] Update should have version (STRONG)")
	}
	if !r0.Update.Mask {
		t.Error("Resource[0] Update should have mask")
	}

	// Resource 1: content — NONE, get+update(no mask)
	r1 := e.Resources[1]
	if r1.BatchGet != nil {
		t.Error("Resource[1] should not have BatchGet")
	}
	if r1.Update == nil {
		t.Error("Resource[1] Update is nil")
	}
	if r1.Update.HasVersion {
		t.Error("Resource[1] Update should not have version (NONE)")
	}
	if r1.Update.Mask {
		t.Error("Resource[1] Update should not have mask")
	}
	if r1.Update.ResponseName != "google.protobuf.Empty" {
		t.Errorf("Resource[1] Update ResponseName = %q, want google.protobuf.Empty", r1.Update.ResponseName)
	}

	// Resource 2: review — WEAK, get only (no batch, no update)
	// List is entity-level pointing to review.
	r2 := e.Resources[2]
	if r2.Get == nil {
		t.Error("Resource[2] Get is nil")
	}
	if r2.BatchGet != nil {
		t.Error("Resource[2] should not have BatchGet")
	}
	if r2.Update != nil {
		t.Error("Resource[2] should not have Update")
	}
	// Entity-level List should be present and aggregate all declared resources.
	if e.List == nil {
		t.Error("Entity List is nil (should target declared resources)")
	}
	if e.List.RPCName != "ListBooks" {
		t.Errorf("Entity List RPCName = %q, want ListBooks", e.List.RPCName)
	}
	if e.List.ItemName != "BookItem" {
		t.Errorf("Entity List ItemName = %q, want BookItem", e.List.ItemName)
	}
	if len(e.List.ItemFields) != 3 {
		t.Errorf("Entity List ItemFields = %d, want 3", len(e.List.ItemFields))
	}
}

// TestBuildServiceNarrowing: service-level resource narrowing produces
// correct method set in the ServiceIR.
func TestBuildServiceNarrowing(t *testing.T) {
	listTrue := true
	cfg := &apigenyaml.Config{
		Syntax: "v1",
		Name:   "test",
		Entities: []apigenyaml.Entity{{
			Name: "book",
			Key:  apigenyaml.KeyDef{Type: "BookId"},
			List: &apigenyaml.EntityListDef{Resources: []string{"meta"}},
			Resources: []apigenyaml.Resource{
				{Name: "meta", Type: "BookMeta", Version: apigenyaml.VersionDef{Kind: "NONE"},
					Reader: &apigenyaml.ReaderDef{Batch: true}, Writer: &apigenyaml.WriterDef{Update: &apigenyaml.UpdateDef{Mask: true}}},
				{Name: "content", Type: "BookContent", Version: apigenyaml.VersionDef{Kind: "NONE"},
					Reader: &apigenyaml.ReaderDef{}, Writer: &apigenyaml.WriterDef{Update: &apigenyaml.UpdateDef{}}},
			},
		}},
		Services: []apigenyaml.Service{
			{
				Name: "FullService",
				Entities: []apigenyaml.ServiceEntity{{Name: "book"}},
			},
			{
				Name: "NarrowService",
				Entities: []apigenyaml.ServiceEntity{{
					Name: "book",
					List: &listTrue,
					Resources: []apigenyaml.Resource{{
						Name:   "meta",
						Reader: &apigenyaml.ReaderDef{},
					}},
				}},
			},
		},
	}
	ir, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if len(ir.Services) != 2 {
		t.Fatalf("Services = %d, want 2", len(ir.Services))
	}

	// FullService — no narrowing, inherits everything
	fullSvc := ir.Services[0]
	if len(fullSvc.Entities[0].Resources) != 0 {
		t.Errorf("FullService should have no resource narrowing, got %d resources", len(fullSvc.Entities[0].Resources))
	}

	// NarrowService — narrowed to meta with entity-level list enabled
	narrowSvc := ir.Services[1]
	if len(narrowSvc.Entities[0].Resources) != 1 {
		t.Fatalf("NarrowService should have 1 narrowed resource, got %d", len(narrowSvc.Entities[0].Resources))
	}
	nr := narrowSvc.Entities[0].Resources[0]
	if nr.Name != "meta" {
		t.Errorf("Narrowed resource name = %q, want meta", nr.Name)
	}
	if nr.Reader == nil {
		t.Fatal("Narrowed reader is nil")
	}
	if nr.Reader.Batch == nil || *nr.Reader.Batch {
		t.Error("Narrowed reader batch should be false")
	}
	if nr.Writer != nil {
		t.Error("Narrowed writer should be nil (not declared in service)")
	}
	// Entity-level List narrowing should be enabled.
	if narrowSvc.Entities[0].List == nil || !*narrowSvc.Entities[0].List {
		t.Error("Narrowed entity List should be true")
	}
}

// TestBuildListTotalSize: total_size is always present (no longer configurable).
func TestBuildListTotalSize(t *testing.T) {
	t.Run("total_size always present", func(t *testing.T) {
		cfg := &apigenyaml.Config{
			Syntax: "v1", Name: "test",
			Entities: []apigenyaml.Entity{{
				Name: "book", Key: apigenyaml.KeyDef{Type: "BookId"},
				List: &apigenyaml.EntityListDef{Resources: []string{"meta"}},
				Resources: []apigenyaml.Resource{{
					Name: "meta", Type: "BookMeta", Version: apigenyaml.VersionDef{Kind: "NONE"},
				}},
			}},
		}
		ir, _ := Build(cfg)
		l := ir.Entities[0].List
		if l == nil {
			t.Fatal("List is nil")
		}
		if l.TotalSize.Name != "total_size" {
			t.Errorf("TotalSize.Name = %q, want total_size", l.TotalSize.Name)
		}
		if l.TotalSize.Number != 2 {
			t.Errorf("TotalSize.Number = %d, want 2", l.TotalSize.Number)
		}
	})
}

// TestBuildPascalCaseConversion: snake_case names are correctly converted.
func TestBuildPascalCaseConversion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"book", "Book"},
		{"book_meta", "BookMeta"},
		{"user_profile", "UserProfile"},
		{"a", "A"},
		{"abc_def_ghi", "AbcDefGhi"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toPascalCase(tt.input)
			if got != tt.want {
				t.Errorf("toPascalCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestBuildSnakeCaseConversion: PascalCase service names converted to snake_case.
func TestBuildSnakeCaseConversion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"LibraryService", "library_service"},
		{"AdminService", "admin_service"},
		{"Svc", "svc"},
		{"MyVeryLongServiceName", "my_very_long_service_name"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ToSnakeCase(tt.input)
			if got != tt.want {
				t.Errorf("toSnakeCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestBuildServiceProtoPackage: service proto package = name + snake(service).
func TestBuildServiceProtoPackage(t *testing.T) {
	cfg := &apigenyaml.Config{
		Syntax: "v1", Name: "demo.business.book",
		Entities: []apigenyaml.Entity{{
			Name: "book", Key: apigenyaml.KeyDef{Type: "BookId"},
			Resources: []apigenyaml.Resource{{Name: "meta", Type: "BookMeta", Version: apigenyaml.VersionDef{Kind: "NONE"}}},
		}},
		Services: []apigenyaml.Service{
			{Name: "LibraryService", Entities: []apigenyaml.ServiceEntity{{Name: "book"}}},
		},
	}
	ir, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	svc := ir.Services[0]
	if svc.ProtoPackage != "demo.business.book.library_service" {
		t.Errorf("ProtoPackage = %q, want demo.business.book.library_service", svc.ProtoPackage)
	}
	if svc.GoPackage != "library_service" {
		t.Errorf("GoPackage = %q, want library_service", svc.GoPackage)
	}
}

// TestBuildUpdateFieldNumbers: verify field numbering for different
// version+mask combinations.
func TestBuildUpdateFieldNumbers(t *testing.T) {
	tests := []struct {
		name       string
		kind       string
		mask       bool
		wantFields int
		wantLast   int
	}{
		{"NONE+mask", "NONE", true, 3, 3},      // resource, key, update_mask
		{"NONE+nomask", "NONE", false, 2, 2},    // resource, key
		{"STRONG+mask", "STRONG", true, 4, 4},   // resource, key, update_mask, version
		{"STRONG+nomask", "STRONG", false, 3, 3}, // resource, key, version
		{"WEAK+mask", "WEAK", true, 4, 4},       // resource, key, update_mask, version(wrapper)
		{"WEAK+nomask", "WEAK", false, 3, 3},    // resource, key, version(wrapper)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &apigenyaml.Config{
				Syntax: "v1", Name: "test",
				Entities: []apigenyaml.Entity{{
					Name: "book", Key: apigenyaml.KeyDef{Type: "BookId"},
					Resources: []apigenyaml.Resource{{
						Name:    "meta",
						Type:    "BookMeta",
						Version: apigenyaml.VersionDef{Kind: tt.kind, Type: "U64"},
						Writer:  &apigenyaml.WriterDef{Update: &apigenyaml.UpdateDef{Mask: tt.mask}},
					}},
				}},
			}
			ir, err := Build(cfg)
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}
			u := ir.Entities[0].Resources[0].Update
			if len(u.RequestFields) != tt.wantFields {
				t.Errorf("RequestFields count = %d, want %d", len(u.RequestFields), tt.wantFields)
			}
			if u.RequestFields[len(u.RequestFields)-1].Number != tt.wantLast {
				t.Errorf("Last field number = %d, want %d", u.RequestFields[len(u.RequestFields)-1].Number, tt.wantLast)
			}
		})
	}
}

// TestBuildCustomMethodNoHTTP: custom method without http block (HTTP disabled).
func TestBuildCustomMethodNoHTTP(t *testing.T) {
	cfg := &apigenyaml.Config{
		Syntax: "v1", Name: "test",
		Entities: []apigenyaml.Entity{{
			Name: "book", Key: apigenyaml.KeyDef{Type: "BookId"},
			Resources: []apigenyaml.Resource{{Name: "meta", Type: "BookMeta", Version: apigenyaml.VersionDef{Kind: "NONE"}}},
		}},
		Services: []apigenyaml.Service{{
			Name: "Svc",
			Entities: []apigenyaml.ServiceEntity{{Name: "book"}},
			CustomMethods: []apigenyaml.CustomMethod{{
				Name: "ArchiveBook", Request: "ArchiveBookRequest", Response: "ArchiveBookResponse",
			}},
		}},
	}
	ir, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	cm := ir.Services[0].CustomMethods[0]
	if cm.Name != "ArchiveBook" {
		t.Errorf("Name = %q", cm.Name)
	}
	if cm.Request != "ArchiveBookRequest" {
		t.Errorf("Request = %q", cm.Request)
	}
	if cm.Response != "ArchiveBookResponse" {
		t.Errorf("Response = %q", cm.Response)
	}
	if cm.HTTPAnnotation != nil {
		t.Error("HTTPAnnotation should be nil when HTTP disabled")
	}
}

// TestBuildCustomMethodStream: custom_methods[].stream is propagated to IR.
func TestBuildCustomMethodStream(t *testing.T) {
	tests := []struct {
		name   string
		stream string
	}{
		{"unary (empty)", ""},
		{"server", "server"},
		{"client", "client"},
		{"bidi", "bidi"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &apigenyaml.Config{
				Syntax: "v1", Name: "test",
				Entities: []apigenyaml.Entity{{
					Name: "book", Key: apigenyaml.KeyDef{Type: "BookId"},
					Resources: []apigenyaml.Resource{{Name: "meta", Type: "BookMeta", Version: apigenyaml.VersionDef{Kind: "NONE"}}},
				}},
				Services: []apigenyaml.Service{{
					Name: "Svc",
					Entities: []apigenyaml.ServiceEntity{{Name: "book"}},
					CustomMethods: []apigenyaml.CustomMethod{{
						Name: "CustomOp", Request: "CustomRequest", Response: "CustomResponse",
						Stream: tt.stream,
					}},
				}},
			}
			irData, err := Build(cfg)
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}
			cm := irData.Services[0].CustomMethods[0]
			if cm.Stream != tt.stream {
				t.Errorf("CustomMethodIR.Stream = %q, want %q", cm.Stream, tt.stream)
			}
		})
	}
}

// TestBuildCreateFieldNumbers: create request fields numbered from 1 per resource.
func TestBuildCreateFieldNumbers(t *testing.T) {
	cfg := &apigenyaml.Config{
		Syntax: "v1", Name: "test",
		Entities: []apigenyaml.Entity{{
			Name:   "book",
			Key:    apigenyaml.KeyDef{Type: "BookId"},
			Create: &apigenyaml.CreateDef{},
			Resources: []apigenyaml.Resource{
				{Name: "meta", Type: "BookMeta", Version: apigenyaml.VersionDef{Kind: "NONE"}},
				{Name: "content", Type: "BookContent", Version: apigenyaml.VersionDef{Kind: "NONE"}},
			},
		}},
	}
	ir, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	c := ir.Entities[0].Create
	if len(c.RequestFields) != 2 {
		t.Fatalf("RequestFields = %d, want 2", len(c.RequestFields))
	}
	if c.RequestFields[0].Name != "meta" || c.RequestFields[0].Number != 1 {
		t.Errorf("RequestFields[0] = {%s, %d}, want {meta, 1}", c.RequestFields[0].Name, c.RequestFields[0].Number)
	}
	if c.RequestFields[1].Name != "content" || c.RequestFields[1].Number != 2 {
		t.Errorf("RequestFields[1] = {%s, %d}, want {content, 2}", c.RequestFields[1].Name, c.RequestFields[1].Number)
	}
}

// TestBuildBatchGetResponseField: BatchGet response field is repeated with
// correct name and type.
func TestBuildBatchGetResponseField(t *testing.T) {
	cfg := &apigenyaml.Config{
		Syntax: "v1", Name: "test",
		Entities: []apigenyaml.Entity{{
			Name: "book", Key: apigenyaml.KeyDef{Type: "BookId"},
			Resources: []apigenyaml.Resource{{
				Name: "meta", Type: "BookMeta", Version: apigenyaml.VersionDef{Kind: "NONE"},
				Reader: &apigenyaml.ReaderDef{Batch: true},
			}},
		}},
	}
	ir, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	bg := ir.Entities[0].Resources[0].BatchGet
	if bg.KeysField.Name != "keys" {
		t.Errorf("KeysField.Name = %q, want keys", bg.KeysField.Name)
	}
	if !bg.KeysField.Repeated {
		t.Error("KeysField should be repeated")
	}
	if bg.ResourcesField.Name != "metas" {
		t.Errorf("ResourcesField.Name = %q, want metas", bg.ResourcesField.Name)
	}
	if !bg.ResourcesField.Repeated {
		t.Error("ResourcesField should be repeated")
	}
}

// TestBuildListRequestFields: List request has limit, offset,
// filter, order_by in correct positions.
func TestBuildListRequestFields(t *testing.T) {
	cfg := &apigenyaml.Config{
		Syntax: "v1", Name: "test",
		Entities: []apigenyaml.Entity{{
			Name: "book", Key: apigenyaml.KeyDef{Type: "BookId"},
		List: &apigenyaml.EntityListDef{Resources: []string{"meta"}},
		Resources: []apigenyaml.Resource{{
			Name: "meta", Type: "BookMeta", Version: apigenyaml.VersionDef{Kind: "NONE"},
		}},
		}},
	}
	ir, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	l := ir.Entities[0].List
	if l.Limit.Name != "limit" || l.Limit.Number != 1 {
		t.Errorf("Limit = {%s, %d}, want {limit, 1}", l.Limit.Name, l.Limit.Number)
	}
	if l.Offset.Name != "offset" || l.Offset.Number != 2 {
		t.Errorf("Offset = {%s, %d}, want {offset, 2}", l.Offset.Name, l.Offset.Number)
	}
	if l.Filter.Name != "filter" || l.Filter.Number != 3 {
		t.Errorf("Filter = {%s, %d}, want {filter, 3}", l.Filter.Name, l.Filter.Number)
	}
	if l.OrderBy.Name != "order_by" || l.OrderBy.Number != 4 {
		t.Errorf("OrderBy = {%s, %d}, want {order_by, 4}", l.OrderBy.Name, l.OrderBy.Number)
	}
	// Response TotalSize is always present at field 2.
	if l.TotalSize.Name != "total_size" || l.TotalSize.Number != 2 {
		t.Errorf("TotalSize = {%s, %d}, want {total_size, 2}", l.TotalSize.Name, l.TotalSize.Number)
	}
}

// TestBuildCreateClientKey: client mode → key=1 prefixed, resources 2..N+1.
func TestBuildCreateClientKey(t *testing.T) {
	cfg := &apigenyaml.Config{
		Syntax: "v1", Name: "test",
		Entities: []apigenyaml.Entity{{
			Name:   "message",
			Key:    apigenyaml.KeyDef{Type: "MessageId"},
			Create: &apigenyaml.CreateDef{Key: "client"},
			Resources: []apigenyaml.Resource{
				{Name: "meta", Type: "MessageMeta", Version: apigenyaml.VersionDef{Kind: "NONE"}},
				{Name: "content", Type: "MessageContent", Version: apigenyaml.VersionDef{Kind: "NONE"}},
			},
		}},
	}
	ir, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	c := ir.Entities[0].Create
	// key field should be prefixed at field 1.
	if c.KeyField == nil {
		t.Fatal("KeyField is nil, want non-nil for client mode")
	}
	if c.KeyField.Name != "key" || c.KeyField.Number != 1 || c.KeyField.Type != "test.MessageId" {
		t.Errorf("KeyField = {%s, %s, %d}, want {key, test.MessageId, 1}", c.KeyField.Name, c.KeyField.Type, c.KeyField.Number)
	}
	// Resources should be numbered 2..N+1.
	if len(c.RequestFields) != 2 {
		t.Fatalf("RequestFields = %d, want 2", len(c.RequestFields))
	}
	if c.RequestFields[0].Name != "meta" || c.RequestFields[0].Number != 2 {
		t.Errorf("RequestFields[0] = {%s, %d}, want {meta, 2}", c.RequestFields[0].Name, c.RequestFields[0].Number)
	}
	if c.RequestFields[1].Name != "content" || c.RequestFields[1].Number != 3 {
		t.Errorf("RequestFields[1] = {%s, %d}, want {content, 3}", c.RequestFields[1].Name, c.RequestFields[1].Number)
	}
}

// TestBuildCreateServerKey: server mode (default) → no key field, resources 1..N.
func TestBuildCreateServerKey(t *testing.T) {
	cfg := &apigenyaml.Config{
		Syntax: "v1", Name: "test",
		Entities: []apigenyaml.Entity{{
			Name:   "book",
			Key:    apigenyaml.KeyDef{Type: "BookId"},
			Create: &apigenyaml.CreateDef{},
			Resources: []apigenyaml.Resource{
				{Name: "meta", Type: "BookMeta", Version: apigenyaml.VersionDef{Kind: "NONE"}},
			},
		}},
	}
	ir, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	c := ir.Entities[0].Create
	if c.KeyField != nil {
		t.Errorf("KeyField = %+v, want nil for server mode", c.KeyField)
	}
	if len(c.RequestFields) != 1 || c.RequestFields[0].Number != 1 {
		t.Errorf("RequestFields = %+v, want [{meta, 1}]", c.RequestFields)
	}
}

// TestBuildCreateClientKeyHTTPAnnotation: client mode + HTTP enabled →
// Create HTTP annotation carries KeyLeaves (path has key segments).
func TestBuildCreateClientKeyHTTPAnnotation(t *testing.T) {
	keyDesc := buildTestKeyDesc(t) // BookId{string id=1;} → 1 leaf "id"
	cfg := &apigenyaml.Config{
		Syntax: "v1",
		Name:   "test",
		Settings: apigenyaml.Settings{
			HTTP: &apigenyaml.HTTPConfig{Enable: true, Prefix: "/api"},
		},
		Entities: []apigenyaml.Entity{{
			Name:   "book",
			Key:    apigenyaml.KeyDef{Type: "BookId"},
			Create: &apigenyaml.CreateDef{Key: "client"},
			Resources: []apigenyaml.Resource{
				{Name: "meta", Type: "BookMeta", Version: apigenyaml.VersionDef{Kind: "NONE"}},
			},
		}},
	}
	keyDescs := map[string]protoreflect.MessageDescriptor{
		"test.BookId": keyDesc,
	}
	irData, err := BuildWithOptions(cfg, BuildOptions{KeyDescriptors: keyDescs})
	if err != nil {
		t.Fatalf("BuildWithOptions failed: %v", err)
	}
	c := irData.Entities[0].Create
	if c.HTTPAnnotation == nil {
		t.Fatal("HTTPAnnotation is nil")
	}
	if len(c.HTTPAnnotation.KeyLeaves) != 1 {
		t.Fatalf("KeyLeaves = %d, want 1", len(c.HTTPAnnotation.KeyLeaves))
	}
	if c.HTTPAnnotation.KeyLeaves[0].DotPath != "id" {
		t.Errorf("KeyLeaves[0].DotPath = %q, want id", c.HTTPAnnotation.KeyLeaves[0].DotPath)
	}
	// ResolvePath should include {key.id} segment.
	if got := c.HTTPAnnotation.ResolvePath("/api", "Svc"); got != "/api/Svc/book/{key.id}" {
		t.Errorf("ResolvePath = %q, want /api/Svc/book/{key.id}", got)
	}
}

// TestBuildCreateServerKeyHTTPAnnotation: server mode + HTTP enabled →
// Create HTTP annotation has no KeyLeaves (path has no key segments).
func TestBuildCreateServerKeyHTTPAnnotation(t *testing.T) {
	keyDesc := buildTestKeyDesc(t)
	cfg := &apigenyaml.Config{
		Syntax: "v1",
		Name:   "test",
		Settings: apigenyaml.Settings{
			HTTP: &apigenyaml.HTTPConfig{Enable: true, Prefix: "/api"},
		},
		Entities: []apigenyaml.Entity{{
			Name:   "book",
			Key:    apigenyaml.KeyDef{Type: "BookId"},
			Create: &apigenyaml.CreateDef{Key: "server"},
			Resources: []apigenyaml.Resource{
				{Name: "meta", Type: "BookMeta", Version: apigenyaml.VersionDef{Kind: "NONE"}},
			},
		}},
	}
	keyDescs := map[string]protoreflect.MessageDescriptor{
		"test.BookId": keyDesc,
	}
	irData, err := BuildWithOptions(cfg, BuildOptions{KeyDescriptors: keyDescs})
	if err != nil {
		t.Fatalf("BuildWithOptions failed: %v", err)
	}
	c := irData.Entities[0].Create
	if c.HTTPAnnotation == nil {
		t.Fatal("HTTPAnnotation is nil")
	}
	if len(c.HTTPAnnotation.KeyLeaves) != 0 {
		t.Errorf("KeyLeaves = %d, want 0 for server mode", len(c.HTTPAnnotation.KeyLeaves))
	}
	if got := c.HTTPAnnotation.ResolvePath("/api", "Svc"); got != "/api/Svc/book" {
		t.Errorf("ResolvePath = %q, want /api/Svc/book", got)
	}
}
