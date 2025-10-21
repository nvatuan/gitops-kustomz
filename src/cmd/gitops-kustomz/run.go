package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gh-nvat/gitops-kustomz/src/internal/runner"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/diff"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/github"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/kustomize"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/models"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/policy"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/template"
)

const (
	RUN_MODE_GITHUB = "github"
	RUN_MODE_LOCAL  = "local"
)

// Initialize creates and initializes the appropriate runner
func initialize(ctx context.Context, opts *runner.Options) (*runner.Runner, error) {
	// Create common dependencies
	baseRunner := &runner.BaseRunner{
		Builder:   kustomize.NewBuilder(),
		Differ:    diff.NewDiffer(),
		Evaluator: policy.NewEvaluator(),
		Reporter:  policy.NewReporter(),
		Renderer:  template.NewRenderer(),
	}

	// Load and validate policy configuration
	complianceConfig := filepath.Join(opts.PoliciesPath, policy.COMPLIANCE_CONFIG_FILENAME)
	fmt.Println("📋 Loading policy configuration...")
	policyConfig, err := baseRunner.Evaluator.LoadAndValidate(complianceConfig, opts.PoliciesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load policy config: %w", err)
	}
	baseRunner.PolicyConfig = policyConfig
	fmt.Printf("✅ Loaded %d policies\n\n", len(policyConfig.Policies))

	var runnerInstance runner.RunnerInterface

	switch opts.RunMode {
	case RUN_MODE_GITHUB:
		ghClient, err := github.NewClient()
		if err != nil {
			return nil, fmt.Errorf("GitHub authentication failed: %w", err)
		}
		runnerInstance, err = runner.NewRunnerGitHub(ctx, opts, ghClient, baseRunner)
		if err != nil {
			return nil, fmt.Errorf("failed to create GitHub runner: %w", err)
		}
	case RUN_MODE_LOCAL:
		runnerInstance, err = runner.NewRunnerLocal(ctx, opts, baseRunner)
		if err != nil {
			return nil, fmt.Errorf("failed to create Local runner: %w", err)
		}
	default:
		return nil, fmt.Errorf("invalid run mode: %s", opts.RunMode)
	}

	// Initialize runner
	if err := runnerInstance.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize runner: %w", err)
	}

	return &runner.Runner{
		RunMode:  opts.RunMode,
		Instance: runnerInstance,
	}, nil
}

func run(ctx context.Context, opts *runner.Options) error {
	// Validate options
	if err := validateOptions(opts); err != nil {
		return fmt.Errorf("invalid options: %w", err)
	}

	// Initialize runner
	runner, err := initialize(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	envResults := make(map[string]models.EnvironmentResult)
	hasErrors := false

	// Process each environment
	for _, environment := range opts.Environments {
		fmt.Printf("═══════════════════════════════════════════════════\n")
		fmt.Printf("🔍 Checking: %s (%s)\n", opts.Service, environment)
		fmt.Printf("═══════════════════════════════════════════════════\n\n")

		result, err := runner.Instance.ProcessEnvironment(environment)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Environment %s failed: %v\n\n", environment, err)
			hasErrors = true
			continue
		}

		envResults[environment] = *result
		fmt.Printf("✅ Environment %s completed\n\n", environment)
	}

	if hasErrors && len(envResults) == 0 {
		return fmt.Errorf("all environments failed")
	}

	// Generate combined report
	fmt.Println("📝 Generating combined report...")
	reportResult, err := runner.Instance.GenerateReport(envResults)
	if err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	// Output results
	fmt.Println("💾 Outputting results...")
	outputResult, err := runner.Instance.OutputResults(reportResult)
	if err != nil {
		return fmt.Errorf("failed to output results: %w", err)
	}

	fmt.Printf("✅ %s\n", outputResult.Message)

	// Check for failures across all environments
	for _, envResult := range envResults {
		if envResult.Enforcement.ShouldBlock {
			return fmt.Errorf("blocking policy failures detected in %s", envResult.Environment)
		}
	}

	for _, envResult := range envResults {
		if envResult.Enforcement.ShouldWarn {
			return fmt.Errorf("warning policy failures detected in %s", envResult.Environment)
		}
	}

	fmt.Println("✅ All environments passed!")
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
