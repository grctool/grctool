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
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/tools"
	"github.com/spf13/cobra"
)

// toolCmd represents the tool command
var toolCmd = &cobra.Command{
	Use:   "tool",
	Short: "Evidence assembly tools with standardized JSON output",
	Long: `Evidence assembly tools provide standardized access to various evidence collection
capabilities with consistent JSON output format, path safety validation, and task reference
normalization.

All tool commands support:
- Consistent JSON envelope output with metadata
- Task reference normalization (ET-101 → 328001)
- Path safety validation (prevents directory traversal)
- Automatic redaction of sensitive data (API keys, tokens, etc.)
- Correlation ID tracking for request tracing
- Duration measurement for operations`,
}

// toolListCmd lists available tools
var toolListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available evidence assembly tools",
	Long:  `List all registered evidence assembly tools with their descriptions and metadata.`,
	RunE:  runToolList,
}

// toolStatsCmd shows tool registry statistics
var toolStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show tool registry statistics",
	Long:  `Display statistics about the tool registry including total tools, categories, and usage information.`,
	RunE:  runToolStats,
}

func init() {
	rootCmd.AddCommand(toolCmd)

	// Add subcommands
	toolCmd.AddCommand(toolListCmd)
	toolCmd.AddCommand(toolStatsCmd)

	// Common flags for all tool commands
	toolCmd.PersistentFlags().String("output", "json", "output format (json)")
	toolCmd.PersistentFlags().String("task-ref", "", "task reference (ET-101, 328001, etc.)")
	toolCmd.PersistentFlags().Bool("quiet", false, "quiet mode - compact JSON output")

	// Register completion functions for common flags
	toolCmd.RegisterFlagCompletionFunc("task-ref", completeTaskRefs)

	// Bind persistent flags to avoid repetition
	// Note: Individual tools will add their specific flags in their own init functions
}

// ToolContext provides common context for tool operations
type ToolContext struct {
	Context       context.Context
	CorrelationID string
	TaskRef       string
	Config        *config.Config
	Logger        logger.Logger
	OutputWriter  tools.OutputWriter
	Validator     *tools.Validator
	StartTime     time.Time
}

