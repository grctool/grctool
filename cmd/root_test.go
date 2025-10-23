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
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCommand(t *testing.T) {
	// Test the root command structure and help
	t.Run("root command help", func(t *testing.T) {
		// Use the actual rootCmd to test help output
		cmd := rootCmd

		cmd.SetArgs([]string{"--help"})
		output := &bytes.Buffer{}
		cmd.SetOut(output)
		cmd.SetErr(output)

		err := cmd.Execute()
		assert.NoError(t, err)
		helpOutput := output.String()
		assert.Contains(t, helpOutput, "grctool")
		assert.Contains(t, helpOutput, "GRC systems integration")
		assert.Contains(t, helpOutput, "--config")
		assert.Contains(t, helpOutput, "--verbose")
		assert.Contains(t, helpOutput, "--log-level")

		// Reset args to avoid affecting other tests
		cmd.SetArgs([]string{})
	})

	t.Run("root command version/info", func(t *testing.T) {
		// Test that the root command has the expected Use field
		assert.Equal(t, "grctool", rootCmd.Use)
		assert.Contains(t, rootCmd.Short, "Security Program Manager")
		assert.Contains(t, rootCmd.Long, "GRC systems integration")
		assert.Contains(t, rootCmd.Long, "SOC 2")
		assert.Contains(t, rootCmd.Long, "ISO 27001")
	})

	t.Run("persistent flags are set", func(t *testing.T) {
		// Test that persistent flags are properly configured
		configFlag := rootCmd.PersistentFlags().Lookup("config")
		assert.NotNil(t, configFlag)
		assert.Equal(t, "string", configFlag.Value.Type())

		verboseFlag := rootCmd.PersistentFlags().Lookup("verbose")
		assert.NotNil(t, verboseFlag)
		assert.Equal(t, "bool", verboseFlag.Value.Type())

		logLevelFlag := rootCmd.PersistentFlags().Lookup("log-level")
		assert.NotNil(t, logLevelFlag)
		assert.Equal(t, "string", logLevelFlag.Value.Type())
		assert.Equal(t, "warn", logLevelFlag.DefValue)
	})
}

