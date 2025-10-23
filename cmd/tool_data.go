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

// terraformEnhancedCmd handles the enhanced terraform-scanner tool
var terraformEnhancedCmd = &cobra.Command{
	Use:   "terraform-scanner",
	Short: "Enhanced Terraform configuration scanner with filtering and pattern matching",
	Long: `Enhanced Terraform configuration scanner that provides:
- Resource type filtering (--resource-types aws_iam_role,aws_security_group)
- Pattern searching with regex support (--pattern "encrypt|kms")
- Control hint matching (--control-hint CC6.8)
- Bounded snippets with file paths and line numbers
- Cached results for improved performance`,
	RunE: runTerraformEnhanced,
}

// githubSearcherCmd handles the github-searcher tool
var githubSearcherCmd = &cobra.Command{
	Use:   "github-searcher",
	Short: "Search GitHub repositories for security evidence",
	Long: `Search GitHub repositories for security-related evidence including:
- Multiple search types: commit, workflow, issue, pr, all
- Date filtering with --since parameter
- Result limiting with --limit parameter
- Cached API results for efficiency
- Relevant excerpts with URLs and metadata`,
	RunE: runGitHubSearcher,
}

// docsReaderCmd handles the docs-reader tool
var docsReaderCmd = &cobra.Command{
	Use:   "docs-reader",
	Short: "Search and analyze documentation files",
	Long: `Search and analyze markdown/text files in documentation directories:
- Pattern matching with glob support (--pattern "*.md")
- Keyword-based relevance scoring
- Section extraction from markdown files
- Relevance-scored content output`,
	RunE: runDocsReader,
}

func init() {
	// Add data source tools to the tool command
	toolCmd.AddCommand(terraformEnhancedCmd)
	toolCmd.AddCommand(githubSearcherCmd)
	toolCmd.AddCommand(docsReaderCmd)

	// Terraform Enhanced flags
	terraformEnhancedCmd.Flags().StringSlice("resource-types", nil, "comma-separated list of resource types to filter (e.g., aws_iam_role,aws_security_group)")
	terraformEnhancedCmd.Flags().String("pattern", "", "regex pattern to search for in resources (e.g., 'encrypt|kms')")
	terraformEnhancedCmd.Flags().String("control-hint", "", "security control hint to match against (e.g., CC6.8)")
	terraformEnhancedCmd.Flags().String("output-format", "json", "output format (json, csv, markdown)")
	terraformEnhancedCmd.Flags().Bool("use-cache", true, "use cached scan results when available")
	terraformEnhancedCmd.Flags().Int("max-results", 100, "maximum number of results to return")

	// GitHub Searcher flags
	githubSearcherCmd.Flags().String("query", "", "search query for GitHub content")
	githubSearcherCmd.Flags().String("search-type", "all", "search type (commit, workflow, issue, pr, all)")
	githubSearcherCmd.Flags().String("since", "", "date filter - only results since this date (YYYY-MM-DD)")
	githubSearcherCmd.Flags().Int("limit", 50, "maximum number of results to return")
	githubSearcherCmd.Flags().Bool("use-cache", true, "use cached API results when available")
	githubSearcherCmd.Flags().StringSlice("labels", nil, "filter by GitHub labels")
	githubSearcherCmd.MarkFlagRequired("query")

	// Docs Reader flags
	docsReaderCmd.Flags().String("pattern", "*.md", "glob pattern for files to search")
	docsReaderCmd.Flags().String("query", "", "search query/keywords")
	docsReaderCmd.Flags().String("docs-path", "docs/", "path to documentation directory")
	docsReaderCmd.Flags().Float64("min-relevance", 0.1, "minimum relevance score to include results")
	docsReaderCmd.Flags().Int("max-results", 20, "maximum number of results to return")
	docsReaderCmd.Flags().Bool("extract-sections", true, "extract relevant sections from markdown")
	docsReaderCmd.MarkFlagRequired("query")
}

