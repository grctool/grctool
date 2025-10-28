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

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"gopkg.in/yaml.v3"
)

// EvidenceScanner scans evidence directories and builds state models
type EvidenceScanner interface {
	// ScanAll scans all evidence directories and returns state for all tasks
	ScanAll(ctx context.Context) (map[string]*models.EvidenceTaskState, error)

	// ScanTask scans a specific task's evidence directory
	ScanTask(ctx context.Context, taskRef string) (*models.EvidenceTaskState, error)

	// ScanWindow scans a specific window within a task
	ScanWindow(ctx context.Context, taskRef string, window string) (*models.WindowState, error)
}

// Storage interface for loading task metadata
type Storage interface {
	GetEvidenceTask(ctx context.Context, taskID int) (*domain.EvidenceTask, error)
}

// evidenceScannerImpl implements the EvidenceScanner interface
type evidenceScannerImpl struct {
	evidenceDir string // Path to evidence directory (e.g., "data/evidence")
	storage     Storage
	logger      logger.Logger
}

// NewEvidenceScanner creates a new evidence scanner instance
func NewEvidenceScanner(evidenceDir string, storage Storage, log logger.Logger) EvidenceScanner {
	return &evidenceScannerImpl{
		evidenceDir: evidenceDir,
		storage:     storage,
		logger:      log,
	}
}

// ScanAll scans all evidence directories and returns state for all tasks
func (s *evidenceScannerImpl) ScanAll(ctx context.Context) (map[string]*models.EvidenceTaskState, error) {
	s.logger.Info("Scanning evidence directory",
		logger.Field{Key: "directory", Value: s.evidenceDir})

	// Check if evidence directory exists
	if _, err := os.Stat(s.evidenceDir); os.IsNotExist(err) {
		s.logger.Warn("Evidence directory does not exist",
			logger.Field{Key: "directory", Value: s.evidenceDir})
		return make(map[string]*models.EvidenceTaskState), nil
	}

	// Walk the evidence directory to find all task directories
	taskStates := make(map[string]*models.EvidenceTaskState)
	taskPattern := regexp.MustCompile(`^(ET-\d+)_`)

	entries, err := os.ReadDir(s.evidenceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read evidence directory: %w", err)
	}

	for _, entry := range entries {
		// Check context cancellation
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		if !entry.IsDir() {
			continue
		}

		// Extract task reference from directory name (e.g., "ET-0001_TaskName" -> "ET-0001")
		matches := taskPattern.FindStringSubmatch(entry.Name())
		if len(matches) < 2 {
			continue
		}

		taskRef := matches[1]
		s.logger.Debug("Found task directory",
			logger.Field{Key: "task_ref", Value: taskRef},
			logger.Field{Key: "directory", Value: entry.Name()})

		// Scan this task
		taskState, err := s.ScanTask(ctx, taskRef)
		if err != nil {
			s.logger.Warn("Failed to scan task",
				logger.Field{Key: "task_ref", Value: taskRef},
				logger.Field{Key: "error", Value: err})
			continue
		}

		taskStates[taskRef] = taskState
	}

	s.logger.Info("Evidence scan complete",
		logger.Field{Key: "tasks_found", Value: len(taskStates)})

	return taskStates, nil
}

