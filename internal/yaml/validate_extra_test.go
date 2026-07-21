package yaml

import (
	"strings"
	"testing"
)

// TestValidateDuplicateEntity: duplicate entity names should fail.
func TestValidateDuplicateEntity(t *testing.T) {
	cfg := &Config{
		Syntax: "v1",
		Name:   "foo",
		Entities: []Entity{
			{Name: "book", Key: KeyDef{Type: "BookId"}, Resources: []Resource{{Name: "meta", Type: "BookMeta", Version: VersionDef{Kind: "NONE"}}}},
			{Name: "book", Key: KeyDef{Type: "BookId"}, Resources: []Resource{{Name: "meta", Type: "BookMeta", Version: VersionDef{Kind: "NONE"}}}},
		},
	}
	err := cfg.ValidateReferences()
	if err == nil {
		t.Fatal("ValidateReferences should fail for duplicate entity name")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("error should mention 'duplicate', got: %v", err)
	}
}

// TestValidateTypeNameInvalid: various invalid type_ values.
func TestValidateTypeNameInvalid(t *testing.T) {
	tests := []struct {
		name    string
		typeVal string
	}{
		{"empty", ""},
		{"starts with dot", ".BookId"},
		{"starts with digit", "1BookId"},
		{"contains space", "Book Id"},
		{"contains hyphen", "Book-Id"},
		{"contains at sign", "Book@Id"},
		{"contains slash", "Book/Id"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Syntax: "v1",
				Name:   "foo",
				Entities: []Entity{{
					Name: "book", Key: KeyDef{Type: tt.typeVal},
					Resources: []Resource{{Name: "meta", Type: "BookMeta", Version: VersionDef{Kind: "NONE"}}},
				}},
			}
			err := cfg.ValidateReferences()
			if err == nil {
				t.Errorf("ValidateReferences should fail for type_=%q", tt.typeVal)
			}
		})
	}
}

// TestValidateResourceTypeInvalid: invalid resource type_ should fail.
func TestValidateResourceTypeInvalid(t *testing.T) {
	cfg := &Config{
		Syntax: "v1",
		Name:   "foo",
		Entities: []Entity{{
			Name: "book", Key: KeyDef{Type: "BookId"},
			Resources: []Resource{{Name: "meta", Type: "", Version: VersionDef{Kind: "NONE"}}},
		}},
	}
	err := cfg.ValidateReferences()
	if err == nil {
		t.Fatal("ValidateReferences should fail for empty resource type_")
	}
	if !strings.Contains(err.Error(), "type_") {
		t.Errorf("error should mention type_, got: %v", err)
	}
}

// TestValidateHTTP_BodyStyleInvalid: invalid body_style value should fail.
func TestValidateHTTP_BodyStyleInvalid(t *testing.T) {
	cfg := &Config{
		Syntax: "v1",
		Name:   "foo",
		ImportProtos: []Import{
			{Path: "proto/**/*.proto"},
			{Path: "third_party/google/api/annotations.proto"},
		},
		Settings: Settings{
			HTTP: &HTTPConfig{Enable: true, BodyStyle: "invalid"},
		},
		Entities: []Entity{
			{Name: "book", Key: KeyDef{Type: "BookId"}, Resources: []Resource{{Name: "meta", Type: "BookMeta", Version: VersionDef{Kind: "NONE"}}}},
		},
	}
	err := cfg.ValidateReferences()
	if err == nil {
		t.Fatal("ValidateReferences should fail for invalid body_style")
	}
	if !strings.Contains(err.Error(), "body_style") {
		t.Errorf("error should mention body_style, got: %v", err)
	}
}

// TestValidateHTTP_GoogleapisViaGlob: broad glob path satisfies googleapis check.
func TestValidateHTTP_GoogleapisViaGlob(t *testing.T) {
	cfg := &Config{
		Syntax: "v1",
		Name:   "foo",
		ImportProtos: []Import{
			{Path: "**/*.proto"}, // broad glob should satisfy googleapis check
		},
		Settings: Settings{
			HTTP: &HTTPConfig{Enable: true, Prefix: "/api"},
		},
		Entities: []Entity{
			{Name: "book", Key: KeyDef{Type: "BookId"}, Resources: []Resource{{Name: "meta", Type: "BookMeta", Version: VersionDef{Kind: "NONE"}}}},
		},
	}
	if err := cfg.ValidateReferences(); err != nil {
		t.Fatalf("ValidateReferences should pass with broad glob path: %v", err)
	}
}

