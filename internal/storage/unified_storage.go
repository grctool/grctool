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
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/utils"
)

// Storage provides storage operations with unified filename patterns
// and implements the StorageService interface
type Storage struct {
	fileStorage       *FileStorage
	localDataStore    *LocalDataStore
	filenameGenerator *utils.FilenameGenerator
	docsDir           string              // Directory for synced documents
	paths             config.StoragePaths // Configured paths
}

// Ensure Storage implements the StorageService interface
var _ interfaces.StorageService = (*Storage)(nil)

// NewStorage creates a new unified storage instance using StorageConfig
func NewStorage(cfg config.StorageConfig) (*Storage, error) {
	if cfg.DataDir == "" {
		return nil, fmt.Errorf("data_dir cannot be empty")
	}

	// Apply defaults and resolve paths relative to data_dir
	paths := cfg.Paths.WithDefaults().ResolveRelativeTo(cfg.DataDir)

	// Use the configured docs directory
	docsDir := paths.Docs

	// FileStorage operates within the docs directory
	fileStorage, err := NewFileStorage(docsDir)
	if err != nil {
		return nil, err
	}

	localDataStore, err := NewLocalDataStore(cfg.DataDir, paths)
	if err != nil {
		return nil, err
	}

	return &Storage{
		fileStorage:       fileStorage,
		localDataStore:    localDataStore,
		filenameGenerator: utils.NewFilenameGenerator(),
		docsDir:           docsDir,
		paths:             paths,
	}, nil
}

// Helper methods to get relative paths for fileStorage
func (us *Storage) policiesPath() string {
	return strings.TrimPrefix(us.paths.PoliciesJSON, us.fileStorage.baseDir+"/")
}

func (us *Storage) controlsPath() string {
	return strings.TrimPrefix(us.paths.ControlsJSON, us.fileStorage.baseDir+"/")
}

func (us *Storage) evidenceTasksPath() string {
	return strings.TrimPrefix(us.paths.EvidenceTasksJSON, us.fileStorage.baseDir+"/")
}

// SavePolicy saves a policy with unified filename pattern
func (us *Storage) SavePolicy(policy *domain.Policy) error {
	if policy == nil {
		return fmt.Errorf("policy cannot be nil")
	}

	// Generate unified filename
	filename := us.filenameGenerator.GenerateFilename(
		policy.ReferenceID,
		policy.ID,
		policy.Name,
		"json",
	)

	// Save using the new filename pattern
	return us.fileStorage.Save(us.policiesPath(), filename[:len(filename)-5], policy) // Remove .json extension
}

// SaveControl saves a control with unified filename pattern
func (us *Storage) SaveControl(control *domain.Control) error {
	if control == nil {
		return fmt.Errorf("control cannot be nil")
	}

	// Generate unified filename
	filename := us.filenameGenerator.GenerateFilename(
		control.ReferenceID,
		strconv.Itoa(control.ID),
		control.Name,
		"json",
	)

	// Save using the new filename pattern
	return us.fileStorage.Save(us.controlsPath(), filename[:len(filename)-5], control) // Remove .json extension
}

// SaveEvidenceTask saves an evidence task with unified filename pattern
func (us *Storage) SaveEvidenceTask(task *domain.EvidenceTask) error {
	if task == nil {
		return fmt.Errorf("evidence task cannot be nil")
	}

	// Generate unified filename
	filename := us.filenameGenerator.GenerateFilename(
		task.ReferenceID,
		strconv.Itoa(task.ID),
		task.Name,
		"json",
	)

	// Save using the new filename pattern
	return us.fileStorage.Save(us.evidenceTasksPath(), filename[:len(filename)-5], task) // Remove .json extension
}

