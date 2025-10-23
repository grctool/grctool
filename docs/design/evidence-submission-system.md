# Evidence Submission System Design

## Executive Summary

This document outlines the design for GRCTool's evidence submission system, enabling automated upload of generated compliance evidence back to Tugboat Logic. The design extends the existing evidence generation workflow with submission tracking, validation, and batch operations.

## 1. Current State Analysis

### Existing Evidence Flow
```
Evidence Task (Tugboat)
    ↓ sync
Evidence Task (Local)
    ↓ generate
Evidence Records (Local)
    ↓ ??? (missing)
Evidence Submission (Tugboat)
```

### Current Strengths
- ✅ Task-based organization (`evidence/ET-0001/`)
- ✅ Time-windowed collection (`2025-Q4/`)
- ✅ Collection plans track completeness
- ✅ Evidence records maintain TaskID linkage
- ✅ Source tracking (terraform, github, etc.)
- ✅ Metadata preservation

### Current Gaps
- ❌ No submission state tracking
- ❌ No submission metadata model
- ❌ No batch submission concept
- ❌ No validation workflow
- ❌ No audit trail
- ❌ No POST/PUT methods in API client

## 2. Evidence Organization Structure

### Directory Structure (Enhanced)

```
data_dir/
├── evidence/                           # Generated evidence
│   ├── ET-0001_Access_Control/
│   │   ├── 2025-Q4/                   # Time window
│   │   │   ├── collection_plan.md
│   │   │   ├── collection_plan_metadata.yaml
│   │   │   ├── 01_terraform_iam_roles.md
│   │   │   ├── 02_github_access_controls.md
│   │   │   ├── 03_access_policy.md
│   │   │   └── .submission/            # NEW: Submission tracking
│   │   │       ├── submission.yaml     # Submission metadata
│   │   │       ├── validation.yaml     # Validation results
│   │   │       └── history.yaml        # Submission history
│   │   └── 2025-Q3/                   # Previous window
│   │       └── .submission/
│   │           └── submission.yaml     # Historical submission
│   └── ET-0047_Repository_Controls/
│       └── 2025-Q4/
│           └── .submission/
└── submissions/                        # NEW: Submission batches
    ├── batch-2025-10-22-143052/       # Timestamp-based batch ID
    │   ├── manifest.yaml               # Batch manifest
    │   ├── ET-0001.yaml                # Task submission metadata
    │   ├── ET-0047.yaml
    │   └── submission_log.yaml         # Batch submission log
    └── batch-2025-10-15-092341/
        └── manifest.yaml
```

### Key Design Decisions

**Decision 1: Per-Window Submission Tracking**
- Submission metadata lives inside each time window (`.submission/`)
- Rationale: Evidence is time-bound; submissions are tied to specific collection periods
- Benefits: Clear audit trail, supports re-submission of updated evidence

**Decision 2: Batch Submission Model**
- New top-level `submissions/` directory for batch operations
- Rationale: Users often submit multiple related tasks together (e.g., all Q4 evidence)
- Benefits: Atomic operations, rollback support, bulk status tracking

**Decision 3: Hidden `.submission/` Directory**
- Use dotfile pattern for submission metadata
- Rationale: Separates submission tracking from evidence content
- Benefits: Clean directory listing, doesn't interfere with evidence files

## 3. Data Models

### 3.1 Submission Metadata

