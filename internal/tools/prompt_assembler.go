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

// PromptAssemblerTool provides evidence prompt generation capabilities
// using template-based approach (no AI API calls)
type PromptAssemblerTool struct {
	config          *config.Config
	logger          logger.Logger
	dataService     DataServiceInterface
	storage         *storage.Storage
	templateManager *templates.Manager
}

// NewPromptAssemblerTool creates a new PromptAssemblerTool
func NewPromptAssemblerTool(cfg *config.Config, log logger.Logger) Tool {
	// Initialize unified storage and data service
	storage, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		log.Error("Failed to initialize unified storage for prompt assembler tool",
			logger.Field{Key: "error", Value: err})
		return nil
	}

	// Get singleton template manager
	templateManager, err := templates.GetSingleton(log)
	if err != nil {
		log.Error("Failed to get singleton template manager for prompt assembler tool",
			logger.Field{Key: "error", Value: err})
		return nil
	}

	dataService := &SimpleDataService{storage: storage}

	return &PromptAssemblerTool{
		config:          cfg,
		logger:          log,
		dataService:     dataService,
		storage:         storage,
		templateManager: templateManager,
	}
}

// Name returns the tool name
func (pat *PromptAssemblerTool) Name() string {
	return "prompt-assembler"
}

// Description returns the tool description
func (pat *PromptAssemblerTool) Description() string {
	return "Generates comprehensive prompts for evidence collection with context and examples (template-based, no AI API calls)"
}

// GetClaudeToolDefinition returns the tool definition for external AI tools
func (pat *PromptAssemblerTool) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        pat.Name(),
		Description: pat.Description(),
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"task_ref": map[string]interface{}{
					"type":        "string",
					"description": "Task reference in format ET-101, ET101, or numeric ID (e.g., 328001)",
					"examples":    []string{"ET-101", "ET101", "328001"},
				},
				"context_level": map[string]interface{}{
					"type":        "string",
					"description": "Context level for prompt generation",
					"enum":        []string{"minimal", "standard", "comprehensive"},
					"default":     "standard",
				},
				"include_examples": map[string]interface{}{
					"type":        "boolean",
					"description": "Include example evidence and best practices in the prompt",
					"default":     true,
				},
				"output_format": map[string]interface{}{
					"type":        "string",
					"description": "Output format for generated evidence",
					"enum":        []string{"markdown", "csv", "json"},
					"default":     "markdown",
				},
				"save_to_file": map[string]interface{}{
					"type":        "boolean",
					"description": "Save the generated prompt to a file",
					"default":     true,
				},
			},
			"required": []string{"task_ref"},
		},
	}
}

