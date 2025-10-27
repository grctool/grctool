// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/config"
)

// Legacy documentation generators - these are being migrated to templates
// TODO: Remove this file once all generators are converted to templates

// writeDocHeader creates a standard header for all documentation files
func writeDocHeader(title, description, version string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n\n", title))
	sb.WriteString(fmt.Sprintf("> %s\n\n", description))
	sb.WriteString("---\n\n")
	sb.WriteString(fmt.Sprintf("**Generated**: %s  \n", time.Now().Format("2006-01-02 15:04:05 MST")))
	sb.WriteString(fmt.Sprintf("**GRCTool Version**: %s  \n", version))
	sb.WriteString(fmt.Sprintf("**Documentation Version**: %s  \n\n", version))
	sb.WriteString("---\n\n")

	return sb.String()
}

// generateDirectoryStructureDocs generates documentation about evidence directory layout
func generateDirectoryStructureDocs(cfg *config.Config, version string) (string, error) {
	var sb strings.Builder

	sb.WriteString(writeDocHeader(
		"Evidence Directory Structure",
		"Complete reference for evidence directory layout and file organization",
		version,
	))

	sb.WriteString("## Overview\n\n")
	sb.WriteString("GRCTool organizes all data in a structured directory hierarchy. Understanding this layout is critical for autonomous evidence generation and navigation.\n\n")

	sb.WriteString("## Root Data Directory\n\n")
	dataDir := cfg.Storage.DataDir
	if dataDir == "" {
		dataDir = "./data"
	}
	sb.WriteString(fmt.Sprintf("**Base Path**: `%s` (configurable in `.grctool.yaml`)\n\n", dataDir))

	sb.WriteString("```\n")
	sb.WriteString(dataDir + "/\n")
	sb.WriteString("├── docs/                  # Synced data from Tugboat Logic\n")
	sb.WriteString("│   ├── policies/          # Policy documents and metadata\n")
	sb.WriteString("│   ├── controls/          # Security controls and requirements\n")
	sb.WriteString("│   └── evidence_tasks/    # Evidence collection task definitions\n")
	sb.WriteString("├── evidence/              # Generated evidence files\n")
	sb.WriteString("│   └── ET-XXXX_TaskName/  # One directory per evidence task\n")
	sb.WriteString("│       └── {window}/      # One directory per collection window\n")
	sb.WriteString("└── .cache/                # Performance cache (safe to delete)\n")
	sb.WriteString("```\n\n")

	sb.WriteString("## Evidence Task Directory Structure\n\n")
	sb.WriteString("Each evidence task has its own directory under `evidence/`, organized by collection windows.\n\n")

	sb.WriteString("### Directory Naming\n\n")
	sb.WriteString("Evidence task directories follow this pattern:\n")
	sb.WriteString("```\n")
	sb.WriteString("ET-{ref}_{TaskName}/\n")
	sb.WriteString("```\n\n")
	sb.WriteString("Examples:\n")
	sb.WriteString("- `ET-0001_GitHub_Access_Controls/`\n")
	sb.WriteString("- `ET-0047_Repository_Permissions/`\n")
	sb.WriteString("- `ET-0104_Infrastructure_Security/`\n\n")

	sb.WriteString("### Window-Based Organization\n\n")
	sb.WriteString("Evidence is organized into collection windows based on the task's collection interval:\n\n")
	sb.WriteString("| Interval | Window Format | Examples |\n")
	sb.WriteString("|----------|---------------|----------|\n")
	sb.WriteString("| Annual | `YYYY` | `2025`, `2026` |\n")
	sb.WriteString("| Quarterly | `YYYY-QN` | `2025-Q4`, `2026-Q1` |\n")
	sb.WriteString("| Monthly | `YYYY-MM` | `2025-10`, `2025-11` |\n")
	sb.WriteString("| Semi-annual | `YYYY-HN` | `2025-H2`, `2026-H1` |\n\n")

	sb.WriteString("### Complete Evidence Task Directory Layout\n\n")
	sb.WriteString("```\n")
	sb.WriteString("evidence/ET-0001_GitHub_Access/\n")
	sb.WriteString("├── 2025-Q4/                           # Current collection window\n")
	sb.WriteString("│   ├── 01_github_permissions.csv      # Evidence file 1\n")
	sb.WriteString("│   ├── 02_team_memberships.json       # Evidence file 2\n")
	sb.WriteString("│   ├── 03_access_summary.md           # Evidence file 3\n")
	sb.WriteString("│   │\n")
	sb.WriteString("│   ├── .context/                      # Generation context (created by 'grctool evidence generate')\n")
	sb.WriteString("│   │   └── generation-context.md      # Task details, applicable tools, existing evidence\n")
	sb.WriteString("│   │\n")
	sb.WriteString("│   ├── .generation/                   # Generation metadata (created by 'grctool tool evidence-writer')\n")
	sb.WriteString("│   │   └── metadata.yaml              # How evidence was generated, checksums, timestamps\n")
	sb.WriteString("│   │\n")
	sb.WriteString("│   └── .submission/                   # Submission tracking (created by 'grctool evidence submit')\n")
	sb.WriteString("│       └── submission.yaml            # Submission status, Tugboat IDs, timestamps\n")
	sb.WriteString("│\n")
	sb.WriteString("├── 2025-Q3/                           # Previous window (reference)\n")
	sb.WriteString("│   ├── 01_github_permissions.csv\n")
	sb.WriteString("│   ├── .generation/\n")
	sb.WriteString("│   │   └── metadata.yaml\n")
	sb.WriteString("│   └── .submission/\n")
	sb.WriteString("│       └── submission.yaml\n")
	sb.WriteString("│\n")
	sb.WriteString("└── collection_plan.md                 # Overall collection plan (deprecated)\n")
	sb.WriteString("```\n\n")

	sb.WriteString("## Special Directories\n\n")

	sb.WriteString("### `.context/` Directory\n\n")
	sb.WriteString("**Purpose**: Contains context for Claude Code assisted evidence generation\n\n")
	sb.WriteString("**Created by**: `grctool evidence generate ET-XXXX --window {window}`\n\n")
	sb.WriteString("**Contents**:\n")
	sb.WriteString("- `generation-context.md` - Comprehensive context document including:\n")
	sb.WriteString("  - Task details and requirements\n")
	sb.WriteString("  - Applicable tools auto-detected from task description\n")
	sb.WriteString("  - Related controls and policies\n")
	sb.WriteString("  - Existing evidence from previous windows\n")
	sb.WriteString("  - Source locations from config\n")
	sb.WriteString("  - Next steps for evidence generation\n\n")

	sb.WriteString("### `.generation/` Directory\n\n")
	sb.WriteString("**Purpose**: Tracks how evidence was generated\n\n")
	sb.WriteString("**Created by**: `grctool tool evidence-writer` (automatic)\n\n")
	sb.WriteString("**Contents**:\n")
	sb.WriteString("- `metadata.yaml` - Generation metadata:\n")
	sb.WriteString("  ```yaml\n")
	sb.WriteString("  generated_at: 2025-10-27T10:30:00Z\n")
	sb.WriteString("  generated_by: claude-code-assisted\n")
	sb.WriteString("  generation_method: tool_coordination\n")
	sb.WriteString("  task_id: 327992\n")
	sb.WriteString("  task_ref: ET-0001\n")
	sb.WriteString("  window: 2025-Q4\n")
	sb.WriteString("  tools_used: [github-permissions]\n")
	sb.WriteString("  files_generated:\n")
	sb.WriteString("    - path: 01_github_permissions.csv\n")
	sb.WriteString("      checksum: sha256:abc123...\n")
	sb.WriteString("      size_bytes: 15420\n")
	sb.WriteString("      generated_at: 2025-10-27T10:30:00Z\n")
	sb.WriteString("  status: generated\n")
	sb.WriteString("  ```\n\n")

	sb.WriteString("**Generation Methods**:\n")
	sb.WriteString("- `claude-code-assisted` - Generated with Claude Code assistance\n")
	sb.WriteString("- `tool_coordination` - Multiple tools orchestrated together\n")
	sb.WriteString("- `grctool-cli` - Direct CLI tool execution\n")
	sb.WriteString("- `manual` - Manually created/uploaded file\n\n")

	sb.WriteString("### `.submission/` Directory\n\n")
	sb.WriteString("**Purpose**: Tracks submission to Tugboat Logic\n\n")
	sb.WriteString("**Created by**: `grctool evidence submit ET-XXXX --window {window}`\n\n")
	sb.WriteString("**Contents**:\n")
	sb.WriteString("- `submission.yaml` - Submission tracking:\n")
	sb.WriteString("  ```yaml\n")
	sb.WriteString("  submitted_at: 2025-10-27T14:00:00Z\n")
	sb.WriteString("  submission_id: sub_abc123\n")
	sb.WriteString("  submission_status: submitted\n")
	sb.WriteString("  files_submitted:\n")
	sb.WriteString("    - 01_github_permissions.csv\n")
	sb.WriteString("    - 02_team_memberships.json\n")
	sb.WriteString("  tugboat_response:\n")
	sb.WriteString("    status: accepted\n")
	sb.WriteString("    message: Evidence received successfully\n")
	sb.WriteString("  ```\n\n")

	sb.WriteString("**Submission Statuses**:\n")
	sb.WriteString("- `draft` - Evidence prepared but not submitted\n")
	sb.WriteString("- `validated` - Evidence validated and ready\n")
	sb.WriteString("- `submitted` - Sent to Tugboat Logic\n")
	sb.WriteString("- `accepted` - Accepted by auditors\n")
	sb.WriteString("- `rejected` - Rejected, needs rework\n\n")

	sb.WriteString("## File Naming Conventions\n\n")

	sb.WriteString("### Evidence Files\n\n")
	sb.WriteString("Evidence files are automatically numbered with zero-padded prefixes:\n\n")
	sb.WriteString("```\n")
	sb.WriteString("01_descriptive_name.csv\n")
	sb.WriteString("02_another_file.json\n")
	sb.WriteString("03_summary_report.md\n")
	sb.WriteString("```\n\n")

	sb.WriteString("**Naming Guidelines**:\n")
	sb.WriteString("- Use lowercase with underscores\n")
	sb.WriteString("- Be descriptive but concise\n")
	sb.WriteString("- Include the tool name or data type\n")
	sb.WriteString("- Use appropriate file extensions\n\n")

	sb.WriteString("**Supported File Types**:\n")
	sb.WriteString("- **Data**: `.csv`, `.json`, `.yaml`, `.txt`\n")
	sb.WriteString("- **Documents**: `.md`, `.pdf`, `.doc`, `.docx`\n")
	sb.WriteString("- **Spreadsheets**: `.xls`, `.xlsx`, `.ods`\n")
	sb.WriteString("- **Images**: `.png`, `.jpg`, `.jpeg`, `.gif`\n\n")

	sb.WriteString("## Synced Data Structure\n\n")

	sb.WriteString("### `docs/policies/`\n\n")
	sb.WriteString("Contains policy documents synced from Tugboat Logic:\n")
	sb.WriteString("```\n")
	sb.WriteString("docs/policies/\n")
	sb.WriteString("├── POL-0001_Information_Security_Policy.json\n")
	sb.WriteString("├── POL-0001_Information_Security_Policy.md\n")
	sb.WriteString("├── POL-0002_Access_Control_Policy.json\n")
	sb.WriteString("└── POL-0002_Access_Control_Policy.md\n")
	sb.WriteString("```\n\n")

	sb.WriteString("**File Formats**:\n")
	sb.WriteString("- `.json` - Structured metadata (ID, status, frameworks)\n")
	sb.WriteString("- `.md` - Human-readable policy content\n\n")

	sb.WriteString("### `docs/controls/`\n\n")
	sb.WriteString("Contains security controls synced from Tugboat Logic:\n")
	sb.WriteString("```\n")
	sb.WriteString("docs/controls/\n")
	sb.WriteString("├── AC-01_Access_Control.json\n")
	sb.WriteString("├── AC-01_Access_Control.md\n")
	sb.WriteString("├── CC6.8_Logical_Access.json\n")
	sb.WriteString("└── CC6.8_Logical_Access.md\n")
	sb.WriteString("```\n\n")

	sb.WriteString("**File Formats**:\n")
	sb.WriteString("- `.json` - Structured metadata (control family, requirements)\n")
	sb.WriteString("- `.md` - Human-readable control description\n\n")

	sb.WriteString("### `docs/evidence_tasks/`\n\n")
	sb.WriteString("Contains evidence task definitions synced from Tugboat Logic:\n")
	sb.WriteString("```\n")
	sb.WriteString("docs/evidence_tasks/\n")
	sb.WriteString("├── ET-0001-327992_github_access.json\n")
	sb.WriteString("├── ET-0047-328456_repository_permissions.json\n")
	sb.WriteString("└── ET-0104-329123_infrastructure_security.json\n")
	sb.WriteString("```\n\n")

	sb.WriteString("**Filename Pattern**: `ET-{ref}-{id}_{name}.json`\n\n")
	sb.WriteString("**Contains**:\n")
	sb.WriteString("- Task reference (ET-XXXX)\n")
	sb.WriteString("- Numeric ID\n")
	sb.WriteString("- Task name and description\n")
	sb.WriteString("- Collection interval\n")
	sb.WriteString("- Related controls\n")
	sb.WriteString("- Assignee information\n")
	sb.WriteString("- Tugboat status\n\n")

	sb.WriteString("## Navigation Tips for Claude Code\n\n")

	sb.WriteString("### Finding Evidence Tasks\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# List all evidence tasks\n")
	sb.WriteString("ls " + dataDir + "/docs/evidence_tasks/\n\n")
	sb.WriteString("# Find a specific task by reference\n")
	sb.WriteString("ls " + dataDir + "/docs/evidence_tasks/ET-0047-*\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### Checking Existing Evidence\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# List evidence for a task\n")
	sb.WriteString("ls " + dataDir + "/evidence/ET-0047_*/\n\n")
	sb.WriteString("# List windows for a task\n")
	sb.WriteString("ls " + dataDir + "/evidence/ET-0047_*/*/\n\n")
	sb.WriteString("# Check if generation context exists\n")
	sb.WriteString("ls " + dataDir + "/evidence/ET-0047_*/2025-Q4/.context/\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### Reading Context Files\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Read generation context\n")
	sb.WriteString("cat " + dataDir + "/evidence/ET-0047_*/2025-Q4/.context/generation-context.md\n\n")
	sb.WriteString("# Check generation metadata\n")
	sb.WriteString("cat " + dataDir + "/evidence/ET-0047_*/2025-Q4/.generation/metadata.yaml\n\n")
	sb.WriteString("# Check submission status\n")
	sb.WriteString("cat " + dataDir + "/evidence/ET-0047_*/2025-Q4/.submission/submission.yaml\n")
	sb.WriteString("```\n\n")

	sb.WriteString("## State Tracking\n\n")
	sb.WriteString("Evidence state is tracked through the presence and content of metadata files:\n\n")
	sb.WriteString("| State | Indicators |\n")
	sb.WriteString("|-------|------------|\n")
	sb.WriteString("| **No Evidence** | No window directory exists |\n")
	sb.WriteString("| **Generated** | Evidence files exist, `.generation/metadata.yaml` present |\n")
	sb.WriteString("| **Validated** | `.generation/metadata.yaml` has `status: validated` |\n")
	sb.WriteString("| **Submitted** | `.submission/submission.yaml` exists with `status: submitted` |\n")
	sb.WriteString("| **Accepted** | `.submission/submission.yaml` has `status: accepted` |\n")
	sb.WriteString("| **Rejected** | `.submission/submission.yaml` has `status: rejected` |\n\n")

	sb.WriteString("Use `grctool status` to see aggregated state across all tasks.\n\n")

	sb.WriteString("---\n\n")
	sb.WriteString("**Next Steps**: Consult `evidence-workflow.md` for the complete evidence generation workflow.\n")

	return sb.String(), nil
}

