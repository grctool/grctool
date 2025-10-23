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
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/utils"
)

// LocalDataStore implements offline-first data access without external dependencies
type LocalDataStore struct {
	baseDir           string
	fileStorage       interfaces.FileService
	filenameGenerator *utils.FilenameGenerator
	fallbackEnabled   bool
	dataSources       []string
	// Configured paths from StorageConfig
	policiesPath      string
	controlsPath      string
	evidenceTasksPath string
}

// GetBaseDir returns the base directory
func (lds *LocalDataStore) GetBaseDir() string {
	return lds.baseDir
}

// NewLocalDataStore creates a new LocalDataStore instance
func NewLocalDataStore(baseDir string, paths config.StoragePaths) (*LocalDataStore, error) {
	// Ensure base directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	// Create subdirectories for data organization using configured paths
	// Remove baseDir prefix if present (paths should be relative to baseDir for FileStorage)
	policiesPath := strings.TrimPrefix(paths.PoliciesJSON, baseDir+string(filepath.Separator))
	controlsPath := strings.TrimPrefix(paths.ControlsJSON, baseDir+string(filepath.Separator))
	evidenceTasksPath := strings.TrimPrefix(paths.EvidenceTasksJSON, baseDir+string(filepath.Separator))

	subdirs := []string{policiesPath, controlsPath, evidenceTasksPath, "evidence_records", "sync_times"}
	for _, subdir := range subdirs {
		if err := os.MkdirAll(filepath.Join(baseDir, subdir), 0755); err != nil {
			return nil, fmt.Errorf("failed to create subdirectory %s: %w", subdir, err)
		}
	}

	fileStorage, err := NewFileStorage(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create file storage: %w", err)
	}

	return &LocalDataStore{
		baseDir:           baseDir,
		fileStorage:       fileStorage,
		filenameGenerator: utils.NewFilenameGenerator(),
		fallbackEnabled:   true,
		dataSources:       []string{"local_files"},
		policiesPath:      policiesPath,
		controlsPath:      controlsPath,
		evidenceTasksPath: evidenceTasksPath,
	}, nil
}

// IsDataAvailable checks if local data is available
func (lds *LocalDataStore) IsDataAvailable() bool {
	// Check if we have at least some data files
	categories := []string{lds.policiesPath, lds.controlsPath, lds.evidenceTasksPath}

	for _, category := range categories {
		files, err := lds.fileStorage.List(category)
		if err == nil && len(files) > 0 {
			return true
		}
	}

	return false
}

// GetDataSources returns the list of available data sources
func (lds *LocalDataStore) GetDataSources() []string {
	return lds.dataSources
}

// ValidateDataIntegrity checks the integrity of stored data
func (lds *LocalDataStore) ValidateDataIntegrity() error {
	categories := []string{lds.policiesPath, lds.controlsPath, lds.evidenceTasksPath}

	for _, category := range categories {
		files, err := lds.fileStorage.List(category)
		if err != nil {
			return fmt.Errorf("failed to list %s: %w", category, err)
		}

		for _, file := range files {
			// Try to load each file to validate it can be parsed
			switch category {
			case lds.policiesPath:
				var policy domain.Policy
				if err := lds.fileStorage.Load(category, file, &policy); err != nil {
					return fmt.Errorf("invalid policy file %s: %w", file, err)
				}
			case lds.controlsPath:
				var control domain.Control
				if err := lds.fileStorage.Load(category, file, &control); err != nil {
					return fmt.Errorf("invalid control file %s: %w", file, err)
				}
			case lds.evidenceTasksPath:
				var task domain.EvidenceTask
				if err := lds.fileStorage.Load(category, file, &task); err != nil {
					return fmt.Errorf("invalid evidence task file %s: %w", file, err)
				}
			}
		}
	}

	return nil
}

// SetFallbackEnabled enables or disables fallback behavior
func (lds *LocalDataStore) SetFallbackEnabled(enabled bool) {
	lds.fallbackEnabled = enabled
}

// IsFallbackEnabled returns whether fallback is enabled
func (lds *LocalDataStore) IsFallbackEnabled() bool {
	return lds.fallbackEnabled
}

