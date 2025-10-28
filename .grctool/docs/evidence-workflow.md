# Evidence Generation Workflow

> Complete workflow for generating and managing evidence with Claude Code

---

**Generated**: 2025-10-28 10:33:19 EDT
**GRCTool Version**: dev
**Documentation Version**: dev

---

## Overview

This document describes the complete end-to-end workflow for evidence generation using GRCTool with Claude Code assistance.

### Workflow Philosophy

GRCTool follows a **Claude Code Assisted** approach:
1. GRCTool generates comprehensive context
2. Claude Code reads the context and helps you interactively
3. You run tools together with Claude's guidance
4. Evidence is saved with automatic metadata tracking
5. State is tracked throughout the lifecycle

---

## Complete Evidence Lifecycle

```
┌─────────────────┐
│ grctool status  │  (1) Check what evidence is needed
└────────┬────────┘
         │
         v
┌──────────────────────┐
│ grctool evidence     │  (2) Generate evidence with Claude Code
│   generate ET-XXXX   │
└────────┬─────────────┘
         │
         v
┌──────────────────────┐
│  Run tools together  │  (3) Execute grctool tools with guidance
│  with Claude's help  │      Evidence saved to root directory
└────────┬─────────────┘
         │
         v
┌──────────────────────┐
│ grctool evidence     │  (4) Automated quality scoring
│   evaluate ET-XXXX   │      (Optional but recommended)
└────────┬─────────────┘
         │
         v
┌──────────────────────┐
│ grctool evidence     │  (5) Human review and assessment
│   review ET-XXXX     │      (Optional)
└────────┬─────────────┘
         │
         v
┌──────────────────────┐
│ grctool evidence     │  (6) Setup collector URL
│   setup ET-XXXX      │      (One-time configuration per task)
└────────┬─────────────┘
         │
         v
┌──────────────────────┐
│ grctool evidence     │  (7) Submit to Tugboat
│   submit ET-XXXX     │      Files auto-move to .submitted/
└────────┬─────────────┘
         │
         v
┌──────────────────────┐
│ grctool sync         │  (8) Sync approved evidence from Tugboat
│                      │      Files appear in archive/
└──────────────────────┘
```

**Key Points**:
- **Working Area**: Root directory (e.g., `2025-Q4/01_evidence.md`)
- **Evaluate**: Automated scoring, not validation
- **Review**: Human assessment, separate from evaluate
- **Submit**: Automatically moves files from root to `.submitted/`
- **Sync**: Downloads approved evidence from Tugboat to `archive/`

---

## Tugboat-Managed Evidence Tasks

### Overview

Some evidence tasks are collected directly by Tugboat Logic rather than through grctool automation. These tasks typically involve:
- **Personnel data** (employee lists, terminations, HRIS exports)
- **Manual documents** (signed policies, approvals, contracts)
- **Vendor-provided reports** (audit reports, certifications)

### Detection Criteria

GRCTool automatically detects Tugboat-managed tasks when **both** conditions are met:
1. **AEC Status**: `enabled` (Automated Evidence Collection is enabled)
2. **Collection Type**: `Hybrid` (Mixed automated and manual collection)

### Behavior

When you run `grctool evidence generate` on a Tugboat-managed task:

**Instead of generating files**, grctool will:
- ✅ Display a helpful message explaining that Tugboat collects this evidence
- ✅ Show what data is needed based on task guidance
- ✅ Provide category-specific instructions (Personnel vs Process vs Infrastructure)
- ✅ Include related controls for context
- ✅ Direct you to Tugboat's web interface

