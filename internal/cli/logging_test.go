package cli

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hellowzsg/api-gen/internal/build"
)

// swapSlog installs a buffer-backed default logger at the given level and
// returns the buffer plus a restore func.
func swapSlog(t *testing.T, level slog.Level) (*bytes.Buffer, func()) {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: level})))
	return &buf, func() { slog.SetDefault(prev) }
}

const loggingFixtureYAML = `
syntax: v1
name: test.book
import_protos:
  - path: "proto/**/*.proto"
settings:
  go_repo: github.com/acme/test
  out:
    proto: generated/proto
    go: generated/go
entities:
  - name: book
    key: { type_: BookId }
    create: {}
    resources:
      - name: meta
        type_: BookMeta
        version: { kind: NONE }
        reader: {}
services:
  - name: LibraryService
    entities:
      - name: book
`

const loggingFixtureProto = `
syntax = "proto3";
package test.book;
message BookId { string id = 1; }
message BookMeta { string title = 1; }
`

// TestLogging_PipelineEmitsKeyStages verifies that Prepare emits structured
// logs at the key pipeline stages when Info level is enabled.
func TestLogging_PipelineEmitsKeyStages(t *testing.T) {
	buf, restore := swapSlog(t, slog.LevelInfo)
	defer restore()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "api.yaml"), loggingFixtureYAML)
	writeTestFile(t, filepath.Join(dir, "proto", "book.proto"), loggingFixtureProto)

	if _, err := Prepare(context.Background(), filepath.Join(dir, "api.yaml")); err != nil {
		t.Fatalf("Prepare failed: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"config parsed", "dep fetch done", "protos resolved", "IR built"} {
		if !strings.Contains(out, want) {
			t.Errorf("pipeline log missing %q, got:\n%s", want, out)
		}
	}
}

// TestLogging_DefaultSilent verifies that no Info-level logs are emitted at
// the default (Warn) level — CLI output stays silent.
func TestLogging_DefaultSilent(t *testing.T) {
	buf, restore := swapSlog(t, slog.LevelWarn)
	defer restore()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "api.yaml"), loggingFixtureYAML)
	writeTestFile(t, filepath.Join(dir, "proto", "book.proto"), loggingFixtureProto)

	if _, err := Prepare(context.Background(), filepath.Join(dir, "api.yaml")); err != nil {
		t.Fatalf("Prepare failed: %v", err)
	}
	if buf.Len() > 0 {
		t.Errorf("expected silent output at Warn level, got:\n%s", buf.String())
	}
}

// TestLogging_CompileEmitsPluginEvents verifies that each plugin execution
// emits start/done log events including the plugin name.
func TestLogging_CompileEmitsPluginEvents(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "plugins.log")
	t.Setenv("FAKE_PLUGIN_LOG", logPath)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if err := os.WriteFile(filepath.Join(dir, "protoc-gen-fake-x"), []byte("#!/bin/sh\ncat > /dev/null\n"), 0755); err != nil {
		t.Fatal(err)
	}

	buf, restore := swapSlog(t, slog.LevelInfo)
	defer restore()

	specs := []build.PluginSpec{{Name: "protoc-gen-fake-x", OutDir: filepath.Join(dir, "out"), Parameter: "p=1"}}
	if err := build.Compile(context.Background(), nil, nil, specs); err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "plugin") || !strings.Contains(out, "protoc-gen-fake-x") {
		t.Errorf("plugin event log missing plugin name, got:\n%s", out)
	}
	if !strings.Contains(out, "duration") {
		t.Errorf("plugin done log should include duration, got:\n%s", out)
	}
}

// TestInitLogging verifies the verbose flag / env var wiring: initLogging
// must enable Info output on stderr only when requested.
func TestInitLogging(t *testing.T) {
	// verbose=false → Info suppressed.
	initLogging(false)
	if !slog.Default().Enabled(context.Background(), slog.LevelWarn) {
		t.Error("default logger should have Warn enabled")
	}
	if slog.Default().Enabled(context.Background(), slog.LevelInfo) {
		t.Error("initLogging(false) should not enable Info")
	}
	// verbose=true → Info enabled.
	initLogging(true)
	if !slog.Default().Enabled(context.Background(), slog.LevelInfo) {
		t.Error("initLogging(true) should enable Info")
	}
	// env override → Debug enabled.
	t.Setenv("APIGEN_LOG_LEVEL", "debug")
	initLogging(false)
	if !slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		t.Error("APIGEN_LOG_LEVEL=debug should enable Debug")
	}
	// Root command exposes a persistent -v flag.
	root := NewRoot()
	if root.PersistentFlags().Lookup("verbose") == nil {
		t.Error("root command missing persistent -v/--verbose flag")
	}
}
