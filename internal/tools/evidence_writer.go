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
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/naming"
	"github.com/grctool/grctool/internal/storage"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

// Sentinel errors following AGENTS.md conventions (line 33)
var (
	ErrTaskNotFound      = errors.New("evidence task not found")
	ErrInvalidFormat     = errors.New("invalid evidence format")
	ErrMissingParameter  = errors.New("required parameter missing")
	ErrInvalidStatus     = errors.New("invalid evidence status")
	ErrInvalidInterval   = errors.New("invalid collection interval")
	ErrDirectoryCreation = errors.New("failed to create directory")
	ErrFileWrite         = errors.New("failed to write file")
)

// EvidenceWriterTool provides evidence writing with window management
type EvidenceWriterTool struct {
	config      *config.Config
	logger      logger.Logger
	dataStore   interfaces.LocalDataStore
	validator   *Validator
	planManager *CollectionPlanManager
}

// NewEvidenceWriterTool creates a new evidence writer tool
func NewEvidenceWriterTool(cfg *config.Config, log logger.Logger) Tool {
	// Initialize local data store
	paths := cfg.Storage.Paths.WithDefaults().ResolveRelativeTo(cfg.Storage.DataDir)
	localDataStore, err := storage.NewLocalDataStore(cfg.Storage.DataDir, paths)
	if err != nil {
		log.Error("Failed to initialize local data store for evidence writer tool",
			logger.Field{Key: "error", Value: err})
		return nil
	}

	// Initialize validator
	validator := NewValidator(cfg.Storage.DataDir)

	// Initialize plan manager
	planManager := NewCollectionPlanManager(cfg, log)

	return &EvidenceWriterTool{
		config:      cfg,
		logger:      log,
		dataStore:   localDataStore,
		validator:   validator,
		planManager: planManager,
	}
}

// Name returns the tool name
func (ewt *EvidenceWriterTool) Name() string {
	return "evidence-writer"
}

// Description returns the tool description
func (ewt *EvidenceWriterTool) Description() string {
	return "Write evidence files with window management and collection planning"
}

// GetClaudeToolDefinition returns the tool definition for Claude
func (ewt *EvidenceWriterTool) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        ewt.Name(),
		Description: ewt.Description(),
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"task_ref": map[string]interface{}{
					"type":        "string",
					"description": "Evidence task reference (ET1, ET-101, or numeric ID)",
					"examples":    []string{"ET1", "ET-101", "327992"},
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Evidence document title",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Evidence content in markdown or CSV format",
				},
				"format": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"markdown", "csv"},
					"default":     "markdown",
					"description": "Output format for the evidence file",
				},
				"source_type": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"terraform", "github", "google_docs", "manual", "screenshot", "api", "database"},
					"description": "Type of evidence source",
				},
				"source_location": map[string]interface{}{
					"type":        "string",
					"description": "Source location (file path, URL, or description)",
				},
				"controls": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Control references this evidence addresses",
					"default":     []string{},
				},
				"summary": map[string]interface{}{
					"type":        "string",
					"description": "Brief summary for the collection plan",
				},
				"reasoning": map[string]interface{}{
					"type":        "string",
					"description": "Why this evidence is relevant and what it demonstrates",
				},
				"status": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"complete", "partial", "pending"},
					"default":     "complete",
					"description": "Evidence collection status",
				},
				"update_plan": map[string]interface{}{
					"type":        "boolean",
					"default":     true,
					"description": "Whether to update the collection plan",
				},
			},
			"required": []string{"task_ref", "title", "content", "format"},
		},
	}
}

// calculateFileChecksum computes the SHA256 checksum of a file
func calculateFileChecksum(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("reading file for checksum: %w", err)
	}
	hash := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", hash), nil
}

