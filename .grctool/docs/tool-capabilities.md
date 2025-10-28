# Tool Capabilities Reference

> Complete listing of all available tools and their purposes

---

**Generated**: 2025-10-28 10:33:19 EDT
**GRCTool Version**: dev
**Documentation Version**: dev

---

## Overview

GRCTool provides 30+ specialized tools for automated evidence collection. Each tool is designed for specific data sources and compliance requirements.

**Quick Command Reference**:
```bash
# List all tools
grctool tool --help

# Get tool-specific help
grctool tool <tool-name> --help
```

---

## Evidence Collection Tools

### Terraform Infrastructure Tools

#### `terraform-security-indexer`
**Purpose**: Fast infrastructure security scanning with indexed queries
**Use When**: You need quick lookups of security configurations
**Evidence Tasks**: Infrastructure security, access controls, encryption policies

```bash
# Query specific control mappings
grctool tool terraform-security-indexer --query-type control_mapping

# Search for specific security configurations
grctool tool terraform-security-indexer --query-type search --search-term "encryption"
```

#### `terraform-security-analyzer`
**Purpose**: Deep security analysis of Terraform configurations
**Use When**: You need comprehensive security assessments
**Evidence Tasks**: Security control analysis, risk assessment

```bash
# Analyze all security domains
grctool tool terraform-security-analyzer --security-domain all

# Focus on specific domain
grctool tool terraform-security-analyzer --security-domain access_control
```

#### `terraform-hcl-parser`
**Purpose**: Parse and extract data from HCL/Terraform files
**Use When**: You need structured data from Terraform code
**Evidence Tasks**: Configuration documentation, resource inventory

#### `terraform-snippets`
**Purpose**: Extract relevant code snippets from Terraform files
**Use When**: You need code examples for auditors
**Evidence Tasks**: Configuration examples, security settings

#### `terraform-atmos-analyzer`
**Purpose**: Multi-environment stack analysis (Atmos framework)
**Use When**: You use Atmos for environment management
**Evidence Tasks**: Multi-environment consistency, configuration management

```bash
# Analyze all stacks
grctool tool terraform-atmos-analyzer --stack all
```

#### `terraform-query-interface`
**Purpose**: Query interface for indexed Terraform data
**Use When**: You need to run custom queries on infrastructure
**Evidence Tasks**: Custom security queries, compliance checks

### GitHub Repository Tools

#### `github-permissions`
**Purpose**: Analyze repository and team permissions
**Use When**: Collecting access control evidence
**Evidence Tasks**: Repository access, team permissions, role assignments

```bash
# Matrix view of permissions
grctool tool github-permissions --repository owner/repo --output-format matrix

# JSON export for processing
grctool tool github-permissions --repository owner/repo --output-format json
```

#### `github-deployment-access`
**Purpose**: Analyze deployment environment access controls
**Use When**: Documenting production access restrictions
**Evidence Tasks**: Deployment controls, environment protection

#### `github-security-features`
**Purpose**: Audit GitHub security features (branch protection, code scanning)
**Use When**: Demonstrating security feature adoption
**Evidence Tasks**: Branch protection, security scanning, vulnerability management

```bash
# Check security features
grctool tool github-security-features --repository owner/repo
```

#### `github-workflow-analyzer`
**Purpose**: Analyze CI/CD workflows and security controls
**Use When**: Documenting build/deploy processes
**Evidence Tasks**: CI/CD security, workflow controls, secrets management

```bash
# Analyze workflows
grctool tool github-workflow-analyzer --repository owner/repo
```

#### `github-review-analyzer`
**Purpose**: Analyze code review practices and enforcement
**Use When**: Demonstrating code review requirements
**Evidence Tasks**: Code review enforcement, approval requirements

#### `github-enhanced`
**Purpose**: Enhanced GitHub data collection with richer metadata
**Use When**: You need comprehensive GitHub evidence
**Evidence Tasks**: Comprehensive repository analysis

### Google Workspace Tools

