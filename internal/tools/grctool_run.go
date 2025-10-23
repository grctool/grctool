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

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
)

// GrctoolRunTool provides safe execution of allowlisted grctool commands
type GrctoolRunTool struct {
	config          *config.Config
	logger          logger.Logger
	allowedCommands map[string]CommandConfig
}

// CommandConfig defines configuration for an allowed command
type CommandConfig struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	AllowedFlags []string `json:"allowed_flags"`
	MaxTimeout   int      `json:"max_timeout"`
	RequiresAuth bool     `json:"requires_auth"`
}

// NewGrctoolRunTool creates a new grctool run tool
func NewGrctoolRunTool(cfg *config.Config, log logger.Logger) *GrctoolRunTool {
	// Define allowed commands and their configurations
	allowedCommands := map[string]CommandConfig{
		"sync": {
			Name:        "sync",
			Description: "Sync data from Tugboat Logic",
			AllowedFlags: []string{
				"--policies", "--controls", "--evidence", "--relationships",
				"--all", "--force", "--cache-only", "--no-cache",
				"--output", "--json", "--verbose",
			},
			MaxTimeout:   300, // 5 minutes
			RequiresAuth: true,
		},
		"evidence": {
			Name:        "evidence",
			Description: "Evidence management operations",
			AllowedFlags: []string{
				"list", "analyze", "generate", "review", "submit",
				"--status", "--framework", "--priority", "--assignee",
				"--overdue", "--due-soon", "--category", "--all",
				"--output", "--markdown", "--tools", "--format",
				"--show-reasoning", "--show-sources",
			},
			MaxTimeout:   600, // 10 minutes for evidence generation
			RequiresAuth: false,
		},
		"policy": {
			Name:        "policy",
			Description: "Policy management operations",
			AllowedFlags: []string{
				"list", "show", "search",
				"--framework", "--status", "--category",
				"--output", "--json", "--verbose",
			},
			MaxTimeout:   120, // 2 minutes
			RequiresAuth: false,
		},
		"control": {
			Name:        "control",
			Description: "Control management operations",
			AllowedFlags: []string{
				"list", "show", "search",
				"--framework", "--status", "--category",
				"--output", "--json", "--verbose",
			},
			MaxTimeout:   120, // 2 minutes
			RequiresAuth: false,
		},
		"config": {
			Name:        "config",
			Description: "Configuration management",
			AllowedFlags: []string{
				"show", "validate", "test-connection",
				"--output", "--json",
			},
			MaxTimeout:   60, // 1 minute
			RequiresAuth: false,
		},
	}

	return &GrctoolRunTool{
		config:          cfg,
		logger:          log,
		allowedCommands: allowedCommands,
	}
}

// GetClaudeToolDefinition returns the tool definition for Claude
func (g *GrctoolRunTool) GetClaudeToolDefinition() models.ClaudeTool {
	// Build command enum from allowed commands
	commandNames := make([]string, 0, len(g.allowedCommands))
	for name := range g.allowedCommands {
		commandNames = append(commandNames, name)
	}

	return models.ClaudeTool{
		Name:        "grctool-run",
		Description: "Execute allowlisted grctool commands with safe flag parsing and structured output capture.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "Grctool command to execute",
					"enum":        commandNames,
				},
				"args": map[string]interface{}{
					"type":        "array",
					"description": "Command arguments and flags",
					"items":       map[string]interface{}{"type": "string"},
					"default":     []string{},
				},
				"capture_output": map[string]interface{}{
					"type":        "boolean",
					"description": "Capture stdout and stderr",
					"default":     true,
				},
				"timeout": map[string]interface{}{
					"type":        "integer",
					"description": "Command timeout in seconds",
					"minimum":     1,
					"maximum":     600,
					"default":     300,
				},
			},
			"required": []string{"command"},
		},
	}
}