// GetPolicyByReferenceAndID retrieves a policy by reference ID and numeric ID
func (us *Storage) GetPolicyByReferenceAndID(referenceID, numericID string) (*domain.Policy, error) {
	// Try to find by parsing filenames
	ids, err := us.fileStorage.List(us.policiesPath())
	if err != nil {
		return nil, err
	}

	for _, id := range ids {
		// Check if this matches our pattern
		fullFilename := id + ".json"
		refID, numID, _, _, err := us.filenameGenerator.ParseFilename(fullFilename)
		if err == nil && refID == referenceID && numID == numericID {
			var policy domain.Policy
			if err := us.fileStorage.Load(us.policiesPath(), id, &policy); err == nil {
				return &policy, nil
			}
		}
	}

	return nil, fmt.Errorf("policy not found: %s_%s", referenceID, numericID)
}

// GetControlByReferenceAndID retrieves a control by reference ID and numeric ID
func (us *Storage) GetControlByReferenceAndID(referenceID, numericID string) (*domain.Control, error) {
	// Try to find by parsing filenames
	ids, err := us.fileStorage.List(us.controlsPath())
	if err != nil {
		return nil, err
	}

	for _, id := range ids {
		// Check if this matches our pattern
		fullFilename := id + ".json"
		refID, numID, _, _, err := us.filenameGenerator.ParseFilename(fullFilename)
		if err == nil && refID == referenceID && numID == numericID {
			var control domain.Control
			if err := us.fileStorage.Load(us.controlsPath(), id, &control); err == nil {
				return &control, nil
			}
		}
	}

	return nil, fmt.Errorf("control not found: %s_%s", referenceID, numericID)
}

// GetEvidenceTaskByReferenceAndID retrieves an evidence task by reference ID and numeric ID
func (us *Storage) GetEvidenceTaskByReferenceAndID(referenceID, numericID string) (*domain.EvidenceTask, error) {
	// Try to find by parsing filenames
	ids, err := us.fileStorage.List(us.evidenceTasksPath())
	if err != nil {
		return nil, err
	}

	for _, id := range ids {
		// Check if this matches our pattern
		fullFilename := id + ".json"
		refID, numID, _, _, err := us.filenameGenerator.ParseFilename(fullFilename)
		if err == nil && refID == referenceID && numID == numericID {
			var task domain.EvidenceTask
			if err := us.fileStorage.Load(us.evidenceTasksPath(), id, &task); err == nil {
				return &task, nil
			}
		}
	}

	return nil, fmt.Errorf("evidence task not found: %s_%s", referenceID, numericID)
}

// GetPolicy retrieves a policy by ID (numeric, reference ID, or filename)
func (us *Storage) GetPolicy(id string) (*domain.Policy, error) {
	policies, err := us.GetAllPolicies()
	if err != nil {
		return nil, err
	}

	// Try numeric/string ID first
	for _, policy := range policies {
		if policy.ID == id {
			return &policy, nil
		}
	}

	// Try reference ID (POL-001)
	for _, policy := range policies {
		if policy.ReferenceID == id {
			return &policy, nil
		}
	}

	// Try filename format (POL_001)
	convertedID := strings.ReplaceAll(id, "_", "-")
	for _, policy := range policies {
		if policy.ReferenceID == convertedID {
			return &policy, nil
		}
	}

	return nil, fmt.Errorf("policy not found: %s", id)
}

// GetControl retrieves a control by ID (numeric, reference ID, or filename)
func (us *Storage) GetControl(id string) (*domain.Control, error) {
	controls, err := us.GetAllControls()
	if err != nil {
		return nil, err
	}

	// Try numeric ID first
	if numID, err := strconv.Atoi(id); err == nil {
		for _, control := range controls {
			if control.ID == numID {
				return &control, nil
			}
		}
	}

	// Try reference ID (CC1.1)
	for _, control := range controls {
		if control.ReferenceID == id {
			return &control, nil
		}
	}

	// Try filename format (CC1_1)
	convertedID := strings.ReplaceAll(id, "_", ".")
	for _, control := range controls {
		if control.ReferenceID == convertedID {
			return &control, nil
		}
	}

	return nil, fmt.Errorf("control not found: %s", id)
}

