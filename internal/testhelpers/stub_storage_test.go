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

package testhelpers

import (
	"testing"
	"time"

	"github.com/grctool/grctool/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStubStorageService_SaveAndGetPolicy(t *testing.T) {
	t.Parallel()
	s := NewStubStorageService()

	policy := SamplePolicy()
	require.NoError(t, s.SavePolicy(policy))

	got, err := s.GetPolicy(policy.ID)
	require.NoError(t, err)
	assert.Equal(t, policy.Name, got.Name)
	assert.Equal(t, policy.ReferenceID, got.ReferenceID)
}

func TestStubStorageService_SaveAndGetControl(t *testing.T) {
	t.Parallel()
	s := NewStubStorageService()

	control := SampleControl()
	require.NoError(t, s.SaveControl(control))

	got, err := s.GetControl("1001")
	require.NoError(t, err)
	assert.Equal(t, control.Name, got.Name)
	assert.Equal(t, control.ReferenceID, got.ReferenceID)
}

func TestStubStorageService_SaveAndGetEvidenceTask(t *testing.T) {
	t.Parallel()
	s := NewStubStorageService()

	task := SampleEvidenceTask()
	require.NoError(t, s.SaveEvidenceTask(task))

	got, err := s.GetEvidenceTask("327992")
	require.NoError(t, err)
	assert.Equal(t, task.Name, got.Name)
	assert.Equal(t, task.ReferenceID, got.ReferenceID)
}

func TestStubStorageService_GetAll(t *testing.T) {
	t.Parallel()
	s := NewStubStorageServiceWithData(
		[]domain.Policy{*SamplePolicy()},
		[]domain.Control{*SampleControl()},
		[]domain.EvidenceTask{*SampleEvidenceTask()},
	)

	policies, err := s.GetAllPolicies()
	require.NoError(t, err)
	assert.Len(t, policies, 1)

	controls, err := s.GetAllControls()
	require.NoError(t, err)
	assert.Len(t, controls, 1)

	tasks, err := s.GetAllEvidenceTasks()
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
}

func TestStubStorageService_GetByReferenceAndID(t *testing.T) {
	t.Parallel()
	s := NewStubStorageService()

	policy := SamplePolicy()
	require.NoError(t, s.SavePolicy(policy))

	// Look up by reference ID
	got, err := s.GetPolicyByReferenceAndID("POL-0001", "")
	require.NoError(t, err)
	assert.Equal(t, policy.Name, got.Name)

	// Look up by numeric ID
	got, err = s.GetPolicyByReferenceAndID("", "12345")
	require.NoError(t, err)
	assert.Equal(t, policy.Name, got.Name)

	// Control by reference
	control := SampleControl()
	require.NoError(t, s.SaveControl(control))

	gotCtrl, err := s.GetControlByReferenceAndID("CC-06.1", "")
	require.NoError(t, err)
	assert.Equal(t, control.Name, gotCtrl.Name)

	// Task by reference
	task := SampleEvidenceTask()
	require.NoError(t, s.SaveEvidenceTask(task))

	gotTask, err := s.GetEvidenceTaskByReferenceAndID("ET-0047", "")
	require.NoError(t, err)
	assert.Equal(t, task.Name, gotTask.Name)
}

func TestStubStorageService_NotFound(t *testing.T) {
	t.Parallel()
	s := NewStubStorageService()

	_, err := s.GetPolicy("nonexistent")
	assert.Error(t, err)

	_, err = s.GetControl("nonexistent")
	assert.Error(t, err)

	_, err = s.GetEvidenceTask("nonexistent")
	assert.Error(t, err)

	_, err = s.GetEvidenceRecord("nonexistent")
	assert.Error(t, err)

	_, err = s.GetPolicyByReferenceAndID("NOPE", "NOPE")
	assert.Error(t, err)

	_, err = s.GetControlByReferenceAndID("NOPE", "NOPE")
	assert.Error(t, err)

	_, err = s.GetEvidenceTaskByReferenceAndID("NOPE", "NOPE")
	assert.Error(t, err)
}

func TestStubStorageService_EvidenceRecords(t *testing.T) {
	t.Parallel()
	s := NewStubStorageService()

	rec := SampleEvidenceRecord()
	require.NoError(t, s.SaveEvidenceRecord(rec))

	got, err := s.GetEvidenceRecord(rec.ID)
	require.NoError(t, err)
	assert.Equal(t, rec.Title, got.Title)

	byTask, err := s.GetEvidenceRecordsByTaskID(rec.TaskID)
	require.NoError(t, err)
	assert.Len(t, byTask, 1)

	// No records for a different task
	byTask, err = s.GetEvidenceRecordsByTaskID("99999")
	require.NoError(t, err)
	assert.Empty(t, byTask)
}

func TestStubStorageService_SyncTimes(t *testing.T) {
	t.Parallel()
	s := NewStubStorageService()

	now := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, s.SetSyncTime("policies", now))

	got, err := s.GetSyncTime("policies")
	require.NoError(t, err)
	assert.Equal(t, now, got)

	_, err = s.GetSyncTime("nonexistent")
	assert.Error(t, err)
}

