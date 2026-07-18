package ir

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// KeyLeaf represents a scalar leaf field in a key type's message tree,
// identified by its dot-separated path from the key message root.
// Example: for message BookId { Org org = 1; int32 id = 2; } with
// message Org { string oid = 1; }, the leaves are:
//   - {DotPath: "org.oid", FieldType: "string"}
//   - {DotPath: "id", FieldType: "int32"}
type KeyLeaf struct {
	// DotPath is the dot-separated field path from the key message root
	// to this scalar leaf (e.g. "org.oid").
	DotPath string
	// FieldType is the proto scalar type name (e.g. "string", "int32",
	// "uint64"). For WKT opaque leaves, this is the fully-qualified WKT
	// message name (e.g. "google.protobuf.Timestamp").
	FieldType string
}

// wktPrefix is the fully-qualified prefix for Well-Known Types.
// Messages whose full name starts with this prefix are treated as opaque
// leaves and are not recursed into during key-leaf extraction.
const wktPrefix = "google.protobuf."

// ExtractKeyLeaves recursively walks a key message type's field tree and
// extracts all scalar leaf fields in depth-first field-declaration order.
//
// Rules (design-v2.md §15.4):
//   - Scalar fields → leaf (record dot path + proto type name).
//   - Message fields:
//   - If the message full name has the "google.protobuf." prefix (WKT),
//     treat as an opaque leaf (do not recurse).
//   - Otherwise, recurse into the message and prefix child dot paths.
//   - optional scalar fields → treated as normal leaves.
//
// Fail-fast cases (return error):
//   - repeated fields → cannot bind to a single URL path segment.
//   - map fields → cannot bind to a single URL path segment.
//   - oneof fields → cannot bind to a single URL path segment.
//   - circular references (A → B → A) → infinite recursion guard.
//
// The field type name for scalars uses the proto source type spelling
// (e.g. "int32", "uint64", "string"). For WKT opaque leaves, the
// fully-qualified message name is used.
func ExtractKeyLeaves(keyMsg protoreflect.MessageDescriptor) ([]KeyLeaf, error) {
	var leaves []KeyLeaf
	visited := make(map[protoreflect.FullName]bool)
	if err := extractLeaves(keyMsg, "", &leaves, visited); err != nil {
		return nil, err
	}
	return leaves, nil
}

// extractLeaves is the recursive worker for ExtractKeyLeaves.
// prefix is the accumulated dot path from the root (e.g. "org").
// visited tracks message full names on the current recursion stack to
// detect cycles.
func extractLeaves(msg protoreflect.MessageDescriptor, prefix string, leaves *[]KeyLeaf, visited map[protoreflect.FullName]bool) error {
	fullName := msg.FullName()
	if visited[fullName] {
		return fmt.Errorf("circular reference detected at message %q (path prefix %q)", fullName, prefix)
	}
	visited[fullName] = true
	defer func() { delete(visited, fullName) }()

	fields := msg.Fields()
	for i := 0; i < fields.Len(); i++ {
		f := fields.Get(i)
		dotPath := string(f.Name())
		if prefix != "" {
			dotPath = prefix + "." + string(f.Name())
		}

		// Fail-fast: repeated fields cannot bind to a URL path segment.
		if f.IsList() {
			return fmt.Errorf("field %q is repeated; repeated fields cannot participate in HTTP path binding", dotPath)
		}
		// Fail-fast: map fields cannot bind to a URL path segment.
		if f.IsMap() {
			return fmt.Errorf("field %q is a map; map fields cannot participate in HTTP path binding", dotPath)
		}
		// Fail-fast: oneof fields cannot bind to a URL path segment.
		if f.ContainingOneof() != nil && !f.HasOptionalKeyword() {
			return fmt.Errorf("field %q is part of oneof %q; oneof fields cannot participate in HTTP path binding", dotPath, f.ContainingOneof().Name())
		}

		switch f.Kind() {
		case protoreflect.MessageKind:
			// Message field — check if WKT (opaque leaf) or recurse.
			msgName := string(f.Message().FullName())
			if strings.HasPrefix(msgName, wktPrefix) {
				// WKT: opaque leaf, do not recurse.
				*leaves = append(*leaves, KeyLeaf{
					DotPath:   dotPath,
					FieldType: msgName,
				})
			} else {
				// Non-WKT message: recurse.
				if err := extractLeaves(f.Message(), dotPath, leaves, visited); err != nil {
					return err
				}
			}
		case protoreflect.EnumKind:
			// Enum leaf — treated as a scalar-like leaf. We record the proto
			// enum type name so the renderer can emit the correct path
			// variable binding. Enums are valid URL path segments (they
			// serialize to their integer value).
			*leaves = append(*leaves, KeyLeaf{
				DotPath:   dotPath,
				FieldType: string(f.Enum().FullName()),
			})
		default:
			// Scalar leaf.
			*leaves = append(*leaves, KeyLeaf{
				DotPath:   dotPath,
				FieldType: scalarKindName(f.Kind()),
			})
		}
	}
	return nil
}

// scalarKindName maps a protoreflect.Kind to the proto source type spelling.
func scalarKindName(k protoreflect.Kind) string {
	switch k {
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return "int32"
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return "int64"
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return "uint32"
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return "uint64"
	case protoreflect.FloatKind:
		return "float"
	case protoreflect.DoubleKind:
		return "double"
	case protoreflect.BoolKind:
		return "bool"
	case protoreflect.StringKind:
		return "string"
	case protoreflect.BytesKind:
		return "bytes"
	default:
		return fmt.Sprintf("kind_%d", int(k))
	}
}
