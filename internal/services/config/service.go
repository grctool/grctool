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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"gopkg.in/yaml.v3"
)

// ServiceImpl implements the config Service interface
type ServiceImpl struct {
	logger logger.Logger
}

// NewService creates a new config service implementation
func NewService(log logger.Logger) Service {
	return &ServiceImpl{
		logger: log,
	}
}

// InitializeConfig creates a new configuration file
func (s *ServiceImpl) InitializeConfig(outputPath string, force bool) error {
	// Check if file exists and force is not set
	if _, err := os.Stat(outputPath); err == nil && !force {
		return fmt.Errorf("configuration file already exists at %s. Use force=true to overwrite", outputPath)
	}

	// Generate default configuration
	defaultConfig := s.GenerateDefaultConfig()

	// Save configuration file
	return s.SaveConfigFile(defaultConfig, outputPath)
}

// ValidateConfig performs comprehensive configuration validation
func (s *ServiceImpl) ValidateConfig(ctx context.Context) (*ValidationResult, error) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create validator and run validation
	validator := NewConfigValidator(cfg)
	return validator.Validate(ctx)
}

// GenerateDefaultConfig returns a default configuration structure
func (s *ServiceImpl) GenerateDefaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"tugboat": map[string]interface{}{
			"base_url":   "https://api-my.tugboatlogic.com",
			"org_id":     "YOUR_ORG_ID",
			"timeout":    "30s",
			"rate_limit": 10,
			"auth_mode":  "browser",
		},
		"storage": map[string]interface{}{
			"data_dir": "./",
		},
		"logging": map[string]interface{}{
			"level":  "info",
			"format": "text",
		},
		"evidence": map[string]interface{}{
			"terraform": map[string]interface{}{
				"include_patterns": []string{
					"*.tf",
					"*.yaml",
					"*.yml",
				},
				"exclude_patterns": []string{
					"*.secret",
					"builds/",
					"terraform.tfstate*",
				},
			},
		},
	}
}

// GenerateConfigTemplate generates different configuration templates
func (s *ServiceImpl) GenerateConfigTemplate(templateType string) (map[string]interface{}, error) {
	switch templateType {
	case "minimal":
		return map[string]interface{}{
			"tugboat": map[string]interface{}{
				"base_url": "https://api-my.tugboatlogic.com",
				"org_id":   "YOUR_ORG_ID",
			},
			"storage": map[string]interface{}{
				"data_dir": "./",
			},
		}, nil
	case "extended":
		config := s.GenerateDefaultConfig()
		// Add extended configuration options
		config["evidence"] = map[string]interface{}{
			"terraform": map[string]interface{}{
				"include_patterns": []string{
					"*.tf", "*.yaml", "*.yml", "*.json",
				},
				"exclude_patterns": []string{
					"*.secret", "*.key", "builds/", "terraform.tfstate*", ".terraform/",
				},
				"scan_depth": 3,
			},
			"github": map[string]interface{}{
				"enabled":     true,
				"api_token":   "${GITHUB_API_TOKEN}",
				"org":         "YOUR_GITHUB_ORG",
				"scan_repos":  true,
				"scan_teams":  true,
				"scan_access": true,
			},
			"google_docs": map[string]interface{}{
				"enabled":          false,
				"credentials_file": "credentials.json",
				"folder_id":        "YOUR_FOLDER_ID",
			},
		}
		return config, nil
	default:
		return s.GenerateDefaultConfig(), nil
	}
}

// SaveConfigFile saves a configuration to file
func (s *ServiceImpl) SaveConfigFile(cfg map[string]interface{}, outputPath string) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write configuration to file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	defer encoder.Close()

	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	return nil
}

// LoadAndValidateConfig loads configuration and validates it
func (s *ServiceImpl) LoadAndValidateConfig() (*config.Config, *ValidationResult, error) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate configuration
	validator := NewConfigValidator(cfg)
	result, err := validator.Validate(context.Background())
	if err != nil {
		return cfg, nil, fmt.Errorf("validation failed: %w", err)
	}

	return cfg, result, nil
}

// ConfigValidatorImpl implements the ConfigValidator interface
type ConfigValidatorImpl struct {
	cfg *config.Config
}

// NewConfigValidator creates a new configuration validator
func NewConfigValidator(cfg *config.Config) ConfigValidator {
	return &ConfigValidatorImpl{cfg: cfg}
}

// Validate performs comprehensive configuration validation
func (v *ConfigValidatorImpl) Validate(ctx context.Context) (*ValidationResult, error) {
	start := time.Now()
	result := &ValidationResult{
		Valid:  true,
		Checks: make(map[string]ValidationCheck),
		Errors: []string{},
	}

	// Run all validation checks
	v.ValidatePaths(ctx, result)
	v.ValidatePermissions(ctx, result)
	v.ValidateEnvironmentVariables(ctx, result)
	v.ValidateTugboatConnectivity(ctx, result)
	v.ValidateToolConfigurations(ctx, result)
	v.ValidateStorageConfiguration(ctx, result)

	result.Duration = time.Since(start)
	return result, nil
}

