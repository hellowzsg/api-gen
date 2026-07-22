package cli

import (
	"log/slog"
	"os"
	"strings"
)

// initLogging configures the default slog logger from the -v/--verbose flag
// and the APIGEN_LOG_LEVEL env var (debug/info/warn/error, env wins).
// Logs go to stderr only — stdout is reserved for command results
// (e.g. `entity list`) so pipelines are never polluted.
func initLogging(verbose bool) {
	level := slog.LevelWarn
	if verbose {
		level = slog.LevelInfo
	}
	if v := os.Getenv("APIGEN_LOG_LEVEL"); v != "" {
		switch strings.ToLower(v) {
		case "debug":
			level = slog.LevelDebug
		case "info":
			level = slog.LevelInfo
		case "warn", "warning":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		}
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))
}
