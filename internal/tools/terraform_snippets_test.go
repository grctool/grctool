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

package tools

import (
	"context"
	"testing"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTerraformSnippetsTool(t *testing.T) {
	// Create a test config
	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				Terraform: config.TerraformToolConfig{
					Enabled:         true,
					ScanPaths:       []string{"testdata"},
					IncludePatterns: []string{"*.tf"},
					ExcludePatterns: []string{},
				},
			},
		},
	}

	cfgLogger := &logger.Config{
		Level:  logger.ErrorLevel,
		Format: "text",
		Output: "stdout",
	}
	log, err := logger.New(cfgLogger)
	require.NoError(t, err)
	tool := NewTerraformSnippetsAdapter(cfg, log)

	t.Run("Tool Properties", func(t *testing.T) {
		assert.Equal(t, "terraform_snippets", tool.Name())
		assert.Contains(t, tool.Description(), "Terraform configuration snippets")

		definition := tool.GetClaudeToolDefinition()
		assert.Equal(t, "terraform_snippets", definition.Name)
		assert.NotNil(t, definition.InputSchema)
	})

	t.Run("Generate Snippets for Control Codes", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"control_codes": []interface{}{"CC6.1", "CC6.8"},
			"output_format": "markdown",
		}

		result, source, err := tool.Execute(ctx, params)
		require.NoError(t, err)
		assert.NotEmpty(t, result)
		assert.NotNil(t, source)
		assert.Equal(t, "terraform_snippets", source.Type)

		// Check that result contains expected snippets
		assert.Contains(t, result, "# Terraform Configuration Snippets")
		assert.Contains(t, result, "CC6.1") // Access control snippet
		assert.Contains(t, result, "CC6.8") // Encryption snippet
		assert.Contains(t, result, "aws_iam_role")
		assert.Contains(t, result, "aws_s3_bucket")
	})

	t.Run("Generate Snippets for Resource Type", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"resource_type": "aws_kms_key",
		}

		result, source, err := tool.Execute(ctx, params)
		require.NoError(t, err)
		assert.NotEmpty(t, result)
		assert.NotNil(t, source)

		// Check that result contains KMS key snippet
		assert.Contains(t, result, "aws_kms_key")
		assert.Contains(t, result, "enable_key_rotation")
		assert.Contains(t, result, "Customer Managed KMS Key")
	})

	t.Run("Search for Patterns", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"pattern_search":   "encryption",
			"include_examples": false, // Disable to avoid file system dependencies
		}

		result, source, err := tool.Execute(ctx, params)
		require.NoError(t, err)
		assert.NotEmpty(t, result)
		assert.NotNil(t, source)
	})

	t.Run("Get Common Security Snippets", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{} // No specific parameters

		result, source, err := tool.Execute(ctx, params)
		require.NoError(t, err)
		assert.NotEmpty(t, result)
		assert.NotNil(t, source)

		// Should return common security snippets
		assert.Contains(t, result, "Least Privilege Policy Document")
		assert.Contains(t, result, "Secure VPC with Private Subnets")
	})

	t.Run("Invalid Control Codes", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"control_codes": []interface{}{"INVALID_CODE"},
		}

		result, source, err := tool.Execute(ctx, params)
		require.NoError(t, err) // Should not error, just return empty or default snippets
		assert.NotNil(t, source)

		// Should return the header even with invalid codes
		assert.Contains(t, result, "No Terraform snippets found matching the criteria.")
	})
}

// TestTerraformSnippetsTool_FormatSnippets was removed as it tested private implementation details.
// The formatting behavior is already validated through the public Execute() method tests above.
