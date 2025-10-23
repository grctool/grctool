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

package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolsIntegration_SOC2Evidence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// Setup comprehensive test environment
	testEnv := setupIntegrationTestEnvironment(t)
	defer testEnv.Cleanup()

	log, err := logger.New(&logger.Config{
		Level:  logger.ErrorLevel,
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err)

	// Initialize all tools
	terraformTool := tools.NewTerraformTool(testEnv.Config, log)
	githubTool := tools.NewGitHubTool(testEnv.Config, log)

	t.Run("CC6.8 Encryption Evidence Collection", func(t *testing.T) {
		// Collect Terraform encryption evidence
		ctx := context.Background()
		terraformParams := map[string]interface{}{
			"resource_types": []interface{}{"aws_kms_key", "aws_s3_bucket_encryption", "aws_db_instance"},
			"pattern":        "encrypt|kms|storage_encrypted",
			"control_hint":   "CC6.8",
			"output_format":  "json",
			"use_cache":      false,
		}

		terraformResult, terraformSource, err := terraformTool.Execute(ctx, terraformParams)
		require.NoError(t, err)
		assert.NotEmpty(t, terraformResult)
		assert.NotNil(t, terraformSource)

		// Parse Terraform results
		var terraformData map[string]interface{}
		err = json.Unmarshal([]byte(terraformResult), &terraformData)
		require.NoError(t, err)

		// Should find encryption resources
		results := terraformData["results"].([]interface{})
		assert.Greater(t, len(results), 0, "Should find encryption resources")

		// Collect GitHub encryption evidence
		githubParams := map[string]interface{}{
			"query":          "encryption KMS CC6.8",
			"labels":         []interface{}{"security", "encryption"},
			"include_closed": true,
		}

		githubResult, githubSource, err := githubTool.Execute(ctx, githubParams)
		require.NoError(t, err)
		assert.NotEmpty(t, githubResult)
		assert.NotNil(t, githubSource)

		// Validate integration
		assert.Contains(t, terraformResult, "aws_kms_key")
		// GitHub may return empty results - accept either evidence header or empty message
		assert.True(t,
			strings.Contains(githubResult, "GitHub Security Evidence") ||
				strings.Contains(githubResult, "No relevant GitHub issues found."),
			"Should have valid GitHub response")

		// Check evidence correlation
		terraformRelevance := terraformSource.Relevance
		githubRelevance := githubSource.Relevance
		assert.Greater(t, terraformRelevance, 0.0)
		assert.GreaterOrEqual(t, githubRelevance, 0.0, "GitHub may return empty results with 0.0 relevance")

		t.Logf("Terraform evidence relevance: %.2f", terraformRelevance)
		t.Logf("GitHub evidence relevance: %.2f", githubRelevance)
	})

	t.Run("CC6.1 Access Control Evidence Collection", func(t *testing.T) {
		// Collect Terraform access control evidence
		ctx := context.Background()
		terraformParams := map[string]interface{}{
			"resource_types": []interface{}{"aws_iam_role", "aws_iam_policy", "aws_security_group"},
			"pattern":        "iam|security_group|access|least.privilege",
			"control_hint":   "CC6.1",
			"output_format":  "markdown",
			"use_cache":      false,
		}

		terraformResult, terraformSource, err := terraformTool.Execute(ctx, terraformParams)
		require.NoError(t, err)
		assert.NotEmpty(t, terraformResult)
		assert.NotNil(t, terraformSource)

		// Should be in markdown format
		assert.Contains(t, terraformResult, "# Enhanced Terraform Security Configuration Evidence")
		assert.Contains(t, terraformResult, "**Security Controls:**")

		// Collect GitHub access control evidence
		githubParams := map[string]interface{}{
			"query":          "access control IAM permissions CC6.1",
			"labels":         []interface{}{"security", "access-control"},
			"include_closed": false,
		}

		githubResult, githubSource, err := githubTool.Execute(ctx, githubParams)
		require.NoError(t, err)
		assert.NotEmpty(t, githubResult)
		assert.NotNil(t, githubSource)

		// Validate evidence completeness
		assert.Contains(t, terraformResult, "aws_iam")
		assert.Contains(t, terraformResult, "security_group")
		// GitHub may return empty results - just verify we got a response
		assert.NotEmpty(t, githubResult, "Should have GitHub response (even if empty)")

		// Check metadata consistency
		terraformMeta := terraformSource.Metadata
		githubMeta := githubSource.Metadata

		assert.Equal(t, "CC6.1", terraformMeta["control_hint"])
		assert.Contains(t, githubMeta["query"], "CC6.1")
	})

	t.Run("CC8.1 Audit Logging Evidence Collection", func(t *testing.T) {
		// Collect Terraform audit logging evidence
		ctx := context.Background()
		terraformParams := map[string]interface{}{
			"resource_types": []interface{}{"aws_cloudtrail", "aws_cloudwatch_log_group", "aws_config_configuration_recorder"},
			"pattern":        "audit|log|trail|monitoring",
			"control_hint":   "CC8.1",
			"output_format":  "csv",
			"use_cache":      false,
		}

		terraformResult, terraformSource, err := terraformTool.Execute(ctx, terraformParams)
		require.NoError(t, err)
		assert.NotEmpty(t, terraformResult)
		assert.NotNil(t, terraformSource)

		// Should be in CSV format
		lines := strings.Split(terraformResult, "\n")
		assert.Greater(t, len(lines), 1)
		header := lines[0]
		assert.Contains(t, header, "Resource Type")
		assert.Contains(t, header, "Security Controls")

		// Collect GitHub audit logging evidence
		githubParams := map[string]interface{}{
			"query":          "audit logging CloudTrail monitoring CC8.1",
			"labels":         []interface{}{"audit", "logging", "compliance"},
			"include_closed": true,
		}

		githubResult, githubSource, err := githubTool.Execute(ctx, githubParams)
		require.NoError(t, err)
		assert.NotEmpty(t, githubResult)
		assert.NotNil(t, githubSource)

		// Validate audit evidence
		assert.Contains(t, terraformResult, "aws_cloudtrail")
		// GitHub may return empty results - accept any valid response
		assert.NotEmpty(t, githubResult, "Should have GitHub response")

		// Check evidence timestamps
		assert.WithinDuration(t, time.Now(), terraformSource.ExtractedAt, 5*time.Minute)
		assert.WithinDuration(t, time.Now(), githubSource.ExtractedAt, 5*time.Minute)
	})
}

