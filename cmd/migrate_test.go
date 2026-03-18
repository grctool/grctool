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

package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/grctool/grctool/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helper: write a JSON file into dir with the given name and content map.
func writeTestJSON(t *testing.T, dir, name string, data map[string]interface{}) string {
	t.Helper()
	path := filepath.Join(dir, name)
	b, err := json.MarshalIndent(data, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, b, 0644))
	return path
}

// helper: read a JSON file back into a map.
func readTestJSON(t *testing.T, path string) map[string]interface{} {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(b, &m))
	return m
}

func TestMigrateCommand_DryRun(t *testing.T) {
	dir := t.TempDir()
	numericFile := writeTestJSON(t, dir, "control.json", map[string]interface{}{
		"id":           float64(778773),
		"reference_id": "AC-01",
		"name":         "Test Control",
	})

	// Capture original bytes for comparison.
	origBytes, err := os.ReadFile(numericFile)
	require.NoError(t, err)

	result, err := services.MigrateJSONFiles(dir, true)
	require.NoError(t, err)

	assert.Equal(t, 1, result.FilesScanned)
	assert.Equal(t, 1, result.FilesModified) // reported as would-be-modified
	assert.Equal(t, 0, result.FilesFailed)
	assert.Len(t, result.Changes, 1)
	assert.Equal(t, "id", result.Changes[0].Field)
	assert.Equal(t, "778773", result.Changes[0].NewValue)

	// File must NOT have been altered.
	afterBytes, err := os.ReadFile(numericFile)
	require.NoError(t, err)
	assert.Equal(t, origBytes, afterBytes, "dry-run should not modify files")
}

func TestMigrateCommand_Execute(t *testing.T) {
	dir := t.TempDir()
	f := writeTestJSON(t, dir, "task.json", map[string]interface{}{
		"id":      float64(328001),
		"task_id": float64(99999),
		"name":    "Evidence Task",
	})

	result, err := services.MigrateJSONFiles(dir, false)
	require.NoError(t, err)

	assert.Equal(t, 1, result.FilesScanned)
	assert.Equal(t, 1, result.FilesModified)
	assert.Equal(t, 0, result.FilesFailed)
	assert.Len(t, result.Changes, 2)

	m := readTestJSON(t, f)
	assert.Equal(t, "328001", m["id"])
	assert.Equal(t, "99999", m["task_id"])
	assert.Equal(t, "Evidence Task", m["name"]) // unchanged
}

func TestMigrateCommand_AlreadyMigrated(t *testing.T) {
	dir := t.TempDir()
	writeTestJSON(t, dir, "policy.json", map[string]interface{}{
		"id":           "94641",
		"reference_id": "POL-001",
		"name":         "Privacy Policy",
	})

	result, err := services.MigrateJSONFiles(dir, false)
	require.NoError(t, err)

	assert.Equal(t, 1, result.FilesScanned)
	assert.Equal(t, 0, result.FilesModified)
	assert.Equal(t, 0, result.FilesFailed)
	assert.Empty(t, result.Changes)
}

func TestMigrateCommand_MixedFiles(t *testing.T) {
	dir := t.TempDir()

	// Numeric ID
	numFile := writeTestJSON(t, dir, "numeric.json", map[string]interface{}{
		"id":   float64(12345),
		"name": "Numeric",
	})

	// String ID
	strFile := writeTestJSON(t, dir, "string.json", map[string]interface{}{
		"id":   "67890",
		"name": "String",
	})

	result, err := services.MigrateJSONFiles(dir, false)
	require.NoError(t, err)

	assert.Equal(t, 2, result.FilesScanned)
	assert.Equal(t, 1, result.FilesModified)
	assert.Equal(t, 0, result.FilesFailed)
	assert.Len(t, result.Changes, 1)
	assert.Equal(t, numFile, result.Changes[0].FilePath)

	// Numeric file should now have string ID.
	m := readTestJSON(t, numFile)
	assert.Equal(t, "12345", m["id"])

	// String file should be untouched.
	m = readTestJSON(t, strFile)
	assert.Equal(t, "67890", m["id"])
}

