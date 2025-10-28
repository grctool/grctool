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

package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/naming"
	"github.com/grctool/grctool/internal/services/conversion"
	"gopkg.in/yaml.v3"
)

const (
	submissionMetadataDir = ".submission"
	submissionFilename    = "submission.yaml"
	validationFilename    = "validation.yaml"
	historyFilename       = "history.yaml"
	batchStorageDir       = "submissions"
)

// SaveSubmission saves submission metadata for a task window
// For backward compatibility, saves at window level (use SaveSubmissionToSubfolder for new structure)
func (us *Storage) SaveSubmission(submission *models.EvidenceSubmission) error {
	if submission == nil {
		return fmt.Errorf("submission cannot be nil")
	}

	// Get evidence directory for this task/window
	evidenceDir := us.getEvidenceWindowDir(submission.TaskRef, submission.Window)
	submissionDir := filepath.Join(evidenceDir, submissionMetadataDir)

	// Create .submission directory if it doesn't exist
	if err := os.MkdirAll(submissionDir, 0755); err != nil {
		return fmt.Errorf("failed to create submission directory: %w", err)
	}

	// Save submission metadata
	submissionPath := filepath.Join(submissionDir, submissionFilename)
	data, err := yaml.Marshal(submission)
	if err != nil {
		return fmt.Errorf("failed to marshal submission: %w", err)
	}

	if err := os.WriteFile(submissionPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write submission file: %w", err)
	}

	return nil
}

// SaveSubmissionToSubfolder saves submission metadata to a specific subfolder (ready/submitted)
func (us *Storage) SaveSubmissionToSubfolder(submission *models.EvidenceSubmission, subfolder string) error {
	if submission == nil {
		return fmt.Errorf("submission cannot be nil")
	}

	// Get evidence subfolder directory for this task/window/subfolder
	evidenceDir := us.getEvidenceSubfolderDir(submission.TaskRef, submission.Window, subfolder)
	submissionDir := filepath.Join(evidenceDir, submissionMetadataDir)

	// Create .submission directory if it doesn't exist
	if err := os.MkdirAll(submissionDir, 0755); err != nil {
		return fmt.Errorf("failed to create submission directory: %w", err)
	}

	// Save submission metadata
	submissionPath := filepath.Join(submissionDir, submissionFilename)
	data, err := yaml.Marshal(submission)
	if err != nil {
		return fmt.Errorf("failed to marshal submission: %w", err)
	}

	if err := os.WriteFile(submissionPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write submission file: %w", err)
	}

	return nil
}

// LoadSubmission loads submission metadata for a task window
func (us *Storage) LoadSubmission(taskRef, window string) (*models.EvidenceSubmission, error) {
	evidenceDir := us.getEvidenceWindowDir(taskRef, window)
	submissionPath := filepath.Join(evidenceDir, submissionMetadataDir, submissionFilename)

	if _, err := os.Stat(submissionPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("submission not found for %s in window %s", taskRef, window)
	}

	data, err := os.ReadFile(submissionPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read submission file: %w", err)
	}

	var submission models.EvidenceSubmission
	if err := yaml.Unmarshal(data, &submission); err != nil {
		return nil, fmt.Errorf("failed to unmarshal submission: %w", err)
	}

	return &submission, nil
}

// SaveValidationResult saves validation results for a task window
// For backward compatibility, saves at window level (use SaveValidationResultToSubfolder for new structure)
func (us *Storage) SaveValidationResult(taskRef, window string, result *models.ValidationResult) error {
	if result == nil {
		return fmt.Errorf("validation result cannot be nil")
	}

	evidenceDir := us.getEvidenceWindowDir(taskRef, window)
	submissionDir := filepath.Join(evidenceDir, submissionMetadataDir)

	// Create .submission directory if it doesn't exist
	if err := os.MkdirAll(submissionDir, 0755); err != nil {
		return fmt.Errorf("failed to create submission directory: %w", err)
	}

	validationPath := filepath.Join(submissionDir, validationFilename)
	data, err := yaml.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal validation result: %w", err)
	}

	if err := os.WriteFile(validationPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write validation file: %w", err)
	}

	return nil
}

