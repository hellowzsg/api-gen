package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

// commitDir atomically replaces `dst` with the contents of `src`.
//
// Strategy (all renames happen within the same parent directory to guarantee
// cross-device rename safety):
//  1. If `dst` exists, rename it to `dst + ".bak"`.
//  2. Rename `src` to `dst`.
//  3. On success, remove the `.bak` directory.
//  4. On failure at step 2, roll back: rename `.bak` back to `dst`.
//
// If `dst` does not exist, `src` is simply renamed to `dst` (no backup needed).
// `src` is always removed on failure. The caller is responsible for creating
// `src` and populating it before calling commitDir.
func commitDir(src, dst string) (retErr error) {
	bak := dst + ".bak"

	// Step 1: move existing dst aside (if present).
	dstExists := true
	if _, err := os.Stat(dst); err != nil {
		if os.IsNotExist(err) {
			dstExists = false
		} else {
			return fmt.Errorf("stat %q: %w", dst, err)
		}
	}
	if dstExists {
		// Remove a stale .bak from a previous crashed run.
		_ = os.RemoveAll(bak)
		if err := os.Rename(dst, bak); err != nil {
			return fmt.Errorf("backup %q → %q: %w", dst, bak, err)
		}
		// Rollback: if anything below fails, restore the backup.
		defer func() {
			if retErr != nil {
				// Best-effort restore; remove any partial dst first.
				_ = os.RemoveAll(dst)
				_ = os.Rename(bak, dst)
			}
		}()
	}

	// Step 2: promote src → dst.
	if err := os.Rename(src, dst); err != nil {
		return fmt.Errorf("promote %q → %q: %w", src, dst, err)
	}

	// Step 3: discard the backup.
	if dstExists {
		if err := os.RemoveAll(bak); err != nil {
			// Non-fatal: the new dst is already in place; leaving a stale
			// .bak is harmless (it will be cleaned on the next run).
			return nil
		}
	}
	return nil
}

// newStagingDir creates a temporary directory that is a sibling of `target`.
// Sibling placement guarantees the staging dir is on the same filesystem as
// `target`, so os.Rename (used by commitDir) works atomically without
// cross-device copy.
func newStagingDir(target string) (string, error) {
	parent := filepath.Dir(target)
	if err := os.MkdirAll(parent, 0755); err != nil {
		return "", fmt.Errorf("create parent %q: %w", parent, err)
	}
	d, err := os.MkdirTemp(parent, ".apigen-stage-*")
	if err != nil {
		return "", fmt.Errorf("create staging dir: %w", err)
	}
	return d, nil
}
