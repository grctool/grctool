# Evidence Directory Structure

> Complete reference for evidence directory layout and file organization

---

**Generated**: 2025-10-28 10:33:19 EDT
**GRCTool Version**: dev
**Documentation Version**: dev

---

## Overview

GRCTool organizes all data in a structured directory hierarchy. Understanding this layout is critical for autonomous evidence generation and navigation.

## Root Data Directory

**Base Path**: `/Users/erik/Projects/7thsense-ops/isms` (configurable in `.grctool.yaml`)

```
/Users/erik/Projects/7thsense-ops/isms/
├── docs/                  # Synced data from Tugboat Logic
│   ├── policies/          # Policy documents and metadata
│   ├── controls/          # Security controls and requirements
│   └── evidence_tasks/    # Evidence collection task definitions
├── evidence/              # Generated evidence files
│   └── TaskName_ET-XXXX_TugboatID/  # One directory per evidence task
│       └── {window}/      # One directory per collection window
└── .cache/                # Performance cache (safe to delete)
```

## Evidence Task Directory Structure

Each evidence task has its own directory under `evidence/`, organized by collection windows.

### Directory Naming

Evidence task directories follow this pattern:
```
{TaskName}_ET-{ref}_{TugboatID}/
```

Examples:
- `GitHub_Access_Controls_ET-0001_328031/`
- `Repository_Permissions_ET-0047_328047/`
- `Infrastructure_Security_ET-0104_328104/`

**Format**: The directory name starts with a human-readable task name, followed by the ET reference (for easy lookup), and ends with the Tugboat ID (the canonical identifier).

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
evidence/GitHub_Access_ET-0001_328001/
├── 2025-Q4/                           # Current collection window
│   │
│   ├── 01_github_permissions_analysis.md  # Working evidence (root = working area)
│   ├── 02_deploy_workflow.yml             # Original source files for auditor review
│   ├── 03_access_control_policy.md        # Control documentation
│   │
│   ├── .generation/                   # Generation metadata (created by 'grctool tool evidence-writer')
│   │   └── metadata.yaml              # How evidence was generated, checksums, timestamps
│   │
│   ├── .validation/                   # Validation results (created by 'grctool evidence evaluate')
│   │   └── validation.yaml            # Quality scores and recommendations
│   │
│   ├── .submitted/                    # Submitted evidence (hidden, moved after upload)
│   │   ├── 01_github_permissions_analysis.md  # Files automatically moved here after submit
│   │   ├── 02_deploy_workflow.yml
│   │   └── .submission/               # Submission tracking
│   │       └── submission.yaml        # Submission status, Tugboat IDs, timestamps
│   │
│   ├── archive/                       # Evidence synced FROM Tugboat Logic
│   │   ├── 01_auditor_approved_evidence.pdf
│   │   └── .submission/
│   │       └── submission.yaml        # Tugboat metadata
│   │
│   └── collection_plan.md             # Collection plan for this window
│
├── 2025-Q3/                           # Previous window (reference)
│   ├── 01_github_permissions.csv      # Working files from last window
│   ├── .generation/
│   │   └── metadata.yaml
│   └── .submitted/
│       ├── 01_github_permissions.csv  # Previously submitted files
│       └── .submission/
│           └── submission.yaml
│
└── README.md                          # Task overview and collection notes
```

## Special Directories

### Root Directory (Working Area)

**Purpose**: Active working directory for evidence generation and review

**Created by**: `grctool tool evidence-writer` (automatic)

**Contents**:
- Evidence files numbered with prefixes (01_, 02_, etc.)
- Files remain here during drafting, evaluation, and review
- Files are evaluated before submission
- Files automatically move to `.submitted/` after successful upload

**Workflow**:
1. Evidence files are written to root directory during generation
2. Files are reviewed and refined in place
3. Files are evaluated using `grctool evidence evaluate`
4. Files are reviewed by humans using `grctool evidence review`
5. Files automatically move to `.submitted/` after successful submission

**Note**: The root directory is your working area. It's safe to regenerate or modify files here without affecting submitted evidence.

### `.generation/` Directory

**Purpose**: Tracks how evidence was generated

**Created by**: `grctool tool evidence-writer` (automatic)

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

**Purpose**: Tracks automated evidence quality evaluation

**Created by**: `grctool evidence evaluate ET-XXXX --window {window}`

**Contents**:
- `validation.yaml` - Automated quality scores and recommendations:
  ```yaml
  evaluated_at: 2025-10-27T12:00:00Z
  quality_score: 85
  completeness_score: 90
  recommendations:
    - Add more specific examples
    - Include control mapping
  status: ready
  ```

**Note**: Use `grctool evidence evaluate` for automated scoring. This is different from `grctool evidence review` which is for human assessment.

### `.submitted/` Directory (Hidden)

**Purpose**: Stores evidence after successful submission to prevent resubmission

**Created by**: `grctool evidence submit ET-XXXX --window {window}` (automatic)

**Contents**:
- Submitted evidence files (automatically moved from root after upload)
- `.submission/submission.yaml` - Submission tracking:
  ```yaml
  submitted_at: 2025-10-27T14:00:00Z
  submission_id: sub_abc123
  submission_status: submitted
  files_submitted:
    - 01_github_permissions_analysis.md
    - 02_deploy_workflow.yml
  tugboat_response:
    status: accepted
    message: Evidence received successfully
  ```

**Workflow**:
1. Evidence exists in root directory
2. `grctool evidence submit` uploads files to Tugboat
3. Files automatically move from root to `.submitted/`
4. Root directory is now empty and ready for next collection

**Resubmission**: To resubmit, move files back from `.submitted/` to root directory

**Submission Statuses**:
- `draft` - Evidence prepared but not submitted
- `validated` - Evidence evaluated and ready
- `submitted` - Sent to Tugboat Logic
- `accepted` - Accepted by auditors
- `rejected` - Rejected, needs rework

### `archive/` Directory

**Purpose**: Stores evidence synced FROM Tugboat Logic

**Created by**: `grctool sync` (automatic)

**Contents**:
- Evidence files downloaded from Tugboat (auditor-approved versions)
- `.submission/submission.yaml` - Tugboat metadata about the synced evidence

**Important**:
- `archive/` = evidence FROM Tugboat (read-only reference)
- `.submitted/` = evidence TO Tugboat (locally uploaded)
- These are separate directories to avoid confusion

## File Naming Conventions

### Evidence Files

Evidence files are automatically numbered with zero-padded prefixes:

```
01_github_permissions_summary.md
02_terraform_main.tf
03_deploy_workflow.yml
```

**Naming Guidelines**:
- Use lowercase with underscores
- Be descriptive but concise
- Include the tool name or data type
- Use appropriate file extensions

**Auditor-Friendly File Types**:
- **Evidence Narratives**: `.md` (markdown documents with findings and analysis)
- **Infrastructure Source**: `.tf`, `.yml`, `.yaml`, `.toml`, `.hcl` (original config files)
- **Application Configs**: `.env.example`, `.config`, configuration files
- **Documents**: `.pdf`, `.doc`, `.docx` (when needed)
- **NOT Included**: `.json` files (tool outputs are for analysis only, not auditor submission)

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
ls /Users/erik/Projects/7thsense-ops/isms/docs/evidence_tasks/

# Find a specific task by reference
ls /Users/erik/Projects/7thsense-ops/isms/docs/evidence_tasks/ET-0047-*
```

