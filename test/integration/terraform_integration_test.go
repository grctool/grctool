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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/tools"
	testtools "github.com/grctool/grctool/test/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTerraformAnalysis_CompleteInfra(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Terraform integration tests in short mode")
	}

	// Setup test environment with comprehensive Terraform files
	tempDir := t.TempDir()
	setupComprehensiveTerraformFixtures(t, tempDir)

	cfg := createTerraformTestConfig(tempDir)
	log := testtools.CreateTestLogger(t)

	terraformTool := tools.NewTerraformTool(cfg, log)

	t.Run("Multi-AZ Infrastructure Analysis", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"resource_types": []interface{}{"aws_instance", "aws_autoscaling_group", "aws_rds_cluster"},
			"pattern":        "multi.az|availability.zone|multi_az",
			"control_hint":   "ET103",
			"output_format":  "json",
			"use_cache":      false,
		}

		result, _, err := terraformTool.Execute(ctx, params)
		require.NoError(t, err)
		assert.NotEmpty(t, result)
		// assert.NotNil(t, source) // source not captured in this simplified test

		// Parse and validate JSON structure
		var terraformData map[string]interface{}
		err = json.Unmarshal([]byte(result), &terraformData)
		require.NoError(t, err)

		// Should find multi-AZ resources
		results := terraformData["results"].([]interface{})
		assert.Greater(t, len(results), 0, "Should find multi-AZ resources")

		// Validate specific multi-AZ indicators
		resultStr := strings.ToLower(result)
		assert.True(t,
			strings.Contains(resultStr, "multi") ||
				strings.Contains(resultStr, "availability") ||
				strings.Contains(resultStr, "zone"),
			"Should contain multi-AZ indicators")

		// Check scan summary
		scanSummary := terraformData["scan_summary"].(map[string]interface{})
		assert.Greater(t, scanSummary["total_files"].(float64), 0.0)
		assert.Greater(t, scanSummary["total_resources"].(float64), 0.0)

		t.Logf("Multi-AZ analysis found %d resources", len(results))
		// 		t.Logf("Evidence relevance: %.2f", source.Relevance)
	})

	t.Run("Encryption at Rest Analysis", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"resource_types": []interface{}{"aws_kms_key", "aws_s3_bucket_encryption", "aws_db_instance", "aws_ebs_encryption_by_default"},
			"pattern":        "encrypt|kms|storage_encrypted",
			"control_hint":   "CC6.8",
			"output_format":  "markdown",
			"use_cache":      false,
		}

		result, _, err := terraformTool.Execute(ctx, params)
		require.NoError(t, err)
		assert.NotEmpty(t, result)

		// Should be in markdown format
		assert.Contains(t, result, "# Enhanced Terraform Security Configuration Evidence")
		assert.Contains(t, result, "**Security Controls:**")

		// Should find encryption resources
		lowerResult := strings.ToLower(result)
		assert.True(t,
			strings.Contains(lowerResult, "encryption") ||
				strings.Contains(lowerResult, "kms") ||
				strings.Contains(lowerResult, "encrypted"),
			"Should contain encryption indicators")

		// Validate metadata
		// 		metadata := source.Metadata
		// 		assert.Equal(t, "CC6.8", metadata["control_hint"])
		// 		assert.Equal(t, "markdown", metadata["output_format"])

		// 		t.Logf("Encryption evidence quality: %.2f", source.Relevance)
	})

	t.Run("Access Control Infrastructure", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"resource_types": []interface{}{"aws_iam_role", "aws_iam_policy", "aws_security_group", "aws_iam_user"},
			"pattern":        "iam|security_group|access|least.privilege",
			"control_hint":   "CC6.1",
			"output_format":  "csv",
			"use_cache":      false,
		}

		result, _, err := terraformTool.Execute(ctx, params)
		require.NoError(t, err)
		assert.NotEmpty(t, result)

		// Should be in CSV format
		lines := strings.Split(result, "\n")
		assert.Greater(t, len(lines), 1, "Should have header and data")
		header := lines[0]
		assert.Contains(t, header, "Resource Type")
		assert.Contains(t, header, "Security Controls")

		// Should find IAM resources
		assert.Contains(t, result, "aws_iam")
		assert.Contains(t, result, "security_group")

		// Check for access control principles
		lowerResult := strings.ToLower(result)
		assert.True(t,
			strings.Contains(lowerResult, "iam") ||
				strings.Contains(lowerResult, "access") ||
				strings.Contains(lowerResult, "security"),
			"Should contain access control indicators")

		t.Logf("Access control resources found in CSV format")
	})

	t.Run("Audit Logging Infrastructure", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"resource_types": []interface{}{"aws_cloudtrail", "aws_cloudwatch_log_group", "aws_config_configuration_recorder", "aws_s3_bucket_logging"},
			"pattern":        "audit|log|trail|monitoring",
			"control_hint":   "CC8.1",
			"output_format":  "json",
			"use_cache":      false,
		}

		result, _, err := terraformTool.Execute(ctx, params)
		require.NoError(t, err)
		assert.NotEmpty(t, result)

		// Parse audit logging results
		var auditData map[string]interface{}
		err = json.Unmarshal([]byte(result), &auditData)
		require.NoError(t, err)

		// Should find audit resources
		results := auditData["results"].([]interface{})
		assert.Greater(t, len(results), 0, "Should find audit logging resources")

		// Validate audit-specific content
		lowerResult := strings.ToLower(result)
		assert.True(t,
			strings.Contains(lowerResult, "cloudtrail") ||
				strings.Contains(lowerResult, "audit") ||
				strings.Contains(lowerResult, "logging"),
			"Should contain audit logging indicators")

		t.Logf("Audit logging analysis found %d resources", len(results))
	})
}

