package positive

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixtureBookDir returns the absolute path to the book fixture directory.
func fixtureBookDir(t *testing.T) string {
	t.Helper()
	// The testcase module root is the parent of positive/
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	return filepath.Join(wd, "..", "fixtures", "book")
}

// mustReadFile reads a file and fails the test on error.
func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

// assertContains fails the test if s does not contain substr.
func assertContains(t *testing.T, name, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("%s: expected to contain %q, but it does not", name, substr)
	}
}

// assertNotContains fails the test if s contains substr.
func assertNotContains(t *testing.T, name, s, substr string) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Errorf("%s: expected NOT to contain %q, but it does", name, substr)
	}
}

// TestGenerateBookProtoStructure verifies that the book fixture's generated proto
// has the correct service, RPC methods, message fields, and AdminService narrowing.
func TestGenerateBookProtoStructure(t *testing.T) {
	dir := fixtureBookDir(t)

	t.Run("LibraryService proto has all 10 RPCs", func(t *testing.T) {
		proto := mustReadFile(t, filepath.Join(dir, "generated", "proto", "library_service", "library_service.proto"))
		rpcs := []string{
			"rpc CreateBook(",
			"rpc DeleteBook(",
			"rpc DeleteBookSoft(",
			"rpc GetBookMeta(",
			"rpc BatchGetBookMetas(",
			"rpc ListBookMetas(",
			"rpc UpdateBookMeta(",
			"rpc GetBookContent(",
			"rpc UpdateBookContent(",
			"rpc ArchiveBook(",
		}
		for _, rpc := range rpcs {
			assertContains(t, "library_service.proto", proto, rpc)
		}
	})

	t.Run("AdminService is narrowed (no BatchGet, no GetContent, no UpdateContent)", func(t *testing.T) {
		proto := mustReadFile(t, filepath.Join(dir, "generated", "proto", "admin_service", "admin_service.proto"))
		assertContains(t, "admin_service.proto", proto, "rpc ListBookMetas(")
		assertContains(t, "admin_service.proto", proto, "rpc GetBookMeta(")
		assertNotContains(t, "admin_service.proto", proto, "rpc BatchGetBookMetas(")
		assertNotContains(t, "admin_service.proto", proto, "rpc GetBookContent(")
		assertNotContains(t, "admin_service.proto", proto, "rpc UpdateBookContent(")
	})

	t.Run("STRONG version generates uint64 field", func(t *testing.T) {
		proto := mustReadFile(t, filepath.Join(dir, "generated", "proto", "library_service", "library_service.proto"))
		assertContains(t, "library_service.proto", proto, "uint64 version")
	})

	t.Run("NONE version has no version field in content response", func(t *testing.T) {
		proto := mustReadFile(t, filepath.Join(dir, "generated", "proto", "library_service", "library_service.proto"))
		// GetBookContentResponse should not have a version field
		assertContains(t, "library_service.proto", proto, "message GetBookContentResponse")
		// The content response should not contain "version" — check within the message block
		idx := strings.Index(proto, "message GetBookContentResponse")
		if idx < 0 {
			t.Fatal("GetBookContentResponse message not found")
		}
		end := strings.Index(proto[idx:], "\n\n")
		if end < 0 {
			end = len(proto) - idx
		}
		contentRespBlock := proto[idx : idx+end]
		if strings.Contains(contentRespBlock, "version") {
			t.Errorf("GetBookContentResponse should not have version field (NONE), but contains: %s", contentRespBlock)
		}
	})

	t.Run("FieldMask present in UpdateBookMetaRequest (mask: true)", func(t *testing.T) {
		proto := mustReadFile(t, filepath.Join(dir, "generated", "proto", "library_service", "library_service.proto"))
		assertContains(t, "library_service.proto", proto, "google.protobuf.FieldMask update_mask")
	})

	t.Run("ListBookMetasRequest has filter and pagination fields", func(t *testing.T) {
		proto := mustReadFile(t, filepath.Join(dir, "generated", "proto", "library_service", "library_service.proto"))
		assertContains(t, "library_service.proto", proto, "int32 page_size")
		assertContains(t, "library_service.proto", proto, "string page_token")
		assertContains(t, "library_service.proto", proto, "BookMetaFilter filter")
		assertContains(t, "library_service.proto", proto, "string order_by")
		assertContains(t, "library_service.proto", proto, "int32 total_size")
	})

	t.Run("custom method ArchiveBook has AIP-136 colon syntax", func(t *testing.T) {
		proto := mustReadFile(t, filepath.Join(dir, "generated", "proto", "library_service", "library_service.proto"))
		assertContains(t, "library_service.proto", proto, ":archive")
	})

	t.Run("HTTP annotations present", func(t *testing.T) {
		proto := mustReadFile(t, filepath.Join(dir, "generated", "proto", "library_service", "library_service.proto"))
		assertContains(t, "library_service.proto", proto, `post: "/library/LibraryService/book"`)
		assertContains(t, "library_service.proto", proto, `delete: "/library/LibraryService/book/{key.id}"`)
		assertContains(t, "library_service.proto", proto, `patch: "/library/LibraryService/book/{key.id}/meta"`)
		assertContains(t, "library_service.proto", proto, `post: "/library/LibraryService/book/meta/batchGet"`)
		assertContains(t, "library_service.proto", proto, `post: "/library/LibraryService/book/meta/list"`)
	})
}
