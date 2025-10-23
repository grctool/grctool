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
	"strings"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"gopkg.in/yaml.v3"
)

// StorageReadTool provides path-safe file reading with format auto-detection
type StorageReadTool struct {
	config *config.Config
	logger logger.Logger
}

// NewStorageReadTool creates a new storage read tool
func NewStorageReadTool(cfg *config.Config, log logger.Logger) *StorageReadTool {
	return &StorageReadTool{
		config: cfg,
		logger: log,
	}
}

// GetClaudeToolDefinition returns the tool definition for Claude
func (s *StorageReadTool) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        "storage-read",
		Description: "Path-safe file reading with format auto-detection and metadata preservation.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "File path to read (must be within data directory)",
				},
				"format": map[string]interface{}{
					"type":        "string",
					"description": "Force format detection (optional)",
					"enum":        []string{"json", "yaml", "markdown", "text"},
				},
				"with_metadata": map[string]interface{}{
					"type":        "boolean",
					"description": "Include file metadata in response",
					"default":     false,
				},
			},
			"required": []string{"path"},
		},
	}
}

// Execute runs the storage read tool
func (s *StorageReadTool) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	s.logger.Info("Starting storage read operation",
		logger.Field{Key: "params", Value: params})

	// Parse parameters
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return "", nil, fmt.Errorf("path parameter is required")
	}

	format, _ := params["format"].(string)
	withMetadata, _ := params["with_metadata"].(bool)

	// Validate and clean path
	cleanPath, err := s.validateAndCleanPath(path)
	if err != nil {
		return "", nil, fmt.Errorf("invalid path: %w", err)
	}

	// Check if file exists
	fileInfo, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, fmt.Errorf("file does not exist: %s", cleanPath)
		}
		return "", nil, fmt.Errorf("failed to access file: %w", err)
	}

	// Check if it's a regular file
	if !fileInfo.Mode().IsRegular() {
		return "", nil, fmt.Errorf("path is not a regular file: %s", cleanPath)
	}

	// Read file content
	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Detect format if not specified
	if format == "" {
		format = s.detectFormat(cleanPath, string(content))
	}

	// Parse content based on format
	parsedContent, err := s.parseContent(string(content), format)
	if err != nil {
		s.logger.Warn("Failed to parse content, returning raw content",
			logger.Field{Key: "format", Value: format},
			logger.Field{Key: "error", Value: err})
		// Return raw content if parsing fails
		parsedContent = string(content)
	}

	// Create evidence source metadata
	source := &models.EvidenceSource{
		Type:        "storage-read",
		Resource:    fmt.Sprintf("File content from %s (format: %s)", filepath.Base(cleanPath), format),
		Content:     parsedContent,
		ExtractedAt: time.Now(),
		Metadata: map[string]interface{}{
			"file_path":       cleanPath,
			"format":          format,
			"file_size":       len(content),
			"detected_format": format,
		},
	}

	// Build response
	response := map[string]interface{}{
		"success":   true,
		"content":   parsedContent,
		"format":    format,
		"file_path": cleanPath,
		"file_size": len(content),
		"read_at":   time.Now(),
	}

	// Add metadata if requested
	if withMetadata {
		metadata := map[string]interface{}{
			"mode":         fileInfo.Mode().String(),
			"size":         fileInfo.Size(),
			"modified":     fileInfo.ModTime(),
			"is_directory": fileInfo.IsDir(),
			"permissions":  fileInfo.Mode().Perm().String(),
		}
		response["metadata"] = metadata
		source.Metadata["file_metadata"] = metadata
	}

	responseJSON, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", source, fmt.Errorf("failed to marshal response: %w", err)
	}

	s.logger.Info("Storage read completed",
		logger.Field{Key: "path", Value: cleanPath},
		logger.Field{Key: "format", Value: format},
		logger.Field{Key: "size", Value: len(content)})

	return string(responseJSON), source, nil
}

// Name returns the tool name
func (s *StorageReadTool) Name() string {
	return "storage-read"
}

// Description returns the tool description
func (s *StorageReadTool) Description() string {
	return "Path-safe file reading with format auto-detection and metadata preservation"
}

// Version returns the tool version
func (s *StorageReadTool) Version() string {
	return "1.0.0"
}

// Category returns the tool category
func (s *StorageReadTool) Category() string {
	return "storage"
}

// StorageWriteTool provides path-safe file writing with format handling
type StorageWriteTool struct {
	config *config.Config
	logger logger.Logger
}

// NewStorageWriteTool creates a new storage write tool
func NewStorageWriteTool(cfg *config.Config, log logger.Logger) *StorageWriteTool {
	return &StorageWriteTool{
		config: cfg,
		logger: log,
	}
}

