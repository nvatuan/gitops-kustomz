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

	"github.com/gh-nvat/gitops-kustomz/src/pkg/models"
	"gopkg.in/yaml.v2"
)

const (
	COMPLIANCE_CONFIG_FILENAME = "compliance-config.yaml"
)

// // PolicyEvaluator defines the interface for policy evaluation operations
// type PolicyEvaluator interface {
// 	// LoadAndValidate loads and validates the compliance configuration
// 	LoadAndValidate(configPath, policiesPath string) (*models.ComplianceConfig, error)
// 	// Evaluate evaluates all policies against the manifest
// 	Evaluate(ctx context.Context, manifest []byte, cfg *models.ComplianceConfig, policiesPath string) (*models.EvaluationResult, error)
// 	// CheckOverrides checks for policy override comments in PR comments
// 	CheckOverrides(comments []*models.Comment, cfg *models.ComplianceConfig) map[string]bool
// 	// Enforce determines if the evaluation result should block the PR
// 	Enforce(result *models.EvaluationResult, overrides map[string]bool) *models.EnforcementResult
// 	// ApplyOverrides applies policy overrides to the evaluation result
// 	ApplyOverrides(result *models.EvaluationResult, overrides map[string]bool)
// }

type PolicyEvaluatorInterface interface {
	LoadAndValidate(policiesPath string) (*models.ComplianceConfig, error)
}

const (
	POLICY_LEVEL_RECOMMEND     = "RECOMMEND"
	POLICY_LEVEL_WARNING       = "WARNING"
	POLICY_LEVEL_BLOCK         = "BLOCK"
	POLICY_LEVEL_OVERRIDE      = "OVERRIDE"
	POLICY_LEVEL_NOT_IN_EFFECT = "NOT_IN_EFFECT"
	POLICY_LEVEL_UNKNOWN       = ""
)

type EvaluatorData struct {
	models.ComplianceConfig

	// map policy id to full path to policy file
	fullPathToPolicy    map[string]string
	evalFailMsgOfPolicy map[string][]string

	// enforcements levels of policies Ids
	policyIdToLevel       map[string]string
	overrideCmdToPolicyId map[string]string
}

type PolicyEvaluator struct {
	policiesPath string
	data         EvaluatorData
}

func NewPolicyEvaluator(policiesPath string) *PolicyEvaluator {
	return &PolicyEvaluator{
		policiesPath: policiesPath,
		data:         EvaluatorData{},
	}
}

// LoadAndValidate loads and validates the compliance configuration
func (e *PolicyEvaluator) LoadAndValidate() error {
	// Load configuration
	if err := e.loadComplianceConfig(); err != nil {
		return err
	}

	// Validate configuration structure
	if err := e.validateComplianceConfig(); err != nil {
		return err
	}

	// Validate policy files exist and check for tests
	for id, policy := range e.data.ComplianceConfig.Policies {
		policyPath := filepath.Join(e.policiesPath, policy.FilePath)
		if _, err := os.Stat(policyPath); os.IsNotExist(err) {
			return fmt.Errorf("policy %s: file not found: %s", id, policyPath)
		}

		// Check for test file (support both .rego and .opa extensions)
		var testPath string
		if strings.HasSuffix(policyPath, ".rego") {
			testPath = strings.TrimSuffix(policyPath, ".rego") + "_test.rego"
		} else {
			return fmt.Errorf("policy %s: unsupported file extension (must be .rego)", id)
		}

		if _, err := os.Stat(testPath); os.IsNotExist(err) {
			return fmt.Errorf("each policy must have testpolicy %s: test file not found: %s", id, testPath)
		}

		// Set full path to policy file
		e.data.fullPathToPolicy[id] = policyPath

		// check override cmd
		if policy.Enforcement.Override.Comment == "" {
			continue
		}
		if _, ok := e.data.overrideCmdToPolicyId[policy.Enforcement.Override.Comment]; ok {
			return fmt.Errorf("policy %s: use another command, this override command already exists: %s", id, policy.Enforcement.Override.Comment)
		}
		e.data.overrideCmdToPolicyId[policy.Enforcement.Override.Comment] = id
	}
	return nil
}

// LoadComplianceConfig loads the compliance configuration from a YAML file
func (e *PolicyEvaluator) loadComplianceConfig() error {
	configPath := filepath.Join(e.policiesPath, COMPLIANCE_CONFIG_FILENAME)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read compliance config: %w", err)
	}

	if err := yaml.Unmarshal(data, &e.data.ComplianceConfig); err != nil {
		return fmt.Errorf("failed to parse compliance config: %w", err)
	}
	return nil
}

