# CLAUDE.md - AI Assistant Guide for GRCTool Users

## 🎯 PROJECT OVERVIEW

This project uses **GRCTool** - an automated compliance evidence collection CLI tool.

**What is GRCTool?**
- Automates evidence collection for SOC2, ISO27001, and other compliance frameworks
- Syncs policies, controls, and evidence tasks from Tugboat Logic
- Integrates with infrastructure (Terraform, GitHub, Google Workspace)
- Generates compliance evidence using AI and automated scanning

**Your Role as an AI Assistant:**
Help users navigate their compliance program, generate evidence, and understand their security controls.

## 📁 PROJECT STRUCTURE

This project is organized as follows:

**Data Directory**: /Users/erik/Projects/7thsense-ops/isms
- **docs/** - Synced data from Tugboat Logic
  - **policies/** - Policy documents and metadata (JSON/Markdown)
  - **controls/** - Security controls and requirements (JSON/Markdown)
  - **evidence_tasks/** - Evidence collection tasks (JSON)
- **evidence/** - Generated evidence files organized by task
- **.cache/** - Performance cache (can be safely deleted)

## 🔧 COMMON COMMANDS

### Initial Setup
```bash
# 1. Initialize configuration (safe to run multiple times)
grctool init

# 2. Authenticate with Tugboat Logic (browser-based)
grctool auth login

# 3. Verify authentication
grctool auth status
```

### Data Synchronization
```bash
# Sync latest data from Tugboat Logic
grctool sync

# This downloads:
# - Policies (governance documents)
# - Controls (security requirements)
# - Evidence Tasks (what evidence needs to be collected)
```

### Evidence Collection Workflow
```bash
# 1. List all evidence tasks
grctool tool evidence-task-list

# 2. Get details about a specific task
grctool tool evidence-task-details --task-ref ET-0001

# 3. Generate evidence for a task
grctool evidence generate ET-0001

# 4. Evaluate generated evidence
grctool evidence evaluate ET-0001

# 5. Review generated evidence (human-readable assessment)
grctool evidence review ET-0001

# 6. Setup collector URL for submission (one-time per task)
grctool evidence setup ET-0001 --collector-url "https://openapi.tugboatlogic.com/..."

# 7. Submit evidence to Tugboat
grctool evidence submit ET-0001
```

### Tool Discovery
```bash
# List all 30 available evidence collection tools
grctool tool --help

# Examples of available tools:
grctool tool terraform-security-indexer  # Indexes and queries Terraform infrastructure re...
grctool tool github-permissions  # Extract comprehensive repository access control...
grctool tool github-workflow-analyzer  # Analyze GitHub Actions workflows for CI/CD secu...
```

### Available Tools by Category

**Evidence Analysis Tools:**
- `control-summary-generator` - Generate focused control summaries for evidence tasks using template-based approach (prompt as data pattern)
- `evidence-relationships` - Maps relationships between evidence tasks, controls, and policies with configurable depth analysis
- `evidence-task-details` - Retrieves detailed information about evidence tasks including requirements, status, and metadata
- `evidence-task-list` - List evidence tasks with filtering capabilities for programmatic access
- `policy-summary-generator` - Generate focused policy summaries for evidence tasks using template-based approach (prompt as data pattern)
- `prompt-assembler` - Generates comprehensive prompts for evidence collection with context and examples (template-based, no AI API calls)


**Data Source Tools:**
- `docs-reader` - Search and analyze documentation files with keyword relevance scoring and section extraction
- `github-deployment-access` - Extract deployment environment access controls and protection rules for SOC2 audit evidence
- `github-enhanced` - Enhanced GitHub repository searcher with multiple search types, date filtering, and caching
- `github-permissions` - Extract comprehensive repository access controls and permissions for SOC2 audit evidence
- `github-review-analyzer` - Analyze GitHub pull request reviews, approval processes, and code review compliance for SOC2 evidence
- `github-searcher` - Search GitHub repository for security-related issues, pull requests, and discussions
- `github-security-features` - Extract repository security feature configuration for SOC2 audit evidence
- `github-workflow-analyzer` - Analyze GitHub Actions workflows for CI/CD security evidence, deployment controls, and approval processes
- `google-workspace` - Extract evidence from Google Workspace documents including Drive, Docs, Sheets, and Forms
- `terraform-evidence-query` - Intelligent query interface for Claude to find and retrieve Terraform infrastructure evidence for compliance frameworks
- `terraform-hcl-parser` - Comprehensive HCL parser for Terraform configurations with deep structural analysis
- `terraform-security-analyzer` - Comprehensive security configuration analyzer for Terraform manifests with SOC2 control mapping
- `terraform-security-indexer` - Indexes and queries Terraform infrastructure resources by security attributes for compliance evidence collection
- `terraform_analyzer` - Analyzes Terraform configuration files for security, modules, data sources, and compliance
- `terraform_snippets` - Suggests Terraform configuration snippets based on existing patterns and security controls


**Evidence Management Tools:**
- `evidence-generator` - Generate compliance evidence using AI coordination with sub-tools
- `evidence-validator` - Validate evidence completeness and quality with scoring and recommendations
- `evidence-writer` - Write evidence files with window management and collection planning
- `grctool-run` - Execute allowlisted grctool commands with safe flag parsing and structured output capture
- `storage-read` - Path-safe file reading with format auto-detection and metadata preservation
- `storage-write` - Path-safe file writing with format handling and directory management
- `tugboat-sync-wrapper` - Wrapper for tugboat sync with structured output and selective resource syncing


**Utility Tools:**
- `name-generator` - Generates concise, filesystem-friendly names for evidence tasks, controls, and policies using Claude AI


**Other Tools:**
- `atmos-stack-analyzer` - Analyzes Atmos stack configurations and multi-environment Terraform deployments for security compliance



## 🏷️ NAMING CONVENTIONS

Understanding file and reference naming:

- **Evidence Tasks**: `ET-0001`, `ET-0104` (4-digit zero-padded)
- **Policies**: `POL-0001`, `POL-0002` (4-digit zero-padded)
- **Controls**: `AC-01`, `CC-01.1`, `SO-19` (varies by framework)
- **Evidence Files**: `ET-0001-327992-access_registration.json`

## 📊 KEY CONCEPTS

### Policies
High-level governance documents that define "what" the organization does.
Example: "Access Control Policy", "Data Protection Policy"

### Controls
Specific security requirements that implement policies. Define "how" things are done.
Example: "CC6.8 - Logical access controls restrict access to authorized users"

### Evidence Tasks
Specific evidence that must be collected to prove controls are implemented.
Example: "ET-0047 - GitHub Repository Access Controls - Show team permissions"

### Tools
Automated scanners and analyzers that collect evidence from infrastructure:
- **Terraform Tools** (7 tools) - Infrastructure as Code security
- **GitHub Tools** (6 tools) - Repository access, workflows, security features
- **Google Workspace Tools** - User access, groups, drive permissions
- **Atmos Tools** - Multi-environment stack analysis

## 🔍 COMMON USER QUESTIONS

### "What evidence do I need to collect?"
```bash
grctool tool evidence-task-list
```
This shows all pending evidence tasks with status and assignees.

### "What controls apply to [some system]?"
```bash
# Search through synced controls
grep -r "keyword" /Users/erik/Projects/7thsense-ops/isms/docs/controls/

# Or ask me - I can read the control files and explain them
```

### "How do I generate evidence for GitHub access controls?"
```bash
# Use the specialized GitHub permission tool
grctool tool github-permissions --repository owner/repo
```

### "What Terraform security evidence can be collected?"
```bash
# Use the comprehensive indexer for fast queries
grctool tool terraform-security-indexer --query-type control_mapping

# Or use the security analyzer for deep analysis
grctool tool terraform-security-analyzer --security-domain all
```

## 🔐 AUTHENTICATION

GRCTool uses **browser-based authentication** with Tugboat Logic:

```bash
# Login (opens Safari, saves credentials securely)
grctool auth login

# Check status
grctool auth status

# Logout (clears credentials)
grctool auth logout
```

**Note**: Credentials are stored in /auth/ and are automatically refreshed.

## 🔗 EVIDENCE SUBMISSION SETUP

GRCTool can submit evidence directly to Tugboat Logic using custom collector URLs.

### Setting Up Collector URLs

Each evidence task requires a unique collector URL from Tugboat:

```bash
# Configure collector URL (one-time setup per task)
grctool evidence setup ET-0001 --collector-url "https://openapi.tugboatlogic.com/api/v0/evidence/collector/805/"

# Interactive mode (prompts for URL)
grctool evidence setup ET-0001

# Using Tugboat task ID (as shown in Tugboat UI)
grctool evidence setup 327992 --collector-url "https://..."

# Preview changes without modifying config
grctool evidence setup ET-0001 --collector-url "https://..." --dry-run
```

### Getting Collector URLs from Tugboat

1. Log into Tugboat Logic
2. Navigate to: **Custom Integrations** > **Evidence Services**
3. Find your evidence task and click **"Copy URL"**
4. Use the copied URL with `grctool evidence setup`

The collector URL is stored in `.grctool.yaml`:
```yaml
tugboat:
  collector_urls:
    ET-0001: "https://openapi.tugboatlogic.com/api/v0/evidence/collector/805/"
    ET-0002: "https://openapi.tugboatlogic.com/api/v0/evidence/collector/806/"
```

**Note**: Collector URLs are specific to each evidence task and must be configured before submission.

## 🎨 TEMPLATE VARIABLES AND CUSTOMIZATION

GRCTool automatically substitutes template variables in policies and controls with your organization's information.

**What are Template Variables?**
- Placeholders like `{{organization.name}}` in policy documents
- Automatically replaced with your actual organization name in generated evidence
- Found in 408+ places across 40 policy documents
- Ensures consistent branding throughout compliance documentation

**Supported Template Variables:**
- `{{organization.name}}` - Organization name (e.g., "Seventh Sense")
- `{{Organization}}` - Alternative format (legacy support)

**Configuration:**
Template variables are configured in `.grctool.yaml`:

```yaml
interpolation:
  enabled: true  # Default: true
  variables:
    organization:
      name: "Seventh Sense"
```

**How It Works:**
1. **Sync**: Policies and controls are synced from Tugboat Logic with template markers intact
2. **Generation**: When generating evidence or markdown documents, templates are automatically replaced
3. **Consistency**: All generated evidence uses your configured organization name

**Adding Custom Variables:**
You can add custom template variables to the configuration:

```yaml
interpolation:
  enabled: true
  variables:
    organization:
      name: "Seventh Sense"
      legal_name: "Seventh Sense Inc."
      security_team: "security@7thsense.ai"
    custom:
      compliance_contact: "compliance@7thsense.ai"
```

Then use them in custom evidence prompts: `{{organization.security_team}}`

**Note**: Template substitution is enabled by default. All synced policy and control documents will automatically use your configured values when generating evidence or documentation.

## 🎯 HELPING USERS WITH EVIDENCE

When a user asks for help with evidence collection:

1. **Understand the task**: Read the evidence task JSON file
2. **Identify applicable tools**: Suggest which grctool tools can collect this evidence
3. **Review existing evidence**: Check if evidence already exists
4. **Guide evidence generation**: Walk through tool usage
5. **Help with formatting**: Ensure evidence meets auditor requirements

### Example: Helping with ET-0047 (GitHub Repository Access)

```bash
# 1. Get task details
grctool tool evidence-task-details --task-ref ET-0047

# 2. Check what controls it maps to
# I can read: /Users/erik/Projects/7thsense-ops/isms/docs/evidence_tasks/ET-0047-*.json

# 3. Run the appropriate tool
grctool tool github-permissions --repository org/repo --output-format matrix

# 4. Review and format the output for compliance
```

## 📚 GETTING MORE HELP

```bash
# General help
grctool --help

# Command-specific help
grctool sync --help
grctool tool evidence-task-list --help

# Tool-specific help
grctool tool github-permissions --help
```

## ✅ CHECKLIST FOR AI ASSISTANTS

When helping users:
- [ ] Confirm data is synced (`grctool sync`)
- [ ] Verify authentication is valid (`grctool auth status`)
- [ ] Read evidence task files to understand requirements
- [ ] Suggest appropriate tools for evidence collection
- [ ] Help interpret tool output for compliance purposes
- [ ] Ensure evidence is properly documented and formatted
- [ ] Never commit secrets, tokens, or credentials

---

**Configuration Details**
- Organization ID: test
- Data Directory: /Users/erik/Projects/7thsense-ops/isms
- Authentication: Browser-based (Safari)
- Tools Available: 30 evidence collection tools

**Last Updated**: Generated by `grctool init`
**Regenerate**: Run `grctool init` anytime to update this file with current configuration