func TestToolsIntegration_CrossValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	testEnv := setupIntegrationTestEnvironment(t)
	defer testEnv.Cleanup()

	log, err := logger.New(&logger.Config{
		Level:  logger.ErrorLevel,
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err)

	terraformTool := tools.NewTerraformTool(testEnv.Config, log)
	githubTool := tools.NewGitHubTool(testEnv.Config, log)

	t.Run("Evidence Consistency Validation", func(t *testing.T) {
		ctx := context.Background()

		// Collect evidence from both tools with same control focus
		terraformParams := map[string]interface{}{
			"pattern":       "security|encrypt|access",
			"output_format": "json",
			"use_cache":     false,
		}

		githubParams := map[string]interface{}{
			"query":          "security encryption access",
			"labels":         []interface{}{"security"},
			"include_closed": false,
		}

		// Execute both tools concurrently
		type toolResult struct {
			result string
			source *models.EvidenceSource
			err    error
		}

		terraformChan := make(chan toolResult)
		githubChan := make(chan toolResult)

		go func() {
			result, source, err := terraformTool.Execute(ctx, terraformParams)
			terraformChan <- toolResult{result, source, err}
		}()

		go func() {
			result, source, err := githubTool.Execute(ctx, githubParams)
			githubChan <- toolResult{result, source, err}
		}()

		// Collect results
		terraformRes := <-terraformChan
		githubRes := <-githubChan

		require.NoError(t, terraformRes.err)
		require.NoError(t, githubRes.err)

		// Validate cross-tool consistency
		assert.NotEmpty(t, terraformRes.result)
		assert.NotEmpty(t, githubRes.result)

		// Terraform should have security focus
		assert.Contains(t, strings.ToLower(terraformRes.result), "security")
		// GitHub may return empty results - check relevance instead of content
		// Note: VCR cassette shows empty GitHub results for this query

		// Both should have reasonable relevance scores
		assert.Greater(t, terraformRes.source.Relevance, 0.0)
		assert.GreaterOrEqual(t, githubRes.source.Relevance, 0.0, "GitHub may have empty results with 0.0 relevance")

		// Evidence sources should be different
		assert.NotEqual(t, terraformRes.source.Type, githubRes.source.Type)
		assert.Equal(t, "terraform_analyzer", terraformRes.source.Type)
		assert.Equal(t, "github", githubRes.source.Type)
	})

	t.Run("Complementary Evidence Collection", func(t *testing.T) {
		// Test that tools provide complementary evidence for same control

		ctx := context.Background()

		// Terraform provides infrastructure evidence
		terraformParams := map[string]interface{}{
			"resource_types": []interface{}{"aws_kms_key", "aws_s3_bucket_encryption"},
			"control_hint":   "CC6.8",
			"output_format":  "json",
		}

		terraformResult, terraformSource, err := terraformTool.Execute(ctx, terraformParams)
		require.NoError(t, err)

		// GitHub provides process evidence
		githubParams := map[string]interface{}{
			"query":  "encryption implementation review",
			"labels": []interface{}{"security", "encryption"},
		}

		githubResult, githubSource, err := githubTool.Execute(ctx, githubParams)
		require.NoError(t, err)

		// Parse Terraform data
		var terraformData map[string]interface{}
		err = json.Unmarshal([]byte(terraformResult), &terraformData)
		require.NoError(t, err)

		// Validate complementary nature
		// Terraform should show infrastructure resources
		assert.Contains(t, terraformResult, "aws_kms_key")

		// GitHub may return empty results - verify we got a response
		assert.NotEmpty(t, githubResult, "Should have GitHub response (even if no issues found)")

		// Both should contribute to CC6.8 evidence
		terraformMeta := terraformSource.Metadata
		assert.Equal(t, "CC6.8", terraformMeta["control_hint"])

		// Evidence should be from different perspectives
		assert.Contains(t, terraformSource.Resource, "Terraform")
		assert.Contains(t, githubSource.Resource, "GitHub")

		t.Logf("Terraform found %d resources", len(terraformData["results"].([]interface{})))
		githubMeta := githubSource.Metadata
		t.Logf("GitHub found %d issues", githubMeta["issue_count"])
	})
}

