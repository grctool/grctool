# Evidence Collection Workflow - Technical Specification

**Version:** 1.0
**Date:** 2025-10-10
**Purpose:** Define the complete evidence collection workflow from task discovery to evidence generation
**Status:** 🚧 **IN DEVELOPMENT** - Specification for implementation

## Executive Summary

This specification defines the **core evidence collection workflow** for GRCTool, establishing how users discover evidence tasks, understand what needs to be collected, and execute collection using automated tools.

### Design Principles

1. **Transparency**: Users must clearly see what evidence is needed and where it comes from
2. **Automation-First**: Prefer automated collection over manual work
3. **Integration-Aware**: Respect existing Tugboat Logic integrations
4. **Plan-Driven**: Collection plans guide execution and track progress
5. **Tool Composition**: Multiple specialized tools combine to create complete evidence

### Current State vs. Target State

| Component | Current State | Target State |
|-----------|--------------|--------------|
| **Task Listing** | ✅ Filtering by status, framework, priority | ⚠️ Add expected sources, collection method visibility |
| **Task Details** | ✅ Requirements, controls, metadata | ⚠️ Add source mapping, tool specifications |
| **Collection Plan** | ✅ Tracks entries, completeness | ❌ Needs source expectations, tool orchestration |
| **Evidence Execution** | ⚠️ Manual tool coordination | ❌ Needs automated plan execution |
| **Source Classification** | ❌ Missing | ❌ Needs manual/automated/tugboat taxonomy |

## Workflow Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Evidence Collection Lifecycle                 │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
        ┌────────────────────────────────────────┐
        │  Stage 1: Task Discovery & Status      │
        │  Command: grctool evidence list        │
        │  Output: Tasks + Expected Sources      │
        └────────────────┬───────────────────────┘
                        │
                        ▼
        ┌────────────────────────────────────────┐
        │  Stage 2: Collection Plan Review       │
        │  Command: grctool evidence plan ET-001 │
        │  Output: Strategy + Tool Specs         │
        └────────────────┬───────────────────────┘
                        │
                        ▼
        ┌────────────────────────────────────────┐
        │  Stage 3: Evidence Collection          │
        │  Command: grctool evidence collect     │
        │  Output: Evidence Files + Plan Update  │
        └────────────────┬───────────────────────┘
                        │
                        ▼
        ┌────────────────────────────────────────┐
        │  Stage 4: Review & Validation          │
        │  Command: grctool evidence review      │
        │  Output: Quality Assessment            │
        └────────────────────────────────────────┘
```

---

## Stage 1: Task Discovery & Status

### Purpose
Enable users to discover what evidence needs to be collected and understand the expected sources for each task.

### Command Specification

```bash
grctool evidence list [flags]
```

#### New Flags

```bash
--with-sources          # Show expected evidence sources for each task
--collection-method     # Filter by method: manual, automated, hybrid, tugboat-integrated
--missing-only          # Show only tasks with incomplete evidence
```

### Expected Output Format

```
Evidence Tasks (Filtered: 15 of 150 total)

┌─────────┬────────────────────────────┬──────────┬─────────────┬────────────────────────────┐
│ Task ID │ Name                       │ Status   │ Completeness│ Expected Sources           │
├─────────┼────────────────────────────┼──────────┼─────────────┼────────────────────────────┤
│ ET-001  │ Access Control Evidence    │ PENDING  │ 0%          │ 🤖 terraform-iam           │
│         │ (CC6.1 Access Security)    │          │             │ 🤖 github-permissions      │
│         │                            │          │             │ ✋ manual-policy-doc        │
├─────────┼────────────────────────────┼──────────┼─────────────┼────────────────────────────┤
│ ET-047  │ Repository Access Controls │ TUGBOAT  │ 100%        │ ✅ tugboat-integration     │
│         │ (CC6.1, CC6.2)             │          │             │ (already collected)        │
├─────────┼────────────────────────────┼──────────┼─────────────┼────────────────────────────┤
│ ET-071  │ CI/CD Security Evidence    │ PROGRESS │ 60%         │ 🤖 github-workflow-analyzer│
│         │ (CC7.2 Change Management)  │          │             │ 🤖 terraform-scanner       │
│         │                            │          │             │ ✋ manual-runbook          │
│         │                            │          │             │   (3 of 5 collected)       │
└─────────┴────────────────────────────┴──────────┴─────────────┴────────────────────────────┘

