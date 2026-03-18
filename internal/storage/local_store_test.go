// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func defaultPaths() config.StoragePaths {
	return config.StoragePaths{}.WithDefaults()
}

func newTestLocalDataStore(t *testing.T) *LocalDataStore {
	t.Helper()
	dir := t.TempDir()
	paths := defaultPaths().ResolveRelativeTo(dir)
	lds, err := NewLocalDataStore(dir, paths)
	require.NoError(t, err)
	return lds
}

func TestNewLocalDataStore(t *testing.T) {
	t.Parallel()

	t.Run("creates subdirectories", func(t *testing.T) {
		t.Parallel()
		lds := newTestLocalDataStore(t)
		assert.NotNil(t, lds)
		assert.True(t, lds.IsFallbackEnabled())
		assert.Contains(t, lds.GetDataSources(), "local_files")
	})
}

func TestLocalDataStore_IsDataAvailable_Empty(t *testing.T) {
	t.Parallel()
	lds := newTestLocalDataStore(t)
	assert.False(t, lds.IsDataAvailable())
}

func TestLocalDataStore_IsDataAvailable_WithData(t *testing.T) {
	t.Parallel()
	lds := newTestLocalDataStore(t)

	policy := testhelpers.SamplePolicy()
	require.NoError(t, lds.SavePolicy(policy))

	assert.True(t, lds.IsDataAvailable())
}

func TestLocalDataStore_PolicyOperations(t *testing.T) {
	t.Parallel()
	lds := newTestLocalDataStore(t)

	policy := testhelpers.SamplePolicy()
	require.NoError(t, lds.SavePolicy(policy))

	// GetPolicy by numeric ID
	retrieved, err := lds.GetPolicy(policy.ID)
	require.NoError(t, err)
	assert.Equal(t, policy.Name, retrieved.Name)
	assert.Equal(t, policy.ReferenceID, retrieved.ReferenceID)

	// GetPolicy not found
	_, err = lds.GetPolicy("99999")
	assert.Error(t, err)

	// GetAllPolicies
	all, err := lds.GetAllPolicies()
	require.NoError(t, err)
	assert.Len(t, all, 1)
	assert.Equal(t, policy.Name, all[0].Name)
}

func TestLocalDataStore_PolicyByReferenceAndID(t *testing.T) {
	t.Parallel()
	lds := newTestLocalDataStore(t)

	policy := testhelpers.SamplePolicy()
	require.NoError(t, lds.SavePolicy(policy))

	retrieved, err := lds.GetPolicyByReferenceAndID(policy.ReferenceID, policy.ID)
	require.NoError(t, err)
	assert.Equal(t, policy.Name, retrieved.Name)

	// Not found
	_, err = lds.GetPolicyByReferenceAndID("POL-9999", "0")
	assert.Error(t, err)
}

func TestLocalDataStore_PolicySummary(t *testing.T) {
	t.Parallel()
	lds := newTestLocalDataStore(t)

	p1 := testhelpers.SamplePolicy()
	p2 := &domain.Policy{
		ID: "22222", ReferenceID: "POL-0002", Name: "Data Protection",
		Framework: "ISO27001", Status: "draft",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, lds.SavePolicy(p1))
	require.NoError(t, lds.SavePolicy(p2))

	summary, err := lds.GetPolicySummary()
	require.NoError(t, err)
	assert.Equal(t, 2, summary.Total)
	assert.Equal(t, 1, summary.ByFramework["SOC2"])
	assert.Equal(t, 1, summary.ByFramework["ISO27001"])
	assert.Equal(t, 1, summary.ByStatus["active"])
	assert.Equal(t, 1, summary.ByStatus["draft"])
}

func TestLocalDataStore_ControlOperations(t *testing.T) {
	t.Parallel()
	lds := newTestLocalDataStore(t)

	control := testhelpers.SampleControl()
	require.NoError(t, lds.SaveControl(control))

	// GetControl by numeric ID
	retrieved, err := lds.GetControl(control.ID)
	require.NoError(t, err)
	assert.Equal(t, control.Name, retrieved.Name)

	// Not found
	_, err = lds.GetControl("99999")
	assert.Error(t, err)

	// GetAllControls
	all, err := lds.GetAllControls()
	require.NoError(t, err)
	assert.Len(t, all, 1)
}

func TestLocalDataStore_ControlByReferenceAndID(t *testing.T) {
	t.Parallel()
	lds := newTestLocalDataStore(t)

	control := testhelpers.SampleControl()
	require.NoError(t, lds.SaveControl(control))

	retrieved, err := lds.GetControlByReferenceAndID(control.ReferenceID, control.ID)
	require.NoError(t, err)
	assert.Equal(t, control.Name, retrieved.Name)

	_, err = lds.GetControlByReferenceAndID("XX-99", "0")
	assert.Error(t, err)
}

