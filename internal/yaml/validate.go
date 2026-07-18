package yaml

import (
	"fmt"
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

	// P1 only supports body_style: wrapper (default). "resource" is P2.
	if hc.BodyStyle == "resource" {
		return fmt.Errorf("settings.http.body_style: %q is not supported in P1 (only \"wrapper\"); body_style: resource will be supported in P2", hc.BodyStyle)
	}
	if hc.BodyStyle != "" && hc.BodyStyle != "wrapper" {
		return fmt.Errorf("settings.http.body_style: invalid value %q (only \"wrapper\" or empty is supported in P1)", hc.BodyStyle)
	}

	// P1 does not support OpenAPI generation.
	if hc.GenerateOpenAPI {
		return fmt.Errorf("settings.http.generate_openapi: OpenAPI generation is not supported in P1; it will be supported in P2")
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
