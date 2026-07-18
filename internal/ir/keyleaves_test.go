package ir

import (
	"strings"
	"testing"

	// Side-effect imports to register Well-Known Types (WKT) in the global
	// proto registry, so that protodesc.NewFile can resolve
	// google.protobuf.* references (e.g. Timestamp, wrappers) during tests.
	_ "google.golang.org/protobuf/types/known/timestamppb"

	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

// strPtr returns a pointer to s.
func strPtr(s string) *string { return &s }

// int32Ptr returns a pointer to i.
func int32Ptr(i int32) *int32 { return &i }

// typePtr returns a pointer to t.
func typePtr(t descriptorpb.FieldDescriptorProto_Type) *descriptorpb.FieldDescriptorProto_Type {
	return &t
}

// labelPtr returns a pointer to l.
func labelPtr(l descriptorpb.FieldDescriptorProto_Label) *descriptorpb.FieldDescriptorProto_Label {
	return &l
}

// mkFile builds a FileDescriptorProto with the given package and messages,
// resolving message-type field references against the provided message names
// (local) and WKT names (global, via protoregistry.GlobalFiles).
func mkFile(pkg string, msgs ...*descriptorpb.DescriptorProto) protoreflect.FileDescriptor {
	return mkFileWithImports(pkg, nil, msgs...)
}

// mkFileWithImports is like mkFile but also adds the given import paths
// (e.g. "google/protobuf/timestamp.proto") to the file descriptor so that
// WKT message-type references can be resolved by protodesc.NewFile.
func mkFileWithImports(pkg string, imports []string, msgs ...*descriptorpb.DescriptorProto) protoreflect.FileDescriptor {
	fd := &descriptorpb.FileDescriptorProto{
		Syntax:      strPtr("proto3"),
		Name:        strPtr("test.proto"),
		Package:     strPtr(pkg),
		MessageType: msgs,
	}
	for _, imp := range imports {
		fd.Dependency = append(fd.Dependency, imp)
	}
	// Build using GlobalFiles as the resolver so WKT (google.protobuf.*)
	// references can be resolved.
	desc, err := protodesc.NewFile(fd, protoregistry.GlobalFiles)
	if err != nil {
		panic(err)
	}
	return desc
}

// mkMsg builds a DescriptorProto with the given name and fields.
func mkMsg(name string, fields ...*descriptorpb.FieldDescriptorProto) *descriptorpb.DescriptorProto {
	return &descriptorpb.DescriptorProto{
		Name:  strPtr(name),
		Field: fields,
	}
}

// mkScalarField builds a scalar field descriptor.
func mkScalarField(name, typ string, num int32) *descriptorpb.FieldDescriptorProto {
	typeMap := map[string]descriptorpb.FieldDescriptorProto_Type{
		"int32":  descriptorpb.FieldDescriptorProto_TYPE_INT32,
		"int64":  descriptorpb.FieldDescriptorProto_TYPE_INT64,
		"uint32": descriptorpb.FieldDescriptorProto_TYPE_UINT32,
		"uint64": descriptorpb.FieldDescriptorProto_TYPE_UINT64,
		"string": descriptorpb.FieldDescriptorProto_TYPE_STRING,
		"bool":   descriptorpb.FieldDescriptorProto_TYPE_BOOL,
		"bytes":  descriptorpb.FieldDescriptorProto_TYPE_BYTES,
		"float":  descriptorpb.FieldDescriptorProto_TYPE_FLOAT,
		"double": descriptorpb.FieldDescriptorProto_TYPE_DOUBLE,
	}
	ft, ok := typeMap[typ]
	if !ok {
		panic("unknown scalar type: " + typ)
	}
	return &descriptorpb.FieldDescriptorProto{
		Name:   strPtr(name),
		Number: int32Ptr(num),
		Label:  labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
		Type:   typePtr(ft),
	}
}

// mkMsgField builds a message-typed field descriptor. `fullTypeName` should be
// the fully-qualified message name without leading dot (e.g.
// "test.Org" or "google.protobuf.Timestamp").
func mkMsgField(name, fullTypeName string, num int32) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:     strPtr(name),
		Number:   int32Ptr(num),
		Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
		Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
		TypeName: strPtr("." + fullTypeName),
	}
}

