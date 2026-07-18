package yaml

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidateReferences validates type_ references and service entity references.
func (c *Config) ValidateReferences() error {
	if err := c.validateTypeReferences(); err != nil {
		return err
	}
	if err := c.validateServiceReferences(); err != nil {
		return err
	}
	if err := c.validateHTTPConfig(); err != nil {
		return err
	}
	if err := c.validatePerMethodHTTPOverrides(); err != nil {
		return err
	}
	return nil
}

// validateHTTPConfig validates the HTTP configuration (P1).
// When HTTP is disabled or not declared, no checks are performed (backward
// compatibility with P0 pure-gRPC users).
// When HTTP is enabled:
//   - body_style must be "" (default) or "wrapper"; "resource" is P2.
//   - generate_openapi must be false; OpenAPI generation is P2.
//   - import_protos must contain at least one googleapis source (path/git/bsr)
//     so that google/api/annotations.proto is resolvable.
func (c *Config) validateHTTPConfig() error {
	if c.Settings.HTTP == nil || !c.Settings.HTTP.Enable {
		return nil
	}
	hc := c.Settings.HTTP

	// body_style: "wrapper" (default) or "resource" (P2). Any other value
	// is invalid.
	if hc.BodyStyle != "" && hc.BodyStyle != "wrapper" && hc.BodyStyle != "resource" {
		return fmt.Errorf("settings.http.body_style: invalid value %q (only \"wrapper\" or \"resource\" is supported)", hc.BodyStyle)
	}

	// HTTP requires googleapis (google/api/annotations.proto) to be resolvable.
	if !c.hasGoogleapisDependency() {
		return fmt.Errorf("settings.http.enable: HTTP is enabled but no googleapis dependency found in import_protos; add a path/git/bsr entry that provides google/api/annotations.proto")
	}

	return nil
}

// hasGoogleapisDependency checks whether import_protos contains at least one
// entry that can provide google/api/annotations.proto. Detection is by source
// type:
//   - path: the path explicitly references the "google/api" directory segment
//     (e.g. "third_party/google/api/annotations.proto"), OR is a broad glob
//     ("**/*.proto") that could cover a vendored google/api directory. The
//     authoritative closure check is performed by the composite resolver
//     during protocompile link (fail-fast if annotations.proto is genuinely
//     absent).
//   - git: the repo URL contains "googleapis" (e.g. googleapis/googleapis).
//   - bsr: the module is buf.build/googleapis/googleapis.
func (c *Config) hasGoogleapisDependency() bool {
	for _, imp := range c.ImportProtos {
		if imp.Path != "" {
			if strings.Contains(imp.Path, "google/api") {
				return true
			}
			// Broad glob (e.g. "proto/**/*.proto") may cover a vendored
			// google/api directory. Accept conservatively; the protocompile
			// link step will fail-fast if annotations.proto is genuinely
			// absent.
			if strings.Contains(imp.Path, "**") {
				return true
			}
		}
		if imp.Git != "" {
			if strings.Contains(strings.ToLower(imp.Git), "googleapis") {
				return true
			}
		}
		if imp.BSR != "" {
			if strings.Contains(imp.BSR, "googleapis") {
				return true
			}
		}
	}
	return false
}

func (c *Config) validateTypeReferences() error {
	for i, e := range c.Entities {
		if err := validateTypeName(e.Key.Type); err != nil {
			return fmt.Errorf("entities[%d].key.type_: %w", i, err)
		}
		for j, r := range e.Resources {
			if err := validateTypeName(r.Type); err != nil {
				return fmt.Errorf("entities[%d].resources[%d].type_: %w", i, j, err)
			}
		}
	}
	return nil
}

// validateTypeName checks that a type_ value is syntactically valid.
func validateTypeName(typeName string) error {
	if typeName == "" {
		return fmt.Errorf("type_ is empty")
	}
	if typeName[0] == '.' || (typeName[0] >= '0' && typeName[0] <= '9') {
		return fmt.Errorf("invalid type_ %q: must start with letter or underscore", typeName)
	}
	for _, ch := range typeName {
		if !(ch == '.' || ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')) {
			return fmt.Errorf("invalid type_ %q: illegal character %q", typeName, ch)
		}
	}
	return nil
}

