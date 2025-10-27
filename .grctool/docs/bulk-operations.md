# Autonomous Bulk Evidence Generation

> Patterns and workflows for generating evidence across multiple tasks

---

**Generated**: 2025-10-27 16:56:21 EDT  
**GRCTool Version**: dev  
**Documentation Version**: dev  

---

## Overview

**This is the most important document for autonomous operation.**

It describes how Claude Code can autonomously generate evidence for multiple tasks in a single session, coordinating tools, managing state, and ensuring completeness.

### Autonomous Mode Philosophy

When told to "update all the evidence" or "generate evidence for all pending tasks", Claude Code should:
1. Check current state with `grctool status`
2. Identify tasks that need evidence
3. Generate context for each task
4. Execute tools efficiently (reusing outputs where possible)
5. Save evidence with proper metadata
6. Track progress and report completion

---

## Autonomous Bulk Generation Workflow

```
User: "Update all the evidence"
         |
         v
┌────────────────────┐
│  grctool status    │  (1) Check what needs work
└─────────┬──────────┘
          v
┌────────────────────┐
│  Filter tasks by   │  (2) Identify automatable tasks
│  automation level  │
└─────────┬──────────┘
          v
┌────────────────────┐
│  Generate context  │  (3) Create context for each task
│  for all tasks     │
└─────────┬──────────┘
          v
┌────────────────────┐
│  Group by tool     │  (4) Optimize execution order
│  requirements      │
└─────────┬──────────┘
          v
┌────────────────────┐
│  Execute tools     │  (5) Run tools once, reuse output
│  efficiently       │
└─────────┬──────────┘
          v
┌────────────────────┐
│  Save evidence for │  (6) Write evidence to all tasks
│  all applicable    │
│  tasks             │
└─────────┬──────────┘
          v
┌────────────────────┐
│  Report progress   │  (7) Show completion summary
└────────────────────┘
```

---

## Step-by-Step Autonomous Workflow

### Step 1: Assess Current State

**First Command**: Always start by checking status

```bash
# Check overall status
grctool status
```

**What to Look For**:
- Tasks in `no_evidence` state
- Tasks marked as `fully_automated` or `partially_automated`
- Current window (e.g., 2025-Q4)
- Total number of pending tasks

**Example Output Interpretation**:
```
Evidence Status Summary:
  No Evidence: 42 tasks
  Generated: 15 tasks
  Submitted: 8 tasks

Automation Levels:
  Fully Automated: 35 tasks   ← Focus here first
  Partially Automated: 12 tasks
  Manual Only: 10 tasks
```

### Step 2: Filter and Prioritize

**Strategy**: Focus on fully automated tasks first

```bash
# List fully automated tasks with no evidence
grctool status \
  --filter state=no_evidence \
  --filter automation=fully_automated
```

**Group by Tool Requirements**:
- **GitHub tasks**: ET-0047, ET-0048, ET-0049 (all use github tools)
- **Terraform tasks**: ET-0023, ET-0024, ET-0025 (all use terraform tools)
- **Google Workspace tasks**: ET-0001, ET-0002 (policy documents)

### Step 3: Generate Assembly Contexts (Bulk)

**Purpose**: Create comprehensive assembly contexts for all pending evidence tasks at once

**Command**:
```bash
grctool evidence generate --all --window 2025-Q4
```

**What Happens:**
- Loads all pending evidence tasks from `evidence_tasks/` directory
- For each task:
  - Generates comprehensive assembly context via `prompt-assembler`
  - Creates Claude workflow instructions
  - Selects appropriate evidence template
  - Saves three assembly files to `.context/` directory
- Shows progress: `[1/10] ET-0001 - Task Name ✅`
- Displays summary with success/failure counts

**Output Structure:**

Each task gets a complete assembly context:
```
evidence/ET-XXXX_{task_name}/{window}/.context/
├── assembly-prompt.md        # 200+ line comprehensive prompt
├── claude-instructions.md    # Workflow guidance
├── evidence-template.md      # Pre-structured report template
└── tool_outputs/             # Empty (for optional tool data)
```

**Example Output:**
```
Found 10 pending task(s)

Generating comprehensive assembly contexts:
  [1/10] ET-0001 - Access Control Documentation ✅
  [2/10] ET-0008 - Workstation Firewall Settings ✅
  [3/10] ET-0047 - GitHub Repository Permissions ✅
  ...

============================================================
Assembly Context Generation Complete
  ✅ Successful: 10 tasks

Next steps:
  1. Ask Claude Code to help with evidence generation
  2. Claude will read assembly prompts and guide you through:
     - Running applicable tools
     - Using evidence-generator for synthesis
     - Creating comprehensive reports

  Example: 'Help me generate all pending evidence'
```

**With Tool Data Collection:**

