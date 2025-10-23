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

package submission

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/services/validation"
	"github.com/grctool/grctool/internal/storage"
	"github.com/grctool/grctool/internal/tugboat"
)

// SubmissionService handles evidence submission to Tugboat
type SubmissionService struct {
	storage       *storage.Storage
	tugboatClient *tugboat.Client
	validator     *validation.EvidenceValidationService
	orgID         string
	collectorURLs map[string]string // TaskRef -> Collector URL mapping
}

// NewSubmissionService creates a new submission service
func NewSubmissionService(
	storage *storage.Storage,
	tugboatClient *tugboat.Client,
	orgID string,
	collectorURLs map[string]string,
) *SubmissionService {
	return &SubmissionService{
		storage:       storage,
		tugboatClient: tugboatClient,
		validator:     validation.NewEvidenceValidationService(storage),
		orgID:         orgID,
		collectorURLs: collectorURLs,
	}
}

// SubmitRequest defines a submission request
type SubmitRequest struct {
	TaskRef        string
	Window         string
	Notes          string
	SkipValidation bool
	ValidationMode validation.EvidenceValidationMode
	SubmittedBy    string
}

// SubmitResponse defines the submission response
type SubmitResponse struct {
	Success          bool
	SubmissionID     string
	Status           string
	Message          string
	ValidationResult *models.ValidationResult
	Submission       *models.EvidenceSubmission
}

// Submit submits evidence for a task/window
func (s *SubmissionService) Submit(ctx context.Context, req *SubmitRequest) (*SubmitResponse, error) {
	// Step 1: Validate evidence unless skipped
	var validationResult *models.ValidationResult
	if !req.SkipValidation {
		valReq := &validation.EvidenceValidationRequest{
			TaskRef:        req.TaskRef,
			Window:         req.Window,
			ValidationMode: req.ValidationMode,
		}

		var err error
		validationResult, err = s.validator.ValidateEvidence(valReq)
		if err != nil {
			return nil, fmt.Errorf("validation failed: %w", err)
		}

		// Check if ready for submission
		if !validationResult.ReadyForSubmission {
			return &SubmitResponse{
				Success:          false,
				Status:           "validation_failed",
				Message:          fmt.Sprintf("Evidence validation failed with %d errors", validationResult.FailedChecks),
				ValidationResult: validationResult,
			}, nil
		}
	}

	// Step 2: Get evidence task details
	task, err := s.getEvidenceTask(req.TaskRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get evidence task: %w", err)
	}

	// Step 3: Prepare submission
	submission, err := s.prepareSubmission(req, task, validationResult)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare submission: %w", err)
	}

	// Step 4: Submit to Tugboat (if client is configured)
	if s.tugboatClient != nil {
		tugboatResp, err := s.submitToTugboat(ctx, submission, task)
		if err != nil {
			// Mark submission as failed
			submission.Status = "submission_failed"
			submission.TugboatResponse = &models.TugboatSubmissionResponse{
				Status:     "failed",
				Message:    err.Error(),
				ReceivedAt: time.Now(),
			}
			s.storage.SaveSubmission(submission)

			return nil, fmt.Errorf("tugboat submission failed: %w", err)
		}

		// Update submission with Tugboat response
		submission.SubmissionID = tugboatResp.SubmissionID
		submission.Status = "submitted"
		submission.TugboatResponse = tugboatResp
		submittedAt := time.Now()
		submission.SubmittedAt = &submittedAt
	} else {
		// No Tugboat client - mark as submitted locally only
		submission.Status = "draft"
		submission.SubmissionID = fmt.Sprintf("local-%d", time.Now().Unix())
	}

	// Step 5: Save submission metadata
	if err := s.storage.SaveSubmission(submission); err != nil {
		return nil, fmt.Errorf("failed to save submission: %w", err)
	}

	// Step 6: Add to history
	historyEntry := models.SubmissionHistoryEntry{
		SubmissionID: submission.SubmissionID,
		SubmittedAt:  time.Now(),
		SubmittedBy:  submission.SubmittedBy,
		Status:       submission.Status,
		FileCount:    submission.TotalFileCount,
		Notes:        submission.Notes,
	}
	if err := s.storage.AddSubmissionHistory(req.TaskRef, req.Window, historyEntry); err != nil {
		// Log error but don't fail
		fmt.Printf("Warning: failed to save submission history: %v\n", err)
	}

	return &SubmitResponse{
		Success:          true,
		SubmissionID:     submission.SubmissionID,
		Status:           submission.Status,
		Message:          "Evidence submitted successfully",
		ValidationResult: validationResult,
		Submission:       submission,
	}, nil
}

