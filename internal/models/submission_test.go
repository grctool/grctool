package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvidenceSubmission_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	batchName := "Q4 2025 Batch"
	sub := EvidenceSubmission{
		TaskID: "1",
		TaskRef:   "ET-0001",
		Window:    "2025-Q4",
		Status:    "submitted",
		BatchID:   "batch-001",
		BatchName: &batchName,
		CreatedAt: now,
		EvidenceFiles: []EvidenceFileRef{
			{
				Filename:          "01_evidence.md",
				RelativePath:      "evidence/ET-0001/2025-Q4/01_evidence.md",
				Title:             "Evidence File",
				Source:            "terraform-scanner",
				SizeBytes:         1024,
				ChecksumSHA256:    "abc123",
				ControlsSatisfied: []string{"CC6.1"},
			},
		},
		TotalFileCount:   1,
		TotalSizeBytes:   1024,
		SubmittedBy:      "user@test.com",
		Notes:            "Test submission",
		Tags:             []string{"automated"},
		ValidationStatus: "passed",
		CompletenessScore: 0.95,
		TugboatResponse: &TugboatSubmissionResponse{
			SubmissionID: "tug-001",
			Status:       "accepted",
			ReceivedAt:   now,
		},
	}

	data, err := json.Marshal(sub)
	require.NoError(t, err)

	var decoded EvidenceSubmission
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, sub.TaskRef, decoded.TaskRef)
	assert.Equal(t, sub.Status, decoded.Status)
	assert.NotNil(t, decoded.BatchName)
	assert.Equal(t, batchName, *decoded.BatchName)
	assert.Len(t, decoded.EvidenceFiles, 1)
	assert.Equal(t, "01_evidence.md", decoded.EvidenceFiles[0].Filename)
	assert.NotNil(t, decoded.TugboatResponse)
	assert.Equal(t, "tug-001", decoded.TugboatResponse.SubmissionID)
}

func TestEvidenceSubmission_JSONRoundTrip_NilOptionals(t *testing.T) {
	t.Parallel()
	sub := EvidenceSubmission{
		TaskID: "1",
		TaskRef: "ET-0001",
		Status:  "draft",
	}

	data, err := json.Marshal(sub)
	require.NoError(t, err)

	var decoded EvidenceSubmission
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Nil(t, decoded.BatchName)
	assert.Nil(t, decoded.ValidatedAt)
	assert.Nil(t, decoded.SubmittedAt)
	assert.Nil(t, decoded.AcceptedAt)
	assert.Nil(t, decoded.TugboatResponse)
}

func TestSubmissionBatch_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	startedAt := now.Add(-time.Hour)
	batch := SubmissionBatch{
		BatchID:        "batch-2025-10-22",
		BatchName:      "Q4 2025 Submissions",
		Status:         "completed",
		TaskRefs:       []string{"ET-0001", "ET-0047"},
		TotalTasks:     2,
		SubmittedTasks: 2,
		FailedTasks:    0,
		CreatedAt:      now,
		StartedAt:      &startedAt,
		CompletedAt:    &now,
		CreatedBy:      "user@test.com",
		Notes:          "Batch note",
		Tags:           []string{"q4"},
		ValidationMode: "strict",
		ContinueOnError: false,
		Submissions: []BatchSubmissionResult{
			{
				TaskRef:      "ET-0001",
				Status:       "submitted",
				SubmissionID: "sub-001",
				SubmittedAt:  &now,
			},
			{
				TaskRef: "ET-0047",
				Status:  "failed",
				Error:   "validation failed",
			},
		},
	}

	data, err := json.Marshal(batch)
	require.NoError(t, err)

	var decoded SubmissionBatch
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, batch.BatchID, decoded.BatchID)
	assert.Equal(t, batch.Status, decoded.Status)
	assert.Len(t, decoded.TaskRefs, 2)
	assert.Len(t, decoded.Submissions, 2)
	assert.Equal(t, "submitted", decoded.Submissions[0].Status)
	assert.Equal(t, "validation failed", decoded.Submissions[1].Error)
}

