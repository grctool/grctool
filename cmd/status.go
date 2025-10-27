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
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/services"
	"github.com/grctool/grctool/internal/storage"
	"github.com/spf13/cobra"
)

// evidenceStatusCmd represents the main status command
var evidenceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show evidence generation status",
	Long: `Display status of evidence generation across all tasks or for specific tasks.

The status command provides visibility into:
- Evidence generation state (no evidence, generated, validated, submitted, accepted)
- Automation capability (fully automated, partially automated, manual only)
- Evidence files organized by collection windows (2025-Q4, etc.)
- Generation and submission metadata

Examples:
  # Show overall status dashboard
  grctool status

  # Filter by state
  grctool status --filter generated

  # Filter by automation level
  grctool status --automation fully_automated

  # Show detailed status for a specific task
  grctool status task ET-0001

  # Force rescan of evidence directories
  grctool status scan`,
	RunE: runStatusDashboard,
}

// statusTaskCmd represents the status task subcommand
var statusTaskCmd = &cobra.Command{
	Use:   "task [task-ref]",
	Short: "Show detailed status for a specific evidence task",
	Long: `Display detailed status information for a specific evidence task including:
- Task metadata (framework, automation level, applicable tools)
- Status by collection window
- Generation metadata (when, how, by whom)
- Submission metadata (status, ID, timestamps)
- File inventory with sizes and checksums

Examples:
  grctool status task ET-0001
  grctool status task ET-0047`,
	Args: cobra.ExactArgs(1),
	RunE: runStatusTask,
}

// statusScanCmd represents the status scan subcommand
var statusScanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Force rescan of evidence directories",
	Long: `Force a fresh scan of all evidence directories to rebuild the state cache.

This is useful when:
- Evidence files have been added or modified manually
- You want to verify the current state of evidence
- The cached state seems out of sync

Example:
  grctool status scan`,
	RunE: runStatusScan,
}

func init() {
	rootCmd.AddCommand(evidenceStatusCmd)
	evidenceStatusCmd.AddCommand(statusTaskCmd)
	evidenceStatusCmd.AddCommand(statusScanCmd)

	// Flags for main status command
	evidenceStatusCmd.Flags().String("filter", "", "Filter by state (no_evidence, generated, validated, submitted, accepted)")
	evidenceStatusCmd.Flags().String("automation", "", "Filter by automation level (fully_automated, partially_automated, manual_only)")
	evidenceStatusCmd.Flags().Bool("verbose", false, "Show detailed information")
}