func (c *Config) validateServiceReferences() error {
	entityMap := make(map[string]*Entity, len(c.Entities))
	for i := range c.Entities {
		e := &c.Entities[i]
		if _, exists := entityMap[e.Name]; exists {
			return fmt.Errorf("duplicate entity name %q", e.Name)
		}
		entityMap[e.Name] = e
	}
	for si, s := range c.Services {
		for ei, se := range s.Entities {
			entity, ok := entityMap[se.Name]
			if !ok {
				return fmt.Errorf("services[%d].entities[%d]: references nonexistent entity %q", si, ei, se.Name)
			}
			if len(se.Resources) == 0 {
				continue
			}
			entityResourceNames := make(map[string]bool, len(entity.Resources))
			for _, r := range entity.Resources {
				entityResourceNames[r.Name] = true
			}
			for ri, sr := range se.Resources {
				if !entityResourceNames[sr.Name] {
					return fmt.Errorf("services[%d].entities[%d].resources[%d]: %q is not declared in entity %q", si, ei, ri, sr.Name, se.Name)
				}
			}
		}
	}
	return nil
}

// ResolveTypeName resolves a type_ reference to a fully-qualified message name.
func (c *Config) ResolveTypeName(typeName string) string {
	if strings.Contains(typeName, ".") {
		return typeName
	}
	return c.Name + "." + typeName
}

// pathVarPattern matches a path variable of the form {name} or {a.b.c}.
// It captures the inner content (without braces). An empty inner (e.g. {})
// or a dangling dot (e.g. {key.}) is considered malformed.
var pathVarPattern = regexp.MustCompile(`\{([^}]*)\}`)

// validatePerMethodHTTPOverrides validates the path variable syntax of every
// per-method HTTP override declared in the config. This is a syntax-level
// check: each {var} must be non-empty and not end with a dot. Deep key-leaf
// reachability validation (via ir.ValidatePathVariables) is performed in the
// IR build phase when key descriptors are available.
//
// TODO(P2 task group 2): supplement with ir.ValidatePathVariables for
// key.* variables once key descriptors are resolved.
func (c *Config) validatePerMethodHTTPOverrides() error {
	for i := range c.Entities {
		for j := range c.Entities[i].Resources {
			r := &c.Entities[i].Resources[j]
			if r.Reader != nil && r.Reader.HTTP != nil {
				if err := validatePathVarSyntax(r.Reader.HTTP.Path); err != nil {
					return fmt.Errorf("entities[%d].resources[%d].reader.http.path: %w", i, j, err)
				}
			}
			if r.Writer != nil && r.Writer.Update != nil && r.Writer.Update.HTTP != nil {
				if err := validatePathVarSyntax(r.Writer.Update.HTTP.Path); err != nil {
					return fmt.Errorf("entities[%d].resources[%d].writer.update.http.path: %w", i, j, err)
				}
			}
		}
	}
	for i := range c.Services {
		for j := range c.Services[i].CustomMethods {
			cm := &c.Services[i].CustomMethods[j]
			if cm.HTTP != nil {
				if err := validatePathVarSyntax(cm.HTTP.Path); err != nil {
					return fmt.Errorf("services[%d].custom_methods[%d].http.path: %w", i, j, err)
				}
			}
		}
	}
	return nil
}

// validatePathVarSyntax checks that every {var} in the path is well-formed:
// non-empty, starts and ends with an identifier character (not a dot).
func validatePathVarSyntax(path string) error {
	matches := pathVarPattern.FindAllStringSubmatch(path, -1)
	for _, m := range matches {
		inner := m[1]
		if inner == "" {
			return fmt.Errorf("empty path variable {} in %q", path)
		}
		if strings.HasPrefix(inner, ".") || strings.HasSuffix(inner, ".") {
			return fmt.Errorf("malformed path variable {%s} in %q (leading/trailing dot)", inner, path)
		}
	}
	return nil
}
