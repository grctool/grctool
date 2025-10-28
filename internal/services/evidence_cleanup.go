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
	"strings"

	"github.com/grctool/grctool/internal/logger"
)

// EvidenceCleanupService handles organizing evidence from flat structure to subfolder structure
type EvidenceCleanupService struct {
	evidenceDir string
	scanner     EvidenceScanner
	logger      logger.Logger
}

// CleanupResult represents the result of cleaning up a single window
type CleanupResult struct {
	TaskRef      string                   `json:"task_ref"`
	Window       string                   `json:"window"`
	WasFlatStructure bool                `json:"was_flat_structure"`
	FilesOrganized   map[string]int      `json:"files_organized"` // subfolder -> count
	MetadataMoved    []string            `json:"metadata_moved"`  // directories moved
	Errors           []string            `json:"errors"`
}

// CleanupSummary represents the summary of cleaning up multiple tasks
type CleanupSummary struct {
	TotalTasks      int                       `json:"total_tasks"`
	TotalWindows    int                       `json:"total_windows"`
	WindowsCleaned  int                       `json:"windows_cleaned"`
	FilesOrganized  int                       `json:"files_organized"`
	Results         []CleanupResult           `json:"results"`
	Errors          []string                  `json:"errors"`
}

// NewEvidenceCleanupService creates a new evidence cleanup service
func NewEvidenceCleanupService(evidenceDir string, scanner EvidenceScanner, log logger.Logger) *EvidenceCleanupService {
	return &EvidenceCleanupService{
		evidenceDir: evidenceDir,
		scanner:     scanner,
		logger:      log.WithComponent("evidence_cleanup"),
	}
}

// CleanupTask organizes evidence for a specific task and window
func (s *EvidenceCleanupService) CleanupTask(ctx context.Context, taskRef string, window string, dryRun bool) (*CleanupResult, error) {
	s.logger.Info("Cleaning up evidence task",
		logger.String("task_ref", taskRef),
		logger.String("window", window),
		logger.Field{Key: "dry_run", Value: dryRun})

	result := &CleanupResult{
		TaskRef:        taskRef,
		Window:         window,
		FilesOrganized: make(map[string]int),
		MetadataMoved:  []string{},
		Errors:         []string{},
	}

	// Find task directory
	taskDir, err := s.findTaskDirectory(taskRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find task directory: %w", err)
	}

	if taskDir == "" {
		return nil, fmt.Errorf("task directory not found for %s", taskRef)
	}

	// Construct window directory path
	windowDir := filepath.Join(taskDir, window)
	if _, err := os.Stat(windowDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("window directory not found: %s", window)
	}

	// Check if this is a flat structure (files at window level, no subfolders)
	hasFlatStructure, err := s.isFlatStructure(windowDir)
	if err != nil {
		return nil, fmt.Errorf("failed to check structure: %w", err)
	}

	result.WasFlatStructure = hasFlatStructure

	if !hasFlatStructure {
		s.logger.Info("Window already uses subfolder structure, skipping",
			logger.String("task_ref", taskRef),
			logger.String("window", window))
		return result, nil
	}

	// Organize files into subfolders
	if err := s.organizeFiles(ctx, windowDir, result, dryRun); err != nil {
		return nil, fmt.Errorf("failed to organize files: %w", err)
	}

	s.logger.Info("Cleanup complete",
		logger.String("task_ref", taskRef),
		logger.String("window", window),
		logger.Int("files_organized", s.getTotalFilesOrganized(result)))

	return result, nil
}

// CleanupAll organizes evidence for all tasks
func (s *EvidenceCleanupService) CleanupAll(ctx context.Context, dryRun bool) (*CleanupSummary, error) {
	s.logger.Info("Starting cleanup of all evidence",
		logger.Field{Key: "dry_run", Value: dryRun})

	summary := &CleanupSummary{
		Results: []CleanupResult{},
		Errors:  []string{},
	}

	// Scan all evidence tasks
	taskStates, err := s.scanner.ScanAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to scan evidence: %w", err)
	}

	summary.TotalTasks = len(taskStates)

	// Clean up each task's windows
	for taskRef, taskState := range taskStates {
		// Check context cancellation
		if ctx.Err() != nil {
			return summary, ctx.Err()
		}

		for window := range taskState.Windows {
			summary.TotalWindows++

			result, err := s.CleanupTask(ctx, taskRef, window, dryRun)
			if err != nil {
				errMsg := fmt.Sprintf("Failed to clean up %s/%s: %v", taskRef, window, err)
				summary.Errors = append(summary.Errors, errMsg)
				s.logger.Warn("Cleanup failed",
					logger.String("task_ref", taskRef),
					logger.String("window", window),
					logger.Error(err))
				continue
			}

			if result.WasFlatStructure {
				summary.WindowsCleaned++
				summary.FilesOrganized += s.getTotalFilesOrganized(result)
			}

			summary.Results = append(summary.Results, *result)
		}
	}

	s.logger.Info("Cleanup complete",
		logger.Int("total_tasks", summary.TotalTasks),
		logger.Int("windows_cleaned", summary.WindowsCleaned),
		logger.Int("files_organized", summary.FilesOrganized))

	return summary, nil
}

