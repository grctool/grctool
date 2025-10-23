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

package models

import "time"

// EvidenceSubmission represents a single task's evidence submission
type EvidenceSubmission struct {
	// Identity
	TaskID  int    `yaml:"task_id" json:"task_id"`
	TaskRef string `yaml:"task_ref" json:"task_ref"` // ET-0001
	Window  string `yaml:"window" json:"window"`     // 2025-Q4

	// Submission tracking
	Status       string  `yaml:"status" json:"status"`                                   // draft, validated, submitted, accepted, rejected
	SubmissionID string  `yaml:"submission_id,omitempty" json:"submission_id,omitempty"` // Tugboat submission ID
	BatchID      string  `yaml:"batch_id,omitempty" json:"batch_id,omitempty"`           // Associated batch
	BatchName    *string `yaml:"batch_name,omitempty" json:"batch_name,omitempty"`       // Optional batch name reference

	// Timestamps
	CreatedAt   time.Time  `yaml:"created_at" json:"created_at"`
	ValidatedAt *time.Time `yaml:"validated_at,omitempty" json:"validated_at,omitempty"`
	SubmittedAt *time.Time `yaml:"submitted_at,omitempty" json:"submitted_at,omitempty"`
	AcceptedAt  *time.Time `yaml:"accepted_at,omitempty" json:"accepted_at,omitempty"`

	// Content
	EvidenceFiles  []EvidenceFileRef `yaml:"evidence_files" json:"evidence_files"`
	TotalFileCount int               `yaml:"total_file_count" json:"total_file_count"`
	TotalSizeBytes int64             `yaml:"total_size_bytes" json:"total_size_bytes"`

	// Metadata
	SubmittedBy string   `yaml:"submitted_by" json:"submitted_by"` // User email
	Notes       string   `yaml:"notes,omitempty" json:"notes,omitempty"`
	Tags        []string `yaml:"tags,omitempty" json:"tags,omitempty"`

	// Validation
	ValidationStatus   string            `yaml:"validation_status" json:"validation_status"` // pending, passed, failed
	ValidationErrors   []ValidationError `yaml:"validation_errors,omitempty" json:"validation_errors,omitempty"`
	ValidationWarnings []ValidationError `yaml:"validation_warnings,omitempty" json:"validation_warnings,omitempty"`
	CompletenessScore  float64           `yaml:"completeness_score" json:"completeness_score"` // 0.0-1.0

	// Tugboat response
	TugboatResponse *TugboatSubmissionResponse `yaml:"tugboat_response,omitempty" json:"tugboat_response,omitempty"`
}

// EvidenceFileRef references a single evidence file
type EvidenceFileRef struct {
	Filename          string   `yaml:"filename" json:"filename"`           // 01_terraform_iam_roles.md
	RelativePath      string   `yaml:"relative_path" json:"relative_path"` // evidence/ET-0001/2025-Q4/01_terraform_iam_roles.md
	Title             string   `yaml:"title" json:"title"`
	Source            string   `yaml:"source" json:"source"` // terraform-scanner, github-permissions
	SizeBytes         int64    `yaml:"size_bytes" json:"size_bytes"`
	ChecksumSHA256    string   `yaml:"checksum_sha256" json:"checksum_sha256"`
	ControlsSatisfied []string `yaml:"controls_satisfied,omitempty" json:"controls_satisfied,omitempty"` // CC6.8, AC-01
}

// ValidationError represents a validation failure or warning
type ValidationError struct {
	Code       string `yaml:"code" json:"code"`         // INCOMPLETE_CONTROLS, MISSING_FILE, FORMAT_ERROR
	Severity   string `yaml:"severity" json:"severity"` // error, warning, info
	Message    string `yaml:"message" json:"message"`
	Field      string `yaml:"field,omitempty" json:"field,omitempty"`
	Suggestion string `yaml:"suggestion,omitempty" json:"suggestion,omitempty"`
}

