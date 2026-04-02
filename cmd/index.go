// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/storage"
	"github.com/spf13/cobra"
)

// indexCmd is the parent command for master index operations.
var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Query the master index of compliance entities",
	Long: `Query the local master index of compliance entities (policies, controls, evidence tasks).

The master index is the StorageService — all synced entities with their
GRCTool-native IDs, external IDs, and sync metadata.`,
}

// indexListCmd lists entities from the master index.
var indexListCmd = &cobra.Command{
	Use:   "list",
	Short: "List entities in the master index with external IDs and sync state",
	Long: `Display all entities in the local master index with their reference IDs,
external provider IDs, and sync metadata.

Per CD-3 (FEAT-004): this queries StorageService, not the ProviderRegistry.

Examples:
  # List all entities
  grctool index list

  # Filter by entity type
  grctool index list --type policies
  grctool index list --type controls
  grctool index list --type evidence-tasks

  # Filter by provider
  grctool index list --provider tugboat

  # Output as JSON
  grctool index list --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		store, err := storage.NewStorage(cfg.Storage)
		if err != nil {
			return fmt.Errorf("failed to initialize storage: %w", err)
		}

		entityType, _ := cmd.Flags().GetString("type")
		providerFilter, _ := cmd.Flags().GetString("provider")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		type indexEntry struct {
			Type        string            `json:"type"`
			ID          string            `json:"id"`
			ReferenceID string            `json:"reference_id,omitempty"`
			Name        string            `json:"name"`
			ExternalIDs map[string]string `json:"external_ids,omitempty"`
		}

		var entries []indexEntry

		// Collect policies
		if entityType == "" || entityType == "policies" {
			policies, err := store.GetAllPolicies()
			if err != nil {
				return fmt.Errorf("failed to get policies: %w", err)
			}
			for _, p := range policies {
				if providerFilter != "" && !hasProvider(p.ExternalIDs, providerFilter) {
					continue
				}
				entries = append(entries, indexEntry{
					Type:        "policy",
					ID:          p.ID,
					ReferenceID: p.ReferenceID,
					Name:        p.Name,
					ExternalIDs: p.ExternalIDs,
				})
			}
		}

		// Collect controls
		if entityType == "" || entityType == "controls" {
			controls, err := store.GetAllControls()
			if err != nil {
				return fmt.Errorf("failed to get controls: %w", err)
			}
			for _, c := range controls {
				if providerFilter != "" && !hasProvider(c.ExternalIDs, providerFilter) {
					continue
				}
				entries = append(entries, indexEntry{
					Type:        "control",
					ID:          c.ID,
					ReferenceID: c.ReferenceID,
					Name:        c.Name,
					ExternalIDs: c.ExternalIDs,
				})
			}
		}

		// Collect evidence tasks
		if entityType == "" || entityType == "evidence-tasks" {
			tasks, err := store.GetAllEvidenceTasks()
			if err != nil {
				return fmt.Errorf("failed to get evidence tasks: %w", err)
			}
			for _, t := range tasks {
				if providerFilter != "" && !hasProvider(t.ExternalIDs, providerFilter) {
					continue
				}
				entries = append(entries, indexEntry{
					Type:        "evidence_task",
					ID:          t.ID,
					ReferenceID: t.ReferenceID,
					Name:        t.Name,
					ExternalIDs: t.ExternalIDs,
				})
			}
		}

		if jsonOutput {
			data, err := json.MarshalIndent(entries, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal index: %w", err)
			}
			cmd.Println(string(data))
			return nil
		}

		if len(entries) == 0 {
			cmd.Println("No entities found in the master index.")
			return nil
		}

		// Human-readable table
		cmd.Printf("%-15s %-14s %-16s %-30s %s\n",
			"TYPE", "ID", "REFERENCE", "NAME", "EXTERNAL IDS")
		cmd.Println(strings.Repeat("-", 100))

		for _, e := range entries {
			name := e.Name
			if len(name) > 28 {
				name = name[:28] + ".."
			}
			extIDs := formatExternalIDs(e.ExternalIDs)
			cmd.Printf("%-15s %-14s %-16s %-30s %s\n",
				e.Type, e.ID, e.ReferenceID, name, extIDs)
		}

		// Summary
		counts := map[string]int{}
		for _, e := range entries {
			counts[e.Type]++
		}
		parts := make([]string, 0)
		for _, t := range []string{"policy", "control", "evidence_task"} {
			if c, ok := counts[t]; ok {
				parts = append(parts, fmt.Sprintf("%d %ss", c, t))
			}
		}
		cmd.Printf("\n%d entities total (%s).\n", len(entries), strings.Join(parts, ", "))
		return nil
	},
}

func hasProvider(externalIDs map[string]string, provider string) bool {
	if externalIDs == nil {
		return false
	}
	_, ok := externalIDs[provider]
	return ok
}

func formatExternalIDs(ids map[string]string) string {
	if len(ids) == 0 {
		return "-"
	}
	keys := make([]string, 0, len(ids))
	for k := range ids {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s:%s", k, ids[k]))
	}
	return strings.Join(parts, ", ")
}

func init() {
	rootCmd.AddCommand(indexCmd)
	indexCmd.AddCommand(indexListCmd)

	indexListCmd.Flags().String("type", "", "Filter by entity type (policies, controls, evidence-tasks)")
	indexListCmd.Flags().String("provider", "", "Filter to entities with external ID from this provider")
	indexListCmd.Flags().Bool("json", false, "Output in JSON format")
}