func TestStubStorageService_Stats(t *testing.T) {
	t.Parallel()
	s := NewStubStorageServiceWithData(
		[]domain.Policy{*SamplePolicy()},
		[]domain.Control{*SampleControl()},
		[]domain.EvidenceTask{*SampleEvidenceTask()},
	)

	stats, err := s.GetStats()
	require.NoError(t, err)
	assert.Equal(t, 1, stats["policies"])
	assert.Equal(t, 1, stats["controls"])
	assert.Equal(t, 1, stats["evidence_tasks"])
}

func TestStubStorageService_Clear(t *testing.T) {
	t.Parallel()
	s := NewStubStorageServiceWithData(
		[]domain.Policy{*SamplePolicy()},
		[]domain.Control{*SampleControl()},
		[]domain.EvidenceTask{*SampleEvidenceTask()},
	)

	require.NoError(t, s.Clear())

	policies, _ := s.GetAllPolicies()
	assert.Empty(t, policies)

	controls, _ := s.GetAllControls()
	assert.Empty(t, controls)

	tasks, _ := s.GetAllEvidenceTasks()
	assert.Empty(t, tasks)
}

func TestStubStorageService_Summaries(t *testing.T) {
	t.Parallel()
	s := NewStubStorageServiceWithData(
		[]domain.Policy{*SamplePolicy()},
		[]domain.Control{*SampleControl()},
		[]domain.EvidenceTask{*SampleEvidenceTask()},
	)

	ps, err := s.GetPolicySummary()
	require.NoError(t, err)
	assert.Equal(t, 1, ps.Total)
	assert.Equal(t, 1, ps.ByFramework["SOC2"])

	cs, err := s.GetControlSummary()
	require.NoError(t, err)
	assert.Equal(t, 1, cs.Total)
	assert.Equal(t, 1, cs.ByStatus["implemented"])

	ts, err := s.GetEvidenceTaskSummary()
	require.NoError(t, err)
	assert.Equal(t, 1, ts.Total)
}

func TestStubStorageService_GetPolicyByExternalID(t *testing.T) {
	t.Parallel()
	s := NewStubStorageService()

	policy := SamplePolicy()
	policy.ExternalIDs = map[string]string{"tugboat": "123"}
	require.NoError(t, s.SavePolicy(policy))

	got, err := s.GetPolicyByExternalID("tugboat", "123")
	require.NoError(t, err)
	assert.Equal(t, policy.ID, got.ID)

	_, err = s.GetPolicyByExternalID("tugboat", "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "policy not found")

	_, err = s.GetPolicyByExternalID("accountablehq", "123")
	assert.Error(t, err)
}

func TestStubStorageService_GetControlByExternalID(t *testing.T) {
	t.Parallel()
	s := NewStubStorageService()

	control := SampleControl()
	control.ExternalIDs = map[string]string{"tugboat": "456"}
	require.NoError(t, s.SaveControl(control))

	got, err := s.GetControlByExternalID("tugboat", "456")
	require.NoError(t, err)
	assert.Equal(t, control.ID, got.ID)

	_, err = s.GetControlByExternalID("tugboat", "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "control not found")
}

func TestStubStorageService_GetEvidenceTaskByExternalID(t *testing.T) {
	t.Parallel()
	s := NewStubStorageService()

	task := SampleEvidenceTask()
	task.ExternalIDs = map[string]string{"tugboat": "789"}
	require.NoError(t, s.SaveEvidenceTask(task))

	got, err := s.GetEvidenceTaskByExternalID("tugboat", "789")
	require.NoError(t, err)
	assert.Equal(t, task.ID, got.ID)

	_, err = s.GetEvidenceTaskByExternalID("tugboat", "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "evidence task not found")
}

func TestStubStorageService_GetByExternalID_NilExternalIDs(t *testing.T) {
	t.Parallel()
	s := NewStubStorageService()

	// Entities without ExternalIDs (nil map)
	require.NoError(t, s.SavePolicy(SamplePolicy()))
	require.NoError(t, s.SaveControl(SampleControl()))
	require.NoError(t, s.SaveEvidenceTask(SampleEvidenceTask()))

	_, err := s.GetPolicyByExternalID("tugboat", "123")
	assert.Error(t, err)

	_, err = s.GetControlByExternalID("tugboat", "456")
	assert.Error(t, err)

	_, err = s.GetEvidenceTaskByExternalID("tugboat", "789")
	assert.Error(t, err)
}

func TestStubStorageService_NilInputs(t *testing.T) {
	t.Parallel()
	s := NewStubStorageService()

	assert.Error(t, s.SavePolicy(nil))
	assert.Error(t, s.SaveControl(nil))
	assert.Error(t, s.SaveEvidenceTask(nil))
	assert.Error(t, s.SaveEvidenceRecord(nil))
}