**Example Output**:
```
═══════════════════════════════════════════════════════════════════
📋 Task: ET-0002 - Population 2 - List of Employees and Contractors
═══════════════════════════════════════════════════════════════════

ℹ️  This evidence is collected directly in Tugboat Logic.

📝 What's needed:
   System-generated list of employees and contractors throughout
   the audit period including start dates, titles, and termination
   dates (as applicable).

🔗 How to provide this evidence:
   1. Log into Tugboat Logic web interface
   2. Navigate to evidence task ET-0002
   3. Use Tugboat's data upload or integration features to provide:

      • Upload CSV or Excel file with employee/contractor data
      • Required fields: Name, Start Date, Title, Department
      • Include termination dates for separated personnel
      • Use your HRIS system export or generate from HR database

═══════════════════════════════════════════════════════════════════
```

### Why This Matters

**GRCTool automation is designed for infrastructure-sourced evidence** from:
- Terraform configurations
- GitHub repositories
- Google Workspace
- Documentation files

**Tugboat-managed tasks require data from**:
- HRIS systems (Workday, BambooHR, etc.)
- Document management systems
- Manual uploads
- Vendor portals

### Workflow for Tugboat-Managed Tasks

```
┌─────────────────┐
│ grctool evidence│  (1) Run generate command
│   generate ET-XX│
└────────┬────────┘
         │
         v
┌──────────────────────┐
│ Tugboat Detection    │  (2) Detects AEC + Hybrid
└────────┬─────────────┘
         │
         v
┌──────────────────────┐
│  Display Guidance    │  (3) Shows what to upload
│  Skip File Creation  │
└────────┬─────────────┘
         │
         v
┌──────────────────────┐
│ Log into Tugboat     │  (4) Use web interface
│ Upload Data/Files    │
└──────────────────────┘
```

### Common Tugboat-Managed Tasks

| Task Type | Examples | Data Source |
|-----------|----------|-------------|
| **Personnel** | Employee lists, terminations, contractors | HRIS system |
| **Process** | Signed policies, approvals, board minutes | Document management |
| **Vendor** | Audit reports, SOC2 reports, certifications | Vendor portals |
| **Training** | Training completion records, acknowledgments | LMS system |

---

## Step-by-Step Workflow

### Step 1: Check Evidence Status

**Purpose**: See what evidence needs to be collected
**Command**: `grctool status`

```bash
# See dashboard
grctool status

# Check specific task
grctool status task ET-0047

# Filter by state
grctool status --filter state=no_evidence
```

**What You See**:
- Overall progress summary
- Tasks grouped by state (no_evidence, generated, submitted, etc.)
- Automation level for each task
- Next steps recommendations

### Step 2: Generate Evidence

**Purpose**: Generate evidence with Claude Code assistance
**Command**: `grctool evidence generate ET-XXXX --window {window}`

```bash
# Generate evidence for current window
grctool evidence generate ET-0047 --window 2025-Q4
```

**What Happens**:
- Launches interactive evidence generation with Claude Code
- Auto-detects applicable tools from task keywords
- Gathers related controls and policies
- Scans for existing evidence from previous windows
- Claude Code guides you through tool execution
- Evidence is saved directly to root directory (e.g., `2025-Q4/01_evidence.md`)

**Interactive Workflow**:
1. Claude Code reads task requirements
2. Claude suggests which tools to run
3. You execute tools with Claude's guidance
4. Claude helps format and save evidence
5. Files are written to root directory using `evidence-writer`

### Step 3: Execute Tools

**Purpose**: Collect evidence using automated tools with Claude's guidance

**With Claude's Guidance**:
```bash
# Claude will help you run commands like:
grctool tool github-permissions \
  --repository org/repo \
  --output-format json > /tmp/permissions.json

grctool tool github-security-features \
  --repository org/repo > /tmp/security.json

# Note: JSON outputs are for Claude's analysis only
# Findings will be written into markdown evidence documents
```

**Save Evidence to Root Directory**:
```bash
# Save markdown evidence (Claude writes findings from JSON analysis)
grctool tool evidence-writer \
  --task-ref ET-0047 \
  --title "GitHub Permissions Analysis" \
  --file /tmp/permissions_summary.md

# Save original source files for auditor review
grctool tool evidence-writer \
  --task-ref ET-0047 \
  --title "GitHub Workflow Configuration" \
  --file /path/to/.github/workflows/deploy.yml
```

