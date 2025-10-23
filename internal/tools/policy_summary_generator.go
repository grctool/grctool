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

// PolicySummaryGeneratorTool provides policy summary generation capabilities
// using template-based approach (no AI API calls, follows "prompt as data" pattern)
type PolicySummaryGeneratorTool struct {
	config          *config.Config
	logger          logger.Logger
	dataService     DataServiceInterface
	storage         *storage.Storage
	templateManager *templates.Manager
}

// NewPolicySummaryGeneratorTool creates a new PolicySummaryGeneratorTool
func NewPolicySummaryGeneratorTool(cfg *config.Config, log logger.Logger) Tool {
	// Initialize unified storage and data service
	storage, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		log.Error("Failed to initialize unified storage for policy summary generator tool",
			logger.Field{Key: "error", Value: err})
		return nil
	}

	// Get singleton template manager
	templateManager, err := templates.GetSingleton(log)
	if err != nil {
		log.Error("Failed to get singleton template manager for policy summary generator tool",
			logger.Field{Key: "error", Value: err})
		return nil
	}

	dataService := &SimpleDataService{storage: storage}

	return &PolicySummaryGeneratorTool{
		config:          cfg,
		logger:          log,
		dataService:     dataService,
		storage:         storage,
		templateManager: templateManager,
	}
}

// Name returns the tool name
func (psgt *PolicySummaryGeneratorTool) Name() string {
	return "policy-summary-generator"
}

// Description returns the tool description
func (psgt *PolicySummaryGeneratorTool) Description() string {
	return "Generate focused policy summaries for evidence tasks using template-based approach (prompt as data pattern)"
}

// GetClaudeToolDefinition returns the tool definition for external AI tools
func (psgt *PolicySummaryGeneratorTool) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        psgt.Name(),
		Description: psgt.Description(),
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"task_ref": map[string]interface{}{
					"type":        "string",
					"description": "Task reference in format ET-101, ET101, or numeric ID (e.g., 328001)",
					"examples":    []string{"ET-101", "ET101", "328001"},
				},
				"policy_id": map[string]interface{}{
					"type":        "string",
					"description": "Policy ID to generate summary for (e.g., POL-001, 94641)",
					"examples":    []string{"POL-001", "94641"},
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
			"required": []string{"task_ref", "policy_id"},
		},
	}
}

