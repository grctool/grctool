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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/grctool/grctool/internal/config"
	"github.com/spf13/cobra"
)

// completeTaskRefs provides completion for evidence task references (ET-001, ET-101, etc.)
func completeTaskRefs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Try multiple approaches to find evidence task files
	var files []string
	var err error

	// Approach 1: Try to load config
	cfg, configErr := config.Load()
	if configErr == nil && cfg.Storage.DataDir != "" {
		// Build path to evidence tasks
		taskDir := filepath.Join(cfg.Storage.DataDir, "docs", "evidence_tasks", "json")
		files, err = filepath.Glob(filepath.Join(taskDir, "ET-*.json"))

		// Try alternative location without "json" subdirectory
		if err != nil || len(files) == 0 {
			taskDir = filepath.Join(cfg.Storage.DataDir, "docs", "evidence_tasks")
			files, _ = filepath.Glob(filepath.Join(taskDir, "ET-*.json"))
		}
	}

	// Approach 2: If config loading failed or no files found, try common locations
	if len(files) == 0 {
		commonPaths := []string{
			"../docs/evidence_tasks/json",               // Typical synced location
			"docs/evidence_tasks/json",                  // From project root
			"test/sample_data/docs/evidence_tasks/json", // Test data
			"./data/docs/evidence_tasks/json",           // Default local data dir
		}

		for _, basePath := range commonPaths {
			pattern := filepath.Join(basePath, "ET-*.json")
			found, err := filepath.Glob(pattern)
			if err == nil && len(found) > 0 {
				files = found
				break
			}
		}
	}

	// No files found - return empty completion
	if len(files) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var task struct {
			Reference   string `json:"reference"`
			ReferenceID string `json:"reference_id"` // Alternative field name used in synced data
			Title       string `json:"title"`
			Name        string `json:"name"` // Alternative field name
		}
		if err := json.Unmarshal(data, &task); err != nil {
			continue
		}

		// Use reference_id if reference is empty
		ref := task.Reference
		if ref == "" {
			ref = task.ReferenceID
		}

		if ref != "" {
			// Filter based on what user has typed so far
			if strings.HasPrefix(ref, toComplete) || toComplete == "" {
				// Use title or name for description
				desc := task.Title
				if desc == "" {
					desc = task.Name
				}
				// Format: "ET-001\tInfrastructure Security Configuration Evidence"
				completion := ref
				if desc != "" {
					completion = ref + "\t" + desc
				}
				completions = append(completions, completion)
			}
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completePolicyRefs provides completion for policy references (POL-001, POL-002, etc.)
func completePolicyRefs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg, err := config.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	policyDir := filepath.Join(cfg.Storage.DataDir, "docs", "policies", "json")
	files, err := filepath.Glob(filepath.Join(policyDir, "POL-*.json"))
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var policy struct {
			Reference string `json:"reference"`
			Title     string `json:"title"`
			Name      string `json:"name"` // Some policies might use 'name' instead of 'title'
		}
		if err := json.Unmarshal(data, &policy); err != nil {
			continue
		}

		if policy.Reference != "" {
			// Filter based on what user has typed so far
			if strings.HasPrefix(policy.Reference, toComplete) || toComplete == "" {
				title := policy.Title
				if title == "" {
					title = policy.Name
				}
				// Format: "POL-001\tInformation Security Policy"
				completion := policy.Reference
				if title != "" {
					completion = policy.Reference + "\t" + title
				}
				completions = append(completions, completion)
			}
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completeControlRefs provides completion for control references (AC-01, CC-01, etc.)
func completeControlRefs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg, err := config.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	controlDir := filepath.Join(cfg.Storage.DataDir, "docs", "controls", "json")
	files, err := filepath.Glob(filepath.Join(controlDir, "*.json"))
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var control struct {
			Reference   string `json:"reference"`
			Code        string `json:"code"` // Alternative field name
			Title       string `json:"title"`
			Description string `json:"description"` // Fallback if no title
		}
		if err := json.Unmarshal(data, &control); err != nil {
			continue
		}

		ref := control.Reference
		if ref == "" {
			ref = control.Code
		}

		if ref != "" {
			// Filter based on what user has typed so far
			if strings.HasPrefix(ref, toComplete) || toComplete == "" {
				title := control.Title
				if title == "" && len(control.Description) > 0 {
					// Use first 50 chars of description if no title
					title = control.Description
					if len(title) > 50 {
						title = title[:50] + "..."
					}
				}
				// Format: "AC-01\tAccess Control Policy and Procedures"
				completion := ref
				if title != "" {
					completion = ref + "\t" + title
				}
				completions = append(completions, completion)
			}
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completeToolNames provides completion for registered tool names
func completeToolNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// This will work after tool registry is initialized
	// For now, return common tool names as static list
	tools := []string{
		"evidence-task-list\tList evidence tasks with filtering",
		"evidence-task-details\tRetrieve detailed evidence task information",
		"terraform-scanner\tTerraform configuration scanner",
		"terraform-hcl-parser\tHCL parser with topology analysis",
		"terraform-security-analyzer\tSecurity configuration analysis",
		"terraform-query-interface\tFlexible query interface",
		"terraform-snippets\tExtract code snippets for evidence",
		"terraform-security-indexer\tFast indexed queries",
		"atmos-stack-analyzer\tMulti-environment Atmos analysis",
		"github-searcher\tSearch repositories for evidence",
		"github-permissions\tRepository access controls",
		"github-deployment-access\tDeployment environment access",
		"github-security-features\tSecurity feature configuration",
		"github-workflow-analyzer\tCI/CD workflow security",
		"github-review-analyzer\tPR review and approval analysis",
		"google-workspace\tGoogle Workspace document analysis",
		"storage-read\tSafe file read operations",
		"storage-write\tSafe file write operations",
		"name-generator\tGenerate filesystem-friendly names",
	}

	var filtered []string
	for _, tool := range tools {
		toolName := strings.Split(tool, "\t")[0]
		if strings.HasPrefix(toolName, toComplete) || toComplete == "" {
			filtered = append(filtered, tool)
		}
	}

	return filtered, cobra.ShellCompDirectiveNoFileComp
}
