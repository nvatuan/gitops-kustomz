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
// If templateDir is provided, all required templates must exist (fail-fast, no fallback)
func (r *Renderer) RenderWithTemplates(templateDir string, data interface{}) (string, error) {
	// Load all template files
	commentPath := filepath.Join(templateDir, "comment.md.tmpl")
	diffPath := filepath.Join(templateDir, "diff.md.tmpl")
	policyPath := filepath.Join(templateDir, "policy.md.tmpl")

	// Check if all templates exist - fail fast if any are missing
	if _, err := os.Stat(commentPath); err != nil {
		return "", fmt.Errorf("comment template not found at %s: %w (hint: use default template by not specifying --templates-path)", commentPath, err)
	}
	if _, err := os.Stat(diffPath); err != nil {
		return "", fmt.Errorf("diff template not found at %s: %w (hint: use default template by not specifying --templates-path)", diffPath, err)
	}
	if _, err := os.Stat(policyPath); err != nil {
		return "", fmt.Errorf("policy template not found at %s: %w (hint: use default template by not specifying --templates-path)", policyPath, err)
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

// GetDefaultCommentTemplate returns the default embedded comment template
func (r *Renderer) GetDefaultCommentTemplate() string {
	return DefaultCommentTemplate
}