// Execute runs the grctool command execution tool
func (g *GrctoolRunTool) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	g.logger.Info("Starting grctool command execution",
		logger.Field{Key: "params", Value: params})

	// Parse parameters
	command, ok := params["command"].(string)
	if !ok || command == "" {
		return "", nil, fmt.Errorf("command parameter is required")
	}

	var args []string
	if argsParam, ok := params["args"].([]interface{}); ok {
		for _, arg := range argsParam {
			if argStr, ok := arg.(string); ok {
				args = append(args, argStr)
			}
		}
	} else if argsParam, ok := params["args"].([]string); ok {
		args = argsParam
	}

	captureOutput, _ := params["capture_output"].(bool)
	if _, exists := params["capture_output"]; !exists {
		captureOutput = true // Default to true
	}

	timeout, _ := params["timeout"].(int)
	if timeout == 0 {
		timeout = 300 // Default timeout
	}

	// Validate command
	cmdConfig, exists := g.allowedCommands[command]
	if !exists {
		return "", nil, fmt.Errorf("command not allowed: %s", command)
	}

	// Apply command-specific timeout limits
	if timeout > cmdConfig.MaxTimeout {
		timeout = cmdConfig.MaxTimeout
	}

	// Validate arguments
	if err := g.validateArguments(command, args, cmdConfig); err != nil {
		return "", nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Check authentication if required
	if cmdConfig.RequiresAuth {
		if err := g.checkAuthentication(); err != nil {
			return "", nil, fmt.Errorf("authentication required: %w", err)
		}
	}

	// Execute command
	result, err := g.executeCommand(ctx, command, args, timeout, captureOutput)
	if err != nil {
		g.logger.Error("Command execution failed",
			logger.Field{Key: "command", Value: command},
			logger.Field{Key: "args", Value: args},
			logger.Field{Key: "error", Value: err})
		return "", nil, fmt.Errorf("command execution failed: %w", err)
	}

	// Create evidence source metadata
	source := &models.EvidenceSource{
		Type:        "grctool-run",
		Resource:    fmt.Sprintf("Execution of grctool %s command", command),
		Content:     result.Stdout,
		ExtractedAt: result.ExecutedAt,
		Metadata: map[string]interface{}{
			"command":     command,
			"args":        args,
			"exit_code":   result.ExitCode,
			"duration":    result.Duration.String(),
			"executed_at": result.ExecutedAt,
		},
	}

	// Format response
	response := map[string]interface{}{
		"success":     result.ExitCode == 0,
		"command":     command,
		"args":        args,
		"exit_code":   result.ExitCode,
		"stdout":      result.Stdout,
		"stderr":      result.Stderr,
		"duration":    result.Duration.String(),
		"executed_at": result.ExecutedAt,
	}

	if result.ExitCode != 0 {
		response["error"] = fmt.Sprintf("Command exited with code %d", result.ExitCode)
	}

	responseJSON, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", source, fmt.Errorf("failed to marshal response: %w", err)
	}

	g.logger.Info("Command execution completed",
		logger.Field{Key: "command", Value: command},
		logger.Field{Key: "exit_code", Value: result.ExitCode},
		logger.Field{Key: "duration", Value: result.Duration},
	)

	return string(responseJSON), source, nil
}

// Name returns the tool name
func (g *GrctoolRunTool) Name() string {
	return "grctool-run"
}

// Description returns the tool description
func (g *GrctoolRunTool) Description() string {
	return "Execute allowlisted grctool commands with safe flag parsing and structured output capture"
}

// Version returns the tool version
func (g *GrctoolRunTool) Version() string {
	return "1.0.0"
}

// Category returns the tool category
func (g *GrctoolRunTool) Category() string {
	return "meta-tool"
}

// CommandExecutionResult represents the result of a command execution
type CommandExecutionResult struct {
	ExitCode   int           `json:"exit_code"`
	Stdout     string        `json:"stdout"`
	Stderr     string        `json:"stderr"`
	Duration   time.Duration `json:"duration"`
	ExecutedAt time.Time     `json:"executed_at"`
}

// validateArguments validates command arguments against allowed flags
func (g *GrctoolRunTool) validateArguments(command string, args []string, cmdConfig CommandConfig) error {
	allowedFlags := make(map[string]bool)
	for _, flag := range cmdConfig.AllowedFlags {
		allowedFlags[flag] = true
	}

	for _, arg := range args {
		// Skip non-flag arguments (subcommands, values, etc.)
		if !strings.HasPrefix(arg, "-") {
			// Check if it's an allowed subcommand
			if allowedFlags[arg] {
				continue
			}
			// For now, allow non-flag arguments (they might be values)
			continue
		}

		// For flags, check if they're in the allowed list
		flag := arg
		if strings.Contains(arg, "=") {
			// Handle --flag=value format
			flag = strings.Split(arg, "=")[0]
		}

		if !allowedFlags[flag] {
			return fmt.Errorf("flag not allowed for command %s: %s", command, flag)
		}
	}

	return nil
}

