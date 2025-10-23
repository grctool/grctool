---
title: "CLI Commands Reference"
type: "reference"
category: "cli"
tags: ["cli", "commands", "reference", "api", "usage"]
related: ["[[api-documentation]]", "[[data-formats]]", "[[naming-conventions]]"]
created: 2025-09-10
modified: 2025-09-10
status: "active"
---

# CLI Commands Reference

## Overview

GRCTool provides a comprehensive command-line interface for security compliance automation, evidence collection, and audit preparation. All commands follow consistent patterns for authentication, configuration, output formatting, and error handling.

## Global Flags

Available across all commands:

```bash
--config string           # Config file (searches $PWD then $HOME for .grctool.yaml)
--log-file string         # Trace log file location (default "grctool.log")
--log-file-level string   # Log level for file output (default "trace")
--log-level string        # Log level (trace, debug, info, warn, error) (default "info")
--no-log-file            # Disable trace logging to file
--verbose                # Verbose output
-h, --help               # Help for any command
```

## Shell Completion

GRCTool provides intelligent tab completion for all major shells, including **smart completion of task/policy/control IDs** from your synced data.

### Setup Instructions

#### Zsh (macOS default)
```bash
# One-time setup - install completion script
grctool completion zsh > $(brew --prefix)/share/zsh/site-functions/_grctool

# Start a new shell session
exec zsh
```

#### Bash (requires bash-completion package)
```bash
# Install bash-completion if needed
brew install bash-completion@2  # macOS
# OR
apt-get install bash-completion # Linux

# One-time setup - install completion script
grctool completion bash > $(brew --prefix)/etc/bash_completion.d/grctool  # macOS
# OR
grctool completion bash > /etc/bash_completion.d/grctool  # Linux

# Start a new shell session
exec bash
```

#### Fish
```bash
# One-time setup
grctool completion fish > ~/.config/fish/completions/grctool.fish

# Reload fish completions
source ~/.config/fish/completions/grctool.fish
```

#### PowerShell
```powershell
# See detailed instructions
grctool completion powershell --help
```

### What Gets Completed

The completion system provides context-aware suggestions for:

**Commands and Subcommands:**
- All main commands: `auth`, `sync`, `evidence`, `tool`, `config`, etc.
- All subcommands: `evidence generate`, `tool terraform-scanner`, etc.

**Flags and Options:**
- Global flags: `--config`, `--verbose`, `--log-level`, etc.
- Command-specific flags: `--task-ref`, `--framework`, `--output`, etc.

**Smart ID Completion** (reads from synced data):
- **Evidence Task IDs**: `ET-001`, `ET-002`, `ET-101`, etc.
- **Policy IDs**: `POL-001`, `POL-002`, etc.
- **Control IDs**: `AC-01`, `CC-01`, `SO-19`, etc.

### Usage Examples

```bash
# Complete evidence task IDs from synced data
grctool tool evidence-task-details --task-ref ET-<TAB>
# Shows: ET-001  ET-002  ET-003  ET-004  ET-005  ET-101  ...

# Complete evidence command arguments
grctool evidence generate ET-<TAB>
# Shows: ET-001  ET-002  ET-003  ...

# Complete tool names
grctool tool terr<TAB>
# Shows: terraform-scanner  terraform-hcl-parser  terraform-security-analyzer  ...

# Partial matching works
grctool tool github-<TAB>
# Shows: github-permissions  github-security-features  github-workflow-analyzer  ...
```

### Smart Completion Features

**Data-Driven Completions:**
- Task/policy/control IDs are read from your synced data directory
- Completions update automatically when you run `grctool sync`
- Shows helpful descriptions next to IDs (e.g., `ET-001    Access Registration`)

**Graceful Handling:**
- If data hasn't been synced yet, completion still works for commands/flags
- No errors or delays if sync data is missing
- Fast performance even with hundreds of tasks

**Prerequisites for ID Completion:**
```bash
# First time setup: sync data from Tugboat Logic
grctool auth login
grctool sync --full

# Now tab completion will show your actual task/policy/control IDs
grctool tool evidence-task-details --task-ref <TAB>
```

## Core Commands

### Authentication Commands

#### `grctool auth`
Manage Tugboat Logic authentication using browser-based flow.

```bash
# Login to Tugboat Logic
grctool auth login
grctool auth login --org-id 12345

# Check authentication status
grctool auth status

# Logout (clear stored credentials)
grctool auth logout

# Refresh expired credentials
grctool auth refresh
```

**Options:**
- `--org-id`: Specify organization ID for multi-tenant setups
- `--browser`: Browser to use for authentication (default: auto-detect)