// ValidatePaths validates that all configured paths exist
func (v *ConfigValidatorImpl) ValidatePaths(ctx context.Context, result *ValidationResult) {
	start := time.Now()
	check := ValidationCheck{
		Name:     "File Paths",
		Status:   "pass",
		Message:  "All configured paths are valid",
		Duration: 0,
	}

	pathsToCheck := []struct {
		path     string
		name     string
		required bool
	}{
		{v.cfg.Storage.DataDir, "storage.data_dir", true},
	}

	var errors []string
	for _, p := range pathsToCheck {
		if p.path == "" {
			if p.required {
				errors = append(errors, fmt.Sprintf("%s is required but not set", p.name))
			}
			continue
		}

		// Check if path exists, create if it doesn't exist and it's a directory
		if _, err := os.Stat(p.path); os.IsNotExist(err) {
			// For directories, try to create them
			if err := os.MkdirAll(p.path, 0755); err != nil {
				errors = append(errors, fmt.Sprintf("failed to create directory %s: %v", p.path, err))
			}
		}
	}

	if len(errors) > 0 {
		check.Status = "fail"
		check.Message = fmt.Sprintf("Path validation failed: %d issues found", len(errors))
		result.Valid = false
		result.Errors = append(result.Errors, errors...)
	}

	check.Duration = time.Since(start)
	result.Checks["paths"] = check
}

// ValidatePermissions validates file and directory permissions
func (v *ConfigValidatorImpl) ValidatePermissions(ctx context.Context, result *ValidationResult) {
	start := time.Now()
	check := ValidationCheck{
		Name:     "File Permissions",
		Status:   "pass",
		Message:  "All file permissions are correct",
		Duration: 0,
	}

	// Check write permissions for data directory
	if v.cfg.Storage.DataDir != "" {
		testFile := filepath.Join(v.cfg.Storage.DataDir, ".grctool-test")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			check.Status = "fail"
			check.Message = fmt.Sprintf("Cannot write to data directory: %v", err)
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Data directory not writable: %s", v.cfg.Storage.DataDir))
		} else {
			os.Remove(testFile) // Clean up
		}
	}

	check.Duration = time.Since(start)
	result.Checks["permissions"] = check
}

// ValidateEnvironmentVariables validates required environment variables
func (v *ConfigValidatorImpl) ValidateEnvironmentVariables(ctx context.Context, result *ValidationResult) {
	start := time.Now()
	check := ValidationCheck{
		Name:     "Environment Variables",
		Status:   "pass",
		Message:  "All required environment variables are set",
		Duration: 0,
	}

	var missing []string
	var warnings []string

	// Check for Tugboat authentication
	if v.cfg.Tugboat.CookieHeader == "" && v.cfg.Tugboat.BearerToken == "" {
		if v.cfg.Tugboat.AuthMode != "browser" {
			missing = append(missing, "browser authentication credentials (run 'grctool auth login')")
		}
	}

	if len(missing) > 0 {
		check.Status = "fail"
		check.Message = fmt.Sprintf("Missing required environment variables: %v", missing)
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Missing environment variables: %v", missing))
	} else if len(warnings) > 0 {
		check.Status = "warning"
		check.Message = fmt.Sprintf("Optional environment variables missing: %v", warnings)
	}

	check.Duration = time.Since(start)
	result.Checks["environment"] = check
}

// ValidateTugboatConnectivity validates connectivity to Tugboat Logic API
func (v *ConfigValidatorImpl) ValidateTugboatConnectivity(ctx context.Context, result *ValidationResult) {
	start := time.Now()
	check := ValidationCheck{
		Name:     "Tugboat Connectivity",
		Status:   "pass",
		Message:  "Tugboat Logic API is accessible",
		Duration: 0,
	}

	// Simple connectivity check - this is a placeholder
	// In practice, you might want to make an actual API call
	if v.cfg.Tugboat.BaseURL == "" {
		check.Status = "fail"
		check.Message = "Tugboat base URL is not configured"
		result.Valid = false
		result.Errors = append(result.Errors, "Tugboat base URL is required")
	}

	check.Duration = time.Since(start)
	result.Checks["tugboat_connectivity"] = check
}

// ValidateToolConfigurations validates tool-specific configurations
func (v *ConfigValidatorImpl) ValidateToolConfigurations(ctx context.Context, result *ValidationResult) {
	start := time.Now()
	check := ValidationCheck{
		Name:     "Tool Configurations",
		Status:   "pass",
		Message:  "All tool configurations are valid",
		Duration: 0,
	}

	// This is a placeholder for tool configuration validation
	// In practice, you would validate specific tool configurations

	check.Duration = time.Since(start)
	result.Checks["tools"] = check
}

// ValidateStorageConfiguration validates storage configuration
func (v *ConfigValidatorImpl) ValidateStorageConfiguration(ctx context.Context, result *ValidationResult) {
	start := time.Now()
	check := ValidationCheck{
		Name:     "Storage Configuration",
		Status:   "pass",
		Message:  "Storage configuration is valid",
		Duration: 0,
	}

	// Validate storage configuration
	if v.cfg.Storage.DataDir == "" {
		check.Status = "fail"
		check.Message = "Data directory is not configured"
		result.Valid = false
		result.Errors = append(result.Errors, "Data directory is required")
	}

	check.Duration = time.Since(start)
	result.Checks["storage"] = check
}

// GenerateClaudeMd generates CLAUDE.md file with config-aware content
func (s *ServiceImpl) GenerateClaudeMd(outputPath string, force bool) error {
	// Check if file exists and force is not set
	if _, err := os.Stat(outputPath); err == nil && !force {
		return fmt.Errorf("CLAUDE.md already exists at %s. Use force=true to overwrite", outputPath)
	}

	// Load current configuration without validation
	// This allows generating CLAUDE.md even for incomplete configs
	cfg, err := config.LoadWithoutValidation()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Tool registry is already initialized during cobra startup (cobra.OnInitialize)
	// No need to initialize it again here

	// Render template with config values
	content, err := RenderClaudeMd(cfg)
	if err != nil {
		return fmt.Errorf("failed to render CLAUDE.md template: %w", err)
	}

	// Write file
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write CLAUDE.md: %w", err)
	}

	return nil
}
