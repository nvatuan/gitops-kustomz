# Template Variables Reference

This document provides a comprehensive reference for all available variables and functions in gitops-kustomz templates.

## Overview

The tool uses `MultiEnvCommentData` as the main data structure for template rendering. Templates support Go's `text/template` syntax with custom functions.

## Template Files Structure

The tool expects three template files in the templates directory:

- `comment.md.tmpl` - Main comment template
- `diff.md.tmpl` - Diff section template (included in comment)
- `policy.md.tmpl` - Policy section template (included in comment)

All templates receive the same `MultiEnvCommentData` structure as their data context.

## Top-Level Variables

| Variable | Type | Description | Example |
|----------|------|-------------|---------|
| `.Service` | `string` | Service name | `"my-app"` |
| `.Environments` | `[]string` | List of environments | `["stg", "prod"]` |
| `.BaseCommit` | `string` | Base branch commit SHA (short) | `"abc1234"` |
| `.HeadCommit` | `string` | Head branch commit SHA (short) | `"def5678"` |
| `.Timestamp` | `time.Time` | When the check ran | `2025-10-21T00:01:04Z` |
| `.EnvironmentDiffs` | `[]EnvironmentDiff` | Diff data per environment | See Environment Diffs section |
| `.MultiEnvPolicyReport` | `MultiEnvPolicyReport` | Policy results across environments | See Policy Report section |

## Environment Diffs (`.EnvironmentDiffs`)

For each environment diff:

| Variable | Type | Description | Example |
|----------|------|-------------|---------|
| `.Environment` | `string` | Environment name | `"stg"` |
| `.HasChanges` | `bool` | Whether any changes detected | `true` |
| `.Content` | `string` | Raw unified diff content | `"--- base\n+++ head\n..."` |
| `.LineCount` | `int` | Number of diff lines | `31` |

## Multi-Environment Policy Report (`.MultiEnvPolicyReport`)

| Variable | Type | Description | Example |
|----------|------|-------------|---------|
| `.Environments` | `[]string` | List of environments | `["stg", "prod"]` |
| `.Policies` | `[]MultiEnvPolicyDetail` | Policy details across environments | See Policy Details section |
| `.Summary` | `map[string]EnvSummary` | Summary per environment | See Environment Summary section |

## Policy Details (`.MultiEnvPolicyReport.Policies`)

For each policy:

| Variable | Type | Description | Example |
|----------|------|-------------|---------|
| `.Name` | `string` | Policy name | `"Service High Availability"` |
| `.Description` | `string` | Policy description | `"Ensures deployments meeting high availability criteria"` |
| `.Level` | `string` | Enforcement level | `"RECOMMEND"`, `"WARNING"`, `"BLOCK"`, `"DISABLED"` |
| `.Results` | `map[string]EnvPolicyResult` | Results per environment | See Environment Policy Results section |

## Environment Policy Results (`.Results[env]`)

For each environment's policy result:

| Variable | Type | Description | Example |
|----------|------|-------------|---------|
| `.Status` | `string` | Policy status | `"PASS"`, `"FAIL"`, `"ERROR"` |
| `.Violations` | `[]string` | List of violation messages | `["Deployment 'my-app' must have at least 2 replicas"]` |
| `.Error` | `string` | Error message if Status == "ERROR" | `"Policy evaluation failed: ..."` |
| `.Overridden` | `bool` | Whether override comment was found | `false` |

## Environment Summary (`.Summary[env]`)

For each environment's summary:

| Variable | Type | Description | Example |
|----------|------|-------------|---------|
| `.TotalPolicies` | `int` | Total number of policies | `2` |
| `.PassedPolicies` | `int` | Number of passed policies | `2` |
| `.FailedPolicies` | `int` | Number of failed policies | `0` |
| `.ErroredPolicies` | `int` | Number of errored policies | `0` |
| `.BlockingFailures` | `int` | Number of blocking failures | `0` |
| `.WarningFailures` | `int` | Number of warning failures | `0` |
| `.RecommendFailures` | `int` | Number of recommend failures | `0` |

## Available Template Functions

