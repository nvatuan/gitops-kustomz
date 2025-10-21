package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gh-nvat/gitops-kustomz/src/pkg/config"
)

const (
	POLICY_STATUS_PASS  = "PASS"
	POLICY_STATUS_FAIL  = "FAIL"
	POLICY_STATUS_ERROR = "ERROR"

	POLICY_LEVEL_DISABLED  = "DISABLED"
	POLICY_LEVEL_RECOMMEND = "RECOMMEND"
	POLICY_LEVEL_WARNING   = "WARNING"
	POLICY_LEVEL_BLOCK     = "BLOCK"
)

// PolicyEvaluator defines the interface for policy evaluation operations
type PolicyEvaluator interface {
	// LoadAndValidate loads and validates the compliance configuration
	LoadAndValidate(configPath, policiesPath string) (*config.ComplianceConfig, error)
	// Evaluate evaluates all policies against the manifest
	Evaluate(ctx context.Context, manifest []byte, cfg *config.ComplianceConfig, policiesPath string) (*config.EvaluationResult, error)
	// CheckOverrides checks for policy override comments in PR comments
	CheckOverrides(comments []*config.Comment, cfg *config.ComplianceConfig) map[string]bool
	// Enforce determines if the evaluation result should block the PR
	Enforce(result *config.EvaluationResult, overrides map[string]bool) *config.EnforcementResult
	// ApplyOverrides applies policy overrides to the evaluation result
	ApplyOverrides(result *config.EvaluationResult, overrides map[string]bool)
}

// Evaluator handles policy evaluation
type Evaluator struct {
	loader *config.Loader
}

// Ensure Evaluator implements PolicyEvaluator
var _ PolicyEvaluator = (*Evaluator)(nil)

// NewEvaluator creates a new policy evaluator
func NewEvaluator() *Evaluator {
	return &Evaluator{
		loader: config.NewLoader(),
	}
}

// LoadAndValidate loads and validates the compliance configuration
func (e *Evaluator) LoadAndValidate(configPath, policiesPath string) (*config.ComplianceConfig, error) {
	// Load configuration
	cfg, err := e.loader.LoadComplianceConfig(configPath)
	if err != nil {
		return nil, err
	}

	// Validate configuration structure
	if err := e.loader.ValidateComplianceConfig(cfg); err != nil {
		return nil, err
	}

	// Validate policy files exist
	for id, policy := range cfg.Policies {
		policyPath := filepath.Join(policiesPath, policy.FilePath)
		if _, err := os.Stat(policyPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("policy %s: file not found: %s", id, policyPath)
		}

		// Check for test file (support both .rego and .opa extensions)
		var testPath string
		if strings.HasSuffix(policyPath, ".rego") {
			testPath = strings.TrimSuffix(policyPath, ".rego") + "_test.rego"
		} else {
			return nil, fmt.Errorf("policy %s: unsupported file extension (must be .rego)", id)
		}

		if _, err := os.Stat(testPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("each policy must have testpolicy %s: test file not found: %s", id, testPath)
		}
	}

	return cfg, nil
}

// Evaluate evaluates all policies against the manifest using conftest
func (e *Evaluator) Evaluate(ctx context.Context, manifest []byte, cfg *config.ComplianceConfig, policiesPath string) (*config.EvaluationResult, error) {
	result := &config.EvaluationResult{
		TotalPolicies: len(cfg.Policies),
		PolicyResults: make([]config.PolicyResult, 0, len(cfg.Policies)),
	}

	// Write manifest to temporary file for conftest
	tmpFile, err := os.CreateTemp("", "manifest-*.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			// Log error but don't fail the operation
			fmt.Printf("Warning: failed to remove temp file %s: %v\n", tmpFile.Name(), err)
		}
	}()
	defer func() {
		if err := tmpFile.Close(); err != nil {
			// Log error but don't fail the operation
			fmt.Printf("Warning: failed to close temp file: %v\n", err)
		}
	}()

	if _, err := tmpFile.Write(manifest); err != nil {
		return nil, fmt.Errorf("failed to write manifest to temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temp file: %w", err)
	}

	// Evaluate each policy using conftest
	for id, policy := range cfg.Policies {
		policyResult := e.evaluatePolicyWithConftest(ctx, id, policy, tmpFile.Name(), policiesPath)
		result.PolicyResults = append(result.PolicyResults, policyResult)

		switch policyResult.Status {
		case POLICY_STATUS_PASS:
			result.PassedPolicies++
		case POLICY_STATUS_FAIL:
			result.FailedPolicies++
		case POLICY_STATUS_ERROR:
			result.ErroredPolicies++
		}
	}

	return result, nil
}

