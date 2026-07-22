package dep

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// BSRDep declares a BSR proto dependency.
type BSRDep struct {
	Module  string `yaml:"module"`
	Version string `yaml:"version,omitempty"`
}

// ref returns the BSR module ref, qualified with the optional version label
// (e.g. buf.build/acme/one:v1.2.3).
func (d BSRDep) ref() string {
	if d.Version != "" {
		return d.Module + ":" + d.Version
	}
	return d.Module
}

// BSRResolver fetches BSR dependencies via the buf CLI. All BSR deps of one
// run are batched into a single resolver: one throwaway buf.yaml, one
// `buf dep update`, and per-module content-addressed exports.
type BSRResolver struct {
	deps        []BSRDep
	bufCmd      string
	cacheDir    string
	importPaths []string
}

// NewBSRResolver creates a BSRResolver.
// cacheDir is the apigen cache root (e.g. ~/.cache/apigen); BSR exports are
// stored under <cacheDir>/<CacheVersion>/module-proxy/bsr/.
func NewBSRResolver(deps []BSRDep, cacheDir string) *BSRResolver {
	return &BSRResolver{deps: deps, cacheDir: cacheDir, bufCmd: "buf"}
}

// NewBSRResolverWithBufCmd creates a BSRResolver with a custom buf binary path.
func NewBSRResolverWithBufCmd(deps []BSRDep, cacheDir, bufCmd string) *BSRResolver {
	r := NewBSRResolver(deps, cacheDir)
	r.bufCmd = bufCmd
	return r
}

// bsrModulePattern validates BSR module names.
var bsrModulePattern = regexp.MustCompile(`^buf\.build/[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+$`)

// bsrVersionPattern validates BSR version labels (tags, branches, commits).
var bsrVersionPattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// validateBSRModule validates a BSR module name for subprocess safety.
func validateBSRModule(module string) error {
	if module == "" {
		return fmt.Errorf("BSR module is empty")
	}
	if !bsrModulePattern.MatchString(module) {
		return fmt.Errorf("invalid BSR module %q: must match buf.build/<owner>/<repo>", module)
	}
	return nil
}

// validateBSRVersion validates a BSR version label for subprocess/YAML safety.
func validateBSRVersion(version string) error {
	if version == "" {
		return nil
	}
	if !bsrVersionPattern.MatchString(version) {
		return fmt.Errorf("invalid BSR version %q: illegal characters", version)
	}
	return nil
}

func (r *BSRResolver) detectBuf() error {
	_, err := exec.LookPath(r.bufCmd)
	if err != nil {
		return fmt.Errorf("buf CLI not found at %q: %w", r.bufCmd, err)
	}
	return nil
}

// bufYAMLContent renders the throwaway buf.yaml (v2) covering all deps.
func bufYAMLContent(deps []BSRDep) string {
	var sb strings.Builder
	sb.WriteString("# buf.yaml — 由 apigen 自动生成，请勿手改\n")
	sb.WriteString("version: v2\n")
	sb.WriteString("modules:\n  - path: .\n")
	sb.WriteString("deps:\n")
	for _, d := range deps {
		sb.WriteString("  - " + d.ref() + "\n")
	}
	return sb.String()
}

// parseBufLockDigests extracts module → digest from a buf.lock file.
func parseBufLockDigests(lockPath string) (map[string]string, error) {
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return nil, err
	}
	var lock struct {
		Deps []struct {
			Name   string `yaml:"name"`
			Digest string `yaml:"digest"`
		} `yaml:"deps"`
	}
	if err := yaml.Unmarshal(data, &lock); err != nil {
		return nil, err
	}
	m := make(map[string]string, len(lock.Deps))
	for _, d := range lock.Deps {
		m[d.Name] = d.Digest
	}
	return m, nil
}