Legend:
  🤖 Automated tool collection
  ✋ Manual evidence required
  ✅ Tugboat integration (complete)

Status:
  PENDING   - No evidence collected
  PROGRESS  - Partially complete
  TUGBOAT   - Handled by Tugboat integration
  COMPLETE  - All evidence collected
```

### Source Classification Taxonomy

Each evidence source is classified into one of these types:

| Source Type | Icon | Description | Implementation |
|-------------|------|-------------|----------------|
| **Automated** | 🤖 | Tool-based collection | Call grctool tool |
| **Manual** | ✋ | Requires human input | Placeholder file with instructions |
| **Tugboat-Integrated** | ✅ | Already in Tugboat via AEC | Mark as complete, no action needed |
| **Hybrid** | 🔄 | Tool provides data, human reviews/enhances | Tool generates draft, manual completion |

### Data Model Changes

**File**: `internal/domain/evidence_task.go`

```go
// EvidenceTask extension
type EvidenceTask struct {
    // ... existing fields ...

    // NEW: Expected sources for this evidence task
    ExpectedSources []ExpectedSource `json:"expected_sources,omitempty"`

    // NEW: Overall collection method
    CollectionMethod string `json:"collection_method,omitempty"` // "manual", "automated", "hybrid", "tugboat-integrated"
}

// NEW: Expected source specification
type ExpectedSource struct {
    Type           string   `json:"type"`            // "automated", "manual", "tugboat-integrated"
    ToolName       string   `json:"tool_name"`       // e.g., "terraform-scanner", "github-permissions"
    Description    string   `json:"description"`     // Human-readable description
    ToolParameters map[string]interface{} `json:"tool_parameters,omitempty"` // Tool-specific params
    Priority       string   `json:"priority"`        // "required", "optional", "nice-to-have"
    Status         string   `json:"status"`          // "pending", "collected", "skipped"
}
```

### Acceptance Criteria

**Stage 1 Complete When:**

- [ ] `grctool evidence list --with-sources` shows expected sources for each task
- [ ] Source icons (🤖 ✋ ✅) clearly indicate collection method
- [ ] Filtering by `--collection-method automated` shows only tool-based tasks
- [ ] Task completeness percentage reflects collected vs. expected sources
- [ ] Tugboat-integrated tasks show "TUGBOAT" status and 100% complete
- [ ] Manual tasks clearly indicate what human input is needed

---

## Stage 2: Collection Plan Review

### Purpose
Display the collection strategy and execution plan for a specific evidence task, showing what tools will be called and what manual work is required.

### Command Specification

```bash
grctool evidence plan <task-ref> [flags]
```

#### Flags

```bash
--format markdown|json|yaml   # Output format (default: markdown)
--show-tool-params            # Show detailed tool parameters
--update-strategy             # Regenerate collection strategy using AI
```

### Expected Output Format

```markdown
# Evidence Collection Plan - ET-001 Access Control Evidence

**Window**: 2025-Q4
**Collection Interval**: Quarterly
**Last Updated**: 2025-10-10T14:23:00Z
**Status**: Not Started (0% Complete)

## Task Requirements

**Description**: Demonstrate that access controls are properly implemented across infrastructure and version control systems, showing least privilege, role-based access, and regular access reviews.

**Controls Addressed**:
- CC6.1: Logical Access Security Measures
- CC6.2: Authentication and Authorization

**Framework**: SOC2 Type II

## Collection Strategy

This evidence task requires a **hybrid approach** combining automated infrastructure analysis with manual policy documentation:

1. **Automated Analysis** (70% of evidence)
   - Terraform IAM configurations showing role definitions
   - GitHub repository permissions and branch protection
   - Access logs demonstrating regular reviews

2. **Manual Documentation** (30% of evidence)
   - Written access control policy
   - Evidence of policy review and approval
   - Screenshots of access review process

### Reasoning

The automated tools can extract technical implementation details from infrastructure-as-code and version control, providing objective evidence of controls in place. Manual evidence is needed to demonstrate the policy framework and governance processes that guide these technical controls.

## Expected Sources & Tools

### 1. Terraform IAM Analysis 🤖 AUTOMATED

**Tool**: `terraform-scanner`
**Purpose**: Extract IAM role definitions, policies, and group assignments
**Status**: ⏳ Pending

**Tool Parameters**:
```json
{
  "analysis_type": "iam_security",
  "include_patterns": ["**/iam*.tf", "**/aws-team-roles/**"],
  "extract_roles": true,
  "extract_policies": true,
  "output_format": "markdown"
}
```

