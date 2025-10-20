package config

import "time"

// ComplianceConfig represents the complete compliance configuration
type ComplianceConfig struct {
	Policies map[string]PolicyConfig `yaml:"policies"`
}

// PolicyConfig represents a single policy configuration
type PolicyConfig struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Type        string            `yaml:"type"` // "opa" only for now
	FilePath    string            `yaml:"filePath"`
	Enforcement EnforcementConfig `yaml:"enforcement"`
}

// EnforcementConfig defines when and how a policy should be enforced
type EnforcementConfig struct {
	InEffectAfter   *time.Time     `yaml:"inEffectAfter,omitempty"`
	IsWarningAfter  *time.Time     `yaml:"isWarningAfter,omitempty"`
	IsBlockingAfter *time.Time     `yaml:"isBlockingAfter,omitempty"`
	Override        OverrideConfig `yaml:"override"`
}

// OverrideConfig defines how a policy can be overridden
type OverrideConfig struct {
	Comment string `yaml:"comment"` // e.g., "/sp-override-ha"
}

// EvaluationResult represents the result of policy evaluation
type EvaluationResult struct {
	TotalPolicies   int
	PassedPolicies  int
	FailedPolicies  int
	ErroredPolicies int
	PolicyResults   []PolicyResult
}

// PolicyResult represents the result of a single policy evaluation
type PolicyResult struct {
	PolicyID   string
	PolicyName string
	Status     string // "PASS", "FAIL", "ERROR"
	Violations []Violation
	Error      string
	Level      string // "RECOMMEND", "WARNING", "BLOCK", "DISABLED"
	Overridden bool
}

// Violation represents a single policy violation
type Violation struct {
	Message  string
	Resource string
}

// EnforcementResult represents the enforcement decision
type EnforcementResult struct {
	ShouldBlock bool
	ShouldWarn  bool
	Summary     string
}

// CommentData represents data for template rendering
type CommentData struct {
	Service      string
	Environment  string // Deprecated: use Environments for multi-env
	BaseCommit   string
	HeadCommit   string
	Diff         DiffData
	PolicyReport PolicyReportData
	Timestamp    time.Time
}

// MultiEnvCommentData represents data for multi-environment template rendering
type MultiEnvCommentData struct {
	Service              string
	Environments         []string
	BaseCommit           string
	HeadCommit           string
	EnvironmentDiffs     []EnvironmentDiff
	MultiEnvPolicyReport MultiEnvPolicyReport
	Timestamp            time.Time
}

// EnvironmentDiff represents diff data for a specific environment
type EnvironmentDiff struct {
	Environment      string
	HasChanges       bool
	Content          string
	LineCount        int
	AddedLineCount   int
	DeletedLineCount int
}

// MultiEnvPolicyReport represents policy results across multiple environments
type MultiEnvPolicyReport struct {
	Environments []string
	Policies     []MultiEnvPolicyDetail
	Summary      map[string]EnvSummary // env -> summary
}

// MultiEnvPolicyDetail represents a policy's results across environments
type MultiEnvPolicyDetail struct {
	Name        string
	Description string
	Level       string
	Results     map[string]EnvPolicyResult // env -> result
}

// EnvPolicyResult represents a policy result for one environment
type EnvPolicyResult struct {
	Status     string // "PASS", "FAIL", "ERROR"
	Violations []string
	Error      string
	Overridden bool
}

// EnvSummary represents a summary for one environment
type EnvSummary struct {
	TotalPolicies     int
	PassedPolicies    int
	FailedPolicies    int
	ErroredPolicies   int
	BlockingFailures  int
	WarningFailures   int
	RecommendFailures int
}

// DiffData represents diff information
type DiffData struct {
	HasChanges bool
	Content    string
	LineCount  int
}

// PolicyReportData represents policy report data for templates
type PolicyReportData struct {
	TotalPolicies     int
	PassedPolicies    int
	FailedPolicies    int
	ErroredPolicies   int
	BlockingFailures  int
	WarningFailures   int
	RecommendFailures int
	Details           []PolicyDetail
}

// PolicyDetail represents a single policy detail for reporting
type PolicyDetail struct {
	Name        string
	Description string
	Status      string
	Level       string
	Overridden  bool
	Error       string
	Violations  []string
}

// PullRequest represents GitHub PR information
type PullRequest struct {
	Number  int
	BaseRef string
	BaseSHA string
	HeadRef string
	HeadSHA string
}

// Comment represents a GitHub comment
type Comment struct {
	ID   int64
	Body string
}
