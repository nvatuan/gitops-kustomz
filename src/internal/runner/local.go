package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gh-nvat/gitops-kustomz/src/pkg/diff"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/kustomize"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/models"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/policy"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/template"
)

type RunnerLocal struct {
	RunnerBase
}

// make RunnerLocal implement RunnerInterface
var _ RunnerInterface = (*RunnerLocal)(nil)

func NewRunnerLocal(
	ctx context.Context,
	options *Options,
	builder *kustomize.Builder,
	differ *diff.Differ,
	evaluator *policy.PolicyEvaluator,
	renderer *template.Renderer,
) (*RunnerLocal, error) {
	baseRunner, err := NewRunnerBase(ctx, options, builder, differ, evaluator, renderer)
	if err != nil {
		return nil, err
	}
	runner := &RunnerLocal{
		RunnerBase: *baseRunner,
	}
	return runner, nil
}

func (r *RunnerLocal) Initialize() error {
	return r.RunnerBase.Initialize()
}

func (r *RunnerLocal) BuildManifests(beforePath, afterPath string) (*models.BuildManifestResult, error) {
	return r.RunnerBase.BuildManifests(beforePath, afterPath)
}

func (r *RunnerLocal) DiffManifests(result *models.BuildManifestResult) (map[string]models.EnvironmentDiff, error) {
	return r.RunnerBase.DiffManifests(result)
}

func (r *RunnerLocal) Process() error {
	logger.Info("Process: starting...")

	beforePath := filepath.Join(r.Options.LcBeforeManifestsPath, r.Options.Service)
	afterPath := filepath.Join(r.Options.LcAfterManifestsPath, r.Options.Service)
	rs, err := r.BuildManifests(beforePath, afterPath)
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
		BaseCommit:       "base",
		HeadCommit:       "head",
		Environments:     r.Options.Environments,
		ManifestChanges:  diffs,
		PolicyEvaluation: *policyEval,
	}

	if err := r.Output(&reportData); err != nil {
		return err
	}
	return nil
}

func (r *RunnerLocal) Output(data *models.ReportData) error {
	logger.Info("Output: starting...")
	if err := r.outputReportJson(data); err != nil {
		return err
	}
	if err := r.outputReportMarkdown(data); err != nil {
		return err
	}
	logger.Info("Output: done.")
	return nil
}

// Exporting report json file to output directory if enabled
func (r *RunnerLocal) outputReportJson(data *models.ReportData) error {
	if !r.Options.EnableExportReport {
		logger.Info("OutputJson: option was disabled")
		return nil
	}
	logger.Info("OutputJson: starting...")

	if err := os.MkdirAll(r.Options.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

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
func (r *RunnerLocal) outputReportMarkdown(data *models.ReportData) error {
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
