#!/bin/bash

# GRCTool Evidence Assembly Demo Script
# This script demonstrates the complete workflow of assembling compliance evidence

# Settings for better recording
export PS1='$ '
GRCTOOL="./build/grctool"

# Helper function to add pauses and show commands
demo_cmd() {
    echo ""
    echo "# $1"
    echo "\$ $2"
    sleep 1
    eval "$2"
    sleep 2
}

clear

# Title
cat << "EOF"
╔═══════════════════════════════════════════════════════════════╗
║                                                               ║
║            GRCTool Evidence Assembly Demo                    ║
║                                                               ║
║    Assembling Compliance Evidence from Policies,             ║
║    Controls, GitHub, and Terraform                           ║
║                                                               ║
╚═══════════════════════════════════════════════════════════════╝

EOF
sleep 3

demo_cmd "Step 1: Check GRCTool version" \
    "$GRCTOOL --version"

demo_cmd "Step 2: Show available evidence collection tools" \
    "$GRCTOOL tool --help | grep -A 2 'Available Tools:'"

demo_cmd "Step 3: List evidence tasks (first 5)" \
    "$GRCTOOL tool evidence-task-list --output json 2>/dev/null | jq -r '.[] | \"\\(.ref): \\(.title)\"' | head -5 || echo 'ET-0001: Access Control & Registration Process\nET-0047: GitHub Repository Access Controls\nET-0104: Infrastructure Security Configuration\nET-0156: Terraform Security Analysis\nET-0201: Deployment Access Controls'"

demo_cmd "Step 4: Get details about an evidence task (ET-0001)" \
    "$GRCTOOL tool evidence-task-details --task-ref ET-0001 --output json 2>/dev/null | jq '{ref, title, description, related_controls}' || echo '{\"ref\": \"ET-0001\", \"title\": \"Access Control Documentation\", \"description\": \"Provide evidence of access control processes\", \"related_controls\": [\"CC6.1\", \"AC-01\"]}' | jq ."

demo_cmd "Step 5: Generate policy summary for context" \
    "$GRCTOOL tool policy-summary-generator --task-ref ET-0001 --output markdown 2>/dev/null | head -20 || echo '# Policy Summary for ET-0001\n\n## Relevant Policies\n- Access Control Policy (POL-0001)\n- User Registration Policy (POL-0023)\n\n## Key Requirements\n- Document user registration process\n- Define access approval workflow\n- Maintain access control lists'"

demo_cmd "Step 6: Generate control summary" \
    "$GRCTOOL tool control-summary-generator --task-ref ET-0001 --output markdown 2>/dev/null | head -20 || echo '# Control Summary for ET-0001\n\n## Related Controls\n- **CC6.1**: Restrict logical access\n- **AC-01**: Access control policy and procedures\n\n## Control Requirements\n- Implement role-based access controls\n- Review access permissions quarterly'"

demo_cmd "Step 7: View evidence relationships" \
    "$GRCTOOL tool evidence-relationships --task-ref ET-0001 --max-depth 2 --output json 2>/dev/null | jq '.relationships | {evidence_task, controls: [.controls[].ref], policies: [.policies[].ref]}' || echo '{\"evidence_task\": \"ET-0001\", \"controls\": [\"CC6.1\", \"AC-01\"], \"policies\": [\"POL-0001\", \"POL-0023\"]}' | jq ."

cat << "EOF"

════════════════════════════════════════════════════════════════
              GitHub Evidence Collection
════════════════════════════════════════════════════════════════
EOF
sleep 2

demo_cmd "Step 8: Analyze GitHub repository permissions" \
    "echo 'Demo: Extracting repository access controls...'; echo '{\"repository\": \"7thsense-ai/platform\", \"collaborators\": 12, \"teams\": 3, \"branch_protection\": {\"main\": {\"required_reviews\": 2, \"dismiss_stale_reviews\": true}}}' | jq ."

demo_cmd "Step 9: Analyze GitHub Actions workflows" \
    "echo 'Demo: Analyzing CI/CD security controls...'; echo '{\"total_workflows\": 5, \"deployment_workflows\": 2, \"workflows_with_approvals\": 2, \"security_scanning_enabled\": true}' | jq ."

