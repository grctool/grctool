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

// evidenceTaskDetailsCmd handles the evidence-task-details tool
var evidenceTaskDetailsCmd = &cobra.Command{
	Use:   "evidence-task-details",
	Short: "Retrieve detailed evidence task information",
	Long: `Retrieves comprehensive details about evidence tasks including requirements, 
status, relationships, and metadata. Accepts task references in various formats 
(ET-101, ET101, 328001).`,
	RunE: runEvidenceTaskDetails,
}

// evidenceRelationshipsCmd handles the evidence-relationships tool
var evidenceRelationshipsCmd = &cobra.Command{
	Use:   "evidence-relationships",
	Short: "Map evidence task relationships",
	Long: `Maps relationships between evidence tasks, controls, and policies with 
configurable depth analysis. Provides comprehensive relationship graphs and 
cross-reference mappings.`,
	RunE: runEvidenceRelationships,
}

// promptAssemblerCmd handles the prompt-assembler tool
var promptAssemblerCmd = &cobra.Command{
	Use:   "prompt-assembler",
	Short: "Generate comprehensive evidence collection prompts",
	Long: `Generates comprehensive AI prompts for evidence collection with full context,
related controls, policies, and examples. Supports multiple context levels and 
output formats.`,
	RunE: runPromptAssembler,
}

// policySummaryGeneratorCmd handles the policy-summary-generator tool
var policySummaryGeneratorCmd = &cobra.Command{
	Use:   "policy-summary-generator",
	Short: "Generate focused policy summaries for evidence tasks",
	Long: `Generates focused policy summaries in the context of specific evidence tasks 
using template-based approach (prompt as data pattern). Creates structured summaries 
that highlight policy requirements relevant to evidence collection.`,
	RunE: runPolicySummaryGenerator,
}

// controlSummaryGeneratorCmd handles the control-summary-generator tool
var controlSummaryGeneratorCmd = &cobra.Command{
	Use:   "control-summary-generator",
	Short: "Generate focused control summaries for evidence tasks",
	Long: `Generates focused control summaries in the context of specific evidence tasks 
using template-based approach (prompt as data pattern). Creates structured summaries 
that highlight control requirements relevant to evidence collection.`,
	RunE: runControlSummaryGenerator,
}

// evidenceTaskListCmd handles the evidence-task-list tool
var evidenceTaskListCmd = &cobra.Command{
	Use:   "evidence-task-list",
	Short: "List evidence tasks with filtering capabilities",
	Long: `List evidence collection tasks with optional filtering by status, category, 
priority, framework, and other criteria. Returns structured JSON data suitable 
for automation and programmatic access.`,
	RunE: runEvidenceTaskList,
}

