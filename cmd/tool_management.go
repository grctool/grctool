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

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/tools"
	"github.com/spf13/cobra"
)

// toolMgmtCmd represents the tool-management command
var toolMgmtCmd = &cobra.Command{
	Use:   "tool-management",
	Short: "Evidence generation and management tools",
	Long:  `Suite of tools for evidence generation, validation, storage, and synchronization used by AI agents and automated workflows.`,
}

var evidenceGeneratorCmd = &cobra.Command{
	Use:   "evidence-generator",
	Short: "Generate evidence from multiple data sources",
	Long: `Generate compliance evidence using AI coordination with sub-tools (terraform, github, docs).
Outputs evidence in multiple formats with source tracking and metadata.`,
	RunE: runEvidenceGenerator,
}

var evidenceValidatorCmd = &cobra.Command{
	Use:   "evidence-validator",
	Short: "Validate evidence completeness and quality",
	Long: `Perform validation checks on evidence files, calculate completeness scores,
and provide recommendations for improvement.`,
	RunE: runEvidenceValidator,
}

var tugboatSyncCmd = &cobra.Command{
	Use:   "tugboat-sync",
	Short: "Wrapper for tugboat sync with JSON output",
	Long: `Wrap existing sync functionality with structured JSON output and
selective resource syncing capabilities.`,
	RunE: runTugboatSync,
}

var storageReadCmd = &cobra.Command{
	Use:   "storage-read",
	Short: "Safe file read operations",
	Long:  `Path-safe file reading with format auto-detection and metadata preservation.`,
	RunE:  runStorageRead,
}

var storageWriteCmd = &cobra.Command{
	Use:   "storage-write",
	Short: "Safe file write operations",
	Long:  `Path-safe file writing with format handling and directory management.`,
	RunE:  runStorageWrite,
}

var grctoolRunCmd = &cobra.Command{
	Use:   "grctool-run",
	Short: "Execute allowlisted grctool commands",
	Long:  `Meta tool for safe execution of existing grctool commands with structured output capture.`,
	RunE:  runGrctoolRun,
}

func init() {
	rootCmd.AddCommand(toolMgmtCmd)

	// Add subcommands
	toolMgmtCmd.AddCommand(evidenceGeneratorCmd)
	toolMgmtCmd.AddCommand(evidenceValidatorCmd)
	toolMgmtCmd.AddCommand(tugboatSyncCmd)
	toolMgmtCmd.AddCommand(storageReadCmd)
	toolMgmtCmd.AddCommand(storageWriteCmd)
	toolMgmtCmd.AddCommand(grctoolRunCmd)

	// Evidence generator flags
	evidenceGeneratorCmd.Flags().String("prompt-file", "", "path to prompt file")
	evidenceGeneratorCmd.Flags().String("task-ref", "", "evidence task reference (e.g., ET1, ET42)")
	evidenceGeneratorCmd.Flags().StringSlice("tools", []string{"terraform", "github", "docs"}, "sub-tools to coordinate")
	evidenceGeneratorCmd.Flags().String("format", "markdown", "output format (markdown, csv, json)")
	evidenceGeneratorCmd.Flags().String("output-dir", "", "directory to save generated evidence")

	// Evidence validator flags
	evidenceValidatorCmd.Flags().String("task-ref", "", "evidence task reference")
	evidenceValidatorCmd.Flags().String("evidence-file", "", "path to evidence file")
	evidenceValidatorCmd.Flags().String("validation-level", "standard", "validation level (basic, standard, comprehensive)")

	// Tugboat sync flags
	tugboatSyncCmd.Flags().StringSlice("resources", []string{"policies", "controls", "evidence_tasks"}, "resources to sync")
	tugboatSyncCmd.Flags().Bool("json-output", true, "return JSON output")
	tugboatSyncCmd.Flags().Bool("stats-only", false, "return only sync statistics")

	// Storage read flags
	storageReadCmd.Flags().String("path", "", "file path to read (required)")
	storageReadCmd.Flags().String("format", "", "force format detection (json, yaml, markdown, text)")
	storageReadCmd.Flags().Bool("with-metadata", false, "include metadata in response")
	storageReadCmd.MarkFlagRequired("path")

	// Storage write flags
	storageWriteCmd.Flags().String("path", "", "file path to write (required)")
	storageWriteCmd.Flags().String("content", "", "content to write (required)")
	storageWriteCmd.Flags().String("format", "", "content format (auto-detected if empty)")
	storageWriteCmd.Flags().Bool("create-dirs", true, "create parent directories if needed")
	storageWriteCmd.MarkFlagRequired("path")
	storageWriteCmd.MarkFlagRequired("content")

	// Grctool run flags
	grctoolRunCmd.Flags().String("command", "", "grctool command to execute (required)")
	grctoolRunCmd.Flags().StringSlice("args", []string{}, "command arguments")
	grctoolRunCmd.Flags().Bool("capture-output", true, "capture stdout/stderr")
	grctoolRunCmd.Flags().Int("timeout", 300, "command timeout in seconds")
	grctoolRunCmd.MarkFlagRequired("command")
}

