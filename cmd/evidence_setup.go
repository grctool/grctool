// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/registry"
	"github.com/grctool/grctool/internal/storage"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var evidenceSetupCmd = &cobra.Command{
	Use:   "setup <task-identifier>",
	Short: "Configure collector URL for evidence submission",
	Long: `Configure the Tugboat collector URL for an evidence task.

The task identifier can be either:
  - ET reference: ET-0001, ET-0047 (case-insensitive)
  - Tugboat task ID: 327992, 327993 (numeric)

Collector URLs are obtained from Tugboat Logic:
  Custom Integrations > Evidence Services > Copy URL

Examples:
  # Configure using ET reference
  grctool evidence setup ET-0001 --collector-url "https://openapi.tugboatlogic.com/..."

  # Configure using Tugboat task ID
  grctool evidence setup 327992 --collector-url "https://openapi.tugboatlogic.com/..."

  # Interactive mode (prompts for URL)
  grctool evidence setup ET-0001

  # Preview changes without modifying config
  grctool evidence setup ET-0001 --collector-url "https://..." --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runEvidenceSetup,
}

func init() {
	evidenceCmd.AddCommand(evidenceSetupCmd)

	evidenceSetupCmd.Flags().String("collector-url", "", "collector URL from Tugboat UI")
	evidenceSetupCmd.Flags().Bool("dry-run", false, "preview changes without modifying config")
	evidenceSetupCmd.Flags().Bool("force", false, "overwrite existing URL without confirmation")
}

// runEvidenceSetup handles the evidence setup command
func runEvidenceSetup(cmd *cobra.Command, args []string) error {
	identifier := args[0]

	// Get flags
	collectorURL, _ := cmd.Flags().GetString("collector-url")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize storage
	stor, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Resolve task identifier
	taskRef, taskID, err := resolveTaskIdentifier(stor, cfg, identifier)
	if err != nil {
		return err
	}

	cmd.Printf("📋 Task: %s (ID: %d)\n", taskRef, taskID)

	// Get collector URL (prompt if not provided)
	if collectorURL == "" {
		reader := bufio.NewReader(os.Stdin)
		collectorURL, err = promptForCollectorURL(reader)
		if err != nil {
			return fmt.Errorf("failed to read URL: %w", err)
		}
	}

	// Validate URL
	if err := validateCollectorURL(collectorURL); err != nil {
		return err
	}

	// Check for existing URL
	existingURL := ""
	if cfg.Tugboat.CollectorURLs != nil {
		if existing, ok := cfg.Tugboat.CollectorURLs[taskRef]; ok {
			existingURL = existing
		}
	}

	if existingURL != "" && !force {
		reader := bufio.NewReader(os.Stdin)
		confirmed, err := confirmOverwrite(reader, taskRef, existingURL, collectorURL)
		if err != nil {
			return err
		}
		if !confirmed {
			cmd.Println("❌ Cancelled")
			return nil
		}
	}

	// Dry run mode
	if dryRun {
		cmd.Println("\n🔍 Dry Run - Changes Preview:")
		cmd.Println("═══════════════════════════════════════════════════════")
		cmd.Printf("  File: %s\n", viper.ConfigFileUsed())
		cmd.Printf("  Section: tugboat.collector_urls\n")
		cmd.Printf("  Key: %s\n", taskRef)
		if existingURL != "" {
			cmd.Printf("  Current: %s\n", existingURL)
		}
		cmd.Printf("  New: %s\n", collectorURL)
		cmd.Println("\n✅ No changes made (dry-run mode)")
		return nil
	}

	// Update config
	configPath := viper.ConfigFileUsed()
	if configPath == "" {
		return fmt.Errorf("no config file found - run 'grctool init' first")
	}

	if err := updateConfigCollectorURL(configPath, taskRef, collectorURL); err != nil {
		return err
	}

	// Success message
	cmd.Println("\n✅ Configuration updated successfully")
	cmd.Println("═══════════════════════════════════════════════════════")
	cmd.Printf("  Task: %s\n", taskRef)
	cmd.Printf("  Collector URL: %s\n", collectorURL)
	cmd.Printf("  Config file: %s\n", configPath)
	cmd.Println()
	cmd.Println("Next steps:")
	cmd.Println("  1. Set TUGBOAT_API_KEY environment variable")
	cmd.Println("  2. Configure username/password in config or environment")
	cmd.Printf("  3. Submit evidence: grctool evidence submit %s\n", taskRef)

	return nil
}

// resolveTaskIdentifier resolves a task identifier (ET reference or numeric ID) to both formats
func resolveTaskIdentifier(stor *storage.Storage, cfg *config.Config, identifier string) (taskRef string, taskID int, err error) {
	// Use existing storage.GetEvidenceTask which handles:
	// - "327992" (numeric)
	// - "ET-0001" (reference with dash)
	// - "ET0001" (reference without dash)
	task, err := stor.GetEvidenceTask(identifier)
	if err != nil {
		return "", 0, fmt.Errorf("task not found: %w\n\nPossible solutions:\n  • Run: grctool sync\n  • Verify task exists in Tugboat Logic\n  • List tasks: grctool evidence list", err)
	}

	// Get ET reference from registry
	evidenceRegistry := registry.NewEvidenceTaskRegistry(cfg.Storage.DataDir)
	if err := evidenceRegistry.LoadRegistry(); err != nil {
		return "", 0, fmt.Errorf("failed to load registry: %w", err)
	}

	ref, ok := evidenceRegistry.GetReference(task.ID)
	if !ok {
		return "", 0, fmt.Errorf("task %d has no reference mapping - run 'grctool sync'", task.ID)
	}

	return ref, task.ID, nil
}

// validateCollectorURL validates that a URL matches the Tugboat collector URL pattern
func validateCollectorURL(urlStr string) error {
	// Pattern for Tugboat collector URLs
	collectorURLPattern := regexp.MustCompile(`^https://openapi\.tugboatlogic\.com/api/v0/evidence/collector/\d+/$`)

	// Parse URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Must be HTTPS
	if parsedURL.Scheme != "https" {
		return fmt.Errorf("collector URL must use HTTPS")
	}

	// Must match pattern
	if !collectorURLPattern.MatchString(urlStr) {
		return fmt.Errorf("URL does not match Tugboat collector URL pattern\n"+
			"Expected format: https://openapi.tugboatlogic.com/api/v0/evidence/collector/<ID>/\n"+
			"Got: %s\n\n"+
			"Please copy the URL directly from Tugboat Logic:\n"+
			"  Custom Integrations > Evidence Services > Copy URL", urlStr)
	}

	// Must have trailing slash
	if !strings.HasSuffix(urlStr, "/") {
		return fmt.Errorf("collector URL must end with trailing slash")
	}

	return nil
}

