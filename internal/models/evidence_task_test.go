package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvidenceTask_GetStatus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		task      EvidenceTask
		expected  string
	}{
		{
			name:     "explicit status returned as-is",
			task:     EvidenceTask{Status: "in_progress"},
			expected: "in_progress",
		},
		{
			name:     "completed flag returns completed",
			task:     EvidenceTask{Completed: true},
			expected: "completed",
		},
		{
			name:     "not completed and no status returns pending",
			task:     EvidenceTask{Completed: false},
			expected: "pending",
		},
		{
			name:     "explicit status takes precedence over completed",
			task:     EvidenceTask{Status: "review", Completed: true},
			expected: "review",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.task.GetStatus())
		})
	}
}

func TestEvidenceTask_GetPriority(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		task     EvidenceTask
		expected string
	}{
		{
			name:     "explicit priority returned",
			task:     EvidenceTask{Priority: "critical"},
			expected: "critical",
		},
		{
			name:     "yearly interval is low",
			task:     EvidenceTask{CollectionInterval: "year"},
			expected: "low",
		},
		{
			name:     "quarterly interval is medium",
			task:     EvidenceTask{CollectionInterval: "quarter"},
			expected: "medium",
		},
		{
			name:     "monthly interval is high",
			task:     EvidenceTask{CollectionInterval: "month"},
			expected: "high",
		},
		{
			name:     "weekly interval is high",
			task:     EvidenceTask{CollectionInterval: "week"},
			expected: "high",
		},
		{
			name:     "unknown interval defaults to medium",
			task:     EvidenceTask{CollectionInterval: "unknown"},
			expected: "medium",
		},
		{
			name:     "empty interval defaults to medium",
			task:     EvidenceTask{},
			expected: "medium",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.task.GetPriority())
		})
	}
}

func TestEvidenceTask_GetFramework(t *testing.T) {
	t.Parallel()
	t.Run("returns framework", func(t *testing.T) {
		t.Parallel()
		task := EvidenceTask{Framework: "SOC2"}
		assert.Equal(t, "SOC2", task.GetFramework())
	})
	t.Run("empty framework", func(t *testing.T) {
		t.Parallel()
		task := EvidenceTask{}
		assert.Equal(t, "", task.GetFramework())
	})
}

func TestEvidenceTask_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	dueDate := now.Add(72 * time.Hour)
	lastCollected := "2025-01-01"

	task := EvidenceTask{
		ID: "327992",
		Name:               "GitHub Repository Access Controls",
		Description:        "Show team permissions",
		Guidance:           "Extract from GitHub",
		AdHoc:              false,
		LastCollected:      &lastCollected,
		CollectionInterval: "quarter",
		DueDaysBefore:      14,
		MasterVersionNum:   2,
		Sensitive:          true,
		OrgID:              100,
		Completed:          false,
		MasterEvidenceID:   999,
		Created:            now,
		EntityType:         "evidence_task",
		PolicyID:           "POL-0001",
		ProcedureID:        "PROC-0001",
		ControlID:          "CC-06.1",
		Status:             "pending",
		Priority:           "high",
		Framework:          "SOC2",
		ReferenceID:        "ET-0047",
		DueDate:            &dueDate,
		AssignedTo:         "user@example.com",
		Requirements:       map[string]string{"req1": "value1"},
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	data, err := json.Marshal(task)
	require.NoError(t, err)

	var decoded EvidenceTask
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, task.ID, decoded.ID)
	assert.Equal(t, task.Name, decoded.Name)
	assert.Equal(t, task.Sensitive, decoded.Sensitive)
	assert.Equal(t, task.CollectionInterval, decoded.CollectionInterval)
	assert.NotNil(t, decoded.LastCollected)
	assert.Equal(t, lastCollected, *decoded.LastCollected)
	assert.Equal(t, task.Priority, decoded.Priority)
	assert.Equal(t, task.Framework, decoded.Framework)
	assert.NotNil(t, decoded.DueDate)
	assert.NotNil(t, decoded.Requirements)
}

