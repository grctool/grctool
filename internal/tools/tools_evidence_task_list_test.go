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

// ---------------------------------------------------------------------------
// helpers — create EvidenceTaskListTool for filter testing
// ---------------------------------------------------------------------------

func newTaskListToolForFilterTesting(t *testing.T) *EvidenceTaskListTool {
	t.Helper()
	return &EvidenceTaskListTool{
		config: newTestConfig(t.TempDir()),
		logger: testhelpers.NewStubLogger(),
		// dataStore is nil — we won't call Execute, only filter methods
	}
}

// ---------------------------------------------------------------------------
// EvidenceTaskListTool metadata
// ---------------------------------------------------------------------------

func TestEvidenceTaskListTool_Metadata(t *testing.T) {
	t.Parallel()
	e := newTaskListToolForFilterTesting(t)

	assert.Equal(t, "evidence-task-list", e.Name())
	assert.NotEmpty(t, e.Description())
	def := e.GetClaudeToolDefinition()
	assert.Equal(t, "evidence-task-list", def.Name)
}

// ---------------------------------------------------------------------------
// buildFilterFromParams
// ---------------------------------------------------------------------------

func TestEvidenceTaskListTool_BuildFilterFromParams(t *testing.T) {
	t.Parallel()

	e := newTaskListToolForFilterTesting(t)

	t.Run("empty params", func(t *testing.T) {
		t.Parallel()
		filter := e.buildFilterFromParams(map[string]interface{}{})
		assert.Empty(t, filter.Status)
		assert.Empty(t, filter.Priority)
		assert.Empty(t, filter.Framework)
	})

	t.Run("all params set", func(t *testing.T) {
		t.Parallel()
		filter := e.buildFilterFromParams(map[string]interface{}{
			"status":          []interface{}{"pending", "completed"},
			"framework":       "SOC2",
			"priority":        []interface{}{"high"},
			"category":        []interface{}{"Infrastructure"},
			"assignee":        "alice",
			"aec_status":      []interface{}{"enabled"},
			"collection_type": []interface{}{"Automated"},
			"complexity":      []interface{}{"Simple", "Complex"},
		})
		assert.Equal(t, []string{"pending", "completed"}, filter.Status)
		assert.Equal(t, "SOC2", filter.Framework)
		assert.Equal(t, []string{"high"}, filter.Priority)
		assert.Equal(t, []string{"Infrastructure"}, filter.Category)
		assert.Equal(t, "alice", filter.AssignedTo)
		assert.Equal(t, []string{"enabled"}, filter.AecStatus)
		assert.Equal(t, []string{"Automated"}, filter.CollectionType)
		assert.Equal(t, []string{"Simple", "Complex"}, filter.ComplexityLevel)
	})
}

// ---------------------------------------------------------------------------
// stringInSlice
// ---------------------------------------------------------------------------

func TestEvidenceTaskListTool_StringInSlice(t *testing.T) {
	t.Parallel()

	e := newTaskListToolForFilterTesting(t)

	assert.True(t, e.stringInSlice("high", []string{"high", "medium", "low"}))
	assert.True(t, e.stringInSlice("HIGH", []string{"high", "medium"})) // case insensitive
	assert.False(t, e.stringInSlice("critical", []string{"high", "medium", "low"}))
	assert.False(t, e.stringInSlice("x", nil))
}

// ---------------------------------------------------------------------------
// getTaskStatus
// ---------------------------------------------------------------------------

func TestEvidenceTaskListTool_GetTaskStatus(t *testing.T) {
	t.Parallel()

	e := newTaskListToolForFilterTesting(t)

	t.Run("completed task", func(t *testing.T) {
		t.Parallel()
		task := domain.EvidenceTask{Completed: true}
		assert.Equal(t, "completed", e.getTaskStatus(task))
	})

	t.Run("pending task", func(t *testing.T) {
		t.Parallel()
		task := domain.EvidenceTask{Completed: false, Status: "pending"}
		assert.Equal(t, "pending", e.getTaskStatus(task))
	})

	t.Run("overdue task", func(t *testing.T) {
		t.Parallel()
		task := domain.EvidenceTask{Completed: false, Status: "overdue"}
		assert.Equal(t, "overdue", e.getTaskStatus(task))
	})
}

// ---------------------------------------------------------------------------
// isTaskOverdue / isTaskDueSoon
// ---------------------------------------------------------------------------

func TestEvidenceTaskListTool_IsTaskOverdue(t *testing.T) {
	t.Parallel()

	e := newTaskListToolForFilterTesting(t)

	assert.True(t, e.isTaskOverdue(domain.EvidenceTask{Status: "overdue"}))
	assert.False(t, e.isTaskOverdue(domain.EvidenceTask{Status: "pending"}))
}

func TestEvidenceTaskListTool_IsTaskDueSoon(t *testing.T) {
	t.Parallel()

	e := newTaskListToolForFilterTesting(t)
	// Current implementation always returns false
	assert.False(t, e.isTaskDueSoon(domain.EvidenceTask{}))
}

// ---------------------------------------------------------------------------
// matchesFilter — comprehensive filtering
// ---------------------------------------------------------------------------