// GetClaudeToolDefinition returns the tool definition for Claude
func (s *StorageWriteTool) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        "storage-write",
		Description: "Path-safe file writing with format handling and directory management.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "File path to write (must be within data directory)",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Content to write to file",
				},
				"format": map[string]interface{}{
					"type":        "string",
					"description": "Content format (auto-detected from extension if empty)",
					"enum":        []string{"json", "yaml", "markdown", "text"},
				},
				"create_dirs": map[string]interface{}{
					"type":        "boolean",
					"description": "Create parent directories if they don't exist",
					"default":     true,
				},
			},
			"required": []string{"path", "content"},
		},
	}
}

// Execute runs the storage write tool
func (s *StorageWriteTool) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	s.logger.Info("Starting storage write operation",
		logger.Field{Key: "params", Value: params})

	// Parse parameters
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return "", nil, fmt.Errorf("path parameter is required")
	}

	content, ok := params["content"].(string)
	if !ok {
		return "", nil, fmt.Errorf("content parameter is required")
	}

	format, _ := params["format"].(string)
	createDirs, _ := params["create_dirs"].(bool)
	if _, exists := params["create_dirs"]; !exists {
		createDirs = true // Default to true
	}

	// Validate and clean path
	cleanPath, err := s.validateAndCleanPath(path)
	if err != nil {
		return "", nil, fmt.Errorf("invalid path: %w", err)
	}

	// Detect format if not specified
	if format == "" {
		format = s.detectFormat(cleanPath, content)
	}

	// Format content based on detected format
	formattedContent, err := s.formatContent(content, format)
	if err != nil {
		s.logger.Warn("Failed to format content, using raw content",
			logger.Field{Key: "format", Value: format},
			logger.Field{Key: "error", Value: err})
		formattedContent = content
	}

	// Create parent directories if needed
	if createDirs {
		parentDir := filepath.Dir(cleanPath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return "", nil, fmt.Errorf("failed to create parent directories: %w", err)
		}
	}

	// Check if file exists for backup/overwrite handling
	var existedBefore bool
	var originalSize int64
	if fileInfo, err := os.Stat(cleanPath); err == nil {
		existedBefore = true
		originalSize = fileInfo.Size()
	}

	// Write content to file
	if err := os.WriteFile(cleanPath, []byte(formattedContent), 0644); err != nil {
		return "", nil, fmt.Errorf("failed to write file: %w", err)
	}

	// Verify write was successful
	if verifyInfo, err := os.Stat(cleanPath); err != nil {
		return "", nil, fmt.Errorf("failed to verify written file: %w", err)
	} else {
		if verifyInfo.Size() != int64(len(formattedContent)) {
			return "", nil, fmt.Errorf("file size mismatch after write")
		}
	}

	// Create evidence source metadata
	source := &models.EvidenceSource{
		Type:        "storage-write",
		Resource:    fmt.Sprintf("File written to %s (format: %s)", filepath.Base(cleanPath), format),
		Content:     fmt.Sprintf("Written %d bytes to %s", len(formattedContent), cleanPath),
		ExtractedAt: time.Now(),
		Metadata: map[string]interface{}{
			"file_path":           cleanPath,
			"format":              format,
			"content_size":        len(formattedContent),
			"existed_before":      existedBefore,
			"original_size":       originalSize,
			"directories_created": createDirs,
		},
	}

	// Build response
	response := map[string]interface{}{
		"success":        true,
		"file_path":      cleanPath,
		"format":         format,
		"content_size":   len(formattedContent),
		"existed_before": existedBefore,
		"written_at":     time.Now(),
	}

	if existedBefore {
		response["original_size"] = originalSize
		response["size_change"] = int64(len(formattedContent)) - originalSize
	}

	responseJSON, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", source, fmt.Errorf("failed to marshal response: %w", err)
	}

	s.logger.Info("Storage write completed",
		logger.Field{Key: "path", Value: cleanPath},
		logger.Field{Key: "format", Value: format},
		logger.Field{Key: "size", Value: len(formattedContent)},
		logger.Field{Key: "existed_before", Value: existedBefore})

	return string(responseJSON), source, nil
}

// Name returns the tool name
func (s *StorageWriteTool) Name() string {
	return "storage-write"
}

// Description returns the tool description
func (s *StorageWriteTool) Description() string {
	return "Path-safe file writing with format handling and directory management"
}

// Version returns the tool version
func (s *StorageWriteTool) Version() string {
	return "1.0.0"
}

// Category returns the tool category
func (s *StorageWriteTool) Category() string {
	return "storage"
}

// Shared utility methods

