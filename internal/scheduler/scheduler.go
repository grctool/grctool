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

// Package scheduler evaluates cron-based schedules and determines which
// scheduled tasks are due for execution. It is designed for CLI-based
// invocation (not a daemon) — an external cron or CI pipeline calls
// `grctool schedule run` and the scheduler evaluates what is due NOW.
package scheduler

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"gopkg.in/yaml.v3"
)

// Schedule represents a configured schedule.
type Schedule struct {
	Name     string `json:"name" yaml:"name"`
	Cron     string `json:"cron" yaml:"cron"`
	Enabled  bool   `json:"enabled" yaml:"enabled"`
	Scope    string `json:"scope" yaml:"scope"`       // "all", "policies", "controls", "evidence"
	Provider string `json:"provider" yaml:"provider"` // provider name or "" for all
}

// ScheduleState tracks execution state for a schedule.
type ScheduleState struct {
	Name      string    `json:"name" yaml:"name"`
	LastRun   time.Time `json:"last_run" yaml:"last_run"`
	NextDue   time.Time `json:"next_due" yaml:"next_due"`
	LastError string    `json:"last_error,omitempty" yaml:"last_error,omitempty"`
	RunCount  int       `json:"run_count" yaml:"run_count"`
}

// SchedulerState holds state for all schedules.
type SchedulerState struct {
	Schedules map[string]*ScheduleState `json:"schedules" yaml:"schedules"`
	UpdatedAt time.Time                 `json:"updated_at" yaml:"updated_at"`
}

// Scheduler evaluates schedules and determines what's due.
type Scheduler struct {
	schedules []Schedule
	stateDir  string
	logger    logger.Logger
}

// stateFileName is the name of the state persistence file.
const stateFileName = "schedule_state.yaml"

// NewScheduler creates a scheduler from config.
func NewScheduler(schedules []config.ScheduleConfig, stateDir string, log logger.Logger) *Scheduler {
	converted := make([]Schedule, len(schedules))
	for i, sc := range schedules {
		converted[i] = Schedule{
			Name:     sc.Name,
			Cron:     sc.Cron,
			Enabled:  sc.Enabled,
			Scope:    sc.Scope,
			Provider: sc.Provider,
		}
	}
	return &Scheduler{
		schedules: converted,
		stateDir:  stateDir,
		logger:    log,
	}
}

// GetDueSchedules returns schedules that should run now.
func (s *Scheduler) GetDueSchedules(now time.Time) ([]Schedule, error) {
	state, err := s.LoadState()
	if err != nil {
		return nil, fmt.Errorf("loading state: %w", err)
	}

	var due []Schedule
	for _, sched := range s.schedules {
		if !sched.Enabled {
			continue
		}

		lastRun := time.Time{} // zero value means never run
		if ss, ok := state.Schedules[sched.Name]; ok {
			lastRun = ss.LastRun
		}

		isDue, err := IsDue(sched.Cron, lastRun, now)
		if err != nil {
			s.logger.Error("failed to evaluate schedule",
				logger.String("schedule", sched.Name),
				logger.String("error", err.Error()),
			)
			continue
		}

		if isDue {
			due = append(due, sched)
		}
	}

	return due, nil
}

// MarkCompleted records a successful schedule execution.
func (s *Scheduler) MarkCompleted(name string, now time.Time) error {
	state, err := s.LoadState()
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	ss, ok := state.Schedules[name]
	if !ok {
		ss = &ScheduleState{Name: name}
		state.Schedules[name] = ss
	}

	ss.LastRun = now
	ss.RunCount++
	ss.LastError = ""

	// Compute next due time from the schedule's cron expression.
	for _, sched := range s.schedules {
		if sched.Name == name {
			next, err := NextRun(sched.Cron, now)
			if err == nil {
				ss.NextDue = next
			}
			break
		}
	}

	state.UpdatedAt = now
	return s.SaveState(state)
}

// MarkFailed records a failed schedule execution.
func (s *Scheduler) MarkFailed(name string, now time.Time, errMsg string) error {
	state, err := s.LoadState()
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	ss, ok := state.Schedules[name]
	if !ok {
		ss = &ScheduleState{Name: name}
		state.Schedules[name] = ss
	}

	ss.LastRun = now
	ss.RunCount++
	ss.LastError = errMsg

	// Compute next due time from the schedule's cron expression.
	for _, sched := range s.schedules {
		if sched.Name == name {
			next, err := NextRun(sched.Cron, now)
			if err == nil {
				ss.NextDue = next
			}
			break
		}
	}

	state.UpdatedAt = now
	return s.SaveState(state)
}

// GetStatus returns the state of all schedules.
func (s *Scheduler) GetStatus() (*SchedulerState, error) {
	return s.LoadState()
}

// statePath returns the full path to the state file.
func (s *Scheduler) statePath() string {
	return filepath.Join(s.stateDir, stateFileName)
}

// LoadState reads persisted state from disk.
// If the state file does not exist, an empty state is returned.
func (s *Scheduler) LoadState() (*SchedulerState, error) {
	state := &SchedulerState{
		Schedules: make(map[string]*ScheduleState),
	}

	data, err := os.ReadFile(s.statePath())
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return nil, fmt.Errorf("reading state file: %w", err)
	}

	if err := yaml.Unmarshal(data, state); err != nil {
		return nil, fmt.Errorf("parsing state file: %w", err)
	}

	if state.Schedules == nil {
		state.Schedules = make(map[string]*ScheduleState)
	}

	return state, nil
}

// SaveState writes state to disk. It creates the state directory if missing
// and uses atomic write (write to temp, then rename) to prevent corruption.
func (s *Scheduler) SaveState(state *SchedulerState) error {
	if err := os.MkdirAll(s.stateDir, 0o755); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}

	data, err := yaml.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	// Atomic write: write to temp file then rename.
	tmp := s.statePath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("writing temp state file: %w", err)
	}

	if err := os.Rename(tmp, s.statePath()); err != nil {
		// Clean up temp file on rename failure.
		_ = os.Remove(tmp)
		return fmt.Errorf("renaming state file: %w", err)
	}

	return nil
}

// NextRun calculates the next run time for a cron expression after the given time.
func NextRun(cronExpr string, after time.Time) (time.Time, error) {
	expr, err := parseCron(cronExpr)
	if err != nil {
		return time.Time{}, err
	}
	return expr.Next(after), nil
}

// IsDue checks if a schedule is due based on cron expression and last run.
// A schedule is due if the next occurrence after lastRun is at or before now.
// If lastRun is zero (never run), the schedule is always due.
func IsDue(cronExpr string, lastRun time.Time, now time.Time) (bool, error) {
	if lastRun.IsZero() {
		return true, nil
	}

	next, err := NextRun(cronExpr, lastRun)
	if err != nil {
		return false, err
	}

	return !next.After(now), nil
}