// Execute generates comprehensive evidence prompts
func (pat *PromptAssemblerTool) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	pat.logger.Debug("Executing prompt assembler", logger.Field{Key: "params", Value: params})

	// Extract and validate parameters
	taskRefRaw, ok := params["task_ref"]
	if !ok {
		return "", nil, fmt.Errorf("task_ref parameter is required")
	}

	taskRef, ok := taskRefRaw.(string)
	if !ok {
		return "", nil, fmt.Errorf("task_ref must be a string")
	}

	// Parse additional parameters
	contextLevel := "standard"
	if cl, exists := params["context_level"]; exists {
		if level, ok := cl.(string); ok {
			contextLevel = level
		}
	}

	includeExamples := true
	if ie, exists := params["include_examples"]; exists {
		if flag, ok := ie.(bool); ok {
			includeExamples = flag
		}
	}

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
	taskID, err := pat.parseTaskReference(taskRef)
	if err != nil {
		return "", nil, fmt.Errorf("invalid task reference '%s': %w", taskRef, err)
	}

	// Get task details directly from data service
	task, err := pat.dataService.GetEvidenceTask(ctx, taskID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get evidence task: %w", err)
	}

	// Build basic evidence context for prompt generation
	evidenceContext, err := pat.buildBasicEvidenceContext(ctx, task, contextLevel, includeExamples)
	if err != nil {
		return "", nil, fmt.Errorf("failed to build evidence context: %w", err)
	}

	// Add tool information to the context
	evidenceContext.AvailableTools = pat.getAvailableTools()

	// Generate the prompt using template-based approach
	promptText := pat.generateTemplatePrompt(evidenceContext, outputFormat)

	// Save prompt to file if requested
	var filePath string
	if saveToFile {
		filePath, err = pat.savePromptToFile(promptText, task, outputFormat)
		if err != nil {
			pat.logger.Warn("Failed to save prompt to file",
				logger.Field{Key: "error", Value: err})
		}
	}

	// Generate JSON response from prompt and metadata
	responseJSON := fmt.Sprintf(`{
	"success": true,
	"prompt_text": %q,
	"prompt_metadata": {
		"task_id": %d,
		"reference_id": %q,
		"context_level": %q,
		"include_examples": %t,
		"output_format": %q,
		"framework_count": %d,
		"generated_at": %q,
		"generation_mode": "template-based"
	},
	"file_path": %q,
	"context_summary": {
		"task_name": %q,
		"framework": %q,
		"priority": %q,
		"status": %q
	}
}`, promptText, task.ID, task.ReferenceID, contextLevel, includeExamples, outputFormat, len(evidenceContext.FrameworkReqs), time.Now().Format(time.RFC3339), filePath, task.Name, task.Framework, task.Priority, task.Status)

	// Create evidence source
	source := &models.EvidenceSource{
		Type:        "evidence_prompt",
		Resource:    fmt.Sprintf("Prompt for task %s", task.ReferenceID),
		Content:     promptText,
		Relevance:   pat.calculateRelevance(evidenceContext, contextLevel),
		ExtractedAt: time.Now(),
		Metadata: map[string]interface{}{
			"task_id":         task.ID,
			"reference_id":    task.ReferenceID,
			"context_level":   contextLevel,
			"output_format":   outputFormat,
			"prompt_length":   len(promptText),
			"file_path":       filePath,
			"generation_mode": "template-based",
		},
	}

	pat.logger.Info("Successfully generated evidence prompt",
		logger.Field{Key: "task_id", Value: taskID},
		logger.Field{Key: "reference_id", Value: task.ReferenceID},
		logger.Field{Key: "context_level", Value: contextLevel},
		logger.Field{Key: "prompt_length", Value: len(promptText)},
		logger.Field{Key: "file_path", Value: filePath},
		logger.Field{Key: "generation_mode", Value: "template-based"})

	// Return the JSON response as the main response
	return responseJSON, source, nil
}

// buildBasicEvidenceContext builds basic evidence context for template-based prompt generation
func (pat *PromptAssemblerTool) buildBasicEvidenceContext(ctx context.Context, task *domain.EvidenceTask, contextLevel string, includeExamples bool) (*models.EvidenceContext, error) {
	// Convert domain task to models task
	modelsTask := pat.convertDomainTaskToModels(task)

	evidenceContext := &models.EvidenceContext{
		Task:             *modelsTask,
		Controls:         []models.Control{}, // Simplified - not loading related controls
		Policies:         []models.Policy{},  // Simplified - not loading related policies
		ControlSummaries: make(map[int]models.AIControlSummary),
		PolicySummaries:  make(map[string]models.AIPolicySummary),
		FrameworkReqs:    pat.extractBasicFrameworkRequirements(task),
		PreviousEvidence: []string{}, // Could be enhanced to include past submissions
		SecurityMappings: pat.getSecurityMappings(),
	}

	// Adjust context based on level
	switch contextLevel {
	case "minimal":
		// Keep only essential information
		evidenceContext.FrameworkReqs = evidenceContext.FrameworkReqs[:min(len(evidenceContext.FrameworkReqs), 2)]
	case "comprehensive":
		// Include all available context
		if includeExamples {
			evidenceContext.PreviousEvidence = pat.generateExampleEvidence(task.Framework)
		}
	}

	return evidenceContext, nil
}

// getAvailableTools retrieves information about available tools from the registry
func (pat *PromptAssemblerTool) getAvailableTools() []ToolInfo {
	return ListTools()
}

// generateTemplatePrompt generates a comprehensive template-based prompt using the template manager
func (pat *PromptAssemblerTool) generateTemplatePrompt(context *models.EvidenceContext, outputFormat string) string {
	// Set the output format in the context for template use
	context.OutputFormat = outputFormat

	// Try to use the comprehensive evidence generation template
	promptText, err := pat.templateManager.Execute("evidence_generation", context)
	if err != nil {
		pat.logger.Warn("Failed to execute evidence generation template, falling back to basic template",
			logger.Field{Key: "error", Value: err})

		// Fallback to basic template
		return pat.generateBasicPrompt(context, outputFormat)
	}

	return promptText
}

