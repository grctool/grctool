# Evidence Submission Tool Interface Design

## Overview

This document defines the tool interface for the evidence submission system, designed for Claude Code orchestration and programmatic use.

## Tool Interface Specification

### 1. Evidence Validation Tool

```bash
grctool tool evidence-validator [options]
```

**Purpose**: Validate evidence for submission readiness

**Input Parameters**:
```yaml
task_ref: ET-0001                           # Required
window: 2025-Q4                             # Optional (uses latest if not specified)
validation_mode: strict                     # strict|lenient|advisory|skip
output_format: json                         # json|yaml|text
report_path: ./validation-report.json       # Where to save report
```

**Output Format** (JSON):
```json
{
  "task_ref": "ET-0001",
  "window": "2025-Q4",
  "status": "passed",
  "validation_mode": "strict",
  "completeness_score": 1.0,
  "total_checks": 15,
  "passed_checks": 15,
  "failed_checks": 0,
  "warnings": 0,
  "errors": [],
  "warnings_list": [],
  "checks": [
    {
      "code": "CONTROLS_COVERAGE",
      "name": "Controls Coverage Check",
      "status": "passed",
      "severity": "error",
      "message": "All 5 controls have evidence"
    }
  ],
  "evidence_files": [
    {
      "filename": "01_terraform_iam_roles.md",
      "size_bytes": 12458,
      "checksum": "sha256:abc123...",
      "controls_satisfied": ["AC-01", "CC6.8"]
    }
  ],
  "ready_for_submission": true,
  "validation_timestamp": "2025-10-22T14:30:52Z"
}
```

**Exit Codes**:
- `0`: Validation passed
- `1`: Validation failed (errors found)
- `2`: Warning level issues (lenient mode can still submit)
- `3`: Configuration error
- `4`: Task not found

**Example Usage**:
```bash
# Validate single task
grctool tool evidence-validator \
  --task-ref ET-0001 \
  --window 2025-Q4 \
  --validation-mode strict \
  --output-format json

# Validate and save report
grctool tool evidence-validator \
  --task-ref ET-0001 \
  --report-path ./reports/ET-0001-validation.json
```

---

### 2. Evidence Submission Tool

```bash
grctool tool evidence-submitter [options]
```

**Purpose**: Submit evidence to Tugboat Logic

**Input Parameters**:
```yaml
task_ref: ET-0001                           # Required
window: 2025-Q4                             # Optional (uses latest if not specified)
skip_validation: false                      # Skip pre-submission validation
validation_mode: strict                     # Validation mode if not skipped
dry_run: false                              # Preview without submitting
notes: "Q4 quarterly submission"            # Submission notes
tags: ["q4", "automated"]                   # Tags for submission
submitter_email: user@example.com           # Submitter (defaults to config)
output_format: json                         # json|yaml|text
```

**Output Format** (JSON):
```json
{
  "task_ref": "ET-0001",
  "window": "2025-Q4",
  "status": "accepted",
  "submission_id": "sub-12345",
  "submission_timestamp": "2025-10-22T14:31:15Z",
  "tugboat_task_id": 1234,
  "files_submitted": 3,
  "total_size_bytes": 25678,
  "submission_url": "https://my.tugboatlogic.com/org/test/evidence/tasks/1234",
  "files": [
    {
      "filename": "01_terraform_iam_roles.md",
      "size_bytes": 12458,
      "upload_status": "success",
      "file_id": "file-abc123"
    }
  ],
  "tugboat_response": {
    "submission_id": "sub-12345",
    "status": "accepted",
    "message": "Evidence received successfully",
    "received_at": "2025-10-22T14:31:15Z"
  },
  "validation_summary": {
    "passed": true,
    "checks_run": 15,
    "errors": 0,
    "warnings": 0
  }
}
```

**Exit Codes**:
- `0`: Submission successful
- `1`: Submission failed (API error)
- `2`: Validation failed (pre-submission)
- `3`: Configuration error
- `4`: Task not found
- `5`: Authentication error

**Example Usage**:
```bash
# Submit single task
grctool tool evidence-submitter \
  --task-ref ET-0001 \
  --window 2025-Q4 \
  --notes "Q4 quarterly submission" \
  --output-format json

# Dry run (preview)
grctool tool evidence-submitter \
  --task-ref ET-0001 \
  --dry-run

# Skip validation (emergency)
grctool tool evidence-submitter \
  --task-ref ET-0001 \
  --skip-validation
```

---

### 3. Batch Submission Tool

