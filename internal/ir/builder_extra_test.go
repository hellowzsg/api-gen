package ir

import (
	"testing"

	apigenyaml "github.com/acme/apigen/internal/yaml"
)

// TestBuildMultiEntity: multiple entities in a single config, each with
// distinct methods and key types.
func TestBuildMultiEntity(t *testing.T) {
	cfg := &apigenyaml.Config{
		Syntax: "v1",
		Name:   "catalog",
		Entities: []apigenyaml.Entity{
			{
				Name: "book",
				Key:  apigenyaml.KeyDef{Type: "BookId"},
				Create: &struct{}{},
				Delete: &struct{}{},
				Resources: []apigenyaml.Resource{{
					Name:    "meta",
					Type:    "BookMeta",
					Version: apigenyaml.VersionDef{Kind: "STRONG", Type: "U64"},
					Reader:  &apigenyaml.ReaderDef{Batch: true, List: true},
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
	if e0.Resources[0].List == nil {
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
	if e1.Resources[0].List != nil {
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
			Resources: []apigenyaml.Resource{
				{Name: "meta", Type: "BookMeta", Version: apigenyaml.VersionDef{Kind: "STRONG", Type: "U64"},
					Reader: &apigenyaml.ReaderDef{Batch: true, List: true}, Writer: &apigenyaml.WriterDef{Update: &apigenyaml.UpdateDef{Mask: true}}},
				{Name: "content", Type: "BookContent", Version: apigenyaml.VersionDef{Kind: "NONE"},
					Reader: &apigenyaml.ReaderDef{}, Writer: &apigenyaml.WriterDef{Update: &apigenyaml.UpdateDef{}}},
				{Name: "review", Type: "BookReview", Version: apigenyaml.VersionDef{Kind: "WEAK", Type: "STRING"},
					Reader: &apigenyaml.ReaderDef{List: true}},
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

	// Resource 0: meta — STRONG, batch+list+update(mask)
	r0 := e.Resources[0]
	if r0.Get == nil || r0.BatchGet == nil || r0.List == nil || r0.Update == nil {
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
	if r1.List != nil {
		t.Error("Resource[1] should not have List")
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

	// Resource 2: review — WEAK, get+list only (no batch, no update)
	r2 := e.Resources[2]
	if r2.Get == nil {
		t.Error("Resource[2] Get is nil")
	}
	if r2.BatchGet != nil {
		t.Error("Resource[2] should not have BatchGet")
	}
	if r2.List == nil {
		t.Error("Resource[2] List is nil")
	}
	if r2.Update != nil {
		t.Error("Resource[2] should not have Update")
	}
}

// TestBuildServiceNarrowing: service-level resource narrowing produces
// correct method set in the ServiceIR.
func TestBuildServiceNarrowing(t *testing.T) {
	cfg := &apigenyaml.Config{
		Syntax: "v1",
		Name:   "test",
		Entities: []apigenyaml.Entity{{
			Name: "book",
			Key:  apigenyaml.KeyDef{Type: "BookId"},
			Resources: []apigenyaml.Resource{
				{Name: "meta", Type: "BookMeta", Version: apigenyaml.VersionDef{Kind: "NONE"},
					Reader: &apigenyaml.ReaderDef{Batch: true, List: true}, Writer: &apigenyaml.WriterDef{Update: &apigenyaml.UpdateDef{Mask: true}}},
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
					Resources: []apigenyaml.Resource{{
						Name:   "meta",
						Reader: &apigenyaml.ReaderDef{List: true},
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

	// NarrowService — narrowed to meta with only list
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
	if nr.Reader.List == nil || !*nr.Reader.List {
		t.Error("Narrowed reader list should be true")
	}
	if nr.Writer != nil {
		t.Error("Narrowed writer should be nil (not declared in service)")
	}
}

// TestBuildListTotalSize: list_config.total_size controls TotalSize field.
func TestBuildListTotalSize(t *testing.T) {
	t.Run("total_size true", func(t *testing.T) {
		cfg := &apigenyaml.Config{
			Syntax: "v1", Name: "test",
			Entities: []apigenyaml.Entity{{
				Name: "book", Key: apigenyaml.KeyDef{Type: "BookId"},
				Resources: []apigenyaml.Resource{{
					Name: "meta", Type: "BookMeta", Version: apigenyaml.VersionDef{Kind: "NONE"},
					Reader: &apigenyaml.ReaderDef{List: true, ListConfig: &apigenyaml.ListConfig{TotalSize: true}},
				}},
			}},
		}
		ir, _ := Build(cfg)
		l := ir.Entities[0].Resources[0].List
		if l.TotalSize == nil {
			t.Error("TotalSize should be non-nil when total_size=true")
		}
	})
	t.Run("total_size false", func(t *testing.T) {
		cfg := &apigenyaml.Config{
			Syntax: "v1", Name: "test",
			Entities: []apigenyaml.Entity{{
				Name: "book", Key: apigenyaml.KeyDef{Type: "BookId"},
				Resources: []apigenyaml.Resource{{
					Name: "meta", Type: "BookMeta", Version: apigenyaml.VersionDef{Kind: "NONE"},
					Reader: &apigenyaml.ReaderDef{List: true, ListConfig: &apigenyaml.ListConfig{TotalSize: false}},
				}},
			}},
		}
		ir, _ := Build(cfg)
		l := ir.Entities[0].Resources[0].List
		if l.TotalSize != nil {
			t.Error("TotalSize should be nil when total_size=false")
		}
	})
	t.Run("list_config omitted defaults to true", func(t *testing.T) {
		cfg := &apigenyaml.Config{
			Syntax: "v1", Name: "test",
			Entities: []apigenyaml.Entity{{
				Name: "book", Key: apigenyaml.KeyDef{Type: "BookId"},
				Resources: []apigenyaml.Resource{{
					Name: "meta", Type: "BookMeta", Version: apigenyaml.VersionDef{Kind: "NONE"},
					Reader: &apigenyaml.ReaderDef{List: true},
				}},
			}},
		}
		ir, _ := Build(cfg)
		l := ir.Entities[0].Resources[0].List
		if l.TotalSize == nil {
			t.Error("TotalSize should be non-nil when list_config omitted (defaults to true)")
		}
	})
}

// TestBuildPascalCaseConversion: snake_case names are correctly converted.
func TestBuildPascalCaseConversion(t *testing.T) {
	tests := []struct {
		input  string
		want   string
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
		input  string
		want   string
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
		{"NONE+mask", "NONE", true, 3, 3},     // resource, key, update_mask
		{"NONE+nomask", "NONE", false, 2, 2},   // resource, key
		{"STRONG+mask", "STRONG", true, 4, 4},  // resource, key, update_mask, version
		{"STRONG+nomask", "STRONG", false, 3, 3}, // resource, key, version
		{"WEAK+mask", "WEAK", true, 4, 4},      // resource, key, update_mask, version(wrapper)
		{"WEAK+nomask", "WEAK", false, 3, 3},   // resource, key, version(wrapper)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &apigenyaml.Config{
				Syntax: "v1", Name: "test",
				Entities: []apigenyaml.Entity{{
					Name: "book", Key: apigenyaml.KeyDef{Type: "BookId"},
					Resources: []apigenyaml.Resource{{
						Name: "meta", Type: "BookMeta",
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

// TestBuildCreateFieldNumbers: create request fields numbered from 1 per resource.
func TestBuildCreateFieldNumbers(t *testing.T) {
	cfg := &apigenyaml.Config{
		Syntax: "v1", Name: "test",
		Entities: []apigenyaml.Entity{{
			Name:   "book",
			Key:    apigenyaml.KeyDef{Type: "BookId"},
			Create: &struct{}{},
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

// TestBuildListRequestFields: List request has page_size, page_token,
// filter, order_by in correct positions.
func TestBuildListRequestFields(t *testing.T) {
	cfg := &apigenyaml.Config{
		Syntax: "v1", Name: "test",
		Entities: []apigenyaml.Entity{{
			Name: "book", Key: apigenyaml.KeyDef{Type: "BookId"},
			Resources: []apigenyaml.Resource{{
				Name: "meta", Type: "BookMeta", Version: apigenyaml.VersionDef{Kind: "NONE"},
				Reader: &apigenyaml.ReaderDef{List: true},
			}},
		}},
	}
	ir, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	l := ir.Entities[0].Resources[0].List
	if l.PageSize.Name != "page_size" || l.PageSize.Number != 1 {
		t.Errorf("PageSize = {%s, %d}, want {page_size, 1}", l.PageSize.Name, l.PageSize.Number)
	}
	if l.PageToken.Name != "page_token" || l.PageToken.Number != 2 {
		t.Errorf("PageToken = {%s, %d}, want {page_token, 2}", l.PageToken.Name, l.PageToken.Number)
	}
	if l.Filter.Name != "filter" || l.Filter.Number != 3 {
		t.Errorf("Filter = {%s, %d}, want {filter, 3}", l.Filter.Name, l.Filter.Number)
	}
	if l.OrderBy.Name != "order_by" || l.OrderBy.Number != 4 {
		t.Errorf("OrderBy = {%s, %d}, want {order_by, 4}", l.OrderBy.Name, l.OrderBy.Number)
	}
	if l.NextPageToken.Name != "next_page_token" || l.NextPageToken.Number != 2 {
		t.Errorf("NextPageToken = {%s, %d}, want {next_page_token, 2}", l.NextPageToken.Name, l.NextPageToken.Number)
	}
}
