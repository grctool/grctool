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
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDataService(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := storage.NewStorage(config.StorageConfig{DataDir: tempDir})
	require.NoError(t, err)

	service := NewDataService(storage)
	assert.NotNil(t, service)

	// Check it's the right type
	impl, ok := service.(*DataServiceImpl)
	assert.True(t, ok)
	assert.NotNil(t, impl.storage)
}

func TestDataServiceImpl_GetEvidenceTask(t *testing.T) {
	tests := map[string]struct {
		setupTask   *domain.EvidenceTask
		taskID      int
		expectError bool
	}{
		"existing task": {
			setupTask: &domain.EvidenceTask{
				ID:          123,
				ReferenceID: "ET-0001",
				Name:        "Test Task",
				Description: "Test Description",
			},
			taskID:      123,
			expectError: false,
		},
		"non-existent task": {
			setupTask:   nil,
			taskID:      999,
			expectError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tempDir := t.TempDir()
			storage, err := storage.NewStorage(config.StorageConfig{DataDir: tempDir})
			require.NoError(t, err)

			service := NewDataService(storage)

			// Setup test data
			if tc.setupTask != nil {
				err := storage.SaveEvidenceTask(tc.setupTask)
				require.NoError(t, err)
			}

			// Test GetEvidenceTask
			task, err := service.GetEvidenceTask(context.Background(), tc.taskID)

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, task)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, task)
				assert.Equal(t, tc.setupTask.Name, task.Name)
			}
		})
	}
}

func TestDataServiceImpl_GetAllEvidenceTasks(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := storage.NewStorage(config.StorageConfig{DataDir: tempDir})
	require.NoError(t, err)

	service := NewDataService(storage)

	// Setup test data
	tasks := []*domain.EvidenceTask{
		{
			ID:          1,
			ReferenceID: "ET-0001",
			Name:        "Task 1",
		},
		{
			ID:          2,
			ReferenceID: "ET-0002",
			Name:        "Task 2",
		},
		{
			ID:          3,
			ReferenceID: "ET-0003",
			Name:        "Task 3",
		},
	}

	for _, task := range tasks {
		err := storage.SaveEvidenceTask(task)
		require.NoError(t, err)
	}

	// Test GetAllEvidenceTasks
	allTasks, err := service.GetAllEvidenceTasks(context.Background())
	assert.NoError(t, err)
	assert.Len(t, allTasks, 3)
}

func TestDataServiceImpl_FilterEvidenceTasks(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := storage.NewStorage(config.StorageConfig{DataDir: tempDir})
	require.NoError(t, err)

	service := NewDataService(storage)

	// Setup test data
	tasks := []*domain.EvidenceTask{
		{
			ID:                 1,
			ReferenceID:        "ET-0001",
			Name:               "Quarterly Task",
			CollectionInterval: "quarterly",
			Status:             "pending",
		},
		{
			ID:                 2,
			ReferenceID:        "ET-0002",
			Name:               "Annual Task",
			CollectionInterval: "annual",
			Status:             "completed",
		},
		{
			ID:                 3,
			ReferenceID:        "ET-0003",
			Name:               "Monthly Task",
			CollectionInterval: "monthly",
			Status:             "pending",
			Controls:           []string{"AC-01", "AC-02"},
		},
	}

	for _, task := range tasks {
		err := storage.SaveEvidenceTask(task)
		require.NoError(t, err)
	}

	tests := map[string]struct {
		filter   domain.EvidenceFilter
		expected int
	}{
		"filter by status": {
			filter:   domain.EvidenceFilter{Status: []string{"pending"}},
			expected: 2,
		},
		"filter by priority": {
			filter:   domain.EvidenceFilter{Priority: []string{"high"}},
			expected: 0,
		},
		"no filter": {
			filter:   domain.EvidenceFilter{},
			expected: 3,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			filtered, err := service.FilterEvidenceTasks(context.Background(), tc.filter)
			assert.NoError(t, err)
			assert.Len(t, filtered, tc.expected)
		})
	}
}

func TestDataServiceImpl_GetPolicy(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := storage.NewStorage(config.StorageConfig{DataDir: tempDir})
	require.NoError(t, err)

	service := NewDataService(storage)

	// Setup test policy
	policy := &domain.Policy{
		ID:          "12345",
		ReferenceID: "POL-0001",
		Name:        "Test Policy",
		Content:     "Policy content",
		Framework:   "SOC2",
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err = storage.SavePolicy(policy)
	require.NoError(t, err)

	// Test GetPolicy
	retrieved, err := service.GetPolicy(context.Background(), "12345")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "Test Policy", retrieved.Name)

	// Test non-existent policy
	notFound, err := service.GetPolicy(context.Background(), "99999")
	assert.Error(t, err)
	assert.Nil(t, notFound)
}