**Output:**
```json
{
  "authenticated": true,
  "org_id": 12345,
  "user_email": "user@example.com",
  "expires_at": "2025-09-10T14:30:00Z",
  "permissions": ["read", "write"]
}
```

### Configuration Commands

#### `grctool config`
Configuration management and validation.

```bash
# Validate current configuration
grctool config validate

# Show current configuration (redacted)
grctool config show

# Set configuration values
grctool config set storage.data_dir ./custom-data

# Initialize default configuration
grctool config init
```

**Options:**
- `--check-connectivity`: Test connections to external services
- `--check-permissions`: Validate file system permissions
- `--output-format`: json, yaml, table (default: table)

### Data Synchronization Commands

#### `grctool sync`
Synchronize data from Tugboat Logic API.

```bash
# Full synchronization (all data types)
grctool sync --full

# Sync specific data types
grctool sync --type policies
grctool sync --type controls
grctool sync --type evidence-tasks

# Incremental sync (changes since last sync)
grctool sync --incremental

# Force sync (ignore cached data)
grctool sync --force
```

**Options:**
- `--full`: Sync all data types
- `--incremental`: Only sync changes since last update  
- `--force`: Ignore cache and force full refresh
- `--type string`: Specific data type to sync
- `--batch-size int`: API request batch size (default: 100)
- `--max-retries int`: Maximum retry attempts (default: 3)

**Output:**
```json
{
  "sync_started": "2025-09-10T10:00:00Z",
  "sync_completed": "2025-09-10T10:05:23Z",
  "duration_ms": 323000,
  "results": {
    "policies": {"synced": 25, "updated": 3, "errors": 0},
    "controls": {"synced": 156, "updated": 12, "errors": 0},
    "evidence_tasks": {"synced": 105, "updated": 8, "errors": 0}
  },
  "next_sync_recommended": "2025-09-11T10:00:00Z"
}
```

## Evidence Management Commands

### Evidence Tasks

#### `grctool evidence`
Core evidence management functionality.

```bash
# List all evidence tasks
grctool evidence list

# List with filtering
grctool evidence list --status pending --framework soc2

# Generate evidence for specific task
grctool evidence generate --task-ref ET-0001

# Generate evidence for all automated tasks
grctool evidence generate --all

# Generate evidence for specific framework
grctool evidence generate --framework soc2 --output-dir ./soc2-evidence

# Validate evidence completeness
grctool evidence validate --task-ref ET-0001
grctool evidence validate --all --framework iso27001
```

**Evidence List Options:**
- `--status`: Filter by status (pending, completed, overdue)
- `--framework`: Filter by compliance framework (soc2, iso27001)
- `--assignee`: Filter by assignee
- `--due-before`: Filter by due date
- `--output-format`: json, table, csv (default: table)

**Evidence Generate Options:**
- `--task-ref`: Specific evidence task reference (ET-0001, etc.)
- `--all`: Generate evidence for all automated tasks
- `--framework`: Generate evidence for specific framework
- `--output-dir`: Directory for evidence files
- `--force`: Regenerate even if current evidence exists
- `--parallel`: Enable parallel generation (use with --all)

### Policy Management

#### `grctool policy`
Manage and view organizational policies.

```bash
# List all policies
grctool policy list

# View specific policy details
grctool policy show --policy-id POL-0001

# Search policies by content
grctool policy search --query "information security"

# Generate policy summary for evidence task
grctool policy summary --task-ref ET-0090
```

**Options:**
- `--policy-id`: Specific policy reference (POL-0001, etc.)
- `--query`: Search term for content search
- `--task-ref`: Generate policy summary for evidence task
- `--output-format`: json, markdown, table (default: table)

### Control Management

#### `grctool control`
Manage and view security controls.

```bash
# List all controls
grctool control list

# List controls for specific framework
grctool control list --framework iso27001

# View specific control details  
grctool control show --control-id AC-01

# Show control relationships
grctool control relationships --control-id AC-01
```

**Options:**
- `--control-id`: Specific control reference (AC-01, CC-01_1, etc.)
- `--framework`: Filter by framework (soc2, iso27001)
- `--category`: Filter by control category
- `--implementation-status`: Filter by implementation status

## Tool Commands

### `grctool tool`
Evidence assembly tools with standardized JSON output.

#### Tool Discovery
```bash
# List all available tools
grctool tool list

# Show tool registry statistics  
grctool tool stats

# Get help for specific tool
grctool tool terraform-scanner --help
```

#### Infrastructure Analysis Tools