**Expected Output**: `01_terraform_iam_roles.md`

---

### 2. GitHub Repository Permissions 🤖 AUTOMATED

**Tool**: `github-permissions`
**Purpose**: Extract repository access controls, team permissions, and branch protection rules
**Status**: ⏳ Pending

**Tool Parameters**:
```json
{
  "repository": "organization/main-app",
  "include_teams": true,
  "include_branch_protection": true,
  "output_format": "markdown"
}
```

**Expected Output**: `02_github_access_controls.md`

---

### 3. Access Control Policy Documentation ✋ MANUAL

**Source Type**: Manual documentation
**Purpose**: Provide written policy describing access control standards and procedures
**Status**: ⏳ Pending

**Instructions**:
1. Locate the organization's Access Control Policy document
2. Export as PDF or Markdown
3. Verify policy includes:
   - Role definitions and responsibilities
   - Least privilege principles
   - Access review procedures
   - Access revocation process
4. Save as: `03_access_control_policy.md`

**Expected Output**: `03_access_control_policy.md`

---

### 4. Tugboat AEC Integration ✅ TUGBOAT-INTEGRATED

**Integration**: Tugboat Logic Automated Evidence Collection
**Purpose**: Automated user access reports from identity provider
**Status**: ✅ Complete (handled by Tugboat)

**Note**: This evidence source is automatically collected by Tugboat Logic's existing integration. No action required.

---

## Evidence Inventory

No evidence collected yet.

## Gaps and Next Steps

| Gap | Priority | Remediation |
|-----|----------|-------------|
| Terraform IAM analysis not run | High | Execute: `grctool evidence collect ET-001` |
| GitHub permissions not extracted | High | Execute: `grctool evidence collect ET-001` |
| Access control policy not uploaded | Medium | Manually locate and upload policy document |

## Completeness Assessment

- Overall Completeness: **0%**
- Status: **Not Started**
- Evidence Files: 0 of 3 expected
- Automated Sources: 0 of 2 complete
- Manual Sources: 0 of 1 complete
- Tugboat Sources: 1 of 1 complete

## Next Actions

**To collect automated evidence**:
```bash
grctool evidence collect ET-001
```

**To add manual evidence**:
```bash
grctool evidence add ET-001 \
  --title "Access Control Policy" \
  --file /path/to/policy.pdf \
  --source-type manual
```
```

### Collection Plan Schema

**File**: `internal/tools/collection_plan.go` (enhancement)

```go
// CollectionPlan extension
type CollectionPlan struct {
    // ... existing fields ...

    // NEW: Detailed source specifications
    SourceSpecs []SourceSpecification `json:"source_specs" yaml:"source_specs"`
}

// NEW: Detailed specification for each evidence source
type SourceSpecification struct {
    Index          int                    `json:"index" yaml:"index"`
    Name           string                 `json:"name" yaml:"name"`
    Type           string                 `json:"type" yaml:"type"` // "automated", "manual", "tugboat-integrated"
    ToolName       string                 `json:"tool_name,omitempty" yaml:"tool_name,omitempty"`
    ToolParameters map[string]interface{} `json:"tool_parameters,omitempty" yaml:"tool_parameters,omitempty"`
    Instructions   string                 `json:"instructions,omitempty" yaml:"instructions,omitempty"` // For manual sources
    ExpectedOutput string                 `json:"expected_output" yaml:"expected_output"` // Filename
    Status         string                 `json:"status" yaml:"status"` // "pending", "collected", "skipped"
    Priority       string                 `json:"priority" yaml:"priority"` // "required", "optional"
}
```

### Acceptance Criteria

**Stage 2 Complete When:**

- [ ] `grctool evidence plan ET-001` displays complete collection strategy
- [ ] Plan shows which tools will be called with exact parameters
- [ ] Manual evidence requirements include step-by-step instructions
- [ ] Tugboat-integrated sources are clearly marked as complete
- [ ] Plan includes reasoning for why each source is needed
- [ ] Expected output filenames are specified for each source
- [ ] Gap analysis shows what's missing and how to remediate
- [ ] Next actions provide exact commands to execute

---

## Stage 3: Evidence Collection Execution

### Purpose
Execute the collection plan by calling specified tools, generating evidence files, and updating plan status.

### Command Specification

```bash
grctool evidence collect <task-ref> [flags]
```

#### Flags

```bash
--dry-run                  # Show what would be collected without executing
--sources <source-names>   # Collect only specific sources (comma-separated)
--skip-manual              # Skip manual sources, only run automated tools
--parallel                 # Run tools in parallel where possible
--output-dir <path>        # Override default evidence output directory
```

### Execution Flow

```
1. Load collection plan for task
2. Validate plan has source specifications
3. Filter sources based on flags (--skip-manual, --sources)
4. For each source in plan:
   a. Check if already collected (skip if complete)
   b. If automated source:
      - Resolve tool from registry
      - Build tool parameters from source spec
      - Execute tool
      - Capture output
      - Write evidence file
      - Update source status
   c. If manual source:
      - Create placeholder file with instructions
      - Mark as "pending manual input"
   d. If tugboat-integrated:
      - Skip (already complete)
      - Log confirmation
