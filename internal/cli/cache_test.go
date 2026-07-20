package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultCacheDir_APIGEN_CACHE_DIR(t *testing.T) {
	t.Setenv("APIGEN_CACHE_DIR", "/custom/cache")
	got := defaultCacheDir()
	if got != "/custom/cache" {
		t.Errorf("defaultCacheDir() = %q, want /custom/cache", got)
	}
}

func TestDefaultCacheDir_XDG(t *testing.T) {
	t.Setenv("APIGEN_CACHE_DIR", "")
	t.Setenv("XDG_CACHE_HOME", "/xdg/cache")
	got := defaultCacheDir()
	if got != filepath.Join("/xdg/cache", "apigen") {
		t.Errorf("defaultCacheDir() = %q, want /xdg/cache/apigen", got)
	}
}

func TestDefaultCacheDir_Default(t *testing.T) {
	t.Setenv("APIGEN_CACHE_DIR", "")
	t.Setenv("XDG_CACHE_HOME", "")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home dir")
	}
	got := defaultCacheDir()
	want := filepath.Join(home, ".cache", "apigen")
	if got != want {
		t.Errorf("defaultCacheDir() = %q, want %q", got, want)
	}
}
