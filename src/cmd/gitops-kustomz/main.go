package main

import (
	"fmt"
	"os"

	"github.com/gh-nvat/gitops-kustomz/src/internal/runner"
	"github.com/spf13/cobra"
)

const COMMENT_MARKER = "<!-- gitops-kustomz: auto-generated comment, please do not remove -->"

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// newRootCmd creates the root command, parse args from CLI
func newRootCmd() *cobra.Command {
	opts := &runner.Options{}

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
	cmd.Flags().StringVar(&opts.RunMode, "run-mode", "github", "Run mode: github or local")

	// Common flags
	cmd.Flags().StringVar(&opts.Service, "service", "", "Service name (required)")
	cmd.Flags().StringSliceVar(&opts.Environments, "environments", []string{},
		"Environments to check (comma-separated, e.g., stg,prod) (required)")
	cmd.Flags().StringVar(&opts.PoliciesPath, "policies-path", "./policies",
		"Path to policies directory (contains compliance-config.yaml)")
	cmd.Flags().StringVar(&opts.TemplatesPath, "templates-path", "./templates",
		"Path to templates directory")

	// GitHub mode flags
	cmd.Flags().StringVar(&opts.GhRepo, "gh-repo", "",
		"GitHub repository (e.g., org/repo) [github mode]")
	cmd.Flags().IntVar(&opts.GhPrNumber, "gh-pr-number", 0,
		"GitHub PR number [github mode]")
	cmd.Flags().StringVar(&opts.ManifestsPath, "manifests-path", "./services",
		"Path to services directory containing service folders [github mode]")

	// Local mode flags
	cmd.Flags().StringVar(&opts.LcBeforeManifestsPath, "lc-before-manifests-path", "",
		"Path to before/base services directory [local mode]")
	cmd.Flags().StringVar(&opts.LcAfterManifestsPath, "lc-after-manifests-path", "",
		"Path to after/head services directory [local mode]")
	cmd.Flags().StringVar(&opts.LcOutputDir, "lc-output-dir", "./output",
		"Local mode output directory [local mode]")

	// Mark required flags
	_ = cmd.MarkFlagRequired("service")
	_ = cmd.MarkFlagRequired("environments")

	return cmd
}
