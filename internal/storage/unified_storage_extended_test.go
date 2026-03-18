// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStorage(t *testing.T) *Storage {
	t.Helper()
	s, err := NewStorage(config.StorageConfig{DataDir: t.TempDir()})
	require.NoError(t, err)
	return s
}

// --- GetPolicy lookups by reference ID and underscore format ---

func TestStorage_GetPolicy_ByReferenceID(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	policy := testhelpers.SamplePolicy()
	require.NoError(t, s.SavePolicy(policy))

	// Lookup by ReferenceID
	retrieved, err := s.GetPolicy("POL-0001")
	require.NoError(t, err)
	assert.Equal(t, policy.Name, retrieved.Name)
}

func TestStorage_GetPolicy_ByUnderscoreFormat(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	policy := testhelpers.SamplePolicy()
	require.NoError(t, s.SavePolicy(policy))

	// Lookup by underscore format (POL_0001 -> POL-0001)
	retrieved, err := s.GetPolicy("POL_0001")
	require.NoError(t, err)
	assert.Equal(t, policy.Name, retrieved.Name)
}

func TestStorage_GetPolicy_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	_, err := s.GetPolicy("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// --- GetControl lookups ---

func TestStorage_GetControl_ByNumericID(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	control := testhelpers.SampleControl()
	require.NoError(t, s.SaveControl(control))

	retrieved, err := s.GetControl("1001")
	require.NoError(t, err)
	assert.Equal(t, control.Name, retrieved.Name)
}

func TestStorage_GetControl_ByReferenceID(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	control := testhelpers.SampleControl()
	require.NoError(t, s.SaveControl(control))

	retrieved, err := s.GetControl("CC-06.1")
	require.NoError(t, err)
	assert.Equal(t, control.Name, retrieved.Name)
}

func TestStorage_GetControl_ByUnderscoreFormat(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	control := testhelpers.SampleControl()
	require.NoError(t, s.SaveControl(control))

	// CC_06.1 -> CC.06.1 ... actually this replaces _ with .
	// The code does: strings.ReplaceAll(id, "_", ".")
	// CC-06.1 has a dot already. Let's test with a control that has dots from underscores.
	control2 := &domain.Control{
		ID: "2002", ReferenceID: "CC1.1", Name: "Test CC Control",
		Framework: "SOC2", Status: "implemented",
	}
	require.NoError(t, s.SaveControl(control2))

	retrieved, err := s.GetControl("CC1_1")
	require.NoError(t, err)
	assert.Equal(t, "Test CC Control", retrieved.Name)
}

func TestStorage_GetControl_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	_, err := s.GetControl("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// --- GetEvidenceTask lookups by various ID formats ---

func TestStorage_GetEvidenceTask_ByNumericID(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	task := testhelpers.SampleEvidenceTask()
	require.NoError(t, s.SaveEvidenceTask(task))

	retrieved, err := s.GetEvidenceTask("327992")
	require.NoError(t, err)
	assert.Equal(t, task.Name, retrieved.Name)
}

func TestStorage_GetEvidenceTask_ByReference(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	task := testhelpers.SampleEvidenceTask()
	require.NoError(t, s.SaveEvidenceTask(task))

	// ET-0047 format
	retrieved, err := s.GetEvidenceTask("ET-0047")
	require.NoError(t, err)
	assert.Equal(t, task.Name, retrieved.Name)
}

func TestStorage_GetEvidenceTask_ByReferenceNoDash(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	task := testhelpers.SampleEvidenceTask()
	require.NoError(t, s.SaveEvidenceTask(task))

	// ET0047 format (without dash)
	retrieved, err := s.GetEvidenceTask("ET0047")
	require.NoError(t, err)
	assert.Equal(t, task.Name, retrieved.Name)
}

func TestStorage_GetEvidenceTask_LowercaseReference(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	task := testhelpers.SampleEvidenceTask()
	require.NoError(t, s.SaveEvidenceTask(task))

	// et-0047 (lowercase)
	retrieved, err := s.GetEvidenceTask("et-0047")
	require.NoError(t, err)
	assert.Equal(t, task.Name, retrieved.Name)
}

func TestStorage_GetEvidenceTask_InvalidFormat(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	_, err := s.GetEvidenceTask("invalid-format")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid task ID format")
}

func TestStorage_GetEvidenceTask_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	task := testhelpers.SampleEvidenceTask()
	require.NoError(t, s.SaveEvidenceTask(task))

	// Valid numeric ID but no matching task
	_, err := s.GetEvidenceTask("999999")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Reference format pointing to non-existent task returns invalid format
	// (because lookupTaskByReferenceNumber returns 0 -> parseTaskID returns 0)
	_, err = s.GetEvidenceTask("ET-9999")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid task ID format")
}

// --- GetPolicyByReferenceAndID ---

func TestStorage_GetPolicyByReferenceAndID(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	policy := testhelpers.SamplePolicy()
	require.NoError(t, s.SavePolicy(policy))

	retrieved, err := s.GetPolicyByReferenceAndID("POL-0001", policy.ID)
	require.NoError(t, err)
	assert.Equal(t, policy.Name, retrieved.Name)

	_, err = s.GetPolicyByReferenceAndID("POL-9999", "0")
	assert.Error(t, err)
}

// --- GetEvidenceTaskByReferenceAndID ---

func TestStorage_GetEvidenceTaskByReferenceAndID(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	task := testhelpers.SampleEvidenceTask()
	require.NoError(t, s.SaveEvidenceTask(task))

	retrieved, err := s.GetEvidenceTaskByReferenceAndID("ET-0047", "327992")
	require.NoError(t, err)
	assert.Equal(t, task.Name, retrieved.Name)

	_, err = s.GetEvidenceTaskByReferenceAndID("ET-9999", "0")
	assert.Error(t, err)
}

// --- Evidence Record operations ---

func TestStorage_SaveAndGetEvidenceRecord(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	record := testhelpers.SampleEvidenceRecord()
	require.NoError(t, s.SaveEvidenceRecord(record))

	retrieved, err := s.GetEvidenceRecord(record.ID)
	require.NoError(t, err)
	assert.Equal(t, record.Title, retrieved.Title)
	assert.Equal(t, record.TaskID, retrieved.TaskID)
}

func TestStorage_GetEvidenceRecord_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	_, err := s.GetEvidenceRecord("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestStorage_SaveEvidenceRecord_Nil(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	err := s.SaveEvidenceRecord(nil)
	assert.Error(t, err)
}

func TestStorage_GetEvidenceRecordsByTaskID(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	rec1 := &domain.EvidenceRecord{
		ID: "rec-1", TaskID: "100", Title: "Record 1",
		Format: "markdown", Source: "test", CollectedAt: time.Now(),
	}
	rec2 := &domain.EvidenceRecord{
		ID: "rec-2", TaskID: "100", Title: "Record 2",
		Format: "markdown", Source: "test", CollectedAt: time.Now(),
	}
	rec3 := &domain.EvidenceRecord{
		ID: "rec-3", TaskID: "200", Title: "Record 3",
		Format: "markdown", Source: "test", CollectedAt: time.Now(),
	}

	require.NoError(t, s.SaveEvidenceRecord(rec1))
	require.NoError(t, s.SaveEvidenceRecord(rec2))
	require.NoError(t, s.SaveEvidenceRecord(rec3))

	records, err := s.GetEvidenceRecordsByTaskID("100")
	require.NoError(t, err)
	assert.Len(t, records, 2)

	records, err = s.GetEvidenceRecordsByTaskID("200")
	require.NoError(t, err)
	assert.Len(t, records, 1)

	records, err = s.GetEvidenceRecordsByTaskID("999")
	require.NoError(t, err)
	assert.Empty(t, records)
}

// --- Summaries ---

func TestStorage_GetPolicySummary(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	p1 := testhelpers.SamplePolicy()
	p2 := &domain.Policy{
		ID: "22222", ReferenceID: "POL-0002", Name: "Data Policy",
		Framework: "ISO27001", Status: "draft",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, s.SavePolicy(p1))
	require.NoError(t, s.SavePolicy(p2))

	summary, err := s.GetPolicySummary()
	require.NoError(t, err)
	assert.Equal(t, 2, summary.Total)
	assert.Equal(t, 1, summary.ByFramework["SOC2"])
	assert.Equal(t, 1, summary.ByFramework["ISO27001"])
}

func TestStorage_GetControlSummary(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	c1 := testhelpers.SampleControl()
	c2 := &domain.Control{
		ID: "2002", ReferenceID: "AC-02", Name: "Network",
		Framework: "SOC2", Category: "Infrastructure", Status: "planned",
	}
	require.NoError(t, s.SaveControl(c1))
	require.NoError(t, s.SaveControl(c2))

	summary, err := s.GetControlSummary()
	require.NoError(t, err)
	assert.Equal(t, 2, summary.Total)
	assert.Equal(t, 2, summary.ByFramework["SOC2"])
}

func TestStorage_GetEvidenceTaskSummary(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	overdue := time.Now().Add(-48 * time.Hour)
	dueSoon := time.Now().Add(3 * 24 * time.Hour)
	farAway := time.Now().Add(30 * 24 * time.Hour)

	t1 := &domain.EvidenceTask{
		ID: "1", ReferenceID: "ET-0001", Name: "Overdue Task",
		Status: "pending", Priority: "high", NextDue: &overdue,
	}
	t2 := &domain.EvidenceTask{
		ID: "2", ReferenceID: "ET-0002", Name: "Due Soon Task",
		Status: "in_progress", Priority: "medium", NextDue: &dueSoon,
	}
	t3 := &domain.EvidenceTask{
		ID: "3", ReferenceID: "ET-0003", Name: "Future Task",
		Status: "pending", Priority: "low", NextDue: &farAway,
	}

	require.NoError(t, s.SaveEvidenceTask(t1))
	require.NoError(t, s.SaveEvidenceTask(t2))
	require.NoError(t, s.SaveEvidenceTask(t3))

	summary, err := s.GetEvidenceTaskSummary()
	require.NoError(t, err)
	assert.Equal(t, 3, summary.Total)
	assert.Equal(t, 1, summary.Overdue)
	assert.Equal(t, 1, summary.DueSoon)
	assert.Equal(t, 2, summary.ByStatus["pending"])
	assert.Equal(t, 1, summary.ByStatus["in_progress"])
	assert.Equal(t, 1, summary.ByPriority["high"])
	assert.Equal(t, 1, summary.ByPriority["medium"])
	assert.Equal(t, 1, summary.ByPriority["low"])
}

// --- SyncTime ---

func TestStorage_SetAndGetSyncTime(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	syncTime := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	require.NoError(t, s.SetSyncTime("full", syncTime))

	retrieved, err := s.GetSyncTime("full")
	require.NoError(t, err)
	assert.Equal(t, syncTime, retrieved)
}

func TestStorage_GetSyncTime_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	_, err := s.GetSyncTime("nonexistent")
	assert.Error(t, err)
}

