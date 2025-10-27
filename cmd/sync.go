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
	"time"

	"github.com/grctool/grctool/internal/appcontext"
	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/services"
	"github.com/grctool/grctool/internal/storage"
	"github.com/grctool/grctool/internal/tugboat"
	"github.com/grctool/grctool/internal/vcr"
	"github.com/spf13/cobra"
)

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize data from Tugboat Logic",
	Long: `Download and store policies, procedures, and evidence tasks from Tugboat Logic.

This command connects to your Tugboat Logic instance and downloads all relevant data
for local processing and evidence collection. The data is stored locally to improve
performance and enable offline work.`,
	RunE: runSync,
}

// syncValidateCmd represents the sync validate command
var syncValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate synchronized data integrity",
	Long: `Validate the integrity and consistency of locally synchronized data.

This command checks that the local data is consistent with remote data,
validates relationships between entities, and identifies any data integrity issues.`,
	RunE: runSyncValidate,
}

// syncSummaryCmd represents the sync summary command
var syncSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show sync summary and statistics",
	Long: `Display a summary of the last sync operation including counts,
timing information, and data freshness indicators.`,
	RunE: runSyncSummary,
}

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.AddCommand(syncValidateCmd)
	syncCmd.AddCommand(syncSummaryCmd)

	// Sync options
	syncCmd.Flags().Bool("incremental", false, "perform incremental sync only")
	syncCmd.Flags().Bool("policies", false, "sync policies only")
	syncCmd.Flags().Bool("procedures", false, "sync procedures only")
	syncCmd.Flags().Bool("evidence", false, "sync evidence tasks only")
	syncCmd.Flags().Bool("controls", false, "sync controls only")
	syncCmd.Flags().Bool("dry-run", false, "show what would be synced without making changes")
	syncCmd.Flags().Bool("force", false, "force full sync even if data is recent")

	// Sync validate options
	syncValidateCmd.Flags().Bool("policies", false, "validate policies only")
	syncValidateCmd.Flags().Bool("controls", false, "validate controls only")
	syncValidateCmd.Flags().Bool("evidence", false, "validate evidence tasks only")
	syncValidateCmd.Flags().String("framework", "", "validate specific framework only")
	syncValidateCmd.Flags().Bool("json", false, "output validation results as JSON")

}