func init() {
	// Add evidence tools to the tool command
	toolCmd.AddCommand(evidenceTaskDetailsCmd)
	toolCmd.AddCommand(evidenceTaskListCmd)
	toolCmd.AddCommand(evidenceRelationshipsCmd)
	toolCmd.AddCommand(promptAssemblerCmd)
	toolCmd.AddCommand(policySummaryGeneratorCmd)
	toolCmd.AddCommand(controlSummaryGeneratorCmd)

	// Evidence task details flags
	evidenceTaskDetailsCmd.Flags().String("task-ref", "", "task reference (ET-101, ET101, or numeric ID)")
	evidenceTaskDetailsCmd.MarkFlagRequired("task-ref")
	evidenceTaskDetailsCmd.RegisterFlagCompletionFunc("task-ref", completeTaskRefs)

	// Evidence task list flags
	evidenceTaskListCmd.Flags().StringSlice("status", []string{}, "filter by status (pending, completed, overdue)")
	evidenceTaskListCmd.Flags().String("framework", "", "filter by framework (soc2, iso27001, etc)")
	evidenceTaskListCmd.Flags().StringSlice("priority", []string{}, "filter by priority (high, medium, low)")
	evidenceTaskListCmd.Flags().StringSlice("category", []string{}, "filter by category (Infrastructure, Personnel, Process, Compliance, Monitoring, Data)")
	evidenceTaskListCmd.Flags().String("assignee", "", "filter by assignee name")
	evidenceTaskListCmd.Flags().Bool("overdue", false, "show only overdue tasks")
	evidenceTaskListCmd.Flags().Bool("due-soon", false, "show tasks due within 7 days")
	evidenceTaskListCmd.Flags().StringSlice("aec-status", []string{}, "filter by AEC status (enabled, disabled, na)")
	evidenceTaskListCmd.Flags().StringSlice("collection-type", []string{}, "filter by collection type (Manual, Automated, Hybrid)")
	evidenceTaskListCmd.Flags().Bool("sensitive", false, "show only sensitive data tasks")
	evidenceTaskListCmd.Flags().StringSlice("complexity", []string{}, "filter by complexity level (Simple, Moderate, Complex)")
	evidenceTaskListCmd.Flags().Int("limit", 0, "maximum number of tasks to return (0 = no limit)")

	// Evidence relationships flags
	evidenceRelationshipsCmd.Flags().String("task-ref", "", "task reference (ET-101, ET101, or numeric ID)")
	evidenceRelationshipsCmd.Flags().Int("depth", 2, "analysis depth (1=direct, 2=extended, 3=full)")
	evidenceRelationshipsCmd.Flags().Bool("include-policies", true, "include policy relationships")
	evidenceRelationshipsCmd.Flags().Bool("include-controls", true, "include control relationships")
	evidenceRelationshipsCmd.MarkFlagRequired("task-ref")
	evidenceRelationshipsCmd.RegisterFlagCompletionFunc("task-ref", completeTaskRefs)

	// Prompt assembler flags
	promptAssemblerCmd.Flags().String("task-ref", "", "task reference (ET-101, ET101, or numeric ID)")
	promptAssemblerCmd.Flags().String("context-level", "standard", "context level (minimal, standard, comprehensive)")
	promptAssemblerCmd.Flags().Bool("include-examples", true, "include example evidence")
	promptAssemblerCmd.Flags().String("output-format", "markdown", "output format (markdown, csv, json)")
	promptAssemblerCmd.Flags().Bool("save-to-file", true, "save prompt to file")
	promptAssemblerCmd.MarkFlagRequired("task-ref")

	// Policy summary generator flags
	policySummaryGeneratorCmd.Flags().String("task-ref", "", "task reference (ET-101, ET101, or numeric ID)")
	policySummaryGeneratorCmd.Flags().String("policy-id", "", "policy ID (POL-001, 94641, etc.)")
	policySummaryGeneratorCmd.Flags().String("output-format", "markdown", "output format (markdown, json)")
	policySummaryGeneratorCmd.Flags().Bool("save-to-file", true, "save summary to file")
	policySummaryGeneratorCmd.MarkFlagRequired("task-ref")
	policySummaryGeneratorCmd.MarkFlagRequired("policy-id")

	// Control summary generator flags
	controlSummaryGeneratorCmd.Flags().String("task-ref", "", "task reference (ET-101, ET101, or numeric ID)")
	controlSummaryGeneratorCmd.Flags().String("control-id", "", "control ID (CC1.1, 11057785, etc.)")
	controlSummaryGeneratorCmd.Flags().String("output-format", "markdown", "output format (markdown, json)")
	controlSummaryGeneratorCmd.Flags().Bool("save-to-file", true, "save summary to file")
	controlSummaryGeneratorCmd.MarkFlagRequired("task-ref")
	controlSummaryGeneratorCmd.MarkFlagRequired("control-id")

}

// runEvidenceTaskDetails executes the evidence-task-details tool
func runEvidenceTaskDetails(cmd *cobra.Command, args []string) error {
	// Get parameters from flags
	params := make(map[string]interface{})

	if taskRef, _ := cmd.Flags().GetString("task-ref"); taskRef != "" {
		params["task_ref"] = taskRef
	}

	// Define validation rules
	validationRules := map[string]tools.ValidationRule{
		"task_ref": TaskRefRule,
	}

	// Execute tool with validation
	return ValidateAndExecuteTool(cmd, "evidence-task-details", params, validationRules)
}

