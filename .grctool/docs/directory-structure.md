# Evidence Directory Structure

> Complete reference for evidence directory layout and file organization

---

**Generated**: 2025-10-27 16:56:21 EDT  
**GRCTool Version**: dev  
**Documentation Version**: dev  

---

## Overview

GRCTool organizes all data in a structured directory hierarchy. Understanding this layout is critical for autonomous evidence generation and navigation.

## Root Data Directory

**Base Path**: `/Users/erik/Projects/grctool` (configurable in `.grctool.yaml`)

```
/Users/erik/Projects/grctool/
├── docs/                  # Synced data from Tugboat Logic
│   ├── policies/          # Policy documents and metadata
│   ├── controls/          # Security controls and requirements
│   └── evidence_tasks/    # Evidence collection task definitions
├── evidence/              # Generated evidence files
│   └── ET-XXXX_TaskName/  # One directory per evidence task
│       └── {window}/      # One directory per collection window
└── .cache/                # Performance cache (safe to delete)
```

## Evidence Task Directory Structure

Each evidence task has its own directory under `evidence/`, organized by collection windows and workflow stages.

### Directory Naming

Evidence task directories follow this pattern:
```
ET-{ref}_{TaskName}/
```

Examples:
- `ET-0001_GitHub_Access_Controls/`
- `ET-0047_Repository_Permissions/`
- `ET-0104_Infrastructure_Security/`

### Window-Based Organization

Evidence is organized into collection windows based on the task's collection interval:

| Interval | Window Format | Examples |
|----------|---------------|----------|
| Annual | `YYYY` | `2025`, `2026` |
| Quarterly | `YYYY-QN` | `2025-Q4`, `2026-Q1` |
| Monthly | `YYYY-MM` | `2025-10`, `2025-11` |
| Semi-annual | `YYYY-HN` | `2025-H2`, `2026-H1` |

### Complete Evidence Task Directory Layout

```
evidence/ET-0001_GitHub_Access/
├── 2025-Q4/                           # Current collection window
│   │
│   ├── wip/                           # Work in progress (LocalState: generated)
│   │   ├── 01_draft_permissions.csv   # Files being actively worked on
│   │   ├── 02_analysis.md             # Drafts, partial evidence
│   │   └── .generation/               # How these files were created
│   │       └── metadata.yaml          # Tools used, timestamps, checksums
│   │
│   ├── ready/                         # Validated, ready to submit (LocalState: validated)
│   │   ├── 01_github_permissions.csv  # Completed, validated evidence
│   │   ├── 02_team_access.json
│   │   ├── 02_analysis.md             # Original markdown (for editing)
│   │   ├── 02_analysis.pdf            # Auto-generated PDF (for submission)
│   │   ├── .generation/               # Moved from wip/ with files
│   │   │   └── metadata.yaml
│   │   ├── .validation/               # Validation results (created by validation)
│   │   │   └── validation.yaml        # Score, completeness, readiness check
│   │   └── .submission/               # Created when submitting from ready/
│   │       └── submission.yaml        # Tugboat submission ID, timestamps
│   │
│   ├── submitted/                     # Downloaded from Tugboat (LocalState: submitted/accepted)
│   │   ├── 01_github_permissions.csv  # What's actually in Tugboat
│   │   ├── 02_team_access.json
│   │   └── .submission/               # Submission metadata (synced from Tugboat)
│   │       ├── submission.yaml        # Tugboat submission ID, timestamps
│   │       └── history.yaml           # Full submission history
│   │
│   ├── .context/                      # Shared generation context for window
│   │   └── generation-context.md      # Task details, tools, requirements
│   │
│   └── collection_plan.md             # Overall collection plan (deprecated)
│
└── 2025-Q3/                           # Previous window (reference)
    └── submitted/                     # Only submitted evidence remains
        ├── 01_github_permissions.csv
        ├── 02_team_memberships.json
        └── .submission/
            ├── submission.yaml
            └── history.yaml
```

## Special Directories

### `.context/` Directory

**Purpose**: Contains context for Claude Code assisted evidence generation

**Created by**: `grctool evidence generate ET-XXXX --window {window}`

**Location**: `{window}/.context/` (shared across all subfolders)

