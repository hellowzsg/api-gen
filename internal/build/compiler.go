// Package build handles proto compilation via protoc-gen-* plugins.
package build

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"github.com/bufbuild/protocompile/linker"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/pluginpb"
)

// BuildCodeGeneratorRequest assembles a CodeGeneratorRequest from resolved
// proto files. It includes all files in the linker.Files set plus their
// transitive dependencies (including Well-Known Types) so that protoc-gen-go
// receives a complete FileDescriptorSet.
//
// The request contains:
//   - FileToGenerate: the files the plugin should generate output for.
//   - ProtoFile: every file in the transitive closure (user protos + generated
//     service protos + all WKT they import), deduplicated by path.
//
// WKT files (google/protobuf/*.proto) are resolved from the protoregistry
// global files (populated by the Go protobuf runtime via init()), since
// protocompile's WithStandardImports links them but does not always surface
// them as linker.File entries in the returned Files slice.
func BuildCodeGeneratorRequest(files linker.Files, fileToGenerate []string) (*pluginpb.CodeGeneratorRequest, error) {
	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: fileToGenerate,
	}

	// Build a path→linker.File index for fast lookup.
	byPath := make(map[string]linker.File, len(files))
	for _, f := range files {
		byPath[string(f.Path())] = f
	}

	// Walk transitive imports starting from every file in `files`.
	// We collect descriptors into req.ProtoFile, deduped by path.
	seen := make(map[string]bool)
	var queue []linker.File
	for _, f := range files {
		path := string(f.Path())
		if !seen[path] {
			seen[path] = true
			req.ProtoFile = append(req.ProtoFile, protodesc.ToFileDescriptorProto(f))
			queue = append(queue, f)
		}
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		imports := cur.Imports()
		for i := 0; i < imports.Len(); i++ {
			impPath := string(imports.Get(i).Path())
			if seen[impPath] {
				continue
			}
			seen[impPath] = true
			// Prefer the linker-resolved file (covers user + generated protos
			// and WKT that protocompile linked via WithStandardImports).
			if impFile, ok := byPath[impPath]; ok {
				req.ProtoFile = append(req.ProtoFile, protodesc.ToFileDescriptorProto(impFile))
				queue = append(queue, impFile)
				continue
			}
			// Fall back to the linker's FindImportByPath, which walks the
			// file's own dependency closure.
			if dep := cur.FindImportByPath(impPath); dep != nil {
				byPath[impPath] = dep
				req.ProtoFile = append(req.ProtoFile, protodesc.ToFileDescriptorProto(dep))
				queue = append(queue, dep)
				continue
			}
			// Final fallback: the global WKT registry (populated by Go
			// protobuf runtime init). Covers google/protobuf/empty.proto,
			// field_mask.proto, wrappers.proto, etc.
			if fd, err := protoregistry.GlobalFiles.FindFileByPath(impPath); err == nil {
				req.ProtoFile = append(req.ProtoFile, protodesc.ToFileDescriptorProto(fd))
				// WKT may themselves import other WKT (e.g. source_context
				// imported by type.proto), so enqueue for further walking.
				if lf, ok := fd.(linker.File); ok {
					queue = append(queue, lf)
				} else {
					// Walk imports via the generic FileDescriptor interface.
					walkImportDeps(fd, seen, req, byPath, &queue)
				}
			}
			// If none of the above found the file, it is genuinely missing —
			// but we do not error here because protocompile already validated
			// the closure in the Resolve/DryRunClosure step. A missing file
			// at this point would indicate a logic error upstream.
		}
	}
	return req, nil
}

// walkImportDeps enqueues imports of a plain protoreflect.FileDescriptor
// (used when the WKT registry returns a non-linker File descriptor).
func walkImportDeps(fd protoreflect.FileDescriptor, seen map[string]bool, req *pluginpb.CodeGeneratorRequest, byPath map[string]linker.File, queue *[]linker.File) {
	imports := fd.Imports()
	for i := 0; i < imports.Len(); i++ {
		impPath := string(imports.Get(i).Path())
		if seen[impPath] {
			continue
		}
		seen[impPath] = true
		if dep, err := protoregistry.GlobalFiles.FindFileByPath(impPath); err == nil {
			req.ProtoFile = append(req.ProtoFile, protodesc.ToFileDescriptorProto(dep))
			walkImportDeps(dep, seen, req, byPath, queue)
		}
	}
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
// `parameter` is passed as the CodeGeneratorRequest.parameter field (e.g.
// "paths=source_relative" for protoc-gen-go to emit files relative to the
// output dir rather than deriving paths from go_package).
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
	// Sort files for deterministic write order.
	files := resp.File
	sort.Slice(files, func(i, j int) bool {
		return files[i].GetName() < files[j].GetName()
	})
	for _, f := range files {
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
// `files` must contain the full transitive closure of protos (user + generated
// service protos + WKT). `fileToGenerate` lists the service proto paths that
// the plugins should emit Go code for.
//
// The protoc-gen-go plugin is invoked with `paths=source_relative` so that
// output files are placed at <goOutDir>/<proto-relative-path>.pb.go rather
// than deriving the output directory from the go_package import path. This
// matches the design doc's layout: generated/go/<service>/<service>.pb.go.
// protoc-gen-go-grpc ignores the parameter and follows protoc-gen-go's lead.
func Compile(ctx context.Context, files linker.Files, fileToGenerate []string, goOutDir string) error {
	req, err := BuildCodeGeneratorRequest(files, fileToGenerate)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if err := os.MkdirAll(goOutDir, 0755); err != nil {
		return fmt.Errorf("create go output dir: %w", err)
	}
	// protoc-gen-go: emit source-relative paths.
	goReq := proto.Clone(req).(*pluginpb.CodeGeneratorRequest)
	param := "paths=source_relative"
	goReq.Parameter = &param
	if err := RunPlugin(ctx, "protoc-gen-go", goReq, goOutDir); err != nil {
		return fmt.Errorf("run protoc-gen-go: %w", err)
	}
	// protoc-gen-go-grpc: also use source_relative so the grpc file lands
	// next to the .pb.go file.
	grpcReq := proto.Clone(req).(*pluginpb.CodeGeneratorRequest)
	grpcReq.Parameter = &param
	if err := RunPlugin(ctx, "protoc-gen-go-grpc", grpcReq, goOutDir); err != nil {
		return fmt.Errorf("run protoc-gen-go-grpc: %w", err)
	}
	return nil
}