// generateEvidenceWorkflowDocs generates documentation about the evidence generation workflow
func generateEvidenceWorkflowDocs(cfg *config.Config, version string) (string, error) {
	var sb strings.Builder

	sb.WriteString(writeDocHeader(
		"Evidence Generation Workflow",
		"Complete workflow for generating and managing evidence with Claude Code",
		version,
	))

	sb.WriteString("## Overview\n\n")
	sb.WriteString("This document describes the complete end-to-end workflow for evidence generation using GRCTool with Claude Code assistance.\n\n")

	sb.WriteString("### Workflow Philosophy\n\n")
	sb.WriteString("GRCTool follows a **Claude Code Assisted** approach:\n")
	sb.WriteString("1. GRCTool generates comprehensive context\n")
	sb.WriteString("2. Claude Code reads the context and helps you interactively\n")
	sb.WriteString("3. You run tools together with Claude's guidance\n")
	sb.WriteString("4. Evidence is saved with automatic metadata tracking\n")
	sb.WriteString("5. State is tracked throughout the lifecycle\n\n")

	sb.WriteString("---\n\n")

	// Complete Workflow
	sb.WriteString("## Complete Evidence Lifecycle\n\n")

	sb.WriteString("```\n")
	sb.WriteString("┌─────────────────┐\n")
	sb.WriteString("│ grctool status  │  (1) Check what evidence is needed\n")
	sb.WriteString("└────────┬────────┘\n")
	sb.WriteString("         │\n")
	sb.WriteString("         v\n")
	sb.WriteString("┌──────────────────────┐\n")
	sb.WriteString("│ grctool evidence     │  (2) Generate context for task\n")
	sb.WriteString("│   generate ET-XXXX   │\n")
	sb.WriteString("└────────┬─────────────┘\n")
	sb.WriteString("         │\n")
	sb.WriteString("         v\n")
	sb.WriteString("┌──────────────────────┐\n")
	sb.WriteString("│  Claude Code reads   │  (3) Claude reads .context/generation-context.md\n")
	sb.WriteString("│     context file     │\n")
	sb.WriteString("└────────┬─────────────┘\n")
	sb.WriteString("         │\n")
	sb.WriteString("         v\n")
	sb.WriteString("┌──────────────────────┐\n")
	sb.WriteString("│  Run tools together  │  (4) Execute grctool tools with guidance\n")
	sb.WriteString("│  with Claude's help  │\n")
	sb.WriteString("└────────┬─────────────┘\n")
	sb.WriteString("         │\n")
	sb.WriteString("         v\n")
	sb.WriteString("┌──────────────────────┐\n")
	sb.WriteString("│ grctool tool         │  (5) Save evidence with metadata\n")
	sb.WriteString("│   evidence-writer    │\n")
	sb.WriteString("└────────┬─────────────┘\n")
	sb.WriteString("         │\n")
	sb.WriteString("         v\n")
	sb.WriteString("┌──────────────────────┐\n")
	sb.WriteString("│ grctool evidence     │  (6) Validate before submission\n")
	sb.WriteString("│   validate ET-XXXX   │\n")
	sb.WriteString("└────────┬─────────────┘\n")
	sb.WriteString("         │\n")
	sb.WriteString("         v\n")
	sb.WriteString("┌──────────────────────┐\n")
	sb.WriteString("│ grctool evidence     │  (7) Submit to Tugboat\n")
	sb.WriteString("│   submit ET-XXXX     │\n")
	sb.WriteString("└──────────────────────┘\n")
	sb.WriteString("```\n\n")

	// Step by Step
	sb.WriteString("---\n\n")
	sb.WriteString("## Step-by-Step Workflow\n\n")

	sb.WriteString("### Step 1: Check Evidence Status\n\n")
	sb.WriteString("**Purpose**: See what evidence needs to be collected  \n")
	sb.WriteString("**Command**: `grctool status`  \n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# See dashboard\n")
	sb.WriteString("grctool status\n\n")
	sb.WriteString("# Check specific task\n")
	sb.WriteString("grctool status task ET-0047\n\n")
	sb.WriteString("# Filter by state\n")
	sb.WriteString("grctool status --filter state=no_evidence\n")
	sb.WriteString("```\n\n")
	sb.WriteString("**What You See**:\n")
	sb.WriteString("- Overall progress summary\n")
	sb.WriteString("- Tasks grouped by state (no_evidence, generated, submitted, etc.)\n")
	sb.WriteString("- Automation level for each task\n")
	sb.WriteString("- Next steps recommendations\n\n")

	sb.WriteString("### Step 2: Generate Context\n\n")
	sb.WriteString("**Purpose**: Create comprehensive context for Claude Code  \n")
	sb.WriteString("**Command**: `grctool evidence generate ET-XXXX --window {window}`  \n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Generate context for current window\n")
	sb.WriteString("grctool evidence generate ET-0047 --window 2025-Q4\n")
	sb.WriteString("```\n\n")
	sb.WriteString("**What Happens**:\n")
	sb.WriteString("- Creates `.context/generation-context.md` in evidence directory\n")
	sb.WriteString("- Auto-detects applicable tools from task keywords\n")
	sb.WriteString("- Gathers related controls and policies\n")
	sb.WriteString("- Scans for existing evidence from previous windows\n")
	sb.WriteString("- Loads source locations from config\n\n")

	sb.WriteString("**Context File Contents**:\n")
	sb.WriteString("```markdown\n")
	sb.WriteString("# Evidence Generation Context: ET-0047\n\n")
	sb.WriteString("## Task Details\n")
	sb.WriteString("- **Reference**: ET-0047\n")
	sb.WriteString("- **Name**: GitHub Repository Access Controls\n")
	sb.WriteString("- **Description**: Demonstrate repository access controls...\n")
	sb.WriteString("- **Collection Interval**: Quarterly\n")
	sb.WriteString("- **Window**: 2025-Q4\n\n")
	sb.WriteString("## Applicable Tools\n")
	sb.WriteString("Based on task keywords, these tools can help:\n")
	sb.WriteString("- `github-permissions` - Repository access analysis\n")
	sb.WriteString("- `github-security-features` - Security feature audit\n")
	sb.WriteString("- `github-deployment-access` - Deployment controls\n\n")
	sb.WriteString("## Related Controls\n")
	sb.WriteString("- CC6.8: Logical access controls\n")
	sb.WriteString("- AC-01: Access control policy\n\n")
	sb.WriteString("## Existing Evidence\n")
	sb.WriteString("Previous window (2025-Q3): 3 files found\n")
	sb.WriteString("- 01_github_permissions.csv\n")
	sb.WriteString("- 02_security_features.json\n")
	sb.WriteString("- 03_deployment_controls.json\n\n")
	sb.WriteString("## Next Steps\n")
	sb.WriteString("1. Read this context file\n")
	sb.WriteString("2. Run suggested tools\n")
	sb.WriteString("3. Save evidence with evidence-writer\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### Step 3: Work with Claude Code\n\n")
	sb.WriteString("**Purpose**: Use Claude Code to help collect evidence  \n\n")
	sb.WriteString("**Start Claude Code**:\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# In your terminal\n")
	sb.WriteString("claude\n")
	sb.WriteString("```\n\n")
	sb.WriteString("**Tell Claude**:\n")
	sb.WriteString("> \"Please read the evidence context at data/evidence/ET-0047_*/2025-Q4/.context/generation-context.md and help me collect the evidence\"\n\n")

	sb.WriteString("**Claude Will**:\n")
	sb.WriteString("1. Read the context file\n")
	sb.WriteString("2. Understand the requirements\n")
	sb.WriteString("3. Suggest which tools to run\n")
	sb.WriteString("4. Help you execute the commands\n")
	sb.WriteString("5. Guide you through saving evidence\n\n")

	sb.WriteString("### Step 4: Execute Tools\n\n")
	sb.WriteString("**Purpose**: Collect evidence using automated tools  \n\n")
	sb.WriteString("**With Claude's Guidance**:\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Claude will help you run commands like:\n")
	sb.WriteString("grctool tool github-permissions \\\n")
	sb.WriteString("  --repository org/repo \\\n")
	sb.WriteString("  --output-format csv > /tmp/permissions.csv\n\n")
	sb.WriteString("grctool tool github-security-features \\\n")
	sb.WriteString("  --repository org/repo > /tmp/security.json\n")
	sb.WriteString("```\n\n")

	sb.WriteString("**Claude Can Help With**:\n")
	sb.WriteString("- Selecting the right tool parameters\n")
	sb.WriteString("- Finding repository names from config\n")
	sb.WriteString("- Formatting output appropriately\n")
	sb.WriteString("- Troubleshooting errors\n\n")

	sb.WriteString("### Step 5: Save Evidence\n\n")
	sb.WriteString("**Purpose**: Save collected evidence with metadata tracking  \n")
	sb.WriteString("**Command**: `grctool tool evidence-writer`  \n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Save each evidence file\n")
	sb.WriteString("grctool tool evidence-writer \\\n")
	sb.WriteString("  --task-ref ET-0047 \\\n")
	sb.WriteString("  --title \"GitHub Permissions\" \\\n")
	sb.WriteString("  --file /tmp/permissions.csv\n\n")
	sb.WriteString("grctool tool evidence-writer \\\n")
	sb.WriteString("  --task-ref ET-0047 \\\n")
	sb.WriteString("  --title \"Security Features\" \\\n")
	sb.WriteString("  --file /tmp/security.json\n")
	sb.WriteString("```\n\n")

	sb.WriteString("**What Happens**:\n")
	sb.WriteString("- Evidence saved to `data/evidence/ET-0047_*/2025-Q4/`\n")
	sb.WriteString("- Files automatically numbered (01_, 02_, etc.)\n")
	sb.WriteString("- `.generation/metadata.yaml` created with:\n")
	sb.WriteString("  - SHA256 checksums\n")
	sb.WriteString("  - Timestamps\n")
	sb.WriteString("  - Tools used\n")
	sb.WriteString("  - Generation method (claude-code-assisted)\n\n")

	sb.WriteString("### Step 6: Validate Evidence\n\n")
	sb.WriteString("**Purpose**: Ensure evidence is complete and correctly formatted  \n")
	sb.WriteString("**Command**: `grctool evidence validate ET-XXXX`  \n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Validate specific task\n")
	sb.WriteString("grctool evidence validate ET-0047 --window 2025-Q4\n")
	sb.WriteString("```\n\n")

	sb.WriteString("**Validation Checks**:\n")
	sb.WriteString("- Files exist and are readable\n")
	sb.WriteString("- Checksums match metadata\n")
	sb.WriteString("- Required file types present\n")
	sb.WriteString("- File sizes within limits\n\n")

	sb.WriteString("### Step 7: Submit to Tugboat\n\n")
	sb.WriteString("**Purpose**: Upload evidence to Tugboat Logic for auditor review  \n")
	sb.WriteString("**Command**: `grctool evidence submit ET-XXXX`  \n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Submit evidence\n")
	sb.WriteString("grctool evidence submit ET-0047 \\\n")
	sb.WriteString("  --window 2025-Q4 \\\n")
	sb.WriteString("  --notes \"Q4 quarterly review\"\n")
	sb.WriteString("```\n\n")

	sb.WriteString("**What Happens**:\n")
	sb.WriteString("- Files uploaded via Tugboat Custom Evidence API\n")
	sb.WriteString("- `.submission/submission.yaml` created with:\n")
	sb.WriteString("  - Submission ID\n")
	sb.WriteString("  - Timestamp\n")
	sb.WriteString("  - Files submitted\n")
	sb.WriteString("  - Tugboat response status\n\n")

	// State Transitions
	sb.WriteString("---\n\n")
	sb.WriteString("## Evidence State Transitions\n\n")

	sb.WriteString("Evidence moves through these states:\n\n")

	sb.WriteString("```\n")
	sb.WriteString("no_evidence → generated → validated → submitted → accepted\n")
	sb.WriteString("                                          ↓\n")
	sb.WriteString("                                      rejected\n")
	sb.WriteString("                                          ↓\n")
	sb.WriteString("                                      generated (rework)\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### State Descriptions\n\n")

	sb.WriteString("| State | Description | Next Action |\n")
	sb.WriteString("|-------|-------------|-------------|\n")
	sb.WriteString("| **no_evidence** | No evidence files exist | Run `grctool evidence generate` |\n")
	sb.WriteString("| **generated** | Evidence created, not validated | Run `grctool evidence validate` |\n")
	sb.WriteString("| **validated** | Evidence validated, ready to submit | Run `grctool evidence submit` |\n")
	sb.WriteString("| **submitted** | Evidence sent to Tugboat | Wait for auditor review |\n")
	sb.WriteString("| **accepted** | Evidence approved by auditors | Done! |\n")
	sb.WriteString("| **rejected** | Evidence needs rework | Review feedback, regenerate |\n\n")

	// Common Patterns
	sb.WriteString("---\n\n")
	sb.WriteString("## Common Patterns\n\n")

	sb.WriteString("### Pattern 1: Single Evidence Task\n\n")
	sb.WriteString("When you need to collect evidence for one task:\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# 1. Generate context\n")
	sb.WriteString("grctool evidence generate ET-0047 --window 2025-Q4\n\n")
	sb.WriteString("# 2. Start Claude Code\n")
	sb.WriteString("claude\n\n")
	sb.WriteString("# 3. Tell Claude\n")
	sb.WriteString("# \"Read the context and help me collect evidence for ET-0047\"\n\n")
	sb.WriteString("# 4. Work interactively with Claude\n")
	sb.WriteString("# Claude will guide you through running tools and saving evidence\n\n")
	sb.WriteString("# 5. Validate and submit\n")
	sb.WriteString("grctool evidence validate ET-0047\n")
	sb.WriteString("grctool evidence submit ET-0047 --window 2025-Q4\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### Pattern 2: Multiple Related Tasks\n\n")
	sb.WriteString("When collecting evidence for multiple related tasks:\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# 1. Generate context for all tasks\n")
	sb.WriteString("grctool evidence generate ET-0047 --window 2025-Q4  # GitHub access\n")
	sb.WriteString("grctool evidence generate ET-0048 --window 2025-Q4  # GitHub workflows\n")
	sb.WriteString("grctool evidence generate ET-0049 --window 2025-Q4  # GitHub security\n\n")
	sb.WriteString("# 2. Start Claude Code\n")
	sb.WriteString("claude\n\n")
	sb.WriteString("# 3. Tell Claude\n")
	sb.WriteString("# \"I need to collect GitHub evidence for ET-0047, ET-0048, and ET-0049.\n")
	sb.WriteString("# Please help me efficiently collect all the evidence.\"\n\n")
	sb.WriteString("# Claude will help you:\n")
	sb.WriteString("# - Run tools once and reuse output\n")
	sb.WriteString("# - Save evidence to multiple tasks\n")
	sb.WriteString("# - Avoid duplicate work\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### Pattern 3: Regenerating Rejected Evidence\n\n")
	sb.WriteString("When evidence is rejected and needs rework:\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# 1. Check what was rejected\n")
	sb.WriteString("grctool status task ET-0047\n\n")
	sb.WriteString("# 2. Review feedback\n")
	sb.WriteString("cat data/evidence/ET-0047_*/2025-Q4/.submission/submission.yaml\n\n")
	sb.WriteString("# 3. Generate fresh context\n")
	sb.WriteString("grctool evidence generate ET-0047 --window 2025-Q4\n\n")
	sb.WriteString("# 4. Work with Claude to address feedback\n")
	sb.WriteString("claude\n")
	sb.WriteString("# \"The evidence for ET-0047 was rejected. Please help me regenerate it\n")
	sb.WriteString("# addressing the auditor feedback.\"\n")
	sb.WriteString("```\n\n")

	// Troubleshooting
	sb.WriteString("---\n\n")
	sb.WriteString("## Troubleshooting\n\n")

	sb.WriteString("### Issue: Context file not found\n\n")
	sb.WriteString("**Problem**: `grctool evidence generate` didn't create context  \n")
	sb.WriteString("**Solution**:\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Check if task exists\n")
	sb.WriteString("grctool tool evidence-task-details --task-ref ET-0047\n\n")
	sb.WriteString("# Try again with explicit window\n")
	sb.WriteString("grctool evidence generate ET-0047 --window 2025-Q4\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### Issue: Tool execution fails\n\n")
	sb.WriteString("**Problem**: Tool returns errors when collecting evidence  \n")
	sb.WriteString("**Solution**:\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Check tool help for required parameters\n")
	sb.WriteString("grctool tool <tool-name> --help\n\n")
	sb.WriteString("# Verify configuration\n")
	sb.WriteString("cat .grctool.yaml\n\n")
	sb.WriteString("# Test with --dry-run\n")
	sb.WriteString("grctool tool <tool-name> --dry-run\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### Issue: Evidence writer fails\n\n")
	sb.WriteString("**Problem**: Cannot save evidence files  \n")
	sb.WriteString("**Solution**:\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Check file exists\n")
	sb.WriteString("ls -lh /tmp/evidence.csv\n\n")
	sb.WriteString("# Verify task reference is correct\n")
	sb.WriteString("grctool tool evidence-task-list | grep ET-0047\n\n")
	sb.WriteString("# Check permissions on data directory\n")
	sb.WriteString("ls -ld data/evidence/\n")
	sb.WriteString("```\n\n")

	sb.WriteString("---\n\n")
	sb.WriteString("**Next Steps**: Consult `bulk-operations.md` for autonomous multi-task workflows.\n")

	return sb.String(), nil
}