// ScanTask scans a specific task's evidence directory
func (s *evidenceScannerImpl) ScanTask(ctx context.Context, taskRef string) (*models.EvidenceTaskState, error) {
	s.logger.Debug("Scanning task",
		logger.Field{Key: "task_ref", Value: taskRef})

	// Find task directory (may have different suffixes: ET-0001_Name, ET-0001_Different_Name, etc.)
	taskDir, err := s.findTaskDirectory(taskRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find task directory: %w", err)
	}

	if taskDir == "" {
		// Task directory doesn't exist, return empty state
		s.logger.Debug("Task directory not found",
			logger.Field{Key: "task_ref", Value: taskRef})
		return s.createEmptyTaskState(taskRef), nil
	}

	// Extract task ID from reference (ET-0001 -> 1)
	taskID := extractTaskIDFromRef(taskRef)

	// Try to load task from storage for metadata
	var task *domain.EvidenceTask
	if s.storage != nil && taskID > 0 {
		task, err = s.storage.GetEvidenceTask(ctx, taskID)
		if err != nil {
			s.logger.Warn("Failed to load task from storage",
				logger.Field{Key: "task_ref", Value: taskRef},
				logger.Field{Key: "task_id", Value: taskID},
				logger.Field{Key: "error", Value: err})
		}
	}

	// Scan all window subdirectories
	windows := make(map[string]models.WindowState)
	windowDirs, err := os.ReadDir(taskDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read task directory: %w", err)
	}

	for _, entry := range windowDirs {
		// Check context cancellation
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		if !entry.IsDir() {
			continue
		}

		// Skip hidden directories
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		// Check if this looks like a window directory (2025-Q4, 2025, 2025-10, 2025-H1)
		if !isWindowDirectory(entry.Name()) {
			continue
		}

		windowName := entry.Name()
		windowState, err := s.scanWindowDirectory(ctx, taskRef, windowName, filepath.Join(taskDir, windowName))
		if err != nil {
			s.logger.Warn("Failed to scan window",
				logger.Field{Key: "task_ref", Value: taskRef},
				logger.Field{Key: "window", Value: windowName},
				logger.Field{Key: "error", Value: err})
			continue
		}

		windows[windowName] = *windowState
	}

	// Build task state
	taskState := &models.EvidenceTaskState{
		TaskRef:       taskRef,
		TaskID:        taskID,
		TaskName:      extractTaskName(filepath.Base(taskDir)),
		Windows:       windows,
		LastScannedAt: time.Now(),
	}

	// Populate fields from loaded task if available
	if task != nil {
		taskState.TaskName = task.Name
		taskState.TugboatStatus = task.Status
		taskState.TugboatCompleted = task.Completed
		taskState.Framework = task.Framework
		// Note: LastSyncedAt would come from sync metadata, not implemented here
	}

	// Determine local state based on windows
	taskState.LocalState = models.DetermineLocalState(windows)

	// Find most recent generation and submission timestamps
	taskState.LastGeneratedAt, taskState.LastSubmittedAt = s.findLatestTimestamps(windows)

	// Determine automation level
	taskState.AutomationLevel = s.determineAutomationLevel(task, windows)
	taskState.ApplicableTools = s.detectApplicableTools(task, windows)

	s.logger.Debug("Task scan complete",
		logger.Field{Key: "task_ref", Value: taskRef},
		logger.Field{Key: "windows", Value: len(windows)},
		logger.Field{Key: "local_state", Value: taskState.LocalState})

	return taskState, nil
}

// ScanWindow scans a specific window within a task
func (s *evidenceScannerImpl) ScanWindow(ctx context.Context, taskRef string, window string) (*models.WindowState, error) {
	s.logger.Debug("Scanning window",
		logger.Field{Key: "task_ref", Value: taskRef},
		logger.Field{Key: "window", Value: window})

	// Find task directory
	taskDir, err := s.findTaskDirectory(taskRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find task directory: %w", err)
	}

	if taskDir == "" {
		return nil, fmt.Errorf("task directory not found for %s", taskRef)
	}

	windowDir := filepath.Join(taskDir, window)
	if _, err := os.Stat(windowDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("window directory not found: %s", window)
	}

	return s.scanWindowDirectory(ctx, taskRef, window, windowDir)
}