```go
// File: internal/models/submission.go

// EvidenceSubmission represents a single task's evidence submission
type EvidenceSubmission struct {
    // Identity
    TaskID         int       `yaml:"task_id" json:"task_id"`
    TaskRef        string    `yaml:"task_ref" json:"task_ref"`           // ET-0001
    Window         string    `yaml:"window" json:"window"`               // 2025-Q4

    // Submission tracking
    Status         string    `yaml:"status" json:"status"`               // draft, validated, submitted, accepted, rejected
    SubmissionID   string    `yaml:"submission_id,omitempty" json:"submission_id,omitempty"` // Tugboat submission ID
    BatchID        string    `yaml:"batch_id,omitempty" json:"batch_id,omitempty"`           // Associated batch

    // Timestamps
    CreatedAt      time.Time `yaml:"created_at" json:"created_at"`
    ValidatedAt    *time.Time `yaml:"validated_at,omitempty" json:"validated_at,omitempty"`
    SubmittedAt    *time.Time `yaml:"submitted_at,omitempty" json:"submitted_at,omitempty"`
    AcceptedAt     *time.Time `yaml:"accepted_at,omitempty" json:"accepted_at,omitempty"`

    // Content
    EvidenceFiles  []EvidenceFileRef `yaml:"evidence_files" json:"evidence_files"`
    TotalFileCount int               `yaml:"total_file_count" json:"total_file_count"`
    TotalSizeBytes int64             `yaml:"total_size_bytes" json:"total_size_bytes"`

    // Metadata
    SubmittedBy    string             `yaml:"submitted_by" json:"submitted_by"`   // User email
    Notes          string             `yaml:"notes,omitempty" json:"notes,omitempty"`
    Tags           []string           `yaml:"tags,omitempty" json:"tags,omitempty"`

    // Validation
    ValidationStatus string           `yaml:"validation_status" json:"validation_status"` // pending, passed, failed
    ValidationErrors []ValidationError `yaml:"validation_errors,omitempty" json:"validation_errors,omitempty"`
    CompletenessScore float64         `yaml:"completeness_score" json:"completeness_score"` // 0.0-1.0

    // Tugboat response
    TugboatResponse *TugboatSubmissionResponse `yaml:"tugboat_response,omitempty" json:"tugboat_response,omitempty"`
}

// EvidenceFileRef references a single evidence file
type EvidenceFileRef struct {
    Filename       string   `yaml:"filename" json:"filename"`           // 01_terraform_iam_roles.md
    RelativePath   string   `yaml:"relative_path" json:"relative_path"` // evidence/ET-0001/2025-Q4/01_terraform_iam_roles.md
    Title          string   `yaml:"title" json:"title"`
    Source         string   `yaml:"source" json:"source"`               // terraform-scanner, github-permissions
    SizeBytes      int64    `yaml:"size_bytes" json:"size_bytes"`
    ChecksumSHA256 string   `yaml:"checksum_sha256" json:"checksum_sha256"`
    ControlsSatisfied []string `yaml:"controls_satisfied" json:"controls_satisfied"` // CC6.8, AC-01
}

// ValidationError represents a validation failure
type ValidationError struct {
    Code       string `yaml:"code" json:"code"`               // INCOMPLETE_CONTROLS, MISSING_FILE, FORMAT_ERROR
    Severity   string `yaml:"severity" json:"severity"`       // error, warning, info
    Message    string `yaml:"message" json:"message"`
    Field      string `yaml:"field,omitempty" json:"field,omitempty"`
    Suggestion string `yaml:"suggestion,omitempty" json:"suggestion,omitempty"`
}

// TugboatSubmissionResponse captures the API response
type TugboatSubmissionResponse struct {
    SubmissionID string                 `yaml:"submission_id" json:"submission_id"`
    Status       string                 `yaml:"status" json:"status"`
    Message      string                 `yaml:"message,omitempty" json:"message,omitempty"`
    ReceivedAt   time.Time              `yaml:"received_at" json:"received_at"`
    Metadata     map[string]interface{} `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}
