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
// It performs a real transitive-closure walk over the resolved linker.Files,
// following each file's imports (including WKT) and returning the deduplicated,
// sorted list of all reachable file paths.
func (c *CompositeResolver) CollectTransitiveClosure(seedFiles []string) []string {
	if !c.resolved {
		result := make([]string, len(seedFiles))
		copy(result, seedFiles)
		sort.Strings(result)
		return result
	}
	seen := make(map[string]bool)
	var result []string
	// Index resolved files by path for quick lookup.
	byPath := make(map[string]linker.File, len(c.files))
	for _, f := range c.files {
		byPath[string(f.Path())] = f
	}
	// BFS queue seeded with the explicit seed files.
	var queue []linker.File
	for _, seed := range seedFiles {
		if !seen[seed] {
			seen[seed] = true
			result = append(result, seed)
			if f, ok := byPath[seed]; ok {
				queue = append(queue, f)
			}
		}
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		imports := cur.Imports()
		for i := 0; i < imports.Len(); i++ {
			impPath := string(imports.Get(i).Path())
			if seen[impPath] {
				continue
			}
			seen[impPath] = true
			result = append(result, impPath)
			if dep, ok := byPath[impPath]; ok {
				queue = append(queue, dep)
			} else if lf := cur.FindImportByPath(impPath); lf != nil {
				queue = append(queue, lf)
			}
			// WKT files from protoregistry are not enqueued as linker.File —
			// they have no further user-relevant imports beyond other WKT,
			// which protoc-gen-go resolves internally.
		}
	}
	sort.Strings(result)
	return result
}

// BuildTypeImportPaths builds a map from fully-qualified message/enum type name
// to the proto file path that defines it. This is used by the renderer to emit
// exact import statements for type_ references instead of guessing the file
// path from the package name.
func (c *CompositeResolver) BuildTypeImportPaths() map[string]string {
	if !c.resolved {
		return nil
	}
	m := make(map[string]string)
	for _, f := range c.files {
		path := string(f.Path())
		// Walk all messages and enums defined in this file.
		collectTypeNames(f.Messages(), path, m)
		collectEnumNames(f.Enums(), path, m)
	}
	return m
}

func collectTypeNames(msgs protoreflect.MessageDescriptors, path string, m map[string]string) {
	for i := 0; i < msgs.Len(); i++ {
		md := msgs.Get(i)
		m[string(md.FullName())] = path
		collectTypeNames(md.Messages(), path, m)
		collectEnumNames(md.Enums(), path, m)
	}
}

func collectEnumNames(enums protoreflect.EnumDescriptors, path string, m map[string]string) {
	for i := 0; i < enums.Len(); i++ {
		ed := enums.Get(i)
		m[string(ed.FullName())] = path
	}
}

// ResolveWithFiles resolves the given proto files (relative paths) using
// the import paths, without relying on pathResolvers.
func (c *CompositeResolver) ResolveWithFiles(protoFiles []string) (linker.Files, error) {
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
