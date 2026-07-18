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
	"google.golang.org/protobuf/types/descriptorpb"
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

	// Topologically sort ProtoFile so that dependencies precede dependents.
	// protoc-gen-go tolerates any order (it indexes via protoregistry), but
	// protoc-gen-es requires strict dependency-first ordering — otherwise it
	// reports "Cannot find <file>.proto, imported by <dependent>.proto".
	sortProtoFilesTopologically(req)

	return req, nil
}

// sortProtoFilesTopologically reorders req.ProtoFile so that every file
// appears before any file that imports it (dependency-first / topological order).
// Files with no dependencies come first; the relative order among
// independent files is preserved as much as possible (stable sort).
func sortProtoFilesTopologically(req *pluginpb.CodeGeneratorRequest) {
	type info struct {
		name    string
		deps    map[string]bool
		idx     int
		visited bool
		onStack bool
	}
	infos := make(map[string]*info, len(req.ProtoFile))
	for i, pf := range req.ProtoFile {
		name := pf.GetName()
		deps := make(map[string]bool, len(pf.GetDependency()))
		for _, d := range pf.GetDependency() {
			deps[d] = true
		}
		infos[name] = &info{name: name, deps: deps, idx: i}
	}

	var sorted []*descriptorpb.FileDescriptorProto
	var visit func(name string)
	visit = func(name string) {
		inf := infos[name]
		if inf == nil || inf.visited {
			return
		}
		if inf.onStack {
			// Circular dependency — bail out and let original order handle it.
			return
		}
		inf.onStack = true
		for dep := range inf.deps {
			visit(dep)
		}
		inf.onStack = false
		if !inf.visited {
			inf.visited = true
			sorted = append(sorted, req.ProtoFile[inf.idx])
		}
	}
	// Visit in original order for stable output.
	for i := range req.ProtoFile {
		visit(req.ProtoFile[i].GetName())
	}
	req.ProtoFile = sorted
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
// When httpEnabled is true, protoc-gen-grpc-gateway is also invoked to
// generate *.pb.gw.go alongside the *.pb.go files.
//
// When generateOpenAPI is true (and httpEnabled is true), protoc-gen-openapiv2
// is invoked to generate <service>.swagger.json into openAPIOutDir.
//
// When generateJS is true, protoc-gen-es is invoked to generate TypeScript
// stubs (*.pb.ts) into jsOutDir with plugin parameter target=ts.
//
// The protoc-gen-go plugin is invoked with `paths=source_relative` so that
// output files are placed at <goOutDir>/<proto-relative-path>.pb.go rather
// than deriving the output directory from the go_package import path. This
// matches the design doc's layout: generated/go/<service>/<service>.pb.go.
// protoc-gen-go-grpc and protoc-gen-grpc-gateway follow the same parameter.
func Compile(ctx context.Context, files linker.Files, fileToGenerate []string, goOutDir, openAPIOutDir, jsOutDir string, httpEnabled, generateOpenAPI, generateJS bool) error {
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
	// protoc-gen-grpc-gateway: only when HTTP is enabled. Generates
	// *.pb.gw.go in the same output directory.
	if httpEnabled {
		gwReq := proto.Clone(req).(*pluginpb.CodeGeneratorRequest)
		gwReq.Parameter = &param
		if err := RunPlugin(ctx, "protoc-gen-grpc-gateway", gwReq, goOutDir); err != nil {
			return fmt.Errorf("run protoc-gen-grpc-gateway: %w", err)
		}
		// protoc-gen-openapiv2: only when OpenAPI generation is enabled.
		// Generates <service>.swagger.json into openAPIOutDir.
		if generateOpenAPI && openAPIOutDir != "" {
			if err := os.MkdirAll(openAPIOutDir, 0755); err != nil {
				return fmt.Errorf("create openapi output dir: %w", err)
			}
			openapiReq := proto.Clone(req).(*pluginpb.CodeGeneratorRequest)
			openapiParam := "logtostderr=false,json_names_for_fields=false"
			openapiReq.Parameter = &openapiParam
			if err := RunPlugin(ctx, "protoc-gen-openapiv2", openapiReq, openAPIOutDir); err != nil {
				return fmt.Errorf("run protoc-gen-openapiv2: %w", err)
			}
		}
	}
	// protoc-gen-es: only when JS stub generation is enabled. Generates
	// TypeScript files (*.pb.ts) into jsOutDir with target=ts.
	if generateJS && jsOutDir != "" {
		if err := os.MkdirAll(jsOutDir, 0755); err != nil {
			return fmt.Errorf("create js output dir: %w", err)
		}
		jsReq := proto.Clone(req).(*pluginpb.CodeGeneratorRequest)
		jsParam := "target=ts"
		jsReq.Parameter = &jsParam
		if err := RunPlugin(ctx, "protoc-gen-es", jsReq, jsOutDir); err != nil {
			return fmt.Errorf("run protoc-gen-es: %w", err)
		}
	}
	return nil
}
