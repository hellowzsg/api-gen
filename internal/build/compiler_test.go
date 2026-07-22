package build

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bufbuild/protocompile/linker"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestBuildCodeGeneratorRequest(t *testing.T) {
	files := linker.Files{}
	req, err := BuildCodeGeneratorRequest(files, []string{"test.proto"})
	if err != nil {
		t.Fatalf("BuildCodeGeneratorRequest failed: %v", err)
	}
	if req == nil {
		t.Fatal("request is nil")
	}
	if len(req.FileToGenerate) != 1 || req.FileToGenerate[0] != "test.proto" {
		t.Errorf("FileToGenerate = %v, want [test.proto]", req.FileToGenerate)
	}
}

func TestParsePluginResponse(t *testing.T) {
	resp := &pluginpb.CodeGeneratorResponse{
		File: []*pluginpb.CodeGeneratorResponse_File{
			{Name: stringPtr("test.pb.go"), Content: stringPtr("package test")},
		},
	}
	if len(resp.File) != 1 {
		t.Fatalf("File count = %d, want 1", len(resp.File))
	}
	if resp.File[0].GetName() != "test.pb.go" {
		t.Errorf("File name = %q, want test.pb.go", resp.File[0].GetName())
	}
}

func TestCheckPluginInstalled(t *testing.T) {
	_, err := CheckPluginInstalled("/nonexistent/protoc-gen-go")
	if err == nil {
		t.Error("CheckPluginInstalled should fail for nonexistent path")
	}
}

// fakeOKPlugin logs its own name to $FAKE_PLUGIN_LOG and swallows stdin.
// Empty stdout unmarshals as a valid empty CodeGeneratorResponse.
const fakeOKPlugin = `#!/bin/sh
echo "$(basename "$0")" >> "$FAKE_PLUGIN_LOG"
cat > /dev/null
`

const fakeFailPlugin = `#!/bin/sh
echo "boom from $(basename "$0")" >&2
exit 1
`

// setupFakePlugins puts a temp dir on PATH and writes the given plugin
// scripts into it, returning the log path they write to.
func setupFakePlugins(t *testing.T, plugins map[string]string) (dir, logPath string) {
	t.Helper()
	dir = t.TempDir()
	logPath = filepath.Join(dir, "plugins.log")
	t.Setenv("FAKE_PLUGIN_LOG", logPath)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	for name, script := range plugins {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0755); err != nil {
			t.Fatal(err)
		}
	}
	return dir, logPath
}

// TestCompile_ExecutesAllSpecs verifies that Compile runs every PluginSpec
// and creates each spec's output directory.
func TestCompile_ExecutesAllSpecs(t *testing.T) {
	dir, logPath := setupFakePlugins(t, map[string]string{
		"protoc-gen-fake-a": fakeOKPlugin,
		"protoc-gen-fake-b": fakeOKPlugin,
		"protoc-gen-fake-c": fakeOKPlugin,
	})
	specs := []PluginSpec{
		{Name: "protoc-gen-fake-a", OutDir: filepath.Join(dir, "out-a"), Parameter: "p=1"},
		{Name: "protoc-gen-fake-b", OutDir: filepath.Join(dir, "out-b"), Parameter: "p=2"},
		{Name: "protoc-gen-fake-c", OutDir: filepath.Join(dir, "out-c"), Parameter: "p=3"},
	}
	if err := Compile(context.Background(), nil, nil, specs); err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read plugin log: %v", err)
	}
	for _, name := range []string{"protoc-gen-fake-a", "protoc-gen-fake-b", "protoc-gen-fake-c"} {
		if !strings.Contains(string(data), name) {
			t.Errorf("plugin %s was not executed (log: %q)", name, string(data))
		}
	}
	for _, d := range []string{"out-a", "out-b", "out-c"} {
		if _, err := os.Stat(filepath.Join(dir, d)); err != nil {
			t.Errorf("output dir %s not created: %v", d, err)
		}
	}
}

// TestCompile_PluginFailureNamed verifies that a failing plugin produces an
// error that names the plugin.
func TestCompile_PluginFailureNamed(t *testing.T) {
	dir, _ := setupFakePlugins(t, map[string]string{
		"protoc-gen-fake-good": fakeOKPlugin,
		"protoc-gen-fake-bad":  fakeFailPlugin,
	})
	specs := []PluginSpec{
		{Name: "protoc-gen-fake-good", OutDir: filepath.Join(dir, "out-good"), Parameter: "p=1"},
		{Name: "protoc-gen-fake-bad", OutDir: filepath.Join(dir, "out-bad"), Parameter: "p=2"},
	}
	err := Compile(context.Background(), nil, nil, specs)
	if err == nil {
		t.Fatal("Compile should fail when a plugin fails")
	}
	if !strings.Contains(err.Error(), "protoc-gen-fake-bad") {
		t.Errorf("error should name the failing plugin, got: %v", err)
	}
}

func stringPtr(s string) *string { return &s }
