package negative

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runBuild runs apigen build on the given fixture and returns the combined error output.
func runBuild(t *testing.T, binary, fixturePath string) (string, int) {
	t.Helper()
	cmd := exec.Command(binary, "build", "-f", fixturePath)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("exec apigen build: %v", err)
		}
	}
	return stdout.String() + stderr.String(), exitCode
}

// TestIRError_OptionValidation verifies option validation errors (I1-I7).
func TestIRError_OptionValidation(t *testing.T) {
	binary := apigenBinary(t)
	dir := fixtureDir(t)

	tests := []struct {
		fixture string
		errMsg  string
	}{
		{"option_invalid_target.yaml", "invalid option target"},
		{"option_field_no_path.yaml", "requires non-empty path"},
		{"option_message_no_path.yaml", "requires non-empty path"},
		{"option_rpc_no_path.yaml", "requires non-empty path"},
		{"option_empty_name.yaml", "option name is empty"},
		{"option_name_with_space.yaml", "contains spaces"},
		{"option_name_illegal_char.yaml", "illegal character"},
	}

	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			output, exitCode := runGenerate(t, binary, filepath.Join(dir, tt.fixture))
			assertExitNonZero(t, tt.fixture, exitCode)
			assertErrorContains(t, tt.fixture, output, tt.errMsg)
		})
	}
}

// TestIRError_KeyLeaves verifies key leaf extraction errors (G3-G6).
// These require HTTP enabled + proto files with problematic key types.
func TestIRError_KeyLeaves(t *testing.T) {
	binary := apigenBinary(t)
	dir := fixtureDir(t)

	tests := []struct {
		fixture string
		errMsg  string
	}{
		{"key_with_repeated.yaml", "repeated"},
		{"key_with_map.yaml", "map"},
		{"key_with_oneof.yaml", "oneof"},
		{"circular_key_ref.yaml", "circular reference"},
	}

	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			output, exitCode := runGenerate(t, binary, filepath.Join(dir, tt.fixture))
			assertExitNonZero(t, tt.fixture, exitCode)
			assertErrorContains(t, tt.fixture, output, tt.errMsg)
		})
	}
}

// TestIRError_ResourceStyleMultiCreate verifies body_style:resource ambiguity error (G2).
func TestIRError_ResourceStyleMultiCreate(t *testing.T) {
	binary := apigenBinary(t)
	dir := fixtureDir(t)

	output, exitCode := runGenerate(t, binary, filepath.Join(dir, "resource_style_multi_create.yaml"))
	assertExitNonZero(t, "resource_style_multi_create.yaml", exitCode)
	assertErrorContains(t, "resource_style_multi_create.yaml", output, "ambiguous")
}

// TestIRError_TypeNotInProto verifies CLI type reference errors (J1-J2).
func TestIRError_TypeNotInProto(t *testing.T) {
	binary := apigenBinary(t)
	dir := fixtureDir(t)

	tests := []struct {
		fixture string
		errMsg  string
	}{
		{"key_type_not_in_proto.yaml", "not found"},
		{"resource_type_not_in_proto.yaml", "not found"},
	}

	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			output, exitCode := runGenerate(t, binary, filepath.Join(dir, tt.fixture))
			assertExitNonZero(t, tt.fixture, exitCode)
			assertErrorContains(t, tt.fixture, output, tt.errMsg)
		})
	}
}