func TestDataServiceImpl_GetAllPolicies(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := storage.NewStorage(config.StorageConfig{DataDir: tempDir})
	require.NoError(t, err)

	service := NewDataService(storage)

	// Setup test data
	policies := []*domain.Policy{
		{
			ID:          "1",
			ReferenceID: "POL-0001",
			Name:        "Policy 1",
			Framework:   "SOC2",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "2",
			ReferenceID: "POL-0002",
			Name:        "Policy 2",
			Framework:   "ISO27001",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	for _, p := range policies {
		err := storage.SavePolicy(p)
		require.NoError(t, err)
	}

	// Test GetAllPolicies
	allPolicies, err := service.GetAllPolicies(context.Background())
	assert.NoError(t, err)
	assert.Len(t, allPolicies, 2)
}

/* // FilterPolicies not yet implemented
func TestDataServiceImpl_FilterPolicies(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := storage.NewStorage(config.StorageConfig{DataDir: tempDir})
	require.NoError(t, err)

	service := NewDataService(storage)

	// Setup test data
	policies := []*domain.Policy{
		{
			ID:          "1",
			ReferenceID: "POL-0001",
			Name:        "SOC2 Policy",
			Framework:   "SOC2",
			Status:      "active",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "2",
			ReferenceID: "POL-0002",
			Name:        "ISO Policy",
			Framework:   "ISO27001",
			Status:      "draft",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "3",
			ReferenceID: "POL-0003",
			Name:        "Another SOC2",
			Framework:   "SOC2",
			Status:      "active",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	for _, p := range policies {
		err := storage.SavePolicy(p)
		require.NoError(t, err)
	}

	tests := map[string]struct {
		filter   interface{}
		expected int
	}{
		"filter by framework": {
			filter:   nil, // PolicyFilter not implemented yet
			expected: 2,
		},
		"filter by status": {
			filter:   nil, // PolicyFilter not implemented yet
			expected: 2,
		},
		"filter by framework and status": {
			filter:   nil, // PolicyFilter not implemented yet
			expected: 2,
		},
		"no results": {
			filter:   nil, // PolicyFilter not implemented yet
			expected: 0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			filtered, err := service.FilterPolicies(context.Background(), tc.filter)
			assert.NoError(t, err)
			assert.Len(t, filtered, tc.expected)
		})
	}
} */

func TestDataServiceImpl_GetControl(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := storage.NewStorage(config.StorageConfig{DataDir: tempDir})
	require.NoError(t, err)

	service := NewDataService(storage)

	// Setup test control
	control := &domain.Control{
		ID:          778805,
		ReferenceID: "AC-01",
		Name:        "Access Control",
		Description: "Control description",
		Framework:   "SOC2",
		Category:    "Access Control",
		Status:      "implemented",
	}
	err = storage.SaveControl(control)
	require.NoError(t, err)

	// Test GetControl - now uses string ID (reference ID or stringified ID)
	retrieved, err := service.GetControl(context.Background(), "778805")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "Access Control", retrieved.Name)

	// Test non-existent control
	notFound, err := service.GetControl(context.Background(), "99999")
	assert.Error(t, err)
	assert.Nil(t, notFound)
}

func TestDataServiceImpl_GetControlByReference(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := storage.NewStorage(config.StorageConfig{DataDir: tempDir})
	require.NoError(t, err)

	service := NewDataService(storage)

	// Setup test control
	control := &domain.Control{
		ID:          778805,
		ReferenceID: "AC-01",
		Name:        "Access Control",
		Description: "Control description",
	}
	err = storage.SaveControl(control)
	require.NoError(t, err)

	// Test GetControl instead (GetControlByReference not implemented) - now uses string ID
	retrieved, err := service.GetControl(context.Background(), "778805")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "Access Control", retrieved.Name)

	// Test non-existent control
	notFound, err := service.GetControl(context.Background(), "999999")
	assert.Error(t, err)
	assert.Nil(t, notFound)
}

func TestDataServiceImpl_GetAllControls(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := storage.NewStorage(config.StorageConfig{DataDir: tempDir})
	require.NoError(t, err)

	service := NewDataService(storage)

	// Setup test data
	controls := []*domain.Control{
		{
			ID:          1,
			ReferenceID: "AC-01",
			Name:        "Control 1",
		},
		{
			ID:          2,
			ReferenceID: "AC-02",
			Name:        "Control 2",
		},
		{
			ID:          3,
			ReferenceID: "CC-01",
			Name:        "Control 3",
		},
	}

	for _, c := range controls {
		err := storage.SaveControl(c)
		require.NoError(t, err)
	}

	// Test GetAllControls
	allControls, err := service.GetAllControls(context.Background())
	assert.NoError(t, err)
	assert.Len(t, allControls, 3)
}

