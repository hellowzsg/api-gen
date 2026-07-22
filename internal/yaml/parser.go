// Package yaml parses the four-section api.yaml configuration file.
package yaml

import (
	"fmt"
	"io"
	"regexp"

	"gopkg.in/yaml.v3"
)

// Config is the root structure of api.yaml.
type Config struct {
	Syntax       string    `yaml:"syntax"`
	Name         string    `yaml:"name"`
	ImportProtos []Import  `yaml:"import_protos"`
	Settings     Settings  `yaml:"settings"`
	Entities     []Entity  `yaml:"entities"`
	Services     []Service `yaml:"services"`
}

// Import declares a proto dependency (one of path/git/bsr).
type Import struct {
	Path    string `yaml:"path,omitempty"`
	Git     string `yaml:"git,omitempty"`
	Ref     string `yaml:"ref,omitempty"`
	Subdir  string `yaml:"subdir,omitempty"`
	Alias   string `yaml:"alias,omitempty"`
	BSR     string `yaml:"bsr,omitempty"`
	Version string `yaml:"version,omitempty"`
}

// Settings holds generation configuration.
type Settings struct {
	GoRepo  string        `yaml:"go_repo"`
	JsRepo  string        `yaml:"js_repo"`
	Out     OutConfig     `yaml:"out"`
	HTTP    *HTTPConfig   `yaml:"http,omitempty"`
	Plugins PluginsConfig `yaml:"plugins,omitempty"`
}

// PluginsConfig holds plugin declarations for code generation.
type PluginsConfig struct {
	JS []string `yaml:"js,omitempty"`
}

// OutConfig defines output directories.
type OutConfig struct {
	Proto   string `yaml:"proto"`
	Go      string `yaml:"go"`
	Js      string `yaml:"js"`
	OpenAPI string `yaml:"openapi,omitempty"`
}

// HTTPConfig holds HTTP routing strategy (P1, not used in P0).
type HTTPConfig struct {
	Enable          bool   `yaml:"enable"`
	Prefix          string `yaml:"prefix,omitempty"`
	BodyStyle       string `yaml:"body_style,omitempty"`
	GenerateOpenAPI bool   `yaml:"generate_openapi,omitempty"`
}

// HTTPOverride declares per-method HTTP routing override (P2).
// When set on a reader/writer/custom_method, the specified fields
// override the global settings.http defaults. Fields left empty
// inherit the global default.
type HTTPOverride struct {
	Verb      string `yaml:"verb,omitempty"`
	Path      string `yaml:"path,omitempty"`
	Body      string `yaml:"body,omitempty"`
	BodyStyle string `yaml:"body_style,omitempty"`
}

// Entity models a business object: key + multiple resources + entity-level writes.
type Entity struct {
	Name       string     `yaml:"name"`
	Key        KeyDef     `yaml:"key"`
	Create     *struct{}  `yaml:"create,omitempty"`
	Delete     *struct{}  `yaml:"delete,omitempty"`
	DeleteSoft *struct{}  `yaml:"delete_soft,omitempty"`
	Resources  []Resource `yaml:"resources"`
}

// KeyDef declares the primary key type.
type KeyDef struct {
	Type string `yaml:"type_"`
}

// Resource is a data facet of an entity.
type Resource struct {
	Name    string      `yaml:"name"`
	Type    string      `yaml:"type_"`
	Version VersionDef  `yaml:"version"`
	Options []OptionDef `yaml:"options,omitempty"`
	Reader  *ReaderDef  `yaml:"reader,omitempty"`
	Writer  *WriterDef  `yaml:"writer,omitempty"`
}

// VersionDef declares the optimistic lock strategy.
type VersionDef struct {
	Kind string `yaml:"kind"`      // STRONG / WEAK / NONE
	Type string `yaml:"type,omitempty"` // U64 / U32 / STRING
}

// OptionDef declares a generic option injection (field/message/rpc/service/file).
type OptionDef struct {
	Target string      `yaml:"target"`
	Path   string      `yaml:"path,omitempty"`
	Option string      `yaml:"option"`
	Value  interface{} `yaml:"value"`
}

// ReaderDef declares resource-level read methods.
type ReaderDef struct {
	Batch      bool        `yaml:"batch,omitempty"`
	List       bool        `yaml:"list,omitempty"`
	ListConfig *ListConfig `yaml:"list_config,omitempty"`
	HTTP       *HTTPOverride `yaml:"http,omitempty"`
}

// ListConfig holds List sub-configuration.
type ListConfig struct {
	TotalSize  bool   `yaml:"total_size"`
	FilterType string `yaml:"filter_type,omitempty"`
}

// WriterDef declares resource-level write methods.
type WriterDef struct {
	Update *UpdateDef `yaml:"update,omitempty"`
}

// UpdateDef declares Update method options.
type UpdateDef struct {
	Mask bool          `yaml:"mask"`
	HTTP *HTTPOverride `yaml:"http,omitempty"`
}

// Service assembles entities into an exposed service.
type Service struct {
	Name          string          `yaml:"name"`
	Entities      []ServiceEntity `yaml:"entities"`
	CustomMethods []CustomMethod  `yaml:"custom_methods,omitempty"`
}

// ServiceEntity references an entity, optionally narrowing resources.
type ServiceEntity struct {
	Name      string     `yaml:"name"`
	Resources []Resource `yaml:"resources,omitempty"`
}

// CustomMethod declares a user-defined RPC.
type CustomMethod struct {
	Name     string        `yaml:"name"`
	Request  string        `yaml:"request"`
	Response string        `yaml:"response"`
	HTTP     *HTTPOverride `yaml:"http,omitempty"`
}

// namePattern validates proto package names (dotted.identifier).
var namePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*(\.[a-zA-Z][a-zA-Z0-9_]*)*$`)

// Parse reads and validates an api.yaml configuration from r.
func Parse(r io.Reader) (*Config, error) {
	var cfg Config
	dec := yaml.NewDecoder(r)
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode api.yaml: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) validate() error {
	if c.Syntax == "" {
		return fmt.Errorf("missing required field: syntax")
	}
	if c.Name == "" {
		return fmt.Errorf("missing required field: name")
	}
	if !namePattern.MatchString(c.Name) {
		return fmt.Errorf("invalid name %q: must be a valid proto package name (dotted identifiers, start with letter)", c.Name)
	}
	if c.Entities == nil {
		return fmt.Errorf("missing required field: entities")
	}
	for i, e := range c.Entities {
		if e.Name == "" {
			return fmt.Errorf("entities[%d]: name is required", i)
		}
		if e.Key.Type == "" {
			return fmt.Errorf("entities[%d].key: type_ is required", i)
		}
		if len(e.Resources) == 0 {
			return fmt.Errorf("entities[%d]: at least one resource is required", i)
		}
		for j, r := range e.Resources {
			if r.Name == "" {
				return fmt.Errorf("entities[%d].resources[%d]: name is required", i, j)
			}
			if r.Type == "" {
				return fmt.Errorf("entities[%d].resources[%d]: type_ is required", i, j)
			}
			if r.Version.Kind == "" {
				return fmt.Errorf("entities[%d].resources[%d].version: kind is required (STRONG/WEAK/NONE)", i, j)
			}
		}
	}
	return nil
}
