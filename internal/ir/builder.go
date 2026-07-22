// Package ir builds the intermediate representation from parsed YAML config.
package ir

import (
	"fmt"
	"strings"
	"unicode"

	"google.golang.org/protobuf/reflect/protoreflect"

	apigenyaml "github.com/hellowzsg/api-gen/internal/yaml"
)

// IR is the intermediate representation for code generation.
type IR struct {
	PackageName string
	Entities    []EntityIR
	Services    []ServiceIR
	// TypeImportPaths maps a fully-qualified type name (e.g.
	// "demo.business.book.BookMeta") to the proto file import path that
	// defines it (e.g. "demo/business/book/book.proto"). Populated by the
	// CLI from the resolved proto files so the renderer emits exact imports
	// instead of guessing the file path from the package name.
	TypeImportPaths map[string]string
	// HTTPEnabled indicates whether HTTP annotations should be generated.
	// When false, all HTTPAnnotation fields on methods are nil and the
	// renderer emits a pure-gRPC proto (P0 behavior).
	HTTPEnabled bool
	// HTTPPrefix is the optional URI prefix (e.g. "/library"). Empty means
	// no prefix.
	HTTPPrefix string
}

// BuildOptions provides optional inputs to BuildWithOptions that are not
// available from the YAML config alone — specifically, resolved proto message
// descriptors for key types, needed to extract scalar leaves for HTTP path
// binding.
type BuildOptions struct {
	// KeyDescriptors maps a fully-qualified key type name (e.g.
	// "test.BookId") to its resolved protoreflect.MessageDescriptor. Used
	// only when HTTP is enabled to extract scalar leaf fields for path
	// binding. When HTTP is disabled or a key type is missing from this
	// map, KeyLeaves is left empty and HTTP path generation is skipped for
	// that entity (with an error if HTTP is enabled).
	KeyDescriptors map[string]protoreflect.MessageDescriptor
	// LenientHTTP skips the KeyDescriptors requirement when HTTP is
	// enabled: key-leaf extraction is skipped (annotations carry no key
	// segments). Used by display-only commands (e.g. `entity list`) that
	// do not need HTTP paths. generate/build keep the strict requirement.
	LenientHTTP bool
}

type EntityIR struct {
	Name       string
	PascalName string
	KeyType    string
	Create     *CreateIR
	Delete     *DeleteIR
	DeleteSoft *DeleteIR
	Resources  []ResourceIR
	// KeyLeaves holds the scalar leaf fields extracted from the key type's
	// message tree. Populated only when HTTP is enabled. Used by the
	// renderer to generate URL path variables (e.g. {key.id}).
	KeyLeaves []KeyLeaf
}

type CreateIR struct {
	RPCName          string
	RequestName      string
	ResponseName     string
	RequestFields    []FieldIR
	ResponseKeyField FieldIR
	HTTPAnnotation   *HTTPAnnotation
}

type DeleteIR struct {
	RPCName        string
	RequestName    string
	ResponseName   string
	KeyField       FieldIR
	HTTPAnnotation *HTTPAnnotation
}

type ResourceIR struct {
	Name       string
	PascalName string
	Type       string
	Version    VersionIR
	Options    []apigenyaml.OptionDef
	Get        *GetIR
	BatchGet   *BatchGetIR
	List       *ListIR
	Update     *UpdateIR
}

type VersionIR struct {
	Kind      string
	Type      string
	FieldType string
	IsWrapper bool
}

type GetIR struct {
	RPCName       string
	RequestName   string
	ResponseName  string
	KeyField      FieldIR
	ResourceField FieldIR
	VersionField  *FieldIR
	HTTPAnnotation *HTTPAnnotation
}

type BatchGetIR struct {
	RPCName        string
	RequestName    string
	ResponseName   string
	KeysField      FieldIR
	ResourcesField FieldIR
	HTTPAnnotation *HTTPAnnotation
}

type ListIR struct {
	RPCName        string
	RequestName    string
	ResponseName   string
	PageSize       FieldIR
	PageToken      FieldIR
	Filter         FieldIR
	OrderBy        FieldIR
	ResourcesField FieldIR
	NextPageToken  FieldIR
	TotalSize      *FieldIR
	HTTPAnnotation *HTTPAnnotation
}

