// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/storage"
	"github.com/spf13/cobra"
)

// syncStatusCmd shows entity-level sync metadata.
var syncStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show entity-level sync freshness, conflicts, and staleness",
	Long: `Display sync metadata for all entities, including last sync time per provider,
content hash status, and conflict state.

Per CD-4 (FEAT-004): this queries StorageService + SyncMetadata on domain entities,
not the ProviderRegistry.

Examples:
  # Show sync status for all entities
  grctool sync status

  # Filter by conflict state
  grctool sync status --conflicts

  # Output as JSON
  grctool sync status --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		store, err := storage.NewStorage(cfg.Storage)
		if err != nil {
			return fmt.Errorf("failed to initialize storage: %w", err)
		}

		conflictsOnly, _ := cmd.Flags().GetBool("conflicts")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		type syncEntry struct {
			Type          string            `json:"type"`
			ID            string            `json:"id"`
			Name          string            `json:"name"`
			LastSyncTime  map[string]string `json:"last_sync_time,omitempty"`
			ContentHash   map[string]string `json:"content_hash,omitempty"`
			ConflictState string            `json:"conflict_state,omitempty"`
		}

		var entries []syncEntry

		// Collect policies
		policies, err := store.GetAllPolicies()
		if err != nil {
			return fmt.Errorf("failed to get policies: %w", err)
		}
		for _, p := range policies {
			e := syncEntry{Type: "policy", ID: p.ID, Name: p.Name}
			if p.SyncMetadata != nil {
				e.LastSyncTime = formatTimeMap(p.SyncMetadata.LastSyncTime)
				e.ContentHash = p.SyncMetadata.ContentHash
				e.ConflictState = p.SyncMetadata.ConflictState
			}
			if !conflictsOnly || e.ConflictState != "" {
				entries = append(entries, e)
			}
		}

		// Collect controls
		controls, err := store.GetAllControls()
		if err != nil {
			return fmt.Errorf("failed to get controls: %w", err)
		}
		for _, c := range controls {
			e := syncEntry{Type: "control", ID: c.ID, Name: c.Name}
			if c.SyncMetadata != nil {
				e.LastSyncTime = formatTimeMap(c.SyncMetadata.LastSyncTime)
				e.ContentHash = c.SyncMetadata.ContentHash
				e.ConflictState = c.SyncMetadata.ConflictState
			}
			if !conflictsOnly || e.ConflictState != "" {
				entries = append(entries, e)
			}
		}

		// Collect evidence tasks
		tasks, err := store.GetAllEvidenceTasks()
		if err != nil {
			return fmt.Errorf("failed to get evidence tasks: %w", err)
		}
		for _, t := range tasks {
			e := syncEntry{Type: "evidence_task", ID: t.ID, Name: t.Name}
			if t.SyncMetadata != nil {
				e.LastSyncTime = formatTimeMap(t.SyncMetadata.LastSyncTime)
				e.ContentHash = t.SyncMetadata.ContentHash
				e.ConflictState = t.SyncMetadata.ConflictState
			}
			if !conflictsOnly || e.ConflictState != "" {
				entries = append(entries, e)
			}
		}

		if jsonOutput {
			data, err := json.MarshalIndent(entries, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal sync status: %w", err)
			}
			cmd.Println(string(data))
			return nil
		}

		if len(entries) == 0 {
			if conflictsOnly {
				cmd.Println("No entities with conflicts found.")
			} else {
				cmd.Println("No entities found.")
			}
			return nil
		}

		// Summary stats
		var withSync, withHash, withConflict int
		for _, e := range entries {
			if len(e.LastSyncTime) > 0 {
				withSync++
			}
			if len(e.ContentHash) > 0 {
				withHash++
			}
			if e.ConflictState != "" {
				withConflict++
			}
		}

		cmd.Printf("Sync Status: %d entities, %d synced, %d hashed, %d with conflicts\n\n",
			len(entries), withSync, withHash, withConflict)

		if withConflict > 0 {
			cmd.Println("Conflicts:")
			for _, e := range entries {
				if e.ConflictState != "" {
					cmd.Printf("  %-15s %-14s %s (%s)\n", e.Type, e.ID, e.Name, e.ConflictState)
				}
			}
			cmd.Println()
		}

		// Show entities grouped by type
		for _, typeName := range []string{"policy", "control", "evidence_task"} {
			var typeEntries []syncEntry
			for _, e := range entries {
				if e.Type == typeName {
					typeEntries = append(typeEntries, e)
				}
			}
			if len(typeEntries) == 0 {
				continue
			}
			label := strings.ReplaceAll(typeName, "_", " ")
			// Capitalize first letter without deprecated strings.Title
			if len(label) > 0 {
				label = strings.ToUpper(label[:1]) + label[1:]
			}
			cmd.Printf("%ss: %d\n", label, len(typeEntries))
			for _, e := range typeEntries {
				syncInfo := "-"
				if len(e.LastSyncTime) > 0 {
					parts := make([]string, 0)
					for provider, t := range e.LastSyncTime {
						parts = append(parts, fmt.Sprintf("%s:%s", provider, t))
					}
					syncInfo = strings.Join(parts, ", ")
				}
				name := e.Name
				if len(name) > 35 {
					name = name[:35] + ".."
				}
				cmd.Printf("  %-14s %-37s sync=%s\n", e.ID, name, syncInfo)
			}
		}

		return nil
	},
}

func formatTimeMap(times map[string]time.Time) map[string]string {
	if len(times) == 0 {
		return nil
	}
	result := make(map[string]string, len(times))
	for k, v := range times {
		result[k] = v.Format(time.RFC3339)
	}
	return result
}

func init() {
	syncCmd.AddCommand(syncStatusCmd)

	syncStatusCmd.Flags().Bool("conflicts", false, "Show only entities with unresolved conflicts")
	syncStatusCmd.Flags().Bool("json", false, "Output in JSON format")
}