// GetEvidenceTask retrieves an evidence task by numeric ID or task reference (ET-0001, ET0001, or 327992)
func (us *Storage) GetEvidenceTask(id string) (*domain.EvidenceTask, error) {
	tasks, err := us.GetAllEvidenceTasks()
	if err != nil {
		return nil, err
	}

	// Parse the ID to get the numeric task ID
	numID := us.parseTaskID(id)
	if numID == 0 {
		return nil, fmt.Errorf("invalid task ID format: %s (expected: numeric ID, ET-0001, or ET0001)", id)
	}

	// Find task by numeric ID
	for _, task := range tasks {
		if task.ID == numID {
			return &task, nil
		}
	}

	return nil, fmt.Errorf("evidence task not found: %s (parsed as ID %d)", id, numID)
}

// parseTaskID extracts the numeric task ID from various formats:
// - "327992" -> 327992 (pure numeric)
// - "ET-0001" -> 327992 (reference with dash, looks up by ReferenceID)
// - "ET0001" -> 327992 (reference without dash, looks up by ReferenceID)
// - "ET-1" -> 327992 (reference with dash, zero-padding)
func (us *Storage) parseTaskID(id string) int {
	trimmed := strings.TrimSpace(id)

	// Try pure numeric ID first
	if numID, err := strconv.Atoi(trimmed); err == nil {
		return numID
	}

	// Try parsing as task reference (ET-0001 or ET0001)
	upper := strings.ToUpper(trimmed)

	// Match ET-XXXX format (with dash)
	if matched, _ := regexp.MatchString(`^ET-\d+$`, upper); matched {
		refNum := strings.TrimPrefix(upper, "ET-")
		if num, err := strconv.Atoi(refNum); err == nil {
			// Look up task by ReferenceID
			return us.lookupTaskByReferenceNumber(num)
		}
	}

	// Match ETXXXX format (without dash)
	if matched, _ := regexp.MatchString(`^ET\d+$`, upper); matched {
		refNum := strings.TrimPrefix(upper, "ET")
		if num, err := strconv.Atoi(refNum); err == nil {
			// Look up task by ReferenceID
			return us.lookupTaskByReferenceNumber(num)
		}
	}

	return 0
}

// lookupTaskByReferenceNumber finds a task by its reference number (e.g., 1 for ET-0001)
func (us *Storage) lookupTaskByReferenceNumber(refNum int) int {
	tasks, err := us.GetAllEvidenceTasks()
	if err != nil {
		return 0
	}

	// Format as ET-XXXX with zero padding
	refID := fmt.Sprintf("ET-%04d", refNum)

	for _, task := range tasks {
		if task.ReferenceID == refID {
			return task.ID
		}
	}

	return 0
}

// GetAllPolicies retrieves all policies regardless of filename format
func (us *Storage) GetAllPolicies() ([]domain.Policy, error) {
	var policies []domain.Policy

	ids, err := us.fileStorage.List(us.policiesPath())
	if err != nil {
		return nil, err
	}

	for _, id := range ids {
		var policy domain.Policy
		if err := us.fileStorage.Load(us.policiesPath(), id, &policy); err == nil {
			policies = append(policies, policy)
		}
	}

	return policies, nil
}

// GetAllControls retrieves all controls regardless of filename format
func (us *Storage) GetAllControls() ([]domain.Control, error) {
	var controls []domain.Control

	ids, err := us.fileStorage.List(us.controlsPath())
	if err != nil {
		return nil, err
	}

	for _, id := range ids {
		var control domain.Control
		if err := us.fileStorage.Load(us.controlsPath(), id, &control); err == nil {
			controls = append(controls, control)
		}
	}

	return controls, nil
}