To pre-collect tool data for all tasks:
```bash
grctool evidence generate --all --window 2025-Q4 --with-tool-data
```

This will execute applicable tools for each task and save outputs automatically.

**Benefits:**
- Batch preparation of all evidence materials
- Consistent comprehensive prompts across all tasks
- Ready for Claude-assisted evidence generation
- Templates pre-structured based on task categories

**Manual Batch Generation (Alternative):**

If you prefer to generate contexts for specific task groups:

```bash
# Generate context for all GitHub tasks
for task in ET-0047 ET-0048 ET-0049; do
  grctool evidence generate $task --window 2025-Q4
done

# Generate context for all Terraform tasks
for task in ET-0023 ET-0024 ET-0025; do
  grctool evidence generate $task --window 2025-Q4
done
```

**Read All Contexts**: Claude should read all assembly-prompt.md files to understand requirements

### Step 4: Optimize Tool Execution

**Key Principle**: Run each tool once, reuse output for multiple tasks

**Example - GitHub Evidence**:
```bash
# Run github-permissions once for the main repository
grctool tool github-permissions \
  --repository org/main-repo \
  --output-format csv > /tmp/github-permissions.csv

# Run github-security-features once
grctool tool github-security-features \
  --repository org/main-repo > /tmp/github-security.json

# Run github-workflow-analyzer once
grctool tool github-workflow-analyzer \
  --repository org/main-repo > /tmp/github-workflows.json
```

**Now save to multiple tasks**:
```bash
# ET-0047 needs permissions
grctool tool evidence-writer \
  --task-ref ET-0047 \
  --title "Repository Permissions" \
  --file /tmp/github-permissions.csv

# ET-0048 needs workflows
grctool tool evidence-writer \
  --task-ref ET-0048 \
  --title "CI/CD Workflows" \
  --file /tmp/github-workflows.json

# ET-0049 needs security features
grctool tool evidence-writer \
  --task-ref ET-0049 \
  --title "Security Features" \
  --file /tmp/github-security.json

# Multiple tasks can use the same file!
grctool tool evidence-writer \
  --task-ref ET-0047 \
  --title "Security Features" \
  --file /tmp/github-security.json
```

### Step 5: Handle Terraform Tasks

**Run Terraform tools once**:
```bash
# Security indexer (fast, comprehensive)
grctool tool terraform-security-indexer \
  --query-type all > /tmp/tf-security-index.csv

# Deep security analysis
grctool tool terraform-security-analyzer \
  --security-domain all > /tmp/tf-security-analysis.json

# If using Atmos
grctool tool terraform-atmos-analyzer \
  --stack all > /tmp/tf-atmos-stacks.json
```

**Save to all relevant tasks**:
```bash
# ET-0023: Infrastructure Security
grctool tool evidence-writer --task-ref ET-0023 \
  --title "Security Index" --file /tmp/tf-security-index.csv
grctool tool evidence-writer --task-ref ET-0023 \
  --title "Security Analysis" --file /tmp/tf-security-analysis.json

# ET-0024: Configuration Management
grctool tool evidence-writer --task-ref ET-0024 \
  --title "Security Index" --file /tmp/tf-security-index.csv
grctool tool evidence-writer --task-ref ET-0024 \
  --title "Atmos Stacks" --file /tmp/tf-atmos-stacks.json

# ET-0025: Encryption Controls
grctool tool evidence-writer --task-ref ET-0025 \
  --title "Security Analysis" --file /tmp/tf-security-analysis.json
```

### Step 6: Progress Tracking

**Check progress periodically**:
```bash
# See what's been completed
grctool status

# Focus on specific task
grctool status task ET-0047
```

**Report to User**:
> "I've completed evidence generation for 8 tasks:
> - GitHub Access (ET-0047): ✅ 2 files
> - GitHub Workflows (ET-0048): ✅ 2 files
> - GitHub Security (ET-0049): ✅ 2 files
> - Infrastructure Security (ET-0023): ✅ 3 files
> ...
> 
> Next: Run `grctool status` to see updated progress."

---

## Decision Making for Autonomous Operation

### When to Run a Tool

**Decision Tree**:
```
Is this tool mentioned in ANY task context?
  ├─ YES → Run it once, save to all applicable tasks
  └─ NO → Skip

Do multiple tasks need this tool?
  ├─ YES → Run once, save to all tasks
  └─ NO → Run for single task

Is tool configuration available?
  ├─ YES → Execute
  └─ NO → Skip task, report to user
```

### Which Tasks to Automate

**Priority Order**:
1. **Fully Automated + No Evidence** → Highest priority
2. **Fully Automated + Generated** → Validate and submit
3. **Partially Automated + No Evidence** → Attempt if tools configured
4. **Manual Only** → Skip, report to user

