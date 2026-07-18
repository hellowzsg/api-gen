// Package dep manages proto dependency resolution across three paths:
// local (path), git, and BSR.
package dep

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// PathResolver resolves local proto files via glob patterns.
type PathResolver struct {
	pattern     string
	baseDir     string
	files       []string
	importPaths []string
}

// NewPathResolver creates a PathResolver with an absolute glob pattern.
func NewPathResolver(pattern string) *PathResolver {
	return &PathResolver{pattern: pattern}
}

// NewPathResolverWithBase creates a PathResolver with a relative glob pattern
// resolved against baseDir.
func NewPathResolverWithBase(pattern, baseDir string) *PathResolver {
	return &PathResolver{pattern: pattern, baseDir: baseDir}
}

// Glob expands the glob pattern and collects matching .proto files.
func (r *PathResolver) Glob() error {
	pattern := r.pattern
	if r.baseDir != "" && !filepath.IsAbs(pattern) {
		pattern = filepath.Join(r.baseDir, pattern)
	}
	matches, err := doubleStarGlob(pattern)
	if err != nil {
		return fmt.Errorf("glob %q: %w", pattern, err)
	}
	if len(matches) == 0 {
		return fmt.Errorf("no .proto files matched pattern %q", pattern)
	}
	r.files = matches
	seen := make(map[string]bool)
	for _, f := range matches {
		d := filepath.Dir(f)
		if !seen[d] {
			seen[d] = true
			r.importPaths = append(r.importPaths, d)
		}
	}
	sort.Strings(r.importPaths)
	return nil
}

// ResolveFiles returns the list of matched .proto file paths.
func (r *PathResolver) ResolveFiles() ([]string, error) {
	if r.files == nil {
		if err := r.Glob(); err != nil {
			return nil, err
		}
	}
	return r.files, nil
}

// ImportPaths returns the directories to add to protocompile's import paths.
func (r *PathResolver) ImportPaths() []string {
	return r.importPaths
}

// doubleStarGlob handles ** glob patterns by walking the directory tree.
func doubleStarGlob(pattern string) ([]string, error) {
	if !strings.Contains(pattern, "**") {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}
		return filterProto(matches), nil
	}
	idx := strings.Index(pattern, "**")
	baseDir := filepath.Clean(pattern[:idx])
	if baseDir == "" {
		baseDir = "."
	}
	suffix := pattern[idx+2:]
	suffix = strings.TrimPrefix(suffix, string(filepath.Separator))

	var matches []string
	err := filepath.WalkDir(baseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".proto" {
			return nil
		}
		if suffix != "" {
			rel, err := filepath.Rel(baseDir, path)
			if err != nil {
				return nil
			}
			if !matchSuffix(suffix, rel) {
				return nil
			}
		}
		matches = append(matches, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	return matches, nil
}

func filterProto(files []string) []string {
	var result []string
	for _, f := range files {
		if filepath.Ext(f) == ".proto" {
			result = append(result, f)
		}
	}
	return result
}

// matchSuffix matches a glob suffix (which may contain **) against a relative path.
// e.g. suffix="*.proto" matches "a.proto" and "sub/b.proto" (after ** stripping).
func matchSuffix(suffix, rel string) bool {
	// Strip leading **/ — means "any depth"
	s := suffix
	if strings.HasPrefix(s, "**/") {
		s = s[3:]
	}
	if s == "*.proto" {
		// Match .proto at any depth
		return filepath.Ext(rel) == ".proto"
	}
	// Strip internal **/ segments and try matching
	s = strings.ReplaceAll(s, "**/", "")
	matched, err := filepath.Match(s, rel)
	if err != nil {
		return false
	}
	if matched {
		return true
	}
	// Try matching against base name
	base := filepath.Base(rel)
	matched, err = filepath.Match(s, base)
	return err == nil && matched
}
