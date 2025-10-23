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

package terraform

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
)

// SnippetsTool provides Terraform configuration snippet suggestions
type SnippetsTool struct {
	terraformTool *Analyzer // Use the new consolidated Analyzer
	config        *config.TerraformToolConfig
	logger        logger.Logger
}

// NewSnippetsTool creates a new TerraformSnippetsTool
func NewSnippetsTool(cfg *config.Config, log logger.Logger) *SnippetsTool {
	return &SnippetsTool{
		terraformTool: NewAnalyzer(cfg, log),
		config:        &cfg.Evidence.Tools.Terraform,
		logger:        log,
	}
}

// Name returns the tool name
func (tst *SnippetsTool) Name() string {
	return "terraform_snippets"
}

// Description returns the tool description
func (tst *SnippetsTool) Description() string {
	return "Suggests Terraform configuration snippets based on existing patterns and security controls"
}

// GetClaudeToolDefinition returns the tool definition for Claude
func (tst *SnippetsTool) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        tst.Name(),
		Description: tst.Description(),
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"control_codes": map[string]interface{}{
					"type":        "array",
					"description": "Security control codes to generate snippets for (e.g., CC6.1, CC6.8)",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"resource_type": map[string]interface{}{
					"type":        "string",
					"description": "Specific resource type to generate snippets for (e.g., aws_iam_role, aws_s3_bucket)",
				},
				"pattern_search": map[string]interface{}{
					"type":        "string",
					"description": "Search for existing patterns containing this text (e.g., 'encryption', 'policy')",
				},
				"include_examples": map[string]interface{}{
					"type":        "boolean",
					"description": "Include examples from existing codebase",
					"default":     true,
				},
			},
			"required": []string{},
		},
	}
}

// Execute runs the snippet suggestion tool
func (tst *SnippetsTool) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	tst.logger.Debug("Executing Terraform snippets tool", logger.Field{Key: "params", Value: params})

	// Extract parameters
	var controlCodes []string
	if cc, ok := params["control_codes"].([]interface{}); ok {
		for _, c := range cc {
			if str, ok := c.(string); ok {
				controlCodes = append(controlCodes, str)
			}
		}
	}

	resourceType := ""
	if rt, ok := params["resource_type"].(string); ok {
		resourceType = rt
	}

	patternSearch := ""
	if ps, ok := params["pattern_search"].(string); ok {
		patternSearch = ps
	}

	includeExamples := true
	if ie, ok := params["include_examples"].(bool); ok {
		includeExamples = ie
	}

	// Generate snippets based on parameters
	snippets := tst.generateSnippets(ctx, controlCodes, resourceType, patternSearch, includeExamples)

	// Create evidence source
	source := &models.EvidenceSource{
		Type:        "terraform_snippets",
		Resource:    fmt.Sprintf("Generated %d Terraform snippets", len(snippets)),
		Content:     tst.formatSnippets(snippets),
		Relevance:   0.8,
		ExtractedAt: time.Now(),
		Metadata: map[string]interface{}{
			"snippet_count":    len(snippets),
			"control_codes":    controlCodes,
			"resource_type":    resourceType,
			"pattern_search":   patternSearch,
			"include_examples": includeExamples,
		},
	}

	return source.Content, source, nil
}

// generateSnippets generates Terraform configuration snippets
func (tst *SnippetsTool) generateSnippets(ctx context.Context, controlCodes []string, resourceType string, patternSearch string, includeExamples bool) []TerraformSnippet {
	var snippets []TerraformSnippet

	// If specific control codes are requested, generate snippets for those
	if len(controlCodes) > 0 {
		snippets = append(snippets, tst.generateSnippetsForControls(ctx, controlCodes, includeExamples)...)
	}

	// If specific resource type is requested, generate snippets for that
	if resourceType != "" {
		snippets = append(snippets, tst.generateSnippetsForResourceType(ctx, resourceType, includeExamples)...)
	}

	// If pattern search is requested, find matching patterns
	if patternSearch != "" {
		snippets = append(snippets, tst.findMatchingPatterns(ctx, patternSearch, includeExamples)...)
	}

	// If no specific criteria, provide common security-related snippets
	if len(controlCodes) == 0 && resourceType == "" && patternSearch == "" {
		snippets = append(snippets, tst.getCommonSecuritySnippets()...)
	}

	return snippets
}