| Function | Signature | Description | Example |
|----------|-----------|-------------|---------|
| `gt` | `func(a, b int) bool` | Returns true if a > b | `{{if gt .FailedPolicies 0}}` |

## Template Examples

### Basic Service Information

```go
# 🔍 GitOps Policy Check: {{.Service}}

**Timestamp:** {{.Timestamp.Format "2006-01-02 15:04:05 UTC"}}  
**Base:** `{{.BaseCommit}}` → **Head:** `{{.HeadCommit}}`  
**Environments:** {{range $i, $env := .Environments}}{{if $i}}, {{end}}`{{$env}}`{{end}}
```

### Environment Iteration

```go
{{range .Environments}}
  Environment: {{.}}
{{end}}
```

### Diff Section

```go
{{range .EnvironmentDiffs}}
### {{.Environment}}

{{if .HasChanges}}
**Lines changed:** {{.LineCount}}

<details>
<summary>Click to expand {{.Environment}} diff</summary>

```diff
{{.Content}}
```

</details>
{{else}}
✅ No changes detected.
{{end}}
{{end}}
```

### Policy Matrix

```go
| Policy | Enforcement |{{range .MultiEnvPolicyReport.Environments}} {{.}} |{{end}}
|--------|-------------|{{range .MultiEnvPolicyReport.Environments}}--------|{{end}}
{{range $policy := .MultiEnvPolicyReport.Policies}}| {{$policy.Name}} | {{$policy.Level}} |{{range $env := $.MultiEnvPolicyReport.Environments}}{{$result := index $policy.Results $env}} {{if $result}}{{if eq $result.Status "PASS"}}✅{{else if eq $result.Status "FAIL"}}⚠️{{else}}{{$result.Status}}{{end}}{{else}}N/A{{end}} |{{end}}
{{end}}
```

### Summary Per Environment

```go
{{range $env, $sum := .MultiEnvPolicyReport.Summary}}
**{{$env}}:** {{$sum.PassedPolicies}}/{{$sum.TotalPolicies}} passed{{if gt $sum.FailedPolicies 0}} | ❌ {{$sum.FailedPolicies}} failed{{end}}{{if gt $sum.ErroredPolicies 0}} | 💥 {{$sum.ErroredPolicies}} errored{{end}}  
{{end}}
```

### Failed Policies Details

```go
{{range $env, $sum := .MultiEnvPolicyReport.Summary}}
{{if gt $sum.FailedPolicies 0}}
### ⚠️ Failed Policies in {{$env}}

{{range $.MultiEnvPolicyReport.Policies}}
{{$result := index .Results $env}}
{{if and $result (ne $result.Status "PASS")}}
#### {{.Name}}
- **Enforcement:** {{.Level}}
{{if $result.Violations}}- **Violations:**{{range $result.Violations}}
  - {{.}}{{end}}{{end}}
{{if $result.Error}}- **Error:** {{$result.Error}}{{end}}
{{end}}
{{end}}
{{end}}
{{end}}
```

### Conditional Rendering

```go
{{if gt .MultiEnvPolicyReport.Summary.stg.FailedPolicies 0}}
  There are failed policies in staging
{{end}}

{{if .EnvironmentDiffs}}
  Changes detected in manifests
{{else}}
  No changes detected
{{end}}
```

## Template Development Tips

1. **Use range for iteration**: Always use `{{range}}` to iterate over slices
2. **Check for nil values**: Use `{{if $result}}` before accessing nested data
3. **Format time properly**: Use `.Format "2006-01-02 15:04:05 UTC"` for timestamps
4. **Use conditional rendering**: Leverage `{{if}}` statements for dynamic content
5. **Escape special characters**: Use backticks for inline code: `` `{{.Service}}` ``
6. **Test with local mode**: Use `make run-local` to test template changes

## Default Templates

The tool includes default embedded templates that can be used as reference:

- **Comment Template**: `src/templates/comment.md.tmpl`
- **Diff Template**: `src/templates/diff.md.tmpl`  
- **Policy Template**: `src/templates/policy.md.tmpl`

These templates demonstrate proper usage of all available variables and functions.
