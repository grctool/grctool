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
	"strings"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/formatters"
	"github.com/grctool/grctool/internal/interpolation"
	"github.com/grctool/grctool/internal/storage"
	"github.com/spf13/cobra"
)

// policyCmd represents the policy command
var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Manage and view policies",
	Long: `Commands for managing and viewing security policies from Tugboat Logic.

This command group provides various operations for policies including:
- Viewing policies in markdown format
- Listing available policies
- Searching policies by framework or status`,
}

// policyViewCmd represents the policy view command
var policyViewCmd = &cobra.Command{
	Use:   "view [policy-id]",
	Short: "View a policy in markdown format",
	Long: `Display a policy document in markdown format with full content and metadata.

The policy is displayed with:
- Policy header with ID, version, and last updated information
- Full policy document content
- Comprehensive metadata including associations and usage statistics
- Assignees, reviewers, and tags

Examples:
  # View a specific policy by ID
  grctool policy view POL-001
  
  # Save policy to markdown file
  grctool policy view POL-001 --output policy-001.md
  
  # View policy with summary format
  grctool policy view POL-001 --summary`,
	Args: cobra.ExactArgs(1),
	RunE: runPolicyView,
}

// policyListCmd represents the policy list command
var policyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available policies",
	Long: `List all available policies with basic information.

This command shows policies with their ID, name, framework, status, and last updated date.

Examples:
  # List all policies
  grctool policy list
  
  # List policies for a specific framework
  grctool policy list --framework SOC2
  
  # List policies with a specific status
  grctool policy list --status active`,
	RunE: runPolicyList,
}

func init() {
	rootCmd.AddCommand(policyCmd)
	policyCmd.AddCommand(policyViewCmd)
	policyCmd.AddCommand(policyListCmd)

	// Policy view flags
	policyViewCmd.Flags().StringP("output", "o", "", "Output file path (optional)")
	policyViewCmd.Flags().Bool("summary", false, "Show summary format instead of full document")
	policyViewCmd.Flags().Bool("metadata-only", false, "Show only metadata without policy content")

	// Policy list flags
	policyListCmd.Flags().String("framework", "", "Filter by framework (SOC2, ISO27001, etc.)")
	policyListCmd.Flags().String("status", "", "Filter by status (active, draft, deprecated, etc.)")
	policyListCmd.Flags().Bool("details", false, "Show detailed information for each policy")
}

func runPolicyView(cmd *cobra.Command, args []string) error {
	policyID := args[0]

	// Get flags
	outputFile, _ := cmd.Flags().GetString("output")
	summary, _ := cmd.Flags().GetBool("summary")
	metadataOnly, _ := cmd.Flags().GetBool("metadata-only")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize unified storage with full config to support custom paths
	storage, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Get policy
	policy, err := storage.GetPolicy(policyID)
	if err != nil {
		return fmt.Errorf("policy not found: %s (hint: run 'grctool sync --policies' to fetch latest policies)", policyID)
	}

	// Initialize formatter with interpolation if enabled
	var formatter *formatters.PolicyFormatter
	if cfg.Interpolation.Enabled {
		config := interpolation.InterpolatorConfig{
			Variables:         cfg.Interpolation.GetFlatVariables(),
			Enabled:           true,
			OnMissingVariable: interpolation.MissingVariableIgnore,
		}
		interpolator := interpolation.NewStandardInterpolator(config)
		formatter = formatters.NewPolicyFormatterWithInterpolation(interpolator)
	} else {
		formatter = formatters.NewPolicyFormatter()
	}

	// Generate markdown based on format requested
	var markdown string
	if summary {
		markdown = formatter.ToSummaryMarkdown(policy)
	} else if metadataOnly {
		// Create a copy with empty content for metadata-only view
		metadataPolicy := *policy
		metadataPolicy.Content = ""
		metadataPolicy.Description = ""
		markdown = formatter.ToMarkdown(&metadataPolicy)
	} else {
		markdown = formatter.ToMarkdown(policy)
	}

	// Output markdown
	if outputFile != "" {
		// Ensure output directory exists
		if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		// Write to file
		if err := os.WriteFile(outputFile, []byte(markdown), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "✅ Policy exported to: %s\n", outputFile)
	} else {
		// Print to stdout
		fmt.Fprint(cmd.OutOrStdout(), markdown)
	}

	return nil
}

func runPolicyList(cmd *cobra.Command, args []string) error {
	// Get flags
	framework, _ := cmd.Flags().GetString("framework")
	status, _ := cmd.Flags().GetString("status")
	details, _ := cmd.Flags().GetBool("details")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize unified storage with full config to support custom paths
	storage, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Get all policies
	policies, err := storage.GetAllPolicies()
	if err != nil {
		return fmt.Errorf("failed to get policies: %w (hint: run 'grctool sync --policies' to fetch policies)", err)
	}

	// Filter policies
	var filteredPolicies []domain.Policy
	for _, policy := range policies {
		// Apply framework filter
		if framework != "" && !strings.EqualFold(policy.Framework, framework) {
			continue
		}

		// Apply status filter
		if status != "" && !strings.EqualFold(policy.Status, status) {
			continue
		}

		filteredPolicies = append(filteredPolicies, policy)
	}

	if len(filteredPolicies) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No policies found matching the criteria.")
		return nil
	}

	// Display policies
	if details {
		fmt.Fprintf(cmd.OutOrStdout(), "Found %d policies:\n\n", len(filteredPolicies))

		// Initialize formatter with interpolation if enabled
		var formatter *formatters.PolicyFormatter
		if cfg.Interpolation.Enabled {
			config := interpolation.InterpolatorConfig{
				Variables:         cfg.Interpolation.GetFlatVariables(),
				Enabled:           true,
				OnMissingVariable: interpolation.MissingVariableIgnore,
			}
			interpolator := interpolation.NewStandardInterpolator(config)
			formatter = formatters.NewPolicyFormatterWithInterpolation(interpolator)
		} else {
			formatter = formatters.NewPolicyFormatter()
		}
		for i, policy := range filteredPolicies {
			summary := formatter.ToSummaryMarkdown(&policy)
			fmt.Fprint(cmd.OutOrStdout(), summary)

			if i < len(filteredPolicies)-1 {
				fmt.Fprintln(cmd.OutOrStdout(), "\n---")
			}
		}
	} else {
		// Simple table format
		fmt.Fprintf(cmd.OutOrStdout(), "Found %d policies:\n\n", len(filteredPolicies))
		fmt.Fprintf(cmd.OutOrStdout(), "%-15s %-40s %-12s %-12s %s\n", "ID", "Name", "Framework", "Status", "Last Updated")
		fmt.Fprintf(cmd.OutOrStdout(), "%-15s %-40s %-12s %-12s %s\n",
			strings.Repeat("-", 15),
			strings.Repeat("-", 40),
			strings.Repeat("-", 12),
			strings.Repeat("-", 12),
			strings.Repeat("-", 12))

		for _, policy := range filteredPolicies {
			name := policy.Name
			if len(name) > 40 {
				name = name[:37] + "..."
			}

			fmt.Fprintf(cmd.OutOrStdout(), "%-15s %-40s %-12s %-12s %s\n",
				policy.ID,
				name,
				policy.Framework,
				policy.Status,
				policy.UpdatedAt.Format("2006-01-02"))
		}
	}

	return nil
}