**Contents**:
- `generation-context.md` - Comprehensive context document including:
  - Task details and requirements
  - Applicable tools auto-detected from task description
  - Related controls and policies
  - Existing evidence from previous windows
  - Source locations from config
  - Next steps for evidence generation

### `.generation/` Directory

**Purpose**: Tracks how evidence was generated

**Created by**: `grctool tool evidence-writer` (automatic)

**Location**:
- `wip/.generation/` - For newly generated evidence
- `ready/.generation/` - Moved with files during validation

**Contents**:
- `metadata.yaml` - Generation metadata:
  ```yaml
  generated_at: 2025-10-27T10:30:00Z
  generated_by: claude-code-assisted
  generation_method: tool_coordination
  task_id: 327992
  task_ref: ET-0001
  window: 2025-Q4
  tools_used: [github-permissions]
  files_generated:
    - path: 01_github_permissions.csv
      checksum: sha256:abc123...
      size_bytes: 15420
      generated_at: 2025-10-27T10:30:00Z
  status: generated
  ```

**Generation Methods**:
- `claude-code-assisted` - Generated with Claude Code assistance
- `tool_coordination` - Multiple tools orchestrated together
- `grctool-cli` - Direct CLI tool execution
- `manual` - Manually created/uploaded file

### `.validation/` Directory

**Purpose**: Contains validation results for evidence ready to submit

**Created by**: `grctool evidence validate ET-XXXX --window {window}`

**Location**: `ready/.validation/`

**Contents**:
- `validation.yaml` - Validation results:
  ```yaml
  validated_at: 2025-10-27T12:00:00Z
  validation_score: 95
  completeness: complete
  issues: []
  recommendations:
    - "Add more detail to section 3"
  status: validated
  ```

### `.submission/` Directory

**Purpose**: Tracks submission to Tugboat Logic

**Created by**:
- `grctool evidence submit ET-XXXX --window {window}` (creates in `ready/`)
- `grctool sync --submissions` (creates in `submitted/`)

**Location**:
- `ready/.submission/` - When submitting from local ready/ folder
- `submitted/.submission/` - When downloading from Tugboat

**Contents**:
- `submission.yaml` - Submission tracking:
  ```yaml
  submitted_at: 2025-10-27T14:00:00Z
  submission_id: sub_abc123
  submission_status: submitted
  files_submitted:
    - 01_github_permissions.csv
    - 02_team_memberships.json
  tugboat_response:
    status: accepted
    message: Evidence received successfully
  ```
- `history.yaml` - Complete submission history (in `submitted/` only)

**Submission Statuses**:
- `draft` - Evidence prepared but not submitted
- `validated` - Evidence validated and ready
- `submitted` - Sent to Tugboat Logic
- `accepted` - Accepted by auditors
- `rejected` - Rejected, needs rework

## Automatic Markdown to PDF Conversion

When evidence files move from `wip/` to `ready/` during validation, **markdown files are automatically converted to PDF** for professional presentation to auditors.

### How It Works

1. **Validation triggers conversion**:
   ```bash
   grctool evidence validate ET-0001 --window 2025-Q4
   ```

2. **For each `.md` file in `ready/`**:
   - Converts to PDF with professional formatting
   - Keeps original `.md` file for future edits
   - Example: `02_analysis.md` → `02_analysis.pdf`

3. **During submission**:
   - Only PDF files are submitted to Tugboat
   - Markdown files are kept locally as source

### Technical Details

- **Parser**: goldmark (CommonMark compliant)
- **Renderer**: gopdf (pure Go, no external dependencies)
- **Supported Elements**:
  - Headings (H1-H6)
  - Paragraphs and text formatting
  - Lists (ordered/unordered, nested)
  - Code blocks (monospace font)
  - Tables (basic support)
  - Page breaks (automatic)

### File Preference

When both `.md` and `.pdf` exist with the same basename:
- **Submission**: PDF is preferred
- **Editing**: Markdown is the source of truth
- **Re-validation**: PDF is regenerated if `.md` is newer

### Fallback Behavior

If PDF conversion fails:
- Warning is logged
- Markdown file is submitted instead
- Validation continues normally

## File Naming Conventions

### Evidence Files

Evidence files are automatically numbered with zero-padded prefixes:

