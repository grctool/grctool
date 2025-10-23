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

package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/services/submission"
	"github.com/grctool/grctool/internal/services/validation"
	"github.com/grctool/grctool/internal/storage"
	"github.com/grctool/grctool/internal/tugboat"
)

// EvidenceSubmitterTool submits evidence to Tugboat Logic
type EvidenceSubmitterTool struct {
	config            *config.Config
	logger            logger.Logger
	storage           *storage.Storage
	submissionService *submission.SubmissionService
}

// NewEvidenceSubmitterTool creates a new evidence submitter tool
func NewEvidenceSubmitterTool(cfg *config.Config, log logger.Logger) (*EvidenceSubmitterTool, error) {
	// Initialize storage
	storage, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Initialize Tugboat client (may be nil if not configured)
	var tugboatClient *tugboat.Client
	if cfg.Tugboat.BaseURL != "" {
		tugboatClient = tugboat.NewClient(&cfg.Tugboat, nil)
	}

	// Initialize submission service
	submissionService := submission.NewSubmissionService(
		storage,
		tugboatClient,
		cfg.Tugboat.OrgID,
		cfg.Tugboat.CollectorURLs,
	)

	return &EvidenceSubmitterTool{
		config:            cfg,
		logger:            log,
		storage:           storage,
		submissionService: submissionService,
	}, nil
}

// GetClaudeToolDefinition returns the tool definition for Claude
func (t *EvidenceSubmitterTool) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        "evidence-submitter",
		Description: "Submit evidence to Tugboat Logic for compliance review. Validates and uploads evidence files.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"task_ref": map[string]interface{}{
					"type":        "string",
					"description": "Evidence task reference (e.g., ET-0001) to submit",
				},
				"window": map[string]interface{}{
					"type":        "string",
					"description": "Collection window (e.g., 2025-Q4) to submit",
				},
				"notes": map[string]interface{}{
					"type":        "string",
					"description": "Submission notes or comments",
				},
				"skip_validation": map[string]interface{}{
					"type":        "boolean",
					"description": "Skip validation checks before submission",
					"default":     false,
				},
				"validation_mode": map[string]interface{}{
					"type":        "string",
					"description": "Validation mode if not skipped: strict, lenient, advisory",
					"enum":        []string{"strict", "lenient", "advisory"},
					"default":     "strict",
				},
				"dry_run": map[string]interface{}{
					"type":        "boolean",
					"description": "Preview submission without actually submitting",
					"default":     false,
				},
			},
			"required": []string{"task_ref", "window"},
		},
	}
}

// Execute runs the evidence submitter tool
func (t *EvidenceSubmitterTool) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	t.logger.Info("Starting evidence submission",
		logger.Field{Key: "params", Value: params})

	// Parse parameters
	taskRef, ok := params["task_ref"].(string)
	if !ok || taskRef == "" {
		return "", nil, fmt.Errorf("task_ref is required")
	}

	window, ok := params["window"].(string)
	if !ok || window == "" {
		return "", nil, fmt.Errorf("window is required")
	}

	notes, _ := params["notes"].(string)
	skipValidation, _ := params["skip_validation"].(bool)
	validationModeStr, _ := params["validation_mode"].(string)
	dryRun, _ := params["dry_run"].(bool)

	if validationModeStr == "" {
		validationModeStr = "strict"
	}

	// Parse validation mode
	var mode validation.EvidenceValidationMode
	switch validationModeStr {
	case "strict":
		mode = validation.EvidenceValidationModeStrict
	case "lenient":
		mode = validation.EvidenceValidationModeLenient
	case "advisory":
		mode = validation.EvidenceValidationModeAdvisory
	default:
		mode = validation.EvidenceValidationModeStrict
	}

	// Get submitter email from config or environment
	submittedBy := t.config.Tugboat.OrgID + "-user"

	// Build submission request
	req := &submission.SubmitRequest{
		TaskRef:        taskRef,
		Window:         window,
		Notes:          notes,
		SkipValidation: skipValidation,
		ValidationMode: mode,
		SubmittedBy:    submittedBy,
	}

	// Handle dry run
	if dryRun {
		return t.dryRunSubmission(req)
	}

	// Submit evidence
	resp, err := t.submissionService.Submit(ctx, req)
	if err != nil {
		return "", nil, fmt.Errorf("submission failed: %w", err)
	}

	// Format response
	output := t.formatResponse(resp)

	// Create evidence source
	source := &models.EvidenceSource{
		Type:     "submission",
		Resource: fmt.Sprintf("%s/%s", taskRef, window),
		Content:  output,
	}

	return output, source, nil
}

// dryRunSubmission performs a dry run without actually submitting
func (t *EvidenceSubmitterTool) dryRunSubmission(req *submission.SubmitRequest) (string, *models.EvidenceSource, error) {
	// Get evidence files
	files, err := t.storage.GetEvidenceFiles(req.TaskRef, req.Window)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get evidence files: %w", err)
	}

	output := fmt.Sprintf("Dry Run: Evidence Submission for %s (%s)\n", req.TaskRef, req.Window)
	output += fmt.Sprintf("==========================================\n\n")
	output += fmt.Sprintf("Files to submit: %d\n", len(files))

	var totalSize int64
	for _, file := range files {
		totalSize += file.SizeBytes
		output += fmt.Sprintf("  - %s (%d bytes)\n", file.Filename, file.SizeBytes)
	}

	output += fmt.Sprintf("\nTotal size: %d bytes\n", totalSize)
	output += fmt.Sprintf("Notes: %s\n", req.Notes)
	output += fmt.Sprintf("\nValidation: %s (skip: %v)\n", req.ValidationMode, req.SkipValidation)
	output += fmt.Sprintf("\n✓ Dry run complete - no submission made\n")

	source := &models.EvidenceSource{
		Type:     "submission-dryrun",
		Resource: fmt.Sprintf("%s/%s", req.TaskRef, req.Window),
		Content:  output,
	}

	return output, source, nil
}

// formatResponse formats the submission response
func (t *EvidenceSubmitterTool) formatResponse(resp *submission.SubmitResponse) string {
	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error formatting response: %v", err)
	}

	output := string(data)

	// Add human-readable summary
	if resp.Success {
		output += fmt.Sprintf("\n\n✓ Evidence submitted successfully\n")
		output += fmt.Sprintf("Submission ID: %s\n", resp.SubmissionID)
		output += fmt.Sprintf("Status: %s\n", resp.Status)
	} else {
		output += fmt.Sprintf("\n\n✗ Submission failed\n")
		output += fmt.Sprintf("Reason: %s\n", resp.Message)
	}

	return output
}
