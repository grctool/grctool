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

	"github.com/grctool/grctool/internal/tools"
	"github.com/grctool/grctool/internal/tools/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// ValidateAndExecuteTypedTool executes a tool using the new typed interface
func ValidateAndExecuteTypedTool(cmd *cobra.Command, toolName string, params map[string]interface{}) error {
	ctx := context.Background()
	toolCtx, err := NewToolContext(cmd, ctx)
	if err != nil {
		return err
	}

	// Get the typed registry
	registry := tools.GetTypedRegistry()

	// Try to get a typed tool first
	typedTool, hasTyped := registry.GetTypedTool(toolName)
	if !hasTyped {
		// Fall back to legacy implementation
		return ValidateAndExecuteTool(cmd, toolName, params, nil)
	}

	// Handle task reference validation if provided
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

	// Create and validate typed request
	req, err := types.ValidateAndConvertParams(toolName, params)
	if err != nil {
		return toolCtx.WriteError(tools.ErrorCodeValidation,
			fmt.Sprintf("request validation failed: %v", err),
			toolName, map[string]interface{}{"params": params})
	}

	// Execute typed tool
	response, err := typedTool.ExecuteTyped(ctx, req)
	if err != nil {
		return toolCtx.WriteError(tools.ErrorCodeExecution,
			fmt.Sprintf("tool execution failed: %v", err),
			toolName, map[string]interface{}{"request": req})
	}

	// Check if response indicates success or failure
	if !response.(*types.ToolResponse).Success {
		return toolCtx.WriteError(tools.ErrorCodeExecution,
			response.(*types.ToolResponse).Error,
			toolName, response.GetMetadata())
	}

	// Write successful response
	return toolCtx.WriteSuccess(response, toolName)
}

// ValidateAndExecuteEitherTool executes a tool using the best available interface
func ValidateAndExecuteEitherTool(cmd *cobra.Command, toolName string, params map[string]interface{}, legacyValidationRules map[string]tools.ValidationRule) error {
	registry := tools.GetTypedRegistry()

	// Check if we have a typed version and it's not just an adapter
	if registry.HasTypedVersion(toolName) {
		return ValidateAndExecuteTypedTool(cmd, toolName, params)
	}

	// Fall back to legacy implementation
	return ValidateAndExecuteTool(cmd, toolName, params, legacyValidationRules)
}

// UpdateToolCommand updates an existing command to support both typed and legacy execution
func UpdateToolCommand(cmd *cobra.Command, toolName string, paramExtractor func(*cobra.Command) map[string]interface{}, validationRules map[string]tools.ValidationRule) {
	originalRunE := cmd.RunE

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		params := paramExtractor(cmd)

		// Try typed execution first, fall back to legacy
		return ValidateAndExecuteEitherTool(cmd, toolName, params, validationRules)
	}

	// Store original run function as backup
	cmd.Annotations = map[string]string{
		"original_run": "available",
		"tool_name":    toolName,
	}

	if originalRunE != nil {
		// We could store the original function if needed for rollback
		cmd.Annotations["has_original"] = "true"
	}
}

// TypedToolCommandBuilder helps create commands that work with typed tools
type TypedToolCommandBuilder struct {
	toolName string
	cmd      *cobra.Command
}

// NewTypedToolCommand creates a new command builder for typed tools
func NewTypedToolCommand(toolName, use, short, long string) *TypedToolCommandBuilder {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Long:  long,
	}

	return &TypedToolCommandBuilder{
		toolName: toolName,
		cmd:      cmd,
	}
}

// WithFlags adds typed flags based on the tool's request schema
func (b *TypedToolCommandBuilder) WithFlags() *TypedToolCommandBuilder {
	// Get field info for the tool to auto-generate flags
	if fieldInfo, err := types.GetRequestFieldInfo(b.toolName); err == nil {
		for fieldName, info := range fieldInfo {
			switch info.Type {
			case "string":
				b.cmd.Flags().String(fieldName, "", info.Description)
				if info.Required {
					b.cmd.MarkFlagRequired(fieldName)
				}
			case "bool":
				b.cmd.Flags().Bool(fieldName, false, info.Description)
			case "int":
				b.cmd.Flags().Int(fieldName, 0, info.Description)
				if info.Required {
					b.cmd.MarkFlagRequired(fieldName)
				}
			case "[]string":
				b.cmd.Flags().StringSlice(fieldName, nil, info.Description)
				if info.Required {
					b.cmd.MarkFlagRequired(fieldName)
				}
			}
		}
	}

	return b
}

// WithCustomFlags allows adding custom flags
func (b *TypedToolCommandBuilder) WithCustomFlags(flagSetup func(*cobra.Command)) *TypedToolCommandBuilder {
	flagSetup(b.cmd)
	return b
}

// Build creates the final command with typed execution
func (b *TypedToolCommandBuilder) Build() *cobra.Command {
	toolName := b.toolName

	b.cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Extract parameters from flags automatically
		params := make(map[string]interface{})

		cmd.Flags().Visit(func(flag *pflag.Flag) {
			switch flag.Value.Type() {
			case "string":
				if val, _ := cmd.Flags().GetString(flag.Name); val != "" {
					params[flag.Name] = val
				}
			case "bool":
				if cmd.Flags().Changed(flag.Name) {
					val, _ := cmd.Flags().GetBool(flag.Name)
					params[flag.Name] = val
				}
			case "int":
				if cmd.Flags().Changed(flag.Name) {
					val, _ := cmd.Flags().GetInt(flag.Name)
					params[flag.Name] = val
				}
			case "stringSlice":
				if vals, _ := cmd.Flags().GetStringSlice(flag.Name); len(vals) > 0 {
					params[flag.Name] = vals
				}
			}
		})

		return ValidateAndExecuteTypedTool(cmd, toolName, params)
	}

	return b.cmd
}