// ImportData imports data from an external source path
func (lds *LocalDataStore) ImportData(sourcePath string) error {
	// Check if source directory exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("source path does not exist: %s", sourcePath)
	}

	// Copy files from source to local storage
	categories := []string{lds.policiesPath, lds.controlsPath, lds.evidenceTasksPath, "evidence_records"}

	for _, category := range categories {
		sourceDir := filepath.Join(sourcePath, category)
		targetDir := filepath.Join(lds.baseDir, category)

		if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
			continue // Skip if category doesn't exist in source
		}

		// Read source directory
		entries, err := os.ReadDir(sourceDir)
		if err != nil {
			return fmt.Errorf("failed to read source directory %s: %w", sourceDir, err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			sourceFile := filepath.Join(sourceDir, entry.Name())
			targetFile := filepath.Join(targetDir, entry.Name())

			// Copy file
			if err := copyFile(sourceFile, targetFile); err != nil {
				return fmt.Errorf("failed to copy file %s: %w", entry.Name(), err)
			}
		}
	}

	return nil
}

// ExportData exports data to a target path
func (lds *LocalDataStore) ExportData(targetPath string) error {
	// Create target directory
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Copy files to target
	categories := []string{lds.policiesPath, lds.controlsPath, lds.evidenceTasksPath, "evidence_records"}

	for _, category := range categories {
		sourceDir := filepath.Join(lds.baseDir, category)
		targetDir := filepath.Join(targetPath, category)

		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return fmt.Errorf("failed to create target category directory %s: %w", category, err)
		}

		files, err := lds.fileStorage.List(category)
		if err != nil {
			return fmt.Errorf("failed to list files in %s: %w", category, err)
		}

		for _, file := range files {
			sourceFile := filepath.Join(sourceDir, file+".json")
			targetFile := filepath.Join(targetDir, file+".json")

			if err := copyFile(sourceFile, targetFile); err != nil {
				return fmt.Errorf("failed to copy file %s: %w", file, err)
			}
		}
	}

	return nil
}

// GetLastDataUpdate returns the last time the data was updated
func (lds *LocalDataStore) GetLastDataUpdate() (time.Time, error) {
	latestTime := time.Time{}

	categories := []string{lds.policiesPath, lds.controlsPath, lds.evidenceTasksPath}

	for _, category := range categories {
		categoryDir := filepath.Join(lds.baseDir, category)

		entries, err := os.ReadDir(categoryDir)
		if err != nil {
			continue // Skip if directory doesn't exist
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			info, err := entry.Info()
			if err != nil {
				continue
			}

			if info.ModTime().After(latestTime) {
				latestTime = info.ModTime()
			}
		}
	}

	if latestTime.IsZero() {
		return latestTime, fmt.Errorf("no data files found")
	}

	return latestTime, nil
}

// Policy operations
func (lds *LocalDataStore) SavePolicy(policy *domain.Policy) error {
	filename := lds.filenameGenerator.GenerateFilename(
		policy.ReferenceID,
		policy.ID,
		policy.Name,
		"json",
	)

	return lds.fileStorage.Save(lds.policiesPath, filename[:len(filename)-5], policy)
}

func (lds *LocalDataStore) GetPolicy(id string) (*domain.Policy, error) {
	policies, err := lds.GetAllPolicies()
	if err != nil {
		return nil, err
	}

	for _, policy := range policies {
		if policy.ID == id {
			return &policy, nil
		}
	}

	return nil, fmt.Errorf("policy not found: %s", id)
}

func (lds *LocalDataStore) GetPolicyByReferenceAndID(referenceID, numericID string) (*domain.Policy, error) {
	files, err := lds.fileStorage.List(lds.policiesPath)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		fullFilename := file + ".json"
		refID, numID, _, _, err := lds.filenameGenerator.ParseFilename(fullFilename)
		if err == nil && refID == referenceID && numID == numericID {
			var policy domain.Policy
			if err := lds.fileStorage.Load(lds.policiesPath, file, &policy); err == nil {
				return &policy, nil
			}
		}
	}

	return nil, fmt.Errorf("policy not found: %s_%s", referenceID, numericID)
}

func (lds *LocalDataStore) GetAllPolicies() ([]domain.Policy, error) {
	var policies []domain.Policy

	files, err := lds.fileStorage.List(lds.policiesPath)
	if err != nil {
		if lds.fallbackEnabled {
			return policies, nil // Return empty slice instead of error
		}
		return nil, err
	}

	for _, file := range files {
		var policy domain.Policy
		if err := lds.fileStorage.Load(lds.policiesPath, file, &policy); err == nil {
			policies = append(policies, policy)
		}
	}

	return policies, nil
}

