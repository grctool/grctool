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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/services"
	"github.com/grctool/grctool/internal/storage"
	"github.com/spf13/cobra"
)

// storageAdapter adapts storage.Storage to services.Storage interface
type storageAdapterCleanup struct {
	storage *storage.Storage
}

func (sa *storageAdapterCleanup) GetEvidenceTask(ctx context.Context, taskID int) (*domain.EvidenceTask, error) {
	idStr := fmt.Sprintf("%d", taskID)
	return sa.storage.GetEvidenceTask(idStr)
}

var evidenceCleanupCmd = &cobra.Command{
	Use:   "cleanup [task-ref]",
	Short: "Organize evidence from flat structure to subfolder structure",
	Long: `Organize evidence files from flat directory structure into the new subfolder structure (wip/ready/submitted).

Files are organized based on their metadata:
  - Has .submission/submission.yaml → submitted/
  - Has .validation/validation.yaml (no submission) → ready/
  - Has .generation/metadata.yaml only → wip/
  - No metadata → wip/ (default)

Metadata directories (.generation/, .validation/, .submission/) are moved with their files.
The .context/ directory stays at the window level as it's shared.

Examples:
  # Organize evidence for a specific task and window
  grctool evidence cleanup ET-0001 --window 2025-Q4

  # Organize evidence for all tasks
  grctool evidence cleanup --all

  # Preview changes without making them
  grctool evidence cleanup ET-0001 --window 2025-Q4 --dry-run`,
	Args: cobra.MaximumNArgs(1),
	RunE: runEvidenceCleanup,
}

func init() {
	evidenceCleanupCmd.Flags().String("window", "", "specific window to clean up (e.g., 2025-Q4)")
	evidenceCleanupCmd.Flags().Bool("all", false, "clean up all evidence tasks")
	evidenceCleanupCmd.Flags().Bool("dry-run", false, "show what would be done without making changes")
	evidenceCleanupCmd.Flags().StringP("output", "o", "", "output results to JSON file")
}

func runEvidenceCleanup(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get flags
	window, _ := cmd.Flags().GetString("window")
	all, _ := cmd.Flags().GetBool("all")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	outputFile, _ := cmd.Flags().GetString("output")

	// Validate arguments
	if !all && len(args) == 0 {
		return fmt.Errorf("task reference required (or use --all flag)")
	}

	if !all && window == "" {
		return fmt.Errorf("--window flag required when cleaning up specific task")
	}

	if all && len(args) > 0 {
		return fmt.Errorf("cannot specify task reference with --all flag")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create services
	evidenceDir := filepath.Join(cfg.Storage.DataDir, "evidence")

	// Initialize logger
	consoleLoggerCfg := cfg.Logging.Loggers["console"]
	log, err := logger.New((&consoleLoggerCfg).ToLoggerConfig())
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Initialize storage
	store, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Wrap storage with adapter
	storageAdapter := &storageAdapterCleanup{storage: store}

	scanner := services.NewEvidenceScanner(evidenceDir, storageAdapter, log)
	cleanupService := services.NewEvidenceCleanupService(evidenceDir, scanner, log)

	if dryRun {
		fmt.Println("=== DRY RUN MODE - No changes will be made ===\n")
	}

	if all {
		// Clean up all tasks
		fmt.Println("Cleaning up all evidence tasks...")
		summary, err := cleanupService.CleanupAll(ctx, dryRun)
		if err != nil {
			return fmt.Errorf("cleanup failed: %w", err)
		}

		// Display summary
		displayCleanupSummary(summary)

		// Save to file if requested
		if outputFile != "" {
			if err := saveCleanupSummaryToFile(summary, outputFile); err != nil {
				return fmt.Errorf("failed to save results: %w", err)
			}
			fmt.Printf("\nResults saved to: %s\n", outputFile)
		}
	} else {
		// Clean up specific task
		taskRef := args[0]
		fmt.Printf("Cleaning up evidence for %s / %s...\n\n", taskRef, window)

		result, err := cleanupService.CleanupTask(ctx, taskRef, window, dryRun)
		if err != nil {
			return fmt.Errorf("cleanup failed: %w", err)
		}

		// Display result
		displayCleanupResult(result)

		// Save to file if requested
		if outputFile != "" {
			if err := saveCleanupResultToFile(result, outputFile); err != nil {
				return fmt.Errorf("failed to save results: %w", err)
			}
			fmt.Printf("\nResults saved to: %s\n", outputFile)
		}
	}

	if dryRun {
		fmt.Println("\n=== DRY RUN COMPLETE - No actual changes were made ===")
	}

	return nil
}

func displayCleanupResult(result *services.CleanupResult) {
	if !result.WasFlatStructure {
		fmt.Printf("✓ Task %s / %s already uses subfolder structure\n", result.TaskRef, result.Window)
		return
	}

	fmt.Printf("Task: %s\n", result.TaskRef)
	fmt.Printf("Window: %s\n", result.Window)
	fmt.Printf("Structure: Flat → Subfolder\n\n")

	if len(result.FilesOrganized) > 0 {
		fmt.Println("Files organized:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "  Subfolder\tFiles\n")
		fmt.Fprintf(w, "  ---------\t-----\n")
		for subfolder, count := range result.FilesOrganized {
			fmt.Fprintf(w, "  %s\t%d\n", subfolder, count)
		}
		w.Flush()
		fmt.Println()
	}

	if len(result.MetadataMoved) > 0 {
		fmt.Printf("Metadata directories moved: %v\n", result.MetadataMoved)
	}

	if len(result.Errors) > 0 {
		fmt.Println("\nErrors:")
		for _, err := range result.Errors {
			fmt.Printf("  ✗ %s\n", err)
		}
	} else {
		fmt.Println("✓ Cleanup completed successfully")
	}
}

func displayCleanupSummary(summary *services.CleanupSummary) {
	fmt.Printf("Total tasks scanned: %d\n", summary.TotalTasks)
	fmt.Printf("Total windows: %d\n", summary.TotalWindows)
	fmt.Printf("Windows cleaned: %d\n", summary.WindowsCleaned)
	fmt.Printf("Files organized: %d\n\n", summary.FilesOrganized)

	if len(summary.Results) > 0 {
		fmt.Println("Cleanup Results:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "Task\tWindow\tStatus\tFiles\n")
		fmt.Fprintf(w, "----\t------\t------\t-----\n")
		for _, result := range summary.Results {
			status := "Already organized"
			totalFiles := 0
			if result.WasFlatStructure {
				status = "Organized"
				for _, count := range result.FilesOrganized {
					totalFiles += count
				}
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", result.TaskRef, result.Window, status, totalFiles)
		}
		w.Flush()
		fmt.Println()
	}

	if len(summary.Errors) > 0 {
		fmt.Println("Errors:")
		for _, err := range summary.Errors {
			fmt.Printf("  ✗ %s\n", err)
		}
	} else {
		fmt.Println("✓ All cleanups completed successfully")
	}
}

func saveCleanupResultToFile(result *services.CleanupResult, filename string) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func saveCleanupSummaryToFile(summary *services.CleanupSummary, filename string) error {
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal summary: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
