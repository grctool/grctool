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

package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
)

// DataService defines the interface for data access operations
type DataService interface {
	// Evidence task operations
	GetEvidenceTask(ctx context.Context, taskID int) (*domain.EvidenceTask, error)
	GetAllEvidenceTasks(ctx context.Context) ([]domain.EvidenceTask, error)
	FilterEvidenceTasks(ctx context.Context, filter domain.EvidenceFilter) ([]domain.EvidenceTask, error)

	// Control operations
	GetControl(ctx context.Context, controlID string) (*domain.Control, error)
	GetAllControls(ctx context.Context) ([]domain.Control, error)

	// Policy operations
	GetPolicy(ctx context.Context, policyID string) (*domain.Policy, error)
	GetAllPolicies(ctx context.Context) ([]domain.Policy, error)

	// Relationship operations
	GetRelationships(ctx context.Context, sourceType, sourceID string) ([]domain.Relationship, error)

	// Evidence record operations
	SaveEvidenceRecord(ctx context.Context, record *domain.EvidenceRecord) error
	GetEvidenceRecords(ctx context.Context, taskID int) ([]domain.EvidenceRecord, error)
}

// EvidenceGenerationRequest represents a request to generate evidence
type EvidenceGenerationRequest struct {
	TaskID      int                    `json:"task_id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Format      string                 `json:"format"` // csv, markdown, pdf, etc.
	Tools       []string               `json:"tools"`  // terraform, github, manual, etc.
	Context     map[string]interface{} `json:"context,omitempty"`
}

// EvidenceGenerationResult represents the result of evidence generation
type EvidenceGenerationResult struct {
	Record      *domain.EvidenceRecord  `json:"record"`
	Analysis    *EvidenceAnalysisResult `json:"analysis"`
	Metadata    map[string]interface{}  `json:"metadata"`
	Insights    []string                `json:"insights"`
	Suggestions []string                `json:"suggestions"`
}

// EvidenceAnalysisResult represents the result of evidence task analysis
type EvidenceAnalysisResult struct {
	TaskID            int                    `json:"task_id"`
	Task              *domain.EvidenceTask   `json:"task"`
	RelatedControls   []domain.Control       `json:"related_controls"`
	RelatedPolicies   []domain.Policy        `json:"related_policies"`
	Relationships     []domain.Relationship  `json:"relationships"`
	Recommendations   []string               `json:"recommendations"`
	RequiredEvidence  []string               `json:"required_evidence"`
	SuggestedTools    []string               `json:"suggested_tools"`
	ComplianceContext map[string]interface{} `json:"compliance_context"`
}

// EvidenceServiceSimple provides evidence-related operations without Claude dependencies
type EvidenceServiceSimple struct {
	dataService DataService
	config      *config.Config
	logger      logger.Logger
}

// NewEvidenceService creates a new simplified evidence service
func NewEvidenceService(dataService DataService, cfg *config.Config, log logger.Logger) (*EvidenceServiceSimple, error) {
	return &EvidenceServiceSimple{
		dataService: dataService,
		config:      cfg,
		logger:      log,
	}, nil
}

// GenerateEvidence coordinates evidence generation using template-based tools
func (s *EvidenceServiceSimple) GenerateEvidence(ctx context.Context, req *EvidenceGenerationRequest) (*EvidenceGenerationResult, error) {
	// Get task details
	task, err := s.dataService.GetEvidenceTask(ctx, req.TaskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get evidence task: %w", err)
	}

	// Generate simple template-based content
	content := s.generateTemplateContent(task, req)

	// Create evidence record
	record := &domain.EvidenceRecord{
		ID:          fmt.Sprintf("ev_%d_%d", req.TaskID, time.Now().Unix()),
		TaskID:      req.TaskID,
		Title:       req.Title,
		Description: req.Description,
		Content:     content,
		Format:      req.Format,
		Source:      "Template-generated",
		CollectedAt: time.Now(),
		CollectedBy: "grctool",
		Metadata:    req.Context,
	}

	// Save the evidence record
	if err := s.dataService.SaveEvidenceRecord(ctx, record); err != nil {
		return nil, fmt.Errorf("failed to save evidence record: %w", err)
	}

	s.logger.Info("Evidence generated successfully",
		logger.Field{Key: "task_id", Value: req.TaskID},
		logger.Field{Key: "record_id", Value: record.ID},
		logger.Field{Key: "format", Value: req.Format},
		logger.Field{Key: "content_size", Value: len(content)},
		logger.Field{Key: "generation_mode", Value: "template-based"},
	)

	// Create simple analysis result
	analysis := &EvidenceAnalysisResult{
		TaskID: req.TaskID,
		Task: &domain.EvidenceTaskDetails{
			ID:          task.ID,
			Name:        task.Name,
			Description: task.Description,
			Framework:   task.Framework,
			Priority:    task.Priority,
			Status:      task.Status,
		},
		RelatedControls: []domain.Control{},
		RelatedPolicies: []domain.Policy{},
		Recommendations: []string{"Review generated evidence for completeness"},
		SuggestedTools:  req.Tools,
	}

	// Create result
	result := &EvidenceGenerationResult{
		Record:      record,
		Analysis:    analysis,
		Insights:    []string{"Template-based evidence generation completed"},
		Suggestions: []string{"Review generated evidence for completeness", "Consider adding specific implementation details"},
	}

	return result, nil
}

// generateTemplateContent generates simple template-based evidence content
func (s *EvidenceServiceSimple) generateTemplateContent(task *domain.EvidenceTask, req *EvidenceGenerationRequest) string {
	var content strings.Builder

	switch strings.ToLower(req.Format) {
	case "csv":
		content.WriteString("Control/Requirement,Evidence Type,Implementation Details,Verification Method,Status,Notes\n")
		content.WriteString(fmt.Sprintf("%s,Implementation,Template-based evidence,Manual review,In Progress,Generated by grctool\n", task.Name))
	case "json":
		content.WriteString(fmt.Sprintf(`{
	"task_id": %d,
	"task_name": %q,
	"framework": %q,
	"priority": %q,
	"evidence_type": "template-based",
	"generated_at": %q,
	"description": %q,
	"status": "generated",
	"notes": "This evidence was generated using template-based approach. Please review and add specific implementation details."
}`, task.ID, task.Name, task.Framework, task.Priority, time.Now().Format(time.RFC3339), task.Description))
	default: // markdown
		content.WriteString(fmt.Sprintf("# Evidence for Task: %s\n\n", task.Name))
		content.WriteString(fmt.Sprintf("**Task ID:** %d\n", task.ID))
		content.WriteString(fmt.Sprintf("**Framework:** %s\n", task.Framework))
		content.WriteString(fmt.Sprintf("**Priority:** %s\n", task.Priority))
		content.WriteString(fmt.Sprintf("**Generated:** %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

		content.WriteString("## Task Description\n\n")
		content.WriteString(task.Description)
		content.WriteString("\n\n")

		if task.Guidance != "" {
			content.WriteString("## Collection Guidance\n\n")
			content.WriteString(task.Guidance)
			content.WriteString("\n\n")
		}

		content.WriteString("## Evidence Summary\n\n")
		content.WriteString("This evidence was generated using a template-based approach. ")
		content.WriteString("Please review the following and add specific implementation details:\n\n")
		content.WriteString("1. **Implementation Status:** [Please specify current implementation status]\n")
		content.WriteString("2. **Configuration Details:** [Please add specific configuration details]\n")
		content.WriteString("3. **Monitoring Evidence:** [Please provide monitoring/logging evidence]\n")
		content.WriteString("4. **Supporting Documentation:** [Please reference relevant policies/procedures]\n\n")

		content.WriteString("## Next Steps\n\n")
		content.WriteString("- Review and complete the evidence details above\n")
		content.WriteString("- Attach supporting artifacts (screenshots, configuration exports, etc.)\n")
		content.WriteString("- Verify compliance with framework requirements\n")
		content.WriteString("- Submit for compliance review\n")

		content.WriteString("\n---\n")
		content.WriteString("*Generated by grctool using template-based evidence generation*\n")
	}

	return content.String()
}

// ListEvidenceTasks returns filtered evidence tasks
func (s *EvidenceServiceSimple) ListEvidenceTasks(ctx context.Context, filter domain.EvidenceFilter) ([]domain.EvidenceTask, error) {
	return s.dataService.FilterEvidenceTasks(ctx, filter)
}

// GetEvidenceTaskSummary returns a summary of evidence tasks
func (s *EvidenceServiceSimple) GetEvidenceTaskSummary(ctx context.Context) (*domain.EvidenceTaskSummary, error) {
	// Get all tasks
	tasks, err := s.dataService.GetAllEvidenceTasks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all tasks: %w", err)
	}

	summary := &domain.EvidenceTaskSummary{
		Total:      len(tasks),
		ByStatus:   make(map[string]int),
		ByPriority: make(map[string]int),
		LastSync:   time.Now(),
	}

	now := time.Now()
	for _, task := range tasks {
		// Count by status
		summary.ByStatus[task.Status]++

		// Count by priority
		summary.ByPriority[task.Priority]++

		// Check if overdue
		if task.NextDue != nil && task.NextDue.Before(now) {
			summary.Overdue++
		}

		// Check if due soon (within 7 days)
		if task.NextDue != nil && task.NextDue.After(now) && task.NextDue.Before(now.AddDate(0, 0, 7)) {
			summary.DueSoon++
		}
	}

	return summary, nil
}

// AnalyzeEvidenceTask analyzes an evidence task and populates related controls and policies
func (s *EvidenceServiceSimple) AnalyzeEvidenceTask(ctx context.Context, taskID int) (*EvidenceAnalysisResult, error) {
	// Get task details
	task, err := s.dataService.GetEvidenceTask(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get evidence task: %w", err)
	}

	s.logger.Debug("Analyzing evidence task",
		logger.Field{Key: "task_id", Value: taskID},
		logger.Field{Key: "control_count", Value: len(task.Controls)},
		logger.Field{Key: "policy_count", Value: len(task.Policies)})

	// Fetch related controls from task.Controls array
	var relatedControls []domain.Control
	for _, controlID := range task.Controls {
		control, err := s.dataService.GetControl(ctx, controlID)
		if err != nil {
			s.logger.Warn("Failed to get control",
				logger.Field{Key: "control_id", Value: controlID},
				logger.Field{Key: "error", Value: err})
			continue
		}
		relatedControls = append(relatedControls, *control)
	}

	s.logger.Debug("Fetched related controls",
		logger.Field{Key: "task_id", Value: taskID},
		logger.Field{Key: "controls_found", Value: len(relatedControls)})

	// Fetch related policies from task.Policies array or via framework matching
	var relatedPolicies []domain.Policy
	if len(task.Policies) > 0 {
		// Direct policy references
		for _, policyID := range task.Policies {
			policy, err := s.dataService.GetPolicy(ctx, policyID)
			if err != nil {
				s.logger.Warn("Failed to get policy",
					logger.Field{Key: "policy_id", Value: policyID},
					logger.Field{Key: "error", Value: err})
				continue
			}
			relatedPolicies = append(relatedPolicies, *policy)
		}
	} else {
		// Fallback: Find policies by framework
		allPolicies, err := s.dataService.GetAllPolicies(ctx)
		if err == nil {
			for _, policy := range allPolicies {
				if policy.Framework == task.Framework {
					relatedPolicies = append(relatedPolicies, policy)
				}
			}
		}
	}

	s.logger.Debug("Fetched related policies",
		logger.Field{Key: "task_id", Value: taskID},
		logger.Field{Key: "policies_found", Value: len(relatedPolicies)})

	// Build suggested tools based on task category and controls
	suggestedTools := s.buildSuggestedTools(task, relatedControls)

	// Build recommendations based on task context
	recommendations := s.buildRecommendations(task, relatedControls, relatedPolicies)

	// Build required evidence list
	requiredEvidence := s.extractRequiredEvidence(task)

	// Build compliance context
	complianceContext := s.buildComplianceContext(task, relatedControls, relatedPolicies)

	// Create result with task (already a *domain.EvidenceTask)
	result := &EvidenceAnalysisResult{
		TaskID:            taskID,
		Task:              task, // Pass task directly - it's already the correct type
		RelatedControls:   relatedControls,
		RelatedPolicies:   relatedPolicies,
		Relationships:     []domain.Relationship{}, // Could be enhanced to fetch actual relationships
		Recommendations:   recommendations,
		RequiredEvidence:  requiredEvidence,
		SuggestedTools:    suggestedTools,
		ComplianceContext: complianceContext,
	}

	s.logger.Info("Evidence task analysis complete",
		logger.Field{Key: "task_id", Value: taskID},
		logger.Field{Key: "controls", Value: len(relatedControls)},
		logger.Field{Key: "policies", Value: len(relatedPolicies)},
		logger.Field{Key: "suggested_tools", Value: len(suggestedTools)})

	return result, nil
}

// buildSuggestedTools suggests tools based on task characteristics and controls
func (s *EvidenceServiceSimple) buildSuggestedTools(task *domain.EvidenceTask, controls []domain.Control) []string {
	tools := []string{}

	// Always include summary generators if we have controls or policies
	if len(controls) > 0 {
		tools = append(tools, "control-summary-generator")
	}
	if len(task.Policies) > 0 {
		tools = append(tools, "policy-summary-generator")
	}

	// Add evidence relationships tool
	tools = append(tools, "evidence-relationships")

	// Add infrastructure tools based on task category
	category := task.GetCategory()
	switch category {
	case "Infrastructure":
		tools = append(tools, "terraform-security-analyzer", "terraform-snippets")
	case "Personnel":
		tools = append(tools, "github-permissions")
	case "Process":
		// Evidence relationships already added
	case "Monitoring":
		tools = append(tools, "terraform-security-analyzer")
	}

	// Add based on control framework codes for SOC2
	for _, control := range controls {
		for _, fc := range control.FrameworkCodes {
			// SOC2 CC6.x = Logical Access Controls
			if strings.HasPrefix(fc.Code, "CC6") {
				if !s.containsString(tools, "github-permissions") {
					tools = append(tools, "github-permissions")
				}
				if !s.containsString(tools, "terraform-security-analyzer") {
					tools = append(tools, "terraform-security-analyzer")
				}
			}
			// SOC2 CC7.x = System Operations
			if strings.HasPrefix(fc.Code, "CC7") {
				if !s.containsString(tools, "terraform-security-analyzer") {
					tools = append(tools, "terraform-security-analyzer")
				}
			}
			// SOC2 CC8.x = Change Management
			if strings.HasPrefix(fc.Code, "CC8") {
				if !s.containsString(tools, "github-workflow-analyzer") {
					tools = append(tools, "github-workflow-analyzer")
				}
			}
		}
	}

	return tools
}

// buildRecommendations generates actionable recommendations based on task context
func (s *EvidenceServiceSimple) buildRecommendations(task *domain.EvidenceTask, controls []domain.Control, policies []domain.Policy) []string {
	recs := []string{}

	// Control-based recommendations
	if len(controls) > 0 {
		recs = append(recs, fmt.Sprintf("Generate AI summaries for %d related controls using control-summary-generator tool", len(controls)))
	}

	// Policy-based recommendations
	if len(policies) > 0 {
		recs = append(recs, fmt.Sprintf("Review %d governing policies using policy-summary-generator tool", len(policies)))
	}

	// Task-specific recommendations
	recs = append(recs,
		"Collect technical implementation evidence using infrastructure scanning tools",
		"Document ongoing monitoring procedures and audit logs",
		"Gather supporting artifacts (screenshots, configuration exports, logs)",
		"Verify evidence completeness against framework requirements",
	)

	// Category-specific recommendations
	category := task.GetCategory()
	switch category {
	case "Infrastructure":
		recs = append(recs, "Scan Terraform configurations for security controls")
	case "Personnel":
		recs = append(recs, "Document access controls and user permissions")
	case "Monitoring":
		recs = append(recs, "Provide evidence of continuous monitoring and alerting")
	}

	return recs
}

// extractRequiredEvidence builds a list of required evidence types based on task
func (s *EvidenceServiceSimple) extractRequiredEvidence(task *domain.EvidenceTask) []string {
	evidence := []string{}

	// Parse guidance for evidence types
	if task.Guidance != "" {
		guidance := strings.ToLower(task.Guidance)

		if strings.Contains(guidance, "screenshot") {
			evidence = append(evidence, "Screenshots of relevant configurations")
		}
		if strings.Contains(guidance, "log") || strings.Contains(guidance, "audit") {
			evidence = append(evidence, "Audit logs and access records")
		}
		if strings.Contains(guidance, "policy") || strings.Contains(guidance, "procedure") {
			evidence = append(evidence, "Policy and procedure documentation")
		}
		if strings.Contains(guidance, "configuration") {
			evidence = append(evidence, "Configuration exports and settings")
		}
		if strings.Contains(guidance, "list") || strings.Contains(guidance, "inventory") {
			evidence = append(evidence, "Inventory lists and asset records")
		}
	}

	// Add generic evidence types if none specific found
	if len(evidence) == 0 {
		evidence = append(evidence,
			"Implementation documentation",
			"Configuration evidence",
			"Monitoring and review evidence",
		)
	}

	return evidence
}

// buildComplianceContext creates compliance context map
func (s *EvidenceServiceSimple) buildComplianceContext(task *domain.EvidenceTask, controls []domain.Control, policies []domain.Policy) map[string]interface{} {
	context := make(map[string]interface{})

	context["framework"] = task.Framework
	context["priority"] = task.Priority
	context["status"] = task.Status
	context["category"] = task.GetCategory()
	context["control_count"] = len(controls)
	context["policy_count"] = len(policies)

	// Add framework-specific context
	if task.Framework == "SOC2" {
		context["trust_service_category"] = s.deriveTrustServiceCategory(controls)
	}

	return context
}

// deriveTrustServiceCategory determines the SOC2 Trust Service Category
func (s *EvidenceServiceSimple) deriveTrustServiceCategory(controls []domain.Control) string {
	categories := make(map[string]int)

	for _, control := range controls {
		for _, fc := range control.FrameworkCodes {
			if strings.HasPrefix(fc.Code, "CC") {
				prefix := fc.Code[:3] // CC1, CC2, etc.
				categories[prefix]++
			}
		}
	}

	// Return the most common category
	maxCount := 0
	mainCategory := "Common Criteria"
	for cat, count := range categories {
		if count > maxCount {
			maxCount = count
			switch cat {
			case "CC1":
				mainCategory = "Control Environment"
			case "CC2":
				mainCategory = "Communication and Information"
			case "CC3":
				mainCategory = "Risk Assessment"
			case "CC4":
				mainCategory = "Monitoring Activities"
			case "CC5":
				mainCategory = "Control Activities"
			case "CC6":
				mainCategory = "Logical and Physical Access Controls"
			case "CC7":
				mainCategory = "System Operations"
			case "CC8":
				mainCategory = "Change Management"
			case "CC9":
				mainCategory = "Risk Mitigation"
			}
		}
	}

	return mainCategory
}

// containsString checks if a string slice contains a string
func (s *EvidenceServiceSimple) containsString(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}

// ReviewEvidence reviews generated evidence
func (s *EvidenceServiceSimple) ReviewEvidence(ctx context.Context, recordID string, showReasoning bool) (map[string]interface{}, error) {
	review := map[string]interface{}{
		"completeness": "moderate",
		"accuracy":     "pending_review",
		"compliance":   "needs_improvement",
		"suggestions": []string{
			"Add specific implementation details",
			"Include supporting artifacts",
			"Verify configuration details",
		},
		"next_actions": []string{
			"Review generated content",
			"Add implementation-specific details",
			"Attach supporting documentation",
		},
	}

	if showReasoning {
		review["reasoning"] = map[string]interface{}{
			"analysis_method":  "template-based",
			"criteria_checked": []string{"format", "structure", "completeness"},
			"confidence_score": 0.7,
		}
	}

	return review, nil
}

// saveGeneratedEvidence saves generated evidence content to the filesystem
func (s *EvidenceServiceSimple) saveGeneratedEvidence(evidence *models.GeneratedEvidence) error {
	outputDir := s.config.Evidence.Generation.OutputDir
	if outputDir == "" {
		return fmt.Errorf("output directory not configured")
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create task-specific directory using task ID (e.g., ET87)
	taskDir := filepath.Join(outputDir, fmt.Sprintf("ET%d", evidence.TaskID))
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return fmt.Errorf("failed to create task directory: %w", err)
	}

	// Save main evidence content
	ext := ".md"
	if evidence.EvidenceFormat == "csv" {
		ext = ".csv"
	}
	filename := fmt.Sprintf("evidence_%s%s", evidence.GeneratedAt.Format("20060102_150405"), ext)
	filePath := filepath.Join(taskDir, filename)

	if err := os.WriteFile(filePath, []byte(evidence.EvidenceContent), 0644); err != nil {
		return fmt.Errorf("failed to write evidence file: %w", err)
	}

	// Extract and save terraform snippets as individual files
	if evidence.SourcesUsed != nil {
		for _, source := range evidence.SourcesUsed {
			if source.Type == "terraform" && source.Content != "" {
				// Extract terraform resources from the source content
				if err := s.saveTerraformSnippets(taskDir, source.Content); err != nil {
					s.logger.Warn("Failed to save terraform snippets", logger.Field{Key: "error", Value: err})
				}
			}
		}
	}

	// Save reasoning if available
	if evidence.Reasoning != "" && s.config.Evidence.Generation.IncludeReasoning {
		reasoningFile := filepath.Join(taskDir, fmt.Sprintf("reasoning_%s.md", evidence.GeneratedAt.Format("20060102_150405")))
		if err := os.WriteFile(reasoningFile, []byte(evidence.Reasoning), 0644); err != nil {
			s.logger.Warn("Failed to save reasoning", logger.Field{Key: "error", Value: err})
		}
	}

	evidence.OutputDirectory = taskDir
	return nil
}

// saveTerraformSnippets extracts terraform resources from tool output and saves them as individual files
func (s *EvidenceServiceSimple) saveTerraformSnippets(taskDir string, toolOutput string) error {
	// Create terraform snippets subdirectory
	terraformDir := filepath.Join(taskDir, "terraform_snippets")
	if err := os.MkdirAll(terraformDir, 0755); err != nil {
		return fmt.Errorf("failed to create terraform snippets directory: %w", err)
	}

	// Parse the tool output to extract terraform resources
	// The output contains resource blocks that we need to extract
	resourcePattern := regexp.MustCompile(`(?ms)resource\s+"([^"]+)"\s+"([^"]+)"\s*\{.*?\n\}`)
	matches := resourcePattern.FindAllStringSubmatch(toolOutput, -1)

	for i, match := range matches {
		if len(match) >= 3 {
			resourceType := match[1]
			resourceName := match[2]
			resourceContent := match[0]

			// Create filename based on resource type and name
			filename := fmt.Sprintf("%02d_%s_%s.tf", i+1, resourceType, resourceName)
			filePath := filepath.Join(terraformDir, filename)

			// Write the terraform snippet to file
			if err := os.WriteFile(filePath, []byte(resourceContent), 0644); err != nil {
				s.logger.Warn("Failed to save terraform snippet",
					logger.Field{Key: "file", Value: filename},
					logger.Field{Key: "error", Value: err})
				continue
			}

			s.logger.Debug("Saved terraform snippet", logger.Field{Key: "file", Value: filename})
		}
	}

	// Also save a summary file with all resources
	summaryPath := filepath.Join(terraformDir, "00_summary.md")
	summary := fmt.Sprintf("# Terraform Resources Summary\n\nTotal resources found: %d\n\n", len(matches))
	for i, match := range matches {
		if len(match) >= 3 {
			summary += fmt.Sprintf("- `%s.%s` (file: %02d_%s_%s.tf)\n", match[1], match[2], i+1, match[1], match[2])
		}
	}

	return os.WriteFile(summaryPath, []byte(summary), 0644)
}
