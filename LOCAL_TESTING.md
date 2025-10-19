# Local Testing Guide

## Running gitops-kustomz in Local Mode

Local mode allows you to test the tool without GitHub PR integration. The tool will run `kustomize build` internally.

### Quick Start with Test Data

```bash
# Run directly on kustomize directories (recommended)
./gitops-kustomz --local-mode \
  --service my-app \
  --environment stg \
  --base-manifest test/local/before/services/my-app/environments/stg \
  --head-manifest test/local/after/services/my-app/environments/stg \
  --policies-path sample/policies \
  --local-output-dir test/output

# View the report
cat test/output/my-app-stg-report.md
```

### Alternative: Using Pre-built Manifests

If you prefer to build manifests separately:

```bash
# 1. Build manifests manually
kustomize build test/local/before/services/my-app/environments/stg > /tmp/before.yaml
kustomize build test/local/after/services/my-app/environments/stg > /tmp/after.yaml

# 2. Run the tool with pre-built manifests
./gitops-kustomz --local-mode \
  --service my-app \
  --environment stg \
  --base-manifest /tmp/before.yaml \
  --head-manifest /tmp/after.yaml \
  --policies-path sample/policies \
  --local-output-dir test/output
```

### Testing Different Environments

```bash
# Test prod environment
./gitops-kustomz --local-mode \
  --service my-app \
  --environment prod \
  --base-manifest test/local/before/services/my-app/environments/prod \
  --head-manifest test/local/after/services/my-app/environments/prod \
  --policies-path sample/policies \
  --local-output-dir test/output

# Test sandbox environment
./gitops-kustomz --local-mode \
  --service my-app \
  --environment sandbox \
  --base-manifest test/local/before/services/my-app/environments/sandbox \
  --head-manifest test/local/after/services/my-app/environments/sandbox \
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
# Test and view report in one command
./gitops-kustomz --local-mode \
  --service my-app --environment stg \
  --base-manifest test/local/before/services/my-app/environments/stg \
  --head-manifest test/local/after/services/my-app/environments/stg \
  --policies-path sample/policies \
  --local-output-dir test/output && \
cat test/output/my-app-stg-report.md
```

## Expected Output

```
📋 Loading policy configuration...
✅ Loaded 2 policies
🏠 Running in local mode
🔨 Building base manifest from kustomize directory...
🔨 Building head manifest from kustomize directory...
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

