package dep

import (
	"path/filepath"
	"testing"
)

// TestCompositeResolver_LocalProto 测试本地 proto 解析（仅 path 依赖）。
func TestCompositeResolver_LocalProto(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "proto", "book.proto"), `
syntax = "proto3";
package test;
message BookId { string id = 1; }
message BookMeta { string title = 1; }
`)

	pathR := NewPathResolverWithBase("proto/**/*.proto", dir)
	if err := pathR.Glob(); err != nil {
		t.Fatalf("Glob failed: %v", err)
	}
	cr := NewCompositeResolver(pathR.ImportPaths())
	if err := cr.AddPathResolver(pathR); err != nil {
		t.Fatalf("AddPathResolver failed: %v", err)
	}
	files, err := cr.Resolve()
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("Resolve returned no files")
	}
}

// TestCompositeResolver_TypeConstraint 测试 type_ 类型约束校验（必须 message 类型）。
func TestCompositeResolver_TypeConstraint(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "proto", "book.proto"), `
syntax = "proto3";
package test;
message BookId { string id = 1; }
enum Status { UNKNOWN = 0; }
`)

	pathR := NewPathResolverWithBase("proto/**/*.proto", dir)
	if err := pathR.Glob(); err != nil {
		t.Fatalf("Glob failed: %v", err)
	}
	cr := NewCompositeResolver(pathR.ImportPaths())
	_ = cr.AddPathResolver(pathR)
	if _, err := cr.Resolve(); err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if err := cr.CheckTypeIsMessage("test.BookId"); err != nil {
		t.Errorf("CheckTypeIsMessage(test.BookId) = %v, want nil", err)
	}
	if err := cr.CheckTypeIsMessage("test.Status"); err == nil {
		t.Error("CheckTypeIsMessage(test.Status) = nil, want error (enum is not message)")
	}
	if err := cr.CheckTypeIsMessage("test.Nonexistent"); err == nil {
		t.Error("CheckTypeIsMessage(test.Nonexistent) = nil, want error")
	}
}

// TestCompositeResolver_IndexConsistency asserts descriptor lookups behave
// identically with the FQN index: named files, transitively imported files,
// and absent symbols.
func TestCompositeResolver_IndexConsistency(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.proto"), `syntax = "proto3"; package test.dep; message MsgA { string id = 1; }`)
	writeFile(t, filepath.Join(dir, "b.proto"), `syntax = "proto3"; package test.dep; import "a.proto"; message MsgB { MsgA a = 1; }`)
	pathR := NewPathResolverWithBase(filepath.Join(dir, "b.proto"), "")
	cr := NewCompositeResolver(pathR.ImportPaths())
	if err := cr.AddPathResolver(pathR); err != nil {
		t.Fatalf("AddPathResolver failed: %v", err)
	}
	if _, err := cr.Resolve(); err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	// Type from the named file.
	if err := cr.CheckTypeIsMessage("test.dep.MsgB"); err != nil {
		t.Errorf("MsgB (named file) should resolve: %v", err)
	}
	// Type from the transitively imported file.
	if err := cr.CheckTypeIsMessage("test.dep.MsgA"); err != nil {
		t.Errorf("MsgA (transitive import) should resolve: %v", err)
	}
	if md := cr.FindMessageDescriptor("test.dep.MsgA"); md == nil {
		t.Error("FindMessageDescriptor(MsgA) = nil, want non-nil")
	}
	// Enum/messages share the index; absent symbols still error.
	if err := cr.CheckTypeIsMessage("test.dep.Absent"); err == nil {
		t.Error("absent type should error")
	}
	// Types from files OUTSIDE the import closure are not indexed.
	if err := cr.CheckTypeIsMessage("test.other.Nope"); err == nil {
		t.Error("type outside closure should error")
	}
}
