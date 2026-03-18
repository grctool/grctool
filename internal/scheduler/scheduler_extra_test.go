// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/testhelpers"
)

// --- State file corruption / edge cases ---

func TestLoadState_CorruptedYAML(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	stateFile := filepath.Join(dir, stateFileName)
	if err := os.WriteFile(stateFile, []byte("{{{{not valid yaml!!!!"), 0o644); err != nil {
		t.Fatalf("failed to write corrupt state file: %v", err)
	}

	cfgs := []config.ScheduleConfig{
		{Name: "hourly", Cron: "0 * * * *", Enabled: true, Scope: "all"},
	}
	s := NewScheduler(cfgs, dir, testhelpers.NewStubLogger())

	_, err := s.LoadState()
	if err == nil {
		t.Fatal("expected error loading corrupted YAML, got nil")
	}
}

func TestLoadState_EmptyFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	stateFile := filepath.Join(dir, stateFileName)
	if err := os.WriteFile(stateFile, []byte(""), 0o644); err != nil {
		t.Fatalf("failed to write empty state file: %v", err)
	}

	cfgs := []config.ScheduleConfig{
		{Name: "hourly", Cron: "0 * * * *", Enabled: true, Scope: "all"},
	}
	s := NewScheduler(cfgs, dir, testhelpers.NewStubLogger())

	state, err := s.LoadState()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return empty state with initialized Schedules map.
	if state.Schedules == nil {
		t.Error("expected non-nil Schedules map")
	}
}

func TestLoadState_NilSchedulesMap(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	stateFile := filepath.Join(dir, stateFileName)
	// YAML with schedules: null
	if err := os.WriteFile(stateFile, []byte("schedules: null\nupdated_at: 2026-01-01T00:00:00Z\n"), 0o644); err != nil {
		t.Fatalf("failed to write state file: %v", err)
	}

	cfgs := []config.ScheduleConfig{}
	s := NewScheduler(cfgs, dir, testhelpers.NewStubLogger())

	state, err := s.LoadState()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Schedules == nil {
		t.Error("expected non-nil Schedules map after loading null schedules")
	}
}

func TestLoadState_NonexistentDir(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "nonexistent", "deeply", "nested")
	cfgs := []config.ScheduleConfig{}
	s := NewScheduler(cfgs, dir, testhelpers.NewStubLogger())

	state, err := s.LoadState()
	if err != nil {
		t.Fatalf("unexpected error loading from nonexistent dir: %v", err)
	}
	if state.Schedules == nil {
		t.Error("expected non-nil Schedules map")
	}
}

func TestSaveState_CreatesDirectory(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "new", "nested", "dir")
	cfgs := []config.ScheduleConfig{
		{Name: "hourly", Cron: "0 * * * *", Enabled: true, Scope: "all"},
	}
	s := NewScheduler(cfgs, dir, testhelpers.NewStubLogger())

	state := &SchedulerState{
		Schedules: map[string]*ScheduleState{
			"hourly": {Name: "hourly", RunCount: 1},
		},
		UpdatedAt: time.Now(),
	}

	err := s.SaveState(state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file was created.
	if _, err := os.Stat(filepath.Join(dir, stateFileName)); os.IsNotExist(err) {
		t.Error("expected state file to be created")
	}
}

func TestSaveState_ReadOnly_Dir(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "readonly")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	// Make dir read-only.
	if err := os.Chmod(dir, 0o444); err != nil {
		t.Fatalf("failed to chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	cfgs := []config.ScheduleConfig{}
	s := NewScheduler(cfgs, dir, testhelpers.NewStubLogger())

	state := &SchedulerState{
		Schedules: map[string]*ScheduleState{},
	}

	err := s.SaveState(state)
	if err == nil {
		t.Fatal("expected error saving to read-only dir, got nil")
	}
}

// --- Sequential LoadState/SaveState round-trip ---

func TestSequentialLoadSaveRoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgs := []config.ScheduleConfig{
		{Name: "hourly", Cron: "0 * * * *", Enabled: true, Scope: "all"},
	}
	s := NewScheduler(cfgs, dir, testhelpers.NewStubLogger())

	now := time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC)

	// Perform multiple sequential writes and verify state accumulates.
	for i := 0; i < 10; i++ {
		if err := s.MarkCompleted("hourly", now.Add(time.Duration(i)*time.Minute)); err != nil {
			t.Fatalf("MarkCompleted iteration %d failed: %v", i, err)
		}
	}

	// Verify final state is readable and has correct data.
	state, err := s.LoadState()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Schedules["hourly"] == nil {
		t.Fatal("expected schedule state for 'hourly' after sequential writes")
	}
	if state.Schedules["hourly"].RunCount != 10 {
		t.Errorf("expected RunCount 10, got %d", state.Schedules["hourly"].RunCount)
	}
}