// runStatusDashboard displays the overall status dashboard
func runStatusDashboard(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get filter flags
	filterState, _ := cmd.Flags().GetString("filter")
	filterAutomation, _ := cmd.Flags().GetString("automation")
	verbose, _ := cmd.Flags().GetBool("verbose")

	// Initialize scanner
	scanner, cfg, err := initializeScanner()
	if err != nil {
		return err
	}

	// Check if evidence directory exists
	evidenceDir := filepath.Join(cfg.Storage.DataDir, "evidence")
	if _, err := filepath.Abs(evidenceDir); err != nil {
		return fmt.Errorf("invalid evidence directory path: %w", err)
	}

	// Perform scan
	cmd.Println("Scanning evidence directories...")
	taskStates, err := scanner.ScanAll(ctx)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	// Build state cache for summary queries
	cache := models.NewStateCache()
	for taskRef, state := range taskStates {
		cache.SetTask(taskRef, state)
	}

	// Apply filters
	var filteredTasks []*models.EvidenceTaskState
	for _, task := range taskStates {
		// State filter
		if filterState != "" && task.LocalState != models.LocalEvidenceState(filterState) {
			continue
		}
		// Automation filter
		if filterAutomation != "" && task.AutomationLevel != models.AutomationCapability(filterAutomation) {
			continue
		}
		filteredTasks = append(filteredTasks, task)
	}

	// Display header
	cmd.Println()
	cmd.Println("Evidence Status Dashboard")
	cmd.Println("========================")
	cmd.Printf("Scanned: %s (%d tasks)\n", evidenceDir, len(taskStates))
	cmd.Printf("Last Scan: %s\n", cache.LastScan.Format("2006-01-02 15:04:05"))

	if filterState != "" || filterAutomation != "" {
		cmd.Printf("Filtered: %d tasks\n", len(filteredTasks))
	}
	cmd.Println()

	// Display state summary
	cmd.Println("By Local State:")
	stateSummary := cache.GetStateSummary()
	totalTasks := len(taskStates)

	// Display in priority order
	stateOrder := []models.LocalEvidenceState{
		models.StateNoEvidence,
		models.StateGenerated,
		models.StateValidated,
		models.StateSubmitted,
		models.StateAccepted,
		models.StateRejected,
	}

	for _, state := range stateOrder {
		count := stateSummary[state]
		if count > 0 || state == models.StateNoEvidence {
			percentage := float64(count) / float64(totalTasks) * 100
			stateLabel := formatStateLabel(state)
			cmd.Printf("  %-20s %3d tasks (%5.1f%%)\n", stateLabel, count, percentage)
		}
	}
	cmd.Println()

	// Display automation summary
	cmd.Println("By Automation:")
	autoSummary := cache.GetAutomationSummary()

	autoOrder := []models.AutomationCapability{
		models.AutomationFully,
		models.AutomationPartially,
		models.AutomationManual,
		models.AutomationUnknown,
	}

	for _, level := range autoOrder {
		count := autoSummary[level]
		if count > 0 {
			percentage := float64(count) / float64(totalTasks) * 100
			autoLabel := formatAutomationLabel(level)
			cmd.Printf("  %-25s %3d tasks (%5.1f%%)\n", autoLabel, count, percentage)
		}
	}
	cmd.Println()

	// Display recent activity
	recentTasks := getRecentActivity(filteredTasks, 10)
	if len(recentTasks) > 0 {
		cmd.Println("Recent Activity (last 10):")
		for _, task := range recentTasks {
			displayTaskSummary(cmd, task, verbose)
		}
		cmd.Println()
	}

	// Display helpful next steps if there's work to do
	if stateSummary[models.StateGenerated] > 0 {
		cmd.Printf("Tip: %d tasks have generated evidence ready for validation\n", stateSummary[models.StateGenerated])
	}
	if stateSummary[models.StateValidated] > 0 {
		cmd.Printf("Tip: %d tasks have validated evidence ready for submission\n", stateSummary[models.StateValidated])
	}

	return nil
}

// runStatusTask displays detailed status for a specific task
func runStatusTask(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	taskRef := strings.ToUpper(args[0])

	// Normalize task reference (e.g., ET-1 -> ET-0001)
	taskRef = normalizeTaskRef(taskRef)

	// Initialize scanner
	scanner, _, err := initializeScanner()
	if err != nil {
		return err
	}

	// Scan the specific task
	cmd.Printf("Scanning task %s...\n", taskRef)
	taskState, err := scanner.ScanTask(ctx, taskRef)
	if err != nil {
		return fmt.Errorf("failed to scan task: %w", err)
	}

	// Display task header
	cmd.Println()
	cmd.Printf("%s: %s\n", taskRef, taskState.TaskName)
	cmd.Println(strings.Repeat("=", len(taskRef)+2+len(taskState.TaskName)))
	cmd.Printf("Status: %s\n", formatStateLabel(taskState.LocalState))
	if taskState.Framework != "" {
		cmd.Printf("Framework: %s\n", taskState.Framework)
	}
	cmd.Printf("Automation: %s\n", formatAutomationLabel(taskState.AutomationLevel))
	if len(taskState.ApplicableTools) > 0 {
		cmd.Printf("Applicable Tools: %s\n", strings.Join(taskState.ApplicableTools, ", "))
	}
	cmd.Println()

	// Check if task has any evidence
	if len(taskState.Windows) == 0 {
		cmd.Println("No evidence found for this task.")
		cmd.Println()
		cmd.Println("Next Steps:")
		cmd.Printf("  - Generate evidence: grctool evidence generate %s --window 2025-Q4\n", taskRef)
		if len(taskState.ApplicableTools) > 0 {
			cmd.Printf("  - Use tools: %s\n", strings.Join(taskState.ApplicableTools, ", "))
		}
		return nil
	}

	// Display windows (sorted by date, newest first)
	cmd.Println("Windows:")
	windows := sortWindowsByDate(taskState.Windows)
	for _, window := range windows {
		displayWindowDetail(cmd, window)
	}

	// Display suggested next steps
	cmd.Println("Next Steps:")
	displayNextSteps(cmd, taskRef, taskState)

	return nil
}