```

### 3.2 Batch Submission Model

```go
// SubmissionBatch represents a group of related submissions
type SubmissionBatch struct {
    // Identity
    BatchID       string    `yaml:"batch_id" json:"batch_id"`           // batch-2025-10-22-143052
    BatchName     string    `yaml:"batch_name,omitempty" json:"batch_name,omitempty"` // "Q4 2025 Submissions"

    // Status
    Status        string    `yaml:"status" json:"status"`               // draft, validating, submitting, completed, failed

    // Tasks
    TaskRefs      []string  `yaml:"task_refs" json:"task_refs"`         // [ET-0001, ET-0047]
    TotalTasks    int       `yaml:"total_tasks" json:"total_tasks"`
    SubmittedTasks int      `yaml:"submitted_tasks" json:"submitted_tasks"`
    FailedTasks   int       `yaml:"failed_tasks" json:"failed_tasks"`

    // Timestamps
    CreatedAt     time.Time  `yaml:"created_at" json:"created_at"`
    StartedAt     *time.Time `yaml:"started_at,omitempty" json:"started_at,omitempty"`
    CompletedAt   *time.Time `yaml:"completed_at,omitempty" json:"completed_at,omitempty"`

    // Metadata
    CreatedBy     string              `yaml:"created_by" json:"created_by"`
    Notes         string              `yaml:"notes,omitempty" json:"notes,omitempty"`
    Tags          []string            `yaml:"tags,omitempty" json:"tags,omitempty"`

    // Validation
    ValidationMode string             `yaml:"validation_mode" json:"validation_mode"` // strict, lenient, skip
    ContinueOnError bool              `yaml:"continue_on_error" json:"continue_on_error"`

    // Results
    Submissions   []BatchSubmissionResult `yaml:"submissions" json:"submissions"`
}

// BatchSubmissionResult tracks individual task results in a batch
type BatchSubmissionResult struct {
    TaskRef        string     `yaml:"task_ref" json:"task_ref"`
    Status         string     `yaml:"status" json:"status"`           // pending, submitted, failed, skipped
    SubmissionID   string     `yaml:"submission_id,omitempty" json:"submission_id,omitempty"`
    Error          string     `yaml:"error,omitempty" json:"error,omitempty"`
    SubmittedAt    *time.Time `yaml:"submitted_at,omitempty" json:"submitted_at,omitempty"`
}
```

### 3.3 Submission History

```go
// SubmissionHistory tracks all submissions for a task window
type SubmissionHistory struct {
    TaskRef   string              `yaml:"task_ref" json:"task_ref"`
    Window    string              `yaml:"window" json:"window"`
    Entries   []SubmissionHistoryEntry `yaml:"entries" json:"entries"`
}

// SubmissionHistoryEntry is a single submission attempt
type SubmissionHistoryEntry struct {
    SubmissionID string    `yaml:"submission_id" json:"submission_id"`
    SubmittedAt  time.Time `yaml:"submitted_at" json:"submitted_at"`
    SubmittedBy  string    `yaml:"submitted_by" json:"submitted_by"`
    Status       string    `yaml:"status" json:"status"`       // submitted, accepted, rejected
    FileCount    int       `yaml:"file_count" json:"file_count"`
    Notes        string    `yaml:"notes,omitempty" json:"notes,omitempty"`
    BatchID      string    `yaml:"batch_id,omitempty" json:"batch_id,omitempty"`
}
```

## 4. Submission Lifecycle

### Status State Machine

```
                    ┌──────────────────────────┐
                    │     Evidence Generated    │
                    └────────────┬──────────────┘
                                 │
                                 ▼
                    ┌──────────────────────────┐
                    │     draft                 │  Initial state after generation
                    └────────────┬──────────────┘
                                 │
                        validate command
                                 │
                                 ▼
                    ┌──────────────────────────┐
                    │     validating            │  Running validation checks
                    └────────────┬──────────────┘
                                 │
                    ┌────────────┼────────────┐
                    │                          │
                    ▼                          ▼
        ┌──────────────────────┐  ┌──────────────────────┐
        │  validation_failed    │  │  validated           │
        └──────────────────────┘  └──────────┬───────────┘
                                              │
                                     submit command
                                              │
                                              ▼
                                  ┌──────────────────────┐
                                  │  submitting          │  Uploading to Tugboat
                                  └──────────┬───────────┘
                                             │
                        ┌────────────────────┼────────────────────┐
                        │                                          │
                        ▼                                          ▼
            ┌──────────────────────┐                  ┌──────────────────────┐
            │  submission_failed    │                  │  submitted           │
            └──────────────────────┘                  └──────────┬───────────┘
                                                                  │
                                                         (Tugboat processes)
                                                                  │
                                            ┌─────────────────────┼─────────────────────┐
                                            │                                            │
                                            ▼                                            ▼
                                ┌──────────────────────┐                    ┌──────────────────────┐
                                │  accepted            │                    │  rejected            │
                                └──────────────────────┘                    └──────────────────────┘
