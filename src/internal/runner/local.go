package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gh-nvat/gitops-kustomz/src/pkg/config"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/models"
)

type RunnerLocal struct {
	BaseRunner
	ctx     context.Context
	options *Options
}

func NewRunnerLocal(
	ctx context.Context,
	options *Options,
	baseRunner *BaseRunner,
) (*RunnerLocal, error) {
	return &RunnerLocal{
		BaseRunner: *baseRunner,
		ctx:        ctx,
		options:    options,
	}, nil
}

func (r *RunnerLocal) Initialize() error {
	return nil
}

// ProcessEnvironment processes a single environment and returns the result
func (r *RunnerLocal) ProcessEnvironment(environment string) (*models.EnvironmentResult, error) {
	// Build manifests for this environment
	buildResult, err := r.BuildManifests(environment)
	if err != nil {
		return nil, fmt.Errorf("failed to build manifests: %w", err)
	}

	// Generate diff
	diffContent, err := r.Differ.Diff(buildResult.Manifests.BaseManifest, buildResult.Manifests.HeadManifest)
	if err != nil {
		return nil, fmt.Errorf("failed to generate diff: %w", err)
	}

	// Count only actual changed lines (lines starting with + or -)
	addedLines, deletedLines := 0, 0
	for _, line := range strings.Split(diffContent, "\n") {
		if strings.HasPrefix(line, "+ ") {
			addedLines++
		}
		if strings.HasPrefix(line, "- ") {
			deletedLines++
		}
	}
	hasChanges := (addedLines+deletedLines > 0)
	envDiffData := config.EnvironmentDiff{
		Environment:      environment,
		HasChanges:       hasChanges,
		Content:          diffContent,
		LineCount:        addedLines + deletedLines,
		AddedLineCount:   addedLines,
		DeletedLineCount: deletedLines,
	}

	// Evaluate policies
	evalResult, err := r.Evaluator.Evaluate(r.ctx, buildResult.Manifests.HeadManifest, r.PolicyConfig, r.options.PoliciesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate policies: %w", err)
	}

	// No overrides in local mode
	enforcement := r.Evaluator.Enforce(evalResult, nil)

	// Generate report
	policyReport := r.Reporter.GenerateReport(evalResult)

	return &models.EnvironmentResult{
		Environment:  environment,
		BaseManifest: buildResult.Manifests.BaseManifest,
		HeadManifest: buildResult.Manifests.HeadManifest,
		DiffData:     envDiffData,
		EvalResult:   evalResult,
		PolicyReport: policyReport,
		Enforcement:  enforcement,
	}, nil
}

// BuildManifests builds manifests for a specific environment
func (r *RunnerLocal) BuildManifests(environment string) (*models.BuildResult, error) {
	baseManifest, headManifest, err := r.buildManifestsFromLocal(environment)
	if err != nil {
		return &models.BuildResult{Error: err}, err
	}

	return &models.BuildResult{
		Manifests: models.ManifestPair{
			BaseManifest: baseManifest,
			HeadManifest: headManifest,
		},
	}, nil
}

// GenerateReport generates the final report
func (r *RunnerLocal) GenerateReport(envResults map[string]models.EnvironmentResult) (*models.ReportResult, error) {
	// Build multi-environment report data
	envArr := make([]models.EnvironmentResult, 0, len(envResults))
	for _, env := range envResults {
		envArr = append(envArr, env)
	}
	multiEnvData := r.buildMultiEnvReport(envArr)

	// Render the report
	renderedMarkdown, err := r.Renderer.RenderWithTemplates(r.options.TemplatesPath, multiEnvData)
	if err != nil {
		return &models.ReportResult{Error: err}, err
	}

	return &models.ReportResult{
		MultiEnvData:     multiEnvData,
		RenderedMarkdown: renderedMarkdown,
	}, nil
}

