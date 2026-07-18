package render

import (
	"fmt"
	"strings"

	"github.com/acme/apigen/internal/ir"
)

// RenderHTTPAnnotation renders a google.api.http option from an HTTPAnnotation.
// Returns empty string if ann is nil (HTTP disabled or method has no HTTP
// binding).
//
// Format (proto):
//
//	option (google.api.http) = { <verb>: "<path>" [body: "<body>"] };
//
// Examples:
//   - POST with body:  option (google.api.http) = { post: "/api/svc/ent" body: "*" };
//   - GET no body:     option (google.api.http) = { get: "/api/svc/ent/{key.id}" };
func RenderHTTPAnnotation(ann *ir.HTTPAnnotation) string {
	if ann == nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`option (google.api.http) = { %s: %q`, strings.ToLower(ann.Verb), ann.Path))
	if ann.Body != "" {
		sb.WriteString(fmt.Sprintf(` body: %q`, ann.Body))
	}
	sb.WriteString(" };")
	return sb.String()
}

// renderRPCWithHTTP writes an RPC declaration with an optional google.api.http
// annotation indented inside the RPC body.
func renderRPCWithHTTP(sb *strings.Builder, rpcName, reqType, respType string, ann *ir.HTTPAnnotation) {
	if ann == nil {
		sb.WriteString(fmt.Sprintf("  rpc %s(%s) returns (%s);\n", rpcName, reqType, respType))
		return
	}
	sb.WriteString(fmt.Sprintf("  rpc %s(%s) returns (%s) {\n", rpcName, reqType, respType))
	sb.WriteString(fmt.Sprintf("    %s\n", RenderHTTPAnnotation(ann)))
	sb.WriteString("  }\n")
}
