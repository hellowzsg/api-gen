package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/acme/apigen/internal/dep"
	"github.com/acme/apigen/internal/ir"
	apigenyaml "github.com/acme/apigen/internal/yaml"
)

func runDepUpdate(ctx context.Context, apiYAMLPath string) error {
	cfg, err := parseConfig(apiYAMLPath)
	if err != nil {
		return err
	}
	baseDir := filepath.Dir(apiYAMLPath)
	cacheDir := filepath.Join(baseDir, ".apigen_cache")
	for _, imp := range cfg.ImportProtos {
		switch {
		case imp.Git != "":
			r := dep.NewGitResolver(dep.GitDep{URL: imp.Git, Ref: imp.Ref, Subdir: imp.Subdir}, filepath.Join(cacheDir, "git"))
			if _, err := r.Fetch(); err != nil {
				return fmt.Errorf("git fetch %s: %w", imp.Git, err)
			}
		case imp.BSR != "":
			r := dep.NewBSRResolver([]dep.BSRDep{{Module: imp.BSR, Version: imp.Version}}, baseDir)
			if _, err := r.Fetch(); err != nil {
				return fmt.Errorf("bsr fetch %s: %w", imp.BSR, err)
			}
		}
	}
	return nil
}

func runDepPrune(ctx context.Context, apiYAMLPath string) error {
	cfg, err := parseConfig(apiYAMLPath)
	if err != nil {
		return err
	}
	baseDir := filepath.Dir(apiYAMLPath)
	lockPath := filepath.Join(baseDir, "api.lock")
	deps, err := dep.ReadAPILock(lockPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read api.lock: %w", err)
	}
	// File-level reverse lookup: keep deps whose files are referenced by type_
	referencedTypes := collectReferencedTypes(cfg)
	usedURLs := make(map[string]bool)
	for _, dep := range deps {
		if isDepReferenced(dep, referencedTypes) {
			usedURLs[dep.URL] = true
		}
	}
	var pruned []dep.GitDep
	for _, d := range deps {
		if usedURLs[d.URL] {
			pruned = append(pruned, d)
		}
	}
	if len(pruned) == len(deps) {
		return nil
	}
	return dep.WriteAPILock(lockPath, pruned)
}

func collectReferencedTypes(cfg *apigenyaml.Config) []string {
	var types []string
	for _, e := range cfg.Entities {
		types = append(types, cfg.ResolveTypeName(e.Key.Type))
		for _, r := range e.Resources {
			types = append(types, cfg.ResolveTypeName(r.Type))
		}
	}
	return types
}

func isDepReferenced(d dep.GitDep, types []string) bool {
	// Simplified: in full implementation, would check if any type's proto file
	// comes from this dep's clone directory. For now, keep all deps.
	return true
}

func runEntityList(ctx context.Context, apiYAMLPath string) error {
	cfg, err := parseConfig(apiYAMLPath)
	if err != nil {
		return err
	}
	if err := cfg.ValidateReferences(); err != nil {
		return err
	}
	irData, err := ir.Build(cfg)
	if err != nil {
		return err
	}
	for _, e := range irData.Entities {
		fmt.Printf("Entity: %s (Pascal: %s, Key: %s)\n", e.Name, e.PascalName, e.KeyType)
		if e.Create != nil {
			fmt.Printf("  Create: %s\n", e.Create.RPCName)
		}
		if e.Delete != nil {
			fmt.Printf("  Delete: %s\n", e.Delete.RPCName)
		}
		if e.DeleteSoft != nil {
			fmt.Printf("  DeleteSoft: %s\n", e.DeleteSoft.RPCName)
		}
		for _, r := range e.Resources {
			fmt.Printf("  Resource: %s (Type: %s, Version: %s)\n", r.Name, r.Type, r.Version.Kind)
			if r.Get != nil {
				fmt.Printf("    Get: %s\n", r.Get.RPCName)
			}
			if r.BatchGet != nil {
				fmt.Printf("    BatchGet: %s\n", r.BatchGet.RPCName)
			}
			if r.List != nil {
				fmt.Printf("    List: %s\n", r.List.RPCName)
			}
			if r.Update != nil {
				fmt.Printf("    Update: %s\n", r.Update.RPCName)
			}
		}
	}
	return nil
}