**Where Files Go**:
- Evidence saved to root directory: `evidence/*_ET-0047_*/2025-Q4/01_evidence.md`
- Files automatically numbered (01_, 02_, etc.)
- `.generation/metadata.yaml` created with checksums and timestamps

### Step 4: Evaluate Evidence (Optional but Recommended)

**Purpose**: Automated quality scoring to ensure completeness
**Command**: `grctool evidence evaluate ET-XXXX`

```bash
# Evaluate specific task
grctool evidence evaluate ET-0047 --window 2025-Q4
```

**What It Does**:
- Analyzes evidence quality and completeness
- Generates quality scores (0-100)
- Provides specific recommendations for improvement
- Creates `.validation/validation.yaml` with results

**Note**: This is different from `review` command. Evaluate = automated scoring, Review = human assessment.

### Step 5: Review Evidence (Optional)

**Purpose**: Human assessment and approval
**Command**: `grctool evidence review ET-XXXX`

```bash
# Review evidence interactively
grctool evidence review ET-0047 --window 2025-Q4
```

**What It Does**:
- Allows human reviewer to assess evidence
- Records reviewer comments and approval status
- Separate from automated evaluation

### Step 6: Setup Collector URL (One-Time Configuration)

**Purpose**: Configure the Tugboat collector URL for evidence submission
**Command**: `grctool evidence setup ET-XXXX --collector-url <URL>`

```bash
# Setup collector URL (one-time per task)
grctool evidence setup ET-0047 --collector-url "https://openapi.tugboatlogic.com/api/v0/evidence/collector/805/"

# Interactive mode (prompts for URL)
grctool evidence setup ET-0047

# Using Tugboat task ID (as displayed in Tugboat UI)
grctool evidence setup 327992 --collector-url "https://..."
```

**What It Does**:
- Configures task-specific collector URL from Tugboat UI
- Accepts both ET reference (ET-0047) and numeric task ID (327992)
- Validates URL format and HTTPS requirement
- Updates `.grctool.yaml` configuration file
- Creates backup before modifications

**Getting the Collector URL**:
1. Log into Tugboat Logic
2. Navigate to: Custom Integrations > Evidence Services
3. Find your evidence task and click "Copy URL"

**Note**: This is a one-time setup per task. Once configured, you can submit evidence multiple times without reconfiguring.

### Step 7: Submit to Tugboat

**Purpose**: Upload evidence to Tugboat Logic for auditor review
**Command**: `grctool evidence submit ET-XXXX`

```bash
# Submit evidence
grctool evidence submit ET-0047 \
  --window 2025-Q4 \
  --notes "Q4 quarterly review"
```

**What Happens**:
1. Files uploaded via Tugboat Custom Evidence API
2. **Files automatically move from root to `.submitted/`**
3. `.submitted/.submission/submission.yaml` created with:
   - Submission ID
   - Timestamp
   - Files submitted
   - Tugboat response status
4. Root directory is now empty and ready for next collection

**Important**: Files are automatically moved to prevent accidental resubmission

### Step 7: Sync From Tugboat

**Purpose**: Download auditor-approved evidence from Tugboat
**Command**: `grctool sync`

```bash
# Sync all evidence from Tugboat
grctool sync
```

**What Happens**:
- Downloads approved evidence from Tugboat
- Files saved to `archive/` directory
- `archive/.submission/submission.yaml` contains Tugboat metadata
- These are read-only reference files

---

## Evidence State Transitions

Evidence moves through these states:

```
no_evidence → generated → evaluated → reviewed → submitted → accepted
                             ↓          ↓           ↓
                         (optional) (optional)   rejected
                                                     ↓
                                                 generated (rework)
```

### State Descriptions