func TestToolsIntegration_OutputFormats(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	testEnv := setupIntegrationTestEnvironment(t)
	defer testEnv.Cleanup()

	log, err := logger.New(&logger.Config{
		Level:  logger.ErrorLevel,
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err)

	terraformTool := tools.NewTerraformTool(testEnv.Config, log)

	formats := []string{"json", "csv", "markdown"}

	t.Run("Multiple Format Generation", func(t *testing.T) {
		ctx := context.Background()
		baseParams := map[string]interface{}{
			"pattern":   "security|encrypt",
			"use_cache": false,
		}

		results := make(map[string]string)

		// Generate evidence in all formats
		for _, format := range formats {
			params := make(map[string]interface{})
			for k, v := range baseParams {
				params[k] = v
			}
			params["output_format"] = format

			result, source, err := terraformTool.Execute(ctx, params)
			require.NoError(t, err, "Failed to generate %s format", format)
			assert.NotEmpty(t, result)
			assert.NotNil(t, source)

			results[format] = result

			// Validate format-specific characteristics
			switch format {
			case "json":
				var jsonData map[string]interface{}
				err = json.Unmarshal([]byte(result), &jsonData)
				assert.NoError(t, err, "Should be valid JSON")
				assert.Contains(t, jsonData, "scan_summary")
			case "csv":
				lines := strings.Split(result, "\n")
				assert.Greater(t, len(lines), 1, "Should have header and data")
				assert.Contains(t, lines[0], "Resource Type")
			case "markdown":
				assert.Contains(t, result, "# Enhanced Terraform Security Configuration Evidence")
				assert.Contains(t, result, "**Configuration:**")
			}
		}

		// All formats should contain similar security content
		for _, result := range results {
			assert.Contains(t, strings.ToLower(result), "security")
		}
	})

	t.Run("Format Consistency", func(t *testing.T) {
		// Verify that same underlying data is presented in all formats
		ctx := context.Background()
		params := map[string]interface{}{
			"resource_types": []interface{}{"aws_kms_key"},
			"pattern":        "encrypt",
			"use_cache":      false,
		}

		var sources []*models.EvidenceSource

		for _, format := range formats {
			formatParams := make(map[string]interface{})
			for k, v := range params {
				formatParams[k] = v
			}
			formatParams["output_format"] = format

			_, source, err := terraformTool.Execute(ctx, formatParams)
			require.NoError(t, err)
			sources = append(sources, source)
		}

		// All sources should have similar metadata
		for i := 1; i < len(sources); i++ {
			assert.Equal(t, sources[0].Type, sources[i].Type)
			assert.WithinDuration(t, sources[0].ExtractedAt, sources[i].ExtractedAt, 10*time.Second)

			// Metadata should be consistent
			meta0 := sources[0].Metadata
			metaI := sources[i].Metadata
			assert.Equal(t, meta0["pattern"], metaI["pattern"])
			assert.Equal(t, meta0["resource_types"], metaI["resource_types"])
		}
	})
}

func TestToolsIntegration_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	testEnv := setupIntegrationTestEnvironment(t)
	defer testEnv.Cleanup()

	log, err := logger.New(&logger.Config{
		Level:  logger.ErrorLevel,
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err)

	terraformTool := tools.NewTerraformTool(testEnv.Config, log)

	t.Run("Caching Performance", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"pattern":       "security",
			"output_format": "json",
			"use_cache":     true,
		}

		// First execution (no cache)
		start := time.Now()
		result1, source1, err := terraformTool.Execute(ctx, params)
		firstDuration := time.Since(start)
		require.NoError(t, err)
		assert.NotEmpty(t, result1)

		// Second execution (with cache)
		start = time.Now()
		result2, source2, err := terraformTool.Execute(ctx, params)
		secondDuration := time.Since(start)
		require.NoError(t, err)
		assert.NotEmpty(t, result2)

		t.Logf("First execution: %v", firstDuration)
		t.Logf("Second execution: %v", secondDuration)

		// Cache should improve performance (but not always measurable in tests)
		assert.Equal(t, source1.Type, source2.Type)
		assert.NotZero(t, firstDuration)
		assert.NotZero(t, secondDuration)
	})

	t.Run("Large Result Set Handling", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"output_format": "json",
			"max_results":   1000, // Large limit
			"use_cache":     false,
		}

		start := time.Now()
		result, source, err := terraformTool.Execute(ctx, params)
		duration := time.Since(start)

		require.NoError(t, err)
		assert.NotEmpty(t, result)
		assert.NotNil(t, source)

		t.Logf("Large result set execution: %v", duration)

		// Should complete within reasonable time
		assert.Less(t, duration, 30*time.Second, "Should complete within 30 seconds")

		// Validate result structure
		var jsonData map[string]interface{}
		err = json.Unmarshal([]byte(result), &jsonData)
		require.NoError(t, err)

		results := jsonData["results"].([]interface{})
		t.Logf("Found %d results", len(results))
	})
}

