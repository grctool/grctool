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

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveConfigPaths(t *testing.T) {
	tests := []struct {
		name           string
		configFileUsed string
		inputConfig    *Config
		expectedConfig *Config
		description    string
	}{
		{
			name:           "no config file - paths unchanged",
			configFileUsed: "",
			inputConfig: &Config{
				Storage: StorageConfig{
					DataDir: "./data",
				},
			},
			expectedConfig: &Config{
				Storage: StorageConfig{
					DataDir: "./data", // Unchanged when no config file
				},
			},
			description: "When no config file is loaded, relative paths remain as-is",
		},
		{
			name:           "relative storage paths resolved",
			configFileUsed: "/home/user/project/.grctool.yaml",
			inputConfig: &Config{
				Storage: StorageConfig{
					DataDir:      "../isms",
					LocalDataDir: "./local_data",
					CacheDir:     "./.cache",
				},
			},
			expectedConfig: &Config{
				Storage: StorageConfig{
					DataDir:      "/home/user/isms",
					LocalDataDir: "/home/user/project/local_data",
					CacheDir:     "/home/user/project/.cache",
				},
			},
			description: "Relative storage paths are resolved relative to config file directory",
		},
		{
			name:           "absolute paths unchanged",
			configFileUsed: "/home/user/project/.grctool.yaml",
			inputConfig: &Config{
				Storage: StorageConfig{
					DataDir:      "/absolute/path/data",
					LocalDataDir: "/absolute/path/local",
					CacheDir:     "/absolute/path/cache",
				},
			},
			expectedConfig: &Config{
				Storage: StorageConfig{
					DataDir:      "/absolute/path/data",
					LocalDataDir: "/absolute/path/local",
					CacheDir:     "/absolute/path/cache",
				},
			},
			description: "Absolute paths are not modified",
		},
		{
			name:           "evidence paths resolved",
			configFileUsed: "/project/.grctool.yaml",
			inputConfig: &Config{
				Evidence: EvidenceConfig{
					Generation: GenerationConfig{
						OutputDir:       "./evidence/generated",
						PromptDir:       "./evidence/prompts",
						SummaryCacheDir: "./.cache/summaries",
					},
					Terraform: TerraformConfig{
						AtmosPath: "../terraform/atmos",
					},
					Tools: ToolsConfig{
						Terraform: TerraformToolConfig{
							ScanPaths: []string{"./terraform", "../infra"},
						},
						GoogleDocs: GoogleDocsToolConfig{
							CredentialsFile: "./credentials.json",
						},
					},
				},
			},
			expectedConfig: &Config{
				Evidence: EvidenceConfig{
					Generation: GenerationConfig{
						OutputDir:       "/project/evidence/generated",
						PromptDir:       "/project/evidence/prompts",
						SummaryCacheDir: "/project/.cache/summaries",
					},
					Terraform: TerraformConfig{
						AtmosPath: "/terraform/atmos",
					},
					Tools: ToolsConfig{
						Terraform: TerraformToolConfig{
							ScanPaths: []string{"/project/terraform", "/infra"},
						},
						GoogleDocs: GoogleDocsToolConfig{
							CredentialsFile: "/project/credentials.json",
						},
					},
				},
			},
			description: "Evidence-related paths are resolved correctly",
		},
		{
			name:           "auth and logging paths resolved",
			configFileUsed: "/test/.grctool.yaml",
			inputConfig: &Config{
				Auth: AuthConfig{
					CacheDir: "./.cache/auth",
				},
				Logging: LoggingConfig{
					Loggers: map[string]LoggerConfig{
						"file": {
							Enabled:  true,
							Level:    "info",
							Format:   "text",
							Output:   "file",
							FilePath: "./grctool.log",
						},
					},
				},
			},
			expectedConfig: &Config{
				Auth: AuthConfig{
					CacheDir: "/test/.cache/auth",
				},
				Logging: LoggingConfig{
					Loggers: map[string]LoggerConfig{
						"file": {
							Enabled:  true,
							Level:    "info",
							Format:   "text",
							Output:   "file",
							FilePath: "/test/grctool.log",
						},
					},
				},
			},
			description: "Auth and logging paths are resolved correctly",
		},
		{
			name:           "empty paths skipped",
			configFileUsed: "/project/.grctool.yaml",
			inputConfig: &Config{
				Storage: StorageConfig{
					DataDir:      "",
					LocalDataDir: "",
					CacheDir:     "",
				},
			},
			expectedConfig: &Config{
				Storage: StorageConfig{
					DataDir:      "",
					LocalDataDir: "",
					CacheDir:     "",
				},
			},
			description: "Empty paths are not modified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock viper.ConfigFileUsed()
			originalConfigFileUsed := viper.ConfigFileUsed
			defer func() {
				// Reset viper state
				viper.Reset()
			}()

			// Set up viper to return our test config file
			if tt.configFileUsed != "" {
				viper.SetConfigFile(tt.configFileUsed)
			}

			// Call the function
			err := resolveConfigPaths(tt.inputConfig)
			require.NoError(t, err, "resolveConfigPaths should not return error")

			// Compare results
			assert.Equal(t, tt.expectedConfig.Storage.DataDir, tt.inputConfig.Storage.DataDir,
				"DataDir: %s", tt.description)
			assert.Equal(t, tt.expectedConfig.Storage.LocalDataDir, tt.inputConfig.Storage.LocalDataDir,
				"LocalDataDir: %s", tt.description)
			assert.Equal(t, tt.expectedConfig.Storage.CacheDir, tt.inputConfig.Storage.CacheDir,
				"CacheDir: %s", tt.description)

			if tt.expectedConfig.Auth.CacheDir != "" || tt.inputConfig.Auth.CacheDir != "" {
				assert.Equal(t, tt.expectedConfig.Auth.CacheDir, tt.inputConfig.Auth.CacheDir,
					"Auth.CacheDir: %s", tt.description)
			}

			if tt.expectedConfig.Evidence.Generation.OutputDir != "" || tt.inputConfig.Evidence.Generation.OutputDir != "" {
				assert.Equal(t, tt.expectedConfig.Evidence.Generation.OutputDir, tt.inputConfig.Evidence.Generation.OutputDir,
					"Evidence.Generation.OutputDir: %s", tt.description)
			}

			if tt.expectedConfig.Evidence.Terraform.AtmosPath != "" || tt.inputConfig.Evidence.Terraform.AtmosPath != "" {
				assert.Equal(t, tt.expectedConfig.Evidence.Terraform.AtmosPath, tt.inputConfig.Evidence.Terraform.AtmosPath,
					"Evidence.Terraform.AtmosPath: %s", tt.description)
			}

			if len(tt.expectedConfig.Evidence.Tools.Terraform.ScanPaths) > 0 {
				assert.Equal(t, tt.expectedConfig.Evidence.Tools.Terraform.ScanPaths, tt.inputConfig.Evidence.Tools.Terraform.ScanPaths,
					"Evidence.Tools.Terraform.ScanPaths: %s", tt.description)
			}

			// Restore original state
			_ = originalConfigFileUsed
		})
	}
}

func TestResolveConfigPathsIntegration(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".grctool.yaml")

	configContent := `
tugboat:
  base_url: "https://api-test.tugboatlogic.com"
  org_id: "test-org"

storage:
  data_dir: "../isms"
  local_data_dir: "./local_data"
  cache_dir: "./.cache"

auth:
  cache_dir: "./.cache/auth"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Reset viper and load config
	viper.Reset()
	viper.SetConfigFile(configPath)
	err = viper.ReadInConfig()
	require.NoError(t, err)

	// Load config (which calls resolveConfigPaths)
	cfg, err := Load()
	require.NoError(t, err)

	// Verify paths were resolved
	expectedDataDir := filepath.Join(filepath.Dir(tmpDir), "isms")
	expectedLocalDataDir := filepath.Join(tmpDir, "local_data")
	expectedCacheDir := filepath.Join(tmpDir, ".cache")

	assert.Equal(t, expectedDataDir, cfg.Storage.DataDir, "DataDir should be resolved relative to config file")
	assert.Equal(t, expectedLocalDataDir, cfg.Storage.LocalDataDir, "LocalDataDir should be resolved relative to config file")
	assert.Equal(t, expectedCacheDir, cfg.Storage.CacheDir, "CacheDir should be resolved relative to config file")
}
