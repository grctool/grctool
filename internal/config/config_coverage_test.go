// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoragePaths_WithDefaults(t *testing.T) {
	t.Parallel()

	t.Run("all defaults when empty", func(t *testing.T) {
		t.Parallel()
		sp := StoragePaths{}
		result := sp.WithDefaults()

		assert.Equal(t, "docs", result.Docs)
		assert.Equal(t, "evidence", result.Evidence)
		assert.Equal(t, ".cache", result.Cache)
		assert.Equal(t, "prompts", result.Prompts)
		assert.Equal(t, filepath.Join("docs", "policies", "json"), result.PoliciesJSON)
		assert.Equal(t, filepath.Join("docs", "policies", "markdown"), result.PoliciesMarkdown)
		assert.Equal(t, filepath.Join("docs", "controls", "json"), result.ControlsJSON)
		assert.Equal(t, filepath.Join("docs", "controls", "markdown"), result.ControlsMarkdown)
		assert.Equal(t, filepath.Join("docs", "evidence_tasks", "json"), result.EvidenceTasksJSON)
		assert.Equal(t, filepath.Join("docs", "evidence_tasks", "markdown"), result.EvidenceTasksMarkdown)
		assert.Equal(t, filepath.Join("docs", "evidence_prompts"), result.EvidencePrompts)
		assert.Equal(t, filepath.Join(".cache", "prompts"), result.PromptCache)
		assert.Equal(t, filepath.Join(".cache", "summaries"), result.SummaryCache)
		assert.Equal(t, filepath.Join(".cache", "tool_outputs"), result.ToolCache)
		assert.Equal(t, filepath.Join(".cache", "relationships"), result.RelationshipCache)
		assert.Equal(t, filepath.Join(".cache", "validations"), result.ValidationCache)
	})

	t.Run("preserves custom values", func(t *testing.T) {
		t.Parallel()
		sp := StoragePaths{
			Docs:         "custom_docs",
			Evidence:     "custom_evidence",
			Cache:        "custom_cache",
			PoliciesJSON: "my/policies",
		}
		result := sp.WithDefaults()

		assert.Equal(t, "custom_docs", result.Docs)
		assert.Equal(t, "custom_evidence", result.Evidence)
		assert.Equal(t, "custom_cache", result.Cache)
		assert.Equal(t, "my/policies", result.PoliciesJSON)
		// Others should still be filled in based on custom top-level values
		assert.Equal(t, filepath.Join("custom_docs", "controls", "json"), result.ControlsJSON)
		assert.Equal(t, filepath.Join("custom_cache", "prompts"), result.PromptCache)
	})
}

func TestStoragePaths_ResolveRelativeTo(t *testing.T) {
	t.Parallel()
	baseDir := "/base/dir"

	t.Run("resolves relative paths", func(t *testing.T) {
		t.Parallel()
		sp := StoragePaths{
			Docs:         "docs",
			Evidence:     "evidence",
			Cache:        ".cache",
			Prompts:      "prompts",
			PoliciesJSON: "docs/policies/json",
			PromptCache:  ".cache/prompts",
			SummaryCache: ".cache/summaries",
			ToolCache:    ".cache/tool_outputs",
		}
		result := sp.ResolveRelativeTo(baseDir)

		assert.Equal(t, filepath.Join(baseDir, "docs"), result.Docs)
		assert.Equal(t, filepath.Join(baseDir, "evidence"), result.Evidence)
		assert.Equal(t, filepath.Join(baseDir, ".cache"), result.Cache)
		assert.Equal(t, filepath.Join(baseDir, "prompts"), result.Prompts)
		assert.Equal(t, filepath.Join(baseDir, "docs/policies/json"), result.PoliciesJSON)
	})

	t.Run("preserves absolute paths", func(t *testing.T) {
		t.Parallel()
		sp := StoragePaths{
			Docs:     "/absolute/docs",
			Evidence: "/absolute/evidence",
		}
		result := sp.ResolveRelativeTo(baseDir)

		assert.Equal(t, "/absolute/docs", result.Docs)
		assert.Equal(t, "/absolute/evidence", result.Evidence)
	})

	t.Run("leaves empty strings empty", func(t *testing.T) {
		t.Parallel()
		sp := StoragePaths{}
		result := sp.ResolveRelativeTo(baseDir)

		assert.Equal(t, "", result.Docs)
		assert.Equal(t, "", result.Evidence)
	})
}