5. Regenerate collection plan with updates
6. Display summary of collected evidence
```

### Expected Output

```bash
$ grctool evidence collect ET-001

Evidence Collection: ET-001 Access Control Evidence
Window: 2025-Q4
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

[1/4] 🤖 Terraform IAM Analysis
  Tool: terraform-scanner
  Status: ⏳ Executing...
  ✅ Complete (2.3s)
  Output: evidence/ET-001_Access_Control_Evidence/2025-Q4/01_terraform_iam_roles.md
  Size: 15.2 KB

[2/4] 🤖 GitHub Repository Permissions
  Tool: github-permissions
  Status: ⏳ Executing...
  ✅ Complete (1.8s)
  Output: evidence/ET-001_Access_Control_Evidence/2025-Q4/02_github_access_controls.md
  Size: 8.7 KB

[3/4] ✋ Access Control Policy Documentation
  Type: Manual
  Status: ⏳ Creating placeholder...
  ⚠️  Manual Input Required
  Output: evidence/ET-001_Access_Control_Evidence/2025-Q4/03_access_control_policy.md
  Instructions: See placeholder file for details

[4/4] ✅ Tugboat AEC Integration
  Type: Tugboat-Integrated
  Status: ✅ Already Complete
  Note: Handled by existing Tugboat Logic integration

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Collection Summary:
  ✅ Automated: 2 of 2 complete (100%)
  ⚠️  Manual: 0 of 1 complete (0%)
  ✅ Tugboat: 1 of 1 complete (100%)

Overall Progress: 75% complete (3 of 4 sources)

Next Steps:
  1. Review automated evidence in evidence/ET-001_Access_Control_Evidence/2025-Q4/
  2. Complete manual evidence: See 03_access_control_policy.md for instructions
  3. Run 'grctool evidence review ET-001' to validate completeness

Collection plan updated: evidence/ET-001_Access_Control_Evidence/2025-Q4/collection_plan.md
```

### Evidence Directory Structure

```
evidence/
└── ET-001_Access_Control_Evidence/
    └── 2025-Q4/
        ├── collection_plan.md          # Human-readable plan
        ├── collection_plan_metadata.yaml  # Machine-readable plan
        ├── 01_terraform_iam_roles.md   # Automated evidence
        ├── 02_github_access_controls.md # Automated evidence
        └── 03_access_control_policy.md  # Manual placeholder
```

### Manual Evidence Placeholder Format

**File**: `03_access_control_policy.md`

```markdown
# 📋 Manual Evidence Required: Access Control Policy Documentation

**Status**: ⏳ PENDING MANUAL INPUT
**Priority**: REQUIRED
**Source Type**: Manual Documentation

## Instructions

This evidence requires manual collection. Please follow these steps:

### What to Provide

Provide the organization's Access Control Policy document that describes:

1. **Role Definitions**: Standard roles and their responsibilities
2. **Least Privilege Principles**: How access is granted based on need
3. **Access Review Procedures**: Regular review schedules and processes
4. **Access Revocation Process**: How access is removed when no longer needed

### How to Collect

**Option 1: Upload Existing Policy Document**
```bash
grctool evidence add ET-001 \
  --title "Access Control Policy" \
  --file /path/to/Access_Control_Policy.pdf \
  --source-type manual \
  --replace 03_access_control_policy.md
```

**Option 2: Create Markdown Documentation**

Replace this file with your policy documentation in Markdown format. Include:

- Policy name and version
- Effective date and last review date
- Policy statements and requirements
- Roles and responsibilities
- Approval signatures

### Validation Criteria

Your evidence will be complete when it includes:

