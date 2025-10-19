package diff

import (
	"bytes"
	"fmt"
	"strings"
)

// Differ handles manifest diffing
type Differ struct{}

// NewDiffer creates a new differ
func NewDiffer() *Differ {
	return &Differ{}
}

// Diff compares two manifests and returns a unified diff
func (d *Differ) Diff(base, head []byte) (string, error) {
	// Use simple diff implementation
	return d.simpleDiff(base, head)
}

// simpleDiff creates a simple unified-style diff
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

// HasChanges checks if there are any changes between base and head
func (d *Differ) HasChanges(base, head []byte) (bool, error) {
	return !bytes.Equal(base, head), nil
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