demo_cmd "Step 10: Check GitHub security features" \
    "echo 'Demo: Checking repository security features...'; echo '{\"repository\": \"7thsense-ai/platform\", \"secret_scanning\": \"enabled\", \"dependabot_alerts\": \"enabled\", \"code_scanning\": \"enabled\", \"branch_protection\": \"enabled\"}' | jq ."

cat << "EOF"

════════════════════════════════════════════════════════════════
              Terraform Evidence Collection
════════════════════════════════════════════════════════════════
EOF
sleep 2

demo_cmd "Step 11: Index Terraform resources" \
    "echo 'Demo: Indexing infrastructure resources...'; echo '{\"total_resources\": 245, \"resource_types\": [\"aws_iam_role\", \"aws_security_group\", \"aws_s3_bucket\"], \"iam_roles\": 15, \"security_groups\": 8}' | jq ."

demo_cmd "Step 12: Query for access control resources" \
    "echo 'Demo: Finding IAM policies and roles...'; cat << 'DEMO'
## Terraform Access Control Resources

### IAM Roles
- **prod-admin-role**: Full administrative access
- **developer-role**: Read-only production access
- **ci-cd-role**: Deployment permissions

### Key Policies
- Principle of least privilege enforced
- MFA required for admin access
- Time-based session restrictions
DEMO
"

demo_cmd "Step 13: Run Terraform security analysis" \
    "echo 'Demo: Analyzing security configurations...'; echo '{\"domain\": \"identity_access\", \"resources_analyzed\": 15, \"findings\": [{\"severity\": \"low\", \"resource\": \"aws_iam_role.developer\", \"soc2_control\": \"CC6.1\"}, {\"severity\": \"info\", \"resource\": \"aws_iam_policy.readonly\", \"soc2_control\": \"CC6.2\"}]}' | jq ."

cat << "EOF"

════════════════════════════════════════════════════════════════
              Evidence Assembly & Generation
════════════════════════════════════════════════════════════════
EOF
sleep 2

demo_cmd "Step 14: Generate complete evidence package" \
    "echo 'Demo: Generating evidence for ET-0001...'; echo '✓ Loaded policy context\n✓ Loaded control requirements\n✓ Collected GitHub evidence\n✓ Collected Terraform evidence\n✓ Generated evidence summary\n✓ Saved to: evidence/ET-0001-access-control/2025-Q4/evidence.json'"

demo_cmd "Step 15: Evaluate evidence quality" \
    "echo 'Demo: Evaluating evidence completeness...'; echo '{\"score\": 95, \"completeness\": \"high\", \"recommendations\": [\"Add quarterly access review logs\", \"Include MFA enrollment metrics\"]}' | jq ."

demo_cmd "Step 16: Review evidence (summary)" \
    "cat << 'REVIEW'
Evidence Review for ET-0001
============================

Status: COMPLETE
Score: 95/100

Evidence Components:
✓ Policy documentation (POL-0001, POL-0023)
✓ Control implementation details (CC6.1, AC-01)
✓ GitHub access controls (12 collaborators, 3 teams)
✓ Branch protection enforced (2 required reviews)
✓ Terraform IAM configuration (15 roles analyzed)
✓ Security scanning enabled

Recommendations:
- Add quarterly access review logs
- Include MFA enrollment metrics

READY FOR SUBMISSION
REVIEW
"

cat << "EOF"

════════════════════════════════════════════════════════════════
                    Summary
════════════════════════════════════════════════════════════════

We've demonstrated the complete evidence assembly workflow:

✓ 1. Identified evidence task requirements
✓ 2. Gathered policy and control context
✓ 3. Collected GitHub repository evidence
✓ 4. Analyzed Terraform infrastructure
✓ 5. Generated comprehensive evidence package
✓ 6. Validated evidence quality

The result: Auditor-ready compliance evidence combining
policy documentation with real infrastructure data!

For more information: grctool --help

════════════════════════════════════════════════════════════════
EOF
sleep 3
