package template

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// Renderer handles template rendering
type Renderer struct {
	funcMap template.FuncMap
}

// NewRenderer creates a new template renderer
func NewRenderer() *Renderer {
	return &Renderer{
		funcMap: template.FuncMap{
			"gt": func(a, b int) bool { return a > b },
		},
	}
}

// RenderWithTemplates renders templates with support for includes
func (r *Renderer) RenderWithTemplates(templateDir string, data interface{}) (string, error) {
	// Load all template files
	commentPath := filepath.Join(templateDir, "comment.md.tmpl")
	diffPath := filepath.Join(templateDir, "diff.md.tmpl")
	policyPath := filepath.Join(templateDir, "policy.md.tmpl")

	// Check if all templates exist
	if _, err := os.Stat(commentPath); err != nil {
		return "", fmt.Errorf("comment template not found: %w", err)
	}
	if _, err := os.Stat(diffPath); err != nil {
		return "", fmt.Errorf("diff template not found: %w", err)
	}
	if _, err := os.Stat(policyPath); err != nil {
		return "", fmt.Errorf("policy template not found: %w", err)
	}

	// Parse all templates with named templates
	tmpl := template.New("").Funcs(r.funcMap)
	
	// Parse diff template as a named template
	diffContent, err := os.ReadFile(diffPath)
	if err != nil {
		return "", fmt.Errorf("failed to read diff template: %w", err)
	}
	if _, err := tmpl.New("diff").Parse(string(diffContent)); err != nil {
		return "", fmt.Errorf("failed to parse diff template: %w", err)
	}

	// Parse policy template as a named template
	policyContent, err := os.ReadFile(policyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read policy template: %w", err)
	}
	if _, err := tmpl.New("policy").Parse(string(policyContent)); err != nil {
		return "", fmt.Errorf("failed to parse policy template: %w", err)
	}

	// Parse main comment template
	commentContent, err := os.ReadFile(commentPath)
	if err != nil {
		return "", fmt.Errorf("failed to read comment template: %w", err)
	}
	mainTmpl, err := tmpl.New("comment").Parse(string(commentContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse comment template: %w", err)
	}

	var buf bytes.Buffer
	if err := mainTmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// Render renders a template file with the provided data
func (r *Renderer) Render(templatePath string, data interface{}) (string, error) {
	// Read template file
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template: %w", err)
	}

	return r.RenderString(string(content), data)
}

// RenderString renders a template string with the provided data
func (r *Renderer) RenderString(templateStr string, data interface{}) (string, error) {
	tmpl, err := template.New("template").Funcs(r.funcMap).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// GetDefaultCommentTemplate returns the default comment template
func (r *Renderer) GetDefaultCommentTemplate() string {
	return `<!-- gitops-kustomz: {{.Service}}-{{.Environment}} -->

# 🔍 GitOps Policy Check: {{.Service}} ({{.Environment}})

**Base Commit:** ` + "`{{.BaseCommit}}`" + `
**Head Commit:** ` + "`{{.HeadCommit}}`" + `
**Timestamp:** {{.Timestamp}}

---

## 📊 Summary

{{if .Diff.HasChanges}}
✏️ **Changes detected in manifests** ({{.Diff.LineCount}} lines changed)
{{else}}
✅ **No changes detected**
{{end}}

{{if .PolicyReport}}
**Policy Evaluation:**
- **Total:** {{.PolicyReport.TotalPolicies}}
- **Passed:** {{.PolicyReport.PassedPolicies}} ✅
- **Failed:** {{.PolicyReport.FailedPolicies}} ❌
{{if gt .PolicyReport.ErroredPolicies 0}}- **Errored:** {{.PolicyReport.ErroredPolicies}} ⚠️{{end}}

{{if gt .PolicyReport.FailedPolicies 0}}**Failures by Level:**
{{if gt .PolicyReport.BlockingFailures 0}}- 🚫 Blocking: {{.PolicyReport.BlockingFailures}}{{end}}
{{if gt .PolicyReport.WarningFailures 0}}- ⚠️  Warning: {{.PolicyReport.WarningFailures}}{{end}}
{{if gt .PolicyReport.RecommendFailures 0}}- 💡 Recommend: {{.PolicyReport.RecommendFailures}}{{end}}
{{end}}
{{end}}

---

## 📝 Manifest Diff

{{if .Diff.HasChanges}}
{{if gt .Diff.LineCount 50}}
<details>
<summary>Click to expand diff ({{.Diff.LineCount}} lines)</summary>

` + "```diff" + `
{{.Diff.Content}}
` + "```" + `

</details>
{{else}}
` + "```diff" + `
{{.Diff.Content}}
` + "```" + `
{{end}}
{{else}}
_No changes detected between base and head._
{{end}}

---

## 🛡️ Policy Evaluation Results

{{range .PolicyReport.Details}}
### {{.Name}} - {{.Status}} {{if .Overridden}}(Overridden){{end}}

**Level:** {{.Level}}

{{if eq .Status "ERROR"}}
⚠️ **Error:** {{.Error}}
{{else if eq .Status "FAIL"}}
{{if .Overridden}}
✋ **Policy failed but was overridden**
{{else}}
❌ **Policy violations:**
{{range .Violations}}
- {{.}}
{{end}}
{{end}}
{{else}}
✅ **Passed**
{{end}}

---
{{end}}

---

_Generated by [gitops-kustomz](https://github.com/gh-nvat/gitops-kustomz)_
`
}
