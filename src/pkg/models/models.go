package models

import (
	"github.com/gh-nvat/gitops-kustomz/src/pkg/config"
)

// EnvironmentResult represents the result of processing a single environment
type EnvironmentResult struct {
	Environment  string
	BaseManifest []byte
	HeadManifest []byte
	DiffData     config.EnvironmentDiff
	EvalResult   *config.EvaluationResult
	PolicyReport *config.PolicyReportData
	Enforcement  *config.EnforcementResult
}

// ManifestPair represents a pair of base and head manifests
type ManifestPair struct {
	BaseManifest []byte
	HeadManifest []byte
}

// ProcessResult represents the result of processing all environments
type ProcessResult struct {
	Environments []EnvironmentResult
	HasErrors    bool
	ErrorCount   int
}

// BuildResult represents the result of building manifests
type BuildResult struct {
	Manifests ManifestPair
	DiffData  config.EnvironmentDiff
	Error     error
}

// PolicyResult represents the result of policy evaluation
type PolicyResult struct {
	EvalResult   *config.EvaluationResult
	PolicyReport *config.PolicyReportData
	Enforcement  *config.EnforcementResult
	Error        error
}

// ReportResult represents the result of report generation
type ReportResult struct {
	MultiEnvData     config.MultiEnvCommentData
	RenderedMarkdown string
	Error            error
}

// OutputResult represents the result of output operations
type OutputResult struct {
	Success bool
	Message string
	Error   error
}

// GitHubData represents GitHub-specific data
type GitHubData struct {
	PRInfo   *config.PullRequest
	Comments []*config.Comment
}

// LocalData represents Local-specific data
type LocalData struct {
	BeforePath string
	AfterPath  string
	OutputDir  string
}
