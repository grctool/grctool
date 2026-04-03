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

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"context"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/scheduler"
	"github.com/grctool/grctool/internal/storage"
	"github.com/grctool/grctool/internal/tools"
	"github.com/spf13/cobra"
)

// scheduleCmd is the parent command for schedule management.
var scheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Manage evidence collection schedules",
	Long: `Manage cron-based evidence collection schedules.

Schedules define when evidence collection tasks run automatically.
Use subcommands to list, check status, and execute scheduled tasks.`,
}

// scheduleListCmd shows all configured schedules.
var scheduleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured schedules",
	Long:  `Display all configured schedules with their cron expression, enabled state, scope, provider, last run, and next due time.`,
	RunE:  runScheduleList,
}

// scheduleStatusCmd shows what's due.
var scheduleStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show schedule status and what's due",
	Long:  `Show which schedules are due now, upcoming, and recently completed.`,
	RunE:  runScheduleStatus,
}

// scheduleRunCmd executes due schedules.
var scheduleRunCmd = &cobra.Command{
	Use:   "run [schedule-name]",
	Short: "Execute due schedules",
	Long: `Execute all due schedules, or a specific named schedule.

By default, only schedules that are currently due will be executed.
Use --force to run a schedule regardless of whether it is due.
Use --dry-run to preview what would be executed without making changes.

Tool execution is performed via the evidence collection orchestrator.
Schedules are marked completed/failed after execution.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runScheduleRun,
}

func init() {
	rootCmd.AddCommand(scheduleCmd)
	scheduleCmd.AddCommand(scheduleListCmd)
	scheduleCmd.AddCommand(scheduleStatusCmd)
	scheduleCmd.AddCommand(scheduleRunCmd)

	// Flags for schedule run
	scheduleRunCmd.Flags().Bool("dry-run", false, "show what would execute without running")
	scheduleRunCmd.Flags().Bool("force", false, "run even if not due")
}

// loadScheduler creates a Scheduler from the current config.
func loadScheduler() (*scheduler.Scheduler, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	// Use data_dir for state storage; fall back to current directory.
	stateDir := "."
	if cfg.Storage.DataDir != "" {
		stateDir = filepath.Join(cfg.Storage.DataDir, ".state")
	}

	log := logger.WithComponent("scheduler")
	return scheduler.NewScheduler(cfg.Schedules.Schedules, stateDir, log), nil
}

func runScheduleList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	schedules := cfg.Schedules.Schedules
	if len(schedules) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No schedules configured.")
		fmt.Fprintln(cmd.OutOrStdout(), "Add schedules to your .grctool.yaml under the 'schedules' section.")
		return nil
	}

	s, err := loadScheduler()
	if err != nil {
		return err
	}

	state, err := s.GetStatus()
	if err != nil {
		return fmt.Errorf("loading schedule state: %w", err)
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tCRON\tENABLED\tSCOPE\tPROVIDER\tLAST RUN\tNEXT DUE")
	fmt.Fprintln(w, "----\t----\t-------\t-----\t--------\t--------\t--------")

	for _, sc := range schedules {
		enabled := "yes"
		if !sc.Enabled {
			enabled = "no"
		}

		scope := sc.Scope
		if scope == "" {
			scope = "all"
		}
		provider := sc.Provider
		if provider == "" {
			provider = "-"
		}

		lastRun := "-"
		nextDue := "-"
		if ss, ok := state.Schedules[sc.Name]; ok {
			if !ss.LastRun.IsZero() {
				lastRun = ss.LastRun.Format(time.RFC3339)
			}
			if !ss.NextDue.IsZero() {
				nextDue = ss.NextDue.Format(time.RFC3339)
			}
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			sc.Name, sc.Cron, enabled, scope, provider, lastRun, nextDue)
	}

	return w.Flush()
}

func runScheduleStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	s, err := loadScheduler()
	if err != nil {
		return err
	}

	now := time.Now()

	due, err := s.GetDueSchedules(now)
	if err != nil {
		return fmt.Errorf("checking due schedules: %w", err)
	}

	state, err := s.GetStatus()
	if err != nil {
		return fmt.Errorf("loading schedule state: %w", err)
	}

	out := cmd.OutOrStdout()

	// Build lookup for due schedules.
	dueSet := make(map[string]bool, len(due))
	for _, d := range due {
		dueSet[d.Name] = true
	}

	// Categorize schedules: overdue, due now, upcoming.
	var overdue, dueNow, upcoming []string
	for _, sc := range cfg.Schedules.Schedules {
		if !sc.Enabled {
			continue
		}
		if dueSet[sc.Name] {
			// If it has run before and the last run had an error or it was due
			// before the last run, it's overdue.
			if ss, ok := state.Schedules[sc.Name]; ok && ss.LastError != "" {
				overdue = append(overdue, sc.Name)
			} else {
				dueNow = append(dueNow, sc.Name)
			}
		} else {
			upcoming = append(upcoming, sc.Name)
		}
	}

	// Overdue
	fmt.Fprintln(out, "=== Overdue ===")
	if len(overdue) == 0 {
		fmt.Fprintln(out, "  No overdue schedules.")
	} else {
		for _, name := range overdue {
			ss := state.Schedules[name]
			fmt.Fprintf(out, "  %s  last_error: %s  last_run: %s  runs: %d\n",
				name, ss.LastError, ss.LastRun.Format(time.RFC3339), ss.RunCount)
		}
	}
	fmt.Fprintln(out)

	// Due Now
	fmt.Fprintln(out, "=== Due Now ===")
	if len(dueNow) == 0 {
		fmt.Fprintln(out, "  No schedules are currently due.")
	} else {
		for _, name := range dueNow {
			for _, d := range due {
				if d.Name == name {
					fmt.Fprintf(out, "  %s  (cron: %s, scope: %s)\n", d.Name, d.Cron, d.Scope)
					break
				}
			}
		}
	}
	fmt.Fprintln(out)

	// Upcoming
	fmt.Fprintln(out, "=== Upcoming ===")
	if len(upcoming) == 0 {
		fmt.Fprintln(out, "  No upcoming schedules.")
	} else {
		for _, name := range upcoming {
			nextDue := "-"
			if ss, ok := state.Schedules[name]; ok && !ss.NextDue.IsZero() {
				nextDue = ss.NextDue.Format(time.RFC3339)
			}
			fmt.Fprintf(out, "  %s  next_due: %s\n", name, nextDue)
		}
	}
	fmt.Fprintln(out)

	// Recently Completed (schedules that have run and have state)
	fmt.Fprintln(out, "=== Recently Completed ===")
	hasCompleted := false
	for name, ss := range state.Schedules {
		if !ss.LastRun.IsZero() {
			hasCompleted = true
			status := "ok"
			if ss.LastError != "" {
				status = fmt.Sprintf("error: %s", ss.LastError)
			}
			fmt.Fprintf(out, "  %s  last_run: %s  runs: %d  status: %s\n",
				name, ss.LastRun.Format(time.RFC3339), ss.RunCount, status)
		}
	}
	if !hasCompleted {
		fmt.Fprintln(out, "  No schedules have run yet.")
	}
	fmt.Fprintln(out)

	// Evidence task due/overdue groupings.
	printTaskDueGroupings(out, cfg, now)

	return nil
}

// printTaskDueGroupings shows evidence tasks grouped by due urgency.
func printTaskDueGroupings(out interface{ Write([]byte) (int, error) }, cfg *config.Config, now time.Time) {
	store, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		return
	}

	tasks, err := store.GetAllEvidenceTasks()
	if err != nil || len(tasks) == 0 {
		return
	}

	var details []scheduler.TaskDueDetail
	for _, t := range tasks {
		ref := t.ReferenceID
		if ref == "" {
			ref = t.ID
		}
		category, daysUntil := scheduler.ClassifyTaskDue(t.NextDue, now)
		if category == scheduler.TaskNoSchedule {
			continue
		}
		details = append(details, scheduler.TaskDueDetail{
			TaskRef:   ref,
			TaskName:  t.Name,
			Category:  category,
			DueDate:   t.NextDue,
			DaysUntil: daysUntil,
		})
	}

	if len(details) == 0 {
		return
	}

	grouping := scheduler.GroupTasksByDue(details)

	fmt.Fprintln(out, "=== Evidence Tasks: Overdue ===")
	if len(grouping.Overdue) == 0 {
		fmt.Fprintln(out, "  No overdue tasks.")
	} else {
		for _, d := range grouping.Overdue {
			fmt.Fprintf(out, "  %s  %s  (overdue %d days, due: %s)\n",
				d.TaskRef, d.TaskName, -d.DaysUntil, d.DueDate.Format("2006-01-02"))
		}
	}
	fmt.Fprintln(out)

	fmt.Fprintln(out, "=== Evidence Tasks: Due This Week ===")
	if len(grouping.DueThisWeek) == 0 {
		fmt.Fprintln(out, "  No tasks due this week.")
	} else {
		for _, d := range grouping.DueThisWeek {
			fmt.Fprintf(out, "  %s  %s  (due in %d days, due: %s)\n",
				d.TaskRef, d.TaskName, d.DaysUntil, d.DueDate.Format("2006-01-02"))
		}
	}
	fmt.Fprintln(out)

	fmt.Fprintln(out, "=== Evidence Tasks: Due This Month ===")
	if len(grouping.DueThisMonth) == 0 {
		fmt.Fprintln(out, "  No tasks due this month.")
	} else {
		for _, d := range grouping.DueThisMonth {
			fmt.Fprintf(out, "  %s  %s  (due in %d days, due: %s)\n",
				d.TaskRef, d.TaskName, d.DaysUntil, d.DueDate.Format("2006-01-02"))
		}
	}
	fmt.Fprintln(out)

	fmt.Fprintln(out, "=== Evidence Tasks: Upcoming ===")
	if len(grouping.Upcoming) == 0 {
		fmt.Fprintln(out, "  No upcoming tasks.")
	} else {
		for _, d := range grouping.Upcoming {
			fmt.Fprintf(out, "  %s  %s  (due in %d days, due: %s)\n",
				d.TaskRef, d.TaskName, d.DaysUntil, d.DueDate.Format("2006-01-02"))
		}
	}
}

func runScheduleRun(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	s, err := loadScheduler()
	if err != nil {
		return err
	}

	now := time.Now()
	out := cmd.OutOrStdout()

	// If a specific schedule name is given, filter to just that one.
	if len(args) == 1 {
		name := args[0]
		return runNamedSchedule(cmd, s, name, now, dryRun, force)
	}

	// Run all due schedules.
	due, err := s.GetDueSchedules(now)
	if err != nil {
		return fmt.Errorf("checking due schedules: %w", err)
	}

	if len(due) == 0 && !force {
		fmt.Fprintln(out, "No schedules are currently due.")
		return nil
	}

	orch := loadOrchestrator(s)

	for _, d := range due {
		if dryRun {
			fmt.Fprintf(out, "[dry-run] Would execute schedule: %s (scope: %s, provider: %s)\n",
				d.Name, d.Scope, d.Provider)
			continue
		}

		fmt.Fprintf(out, "Executing schedule: %s (scope: %s, provider: %s)\n",
			d.Name, d.Scope, d.Provider)

		summary := executeSchedule(cmd.Context(), orch, d)
		printCollectionSummary(out, summary)

		if summary.Failed > 0 {
			if err := s.MarkFailed(d.Name, now, fmt.Sprintf("%d tools failed", summary.Failed)); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save state for %s: %v\n", d.Name, err)
			}
		} else {
			if err := s.MarkCompleted(d.Name, now); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save state for %s: %v\n", d.Name, err)
			}
		}
	}

	return nil
}

// runNamedSchedule runs a specific schedule by name.
func runNamedSchedule(cmd *cobra.Command, s *scheduler.Scheduler, name string, now time.Time, dryRun, force bool) error {
	out := cmd.OutOrStdout()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Find the named schedule.
	var found *config.ScheduleConfig
	for i := range cfg.Schedules.Schedules {
		if cfg.Schedules.Schedules[i].Name == name {
			found = &cfg.Schedules.Schedules[i]
			break
		}
	}

	if found == nil {
		return fmt.Errorf("schedule %q not found in configuration", name)
	}

	if !force {
		// Check if it's due.
		due, err := s.GetDueSchedules(now)
		if err != nil {
			return fmt.Errorf("checking due schedules: %w", err)
		}

		isDue := false
		for _, d := range due {
			if d.Name == name {
				isDue = true
				break
			}
		}

		if !isDue {
			fmt.Fprintf(out, "Schedule %q is not currently due. Use --force to run anyway.\n", name)
			return nil
		}
	}

	if dryRun {
		fmt.Fprintf(out, "[dry-run] Would execute schedule: %s (scope: %s, provider: %s)\n",
			found.Name, found.Scope, found.Provider)
		return nil
	}

	fmt.Fprintf(out, "Executing schedule: %s (scope: %s, provider: %s)\n",
		found.Name, found.Scope, found.Provider)

	orch := loadOrchestrator(s)
	summary := executeSchedule(cmd.Context(), orch, scheduler.Schedule{
		Name: found.Name, Cron: found.Cron, Enabled: found.Enabled,
		Scope: found.Scope, Provider: found.Provider,
	})
	printCollectionSummary(out, summary)

	if summary.Failed > 0 {
		if err := s.MarkFailed(name, now, fmt.Sprintf("%d tools failed", summary.Failed)); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save state for %s: %v\n", name, err)
		}
	} else {
		if err := s.MarkCompleted(name, now); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save state for %s: %v\n", name, err)
		}
	}

	return nil
}

// loadOrchestrator creates an Orchestrator from schedule-task-tool mappings in config.
func loadOrchestrator(_ *scheduler.Scheduler) *scheduler.Orchestrator {
	log := logger.WithComponent("orchestrator")

	cfg, err := config.Load()
	if err != nil {
		log.Warn("failed to load config for task mappings; orchestrator will have no mappings",
			logger.String("error", err.Error()))
		return scheduler.NewOrchestrator(nil, log)
	}

	var mappings []scheduler.TaskToolMapping
	for _, tm := range cfg.Schedules.TaskMappings {
		mappings = append(mappings, scheduler.TaskToolMapping{
			TaskRef:  tm.TaskRef,
			Tools:    tm.Tools,
			Schedule: tm.Schedule,
		})
	}

	return scheduler.NewOrchestrator(mappings, log)
}

// toolExecutor wraps the global tool registry ExecuteTool as the executor
// function expected by Orchestrator.Execute.
func toolExecutor(ctx context.Context, toolName string, params map[string]interface{}) (string, error) {
	result, _, err := tools.ExecuteTool(ctx, toolName, params)
	return result, err
}

// executeSchedule runs a single schedule through the orchestrator.
func executeSchedule(ctx context.Context, orch *scheduler.Orchestrator, sched scheduler.Schedule) *scheduler.CollectionSummary {
	mappings := orch.GetMappingsForSchedule(sched.Name)
	var taskRefs []string
	for _, m := range mappings {
		taskRefs = append(taskRefs, m.TaskRef)
	}
	plan := orch.BuildPlan(taskRefs)
	return orch.Execute(ctx, plan, toolExecutor)
}

// printCollectionSummary outputs the per-task results and overall summary of a collection run.
func printCollectionSummary(out interface{ Write([]byte) (int, error) }, summary *scheduler.CollectionSummary) {
	// Print per-task results when there are results to show.
	if len(summary.Results) > 0 {
		currentTask := ""
		for _, r := range summary.Results {
			if r.TaskRef != currentTask {
				currentTask = r.TaskRef
				fmt.Fprintf(out, "  Task %s:\n", currentTask)
			}
			status := "ok"
			if !r.Success {
				status = "FAILED"
				if r.Error != "" {
					status = fmt.Sprintf("FAILED: %s", r.Error)
				}
			}
			fmt.Fprintf(out, "    - %s: %s (%s)\n", r.ToolName, status, r.Duration.Round(time.Millisecond))
		}
	}
	fmt.Fprintf(out, "  Tasks: %d | Tools: %d | Succeeded: %d | Failed: %d | Skipped: %d | Duration: %s\n",
		summary.TotalTasks, summary.TotalTools, summary.Succeeded, summary.Failed, summary.Skipped,
		summary.Duration.Round(time.Millisecond))
}