```

### Status Definitions

| Status | Description | User Actions Available |
|--------|-------------|------------------------|
| `draft` | Evidence generated but not validated | validate, edit, delete |
| `validating` | Validation checks in progress | wait |
| `validation_failed` | Validation checks failed | fix issues, re-validate |
| `validated` | Passed validation checks | submit, edit (triggers re-validation) |
| `submitting` | Upload to Tugboat in progress | wait, cancel |
| `submission_failed` | Upload failed | retry, fix issues |
| `submitted` | Successfully uploaded, awaiting Tugboat processing | view status |
| `accepted` | Tugboat accepted the evidence | view, archive |
| `rejected` | Tugboat rejected the evidence | view reasons, resubmit |

## 5. Validation Framework

### Validation Checks

```go
// ValidationRule defines a validation check
type ValidationRule struct {
    Code        string                          // CONTROLS_COVERAGE
    Name        string                          // "Controls Coverage Check"
    Description string                          // "Ensures all required controls are addressed"
    Severity    string                          // error, warning, info
    Category    string                          // completeness, format, content, metadata
    Validator   func(*EvidenceSubmission) []ValidationError
}
```

### Standard Validation Rules

**Completeness Checks:**
1. **CONTROLS_COVERAGE** - All task controls have evidence
2. **COLLECTION_PLAN_COMPLETE** - Collection plan shows 100% completeness
3. **REQUIRED_FILES_PRESENT** - All required evidence files exist
4. **MINIMUM_FILE_COUNT** - At least one evidence file exists

**Format Checks:**
5. **VALID_FILE_EXTENSIONS** - Files use allowed extensions (.md, .csv, .pdf, .json)
6. **FILE_SIZE_LIMITS** - Files within size limits (e.g., 50MB max)
7. **MARKDOWN_SYNTAX** - Valid markdown syntax
8. **CSV_STRUCTURE** - Valid CSV structure if applicable

**Content Checks:**
9. **NON_EMPTY_CONTENT** - Files contain actual content
10. **TIMESTAMP_VALIDITY** - Evidence timestamps within window period
11. **SOURCE_ATTRIBUTION** - All evidence cites sources/tools
12. **NO_PLACEHOLDER_TEXT** - No TODO or placeholder markers

**Metadata Checks:**
13. **VALID_TASK_REF** - Task reference matches pattern ET-XXXX
14. **WINDOW_FORMAT** - Window follows YYYY-QX or YYYY-MM-DD pattern
15. **CHECKSUM_PRESENT** - All files have SHA256 checksums

### Validation Modes

| Mode | Description | Use Case |
|------|-------------|----------|
| `strict` | All checks must pass; warnings block submission | Production submissions |
| `lenient` | Only errors block; warnings allowed | Development/testing |
| `advisory` | Nothing blocks; all issues reported as info | Pre-validation assessment |
| `skip` | No validation performed | Emergency bypass (logged) |

## 6. Tugboat API Integration

### New Client Methods

```go
// File: internal/tugboat/client.go (additions)

// post executes a POST request
func (c *Client) post(ctx context.Context, endpoint string, body interface{}, result interface{}) error

// put executes a PUT request
func (c *Client) put(ctx context.Context, endpoint string, body interface{}, result interface{}) error

// patch executes a PATCH request
func (c *Client) patch(ctx context.Context, endpoint string, body interface{}, result interface{}) error

