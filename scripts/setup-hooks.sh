#!/bin/bash

# BreatheRoute Git Hooks Setup Script
# Run this script to install git hooks for the project

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${CYAN}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║  BreatheRoute Git Hooks Setup                              ║${NC}"
echo -e "${CYAN}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Get the root directory of the git repository
REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null)

if [ -z "$REPO_ROOT" ]; then
    echo -e "${RED}Error: Not a git repository${NC}"
    exit 1
fi

HOOKS_SOURCE="$REPO_ROOT/.githooks"
HOOKS_TARGET="$REPO_ROOT/.git/hooks"

# Check if source hooks exist
if [ ! -d "$HOOKS_SOURCE" ]; then
    echo -e "${RED}Error: Hooks source directory not found: $HOOKS_SOURCE${NC}"
    exit 1
fi

echo -e "Installing git hooks from ${CYAN}$HOOKS_SOURCE${NC}"
echo ""

# Method 1: Configure git to use custom hooks directory (Git 2.9+)
GIT_VERSION=$(git --version | awk '{print $3}')
GIT_MAJOR=$(echo "$GIT_VERSION" | cut -d. -f1)
GIT_MINOR=$(echo "$GIT_VERSION" | cut -d. -f2)

if [ "$GIT_MAJOR" -gt 2 ] || ([ "$GIT_MAJOR" -eq 2 ] && [ "$GIT_MINOR" -ge 9 ]); then
    echo -e "Using Git core.hooksPath (Git $GIT_VERSION detected)"
    git config core.hooksPath .githooks
    echo -e "${GREEN}✓ Configured core.hooksPath to use .githooks${NC}"
else
    # Method 2: Symlink hooks for older Git versions
    echo -e "Symlinking hooks (Git $GIT_VERSION detected)"

    for hook in "$HOOKS_SOURCE"/*; do
        if [ -f "$hook" ]; then
            hook_name=$(basename "$hook")
            target="$HOOKS_TARGET/$hook_name"

            # Backup existing hook if it exists and is not a symlink
            if [ -f "$target" ] && [ ! -L "$target" ]; then
                echo -e "${YELLOW}  Backing up existing $hook_name to $hook_name.backup${NC}"
                mv "$target" "$target.backup"
            fi

            # Create symlink
            ln -sf "$hook" "$target"
            echo -e "${GREEN}  ✓ Installed $hook_name${NC}"
        fi
    done
fi

# Make hooks executable
chmod +x "$HOOKS_SOURCE"/*
echo -e "${GREEN}✓ Made hooks executable${NC}"

echo ""
echo -e "${GREEN}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  Git hooks installed successfully!                         ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "Installed hooks:"
for hook in "$HOOKS_SOURCE"/*; do
    if [ -f "$hook" ]; then
        echo -e "  - $(basename "$hook")"
    fi
done
echo ""
echo -e "To skip hooks temporarily, use: ${YELLOW}git commit --no-verify${NC}"
echo ""

# Check for required tools
echo -e "${CYAN}Checking for required tools...${NC}"

check_tool() {
    if command -v "$1" &> /dev/null; then
        echo -e "  ${GREEN}✓${NC} $1 is installed"
        return 0
    else
        echo -e "  ${YELLOW}✗${NC} $1 is not installed"
        return 1
    fi
}

echo ""
echo "Go tools:"
check_tool "go" || true
check_tool "gofmt" || true
check_tool "golangci-lint" || echo -e "    Install: ${CYAN}go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest${NC}"

echo ""
echo "Swift tools:"
check_tool "swiftlint" || echo -e "    Install: ${CYAN}brew install swiftlint${NC}"
check_tool "swiftformat" || echo -e "    Install: ${CYAN}brew install swiftformat${NC}"

echo ""
echo "Terraform tools:"
check_tool "terraform" || echo -e "    Install: ${CYAN}brew install terraform${NC}"
check_tool "tfsec" || echo -e "    Install: ${CYAN}brew install tfsec${NC}"

echo ""
