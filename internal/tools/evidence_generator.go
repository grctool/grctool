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
	"strconv"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/storage"
	"github.com/grctool/grctool/internal/utils"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// EvidenceGeneratorTool coordinates evidence generation using AI and sub-tools
type EvidenceGeneratorTool struct {
	config      *config.Config
	logger      logger.Logger
	dataService DataServiceInterface
	storage     *storage.Storage
}

// NewEvidenceGeneratorTool creates a new evidence generator tool
func NewEvidenceGeneratorTool(cfg *config.Config, log logger.Logger) (*EvidenceGeneratorTool, error) {
	// Initialize unified storage
	storage, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	dataService := &SimpleDataService{storage: storage}

	return &EvidenceGeneratorTool{
		config:      cfg,
		logger:      log,
		dataService: dataService,
		storage:     storage,
	}, nil
}

// GetClaudeToolDefinition returns the tool definition for external AI tools
func (e *EvidenceGeneratorTool) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        "evidence-generator",
		Description: "Generate compliance evidence using AI coordination with sub-tools (terraform, github, docs). Outputs evidence in multiple formats with source tracking and metadata.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"prompt_file": map[string]interface{}{
					"type":        "string",
					"description": "Path to prompt file containing evidence generation instructions",
				},
				"task_ref": map[string]interface{}{
					"type":        "string",
					"description": "Evidence task reference (e.g., ET1, ET42) to generate evidence for",
				},
				"tools": map[string]interface{}{
					"type":        "array",
					"description": "Sub-tools to coordinate during evidence generation",
					"items": map[string]interface{}{
						"type": "string",
						"enum": []string{"terraform", "github", "docs", "manual"},
					},
					"default": []string{"terraform", "github", "docs"},
				},
				"format": map[string]interface{}{
					"type":        "string",
					"description": "Output format for generated evidence",
					"enum":        []string{"markdown", "csv", "json"},
					"default":     "markdown",
				},
				"output_dir": map[string]interface{}{
					"type":        "string",
					"description": "Directory to save generated evidence (optional)",
				},
			},
			"required": []string{}, // Either prompt_file or task_ref must be provided
			"oneOf": []map[string]interface{}{
				{"required": []string{"prompt_file"}},
				{"required": []string{"task_ref"}},
			},
		},
	}
}

// Execute runs the evidence generation tool
func (e *EvidenceGeneratorTool) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	e.logger.Info("Starting evidence generation",
		logger.Field{Key: "params", Value: params})

	// Parse parameters
	promptFile, hasPromptFile := params["prompt_file"].(string)
	taskRef, hasTaskRef := params["task_ref"].(string)
	format, _ := params["format"].(string)
	outputDir, _ := params["output_dir"].(string)

	// Default format
	if format == "" {
		format = "markdown"
	}

	// Parse tools list
	var toolsList []string
	if tools, ok := params["tools"].([]interface{}); ok {
		for _, tool := range tools {
			if toolStr, ok := tool.(string); ok {
				toolsList = append(toolsList, toolStr)
			}
		}
	} else if tools, ok := params["tools"].([]string); ok {
		toolsList = tools
	} else {
		toolsList = []string{"terraform", "github", "docs"}
	}

	// Validate required parameters
	if !hasPromptFile && !hasTaskRef {
		return "", nil, fmt.Errorf("either prompt_file or task_ref must be specified")
	}

	var result *EvidenceGenerationResult
	var err error

	if hasPromptFile {
		result, err = e.generateFromPromptFile(ctx, promptFile, toolsList, format, outputDir)
	} else {
		result, err = e.generateFromTaskRef(ctx, taskRef, toolsList, format, outputDir)
	}

	if err != nil {
		e.logger.Error("Evidence generation failed",
			logger.Field{Key: "error", Value: err})
		return "", nil, fmt.Errorf("evidence generation failed: %w", err)
	}

	// Create evidence source metadata
	source := &models.EvidenceSource{
		Type:        "evidence-generator",
		Resource:    fmt.Sprintf("Generated evidence using tools: %s", strings.Join(toolsList, ", ")),
		Content:     result.Content,
		ExtractedAt: result.GeneratedAt,
		Metadata: map[string]interface{}{
			"format":       format,
			"tools_used":   toolsList,
			"generated_at": result.GeneratedAt,
			"task_id":      result.TaskID,
			"synthesis":    result.Synthesis,
			"sources":      result.Sources,
		},
	}

	// Format response
	response := map[string]interface{}{
		"success":      true,
		"content":      result.Content,
		"format":       format,
		"task_id":      result.TaskID,
		"generated_at": result.GeneratedAt,
		"synthesis":    result.Synthesis,
		"sources":      result.Sources,
		"metadata":     result.Metadata,
	}

	if result.OutputPath != "" {
		response["output_path"] = result.OutputPath
	}

	responseJSON, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", source, fmt.Errorf("failed to marshal response: %w", err)
	}

	e.logger.Info("Evidence generation completed",
		logger.Field{Key: "task_id", Value: result.TaskID},
		logger.Field{Key: "format", Value: format},
		logger.Field{Key: "sources_count", Value: len(result.Sources)},
	)

	return string(responseJSON), source, nil
}

