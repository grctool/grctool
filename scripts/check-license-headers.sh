#!/bin/bash

# Script to verify all Go files have Apache 2.0 license headers
# Usage: ./check-license-headers.sh
# Exit code: 0 if all files have headers, 1 otherwise

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "Checking license headers in Go files..."
echo "Project root: $PROJECT_ROOT"
echo "========================================"
echo ""

# Use grep to find files missing the SPDX header
# -L = files without match, -r = recursive, --include = pattern
missing_files=$(find "$PROJECT_ROOT" -name "*.go" -type f -exec grep -L "SPDX-License-Identifier: Apache-2.0" {} \;)
total_files=$(find "$PROJECT_ROOT" -name "*.go" -type f | wc -l)
missing_count=$(echo "$missing_files" | grep -c "\.go$" || echo "0")

if [ -n "$missing_files" ]; then
    echo "Files missing license headers:"
    echo "$missing_files"
else
    missing_count=0
fi

echo ""
echo "========================================"
echo "Total files checked: $total_files"
echo "Files missing headers: $missing_count"

if [ $missing_count -eq 0 ]; then
    echo "✓ All Go files have license headers"
    exit 0
else
    echo "✗ Some files are missing license headers"
    exit 1
fi