// isFlatStructure checks if a window directory uses flat structure
func (s *EvidenceCleanupService) isFlatStructure(windowDir string) (bool, error) {
	// Check if subdirectories wip/, ready/, submitted/ exist
	subfolders := []string{"wip", "ready", "submitted"}
	hasSubfolders := false

	for _, subfolder := range subfolders {
		subfolderPath := filepath.Join(windowDir, subfolder)
		if stat, err := os.Stat(subfolderPath); err == nil && stat.IsDir() {
			hasSubfolders = true
			break
		}
	}

	if hasSubfolders {
		return false, nil // Already has subfolders
	}

	// Check if there are any evidence files at window level
	entries, err := os.ReadDir(windowDir)
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		// Skip directories (including hidden ones like .generation, .submission, .context)
		if entry.IsDir() {
			continue
		}

		// Skip special files
		if entry.Name() == "collection_plan.md" || entry.Name() == "collection_plan_metadata.yaml" {
			continue
		}

		// Found a regular file at window level - this is flat structure
		return true, nil
	}

	return false, nil // No files to organize
}

// organizeFiles organizes files from flat structure into subfolders
func (s *EvidenceCleanupService) organizeFiles(ctx context.Context, windowDir string, result *CleanupResult, dryRun bool) error {
	// Determine target subfolder based on metadata
	targetSubfolder, err := s.determineTargetSubfolder(windowDir)
	if err != nil {
		return fmt.Errorf("failed to determine target subfolder: %w", err)
	}

	s.logger.Debug("Target subfolder determined",
		logger.String("window_dir", windowDir),
		logger.String("target", targetSubfolder))

	// Create target directory
	targetDir := filepath.Join(windowDir, targetSubfolder)
	if !dryRun {
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return fmt.Errorf("failed to create target directory: %w", err)
		}
	}

	// Move evidence files
	entries, err := os.ReadDir(windowDir)
	if err != nil {
		return fmt.Errorf("failed to read window directory: %w", err)
	}

	for _, entry := range entries {
		// Check context cancellation
		if ctx.Err() != nil {
			return ctx.Err()
		}

		sourcePath := filepath.Join(windowDir, entry.Name())

		// Handle directories (metadata)
		if entry.IsDir() {
			// Move metadata directories if appropriate
			if strings.HasPrefix(entry.Name(), ".") {
				// Metadata directory
				if s.shouldMoveMetadata(entry.Name(), targetSubfolder) {
					destPath := filepath.Join(targetDir, entry.Name())
					s.logger.Debug("Moving metadata directory",
						logger.String("from", sourcePath),
						logger.String("to", destPath),
						logger.Field{Key: "dry_run", Value: dryRun})

					if !dryRun {
						if err := os.Rename(sourcePath, destPath); err != nil {
							errMsg := fmt.Sprintf("Failed to move %s: %v", entry.Name(), err)
							result.Errors = append(result.Errors, errMsg)
							continue
						}
					}
					result.MetadataMoved = append(result.MetadataMoved, entry.Name())
				}
			}
			continue
		}

		// Skip special files
		if entry.Name() == "collection_plan.md" || entry.Name() == "collection_plan_metadata.yaml" {
			continue
		}

		// Move evidence file
		destPath := filepath.Join(targetDir, entry.Name())
		s.logger.Debug("Moving evidence file",
			logger.String("from", sourcePath),
			logger.String("to", destPath),
			logger.Field{Key: "dry_run", Value: dryRun})

		if !dryRun {
			if err := os.Rename(sourcePath, destPath); err != nil {
				errMsg := fmt.Sprintf("Failed to move %s: %v", entry.Name(), err)
				result.Errors = append(result.Errors, errMsg)
				continue
			}
		}

		result.FilesOrganized[targetSubfolder]++
	}

	return nil
}

// determineTargetSubfolder determines which subfolder files should go to based on metadata
func (s *EvidenceCleanupService) determineTargetSubfolder(windowDir string) (string, error) {
	// Check for .submission/submission.yaml
	submissionPath := filepath.Join(windowDir, ".submission", "submission.yaml")
	if _, err := os.Stat(submissionPath); err == nil {
		return "submitted", nil
	}

	// Check for .validation/validation.yaml
	validationPath := filepath.Join(windowDir, ".validation", "validation.yaml")
	if _, err := os.Stat(validationPath); err == nil {
		return "ready", nil
	}

	// Check for .generation/metadata.yaml
	generationPath := filepath.Join(windowDir, ".generation", "metadata.yaml")
	if _, err := os.Stat(generationPath); err == nil {
		return "wip", nil
	}

	// Default to wip if no metadata found
	return "wip", nil
}

// shouldMoveMetadata determines if a metadata directory should be moved
func (s *EvidenceCleanupService) shouldMoveMetadata(metadataDir string, targetSubfolder string) bool {
	switch metadataDir {
	case ".generation":
		// Always move with files
		return true
	case ".validation":
		// Only move if going to ready or submitted
		return targetSubfolder == "ready" || targetSubfolder == "submitted"
	case ".submission":
		// Only move if going to submitted
		return targetSubfolder == "submitted"
	case ".context":
		// Never move - stays at window level (shared)
		return false
	default:
		// Unknown metadata directory - don't move
		return false
	}
}

// findTaskDirectory finds the directory for a given task reference
func (s *EvidenceCleanupService) findTaskDirectory(taskRef string) (string, error) {
	entries, err := os.ReadDir(s.evidenceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
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

	return "", nil
}

// getTotalFilesOrganized calculates total files organized from result
func (s *EvidenceCleanupService) getTotalFilesOrganized(result *CleanupResult) int {
	total := 0
	for _, count := range result.FilesOrganized {
		total += count
	}
	return total
}