type UpdateIR struct {
	RPCName       string
	RequestName   string
	ResponseName  string
	RequestFields []FieldIR
	HasVersion    bool
	VersionField  *FieldIR
	Mask          bool
	HTTPAnnotation *HTTPAnnotation
}

type FieldIR struct {
	Name     string
	Type     string
	Number   int
	Repeated bool
}

// HTTPAnnotation describes a google.api.http annotation for an RPC method.
// Populated only when HTTP is enabled (IR.HTTPEnabled == true). When nil,
// the renderer omits the google.api.http option for that RPC.
//
// The annotation is service-agnostic: it carries structured path segments
// instead of a pre-baked path string. The final URL path is resolved per
// service at render time via ResolvePath.
type HTTPAnnotation struct {
	// Verb is the HTTP method: GET, POST, PATCH, or DELETE.
	Verb string
	// Body is the body binding: "" (no body), "*" (whole wrapper), or
	// "<field>" (P2 body_style: resource). In P1, Body is either "" or "*".
	Body string
	// IsOverride indicates the path was user-declared (per-method http
	// override or custom method). The override path is used verbatim unless
	// OverrideTemplateSvc is also set (entity-level overrides are templates:
	// the service segment is rewritten per rendering service).
	IsOverride bool
	// OverridePath is the full user-declared path (only when IsOverride).
	OverridePath string
	// OverrideTemplateSvc, when non-empty, marks the override path as a
	// template whose path segment equal to this value is replaced by the
	// rendering service name at ResolvePath time. Used for entity-level
	// reader.http/writer.update.http overrides that are inherited across
	// services via narrowing — without rewriting, two services sharing an
	// entity would register identical routes and collide. Empty for custom
	// method overrides (those are bound to a single service and verbatim).
	OverrideTemplateSvc string
	// Entity is the entity (collection) segment for default paths.
	Entity string
	// KeyLeaves are the scalar key leaf fields bound as URL path variables
	// for default paths (e.g. {key.id}).
	KeyLeaves []KeyLeaf
	// Resource is the resource segment for resource-level methods
	// ("" for entity-level methods).
	Resource string
	// Suffix is the trailing custom segment: "batchGet" / "list" /
	// "deleteSoft" / "".
	Suffix string
}

// ResolvePath computes the final URL path for the given service and prefix.
//
//   - Default paths compose as:
//     <prefix>/<svcName>/<entity>[/{key.leaf}...][/<resource>][/<suffix>]
//   - Verbatim overrides (custom methods) are returned as-is.
//   - Template overrides (entity-level reader/writer http): the path segment
//     equal to OverrideTemplateSvc is replaced by svcName so each service
//     inheriting the entity gets an isolated route.
func (a *HTTPAnnotation) ResolvePath(prefix, svcName string) string {
	if a.IsOverride {
		if a.OverrideTemplateSvc == "" || a.OverrideTemplateSvc == svcName {
			return a.OverridePath
		}
		return rewriteSegment(a.OverridePath, a.OverrideTemplateSvc, svcName)
	}
	parts := make([]string, 0, 4+len(a.KeyLeaves))
	parts = append(parts, svcName, a.Entity)
	for _, l := range a.KeyLeaves {
		parts = append(parts, "{key."+l.DotPath+"}")
	}
	if a.Resource != "" {
		parts = append(parts, a.Resource)
	}
	if a.Suffix != "" {
		parts = append(parts, a.Suffix)
	}
	path := "/" + strings.Join(parts, "/")
	if prefix != "" {
		path = prefix + path
	}
	return path
}

// rewriteSegment replaces the first path segment equal to old with new.
// Segments are split on "/"; variable segments ({...}) and the leading
// empty segment from the leading "/" are never matched.
func rewriteSegment(path, old, new string) string {
	segs := strings.Split(path, "/")
	for i, s := range segs {
		if s == old {
			segs[i] = new
			return strings.Join(segs, "/")
		}
	}
	return path
}

type ServiceIR struct {
	Name          string
	ProtoPackage  string
	GoPackage     string
	GoRepo        string
	OutGoDir      string
	Entities      []ServiceEntityIR
	CustomMethods []CustomMethodIR
}

