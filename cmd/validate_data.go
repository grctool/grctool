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

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/services/validation"
	"github.com/grctool/grctool/internal/storage"
	"github.com/spf13/cobra"
)

// validateDataCmd represents the validate-data command
var validateDataCmd = &cobra.Command{
	Use:   "validate-data",
	Short: "Validate the completeness and quality of synchronized data",
	Long: `Validate that the inbound Tugboat interface data meets quality requirements:

- Every policy, control, and evidence task should have id, name, description, and detailed content
- Policies should have substantial content (10+ lines, many with 100+ lines)  
- Every control should be linked to one or more policies
- Data integrity and relationship validation`,
	RunE: runValidateData,
}

func init() {
	rootCmd.AddCommand(validateDataCmd)
	validateDataCmd.Flags().Bool("detailed", false, "show detailed validation results")
	validateDataCmd.Flags().Bool("json", false, "output results as JSON")
	validateDataCmd.Flags().String("output", "", "output file path for detailed report")
}

func runValidateData(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Parse flags
	detailed, _ := cmd.Flags().GetBool("detailed")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	outputFile, _ := cmd.Flags().GetString("output")

	// Initialize validation service
	validationService, err := initializeValidationService()
	if err != nil {
		return err
	}

	// Build options
	options := validation.ValidationOptions{
		Detailed:   detailed,
		JSONOutput: jsonOutput,
		OutputFile: outputFile,
		Format:     "text",
	}

	if jsonOutput {
		options.Format = "json"
	}

	// Generate report
	report, err := validationService.GenerateReport(ctx, options)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Output results
	if jsonOutput {
		content, err := validationService.FormatReport(report, "json")
		if err != nil {
			return fmt.Errorf("failed to format JSON: %w", err)
		}
		cmd.Print(content)
	} else {
		// Display text summary
		displayValidationSummary(cmd, report, detailed)

		// Save detailed report if requested
		if outputFile != "" {
			if err := validationService.SaveReport(report, outputFile, "text"); err != nil {
				return fmt.Errorf("failed to save report: %w", err)
			}
			cmd.Printf("Detailed validation report written to: %s\n", outputFile)
		}
	}

	// Return error code based on validation results
	if report.Summary.Status == "fail" {
		return fmt.Errorf("data validation failed: %d critical issues found", report.Summary.CriticalIssues)
	}

	return nil
}

// Helper functions

func initializeValidationService() (validation.Service, error) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize storage with full config to support custom paths
	storage, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Initialize logger
	consoleLoggerCfg := cfg.Logging.Loggers["console"]
	log, err := logger.New((&consoleLoggerCfg).ToLoggerConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Initialize validation service
	validationService, err := validation.NewService(storage, cfg, log)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize validation service: %w", err)
	}

	return validationService, nil
}

func displayValidationSummary(cmd *cobra.Command, report *validation.ValidationReport, detailed bool) {
	// Display summary
	cmd.Printf("=== DATA VALIDATION REPORT ===\n\n")
	cmd.Printf("Overall Status: %s\n", report.Summary.Status)
	cmd.Printf("Overall Score: %.1f%%\n", report.Summary.OverallScore)
	cmd.Printf("Total Entities: %d policies, %d controls, %d evidence tasks\n",
		report.Summary.TotalPolicies, report.Summary.TotalControls, report.Summary.TotalEvidenceTasks)
	cmd.Printf("Issues: %d critical, %d warnings\n\n", report.Summary.CriticalIssues, report.Summary.Warnings)

	// Display detailed validation results
	cmd.Printf("Policy Validation:\n")
	cmd.Printf("  Content Completeness: %.1f%% (%d/%d with content)\n",
		report.Policies.ContentCompleteness,
		report.Policies.WithContent,
		report.Policies.WithContent+report.Policies.MissingContent)

	cmd.Printf("Control Validation:\n")
	cmd.Printf("  Linkage Completeness: %.1f%% (%d/%d with policy links)\n",
		report.Controls.LinkageCompleteness,
		report.Controls.WithPolicyLinks,
		report.Controls.WithPolicyLinks+report.Controls.MissingPolicyLinks)

	cmd.Printf("Evidence Validation:\n")
	cmd.Printf("  Content Completeness: %.1f%% (%d/%d with guidance)\n",
		report.Evidence.ContentCompleteness,
		report.Evidence.WithGuidance,
		report.Evidence.WithGuidance+report.Evidence.MissingGuidance)

	if detailed && len(report.Issues) > 0 {
		cmd.Printf("\nDetailed Issues:\n")
		for i, issue := range report.Issues {
			if i >= 20 { // Limit display
				cmd.Printf("... and %d more issues\n", len(report.Issues)-20)
				break
			}
			cmd.Printf("  [%s] %s: %s\n", issue.Type, issue.Category, issue.Description)
		}
	}
}
