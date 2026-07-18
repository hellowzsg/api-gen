package cli

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/acme/apigen/internal/build"
	"github.com/acme/apigen/internal/dep"
	apigenyaml "github.com/acme/apigen/internal/yaml"
)

func runBuild(ctx context.Context, apiYAMLPath string) error {
	if err := runGenerate(ctx, apiYAMLPath); err != nil {
		return err
	}
	cfg, err := parseConfig(apiYAMLPath)
	if err != nil {
		return err
	}
	baseDir := filepath.Dir(apiYAMLPath)
	cacheDir := filepath.Join(baseDir, ".apigen_cache")
	importPaths, err := resolveDependencies(cfg, baseDir, cacheDir)
	if err != nil {
		return err
	}
	pathResolver := setupPathResolver(cfg, baseDir)
	if pathResolver != nil {
		_ = pathResolver.Glob()
		importPaths = append(importPaths, pathResolver.ImportPaths()...)
	}
	// Also resolve the generated service protos (output of runGenerate).
	protoOutDir := filepath.Join(baseDir, cfg.Settings.Out.Proto)
	genResolver := dep.NewPathResolverWithBase(filepath.Join(cfg.Settings.Out.Proto, "**/*.proto"), baseDir)
	cr := dep.NewCompositeResolver(importPaths)
	if pathResolver != nil {
		_ = cr.AddPathResolver(pathResolver)
	}
	_ = cr.AddPathResolver(genResolver)
	files, err := cr.Resolve()
	if err != nil {
		return fmt.Errorf("resolve proto for build: %w", err)
	}
	seedFiles := collectSeedFiles(cfg, baseDir)
	// Convert absolute seed paths to import-root-relative so they match
	// linker.File.Path() keys used inside CollectTransitiveClosure.
	relSeeds := make([]string, 0, len(seedFiles))
	for _, s := range seedFiles {
		rel, err := filepath.Rel(protoOutDir, s)
		if err != nil {
			rel = s
		}
		relSeeds = append(relSeeds, rel)
	}
	fileToGenerate := cr.CollectTransitiveClosure(relSeeds)
	goOut := filepath.Join(baseDir, cfg.Settings.Out.Go)
	openAPIOut := ""
	if cfg.Settings.Out.OpenAPI != "" {
		openAPIOut = filepath.Join(baseDir, cfg.Settings.Out.OpenAPI)
	}
	jsOut := ""
	if cfg.Settings.Out.Js != "" {
		jsOut = filepath.Join(baseDir, cfg.Settings.Out.Js)
	}
	httpEnabled := cfg.Settings.HTTP != nil && cfg.Settings.HTTP.Enable
	generateOpenAPI := httpEnabled && cfg.Settings.HTTP.GenerateOpenAPI
	generateJS := len(cfg.Settings.Plugins.JS) > 0
	if err := build.Compile(ctx, files, fileToGenerate,
		goOut, openAPIOut, jsOut,
		httpEnabled, generateOpenAPI, generateJS); err != nil {
		return fmt.Errorf("compile: %w", err)
	}
	return nil
}

func collectSeedFiles(cfg *apigenyaml.Config, baseDir string) []string {
	var seeds []string
	for _, svc := range cfg.Services {
		seeds = append(seeds, filepath.Join(baseDir, cfg.Settings.Out.Proto,
			toSnakeCase(svc.Name), toSnakeCase(svc.Name)+".proto"))
	}
	return seeds
}