// Name returns the tool name
func (e *EvidenceGeneratorTool) Name() string {
	return "evidence-generator"
}

// Description returns the tool description
func (e *EvidenceGeneratorTool) Description() string {
	return "Generate compliance evidence using AI coordination with sub-tools"
}

// Version returns the tool version
func (e *EvidenceGeneratorTool) Version() string {
	return "1.0.0"
}

// Category returns the tool category
func (e *EvidenceGeneratorTool) Category() string {
	return "evidence-management"
}

// EvidenceGenerationResult represents the result of evidence generation
type EvidenceGenerationResult struct {
	TaskID      int                     `json:"task_id"`
	Content     string                  `json:"content"`
	Format      string                  `json:"format"`
	GeneratedAt time.Time               `json:"generated_at"`
	Synthesis   string                  `json:"synthesis"`
	Sources     []models.EvidenceSource `json:"sources"`
	Metadata    map[string]interface{}  `json:"metadata"`
	OutputPath  string                  `json:"output_path,omitempty"`
}

// generateFromPromptFile generates evidence from a prompt file
func (e *EvidenceGeneratorTool) generateFromPromptFile(ctx context.Context, promptFile string, tools []string, format string, outputDir string) (*EvidenceGenerationResult, error) {
	// Check if prompt file exists and is safe to read
	if !e.isPathSafe(promptFile) {
		return nil, fmt.Errorf("unsafe path: %s", promptFile)
	}

	promptContent, err := os.ReadFile(promptFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read prompt file: %w", err)
	}

	e.logger.Info("Loaded prompt file",
		logger.Field{Key: "file", Value: promptFile},
		logger.Field{Key: "size", Value: len(promptContent)})

	// Try to extract task reference from prompt content
	taskID := e.extractTaskIDFromPrompt(string(promptContent))

	// Generate evidence using the prompt
	result := &EvidenceGenerationResult{
		TaskID:      taskID,
		Content:     "",
		Format:      format,
		GeneratedAt: time.Now(),
		Sources:     []models.EvidenceSource{},
		Metadata: map[string]interface{}{
			"prompt_file": promptFile,
			"source_type": "prompt_file",
		},
	}

	// Coordinate sub-tools to gather evidence
	sources, synthesis, err := e.coordinateSubTools(ctx, taskID, string(promptContent), tools)
	if err != nil {
		return nil, fmt.Errorf("failed to coordinate sub-tools: %w", err)
	}

	result.Sources = sources
	result.Synthesis = synthesis

	// Generate final evidence content
	content, err := e.generateFinalEvidence(ctx, taskID, string(promptContent), sources, synthesis, format)
	if err != nil {
		return nil, fmt.Errorf("failed to generate final evidence: %w", err)
	}

	result.Content = content

	// Save to output directory if specified
	if outputDir != "" {
		outputPath, err := e.saveEvidenceToFile(result, outputDir)
		if err != nil {
			e.logger.Warn("Failed to save evidence to file",
				logger.Field{Key: "error", Value: err})
		} else {
			result.OutputPath = outputPath
		}
	}

	return result, nil
}

