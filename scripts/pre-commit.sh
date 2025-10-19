#!/bin/sh
# Pre-commit hook for Go formatting and linting

# Check if gofmt is installed
if ! command -v gofmt &> /dev/null; then
    echo "gofmt not found. Please install Go."
    exit 1
fi

# Check Go formatting
UNFORMATTED=$(gofmt -l .)
if [ -n "$UNFORMATTED" ]; then
    echo "The following files are not formatted:"
    echo "$UNFORMATTED"
    echo ""
    echo "Run 'gofmt -w .' to fix formatting"
    exit 1
fi

# Check if golangci-lint is installed
if ! command -v golangci-lint &> /dev/null; then
    echo "Warning: golangci-lint not found. Skipping lint checks."
    echo "Install it from: https://golangci-lint.run/usage/install/"
    exit 0
fi

# Run golangci-lint on staged files
STAGED_GO_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$')

if [ -n "$STAGED_GO_FILES" ]; then
    echo "Running golangci-lint on staged files..."
    echo "$STAGED_GO_FILES" | xargs golangci-lint run --timeout=5m
    if [ $? -ne 0 ]; then
        echo "golangci-lint found issues. Please fix them before committing."
        exit 1
    fi
fi

exit 0

