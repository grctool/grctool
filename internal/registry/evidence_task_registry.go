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

package registry

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/grctool/grctool/internal/domain"
)

// EvidenceTaskRegistry manages the mapping between evidence task IDs and their reference codes
type EvidenceTaskRegistry struct {
	filePath   string
	entries    map[int]*RegistryEntry // Maps task ID to registry entry
	references map[string]int         // Maps reference (ET1, ET2, etc.) to task ID
	nextRefNum int                    // Next available reference number
}

// RegistryEntry represents a single entry in the evidence task registry
type RegistryEntry struct {
	TaskID    int    `json:"task_id"`
	Reference string `json:"reference"`
	Name      string `json:"name"`
	Framework string `json:"framework"`
	Status    string `json:"status"`
}

// NewEvidenceTaskRegistry creates a new evidence task registry
func NewEvidenceTaskRegistry(dataDir string) *EvidenceTaskRegistry {
	registryPath := filepath.Join(dataDir, "docs", "evidence_task_registry.csv")
	return &EvidenceTaskRegistry{
		filePath:   registryPath,
		entries:    make(map[int]*RegistryEntry),
		references: make(map[string]int),
		nextRefNum: 1,
	}
}

// LoadRegistry loads the registry from the CSV file
func (r *EvidenceTaskRegistry) LoadRegistry() error {
	// Check if file exists
	if _, err := os.Stat(r.filePath); os.IsNotExist(err) {
		// File doesn't exist, start with empty registry
		return nil
	}

	file, err := os.Open(r.filePath)
	if err != nil {
		return fmt.Errorf("failed to open registry file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Read header row
	header, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			// Empty file, that's okay
			return nil
		}
		return fmt.Errorf("failed to read header: %w", err)
	}

	// Validate header
	expectedHeader := []string{"task_id", "reference", "name", "framework", "status"}
	if len(header) != len(expectedHeader) {
		return fmt.Errorf("invalid header: expected %v, got %v", expectedHeader, header)
	}

	// Read entries
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read registry entry: %w", err)
		}

		if len(record) != 5 {
			continue // Skip malformed rows
		}

		taskID, err := strconv.Atoi(record[0])
		if err != nil {
			continue // Skip rows with invalid task ID
		}

		entry := &RegistryEntry{
			TaskID:    taskID,
			Reference: record[1],
			Name:      record[2],
			Framework: record[3],
			Status:    record[4],
		}

		r.entries[taskID] = entry
		r.references[entry.Reference] = taskID

		// Update next reference number
		if refNum := r.extractReferenceNumber(entry.Reference); refNum >= r.nextRefNum {
			r.nextRefNum = refNum + 1
		}
	}

	return nil
}

// SaveRegistry saves the registry to the CSV file
func (r *EvidenceTaskRegistry) SaveRegistry() error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(r.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create registry directory: %w", err)
	}

	file, err := os.Create(r.filePath)
	if err != nil {
		return fmt.Errorf("failed to create registry file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"task_id", "reference", "name", "framework", "status"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Sort entries by reference number for consistent ordering
	var entries []*RegistryEntry
	for _, entry := range r.entries {
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		refNumI := r.extractReferenceNumber(entries[i].Reference)
		refNumJ := r.extractReferenceNumber(entries[j].Reference)
		return refNumI < refNumJ
	})

	// Write entries
	for _, entry := range entries {
		record := []string{
			strconv.Itoa(entry.TaskID),
			entry.Reference,
			entry.Name,
			entry.Framework,
			entry.Status,
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write registry entry: %w", err)
		}
	}

	return nil
}

