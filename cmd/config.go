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
	"os"
	"strings"

	internalConfig "github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/services"
	"github.com/grctool/grctool/internal/services/config"
	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management commands",
	Long:  `Manage configuration for the Security Program Manager`,
}

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration file",
	Long:  `Create a new configuration file with default values`,
	RunE:  runConfigInit,
}

// configValidateCmd represents the config validate command
var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration file",
	Long:  `Validate the current configuration file for correctness`,
	RunE:  runConfigValidate,
}

func init() {
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(initCmd)
	configCmd.AddCommand(configValidateCmd)

	initCmd.Flags().StringP("output", "o", ".grctool.yaml", "output file path")
	initCmd.Flags().Bool("force", false, "overwrite existing configuration file")
	initCmd.Flags().Bool("skip-claude-md", false, "skip CLAUDE.md generation")
	initCmd.Flags().String("claude-md-output", "CLAUDE.md", "CLAUDE.md output path")
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	outputPath, _ := cmd.Flags().GetString("output")
	force, _ := cmd.Flags().GetBool("force")
	skipClaudeMd, _ := cmd.Flags().GetBool("skip-claude-md")
	claudeMdOutput, _ := cmd.Flags().GetString("claude-md-output")

	// Initialize config service
	configService, err := initializeConfigService()
	if err != nil {
		return err
	}

	// Check if config file exists
	configExists := fileExists(outputPath)

	// Only create/overwrite config if it doesn't exist or force is set
	if !configExists || force {
		if err := configService.InitializeConfig(outputPath, force); err != nil {
			return err
		}

		if configExists {
			cmd.Printf("Configuration file recreated at: %s\n", outputPath)
		} else {
			cmd.Printf("Configuration file created at: %s\n", outputPath)
		}
	} else {
		cmd.Printf("Configuration file already exists at: %s (use --force to overwrite)\n", outputPath)
	}

	// Generate CLAUDE.md unless skipped
	if !skipClaudeMd {
		claudeMdExists := fileExists(claudeMdOutput)

		// Always generate/update CLAUDE.md (it's always safe to regenerate)
		if err := configService.GenerateClaudeMd(claudeMdOutput, true); err != nil {
			return fmt.Errorf("failed to generate CLAUDE.md: %w", err)
		}

		if claudeMdExists {
			cmd.Printf("CLAUDE.md updated at: %s\n", claudeMdOutput)
		} else {
			cmd.Printf("CLAUDE.md created at: %s\n", claudeMdOutput)
		}

		// Generate agent documentation
		cfg, err := internalConfig.Load()
		if err != nil {
			// If config load fails, use a minimal default config for doc generation
			cmd.Printf("Warning: failed to load config: %v\n", err)
			cmd.Printf("Generating docs with default configuration...\n")
			cfg = &internalConfig.Config{
				Storage: internalConfig.StorageConfig{
					DataDir: "./data",
				},
			}
		}

		if err := services.GenerateAgentDocs(cfg, version); err != nil {
			cmd.Printf("Warning: failed to generate agent documentation: %v\n", err)
		} else {
			cmd.Printf("Agent documentation generated at: .grctool/docs/\n")
		}
	}

	// Print next steps
	cmd.Println("\nNext steps:")
	cmd.Println("1. Edit the configuration file to set your organization ID")
	cmd.Println("2. Run 'grctool auth login' to authenticate with Tugboat Logic")
	cmd.Println("3. Adjust paths and settings as needed")
	cmd.Println("4. Run 'grctool config validate' to verify your configuration")
	if !skipClaudeMd {
		cmd.Println("\nCLAUDE.md has been generated with AI assistant instructions.")
		cmd.Println("Agent documentation generated in .grctool/docs/ (use 'grctool agent-context' to view)")
		cmd.Println("Run 'grctool init' anytime to regenerate with updated configuration.")
	}

	return nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func runConfigValidate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Initialize config service
	configService, err := initializeConfigService()
	if err != nil {
		return err
	}

	// Validate configuration
	result, err := configService.ValidateConfig(ctx)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Print validation results
	printValidationResults(cmd, result)

	// Return error if validation failed
	if !result.Valid {
		return fmt.Errorf("configuration validation failed")
	}

	cmd.Println("Configuration validation passed")
	return nil
}

// Helper functions

func initializeConfigService() (config.Service, error) {
	// Initialize logger with basic configuration
	log, err := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "text",
		Output: "stderr",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	return config.NewService(log), nil
}

func printValidationResults(cmd *cobra.Command, result *config.ValidationResult) {
	cmd.Printf("Configuration Validation Report\n")
	cmd.Printf("==============================\n")
	cmd.Printf("Overall Status: %s\n", getStatusSymbol(result.Valid))
	cmd.Printf("Duration: %v\n\n", result.Duration)

	// Print individual check results
	for _, check := range result.Checks {
		symbol := getCheckStatusSymbol(check.Status)
		cmd.Printf("%s %s: %s (%v)\n", symbol, check.Name, check.Message, check.Duration)
	}

	// Print errors if any
	if len(result.Errors) > 0 {
		cmd.Printf("\nErrors:\n")
		for _, err := range result.Errors {
			cmd.Printf("  - %s\n", err)
		}
	}

	cmd.Println()
}

func getStatusSymbol(valid bool) string {
	if valid {
		return "PASS"
	}
	return "FAIL"
}

func getCheckStatusSymbol(status string) string {
	switch strings.ToLower(status) {
	case "pass":
		return "✓"
	case "fail":
		return "✗"
	case "warning":
		return "⚠"
	default:
		return "?"
	}
}