func runSync(cmd *cobra.Command, args []string) error {
	// Create context with basic enrichment
	ctx := context.Background()
	ctx = appcontext.EnrichContext(ctx, cmd)
	ctx = appcontext.WithRequestID(ctx, appcontext.GenerateRequestID())

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	ctx = appcontext.WithConfig(ctx, cfg)

	// Set up logger and add to context
	log := logger.WithComponent("sync")
	ctx = appcontext.WithLogger(ctx, log)
	cmd.SetContext(ctx) // Update command context with everything

	policies, _ := cmd.Flags().GetBool("policies")
	procedures, _ := cmd.Flags().GetBool("procedures")
	evidence, _ := cmd.Flags().GetBool("evidence")
	controls, _ := cmd.Flags().GetBool("controls")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	cmd.Println("🔄 Starting Tugboat Logic sync with new architecture...")
	log.Info("starting sync operation",
		logger.Field{Key: "policies", Value: policies},
		logger.Field{Key: "evidence", Value: evidence},
		logger.Field{Key: "controls", Value: controls},
		logger.Field{Key: "dry_run", Value: dryRun},
	)

	if dryRun {
		cmd.Println("🔍 Dry run mode - no changes will be made")
	}

	// Get configuration from context (with type assertion)
	cfgInterface := appcontext.GetConfig(ctx)
	if cfgInterface == nil {
		return fmt.Errorf("configuration not found in context")
	}
	cfg, ok := cfgInterface.(*config.Config)
	if !ok {
		return fmt.Errorf("configuration has wrong type in context")
	}

	// Get VCR config from context (with type assertion) or environment
	var vcrConfig *vcr.Config
	vcrInterface := appcontext.GetVCRConfig(ctx)
	if vcrInterface != nil {
		vcrConfig, _ = vcrInterface.(*vcr.Config)
	}
	if vcrConfig == nil {
		// Check environment for VCR_MODE (test/dev only)
		vcrConfig = vcr.FromEnvironment()
	}

	// Initialize Tugboat client
	client := tugboat.NewClient(&cfg.Tugboat, vcrConfig)
	defer client.Close()

	// Test connection
	cmd.Println("🔗 Testing connection to Tugboat Logic...")
	if err := client.TestConnection(ctx); err != nil {
		log.Error("connection test failed", logger.Error(err))
		return fmt.Errorf("connection test failed: %w", err)
	}
	cmd.Println("✅ Connection successful")

	// Initialize unified storage with full config to support custom paths
	storage, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		log.Error("failed to initialize unified storage", logger.Error(err))
		return fmt.Errorf("failed to initialize unified storage: %w", err)
	}

	// Initialize sync service with logger from context
	syncService := services.NewSyncService(client, storage, cfg, log)

	// Determine what to sync
	syncAll := !policies && !procedures && !evidence && !controls

	// Prepare sync options
	syncOptions := services.SyncOptions{
		OrgID:       cfg.Tugboat.OrgID,
		Framework:   "", // Sync all frameworks
		Policies:    syncAll || policies,
		Controls:    syncAll || controls,
		Evidence:    syncAll || evidence,
		Submissions: syncAll, // Always sync submissions when doing a full sync
	}

	if dryRun {
		cmd.Println("📋 Dry run - would sync:")
		if syncOptions.Policies {
			cmd.Println("  ✓ Policies")
		}
		if syncOptions.Controls {
			cmd.Println("  ✓ Controls")
		}
		if syncOptions.Evidence {
			cmd.Println("  ✓ Evidence tasks")
		}
		if syncOptions.Submissions {
			cmd.Println("  ✓ Submissions")
		}
		if procedures {
			cmd.Println("  ⚠️  Procedures (not yet implemented)")
		}
		return nil
	}

	// Perform the sync using the new service
	cmd.Println("🚀 Starting sync with new service architecture...")
	result, err := syncService.SyncAll(ctx, syncOptions)
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	// Display results
	cmd.Printf("✅ Sync completed in %v\n", result.Duration)
	cmd.Println("📊 Sync Results:")

	if syncOptions.Policies {
		cmd.Printf("  📋 Policies: %d total, %d synced, %d detailed, %d errors\n",
			result.Policies.Total, result.Policies.Synced, result.Policies.Detailed, result.Policies.Errors)
	}

	if syncOptions.Controls {
		cmd.Printf("  🛡️  Controls: %d total, %d synced, %d detailed, %d errors\n",
			result.Controls.Total, result.Controls.Synced, result.Controls.Detailed, result.Controls.Errors)
	}

	if syncOptions.Evidence {
		cmd.Printf("  📝 Evidence Tasks: %d total, %d synced, %d detailed, %d errors\n",
			result.EvidenceTasks.Total, result.EvidenceTasks.Synced, result.EvidenceTasks.Detailed, result.EvidenceTasks.Errors)
	}

	if syncOptions.Submissions {
		cmd.Printf("  📎 Submissions: %d total, %d tasks synced, %d files downloaded, %d errors\n",
			result.Submissions.Total, result.Submissions.Synced, result.Submissions.Downloaded, result.Submissions.Errors)
	}

	if len(result.Errors) > 0 {
		cmd.Printf("⚠️  Encountered %d errors during sync:\n", len(result.Errors))
		for _, errMsg := range result.Errors {
			cmd.Printf("  - %s\n", errMsg)
		}
	}

	if procedures {
		cmd.Println("⚠️  Procedure sync not yet implemented in new architecture")
	}

	// Update last sync time
	if err := storage.SetSyncTime("full_sync", time.Now()); err != nil {
		cmd.Printf("⚠️  Warning: failed to save sync time: %v\n", err)
	}

	cmd.Println("✅ Sync completed successfully with new architecture")
	return nil
}

func runSyncValidate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get flags
	policies, _ := cmd.Flags().GetBool("policies")
	controls, _ := cmd.Flags().GetBool("controls")
	evidence, _ := cmd.Flags().GetBool("evidence")
	framework, _ := cmd.Flags().GetString("framework")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Check environment for VCR_MODE (test/dev only)
	vcrConfig := vcr.FromEnvironment()

	// Initialize Tugboat client
	client := tugboat.NewClient(&cfg.Tugboat, vcrConfig)
	defer client.Close()

	// Initialize unified storage with full config to support custom paths
	storage, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		return fmt.Errorf("failed to initialize domain storage: %w", err)
	}

	// Initialize sync service
	syncService := services.NewSyncService(client, storage, cfg, logger.WithComponent("sync"))

	// Prepare sync options for validation
	syncOptions := services.SyncOptions{
		OrgID:     cfg.Tugboat.OrgID,
		Framework: framework,
		Policies:  policies,
		Controls:  controls,
		Evidence:  evidence,
	}

	if !jsonOutput {
		cmd.Println("🔍 Validating synchronized data...")
	}

	// Perform validation
	result, err := syncService.ValidateSync(ctx, syncOptions)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Output results
	if jsonOutput {
		jsonData, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal validation results: %w", err)
		}
		cmd.Println(string(jsonData))
	} else {
		printSyncValidationResults(cmd, result)
	}

	// Return error if validation found issues
	if result["status"] == "error" {
		return fmt.Errorf("validation found critical issues")
	}

	return nil
}