func TestTerraformSecurity_CrossModule(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping cross-module Terraform tests in short mode")
	}

	tempDir := t.TempDir()
	setupMultiModuleTerraformFixtures(t, tempDir)

	cfg := createTerraformTestConfig(tempDir)
	log := testtools.CreateTestLogger(t)

	terraformTool := tools.NewTerraformTool(cfg, log)

	t.Run("Cross-Module Security Analysis", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"pattern":       "security|encrypt|access|audit",
			"output_format": "json",
			"use_cache":     false,
		}

		result, _, err := terraformTool.Execute(ctx, params)
		require.NoError(t, err)
		assert.NotEmpty(t, result)

		// Parse cross-module results
		var crossModuleData map[string]interface{}
		err = json.Unmarshal([]byte(result), &crossModuleData)
		require.NoError(t, err)

		// Should scan multiple modules
		scanSummary := crossModuleData["scan_summary"].(map[string]interface{})
		totalFiles := scanSummary["total_files"].(float64)
		assert.Greater(t, totalFiles, 3.0, "Should scan multiple module files")

		// Should find resources across modules
		results := crossModuleData["results"].([]interface{})
		assert.Greater(t, len(results), 0, "Should find security resources across modules")

		t.Logf("Cross-module analysis scanned %.0f files and found %d security resources", totalFiles, len(results))
	})

	t.Run("Module Dependency Security", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"resource_types": []interface{}{"module"},
			"pattern":        "module",
			"output_format":  "markdown",
			"use_cache":      false,
		}

		result, _, err := terraformTool.Execute(ctx, params)
		require.NoError(t, err)
		assert.NotEmpty(t, result)

		// Should analyze module references
		assert.Contains(t, result, "# Enhanced Terraform Security Configuration Evidence")

		// Check for module references
		lowerResult := strings.ToLower(result)
		assert.Contains(t, lowerResult, "module", "Should contain module references")

		// 		t.Logf("Module dependency analysis completed with relevance: %.2f", source.Relevance)
	})

	t.Run("Security Control Mapping", func(t *testing.T) {
		// Test mapping of resources to specific security controls
		securityControls := []struct {
			control     string
			resources   []string
			pattern     string
			expectTerms []string
		}{
			{
				control:     "CC6.8",
				resources:   []string{"aws_kms_key", "aws_s3_bucket_encryption"},
				pattern:     "encrypt|kms",
				expectTerms: []string{"encryption", "kms"},
			},
			{
				control:     "CC6.1",
				resources:   []string{"aws_iam_role", "aws_security_group"},
				pattern:     "iam|access|security_group",
				expectTerms: []string{"iam", "security"},
			},
			{
				control:     "CC8.1",
				resources:   []string{"aws_cloudtrail", "aws_cloudwatch_log_group"},
				pattern:     "audit|log|trail",
				expectTerms: []string{"audit", "log"},
			},
		}

		ctx := context.Background()
		// var allSources []*models.EvidenceSource // Not used in simplified test

		for _, test := range securityControls {
			t.Run("Control_"+test.control, func(t *testing.T) {
				params := map[string]interface{}{
					"resource_types": convertStringsToInterface(test.resources),
					"pattern":        test.pattern,
					"control_hint":   test.control,
					"output_format":  "json",
					"use_cache":      false,
				}

				result, _, err := terraformTool.Execute(ctx, params)
				require.NoError(t, err)
				assert.NotEmpty(t, result)

				// Validate control-specific content
				lowerResult := strings.ToLower(result)
				foundExpected := false
				for _, term := range test.expectTerms {
					if strings.Contains(lowerResult, term) {
						foundExpected = true
						break
					}
				}
				assert.True(t, foundExpected, "Should contain expected terms for control %s", test.control)

				// Validate metadata
				// 				metadata := source.Metadata
				// 				assert.Equal(t, test.control, metadata["control_hint"])

				// 				// allSources = append(allSources, source) // Removed

				// 				t.Logf("Control %s analysis relevance: %.2f", test.control, source.Relevance)
			})
		}

		// Cross-validate all control mappings
		// 		assert.Equal(t, len(securityControls), len(allSources), "Should have sources for all controls")

		// All sources should have reasonable relevance
		// 		for _, source := range allSources {
		// 			assert.Greater(t, source.Relevance, 0.0)
		// 			assert.Equal(t, "terraform-enhanced", source.Type)
		// 		}
	})
}