// generateFromTaskRef generates evidence from a task reference
func (e *EvidenceGeneratorTool) generateFromTaskRef(ctx context.Context, taskRef string, tools []string, format string, outputDir string) (*EvidenceGenerationResult, error) {
	// Resolve task ID from reference
	taskID, err := e.resolveTaskID(ctx, taskRef)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve task ID: %w", err)
	}

	// Get task details
	task, err := e.dataService.GetEvidenceTask(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get evidence task: %w", err)
	}

	e.logger.Info("Generating evidence for task",
		logger.Field{Key: "task_id", Value: taskID},
		logger.Field{Key: "task_name", Value: task.Name},
		logger.Field{Key: "reference_id", Value: task.ReferenceID})

	// Generate evidence directly without using evidence service to avoid circular dependency
	result := &EvidenceGenerationResult{
		TaskID:      taskID,
		Format:      format,
		GeneratedAt: time.Now(),
		Sources:     []models.EvidenceSource{},
		Metadata: map[string]interface{}{
			"task_ref":    taskRef,
			"source_type": "task_ref",
			"task_name":   task.Name,
			"framework":   task.Framework,
			"priority":    task.Priority,
		},
	}

	// Create prompt from task details
	prompt := fmt.Sprintf("Generate evidence for task: %s\nDescription: %s\nFramework: %s\nPriority: %s",
		task.Name, task.Description, task.Framework, task.Priority)

	// Coordinate sub-tools to gather evidence
	sources, synthesis, err := e.coordinateSubTools(ctx, taskID, prompt, tools)
	if err != nil {
		return nil, fmt.Errorf("failed to coordinate sub-tools: %w", err)
	}

	result.Sources = sources
	result.Synthesis = synthesis

	// Generate final evidence content
	content, err := e.generateFinalEvidence(ctx, taskID, prompt, sources, synthesis, format)
	if err != nil {
		return nil, fmt.Errorf("failed to generate final evidence: %w", err)
	}

	result.Content = content

	// Save to output directory if specified
	if outputDir != "" {
		outputPath, err := e.saveEvidenceToFile(result, outputDir)
		if err != nil {
			e.logger.Warn("Failed to save evidence to file",
				logger.Field{Key: "error", Value: err})
		} else {
			result.OutputPath = outputPath
		}
	}

	return result, nil
}

