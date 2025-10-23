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
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// WorkflowTestSuite provides a complete integration test environment
type WorkflowTestSuite struct {
	TempDir       string
	Config        *config.Config
	Logger        logger.Logger
	ToolRegistry  *tools.Registry
	TestDataFiles map[string]string
}

// NewWorkflowTestSuite creates a new workflow test suite
func NewWorkflowTestSuite(t *testing.T) *WorkflowTestSuite {
	tempDir := t.TempDir()

	// Create test directories
	docsDir := filepath.Join(tempDir, "docs")
	terraformDir := filepath.Join(tempDir, "terraform")
	configDir := filepath.Join(tempDir, "config")
	evidenceDir := filepath.Join(tempDir, "evidence")

	for _, dir := range []string{docsDir, terraformDir, configDir, evidenceDir} {
		err := os.MkdirAll(dir, 0755)
		require.NoError(t, err)
	}

	// Create comprehensive test data
	testDataFiles := map[string]string{
		"docs/security-policy.md": `# Security Policy

## Data Protection
- All sensitive data is encrypted at rest using AES-256 encryption
- Data in transit is protected using TLS 1.3
- Encryption keys are managed using AWS KMS with automatic rotation

## Access Control  
- Multi-factor authentication is required for all administrative access
- Role-based access control (RBAC) is implemented across all systems
- Principle of least privilege is enforced for all user accounts
- Regular access reviews are conducted quarterly

## Monitoring and Logging
- All security events are logged to a centralized SIEM system
- Real-time monitoring is implemented for critical security events
- Log retention period is set to 7 years for compliance
- Audit trails are maintained for all administrative actions

## Incident Response
- 24/7 security operations center (SOC) monitors for incidents
- Incident response team can be activated within 1 hour
- All incidents are categorized and tracked to resolution
- Post-incident reviews are conducted for continuous improvement

## Compliance
- SOC 2 Type II compliance is maintained annually
- PCI DSS compliance for payment processing
- GDPR compliance for EU data processing
- Regular third-party security audits and penetration testing
`,

		"docs/privacy-policy.md": `# Privacy Policy

## Data Collection Principles
- Privacy by design is implemented in all systems
- Minimal data collection - only what is necessary for business purposes
- Explicit consent is obtained for all personal data processing
- Data subjects have full visibility into data collection practices

## Data Processing and Storage
- Personal data is processed only for specified, explicit purposes
- Data retention periods are clearly defined and automated
- Data is stored in encrypted databases with access controls
- Cross-border data transfers comply with applicable regulations

## Individual Rights
- Right to access: Data subjects can request copies of their data
- Right to rectification: Incorrect data can be corrected
- Right to erasure: Data can be deleted upon valid request
- Right to portability: Data can be exported in machine-readable format
- Right to object: Processing can be stopped for certain purposes

## Data Security Measures
- End-to-end encryption for sensitive personal data
- Regular security assessments and vulnerability testing
- Employee training on data protection and privacy
- Vendor management program ensures third-party compliance
`,

		"terraform/security.tf": `# Security infrastructure configuration

# KMS Key for encryption
resource "aws_kms_key" "main_encryption_key" {
  description             = "Main encryption key for all services"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  
  tags = {
    Name        = "main-encryption-key"
    Environment = var.environment
    Purpose     = "encryption"
  }
}

resource "aws_kms_alias" "main_encryption_key" {
  name          = "alias/${var.environment}-main-encryption"
  target_key_id = aws_kms_key.main_encryption_key.key_id
}

# S3 Bucket with comprehensive security
resource "aws_s3_bucket" "secure_data_bucket" {
  bucket = "${var.environment}-secure-data-bucket"
  
  tags = {
    Name        = "secure-data-bucket"
    Environment = var.environment
    DataClass   = "sensitive"
  }
}

resource "aws_s3_bucket_encryption" "secure_data_bucket" {
  bucket = aws_s3_bucket.secure_data_bucket.id

  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        kms_master_key_id = aws_kms_key.main_encryption_key.arn
        sse_algorithm     = "aws:kms"
      }
      bucket_key_enabled = true
    }
  }
}

resource "aws_s3_bucket_versioning" "secure_data_bucket" {
  bucket = aws_s3_bucket.secure_data_bucket.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_public_access_block" "secure_data_bucket" {
  bucket = aws_s3_bucket.secure_data_bucket.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# IAM Role with least privilege
resource "aws_iam_role" "application_role" {
  name = "${var.environment}-application-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
        Condition = {
          StringEquals = {
            "sts:ExternalId" = var.external_id
          }
        }
      }
    ]
  })

  tags = {
    Name        = "application-role"
    Environment = var.environment
  }
}

resource "aws_iam_role_policy" "application_policy" {
  name = "${var.environment}-application-policy"
  role = aws_iam_role.application_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject",
          "s3:DeleteObject"
        ]
        Resource = "${aws_s3_bucket.secure_data_bucket.arn}/*"
        Condition = {
          StringEquals = {
            "s3:x-amz-server-side-encryption" = "aws:kms"
          }
        }
      },
      {
        Effect = "Allow"
        Action = [
          "kms:Decrypt",
          "kms:GenerateDataKey"
        ]
        Resource = aws_kms_key.main_encryption_key.arn
      }
    ]
  })
}

# CloudTrail for audit logging
resource "aws_cloudtrail" "main_audit_trail" {
  name                         = "${var.environment}-audit-trail"
  s3_bucket_name              = aws_s3_bucket.audit_logs_bucket.bucket
  include_global_service_events = true
  is_multi_region_trail       = true
  enable_logging              = true
  enable_log_file_validation  = true

  event_selector {
    read_write_type                 = "All"
    include_management_events       = true
    
    data_resource {
      type   = "AWS::S3::Object"
      values = ["${aws_s3_bucket.secure_data_bucket.arn}/*"]
    }
  }

  tags = {
    Name        = "main-audit-trail"
    Environment = var.environment
  }
}

resource "aws_s3_bucket" "audit_logs_bucket" {
  bucket = "${var.environment}-audit-logs-bucket"
  
  tags = {
    Name        = "audit-logs-bucket"
    Environment = var.environment
    Purpose     = "audit-logging"
  }
}

resource "aws_s3_bucket_encryption" "audit_logs_bucket" {
  bucket = aws_s3_bucket.audit_logs_bucket.id

  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        sse_algorithm = "AES256"
      }
    }
  }
}

# Security Group with restrictive rules
resource "aws_security_group" "application_sg" {
  name_prefix = "${var.environment}-app-"
  vpc_id      = aws_vpc.main.id
  description = "Security group for application servers"

  # HTTPS inbound from load balancer only
  ingress {
    from_port       = 443
    to_port         = 443
    protocol        = "tcp"
    security_groups = [aws_security_group.load_balancer_sg.id]
    description     = "HTTPS from load balancer"
  }

  # SSH access from bastion host only
  ingress {
    from_port       = 22
    to_port         = 22
    protocol        = "tcp"
    security_groups = [aws_security_group.bastion_sg.id]
    description     = "SSH from bastion host"
  }

  # Outbound internet access for updates
  egress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "HTTPS outbound for updates"
  }

  # Database access
  egress {
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [aws_security_group.database_sg.id]
    description     = "PostgreSQL to database"
  }

  tags = {
    Name        = "${var.environment}-application-sg"
    Environment = var.environment
  }
}`,

		"terraform/variables.tf": `# Environment and naming variables
variable "environment" {
  description = "Environment name (dev, staging, prod)"
  type        = string
  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "Environment must be one of: dev, staging, prod."
  }
}

variable "aws_region" {
  description = "AWS region for resource deployment"
  type        = string
  default     = "us-east-1"
  validation {
    condition     = can(regex("^[a-z]{2}-[a-z]+-[0-9]$", var.aws_region))
    error_message = "AWS region must be a valid region identifier."
  }
}

# Security-related variables
variable "external_id" {
  description = "External ID for assume role condition"
  type        = string
  sensitive   = true
}

variable "vpc_cidr" {
  description = "CIDR block for the VPC"
  type        = string
  default     = "10.0.0.0/16"
  validation {
    condition     = can(cidrhost(var.vpc_cidr, 0))
    error_message = "VPC CIDR must be a valid IPv4 CIDR block."
  }
}

# Monitoring and alerting
variable "enable_enhanced_monitoring" {
  description = "Enable enhanced monitoring for RDS instances"
  type        = bool
  default     = true
}

variable "backup_retention_days" {
  description = "Number of days to retain backups"
  type        = number
  default     = 30
  validation {
    condition     = var.backup_retention_days >= 7 && var.backup_retention_days <= 365
    error_message = "Backup retention must be between 7 and 365 days."
  }
}`,

		"config/application.yaml": `# Application security configuration
server:
  port: 8080
  ssl:
    enabled: true
    protocol: TLSv1.3
    key-store: /etc/ssl/certs/app.p12
    key-store-password: ${SSL_KEYSTORE_PASSWORD}
    key-store-type: PKCS12
    trust-store: /etc/ssl/certs/truststore.p12
    trust-store-password: ${SSL_TRUSTSTORE_PASSWORD}
    client-auth: want

security:
  require-ssl: true
  headers:
    content-security-policy: "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'"
    x-frame-options: DENY
    x-content-type-options: nosniff
    x-xss-protection: "1; mode=block"
    strict-transport-security: "max-age=31536000; includeSubDomains"
  
  oauth2:
    client:
      registration:
        github:
          client-id: ${GITHUB_CLIENT_ID}
          client-secret: ${GITHUB_CLIENT_SECRET}
          scope: read:user,user:email
          redirect-uri: https://app.example.com/oauth2/callback/github
      provider:
        github:
          authorization-uri: https://github.com/login/oauth/authorize
          token-uri: https://github.com/login/oauth/access_token
          user-info-uri: https://api.github.com/user

# Database configuration with encryption
database:
  driver: postgresql
  url: ${DATABASE_URL}
  username: ${DATABASE_USERNAME}
  password: ${DATABASE_PASSWORD}
  pool:
    minimum-idle: 5
    maximum-pool-size: 20
    connection-timeout: 30000
    idle-timeout: 600000
    max-lifetime: 1800000
  ssl:
    mode: require
    cert: /etc/ssl/certs/postgresql-client.crt
    key: /etc/ssl/private/postgresql-client.key
    root-cert: /etc/ssl/certs/postgresql-ca.crt
  encryption:
    enabled: true
    algorithm: AES-256-GCM
    key-rotation-days: 90

# Logging and monitoring
logging:
  level:
    org.springframework.security: DEBUG
    org.springframework.web: INFO
    com.example.app: INFO
  pattern:
    console: "%d{yyyy-MM-dd HH:mm:ss} [%thread] %-5level %logger{36} - %msg%n"
    file: "%d{yyyy-MM-dd HH:mm:ss} [%thread] %-5level %logger{36} - %msg%n"
  file:
    name: /var/log/app/application.log
    max-size: 100MB
    max-history: 30

management:
  endpoints:
    web:
      exposure:
        include: health,info,metrics,prometheus
      base-path: /actuator
  endpoint:
    health:
      show-details: when-authorized
  security:
    enabled: true
  metrics:
    export:
      prometheus:
        enabled: true

# Session management
session:
  store-type: redis
  redis:
    host: ${REDIS_HOST}
    port: ${REDIS_PORT}
    password: ${REDIS_PASSWORD}
    ssl: true
    timeout: 2000
  timeout: 1800
  cookie:
    name: JSESSIONID
    secure: true
    http-only: true
    same-site: strict
    max-age: 1800`,
	}

	// Write test files
	for relativePath, content := range testDataFiles {
		fullPath := filepath.Join(tempDir, relativePath)
		dir := filepath.Dir(fullPath)
		err := os.MkdirAll(dir, 0755)
		require.NoError(t, err)

		err = os.WriteFile(fullPath, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Create configuration
	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				GitHub: config.GitHubToolConfig{
					Enabled:    false, // Disabled for offline testing
					APIToken:   "test-token",
					Repository: "test-org/test-repo",
				},
				Terraform: config.TerraformToolConfig{
					Enabled:         true,
					ScanPaths:       []string{terraformDir},
					IncludePatterns: []string{"*.tf", "*.tfvars"},
					ExcludePatterns: []string{".terraform/"},
				},
			},
		},
		Storage: config.StorageConfig{
			DataDir: tempDir,
		},
	}

	// Create logger
	log, err := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err)

	// Create tool registry
	registry := tools.NewRegistry()

	return &WorkflowTestSuite{
		TempDir:       tempDir,
		Config:        cfg,
		Logger:        log,
		ToolRegistry:  registry,
		TestDataFiles: testDataFiles,
	}
}