type ServiceEntityIR struct {
	Name      string
	// Resources is the per-service resource narrowing spec. When non-empty,
	// only the listed resources (and their narrowed reader/writer methods)
	// are exposed by this service. When empty, the entity's full resource
	// set is inherited.
	Resources []ResourceNarrowIR
}

// ResourceNarrowIR describes a per-service resource narrowing.
type ResourceNarrowIR struct {
	Name   string
	Reader *ReaderNarrowIR
	Writer *WriterNarrowIR
}

type ReaderNarrowIR struct {
	Batch *bool // nil = inherit, true/false = override
	List  *bool
}

type WriterNarrowIR struct {
	Update *bool // nil = inherit; when false, no Update generated
}

type CustomMethodIR struct {
	Name     string
	Request  string
	Response string
	// HTTPAnnotation is populated when the custom method declares an http
	// block and HTTP is enabled. nil otherwise.
	HTTPAnnotation *HTTPAnnotation
}

// Build constructs the IR from parsed YAML config. This is the P0 entry
// point (HTTP not configured). For HTTP support use BuildWithOptions.
func Build(cfg *apigenyaml.Config) (*IR, error) {
	return BuildWithOptions(cfg, BuildOptions{})
}

// BuildWithOptions constructs the IR from parsed YAML config with optional
// inputs (key descriptors for HTTP path binding). When cfg.Settings.HTTP is
// nil or Enable is false, behavior is identical to Build (P0 pure-gRPC).
func BuildWithOptions(cfg *apigenyaml.Config, opts BuildOptions) (*IR, error) {
	httpEnabled := cfg.Settings.HTTP != nil && cfg.Settings.HTTP.Enable
	httpPrefix := ""
	if httpEnabled {
		httpPrefix = cfg.Settings.HTTP.Prefix
	}
	ir := &IR{
		PackageName: cfg.Name,
		HTTPEnabled: httpEnabled,
		HTTPPrefix:  httpPrefix,
	}
	for i := range cfg.Entities {
		entityIR, err := buildEntity(&cfg.Entities[i], cfg, opts, httpEnabled)
		if err != nil {
			return nil, err
		}
		ir.Entities = append(ir.Entities, *entityIR)
	}
	for i := range cfg.Services {
		ir.Services = append(ir.Services, buildService(&cfg.Services[i], cfg, httpEnabled))
	}
	return ir, nil
}

func buildEntity(e *apigenyaml.Entity, cfg *apigenyaml.Config, opts BuildOptions, httpEnabled bool) (*EntityIR, error) {
	pascalName := toPascalCase(e.Name)
	entity := &EntityIR{
		Name:       e.Name,
		PascalName: pascalName,
		KeyType:    cfg.ResolveTypeName(e.Key.Type),
	}
	// Extract key leaves for HTTP path binding when HTTP is enabled.
	// LenientHTTP (display-only commands) skips this entirely.
	if httpEnabled && !opts.LenientHTTP {
		keyDesc, ok := opts.KeyDescriptors[entity.KeyType]
		if !ok {
			return nil, fmt.Errorf("HTTP enabled but key descriptor for %q not provided in BuildOptions.KeyDescriptors", entity.KeyType)
		}
		leaves, err := ExtractKeyLeaves(keyDesc)
		if err != nil {
			return nil, fmt.Errorf("entity %q key type %q: %w", e.Name, entity.KeyType, err)
		}
		entity.KeyLeaves = leaves
	}
	// Build method-specific HTTP annotations. Annotations are
	// service-agnostic structured segments (entity/keyLeaves/resource/
	// suffix); the final path is resolved per service at render time via
	// HTTPAnnotation.ResolvePath.
	httpCtx := &httpBuildContext{
		enabled:     httpEnabled,
		entity:      e.Name,
		keyLeaves:   entity.KeyLeaves,
		templateSvc: firstServiceForEntity(cfg, e.Name),
	}
	if httpEnabled && cfg.Settings.HTTP != nil {
		httpCtx.bodyStyle = cfg.Settings.HTTP.BodyStyle
	}
	if e.Create != nil {
		entity.Create = buildCreate(pascalName, e.Resources, entity.KeyType, cfg)
		ann, err := httpCtx.buildCreateAnnotationWithResources(len(e.Resources))
		if err != nil {
			return nil, err
		}
		entity.Create.HTTPAnnotation = ann
	}
	if e.Delete != nil {
		entity.Delete = buildDelete(entity.KeyType, "Delete"+pascalName)
		entity.Delete.HTTPAnnotation = httpCtx.buildDeleteAnnotation()
	}
	if e.DeleteSoft != nil {
		entity.DeleteSoft = buildDelete(entity.KeyType, "Delete"+pascalName+"Soft")
		entity.DeleteSoft.HTTPAnnotation = httpCtx.buildDeleteSoftAnnotation()
	}
	for i := range e.Resources {
		resource, err := buildResource(&e.Resources[i], pascalName, entity.KeyType, cfg)
		if err != nil {
			return nil, err
		}
		if err := httpCtx.fillResourceAnnotations(resource, &e.Resources[i]); err != nil {
			return nil, fmt.Errorf("entity %q: %w", e.Name, err)
		}
		entity.Resources = append(entity.Resources, *resource)
	}
	return entity, nil
}

