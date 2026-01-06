// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/naming"
	"github.com/grctool/grctool/internal/registry"
	"github.com/grctool/grctool/internal/storage"
	"github.com/grctool/grctool/internal/tugboat"
	tugboatModels "github.com/grctool/grctool/internal/tugboat/models"
	"github.com/spf13/cobra"
)

// Predefined categories for evidence filtering
var evidenceCategories = map[string][]string{
	"risk":          {"risk", "assessment", "hartford", "insurance", "policy renewal"},
	"access-review": {"access review", "user access", "quarterly review", "ET-0050"},
	"training":      {"training", "awareness", "onboarding"},
	"termination":   {"terminated", "offboarding", "access removal"},
}

var evidenceDownloadCmd = &cobra.Command{
	Use:   "download [task-ref]",
	Short: "Download evidence submissions from Tugboat Logic",
	Long: `Downloads evidence submissions for a specific task or filtered set of tasks.

The task identifier can be:
  - ET reference: ET-0001, ET-0050 (case-insensitive)
  - Tugboat task ID: 327992 (numeric)

Use --category or --search to download evidence for multiple tasks at once.

Examples:
  # Download all evidence for a specific task
  grctool evidence download ET-0050

  # Download evidence for a date range
  grctool evidence download ET-0050 --start-date 2025-07-01 --end-date 2025-09-30

  # List available submissions without downloading
  grctool evidence download ET-0050 --list-only

  # Download all risk assessment evidence
  grctool evidence download --category risk

  # Download all access review evidence
  grctool evidence download --category access-review

  # Search by keyword
  grctool evidence download --search "Hartford"

  # Download for all tasks
  grctool evidence download --all`,
	Args: cobra.MaximumNArgs(1),
	RunE: runEvidenceDownload,
}

func init() {
	evidenceCmd.AddCommand(evidenceDownloadCmd)

	evidenceDownloadCmd.Flags().String("start-date", "", "start of date range (YYYY-MM-DD)")
	evidenceDownloadCmd.Flags().String("end-date", "", "end of date range (YYYY-MM-DD)")
	evidenceDownloadCmd.Flags().String("window", "", "collection window (e.g., 2025-Q3)")
	evidenceDownloadCmd.Flags().Bool("list-only", false, "list submissions without downloading")
	evidenceDownloadCmd.Flags().String("type", "", "filter by attachment type: file or url")
	evidenceDownloadCmd.Flags().String("output", "table", "output format: table, json, quiet")
	evidenceDownloadCmd.Flags().Bool("all", false, "download for all tasks")
	evidenceDownloadCmd.Flags().Bool("force", false, "re-download existing files")
	evidenceDownloadCmd.Flags().String("category", "", "filter by category: risk, access-review, training, termination")
	evidenceDownloadCmd.Flags().String("search", "", "search task names by keyword pattern")

	// Tab completion for task references
	evidenceDownloadCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return completeTaskRefs(cmd, args, toComplete)
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}

// downloadStats tracks download progress
type downloadStats struct {
	TasksProcessed int
	Downloaded     int
	Skipped        int
	URLReferences  int
	Errors         int
	TotalBytes     int64
}

