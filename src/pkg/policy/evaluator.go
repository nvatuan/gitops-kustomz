package policy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gh-nvat/gitops-kustomz/src/pkg/config"
	"github.com/open-policy-agent/opa/rego"
	"gopkg.in/yaml.v3"
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
		} else if strings.HasSuffix(policyPath, ".opa") {
			testPath = strings.TrimSuffix(policyPath, ".opa") + "_test.opa"
		} else {
			return nil, fmt.Errorf("policy %s: unsupported file extension (must be .rego or .opa)", id)
		}

		if _, err := os.Stat(testPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("policy %s: test file not found: %s", id, testPath)
		}
	}

	return cfg, nil
}

// Evaluate evaluates all policies against the manifest
func (e *Evaluator) Evaluate(ctx context.Context, manifest []byte, cfg *config.ComplianceConfig, policiesPath string) (*config.EvaluationResult, error) {
	result := &config.EvaluationResult{
		TotalPolicies: len(cfg.Policies),
		PolicyResults: make([]config.PolicyResult, 0, len(cfg.Policies)),
	}

	// Parse manifest into individual resources
	resources, err := e.parseManifest(manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Evaluate each policy
	for id, policy := range cfg.Policies {
		policyResult := e.evaluatePolicy(ctx, id, policy, resources, policiesPath)
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

// evaluatePolicy evaluates a single policy against all resources
func (e *Evaluator) evaluatePolicy(ctx context.Context, id string, policy config.PolicyConfig, resources []map[string]interface{}, policiesPath string) config.PolicyResult {
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

	// Load OPA policy
	policyPath := filepath.Join(policiesPath, policy.FilePath)
	policyContent, err := os.ReadFile(policyPath)
	if err != nil {
		result.Status = POLICY_STATUS_ERROR
		result.Error = fmt.Sprintf("Failed to read policy file: %v", err)
		return result
	}

	// Evaluate policy for each resource
	for _, resource := range resources {
		violations, err := e.evaluateResourceWithOPA(ctx, policyContent, resource)
		if err != nil {
			result.Status = POLICY_STATUS_ERROR
			result.Error = fmt.Sprintf("Policy evaluation failed: %v", err)
			return result
		}

		// Add violations
		for _, v := range violations {
			result.Violations = append(result.Violations, config.Violation{
				Message:  v,
				Resource: fmt.Sprintf("%s/%s", resource["kind"], resource["metadata"].(map[string]interface{})["name"]),
			})
		}
	}

	// Set status based on violations
	if len(result.Violations) > 0 {
		result.Status = POLICY_STATUS_FAIL
	}

	return result
}

// evaluateResourceWithOPA evaluates a single resource using OPA
func (e *Evaluator) evaluateResourceWithOPA(ctx context.Context, policyContent []byte, resource map[string]interface{}) ([]string, error) {
	// Prepare input for OPA
	input := map[string]interface{}{
		"request": map[string]interface{}{
			"kind": map[string]interface{}{
				"kind":    resource["kind"],
				"version": resource["apiVersion"],
			},
			"object": resource,
			"namespace": func() string {
				if metadata, ok := resource["metadata"].(map[string]interface{}); ok {
					if ns, ok := metadata["namespace"].(string); ok {
						return ns
					}
				}
				return "default"
			}(),
		},
	}

	// Create Rego query
	query, err := rego.New(
		rego.Query("data.kustomization.deny"),
		rego.Module("policy.rego", string(policyContent)),
	).PrepareForEval(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to prepare OPA query: %w", err)
	}

	// Evaluate
	results, err := query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate policy: %w", err)
	}

	// Extract violations
	var violations []string
	if len(results) > 0 && len(results[0].Expressions) > 0 {
		if denySet, ok := results[0].Expressions[0].Value.([]interface{}); ok {
			for _, v := range denySet {
				if msg, ok := v.(string); ok {
					violations = append(violations, msg)
				}
			}
		}
	}

	return violations, nil
}

// parseManifest parses a YAML manifest into individual resources
func (e *Evaluator) parseManifest(manifest []byte) ([]map[string]interface{}, error) {
	var resources []map[string]interface{}

	// Split by "---" YAML document separator
	docs := strings.Split(string(manifest), "\n---\n")

	for _, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		var resource map[string]interface{}
		if err := yaml.Unmarshal([]byte(doc), &resource); err != nil {
			continue // Skip invalid documents
		}

		// Only include resources with kind (skip ConfigMap data, etc.)
		if _, ok := resource["kind"]; ok {
			resources = append(resources, resource)
		}
	}

	return resources, nil
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
