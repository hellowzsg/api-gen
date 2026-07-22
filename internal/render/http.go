package render

import (
	"fmt"
	"strings"

	"github.com/hellowzsg/api-gen/internal/ir"
)

// httpRenderContext carries the per-service values needed to resolve HTTP
// annotation paths at render time.
type httpRenderContext struct {
	prefix  string
	svcName string
}

// RenderHTTPAnnotation renders a google.api.http option from an
// HTTPAnnotation, resolving the final path for the given prefix and service
// name. Returns empty string if ann is nil (HTTP disabled or method has no
// HTTP binding).
//
// Format (proto):
//
//	option (google.api.http) = { <verb>: "<path>" [body: "<body>"] };
//
// Examples:
//   - POST with body:  option (google.api.http) = { post: "/api/svc/ent" body: "*" };
//   - GET no body:     option (google.api.http) = { get: "/api/svc/ent/{key.id}" };
func RenderHTTPAnnotation(ann *ir.HTTPAnnotation, prefix, svcName string) string {
	if ann == nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`option (google.api.http) = { %s: %q`, strings.ToLower(ann.Verb), ann.ResolvePath(prefix, svcName)))
	if ann.Body != "" {
		sb.WriteString(fmt.Sprintf(` body: %q`, ann.Body))
	}
	sb.WriteString(" };")
	return sb.String()
}

// renderRPCWithHTTP writes an RPC declaration with an optional google.api.http
// annotation indented inside the RPC body.
func renderRPCWithHTTP(sb *strings.Builder, rpcName, reqType, respType string, ann *ir.HTTPAnnotation, hctx httpRenderContext) {
	if ann == nil {
		sb.WriteString(fmt.Sprintf("  rpc %s(%s) returns (%s);\n", rpcName, reqType, respType))
		return
	}
	sb.WriteString(fmt.Sprintf("  rpc %s(%s) returns (%s) {\n", rpcName, reqType, respType))
	sb.WriteString(fmt.Sprintf("    %s\n", RenderHTTPAnnotation(ann, hctx.prefix, hctx.svcName)))
	sb.WriteString("  }\n")
}
