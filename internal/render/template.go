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
	// Build per-service, narrowed entity views: each service entity may
	// narrow the set of exposed resources and/or their reader/writer methods.
	var entities []ir.EntityIR
	for _, se := range svc.Entities {
		for i := range irData.Entities {
			if irData.Entities[i].Name == se.Name {
				entities = append(entities, narrowEntity(irData.Entities[i], se))
				break
			}
		}
	}
	// HTTP annotation paths are resolved per service from structured
	// segments (entity/keyLeaves/resource/suffix) — no string rewriting.
	hctx := httpRenderContext{prefix: irData.HTTPPrefix, svcName: svc.Name}
	needEmpty, needMask, needWrapper := analyzeImports(entities)
	needHTTP := irData.HTTPEnabled
	typeImports := collectTypeImports(entities, irData.TypeImportPaths)
	imports := generateImports(needEmpty, needMask, needWrapper, needHTTP, typeImports)
	exemptions := generateExemptions(entities, needHTTP)
	renderExemptions(&sb, exemptions)
	sb.WriteString(`syntax = "proto3";` + "\n")
	sb.WriteString(fmt.Sprintf("package %s;\n\n", svc.ProtoPackage))
	for _, imp := range imports {
		sb.WriteString(fmt.Sprintf("import %q;\n", imp))
	}
	if len(imports) > 0 {
		sb.WriteString("\n")
	}
	sb.WriteString(fmt.Sprintf("option go_package = %q;\n\n", buildGoPackage(svc)))
	sb.WriteString(fmt.Sprintf("service %s {\n", svc.Name))
	for i := range entities {
		renderServiceRPCs(&sb, &entities[i], hctx)
	}
	for _, cm := range svc.CustomMethods {
		renderRPCWithHTTP(&sb, cm.Name, cm.Request, cm.Response, cm.HTTPAnnotation, hctx)
	}
	sb.WriteString("}\n\n")
	for i := range entities {
		renderMessages(&sb, &entities[i])
	}
	return sb.String(), nil
}

// narrowEntity applies the service's resource narrowing rules (per design
// §十一):
//   - If se.Resources is empty, the entity's full resource set is inherited.
//   - If se.Resources is non-empty, only the listed resources are exposed.
//   - For each listed resource, if se's reader/writer block is present, it
//     overrides (narrows) the entity's reader/writer: a method is generated
//     only if the narrow spec enables it. When the narrow block is nil, the
//     entity's methods for that resource are inherited as-is.
//   - Entity-level methods (Create/Delete/DeleteSoft) are always inherited —
//     narrowing only applies to resource-level methods.
func narrowEntity(e ir.EntityIR, se ir.ServiceEntityIR) ir.EntityIR {
	out := ir.EntityIR{
		Name:       e.Name,
		PascalName: e.PascalName,
		KeyType:    e.KeyType,
		Create:     e.Create,
		Delete:     e.Delete,
		DeleteSoft: e.DeleteSoft,
	}
	if len(se.Resources) == 0 {
		out.Resources = e.Resources
		return out
	}
	// Build a name→narrow index.
	narrowByName := make(map[string]ir.ResourceNarrowIR, len(se.Resources))
	for _, nr := range se.Resources {
		narrowByName[nr.Name] = nr
	}
	for _, r := range e.Resources {
		narrow, ok := narrowByName[r.Name]
		if !ok {
			continue // resource not listed in the service → not exposed
		}
		filt := r // copy of ResourceIR
		// Apply reader narrowing.
		if narrow.Reader != nil {
			if narrow.Reader.Batch != nil && !*narrow.Reader.Batch {
				filt.BatchGet = nil
			}
			if narrow.Reader.List != nil && !*narrow.Reader.List {
				filt.List = nil
			}
			// When a reader block is present but neither batch nor list is
			// true, only the base Get remains (inherited). When Get should
			// also be narrowed off, a future flag can control it; for now
			// Get is inherited when reader block is present.
		}
		// Apply writer narrowing.
		if narrow.Writer != nil {
			if narrow.Writer.Update != nil && !*narrow.Writer.Update {
				filt.Update = nil
			}
		}
		out.Resources = append(out.Resources, filt)
	}
	return out
}

