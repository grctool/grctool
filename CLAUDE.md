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

**Data Directory**: ./data (configurable in .grctool.yaml)
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

# 4. Review generated evidence
grctool evidence review ET-0001
```

### Tool Discovery
```bash
# List all 29+ available evidence collection tools
grctool tool --help

# Examples of available tools:
grctool tool terraform-security-indexer  # Infrastructure security
grctool tool github-permissions          # Access controls
grctool tool github-workflow-analyzer    # CI/CD security
grctool tool atmos-stack-analyzer        # Multi-env analysis
```

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
- **Google Workspace Tools** - Document evidence from Drive, Docs, Sheets, Forms
- **Atmos Tools** - Multi-environment stack analysis

## 🔍 COMMON USER QUESTIONS

### "What evidence do I need to collect?"
```bash
grctool tool evidence-task-list
```
This shows all pending evidence tasks with status and assignees.

### "What controls apply to [some system]?"
```bash
# Search through synced controls in your data directory
grep -r "keyword" ./data/docs/controls/

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

### "How do I collect evidence from Google Workspace?"
```bash
# First, ensure you have Google Workspace authentication configured
# See: docs/01-User-Guide/google-workspace-setup.md for setup instructions

# Extract policy documents from a Drive folder
grctool tool google-workspace --document-id 1A2B3C4D5E6F7G8H9I0J --document-type drive

# Extract content from a Google Doc (policy, procedure)
grctool tool google-workspace --document-id 1K2L3M4N5O6P7Q8R9S0T --document-type docs

# Extract data from a Google Sheet (access reviews, training records)
grctool tool google-workspace --document-id 1U2V3W4X5Y6Z7A8B9C0D --document-type sheets --sheet-range "Sheet1!A1:D50"

# Extract responses from a Google Form (training quiz, security questionnaire)
grctool tool google-workspace --document-id 1E2F3G4H5I6J7K8L9M0N --document-type forms
```

## 🔐 AUTHENTICATION

### Tugboat Logic Authentication

GRCTool uses **browser-based authentication** with Tugboat Logic:

```bash
# Login (opens your default browser, saves credentials securely)
grctool auth login

# Check status
grctool auth status

# Logout (clears credentials)
grctool auth logout
```

**Note**: Credentials are stored securely in the auth directory and are automatically refreshed.

### Google Workspace Authentication

Google Workspace tools use **service account authentication** with JSON credentials:

**Setup Steps:**
1. Create Google Cloud project and enable APIs (Drive, Docs, Sheets, Forms)
2. Create service account with appropriate permissions
3. Download service account JSON credentials
4. Set environment variable or use explicit path

**Quick Setup:**
```bash
# Set environment variable (recommended)
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/google-credentials.json"

# Or use explicit path in commands
grctool tool google-workspace --credentials-path /path/to/google-credentials.json --document-id <ID>
```

**Important:** The service account email must have Viewer permission on the documents you want to access. Share documents with the service account email found in your credentials file.

**Detailed Setup Guide:** See `docs/01-User-Guide/google-workspace-setup.md` for complete step-by-step instructions.

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
# I can read: ./data/docs/evidence_tasks/ET-0047-*.json

# 3. Run the appropriate tool
grctool tool github-permissions --repository org/repo --output-format matrix

# 4. Review and format the output for compliance
```

### Example: Helping with Google Workspace Evidence (Policy Documentation)

```bash
# User asks: "I need to collect evidence of our security policies from Google Docs"

# 1. Verify Google Workspace authentication is configured
echo $GOOGLE_APPLICATION_CREDENTIALS
# If not set, guide user through: docs/01-User-Guide/google-workspace-setup.md

# 2. Identify the policy folder/document IDs
# User provides: "Our policies are in this folder: https://drive.google.com/drive/folders/1A2B3C4D5E6F7G8H9I0J"
# Extract folder ID: 1A2B3C4D5E6F7G8H9I0J

# 3. Extract folder contents to see what policies exist
grctool tool google-workspace \
  --document-id 1A2B3C4D5E6F7G8H9I0J \
  --document-type drive

# 4. For each policy document, extract full content
# Example: Information Security Policy ID is 1K2L3M4N5O6P7Q8R9S0T
grctool tool google-workspace \
  --document-id 1K2L3M4N5O6P7Q8R9S0T \
  --document-type docs \
  > evidence/information-security-policy.json

# 5. If collecting access reviews from a spreadsheet
grctool tool google-workspace \
  --document-id 1U2V3W4X5Y6Z7A8B9C0D \
  --document-type sheets \
  --sheet-range "Q3 2025!A1:F100" \
  > evidence/access-review-q3-2025.json

# 6. If collecting training quiz responses
grctool tool google-workspace \
  --document-id 1E2F3G4H5I6J7K8L9M0N \
  --document-type forms \
  > evidence/security-training-completions.json
