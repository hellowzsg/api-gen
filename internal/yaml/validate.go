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
	return nil
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
