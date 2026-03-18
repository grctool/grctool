// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"testing"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newEvidenceRelationshipsToolForPureTesting creates a minimal tool for testing pure methods.
// The dataStore is nil so Execute won't work, but pure calculation methods will.
func newEvidenceRelationshipsToolForPureTesting(t *testing.T) *EvidenceRelationshipsTool {
	t.Helper()
	return &EvidenceRelationshipsTool{
		config: newTestConfig(t.TempDir()),
		logger: testhelpers.NewStubLogger(),
	}
}

// ---------------------------------------------------------------------------
// calculateRelevance
// ---------------------------------------------------------------------------

func TestEvidenceRelationshipsTool_CalculateRelevance(t *testing.T) {
	t.Parallel()

	ert := newEvidenceRelationshipsToolForPureTesting(t)

	tests := map[string]struct {
		task  *domain.EvidenceTask
		depth int
		want  float64
	}{
		"no controls, depth 1": {
			task:  &domain.EvidenceTask{Controls: nil},
			depth: 1,
			want:  0.7,
		},
		"1 control, depth 1": {
			task:  &domain.EvidenceTask{Controls: []string{"CC-06.1"}},
			depth: 1,
			want:  0.8,
		},
		"5 controls, depth 2": {
			task:  &domain.EvidenceTask{Controls: []string{"A", "B", "C", "D", "E"}},
			depth: 2,
			want:  0.9,
		},
		"10 controls, depth 3": {
			task:  &domain.EvidenceTask{Controls: make([]string, 10)},
			depth: 3,
			want:  1.0,
		},
		"capped at 1.0": {
			task:  &domain.EvidenceTask{Controls: make([]string, 20)},
			depth: 3,
			want:  1.0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := ert.calculateRelevance(tc.task, tc.depth)
			assert.InDelta(t, tc.want, got, 0.01)
		})
	}
}

// ---------------------------------------------------------------------------
// buildBasicDependencyGraph
// ---------------------------------------------------------------------------

func TestEvidenceRelationshipsTool_BuildBasicDependencyGraph(t *testing.T) {
	t.Parallel()

	ert := newEvidenceRelationshipsToolForPureTesting(t)

	task := &domain.EvidenceTask{
		ID: "327992",
		Name:      "GitHub Access Controls",
		Framework: "SOC2",
		Status:    "pending",
		Controls:  []string{"CC-06.1", "CC-06.3"},
	}

	graph := ert.buildBasicDependencyGraph(task)
	require.NotNil(t, graph)

	// Should have nodes: 1 task + 2 controls = 3
	nodes := graph["nodes"].([]map[string]interface{})
	assert.Len(t, nodes, 3)

	// Should have edges: 1 task -> 2 controls = 2
	edges := graph["edges"].([]map[string]interface{})
	assert.Len(t, edges, 2)

	// First node should be the task
	assert.Equal(t, "evidence_task", nodes[0]["type"])
	assert.Contains(t, nodes[0]["id"], "task_327992")
}

// ---------------------------------------------------------------------------
// buildExtendedRelationships
// ---------------------------------------------------------------------------

func TestEvidenceRelationshipsTool_BuildExtendedRelationships(t *testing.T) {
	t.Parallel()

	ert := newEvidenceRelationshipsToolForPureTesting(t)

	task := &domain.EvidenceTask{
		ID: "327992",
		Name:      "Test Task",
		Framework: "SOC2",
		Controls:  []string{"CC-06.1", "CC-06.3"},
	}

	t.Run("depth 2 with controls", func(t *testing.T) {
		t.Parallel()
		extended := ert.buildExtendedRelationships(nil, task, 2, true, true)
		assert.NotNil(t, extended)
		assert.Contains(t, extended, "framework_mapping")
		assert.Contains(t, extended, "cross_references")
	})

	t.Run("depth 3 includes dependency graph", func(t *testing.T) {
		t.Parallel()
		extended := ert.buildExtendedRelationships(nil, task, 3, true, true)
		assert.NotNil(t, extended)
		assert.Contains(t, extended, "dependency_graph")
	})

	t.Run("without controls", func(t *testing.T) {
		t.Parallel()
		extended := ert.buildExtendedRelationships(nil, task, 2, false, false)
		assert.NotNil(t, extended)
	})

	t.Run("empty framework", func(t *testing.T) {
		t.Parallel()
		noFwTask := &domain.EvidenceTask{Controls: []string{"A"}}
		extended := ert.buildExtendedRelationships(nil, noFwTask, 2, true, true)
		assert.NotNil(t, extended)
	})
}

