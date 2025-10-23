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
	"strings"
	"time"

	"github.com/grctool/grctool/internal/auth"
	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// authCmd represents the auth command
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage Tugboat Logic authentication",
	Long: `Manage authentication with Tugboat Logic.

This command provides subcommands for handling authentication,
including automated browser-based login.`,
}

// loginCmd represents the auth login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Tugboat Logic via Safari",
	Long: `Authenticate with Tugboat Logic using Safari browser (macOS only).

This command will:
1. Open Safari to the Tugboat Logic login page
2. Wait for you to complete the login process (supports Touch ID, Face ID, 1Password)
3. Automatically extract authentication cookies using AppleScript
4. Save the credentials to your configuration file

Requirements:
- macOS only (Safari automation uses AppleScript)
- Enable "Allow JavaScript from Apple Events" in Safari Developer settings for automatic extraction

Example:
  grctool auth login
  grctool auth login --timeout 10m`,
	RunE: runLogin,
}

// logoutCmd represents the auth logout command
var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored Tugboat Logic credentials",
	Long: `Remove stored Tugboat Logic credentials from the configuration file.

This will clear the stored authentication cookies and tokens.`,
	RunE: runLogout,
}

// statusCmd represents the auth status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check authentication status",
	Long: `Check the current authentication status with Tugboat Logic.

This will validate stored credentials and display session information.`,
	RunE: runAuthStatus,
}

var (
	authTimeout time.Duration
	saveToFile  string
)

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(statusCmd)

	// Login command flags
	loginCmd.Flags().DurationVar(&authTimeout, "timeout", 5*time.Minute, "Authentication timeout")
	loginCmd.Flags().StringVar(&saveToFile, "save-to", "", "Save credentials to specific file (default: current config)")
}

func runLogin(cmd *cobra.Command, args []string) error {
	// Load configuration to get base URL
	cfg, err := loadConfigForAuth()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create browser auth handler (Safari only)
	browserAuth := auth.NewBrowserAuth(cfg.Tugboat.BaseURL)
	browserAuth.Timeout = authTimeout

	cmd.Println("🚀 Starting Safari authentication flow...")
	cmd.Printf("⏱️  Timeout: %v\n\n", authTimeout)

	// Perform browser authentication
	ctx := context.Background()
	creds, err := browserAuth.Login(ctx)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	cmd.Println("\n✅ Authentication successful!")
	cmd.Println("🔍 Validating credentials...")

	// Validate the captured credentials
	if err := auth.ValidateCredentials(ctx, cfg.Tugboat.BaseURL, creds); err != nil {
		// Check if this is a bearer token issue vs other validation issues
		if strings.Contains(err.Error(), "credentials are invalid or expired") && creds.BearerToken == "" {
			cmd.Printf("❌ Authentication failed: No valid bearer token found in cookies\n")
			cmd.Printf("💡 This often happens on the first login attempt after token expiry\n")
			cmd.Printf("🔄 Try running 'grctool auth login' again - it usually works on the second attempt\n")
			cmd.Printf("📋 If the problem persists:\n")
			cmd.Printf("   1. Refresh the Tugboat Logic page in Safari\n")
			cmd.Printf("   2. Log out and log back in to Tugboat Logic\n")
			cmd.Printf("   3. Clear Safari cookies for tugboatlogic.com and try again\n")
			return fmt.Errorf("authentication failed due to missing bearer token")
		} else {
			// Other validation errors - still save credentials but warn
			cmd.Printf("⚠️  Warning: %v\n", err)
			cmd.Println("💡 Credentials saved anyway - try 'grctool sync' to verify they work")
		}
	} else {
		cmd.Println("✅ Credentials validated successfully!")
	}

	// Save credentials to configuration
	if err := saveCredentials(cfg, creds); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	cmd.Printf("\n🔐 Credentials saved to: %s\n", getConfigPath())
	if creds.OrgID != "" {
		cmd.Printf("🏢 Organization ID: %s\n", creds.OrgID)
	}
	cmd.Printf("⏰ Session expires: %s\n", creds.ExpiresAt.Format(time.RFC3339))
	cmd.Println("\n✨ You can now use 'grctool sync' without manual authentication!")

	return nil
}

func runLogout(cmd *cobra.Command, args []string) error {
	configPath := getConfigPath()

	// Load current configuration
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			cmd.Println("No configuration file found. Nothing to logout from.")
			return nil
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var configData map[string]interface{}
	if err := yaml.Unmarshal(data, &configData); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Remove authentication fields
	if tugboat, ok := configData["tugboat"].(map[string]interface{}); ok {
		delete(tugboat, "cookie_header")
		delete(tugboat, "bearer_token")
		delete(tugboat, "auth_expires")
	}

	// Write back to file
	updatedData, err := yaml.Marshal(configData)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, updatedData, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	cmd.Println("✅ Logged out successfully!")
	cmd.Printf("🔒 Credentials removed from: %s\n", configPath)

	return nil
}

