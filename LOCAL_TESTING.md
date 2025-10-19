# Local Testing Guide

## Running gitops-kustomz in Local Mode

Local mode allows you to test the tool without GitHub PR integration.

### Quick Start with Test Data

```bash
# 1. Build manifests from before/after test data
kustomize build test/local/before/services/my-app/environments/stg > /tmp/before.yaml
kustomize build test/local/after/services/my-app/environments/stg > /tmp/after.yaml

# 2. Run the tool in local mode
./gitops-kustomz --local-mode \
  --service my-app \
  --environment stg \
  --base-manifest /tmp/before.yaml \
  --head-manifest /tmp/after.yaml \
  --policies-path sample/policies \
  --local-output-dir test/output

# 3. View the report
cat test/output/my-app-stg-report.md
```

### Testing Different Environments

```bash
# Test prod environment
kustomize build test/local/before/services/my-app/environments/prod > /tmp/before-prod.yaml
kustomize build test/local/after/services/my-app/environments/prod > /tmp/after-prod.yaml

./gitops-kustomz --local-mode \
  --service my-app \
  --environment prod \
  --base-manifest /tmp/before-prod.yaml \
  --head-manifest /tmp/after-prod.yaml \
  --policies-path sample/policies \
  --local-output-dir test/output

# Test sandbox environment
kustomize build test/local/before/services/my-app/environments/sandbox > /tmp/before-sandbox.yaml
kustomize build test/local/after/services/my-app/environments/sandbox > /tmp/after-sandbox.yaml

./gitops-kustomz --local-mode \
  --service my-app \
  --environment sandbox \
  --base-manifest /tmp/before-sandbox.yaml \
  --head-manifest /tmp/after-sandbox.yaml \
  --policies-path sample/policies \
  --local-output-dir test/output
```

### Using Custom Templates

```bash
# Use custom comment template
./gitops-kustomz --local-mode \
  --service my-app \
  --environment stg \
  --base-manifest /tmp/before.yaml \
  --head-manifest /tmp/after.yaml \
  --policies-path sample/policies \
  --comment-template ./templates/comment.md.tmpl \
  --local-output-dir test/output
```

### One-Liner for Quick Testing

```bash
# Build and test in one command
kustomize build test/local/before/services/my-app/environments/stg > /tmp/before.yaml && \
kustomize build test/local/after/services/my-app/environments/stg > /tmp/after.yaml && \
./gitops-kustomz --local-mode \
  --service my-app --environment stg \
  --base-manifest /tmp/before.yaml \
  --head-manifest /tmp/after.yaml \
  --policies-path sample/policies \
  --local-output-dir test/output && \
cat test/output/my-app-stg-report.md
```

## Expected Output

```
📋 Loading policy configuration...
✅ Loaded 2 policies
🏠 Running in local mode
📊 Generating diff...
   10 lines changed
🛡️  Evaluating policies...
   Total: 2, Passed: 2, Failed: 0, Errored: 0
   All checks passed
✅ Report written to: test/output/my-app-stg-report.md
✅ All checks passed!
```

## Interpreting Results

### Exit Codes
- **0**: All checks passed or only RECOMMEND failures
- **1**: WARNING or BLOCKING policy failures
- **2**: Tool error (auth, build, config issues)

### Report Sections
1. **Summary**: Overview of changes and policy results
2. **Manifest Diff**: Line-by-line changes between base and head
3. **Policy Evaluation Results**: Detailed policy check results

## Testing Policy Violations

To test policy failures, modify the after manifests to violate policies:

```bash
# Edit deployment to have only 1 replica (violates HA policy)
vim test/local/after/services/my-app/base/deployment.yaml
# Change replicas: 2 to replicas: 1

# Rebuild and test
kustomize build test/local/after/services/my-app/environments/stg > /tmp/after.yaml
./gitops-kustomz --local-mode --service my-app --environment stg \
  --base-manifest /tmp/before.yaml --head-manifest /tmp/after.yaml \
  --policies-path sample/policies --local-output-dir test/output
```