// --- GetStats ---

func TestStorage_GetStats_WithData(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	require.NoError(t, s.SavePolicy(testhelpers.SamplePolicy()))
	require.NoError(t, s.SaveControl(testhelpers.SampleControl()))
	require.NoError(t, s.SaveEvidenceTask(testhelpers.SampleEvidenceTask()))

	stats, err := s.GetStats()
	require.NoError(t, err)
	assert.Equal(t, 1, stats["total_policies"])
	assert.Equal(t, 1, stats["total_controls"])
	assert.Equal(t, 1, stats["total_evidence_tasks"])
	assert.Contains(t, stats, "generated_at")
}

// --- GetBaseDir ---

func TestStorage_GetBaseDir(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)
	assert.NotEmpty(t, s.GetBaseDir())
}

// --- parseTaskID ---

func TestStorage_ParseTaskID_Formats(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	task := testhelpers.SampleEvidenceTask()
	require.NoError(t, s.SaveEvidenceTask(task))

	// Pure numeric
	assert.Equal(t, "327992", s.parseTaskID("327992"))

	// ET-XXXX format
	assert.Equal(t, "327992", s.parseTaskID("ET-0047"))

	// ETXXXX format
	assert.Equal(t, "327992", s.parseTaskID("ET0047"))

	// lowercase
	assert.Equal(t, "327992", s.parseTaskID("et-0047"))

	// Non-matching reference
	assert.Equal(t, "", s.parseTaskID("ET-9999"))

	// Invalid format
	assert.Equal(t, "", s.parseTaskID("INVALID"))

	// Whitespace trimmed
	assert.Equal(t, "327992", s.parseTaskID("  327992  "))
}