// mkRepeatedField builds a repeated scalar field descriptor.
func mkRepeatedScalarField(name, typ string, num int32) *descriptorpb.FieldDescriptorProto {
	f := mkScalarField(name, typ, num)
	f.Label = labelPtr(descriptorpb.FieldDescriptorProto_LABEL_REPEATED)
	return f
}

// mkMapField builds a real proto map field. A map field is represented as a
// repeated message with the map_entry option set to true, where the nested
// message has key=1 and value=2 fields. The entry message name must be
// <FieldName capitalized>Entry per proto map naming convention.
func mkMapField(parent *descriptorpb.DescriptorProto, name string, num int32) *descriptorpb.FieldDescriptorProto {
	// Capitalize first letter for the entry name (proto convention).
	entryName := strings.ToUpper(name[:1]) + name[1:] + "Entry"
	entry := &descriptorpb.DescriptorProto{
		Name: strPtr(entryName),
		Field: []*descriptorpb.FieldDescriptorProto{
			{
				Name:   strPtr("key"),
				Number: int32Ptr(1),
				Label:  labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
				Type:   typePtr(descriptorpb.FieldDescriptorProto_TYPE_STRING),
			},
			{
				Name:   strPtr("value"),
				Number: int32Ptr(2),
				Label:  labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
				Type:   typePtr(descriptorpb.FieldDescriptorProto_TYPE_STRING),
			},
		},
		Options: &descriptorpb.MessageOptions{
			MapEntry: boolPtr(true),
		},
	}
	parent.NestedType = append(parent.NestedType, entry)
	return &descriptorpb.FieldDescriptorProto{
		Name:     strPtr(name),
		Number:   int32Ptr(num),
		Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_REPEATED),
		Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
		TypeName: strPtr(".test." + *parent.Name + "." + entryName),
	}
}

// boolPtr returns a pointer to b.
func boolPtr(b bool) *bool { return &b }

// mkOptionalScalarField builds a proto3 optional scalar field (with
// Proto3Optional=true and a synthetic oneof).
func mkOptionalScalarField(name, typ string, num int32) *descriptorpb.FieldDescriptorProto {
	f := mkScalarField(name, typ, num)
	t := true
	f.Proto3Optional = &t
	return f
}

// mkOneofField builds a field that belongs to a oneof (oneof index 0).
func mkOneofField(name, typ string, num int32) *descriptorpb.FieldDescriptorProto {
	f := mkScalarField(name, typ, num)
	zero := int32(0)
	f.OneofIndex = &zero
	return f
}

// withOneofs attaches a single oneof declaration to a message, so that
// oneof-indexed fields resolve correctly.
func withOneof(m *descriptorpb.DescriptorProto, oneofName string) *descriptorpb.DescriptorProto {
	m.OneofDecl = append(m.OneofDecl, &descriptorpb.OneofDescriptorProto{Name: strPtr(oneofName)})
	return m
}

// --- Tests for key leaf extraction (task 2.1: scalar leaf recursion) ---

// TestKeyLeaves_SimpleKey: message BookId { string id = 1; } → 1 leaf "id".
func TestKeyLeaves_SimpleKey(t *testing.T) {
	bookId := mkMsg("BookId",
		mkScalarField("id", "string", 1),
	)
	fd := mkFile("test", bookId)
	leaves, err := ExtractKeyLeaves(fd.Messages().Get(0))
	if err != nil {
		t.Fatalf("ExtractKeyLeaves failed: %v", err)
	}
	if len(leaves) != 1 {
		t.Fatalf("expected 1 leaf, got %d: %+v", len(leaves), leaves)
	}
	if leaves[0].DotPath != "id" {
		t.Errorf("expected DotPath 'id', got %q", leaves[0].DotPath)
	}
	if leaves[0].FieldType != "string" {
		t.Errorf("expected FieldType 'string', got %q", leaves[0].FieldType)
	}
}

// TestKeyLeaves_CompositeKey: BookId{Org org=1; int32 id=2;} + Org{string oid=1; int32 qid=2;}
// → depth-first leaves: org.oid, org.qid, id.
func TestKeyLeaves_CompositeKey(t *testing.T) {
	org := mkMsg("Org",
		mkScalarField("oid", "string", 1),
		mkScalarField("qid", "int32", 2),
	)
	bookId := mkMsg("BookId",
		mkMsgField("org", "test.Org", 1),
		mkScalarField("id", "int32", 2),
	)
	fd := mkFile("test", bookId, org)
	leaves, err := ExtractKeyLeaves(fd.Messages().Get(0))
	if err != nil {
		t.Fatalf("ExtractKeyLeaves failed: %v", err)
	}
	expected := []string{"org.oid", "org.qid", "id"}
	if len(leaves) != len(expected) {
		t.Fatalf("expected %d leaves, got %d: %+v", len(expected), len(leaves), leaves)
	}
	for i, e := range expected {
		if leaves[i].DotPath != e {
			t.Errorf("leaf[%d]: expected DotPath %q, got %q", i, e, leaves[i].DotPath)
		}
	}
}