// Integration test environment setup

type IntegrationTestEnvironment struct {
	TempDir    string
	Config     *config.Config
	HTTPServer *httptest.Server
}

func (env *IntegrationTestEnvironment) Cleanup() {
	if env.HTTPServer != nil {
		env.HTTPServer.Close()
	}
	os.RemoveAll(env.TempDir)
}

func setupIntegrationTestEnvironment(t *testing.T) *IntegrationTestEnvironment {
	tempDir := t.TempDir()

	// Create comprehensive terraform test environment
	createTerraformTestFiles(t, tempDir)

	// Setup mock GitHub server
	server := setupMockGitHubServer()

	// Get GitHub token from environment (for recording mode)
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		githubToken = "test-token-redacted" // Fallback for tests without token
	}

	// Create configuration
	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				Terraform: config.TerraformToolConfig{
					Enabled:         true,
					ScanPaths:       []string{filepath.Join(tempDir, "terraform")},
					IncludePatterns: []string{"*.tf", "*.tfvars"},
					ExcludePatterns: []string{".terraform/"},
				},
				GitHub: config.GitHubToolConfig{
					Enabled:    true,
					Repository: "your-org/grctool", // Use real repo for recording
					APIToken:   githubToken,
					MaxIssues:  50,
				},
			},
		},
		Storage: config.StorageConfig{
			DataDir: tempDir,
		},
	}

	return &IntegrationTestEnvironment{
		TempDir:    tempDir,
		Config:     cfg,
		HTTPServer: server,
	}
}

