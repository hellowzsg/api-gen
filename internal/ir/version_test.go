package ir

import (
	"testing"

	apigenyaml "github.com/hellowzsg/api-gen/internal/yaml"
)

func TestVersionSTRONG(t *testing.T) {
	tests := []struct {
		typeName  string
		wantField string
	}{
		{"U64", "uint64"},
		{"U32", "uint32"},
		{"STRING", "string"},
	}
	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			cfg := &apigenyaml.Config{
				Syntax: "v1", Name: "test",
				Entities: []apigenyaml.Entity{{
					Name: "book", Key: apigenyaml.KeyDef{Type: "BookId"},
					Resources: []apigenyaml.Resource{{
						Name:    "meta", Type: "BookMeta",
						Version: apigenyaml.VersionDef{Kind: "STRONG", Type: tt.typeName},
						Writer:  &apigenyaml.WriterDef{Update: &apigenyaml.UpdateDef{Mask: true}},
					}},
				}},
			}
			ir, err := Build(cfg)
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}
			v := ir.Entities[0].Resources[0].Version
			if v.FieldType != tt.wantField {
				t.Errorf("STRONG FieldType = %q, want %q", v.FieldType, tt.wantField)
			}
			if v.IsWrapper {
				t.Error("STRONG should not be wrapper")
			}
		})
	}
}

func TestVersionWEAK(t *testing.T) {
	tests := []struct {
		typeName  string
		wantField string
	}{
		{"U64", "google.protobuf.UInt64Value"},
		{"U32", "google.protobuf.UInt32Value"},
		{"STRING", "google.protobuf.StringValue"},
	}
	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			cfg := &apigenyaml.Config{
				Syntax: "v1", Name: "test",
				Entities: []apigenyaml.Entity{{
					Name: "book", Key: apigenyaml.KeyDef{Type: "BookId"},
					Resources: []apigenyaml.Resource{{
						Name:    "meta", Type: "BookMeta",
						Version: apigenyaml.VersionDef{Kind: "WEAK", Type: tt.typeName},
						Writer:  &apigenyaml.WriterDef{Update: &apigenyaml.UpdateDef{Mask: true}},
					}},
				}},
			}
			ir, err := Build(cfg)
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}
			v := ir.Entities[0].Resources[0].Version
			if v.FieldType != tt.wantField {
				t.Errorf("WEAK FieldType = %q, want %q", v.FieldType, tt.wantField)
			}
			if !v.IsWrapper {
				t.Error("WEAK should be wrapper")
			}
		})
	}
}

func TestVersionNONE(t *testing.T) {
	cfg := &apigenyaml.Config{
		Syntax: "v1", Name: "test",
		Entities: []apigenyaml.Entity{{
			Name: "book", Key: apigenyaml.KeyDef{Type: "BookId"},
			Resources: []apigenyaml.Resource{{
				Name:    "meta", Type: "BookMeta",
				Version: apigenyaml.VersionDef{Kind: "NONE"},
				Writer:  &apigenyaml.WriterDef{Update: &apigenyaml.UpdateDef{Mask: true}},
			}},
		}},
	}
	ir, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	r := ir.Entities[0].Resources[0]
	if r.Update.HasVersion {
		t.Error("NONE should not have version in Update")
	}
	if len(r.Update.RequestFields) != 3 {
		t.Errorf("NONE Update RequestFields = %d, want 3", len(r.Update.RequestFields))
	}
	if r.Update.ResponseName != "google.protobuf.Empty" {
		t.Errorf("NONE Update ResponseName = %q, want google.protobuf.Empty", r.Update.ResponseName)
	}
}

func TestVersionSTRONGUpdateResponse(t *testing.T) {
	cfg := &apigenyaml.Config{
		Syntax: "v1", Name: "test",
		Entities: []apigenyaml.Entity{{
			Name: "book", Key: apigenyaml.KeyDef{Type: "BookId"},
			Resources: []apigenyaml.Resource{{
				Name:    "meta", Type: "BookMeta",
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
	if r.Update.ResponseName == "google.protobuf.Empty" {
		t.Error("STRONG Update Response should not be Empty")
	}
	if r.Update.VersionField == nil {
		t.Error("STRONG Update should have VersionField in response")
	}
}

func TestVersionGetResponse(t *testing.T) {
	cfg := &apigenyaml.Config{
		Syntax: "v1", Name: "test",
		Entities: []apigenyaml.Entity{{
			Name: "book", Key: apigenyaml.KeyDef{Type: "BookId"},
			Resources: []apigenyaml.Resource{{
				Name:    "meta", Type: "BookMeta",
				Version: apigenyaml.VersionDef{Kind: "STRONG", Type: "U64"},
				Reader:  &apigenyaml.ReaderDef{},
			}},
		}},
	}
	ir, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	g := ir.Entities[0].Resources[0].Get
	if g.VersionField == nil {
		t.Fatal("Get VersionField is nil for STRONG")
	}
	if g.VersionField.Number != 2 {
		t.Errorf("Get VersionField.Number = %d, want 2", g.VersionField.Number)
	}
}
