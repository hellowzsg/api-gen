package lint

import (
	"context"
	"fmt"
	"os/exec"
)

// ShouldRunBreaking returns true if buf breaking should run (only with BSR deps).
func ShouldRunBreaking(hasBSRDeps bool) bool {
	return hasBSRDeps
}

// RunBufBreaking runs buf breaking check.
func RunBufBreaking(ctx context.Context, workDir, againstRef string) error {
	path, err := exec.LookPath("buf")
	if err != nil {
		return fmt.Errorf("buf CLI not found: %w", err)
	}
	cmd := exec.CommandContext(ctx, path, "breaking", "--against", againstRef)
	cmd.Dir = workDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("buf breaking failed: %w\n%s", err, string(output))
	}
	return nil
}
