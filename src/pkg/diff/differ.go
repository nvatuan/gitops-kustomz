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
	Diff(before, after []byte) (string, error)
	DiffText(before, after string) (string, error)
}

// Differ handles manifest diffing
type Differ struct{}

// Ensure Differ implements ManifestDiffer
var _ ManifestDiffer = (*Differ)(nil)

// NewDiffer creates a new differ
func NewDiffer() *Differ {
	return &Differ{}
}

// Convert text to bytes and call Diff
func (d *Differ) DiffText(before, after string) (string, error) {
	return d.Diff([]byte(before), []byte(after))
}

// Diff compares two manifests and returns a unified diff
func (d *Differ) Diff(before, after []byte) (string, error) {
	// Use system diff -u for unified diff with context
	return d.unifiedDiff(before, after)
}

// unifiedDiff uses system diff -u command for proper unified diff with context
func (d *Differ) unifiedDiff(before, after []byte) (string, error) {
	if bytes.Equal(before, after) {
		return "", nil
	}

	// Create temp files for diff
	beforeFile, err := os.CreateTemp("", "before-*.yaml")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		beforeFile.Close()
		os.Remove(beforeFile.Name())
	}()

	afterFile, err := os.CreateTemp("", "after-*.yaml")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		afterFile.Close()
		os.Remove(afterFile.Name())
	}()

	// Write manifests to temp files
	if _, err := beforeFile.Write(before); err != nil {
		return "", fmt.Errorf("failed to write base manifest: %w", err)
	}
	if err := beforeFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close base file: %w", err)
	}

	if _, err := afterFile.Write(after); err != nil {
		return "", fmt.Errorf("failed to write after manifest: %w", err)
	}
	if err := afterFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close after file: %w", err)
	}

	// Run diff -u
	cmd := exec.Command("diff", "-u", beforeFile.Name(), afterFile.Name())
	output, err := cmd.CombinedOutput()

	// diff returns exit code 1 when files differ (not an error)
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// This is expected when files differ
		} else {
			return "", fmt.Errorf("diff command failed: %w", err)
		}
	}

	// Replace temp file names with "before" and "after"
	diffOutput := string(output)
	diffOutput = strings.ReplaceAll(diffOutput, beforeFile.Name(), "before")
	diffOutput = strings.ReplaceAll(diffOutput, afterFile.Name(), "after")

	return diffOutput, nil
}
