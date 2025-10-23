#!/bin/bash
set -e

# Script to record VCR cassettes for integration tests
# Usage: GITHUB_TOKEN=your_token ./scripts/record-vcr-cassettes.sh

if [ -z "$GITHUB_TOKEN" ]; then
    echo "❌ Error: GITHUB_TOKEN environment variable is not set"
    echo ""
    echo "Please set your GitHub token:"
    echo "  export GITHUB_TOKEN='your_github_token_here'"
    echo ""
    echo "Then run this script again:"
    echo "  ./scripts/record-vcr-cassettes.sh"
    exit 1
fi

echo "🎬 Recording VCR cassettes for integration tests"
echo "================================================"
echo ""
echo "This will make real GitHub API calls to record HTTP interactions."
echo "The cassettes will be saved to: test/vcr_cassettes/"
echo ""

# Create backup of existing cassettes
BACKUP_DIR="test/vcr_cassettes/.backup-$(date +%Y%m%d-%H%M%S)"
if [ -d "test/vcr_cassettes" ]; then
    echo "📦 Creating backup of existing cassettes..."
    mkdir -p "$BACKUP_DIR"
    cp -r test/vcr_cassettes/*.yaml "$BACKUP_DIR/" 2>/dev/null || true
    echo "   Backup saved to: $BACKUP_DIR"
    echo ""
fi

# Temporarily modify VCR config to use record mode
# We'll do this by setting an environment variable that the tests can check
export VCR_MODE=record
export VCR_RECORD=true

echo "🎥 Recording cassette: evidence_task_validation.yaml"
echo "   Test: TestEvidenceTaskValidation"
go test -v -tags=integration -timeout=5m \
    -run "TestEvidenceTaskValidation" \
    ./test/integration/evidence_task_validation_test.go \
    ./test/integration/testtools.go \
    2>&1 | grep -E "(RUN|PASS|FAIL|Error)" || true
echo ""

echo "🎥 Recording cassette: tools_integration.yaml"
echo "   Tests: TestToolsIntegration_*"
go test -v -tags=integration -timeout=5m \
    -run "TestToolsIntegration" \
    ./test/integration/tools_integration_test.go \
    2>&1 | grep -E "(RUN|PASS|FAIL|Error)" || true
echo ""

echo "✅ Recording complete!"
echo ""
echo "📋 Recorded cassettes:"
ls -lh test/vcr_cassettes/*.yaml | tail -10
echo ""
echo "🧪 Now run the tests to verify:"
echo "   go test -v -tags=integration ./test/integration/..."