func TestEvidenceTask_JSONRoundTrip_NilOptionals(t *testing.T) {
	t.Parallel()
	task := EvidenceTask{
		ID: "1",
		Name: "Minimal task",
	}

	data, err := json.Marshal(task)
	require.NoError(t, err)

	var decoded EvidenceTask
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "1", decoded.ID)
	assert.Nil(t, decoded.LastCollected)
	assert.Nil(t, decoded.DueDate)
	assert.Nil(t, decoded.EntityRole)
}

func TestEvidenceTaskDetails_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	details := EvidenceTaskDetails{
		EvidenceTask: EvidenceTask{
			ID: "123",
			Name: "Test Task",
		},
		MasterContent: &MasterContent{
			ID:      "mc-1",
			Title:   "Master",
			Content: "Content here",
		},
		Tags: []EvidenceTag{
			{ID: "t1", Name: "security", Color: "#ff0000"},
		},
		Assignees: []EvidenceAssignee{
			{ID: "a1", Name: "Alice", Email: "alice@test.com"},
		},
		OpenIncidentCount: 2,
		SubtaskMetadata: &SubtaskMetadata{
			TotalSubtasks:     5,
			CompletedSubtasks: 3,
			PendingSubtasks:   2,
		},
		LastRemindedAt: &now,
	}

	data, err := json.Marshal(details)
	require.NoError(t, err)

	var decoded EvidenceTaskDetails
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "123", decoded.ID)
	assert.NotNil(t, decoded.MasterContent)
	assert.Equal(t, "Master", decoded.MasterContent.Title)
	assert.Len(t, decoded.Tags, 1)
	assert.Len(t, decoded.Assignees, 1)
	assert.Equal(t, 2, decoded.OpenIncidentCount)
	assert.NotNil(t, decoded.SubtaskMetadata)
	assert.Equal(t, 5, decoded.SubtaskMetadata.TotalSubtasks)
}

func TestEvidenceTaskSummary_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	summary := EvidenceTaskSummary{
		Total:      10,
		ByStatus:   map[string]int{"pending": 5, "completed": 5},
		ByPriority: map[string]int{"high": 3, "medium": 7},
		Overdue:    2,
		DueSoon:    3,
		LastSync:   time.Now().UTC().Truncate(time.Second),
	}

	data, err := json.Marshal(summary)
	require.NoError(t, err)

	var decoded EvidenceTaskSummary
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, summary.Total, decoded.Total)
	assert.Equal(t, summary.Overdue, decoded.Overdue)
	assert.Equal(t, summary.DueSoon, decoded.DueSoon)
	assert.Equal(t, 5, decoded.ByStatus["pending"])
}

func TestEvidenceFilter_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	filter := EvidenceFilter{
		Status:     []string{"pending", "in_progress"},
		Priority:   []string{"high"},
		Framework:  "SOC2",
		AssignedTo: "user@test.com",
		DueBefore:  &now,
	}

	data, err := json.Marshal(filter)
	require.NoError(t, err)

	var decoded EvidenceFilter
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, filter.Status, decoded.Status)
	assert.Equal(t, filter.Framework, decoded.Framework)
	assert.NotNil(t, decoded.DueBefore)
}

func TestAecStatus_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	aec := AecStatus{
		ID:                "aec-1",
		Status:            "active",
		LastExecuted:      &now,
		NextScheduled:     &now,
		SuccessfulRuns:    10,
		FailedRuns:        2,
		LastSuccessfulRun: &now,
	}

	data, err := json.Marshal(aec)
	require.NoError(t, err)

	var decoded AecStatus
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, aec.ID, decoded.ID)
	assert.Equal(t, aec.SuccessfulRuns, decoded.SuccessfulRuns)
	assert.Equal(t, aec.FailedRuns, decoded.FailedRuns)
}
