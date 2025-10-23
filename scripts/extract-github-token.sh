#!/bin/bash

# Extract GitHub token from git credentials store
# This reads from ~/.git-credentials file

if [ -f ~/.git-credentials ]; then
    # Extract token from https://username:TOKEN@github.com format
    TOKEN=$(grep 'github.com' ~/.git-credentials | sed -n 's/.*:\/\/[^:]*:\([^@]*\)@.*/\1/p' | head -1)

    if [ -n "$TOKEN" ]; then
        echo "$TOKEN"
        exit 0
    fi
fi

# If not found in credentials file, check environment
if [ -n "$GITHUB_TOKEN" ]; then
    echo "$GITHUB_TOKEN"
    exit 0
fi

if [ -n "$GH_TOKEN" ]; then
    echo "$GH_TOKEN"
    exit 0
fi

echo "ERROR: No GitHub token found" >&2
echo "Checked:" >&2
echo "  - ~/.git-credentials" >&2
echo "  - GITHUB_TOKEN environment variable" >&2
echo "  - GH_TOKEN environment variable" >&2
exit 1