// TestKeyLeaves_WKTOpaqueLeaf: BookId{google.protobuf.Timestamp created_at=1; string id=2;}
// → WKT Timestamp treated as opaque single leaf "created_at", not recursed.
func TestKeyLeaves_WKTOpaqueLeaf(t *testing.T) {
	bookId := mkMsg("BookId",
		mkMsgField("created_at", "google.protobuf.Timestamp", 1),
		mkScalarField("id", "string", 2),
	)
	fd := mkFileWithImports("test", []string{"google/protobuf/timestamp.proto"}, bookId)
	leaves, err := ExtractKeyLeaves(fd.Messages().Get(0))
	if err != nil {
		t.Fatalf("ExtractKeyLeaves failed: %v", err)
	}
	expected := []string{"created_at", "id"}
	if len(leaves) != len(expected) {
		t.Fatalf("expected %d leaves (WKT opaque), got %d: %+v", len(expected), len(leaves), leaves)
	}
	for i, e := range expected {
		if leaves[i].DotPath != e {
			t.Errorf("leaf[%d]: expected DotPath %q, got %q", i, e, leaves[i].DotPath)
		}
	}
}

// TestKeyLeaves_OptionalScalar: optional string id = 1 → treated as normal leaf.
func TestKeyLeaves_OptionalScalar(t *testing.T) {
	bookId := mkMsg("BookId",
		mkOptionalScalarField("id", "string", 1),
	)
	fd := mkFile("test", bookId)
	leaves, err := ExtractKeyLeaves(fd.Messages().Get(0))
	if err != nil {
		t.Fatalf("ExtractKeyLeaves failed: %v", err)
	}
	if len(leaves) != 1 {
		t.Fatalf("expected 1 leaf, got %d", len(leaves))
	}
	if leaves[0].DotPath != "id" {
		t.Errorf("expected DotPath 'id', got %q", leaves[0].DotPath)
	}
}

// TestKeyLeaves_DotPathFormat: nested message → dot path "org.oid".
func TestKeyLeaves_DotPathFormat(t *testing.T) {
	org := mkMsg("Org",
		mkScalarField("oid", "string", 1),
	)
	bookId := mkMsg("BookId",
		mkMsgField("org", "test.Org", 1),
	)
	fd := mkFile("test", bookId, org)
	leaves, err := ExtractKeyLeaves(fd.Messages().Get(0))
	if err != nil {
		t.Fatalf("ExtractKeyLeaves failed: %v", err)
	}
	if len(leaves) != 1 {
		t.Fatalf("expected 1 leaf, got %d", len(leaves))
	}
	if leaves[0].DotPath != "org.oid" {
		t.Errorf("expected DotPath 'org.oid', got %q", leaves[0].DotPath)
	}
}

// TestKeyLeaves_DeepNesting: 3-level nesting → depth-first dot paths.
// A{B b=1;} B{C c=1;} C{string v=1;} → leaf "b.c.v".
func TestKeyLeaves_DeepNesting(t *testing.T) {
	c := mkMsg("C", mkScalarField("v", "string", 1))
	b := mkMsg("B", mkMsgField("c", "test.C", 1))
	a := mkMsg("A", mkMsgField("b", "test.B", 1))
	fd := mkFile("test", a, b, c)
	leaves, err := ExtractKeyLeaves(fd.Messages().Get(0))
	if err != nil {
		t.Fatalf("ExtractKeyLeaves failed: %v", err)
	}
	if len(leaves) != 1 {
		t.Fatalf("expected 1 leaf, got %d", len(leaves))
	}
	if leaves[0].DotPath != "b.c.v" {
		t.Errorf("expected DotPath 'b.c.v', got %q", leaves[0].DotPath)
	}
}