// firstServiceForEntity returns the name of the first service that references
// the given entity, or "" if no service references it. Used as the
// service-segment placeholder for entity-level http override templates: the
// user writes the override path using this service's name, and at render
// time the segment equal to this value is replaced with the rendering
// service's name (so two services inheriting the entity via narrowing do
// not register identical routes and collide).
func firstServiceForEntity(cfg *apigenyaml.Config, entityName string) string {
	for _, s := range cfg.Services {
		for _, se := range s.Entities {
			if se.Name == entityName {
				return s.Name
			}
		}
	}
	return ""
}

// httpBuildContext carries the context needed to construct HTTPAnnotation
// for each method of a single entity. Paths are NOT built here — only
// structured segments.
type httpBuildContext struct {
	enabled     bool
	entity      string
	keyLeaves   []KeyLeaf
	bodyStyle   string // global body_style from settings.http
	templateSvc string // first service referencing this entity; used as the
	// service-segment placeholder for entity-level http override templates.
}

// buildCreateAnnotationWithResources builds the Create HTTP annotation,
// returning an error when body_style: resource is set but the entity has
// multiple resources (ambiguous which resource field to bind as body).
func (h *httpBuildContext) buildCreateAnnotationWithResources(resourceCount int) (*HTTPAnnotation, error) {
	if !h.enabled {
		return nil, nil
	}
	if h.bodyStyle == "resource" && resourceCount > 1 {
		return nil, fmt.Errorf("body_style: resource is ambiguous for Create with %d resources; use body_style: wrapper (default) for multi-resource Create", resourceCount)
	}
	return &HTTPAnnotation{
		Verb:   "POST",
		Body:   "*",
		Entity: h.entity,
	}, nil
}

func (h *httpBuildContext) buildDeleteAnnotation() *HTTPAnnotation {
	if !h.enabled {
		return nil
	}
	return &HTTPAnnotation{
		Verb:      "DELETE",
		Entity:    h.entity,
		KeyLeaves: h.keyLeaves,
	}
}

func (h *httpBuildContext) buildDeleteSoftAnnotation() *HTTPAnnotation {
	if !h.enabled {
		return nil
	}
	return &HTTPAnnotation{
		Verb:   "POST",
		Body:   "*",
		Entity: h.entity,
		Suffix: "deleteSoft",
	}
}

