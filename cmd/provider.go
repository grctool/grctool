// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grctool/grctool/internal/providers"
	"github.com/spf13/cobra"
)

// providerCmd is the parent command for provider management.
var providerCmd = &cobra.Command{
	Use:   "provider",
	Short: "Manage compliance data providers",
	Long: `Manage registered compliance data providers (Tugboat Logic, AccountableHQ, Google Drive, etc.).

Use subcommands to view provider status, capabilities, and health.`,
}

// providerStatusCmd shows registered providers with capabilities and health.
var providerStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show registered providers with capabilities and health",
	Long: `Display all registered compliance data providers with their capabilities,
read/write support, and connectivity health.

Examples:
  # Show all providers
  grctool provider status

  # Show providers in JSON format
  grctool provider status --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		registry := providers.GlobalRegistry()
		if registry == nil {
			cmd.Println("No provider registry initialized.")
			return nil
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")

		ctx := cmd.Context()
		infos, err := registry.ListProviderInfo(ctx)
		if err != nil {
			return fmt.Errorf("failed to list provider info: %w", err)
		}

		if len(infos) == 0 {
			if jsonOutput {
				cmd.Println("[]")
			} else {
				cmd.Println("No providers registered.")
			}
			return nil
		}

		if jsonOutput {
			data, err := json.MarshalIndent(infos, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal provider info: %w", err)
			}
			cmd.Println(string(data))
			return nil
		}

		// Human-readable table output
		cmd.Printf("%-20s %-12s %-10s %-10s %-10s %-8s\n",
			"PROVIDER", "HEALTH", "POLICIES", "CONTROLS", "EVIDENCE", "WRITE")
		cmd.Println(strings.Repeat("-", 74))

		for _, info := range infos {
			health := "healthy"
			if !info.Healthy {
				health = "unhealthy"
			}
			policies := boolToCheck(info.Capabilities.SupportsPolicies)
			controls := boolToCheck(info.Capabilities.SupportsControls)
			evidence := boolToCheck(info.Capabilities.SupportsEvidenceTasks)
			write := boolToCheck(info.Capabilities.SupportsWrite)

			cmd.Printf("%-20s %-12s %-10s %-10s %-10s %-8s\n",
				info.Name, health, policies, controls, evidence, write)
		}

		cmd.Printf("\n%d provider(s) registered.\n", len(infos))
		return nil
	},
}

func boolToCheck(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func init() {
	rootCmd.AddCommand(providerCmd)
	providerCmd.AddCommand(providerStatusCmd)

	providerStatusCmd.Flags().Bool("json", false, "Output in JSON format")
}