#### `google-workspace`
**Purpose**: Extract evidence from Google Drive, Docs, Sheets, Forms
**Use When**: Policy documents, training records, reviews stored in Google Workspace
**Evidence Tasks**: Policy documentation, training completion, access reviews

```bash
# Extract policy document
grctool tool google-workspace \
  --document-id 1A2B3C4D5E6F7G8H9I0J \
  --document-type docs

# Extract access review spreadsheet
grctool tool google-workspace \
  --document-id 1K2L3M4N5O6P7Q8R9S0T \
  --document-type sheets \
  --sheet-range "Q4 2025!A1:F100"

# Extract training quiz responses
grctool tool google-workspace \
  --document-id 1U2V3W4X5Y6Z7A8B9C0D \
  --document-type forms
```

**Setup Required**: Google service account credentials (see `docs/01-User-Guide/google-workspace-setup.md`)

---

## Evidence Analysis & Discovery Tools

### `evidence-task-list`
**Purpose**: List all evidence tasks from Tugboat
**Use When**: You need to see what evidence is required

```bash
# List all tasks
grctool tool evidence-task-list

# Filter by status
grctool tool evidence-task-list --status pending
```

### `evidence-task-details`
**Purpose**: Get detailed information about a specific evidence task
**Use When**: You need to understand task requirements

```bash
grctool tool evidence-task-details --task-ref ET-0047
```

### `evidence-relationships`
**Purpose**: Show relationships between tasks, controls, and policies
**Use When**: Understanding control mappings

### `policy-summary-generator`
**Purpose**: Generate summaries of policy documents
**Use When**: Creating policy overviews for evidence

### `control-summary-generator`
**Purpose**: Generate summaries of security controls
**Use When**: Creating control documentation

### `docs-reader`
**Purpose**: Read and parse synced documentation files
**Use When**: You need to access policies/controls programmatically

---

## Evidence Management Tools

### `evidence-writer`
**Purpose**: Write evidence files with automatic metadata tracking
**Use When**: Saving generated evidence

```bash
# Write from file
grctool tool evidence-writer \
  --task-ref ET-0047 \
  --title "GitHub Permissions" \
  --file permissions.csv

# Write from stdin
grctool tool github-permissions --repository org/repo | \
  grctool tool evidence-writer \
    --task-ref ET-0047 \
    --title "Permissions" \
    --format csv
```

**Automatic Metadata**: Creates `.generation/metadata.yaml` with checksums, timestamps, and tool tracking

### `evidence-generator`
**Purpose**: Generate evidence context for Claude Code assistance
**Use When**: Starting new evidence generation
**Note**: Use via `grctool evidence generate` command

### `evidence-validator`
**Purpose**: Evaluate evidence quality and completeness with automated scoring
**Use When**: Before submitting evidence to ensure quality
**Note**: Use via `grctool evidence evaluate` command (not `validate`)

### `tugboat-sync-wrapper`
**Purpose**: Sync data from Tugboat Logic API
**Use When**: Updating local data from Tugboat
**Note**: Use via `grctool sync` command

---

## Storage & Utility Tools

### `storage-read`
**Purpose**: Read files from GRCTool data directory
**Use When**: Accessing synced data programmatically

### `storage-write`
**Purpose**: Write files to GRCTool data directory
**Use When**: Saving processed data

### `name-generator`
**Purpose**: Generate standardized evidence file names
**Use When**: Maintaining naming conventions

### `grctool-run`
**Purpose**: Execute grctool commands programmatically
**Use When**: Orchestrating multiple commands

---

## Tool Selection Guide

### By Evidence Type

| Evidence Type | Recommended Tools |
|---------------|-------------------|
| **Infrastructure Security** | `terraform-security-indexer`, `terraform-security-analyzer` |
| **Access Controls** | `github-permissions`, `github-deployment-access` |
| **CI/CD Security** | `github-workflow-analyzer`, `github-security-features` |
| **Code Review** | `github-review-analyzer` |
| **Policy Documentation** | `google-workspace` (docs), `policy-summary-generator` |
| **Training Records** | `google-workspace` (forms, sheets) |
| **Access Reviews** | `google-workspace` (sheets), `github-permissions` |
| **Multi-Environment** | `terraform-atmos-analyzer` |