// evaluatePolicyWithConftest evaluates a single policy using conftest
func (e *Evaluator) evaluatePolicyWithConftest(ctx context.Context, id string, policy config.PolicyConfig, manifestPath, policiesPath string) config.PolicyResult {
	result := config.PolicyResult{
		PolicyID:   id,
		PolicyName: policy.Name,
		Status:     POLICY_STATUS_PASS,
		Violations: []config.Violation{},
	}

	// Determine enforcement level
	result.Level = e.determineEnforcementLevel(policy.Enforcement)

	// If policy is not in effect, skip it
	if result.Level == "DISABLED" {
		return result
	}

	// Use opa to evaluate the policy
	violations, err := e.evaluateResourceWithOPA(ctx, manifestPath, policiesPath)
	if err != nil {
		result.Status = POLICY_STATUS_ERROR
		result.Error = fmt.Sprintf("Policy evaluation failed: %v", err)
		return result
	}

	// Add violations
	for _, v := range violations {
		result.Violations = append(result.Violations, config.Violation{
			Message:  v,
			Resource: "manifest", // conftest doesn't provide individual resource names in this context
		})
	}

	// Set status based on violations
	if len(result.Violations) > 0 {
		result.Status = POLICY_STATUS_FAIL
	}

	return result
}

// evaluateResourceWithOPA evaluates manifest using OPA binary directly
func (e *Evaluator) evaluateResourceWithOPA(ctx context.Context, manifestPath, policiesPath string) ([]string, error) {
	// Use opa eval command directly with YAML input
	cmd := exec.CommandContext(ctx, "opa", "eval",
		"--data", policiesPath,
		"--input", manifestPath,
		"--format", "json",
		"data.kustomization.deny")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("opa command failed: %w\nOutput: %s", err, string(output))
	}

	// Parse OPA output
	violations, err := e.parseOPAOutput(output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OPA output: %w", err)
	}

	return violations, nil
}

// parseOPAOutput parses OPA JSON output to extract violations
func (e *Evaluator) parseOPAOutput(output []byte) ([]string, error) {
	var result struct {
		Result []interface{} `json:"result"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse OPA output: %w", err)
	}

	var violations []string
	if len(result.Result) > 0 {
		if denySet, ok := result.Result[0].([]interface{}); ok {
			for _, v := range denySet {
				if msg, ok := v.(string); ok {
					violations = append(violations, msg)
				}
			}
		}
	}

	return violations, nil
}

// determineEnforcementLevel determines the current enforcement level based on time
func (e *Evaluator) determineEnforcementLevel(enforcement config.EnforcementConfig) string {
	now := time.Now()

	// Check if policy is in effect
	if enforcement.InEffectAfter != nil && now.Before(*enforcement.InEffectAfter) {
		return POLICY_LEVEL_DISABLED
	}

	// Check blocking level
	if enforcement.IsBlockingAfter != nil && !now.Before(*enforcement.IsBlockingAfter) {
		return POLICY_LEVEL_BLOCK
	}

	// Check warning level
	if enforcement.IsWarningAfter != nil && !now.Before(*enforcement.IsWarningAfter) {
		return POLICY_LEVEL_WARNING
	}

	// Default to recommend if in effect
	if enforcement.InEffectAfter != nil {
		return POLICY_LEVEL_RECOMMEND
	}

	return POLICY_LEVEL_DISABLED
}

// CheckOverrides checks PR comments for policy override commands
func (e *Evaluator) CheckOverrides(comments []*config.Comment, cfg *config.ComplianceConfig) map[string]bool {
	overrides := make(map[string]bool)

	for policyID, policy := range cfg.Policies {
		if policy.Enforcement.Override.Comment == "" {
			continue
		}

		// Check if override comment exists
		for _, comment := range comments {
			if strings.Contains(comment.Body, policy.Enforcement.Override.Comment) {
				overrides[policyID] = true
				break
			}
		}
	}

	return overrides
}

// Enforce determines the enforcement action based on results and overrides
func (e *Evaluator) Enforce(result *config.EvaluationResult, overrides map[string]bool) *config.EnforcementResult {
	enforcement := &config.EnforcementResult{}

	blockingCount := 0
	warningCount := 0

	for _, pr := range result.PolicyResults {
		if pr.Status != POLICY_STATUS_FAIL {
			continue
		}

		// Check if overridden
		if overrides[pr.PolicyID] {
			continue
		}

		switch pr.Level {
		case POLICY_LEVEL_BLOCK:
			blockingCount++
			enforcement.ShouldBlock = true
		case POLICY_LEVEL_WARNING:
			warningCount++
			enforcement.ShouldWarn = true
		}
	}

	if blockingCount > 0 {
		enforcement.Summary = fmt.Sprintf("%d blocking policy failure(s)", blockingCount)
	} else if warningCount > 0 {
		enforcement.Summary = fmt.Sprintf("%d warning policy failure(s)", warningCount)
	} else {
		enforcement.Summary = "All checks passed"
	}

	return enforcement
}

// ApplyOverrides applies overrides to policy results
func (e *Evaluator) ApplyOverrides(result *config.EvaluationResult, overrides map[string]bool) {
	for i := range result.PolicyResults {
		if overrides[result.PolicyResults[i].PolicyID] {
			result.PolicyResults[i].Overridden = true
		}
	}
}