func TestValidationResult_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	vr := ValidationResult{
		TaskRef:           "ET-0001",
		Window:            "2025-Q4",
		Status:            "passed",
		ValidationMode:    "strict",
		CompletenessScore: 0.95,
		TotalChecks:       10,
		PassedChecks:      9,
		FailedChecks:      1,
		Warnings:          2,
		Errors: []ValidationError{
			{Code: "MISSING_FILE", Severity: "error", Message: "File missing"},
		},
		WarningsList: []ValidationError{
			{Code: "FORMAT_WARNING", Severity: "warning", Message: "Consider better format"},
		},
		Checks: []ValidationCheck{
			{Code: "FILE_EXISTS", Name: "File Existence", Status: "passed", Severity: "error", Message: "OK"},
		},
		ReadyForSubmission:  true,
		ValidationTimestamp: now,
	}

	data, err := json.Marshal(vr)
	require.NoError(t, err)

	var decoded ValidationResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, vr.TaskRef, decoded.TaskRef)
	assert.Equal(t, vr.CompletenessScore, decoded.CompletenessScore)
	assert.True(t, decoded.ReadyForSubmission)
	assert.Len(t, decoded.Errors, 1)
	assert.Len(t, decoded.WarningsList, 1)
	assert.Len(t, decoded.Checks, 1)
}

func TestSubmissionHistory_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	history := SubmissionHistory{
		TaskRef: "ET-0001",
		Window:  "2025-Q4",
		Entries: []SubmissionHistoryEntry{
			{
				SubmissionID: "sub-001",
				SubmittedAt:  now,
				SubmittedBy:  "user@test.com",
				Status:       "submitted",
				FileCount:    3,
				Notes:        "First attempt",
				BatchID:      "batch-001",
			},
			{
				SubmissionID: "sub-002",
				SubmittedAt:  now,
				SubmittedBy:  "user@test.com",
				Status:       "accepted",
				FileCount:    3,
			},
		},
	}

	data, err := json.Marshal(history)
	require.NoError(t, err)

	var decoded SubmissionHistory
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, history.TaskRef, decoded.TaskRef)
	assert.Len(t, decoded.Entries, 2)
	assert.Equal(t, "sub-001", decoded.Entries[0].SubmissionID)
	assert.Equal(t, "accepted", decoded.Entries[1].Status)
}

func TestSubmitEvidenceRequest_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	req := SubmitEvidenceRequest{
		TaskID: "1",
		Content:          "Evidence content here",
		ContentType:      "markdown",
		CollectionWindow: "2025-Q4",
		CollectionDate:   "2025-12-15T10:00:00Z",
		Sources: []EvidenceSourceRef{
			{Type: "terraform", Tool: "terraform-scanner", Timestamp: "2025-12-15T10:00:00Z"},
		},
		Metadata:        map[string]interface{}{"version": "1.0"},
		ControlsCovered: []string{"CC6.1", "CC6.8"},
		Attachments: []AttachmentRef{
			{FileID: "file-001", Filename: "report.pdf", Size: 5120},
		},
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var decoded SubmitEvidenceRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, req.TaskID, decoded.TaskID)
	assert.Equal(t, req.ContentType, decoded.ContentType)
	assert.Len(t, decoded.Sources, 1)
	assert.Len(t, decoded.ControlsCovered, 2)
	assert.Len(t, decoded.Attachments, 1)
}

func TestSubmitEvidenceResponse_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	resp := SubmitEvidenceResponse{
		SubmissionID: "sub-001",
		Status:       "accepted",
		Message:      "Evidence received",
		TaskID: "1",
		ReceivedAt:   now,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded SubmitEvidenceResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, resp.SubmissionID, decoded.SubmissionID)
	assert.Equal(t, resp.Status, decoded.Status)
}

func TestSubmissionStatusResponse_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	resp := SubmissionStatusResponse{
		SubmissionID: "sub-001",
		TaskID: "1",
		Status:       "accepted",
		SubmittedAt:  now,
		ReviewedAt:   &now,
		ReviewedBy:   "auditor@test.com",
		ReviewNotes:  "Looks good",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded SubmissionStatusResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, resp.SubmissionID, decoded.SubmissionID)
	assert.NotNil(t, decoded.ReviewedAt)
	assert.Equal(t, resp.ReviewedBy, decoded.ReviewedBy)
}

func TestFileUploadResponse_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	resp := FileUploadResponse{
		FileID:   "file-001",
		Filename: "evidence.pdf",
		Size:     2048,
		Uploaded: now,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded FileUploadResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, resp.FileID, decoded.FileID)
	assert.Equal(t, resp.Size, decoded.Size)
}

func TestSubmissionListResponse_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	resp := SubmissionListResponse{
		TaskID: "1",
		Total:  2,
		Submissions: []SubmissionStatusResponse{
			{SubmissionID: "sub-001", TaskID: "1", Status: "accepted", SubmittedAt: now},
			{SubmissionID: "sub-002", TaskID: "1", Status: "pending", SubmittedAt: now},
		},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded SubmissionListResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, 2, decoded.Total)
	assert.Len(t, decoded.Submissions, 2)
}
