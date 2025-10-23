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
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestConfig creates a test configuration file and initializes viper
func setupTestConfig(t *testing.T) string {
	// Create temporary directory for test
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, ".grctool.yaml")

	// Create minimal test configuration
	configContent := `
tugboat:
  base_url: "https://api-my.tugboatlogic.com"
  org_id: "13888"
  timeout: "30s"
  rate_limit: 10
  cookie_header: "test-cookie"

evidence:
  terraform:
    atmos_path: "%s"
    include_patterns: ["*.tf"]
    exclude_patterns: ["*.tfstate"]

storage:
  data_dir: "%s"

vcr:
  enabled: true
  mode: "playback"
  cassette_dir: "../internal/tugboat/testdata/vcr_cassettes"
  sanitize_headers: true
  sanitize_params: true
  redact_headers: ["authorization", "cookie", "x-api-key", "token"]
  redact_params: ["api_key", "token", "password", "secret"]
  match_method: true
  match_uri: true
  match_query: false
  match_headers: false
  match_body: false

logging:
  level: "info"
  format: "text"
`
	// Create a dummy terraform directory for the test
	terraformDir := filepath.Join(tempDir, "terraform")
	err := os.MkdirAll(terraformDir, 0755)
	require.NoError(t, err)

	configContent = fmt.Sprintf(configContent, terraformDir, tempDir)

	err = os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Reset viper for clean test state
	viper.Reset()
	viper.SetConfigFile(configFile)
	err = viper.ReadInConfig()
	require.NoError(t, err)

	return tempDir
}

func TestSyncCommand(t *testing.T) {
	// Test sync command flag parsing and basic behavior
	tests := []struct {
		name      string
		args      []string
		expectErr bool
		errMsg    string
	}{
		{
			name:      "help flag",
			args:      []string{"--help"},
			expectErr: false, // --help exits successfully
		},
		{
			name:      "invalid flag",
			args:      []string{"--invalid-flag"},
			expectErr: true,
		},
		{
			name:      "policies flag",
			args:      []string{"--policies", "--dry-run"},
			expectErr: true, // Will fail without valid config but flags should parse
			errMsg:    "configuration",
		},
		{
			name:      "controls flag",
			args:      []string{"--controls", "--dry-run"},
			expectErr: true, // Will fail without valid config but flags should parse
			errMsg:    "configuration",
		},
		{
			name:      "evidence flag",
			args:      []string{"--evidence", "--dry-run"},
			expectErr: true, // Will fail without valid config but flags should parse
			errMsg:    "configuration",
		},
		{
			name:      "procedures flag",
			args:      []string{"--procedures"},
			expectErr: true, // Will fail without valid config but flags should parse
			errMsg:    "configuration",
		},
		{
			name:      "dry run flag",
			args:      []string{"--dry-run"},
			expectErr: true, // Will fail without valid config but flags should parse
			errMsg:    "configuration",
		},
		{
			name:      "force flag",
			args:      []string{"--force", "--dry-run"},
			expectErr: true, // Will fail without valid config but flags should parse
			errMsg:    "configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new sync command
			cmd := &cobra.Command{
				Use:  "sync",
				RunE: runSync,
			}

			// Add all the flags that sync command expects
			cmd.Flags().Bool("incremental", false, "perform incremental sync only")
			cmd.Flags().Bool("policies", false, "sync policies only")
			cmd.Flags().Bool("procedures", false, "sync procedures only")
			cmd.Flags().Bool("evidence", false, "sync evidence tasks only")
			cmd.Flags().Bool("controls", false, "sync controls only")
			cmd.Flags().Bool("dry-run", false, "show what would be synced without making changes")
			cmd.Flags().Bool("force", false, "force full sync even if cache is recent")

			// Set command args
			cmd.SetArgs(tt.args)

			// Capture output
			output := &bytes.Buffer{}
			cmd.SetOut(output)
			cmd.SetErr(output)

			// Execute command
			err := cmd.Execute()

			if tt.expectErr {
				assert.Error(t, err, "Expected error for args: %v", tt.args)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg, "Expected error message to contain: %s", tt.errMsg)
				}
			} else {
				// For help flag, it exits successfully
				if contains(tt.args, "--help") {
					assert.Contains(t, output.String(), "Usage:")
				}
			}
		})
	}
}

// TODO: Update these tests to use the new service layer architecture
// These tests were testing old helper functions that have been moved to services

/*
func TestGetSyncSummary(t *testing.T) {
	// Test moved to service layer - use services.SyncService in future tests
}

func TestGetEvidenceTaskSyncSummary(t *testing.T) {
	// Test moved to service layer - use services.SyncService in future tests
}

func TestGetControlSyncSummary(t *testing.T) {
	// Test moved to service layer - use services.SyncService in future tests
}
*/

// Helper functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
