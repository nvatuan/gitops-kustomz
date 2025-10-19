# gitops-kustomz

GitOps policy enforcement tool for Kubernetes manifests managed with Kustomize.

## Overview

`gitops-kustomz` is designed to run in GitHub Actions CI on Pull Requests. It analyzes Kubernetes manifest changes managed with Kustomize, enforces OPA policies, and provides detailed feedback via PR comments.

## Features

- 🔍 **Kustomize Build & Diff**: Builds manifests from base and head branches, generates clear diffs
- 📋 **Policy Enforcement**: Evaluates OPA policies with configurable enforcement levels (RECOMMEND/WARNING/BLOCK)
- 💬 **GitHub Integration**: Posts detailed policy reports and diffs as PR comments
- ⚡ **Fast**: Parallel policy evaluation with goroutines, <2s build time target
- 🧪 **Local Testing**: Test policies locally without GitHub PR

## Quick Start

```bash
# Run on a PR
gitops-kustomz \
  --repo owner/repo \
  --pr-number 123 \
  --service my-app \
  --environment stg \
  --manifests-path ./services \
  --policies-path ./policies

# Local testing
gitops-kustomz --local-mode \
  --base-manifest ./base.yaml \
  --head-manifest ./head.yaml \
  --policies-path ./policies \
  --local-output-dir ./output
```

## Documentation

- [ARCHITECTURE.md](./ARCHITECTURE.md) - High-level architecture and use cases
- [DESIGN.md](./DESIGN.md) - Detailed design and implementation specs

## Requirements

- Go 1.22+
- `kustomize` binary in PATH
- GitHub token with PR comment permissions (for CI mode)

## Installation

```bash
go install github.com/gh-nvat/gitops-kustomz@latest
```

## Development

```bash
# Clone the repo
git clone https://github.com/gh-nvat/gitops-kustomz.git
cd gitops-kustomz

# Build
go build -o gitops-kustomz ./cmd

# Run tests
go test ./...
```

## License

MIT