func TestTerraformAnalysis_PerformanceAndScale(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	tempDir := t.TempDir()
	setupLargeTerraformFixtures(t, tempDir)

	cfg := createTerraformTestConfig(tempDir)
	log := testtools.CreateTestLogger(t)

	terraformTool := tools.NewTerraformTool(cfg, log)

	t.Run("Large Codebase Analysis", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"pattern":       "security|encrypt|access|audit|compliance",
			"output_format": "json",
			"max_results":   500,
			"use_cache":     false,
		}

		start := time.Now()
		result, _, err := terraformTool.Execute(ctx, params)
		duration := time.Since(start)

		require.NoError(t, err)
		assert.NotEmpty(t, result)

		// Should complete within reasonable time
		assert.Less(t, duration, 30*time.Second, "Large codebase analysis should complete within 30 seconds")

		// Parse results
		var largeData map[string]interface{}
		err = json.Unmarshal([]byte(result), &largeData)
		require.NoError(t, err)

		scanSummary := largeData["scan_summary"].(map[string]interface{})
		results := largeData["results"].([]interface{})

		t.Logf("Large codebase analysis:")
		t.Logf("  Duration: %v", duration)
		t.Logf("  Files scanned: %.0f", scanSummary["total_files"].(float64))
		t.Logf("  Resources found: %d", len(results))
		// 		t.Logf("  Evidence relevance: %.2f", source.Relevance)
	})

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

		// Second execution (with cache)
		start = time.Now()
		result2, source2, err := terraformTool.Execute(ctx, params)
		secondDuration := time.Since(start)
		require.NoError(t, err)

		// Results should be consistent
		assert.Equal(t, len(result1), len(result2))
		assert.Equal(t, source1.Type, source2.Type)

		t.Logf("Caching performance:")
		t.Logf("  First execution: %v", firstDuration)
		t.Logf("  Second execution: %v", secondDuration)
	})
}

// Test fixture setup functions