func TestConfig_Validate_BaseURL_Required(t *testing.T) {
	t.Parallel()
	cfg := &Config{}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tugboat.base_url is required")
}

func TestConfig_Validate_InvalidAuthMode(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Tugboat: TugboatConfig{
			BaseURL:  "https://example.com",
			AuthMode: "invalid",
		},
	}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "auth_mode must be 'form' or 'browser'")
}

func TestConfig_Validate_DefaultValues(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Tugboat: TugboatConfig{
			BaseURL: "https://tugboat.example.com",
		},
	}

	err := cfg.Validate()
	require.NoError(t, err)

	assert.Equal(t, 30*time.Second, cfg.Tugboat.Timeout)
	assert.Equal(t, 10, cfg.Tugboat.RateLimit)
	assert.Equal(t, "csv", cfg.Evidence.Generation.DefaultFormat)
	assert.Equal(t, 50, cfg.Evidence.Generation.MaxToolCalls)
	assert.Equal(t, 2, cfg.Evidence.Quality.MinSources)
	// Zero is within [0,1] range, so defaults are not applied
	assert.Equal(t, 0.0, cfg.Evidence.Quality.MinCompletenessScore)
	assert.Equal(t, 0.0, cfg.Evidence.Quality.MinQualityScore)
	assert.NotEmpty(t, cfg.Storage.DataDir)
	assert.NotEmpty(t, cfg.Storage.CacheDir)
	assert.NotEmpty(t, cfg.Logging.Loggers)
}

func TestConfig_Validate_InvalidDefaultFormat(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Tugboat: TugboatConfig{
			BaseURL: "https://tugboat.example.com",
		},
		Evidence: EvidenceConfig{
			Generation: GenerationConfig{
				DefaultFormat: "xml",
			},
		},
	}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "default_format must be 'csv' or 'markdown'")
}

func TestConfig_Validate_GoogleDocsEnabled_MissingCredentials(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Tugboat: TugboatConfig{
			BaseURL: "https://tugboat.example.com",
		},
		Evidence: EvidenceConfig{
			Tools: ToolsConfig{
				GoogleDocs: GoogleDocsToolConfig{
					Enabled: true,
				},
			},
		},
	}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "credentials_file is required")
}

func TestConfig_Validate_TerraformEnabled_DefaultPaths(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Tugboat: TugboatConfig{
			BaseURL: "https://tugboat.example.com",
		},
		Evidence: EvidenceConfig{
			Tools: ToolsConfig{
				Terraform: TerraformToolConfig{
					Enabled: true,
				},
			},
		},
	}

	err := cfg.Validate()
	require.NoError(t, err)
	assert.NotEmpty(t, cfg.Evidence.Tools.Terraform.ScanPaths)
	assert.NotEmpty(t, cfg.Evidence.Tools.Terraform.IncludePatterns)
	assert.NotEmpty(t, cfg.Evidence.Tools.Terraform.ExcludePatterns)
}

func TestConfig_Validate_GitHubEnabled_DefaultMaxIssues(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Tugboat: TugboatConfig{
			BaseURL: "https://tugboat.example.com",
		},
		Evidence: EvidenceConfig{
			Tools: ToolsConfig{
				GitHub: GitHubToolConfig{
					Enabled: true,
				},
			},
		},
	}

	err := cfg.Validate()
	require.NoError(t, err)
	assert.Equal(t, 100, cfg.Evidence.Tools.GitHub.MaxIssues)
	assert.Equal(t, 30, cfg.Evidence.Tools.GitHub.RateLimit)
}

func TestConfig_Validate_InvalidTerraformPath(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Tugboat: TugboatConfig{
			BaseURL: "https://tugboat.example.com",
		},
		Evidence: EvidenceConfig{
			Terraform: TerraformConfig{
				AtmosPath: "/nonexistent/path/that/should/not/exist",
			},
		},
	}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "atmos_path does not exist")
}