// TugboatSubmissionResponse captures the API response from Tugboat
type TugboatSubmissionResponse struct {
	SubmissionID string                 `yaml:"submission_id" json:"submission_id"`
	Status       string                 `yaml:"status" json:"status"`
	Message      string                 `yaml:"message,omitempty" json:"message,omitempty"`
	ReceivedAt   time.Time              `yaml:"received_at" json:"received_at"`
	Metadata     map[string]interface{} `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

// SubmissionBatch represents a group of related submissions
type SubmissionBatch struct {
	// Identity
	BatchID   string `yaml:"batch_id" json:"batch_id"`                         // batch-2025-10-22-143052
	BatchName string `yaml:"batch_name,omitempty" json:"batch_name,omitempty"` // "Q4 2025 Submissions"

	// Status
	Status string `yaml:"status" json:"status"` // draft, validating, submitting, completed, failed

	// Tasks
	TaskRefs       []string `yaml:"task_refs" json:"task_refs"` // [ET-0001, ET-0047]
	TotalTasks     int      `yaml:"total_tasks" json:"total_tasks"`
	SubmittedTasks int      `yaml:"submitted_tasks" json:"submitted_tasks"`
	FailedTasks    int      `yaml:"failed_tasks" json:"failed_tasks"`

	// Timestamps
	CreatedAt   time.Time  `yaml:"created_at" json:"created_at"`
	StartedAt   *time.Time `yaml:"started_at,omitempty" json:"started_at,omitempty"`
	CompletedAt *time.Time `yaml:"completed_at,omitempty" json:"completed_at,omitempty"`

	// Metadata
	CreatedBy string   `yaml:"created_by" json:"created_by"`
	Notes     string   `yaml:"notes,omitempty" json:"notes,omitempty"`
	Tags      []string `yaml:"tags,omitempty" json:"tags,omitempty"`

	// Validation
	ValidationMode  string `yaml:"validation_mode" json:"validation_mode"` // strict, lenient, skip
	ContinueOnError bool   `yaml:"continue_on_error" json:"continue_on_error"`

	// Results
	Submissions []BatchSubmissionResult `yaml:"submissions" json:"submissions"`
}

// BatchSubmissionResult tracks individual task results in a batch
type BatchSubmissionResult struct {
	TaskRef      string     `yaml:"task_ref" json:"task_ref"`
	Status       string     `yaml:"status" json:"status"` // pending, submitted, failed, skipped
	SubmissionID string     `yaml:"submission_id,omitempty" json:"submission_id,omitempty"`
	Error        string     `yaml:"error,omitempty" json:"error,omitempty"`
	SubmittedAt  *time.Time `yaml:"submitted_at,omitempty" json:"submitted_at,omitempty"`
}

// SubmissionHistory tracks all submissions for a task window
type SubmissionHistory struct {
	TaskRef string                   `yaml:"task_ref" json:"task_ref"`
	Window  string                   `yaml:"window" json:"window"`
	Entries []SubmissionHistoryEntry `yaml:"entries" json:"entries"`
}

// SubmissionHistoryEntry is a single submission attempt
type SubmissionHistoryEntry struct {
	SubmissionID string    `yaml:"submission_id" json:"submission_id"`
	SubmittedAt  time.Time `yaml:"submitted_at" json:"submitted_at"`
	SubmittedBy  string    `yaml:"submitted_by" json:"submitted_by"`
	Status       string    `yaml:"status" json:"status"` // submitted, accepted, rejected
	FileCount    int       `yaml:"file_count" json:"file_count"`
	Notes        string    `yaml:"notes,omitempty" json:"notes,omitempty"`
	BatchID      string    `yaml:"batch_id,omitempty" json:"batch_id,omitempty"`
}

// ValidationResult represents the complete validation result
type ValidationResult struct {
	TaskRef             string            `json:"task_ref"`
	Window              string            `json:"window"`
	Status              string            `json:"status"` // passed, failed, warning
	ValidationMode      string            `json:"validation_mode"`
	CompletenessScore   float64           `json:"completeness_score"`
	TotalChecks         int               `json:"total_checks"`
	PassedChecks        int               `json:"passed_checks"`
	FailedChecks        int               `json:"failed_checks"`
	Warnings            int               `json:"warnings"`
	Errors              []ValidationError `json:"errors"`
	WarningsList        []ValidationError `json:"warnings_list"`
	Checks              []ValidationCheck `json:"checks"`
	EvidenceFiles       []EvidenceFileRef `json:"evidence_files"`
	ReadyForSubmission  bool              `json:"ready_for_submission"`
	ValidationTimestamp time.Time         `json:"validation_timestamp"`
}

// ValidationCheck represents a single validation check result
type ValidationCheck struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Status   string `json:"status"`   // passed, failed, warning, skipped
	Severity string `json:"severity"` // error, warning, info
	Message  string `json:"message"`
}

// SubmitEvidenceRequest is the submission payload for Tugboat
type SubmitEvidenceRequest struct {
	TaskID           int                    `json:"task_id"`
	Content          string                 `json:"content"`           // Main evidence content
	ContentType      string                 `json:"content_type"`      // markdown, json, csv
	CollectionWindow string                 `json:"collection_window"` // 2025-Q4
	CollectionDate   string                 `json:"collection_date"`   // ISO 8601
	Sources          []EvidenceSourceRef    `json:"sources"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	Notes            string                 `json:"notes,omitempty"`
	ControlsCovered  []string               `json:"controls_covered"` // Control IDs
	Attachments      []AttachmentRef        `json:"attachments,omitempty"`
}

// EvidenceSourceRef references a source used to generate evidence
type EvidenceSourceRef struct {
	Type      string `json:"type"`      // terraform, github, manual
	Tool      string `json:"tool"`      // tool name
	Timestamp string `json:"timestamp"` // ISO 8601
}

// AttachmentRef references an uploaded file
type AttachmentRef struct {
	FileID   string `json:"file_id"` // Returned from upload endpoint
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
}

// SubmitEvidenceResponse is Tugboat's response
type SubmitEvidenceResponse struct {
	SubmissionID string                 `json:"submission_id"`
	Status       string                 `json:"status"` // accepted, pending_review, rejected
	Message      string                 `json:"message,omitempty"`
	TaskID       int                    `json:"task_id"`
	ReceivedAt   time.Time              `json:"received_at"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// SubmissionStatusResponse provides submission status from Tugboat
type SubmissionStatusResponse struct {
	SubmissionID string     `json:"submission_id"`
	TaskID       int        `json:"task_id"`
	Status       string     `json:"status"` // pending, accepted, rejected
	SubmittedAt  time.Time  `json:"submitted_at"`
	ReviewedAt   *time.Time `json:"reviewed_at,omitempty"`
	ReviewedBy   string     `json:"reviewed_by,omitempty"`
	ReviewNotes  string     `json:"review_notes,omitempty"`
}

// FileUploadResponse is the response from file upload
type FileUploadResponse struct {
	FileID   string    `json:"file_id"`
	Filename string    `json:"filename"`
	Size     int64     `json:"size"`
	Uploaded time.Time `json:"uploaded"`
}

// SubmissionListResponse lists submissions for a task
type SubmissionListResponse struct {
	TaskID      int                        `json:"task_id"`
	Total       int                        `json:"total"`
	Submissions []SubmissionStatusResponse `json:"submissions"`
}