// coordinateSubTools coordinates execution of sub-tools to gather evidence
func (e *EvidenceGeneratorTool) coordinateSubTools(ctx context.Context, taskID int, prompt string, tools []string) ([]models.EvidenceSource, string, error) {
	sources := make([]models.EvidenceSource, 0, len(tools))
	var synthesisBuilder strings.Builder

	synthesisBuilder.WriteString(fmt.Sprintf("Evidence synthesis for task %d using tools: %s\n\n", taskID, strings.Join(tools, ", ")))

	// Capture git hash for terraform repository if configured
	var terraformGitHash string
	if e.config.Evidence.Terraform.RepoPath != "" {
		hash, err := utils.GetCommitHash(e.config.Evidence.Terraform.RepoPath)
		if err != nil {
			e.logger.Warn("Failed to get terraform repository git hash",
				logger.Field{Key: "repo_path", Value: e.config.Evidence.Terraform.RepoPath},
				logger.Field{Key: "error", Value: err})
		} else {
			terraformGitHash = hash
			e.logger.Debug("Captured terraform repository git hash",
				logger.Field{Key: "hash", Value: hash})
		}
	}

	for _, toolName := range tools {
		e.logger.Debug("Coordinating sub-tool",
			logger.Field{Key: "tool", Value: toolName},
			logger.Field{Key: "task_id", Value: taskID})

		// Pass git hash to terraform tools
		source, err := e.executeTool(ctx, toolName, taskID, prompt, terraformGitHash)
		if err != nil {
			e.logger.Warn("Sub-tool execution failed",
				logger.Field{Key: "tool", Value: toolName},
				logger.Field{Key: "error", Value: err})

			// Continue with other tools but add error info
			source = models.EvidenceSource{
				Type:        toolName,
				Resource:    fmt.Sprintf("Tool %s failed: %s", toolName, err.Error()),
				Content:     "",
				ExtractedAt: time.Now(),
				Metadata: map[string]interface{}{
					"error":   err.Error(),
					"success": false,
				},
			}
		} else {
			synthesisBuilder.WriteString(fmt.Sprintf("### %s Results\n", cases.Title(language.English).String(toolName)))
			synthesisBuilder.WriteString(fmt.Sprintf("- Type: %s\n", source.Type))
			synthesisBuilder.WriteString(fmt.Sprintf("- Resource: %s\n", source.Resource))
			if len(source.Content) > 200 {
				synthesisBuilder.WriteString(fmt.Sprintf("- Content preview: %s...\n", source.Content[:200]))
			} else {
				synthesisBuilder.WriteString(fmt.Sprintf("- Content: %s\n", source.Content))
			}
			synthesisBuilder.WriteString("\n")
		}

		sources = append(sources, source)
	}

	synthesisBuilder.WriteString(fmt.Sprintf("Generated %d source(s) at %s", len(sources), time.Now().Format(time.RFC3339)))

	return sources, synthesisBuilder.String(), nil
}

// executeTool executes a specific sub-tool
func (e *EvidenceGeneratorTool) executeTool(ctx context.Context, toolName string, taskID int, prompt string, terraformGitHash string) (models.EvidenceSource, error) {
	// Get tool from registry
	tool, err := GetTool(toolName)
	if err != nil {
		return models.EvidenceSource{}, fmt.Errorf("tool %s not found: %w", toolName, err)
	}

	// Build parameters for the tool
	params := map[string]interface{}{
		"task_id":             taskID,
		"prompt":              prompt,
		"evidence_generation": true, // Flag to indicate we're in evidence generation context
	}

	// Add git hash for terraform tools
	if strings.Contains(strings.ToLower(toolName), "terraform") && terraformGitHash != "" {
		params["terraform_git_hash"] = terraformGitHash
	}

	// Execute tool
	result, source, err := tool.Execute(ctx, params)
	if err != nil {
		return models.EvidenceSource{}, fmt.Errorf("tool execution failed: %w", err)
	}

	// Return source if provided, otherwise create from result
	if source != nil {
		return *source, nil
	}

	return models.EvidenceSource{
		Type:        toolName,
		Resource:    fmt.Sprintf("Results from %s tool", toolName),
		Content:     result,
		ExtractedAt: time.Now(),
		Metadata: map[string]interface{}{
			"task_id":      taskID,
			"generated_at": time.Now(),
		},
	}, nil
}