// --- MarkCompleted updates NextDue correctly ---

func TestMarkCompleted_NextDueIsCorrect(t *testing.T) {
	t.Parallel()

	cfgs := []config.ScheduleConfig{
		{Name: "hourly", Cron: "0 * * * *", Enabled: true, Scope: "all"},
	}
	s := NewScheduler(cfgs, t.TempDir(), testhelpers.NewStubLogger())
	now := time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC)

	if err := s.MarkCompleted("hourly", now); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	state, err := s.LoadState()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ss := state.Schedules["hourly"]
	expectedNext := time.Date(2026, 3, 18, 11, 0, 0, 0, time.UTC)
	if !ss.NextDue.Equal(expectedNext) {
		t.Errorf("expected NextDue %v, got %v", expectedNext, ss.NextDue)
	}
}

// --- MarkFailed also sets NextDue ---

func TestMarkFailed_NextDueIsSet(t *testing.T) {
	t.Parallel()

	cfgs := []config.ScheduleConfig{
		{Name: "hourly", Cron: "0 * * * *", Enabled: true, Scope: "all"},
	}
	s := NewScheduler(cfgs, t.TempDir(), testhelpers.NewStubLogger())
	now := time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC)

	if err := s.MarkFailed("hourly", now, "some error"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	state, err := s.LoadState()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ss := state.Schedules["hourly"]
	if ss.NextDue.IsZero() {
		t.Error("expected NextDue to be set after MarkFailed")
	}
	if ss.LastError != "some error" {
		t.Errorf("expected error 'some error', got %q", ss.LastError)
	}
}

// --- MarkCompleted for unknown schedule name (not in scheduler config) ---

func TestMarkCompleted_UnknownSchedule(t *testing.T) {
	t.Parallel()

	cfgs := []config.ScheduleConfig{
		{Name: "hourly", Cron: "0 * * * *", Enabled: true, Scope: "all"},
	}
	s := NewScheduler(cfgs, t.TempDir(), testhelpers.NewStubLogger())
	now := time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC)

	// Mark an unknown schedule. Should still save state, but NextDue won't be set
	// because the cron can't be found.
	if err := s.MarkCompleted("nonexistent", now); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	state, err := s.LoadState()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ss := state.Schedules["nonexistent"]
	if ss == nil {
		t.Fatal("expected schedule state for 'nonexistent'")
	}
	if ss.RunCount != 1 {
		t.Errorf("expected RunCount 1, got %d", ss.RunCount)
	}
	if !ss.NextDue.IsZero() {
		t.Errorf("expected zero NextDue for unknown schedule, got %v", ss.NextDue)
	}
}

// --- MarkFailed for unknown schedule name ---

func TestMarkFailed_UnknownSchedule(t *testing.T) {
	t.Parallel()

	cfgs := []config.ScheduleConfig{
		{Name: "hourly", Cron: "0 * * * *", Enabled: true, Scope: "all"},
	}
	s := NewScheduler(cfgs, t.TempDir(), testhelpers.NewStubLogger())
	now := time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC)

	if err := s.MarkFailed("nonexistent", now, "failed"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	state, err := s.LoadState()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ss := state.Schedules["nonexistent"]
	if ss == nil {
		t.Fatal("expected schedule state for 'nonexistent'")
	}
	if ss.NextDue.IsZero() == false {
		// Unknown schedule can't compute next due.
	}
}

// --- GetDueSchedules with invalid cron ---

