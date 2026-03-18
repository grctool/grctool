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

// Regression tests verifying the int→string ID type migration works correctly
// end-to-end across domain unmarshaling, adapter conversion, and the migration tool.
//
// Bead: grct-3gl.18
package integration_test

import (
	stdjson "encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/grctool/grctool/internal/adapters"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/services"
	tugboatmodels "github.com/grctool/grctool/internal/tugboat/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// 1. Control – JSON backward compatibility
// ---------------------------------------------------------------------------

func TestIDMigration_Control_UnmarshalJSON_NumericID(t *testing.T) {
	t.Parallel()
	raw := `{"id": 12345, "name": "Test Control", "reference_id": "CC-06.1"}`
	var c domain.Control
	err := stdjson.Unmarshal([]byte(raw), &c)
	require.NoError(t, err)
	assert.Equal(t, "12345", c.ID, "numeric id must be converted to string")
	assert.Equal(t, "CC-06.1", c.ReferenceID)
	assert.Equal(t, "Test Control", c.Name)
}

func TestIDMigration_Control_UnmarshalJSON_StringID(t *testing.T) {
	t.Parallel()
	raw := `{"id": "12345", "name": "Test Control"}`
	var c domain.Control
	err := stdjson.Unmarshal([]byte(raw), &c)
	require.NoError(t, err)
	assert.Equal(t, "12345", c.ID, "string id must be preserved as-is")
}

func TestIDMigration_Control_UnmarshalJSON_ZeroID(t *testing.T) {
	t.Parallel()
	raw := `{"id": 0, "name": "Zero Control"}`
	var c domain.Control
	err := stdjson.Unmarshal([]byte(raw), &c)
	require.NoError(t, err)
	assert.Equal(t, "0", c.ID)
}

func TestIDMigration_Control_UnmarshalJSON_NullID(t *testing.T) {
	t.Parallel()
	raw := `{"id": null, "name": "Null Control"}`
	var c domain.Control
	err := stdjson.Unmarshal([]byte(raw), &c)
	require.NoError(t, err)
	assert.Equal(t, "", c.ID, "null id must result in empty string")
}

func TestIDMigration_Control_UnmarshalJSON_FloatID(t *testing.T) {
	t.Parallel()
	raw := `{"id": 12345.0, "name": "Float Control"}`
	var c domain.Control
	err := stdjson.Unmarshal([]byte(raw), &c)
	require.NoError(t, err)
	assert.Equal(t, "12345", c.ID, "float id must be converted to clean string")
}

func TestIDMigration_Control_UnmarshalJSON_MissingID(t *testing.T) {
	t.Parallel()
	raw := `{"name": "No ID Control"}`
	var c domain.Control
	err := stdjson.Unmarshal([]byte(raw), &c)
	require.NoError(t, err)
	assert.Equal(t, "", c.ID, "missing id field must result in empty string")
}

func TestIDMigration_Control_UnmarshalJSON_OtherFieldsPreserved(t *testing.T) {
	t.Parallel()
	raw := `{
		"id": 999,
		"name": "Full Control",
		"reference_id": "CC-06.1",
		"description": "A description",
		"category": "Security",
		"framework": "SOC2",
		"status": "implemented",
		"is_auto_implemented": true,
		"master_version_num": 3,
		"master_control_id": 42,
		"org_id": 7,
		"org_scope_id": 11
	}`
	var c domain.Control
	err := stdjson.Unmarshal([]byte(raw), &c)
	require.NoError(t, err)
	assert.Equal(t, "999", c.ID)
	assert.Equal(t, "Full Control", c.Name)
	assert.Equal(t, "CC-06.1", c.ReferenceID)
	assert.Equal(t, "A description", c.Description)
	assert.Equal(t, "Security", c.Category)
	assert.Equal(t, "SOC2", c.Framework)
	assert.Equal(t, "implemented", c.Status)
	assert.True(t, c.IsAutoImplemented)
	assert.Equal(t, 3, c.MasterVersionNum)
	assert.Equal(t, 42, c.MasterControlID)
	assert.Equal(t, 7, c.OrgID)
	assert.Equal(t, 11, c.OrgScopeID)
}

// ---------------------------------------------------------------------------
// 2. EvidenceTask – JSON backward compatibility
// ---------------------------------------------------------------------------

func TestIDMigration_EvidenceTask_UnmarshalJSON_NumericID(t *testing.T) {
	t.Parallel()
	raw := `{"id": 54321, "name": "Task A", "description": "desc"}`
	var et domain.EvidenceTask
	err := stdjson.Unmarshal([]byte(raw), &et)
	require.NoError(t, err)
	assert.Equal(t, "54321", et.ID)
	assert.Equal(t, "Task A", et.Name)
}

func TestIDMigration_EvidenceTask_UnmarshalJSON_StringID(t *testing.T) {
	t.Parallel()
	raw := `{"id": "54321", "name": "Task A"}`
	var et domain.EvidenceTask
	err := stdjson.Unmarshal([]byte(raw), &et)
	require.NoError(t, err)
	assert.Equal(t, "54321", et.ID)
}

func TestIDMigration_EvidenceTask_UnmarshalJSON_ZeroID(t *testing.T) {
	t.Parallel()
	raw := `{"id": 0, "name": "Zero Task"}`
	var et domain.EvidenceTask
	err := stdjson.Unmarshal([]byte(raw), &et)
	require.NoError(t, err)
	assert.Equal(t, "0", et.ID)
}

func TestIDMigration_EvidenceTask_UnmarshalJSON_NullID(t *testing.T) {
	t.Parallel()
	raw := `{"id": null, "name": "Null Task"}`
	var et domain.EvidenceTask
	err := stdjson.Unmarshal([]byte(raw), &et)
	require.NoError(t, err)
	assert.Equal(t, "", et.ID)
}

func TestIDMigration_EvidenceTask_UnmarshalJSON_FloatID(t *testing.T) {
	t.Parallel()
	raw := `{"id": 54321.0, "name": "Float Task"}`
	var et domain.EvidenceTask
	err := stdjson.Unmarshal([]byte(raw), &et)
	require.NoError(t, err)
	assert.Equal(t, "54321", et.ID)
}

// ---------------------------------------------------------------------------
// 3. EvidenceRecord – JSON backward compatibility (task_id field)
// ---------------------------------------------------------------------------

func TestIDMigration_EvidenceRecord_UnmarshalJSON_NumericTaskID(t *testing.T) {
	t.Parallel()
	raw := `{"id": "rec-1", "task_id": 99999, "title": "Record A"}`
	var er domain.EvidenceRecord
	err := stdjson.Unmarshal([]byte(raw), &er)
	require.NoError(t, err)
	assert.Equal(t, "99999", er.TaskID, "numeric task_id must be converted to string")
	assert.Equal(t, "rec-1", er.ID)
}

func TestIDMigration_EvidenceRecord_UnmarshalJSON_StringTaskID(t *testing.T) {
	t.Parallel()
	raw := `{"id": "rec-1", "task_id": "99999", "title": "Record A"}`
	var er domain.EvidenceRecord
	err := stdjson.Unmarshal([]byte(raw), &er)
	require.NoError(t, err)
	assert.Equal(t, "99999", er.TaskID)
}

func TestIDMigration_EvidenceRecord_UnmarshalJSON_NullTaskID(t *testing.T) {
	t.Parallel()
	raw := `{"id": "rec-1", "task_id": null, "title": "Record A"}`
	var er domain.EvidenceRecord
	err := stdjson.Unmarshal([]byte(raw), &er)
	require.NoError(t, err)
	assert.Equal(t, "", er.TaskID)
}

func TestIDMigration_EvidenceRecord_UnmarshalJSON_FloatTaskID(t *testing.T) {
	t.Parallel()
	raw := `{"id": "rec-1", "task_id": 99999.0, "title": "Record A"}`
	var er domain.EvidenceRecord
	err := stdjson.Unmarshal([]byte(raw), &er)
	require.NoError(t, err)
	assert.Equal(t, "99999", er.TaskID)
}

// ---------------------------------------------------------------------------
// 4. EvidenceTaskState – JSON backward compatibility (task_id field)
// ---------------------------------------------------------------------------

func TestIDMigration_EvidenceTaskState_UnmarshalJSON_NumericTaskID(t *testing.T) {
	t.Parallel()
	raw := `{"task_ref": "ET-0001", "task_id": 327992, "task_name": "Access Controls"}`
	var s models.EvidenceTaskState
	err := stdjson.Unmarshal([]byte(raw), &s)
	require.NoError(t, err)
	assert.Equal(t, "327992", s.TaskID, "numeric task_id must be converted to string")
	assert.Equal(t, "ET-0001", s.TaskRef)
}

func TestIDMigration_EvidenceTaskState_UnmarshalJSON_StringTaskID(t *testing.T) {
	t.Parallel()
	raw := `{"task_ref": "ET-0001", "task_id": "327992", "task_name": "Access Controls"}`
	var s models.EvidenceTaskState
	err := stdjson.Unmarshal([]byte(raw), &s)
	require.NoError(t, err)
	assert.Equal(t, "327992", s.TaskID)
}

func TestIDMigration_EvidenceTaskState_UnmarshalJSON_NullTaskID(t *testing.T) {
	t.Parallel()
	raw := `{"task_ref": "ET-0001", "task_id": null, "task_name": "Access Controls"}`
	var s models.EvidenceTaskState
	err := stdjson.Unmarshal([]byte(raw), &s)
	require.NoError(t, err)
	assert.Equal(t, "", s.TaskID)
}

func TestIDMigration_EvidenceTaskState_UnmarshalJSON_FloatTaskID(t *testing.T) {
	t.Parallel()
	raw := `{"task_ref": "ET-0001", "task_id": 327992.0, "task_name": "Access Controls"}`
	var s models.EvidenceTaskState
	err := stdjson.Unmarshal([]byte(raw), &s)
	require.NoError(t, err)
	assert.Equal(t, "327992", s.TaskID)
}

// ---------------------------------------------------------------------------
// 5. Adapter conversion – Tugboat API int → domain string
// ---------------------------------------------------------------------------

func TestIDMigration_AdapterConvertsControlIntToString(t *testing.T) {
	t.Parallel()
	adapter := adapters.NewTugboatToDomain()
	apiControl := tugboatmodels.Control{
		ID:   99999,
		Name: "API Control",
	}
	domainControl := adapter.ConvertControl(apiControl)
	assert.Equal(t, "99999", domainControl.ID, "adapter must convert int ID to string")
	assert.Equal(t, "99999", domainControl.ExternalIDs["tugboat"], "tugboat external ID must match")
}

func TestIDMigration_AdapterConvertsEvidenceTaskIntToString(t *testing.T) {
	t.Parallel()
	adapter := adapters.NewTugboatToDomain()
	apiTask := tugboatmodels.EvidenceTask{
		ID:   88888,
		Name: "API Task",
	}
	domainTask := adapter.ConvertEvidenceTask(apiTask)
	assert.Equal(t, "88888", domainTask.ID, "adapter must convert int ID to string")
	assert.Equal(t, "88888", domainTask.ExternalIDs["tugboat"], "tugboat external ID must match")
}

func TestIDMigration_AdapterConvertsZeroID(t *testing.T) {
	t.Parallel()
	adapter := adapters.NewTugboatToDomain()
	apiControl := tugboatmodels.Control{
		ID:   0,
		Name: "Zero ID Control",
	}
	domainControl := adapter.ConvertControl(apiControl)
	assert.Equal(t, "0", domainControl.ID, "zero int ID must become string \"0\"")
}

// ---------------------------------------------------------------------------
// 6. MigrateJSONFiles – migration tool converts numeric IDs to strings
// ---------------------------------------------------------------------------

func TestIDMigration_MigrateToolConvertsFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Write a legacy control file with numeric ID.
	controlJSON := `{
  "id": 12345,
  "name": "Legacy Control",
  "reference_id": "CC-06.1"
}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "control.json"), []byte(controlJSON), 0644))

	// Write a legacy evidence task file with numeric ID.
	taskJSON := `{
  "id": 54321,
  "task_id": 54321,
  "name": "Legacy Task"
}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "task.json"), []byte(taskJSON), 0644))

	// Run migration.
	result, err := services.MigrateJSONFiles(dir, false)
	require.NoError(t, err)
	assert.Equal(t, 2, result.FilesScanned)
	assert.Equal(t, 2, result.FilesModified)
	assert.Equal(t, 0, result.FilesFailed)
	assert.NotEmpty(t, result.Changes, "should have at least one change")

	// Verify the control file was migrated.
	controlData, err := os.ReadFile(filepath.Join(dir, "control.json"))
	require.NoError(t, err)
	var controlMap map[string]interface{}
	require.NoError(t, stdjson.Unmarshal(controlData, &controlMap))
	assert.Equal(t, "12345", controlMap["id"], "control id must be string after migration")

	// Verify the task file was migrated.
	taskData, err := os.ReadFile(filepath.Join(dir, "task.json"))
	require.NoError(t, err)
	var taskMap map[string]interface{}
	require.NoError(t, stdjson.Unmarshal(taskData, &taskMap))
	assert.Equal(t, "54321", taskMap["id"], "task id must be string after migration")
	assert.Equal(t, "54321", taskMap["task_id"], "task_id must be string after migration")
}