// ValidateComplianceConfig validates the common fields
func (e *PolicyEvaluator) validateComplianceConfig() error {
	if len(e.data.ComplianceConfig.Policies) == 0 {
		return fmt.Errorf("no policies defined in compliance config")
	}

	for id, policy := range e.data.ComplianceConfig.Policies {
		if policy.Name == "" {
			return fmt.Errorf("policy %s: name is required", id)
		}
		if policy.Type == "" {
			return fmt.Errorf("policy %s: type is required", id)
		}
		if policy.Type != "opa" {
			return fmt.Errorf("policy %s: unsupported type %s (only 'opa' is supported)", id, policy.Type)
		}
		if policy.FilePath == "" {
			return fmt.Errorf("policy %s: filePath is required", id)
		}

		// Validate enforcement dates are in order if set
		if policy.Enforcement.InEffectAfter != nil && policy.Enforcement.IsWarningAfter != nil {
			if policy.Enforcement.IsWarningAfter.Before(*policy.Enforcement.InEffectAfter) {
				return fmt.Errorf("policy %s: isWarningAfter cannot be before inEffectAfter", id)
			}
		}
		if policy.Enforcement.IsWarningAfter != nil && policy.Enforcement.IsBlockingAfter != nil {
			if policy.Enforcement.IsBlockingAfter.Before(*policy.Enforcement.IsWarningAfter) {
				return fmt.Errorf("policy %s: isBlockingAfter cannot be before isWarningAfter", id)
			}
		}

		// override comment not too long
		if policy.Enforcement.Override.Comment != "" && len(policy.Enforcement.Override.Comment) > 255 {
			return fmt.Errorf("policy %s: override comment is too long (max 255 characters)", id)
		}
	}

	return nil
}

// Evaluate evaluates all policies against the manifest using conftest and store the evaluation results in the EvaluatorData
func (e *PolicyEvaluator) Evaluate(
	ctx context.Context,
	manifest []byte,
) error {
	// Write manifest to temporary file for conftest
	tmpFile, err := os.CreateTemp("", "manifest-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
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
		return fmt.Errorf("failed to write manifest to temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Evaluate each policy using conftest
	for id, _ := range e.data.ComplianceConfig.Policies {
		failMsgs, err := e.evaluatePolicyWithConftest(
			ctx, id, e.data.fullPathToPolicy[id], tmpFile.Name(),
		)
		if err != nil {
			return fmt.Errorf("failed to evaluate policy %s: %w", id, err)
		}
		e.data.evalFailMsgOfPolicy[id] = failMsgs
	}

	return nil
}

// evaluatePolicyWithConftest evaluates a single policy using conftest
// returns: failureMsgs, evalError
func (e *PolicyEvaluator) evaluatePolicyWithConftest(
	ctx context.Context,
	id string,
	singlePolicyPath string, manifestPath string,
) ([]string, error) {
	cmd := exec.CommandContext(ctx,
		"conftest", "test", "--all-namespaces", "--combine",
		"--policy", singlePolicyPath,
		manifestPath,
		"-o", "json",
	)
	outputBytes, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("conftest command failed: %w\nOutput: %s", err, string(outputBytes))
	}

	// Sample conftest output
	// 	[
	//   {
	//     "filename": "Combined",
	//     "namespace": "main",
	//     "successes": 2,
	//     "failures": [
	//       {
	//         "msg": "Deployment 'prod-my-app' must have at least 2 replicas for high availability, found: 1",
	//         "metadata": {
	//           "query": "data.main.deny"
	//         }
	//       }
	//     ]
	//   }
	// ]
	outputJson := []struct {
		Filename  string `json:"filename"`
		Namespace string `json:"namespace"`
		Successes int    `json:"successes"`
		Failures  []struct {
			Msg      string `json:"msg"`
			Metadata struct {
				Query string `json:"query"`
			}
		}
	}{}
	if err := json.Unmarshal(outputBytes, &outputJson); err != nil {
		return nil, fmt.Errorf("failed to parse conftest output: %w", err)
	}

	if len(outputJson) == 0 {
		return nil, fmt.Errorf("no results found in conftest output: %s", string(outputBytes))
	}
	// Success case: [
	// 	 {
	// 			"filename": "Combined",
	// 			"namespace": "main",
	// 			"successes": 3
	//	 }
	// ]
	if len(outputJson[0].Failures) == 0 {
		return []string{}, nil
	}

	failureMsgs := []string{}
	for _, failure := range outputJson[0].Failures {
		failureMsgs = append(failureMsgs, failure.Msg)
	}
	return failureMsgs, nil
}

// DetermineEnforcementLevel determines the current enforcement level based on time and overrides
// Set the results to internal struct data
func (e *PolicyEvaluator) DetermineEnforcementLevel(
	comments []string,
) error {
	now := time.Now()

	for _, comment := range comments {
		if _, ok := e.data.overrideCmdToPolicyId[comment]; ok {
			e.data.policyIdToLevel[e.data.overrideCmdToPolicyId[comment]] = POLICY_LEVEL_OVERRIDE
		}
	}

	for policyId, policy := range e.data.ComplianceConfig.Policies {
		enforcementLevel := POLICY_LEVEL_UNKNOWN
		enforcement := policy.Enforcement

		if enforcement.InEffectAfter != nil && now.Before(*enforcement.InEffectAfter) {
			enforcementLevel = POLICY_LEVEL_NOT_IN_EFFECT
		}
		if enforcement.IsWarningAfter != nil && now.Before(*enforcement.IsWarningAfter) {
			enforcementLevel = POLICY_LEVEL_RECOMMEND
		}
		if enforcement.IsBlockingAfter != nil && now.Before(*enforcement.IsBlockingAfter) {
			enforcementLevel = POLICY_LEVEL_WARNING
		}
		if enforcement.IsBlockingAfter != nil && !now.Before(*enforcement.IsBlockingAfter) {
			enforcementLevel = POLICY_LEVEL_BLOCK
		}

		e.data.policyIdToLevel[policyId] = enforcementLevel
	}

	return nil
}
