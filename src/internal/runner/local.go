package runner

import (
	"context"

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

func (r *RunnerLocal) BuildManifests() (*models.BuildManifestResult, error) {
	return r.RunnerBase.BuildManifests()
}

func (r *RunnerLocal) DiffManifests(result *models.BuildManifestResult) (map[string]models.EnvironmentDiff, error) {
	return r.RunnerBase.DiffManifests(result)
}

func (r *RunnerLocal) Process() error {
	return r.RunnerBase.Process()
}

func (r *RunnerLocal) Output(data *models.ReportData) error {
	return r.RunnerBase.Output(data)
}
