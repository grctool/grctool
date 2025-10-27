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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/logger"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Tugboat       TugboatConfig       `mapstructure:"tugboat" yaml:"tugboat"`
	Evidence      EvidenceConfig      `mapstructure:"evidence" yaml:"evidence"`
	Storage       StorageConfig       `mapstructure:"storage" yaml:"storage"`
	Logging       LoggingConfig       `mapstructure:"logging" yaml:"logging"`
	Interpolation InterpolationConfig `mapstructure:"interpolation" yaml:"interpolation"`
	Auth          AuthConfig          `mapstructure:"auth" yaml:"auth"`
}

// TugboatConfig holds Tugboat Logic API configuration
type TugboatConfig struct {
	BaseURL         string        `mapstructure:"base_url" yaml:"base_url"`
	OrgID           string        `mapstructure:"org_id" yaml:"org_id"`
	Timeout         time.Duration `mapstructure:"timeout" yaml:"timeout"`
	RateLimit       int           `mapstructure:"rate_limit" yaml:"rate_limit"`
	AuthMode        string        `mapstructure:"auth_mode" yaml:"auth_mode"`                 // "form" or "browser"
	CookieHeader    string        `mapstructure:"cookie_header" yaml:"cookie_header"`         // For browser auth mode
	BearerToken     string        `mapstructure:"bearer_token" yaml:"bearer_token"`           // Extracted bearer token
	AuthExpires     string        `mapstructure:"auth_expires" yaml:"auth_expires"`           // When auth expires
	LogAPIRequests  bool          `mapstructure:"log_api_requests" yaml:"log_api_requests"`   // Log HTTP request details
	LogAPIResponses bool          `mapstructure:"log_api_responses" yaml:"log_api_responses"` // Log HTTP response details

	// Custom Evidence Integration API (for evidence submission)
	// API Key should be set via TUGBOAT_API_KEY environment variable (not stored in config for security)
	Username      string            `mapstructure:"username" yaml:"username"`             // HTTP Basic Auth username
	Password      string            `mapstructure:"password" yaml:"password"`             // HTTP Basic Auth password
	CollectorURLs map[string]string `mapstructure:"collector_urls" yaml:"collector_urls"` // Evidence task ref -> collector URL mapping
}

// EvidenceConfig holds evidence collection configuration
type EvidenceConfig struct {
	Generation       GenerationConfig       `mapstructure:"generation" yaml:"generation"`
	Tools            ToolsConfig            `mapstructure:"tools" yaml:"tools"`
	Quality          QualityConfig          `mapstructure:"quality" yaml:"quality"`
	SecurityControls SecurityControlsConfig `mapstructure:"security_controls" yaml:"security_controls"`
	Terraform        TerraformConfig        `mapstructure:"terraform" yaml:"terraform"` // Terraform tool configuration
}

// GenerationConfig holds evidence generation settings
type GenerationConfig struct {
	OutputDir        string `mapstructure:"output_dir" yaml:"output_dir"`
	PromptDir        string `mapstructure:"prompt_dir" yaml:"prompt_dir"`
	IncludeReasoning bool   `mapstructure:"include_reasoning" yaml:"include_reasoning"`
	MaxToolCalls     int    `mapstructure:"max_tool_calls" yaml:"max_tool_calls"`
	DefaultFormat    string `mapstructure:"default_format" yaml:"default_format"` // csv or markdown
	SummaryCacheDir  string `mapstructure:"summary_cache_dir" yaml:"summary_cache_dir"`
	MaxSummaryLength int    `mapstructure:"max_summary_length" yaml:"max_summary_length"`
}

// ToolsConfig holds configuration for evidence collection tools
type ToolsConfig struct {
	Terraform  TerraformToolConfig  `mapstructure:"terraform" yaml:"terraform"`
	GitHub     GitHubToolConfig     `mapstructure:"github" yaml:"github"`
	GoogleDocs GoogleDocsToolConfig `mapstructure:"google_docs" yaml:"google_docs"`
}

// TerraformToolConfig holds Terraform tool configuration
type TerraformToolConfig struct {
	Enabled         bool     `mapstructure:"enabled" yaml:"enabled"`
	ScanPaths       []string `mapstructure:"scan_paths" yaml:"scan_paths"`
	IncludePatterns []string `mapstructure:"include_patterns" yaml:"include_patterns"`
	ExcludePatterns []string `mapstructure:"exclude_patterns" yaml:"exclude_patterns"`
}

// GitHubToolConfig holds GitHub integration configuration
type GitHubToolConfig struct {
	Enabled          bool   `mapstructure:"enabled" yaml:"enabled"`
	APIToken         string `mapstructure:"api_token" yaml:"api_token"`
	Repository       string `mapstructure:"repository" yaml:"repository"`
	IncludeWorkflows bool   `mapstructure:"include_workflows" yaml:"include_workflows"`
	IncludeIssues    bool   `mapstructure:"include_issues" yaml:"include_issues"`
	MaxIssues        int    `mapstructure:"max_issues" yaml:"max_issues"`
	RateLimit        int    `mapstructure:"rate_limit" yaml:"rate_limit"` // Requests per minute for Search API (default: 30)
}

