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
	"path/filepath"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/appcontext"
	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/tools/terraform"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// terraformIndexCmd represents the terraform-index command
var terraformIndexCmd = &cobra.Command{
	Use:   "terraform-index",
	Short: "Manage Terraform security indices for fast evidence collection",
	Long: `Terraform Index Management Commands

Build, query, and manage persistent security indices for Terraform configurations.
Indices provide instant (<10ms) query performance vs. full scans (30s+).

Examples:
  # Build or rebuild the index
  grctool terraform-index build

  # Check index status and health
  grctool terraform-index status

  # Query resources by SOC2 control
  grctool terraform-index query --control CC6.8

  # Clear the index cache
  grctool terraform-index clear

  # Validate index integrity
  grctool terraform-index validate`,
}

// buildCmd builds or rebuilds the Terraform security index
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build or rebuild the Terraform security index",
	Long: `Build Terraform Security Index

Scans all configured Terraform directories and builds a comprehensive security
index with SOC2 control mappings, security attributes, and compliance data.

The index is persisted to disk with compression and checksum-based invalidation.`,
	RunE: runBuild,
}

// terraformStatusCmd shows the current index status
var terraformStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Terraform index status and health",
	Long: `Display Index Status

Shows index existence, age, size, version, and key statistics including:
- Total indexed resources and files
- Scan duration
- Compliance coverage
- Control coverage by SOC2 control`,
	RunE: runStatus,
}

// queryCmd queries the index
var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query the Terraform security index",
	Long: `Query Terraform Index

Fast querying of indexed Terraform resources by various criteria.
Supports multiple filter types and output formats.

Examples:
  # Query by control code
  grctool terraform-index query --control CC6.8

  # Query by security attribute
  grctool terraform-index query --attribute encryption

  # Query by resource type
  grctool terraform-index query --resource-type aws_kms_key

  # Query by environment
  grctool terraform-index query --environment prod

  # Query with JSON output
  grctool terraform-index query --control CC6.1 --output json`,
	RunE: runQuery,
}

// clearCmd clears the index
var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear the Terraform security index",
	Long: `Clear Terraform Index

Deletes the persisted index file. The index will be rebuilt on next use.`,
	RunE: runClear,
}

// validateCmd validates the index
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate Terraform index integrity",
	Long: `Validate Index Integrity

Checks the index file for:
- Version compatibility
- Data integrity
- Metadata consistency
- Corruption detection`,
	RunE: runValidate,
}

// Command flags
var (
	buildForce        bool
	buildPaths        []string
	buildOutputPath   string
	queryControl      string
	queryAttribute    string
	queryResourceType string
	queryEnvironment  string
	queryOutputFormat string
	clearYes          bool
)

func init() {
	rootCmd.AddCommand(terraformIndexCmd)

	// Add subcommands
	terraformIndexCmd.AddCommand(buildCmd)
	terraformIndexCmd.AddCommand(terraformStatusCmd)
	terraformIndexCmd.AddCommand(queryCmd)
	terraformIndexCmd.AddCommand(clearCmd)
	terraformIndexCmd.AddCommand(validateCmd)

	// Build command flags
	buildCmd.Flags().BoolVar(&buildForce, "force", false, "Force rebuild even if index is fresh")
	buildCmd.Flags().StringSliceVar(&buildPaths, "paths", nil, "Additional paths to scan (supplements config)")
	buildCmd.Flags().StringVar(&buildOutputPath, "output", "", "Custom output path for index file")

	// Query command flags
	queryCmd.Flags().StringVar(&queryControl, "control", "", "Query by SOC2 control code (e.g., CC6.8)")
	queryCmd.Flags().StringVar(&queryAttribute, "attribute", "", "Query by security attribute (e.g., encryption)")
	queryCmd.Flags().StringVar(&queryResourceType, "resource-type", "", "Query by Terraform resource type (e.g., aws_kms_key)")
	queryCmd.Flags().StringVar(&queryEnvironment, "environment", "", "Query by environment (e.g., prod, staging)")
	queryCmd.Flags().StringVarP(&queryOutputFormat, "output", "o", "summary", "Output format (summary|json|csv|markdown)")

	// Clear command flags
	clearCmd.Flags().BoolVarP(&clearYes, "yes", "y", false, "Skip confirmation prompt")
}