func TestLocalDataStore_ControlSummary(t *testing.T) {
	t.Parallel()
	lds := newTestLocalDataStore(t)

	c1 := testhelpers.SampleControl()
	c2 := &domain.Control{
		ID: "2002", ReferenceID: "AC-02", Name: "Network Security",
		Framework: "SOC2", Category: "Infrastructure", Status: "planned",
	}
	require.NoError(t, lds.SaveControl(c1))
	require.NoError(t, lds.SaveControl(c2))

	summary, err := lds.GetControlSummary()
	require.NoError(t, err)
	assert.Equal(t, 2, summary.Total)
	assert.Equal(t, 2, summary.ByFramework["SOC2"])
	assert.Equal(t, 1, summary.ByCategory["Common Criteria"])
	assert.Equal(t, 1, summary.ByCategory["Infrastructure"])
}

func TestLocalDataStore_EvidenceTaskOperations(t *testing.T) {
	t.Parallel()
	lds := newTestLocalDataStore(t)

	task := testhelpers.SampleEvidenceTask()
	require.NoError(t, lds.SaveEvidenceTask(task))

	// GetEvidenceTask by numeric ID
	retrieved, err := lds.GetEvidenceTask(task.ID)
	require.NoError(t, err)
	assert.Equal(t, task.Name, retrieved.Name)
	assert.Equal(t, task.ReferenceID, retrieved.ReferenceID)

	// Not found
	_, err = lds.GetEvidenceTask("99999")
	assert.Error(t, err)

	// GetAllEvidenceTasks
	all, err := lds.GetAllEvidenceTasks()
	require.NoError(t, err)
	assert.Len(t, all, 1)
}

func TestLocalDataStore_EvidenceTaskByReferenceAndID(t *testing.T) {
	t.Parallel()
	lds := newTestLocalDataStore(t)

	task := testhelpers.SampleEvidenceTask()
	require.NoError(t, lds.SaveEvidenceTask(task))

	retrieved, err := lds.GetEvidenceTaskByReferenceAndID(task.ReferenceID, task.ID)
	require.NoError(t, err)
	assert.Equal(t, task.Name, retrieved.Name)

	_, err = lds.GetEvidenceTaskByReferenceAndID("ET-9999", "0")
	assert.Error(t, err)
}

func TestLocalDataStore_EvidenceTaskSummary(t *testing.T) {
	t.Parallel()
	lds := newTestLocalDataStore(t)

	overdue := time.Now().Add(-48 * time.Hour)
	dueSoon := time.Now().Add(3 * 24 * time.Hour)

	t1 := testhelpers.SampleEvidenceTask()
	t1.NextDue = &overdue

	t2 := &domain.EvidenceTask{
		ID: "327993", ReferenceID: "ET-0048", Name: "Terraform Security",
		Status: "in_progress", Priority: "medium", NextDue: &dueSoon,
	}

	require.NoError(t, lds.SaveEvidenceTask(t1))
	require.NoError(t, lds.SaveEvidenceTask(t2))

	summary, err := lds.GetEvidenceTaskSummary()
	require.NoError(t, err)
	assert.Equal(t, 2, summary.Total)
	assert.Equal(t, 1, summary.Overdue)
	assert.Equal(t, 1, summary.DueSoon)
}

func TestLocalDataStore_EvidenceRecordOperations(t *testing.T) {
	t.Parallel()
	lds := newTestLocalDataStore(t)

	record := testhelpers.SampleEvidenceRecord()
	require.NoError(t, lds.SaveEvidenceRecord(record))

	// GetEvidenceRecord
	retrieved, err := lds.GetEvidenceRecord(record.ID)
	require.NoError(t, err)
	assert.Equal(t, record.Title, retrieved.Title)

	// Not found
	_, err = lds.GetEvidenceRecord("nonexistent")
	assert.Error(t, err)

	// GetEvidenceRecordsByTaskID
	records, err := lds.GetEvidenceRecordsByTaskID(record.TaskID)
	require.NoError(t, err)
	assert.Len(t, records, 1)

	// No records for other task
	records, err = lds.GetEvidenceRecordsByTaskID("99999")
	require.NoError(t, err)
	assert.Empty(t, records)
}

