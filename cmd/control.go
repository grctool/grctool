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
	"strconv"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/formatters"
	"github.com/grctool/grctool/internal/interpolation"
	"github.com/grctool/grctool/internal/markdown"
	"github.com/grctool/grctool/internal/storage"
	"github.com/spf13/cobra"
)

// controlCmd represents the control command
var controlCmd = &cobra.Command{
	Use:   "control",
	Short: "Manage and view controls",
	Long: `Commands for managing and viewing security controls from Tugboat Logic.

This command group provides various operations for controls including:
- Viewing controls in markdown format
- Listing available controls
- Searching controls by framework, category, or status`,
}

// controlViewCmd represents the control view command
var controlViewCmd = &cobra.Command{
	Use:   "view [control-id]",
	Short: "View a control in markdown format",
	Long: `Display a control document in markdown format with full content and metadata.

The control is displayed with:
- Control header with ID, category, framework, and status information
- Full control description, help, and guidance content
- Comprehensive metadata including associations and evidence metrics
- Assignees, audit projects, and framework codes
- Implementation status and testing information

Examples:
  # View a specific control by ID
  grctool control view 778771
  
  # Save control to markdown file
  grctool control view 778771 --output control-778771.md
  
  # View control with summary format
  grctool control view 778771 --summary`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		controlID := args[0]

		// Parse control ID to integer
		id, err := strconv.Atoi(controlID)
		if err != nil {
			return fmt.Errorf("invalid control ID '%s': must be a number", controlID)
		}

		// Load configuration
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Initialize storage
		storage, err := storage.NewStorage(cfg.Storage)
		if err != nil {
			return fmt.Errorf("failed to initialize storage: %w", err)
		}

		// Get control from storage
		control, err := storage.GetControl(strconv.Itoa(id))
		if err != nil {
			return fmt.Errorf("failed to get control %d: %w", id, err)
		}

		if control == nil {
			return fmt.Errorf("control %d not found", id)
		}

		// Create formatter with interpolation
		interpolatorConfig := interpolation.InterpolatorConfig{
			Variables:         cfg.Interpolation.GetFlatVariables(),
			Enabled:           cfg.Interpolation.Enabled,
			OnMissingVariable: interpolation.MissingVariableIgnore,
		}
		interpolator := interpolation.NewStandardInterpolator(interpolatorConfig)
		formatter := formatters.NewControlFormatterWithInterpolation(interpolator)

		// Get flags
		outputFile, _ := cmd.Flags().GetString("output")
		summaryMode, _ := cmd.Flags().GetBool("summary")
		metadataOnly, _ := cmd.Flags().GetBool("metadata-only")
		showDetails, _ := cmd.Flags().GetBool("details")

		var content string
		if summaryMode {
			content = formatter.ToSummaryMarkdown(control)
		} else if metadataOnly {
			// Create a simplified metadata-only view
			content = createControlMetadataView(control, formatter)
		} else {
			content = formatter.ToMarkdown(control)
		}

		// Add additional details if requested
		if showDetails && !summaryMode {
			content += "\n\n" + createControlDetailsSection(control)
		}

		// Output to file or stdout
		if outputFile != "" {
			if err := os.WriteFile(outputFile, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to write to file %s: %w", outputFile, err)
			}
			cmd.Printf("Control %d saved to %s\n", id, outputFile)
		} else {
			cmd.Print(content)
		}

		return nil
	},
}

