#!/bin/bash
set -e

echo "🎬 VCR Cassette Recording Script"
echo "================================="
echo ""

# Extract GitHub token
GITHUB_TOKEN=$(./scripts/extract-github-token.sh)
if [ -z "$GITHUB_TOKEN" ]; then
    echo "❌ Failed to extract GitHub token"
    exit 1
fi
echo "✓ GitHub token extracted"
echo ""

# Export for tests
export GITHUB_TOKEN
export VCR_MODE=record

# Record evidence_task_validation cassette
echo "📼 Recording: evidence_task_validation.yaml"
echo "   Running TestEvidenceTaskValidation..."
cd "$(dirname "$0")/.."
go test -v -tags=integration -timeout=5m -run "^TestEvidenceTaskValidation$" ./test/integration/
echo ""

# Record tools_integration cassette
echo "📼 Recording: tools_integration.yaml"
echo "   Running TestToolsIntegration..."
go test -v -tags=integration -timeout=5m -run "^TestToolsIntegration" ./test/integration/
echo ""

echo "✅ Recording complete!"
echo ""
echo "📋 Created cassettes:"
ls -lh test/vcr_cassettes/*.yaml | tail -10