// generateToolCapabilitiesDocs generates documentation about available tools
func generateToolCapabilitiesDocs(cfg *config.Config, version string) (string, error) {
	var sb strings.Builder

	sb.WriteString(writeDocHeader(
		"Tool Capabilities Reference",
		"Complete listing of all available tools and their purposes",
		version,
	))

	sb.WriteString("## Overview\n\n")
	sb.WriteString("GRCTool provides 30+ specialized tools for automated evidence collection. Each tool is designed for specific data sources and compliance requirements.\n\n")

	sb.WriteString("**Quick Command Reference**:\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# List all tools\n")
	sb.WriteString("grctool tool --help\n\n")
	sb.WriteString("# Get tool-specific help\n")
	sb.WriteString("grctool tool <tool-name> --help\n")
	sb.WriteString("```\n\n")

	sb.WriteString("---\n\n")

	// Evidence Collection Tools
	sb.WriteString("## Evidence Collection Tools\n\n")
	sb.WriteString("### Terraform Infrastructure Tools\n\n")

	sb.WriteString("#### `terraform-security-indexer`\n")
	sb.WriteString("**Purpose**: Fast infrastructure security scanning with indexed queries  \n")
	sb.WriteString("**Use When**: You need quick lookups of security configurations  \n")
	sb.WriteString("**Evidence Tasks**: Infrastructure security, access controls, encryption policies  \n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Query specific control mappings\n")
	sb.WriteString("grctool tool terraform-security-indexer --query-type control_mapping\n\n")
	sb.WriteString("# Search for specific security configurations\n")
	sb.WriteString("grctool tool terraform-security-indexer --query-type search --search-term \"encryption\"\n")
	sb.WriteString("```\n\n")

	sb.WriteString("#### `terraform-security-analyzer`\n")
	sb.WriteString("**Purpose**: Deep security analysis of Terraform configurations  \n")
	sb.WriteString("**Use When**: You need comprehensive security assessments  \n")
	sb.WriteString("**Evidence Tasks**: Security control validation, risk assessment  \n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Analyze all security domains\n")
	sb.WriteString("grctool tool terraform-security-analyzer --security-domain all\n\n")
	sb.WriteString("# Focus on specific domain\n")
	sb.WriteString("grctool tool terraform-security-analyzer --security-domain access_control\n")
	sb.WriteString("```\n\n")

	sb.WriteString("#### `terraform-hcl-parser`\n")
	sb.WriteString("**Purpose**: Parse and extract data from HCL/Terraform files  \n")
	sb.WriteString("**Use When**: You need structured data from Terraform code  \n")
	sb.WriteString("**Evidence Tasks**: Configuration documentation, resource inventory  \n\n")

	sb.WriteString("#### `terraform-snippets`\n")
	sb.WriteString("**Purpose**: Extract relevant code snippets from Terraform files  \n")
	sb.WriteString("**Use When**: You need code examples for auditors  \n")
	sb.WriteString("**Evidence Tasks**: Configuration examples, security settings  \n\n")

	sb.WriteString("#### `terraform-atmos-analyzer`\n")
	sb.WriteString("**Purpose**: Multi-environment stack analysis (Atmos framework)  \n")
	sb.WriteString("**Use When**: You use Atmos for environment management  \n")
	sb.WriteString("**Evidence Tasks**: Multi-environment consistency, configuration management  \n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Analyze all stacks\n")
	sb.WriteString("grctool tool terraform-atmos-analyzer --stack all\n")
	sb.WriteString("```\n\n")

	sb.WriteString("#### `terraform-query-interface`\n")
	sb.WriteString("**Purpose**: Query interface for indexed Terraform data  \n")
	sb.WriteString("**Use When**: You need to run custom queries on infrastructure  \n")
	sb.WriteString("**Evidence Tasks**: Custom security queries, compliance checks  \n\n")

	// GitHub Tools
	sb.WriteString("### GitHub Repository Tools\n\n")

	sb.WriteString("#### `github-permissions`\n")
	sb.WriteString("**Purpose**: Analyze repository and team permissions  \n")
	sb.WriteString("**Use When**: Collecting access control evidence  \n")
	sb.WriteString("**Evidence Tasks**: Repository access, team permissions, role assignments  \n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Matrix view of permissions\n")
	sb.WriteString("grctool tool github-permissions --repository owner/repo --output-format matrix\n\n")
	sb.WriteString("# JSON export for processing\n")
	sb.WriteString("grctool tool github-permissions --repository owner/repo --output-format json\n")
	sb.WriteString("```\n\n")

	sb.WriteString("#### `github-deployment-access`\n")
	sb.WriteString("**Purpose**: Analyze deployment environment access controls  \n")
	sb.WriteString("**Use When**: Documenting production access restrictions  \n")
	sb.WriteString("**Evidence Tasks**: Deployment controls, environment protection  \n\n")

	sb.WriteString("#### `github-security-features`\n")
	sb.WriteString("**Purpose**: Audit GitHub security features (branch protection, code scanning)  \n")
	sb.WriteString("**Use When**: Demonstrating security feature adoption  \n")
	sb.WriteString("**Evidence Tasks**: Branch protection, security scanning, vulnerability management  \n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Check security features\n")
	sb.WriteString("grctool tool github-security-features --repository owner/repo\n")
	sb.WriteString("```\n\n")

	sb.WriteString("#### `github-workflow-analyzer`\n")
	sb.WriteString("**Purpose**: Analyze CI/CD workflows and security controls  \n")
	sb.WriteString("**Use When**: Documenting build/deploy processes  \n")
	sb.WriteString("**Evidence Tasks**: CI/CD security, workflow controls, secrets management  \n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Analyze workflows\n")
	sb.WriteString("grctool tool github-workflow-analyzer --repository owner/repo\n")
	sb.WriteString("```\n\n")

	sb.WriteString("#### `github-review-analyzer`\n")
	sb.WriteString("**Purpose**: Analyze code review practices and enforcement  \n")
	sb.WriteString("**Use When**: Demonstrating code review requirements  \n")
	sb.WriteString("**Evidence Tasks**: Code review enforcement, approval requirements  \n\n")

	sb.WriteString("#### `github-enhanced`\n")
	sb.WriteString("**Purpose**: Enhanced GitHub data collection with richer metadata  \n")
	sb.WriteString("**Use When**: You need comprehensive GitHub evidence  \n")
	sb.WriteString("**Evidence Tasks**: Comprehensive repository analysis  \n\n")

	// Google Workspace
	sb.WriteString("### Google Workspace Tools\n\n")

	sb.WriteString("#### `google-workspace`\n")
	sb.WriteString("**Purpose**: Extract evidence from Google Drive, Docs, Sheets, Forms  \n")
	sb.WriteString("**Use When**: Policy documents, training records, reviews stored in Google Workspace  \n")
	sb.WriteString("**Evidence Tasks**: Policy documentation, training completion, access reviews  \n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Extract policy document\n")
	sb.WriteString("grctool tool google-workspace \\\n")
	sb.WriteString("  --document-id 1A2B3C4D5E6F7G8H9I0J \\\n")
	sb.WriteString("  --document-type docs\n\n")
	sb.WriteString("# Extract access review spreadsheet\n")
	sb.WriteString("grctool tool google-workspace \\\n")
	sb.WriteString("  --document-id 1K2L3M4N5O6P7Q8R9S0T \\\n")
	sb.WriteString("  --document-type sheets \\\n")
	sb.WriteString("  --sheet-range \"Q4 2025!A1:F100\"\n\n")
	sb.WriteString("# Extract training quiz responses\n")
	sb.WriteString("grctool tool google-workspace \\\n")
	sb.WriteString("  --document-id 1U2V3W4X5Y6Z7A8B9C0D \\\n")
	sb.WriteString("  --document-type forms\n")
	sb.WriteString("```\n\n")
	sb.WriteString("**Setup Required**: Google service account credentials (see `docs/01-User-Guide/google-workspace-setup.md`)\n\n")

	// Evidence Analysis Tools
	sb.WriteString("---\n\n")
	sb.WriteString("## Evidence Analysis & Discovery Tools\n\n")

	sb.WriteString("### `evidence-task-list`\n")
	sb.WriteString("**Purpose**: List all evidence tasks from Tugboat  \n")
	sb.WriteString("**Use When**: You need to see what evidence is required  \n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# List all tasks\n")
	sb.WriteString("grctool tool evidence-task-list\n\n")
	sb.WriteString("# Filter by status\n")
	sb.WriteString("grctool tool evidence-task-list --status pending\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### `evidence-task-details`\n")
	sb.WriteString("**Purpose**: Get detailed information about a specific evidence task  \n")
	sb.WriteString("**Use When**: You need to understand task requirements  \n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("grctool tool evidence-task-details --task-ref ET-0047\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### `evidence-relationships`\n")
	sb.WriteString("**Purpose**: Show relationships between tasks, controls, and policies  \n")
	sb.WriteString("**Use When**: Understanding control mappings  \n\n")

	sb.WriteString("### `policy-summary-generator`\n")
	sb.WriteString("**Purpose**: Generate summaries of policy documents  \n")
	sb.WriteString("**Use When**: Creating policy overviews for evidence  \n\n")

	sb.WriteString("### `control-summary-generator`\n")
	sb.WriteString("**Purpose**: Generate summaries of security controls  \n")
	sb.WriteString("**Use When**: Creating control documentation  \n\n")

	sb.WriteString("### `docs-reader`\n")
	sb.WriteString("**Purpose**: Read and parse synced documentation files  \n")
	sb.WriteString("**Use When**: You need to access policies/controls programmatically  \n\n")

	// Evidence Management Tools
	sb.WriteString("---\n\n")
	sb.WriteString("## Evidence Management Tools\n\n")

	sb.WriteString("### `evidence-writer`\n")
	sb.WriteString("**Purpose**: Write evidence files with automatic metadata tracking  \n")
	sb.WriteString("**Use When**: Saving generated evidence  \n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Write from file\n")
	sb.WriteString("grctool tool evidence-writer \\\n")
	sb.WriteString("  --task-ref ET-0047 \\\n")
	sb.WriteString("  --title \"GitHub Permissions\" \\\n")
	sb.WriteString("  --file permissions.csv\n\n")
	sb.WriteString("# Write from stdin\n")
	sb.WriteString("grctool tool github-permissions --repository org/repo | \\\n")
	sb.WriteString("  grctool tool evidence-writer \\\n")
	sb.WriteString("    --task-ref ET-0047 \\\n")
	sb.WriteString("    --title \"Permissions\" \\\n")
	sb.WriteString("    --format csv\n")
	sb.WriteString("```\n\n")
	sb.WriteString("**Automatic Metadata**: Creates `.generation/metadata.yaml` with checksums, timestamps, and tool tracking\n\n")

	sb.WriteString("### `evidence-generator`\n")
	sb.WriteString("**Purpose**: Generate evidence context for Claude Code assistance  \n")
	sb.WriteString("**Use When**: Starting new evidence generation  \n")
	sb.WriteString("**Note**: Use via `grctool evidence generate` command  \n\n")

	sb.WriteString("### `evidence-validator`\n")
	sb.WriteString("**Purpose**: Validate evidence completeness and format  \n")
	sb.WriteString("**Use When**: Before submitting evidence  \n")
	sb.WriteString("**Note**: Use via `grctool evidence validate` command  \n\n")

	sb.WriteString("### `tugboat-sync-wrapper`\n")
	sb.WriteString("**Purpose**: Sync data from Tugboat Logic API  \n")
	sb.WriteString("**Use When**: Updating local data from Tugboat  \n")
	sb.WriteString("**Note**: Use via `grctool sync` command  \n\n")

	// Storage and Utility Tools
	sb.WriteString("---\n\n")
	sb.WriteString("## Storage & Utility Tools\n\n")

	sb.WriteString("### `storage-read`\n")
	sb.WriteString("**Purpose**: Read files from GRCTool data directory  \n")
	sb.WriteString("**Use When**: Accessing synced data programmatically  \n\n")

	sb.WriteString("### `storage-write`\n")
	sb.WriteString("**Purpose**: Write files to GRCTool data directory  \n")
	sb.WriteString("**Use When**: Saving processed data  \n\n")

	sb.WriteString("### `name-generator`\n")
	sb.WriteString("**Purpose**: Generate standardized evidence file names  \n")
	sb.WriteString("**Use When**: Maintaining naming conventions  \n\n")

	sb.WriteString("### `grctool-run`\n")
	sb.WriteString("**Purpose**: Execute grctool commands programmatically  \n")
	sb.WriteString("**Use When**: Orchestrating multiple commands  \n\n")

	// Tool Selection Guide
	sb.WriteString("---\n\n")
	sb.WriteString("## Tool Selection Guide\n\n")

	sb.WriteString("### By Evidence Type\n\n")

	sb.WriteString("| Evidence Type | Recommended Tools |\n")
	sb.WriteString("|---------------|-------------------|\n")
	sb.WriteString("| **Infrastructure Security** | `terraform-security-indexer`, `terraform-security-analyzer` |\n")
	sb.WriteString("| **Access Controls** | `github-permissions`, `github-deployment-access` |\n")
	sb.WriteString("| **CI/CD Security** | `github-workflow-analyzer`, `github-security-features` |\n")
	sb.WriteString("| **Code Review** | `github-review-analyzer` |\n")
	sb.WriteString("| **Policy Documentation** | `google-workspace` (docs), `policy-summary-generator` |\n")
	sb.WriteString("| **Training Records** | `google-workspace` (forms, sheets) |\n")
	sb.WriteString("| **Access Reviews** | `google-workspace` (sheets), `github-permissions` |\n")
	sb.WriteString("| **Multi-Environment** | `terraform-atmos-analyzer` |\n\n")

	sb.WriteString("### By Automation Level\n\n")

	sb.WriteString("**Fully Automated** (can run without manual intervention):\n")
	sb.WriteString("- All Terraform tools\n")
	sb.WriteString("- All GitHub tools\n")
	sb.WriteString("- Evidence analysis tools\n\n")

	sb.WriteString("**Partially Automated** (needs configuration/authentication):\n")
	sb.WriteString("- `google-workspace` (requires service account setup)\n\n")

	sb.WriteString("**Manual** (requires manual document creation):\n")
	sb.WriteString("- Policy documents not in Google Workspace\n")
	sb.WriteString("- Screenshots and images\n")
	sb.WriteString("- Third-party vendor documentation\n\n")

	// Common Workflows
	sb.WriteString("---\n\n")
	sb.WriteString("## Common Evidence Collection Workflows\n\n")

	sb.WriteString("### Workflow 1: GitHub Access Control Evidence\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# 1. Generate context\n")
	sb.WriteString("grctool evidence generate ET-0047 --window 2025-Q4\n\n")
	sb.WriteString("# 2. Collect permissions\n")
	sb.WriteString("grctool tool github-permissions --repository org/repo > perms.csv\n\n")
	sb.WriteString("# 3. Collect security features\n")
	sb.WriteString("grctool tool github-security-features --repository org/repo > security.json\n\n")
	sb.WriteString("# 4. Save evidence\n")
	sb.WriteString("grctool tool evidence-writer --task-ref ET-0047 --title \"Permissions\" --file perms.csv\n")
	sb.WriteString("grctool tool evidence-writer --task-ref ET-0047 --title \"Security\" --file security.json\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### Workflow 2: Infrastructure Security Evidence\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# 1. Generate context\n")
	sb.WriteString("grctool evidence generate ET-0023 --window 2025-Q4\n\n")
	sb.WriteString("# 2. Run security indexer\n")
	sb.WriteString("grctool tool terraform-security-indexer --query-type all > infra-security.csv\n\n")
	sb.WriteString("# 3. Deep security analysis\n")
	sb.WriteString("grctool tool terraform-security-analyzer --security-domain all > security-analysis.json\n\n")
	sb.WriteString("# 4. Save evidence\n")
	sb.WriteString("grctool tool evidence-writer --task-ref ET-0023 --title \"Infrastructure\" --file infra-security.csv\n")
	sb.WriteString("grctool tool evidence-writer --task-ref ET-0023 --title \"Analysis\" --file security-analysis.json\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### Workflow 3: Policy Documentation Evidence\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# 1. Generate context\n")
	sb.WriteString("grctool evidence generate ET-0001 --window 2025-Q4\n\n")
	sb.WriteString("# 2. Extract policy from Google Docs\n")
	sb.WriteString("grctool tool google-workspace \\\n")
	sb.WriteString("  --document-id 1A2B3C4D \\\n")
	sb.WriteString("  --document-type docs > policy.json\n\n")
	sb.WriteString("# 3. Save evidence\n")
	sb.WriteString("grctool tool evidence-writer --task-ref ET-0001 --title \"Policy\" --file policy.json\n")
	sb.WriteString("```\n\n")

	// Tool Discovery
	sb.WriteString("---\n\n")
	sb.WriteString("## Tool Discovery & Help\n\n")

	sb.WriteString("### List Available Tools\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Show all tools\n")
	sb.WriteString("grctool tool --help\n\n")
	sb.WriteString("# List with descriptions\n")
	sb.WriteString("grctool tool list\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### Get Tool-Specific Help\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Detailed help for any tool\n")
	sb.WriteString("grctool tool <tool-name> --help\n\n")
	sb.WriteString("# Examples:\n")
	sb.WriteString("grctool tool github-permissions --help\n")
	sb.WriteString("grctool tool terraform-security-indexer --help\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### Test Tool Availability\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Check if tool configuration is valid\n")
	sb.WriteString("grctool tool <tool-name> --dry-run\n")
	sb.WriteString("```\n\n")

	sb.WriteString("---\n\n")
	sb.WriteString("**Next Steps**: Consult `evidence-workflow.md` for complete workflow integration.\n")

	return sb.String(), nil
}

// generateStatusCommandsDocs generates documentation about status commands
func generateStatusCommandsDocs(cfg *config.Config, version string) (string, error) {
	var sb strings.Builder

	sb.WriteString(writeDocHeader(
		"Status Commands Reference",
		"Usage guide for status commands and filtering options",
		version,
	))

	sb.WriteString("## Overview\n\n")
	sb.WriteString("The `grctool status` command provides visibility into evidence state across all tasks. Use it to understand what evidence needs to be collected, what's been generated, and what's ready for submission.\n\n")

	sb.WriteString("---\n\n")
	sb.WriteString("## Basic Commands\n\n")

	sb.WriteString("### Dashboard View\n")
	sb.WriteString("```bash\n")
	sb.WriteString("grctool status\n")
	sb.WriteString("```\n\n")
	sb.WriteString("Shows summary statistics:\n")
	sb.WriteString("- Evidence state counts (no_evidence, generated, submitted, etc.)\n")
	sb.WriteString("- Automation level breakdown\n")
	sb.WriteString("- Recent activity\n")
	sb.WriteString("- Next steps recommendations\n\n")

	sb.WriteString("### Task Details\n")
	sb.WriteString("```bash\n")
	sb.WriteString("grctool status task ET-0047\n")
	sb.WriteString("```\n\n")
	sb.WriteString("Shows detailed information for a specific task:\n")
	sb.WriteString("- Current state\n")
	sb.WriteString("- Evidence files (count, size, timestamps)\n")
	sb.WriteString("- Generation metadata\n")
	sb.WriteString("- Submission status\n")
	sb.WriteString("- Applicable tools\n\n")

	sb.WriteString("### Force Rescan\n")
	sb.WriteString("```bash\n")
	sb.WriteString("grctool status scan\n")
	sb.WriteString("```\n\n")
	sb.WriteString("Forces a fresh scan of evidence directories, bypassing cache.\n\n")

	sb.WriteString("---\n\n")
	sb.WriteString("## Filtering Options\n\n")

	sb.WriteString("### Filter by State\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Tasks with no evidence\n")
	sb.WriteString("grctool status --filter state=no_evidence\n\n")
	sb.WriteString("# Generated but not submitted\n")
	sb.WriteString("grctool status --filter state=generated\n\n")
	sb.WriteString("# Submitted tasks\n")
	sb.WriteString("grctool status --filter state=submitted\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### Filter by Automation Level\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Fully automated tasks\n")
	sb.WriteString("grctool status --filter automation=fully_automated\n\n")
	sb.WriteString("# Manual tasks\n")
	sb.WriteString("grctool status --filter automation=manual_only\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### Combined Filters\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Fully automated tasks with no evidence (highest priority)\n")
	sb.WriteString("grctool status \\\n")
	sb.WriteString("  --filter state=no_evidence \\\n")
	sb.WriteString("  --filter automation=fully_automated\n")
	sb.WriteString("```\n\n")

	sb.WriteString("---\n\n")
	sb.WriteString("## Understanding Output\n\n")

	sb.WriteString("### State Indicators\n")
	sb.WriteString("| State | Meaning | Action |\n")
	sb.WriteString("|-------|---------|--------|\n")
	sb.WriteString("| 🔴 no_evidence | No files | Generate evidence |\n")
	sb.WriteString("| 🟡 generated | Files exist | Validate |\n")
	sb.WriteString("| 🟢 validated | Ready | Submit |\n")
	sb.WriteString("| 📤 submitted | Sent to Tugboat | Wait for review |\n")
	sb.WriteString("| ✅ accepted | Approved | Done |\n")
	sb.WriteString("| ❌ rejected | Needs rework | Regenerate |\n\n")

	sb.WriteString("### Automation Levels\n")
	sb.WriteString("| Level | Meaning |\n")
	sb.WriteString("|-------|----------|\n")
	sb.WriteString("| 🤖 fully_automated | Can be fully automated with tools |\n")
	sb.WriteString("| ⚙️  partially_automated | Some automation possible |\n")
	sb.WriteString("| 👤 manual_only | Requires manual collection |\n")
	sb.WriteString("| ❓ unknown | Automation level not determined |\n\n")

	sb.WriteString("---\n\n")
	sb.WriteString("## Common Use Cases\n\n")

	sb.WriteString("### \"What needs to be done?\"\n")
	sb.WriteString("```bash\n")
	sb.WriteString("grctool status --filter state=no_evidence\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### \"What can Claude automate?\"\n")
	sb.WriteString("```bash\n")
	sb.WriteString("grctool status --filter automation=fully_automated --filter state=no_evidence\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### \"What's ready to submit?\"\n")
	sb.WriteString("```bash\n")
	sb.WriteString("grctool status --filter state=validated\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### \"Show me everything for ET-0047\"\n")
	sb.WriteString("```bash\n")
	sb.WriteString("grctool status task ET-0047\n")
	sb.WriteString("```\n\n")

	sb.WriteString("---\n\n")
	sb.WriteString("**Next Steps**: Use status output to prioritize evidence collection with `evidence-workflow.md` guidance.\n")

	return sb.String(), nil
}

// generateSubmissionProcessDocs generates documentation about evidence submission
func generateSubmissionProcessDocs(cfg *config.Config, version string) (string, error) {
	var sb strings.Builder

	sb.WriteString(writeDocHeader(
		"Evidence Submission Process",
		"Complete guide for submitting evidence to Tugboat Logic",
		version,
	))

	sb.WriteString("## Overview\n\n")
	sb.WriteString("Evidence submission uploads files to Tugboat Logic via the Custom Evidence Integration API. This document describes the setup, submission workflow, and troubleshooting.\n\n")

	sb.WriteString("---\n\n")
	sb.WriteString("## Setup Requirements\n\n")

	sb.WriteString("### 1. Generate Credentials in Tugboat UI\n\n")
	sb.WriteString("Navigate to: **Integrations** > **Custom Integrations**\n\n")
	sb.WriteString("1. Click **+** to add new integration\n")
	sb.WriteString("2. Enter account name and description\n")
	sb.WriteString("3. Click **Generate Password**\n")
	sb.WriteString("4. **IMPORTANT**: Save Username, Password, and X-API-KEY (cannot be recovered)\n\n")

	sb.WriteString("### 2. Generate Collector URLs\n\n")
	sb.WriteString("For each evidence task:\n")
	sb.WriteString("1. Click **+** to configure evidence service\n")
	sb.WriteString("2. Select scope and evidence task\n")
	sb.WriteString("3. Click **Copy URL** - this is your collector URL\n")
	sb.WriteString("4. Repeat for each task\n\n")

	sb.WriteString("### 3. Configure GRCTool\n\n")
	sb.WriteString("Add to `.grctool.yaml`:\n")
	sb.WriteString("```yaml\n")
	sb.WriteString("tugboat:\n")
	sb.WriteString("  username: \"your-username\"\n")
	sb.WriteString("  password: \"your-password\"\n")
	sb.WriteString("  collector_urls:\n")
	sb.WriteString("    \"ET-0001\": \"https://openapi.tugboatlogic.com/api/v0/evidence/collector/805/\"\n")
	sb.WriteString("    \"ET-0047\": \"https://openapi.tugboatlogic.com/api/v0/evidence/collector/806/\"\n")
	sb.WriteString("    # Add more task -> URL mappings\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### 4. Set API Key Environment Variable\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Direct\n")
	sb.WriteString("export TUGBOAT_API_KEY=\"your-x-api-key-from-step-1\"\n\n")
	sb.WriteString("# OR use 1Password\n")
	sb.WriteString("op run --env-file=\".env.tugboat\" -- grctool evidence submit ET-0001\n")
	sb.WriteString("```\n\n")

	sb.WriteString("---\n\n")
	sb.WriteString("## Submission Workflow\n\n")

	sb.WriteString("### Submit Evidence for Single Task\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# 1. Validate first\n")
	sb.WriteString("grctool evidence validate ET-0047 --window 2025-Q4\n\n")
	sb.WriteString("# 2. Preview submission (dry-run)\n")
	sb.WriteString("grctool evidence submit ET-0047 --window 2025-Q4 --dry-run\n\n")
	sb.WriteString("# 3. Submit\n")
	sb.WriteString("grctool evidence submit ET-0047 \\\n")
	sb.WriteString("  --window 2025-Q4 \\\n")
	sb.WriteString("  --notes \"Q4 quarterly review\"\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### Submit Manual Evidence\n\n")
	sb.WriteString("GRCTool can submit ANY files in the evidence folder:\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Place files in folder\n")
	sb.WriteString("cp ~/Documents/policy.pdf data/evidence/ET-0001_Policy/2025-Q4/\n")
	sb.WriteString("cp ~/Documents/training.xlsx data/evidence/ET-0001_Policy/2025-Q4/\n\n")
	sb.WriteString("# Submit\n")
	sb.WriteString("grctool evidence submit ET-0001 --window 2025-Q4 --skip-validation\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### Bulk Submission\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Submit all validated evidence\n")
	sb.WriteString("for task in ET-0047 ET-0048 ET-0049; do\n")
	sb.WriteString("  grctool evidence submit $task --window 2025-Q4\n")
	sb.WriteString("done\n")
	sb.WriteString("```\n\n")

	sb.WriteString("---\n\n")
	sb.WriteString("## Supported File Types\n\n")

	sb.WriteString("**Supported**: txt, csv, json, pdf, png, gif, jpg, jpeg, md, doc, docx, xls, xlsx, odt, ods  \n")
	sb.WriteString("**Max Size**: 20MB per file  \n")
	sb.WriteString("**Not Supported**: html, htm, js, exe, php  \n\n")

	sb.WriteString("---\n\n")
	sb.WriteString("## Troubleshooting\n\n")

	sb.WriteString("### Error: TUGBOAT_API_KEY not set\n")
	sb.WriteString("**Solution**: Export the environment variable  \n")
	sb.WriteString("```bash\n")
	sb.WriteString("export TUGBOAT_API_KEY=\"your-key\"\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### Error: credentials not configured\n")
	sb.WriteString("**Solution**: Add username/password to .grctool.yaml  \n\n")

	sb.WriteString("### Error: collector URL not configured\n")
	sb.WriteString("**Solution**: Add task mapping to tugboat.collector_urls  \n\n")

	sb.WriteString("### Error: file extension not supported\n")
	sb.WriteString("**Solution**: Check file type is in supported list  \n\n")

	sb.WriteString("### Error: file size exceeds maximum\n")
	sb.WriteString("**Solution**: File must be under 20MB  \n\n")

	sb.WriteString("### Error: 401 Unauthorized\n")
	sb.WriteString("**Solution**: Check username/password are correct  \n\n")

	sb.WriteString("### Error: 403 Forbidden\n")
	sb.WriteString("**Solution**: Verify API key is correct and has permissions  \n\n")

	sb.WriteString("---\n\n")
	sb.WriteString("**Next Steps**: After submission, monitor status with `grctool status` and wait for auditor review.\n")

	return sb.String(), nil
}

// generateBulkOperationsDocs generates documentation about autonomous bulk operations
func generateBulkOperationsDocs(cfg *config.Config, version string) (string, error) {
	var sb strings.Builder

	sb.WriteString(writeDocHeader(
		"Autonomous Bulk Evidence Generation",
		"Patterns and workflows for generating evidence across multiple tasks",
		version,
	))

	sb.WriteString("## Overview\n\n")
	sb.WriteString("**This is the most important document for autonomous operation.**\n\n")
	sb.WriteString("It describes how Claude Code can autonomously generate evidence for multiple tasks in a single session, coordinating tools, managing state, and ensuring completeness.\n\n")

	sb.WriteString("### Autonomous Mode Philosophy\n\n")
	sb.WriteString("When told to \"update all the evidence\" or \"generate evidence for all pending tasks\", Claude Code should:\n")
	sb.WriteString("1. Check current state with `grctool status`\n")
	sb.WriteString("2. Identify tasks that need evidence\n")
	sb.WriteString("3. Generate context for each task\n")
	sb.WriteString("4. Execute tools efficiently (reusing outputs where possible)\n")
	sb.WriteString("5. Save evidence with proper metadata\n")
	sb.WriteString("6. Track progress and report completion\n\n")

	sb.WriteString("---\n\n")

	// High-Level Workflow
	sb.WriteString("## Autonomous Bulk Generation Workflow\n\n")

	sb.WriteString("```\n")
	sb.WriteString("User: \"Update all the evidence\"\n")
	sb.WriteString("         |\n")
	sb.WriteString("         v\n")
	sb.WriteString("┌────────────────────┐\n")
	sb.WriteString("│  grctool status    │  (1) Check what needs work\n")
	sb.WriteString("└─────────┬──────────┘\n")
	sb.WriteString("          v\n")
	sb.WriteString("┌────────────────────┐\n")
	sb.WriteString("│  Filter tasks by   │  (2) Identify automatable tasks\n")
	sb.WriteString("│  automation level  │\n")
	sb.WriteString("└─────────┬──────────┘\n")
	sb.WriteString("          v\n")
	sb.WriteString("┌────────────────────┐\n")
	sb.WriteString("│  Generate context  │  (3) Create context for each task\n")
	sb.WriteString("│  for all tasks     │\n")
	sb.WriteString("└─────────┬──────────┘\n")
	sb.WriteString("          v\n")
	sb.WriteString("┌────────────────────┐\n")
	sb.WriteString("│  Group by tool     │  (4) Optimize execution order\n")
	sb.WriteString("│  requirements      │\n")
	sb.WriteString("└─────────┬──────────┘\n")
	sb.WriteString("          v\n")
	sb.WriteString("┌────────────────────┐\n")
	sb.WriteString("│  Execute tools     │  (5) Run tools once, reuse output\n")
	sb.WriteString("│  efficiently       │\n")
	sb.WriteString("└─────────┬──────────┘\n")
	sb.WriteString("          v\n")
	sb.WriteString("┌────────────────────┐\n")
	sb.WriteString("│  Save evidence for │  (6) Write evidence to all tasks\n")
	sb.WriteString("│  all applicable    │\n")
	sb.WriteString("│  tasks             │\n")
	sb.WriteString("└─────────┬──────────┘\n")
	sb.WriteString("          v\n")
	sb.WriteString("┌────────────────────┐\n")
	sb.WriteString("│  Report progress   │  (7) Show completion summary\n")
	sb.WriteString("└────────────────────┘\n")
	sb.WriteString("```\n\n")

	// Step by Step
	sb.WriteString("---\n\n")
	sb.WriteString("## Step-by-Step Autonomous Workflow\n\n")

	sb.WriteString("### Step 1: Assess Current State\n\n")
	sb.WriteString("**First Command**: Always start by checking status\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Check overall status\n")
	sb.WriteString("grctool status\n")
	sb.WriteString("```\n\n")

	sb.WriteString("**What to Look For**:\n")
	sb.WriteString("- Tasks in `no_evidence` state\n")
	sb.WriteString("- Tasks marked as `fully_automated` or `partially_automated`\n")
	sb.WriteString("- Current window (e.g., 2025-Q4)\n")
	sb.WriteString("- Total number of pending tasks\n\n")

	sb.WriteString("**Example Output Interpretation**:\n")
	sb.WriteString("```\n")
	sb.WriteString("Evidence Status Summary:\n")
	sb.WriteString("  No Evidence: 42 tasks\n")
	sb.WriteString("  Generated: 15 tasks\n")
	sb.WriteString("  Submitted: 8 tasks\n\n")
	sb.WriteString("Automation Levels:\n")
	sb.WriteString("  Fully Automated: 35 tasks   ← Focus here first\n")
	sb.WriteString("  Partially Automated: 12 tasks\n")
	sb.WriteString("  Manual Only: 10 tasks\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### Step 2: Filter and Prioritize\n\n")
	sb.WriteString("**Strategy**: Focus on fully automated tasks first\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# List fully automated tasks with no evidence\n")
	sb.WriteString("grctool status \\\n")
	sb.WriteString("  --filter state=no_evidence \\\n")
	sb.WriteString("  --filter automation=fully_automated\n")
	sb.WriteString("```\n\n")

	sb.WriteString("**Group by Tool Requirements**:\n")
	sb.WriteString("- **GitHub tasks**: ET-0047, ET-0048, ET-0049 (all use github tools)\n")
	sb.WriteString("- **Terraform tasks**: ET-0023, ET-0024, ET-0025 (all use terraform tools)\n")
	sb.WriteString("- **Google Workspace tasks**: ET-0001, ET-0002 (policy documents)\n\n")

	sb.WriteString("### Step 3: Generate Context for All Tasks\n\n")
	sb.WriteString("**Batch context generation**:\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Generate context for all GitHub tasks\n")
	sb.WriteString("for task in ET-0047 ET-0048 ET-0049; do\n")
	sb.WriteString("  grctool evidence generate $task --window 2025-Q4\n")
	sb.WriteString("done\n\n")
	sb.WriteString("# Generate context for all Terraform tasks\n")
	sb.WriteString("for task in ET-0023 ET-0024 ET-0025; do\n")
	sb.WriteString("  grctool evidence generate $task --window 2025-Q4\n")
	sb.WriteString("done\n")
	sb.WriteString("```\n\n")

	sb.WriteString("**Read All Contexts**: Claude should read all context files to understand requirements\n\n")

	sb.WriteString("### Step 4: Optimize Tool Execution\n\n")
	sb.WriteString("**Key Principle**: Run each tool once, reuse output for multiple tasks\n\n")

	sb.WriteString("**Example - GitHub Evidence**:\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Run github-permissions once for the main repository\n")
	sb.WriteString("grctool tool github-permissions \\\n")
	sb.WriteString("  --repository org/main-repo \\\n")
	sb.WriteString("  --output-format csv > /tmp/github-permissions.csv\n\n")
	sb.WriteString("# Run github-security-features once\n")
	sb.WriteString("grctool tool github-security-features \\\n")
	sb.WriteString("  --repository org/main-repo > /tmp/github-security.json\n\n")
	sb.WriteString("# Run github-workflow-analyzer once\n")
	sb.WriteString("grctool tool github-workflow-analyzer \\\n")
	sb.WriteString("  --repository org/main-repo > /tmp/github-workflows.json\n")
	sb.WriteString("```\n\n")

	sb.WriteString("**Now save to multiple tasks**:\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# ET-0047 needs permissions\n")
	sb.WriteString("grctool tool evidence-writer \\\n")
	sb.WriteString("  --task-ref ET-0047 \\\n")
	sb.WriteString("  --title \"Repository Permissions\" \\\n")
	sb.WriteString("  --file /tmp/github-permissions.csv\n\n")
	sb.WriteString("# ET-0048 needs workflows\n")
	sb.WriteString("grctool tool evidence-writer \\\n")
	sb.WriteString("  --task-ref ET-0048 \\\n")
	sb.WriteString("  --title \"CI/CD Workflows\" \\\n")
	sb.WriteString("  --file /tmp/github-workflows.json\n\n")
	sb.WriteString("# ET-0049 needs security features\n")
	sb.WriteString("grctool tool evidence-writer \\\n")
	sb.WriteString("  --task-ref ET-0049 \\\n")
	sb.WriteString("  --title \"Security Features\" \\\n")
	sb.WriteString("  --file /tmp/github-security.json\n\n")
	sb.WriteString("# Multiple tasks can use the same file!\n")
	sb.WriteString("grctool tool evidence-writer \\\n")
	sb.WriteString("  --task-ref ET-0047 \\\n")
	sb.WriteString("  --title \"Security Features\" \\\n")
	sb.WriteString("  --file /tmp/github-security.json\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### Step 5: Handle Terraform Tasks\n\n")
	sb.WriteString("**Run Terraform tools once**:\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Security indexer (fast, comprehensive)\n")
	sb.WriteString("grctool tool terraform-security-indexer \\\n")
	sb.WriteString("  --query-type all > /tmp/tf-security-index.csv\n\n")
	sb.WriteString("# Deep security analysis\n")
	sb.WriteString("grctool tool terraform-security-analyzer \\\n")
	sb.WriteString("  --security-domain all > /tmp/tf-security-analysis.json\n\n")
	sb.WriteString("# If using Atmos\n")
	sb.WriteString("grctool tool terraform-atmos-analyzer \\\n")
	sb.WriteString("  --stack all > /tmp/tf-atmos-stacks.json\n")
	sb.WriteString("```\n\n")

	sb.WriteString("**Save to all relevant tasks**:\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# ET-0023: Infrastructure Security\n")
	sb.WriteString("grctool tool evidence-writer --task-ref ET-0023 \\\n")
	sb.WriteString("  --title \"Security Index\" --file /tmp/tf-security-index.csv\n")
	sb.WriteString("grctool tool evidence-writer --task-ref ET-0023 \\\n")
	sb.WriteString("  --title \"Security Analysis\" --file /tmp/tf-security-analysis.json\n\n")
	sb.WriteString("# ET-0024: Configuration Management\n")
	sb.WriteString("grctool tool evidence-writer --task-ref ET-0024 \\\n")
	sb.WriteString("  --title \"Security Index\" --file /tmp/tf-security-index.csv\n")
	sb.WriteString("grctool tool evidence-writer --task-ref ET-0024 \\\n")
	sb.WriteString("  --title \"Atmos Stacks\" --file /tmp/tf-atmos-stacks.json\n\n")
	sb.WriteString("# ET-0025: Encryption Controls\n")
	sb.WriteString("grctool tool evidence-writer --task-ref ET-0025 \\\n")
	sb.WriteString("  --title \"Security Analysis\" --file /tmp/tf-security-analysis.json\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### Step 6: Progress Tracking\n\n")
	sb.WriteString("**Check progress periodically**:\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# See what's been completed\n")
	sb.WriteString("grctool status\n\n")
	sb.WriteString("# Focus on specific task\n")
	sb.WriteString("grctool status task ET-0047\n")
	sb.WriteString("```\n\n")

	sb.WriteString("**Report to User**:\n")
	sb.WriteString("> \"I've completed evidence generation for 8 tasks:\n")
	sb.WriteString("> - GitHub Access (ET-0047): ✅ 2 files\n")
	sb.WriteString("> - GitHub Workflows (ET-0048): ✅ 2 files\n")
	sb.WriteString("> - GitHub Security (ET-0049): ✅ 2 files\n")
	sb.WriteString("> - Infrastructure Security (ET-0023): ✅ 3 files\n")
	sb.WriteString("> ...\n")
	sb.WriteString("> \n")
	sb.WriteString("> Next: Run `grctool status` to see updated progress.\"\n\n")

	// Decision Making
	sb.WriteString("---\n\n")
	sb.WriteString("## Decision Making for Autonomous Operation\n\n")

	sb.WriteString("### When to Run a Tool\n\n")
	sb.WriteString("**Decision Tree**:\n")
	sb.WriteString("```\n")
	sb.WriteString("Is this tool mentioned in ANY task context?\n")
	sb.WriteString("  ├─ YES → Run it once, save to all applicable tasks\n")
	sb.WriteString("  └─ NO → Skip\n\n")
	sb.WriteString("Do multiple tasks need this tool?\n")
	sb.WriteString("  ├─ YES → Run once, save to all tasks\n")
	sb.WriteString("  └─ NO → Run for single task\n\n")
	sb.WriteString("Is tool configuration available?\n")
	sb.WriteString("  ├─ YES → Execute\n")
	sb.WriteString("  └─ NO → Skip task, report to user\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### Which Tasks to Automate\n\n")
	sb.WriteString("**Priority Order**:\n")
	sb.WriteString("1. **Fully Automated + No Evidence** → Highest priority\n")
	sb.WriteString("2. **Fully Automated + Generated** → Validate and submit\n")
	sb.WriteString("3. **Partially Automated + No Evidence** → Attempt if tools configured\n")
	sb.WriteString("4. **Manual Only** → Skip, report to user\n\n")

	sb.WriteString("### Error Handling\n\n")
	sb.WriteString("**When a tool fails**:\n")
	sb.WriteString("1. Log the error\n")
	sb.WriteString("2. Skip that specific task\n")
	sb.WriteString("3. Continue with other tasks\n")
	sb.WriteString("4. Report all failures at the end\n\n")

	sb.WriteString("**Example**:\n")
	sb.WriteString("> \"⚠️ Could not collect evidence for ET-0047: github-permissions tool failed (repository not found).\n")
	sb.WriteString("> \n")
	sb.WriteString("> ✅ Successfully completed: ET-0048, ET-0049, ET-0023, ET-0024, ET-0025 (5 tasks)\"\n\n")

	// Complete Example
	sb.WriteString("---\n\n")
	sb.WriteString("## Complete Autonomous Example\n\n")

	sb.WriteString("**User Request**: \"Update all the evidence\"\n\n")

	sb.WriteString("**Claude's Execution**:\n\n")

	sb.WriteString("```bash\n")
	sb.WriteString("# Step 1: Check status\n")
	sb.WriteString("grctool status --filter state=no_evidence\n\n")
	sb.WriteString("# Step 2: Generate context for all pending tasks\n")
	sb.WriteString("grctool evidence generate ET-0047 --window 2025-Q4  # GitHub Access\n")
	sb.WriteString("grctool evidence generate ET-0048 --window 2025-Q4  # GitHub Workflows\n")
	sb.WriteString("grctool evidence generate ET-0049 --window 2025-Q4  # GitHub Security\n")
	sb.WriteString("grctool evidence generate ET-0023 --window 2025-Q4  # Infrastructure\n")
	sb.WriteString("grctool evidence generate ET-0024 --window 2025-Q4  # Config Mgmt\n\n")
	sb.WriteString("# Step 3: Execute GitHub tools (run once)\n")
	sb.WriteString("grctool tool github-permissions --repository org/repo > /tmp/gh-perms.csv\n")
	sb.WriteString("grctool tool github-workflow-analyzer --repository org/repo > /tmp/gh-workflows.json\n")
	sb.WriteString("grctool tool github-security-features --repository org/repo > /tmp/gh-security.json\n\n")
	sb.WriteString("# Step 4: Execute Terraform tools (run once)\n")
	sb.WriteString("grctool tool terraform-security-indexer > /tmp/tf-index.csv\n")
	sb.WriteString("grctool tool terraform-security-analyzer --security-domain all > /tmp/tf-analysis.json\n\n")
	sb.WriteString("# Step 5: Save evidence to all applicable tasks\n")
	sb.WriteString("# GitHub tasks\n")
	sb.WriteString("grctool tool evidence-writer --task-ref ET-0047 --title \"Permissions\" --file /tmp/gh-perms.csv\n")
	sb.WriteString("grctool tool evidence-writer --task-ref ET-0047 --title \"Security\" --file /tmp/gh-security.json\n\n")
	sb.WriteString("grctool tool evidence-writer --task-ref ET-0048 --title \"Workflows\" --file /tmp/gh-workflows.json\n\n")
	sb.WriteString("grctool tool evidence-writer --task-ref ET-0049 --title \"Security\" --file /tmp/gh-security.json\n\n")
	sb.WriteString("# Terraform tasks\n")
	sb.WriteString("grctool tool evidence-writer --task-ref ET-0023 --title \"Security\" --file /tmp/tf-index.csv\n")
	sb.WriteString("grctool tool evidence-writer --task-ref ET-0023 --title \"Analysis\" --file /tmp/tf-analysis.json\n\n")
	sb.WriteString("grctool tool evidence-writer --task-ref ET-0024 --title \"Security\" --file /tmp/tf-index.csv\n\n")
	sb.WriteString("# Step 6: Verify completion\n")
	sb.WriteString("grctool status\n")
	sb.WriteString("```\n\n")

	sb.WriteString("**Claude's Report**:\n")
	sb.WriteString("> \"✅ Completed bulk evidence generation:\n")
	sb.WriteString("> \n")
	sb.WriteString("> **GitHub Evidence**:\n")
	sb.WriteString("> - ET-0047 (Access Controls): 2 files generated\n")
	sb.WriteString("> - ET-0048 (CI/CD Workflows): 1 file generated\n")
	sb.WriteString("> - ET-0049 (Security Features): 1 file generated\n")
	sb.WriteString("> \n")
	sb.WriteString("> **Infrastructure Evidence**:\n")
	sb.WriteString("> - ET-0023 (Security): 2 files generated\n")
	sb.WriteString("> - ET-0024 (Config Mgmt): 1 file generated\n")
	sb.WriteString("> \n")
	sb.WriteString("> **Summary**: 5 tasks completed, 7 evidence files generated\n")
	sb.WriteString("> \n")
	sb.WriteString("> **Next Steps**:\n")
	sb.WriteString("> 1. Review evidence with: `grctool status task ET-XXXX`\n")
	sb.WriteString("> 2. Validate all: `grctool evidence validate --all`\n")
	sb.WriteString("> 3. Submit when ready: `grctool evidence submit --all`\"\n\n")

	// Best Practices
	sb.WriteString("---\n\n")
	sb.WriteString("## Best Practices\n\n")

	sb.WriteString("### DO:\n")
	sb.WriteString("✅ Check status before starting  \n")
	sb.WriteString("✅ Generate context for all tasks first  \n")
	sb.WriteString("✅ Group tasks by tool requirements  \n")
	sb.WriteString("✅ Run each tool once, reuse output  \n")
	sb.WriteString("✅ Track progress and report completion  \n")
	sb.WriteString("✅ Handle errors gracefully  \n")
	sb.WriteString("✅ Provide detailed completion summary  \n\n")

	sb.WriteString("### DON'T:\n")
	sb.WriteString("❌ Run the same tool multiple times unnecessarily  \n")
	sb.WriteString("❌ Skip status checks  \n")
	sb.WriteString("❌ Fail silently on errors  \n")
	sb.WriteString("❌ Process tasks one-by-one (be efficient!)  \n")
	sb.WriteString("❌ Forget to report progress to user  \n\n")

	// Performance Tips
	sb.WriteString("---\n\n")
	sb.WriteString("## Performance Optimization\n\n")

	sb.WriteString("### Tool Reuse Strategy\n\n")
	sb.WriteString("**Scenario**: 10 tasks all need `github-permissions`\n\n")
	sb.WriteString("**❌ Inefficient** (10 tool executions):\n")
	sb.WriteString("```bash\n")
	sb.WriteString("grctool tool github-permissions ... > ET-0001.csv\n")
	sb.WriteString("grctool tool github-permissions ... > ET-0002.csv\n")
	sb.WriteString("# ... 10 times!\n")
	sb.WriteString("```\n\n")

	sb.WriteString("**✅ Efficient** (1 tool execution):\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Run once\n")
	sb.WriteString("grctool tool github-permissions ... > /tmp/permissions.csv\n\n")
	sb.WriteString("# Save to all 10 tasks\n")
	sb.WriteString("for task in ET-0001 ET-0002 ... ET-0010; do\n")
	sb.WriteString("  grctool tool evidence-writer --task-ref $task \\\n")
	sb.WriteString("    --title \"Permissions\" --file /tmp/permissions.csv\n")
	sb.WriteString("done\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### Parallel vs Sequential\n\n")
	sb.WriteString("**Tools can run in parallel** if they're independent:\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Run these concurrently\n")
	sb.WriteString("grctool tool github-permissions ... > /tmp/gh-perms.csv &\n")
	sb.WriteString("grctool tool terraform-security-indexer > /tmp/tf-index.csv &\n")
	sb.WriteString("wait  # Wait for both to finish\n")
	sb.WriteString("```\n\n")

	sb.WriteString("---\n\n")
	sb.WriteString("**Next Steps**: Refer back to `tool-capabilities.md` for detailed tool usage, and `evidence-workflow.md` for single-task patterns.\n")

	return sb.String(), nil
}