func (lds *LocalDataStore) GetPolicySummary() (*domain.PolicySummary, error) {
	policies, err := lds.GetAllPolicies()
	if err != nil {
		return nil, err
	}

	summary := &domain.PolicySummary{
		Total:       len(policies),
		ByFramework: make(map[string]int),
		ByStatus:    make(map[string]int),
		LastSync:    time.Now(),
	}

	for _, policy := range policies {
		if policy.Framework != "" {
			summary.ByFramework[policy.Framework]++
		}
		if policy.Status != "" {
			summary.ByStatus[policy.Status]++
		}
	}

	return summary, nil
}

// Control operations
func (lds *LocalDataStore) SaveControl(control *domain.Control) error {
	filename := lds.filenameGenerator.GenerateFilename(
		control.ReferenceID,
		strconv.Itoa(control.ID),
		control.Name,
		"json",
	)

	return lds.fileStorage.Save(lds.controlsPath, filename[:len(filename)-5], control)
}

func (lds *LocalDataStore) GetControl(id string) (*domain.Control, error) {
	controls, err := lds.GetAllControls()
	if err != nil {
		return nil, err
	}

	numID, _ := strconv.Atoi(id)
	for _, control := range controls {
		if control.ID == numID {
			return &control, nil
		}
	}

	return nil, fmt.Errorf("control not found: %s", id)
}

func (lds *LocalDataStore) GetControlByReferenceAndID(referenceID, numericID string) (*domain.Control, error) {
	files, err := lds.fileStorage.List(lds.controlsPath)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		fullFilename := file + ".json"
		refID, numID, _, _, err := lds.filenameGenerator.ParseFilename(fullFilename)
		if err == nil && refID == referenceID && numID == numericID {
			var control domain.Control
			if err := lds.fileStorage.Load(lds.controlsPath, file, &control); err == nil {
				return &control, nil
			}
		}
	}

	return nil, fmt.Errorf("control not found: %s_%s", referenceID, numericID)
}

func (lds *LocalDataStore) GetAllControls() ([]domain.Control, error) {
	var controls []domain.Control

	files, err := lds.fileStorage.List(lds.controlsPath)
	if err != nil {
		if lds.fallbackEnabled {
			return controls, nil // Return empty slice instead of error
		}
		return nil, err
	}

	for _, file := range files {
		var control domain.Control
		if err := lds.fileStorage.Load(lds.controlsPath, file, &control); err == nil {
			controls = append(controls, control)
		}
	}

	return controls, nil
}

func (lds *LocalDataStore) GetControlSummary() (*domain.ControlSummary, error) {
	controls, err := lds.GetAllControls()
	if err != nil {
		return nil, err
	}

	summary := &domain.ControlSummary{
		Total:       len(controls),
		ByFramework: make(map[string]int),
		ByStatus:    make(map[string]int),
		ByCategory:  make(map[string]int),
		LastSync:    time.Now(),
	}

	for _, control := range controls {
		if control.Framework != "" {
			summary.ByFramework[control.Framework]++
		}
		if control.Status != "" {
			summary.ByStatus[control.Status]++
		}
		if control.Category != "" {
			summary.ByCategory[control.Category]++
		}
	}

	return summary, nil
}

// Evidence task operations
func (lds *LocalDataStore) SaveEvidenceTask(task *domain.EvidenceTask) error {
	filename := lds.filenameGenerator.GenerateFilename(
		task.ReferenceID,
		strconv.Itoa(task.ID),
		task.Name,
		"json",
	)

	return lds.fileStorage.Save(lds.evidenceTasksPath, filename[:len(filename)-5], task)
}

func (lds *LocalDataStore) GetEvidenceTask(id string) (*domain.EvidenceTask, error) {
	tasks, err := lds.GetAllEvidenceTasks()
	if err != nil {
		return nil, err
	}

	numID, _ := strconv.Atoi(id)
	for _, task := range tasks {
		if task.ID == numID {
			return &task, nil
		}
	}

	return nil, fmt.Errorf("evidence task not found: %s", id)
}

func (lds *LocalDataStore) GetEvidenceTaskByReferenceAndID(referenceID, numericID string) (*domain.EvidenceTask, error) {
	files, err := lds.fileStorage.List(lds.evidenceTasksPath)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		fullFilename := file + ".json"
		refID, numID, _, _, err := lds.filenameGenerator.ParseFilename(fullFilename)
		if err == nil && refID == referenceID && numID == numericID {
			var task domain.EvidenceTask
			if err := lds.fileStorage.Load(lds.evidenceTasksPath, file, &task); err == nil {
				return &task, nil
			}
		}
	}

	return nil, fmt.Errorf("evidence task not found: %s_%s", referenceID, numericID)
}

