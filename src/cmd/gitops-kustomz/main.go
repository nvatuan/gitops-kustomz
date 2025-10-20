package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gh-nvat/gitops-kustomz/src/pkg/config"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/diff"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/github"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/kustomize"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/policy"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/template"
	"github.com/spf13/cobra"
)

const COMMENT_MARKER = "<!-- gitops-kustomz: auto-generated comment, please do not remove -->"

var (
	Version   = "dev"
	BuildTime = "unknown"
)

type options struct {
	// Run mode
	runMode string // "github" or "local"

	// Common options
	service       string
	environments  []string // Support multiple environments
	policiesPath  string
	templatesPath string

	// GitHub mode options
	ghRepo        string
	ghPrNumber    int
	manifestsPath string // Path to services directory (default: ./services)

	// Local mode options
	lcBeforeManifestsPath string
	lcAfterManifestsPath  string
	lcOutputDir           string
}

type envData struct {
	environment  string
	diffData     config.EnvironmentDiff
	evalResult   *config.EvaluationResult
	policyReport *config.PolicyReportData
	enforcement  *config.EnforcementResult
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	opts := &options{}

	cmd := &cobra.Command{
		Use:   "gitops-kustomz",
		Short: "GitOps policy enforcement tool for Kubernetes manifests",
		Long: `gitops-kustomz enforces policy compliance for k8s GitOps repositories via GitHub PR checks.
It builds kustomize manifests, diffs them, evaluates OPA policies, and posts detailed comments on PRs.`,
		Version: fmt.Sprintf("%s (built: %s)", Version, BuildTime),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context(), opts)
		},
	}

	// Run mode
	cmd.Flags().StringVar(&opts.runMode, "run-mode", "github", "Run mode: github or local")

	// Common flags
	cmd.Flags().StringVar(&opts.service, "service", "", "Service name (required)")
	cmd.Flags().StringSliceVar(&opts.environments, "environments", []string{}, "Environments to check (comma-separated, e.g., stg,prod) (required)")
	cmd.Flags().StringVar(&opts.policiesPath, "policies-path", "./policies", "Path to policies directory (contains compliance-config.yaml)")
	cmd.Flags().StringVar(&opts.templatesPath, "templates-path", "./templates", "Path to templates directory")

	// GitHub mode flags
	cmd.Flags().StringVar(&opts.ghRepo, "gh-repo", "", "GitHub repository (e.g., org/repo) [github mode]")
	cmd.Flags().IntVar(&opts.ghPrNumber, "gh-pr-number", 0, "GitHub PR number [github mode]")
	cmd.Flags().StringVar(&opts.manifestsPath, "manifests-path", "./services", "Path to services directory containing service folders [github mode]")

	// Local mode flags
	cmd.Flags().StringVar(&opts.lcBeforeManifestsPath, "lc-before-manifests-path", "", "Path to before/base services directory [local mode]")
	cmd.Flags().StringVar(&opts.lcAfterManifestsPath, "lc-after-manifests-path", "", "Path to after/head services directory [local mode]")
	cmd.Flags().StringVar(&opts.lcOutputDir, "lc-output-dir", "./output", "Local mode output directory [local mode]")

	// Mark required flags
	_ = cmd.MarkFlagRequired("service")
	_ = cmd.MarkFlagRequired("environments")

	return cmd
}