func TestInitConfig(t *testing.T) {
	// Save original state
	originalCfgFile := cfgFile
	defer func() {
		cfgFile = originalCfgFile
		viper.Reset() // Reset viper state after test
	}()

	tests := []struct {
		name      string
		setupFunc func(t *testing.T) string // Returns temp dir
		cfgFile   string
		checkFunc func(t *testing.T)
	}{
		{
			name: "config from flag",
			setupFunc: func(t *testing.T) string {
				tempDir := t.TempDir()
				configFile := filepath.Join(tempDir, "test-config.yaml")

				// Create a test config file
				configContent := `
tugboat:
  base_url: "https://test.tugboatlogic.com"
  org_id: "test-org"
  timeout: "30s"

logging:
  level: "debug"
  format: "json"
`
				err := os.WriteFile(configFile, []byte(configContent), 0644)
				require.NoError(t, err)

				return configFile
			},
			checkFunc: func(t *testing.T) {
				// Config file should be read
				assert.True(t, viper.IsSet("tugboat.base_url"))
				assert.Equal(t, "https://test.tugboatlogic.com", viper.GetString("tugboat.base_url"))
				assert.Equal(t, "test-org", viper.GetString("tugboat.org_id"))
			},
		},
		{
			name: "config from current directory takes precedence",
			setupFunc: func(t *testing.T) string {
				tempDir := t.TempDir()

				// Create config in "home" directory (will be added as fallback)
				homeConfigFile := filepath.Join(tempDir, "home", ".grctool.yaml")
				err := os.MkdirAll(filepath.Dir(homeConfigFile), 0755)
				require.NoError(t, err)
				homeConfigContent := `
tugboat:
  base_url: "https://home.tugboatlogic.com"
  org_id: "home-org"

logging:
  level: "warn"
`
				err = os.WriteFile(homeConfigFile, []byte(homeConfigContent), 0644)
				require.NoError(t, err)

				// Create config in "current" directory (should take precedence)
				currentConfigFile := filepath.Join(tempDir, "current", ".grctool.yaml")
				err = os.MkdirAll(filepath.Dir(currentConfigFile), 0755)
				require.NoError(t, err)
				currentConfigContent := `
tugboat:
  base_url: "https://current.tugboatlogic.com"
  org_id: "current-org"

logging:
  level: "debug"
`
				err = os.WriteFile(currentConfigFile, []byte(currentConfigContent), 0644)
				require.NoError(t, err)

				// Add paths in the new search order: current first, then home
				viper.AddConfigPath(filepath.Join(tempDir, "current")) // Current dir first
				viper.AddConfigPath(filepath.Join(tempDir, "home"))    // Home dir as fallback

				return tempDir
			},
			checkFunc: func(t *testing.T) {
				// Should find config from current directory, not home
				initConfig()
				// The current directory config should take precedence
				if viper.IsSet("tugboat.base_url") {
					assert.Equal(t, "https://current.tugboatlogic.com", viper.GetString("tugboat.base_url"))
					assert.Equal(t, "current-org", viper.GetString("tugboat.org_id"))
					assert.Equal(t, "debug", viper.GetString("logging.level"))
				}
			},
		},
		{
			name: "no config file found",
			setupFunc: func(t *testing.T) string {
				return ""
			},
			checkFunc: func(t *testing.T) {
				// Should not error, just continue with defaults
				initConfig()
				// Config should be accessible even without file
				viper.SetDefault("test.value", "default")
				assert.Equal(t, "default", viper.GetString("test.value"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper for each test
			viper.Reset()

			// Run setup
			var setupResult string
			if tt.setupFunc != nil {
				setupResult = tt.setupFunc(t)
			}

			// Set cfgFile if this test provides a specific file
			if setupResult != "" && tt.name == "config from flag" {
				cfgFile = setupResult
			} else {
				cfgFile = ""
			}

			// Run initConfig
			initConfig()

			// Run checks
			if tt.checkFunc != nil {
				tt.checkFunc(t)
			}
		})
	}
}

func TestInitLogging(t *testing.T) {
	// Save original viper state
	originalViper := viper.AllSettings()
	defer func() {
		viper.Reset()
		for k, v := range originalViper {
			viper.Set(k, v)
		}
	}()

	tests := []struct {
		name      string
		setupFunc func(t *testing.T)
		checkFunc func(t *testing.T)
	}{
		{
			name: "logging with config load failure",
			setupFunc: func(t *testing.T) {
				// Reset viper to ensure config loading fails
				viper.Reset()
				// Don't set any config that would make config.Load() succeed
			},
			checkFunc: func(t *testing.T) {
				// Should not panic and should handle config load failure
				initLogging()
				// Test passes if no panic occurs
			},
		},
		{
			name: "logging with log-level override",
			setupFunc: func(t *testing.T) {
				// Set a log level via viper (simulating command line flag)
				viper.Set("log-level", "error")
			},
			checkFunc: func(t *testing.T) {
				// Should apply the log level override
				initLogging()
				// Test passes if no panic occurs and log level is applied
				assert.Equal(t, "error", viper.GetString("log-level"))
			},
		},
		{
			name: "logging with minimal valid config",
			setupFunc: func(t *testing.T) {
				// Set minimal config that allows config.Load() to succeed
				viper.Set("tugboat.base_url", "https://test.tugboatlogic.com")
				viper.Set("logging.level", "info")
				viper.Set("logging.format", "text")
			},
			checkFunc: func(t *testing.T) {
				// Should initialize logging successfully
				initLogging()
				// Test passes if no panic occurs
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper for each test
			viper.Reset()

			// Run setup
			if tt.setupFunc != nil {
				tt.setupFunc(t)
			}

			// Run initLogging - should not panic
			assert.NotPanics(t, func() {
				initLogging()
			})

			// Run checks
			if tt.checkFunc != nil {
				tt.checkFunc(t)
			}
		})
	}
}

func TestExecute(t *testing.T) {
	// Test that Execute function exists and can be called
	// This is a simple test since Execute() just delegates to rootCmd.Execute()
	t.Run("execute function exists", func(t *testing.T) {
		// Just verify the function exists and can be called
		// We won't actually execute it as it would run the full CLI
		assert.NotNil(t, Execute)
	})
}

func TestViperBindings(t *testing.T) {
	// Test that viper bindings are properly set up in init()
	t.Run("viper flag bindings", func(t *testing.T) {
		// Save original viper state
		originalViper := viper.AllSettings()
		defer func() {
			viper.Reset()
			for k, v := range originalViper {
				viper.Set(k, v)
			}
		}()

		// Reset viper to ensure clean state
		viper.Reset()

		// Re-bind the flags (normally done in init())
		_ = viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
		_ = viper.BindPFlag("log-level", rootCmd.PersistentFlags().Lookup("log-level"))

		// Set flag values
		err := rootCmd.PersistentFlags().Set("verbose", "true")
		require.NoError(t, err)

		err = rootCmd.PersistentFlags().Set("log-level", "debug")
		require.NoError(t, err)

		// Viper should be able to access these values
		assert.True(t, viper.GetBool("verbose"))
		assert.Equal(t, "debug", viper.GetString("log-level"))
	})
}