// OutputResults outputs the results to local files
func (r *RunnerLocal) OutputResults(report *models.ReportResult) (*models.OutputResult, error) {
	// Always prepend the marker to the rendered content
	const COMMENT_MARKER = "<!-- gitops-kustomz: auto-generated comment, please do not remove -->"
	finalComment := COMMENT_MARKER + "\n\n" + report.RenderedMarkdown

	// Create output directory
	if err := os.MkdirAll(r.options.LcOutputDir, 0755); err != nil {
		return &models.OutputResult{Error: err}, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write to file
	outputFile := filepath.Join(r.options.LcOutputDir, fmt.Sprintf("%s-report.md", r.options.Service))
	if err := os.WriteFile(outputFile, []byte(finalComment), 0644); err != nil {
		return &models.OutputResult{Error: err}, fmt.Errorf("failed to write output file: %w", err)
	}

	return &models.OutputResult{
		Success: true,
		Message: fmt.Sprintf("Report written to: %s", outputFile),
	}, nil
}

// buildManifestsFromLocal builds manifests from local directories (Local-specific internal function)
func (r *RunnerLocal) buildManifestsFromLocal(environment string) ([]byte, []byte, error) {
	// Build paths: lcBeforeManifestsPath/service/environments/env
	beforeServicePath := filepath.Join(r.options.LcBeforeManifestsPath, r.options.Service, "environments", environment)
	afterServicePath := filepath.Join(r.options.LcAfterManifestsPath, r.options.Service, "environments", environment)

	// Build base manifest
	baseManifest, err := r.Builder.Build(r.ctx, beforeServicePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build base manifest: %w", err)
	}

	// Build head manifest
	headManifest, err := r.Builder.Build(r.ctx, afterServicePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build head manifest: %w", err)
	}

	return baseManifest, headManifest, nil
}

// buildMultiEnvReport builds the multi-environment report data (Local-specific internal function)
func (r *RunnerLocal) buildMultiEnvReport(environments []models.EnvironmentResult) config.MultiEnvCommentData {
	// Collect environments and diffs
	var envNames []string
	var envDiffs []config.EnvironmentDiff
	summary := make(map[string]config.EnvSummary)

	for _, env := range environments {
		envNames = append(envNames, env.Environment)
		envDiffs = append(envDiffs, env.DiffData)

		summary[env.Environment] = config.EnvSummary{
			TotalPolicies:     env.PolicyReport.TotalPolicies,
			PassedPolicies:    env.PolicyReport.PassedPolicies,
			FailedPolicies:    env.PolicyReport.FailedPolicies,
			ErroredPolicies:   env.PolicyReport.ErroredPolicies,
			BlockingFailures:  env.PolicyReport.BlockingFailures,
			WarningFailures:   env.PolicyReport.WarningFailures,
			RecommendFailures: env.PolicyReport.RecommendFailures,
		}
	}

	// Build policy matrix (policies x environments)
	policyMap := make(map[string]*config.MultiEnvPolicyDetail)

	for _, env := range environments {
		for _, detail := range env.PolicyReport.Details {
			if policyMap[detail.Name] == nil {
				policyMap[detail.Name] = &config.MultiEnvPolicyDetail{
					Name:        detail.Name,
					Description: detail.Description,
					Level:       detail.Level,
					Results:     make(map[string]config.EnvPolicyResult),
				}
			}

			policyMap[detail.Name].Results[env.Environment] = config.EnvPolicyResult{
				Status:     detail.Status,
				Violations: detail.Violations,
				Error:      detail.Error,
				Overridden: detail.Overridden,
			}
		}
	}

	// Convert map to slice
	var policies []config.MultiEnvPolicyDetail
	for _, policy := range policyMap {
		policies = append(policies, *policy)
	}

	return config.MultiEnvCommentData{
		Service:          r.options.Service,
		Environments:     envNames,
		BaseCommit:       "local",
		HeadCommit:       "local",
		EnvironmentDiffs: envDiffs,
		MultiEnvPolicyReport: config.MultiEnvPolicyReport{
			Environments: envNames,
			Policies:     policies,
			Summary:      summary,
		},
		Timestamp: time.Now(),
	}
}
