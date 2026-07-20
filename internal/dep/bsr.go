package dep

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// BSRDep declares a BSR proto dependency.
type BSRDep struct {
	Module  string `yaml:"module"`
	Version string `yaml:"version,omitempty"`
}

// BSRResolver manages BSR dependencies via buf CLI subprocess.
type BSRResolver struct {
	deps     []BSRDep
	workDir  string
	bufCmd   string
	cacheDir string
}

// NewBSRResolver creates a BSRResolver.
// cacheDir is the apigen cache root (e.g. ~/.cache/apigen); BSR exports are
// stored under <cacheDir>/bsr/.
func NewBSRResolver(deps []BSRDep, workDir, cacheDir string) *BSRResolver {
	return &BSRResolver{deps: deps, workDir: workDir, cacheDir: cacheDir, bufCmd: "buf"}
}

// NewBSRResolverWithBufCmd creates a BSRResolver with a custom buf binary path.
func NewBSRResolverWithBufCmd(deps []BSRDep, workDir, cacheDir, bufCmd string) *BSRResolver {
	r := NewBSRResolver(deps, workDir, cacheDir)
	r.bufCmd = bufCmd
	return r
}

// bsrModulePattern validates BSR module names.
var bsrModulePattern = regexp.MustCompile(`^buf\.build/[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+$`)

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

// GenerateBufYAML generates buf.yaml (v2) with BSR dependencies.
func (r *BSRResolver) GenerateBufYAML() error {
	if len(r.deps) == 0 {
		return nil
	}
	for _, d := range r.deps {
		if err := validateBSRModule(d.Module); err != nil {
			return err
		}
	}
	var sb strings.Builder
	sb.WriteString("# buf.yaml — 由 apigen 自动生成，请勿手改\n")
	sb.WriteString("version: v2\n")
	sb.WriteString("modules:\n  - path: .\n")
	sb.WriteString("deps:\n")
	for _, d := range r.deps {
		sb.WriteString(fmt.Sprintf("  - %s\n", d.Module))
	}
	return os.WriteFile(filepath.Join(r.workDir, "buf.yaml"), []byte(sb.String()), 0644)
}

// HasDeps returns true if there are BSR dependencies.
func (r *BSRResolver) HasDeps() bool {
	return len(r.deps) > 0
}

func (r *BSRResolver) detectBuf() error {
	_, err := exec.LookPath(r.bufCmd)
	if err != nil {
		return fmt.Errorf("buf CLI not found at %q: %w", r.bufCmd, err)
	}
	return nil
}

// Fetch runs buf dep update + buf export to fetch BSR dependencies.
func (r *BSRResolver) Fetch() ([]string, error) {
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
	}
	if err := r.GenerateBufYAML(); err != nil {
		return nil, err
	}
	updateCmd := exec.Command(r.bufCmd, "dep", "update")
	updateCmd.Dir = r.workDir
	if output, err := updateCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("buf dep update failed: %w\n%s", err, string(output))
	}
	bsrCacheDir := filepath.Join(r.cacheDir, "bsr")
	if err := os.MkdirAll(bsrCacheDir, 0755); err != nil {
		return nil, fmt.Errorf("create bsr cache dir: %w", err)
	}
	// After Fetch, r.cacheDir points to the bsr subdirectory so that
	// ImportPaths() can enumerate module export directories.
	r.cacheDir = bsrCacheDir
	var importPaths []string
	seen := make(map[string]bool)
	for _, d := range r.deps {
		exportDir := filepath.Join(bsrCacheDir, strings.ReplaceAll(d.Module, "/", "_"))
		exportCmd := exec.Command(r.bufCmd, "export", d.Module, "--output", exportDir)
		exportCmd.Dir = r.workDir
		if output, err := exportCmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("buf export %s failed: %w\n%s", d.Module, err, string(output))
		}
		if !seen[exportDir] {
			seen[exportDir] = true
			importPaths = append(importPaths, exportDir)
		}
	}
	sort.Strings(importPaths)
	return importPaths, nil
}

// ImportPaths returns the BSR export directories for protocompile.
func (r *BSRResolver) ImportPaths() []string {
	if r.cacheDir == "" {
		return nil
	}
	entries, err := os.ReadDir(r.cacheDir)
	if err != nil {
		return nil
	}
	var paths []string
	for _, e := range entries {
		if e.IsDir() {
			paths = append(paths, filepath.Join(r.cacheDir, e.Name()))
		}
	}
	sort.Strings(paths)
	return paths
}
