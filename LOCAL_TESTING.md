# Local Testing Guide

## Running gitops-kustomz in Local Mode

Local mode allows you to test the tool without GitHub PR integration. The tool will run `kustomize build` internally and supports checking multiple environments in a single run.

### Quick Start with Test Data

```bash
# Check multiple environments at once
./gitops-kustomz --run-mode local \
  --service my-app \
  --environments stg,prod \
  --lc-before-manifests-path test/local/before/services \
  --lc-after-manifests-path test/local/after/services \
  --policies-path sample/policies \
  --lc-output-dir test/output

# Check single environment
./gitops-kustomz --run-mode local \
  --service my-app \
  --environments stg \
  --lc-before-manifests-path test/local/before/services \
  --lc-after-manifests-path test/local/after/services \
  --policies-path sample/policies \
  --lc-output-dir test/output

# View the reports
ls -lh test/output/
cat test/output/my-app-stg-report.md
cat test/output/my-app-prod-report.md
```

### Directory Structure

The tool expects environment-specific directories under the base paths:

```
test/local/
├── before/services/my-app/environments/
│   ├── stg/           # kustomization for staging
│   └── prod/          # kustomization for production
└── after/services/my-app/environments/
    ├── stg/           # kustomization for staging
    └── prod/          # kustomization for production
```

Each environment directory should contain a valid `kustomization.yaml`.

### Using Custom Templates

```bash
# Use custom templates directory
./gitops-kustomz --run-mode local \
  --service my-app \
  --environments stg,prod \
  --lc-before-manifests-path test/local/before/services \
  --lc-after-manifests-path test/local/after/services \
  --policies-path sample/policies \
  --templates-path ./my-custom-templates \
  --lc-output-dir test/output
```

### One-Liner for Quick Testing

```bash
# Test and view reports in one command
./gitops-kustomz --run-mode local \
  --service my-app --environments stg,prod \
  --lc-before-manifests-path test/local/before/services \
  --lc-after-manifests-path test/local/after/services \
  --policies-path sample/policies \
  --lc-output-dir test/output && \
ls -lh test/output/
```

### Using Make

The Makefile includes a convenient target:

```bash
make run-local
```

This will:
1. Build the binary
2. Run checks for `stg` and `prod` environments
3. Display the generated reports

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