// uploadFile handles multipart file uploads
func (c *Client) uploadFile(ctx context.Context, endpoint string, file io.Reader, filename string, metadata map[string]string) error
```

### Evidence Submission Endpoints (Projected)

```go
// File: internal/tugboat/evidence_submission.go (new)

// SubmitEvidence submits evidence content for a task
func (c *Client) SubmitEvidence(ctx context.Context, orgID string, taskID int, req *SubmitEvidenceRequest) (*SubmitEvidenceResponse, error)

// UploadEvidenceFile uploads an evidence file
func (c *Client) UploadEvidenceFile(ctx context.Context, orgID string, taskID int, file io.Reader, filename string) (*FileUploadResponse, error)

// UpdateTaskCompletionStatus marks a task as completed
func (c *Client) UpdateTaskCompletionStatus(ctx context.Context, orgID string, taskID int, completed bool, lastCollected time.Time) error

// GetSubmissionStatus retrieves submission status
func (c *Client) GetSubmissionStatus(ctx context.Context, orgID string, taskID int, submissionID string) (*SubmissionStatusResponse, error)

// ListTaskSubmissions lists all submissions for a task
func (c *Client) ListTaskSubmissions(ctx context.Context, orgID string, taskID int) (*SubmissionListResponse, error)
```

### API Request/Response Models

```go
// SubmitEvidenceRequest is the submission payload
type SubmitEvidenceRequest struct {
    TaskID           int                    `json:"task_id"`
    Content          string                 `json:"content"`                    // Main evidence content
    ContentType      string                 `json:"content_type"`               // markdown, json, csv
    CollectionWindow string                 `json:"collection_window"`          // 2025-Q4
    CollectionDate   string                 `json:"collection_date"`            // ISO 8601
    Sources          []EvidenceSourceRef    `json:"sources"`
    Metadata         map[string]interface{} `json:"metadata,omitempty"`
    Notes            string                 `json:"notes,omitempty"`
    ControlsCovered  []string               `json:"controls_covered"`           // Control IDs
    Attachments      []AttachmentRef        `json:"attachments,omitempty"`
}

// AttachmentRef references an uploaded file
type AttachmentRef struct {
    FileID   string `json:"file_id"`    // Returned from upload endpoint
    Filename string `json:"filename"`
    Size     int64  `json:"size"`
}