// runEvidenceRelationships executes the evidence-relationships tool
func runEvidenceRelationships(cmd *cobra.Command, args []string) error {
	// Get parameters from flags
	params := make(map[string]interface{})

	if taskRef, _ := cmd.Flags().GetString("task-ref"); taskRef != "" {
		params["task_ref"] = taskRef
	}

	if depth, _ := cmd.Flags().GetInt("depth"); depth != 0 {
		params["depth"] = depth
	}

	if includePolicies, _ := cmd.Flags().GetBool("include-policies"); cmd.Flags().Changed("include-policies") {
		params["include_policies"] = includePolicies
	}

	if includeControls, _ := cmd.Flags().GetBool("include-controls"); cmd.Flags().Changed("include-controls") {
		params["include_controls"] = includeControls
	}

	// Define validation rules
	validationRules := map[string]tools.ValidationRule{
		"task_ref": TaskRefRule,
		"depth": {
			Required: false,
			Type:     "int",
		},
		"include_policies": BoolRule,
		"include_controls": BoolRule,
	}

	// Execute tool with validation
	return ValidateAndExecuteTool(cmd, "evidence-relationships", params, validationRules)
}

// runPromptAssembler executes the prompt-assembler tool
func runPromptAssembler(cmd *cobra.Command, args []string) error {
	// Get parameters from flags
	params := make(map[string]interface{})

	if taskRef, _ := cmd.Flags().GetString("task-ref"); taskRef != "" {
		params["task_ref"] = taskRef
	}

	if contextLevel, _ := cmd.Flags().GetString("context-level"); contextLevel != "" {
		params["context_level"] = contextLevel
	}

	if includeExamples, _ := cmd.Flags().GetBool("include-examples"); cmd.Flags().Changed("include-examples") {
		params["include_examples"] = includeExamples
	}

	if outputFormat, _ := cmd.Flags().GetString("output-format"); outputFormat != "" {
		params["output_format"] = outputFormat
	}

	if saveToFile, _ := cmd.Flags().GetBool("save-to-file"); cmd.Flags().Changed("save-to-file") {
		params["save_to_file"] = saveToFile
	}

	// Define validation rules
	validationRules := map[string]tools.ValidationRule{
		"task_ref": TaskRefRule,
		"context_level": {
			Required:      false,
			Type:          "string",
			AllowedValues: []string{"minimal", "standard", "comprehensive"},
		},
		"include_examples": BoolRule,
		"output_format": {
			Required:      false,
			Type:          "string",
			AllowedValues: []string{"markdown", "csv", "json"},
		},
		"save_to_file": BoolRule,
	}

	// Execute tool with validation
	return ValidateAndExecuteTool(cmd, "prompt-assembler", params, validationRules)
}

// runPolicySummaryGenerator executes the policy-summary-generator tool
func runPolicySummaryGenerator(cmd *cobra.Command, args []string) error {
	// Get parameters from flags
	params := make(map[string]interface{})

	if taskRef, _ := cmd.Flags().GetString("task-ref"); taskRef != "" {
		params["task_ref"] = taskRef
	}

	if policyID, _ := cmd.Flags().GetString("policy-id"); policyID != "" {
		params["policy_id"] = policyID
	}

	if outputFormat, _ := cmd.Flags().GetString("output-format"); outputFormat != "" {
		params["output_format"] = outputFormat
	}

	if saveToFile, _ := cmd.Flags().GetBool("save-to-file"); cmd.Flags().Changed("save-to-file") {
		params["save_to_file"] = saveToFile
	}

	// Define validation rules
	validationRules := map[string]tools.ValidationRule{
		"task_ref":  TaskRefRule,
		"policy_id": StringRule,
		"output_format": {
			Required:      false,
			Type:          "string",
			AllowedValues: []string{"markdown", "json"},
		},
		"save_to_file": BoolRule,
	}

	// Execute tool with validation
	return ValidateAndExecuteTool(cmd, "policy-summary-generator", params, validationRules)
}

// runControlSummaryGenerator executes the control-summary-generator tool
func runControlSummaryGenerator(cmd *cobra.Command, args []string) error {
	// Get parameters from flags
	params := make(map[string]interface{})

	if taskRef, _ := cmd.Flags().GetString("task-ref"); taskRef != "" {
		params["task_ref"] = taskRef
	}

	if controlID, _ := cmd.Flags().GetString("control-id"); controlID != "" {
		params["control_id"] = controlID
	}

	if outputFormat, _ := cmd.Flags().GetString("output-format"); outputFormat != "" {
		params["output_format"] = outputFormat
	}

	if saveToFile, _ := cmd.Flags().GetBool("save-to-file"); cmd.Flags().Changed("save-to-file") {
		params["save_to_file"] = saveToFile
	}

	// Define validation rules
	validationRules := map[string]tools.ValidationRule{
		"task_ref":   TaskRefRule,
		"control_id": StringRule,
		"output_format": {
			Required:      false,
			Type:          "string",
			AllowedValues: []string{"markdown", "json"},
		},
		"save_to_file": BoolRule,
	}

	// Execute tool with validation
	return ValidateAndExecuteTool(cmd, "control-summary-generator", params, validationRules)
}

