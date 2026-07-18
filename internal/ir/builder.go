// Package ir builds the intermediate representation from parsed YAML config.
package ir

import (
	"strings"
	"unicode"

	apigenyaml "github.com/acme/apigen/internal/yaml"
)

// IR is the intermediate representation for code generation.
type IR struct {
	PackageName string
	Entities    []EntityIR
	Services    []ServiceIR
}

type EntityIR struct {
	Name       string
	PascalName string
	KeyType    string
	Create     *CreateIR
	Delete     *DeleteIR
	DeleteSoft *DeleteIR
	Resources  []ResourceIR
}

type CreateIR struct {
	RPCName          string
	RequestName      string
	ResponseName     string
	RequestFields    []FieldIR
	ResponseKeyField FieldIR
}

type DeleteIR struct {
	RPCName      string
	RequestName  string
	ResponseName string
	KeyField     FieldIR
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
}

type BatchGetIR struct {
	RPCName        string
	RequestName    string
	ResponseName   string
	KeysField      FieldIR
	ResourcesField FieldIR
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
}

type UpdateIR struct {
	RPCName       string
	RequestName   string
	ResponseName  string
	RequestFields []FieldIR
	HasVersion    bool
	VersionField  *FieldIR
	Mask          bool
}

type FieldIR struct {
	Name     string
	Type     string
	Number   int
	Repeated bool
}

type ServiceIR struct {
	Name          string
	ProtoPackage  string
	GoPackage     string
	Entities      []ServiceEntityIR
	CustomMethods []CustomMethodIR
}

type ServiceEntityIR struct {
	Name      string
	Resources []ResourceIR
}

type CustomMethodIR struct {
	Name     string
	Request  string
	Response string
}

// Build constructs the IR from parsed YAML config.
func Build(cfg *apigenyaml.Config) (*IR, error) {
	ir := &IR{PackageName: cfg.Name}
	for i := range cfg.Entities {
		entityIR, err := buildEntity(&cfg.Entities[i], cfg)
		if err != nil {
			return nil, err
		}
		ir.Entities = append(ir.Entities, *entityIR)
	}
	for i := range cfg.Services {
		ir.Services = append(ir.Services, buildService(&cfg.Services[i], cfg))
	}
	return ir, nil
}

func buildEntity(e *apigenyaml.Entity, cfg *apigenyaml.Config) (*EntityIR, error) {
	pascalName := toPascalCase(e.Name)
	entity := &EntityIR{
		Name:       e.Name,
		PascalName: pascalName,
		KeyType:    cfg.ResolveTypeName(e.Key.Type),
	}
	if e.Create != nil {
		entity.Create = buildCreate(pascalName, e.Resources, entity.KeyType, cfg)
	}
	if e.Delete != nil {
		entity.Delete = buildDelete(entity.KeyType, "Delete"+pascalName)
	}
	if e.DeleteSoft != nil {
		entity.DeleteSoft = buildDelete(entity.KeyType, "Delete"+pascalName+"Soft")
	}
	for i := range e.Resources {
		resource, err := buildResource(&e.Resources[i], pascalName, entity.KeyType, cfg)
		if err != nil {
			return nil, err
		}
		entity.Resources = append(entity.Resources, *resource)
	}
	return entity, nil
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

func buildService(s *apigenyaml.Service, cfg *apigenyaml.Config) ServiceIR {
	sir := ServiceIR{
		Name:         s.Name,
		ProtoPackage: cfg.Name + "." + toSnakeCase(s.Name),
		GoPackage:    toSnakeCase(s.Name),
	}
	for _, se := range s.Entities {
		sir.Entities = append(sir.Entities, ServiceEntityIR{Name: se.Name})
	}
	for _, cm := range s.CustomMethods {
		sir.CustomMethods = append(sir.CustomMethods, CustomMethodIR{Name: cm.Name, Request: cm.Request, Response: cm.Response})
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

func toSnakeCase(s string) string {
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