// scanWindowDirectory scans a single window directory and builds WindowState
// Supports both old (flat) and new (subfolder) structures
func (s *evidenceScannerImpl) scanWindowDirectory(ctx context.Context, taskRef string, window string, windowDir string) (*models.WindowState, error) {
	// Check if this uses the new subfolder structure (wip/, ready/, submitted/)
	subfolders := []string{"wip", "ready", "submitted"}
	hasSubfolders := false
	for _, subfolder := range subfolders {
		subfolderPath := filepath.Join(windowDir, subfolder)
		if _, err := os.Stat(subfolderPath); err == nil {
			hasSubfolders = true
			break
		}
	}

	if hasSubfolders {
		// New structure: scan each subfolder
		return s.scanSubfolderStructure(ctx, taskRef, window, windowDir)
	}

	// Old structure: scan window directory directly (for backward compatibility)
	return s.scanFlatStructure(ctx, taskRef, window, windowDir)
}

// scanSubfolderStructure scans the new subfolder-based structure
func (s *evidenceScannerImpl) scanSubfolderStructure(ctx context.Context, taskRef string, window string, windowDir string) (*models.WindowState, error) {
	windowState := &models.WindowState{
		Window: window,
	}

	var allFiles []models.FileState
	var totalBytes int64
	var oldestFile, newestFile *time.Time

	// Scan ready/ subfolder (highest priority - determines submission status)
	readyDir := filepath.Join(windowDir, "ready")
	if stat, err := os.Stat(readyDir); err == nil && stat.IsDir() {
		files, bytes, oldest, newest := s.scanFilesInDir(ctx, readyDir)
		allFiles = append(allFiles, files...)
		totalBytes += bytes
		updateTimestamps(&oldestFile, &newestFile, oldest, newest)

		// Check for validation metadata in ready/.validation/
		if validationMeta := s.readValidationMetadata(readyDir); validationMeta != nil {
			windowState.SubmissionStatus = "validated"
		}

		// Check for submission metadata in ready/.submission/ (submitted from ready/)
		if subMeta, err := s.readSubmissionMetadata(readyDir); err == nil && subMeta != nil {
			windowState.HasSubmissionMeta = true
			windowState.SubmissionStatus = subMeta.Status
			windowState.SubmittedAt = subMeta.SubmittedAt
			windowState.SubmissionID = subMeta.SubmissionID
		}

		// Check for generation metadata in ready/.generation/ (moved from wip/)
		if genMeta, err := s.readGenerationMetadata(readyDir); err == nil && genMeta != nil {
			windowState.HasGenerationMeta = true
			windowState.GenerationMethod = genMeta.GenerationMethod
			windowState.GeneratedAt = &genMeta.GeneratedAt
			windowState.GeneratedBy = genMeta.GeneratedBy
			windowState.ToolsUsed = genMeta.ToolsUsed
		}
	}

	// Scan submitted/ subfolder (downloaded from Tugboat)
	submittedDir := filepath.Join(windowDir, "submitted")
	if stat, err := os.Stat(submittedDir); err == nil && stat.IsDir() {
		files, bytes, oldest, newest := s.scanFilesInDir(ctx, submittedDir)
		allFiles = append(allFiles, files...)
		totalBytes += bytes
		updateTimestamps(&oldestFile, &newestFile, oldest, newest)

		// Check for submission metadata in submitted/.submission/ (synced from Tugboat)
		if subMeta, err := s.readSubmissionMetadata(submittedDir); err == nil && subMeta != nil {
			// Only override if we don't already have submission info from ready/
			if !windowState.HasSubmissionMeta {
				windowState.HasSubmissionMeta = true
				windowState.SubmissionStatus = subMeta.Status
				windowState.SubmittedAt = subMeta.SubmittedAt
				windowState.SubmissionID = subMeta.SubmissionID
			}
		}
	}

	// Scan wip/ subfolder (work in progress)
	wipDir := filepath.Join(windowDir, "wip")
	if stat, err := os.Stat(wipDir); err == nil && stat.IsDir() {
		files, bytes, oldest, newest := s.scanFilesInDir(ctx, wipDir)
		allFiles = append(allFiles, files...)
		totalBytes += bytes
		updateTimestamps(&oldestFile, &newestFile, oldest, newest)

		// Check for generation metadata in wip/.generation/
		if genMeta, err := s.readGenerationMetadata(wipDir); err == nil && genMeta != nil {
			// Only use wip generation metadata if we don't have it from ready/
			if !windowState.HasGenerationMeta {
				windowState.HasGenerationMeta = true
				windowState.GenerationMethod = genMeta.GenerationMethod
				windowState.GeneratedAt = &genMeta.GeneratedAt
				windowState.GeneratedBy = genMeta.GeneratedBy
				windowState.ToolsUsed = genMeta.ToolsUsed
			}
		}
	}

	windowState.FileCount = len(allFiles)
	windowState.TotalBytes = totalBytes
	windowState.OldestFile = oldestFile
	windowState.NewestFile = newestFile
	windowState.Files = allFiles

	return windowState, nil
}

