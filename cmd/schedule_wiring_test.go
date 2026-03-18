// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/scheduler"
	"github.com/grctool/grctool/internal/testhelpers"
	"github.com/stretchr/testify/assert"
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
	}
	printCollectionSummary(&buf, summary)
	output := buf.String()
	assert.Contains(t, output, "Tasks: 3")
	assert.Contains(t, output, "Tools: 7")
	assert.Contains(t, output, "Succeeded: 5")
	assert.Contains(t, output, "Failed: 1")
	assert.Contains(t, output, "Skipped: 1")
	assert.Contains(t, output, "2.345s")
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