func run(ctx context.Context, opts *options) error {
	// Validate options
	if err := validateOptions(opts); err != nil {
		return fmt.Errorf("invalid options: %w", err)
	}

	// Set compliance config path
	complianceConfig := filepath.Join(opts.policiesPath, "compliance-config.yaml")

	// Initialize components
	builder := kustomize.NewBuilder()
	differ := diff.NewDiffer()
	evaluator := policy.NewEvaluator()
	reporter := policy.NewReporter()
	renderer := template.NewRenderer()

	// Load and validate policy configuration
	fmt.Println("📋 Loading policy configuration...")
	policyConfig, err := evaluator.LoadAndValidate(complianceConfig, opts.policiesPath)
	if err != nil {
		return fmt.Errorf("failed to load policy config: %w", err)
	}
	fmt.Printf("✅ Loaded %d policies\n\n", len(policyConfig.Policies))

	// Collect data from all environments
	var allEnvData []envData
	var prInfo *config.PullRequest
	var comments []*config.Comment
	hasErrors := false

	// Get PR info once (for GitHub mode)
	if opts.runMode == "github" {
		ghClient, err := github.NewClient()
		if err != nil {
			return fmt.Errorf("GitHub authentication failed: %w", err)
		}

		owner, repo := parseRepo(opts.ghRepo)
		fmt.Printf("📥 Fetching PR #%d information...\n", opts.ghPrNumber)
		prInfo, err = ghClient.GetPR(ctx, owner, repo, opts.ghPrNumber)
		if err != nil {
			return fmt.Errorf("failed to get PR info: %w", err)
		}

		comments, err = ghClient.GetComments(ctx, owner, repo, opts.ghPrNumber)
		if err != nil {
			return fmt.Errorf("failed to get PR comments: %w", err)
		}
	} else {
		prInfo = &config.PullRequest{
			BaseSHA: "base",
			HeadSHA: "head",
		}
	}

	// Process each environment
	for _, environment := range opts.environments {
		fmt.Printf("═══════════════════════════════════════════════════\n")
		fmt.Printf("🔍 Checking: %s (%s)\n", opts.service, environment)
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
	multiEnvData := buildMultiEnvReport(opts.service, allEnvData, prInfo)

	// Render combined comment/output
	var renderedComment string
	var renderErr error

	// Check if user explicitly provided templates-path
	if opts.templatesPath != "./templates" {
		// User specified a custom path - use it and fail if templates don't exist
		fmt.Printf("📝 Using custom templates from: %s\n", opts.templatesPath)
		renderedComment, renderErr = renderer.RenderWithTemplates(opts.templatesPath, multiEnvData)
		if renderErr != nil {
			return fmt.Errorf("failed to render comment with custom templates: %w", renderErr)
		}
	} else {
		// Check if default templates directory exists
		if _, statErr := os.Stat(opts.templatesPath); statErr == nil {
			// Default templates directory exists, use it
			fmt.Printf("📝 Using templates from: %s\n", opts.templatesPath)
			renderedComment, renderErr = renderer.RenderWithTemplates(opts.templatesPath, multiEnvData)
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
	if opts.runMode == "local" {
		if err := os.MkdirAll(opts.lcOutputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		outputFile := filepath.Join(opts.lcOutputDir, fmt.Sprintf("%s-report.md", opts.service))
		if err := os.WriteFile(outputFile, []byte(finalComment), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}

		fmt.Printf("✅ Combined report written to: %s\n", outputFile)
	} else {
		// Post or update GitHub comment
		fmt.Println("💬 Posting results to GitHub PR...")
		owner, repo := parseRepo(opts.ghRepo)

		fmt.Printf("   Repository: %s/%s\n", owner, repo)
		fmt.Printf("   PR Number: #%d\n", opts.ghPrNumber)

		fmt.Println("   Authenticating with GitHub...")
		ghClient, err := github.NewClient()
		if err != nil {
			return fmt.Errorf("failed to create GitHub client: %w (check GH_TOKEN environment variable)", err)
		}
		fmt.Println("   ✓ GitHub client authenticated")

		fmt.Println("   Searching for existing comment...")

		// Get all comments to check for duplicates
		allComments, err := ghClient.GetComments(ctx, owner, repo, opts.ghPrNumber)
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

		existingComment, err := ghClient.FindToolComment(ctx, owner, repo, opts.ghPrNumber)
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
			newComment, err := ghClient.CreateComment(ctx, owner, repo, opts.ghPrNumber, finalComment)
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
	opts *options,
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

	if opts.runMode == "local" {
		// Local mode: build from kustomize directories
		fmt.Println("🏠 Running in local mode")

		// Build paths: lcBeforeManifestsPath/service/environments/env
		beforeServicePath := filepath.Join(opts.lcBeforeManifestsPath, opts.service, "environments", environment)
		afterServicePath := filepath.Join(opts.lcAfterManifestsPath, opts.service, "environments", environment)

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
		baseManifest, headManifest, err = buildManifestsFromPR(ctx, builder, opts, environment, prInfo)
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
	envDiffData := config.EnvironmentDiff{
		Environment: environment,
		HasChanges:  hasChanges,
		Content:     diffContent,
		LineCount:   strings.Count(diffContent, "\n"),
	}

	fmt.Printf("   %d lines changed\n", envDiffData.LineCount)

	// Evaluate policies
	fmt.Println("🛡️  Evaluating policies...")
	evalResult, err := evaluator.Evaluate(ctx, headManifest, policyConfig, opts.policiesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate policies: %w", err)
	}

	// Check for overrides
	overrides := evaluator.CheckOverrides(comments, policyConfig)
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
	if opts.service == "" {
		return fmt.Errorf("service is required")
	}

	if len(opts.environments) == 0 {
		return fmt.Errorf("at least one environment is required")
	}

	// Validate run mode
	if opts.runMode != "github" && opts.runMode != "local" {
		return fmt.Errorf("run-mode must be 'github' or 'local', got: %s", opts.runMode)
	}

	// Validate mode-specific options
	if opts.runMode == "local" {
		if opts.lcBeforeManifestsPath == "" || opts.lcAfterManifestsPath == "" {
			return fmt.Errorf("local mode requires --lc-before-manifests-path and --lc-after-manifests-path")
		}
	} else {
		// GitHub mode
		if opts.ghRepo == "" {
			return fmt.Errorf("github mode requires --gh-repo")
		}
		if opts.ghPrNumber == 0 {
			return fmt.Errorf("github mode requires --gh-pr-number")
		}
	}

	return nil
}

func parseRepo(repo string) (string, string) {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

func buildManifestsFromPR(ctx context.Context, builder *kustomize.Builder, opts *options, environment string, prInfo *config.PullRequest) ([]byte, []byte, error) {
	// Use manifestsPath from options (default: ./services)
	servicePath := builder.GetServiceEnvironmentPath(opts.manifestsPath, opts.service, environment)

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

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
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

	return config.MultiEnvCommentData{
		Service:          service,
		Environments:     environments,
		BaseCommit:       shortSHA(prInfo.BaseSHA),
		HeadCommit:       shortSHA(prInfo.HeadSHA),
		EnvironmentDiffs: envDiffs,
		MultiEnvPolicyReport: config.MultiEnvPolicyReport{
			Environments: environments,
			Policies:     policies,
			Summary:      summary,
		},
		Timestamp: time.Now(),
	}
}