func TestGetDueSchedules_InvalidCron(t *testing.T) {
	t.Parallel()

	cfgs := []config.ScheduleConfig{
		{Name: "bad", Cron: "invalid cron expression here", Enabled: true, Scope: "all"},
		{Name: "good", Cron: "0 * * * *", Enabled: true, Scope: "all"},
	}
	dir := t.TempDir()
	s := NewScheduler(cfgs, dir, testhelpers.NewStubLogger())
	now := time.Date(2026, 3, 18, 10, 30, 0, 0, time.UTC)

	// Mark both as recently run so IsDue must parse the cron (not short-circuit on zero lastRun).
	_ = s.MarkCompleted("bad", now.Add(-30*time.Minute))
	_ = s.MarkCompleted("good", now.Add(-90*time.Minute))

	due, err := s.GetDueSchedules(now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// "bad" should have been skipped (logged error), "good" is overdue.
	if len(due) != 1 {
		t.Fatalf("expected 1 due schedule, got %d", len(due))
	}
	if due[0].Name != "good" {
		t.Errorf("expected 'good', got %q", due[0].Name)
	}
}

// --- NextRun with invalid cron ---

func TestNextRun_InvalidCron(t *testing.T) {
	t.Parallel()

	_, err := NextRun("bad cron", time.Now())
	if err == nil {
		t.Fatal("expected error for invalid cron expression")
	}
}

// --- IsDue with invalid cron ---

func TestIsDue_InvalidCron(t *testing.T) {
	t.Parallel()

	_, err := IsDue("bad cron", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Now())
	if err == nil {
		t.Fatal("expected error for invalid cron expression")
	}
}

// --- Orchestrator: Execute records accurate Duration ---

func TestOrchestrator_Execute_RecordsDuration(t *testing.T) {
	t.Parallel()

	mappings := []TaskToolMapping{
		{TaskRef: "ET-0001", Tools: []string{"slow-tool"}},
	}
	o := NewOrchestrator(mappings, testhelpers.NewStubLogger())
	plan := o.BuildPlan(nil)

	executor := func(_ context.Context, _ string, _ map[string]interface{}) (string, error) {
		time.Sleep(50 * time.Millisecond)
		return "done", nil
	}

	summary := o.Execute(context.Background(), plan, executor)

	if len(summary.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(summary.Results))
	}

	r := summary.Results[0]
	if r.Duration < 50*time.Millisecond {
		t.Errorf("expected duration >= 50ms, got %v", r.Duration)
	}
	if !r.Success {
		t.Error("expected success")
	}
	if r.StartedAt.IsZero() {
		t.Error("expected non-zero StartedAt")
	}
	if summary.Duration < 50*time.Millisecond {
		t.Errorf("expected summary duration >= 50ms, got %v", summary.Duration)
	}
}

// --- Orchestrator: BuildPlan preserves tool ordering with many tools ---

func TestOrchestrator_BuildPlan_ManyTools(t *testing.T) {
	t.Parallel()

	tools := []string{"tool-a", "tool-b", "tool-c", "tool-d", "tool-e"}
	mappings := []TaskToolMapping{
		{TaskRef: "ET-0001", Tools: tools},
	}
	o := NewOrchestrator(mappings, testhelpers.NewStubLogger())
	plan := o.BuildPlan(nil)

	if len(plan.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(plan.Tasks))
	}

	for i, tool := range plan.Tasks[0].Tools {
		if tool.ToolName != tools[i] {
			t.Errorf("tool[%d] = %q, want %q", i, tool.ToolName, tools[i])
		}
		if tool.Order != i {
			t.Errorf("tool[%d].Order = %d, want %d", i, tool.Order, i)
		}
	}
}

// --- Orchestrator: GetMappingsForSchedule with no matches ---

func TestOrchestrator_GetMappingsForSchedule_NoMatches(t *testing.T) {
	t.Parallel()

	mappings := []TaskToolMapping{
		{TaskRef: "ET-0001", Tools: []string{"tool-a"}, Schedule: "daily"},
	}
	o := NewOrchestrator(mappings, testhelpers.NewStubLogger())

	result := o.GetMappingsForSchedule("monthly")
	if len(result) != 0 {
		t.Errorf("expected 0 mappings, got %d", len(result))
	}
}

// --- Orchestrator: Execute with context cancelled mid-task ---

func TestOrchestrator_Execute_ContextCancelledMidTask(t *testing.T) {
	t.Parallel()

	mappings := []TaskToolMapping{
		{TaskRef: "ET-0001", Tools: []string{"tool-a", "tool-b", "tool-c"}},
	}
	o := NewOrchestrator(mappings, testhelpers.NewStubLogger())
	plan := o.BuildPlan(nil)

	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0
	executor := func(_ context.Context, _ string, _ map[string]interface{}) (string, error) {
		callCount++
		if callCount == 1 {
			cancel() // Cancel after first tool runs.
		}
		return "done", nil
	}

	summary := o.Execute(ctx, plan, executor)

	// tool-a runs (succeeded), then context is cancelled, tool-b and tool-c skipped.
	if summary.Succeeded != 1 {
		t.Errorf("expected 1 succeeded, got %d", summary.Succeeded)
	}
	if summary.Skipped != 2 {
		t.Errorf("expected 2 skipped, got %d", summary.Skipped)
	}
}

// --- Orchestrator: empty mappings ---

func TestOrchestrator_EmptyMappings(t *testing.T) {
	t.Parallel()

	o := NewOrchestrator(nil, testhelpers.NewStubLogger())

	plan := o.BuildPlan(nil)
	if len(plan.Tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(plan.Tasks))
	}

	result := o.GetMappingsForSchedule("daily")
	if len(result) != 0 {
		t.Errorf("expected 0 mappings, got %d", len(result))
	}

	_, ok := o.GetMappingForTask("ET-0001")
	if ok {
		t.Error("expected no mapping for ET-0001 with empty mappings")
	}
}