func TestIDMigration_MigrateToolIdempotent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Write a file with numeric IDs.
	raw := `{"id": 111, "name": "Idempotent Test"}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "test.json"), []byte(raw), 0644))

	// First migration.
	result1, err := services.MigrateJSONFiles(dir, false)
	require.NoError(t, err)
	assert.Equal(t, 1, result1.FilesModified)

	// Second migration — should find 0 changes (idempotent).
	result2, err := services.MigrateJSONFiles(dir, false)
	require.NoError(t, err)
	assert.Equal(t, 0, result2.FilesModified, "second run must be idempotent, 0 changes")
	assert.Empty(t, result2.Changes)
}

func TestIDMigration_MigrateToolDryRun(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	raw := `{"id": 222, "name": "DryRun Test"}`
	fpath := filepath.Join(dir, "test.json")
	require.NoError(t, os.WriteFile(fpath, []byte(raw), 0644))

	// Dry run — reports changes but does NOT modify.
	result, err := services.MigrateJSONFiles(dir, true)
	require.NoError(t, err)
	assert.Equal(t, 1, result.FilesModified, "dry run should still count as modified")
	assert.NotEmpty(t, result.Changes)

	// Verify the file was NOT modified.
	data, err := os.ReadFile(fpath)
	require.NoError(t, err)
	assert.Equal(t, raw, string(data), "dry run must not write to disk")
}

func TestIDMigration_MigrateToolNestedIDs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Nested structure with numeric IDs at multiple levels.
	raw := `{
  "id": 100,
  "name": "Parent",
  "related_controls": [
    {"id": 200, "name": "Child Control"}
  ]
}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "nested.json"), []byte(raw), 0644))

	result, err := services.MigrateJSONFiles(dir, false)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(result.Changes), 2, "must find IDs at both levels")

	data, err := os.ReadFile(filepath.Join(dir, "nested.json"))
	require.NoError(t, err)
	var root map[string]interface{}
	require.NoError(t, stdjson.Unmarshal(data, &root))
	assert.Equal(t, "100", root["id"])

	controls := root["related_controls"].([]interface{})
	child := controls[0].(map[string]interface{})
	assert.Equal(t, "200", child["id"], "nested id must also be migrated")
}