func TestLocalDataStore_Stats(t *testing.T) {
	t.Parallel()
	lds := newTestLocalDataStore(t)

	require.NoError(t, lds.SavePolicy(testhelpers.SamplePolicy()))
	require.NoError(t, lds.SaveControl(testhelpers.SampleControl()))
	require.NoError(t, lds.SaveEvidenceTask(testhelpers.SampleEvidenceTask()))

	stats, err := lds.GetStats()
	require.NoError(t, err)
	assert.Equal(t, 1, stats["total_policies"])
	assert.Equal(t, 1, stats["total_controls"])
	assert.Equal(t, 1, stats["total_evidence_tasks"])
	assert.Equal(t, true, stats["fallback_enabled"])
}

func TestLocalDataStore_SyncTime(t *testing.T) {
	t.Parallel()
	lds := newTestLocalDataStore(t)

	syncTime := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	require.NoError(t, lds.SetSyncTime("policies", syncTime))

	retrieved, err := lds.GetSyncTime("policies")
	require.NoError(t, err)
	assert.Equal(t, syncTime, retrieved)

	// Non-existent sync type
	_, err = lds.GetSyncTime("nonexistent")
	assert.Error(t, err)
}

func TestLocalDataStore_Clear(t *testing.T) {
	t.Parallel()
	lds := newTestLocalDataStore(t)

	require.NoError(t, lds.SavePolicy(testhelpers.SamplePolicy()))
	require.NoError(t, lds.SaveControl(testhelpers.SampleControl()))
	require.NoError(t, lds.SaveEvidenceTask(testhelpers.SampleEvidenceTask()))

	err := lds.Clear()
	require.NoError(t, err)

	policies, err := lds.GetAllPolicies()
	require.NoError(t, err)
	assert.Empty(t, policies)

	controls, err := lds.GetAllControls()
	require.NoError(t, err)
	assert.Empty(t, controls)

	tasks, err := lds.GetAllEvidenceTasks()
	require.NoError(t, err)
	assert.Empty(t, tasks)
}

func TestLocalDataStore_ValidateDataIntegrity(t *testing.T) {
	t.Parallel()
	lds := newTestLocalDataStore(t)

	// Empty store should validate fine
	err := lds.ValidateDataIntegrity()
	require.NoError(t, err)

	// Store valid data
	require.NoError(t, lds.SavePolicy(testhelpers.SamplePolicy()))
	require.NoError(t, lds.SaveControl(testhelpers.SampleControl()))
	require.NoError(t, lds.SaveEvidenceTask(testhelpers.SampleEvidenceTask()))

	err = lds.ValidateDataIntegrity()
	require.NoError(t, err)
}

func TestLocalDataStore_ValidateDataIntegrity_CorruptFile(t *testing.T) {
	t.Parallel()
	lds := newTestLocalDataStore(t)

	// Save a valid policy first
	require.NoError(t, lds.SavePolicy(testhelpers.SamplePolicy()))

	// Corrupt the file by writing invalid JSON
	policiesDir := filepath.Join(lds.baseDir, lds.policiesPath)
	entries, err := os.ReadDir(policiesDir)
	require.NoError(t, err)
	require.NotEmpty(t, entries)
	corruptPath := filepath.Join(policiesDir, entries[0].Name())
	require.NoError(t, os.WriteFile(corruptPath, []byte("{invalid json!!!"), 0644))

	err = lds.ValidateDataIntegrity()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid policy file")
}

func TestLocalDataStore_FallbackEnabled(t *testing.T) {
	t.Parallel()
	lds := newTestLocalDataStore(t)

	assert.True(t, lds.IsFallbackEnabled())

	lds.SetFallbackEnabled(false)
	assert.False(t, lds.IsFallbackEnabled())

	lds.SetFallbackEnabled(true)
	assert.True(t, lds.IsFallbackEnabled())
}

func TestLocalDataStore_ImportExport(t *testing.T) {
	t.Parallel()

	// Create source store with data
	sourceStore := newTestLocalDataStore(t)
	require.NoError(t, sourceStore.SavePolicy(testhelpers.SamplePolicy()))
	require.NoError(t, sourceStore.SaveControl(testhelpers.SampleControl()))
	require.NoError(t, sourceStore.SaveEvidenceTask(testhelpers.SampleEvidenceTask()))

	// Export to a temp directory
	exportDir := t.TempDir()
	err := sourceStore.ExportData(exportDir)
	require.NoError(t, err)

	// Create a fresh store and import
	targetStore := newTestLocalDataStore(t)
	err = targetStore.ImportData(exportDir)
	require.NoError(t, err)

	// Verify imported data
	policies, err := targetStore.GetAllPolicies()
	require.NoError(t, err)
	assert.Len(t, policies, 1)

	controls, err := targetStore.GetAllControls()
	require.NoError(t, err)
	assert.Len(t, controls, 1)

	tasks, err := targetStore.GetAllEvidenceTasks()
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
}

