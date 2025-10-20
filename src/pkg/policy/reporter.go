package policy

import (
	"github.com/gh-nvat/gitops-kustomz/src/pkg/config"
)

// Reporter generates policy evaluation reports
type Reporter struct{}

// NewReporter creates a new policy reporter
func NewReporter() *Reporter {
	return &Reporter{}
}

// GenerateReport generates a policy report from evaluation results
func (r *Reporter) GenerateReport(result *config.EvaluationResult) *config.PolicyReportData {
	report := &config.PolicyReportData{
		TotalPolicies:   result.TotalPolicies,
		PassedPolicies:  result.PassedPolicies,
		FailedPolicies:  result.FailedPolicies,
		ErroredPolicies: result.ErroredPolicies,
		Details:         make([]config.PolicyDetail, 0, len(result.PolicyResults)),
	}

	// Count failures by level
	for _, pr := range result.PolicyResults {
		if pr.Status == "FAIL" && !pr.Overridden {
			switch pr.Level {
			case "BLOCK":
				report.BlockingFailures++
			case "WARNING":
				report.WarningFailures++
			case "RECOMMEND":
				report.RecommendFailures++
			}
		}

		// Add detail
		detail := config.PolicyDetail{
			Name:        pr.PolicyName,
			Description: "", // Can be populated from config if needed
			Status:      pr.Status,
			Level:       pr.Level,
			Overridden:  pr.Overridden,
			Error:       pr.Error,
			Violations:  make([]string, 0, len(pr.Violations)),
		}

		for _, v := range pr.Violations {
			detail.Violations = append(detail.Violations, v.Message)
		}

		report.Details = append(report.Details, detail)
	}

	return report
}