| State | Description | Location | Next Action |
|-------|-------------|----------|-------------|
| **no_evidence** | No evidence files exist | Empty window | Run `grctool evidence generate` |
| **generated** | Evidence created | Root directory | Run `grctool evidence evaluate` (optional) |
| **evaluated** | Automated quality scored | Root directory + `.validation/` | Run `grctool evidence review` (optional) |
| **reviewed** | Human-reviewed | Root directory | Run `grctool evidence submit` |
| **submitted** | Evidence sent to Tugboat | `.submitted/` directory | Wait for auditor review |
| **accepted** | Evidence approved by auditors | `.submitted/` + `archive/` | Done! |
| **rejected** | Evidence needs rework | `.submitted/` | Move back to root, regenerate |

**Key Directory States**:
- **Root directory has files** = Working/Active evidence
- **`.submitted/` has files** = Submitted to Tugboat
- **`archive/` has files** = Synced from Tugboat (auditor-approved)

---

## Common Patterns

### Pattern 1: Single Evidence Task

When you need to collect evidence for one task:

```bash
# 1. Generate evidence with Claude Code
grctool evidence generate ET-0047 --window 2025-Q4
# Claude will guide you through tool execution and save to root directory

# 2. (Optional) Evaluate quality
grctool evidence evaluate ET-0047 --window 2025-Q4

# 3. (Optional) Human review
grctool evidence review ET-0047 --window 2025-Q4

# 4. (One-time) Setup collector URL
grctool evidence setup ET-0047 --collector-url "https://openapi.tugboatlogic.com/..."

# 5. Submit (files auto-move to .submitted/)
grctool evidence submit ET-0047 --window 2025-Q4
```

### Pattern 2: Multiple Related Tasks

When collecting evidence for multiple related tasks:

```bash
# 1. Generate context for all tasks
grctool evidence generate ET-0047 --window 2025-Q4  # GitHub access
grctool evidence generate ET-0048 --window 2025-Q4  # GitHub workflows
grctool evidence generate ET-0049 --window 2025-Q4  # GitHub security

# 2. Start Claude Code
claude

# 3. Tell Claude
# "I need to collect GitHub evidence for ET-0047, ET-0048, and ET-0049.
# Please help me efficiently collect all the evidence."

# Claude will help you:
# - Run tools once and reuse output
# - Save evidence to multiple tasks
# - Avoid duplicate work
```

### Pattern 3: Regenerating Rejected Evidence

When evidence is rejected and needs rework:

```bash
# 1. Check what was rejected
grctool status task ET-0047

# 2. Review feedback
cat data/evidence/*_ET-0047_*/2025-Q4/.submitted/.submission/submission.yaml

# 3. Move rejected files back to root directory
mv data/evidence/*_ET-0047_*/2025-Q4/.submitted/*.md data/evidence/*_ET-0047_*/2025-Q4/

# 4. Regenerate addressing feedback
grctool evidence generate ET-0047 --window 2025-Q4
# Claude will help you address the auditor feedback

# 5. Resubmit (files move back to .submitted/)
# Note: Collector URL already configured from initial setup
grctool evidence submit ET-0047 --window 2025-Q4
```

---

## Troubleshooting

### Issue: Context file not found

**Problem**: `grctool evidence generate` didn't create context
**Solution**:
```bash
# Check if task exists
grctool tool evidence-task-details --task-ref ET-0047

# Try again with explicit window
grctool evidence generate ET-0047 --window 2025-Q4
```

### Issue: Tool execution fails

**Problem**: Tool returns errors when collecting evidence
**Solution**:
```bash
# Check tool help for required parameters
grctool tool <tool-name> --help

# Verify configuration
cat .grctool.yaml

# Test with --dry-run
grctool tool <tool-name> --dry-run
```

### Issue: Evidence writer fails

**Problem**: Cannot save evidence files
**Solution**:
```bash
# Check file exists
ls -lh /tmp/evidence.csv

# Verify task reference is correct
grctool tool evidence-task-list | grep ET-0047

# Check permissions on data directory
ls -ld data/evidence/
```

---

**Next Steps**: Consult `bulk-operations.md` for autonomous multi-task workflows.
