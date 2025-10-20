package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ConfigLoader defines the interface for loading configuration files
type ConfigLoader interface {
	// LoadComplianceConfig loads the compliance configuration from a YAML file
	LoadComplianceConfig(path string) (*ComplianceConfig, error)
	// ValidateComplianceConfig validates the compliance configuration
	ValidateComplianceConfig(config *ComplianceConfig) error
}

// Loader handles loading configuration files
type Loader struct{}

// Ensure Loader implements ConfigLoader
var _ ConfigLoader = (*Loader)(nil)

// NewLoader creates a new configuration loader
func NewLoader() *Loader {
	return &Loader{}
}

// LoadComplianceConfig loads the compliance configuration from a YAML file
func (l *Loader) LoadComplianceConfig(path string) (*ComplianceConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read compliance config: %w", err)
	}

	var config ComplianceConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse compliance config: %w", err)
	}

	return &config, nil
}

// ValidateComplianceConfig validates the compliance configuration
func (l *Loader) ValidateComplianceConfig(config *ComplianceConfig) error {
	if len(config.Policies) == 0 {
		return fmt.Errorf("no policies defined in compliance config")
	}

	for id, policy := range config.Policies {
		if policy.Name == "" {
			return fmt.Errorf("policy %s: name is required", id)
		}
		if policy.Type == "" {
			return fmt.Errorf("policy %s: type is required", id)
		}
		if policy.Type != "opa" {
			return fmt.Errorf("policy %s: unsupported type %s (only 'opa' is supported)", id, policy.Type)
		}
		if policy.FilePath == "" {
			return fmt.Errorf("policy %s: filePath is required", id)
		}

		// Validate enforcement dates are in order if set
		if policy.Enforcement.InEffectAfter != nil && policy.Enforcement.IsWarningAfter != nil {
			if policy.Enforcement.IsWarningAfter.Before(*policy.Enforcement.InEffectAfter) {
				return fmt.Errorf("policy %s: isWarningAfter cannot be before inEffectAfter", id)
			}
		}
		if policy.Enforcement.IsWarningAfter != nil && policy.Enforcement.IsBlockingAfter != nil {
			if policy.Enforcement.IsBlockingAfter.Before(*policy.Enforcement.IsWarningAfter) {
				return fmt.Errorf("policy %s: isBlockingAfter cannot be before isWarningAfter", id)
			}
		}
	}

	return nil
}
