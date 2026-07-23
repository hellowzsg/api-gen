package positive

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixtureSimpleDir returns the path to the simple fixture directory.
func fixtureSimpleDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	return filepath.Join(wd, "..", "fixtures", "simple")
}

// TestSimpleGRPC_PureGRPC verifies that the simple fixture (P0, no HTTP config)
// generates proto without HTTP annotations, without annotations import, and only gRPC service.
func TestSimpleGRPC_PureGRPC(t *testing.T) {
	dir := fixtureSimpleDir(t)
	protoPath := filepath.Join(dir, "generated", "proto", "config_service", "config_service.proto")
	proto := mustReadFile(t, protoPath)

	t.Run("no google.api.http annotation", func(t *testing.T) {
		if strings.Contains(proto, "google.api.http") {
			t.Errorf("simple fixture proto should not contain google.api.http annotation, but does:\n%s", proto)
		}
	})

	t.Run("no google/api/annotations.proto import", func(t *testing.T) {
		if strings.Contains(proto, `import "google/api/annotations.proto"`) {
			t.Errorf("simple fixture proto should not import google/api/annotations.proto, but does")
		}
	})

	t.Run("only gRPC service definition (no HTTP gateway)", func(t *testing.T) {
		// Verify service exists
		if !strings.Contains(proto, "service ConfigService") {
			t.Error("proto should contain 'service ConfigService'")
		}
		// Verify RPCs exist without HTTP options
		assertContains(t, "config_service.proto", proto, "rpc GetConfigEntry(")
		assertContains(t, "config_service.proto", proto, "rpc UpdateConfigEntry(")
		// No pb.gw.go should be generated for simple fixture
		gwFiles, _ := filepath.Glob(filepath.Join(dir, "generated", "go", "**", "*.pb.gw.go"))
		if len(gwFiles) > 0 {
			t.Errorf("simple fixture should not generate .pb.gw.go files, found: %v", gwFiles)
		}
	})

	t.Run("no openapi directory generated", func(t *testing.T) {
		openapiDir := filepath.Join(dir, "generated", "openapi")
		if _, err := os.Stat(openapiDir); !os.IsNotExist(err) {
			t.Errorf("simple fixture should not generate openapi/ directory, but it exists")
		}
	})

	t.Run("NONE version has no version field", func(t *testing.T) {
		// ConfigEntry resource has version.kind=NONE, so GetConfigEntryResponse should not have version field
		idx := strings.Index(proto, "message GetConfigEntryResponse")
		if idx < 0 {
			t.Fatal("GetConfigEntryResponse message not found")
		}
		end := strings.Index(proto[idx:], "\n\n")
		if end < 0 {
			end = len(proto) - idx
		}
		block := proto[idx : idx+end]
		if strings.Contains(block, "version") {
			t.Errorf("GetConfigEntryResponse should not have version field (NONE), but contains: %s", block)
		}
	})
}