// scanFlatStructure scans the old flat structure (backward compatibility)
func (s *evidenceScannerImpl) scanFlatStructure(ctx context.Context, taskRef string, window string, windowDir string) (*models.WindowState, error) {
	windowState := &models.WindowState{
		Window: window,
	}

	files, totalBytes, oldestFile, newestFile := s.scanFilesInDir(ctx, windowDir)
	windowState.FileCount = len(files)
	windowState.TotalBytes = totalBytes
	windowState.OldestFile = oldestFile
	windowState.NewestFile = newestFile
	windowState.Files = files

	// Read generation metadata if exists
	genMetadata, err := s.readGenerationMetadata(windowDir)
	if err == nil && genMetadata != nil {
		windowState.HasGenerationMeta = true
		windowState.GenerationMethod = genMetadata.GenerationMethod
		windowState.GeneratedAt = &genMetadata.GeneratedAt
		windowState.GeneratedBy = genMetadata.GeneratedBy
		windowState.ToolsUsed = genMetadata.ToolsUsed

		// Mark files as generated if they appear in generation metadata
		for i := range files {
			for _, genFile := range genMetadata.FilesGenerated {
				if files[i].Filename == filepath.Base(genFile.Path) {
					files[i].IsGenerated = true
					files[i].Checksum = genFile.Checksum
					break
				}
			}
		}
		windowState.Files = files
	}

	// Read submission metadata if exists
	subMetadata, err := s.readSubmissionMetadata(windowDir)
	if err == nil && subMetadata != nil {
		windowState.HasSubmissionMeta = true
		windowState.SubmissionStatus = subMetadata.Status
		windowState.SubmittedAt = subMetadata.SubmittedAt
		windowState.SubmissionID = subMetadata.SubmissionID
	}

	return windowState, nil
}

// scanFilesInDir scans files in a directory and returns file states and statistics
func (s *evidenceScannerImpl) scanFilesInDir(ctx context.Context, dir string) ([]models.FileState, int64, *time.Time, *time.Time) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, 0, nil, nil
	}

	var files []models.FileState
	var totalBytes int64
	var oldestFile, newestFile *time.Time

	for _, entry := range entries {
		// Check context cancellation
		if ctx.Err() != nil {
			return files, totalBytes, oldestFile, newestFile
		}

		// Skip directories (including .generation, .submission, .context, .validation)
		if entry.IsDir() {
			continue
		}

		// Get file info
		info, err := entry.Info()
		if err != nil {
			continue
		}

		modTime := info.ModTime()
		fileSize := info.Size()

		// Track oldest and newest files
		if oldestFile == nil || modTime.Before(*oldestFile) {
			oldestFile = &modTime
		}
		if newestFile == nil || modTime.After(*newestFile) {
			newestFile = &modTime
		}

		totalBytes += fileSize

		// Create file state
		fileState := models.FileState{
			Filename:   entry.Name(),
			SizeBytes:  fileSize,
			ModifiedAt: modTime,
		}

		files = append(files, fileState)
	}

	return files, totalBytes, oldestFile, newestFile
}

