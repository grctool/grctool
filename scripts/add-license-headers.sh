#!/bin/bash

# Script to add Apache 2.0 license headers to Go files
# Usage: ./add-license-headers.sh

set -e

LICENSE_HEADER="// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the \"License\");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an \"AS IS\" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
"

add_header() {
    local file="$1"

    # Check if file already has SPDX header
    if grep -q "SPDX-License-Identifier: Apache-2.0" "$file"; then
        echo "SKIP: $file (already has license header)"
        return 0
    fi

    # Create temporary file with header + original content
    {
        echo "$LICENSE_HEADER"
        cat "$file"
    } > "$file.tmp"

    # Replace original file
    mv "$file.tmp" "$file"
    echo "ADDED: $file"
}

# Count files
total_files=$(find /pool0/erik/Projects/grctool -name "*.go" -type f | wc -l)
processed=0

echo "Processing $total_files Go files..."
echo "=================================="

# Process all Go files
while IFS= read -r file; do
    add_header "$file"
    ((processed++))

    # Report progress every 20 files
    if (( processed % 20 == 0 )); then
        echo ""
        echo "Progress: $processed / $total_files files processed"
        echo ""
    fi
done < <(find /pool0/erik/Projects/grctool -name "*.go" -type f | sort)

echo ""
echo "=================================="
echo "Complete: $processed files processed"