// fillResourceAnnotations populates HTTPAnnotation for Get/BatchGet/List/Update.
// When yamlResource declares per-method HTTP overrides (reader.http /
// writer.update.http), the overridden fields replace the defaults. Override
// paths are validated against the entity's key leaves (fail-fast).
func (h *httpBuildContext) fillResourceAnnotations(r *ResourceIR, yr *apigenyaml.Resource) error {
	if !h.enabled {
		return nil
	}
	if r.Get != nil {
		r.Get.HTTPAnnotation = &HTTPAnnotation{
			Verb:      "GET",
			Entity:    h.entity,
			KeyLeaves: h.keyLeaves,
			Resource:  r.Name,
		}
		// Get has no http override in P2 (only reader-level List/BatchGet
		// overrides are supported via reader.http, which applies to the
		// reader block as a whole — but design §8.3 shows reader.http on
		// the reader block. We interpret reader.http as applying to the
		// List method when list is enabled, and to BatchGet when batch is
		// enabled. Get does not use reader.http.)
	}
	if r.BatchGet != nil {
		r.BatchGet.HTTPAnnotation = &HTTPAnnotation{
			Verb:     "POST",
			Body:     "*",
			Entity:   h.entity,
			Resource: r.Name,
			Suffix:   "batchGet",
		}
		// reader.http override applies only to List (the primary reader
		// method), not to BatchGet. BatchGet retains its default route.
	}
	if r.List != nil {
		r.List.HTTPAnnotation = &HTTPAnnotation{
			Verb:     "POST",
			Body:     "*",
			Entity:   h.entity,
			Resource: r.Name,
			Suffix:   "list",
		}
		if yr.Reader != nil && yr.Reader.HTTP != nil {
			ann, err := h.applyOverride(r.List.HTTPAnnotation, yr.Reader.HTTP, r.Name, "List", h.templateSvc)
			if err != nil {
				return err
			}
			r.List.HTTPAnnotation = ann
		}
	}
	if r.Update != nil {
		r.Update.HTTPAnnotation = &HTTPAnnotation{
			Verb:      "PATCH",
			Entity:    h.entity,
			KeyLeaves: h.keyLeaves,
			Resource:  r.Name,
			Body:      h.bodyForStyle(h.bodyStyle, r.Name),
		}
		if yr.Writer != nil && yr.Writer.Update != nil && yr.Writer.Update.HTTP != nil {
			ann, err := h.applyOverride(r.Update.HTTPAnnotation, yr.Writer.Update.HTTP, r.Name, "Update", h.templateSvc)
			if err != nil {
				return err
			}
			r.Update.HTTPAnnotation = ann
		}
	}
	return nil
}

// applyOverride returns a new HTTPAnnotation with fields from override
// applied on top of def. Fields empty in override inherit from def. An
// override path is stored as a template: the segment equal to templateSvc
// (the first service referencing this entity) is replaced per rendering
// service at ResolvePath time, so two services inheriting the entity via
// narrowing do not register colliding routes. Its {key.*} variables are
// validated against the entity's key leaves.
// When override declares body_style: resource, body is set to resourceName.
func (h *httpBuildContext) applyOverride(def *HTTPAnnotation, override *apigenyaml.HTTPOverride, resourceName, methodName, templateSvc string) (*HTTPAnnotation, error) {
	out := *def
	if override.Verb != "" {
		out.Verb = strings.ToUpper(override.Verb)
	}
	if override.Path != "" {
		if err := ValidatePathVariables(override.Path, h.keyLeaves); err != nil {
			return nil, fmt.Errorf("resource %q method %s: %w", resourceName, methodName, err)
		}
		out.IsOverride = true
		out.OverridePath = override.Path
		out.OverrideTemplateSvc = templateSvc
	}
	// Body resolution priority: explicit body > body_style > verb-derived default.
	switch {
	case override.Body != "":
		out.Body = override.Body
	case override.BodyStyle != "":
		out.Body = h.bodyForStyle(override.BodyStyle, resourceName)
	case override.Verb != "" && (strings.ToUpper(override.Verb) == "GET" || strings.ToUpper(override.Verb) == "DELETE"):
		// Verb overridden to GET/DELETE with no body/body_style → no body.
		out.Body = ""
	}
	return &out, nil
}

// bodyForStyle returns the body binding for a given body_style and resource.
// "wrapper" (default) → "*"; "resource" → resourceName.
func (h *httpBuildContext) bodyForStyle(bodyStyle, resourceName string) string {
	if bodyStyle == "resource" {
		return resourceName
	}
	return "*"
}

func buildCreate(entityName string, resources []apigenyaml.Resource, keyType string, cfg *apigenyaml.Config) *CreateIR {
	c := &CreateIR{
		RPCName:      "Create" + entityName,
		RequestName:  "Create" + entityName + "Request",
		ResponseName: "Create" + entityName + "Response",
	}
	for i, r := range resources {
		c.RequestFields = append(c.RequestFields, FieldIR{
			Name:   r.Name,
			Type:   cfg.ResolveTypeName(r.Type),
			Number: i + 1,
		})
	}
	c.ResponseKeyField = FieldIR{Name: "key", Type: keyType, Number: 1}
	return c
}

