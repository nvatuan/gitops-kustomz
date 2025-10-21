package kustomize

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	KUSTOMIZE_BASE_DIR = "base"
	KUSTOMIZE_ENV_DIR  = "environments"
)

var (
	KUSTOMIZE_FILE_NAMES = []string{"kustomization.yaml", "kustomization.yml"}
)

// KustomizeBuilder defines the interface for building Kubernetes manifests
type KustomizeBuilder interface {
	// Build runs kustomize build on the specified path
	Build(ctx context.Context, path string) ([]byte, error)
	// ValidateServiceEnvironment checks if a service/environment combination exists
	ValidateServiceEnvironment(manifestsPath, service, environment string) error
	// GetServiceEnvironmentPath returns the path to build for a service/environment
	GetServiceEnvironmentPath(manifestsPath, service, environment string) string
}

// Builder handles kustomize builds
type Builder struct{}

// Ensure Builder implements KustomizeBuilder
var _ KustomizeBuilder = (*Builder)(nil)

// NewBuilder creates a new kustomize builder
func NewBuilder() *Builder {
	return &Builder{}
}

// Build runs kustomize build on the specified path
func (b *Builder) Build(ctx context.Context, path string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "kustomize", "build", path)
	output, err := cmd.CombinedOutput()

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

	for _, kustomizeFileName := range KUSTOMIZE_FILE_NAMES {
		baseKustomization := filepath.Join(basePath, kustomizeFileName)
		if _, err := os.Stat(baseKustomization); os.IsNotExist(err) {
			return fmt.Errorf("%s not found in base directory for service '%s'", kustomizeFileName, service)
		}
	}

	// Check if environment exists
	envPath := filepath.Join(servicePath, KUSTOMIZE_ENV_DIR, environment)
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return fmt.Errorf("environment '%s' not found for service '%s' (service may not be deployed to this environment)", environment, service)
	}

	for _, kustomizeFileName := range KUSTOMIZE_FILE_NAMES {
		envKustomization := filepath.Join(envPath, kustomizeFileName)
		if _, err := os.Stat(envKustomization); os.IsNotExist(err) {
			return fmt.Errorf("%s not found in environment '%s' for service '%s'", kustomizeFileName, environment, service)
		}
	}

	return nil
}

// GetServiceEnvironmentPath returns the path to build for a service/environment
func (b *Builder) GetServiceEnvironmentPath(manifestsPath, service, environment string) string {
	return filepath.Join(manifestsPath, service, KUSTOMIZE_ENV_DIR, environment)
}