// runBuild builds the Terraform security index
func runBuild(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	cfg := getConfigFromContext(ctx)
	log := getLoggerFromContext(ctx)

	log.Info("Building Terraform security index...")

	// Determine scan paths
	scanPaths := cfg.Evidence.Tools.Terraform.ScanPaths
	if len(buildPaths) > 0 {
		scanPaths = append(scanPaths, buildPaths...)
	}

	if len(scanPaths) == 0 {
		return fmt.Errorf("no scan paths configured - add paths via config or --paths flag")
	}

	log.Debug("Scan paths", logger.Field{Key: "paths", Value: scanPaths})

	// Determine index path
	indexPath := buildOutputPath
	if indexPath == "" {
		cacheDir := cfg.Storage.CacheDir
		if cacheDir == "" {
			cacheDir = ".cache"
		}
		indexPath = filepath.Join(cacheDir, "terraform", terraform.IndexFileName)
	}

	// Create indexer
	indexer := terraform.NewSecurityAttributeIndexer(cfg, log)

	// Build index with timing
	start := time.Now()
	persistedIndex, err := indexer.BuildAndPersistIndex(ctx)
	if err != nil {
		return fmt.Errorf("failed to build index: %w", err)
	}
	duration := time.Since(start)

	// Display results
	fmt.Printf("\n✅ Index built successfully\n\n")
	fmt.Printf("📊 Index Statistics:\n")
	fmt.Printf("  Total Resources: %d\n", persistedIndex.Metadata.TotalResources)
	fmt.Printf("  Total Files:     %d\n", persistedIndex.Metadata.TotalFiles)
	fmt.Printf("  Build Duration:  %v\n", duration.Round(time.Millisecond))
	fmt.Printf("  Index Path:      %s\n", indexPath)
	fmt.Printf("  Index Version:   %s\n", persistedIndex.Version)

	if persistedIndex.Statistics != nil {
		fmt.Printf("\n📈 Compliance Coverage:\n")
		fmt.Printf("  Overall: %.1f%%\n", persistedIndex.Statistics.ComplianceCoverage*100)

		if len(persistedIndex.Statistics.ControlCoverage) > 0 {
			fmt.Printf("\n🎯 Top Control Coverage:\n")
			count := 0
			for control, stats := range persistedIndex.Statistics.ControlCoverage {
				if count >= 5 {
					break
				}
				fmt.Printf("  %s: %d resources (%.1f%% compliant)\n",
					control, stats.TotalResources, stats.ComplianceRate*100)
				count++
			}
		}
	}

	return nil
}

// runStatus shows index status
func runStatus(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	cfg := getConfigFromContext(ctx)
	log := getLoggerFromContext(ctx)

	// Determine index path
	cacheDir := cfg.Storage.CacheDir
	if cacheDir == "" {
		cacheDir = ".cache"
	}
	indexPath := filepath.Join(cacheDir, "terraform", terraform.IndexFileName)

	storage := terraform.NewIndexStorage(indexPath, log)

	// Get index info
	info, err := storage.GetIndexInfo()
	if err != nil {
		return fmt.Errorf("failed to get index info: %w", err)
	}

	// Display status
	if exists, ok := info["exists"].(bool); !ok || !exists {
		fmt.Println("❌ No index found")
		fmt.Printf("\nTo build the index, run:\n  grctool terraform-index build\n")
		return nil
	}

	fmt.Println("✅ Index exists")
	fmt.Printf("\n📁 Index Information:\n")
	fmt.Printf("  Path:    %v\n", info["path"])
	fmt.Printf("  Size:    %v bytes\n", info["size"])
	fmt.Printf("  Age:     %v\n", info["age"])
	fmt.Printf("  Version: %v\n", info["version"])

	if indexedAt, ok := info["indexed_at"].(time.Time); ok {
		fmt.Printf("  Created: %s\n", indexedAt.Format(time.RFC3339))
	}

	if corrupted, ok := info["corrupted"].(bool); ok && corrupted {
		fmt.Printf("\n⚠️  Index appears corrupted: %v\n", info["error"])
		fmt.Printf("\nTo rebuild the index, run:\n  grctool terraform-index build --force\n")
		return nil
	}

	fmt.Printf("\n📊 Index Statistics:\n")
	fmt.Printf("  Total Resources: %v\n", info["total_resources"])
	fmt.Printf("  Total Files:     %v\n", info["total_files"])
	fmt.Printf("  Scan Duration:   %vms\n", info["scan_duration_ms"])

	if coverage, ok := info["compliance_coverage"].(float64); ok {
		fmt.Printf("\n📈 Compliance Coverage: %.1f%%\n", coverage*100)
	}

	return nil
}

