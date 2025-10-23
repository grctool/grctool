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
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/formatters"
	"github.com/grctool/grctool/internal/interpolation"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/storage"
	"github.com/grctool/grctool/internal/templates"
)

// ControlSummaryGeneratorTool provides control summary generation capabilities
// using template-based approach (no AI API calls, follows "prompt as data" pattern)
type ControlSummaryGeneratorTool struct {
	config          *config.Config
	logger          logger.Logger
	dataService     DataServiceInterface
	storage         *storage.Storage
	templateManager *templates.Manager
}

// NewControlSummaryGeneratorTool creates a new ControlSummaryGeneratorTool
func NewControlSummaryGeneratorTool(cfg *config.Config, log logger.Logger) Tool {
	// Initialize unified storage and data service
	storage, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		log.Error("Failed to initialize unified storage for control summary generator tool",
			logger.Field{Key: "error", Value: err})
		return nil
	}

	// Get singleton template manager
	templateManager, err := templates.GetSingleton(log)
	if err != nil {
		log.Error("Failed to get singleton template manager for control summary generator tool",
			logger.Field{Key: "error", Value: err})
		return nil
	}

	dataService := &SimpleDataService{storage: storage}

	return &ControlSummaryGeneratorTool{
		config:          cfg,
		logger:          log,
		dataService:     dataService,
		storage:         storage,
		templateManager: templateManager,
	}
}

// Name returns the tool name
func (csgt *ControlSummaryGeneratorTool) Name() string {
	return "control-summary-generator"
}

// Description returns the tool description
func (csgt *ControlSummaryGeneratorTool) Description() string {
	return "Generate focused control summaries for evidence tasks using template-based approach (prompt as data pattern)"
}

// GetClaudeToolDefinition returns the tool definition for external AI tools
func (csgt *ControlSummaryGeneratorTool) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        csgt.Name(),
		Description: csgt.Description(),
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"task_ref": map[string]interface{}{
					"type":        "string",
					"description": "Task reference in format ET-101, ET101, or numeric ID (e.g., 328001)",
					"examples":    []string{"ET-101", "ET101", "328001"},
				},
				"control_id": map[string]interface{}{
					"type":        "string",
					"description": "Control ID to generate summary for (e.g., CC1.1, 11057785)",
					"examples":    []string{"CC1.1", "11057785"},
				},
				"output_format": map[string]interface{}{
					"type":        "string",
					"description": "Output format for generated summary",
					"enum":        []string{"markdown", "json"},
					"default":     "markdown",
				},
				"save_to_file": map[string]interface{}{
					"type":        "boolean",
					"description": "Save the generated summary to a file",
					"default":     true,
				},
			},
			"required": []string{"task_ref", "control_id"},
		},
	}
}