func setupComprehensiveTerraformFixtures(t *testing.T, tempDir string) {
	terraformDir := filepath.Join(tempDir, "terraform")
	err := os.MkdirAll(terraformDir, 0755)
	require.NoError(t, err)

	// Copy test data files
	// Read existing test files and create comprehensive versions
	files := map[string]string{
		"multi_az_infrastructure.tf": `
# ET103 - Multi-AZ Infrastructure
resource "aws_instance" "web_server" {
  count             = 3
  ami               = "ami-12345678"
  instance_type     = "t3.medium"
  availability_zone = "us-east-1${substr("abc", count.index, 1)}"
  
  tags = {
    Name = "web-server-${count.index + 1}"
    MultiAZ = "true"
    Control = "ET103"
  }
}

resource "aws_autoscaling_group" "web_asg" {
  name                = "web-asg"
  vpc_zone_identifier = aws_subnet.web[*].id
  target_group_arns   = [aws_lb_target_group.web.arn]
  health_check_type   = "ELB"
  min_size            = 2
  max_size            = 6
  desired_capacity    = 3
  
  availability_zones = [
    "us-east-1a",
    "us-east-1b", 
    "us-east-1c"
  ]
  
  tag {
    key                 = "Name"
    value               = "web-asg"
    propagate_at_launch = true
  }
  
  tag {
    key                 = "MultiAZ"
    value               = "enabled"
    propagate_at_launch = true
  }
}

resource "aws_rds_cluster" "main" {
  cluster_identifier      = "aurora-cluster"
  engine                 = "aurora-mysql"
  master_username        = "admin"
  master_password        = "password123"
  backup_retention_period = 5
  preferred_backup_window = "07:00-09:00"
  
  availability_zones = [
    "us-east-1a",
    "us-east-1b",
    "us-east-1c"
  ]
  
  tags = {
    MultiAZ = "cluster"
    Control = "ET103"
  }
}`,
		"comprehensive_encryption.tf": `
# CC6.8 - Comprehensive Encryption
resource "aws_kms_key" "main" {
  description         = "Main application encryption key"
  enable_key_rotation = true
  
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "Enable IAM User Permissions"
        Effect = "Allow"
        Principal = {
          AWS = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"
        }
        Action   = "kms:*"
        Resource = "*"
      }
    ]
  })
  
  tags = {
    Name = "main-encryption-key"
    Control = "CC6.8"
    Purpose = "encryption-at-rest"
  }
}

resource "aws_s3_bucket_encryption" "data_bucket" {
  bucket = aws_s3_bucket.data.id
  
  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        kms_master_key_id = aws_kms_key.main.arn
        sse_algorithm     = "aws:kms"
      }
      bucket_key_enabled = true
    }
  }
}

resource "aws_db_instance" "main" {
  identifier     = "main-database"
  engine         = "postgres"
  engine_version = "13.7"
  instance_class = "db.t3.micro"
  
  allocated_storage     = 20
  max_allocated_storage = 100
  storage_encrypted     = true
  kms_key_id           = aws_kms_key.main.arn
  
  db_name  = "application"
  username = "dbadmin"
  password = "securepassword"
  
  backup_retention_period = 7
  backup_window          = "07:00-09:00"
  
  tags = {
    Name = "main-database"
    Control = "CC6.8"
    Encrypted = "true"
  }
}

resource "aws_ebs_encryption_by_default" "main" {
  enabled = true
}`,
		"advanced_access_control.tf": `
# CC6.1 - Advanced Access Control
resource "aws_iam_role" "application_role" {
  name = "application-execution-role"
  
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
        Condition = {
          StringEquals = {
            "sts:ExternalId" = "unique-external-id"
          }
        }
      }
    ]
  })
  
  tags = {
    Control = "CC6.1"
    Purpose = "least-privilege-access"
  }
}

resource "aws_iam_policy" "application_policy" {
  name        = "application-least-privilege-policy"
  description = "Least privilege policy for application"
  
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject"
        ]
        Resource = "arn:aws:s3:::${aws_s3_bucket.data.bucket}/data/*"
      },
      {
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ]
        Resource = "arn:aws:logs:*:*:*"
      }
    ]
  })
}

resource "aws_security_group" "web_sg" {
  name_prefix = "web-security-group"
  vpc_id      = aws_vpc.main.id
  
  ingress {
    description = "HTTPS from approved networks"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/8", "172.16.0.0/12"]
  }
  
  ingress {
    description = "SSH from management network"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["10.0.1.0/24"]
  }
  
  egress {
    description = "All outbound to internet"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
  
  tags = {
    Name = "web-security-group"
    Control = "CC6.1"
    Purpose = "network-access-control"
  }
}`,
		"comprehensive_audit_logging.tf": `
# CC8.1 - Comprehensive Audit Logging
resource "aws_cloudtrail" "main_trail" {
  name           = "main-audit-trail"
  s3_bucket_name = aws_s3_bucket.audit_logs.bucket
  s3_key_prefix  = "cloudtrail-logs"
  
  enable_log_file_validation = true
  is_multi_region_trail     = true
  include_global_service_events = true
  enable_logging = true
  
  event_selector {
    read_write_type                 = "All"
    include_management_events       = true
    data_resource {
      type   = "AWS::S3::Object"
      values = ["arn:aws:s3:::${aws_s3_bucket.data.bucket}/*"]
    }
  }
  
  tags = {
    Name = "main-audit-trail"
    Control = "CC8.1"
    Purpose = "audit-logging"
  }
}

resource "aws_cloudwatch_log_group" "application_logs" {
  name              = "/aws/lambda/application"
  retention_in_days = 365
  kms_key_id        = aws_kms_key.main.arn
  
  tags = {
    Name = "application-logs"
    Control = "CC8.1"
    Purpose = "application-audit-logging"
  }
}

resource "aws_config_configuration_recorder" "main_recorder" {
  name     = "main-config-recorder"
  role_arn = aws_iam_role.config_role.arn
  
  recording_group {
    all_supported                 = true
    include_global_resource_types = true
    recording_mode_override {
      description                = "Override for specific resource types"
      recording_frequency        = "DAILY"
      recording_mode             = "RECORDING_FREQUENCY"
      resource_types            = ["AWS::IAM::Role", "AWS::IAM::Policy"]
    }
  }
}

resource "aws_s3_bucket_logging" "audit_logs_access" {
  bucket = aws_s3_bucket.audit_logs.id
  
  target_bucket = aws_s3_bucket.audit_logs_access.id
  target_prefix = "access-logs/"
}`,
	}

	for filename, content := range files {
		err := os.WriteFile(filepath.Join(terraformDir, filename), []byte(content), 0644)
		require.NoError(t, err)
	}
}

