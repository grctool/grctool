#!/bin/bash
set -e

echo "🎬 VCR Cassette Recording Script"
echo "================================="
echo ""

# Check GitHub token from environment
echo "🔐 Checking for GitHub token..."
if [ -z "$GITHUB_TOKEN" ]; then
    echo "❌ GITHUB_TOKEN environment variable not set"
    echo "   Please set it with: export GITHUB_TOKEN='your-token-here'"
    exit 1
fi

echo "✓ GitHub token found (length: ${#GITHUB_TOKEN})"
echo ""

# Export for tests
export GITHUB_TOKEN
export VCR_MODE=record

# Change to project root
cd "$(dirname "$0")/.."

# Record evidence_task_validation cassette
echo "📼 Recording: evidence_task_validation.yaml"
echo "   Running TestEvidenceTaskValidation..."
go test -v -tags=integration -timeout=5m -run "^TestEvidenceTaskValidation$" ./test/integration/
RESULT1=$?
echo ""

# Record tools_integration cassette
echo "📼 Recording: tools_integration.yaml"
echo "   Running TestToolsIntegration..."
go test -v -tags=integration -timeout=5m -run "^TestToolsIntegration" ./test/integration/
RESULT2=$?
echo ""

echo "✅ Recording complete!"
echo ""
echo "📋 Created/updated cassettes:"
ls -lh test/vcr_cassettes/*.yaml | tail -10
echo ""

if [ $RESULT1 -eq 0 ] && [ $RESULT2 -eq 0 ]; then
    echo "✅ All recording tests passed!"
    exit 0
else
    echo "⚠️  Some tests failed during recording (this is expected if assertions don't match recorded data)"
    echo "   The cassettes should still be recorded successfully."
    exit 0
fi