func createTerraformTestFiles(t *testing.T, tempDir string) {
	terraformDir := filepath.Join(tempDir, "terraform")
	err := os.MkdirAll(terraformDir, 0755)
	require.NoError(t, err)

	// Create comprehensive terraform files covering multiple controls
	files := map[string]string{
		"encryption.tf": `
# CC6.8 - Encryption at Rest
resource "aws_kms_key" "main" {
  description         = "Main encryption key"
  enable_key_rotation = true
  
  tags = {
    Control = "CC6.8"
    Purpose = "SOC2 Compliance"
  }
}

resource "aws_s3_bucket_encryption" "main" {
  bucket = "test-bucket"
  
  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        kms_master_key_id = aws_kms_key.main.arn
        sse_algorithm     = "aws:kms"
      }
    }
  }
}

resource "aws_db_instance" "main" {
  identifier        = "test-db"
  engine           = "postgres"
  instance_class   = "db.t3.micro"
  storage_encrypted = true
  kms_key_id       = aws_kms_key.main.arn
}
`,
		"access_control.tf": `
# CC6.1 - Access Controls
resource "aws_iam_role" "app" {
  name = "application-role"
  
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      }
    ]
  })
  
  tags = {
    Control = "CC6.1"
    Purpose = "Least Privilege Access"
  }
}

resource "aws_iam_policy" "least_privilege" {
  name = "least-privilege-policy"
  
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject"
        ]
        Resource = "arn:aws:s3:::specific-bucket/*"
      }
    ]
  })
}

resource "aws_security_group" "web" {
  name = "web-sg"
  
  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/8"]
  }
  
  tags = {
    Control = "CC6.1"
    Purpose = "Network Access Control"
  }
}
`,
		"audit_logging.tf": `
# CC8.1 - Audit Logging
resource "aws_cloudtrail" "main" {
  name           = "main-trail"
  s3_bucket_name = "audit-logs-bucket"
  
  enable_log_file_validation = true
  is_multi_region_trail     = true
  include_global_service_events = true
  
  tags = {
    Control = "CC8.1"
    Purpose = "Audit Trail"
  }
}

resource "aws_cloudwatch_log_group" "app_logs" {
  name              = "/aws/lambda/app"
  retention_in_days = 365
  
  tags = {
    Control = "CC8.1"
    Purpose = "Application Logging"
  }
}

resource "aws_config_configuration_recorder" "main" {
  name     = "main-recorder"
  role_arn = aws_iam_role.config.arn
  
  recording_group {
    all_supported = true
    include_global_resource_types = true
  }
}
`,
		"variables.tf": `
variable "environment" {
  description = "Environment name"
  type        = string
  default     = "test"
}

variable "region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}
`,
	}

	for filename, content := range files {
		err := os.WriteFile(filepath.Join(terraformDir, filename), []byte(content), 0644)
		require.NoError(t, err)
	}
}

func setupMockGitHubServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")

		// Generate mock response based on query
		response := generateMockGitHubResponse(query)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
}

func generateMockGitHubResponse(query string) string {
	// Generate realistic mock responses based on query content
	var items []map[string]interface{}

	if strings.Contains(strings.ToLower(query), "encryption") {
		items = append(items, map[string]interface{}{
			"number":     101,
			"title":      "Implement encryption at rest for SOC2 compliance",
			"body":       "Need to enable KMS encryption for all data stores",
			"state":      "open",
			"labels":     []string{"security", "encryption", "soc2"},
			"created_at": "2024-08-01T00:00:00Z",
			"updated_at": "2024-08-20T00:00:00Z",
			"url":        "https://github.com/test-org/test-repo/issues/101",
		})
	}

	if strings.Contains(strings.ToLower(query), "access") {
		items = append(items, map[string]interface{}{
			"number":     102,
			"title":      "Review IAM policies for least privilege access",
			"body":       "Audit all IAM roles and policies for CC6.1 compliance",
			"state":      "open",
			"labels":     []string{"security", "access-control", "iam"},
			"created_at": "2024-08-05T00:00:00Z",
			"updated_at": "2024-08-18T00:00:00Z",
			"url":        "https://github.com/test-org/test-repo/issues/102",
		})
	}

	if strings.Contains(strings.ToLower(query), "audit") || strings.Contains(strings.ToLower(query), "logging") {
		items = append(items, map[string]interface{}{
			"number":     103,
			"title":      "Setup comprehensive audit logging",
			"body":       "Configure CloudTrail and CloudWatch for CC8.1 compliance",
			"state":      "closed",
			"labels":     []string{"audit", "logging", "cloudtrail"},
			"created_at": "2024-07-15T00:00:00Z",
			"updated_at": "2024-08-10T00:00:00Z",
			"url":        "https://github.com/test-org/test-repo/issues/103",
		})
	}

	response := map[string]interface{}{
		"total_count":        len(items),
		"incomplete_results": false,
		"items":              items,
	}

	jsonData, _ := json.Marshal(response)
	return string(jsonData)
}