func TestMigrateCommand_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	result, err := services.MigrateJSONFiles(dir, false)
	require.NoError(t, err)

	assert.Equal(t, 0, result.FilesScanned)
	assert.Equal(t, 0, result.FilesModified)
	assert.Equal(t, 0, result.FilesFailed)
	assert.Empty(t, result.Changes)
	assert.Empty(t, result.Errors)
}

func TestMigrateCommand_InvalidJSON(t *testing.T) {
	dir := t.TempDir()

	// Write a valid file and an invalid one.
	writeTestJSON(t, dir, "good.json", map[string]interface{}{
		"id":   float64(111),
		"name": "Good",
	})
	badPath := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(badPath, []byte("{invalid json"), 0644))

	result, err := services.MigrateJSONFiles(dir, false)
	require.NoError(t, err) // overall operation succeeds

	assert.Equal(t, 2, result.FilesScanned)
	assert.Equal(t, 1, result.FilesModified) // good.json
	assert.Equal(t, 1, result.FilesFailed)   // bad.json
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "bad.json")
}

func TestMigrateCommand_Idempotent(t *testing.T) {
	dir := t.TempDir()
	f := writeTestJSON(t, dir, "control.json", map[string]interface{}{
		"id":         float64(778773),
		"control_id": float64(42),
		"name":       "Test",
	})

	// First run: should make changes.
	r1, err := services.MigrateJSONFiles(dir, false)
	require.NoError(t, err)
	assert.Equal(t, 1, r1.FilesModified)
	assert.True(t, len(r1.Changes) > 0)

	// Verify conversion.
	m := readTestJSON(t, f)
	assert.Equal(t, "778773", m["id"])
	assert.Equal(t, "42", m["control_id"])

	// Second run: no changes.
	r2, err := services.MigrateJSONFiles(dir, false)
	require.NoError(t, err)
	assert.Equal(t, 1, r2.FilesScanned)
	assert.Equal(t, 0, r2.FilesModified)
	assert.Empty(t, r2.Changes)
}

func TestMigrateCommand_NestedObjects(t *testing.T) {
	dir := t.TempDir()
	f := writeTestJSON(t, dir, "task.json", map[string]interface{}{
		"id":   "328001",
		"name": "Task",
		"related_controls": []interface{}{
			map[string]interface{}{
				"id":           float64(778780),
				"reference_id": "CC6.1",
			},
		},
		"related_policies": []interface{}{
			map[string]interface{}{
				"id":           "94645",
				"reference_id": "POL-005",
			},
		},
	})

	result, err := services.MigrateJSONFiles(dir, false)
	require.NoError(t, err)

	assert.Equal(t, 1, result.FilesModified)
	assert.Len(t, result.Changes, 1)
	assert.Equal(t, "id", result.Changes[0].Field)
	assert.Equal(t, "778780", result.Changes[0].NewValue)

	m := readTestJSON(t, f)
	controls := m["related_controls"].([]interface{})
	ctrl := controls[0].(map[string]interface{})
	assert.Equal(t, "778780", ctrl["id"])

	// Already-string ID in related_policies should be untouched.
	policies := m["related_policies"].([]interface{})
	pol := policies[0].(map[string]interface{})
	assert.Equal(t, "94645", pol["id"])
}

func TestMigrateCommand_SubDirectories(t *testing.T) {
	dir := t.TempDir()

	// Create nested structure like real data dir.
	controlsDir := filepath.Join(dir, "docs", "controls", "json")
	require.NoError(t, os.MkdirAll(controlsDir, 0755))

	tasksDir := filepath.Join(dir, "docs", "evidence_tasks", "json")
	require.NoError(t, os.MkdirAll(tasksDir, 0755))

	writeTestJSON(t, controlsDir, "AC-01.json", map[string]interface{}{
		"id":   float64(100),
		"name": "Control",
	})
	writeTestJSON(t, tasksDir, "ET-001.json", map[string]interface{}{
		"id":   float64(200),
		"name": "Task",
	})

	result, err := services.MigrateJSONFiles(dir, false)
	require.NoError(t, err)

	assert.Equal(t, 2, result.FilesScanned)
	assert.Equal(t, 2, result.FilesModified)
	assert.Len(t, result.Changes, 2)
}