// Execute runs the evidence writer tool
func (ewt *EvidenceWriterTool) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	// AGENTS.md compliance: Add operation timing (line 44)
	start := time.Now()
	defer func() {
		ewt.logger.Info("Evidence write operation completed",
			logger.Field{Key: "component", Value: "evidence_writer"},
			logger.Field{Key: "operation", Value: "execute"},
			logger.Field{Key: "duration_ms", Value: time.Since(start).Milliseconds()})
	}()

	// AGENTS.md compliance: Check context cancellation (line 38)
	select {
	case <-ctx.Done():
		return "", nil, fmt.Errorf("evidence write operation cancelled: %w", ctx.Err())
	default:
	}

	// Parse and validate parameters with proper error wrapping (line 31)
	taskRef, ok := params["task_ref"].(string)
	if !ok || taskRef == "" {
		return "", nil, fmt.Errorf("validating parameters: %w: task_ref", ErrMissingParameter)
	}

	title, ok := params["title"].(string)
	if !ok || title == "" {
		return "", nil, fmt.Errorf("validating parameters: %w: title", ErrMissingParameter)
	}

	content, ok := params["content"].(string)
	if !ok || content == "" {
		return "", nil, fmt.Errorf("validating parameters: %w: content", ErrMissingParameter)
	}

	format, ok := params["format"].(string)
	if !ok {
		format = "markdown"
	}

	// Enhanced logging with key parameters (AGENTS.md compliance)
	ewt.logger.Info("Processing evidence write request",
		logger.Field{Key: "component", Value: "evidence_writer"},
		logger.Field{Key: "operation", Value: "execute"},
		logger.Field{Key: "task_ref", Value: taskRef},
		logger.Field{Key: "title", Value: title},
		logger.Field{Key: "format", Value: format})

	// Validate format parameter
	validFormats := map[string]bool{
		"markdown": true,
		"csv":      true,
	}
	if !validFormats[format] {
		return "", nil, fmt.Errorf("validating format parameter '%s': %w", format, ErrInvalidFormat)
	}

	sourceType, _ := params["source_type"].(string)
	sourceLocation, _ := params["source_location"].(string)
	summary, _ := params["summary"].(string)
	reasoning, _ := params["reasoning"].(string)
	status, _ := params["status"].(string)
	if status == "" {
		status = "complete"
	}

	// Validate status parameter
	validStatuses := map[string]bool{
		"complete": true,
		"partial":  true,
		"pending":  true,
	}
	if !validStatuses[status] {
		return "", nil, fmt.Errorf("validating status parameter '%s': %w", status, ErrInvalidStatus)
	}

	updatePlan := true
	if up, ok := params["update_plan"].(bool); ok {
		updatePlan = up
	}

	// Parse controls array
	controls := []string{}
	if controlsParam, ok := params["controls"].([]interface{}); ok {
		for _, c := range controlsParam {
			if controlStr, ok := c.(string); ok {
				controls = append(controls, controlStr)
			}
		}
	}

	// Check context cancellation before resolving task
	if err := ctx.Err(); err != nil {
		return "", nil, fmt.Errorf("operation cancelled before task resolution: %w", err)
	}

	// Resolve task reference to get task details
	task, err := ewt.resolveTaskReference(taskRef)
	if err != nil {
		return "", nil, fmt.Errorf("failed to resolve task reference '%s': %w", taskRef, err)
	}

	// Calculate evidence window
	window := CalculateEvidenceWindow(task.CollectionInterval, time.Now())

	// Check context cancellation before directory operations
	if err := ctx.Err(); err != nil {
		return "", nil, fmt.Errorf("operation cancelled before directory creation: %w", err)
	}

	// Create evidence directory structure (hybrid approach - working files at root)
	taskDirName := naming.GetEvidenceTaskDirName(task.ReferenceID, task.Name)
	windowDir := filepath.Join(ewt.config.Storage.DataDir, "evidence", taskDirName, window)
	evidenceDir := windowDir // Write directly to root

	if err := os.MkdirAll(evidenceDir, 0755); err != nil {
		return "", nil, fmt.Errorf("creating evidence directory '%s': %w: %w", evidenceDir, ErrDirectoryCreation, err)
	}

	// Load or create collection plan (at window level)
	planPath := filepath.Join(windowDir, "collection_plan.md")
	plan, err := ewt.planManager.LoadOrCreatePlan(task, window, planPath)
	if err != nil {
		return "", nil, fmt.Errorf("loading collection plan from '%s': %w", planPath, err)
	}

	// Generate evidence filename with proper numbering
	evidenceIndex := len(plan.Entries) + 1
	var filename string
	if format == "csv" {
		filename = GenerateEvidenceFilename(evidenceIndex, title) + ".csv"
	} else {
		filename = GenerateEvidenceFilename(evidenceIndex, title) + ".md"
	}

	evidencePath := filepath.Join(evidenceDir, filename)

	// Check context cancellation before file write
	if err := ctx.Err(); err != nil {
		return "", nil, fmt.Errorf("operation cancelled before file write: %w", err)
	}

	// Write evidence file
	if err := ewt.writeEvidenceFile(evidencePath, content, format); err != nil {
		return "", nil, fmt.Errorf("writing evidence file '%s': %w: %w", evidencePath, ErrFileWrite, err)
	}

	// Calculate file checksum for metadata
	checksum, err := calculateFileChecksum(evidencePath)
	if err != nil {
		ewt.logger.Warn("Failed to calculate file checksum",
			logger.Field{Key: "error", Value: err},
			logger.Field{Key: "file", Value: evidencePath})
		checksum = "" // Continue even if checksum fails
	}

	// Get file size
	fileInfo, err := os.Stat(evidencePath)
	sizeBytes := int64(0)
	if err == nil {
		sizeBytes = fileInfo.Size()
	} else {
		ewt.logger.Warn("Failed to get file size",
			logger.Field{Key: "error", Value: err},
			logger.Field{Key: "file", Value: evidencePath})
	}

	// Create file metadata entry
	fileMetadata := models.FileMetadata{
		Path:        filename, // Just the filename, not full path
		Checksum:    checksum,
		SizeBytes:   sizeBytes,
		GeneratedAt: time.Now(),
	}

	// Determine tools used from parameters
	toolsUsed := []string{}
	if sourceType != "" {
		toolsUsed = []string{sourceType}
	}

	// Write generation metadata
	if err := ewt.writeGenerationMetadata(
		evidenceDir,
		task,
		window,
		[]models.FileMetadata{fileMetadata},
		"grctool-cli", // Generated by CLI directly
		toolsUsed,
	); err != nil {
		ewt.logger.Warn("Failed to write generation metadata",
			logger.Field{Key: "error", Value: err},
			logger.Field{Key: "evidence_dir", Value: evidenceDir})
		// Don't fail the operation for metadata errors
	}

	// Update collection plan if requested
	if updatePlan {
		// Create evidence entry
		entry := EvidenceEntry{
			Filename:          filename,
			Title:             title,
			Source:            ewt.formatSource(sourceType, sourceLocation),
			ControlsSatisfied: controls,
			Status:            status,
			CollectedAt:       time.Now(),
			Summary:           summary,
		}

		// Add entry to plan
		ewt.planManager.AddEvidenceEntry(plan, entry)

		// Update strategy and reasoning if provided
		if reasoning != "" {
			ewt.planManager.UpdateStrategy(plan, plan.Strategy, reasoning)
		}

		// Save updated plan
		if err := ewt.planManager.SavePlan(plan, planPath); err != nil {
			ewt.logger.Warn("Failed to save collection plan",
				logger.Field{Key: "error", Value: err})
			// Don't fail the operation for plan save errors
		}
	}

	// Create evidence source for return
	evidenceSource := &models.EvidenceSource{
		Type:        sourceType,
		Resource:    sourceLocation,
		Content:     content,
		Relevance:   1.0, // Full relevance since explicitly provided
		ExtractedAt: time.Now(),
		Metadata: map[string]interface{}{
			"title":              title,
			"format":             format,
			"status":             status,
			"controls_satisfied": controls,
			"window":             window,
			"evidence_path":      evidencePath,
		},
	}

	result := fmt.Sprintf("Evidence written successfully:\n"+
		"- File: %s\n"+
		"- Format: %s\n"+
		"- Status: %s\n"+
		"- Window: %s\n"+
		"- Collection Plan: %s",
		evidencePath, format, status, window,
		FormatCompleteness(plan.Completeness)+" complete")

	ewt.logger.Info("Evidence written successfully",
		logger.Field{Key: "evidence_path", Value: evidencePath},
		logger.Field{Key: "format", Value: format},
		logger.Field{Key: "window", Value: window},
		logger.Field{Key: "plan_completeness", Value: FormatCompleteness(plan.Completeness)})

	return result, evidenceSource, nil
}