- [ ] Policy document is current (reviewed within last 12 months)
- [ ] Approval signatures or electronic approval records
- [ ] Role definitions clearly stated
- [ ] Access review procedures documented
- [ ] Access revocation process described

---

**After completing**: Run `grctool evidence review ET-001` to validate

**Questions?**: See collection plan for more context: `collection_plan.md`
```

### Tool Orchestration Logic

**File**: `internal/services/evidence/collector.go` (new)

```go
// EvidenceCollector orchestrates evidence collection based on plans
type EvidenceCollector struct {
    config      *config.Config
    logger      logger.Logger
    toolRegistry *tools.Registry
    planManager  *tools.CollectionPlanManager
}

// CollectEvidence executes collection plan for a task
func (ec *EvidenceCollector) CollectEvidence(ctx context.Context, taskRef string, opts CollectionOptions) (*CollectionResult, error) {
    // 1. Load task and plan
    task := ec.loadTask(taskRef)
    plan := ec.planManager.LoadOrCreatePlan(task, ...)

    // 2. Filter sources based on options
    sources := ec.filterSources(plan.SourceSpecs, opts)

    // 3. Execute collection
    results := []SourceResult{}
    for _, source := range sources {
        result := ec.collectSource(ctx, source, task, plan)
        results = append(results, result)
    }

    // 4. Update plan
    ec.planManager.SavePlan(plan, ...)

    // 5. Return summary
    return ec.buildCollectionResult(results), nil
}

func (ec *EvidenceCollector) collectSource(ctx context.Context, source SourceSpecification, task *domain.EvidenceTask, plan *CollectionPlan) SourceResult {
    switch source.Type {
    case "automated":
        return ec.collectAutomatedSource(ctx, source, task, plan)
    case "manual":
        return ec.createManualPlaceholder(source, task, plan)
    case "tugboat-integrated":
        return ec.markTugboatComplete(source)
    default:
        return SourceResult{Status: "error", Error: "unknown source type"}
    }
}

func (ec *EvidenceCollector) collectAutomatedSource(ctx context.Context, source SourceSpecification, task *domain.EvidenceTask, plan *CollectionPlan) SourceResult {
    // 1. Get tool from registry
    tool, err := ec.toolRegistry.GetTool(source.ToolName)
    if err != nil {
        return SourceResult{Status: "error", Error: err.Error()}
    }

    // 2. Build parameters from source spec
    params := source.ToolParameters
    params["task_ref"] = task.ReferenceID

    // 3. Execute tool
    output, evidenceSource, err := tool.Execute(ctx, params)
    if err != nil {
        return SourceResult{Status: "error", Error: err.Error()}
    }

    // 4. Write evidence file
    evidencePath := ec.buildEvidencePath(task, plan.Window, source.ExpectedOutput)
    ec.writeEvidenceFile(evidencePath, output)

    // 5. Update plan entry
    entry := EvidenceEntry{
        Filename: source.ExpectedOutput,
        Title: source.Name,
        Source: source.ToolName,
        Status: "complete",
        CollectedAt: time.Now(),
    }
    ec.planManager.AddEvidenceEntry(plan, entry)

    return SourceResult{Status: "success", FilePath: evidencePath, Size: len(output)}
}
```

### Acceptance Criteria

**Stage 3 Complete When:**

- [ ] `grctool evidence collect ET-001` executes all automated tools
- [ ] Each tool is called with parameters from plan source specifications
- [ ] Evidence files are written to correct directory with sequential numbering
- [ ] Manual sources create placeholder files with clear instructions
- [ ] Tugboat-integrated sources are skipped with confirmation message
- [ ] Collection plan is updated with status of each source
- [ ] Summary shows progress breakdown (automated/manual/tugboat)
- [ ] `--dry-run` flag shows what would be collected without executing
- [ ] `--sources` flag allows selective collection
- [ ] `--skip-manual` flag skips manual placeholders
- [ ] Errors in one tool don't block collection of other sources

---

## Stage 4: Review & Validation

### Purpose
Validate collected evidence for completeness, quality, and compliance requirements.

### Command Specification

```bash
grctool evidence review <task-ref> [flags]
```

### Expected Output

```bash
$ grctool evidence review ET-001