### Error Handling

**When a tool fails**:
1. Log the error
2. Skip that specific task
3. Continue with other tasks
4. Report all failures at the end

**Example**:
> "⚠️ Could not collect evidence for ET-0047: github-permissions tool failed (repository not found).
> 
> ✅ Successfully completed: ET-0048, ET-0049, ET-0023, ET-0024, ET-0025 (5 tasks)"

---

## Complete Autonomous Example

**User Request**: "Update all the evidence"

**Claude's Execution**:

```bash
# Step 1: Check status
grctool status --filter state=no_evidence

# Step 2: Generate context for all pending tasks
grctool evidence generate ET-0047 --window 2025-Q4  # GitHub Access
grctool evidence generate ET-0048 --window 2025-Q4  # GitHub Workflows
grctool evidence generate ET-0049 --window 2025-Q4  # GitHub Security
grctool evidence generate ET-0023 --window 2025-Q4  # Infrastructure
grctool evidence generate ET-0024 --window 2025-Q4  # Config Mgmt

# Step 3: Execute GitHub tools (run once)
grctool tool github-permissions --repository org/repo > /tmp/gh-perms.csv
grctool tool github-workflow-analyzer --repository org/repo > /tmp/gh-workflows.json
grctool tool github-security-features --repository org/repo > /tmp/gh-security.json

# Step 4: Execute Terraform tools (run once)
grctool tool terraform-security-indexer > /tmp/tf-index.csv
grctool tool terraform-security-analyzer --security-domain all > /tmp/tf-analysis.json

# Step 5: Save evidence to all applicable tasks
# GitHub tasks
grctool tool evidence-writer --task-ref ET-0047 --title "Permissions" --file /tmp/gh-perms.csv
grctool tool evidence-writer --task-ref ET-0047 --title "Security" --file /tmp/gh-security.json

grctool tool evidence-writer --task-ref ET-0048 --title "Workflows" --file /tmp/gh-workflows.json

grctool tool evidence-writer --task-ref ET-0049 --title "Security" --file /tmp/gh-security.json

# Terraform tasks
grctool tool evidence-writer --task-ref ET-0023 --title "Security" --file /tmp/tf-index.csv
grctool tool evidence-writer --task-ref ET-0023 --title "Analysis" --file /tmp/tf-analysis.json

grctool tool evidence-writer --task-ref ET-0024 --title "Security" --file /tmp/tf-index.csv

# Step 6: Verify completion
grctool status
```

**Claude's Report**:
> "✅ Completed bulk evidence generation:
> 
> **GitHub Evidence**:
> - ET-0047 (Access Controls): 2 files generated
> - ET-0048 (CI/CD Workflows): 1 file generated
> - ET-0049 (Security Features): 1 file generated
> 
> **Infrastructure Evidence**:
> - ET-0023 (Security): 2 files generated
> - ET-0024 (Config Mgmt): 1 file generated
> 
> **Summary**: 5 tasks completed, 7 evidence files generated
> 
> **Next Steps**:
> 1. Review evidence with: `grctool status task ET-XXXX`
> 2. Validate all: `grctool evidence validate --all`
> 3. Submit when ready: `grctool evidence submit --all`"

---

## Best Practices

### DO:
✅ Check status before starting  
✅ Generate context for all tasks first  
✅ Group tasks by tool requirements  
✅ Run each tool once, reuse output  
✅ Track progress and report completion  
✅ Handle errors gracefully  
✅ Provide detailed completion summary  

### DON'T:
❌ Run the same tool multiple times unnecessarily  
❌ Skip status checks  
❌ Fail silently on errors  
❌ Process tasks one-by-one (be efficient!)  
❌ Forget to report progress to user  

---

## Performance Optimization

### Tool Reuse Strategy

**Scenario**: 10 tasks all need `github-permissions`

**❌ Inefficient** (10 tool executions):
```bash
grctool tool github-permissions ... > ET-0001.csv
grctool tool github-permissions ... > ET-0002.csv
# ... 10 times!
```

**✅ Efficient** (1 tool execution):
```bash
# Run once
grctool tool github-permissions ... > /tmp/permissions.csv

# Save to all 10 tasks
for task in ET-0001 ET-0002 ... ET-0010; do
  grctool tool evidence-writer --task-ref $task \
    --title "Permissions" --file /tmp/permissions.csv
done
```

### Parallel vs Sequential

**Tools can run in parallel** if they're independent:
```bash
# Run these concurrently
grctool tool github-permissions ... > /tmp/gh-perms.csv &
grctool tool terraform-security-indexer > /tmp/tf-index.csv &
wait  # Wait for both to finish
```

---

**Next Steps**: Refer back to `tool-capabilities.md` for detailed tool usage, and `evidence-workflow.md` for single-task patterns.
