package negative

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// apigenBinary returns the path to the apigen binary.
// The binary must be built before running tests: go build -o /tmp/apigen ./cmd/apigen
func apigenBinary(t *testing.T) string {
	t.Helper()
	for _, p := range []string{"/tmp/apigen", "../apigen"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	// Fall back to building it
	repoRoot, _ := filepath.Abs("../..")
	binary := filepath.Join(os.TempDir(), "apigen")
	cmd := exec.Command("go", "build", "-o", binary, "./cmd/apigen")
	cmd.Dir = repoRoot
	if err := cmd.Run(); err != nil {
		t.Fatalf("build apigen: %v", err)
	}
	return binary
}

// fixtureDir returns the absolute path to the invalid fixtures directory.
func fixtureDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	return filepath.Join(wd, "..", "fixtures", "invalid")
}

// runGenerate runs apigen generate on the given fixture and returns the combined error output.
func runGenerate(t *testing.T, binary, fixturePath string) (string, int) {
	t.Helper()
	cmd := exec.Command(binary, "generate", "-f", fixturePath)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("exec apigen: %v", err)
		}
	}
	return stdout.String() + stderr.String(), exitCode
}

// assertErrorContains verifies that the output contains the expected substring.
func assertErrorContains(t *testing.T, fixtureName, output string, expectedSubstr string) {
	t.Helper()
	if !strings.Contains(output, expectedSubstr) {
		t.Errorf("fixture %q: expected error containing %q, got:\n%s", fixtureName, expectedSubstr, output)
	}
}

// assertExitNonZero verifies the command exited with non-zero status.
func assertExitNonZero(t *testing.T, fixtureName string, exitCode int) {
	t.Helper()
	if exitCode == 0 {
		t.Errorf("fixture %q: expected non-zero exit code, got 0", fixtureName)
	}
}

// TestGenerateError_YAMLParsing verifies YAML parsing layer errors (A1-A12).
func TestGenerateError_YAMLParsing(t *testing.T) {
	binary := apigenBinary(t)
	dir := fixtureDir(t)

	tests := []struct {
		fixture string
		errMsg  string
	}{
		{"missing_syntax.yaml", "missing required field: syntax"},
		{"missing_name.yaml", "missing required field: name"},
		{"invalid_name_format.yaml", "invalid name"},
		{"missing_entities.yaml", "missing required field: entities"},
		{"entity_missing_name.yaml", "name is required"},
		{"entity_missing_key_type.yaml", "type_ is required"},
		{"entity_no_resources.yaml", "at least one resource"},
		{"resource_missing_name.yaml", "name is required"},
		{"resource_missing_type.yaml", "type_ is required"},
		{"resource_missing_version.yaml", "kind is required"},
		{"unknown_yaml_field.yaml", "decode api.yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			output, exitCode := runGenerate(t, binary, filepath.Join(dir, tt.fixture))
			assertExitNonZero(t, tt.fixture, exitCode)
			assertErrorContains(t, tt.fixture, output, tt.errMsg)
		})
	}
}

// TestGenerateError_TypeReferences verifies type reference validation errors (B1-B5).
func TestGenerateError_TypeReferences(t *testing.T) {
	binary := apigenBinary(t)
	dir := fixtureDir(t)

	tests := []struct {
		fixture string
		errMsg  string
	}{
		{"key_type_empty.yaml", "type_ is required"},
		{"key_type_dot_prefix.yaml", "must start with letter or underscore"},
		{"key_type_digit_prefix.yaml", "must start with letter or underscore"},
		{"key_type_illegal_char.yaml", "illegal character"},
		{"filter_type_invalid.yaml", "must start with letter or underscore"},
	}

	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			output, exitCode := runGenerate(t, binary, filepath.Join(dir, tt.fixture))
			assertExitNonZero(t, tt.fixture, exitCode)
			assertErrorContains(t, tt.fixture, output, tt.errMsg)
		})
	}
}

// TestGenerateError_ServiceReferences verifies service reference validation errors (C1-C3).
func TestGenerateError_ServiceReferences(t *testing.T) {
	binary := apigenBinary(t)
	dir := fixtureDir(t)

	tests := []struct {
		fixture string
		errMsg  string
	}{
		{"duplicate_entity.yaml", "duplicate entity name"},
		{"service_ref_undefined_entity.yaml", "references nonexistent entity"},
		{"service_ref_undefined_resource.yaml", "is not declared in entity"},
	}

	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			output, exitCode := runGenerate(t, binary, filepath.Join(dir, tt.fixture))
			assertExitNonZero(t, tt.fixture, exitCode)
			assertErrorContains(t, tt.fixture, output, tt.errMsg)
		})
	}
}

// TestGenerateError_HTTPConfig verifies HTTP configuration errors (D1-D2).
func TestGenerateError_HTTPConfig(t *testing.T) {
	binary := apigenBinary(t)
	dir := fixtureDir(t)

	tests := []struct {
		fixture string
		errMsg  string
	}{
		{"http_without_googleapis.yaml", "no googleapis dependency"},
		{"invalid_body_style.yaml", "invalid value"},
	}

	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			output, exitCode := runGenerate(t, binary, filepath.Join(dir, tt.fixture))
			assertExitNonZero(t, tt.fixture, exitCode)
			assertErrorContains(t, tt.fixture, output, tt.errMsg)
		})
	}
}

// TestGenerateError_PathSyntax verifies path variable syntax errors (E1-E3).
func TestGenerateError_PathSyntax(t *testing.T) {
	binary := apigenBinary(t)
	dir := fixtureDir(t)

	tests := []struct {
		fixture string
		errMsg  string
	}{
		{"empty_path_var.yaml", "empty path variable"},
		{"leading_dot_path_var.yaml", "malformed path variable"},
		{"trailing_dot_path_var.yaml", "malformed path variable"},
	}

	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			output, exitCode := runGenerate(t, binary, filepath.Join(dir, tt.fixture))
			assertExitNonZero(t, tt.fixture, exitCode)
			assertErrorContains(t, tt.fixture, output, tt.errMsg)
		})
	}
}

// TestGenerateError_Plugin verifies plugin validation errors (F1).
func TestGenerateError_Plugin(t *testing.T) {
	binary := apigenBinary(t)
	dir := fixtureDir(t)

	output, exitCode := runGenerate(t, binary, filepath.Join(dir, "unknown_js_plugin.yaml"))
	assertExitNonZero(t, "unknown_js_plugin.yaml", exitCode)
	assertErrorContains(t, "unknown_js_plugin.yaml", output, "unknown JS plugin")
}
