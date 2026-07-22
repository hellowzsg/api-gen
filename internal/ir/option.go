package ir

import (
	"fmt"
	"sort"
	"strings"

	apigenyaml "github.com/acme/apigen/internal/yaml"
)

var validOptionTargets = map[string]bool{
	"field":   true,
	"message": true,
	"rpc":     true,
	"service": true,
	"file":    true,
}

var targetsRequirePath = map[string]bool{
	"field":   true,
	"message": true,
	"rpc":     true,
}

// ValidateOption validates an option injection declaration.
func ValidateOption(opt apigenyaml.OptionDef) error {
	if !validOptionTargets[opt.Target] {
		return fmt.Errorf("invalid option target %q: must be one of field/message/rpc/service/file", opt.Target)
	}
	if targetsRequirePath[opt.Target] && opt.Path == "" {
		return fmt.Errorf("option target %q requires non-empty path", opt.Target)
	}
	if opt.Option == "" {
		return fmt.Errorf("option name is empty")
	}
	if err := validateOptionName(opt.Option); err != nil {
		return fmt.Errorf("invalid option name %q: %w", opt.Option, err)
	}
	return nil
}

func validateOptionName(name string) error {
	if strings.Contains(name, " ") {
		return fmt.Errorf("contains spaces")
	}
	if name == "" {
		return fmt.Errorf("empty")
	}
	for _, ch := range name {
		if !(ch == '.' || ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')) {
			return fmt.Errorf("illegal character %q", ch)
		}
	}
	return nil
}

// ValidateAllOptions validates all options across all entities/resources.
func ValidateAllOptions(ir *IR) error {
	for _, e := range ir.Entities {
		for _, r := range e.Resources {
			for _, opt := range r.Options {
				if err := ValidateOption(opt); err != nil {
					return fmt.Errorf("entity %q resource %q: %w", e.Name, r.Name, err)
				}
			}
		}
	}
	return nil
}

// FormatOptionValue formats an option value for proto text format.
func FormatOptionValue(value interface{}) string {
	switch v := value.(type) {
	case bool:
		if v {
			return "true"
		}
		return "false"
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%g", v)
	case string:
		return fmt.Sprintf("%q", v)
	case map[string]interface{}:
		// Map iteration order in Go is random; sort keys so generated
		// output is byte-for-byte reproducible across runs.
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%s: %s", k, FormatOptionValue(v[k])))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	default:
		return fmt.Sprintf("%v", v)
	}
}