// runEvidenceTaskList executes the evidence-task-list tool
func runEvidenceTaskList(cmd *cobra.Command, args []string) error {
	// Get parameters from flags
	params := make(map[string]interface{})

	if status, _ := cmd.Flags().GetStringSlice("status"); len(status) > 0 {
		// Convert []string to []interface{} for tool parameter processing
		statusInterface := make([]interface{}, len(status))
		for i, v := range status {
			statusInterface[i] = v
		}
		params["status"] = statusInterface
	}

	if framework, _ := cmd.Flags().GetString("framework"); framework != "" {
		params["framework"] = framework
	}

	if priority, _ := cmd.Flags().GetStringSlice("priority"); len(priority) > 0 {
		// Convert []string to []interface{} for tool parameter processing
		priorityInterface := make([]interface{}, len(priority))
		for i, v := range priority {
			priorityInterface[i] = v
		}
		params["priority"] = priorityInterface
	}

	if category, _ := cmd.Flags().GetStringSlice("category"); len(category) > 0 {
		// Convert []string to []interface{} for tool parameter processing
		categoryInterface := make([]interface{}, len(category))
		for i, v := range category {
			categoryInterface[i] = v
		}
		params["category"] = categoryInterface
	}

	if assignee, _ := cmd.Flags().GetString("assignee"); assignee != "" {
		params["assignee"] = assignee
	}

	if overdue, _ := cmd.Flags().GetBool("overdue"); cmd.Flags().Changed("overdue") {
		params["overdue"] = overdue
	}

	if dueSoon, _ := cmd.Flags().GetBool("due-soon"); cmd.Flags().Changed("due-soon") {
		params["due_soon"] = dueSoon
	}

	if aecStatus, _ := cmd.Flags().GetStringSlice("aec-status"); len(aecStatus) > 0 {
		// Convert []string to []interface{} for tool parameter processing
		aecStatusInterface := make([]interface{}, len(aecStatus))
		for i, v := range aecStatus {
			aecStatusInterface[i] = v
		}
		params["aec_status"] = aecStatusInterface
	}

	if collectionType, _ := cmd.Flags().GetStringSlice("collection-type"); len(collectionType) > 0 {
		// Convert []string to []interface{} for tool parameter processing
		collectionTypeInterface := make([]interface{}, len(collectionType))
		for i, v := range collectionType {
			collectionTypeInterface[i] = v
		}
		params["collection_type"] = collectionTypeInterface
	}

	if sensitive, _ := cmd.Flags().GetBool("sensitive"); cmd.Flags().Changed("sensitive") {
		params["sensitive"] = sensitive
	}

	if complexity, _ := cmd.Flags().GetStringSlice("complexity"); len(complexity) > 0 {
		// Convert []string to []interface{} for tool parameter processing
		complexityInterface := make([]interface{}, len(complexity))
		for i, v := range complexity {
			complexityInterface[i] = v
		}
		params["complexity"] = complexityInterface
	}

	if limit, _ := cmd.Flags().GetInt("limit"); limit > 0 {
		params["limit"] = limit
	}

	// Define validation rules
	validationRules := map[string]tools.ValidationRule{
		"status": {
			Required: false,
			Type:     "[]string",
		},
		"framework": OptionalStringRule,
		"priority": {
			Required: false,
			Type:     "[]string",
		},
		"category": {
			Required: false,
			Type:     "[]string",
		},
		"assignee": OptionalStringRule,
		"overdue":  BoolRule,
		"due_soon": BoolRule,
		"aec_status": {
			Required: false,
			Type:     "[]string",
		},
		"collection_type": {
			Required: false,
			Type:     "[]string",
		},
		"sensitive": BoolRule,
		"complexity": {
			Required: false,
			Type:     "[]string",
		},
		"limit": {
			Required: false,
			Type:     "int",
		},
	}

	// Execute tool with validation
	return ValidateAndExecuteTool(cmd, "evidence-task-list", params, validationRules)
}
