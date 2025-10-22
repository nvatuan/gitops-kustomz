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

	log "github.com/sirupsen/logrus"
)

var logger *log.Entry = log.New().WithFields(log.Fields{
	"package": "runner",
})

type RunnerBase struct {
	Context context.Context
	Options *Options

	RunMode string

	Builder   *kustomize.Builder
	Differ    *diff.Differ
	Evaluator *policy.PolicyEvaluator
	Renderer  *template.Renderer

	Instance RunnerInterface
}

// make RunnerLocal implement RunnerInterface
var _ RunnerInterface = (*RunnerBase)(nil)

func NewRunnerBase(
	ctx context.Context,
	options *Options,
	builder *kustomize.Builder,
	differ *diff.Differ,
	evaluator *policy.PolicyEvaluator,
	renderer *template.Renderer,
) (*RunnerBase, error) {
	runner := &RunnerBase{
		Context:   ctx,
		Options:   options,
		RunMode:   options.RunMode,
		Builder:   builder,
		Differ:    differ,
		Evaluator: evaluator,
		Renderer:  renderer,
	}
	return runner, nil
}

func (r *RunnerBase) Initialize() error {
	logger.Info("Initializing runner: starting...")

	// if any is nil, return error
	if r.Builder == nil || r.Differ == nil || r.Evaluator == nil || r.Renderer == nil {
		return fmt.Errorf("builder, differ, evaluator, reporter, and renderer are required")
	}

	logger.Info("Initalize runner: Evaluator: Loading and validating policy configuration")
	// load and validate policy configuration
	err := r.Evaluator.LoadAndValidate()
	if err != nil {
		return fmt.Errorf("failed to load policy config: %w", err)
	}

	logger.Info("Initalize runner: done.")
	return nil
}

func (r *RunnerBase) BuildManifests() (*models.BuildManifestResult, error) {
	logger.Info("BuildManifests: starting...")

	results := make(map[string]models.BuildEnvManifestResult)
	envs := r.Options.Environments
	for _, env := range envs {
		beforeManifest, err := r.Builder.Build(r.Context, r.Options.LcBeforeManifestsPath, r.Options.Service, env)
		if err != nil {
			return nil, err
		}
		afterManifest, err := r.Builder.Build(r.Context, r.Options.LcAfterManifestsPath, r.Options.Service, env)
		if err != nil {
			return nil, err
		}
		results[env] = models.BuildEnvManifestResult{
			Environment:    env,
			BeforeManifest: beforeManifest,
			AfterManifest:  afterManifest,
		}
		logger.WithField("env", env).WithField("beforeManifest", string(beforeManifest)).Debug("Built Manifest")
		logger.WithField("env", env).WithField("afterManifest", string(afterManifest)).Debug("Built Manifest")
	}

	logger.Info("BuildManifests: done.")
	return &models.BuildManifestResult{
		EnvManifestBuild: results,
	}, nil
}

func (r *RunnerBase) DiffManifests(result *models.BuildManifestResult) (map[string]models.EnvironmentDiff, error) {
	logger.Info("DiffManifests: starting...")

	results := make(map[string]models.EnvironmentDiff)

	for env, envResult := range result.EnvManifestBuild {
		diffContent, err := r.Differ.Diff(envResult.BeforeManifest, envResult.AfterManifest)
		if err != nil {
			logger.WithField("env", envResult.Environment).WithField("error", err).Error("Failed to diff manifests")
			return nil, err
		}
		logger.WithField("env", envResult.Environment).WithField("diffContent", diffContent).Debug("Diffed Manifest")

		addedLines, deletedLines, totalLines := diff.CalcLineChangesFromDiffContent(diffContent)
		results[env] = models.EnvironmentDiff{
			LineCount:        totalLines,
			AddedLineCount:   addedLines,
			DeletedLineCount: deletedLines,
			Content:          diffContent,
		}
	}

	logger.Info("DiffManifests: done.")
	return results, nil
}

func (r *RunnerBase) EvaluatePolicies(mf *models.BuildManifestResult) (*models.PolicyEvaluateResult, error) {
	logger.Info("EvaluatePolicies: starting...")

	results := models.PolicyEvaluateResult{}

	for _, envResult := range mf.EnvManifestBuild {
		// only evaluate the after manifest
		envManifest := envResult.AfterManifest
		failMsgs, err := r.Evaluator.Evaluate(r.Context, envManifest)
		if err != nil {
			return nil, err
		}
		results.EnvPolicyEvaluate[envResult.Environment] = models.PolicyEnvEvaluateResult{
			Environment:            envResult.Environment,
			PolicyIdToEvalFailMsgs: failMsgs,
		}
	}

	logger.Info("EvaluatePolicies: done.")
	return &results, nil
}

func (r *RunnerBase) Process() error {
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

func (r *RunnerBase) Output(data *models.ReportData) error {
	logger.Info("Output: starting...")
	if err := r.outputReportJson(data); err != nil {
		return err
	}
	logger.Info("Output: done.")
	return nil
}

// Exporting report json file to output directory if enabled
func (r *RunnerBase) outputReportJson(data *models.ReportData) error {
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