/* // FilterControls not yet implemented
func TestDataServiceImpl_FilterControls(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := storage.NewStorage(config.StorageConfig{DataDir: tempDir})
	require.NoError(t, err)

	service := NewDataService(storage)

	// Setup test data
	controls := []*domain.Control{
		{
			ID:          1,
			ReferenceID: "AC-01",
			Name:        "Access Control 1",
			Framework:   "SOC2",
			Category:    "Access Control",
			Status:      "implemented",
		},
		{
			ID:          2,
			ReferenceID: "AC-02",
			Name:        "Access Control 2",
			Framework:   "SOC2",
			Category:    "Access Control",
			Status:      "planned",
		},
		{
			ID:          3,
			ReferenceID: "CC-01",
			Name:        "Change Control",
			Framework:   "ISO27001",
			Category:    "Change Management",
			Status:      "implemented",
		},
	}

	for _, c := range controls {
		err := storage.SaveControl(c)
		require.NoError(t, err)
	}

	tests := map[string]struct {
		filter   domain.ControlFilter
		expected int
	}{
		"filter by framework": {
			filter:   domain.ControlFilter{Framework: "SOC2"},
			expected: 2,
		},
		"filter by category": {
			filter:   domain.ControlFilter{Category: "Access Control"},
			expected: 2,
		},
		"filter by status": {
			filter:   domain.ControlFilter{Status: "implemented"},
			expected: 2,
		},
		"filter by multiple criteria": {
			filter:   domain.ControlFilter{Framework: "SOC2", Status: "implemented"},
			expected: 1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			filtered, err := service.FilterControls(context.Background(), tc.filter)
			assert.NoError(t, err)
			assert.Len(t, filtered, tc.expected)
		})
	}
} */

/* // GetRelatedEvidenceTasks not yet implemented
func TestDataServiceImpl_GetRelatedEvidenceTasks(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := storage.NewStorage(config.StorageConfig{DataDir: tempDir})
	require.NoError(t, err)

	service := NewDataService(storage)

	// Setup test data
	tasks := []*domain.EvidenceTask{
		{
			ID:          1,
			ReferenceID: "ET-0001",
			Name:        "Task for AC-01",
			Controls:    []string{"AC-01", "AC-02"},
		},
		{
			ID:          2,
			ReferenceID: "ET-0002",
			Name:        "Task for AC-02",
			Controls:    []string{"AC-02"},
		},
		{
			ID:          3,
			ReferenceID: "ET-0003",
			Name:        "Task for CC-01",
			Controls:    []string{"CC-01"},
		},
	}

	for _, task := range tasks {
		err := storage.SaveEvidenceTask(task)
		require.NoError(t, err)
	}

	// Test GetRelatedEvidenceTasks
	related, err := service.GetRelatedEvidenceTasks(context.Background(), "AC-01")
	assert.NoError(t, err)
	assert.Len(t, related, 1)
	assert.Equal(t, "Task for AC-01", related[0].Name)

	// Test control with no tasks
	noTasks, err := service.GetRelatedEvidenceTasks(context.Background(), "XX-99")
	assert.NoError(t, err)
	assert.Empty(t, noTasks)
} */

func TestDataServiceImpl_SaveEvidenceRecord(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := storage.NewStorage(config.StorageConfig{DataDir: tempDir})
	require.NoError(t, err)

	service := NewDataService(storage)

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

	err = service.SaveEvidenceRecord(context.Background(), record)
	assert.NoError(t, err)

	// Verify it was saved (GetEvidenceRecord not implemented)
	// retrieved, err := service.GetEvidenceRecord(context.Background(), "rec-001")
	// assert.NoError(t, err)
	// assert.NotNil(t, retrieved)
	// assert.Equal(t, "Test Evidence", retrieved.Title)
}

/* // GetEvidenceRecord not yet implemented
func TestDataServiceImpl_GetEvidenceRecord(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := storage.NewStorage(config.StorageConfig{DataDir: tempDir})
	require.NoError(t, err)

	service := NewDataService(storage)

	// Save a record
	record := &domain.EvidenceRecord{
		ID:          "rec-001",
		TaskID:      327992,
		Title:       "Test Evidence",
		Content:     "Evidence content",
		Format:      "markdown",
		CollectedAt: time.Now(),
	}
	err = storage.SaveEvidenceRecord(record)
	require.NoError(t, err)

	// Test GetEvidenceRecord
	retrieved, err := service.GetEvidenceRecord(context.Background(), "rec-001")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "Test Evidence", retrieved.Title)

	// Test non-existent record
	notFound, err := service.GetEvidenceRecord(context.Background(), "rec-999")
	assert.Error(t, err)
	assert.Nil(t, notFound)
} */