// GoogleDocsToolConfig holds Google Docs integration configuration
type GoogleDocsToolConfig struct {
	Enabled         bool   `mapstructure:"enabled" yaml:"enabled"`
	CredentialsFile string `mapstructure:"credentials_file" yaml:"credentials_file"`
	SharedDriveID   string `mapstructure:"shared_drive_id" yaml:"shared_drive_id"`
}

// QualityConfig holds evidence quality settings
type QualityConfig struct {
	MinSources           int     `mapstructure:"min_sources" yaml:"min_sources"`
	RequireReasoning     bool    `mapstructure:"require_reasoning" yaml:"require_reasoning"`
	ValidateCompleteness bool    `mapstructure:"validate_completeness" yaml:"validate_completeness"`
	MinCompletenessScore float64 `mapstructure:"min_completeness_score" yaml:"min_completeness_score"`
	MinQualityScore      float64 `mapstructure:"min_quality_score" yaml:"min_quality_score"`
}

// SecurityControlsConfig holds security control mappings
type SecurityControlsConfig struct {
	SOC2 map[string]SecurityControlMapping `mapstructure:"soc2" yaml:"soc2"`
}

// SecurityControlMapping represents a security control mapping
type SecurityControlMapping struct {
	TerraformResources []string `mapstructure:"terraform_resources" yaml:"terraform_resources"`
	Description        string   `mapstructure:"description" yaml:"description"`
	Requirements       []string `mapstructure:"requirements" yaml:"requirements"`
}

// TerraformConfig holds Terraform-specific evidence collection settings
type TerraformConfig struct {
	AtmosPath       string   `mapstructure:"atmos_path" yaml:"atmos_path"`
	RepoPath        string   `mapstructure:"repo_path" yaml:"repo_path"` // Path to terraform repository for git hash tracking
	IncludePatterns []string `mapstructure:"include_patterns" yaml:"include_patterns"`
	ExcludePatterns []string `mapstructure:"exclude_patterns" yaml:"exclude_patterns"`
}

// StorageConfig holds local storage configuration
type StorageConfig struct {
	DataDir      string       `mapstructure:"data_dir" yaml:"data_dir"`             // For processed data files
	LocalDataDir string       `mapstructure:"local_data_dir" yaml:"local_data_dir"` // For offline-first local storage
	CacheDir     string       `mapstructure:"cache_dir" yaml:"cache_dir"`           // For performance cache
	Paths        StoragePaths `mapstructure:"paths" yaml:"paths,omitempty"`         // Customizable subdirectory paths
}

// StoragePaths defines customizable subdirectory paths within data_dir
type StoragePaths struct {
	// Top-level directories
	Docs     string `mapstructure:"docs" yaml:"docs,omitempty"`         // Documents directory (policies, controls, tasks)
	Evidence string `mapstructure:"evidence" yaml:"evidence,omitempty"` // Evidence output directory
	Cache    string `mapstructure:"cache" yaml:"cache,omitempty"`       // Cache directory
	Prompts  string `mapstructure:"prompts" yaml:"prompts,omitempty"`   // Custom prompts directory

	// Document subdirectories - organized by type then format (docs/{type}/{format}/)
	PoliciesJSON          string `mapstructure:"policies_json" yaml:"policies_json,omitempty"`                     // Policy JSON files
	PoliciesMarkdown      string `mapstructure:"policies_markdown" yaml:"policies_markdown,omitempty"`             // Policy markdown files
	ControlsJSON          string `mapstructure:"controls_json" yaml:"controls_json,omitempty"`                     // Control JSON files
	ControlsMarkdown      string `mapstructure:"controls_markdown" yaml:"controls_markdown,omitempty"`             // Control markdown files
	EvidenceTasksJSON     string `mapstructure:"evidence_tasks_json" yaml:"evidence_tasks_json,omitempty"`         // Evidence task JSON files
	EvidenceTasksMarkdown string `mapstructure:"evidence_tasks_markdown" yaml:"evidence_tasks_markdown,omitempty"` // Evidence task markdown files
	EvidencePrompts       string `mapstructure:"evidence_prompts" yaml:"evidence_prompts,omitempty"`               // Evidence prompts

	// Cache subdirectories (within cache/)
	PromptCache       string `mapstructure:"prompt_cache" yaml:"prompt_cache,omitempty"`             // Cached prompts
	SummaryCache      string `mapstructure:"summary_cache" yaml:"summary_cache,omitempty"`           // Cached summaries
	ToolCache         string `mapstructure:"tool_cache" yaml:"tool_cache,omitempty"`                 // Tool output cache
	RelationshipCache string `mapstructure:"relationship_cache" yaml:"relationship_cache,omitempty"` // Relationship cache
	ValidationCache   string `mapstructure:"validation_cache" yaml:"validation_cache,omitempty"`     // Validation cache
}

