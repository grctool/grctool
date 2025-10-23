#!/usr/bin/env python3
"""Add Apache 2.0 license headers to Go files."""

import os
import sys
from pathlib import Path

LICENSE_HEADER = """// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

"""

def has_license_header(content):
    """Check if file already has SPDX license header."""
    return "SPDX-License-Identifier: Apache-2.0" in content[:500]

def add_header_to_file(filepath):
    """Add license header to a single Go file."""
    try:
        with open(filepath, 'r', encoding='utf-8') as f:
            content = f.read()

        if has_license_header(content):
            return 'skip'

        new_content = LICENSE_HEADER + content

        with open(filepath, 'w', encoding='utf-8') as f:
            f.write(new_content)

        return 'added'
    except Exception as e:
        print(f"ERROR processing {filepath}: {e}", file=sys.stderr)
        return 'error'

def main():
    root_dir = Path('/pool0/erik/Projects/grctool')
    go_files = sorted(root_dir.rglob('*.go'))

    total = len(go_files)
    added = 0
    skipped = 0
    errors = 0

    print(f"Processing {total} Go files...")
    print("=" * 50)

    for i, filepath in enumerate(go_files, 1):
        result = add_header_to_file(filepath)

        if result == 'added':
            print(f"ADDED: {filepath}")
            added += 1
        elif result == 'skip':
            skipped += 1
        else:
            errors += 1

        # Progress report every 20 files
        if i % 20 == 0:
            print()
            print(f"Progress: {i}/{total} files processed")
            print(f"Added: {added}, Skipped: {skipped}, Errors: {errors}")
            print()

    print()
    print("=" * 50)
    print(f"Complete: {total} files processed")
    print(f"  Added: {added}")
    print(f"  Skipped: {skipped}")
    print(f"  Errors: {errors}")

if __name__ == '__main__':
    main()
