package dep

import "context"

// Resolver fetches one declared proto dependency (path, git or BSR) and
// reports what it contributes to the compilation.
//
// Implementations must be safe to use from a single goroutine; dependency
// fetching is intentionally sequential to avoid concurrent writes to the
// shared dependency cache.
type Resolver interface {
	// Fetch resolves the dependency (glob, clone or buf export as needed)
	// and returns the protocompile import paths it contributes.
	Fetch(ctx context.Context) (importPaths []string, err error)
	// ProtoFiles returns the local proto files that must be explicitly
	// named for compilation. Dependencies whose protos compile lazily as
	// transitive imports (git, BSR) return nil — explicitly naming them
	// would compile entire dependency repositories.
	ProtoFiles() []string
}