**terraform-scanner**: Enhanced Terraform configuration scanner
```bash
# Basic security scan
grctool tool terraform-scanner --path ./infrastructure

# Scan specific resource types
grctool tool terraform-scanner --path ./infrastructure --resource-types aws_security_group,aws_iam_role

# Security-focused scan
grctool tool terraform-scanner --path ./infrastructure --focus security,encryption

# Generate evidence for specific task
grctool tool terraform-scanner --path ./infrastructure --task-ref ET-0011
```

**terraform-hcl-parser**: Comprehensive HCL parser with topology analysis
```bash
# Parse HCL with focus areas
grctool tool terraform-hcl-parser --path ./infrastructure --focus network-security,monitoring

# Include resource relationships
grctool tool terraform-hcl-parser --path ./infrastructure --include-relationships

# Compliance-focused analysis
grctool tool terraform-hcl-parser --path ./infrastructure --compliance iso27001
```

#### GitHub Analysis Tools

**github-permissions**: Repository access controls and permissions
```bash
# Comprehensive permissions analysis
grctool tool github-permissions --repository org/repo

# Include team and collaborator details
grctool tool github-permissions --repository org/repo --include-collaborators --include-teams

# Admin-only analysis
grctool tool github-permissions --repository org/repo --admin-only

# Generate evidence for access control task
grctool tool github-permissions --repository org/repo --task-ref ET-0001
```

**github-security-features**: Repository security configuration
```bash
# Complete security feature audit
grctool tool github-security-features --repository org/repo

# Include code scanning results
grctool tool github-security-features --repository org/repo --include-code-scanning

# Security policy and vulnerability focus
grctool tool github-security-features --repository org/repo --focus security-policies,vulnerabilities
```

**github-workflow-analyzer**: GitHub Actions workflows analysis
```bash
# Analyze all workflows
grctool tool github-workflow-analyzer --repository org/repo

# Security-focused workflow analysis
grctool tool github-workflow-analyzer --repository org/repo --workflow-type security

# Deployment workflow analysis
grctool tool github-workflow-analyzer --repository org/repo --workflow-type deployment --include-approvals
```

**github-deployment-access**: Deployment environment access controls
```bash
# Analyze deployment environments
grctool tool github-deployment-access --repository org/repo

# Specific environment analysis
grctool tool github-deployment-access --repository org/repo --environment production

# Include protection rules
grctool tool github-deployment-access --repository org/repo --include-protection-rules
```

**github-review-analyzer**: Pull request review and approval processes
```bash
# Analyze review processes
grctool tool github-review-analyzer --repository org/repo

# Include security-focused reviews
grctool tool github-review-analyzer --repository org/repo --include-security-reviews

# Time period analysis
grctool tool github-review-analyzer --repository org/repo --since 2025-01-01 --until 2025-03-31
```

#### Document Management Tools

**google-workspace**: Google Workspace document analysis
```bash
# Policy document collection
grctool tool google-workspace --document-type policies

# Search for specific content
grctool tool google-workspace --search-term "information security policy"

# Folder-specific analysis
grctool tool google-workspace --folder-id folder-123456 --include-metadata

# Training materials collection
grctool tool google-workspace --document-type training --include-completion-data
```

#### Evidence Management Tools

**evidence-task-list**: List evidence tasks with filtering
```bash
# List all evidence tasks
grctool tool evidence-task-list

# Filter by framework
grctool tool evidence-task-list --framework soc2

# Filter by status
grctool tool evidence-task-list --status pending --due-within 30

# Output as IDs only for scripting
grctool tool evidence-task-list --format ids-only --filter automated
```

**evidence-task-details**: Retrieve detailed evidence task information
```bash
# Get task details
grctool tool evidence-task-details --task-ref ET-0001

# Include embedded data
grctool tool evidence-task-details --task-ref ET-0001 --include-embeds

# Generate with context for AI processing
grctool tool evidence-task-details --task-ref ET-0001 --ai-context
```

**evidence-generator**: Generate evidence from multiple sources
```bash
# Generate evidence for specific task
grctool tool evidence-generator --task-ref ET-0001

# Multi-source evidence generation
grctool tool evidence-generator --task-ref ET-0001 --sources terraform,github,policies

# Generate with specific output format
grctool tool evidence-generator --task-ref ET-0001 --output-format audit-ready
```

**evidence-validator**: Validate evidence completeness and quality
```bash
# Validate specific evidence
grctool tool evidence-validator --evidence-file ./evidence/ET-0001.json

# Validate evidence directory
grctool tool evidence-validator --evidence-dir ./evidence --framework soc2

# Quality score assessment
grctool tool evidence-validator --evidence-dir ./evidence --quality-score --min-score 85

# Check for gaps
grctool tool evidence-validator --evidence-dir ./evidence --check-gaps --output-format gap-report
```