func TestConfig_Validate_InvalidRepoPath(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Tugboat: TugboatConfig{
			BaseURL: "https://tugboat.example.com",
		},
		Evidence: EvidenceConfig{
			Terraform: TerraformConfig{
				RepoPath: "/nonexistent/repo/path",
			},
		},
	}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "repo_path does not exist")
}

func TestConfig_Validate_InterpolationDefaultsExtended(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Tugboat: TugboatConfig{
			BaseURL: "https://tugboat.example.com",
		},
	}

	err := cfg.Validate()
	require.NoError(t, err)

	assert.True(t, cfg.Interpolation.Enabled)
	assert.NotNil(t, cfg.Interpolation.Variables)
	flat := cfg.Interpolation.GetFlatVariables()
	assert.Equal(t, "Seventh Sense", flat["organization.name"])
}

func TestConfig_Validate_PreservesExistingInterpolation(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Tugboat: TugboatConfig{
			BaseURL: "https://tugboat.example.com",
		},
		Interpolation: InterpolationConfig{
			Variables: map[string]interface{}{
				"organization": map[string]interface{}{
					"name": "Custom Corp",
				},
			},
		},
	}

	err := cfg.Validate()
	require.NoError(t, err)

	flat := cfg.Interpolation.GetFlatVariables()
	assert.Equal(t, "Custom Corp", flat["organization.name"])
}

func TestConfig_Validate_QualityScoreDefaults(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Tugboat: TugboatConfig{
			BaseURL: "https://tugboat.example.com",
		},
		Evidence: EvidenceConfig{
			Quality: QualityConfig{
				MinCompletenessScore: -0.5,
				MinQualityScore:      1.5,
			},
		},
	}

	err := cfg.Validate()
	require.NoError(t, err)
	// Out-of-range values should be reset to defaults
	assert.Equal(t, 0.7, cfg.Evidence.Quality.MinCompletenessScore)
	assert.Equal(t, 0.8, cfg.Evidence.Quality.MinQualityScore)
}

func TestProcessEnvVars_CookieHeader(t *testing.T) {
	cfg := &Config{
		Tugboat: TugboatConfig{
			CookieHeader: "${TEST_GRCTOOL_COOKIE}",
		},
	}

	t.Setenv("TEST_GRCTOOL_COOKIE", "session=abc123")
	err := processEnvVars(cfg)
	require.NoError(t, err)
	assert.Equal(t, "session=abc123", cfg.Tugboat.CookieHeader)
}

func TestProcessEnvVars_CookieHeader_NotSet(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Tugboat: TugboatConfig{
			CookieHeader: "${NONEXISTENT_ENV_VAR_GRCTOOL}",
		},
	}

	err := processEnvVars(cfg)
	require.NoError(t, err)
	assert.Equal(t, "", cfg.Tugboat.CookieHeader)
}

func TestProcessEnvVars_GitHubToken(t *testing.T) {
	cfg := &Config{
		Evidence: EvidenceConfig{
			Tools: ToolsConfig{
				GitHub: GitHubToolConfig{
					APIToken: "${TEST_GH_TOKEN}",
				},
			},
		},
	}

	t.Setenv("TEST_GH_TOKEN", "ghp_testtoken123")
	err := processEnvVars(cfg)
	require.NoError(t, err)
	assert.Equal(t, "ghp_testtoken123", cfg.Evidence.Tools.GitHub.APIToken)
}

func TestProcessEnvVars_GoogleDocsCredentials(t *testing.T) {
	cfg := &Config{
		Evidence: EvidenceConfig{
			Tools: ToolsConfig{
				GoogleDocs: GoogleDocsToolConfig{
					CredentialsFile: "${TEST_GDOCS_CREDS}",
				},
			},
		},
	}

	t.Setenv("TEST_GDOCS_CREDS", "/path/to/creds.json")
	err := processEnvVars(cfg)
	require.NoError(t, err)
	assert.Equal(t, "/path/to/creds.json", cfg.Evidence.Tools.GoogleDocs.CredentialsFile)
}

