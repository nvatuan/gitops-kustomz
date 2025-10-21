package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gh-nvat/gitops-kustomz/src/internal/runner"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/config"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/diff"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/github"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/kustomize"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/policy"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/template"
)

const (
	RUN_MODE_GITHUB = "github"
	RUN_MODE_LOCAL  = "local"
)

type envData struct {
	environment  string
	diffData     config.EnvironmentDiff
	evalResult   *config.EvaluationResult
	policyReport *config.PolicyReportData
	enforcement  *config.EnforcementResult
}

var (
	builder      *kustomize.Builder
	differ       *diff.Differ
	evaluator    *policy.Evaluator
	reporter     *policy.Reporter
	renderer     *template.Renderer
	policyConfig *config.ComplianceConfig

	ghClient *github.Client
)

// Do all initialization steps here:
// 1. Initialize the builder, differ, evaluator, reporter, renderer
// 2. Initialize the runner instance
// // a. Initialize the GitHub runner if running in GitHub mode
// //    - Fetch and set pull request info
// //    - Fetch and set pull request comments
// // b. Initialize the Local runner if running in Local mode
// 3. Load and validate the policy configuration
// 4. Return the runner instance
func initialize(ctx context.Context, opts *runner.Options) (
	*runner.Runner, error,
) {
	builder = kustomize.NewBuilder()
	differ = diff.NewDiffer()
	evaluator = policy.NewEvaluator()
	reporter = policy.NewReporter()
	renderer = template.NewRenderer()

	var runnerInstance runner.RunnerInterface
	var err error
	switch opts.RunMode {
	case RUN_MODE_GITHUB:
		ghClient, err = github.NewClient()
		if err != nil {
			return nil, fmt.Errorf("GitHub authentication failed: %w", err)
		}
		runnerInstance, err = runner.NewRunnerGitHub(ctx, opts, ghClient)
	case RUN_MODE_LOCAL:
		runnerInstance, err = runner.NewRunnerLocal(ctx, opts)
	default:
		return nil, fmt.Errorf("invalid run mode: %s", opts.RunMode)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub runner: %w", err)
	}

	// Initialize runner in a goroutine to avoid blocking
	initDone := make(chan error, 1)
	go func() {
		if err := runnerInstance.Initialize(); err != nil {
			initDone <- fmt.Errorf("failed to initialize runner: %w", err)
			return
		}
		initDone <- nil
	}()

	// Set compliance config path
	complianceConfig := filepath.Join(opts.PoliciesPath, policy.COMPLIANCE_CONFIG_FILENAME)

	// Load and validate policy configuration
	fmt.Println("📋 Loading policy configuration...")
	policyConfig, err = evaluator.LoadAndValidate(complianceConfig, opts.PoliciesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load policy config: %w", err)
	}
	fmt.Printf("✅ Loaded %d policies\n\n", len(policyConfig.Policies))

	// Wait for initialization to complete
	select {
	case err := <-initDone:
		if err != nil {
			return nil, err
		}
	case <-ctx.Done():
		return nil, fmt.Errorf("initialization cancelled: %w", ctx.Err())
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

	runner, err := initialize(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	var allEnvData []envData
	hasErrors := false

	// Process each environment
	for _, environment := range opts.Environments {
		fmt.Printf("═══════════════════════════════════════════════════\n")
		fmt.Printf("🔍 Checking: %s (%s)\n", opts.Service, environment)
		fmt.Printf("═══════════════════════════════════════════════════\n\n")

		result, err := processEnvironment(ctx, opts, environment, builder, differ, evaluator, reporter, policyConfig, prInfo, comments)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Environment %s failed: %v\n\n", environment, err)
			hasErrors = true
			continue
		}

		allEnvData = append(allEnvData, *result)
		fmt.Printf("✅ Environment %s completed\n\n", environment)
	}

	if hasErrors && len(allEnvData) == 0 {
		return fmt.Errorf("all environments failed")
	}

	// Generate combined report
	fmt.Println("📝 Generating combined report...")
	multiEnvData := buildMultiEnvReport(opts.Service, allEnvData, prInfo)

	// Render combined comment/output
	var renderedComment string
	var renderErr error

	// Check if user explicitly provided templates-path
	if opts.TemplatesPath != "./templates" {
		// User specified a custom path - use it and fail if templates don't exist
		fmt.Printf("📝 Using custom templates from: %s\n", opts.TemplatesPath)
		renderedComment, renderErr = renderer.RenderWithTemplates(opts.TemplatesPath, multiEnvData)
		if renderErr != nil {
			return fmt.Errorf("failed to render comment with custom templates: %w", renderErr)
		}
	} else {
		// Check if default templates directory exists
		if _, statErr := os.Stat(opts.TemplatesPath); statErr == nil {
			// Default templates directory exists, use it
			fmt.Printf("📝 Using templates from: %s\n", opts.TemplatesPath)
			renderedComment, renderErr = renderer.RenderWithTemplates(opts.TemplatesPath, multiEnvData)
			if renderErr != nil {
				return fmt.Errorf("failed to render comment: %w", renderErr)
			}
		} else {
			// Use default embedded template
			fmt.Println("📝 Using default embedded template")
			renderedComment, renderErr = renderer.RenderString(renderer.GetDefaultCommentTemplate(), multiEnvData)
			if renderErr != nil {
				return fmt.Errorf("failed to render comment: %w", renderErr)
			}
		}
	}

	// Always prepend the marker to the rendered content (both local and GitHub modes)
	finalComment := COMMENT_MARKER + "\n\n" + renderedComment

	// Save output
	if opts.RunMode == "local" {
		if err := os.MkdirAll(opts.LcOutputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		outputFile := filepath.Join(opts.LcOutputDir, fmt.Sprintf("%s-report.md", opts.Service))
		if err := os.WriteFile(outputFile, []byte(finalComment), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}

		fmt.Printf("✅ Combined report written to: %s\n", outputFile)
	} else {
		// Post or update GitHub comment
		fmt.Println("💬 Posting results to GitHub PR...")
		owner, repo, err := github.ParseOwnerRepo(opts.GhRepo)
		if err != nil {
			return fmt.Errorf("failed to parse repository: %w", err)
		}

		fmt.Printf("   Repository: %s/%s\n", owner, repo)
		fmt.Printf("   PR Number: #%d\n", opts.GhPrNumber)

		fmt.Println("   Authenticating with GitHub...")
		ghClient, err := github.NewClient()
		if err != nil {
			return fmt.Errorf("failed to create GitHub client: %w (check GH_TOKEN environment variable)", err)
		}
		fmt.Println("   ✓ GitHub client authenticated")

		fmt.Println("   Searching for existing comment...")

		// Get all comments to check for duplicates
		allComments, err := ghClient.GetComments(ctx, owner, repo, opts.GhPrNumber)
		if err != nil {
			fmt.Printf("   ⚠️  Failed to get PR comments: %v\n", err)
		} else {
			matchingCount := 0
			for _, c := range allComments {
				if strings.Contains(c.Body, COMMENT_MARKER) {
					matchingCount++
				}
			}
			if matchingCount > 1 {
				fmt.Printf("   ⚠️  Found %d comments with the same marker, will update the latest one\n", matchingCount)
			}
		}

		existingComment, err := ghClient.FindToolComment(ctx, owner, repo, opts.GhPrNumber)
		if err != nil {
			fmt.Printf("   ⚠️  Failed to search for existing comment: %v\n", err)
			fmt.Println("   Will attempt to create new comment anyway...")
		}

		if existingComment != nil {
			// Update existing comment
			fmt.Printf("   Found existing comment (ID: %d), updating...\n", existingComment.ID)
			if err := ghClient.UpdateComment(ctx, owner, repo, existingComment.ID, finalComment); err != nil {
				return fmt.Errorf("failed to update comment: %w", err)
			}
			fmt.Println("✅ GitHub comment updated successfully")
		} else {
			// Create new comment
			fmt.Println("   No existing comment found, creating new comment...")
			newComment, err := ghClient.CreateComment(ctx, owner, repo, opts.GhPrNumber, finalComment)
			if err != nil {
				return fmt.Errorf("failed to create comment: %w", err)
			}
			fmt.Printf("✅ GitHub comment created successfully (ID: %d)\n", newComment.ID)
		}
	}

	// Check for failures across all environments
	for _, envData := range allEnvData {
		if envData.enforcement.ShouldBlock {
			return fmt.Errorf("blocking policy failures detected in %s", envData.environment)
		}
	}

	for _, envData := range allEnvData {
		if envData.enforcement.ShouldWarn {
			return fmt.Errorf("warning policy failures detected in %s", envData.environment)
		}
	}

	fmt.Println("✅ All environments passed!")
	return nil
}

func processEnvironment(
	ctx context.Context,
	opts *runner.Options,
	environment string,
	builder *kustomize.Builder,
	differ *diff.Differ,
	evaluator *policy.Evaluator,
	reporter *policy.Reporter,
	policyConfig *config.ComplianceConfig,
	prInfo *config.PullRequest,
	comments []*config.Comment,
) (*envData, error) {

	var baseManifest, headManifest []byte
	var err error

	if opts.RunMode == "local" {
		// Local mode: build from kustomize directories
		fmt.Println("🏠 Running in local mode")

		// Build paths: lcBeforeManifestsPath/service/environments/env
		beforeServicePath := filepath.Join(opts.LcBeforeManifestsPath, opts.Service, "environments", environment)
		afterServicePath := filepath.Join(opts.LcAfterManifestsPath, opts.Service, "environments", environment)

		fmt.Printf("🔨 Building base manifest from: %s\n", beforeServicePath)
		baseManifest, err = builder.Build(ctx, beforeServicePath)
		if err != nil {
			return nil, fmt.Errorf("failed to build base manifest: %w", err)
		}

		fmt.Printf("🔨 Building head manifest from: %s\n", afterServicePath)
		headManifest, err = builder.Build(ctx, afterServicePath)
		if err != nil {
			return nil, fmt.Errorf("failed to build head manifest: %w", err)
		}
	} else {
		// GitHub mode: build manifests from PR
		fmt.Println("🐙 Building manifests from PR...")
		baseManifest, headManifest, err = buildManifestsFromPR(ctx, builder, opts, environment, runner.prInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to build manifests: %w", err)
		}
	}

	// Generate diff
	fmt.Println("📊 Generating diff...")
	diffContent, err := differ.Diff(baseManifest, headManifest)
	if err != nil {
		return nil, fmt.Errorf("failed to generate diff: %w", err)
	}

	hasChanges, _ := differ.HasChanges(baseManifest, headManifest)

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

	envDiffData := config.EnvironmentDiff{
		Environment:      environment,
		HasChanges:       hasChanges,
		Content:          diffContent,
		LineCount:        addedLines + deletedLines,
		AddedLineCount:   addedLines,
		DeletedLineCount: deletedLines,
	}

	fmt.Printf("   %d lines changed\n", envDiffData.LineCount)

	// Evaluate policies
	fmt.Println("🛡️  Evaluating policies...")
	evalResult, err := evaluator.Evaluate(ctx, headManifest, policyConfig, opts.PoliciesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate policies: %w", err)
	}

	// Check for overrides
	overrides := evaluator.CheckOverrides(runner.comments, policyConfig)
	evaluator.ApplyOverrides(evalResult, overrides)

	// Determine enforcement
	enforcement := evaluator.Enforce(evalResult, overrides)

	// Generate report
	policyReport := reporter.GenerateReport(evalResult)

	fmt.Printf("   Total: %d, Passed: %d, Failed: %d, Errored: %d\n",
		evalResult.TotalPolicies, evalResult.PassedPolicies, evalResult.FailedPolicies, evalResult.ErroredPolicies)
	fmt.Printf("   %s\n", enforcement.Summary)

	return &envData{
		environment:  environment,
		diffData:     envDiffData,
		evalResult:   evalResult,
		policyReport: policyReport,
		enforcement:  enforcement,
	}, nil
}

func validateOptions(opts *options) error {
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

func buildManifestsFromPR(ctx context.Context, builder *kustomize.Builder, opts *runner.Options, environment string, prInfo *config.PullRequest) ([]byte, []byte, error) {
	// Use manifestsPath from options (default: ./services)
	servicePath := builder.GetServiceEnvironmentPath(opts.ManifestsPath, opts.Service, environment)

	// Build base manifest
	fmt.Printf("🔨 Building base manifest (ref: %s)...\n", prInfo.BaseRef)
	if err := checkoutRef(prInfo.BaseSHA); err != nil {
		return nil, nil, fmt.Errorf("failed to checkout base ref: %w", err)
	}

	baseManifest, err := builder.Build(ctx, servicePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build base manifest: %w", err)
	}

	// Build head manifest
	fmt.Printf("🔨 Building head manifest (ref: %s)...\n", prInfo.HeadRef)
	if err := checkoutRef(prInfo.HeadSHA); err != nil {
		return nil, nil, fmt.Errorf("failed to checkout head ref: %w", err)
	}

	headManifest, err := builder.Build(ctx, servicePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build head manifest: %w", err)
	}

	return baseManifest, headManifest, nil
}

func checkoutRef(ref string) error {
	cmd := exec.Command("git", "checkout", ref)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout failed: %w\nOutput: %s", err, string(output))
	}
	return nil
}
func buildMultiEnvReport(
	service string,
	allEnvData []envData,
	prInfo *config.PullRequest,
) config.MultiEnvCommentData {
	// Collect environments and diffs
	var environments []string
	var envDiffs []config.EnvironmentDiff
	summary := make(map[string]config.EnvSummary)

	for _, ed := range allEnvData {
		environments = append(environments, ed.environment)
		envDiffs = append(envDiffs, ed.diffData)

		summary[ed.environment] = config.EnvSummary{
			TotalPolicies:     ed.policyReport.TotalPolicies,
			PassedPolicies:    ed.policyReport.PassedPolicies,
			FailedPolicies:    ed.policyReport.FailedPolicies,
			ErroredPolicies:   ed.policyReport.ErroredPolicies,
			BlockingFailures:  ed.policyReport.BlockingFailures,
			WarningFailures:   ed.policyReport.WarningFailures,
			RecommendFailures: ed.policyReport.RecommendFailures,
		}
	}

	// Build policy matrix (policies x environments)
	policyMap := make(map[string]*config.MultiEnvPolicyDetail)

	for _, ed := range allEnvData {
		for _, detail := range ed.policyReport.Details {
			if policyMap[detail.Name] == nil {
				policyMap[detail.Name] = &config.MultiEnvPolicyDetail{
					Name:        detail.Name,
					Description: detail.Description,
					Level:       detail.Level,
					Results:     make(map[string]config.EnvPolicyResult),
				}
			}

			policyMap[detail.Name].Results[ed.environment] = config.EnvPolicyResult{
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
	if prInfo != nil {
		baseCommit = github.ShortSHA(prInfo.BaseSHA)
		headCommit = github.ShortSHA(prInfo.HeadSHA)
	} else {
		baseCommit = "local"
		headCommit = "local"
	}

	return config.MultiEnvCommentData{
		Service:          service,
		Environments:     environments,
		BaseCommit:       baseCommit,
		HeadCommit:       headCommit,
		EnvironmentDiffs: envDiffs,
		MultiEnvPolicyReport: config.MultiEnvPolicyReport{
			Environments: environments,
			Policies:     policies,
			Summary:      summary,
		},
		Timestamp: time.Now(),
	}
}
