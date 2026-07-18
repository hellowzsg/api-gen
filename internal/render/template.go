// Package render renders the IR into proto files.
package render

import (
	"fmt"
	"sort"
	"strings"

	"github.com/acme/apigen/internal/ir"
)

// RenderServiceProto renders a service proto file from IR.
func RenderServiceProto(irData *ir.IR, svc ir.ServiceIR) (string, error) {
	var sb strings.Builder
	var entities []ir.EntityIR
	for _, se := range svc.Entities {
		for i := range irData.Entities {
			if irData.Entities[i].Name == se.Name {
				entities = append(entities, irData.Entities[i])
				break
			}
		}
	}
	needEmpty, needMask, needWrapper := analyzeImports(entities)
	typeImports := collectTypeImports(entities)
	imports := generateImports(needEmpty, needMask, needWrapper, false, typeImports)
	exemptions := generateExemptions(entities)
	renderExemptions(&sb, exemptions)
	sb.WriteString(`syntax = "proto3";` + "\n")
	sb.WriteString(fmt.Sprintf("package %s;\n\n", svc.ProtoPackage))
	for _, imp := range imports {
		sb.WriteString(fmt.Sprintf("import %q;\n", imp))
	}
	if len(imports) > 0 {
		sb.WriteString("\n")
	}
	sb.WriteString(fmt.Sprintf("option go_package = \".../generated/go/%s;%s\";\n\n", svc.GoPackage, svc.GoPackage))
	sb.WriteString(fmt.Sprintf("service %s {\n", svc.Name))
	for _, e := range entities {
		renderServiceRPCs(&sb, &e)
	}
	for _, cm := range svc.CustomMethods {
		sb.WriteString(fmt.Sprintf("  rpc %s(%s) returns (%s);\n", cm.Name, cm.Request, cm.Response))
	}
	sb.WriteString("}\n\n")
	for _, e := range entities {
		renderMessages(&sb, &e)
	}
	return sb.String(), nil
}

func renderServiceRPCs(sb *strings.Builder, e *ir.EntityIR) {
	if e.Create != nil {
		sb.WriteString(fmt.Sprintf("  rpc %s(%s) returns (%s);\n", e.Create.RPCName, e.Create.RequestName, e.Create.ResponseName))
	}
	if e.Delete != nil {
		sb.WriteString(fmt.Sprintf("  rpc %s(%s) returns (%s);\n", e.Delete.RPCName, e.Delete.RequestName, e.Delete.ResponseName))
	}
	if e.DeleteSoft != nil {
		sb.WriteString(fmt.Sprintf("  rpc %s(%s) returns (%s);\n", e.DeleteSoft.RPCName, e.DeleteSoft.RequestName, e.DeleteSoft.ResponseName))
	}
	for _, r := range e.Resources {
		if r.Get != nil {
			sb.WriteString(fmt.Sprintf("  rpc %s(%s) returns (%s);\n", r.Get.RPCName, r.Get.RequestName, r.Get.ResponseName))
		}
		if r.BatchGet != nil {
			sb.WriteString(fmt.Sprintf("  rpc %s(%s) returns (%s);\n", r.BatchGet.RPCName, r.BatchGet.RequestName, r.BatchGet.ResponseName))
		}
		if r.List != nil {
			sb.WriteString(fmt.Sprintf("  rpc %s(%s) returns (%s);\n", r.List.RPCName, r.List.RequestName, r.List.ResponseName))
		}
		if r.Update != nil {
			sb.WriteString(fmt.Sprintf("  rpc %s(%s) returns (%s);\n", r.Update.RPCName, r.Update.RequestName, r.Update.ResponseName))
		}
	}
}

