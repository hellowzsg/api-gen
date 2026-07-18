package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestE2EJSStub verifies that apigen build with plugins.js: [es] triggers
// protoc-gen-es invocation and generates TypeScript stub files.
//
// This test has two sub-tests:
//  1. PluginInvocationTriggered: when plugins.js: [es] is declared, the build
//     flow attempts to call protoc-gen-es. If the plugin is not installed,
//     build fails fast with a "protoc-gen-es" error — proving the generateJS
//     path was correctly triggered.
//  2. TSFilesGenerated: when protoc-gen-es is installed and build succeeds,
//     generated/js/ contains *_pb.ts files with valid TypeScript module
//     syntax (import/export keywords) and service definitions.
func TestE2EJSStub(t *testing.T) {
	// Determine whether protoc-gen-es is available in PATH.
	_, pluginErr := exec.LookPath("protoc-gen-es")
	pluginAvailable := pluginErr == nil

	t.Run("PluginInvocationTriggered", func(t *testing.T) {
		// Build the project from the main repo so the cmd/apigen binary
		// is within the module. We run `go run ./cmd/apigen build` from
		// the repo root.
		cmd := exec.Command("go", "run", "./cmd/apigen", "build", "-f", "examples/book/api.yaml")
		cmd.Dir = "../.."
		output, err := cmd.CombinedOutput()
		out := string(output)

		if pluginAvailable {
			// Plugin is installed: build should succeed.
			if err != nil {
				t.Fatalf("apigen build failed (plugin available): %v\noutput:\n%s", err, out)
			}
		} else {
			// Plugin NOT installed: build should fail fast with a
			// protoc-gen-es error, proving the generateJS path was
			// triggered by the plugins.js: [es] declaration.
			if err == nil {
				t.Skip("apigen build succeeded but protoc-gen-es not in PATH; " +
					"cannot verify plugin invocation trigger (plugin may be " +
					"available via another mechanism)")
			}
			if !strings.Contains(out, "protoc-gen-es") {
				t.Errorf("build failed but error does not mention protoc-gen-es;\noutput:\n%s", out)
			}
		}
	})

	t.Run("TSFilesGenerated", func(t *testing.T) {
		if !pluginAvailable {
			t.Skip("protoc-gen-es not installed; skipping TS file generation check")
		}

		// NOTE: This sub-test relies on the side effect of
		// PluginInvocationTriggered having run `apigen build` above (Go
		// runs sub-tests in declaration order). If the build succeeded
		// with the plugin available, generated/js/ should contain the
		// TS stubs. If you refactor test ordering or parallelize, ensure
		// `apigen build` runs before this check.

		// Walk generated/js/ and collect *_pb.ts files.
		jsDir := filepath.Join("generated", "js")
		var tsFiles []string
		err := filepath.Walk(jsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(path, "_pb.ts") {
				tsFiles = append(tsFiles, path)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk generated/js/: %v (did you run `apigen build`?)", err)
		}
		if len(tsFiles) == 0 {
			t.Fatalf("no *_pb.ts files found under %s (did you run `apigen build`?)", jsDir)
		}

		// At least one TS file should contain a service definition
		// (LibraryService) and valid TypeScript module syntax (import/export).
		foundService := false
		foundModuleSyntax := false
		for _, f := range tsFiles {
			data, err := os.ReadFile(f)
			if err != nil {
				t.Errorf("read %s: %v", f, err)
				continue
			}
			content := string(data)
			if strings.Contains(content, "LibraryService") {
				foundService = true
			}
			if strings.Contains(content, "import ") || strings.Contains(content, "export ") {
				foundModuleSyntax = true
			}
		}
		if !foundService {
			t.Errorf("no TS file contains 'LibraryService'; files: %v", tsFiles)
		}
		if !foundModuleSyntax {
			t.Errorf("no TS file contains import/export; files: %v", tsFiles)
		}
	})
}