func TestProcessEnvVars_AuthTokens(t *testing.T) {
	cfg := &Config{
		Auth: AuthConfig{
			GitHub: GitHubAuthConfig{
				Token: "${TEST_AUTH_GH}",
			},
			Tugboat: TugboatAuthConfig{
				BearerToken: "${TEST_AUTH_TUG}",
			},
		},
	}

	t.Setenv("TEST_AUTH_GH", "ghp_auth")
	t.Setenv("TEST_AUTH_TUG", "bearer_auth")
	err := processEnvVars(cfg)
	require.NoError(t, err)
	assert.Equal(t, "ghp_auth", cfg.Auth.GitHub.Token)
	assert.Equal(t, "bearer_auth", cfg.Auth.Tugboat.BearerToken)
}

func TestProcessEnvVars_Password(t *testing.T) {
	cfg := &Config{
		Tugboat: TugboatConfig{
			Password: "${TEST_TUG_PASSWORD}",
		},
	}

	t.Setenv("TEST_TUG_PASSWORD", "secret123")
	err := processEnvVars(cfg)
	require.NoError(t, err)
	assert.Equal(t, "secret123", cfg.Tugboat.Password)
}

func TestProcessEnvVars_NoSubstitution_LiteralValue(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Tugboat: TugboatConfig{
			CookieHeader: "literal-cookie-value",
		},
		Evidence: EvidenceConfig{
			Tools: ToolsConfig{
				GitHub: GitHubToolConfig{
					APIToken: "ghp_literal",
				},
			},
		},
	}

	err := processEnvVars(cfg)
	require.NoError(t, err)
	assert.Equal(t, "literal-cookie-value", cfg.Tugboat.CookieHeader)
	assert.Equal(t, "ghp_literal", cfg.Evidence.Tools.GitHub.APIToken)
}

func TestLoggerConfig_ToLoggerConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		level string
	}{
		{"trace", "trace"},
		{"debug", "debug"},
		{"info", "info"},
		{"warn", "warn"},
		{"error", "error"},
		{"default for unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			lc := &LoggerConfig{
				Level:        tt.level,
				Format:       "json",
				Output:       "stdout",
				SanitizeURLs: true,
				RedactFields: []string{"password"},
				ShowCaller:   true,
				BufferSize:   50,
			}
			result := lc.ToLoggerConfig()
			assert.NotNil(t, result)
			assert.Equal(t, "json", result.Format)
			assert.Equal(t, "stdout", result.Output)
			assert.True(t, result.SanitizeURLs)
			assert.Equal(t, 50, result.BufferSize)
		})
	}
}

func TestDefaultLoggingConfig(t *testing.T) {
	t.Parallel()
	cfg := DefaultLoggingConfig()
	assert.NotNil(t, cfg)
	assert.Contains(t, cfg.Loggers, "console")
	assert.Contains(t, cfg.Loggers, "file")
	assert.True(t, cfg.Loggers["console"].Enabled)
	assert.Equal(t, "warn", cfg.Loggers["console"].Level)
	assert.Equal(t, "info", cfg.Loggers["file"].Level)
}

func TestResolveConfigPaths_WithAbsolutePaths(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Create a dummy config file so viper has something
	cfgFile := filepath.Join(tmpDir, ".grctool.yaml")
	require.NoError(t, os.WriteFile(cfgFile, []byte("tugboat:\n  base_url: test\n"), 0644))

	cfg := &Config{
		Storage: StorageConfig{
			DataDir:  "/absolute/data",
			CacheDir: "/absolute/cache",
		},
	}

	// resolveConfigPaths won't change absolute paths
	// We call it directly but it uses viper.ConfigFileUsed which returns ""
	// so it will just return nil
	err := resolveConfigPaths(cfg)
	require.NoError(t, err)
	assert.Equal(t, "/absolute/data", cfg.Storage.DataDir)
	assert.Equal(t, "/absolute/cache", cfg.Storage.CacheDir)
}