```bash
grctool tool batch-submitter [options]
```

**Purpose**: Submit multiple evidence tasks as a batch

**Input Parameters**:
```yaml
# Option 1: Direct task list
task_refs: ["ET-0001", "ET-0047", "ET-0103"]  # List of tasks

# Option 2: File-based batch
batch_file: ./batches/q4-submission.yaml      # Batch definition file

# Option 3: Filter-based selection
window: 2025-Q4                                # Submit all tasks in window
framework: soc2                                # Filter by framework
status: validated                              # Only submit validated tasks

# Common options
batch_name: "Q4 2025 Submission"               # Human-readable name
validation_mode: strict                        # Validation mode
continue_on_error: true                        # Continue if some fail
max_parallel: 5                                # Parallel submission limit
dry_run: false                                 # Preview without submitting
notes: "End of quarter compliance submission"  # Batch notes
tags: ["q4", "automated", "soc2"]             # Batch tags
output_format: json                            # json|yaml|text
report_path: ./batch-report.json               # Save detailed report
```

**Batch File Format** (YAML):
```yaml
batch_name: Q4 2025 SOC2 Submission
batch_id: batch-2025-10-22-143052
tasks:
  - task_ref: ET-0001
    window: 2025-Q4
    notes: "Access control evidence"
    priority: high
  - task_ref: ET-0047
    window: 2025-Q4
    notes: "Repository security evidence"
  - task_ref: ET-0103
    window: 2025-Q4
    notes: "Data encryption evidence"
options:
  validation_mode: strict
  continue_on_error: true
  max_parallel: 5
metadata:
  framework: soc2
  period: "Q4 2025"
  submitter: user@example.com
  tags: ["q4", "soc2", "automated"]
```

**Output Format** (JSON):
```json
{
  "batch_id": "batch-2025-10-22-143052",
  "batch_name": "Q4 2025 Submission",
  "status": "completed",
  "started_at": "2025-10-22T14:30:00Z",
  "completed_at": "2025-10-22T14:35:42Z",
  "duration_seconds": 342,
  "summary": {
    "total_tasks": 3,
    "successful": 2,
    "failed": 1,
    "skipped": 0,
    "success_rate": 0.67
  },
  "tasks": [
    {
      "task_ref": "ET-0001",
      "status": "success",
      "submission_id": "sub-12345",
      "submitted_at": "2025-10-22T14:31:15Z",
      "duration_seconds": 45
    },
    {
      "task_ref": "ET-0047",
      "status": "success",
      "submission_id": "sub-12346",
      "submitted_at": "2025-10-22T14:32:33Z",
      "duration_seconds": 38
    },
    {
      "task_ref": "ET-0103",
      "status": "failed",
      "error": "Validation failed: Missing evidence for CC6.8",
      "error_code": "VALIDATION_ERROR"
    }
  ],
  "batch_report_path": "./submissions/batch-2025-10-22-143052/report.json"
}
```

**Exit Codes**:
- `0`: All tasks submitted successfully
- `1`: Some tasks failed (check report for details)
- `2`: All tasks failed
- `3`: Configuration error
- `4`: Authentication error

**Example Usage**:
```bash
# Submit specific tasks
grctool tool batch-submitter \
  --task-refs ET-0001,ET-0047,ET-0103 \
  --batch-name "Q4 2025 Submission" \
  --continue-on-error

# Submit from batch file
grctool tool batch-submitter \
  --batch-file ./batches/q4-submission.yaml \
  --report-path ./reports/q4-batch-report.json

# Submit all validated tasks in window
grctool tool batch-submitter \
  --window 2025-Q4 \
  --status validated \
  --batch-name "Q4 Auto-Submit"

# Dry run
grctool tool batch-submitter \
  --task-refs ET-0001,ET-0047 \
  --dry-run
```

---

### 4. Submission Status Tool

```bash
grctool tool submission-status [options]
```

**Purpose**: Check submission status and history

**Input Parameters**:
```yaml
task_ref: ET-0001                          # Required (task to check)
window: 2025-Q4                            # Optional (specific window)
submission_id: sub-12345                   # Optional (specific submission)
include_history: true                      # Include all historical submissions
output_format: json                        # json|yaml|text
```