// prepareSubmission creates a submission record
func (s *SubmissionService) prepareSubmission(
	req *SubmitRequest,
	task *domain.EvidenceTask,
	validationResult *models.ValidationResult,
) (*models.EvidenceSubmission, error) {
	// Get evidence files
	files, err := s.storage.GetEvidenceFiles(req.TaskRef, req.Window)
	if err != nil {
		return nil, fmt.Errorf("failed to get evidence files: %w", err)
	}

	// Calculate total size
	var totalSize int64
	for _, file := range files {
		totalSize += file.SizeBytes
	}

	submission := &models.EvidenceSubmission{
		TaskID:         task.ID,
		TaskRef:        req.TaskRef,
		Window:         req.Window,
		Status:         "draft",
		CreatedAt:      time.Now(),
		EvidenceFiles:  files,
		TotalFileCount: len(files),
		TotalSizeBytes: totalSize,
		SubmittedBy:    req.SubmittedBy,
		Notes:          req.Notes,
	}

	if validationResult != nil {
		submission.ValidationStatus = validationResult.Status
		submission.ValidationErrors = validationResult.Errors
		submission.ValidationWarnings = validationResult.WarningsList
		submission.CompletenessScore = validationResult.CompletenessScore
		validatedAt := time.Now()
		submission.ValidatedAt = &validatedAt
	}

	return submission, nil
}

// submitToTugboat submits evidence to Tugboat Custom Evidence Integration API
func (s *SubmissionService) submitToTugboat(
	ctx context.Context,
	submission *models.EvidenceSubmission,
	task *domain.EvidenceTask,
) (*models.TugboatSubmissionResponse, error) {
	// Get collector URL for this task from config
	collectorURL, ok := s.collectorURLs[submission.TaskRef]
	if !ok || collectorURL == "" {
		return nil, fmt.Errorf("collector URL not configured for task %s - add to tugboat.collector_urls in config", submission.TaskRef)
	}

	// Get storage base directory to resolve file paths
	baseDir := s.storage.GetBaseDir()

	// Submit each evidence file individually
	// Note: Custom Evidence Integration API accepts one file per submission
	var lastResponse *tugboat.SubmitEvidenceResponse
	submittedFiles := 0
	failedFiles := []string{}
	collectionDate := time.Now() // Use current time as collection date

	for _, fileRef := range submission.EvidenceFiles {
		// Resolve full file path
		filePath := filepath.Join(baseDir, fileRef.RelativePath)

		// Validate file type before submission
		if err := tugboat.ValidateFileType(fileRef.Filename); err != nil {
			failedFiles = append(failedFiles, fmt.Sprintf("%s: %v", fileRef.Filename, err))
			continue
		}

		// Build submission request for this file
		submitReq := &tugboat.SubmitEvidenceRequest{
			CollectorURL:  collectorURL,
			FilePath:      filePath,
			CollectedDate: collectionDate,
			ContentType:   s.getContentType(fileRef.Filename),
		}

		// Submit to Tugboat
		resp, err := s.tugboatClient.SubmitEvidence(ctx, submitReq)
		if err != nil {
			// Collect error but continue with other files
			failedFiles = append(failedFiles, fmt.Sprintf("%s: %v", fileRef.Filename, err))
			continue
		}

		lastResponse = resp
		submittedFiles++
	}

	if submittedFiles == 0 {
		if len(failedFiles) > 0 {
			return nil, fmt.Errorf("all %d file(s) failed submission:\n  - %s", len(failedFiles), failedFiles[0])
		}
		return nil, fmt.Errorf("no evidence files to submit")
	}

	// Build response message
	message := fmt.Sprintf("Successfully submitted %d file(s) to Tugboat", submittedFiles)
	if len(failedFiles) > 0 {
		message = fmt.Sprintf("Submitted %d file(s), %d failed", submittedFiles, len(failedFiles))
	}

	// Return response from last submission
	// Note: In the Custom Evidence Integration API, each file is submitted separately
	// so we track the last successful response
	response := &models.TugboatSubmissionResponse{
		SubmissionID: fmt.Sprintf("batch-%d-files-%d", time.Now().Unix(), submittedFiles),
		Status:       "submitted",
		Message:      message,
		ReceivedAt:   lastResponse.ReceivedAt,
		Metadata: map[string]interface{}{
			"files_submitted": submittedFiles,
			"files_failed":    len(failedFiles),
		},
	}

	// Add failed files to metadata if any
	if len(failedFiles) > 0 {
		response.Metadata["failed_files"] = failedFiles
	}

	return response, nil
}

