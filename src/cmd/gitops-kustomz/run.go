package main

import (
	"context"
	"fmt"

	"github.com/gh-nvat/gitops-kustomz/src/internal/runner"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/diff"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/kustomize"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/policy"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/template"
	log "github.com/sirupsen/logrus"
)

var logger *log.Entry = log.New().WithFields(log.Fields{
	"package": "run",
})

const (
	RUN_MODE_GITHUB = "github"
	RUN_MODE_LOCAL  = "local"
)

// Initialize creates and initializes the appropriate runner
func createRunner(ctx context.Context, opts *runner.Options) (runner.RunnerInterface, error) {
	logger.WithField("opts", opts).Debug("Creating runner..")

	builder := kustomize.NewBuilder()
	differ := diff.NewDiffer()
	evaluator := policy.NewPolicyEvaluator(opts.PoliciesPath)
	renderer := template.NewRenderer()

	switch opts.RunMode {
	case RUN_MODE_GITHUB:
		// ghClient, err := github.NewClient()
		// if err != nil {
		// 	return nil, fmt.Errorf("GitHub authentication failed: %w", err)
		// }
		// runnerInstance, err = runner.NewRunnerGitHub(ctx, opts, ghClient, baseRunner)
		// if err != nil {
		// 	return nil, fmt.Errorf("failed to create GitHub runner: %w", err)
		// }
	case RUN_MODE_LOCAL:
		runner, err := runner.NewRunnerLocal(
			ctx, opts, builder, differ, evaluator, renderer,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create Local runner: %w", err)
		}
		return runner, nil
	default:
		return nil, fmt.Errorf("invalid run mode: %s", opts.RunMode)
	}
	return nil, fmt.Errorf("invalid run mode: %s", opts.RunMode)
}

func initialize(ctx context.Context, opts *runner.Options) (runner.RunnerInterface, error) {
	runner, err := createRunner(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create runner: %w", err)
	}
	if err := runner.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize runner: %w", err)
	}
	return runner, nil
}

func run(ctx context.Context, opts *runner.Options) error {
	logger.WithField("opts", opts).Info("Running..")
	log.SetLevel(log.DebugLevel)

	// Validate options
	if err := validateOptions(opts); err != nil {
		return fmt.Errorf("invalid options: %w", err)
	}

	// Initialize runner
	runner, err := initialize(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	err = runner.Process()
	if err != nil {
		return fmt.Errorf("failed to process: %w", err)
	}

	return nil
}

func validateOptions(opts *runner.Options) error {
	// Validate common options
	if opts.Service == "" {
		return fmt.Errorf("service is required")
	}

	if len(opts.Environments) == 0 {
		return fmt.Errorf("at least one environment is required")
	}

	// Validate run mode
	if opts.RunMode != "github" && opts.RunMode != "local" {
		return fmt.Errorf("run-mode must be 'github' or 'local', got: %s", opts.RunMode)
	}

	// Validate mode-specific options
	if opts.RunMode == "local" {
		if opts.LcBeforeManifestsPath == "" || opts.LcAfterManifestsPath == "" {
			return fmt.Errorf("local mode requires --lc-before-manifests-path and --lc-after-manifests-path")
		}
	} else {
		// GitHub mode
		if opts.GhRepo == "" {
			return fmt.Errorf("github mode requires --gh-repo")
		}
		if opts.GhPrNumber == 0 {
			return fmt.Errorf("github mode requires --gh-pr-number")
		}
	}

	return nil
}
