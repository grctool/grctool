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

package evidence

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
	"github.com/grctool/grctool/internal/markdown"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/services"
	"github.com/grctool/grctool/internal/tugboat"
)

// ServiceImpl implements the evidence Service interface
type ServiceImpl struct {
	dataService     services.DataService
	evidenceService *services.EvidenceServiceSimple
	documentService *services.DocumentService
	config          *config.Config
	logger          logger.Logger
}

// NewService creates a new evidence service implementation
func NewService(dataService services.DataService, cfg *config.Config, log logger.Logger) (Service, error) {
	evidenceService, err := services.NewEvidenceService(dataService, cfg, log)
	if err != nil {
		return nil, fmt.Errorf("failed to create evidence service: %w", err)
	}

	documentService := services.NewDocumentService(cfg)

	return &ServiceImpl{
		dataService:     dataService,
		evidenceService: evidenceService,
		documentService: documentService,
		config:          cfg,
		logger:          log,
	}, nil
}

// ListEvidenceTasks returns filtered evidence tasks with enhanced filtering
func (s *ServiceImpl) ListEvidenceTasks(ctx context.Context, filter domain.EvidenceFilter) ([]domain.EvidenceTask, error) {
	tasks, err := s.evidenceService.ListEvidenceTasks(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Enrich tasks with Tugboat web URLs
	s.enrichTasksWithURLs(tasks)

	return tasks, nil
}

// GetEvidenceTaskSummary returns a summary of evidence tasks
func (s *ServiceImpl) GetEvidenceTaskSummary(ctx context.Context) (*domain.EvidenceTaskSummary, error) {
	return s.evidenceService.GetEvidenceTaskSummary(ctx)
}

// ResolveTaskID converts a task identifier (numeric ID or reference ID) to a numeric task ID
func (s *ServiceImpl) ResolveTaskID(ctx context.Context, identifier string) (int, error) {
	// First try to parse as integer
	taskID, err := strconv.Atoi(identifier)
	if err == nil {
		return taskID, nil
	}

	// Try to parse as reference ID (e.g., "ET1", "ET42")
	if strings.HasPrefix(strings.ToUpper(identifier), "ET") {
		// Get all tasks to find by reference ID
		tasks, err := s.dataService.GetAllEvidenceTasks(ctx)
		if err != nil {
			return 0, fmt.Errorf("failed to get evidence tasks: %w", err)
		}

		// Search for matching reference ID
		upperIdentifier := strings.ToUpper(identifier)
		for _, task := range tasks {
			if task.ReferenceID == upperIdentifier {
				return task.ID, nil
			}
		}
		return 0, fmt.Errorf("evidence task with reference ID %s not found", identifier)
	}

	return 0, fmt.Errorf("invalid task identifier: %s (must be numeric ID or reference ID like ET1)", identifier)
}

// AnalyzeEvidenceTask analyzes an evidence task and generates prompts
func (s *ServiceImpl) AnalyzeEvidenceTask(ctx context.Context, taskID int) (*services.EvidenceAnalysisResult, error) {
	return s.evidenceService.AnalyzeEvidenceTask(ctx, taskID)
}

// GenerateTemplateBasedPrompt generates a template-based prompt without AI API calls
func (s *ServiceImpl) GenerateTemplateBasedPrompt(context *models.EvidenceContext, outputFormat string) string {
	var prompt strings.Builder

	// Header section
	prompt.WriteString(fmt.Sprintf("# Evidence Collection Task: %s\n\n", context.Task.Name))
	prompt.WriteString(fmt.Sprintf("**Task ID:** %s\n", context.Task.ReferenceID))
	prompt.WriteString(fmt.Sprintf("**Framework:** %s\n", context.Task.GetFramework()))
	prompt.WriteString(fmt.Sprintf("**Priority:** %s\n", context.Task.GetPriority()))
	prompt.WriteString(fmt.Sprintf("**Status:** %s\n", context.Task.GetStatus()))
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

	// Framework requirements
	if len(context.FrameworkReqs) > 0 {
		prompt.WriteString("## Framework Requirements\n")
		for _, req := range context.FrameworkReqs {
			prompt.WriteString(fmt.Sprintf("- %s\n", req))
		}
		prompt.WriteString("\n")
	}

	// Related controls
	if len(context.Controls) > 0 {
		prompt.WriteString("## Related Controls\n")
		for _, control := range context.Controls {
			prompt.WriteString(fmt.Sprintf("### %s\n", control.Name))
			prompt.WriteString(fmt.Sprintf("**Category:** %s\n", control.Category))
			prompt.WriteString(fmt.Sprintf("**Status:** %s\n", control.Status))
			if control.Body != "" {
				prompt.WriteString(fmt.Sprintf("**Description:** %s\n", control.Body))
			}
			prompt.WriteString("\n")
		}
	}

	// Related policies
	if len(context.Policies) > 0 {
		prompt.WriteString("## Related Policies\n")
		for _, policy := range context.Policies {
			prompt.WriteString(fmt.Sprintf("### %s\n", policy.Name))
			prompt.WriteString(fmt.Sprintf("**Framework:** %s\n", policy.Framework))
			prompt.WriteString(fmt.Sprintf("**Status:** %s\n", policy.Status))
			if policy.Description != "" {
				prompt.WriteString(fmt.Sprintf("**Description:** %s\n", policy.Description))
			}
			prompt.WriteString("\n")
		}
	}

	// Suggested evidence collection tools
	if len(context.AvailableTools) > 0 {
		prompt.WriteString("## Suggested Evidence Collection Tools\n\n")
		prompt.WriteString("Use these tools to gather comprehensive evidence systematically:\n\n")
		for _, tool := range context.AvailableTools {
			prompt.WriteString(fmt.Sprintf("- **`%s`** - Use `grctool tool %s --help` for usage details\n", tool.Name, tool.Name))
		}
		prompt.WriteString("\n### Tool Usage Examples\n\n")

		// Add specific examples for common tools
		for _, tool := range context.AvailableTools {
			switch tool.Name {
			case "control-summary-generator":
				if len(context.Controls) > 0 {
					prompt.WriteString(fmt.Sprintf("```bash\n# Generate AI summary for control %d\n", context.Controls[0].ID))
					prompt.WriteString(fmt.Sprintf("grctool tool control-summary-generator --task-ref %s --control-id %d\n```\n\n", context.Task.ReferenceID, context.Controls[0].ID))
				}
			case "policy-summary-generator":
				if len(context.Policies) > 0 {
					prompt.WriteString(fmt.Sprintf("```bash\n# Generate AI summary for policy\n"))
					prompt.WriteString(fmt.Sprintf("grctool tool policy-summary-generator --task-ref %s --policy-id %s\n```\n\n", context.Task.ReferenceID, context.Policies[0].ID))
				}
			case "terraform-security-analyzer":
				prompt.WriteString("```bash\n# Scan Terraform configurations for security controls\n")
				prompt.WriteString("grctool tool terraform-security-analyzer --analysis-type security_controls\n```\n\n")
			case "github-permissions":
				prompt.WriteString("```bash\n# Analyze GitHub repository permissions\n")
				prompt.WriteString("grctool tool github-permissions --repository owner/repo\n```\n\n")
			}
		}
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

	prompt.WriteString("\n---\n")
	prompt.WriteString("*This prompt was generated using template-based assembly. Please collect evidence systematically and thoroughly.*\n")

	return prompt.String()
}

// ProcessAnalysisForTask processes an evidence task and generates a prompt
func (s *ServiceImpl) ProcessAnalysisForTask(ctx context.Context, taskID int, outputFormat string) (string, string, error) {
	// Get task details
	task, err := s.dataService.GetEvidenceTask(ctx, taskID)
	if err != nil {
		return "", "", fmt.Errorf("failed to get task: %w", err)
	}

	s.logger.Info("Generating Claude AI prompt for evidence task",
		logger.Field{Key: "task_id", Value: taskID},
		logger.Field{Key: "task_name", Value: task.Name},
		logger.Field{Key: "reference_id", Value: task.ReferenceID},
	)

	// Analyze the task to get relationships
	analysis, err := s.evidenceService.AnalyzeEvidenceTask(ctx, taskID)
	if err != nil {
		return "", "", fmt.Errorf("failed to analyze task: %w", err)
	}

	s.logger.Debug("Evidence task analysis completed",
		logger.Field{Key: "task_id", Value: taskID},
		logger.Field{Key: "related_controls", Value: len(analysis.RelatedControls)},
		logger.Field{Key: "related_policies", Value: len(analysis.RelatedPolicies)},
	)

	// Build basic evidence context for template-based prompt
	evidenceContext := &models.EvidenceContext{
		Task:             *s.convertToModelsTask(analysis.Task),
		Controls:         s.convertToModelsControls(analysis.RelatedControls),
		Policies:         s.convertToModelsPolicies(analysis.RelatedPolicies),
		ControlSummaries: make(map[int]models.AIControlSummary),
		PolicySummaries:  make(map[string]models.AIPolicySummary),
		FrameworkReqs:    []string{}, // Will be populated from controls
		PreviousEvidence: []string{}, // Could be enhanced to include past submissions
		SecurityMappings: models.SecurityMappings{SOC2: make(map[string]models.SecurityControlMapping)},
		AvailableTools:   []models.ToolInfo{}, // Will be populated from analysis
	}

	// Extract framework requirements from controls
	frameworkReqs := make(map[string]bool)
	for _, control := range analysis.RelatedControls {
		for _, fc := range control.FrameworkCodes {
			frameworkReqs[fc.Code] = true
		}
	}
	for req := range frameworkReqs {
		evidenceContext.FrameworkReqs = append(evidenceContext.FrameworkReqs, req)
	}

	// Populate available tools from analysis suggested tools
	for _, toolName := range analysis.SuggestedTools {
		evidenceContext.AvailableTools = append(evidenceContext.AvailableTools, models.ToolInfo{
			Name:    toolName,
			Enabled: true,
		})
	}

	// Generate template-based prompt (no AI API calls)
	s.logger.Info("Creating template-based evidence prompt",
		logger.Field{Key: "task_id", Value: taskID},
		logger.Field{Key: "output_format", Value: outputFormat},
		logger.Field{Key: "generation_mode", Value: "template-based"},
	)

	promptText := s.GenerateTemplateBasedPrompt(evidenceContext, outputFormat)

	// Generate filename with reference ID, task ID, and sanitized name
	interpolatorConfig := interpolation.InterpolatorConfig{
		Variables:         make(map[string]string),
		Enabled:           false,
		OnMissingVariable: interpolation.MissingVariableIgnore,
	}
	interpolator := interpolation.NewStandardInterpolator(interpolatorConfig)
	baseFormatter := formatters.NewBaseFormatter(interpolator)

	// Sanitize task name using the unified method
	sanitizedName := baseFormatter.SanitizeFilename(task.Name)

	filename := fmt.Sprintf("%s_%d_%s.md",
		task.ReferenceID,
		taskID,
		sanitizedName)

	// Save prompt to evidence_prompts folder
	if err := s.documentService.GenerateDocument(services.EvidencePromptDocument, filename, promptText); err != nil {
		return "", "", fmt.Errorf("failed to save prompt: %w", err)
	}

	promptPath := s.documentService.GetDocumentPath(services.EvidencePromptDocument, filename)
	s.logger.Info("Claude AI prompt saved",
		logger.Field{Key: "task_id", Value: taskID},
		logger.Field{Key: "file_path", Value: promptPath},
		logger.Field{Key: "prompt_size", Value: len(promptText)},
	)

	return promptPath, promptText, nil
}

// ProcessBulkAnalysis processes multiple evidence tasks for bulk analysis
func (s *ServiceImpl) ProcessBulkAnalysis(ctx context.Context, outputFormat string) error {
	// Get all evidence tasks
	tasks, err := s.dataService.GetAllEvidenceTasks(ctx)
	if err != nil {
		return fmt.Errorf("failed to get all evidence tasks: %w", err)
	}

	if len(tasks) == 0 {
		return fmt.Errorf("no evidence tasks found")
	}

	// Process each task
	for _, task := range tasks {
		_, _, err := s.ProcessAnalysisForTask(ctx, task.ID, outputFormat)
		if err != nil {
			s.logger.Error("Failed to process task analysis",
				logger.Field{Key: "task_id", Value: task.ID},
				logger.Field{Key: "error", Value: err.Error()})
			continue
		}
	}

	return nil
}

// MapEvidenceRelationships creates a visual map showing relationships between tasks, controls, and policies
func (s *ServiceImpl) MapEvidenceRelationships(ctx context.Context) (*EvidenceMapResult, error) {
	// Get all evidence tasks
	tasks, err := s.evidenceService.ListEvidenceTasks(ctx, domain.EvidenceFilter{})
	if err != nil {
		return nil, fmt.Errorf("failed to get evidence tasks: %w", err)
	}

	if len(tasks) == 0 {
		return &EvidenceMapResult{
			Tasks:           []domain.EvidenceTask{},
			Controls:        []domain.Control{},
			Policies:        []domain.Policy{},
			FrameworkGroups: make(map[string][]domain.EvidenceTask),
			Summary: &EvidenceMapSummary{
				FrameworkCounts: make(map[string]int),
				StatusCounts:    make(map[string]int),
				PriorityCounts:  make(map[string]int),
			},
		}, nil
	}

	// Get all controls and policies for context
	controls, err := s.dataService.GetAllControls(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get controls: %w", err)
	}

	policies, err := s.dataService.GetAllPolicies(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get policies: %w", err)
	}

	// Group tasks by framework
	frameworkGroups := make(map[string][]domain.EvidenceTask)
	for _, task := range tasks {
		frameworkGroups[task.Framework] = append(frameworkGroups[task.Framework], task)
	}

	// Calculate summary statistics
	summary := s.calculateMapSummary(tasks, controls, policies, frameworkGroups)

	// Calculate total relationships
	totalRelationships := 0
	for _, task := range tasks {
		relationships, err := s.dataService.GetRelationships(ctx, "evidence_task", strconv.Itoa(task.ID))
		if err == nil {
			totalRelationships += len(relationships)
		}
	}

	summary.AverageRelationships = float64(totalRelationships) / float64(len(tasks))

	return &EvidenceMapResult{
		Tasks:              tasks,
		Controls:           controls,
		Policies:           policies,
		FrameworkGroups:    frameworkGroups,
		TotalRelationships: totalRelationships,
		Summary:            summary,
	}, nil
}

// calculateMapSummary calculates summary statistics for evidence mapping
func (s *ServiceImpl) calculateMapSummary(tasks []domain.EvidenceTask, controls []domain.Control, policies []domain.Policy, frameworkGroups map[string][]domain.EvidenceTask) *EvidenceMapSummary {
	summary := &EvidenceMapSummary{
		TotalTasks:      len(tasks),
		TotalControls:   len(controls),
		TotalPolicies:   len(policies),
		FrameworkCounts: make(map[string]int),
		StatusCounts:    make(map[string]int),
		PriorityCounts:  make(map[string]int),
	}

	now := time.Now()
	for _, task := range tasks {
		// Count by framework
		summary.FrameworkCounts[task.Framework]++

		// Count by status
		summary.StatusCounts[task.Status]++

		// Count by priority
		summary.PriorityCounts[task.Priority]++

		// Check if overdue
		if task.NextDue != nil && task.NextDue.Before(now) {
			summary.OverdueCount++
		}
	}

	return summary
}

// GenerateEvidence coordinates evidence generation using template-based tools
func (s *ServiceImpl) GenerateEvidence(ctx context.Context, req *services.EvidenceGenerationRequest) (*services.EvidenceGenerationResult, error) {
	return s.evidenceService.GenerateEvidence(ctx, req)
}

// ReviewEvidence reviews generated evidence
func (s *ServiceImpl) ReviewEvidence(ctx context.Context, recordID string, showReasoning bool) (map[string]interface{}, error) {
	return s.evidenceService.ReviewEvidence(ctx, recordID, showReasoning)
}

// SaveAnalysisToFile saves the analysis content to a markdown file
func (s *ServiceImpl) SaveAnalysisToFile(filename, content string) error {
	// Ensure the filename has .md extension
	if !strings.HasSuffix(filename, ".md") {
		filename += ".md"
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(filename)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Format the content before saving
	mdFormatter := markdown.NewFormatter(markdown.DefaultConfig())
	formattedContent := mdFormatter.FormatDocument(content)

	// Write content to file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = file.WriteString(formattedContent)
	if err != nil {
		return fmt.Errorf("failed to write content: %w", err)
	}

	return nil
}

// SaveEvidenceToFile saves evidence record to a file in the specified directory
func (s *ServiceImpl) SaveEvidenceToFile(outputDir string, record *domain.EvidenceRecord) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create filename based on record ID and format
	var filename string
	switch record.Format {
	case "csv":
		filename = fmt.Sprintf("evidence_%s.csv", record.ID)
	case "markdown":
		filename = fmt.Sprintf("evidence_%s.md", record.ID)
	default:
		filename = fmt.Sprintf("evidence_%s.%s", record.ID, record.Format)
	}

	filepath := fmt.Sprintf("%s/%s", outputDir, filename)

	// Write content to file
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = file.WriteString(record.Content)
	if err != nil {
		return fmt.Errorf("failed to write content: %w", err)
	}

	return nil
}

// Conversion functions for domain models to API models

func (s *ServiceImpl) convertToModelsTask(task *domain.EvidenceTask) *models.EvidenceTaskDetails {
	var lastCollectedStr *string
	if task.LastCollected != nil {
		str := task.LastCollected.Format(time.RFC3339)
		lastCollectedStr = &str
	}

	return &models.EvidenceTaskDetails{
		EvidenceTask: models.EvidenceTask{
			ID:                 task.ID,
			ReferenceID:        task.ReferenceID,
			Name:               task.Name,
			Description:        task.Description,
			Guidance:           task.Guidance,
			CollectionInterval: task.CollectionInterval,
			Priority:           task.Priority,
			Status:             task.Status,
			Framework:          task.Framework,
			Completed:          task.Completed,
			LastCollected:      lastCollectedStr,
			DueDaysBefore:      task.DueDaysBefore,
			AdHoc:              task.AdHoc,
			Sensitive:          task.Sensitive,
			CreatedAt:          task.CreatedAt,
			UpdatedAt:          task.UpdatedAt,
		},
	}
}

func (s *ServiceImpl) convertToModelsControls(controls []domain.Control) []models.Control {
	var result []models.Control
	for _, control := range controls {
		var frameworkCodes []models.FrameworkCode
		for _, fc := range control.FrameworkCodes {
			frameworkCodes = append(frameworkCodes, models.FrameworkCode{
				Code:          fc.Code,
				FrameworkName: fc.Framework,
			})
		}

		modelControl := models.Control{
			ID:                control.ID,
			Name:              control.Name,
			Body:              control.Description, // Description -> Body
			Category:          control.Category,
			Status:            control.Status,
			ImplementedDate:   control.ImplementedDate,
			IsAutoImplemented: control.IsAutoImplemented,
			FrameworkCodes:    frameworkCodes,
		}

		// Handle OrgScope if present
		if control.OrgScope != nil {
			modelControl.OrgScope = &models.OrgScope{
				ID:          control.OrgScope.ID,
				Name:        control.OrgScope.Name,
				ScopeType:   control.OrgScope.Type, // Type -> ScopeType
				Description: control.OrgScope.Description,
			}
		}

		result = append(result, modelControl)
	}
	return result
}

func (s *ServiceImpl) convertToModelsPolicies(policies []domain.Policy) []models.Policy {
	var result []models.Policy
	for _, policy := range policies {
		// Convert domain.Policy to models.Policy
		modelPolicy := models.Policy{
			ID:          models.IntOrString(policy.ID),
			Name:        policy.Name,
			Description: policy.Description,
			Status:      policy.Status,
			Framework:   policy.Framework,
		}

		// Controls in models.Policy is a slice of Control, not PolicyControl
		// For now, we'll leave it empty as we don't have full control data here
		modelPolicy.Controls = []models.Control{}

		result = append(result, modelPolicy)
	}
	return result
}

// enrichTasksWithURLs adds Tugboat web UI URLs to each task
func (s *ServiceImpl) enrichTasksWithURLs(tasks []domain.EvidenceTask) {
	if s.config.Tugboat.BaseURL == "" {
		// If no base URL configured, skip URL generation
		return
	}

	for i := range tasks {
		if tasks[i].ID > 0 {
			// Use task's org ID if available, otherwise use config org ID
			orgID := tasks[i].OrgID
			if orgID == 0 {
				// Parse org ID from config string to int
				// Config stores it as string, we need int for URL builder
				fmt.Sscanf(s.config.Tugboat.OrgID, "%d", &orgID)
			}

			if orgID > 0 {
				tasks[i].TugboatURL = tugboat.BuildEvidenceTaskURL(s.config.Tugboat.BaseURL, orgID, tasks[i].ID)
			}
		}
	}
}
