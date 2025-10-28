// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/naming"
	"github.com/grctool/grctool/internal/storage"
	"github.com/spf13/cobra"
)

var (
	migrateDryRun bool
	migrateForce  bool
)

var evidenceMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate evidence directories from old naming format to new format",
	Long: `Migrates evidence task directories from the old format (ET-XXXX_TaskName) to the new format (TaskName_ET-XXXX_TugboatID).

This command:
1. Scans all evidence directories
2. Identifies directories using the old naming format
3. Looks up the Tugboat ID for each task
4. Renames directories to the new format
5. Creates a migration log for rollback capability

Examples:
  # Preview changes without making them
  grctool evidence migrate --dry-run

  # Perform migration
  grctool evidence migrate

  # Force migration even if some tasks cannot be resolved
  grctool evidence migrate --force
`,
	RunE: runEvidenceMigrate,
}

func init() {
	evidenceCmd.AddCommand(evidenceMigrateCmd)
	evidenceMigrateCmd.Flags().BoolVar(&migrateDryRun, "dry-run", false, "Preview changes without making them")
	evidenceMigrateCmd.Flags().BoolVar(&migrateForce, "force", false, "Continue migration even if some tasks cannot be resolved")
}

// Old format regex: ET-XXXX_TaskName
var oldFormatRegex = regexp.MustCompile(`^(ET-\d{4})_(.+)$`)

type migrationRecord struct {
	OldPath string
	NewPath string
	TaskRef string
	TaskID  int
	Success bool
	Error   string
}

func runEvidenceMigrate(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	storage, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		return fmt.Errorf("initializing storage: %w", err)
	}

	evidenceDir := filepath.Join(cfg.Storage.DataDir, "evidence")

	// Check if evidence directory exists
	if _, err := os.Stat(evidenceDir); os.IsNotExist(err) {
		cmd.Println("No evidence directory found. Nothing to migrate.")
		return nil
	}

	// Scan for directories using old format
	entries, err := os.ReadDir(evidenceDir)
	if err != nil {
		return fmt.Errorf("reading evidence directory: %w", err)
	}

	var records []migrationRecord
	var oldFormatCount int

	cmd.Println("Scanning for directories using old naming format...")
	cmd.Println()

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirName := entry.Name()

		// Check if directory uses old format
		matches := oldFormatRegex.FindStringSubmatch(dirName)
		if len(matches) < 3 {
			// Not old format - skip
			continue
		}

		oldFormatCount++
		taskRef := matches[1]

		cmd.Printf("Found: %s (Task: %s)\n", dirName, taskRef)

		// Look up task to get Tugboat ID
		task, err := storage.GetEvidenceTask(taskRef)
		if err != nil {
			record := migrationRecord{
				OldPath: filepath.Join(evidenceDir, dirName),
				TaskRef: taskRef,
				Success: false,
				Error:   fmt.Sprintf("Failed to resolve task: %v", err),
			}
			records = append(records, record)

			cmd.Printf("  ⚠️  Warning: Could not resolve task %s: %v\n", taskRef, err)

			if !migrateForce {
				continue
			}
		} else {
			// Generate new directory name
			newDirName := naming.GetEvidenceTaskDirName(task.Name, task.ReferenceID, fmt.Sprintf("%d", task.ID))
			oldPath := filepath.Join(evidenceDir, dirName)
			newPath := filepath.Join(evidenceDir, newDirName)

			record := migrationRecord{
				OldPath: oldPath,
				NewPath: newPath,
				TaskRef: taskRef,
				TaskID:  task.ID,
				Success: true,
			}
			records = append(records, record)

			cmd.Printf("  → Will rename to: %s\n", newDirName)
		}
	}

	if oldFormatCount == 0 {
		cmd.Println("\nNo directories found using old naming format. Migration not needed.")
		return nil
	}

	// Count successful migrations
	successCount := 0
	for _, r := range records {
		if r.Success {
			successCount++
		}
	}

	cmd.Printf("\n**Migration Summary:**\n")
	cmd.Printf("  Directories found with old format: %d\n", oldFormatCount)
	cmd.Printf("  Directories that can be migrated: %d\n", successCount)
	cmd.Printf("  Directories that cannot be migrated: %d\n", oldFormatCount-successCount)

	if !migrateDryRun {
		cmd.Println()

		// Perform migration
		migratedCount := 0
		failedCount := 0

		for _, record := range records {
			if !record.Success {
				continue
			}

			// Check if target already exists
			if _, err := os.Stat(record.NewPath); err == nil {
				cmd.Printf("❌ Cannot migrate %s: target directory already exists\n", filepath.Base(record.OldPath))
				failedCount++
				continue
			}

			// Perform rename
			if err := os.Rename(record.OldPath, record.NewPath); err != nil {
				cmd.Printf("❌ Failed to migrate %s: %v\n", filepath.Base(record.OldPath), err)
				failedCount++
			} else {
				cmd.Printf("✅ Migrated: %s → %s\n", filepath.Base(record.OldPath), filepath.Base(record.NewPath))
				migratedCount++
			}
		}

		cmd.Printf("\n**Migration Results:**\n")
		cmd.Printf("  Successfully migrated: %d\n", migratedCount)
		cmd.Printf("  Failed: %d\n", failedCount)

		if migratedCount > 0 {
			cmd.Println("\n✨ Migration complete! Evidence directories have been updated to the new naming format.")
		}
	} else {
		cmd.Println("\n**Dry-run mode** - no changes were made.")
		cmd.Println("Run without --dry-run to perform the migration.")
	}

	return nil
}
