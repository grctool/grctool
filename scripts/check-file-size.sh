#!/bin/bash
# File Size Check Script for Lefthook
# Checks that files don't exceed maximum size limits

set -euo pipefail

# Configuration from .pre-commit-config.yaml
MAX_SIZE=1048576  # 1MB default

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Files to check
FILES_TO_CHECK=("$@")

# Exit if no files provided
if [[ ${#FILES_TO_CHECK[@]} -eq 0 ]]; then
    echo -e "${GREEN}✅ No files to check${NC}"
    exit 0
fi

# Track errors
ERRORS=0

# Check each file
for file in "${FILES_TO_CHECK[@]}"; do
    if [[ -f "${file}" ]]; then
        # Get file size (cross-platform)
        if command -v stat >/dev/null 2>&1; then
            # Try macOS format first, then Linux format
            size=$(stat -f%z "${file}" 2>/dev/null || stat -c%s "${file}" 2>/dev/null || echo 0)
        else
            # Fallback to wc if stat not available
            size=$(wc -c < "${file}" 2>/dev/null || echo 0)
        fi

        if [[ ${size} -gt ${MAX_SIZE} ]]; then
            # Check if file is in exclusion list
            if ! echo "${file}" | grep -qE '\.(pdf|png|jpg|gif|svg|tar\.gz|zip)$|^vendor/|^testdata/'; then
                # Format size for human reading
                if command -v numfmt >/dev/null 2>&1; then
                    human_size=$(numfmt --to=iec ${size} 2>/dev/null || echo "${size} bytes")
                else
                    human_size="${size} bytes"
                fi

                echo -e "${RED}❌ File ${file} exceeds maximum size (${human_size})${NC}"
                echo -e "${YELLOW}💡 Maximum allowed size: $(numfmt --to=iec ${MAX_SIZE} 2>/dev/null || echo "1MB")${NC}"
                ((ERRORS++))
            fi
        fi
    fi
done

# Summary
if [[ ${ERRORS} -eq 0 ]]; then
    echo -e "${GREEN}✅ All files are within size limits${NC}"
    exit 0
else
    echo -e "${RED}❌ ${ERRORS} file(s) exceed size limits${NC}"
    echo -e "${YELLOW}💡 Compress large files or add them to exclusion list${NC}"
    exit 1
fi