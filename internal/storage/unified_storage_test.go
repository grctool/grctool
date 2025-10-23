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
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStorage(t *testing.T) {
	tests := map[string]struct {
		baseDir     string
		expectError bool
	}{
		"valid directory": {
			baseDir:     t.TempDir(),
			expectError: false,
		},
		"empty directory": {
			baseDir:     "",
			expectError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cfg := config.StorageConfig{
				DataDir: tc.baseDir,
			}
			storage, err := NewStorage(cfg)

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, storage)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, storage)
				assert.NotNil(t, storage.fileStorage)
				assert.NotNil(t, storage.localDataStore)
				assert.NotNil(t, storage.filenameGenerator)
			}
		})
	}
}

func TestStorage_SavePolicy(t *testing.T) {
	tests := map[string]struct {
		policy      *domain.Policy
		expectError bool
		checkFile   bool
	}{
		"valid policy": {
			policy: &domain.Policy{
				ID:          "12345",
				ReferenceID: "POL-0001",
				Name:        "Test Policy",
				Content:     "Policy content",
				Framework:   "SOC2",
				Status:      "active",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			expectError: false,
			checkFile:   true,
		},
		"policy without ID": {
			policy: &domain.Policy{
				ReferenceID: "POL-0002",
				Name:        "Test Policy 2",
				Content:     "Policy content",
			},
			expectError: false,
			checkFile:   true,
		},
		"nil policy": {
			policy:      nil,
			expectError: true,
			checkFile:   false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tempDir := t.TempDir()
			storage, err := NewStorage(config.StorageConfig{DataDir: tempDir})
			require.NoError(t, err)

			err = storage.SavePolicy(tc.policy)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				if tc.checkFile {
					// Check that file was created
					files, err := filepath.Glob(filepath.Join(tempDir, "docs", "policies", "json", "*.json"))
					require.NoError(t, err)
					assert.NotEmpty(t, files, "Policy file should be created")
				}
			}
		})
	}
}

func TestStorage_SaveControl(t *testing.T) {
	tests := map[string]struct {
		control     *domain.Control
		expectError bool
	}{
		"valid control": {
			control: &domain.Control{
				ID:          778805,
				ReferenceID: "AC-01",
				Name:        "Access Control",
				Description: "Control description",
				Framework:   "SOC2",
				Category:    "Access Control",
				Status:      "implemented",
			},
			expectError: false,
		},
		"control without reference ID": {
			control: &domain.Control{
				ID:          778806,
				Name:        "Another Control",
				Description: "Control description",
			},
			expectError: false,
		},
		"nil control": {
			control:     nil,
			expectError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tempDir := t.TempDir()
			storage, err := NewStorage(config.StorageConfig{DataDir: tempDir})
			require.NoError(t, err)

			err = storage.SaveControl(tc.control)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Check that file was created
				files, err := filepath.Glob(filepath.Join(tempDir, "docs", "controls", "json", "*.json"))
				require.NoError(t, err)
				assert.NotEmpty(t, files, "Control file should be created")
			}
		})
	}
}

func TestStorage_SaveEvidenceTask(t *testing.T) {
	tests := map[string]struct {
		task        *domain.EvidenceTask
		expectError bool
	}{
		"valid evidence task": {
			task: &domain.EvidenceTask{
				ID:                 327992,
				ReferenceID:        "ET-0001",
				Name:               "Test Evidence Task",
				Description:        "Task description",
				CollectionInterval: "quarterly",
				Status:             "pending",
			},
			expectError: false,
		},
		"task with details": {
			task: &domain.EvidenceTask{
				ID:          327993,
				ReferenceID: "ET-0002",
				Name:        "Detailed Task",
				Description: "Task with details",
				Guidance:    "Do this and that with these tips",
			},
			expectError: false,
		},
		"nil task": {
			task:        nil,
			expectError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tempDir := t.TempDir()
			storage, err := NewStorage(config.StorageConfig{DataDir: tempDir})
			require.NoError(t, err)

			err = storage.SaveEvidenceTask(tc.task)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Check that file was created
				files, err := filepath.Glob(filepath.Join(tempDir, "docs", "evidence_tasks", "json", "*.json"))
				require.NoError(t, err)
				assert.NotEmpty(t, files, "Evidence task file should be created")
			}
		})
	}
}

