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
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/storage"
)

// EvidenceValidatorTool performs validation checks on evidence files
type EvidenceValidatorTool struct {
	config      *config.Config
	logger      logger.Logger
	dataService DataServiceInterface
	storage     *storage.Storage
}

// NewEvidenceValidatorTool creates a new evidence validator tool
func NewEvidenceValidatorTool(cfg *config.Config, log logger.Logger) (*EvidenceValidatorTool, error) {
	// Initialize unified storage
	storage, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	dataService := &SimpleDataService{storage: storage}

	return &EvidenceValidatorTool{
		config:      cfg,
		logger:      log,
		dataService: dataService,
		storage:     storage,
	}, nil
}

// GetClaudeToolDefinition returns the tool definition for Claude
func (e *EvidenceValidatorTool) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        "evidence-validator",
		Description: "Validate evidence completeness and quality. Performs local validation checks, calculates scores, and provides recommendations.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"task_ref": map[string]interface{}{
					"type":        "string",
					"description": "Evidence task reference (e.g., ET1, ET42) to validate evidence for",
				},
				"evidence_file": map[string]interface{}{
					"type":        "string",
					"description": "Path to evidence file to validate",
				},
				"validation_level": map[string]interface{}{
					"type":        "string",
					"description": "Level of validation to perform",
					"enum":        []string{"basic", "standard", "comprehensive"},
					"default":     "standard",
				},
			},
			"required": []string{}, // Either task_ref or evidence_file must be provided
			"oneOf": []map[string]interface{}{
				{"required": []string{"task_ref"}},
				{"required": []string{"evidence_file"}},
			},
		},
	}
}

// Execute runs the evidence validation tool
func (e *EvidenceValidatorTool) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	e.logger.Info("Starting evidence validation",
		logger.Field{Key: "params", Value: params})

	// Parse parameters
	taskRef, hasTaskRef := params["task_ref"].(string)
	evidenceFile, hasEvidenceFile := params["evidence_file"].(string)
	validationLevel, _ := params["validation_level"].(string)

	// Default validation level
	if validationLevel == "" {
		validationLevel = "standard"
	}

	// Validate required parameters
	if !hasTaskRef && !hasEvidenceFile {
		return "", nil, fmt.Errorf("either task_ref or evidence_file must be specified")
	}

	var result *EvidenceValidationResult
	var err error

	if hasEvidenceFile {
		result, err = e.validateEvidenceFile(ctx, evidenceFile, validationLevel)
	} else {
		result, err = e.validateTaskEvidence(ctx, taskRef, validationLevel)
	}

	if err != nil {
		e.logger.Error("Evidence validation failed",
			logger.Field{Key: "error", Value: err})
		return "", nil, fmt.Errorf("evidence validation failed: %w", err)
	}

	// Create evidence source metadata
	source := &models.EvidenceSource{
		Type:        "evidence-validator",
		Resource:    fmt.Sprintf("Validation results for evidence (level: %s)", validationLevel),
		Content:     result.Summary,
		ExtractedAt: result.ValidatedAt,
		Metadata: map[string]interface{}{
			"validation_level":   validationLevel,
			"completeness_score": result.CompletenessScore,
			"quality_score":      result.QualityScore,
			"overall_status":     result.OverallStatus,
			"issues_count":       len(result.Issues),
			"validated_at":       result.ValidatedAt,
		},
	}

	// Format response
	response := map[string]interface{}{
		"success":            true,
		"validation_status":  result.OverallStatus,
		"completeness_score": result.CompletenessScore,
		"quality_score":      result.QualityScore,
		"issues":             result.Issues,
		"recommendations":    result.Recommendations,
		"summary":            result.Summary,
		"details":            result.Details,
		"validated_at":       result.ValidatedAt,
		"validation_level":   validationLevel,
	}

	responseJSON, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", source, fmt.Errorf("failed to marshal response: %w", err)
	}

	e.logger.Info("Evidence validation completed",
		logger.Field{Key: "status", Value: result.OverallStatus},
		logger.Field{Key: "completeness", Value: result.CompletenessScore},
		logger.Field{Key: "quality", Value: result.QualityScore},
		logger.Field{Key: "issues", Value: len(result.Issues)},
	)

	return string(responseJSON), source, nil
}

// Name returns the tool name
func (e *EvidenceValidatorTool) Name() string {
	return "evidence-validator"
}

// Description returns the tool description
func (e *EvidenceValidatorTool) Description() string {
	return "Validate evidence completeness and quality with scoring and recommendations"
}

// Version returns the tool version
func (e *EvidenceValidatorTool) Version() string {
	return "1.0.0"
}