### By Automation Level

**Fully Automated** (can run without manual intervention):
- All Terraform tools
- All GitHub tools
- Evidence analysis tools

**Partially Automated** (needs configuration/authentication):
- `google-workspace` (requires service account setup)

**Manual** (requires manual document creation):
- Policy documents not in Google Workspace
- Screenshots and images
- Third-party vendor documentation

---

## Common Evidence Collection Workflows

### Workflow 1: GitHub Access Control Evidence

```bash
# 1. Generate context
grctool evidence generate ET-0047 --window 2025-Q4

# 2. Collect permissions data (for analysis)
grctool tool github-permissions --repository org/repo > /tmp/perms.json

# 3. Collect security features (for analysis)
grctool tool github-security-features --repository org/repo > /tmp/security.json

# 4. With Claude Code: analyze JSON and write markdown evidence
# Claude reads the JSON and creates human-readable summaries

# 5. Save markdown evidence to root directory
grctool tool evidence-writer --task-ref ET-0047 --title "Permissions Analysis" --file /tmp/permissions_summary.md
# Saved to: evidence/ET-0047_*/2025-Q4/01_permissions_analysis.md

grctool tool evidence-writer --task-ref ET-0047 --title "Security Features" --file /tmp/security_summary.md
# Saved to: evidence/ET-0047_*/2025-Q4/02_security_summary.md
```

### Workflow 2: Infrastructure Security Evidence

```bash
# 1. Generate context
grctool evidence generate ET-0023 --window 2025-Q4

# 2. Run security indexer (for analysis)
grctool tool terraform-security-indexer --query-type all > /tmp/infra-security.json

# 3. Deep security analysis (for analysis)
grctool tool terraform-security-analyzer --security-domain all > /tmp/security-analysis.json

# 4. With Claude Code: analyze JSON and write markdown evidence
# Claude reads the JSON outputs and creates comprehensive summaries

# 5. Save markdown evidence and original source files to root directory
grctool tool evidence-writer --task-ref ET-0023 --title "Infrastructure Security Summary" --file /tmp/infra_summary.md
# Saved to: evidence/ET-0023_*/2025-Q4/01_infra_summary.md

grctool tool evidence-writer --task-ref ET-0023 --title "Main Terraform Config" --file /path/to/terraform/main.tf
# Saved to: evidence/ET-0023_*/2025-Q4/02_main.tf
```

### Workflow 3: Policy Documentation Evidence

```bash
# 1. Generate context
grctool evidence generate ET-0001 --window 2025-Q4

# 2. Extract policy from Google Docs (for analysis)
grctool tool google-workspace \
  --document-id 1A2B3C4D \
  --document-type docs > /tmp/policy.json

# 3. With Claude Code: analyze policy data and write markdown evidence
# Claude reads the JSON and creates a well-formatted markdown document

# 4. Save markdown evidence and control documentation to root directory
grctool tool evidence-writer --task-ref ET-0001 --title "Policy Summary" --file /tmp/policy_summary.md
# Saved to: evidence/ET-0001_*/2025-Q4/01_policy_summary.md

grctool tool evidence-writer --task-ref ET-0001 --title "Access Control Policy" --file docs/policies/markdown/AC1-access-control.md
# Saved to: evidence/ET-0001_*/2025-Q4/02_ac1_access_control.md
```

---

## Tool Discovery & Help

### List Available Tools
```bash
# Show all tools
grctool tool --help

# List with descriptions
grctool tool list
```

### Get Tool-Specific Help
```bash
# Detailed help for any tool
grctool tool <tool-name> --help

# Examples:
grctool tool github-permissions --help
grctool tool terraform-security-indexer --help
```

### Test Tool Availability
```bash
# Check if tool configuration is valid
grctool tool <tool-name> --dry-run
```

---

**Next Steps**: Consult `evidence-workflow.md` for complete workflow integration.