```
01_descriptive_name.csv
02_another_file.json
03_summary_report.md
```

**Naming Guidelines**:
- Use lowercase with underscores
- Be descriptive but concise
- Include the tool name or data type
- Use appropriate file extensions

**Supported File Types**:
- **Data**: `.csv`, `.json`, `.yaml`, `.txt`
- **Documents**: `.md`, `.pdf`, `.doc`, `.docx`
- **Spreadsheets**: `.xls`, `.xlsx`, `.ods`
- **Images**: `.png`, `.jpg`, `.jpeg`, `.gif`

## Synced Data Structure

### `docs/policies/`

Contains policy documents synced from Tugboat Logic:
```
docs/policies/
├── POL-0001_Information_Security_Policy.json
├── POL-0001_Information_Security_Policy.md
├── POL-0002_Access_Control_Policy.json
└── POL-0002_Access_Control_Policy.md
```

**File Formats**:
- `.json` - Structured metadata (ID, status, frameworks)
- `.md` - Human-readable policy content

### `docs/controls/`

Contains security controls synced from Tugboat Logic:
```
docs/controls/
├── AC-01_Access_Control.json
├── AC-01_Access_Control.md
├── CC6.8_Logical_Access.json
└── CC6.8_Logical_Access.md
```

**File Formats**:
- `.json` - Structured metadata (control family, requirements)
- `.md` - Human-readable control description

### `docs/evidence_tasks/`

Contains evidence task definitions synced from Tugboat Logic:
```
docs/evidence_tasks/
├── ET-0001-327992_github_access.json
├── ET-0047-328456_repository_permissions.json
└── ET-0104-329123_infrastructure_security.json
```

**Filename Pattern**: `ET-{ref}-{id}_{name}.json`

**Contains**:
- Task reference (ET-XXXX)
- Numeric ID
- Task name and description
- Collection interval
- Related controls
- Assignee information
- Tugboat status

## Navigation Tips for Claude Code

### Finding Evidence Tasks
```bash
# List all evidence tasks
ls /Users/erik/Projects/grctool/docs/evidence_tasks/

# Find a specific task by reference
ls /Users/erik/Projects/grctool/docs/evidence_tasks/ET-0047-*
```

### Checking Existing Evidence
```bash
# List evidence for a task
ls /Users/erik/Projects/grctool/evidence/ET-0047_*/

# List windows for a task
ls /Users/erik/Projects/grctool/evidence/ET-0047_*/*/

# Check if generation context exists
ls /Users/erik/Projects/grctool/evidence/ET-0047_*/2025-Q4/.context/
```

### Reading Context Files
```bash
# Read generation context
cat /Users/erik/Projects/grctool/evidence/ET-0047_*/2025-Q4/.context/generation-context.md

# Check generation metadata
cat /Users/erik/Projects/grctool/evidence/ET-0047_*/2025-Q4/.generation/metadata.yaml

# Check submission status
cat /Users/erik/Projects/grctool/evidence/ET-0047_*/2025-Q4/.submission/submission.yaml
```

## State Tracking

Evidence state is tracked through **folder location** and metadata files:

| State | Folder Location | Metadata Indicators |
|-------|----------------|---------------------|
| **No Evidence** | No window directory exists | - |
| **Generated** | Files in `wip/` | `wip/.generation/metadata.yaml` exists |
| **Validated** | Files in `ready/` | `ready/.validation/validation.yaml` exists |
| **Submitted** | Files in `ready/` with submission metadata | `ready/.submission/submission.yaml` with `status: submitted` |
| **Accepted** | Files in `submitted/` from Tugboat sync | `submitted/.submission/submission.yaml` with `status: accepted` |
| **Rejected** | Files in `ready/` or `submitted/` | `.submission/submission.yaml` has `status: rejected` |

### State Inference

The evidence scanner determines state by:

1. **Checking folder structure**: Scans `wip/`, `ready/`, `submitted/` subfolders
2. **Reading metadata**: Checks for `.generation/`, `.validation/`, `.submission/` directories
3. **Prioritizing ready/**: If files exist in multiple folders, `ready/` takes precedence for determining state

Use `grctool status` to see aggregated state across all tasks.

---

**Next Steps**: Consult `evidence-workflow.md` for the complete evidence generation workflow.