// ---------------------------------------------------------------------------
// 7. Round-trip: marshal → unmarshal preserves string IDs
// ---------------------------------------------------------------------------

func TestIDMigration_Control_RoundTrip(t *testing.T) {
	t.Parallel()
	original := domain.Control{
		ID:          "12345",
		ReferenceID: "CC-06.1",
		Name:        "Round Trip Control",
		Description: "A test control",
		Framework:   "SOC2",
		Status:      "implemented",
		ExternalIDs: map[string]string{"tugboat": "12345", "accountablehq": "abc-123"},
	}

	data, err := stdjson.Marshal(original)
	require.NoError(t, err)

	var restored domain.Control
	err = stdjson.Unmarshal(data, &restored)
	require.NoError(t, err)
	assert.Equal(t, original.ID, restored.ID)
	assert.Equal(t, original.ReferenceID, restored.ReferenceID)
	assert.Equal(t, original.Name, restored.Name)
	assert.Equal(t, original.ExternalIDs, restored.ExternalIDs)
}

func TestIDMigration_EvidenceTask_RoundTrip(t *testing.T) {
	t.Parallel()
	original := domain.EvidenceTask{
		ID:          "54321",
		ReferenceID: "ET1",
		Name:        "Round Trip Task",
		Description: "A test task",
		ExternalIDs: map[string]string{"tugboat": "54321"},
	}

	data, err := stdjson.Marshal(original)
	require.NoError(t, err)

	var restored domain.EvidenceTask
	err = stdjson.Unmarshal(data, &restored)
	require.NoError(t, err)
	assert.Equal(t, original.ID, restored.ID)
	assert.Equal(t, original.ReferenceID, restored.ReferenceID)
	assert.Equal(t, original.Name, restored.Name)
	assert.Equal(t, original.ExternalIDs, restored.ExternalIDs)
}