// updateTimestamps helper to update oldest and newest timestamps
func updateTimestamps(oldestFile, newestFile **time.Time, oldest, newest *time.Time) {
	if oldest != nil && (*oldestFile == nil || oldest.Before(**oldestFile)) {
		*oldestFile = oldest
	}
	if newest != nil && (*newestFile == nil || newest.After(**newestFile)) {
		*newestFile = newest
	}
}

// readGenerationMetadata reads .generation/metadata.yaml
func (s *evidenceScannerImpl) readGenerationMetadata(windowDir string) (*models.GenerationMetadata, error) {
	metadataPath := filepath.Join(windowDir, ".generation", "metadata.yaml")

	// Check if file exists
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		return nil, nil // Not an error, just no metadata
	}

	// Read and unmarshal YAML
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("reading generation metadata: %w", err)
	}

	var metadata models.GenerationMetadata
	if err := yaml.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("parsing generation metadata: %w", err)
	}

	return &metadata, nil
}

// readSubmissionMetadata reads .submission/submission.yaml
func (s *evidenceScannerImpl) readSubmissionMetadata(windowDir string) (*models.EvidenceSubmission, error) {
	metadataPath := filepath.Join(windowDir, ".submission", "submission.yaml")

	// Check if file exists
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		return nil, nil // Not an error, just no metadata
	}

	// Read and unmarshal YAML
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("reading submission metadata: %w", err)
	}

	var metadata models.EvidenceSubmission
	if err := yaml.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("parsing submission metadata: %w", err)
	}

	return &metadata, nil
}

// readValidationMetadata reads .validation/validation.yaml
func (s *evidenceScannerImpl) readValidationMetadata(windowDir string) *models.ValidationResult {
	metadataPath := filepath.Join(windowDir, ".validation", "validation.yaml")

	// Check if file exists
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		return nil // Not an error, just no metadata
	}

	// Read and unmarshal YAML
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil
	}

	var metadata models.ValidationResult
	if err := yaml.Unmarshal(data, &metadata); err != nil {
		return nil
	}

	return &metadata
}

// findTaskDirectory finds the directory for a given task reference
func (s *evidenceScannerImpl) findTaskDirectory(taskRef string) (string, error) {
	entries, err := os.ReadDir(s.evidenceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // Evidence directory doesn't exist
		}
		return "", err
	}

	// Look for directory starting with task reference (ET-0001_)
	prefix := taskRef + "_"
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), prefix) {
			return filepath.Join(s.evidenceDir, entry.Name()), nil
		}
	}

	return "", nil // Not found
}

// createEmptyTaskState creates an empty task state for a task with no evidence
func (s *evidenceScannerImpl) createEmptyTaskState(taskRef string) *models.EvidenceTaskState {
	taskID := extractTaskIDFromRef(taskRef)
	return &models.EvidenceTaskState{
		TaskRef:          taskRef,
		TaskID:           taskID,
		TaskName:         "",
		TugboatStatus:    "",
		TugboatCompleted: false,
		LocalState:       models.StateNoEvidence,
		Windows:          make(map[string]models.WindowState),
		AutomationLevel:  models.AutomationUnknown,
		ApplicableTools:  []string{},
		LastScannedAt:    time.Now(),
	}
}

// findLatestTimestamps finds the most recent generation and submission timestamps across windows
func (s *evidenceScannerImpl) findLatestTimestamps(windows map[string]models.WindowState) (*time.Time, *time.Time) {
	var latestGenerated, latestSubmitted *time.Time

	for _, window := range windows {
		if window.GeneratedAt != nil {
			if latestGenerated == nil || window.GeneratedAt.After(*latestGenerated) {
				latestGenerated = window.GeneratedAt
			}
		}

		if window.SubmittedAt != nil {
			if latestSubmitted == nil || window.SubmittedAt.After(*latestSubmitted) {
				latestSubmitted = window.SubmittedAt
			}
		}
	}

	return latestGenerated, latestSubmitted
}