### Checking Existing Evidence
```bash
# List evidence for a task
ls /Users/erik/Projects/7thsense-ops/isms/evidence/*_ET-0047_*/

# List windows for a task
ls /Users/erik/Projects/7thsense-ops/isms/evidence/*_ET-0047_*/*/

# Check working evidence files (root directory)
ls /Users/erik/Projects/7thsense-ops/isms/evidence/*_ET-0047_*/2025-Q4/

# Check submitted evidence (hidden folder)
ls /Users/erik/Projects/7thsense-ops/isms/evidence/*_ET-0047_*/2025-Q4/.submitted/

# Check synced evidence from Tugboat
ls /Users/erik/Projects/7thsense-ops/isms/evidence/*_ET-0047_*/2025-Q4/archive/
```

### Reading Metadata Files
```bash
# Check generation metadata
cat /Users/erik/Projects/7thsense-ops/isms/evidence/*_ET-0047_*/2025-Q4/.generation/metadata.yaml

# Check validation results
cat /Users/erik/Projects/7thsense-ops/isms/evidence/*_ET-0047_*/2025-Q4/.validation/validation.yaml

# Check submission status
cat /Users/erik/Projects/7thsense-ops/isms/evidence/*_ET-0047_*/2025-Q4/.submitted/.submission/submission.yaml

# Check Tugboat sync metadata
cat /Users/erik/Projects/7thsense-ops/isms/evidence/*_ET-0047_*/2025-Q4/archive/.submission/submission.yaml
```

## State Tracking

Evidence state is tracked through the presence and content of metadata files:

| State | Indicators |
|-------|------------|
| **No Evidence** | No window directory exists OR window directory is empty |
| **Generated** | Evidence files exist in root directory, `.generation/metadata.yaml` present |
| **Evaluated** | `.validation/validation.yaml` exists with quality scores |
| **Submitted** | Files moved to `.submitted/`, `.submitted/.submission/submission.yaml` exists |
| **Accepted** | `.submitted/.submission/submission.yaml` has `status: accepted` |
| **Rejected** | `.submitted/.submission/submission.yaml` has `status: rejected` |
| **Synced** | `archive/` directory contains evidence downloaded from Tugboat |

**Key Workflow States**:
- **Working**: Files in root directory (actively being created/refined)
- **Submitted**: Files in `.submitted/` directory (uploaded to Tugboat)
- **Archived**: Files in `archive/` directory (synced from Tugboat)

Use `grctool status` to see aggregated state across all tasks.

---

**Next Steps**: Consult `evidence-workflow.md` for the complete evidence generation workflow.