func runEvidenceDownload(cmd *cobra.Command, args []string) error {
	// Get flags
	startDate, _ := cmd.Flags().GetString("start-date")
	endDate, _ := cmd.Flags().GetString("end-date")
	window, _ := cmd.Flags().GetString("window")
	listOnly, _ := cmd.Flags().GetBool("list-only")
	typeFilter, _ := cmd.Flags().GetString("type")
	outputFormat, _ := cmd.Flags().GetString("output")
	downloadAll, _ := cmd.Flags().GetBool("all")
	force, _ := cmd.Flags().GetBool("force")
	category, _ := cmd.Flags().GetString("category")
	search, _ := cmd.Flags().GetString("search")

	// Validate flags
	if typeFilter != "" && typeFilter != "file" && typeFilter != "url" {
		return fmt.Errorf("invalid type filter: %s (must be 'file' or 'url')", typeFilter)
	}

	if category != "" {
		if _, ok := evidenceCategories[category]; !ok {
			return fmt.Errorf("invalid category: %s\nValid categories: risk, access-review, training, termination", category)
		}
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize storage
	stor, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Determine target tasks
	var tasks []domain.EvidenceTask

	if len(args) == 1 {
		// Single task specified
		task, err := stor.GetEvidenceTask(args[0])
		if err != nil {
			return fmt.Errorf("task not found: %w\n\nPossible solutions:\n  • Run: grctool sync\n  • Verify task exists in Tugboat Logic\n  • List tasks: grctool evidence list", err)
		}
		tasks = []domain.EvidenceTask{*task}
	} else if category != "" || search != "" || downloadAll {
		// Multiple tasks - filter by category/search
		allTasks, err := stor.GetAllEvidenceTasks()
		if err != nil {
			return fmt.Errorf("failed to get evidence tasks: %w", err)
		}
		tasks = filterTasks(allTasks, category, search, downloadAll)
		if len(tasks) == 0 {
			if category != "" {
				return fmt.Errorf("no tasks found matching category: %s", category)
			}
			if search != "" {
				return fmt.Errorf("no tasks found matching search: %s", search)
			}
			return fmt.Errorf("no tasks found")
		}
	} else {
		return fmt.Errorf("specify a task reference, --category, --search, or --all")
	}

	// Initialize Tugboat client
	tugboatClient := tugboat.NewClient(&cfg.Tugboat, nil)

	// Load registry for reference IDs
	evidenceRegistry := registry.NewEvidenceTaskRegistry(cfg.Storage.DataDir)
	if err := evidenceRegistry.LoadRegistry(); err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	ctx := context.Background()
	stats := &downloadStats{}

	// Process each task
	for _, task := range tasks {
		ref, ok := evidenceRegistry.GetReference(task.ID)
		if !ok {
			ref = fmt.Sprintf("TASK-%d", task.ID)
		}

		if outputFormat != "quiet" {
			cmd.Printf("\n📋 %s (%s)\n", task.Name, ref)
		}

		// Query attachments
		var attachments []tugboatModels.EvidenceAttachment
		if window != "" {
			// Convert window (e.g., 2025-Q3) to date range
			start, end := windowToDateRange(window)
			attachments, err = tugboatClient.GetEvidenceAttachmentsByTaskAndWindow(ctx, task.ID, start, end)
		} else if startDate != "" || endDate != "" {
			// Use specified date range (default to wide range if one is missing)
			if startDate == "" {
				startDate = "2013-01-01"
			}
			if endDate == "" {
				endDate = "2099-12-31"
			}
			attachments, err = tugboatClient.GetEvidenceAttachmentsByTaskAndWindow(ctx, task.ID, startDate, endDate)
		} else {
			// Get all attachments
			attachments, err = tugboatClient.GetEvidenceAttachmentsByTask(ctx, task.ID)
		}

		if err != nil {
			cmd.Printf("  ⚠️  Failed to query: %v\n", err)
			stats.Errors++
			continue
		}

		// Filter by type if specified
		if typeFilter != "" {
			attachments = filterAttachmentsByType(attachments, typeFilter)
		}

		if len(attachments) == 0 {
			if outputFormat != "quiet" {
				cmd.Println("  No submissions found")
			}
			continue
		}

		stats.TasksProcessed++

		if listOnly {
			// List mode - display submissions without downloading
			displaySubmissions(cmd, attachments, outputFormat)
		} else {
			// Download mode
			taskStats := downloadAttachments(ctx, cmd, cfg, tugboatClient, &task, ref, attachments, force, outputFormat)
			stats.Downloaded += taskStats.Downloaded
			stats.Skipped += taskStats.Skipped
			stats.URLReferences += taskStats.URLReferences
			stats.Errors += taskStats.Errors
			stats.TotalBytes += taskStats.TotalBytes
		}
	}

	// Summary
	if outputFormat != "quiet" {
		cmd.Println()
		cmd.Println("═══════════════════════════════════════════════════════")
		if listOnly {
			cmd.Printf("Tasks scanned: %d\n", stats.TasksProcessed)
		} else {
			cmd.Printf("Tasks processed: %d\n", stats.TasksProcessed)
			cmd.Printf("Files downloaded: %d (%s)\n", stats.Downloaded, formatDownloadBytes(stats.TotalBytes))
			if stats.Skipped > 0 {
				cmd.Printf("Files skipped (existing): %d\n", stats.Skipped)
			}
			if stats.URLReferences > 0 {
				cmd.Printf("URL references saved: %d\n", stats.URLReferences)
			}
			if stats.Errors > 0 {
				cmd.Printf("Errors: %d\n", stats.Errors)
			}
		}
	}

	return nil
}

// filterTasks filters tasks by category, search term, or returns all
func filterTasks(tasks []domain.EvidenceTask, category, search string, all bool) []domain.EvidenceTask {
	if all {
		return tasks
	}

	var filtered []domain.EvidenceTask

	for _, task := range tasks {
		taskNameLower := strings.ToLower(task.Name)
		taskRefLower := strings.ToLower(task.ReferenceID)

		if category != "" {
			keywords := evidenceCategories[category]
			for _, kw := range keywords {
				if strings.Contains(taskNameLower, strings.ToLower(kw)) ||
					strings.Contains(taskRefLower, strings.ToLower(kw)) {
					filtered = append(filtered, task)
					break
				}
			}
		} else if search != "" {
			searchLower := strings.ToLower(search)
			if strings.Contains(taskNameLower, searchLower) ||
				strings.Contains(taskRefLower, searchLower) {
				filtered = append(filtered, task)
			}
		}
	}

	return filtered
}

// filterAttachmentsByType filters attachments by type (file or url)
func filterAttachmentsByType(attachments []tugboatModels.EvidenceAttachment, typeFilter string) []tugboatModels.EvidenceAttachment {
	var filtered []tugboatModels.EvidenceAttachment
	for _, att := range attachments {
		if att.Type == typeFilter {
			filtered = append(filtered, att)
		}
	}
	return filtered
}

// windowToDateRange converts a window like "2025-Q3" to start and end dates
func windowToDateRange(window string) (string, string) {
	// Parse window format: YYYY-Qn
	re := regexp.MustCompile(`^(\d{4})-Q([1-4])$`)
	matches := re.FindStringSubmatch(window)
	if len(matches) != 3 {
		// Default to wide range if parsing fails
		return "2013-01-01", "2099-12-31"
	}

	year := matches[1]
	quarter, _ := strconv.Atoi(matches[2])

	var startMonth, endMonth int
	switch quarter {
	case 1:
		startMonth, endMonth = 1, 3
	case 2:
		startMonth, endMonth = 4, 6
	case 3:
		startMonth, endMonth = 7, 9
	case 4:
		startMonth, endMonth = 10, 12
	}

	startDate := fmt.Sprintf("%s-%02d-01", year, startMonth)
	// Get last day of end month
	endYear, _ := strconv.Atoi(year)
	lastDay := time.Date(endYear, time.Month(endMonth+1), 0, 0, 0, 0, 0, time.UTC).Day()
	endDate := fmt.Sprintf("%s-%02d-%02d", year, endMonth, lastDay)

	return startDate, endDate
}

// displaySubmissions shows a list of submissions
func displaySubmissions(cmd *cobra.Command, attachments []tugboatModels.EvidenceAttachment, format string) {
	fileCount := 0
	urlCount := 0

	if format == "table" {
		cmd.Printf("\n  %-8s %-6s %-12s %-40s %s\n", "ID", "Type", "Collected", "Filename", "Size")
		cmd.Printf("  %-8s %-6s %-12s %-40s %s\n", "--------", "------", "------------", "----------------------------------------", "--------")
	}

	for _, att := range attachments {
		if att.Type == "file" {
			fileCount++
			filename := "unknown"
			size := "-"
			if att.Attachment != nil {
				filename = att.Attachment.OriginalFilename
				if len(filename) > 40 {
					filename = filename[:37] + "..."
				}
			}
			if format == "table" {
				cmd.Printf("  %-8d %-6s %-12s %-40s %s\n", att.ID, att.Type, att.Collected, filename, size)
			}
		} else if att.Type == "url" {
			urlCount++
			url := att.URL
			if len(url) > 40 {
				url = url[:37] + "..."
			}
			if format == "table" {
				cmd.Printf("  %-8d %-6s %-12s %-40s %s\n", att.ID, att.Type, att.Collected, url, "-")
			}
		}
	}

	cmd.Printf("\n  Total: %d submissions (%d files, %d URLs)\n", len(attachments), fileCount, urlCount)
}

// downloadAttachments downloads all attachments for a task
func downloadAttachments(ctx context.Context, cmd *cobra.Command, cfg *config.Config, client *tugboat.Client, task *domain.EvidenceTask, ref string, attachments []tugboatModels.EvidenceAttachment, force bool, format string) *downloadStats {
	stats := &downloadStats{}

	// Group attachments by window
	windowMap := make(map[string][]tugboatModels.EvidenceAttachment)
	for _, att := range attachments {
		window := getWindowFromDate(att.Collected)
		windowMap[window] = append(windowMap[window], att)
	}

	for window, windowAttachments := range windowMap {
		// Create directory
		taskDirName := naming.GetEvidenceTaskDirName(task.Name, ref, strconv.Itoa(task.ID))
		evidenceDir := filepath.Join(cfg.Storage.DataDir, "evidence", taskDirName, window, naming.SubfolderArchive)
		if err := os.MkdirAll(evidenceDir, 0755); err != nil {
			cmd.Printf("  ⚠️  Failed to create directory: %v\n", err)
			stats.Errors++
			continue
		}

		if format != "quiet" {
			cmd.Printf("  📁 %s (%d items)\n", window, len(windowAttachments))
		}

		for i, att := range windowAttachments {
			if att.Type == "file" && att.Attachment != nil {
				filename := att.Attachment.OriginalFilename
				if filename == "" {
					filename = fmt.Sprintf("attachment_%d", att.ID)
				}
				destPath := filepath.Join(evidenceDir, filename)

				// Check if file exists
				if !force {
					if _, err := os.Stat(destPath); err == nil {
						if format != "quiet" {
							cmd.Printf("     [%d/%d] %s (skipped - exists)\n", i+1, len(windowAttachments), filename)
						}
						stats.Skipped++
						continue
					}
				}

				// Download
				if format != "quiet" {
					cmd.Printf("     [%d/%d] %s", i+1, len(windowAttachments), filename)
				}

				if err := client.DownloadAttachment(ctx, att.ID, destPath); err != nil {
					if format != "quiet" {
						cmd.Printf(" ❌ %v\n", err)
					}
					stats.Errors++
					continue
				}

				// Get file size
				if info, err := os.Stat(destPath); err == nil {
					stats.TotalBytes += info.Size()
					if format != "quiet" {
						cmd.Printf(" ✓ %s\n", formatDownloadBytes(info.Size()))
					}
				} else if format != "quiet" {
					cmd.Println(" ✓")
				}

				stats.Downloaded++

			} else if att.Type == "url" {
				// Save URL reference
				filename := fmt.Sprintf("url_reference_%d.txt", att.ID)
				destPath := filepath.Join(evidenceDir, filename)

				// Check if file exists
				if !force {
					if _, err := os.Stat(destPath); err == nil {
						stats.Skipped++
						continue
					}
				}

				urlContent := fmt.Sprintf("URL: %s\nNotes: %s\nCollected: %s\n", att.URL, att.Notes, att.Collected)
				if err := os.WriteFile(destPath, []byte(urlContent), 0644); err != nil {
					stats.Errors++
					continue
				}

				stats.URLReferences++
			}
		}
	}

	return stats
}

// getWindowFromDate converts a date string (YYYY-MM-DD) to a window identifier (YYYY-Qn)
func getWindowFromDate(dateStr string) string {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		t = time.Now()
	}

	year := t.Year()
	month := t.Month()

	var quarter int
	switch {
	case month >= 1 && month <= 3:
		quarter = 1
	case month >= 4 && month <= 6:
		quarter = 2
	case month >= 7 && month <= 9:
		quarter = 3
	default:
		quarter = 4
	}

	return fmt.Sprintf("%d-Q%d", year, quarter)
}

// formatDownloadBytes formats bytes into human-readable format
func formatDownloadBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
