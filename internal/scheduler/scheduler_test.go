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

package scheduler

import (
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/testhelpers"
)

// helper to create a scheduler with common test schedules.
func newTestScheduler(t *testing.T, schedules []config.ScheduleConfig) *Scheduler {
	t.Helper()
	return NewScheduler(schedules, t.TempDir(), testhelpers.NewStubLogger())
}

func TestNewScheduler(t *testing.T) {
	cfgs := []config.ScheduleConfig{
		{Name: "hourly", Cron: "0 * * * *", Enabled: true, Scope: "all"},
		{Name: "daily", Cron: "0 9 * * *", Enabled: true, Scope: "evidence", Provider: "tugboat"},
	}
	s := newTestScheduler(t, cfgs)

	if len(s.schedules) != 2 {
		t.Fatalf("expected 2 schedules, got %d", len(s.schedules))
	}
	if s.schedules[0].Name != "hourly" {
		t.Errorf("expected schedule name 'hourly', got %q", s.schedules[0].Name)
	}
	if s.schedules[1].Provider != "tugboat" {
		t.Errorf("expected provider 'tugboat', got %q", s.schedules[1].Provider)
	}
}

func TestGetDueSchedules_NeverRun(t *testing.T) {
	cfgs := []config.ScheduleConfig{
		{Name: "hourly", Cron: "0 * * * *", Enabled: true, Scope: "all"},
	}
	s := newTestScheduler(t, cfgs)
	now := time.Date(2026, 3, 18, 10, 30, 0, 0, time.UTC)

	due, err := s.GetDueSchedules(now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(due) != 1 {
		t.Fatalf("expected 1 due schedule, got %d", len(due))
	}
	if due[0].Name != "hourly" {
		t.Errorf("expected 'hourly', got %q", due[0].Name)
	}
}

func TestGetDueSchedules_RecentlyRun(t *testing.T) {
	cfgs := []config.ScheduleConfig{
		{Name: "hourly", Cron: "0 * * * *", Enabled: true, Scope: "all"},
	}
	s := newTestScheduler(t, cfgs)
	now := time.Date(2026, 3, 18, 10, 30, 0, 0, time.UTC)

	// Mark as completed 1 minute ago — next due is 11:00, so not due at 10:30.
	err := s.MarkCompleted("hourly", now.Add(-1*time.Minute))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	due, err := s.GetDueSchedules(now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("expected 0 due schedules, got %d", len(due))
	}
}

func TestGetDueSchedules_Overdue(t *testing.T) {
	cfgs := []config.ScheduleConfig{
		{Name: "hourly", Cron: "0 * * * *", Enabled: true, Scope: "all"},
	}
	s := newTestScheduler(t, cfgs)
	now := time.Date(2026, 3, 18, 12, 30, 0, 0, time.UTC)

	// Last ran 2 hours ago at 10:30 — next due was 11:00, which is before 12:30.
	err := s.MarkCompleted("hourly", time.Date(2026, 3, 18, 10, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	due, err := s.GetDueSchedules(now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(due) != 1 {
		t.Fatalf("expected 1 due schedule, got %d", len(due))
	}
}

func TestGetDueSchedules_Disabled(t *testing.T) {
	cfgs := []config.ScheduleConfig{
		{Name: "hourly", Cron: "0 * * * *", Enabled: false, Scope: "all"},
	}
	s := newTestScheduler(t, cfgs)
	now := time.Date(2026, 3, 18, 10, 30, 0, 0, time.UTC)

	due, err := s.GetDueSchedules(now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("expected 0 due schedules (disabled), got %d", len(due))
	}
}

func TestGetDueSchedules_MultipleDue(t *testing.T) {
	cfgs := []config.ScheduleConfig{
		{Name: "hourly", Cron: "0 * * * *", Enabled: true, Scope: "all"},
		{Name: "daily", Cron: "0 9 * * *", Enabled: true, Scope: "evidence"},
		{Name: "weekly", Cron: "0 9 * * 1", Enabled: true, Scope: "policies"},
	}
	s := newTestScheduler(t, cfgs)
	// Wednesday 2026-03-18 10:30 UTC
	now := time.Date(2026, 3, 18, 10, 30, 0, 0, time.UTC)

	// hourly last ran at 09:00 — next due 10:00 <= 10:30 → DUE
	_ = s.MarkCompleted("hourly", time.Date(2026, 3, 18, 9, 0, 0, 0, time.UTC))
	// daily last ran at 09:00 today — next due 09:00 tomorrow → NOT DUE
	_ = s.MarkCompleted("daily", time.Date(2026, 3, 18, 9, 0, 0, 0, time.UTC))
	// weekly never ran → DUE
	// (no MarkCompleted for weekly)

	due, err := s.GetDueSchedules(now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	names := make(map[string]bool)
	for _, d := range due {
		names[d.Name] = true
	}

	if !names["hourly"] {
		t.Error("expected 'hourly' to be due")
	}
	if names["daily"] {
		t.Error("expected 'daily' to NOT be due")
	}
	if !names["weekly"] {
		t.Error("expected 'weekly' to be due (never run)")
	}
}

func TestMarkCompleted(t *testing.T) {
	cfgs := []config.ScheduleConfig{
		{Name: "hourly", Cron: "0 * * * *", Enabled: true, Scope: "all"},
	}
	s := newTestScheduler(t, cfgs)
	now := time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC)

	err := s.MarkCompleted("hourly", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	state, err := s.LoadState()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ss := state.Schedules["hourly"]
	if ss == nil {
		t.Fatal("expected schedule state for 'hourly'")
	}
	if ss.RunCount != 1 {
		t.Errorf("expected run count 1, got %d", ss.RunCount)
	}
	if !ss.LastRun.Equal(now) {
		t.Errorf("expected last run %v, got %v", now, ss.LastRun)
	}
	if ss.LastError != "" {
		t.Errorf("expected no error, got %q", ss.LastError)
	}
	if ss.NextDue.IsZero() {
		t.Error("expected next_due to be set")
	}

	// Mark again to check increment.
	err = s.MarkCompleted("hourly", now.Add(time.Hour))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	state, _ = s.LoadState()
	if state.Schedules["hourly"].RunCount != 2 {
		t.Errorf("expected run count 2 after second completion, got %d", state.Schedules["hourly"].RunCount)
	}
}

func TestMarkFailed(t *testing.T) {
	cfgs := []config.ScheduleConfig{
		{Name: "hourly", Cron: "0 * * * *", Enabled: true, Scope: "all"},
	}
	s := newTestScheduler(t, cfgs)
	now := time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC)

	err := s.MarkFailed("hourly", now, "connection timeout")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	state, err := s.LoadState()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ss := state.Schedules["hourly"]
	if ss == nil {
		t.Fatal("expected schedule state for 'hourly'")
	}
	if ss.RunCount != 1 {
		t.Errorf("expected run count 1, got %d", ss.RunCount)
	}
	if ss.LastError != "connection timeout" {
		t.Errorf("expected error 'connection timeout', got %q", ss.LastError)
	}
}

func TestGetStatus(t *testing.T) {
	cfgs := []config.ScheduleConfig{
		{Name: "hourly", Cron: "0 * * * *", Enabled: true, Scope: "all"},
		{Name: "daily", Cron: "0 9 * * *", Enabled: true, Scope: "evidence"},
	}
	s := newTestScheduler(t, cfgs)
	now := time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC)

	_ = s.MarkCompleted("hourly", now)
	_ = s.MarkFailed("daily", now, "auth expired")

	status, err := s.GetStatus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(status.Schedules) != 2 {
		t.Fatalf("expected 2 schedule states, got %d", len(status.Schedules))
	}
	if status.Schedules["hourly"].LastError != "" {
		t.Error("expected no error on hourly")
	}
	if status.Schedules["daily"].LastError != "auth expired" {
		t.Errorf("expected 'auth expired' error on daily, got %q", status.Schedules["daily"].LastError)
	}
}

func TestStatePersistence(t *testing.T) {
	stateDir := t.TempDir()
	log := testhelpers.NewStubLogger()
	cfgs := []config.ScheduleConfig{
		{Name: "hourly", Cron: "0 * * * *", Enabled: true, Scope: "all"},
	}

	// Create first scheduler, mark completed.
	s1 := NewScheduler(cfgs, stateDir, log)
	now := time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC)
	err := s1.MarkCompleted("hourly", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Create second scheduler with same state dir — should load persisted state.
	s2 := NewScheduler(cfgs, stateDir, log)
	state, err := s2.LoadState()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ss := state.Schedules["hourly"]
	if ss == nil {
		t.Fatal("expected persisted schedule state for 'hourly'")
	}
	if ss.RunCount != 1 {
		t.Errorf("expected persisted run count 1, got %d", ss.RunCount)
	}
	if !ss.LastRun.Equal(now) {
		t.Errorf("expected persisted last run %v, got %v", now, ss.LastRun)
	}
}

func TestNextRun_Hourly(t *testing.T) {
	// "0 * * * *" — minute 0 of every hour
	after := time.Date(2026, 3, 18, 10, 15, 0, 0, time.UTC)
	next, err := NextRun("0 * * * *", after)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := time.Date(2026, 3, 18, 11, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}

func TestNextRun_Daily(t *testing.T) {
	// "0 9 * * *" — 09:00 daily
	after := time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC) // after 9 AM
	next, err := NextRun("0 9 * * *", after)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := time.Date(2026, 3, 19, 9, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}

	// Before 9 AM — should be same day.
	after2 := time.Date(2026, 3, 18, 8, 0, 0, 0, time.UTC)
	next2, err := NextRun("0 9 * * *", after2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected2 := time.Date(2026, 3, 18, 9, 0, 0, 0, time.UTC)
	if !next2.Equal(expected2) {
		t.Errorf("expected %v, got %v", expected2, next2)
	}
}

func TestIsDue_Simple(t *testing.T) {
	tests := []struct {
		name    string
		cron    string
		lastRun time.Time
		now     time.Time
		want    bool
	}{
		{
			name:    "never run is always due",
			cron:    "0 * * * *",
			lastRun: time.Time{},
			now:     time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC),
			want:    true,
		},
		{
			name:    "just ran, not due yet",
			cron:    "0 * * * *",
			lastRun: time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC),
			now:     time.Date(2026, 3, 18, 10, 30, 0, 0, time.UTC),
			want:    false,
		},
		{
			name:    "overdue by one interval",
			cron:    "0 * * * *",
			lastRun: time.Date(2026, 3, 18, 9, 0, 0, 0, time.UTC),
			now:     time.Date(2026, 3, 18, 10, 30, 0, 0, time.UTC),
			want:    true,
		},
		{
			name:    "exactly at next run time",
			cron:    "0 * * * *",
			lastRun: time.Date(2026, 3, 18, 9, 0, 0, 0, time.UTC),
			now:     time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC),
			want:    true,
		},
		{
			name:    "daily schedule not due same day",
			cron:    "0 9 * * *",
			lastRun: time.Date(2026, 3, 18, 9, 0, 0, 0, time.UTC),
			now:     time.Date(2026, 3, 18, 15, 0, 0, 0, time.UTC),
			want:    false,
		},
		{
			name:    "daily schedule due next day",
			cron:    "0 9 * * *",
			lastRun: time.Date(2026, 3, 18, 9, 0, 0, 0, time.UTC),
			now:     time.Date(2026, 3, 19, 10, 0, 0, 0, time.UTC),
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IsDue(tt.cron, tt.lastRun, tt.now)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("IsDue(%q, %v, %v) = %v, want %v", tt.cron, tt.lastRun, tt.now, got, tt.want)
			}
		})
	}
}

func TestGetDueSchedules_EmptySchedules(t *testing.T) {
	s := newTestScheduler(t, nil)
	now := time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC)

	due, err := s.GetDueSchedules(now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("expected 0 due schedules for empty config, got %d", len(due))
	}
}
