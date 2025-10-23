#!/bin/bash

# Check for token in environment
TOKEN="${GITHUB_TOKEN}"

if [ -z "$TOKEN" ]; then
    echo "❌ GITHUB_TOKEN environment variable not set"
    echo "   Please set it with: export GITHUB_TOKEN='your-token-here'"
    exit 1
fi

echo "Token length: ${#TOKEN}"
echo "First 7 chars: ${TOKEN:0:7}"
echo ""
echo "Testing GitHub API..."
curl -s -H "Authorization: Bearer $TOKEN" https://api.github.com/user | jq -r '.login // .message'
