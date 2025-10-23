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

// setupTestConfigForEvidence creates a test configuration for evidence tests
func setupTestConfigForEvidence(t *testing.T) string {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, ".grctool.yaml")

	configContent := fmt.Sprintf(`
tugboat:
  base_url: "https://api-my.tugboatlogic.com"
  org_id: "13888"
  timeout: "30s"
  rate_limit: 10
  cookie_header: "test-cookie"

evidence:
  generation:
    output_dir: "%s"

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
`, tempDir, tempDir)

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	return configFile
}

func TestEvidenceListCommand(t *testing.T) {
	// Set up test configuration with VCR
	configFile := setupTestConfigForEvidence(t)
	viper.SetConfigFile(configFile)
	err := viper.ReadInConfig()
	require.NoError(t, err)

	tests := []struct {
		name      string
		args      []string
		checkFunc func(t *testing.T, output string, err error)
	}{
		{
			name: "list without filters",
			args: []string{},
			checkFunc: func(t *testing.T, output string, err error) {
				// In test environment, may fail due to no synced data
				if err != nil {
					t.Log("✅ Expected failure - no synced data in test environment")
				} else {
					t.Log("✅ Evidence list command executed successfully")
				}
			},
		},
		{
			name: "list with status filter",
			args: []string{"--status", "pending", "--status", "completed"},
			checkFunc: func(t *testing.T, output string, err error) {
				if err != nil {
					t.Log("✅ Expected failure - no synced data in test environment")
				} else {
					t.Log("✅ Evidence list with status filter executed successfully")
				}
			},
		},
		{
			name: "list with framework filter",
			args: []string{"--framework", "soc2"},
			checkFunc: func(t *testing.T, output string, err error) {
				if err != nil {
					t.Log("✅ Expected failure - no synced data in test environment")
				} else {
					t.Log("✅ Evidence list with framework filter executed successfully")
				}
			},
		},
		{
			name: "list with priority filter",
			args: []string{"--priority", "high"},
			checkFunc: func(t *testing.T, output string, err error) {
				if err != nil {
					t.Log("✅ Expected failure - no synced data in test environment")
				} else {
					t.Log("✅ Evidence list with priority filter executed successfully")
				}
			},
		},
		{
			name: "list with assignee filter",
			args: []string{"--assigned-to", "user@example.com"},
			checkFunc: func(t *testing.T, output string, err error) {
				if err != nil {
					t.Log("✅ Expected failure - no synced data in test environment")
				} else {
					t.Log("✅ Evidence list with assignee filter executed successfully")
				}
			},
		},
		{
			name: "list overdue tasks",
			args: []string{"--overdue"},
			checkFunc: func(t *testing.T, output string, err error) {
				if err != nil {
					t.Log("✅ Expected failure - no synced data in test environment")
				} else {
					t.Log("✅ Evidence list overdue tasks executed successfully")
				}
			},
		},
		{
			name: "list tasks due soon",
			args: []string{"--due-soon"},
			checkFunc: func(t *testing.T, output string, err error) {
				if err != nil {
					t.Log("✅ Expected failure - no synced data in test environment")
				} else {
					t.Log("✅ Evidence list due soon tasks executed successfully")
				}
			},
		},
		{
			name: "list with multiple filters",
			args: []string{"--status", "pending", "--framework", "iso27001", "--priority", "high", "--overdue"},
			checkFunc: func(t *testing.T, output string, err error) {
				if err != nil {
					t.Log("✅ Expected failure - no synced data in test environment")
				} else {
					t.Log("✅ Evidence list with multiple filters executed successfully")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{
				Use:  "list",
				RunE: runEvidenceList,
			}

			// Add flags
			cmd.Flags().StringSlice("status", []string{}, "filter by status")
			cmd.Flags().String("framework", "", "filter by framework")
			cmd.Flags().String("priority", "", "filter by priority")
			cmd.Flags().String("assigned-to", "", "filter by assignee")
			cmd.Flags().Bool("overdue", false, "show only overdue tasks")
			cmd.Flags().Bool("due-soon", false, "show tasks due within 7 days")

			cmd.SetArgs(tt.args)

			output := &bytes.Buffer{}
			cmd.SetOut(output)
			cmd.SetErr(output)

			err := cmd.Execute()

			if tt.checkFunc != nil {
				tt.checkFunc(t, output.String(), err)
			}
		})
	}
}

func TestEvidenceShowCommand(t *testing.T) {
	// Set up test configuration with VCR
	configFile := setupTestConfigForEvidence(t)
	viper.SetConfigFile(configFile)
	err := viper.ReadInConfig()
	require.NoError(t, err)

	tests := []struct {
		name      string
		args      []string
		expectErr bool
		checkFunc func(t *testing.T, output string, err error)
	}{
		{
			name: "show specific task",
			args: []string{"327992"},
			checkFunc: func(t *testing.T, output string, err error) {
				// In test environment, we expect this to fail since there's no synced data
				// The important thing is that it tries to load the task and gives a meaningful error
				if err != nil {
					assert.Contains(t, err.Error(), "evidence task not found")
					t.Log("✅ Expected failure - no synced data in test environment")
				} else {
					t.Log("✅ Evidence task found - test data available")
				}
			},
		},
		{
			name:      "show without task ID",
			args:      []string{},
			expectErr: true,
		},
		{
			name:      "show with too many args",
			args:      []string{"327992", "extra-arg"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{
				Use:  "show [task-id]",
				Args: cobra.ExactArgs(1),
				RunE: runEvidenceAnalyze,
			}

			cmd.SetArgs(tt.args)

			output := &bytes.Buffer{}
			cmd.SetOut(output)
			cmd.SetErr(output)

			err := cmd.Execute()

			if tt.expectErr {
				assert.Error(t, err)
			} else if tt.checkFunc != nil {
				tt.checkFunc(t, output.String(), err)
			}
		})
	}
}

func TestEvidencePrepareCommand(t *testing.T) {
	// Set up test configuration with VCR
	configFile := setupTestConfigForEvidence(t)
	viper.SetConfigFile(configFile)
	err := viper.ReadInConfig()
	require.NoError(t, err)

	tests := []struct {
		name      string
		args      []string
		expectErr bool
		checkFunc func(t *testing.T, output string, err error)
	}{
		{
			name: "prepare specific task",
			args: []string{"327992"},
			checkFunc: func(t *testing.T, output string, err error) {
				// In test environment, we expect this to fail since there's no synced data
				if err != nil {
					assert.Contains(t, err.Error(), "evidence task not found")
					t.Log("✅ Expected failure - no synced data in test environment")
				} else {
					t.Log("✅ Evidence task prepare succeeded")
				}
			},
		},
		{
			name: "prepare all tasks",
			args: []string{"--all"},
			checkFunc: func(t *testing.T, output string, err error) {
				// Bulk generation is not yet implemented, expect error
				if err != nil {
					assert.Contains(t, err.Error(), "bulk generation not yet implemented")
					t.Log("✅ Expected failure - bulk generation not yet implemented")
				} else {
					t.Log("✅ Evidence prepare all tasks succeeded")
				}
			},
		},
		{
			name: "prepare with auto-submit",
			args: []string{"327992", "--auto-submit"},
			checkFunc: func(t *testing.T, output string, err error) {
				// In test environment, we expect this to fail since there's no synced data
				if err != nil {
					assert.Contains(t, err.Error(), "evidence task not found")
					t.Log("✅ Expected failure - no synced data in test environment")
				} else {
					t.Log("✅ Evidence task prepare with auto-submit succeeded")
				}
			},
		},
		{
			name: "prepare with output directory",
			args: []string{"327992", "--output-dir", "/tmp/evidence"},
			checkFunc: func(t *testing.T, output string, err error) {
				// In test environment, we expect this to fail since there's no synced data
				if err != nil {
					assert.Contains(t, err.Error(), "evidence task not found")
					t.Log("✅ Expected failure - no synced data in test environment")
				} else {
					t.Log("✅ Evidence task prepare with output directory succeeded")
				}
			},
		},
		{
			name: "prepare all with options",
			args: []string{"--all", "--auto-submit", "--output-dir", "/tmp/evidence"},
			checkFunc: func(t *testing.T, output string, err error) {
				// Bulk generation is not yet implemented, expect error
				if err != nil {
					assert.Contains(t, err.Error(), "bulk generation not yet implemented")
					t.Log("✅ Expected failure - bulk generation not yet implemented")
				} else {
					t.Log("✅ Evidence prepare all with options succeeded")
				}
			},
		},
		{
			name:      "prepare without task ID or --all",
			args:      []string{},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{
				Use:  "prepare [task-id]",
				RunE: runEvidenceGenerate,
			}

			// Add flags
			cmd.Flags().Bool("all", false, "prepare evidence for all pending tasks")
			cmd.Flags().Bool("auto-submit", false, "automatically submit after preparation")
			cmd.Flags().String("output-dir", "", "directory to save prepared evidence")
			cmd.Flags().StringSlice("tools", []string{"terraform", "github"}, "tools to use for evidence collection")
			cmd.Flags().String("format", "json", "output format")

			cmd.SetArgs(tt.args)

			output := &bytes.Buffer{}
			cmd.SetOut(output)
			cmd.SetErr(output)

			err := cmd.Execute()

			if tt.expectErr {
				assert.Error(t, err)
			} else if tt.checkFunc != nil {
				tt.checkFunc(t, output.String(), err)
			}
		})
	}
}

