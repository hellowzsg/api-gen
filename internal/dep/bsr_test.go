package dep

import (
	"os"
	"path/filepath"
	"testing"
)

// TestBSRResolver_BufYAMLGeneration 测试 buf.yaml(v2) 生成。
func TestBSRResolver_BufYAMLGeneration(t *testing.T) {
	dir := t.TempDir()
	r := NewBSRResolver([]BSRDep{{Module: "buf.build/googleapis/googleapis"}}, dir)
	if err := r.GenerateBufYAML(); err != nil {
		t.Fatalf("GenerateBufYAML failed: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "buf.yaml"))
	if err != nil {
		t.Fatalf("read buf.yaml: %v", err)
	}
	content := string(data)
	if !contains(content, "version: v2") {
		t.Error("buf.yaml missing 'version: v2'")
	}
	if !contains(content, "buf.build/googleapis/googleapis") {
		t.Error("buf.yaml missing BSR module")
	}
}

// TestBSRResolver_BufNotInstalled 测试 buf 未安装检测。
func TestBSRResolver_BufNotInstalled(t *testing.T) {
	dir := t.TempDir()
	r := NewBSRResolverWithBufCmd([]BSRDep{{Module: "buf.build/googleapis/googleapis"}}, dir, "/nonexistent/buf")
	_, err := r.Fetch()
	if err == nil {
		t.Fatal("Fetch should fail when buf not installed")
	}
}

// TestBSRResolver_ValidateModule 测试 BSR module 名校验（白名单）。
func TestBSRResolver_ValidateModule(t *testing.T) {
	tests := []struct {
		module string
		valid  bool
	}{
		{"buf.build/googleapis/googleapis", true},
		{"buf.build/acme/options", true},
		{"buf.build/googleapis/googleapis;rm -rf /", false},
		{"buf.build/foo/bar extra", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.module, func(t *testing.T) {
			err := validateBSRModule(tt.module)
			if tt.valid && err != nil {
				t.Errorf("validateBSRModule(%q) = %v, want nil", tt.module, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("validateBSRModule(%q) = nil, want error", tt.module)
			}
		})
	}
}

// TestBSRResolver_NoDeps 测试无 BSR 依赖时不生成 buf.yaml。
func TestBSRResolver_NoDeps(t *testing.T) {
	dir := t.TempDir()
	r := NewBSRResolver(nil, dir)
	if err := r.GenerateBufYAML(); err != nil {
		t.Errorf("GenerateBufYAML with no deps should not error, got: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "buf.yaml")); !os.IsNotExist(err) {
		t.Error("buf.yaml should not exist when no BSR deps")
	}
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
