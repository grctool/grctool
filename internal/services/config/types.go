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
	"time"

	"github.com/grctool/grctool/internal/config"
)

// Service provides configuration management operations
type Service interface {
	// Configuration initialization and management
	InitializeConfig(outputPath string, force bool) error
	ValidateConfig(ctx context.Context) (*ValidationResult, error)

	// Template and default configuration generation
	GenerateDefaultConfig() map[string]interface{}
	GenerateConfigTemplate(templateType string) (map[string]interface{}, error)

	// Configuration file operations
	SaveConfigFile(cfg map[string]interface{}, outputPath string) error
	LoadAndValidateConfig() (*config.Config, *ValidationResult, error)

	// AI assistance documentation
	GenerateClaudeMd(outputPath string, force bool) error
}

// ValidationResult represents the result of configuration validation
type ValidationResult struct {
	Valid    bool                       `json:"valid"`
	Checks   map[string]ValidationCheck `json:"checks"`
	Duration time.Duration              `json:"duration"`
	Errors   []string                   `json:"errors,omitempty"`
}

// ValidationCheck represents a single validation check result
type ValidationCheck struct {
	Name     string        `json:"name"`
	Status   string        `json:"status"` // "pass", "fail", "warning"
	Message  string        `json:"message"`
	Duration time.Duration `json:"duration"`
}

// ConfigValidator validates configuration
type ConfigValidator interface {
	Validate(ctx context.Context) (*ValidationResult, error)
	ValidatePaths(ctx context.Context, result *ValidationResult)
	ValidatePermissions(ctx context.Context, result *ValidationResult)
	ValidateEnvironmentVariables(ctx context.Context, result *ValidationResult)
	ValidateTugboatConnectivity(ctx context.Context, result *ValidationResult)
	ValidateToolConfigurations(ctx context.Context, result *ValidationResult)
	ValidateStorageConfiguration(ctx context.Context, result *ValidationResult)
}

// InitializationOptions controls config initialization
type InitializationOptions struct {
	OutputPath   string `json:"output_path"`
	Force        bool   `json:"force"`
	TemplateType string `json:"template_type"` // "default", "minimal", "extended"
	OrgID        string `json:"org_id,omitempty"`
	BaseURL      string `json:"base_url,omitempty"`
}

// ValidationOptions controls validation behavior
type ValidationOptions struct {
	SkipConnectivity bool `json:"skip_connectivity"`
	SkipPermissions  bool `json:"skip_permissions"`
	CreatePaths      bool `json:"create_paths"`
	Verbose          bool `json:"verbose"`
}

// ConfigTemplate represents different configuration templates
type ConfigTemplate struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
	Required    []string               `json:"required"`
	Optional    []string               `json:"optional"`
}
