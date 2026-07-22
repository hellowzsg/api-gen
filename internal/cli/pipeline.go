package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/acme/apigen/internal/dep"
	"github.com/acme/apigen/internal/ir"
	apigenyaml "github.com/acme/apigen/internal/yaml"
)

// Pipeline holds the shared products of the config → resolve → compile → IR
// stages. Both runGenerate and runBuild start from a Pipeline so that api.yaml
// parsing, dependency fetching and protocompile each happen exactly once per
// command invocation.
type Pipeline struct {
	Config        *apigenyaml.Config
	BaseDir       string
	ImportPaths   []string
	PathResolvers []*dep.PathResolver // all path imports (multi-path fix)
	Resolver      *dep.CompositeResolver
	IR            *ir.IR
}

// Prepare runs the shared front half of generate/build: parse api.yaml,
// validate references, fetch dependencies, compile all local protos and
// build the IR (including TypeImportPaths and option validation).
func Prepare(ctx context.Context, apiYAMLPath string) (*Pipeline, error) {
	cfg, err := parseConfig(apiYAMLPath)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.ValidateReferences(); err != nil {
		return nil, fmt.Errorf("validate references: %w", err)
	}
	slog.Info("config parsed", "entities", len(cfg.Entities), "services", len(cfg.Services), "imports", len(cfg.ImportProtos))
	baseDir := filepath.Dir(apiYAMLPath)
	cacheDir := defaultCacheDir()
	importPaths, pathResolvers, err := resolveDependencies(ctx, cfg, baseDir, cacheDir)
	if err != nil {
		return nil, fmt.Errorf("resolve dependencies: %w", err)
	}
	cr := dep.NewCompositeResolver(importPaths)
	for _, pr := range pathResolvers {
		if err := cr.AddPathResolver(pr); err != nil {
			return nil, fmt.Errorf("add path resolver: %w", err)
		}
	}
	resolveStart := time.Now()
	files, err := cr.Resolve()
	if err != nil {
		return nil, fmt.Errorf("resolve proto: %w", err)
	}
	slog.Info("protos resolved", "files", len(files), "duration", time.Since(resolveStart))
	if err := validateTypeReferences(cfg, cr); err != nil {
		return nil, fmt.Errorf("validate type references: %w", err)
	}
	irData, err := buildIR(cfg, cr)
	if err != nil {
		return nil, fmt.Errorf("build IR: %w", err)
	}
	irData.TypeImportPaths = cr.BuildTypeImportPaths()
	if err := ir.ValidateAllOptions(irData); err != nil {
		return nil, fmt.Errorf("validate options: %w", err)
	}
	slog.Info("IR built", "entities", len(irData.Entities), "services", len(irData.Services))
	return &Pipeline{
		Config:        cfg,
		BaseDir:       baseDir,
		ImportPaths:   importPaths,
		PathResolvers: pathResolvers,
		Resolver:      cr,
		IR:            irData,
	}, nil
}

// namedResolver pairs a Resolver with its api.yaml identity for error
// messages identical to the pre-refactor per-kind wrapping.
type namedResolver struct {
	r    dep.Resolver
	kind string // "path" / "git" / "bsr"
	name string
}

// resolveDependencies fetches all declared dependencies via the unified
// dep.Resolver interface, in api.yaml declaration order. It returns the
// aggregated import paths plus one PathResolver per path import (already
// globbed, in declaration order).
func resolveDependencies(ctx context.Context, cfg *apigenyaml.Config, baseDir, cacheDir string) ([]string, []*dep.PathResolver, error) {
	// Load resolved commits from api.lock so git deps use content-addressed
	// cache keys (URL+commit) instead of moving refs (URL+branch). A missing
	// lock file is fine; a corrupt one is a hard error — silently ignoring it
	// would mask lock corruption and fall back to moving-ref caches.
	lockedCommits, err := dep.ReadAPILock(filepath.Join(baseDir, "api.lock"))
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, nil, err
		}
		lockedCommits = nil
	}
	commitByURL := make(map[string]string, len(lockedCommits))
	for _, d := range lockedCommits {
		commitByURL[d.URL] = d.ResolvedCommit
	}

	// All BSR deps are batched into ONE resolver (single throwaway buf.yaml
	// and single buf dep update per run), placed at the first BSR entry.
	var bsrDeps []dep.BSRDep
	for _, imp := range cfg.ImportProtos {
		if imp.BSR != "" {
			bsrDeps = append(bsrDeps, dep.BSRDep{Module: imp.BSR, Version: imp.Version})
		}
	}
	resolvers := make([]namedResolver, 0, len(cfg.ImportProtos))
	bsrAdded := false
	for _, imp := range cfg.ImportProtos {
		switch {
		case imp.Path != "":
			resolvers = append(resolvers, namedResolver{
				r:    dep.NewPathResolverWithBase(imp.Path, baseDir),
				kind: "path",
				name: imp.Path,
			})
		case imp.Git != "":
			gd := dep.GitDep{URL: imp.Git, Ref: imp.Ref, Subdir: imp.Subdir}
			if c, ok := commitByURL[imp.Git]; ok {
				gd.ResolvedCommit = c
			}
			resolvers = append(resolvers, namedResolver{
				r:    dep.NewGitResolver(gd, cacheDir),
				kind: "git",
				name: imp.Git,
			})
		case imp.BSR != "":
			if !bsrAdded {
				bsrAdded = true
				name := imp.BSR
				if len(bsrDeps) > 1 {
					name = fmt.Sprintf("%s (+%d more)", imp.BSR, len(bsrDeps)-1)
				}
				resolvers = append(resolvers, namedResolver{
					r:    dep.NewBSRResolver(bsrDeps, cacheDir),
					kind: "bsr",
					name: name,
				})
			}
		}
	}

	var importPaths []string
	var pathResolvers []*dep.PathResolver
	for _, nr := range resolvers {
		slog.Info("dep fetch start", "kind", nr.kind, "name", nr.name)
		start := time.Now()
		paths, err := nr.r.Fetch(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("%s dependency %q: %w", nr.kind, nr.name, err)
		}
		slog.Info("dep fetch done", "kind", nr.kind, "name", nr.name, "duration", time.Since(start))
		importPaths = append(importPaths, paths...)
		if pr, ok := nr.r.(*dep.PathResolver); ok {
			pathResolvers = append(pathResolvers, pr)
		}
	}
	return importPaths, pathResolvers, nil
}
