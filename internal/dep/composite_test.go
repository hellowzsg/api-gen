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

// TestCompositeResolver_SymbolReachability 测试符号可达性校验。
func TestCompositeResolver_SymbolReachability(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "proto", "book.proto"), `
syntax = "proto3";
package test;
message BookId { string id = 1; }
`)

	pathR := NewPathResolverWithBase("proto/**/*.proto", dir)
	if err := pathR.Glob(); err != nil {
		t.Fatalf("Glob failed: %v", err)
	}
	cr := NewCompositeResolver(pathR.ImportPaths())
	_ = cr.AddPathResolver(pathR)
	_, _ = cr.Resolve()

	if err := cr.CheckSymbolReachable("test.BookId"); err != nil {
		t.Errorf("CheckSymbolReachable(test.BookId) = %v, want nil", err)
	}
	if err := cr.CheckSymbolReachable("test.Nonexistent"); err == nil {
		t.Error("CheckSymbolReachable(test.Nonexistent) = nil, want error")
	}
}

// TestCompositeResolver_ImportClosureDryRun 测试 import 闭包 dry-run 校验。
func TestCompositeResolver_ImportClosureDryRun(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "proto", "a.proto"), `
syntax = "proto3";
package test;
import "b.proto";
message A { B b = 1; }
`)
	writeFile(t, filepath.Join(dir, "proto", "b.proto"), `
syntax = "proto3";
package test;
message B { string id = 1; }
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
	if err := cr.DryRunClosure(); err != nil {
		t.Errorf("DryRunClosure failed: %v", err)
	}
}