// Execute generates policy summaries for evidence tasks
func (psgt *PolicySummaryGeneratorTool) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	psgt.logger.Debug("Executing policy summary generator", logger.Field{Key: "params", Value: params})

	// Extract and validate parameters
	taskRefRaw, ok := params["task_ref"]
	if !ok {
		return "", nil, fmt.Errorf("task_ref parameter is required")
	}

	taskRef, ok := taskRefRaw.(string)
	if !ok {
		return "", nil, fmt.Errorf("task_ref must be a string")
	}

	policyIDRaw, ok := params["policy_id"]
	if !ok {
		return "", nil, fmt.Errorf("policy_id parameter is required")
	}

	policyID, ok := policyIDRaw.(string)
	if !ok {
		return "", nil, fmt.Errorf("policy_id must be a string")
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
	taskID, err := psgt.parseTaskReference(taskRef)
	if err != nil {
		return "", nil, fmt.Errorf("invalid task reference '%s': %w", taskRef, err)
	}

	// Get task details
	task, err := psgt.dataService.GetEvidenceTask(ctx, taskID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get evidence task: %w", err)
	}

	// Get policy details using original policy ID (supports POL-001, numeric IDs, etc.)
	policy, err := psgt.dataService.GetPolicy(ctx, policyID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get policy: %w", err)
	}

	// Build template context for policy summary generation
	templateContext := psgt.buildPolicySummaryContext(task, policy)

	// Generate the summary using template-based approach
	summaryText, err := psgt.generateTemplateSummary(templateContext)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate policy summary: %w", err)
	}

	// Save summary to file if requested
	var filePath string
	if saveToFile {
		filePath, err = psgt.saveSummaryToFile(summaryText, task, policy)
		if err != nil {
			psgt.logger.Warn("Failed to save summary to file",
				logger.Field{Key: "error", Value: err})
		}
	}

	// Generate structured summary data for JSON output
	structuredSummary := psgt.extractStructuredSummary(summaryText, task, policy)

	// Generate JSON response
	responseJSON := fmt.Sprintf(`{
	"success": true,
	"summary_text": %q,
	"structured_summary": {
		"policy_id": %q,
		"policy_name": %q,
		"task_id": %d,
		"task_name": %q,
		"summary": %q,
		"key_requirements": %s,
		"relevant_sections": %s
	},
	"summary_metadata": {
		"task_id": %d,
		"task_reference_id": %q,
		"policy_id": %q,
		"output_format": %q,
		"generated_at": %q,
		"generation_mode": "template-based"
	},
	"file_path": %q
}`,
		summaryText,
		policy.ID,
		policy.Name,
		task.ID,
		task.Name,
		structuredSummary.Summary,
		psgt.toJSONArray(structuredSummary.KeyRequirements),
		psgt.toJSONArray(structuredSummary.RelevantSections),
		task.ID,
		task.ReferenceID,
		policy.ID,
		outputFormat,
		time.Now().Format(time.RFC3339),
		filePath)

	// Create evidence source
	source := &models.EvidenceSource{
		Type:        "policy_summary",
		Resource:    fmt.Sprintf("Policy summary for %s in context of task %s", policy.ID, task.ReferenceID),
		Content:     summaryText,
		Relevance:   psgt.calculateRelevance(task, policy),
		ExtractedAt: time.Now(),
		Metadata: map[string]interface{}{
			"task_id":         task.ID,
			"task_reference":  task.ReferenceID,
			"policy_id":       policy.ID,
			"policy_name":     policy.Name,
			"output_format":   outputFormat,
			"summary_length":  len(summaryText),
			"file_path":       filePath,
			"generation_mode": "template-based",
		},
	}

	psgt.logger.Info("Successfully generated policy summary",
		logger.Field{Key: "task_id", Value: taskID},
		logger.Field{Key: "task_reference", Value: task.ReferenceID},
		logger.Field{Key: "policy_id", Value: policy.ID},
		logger.Field{Key: "summary_length", Value: len(summaryText)},
		logger.Field{Key: "file_path", Value: filePath},
		logger.Field{Key: "generation_mode", Value: "template-based"})

	return responseJSON, source, nil
}

// buildPolicySummaryContext builds template context for policy summary generation
func (psgt *PolicySummaryGeneratorTool) buildPolicySummaryContext(task *domain.EvidenceTask, policy *domain.Policy) *models.PolicySummaryContext {
	// Convert domain objects to models
	modelsTask := psgt.convertDomainTaskToModels(task)
	modelsPolicy := psgt.convertDomainPolicyToModels(policy)

	return &models.PolicySummaryContext{
		Task:   *modelsTask,
		Policy: *modelsPolicy,
	}
}

// generateTemplateSummary generates policy summary using the policy_summary template
func (psgt *PolicySummaryGeneratorTool) generateTemplateSummary(context *models.PolicySummaryContext) (string, error) {
	// Use the policy_summary template
	summaryText, err := psgt.templateManager.Execute("policy_summary", context)
	if err != nil {
		psgt.logger.Warn("Failed to execute policy_summary template, falling back to basic summary",
			logger.Field{Key: "error", Value: err})

		// Fallback to basic summary generation
		return psgt.generateBasicSummary(context), nil
	}

	return summaryText, nil
}

// generateBasicSummary generates a basic policy summary as fallback
func (psgt *PolicySummaryGeneratorTool) generateBasicSummary(context *models.PolicySummaryContext) string {
	var summary strings.Builder

	summary.WriteString(fmt.Sprintf("# Policy Summary: %s\n\n", context.Policy.Name))
	summary.WriteString(fmt.Sprintf("**Policy ID:** %s\n", context.Policy.ID))
	summary.WriteString(fmt.Sprintf("**Evidence Task:** %s (%s)\n", context.Task.Name, context.Task.ReferenceID))

	if context.Policy.Framework != "" {
		summary.WriteString(fmt.Sprintf("**Framework:** %s\n", context.Policy.Framework))
	}

	summary.WriteString("\n## Policy Relevance to Evidence Task\n")
	summary.WriteString(fmt.Sprintf("This policy governs aspects relevant to the evidence collection task: %s\n\n", context.Task.Description))

	if context.Policy.Description != "" {
		summary.WriteString("## Policy Content\n")
		summary.WriteString(context.Policy.Description)
		summary.WriteString("\n\n")
	}

	summary.WriteString("## Key Requirements\n")
	summary.WriteString("• Review policy compliance documentation\n")
	summary.WriteString("• Verify implementation of policy controls\n")
	summary.WriteString("• Collect evidence of policy enforcement\n\n")

	summary.WriteString("## Relevant Sections\n")
	summary.WriteString("• Implementation procedures\n")
	summary.WriteString("• Monitoring and review processes\n")
	summary.WriteString("• Compliance verification methods\n\n")

	summary.WriteString("---\n")
	summary.WriteString("*This summary was generated using template-based analysis to focus on evidence collection requirements.*\n")

	return summary.String()
}

