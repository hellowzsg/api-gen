package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/acme/apigen/internal/build"
	"github.com/acme/apigen/internal/dep"
)

// runBuild runs generate (producing service protos) and then compiles the
// generated service protos (plus their transitive dependency closure, which
// includes user type_ protos and Well-Known Types) into Go stubs via
// protoc-gen-go / protoc-gen-go-grpc.
//
// The build step re-resolves the full proto set (user protos + generated
// service protos) through protocompile so that the CodeGeneratorRequest sent
// to the plugins contains a complete FileDescriptorSet. This is the "fourth
// line of defense" from the design doc: generate and build share the same
// resolved file set, guaranteeing the generated protos are compilable.
func runBuild(ctx context.Context, apiYAMLPath string) error {
	// Step 1: generate service protos (also validates deps + type_ references).
	if err := runGenerate(ctx, apiYAMLPath); err != nil {
		return err
	}

	cfg, err := parseConfig(apiYAMLPath)
	if err != nil {
		return err
	}
	baseDir := filepath.Dir(apiYAMLPath)
	cacheDir := filepath.Join(baseDir, ".apigen_cache")
	genProtoDir := filepath.Join(baseDir, cfg.Settings.Out.Proto)
	goOutDir := filepath.Join(baseDir, cfg.Settings.Out.Go)
	// OpenAPI output dir: explicit setting or default "generated/openapi".
	openAPIOutDir := ""
	generateOpenAPI := false
	if cfg.Settings.HTTP != nil && cfg.Settings.HTTP.Enable && cfg.Settings.HTTP.GenerateOpenAPI {
		generateOpenAPI = true
		openAPISubDir := cfg.Settings.Out.OpenAPI
		if openAPISubDir == "" {
			openAPISubDir = "generated/openapi"
		}
		openAPIOutDir = filepath.Join(baseDir, openAPISubDir)
	}

	// Step 2: collect the service proto files to generate Go code for.
	var serviceProtoFiles []string
	for _, svc := range cfg.Services {
		svcSnake := toSnakeCase(svc.Name)
		serviceProtoFiles = append(serviceProtoFiles, svcSnake+"/"+svcSnake+".proto")
	}
	sort.Strings(serviceProtoFiles)

	// Step 3: build the complete import-path set and the full proto file list.
	// Import paths = user proto roots (from path deps) + git/bsr dep dirs +
	// generated proto root (so service protos can import user type_ protos
	// and each other).
	var importPaths []string
	var userTypeProtoFiles []string
	for _, imp := range cfg.ImportProtos {
		switch {
		case imp.Path != "":
			r := dep.NewPathResolverWithBase(imp.Path, baseDir)
			if err := r.Glob(); err != nil {
				return fmt.Errorf("path dependency %q: %w", imp.Path, err)
			}
			importPaths = append(importPaths, r.ImportPaths()...)
			matchedFiles, _ := r.ResolveFiles()
			for _, f := range matchedFiles {
				// Compute the proto import path (relative to the nearest
				// import root) for each matched user proto file.
				rel := f
				for _, ip := range r.ImportPaths() {
					if rp, err := filepath.Rel(ip, f); err == nil && !strings.HasPrefix(rp, "..") {
						rel = rp
						break
					}
				}
				userTypeProtoFiles = append(userTypeProtoFiles, filepath.ToSlash(rel))
			}
		case imp.Git != "":
			r := dep.NewGitResolver(dep.GitDep{URL: imp.Git, Ref: imp.Ref, Subdir: imp.Subdir}, filepath.Join(cacheDir, "git"))
			paths, err := r.Fetch()
			if err != nil {
				return fmt.Errorf("git dependency %q: %w", imp.Git, err)
			}
			importPaths = append(importPaths, paths...)
		case imp.BSR != "":
			r := dep.NewBSRResolver([]dep.BSRDep{{Module: imp.BSR, Version: imp.Version}}, baseDir)
			paths, err := r.Fetch()
			if err != nil {
				return fmt.Errorf("bsr dependency %q: %w", imp.BSR, err)
			}
			importPaths = append(importPaths, paths...)
		}
	}
	importPaths = append(importPaths, genProtoDir)

	// The full proto set to resolve: user type_ protos + generated service
	// protos. protocompile will pull in WKT (google/protobuf/*.proto) via
	// WithStandardImports.
	allProtoFiles := make([]string, 0, len(userTypeProtoFiles)+len(serviceProtoFiles))
	allProtoFiles = append(allProtoFiles, userTypeProtoFiles...)
	allProtoFiles = append(allProtoFiles, serviceProtoFiles...)
	sort.Strings(allProtoFiles)

	// Files to generate Go code for: both the service protos AND the user
	// type_ protos. The service stubs import the user type packages, so the
	// user type protos must also be compiled into .pb.go for the generated
	// module to be self-consistent.
	//
	// Exclude any vendored googleapis protos (e.g. google/api/*.proto) that
	// were matched by broad import_protos globs. These must remain as
	// link-only dependencies: their Go bindings come from the
	// google.golang.org/genproto/googleapis/api/annotations module, not from
	// locally generated code. Generating them would produce duplicate symbol
	// errors when users import the canonical genproto package.
	fileToGenerate := make([]string, 0, len(allProtoFiles))
	for _, f := range allProtoFiles {
		if isVendoredDependency(f) {
			continue
		}
		fileToGenerate = append(fileToGenerate, f)
	}

	// Step 4: resolve the full set through protocompile to obtain linked
	// FileDescriptors. This produces the complete transitive closure (user
	// protos + generated service protos + WKT) that the plugins need.
	cr := dep.NewCompositeResolver(importPaths)
	files, err := cr.ResolveWithFiles(allProtoFiles)
	if err != nil {
		return fmt.Errorf("resolve proto for build: %w", err)
	}

	// Step 5: compile with protoc-gen-go + protoc-gen-go-grpc into a staging
	// directory. Only swap into the real go output dir once both plugins
	// succeed, so a compile failure leaves the previous Go stubs intact.
	staging, err := newStagingDir(goOutDir)
	if err != nil {
		return fmt.Errorf("create staging dir: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(staging)
		}
	}()
	if err := build.Compile(ctx, files, fileToGenerate, staging, openAPIOutDir, cfg.Settings.HTTP != nil && cfg.Settings.HTTP.Enable, generateOpenAPI); err != nil {
		return fmt.Errorf("compile: %w", err)
	}
	if err := commitDir(staging, goOutDir); err != nil {
		return fmt.Errorf("commit go output: %w", err)
	}
	committed = true
	return nil
}

// isVendoredDependency reports whether a proto import path refers to a
// third-party dependency that should be linked but not compiled to Go code.
//
// This covers:
//   - google/api/*.proto      (googleapis HTTP annotations — Go bindings from
//     google.golang.org/genproto/googleapis/api/annotations)
//   - google/protobuf/*.proto (Well-Known Types — Go bindings from
//     google.golang.org/protobuf/types/known/*)
//
// User-authored protos and apigen-generated service protos never start with
// "google/", so they are always compiled.
func isVendoredDependency(protoPath string) bool {
	return strings.HasPrefix(protoPath, "google/")
}