// TestKeyLeaves_MultipleScalarsAtRoot: BookId{string a=1; int32 b=2; uint64 c=3;}
// → 3 leaves a, b, c in field order.
func TestKeyLeaves_MultipleScalarsAtRoot(t *testing.T) {
	bookId := mkMsg("BookId",
		mkScalarField("a", "string", 1),
		mkScalarField("b", "int32", 2),
		mkScalarField("c", "uint64", 3),
	)
	fd := mkFile("test", bookId)
	leaves, err := ExtractKeyLeaves(fd.Messages().Get(0))
	if err != nil {
		t.Fatalf("ExtractKeyLeaves failed: %v", err)
	}
	expected := []string{"a", "b", "c"}
	if len(leaves) != len(expected) {
		t.Fatalf("expected %d leaves, got %d", len(expected), len(leaves))
	}
	for i, e := range expected {
		if leaves[i].DotPath != e {
			t.Errorf("leaf[%d]: expected %q, got %q", i, e, leaves[i].DotPath)
		}
	}
}

// --- Fail-fast tests (task 2.2) ---

// TestKeyLeaves_FailFast_Repeated: repeated field → error.
func TestKeyLeaves_FailFast_Repeated(t *testing.T) {
	bookId := mkMsg("BookId",
		mkRepeatedScalarField("tags", "string", 1),
		mkScalarField("id", "string", 2),
	)
	fd := mkFile("test", bookId)
	_, err := ExtractKeyLeaves(fd.Messages().Get(0))
	if err == nil {
		t.Fatal("ExtractKeyLeaves should fail-fast for repeated field")
	}
	if !strings.Contains(err.Error(), "repeated") {
		t.Errorf("error should mention 'repeated', got: %v", err)
	}
}

// TestKeyLeaves_FailFast_Map: map field → error.
func TestKeyLeaves_FailFast_Map(t *testing.T) {
	bookId := mkMsg("BookId",
		mkScalarField("id", "string", 2),
	)
	// Add map field (needs parent reference for nested map entry).
	bookId.Field = append([]*descriptorpb.FieldDescriptorProto{
		mkMapField(bookId, "labels", 1),
	}, bookId.Field...)
	fd := mkFile("test", bookId)
	_, err := ExtractKeyLeaves(fd.Messages().Get(0))
	if err == nil {
		t.Fatal("ExtractKeyLeaves should fail-fast for map field")
	}
	if !strings.Contains(err.Error(), "map") {
		t.Errorf("error should mention 'map', got: %v", err)
	}
}

// TestKeyLeaves_FailFast_Oneof: oneof field → error.
func TestKeyLeaves_FailFast_Oneof(t *testing.T) {
	bookId := withOneof(mkMsg("BookId",
		mkOneofField("sid", "string", 1),
		mkOneofField("iid", "int32", 2),
	), "id")
	fd := mkFile("test", bookId)
	_, err := ExtractKeyLeaves(fd.Messages().Get(0))
	if err == nil {
		t.Fatal("ExtractKeyLeaves should fail-fast for oneof field")
	}
	if !strings.Contains(err.Error(), "oneof") {
		t.Errorf("error should mention 'oneof', got: %v", err)
	}
}

// TestKeyLeaves_FailFast_CircularRef: A → B → A → error.
// message A { B b = 1; } message B { A a = 1; }
func TestKeyLeaves_FailFast_CircularRef(t *testing.T) {
	a := mkMsg("A", mkMsgField("b", "test.B", 1))
	b := mkMsg("B", mkMsgField("a", "test.A", 1))
	fd := mkFile("test", a, b)
	_, err := ExtractKeyLeaves(fd.Messages().Get(0))
	if err == nil {
		t.Fatal("ExtractKeyLeaves should fail-fast for circular reference")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("error should mention 'circular', got: %v", err)
	}
}

// TestKeyLeaves_NoCycleNested: nested message without cycle → no error.
// message A { B b = 1; } message B { string v = 1; }
func TestKeyLeaves_NoCycleNested(t *testing.T) {
	b := mkMsg("B", mkScalarField("v", "string", 1))
	a := mkMsg("A", mkMsgField("b", "test.B", 1))
	fd := mkFile("test", a, b)
	leaves, err := ExtractKeyLeaves(fd.Messages().Get(0))
	if err != nil {
		t.Fatalf("ExtractKeyLeaves should succeed for nested (no cycle): %v", err)
	}
	if len(leaves) != 1 || leaves[0].DotPath != "b.v" {
		t.Errorf("expected 1 leaf 'b.v', got %+v", leaves)
	}
}