// generateBasicPrompt generates a basic prompt as fallback
func (pat *PromptAssemblerTool) generateBasicPrompt(context *models.EvidenceContext, outputFormat string) string {
	var prompt strings.Builder

	// Header section
	prompt.WriteString(fmt.Sprintf("# Evidence Collection Task: %s\n\n", context.Task.Name))
	prompt.WriteString(fmt.Sprintf("**Task ID:** %s\n", context.Task.ReferenceID))
	prompt.WriteString(fmt.Sprintf("**Framework:** %s\n", context.Task.Framework))
	prompt.WriteString(fmt.Sprintf("**Priority:** %s\n", context.Task.Priority))
	prompt.WriteString(fmt.Sprintf("**Status:** %s\n", context.Task.Status))
	prompt.WriteString(fmt.Sprintf("**Output Format:** %s\n\n", outputFormat))

	// Task description
	prompt.WriteString("## Task Description\n")
	prompt.WriteString(context.Task.Description)
	prompt.WriteString("\n\n")

	// Guidance if available
	if context.Task.Guidance != "" {
		prompt.WriteString("## Collection Guidance\n")
		prompt.WriteString(context.Task.Guidance)
		prompt.WriteString("\n\n")
	}

	// Available Tools
	if len(context.AvailableTools) > 0 {
		prompt.WriteString("## Available Evidence Collection Tools\n\n")
		prompt.WriteString("Use these tools to gather evidence systematically:\n\n")
		for _, tool := range context.AvailableTools {
			prompt.WriteString(fmt.Sprintf("### `%s`\n", tool.Name))
			prompt.WriteString(fmt.Sprintf("- **Description**: %s\n", tool.Description))
			if tool.Category != "" {
				prompt.WriteString(fmt.Sprintf("- **Category**: %s\n", tool.Category))
			}
			if tool.Version != "" {
				prompt.WriteString(fmt.Sprintf("- **Version**: %s\n", tool.Version))
			}
			prompt.WriteString("\n")
		}
	}

	// Framework requirements
	if len(context.FrameworkReqs) > 0 {
		prompt.WriteString("## Framework Requirements\n")
		for _, req := range context.FrameworkReqs {
			prompt.WriteString(fmt.Sprintf("- %s\n", req))
		}
		prompt.WriteString("\n")
	}

	// Evidence collection instructions
	prompt.WriteString("## Evidence Collection Instructions\n")
	prompt.WriteString("Please provide comprehensive evidence that demonstrates compliance with the above requirements.\n\n")

	switch strings.ToLower(outputFormat) {
	case "csv":
		prompt.WriteString("### CSV Format Requirements\n")
		prompt.WriteString("Structure the evidence as a CSV with the following columns:\n")
		prompt.WriteString("- Control/Requirement\n")
		prompt.WriteString("- Evidence Type\n")
		prompt.WriteString("- Implementation Details\n")
		prompt.WriteString("- Verification Method\n")
		prompt.WriteString("- Status\n")
		prompt.WriteString("- Notes\n\n")
	case "json":
		prompt.WriteString("### JSON Format Requirements\n")
		prompt.WriteString("Structure the evidence as a JSON object with clear hierarchy and metadata.\n\n")
	default: // markdown
		prompt.WriteString("### Markdown Format Requirements\n")
		prompt.WriteString("Structure the evidence with clear headings and sections:\n")
		prompt.WriteString("1. **Executive Summary** - Brief overview of compliance status\n")
		prompt.WriteString("2. **Implementation Evidence** - Specific configurations and controls\n")
		prompt.WriteString("3. **Monitoring Evidence** - Ongoing oversight and maintenance\n")
		prompt.WriteString("4. **Supporting Artifacts** - References to logs, screenshots, documentation\n\n")
	}

	// Additional guidelines
	prompt.WriteString("## Evidence Collection Guidelines\n")
	prompt.WriteString("- Include specific configuration details and settings\n")
	prompt.WriteString("- Reference relevant policies, procedures, and documentation\n")
	prompt.WriteString("- Provide timestamps and version information where applicable\n")
	prompt.WriteString("- Include screenshots or exports of key configurations\n")
	prompt.WriteString("- Demonstrate both implementation and ongoing monitoring\n")
	prompt.WriteString("- Use the available tools systematically to gather comprehensive evidence\n")

	// Example evidence if available
	if len(context.PreviousEvidence) > 0 {
		prompt.WriteString("\n## Example Evidence Types\n")
		for _, example := range context.PreviousEvidence {
			prompt.WriteString(fmt.Sprintf("- %s\n", example))
		}
	}

	prompt.WriteString("\n---\n")
	prompt.WriteString("*This prompt was generated using template-based assembly. Please collect evidence systematically and thoroughly.*\n")

	return prompt.String()
}

