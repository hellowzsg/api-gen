package ir

import (
	"testing"

	apigenyaml "github.com/acme/apigen/internal/yaml"
)

func TestOptionInjectionField(t *testing.T) {
	cfg := &apigenyaml.Config{
		Syntax: "v1", Name: "test",
		Entities: []apigenyaml.Entity{{
			Name: "book", Key: apigenyaml.KeyDef{Type: "BookId"},
			Resources: []apigenyaml.Resource{{
				Name:    "meta", Type: "BookMeta",
				Version: apigenyaml.VersionDef{Kind: "NONE"},
				Options: []apigenyaml.OptionDef{{
					Target: "field", Path: "meta", Option: "acme.cache", Value: false,
				}},
				Reader: &apigenyaml.ReaderDef{},
			}},
		}},
	}
	ir, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	r := ir.Entities[0].Resources[0]
	if len(r.Options) != 1 {
		t.Fatalf("len(Options) = %d, want 1", len(r.Options))
	}
	opt := r.Options[0]
	if opt.Target != "field" {
		t.Errorf("Target = %q, want field", opt.Target)
	}
}

func TestOptionValidateTarget(t *testing.T) {
	validTargets := []string{"field", "message", "rpc", "service", "file"}
	for _, target := range validTargets {
		t.Run(target, func(t *testing.T) {
			opt := apigenyaml.OptionDef{Target: target, Path: "x", Option: "acme.x", Value: true}
			if err := ValidateOption(opt); err != nil {
				t.Errorf("ValidateOption(target=%q) = %v, want nil", target, err)
			}
		})
	}
	opt := apigenyaml.OptionDef{Target: "invalid", Option: "acme.x", Value: true}
	if err := ValidateOption(opt); err == nil {
		t.Error("ValidateOption(target=invalid) = nil, want error")
	}
}

func TestOptionValidateFieldTargetRequiresPath(t *testing.T) {
	opt := apigenyaml.OptionDef{Target: "field", Path: "", Option: "acme.x", Value: true}
	if err := ValidateOption(opt); err == nil {
		t.Error("field target with empty path should fail")
	}
}

func TestOptionValidateServiceFileTargetNoPath(t *testing.T) {
	for _, target := range []string{"service", "file"} {
		t.Run(target, func(t *testing.T) {
			opt := apigenyaml.OptionDef{Target: target, Option: "acme.x", Value: true}
			if err := ValidateOption(opt); err != nil {
				t.Errorf("ValidateOption(target=%q) = %v, want nil", target, err)
			}
		})
	}
}

func TestOptionValidateOptionName(t *testing.T) {
	tests := []struct {
		name    string
		option  string
		wantErr bool
	}{
		{"valid dotted", "acme.cache", false},
		{"valid single", "cache", false},
		{"empty", "", true},
		{"with spaces", "acme .cache", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := apigenyaml.OptionDef{Target: "service", Option: tt.option, Value: true}
			err := ValidateOption(opt)
			if tt.wantErr && err == nil {
				t.Errorf("ValidateOption(option=%q) = nil, want error", tt.option)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateOption(option=%q) = %v, want nil", tt.option, err)
			}
		})
	}
}

// TestFormatOptionValue_MapKeySorted verifies that map values are formatted
// with keys in deterministic (lexicographic) order — same input must always
// produce the same output (reproducible generation).
func TestFormatOptionValue_MapKeySorted(t *testing.T) {
	value := map[string]interface{}{
		"zeta":  1,
		"alpha": "a",
		"mid":   true,
		"nested": map[string]interface{}{
			"b": 2,
			"a": 1,
		},
	}
	first := FormatOptionValue(value)
	for i := 0; i < 50; i++ {
		if got := FormatOptionValue(value); got != first {
			t.Fatalf("FormatOptionValue not deterministic: %q vs %q", first, got)
		}
	}
	want := "{alpha: \"a\", mid: true, nested: {a: 1, b: 2}, zeta: 1}"
	if first != want {
		t.Errorf("FormatOptionValue = %q, want %q (keys sorted, nested sorted)", first, want)
	}
}