func TestIDMigration_EvidenceRecord_RoundTrip(t *testing.T) {
	t.Parallel()
	original := domain.EvidenceRecord{
		ID:     "rec-001",
		TaskID: "54321",
		Title:  "Round Trip Record",
	}

	data, err := stdjson.Marshal(original)
	require.NoError(t, err)

	var restored domain.EvidenceRecord
	err = stdjson.Unmarshal(data, &restored)
	require.NoError(t, err)
	assert.Equal(t, original.ID, restored.ID)
	assert.Equal(t, original.TaskID, restored.TaskID)
}

// ---------------------------------------------------------------------------
// 8. ExternalIDs preservation
// ---------------------------------------------------------------------------

func TestIDMigration_ExternalIDsSurviveRoundTrip(t *testing.T) {
	t.Parallel()
	original := domain.Control{
		ID:   "42",
		Name: "ExternalIDs Test",
		ExternalIDs: map[string]string{
			"tugboat":       "42",
			"accountablehq": "ctrl-abc-123",
			"custom":        "xyz",
		},
	}

	data, err := stdjson.Marshal(original)
	require.NoError(t, err)

	var restored domain.Control
	err = stdjson.Unmarshal(data, &restored)
	require.NoError(t, err)
	assert.Equal(t, original.ExternalIDs, restored.ExternalIDs)
	assert.Equal(t, "42", restored.ExternalIDs["tugboat"])
	assert.Equal(t, "ctrl-abc-123", restored.ExternalIDs["accountablehq"])
	assert.Equal(t, "xyz", restored.ExternalIDs["custom"])
}