// generateFinalEvidence creates the final evidence content from all sources
func (e *EvidenceGeneratorTool) generateFinalEvidence(ctx context.Context, taskID int, prompt string, sources []models.EvidenceSource, synthesis string, format string) (string, error) {
	var contentBuilder strings.Builder

	switch format {
	case "markdown":
		contentBuilder.WriteString(fmt.Sprintf("# Evidence for Task %d\n\n", taskID))
		contentBuilder.WriteString(fmt.Sprintf("**Generated:** %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

		if prompt != "" {
			contentBuilder.WriteString("## Requirements\n\n")
			contentBuilder.WriteString(fmt.Sprintf("```\n%s\n```\n\n", prompt))
		}

		contentBuilder.WriteString("## Evidence Sources\n\n")
		for i, source := range sources {
			contentBuilder.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, cases.Title(language.English).String(source.Type)))
			contentBuilder.WriteString(fmt.Sprintf("**Resource:** %s\n\n", source.Resource))

			if source.Content != "" {
				contentBuilder.WriteString("**Content:**\n\n")
				contentBuilder.WriteString(fmt.Sprintf("```\n%s\n```\n\n", source.Content))
			}
		}

		contentBuilder.WriteString("## Synthesis\n\n")
		contentBuilder.WriteString(synthesis)

	case "json":
		evidenceData := map[string]interface{}{
			"task_id":      taskID,
			"generated_at": time.Now(),
			"prompt":       prompt,
			"sources":      sources,
			"synthesis":    synthesis,
			"format":       format,
		}

		jsonData, err := json.MarshalIndent(evidenceData, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal JSON evidence: %w", err)
		}
		contentBuilder.Write(jsonData)

	case "csv":
		contentBuilder.WriteString("Source,Type,Description,Content,Metadata\n")
		for i, source := range sources {
			metadataJSON, _ := json.Marshal(source.Metadata)
			contentBuilder.WriteString(fmt.Sprintf("%d,%s,\"%s\",\"%s\",\"%s\"\n",
				i+1,
				source.Type,
				strings.ReplaceAll(source.Resource, "\"", "\"\""),
				strings.ReplaceAll(source.Content, "\"", "\"\""),
				strings.ReplaceAll(string(metadataJSON), "\"", "\"\""),
			))
		}

	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}

	return contentBuilder.String(), nil
}

// Utility methods

func (e *EvidenceGeneratorTool) isPathSafe(path string) bool {
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

func (e *EvidenceGeneratorTool) extractTaskIDFromPrompt(prompt string) int {
	// Try to extract task ID from prompt content
	// Look for patterns like "ET1", "task 123", "ID: 123", etc.

	// Simple regex-like extraction - could be enhanced
	lines := strings.Split(prompt, "\n")
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), "task") || strings.Contains(strings.ToLower(line), "et") {
			words := strings.Fields(line)
			for _, word := range words {
				if strings.HasPrefix(strings.ToUpper(word), "ET") {
					if numStr := strings.TrimPrefix(strings.ToUpper(word), "ET"); numStr != "" {
						if num, err := strconv.Atoi(numStr); err == nil {
							return 327991 + num // Convert ET reference to internal ID
						}
					}
				}
			}
		}
	}

	return 0 // Unknown task ID
}

func (e *EvidenceGeneratorTool) resolveTaskID(ctx context.Context, taskRef string) (int, error) {
	// First try to parse as integer
	if taskID, err := strconv.Atoi(taskRef); err == nil {
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

func (e *EvidenceGeneratorTool) saveEvidenceToFile(result *EvidenceGenerationResult, outputDir string) (string, error) {
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate filename
	ext := ".txt"
	switch result.Format {
	case "markdown":
		ext = ".md"
	case "json":
		ext = ".json"
	case "csv":
		ext = ".csv"
	}

	filename := fmt.Sprintf("evidence_task_%d_%s%s",
		result.TaskID,
		result.GeneratedAt.Format("20060102_150405"),
		ext)

	filePath := filepath.Join(outputDir, filename)

	// Write content to file
	if err := os.WriteFile(filePath, []byte(result.Content), 0644); err != nil {
		return "", fmt.Errorf("failed to write evidence file: %w", err)
	}

	e.logger.Info("Evidence saved to file",
		logger.Field{Key: "path", Value: filePath},
		logger.Field{Key: "size", Value: len(result.Content)})

	return filePath, nil
}