// GetAllEvidenceTasks retrieves all evidence tasks regardless of filename format
func (us *Storage) GetAllEvidenceTasks() ([]domain.EvidenceTask, error) {
	var tasks []domain.EvidenceTask

	ids, err := us.fileStorage.List(us.evidenceTasksPath())
	if err != nil {
		return nil, err
	}

	for _, id := range ids {
		var task domain.EvidenceTask
		if err := us.fileStorage.Load(us.evidenceTasksPath(), id, &task); err == nil {
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

// SaveEvidenceRecord saves an evidence record (no unified naming needed for records)
func (us *Storage) SaveEvidenceRecord(record *domain.EvidenceRecord) error {
	if record == nil {
		return fmt.Errorf("evidence record cannot be nil")
	}

	// Evidence records use their own ID format, no unified naming needed
	return us.fileStorage.Save("evidence_records", record.ID, record)
}

// GetEvidenceRecord retrieves an evidence record by ID
func (us *Storage) GetEvidenceRecord(id string) (*domain.EvidenceRecord, error) {
	var record domain.EvidenceRecord
	if err := us.fileStorage.Load("evidence_records", id, &record); err != nil {
		return nil, fmt.Errorf("evidence record not found: %s", id)
	}
	return &record, nil
}

// GetEvidenceRecordsByTaskID retrieves all evidence records for a specific task
func (us *Storage) GetEvidenceRecordsByTaskID(taskID int) ([]domain.EvidenceRecord, error) {
	var records []domain.EvidenceRecord

	ids, err := us.fileStorage.List("evidence_records")
	if err != nil {
		return nil, err
	}

	for _, id := range ids {
		var record domain.EvidenceRecord
		if err := us.fileStorage.Load("evidence_records", id, &record); err == nil {
			if record.TaskID == taskID {
				records = append(records, record)
			}
		}
	}

	return records, nil
}

// GetPolicySummary returns a summary of all policies
func (us *Storage) GetPolicySummary() (*domain.PolicySummary, error) {
	policies, err := us.GetAllPolicies()
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

// GetControlSummary returns a summary of all controls
func (us *Storage) GetControlSummary() (*domain.ControlSummary, error) {
	controls, err := us.GetAllControls()
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

// GetEvidenceTaskSummary returns a summary of all evidence tasks
func (us *Storage) GetEvidenceTaskSummary() (*domain.EvidenceTaskSummary, error) {
	tasks, err := us.GetAllEvidenceTasks()
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

		// Count overdue and due soon
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

// GetStats returns statistics about stored data
func (us *Storage) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Count policies
	policies, err := us.GetAllPolicies()
	if err == nil {
		stats["policies"] = len(policies)
		stats["total_policies"] = len(policies)
	}

	// Count controls
	controls, err := us.GetAllControls()
	if err == nil {
		stats["controls"] = len(controls)
		stats["total_controls"] = len(controls)
	}

	// Count evidence tasks
	tasks, err := us.GetAllEvidenceTasks()
	if err == nil {
		stats["total_evidence_tasks"] = len(tasks)
	}

	stats["generated_at"] = time.Now()

	return stats, nil
}

// SetSyncTime stores the last sync time for a given sync type
func (us *Storage) SetSyncTime(syncType string, syncTime time.Time) error {
	return us.fileStorage.Save("sync_times", syncType, map[string]interface{}{
		"last_sync": syncTime,
		"type":      syncType,
	})
}

// GetSyncTime retrieves the last sync time for a given sync type
func (us *Storage) GetSyncTime(syncType string) (time.Time, error) {
	var syncData map[string]interface{}
	if err := us.fileStorage.Load("sync_times", syncType, &syncData); err != nil {
		return time.Time{}, err
	}

	if lastSyncStr, ok := syncData["last_sync"].(string); ok {
		return time.Parse(time.RFC3339, lastSyncStr)
	}

	return time.Time{}, fmt.Errorf("invalid sync time format")
}

// Clear removes all stored data (use with caution!)
func (us *Storage) Clear() error {
	// Clear local data store first
	if err := us.localDataStore.Clear(); err != nil {
		return fmt.Errorf("failed to clear local data store: %w", err)
	}

	// Clear file storage
	if err := us.fileStorage.ClearAll(); err != nil {
		return fmt.Errorf("failed to clear file storage: %w", err)
	}

	return nil
}

// GetBaseDir returns the base directory for storage
func (us *Storage) GetBaseDir() string {
	return us.localDataStore.GetBaseDir()
}