#### Utility Tools

**storage-read**: Safe file read operations
```bash
# Read file with safety checks
grctool tool storage-read --file-path ./data/policies.json

# Read with content validation
grctool tool storage-read --file-path ./data/evidence.json --validate-json
```

**storage-write**: Safe file write operations
```bash
# Write with atomic operations
grctool tool storage-write --file-path ./output/evidence.json --content '{...}' --atomic

# Write with backup
grctool tool storage-write --file-path ./data/config.yaml --content '...' --backup
```

**name-generator**: Generate filesystem-friendly names
```bash
# Generate name for evidence task
grctool tool name-generator --document-type evidence --reference-id ET-0001

# Generate name for policy document
grctool tool name-generator --document-type policy --reference-id POL-0001 --tugboat-id 94641
```

### Data Validation Commands

#### `grctool validate-data`
Comprehensive data validation and quality checks.

```bash
# Full data validation
grctool validate-data

# Check data completeness
grctool validate-data --check-completeness

# Validate specific data types
grctool validate-data --type policies --check-integrity

# Framework-specific validation
grctool validate-data --framework soc2 --check-evidence-mapping

# Generate validation report
grctool validate-data --output-format report --output-file validation-report.json
```

**Options:**
- `--check-completeness`: Verify all required data is present
- `--check-integrity`: Validate data structure and relationships
- `--check-evidence-mapping`: Verify evidence task mappings
- `--type`: Specific data type to validate
- `--framework`: Framework-specific validation rules
- `--fix-issues`: Attempt to fix detected issues automatically

## Output Formats

### Standard Output Formats
All commands support consistent output formatting:

```bash
# JSON output (default for tools)
grctool tool terraform-scanner --output json

# Table output (default for lists)
grctool evidence list --output table

# CSV output (for data export)
grctool evidence list --output csv

# YAML output (for configuration)
grctool config show --output yaml
```

### Tool-Specific Output Formats

**Evidence Tools:**
- `audit-ready`: Formatted for auditor consumption
- `executive-summary`: High-level summary for management
- `technical-details`: Detailed technical information
- `compliance-matrix`: Control-to-evidence mapping

**Analysis Tools:**
- `security-report`: Security-focused analysis
- `compliance-check`: Compliance validation results
- `gap-analysis`: Identified gaps and recommendations
- `risk-assessment`: Risk-based analysis

### Quiet Mode
Compact JSON output for programmatic consumption:

```bash
# Compact JSON output
grctool tool terraform-scanner --quiet

# Standard pretty JSON output (default)
grctool tool terraform-scanner
```

## Common Usage Patterns

### Audit Preparation Workflow
```bash
# 1. Ensure authentication and sync
grctool auth status && grctool sync --incremental

# 2. Generate evidence for all automated tasks
grctool evidence generate --all --output-dir ./audit-evidence

# 3. Validate evidence completeness
grctool evidence validate --all --framework soc2

# 4. Generate evidence summary
grctool tool evidence-generator --task-ref all --output-format executive-summary
```

### Daily Evidence Collection
```bash
# Automated evidence collection script
#!/bin/bash
set -e

# Update data
grctool sync --incremental

# Collect infrastructure evidence
grctool tool terraform-scanner --path ./infrastructure --output-dir ./daily-evidence/$(date +%Y-%m-%d)/

# Collect access control evidence
for repo in production-api frontend-app admin-dashboard; do
  grctool tool github-permissions --repository myorg/$repo --output-dir ./daily-evidence/$(date +%Y-%m-%d)/$repo/
done

# Generate daily summary
grctool tool evidence-generator --sources infrastructure,github --output-format daily-summary
```

### Security Review Workflow
```bash
# Comprehensive security review
grctool tool terraform-scanner --focus security,encryption --output-dir ./security-review/
grctool tool github-security-features --repository myorg/production-api --include-code-scanning
grctool tool github-workflow-analyzer --repository myorg/production-api --workflow-type security

# Generate security summary
grctool tool evidence-generator --sources terraform,github --focus security --output-format security-report
```

## Error Handling

### Exit Codes
- `0`: Success
- `1`: General error
- `2`: Command line argument error
- `3`: Configuration error
- `4`: Authentication error
- `5`: Network/API error
- `6`: File system error
- `7`: Data validation error

