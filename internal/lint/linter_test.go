package lint

import (
	"strings"
	"testing"
)

func TestGenerateExemptions(t *testing.T) {
	tests := []struct {
		name       string
		hasCreate  bool
		hasDelete  bool
		hasGet     bool
		hasUpdate  bool
		hasBatchGet bool
		wantExempt []string
	}{
		{"create only", true, false, false, false, false, []string{"core::0133::response-message-name"}},
		{"delete only", false, true, false, false, false, []string{"core::0135::request-name-field"}},
		{"get only", false, false, true, false, false, []string{"core::0131::response-message-name"}},
		{"all", true, true, true, true, true, []string{"core::0131", "core::0133", "core::0134", "core::0135", "core::0231"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exemptions := GenerateExemptions(tt.hasCreate, tt.hasDelete, false, tt.hasGet, tt.hasBatchGet, tt.hasUpdate)
			for _, want := range tt.wantExempt {
				found := false
				for _, got := range exemptions {
					if strings.Contains(got, want) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("exemption containing %q not generated", want)
				}
			}
		})
	}
}

func TestFormatExemptionComment(t *testing.T) {
	c := FormatExemptionComment("core::0133::response-message-name")
	if !strings.Contains(c, "api-linter") {
		t.Error("missing api-linter keyword")
	}
	if !strings.Contains(c, "core::0133::response-message-name") {
		t.Error("missing rule name")
	}
	if !strings.Contains(c, "disabled") {
		t.Error("missing disabled")
	}
}
