package runner

import (
	"context"
	"fmt"

	"github.com/gh-nvat/gitops-kustomz/src/pkg/github"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/models"
)

type RunnerGitHub struct {
	RunnerBase

	options  *Options
	ghclient *github.Client

	prInfo   *models.PullRequest
	comments []*models.Comment
}

func NewRunnerGitHub(
	ctx context.Context,
	ghclient *github.Client,
	baseRunner *RunnerBase,
) (*RunnerGitHub, error) {
	if ghclient == nil {
		return nil, fmt.Errorf("GitHub client is not initialized")
	}
	if baseRunner == nil {
		return nil, fmt.Errorf("base runner is not initialized")
	}
	return &RunnerGitHub{
		RunnerBase: *baseRunner,
		ghclient:   ghclient,
		options:    baseRunner.Options,
	}, nil
}

func (r *RunnerGitHub) Initialize() error {
	if err := r.fetchAndSetPullRequestInfo(); err != nil {
		return fmt.Errorf("failed to fetch pull request info: %w", err)
	}
	return nil
}

// Fetch and set pull request data into struct from GitHub
func (r *RunnerGitHub) fetchAndSetPullRequestInfo() error {
	owner, repo, err := github.ParseOwnerRepo(r.options.GhRepo)
	if err != nil {
		return fmt.Errorf("failed to parse repository: %w", err)
	}

	// Create channels for parallel execution
	type prResult struct {
		pr  *models.PullRequest
		err error
	}
	type commentsResult struct {
		comments []*models.Comment
		err      error
	}

	prChan := make(chan prResult, 1)
	commentsChan := make(chan commentsResult, 1)

	// Fetch PR info in parallel
	go func() {
		pr, err := r.ghclient.GetPR(r.Context, owner, repo, r.options.GhPrNumber)
		prChan <- prResult{pr: pr, err: err}
	}()

	// Fetch comments in parallel
	go func() {
		comments, err := r.ghclient.GetComments(r.Context, owner, repo, r.options.GhPrNumber)
		commentsChan <- commentsResult{comments: comments, err: err}
	}()

	// Wait for both results
	select {
	case prRes := <-prChan:
		if prRes.err != nil {
			return fmt.Errorf("failed to get PR info: %w", prRes.err)
		}
		r.prInfo = prRes.pr
	case <-r.Context.Done():
		return fmt.Errorf("PR fetch cancelled: %w", r.Context.Err())
	}

	select {
	case commentsRes := <-commentsChan:
		if commentsRes.err != nil {
			return fmt.Errorf("failed to get PR comments: %w", commentsRes.err)
		}
		r.comments = commentsRes.comments
	case <-r.Context.Done():
		return fmt.Errorf("comments fetch cancelled: %w", r.Context.Err())
	}

	return nil
}

// ProcessEnvironment processes a single environment and returns the result
// func (r *RunnerGitHub) ProcessEnvironment(environment string) (*models.ReportData, error) {
// 	// Build manifests for this environment
// 	buildResult, err := r.BuildManifests(environment)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to build manifests: %w", err)
// 	}

// 	// Generate diff
// 	diffContent, err := r.Differ.Diff(buildResult.Manifests.BeforeManifest, buildResult.Manifests.AfterManifest)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to generate diff: %w", err)
// 	}

// 	// Count only actual changed lines (lines starting with + or -)
// 	addedLines, deletedLines := 0, 0
// 	for _, line := range strings.Split(diffContent, "\n") {
// 		if strings.HasPrefix(line, "+ ") {
// 			addedLines++
// 		}
// 		if strings.HasPrefix(line, "- ") {
// 			deletedLines++
// 		}
// 	}

// 	// Create environment diff
// 	envDiff := models.EnvironmentDiff{
// 		LineCount:        addedLines + deletedLines,
// 		AddedLineCount:   addedLines,
// 		DeletedLineCount: deletedLines,
// 		Content:          diffContent,
// 	}

// 	// TODO: Implement actual policy evaluation and parse directly into PolicyEvaluation
// 	policyEvaluation := models.PolicyEvaluation{
// 		EnvironmentSummary: map[string]models.PolicyCounts{
// 			environment: {
// 				Success: 0,
// 				Failed:  0,
// 				Errored: 0,
// 			},
// 		},
// 		PolicyMatrix: map[string]models.PolicyMatrix{
// 			environment: {
// 				BlockingPolicies:    []models.PolicyResult{},
// 				WarningPolicies:     []models.PolicyResult{},
// 				RecommendPolicies:   []models.PolicyResult{},
// 				OverriddenPolicies:  []models.PolicyResult{},
// 				NotInEffectPolicies: []models.PolicyResult{},
// 			},
// 		},
// 	}

// 	// Create report data for this environment
// 	reportData := &models.ReportData{
// 		Service:      r.options.Service,
// 		Timestamp:    time.Now(),
// 		BaseCommit:   r.prInfo.BaseSHA,
// 		HeadCommit:   r.prInfo.HeadSHA,
// 		Environments: []string{environment},
// 		ManifestChanges: map[string]models.EnvironmentDiff{
// 			environment: envDiff,
// 		},
// 		PolicyEvaluation: policyEvaluation,
// 	}

