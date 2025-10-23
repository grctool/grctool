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
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubmissionStorage_SaveAndLoadSubmission(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()

	// Create storage
	cfg := config.StorageConfig{
		DataDir: tmpDir,
		Paths:   config.StoragePaths{}.WithDefaults(),
	}
	storage, err := NewStorage(cfg)
	require.NoError(t, err)

	// Create test submission
	submission := &models.EvidenceSubmission{
		TaskID:         1234,
		TaskRef:        "ET-0001",
		Window:         "2025-Q4",
		Status:         "draft",
		CreatedAt:      time.Now(),
		TotalFileCount: 3,
		TotalSizeBytes: 12458,
		SubmittedBy:    "test@example.com",
		Notes:          "Test submission",
		EvidenceFiles: []models.EvidenceFileRef{
			{
				Filename:       "test.md",
				SizeBytes:      1024,
				ChecksumSHA256: "abc123",
			},
		},
		ValidationStatus:  "passed",
		CompletenessScore: 1.0,
	}

	// Create evidence directory first
	evidenceDir := filepath.Join(tmpDir, "evidence", "ET-0001", "2025-Q4")
	require.NoError(t, os.MkdirAll(evidenceDir, 0755))

	// Save submission
	err = storage.SaveSubmission(submission)
	require.NoError(t, err)

	// Verify .submission directory was created
	submissionDir := filepath.Join(evidenceDir, submissionMetadataDir)
	assert.DirExists(t, submissionDir)

	// Load submission
	loaded, err := storage.LoadSubmission("ET-0001", "2025-Q4")
	require.NoError(t, err)

	// Verify loaded data
	assert.Equal(t, submission.TaskID, loaded.TaskID)
	assert.Equal(t, submission.TaskRef, loaded.TaskRef)
	assert.Equal(t, submission.Window, loaded.Window)
	assert.Equal(t, submission.Status, loaded.Status)
	assert.Equal(t, submission.TotalFileCount, loaded.TotalFileCount)
	assert.Equal(t, submission.SubmittedBy, loaded.SubmittedBy)
	assert.Equal(t, len(submission.EvidenceFiles), len(loaded.EvidenceFiles))
}

func TestSubmissionStorage_SaveValidationResult(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.StorageConfig{
		DataDir: tmpDir,
		Paths:   config.StoragePaths{}.WithDefaults(),
	}
	storage, err := NewStorage(cfg)
	require.NoError(t, err)

	// Create evidence directory
	evidenceDir := filepath.Join(tmpDir, "evidence", "ET-0001", "2025-Q4")
	require.NoError(t, os.MkdirAll(evidenceDir, 0755))

	// Create validation result
	result := &models.ValidationResult{
		TaskRef:             "ET-0001",
		Window:              "2025-Q4",
		Status:              "passed",
		ValidationMode:      "strict",
		CompletenessScore:   1.0,
		TotalChecks:         8,
		PassedChecks:        8,
		FailedChecks:        0,
		ReadyForSubmission:  true,
		ValidationTimestamp: time.Now(),
	}

	// Save validation result
	err = storage.SaveValidationResult("ET-0001", "2025-Q4", result)
	require.NoError(t, err)

	// Load validation result
	loaded, err := storage.LoadValidationResult("ET-0001", "2025-Q4")
	require.NoError(t, err)

	// Verify
	assert.Equal(t, result.TaskRef, loaded.TaskRef)
	assert.Equal(t, result.Window, loaded.Window)
	assert.Equal(t, result.Status, loaded.Status)
	assert.Equal(t, result.TotalChecks, loaded.TotalChecks)
	assert.Equal(t, result.PassedChecks, loaded.PassedChecks)
	assert.True(t, loaded.ReadyForSubmission)
}

func TestSubmissionStorage_AddSubmissionHistory(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.StorageConfig{
		DataDir: tmpDir,
		Paths:   config.StoragePaths{}.WithDefaults(),
	}
	storage, err := NewStorage(cfg)
	require.NoError(t, err)

	// Create evidence directory
	evidenceDir := filepath.Join(tmpDir, "evidence", "ET-0001", "2025-Q4")
	require.NoError(t, os.MkdirAll(evidenceDir, 0755))

	// Add history entries
	entry1 := models.SubmissionHistoryEntry{
		SubmissionID: "sub-001",
		SubmittedAt:  time.Now(),
		SubmittedBy:  "user1@example.com",
		Status:       "submitted",
		FileCount:    3,
	}

	entry2 := models.SubmissionHistoryEntry{
		SubmissionID: "sub-002",
		SubmittedAt:  time.Now().Add(1 * time.Hour),
		SubmittedBy:  "user2@example.com",
		Status:       "accepted",
		FileCount:    5,
	}

	err = storage.AddSubmissionHistory("ET-0001", "2025-Q4", entry1)
	require.NoError(t, err)

	err = storage.AddSubmissionHistory("ET-0001", "2025-Q4", entry2)
	require.NoError(t, err)

	// Load history
	history, err := storage.LoadSubmissionHistory("ET-0001", "2025-Q4")
	require.NoError(t, err)

	// Verify history (should be sorted by time, most recent first)
	assert.Len(t, history.Entries, 2)
	assert.Equal(t, "sub-002", history.Entries[0].SubmissionID) // Most recent
	assert.Equal(t, "sub-001", history.Entries[1].SubmissionID)
}