func buildDelete(keyType, rpcName string) *DeleteIR {
	return &DeleteIR{
		RPCName:      rpcName,
		RequestName:  rpcName + "Request",
		ResponseName: "google.protobuf.Empty",
		KeyField:     FieldIR{Name: "key", Type: keyType, Number: 1},
	}
}

func buildResource(r *apigenyaml.Resource, entityName, keyType string, cfg *apigenyaml.Config) (*ResourceIR, error) {
	pascalName := toPascalCase(r.Name)
	resource := &ResourceIR{
		Name:       r.Name,
		PascalName: pascalName,
		Type:       cfg.ResolveTypeName(r.Type),
		Version:    buildVersion(r.Version),
		Options:    r.Options,
	}
	if r.Reader != nil {
		resource.Get = buildGet(entityName, pascalName, resource.Type, keyType, resource.Version)
		if r.Reader.Batch {
			resource.BatchGet = buildBatchGet(entityName, pascalName, resource.Type, keyType)
		}
		if r.Reader.List {
			resource.List = buildList(entityName, pascalName, resource.Type, r.Reader.ListConfig)
		}
	}
	if r.Writer != nil && r.Writer.Update != nil {
		resource.Update = buildUpdate(entityName, pascalName, resource.Type, keyType, resource.Version, r.Writer.Update.Mask)
	}
	return resource, nil
}

func buildGet(entityName, resourcePascal, resourceType, keyType string, v VersionIR) *GetIR {
	g := &GetIR{
		RPCName:       "Get" + entityName + resourcePascal,
		RequestName:   "Get" + entityName + resourcePascal + "Request",
		ResponseName:  "Get" + entityName + resourcePascal + "Response",
		KeyField:      FieldIR{Name: "key", Type: keyType, Number: 1},
		ResourceField: FieldIR{Name: strings.ToLower(entityName) + "_" + strings.ToLower(resourcePascal), Type: resourceType, Number: 1},
	}
	if v.Kind != "NONE" {
		g.VersionField = &FieldIR{Name: "version", Type: v.FieldType, Number: 2}
	}
	return g
}

func buildBatchGet(entityName, resourcePascal, resourceType, keyType string) *BatchGetIR {
	return &BatchGetIR{
		RPCName:        "BatchGet" + entityName + resourcePascal + "s",
		RequestName:    "BatchGet" + entityName + resourcePascal + "sRequest",
		ResponseName:   "BatchGet" + entityName + resourcePascal + "sResponse",
		KeysField:      FieldIR{Name: "keys", Type: keyType, Number: 1, Repeated: true},
		ResourcesField: FieldIR{Name: strings.ToLower(resourcePascal) + "s", Type: resourceType, Number: 1, Repeated: true},
	}
}

func buildList(entityName, resourcePascal, resourceType string, lc *apigenyaml.ListConfig) *ListIR {
	l := &ListIR{
		RPCName:        "List" + entityName + resourcePascal + "s",
		RequestName:    "List" + entityName + resourcePascal + "sRequest",
		ResponseName:   "List" + entityName + resourcePascal + "sResponse",
		PageSize:       FieldIR{Name: "page_size", Type: "int32", Number: 1},
		PageToken:      FieldIR{Name: "page_token", Type: "string", Number: 2},
		Filter:         FieldIR{Name: "filter", Type: "string", Number: 3},
		OrderBy:        FieldIR{Name: "order_by", Type: "string", Number: 4},
		ResourcesField: FieldIR{Name: strings.ToLower(resourcePascal) + "s", Type: resourceType, Number: 1, Repeated: true},
		NextPageToken:  FieldIR{Name: "next_page_token", Type: "string", Number: 2},
	}
	if lc == nil || lc.TotalSize {
		l.TotalSize = &FieldIR{Name: "total_size", Type: "int32", Number: 3}
	}
	return l
}