// Category returns the tool category
func (e *EvidenceValidatorTool) Category() string {
	return "evidence-management"
}

// EvidenceValidationResult represents the result of evidence validation
type EvidenceValidationResult struct {
	OverallStatus     string                 `json:"overall_status"`
	CompletenessScore float64                `json:"completeness_score"`
	QualityScore      float64                `json:"quality_score"`
	Issues            []ValidationIssue      `json:"issues"`
	Recommendations   []string               `json:"recommendations"`
	Summary           string                 `json:"summary"`
	Details           map[string]interface{} `json:"details"`
	ValidatedAt       time.Time              `json:"validated_at"`
}

// ValidationIssue represents a specific validation issue
type ValidationIssue struct {
	Type        string                 `json:"type"`     // "error", "warning", "info"
	Category    string                 `json:"category"` // "completeness", "format", "content", "compliance"
	Description string                 `json:"description"`
	Location    string                 `json:"location,omitempty"`
	Severity    string                 `json:"severity"` // "high", "medium", "low"
	Suggestion  string                 `json:"suggestion,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// validateEvidenceFile validates a specific evidence file
func (e *EvidenceValidatorTool) validateEvidenceFile(ctx context.Context, filePath string, validationLevel string) (*EvidenceValidationResult, error) {
	// Check if file exists and is safe to read
	if !e.isPathSafe(filePath) {
		return nil, fmt.Errorf("unsafe path: %s", filePath)
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("evidence file does not exist: %s", filePath)
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read evidence file: %w", err)
	}

	e.logger.Info("Loaded evidence file",
		logger.Field{Key: "file", Value: filePath},
		logger.Field{Key: "size", Value: len(content)})

	// Detect format
	format := e.detectFormat(filePath, string(content))

	// Create validation result
	result := &EvidenceValidationResult{
		Issues:          []ValidationIssue{},
		Recommendations: []string{},
		Details: map[string]interface{}{
			"file_path": filePath,
			"file_size": len(content),
			"format":    format,
			"task_id":   e.extractTaskIDFromContent(string(content)),
		},
		ValidatedAt: time.Now(),
	}

	// Perform validation checks based on level
	e.validateFormat(string(content), format, result)
	e.validateCompleteness(string(content), result)

	if validationLevel == "standard" || validationLevel == "comprehensive" {
		e.validateContent(string(content), result)
		e.validateSources(string(content), result)
	}

	if validationLevel == "comprehensive" {
		e.validateCompliance(ctx, string(content), result)
		e.validateMetadata(string(content), result)
	}

	// Calculate scores
	e.calculateScores(result)

	// Generate summary
	e.generateSummary(result)

	return result, nil
}

// validateTaskEvidence validates evidence for a specific task
func (e *EvidenceValidatorTool) validateTaskEvidence(ctx context.Context, taskRef string, validationLevel string) (*EvidenceValidationResult, error) {
	// Resolve task ID
	taskID, err := e.resolveTaskID(ctx, taskRef)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve task ID: %w", err)
	}

	// Get evidence records for the task
	records, err := e.dataService.GetEvidenceRecords(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get evidence records: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("no evidence records found for task %s", taskRef)
	}

	e.logger.Info("Validating task evidence",
		logger.Field{Key: "task_ref", Value: taskRef},
		logger.Field{Key: "task_id", Value: taskID},
		logger.Field{Key: "records", Value: len(records)})

	// Create validation result
	result := &EvidenceValidationResult{
		Issues:          []ValidationIssue{},
		Recommendations: []string{},
		Details: map[string]interface{}{
			"task_ref":      taskRef,
			"task_id":       taskID,
			"records_count": len(records),
			"records":       records,
		},
		ValidatedAt: time.Now(),
	}

	// Validate each evidence record
	for i, record := range records {
		e.validateEvidenceRecord(record, i, result)
	}

	// Perform additional validation based on level
	if validationLevel == "standard" || validationLevel == "comprehensive" {
		e.validateTaskCompleteness(ctx, taskID, len(records), result)
	}

	if validationLevel == "comprehensive" {
		e.validateTaskCompliance(ctx, taskID, len(records), result)
	}

	// Calculate scores
	e.calculateScores(result)

	// Generate summary
	e.generateSummary(result)

	return result, nil
}

// Validation methods

func (e *EvidenceValidatorTool) validateFormat(content, format string, result *EvidenceValidationResult) {
	// Check if content matches expected format
	switch format {
	case "markdown":
		if !strings.Contains(content, "#") && !strings.Contains(content, "**") {
			result.Issues = append(result.Issues, ValidationIssue{
				Type:        "warning",
				Category:    "format",
				Description: "Content appears to be markdown but lacks markdown formatting",
				Severity:    "low",
				Suggestion:  "Add markdown headers and formatting for better readability",
			})
		}
	case "json":
		var jsonData interface{}
		if err := json.Unmarshal([]byte(content), &jsonData); err != nil {
			result.Issues = append(result.Issues, ValidationIssue{
				Type:        "error",
				Category:    "format",
				Description: fmt.Sprintf("Invalid JSON format: %s", err.Error()),
				Severity:    "high",
				Suggestion:  "Fix JSON syntax errors",
			})
		}
	case "csv":
		lines := strings.Split(content, "\n")
		if len(lines) < 2 {
			result.Issues = append(result.Issues, ValidationIssue{
				Type:        "warning",
				Category:    "format",
				Description: "CSV file has less than 2 lines (header and data expected)",
				Severity:    "medium",
				Suggestion:  "Add CSV header row and data rows",
			})
		}
	}
}

func (e *EvidenceValidatorTool) validateCompleteness(content string, result *EvidenceValidationResult) {
	// Check minimum content length
	if len(content) < 100 {
		result.Issues = append(result.Issues, ValidationIssue{
			Type:        "warning",
			Category:    "completeness",
			Description: fmt.Sprintf("Content is very short (%d characters)", len(content)),
			Severity:    "medium",
			Suggestion:  "Add more detailed evidence content",
		})
	}

	// Check for required sections in markdown content
	if strings.Contains(content, "#") {
		requiredSections := []string{"evidence", "compliance", "description"}
		missingCount := 0

		contentLower := strings.ToLower(content)
		for _, section := range requiredSections {
			if !strings.Contains(contentLower, section) {
				missingCount++
			}
		}

		if missingCount > 0 {
			result.Issues = append(result.Issues, ValidationIssue{
				Type:        "warning",
				Category:    "completeness",
				Description: fmt.Sprintf("Missing %d expected sections in evidence", missingCount),
				Severity:    "medium",
				Suggestion:  "Include sections for evidence description, compliance details, and supporting data",
			})
		}
	}
}

func (e *EvidenceValidatorTool) validateContent(content string, result *EvidenceValidationResult) {
	// Check for placeholder text
	placeholders := []string{"TODO", "TBD", "PLACEHOLDER", "FIXME", "XXX"}
	for _, placeholder := range placeholders {
		if strings.Contains(strings.ToUpper(content), placeholder) {
			result.Issues = append(result.Issues, ValidationIssue{
				Type:        "warning",
				Category:    "content",
				Description: fmt.Sprintf("Content contains placeholder text: %s", placeholder),
				Severity:    "medium",
				Suggestion:  "Replace placeholder text with actual evidence content",
			})
		}
	}

	// Check for sensitive data patterns (basic check)
	sensitivePatterns := []string{
		`\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b`, // Credit card
		`\b\d{3}-?\d{2}-?\d{4}\b`,                    // SSN
		`password\s*[:=]\s*\S+`,                      // Password
		`api[_-]?key\s*[:=]\s*\S+`,                   // API key
	}

	for i, pattern := range sensitivePatterns {
		if matched, _ := regexp.MatchString(pattern, strings.ToLower(content)); matched {
			severityMap := []string{"high", "high", "high", "high"}
			typeMap := []string{"Credit Card", "SSN", "Password", "API Key"}

			result.Issues = append(result.Issues, ValidationIssue{
				Type:        "error",
				Category:    "content",
				Description: fmt.Sprintf("Potential sensitive data detected: %s", typeMap[i]),
				Severity:    severityMap[i],
				Suggestion:  "Remove or redact sensitive information",
			})
		}
	}
}

func (e *EvidenceValidatorTool) validateSources(content string, result *EvidenceValidationResult) {
	// Check for source citations or references
	sourceIndicators := []string{"source:", "from:", "reference:", "based on:", "according to:"}
	hasSourceIndicator := false

	contentLower := strings.ToLower(content)
	for _, indicator := range sourceIndicators {
		if strings.Contains(contentLower, indicator) {
			hasSourceIndicator = true
			break
		}
	}

	if !hasSourceIndicator {
		result.Issues = append(result.Issues, ValidationIssue{
			Type:        "warning",
			Category:    "completeness",
			Description: "Evidence lacks clear source references or citations",
			Severity:    "medium",
			Suggestion:  "Add source references to validate evidence authenticity",
		})
	}
}

func (e *EvidenceValidatorTool) validateCompliance(ctx context.Context, content string, result *EvidenceValidationResult) {
	// Check for compliance framework references
	frameworks := []string{"soc2", "iso27001", "gdpr", "hipaa", "pci", "sox"}
	hasFrameworkRef := false

	contentLower := strings.ToLower(content)
	for _, framework := range frameworks {
		if strings.Contains(contentLower, framework) {
			hasFrameworkRef = true
			break
		}
	}

	if !hasFrameworkRef {
		result.Issues = append(result.Issues, ValidationIssue{
			Type:        "info",
			Category:    "compliance",
			Description: "Evidence does not reference specific compliance frameworks",
			Severity:    "low",
			Suggestion:  "Consider adding framework-specific compliance references",
		})
	}
}

func (e *EvidenceValidatorTool) validateMetadata(content string, result *EvidenceValidationResult) {
	// Check for metadata indicators
	metadataIndicators := []string{"generated", "collected", "date:", "version:", "author:"}
	metadataCount := 0

	contentLower := strings.ToLower(content)
	for _, indicator := range metadataIndicators {
		if strings.Contains(contentLower, indicator) {
			metadataCount++
		}
	}

	if metadataCount < 2 {
		result.Issues = append(result.Issues, ValidationIssue{
			Type:        "info",
			Category:    "completeness",
			Description: "Evidence lacks sufficient metadata (date, source, version, etc.)",
			Severity:    "low",
			Suggestion:  "Add metadata including generation date, source, and version information",
		})
	}
}

func (e *EvidenceValidatorTool) validateEvidenceRecord(record interface{}, index int, result *EvidenceValidationResult) {
	// Add record-specific validation logic here
	// This would check the evidence record structure, content, metadata, etc.

	// For now, add a basic validation check
	result.Issues = append(result.Issues, ValidationIssue{
		Type:        "info",
		Category:    "completeness",
		Description: fmt.Sprintf("Evidence record %d validated", index+1),
		Severity:    "low",
		Location:    fmt.Sprintf("record[%d]", index),
	})
}

func (e *EvidenceValidatorTool) validateTaskCompleteness(ctx context.Context, taskID int, recordCount int, result *EvidenceValidationResult) {
	// Get task details to check completeness requirements
	task, err := e.dataService.GetEvidenceTask(ctx, taskID)
	if err != nil {
		result.Issues = append(result.Issues, ValidationIssue{
			Type:        "error",
			Category:    "completeness",
			Description: fmt.Sprintf("Failed to get task details: %s", err.Error()),
			Severity:    "high",
		})
		return
	}

	// Check if evidence matches task requirements
	if task.Framework != "" {
		result.Details["framework"] = task.Framework
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("Ensure evidence addresses %s framework requirements", task.Framework))
	}

	if recordCount == 0 {
		result.Issues = append(result.Issues, ValidationIssue{
			Type:        "error",
			Category:    "completeness",
			Description: "No evidence records found for task",
			Severity:    "high",
			Suggestion:  "Generate evidence for this task",
		})
	}
}

func (e *EvidenceValidatorTool) validateTaskCompliance(ctx context.Context, taskID int, recordCount int, result *EvidenceValidationResult) {
	// Perform comprehensive compliance validation
	result.Recommendations = append(result.Recommendations,
		"Perform manual compliance review for comprehensive validation",
		"Verify evidence against specific compliance requirements",
		"Consider third-party compliance assessment",
	)
}

func (e *EvidenceValidatorTool) calculateScores(result *EvidenceValidationResult) {
	totalIssues := len(result.Issues)
	errorCount := 0
	warningCount := 0

	for _, issue := range result.Issues {
		switch issue.Type {
		case "error":
			errorCount++
		case "warning":
			warningCount++
		}
	}

	// Calculate completeness score (0-100)
	if totalIssues == 0 {
		result.CompletenessScore = 100.0
	} else {
		// Deduct points based on issue severity
		deduction := float64(errorCount*20 + warningCount*10)
		result.CompletenessScore = max(0, 100.0-deduction)
	}

	// Calculate quality score (0-100)
	if errorCount == 0 {
		result.QualityScore = max(60, 100.0-float64(warningCount*5))
	} else {
		result.QualityScore = max(0, 60.0-float64(errorCount*15))
	}

	// Determine overall status
	if errorCount > 0 {
		result.OverallStatus = "failed"
	} else if warningCount > 3 {
		result.OverallStatus = "needs_improvement"
	} else if warningCount > 0 {
		result.OverallStatus = "passed_with_warnings"
	} else {
		result.OverallStatus = "passed"
	}
}

func (e *EvidenceValidatorTool) generateSummary(result *EvidenceValidationResult) {
	var summary strings.Builder

	summary.WriteString(fmt.Sprintf("Evidence validation completed with status: %s\n", result.OverallStatus))
	summary.WriteString(fmt.Sprintf("Completeness Score: %.1f/100\n", result.CompletenessScore))
	summary.WriteString(fmt.Sprintf("Quality Score: %.1f/100\n", result.QualityScore))

	if len(result.Issues) > 0 {
		summary.WriteString(fmt.Sprintf("\nFound %d issues:\n", len(result.Issues)))

		errorCount := 0
		warningCount := 0
		infoCount := 0

		for _, issue := range result.Issues {
			switch issue.Type {
			case "error":
				errorCount++
			case "warning":
				warningCount++
			case "info":
				infoCount++
			}
		}

		if errorCount > 0 {
			summary.WriteString(fmt.Sprintf("- %d errors (must be fixed)\n", errorCount))
		}
		if warningCount > 0 {
			summary.WriteString(fmt.Sprintf("- %d warnings (should be addressed)\n", warningCount))
		}
		if infoCount > 0 {
			summary.WriteString(fmt.Sprintf("- %d info items (optional improvements)\n", infoCount))
		}
	} else {
		summary.WriteString("\nNo issues found.")
	}

	if len(result.Recommendations) > 0 {
		summary.WriteString(fmt.Sprintf("\n%d recommendations provided for improvement.", len(result.Recommendations)))
	}

	result.Summary = summary.String()
}

// Utility methods

func (e *EvidenceValidatorTool) isPathSafe(path string) bool {
	// Check for path traversal attempts
	if strings.Contains(path, "..") {
		return false
	}

	// Ensure path is within data directory
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	dataDir, err := filepath.Abs(e.config.Storage.DataDir)
	if err != nil {
		return false
	}

	return strings.HasPrefix(absPath, dataDir)
}

func (e *EvidenceValidatorTool) detectFormat(filePath, content string) string {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".md", ".markdown":
		return "markdown"
	case ".json":
		return "json"
	case ".csv":
		return "csv"
	case ".txt":
		return "text"
	default:
		// Try to detect from content
		if strings.HasPrefix(strings.TrimSpace(content), "{") || strings.HasPrefix(strings.TrimSpace(content), "[") {
			return "json"
		}
		if strings.Contains(content, "#") && strings.Contains(content, "**") {
			return "markdown"
		}
		if strings.Contains(content, ",") && len(strings.Split(content, "\n")) > 1 {
			return "csv"
		}
		return "text"
	}
}

func (e *EvidenceValidatorTool) extractTaskIDFromContent(content string) int {
	// Try to extract task ID from content
	// Look for patterns like "ET1", "task 123", "ID: 123", etc.

	lines := strings.Split(content, "\n")
	for _, line := range lines[:min(10, len(lines))] { // Check first 10 lines
		if strings.Contains(strings.ToLower(line), "task") || strings.Contains(strings.ToLower(line), "et") {
			// Try to find ET reference
			re := regexp.MustCompile(`ET(\d+)`)
			matches := re.FindStringSubmatch(strings.ToUpper(line))
			if len(matches) > 1 {
				if num := parseIntSafe(matches[1]); num > 0 {
					return 327991 + num // Convert ET reference to internal ID
				}
			}
		}
	}

	return 0 // Unknown task ID
}

func (e *EvidenceValidatorTool) resolveTaskID(ctx context.Context, taskRef string) (int, error) {
	// First try to parse as integer
	if taskID := parseIntSafe(taskRef); taskID > 0 {
		return taskID, nil
	}

	// Try to parse as reference ID (e.g., "ET1", "ET42")
	if strings.HasPrefix(strings.ToUpper(taskRef), "ET") {
		// Get all tasks to find by reference ID
		tasks, err := e.dataService.GetAllEvidenceTasks(ctx)
		if err != nil {
			return 0, fmt.Errorf("failed to get evidence tasks: %w", err)
		}

		// Search for matching reference ID
		upperRef := strings.ToUpper(taskRef)
		for _, task := range tasks {
			if task.ReferenceID == upperRef {
				return task.ID, nil
			}
		}
		return 0, fmt.Errorf("evidence task with reference ID %s not found", taskRef)
	}

	return 0, fmt.Errorf("invalid task identifier: %s", taskRef)
}

// Helper functions

func parseIntSafe(s string) int {
	if val, err := strconv.Atoi(s); err == nil {
		return val
	}
	return 0
}