// generateSnippetsForControls generates snippets for specific security controls
func (tst *SnippetsTool) generateSnippetsForControls(ctx context.Context, controlCodes []string, includeExamples bool) []TerraformSnippet {
	var snippets []TerraformSnippet

	// Control-specific snippet templates
	controlSnippets := map[string][]TerraformSnippet{
		"CC6.1": { // Logical and Physical Access Controls
			{
				ResourceType: "aws_iam_role",
				Name:         "Least Privilege IAM Role",
				Description:  "IAM role with minimal required permissions",
				ControlCodes: []string{"CC6.1", "CC6.3"},
				Configuration: `resource "aws_iam_role" "app_role" {
  name               = "app-least-privilege-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
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
    }]
  })

  tags = {
    Security = "least-privilege"
    Control  = "CC6.1"
  }
}`,
				SecurityFeatures: []string{"Least privilege", "External ID", "Service-specific assume role"},
			},
		},
		"CC6.6": { // Network Security
			{
				ResourceType: "aws_security_group",
				Name:         "Restrictive Security Group",
				Description:  "Security group with minimal ingress rules",
				ControlCodes: []string{"CC6.6", "CC7.1"},
				Configuration: `resource "aws_security_group" "app_sg" {
  name        = "app-restrictive-sg"
  description = "Security group with minimal access"
  vpc_id      = var.vpc_id

  # Deny all ingress by default
  ingress {
    description = "HTTPS from allowed IPs"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = var.allowed_cidr_blocks
  }

  # Allow all egress for updates
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Security = "restrictive"
    Control  = "CC6.6"
  }
}`,
				SecurityFeatures: []string{"Minimal ingress", "Specific port access", "IP allowlisting"},
			},
		},
		"CC6.7": { // SSL/TLS Configuration
			{
				ResourceType: "aws_lb_listener",
				Name:         "HTTPS Load Balancer Listener",
				Description:  "Load balancer with TLS 1.2+ enforcement",
				ControlCodes: []string{"CC6.7"},
				Configuration: `resource "aws_lb_listener" "https" {
  load_balancer_arn = aws_lb.main.arn
  port              = "443"
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-TLS-1-2-2017-01"
  certificate_arn   = aws_acm_certificate.cert.arn

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.app.arn
  }

  tags = {
    Security = "tls-1.2-minimum"
    Control  = "CC6.7"
  }
}`,
				SecurityFeatures: []string{"TLS 1.2+", "ACM certificate", "Secure SSL policy"},
			},
		},
		"CC6.8": { // Data Protection
			{
				ResourceType: "aws_s3_bucket",
				Name:         "Encrypted S3 Bucket",
				Description:  "S3 bucket with encryption and access controls",
				ControlCodes: []string{"CC6.8"},
				Configuration: `resource "aws_s3_bucket" "secure_bucket" {
  bucket = "secure-data-bucket"

  tags = {
    Security = "encrypted"
    Control  = "CC6.8"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "secure_bucket_encryption" {
  bucket = aws_s3_bucket.secure_bucket.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm     = "aws:kms"
      kms_master_key_id = aws_kms_key.bucket_key.arn
    }
  }
}

resource "aws_s3_bucket_public_access_block" "secure_bucket_pab" {
  bucket = aws_s3_bucket.secure_bucket.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}`,
				SecurityFeatures: []string{"KMS encryption", "Public access blocked", "Encryption by default"},
			},
		},
		"CC7.2": { // Monitoring and Logging
			{
				ResourceType: "aws_cloudtrail",
				Name:         "CloudTrail with S3 Logging",
				Description:  "CloudTrail configuration for audit logging",
				ControlCodes: []string{"CC7.2", "CC7.4"},
				Configuration: `resource "aws_cloudtrail" "main" {
  name                          = "main-trail"
  s3_bucket_name               = aws_s3_bucket.trail.id
  include_global_service_events = true
  is_multi_region_trail        = true
  enable_logging               = true
  enable_log_file_validation   = true

  event_selector {
    read_write_type           = "All"
    include_management_events = true

    data_resource {
      type   = "AWS::S3::Object"
      values = ["arn:aws:s3:::*/*"]
    }
  }

  tags = {
    Security = "audit-logging"
    Control  = "CC7.2"
  }
}`,
				SecurityFeatures: []string{"Multi-region", "Log validation", "S3 object logging"},
			},
		},
	}

	// Add snippets for requested control codes
	for _, code := range controlCodes {
		if snippetList, exists := controlSnippets[code]; exists {
			snippets = append(snippets, snippetList...)
		}
	}

	// If examples are requested, scan existing codebase
	if includeExamples && len(snippets) > 0 {
		tst.enrichWithExamples(ctx, snippets)
	}

	return snippets
}

