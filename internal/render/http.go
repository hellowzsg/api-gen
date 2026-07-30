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
// annotation indented inside the RPC body. streamMode controls the `stream`
// keyword placement: "" (unary), "server" (stream response), "client"
// (stream request), "bidi" (stream both request and response).
func renderRPCWithHTTP(sb *strings.Builder, rpcName, reqType, respType, streamMode string, ann *ir.HTTPAnnotation, hctx httpRenderContext) {
	reqSide := reqType
	respSide := respType
	switch streamMode {
	case "server":
		respSide = "stream " + respType
	case "client":
		reqSide = "stream " + reqType
	case "bidi":
		reqSide = "stream " + reqType
		respSide = "stream " + respType
	}
	if ann == nil {
		sb.WriteString(fmt.Sprintf("  rpc %s(%s) returns (%s);\n", rpcName, reqSide, respSide))
		return
	}
	sb.WriteString(fmt.Sprintf("  rpc %s(%s) returns (%s) {\n", rpcName, reqSide, respSide))
	sb.WriteString(fmt.Sprintf("    %s\n", RenderHTTPAnnotation(ann, hctx.prefix, hctx.svcName)))
	sb.WriteString("  }\n")
}
