// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package interfaces_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubRelationshipQuerier implements both DataProvider and RelationshipQuerier.
type stubRelationshipQuerier struct {
	*testhelpers.StubDataProvider
	controlToTasks    map[string][]domain.EvidenceTask
	policyToControls  map[string][]domain.Control
	controlToPolicies map[string][]domain.Policy
}

func (s *stubRelationshipQuerier) GetEvidenceTasksByControl(ctx context.Context, controlID string) ([]domain.EvidenceTask, error) {
	tasks, ok := s.controlToTasks[controlID]
	if !ok {
		return nil, nil
	}
	return tasks, nil
}

func (s *stubRelationshipQuerier) GetControlsByPolicy(ctx context.Context, policyID string) ([]domain.Control, error) {
	controls, ok := s.policyToControls[policyID]
	if !ok {
		return nil, nil
	}
	return controls, nil
}

func (s *stubRelationshipQuerier) GetPoliciesByControl(ctx context.Context, controlID string) ([]domain.Policy, error) {
	policies, ok := s.controlToPolicies[controlID]
	if !ok {
		return nil, nil
	}
	return policies, nil
}

// --- Tests ---

func TestRelationshipQuerier_TypeAssertion_Positive(t *testing.T) {
	t.Parallel()
	var provider interfaces.DataProvider = &stubRelationshipQuerier{
		StubDataProvider: testhelpers.NewStubDataProvider("rel-test"),
	}
	rq, ok := provider.(interfaces.RelationshipQuerier)
	assert.True(t, ok, "stubRelationshipQuerier should implement RelationshipQuerier")
	assert.NotNil(t, rq)
}

func TestRelationshipQuerier_TypeAssertion_Negative(t *testing.T) {
	t.Parallel()
	// StubDataProvider does NOT implement RelationshipQuerier.
	var provider interfaces.DataProvider = testhelpers.NewStubDataProvider("plain")
	_, ok := provider.(interfaces.RelationshipQuerier)
	assert.False(t, ok, "plain StubDataProvider should NOT implement RelationshipQuerier")
}

func TestRelationshipQuerier_GetEvidenceTasksByControl(t *testing.T) {
	t.Parallel()
	task1 := domain.EvidenceTask{ID: "ET-0047", Name: "GitHub Access Controls"}
	task2 := domain.EvidenceTask{ID: "ET-0048", Name: "GitHub Workflow Security"}

	rq := &stubRelationshipQuerier{
		StubDataProvider: testhelpers.NewStubDataProvider("rel-test"),
		controlToTasks: map[string][]domain.EvidenceTask{
			"CC-06.1": {task1, task2},
		},
	}

	tasks, err := rq.GetEvidenceTasksByControl(context.Background(), "CC-06.1")
	require.NoError(t, err)
	assert.Len(t, tasks, 2)
	assert.Equal(t, "ET-0047", tasks[0].ID)
}

func TestRelationshipQuerier_GetEvidenceTasksByControl_NoTasks(t *testing.T) {
	t.Parallel()
	rq := &stubRelationshipQuerier{
		StubDataProvider: testhelpers.NewStubDataProvider("rel-test"),
		controlToTasks:   map[string][]domain.EvidenceTask{},
	}

	tasks, err := rq.GetEvidenceTasksByControl(context.Background(), "UNKNOWN")
	require.NoError(t, err)
	assert.Empty(t, tasks)
}

func TestRelationshipQuerier_GetControlsByPolicy(t *testing.T) {
	t.Parallel()
	ctrl1 := domain.Control{ID: "CC-06.1", Name: "Logical Access", ReferenceID: "CC-06.1"}
	ctrl2 := domain.Control{ID: "CC-06.8", Name: "Access Revocation", ReferenceID: "CC-06.8"}

	rq := &stubRelationshipQuerier{
		StubDataProvider: testhelpers.NewStubDataProvider("rel-test"),
		policyToControls: map[string][]domain.Control{
			"POL-001": {ctrl1, ctrl2},
		},
	}

	controls, err := rq.GetControlsByPolicy(context.Background(), "POL-001")
	require.NoError(t, err)
	assert.Len(t, controls, 2)
}

func TestRelationshipQuerier_GetPoliciesByControl(t *testing.T) {
	t.Parallel()
	pol := domain.Policy{ID: "POL-001", Name: "Access Control Policy"}

	rq := &stubRelationshipQuerier{
		StubDataProvider: testhelpers.NewStubDataProvider("rel-test"),
		controlToPolicies: map[string][]domain.Policy{
			"CC-06.1": {pol},
		},
	}

	policies, err := rq.GetPoliciesByControl(context.Background(), "CC-06.1")
	require.NoError(t, err)
	assert.Len(t, policies, 1)
	assert.Equal(t, "Access Control Policy", policies[0].Name)
}

func TestRelationshipQuerier_EmptyMaps(t *testing.T) {
	t.Parallel()
	rq := &stubRelationshipQuerier{
		StubDataProvider:  testhelpers.NewStubDataProvider("empty"),
		controlToTasks:    map[string][]domain.EvidenceTask{},
		policyToControls:  map[string][]domain.Control{},
		controlToPolicies: map[string][]domain.Policy{},
	}

	tasks, _ := rq.GetEvidenceTasksByControl(context.Background(), "any")
	assert.Empty(t, tasks)

	controls, _ := rq.GetControlsByPolicy(context.Background(), "any")
	assert.Empty(t, controls)

	policies, _ := rq.GetPoliciesByControl(context.Background(), "any")
	assert.Empty(t, policies)
}

// Verify the pattern: callers check capability before calling.
func TestRelationshipQuerier_CallerPattern(t *testing.T) {
	t.Parallel()

	queryRelationships := func(provider interfaces.DataProvider, controlID string) ([]domain.EvidenceTask, error) {
		rq, ok := provider.(interfaces.RelationshipQuerier)
		if !ok {
			return nil, fmt.Errorf("provider %s does not support relationship queries", provider.Name())
		}
		return rq.GetEvidenceTasksByControl(context.Background(), controlID)
	}

	// Provider WITH relationships
	withRel := &stubRelationshipQuerier{
		StubDataProvider: testhelpers.NewStubDataProvider("with-rel"),
		controlToTasks: map[string][]domain.EvidenceTask{
			"CC-06.1": {{ID: "ET-0047"}},
		},
	}
	tasks, err := queryRelationships(withRel, "CC-06.1")
	require.NoError(t, err)
	assert.Len(t, tasks, 1)

	// Provider WITHOUT relationships
	withoutRel := testhelpers.NewStubDataProvider("no-rel")
	_, err = queryRelationships(withoutRel, "CC-06.1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not support relationship queries")
}