// resolveTaskReference resolves a task reference to a task object
func (ewt *EvidenceWriterTool) resolveTaskReference(taskRef string) (*domain.EvidenceTask, error) {
	// Use validator to resolve the task reference
	if ewt.validator == nil {
		return nil, fmt.Errorf("validator not initialized")
	}

	// Try to validate the task reference - this will resolve it
	validation, err := ewt.validator.ValidateTaskReference(taskRef)
	if err != nil {
		return nil, fmt.Errorf("error validating task reference: %w", err)
	}
	if !validation.Valid {
		return nil, fmt.Errorf("invalid task reference: %s", validation.Errors)
	}

	// Load the task using the validated reference
	// For numeric IDs, convert to string for GetEvidenceTask call
	var taskIDStr string
	if taskID, parseErr := strconv.Atoi(taskRef); parseErr == nil {
		taskIDStr = strconv.Itoa(taskID)
	} else {
		// For reference IDs like ET1, we need to convert to numeric ID
		// Use the normalized ID from validation if available
		if normalized, exists := validation.Normalized["task_id"]; exists {
			taskIDStr = normalized
		} else {
			return nil, fmt.Errorf("could not resolve task reference to numeric ID: %s", taskRef)
		}
	}

	task, err := ewt.dataStore.GetEvidenceTask(taskIDStr)
	if err != nil {
		return nil, fmt.Errorf("failed to load task: %w", err)
	}

	if task == nil {
		return nil, fmt.Errorf("task not found: %s", taskRef)
	}

	return task, nil
}

