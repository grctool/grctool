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

// nameGeneratorCmd handles the name-generator tool
var nameGeneratorCmd = &cobra.Command{
	Use:   "name-generator",
	Short: "Generate concise, filesystem-friendly names for documents",
	Long: `Generates concise, filesystem-friendly names for evidence tasks, controls, and policies
using Claude AI. Names are designed to be under 40 characters, use underscores for word
separation, and capture the essence of the document for easy filesystem navigation.

Supports both single document and batch processing modes.`,
	RunE: runNameGenerator,
}

func init() {
	// Add name generator tool to the tool command
	toolCmd.AddCommand(nameGeneratorCmd)

	// Name generator flags
	nameGeneratorCmd.Flags().String("document-type", "", "document type: evidence, control, or policy")
	nameGeneratorCmd.Flags().String("reference-id", "", "document reference ID (ET-101, AC-1, POL-001, etc.)")
	nameGeneratorCmd.Flags().Bool("batch-mode", false, "process all documents of the specified type")

	// Mark document-type as required
	nameGeneratorCmd.MarkFlagRequired("document-type")
}

// runNameGenerator executes the name-generator tool
func runNameGenerator(cmd *cobra.Command, args []string) error {
	// Get parameters from flags
	params := make(map[string]interface{})

	if docType, _ := cmd.Flags().GetString("document-type"); docType != "" {
		params["document_type"] = docType
	}

	if refID, _ := cmd.Flags().GetString("reference-id"); refID != "" {
		params["reference_id"] = refID
	}

	if batchMode, _ := cmd.Flags().GetBool("batch-mode"); cmd.Flags().Changed("batch-mode") {
		params["batch_mode"] = batchMode
	}

	// Define validation rules
	validationRules := map[string]tools.ValidationRule{
		"document_type": {
			Required:      true,
			Type:          "string",
			AllowedValues: []string{"evidence", "control", "policy"},
		},
		"reference_id": {
			Required: false, // Not required in batch mode
			Type:     "string",
		},
		"batch_mode": BoolRule,
	}

	// Execute tool with validation
	return ValidateAndExecuteTool(cmd, "name-generator", params, validationRules)
}
