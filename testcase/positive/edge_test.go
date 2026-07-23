package positive

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixtureEdgeDir returns the path to the edge fixture directory.
func fixtureEdgeDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	return filepath.Join(wd, "..", "fixtures", "edge")
}

// TestEdge_MultiEntityRPCs verifies that both entities (doc + tag) generate their own RPCs.
func TestEdge_MultiEntityRPCs(t *testing.T) {
	dir := fixtureEdgeDir(t)
	proto := mustReadFile(t, filepath.Join(dir, "generated", "proto", "edge_service", "edge_service.proto"))

	// doc entity RPCs
	docRPCs := []string{
		"rpc CreateDoc(",
		"rpc DeleteDoc(",
		"rpc GetDocMeta(",
		"rpc BatchGetDocMetas(",
		"rpc UpdateDocMeta(",
		"rpc GetDocContent(",
		"rpc UpdateDocContent(",
		"rpc GetDocLog(",
		"rpc UpdateDocLog(",
	}
	for _, rpc := range docRPCs {
		assertContains(t, "edge_service.proto", proto, rpc)
	}

	// tag entity RPCs
	tagRPCs := []string{
		"rpc CreateTag(",
		"rpc DeleteTag(",
		"rpc GetTagInfo(",
		"rpc UpdateTagInfo(",
	}
	for _, rpc := range tagRPCs {
		assertContains(t, "edge_service.proto", proto, rpc)
	}

	// custom method
	assertContains(t, "edge_service.proto", proto, "rpc ExportDoc(")
}

// TestEdge_CompositeKeyPathBinding verifies nested key type path segment binding.
func TestEdge_CompositeKeyPathBinding(t *testing.T) {
	dir := fixtureEdgeDir(t)
	proto := mustReadFile(t, filepath.Join(dir, "generated", "proto", "edge_service", "edge_service.proto"))

	// Composite key: DocId{OrgId org; int32 seq;} → {key.org.oid}/{key.org.qid}/{key.seq}
	assertContains(t, "edge_service.proto", proto, `{key.org.oid}/{key.org.qid}/{key.seq}`)

	// Simple key: TagId{string name;} → {key.name}
	assertContains(t, "edge_service.proto", proto, `{key.name}`)
}

// TestEdge_WEAKVersionWrapper verifies WEAK version generates google.protobuf.UInt64Value wrapper.
func TestEdge_WEAKVersionWrapper(t *testing.T) {
	dir := fixtureEdgeDir(t)
	proto := mustReadFile(t, filepath.Join(dir, "generated", "proto", "edge_service", "edge_service.proto"))

	// WEAK version on DocContent → UInt64Value wrapper in GetDocContentResponse and UpdateDocContentRequest
	assertContains(t, "edge_service.proto", proto, "google.protobuf.UInt64Value version")

	// STRONG version on DocMeta → plain uint64
	assertContains(t, "edge_service.proto", proto, "uint64 version")
}

// TestEdge_NONEVersionNoField verifies NONE version has no version field.
func TestEdge_NONEVersionNoField(t *testing.T) {
	dir := fixtureEdgeDir(t)
	proto := mustReadFile(t, filepath.Join(dir, "generated", "proto", "edge_service", "edge_service.proto"))

	// GetDocLogResponse (NONE version) should not have a version field
	idx := strings.Index(proto, "message GetDocLogResponse")
	if idx < 0 {
		t.Fatal("GetDocLogResponse message not found")
	}
	end := strings.Index(proto[idx:], "\n\n")
	if end < 0 {
		end = len(proto) - idx
	}
	block := proto[idx : idx+end]
	if strings.Contains(block, "version") {
		t.Errorf("GetDocLogResponse (NONE version) should not have version field, but contains: %s", block)
	}

	// UpdateDocLogRequest (NONE version) should not have a version field
	idx = strings.Index(proto, "message UpdateDocLogRequest")
	if idx < 0 {
		t.Fatal("UpdateDocLogRequest message not found")
	}
	end = strings.Index(proto[idx:], "\n\n")
	if end < 0 {
		end = len(proto) - idx
	}
	block = proto[idx : idx+end]
	if strings.Contains(block, "version") {
		t.Errorf("UpdateDocLogRequest (NONE version) should not have version field, but contains: %s", block)
	}
}