// generateSnippetsForResourceType generates snippets for a specific resource type
func (tst *SnippetsTool) generateSnippetsForResourceType(ctx context.Context, resourceType string, includeExamples bool) []TerraformSnippet {
	var snippets []TerraformSnippet

	// Resource-specific templates
	resourceTemplates := map[string]TerraformSnippet{
		"aws_kms_key": {
			ResourceType: "aws_kms_key",
			Name:         "Customer Managed KMS Key",
			Description:  "KMS key with rotation enabled",
			ControlCodes: []string{"CC6.8"},
			Configuration: `resource "aws_kms_key" "main" {
  description             = "Customer managed key for encryption"
  deletion_window_in_days = 30
  enable_key_rotation     = true

  tags = {
    Security = "encryption"
    Control  = "CC6.8"
  }
}

resource "aws_kms_alias" "main" {
  name          = "alias/main-encryption-key"
  target_key_id = aws_kms_key.main.key_id
}`,
			SecurityFeatures: []string{"Key rotation", "30-day deletion window", "Customer managed"},
		},
		"aws_config_configuration_recorder": {
			ResourceType: "aws_config_configuration_recorder",
			Name:         "AWS Config Recorder",
			Description:  "Configuration recorder for compliance monitoring",
			ControlCodes: []string{"CC7.2", "CC8.1"},
			Configuration: `resource "aws_config_configuration_recorder" "main" {
  name     = "main-recorder"
  role_arn = aws_iam_role.config.arn

  recording_group {
    all_supported                 = true
    include_global_resource_types = true
  }
}

resource "aws_config_configuration_recorder_status" "main" {
  name       = aws_config_configuration_recorder.main.name
  is_enabled = true
}`,
			SecurityFeatures: []string{"All resources", "Global resources", "Continuous monitoring"},
		},
	}

	if template, exists := resourceTemplates[resourceType]; exists {
		snippets = append(snippets, template)
	}

	// If examples are requested, find existing usage
	if includeExamples {
		examples, _ := tst.terraformTool.ScanForResources(ctx, []string{resourceType})
		if len(examples) > 0 {
			for i := range snippets {
				snippets[i].ExamplePath = examples[0].FilePath
			}
		}
	}

	return snippets
}

// findMatchingPatterns searches for patterns in existing Terraform files
func (tst *SnippetsTool) findMatchingPatterns(ctx context.Context, pattern string, includeExamples bool) []TerraformSnippet {
	var snippets []TerraformSnippet

	if !includeExamples {
		return snippets
	}

	// Scan all Terraform files
	results, err := tst.terraformTool.ScanForResources(ctx, []string{})
	if err != nil {
		return snippets
	}

	// Filter results that match the pattern
	patternLower := strings.ToLower(pattern)
	seen := make(map[string]bool)

	for _, result := range results {
		// Check if configuration contains the pattern
		configStr := fmt.Sprintf("%v", result.Configuration)
		if strings.Contains(strings.ToLower(configStr), patternLower) ||
			strings.Contains(strings.ToLower(result.ResourceType), patternLower) {

			// Avoid duplicates
			key := fmt.Sprintf("%s:%s", result.ResourceType, result.ResourceName)
			if seen[key] {
				continue
			}
			seen[key] = true

			// Extract the actual content if available
			content := ""
			if c, ok := result.Configuration["_content"].(string); ok {
				content = c
			}

			snippet := TerraformSnippet{
				ResourceType:     result.ResourceType,
				Name:             fmt.Sprintf("Example: %s", result.ResourceName),
				Description:      fmt.Sprintf("Found in %s", result.FilePath),
				ControlCodes:     result.SecurityRelevance,
				Configuration:    content,
				ExamplePath:      result.FilePath,
				SecurityFeatures: tst.extractSecurityFeatures(result),
			}

			snippets = append(snippets, snippet)

			// Limit results
			if len(snippets) >= 5 {
				break
			}
		}
	}

	return snippets
}

