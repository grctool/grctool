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
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	evidenceTaskRef     string
	evidenceTitle       string
	evidenceContent     string
	evidenceContentFile string
	evidenceFormat      string
	evidenceSourceType  string
	evidenceSourceLoc   string
	evidenceControls    []string
	evidenceSummary     string
	evidenceReasoning   string
	evidenceStatus      string
	evidenceUpdatePlan  bool
)

var toolEvidenceWriterCmd = &cobra.Command{
	Use:   "evidence-writer",
	Short: "Write evidence files with window management and collection planning",
	Long: `Write evidence files organized by collection windows with automatic
collection plan management and completeness tracking.

The evidence writer tool creates evidence files in the appropriate collection 
window directory based on the task's collection interval. It automatically
manages collection plans that combine strategy documentation with evidence
inventory tracking.

Examples:
  # Write markdown evidence from command line
  grctool tool evidence-writer --task-ref ET1 --title "Access Control Policy" \
    --content "# Policy Document..." --format markdown

  # Write evidence from file with detailed metadata
  grctool tool evidence-writer --task-ref ET-101 --title "Terraform Config" \
    --file terraform_analysis.md --source-type terraform \
    --source-location "infrastructure/iam/*.tf" --controls AC1,AC2

  # Write CSV evidence data
  grctool tool evidence-writer --task-ref ET10 --title "User Access Report" \
    --content "user,role,last_login..." --format csv \
    --status partial --reasoning "Quarterly data collection in progress"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Read content from file if specified
		content := evidenceContent
		if evidenceContentFile != "" {
			if content != "" {
				return fmt.Errorf("cannot specify both --content and --file")
			}

			fileContent, err := readContentFromFile(evidenceContentFile)
			if err != nil {
				return fmt.Errorf("failed to read content from file %s: %w", evidenceContentFile, err)
			}
			content = fileContent
		}

		// If no content specified, read from stdin
		if content == "" {
			cmd.Println("Enter evidence content (press Ctrl+D when finished):")
			stdinContent, err := readContentFromStdin()
			if err != nil {
				return fmt.Errorf("failed to read content from stdin: %w", err)
			}
			content = stdinContent
		}

		if content == "" {
			return fmt.Errorf("evidence content is required (use --content, --file, or provide via stdin)")
		}

		// Build parameters
		params := map[string]interface{}{
			"task_ref":        evidenceTaskRef,
			"title":           evidenceTitle,
			"content":         content,
			"format":          evidenceFormat,
			"source_type":     evidenceSourceType,
			"source_location": evidenceSourceLoc,
			"controls":        evidenceControls,
			"summary":         evidenceSummary,
			"reasoning":       evidenceReasoning,
			"status":          evidenceStatus,
			"update_plan":     evidenceUpdatePlan,
		}

		return ValidateAndExecuteTool(cmd, "evidence-writer", params, nil)
	},
}

func init() {
	toolCmd.AddCommand(toolEvidenceWriterCmd)

	// Required flags
	toolEvidenceWriterCmd.Flags().StringVar(&evidenceTaskRef, "task-ref", "",
		"Evidence task reference (e.g., ET1, ET-101, 327992)")
	toolEvidenceWriterCmd.Flags().StringVar(&evidenceTitle, "title", "",
		"Evidence document title")

	// Content input options
	toolEvidenceWriterCmd.Flags().StringVar(&evidenceContent, "content", "",
		"Evidence content (or use --file or stdin)")
	toolEvidenceWriterCmd.Flags().StringVar(&evidenceContentFile, "file", "",
		"Read evidence content from file")

	// Format and metadata
	toolEvidenceWriterCmd.Flags().StringVar(&evidenceFormat, "format", "markdown",
		"Output format: markdown or csv")
	toolEvidenceWriterCmd.Flags().StringVar(&evidenceSourceType, "source-type", "",
		"Type of evidence source: terraform, github, google_docs, manual, screenshot, api, database")
	toolEvidenceWriterCmd.Flags().StringVar(&evidenceSourceLoc, "source-location", "",
		"Source location (file path, URL, or description)")
	toolEvidenceWriterCmd.Flags().StringSliceVar(&evidenceControls, "controls", []string{},
		"Control references this evidence addresses (comma-separated)")

	// Collection plan options
	toolEvidenceWriterCmd.Flags().StringVar(&evidenceSummary, "summary", "",
		"Brief summary for the collection plan")
	toolEvidenceWriterCmd.Flags().StringVar(&evidenceReasoning, "reasoning", "",
		"Why this evidence is relevant and what it demonstrates")
	toolEvidenceWriterCmd.Flags().StringVar(&evidenceStatus, "status", "complete",
		"Evidence collection status: complete, partial, pending")
	toolEvidenceWriterCmd.Flags().BoolVar(&evidenceUpdatePlan, "update-plan", true,
		"Whether to update the collection plan")

	// Mark required flags
	toolEvidenceWriterCmd.MarkFlagRequired("task-ref")
	toolEvidenceWriterCmd.MarkFlagRequired("title")

	// Add usage examples
	toolEvidenceWriterCmd.Example = `  # Write markdown evidence with metadata
  grctool tool evidence-writer \
    --task-ref ET1 \
    --title "Access Control Policy Document" \
    --file policy_document.md \
    --source-type google_docs \
    --source-location "Security Policies/Access Control Policy v2.1" \
    --controls AC1,AC2 \
    --summary "Management-approved access control policy" \
    --reasoning "Demonstrates formal policy approval and comprehensive access controls"

  # Write CSV evidence data
  grctool tool evidence-writer \
    --task-ref ET15 \
    --title "Terminated Users Report Q1 2025" \
    --content "user_id,name,termination_date,access_revoked
123,John Doe,2025-01-15,2025-01-15
456,Jane Smith,2025-02-28,2025-02-28" \
    --format csv \
    --source-type database \
    --controls AC2 \
    --status complete

  # Write evidence from stdin (interactive)
  grctool tool evidence-writer --task-ref ET20 --title "Incident Response Log"
  # Then paste content and press Ctrl+D`
}

// readContentFromFile reads content from a specified file
func readContentFromFile(filename string) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// readContentFromStdin reads content from stdin until EOF
func readContentFromStdin() (string, error) {
	scanner := bufio.NewScanner(os.Stdin)
	var lines []string

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return strings.Join(lines, "\n"), nil
}
