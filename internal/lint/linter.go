// Package lint handles api-linter exemption generation and invocation.
package lint

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// GenerateExemptions returns the api-linter exemption rules needed.
func GenerateExemptions(hasCreate, hasDelete, hasDeleteSoft, hasGet, hasBatchGet, hasUpdate bool) []string {
	var exemptions []string
	if hasGet {
		exemptions = append(exemptions, "core::0131::response-message-name", "core::0131::request-name-field")
	}
	if hasCreate {
		exemptions = append(exemptions, "core::0133::response-message-name", "core::0133::request-parent-field", "core::0133::field-numbers")
	}
	if hasUpdate {
		exemptions = append(exemptions, "core::0134::response-message-name", "core::0134::request-unknown-fields")
	}
	if hasDelete || hasDeleteSoft {
		exemptions = append(exemptions, "core::0135::request-name-field")
	}
	if hasBatchGet {
		exemptions = append(exemptions, "core::0231::response-message-name", "core::0231::method-name")
	}
	return exemptions
}

// FormatExemptionComment formats a single exemption as a proto comment.
func FormatExemptionComment(rule string) string {
	return fmt.Sprintf("// (-- api-linter: %s=disabled --)", rule)
}

// CheckApiLinterInstalled checks if api-linter is available.
func CheckApiLinterInstalled() (string, error) {
	path, err := exec.LookPath("api-linter")
	if err != nil {
		return "", fmt.Errorf("api-linter not found in PATH: %w", err)
	}
	return path, nil
}

// RunApiLinter runs api-linter on the given proto file.
func RunApiLinter(ctx context.Context, protoPath string) (string, error) {
	path, err := CheckApiLinterInstalled()
	if err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, path, protoPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("api-linter failed: %w", err)
	}
	return string(output), nil
}

// FormatAllExemptions formats all exemptions as proto comments.
func FormatAllExemptions(exemptions []string) string {
	var sb strings.Builder
	for _, e := range exemptions {
		sb.WriteString(FormatExemptionComment(e))
		sb.WriteString("\n")
	}
	return sb.String()
}