// NewToolContext creates a new tool context with common setup
func NewToolContext(cmd *cobra.Command, ctx context.Context) (*ToolContext, error) {
	startTime := time.Now()
	correlationID := tools.GenerateCorrelationID()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logger with tool-specific configuration
	consoleLoggerCfg := cfg.Logging.Loggers["console"]
	logConfig := (&consoleLoggerCfg).ToLoggerConfig()
	// Override for tool commands: use warn level and stderr output
	logConfig.Level = logger.WarnLevel
	logConfig.Output = "stderr"

	log, err := logger.New(logConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Tool registry is already initialized globally in root command OnInitialize

	// Get task reference from flags
	taskRef, _ := cmd.Flags().GetString("task-ref")

	// Create validator
	validator := tools.NewValidator(cfg.Storage.DataDir)

	// Create output writer
	outputWriter := tools.NewJSONOutputWriter()

	toolCtx := &ToolContext{
		Context:       ctx,
		CorrelationID: correlationID,
		TaskRef:       taskRef,
		Config:        cfg,
		Logger:        log,
		OutputWriter:  outputWriter,
		Validator:     validator,
		StartTime:     startTime,
	}

	// Log tool invocation
	log.Info("tool invocation started",
		logger.String("correlation_id", correlationID),
		logger.String("task_ref", taskRef),
		logger.String("command", cmd.Name()),
	)

	return toolCtx, nil
}

// WriteSuccess writes a successful tool output
func (tc *ToolContext) WriteSuccess(data interface{}, toolName string) error {
	duration := time.Since(tc.StartTime)
	quiet, _ := rootCmd.PersistentFlags().GetBool("quiet")

	meta := tools.NewToolMeta(tc.CorrelationID, tc.TaskRef, toolName, duration)
	output := tools.NewSuccessOutput(data, meta)

	tc.Logger.Info("tool operation completed successfully",
		logger.String("correlation_id", tc.CorrelationID),
		logger.String("tool", toolName),
		logger.Int("duration_ms", int(duration.Milliseconds())),
	)

	return tc.OutputWriter.WriteOutput(output, quiet)
}

// WriteError writes an error tool output
func (tc *ToolContext) WriteError(code, message, toolName string, details map[string]interface{}) error {
	duration := time.Since(tc.StartTime)
	quiet, _ := rootCmd.PersistentFlags().GetBool("quiet")

	meta := tools.NewToolMeta(tc.CorrelationID, tc.TaskRef, toolName, duration)
	toolError := tools.NewToolError(code, message, tc.CorrelationID, details)
	output := tools.NewErrorOutput(toolError, meta)

	tc.Logger.Error("tool operation failed",
		logger.String("correlation_id", tc.CorrelationID),
		logger.String("tool", toolName),
		logger.String("error_code", code),
		logger.String("error_message", message),
		logger.Int("duration_ms", int(duration.Milliseconds())),
	)

	return tc.OutputWriter.WriteOutput(output, quiet)
}

// WriteSuccessWithAuth writes a successful tool output including auth metadata
func (tc *ToolContext) WriteSuccessWithAuth(data interface{}, toolName string, authStatus *tools.AuthStatus, dataSource string) error {
	duration := time.Since(tc.StartTime)
	quiet, _ := rootCmd.PersistentFlags().GetBool("quiet")

	meta := tools.NewToolMetaWithAuth(tc.CorrelationID, tc.TaskRef, toolName, duration, authStatus, dataSource)
	output := tools.NewSuccessOutput(data, meta)

	tc.Logger.Info("tool operation completed successfully",
		logger.String("correlation_id", tc.CorrelationID),
		logger.String("tool", toolName),
		logger.Int("duration_ms", int(duration.Milliseconds())),
	)
	return tc.OutputWriter.WriteOutput(output, quiet)
}

// WriteErrorWithAuth writes an error tool output including auth metadata
func (tc *ToolContext) WriteErrorWithAuth(code, message, toolName string, details map[string]interface{}, authStatus *tools.AuthStatus, dataSource string) error {
	duration := time.Since(tc.StartTime)
	quiet, _ := rootCmd.PersistentFlags().GetBool("quiet")

	meta := tools.NewToolMetaWithAuth(tc.CorrelationID, tc.TaskRef, toolName, duration, authStatus, dataSource)
	toolError := tools.NewToolError(code, message, tc.CorrelationID, details)
	output := tools.NewErrorOutput(toolError, meta)

	tc.Logger.Error("tool operation failed",
		logger.String("correlation_id", tc.CorrelationID),
		logger.String("tool", toolName),
		logger.String("error_code", code),
		logger.String("error_message", message),
		logger.Int("duration_ms", int(duration.Milliseconds())),
	)
	return tc.OutputWriter.WriteOutput(output, quiet)
}

// ValidateTaskRef validates and normalizes the task reference
func (tc *ToolContext) ValidateTaskRef() (*tools.ValidationResult, error) {
	if tc.TaskRef == "" {
		return &tools.ValidationResult{Valid: true}, nil // Task ref is optional for some tools
	}

	return tc.Validator.ValidateTaskReference(tc.TaskRef)
}

// runToolList handles the tool list command
func runToolList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	toolCtx, err := NewToolContext(cmd, ctx)
	if err != nil {
		return err
	}

	// Get all registered tools
	toolInfos := tools.ListTools()

	// Create response data
	data := map[string]interface{}{
		"tools": toolInfos,
		"count": len(toolInfos),
	}

	return toolCtx.WriteSuccess(data, "list")
}

// runToolStats handles the tool stats command
func runToolStats(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	toolCtx, err := NewToolContext(cmd, ctx)
	if err != nil {
		return err
	}

	// Get registry statistics
	stats := tools.GetRegistryStats()

	return toolCtx.WriteSuccess(stats, "stats")
}