// Fetch runs a single `buf dep update` for all deps (in a throwaway temp
// dir, never the user workspace) and exports each module into a
// content-addressed cache directory keyed by its buf.lock digest. Modules
// whose digest is already exported are skipped (cache hit).
func (r *BSRResolver) Fetch(ctx context.Context) ([]string, error) {
	if len(r.deps) == 0 {
		return nil, nil
	}
	if err := r.detectBuf(); err != nil {
		return nil, err
	}
	for _, d := range r.deps {
		if err := validateBSRModule(d.Module); err != nil {
			return nil, err
		}
		if err := validateBSRVersion(d.Version); err != nil {
			return nil, err
		}
	}
	// buf.yaml/buf.lock live in a throwaway temp dir — the user workspace is
	// never touched, and the temp dir is always cleaned up.
	workDir, err := os.MkdirTemp("", "apigen-buf-*")
	if err != nil {
		return nil, fmt.Errorf("create buf workdir: %w", err)
	}
	defer os.RemoveAll(workDir)
	if err := os.WriteFile(filepath.Join(workDir, "buf.yaml"), []byte(bufYAMLContent(r.deps)), 0644); err != nil {
		return nil, fmt.Errorf("write buf.yaml: %w", err)
	}
	updateCmd := exec.CommandContext(ctx, r.bufCmd, "dep", "update")
	updateCmd.Dir = workDir
	if output, err := updateCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("buf dep update failed: %w\n%s", err, string(output))
	}
	digests, err := parseBufLockDigests(filepath.Join(workDir, "buf.lock"))
	if err != nil {
		return nil, fmt.Errorf("parse buf.lock: %w", err)
	}

	bsrCacheDir := filepath.Join(moduleProxyBase(r.cacheDir), "bsr")
	var importPaths []string
	for _, d := range r.deps {
		// The cache key is the content digest from buf.lock — bumping a
		// version or a moving ref automatically invalidates old exports.
		digest := strings.ReplaceAll(digests[d.Module], ":", "_")
		if digest == "" {
			digest = "unknown"
		}
		moduleDir := filepath.Join(bsrCacheDir, strings.ReplaceAll(d.Module, "/", "_"))
		exportDir := filepath.Join(moduleDir, digest)
		if _, err := os.Stat(exportDir); err == nil {
			// Cache hit: skip buf export entirely.
			slog.Info("dep fetch cache hit", "kind", "bsr", "module", d.Module, "cache_hit", true)
			importPaths = append(importPaths, exportDir)
			continue
		}
		// Export into a sibling temp dir, then atomically publish by rename
		// so a crashed run never leaves a half-populated cache entry.
		if err := os.MkdirAll(moduleDir, 0755); err != nil {
			return nil, fmt.Errorf("create module cache dir: %w", err)
		}
		tmpDir, err := os.MkdirTemp(moduleDir, ".export-*")
		if err != nil {
			return nil, fmt.Errorf("create export staging dir: %w", err)
		}
		exportCmd := exec.CommandContext(ctx, r.bufCmd, "export", d.ref(), "--output", tmpDir)
		exportCmd.Dir = workDir
		if output, err := exportCmd.CombinedOutput(); err != nil {
			os.RemoveAll(tmpDir)
			return nil, fmt.Errorf("buf export %s failed: %w\n%s", d.ref(), err, string(output))
		}
		if err := os.Rename(tmpDir, exportDir); err != nil {
			os.RemoveAll(tmpDir)
			return nil, fmt.Errorf("publish export %s: %w", d.Module, err)
		}
		importPaths = append(importPaths, exportDir)
	}
	r.importPaths = importPaths
	return importPaths, nil
}

// ProtoFiles implements Resolver. BSR protos compile lazily as transitive
// imports of explicitly named files, so this always returns nil.
func (r *BSRResolver) ProtoFiles() []string {
	return nil
}

// ImportPaths returns the export directories of THIS resolver's declared
// modules only. Unlike the previous implementation it never enumerates the
// whole bsr cache directory, so unrelated cached modules cannot leak into
// the compilation import paths.
func (r *BSRResolver) ImportPaths() []string {
	return r.importPaths
}