// writeEvidenceFile writes the evidence content to the specified file
func (ewt *EvidenceWriterTool) writeEvidenceFile(filePath, content, format string) error {
	// For markdown files, ensure proper formatting
	if format == "markdown" && !strings.HasPrefix(content, "#") {
		// Add a basic header if content doesn't start with one
		content = "# Evidence\n\n" + content
	}

	// Write the file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing file to disk: %w", err)
	}

	return nil
}

// writeGenerationMetadata creates a .generation/metadata.yaml file with generation details
// windowDir should point to the window root directory where evidence files are written
func (ewt *EvidenceWriterTool) writeGenerationMetadata(
	windowDir string,
	task *domain.EvidenceTask,
	window string,
	files []models.FileMetadata,
	generatedBy string,
	toolsUsed []string,
) error {
	// Create .generation directory inside window root
	metadataDir := filepath.Join(windowDir, ".generation")
	if err := os.MkdirAll(metadataDir, 0755); err != nil {
		return fmt.Errorf("creating metadata directory: %w", err)
	}

	// Build metadata structure
	metadata := models.GenerationMetadata{
		GeneratedAt:      time.Now(),
		GeneratedBy:      generatedBy,
		GenerationMethod: "tool_coordination",
		TaskID:           task.ID,
		TaskRef:          task.ReferenceID,
		Window:           window,
		ToolsUsed:        toolsUsed,
		FilesGenerated:   files,
		Status:           "generated",
	}

	// Serialize to YAML
	yamlData, err := yaml.Marshal(&metadata)
	if err != nil {
		return fmt.Errorf("marshaling metadata to YAML: %w", err)
	}

	// Write metadata file
	metadataPath := filepath.Join(metadataDir, "metadata.yaml")
	if err := os.WriteFile(metadataPath, yamlData, 0644); err != nil {
		return fmt.Errorf("writing metadata file: %w", err)
	}

	ewt.logger.Info("Generation metadata written successfully",
		logger.Field{Key: "metadata_path", Value: metadataPath},
		logger.Field{Key: "files_tracked", Value: len(files)})

	return nil
}

// formatSource creates a human-readable source description
func (ewt *EvidenceWriterTool) formatSource(sourceType, sourceLocation string) string {
	if sourceType == "" && sourceLocation == "" {
		return "Manual entry"
	}

	if sourceType == "" {
		return sourceLocation
	}

	if sourceLocation == "" {
		return cases.Title(language.English).String(sourceType)
	}

	return fmt.Sprintf("%s - %s", cases.Title(language.English).String(sourceType), sourceLocation)
}

// Helper functions for evidence window management

// CalculateEvidenceWindow determines the window for a given interval and date
// Pure function - no side effects, easily testable
func CalculateEvidenceWindow(interval string, date time.Time) string {
	switch strings.ToLower(interval) {
	case "year", "annual", "annually":
		return fmt.Sprintf("%d", date.Year())
	case "quarter", "quarterly":
		quarter := (date.Month()-1)/3 + 1
		return fmt.Sprintf("%d-Q%d", date.Year(), quarter)
	case "month", "monthly":
		return fmt.Sprintf("%d-%02d", date.Year(), date.Month())
	case "six_month", "semi-annual", "semiannual":
		half := 1
		if date.Month() > 6 {
			half = 2
		}
		return fmt.Sprintf("%d-H%d", date.Year(), half)
	default:
		return fmt.Sprintf("%d", date.Year())
	}
}

// GenerateEvidenceFilename creates a numbered filename
// Pure function for deterministic file naming
func GenerateEvidenceFilename(index int, title string) string {
	// Sanitize title for filesystem - keep alphanumeric, spaces, hyphens, underscores
	safe := regexp.MustCompile(`[^a-zA-Z0-9\s\-_]`).ReplaceAllString(title, "")

	// Replace spaces and hyphens with underscores, convert to lowercase
	safe = strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(safe, " ", "_"), "-", "_"))

	// Remove multiple consecutive underscores
	safe = regexp.MustCompile(`_{2,}`).ReplaceAllString(safe, "_")

	// Trim underscores from start and end
	safe = strings.Trim(safe, "_")

	// Ensure we have something
	if safe == "" {
		safe = "evidence"
	}

	return fmt.Sprintf("%02d_%s", index, safe)
}