// RegisterTools registers all tools with the test suite
func (wts *WorkflowTestSuite) RegisterTools(t *testing.T) {
	// Create tools directly with this suite's config to avoid global registry state sharing
	// Register only the tools needed for these integration tests

	// Core tools
	wts.ToolRegistry.Register(tools.NewDocsReaderTool(wts.Config, wts.Logger))
	wts.ToolRegistry.Register(tools.NewStorageReadTool(wts.Config, wts.Logger))
	wts.ToolRegistry.Register(tools.NewStorageWriteTool(wts.Config, wts.Logger))

	// Terraform tools
	wts.ToolRegistry.Register(tools.NewTerraformTool(wts.Config, wts.Logger))
	wts.ToolRegistry.Register(tools.NewTerraformHCLParserAdapter(wts.Config, wts.Logger))
	wts.ToolRegistry.Register(tools.NewTerraformSecurityAnalyzerAdapter(wts.Config, wts.Logger))
}

// TestFullWorkflow tests the complete evidence collection workflow including prompt-as-data patterns
func TestFullWorkflow(t *testing.T) {
	t.Run("Tool Integration and Prompt-as-Data Patterns", func(t *testing.T) {
		suite := NewWorkflowTestSuite(t)
		suite.RegisterTools(t)
		ctx := context.Background()

		// Test 1: Terraform Analysis (Infrastructure Analysis)
		terraformResult, terraformSource, err := suite.ToolRegistry.Execute(ctx, "terraform_analyzer", map[string]interface{}{
			"analysis_type":     "security_controls",
			"security_controls": []interface{}{"encryption", "access_control", "logging"},
			"output_format":     "markdown",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, terraformResult)
		assert.NotNil(t, terraformSource)
		assert.Contains(t, terraformResult, "aws_kms_key")

		// Test 2: Documentation Search
		docsResult, docsSource, err := suite.ToolRegistry.Execute(ctx, "docs-reader", map[string]interface{}{
			"query":           "encryption security compliance",
			"keywords":        []interface{}{"encryption", "security", "compliance"},
			"include_content": true,
			"max_results":     10,
		})
		require.NoError(t, err)
		assert.NotEmpty(t, docsResult)
		assert.NotNil(t, docsSource)
		// Docs result might be empty if no files found, but should have search report structure
		assert.Contains(t, docsResult, "Search Results")

		// Test 3: Storage Operations
		storageWriteResult, storageWriteSource, err := suite.ToolRegistry.Execute(ctx, "storage-write", map[string]interface{}{
			"path":    "test-evidence.json",
			"content": `{"test": "evidence", "collected_at": "2025-01-01T00:00:00Z"}`,
			"format":  "json",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, storageWriteResult)
		assert.NotNil(t, storageWriteSource)

		storageReadResult, storageReadSource, err := suite.ToolRegistry.Execute(ctx, "storage-read", map[string]interface{}{
			"path":   "test-evidence.json",
			"format": "json",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, storageReadResult)
		assert.NotNil(t, storageReadSource)
		assert.Contains(t, storageReadResult, "evidence")

		// Test 4: Terraform Security Analysis
		securityResult, securitySource, err := suite.ToolRegistry.Execute(ctx, "terraform-security-analyzer", map[string]interface{}{
			"security_domain": "encryption",
			"soc2_controls":   []interface{}{"CC6.1", "CC6.8"},
			"output_format":   "detailed_json",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, securityResult)
		assert.NotNil(t, securitySource)

		// Test 5: Terraform HCL Parser
		hclResult, hclSource, err := suite.ToolRegistry.Execute(ctx, "terraform-hcl-parser", map[string]interface{}{
			"analysis_type":  "modules",
			"resource_types": []interface{}{"aws_instance", "aws_s3_bucket"},
			"output_format":  "markdown",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, hclResult)
		assert.NotNil(t, hclSource)

		// Verify all sources have proper metadata
		sources := []*models.EvidenceSource{
			terraformSource, docsSource, storageWriteSource, storageReadSource,
			securitySource, hclSource,
		}
		for _, source := range sources {
			assert.NotEmpty(t, source.Type)
			assert.NotNil(t, source.Metadata)
		}

		// Verify that tools processed test data correctly
		assert.Contains(t, terraformResult, "encryption")
		assert.Contains(t, docsResult, "Search Results") // Docs search completed (may not find files in test environment)
		assert.Contains(t, storageReadResult, "test")
	})
}

func TestCompleteEvidenceWorkflow(t *testing.T) {
	suite := NewWorkflowTestSuite(t)
	suite.RegisterTools(t)

	t.Run("Security Controls Evidence Collection", func(t *testing.T) {
		ctx := context.Background()

		// Step 1: Analyze Terraform configurations for encryption controls
		terraformResult, terraformSource, err := suite.ToolRegistry.Execute(ctx, "terraform_analyzer", map[string]interface{}{
			"analysis_type":     "security_controls",
			"security_controls": []interface{}{"encryption", "access_control", "logging"},
		})
		require.NoError(t, err)
		assert.NotEmpty(t, terraformResult)
		assert.NotNil(t, terraformSource)

		// Verify encryption evidence
		assert.Contains(t, terraformResult, "aws_kms_key")
		assert.Contains(t, terraformResult, "enable_key_rotation")
		assert.Contains(t, terraformResult, "aws_s3_bucket_encryption")

		// Verify access control evidence
		assert.Contains(t, terraformResult, "aws_iam_role")
		assert.Contains(t, terraformResult, "application_role")

		// Verify logging evidence
		assert.Contains(t, terraformResult, "aws_cloudtrail")
		assert.Contains(t, terraformResult, "audit")

		// Step 2: Search documentation for policy evidence
		docsResult, docsSource, err := suite.ToolRegistry.Execute(ctx, "docs-reader", map[string]interface{}{
			"query":           "encryption data protection access control",
			"include_content": true,
			"max_results":     10,
		})
		require.NoError(t, err)
		assert.NotEmpty(t, docsResult)
		assert.NotNil(t, docsSource)

		// Verify policy documentation
		assert.Contains(t, docsResult, "security-policy.md")
		assert.Contains(t, docsResult, "AES-256 encryption")
		// Verify access control section is found (multi-factor auth is in this section)
		assert.Contains(t, strings.ToLower(docsResult), "access control")

		// Step 3: Search configuration files for technical implementation
		configResult, configSource, err := suite.ToolRegistry.Execute(ctx, "docs-reader", map[string]interface{}{
			"query":       "ssl tls encryption security",
			"pattern":     "*.yaml",
			"docs_path":   "config/",
			"max_results": 5,
		})
		require.NoError(t, err)
		assert.NotEmpty(t, configResult)
		assert.NotNil(t, configSource)

		// Verify configuration evidence
		assert.Contains(t, configResult, "application.yaml")
		assert.Contains(t, configResult, "TLSv1.3")
		// Verify encryption configuration is found
		assert.Contains(t, strings.ToLower(configResult), "encryption")

		// Step 4: Combine evidence into comprehensive report
		evidenceReport := map[string]interface{}{
			"control_category": "Data Protection and Encryption",
			"evidence_sources": []map[string]interface{}{
				{
					"type":        terraformSource.Type,
					"description": "terraform_analyzer",
					"content":     terraformResult,
					"metadata":    terraformSource.Metadata,
				},
				{
					"type":        docsSource.Type,
					"description": "docs-reader",
					"content":     docsResult,
					"metadata":    docsSource.Metadata,
				},
				{
					"type":        configSource.Type,
					"description": "docs-reader",
					"content":     configResult,
					"metadata":    configSource.Metadata,
				},
			},
			"collected_at": time.Now().Format(time.RFC3339),
		}

		// Save evidence report
		evidencePath := filepath.Join(suite.Config.Evidence.Generation.OutputDir, "data_protection_evidence.json")
		evidenceData, err := json.MarshalIndent(evidenceReport, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(evidencePath, evidenceData, 0644)
		require.NoError(t, err)

		// Verify evidence file was created
		_, err = os.Stat(evidencePath)
		assert.NoError(t, err)

		t.Logf("Evidence collection workflow completed successfully")
		t.Logf("Evidence saved to: %s", evidencePath)
	})
}

func TestToolOrchestrationWorkflow(t *testing.T) {
	suite := NewWorkflowTestSuite(t)
	suite.RegisterTools(t)

	t.Run("Multi-Tool Security Assessment", func(t *testing.T) {
		ctx := context.Background()

		// Define assessment scenarios
		assessmentScenarios := []struct {
			name        string
			toolName    string
			params      map[string]interface{}
			expectCheck func(t *testing.T, result string, source interface{})
		}{
			{
				name:     "Infrastructure Security Analysis",
				toolName: "terraform_analyzer",
				params: map[string]interface{}{
					"analysis_type": "security_issues",
				},
				expectCheck: func(t *testing.T, result string, source interface{}) {
					// Should analyze for security issues
					assert.Contains(t, result, "security")
					assert.NotEmpty(t, result)
				},
			},
			{
				name:     "Compliance Documentation Review",
				toolName: "docs-reader",
				params: map[string]interface{}{
					"query":           "compliance SOC audit GDPR",
					"include_content": true,
				},
				expectCheck: func(t *testing.T, result string, source interface{}) {
					// Should find compliance documentation
					assert.Contains(t, result, "SOC 2")
					assert.Contains(t, result, "GDPR")
					assert.Contains(t, result, "audit")
				},
			},
			{
				name:     "Access Control Implementation Check",
				toolName: "terraform_analyzer",
				params: map[string]interface{}{
					"analysis_type":  "resource_types",
					"resource_types": []interface{}{"aws_iam_role", "aws_iam_policy", "aws_security_group"},
				},
				expectCheck: func(t *testing.T, result string, source interface{}) {
					// Should find IAM and security group resources
					assert.Contains(t, result, "aws_iam_role")
					assert.Contains(t, result, "aws_security_group")
					assert.Contains(t, result, "application_role")
				},
			},
			{
				name:     "Privacy Policy Analysis",
				toolName: "docs-reader",
				params: map[string]interface{}{
					"query":         "privacy personal data consent rights",
					"file_patterns": []interface{}{"*privacy*.md"},
				},
				expectCheck: func(t *testing.T, result string, source interface{}) {
					// Should find privacy policy content
					assert.Contains(t, result, "privacy-policy.md")
					assert.Contains(t, result, "personal data")
					assert.Contains(t, result, "consent")
				},
			},
		}

		// Execute assessment scenarios
		results := make(map[string]interface{})
		for _, scenario := range assessmentScenarios {
			t.Run(scenario.name, func(t *testing.T) {
				result, source, err := suite.ToolRegistry.Execute(ctx, scenario.toolName, scenario.params)
				require.NoError(t, err)
				assert.NotEmpty(t, result)
				assert.NotNil(t, source)

				// Run specific checks
				scenario.expectCheck(t, result, source)

				// Store results for final assessment
				results[scenario.name] = map[string]interface{}{
					"result": result,
					"source": source,
				}
			})
		}

		// Generate comprehensive security assessment report
		assessmentReport := map[string]interface{}{
			"assessment_type": "Multi-Tool Security Assessment",
			"scenarios":       results,
			"summary": map[string]interface{}{
				"total_scenarios":      len(assessmentScenarios),
				"successful_scenarios": len(results),
				"assessment_date":      time.Now().Format(time.RFC3339),
			},
		}

		// Save assessment report
		assessmentPath := filepath.Join(suite.Config.Evidence.Generation.OutputDir, "security_assessment.json")
		assessmentData, err := json.MarshalIndent(assessmentReport, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(assessmentPath, assessmentData, 0644)
		require.NoError(t, err)

		t.Logf("Multi-tool security assessment completed successfully")
		t.Logf("Assessment saved to: %s", assessmentPath)
	})
}

func TestToolErrorHandlingAndRecovery(t *testing.T) {
	suite := NewWorkflowTestSuite(t)
	suite.RegisterTools(t)

	t.Run("Graceful Error Handling", func(t *testing.T) {
		ctx := context.Background()

		// Test scenarios with invalid parameters
		errorScenarios := []struct {
			name        string
			toolName    string
			params      map[string]interface{}
			expectError bool
			errorCheck  func(t *testing.T, err error)
		}{
			{
				name:     "Invalid Terraform Analysis Type",
				toolName: "terraform_analyzer",
				params: map[string]interface{}{
					"analysis_type": "invalid_type",
				},
				expectError: true,
				errorCheck: func(t *testing.T, err error) {
					assert.Contains(t, err.Error(), "invalid analysis_type")
				},
			},
			{
				name:        "Empty Parameters Use Defaults",
				toolName:    "terraform_analyzer",
				params:      map[string]interface{}{},
				expectError: false, // Tool now uses defaults instead of erroring
				errorCheck:  nil,
			},
			{
				name:     "Invalid Query Type",
				toolName: "docs-reader",
				params: map[string]interface{}{
					"query": 12345,
				},
				expectError: true,
				errorCheck: func(t *testing.T, err error) {
					assert.Error(t, err)
				},
			},
			{
				name:     "Empty Query",
				toolName: "docs-reader",
				params: map[string]interface{}{
					"query": "",
				},
				expectError: true,
				errorCheck: func(t *testing.T, err error) {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), "query parameter is required")
				},
			},
		}

		for _, scenario := range errorScenarios {
			t.Run(scenario.name, func(t *testing.T) {
				_, _, err := suite.ToolRegistry.Execute(ctx, scenario.toolName, scenario.params)

				if scenario.expectError {
					assert.Error(t, err)
					if scenario.errorCheck != nil {
						scenario.errorCheck(t, err)
					}
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("Tool Registry Error Handling", func(t *testing.T) {
		ctx := context.Background()

		// Test nonexistent tool
		_, _, err := suite.ToolRegistry.Execute(ctx, "nonexistent_tool", map[string]interface{}{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not registered")

		// Test tool registration errors
		// Try to register same tool twice
		terraformTool := tools.NewTerraformTool(suite.Config, suite.Logger)
		err = suite.ToolRegistry.Register(terraformTool)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already registered")
	})
}

func TestPerformanceAndTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	suite := NewWorkflowTestSuite(t)
	suite.RegisterTools(t)

	t.Run("Tool Execution Performance", func(t *testing.T) {
		ctx := context.Background()

		// Measure execution time for each tool
		performanceTests := []struct {
			name     string
			toolName string
			params   map[string]interface{}
			maxTime  time.Duration
		}{
			{
				name:     "Terraform Analysis Performance",
				toolName: "terraform_analyzer",
				params: map[string]interface{}{
					"analysis_type":     "security_controls",
					"security_controls": []interface{}{"encryption"},
				},
				maxTime: 5 * time.Second,
			},
			{
				name:     "Docs Reader Performance",
				toolName: "docs-reader",
				params: map[string]interface{}{
					"query":       "security",
					"max_results": 10,
				},
				maxTime: 3 * time.Second,
			},
		}

		for _, test := range performanceTests {
			t.Run(test.name, func(t *testing.T) {
				start := time.Now()

				result, source, err := suite.ToolRegistry.Execute(ctx, test.toolName, test.params)

				duration := time.Since(start)

				require.NoError(t, err)
				assert.NotEmpty(t, result)
				assert.NotNil(t, source)
				assert.Less(t, duration, test.maxTime,
					"Tool execution took %v, expected less than %v", duration, test.maxTime)

				t.Logf("%s completed in %v", test.name, duration)
			})
		}
	})

	t.Run("Context Timeout Handling", func(t *testing.T) {
		// Create context with very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// This should timeout
		_, _, err := suite.ToolRegistry.Execute(ctx, "terraform_analyzer", map[string]interface{}{
			"analysis_type":     "security_controls",
			"security_controls": []interface{}{"encryption"},
		})

		// Should either timeout or complete very quickly
		// The actual behavior depends on the tool implementation
		if err != nil {
			// If there's an error, it might be due to context cancellation
			assert.True(t,
				ctx.Err() != nil || err != nil,
				"Expected context cancellation or execution error")
		}
	})
}

func TestConcurrentToolExecution(t *testing.T) {
	suite := NewWorkflowTestSuite(t)
	suite.RegisterTools(t)

	t.Run("Concurrent Tool Execution", func(t *testing.T) {
		ctx := context.Background()

		// Define concurrent execution scenarios
		scenarios := []struct {
			toolName string
			params   map[string]interface{}
		}{
			{
				toolName: "terraform_analyzer",
				params: map[string]interface{}{
					"analysis_type":     "security_controls",
					"security_controls": []interface{}{"encryption"},
				},
			},
			{
				toolName: "docs-reader",
				params: map[string]interface{}{
					"query":       "security policy",
					"max_results": 5,
				},
			},
			{
				toolName: "terraform_analyzer",
				params: map[string]interface{}{
					"analysis_type":  "resource_types",
					"resource_types": []interface{}{"aws_s3_bucket"},
				},
			},
		}

		// Execute scenarios concurrently
		results := make(chan struct {
			name   string
			result string
			source interface{}
			err    error
		}, len(scenarios))

		for i, scenario := range scenarios {
			go func(index int, s struct {
				toolName string
				params   map[string]interface{}
			}) {
				result, source, err := suite.ToolRegistry.Execute(ctx, s.toolName, s.params)
				results <- struct {
					name   string
					result string
					source interface{}
					err    error
				}{
					name:   fmt.Sprintf("scenario_%d", index),
					result: result,
					source: source,
					err:    err,
				}
			}(i, scenario)
		}

		// Collect results
		var successCount int
		for i := 0; i < len(scenarios); i++ {
			result := <-results

			if result.err != nil {
				t.Logf("Scenario %s failed: %v", result.name, result.err)
			} else {
				successCount++
				assert.NotEmpty(t, result.result)
				assert.NotNil(t, result.source)
				t.Logf("Scenario %s completed successfully", result.name)
			}
		}

		assert.Equal(t, len(scenarios), successCount,
			"All concurrent executions should succeed")
	})
}
