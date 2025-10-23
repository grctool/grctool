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
	"github.com/grctool/grctool/internal/services/validation"
	"github.com/grctool/grctool/internal/storage"
)

// EvidenceSubmissionValidatorTool validates evidence for submission readiness
type EvidenceSubmissionValidatorTool struct {
	config    *config.Config
	logger    logger.Logger
	storage   *storage.Storage
	validator *validation.EvidenceValidationService
}

// NewEvidenceSubmissionValidatorTool creates a new evidence submission validator tool
func NewEvidenceSubmissionValidatorTool(cfg *config.Config, log logger.Logger) (*EvidenceSubmissionValidatorTool, error) {
	// Initialize storage
	storage, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	validator := validation.NewEvidenceValidationService(storage)

	return &EvidenceSubmissionValidatorTool{
		config:    cfg,
		logger:    log,
		storage:   storage,
		validator: validator,
	}, nil
}

// GetClaudeToolDefinition returns the tool definition for Claude
func (t *EvidenceSubmissionValidatorTool) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        "evidence-submission-validator",
		Description: "Validate evidence for submission readiness. Checks completeness, file formats, and compliance with submission requirements.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"task_ref": map[string]interface{}{
					"type":        "string",
					"description": "Evidence task reference (e.g., ET-0001) to validate",
				},
				"window": map[string]interface{}{
					"type":        "string",
					"description": "Collection window (e.g., 2025-Q4) to validate",
				},
				"validation_mode": map[string]interface{}{
					"type":        "string",
					"description": "Validation mode: strict, lenient, advisory, or skip",
					"enum":        []string{"strict", "lenient", "advisory", "skip"},
					"default":     "strict",
				},
				"output_format": map[string]interface{}{
					"type":        "string",
					"description": "Output format: json, yaml, or text",
					"enum":        []string{"json", "yaml", "text"},
					"default":     "json",
				},
			},
			"required": []string{"task_ref", "window"},
		},
	}
}

// Execute runs the evidence submission validation tool
func (t *EvidenceSubmissionValidatorTool) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	t.logger.Info("Starting evidence submission validation",
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

	validationMode, _ := params["validation_mode"].(string)
	if validationMode == "" {
		validationMode = "strict"
	}

	outputFormat, _ := params["output_format"].(string)
	if outputFormat == "" {
		outputFormat = "json"
	}

	// Parse validation mode
	var mode validation.EvidenceValidationMode
	switch validationMode {
	case "strict":
		mode = validation.EvidenceValidationModeStrict
	case "lenient":
		mode = validation.EvidenceValidationModeLenient
	case "advisory":
		mode = validation.EvidenceValidationModeAdvisory
	case "skip":
		mode = validation.EvidenceValidationModeSkip
	default:
		mode = validation.EvidenceValidationModeStrict
	}

	// Run validation
	req := &validation.EvidenceValidationRequest{
		TaskRef:        taskRef,
		Window:         window,
		ValidationMode: mode,
	}

	result, err := t.validator.ValidateEvidence(req)
	if err != nil {
		return "", nil, fmt.Errorf("validation failed: %w", err)
	}

	// Format output
	var output string
	switch outputFormat {
	case "json":
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", nil, fmt.Errorf("failed to marshal JSON: %w", err)
		}
		output = string(data)
	case "text":
		output = t.formatTextOutput(result)
	default:
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", nil, fmt.Errorf("failed to marshal JSON: %w", err)
		}
		output = string(data)
	}

	// Create evidence source
	source := &models.EvidenceSource{
		Type:     "validation",
		Resource: fmt.Sprintf("%s/%s", taskRef, window),
		Content:  output,
	}

	return output, source, nil
}

// formatTextOutput formats the validation result as text
func (t *EvidenceSubmissionValidatorTool) formatTextOutput(result *models.ValidationResult) string {
	output := fmt.Sprintf("Evidence Validation Report: %s (%s)\n", result.TaskRef, result.Window)
	output += fmt.Sprintf("==============================================\n\n")
	output += fmt.Sprintf("Status: %s\n", result.Status)
	output += fmt.Sprintf("Ready for Submission: %v\n", result.ReadyForSubmission)
	output += fmt.Sprintf("Completeness Score: %.1f%%\n\n", result.CompletenessScore*100)

	output += fmt.Sprintf("Checks: %d total, %d passed, %d failed, %d warnings\n\n",
		result.TotalChecks, result.PassedChecks, result.FailedChecks, result.Warnings)

	if len(result.Errors) > 0 {
		output += "Errors:\n"
		for _, err := range result.Errors {
			output += fmt.Sprintf("  ✗ [%s] %s\n", err.Code, err.Message)
			if err.Suggestion != "" {
				output += fmt.Sprintf("    Suggestion: %s\n", err.Suggestion)
			}
		}
		output += "\n"
	}

	if len(result.WarningsList) > 0 {
		output += "Warnings:\n"
		for _, warn := range result.WarningsList {
			output += fmt.Sprintf("  ⚠ [%s] %s\n", warn.Code, warn.Message)
		}
		output += "\n"
	}

	output += "Evidence Files:\n"
	for _, file := range result.EvidenceFiles {
		output += fmt.Sprintf("  - %s (%d bytes)\n", file.Filename, file.SizeBytes)
	}

	return output
}