### Common Error Patterns
```bash
# Authentication required
grctool tool terraform-scanner
# Error: authentication required. Run 'grctool auth login' first.

# Invalid task reference
grctool evidence generate --task-ref ET-9999
# Error: evidence task ET-9999 not found. Use 'grctool evidence list' to see available tasks.

# Missing required parameters
grctool tool github-permissions
# Error: --repository flag is required

# Path safety validation
grctool tool terraform-scanner --path ../../../etc/passwd
# Error: path traversal detected. Paths must be within current directory or explicitly allowed.
```

## Performance Considerations

### Large Dataset Handling
```bash
# Use batch processing for large operations
grctool evidence generate --all --batch-size 10 --parallel

# Limit resource usage
grctool tool terraform-scanner --max-files 1000 --timeout 300s

# Use incremental operations where possible
grctool sync --incremental --since 2025-09-01
```

### Concurrent Execution
```bash
# Safe parallel evidence generation
grctool evidence generate --all --parallel --max-workers 4

# Individual tool concurrent execution
grctool tool terraform-scanner --parallel --max-workers 8
```

## Security Features

### Path Safety
All file operations include path traversal protection:
```bash
# Safe paths (allowed)
grctool tool terraform-scanner --path ./infrastructure
grctool tool terraform-scanner --path /home/user/project/terraform

# Unsafe paths (blocked)
grctool tool terraform-scanner --path ../../../etc
grctool tool terraform-scanner --path ~/.ssh
```

### Credential Security
- No API keys stored in configuration files
- Browser-based authentication flow
- Automatic credential expiration and refresh
- Secure credential storage using OS keychain
- Audit trail for all authentication events

### Data Redaction
Automatic redaction of sensitive data in logs and outputs:
- API keys and tokens
- Authorization headers
- Cookie values
- Private keys and certificates
- Personal identifiable information (when detected)

## Integration Examples

### Shell Scripts
```bash
#!/bin/bash
# Evidence collection automation

# Check authentication
if ! grctool auth status --quiet >/dev/null 2>&1; then
    echo "Authentication required"
    grctool auth login
fi

# Collect evidence for specific tasks
TASKS="ET-0001 ET-0011 ET-0030 ET-0070"
for task in $TASKS; do
    echo "Collecting evidence for $task"
    grctool evidence generate --task-ref $task --output-dir "./evidence/$task/"
done

# Generate summary
grctool tool evidence-generator --task-ref all --output-format summary > evidence-summary.json
```

### CI/CD Integration
```yaml
# GitHub Actions example
name: Evidence Collection
on:
  schedule:
    - cron: '0 2 * * *'  # Daily at 2 AM

jobs:
  collect-evidence:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Install GRCTool
      run: |
        curl -L https://github.com/org/grctool/releases/latest/download/grctool-linux-amd64 -o grctool
        chmod +x grctool
        
    - name: Authenticate
      run: ./grctool auth login --non-interactive
      env:
        TUGBOAT_AUTH_TOKEN: ${{ secrets.TUGBOAT_AUTH_TOKEN }}
        
    - name: Collect Infrastructure Evidence
      run: ./grctool tool terraform-scanner --path ./infrastructure --output-dir ./evidence/
      
    - name: Upload Evidence
      uses: actions/upload-artifact@v3
      with:
        name: evidence-$(date +%Y-%m-%d)
        path: ./evidence/
```

## Troubleshooting

### Common Issues

**Authentication Failures:**
```bash
# Check current auth status
grctool auth status --verbose

# Clear and re-authenticate
grctool auth logout && grctool auth login

# Check network connectivity
grctool config validate --check-connectivity
```

**Performance Issues:**
```bash
# Enable verbose logging for diagnostics
grctool --verbose tool terraform-scanner --path ./infrastructure

# Check log files for detailed information
tail -f grctool.log

# Use smaller batch sizes
grctool sync --batch-size 50 --max-retries 5
```

**Data Validation Errors:**
```bash
# Check data integrity
grctool validate-data --check-integrity --verbose

# Force refresh of problematic data
grctool sync --force --type evidence-tasks

# Validate specific evidence
grctool tool evidence-validator --evidence-file ./evidence/ET-0001.json --verbose
```

## References

- [[api-documentation]] - Internal API documentation
- [[data-formats]] - JSON schemas and file formats  
- [[naming-conventions]] - Reference ID standards and naming patterns
- [[glossary]] - Terms and definitions
- [[helix/05-deploy/deployment-operations|Configuration Guide]] - Detailed configuration options
- [[helix/02-design/security-architecture|Authentication Guide]] - Authentication setup and troubleshooting

---

*This reference is automatically updated from CLI help text and usage patterns. Last updated: 2025-09-10*