// runStatusScan performs a force rescan of evidence directories
func runStatusScan(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	cmd.Println("Scanning evidence directories...")

	// Initialize scanner
	scanner, cfg, err := initializeScanner()
	if err != nil {
		return err
	}

	evidenceDir := filepath.Join(cfg.Storage.DataDir, "evidence")

	// Perform scan
	taskStates, err := scanner.ScanAll(ctx)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	cmd.Printf("\nScan complete: found %d tasks in %s\n", len(taskStates), evidenceDir)

	// Display quick summary
	cache := models.NewStateCache()
	for taskRef, state := range taskStates {
		cache.SetTask(taskRef, state)
	}

	stateSummary := cache.GetStateSummary()
	if stateSummary[models.StateNoEvidence] < len(taskStates) {
		cmd.Printf("Evidence found in %d tasks\n", len(taskStates)-stateSummary[models.StateNoEvidence])
	}

	return nil
}

// Helper functions

// storageAdapter adapts storage.Storage to services.Storage interface
type storageAdapter struct {
	storage *storage.Storage
}

func (sa *storageAdapter) GetEvidenceTask(ctx context.Context, taskID int) (*domain.EvidenceTask, error) {
	// Convert int ID to string for storage lookup
	idStr := fmt.Sprintf("%d", taskID)
	return sa.storage.GetEvidenceTask(idStr)
}

// initializeScanner creates a new evidence scanner instance
func initializeScanner() (services.EvidenceScanner, *config.Config, error) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize storage
	store, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Wrap storage with adapter
	storageAdapter := &storageAdapter{storage: store}

	// Get console logger from config
	consoleLoggerCfg := cfg.Logging.Loggers["console"]
	log, err := logger.New((&consoleLoggerCfg).ToLoggerConfig())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Create scanner
	evidenceDir := filepath.Join(cfg.Storage.DataDir, "evidence")
	scanner := services.NewEvidenceScanner(evidenceDir, storageAdapter, log)

	return scanner, cfg, nil
}

// formatStateLabel formats a state for display
func formatStateLabel(state models.LocalEvidenceState) string {
	switch state {
	case models.StateNoEvidence:
		return "No Evidence"
	case models.StateGenerated:
		return "Generated"
	case models.StateValidated:
		return "Validated"
	case models.StateSubmitted:
		return "Submitted"
	case models.StateAccepted:
		return "Accepted"
	case models.StateRejected:
		return "Rejected"
	default:
		return string(state)
	}
}

// formatAutomationLabel formats an automation level for display
func formatAutomationLabel(level models.AutomationCapability) string {
	switch level {
	case models.AutomationFully:
		return "Fully Automated"
	case models.AutomationPartially:
		return "Partially Automated"
	case models.AutomationManual:
		return "Manual Only"
	case models.AutomationUnknown:
		return "Unknown"
	default:
		return string(level)
	}
}

// formatBytes formats a byte count for display
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1fGB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1fMB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1fKB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