// runQuery queries the index
func runQuery(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	cfg := getConfigFromContext(ctx)
	log := getLoggerFromContext(ctx)

	// Validate that at least one query parameter is provided
	if queryControl == "" && queryAttribute == "" && queryResourceType == "" && queryEnvironment == "" {
		return fmt.Errorf("at least one query parameter required (--control, --attribute, --resource-type, or --environment)")
	}

	// Determine index path
	cacheDir := cfg.Storage.CacheDir
	if cacheDir == "" {
		cacheDir = ".cache"
	}
	indexPath := filepath.Join(cacheDir, "terraform", terraform.IndexFileName)

	storage := terraform.NewIndexStorage(indexPath, log)

	// Load index
	index, err := storage.LoadIndex()
	if err != nil {
		return fmt.Errorf("failed to load index: %w\nRun 'grctool terraform-index build' to create the index", err)
	}

	// Create query interface
	query := terraform.NewIndexQuery(index)

	// Execute query
	var result *terraform.QueryResult

	switch {
	case queryControl != "":
		result = query.ByControl(queryControl)
	case queryAttribute != "":
		result = query.ByAttribute(queryAttribute)
	case queryResourceType != "":
		result = query.ByResourceType(queryResourceType)
	case queryEnvironment != "":
		result = query.ByEnvironment(queryEnvironment)
	}

	// Format output
	switch queryOutputFormat {
	case "json":
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))

	case "csv":
		fmt.Println("ResourceID,ResourceType,Environment,RiskLevel,ComplianceStatus")
		for _, res := range result.Resources {
			fmt.Printf("%s,%s,%s,%s,%s\n",
				res.ResourceID, res.ResourceType, res.Environment, res.RiskLevel, res.ComplianceStatus)
		}

	case "markdown":
		fmt.Printf("# Query Results\n\n")
		fmt.Printf("**Query Time:** %.2fms\n", float64(result.QueryTime.Microseconds())/1000.0)
		fmt.Printf("**Results:** %d resources\n\n", result.Count)

		for i, res := range result.Resources {
			if i >= 20 {
				fmt.Printf("\n... and %d more (use --output json for all results)\n", result.Count-20)
				break
			}
			fmt.Printf("## %s\n", res.ResourceID)
			fmt.Printf("- **Type:** %s\n", res.ResourceType)
			fmt.Printf("- **Environment:** %s\n", res.Environment)
			fmt.Printf("- **Risk Level:** %s\n", res.RiskLevel)
			fmt.Printf("- **Compliance:** %s\n", res.ComplianceStatus)
			fmt.Printf("- **File:** %s:%s\n\n", res.FilePath, res.LineRange)
		}

	case "summary":
		fallthrough
	default:
		fmt.Printf("Query completed in %.2fms\n", float64(result.QueryTime.Microseconds())/1000.0)
		fmt.Printf("Found %d resources\n\n", result.Count)

		if result.Count == 0 {
			fmt.Println("No resources found matching the query criteria.")
			return nil
		}

		// Show aggregations
		riskAgg := result.AggregateByRisk()
		if len(riskAgg) > 0 {
			fmt.Println("Risk Distribution:")
			for _, risk := range []string{"high", "medium", "low"} {
				if count, exists := riskAgg[risk]; exists {
					fmt.Printf("  %s: %d\n", cases.Title(language.English).String(risk), count)
				}
			}
			fmt.Println()
		}

		complianceAgg := result.AggregateByComplianceStatus()
		if len(complianceAgg) > 0 {
			fmt.Println("Compliance Status:")
			for status, count := range complianceAgg {
				fmt.Printf("  %s: %d\n", status, count)
			}
			fmt.Println()
		}

		// Show sample resources
		fmt.Println("Sample Resources (first 10):")
		for i, res := range result.Resources {
			if i >= 10 {
				fmt.Printf("... and %d more (use --output json for full list)\n", result.Count-10)
				break
			}
			fmt.Printf("  - %s (%s) - %s:%s\n",
				res.ResourceID, res.ResourceType, res.FilePath, res.LineRange)
		}
	}

	return nil
}