func renderMessages(sb *strings.Builder, e *ir.EntityIR) {
	if e.Create != nil {
		sb.WriteString(fmt.Sprintf("message %s {\n", e.Create.RequestName))
		for _, f := range e.Create.RequestFields {
			sb.WriteString(fmt.Sprintf("  %s %s = %d;\n", f.Type, f.Name, f.Number))
		}
		sb.WriteString("}\n")
		sb.WriteString(fmt.Sprintf("message %s { %s key = %d; }\n\n", e.Create.ResponseName, e.Create.ResponseKeyField.Type, e.Create.ResponseKeyField.Number))
	}
	if e.Delete != nil {
		sb.WriteString(fmt.Sprintf("message %s { %s key = %d; }\n", e.Delete.RequestName, e.Delete.KeyField.Type, e.Delete.KeyField.Number))
	}
	if e.DeleteSoft != nil {
		sb.WriteString(fmt.Sprintf("message %s { %s key = %d; }\n", e.DeleteSoft.RequestName, e.DeleteSoft.KeyField.Type, e.DeleteSoft.KeyField.Number))
	}
	for _, r := range e.Resources {
		if r.Get != nil {
			sb.WriteString(fmt.Sprintf("message %s { %s key = %d; }\n", r.Get.RequestName, r.Get.KeyField.Type, r.Get.KeyField.Number))
			sb.WriteString(fmt.Sprintf("message %s {\n", r.Get.ResponseName))
			sb.WriteString(fmt.Sprintf("  %s %s = %d;\n", r.Get.ResourceField.Type, r.Get.ResourceField.Name, r.Get.ResourceField.Number))
			if r.Get.VersionField != nil {
				sb.WriteString(fmt.Sprintf("  %s version = %d;\n", r.Get.VersionField.Type, r.Get.VersionField.Number))
			}
			sb.WriteString("}\n")
		}
		if r.BatchGet != nil {
			sb.WriteString(fmt.Sprintf("message %s { repeated %s keys = %d; }\n", r.BatchGet.RequestName, r.BatchGet.KeysField.Type, r.BatchGet.KeysField.Number))
			sb.WriteString(fmt.Sprintf("message %s { repeated %s %s = %d; }\n", r.BatchGet.ResponseName, r.BatchGet.ResourcesField.Type, r.BatchGet.ResourcesField.Name, r.BatchGet.ResourcesField.Number))
		}
		if r.List != nil {
			sb.WriteString(fmt.Sprintf("message %s {\n", r.List.RequestName))
			sb.WriteString(fmt.Sprintf("  int32 page_size = %d;\n", r.List.PageSize.Number))
			sb.WriteString(fmt.Sprintf("  string page_token = %d;\n", r.List.PageToken.Number))
			sb.WriteString(fmt.Sprintf("  string filter = %d;\n", r.List.Filter.Number))
			sb.WriteString(fmt.Sprintf("  string order_by = %d;\n", r.List.OrderBy.Number))
			sb.WriteString("}\n")
			sb.WriteString(fmt.Sprintf("message %s {\n", r.List.ResponseName))
			sb.WriteString(fmt.Sprintf("  repeated %s %s = %d;\n", r.List.ResourcesField.Type, r.List.ResourcesField.Name, r.List.ResourcesField.Number))
			sb.WriteString(fmt.Sprintf("  string next_page_token = %d;\n", r.List.NextPageToken.Number))
			if r.List.TotalSize != nil {
				sb.WriteString(fmt.Sprintf("  int32 total_size = %d;\n", r.List.TotalSize.Number))
			}
			sb.WriteString("}\n")
		}
		if r.Update != nil {
			sb.WriteString(fmt.Sprintf("message %s {\n", r.Update.RequestName))
			for _, f := range r.Update.RequestFields {
				sb.WriteString(fmt.Sprintf("  %s %s = %d;\n", f.Type, f.Name, f.Number))
			}
			sb.WriteString("}\n")
			if r.Update.ResponseName != "google.protobuf.Empty" && r.Update.VersionField != nil {
				sb.WriteString(fmt.Sprintf("message %s { %s version = %d; }\n", r.Update.ResponseName, r.Update.VersionField.Type, r.Update.VersionField.Number))
			}
		}
	}
	sb.WriteString("\n")
}

func analyzeImports(entities []ir.EntityIR) (needEmpty, needMask, needWrapper bool) {
	for _, e := range entities {
		if e.Delete != nil || e.DeleteSoft != nil {
			needEmpty = true
		}
		for _, r := range e.Resources {
			if r.Update != nil {
				if r.Update.Mask {
					needMask = true
				}
				if r.Update.ResponseName == "google.protobuf.Empty" {
					needEmpty = true
				}
			}
			if r.Version.IsWrapper {
				needWrapper = true
			}
		}
	}
	return
}

func collectTypeImports(entities []ir.EntityIR) []string {
	seen := make(map[string]bool)
	var imports []string
	addImport := func(typeName string) {
		if strings.Contains(typeName, ".") {
			parts := strings.Split(typeName, ".")
			if len(parts) >= 2 {
				pkg := strings.Join(parts[:len(parts)-1], "/")
				imp := pkg + ".proto"
				if !seen[imp] && !strings.HasPrefix(imp, "google.protobuf") && !strings.HasPrefix(imp, "google/api") {
					seen[imp] = true
					imports = append(imports, imp)
				}
			}
		}
	}
	for _, e := range entities {
		addImport(e.KeyType)
		for _, r := range e.Resources {
			addImport(r.Type)
		}
	}
	return imports
}

func generateImports(needEmpty, needMask, needWrapper, needHTTP bool, typeImports []string) []string {
	var imports []string
	if needEmpty {
		imports = append(imports, "google/protobuf/empty.proto")
	}
	if needMask {
		imports = append(imports, "google/protobuf/field_mask.proto")
	}
	if needWrapper {
		imports = append(imports, "google/protobuf/wrappers.proto")
	}
	if needHTTP {
		imports = append(imports, "google/api/annotations.proto")
	}
	imports = append(imports, typeImports...)
	sort.Strings(imports)
	return imports
}

func generateExemptions(entities []ir.EntityIR) []string {
	var exemptions []string
	hasCreate, hasDelete, hasDeleteSoft, hasGet, hasBatchGet, hasUpdate := false, false, false, false, false, false
	for _, e := range entities {
		if e.Create != nil { hasCreate = true }
		if e.Delete != nil { hasDelete = true }
		if e.DeleteSoft != nil { hasDeleteSoft = true }
		for _, r := range e.Resources {
			if r.Get != nil { hasGet = true }
			if r.BatchGet != nil { hasBatchGet = true }
			if r.Update != nil { hasUpdate = true }
		}
	}
	if hasGet {
		exemptions = append(exemptions, "core::0131::response-message-name", "core::0131::request-name-field")
	}
	if hasCreate {
		exemptions = append(exemptions, "core::0133::response-message-name", "core::0133::request-parent-field", "core::0133::field-numbers")
	}
	if hasUpdate {
		exemptions = append(exemptions, "core::0134::response-message-name", "core::0134::request-unknown-fields")
	}
	if hasDelete || hasDeleteSoft {
		exemptions = append(exemptions, "core::0135::request-name-field")
	}
	if hasBatchGet {
		exemptions = append(exemptions, "core::0231::response-message-name", "core::0231::method-name")
	}
	return exemptions
}

func renderExemptions(sb *strings.Builder, exemptions []string) {
	for _, e := range exemptions {
		sb.WriteString(fmt.Sprintf("// (-- api-linter: %s=disabled --)\n", e))
	}
}