// TestValidateHTTPOverridePath_EdgeCases: additional path variable edge cases.
func TestValidateHTTPOverridePath_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantError bool
	}{
		{"leading dot var", "/svc/{.key}/meta", true},
		{"multiple valid vars", "/svc/{key.org}/{key.id}/meta", false},
		{"var with underscore", "/svc/{book_id}/meta", false},
		{"var with numbers", "/svc/{key123}/meta", false},
		{"trailing slash", "/svc/book/meta/", false},
		{"root path", "/", false},
		{"colon syntax", "/svc/book/{book_id}:archive", false},
		{"multiple colons", "/svc/book/{book_id}:archive:force", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Syntax: "v1",
				Name:   "foo",
				ImportProtos: []Import{
					{Path: "third_party/google/api/annotations.proto"},
				},
				Settings: Settings{
					HTTP: &HTTPConfig{Enable: true},
				},
				Entities: []Entity{
					{
						Name: "book",
						Key:  KeyDef{Type: "BookId"},
						Resources: []Resource{
							{
								Name:    "meta",
								Type:    "BookMeta",
								Version: VersionDef{Kind: "NONE"},
								Reader: &ReaderDef{
									HTTP: &HTTPOverride{Verb: "get", Path: tt.path},
								},
							},
						},
					},
				},
			}
			err := cfg.ValidateReferences()
			if tt.wantError && err == nil {
				t.Errorf("should fail for path %q", tt.path)
			}
			if !tt.wantError && err != nil {
				t.Errorf("should pass for path %q, got: %v", tt.path, err)
			}
		})
	}
}

// TestValidatePluginsJS_Multiple: multiple es declarations are valid.
func TestValidatePluginsJS_Multiple(t *testing.T) {
	cfg := &Config{
		Syntax: "v1",
		Name:   "foo",
		Entities: []Entity{
			{Name: "book", Key: KeyDef{Type: "BookId"}, Resources: []Resource{{Name: "meta", Type: "BookMeta", Version: VersionDef{Kind: "NONE"}}}},
		},
		Settings: Settings{
			Plugins: PluginsConfig{JS: []string{"es", "es"}},
		},
	}
	if err := cfg.ValidateReferences(); err != nil {
		t.Errorf("ValidateReferences should pass for multiple es plugins, got: %v", err)
	}
}

// TestValidateHTTPOverrideWriterUpdate: writer.update.http path validation.
func TestValidateHTTPOverrideWriterUpdate(t *testing.T) {
	t.Run("valid writer update http", func(t *testing.T) {
		cfg := &Config{
			Syntax: "v1",
			Name:   "foo",
			ImportProtos: []Import{
				{Path: "third_party/google/api/annotations.proto"},
			},
			Settings: Settings{HTTP: &HTTPConfig{Enable: true}},
			Entities: []Entity{{
				Name: "book", Key: KeyDef{Type: "BookId"},
				Resources: []Resource{{
					Name: "meta", Type: "BookMeta", Version: VersionDef{Kind: "NONE"},
					Writer: &WriterDef{Update: &UpdateDef{
						Mask: true,
						HTTP: &HTTPOverride{Verb: "put", Path: "/svc/book/{key.id}/meta"},
					}},
				}},
			}},
		}
		if err := cfg.ValidateReferences(); err != nil {
			t.Errorf("should pass: %v", err)
		}
	})

	t.Run("invalid writer update http path", func(t *testing.T) {
		cfg := &Config{
			Syntax: "v1",
			Name:   "foo",
			ImportProtos: []Import{
				{Path: "third_party/google/api/annotations.proto"},
			},
			Settings: Settings{HTTP: &HTTPConfig{Enable: true}},
			Entities: []Entity{{
				Name: "book", Key: KeyDef{Type: "BookId"},
				Resources: []Resource{{
					Name: "meta", Type: "BookMeta", Version: VersionDef{Kind: "NONE"},
					Writer: &WriterDef{Update: &UpdateDef{
						Mask: true,
						HTTP: &HTTPOverride{Verb: "put", Path: "/svc/book/{}/meta"},
					}},
				}},
			}},
		}
		err := cfg.ValidateReferences()
		if err == nil {
			t.Fatal("should fail for empty path variable in writer.update.http")
		}
	})
}