func runSyncSummary(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Check environment for VCR_MODE (test/dev only)
	vcrConfig := vcr.FromEnvironment()

	// Initialize Tugboat client
	client := tugboat.NewClient(&cfg.Tugboat, vcrConfig)
	defer client.Close()

	// Initialize unified storage with full config to support custom paths
	storage, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		return fmt.Errorf("failed to initialize domain storage: %w", err)
	}

	// Initialize sync service
	syncService := services.NewSyncService(client, storage, cfg, logger.WithComponent("sync"))

	// Get sync summary
	summary, err := syncService.GetSyncSummary(ctx)
	if err != nil {
		return fmt.Errorf("failed to get sync summary: %w", err)
	}

	// Display summary
	printSyncSummary(cmd, summary)

	return nil
}

// printSyncValidationResults prints validation results in a human-readable format
func printSyncValidationResults(cmd *cobra.Command, result map[string]interface{}) {
	status := result["status"].(string)
	duration := result["duration"].(string)

	cmd.Printf("Sync Validation Results (took %s)\n", duration)
	cmd.Println("=====================================")

	// Overall status
	var statusIcon string
	switch status {
	case "valid":
		statusIcon = "✅"
	case "warning":
		statusIcon = "⚠️"
	case "error":
		statusIcon = "❌"
	default:
		statusIcon = "❓"
	}

	cmd.Printf("%s Overall Status: %s\n\n", statusIcon, status)

	// Individual checks
	checks, ok := result["checks"].(map[string]interface{})
	if ok {
		for checkName, checkData := range checks {
			if checkMap, ok := checkData.(map[string]interface{}); ok {
				checkStatus := checkMap["status"].(string)
				var checkIcon string
				switch checkStatus {
				case "valid":
					checkIcon = "✅"
				case "warning":
					checkIcon = "⚠️"
				case "error":
					checkIcon = "❌"
				default:
					checkIcon = "❓"
				}

				cmd.Printf("%s %s\n", checkIcon, checkName)
				if message, exists := checkMap["message"]; exists {
					cmd.Printf("   %s\n", message)
				}
				if checkMap["local_count"] != nil && checkMap["remote_count"] != nil {
					cmd.Printf("   Local: %v, Remote: %v\n", checkMap["local_count"], checkMap["remote_count"])
				}
				cmd.Println()
			}
		}
	}

	// Errors
	if errors, ok := result["errors"].([]string); ok && len(errors) > 0 {
		cmd.Println("Errors:")
		for _, err := range errors {
			cmd.Printf("  - %s\n", err)
		}
		cmd.Println()
	}

	// Warnings
	if warnings, ok := result["warnings"].([]string); ok && len(warnings) > 0 {
		cmd.Println("Warnings:")
		for _, warning := range warnings {
			cmd.Printf("  - %s\n", warning)
		}
		cmd.Println()
	}
}

// printSyncSummary prints sync summary in a human-readable format
func printSyncSummary(cmd *cobra.Command, summary map[string]interface{}) {
	cmd.Println("Sync Summary")
	cmd.Println("============")

	if lastSync, ok := summary["last_sync"].(time.Time); ok {
		if lastSync.IsZero() {
			cmd.Println("📅 Last Sync: Never")
		} else {
			cmd.Printf("📅 Last Sync: %s\n", lastSync.Format("2006-01-02 15:04:05"))
		}
	}

	if status, ok := summary["sync_status"].(string); ok {
		var statusIcon string
		switch status {
		case "completed":
			statusIcon = "✅"
		case "failed":
			statusIcon = "❌"
		case "in_progress":
			statusIcon = "🔄"
		case "never_synced":
			statusIcon = "⏳"
		default:
			statusIcon = "❓"
		}
		cmd.Printf("📊 Status: %s %s\n", statusIcon, status)
	}

	if freshness, ok := summary["data_freshness"].(string); ok {
		cmd.Printf("⏰ Data Freshness: %s\n", freshness)
	}

	cmd.Println("\n📈 Data Counts:")
	if policies, ok := summary["total_policies"].(int); ok {
		cmd.Printf("  📋 Policies: %d\n", policies)
	}
	if controls, ok := summary["total_controls"].(int); ok {
		cmd.Printf("  🛡️  Controls: %d\n", controls)
	}
	if evidenceTasks, ok := summary["total_evidence_tasks"].(int); ok {
		cmd.Printf("  📝 Evidence Tasks: %d\n", evidenceTasks)
	}

	if nextScheduled, ok := summary["next_scheduled"].(time.Time); ok {
		cmd.Printf("\n⏰ Next Scheduled Sync: %s\n", nextScheduled.Format("2006-01-02 15:04:05"))
	}
}