// WithDefaults returns a copy of StoragePaths with default values filled in for empty fields
func (sp StoragePaths) WithDefaults() StoragePaths {
	result := sp

	// Set defaults for top-level directories
	if result.Docs == "" {
		result.Docs = "docs"
	}
	if result.Evidence == "" {
		result.Evidence = "evidence"
	}
	if result.Cache == "" {
		result.Cache = ".cache"
	}
	if result.Prompts == "" {
		result.Prompts = "prompts"
	}

	// Set defaults for document subdirectories (organized by type then format)
	if result.PoliciesJSON == "" {
		result.PoliciesJSON = filepath.Join(result.Docs, "policies", "json")
	}
	if result.PoliciesMarkdown == "" {
		result.PoliciesMarkdown = filepath.Join(result.Docs, "policies", "markdown")
	}
	if result.ControlsJSON == "" {
		result.ControlsJSON = filepath.Join(result.Docs, "controls", "json")
	}
	if result.ControlsMarkdown == "" {
		result.ControlsMarkdown = filepath.Join(result.Docs, "controls", "markdown")
	}
	if result.EvidenceTasksJSON == "" {
		result.EvidenceTasksJSON = filepath.Join(result.Docs, "evidence_tasks", "json")
	}
	if result.EvidenceTasksMarkdown == "" {
		result.EvidenceTasksMarkdown = filepath.Join(result.Docs, "evidence_tasks", "markdown")
	}
	if result.EvidencePrompts == "" {
		result.EvidencePrompts = filepath.Join(result.Docs, "evidence_prompts")
	}

	// Set defaults for cache subdirectories
	if result.PromptCache == "" {
		result.PromptCache = filepath.Join(result.Cache, "prompts")
	}
	if result.SummaryCache == "" {
		result.SummaryCache = filepath.Join(result.Cache, "summaries")
	}
	if result.ToolCache == "" {
		result.ToolCache = filepath.Join(result.Cache, "tool_outputs")
	}
	if result.RelationshipCache == "" {
		result.RelationshipCache = filepath.Join(result.Cache, "relationships")
	}
	if result.ValidationCache == "" {
		result.ValidationCache = filepath.Join(result.Cache, "validations")
	}

	return result
}

// ResolveRelativeTo resolves all paths relative to the given base directory
func (sp StoragePaths) ResolveRelativeTo(baseDir string) StoragePaths {
	result := sp

	// Resolve top-level directories
	if result.Docs != "" && !filepath.IsAbs(result.Docs) {
		result.Docs = filepath.Join(baseDir, result.Docs)
	}
	if result.Evidence != "" && !filepath.IsAbs(result.Evidence) {
		result.Evidence = filepath.Join(baseDir, result.Evidence)
	}
	if result.Cache != "" && !filepath.IsAbs(result.Cache) {
		result.Cache = filepath.Join(baseDir, result.Cache)
	}
	if result.Prompts != "" && !filepath.IsAbs(result.Prompts) {
		result.Prompts = filepath.Join(baseDir, result.Prompts)
	}

	// Resolve document subdirectories
	if result.PoliciesJSON != "" && !filepath.IsAbs(result.PoliciesJSON) {
		result.PoliciesJSON = filepath.Join(baseDir, result.PoliciesJSON)
	}
	if result.PoliciesMarkdown != "" && !filepath.IsAbs(result.PoliciesMarkdown) {
		result.PoliciesMarkdown = filepath.Join(baseDir, result.PoliciesMarkdown)
	}
	if result.ControlsJSON != "" && !filepath.IsAbs(result.ControlsJSON) {
		result.ControlsJSON = filepath.Join(baseDir, result.ControlsJSON)
	}
	if result.ControlsMarkdown != "" && !filepath.IsAbs(result.ControlsMarkdown) {
		result.ControlsMarkdown = filepath.Join(baseDir, result.ControlsMarkdown)
	}
	if result.EvidenceTasksJSON != "" && !filepath.IsAbs(result.EvidenceTasksJSON) {
		result.EvidenceTasksJSON = filepath.Join(baseDir, result.EvidenceTasksJSON)
	}
	if result.EvidencePrompts != "" && !filepath.IsAbs(result.EvidencePrompts) {
		result.EvidencePrompts = filepath.Join(baseDir, result.EvidencePrompts)
	}

	// Resolve cache subdirectories
	if result.PromptCache != "" && !filepath.IsAbs(result.PromptCache) {
		result.PromptCache = filepath.Join(baseDir, result.PromptCache)
	}
	if result.SummaryCache != "" && !filepath.IsAbs(result.SummaryCache) {
		result.SummaryCache = filepath.Join(baseDir, result.SummaryCache)
	}
	if result.ToolCache != "" && !filepath.IsAbs(result.ToolCache) {
		result.ToolCache = filepath.Join(baseDir, result.ToolCache)
	}
	if result.RelationshipCache != "" && !filepath.IsAbs(result.RelationshipCache) {
		result.RelationshipCache = filepath.Join(baseDir, result.RelationshipCache)
	}
	if result.ValidationCache != "" && !filepath.IsAbs(result.ValidationCache) {
		result.ValidationCache = filepath.Join(baseDir, result.ValidationCache)
	}

	return result
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Loggers map[string]LoggerConfig `mapstructure:"loggers" yaml:"loggers,omitempty"`
}