// extractStructuredSummary extracts structured data from the generated summary text
func (psgt *PolicySummaryGeneratorTool) extractStructuredSummary(summaryText string, task *domain.EvidenceTask, policy *domain.Policy) models.AIPolicySummary {
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
		"Review policy compliance documentation",
		"Verify implementation of policy controls",
		"Collect evidence of policy enforcement",
	}

	// Default relevant sections
	relevantSections := []string{
		"Implementation procedures",
		"Monitoring and review processes",
		"Compliance verification methods",
	}

	return models.AIPolicySummary{
		PolicyID:         policy.ID,
		PolicyName:       policy.Name,
		Summary:          summary,
		KeyRequirements:  keyRequirements,
		RelevantSections: relevantSections,
	}
}

// saveSummaryToFile saves the generated summary to a file
func (psgt *PolicySummaryGeneratorTool) saveSummaryToFile(summaryText string, task *domain.EvidenceTask, policy *domain.Policy) (string, error) {
	// Create filename with task reference, policy ID, and sanitized names
	interpolatorConfig := interpolation.InterpolatorConfig{
		Variables:         make(map[string]string),
		Enabled:           false,
		OnMissingVariable: interpolation.MissingVariableIgnore,
	}
	interpolator := interpolation.NewStandardInterpolator(interpolatorConfig)
	baseFormatter := formatters.NewBaseFormatter(interpolator)

	// Sanitize names for filename
	sanitizedTaskName := baseFormatter.SanitizeFilename(task.Name)
	sanitizedPolicyName := baseFormatter.SanitizeFilename(policy.Name)

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("policy_summary_%s_%s_%s_%s_%s.md",
		task.ReferenceID,
		policy.ID,
		sanitizedTaskName,
		sanitizedPolicyName,
		timestamp)

	// Ensure summaries directory exists
	summaryDir := filepath.Join(psgt.config.Storage.DataDir, "summaries", "policies")
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

func (psgt *PolicySummaryGeneratorTool) parseTaskReference(taskRef string) (int, error) {
	validator := NewValidator(psgt.config.Storage.DataDir)

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

func (psgt *PolicySummaryGeneratorTool) calculateRelevance(task *domain.EvidenceTask, policy *domain.Policy) float64 {
	base := 0.7 // Base relevance for policy summary

	// Increase relevance if policy framework matches task framework
	if policy.Framework == task.Framework {
		base += 0.2
	}

	// Increase relevance based on policy content length (more content = potentially more relevant)
	if len(policy.Description) > 500 {
		base += 0.1
	}

	if base > 1.0 {
		base = 1.0
	}

	return base
}

func (psgt *PolicySummaryGeneratorTool) toJSONArray(items []string) string {
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

func (psgt *PolicySummaryGeneratorTool) Category() string {
	return "evidence-management"
}

func (psgt *PolicySummaryGeneratorTool) Version() string {
	return "1.0.0"
}

// convertDomainTaskToModels converts a domain task to models task
func (psgt *PolicySummaryGeneratorTool) convertDomainTaskToModels(task *domain.EvidenceTask) *models.EvidenceTaskDetails {
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

// convertDomainPolicyToModels converts a domain policy to models policy
func (psgt *PolicySummaryGeneratorTool) convertDomainPolicyToModels(policy *domain.Policy) *models.Policy {
	return &models.Policy{
		ID:          models.IntOrString(policy.ID),
		Name:        policy.Name,
		Description: policy.Description,
		Framework:   policy.Framework,
		Controls:    []models.Control{}, // Simplified - not loading related controls
	}
}
