package build

import (
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

func stringPtr(s string) *string { return &s }