// RegisterTask registers a new evidence task or updates an existing one
func (r *EvidenceTaskRegistry) RegisterTask(task *domain.EvidenceTask) string {
	// Check if task is already registered
	if entry, exists := r.entries[task.ID]; exists {
		// Update existing entry
		entry.Name = task.Name
		entry.Framework = task.Framework
		entry.Status = task.Status
		return entry.Reference
	}

	// Create new entry with next available reference
	reference := fmt.Sprintf("ET-%04d", r.nextRefNum)
	r.nextRefNum++

	entry := &RegistryEntry{
		TaskID:    task.ID,
		Reference: reference,
		Name:      task.Name,
		Framework: task.Framework,
		Status:    task.Status,
	}

	r.entries[task.ID] = entry
	r.references[reference] = task.ID

	return reference
}

// GetReference returns the reference for a given task ID
func (r *EvidenceTaskRegistry) GetReference(taskID int) (string, bool) {
	entry, exists := r.entries[taskID]
	if !exists {
		return "", false
	}
	return entry.Reference, true
}

// GetTaskID returns the task ID for a given reference
func (r *EvidenceTaskRegistry) GetTaskID(reference string) (int, bool) {
	taskID, exists := r.references[reference]
	return taskID, exists
}

// GetEntry returns the full registry entry for a given task ID
func (r *EvidenceTaskRegistry) GetEntry(taskID int) (*RegistryEntry, bool) {
	entry, exists := r.entries[taskID]
	return entry, exists
}

// GetAllEntries returns all registry entries sorted by reference number
func (r *EvidenceTaskRegistry) GetAllEntries() []*RegistryEntry {
	var entries []*RegistryEntry
	for _, entry := range r.entries {
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		refNumI := r.extractReferenceNumber(entries[i].Reference)
		refNumJ := r.extractReferenceNumber(entries[j].Reference)
		return refNumI < refNumJ
	})

	return entries
}

// UpdateTaskInfo updates the task information in the registry
func (r *EvidenceTaskRegistry) UpdateTaskInfo(task *domain.EvidenceTask) {
	if entry, exists := r.entries[task.ID]; exists {
		entry.Name = task.Name
		entry.Framework = task.Framework
		entry.Status = task.Status
	}
}

// RemoveTask removes a task from the registry (use with caution)
func (r *EvidenceTaskRegistry) RemoveTask(taskID int) bool {
	entry, exists := r.entries[taskID]
	if !exists {
		return false
	}

	delete(r.entries, taskID)
	delete(r.references, entry.Reference)
	return true
}

// GetStats returns registry statistics
func (r *EvidenceTaskRegistry) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"total_entries":      len(r.entries),
		"next_reference_num": r.nextRefNum,
		"highest_task_id":    r.getHighestTaskID(),
		"registry_file_path": r.filePath,
	}
}

// extractReferenceNumber extracts the numeric part from a reference like "ET-0123" or "ET123"
func (r *EvidenceTaskRegistry) extractReferenceNumber(reference string) int {
	if !strings.HasPrefix(reference, "ET") {
		return 0
	}

	// Remove "ET" prefix
	numStr := strings.TrimPrefix(reference, "ET")
	// Remove hyphen if present (for new format ET-0123)
	numStr = strings.TrimPrefix(numStr, "-")

	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0
	}

	return num
}

// getHighestTaskID returns the highest task ID in the registry
func (r *EvidenceTaskRegistry) getHighestTaskID() int {
	highest := 0
	for taskID := range r.entries {
		if taskID > highest {
			highest = taskID
		}
	}
	return highest
}

// InitializeFromTasks creates initial registry from existing evidence tasks
func (r *EvidenceTaskRegistry) InitializeFromTasks(tasks []domain.EvidenceTask) {
	// Sort tasks by ID to ensure consistent assignment
	tasksCopy := make([]domain.EvidenceTask, len(tasks))
	copy(tasksCopy, tasks)

	sort.Slice(tasksCopy, func(i, j int) bool {
		return tasksCopy[i].ID < tasksCopy[j].ID
	})

	// Register each task
	for _, task := range tasksCopy {
		r.RegisterTask(&task)
	}
}