// TestValidateHTTPOverrideCustomMethod: custom_methods[].http path validation.
func TestValidateHTTPOverrideCustomMethod(t *testing.T) {
	t.Run("valid custom method http", func(t *testing.T) {
		cfg := &Config{
			Syntax: "v1",
			Name:   "foo",
			ImportProtos: []Import{
				{Path: "third_party/google/api/annotations.proto"},
			},
			Settings: Settings{HTTP: &HTTPConfig{Enable: true}},
			Entities: []Entity{{
				Name: "book", Key: KeyDef{Type: "BookId"},
				Resources: []Resource{{Name: "meta", Type: "BookMeta", Version: VersionDef{Kind: "NONE"}}},
			}},
			Services: []Service{{
				Name: "Svc",
				Entities: []ServiceEntity{{Name: "book"}},
				CustomMethods: []CustomMethod{{
					Name: "Archive", Request: "ArchiveRequest", Response: "ArchiveResponse",
					HTTP: &HTTPOverride{Verb: "post", Path: "/svc/book/{book_id}:archive", Body: "*"},
				}},
			}},
		}
		if err := cfg.ValidateReferences(); err != nil {
			t.Errorf("should pass: %v", err)
		}
	})

	t.Run("invalid custom method http path", func(t *testing.T) {
		cfg := &Config{
			Syntax: "v1",
			Name:   "foo",
			ImportProtos: []Import{
				{Path: "third_party/google/api/annotations.proto"},
			},
			Settings: Settings{HTTP: &HTTPConfig{Enable: true}},
			Entities: []Entity{{
				Name: "book", Key: KeyDef{Type: "BookId"},
				Resources: []Resource{{Name: "meta", Type: "BookMeta", Version: VersionDef{Kind: "NONE"}}},
			}},
			Services: []Service{{
				Name: "Svc",
				Entities: []ServiceEntity{{Name: "book"}},
				CustomMethods: []CustomMethod{{
					Name: "Archive", Request: "ArchiveRequest", Response: "ArchiveResponse",
					HTTP: &HTTPOverride{Verb: "post", Path: "/svc/book/{book_id.}:archive", Body: "*"},
				}},
			}},
		}
		err := cfg.ValidateReferences()
		if err == nil {
			t.Fatal("should fail for dangling dot in custom method path")
		}
	})
}

// TestResolveTypeName: type name resolution for short names and FQMs.
func TestResolveTypeName(t *testing.T) {
	cfg := &Config{Name: "demo.business.book"}
	tests := []struct {
		input string
		want  string
	}{
		{"BookId", "demo.business.book.BookId"},
		{"demo.business.book.BookId", "demo.business.book.BookId"},
		{"common.Foo", "common.Foo"},
		{"a.b.c.Type", "a.b.c.Type"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := cfg.ResolveTypeName(tt.input)
			if got != tt.want {
				t.Errorf("ResolveTypeName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestValidateServiceNarrowing_ResourceNotInEntity: narrowing to a resource
// that exists on the entity but with different capabilities.
func TestValidateServiceNarrowing_ValidMetaNarrowing(t *testing.T) {
	cfg := &Config{
		Syntax: "v1",
		Name:   "foo",
		Entities: []Entity{{
			Name: "book", Key: KeyDef{Type: "BookId"},
			Resources: []Resource{
				{Name: "meta", Type: "BookMeta", Version: VersionDef{Kind: "NONE"},
					Reader: &ReaderDef{Batch: true, List: true}, Writer: &WriterDef{Update: &UpdateDef{Mask: true}}},
				{Name: "content", Type: "BookContent", Version: VersionDef{Kind: "NONE"},
					Reader: &ReaderDef{}, Writer: &WriterDef{Update: &UpdateDef{}}},
			},
		}},
		Services: []Service{{
			Name: "AdminSvc",
			Entities: []ServiceEntity{{
				Name: "book",
				Resources: []Resource{
					{Name: "meta", Reader: &ReaderDef{List: true}},
				},
			}},
		}},
	}
	if err := cfg.ValidateReferences(); err != nil {
		t.Errorf("should pass for valid narrowing: %v", err)
	}
}