Evidence Review: ET-001 Access Control Evidence
Window: 2025-Q4
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✅ AUTOMATED EVIDENCE (2 of 2 complete)

  ✅ 01_terraform_iam_roles.md
     Source: terraform-scanner
     Size: 15.2 KB
     Controls: CC6.1, CC6.2
     Quality: ⭐⭐⭐⭐⭐ Excellent
     - Contains 15 IAM role definitions
     - Shows least privilege configuration
     - Includes policy attachments

  ✅ 02_github_access_controls.md
     Source: github-permissions
     Size: 8.7 KB
     Controls: CC6.1, CC6.2
     Quality: ⭐⭐⭐⭐⭐ Excellent
     - Team permissions documented
     - Branch protection rules present
     - Required reviewers configured

⚠️  MANUAL EVIDENCE (0 of 1 complete)

  ⏳ 03_access_control_policy.md
     Source: Manual
     Status: PLACEHOLDER - Manual input required
     Action: Upload policy document or create markdown content

✅ TUGBOAT-INTEGRATED (1 of 1 complete)

  ✅ User Access Reports
     Source: Tugboat AEC
     Status: Complete via existing integration

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Overall Assessment:
  Completeness: 75% (3 of 4 sources)
  Quality Score: 4.5/5.0
  Audit Readiness: ⚠️  NEEDS ATTENTION

Gaps:
  ❌ Manual policy documentation required
  ⚠️  No policy review date in automated evidence

Recommendations:
  1. Upload access control policy document to complete manual evidence
  2. Ensure policy shows review/approval within last 12 months
  3. Consider adding screenshots of access review process

Next Steps:
  grctool evidence add ET-001 --file /path/to/policy.pdf --source-type manual
```

### Acceptance Criteria

**Stage 4 Complete When:**

- [ ] Review shows completeness breakdown by source type
- [ ] Quality assessment evaluates each evidence file
- [ ] Gap identification highlights missing or incomplete sources
- [ ] Recommendations provide actionable next steps
- [ ] Audit readiness score indicates if evidence is ready for submission

---

## Complete End-to-End Example

### Scenario: Quarterly Access Control Evidence (ET-001)

**Compliance Manager Goal**: Prepare SOC 2 access control evidence for Q4 2025 audit

#### Step 1: Discover Tasks

```bash
$ grctool sync
Synchronizing from Tugboat Logic...
✅ Downloaded 150 evidence tasks
✅ Downloaded 42 controls
✅ Downloaded 28 policies

$ grctool evidence list --framework soc2 --status pending --with-sources

Evidence Tasks (Filtered: 15 of 150 total)

┌─────────┬────────────────────────────┬──────────┬─────────────┬────────────────────────────┐
│ Task ID │ Name                       │ Status   │ Completeness│ Expected Sources           │
├─────────┼────────────────────────────┼──────────┼─────────────┼────────────────────────────┤
│ ET-001  │ Access Control Evidence    │ PENDING  │ 0%          │ 🤖 terraform-scanner       │
│         │ (CC6.1 Access Security)    │          │             │ 🤖 github-permissions      │
│         │                            │          │             │ ✋ manual-policy-doc        │
│         │                            │          │             │ ✅ tugboat-user-access     │
...
```

**Decision**: ET-001 is a good candidate - mix of automated and manual sources

#### Step 2: Review Collection Plan

```bash
$ grctool evidence plan ET-001

# Evidence Collection Plan - ET-001 Access Control Evidence
...
(See Stage 2 output above for complete plan)
...

## Expected Sources & Tools

### 1. Terraform IAM Analysis 🤖 AUTOMATED
Tool: terraform-scanner
Parameters: {...}

### 2. GitHub Repository Permissions 🤖 AUTOMATED
Tool: github-permissions
Parameters: {...}

### 3. Access Control Policy Documentation ✋ MANUAL
Instructions: Locate and upload policy document...

### 4. Tugboat AEC Integration ✅ TUGBOAT-INTEGRATED
Status: Complete (no action needed)
```

**Decision**: Plan looks good, proceed with collection

#### Step 3: Execute Collection

```bash
$ grctool evidence collect ET-001

Evidence Collection: ET-001 Access Control Evidence
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

[1/4] 🤖 Terraform IAM Analysis
  ✅ Complete (2.3s)
  Output: evidence/ET-001_Access_Control_Evidence/2025-Q4/01_terraform_iam_roles.md

[2/4] 🤖 GitHub Repository Permissions
  ✅ Complete (1.8s)
  Output: evidence/ET-001_Access_Control_Evidence/2025-Q4/02_github_access_controls.md

[3/4] ✋ Access Control Policy Documentation
  ⚠️  Manual Input Required
  Output: evidence/ET-001_Access_Control_Evidence/2025-Q4/03_access_control_policy.md