// ---------------------------------------------------------------------------
// 9. SyncMetadata preservation
// ---------------------------------------------------------------------------

func TestIDMigration_SyncMetadataSurvivesRoundTrip(t *testing.T) {
	t.Parallel()
	raw := `{
		"id": "42",
		"name": "SyncMeta Test",
		"sync_metadata": {
			"last_sync_time": {"tugboat": "2025-01-15T10:00:00Z"},
			"content_hash": {"tugboat": "abc123"},
			"conflict_state": ""
		}
	}`

	var c domain.Control
	err := stdjson.Unmarshal([]byte(raw), &c)
	require.NoError(t, err)
	assert.Equal(t, "42", c.ID)
	require.NotNil(t, c.SyncMetadata)
	assert.Contains(t, c.SyncMetadata.ContentHash, "tugboat")
	assert.Equal(t, "abc123", c.SyncMetadata.ContentHash["tugboat"])

	// Round-trip
	data, err := stdjson.Marshal(c)
	require.NoError(t, err)

	var restored domain.Control
	err = stdjson.Unmarshal(data, &restored)
	require.NoError(t, err)
	assert.Equal(t, c.SyncMetadata.ContentHash, restored.SyncMetadata.ContentHash)
}

// ---------------------------------------------------------------------------
// 10. Edge cases
// ---------------------------------------------------------------------------