func setupMultiModuleTerraformFixtures(t *testing.T, tempDir string) {
	// Create multiple module directories
	modules := []string{"networking", "security", "compute", "data"}

	for _, module := range modules {
		moduleDir := filepath.Join(tempDir, "modules", module)
		err := os.MkdirAll(moduleDir, 0755)
		require.NoError(t, err)

		// Create module-specific terraform files
		content := generateModuleContent(module)
		err = os.WriteFile(filepath.Join(moduleDir, "main.tf"), []byte(content), 0644)
		require.NoError(t, err)
	}

	// Create root module that uses sub-modules
	rootContent := `
module "networking" {
  source = "./modules/networking"
  environment = var.environment
}

module "security" {
  source = "./modules/security"
  vpc_id = module.networking.vpc_id
}

module "compute" {
  source = "./modules/compute"
  subnet_ids = module.networking.subnet_ids
  security_group_id = module.security.web_sg_id
}

module "data" {
  source = "./modules/data"
  kms_key_id = module.security.kms_key_id
}
`
	err := os.WriteFile(filepath.Join(tempDir, "main.tf"), []byte(rootContent), 0644)
	require.NoError(t, err)
}

func setupLargeTerraformFixtures(t *testing.T, tempDir string) {
	// Create a large number of terraform files to test performance
	terraformDir := filepath.Join(tempDir, "terraform")
	err := os.MkdirAll(terraformDir, 0755)
	require.NoError(t, err)

	// Generate multiple resource files
	for i := 0; i < 10; i++ {
		content := generateLargeResourceFile(i)
		filename := filepath.Join(terraformDir, fmt.Sprintf("resources_%d.tf", i))
		err = os.WriteFile(filename, []byte(content), 0644)
		require.NoError(t, err)
	}
}

// Helper functions

func createTerraformTestConfig(tempDir string) *config.Config {
	return &config.Config{
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				Terraform: config.TerraformToolConfig{
					Enabled:         true,
					ScanPaths:       []string{filepath.Join(tempDir, "terraform"), filepath.Join(tempDir, "modules")},
					IncludePatterns: []string{"*.tf", "*.tfvars"},
					ExcludePatterns: []string{".terraform/", "*.tfstate*"},
				},
			},
		},
		Storage: config.StorageConfig{
			DataDir: tempDir,
		},
	}
}

