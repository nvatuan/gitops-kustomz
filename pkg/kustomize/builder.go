package kustomize

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	KUSTOMIZE_BASE_DIR  = "base"
	KUSTOMIZE_ENV_DIR   = "environments"
	KUSTOMIZE_FILE_NAME = "kustomization.yaml"
)

// Builder handles kustomize builds
type Builder struct{}

// NewBuilder creates a new kustomize builder
func NewBuilder() *Builder {
	return &Builder{}
}

// Build runs kustomize build on the specified path
func (b *Builder) Build(ctx context.Context, path string) ([]byte, error) {
	start := time.Now()

	cmd := exec.CommandContext(ctx, "kustomize", "build", path)
	output, err := cmd.CombinedOutput()

	duration := time.Since(start)
	if duration > 2*time.Second {
		fmt.Fprintf(os.Stderr, "Warning: kustomize build took %v (>2s target)\n", duration)
	}

	if err != nil {
		return nil, fmt.Errorf("kustomize build failed: %w\nOutput: %s", err, string(output))
	}

	return output, nil
}

// ValidateServiceEnvironment checks if a service/environment combination exists
func (b *Builder) ValidateServiceEnvironment(manifestsPath, service, environment string) error {
	// Check if service exists
	servicePath := filepath.Join(manifestsPath, service)
	if _, err := os.Stat(servicePath); os.IsNotExist(err) {
		return fmt.Errorf("service '%s' not found in %s", service, manifestsPath)
	}

	// Check if base exists
	basePath := filepath.Join(servicePath, KUSTOMIZE_BASE_DIR)
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		return fmt.Errorf("base directory not found for service '%s'", service)
	}

	baseKustomization := filepath.Join(basePath, KUSTOMIZE_FILE_NAME)
	if _, err := os.Stat(baseKustomization); os.IsNotExist(err) {
		return fmt.Errorf("kustomization.yaml not found in base directory for service '%s'", service)
	}

	// Check if environment exists
	envPath := filepath.Join(servicePath, KUSTOMIZE_ENV_DIR, environment)
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return fmt.Errorf("environment '%s' not found for service '%s' (service may not be deployed to this environment)", environment, service)
	}

	envKustomization := filepath.Join(envPath, KUSTOMIZE_FILE_NAME)
	if _, err := os.Stat(envKustomization); os.IsNotExist(err) {
		return fmt.Errorf("kustomization.yaml not found in environment '%s' for service '%s'", environment, service)
	}

	return nil
}

// GetServiceEnvironmentPath returns the path to build for a service/environment
func (b *Builder) GetServiceEnvironmentPath(manifestsPath, service, environment string) string {
	return filepath.Join(manifestsPath, service, KUSTOMIZE_ENV_DIR, environment)
}