func TestLocalDataStore_ImportData_NonExistentSource(t *testing.T) {
	t.Parallel()
	lds := newTestLocalDataStore(t)

	err := lds.ImportData("/nonexistent/path/that/does/not/exist")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestLocalDataStore_GetLastDataUpdate(t *testing.T) {
	t.Parallel()
	lds := newTestLocalDataStore(t)

	// No data => error
	_, err := lds.GetLastDataUpdate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no data files found")

	// Add data
	require.NoError(t, lds.SavePolicy(testhelpers.SamplePolicy()))

	lastUpdate, err := lds.GetLastDataUpdate()
	require.NoError(t, err)
	assert.False(t, lastUpdate.IsZero())
	// Should be recent (within last minute)
	assert.WithinDuration(t, time.Now(), lastUpdate, 1*time.Minute)
}

func TestLocalDataStore_GetBaseDir(t *testing.T) {
	t.Parallel()
	lds := newTestLocalDataStore(t)
	assert.NotEmpty(t, lds.GetBaseDir())
	assert.DirExists(t, lds.GetBaseDir())
}

// TestLocalDataStore_ValidateDataIntegrity_CorruptControl ensures corrupt
// control files are detected.
func TestLocalDataStore_ValidateDataIntegrity_CorruptControl(t *testing.T) {
	t.Parallel()
	lds := newTestLocalDataStore(t)

	require.NoError(t, lds.SaveControl(testhelpers.SampleControl()))

	controlsDir := filepath.Join(lds.baseDir, lds.controlsPath)
	entries, err := os.ReadDir(controlsDir)
	require.NoError(t, err)
	require.NotEmpty(t, entries)
	corruptPath := filepath.Join(controlsDir, entries[0].Name())
	require.NoError(t, os.WriteFile(corruptPath, []byte("not json"), 0644))

	err = lds.ValidateDataIntegrity()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid control file")
}

// TestLocalDataStore_ValidateDataIntegrity_CorruptEvidenceTask ensures corrupt
// evidence task files are detected.
func TestLocalDataStore_ValidateDataIntegrity_CorruptEvidenceTask(t *testing.T) {
	t.Parallel()
	lds := newTestLocalDataStore(t)

	require.NoError(t, lds.SaveEvidenceTask(testhelpers.SampleEvidenceTask()))

	tasksDir := filepath.Join(lds.baseDir, lds.evidenceTasksPath)
	entries, err := os.ReadDir(tasksDir)
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	// Find the .json file
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".json" {
			corruptPath := filepath.Join(tasksDir, entry.Name())
			require.NoError(t, os.WriteFile(corruptPath, []byte("broken"), 0644))
			break
		}
	}

	err = lds.ValidateDataIntegrity()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid evidence task file")
}

// TestLocalDataStore_GetAllPolicies_FallbackDisabled verifies behavior when
// fallback is disabled and data directories are problematic.
func TestLocalDataStore_ExportData_EmptyStore(t *testing.T) {
	t.Parallel()
	lds := newTestLocalDataStore(t)

	exportDir := t.TempDir()
	err := lds.ExportData(exportDir)
	require.NoError(t, err)

	// Export directories should exist but be empty of JSON files
	assert.DirExists(t, filepath.Join(exportDir, lds.policiesPath))
}

// Test the copyFile helper indirectly through import/export.
func TestLocalDataStore_ImportExport_RoundTrip(t *testing.T) {
	t.Parallel()
	lds := newTestLocalDataStore(t)

	// Create a sample evidence record too
	record := testhelpers.SampleEvidenceRecord()
	require.NoError(t, lds.SaveEvidenceRecord(record))

	exportDir := t.TempDir()
	require.NoError(t, lds.ExportData(exportDir))

	target := newTestLocalDataStore(t)
	require.NoError(t, target.ImportData(exportDir))

	imported, err := target.GetEvidenceRecord(record.ID)
	require.NoError(t, err)
	assert.Equal(t, record.Title, imported.Title)
}

// TestCopyFile_WritesCorrectContent tests the internal copyFile helper.
func TestCopyFile_WritesCorrectContent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	srcPath := filepath.Join(dir, "source.json")
	data := map[string]string{"hello": "world"}
	jsonBytes, err := json.Marshal(data)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(srcPath, jsonBytes, 0644))

	dstPath := filepath.Join(dir, "dest.json")
	err = copyFile(srcPath, dstPath)
	require.NoError(t, err)

	read, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, jsonBytes, read)
}