func TestIDMigration_EmptyStringID(t *testing.T) {
	t.Parallel()
	raw := `{"id": "", "name": "Empty ID"}`
	var c domain.Control
	err := stdjson.Unmarshal([]byte(raw), &c)
	require.NoError(t, err)
	assert.Equal(t, "", c.ID)
}

func TestIDMigration_VeryLargeNumericID(t *testing.T) {
	t.Parallel()
	raw := `{"id": 9999999999, "name": "Large ID"}`
	var c domain.Control
	err := stdjson.Unmarshal([]byte(raw), &c)
	require.NoError(t, err)
	assert.Equal(t, "9999999999", c.ID)
}

func TestIDMigration_NegativeNumericID(t *testing.T) {
	t.Parallel()
	raw := `{"id": -1, "name": "Negative ID"}`
	var c domain.Control
	err := stdjson.Unmarshal([]byte(raw), &c)
	require.NoError(t, err)
	assert.Equal(t, "-1", c.ID)
}

func TestIDMigration_MixedFieldTypes(t *testing.T) {
	t.Parallel()
	// Some fields numeric, some string — simulates partially-migrated file.
	raw := `{
		"id": 77777,
		"name": "Mixed Types",
		"reference_id": "CC-01.1",
		"master_version_num": 5,
		"org_id": 10
	}`
	var c domain.Control
	err := stdjson.Unmarshal([]byte(raw), &c)
	require.NoError(t, err)
	assert.Equal(t, "77777", c.ID, "numeric id must become string")
	assert.Equal(t, "CC-01.1", c.ReferenceID, "string fields must stay strings")
	assert.Equal(t, 5, c.MasterVersionNum, "int fields must stay ints")
	assert.Equal(t, 10, c.OrgID, "int fields must stay ints")
}

func TestIDMigration_BooleanID_ReturnsError(t *testing.T) {
	t.Parallel()
	raw := `{"id": true, "name": "Boolean ID"}`
	var c domain.Control
	err := stdjson.Unmarshal([]byte(raw), &c)
	assert.Error(t, err, "boolean id must fail unmarshaling")
}

func TestIDMigration_ArrayID_ReturnsError(t *testing.T) {
	t.Parallel()
	raw := `{"id": [1, 2], "name": "Array ID"}`
	var c domain.Control
	err := stdjson.Unmarshal([]byte(raw), &c)
	assert.Error(t, err, "array id must fail unmarshaling")
}

// ---------------------------------------------------------------------------
// 11. Legacy file loading – write raw numeric-ID JSON, unmarshal via domain types
// ---------------------------------------------------------------------------

func TestIDMigration_LoadLegacyControlFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Simulate an on-disk legacy control file with numeric ID.
	legacyJSON := `{
  "id": 12345,
  "name": "Legacy Disk Control",
  "reference_id": "AC-01",
  "description": "Loaded from legacy file",
  "category": "Access Control",
  "framework": "SOC2",
  "status": "implemented",
  "external_ids": {"tugboat": "12345"}
}`
	fpath := filepath.Join(dir, "control-12345.json")
	require.NoError(t, os.WriteFile(fpath, []byte(legacyJSON), 0644))

	data, err := os.ReadFile(fpath)
	require.NoError(t, err)

	var c domain.Control
	err = stdjson.Unmarshal(data, &c)
	require.NoError(t, err)
	assert.Equal(t, "12345", c.ID, "numeric on-disk ID must be read as string")
	assert.Equal(t, "AC-01", c.ReferenceID)
	assert.Equal(t, "SOC2", c.Framework)
	assert.Equal(t, "12345", c.ExternalIDs["tugboat"])
}

func TestIDMigration_LoadLegacyEvidenceTaskFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	legacyJSON := `{
  "id": 54321,
  "name": "Legacy Disk Task",
  "description": "Legacy evidence task",
  "collection_interval": "quarter"
}`
	fpath := filepath.Join(dir, "task-54321.json")
	require.NoError(t, os.WriteFile(fpath, []byte(legacyJSON), 0644))

	data, err := os.ReadFile(fpath)
	require.NoError(t, err)

	var et domain.EvidenceTask
	err = stdjson.Unmarshal(data, &et)
	require.NoError(t, err)
	assert.Equal(t, "54321", et.ID)
	assert.Equal(t, "quarter", et.CollectionInterval)
}