// controlListCmd represents the control list command
var controlListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all controls",
	Long: `List all controls with optional filtering.

This command displays a formatted list of all controls with basic information
including ID, name, category, framework, and status.

Examples:
  # List all controls
  grctool control list
  
  # Filter by framework
  grctool control list --framework SOC2
  
  # Filter by status
  grctool control list --status implemented
  
  # Filter by category
  grctool control list --category "Access Control"
  
  # Show summary format
  grctool control list --summary
  
  # Save list to file
  grctool control list --output controls-list.md`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Initialize storage
		storage, err := storage.NewStorage(cfg.Storage)
		if err != nil {
			return fmt.Errorf("failed to initialize storage: %w", err)
		}

		// Get all controls
		controls, err := storage.GetAllControls()
		if err != nil {
			return fmt.Errorf("failed to get controls: %w", err)
		}

		// Get filters
		frameworkFilter, _ := cmd.Flags().GetString("framework")
		statusFilter, _ := cmd.Flags().GetString("status")
		categoryFilter, _ := cmd.Flags().GetString("category")

		// Convert to pointer slice and apply filters
		controlPtrs := make([]*domain.Control, len(controls))
		for i := range controls {
			controlPtrs[i] = &controls[i]
		}
		filteredControls := filterControls(controlPtrs, frameworkFilter, statusFilter, categoryFilter)

		// Get flags
		outputFile, _ := cmd.Flags().GetString("output")
		summaryMode, _ := cmd.Flags().GetBool("summary")

		// Create formatter with interpolation
		interpolatorConfig := interpolation.InterpolatorConfig{
			Variables:         cfg.Interpolation.GetFlatVariables(),
			Enabled:           cfg.Interpolation.Enabled,
			OnMissingVariable: interpolation.MissingVariableIgnore,
		}
		interpolator := interpolation.NewStandardInterpolator(interpolatorConfig)
		formatter := formatters.NewControlFormatterWithInterpolation(interpolator)

		// Format output
		var content string
		if summaryMode {
			content = formatControlsListSummary(filteredControls, formatter)
		} else {
			content = formatControlsList(filteredControls, formatter)
		}

		// Output to file or stdout
		if outputFile != "" {
			if err := os.WriteFile(outputFile, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to write to file %s: %w", outputFile, err)
			}
			cmd.Printf("Controls list saved to %s\n", outputFile)
		} else {
			cmd.Print(content)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(controlCmd)
	controlCmd.AddCommand(controlViewCmd)
	controlCmd.AddCommand(controlListCmd)

	// Flags for control view command
	controlViewCmd.Flags().StringP("output", "o", "", "Output file path")
	controlViewCmd.Flags().BoolP("summary", "s", false, "Show summary format")
	controlViewCmd.Flags().Bool("metadata-only", false, "Show only metadata")
	controlViewCmd.Flags().Bool("details", false, "Show additional details")

	// Flags for control list command
	controlListCmd.Flags().StringP("output", "o", "", "Output file path")
	controlListCmd.Flags().BoolP("summary", "s", false, "Show summary format")
	controlListCmd.Flags().String("framework", "", "Filter by framework")
	controlListCmd.Flags().String("status", "", "Filter by status")
	controlListCmd.Flags().String("category", "", "Filter by category")
}

// filterControls applies filters to the controls list
func filterControls(controls []*domain.Control, framework, status, category string) []*domain.Control {
	var filtered []*domain.Control

	for _, control := range controls {
		// Apply framework filter
		if framework != "" && !strings.EqualFold(control.Framework, framework) {
			continue
		}

		// Apply status filter
		if status != "" && !strings.EqualFold(control.Status, status) {
			continue
		}

		// Apply category filter
		if category != "" && !strings.Contains(strings.ToLower(control.Category), strings.ToLower(category)) {
			continue
		}

		filtered = append(filtered, control)
	}

	return filtered
}

// formatControlsList creates a formatted list of controls
func formatControlsList(controls []*domain.Control, formatter *formatters.ControlFormatter) string {
	var content strings.Builder

	content.WriteString("# Controls List\n\n")
	content.WriteString(fmt.Sprintf("Found %d controls\n\n", len(controls)))

	if len(controls) == 0 {
		content.WriteString("No controls found matching the specified criteria.\n")
		return content.String()
	}

	// Create table
	content.WriteString("| ID | Name | Category | Framework | Status | Implemented |\n")
	content.WriteString("|----|------|----------|-----------|--------|--------------|\n")

	for _, control := range controls {
		implemented := "No"
		if control.ImplementedDate != nil && !control.ImplementedDate.IsZero() {
			implemented = control.ImplementedDate.Format("2006-01-02")
		}

		content.WriteString(fmt.Sprintf("| %d | %s | %s | %s | %s | %s |\n",
			control.ID,
			truncateString(control.Name, 40),
			truncateString(control.Category, 20),
			truncateString(control.Framework, 15),
			control.Status,
			implemented,
		))
	}

	content.WriteString(fmt.Sprintf("\n*Generated on %s*\n", time.Now().Format("2006-01-02 15:04:05 MST")))

	// Format the markdown content before returning
	mdFormatter := markdown.NewFormatter(markdown.DefaultConfig())
	return mdFormatter.FormatDocument(content.String())
}