[4/4] ✅ Tugboat AEC Integration
  ✅ Already Complete

Overall Progress: 75% complete (3 of 4 sources)
```

#### Step 4: Complete Manual Evidence

```bash
$ grctool evidence add ET-001 \
  --title "Access Control Policy" \
  --file ~/Documents/Policies/Access_Control_Policy_v2.1.pdf \
  --source-type manual \
  --replace 03_access_control_policy.md

✅ Evidence added successfully
   File: evidence/ET-001_Access_Control_Evidence/2025-Q4/03_access_control_policy.md
   Size: 156 KB
   Format: PDF (converted to markdown reference)
```

#### Step 5: Review Completeness

```bash
$ grctool evidence review ET-001

Overall Assessment:
  Completeness: 100% (4 of 4 sources)
  Quality Score: 4.8/5.0
  Audit Readiness: ✅ READY FOR SUBMISSION

All evidence sources collected successfully!
```

#### Step 6: Package for Auditor

```bash
$ grctool evidence package ET-001 --format pdf --output ~/Desktop/

✅ Evidence package created:
   ~/Desktop/ET-001_Access_Control_Evidence_2025-Q4.pdf
   Contains: 4 evidence files, collection plan, control mapping
```

**Result**: Complete evidence package ready for SOC 2 audit in < 5 minutes

---

## Implementation Phases

### Phase 1: Data Model Updates (Week 1)

**Goal**: Extend data models to support source specifications

**Tasks**:
- [ ] Add `ExpectedSources` field to `EvidenceTask`
- [ ] Create `ExpectedSource` struct with tool parameters
- [ ] Add `SourceSpecification` to `CollectionPlan`
- [ ] Update JSON/YAML serialization
- [ ] Write unit tests for new models

**Deliverables**:
- Updated `internal/domain/evidence_task.go`
- Updated `internal/tools/collection_plan.go`
- Migration script for existing plans

### Phase 2: Plan Management (Week 2)

**Goal**: Implement plan viewing and source specification

**Tasks**:
- [ ] Implement `grctool evidence plan` command
- [ ] Create plan rendering templates (markdown/JSON/YAML)
- [ ] Add source specification parsing from task metadata
- [ ] Implement `--update-strategy` flag to regenerate plans
- [ ] Write integration tests

**Deliverables**:
- `cmd/evidence_plan.go`
- Plan templates in `internal/templates/`
- Integration tests in `test/integration/`

### Phase 3: Collection Execution (Week 3-4)

**Goal**: Implement automated collection based on plans

**Tasks**:
- [ ] Create `EvidenceCollector` service
- [ ] Implement tool orchestration logic
- [ ] Add manual placeholder generation
- [ ] Implement progress tracking and reporting
- [ ] Add `--dry-run`, `--sources`, `--skip-manual` flags
- [ ] Write functional tests

**Deliverables**:
- `internal/services/evidence/collector.go`
- Updated `cmd/evidence.go` with `collect` subcommand
- Functional tests in `test/functional/`

### Phase 4: Integration & Documentation (Week 5)

**Goal**: End-to-end testing and user documentation

**Tasks**:
- [ ] Create end-to-end test scenarios
- [ ] Write user guide for evidence collection workflow
- [ ] Create video walkthrough (optional)
- [ ] Performance testing and optimization
- [ ] Update README with new commands

**Deliverables**:
- End-to-end tests in `test/e2e/`
- User guide in `docs/01-User-Guide/evidence-collection.md`
- Updated README.md

---

## Testing Strategy

### Unit Tests

**Coverage Targets**: >90% for new code

- Data model serialization/deserialization
- Evidence window calculation
- Source filtering logic
- Filename generation

### Integration Tests

**Coverage**: Each tool integration

- Tool parameter building from source specs
- Tool execution and output capture
- Plan updates after collection
- Error handling for failed tools

### Functional Tests

**Coverage**: Complete workflows

- Full collection workflow (ET-001 example)
- Selective collection with `--sources`
- Dry run mode
- Manual placeholder generation
- Tugboat integration skipping

### End-to-End Tests

**Coverage**: User scenarios

- Quarterly audit preparation scenario
- Mixed automated/manual evidence
- Multi-task collection
- Error recovery

---

## Edge Cases & Error Handling

### Edge Case 1: Tool Not Available

**Scenario**: Collection plan specifies `aws-scanner` but tool not installed

**Handling**:
- Skip source with warning
- Create placeholder file explaining missing tool
- Continue with other sources
- Report in summary

### Edge Case 2: Tool Execution Failure

**Scenario**: `github-permissions` fails due to API rate limit

**Handling**:
- Log detailed error
- Create placeholder with retry instructions
- Mark source as "failed" not "complete"
- Continue with other sources
- Suggest retry strategy in summary

### Edge Case 3: Manual Source Already Complete

**Scenario**: User manually uploaded evidence before running `collect`

**Handling**:
- Detect existing evidence file
- Skip placeholder creation
- Mark source as "complete"
- Validate file is not a placeholder

### Edge Case 4: Tugboat Integration Disabled

**Scenario**: Task marked as tugboat-integrated but AEC is disabled

**Handling**:
- Detect AEC status from task metadata
- Warn user that integration is not active
- Suggest alternative collection method
- Mark as "needs attention" in plan

### Edge Case 5: Empty Tool Output

**Scenario**: Tool executes successfully but returns no data

**Handling**:
- Write evidence file with "no results" message
- Mark as "partial" not "complete"
- Add gap to plan explaining empty results
- Suggest reasons (no resources found, incorrect parameters)

---

## Success Metrics

### User Experience

- **Time to First Evidence**: < 5 minutes from `sync` to collected evidence
- **Automation Rate**: > 70% of evidence collected without manual intervention
- **Error Recovery**: Users can complete workflow despite 1-2 tool failures
- **Clarity**: Users understand what's needed after reading plan

### Technical Performance

- **Collection Speed**: < 30 seconds per automated source
- **Plan Generation**: < 2 seconds to render plan
- **Parallel Execution**: 3x faster with `--parallel` for independent sources
- **Resource Usage**: < 500MB memory for typical collection

### Quality

- **Completeness**: 95% of tasks have all automated sources collected
- **Accuracy**: < 5% false positives in automated evidence
- **Audit Acceptance**: > 90% of generated evidence accepted by auditors
- **Test Coverage**: > 85% for evidence collection code

---

## Future Enhancements

### v1.1: AI-Assisted Plan Generation

- Use Claude AI to analyze task requirements and suggest tools
- Auto-generate source specifications from task descriptions
- Recommend manual evidence based on framework requirements

### v1.2: Continuous Collection

- Schedule automated collection (daily/weekly/monthly)
- Incremental updates to evidence
- Change detection and re-collection triggers

### v1.3: Evidence Templates

- Pre-built templates for common evidence types
- Organization-specific customization
- Template marketplace/sharing

### v1.4: Multi-Framework Support

- Map single evidence to multiple frameworks (SOC2 + ISO27001)
- Framework-specific formatting
- Cross-framework gap analysis

---

## Appendices

### Appendix A: Source Type Decision Tree

```
Is evidence source in Tugboat AEC?
├─ YES → Mark as "tugboat-integrated", skip collection
└─ NO → Can evidence be collected by a grctool tool?
    ├─ YES → Is human review/enhancement needed?
    │   ├─ YES → Mark as "hybrid" (tool + manual)
    │   └─ NO → Mark as "automated"
    └─ NO → Mark as "manual"
```

### Appendix B: Tool Parameter Schema

Each automated source must specify tool parameters:

```json
{
  "tool_name": "terraform-scanner",
  "tool_parameters": {
    "analysis_type": "iam_security",
    "include_patterns": ["**/iam*.tf"],
    "output_format": "markdown",
    "extract_roles": true
  }
}
```

### Appendix C: Evidence Naming Convention

**Pattern**: `{sequence}_{sanitized_title}.{ext}`

**Examples**:
- `01_terraform_iam_roles.md`
- `02_github_access_controls.md`
- `03_access_control_policy.pdf`

**Rules**:
- Sequence: 2-digit zero-padded
- Title: lowercase, alphanumeric + underscore
- Extension: based on format (.md, .csv, .json, .pdf)

---

## References

- [Terraform Tools Specification](terraform-tools-specification.md)
- [GitHub Tools Specification](github-tools-specification.md)
- [User Stories](../helix/01-frame/user-stories/stories.md)
- [Collection Plan Implementation](../../internal/tools/collection_plan.go)
- [Evidence Writer Tool](../../internal/tools/evidence_writer.go)

---

**Document Status**: 📝 DRAFT for Implementation
**Next Review**: After Phase 1 completion
**Feedback**: Create GitHub issue or discuss in team sync
