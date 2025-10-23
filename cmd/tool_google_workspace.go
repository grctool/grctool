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
	"github.com/grctool/grctool/internal/tools"
	"github.com/spf13/cobra"
)

// googleWorkspaceCmd handles the google-workspace tool
var googleWorkspaceCmd = &cobra.Command{
	Use:   "google-workspace",
	Short: "Extract evidence from Google Workspace documents including Drive, Docs, Sheets, and Forms",
	Long: `Extract evidence from Google Workspace documents for SOC2 audit evidence including:
- Google Drive folder contents and permissions
- Google Docs text content and revision history
- Google Sheets data and metadata
- Google Forms responses and configuration
- Document metadata (created, modified, editors)
- Sharing and permission settings
- Revision history and change tracking`,
	RunE: runGoogleWorkspace,
}

func init() {
	// Add Google Workspace tool to the tool command
	toolCmd.AddCommand(googleWorkspaceCmd)

	// Google Workspace direct access flags
	googleWorkspaceCmd.Flags().String("document-id", "", "Google document ID (from URL or share link)")
	googleWorkspaceCmd.Flags().String("document-type", "drive", "Type of Google document: drive, docs, sheets, forms")
	googleWorkspaceCmd.Flags().Bool("include-metadata", true, "Include document metadata (created, modified, editors)")
	googleWorkspaceCmd.Flags().Bool("include-revisions", false, "Include revision history")
	googleWorkspaceCmd.Flags().String("sheet-range", "", "For sheets: range to extract (e.g., 'A1:D10', 'Sheet1!A:Z')")
	googleWorkspaceCmd.Flags().String("search-query", "", "Search query for Drive folder content")
	googleWorkspaceCmd.Flags().Int("max-results", 20, "Maximum number of results to return")
	googleWorkspaceCmd.Flags().String("credentials-path", "", "Path to Google service account credentials JSON file")
	googleWorkspaceCmd.MarkFlagRequired("document-id")
}

// runGoogleWorkspace executes the google-workspace tool
func runGoogleWorkspace(cmd *cobra.Command, args []string) error {
	// Get parameters from flags
	params := make(map[string]interface{})

	if documentID, _ := cmd.Flags().GetString("document-id"); documentID != "" {
		params["document_id"] = documentID
	}

	if documentType, _ := cmd.Flags().GetString("document-type"); documentType != "" {
		params["document_type"] = documentType
	}

	if credentialsPath, _ := cmd.Flags().GetString("credentials-path"); credentialsPath != "" {
		params["credentials_path"] = credentialsPath
	}

	// Build extraction rules from flags
	extractionRules := make(map[string]interface{})

	if includeMetadata, _ := cmd.Flags().GetBool("include-metadata"); cmd.Flags().Changed("include-metadata") {
		extractionRules["include_metadata"] = includeMetadata
	}

	if includeRevisions, _ := cmd.Flags().GetBool("include-revisions"); cmd.Flags().Changed("include-revisions") {
		extractionRules["include_revisions"] = includeRevisions
	}

	if sheetRange, _ := cmd.Flags().GetString("sheet-range"); sheetRange != "" {
		extractionRules["sheet_range"] = sheetRange
	}

	if searchQuery, _ := cmd.Flags().GetString("search-query"); searchQuery != "" {
		extractionRules["search_query"] = searchQuery
	}

	if maxResults, _ := cmd.Flags().GetInt("max-results"); cmd.Flags().Changed("max-results") {
		extractionRules["max_results"] = maxResults
	}

	if len(extractionRules) > 0 {
		params["extraction_rules"] = extractionRules
	}

	// Define validation rules
	validationRules := map[string]tools.ValidationRule{
		"document_id": {
			Required:  true,
			Type:      "string",
			MinLength: 1,
			MaxLength: 200,
		},
		"document_type": {
			Required:      false,
			Type:          "string",
			AllowedValues: []string{"drive", "docs", "sheets", "forms"},
		},
		"credentials_path": OptionalPathRule,
		"extraction_rules": {
			Required: false,
			Type:     "object",
		},
	}

	// Execute tool with validation
	return ValidateAndExecuteTool(cmd, "google-workspace", params, validationRules)
}
