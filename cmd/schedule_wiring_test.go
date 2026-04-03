// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/scheduler"
	"github.com/grctool/grctool/internal/storage"
	"github.com/grctool/grctool/internal/testhelpers"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintCollectionSummary(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	summary := &scheduler.CollectionSummary{
		TotalTasks: 3,
		TotalTools: 7,
		Succeeded:  5,
		Failed:     1,
		Skipped:    1,
		Duration:   2*time.Second + 345*time.Millisecond,
		Results: []scheduler.CollectionResult{
			{TaskRef: "ET-0001", ToolName: "github-permissions", Success: true, Duration: 500 * time.Millisecond},
			{TaskRef: "ET-0001", ToolName: "github-security", Success: false, Error: "auth required", Duration: 100 * time.Millisecond},
			{TaskRef: "ET-0002", ToolName: "terraform-analyzer", Success: true, Duration: 1 * time.Second},
		},
	}
	printCollectionSummary(&buf, summary)
	output := buf.String()
	assert.Contains(t, output, "Tasks: 3")
	assert.Contains(t, output, "Tools: 7")
	assert.Contains(t, output, "Succeeded: 5")
	assert.Contains(t, output, "Failed: 1")
	assert.Contains(t, output, "Skipped: 1")
	assert.Contains(t, output, "2.345s")
	// Per-task results
	assert.Contains(t, output, "Task ET-0001:")
	assert.Contains(t, output, "github-permissions: ok")
	assert.Contains(t, output, "github-security: FAILED: auth required")
	assert.Contains(t, output, "Task ET-0002:")
	assert.Contains(t, output, "terraform-analyzer: ok")
}

func TestPrintCollectionSummary_Empty(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	summary := &scheduler.CollectionSummary{}
	printCollectionSummary(&buf, summary)
	output := buf.String()
	assert.Contains(t, output, "Tasks: 0")
	assert.Contains(t, output, "Succeeded: 0")
}

func TestExecuteSchedule_NoMappings(t *testing.T) {
	t.Parallel()
	log := testhelpers.NewStubLogger()
	orch := scheduler.NewOrchestrator(nil, log)

	sched := scheduler.Schedule{Name: "test-schedule", Scope: "all"}
	summary := executeSchedule(context.Background(), orch, sched)
	assert.Equal(t, 0, summary.TotalTasks)
	assert.Equal(t, 0, summary.Failed)
}

func TestExecuteSchedule_WithMappings(t *testing.T) {
	t.Parallel()
	log := testhelpers.NewStubLogger()
	mappings := []scheduler.TaskToolMapping{
		{TaskRef: "ET-0001", Tools: []string{"test-tool"}, Schedule: "nightly"},
	}
	orch := scheduler.NewOrchestrator(mappings, log)

	// Use a stub executor that always succeeds.
	sched := scheduler.Schedule{Name: "nightly", Scope: "all"}

	// Test with matching schedule name — plan has 1 task.
	summary := executeSchedule(context.Background(), orch, sched)
	assert.Equal(t, 1, summary.TotalTasks)
	// Tool "test-tool" isn't registered, so it fails, but plan was built.
	assert.Equal(t, 1, summary.Failed)
}

func TestLoadOrchestrator(t *testing.T) {
	t.Parallel()
	// loadOrchestrator creates an orchestrator with nil mappings.
	orch := loadOrchestrator(nil)
	assert.NotNil(t, orch)
	// With nil mappings, BuildPlan returns empty.
	plan := orch.BuildPlan(nil)
	assert.NotNil(t, plan)
	assert.Empty(t, plan.Tasks)
}

func TestScheduleRunCmd_DescriptionUpdated(t *testing.T) {
	t.Parallel()
	// Verify the TODO note is gone from the command description.
	assert.NotContains(t, scheduleRunCmd.Long, "TODO")
	assert.NotContains(t, scheduleRunCmd.Long, "will be wired")
	assert.Contains(t, scheduleRunCmd.Long, "orchestrator")
}

func TestPrintTaskDueGroupings(t *testing.T) {
	t.Parallel()
	// Create a temp directory with a config and some evidence tasks with due dates.
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")

	configContent := `
tugboat:
  base_url: "https://api-test.example.com"
  org_id: "test"
  timeout: "30s"
  rate_limit: 10
  cookie_header: "test-cookie"

storage:
  data_dir: "` + dataDir + `"
`
	configFile := filepath.Join(tempDir, ".grctool.yaml")
	require.NoError(t, os.WriteFile(configFile, []byte(configContent), 0o644))

	viper.Reset()
	viper.SetConfigFile(configFile)
	require.NoError(t, viper.ReadInConfig())

	cfg, err := config.Load()
	require.NoError(t, err)

	store, err := storage.NewStorage(cfg.Storage)
	require.NoError(t, err)

	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)

	// Create evidence tasks with different due dates.
	overdueDue := now.Add(-5 * 24 * time.Hour)
	weekDue := now.Add(3 * 24 * time.Hour)
	monthDue := now.Add(20 * 24 * time.Hour)

	require.NoError(t, store.SaveEvidenceTask(&domain.EvidenceTask{
		ID: "1001", ReferenceID: "ET-0001", Name: "Overdue Task", NextDue: &overdueDue,
	}))
	require.NoError(t, store.SaveEvidenceTask(&domain.EvidenceTask{
		ID: "1002", ReferenceID: "ET-0002", Name: "Due This Week Task", NextDue: &weekDue,
	}))
	require.NoError(t, store.SaveEvidenceTask(&domain.EvidenceTask{
		ID: "1003", ReferenceID: "ET-0003", Name: "Due This Month Task", NextDue: &monthDue,
	}))
	require.NoError(t, store.SaveEvidenceTask(&domain.EvidenceTask{
		ID: "1004", ReferenceID: "ET-0004", Name: "No Schedule Task",
	}))

	var buf bytes.Buffer
	printTaskDueGroupings(&buf, cfg, now)
	output := buf.String()

	assert.Contains(t, output, "Evidence Tasks: Overdue")
	assert.Contains(t, output, "ET-0001")
	assert.Contains(t, output, "Overdue Task")
	assert.Contains(t, output, "overdue")

	assert.Contains(t, output, "Evidence Tasks: Due This Week")
	assert.Contains(t, output, "ET-0002")
	assert.Contains(t, output, "Due This Week Task")

	assert.Contains(t, output, "Evidence Tasks: Due This Month")
	assert.Contains(t, output, "ET-0003")
	assert.Contains(t, output, "Due This Month Task")

	// ET-0004 has no NextDue, should not appear
	assert.NotContains(t, output, "ET-0004")
}