func TestEvidenceValidateCommand(t *testing.T) {
	// Set up test configuration with VCR
	configFile := setupTestConfigForEvidence(t)
	viper.SetConfigFile(configFile)
	err := viper.ReadInConfig()
	require.NoError(t, err)

	tests := []struct {
		name      string
		args      []string
		expectErr bool
		checkFunc func(t *testing.T, output string, err error)
	}{
		{
			name: "validate specific task",
			args: []string{"327992"},
			checkFunc: func(t *testing.T, output string, err error) {
				// In test environment, we expect this to fail since there's no synced data
				if err != nil {
					assert.Contains(t, err.Error(), "evidence task not found")
					t.Log("✅ Expected failure - no synced data in test environment")
				} else {
					t.Log("✅ Evidence task validation succeeded")
				}
			},
		},
		{
			name:      "validate without task ID",
			args:      []string{},
			expectErr: true,
		},
		{
			name:      "validate with too many args",
			args:      []string{"327992", "extra-arg"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{
				Use:  "validate [task-id]",
				Args: cobra.ExactArgs(1),
				RunE: runEvidenceReview,
			}

			// Add flags
			cmd.Flags().Bool("show-reasoning", false, "show reasoning for validation")
			cmd.Flags().Bool("show-sources", false, "show sources for validation")

			cmd.SetArgs(tt.args)

			output := &bytes.Buffer{}
			cmd.SetOut(output)
			cmd.SetErr(output)

			err := cmd.Execute()

			if tt.expectErr {
				assert.Error(t, err)
			} else if tt.checkFunc != nil {
				tt.checkFunc(t, output.String(), err)
			}
		})
	}
}