// Helper function to initialize config, logger, and tool registry
func initializeToolManagement() (*config.Config, logger.Logger, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	consoleLoggerCfg := cfg.Logging.Loggers["console"]
	log, err := logger.New((&consoleLoggerCfg).ToLoggerConfig())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Initialize tool registry
	if err := tools.InitializeToolRegistry(cfg, log); err != nil {
		log.Warn("Failed to initialize tool registry", logger.Field{Key: "error", Value: err})
	}

	return cfg, log, nil
}

func runEvidenceGenerator(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Parse flags
	promptFile, _ := cmd.Flags().GetString("prompt-file")
	taskRef, _ := cmd.Flags().GetString("task-ref")
	toolsList, _ := cmd.Flags().GetStringSlice("tools")
	format, _ := cmd.Flags().GetString("format")
	outputDir, _ := cmd.Flags().GetString("output-dir")

	// Validate required parameters
	if promptFile == "" && taskRef == "" {
		return fmt.Errorf("either --prompt-file or --task-ref must be specified")
	}

	// Initialize tool management
	cfg, log, err := initializeToolManagement()
	if err != nil {
		return err
	}
	_ = cfg

	// Get evidence generator tool from registry
	evidenceGenerator, err := tools.GetTool("evidence-generator")
	if err != nil {
		return fmt.Errorf("evidence-generator tool not registered: %w", err)
	}

	// Build parameters
	params := map[string]interface{}{
		"format":     format,
		"tools":      toolsList,
		"output_dir": outputDir,
	}

	if promptFile != "" {
		params["prompt_file"] = promptFile
	}
	if taskRef != "" {
		params["task_ref"] = taskRef
	}

	// Execute tool
	result, source, err := evidenceGenerator.Execute(ctx, params)
	if err != nil {
		return fmt.Errorf("evidence generation failed: %w", err)
	}

	// Create response envelope
	response := map[string]interface{}{
		"success":    true,
		"result":     result,
		"source":     source,
		"format":     format,
		"tools_used": toolsList,
		"timestamp":  cmd.Context().Value("start_time"),
	}

	// Output JSON response
	responseJSON, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	cmd.Println(string(responseJSON))

	log.Info("Evidence generation completed",
		logger.Field{Key: "task_ref", Value: taskRef},
		logger.Field{Key: "format", Value: format},
		logger.Field{Key: "tools", Value: toolsList},
	)

	return nil
}

func runEvidenceValidator(cmd *cobra.Command, args []string) error {
	// Get parameters from flags
	params := make(map[string]interface{})

	if taskRef, _ := cmd.Flags().GetString("task-ref"); taskRef != "" {
		params["task_ref"] = taskRef
	}

	if evidenceFile, _ := cmd.Flags().GetString("evidence-file"); evidenceFile != "" {
		params["evidence_file"] = evidenceFile
	}

	if validationLevel, _ := cmd.Flags().GetString("validation-level"); validationLevel != "" {
		params["validation_level"] = validationLevel
	}

	// Define validation rules
	validationRules := map[string]tools.ValidationRule{
		"task_ref":      TaskRefRule,
		"evidence_file": StringRule,
		"validation_level": {
			Required:      false,
			Type:          "string",
			AllowedValues: []string{"basic", "standard", "comprehensive"},
		},
	}

	// Execute tool with validation
	return ValidateAndExecuteTool(cmd, "evidence-validator", params, validationRules)
}

func runTugboatSync(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Parse flags
	resources, _ := cmd.Flags().GetStringSlice("resources")
	jsonOutput, _ := cmd.Flags().GetBool("json-output")
	statsOnly, _ := cmd.Flags().GetBool("stats-only")

	// Initialize tool management
	cfg, log, err := initializeToolManagement()
	if err != nil {
		return err
	}
	_ = cfg

	// Get tugboat sync wrapper tool from registry
	tugboatSync, err := tools.GetTool("tugboat-sync-wrapper")
	if err != nil {
		return fmt.Errorf("tugboat-sync-wrapper tool not registered: %w", err)
	}

	// Build parameters
	params := map[string]interface{}{
		"resources":   resources,
		"json_output": jsonOutput,
		"stats_only":  statsOnly,
	}

	// Execute tool
	result, source, err := tugboatSync.Execute(ctx, params)
	if err != nil {
		return fmt.Errorf("tugboat sync failed: %w", err)
	}

	// Create response envelope
	response := map[string]interface{}{
		"success":     true,
		"sync_result": result,
		"source":      source,
		"resources":   resources,
		"timestamp":   cmd.Context().Value("start_time"),
	}

	// Output JSON response if requested, otherwise pass through result
	if jsonOutput {
		responseJSON, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal response: %w", err)
		}
		cmd.Println(string(responseJSON))
	} else {
		cmd.Println(result)
	}

	log.Info("Tugboat sync completed",
		logger.Field{Key: "resources", Value: resources},
		logger.Field{Key: "stats_only", Value: statsOnly},
	)

	return nil
}