// savePromptToFile saves the generated prompt to a file
func (pat *PromptAssemblerTool) savePromptToFile(promptText string, task *domain.EvidenceTask, outputFormat string) (string, error) {
	// Create filename with reference ID, task ID, and sanitized name
	interpolatorConfig := interpolation.InterpolatorConfig{
		Variables:         make(map[string]string),
		Enabled:           false,
		OnMissingVariable: interpolation.MissingVariableIgnore,
	}
	interpolator := interpolation.NewStandardInterpolator(interpolatorConfig)
	baseFormatter := formatters.NewBaseFormatter(interpolator)

	// Sanitize task name for filename
	sanitizedName := baseFormatter.SanitizeFilename(task.Name)

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("prompt_%s_%d_%s_%s.md",
		task.ReferenceID,
		task.ID,
		sanitizedName,
		timestamp)

	// Ensure prompt directory exists
	promptDir := filepath.Join(pat.config.Storage.DataDir, "prompts")
	if err := os.MkdirAll(promptDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create prompt directory: %w", err)
	}

	// Save prompt to file
	filePath := filepath.Join(promptDir, filename)
	if err := os.WriteFile(filePath, []byte(promptText), 0644); err != nil {
		return "", fmt.Errorf("failed to save prompt: %w", err)
	}

	return filePath, nil
}

// Helper methods for conversion and data processing

func (pat *PromptAssemblerTool) extractBasicFrameworkRequirements(task *domain.EvidenceTask) []string {
	var reqs []string

	if task.Framework != "" {
		reqs = append(reqs, task.Framework)

		// Add common framework-specific requirements
		switch strings.ToUpper(task.Framework) {
		case "SOC2":
			reqs = append(reqs, "Security controls must be documented and implemented")
			reqs = append(reqs, "Evidence of ongoing monitoring and review")
			reqs = append(reqs, "Incident response procedures documented")
		case "ISO27001":
			reqs = append(reqs, "Information security management system (ISMS) in place")
			reqs = append(reqs, "Risk assessment and treatment documented")
			reqs = append(reqs, "Security policies and procedures established")
		case "NIST":
			reqs = append(reqs, "NIST Cybersecurity Framework controls implemented")
			reqs = append(reqs, "Continuous monitoring and assessment")
		default:
			reqs = append(reqs, "Compliance with applicable security standards")
			reqs = append(reqs, "Documentation of security controls")
		}
	}

	return reqs
}

func (pat *PromptAssemblerTool) getSecurityMappings() models.SecurityMappings {
	mappings := make(map[string]models.SecurityControlMapping)
	for k, v := range pat.config.Evidence.SecurityControls.SOC2 {
		mappings[k] = models.SecurityControlMapping{
			TerraformResources: v.TerraformResources,
			Description:        v.Description,
			Requirements:       v.Requirements,
		}
	}
	return models.SecurityMappings{
		SOC2: mappings,
	}
}

func (pat *PromptAssemblerTool) generateExampleEvidence(framework string) []string {
	examples := map[string][]string{
		"SOC2": {
			"AWS IAM role configurations with appropriate permissions",
			"CloudTrail logs showing access monitoring",
			"Security group rules demonstrating network controls",
		},
		"ISO27001": {
			"Information security policy documentation",
			"Risk assessment and treatment records",
			"Security incident response procedures",
		},
	}

	if frameworkExamples, exists := examples[framework]; exists {
		return frameworkExamples
	}

	return []string{
		"Configuration screenshots or exports",
		"Policy and procedure documentation",
		"Audit logs and monitoring evidence",
	}
}

func (pat *PromptAssemblerTool) calculateRelevance(context *models.EvidenceContext, contextLevel string) float64 {
	base := 0.8 // High base relevance for prompt generation

	// Adjust based on context level
	switch contextLevel {
	case "comprehensive":
		base += 0.15
	case "standard":
		base += 0.05
	case "minimal":
		// No adjustment
	}

	// Adjust based on available context
	if len(context.Controls) > 0 {
		base += 0.05
	}
	if len(context.Policies) > 0 {
		base += 0.05
	}

	if base > 1.0 {
		base = 1.0
	}

	return base
}

func (pat *PromptAssemblerTool) parseTaskReference(taskRef string) (int, error) {
	validator := NewValidator(pat.config.Storage.DataDir)

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

func (pat *PromptAssemblerTool) Category() string {
	return "evidence"
}

func (pat *PromptAssemblerTool) Version() string {
	return "1.0.0"
}

// convertDomainTaskToModels converts a domain task to models task
func (pat *PromptAssemblerTool) convertDomainTaskToModels(task *domain.EvidenceTask) *models.EvidenceTaskDetails {
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
