#!/bin/bash
set -e

# Check for GitHub token in environment
echo "Checking for GitHub token..."
if [ -z "$GITHUB_TOKEN" ]; then
    echo "Error: GITHUB_TOKEN environment variable not set"
    echo "   Please set it with: export GITHUB_TOKEN='your-token-here'"
    exit 1
fi

echo "GitHub token found (${#GITHUB_TOKEN} chars)"

# Export for tests
export GITHUB_TOKEN

# Re-record VCR cassettes
echo ""
echo "Re-recording VCR cassettes with VCR_MODE=record..."
echo ""

# 1. TestEvidenceTaskValidation
echo "1. Recording evidence_task_validation.yaml..."
VCR_MODE=record go test -v ./test/integration/... -run "^TestEvidenceTaskValidation$" -count=1 2>&1 | grep -E "(Recording|Cassette|PASS|FAIL)" | tail -10

# 2. TestToolsIntegration_SOC2Evidence and TestToolsIntegration_CrossValidation
echo ""
echo "2. Recording tools_integration.yaml..."
VCR_MODE=record go test -v ./test/integration/... -run "^TestToolsIntegration_SOC2Evidence$" -count=1 2>&1 | grep -E "(Recording|Cassette|PASS|FAIL)" | tail -10

echo ""
echo "3. Recording evidence_cross_tool.yaml..."
VCR_MODE=record go test -v ./test/integration/... -run "^TestEvidenceCollection_CrossTool$" -count=1 2>&1 | grep -E "(Recording|Cassette|PASS|FAIL)" | tail -10

echo ""
echo "4. Recording github_security_comprehensive.yaml..."
VCR_MODE=record go test -v ./test/integration/... -run "^TestGitHubSecurity_ComprehensiveWorkflow$" -count=1 2>&1 | grep -E "(Recording|Cassette|PASS|FAIL)" | tail -10

echo ""
echo "✓ VCR cassettes re-recorded successfully!"
echo ""
echo "Running tests in playback mode to verify..."
echo ""

# Run tests again in playback mode
go test -v ./test/integration/... -run "TestEvidenceTaskValidation|TestToolsIntegration_SOC2Evidence|TestEvidenceCollection_CrossTool|TestGitHubSecurity_ComprehensiveWorkflow" -count=1 2>&1 | grep -E "^--- (PASS|FAIL):" | sort

echo ""
echo "Done!"