func TestSubmissionStorage_BatchOperations(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.StorageConfig{
		DataDir: tmpDir,
		Paths:   config.StoragePaths{}.WithDefaults(),
	}
	storage, err := NewStorage(cfg)
	require.NoError(t, err)

	// Create batch
	batch := &models.SubmissionBatch{
		BatchID:    "batch-001",
		BatchName:  "Q4 2025 Submissions",
		Status:     "draft",
		TaskRefs:   []string{"ET-0001", "ET-0047"},
		TotalTasks: 2,
		CreatedAt:  time.Now(),
		CreatedBy:  "test@example.com",
	}

	// Save batch
	err = storage.SaveBatch(batch)
	require.NoError(t, err)

	// Load batch
	loaded, err := storage.LoadBatch("batch-001")
	require.NoError(t, err)

	// Verify
	assert.Equal(t, batch.BatchID, loaded.BatchID)
	assert.Equal(t, batch.BatchName, loaded.BatchName)
	assert.Equal(t, batch.TotalTasks, loaded.TotalTasks)
	assert.Len(t, loaded.TaskRefs, 2)

	// List batches
	batches, err := storage.ListBatches()
	require.NoError(t, err)
	assert.Len(t, batches, 1)
	assert.Equal(t, "batch-001", batches[0].BatchID)
}

func TestSubmissionStorage_CalculateFileChecksum(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.StorageConfig{
		DataDir: tmpDir,
		Paths:   config.StoragePaths{}.WithDefaults(),
	}
	storage, err := NewStorage(cfg)
	require.NoError(t, err)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("test content")
	err = os.WriteFile(testFile, content, 0644)
	require.NoError(t, err)

	// Calculate checksum
	checksum, err := storage.CalculateFileChecksum(testFile)
	require.NoError(t, err)

	// Verify checksum is not empty and has expected format (64 hex chars)
	assert.NotEmpty(t, checksum)
	assert.Len(t, checksum, 64) // SHA256 = 64 hex chars

	// Calculate again - should be same
	checksum2, err := storage.CalculateFileChecksum(testFile)
	require.NoError(t, err)
	assert.Equal(t, checksum, checksum2)
}

func TestSubmissionStorage_InitializeSubmissionMetadata(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.StorageConfig{
		DataDir: tmpDir,
		Paths:   config.StoragePaths{}.WithDefaults(),
	}
	storage, err := NewStorage(cfg)
	require.NoError(t, err)

	// Create evidence directory
	evidenceDir := filepath.Join(tmpDir, "evidence", "ET-0001", "2025-Q4")
	require.NoError(t, os.MkdirAll(evidenceDir, 0755))

	// Initialize submission metadata
	err = storage.InitializeSubmissionMetadata("ET-0001", "2025-Q4")
	require.NoError(t, err)

	// Verify .submission directory exists
	submissionDir := filepath.Join(evidenceDir, submissionMetadataDir)
	assert.DirExists(t, submissionDir)

	// Verify submission.yaml was created
	submissionFile := filepath.Join(submissionDir, submissionFilename)
	assert.FileExists(t, submissionFile)

	// Should be able to load the submission
	submission, err := storage.LoadSubmission("ET-0001", "2025-Q4")
	require.NoError(t, err)
	assert.Equal(t, "ET-0001", submission.TaskRef)
	assert.Equal(t, "2025-Q4", submission.Window)
	assert.Equal(t, "draft", submission.Status)
}

func TestSubmissionStorage_SubmissionExists(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.StorageConfig{
		DataDir: tmpDir,
		Paths:   config.StoragePaths{}.WithDefaults(),
	}
	storage, err := NewStorage(cfg)
	require.NoError(t, err)

	// Should not exist initially
	exists := storage.SubmissionExists("ET-0001", "2025-Q4")
	assert.False(t, exists)

	// Create evidence directory and initialize
	evidenceDir := filepath.Join(tmpDir, "evidence", "ET-0001", "2025-Q4")
	require.NoError(t, os.MkdirAll(evidenceDir, 0755))
	err = storage.InitializeSubmissionMetadata("ET-0001", "2025-Q4")
	require.NoError(t, err)

	// Should exist now
	exists = storage.SubmissionExists("ET-0001", "2025-Q4")
	assert.True(t, exists)
}