// SaveValidationResultToSubfolder saves validation results to a specific subfolder (typically "ready")
func (us *Storage) SaveValidationResultToSubfolder(taskRef, window string, result *models.ValidationResult, subfolder string) error {
	if result == nil {
		return fmt.Errorf("validation result cannot be nil")
	}

	evidenceDir := us.getEvidenceSubfolderDir(taskRef, window, subfolder)
	validationDir := filepath.Join(evidenceDir, ".validation")

	// Create .validation directory if it doesn't exist
	if err := os.MkdirAll(validationDir, 0755); err != nil {
		return fmt.Errorf("failed to create validation directory: %w", err)
	}

	validationPath := filepath.Join(validationDir, validationFilename)
	data, err := yaml.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal validation result: %w", err)
	}

	if err := os.WriteFile(validationPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write validation file: %w", err)
	}

	return nil
}

// LoadValidationResult loads validation results for a task window
func (us *Storage) LoadValidationResult(taskRef, window string) (*models.ValidationResult, error) {
	evidenceDir := us.getEvidenceWindowDir(taskRef, window)
	validationPath := filepath.Join(evidenceDir, submissionMetadataDir, validationFilename)

	if _, err := os.Stat(validationPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("validation result not found for %s in window %s", taskRef, window)
	}

	data, err := os.ReadFile(validationPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read validation file: %w", err)
	}

	var result models.ValidationResult
	if err := yaml.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal validation result: %w", err)
	}

	return &result, nil
}

// AddSubmissionHistory adds an entry to the submission history
func (us *Storage) AddSubmissionHistory(taskRef, window string, entry models.SubmissionHistoryEntry) error {
	// Load existing history or create new
	history, err := us.LoadSubmissionHistory(taskRef, window)
	if err != nil {
		// Create new history if it doesn't exist
		history = &models.SubmissionHistory{
			TaskRef: taskRef,
			Window:  window,
			Entries: []models.SubmissionHistoryEntry{},
		}
	}

	// Add new entry
	history.Entries = append(history.Entries, entry)

	// Sort by submitted time (most recent first)
	sort.Slice(history.Entries, func(i, j int) bool {
		return history.Entries[i].SubmittedAt.After(history.Entries[j].SubmittedAt)
	})

	// Save history
	return us.SaveSubmissionHistory(history)
}

// AddSubmissionHistoryToSubfolder adds an entry to the submission history in a specific subfolder
func (us *Storage) AddSubmissionHistoryToSubfolder(taskRef, window string, entry models.SubmissionHistoryEntry, subfolder string) error {
	// Load existing history or create new
	// Note: LoadSubmissionHistory doesn't have a subfolder variant, so history starts fresh
	history := &models.SubmissionHistory{
		TaskRef: taskRef,
		Window:  window,
		Entries: []models.SubmissionHistoryEntry{entry},
	}

	// Sort by submitted time (most recent first)
	sort.Slice(history.Entries, func(i, j int) bool {
		return history.Entries[i].SubmittedAt.After(history.Entries[j].SubmittedAt)
	})

	// Save history to subfolder
	return us.SaveSubmissionHistoryToSubfolder(history, subfolder)
}

// SaveSubmissionHistory saves the complete submission history
// For backward compatibility, saves at window level (use SaveSubmissionHistoryToSubfolder for new structure)
func (us *Storage) SaveSubmissionHistory(history *models.SubmissionHistory) error {
	if history == nil {
		return fmt.Errorf("history cannot be nil")
	}

	evidenceDir := us.getEvidenceWindowDir(history.TaskRef, history.Window)
	submissionDir := filepath.Join(evidenceDir, submissionMetadataDir)

	// Create .submission directory if it doesn't exist
	if err := os.MkdirAll(submissionDir, 0755); err != nil {
		return fmt.Errorf("failed to create submission directory: %w", err)
	}

	historyPath := filepath.Join(submissionDir, historyFilename)
	data, err := yaml.Marshal(history)
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	if err := os.WriteFile(historyPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write history file: %w", err)
	}

	return nil
}

