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

//go:build !e2e

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestAuthCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "auth command help",
			args:        []string{"auth", "--help"},
			expectError: false,
		},
		{
			name:        "auth login help",
			args:        []string{"auth", "login", "--help"},
			expectError: false,
		},
		{
			name:        "auth logout help",
			args:        []string{"auth", "logout", "--help"},
			expectError: false,
		},
		{
			name:        "auth status help",
			args:        []string{"auth", "status", "--help"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := rootCmd
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuthLogout(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	// Create a test config file with auth data
	testConfig := map[string]interface{}{
		"tugboat": map[string]interface{}{
			"base_url":      "https://app.tugboatlogic.com",
			"org_id":        "12345",
			"cookie_header": "session=test123",
			"bearer_token":  "test-bearer",
			"auth_expires":  "2024-01-01T00:00:00Z",
		},
		"storage": map[string]interface{}{
			"data_dir": "./data",
		},
	}

	data, err := yaml.Marshal(testConfig)
	require.NoError(t, err)
	err = os.WriteFile(configPath, data, 0600)
	require.NoError(t, err)

	// Set config file path
	oldCfgFile := cfgFile
	cfgFile = configPath
	defer func() { cfgFile = oldCfgFile }()

	// Create logout command
	cmd := &cobra.Command{
		Use:  "logout",
		RunE: runLogout,
	}

	// Capture output
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	// Run logout command
	err = cmd.Execute()
	require.NoError(t, err)

	// Read the config file and check auth fields are removed
	updatedData, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var updatedConfig map[string]interface{}
	err = yaml.Unmarshal(updatedData, &updatedConfig)
	require.NoError(t, err)

	tugboat := updatedConfig["tugboat"].(map[string]interface{})
	assert.NotContains(t, tugboat, "cookie_header")
	assert.NotContains(t, tugboat, "bearer_token")
	assert.NotContains(t, tugboat, "auth_expires")
	assert.Equal(t, "https://app.tugboatlogic.com", tugboat["base_url"])
}

func TestAuthStatus(t *testing.T) {
	tests := []struct {
		name           string
		configContent  map[string]interface{}
		expectedOutput string
	}{
		{
			name: "Not authenticated",
			configContent: map[string]interface{}{
				"tugboat": map[string]interface{}{
					"base_url": "https://app.tugboatlogic.com",
				},
			},
			expectedOutput: "❌ Not authenticated",
		},
		{
			name: "Has credentials",
			configContent: map[string]interface{}{
				"tugboat": map[string]interface{}{
					"base_url":      "https://app.tugboatlogic.com",
					"cookie_header": "session=abc123; token=xyz789",
					"bearer_token":  "test-bearer",
				},
			},
			expectedOutput: "🔍 Checking authentication status...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "test-config.yaml")

			data, err := yaml.Marshal(tt.configContent)
			require.NoError(t, err)
			err = os.WriteFile(configPath, data, 0600)
			require.NoError(t, err)

			// Set config file path
			oldCfgFile := cfgFile
			cfgFile = configPath
			defer func() { cfgFile = oldCfgFile }()

			// Create a fresh command instance to avoid state pollution
			cmd := &cobra.Command{
				Use:   "status",
				Short: "Check authentication status",
				RunE:  runAuthStatus,
			}

			// Capture command output
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			// Execute the status command directly
			err = cmd.Execute()
			require.NoError(t, err)

			output := buf.String()
			assert.Contains(t, output, tt.expectedOutput)
		})
	}
}

func TestLoadConfigForAuth(t *testing.T) {
	tests := []struct {
		name          string
		setupConfig   bool
		configContent map[string]interface{}
		expectedURL   string
	}{
		{
			name:        "No config file",
			setupConfig: false,
			expectedURL: "https://app.tugboatlogic.com",
		},
		{
			name:        "Config with custom URL",
			setupConfig: true,
			configContent: map[string]interface{}{
				"tugboat": map[string]interface{}{
					"base_url": "https://custom.tugboatlogic.com",
				},
			},
			expectedURL: "https://custom.tugboatlogic.com",
		},
		{
			name:          "Empty config",
			setupConfig:   true,
			configContent: map[string]interface{}{},
			expectedURL:   "https://app.tugboatlogic.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()
			oldDir, _ := os.Getwd()
			_ = os.Chdir(tmpDir)
			defer func() { _ = os.Chdir(oldDir) }()

			if tt.setupConfig {
				data, err := yaml.Marshal(tt.configContent)
				require.NoError(t, err)
				err = os.WriteFile(".grctool.yaml", data, 0600)
				require.NoError(t, err)
			}

			// Load config
			cfg, err := loadConfigForAuth()
			require.NoError(t, err)
			assert.Equal(t, tt.expectedURL, cfg.Tugboat.BaseURL)
		})
	}
}

func TestGetConfigPath(t *testing.T) {
	tests := []struct {
		name         string
		cfgFileValue string
		expected     string
	}{
		{
			name:         "Default path",
			cfgFileValue: "",
			expected:     ".grctool.yaml",
		},
		{
			name:         "Custom path",
			cfgFileValue: "/custom/path/config.yaml",
			expected:     "/custom/path/config.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldCfgFile := cfgFile
			cfgFile = tt.cfgFileValue
			defer func() { cfgFile = oldCfgFile }()

			result := getConfigPath()
			assert.Equal(t, tt.expected, result)
		})
	}
}