// LoggerConfig holds configuration for a single logger instance
type LoggerConfig struct {
	Enabled       bool     `mapstructure:"enabled" yaml:"enabled"`
	Level         string   `mapstructure:"level" yaml:"level"`
	Format        string   `mapstructure:"format" yaml:"format"` // "text" or "json"
	Output        string   `mapstructure:"output" yaml:"output"` // "stdout", "stderr", "file"
	FilePath      string   `mapstructure:"file_path" yaml:"file_path,omitempty"`
	SanitizeURLs  bool     `mapstructure:"sanitize_urls" yaml:"sanitize_urls"`
	RedactFields  []string `mapstructure:"redact_fields" yaml:"redact_fields"`
	ShowCaller    bool     `mapstructure:"show_caller" yaml:"show_caller"`
	BufferSize    int      `mapstructure:"buffer_size" yaml:"buffer_size"`
	FlushInterval string   `mapstructure:"flush_interval" yaml:"flush_interval"`
}

// InterpolationConfig holds configuration for variable interpolation
type InterpolationConfig struct {
	Enabled   bool                   `mapstructure:"enabled" yaml:"enabled"`
	Variables map[string]interface{} `mapstructure:"variables" yaml:"variables"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	CacheDir string            `mapstructure:"cache_dir" yaml:"cache_dir"`
	GitHub   GitHubAuthConfig  `mapstructure:"github" yaml:"github"`
	Tugboat  TugboatAuthConfig `mapstructure:"tugboat" yaml:"tugboat"`
}

// GitHubAuthConfig holds GitHub authentication configuration
type GitHubAuthConfig struct {
	Token string `mapstructure:"token" yaml:"token"`
}

// TugboatAuthConfig holds Tugboat authentication configuration
type TugboatAuthConfig struct {
	BearerToken string `mapstructure:"bearer_token" yaml:"bearer_token"`
}

// GetFlatVariables returns a flattened map where nested keys are joined with dots
// For example: {"organization": {"name": "Acme"}} becomes {"organization.name": "Acme"}
func (ic *InterpolationConfig) GetFlatVariables() map[string]string {
	result := make(map[string]string)
	ic.flattenVariables("", ic.Variables, result)
	return result
}

// flattenVariables recursively flattens nested variables into dot notation
func (ic *InterpolationConfig) flattenVariables(prefix string, variables map[string]interface{}, result map[string]string) {
	for key, value := range variables {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch v := value.(type) {
		case string:
			result[fullKey] = v
		case map[string]interface{}:
			ic.flattenVariables(fullKey, v, result)
		case map[interface{}]interface{}:
			// Handle YAML's tendency to create map[interface{}]interface{}
			converted := make(map[string]interface{})
			for k, val := range v {
				if strKey, ok := k.(string); ok {
					converted[strKey] = val
				}
			}
			ic.flattenVariables(fullKey, converted, result)
		default:
			// Convert other types to string
			result[fullKey] = fmt.Sprintf("%v", v)
		}
	}
}

// validateConfigStructure checks for unknown or misplaced configuration keys
func validateConfigStructure() {
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		return // No config file, nothing to validate
	}

	// Read the raw YAML to detect unknown keys
	data, err := os.ReadFile(configFile)
	if err != nil {
		return // Can't read file, skip validation
	}

	var rawConfig map[string]interface{}
	if err := yaml.Unmarshal(data, &rawConfig); err != nil {
		return // Invalid YAML, will be caught by viper
	}

	// Known top-level keys
	knownKeys := map[string]bool{
		"tugboat":       true,
		"evidence":      true,
		"storage":       true,
		"logging":       true,
		"interpolation": true,
		"auth":          true,
	}

	// Check top-level keys
	for key := range rawConfig {
		if !knownKeys[key] {
			fmt.Fprintf(os.Stderr, "⚠️  Warning: Unknown config key '%s' in %s\n", key, configFile)
		}
	}

	// Check evidence structure for common mistakes
	if evidence, ok := rawConfig["evidence"].(map[string]interface{}); ok {
		// Check if user put terraform directly under evidence instead of evidence.tools.terraform
		if _, hasTerraform := evidence["terraform"]; hasTerraform {
			if _, hasTools := evidence["tools"]; !hasTools {
				fmt.Fprintf(os.Stderr, "⚠️  Warning: Found 'evidence.terraform' but expected 'evidence.tools.terraform'\n")
				fmt.Fprintf(os.Stderr, "   The Terraform tool configuration should be under 'evidence.tools.terraform'\n")
				fmt.Fprintf(os.Stderr, "   See .grctool.example.yaml for correct structure\n")
			}
		}

		// Check if tools section exists and validate its structure
		if tools, ok := evidence["tools"].(map[string]interface{}); ok {
			knownTools := map[string]bool{
				"terraform":   true,
				"github":      true,
				"google_docs": true,
			}
			for tool := range tools {
				if !knownTools[tool] {
					fmt.Fprintf(os.Stderr, "⚠️  Warning: Unknown tool '%s' under evidence.tools\n", tool)
				}
			}

			// Validate terraform tool config
			if terraform, ok := tools["terraform"].(map[string]interface{}); ok {
				if _, hasEnabled := terraform["enabled"]; !hasEnabled {
					fmt.Fprintf(os.Stderr, "⚠️  Warning: Terraform tool is missing 'enabled' field\n")
					fmt.Fprintf(os.Stderr, "   Add 'evidence.tools.terraform.enabled: true' to use Terraform tools\n")
				}
				if _, hasScanPaths := terraform["scan_paths"]; !hasScanPaths {
					fmt.Fprintf(os.Stderr, "⚠️  Warning: Terraform tool is missing 'scan_paths'\n")
					fmt.Fprintf(os.Stderr, "   Add paths to scan, e.g. 'scan_paths: [\"terraform/**/*.tf\"]'\n")
				}
			}
		}
	}
}

// Load loads the configuration from viper
func Load() (*Config, error) {
	var config Config

	// Validate structure before unmarshaling
	validateConfigStructure()

	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Process environment variable substitutions
	if err := processEnvVars(&config); err != nil {
		return nil, fmt.Errorf("failed to process environment variables: %w", err)
	}

	// Resolve relative paths relative to config file location
	if err := resolveConfigPaths(&config); err != nil {
		return nil, fmt.Errorf("failed to resolve config paths: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// LoadWithoutValidation loads the configuration without validation
// Useful for templates or incomplete configurations
func LoadWithoutValidation() (*Config, error) {
	var config Config

	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Process environment variable substitutions (ignore errors)
	_ = processEnvVars(&config)

	// Resolve relative paths relative to config file location (ignore errors)
	_ = resolveConfigPaths(&config)

	return &config, nil
}

// processEnvVars processes environment variable substitutions in config values
func processEnvVars(config *Config) error {
	// Removed API key processing - browser auth only

	// Process cookie header (optional)
	if strings.HasPrefix(config.Tugboat.CookieHeader, "${") && strings.HasSuffix(config.Tugboat.CookieHeader, "}") {
		envVar := strings.TrimSuffix(strings.TrimPrefix(config.Tugboat.CookieHeader, "${"), "}")
		if value := os.Getenv(envVar); value != "" {
			config.Tugboat.CookieHeader = value
		} else {
			// Cookie header is optional - clear it if env var not set
			config.Tugboat.CookieHeader = ""
		}
	}

	// Process GitHub API token (optional)
	if strings.HasPrefix(config.Evidence.Tools.GitHub.APIToken, "${") && strings.HasSuffix(config.Evidence.Tools.GitHub.APIToken, "}") {
		envVar := strings.TrimSuffix(strings.TrimPrefix(config.Evidence.Tools.GitHub.APIToken, "${"), "}")
		if value := os.Getenv(envVar); value != "" {
			config.Evidence.Tools.GitHub.APIToken = value
		} else {
			// GitHub API token is optional - clear it if env var not set
			config.Evidence.Tools.GitHub.APIToken = ""
		}
	}

	// Populate GitHub token from gh CLI if not configured
	// This ensures all GitHub tools have access to the token
	if config.Auth.GitHub.Token == "" && config.Evidence.Tools.GitHub.APIToken == "" {
		if token := getGitHubTokenFromCLI(); token != "" {
			// Populate both locations for backwards compatibility
			config.Auth.GitHub.Token = token
			config.Evidence.Tools.GitHub.APIToken = token
		}
	}

	// Process Google Docs credentials file (optional)
	if strings.HasPrefix(config.Evidence.Tools.GoogleDocs.CredentialsFile, "${") && strings.HasSuffix(config.Evidence.Tools.GoogleDocs.CredentialsFile, "}") {
		envVar := strings.TrimSuffix(strings.TrimPrefix(config.Evidence.Tools.GoogleDocs.CredentialsFile, "${"), "}")
		if value := os.Getenv(envVar); value != "" {
			config.Evidence.Tools.GoogleDocs.CredentialsFile = value
		} else {
			// Google Docs credentials is optional - clear it if env var not set
			config.Evidence.Tools.GoogleDocs.CredentialsFile = ""
		}
	}

	// Process Auth configuration environment variables
	// GitHub token (optional)
	if strings.HasPrefix(config.Auth.GitHub.Token, "${") && strings.HasSuffix(config.Auth.GitHub.Token, "}") {
		envVar := strings.TrimSuffix(strings.TrimPrefix(config.Auth.GitHub.Token, "${"), "}")
		if value := os.Getenv(envVar); value != "" {
			config.Auth.GitHub.Token = value
		} else {
			// GitHub token is optional - clear it if env var not set
			config.Auth.GitHub.Token = ""
		}
	}

	// Tugboat bearer token (optional)
	if strings.HasPrefix(config.Auth.Tugboat.BearerToken, "${") && strings.HasSuffix(config.Auth.Tugboat.BearerToken, "}") {
		envVar := strings.TrimSuffix(strings.TrimPrefix(config.Auth.Tugboat.BearerToken, "${"), "}")
		if value := os.Getenv(envVar); value != "" {
			config.Auth.Tugboat.BearerToken = value
		} else {
			// Tugboat bearer token is optional - clear it if env var not set
			config.Auth.Tugboat.BearerToken = ""
		}
	}

	// Custom Evidence Integration API credentials
	// Password can be in config or environment variable
	if strings.HasPrefix(config.Tugboat.Password, "${") && strings.HasSuffix(config.Tugboat.Password, "}") {
		envVar := strings.TrimSuffix(strings.TrimPrefix(config.Tugboat.Password, "${"), "}")
		if value := os.Getenv(envVar); value != "" {
			config.Tugboat.Password = value
		} else {
			// Password is optional - clear it if env var not set
			config.Tugboat.Password = ""
		}
	}

	return nil
}

// getGitHubTokenFromCLI attempts to retrieve a GitHub token from the gh CLI
// Returns an empty string if gh CLI is not available or not authenticated
// This is a helper function to centralize GitHub authentication
func getGitHubTokenFromCLI() string {
	cmd := exec.Command("gh", "auth", "token")
	output, err := cmd.Output()
	if err != nil {
		return "" // gh CLI not available or not authenticated
	}
	return strings.TrimSpace(string(output))
}

// resolveConfigPaths resolves relative paths in config relative to config file location
func resolveConfigPaths(cfg *Config) error {
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		// No config file loaded (using defaults), paths are relative to CWD
		return nil
	}

	// Get directory containing config file
	configDir := filepath.Dir(configFile)

	// Resolve storage paths
	if cfg.Storage.DataDir != "" && !filepath.IsAbs(cfg.Storage.DataDir) {
		cfg.Storage.DataDir = filepath.Join(configDir, cfg.Storage.DataDir)
	}
	if cfg.Storage.LocalDataDir != "" && !filepath.IsAbs(cfg.Storage.LocalDataDir) {
		cfg.Storage.LocalDataDir = filepath.Join(configDir, cfg.Storage.LocalDataDir)
	}
	if cfg.Storage.CacheDir != "" && !filepath.IsAbs(cfg.Storage.CacheDir) {
		cfg.Storage.CacheDir = filepath.Join(configDir, cfg.Storage.CacheDir)
	}

	// Resolve storage.paths subdirectories (these are relative to data_dir, not config dir)
	// Note: These paths are stored as configured and resolved later relative to data_dir

	// Resolve auth cache dir
	if cfg.Auth.CacheDir != "" && !filepath.IsAbs(cfg.Auth.CacheDir) {
		cfg.Auth.CacheDir = filepath.Join(configDir, cfg.Auth.CacheDir)
	}

	// Resolve evidence generation paths
	if cfg.Evidence.Generation.OutputDir != "" && !filepath.IsAbs(cfg.Evidence.Generation.OutputDir) {
		cfg.Evidence.Generation.OutputDir = filepath.Join(configDir, cfg.Evidence.Generation.OutputDir)
	}
	if cfg.Evidence.Generation.PromptDir != "" && !filepath.IsAbs(cfg.Evidence.Generation.PromptDir) {
		cfg.Evidence.Generation.PromptDir = filepath.Join(configDir, cfg.Evidence.Generation.PromptDir)
	}
	if cfg.Evidence.Generation.SummaryCacheDir != "" && !filepath.IsAbs(cfg.Evidence.Generation.SummaryCacheDir) {
		cfg.Evidence.Generation.SummaryCacheDir = filepath.Join(configDir, cfg.Evidence.Generation.SummaryCacheDir)
	}

	// Resolve Terraform paths
	if cfg.Evidence.Terraform.AtmosPath != "" && !filepath.IsAbs(cfg.Evidence.Terraform.AtmosPath) {
		cfg.Evidence.Terraform.AtmosPath = filepath.Join(configDir, cfg.Evidence.Terraform.AtmosPath)
	}
	if cfg.Evidence.Terraform.RepoPath != "" && !filepath.IsAbs(cfg.Evidence.Terraform.RepoPath) {
		cfg.Evidence.Terraform.RepoPath = filepath.Join(configDir, cfg.Evidence.Terraform.RepoPath)
	}

	// Resolve terraform tool scan paths
	for i, path := range cfg.Evidence.Tools.Terraform.ScanPaths {
		if !filepath.IsAbs(path) {
			cfg.Evidence.Tools.Terraform.ScanPaths[i] = filepath.Join(configDir, path)
		}
	}

	// Resolve Google Docs credentials file
	if cfg.Evidence.Tools.GoogleDocs.CredentialsFile != "" && !filepath.IsAbs(cfg.Evidence.Tools.GoogleDocs.CredentialsFile) {
		cfg.Evidence.Tools.GoogleDocs.CredentialsFile = filepath.Join(configDir, cfg.Evidence.Tools.GoogleDocs.CredentialsFile)
	}

	// Resolve file paths in logger configs
	for name, loggerCfg := range cfg.Logging.Loggers {
		if loggerCfg.FilePath != "" && !filepath.IsAbs(loggerCfg.FilePath) {
			loggerCfg.FilePath = filepath.Join(configDir, loggerCfg.FilePath)
			cfg.Logging.Loggers[name] = loggerCfg
		}
	}

	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate Tugboat configuration
	if c.Tugboat.BaseURL == "" {
		return fmt.Errorf("tugboat.base_url is required")
	}
	// Note: org_id is only required for web scraping mode, not API mode
	// This will be validated when used

	// Validate auth mode
	if c.Tugboat.AuthMode != "" && c.Tugboat.AuthMode != "form" && c.Tugboat.AuthMode != "browser" {
		return fmt.Errorf("tugboat.auth_mode must be 'form' or 'browser', got: %s", c.Tugboat.AuthMode)
	}
	if c.Tugboat.Timeout <= 0 {
		c.Tugboat.Timeout = 30 * time.Second // default
	}
	if c.Tugboat.RateLimit <= 0 {
		c.Tugboat.RateLimit = 10 // default
	}

	// Validate Evidence configuration
	// Terraform configuration validation
	if c.Evidence.Terraform.AtmosPath != "" {
		if _, err := os.Stat(c.Evidence.Terraform.AtmosPath); os.IsNotExist(err) {
			return fmt.Errorf("evidence.terraform.atmos_path does not exist: %s", c.Evidence.Terraform.AtmosPath)
		}
	}
	if c.Evidence.Terraform.RepoPath != "" {
		if _, err := os.Stat(c.Evidence.Terraform.RepoPath); os.IsNotExist(err) {
			return fmt.Errorf("evidence.terraform.repo_path does not exist: %s", c.Evidence.Terraform.RepoPath)
		}
	}

	// Validate Generation configuration
	if c.Evidence.Generation.OutputDir == "" {
		c.Evidence.Generation.OutputDir = "evidence/generated" // default
	}
	if c.Evidence.Generation.PromptDir == "" {
		c.Evidence.Generation.PromptDir = "evidence/prompts" // default
	}
	if c.Evidence.Generation.MaxToolCalls <= 0 {
		c.Evidence.Generation.MaxToolCalls = 50 // default
	}
	if c.Evidence.Generation.DefaultFormat == "" {
		c.Evidence.Generation.DefaultFormat = "csv" // default per spec
	}
	if c.Evidence.Generation.DefaultFormat != "csv" && c.Evidence.Generation.DefaultFormat != "markdown" {
		return fmt.Errorf("evidence.generation.default_format must be 'csv' or 'markdown', got: %s", c.Evidence.Generation.DefaultFormat)
	}

	// Validate Tools configuration
	// Terraform tool validation
	if c.Evidence.Tools.Terraform.Enabled {
		if len(c.Evidence.Tools.Terraform.ScanPaths) == 0 {
			// Use configured path if available
			if c.Evidence.Terraform.AtmosPath != "" {
				c.Evidence.Tools.Terraform.ScanPaths = []string{c.Evidence.Terraform.AtmosPath + "/**/*.tf"}
			} else {
				c.Evidence.Tools.Terraform.ScanPaths = []string{"deploy/atmos/**/*.tf"} // default
			}
		}
		if len(c.Evidence.Tools.Terraform.IncludePatterns) == 0 {
			c.Evidence.Tools.Terraform.IncludePatterns = []string{"*.tf", "*.tfvars"} // default
		}
		if len(c.Evidence.Tools.Terraform.ExcludePatterns) == 0 {
			c.Evidence.Tools.Terraform.ExcludePatterns = []string{"*.secret", ".terraform/**"} // default
		}
	}

	// GitHub tool validation
	if c.Evidence.Tools.GitHub.Enabled {
		// API token is no longer required - gh CLI can provide it
		// Just validate repository is set (can be empty if provided via --repository flag)
		if c.Evidence.Tools.GitHub.MaxIssues <= 0 {
			c.Evidence.Tools.GitHub.MaxIssues = 100 // default
		}
		if c.Evidence.Tools.GitHub.RateLimit <= 0 {
			c.Evidence.Tools.GitHub.RateLimit = 30 // default - GitHub Search API limit
		}
	}

	// Google Docs tool validation
	if c.Evidence.Tools.GoogleDocs.Enabled {
		if c.Evidence.Tools.GoogleDocs.CredentialsFile == "" {
			return fmt.Errorf("evidence.tools.google_docs.credentials_file is required when Google Docs tool is enabled")
		}
		if _, err := os.Stat(c.Evidence.Tools.GoogleDocs.CredentialsFile); os.IsNotExist(err) {
			return fmt.Errorf("evidence.tools.google_docs.credentials_file does not exist: %s", c.Evidence.Tools.GoogleDocs.CredentialsFile)
		}
	}

	// Validate Quality configuration
	if c.Evidence.Quality.MinSources <= 0 {
		c.Evidence.Quality.MinSources = 2 // default
	}
	if c.Evidence.Quality.MinCompletenessScore < 0 || c.Evidence.Quality.MinCompletenessScore > 1 {
		c.Evidence.Quality.MinCompletenessScore = 0.7 // default
	}
	if c.Evidence.Quality.MinQualityScore < 0 || c.Evidence.Quality.MinQualityScore > 1 {
		c.Evidence.Quality.MinQualityScore = 0.8 // default
	}

	// Validate Storage configuration
	if c.Storage.DataDir == "" {
		c.Storage.DataDir = "./data" // default
	}
	if c.Storage.LocalDataDir == "" {
		c.Storage.LocalDataDir = "./local_data" // default
	}
	if c.Storage.CacheDir == "" {
		c.Storage.CacheDir = "./.cache" // default
	}

	// Validate Logging configuration - use defaults if not configured
	if len(c.Logging.Loggers) == 0 {
		c.Logging = *DefaultLoggingConfig()
	}

	// Validate Interpolation configuration
	// Set defaults
	if c.Interpolation.Variables == nil {
		c.Interpolation.Variables = make(map[string]interface{})
	}
	// Add default organization name mapping if not specified
	flatVars := c.Interpolation.GetFlatVariables()
	if _, exists := flatVars["organization.name"]; !exists && len(c.Interpolation.Variables) == 0 {
		// Only set default if no variables are configured at all
		c.Interpolation.Variables = map[string]interface{}{
			"organization": map[string]interface{}{
				"name": "[Organization Name]",
			},
			"Organization Name": "Your Organization",
		}
	}

	// Check for circular references in variable definitions
	if err := c.validateInterpolationVariables(); err != nil {
		return fmt.Errorf("interpolation configuration error: %w", err)
	}

	// Validate Auth configuration
	if c.Auth.CacheDir == "" {
		c.Auth.CacheDir = filepath.Join(c.Storage.CacheDir, "auth") // default
	}

	return nil
}

// validateInterpolationVariables checks for circular references and other issues in variable definitions
func (c *Config) validateInterpolationVariables() error {
	// Use flattened variables for validation
	flatVars := c.Interpolation.GetFlatVariables()
	visited := make(map[string]bool)
	recursionStack := make(map[string]bool)

	for key := range flatVars {
		if !visited[key] {
			if c.hasCircularReference(key, flatVars, visited, recursionStack) {
				return fmt.Errorf("circular reference detected involving variable: %s", key)
			}
		}
	}

	return nil
}

// hasCircularReference performs DFS to detect circular references in variable definitions
func (c *Config) hasCircularReference(key string, flatVars map[string]string, visited, recursionStack map[string]bool) bool {
	visited[key] = true
	recursionStack[key] = true

	value := flatVars[key]

	// Look for references to other variables in the value
	// This is a simple check - could be enhanced with regex if needed
	for otherKey := range flatVars {
		if otherKey != key {
			// Check if this value references another variable (simple contains check)
			if strings.Contains(value, "{{"+otherKey+"}}") || strings.Contains(value, "["+otherKey+"]") {
				if !visited[otherKey] {
					if c.hasCircularReference(otherKey, flatVars, visited, recursionStack) {
						return true
					}
				} else if recursionStack[otherKey] {
					return true
				}
			}
		}
	}

	recursionStack[key] = false
	return false
}

// ToLoggerConfig converts a LoggerConfig to logger.Config
func (lc *LoggerConfig) ToLoggerConfig() *logger.Config {
	// Convert string level to logger.LogLevel
	var level logger.LogLevel
	switch strings.ToLower(lc.Level) {
	case "trace":
		level = logger.TraceLevel
	case "debug":
		level = logger.DebugLevel
	case "info":
		level = logger.InfoLevel
	case "warn":
		level = logger.WarnLevel
	case "error":
		level = logger.ErrorLevel
	default:
		level = logger.InfoLevel
	}

	return &logger.Config{
		Level:         level,
		Format:        lc.Format,
		Output:        lc.Output,
		FilePath:      lc.FilePath,
		SanitizeURLs:  lc.SanitizeURLs,
		RedactFields:  lc.RedactFields,
		ShowCaller:    lc.ShowCaller,
		BufferSize:    lc.BufferSize,
		FlushInterval: lc.FlushInterval,
	}
}

// DefaultLoggingConfig returns default logging configuration with console and file loggers
func DefaultLoggingConfig() *LoggingConfig {
	defaultRedactFields := []string{"password", "token", "key", "secret", "api_key", "cookie"}

	return &LoggingConfig{
		Loggers: map[string]LoggerConfig{
			"console": {
				Enabled:       true,
				Level:         "warn",
				Format:        "text",
				Output:        "stderr",
				SanitizeURLs:  true,
				RedactFields:  defaultRedactFields,
				ShowCaller:    false,
				BufferSize:    100,
				FlushInterval: "5s",
			},
			"file": {
				Enabled:       true,
				Level:         "info",
				Format:        "text",
				Output:        "file",
				FilePath:      logger.DefaultLogFilePath(),
				SanitizeURLs:  true,
				RedactFields:  defaultRedactFields,
				ShowCaller:    true,
				BufferSize:    100,
				FlushInterval: "5s",
			},
		},
	}
}
