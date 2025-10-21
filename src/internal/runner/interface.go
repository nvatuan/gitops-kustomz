package runner

import (
	"github.com/gh-nvat/gitops-kustomz/src/pkg/config"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/diff"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/kustomize"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/models"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/policy"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/template"
)

type Runner struct {
	RunMode  string
	Instance RunnerInterface
}

type RunnerInterface interface {
	// Initialize the runner with necessary context and data
	Initialize() error

	// Process a single environment and return the result
	ProcessEnvironment(environment string) (*models.EnvironmentResult, error)

	// Build manifests for a specific environment
	BuildManifests(environment string) (*models.BuildResult, error)

	// Generate the final report
	GenerateReport(envResults map[string]models.EnvironmentResult) (*models.ReportResult, error)

	// Output the results (GitHub post, local file, etc.)
	OutputResults(report *models.ReportResult) (*models.OutputResult, error)
}

// BaseRunner contains common dependencies used by both GitHub and Local runners
type BaseRunner struct {
	// Common dependencies
	Builder      *kustomize.Builder
	Differ       *diff.Differ
	Evaluator    *policy.Evaluator
	Reporter     *policy.Reporter
	Renderer     *template.Renderer
	PolicyConfig *config.ComplianceConfig
}