func buildUpdate(entityName, resourcePascal, resourceType, keyType string, v VersionIR, mask bool) *UpdateIR {
	u := &UpdateIR{
		RPCName:      "Update" + entityName + resourcePascal,
		RequestName:  "Update" + entityName + resourcePascal + "Request",
		ResponseName: "Update" + entityName + resourcePascal + "Response",
		Mask:         mask,
	}
	u.RequestFields = append(u.RequestFields, FieldIR{Name: strings.ToLower(resourcePascal), Type: resourceType, Number: 1})
	u.RequestFields = append(u.RequestFields, FieldIR{Name: "key", Type: keyType, Number: 2})
	if mask {
		u.RequestFields = append(u.RequestFields, FieldIR{Name: "update_mask", Type: "google.protobuf.FieldMask", Number: 3})
	}
	if v.Kind != "NONE" {
		num := 3
		if mask {
			num = 4
		}
		u.RequestFields = append(u.RequestFields, FieldIR{Name: "version", Type: v.FieldType, Number: num})
		u.HasVersion = true
		u.VersionField = &FieldIR{Name: "version", Type: v.FieldType, Number: 1}
	}
	if v.Kind != "STRONG" {
		u.ResponseName = "google.protobuf.Empty"
	}
	return u
}

func buildVersion(v apigenyaml.VersionDef) VersionIR {
	vir := VersionIR{Kind: v.Kind, Type: v.Type}
	switch v.Kind {
	case "STRONG":
		vir.FieldType = versionScalarType(v.Type)
	case "WEAK":
		vir.FieldType = versionWrapperType(v.Type)
		vir.IsWrapper = true
	}
	return vir
}

func versionScalarType(t string) string {
	switch t {
	case "U64":
		return "uint64"
	case "U32":
		return "uint32"
	case "STRING":
		return "string"
	}
	return "uint64"
}

func versionWrapperType(t string) string {
	switch t {
	case "U64":
		return "google.protobuf.UInt64Value"
	case "U32":
		return "google.protobuf.UInt32Value"
	case "STRING":
		return "google.protobuf.StringValue"
	}
	return "google.protobuf.UInt64Value"
}

func buildService(s *apigenyaml.Service, cfg *apigenyaml.Config, httpEnabled bool) ServiceIR {
	goPkg := ToSnakeCase(s.Name)
	sir := ServiceIR{
		Name:         s.Name,
		ProtoPackage: cfg.Name + "." + ToSnakeCase(s.Name),
		GoPackage:    goPkg,
		GoRepo:       cfg.Settings.GoRepo,
		OutGoDir:     cfg.Settings.Out.Go,
	}
	for _, se := range s.Entities {
		ser := ServiceEntityIR{Name: se.Name}
		for _, sr := range se.Resources {
			narrow := ResourceNarrowIR{Name: sr.Name}
			// When the service resource declares a reader block, only the
			// explicitly-enabled methods survive; unspecified methods are
			// narrowed off. (Design §十一 example: `reader: { list: true }`
			// → only List is exposed.)
			if sr.Reader != nil {
				b := sr.Reader.Batch
				l := sr.Reader.List
				narrow.Reader = &ReaderNarrowIR{Batch: &b, List: &l}
			}
			// When the service resource declares a writer block, Update is
			// exposed only if the update sub-block is present.
			if sr.Writer != nil {
				u := sr.Writer.Update != nil
				narrow.Writer = &WriterNarrowIR{Update: &u}
			}
			ser.Resources = append(ser.Resources, narrow)
		}
		sir.Entities = append(sir.Entities, ser)
	}
	for _, cm := range s.CustomMethods {
		cmIR := CustomMethodIR{Name: cm.Name, Request: cm.Request, Response: cm.Response}
		if httpEnabled && cm.HTTP != nil {
			// Custom method paths are fully user-declared (AIP-136 colon
			// syntax included) — stored verbatim, never rewritten.
			cmIR.HTTPAnnotation = &HTTPAnnotation{
				Verb:         strings.ToUpper(cm.HTTP.Verb),
				Body:         cm.HTTP.Body,
				IsOverride:   true,
				OverridePath: cm.HTTP.Path,
			}
		}
		sir.CustomMethods = append(sir.CustomMethods, cmIR)
	}
	return sir
}

func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

// ToSnakeCase converts a CamelCase name to snake_case (e.g.
// "LibraryService" → "library_service"). Shared by the IR builder and the
// CLI for output path derivation.
func ToSnakeCase(s string) string {
	var sb strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				sb.WriteByte('_')
			}
			sb.WriteRune(unicode.ToLower(r))
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}
