package dep

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// CompositeResolver combines multiple resolvers (path, git, bsr) into a single
// protocompile resolver.
type CompositeResolver struct {
	importPaths   []string
	pathResolvers []*PathResolver
	files         linker.Files
	resolved      bool
}

// NewCompositeResolver creates a CompositeResolver with initial import paths.
func NewCompositeResolver(importPaths []string) *CompositeResolver {
	return &CompositeResolver{importPaths: importPaths}
}

// AddPathResolver adds a path resolver's files and import paths.
func (c *CompositeResolver) AddPathResolver(r *PathResolver) error {
	if _, err := r.ResolveFiles(); err != nil {
		return err
	}
	c.pathResolvers = append(c.pathResolvers, r)
	c.importPaths = append(c.importPaths, r.ImportPaths()...)
	return nil
}

// AddImportPaths adds additional import paths.
func (c *CompositeResolver) AddImportPaths(paths []string) {
	c.importPaths = append(c.importPaths, paths...)
}

// Resolve runs protocompile to parse all proto files.
func (c *CompositeResolver) Resolve() (linker.Files, error) {
	var protoFiles []string
	seen := make(map[string]bool)
	for _, r := range c.pathResolvers {
		files, _ := r.ResolveFiles()
		for _, f := range files {
			if !seen[f] {
				seen[f] = true
				// Convert absolute path to relative path from the nearest import path
				rel := f
				for _, ip := range c.importPaths {
					if relPath, err := filepath.Rel(ip, f); err == nil && !strings.HasPrefix(relPath, "..") {
						rel = relPath
						break
					}
				}
				protoFiles = append(protoFiles, rel)
			}
		}
	}
	sort.Strings(protoFiles)

	srcResolver := &protocompile.SourceResolver{ImportPaths: c.importPaths}
	resolver := protocompile.WithStandardImports(srcResolver)
	compiler := protocompile.Compiler{Resolver: resolver}
	files, err := compiler.Compile(context.Background(), protoFiles...)
	if err != nil {
		return nil, fmt.Errorf("protocompile failed: %w", err)
	}
	c.files = files
	c.resolved = true
	return files, nil
}

func (c *CompositeResolver) findDescriptor(fqn string) protoreflect.Descriptor {
	for _, f := range c.files {
		if d := f.FindDescriptorByName(protoreflect.FullName(fqn)); d != nil {
			return d
		}
	}
	return nil
}

// CheckSymbolReachable checks that a fully-qualified symbol name exists.
func (c *CompositeResolver) CheckSymbolReachable(fqn string) error {
	if !c.resolved {
		return fmt.Errorf("Resolve not called")
	}
	if d := c.findDescriptor(fqn); d != nil {
		return nil
	}
	return fmt.Errorf("symbol %q not found in resolved proto files", fqn)
}

// CheckTypeIsMessage checks that a type resolves to a proto message.
func (c *CompositeResolver) CheckTypeIsMessage(fqn string) error {
	if !c.resolved {
		return fmt.Errorf("Resolve not called")
	}
	d := c.findDescriptor(fqn)
	if d == nil {
		return fmt.Errorf("type %q not found", fqn)
	}
	if _, ok := d.(protoreflect.MessageDescriptor); !ok {
		return fmt.Errorf("type %q is not a message (got %T)", fqn, d)
	}
	return nil
}

// DryRunClosure validates that all imports are resolvable.
func (c *CompositeResolver) DryRunClosure() error {
	if !c.resolved {
		return fmt.Errorf("Resolve not called")
	}
	for _, f := range c.files {
		imports := f.Imports()
		for i := 0; i < imports.Len(); i++ {
			imp := imports.Get(i)
			found := false
			for _, rf := range c.files {
				if string(rf.Path()) == string(imp.Path()) || string(rf.Name()) == string(imp.Path()) {
					found = true
					break
				}
			}
			if !found && isStandardImport(string(imp.Path())) {
				continue
			}
			if !found {
				return fmt.Errorf("unresolved import %q in file %q", imp.Path(), f.Path())
			}
		}
	}
	return nil
}

func isStandardImport(path string) bool {
	return strings.HasPrefix(path, "google/protobuf/") ||
		path == "google/api/annotations.proto" ||
		path == "google/api/http.proto"
}

// Files returns the resolved linker.Files.
func (c *CompositeResolver) Files() linker.Files {
	return c.files
}

// CollectTransitiveClosure returns proto file paths for plugins.
func (c *CompositeResolver) CollectTransitiveClosure(seedFiles []string) []string {
	result := make([]string, len(seedFiles))
	copy(result, seedFiles)
	sort.Strings(result)
	return result
}