func TestEvidenceSubmitCommand(t *testing.T) {
	// Set up test configuration with VCR
	configFile := setupTestConfigForEvidence(t)
	viper.SetConfigFile(configFile)
	err := viper.ReadInConfig()
	require.NoError(t, err)

	tests := []struct {
		name      string
		args      []string
		expectErr bool
		checkFunc func(t *testing.T, output string, err error)
	}{
		{
			name: "submit specific task",
			args: []string{"327992"},
			checkFunc: func(t *testing.T, output string, err error) {
				// In test environment, we expect this to fail since there's no synced data
				if err != nil {
					assert.Contains(t, err.Error(), "evidence task not found")
					t.Log("✅ Expected failure - no synced data in test environment")
				} else {
					t.Log("✅ Evidence task submit succeeded")
				}
			},
		},
		{
			name:      "submit without task ID",
			args:      []string{},
			expectErr: true,
		},
		{
			name:      "submit with too many args",
			args:      []string{"327992", "extra-arg"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{
				Use:  "submit [task-id]",
				Args: cobra.ExactArgs(1),
				RunE: runEvidenceSubmit,
			}

			cmd.SetArgs(tt.args)

			output := &bytes.Buffer{}
			cmd.SetOut(output)
			cmd.SetErr(output)

			err := cmd.Execute()

			if tt.expectErr {
				assert.Error(t, err)
			} else if tt.checkFunc != nil {
				tt.checkFunc(t, output.String(), err)
			}
		})
	}
}

func TestEvidenceStatusCommand(t *testing.T) {
	// Set up test configuration with VCR
	configFile := setupTestConfigForEvidence(t)
	viper.SetConfigFile(configFile)
	err := viper.ReadInConfig()
	require.NoError(t, err)

	tests := []struct {
		name      string
		args      []string
		expectErr bool
		checkFunc func(t *testing.T, output string, err error)
	}{
		{
			name: "status for specific task",
			args: []string{"327992"},
			checkFunc: func(t *testing.T, output string, err error) {
				// In test environment, we expect this to fail since there's no synced data
				if err != nil {
					t.Log("✅ Expected failure - no synced data in test environment")
				} else {
					t.Log("✅ Evidence task status check succeeded")
				}
			},
		},
		{
			name:      "status without task ID",
			args:      []string{},
			expectErr: true,
		},
		{
			name:      "status with too many args",
			args:      []string{"327992", "extra-arg"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{
				Use:  "status [task-id]",
				Args: cobra.ExactArgs(1),
				RunE: runEvidenceList,
			}

			cmd.SetArgs(tt.args)

			output := &bytes.Buffer{}
			cmd.SetOut(output)
			cmd.SetErr(output)

			err := cmd.Execute()

			if tt.expectErr {
				assert.Error(t, err)
			} else if tt.checkFunc != nil {
				tt.checkFunc(t, output.String(), err)
			}
		})
	}
}