// SaveSubmissionHistoryToSubfolder saves submission history to a specific subfolder (ready/submitted)
func (us *Storage) SaveSubmissionHistoryToSubfolder(history *models.SubmissionHistory, subfolder string) error {
	if history == nil {
		return fmt.Errorf("history cannot be nil")
	}

	evidenceDir := us.getEvidenceSubfolderDir(history.TaskRef, history.Window, subfolder)
	submissionDir := filepath.Join(evidenceDir, submissionMetadataDir)

	// Create .submission directory if it doesn't exist
	if err := os.MkdirAll(submissionDir, 0755); err != nil {
		return fmt.Errorf("failed to create submission directory: %w", err)
	}

	historyPath := filepath.Join(submissionDir, historyFilename)
	data, err := yaml.Marshal(history)
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	if err := os.WriteFile(historyPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write history file: %w", err)
	}

	return nil
}

// LoadSubmissionHistory loads submission history for a task window
func (us *Storage) LoadSubmissionHistory(taskRef, window string) (*models.SubmissionHistory, error) {
	evidenceDir := us.getEvidenceWindowDir(taskRef, window)
	historyPath := filepath.Join(evidenceDir, submissionMetadataDir, historyFilename)

	if _, err := os.Stat(historyPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("history not found for %s in window %s", taskRef, window)
	}

	data, err := os.ReadFile(historyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read history file: %w", err)
	}

	var history models.SubmissionHistory
	if err := yaml.Unmarshal(data, &history); err != nil {
		return nil, fmt.Errorf("failed to unmarshal history: %w", err)
	}

	return &history, nil
}

// SaveBatch saves a submission batch
func (us *Storage) SaveBatch(batch *models.SubmissionBatch) error {
	if batch == nil {
		return fmt.Errorf("batch cannot be nil")
	}

	// Get batch directory
	batchDir := filepath.Join(us.localDataStore.GetBaseDir(), batchStorageDir, batch.BatchID)

	// Create batch directory if it doesn't exist
	if err := os.MkdirAll(batchDir, 0755); err != nil {
		return fmt.Errorf("failed to create batch directory: %w", err)
	}

	// Save batch manifest
	manifestPath := filepath.Join(batchDir, "manifest.yaml")
	data, err := yaml.Marshal(batch)
	if err != nil {
		return fmt.Errorf("failed to marshal batch: %w", err)
	}

	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write batch manifest: %w", err)
	}

	return nil
}

// LoadBatch loads a submission batch by ID
func (us *Storage) LoadBatch(batchID string) (*models.SubmissionBatch, error) {
	batchDir := filepath.Join(us.localDataStore.GetBaseDir(), batchStorageDir, batchID)
	manifestPath := filepath.Join(batchDir, "manifest.yaml")

	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("batch not found: %s", batchID)
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read batch manifest: %w", err)
	}

	var batch models.SubmissionBatch
	if err := yaml.Unmarshal(data, &batch); err != nil {
		return nil, fmt.Errorf("failed to unmarshal batch: %w", err)
	}

	return &batch, nil
}

// ListBatches lists all submission batches
func (us *Storage) ListBatches() ([]models.SubmissionBatch, error) {
	batchesDir := filepath.Join(us.localDataStore.GetBaseDir(), batchStorageDir)

	// Check if batches directory exists
	if _, err := os.Stat(batchesDir); os.IsNotExist(err) {
		return []models.SubmissionBatch{}, nil
	}

	entries, err := os.ReadDir(batchesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read batches directory: %w", err)
	}

	var batches []models.SubmissionBatch
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		batch, err := us.LoadBatch(entry.Name())
		if err != nil {
			// Skip batches that can't be loaded
			continue
		}

		batches = append(batches, *batch)
	}

	// Sort by created time (most recent first)
	sort.Slice(batches, func(i, j int) bool {
		return batches[i].CreatedAt.After(batches[j].CreatedAt)
	})

	return batches, nil
}