func TestEvidenceTaskListTool_MatchesFilter(t *testing.T) {
	t.Parallel()

	e := newTaskListToolForFilterTesting(t)

	baseTasks := domain.EvidenceTask{
		Framework:       "SOC2",
		Priority:        "high",
		Category:        "Infrastructure",
		CollectionType:  "Automated",
		ComplexityLevel: "Moderate",
		Sensitive:       false,
		Status:          "pending",
	}

	t.Run("empty filter matches all", func(t *testing.T) {
		t.Parallel()
		filter := domain.EvidenceFilter{}
		assert.True(t, e.matchesFilter(baseTasks, filter, map[string]interface{}{}))
	})

	t.Run("status filter - matches", func(t *testing.T) {
		t.Parallel()
		filter := domain.EvidenceFilter{Status: []string{"pending"}}
		assert.True(t, e.matchesFilter(baseTasks, filter, map[string]interface{}{}))
	})

	t.Run("status filter - no match", func(t *testing.T) {
		t.Parallel()
		filter := domain.EvidenceFilter{Status: []string{"completed"}}
		assert.False(t, e.matchesFilter(baseTasks, filter, map[string]interface{}{}))
	})

	t.Run("framework filter - matches", func(t *testing.T) {
		t.Parallel()
		filter := domain.EvidenceFilter{Framework: "soc2"} // case insensitive
		assert.True(t, e.matchesFilter(baseTasks, filter, map[string]interface{}{}))
	})

	t.Run("framework filter - no match", func(t *testing.T) {
		t.Parallel()
		filter := domain.EvidenceFilter{Framework: "iso27001"}
		assert.False(t, e.matchesFilter(baseTasks, filter, map[string]interface{}{}))
	})

	t.Run("priority filter", func(t *testing.T) {
		t.Parallel()
		filter := domain.EvidenceFilter{Priority: []string{"high"}}
		assert.True(t, e.matchesFilter(baseTasks, filter, map[string]interface{}{}))

		filter = domain.EvidenceFilter{Priority: []string{"low"}}
		assert.False(t, e.matchesFilter(baseTasks, filter, map[string]interface{}{}))
	})

	t.Run("category filter", func(t *testing.T) {
		t.Parallel()
		filter := domain.EvidenceFilter{Category: []string{"Infrastructure"}}
		assert.True(t, e.matchesFilter(baseTasks, filter, map[string]interface{}{}))
	})

	t.Run("collection type filter", func(t *testing.T) {
		t.Parallel()
		filter := domain.EvidenceFilter{CollectionType: []string{"Automated"}}
		assert.True(t, e.matchesFilter(baseTasks, filter, map[string]interface{}{}))
	})

	t.Run("complexity filter", func(t *testing.T) {
		t.Parallel()
		filter := domain.EvidenceFilter{ComplexityLevel: []string{"Moderate"}}
		assert.True(t, e.matchesFilter(baseTasks, filter, map[string]interface{}{}))
	})

	t.Run("sensitive filter", func(t *testing.T) {
		t.Parallel()
		filter := domain.EvidenceFilter{}
		params := map[string]interface{}{"sensitive": true}
		assert.False(t, e.matchesFilter(baseTasks, filter, params))

		sensitiveTask := baseTasks
		sensitiveTask.Sensitive = true
		assert.True(t, e.matchesFilter(sensitiveTask, filter, params))
	})

	t.Run("assignee filter", func(t *testing.T) {
		t.Parallel()
		taskWithAssignee := baseTasks
		taskWithAssignee.Assignees = []domain.Person{
			{Name: "Alice Smith", Email: "alice@example.com"},
		}
		filter := domain.EvidenceFilter{AssignedTo: "alice"}
		assert.True(t, e.matchesFilter(taskWithAssignee, filter, map[string]interface{}{}))

		filter = domain.EvidenceFilter{AssignedTo: "bob"}
		assert.False(t, e.matchesFilter(taskWithAssignee, filter, map[string]interface{}{}))
	})

	t.Run("aec status filter", func(t *testing.T) {
		t.Parallel()
		// Task without AEC status should default to "na"
		filter := domain.EvidenceFilter{AecStatus: []string{"na"}}
		assert.True(t, e.matchesFilter(baseTasks, filter, map[string]interface{}{}))

		filter = domain.EvidenceFilter{AecStatus: []string{"enabled"}}
		assert.False(t, e.matchesFilter(baseTasks, filter, map[string]interface{}{}))
	})
}

// ---------------------------------------------------------------------------
// applyFilters
// ---------------------------------------------------------------------------

func TestEvidenceTaskListTool_ApplyFilters(t *testing.T) {
	t.Parallel()

	e := newTaskListToolForFilterTesting(t)

	tasks := []domain.EvidenceTask{
		{Priority: "high", Framework: "SOC2"},
		{Priority: "low", Framework: "SOC2"},
		{Priority: "high", Framework: "ISO27001"},
	}

	filter := domain.EvidenceFilter{Priority: []string{"high"}}
	filtered := e.applyFilters(tasks, filter, map[string]interface{}{})

	require.Len(t, filtered, 2)
	for _, task := range filtered {
		assert.Equal(t, "high", task.Priority)
	}
}

// ---------------------------------------------------------------------------
// enrichTasksWithURLs
// ---------------------------------------------------------------------------

func TestEvidenceTaskListTool_EnrichTasksWithURLs(t *testing.T) {
	t.Parallel()

	t.Run("no base URL configured", func(t *testing.T) {
		t.Parallel()
		e := newTaskListToolForFilterTesting(t)
		tasks := []domain.EvidenceTask{{ID: 1}}
		e.enrichTasksWithURLs(tasks)
		assert.Empty(t, tasks[0].TugboatURL)
	})

	t.Run("with base URL and org ID", func(t *testing.T) {
		t.Parallel()
		e := newTaskListToolForFilterTesting(t)
		e.config.Tugboat.BaseURL = "https://app.tugboatlogic.com"
		e.config.Tugboat.OrgID = "123"

		tasks := []domain.EvidenceTask{{ID: 456, OrgID: 0}}
		e.enrichTasksWithURLs(tasks)
		assert.NotEmpty(t, tasks[0].TugboatURL)
	})
}