// getCommonSecuritySnippets returns commonly needed security snippets
func (tst *SnippetsTool) getCommonSecuritySnippets() []TerraformSnippet {
	return []TerraformSnippet{
		{
			ResourceType: "aws_iam_policy_document",
			Name:         "Least Privilege Policy Document",
			Description:  "Policy document template with conditions",
			ControlCodes: []string{"CC6.1", "CC6.3"},
			Configuration: `data "aws_iam_policy_document" "least_privilege" {
  statement {
    sid    = "MinimalAccess"
    effect = "Allow"
    
    actions = [
      "s3:GetObject",
      "s3:ListBucket"
    ]
    
    resources = [
      aws_s3_bucket.app.arn,
      "${aws_s3_bucket.app.arn}/*"
    ]
    
    condition {
      test     = "StringEquals"
      variable = "s3:ExistingObjectTag/Environment"
      values   = ["production"]
    }
  }
}`,
			SecurityFeatures: []string{"Conditional access", "Resource-specific", "Minimal actions"},
		},
		{
			ResourceType: "aws_vpc",
			Name:         "Secure VPC with Private Subnets",
			Description:  "VPC configuration with network segmentation",
			ControlCodes: []string{"CC6.6", "CC7.1"},
			Configuration: `resource "aws_vpc" "main" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name     = "secure-vpc"
    Security = "network-segmentation"
  }
}

resource "aws_subnet" "private" {
  count             = 2
  vpc_id            = aws_vpc.main.id
  cidr_block        = "10.0.${count.index + 1}.0/24"
  availability_zone = data.aws_availability_zones.available.names[count.index]

  tags = {
    Name = "private-subnet-${count.index + 1}"
    Type = "private"
  }
}`,
			SecurityFeatures: []string{"Private subnets", "Network segmentation", "Multi-AZ"},
		},
	}
}

// enrichWithExamples adds example paths from existing codebase
func (tst *SnippetsTool) enrichWithExamples(ctx context.Context, snippets []TerraformSnippet) {
	for i := range snippets {
		if snippets[i].ExamplePath == "" {
			// Try to find examples of this resource type
			results, err := tst.terraformTool.ScanForResources(ctx, []string{snippets[i].ResourceType})
			if err == nil && len(results) > 0 {
				snippets[i].ExamplePath = results[0].FilePath
			}
		}
	}
}

// extractSecurityFeatures extracts security features from a scan result
func (tst *SnippetsTool) extractSecurityFeatures(result models.TerraformScanResult) []string {
	var features []string

	// Check for encryption
	if _, hasEncryption := result.Configuration["encryption"]; hasEncryption {
		features = append(features, "Encryption enabled")
	}
	if _, hasKMS := result.Configuration["kms_key_id"]; hasKMS {
		features = append(features, "KMS encryption")
	}

	// Check for access controls
	if _, hasPolicy := result.Configuration["policy"]; hasPolicy {
		features = append(features, "Access policy defined")
	}

	// Check for logging
	if _, hasLogging := result.Configuration["logging"]; hasLogging {
		features = append(features, "Logging enabled")
	}

	// Check for versioning
	if _, hasVersioning := result.Configuration["versioning"]; hasVersioning {
		features = append(features, "Versioning enabled")
	}

	return features
}

// formatSnippets formats the snippets for output
func (tst *SnippetsTool) formatSnippets(snippets []TerraformSnippet) string {
	if len(snippets) == 0 {
		return "No Terraform snippets found matching the criteria."
	}

	var output strings.Builder
	output.WriteString("# Terraform Configuration Snippets\n\n")

	for i, snippet := range snippets {
		output.WriteString(fmt.Sprintf("## %d. %s\n\n", i+1, snippet.Name))

		if snippet.Description != "" {
			output.WriteString(fmt.Sprintf("**Description:** %s\n\n", snippet.Description))
		}

		if snippet.ResourceType != "" {
			output.WriteString(fmt.Sprintf("**Resource Type:** `%s`\n\n", snippet.ResourceType))
		}

		if len(snippet.ControlCodes) > 0 {
			output.WriteString("**Security Controls:** ")
			for j, code := range snippet.ControlCodes {
				if j > 0 {
					output.WriteString(", ")
				}
				output.WriteString(fmt.Sprintf("`%s`", code))
			}
			output.WriteString("\n\n")
		}

		if len(snippet.SecurityFeatures) > 0 {
			output.WriteString("**Security Features:**\n")
			for _, feature := range snippet.SecurityFeatures {
				output.WriteString(fmt.Sprintf("- %s\n", feature))
			}
			output.WriteString("\n")
		}

		if snippet.ExamplePath != "" {
			output.WriteString(fmt.Sprintf("**Example Found In:** `%s`\n\n", snippet.ExamplePath))
		}

		if snippet.Configuration != "" {
			output.WriteString("**Configuration:**\n\n```hcl\n")
			output.WriteString(snippet.Configuration)
			output.WriteString("\n```\n\n")
		}

		output.WriteString("---\n\n")
	}

	return output.String()
}
