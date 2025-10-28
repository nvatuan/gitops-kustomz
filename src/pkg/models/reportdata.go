package models

import "time"

// ReportData represents the complete report data structure
type ReportData struct {
	Service      string    `json:"service"`
	Timestamp    time.Time `json:"timestamp"`
	BaseCommit   string    `json:"baseCommit"`
	HeadCommit   string    `json:"headCommit"`
	Environments []string  `json:"environments"`

	// Manifest changes per environment
	ManifestChanges map[string]EnvironmentDiff `json:"manifestChanges"`

	// Policy evaluation results
	PolicyEvaluation PolicyEvaluation `json:"policyEvaluation"`
}

// EnvironmentDiff represents diff data for a single environment
type EnvironmentDiff struct {
	LineCount        int `json:"lineCount"`
	AddedLineCount   int `json:"addedLineCount"`
	DeletedLineCount int `json:"deletedLineCount"`

	ContentGHFilePath *string `json:"contentGHFilePath"` // file path in the runner's output directory if the diff is too long
	ContentType       string  `json:"contentType"`       // "text" or "ext_ghartifact"
	Content           string  `json:"content"`           // diff text OR artifact URL
}

// PolicyEvaluationSummary represents the overall policy evaluation results
type PolicyEvaluation struct {
	// Summary table: Environment -> Success/Failed/Errored counts
	EnvironmentSummary map[string]EnvironmentSummaryEnv `json:"environmentSummary"`

	// Detailed policy matrix
	PolicyMatrix map[string]PolicyMatrix `json:"policyMatrix"`
}

type EnvironmentSummaryEnv struct {
	PassingStatus EnforcementPassingStatus `json:"passingStatus"`
	PolicyCounts  PolicyCounts             `json:"policyCounts"`
}

type EnforcementPassingStatus struct {
	PassBlockingCheck  bool `json:"passBlockingCheck"`
	PassWarningCheck   bool `json:"passWarningCheck"`
	PassRecommendCheck bool `json:"passRecommendCheck"`
}

// PolicyCounts represents the count of policies by status for an environment
type PolicyCounts struct {
	TotalCount          int `json:"totalCount"`
	TotalSuccess        int `json:"totalSuccess"`        // total number of policies that passed
	TotalFailed         int `json:"totalFailed"`         // total number of policies of level RECOMMEND, WARNING, BLOCKING that failed
	TotalOmitted        int `json:"totalOmitted"`        // total number of policies of level OVERRIDE, NOT_IN_EFFECT that either failed or passed
	TotalOmittedFailed  int `json:"totalOmittedFailed"`  // total number of policies of level OVERRIDE, NOT_IN_EFFECT that failed
	TotalOmittedSuccess int `json:"totalOmittedSuccess"` // total number of policies of level OVERRIDE, NOT_IN_EFFECT that passed

	BlockingSuccessCount    int `json:"blockingSuccessCount"`
	BlockingFailedCount     int `json:"blockingFailedCount"`
	WarningSuccessCount     int `json:"warningSuccessCount"`
	WarningFailedCount      int `json:"warningFailedCount"`
	RecommendSuccessCount   int `json:"recommendSuccessCount"`
	RecommendFailedCount    int `json:"recommendFailedCount"`
	OverriddenSuccessCount  int `json:"overriddenSuccessCount"`
	OverriddenFailedCount   int `json:"overriddenFailedCount"`
	NotInEffectSuccessCount int `json:"notInEffectSuccessCount"`
	NotInEffectFailedCount  int `json:"notInEffectFailedCount"`
}

// PolicyMatrix represents the detailed policy evaluation matrix
type PolicyMatrix struct {
	// Policies grouped by enforcement level
	BlockingPolicies    []PolicyResult `json:"blockingPolicies"`
	WarningPolicies     []PolicyResult `json:"warningPolicies"`
	RecommendPolicies   []PolicyResult `json:"recommendPolicies"`
	OverriddenPolicies  []PolicyResult `json:"overriddenPolicies"`
	NotInEffectPolicies []PolicyResult `json:"notInEffectPolicies"`
}

// PolicyResult represents the result of a single policy evaluation
type PolicyResult struct {
	PolicyId     string   `json:"policyId"`
	PolicyName   string   `json:"policyName"`
	IsPassing    bool     `json:"isPassing"` // true or false, if false it means FailMessages is not empty
	FailMessages []string `json:"failMessages"`
}

// ReportTemplateData represents the data structure for template rendering
type ReportTemplateData struct {
	ReportData
	RenderedMarkdown string `json:"renderedMarkdown"`
}