func runStorageRead(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Parse flags
	path, _ := cmd.Flags().GetString("path")
	format, _ := cmd.Flags().GetString("format")
	withMetadata, _ := cmd.Flags().GetBool("with-metadata")

	// Initialize tool management
	cfg, log, err := initializeToolManagement()
	if err != nil {
		return err
	}
	_ = cfg

	// Get storage read tool from registry
	storageRead, err := tools.GetTool("storage-read")
	if err != nil {
		return fmt.Errorf("storage-read tool not registered: %w", err)
	}

	// Build parameters
	params := map[string]interface{}{
		"path":          path,
		"format":        format,
		"with_metadata": withMetadata,
	}

	// Execute tool
	result, source, err := storageRead.Execute(ctx, params)
	if err != nil {
		return fmt.Errorf("storage read failed: %w", err)
	}

	// Create response envelope
	response := map[string]interface{}{
		"success":   true,
		"content":   result,
		"source":    source,
		"path":      path,
		"timestamp": cmd.Context().Value("start_time"),
	}

	// Output JSON response
	responseJSON, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	cmd.Println(string(responseJSON))

	log.Info("Storage read completed",
		logger.Field{Key: "path", Value: path},
		logger.Field{Key: "format", Value: format},
	)

	return nil
}

func runStorageWrite(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Parse flags
	path, _ := cmd.Flags().GetString("path")
	content, _ := cmd.Flags().GetString("content")
	format, _ := cmd.Flags().GetString("format")
	createDirs, _ := cmd.Flags().GetBool("create-dirs")

	// Initialize tool management
	cfg, log, err := initializeToolManagement()
	if err != nil {
		return err
	}
	_ = cfg

	// Get storage write tool from registry
	storageWrite, err := tools.GetTool("storage-write")
	if err != nil {
		return fmt.Errorf("storage-write tool not registered: %w", err)
	}

	// Build parameters
	params := map[string]interface{}{
		"path":        path,
		"content":     content,
		"format":      format,
		"create_dirs": createDirs,
	}

	// Execute tool
	result, source, err := storageWrite.Execute(ctx, params)
	if err != nil {
		return fmt.Errorf("storage write failed: %w", err)
	}

	// Create response envelope
	response := map[string]interface{}{
		"success":   true,
		"result":    result,
		"source":    source,
		"path":      path,
		"timestamp": cmd.Context().Value("start_time"),
	}

	// Output JSON response
	responseJSON, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	cmd.Println(string(responseJSON))

	log.Info("Storage write completed",
		logger.Field{Key: "path", Value: path},
		logger.Field{Key: "size", Value: len(content)},
	)

	return nil
}

func runGrctoolRun(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Parse flags
	command, _ := cmd.Flags().GetString("command")
	cmdArgs, _ := cmd.Flags().GetStringSlice("args")
	captureOutput, _ := cmd.Flags().GetBool("capture-output")
	timeout, _ := cmd.Flags().GetInt("timeout")

	// Initialize tool management
	cfg, log, err := initializeToolManagement()
	if err != nil {
		return err
	}
	_ = cfg

	// Get grctool run tool from registry
	grctoolRun, err := tools.GetTool("grctool-run")
	if err != nil {
		return fmt.Errorf("grctool-run tool not registered: %w", err)
	}

	// Build parameters
	params := map[string]interface{}{
		"command":        command,
		"args":           cmdArgs,
		"capture_output": captureOutput,
		"timeout":        timeout,
	}

	// Execute tool
	result, source, err := grctoolRun.Execute(ctx, params)
	if err != nil {
		return fmt.Errorf("grctool command execution failed: %w", err)
	}

	// Create response envelope
	response := map[string]interface{}{
		"success":   true,
		"result":    result,
		"source":    source,
		"command":   command,
		"args":      cmdArgs,
		"timestamp": cmd.Context().Value("start_time"),
	}

	// Output JSON response
	responseJSON, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	cmd.Println(string(responseJSON))

	log.Info("Grctool command execution completed",
		logger.Field{Key: "command", Value: command},
		logger.Field{Key: "args", Value: cmdArgs},
	)

	return nil
}

func init() {
	// Add management tools to the tool command (flags already defined in existing init function above)
	toolCmd.AddCommand(evidenceValidatorCmd)
	toolCmd.AddCommand(evidenceGeneratorCmd)
	toolCmd.AddCommand(tugboatSyncCmd)
	toolCmd.AddCommand(storageReadCmd)
	toolCmd.AddCommand(storageWriteCmd)
	toolCmd.AddCommand(grctoolRunCmd)
}
