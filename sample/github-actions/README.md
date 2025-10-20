# GitHub Actions Workflows for gitops-kustomz

This directory contains sample GitHub Actions workflows that you can copy to your GitOps repository.

## Workflows

### 1. `gitops-policy-check.yml` - Per-Environment Checks

**Use this when:** You want separate checks for each environment (sandbox, stg, prod).

**Features:**
- Automatically detects changed services and environments
- Runs parallel checks for each service-environment combination
- Posts separate comments for each combination
- Matrix strategy for efficient parallel execution

**Example output:** 
- PR comment for `my-app` in `stg`
- PR comment for `my-app` in `prod`
- PR comment for `other-service` in `sandbox`

### 2. `gitops-policy-check-multi-env.yml` - Combined Environment Checks

**Use this when:** You want a single combined report for all environments.

**Features:**
- Detects changed services
- Checks all environments (sandbox, stg, prod) in one run
- Posts a single combined comment per service with environment matrix
- Faster for repos with many environments

**Example output:**
- One PR comment for `my-app` showing results for all environments in a matrix

## Setup Instructions

### 1. Copy Workflow to Your Repo

Choose one of the workflows above and copy it to your repository:

```bash
# In your GitOps repository
mkdir -p .github/workflows
cp gitops-policy-check.yml .github/workflows/
# OR
cp gitops-policy-check-multi-env.yml .github/workflows/
```

### 2. Adjust Paths

Edit the workflow file to match your repository structure:

```yaml
on:
  pull_request:
    paths:
      - 'services/**'  # Change if your manifests are elsewhere
      - 'policies/**'  # Change if your policies are elsewhere
```

### 3. Configure Environments

If you use different environment names, update the `--environments` flag:

```yaml
# For per-environment workflow:
# The detection script looks for: services/<service>/environments/<env>/
# Adjust the sed pattern if your structure is different

# For multi-env workflow:
--environments sandbox,stg,prod  # Change to your environment names
```

### 4. Set Up Policies

Ensure your repository has:

```
your-repo/
├── .github/
│   └── workflows/
│       └── gitops-policy-check.yml
├── services/
│   └── my-app/
│       ├── base/
│       │   └── kustomization.yaml
│       └── environments/
│           ├── sandbox/
│           │   └── kustomization.yaml
│           ├── stg/
│           │   └── kustomization.yaml
│           └── prod/
│               └── kustomization.yaml
└── policies/
    ├── compliance-config.yaml
    ├── ha.opa
    ├── ha_test.opa
    └── ingress-tls.opa
```

### 5. (Optional) Custom Templates

If you want custom comment templates:

```bash
# Copy default templates
mkdir -p templates
cp <path-to-gitops-kustomz>/src/templates/*.tmpl templates/

# Edit templates as needed
vim templates/comment.md.tmpl

# Workflow will automatically use them if they exist
```

## Repository Structure Requirements

The tool expects this structure:

```
services/
└── <service-name>/
    ├── base/
    │   └── kustomization.yaml
    └── environments/
        ├── sandbox/
        │   └── kustomization.yaml
        ├── stg/
        │   └── kustomization.yaml
        └── prod/
            └── kustomization.yaml
```

If your structure is different, you may need to:
1. Adjust the change detection script
2. Modify the `--lc-before` and `--lc-after` paths (for local mode)

## Permissions

The workflow requires these permissions:

```yaml
permissions:
  contents: read        # To checkout code
  pull-requests: write  # To post comments
```

These are automatically provided by `secrets.GITHUB_TOKEN`.

## Advanced Configuration

### Use Specific Version

Instead of `@latest`, pin to a specific version:

```yaml
- name: Install gitops-kustomz
  run: go install github.com/gh-nvat/gitops-kustomz@v1.0.0
```

### Use Pre-built Binary

For faster CI runs, use pre-built binaries:

```yaml
- name: Install gitops-kustomz
  run: |
    VERSION="v1.0.0"
    curl -L -o gitops-kustomz \
      "https://github.com/gh-nvat/gitops-kustomz/releases/download/${VERSION}/gitops-kustomz-linux-amd64"
    chmod +x gitops-kustomz
    sudo mv gitops-kustomz /usr/local/bin/
```

### Custom Templates Path

```yaml
- name: Run policy check
  run: |
    gitops-kustomz \
      --run-mode github \
      --gh-repo ${{ github.repository }} \
      --gh-pr-number ${{ github.event.pull_request.number }} \
      --service ${{ matrix.service }} \
      --environments ${{ matrix.environment }} \
      --policies-path ./policies \
      --templates-path ./custom-templates  # Custom templates
```

### Skip Certain Services

```yaml
- name: Detect changed services
  id: set-services
  run: |
    SERVICES=$(echo "$CHANGED_FILES" | \
      grep -E '^services/[^/]+/' | \
      sed -E 's|^services/([^/]+)/.*|\1|' | \
      grep -v -E '^(test-service|deprecated-app)$' | \  # Skip these
      sort -u | \
      jq -R -s -c 'split("\n") | map(select(length > 0))')
```

## Troubleshooting

### No comments appearing

1. Check workflow permissions in repo settings
2. Verify `GH_TOKEN` is set correctly
3. Check workflow logs for errors

### Tool not found

Make sure Go is installed and `$GOPATH/bin` is in PATH:

```yaml
- name: Install gitops-kustomz
  run: |
    go install github.com/gh-nvat/gitops-kustomz@latest
    echo "$HOME/go/bin" >> $GITHUB_PATH
```

### Kustomize build fails

Ensure your kustomization files are valid:

```bash
# Test locally
kustomize build services/my-app/environments/stg
```

### Policy evaluation errors

Check policy syntax:

```bash
# Test policies locally
opa test policies/*.opa
```

## Examples

See the main repository for complete examples:
- [sample/policies/](../../policies/) - Example OPA policies
- [sample/k8s-manifests/](../../k8s-manifests/) - Example Kubernetes manifests
- [test/local/](../../../test/local/) - Local testing examples

## Support

For issues or questions:
- GitHub Issues: https://github.com/gh-nvat/gitops-kustomz/issues
- Documentation: [docs/DESIGN.md](../../../docs/DESIGN.md)