// TestEdge_STRONGVersionUpdateWithMask verifies STRONG version + mask:true includes FieldMask + version.
func TestEdge_STRONGVersionUpdateWithMask(t *testing.T) {
	dir := fixtureEdgeDir(t)
	proto := mustReadFile(t, filepath.Join(dir, "generated", "proto", "edge_service", "edge_service.proto"))

	// UpdateDocMetaRequest should have update_mask (mask: true) and version (STRONG)
	idx := strings.Index(proto, "message UpdateDocMetaRequest")
	if idx < 0 {
		t.Fatal("UpdateDocMetaRequest message not found")
	}
	end := strings.Index(proto[idx:], "\n\n")
	if end < 0 {
		end = len(proto) - idx
	}
	block := proto[idx : idx+end]
	assertContains(t, "UpdateDocMetaRequest", block, "google.protobuf.FieldMask update_mask")
	assertContains(t, "UpdateDocMetaRequest", block, "uint64 version")
}

// TestEdge_CrossPackageTypes verifies that different entities in the same api.yaml
// can reference key/resource types from different proto packages.
func TestEdge_CrossPackageTypes(t *testing.T) {
	dir := fixtureEdgeDir(t)
	proto := mustReadFile(t, filepath.Join(dir, "generated", "proto", "edge_service", "edge_service.proto"))

	t.Run("doc entity types from edge.example package", func(t *testing.T) {
		// doc entity key and resources should use edge.example.* types
		assertContains(t, "edge_service.proto", proto, "edge.example.DocId")
		assertContains(t, "edge_service.proto", proto, "edge.example.DocMeta")
		assertContains(t, "edge_service.proto", proto, "edge.example.DocContent")
		assertContains(t, "edge_service.proto", proto, "edge.example.DocLog")

		// doc entity should NOT reference edge.tag.* types
		// (CreateDocRequest should not contain edge.tag)
		idx := strings.Index(proto, "message CreateDocRequest")
		if idx < 0 {
			t.Fatal("CreateDocRequest message not found")
		}
		end := strings.Index(proto[idx:], "\n}")
		if end < 0 {
			end = 300
		}
		createDocBlock := proto[idx : idx+end]
		if strings.Contains(createDocBlock, "edge.tag.") {
			t.Errorf("CreateDocRequest should not reference edge.tag.* types, but found: %s", createDocBlock)
		}
	})

	t.Run("tag entity cross-package key and resources", func(t *testing.T) {
		// tag entity key (edge.tag.TagId) and info resource (edge.tag.Tag)
		// come from edge.tag package
		assertContains(t, "edge_service.proto", proto, "edge.tag.TagId")
		assertContains(t, "edge_service.proto", proto, "edge.tag.Tag")

		// tag entity meta resource (TagMeta) comes from edge.example package
		// — key and meta are from DIFFERENT packages, the key test case
		assertContains(t, "edge_service.proto", proto, "edge.example.TagMeta")

		// CreateTagRequest should contain both edge.example.TagMeta and edge.tag.Tag
		idx := strings.Index(proto, "message CreateTagRequest")
		if idx < 0 {
			t.Fatal("CreateTagRequest message not found")
		}
		end := strings.Index(proto[idx:], "\n}")
		if end < 0 {
			end = 300
		}
		createTagBlock := proto[idx : idx+end]
		if !strings.Contains(createTagBlock, "edge.example.TagMeta") {
			t.Errorf("CreateTagRequest should contain edge.example.TagMeta (cross-package meta), but not found in: %s", createTagBlock)
		}
		if !strings.Contains(createTagBlock, "edge.tag.Tag") {
			t.Errorf("CreateTagRequest should contain edge.tag.Tag (same-package resource), but not found in: %s", createTagBlock)
		}
	})

	t.Run("service proto imports both packages", func(t *testing.T) {
		assertContains(t, "edge_service.proto", proto, `import "edge.proto"`)
		assertContains(t, "edge_service.proto", proto, `import "edge/tag/tag.proto"`)
	})

	t.Run("doc and tag types coexist without conflict", func(t *testing.T) {
		// Verify both type prefixes appear in the same service proto
		assertContains(t, "edge_service.proto", proto, "edge.example.")
		assertContains(t, "edge_service.proto", proto, "edge.tag.")
	})
}