// CalculateFileChecksum calculates SHA256 checksum for a file
func (us *Storage) CalculateFileChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to hash file: %w", err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// GetEvidenceFiles gets all evidence files for a task window with metadata
// This method scans the window directory for backward compatibility (pre-subfolder structure)
func (us *Storage) GetEvidenceFiles(taskRef, window string) ([]models.EvidenceFileRef, error) {
	evidenceDir := us.getEvidenceWindowDir(taskRef, window)

	if _, err := os.Stat(evidenceDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("evidence directory not found for %s in window %s", taskRef, window)
	}

	entries, err := os.ReadDir(evidenceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read evidence directory: %w", err)
	}

	var files []models.EvidenceFileRef
	for _, entry := range entries {
		// Skip directories and hidden files
		if entry.IsDir() || entry.Name()[0] == '.' {
			continue
		}

		// Skip non-evidence files (collection_plan, etc.)
		if entry.Name() == "collection_plan.md" || entry.Name() == "collection_plan_metadata.yaml" {
			continue
		}

		filePath := filepath.Join(evidenceDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Calculate checksum
		checksum, err := us.CalculateFileChecksum(filePath)
		if err != nil {
			checksum = ""
		}

		// Get relative path from data directory
		relPath, err := filepath.Rel(us.localDataStore.GetBaseDir(), filePath)
		if err != nil {
			relPath = filePath
		}

		files = append(files, models.EvidenceFileRef{
			Filename:       entry.Name(),
			RelativePath:   relPath,
			Title:          entry.Name(),
			SizeBytes:      info.Size(),
			ChecksumSHA256: checksum,
		})
	}

	return files, nil
}

// GetEvidenceFilesFromSubfolder gets all evidence files from a specific subfolder (wip/ready/submitted)
func (us *Storage) GetEvidenceFilesFromSubfolder(taskRef, window, subfolder string) ([]models.EvidenceFileRef, error) {
	evidenceDir := us.getEvidenceSubfolderDir(taskRef, window, subfolder)

	if _, err := os.Stat(evidenceDir); os.IsNotExist(err) {
		// Return empty list if subfolder doesn't exist (not an error - just no files yet)
		return []models.EvidenceFileRef{}, nil
	}

	entries, err := os.ReadDir(evidenceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read evidence directory: %w", err)
	}

	var files []models.EvidenceFileRef
	for _, entry := range entries {
		// Skip directories and hidden files
		if entry.IsDir() || entry.Name()[0] == '.' {
			continue
		}

		// Skip non-evidence files
		if entry.Name() == "collection_plan.md" || entry.Name() == "collection_plan_metadata.yaml" {
			continue
		}

		filePath := filepath.Join(evidenceDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Calculate checksum
		checksum, err := us.CalculateFileChecksum(filePath)
		if err != nil {
			checksum = ""
		}

		// Get relative path from data directory
		relPath, err := filepath.Rel(us.localDataStore.GetBaseDir(), filePath)
		if err != nil {
			relPath = filePath
		}

		files = append(files, models.EvidenceFileRef{
			Filename:       entry.Name(),
			RelativePath:   relPath,
			Title:          entry.Name(),
			SizeBytes:      info.Size(),
			ChecksumSHA256: checksum,
		})
	}

	return files, nil
}

// getEvidenceWindowDir returns the evidence directory path for a task/window
func (us *Storage) getEvidenceWindowDir(taskRef, window string) string {
	// Evidence directory pattern: evidence/ET-{num}_{name}/{window}/
	evidenceBase := filepath.Join(us.localDataStore.GetBaseDir(), "evidence")

	// Find the task directory
	entries, err := os.ReadDir(evidenceBase)
	if err != nil {
		// If evidence base doesn't exist, create path anyway
		return filepath.Join(evidenceBase, taskRef, window)
	}

	// Look for directory matching task reference using naming service
	for _, entry := range entries {
		if entry.IsDir() && naming.MatchesTaskRef(entry.Name(), taskRef) {
			return filepath.Join(evidenceBase, entry.Name(), window)
		}
	}

	// Default fallback
	return filepath.Join(evidenceBase, taskRef, window)
}

// getEvidenceSubfolderDir returns the evidence directory path for a task/window/subfolder
// subfolder can be "wip", "ready", or "submitted"
func (us *Storage) getEvidenceSubfolderDir(taskRef, window, subfolder string) string {
	windowDir := us.getEvidenceWindowDir(taskRef, window)
	return filepath.Join(windowDir, subfolder)
}

// InitializeSubmissionMetadata creates the .submission directory structure
func (us *Storage) InitializeSubmissionMetadata(taskRef, window string) error {
	evidenceDir := us.getEvidenceWindowDir(taskRef, window)
	submissionDir := filepath.Join(evidenceDir, submissionMetadataDir)

	// Create .submission directory
	if err := os.MkdirAll(submissionDir, 0755); err != nil {
		return fmt.Errorf("failed to create submission directory: %w", err)
	}

	// Initialize empty submission if it doesn't exist
	submissionPath := filepath.Join(submissionDir, submissionFilename)
	if _, err := os.Stat(submissionPath); os.IsNotExist(err) {
		submission := &models.EvidenceSubmission{
			TaskRef:          taskRef,
			Window:           window,
			Status:           "draft",
			CreatedAt:        time.Now(),
			ValidationStatus: "pending",
			EvidenceFiles:    []models.EvidenceFileRef{},
		}
		if err := us.SaveSubmission(submission); err != nil {
			return err
		}
	}

	return nil
}

// SubmissionExists checks if submission metadata exists for a task/window
func (us *Storage) SubmissionExists(taskRef, window string) bool {
	evidenceDir := us.getEvidenceWindowDir(taskRef, window)
	submissionPath := filepath.Join(evidenceDir, submissionMetadataDir, submissionFilename)
	_, err := os.Stat(submissionPath)
	return err == nil
}

// MoveEvidenceFiles moves evidence files from one subfolder to another
// Moves both the evidence files and their metadata directories (.generation/, .validation/, etc.)
// If moving to "ready", automatically converts markdown files to PDF
func (us *Storage) MoveEvidenceFiles(taskRef, window, fromSubfolder, toSubfolder string) error {
	fromDir := us.getEvidenceSubfolderDir(taskRef, window, fromSubfolder)
	toDir := us.getEvidenceSubfolderDir(taskRef, window, toSubfolder)

	// Check if source directory exists
	if _, err := os.Stat(fromDir); os.IsNotExist(err) {
		return fmt.Errorf("source directory does not exist: %s", fromDir)
	}

	// Create destination directory
	if err := os.MkdirAll(toDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Read all entries from source directory
	entries, err := os.ReadDir(fromDir)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	// Move each file and directory
	for _, entry := range entries {
		sourcePath := filepath.Join(fromDir, entry.Name())
		destPath := filepath.Join(toDir, entry.Name())

		// Rename (move) the file/directory
		if err := os.Rename(sourcePath, destPath); err != nil {
			return fmt.Errorf("failed to move %s: %w", entry.Name(), err)
		}
	}

	// If moving to "ready", convert markdown files to PDF
	if toSubfolder == "ready" {
		if err := us.convertMarkdownToPDF(taskRef, window); err != nil {
			// Log warning but don't fail the move operation
			// User can still submit .md files as fallback
			fmt.Printf("Warning: failed to convert markdown to PDF: %v\n", err)
		}
	}

	// Optionally remove the now-empty source directory
	// We'll keep it for now - the scanner will handle empty directories gracefully

	return nil
}

// convertMarkdownToPDF converts all markdown files in the ready/ folder to PDF
func (us *Storage) convertMarkdownToPDF(taskRef, window string) error {
	readyDir := us.getEvidenceSubfolderDir(taskRef, window, "ready")

	// Import conversion package (will be added in imports)
	converter := conversion.NewConverter()

	// Find all .md files in ready/
	entries, err := os.ReadDir(readyDir)
	if err != nil {
		return fmt.Errorf("failed to read ready directory: %w", err)
	}

	for _, entry := range entries {
		// Skip directories and non-markdown files
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}

		inputPath := filepath.Join(readyDir, entry.Name())
		outputPath := strings.TrimSuffix(inputPath, ".md") + ".pdf"

		// Skip if PDF already exists and is newer than markdown
		if pdfInfo, err := os.Stat(outputPath); err == nil {
			mdInfo, _ := os.Stat(inputPath)
			if pdfInfo.ModTime().After(mdInfo.ModTime()) {
				continue // PDF is up to date
			}
		}

		// Convert markdown to PDF
		opts := conversion.DefaultOptions()
		opts.Title = strings.TrimSuffix(entry.Name(), ".md")
		opts.Subject = fmt.Sprintf("Evidence for %s - %s", taskRef, window)

		if err := converter.ConvertMarkdownToPDF(inputPath, outputPath, opts); err != nil {
			// Log error but continue with other files
			fmt.Printf("Warning: failed to convert %s to PDF: %v\n", entry.Name(), err)
			continue
		}

		fmt.Printf("Converted %s → %s\n", entry.Name(), filepath.Base(outputPath))
	}

	return nil
}
