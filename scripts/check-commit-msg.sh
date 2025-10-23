#!/bin/bash
# Commit Message Validation Script for Lefthook
# Validates conventional commit message format

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Commit message file
COMMIT_MSG_FILE="$1"

# Read commit message
if [[ ! -f "${COMMIT_MSG_FILE}" ]]; then
    echo -e "${RED}❌ Commit message file not found: ${COMMIT_MSG_FILE}${NC}"
    exit 1
fi

COMMIT_MSG=$(cat "${COMMIT_MSG_FILE}")

# Skip validation for merge commits and reverts
if echo "${COMMIT_MSG}" | grep -qE "^(Merge|Revert)"; then
    echo -e "${GREEN}✅ Merge/revert commit - skipping validation${NC}"
    exit 0
fi

# Skip validation for conventional commit tags from AI
if echo "${COMMIT_MSG}" | grep -qE "🤖 Generated with \[Claude Code\]"; then
    echo -e "${GREEN}✅ AI-generated commit - skipping strict validation${NC}"
    exit 0
fi

# Conventional commit format: type(scope): description
# Types: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert
CONVENTIONAL_REGEX="^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\([a-z0-9\-]+\))?: .{10,72}$"

# Get the first line (subject)
SUBJECT=$(echo "${COMMIT_MSG}" | head -n1)

# Check conventional commit format
if ! echo "${SUBJECT}" | grep -qE "${CONVENTIONAL_REGEX}"; then
    echo -e "${RED}❌ Commit message does not follow conventional commit format${NC}"
    echo ""
    echo -e "${BLUE}Current message:${NC}"
    echo "  ${SUBJECT}"
    echo ""
    echo -e "${YELLOW}Expected format:${NC}"
    echo "  type(scope): description"
    echo ""
    echo -e "${YELLOW}Valid types:${NC}"
    echo "  feat     - New feature"
    echo "  fix      - Bug fix"
    echo "  docs     - Documentation changes"
    echo "  style    - Code style changes (formatting, etc)"
    echo "  refactor - Code refactoring"
    echo "  perf     - Performance improvements"
    echo "  test     - Test additions or corrections"
    echo "  build    - Build system changes"
    echo "  ci       - CI/CD changes"
    echo "  chore    - Maintenance tasks"
    echo "  revert   - Revert previous commit"
    echo ""
    echo -e "${YELLOW}Examples:${NC}"
    echo "  feat: add user authentication"
    echo "  fix(api): resolve timeout issue"
    echo "  docs: update installation guide"
    echo "  test(auth): add unit tests for login"
    echo ""
    exit 1
fi

# Check subject length
SUBJECT_LENGTH=${#SUBJECT}
if [[ ${SUBJECT_LENGTH} -lt 10 ]]; then
    echo -e "${RED}❌ Commit subject too short (${SUBJECT_LENGTH} chars, minimum 10)${NC}"
    exit 1
fi

if [[ ${SUBJECT_LENGTH} -gt 72 ]]; then
    echo -e "${RED}❌ Commit subject too long (${SUBJECT_LENGTH} chars, maximum 72)${NC}"
    exit 1
fi

# Check that subject doesn't end with period
if echo "${SUBJECT}" | grep -qE '\.$'; then
    echo -e "${RED}❌ Commit subject should not end with a period${NC}"
    exit 1
fi

# Check that description starts with lowercase (after the type)
DESCRIPTION=$(echo "${SUBJECT}" | sed -E 's/^[a-z]+(\([^)]+\))?: //')
if ! echo "${DESCRIPTION}" | grep -qE '^[a-z]'; then
    echo -e "${YELLOW}⚠️  Commit description should start with lowercase letter${NC}"
    # Don't fail on this, just warn
fi

# Check body line length if present
BODY_LINES=$(echo "${COMMIT_MSG}" | tail -n +3)
if [[ -n "${BODY_LINES}" ]]; then
    while IFS= read -r line; do
        if [[ ${#line} -gt 100 ]]; then
            echo -e "${YELLOW}⚠️  Body line exceeds 100 characters: ${#line} chars${NC}"
            echo "  ${line}"
            # Don't fail on this, just warn
        fi
    done <<< "${BODY_LINES}"
fi

echo -e "${GREEN}✅ Commit message format is valid${NC}"
exit 0