// 	return reportData, nil
// }

// // BuildManifests builds manifests for a specific environment
// func (r *RunnerGitHub) BuildManifests() (*BuildManifestResult, error) {
// 	results := BuildManifestResult{
// 		EnvManifestBuild: make(map[string]BuildEnvManifestResult),
// 	}
// 	for _, environment := range r.options.Environments {
// 		baseManifest, headManifest, err := r.buildManifestsFromPR(environment)
// 		if err != nil {
// 			return nil, err
// 		}
// 		results.EnvManifestBuild[environment] = BuildEnvManifestResult{
// 			Environment:    environment,
// 			BeforeManifest: baseManifest,
// 			AfterManifest:  headManifest,
// 		}
// 	}

// 	return &results, nil
// }

// // GenerateReport generates the final report
// // func (r *RunnerGitHub) GenerateReport(result *BuildManifestResult) (*models.ReportResult, error) {
// // 	// TODO: Implement report generation with proper template rendering
// // 	// For now, just combine the environment results

// // }

// // // OutputResults outputs the results to GitHub
// // func (r *RunnerGitHub) OutputResults(report *models.ReportResult) (*models.OutputResult, error) {
// // 	// Always prepend the marker to the rendered content
// // 	const COMMENT_MARKER = "<!-- gitops-kustomz: auto-generated comment, please do not remove -->"
// // 	finalComment := COMMENT_MARKER + "\n\n" + report.RenderedMarkdown

// // 	owner, repo, err := github.ParseOwnerRepo(r.options.GhRepo)
// // 	if err != nil {
// // 		return &models.OutputResult{Error: err}, fmt.Errorf("failed to parse repository: %w", err)
// // 	}

// // 	// Find existing comment
// // 	existingComment, err := r.ghclient.FindToolComment(r.ctx, owner, repo, r.options.GhPrNumber)
// // 	if err != nil {
// // 		return &models.OutputResult{Error: err}, fmt.Errorf("failed to search for existing comment: %w", err)
// // 	}

// // 	if existingComment != nil {
// // 		// Update existing comment
// // 		if err := r.ghclient.UpdateComment(r.ctx, owner, repo, existingComment.ID, finalComment); err != nil {
// // 			return &models.OutputResult{Error: err}, fmt.Errorf("failed to update comment: %w", err)
// // 		}
// // 		return &models.OutputResult{
// // 			Success: true,
// // 			Message: fmt.Sprintf("Updated existing comment (ID: %d)", existingComment.ID),
// // 		}, nil
// // 	} else {
// // 		// Create new comment
// // 		newComment, err := r.ghclient.CreateComment(r.ctx, owner, repo, r.options.GhPrNumber, finalComment)
// // 		if err != nil {
// // 			return &models.OutputResult{Error: err}, fmt.Errorf("failed to create comment: %w", err)
// // 		}
// // 		return &models.OutputResult{
// // 			Success: true,
// // 			Message: fmt.Sprintf("Created new comment (ID: %d)", newComment.ID),
// // 		}, nil
// // 	}
// // }

// // // buildManifestsFromPR builds manifests from PR (GitHub-specific internal function)
// // func (r *RunnerGitHub) buildManifestsFromPR(environment string) ([]byte, []byte, error) {
// // 	// Use manifestsPath from options (default: ./services)
// // 	servicePath := r.Builder.GetServiceEnvironmentPath(r.options.ManifestsPath, r.options.Service, environment)

// // 	// Build base manifest
// // 	if err := r.checkoutRef(r.prInfo.BaseSHA); err != nil {
// // 		return nil, nil, fmt.Errorf("failed to checkout base ref: %w", err)
// // 	}

// // 	baseManifest, err := r.Builder.Build(r.ctx, servicePath)
// // 	if err != nil {
// // 		return nil, nil, fmt.Errorf("failed to build base manifest: %w", err)
// // 	}

// // 	// Build head manifest
// // 	if err := r.checkoutRef(r.prInfo.HeadSHA); err != nil {
// // 		return nil, nil, fmt.Errorf("failed to checkout head ref: %w", err)
// // 	}

// // 	headManifest, err := r.Builder.Build(r.ctx, servicePath)
// // 	if err != nil {
// // 		return nil, nil, fmt.Errorf("failed to build head manifest: %w", err)
// // 	}

// // 	return baseManifest, headManifest, nil
// // }

// // // checkoutRef checks out a specific git ref (GitHub-specific internal function)
// // func (r *RunnerGitHub) checkoutRef(ref string) error {
// // 	cmd := exec.Command("git", "checkout", ref)
// // 	if output, err := cmd.CombinedOutput(); err != nil {
// // 		return fmt.Errorf("git checkout failed: %w\nOutput: %s", err, string(output))
// // 	}
// // 	return nil
// // }
