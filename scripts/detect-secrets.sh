#!/bin/bash
# Secret Detection Script for GRCTool
# Scans files for potential secrets, API keys, and sensitive information

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Files to scan (passed as arguments or read from stdin)
FILES_TO_SCAN=("$@")
if [[ ${#FILES_TO_SCAN[@]} -eq 0 ]]; then
    # Read from stdin if no arguments
    while IFS= read -r file; do
        FILES_TO_SCAN+=("${file}")
    done
fi

# Exit codes
EXIT_SUCCESS=0
EXIT_SECRETS_FOUND=1

# Counter for found secrets
SECRETS_FOUND=0
FILES_WITH_SECRETS=()

# Patterns for detecting secrets (regex)
declare -a SECRET_PATTERNS=(
    # API Keys and Tokens
    'api[_-]?key[\s]*[:=][\s]*["\x27]?[a-zA-Z0-9_\-]{32,}["\x27]?'
    'apikey[\s]*[:=][\s]*["\x27]?[a-zA-Z0-9_\-]{32,}["\x27]?'
    'access[_-]?token[\s]*[:=][\s]*["\x27]?[a-zA-Z0-9_\-]{32,}["\x27]?'
    'auth[_-]?token[\s]*[:=][\s]*["\x27]?[a-zA-Z0-9_\-]{32,}["\x27]?'
    'bearer[\s]+[a-zA-Z0-9_\-\.]{20,}'
    
    # AWS
    'AKIA[0-9A-Z]{16}'
    'aws[_-]?access[_-]?key[_-]?id[\s]*[:=][\s]*["\x27]?[A-Z0-9]{20}["\x27]?'
    'aws[_-]?secret[_-]?access[_-]?key[\s]*[:=][\s]*["\x27]?[a-zA-Z0-9/\+=]{40}["\x27]?'
    
    # Google Cloud
    'AIza[0-9A-Za-z_\-]{35}'
    '(gcp|google)[_-]?api[_-]?key[\s]*[:=]'
    
    # GitHub
    'ghp_[a-zA-Z0-9]{36}'
    'gho_[a-zA-Z0-9]{36}'
    'ghu_[a-zA-Z0-9]{36}'
    'ghs_[a-zA-Z0-9]{36}'
    'ghr_[a-zA-Z0-9]{36}'
    
    # Private Keys
    '-----BEGIN[\s]+(RSA|DSA|EC|OPENSSH|PGP)[\s]+PRIVATE[\s]+KEY'
    '-----BEGIN[\s]+PRIVATE[\s]+KEY-----'
    '-----BEGIN[\s]+ENCRYPTED[\s]+PRIVATE[\s]+KEY-----'
    
    # Passwords (be careful with false positives)
    'password[\s]*[:=][\s]*["\x27][^"\x27]{8,}["\x27]'
    'passwd[\s]*[:=][\s]*["\x27][^"\x27]{8,}["\x27]'
    'pwd[\s]*[:=][\s]*["\x27][^"\x27]{8,}["\x27]'
    
    # Database Connection Strings
    'postgres://[^:]+:[^@]+@[^/\s]+/[^\s]+'
    'mysql://[^:]+:[^@]+@[^/\s]+/[^\s]+'
    'mongodb(\+srv)?://[^:]+:[^@]+@[^/\s]+/[^\s]+'
    'redis://[^:]*:[^@]+@[^/\s]+'
    
    # Slack
    'xox[baprs]-[0-9a-zA-Z]{10,48}'
    
    # Stripe
    '(sk|pk)_(test|live)_[0-9a-zA-Z]{24,}'
    
    # Square
    'sq0[a-z]{3}-[0-9A-Za-z\-_]{22,43}'
    
    # Twilio
    'SK[0-9a-fA-F]{32}'
    
    # Generic Secrets
    'secret[\s]*[:=][\s]*["\x27]?[a-zA-Z0-9_\-]{16,}["\x27]?'
    'client[_-]?secret[\s]*[:=][\s]*["\x27]?[a-zA-Z0-9_\-]{16,}["\x27]?'
)

# Allowed patterns (false positives to ignore)
declare -a ALLOWED_PATTERNS=(
    # Test/Example credentials
    'password[\s]*[:=][\s]*["\x27]?(test|example|sample|dummy|mock|placeholder|changeme|default)["\x27]?'
    'ExamplePassword'
    'TestToken'
    'test[_-]?api[_-]?key'
    'example[_-]?api[_-]?key'
    'sample[_-]?api[_-]?key'
    'YOUR[_-]API[_-]KEY'
    'your-api-key-here'
    '<.*api[_-]?key.*>'
    '\$\{.*\}'  # Environment variable placeholders
    '{{.*}}'    # Template placeholders
)

# File extensions to skip
declare -a SKIP_EXTENSIONS=(
    ".md"
    ".txt"
    ".sum"
    ".mod"
    ".lock"
    ".pdf"
    ".png"
    ".jpg"
    ".jpeg"
    ".gif"
    ".svg"
    ".ico"
)

# Directories to skip
declare -a SKIP_DIRS=(
    "vendor"
    "node_modules"
    ".git"
    "testdata"
    "test/fixtures"
)

# Function to check if file should be skipped
should_skip_file() {
    local file="$1"
    
    # Check if file exists
    if [[ ! -f "${file}" ]]; then
        return 0
    fi
    
    # Check extensions
    for ext in "${SKIP_EXTENSIONS[@]}"; do
        if [[ "${file}" == *"${ext}" ]]; then
            return 0
        fi
    done
    
    # Check directories
    for dir in "${SKIP_DIRS[@]}"; do
        if echo "${file}" | grep -q "/${dir}/"; then
            return 0
        fi
    done
    
    # Check if file is binary
    if file "${file}" | grep -q "binary\|data\|executable"; then
        return 0
    fi
    
    return 1
}

# Function to check if a match is allowed
is_allowed_match() {
    local line="$1"
    
    for pattern in "${ALLOWED_PATTERNS[@]}"; do
        if echo "${line}" | grep -qiE "${pattern}"; then
            return 0
        fi
    done
    
    return 1
}

# Function to scan a single file
scan_file() {
    local file="$1"
    local found_in_file=0
    
    # Skip if file should be ignored
    if should_skip_file "${file}"; then
        return 0
    fi
    
    # Scan for each pattern
    for pattern in "${SECRET_PATTERNS[@]}"; do
        # Use grep with line numbers
        matches=$(grep -nE "${pattern}" "${file}" 2>/dev/null || true)
        
        if [[ -n "${matches}" ]]; then
            while IFS= read -r match; do
                line_num=$(echo "${match}" | cut -d: -f1)
                line_content=$(echo "${match}" | cut -d: -f2-)
                
                # Check if this is an allowed pattern
                if ! is_allowed_match "${line_content}"; then
                    if [[ ${found_in_file} -eq 0 ]]; then
                        echo -e "${RED}Potential secrets found in ${file}:${NC}"
                        found_in_file=1
                    fi
                    echo -e "  Line ${line_num}: ${YELLOW}${line_content:0:100}${NC}"
                    ((SECRETS_FOUND++))
                fi
            done <<< "${matches}"
        fi
    done
    
    if [[ ${found_in_file} -gt 0 ]]; then
        FILES_WITH_SECRETS+=("${file}")
        echo ""
    fi
    
    return ${found_in_file}
}

# Main scanning logic
main() {
    echo "Scanning ${#FILES_TO_SCAN[@]} files for potential secrets..."
    echo ""
    
    for file in "${FILES_TO_SCAN[@]}"; do
        scan_file "${file}"
    done
    
    # Summary
    if [[ ${SECRETS_FOUND} -eq 0 ]]; then
        echo -e "${GREEN}✓ No secrets detected${NC}"
        exit ${EXIT_SUCCESS}
    else
        echo "=================================================================================="
        echo -e "${RED}⚠ Found ${SECRETS_FOUND} potential secret(s) in ${#FILES_WITH_SECRETS[@]} file(s)${NC}"
        echo ""
        echo "Files with potential secrets:"
        for file in "${FILES_WITH_SECRETS[@]}"; do
            echo "  - ${file}"
        done
        echo ""
        echo -e "${YELLOW}Please review and remove any actual secrets before committing.${NC}"
        echo "If these are false positives, you can:"
        echo "  1. Add the pattern to ALLOWED_PATTERNS in this script"
        echo "  2. Use environment variables or config files for sensitive data"
        echo "  3. Add the file to .gitignore if it contains secrets"
        echo ""
        echo "To bypass (NOT RECOMMENDED): git commit --no-verify"
        exit ${EXIT_SECRETS_FOUND}
    fi
}

# Run main function
main