// Execute generates control summaries for evidence tasks
func (csgt *ControlSummaryGeneratorTool) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	csgt.logger.Debug("Executing control summary generator", logger.Field{Key: "params", Value: params})

	// Extract and validate parameters
	taskRefRaw, ok := params["task_ref"]
	if !ok {
		return "", nil, fmt.Errorf("task_ref parameter is required")
	}

	taskRef, ok := taskRefRaw.(string)
	if !ok {
		return "", nil, fmt.Errorf("task_ref must be a string")
	}

	controlIDRaw, ok := params["control_id"]
	if !ok {
		return "", nil, fmt.Errorf("control_id parameter is required")
	}

	controlID, ok := controlIDRaw.(string)
	if !ok {
		return "", nil, fmt.Errorf("control_id must be a string")
	}

	// Parse additional parameters
	outputFormat := "markdown"
	if of, exists := params["output_format"]; exists {
		if format, ok := of.(string); ok {
			outputFormat = format
		}
	}

	saveToFile := true
	if stf, exists := params["save_to_file"]; exists {
		if flag, ok := stf.(bool); ok {
			saveToFile = flag
		}
	}

	// Parse task reference to get numeric ID
	taskID, err := csgt.parseTaskReference(taskRef)
	if err != nil {
		return "", nil, fmt.Errorf("invalid task reference '%s': %w", taskRef, err)
	}

	// Parse control ID to get storage ID
	controlStorageID, err := csgt.parseControlID(controlID)
	if err != nil {
		return "", nil, fmt.Errorf("invalid control ID '%s': %w", controlID, err)
	}

	// Get task details
	task, err := csgt.dataService.GetEvidenceTask(ctx, taskID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get evidence task: %w", err)
	}

	// Get control details
	control, err := csgt.dataService.GetControl(ctx, controlStorageID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get control: %w", err)
	}

	// Build template context for control summary generation
	templateContext := csgt.buildControlSummaryContext(task, control)

	// Generate the summary using template-based approach
	summaryText, err := csgt.generateTemplateSummary(templateContext)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate control summary: %w", err)
	}

	// Save summary to file if requested
	var filePath string
	if saveToFile {
		filePath, err = csgt.saveSummaryToFile(summaryText, task, control)
		if err != nil {
			csgt.logger.Warn("Failed to save summary to file",
				logger.Field{Key: "error", Value: err})
		}
	}

	// Generate structured summary data for JSON output
	structuredSummary := csgt.extractStructuredSummary(summaryText, task, control)

	// Generate JSON response
	responseJSON := fmt.Sprintf(`{
	"success": true,
	"summary_text": %q,
	"structured_summary": {
		"control_id": %d,
		"control_name": %q,
		"task_id": %d,
		"task_name": %q,
		"summary": %q,
		"key_requirements": %s,
		"verification_points": %s
	},
	"summary_metadata": {
		"task_id": %d,
		"task_reference_id": %q,
		"control_id": %d,
		"output_format": %q,
		"generated_at": %q,
		"generation_mode": "template-based"
	},
	"file_path": %q
}`,
		summaryText,
		control.ID,
		control.Name,
		task.ID,
		task.Name,
		structuredSummary.Summary,
		csgt.toJSONArray(structuredSummary.KeyRequirements),
		csgt.toJSONArray(structuredSummary.VerificationPoints),
		task.ID,
		task.ReferenceID,
		control.ID,
		outputFormat,
		time.Now().Format(time.RFC3339),
		filePath)

	// Create evidence source
	source := &models.EvidenceSource{
		Type:        "control_summary",
		Resource:    fmt.Sprintf("Control summary for %d in context of task %s", control.ID, task.ReferenceID),
		Content:     summaryText,
		Relevance:   csgt.calculateRelevance(task, control),
		ExtractedAt: time.Now(),
		Metadata: map[string]interface{}{
			"task_id":         task.ID,
			"task_reference":  task.ReferenceID,
			"control_id":      control.ID,
			"control_name":    control.Name,
			"output_format":   outputFormat,
			"summary_length":  len(summaryText),
			"file_path":       filePath,
			"generation_mode": "template-based",
		},
	}

	csgt.logger.Info("Successfully generated control summary",
		logger.Field{Key: "task_id", Value: taskID},
		logger.Field{Key: "task_reference", Value: task.ReferenceID},
		logger.Field{Key: "control_id", Value: control.ID},
		logger.Field{Key: "summary_length", Value: len(summaryText)},
		logger.Field{Key: "file_path", Value: filePath},
		logger.Field{Key: "generation_mode", Value: "template-based"})

	return responseJSON, source, nil
}

// buildControlSummaryContext builds template context for control summary generation
func (csgt *ControlSummaryGeneratorTool) buildControlSummaryContext(task *domain.EvidenceTask, control *domain.Control) *models.ControlSummaryContext {
	// Convert domain objects to models
	modelsTask := csgt.convertDomainTaskToModels(task)
	modelsControl := csgt.convertDomainControlToModels(control)

	return &models.ControlSummaryContext{
		Task:    *modelsTask,
		Control: *modelsControl,
	}
}

// generateTemplateSummary generates control summary using the control_summary template
func (csgt *ControlSummaryGeneratorTool) generateTemplateSummary(context *models.ControlSummaryContext) (string, error) {
	// Use the control_summary template
	summaryText, err := csgt.templateManager.Execute("control_summary", context)
	if err != nil {
		csgt.logger.Warn("Failed to execute control_summary template, falling back to basic summary",
			logger.Field{Key: "error", Value: err})

		// Fallback to basic summary generation
		return csgt.generateBasicSummary(context), nil
	}

	return summaryText, nil
}