// checkAuthentication verifies if authentication is available for commands that require it
func (g *GrctoolRunTool) checkAuthentication() error {
	// Check if Tugboat authentication is configured
	if g.config.Tugboat.CookieHeader == "" && g.config.Tugboat.BearerToken == "" {
		return fmt.Errorf("tugboat authentication not configured")
	}
	return nil
}

// executeCommand executes the grctool command with the specified arguments
func (g *GrctoolRunTool) executeCommand(ctx context.Context, command string, args []string, timeoutSeconds int, captureOutput bool) (*CommandExecutionResult, error) {
	startTime := time.Now()

	// Find grctool executable
	grctoolPath, err := g.findGrctoolExecutable()
	if err != nil {
		return nil, fmt.Errorf("failed to find grctool executable: %w", err)
	}

	// Build command arguments
	cmdArgs := []string{command}
	cmdArgs = append(cmdArgs, args...)

	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	// Create command
	cmd := exec.CommandContext(timeoutCtx, grctoolPath, cmdArgs...)

	// Set working directory to current directory
	cmd.Dir, _ = os.Getwd()

	// Set environment variables (inherit current environment)
	cmd.Env = os.Environ()

	result := &CommandExecutionResult{
		ExecutedAt: startTime,
	}

	if captureOutput {
		// Capture stdout and stderr
		stdout, err := cmd.Output()
		if err != nil {
			// Handle exit errors
			if exitError, ok := err.(*exec.ExitError); ok {
				result.ExitCode = exitError.ExitCode()
				result.Stderr = string(exitError.Stderr)
				result.Stdout = string(stdout)
			} else {
				return nil, fmt.Errorf("failed to execute command: %w", err)
			}
		} else {
			result.ExitCode = 0
			result.Stdout = string(stdout)
		}
	} else {
		// Run command without capturing output
		if err := cmd.Run(); err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				result.ExitCode = exitError.ExitCode()
			} else {
				return nil, fmt.Errorf("failed to execute command: %w", err)
			}
		} else {
			result.ExitCode = 0
		}
		result.Stdout = "Output not captured"
		result.Stderr = "Output not captured"
	}

	result.Duration = time.Since(startTime)

	g.logger.Debug("Command executed",
		logger.Field{Key: "command", Value: command},
		logger.Field{Key: "args", Value: args},
		logger.Field{Key: "exit_code", Value: result.ExitCode},
		logger.Field{Key: "duration", Value: result.Duration},
	)

	return result, nil
}

// findGrctoolExecutable finds the grctool executable
func (g *GrctoolRunTool) findGrctoolExecutable() (string, error) {
	// First, check if grctool is in PATH
	if grctoolPath, err := exec.LookPath("grctool"); err == nil {
		return grctoolPath, nil
	}

	// Check if we're running as part of grctool (self-execution)
	if currentExe, err := os.Executable(); err == nil {
		if strings.Contains(filepath.Base(currentExe), "grctool") {
			return currentExe, nil
		}
	}

	// Check common locations
	possiblePaths := []string{
		"./grctool",
		"../grctool",
		"/usr/local/bin/grctool",
		"/usr/bin/grctool",
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("grctool executable not found")
}

// GetAllowedCommands returns the list of allowed commands with their configurations
func (g *GrctoolRunTool) GetAllowedCommands() map[string]CommandConfig {
	return g.allowedCommands
}

// isCommandAllowed checks if a command is allowed
func (g *GrctoolRunTool) isCommandAllowed(command string) bool {
	_, exists := g.allowedCommands[command]
	return exists
}

// getCommandTimeout returns the maximum timeout for a command
func (g *GrctoolRunTool) getCommandTimeout(command string) int {
	if cmdConfig, exists := g.allowedCommands[command]; exists {
		return cmdConfig.MaxTimeout
	}
	return 60 // Default timeout
}

// Helper function to get exit code from process state
func getExitCode(processState *os.ProcessState) int {
	if processState == nil {
		return 0
	}

	// Platform-specific exit code extraction
	if waitStatus, ok := processState.Sys().(syscall.WaitStatus); ok {
		return waitStatus.ExitStatus()
	}

	return 0
}
