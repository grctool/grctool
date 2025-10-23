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
	"path/filepath"

	"github.com/grctool/grctool/internal/tools"
	"github.com/spf13/cobra"
)

// contextCmd represents the AI context command group
var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "AI context generation tools",
	Long: `AI context generation tools provide comprehensive codebase analysis
for AI agents to better understand the project structure, dependencies,
and architectural patterns.

Available context types:
- summary: High-level codebase overview with metrics and structure
- interfaces: Map of all Go interfaces and their implementations  
- dependencies: Module and package dependency graph
- recent: Recent file changes with semantic context`,
}

// contextSummaryCmd generates codebase summary
var contextSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Generate codebase summary for AI agents",
	Long: `Generate a comprehensive codebase summary including:
- File counts and metrics
- Package structure and organization
- Key architectural patterns
- Test coverage information
- Core modules and entry points

Output is optimized for AI agent consumption.`,
	RunE: runContextSummary,
}

// contextInterfacesCmd generates interface mapping
var contextInterfacesCmd = &cobra.Command{
	Use:   "interfaces",
	Short: "Map all Go interfaces and implementations",
	Long: `Generate a detailed mapping of all Go interfaces including:
- Interface definitions with method signatures
- Package locations and file paths
- Known implementations (when detectable)
- Cross-package interface relationships

Useful for AI agents to understand the codebase's interface contracts.`,
	RunE: runContextInterfaces,
}

// contextDepsCmd generates dependency graph
var contextDepsCmd = &cobra.Command{
	Use:   "deps",
	Short: "Generate dependency relationship graph",
	Long: `Generate a comprehensive dependency graph including:
- External module dependencies from go.mod
- Internal package import relationships
- Package centrality metrics (how connected each package is)
- Direct vs. indirect dependency classification

Helps AI agents understand the codebase's dependency structure.`,
	RunE: runContextDeps,
}

// contextAllCmd generates all context types
var contextAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Generate all context types",
	Long: `Generate all context types (summary, interfaces, dependencies) and
save them to the AI context cache directory. This provides comprehensive
context for AI agents working with the codebase.`,
	RunE: runContextAll,
}

func init() {
	// Add context command group to tool command
	toolCmd.AddCommand(contextCmd)

	// Add individual context commands
	contextCmd.AddCommand(contextSummaryCmd)
	contextCmd.AddCommand(contextInterfacesCmd)
	contextCmd.AddCommand(contextDepsCmd)
	contextCmd.AddCommand(contextAllCmd)

	// Common flags for context commands
	contextCmd.PersistentFlags().String("output-dir", ".ai-context", "output directory for context cache")
	contextCmd.PersistentFlags().Bool("pretty", true, "pretty print JSON output")
	contextCmd.PersistentFlags().Bool("save", true, "save output to cache files")

	// Individual command flags
	contextSummaryCmd.Flags().Bool("include-metrics", true, "include detailed metrics")
	contextInterfacesCmd.Flags().Bool("include-methods", true, "include method signatures")
	contextDepsCmd.Flags().Bool("include-internal", true, "include internal package dependencies")
}

// runContextSummary generates codebase summary
func runContextSummary(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get current working directory as root
	rootPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Create context generator
	generator := tools.NewContextGeneratorTool(rootPath)

	// Get flags
	outputDir, _ := cmd.Flags().GetString("output-dir")
	save, _ := cmd.Flags().GetBool("save")
	pretty, _ := cmd.Flags().GetBool("pretty")

	// Prepare parameters
	params := map[string]interface{}{
		"context_type": "summary",
	}

	if save {
		outputPath := filepath.Join(outputDir, "codebase-summary.json")
		params["output_path"] = outputPath
	}

	// Execute tool
	result, source, err := generator.Execute(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to generate context summary: %w", err)
	}

	// Output result
	if pretty {
		fmt.Println(result)
	} else {
		// Compact output for quiet mode
		fmt.Print(result)
	}

	// Log metadata if not quiet
	quiet, _ := cmd.Flags().GetBool("quiet")
	if !quiet && source != nil {
		fmt.Fprintf(os.Stderr, "Context generated: %s at %s\n",
			source.Type, source.ExtractedAt.Format("15:04:05"))
	}

	return nil
}

