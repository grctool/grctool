// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeJSONFile(t *testing.T, path string, v interface{}) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	data, err := json.Marshal(v)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0644))
}

func setupToolsStorage(t *testing.T) (*storage.Storage, string) {
	t.Helper()
	tmpDir := t.TempDir()
	cfg := config.StorageConfig{
		DataDir: tmpDir,
		Paths:   config.StoragePaths{}.WithDefaults(),
	}
	stor, err := storage.NewStorage(cfg)
	require.NoError(t, err)
	return stor, tmpDir
}

// ---- SimpleDataService tests ----

func TestSimpleDataService_GetEvidenceTask(t *testing.T) {
	t.Parallel()
	stor, tmpDir := setupToolsStorage(t)

	paths := config.StoragePaths{}.WithDefaults()
	etDir := filepath.Join(tmpDir, paths.EvidenceTasksJSON)
	writeJSONFile(t, filepath.Join(etDir, "et-001.json"), map[string]interface{}{
		"id":          "42",
		"name":        "Test Evidence Task",
		"description": "Test description",
	})

	svc := &SimpleDataService{storage: stor}
	task, err := svc.GetEvidenceTask(context.Background(), "42")
	require.NoError(t, err)
	assert.Equal(t, "Test Evidence Task", task.Name)
}

func TestSimpleDataService_GetEvidenceTask_NotFound(t *testing.T) {
	t.Parallel()
	stor, _ := setupToolsStorage(t)

	svc := &SimpleDataService{storage: stor}
	_, err := svc.GetEvidenceTask(context.Background(), "nonexistent")
	require.Error(t, err)
}

func TestSimpleDataService_GetAllEvidenceTasks(t *testing.T) {
	t.Parallel()
	stor, tmpDir := setupToolsStorage(t)

	paths := config.StoragePaths{}.WithDefaults()
	etDir := filepath.Join(tmpDir, paths.EvidenceTasksJSON)
	writeJSONFile(t, filepath.Join(etDir, "et-001.json"), map[string]interface{}{
		"id":   "1",
		"name": "Task One",
	})
	writeJSONFile(t, filepath.Join(etDir, "et-002.json"), map[string]interface{}{
		"id":   "2",
		"name": "Task Two",
	})

	svc := &SimpleDataService{storage: stor}
	tasks, err := svc.GetAllEvidenceTasks(context.Background())
	require.NoError(t, err)
	assert.Len(t, tasks, 2)
}

func TestSimpleDataService_GetAllEvidenceTasks_Empty(t *testing.T) {
	t.Parallel()
	stor, _ := setupToolsStorage(t)

	svc := &SimpleDataService{storage: stor}
	tasks, err := svc.GetAllEvidenceTasks(context.Background())
	require.NoError(t, err)
	assert.Empty(t, tasks)
}

func TestSimpleDataService_GetEvidenceRecords(t *testing.T) {
	t.Parallel()
	stor, _ := setupToolsStorage(t)

	svc := &SimpleDataService{storage: stor}
	records, err := svc.GetEvidenceRecords(context.Background(), "any-id")
	require.NoError(t, err)
	assert.Empty(t, records)
}

func TestSimpleDataService_GetPolicy(t *testing.T) {
	t.Parallel()
	stor, tmpDir := setupToolsStorage(t)

	paths := config.StoragePaths{}.WithDefaults()
	polDir := filepath.Join(tmpDir, paths.PoliciesJSON)
	writeJSONFile(t, filepath.Join(polDir, "pol-001.json"), map[string]interface{}{
		"id":          "99",
		"name":        "Access Control Policy",
		"description": "Defines access control requirements.",
	})

	svc := &SimpleDataService{storage: stor}
	policy, err := svc.GetPolicy(context.Background(), "99")
	require.NoError(t, err)
	assert.Equal(t, "Access Control Policy", policy.Name)
}

func TestSimpleDataService_GetPolicy_NotFound(t *testing.T) {
	t.Parallel()
	stor, _ := setupToolsStorage(t)

	svc := &SimpleDataService{storage: stor}
	_, err := svc.GetPolicy(context.Background(), "nonexistent")
	require.Error(t, err)
}

func TestSimpleDataService_GetControl(t *testing.T) {
	t.Parallel()
	stor, tmpDir := setupToolsStorage(t)

	paths := config.StoragePaths{}.WithDefaults()
	ctrlDir := filepath.Join(tmpDir, paths.ControlsJSON)
	writeJSONFile(t, filepath.Join(ctrlDir, "ctrl-001.json"), map[string]interface{}{
		"id":          "CC6.1",
		"name":        "Logical Access",
		"description": "Controls logical access to systems.",
	})

	svc := &SimpleDataService{storage: stor}
	ctrl, err := svc.GetControl(context.Background(), "CC6.1")
	require.NoError(t, err)
	assert.Equal(t, "Logical Access", ctrl.Name)
}

func TestSimpleDataService_GetControl_NotFound(t *testing.T) {
	t.Parallel()
	stor, _ := setupToolsStorage(t)

	svc := &SimpleDataService{storage: stor}
	_, err := svc.GetControl(context.Background(), "nonexistent")
	require.Error(t, err)
}
