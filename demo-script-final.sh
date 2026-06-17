#!/bin/bash

# GRCTool Evidence Assembly Screencast
# Demonstrates assembling compliance evidence from policies, controls, GitHub, and Terraform

export PS1='$ '
GRCTOOL="./build/grctool"

# Helper function for better presentation
show() {
    echo ""
    echo "# $1"
    sleep 1
}

cmd() {
    echo "\$ $1"
    sleep 0.5
    eval "$1"
    sleep 2
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
sleep 3

show "1. Check GRCTool version"
cmd "$GRCTOOL version"

show "2. List available evidence collection tools"
cmd "$GRCTOOL tool list --output json | jq -r '.data.result | fromjson | .tools[0:5] | .[] | \"  - \\(.name): \\(.description[0:60])...\"'"

cat << "EOF"

══════════════════════════════════════════════════════════
           Evidence Task Discovery
══════════════════════════════════════════════════════════
EOF
sleep 2

show "3. List evidence tasks (showing first 5)"
cmd "$GRCTOOL tool evidence-task-list --output json | jq -r '.data.evidence_source.content | fromjson | .tasks[0:5] | .[] | \"\\(.reference_id): \\(.name)\"'"

show "4. Get detailed information for ET-0001"
cmd "$GRCTOOL tool evidence-task-details --task-ref ET-0001 --output json | jq -r '.data.result | fromjson | {
  ref: .task.reference_id,
  title: .task.name,
  description: .task.description,
  category: .requirements.category,
  controls: .relationships.control_count
}'"

cat << "EOF"

══════════════════════════════════════════════════════════
           Understand Context: Relationships
══════════════════════════════════════════════════════════
EOF
sleep 2

show "5. Map relationships between evidence task, controls, and policies"
cmd "$GRCTOOL tool evidence-relationships --task-ref ET-0001 --depth 2 --output json | jq -r '.data.result | {
  task: .task.reference_id,
  controls: [.relationships.direct.controls[] | .reference_id],
  total_controls: .summary.total_controls
}'"

show "6. View control details (AC1)"
cmd "$GRCTOOL tool evidence-relationships --task-ref ET-0001 --depth 2 --output json | jq -r '.data.result | .relationships.direct.controls[0] | {
  id: .reference_id,
  name: .name,
  description: .description,
  status: .status
}'"

cat << "EOF"

══════════════════════════════════════════════════════════
           Generate Control Summary
══════════════════════════════════════════════════════════
EOF
sleep 2

show "7. Generate control summary for AC1 in context of ET-0001"
cmd "$GRCTOOL tool control-summary-generator --control-id AC1 --task-ref ET-0001 --output json | jq -r '.data.evidence_source.metadata | {
  control_name: .control_name,
  task_reference: .task_reference,
  output_format: .output_format,
  file_saved: .file_path
}'"

cat << "EOF"

══════════════════════════════════════════════════════════
           GitHub Evidence Collection
══════════════════════════════════════════════════════════
EOF
sleep 2

show "8. Extract GitHub repository permissions"
cmd "$GRCTOOL tool github-permissions --repository telepathdata/7thsense --output-format summary --output json | jq -r '.data.evidence_source.metadata | {
  repository: .repository,
  collaborators: .total_collaborators,
  teams: .total_teams,
  protected_branches: .protected_branches,
  authenticated: .auth_status.authenticated
}'"

show "9. Analyze GitHub workflows for CI/CD controls"
cmd "$GRCTOOL tool github-workflow-analyzer --analysis-type full --output json | jq -r '.data.evidence_source.metadata | {
  repository: .repository,
  workflows: .workflow_count,
  security_scans: .security_scans,
  approval_rules: .approval_rules
}'"

cat << "EOF"

══════════════════════════════════════════════════════════
           Terraform Infrastructure Evidence
══════════════════════════════════════════════════════════
EOF
sleep 2

show "10. Scan Terraform configurations for IAM roles"
cmd "$GRCTOOL tool terraform-scanner --resource-types aws_iam_role --max-results 5 --output json | jq -r '.data.evidence_source.metadata | {
  scan_paths: .scan_paths,
  resources_found: .resource_count,
  analysis_type: .analysis_type
}'"

show "11. Search for encryption-related resources"
cmd "$GRCTOOL tool terraform-scanner --pattern 'encrypt|kms' --max-results 3 --output json | jq -r '.data.evidence_source.metadata | {
  pattern_searched: \"encrypt|kms\",
  resources_found: .resource_count
}'"

cat << "EOF"

══════════════════════════════════════════════════════════
           Review Existing Evidence
══════════════════════════════════════════════════════════
EOF
sleep 2

show "12. List existing evidence for ET-0001"
cmd "ls -la /Users/erik/Projects/7thsense-ops/isms/evidence/Access_Control_Registration_and_De-registration_Process_Document_ET-0001_327992/2025-Q4/ | grep -E '\.md|\.json' | tail -5"

show "13. Show evidence directory structure"
cmd "find /Users/erik/Projects/7thsense-ops/isms/evidence/Access_Control_Registration_and_De-registration_Process_Document_ET-0001_327992/2025-Q4 -type f -name '*.md' | head -5"

cat << "EOF"

══════════════════════════════════════════════════════════
           Evidence Assembly Summary
══════════════════════════════════════════════════════════

We've demonstrated the complete evidence assembly workflow:

✓ 1. Discovered evidence tasks and requirements
✓ 2. Mapped relationships to controls and policies
✓ 3. Generated control-specific summaries
✓ 4. Collected GitHub repository evidence
✓ 5. Analyzed Terraform infrastructure
✓ 6. Reviewed existing evidence files

The result: A comprehensive, auditor-ready evidence package
that combines policy documentation with real infrastructure data!

══════════════════════════════════════════════════════════

Next Steps:
  - grctool evidence generate ET-0001    # Generate new evidence
  - grctool evidence evaluate ET-0001    # Validate evidence quality
  - grctool evidence submit ET-0001      # Submit to Tugboat Logic

For more information: grctool --help

══════════════════════════════════════════════════════════
EOF
sleep 5