func (s *StorageReadTool) validateAndCleanPath(path string) (string, error) {
	return validateAndCleanPath(path, s.config.Storage.DataDir)
}

func (s *StorageWriteTool) validateAndCleanPath(path string) (string, error) {
	return validateAndCleanPath(path, s.config.Storage.DataDir)
}

// validateAndCleanPath validates and cleans a file path to ensure it's safe
func validateAndCleanPath(path, dataDir string) (string, error) {
	// Clean the path to resolve any relative components
	cleanPath := filepath.Clean(path)

	// Check for path traversal attempts
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("path traversal detected: %s", path)
	}

	// If path is not absolute, make it relative to data directory
	if !filepath.IsAbs(cleanPath) {
		cleanPath = filepath.Join(dataDir, cleanPath)
	}

	// Get absolute paths for comparison
	absCleanPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	absDataDir, err := filepath.Abs(dataDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute data directory: %w", err)
	}

	// Ensure the path is within the data directory
	if !strings.HasPrefix(absCleanPath, absDataDir) {
		return "", fmt.Errorf("path outside data directory: %s", absCleanPath)
	}

	return absCleanPath, nil
}

// detectFormat attempts to detect the file format from extension and content
func (s *StorageReadTool) detectFormat(filePath, content string) string {
	return detectFormat(filePath, content)
}

func (s *StorageWriteTool) detectFormat(filePath, content string) string {
	return detectFormat(filePath, content)
}

func detectFormat(filePath, content string) string {
	// First check file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".md", ".markdown":
		return "markdown"
	case ".txt":
		return "text"
	}

	// Try to detect from content
	trimmedContent := strings.TrimSpace(content)

	// JSON detection
	if (strings.HasPrefix(trimmedContent, "{") && strings.HasSuffix(trimmedContent, "}")) ||
		(strings.HasPrefix(trimmedContent, "[") && strings.HasSuffix(trimmedContent, "]")) {
		return "json"
	}

	// YAML detection (basic)
	if strings.Contains(content, ":") &&
		(strings.Contains(content, "\n- ") || strings.HasPrefix(trimmedContent, "---")) {
		return "yaml"
	}

	// Markdown detection
	if strings.Contains(content, "#") || strings.Contains(content, "**") || strings.Contains(content, "*") {
		return "markdown"
	}

	// Default to text
	return "text"
}

// parseContent parses content based on its format
func (s *StorageReadTool) parseContent(content, format string) (string, error) {
	switch format {
	case "json":
		// Validate JSON and pretty-print
		var jsonData interface{}
		if err := json.Unmarshal([]byte(content), &jsonData); err != nil {
			return content, fmt.Errorf("invalid JSON: %w", err)
		}
		prettyJSON, err := json.MarshalIndent(jsonData, "", "  ")
		if err != nil {
			return content, fmt.Errorf("failed to format JSON: %w", err)
		}
		return string(prettyJSON), nil

	case "yaml":
		// Validate YAML
		var yamlData interface{}
		if err := yaml.Unmarshal([]byte(content), &yamlData); err != nil {
			return content, fmt.Errorf("invalid YAML: %w", err)
		}
		return content, nil

	case "markdown", "text":
		// Return as-is for markdown and text
		return content, nil

	default:
		return content, nil
	}
}

// formatContent formats content based on its format
func (s *StorageWriteTool) formatContent(content, format string) (string, error) {
	switch format {
	case "json":
		// Try to parse and reformat JSON
		var jsonData interface{}
		if err := json.Unmarshal([]byte(content), &jsonData); err != nil {
			// If it's not valid JSON, return as-is
			return content, nil
		}
		prettyJSON, err := json.MarshalIndent(jsonData, "", "  ")
		if err != nil {
			return content, fmt.Errorf("failed to format JSON: %w", err)
		}
		return string(prettyJSON), nil

	case "yaml":
		// Try to parse and reformat YAML
		var yamlData interface{}
		if err := yaml.Unmarshal([]byte(content), &yamlData); err != nil {
			// If it's not valid YAML, return as-is
			return content, nil
		}
		formattedYAML, err := yaml.Marshal(yamlData)
		if err != nil {
			return content, fmt.Errorf("failed to format YAML: %w", err)
		}
		return string(formattedYAML), nil

	case "markdown":
		// Basic markdown formatting - ensure proper line endings
		lines := strings.Split(content, "\n")
		var formattedLines []string
		for _, line := range lines {
			formattedLines = append(formattedLines, strings.TrimRight(line, " \t"))
		}
		return strings.Join(formattedLines, "\n"), nil

	case "text":
		// Ensure consistent line endings for text
		return strings.ReplaceAll(content, "\r\n", "\n"), nil

	default:
		return content, nil
	}
}
