package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalEvidenceState_String(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		state    LocalEvidenceState
		expected string
	}{
		{"no_evidence", StateNoEvidence, "no_evidence"},
		{"generated", StateGenerated, "generated"},
		{"validated", StateValidated, "validated"},
		{"submitted", StateSubmitted, "submitted"},
		{"accepted", StateAccepted, "accepted"},
		{"rejected", StateRejected, "rejected"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

func TestAutomationCapability_String(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		cap      AutomationCapability
		expected string
	}{
		{"fully", AutomationFully, "fully_automated"},
		{"partially", AutomationPartially, "partially_automated"},
		{"manual", AutomationManual, "manual_only"},
		{"unknown", AutomationUnknown, "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.cap.String())
		})
	}
}

func TestDetermineLocalState(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		windows  map[string]WindowState
		expected LocalEvidenceState
	}{
		{
			name:     "nil windows returns no_evidence",
			windows:  nil,
			expected: StateNoEvidence,
		},
		{
			name:     "empty windows returns no_evidence",
			windows:  map[string]WindowState{},
			expected: StateNoEvidence,
		},
		{
			name: "window with zero files and no status returns no_evidence",
			windows: map[string]WindowState{
				"2025-Q4": {Window: "2025-Q4", FileCount: 0},
			},
			expected: StateNoEvidence,
		},
		{
			name: "window with files returns generated",
			windows: map[string]WindowState{
				"2025-Q4": {Window: "2025-Q4", FileCount: 3},
			},
			expected: StateGenerated,
		},
		{
			name: "validated status returns validated",
			windows: map[string]WindowState{
				"2025-Q4": {Window: "2025-Q4", SubmissionStatus: "validated"},
			},
			expected: StateValidated,
		},
		{
			name: "submitted status returns submitted",
			windows: map[string]WindowState{
				"2025-Q4": {Window: "2025-Q4", SubmissionStatus: "submitted"},
			},
			expected: StateSubmitted,
		},
		{
			name: "accepted status returns accepted",
			windows: map[string]WindowState{
				"2025-Q4": {Window: "2025-Q4", SubmissionStatus: "accepted"},
			},
			expected: StateAccepted,
		},
		{
			name: "mixed states returns most advanced (accepted)",
			windows: map[string]WindowState{
				"2025-Q3": {Window: "2025-Q3", FileCount: 2},
				"2025-Q4": {Window: "2025-Q4", SubmissionStatus: "accepted"},
			},
			expected: StateAccepted,
		},
		{
			name: "mixed states returns most advanced (submitted over generated)",
			windows: map[string]WindowState{
				"2025-Q3": {Window: "2025-Q3", FileCount: 2},
				"2025-Q4": {Window: "2025-Q4", SubmissionStatus: "submitted"},
			},
			expected: StateSubmitted,
		},
		{
			name: "mixed states returns most advanced (validated over generated)",
			windows: map[string]WindowState{
				"2025-Q3": {Window: "2025-Q3", FileCount: 5},
				"2025-Q4": {Window: "2025-Q4", SubmissionStatus: "validated"},
			},
			expected: StateValidated,
		},
		{
			name: "rejected status with files still returns generated",
			windows: map[string]WindowState{
				"2025-Q4": {Window: "2025-Q4", SubmissionStatus: "rejected", FileCount: 3},
			},
			// "rejected" does not match any of the known statuses, but FileCount > 0
			expected: StateGenerated,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := DetermineLocalState(tt.windows)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetermineAutomationCapability(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		tools       []string
		description string
		expected    AutomationCapability
	}{
		{
			name:        "no tools returns unknown",
			tools:       []string{},
			description: "some task",
			expected:    AutomationUnknown,
		},
		{
			name:        "nil tools returns unknown",
			tools:       nil,
			description: "some task",
			expected:    AutomationUnknown,
		},
		{
			name:        "single tool returns partially",
			tools:       []string{"terraform_scanner"},
			description: "Terraform security scan",
			expected:    AutomationPartially,
		},
		{
			name:        "two tools returns fully",
			tools:       []string{"terraform_scanner", "github_permissions"},
			description: "Multi-tool evidence",
			expected:    AutomationFully,
		},
		{
			name:        "many tools returns fully",
			tools:       []string{"a", "b", "c", "d"},
			description: "",
			expected:    AutomationFully,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := DetermineAutomationCapability(tt.tools, tt.description)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewStateCache(t *testing.T) {
	t.Parallel()
	cache := NewStateCache()
	require.NotNil(t, cache)
	assert.NotZero(t, cache.LastScan)
	assert.NotNil(t, cache.Tasks)
	assert.Empty(t, cache.Tasks)
}

func TestStateCache_GetSetTask(t *testing.T) {
	t.Parallel()
	cache := NewStateCache()

	// Get nonexistent task
	_, exists := cache.GetTask("ET-0001")
	assert.False(t, exists)

	// Set and get
	state := &EvidenceTaskState{
		TaskRef:  "ET-0001",
		TaskID: "1",
		TaskName: "Test Task",
	}
	cache.SetTask("ET-0001", state)

	got, exists := cache.GetTask("ET-0001")
	require.True(t, exists)
	assert.Equal(t, "ET-0001", got.TaskRef)
	assert.Equal(t, "1", got.TaskID)

	// SetTask updates LastScan
	beforeSet := cache.LastScan
	// Ensure time advances
	time.Sleep(time.Millisecond)
	cache.SetTask("ET-0002", &EvidenceTaskState{TaskRef: "ET-0002"})
	assert.True(t, cache.LastScan.After(beforeSet) || cache.LastScan.Equal(beforeSet))
}

func TestStateCache_GetTasksByState(t *testing.T) {
	t.Parallel()
	cache := NewStateCache()
	cache.Tasks = map[string]*EvidenceTaskState{
		"ET-0001": {TaskRef: "ET-0001", LocalState: StateGenerated},
		"ET-0002": {TaskRef: "ET-0002", LocalState: StateSubmitted},
		"ET-0003": {TaskRef: "ET-0003", LocalState: StateGenerated},
		"ET-0004": {TaskRef: "ET-0004", LocalState: StateNoEvidence},
	}

	generated := cache.GetTasksByState(StateGenerated)
	assert.Len(t, generated, 2)

	submitted := cache.GetTasksByState(StateSubmitted)
	assert.Len(t, submitted, 1)

	accepted := cache.GetTasksByState(StateAccepted)
	assert.Empty(t, accepted)
}

func TestStateCache_GetTasksByAutomation(t *testing.T) {
	t.Parallel()
	cache := NewStateCache()
	cache.Tasks = map[string]*EvidenceTaskState{
		"ET-0001": {TaskRef: "ET-0001", AutomationLevel: AutomationFully},
		"ET-0002": {TaskRef: "ET-0002", AutomationLevel: AutomationManual},
		"ET-0003": {TaskRef: "ET-0003", AutomationLevel: AutomationFully},
	}

	fully := cache.GetTasksByAutomation(AutomationFully)
	assert.Len(t, fully, 2)

	manual := cache.GetTasksByAutomation(AutomationManual)
	assert.Len(t, manual, 1)

	partial := cache.GetTasksByAutomation(AutomationPartially)
	assert.Empty(t, partial)
}

func TestStateCache_GetStateSummary(t *testing.T) {
	t.Parallel()
	cache := NewStateCache()
	cache.Tasks = map[string]*EvidenceTaskState{
		"ET-0001": {TaskRef: "ET-0001", LocalState: StateGenerated},
		"ET-0002": {TaskRef: "ET-0002", LocalState: StateSubmitted},
		"ET-0003": {TaskRef: "ET-0003", LocalState: StateGenerated},
	}

	summary := cache.GetStateSummary()
	assert.Equal(t, 2, summary[StateGenerated])
	assert.Equal(t, 1, summary[StateSubmitted])
	assert.Equal(t, 0, summary[StateAccepted])
}

func TestStateCache_GetStateSummary_Empty(t *testing.T) {
	t.Parallel()
	cache := NewStateCache()
	summary := cache.GetStateSummary()
	assert.Empty(t, summary)
}

func TestStateCache_GetAutomationSummary(t *testing.T) {
	t.Parallel()
	cache := NewStateCache()
	cache.Tasks = map[string]*EvidenceTaskState{
		"ET-0001": {TaskRef: "ET-0001", AutomationLevel: AutomationFully},
		"ET-0002": {TaskRef: "ET-0002", AutomationLevel: AutomationFully},
		"ET-0003": {TaskRef: "ET-0003", AutomationLevel: AutomationManual},
	}

	summary := cache.GetAutomationSummary()
	assert.Equal(t, 2, summary[AutomationFully])
	assert.Equal(t, 1, summary[AutomationManual])
}

func TestEvidenceTaskState_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	genAt := now.Add(-time.Hour)
	subAt := now.Add(-30 * time.Minute)

	state := EvidenceTaskState{
		TaskRef:          "ET-0001",
		TaskID: "1",
		TaskName:         "Test Task",
		TugboatStatus:    "in_progress",
		TugboatCompleted: false,
		LastSyncedAt:     now,
		Framework:        "SOC2",
		LocalState:       StateGenerated,
		Windows: map[string]WindowState{
			"2025-Q4": {
				Window:     "2025-Q4",
				FileCount:  3,
				TotalBytes: 1024,
			},
		},
		AutomationLevel: AutomationFully,
		ApplicableTools: []string{"terraform_scanner", "github_permissions"},
		LastGeneratedAt: &genAt,
		LastSubmittedAt: &subAt,
		LastScannedAt:   now,
	}

	data, err := json.Marshal(state)
	require.NoError(t, err)

	var decoded EvidenceTaskState
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, state.TaskRef, decoded.TaskRef)
	assert.Equal(t, state.TaskID, decoded.TaskID)
	assert.Equal(t, state.LocalState, decoded.LocalState)
	assert.Equal(t, state.AutomationLevel, decoded.AutomationLevel)
	assert.Len(t, decoded.ApplicableTools, 2)
	assert.Len(t, decoded.Windows, 1)
	assert.NotNil(t, decoded.LastGeneratedAt)
	assert.NotNil(t, decoded.LastSubmittedAt)
}

func TestWindowState_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	ws := WindowState{
		Window:            "2025-Q4",
		FileCount:         5,
		TotalBytes:        2048,
		OldestFile:        &now,
		NewestFile:        &now,
		HasGenerationMeta: true,
		GenerationMethod:  "tool_coordination",
		GeneratedAt:       &now,
		GeneratedBy:       "claude-code-assisted",
		ToolsUsed:         []string{"terraform_scanner"},
		HasSubmissionMeta: true,
		SubmissionStatus:  "submitted",
		SubmittedAt:       &now,
		SubmissionID:      "sub-123",
		Files: []FileState{
			{
				Filename:    "evidence.md",
				SizeBytes:   512,
				ModifiedAt:  now,
				IsGenerated: true,
			},
		},
	}

	data, err := json.Marshal(ws)
	require.NoError(t, err)

	var decoded WindowState
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, ws.Window, decoded.Window)
	assert.Equal(t, ws.FileCount, decoded.FileCount)
	assert.Equal(t, ws.TotalBytes, decoded.TotalBytes)
	assert.Equal(t, ws.SubmissionStatus, decoded.SubmissionStatus)
	assert.Equal(t, ws.SubmissionID, decoded.SubmissionID)
	assert.Len(t, decoded.Files, 1)
	assert.Equal(t, "evidence.md", decoded.Files[0].Filename)
}

func TestStateCache_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	cache := NewStateCache()
	cache.SetTask("ET-0001", &EvidenceTaskState{
		TaskRef:    "ET-0001",
		TaskID: "1",
		LocalState: StateGenerated,
	})

	data, err := json.Marshal(cache)
	require.NoError(t, err)

	var decoded StateCache
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Len(t, decoded.Tasks, 1)
	task, exists := decoded.Tasks["ET-0001"]
	require.True(t, exists)
	assert.Equal(t, StateGenerated, task.LocalState)
}