// formatControlsListSummary creates a summary list of controls
func formatControlsListSummary(controls []*domain.Control, formatter *formatters.ControlFormatter) string {
	var content strings.Builder

	content.WriteString("# Controls Summary\n\n")
	content.WriteString(fmt.Sprintf("Found %d controls\n\n", len(controls)))

	if len(controls) == 0 {
		content.WriteString("No controls found matching the specified criteria.\n")
		return content.String()
	}

	// Group by status
	statusGroups := make(map[string][]*domain.Control)
	for _, control := range controls {
		statusGroups[control.Status] = append(statusGroups[control.Status], control)
	}

	// Display each status group
	for status, groupControls := range statusGroups {
		content.WriteString(fmt.Sprintf("## %s (%d controls)\n\n", strings.ToTitle(status), len(groupControls)))

		for _, control := range groupControls {
			content.WriteString(fmt.Sprintf("- **%d** - %s (%s)\n",
				control.ID,
				truncateString(control.Name, 60),
				control.Category,
			))
		}
		content.WriteString("\n")
	}

	content.WriteString(fmt.Sprintf("*Generated on %s*\n", time.Now().Format("2006-01-02 15:04:05 MST")))

	// Format the markdown content before returning
	mdFormatter := markdown.NewFormatter(markdown.DefaultConfig())
	return mdFormatter.FormatDocument(content.String())
}

// createControlMetadataView creates a metadata-only view of a control
func createControlMetadataView(control *domain.Control, formatter *formatters.ControlFormatter) string {
	var content strings.Builder

	content.WriteString(fmt.Sprintf("# Control %d Metadata\n\n", control.ID))

	// Basic info
	content.WriteString("## Basic Information\n\n")
	content.WriteString("| Field | Value |\n")
	content.WriteString("|-------|-------|\n")
	content.WriteString(fmt.Sprintf("| **Control ID** | %d |\n", control.ID))
	content.WriteString(fmt.Sprintf("| **Name** | %s |\n", control.Name))
	content.WriteString(fmt.Sprintf("| **Category** | %s |\n", control.Category))
	content.WriteString(fmt.Sprintf("| **Framework** | %s |\n", control.Framework))
	content.WriteString(fmt.Sprintf("| **Status** | %s |\n", control.Status))

	if control.RiskLevel != "" {
		content.WriteString(fmt.Sprintf("| **Risk Level** | %s |\n", control.RiskLevel))
	}

	if control.ImplementedDate != nil && !control.ImplementedDate.IsZero() {
		content.WriteString(fmt.Sprintf("| **Implemented** | %s |\n", control.ImplementedDate.Format("2006-01-02")))
	}

	if control.TestedDate != nil && !control.TestedDate.IsZero() {
		content.WriteString(fmt.Sprintf("| **Last Tested** | %s |\n", control.TestedDate.Format("2006-01-02")))
	}

	// Associations
	if control.Associations != nil && (control.Associations.Policies > 0 ||
		control.Associations.Procedures > 0 || control.Associations.Evidence > 0) {
		content.WriteString("\n## Associations\n\n")
		content.WriteString("| Type | Count |\n")
		content.WriteString("|------|-------|\n")
		if control.Associations.Policies > 0 {
			content.WriteString(fmt.Sprintf("| **Policies** | %d |\n", control.Associations.Policies))
		}
		if control.Associations.Procedures > 0 {
			content.WriteString(fmt.Sprintf("| **Procedures** | %d |\n", control.Associations.Procedures))
		}
		if control.Associations.Evidence > 0 {
			content.WriteString(fmt.Sprintf("| **Evidence** | %d |\n", control.Associations.Evidence))
		}
	}

	// Format the markdown content before returning
	mdFormatter := markdown.NewFormatter(markdown.DefaultConfig())
	return mdFormatter.FormatDocument(content.String())
}

// createControlDetailsSection creates an additional details section
func createControlDetailsSection(control *domain.Control) string {
	var content strings.Builder

	content.WriteString("## Additional Details\n\n")

	// Technical details
	content.WriteString("### Technical Information\n\n")
	content.WriteString("| Field | Value |\n")
	content.WriteString("|-------|-------|\n")
	content.WriteString(fmt.Sprintf("| **Master Control ID** | %d |\n", control.MasterControlID))
	content.WriteString(fmt.Sprintf("| **Master Version** | %d |\n", control.MasterVersionNum))
	content.WriteString(fmt.Sprintf("| **Organization ID** | %d |\n", control.OrgID))
	content.WriteString(fmt.Sprintf("| **Org Scope ID** | %d |\n", control.OrgScopeID))

	if control.OrgScope != nil {
		content.WriteString(fmt.Sprintf("| **Org Scope Name** | %s |\n", control.OrgScope.Name))
	}

	// Incident information
	if control.OpenIncidentCount > 0 {
		content.WriteString(fmt.Sprintf("| **Open Incidents** | %d |\n", control.OpenIncidentCount))
	}

	if control.RecommendedEvidenceCount > 0 {
		content.WriteString(fmt.Sprintf("| **Recommended Evidence** | %d |\n", control.RecommendedEvidenceCount))
	}

	// Format the markdown content before returning
	mdFormatter := markdown.NewFormatter(markdown.DefaultConfig())
	return mdFormatter.FormatDocument(content.String())
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
