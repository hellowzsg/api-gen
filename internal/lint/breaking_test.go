package lint

import (
	"testing"
)

func TestShouldRunBreaking_NoBSR(t *testing.T) {
	hasBSR := false
	if ShouldRunBreaking(hasBSR) {
		t.Error("ShouldRunBreaking(false) = true, want false")
	}
}

func TestShouldRunBreaking_HasBSR(t *testing.T) {
	hasBSR := true
	if !ShouldRunBreaking(hasBSR) {
		t.Error("ShouldRunBreaking(true) = false, want true")
	}
}
