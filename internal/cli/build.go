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
	cr := dep.NewCompositeResolver(importPaths)
	if pathResolver != nil {
		_ = cr.AddPathResolver(pathResolver)
	}
	files, err := cr.Resolve()
	if err != nil {
		return fmt.Errorf("resolve proto for build: %w", err)
	}
	seedFiles := collectSeedFiles(cfg, baseDir)
	fileToGenerate := cr.CollectTransitiveClosure(seedFiles)
	goOut := filepath.Join(baseDir, cfg.Settings.Out.Go)
	if err := build.Compile(ctx, files, fileToGenerate, goOut); err != nil {
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
