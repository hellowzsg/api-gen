package cli

import (
	"os"
	"path/filepath"
	"runtime"
)

// defaultCacheDir returns the apigen dependency cache directory.
//
// Resolution order (mirrors buf's BUF_CACHE_DIR behaviour):
//  1. $APIGEN_CACHE_DIR — explicit override
//  2. $XDG_CACHE_HOME/apigen — XDG base directory spec
//  3. OS default:
//     - macOS/Linux: $HOME/.cache/apigen
//     - Windows:     %LocalAppData%/apigen
//
// The returned path is the cache ROOT; resolvers append a version directory
// (e.g. ~/.cache/apigen/v1/module-proxy/...) so that bumping
// dep.CacheVersion invalidates old caches without manual cleanup.
//
// The returned path is NOT created; callers (resolvers) create it on demand.
func defaultCacheDir() string {
	if v := os.Getenv("APIGEN_CACHE_DIR"); v != "" {
		return v
	}
	if v := os.Getenv("XDG_CACHE_HOME"); v != "" {
		return filepath.Join(v, "apigen")
	}
	if runtime.GOOS == "windows" {
		if v := os.Getenv("LocalAppData"); v != "" {
			return filepath.Join(v, "apigen")
		}
		// Fallback for edge cases where LocalAppData is unset.
		return filepath.Join(os.TempDir(), "apigen")
	}
	// macOS and Linux: ~/.cache/apigen
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(os.TempDir(), "apigen")
	}
	return filepath.Join(home, ".cache", "apigen")
}