// SubmitEvidenceResponse is Tugboat's response
type SubmitEvidenceResponse struct {
    SubmissionID string                 `json:"submission_id"`
    Status       string                 `json:"status"`           // accepted, pending_review, rejected
    Message      string                 `json:"message,omitempty"`
    TaskID       int                    `json:"task_id"`
    ReceivedAt   time.Time              `json:"received_at"`
    Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// SubmissionStatusResponse provides submission status
type SubmissionStatusResponse struct {
    SubmissionID string    `json:"submission_id"`
    TaskID       int       `json:"task_id"`
    Status       string    `json:"status"`           // pending, accepted, rejected
    SubmittedAt  time.Time `json:"submitted_at"`
    ReviewedAt   *time.Time `json:"reviewed_at,omitempty"`
    ReviewedBy   string    `json:"reviewed_by,omitempty"`
    ReviewNotes  string    `json:"review_notes,omitempty"`
}
```

### Endpoint Patterns (Projected)

Based on existing patterns, projected submission endpoints:

```
POST   /api/org_evidence/{task_id}/submissions/          Submit evidence
GET    /api/org_evidence/{task_id}/submissions/          List submissions
GET    /api/org_evidence/{task_id}/submissions/{sub_id}/ Get submission details
PATCH  /api/org_evidence/{task_id}/                      Update task status
POST   /api/org_evidence/{task_id}/files/                Upload file
DELETE /api/org_evidence/{task_id}/submissions/{sub_id}/ Delete submission
```

## 7. Command-Line Interface

### New Commands

```bash
# Validation commands
grctool evidence validate ET-0001              # Validate single task
grctool evidence validate ET-0001 --strict     # Use strict validation mode
grctool evidence validate --all                # Validate all ready evidence
grctool evidence validate --window 2025-Q4     # Validate all Q4 evidence

# Submission commands
grctool evidence submit ET-0001                # Submit single task
grctool evidence submit ET-0001 --skip-validation  # Skip validation (logs warning)
grctool evidence submit --batch "Q4 Tasks" ET-0001 ET-0047 ET-0103  # Batch submit
grctool evidence submit --batch-file batch.yaml    # Submit from batch file
grctool evidence submit --all --window 2025-Q4     # Submit all Q4 evidence

# Status commands
grctool evidence status ET-0001                # Check submission status
grctool evidence status --all                  # Show all submission statuses
grctool evidence status --batch batch-2025-10-22-143052  # Batch status
grctool evidence submissions ET-0001           # List all submissions for task

# History commands
grctool evidence history ET-0001               # View submission history
grctool evidence history --window 2025-Q4      # History for window

# Batch management
grctool batch create "Q4 2025" ET-0001 ET-0047  # Create submission batch
grctool batch list                              # List all batches
grctool batch show batch-2025-10-22-143052      # Show batch details
grctool batch submit batch-2025-10-22-143052    # Submit batch
grctool batch status batch-2025-10-22-143052    # Check batch status
```

### Example Workflows

**Single Task Submission:**
```bash
# 1. Generate evidence
grctool evidence generate ET-0001

# 2. Validate
grctool evidence validate ET-0001
# Output: ✓ All checks passed (15/15)
#         Completeness: 100%
#         Ready for submission

# 3. Submit
grctool evidence submit ET-0001
# Output: Submitting evidence for ET-0001...
#         ✓ Uploaded 3 files (2.4 MB)
#         ✓ Submission accepted (ID: sub-12345)
#         Status: accepted

# 4. Check status
grctool evidence status ET-0001
# Output: Status: accepted
#         Submitted: 2025-10-22 14:30:52
#         Reviewed: 2025-10-22 14:31:15
#         Reviewer: auditor@example.com
```

**Batch Submission:**
```bash
# 1. Create batch
grctool batch create "Q4 2025 Submissions" \
    ET-0001 ET-0047 ET-0103 \
    --window 2025-Q4 \
    --notes "End of quarter compliance submission"

# Output: Created batch: batch-2025-10-22-143052
#         Tasks: 3
#         Window: 2025-Q4

# 2. Validate batch
grctool evidence validate --batch batch-2025-10-22-143052

# Output: Validating batch: batch-2025-10-22-143052
#         ✓ ET-0001: 15/15 checks passed
#         ✓ ET-0047: 15/15 checks passed
#         ✗ ET-0103: 2 errors, 1 warning
#           - ERROR: CONTROLS_COVERAGE: Missing evidence for CC6.8
#           - ERROR: REQUIRED_FILES_PRESENT: File 02_github_perms.md not found
#           - WARNING: MINIMUM_FILE_COUNT: Only 1 file (recommend 2+)

# 3. Fix issues and re-validate
grctool evidence generate ET-0103
grctool evidence validate ET-0103

# 4. Submit batch
grctool batch submit batch-2025-10-22-143052

# Output: Submitting batch: batch-2025-10-22-143052
#         [1/3] ET-0001... ✓ accepted (sub-12345)
#         [2/3] ET-0047... ✓ accepted (sub-12346)
#         [3/3] ET-0103... ✓ accepted (sub-12347)
#
#         Batch complete: 3/3 successful
```

## 8. Implementation Plan

### Phase 1: Core Models and Storage (Week 1)
- [ ] Create submission models (`internal/models/submission.go`)
- [ ] Implement submission metadata storage (`internal/storage/submission_storage.go`)
- [ ] Add `.submission/` directory management
- [ ] Create batch models and storage
- [ ] Add submission history tracking

### Phase 2: Validation Framework (Week 1-2)
- [ ] Design validation rule interface
- [ ] Implement standard validation rules
- [ ] Create validation service (`internal/services/validation/service.go`)
- [ ] Add validation CLI commands
- [ ] Build validation reporting

### Phase 3: Tugboat API Extensions (Week 2)
- [ ] Add POST/PUT/PATCH methods to client
- [ ] Create submission endpoint methods
- [ ] Implement file upload support
- [ ] Add submission status polling
- [ ] Create submission API models

### Phase 4: Submission Service (Week 2-3)
- [ ] Build submission service (`internal/services/submission/service.go`)
- [ ] Implement single task submission
- [ ] Add batch submission logic
- [ ] Create submission status tracking
- [ ] Implement retry and error handling

### Phase 5: CLI Integration (Week 3)
- [ ] Add `evidence validate` command
- [ ] Add `evidence submit` command
- [ ] Add `evidence status` command
- [ ] Add `batch` command group
- [ ] Create submission history commands

### Phase 6: Testing and Documentation (Week 4)
- [ ] Unit tests for all components
- [ ] Integration tests with VCR recordings
- [ ] End-to-end submission workflow tests
- [ ] User documentation
- [ ] API documentation updates

## 9. Edge Cases and Considerations

### Concurrency and Locking
- **Problem**: Multiple users submitting same evidence simultaneously
- **Solution**: Use file-based locking or atomic operations
- **Implementation**: Lock file at `.submission/.lock` during submission

### Partial Batch Failures
- **Problem**: 2 of 5 tasks in batch fail to submit
- **Solution**: Continue-on-error flag; mark failed tasks; allow retry
- **Implementation**: Store per-task status in batch manifest

### Network Failures During Upload
- **Problem**: Connection lost mid-submission
- **Solution**: Resumable uploads or retry with idempotency
- **Implementation**: Track upload progress; use idempotency keys

### Evidence Updates After Submission
- **Problem**: User edits evidence after submission
- **Solution**: Create new submission version; mark old as superseded
- **Implementation**: Add `superseded_by` field to submission history

### Large File Uploads
- **Problem**: Evidence includes large PDF/video files
- **Solution**: Chunked uploads, progress tracking, compression
- **Implementation**: Stream uploads with progress callbacks

### Validation Performance
- **Problem**: Validating 100+ tasks is slow
- **Solution**: Parallel validation; cache validation results
- **Implementation**: Worker pool; checksum-based caching

### Submission Conflicts
- **Problem**: Evidence submitted elsewhere (web UI) conflicts with CLI
- **Solution**: Check for existing submissions before upload; warn user
- **Implementation**: Query `/submissions/` endpoint before POST

### Rollback Support
- **Problem**: Need to undo a bad batch submission
- **Solution**: Store enough metadata to identify and request deletion
- **Implementation**: Batch manifest includes all submission IDs

## 10. Future Enhancements

### Phase 2 Features (Post-MVP)
- **Scheduled Submissions**: Cron-like submission scheduling
- **Auto-validation**: Run validation on evidence generation
- **Submission Templates**: Pre-configured batch templates
- **Approval Workflow**: Multi-step approval before submission
- **Evidence Versioning**: Full version control with diffs
- **Submission Analytics**: Success rates, timing metrics
- **Webhooks**: Notify external systems on submission events
- **Evidence Archival**: Compress old evidence windows
- **Submission Dry-run**: Preview submission without actual upload
- **Evidence Comparison**: Diff between windows/versions

### Integration Points
- **Claude Code**: Interactive submission assistance
- **CI/CD**: Automated submission in pipelines
- **Slack/Teams**: Submission status notifications
- **Git**: Evidence commit tracking
- **Jira**: Link submissions to tickets

## 11. Security and Privacy

### Authentication
- Reuse existing bearer token authentication
- Validate token before submission
- Support token refresh during long operations

### Data Sensitivity
- Never log evidence content
- Redact sensitive fields in logs/errors
- Support evidence encryption at rest
- Checksum validation for integrity

### Audit Trail
- Log all submission attempts
- Track who submitted what and when
- Preserve submission history indefinitely
- Support compliance audit exports

### Access Control
- Respect Tugboat permissions
- Validate user can submit for task
- Support team-based submissions
- Track submitter identity

## 12. Success Metrics

### Performance Targets
- Validation: < 5 seconds for typical task
- Single submission: < 30 seconds for 3 files
- Batch submission: < 5 minutes for 10 tasks
- Status check: < 2 seconds

### Reliability Targets
- Submission success rate: > 95%
- Validation accuracy: > 99%
- Idempotency: 100% (safe retries)
- Data integrity: 100% (checksums match)

### User Experience
- Clear error messages with actionable fixes
- Progress indicators for long operations
- Rich status information
- Helpful validation feedback

## 13. Configuration

### New Config Fields

```go
// Add to internal/config/config.go

type SubmissionConfig struct {
    // Validation
    DefaultValidationMode    string        `yaml:"default_validation_mode"`     // strict, lenient, advisory
    EnableAutoValidation     bool          `yaml:"enable_auto_validation"`      // Validate on generate
    ValidationCacheEnabled   bool          `yaml:"validation_cache_enabled"`
    ValidationCacheTTL       time.Duration `yaml:"validation_cache_ttl"`

    // Submission
    DefaultBatchSize         int           `yaml:"default_batch_size"`          // Max tasks per batch
    EnableBatchMode          bool          `yaml:"enable_batch_mode"`
    ContinueOnError          bool          `yaml:"continue_on_error"`           // Batch behavior
    MaxRetries               int           `yaml:"max_retries"`
    RetryBackoff             time.Duration `yaml:"retry_backoff"`

    // Uploads
    MaxFileSize              int64         `yaml:"max_file_size"`               // Bytes
    ChunkSize                int64         `yaml:"chunk_size"`                  // For large files
    EnableCompression        bool          `yaml:"enable_compression"`
    AllowedFileExtensions    []string      `yaml:"allowed_file_extensions"`

    // Tracking
    SubmissionHistoryLimit   int           `yaml:"submission_history_limit"`    // Keep last N submissions
    EnableDetailedLogging    bool          `yaml:"enable_detailed_logging"`
    SubmissionMetadataDir    string        `yaml:"submission_metadata_dir"`     // Default: .submission
    BatchStorageDir          string        `yaml:"batch_storage_dir"`           // Default: submissions/
}
```

### Example Configuration

```yaml
# grctool.yaml
submission:
  # Validation settings
  default_validation_mode: strict
  enable_auto_validation: false
  validation_cache_enabled: true
  validation_cache_ttl: 1h

  # Submission settings
  default_batch_size: 25
  enable_batch_mode: true
  continue_on_error: true
  max_retries: 3
  retry_backoff: 5s

  # Upload settings
  max_file_size: 52428800  # 50 MB
  chunk_size: 1048576      # 1 MB chunks
  enable_compression: true
  allowed_file_extensions:
    - .md
    - .csv
    - .json
    - .pdf
    - .xlsx

  # Tracking settings
  submission_history_limit: 50
  enable_detailed_logging: true
  submission_metadata_dir: .submission
  batch_storage_dir: submissions
```

## 14. Summary

This design provides a comprehensive evidence submission system that:

✅ **Preserves existing architecture** - Builds on current task/window structure
✅ **Adds submission tracking** - Complete audit trail and status management
✅ **Enables batch operations** - Efficient multi-task submissions
✅ **Implements validation** - Pre-submission quality checks
✅ **Extends Tugboat API** - RESTful submission endpoints
✅ **Provides rich CLI** - Intuitive commands and workflows
✅ **Handles edge cases** - Retry, rollback, conflict resolution
✅ **Scales for future** - Extensible design for enhancements

The system is designed to be implemented incrementally, with each phase delivering user value while building toward the complete vision.

---

**Document Version**: 1.0
**Last Updated**: 2025-10-22
**Authors**: GRCTool Architecture Team
**Status**: Design Review