func TestStorage_GetAllPolicies(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewStorage(config.StorageConfig{DataDir: tempDir})
	require.NoError(t, err)

	// Save some test policies
	policies := []*domain.Policy{
		{
			ID:          "1",
			ReferenceID: "POL-0001",
			Name:        "Policy 1",
			Content:     "Content 1",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "2",
			ReferenceID: "POL-0002",
			Name:        "Policy 2",
			Content:     "Content 2",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	for _, p := range policies {
		err := storage.SavePolicy(p)
		require.NoError(t, err)
	}

	// Retrieve all policies
	retrieved, err := storage.GetAllPolicies()
	assert.NoError(t, err)
	assert.Len(t, retrieved, 2, "Should retrieve 2 policies")

	// Check that policies have expected data
	for _, p := range retrieved {
		assert.NotEmpty(t, p.Name)
		assert.NotEmpty(t, p.ReferenceID)
	}
}

func TestStorage_GetAllControls(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewStorage(config.StorageConfig{DataDir: tempDir})
	require.NoError(t, err)

	// Save some test controls
	controls := []*domain.Control{
		{
			ID:          1,
			ReferenceID: "AC-01",
			Name:        "Control 1",
			Description: "Description 1",
		},
		{
			ID:          2,
			ReferenceID: "AC-02",
			Name:        "Control 2",
			Description: "Description 2",
		},
	}

	for _, c := range controls {
		err := storage.SaveControl(c)
		require.NoError(t, err)
	}

	// Retrieve all controls
	retrieved, err := storage.GetAllControls()
	assert.NoError(t, err)
	assert.Len(t, retrieved, 2, "Should retrieve 2 controls")
}

func TestStorage_GetAllEvidenceTasks(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewStorage(config.StorageConfig{DataDir: tempDir})
	require.NoError(t, err)

	// Save some test tasks
	tasks := []*domain.EvidenceTask{
		{
			ID:          1,
			ReferenceID: "ET-0001",
			Name:        "Task 1",
			Description: "Description 1",
		},
		{
			ID:          2,
			ReferenceID: "ET-0002",
			Name:        "Task 2",
			Description: "Description 2",
		},
	}

	for _, task := range tasks {
		err := storage.SaveEvidenceTask(task)
		require.NoError(t, err)
	}

	// Retrieve all tasks
	retrieved, err := storage.GetAllEvidenceTasks()
	assert.NoError(t, err)
	assert.Len(t, retrieved, 2, "Should retrieve 2 tasks")
}

func TestStorage_GetPolicy(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewStorage(config.StorageConfig{DataDir: tempDir})
	require.NoError(t, err)

	// Save a test policy
	policy := &domain.Policy{
		ID:          "12345",
		ReferenceID: "POL-0001",
		Name:        "Test Policy",
		Content:     "Policy content",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err = storage.SavePolicy(policy)
	require.NoError(t, err)

	// Test retrieving by ID
	retrieved, err := storage.GetPolicy("12345")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "12345", retrieved.ID)
	assert.Equal(t, "Test Policy", retrieved.Name)

	// Test retrieving non-existent policy
	notFound, err := storage.GetPolicy("99999")
	assert.Error(t, err)
	assert.Nil(t, notFound)
}

func TestStorage_GetControlByReferenceAndID(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewStorage(config.StorageConfig{DataDir: tempDir})
	require.NoError(t, err)

	// Save a test control
	control := &domain.Control{
		ID:          778805,
		ReferenceID: "AC-01",
		Name:        "Access Control",
		Description: "Control description",
	}
	err = storage.SaveControl(control)
	require.NoError(t, err)

	// Test retrieving by reference and ID
	retrieved, err := storage.GetControlByReferenceAndID("AC-01", "778805")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "AC-01", retrieved.ReferenceID)
	assert.Equal(t, "Access Control", retrieved.Name)

	// Test retrieving non-existent control
	notFound, err := storage.GetControlByReferenceAndID("XX-99", "0")
	assert.Error(t, err)
	assert.Nil(t, notFound)
}

func TestStorage_SaveEvidenceRecord(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewStorage(config.StorageConfig{DataDir: tempDir})
	require.NoError(t, err)

	record := &domain.EvidenceRecord{
		ID:          "rec-001",
		TaskID:      327992,
		Title:       "Test Evidence",
		Description: "Evidence description",
		Content:     "Evidence content",
		Format:      "markdown",
		Source:      "manual",
		CollectedAt: time.Now(),
		CollectedBy: "Test User",
	}

	err = storage.SaveEvidenceRecord(record)
	assert.NoError(t, err)

	// Check that file was created
	files, err := filepath.Glob(filepath.Join(tempDir, "docs", "evidence_records", "*.json"))
	require.NoError(t, err)
	assert.NotEmpty(t, files, "Evidence record file should be created")
}

func TestStorage_Clear(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewStorage(config.StorageConfig{DataDir: tempDir})
	require.NoError(t, err)

	// Save a policy
	policy := &domain.Policy{
		ID:          "12345",
		ReferenceID: "POL-0001",
		Name:        "Test Policy",
		Content:     "Policy content",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err = storage.SavePolicy(policy)
	require.NoError(t, err)

	// Verify it exists
	retrieved, err := storage.GetPolicy("12345")
	require.NoError(t, err)
	assert.NotNil(t, retrieved)

	// Clear all data
	err = storage.Clear()
	assert.NoError(t, err)

	// Verify it's gone
	deleted, err := storage.GetPolicy("12345")
	assert.Error(t, err)
	assert.Nil(t, deleted)
}

func TestStorage_ClearData(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewStorage(config.StorageConfig{DataDir: tempDir})
	require.NoError(t, err)

	// Save various items
	policy := &domain.Policy{
		ID:          "1",
		ReferenceID: "POL-0001",
		Name:        "Test Policy",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	control := &domain.Control{
		ID:          1,
		ReferenceID: "AC-01",
		Name:        "Test Control",
	}
	task := &domain.EvidenceTask{
		ID:          1,
		ReferenceID: "ET-0001",
		Name:        "Test Task",
	}

	require.NoError(t, storage.SavePolicy(policy))
	require.NoError(t, storage.SaveControl(control))
	require.NoError(t, storage.SaveEvidenceTask(task))

	// Clear all
	err = storage.Clear()
	assert.NoError(t, err)

	// Verify everything is gone
	policies, _ := storage.GetAllPolicies()
	controls, _ := storage.GetAllControls()
	tasks, _ := storage.GetAllEvidenceTasks()

	assert.Empty(t, policies)
	assert.Empty(t, controls)
	assert.Empty(t, tasks)
}

func TestStorage_SaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewStorage(config.StorageConfig{DataDir: tempDir})
	require.NoError(t, err)

	// Save some data
	policy := &domain.Policy{
		ID:          "1",
		ReferenceID: "POL-0001",
		Name:        "Test Policy",
		Content:     "Policy content",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err = storage.SavePolicy(policy)
	require.NoError(t, err)

	// Note: Storage doesn't have built-in backup/restore
	// We'll just test save and load operations

	// Verify data can be saved and retrieved
	restored, err := storage.GetPolicy("1")
	assert.NoError(t, err)
	assert.NotNil(t, restored)
	assert.Equal(t, "Test Policy", restored.Name)
}

func TestStorage_GetStatistics(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewStorage(config.StorageConfig{DataDir: tempDir})
	require.NoError(t, err)

	// Save various items
	for i := 1; i <= 3; i++ {
		policy := &domain.Policy{
			ID:          strconv.Itoa(i),
			ReferenceID: fmt.Sprintf("POL-%04d", i),
			Name:        fmt.Sprintf("Policy %d", i),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		require.NoError(t, storage.SavePolicy(policy))
	}

	for i := 1; i <= 2; i++ {
		control := &domain.Control{
			ID:          i,
			ReferenceID: fmt.Sprintf("AC-%02d", i),
			Name:        fmt.Sprintf("Control %d", i),
		}
		require.NoError(t, storage.SaveControl(control))
	}

	stats, err := storage.GetStats()
	assert.NoError(t, err)
	// Stats returns a map
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "policies")
	assert.Contains(t, stats, "controls")
}
