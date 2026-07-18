package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/acme/apigen/internal/dep"
	"github.com/acme/apigen/internal/ir"
	"github.com/acme/apigen/internal/render"
	apigenyaml "github.com/acme/apigen/internal/yaml"
)

func runGenerate(ctx context.Context, apiYAMLPath string) error {
	cfg, err := parseConfig(apiYAMLPath)
	if err != nil {
		return fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.ValidateReferences(); err != nil {
		return fmt.Errorf("validate references: %w", err)
	}
	baseDir := filepath.Dir(apiYAMLPath)
	cacheDir := filepath.Join(baseDir, ".apigen_cache")
	importPaths, err := resolveDependencies(cfg, baseDir, cacheDir)
	if err != nil {
		return fmt.Errorf("resolve dependencies: %w", err)
	}
	pathResolver := setupPathResolver(cfg, baseDir)
	if pathResolver != nil {
		if err := pathResolver.Glob(); err != nil {
			return fmt.Errorf("glob proto files: %w", err)
		}
		importPaths = append(importPaths, pathResolver.ImportPaths()...)
	}
	cr := dep.NewCompositeResolver(importPaths)
	if pathResolver != nil {
		if err := cr.AddPathResolver(pathResolver); err != nil {
			return fmt.Errorf("add path resolver: %w", err)
		}
	}
	if _, err := cr.Resolve(); err != nil {
		return fmt.Errorf("resolve proto: %w", err)
	}
	if err := validateTypeReferences(cfg, cr); err != nil {
		return fmt.Errorf("validate type references: %w", err)
	}
	irData, err := buildIR(cfg, cr)
	if err != nil {
		return fmt.Errorf("build IR: %w", err)
	}
	irData.TypeImportPaths = cr.BuildTypeImportPaths()
	if err := ir.ValidateAllOptions(irData); err != nil {
		return fmt.Errorf("validate options: %w", err)
	}
	// Render all service protos into a staging directory first; only swap
	// into the real output dir once every service has rendered successfully.
	// This makes generate atomic — a failure midway leaves the previous
	// output intact.
	protoOutDir := filepath.Join(baseDir, cfg.Settings.Out.Proto)
	staging, err := newStagingDir(protoOutDir)
	if err != nil {
		return fmt.Errorf("create staging dir: %w", err)
	}
	// Ensure staging is cleaned up if we return before committing.
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(staging)
		}
	}()
	for _, svc := range irData.Services {
		output, err := render.RenderServiceProto(irData, svc)
		if err != nil {
			return fmt.Errorf("render service %s: %w", svc.Name, err)
		}
		outPath := filepath.Join(staging, toSnakeCase(svc.Name), toSnakeCase(svc.Name)+".proto")
		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return fmt.Errorf("create output dir: %w", err)
		}
		if err := os.WriteFile(outPath, []byte(output), 0644); err != nil {
			return fmt.Errorf("write proto file: %w", err)
		}
	}
	if err := commitDir(staging, protoOutDir); err != nil {
		return fmt.Errorf("commit proto output: %w", err)
	}
	committed = true
	return nil
}

func parseConfig(path string) (*apigenyaml.Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return apigenyaml.Parse(f)
}

// buildIR constructs the IR from the YAML config, optionally enriching it
// with key type descriptors for HTTP path binding. When HTTP is disabled,
// behavior is identical to ir.Build (P0). When HTTP is enabled, key type
// descriptors are fetched from the CompositeResolver so that ExtractKeyLeaves
// can run.
func buildIR(cfg *apigenyaml.Config, cr *dep.CompositeResolver) (*ir.IR, error) {
	httpEnabled := cfg.Settings.HTTP != nil && cfg.Settings.HTTP.Enable
	if !httpEnabled {
		return ir.Build(cfg)
	}
	// Build KeyDescriptors map for HTTP key-leaf extraction.
	keyDescs := make(map[string]protoreflect.MessageDescriptor, len(cfg.Entities))
	for _, e := range cfg.Entities {
		keyType := cfg.ResolveTypeName(e.Key.Type)
		md := cr.FindMessageDescriptor(keyType)
		if md == nil {
			return nil, fmt.Errorf("HTTP enabled but key type %q descriptor not found in resolved protos", keyType)
		}
		keyDescs[keyType] = md
	}
	return ir.BuildWithOptions(cfg, ir.BuildOptions{KeyDescriptors: keyDescs})
}

func resolveDependencies(cfg *apigenyaml.Config, baseDir, cacheDir string) ([]string, error) {
	var importPaths []string
	for _, imp := range cfg.ImportProtos {
		switch {
		case imp.Path != "":
			r := dep.NewPathResolverWithBase(imp.Path, baseDir)
			if err := r.Glob(); err != nil {
				return nil, fmt.Errorf("path dependency %q: %w", imp.Path, err)
			}
			importPaths = append(importPaths, r.ImportPaths()...)
		case imp.Git != "":
			r := dep.NewGitResolver(dep.GitDep{URL: imp.Git, Ref: imp.Ref, Subdir: imp.Subdir}, filepath.Join(cacheDir, "git"))
			paths, err := r.Fetch()
			if err != nil {
				return nil, fmt.Errorf("git dependency %q: %w", imp.Git, err)
			}
			importPaths = append(importPaths, paths...)
		case imp.BSR != "":
			r := dep.NewBSRResolver([]dep.BSRDep{{Module: imp.BSR, Version: imp.Version}}, baseDir)
			paths, err := r.Fetch()
			if err != nil {
				return nil, fmt.Errorf("bsr dependency %q: %w", imp.BSR, err)
			}
			importPaths = append(importPaths, paths...)
		}
	}
	return importPaths, nil
}

func setupPathResolver(cfg *apigenyaml.Config, baseDir string) *dep.PathResolver {
	for _, imp := range cfg.ImportProtos {
		if imp.Path != "" {
			return dep.NewPathResolverWithBase(imp.Path, baseDir)
		}
	}
	return nil
}

func validateTypeReferences(cfg *apigenyaml.Config, cr *dep.CompositeResolver) error {
	for _, e := range cfg.Entities {
		keyType := cfg.ResolveTypeName(e.Key.Type)
		if err := cr.CheckTypeIsMessage(keyType); err != nil {
			return fmt.Errorf("entity %q key.type_ %q: %w", e.Name, keyType, err)
		}
		for _, r := range e.Resources {
			resType := cfg.ResolveTypeName(r.Type)
			if err := cr.CheckTypeIsMessage(resType); err != nil {
				return fmt.Errorf("entity %q resource %q type_ %q: %w", e.Name, r.Name, resType, err)
			}
		}
	}
	return nil
}

func toSnakeCase(s string) string {
	var sb strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				sb.WriteByte('_')
			}
			sb.WriteRune(r + 32)
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}
