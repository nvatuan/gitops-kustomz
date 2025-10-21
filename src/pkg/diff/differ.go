package diff

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ManifestDiffer defines the interface for comparing Kubernetes manifests
type ManifestDiffer interface {
	// Diff compares two manifests and returns a unified diff
	Diff(base, head []byte) (string, error)
}

// Differ handles manifest diffing
type Differ struct{}

// Ensure Differ implements ManifestDiffer
var _ ManifestDiffer = (*Differ)(nil)

// NewDiffer creates a new differ
func NewDiffer() *Differ {
	return &Differ{}
}

// Diff compares two manifests and returns a unified diff
func (d *Differ) Diff(base, head []byte) (string, error) {
	// Use system diff -u for unified diff with context
	return d.unifiedDiff(base, head)
}

// unifiedDiff uses system diff -u command for proper unified diff with context
func (d *Differ) unifiedDiff(base, head []byte) (string, error) {
	if bytes.Equal(base, head) {
		return "", nil
	}

	// Create temp files for diff
	baseFile, err := os.CreateTemp("", "base-*.yaml")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		_ = os.Remove(baseFile.Name())
	}()
	defer func() {
		_ = baseFile.Close()
	}()

	headFile, err := os.CreateTemp("", "head-*.yaml")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		_ = os.Remove(headFile.Name())
	}()
	defer func() {
		_ = headFile.Close()
	}()

	// Write manifests to temp files
	if _, err := baseFile.Write(base); err != nil {
		return "", fmt.Errorf("failed to write base manifest: %w", err)
	}
	if err := baseFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close base file: %w", err)
	}

	if _, err := headFile.Write(head); err != nil {
		return "", fmt.Errorf("failed to write head manifest: %w", err)
	}
	if err := headFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close head file: %w", err)
	}

	// Run diff -u
	cmd := exec.Command("diff", "-u", baseFile.Name(), headFile.Name())
	output, err := cmd.CombinedOutput()

	// diff returns exit code 1 when files differ (not an error)
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// This is expected when files differ
		} else {
			return "", fmt.Errorf("diff command failed: %w", err)
		}
	}

	// Replace temp file names with "base" and "head"
	diffOutput := string(output)
	diffOutput = strings.ReplaceAll(diffOutput, baseFile.Name(), "base")
	diffOutput = strings.ReplaceAll(diffOutput, headFile.Name(), "head")

	return diffOutput, nil
}

// simpleDiff creates a simple unified-style diff (fallback - kept for future use)
// nolint:unused
func (d *Differ) simpleDiff(base, head []byte) (string, error) {
	if bytes.Equal(base, head) {
		return "", nil
	}

	var result strings.Builder
	result.WriteString("--- base\n")
	result.WriteString("+++ head\n")

	baseLines := strings.Split(string(base), "\n")
	headLines := strings.Split(string(head), "\n")

	// Simple line-by-line comparison
	// For production, consider using a proper diff algorithm like Myers or patience diff
	maxLen := len(baseLines)
	if len(headLines) > maxLen {
		maxLen = len(headLines)
	}

	for i := 0; i < maxLen; i++ {
		var baseLine, headLine string
		if i < len(baseLines) {
			baseLine = baseLines[i]
		}
		if i < len(headLines) {
			headLine = headLines[i]
		}

		if baseLine != headLine {
			if baseLine != "" {
				result.WriteString("- " + baseLine + "\n")
			}
			if headLine != "" {
				result.WriteString("+ " + headLine + "\n")
			}
		}
	}

	return result.String(), nil
}

// FormatForMarkdown formats the diff for display in markdown
func (d *Differ) FormatForMarkdown(diff string, maxLines int) string {
	if diff == "" {
		return "_No changes detected_"
	}

	lines := strings.Split(diff, "\n")
	lineCount := len(lines)

	// If diff is large, wrap in <details>
	if lineCount > maxLines {
		var result strings.Builder
		result.WriteString(fmt.Sprintf("<details>\n<summary>📝 Diff (%d lines - click to expand)</summary>\n\n", lineCount))
		result.WriteString("```diff\n")
		result.WriteString(diff)
		result.WriteString("\n```\n")
		result.WriteString("</details>")
		return result.String()
	}

	// Small diff, show directly
	var result strings.Builder
	result.WriteString("```diff\n")
	result.WriteString(diff)
	result.WriteString("\n```")
	return result.String()
}