// getContentType determines the MIME type based on file extension
func (s *SubmissionService) getContentType(filename string) string {
	ext := filepath.Ext(filename)
	if ext == "" {
		return "application/octet-stream"
	}
	ext = ext[1:] // Remove leading dot

	contentTypes := map[string]string{
		"txt":  "text/plain",
		"csv":  "text/csv",
		"json": "application/json",
		"pdf":  "application/pdf",
		"png":  "image/png",
		"gif":  "image/gif",
		"jpg":  "image/jpeg",
		"jpeg": "image/jpeg",
		"md":   "text/markdown",
		"doc":  "application/msword",
		"docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"xls":  "application/vnd.ms-excel",
		"xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"odt":  "application/vnd.oasis.opendocument.text",
		"ods":  "application/vnd.oasis.opendocument.spreadsheet",
	}

	if ct, ok := contentTypes[ext]; ok {
		return ct
	}
	return "application/octet-stream"
}

// buildEvidenceContent builds the evidence content from files
func (s *SubmissionService) buildEvidenceContent(submission *models.EvidenceSubmission) string {
	// For now, just list the files
	// In the future, this could concatenate file contents or create a summary
	content := fmt.Sprintf("# Evidence Submission: %s\n\n", submission.TaskRef)
	content += fmt.Sprintf("**Collection Window**: %s\n\n", submission.Window)
	content += fmt.Sprintf("**Files Submitted**: %d\n\n", len(submission.EvidenceFiles))

	for _, file := range submission.EvidenceFiles {
		content += fmt.Sprintf("- %s (%d bytes)\n", file.Filename, file.SizeBytes)
	}

	if submission.Notes != "" {
		content += fmt.Sprintf("\n## Notes\n\n%s\n", submission.Notes)
	}

	return content
}

// buildEvidenceSources builds the evidence sources list
func (s *SubmissionService) buildEvidenceSources(submission *models.EvidenceSubmission) []models.EvidenceSourceRef {
	sources := []models.EvidenceSourceRef{}

	for _, file := range submission.EvidenceFiles {
		if file.Source != "" {
			sources = append(sources, models.EvidenceSourceRef{
				Type:      "tool",
				Tool:      file.Source,
				Timestamp: time.Now().Format(time.RFC3339),
			})
		}
	}

	return sources
}

// extractControlsCovered extracts controls from evidence files
func (s *SubmissionService) extractControlsCovered(submission *models.EvidenceSubmission) []string {
	controlsMap := make(map[string]bool)

	for _, file := range submission.EvidenceFiles {
		for _, control := range file.ControlsSatisfied {
			controlsMap[control] = true
		}
	}

	controls := []string{}
	for control := range controlsMap {
		controls = append(controls, control)
	}

	return controls
}

// getEvidenceTask gets an evidence task by reference
func (s *SubmissionService) getEvidenceTask(taskRef string) (*domain.EvidenceTask, error) {
	task, err := s.storage.GetEvidenceTask(taskRef)
	if err != nil {
		return nil, fmt.Errorf("task %s not found: %w", taskRef, err)
	}
	return task, nil
}

// GetSubmissionStatus gets the status of a submission
func (s *SubmissionService) GetSubmissionStatus(ctx context.Context, taskRef, window string) (*models.EvidenceSubmission, error) {
	submission, err := s.storage.LoadSubmission(taskRef, window)
	if err != nil {
		return nil, fmt.Errorf("submission not found: %w", err)
	}

	// Note: Custom Evidence Integration API does not support status queries
	// Status is tracked locally only
	// The API is fire-and-forget for file uploads

	return submission, nil
}

// GetSubmissionHistory gets submission history for a task/window
func (s *SubmissionService) GetSubmissionHistory(taskRef, window string) (*models.SubmissionHistory, error) {
	history, err := s.storage.LoadSubmissionHistory(taskRef, window)
	if err != nil {
		return nil, fmt.Errorf("history not found: %w", err)
	}
	return history, nil
}