func (lds *LocalDataStore) GetAllEvidenceTasks() ([]domain.EvidenceTask, error) {
	var tasks []domain.EvidenceTask

	files, err := lds.fileStorage.List(lds.evidenceTasksPath)
	if err != nil {
		if lds.fallbackEnabled {
			return tasks, nil // Return empty slice instead of error
		}
		return nil, err
	}

	for _, file := range files {
		var task domain.EvidenceTask
		if err := lds.fileStorage.Load(lds.evidenceTasksPath, file, &task); err == nil {
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

func (lds *LocalDataStore) GetEvidenceTaskSummary() (*domain.EvidenceTaskSummary, error) {
	tasks, err := lds.GetAllEvidenceTasks()
	if err != nil {
		return nil, err
	}

	summary := &domain.EvidenceTaskSummary{
		Total:      len(tasks),
		ByStatus:   make(map[string]int),
		ByPriority: make(map[string]int),
		LastSync:   time.Now(),
	}

	now := time.Now()
	for _, task := range tasks {
		if task.Status != "" {
			summary.ByStatus[task.Status]++
		}
		if task.Priority != "" {
			summary.ByPriority[task.Priority]++
		}

		if task.NextDue != nil {
			if task.NextDue.Before(now) {
				summary.Overdue++
			} else if task.NextDue.Before(now.AddDate(0, 0, 7)) {
				summary.DueSoon++
			}
		}
	}

	return summary, nil
}

// Evidence record operations
func (lds *LocalDataStore) SaveEvidenceRecord(record *domain.EvidenceRecord) error {
	return lds.fileStorage.Save("evidence_records", record.ID, record)
}

func (lds *LocalDataStore) GetEvidenceRecord(id string) (*domain.EvidenceRecord, error) {
	var record domain.EvidenceRecord
	if err := lds.fileStorage.Load("evidence_records", id, &record); err != nil {
		return nil, fmt.Errorf("evidence record not found: %s", id)
	}
	return &record, nil
}

func (lds *LocalDataStore) GetEvidenceRecordsByTaskID(taskID int) ([]domain.EvidenceRecord, error) {
	var records []domain.EvidenceRecord

	files, err := lds.fileStorage.List("evidence_records")
	if err != nil {
		if lds.fallbackEnabled {
			return records, nil // Return empty slice instead of error
		}
		return nil, err
	}

	for _, file := range files {
		var record domain.EvidenceRecord
		if err := lds.fileStorage.Load("evidence_records", file, &record); err == nil {
			if record.TaskID == taskID {
				records = append(records, record)
			}
		}
	}

	return records, nil
}

// Statistics and metadata
func (lds *LocalDataStore) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	policies, err := lds.GetAllPolicies()
	if err == nil {
		stats["total_policies"] = len(policies)
	}

	controls, err := lds.GetAllControls()
	if err == nil {
		stats["total_controls"] = len(controls)
	}

	tasks, err := lds.GetAllEvidenceTasks()
	if err == nil {
		stats["total_evidence_tasks"] = len(tasks)
	}

	stats["generated_at"] = time.Now()
	stats["data_sources"] = lds.dataSources
	stats["fallback_enabled"] = lds.fallbackEnabled

	return stats, nil
}

func (lds *LocalDataStore) SetSyncTime(syncType string, syncTime time.Time) error {
	return lds.fileStorage.Save("sync_times", syncType, map[string]interface{}{
		"last_sync": syncTime,
		"type":      syncType,
	})
}

func (lds *LocalDataStore) GetSyncTime(syncType string) (time.Time, error) {
	var syncData map[string]interface{}
	if err := lds.fileStorage.Load("sync_times", syncType, &syncData); err != nil {
		return time.Time{}, err
	}

	if lastSyncStr, ok := syncData["last_sync"].(string); ok {
		return time.Parse(time.RFC3339, lastSyncStr)
	}

	return time.Time{}, fmt.Errorf("invalid sync time format")
}

// Utility operations
func (lds *LocalDataStore) Clear() error {
	// Clear all collections by delegating to file storage
	collections := []string{lds.policiesPath, lds.controlsPath, lds.evidenceTasksPath, "evidence_records", "sync_times"}

	for _, collection := range collections {
		// Cast to concrete type to access Clear method
		if fs, ok := lds.fileStorage.(*FileStorage); ok {
			if err := fs.Clear(collection); err != nil {
				return fmt.Errorf("failed to clear collection %s: %w", collection, err)
			}
		}
	}

	return nil
}

// copyFile is a helper function to copy files
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