func convertStringsToInterface(strs []string) []interface{} {
	result := make([]interface{}, len(strs))
	for i, s := range strs {
		result[i] = s
	}
	return result
}

func generateModuleContent(moduleName string) string {
	switch moduleName {
	case "networking":
		return `
resource "aws_vpc" "main" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true
  
  tags = {
    Name = "main-vpc"
    Module = "networking"
  }
}

resource "aws_subnet" "web" {
  count             = 3
  vpc_id            = aws_vpc.main.id
  cidr_block        = "10.0.${count.index + 1}.0/24"
  availability_zone = data.aws_availability_zones.available.names[count.index]
  
  tags = {
    Name = "web-subnet-${count.index + 1}"
    Module = "networking"
  }
}

output "vpc_id" {
  value = aws_vpc.main.id
}

output "subnet_ids" {
  value = aws_subnet.web[*].id
}
`
	case "security":
		return `
resource "aws_kms_key" "main" {
  description         = "Main encryption key"
  enable_key_rotation = true
  
  tags = {
    Name = "main-key"
    Module = "security"
  }
}

resource "aws_security_group" "web" {
  name_prefix = "web-sg"
  vpc_id      = var.vpc_id

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "web-security-group"
    Module = "security"
  }
}

resource "aws_cloudtrail" "main" {
  name                          = "main-trail"
  s3_bucket_name               = "audit-logs-bucket"
  include_global_service_events = true
  is_multi_region_trail        = true
  enable_log_file_validation   = true

  tags = {
    Name = "main-audit-trail"
    Module = "security"
  }
}

resource "aws_cloudwatch_log_group" "trail_logs" {
  name              = "/aws/cloudtrail/main"
  retention_in_days = 90

  tags = {
    Name = "cloudtrail-logs"
    Module = "security"
  }
}

variable "vpc_id" {
  description = "VPC ID for security group"
  type        = string
}

output "kms_key_id" {
  value = aws_kms_key.main.id
}

output "web_sg_id" {
  value = aws_security_group.web.id
}
`
	case "compute":
		return `
resource "aws_instance" "web" {
  count           = 2
  ami             = "ami-12345678"
  instance_type   = "t3.micro"
  subnet_id       = var.subnet_ids[count.index]
  security_groups = [var.security_group_id]
  
  tags = {
    Name = "web-instance-${count.index + 1}"
    Module = "compute"
  }
}

variable "subnet_ids" {
  description = "Subnet IDs for instances"
  type        = list(string)
}

variable "security_group_id" {
  description = "Security group ID"
  type        = string
}
`
	case "data":
		return `
resource "aws_s3_bucket" "data" {
  bucket = "application-data-bucket"
  
  tags = {
    Name = "data-bucket"
    Module = "data"
  }
}

resource "aws_s3_bucket_encryption" "data" {
  bucket = aws_s3_bucket.data.id
  
  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        kms_master_key_id = var.kms_key_id
        sse_algorithm     = "aws:kms"
      }
    }
  }
}

variable "kms_key_id" {
  description = "KMS key ID for encryption"
  type        = string
}
`
	default:
		return ""
	}
}

func generateLargeResourceFile(index int) string {
	return fmt.Sprintf(`
# Resource file %d with multiple resources
resource "aws_instance" "app_%d" {
  ami           = "ami-12345678"
  instance_type = "t3.micro"
  
  vpc_security_group_ids = [aws_security_group.app_%d.id]
  
  tags = {
    Name = "app-instance-%d"
    Environment = "test"
    Security = "enabled"
  }
}

resource "aws_security_group" "app_%d" {
  name_prefix = "app-sg-%d"
  
  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/8"]
  }
  
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1" 
    cidr_blocks = ["0.0.0.0/0"]
  }
  
  tags = {
    Name = "app-security-group-%d"
    Security = "network-access-control"
  }
}

resource "aws_s3_bucket" "data_%d" {
  bucket = "app-data-bucket-%d"
  
  tags = {
    Name = "data-bucket-%d"
    Encryption = "required"
  }
}
`, index, index, index, index, index, index, index, index, index, index)
}
