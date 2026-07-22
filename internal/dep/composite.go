package dep

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// CompositeResolver combines multiple resolvers (path, git, bsr) into a single
// protocompile resolver.
type CompositeResolver struct {
	importPaths   []string
	pathResolvers []*PathResolver
	files         linker.Files
	resolved      bool
	// descByFQN indexes every descriptor (messages, enums, services,
	// methods, fields, extensions) across the named files and their
	// transitive imports, built once at Resolve time — lookups are O(1).
	descByFQN map[string]protoreflect.Descriptor
}

// NewCompositeResolver creates a CompositeResolver with initial import paths.
func NewCompositeResolver(importPaths []string) *CompositeResolver {
	return &CompositeResolver{importPaths: importPaths}
}

// AddPathResolver adds a path resolver's files and import paths. Import
// paths already present (e.g. aggregated earlier via Resolver.Fetch) are
// deduplicated, preserving their original declaration order.
func (c *CompositeResolver) AddPathResolver(r *PathResolver) error {
	if _, err := r.ResolveFiles(); err != nil {
		return err
	}
	c.pathResolvers = append(c.pathResolvers, r)
	for _, ip := range r.ImportPaths() {
		if !slices.Contains(c.importPaths, ip) {
			c.importPaths = append(c.importPaths, ip)
		}
	}
	return nil
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
				protoFiles = append(protoFiles, RelToImportRoot(c.importPaths, f))
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
	c.descByFQN = c.buildFQNIndex()
	return files, nil
}

// buildFQNIndex registers every descriptor in the named files and their
// transitive imports by fully-qualified name.
func (c *CompositeResolver) buildFQNIndex() map[string]protoreflect.Descriptor {
	idx := make(map[string]protoreflect.Descriptor)
	for _, f := range c.allFiles() {
		registerEnums(f.Enums(), idx)
		registerMessages(f.Messages(), idx)
		svcs := f.Services()
		for i := 0; i < svcs.Len(); i++ {
			s := svcs.Get(i)
			idx[string(s.FullName())] = s
			methods := s.Methods()
			for j := 0; j < methods.Len(); j++ {
				m := methods.Get(j)
				idx[string(m.FullName())] = m
			}
		}
		exts := f.Extensions()
		for i := 0; i < exts.Len(); i++ {
			idx[string(exts.Get(i).FullName())] = exts.Get(i)
		}
	}
	return idx
}

func registerMessages(msgs protoreflect.MessageDescriptors, idx map[string]protoreflect.Descriptor) {
	for i := 0; i < msgs.Len(); i++ {
		m := msgs.Get(i)
		idx[string(m.FullName())] = m
		fields := m.Fields()
		for j := 0; j < fields.Len(); j++ {
			idx[string(fields.Get(j).FullName())] = fields.Get(j)
		}
		exts := m.Extensions()
		for j := 0; j < exts.Len(); j++ {
			idx[string(exts.Get(j).FullName())] = exts.Get(j)
		}
		registerEnums(m.Enums(), idx)
		registerMessages(m.Messages(), idx)
	}
}

func registerEnums(enums protoreflect.EnumDescriptors, idx map[string]protoreflect.Descriptor) {
	for i := 0; i < enums.Len(); i++ {
		e := enums.Get(i)
		idx[string(e.FullName())] = e
	}
}

// allFiles returns the named files plus their transitively imported files,
// deduplicated by path. Types from git/BSR dependencies are only compiled
// as transitive imports of the named (path-import) files, so descriptor
// lookups must walk this set rather than c.files alone.
func (c *CompositeResolver) allFiles() []linker.File {
	seen := make(map[string]bool, len(c.files))
	out := make([]linker.File, 0, len(c.files))
	var queue []linker.File
	for _, f := range c.files {
		p := string(f.Path())
		if !seen[p] {
			seen[p] = true
			out = append(out, f)
			queue = append(queue, f)
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
			if dep := cur.FindImportByPath(impPath); dep != nil {
				out = append(out, dep)
				queue = append(queue, dep)
			}
		}
	}
	return out
}

func (c *CompositeResolver) findDescriptor(fqn string) protoreflect.Descriptor {
	return c.descByFQN[fqn]
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

// FindMessageDescriptor returns the protoreflect.MessageDescriptor for the
// given fully-qualified message name, or nil if not found / not a message.
// Used by the CLI to obtain key type descriptors for HTTP key-leaf extraction.
func (c *CompositeResolver) FindMessageDescriptor(fqn string) protoreflect.MessageDescriptor {
	if !c.resolved {
		return nil
	}
	d := c.findDescriptor(fqn)
	if d == nil {
		return nil
	}
	md, _ := d.(protoreflect.MessageDescriptor)
	return md
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

// BuildTypeImportPaths builds a map from fully-qualified message/enum type name
// to the proto file path that defines it. This is used by the renderer to emit
// exact import statements for type_ references instead of guessing the file
// path from the package name.
func (c *CompositeResolver) BuildTypeImportPaths() map[string]string {
	if !c.resolved {
		return nil
	}
	m := make(map[string]string)
	for name, d := range c.descByFQN {
		switch d.(type) {
		case protoreflect.MessageDescriptor, protoreflect.EnumDescriptor:
		default:
			continue
		}
		path := string(d.ParentFile().Path())
		// Standard imports never hold user entity types; excluding them
		// keeps the map identical to the named-files-only behavior.
		if isStandardImport(path) {
			continue
		}
		m[name] = path
	}
	return m
}

// ResolveExtra compiles additional proto files (e.g. freshly generated
// service protos) while reusing the already-resolved files as fully-linked
// inputs — previously compiled files are never recompiled. importPaths must
// cover the extra files' roots plus any sources not already resolved.
// Returns the newly compiled files only; their imports link back to the
// reused files.
func (c *CompositeResolver) ResolveExtra(importPaths, protoFiles []string) (linker.Files, error) {
	if !c.resolved {
		return nil, fmt.Errorf("Resolve not called")
	}
	sort.Strings(protoFiles)
	reuse := protocompile.ResolverFunc(func(path string) (protocompile.SearchResult, error) {
		if f := c.files.FindFileByPath(path); f != nil {
			return protocompile.SearchResult{Desc: f}, nil
		}
		return protocompile.SearchResult{}, protoregistry.NotFound
	})
	src := &protocompile.SourceResolver{ImportPaths: importPaths}
	resolver := protocompile.WithStandardImports(protocompile.CompositeResolver{reuse, src})
	compiler := protocompile.Compiler{Resolver: resolver}
	files, err := compiler.Compile(context.Background(), protoFiles...)
	if err != nil {
		return nil, fmt.Errorf("protocompile failed: %w", err)
	}
	return files, nil
}