// formatTimestamp formats a timestamp for display
func formatTimestamp(t *time.Time) string {
	if t == nil {
		return "N/A"
	}
	return t.Format("2006-01-02 15:04:05")
}

// formatTimestampRelative formats a timestamp with relative time
func formatTimestampRelative(t *time.Time) string {
	if t == nil {
		return "N/A"
	}

	now := time.Now()
	diff := now.Sub(*t)

	days := int(diff.Hours() / 24)
	if days == 0 {
		return t.Format("15:04:05 today")
	} else if days == 1 {
		return "yesterday"
	} else if days < 7 {
		return fmt.Sprintf("%d days ago", days)
	} else if days < 30 {
		weeks := days / 7
		return fmt.Sprintf("%d weeks ago", weeks)
	} else {
		return t.Format("2006-01-02")
	}
}

// truncateChecksum truncates a checksum for display
func truncateChecksum(checksum string) string {
	if len(checksum) <= 8 {
		return checksum
	}
	return checksum[:8] + "..."
}

// getRecentActivity returns tasks sorted by most recent activity
func getRecentActivity(tasks []*models.EvidenceTaskState, limit int) []*models.EvidenceTaskState {
	// Sort by most recent timestamp (generated or submitted)
	sorted := make([]*models.EvidenceTaskState, len(tasks))
	copy(sorted, tasks)

	sort.Slice(sorted, func(i, j int) bool {
		timeI := getLatestTimestamp(sorted[i])
		timeJ := getLatestTimestamp(sorted[j])

		if timeI == nil && timeJ == nil {
			return false
		}
		if timeI == nil {
			return false
		}
		if timeJ == nil {
			return true
		}

		return timeI.After(*timeJ)
	})

	if len(sorted) > limit {
		return sorted[:limit]
	}
	return sorted
}

// getLatestTimestamp gets the most recent timestamp from a task
func getLatestTimestamp(task *models.EvidenceTaskState) *time.Time {
	var latest *time.Time

	if task.LastSubmittedAt != nil {
		latest = task.LastSubmittedAt
	}
	if task.LastGeneratedAt != nil {
		if latest == nil || task.LastGeneratedAt.After(*latest) {
			latest = task.LastGeneratedAt
		}
	}

	return latest
}

// displayTaskSummary displays a one-line summary of a task
func displayTaskSummary(cmd *cobra.Command, task *models.EvidenceTaskState, verbose bool) {
	// Find most recent window
	var latestWindow *models.WindowState
	var latestWindowName string

	for name, window := range task.Windows {
		if latestWindow == nil ||
			(window.NewestFile != nil && (latestWindow.NewestFile == nil || window.NewestFile.After(*latestWindow.NewestFile))) {
			w := window
			latestWindow = &w
			latestWindowName = name
		}
	}

	if latestWindow == nil {
		return
	}

	// Format the summary line
	var status string
	var timestamp string

	if latestWindow.SubmittedAt != nil {
		status = "submitted"
		timestamp = formatTimestampRelative(latestWindow.SubmittedAt)
	} else if latestWindow.GeneratedAt != nil {
		status = "generated"
		timestamp = formatTimestampRelative(latestWindow.GeneratedAt)
	} else {
		status = "files only"
		timestamp = formatTimestampRelative(latestWindow.NewestFile)
	}

	cmd.Printf("  %s (%s): %d files, %s %s\n",
		task.TaskRef,
		latestWindowName,
		latestWindow.FileCount,
		status,
		timestamp,
	)

	if verbose && task.TaskName != "" {
		cmd.Printf("    %s\n", task.TaskName)
	}
}

