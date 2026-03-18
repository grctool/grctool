// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"testing"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- GetPolicyByExternalID ---

func TestStorage_GetPolicyByExternalID(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	policy := testhelpers.SamplePolicy()
	policy.ExternalIDs = map[string]string{"tugboat": "123"}
	require.NoError(t, s.SavePolicy(policy))

	got, err := s.GetPolicyByExternalID("tugboat", "123")
	require.NoError(t, err)
	assert.Equal(t, policy.ID, got.ID)
	assert.Equal(t, policy.Name, got.Name)
	assert.Equal(t, "123", got.ExternalIDs["tugboat"])
}

func TestStorage_GetPolicyByExternalID_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	_, err := s.GetPolicyByExternalID("tugboat", "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "policy not found")
	assert.Contains(t, err.Error(), "tugboat")
	assert.Contains(t, err.Error(), "nonexistent")
}

// --- GetControlByExternalID ---

func TestStorage_GetControlByExternalID(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	control := testhelpers.SampleControl()
	control.ExternalIDs = map[string]string{"tugboat": "456"}
	require.NoError(t, s.SaveControl(control))

	got, err := s.GetControlByExternalID("tugboat", "456")
	require.NoError(t, err)
	assert.Equal(t, control.ID, got.ID)
	assert.Equal(t, control.Name, got.Name)
	assert.Equal(t, "456", got.ExternalIDs["tugboat"])
}

func TestStorage_GetControlByExternalID_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	_, err := s.GetControlByExternalID("tugboat", "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "control not found")
	assert.Contains(t, err.Error(), "tugboat")
}

// --- GetEvidenceTaskByExternalID ---

func TestStorage_GetEvidenceTaskByExternalID(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	task := testhelpers.SampleEvidenceTask()
	task.ExternalIDs = map[string]string{"tugboat": "789"}
	require.NoError(t, s.SaveEvidenceTask(task))

	got, err := s.GetEvidenceTaskByExternalID("tugboat", "789")
	require.NoError(t, err)
	assert.Equal(t, task.ID, got.ID)
	assert.Equal(t, task.Name, got.Name)
	assert.Equal(t, "789", got.ExternalIDs["tugboat"])
}

func TestStorage_GetEvidenceTaskByExternalID_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	_, err := s.GetEvidenceTaskByExternalID("tugboat", "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "evidence task not found")
	assert.Contains(t, err.Error(), "tugboat")
}

// --- Edge cases ---

func TestStorage_GetByExternalID_NoExternalIDs(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	// Save entities without ExternalIDs set (nil map, simulating legacy data)
	policy := testhelpers.SamplePolicy()
	require.NoError(t, s.SavePolicy(policy))

	control := testhelpers.SampleControl()
	require.NoError(t, s.SaveControl(control))

	task := testhelpers.SampleEvidenceTask()
	require.NoError(t, s.SaveEvidenceTask(task))

	_, err := s.GetPolicyByExternalID("tugboat", "123")
	assert.Error(t, err)

	_, err = s.GetControlByExternalID("tugboat", "456")
	assert.Error(t, err)

	_, err = s.GetEvidenceTaskByExternalID("tugboat", "789")
	assert.Error(t, err)
}

func TestStorage_GetByExternalID_WrongProvider(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	policy := testhelpers.SamplePolicy()
	policy.ExternalIDs = map[string]string{"tugboat": "123"}
	require.NoError(t, s.SavePolicy(policy))

	control := testhelpers.SampleControl()
	control.ExternalIDs = map[string]string{"tugboat": "456"}
	require.NoError(t, s.SaveControl(control))

	task := testhelpers.SampleEvidenceTask()
	task.ExternalIDs = map[string]string{"tugboat": "789"}
	require.NoError(t, s.SaveEvidenceTask(task))

	// Query with wrong provider
	_, err := s.GetPolicyByExternalID("accountablehq", "123")
	assert.Error(t, err)

	_, err = s.GetControlByExternalID("accountablehq", "456")
	assert.Error(t, err)

	_, err = s.GetEvidenceTaskByExternalID("accountablehq", "789")
	assert.Error(t, err)
}

func TestStorage_GetPolicyByExternalID_MultipleProviders(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	policy := testhelpers.SamplePolicy()
	policy.ExternalIDs = map[string]string{
		"tugboat":       "123",
		"accountablehq": "abc",
	}
	require.NoError(t, s.SavePolicy(policy))

	// Both providers should resolve
	got, err := s.GetPolicyByExternalID("tugboat", "123")
	require.NoError(t, err)
	assert.Equal(t, policy.ID, got.ID)

	got, err = s.GetPolicyByExternalID("accountablehq", "abc")
	require.NoError(t, err)
	assert.Equal(t, policy.ID, got.ID)

	// Wrong external ID for a valid provider
	_, err = s.GetPolicyByExternalID("tugboat", "abc")
	assert.Error(t, err)
}

func TestStorage_GetByExternalID_MultipleEntities(t *testing.T) {
	t.Parallel()
	s := newTestStorage(t)

	p1 := &domain.Policy{
		ID:          "p1",
		ReferenceID: "POL-0001",
		Name:        "Policy One",
		ExternalIDs: map[string]string{"tugboat": "100"},
	}
	p2 := &domain.Policy{
		ID:          "p2",
		ReferenceID: "POL-0002",
		Name:        "Policy Two",
		ExternalIDs: map[string]string{"tugboat": "200"},
	}
	require.NoError(t, s.SavePolicy(p1))
	require.NoError(t, s.SavePolicy(p2))

	got, err := s.GetPolicyByExternalID("tugboat", "200")
	require.NoError(t, err)
	assert.Equal(t, "p2", got.ID)

	got, err = s.GetPolicyByExternalID("tugboat", "100")
	require.NoError(t, err)
	assert.Equal(t, "p1", got.ID)
}
