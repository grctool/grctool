#!/bin/bash

# GRCTool Evidence Assembly Screencast - Enhanced Version
# Demonstrates assembling compliance evidence from policies, controls, GitHub, and Terraform

export PS1='$ '
GRCTOOL="./build/grctool"

# Helper functions
show() {
    echo ""
    echo "# $1"
    sleep 1.5
}

cmd() {
    echo "\$ $1"
    sleep 0.8
    eval "$1"
    sleep 3
}

clear

# Title
cat << "EOF"
╔═══════════════════════════════════════════════════════════════╗
║                                                               ║
║          GRCTool Evidence Assembly Demonstration             ║
║                                                               ║
║    Assembling Compliance Evidence from Policies,             ║
║    Controls, GitHub, and Terraform                           ║
║                                                               ║
╚═══════════════════════════════════════════════════════════════╝

EOF
sleep 4

show "Step 1: Check GRCTool version and build info"
cmd "$GRCTOOL version"

cat << "EOF"

══════════════════════════════════════════════════════════
           Discover Evidence Tasks
══════════════════════════════════════════════════════════
EOF
sleep 2

show "Step 2: List evidence tasks (first 5 tasks)"
cmd "$GRCTOOL tool evidence-task-list --output json | jq -r '.data.evidence_source.content | fromjson | .tasks[0:5] | .[] | \"\\(.reference_id): \\(.name)\"'"

show "Step 3: Get detailed information for ET-0001"
cmd "$GRCTOOL tool evidence-task-details --task-ref ET-0001 --output json | jq '.data.result | fromjson | {
  ref: .task.reference_id,
  title: .task.name,
  description: .task.description,
  category: .requirements.category,
  controls: .relationships.control_count
}'"

cat << "EOF"

══════════════════════════════════════════════════════════
           Analyze Relationships
══════════════════════════════════════════════════════════
EOF
sleep 2

show "Step 4: Map relationships (evidence → controls → policies)"
cmd "$GRCTOOL tool evidence-relationships --task-ref ET-0001 --depth 2 --output json | jq '.data.result | fromjson | {
  task: .task.reference_id,
  controls: [.relationships.direct.controls[] | .reference_id],
  total_controls: .summary.total_controls,
  total_policies: .summary.total_policies
}'"

show "Step 5: View details for control AC1 (Access Provisioning)"
cmd "$GRCTOOL tool evidence-relationships --task-ref ET-0001 --depth 2 --output json | jq '.data.result | fromjson | .relationships.direct.controls[0] | {
  id: .reference_id,
  name: .name,
  category: .category,
  status: .status
}'"

cat << "EOF"

══════════════════════════════════════════════════════════
           Generate Control Context
══════════════════════════════════════════════════════════
EOF
sleep 2

show "Step 6: Generate control summary for AC1 (creates markdown file)"
cmd "$GRCTOOL tool control-summary-generator --control-id AC1 --task-ref ET-0001 --output json | jq '.data.evidence_source.metadata | {
  control: .control_name,
  task: .task_reference,
  file: .file_path
}'"

show "Step 6b: Preview the generated markdown summary (first 25 lines)"
cmd "ls -t /home/erik/Projects/7thsense-ops/isms/summaries/controls/control_summary_ET-0001_778771_*.md 2>/dev/null | head -1 | xargs head -25 || echo 'No summary file found'"

cat << "EOF"

══════════════════════════════════════════════════════════
           Collect GitHub Evidence
══════════════════════════════════════════════════════════
EOF
sleep 2

show "Step 7: Extract GitHub repository permissions"
cmd "$GRCTOOL tool github-permissions --repository telepathdata/7thsense --output-format summary --output json | jq '.data.evidence_source.metadata | {
  repository: .repository,
  collaborators: .total_collaborators,
  teams: .total_teams,
  protected_branches: .protected_branches,
  authenticated: .auth_status.authenticated,
  cache_used: .auth_status.cache_used
}'"

show "Step 8: Analyze GitHub workflows"
cmd "$GRCTOOL tool github-workflow-analyzer --analysis-type full --output json | jq '.data.evidence_source.metadata | {
  repository: .repository,
  workflows: .workflow_count,
  security_scans: .security_scans,
  approval_rules: .approval_rules,
  analysis: .analysis_type
}'"

cat << "EOF"

══════════════════════════════════════════════════════════
           Scan Terraform Infrastructure
══════════════════════════════════════════════════════════
EOF
sleep 2

show "Step 9: Scan Terraform for IAM roles"
cmd "$GRCTOOL tool terraform-scanner --resource-types aws_iam_role --max-results 5 --output json | jq '.data.evidence_source.metadata | {
  scan_paths: .scan_paths,
  resources_found: .resource_count
}'"

show "Step 10: Search for encryption patterns"
cmd "$GRCTOOL tool terraform-scanner --pattern 'encrypt|kms' --max-results 3 --output json | jq '.data.evidence_source.metadata | {
  pattern: \"encrypt|kms\",
  resources: .resource_count
}'"

cat << "EOF"

══════════════════════════════════════════════════════════
           Review Generated Evidence
══════════════════════════════════════════════════════════
EOF
sleep 2

show "Step 11: Show evidence files for ET-0001"
cmd "find /Users/erik/Projects/7thsense-ops/isms/evidence/Access_Control_Registration_and_De-registration_Process_Document_ET-0001_327992/2025-Q4 -type f -name '*.md' | head -5 | xargs -I {} basename {}"

show "Step 12: Preview an evidence file (first 30 lines)"
cmd "head -30 /Users/erik/Projects/7thsense-ops/isms/evidence/Access_Control_Registration_and_De-registration_Process_Document_ET-0001_327992/2025-Q4/archive/ET-0001_Evidence.md"

cat << "EOF"

══════════════════════════════════════════════════════════
                    Summary
══════════════════════════════════════════════════════════

✅ Evidence Assembly Complete!

We demonstrated:
  1. Discovering evidence tasks and requirements
  2. Mapping relationships to controls and policies
  3. Generating control-specific context (with preview!)
  4. Collecting GitHub repository evidence
  5. Scanning Terraform infrastructure
  6. Reviewing generated evidence files (with preview!)

Result: A comprehensive, auditor-ready evidence package!

══════════════════════════════════════════════════════════

Next Commands to Try:

  $ grctool evidence generate ET-0001
    → Generate new evidence for a task

  $ grctool evidence evaluate ET-0001
    → Validate evidence completeness and quality

  $ grctool evidence submit ET-0001
    → Submit evidence to Tugboat Logic

For help: grctool --help

══════════════════════════════════════════════════════════
EOF
sleep 6