// determineAutomationLevel determines the automation level based on task and windows
func (s *evidenceScannerImpl) determineAutomationLevel(task *domain.EvidenceTask, windows map[string]models.WindowState) models.AutomationCapability {
	// Collect all tools used across windows
	toolsUsed := make(map[string]bool)
	for _, window := range windows {
		for _, tool := range window.ToolsUsed {
			toolsUsed[tool] = true
		}
	}

	var toolsList []string
	for tool := range toolsUsed {
		toolsList = append(toolsList, tool)
	}

	// If we have task info, use it to determine automation capability
	if task != nil {
		return models.DetermineAutomationCapability(toolsList, task.Description)
	}

	// Fallback logic based on tools used
	if len(toolsList) == 0 {
		return models.AutomationUnknown
	}

	return models.DetermineAutomationCapability(toolsList, "")
}

// detectApplicableTools detects applicable tools based on task and existing evidence
func (s *evidenceScannerImpl) detectApplicableTools(task *domain.EvidenceTask, windows map[string]models.WindowState) []string {
	toolsMap := make(map[string]bool)

	// Collect tools from existing evidence
	for _, window := range windows {
		for _, tool := range window.ToolsUsed {
			toolsMap[tool] = true
		}
	}

	// If we have task info, detect additional applicable tools based on keywords
	if task != nil {
		taskText := strings.ToLower(task.Name + " " + task.Description)

		// GitHub tools
		if strings.Contains(taskText, "github") || strings.Contains(taskText, "repository") ||
			strings.Contains(taskText, "access") || strings.Contains(taskText, "permissions") {
			toolsMap["github-permissions"] = true
		}

		// Terraform tools
		if strings.Contains(taskText, "terraform") || strings.Contains(taskText, "infrastructure") ||
			strings.Contains(taskText, "iam") || strings.Contains(taskText, "security") {
			toolsMap["terraform-security-analyzer"] = true
		}

		// Google Workspace tools
		if strings.Contains(taskText, "google") || strings.Contains(taskText, "workspace") ||
			strings.Contains(taskText, "drive") || strings.Contains(taskText, "docs") {
			toolsMap["google-workspace"] = true
		}

		// Atmos tools
		if strings.Contains(taskText, "atmos") || strings.Contains(taskText, "stack") ||
			strings.Contains(taskText, "multi-environment") {
			toolsMap["atmos-stack-analyzer"] = true
		}
	}

	// Convert map to slice
	var tools []string
	for tool := range toolsMap {
		tools = append(tools, tool)
	}

	return tools
}

// Helper functions

// extractTaskIDFromRef extracts the numeric task ID from a reference (ET-0001 -> 1)
func extractTaskIDFromRef(taskRef string) int {
	re := regexp.MustCompile(`^ET-(\d+)$`)
	matches := re.FindStringSubmatch(taskRef)
	if len(matches) < 2 {
		return 0
	}

	var id int
	fmt.Sscanf(matches[1], "%d", &id)
	return id
}

// extractTaskName extracts task name from directory name (ET-0001_TaskName -> TaskName)
func extractTaskName(dirname string) string {
	re := regexp.MustCompile(`^ET-\d+_(.+)$`)
	matches := re.FindStringSubmatch(dirname)
	if len(matches) < 2 {
		return ""
	}

	// Replace underscores with spaces
	return strings.ReplaceAll(matches[1], "_", " ")
}

// isWindowDirectory checks if a directory name looks like a window (2025-Q4, 2025, 2025-10, 2025-H1)
func isWindowDirectory(name string) bool {
	// Year only: 2025
	if regexp.MustCompile(`^\d{4}$`).MatchString(name) {
		return true
	}

	// Quarterly: 2025-Q4
	if regexp.MustCompile(`^\d{4}-Q[1-4]$`).MatchString(name) {
		return true
	}

	// Monthly: 2025-01 to 2025-12
	if monthMatch := regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`).MatchString(name); monthMatch {
		return true
	}

	// Half-yearly: 2025-H1
	if regexp.MustCompile(`^\d{4}-H[1-2]$`).MatchString(name) {
		return true
	}

	return false
}