func TestIDMigration_LoadLegacyEvidenceTaskStateFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	legacyJSON := `{
  "task_ref": "ET-0047",
  "task_id": 327992,
  "task_name": "GitHub Repository Access Controls",
  "tugboat_status": "pending",
  "tugboat_completed": false
}`
	fpath := filepath.Join(dir, "state-ET-0047.json")
	require.NoError(t, os.WriteFile(fpath, []byte(legacyJSON), 0644))

	data, err := os.ReadFile(fpath)
	require.NoError(t, err)

	var s models.EvidenceTaskState
	err = stdjson.Unmarshal(data, &s)
	require.NoError(t, err)
	assert.Equal(t, "327992", s.TaskID)
	assert.Equal(t, "ET-0047", s.TaskRef)
	assert.Equal(t, "pending", s.TugboatStatus)
}

// ---------------------------------------------------------------------------
// 12. Full pipeline: adapter → marshal → legacy load → unmarshal
// ---------------------------------------------------------------------------

func TestIDMigration_FullPipeline_AdapterThroughStorage(t *testing.T) {
	t.Parallel()

	// Step 1: Adapter converts API int → domain string.
	adapter := adapters.NewTugboatToDomain()
	apiControl := tugboatmodels.Control{
		ID:       77777,
		Name:     "Pipeline Control",
		Category: "Logical Access",
		Status:   "implemented",
	}
	domainCtrl := adapter.ConvertControl(apiControl)
	assert.Equal(t, "77777", domainCtrl.ID)

	// Step 2: Marshal to JSON (simulates writing to disk).
	data, err := stdjson.Marshal(domainCtrl)
	require.NoError(t, err)

	// Step 3: Unmarshal from JSON (simulates reading from disk).
	var restored domain.Control
	err = stdjson.Unmarshal(data, &restored)
	require.NoError(t, err)
	assert.Equal(t, "77777", restored.ID, "ID must survive full pipeline")
	assert.Equal(t, "Pipeline Control", restored.Name)
	assert.Equal(t, "77777", restored.ExternalIDs["tugboat"])
}

func TestIDMigration_MigrateToolHandlesAllIDFields(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// File with all known ID fields as numeric values.
	raw := `{
  "id": 100,
  "task_id": 200,
  "control_id": 300,
  "policy_id": 400,
  "name": "All ID Fields"
}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "all_ids.json"), []byte(raw), 0644))

	result, err := services.MigrateJSONFiles(dir, false)
	require.NoError(t, err)
	assert.Equal(t, 4, len(result.Changes), "must convert all 4 ID fields")

	data, err := os.ReadFile(filepath.Join(dir, "all_ids.json"))
	require.NoError(t, err)
	var m map[string]interface{}
	require.NoError(t, stdjson.Unmarshal(data, &m))
	assert.Equal(t, "100", m["id"])
	assert.Equal(t, "200", m["task_id"])
	assert.Equal(t, "300", m["control_id"])
	assert.Equal(t, "400", m["policy_id"])
}

func TestIDMigration_MigrateToolSkipsNonJSON(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Write a non-JSON file.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# hello"), 0644))
	// Write a JSON file that needs migration.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data.json"), []byte(`{"id": 1}`), 0644))

	result, err := services.MigrateJSONFiles(dir, false)
	require.NoError(t, err)
	assert.Equal(t, 1, result.FilesScanned, "must only scan .json files")
	assert.Equal(t, 1, result.FilesModified)
}

func TestIDMigration_MigrateToolHandlesAlreadyStringIDs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// File with IDs already as strings.
	raw := `{"id": "already-string", "task_id": "also-string", "name": "New Format"}`
	fpath := filepath.Join(dir, "new.json")
	require.NoError(t, os.WriteFile(fpath, []byte(raw), 0644))

	result, err := services.MigrateJSONFiles(dir, false)
	require.NoError(t, err)
	assert.Equal(t, 0, result.FilesModified, "already-string IDs must not be modified")
}
