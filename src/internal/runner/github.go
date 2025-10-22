package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gh-nvat/gitops-kustomz/src/pkg/diff"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/github"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/kustomize"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/models"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/policy"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/template"
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
	options *Options,
	ghclient *github.Client,
	builder *kustomize.Builder,
	differ *diff.Differ,
	evaluator *policy.PolicyEvaluator,
	renderer *template.Renderer,
) (*RunnerGitHub, error) {
	if ghclient == nil {
		return nil, fmt.Errorf("GitHub client is not initialized")
	}
	baseRunner, err := NewRunnerBase(ctx, options, builder, differ, evaluator, renderer)
	if err != nil {
		return nil, err
	}
	runner := &RunnerGitHub{
		RunnerBase: *baseRunner,
		ghclient:   ghclient,
		options:    options,
	}
	return runner, nil
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

func (r *RunnerGitHub) BuildManifests() (*models.BuildManifestResult, error) {
	return r.RunnerBase.BuildManifests()
}

func (r *RunnerGitHub) DiffManifests(result *models.BuildManifestResult) (map[string]models.EnvironmentDiff, error) {
	return r.RunnerBase.DiffManifests(result)
}

func (r *RunnerGitHub) Process() error {
	logger.Info("Process: starting...")

	rs, err := r.BuildManifests()
	if err != nil {
		return err
	}
	logger.WithField("results", rs).Debug("Built Manifests")

	diffs, err := r.DiffManifests(rs)
	if err != nil {
		return err
	}
	logger.WithField("results", diffs).Debug("Diffed Manifests")

	policyEval, err := r.Evaluator.GeneratePolicyEvalResultForManifests(r.Context, *rs, []string{})
	if err != nil {
		return err
	}
	logger.WithField("results", policyEval).Debug("Evaluated Policies")

	reportData := models.ReportData{
		Service:          r.Options.Service,
		Timestamp:        time.Now(),
		BaseCommit:       r.prInfo.BaseSHA,
		HeadCommit:       r.prInfo.HeadSHA,
		Environments:     r.Options.Environments,
		ManifestChanges:  diffs,
		PolicyEvaluation: *policyEval,
	}

	if err := r.Output(&reportData); err != nil {
		return err
	}
	return nil
}

func (r *RunnerGitHub) Output(data *models.ReportData) error {
	logger.Info("Output: starting...")
	if err := r.outputReportJson(data); err != nil {
		return err
	}
	if err := r.outputReportMarkdown(data); err != nil {
		return err
	}
	if err := r.outputGitHubComment(data); err != nil {
		return err
	}
	logger.Info("Output: done.")
	return nil
}

// Exporting report json file to output directory if enabled
func (r *RunnerGitHub) outputReportJson(data *models.ReportData) error {
	if !r.Options.EnableExportReport {
		logger.Info("OutputJson: option was disabled")
		return nil
	}
	logger.Info("OutputJson: starting...")

	resultsJson, err := json.Marshal(data)
	if err != nil {
		return err
	}
	filePath := filepath.Join(r.Options.OutputDir, "report.json")
	if err := os.WriteFile(filePath, resultsJson, 0644); err != nil {
		logger.WithField("filePath", filePath).WithField("error", err).Error("Failed to write report data to file")
		return err
	}
	logger.WithField("filePath", filePath).Info("Written report data to file")
	return nil
}

// Exporting report markdown file to output directory
func (r *RunnerGitHub) outputReportMarkdown(data *models.ReportData) error {
	logger.Info("OutputMarkdown: starting...")

	// Render the markdown using templates
	renderedMarkdown, err := r.Renderer.RenderWithTemplates(r.Options.TemplatesPath, data)
	if err != nil {
		logger.WithField("error", err).Error("Failed to render markdown template")
		return err
	}

	// Write the rendered markdown to file
	filePath := filepath.Join(r.Options.OutputDir, "report.md")
	if err := os.WriteFile(filePath, []byte(renderedMarkdown), 0644); err != nil {
		logger.WithField("filePath", filePath).WithField("error", err).Error("Failed to write markdown report to file")
		return err
	}

	logger.WithField("filePath", filePath).Info("Written markdown report to file")
	return nil
}

// Post comment to GitHub PR
func (r *RunnerGitHub) outputGitHubComment(data *models.ReportData) error {
	logger.Info("OutputGitHubComment: starting...")

	// Render the markdown using templates
	renderedMarkdown, err := r.Renderer.RenderWithTemplates(r.Options.TemplatesPath, data)
	if err != nil {
		logger.WithField("error", err).Error("Failed to render markdown template")
		return err
	}

	// Add the comment marker
	finalComment := template.ToolCommentSignature + "\n\n" + renderedMarkdown

	owner, repo, err := github.ParseOwnerRepo(r.Options.GhRepo)
	if err != nil {
		return fmt.Errorf("failed to parse repository: %w", err)
	}

	// Check if there's an existing comment from this tool
	existingComment, err := r.ghclient.FindToolComment(r.Context, owner, repo, r.Options.GhPrNumber)
	if err != nil {
		logger.WithField("error", err).Warn("Failed to find existing comment, will create new one")
	}

	if existingComment != nil {
		// Update existing comment
		if err := r.ghclient.UpdateComment(r.Context, owner, repo, existingComment.ID, finalComment); err != nil {
			logger.WithField("error", err).Error("Failed to update existing comment")
			return err
		}
		logger.Info("Updated existing GitHub comment")
	} else {
		// Create new comment
		if _, err := r.ghclient.CreateComment(r.Context, owner, repo, r.Options.GhPrNumber, finalComment); err != nil {
			logger.WithField("error", err).Error("Failed to create new comment")
			return err
		}
		logger.Info("Created new GitHub comment")
	}

	return nil
}