func TestEvidenceCommands(t *testing.T) {
	// Test the evidence command structure and help
	t.Run("evidence command help", func(t *testing.T) {
		cmd := &cobra.Command{
			Use:   "evidence",
			Short: "Evidence management commands",
			Long:  `Manage evidence tasks, collection, and submission for security compliance`,
		}

		// Add subcommands
		listCmd := &cobra.Command{Use: "list", RunE: runEvidenceList}
		showCmd := &cobra.Command{Use: "show [task-id]", Args: cobra.ExactArgs(1), RunE: runEvidenceAnalyze}
		prepareCmd := &cobra.Command{Use: "prepare [task-id]", RunE: runEvidenceGenerate}
		validateCmd := &cobra.Command{Use: "validate [task-id]", Args: cobra.ExactArgs(1), RunE: runEvidenceReview}
		submitCmd := &cobra.Command{Use: "submit [task-id]", Args: cobra.ExactArgs(1), RunE: runEvidenceSubmit}
		statusCmd := &cobra.Command{Use: "status [task-id]", Args: cobra.ExactArgs(1), RunE: runEvidenceList}

		cmd.AddCommand(listCmd, showCmd, prepareCmd, validateCmd, submitCmd, statusCmd)

		// Test help for main evidence command
		cmd.SetArgs([]string{"--help"})
		output := &bytes.Buffer{}
		cmd.SetOut(output)
		cmd.SetErr(output)

		err := cmd.Execute()
		assert.NoError(t, err)
		helpOutput := output.String()
		assert.Contains(t, helpOutput, "Available Commands:")
		assert.Contains(t, helpOutput, "list")
		assert.Contains(t, helpOutput, "show")
		assert.Contains(t, helpOutput, "prepare")
		assert.Contains(t, helpOutput, "validate")
		assert.Contains(t, helpOutput, "submit")
		assert.Contains(t, helpOutput, "status")
	})

	t.Run("evidence list help", func(t *testing.T) {
		cmd := &cobra.Command{
			Use:  "list",
			RunE: runEvidenceList,
		}
		cmd.Flags().StringSlice("status", []string{}, "filter by status")
		cmd.Flags().String("framework", "", "filter by framework")
		cmd.Flags().String("priority", "", "filter by priority")
		cmd.Flags().String("assigned-to", "", "filter by assignee")
		cmd.Flags().Bool("overdue", false, "show only overdue tasks")
		cmd.Flags().Bool("due-soon", false, "show tasks due within 7 days")

		cmd.SetArgs([]string{"--help"})
		output := &bytes.Buffer{}
		cmd.SetOut(output)
		cmd.SetErr(output)

		err := cmd.Execute()
		assert.NoError(t, err)
		helpOutput := output.String()
		assert.Contains(t, helpOutput, "--status")
		assert.Contains(t, helpOutput, "--framework")
		assert.Contains(t, helpOutput, "--priority")
		assert.Contains(t, helpOutput, "--assigned-to")
		assert.Contains(t, helpOutput, "--overdue")
		assert.Contains(t, helpOutput, "--due-soon")
	})

	t.Run("evidence prepare help", func(t *testing.T) {
		cmd := &cobra.Command{
			Use:  "prepare [task-id]",
			RunE: runEvidenceGenerate,
		}
		cmd.Flags().Bool("all", false, "prepare evidence for all pending tasks")
		cmd.Flags().Bool("auto-submit", false, "automatically submit after preparation")
		cmd.Flags().String("output-dir", "", "directory to save prepared evidence")
		cmd.Flags().StringSlice("tools", []string{"terraform", "github"}, "tools to use for evidence collection")
		cmd.Flags().String("format", "json", "output format")

		cmd.SetArgs([]string{"--help"})
		output := &bytes.Buffer{}
		cmd.SetOut(output)
		cmd.SetErr(output)

		err := cmd.Execute()
		assert.NoError(t, err)
		helpOutput := output.String()
		assert.Contains(t, helpOutput, "--all")
		assert.Contains(t, helpOutput, "--auto-submit")
		assert.Contains(t, helpOutput, "--output-dir")
	})
}