// displayWindowDetail displays detailed information about a window
func displayWindowDetail(cmd *cobra.Command, window models.WindowState) {
	cmd.Printf("\n  %s: %d files (%s)\n", window.Window, window.FileCount, formatBytes(window.TotalBytes))

	// Display generation metadata if available
	if window.HasGenerationMeta {
		cmd.Printf("    Generated: %s via %s\n", formatTimestamp(window.GeneratedAt), window.GeneratedBy)
		if len(window.ToolsUsed) > 0 {
			cmd.Printf("    Tools Used: %s\n", strings.Join(window.ToolsUsed, ", "))
		}
		cmd.Printf("    Status: %s\n", formatStateLabel(models.StateGenerated))
	}

	// Display submission metadata if available
	if window.HasSubmissionMeta {
		cmd.Printf("    Submitted: %s\n", formatTimestamp(window.SubmittedAt))
		if window.SubmissionID != "" {
			cmd.Printf("    Submission ID: %s\n", window.SubmissionID)
		}
		cmd.Printf("    Status: %s\n", window.SubmissionStatus)
	}

	// Display files
	if len(window.Files) > 0 {
		cmd.Println("    Files:")
		for _, file := range window.Files {
			checksumInfo := ""
			if file.Checksum != "" {
				checksumInfo = fmt.Sprintf(", %s", truncateChecksum(file.Checksum))
			}
			cmd.Printf("      - %s (%s%s)\n", file.Filename, formatBytes(file.SizeBytes), checksumInfo)
		}
	}
}

// displayNextSteps displays suggested next steps based on task state
func displayNextSteps(cmd *cobra.Command, taskRef string, task *models.EvidenceTaskState) {
	// Find newest window
	var newestWindow *models.WindowState
	var newestWindowName string

	for name, window := range task.Windows {
		if newestWindow == nil || name > newestWindowName {
			w := window
			newestWindow = &w
			newestWindowName = name
		}
	}

	if newestWindow == nil {
		cmd.Printf("  - Generate evidence: grctool evidence generate %s --window 2025-Q4\n", taskRef)
		return
	}

	// Determine next steps based on state
	if newestWindow.HasSubmissionMeta {
		switch newestWindow.SubmissionStatus {
		case "accepted":
			cmd.Println("  - Evidence accepted, no action needed for this window")
			cmd.Printf("  - Generate evidence for new window if needed\n")
		case "submitted":
			cmd.Println("  - Waiting for submission review")
		case "rejected":
			cmd.Printf("  - Revise and resubmit evidence for %s\n", newestWindowName)
		case "validated":
			cmd.Printf("  - Submit via: grctool evidence submit %s --window %s\n", taskRef, newestWindowName)
		default:
			cmd.Printf("  - Validate evidence: grctool evidence validate %s --window %s\n", taskRef, newestWindowName)
		}
	} else if newestWindow.HasGenerationMeta {
		cmd.Printf("  - Validate evidence: grctool evidence validate %s --window %s\n", taskRef, newestWindowName)
		cmd.Printf("  - Submit via: grctool evidence submit %s --window %s\n", taskRef, newestWindowName)
	} else {
		cmd.Printf("  - Review evidence files in %s\n", newestWindowName)
		cmd.Printf("  - Submit via: grctool evidence submit %s --window %s\n", taskRef, newestWindowName)
	}
}

// sortWindowsByDate sorts windows by name (which are date-based) in descending order
func sortWindowsByDate(windows map[string]models.WindowState) []models.WindowState {
	// Extract window names and sort
	names := make([]string, 0, len(windows))
	for name := range windows {
		names = append(names, name)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))

	// Build sorted slice
	result := make([]models.WindowState, 0, len(windows))
	for _, name := range names {
		result = append(result, windows[name])
	}

	return result
}

// normalizeTaskRef normalizes a task reference (ET-1 -> ET-0001)
func normalizeTaskRef(ref string) string {
	// Check if already normalized
	if len(ref) >= 7 && ref[:3] == "ET-" {
		return ref
	}

	// Extract number and format
	var num int
	if _, err := fmt.Sscanf(ref, "ET-%d", &num); err == nil {
		return fmt.Sprintf("ET-%04d", num)
	}

	// Return as-is if not parseable
	return ref
}
