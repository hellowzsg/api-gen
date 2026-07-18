// Package build handles proto compilation via protoc-gen-* plugins.
package build

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/bufbuild/protocompile/linker"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/pluginpb"
)

// BuildCodeGeneratorRequest assembles a CodeGeneratorRequest from resolved proto files.
func BuildCodeGeneratorRequest(files linker.Files, fileToGenerate []string) (*pluginpb.CodeGeneratorRequest, error) {
	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: fileToGenerate,
	}
	for _, f := range files {
		fdProto := protodesc.ToFileDescriptorProto(f)
		req.ProtoFile = append(req.ProtoFile, fdProto)
	}
	return req, nil
}

// CheckPluginInstalled checks if a protoc-gen-* plugin is available.
func CheckPluginInstalled(name string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("plugin %q not found in PATH: %w", name, err)
	}
	return path, nil
}

// RunPlugin calls a protoc-gen-* plugin via stdin/stdout protocol.
func RunPlugin(ctx context.Context, pluginName string, req *pluginpb.CodeGeneratorRequest, outDir string) error {
	path, err := CheckPluginInstalled(pluginName)
	if err != nil {
		return err
	}
	reqData, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	cmd := exec.CommandContext(ctx, path)
	cmd.Stdin = bytes.NewReader(reqData)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("plugin %s failed: %w\nstderr: %s", pluginName, err, stderr.String())
	}
	resp := &pluginpb.CodeGeneratorResponse{}
	if err := proto.Unmarshal(stdout.Bytes(), resp); err != nil {
		return fmt.Errorf("unmarshal %s response: %w", pluginName, err)
	}
	if resp.Error != nil {
		return fmt.Errorf("plugin %s error: %s", pluginName, *resp.Error)
	}
	for _, f := range resp.File {
		if f.Name == nil || f.Content == nil {
			continue
		}
		outPath := filepath.Join(outDir, *f.Name)
		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return fmt.Errorf("create dir for %s: %w", *f.Name, err)
		}
		if err := os.WriteFile(outPath, []byte(*f.Content), 0644); err != nil {
			return fmt.Errorf("write %s: %w", *f.Name, err)
		}
	}
	return nil
}

// Compile runs protoc-gen-go and protoc-gen-go-grpc on the given files.
func Compile(ctx context.Context, files linker.Files, fileToGenerate []string, goOutDir string) error {
	req, err := BuildCodeGeneratorRequest(files, fileToGenerate)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if err := os.MkdirAll(goOutDir, 0755); err != nil {
		return fmt.Errorf("create go output dir: %w", err)
	}
	for _, plugin := range []string{"protoc-gen-go", "protoc-gen-go-grpc"} {
		if err := RunPlugin(ctx, plugin, req, goOutDir); err != nil {
			return fmt.Errorf("run %s: %w", plugin, err)
		}
	}
	return nil
}

// suppress unused import
var _ io.Reader
