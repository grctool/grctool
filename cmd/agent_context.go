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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// Available agent-context topics
var agentContextTopics = map[string]topicInfo{
	"directory-structure": {
		filename:    "directory-structure.md",
		description: "Evidence directory layout and structure",
	},
	"evidence-workflow": {
		filename:    "evidence-workflow.md",
		description: "Complete evidence generation workflow",
	},
	"tool-capabilities": {
		filename:    "tool-capabilities.md",
		description: "Available tools and their purposes",
	},
	"status-commands": {
		filename:    "status-commands.md",
		description: "Status command usage and filtering",
	},
	"submission-process": {
		filename:    "submission-process.md",
		description: "Evidence submission workflow",
	},
	"bulk-operations": {
		filename:    "bulk-operations.md",
		description: "Autonomous bulk evidence generation patterns",
	},
}

type topicInfo struct {
	filename    string
	description string
}

var agentContextCmd = &cobra.Command{
	Use:   "agent-context [topic]",
	Short: "Display detailed documentation for AI agents",
	Long: `Provides detailed technical documentation for AI agents to reference on-demand.

This command displays comprehensive documentation that AI assistants can use
to perform complex operations autonomously. Each topic provides in-depth
guidance, examples, and patterns for specific areas of GRCTool operation.

Available topics:
  directory-structure - Evidence directory layout and file organization
  evidence-workflow   - Complete evidence generation workflow
  tool-capabilities   - All available tools and when to use them
  status-commands     - Status command reference and filtering
  submission-process  - Evidence submission and tracking
  bulk-operations     - Autonomous bulk evidence generation (most important)

Examples:
  # List all available topics
  grctool agent-context

  # View directory structure documentation
  grctool agent-context directory-structure

  # View bulk operations guide (for autonomous operation)
  grctool agent-context bulk-operations

  # Show documentation version
  grctool agent-context --version

The documentation is generated during 'grctool init' and is versioned
with the grctool release for consistency.`,
	RunE:              runAgentContext,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeAgentContextTopics,
}

var showDocVersion bool

func init() {
	rootCmd.AddCommand(agentContextCmd)
	agentContextCmd.Flags().BoolVar(&showDocVersion, "version", false, "show documentation version")
}

func runAgentContext(cmd *cobra.Command, args []string) error {
	// Show version if requested
	if showDocVersion {
		cmd.Printf("Agent Documentation Version: %s\n", version)
		cmd.Printf("Documentation Location: .grctool/docs/\n")
		return nil
	}

	// If no topic specified, list all available topics
	if len(args) == 0 {
		return listAgentContextTopics(cmd)
	}

	topic := args[0]

	// Check if topic exists
	topicInfo, exists := agentContextTopics[topic]
	if !exists {
		cmd.Printf("❌ Unknown topic: %s\n\n", topic)
		cmd.Println("Available topics:")
		for t, info := range agentContextTopics {
			cmd.Printf("  %-20s - %s\n", t, info.description)
		}
		return fmt.Errorf("unknown topic: %s", topic)
	}

	// Find documentation file
	docPath := filepath.Join(".grctool", "docs", topicInfo.filename)

	// Check if file exists
	if _, err := os.Stat(docPath); os.IsNotExist(err) {
		cmd.Printf("❌ Documentation not found: %s\n\n", docPath)
		cmd.Println("Run 'grctool init' to generate agent documentation.")
		return fmt.Errorf("documentation not found (run 'grctool init')")
	}

	// Read and display documentation
	content, err := os.ReadFile(docPath)
	if err != nil {
		return fmt.Errorf("failed to read documentation: %w", err)
	}

	// Display the content
	cmd.Print(string(content))

	return nil
}

func listAgentContextTopics(cmd *cobra.Command) error {
	cmd.Println("Available agent-context topics:")
	cmd.Println()

	// Find longest topic name for formatting
	maxLen := 0
	for topic := range agentContextTopics {
		if len(topic) > maxLen {
			maxLen = len(topic)
		}
	}

	// Display topics in sorted order
	topics := []string{
		"directory-structure",
		"evidence-workflow",
		"tool-capabilities",
		"status-commands",
		"submission-process",
		"bulk-operations",
	}

	for _, topic := range topics {
		info := agentContextTopics[topic]
		padding := strings.Repeat(" ", maxLen-len(topic))
		cmd.Printf("  %s%s - %s\n", topic, padding, info.description)
	}

	cmd.Println()
	cmd.Println("Usage:")
	cmd.Println("  grctool agent-context <topic>")
	cmd.Println()
	cmd.Println("Examples:")
	cmd.Println("  grctool agent-context bulk-operations     # Most important for autonomous operation")
	cmd.Println("  grctool agent-context directory-structure # Understand file layout")
	cmd.Println("  grctool agent-context tool-capabilities   # See all available tools")
	cmd.Println()
	cmd.Printf("Documentation version: %s\n", version)
	cmd.Println("Documentation location: .grctool/docs/")
	cmd.Println()
	cmd.Println("💡 Tip: Run 'grctool init' to regenerate documentation if missing.")

	return nil
}

// completeAgentContextTopics provides shell completion for topics
func completeAgentContextTopics(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		// Return all topic names
		topics := make([]string, 0, len(agentContextTopics))
		for topic := range agentContextTopics {
			topics = append(topics, topic)
		}
		return topics, cobra.ShellCompDirectiveNoFileComp
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}