// ---------------------------------------------------------------------------
// buildRelationshipGraph (needs a nil-safe dataStore usage path)
// The dataStore calls will fail gracefully (controls/policies just won't resolve)
// ---------------------------------------------------------------------------

func TestEvidenceRelationshipsTool_BuildRelationshipGraph(t *testing.T) {
	t.Parallel()

	ert := newEvidenceRelationshipsToolForPureTesting(t)

	t.Run("task without controls, depth 1", func(t *testing.T) {
		t.Parallel()
		task := &domain.EvidenceTask{
			ID: "327992",
			ReferenceID: "ET-0001",
			Name:        "GitHub Access",
			Framework:   "SOC2",
			Status:      "pending",
			Priority:    "high",
			Description: "Access control evidence",
			Controls:    nil, // No controls to avoid nil dataStore panic
		}

		graph, err := ert.buildRelationshipGraph(nil, task, 1, false, false)
		require.NoError(t, err)
		require.NotNil(t, graph)

		// Check task info
		taskInfo := graph["task"].(map[string]interface{})
		assert.Equal(t, "327992", taskInfo["id"])
		assert.Equal(t, "ET-0001", taskInfo["reference_id"])

		// Check recommendations
		recs := graph["recommendations"].([]string)
		assert.NotEmpty(t, recs)

		// Check suggested tools
		tools := graph["suggested_tools"].([]string)
		assert.NotEmpty(t, tools)

		// Check compliance context
		cc := graph["compliance_context"].(map[string]interface{})
		assert.Equal(t, "SOC2", cc["framework"])
	})

	t.Run("task depth 2 adds extended", func(t *testing.T) {
		t.Parallel()
		task := &domain.EvidenceTask{
			ID: "1",
			Controls: []string{"A", "B"},
		}
		// Don't include controls/policies from dataStore (nil)
		graph, err := ert.buildRelationshipGraph(nil, task, 2, false, false)
		require.NoError(t, err)
		rels := graph["relationships"].(map[string]interface{})
		assert.Contains(t, rels, "extended")
	})

	t.Run("without controls or policies depth 1", func(t *testing.T) {
		t.Parallel()
		task := &domain.EvidenceTask{ID: "1"}
		graph, err := ert.buildRelationshipGraph(nil, task, 1, false, false)
		require.NoError(t, err)
		require.NotNil(t, graph)
	})
}

// ---------------------------------------------------------------------------
// Metadata methods
// ---------------------------------------------------------------------------

func TestEvidenceRelationshipsTool_Metadata(t *testing.T) {
	t.Parallel()

	ert := newEvidenceRelationshipsToolForPureTesting(t)
	assert.Equal(t, "evidence-relationships", ert.Name())
	assert.NotEmpty(t, ert.Description())
	assert.Equal(t, "evidence", ert.Category())
	assert.Equal(t, "1.0.0", ert.Version())

	def := ert.GetClaudeToolDefinition()
	assert.Equal(t, "evidence-relationships", def.Name)
}

// ---------------------------------------------------------------------------
// EvidenceTaskDetailsTool metadata
// ---------------------------------------------------------------------------

func TestEvidenceTaskDetailsTool_Metadata(t *testing.T) {
	t.Parallel()

	ett := &EvidenceTaskDetailsTool{
		config: newTestConfig(t.TempDir()),
		logger: testhelpers.NewStubLogger(),
	}
	assert.Equal(t, "evidence-task-details", ett.Name())
	assert.NotEmpty(t, ett.Description())
	assert.Equal(t, "evidence", ett.Category())
	assert.Equal(t, "1.0.0", ett.Version())

	def := ett.GetClaudeToolDefinition()
	assert.Equal(t, "evidence-task-details", def.Name)
}