// promptForCollectorURL prompts the user to enter a collector URL
func promptForCollectorURL(reader *bufio.Reader) (string, error) {
	fmt.Println("\nCollector URL Configuration")
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("To get your collector URL:")
	fmt.Println("  1. Log into Tugboat Logic")
	fmt.Println("  2. Navigate to: Custom Integrations > Evidence Services")
	fmt.Println("  3. Find your evidence task and click 'Copy URL'")
	fmt.Println()
	fmt.Print("Enter collector URL: ")

	url, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(url), nil
}

// confirmOverwrite prompts the user to confirm overwriting an existing URL
func confirmOverwrite(reader *bufio.Reader, taskRef, oldURL, newURL string) (bool, error) {
	fmt.Printf("\n⚠️  Task %s already has a collector URL configured\n", taskRef)
	fmt.Printf("   Current: %s\n", oldURL)
	fmt.Printf("   New:     %s\n", newURL)
	fmt.Print("\nOverwrite? [y/N]: ")

	response, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes", nil
}

// updateConfigCollectorURL updates the collector URL in the config file
func updateConfigCollectorURL(configPath, taskRef, collectorURL string) error {
	// 1. Read existing config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	// 2. Parse YAML
	var configMap map[string]interface{}
	if err := yaml.Unmarshal(data, &configMap); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// 3. Navigate to tugboat.collector_urls
	tugboat, ok := configMap["tugboat"].(map[string]interface{})
	if !ok {
		tugboat = make(map[string]interface{})
		configMap["tugboat"] = tugboat
	}

	collectorURLs, ok := tugboat["collector_urls"].(map[string]interface{})
	if !ok {
		collectorURLs = make(map[string]interface{})
		tugboat["collector_urls"] = collectorURLs
	}

	// 4. Update or add mapping
	collectorURLs[taskRef] = collectorURL

	// 5. Create backup
	backupPath := configPath + ".backup"
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// 6. Write updated config
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	if err := encoder.Encode(configMap); err != nil {
		// Restore from backup
		backupData, _ := os.ReadFile(backupPath)
		os.WriteFile(configPath, backupData, 0644)
		return fmt.Errorf("failed to write config: %w", err)
	}

	// 7. Clean up backup on success
	os.Remove(backupPath)

	return nil
}
