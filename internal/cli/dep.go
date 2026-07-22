package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hellowzsg/api-gen/internal/dep"
	"github.com/hellowzsg/api-gen/internal/ir"
	apigenyaml "github.com/hellowzsg/api-gen/internal/yaml"
)

func runDepUpdate(ctx context.Context, apiYAMLPath string) error {
	cfg, err := parseConfig(apiYAMLPath)
	if err != nil {
		return err
	}
	baseDir := filepath.Dir(apiYAMLPath)
	cacheDir := defaultCacheDir()
	lockPath := filepath.Join(baseDir, "api.lock")

	// Load existing lock so we preserve any previously resolved commits and
	// only re-fetch when necessary.
	existing, _ := dep.ReadAPILock(lockPath)
	commitByURL := make(map[string]string, len(existing))
	for _, d := range existing {
		commitByURL[d.URL] = d.ResolvedCommit
	}

	var updated []dep.GitDep
	var bsrDeps []dep.BSRDep
	for _, imp := range cfg.ImportProtos {
		switch {
		case imp.Git != "":
			gd := dep.GitDep{URL: imp.Git, Ref: imp.Ref, Subdir: imp.Subdir}
			if c, ok := commitByURL[imp.Git]; ok {
				gd.ResolvedCommit = c
			}
			r := dep.NewGitResolver(gd, cacheDir)
			if _, err := r.Fetch(ctx); err != nil {
				return fmt.Errorf("git fetch %s: %w", imp.Git, err)
			}
			updated = append(updated, dep.GitDep{
				URL:            imp.Git,
				Ref:            imp.Ref,
				Subdir:         imp.Subdir,
				ResolvedCommit: r.ResolvedCommit(),
			})
		case imp.BSR != "":
			bsrDeps = append(bsrDeps, dep.BSRDep{Module: imp.BSR, Version: imp.Version})
		}
	}
	// BSR deps are batched into a single fetch (one buf dep update).
	if len(bsrDeps) > 0 {
		r := dep.NewBSRResolver(bsrDeps, cacheDir)
		if _, err := r.Fetch(ctx); err != nil {
			return fmt.Errorf("bsr fetch: %w", err)
		}
	}
	if len(updated) > 0 {
		if err := dep.WriteAPILock(lockPath, updated); err != nil {
			return fmt.Errorf("write api.lock: %w", err)
		}
	}
	return nil
}

func runDepPrune(ctx context.Context, apiYAMLPath string) error {
	// Prepare resolves + compiles everything so we know exactly which proto
	// file defines each referenced type.
	p, err := Prepare(ctx, apiYAMLPath)
	if err != nil {
		return err
	}
	lockPath := filepath.Join(p.BaseDir, "api.lock")
	deps, err := dep.ReadAPILock(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read api.lock: %w", err)
	}
	if len(deps) == 0 {
		return nil
	}
	referencedTypes := collectReferencedTypes(p.Config)
	cacheDir := defaultCacheDir()
	var pruned []dep.GitDep
	for _, d := range deps {
		if isGitDepReferenced(d, referencedTypes, p.IR.TypeImportPaths, cacheDir) {
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

// isGitDepReferenced reports whether any referenced type's proto file exists
// under the dep's cache clone directory. Undecidable cases (clone absent
// from the cache, or a referenced type missing from the compiled type map)
// conservatively return true — prune never deletes what it cannot prove
// unused.
func isGitDepReferenced(d dep.GitDep, types []string, typeFiles map[string]string, cacheDir string) bool {
	root := dep.NewGitResolver(d, cacheDir).CloneDir()
	if d.Subdir != "" {
		root = filepath.Join(root, d.Subdir)
	}
	if _, err := os.Stat(root); err != nil {
		return true
	}
	for _, t := range types {
		f := typeFiles[t]
		if f == "" {
			return true
		}
		if _, err := os.Stat(filepath.Join(root, f)); err == nil {
			return true
		}
	}
	return false
}

func runEntityList(ctx context.Context, apiYAMLPath string) error {
	cfg, err := parseConfig(apiYAMLPath)
	if err != nil {
		return err
	}
	if err := cfg.ValidateReferences(); err != nil {
		return err
	}
	// entity list is display-only: skip the HTTP key-descriptor requirement
	// (no HTTP paths are printed).
	irData, err := ir.BuildWithOptions(cfg, ir.BuildOptions{LenientHTTP: true})
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