**Output Format** (JSON):
```json
{
  "task_ref": "ET-0001",
  "window": "2025-Q4",
  "current_submission": {
    "submission_id": "sub-12345",
    "status": "accepted",
    "submitted_at": "2025-10-22T14:31:15Z",
    "submitted_by": "user@example.com",
    "reviewed_at": "2025-10-22T14:31:45Z",
    "reviewed_by": "auditor@example.com",
    "review_notes": "Evidence is complete and acceptable",
    "files_count": 3,
    "batch_id": "batch-2025-10-22-143052"
  },
  "history": [
    {
      "submission_id": "sub-12345",
      "submitted_at": "2025-10-22T14:31:15Z",
      "status": "accepted",
      "files_count": 3
    },
    {
      "submission_id": "sub-12299",
      "submitted_at": "2025-10-20T10:15:30Z",
      "status": "rejected",
      "files_count": 2,
      "notes": "Previous attempt - missing controls"
    }
  ],
  "task_details": {
    "task_id": 1234,
    "name": "Access Control Evidence",
    "completion_status": "completed",
    "last_collected": "2025-10-22T14:30:00Z",
    "next_due": "2026-01-22T00:00:00Z"
  }
}
```

**Example Usage**:
```bash
# Check current submission status
grctool tool submission-status --task-ref ET-0001

# Check specific submission
grctool tool submission-status \
  --task-ref ET-0001 \
  --submission-id sub-12345

# Include full history
grctool tool submission-status \
  --task-ref ET-0001 \
  --include-history
```

---

### 5. Submission Preparation Tool

```bash
grctool tool evidence-preparer [options]
```

**Purpose**: Prepare evidence directory structure for submission

**Input Parameters**:
```yaml
task_ref: ET-0001                          # Required
window: 2025-Q4                            # Optional (creates if not exists)
initialize_submission: true                # Create .submission/ structure
generate_checksums: true                   # Generate SHA256 checksums
validate_structure: true                   # Validate directory structure
output_format: json                        # json|yaml|text
```

**Output Format** (JSON):
```json
{
  "task_ref": "ET-0001",
  "window": "2025-Q4",
  "evidence_directory": "/path/to/evidence/ET-0001/2025-Q4",
  "actions_taken": [
    "Created .submission/ directory",
    "Initialized submission.yaml",
    "Generated checksums for 3 files",
    "Validated directory structure"
  ],
  "structure": {
    "submission_metadata": ".submission/submission.yaml",
    "validation_metadata": ".submission/validation.yaml",
    "history_file": ".submission/history.yaml",
    "evidence_files": [
      "01_terraform_iam_roles.md",
      "02_github_access_controls.md",
      "03_access_policy.md"
    ]
  },
  "ready_for_submission": true
}
```

**Example Usage**:
```bash
# Prepare evidence directory
grctool tool evidence-preparer \
  --task-ref ET-0001 \
  --window 2025-Q4 \
  --initialize-submission

# Prepare with validation
grctool tool evidence-preparer \
  --task-ref ET-0001 \
  --validate-structure
```

---

## Tool Orchestration with Claude Code

### Example 1: Generate and Submit Evidence

```python
# Claude Code orchestration script
tasks = ["ET-0001", "ET-0047", "ET-0103"]
window = "2025-Q4"

for task in tasks:
    # Step 1: Generate evidence
    run_command(f"grctool evidence generate {task}")

    # Step 2: Validate
    result = run_tool("evidence-validator", {
        "task_ref": task,
        "window": window,
        "validation_mode": "strict",
        "output_format": "json"
    })

    if result["ready_for_submission"]:
        # Step 3: Submit
        submission = run_tool("evidence-submitter", {
            "task_ref": task,
            "window": window,
            "notes": f"Automated submission for {window}",
            "output_format": "json"
        })

        print(f"✓ {task}: {submission['status']}")
    else:
        print(f"✗ {task}: Validation failed")
        for error in result["errors"]:
            print(f"  - {error['message']}")
```

### Example 2: Batch Submission with Error Handling

```python
# Create batch definition
batch = {
    "batch_name": "Q4 2025 SOC2 Submission",
    "tasks": [
        {"task_ref": "ET-0001", "window": "2025-Q4"},
        {"task_ref": "ET-0047", "window": "2025-Q4"},
        {"task_ref": "ET-0103", "window": "2025-Q4"}
    ],
    "options": {
        "validation_mode": "strict",
        "continue_on_error": True
    }
}

# Save batch file
save_yaml("batch.yaml", batch)

# Submit batch
result = run_tool("batch-submitter", {
    "batch_file": "batch.yaml",
    "output_format": "json"
})

# Handle results
if result["summary"]["failed"] > 0:
    for task in result["tasks"]:
        if task["status"] == "failed":
            print(f"Failed: {task['task_ref']}")
            print(f"  Error: {task['error']}")

            # Attempt to fix and retry
            fix_task(task["task_ref"])
            retry_submission(task["task_ref"])
```

