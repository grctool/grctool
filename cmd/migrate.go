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

	"github.com/grctool/grctool/internal/services"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate on-disk JSON files to use string IDs",
	Long: `Scan the data directory for JSON files (policies, controls, evidence tasks)
and convert any numeric ID fields to string-quoted IDs for consistency with the
current domain model.

Fields checked: id, task_id, control_id, policy_id (including nested objects).

This migration is idempotent — running it multiple times is safe.`,
	RunE: runMigrate,
}

func init() {
	rootCmd.AddCommand(migrateCmd)

	migrateCmd.Flags().Bool("dry-run", false, "report what would change without modifying files")
	migrateCmd.Flags().String("data-dir", "", "override data directory (default: from config)")
}

func runMigrate(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	dataDir, _ := cmd.Flags().GetString("data-dir")

	if dataDir == "" {
		dataDir = viper.GetString("storage.data_dir")
	}
	if dataDir == "" {
		return fmt.Errorf("data directory not configured; use --data-dir or set storage.data_dir in config")
	}

	if dryRun {
		fmt.Println("Dry-run mode: no files will be modified")
	}

	result, err := services.MigrateJSONFiles(dataDir, dryRun)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Report changes
	for _, c := range result.Changes {
		fmt.Printf("  %s: %s %v -> %q\n", c.FilePath, c.Field, c.OldValue, c.NewValue)
	}

	// Report errors
	for _, e := range result.Errors {
		fmt.Printf("  ERROR: %s\n", e)
	}

	// Summary
	fmt.Printf("\nMigration summary:\n")
	fmt.Printf("  Files scanned:  %d\n", result.FilesScanned)
	fmt.Printf("  Files modified: %d\n", result.FilesModified)
	fmt.Printf("  Files failed:   %d\n", result.FilesFailed)

	if dryRun && result.FilesModified > 0 {
		fmt.Println("\nRe-run without --dry-run to apply changes.")
	}

	return nil
}