func runAuthStatus(cmd *cobra.Command, args []string) error {
	logger.Trace("starting auth status check")

	// Load configuration
	cfg, err := loadConfigForAuth()
	if err != nil {
		logger.Debug("failed to load config", logger.Error(err))
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Check if we have credentials
	if cfg.Tugboat.CookieHeader == "" {
		logger.Trace("no credentials found in config")
		cmd.Println("❌ Not authenticated")
		cmd.Println("Run 'grctool auth login' to authenticate")
		return nil
	}

	logger.Debug("found stored credentials", logger.Int("cookie_length", len(cfg.Tugboat.CookieHeader)))
	cmd.Println("🔍 Checking authentication status...")

	// Create credentials object from config
	creds := &auth.AuthCredentials{
		CookieHeader: cfg.Tugboat.CookieHeader,
		BearerToken:  cfg.Tugboat.BearerToken,
	}

	// Validate credentials
	logger.Trace("validating credentials with API")
	ctx := context.Background()
	if err := auth.ValidateCredentials(ctx, cfg.Tugboat.BaseURL, creds); err != nil {
		cmd.Println("❌ Authentication invalid or expired")
		cmd.Printf("Error: %v\n", err)
		cmd.Println("\nRun 'grctool auth login' to re-authenticate")
		return nil
	}

	cmd.Println("✅ Authenticated successfully!")
	cmd.Printf("🌐 Connected to: %s\n", cfg.Tugboat.BaseURL)

	// Show cookie info (first 20 chars only for security)
	if len(cfg.Tugboat.CookieHeader) > 20 {
		cmd.Printf("🍪 Cookie: %s...\n", cfg.Tugboat.CookieHeader[:20])
	}

	// Show token info if available
	if cfg.Tugboat.BearerToken != "" {
		cmd.Println("🔑 Bearer token: Present")
	}

	return nil
}

// loadConfigForAuth loads minimal config needed for authentication
func loadConfigForAuth() (*config.Config, error) {
	// Try to load existing config
	cfg := &config.Config{}

	// Set defaults
	viper.SetDefault("tugboat.base_url", "https://app.tugboatlogic.com")

	// Read config file if it exists
	configPath := getConfigPath()
	if _, err := os.Stat(configPath); err == nil {
		viper.SetConfigFile(configPath)
		if err := viper.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Unmarshal into config struct
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Ensure we have a base URL
	if cfg.Tugboat.BaseURL == "" {
		cfg.Tugboat.BaseURL = "https://app.tugboatlogic.com"
	}

	return cfg, nil
}

// saveCredentials saves authentication credentials to the config file
func saveCredentials(cfg *config.Config, creds *auth.AuthCredentials) error {
	configPath := getConfigPath()
	if saveToFile != "" {
		configPath = saveToFile
	}

	// Update config with new credentials
	cfg.Tugboat.AuthMode = "browser"
	cfg.Tugboat.CookieHeader = creds.CookieHeader
	cfg.Tugboat.BearerToken = creds.BearerToken

	// Load existing config file to preserve other settings
	var configData map[string]interface{}

	if data, err := os.ReadFile(configPath); err == nil {
		if err := yaml.Unmarshal(data, &configData); err != nil {
			return fmt.Errorf("failed to parse existing config: %w", err)
		}
	} else {
		configData = make(map[string]interface{})
	}

	// Update tugboat section
	if _, ok := configData["tugboat"]; !ok {
		configData["tugboat"] = make(map[string]interface{})
	}

	tugboat := configData["tugboat"].(map[string]interface{})
	tugboat["auth_mode"] = "browser"
	tugboat["cookie_header"] = creds.CookieHeader
	if creds.BearerToken != "" {
		tugboat["bearer_token"] = creds.BearerToken
	}
	tugboat["auth_expires"] = creds.ExpiresAt.Format(time.RFC3339)

	// Save org_id if extracted from API
	if creds.OrgID != "" {
		tugboat["org_id"] = creds.OrgID
	}

	// Ensure base_url is set
	if _, ok := tugboat["base_url"]; !ok {
		tugboat["base_url"] = cfg.Tugboat.BaseURL
	}

	// Marshal to YAML
	data, err := yaml.Marshal(configData)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write to file with secure permissions
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// getConfigPath returns the path to the config file
func getConfigPath() string {
	if cfgFile != "" {
		return cfgFile
	}
	return ".grctool.yaml"
}
