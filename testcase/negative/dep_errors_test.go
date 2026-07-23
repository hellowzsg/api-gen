package negative

import (
	"path/filepath"
	"testing"
)

// TestDepError_GlobNoMatch verifies that glob pattern with no matches returns error.
func TestDepError_GlobNoMatch(t *testing.T) {
	binary := apigenBinary(t)
	dir := fixtureDir(t)

	output, exitCode := runGenerate(t, binary, filepath.Join(dir, "glob_no_match.yaml"))
	assertExitNonZero(t, "glob_no_match.yaml", exitCode)
	assertErrorContains(t, "glob_no_match.yaml", output, "no .proto files matched")
}

// TestDepError_ProtocompileFailed verifies that proto compilation errors are reported.
func TestDepError_ProtocompileFailed(t *testing.T) {
	binary := apigenBinary(t)
	dir := fixtureDir(t)

	output, exitCode := runGenerate(t, binary, filepath.Join(dir, "protocompile_error", "api.yaml"))
	assertExitNonZero(t, "protocompile_error/api.yaml", exitCode)
	// protocompile should fail with a parse/syntax error
	assertErrorContains(t, "protocompile_error/api.yaml", output, "resolve proto")
}

// TestDepError_CorruptAPILock verifies that a corrupt api.lock file causes an error.
func TestDepError_CorruptAPILock(t *testing.T) {
	binary := apigenBinary(t)
	dir := fixtureDir(t)

	output, exitCode := runGenerate(t, binary, filepath.Join(dir, "corrupt_lock", "api.yaml"))
	assertExitNonZero(t, "corrupt_lock/api.yaml", exitCode)
	// The error should be about reading api.lock (not "file not found")
	assertErrorContains(t, "corrupt_lock/api.yaml", output, "resolve dependencies")
}