// runClear clears the index
func runClear(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	cfg := getConfigFromContext(ctx)
	log := getLoggerFromContext(ctx)

	// Determine index path
	cacheDir := cfg.Storage.CacheDir
	if cacheDir == "" {
		cacheDir = ".cache"
	}
	indexPath := filepath.Join(cacheDir, "terraform", terraform.IndexFileName)

	storage := terraform.NewIndexStorage(indexPath, log)

	// Check if index exists
	if !storage.IndexExists() {
		fmt.Println("No index found to clear.")
		return nil
	}

	// Confirm deletion
	if !clearYes {
		fmt.Printf("Are you sure you want to delete the index at %s? [y/N] ", indexPath)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Delete index
	if err := storage.DeleteIndex(); err != nil {
		return fmt.Errorf("failed to delete index: %w", err)
	}

	fmt.Println("✅ Index cleared successfully")
	return nil
}

// runValidate validates the index
func runValidate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	cfg := getConfigFromContext(ctx)
	log := getLoggerFromContext(ctx)

	// Determine index path
	cacheDir := cfg.Storage.CacheDir
	if cacheDir == "" {
		cacheDir = ".cache"
	}
	indexPath := filepath.Join(cacheDir, "terraform", terraform.IndexFileName)

	storage := terraform.NewIndexStorage(indexPath, log)

	// Check if index exists
	if !storage.IndexExists() {
		fmt.Println("❌ No index found")
		return fmt.Errorf("index does not exist")
	}

	fmt.Println("Validating index...")

	// Try to load index
	index, err := storage.LoadIndex()
	if err != nil {
		fmt.Printf("❌ Index validation failed: %v\n", err)
		return err
	}

	// Perform validation checks
	fmt.Println("✅ Index loaded successfully")

	// Check version
	fmt.Printf("  Version: %s", index.Version)
	if index.Version == terraform.IndexVersion {
		fmt.Println(" ✓")
	} else {
		fmt.Printf(" ⚠️  (expected %s)\n", terraform.IndexVersion)
	}

	// Check data integrity
	fmt.Print("  Data integrity: ")
	if index.Index == nil {
		fmt.Println("❌ No index data")
		return fmt.Errorf("index has no data")
	}
	fmt.Println("✓")

	// Check metadata consistency
	fmt.Print("  Metadata consistency: ")
	if index.Metadata.TotalResources != len(index.Index.IndexedResources) {
		fmt.Printf("⚠️  Mismatch (metadata: %d, actual: %d)\n",
			index.Metadata.TotalResources, len(index.Index.IndexedResources))
	} else {
		fmt.Println("✓")
	}

	// Check source files
	fmt.Printf("  Source files tracked: %d ✓\n", len(index.SourceFiles))

	// Check statistics
	fmt.Print("  Statistics: ")
	if index.Statistics == nil {
		fmt.Println("⚠️  Missing")
	} else {
		fmt.Println("✓")
	}

	fmt.Println("\n✅ Index validation passed")
	return nil
}

// Helper functions to get config and logger from context
func getConfigFromContext(ctx context.Context) *config.Config {
	cfg := appcontext.GetConfig(ctx)
	if cfg == nil {
		// Return default config if not in context
		return &config.Config{}
	}
	return cfg.(*config.Config)
}

func getLoggerFromContext(ctx context.Context) logger.Logger {
	log := appcontext.GetLogger(ctx)
	if log == nil {
		// Return console logger if not in context
		consoleLog, _ := logger.New(&logger.Config{
			Level:  logger.InfoLevel,
			Format: "text",
			Output: "stdout",
		})
		return consoleLog
	}
	return log.(logger.Logger)
}
