#!/bin/bash
set -e

# Get GitHub token from environment
echo "Checking for GitHub token..."
TOKEN="${GITHUB_TOKEN}"

if [ -z "$TOKEN" ]; then
    echo "❌ GITHUB_TOKEN environment variable not set"
    echo "   Please set it with: export GITHUB_TOKEN='your-token-here'"
    exit 1
fi

REPO="grctool/grctool"

echo "Creating test issues in $REPO..."
echo ""

# Issue 1: Privacy Policy
echo "1. Creating privacy policy issue..."
ISSUE1=$(curl -s -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Accept: application/vnd.github.v3+json" \
  https://api.github.com/repos/$REPO/issues \
  -d '{"title":"Privacy policy documentation","body":"Document privacy policies and procedures for SOC2 compliance","labels":["policy","privacy","documentation"]}')
echo "   Issue #$(echo $ISSUE1 | jq -r '.number // "ERROR"')"

# Issue 2: Encryption
echo "2. Creating encryption issue..."
ISSUE2=$(curl -s -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Accept: application/vnd.github.v3+json" \
  https://api.github.com/repos/$REPO/issues \
  -d '{"title":"Implement encryption at rest","body":"Enable KMS encryption for all data stores for CC6.8 compliance","labels":["encryption","security","soc2","data-protection"]}')
echo "   Issue #$(echo $ISSUE2 | jq -r '.number // "ERROR"')"

# Issue 3: Access Control
echo "3. Creating access control issue..."
ISSUE3=$(curl -s -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Accept: application/vnd.github.v3+json" \
  https://api.github.com/repos/$REPO/issues \
  -d '{"title":"Review IAM policies for least privilege","body":"Audit all IAM roles and policies for CC6.1 compliance","labels":["access-control","security","iam"]}')
echo "   Issue #$(echo $ISSUE3 | jq -r '.number // "ERROR"')"

# Issue 4: Audit Logging
echo "4. Creating audit logging issue..."
ISSUE4=$(curl -s -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Accept: application/vnd.github.v3+json" \
  https://api.github.com/repos/$REPO/issues \
  -d '{"title":"Setup comprehensive audit logging","body":"Configure CloudTrail and CloudWatch for CC8.1 compliance","labels":["audit","logging","compliance"]}')
echo "   Issue #$(echo $ISSUE4 | jq -r '.number // "ERROR"')"

# Issue 5: Change Management
echo "5. Creating change management issue..."
ISSUE5=$(curl -s -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Accept: application/vnd.github.v3+json" \
  https://api.github.com/repos/$REPO/issues \
  -d '{"title":"Document change management process","body":"Create documentation for change management and audit logging procedures","labels":["change-management","audit","logging"]}')
echo "   Issue #$(echo $ISSUE5 | jq -r '.number // "ERROR"')"

# Issue 6: Incident Response
echo "6. Creating incident response issue..."
ISSUE6=$(curl -s -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Accept: application/vnd.github.v3+json" \
  https://api.github.com/repos/$REPO/issues \
  -d '{"title":"Security incident response plan","body":"Create and document incident response procedures for security events","labels":["incident-response","security","process"]}')
echo "   Issue #$(echo $ISSUE6 | jq -r '.number // "ERROR"')"

echo ""
echo "✅ All test issues created successfully!"
echo ""
echo "View issues at: https://github.com/$REPO/issues"
