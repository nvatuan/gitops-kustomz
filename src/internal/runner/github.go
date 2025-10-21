package runner

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/gh-nvat/gitops-kustomz/src/pkg/github"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/models"
)

type RunnerGitHub struct {
	BaseRunner
	ctx     context.Context
	options *Options

	ghclient *github.Client

	prInfo   *config.PullRequest
	comments []*config.Comment
}

func NewRunnerGitHub(
	ctx context.Context,
	options *Options,
	ghclient *github.Client,
	baseRunner *BaseRunner,
) (*RunnerGitHub, error) {
	if ghclient == nil {
		return nil, fmt.Errorf("GitHub client is not initialized")
	}
	return &RunnerGitHub{
		BaseRunner: *baseRunner,
		ctx:        ctx,
		options:    options,
		ghclient:   ghclient,
	}, nil
}

func (c *RunnerGitHub) Initialize() error {
	if err := c.fetchAndSetPullRequestInfo(); err != nil {
		return fmt.Errorf("failed to fetch pull request info: %w", err)
	}
	return nil
}

// Fetch and set pull request data into struct from GitHub
func (c *RunnerGitHub) fetchAndSetPullRequestInfo() error {
	owner, repo, err := github.ParseOwnerRepo(c.options.GhRepo)
	if err != nil {
		return fmt.Errorf("failed to parse repository: %w", err)
	}

	// Create channels for parallel execution
	type prResult struct {
		pr  *config.PullRequest
		err error
	}
	type commentsResult struct {
		comments []*config.Comment
		err      error
	}

	prChan := make(chan prResult, 1)
	commentsChan := make(chan commentsResult, 1)

	// Fetch PR info in parallel
	go func() {
		pr, err := c.ghclient.GetPR(c.ctx, owner, repo, c.options.GhPrNumber)
		prChan <- prResult{pr: pr, err: err}
	}()

	// Fetch comments in parallel
	go func() {
		comments, err := c.ghclient.GetComments(c.ctx, owner, repo, c.options.GhPrNumber)
		commentsChan <- commentsResult{comments: comments, err: err}
	}()

	// Wait for both results
	select {
	case prRes := <-prChan:
		if prRes.err != nil {
			return fmt.Errorf("failed to get PR info: %w", prRes.err)
		}
		c.prInfo = prRes.pr
	case <-c.ctx.Done():
		return fmt.Errorf("PR fetch cancelled: %w", c.ctx.Err())
	}

	select {
	case commentsRes := <-commentsChan:
		if commentsRes.err != nil {
			return fmt.Errorf("failed to get PR comments: %w", commentsRes.err)
		}
		c.comments = commentsRes.comments
	case <-c.ctx.Done():
		return fmt.Errorf("comments fetch cancelled: %w", c.ctx.Err())
	}

	return nil
}

// ProcessEnvironment processes a single environment and returns the result
func (r *RunnerGitHub) ProcessEnvironment(environment string) (*models.EnvironmentResult, error) {
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
	evalResult, err := r.Evaluator.Evaluate(
		r.ctx,
		buildResult.Manifests.HeadManifest,
		r.PolicyConfig,
		r.options.PoliciesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate policies: %w", err)
	}

	// Check for overrides
	overrides := r.Evaluator.CheckOverrides(r.comments, r.PolicyConfig)
	r.Evaluator.ApplyOverrides(evalResult, overrides)

	// Determine enforcement
	enforcement := r.Evaluator.Enforce(evalResult, overrides)

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
func (r *RunnerGitHub) BuildManifests(environment string) (*models.BuildResult, error) {
	baseManifest, headManifest, err := r.buildManifestsFromPR(environment)
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
func (r *RunnerGitHub) GenerateReport(envResults map[string]models.EnvironmentResult) (*models.ReportResult, error) {
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

// OutputResults outputs the results to GitHub
func (r *RunnerGitHub) OutputResults(report *models.ReportResult) (*models.OutputResult, error) {
	// Always prepend the marker to the rendered content
	const COMMENT_MARKER = "<!-- gitops-kustomz: auto-generated comment, please do not remove -->"
	finalComment := COMMENT_MARKER + "\n\n" + report.RenderedMarkdown

	owner, repo, err := github.ParseOwnerRepo(r.options.GhRepo)
	if err != nil {
		return &models.OutputResult{Error: err}, fmt.Errorf("failed to parse repository: %w", err)
	}

	// Find existing comment
	existingComment, err := r.ghclient.FindToolComment(r.ctx, owner, repo, r.options.GhPrNumber)
	if err != nil {
		return &models.OutputResult{Error: err}, fmt.Errorf("failed to search for existing comment: %w", err)
	}

	if existingComment != nil {
		// Update existing comment
		if err := r.ghclient.UpdateComment(r.ctx, owner, repo, existingComment.ID, finalComment); err != nil {
			return &models.OutputResult{Error: err}, fmt.Errorf("failed to update comment: %w", err)
		}
		return &models.OutputResult{
			Success: true,
			Message: fmt.Sprintf("Updated existing comment (ID: %d)", existingComment.ID),
		}, nil
	} else {
		// Create new comment
		newComment, err := r.ghclient.CreateComment(r.ctx, owner, repo, r.options.GhPrNumber, finalComment)
		if err != nil {
			return &models.OutputResult{Error: err}, fmt.Errorf("failed to create comment: %w", err)
		}
		return &models.OutputResult{
			Success: true,
			Message: fmt.Sprintf("Created new comment (ID: %d)", newComment.ID),
		}, nil
	}
}

// buildManifestsFromPR builds manifests from PR (GitHub-specific internal function)
func (r *RunnerGitHub) buildManifestsFromPR(environment string) ([]byte, []byte, error) {
	// Use manifestsPath from options (default: ./services)
	servicePath := r.Builder.GetServiceEnvironmentPath(r.options.ManifestsPath, r.options.Service, environment)

	// Build base manifest
	if err := r.checkoutRef(r.prInfo.BaseSHA); err != nil {
		return nil, nil, fmt.Errorf("failed to checkout base ref: %w", err)
	}

	baseManifest, err := r.Builder.Build(r.ctx, servicePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build base manifest: %w", err)
	}

	// Build head manifest
	if err := r.checkoutRef(r.prInfo.HeadSHA); err != nil {
		return nil, nil, fmt.Errorf("failed to checkout head ref: %w", err)
	}

	headManifest, err := r.Builder.Build(r.ctx, servicePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build head manifest: %w", err)
	}

	return baseManifest, headManifest, nil
}

// checkoutRef checks out a specific git ref (GitHub-specific internal function)
func (r *RunnerGitHub) checkoutRef(ref string) error {
	cmd := exec.Command("git", "checkout", ref)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout failed: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// buildMultiEnvReport builds the multi-environment report data (GitHub-specific internal function)
func (r *RunnerGitHub) buildMultiEnvReport(environments []models.EnvironmentResult) config.MultiEnvCommentData {
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

	var baseCommit, headCommit string
	if r.prInfo != nil {
		baseCommit = github.ShortSHA(r.prInfo.BaseSHA)
		headCommit = github.ShortSHA(r.prInfo.HeadSHA)
	} else {
		baseCommit = "local"
		headCommit = "local"
	}

	return config.MultiEnvCommentData{
		Service:          r.options.Service,
		Environments:     envNames,
		BaseCommit:       baseCommit,
		HeadCommit:       headCommit,
		EnvironmentDiffs: envDiffs,
		MultiEnvPolicyReport: config.MultiEnvPolicyReport{
			Environments: envNames,
			Policies:     policies,
			Summary:      summary,
		},
		Timestamp: time.Now(),
	}
}
