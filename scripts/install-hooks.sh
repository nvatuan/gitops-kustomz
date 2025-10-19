#!/bin/sh
# Install git hooks for this repository

HOOK_DIR=".git/hooks"
SCRIPTS_DIR="scripts"

# Check if we're in a git repository
if [ ! -d ".git" ]; then
    echo "Error: Not a git repository. Please run this from the repository root."
    exit 1
fi

# Install pre-commit hook
echo "Installing pre-commit hook..."
cp "$SCRIPTS_DIR/pre-commit.sh" "$HOOK_DIR/pre-commit"
chmod +x "$HOOK_DIR/pre-commit"

echo "Git hooks installed successfully!"
echo ""
echo "To uninstall, run: rm .git/hooks/pre-commit"