```

**Troubleshooting Tips for AI Assistants:**

When users encounter Google Workspace authentication issues:
1. **"credentials not found"** → Check `GOOGLE_APPLICATION_CREDENTIALS` is set
2. **"403 Forbidden"** → Verify document is shared with service account email
3. **"404 Not Found"** → Confirm document ID is correct (check URL)
4. **"API not enabled"** → User needs to enable APIs in Google Cloud Console

To help users find document IDs:
- **Drive folder**: `https://drive.google.com/drive/folders/FOLDER_ID_HERE`
- **Google Docs**: `https://docs.google.com/document/d/DOCUMENT_ID_HERE/edit`
- **Google Sheets**: `https://docs.google.com/spreadsheets/d/SHEET_ID_HERE/edit`
- **Google Forms**: `https://docs.google.com/forms/d/FORM_ID_HERE/edit`

## 📤 EVIDENCE SUBMISSION

### Tugboat Custom Evidence Integration API

GRCTool uses the **Custom Evidence Integration API** to submit evidence to Tugboat Logic.
**API Documentation**: https://support.tugboatlogic.com/hc/en-us/articles/360049620392

### Setup Requirements

**1. Generate credentials in Tugboat UI:**
   - Navigate to: Integrations > Custom Integrations
   - Click **+** to add a new integration
   - Enter account name and description
   - Click **Generate Password**
   - **IMPORTANT**: Copy and save the Username, Password, and X-API-KEY (cannot be recovered later)

**2. Generate collector URLs for each evidence task:**
   - Click **+** to configure a new evidence service
   - Select scope and evidence task
   - Click **Copy URL** - this is your collector URL
   - Repeat for each evidence task you want to submit

**3. Configure GRCTool:**

Add to `.grctool.yaml`:
```yaml
tugboat:
  username: "your-username"  # From step 1
  password: "your-password"  # From step 1
  collector_urls:
    "ET-0001": "https://openapi.tugboatlogic.com/api/v0/evidence/collector/805/"
    "ET-0047": "https://openapi.tugboatlogic.com/api/v0/evidence/collector/806/"
    # Add more task -> URL mappings
```

**4. Set API Key environment variable:**
```bash
# Direct environment variable
export TUGBOAT_API_KEY="your-x-api-key-from-step-1"

# OR use 1Password (recommended for security)
# Store in 1Password as TUGBOAT_API_KEY
op run --env-file=".env.tugboat" -- grctool evidence submit ET-0001
```

### Evidence Submission Workflow

```bash
# 1. Collect evidence using tools
grctool tool github-permissions --repository org/repo > data/evidence/ET-0047/2025-Q4/github-access.csv

# 2. Validate evidence before submission
grctool evidence validate ET-0047 --window 2025-Q4

# 3. Preview submission (dry-run)
grctool evidence submit ET-0047 --window 2025-Q4 --dry-run

# 4. Submit evidence to Tugboat
grctool evidence submit ET-0047 --window 2025-Q4 --notes "Q4 quarterly review"

# 5. With 1Password integration
op run --env-file=".env.tugboat" -- grctool evidence submit ET-0047 --window 2025-Q4
```

### Supported File Types

**Supported**: txt, csv, json, pdf, png, gif, jpg, jpeg, odt, ods, xls
**Max Size**: 20MB per file
**Not Supported**: html, htm, js, exe, php, etc.

### Troubleshooting Evidence Submission

When users encounter submission issues:

1. **"TUGBOAT_API_KEY not set"** → Check environment variable is exported
2. **"credentials not configured"** → Verify username/password in .grctool.yaml
3. **"collector URL not configured"** → Add task mapping to tugboat.collector_urls
4. **"file extension not supported"** → Check file type is in supported list
5. **"file size exceeds maximum"** → File must be under 20MB
6. **"401 Unauthorized"** → Check username/password are correct
7. **"403 Forbidden"** → Verify API key is correct and has permissions

### Example: Complete Submission Workflow

```bash
# User asks: "How do I submit GitHub access controls evidence for ET-0047?"

# 1. Check if task is configured
grep "ET-0047" .grctool.yaml
# If not found, guide user to add collector URL to config

# 2. Collect the evidence
grctool tool github-permissions \
  --repository acme/infrastructure \
  --output-format csv \
  > data/evidence/ET-0047_GitHub_Access/2025-Q4/github-permissions.csv

# 3. Validate evidence
grctool evidence validate ET-0047 --window 2025-Q4
# Check output for any validation errors

# 4. Set API key and submit
export TUGBOAT_API_KEY="762bbc8a-0363-11eb-ae9b-0e8ffbd46778-org-id-11455"
grctool evidence submit ET-0047 \
  --window 2025-Q4 \
  --notes "GitHub team permissions audit for Q4 2025"

# Success! Evidence submitted to Tugboat Logic
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
- [ ] For Google Workspace: Check `GOOGLE_APPLICATION_CREDENTIALS` is set
- [ ] For Google Workspace: Verify documents are shared with service account
- [ ] Read evidence task files to understand requirements
- [ ] Suggest appropriate tools for evidence collection
- [ ] Help interpret tool output for compliance purposes
- [ ] Ensure evidence is properly documented and formatted
- [ ] Never commit secrets, tokens, or credentials
- [ ] Guide users to setup docs if authentication is not configured

---

**Configuration Details**
- Organization ID: Set via `grctool init` or in .grctool.yaml
- Data Directory: Configurable (default: ./data)
- Authentication: Browser-based (uses default browser)
- Tools Available: 29+ evidence collection tools

**Last Updated**: Generated by `grctool init`
**Regenerate**: Run `grctool init` anytime to update this file with current configuration
