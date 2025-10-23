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
}

// NewSubmissionService creates a new submission service
func NewSubmissionService(
	storage *storage.Storage,
	tugboatClient *tugboat.Client,
	orgID string,
) *SubmissionService {
	return &SubmissionService{
		storage:       storage,
		tugboatClient: tugboatClient,
		validator:     validation.NewEvidenceValidationService(storage),
		orgID:         orgID,
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

// submitToTugboat submits evidence to Tugboat API
func (s *SubmissionService) submitToTugboat(
	ctx context.Context,
	submission *models.EvidenceSubmission,
	task *domain.EvidenceTask,
) (*models.TugboatSubmissionResponse, error) {
	// Build submission request
	submitReq := &models.SubmitEvidenceRequest{
		TaskID:           task.ID,
		Content:          s.buildEvidenceContent(submission),
		ContentType:      "markdown",
		CollectionWindow: submission.Window,
		CollectionDate:   time.Now().Format(time.RFC3339),
		Sources:          s.buildEvidenceSources(submission),
		Notes:            submission.Notes,
		ControlsCovered:  s.extractControlsCovered(submission),
	}

	// Submit to Tugboat
	resp, err := s.tugboatClient.SubmitEvidence(ctx, s.orgID, task.ID, submitReq)
	if err != nil {
		return nil, err
	}

	return &models.TugboatSubmissionResponse{
		SubmissionID: resp.SubmissionID,
		Status:       resp.Status,
		Message:      resp.Message,
		ReceivedAt:   resp.ReceivedAt,
		Metadata:     resp.Metadata,
	}, nil
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

	// Optionally refresh from Tugboat if we have a submission ID
	if submission.SubmissionID != "" && s.tugboatClient != nil {
		task, err := s.getEvidenceTask(taskRef)
		if err == nil {
			tugboatStatus, err := s.tugboatClient.GetSubmissionStatus(ctx, s.orgID, task.ID, submission.SubmissionID)
			if err == nil {
				// Update local status
				submission.Status = tugboatStatus.Status
				if tugboatStatus.ReviewedAt != nil {
					submission.AcceptedAt = tugboatStatus.ReviewedAt
				}
				// Save updated status
				s.storage.SaveSubmission(submission)
			}
		}
	}

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