// generateBasicSummary generates a basic control summary as fallback
func (csgt *ControlSummaryGeneratorTool) generateBasicSummary(context *models.ControlSummaryContext) string {
	var summary strings.Builder

	summary.WriteString(fmt.Sprintf("# Control Summary: %s\n\n", context.Control.Name))
	summary.WriteString(fmt.Sprintf("**Control ID:** %d\n", context.Control.ID))
	summary.WriteString(fmt.Sprintf("**Evidence Task:** %s (%s)\n", context.Task.Name, context.Task.ReferenceID))

	if context.Control.Category != "" {
		summary.WriteString(fmt.Sprintf("**Category:** %s\n", context.Control.Category))
	}

	summary.WriteString("\n## Control Relevance to Evidence Task\n")
	summary.WriteString(fmt.Sprintf("This control supports the evidence collection task: %s\n\n", context.Task.Description))

	if context.Control.Body != "" {
		summary.WriteString("## Control Description\n")
		summary.WriteString(context.Control.Body)
		summary.WriteString("\n\n")
	}

	summary.WriteString("## Key Requirements\n")
	summary.WriteString("• Review control implementation documentation\n")
	summary.WriteString("• Verify technical control configurations\n")
	summary.WriteString("• Collect evidence of control effectiveness\n\n")

	summary.WriteString("## Verification Points\n")
	summary.WriteString("• Configuration screenshots or exports\n")
	summary.WriteString("• Audit logs demonstrating control operation\n")
	summary.WriteString("• Testing evidence showing control effectiveness\n\n")

	summary.WriteString("---\n")
	summary.WriteString("*This summary was generated using template-based analysis to focus on evidence collection requirements.*\n")

	return summary.String()
}

// extractStructuredSummary extracts structured data from the generated summary text
func (csgt *ControlSummaryGeneratorTool) extractStructuredSummary(summaryText string, task *domain.EvidenceTask, control *domain.Control) models.AIControlSummary {
	// This is a simplified extraction - in a real implementation, you might use
	// more sophisticated parsing or generate structured data directly from the template

	// Extract summary (first paragraph or sentence)
	lines := strings.Split(summaryText, "\n")
	var summary string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "**") {
			summary = line
			break
		}
	}

	// Default key requirements
	keyRequirements := []string{
		"Review control implementation documentation",
		"Verify technical control configurations",
		"Collect evidence of control effectiveness",
	}

	// Default verification points
	verificationPoints := []string{
		"Configuration screenshots or exports",
		"Audit logs demonstrating control operation",
		"Testing evidence showing control effectiveness",
	}

	return models.AIControlSummary{
		ControlID:          control.ID,
		ControlName:        control.Name,
		Summary:            summary,
		KeyRequirements:    keyRequirements,
		VerificationPoints: verificationPoints,
	}
}

// saveSummaryToFile saves the generated summary to a file
func (csgt *ControlSummaryGeneratorTool) saveSummaryToFile(summaryText string, task *domain.EvidenceTask, control *domain.Control) (string, error) {
	// Create filename with task reference, control ID, and sanitized names
	interpolatorConfig := interpolation.InterpolatorConfig{
		Variables:         make(map[string]string),
		Enabled:           false,
		OnMissingVariable: interpolation.MissingVariableIgnore,
	}
	interpolator := interpolation.NewStandardInterpolator(interpolatorConfig)
	baseFormatter := formatters.NewBaseFormatter(interpolator)

	// Sanitize names for filename
	sanitizedTaskName := baseFormatter.SanitizeFilename(task.Name)
	sanitizedControlName := baseFormatter.SanitizeFilename(control.Name)

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("control_summary_%s_%d_%s_%s_%s.md",
		task.ReferenceID,
		control.ID,
		sanitizedTaskName,
		sanitizedControlName,
		timestamp)

	// Ensure summaries directory exists
	summaryDir := filepath.Join(csgt.config.Storage.DataDir, "summaries", "controls")
	if err := os.MkdirAll(summaryDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create summary directory: %w", err)
	}

	// Save summary to file
	filePath := filepath.Join(summaryDir, filename)
	if err := os.WriteFile(filePath, []byte(summaryText), 0644); err != nil {
		return "", fmt.Errorf("failed to save summary: %w", err)
	}

	return filePath, nil
}