func renderServiceRPCs(sb *strings.Builder, e *ir.EntityIR, hctx httpRenderContext) {
	if e.Create != nil {
		renderRPCWithHTTP(sb, e.Create.RPCName, e.Create.RequestName, e.Create.ResponseName, e.Create.HTTPAnnotation, hctx)
	}
	if e.Delete != nil {
		renderRPCWithHTTP(sb, e.Delete.RPCName, e.Delete.RequestName, e.Delete.ResponseName, e.Delete.HTTPAnnotation, hctx)
	}
	if e.DeleteSoft != nil {
		renderRPCWithHTTP(sb, e.DeleteSoft.RPCName, e.DeleteSoft.RequestName, e.DeleteSoft.ResponseName, e.DeleteSoft.HTTPAnnotation, hctx)
	}
	for _, r := range e.Resources {
		if r.Get != nil {
			renderRPCWithHTTP(sb, r.Get.RPCName, r.Get.RequestName, r.Get.ResponseName, r.Get.HTTPAnnotation, hctx)
		}
		if r.BatchGet != nil {
			renderRPCWithHTTP(sb, r.BatchGet.RPCName, r.BatchGet.RequestName, r.BatchGet.ResponseName, r.BatchGet.HTTPAnnotation, hctx)
		}
		if r.List != nil {
			renderRPCWithHTTP(sb, r.List.RPCName, r.List.RequestName, r.List.ResponseName, r.List.HTTPAnnotation, hctx)
		}
		if r.Update != nil {
			renderRPCWithHTTP(sb, r.Update.RPCName, r.Update.RequestName, r.Update.ResponseName, r.Update.HTTPAnnotation, hctx)
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

// buildGoPackage derives the go_package option string for a service proto.
// Format: "<go_repo>/<out_go_dir>/<go_package>;<go_package>"
// e.g. "github.com/acme/demo-book/generated/go/library_service;library_service"
// Falls back to ".../<out>/<pkg>;<pkg>" when GoRepo is unset (e.g. in tests).
func buildGoPackage(svc ir.ServiceIR) string {
	pkg := svc.GoPackage
	if svc.GoRepo != "" {
		outDir := svc.OutGoDir
		if outDir == "" {
			outDir = "generated/go"
		}
		return svc.GoRepo + "/" + outDir + "/" + pkg + ";" + pkg
	}
	outDir := svc.OutGoDir
	if outDir == "" {
		outDir = "generated/go"
	}
	return ".../" + outDir + "/" + pkg + ";" + pkg
}

func collectTypeImports(entities []ir.EntityIR, resolved map[string]string) []string {
	seen := make(map[string]bool)
	var imports []string
	addImport := func(typeName string) {
		// Strip leading dot if present (fully-qualified form).
		tn := strings.TrimPrefix(typeName, ".")
		// Skip WKT — they are handled by needEmpty/needMask/needWrapper.
		if strings.HasPrefix(tn, "google.protobuf.") {
			return
		}
		var imp string
		if resolved != nil {
			// Use the exact file path from the resolved proto files.
			if p, ok := resolved[tn]; ok {
				imp = p
			}
		}
		if imp == "" {
			// Heuristic fallback (used in unit tests without a resolver):
			// pkg.subpkg.MessageName → pkg/subpkg/subpkg.proto
			// This assumes the conventional file-name == last-pkg-segment.
			if strings.Contains(tn, ".") {
				parts := strings.Split(tn, ".")
				if len(parts) >= 2 {
					pkgParts := parts[:len(parts)-1]
					lastPkg := pkgParts[len(pkgParts)-1]
					pkgPath := strings.Join(pkgParts, "/")
					imp = pkgPath + "/" + lastPkg + ".proto"
				}
			}
		}
		if imp != "" && !seen[imp] && !strings.HasPrefix(imp, "google/protobuf") && !strings.HasPrefix(imp, "google/api") {
			seen[imp] = true
			imports = append(imports, imp)
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

func generateExemptions(entities []ir.EntityIR, httpEnabled bool) []string {
	var exemptions []string
	hasCreate, hasDelete, hasDeleteSoft, hasGet, hasBatchGet, hasList, hasUpdate := false, false, false, false, false, false, false
	for _, e := range entities {
		if e.Create != nil { hasCreate = true }
		if e.Delete != nil { hasDelete = true }
		if e.DeleteSoft != nil { hasDeleteSoft = true }
		for _, r := range e.Resources {
			if r.Get != nil { hasGet = true }
			if r.BatchGet != nil { hasBatchGet = true }
			if r.List != nil { hasList = true }
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
	// HTTP-specific exemptions (only when HTTP is enabled).
	if httpEnabled {
		if hasCreate {
			// core::0133::http-body exemption is needed only when Create
			// uses body:"*" (wrapper style). When body_style: resource is
			// in effect (body = resource field name), the body binding is
			// a named field, not "*", so the exemption is not needed.
			createUsesWrapperBody := true
			for _, e := range entities {
				if e.Create != nil && e.Create.HTTPAnnotation != nil && e.Create.HTTPAnnotation.Body != "*" && e.Create.HTTPAnnotation.Body != "" {
					createUsesWrapperBody = false
					break
				}
			}
			if createUsesWrapperBody {
				exemptions = append(exemptions, "core::0133::http-body")
			}
		}
		if hasBatchGet {
			exemptions = append(exemptions, "core::0231::http-body", "core::0231::http-method")
		}
		if hasList {
			exemptions = append(exemptions, "core::0132::http-method", "core::0132::http-body")
		}
		if hasDeleteSoft {
			exemptions = append(exemptions, "core::0135::http-method", "core::0135::http-body")
		}
	}
	return exemptions
}

func renderExemptions(sb *strings.Builder, exemptions []string) {
	for _, e := range exemptions {
		sb.WriteString(fmt.Sprintf("// (-- api-linter: %s=disabled --)\n", e))
	}
}
