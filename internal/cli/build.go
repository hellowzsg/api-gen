package cli

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/hellowzsg/api-gen/internal/build"
	"github.com/hellowzsg/api-gen/internal/dep"
	"github.com/hellowzsg/api-gen/internal/ir"
	apigenyaml "github.com/hellowzsg/api-gen/internal/yaml"
)

// Test seams: allow tests to count Prepare invocations and capture compiler
// inputs without invoking real protoc plugins.
var (
	prepareFn = Prepare
	compileFn = build.Compile
)

func runBuild(ctx context.Context, apiYAMLPath string) error {
	p, err := prepareFn(ctx, apiYAMLPath)
	if err != nil {
		return err
	}
	if err := renderServiceProtos(p); err != nil {
		return err
	}
	cfg := p.Config
	protoOutDir := filepath.Join(p.BaseDir, cfg.Settings.Out.Proto)
	// Compile ONLY the freshly generated service protos; user protos resolved
	// by Prepare are reused as fully-linked inputs (never recompiled).
	genRel := make([]string, 0, len(p.IR.Services))
	for _, svc := range p.IR.Services {
		name := ir.ToSnakeCase(svc.Name)
		genRel = append(genRel, filepath.Join(name, name+".proto"))
	}
	files, err := p.Resolver.ResolveExtra(append([]string{protoOutDir}, p.ImportPaths...), genRel)
	if err != nil {
		return fmt.Errorf("resolve proto for build: %w", err)
	}
	fileToGenerate, err := buildFileToGenerate(p, genRel)
	if err != nil {
		return err
	}
	if err := compileFn(ctx, files, fileToGenerate, pluginSpecsForConfig(cfg, p.BaseDir)); err != nil {
		return fmt.Errorf("compile: %w", err)
	}
	return nil
}

// pluginSpecsForConfig assembles the protoc plugin list from configuration:
// protoc-gen-go and protoc-gen-go-grpc always; grpc-gateway (+openapiv2)
// when HTTP is enabled; protoc-gen-es when JS stubs are configured.
//
// protoc-gen-go is invoked with `paths=source_relative` so output files are
// placed at <goOut>/<proto-relative-path>.pb.go rather than deriving the
// output directory from the go_package import path; grpc and gateway follow
// the same parameter.
func pluginSpecsForConfig(cfg *apigenyaml.Config, baseDir string) []build.PluginSpec {
	goOut := filepath.Join(baseDir, cfg.Settings.Out.Go)
	specs := []build.PluginSpec{
		{Name: "protoc-gen-go", OutDir: goOut, Parameter: "paths=source_relative"},
		{Name: "protoc-gen-go-grpc", OutDir: goOut, Parameter: "paths=source_relative"},
	}
	httpEnabled := cfg.Settings.HTTP != nil && cfg.Settings.HTTP.Enable
	if httpEnabled {
		specs = append(specs, build.PluginSpec{
			Name: "protoc-gen-grpc-gateway", OutDir: goOut, Parameter: "paths=source_relative",
		})
		if cfg.Settings.HTTP.GenerateOpenAPI && cfg.Settings.Out.OpenAPI != "" {
			specs = append(specs, build.PluginSpec{
				Name:      "protoc-gen-openapiv2",
				OutDir:    filepath.Join(baseDir, cfg.Settings.Out.OpenAPI),
				Parameter: "logtostderr=false,json_names_for_fields=false",
			})
		}
	}
	if len(cfg.Settings.Plugins.JS) > 0 && cfg.Settings.Out.Js != "" {
		specs = append(specs, build.PluginSpec{
			Name:      "protoc-gen-es",
			OutDir:    filepath.Join(baseDir, cfg.Settings.Out.Js),
			Parameter: "target=ts",
		})
	}
	return specs
}

// buildFileToGenerate assembles the plugin file list: the generated service
// protos (paths relative to the proto output dir) plus local protos declared
// via import_protos.path (converted from absolute paths to import-root-
// relative paths, matching linker.File.Path() keys).
//
// Remote dependencies (git/BSR) and WKT (google/api/*, google/protobuf/*)
// are deliberately excluded: the transitive closure is already walked inside
// BuildCodeGeneratorRequest to populate ProtoFile. If remote/WKT protos ended
// up in FileToGenerate, plugins would emit code for them under
// generated/<lang>/google/, which is undesirable.
func buildFileToGenerate(p *Pipeline, genRel []string) ([]string, error) {
	fileToGenerate := make([]string, 0, len(genRel))
	fileToGenerate = append(fileToGenerate, genRel...)
	for _, pr := range p.PathResolvers {
		// Already globbed successfully inside Prepare; cannot fail here.
		localFiles, err := pr.ResolveFiles()
		if err != nil {
			return nil, fmt.Errorf("resolve local proto files: %w", err)
		}
		for _, f := range localFiles {
			fileToGenerate = append(fileToGenerate, dep.RelToImportRoot(p.ImportPaths, f))
		}
	}
	return fileToGenerate, nil
}