// runTerraformEnhanced executes the enhanced terraform-scanner tool
func runTerraformEnhanced(cmd *cobra.Command, args []string) error {
	// Get parameters from flags
	params := make(map[string]interface{})

	if resourceTypes, _ := cmd.Flags().GetStringSlice("resource-types"); len(resourceTypes) > 0 {
		// Convert []string to []interface{} for tool compatibility
		rtInterface := make([]interface{}, len(resourceTypes))
		for i, rt := range resourceTypes {
			rtInterface[i] = rt
		}
		params["resource_types"] = rtInterface
	}

	if pattern, _ := cmd.Flags().GetString("pattern"); pattern != "" {
		params["pattern"] = pattern
	}

	if controlHint, _ := cmd.Flags().GetString("control-hint"); controlHint != "" {
		params["control_hint"] = controlHint
	}

	if outputFormat, _ := cmd.Flags().GetString("output-format"); outputFormat != "" {
		params["output_format"] = outputFormat
	}

	if useCache, _ := cmd.Flags().GetBool("use-cache"); cmd.Flags().Changed("use-cache") {
		params["use_cache"] = useCache
	}

	if maxResults, _ := cmd.Flags().GetInt("max-results"); maxResults != 0 {
		params["max_results"] = maxResults
	}

	// Define validation rules
	validationRules := map[string]tools.ValidationRule{
		"resource_types": {
			Required: false,
			Type:     "array",
		},
		"pattern":      OptionalStringRule,
		"control_hint": OptionalStringRule,
		"output_format": {
			Required:      false,
			Type:          "string",
			AllowedValues: []string{"json", "csv", "markdown"},
		},
		"use_cache": BoolRule,
		"max_results": {
			Required:  false,
			Type:      "int",
			MinLength: 1,
			MaxLength: 1000,
		},
	}

	// Execute tool with validation
	return ValidateAndExecuteTool(cmd, "terraform_analyzer", params, validationRules)
}

// runGitHubSearcher executes the github-searcher tool
func runGitHubSearcher(cmd *cobra.Command, args []string) error {
	// Get parameters from flags
	params := make(map[string]interface{})

	if query, _ := cmd.Flags().GetString("query"); query != "" {
		params["query"] = query
	}

	if searchType, _ := cmd.Flags().GetString("search-type"); searchType != "" {
		params["search_type"] = searchType
	}

	if since, _ := cmd.Flags().GetString("since"); since != "" {
		params["since"] = since
	}

	if limit, _ := cmd.Flags().GetInt("limit"); limit != 0 {
		params["limit"] = limit
	}

	if useCache, _ := cmd.Flags().GetBool("use-cache"); cmd.Flags().Changed("use-cache") {
		params["use_cache"] = useCache
	}

	if labels, _ := cmd.Flags().GetStringSlice("labels"); len(labels) > 0 {
		// Convert []string to []interface{} for tool compatibility
		labelsInterface := make([]interface{}, len(labels))
		for i, label := range labels {
			labelsInterface[i] = label
		}
		params["labels"] = labelsInterface
	}

	// Define validation rules
	validationRules := map[string]tools.ValidationRule{
		"query": StringRule,
		"search_type": {
			Required:      false,
			Type:          "string",
			AllowedValues: []string{"commit", "workflow", "issue", "pr", "all"},
		},
		"since": {
			Required: false,
			Type:     "string",
			Pattern:  `^\d{4}-\d{2}-\d{2}$`,
		},
		"limit": {
			Required:  false,
			Type:      "int",
			MinLength: 1,
			MaxLength: 500,
		},
		"use_cache": BoolRule,
		"labels": {
			Required: false,
			Type:     "array",
		},
	}

	// Execute tool with validation
	return ValidateAndExecuteTool(cmd, "github-enhanced", params, validationRules)
}

// runDocsReader executes the docs-reader tool
func runDocsReader(cmd *cobra.Command, args []string) error {
	// Get parameters from flags
	params := make(map[string]interface{})

	if pattern, _ := cmd.Flags().GetString("pattern"); pattern != "" {
		params["pattern"] = pattern
	}

	if query, _ := cmd.Flags().GetString("query"); query != "" {
		params["query"] = query
	}

	if docsPath, _ := cmd.Flags().GetString("docs-path"); docsPath != "" {
		params["docs_path"] = docsPath
	}

	if minRelevance, _ := cmd.Flags().GetFloat64("min-relevance"); minRelevance != 0 {
		params["min_relevance"] = minRelevance
	}

	if maxResults, _ := cmd.Flags().GetInt("max-results"); maxResults != 0 {
		params["max_results"] = maxResults
	}

	if extractSections, _ := cmd.Flags().GetBool("extract-sections"); cmd.Flags().Changed("extract-sections") {
		params["extract_sections"] = extractSections
	}

	// Define validation rules
	validationRules := map[string]tools.ValidationRule{
		"pattern": OptionalStringRule,
		"query":   StringRule,
		"docs_path": {
			Required:   false,
			Type:       "path",
			PathSafety: true,
		},
		"min_relevance": {
			Required:  false,
			Type:      "float",
			MinLength: 0.0,
			MaxLength: 1.0,
		},
		"max_results": {
			Required:  false,
			Type:      "int",
			MinLength: 1,
			MaxLength: 100,
		},
		"extract_sections": BoolRule,
	}

	// Execute tool with validation
	return ValidateAndExecuteTool(cmd, "docs-reader", params, validationRules)
}