// Helper functions for tool command implementations

// ValidateAndExecuteTool provides common validation and execution logic for tools
func ValidateAndExecuteTool(cmd *cobra.Command, toolName string, params map[string]interface{}, validationRules map[string]tools.ValidationRule) error {
	ctx := context.Background()
	toolCtx, err := NewToolContext(cmd, ctx)
	if err != nil {
		return err
	}

	// Validate task reference if provided
	if toolCtx.TaskRef != "" {
		taskResult, err := toolCtx.ValidateTaskRef()
		if err != nil {
			return toolCtx.WriteError(tools.ErrorCodeValidation,
				fmt.Sprintf("task reference validation failed: %v", err),
				toolName, map[string]interface{}{"task_ref": toolCtx.TaskRef})
		}

		if !taskResult.Valid {
			return toolCtx.WriteError(tools.ErrorCodeValidation,
				"invalid task reference",
				toolName, map[string]interface{}{
					"task_ref": toolCtx.TaskRef,
					"errors":   taskResult.Errors,
				})
		}

		// Use normalized task reference if available
		if normalized, exists := taskResult.Normalized["task_ref"]; exists {
			toolCtx.TaskRef = normalized
			params["task_ref"] = normalized
		}
	}

	// Validate parameters
	if len(validationRules) > 0 {
		paramResult, err := toolCtx.Validator.ValidateParameters(params, validationRules)
		if err != nil {
			return toolCtx.WriteError(tools.ErrorCodeValidation,
				fmt.Sprintf("parameter validation failed: %v", err),
				toolName, map[string]interface{}{"params": params})
		}

		if !paramResult.Valid {
			return toolCtx.WriteError(tools.ErrorCodeValidation,
				"invalid parameters",
				toolName, map[string]interface{}{
					"params": params,
					"errors": paramResult.Errors,
				})
		}

		// Apply normalized values
		for key, normalizedValue := range paramResult.Normalized {
			params[key] = normalizedValue
		}
	}

	// Execute the tool
	result, evidenceSource, err := tools.ExecuteTool(toolCtx.Context, toolName, params)
	if err != nil {
		// Determine error code based on error type
		errorCode := tools.ErrorCodeInternal
		if _, ok := err.(*tools.ValidationError); ok {
			errorCode = tools.ErrorCodeValidation
		}

		return toolCtx.WriteError(errorCode,
			fmt.Sprintf("tool execution failed: %v", err),
			toolName, map[string]interface{}{
				"params": params,
				"error":  err.Error(),
			})
	}

	// Prepare response data
	data := map[string]interface{}{
		"result": result,
	}

	if evidenceSource != nil {
		data["evidence_source"] = evidenceSource
	}

	return toolCtx.WriteSuccess(data, toolName)
}

// Common validation rules for reuse across tools
var (
	// TaskRefRule validates task reference format
	TaskRefRule = tools.ValidationRule{
		Required: false,
		Type:     "string",
		Pattern:  `^(ET-?\s*\d+|\d+)$`,
	}

	// PathRule validates file paths with safety checks
	PathRule = tools.ValidationRule{
		Required:   true,
		Type:       "path",
		PathSafety: true,
	}

	// OptionalPathRule validates optional file paths with safety checks
	OptionalPathRule = tools.ValidationRule{
		Required:   false,
		Type:       "path",
		PathSafety: true,
	}

	// StringRule validates required string parameters
	StringRule = tools.ValidationRule{
		Required:  true,
		Type:      "string",
		MinLength: 1,
	}

	// OptionalStringRule validates optional string parameters
	OptionalStringRule = tools.ValidationRule{
		Required: false,
		Type:     "string",
	}

	// IntRule validates integer parameters
	IntRule = tools.ValidationRule{
		Required: true,
		Type:     "int",
	}

	// BoolRule validates boolean parameters
	BoolRule = tools.ValidationRule{
		Required: false,
		Type:     "bool",
	}
)