### Example 3: Validation-First Workflow

```python
# Validate all evidence in window
tasks = get_tasks_in_window("2025-Q4")

validation_results = []
for task in tasks:
    result = run_tool("evidence-validator", {
        "task_ref": task,
        "window": "2025-Q4",
        "validation_mode": "strict",
        "output_format": "json"
    })
    validation_results.append(result)

# Filter to ready tasks
ready_tasks = [
    r["task_ref"]
    for r in validation_results
    if r["ready_for_submission"]
]

# Batch submit only validated tasks
if ready_tasks:
    run_tool("batch-submitter", {
        "task_refs": ready_tasks,
        "batch_name": "Q4 Validated Submissions",
        "skip_validation": True  # Already validated
    })

# Report on not-ready tasks
not_ready = [
    r for r in validation_results
    if not r["ready_for_submission"]
]

if not_ready:
    print("\nTasks requiring attention:")
    for task in not_ready:
        print(f"\n{task['task_ref']}:")
        for error in task["errors"]:
            print(f"  - {error['message']}")
```

---

## Tool Integration Patterns

### 1. Idempotent Operations

All tools are designed to be idempotent:
- Multiple validation runs produce same result
- Re-submitting same evidence updates (doesn't duplicate)
- Preparation tool can be run multiple times safely

### 2. Exit Code Handling

```bash
# Exit code-based orchestration
if grctool tool evidence-validator --task-ref ET-0001; then
    grctool tool evidence-submitter --task-ref ET-0001
else
    echo "Validation failed, see report"
    exit 1
fi
```

### 3. JSON Pipe Processing

```bash
# Chain tools via JSON
grctool tool evidence-validator \
    --task-ref ET-0001 \
    --output-format json \
    | jq -r '.ready_for_submission' \
    | xargs -I {} sh -c 'test {} = true && grctool tool evidence-submitter --task-ref ET-0001'
```

### 4. Parallel Execution

```bash
# Validate tasks in parallel
export -f validate_task
parallel -j 5 validate_task ::: ET-0001 ET-0047 ET-0103

# Function definition
validate_task() {
    grctool tool evidence-validator \
        --task-ref "$1" \
        --output-format json \
        > "reports/${1}-validation.json"
}
```

---

## API-Style Tool Interface

For programmatic use, tools can be invoked via Go API:

```go
import (
    "github.com/yourusername/grctool/internal/tools"
    "github.com/yourusername/grctool/internal/models"
)

// Validation
validator := tools.NewEvidenceValidator(config)
result, err := validator.Validate(&tools.ValidationRequest{
    TaskRef:        "ET-0001",
    Window:         "2025-Q4",
    ValidationMode: "strict",
})

if result.ReadyForSubmission {
    // Submission
    submitter := tools.NewEvidenceSubmitter(config, tugboatClient)
    submission, err := submitter.Submit(&tools.SubmissionRequest{
        TaskRef: "ET-0001",
        Window:  "2025-Q4",
        Notes:   "Automated submission",
    })
}
```

---

## Configuration File Support

### Tool-Specific Config

```yaml
# grctool.yaml
tools:
  evidence_validator:
    default_mode: strict
    cache_enabled: true
    cache_ttl: 1h
    parallel_checks: true

  evidence_submitter:
    max_retries: 3
    retry_backoff: 5s
    timeout: 300s
    verify_after_submit: true

  batch_submitter:
    max_parallel: 5
    continue_on_error: true
    generate_reports: true
    report_format: json
```

---

## Summary

This tool interface design provides:

✅ **Consistent CLI patterns** - All tools follow same option structure
✅ **Machine-readable output** - JSON/YAML for automation
✅ **Exit codes for scripting** - Standard exit codes for control flow
✅ **Idempotent operations** - Safe to retry/re-run
✅ **Parallel execution support** - Tools designed for concurrent use
✅ **Claude Code friendly** - Easy orchestration in automated workflows
✅ **Error handling** - Clear error messages and codes
✅ **Dry-run support** - Preview before execution

These tools enable:
1. **Manual CLI usage** - Direct command-line operations
2. **Claude Code orchestration** - AI-driven workflows
3. **CI/CD integration** - Automated pipelines
4. **Programmatic access** - Go API for custom tools

---

**Document Version**: 1.0
**Last Updated**: 2025-10-22
**Status**: Ready for Implementation