// runContextInterfaces generates interface mapping
func runContextInterfaces(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	rootPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	generator := tools.NewContextGeneratorTool(rootPath)

	outputDir, _ := cmd.Flags().GetString("output-dir")
	save, _ := cmd.Flags().GetBool("save")
	pretty, _ := cmd.Flags().GetBool("pretty")

	params := map[string]interface{}{
		"context_type": "interfaces",
	}

	if save {
		outputPath := filepath.Join(outputDir, "interfaces.json")
		params["output_path"] = outputPath
	}

	result, source, err := generator.Execute(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to generate interface mapping: %w", err)
	}

	if pretty {
		fmt.Println(result)
	} else {
		fmt.Print(result)
	}

	quiet, _ := cmd.Flags().GetBool("quiet")
	if !quiet && source != nil {
		fmt.Fprintf(os.Stderr, "Interface mapping generated: %d interfaces found\n",
			len(source.Metadata))
	}

	return nil
}

// runContextDeps generates dependency graph
func runContextDeps(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	rootPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	generator := tools.NewContextGeneratorTool(rootPath)

	outputDir, _ := cmd.Flags().GetString("output-dir")
	save, _ := cmd.Flags().GetBool("save")
	pretty, _ := cmd.Flags().GetBool("pretty")

	params := map[string]interface{}{
		"context_type": "dependencies",
	}

	if save {
		outputPath := filepath.Join(outputDir, "dependencies.json")
		params["output_path"] = outputPath
	}

	result, source, err := generator.Execute(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to generate dependency graph: %w", err)
	}

	if pretty {
		fmt.Println(result)
	} else {
		fmt.Print(result)
	}

	quiet, _ := cmd.Flags().GetBool("quiet")
	if !quiet && source != nil {
		fmt.Fprintf(os.Stderr, "Dependency graph generated at %s\n",
			source.ExtractedAt.Format("15:04:05"))
	}

	return nil
}

// runContextAll generates all context types
func runContextAll(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	rootPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	generator := tools.NewContextGeneratorTool(rootPath)

	outputDir, _ := cmd.Flags().GetString("output-dir")
	save, _ := cmd.Flags().GetBool("save")
	quiet, _ := cmd.Flags().GetBool("quiet")

	// Ensure output directory exists
	if save {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	contextTypes := []struct {
		name string
		file string
	}{
		{"summary", "codebase-summary.json"},
		{"interfaces", "interfaces.json"},
		{"dependencies", "dependencies.json"},
	}

	results := make(map[string]string)

	// Generate each context type
	for _, ctxType := range contextTypes {
		if !quiet {
			fmt.Fprintf(os.Stderr, "Generating %s context...\n", ctxType.name)
		}

		params := map[string]interface{}{
			"context_type": ctxType.name,
		}

		if save {
			outputPath := filepath.Join(outputDir, ctxType.file)
			params["output_path"] = outputPath
		}

		result, source, err := generator.Execute(ctx, params)
		if err != nil {
			return fmt.Errorf("failed to generate %s context: %w", ctxType.name, err)
		}

		results[ctxType.name] = result

		if !quiet && source != nil {
			fmt.Fprintf(os.Stderr, "✓ %s context generated\n", ctxType.name)
		}
	}

	// Output summary
	if !quiet {
		fmt.Fprintf(os.Stderr, "\nAI Context Generation Complete\n")
		fmt.Fprintf(os.Stderr, "Generated: %d context files\n", len(contextTypes))
		if save {
			fmt.Fprintf(os.Stderr, "Saved to: %s/\n", outputDir)
		}
		fmt.Fprintf(os.Stderr, "Ready for AI agent consumption\n")
	}

	// Output combined results as JSON array
	fmt.Printf("{\n")
	first := true
	for name, result := range results {
		if !first {
			fmt.Printf(",\n")
		}
		fmt.Printf("  \"%s\": %s", name, result)
		first = false
	}
	fmt.Printf("\n}\n")

	return nil
}