// Helper methods

func (csgt *ControlSummaryGeneratorTool) parseTaskReference(taskRef string) (int, error) {
	validator := NewValidator(csgt.config.Storage.DataDir)

	result, err := validator.ValidateTaskReference(taskRef)
	if err != nil {
		return 0, err
	}

	if !result.Valid {
		return 0, fmt.Errorf("invalid task reference format")
	}

	normalizedRef := result.Normalized["task_ref"]
	if normalizedRef == "" {
		normalizedRef = taskRef
	}

	taskID, err := strconv.Atoi(normalizedRef)
	if err != nil {
		return 0, fmt.Errorf("failed to convert task reference to numeric ID: %w", err)
	}

	return taskID, nil
}

func (csgt *ControlSummaryGeneratorTool) parseControlID(controlID string) (string, error) {
	// Handle different control ID formats (CC1.1, 11057785, etc.)
	// The control ID should match the filename in the storage system

	// If it's a SOC2 control code like CC1.1, convert to filename format
	if strings.Contains(controlID, ".") {
		// Convert CC1.1 to CC1_1 to match filename
		return strings.ReplaceAll(controlID, ".", "_"), nil
	}

	// If it's already in the right format (CC1_1, numeric ID), use as-is
	return controlID, nil
}

func (csgt *ControlSummaryGeneratorTool) calculateRelevance(task *domain.EvidenceTask, control *domain.Control) float64 {
	base := 0.7 // Base relevance for control summary

	// Increase relevance if control category matches common evidence task categories
	relevantCategories := []string{"access", "security", "privacy", "monitoring", "encryption"}
	for _, category := range relevantCategories {
		if strings.Contains(strings.ToLower(control.Category), category) ||
			strings.Contains(strings.ToLower(task.Name), category) {
			base += 0.1
			break
		}
	}

	// Increase relevance based on control description length (more content = potentially more relevant)
	if len(control.Description) > 500 {
		base += 0.1
	}

	if base > 1.0 {
		base = 1.0
	}

	return base
}

func (csgt *ControlSummaryGeneratorTool) toJSONArray(items []string) string {
	if len(items) == 0 {
		return "[]"
	}

	var result strings.Builder
	result.WriteString("[")
	for i, item := range items {
		if i > 0 {
			result.WriteString(", ")
		}
		result.WriteString(fmt.Sprintf("%q", item))
	}
	result.WriteString("]")
	return result.String()
}

func (csgt *ControlSummaryGeneratorTool) Category() string {
	return "evidence-management"
}

func (csgt *ControlSummaryGeneratorTool) Version() string {
	return "1.0.0"
}

// convertDomainTaskToModels converts a domain task to models task
func (csgt *ControlSummaryGeneratorTool) convertDomainTaskToModels(task *domain.EvidenceTask) *models.EvidenceTaskDetails {
	var lastCollectedStr *string
	if task.LastCollected != nil {
		str := task.LastCollected.Format(time.RFC3339)
		lastCollectedStr = &str
	}

	return &models.EvidenceTaskDetails{
		EvidenceTask: models.EvidenceTask{
			ID:                 task.ID,
			Name:               task.Name,
			Description:        task.Description,
			Guidance:           task.Guidance,
			CollectionInterval: task.CollectionInterval,
			Priority:           task.Priority,
			Status:             task.Status,
			Completed:          task.Completed,
			LastCollected:      lastCollectedStr,
			DueDaysBefore:      task.DueDaysBefore,
			AdHoc:              task.AdHoc,
			Sensitive:          task.Sensitive,
			CreatedAt:          task.CreatedAt,
			UpdatedAt:          task.UpdatedAt,
			Framework:          task.Framework,
			ReferenceID:        task.ReferenceID,
		},
	}
}

// convertDomainControlToModels converts a domain control to models control
func (csgt *ControlSummaryGeneratorTool) convertDomainControlToModels(control *domain.Control) *models.Control {
	return &models.Control{
		ID:       control.ID,
		Name:     control.Name,
		Body:     control.Description, // domain.Control uses Description, models.Control uses Body
		Category: control.Category,
		Status:   control.Status,
	}
}
