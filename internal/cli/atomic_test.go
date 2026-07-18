package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCommitDir_NewTarget(t *testing.T) {
	parent := t.TempDir()
	src := filepath.Join(parent, "src")
	if err := os.MkdirAll(src, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "a.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(parent, "dst")
	if err := commitDir(src, dst); err != nil {
		t.Fatalf("commitDir: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dst, "a.txt"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("got %q, want hello", data)
	}
	// src should be gone (renamed to dst).
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("src should not exist after commit")
	}
}

func TestCommitDir_ReplacesExisting(t *testing.T) {
	parent := t.TempDir()
	// Pre-existing dst with old content.
	dst := filepath.Join(parent, "dst")
	if err := os.WriteFile(filepath.Join(dst, "old.txt"), []byte("old"), 0644); err != nil {
		// MkdirAll needed first
		_ = os.MkdirAll(dst, 0755)
		if err := os.WriteFile(filepath.Join(dst, "old.txt"), []byte("old"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	// New src with new content.
	src := filepath.Join(parent, "src")
	if err := os.MkdirAll(src, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "new.txt"), []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := commitDir(src, dst); err != nil {
		t.Fatalf("commitDir: %v", err)
	}
	// dst should have new content, not old.
	if _, err := os.Stat(filepath.Join(dst, "old.txt")); !os.IsNotExist(err) {
		t.Error("old file should be gone after commit")
	}
	data, err := os.ReadFile(filepath.Join(dst, "new.txt"))
	if err != nil {
		t.Fatalf("read new: %v", err)
	}
	if string(data) != "new" {
		t.Errorf("got %q, want new", data)
	}
	// No .bak residue.
	if _, err := os.Stat(dst + ".bak"); !os.IsNotExist(err) {
		t.Error(".bak should be removed after successful commit")
	}
}

func TestCommitDir_RollbackOnFailure(t *testing.T) {
	parent := t.TempDir()
	// Pre-existing dst with old content.
	dst := filepath.Join(parent, "dst")
	if err := os.MkdirAll(dst, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dst, "old.txt"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	// src that does NOT exist — commitDir should fail at the promote step
	// (rename nonexistent src → dst) and roll back the backup.
	src := filepath.Join(parent, "nonexistent-src")
	err := commitDir(src, dst)
	if err == nil {
		t.Fatal("expected error for nonexistent src")
	}
	// dst should still have old content (rolled back).
	data, err := os.ReadFile(filepath.Join(dst, "old.txt"))
	if err != nil {
		t.Fatalf("old file should survive rollback: %v", err)
	}
	if string(data) != "old" {
		t.Errorf("got %q, want old (rollback failed)", data)
	}
	// No .bak residue after rollback.
	if _, err := os.Stat(dst + ".bak"); !os.IsNotExist(err) {
		t.Error(".bak should be removed after rollback")
	}
}

func TestNewStagingDir_SiblingOfTarget(t *testing.T) {
	parent := t.TempDir()
	target := filepath.Join(parent, "out", "proto")
	d, err := newStagingDir(target)
	if err != nil {
		t.Fatalf("newStagingDir: %v", err)
	}
	defer os.RemoveAll(d)
	// Staging dir must be a sibling of target (same parent) for rename safety.
	if filepath.Dir(d) != filepath.Dir(target) {
		t.Errorf("staging %q not sibling of target %q", d, target)
	}
}
