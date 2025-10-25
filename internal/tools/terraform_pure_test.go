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
	"math"
	"strings"
	"testing"

	"github.com/grctool/grctool/internal/models"
)

// TestParseTerraformContent tests the ParseTerraformContent pure function
func TestParseTerraformContent(t *testing.T) {
	terraformContent := `
resource "aws_s3_bucket" "example" {
  bucket = "my-test-bucket"
  
  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        sse_algorithm = "AES256"
      }
    }
  }
}

resource "aws_iam_role" "test_role" {
  name = "test-role"
  
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
      }
    ]
  })
}
`

	reader := strings.NewReader(terraformContent)
	results, err := ParseTerraformContent(reader, "test.tf", []string{})

	if err != nil {
		t.Fatalf("ParseTerraformContent failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 resources, got %d", len(results))
	}

	// Check first resource (S3 bucket)
	s3Resource := results[0]
	if s3Resource.ResourceType != "aws_s3_bucket" {
		t.Errorf("Expected resource type 'aws_s3_bucket', got '%s'", s3Resource.ResourceType)
	}

	if s3Resource.ResourceName != "example" {
		t.Errorf("Expected resource name 'example', got '%s'", s3Resource.ResourceName)
	}

	if s3Resource.FilePath != "test.tf" {
		t.Errorf("Expected file path 'test.tf', got '%s'", s3Resource.FilePath)
	}

	if s3Resource.LineStart != 2 {
		t.Errorf("Expected line start 2, got %d", s3Resource.LineStart)
	}

	// Check configuration
	if bucket, ok := s3Resource.Configuration["bucket"]; !ok || bucket != "my-test-bucket" {
		t.Errorf("Expected bucket configuration 'my-test-bucket', got %v", bucket)
	}

	// Check security relevance
	expectedRelevance := []string{"CC6.8", "CC7.2"}
	if len(s3Resource.SecurityRelevance) != len(expectedRelevance) {
		t.Errorf("Expected %d security relevance items, got %d", len(expectedRelevance), len(s3Resource.SecurityRelevance))
	}

	// Check second resource (IAM role)
	iamResource := results[1]
	if iamResource.ResourceType != "aws_iam_role" {
		t.Errorf("Expected resource type 'aws_iam_role', got '%s'", iamResource.ResourceType)
	}

	if iamResource.ResourceName != "test_role" {
		t.Errorf("Expected resource name 'test_role', got '%s'", iamResource.ResourceName)
	}
}

// TestParseTerraformHCLBlocks tests the ParseTerraformHCLBlocks pure function
func TestParseTerraformHCLBlocks(t *testing.T) {
	terraformContent := `
module "vpc" {
  source = "./modules/vpc"
  
  cidr_block = "10.0.0.0/16"
  environment = "production"
}

data "aws_availability_zones" "available" {
  state = "available"
}

locals {
  common_tags = {
    Environment = "production"
    Project     = "test-project"
  }
}
`

	tests := []struct {
		blockType    string
		expectedName string
		expectedType string
	}{
		{"module", "vpc", "module"},
		{"data", "available", "aws_availability_zones"},
		{"locals", "local_values", "locals"},
	}

	for _, tt := range tests {
		t.Run(tt.blockType, func(t *testing.T) {
			reader := strings.NewReader(terraformContent)
			results, err := ParseTerraformHCLBlocks(reader, "test.tf", tt.blockType)

			if err != nil {
				t.Fatalf("ParseTerraformHCLBlocks failed: %v", err)
			}

			if len(results) != 1 {
				t.Errorf("Expected 1 %s block, got %d", tt.blockType, len(results))
			}

			result := results[0]
			if result.ResourceType != tt.expectedType {
				t.Errorf("Expected resource type '%s', got '%s'", tt.expectedType, result.ResourceType)
			}

			if result.ResourceName != tt.expectedName {
				t.Errorf("Expected resource name '%s', got '%s'", tt.expectedName, result.ResourceName)
			}

			if result.FilePath != "test.tf" {
				t.Errorf("Expected file path 'test.tf', got '%s'", result.FilePath)
			}
		})
	}
}

// TestIsResourceTypeOfInterest tests the IsResourceTypeOfInterest pure function
func TestIsResourceTypeOfInterest(t *testing.T) {
	tests := []struct {
		name          string
		resourceType  string
		resourceTypes []string
		expected      bool
	}{
		{
			name:          "empty filter returns all",
			resourceType:  "aws_s3_bucket",
			resourceTypes: []string{},
			expected:      true,
		},
		{
			name:          "exact match",
			resourceType:  "aws_s3_bucket",
			resourceTypes: []string{"aws_s3_bucket", "aws_iam_role"},
			expected:      true,
		},
		{
			name:          "no match",
			resourceType:  "aws_ec2_instance",
			resourceTypes: []string{"aws_s3_bucket", "aws_iam_role"},
			expected:      false,
		},
		{
			name:          "wildcard match",
			resourceType:  "aws_s3_bucket",
			resourceTypes: []string{"aws_s3_*"},
			expected:      true,
		},
		{
			name:          "wildcard no match",
			resourceType:  "aws_iam_role",
			resourceTypes: []string{"aws_s3_*"},
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsResourceTypeOfInterest(tt.resourceType, tt.resourceTypes)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestGetSecurityRelevance tests the GetSecurityRelevance pure function
func TestGetSecurityRelevance(t *testing.T) {
	tests := []struct {
		resourceType string
		expected     []string
	}{
		{
			resourceType: "aws_iam_role",
			expected:     []string{"CC6.1", "CC6.3"},
		},
		{
			resourceType: "aws_s3_bucket",
			expected:     []string{"CC6.8", "CC7.2"},
		},
		{
			resourceType: "aws_security_group",
			expected:     []string{"CC6.6", "CC7.1"},
		},
		{
			resourceType: "aws_kms_key",
			expected:     []string{"CC6.8"},
		},
		{
			resourceType: "aws_autoscaling_group",
			expected:     []string{"SO2"},
		},
		{
			resourceType: "unknown_resource_type",
			expected:     []string{},
		},
		{
			resourceType: "custom_iam_thing", // Generic pattern matching
			expected:     []string{"CC6.1", "CC6.3"},
		},
		{
			resourceType: "custom_encrypt_storage", // Generic pattern matching
			expected:     []string{"CC6.8"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			result := GetSecurityRelevance(tt.resourceType)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d relevance items, got %d", len(tt.expected), len(result))
			}

			for _, expectedItem := range tt.expected {
				found := false
				for _, actualItem := range result {
					if actualItem == expectedItem {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected relevance item '%s' not found in %v", expectedItem, result)
				}
			}
		})
	}
}

// TestAnalyzeEncryptionSettings tests the AnalyzeEncryptionSettings pure function
func TestAnalyzeEncryptionSettings(t *testing.T) {
	resources := []models.TerraformScanResult{
		{
			ResourceType: "aws_s3_bucket",
			ResourceName: "encrypted_bucket",
			Configuration: map[string]interface{}{
				"kms_key_id": "alias/my-key",
			},
		},
		{
			ResourceType: "aws_s3_bucket",
			ResourceName: "unencrypted_bucket",
			Configuration: map[string]interface{}{
				"bucket": "my-bucket",
			},
		},
		{
			ResourceType: "aws_kms_key",
			ResourceName: "encryption_key",
			Configuration: map[string]interface{}{
				"description": "My encryption key",
			},
		},
		{
			ResourceType: "aws_ec2_instance",
			ResourceName: "web_server",
			Configuration: map[string]interface{}{
				"instance_type": "t3.micro",
			},
		},
	}

	analysis := AnalyzeEncryptionSettings(resources)

	// Test encrypted resources
	expectedEncrypted := []string{"aws_s3_bucket.encrypted_bucket"}
	if len(analysis.EncryptedResources) != len(expectedEncrypted) {
		t.Errorf("Expected %d encrypted resources, got %d", len(expectedEncrypted), len(analysis.EncryptedResources))
	}

	for _, expected := range expectedEncrypted {
		found := false
		for _, actual := range analysis.EncryptedResources {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected encrypted resource '%s' not found", expected)
		}
	}

	// Test unencrypted resources (only those that require encryption)
	expectedUnencrypted := []string{"aws_s3_bucket.unencrypted_bucket"}
	if len(analysis.UnencryptedResources) != len(expectedUnencrypted) {
		t.Errorf("Expected %d unencrypted resources, got %d", len(expectedUnencrypted), len(analysis.UnencryptedResources))
	}

	// Test total resources
	if analysis.TotalResources != 4 {
		t.Errorf("Expected 4 total resources, got %d", analysis.TotalResources)
	}

	// Test encryption methods
	if len(analysis.EncryptionMethods["KMS"]) != 1 {
		t.Errorf("Expected 1 KMS encrypted resource, got %d", len(analysis.EncryptionMethods["KMS"]))
	}
}

// TestAnalyzeIAMConfiguration tests the AnalyzeIAMConfiguration pure function
func TestAnalyzeIAMConfiguration(t *testing.T) {
	resources := []models.TerraformScanResult{
		{
			ResourceType: "aws_iam_role",
			ResourceName: "safe_role",
			Configuration: map[string]interface{}{
				"name": "safe-role",
			},
		},
		{
			ResourceType: "aws_iam_policy",
			ResourceName: "overly_permissive",
			Configuration: map[string]interface{}{
				"name":     "bad-policy",
				"action":   "*",
				"resource": "*",
			},
		},
		{
			ResourceType: "aws_iam_role",
			ResourceName: "permissive_role",
			Configuration: map[string]interface{}{
				"principal": "*",
			},
		},
		{
			ResourceType: "aws_s3_bucket",
			ResourceName: "bucket",
			Configuration: map[string]interface{}{
				"bucket": "my-bucket",
			},
		},
	}

	analysis := AnalyzeIAMConfiguration(resources)

	// Test IAM resources
	expectedRoles := []string{"aws_iam_role.safe_role", "aws_iam_role.permissive_role"}
	if len(analysis.IAMRoles) != len(expectedRoles) {
		t.Errorf("Expected %d IAM roles, got %d", len(expectedRoles), len(analysis.IAMRoles))
	}

	expectedPolicies := []string{"aws_iam_policy.overly_permissive"}
	if len(analysis.IAMPolicies) != len(expectedPolicies) {
		t.Errorf("Expected %d IAM policies, got %d", len(expectedPolicies), len(analysis.IAMPolicies))
	}

	// Test overly permissive resources
	expectedPermissive := []string{"aws_iam_policy.overly_permissive", "aws_iam_role.permissive_role"}
	if len(analysis.OverlyPermissive) != len(expectedPermissive) {
		t.Errorf("Expected %d overly permissive resources, got %d", len(expectedPermissive), len(analysis.OverlyPermissive))
	}

	// Test total IAM resources
	if analysis.TotalIAMResources != 3 {
		t.Errorf("Expected 3 total IAM resources, got %d", analysis.TotalIAMResources)
	}
}

// TestAnalyzeNetworkSecurity tests the AnalyzeNetworkSecurity pure function
func TestAnalyzeNetworkSecurity(t *testing.T) {
	resources := []models.TerraformScanResult{
		{
			ResourceType: "aws_security_group",
			ResourceName: "open_sg",
			Configuration: map[string]interface{}{
				"cidr_blocks": "0.0.0.0/0",
			},
		},
		{
			ResourceType: "aws_security_group",
			ResourceName: "restricted_sg",
			Configuration: map[string]interface{}{
				"cidr_blocks": "10.0.0.0/16",
			},
		},
		{
			ResourceType: "aws_security_group",
			ResourceName: "no_rules_sg",
			Configuration: map[string]interface{}{
				"name": "no-rules",
			},
		},
	}

	analysis := AnalyzeNetworkSecurity(resources)

	// Test security groups
	if len(analysis.SecurityGroups) != 3 {
		t.Errorf("Expected 3 security groups, got %d", len(analysis.SecurityGroups))
	}

	// Test open to internet
	expectedOpen := []string{"aws_security_group.open_sg"}
	if len(analysis.OpenToInternet) != len(expectedOpen) {
		t.Errorf("Expected %d open resources, got %d", len(expectedOpen), len(analysis.OpenToInternet))
	}

	// Test restricted access
	expectedRestricted := []string{"aws_security_group.restricted_sg"}
	if len(analysis.RestrictedAccess) != len(expectedRestricted) {
		t.Errorf("Expected %d restricted resources, got %d", len(expectedRestricted), len(analysis.RestrictedAccess))
	}

	// Test total network resources
	if analysis.TotalNetworkResources != 3 {
		t.Errorf("Expected 3 total network resources, got %d", analysis.TotalNetworkResources)
	}
}

// TestCalculateComplianceScore tests the CalculateComplianceScore pure function
func TestCalculateComplianceScore(t *testing.T) {
	encryptionAnalysis := EncryptionAnalysis{EncryptionScore: 0.8}
	iamAnalysis := IAMAnalysis{SecurityScore: 0.6}
	networkAnalysis := NetworkSecurityAnalysis{SecurityScore: 0.9}

	score := CalculateComplianceScore(encryptionAnalysis, iamAnalysis, networkAnalysis)

	expectedOverall := (0.8 + 0.6 + 0.9) / 3.0
	if math.Abs(score.OverallScore-expectedOverall) > 0.001 {
		t.Errorf("Expected overall score %.3f, got %.3f", expectedOverall, score.OverallScore)
	}

	if score.Categories["encryption"] != 0.8 {
		t.Errorf("Expected encryption score 0.8, got %.3f", score.Categories["encryption"])
	}

	if score.Categories["iam"] != 0.6 {
		t.Errorf("Expected IAM score 0.6, got %.3f", score.Categories["iam"])
	}

	if score.Categories["network"] != 0.9 {
		t.Errorf("Expected network score 0.9, got %.3f", score.Categories["network"])
	}
}

// TestIdentifySecurityGaps tests the IdentifySecurityGaps pure function
func TestIdentifySecurityGaps(t *testing.T) {
	currentState := SecurityState{
		UnencryptedResources:      []string{"aws_s3_bucket.unencrypted"},
		OverlyPermissiveResources: []string{"aws_iam_policy.permissive"},
		OpenNetworkResources:      []string{"aws_security_group.open"},
	}

	requirements := SecurityRequirements{
		RequireEncryption: true,
	}

	gaps := IdentifySecurityGaps(currentState, requirements)

	expectedGaps := 3 // encryption + iam + network
	if len(gaps) != expectedGaps {
		t.Errorf("Expected %d security gaps, got %d", expectedGaps, len(gaps))
	}

	// Test gap types
	gapTypes := make(map[string]int)
	for _, gap := range gaps {
		gapTypes[gap.Type]++
	}

	if gapTypes["encryption"] != 1 {
		t.Errorf("Expected 1 encryption gap, got %d", gapTypes["encryption"])
	}

	if gapTypes["iam"] != 1 {
		t.Errorf("Expected 1 IAM gap, got %d", gapTypes["iam"])
	}

	if gapTypes["network"] != 1 {
		t.Errorf("Expected 1 network gap, got %d", gapTypes["network"])
	}
}

// TestCalculateTerraformRelevance tests the CalculateTerraformRelevance pure function
func TestCalculateTerraformRelevance(t *testing.T) {
	tests := []struct {
		name     string
		results  []models.TerraformScanResult
		expected float64
	}{
		{
			name:     "no results",
			results:  []models.TerraformScanResult{},
			expected: 0.0,
		},
		{
			name: "few non-critical resources",
			results: []models.TerraformScanResult{
				{ResourceType: "aws_ec2_instance"},
				{ResourceType: "aws_vpc"},
			},
			expected: 0.5, // Base score only
		},
		{
			name: "many high-value resources",
			results: func() []models.TerraformScanResult {
				var results []models.TerraformScanResult
				// Add 15 high-value resources
				highValueTypes := []string{
					"aws_iam_role", "aws_iam_policy", "aws_s3_bucket",
					"aws_security_group", "aws_kms_key", "aws_cloudtrail",
					"aws_config_configuration_recorder",
				}
				for i := 0; i < 15; i++ {
					results = append(results, models.TerraformScanResult{
						ResourceType: highValueTypes[i%len(highValueTypes)],
					})
				}
				return results
			}(),
			expected: 1.0, // All bonuses applied, capped at 1.0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			relevance := CalculateTerraformRelevance(tt.results)
			if relevance != tt.expected {
				t.Errorf("Expected relevance %.1f, got %.1f", tt.expected, relevance)
			}
		})
	}
}

// TestRequiresEncryption tests the RequiresEncryption pure function
func TestRequiresEncryption(t *testing.T) {
	tests := []struct {
		resourceType string
		expected     bool
	}{
		{"aws_s3_bucket", true},
		{"aws_rds_cluster", true},
		{"aws_ebs_volume", true},
		{"aws_ec2_instance", false},
		{"aws_vpc", false},
		{"unknown_type", false},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			result := RequiresEncryption(tt.resourceType)
			if result != tt.expected {
				t.Errorf("Expected RequiresEncryption('%s') = %v, got %v", tt.resourceType, tt.expected, result)
			}
		})
	}
}

// TestFilterResultsBySecurityControls tests the FilterResultsBySecurityControls pure function
func TestFilterResultsBySecurityControls(t *testing.T) {
	results := []models.TerraformScanResult{
		{
			ResourceType:      "aws_s3_bucket",
			ResourceName:      "bucket1",
			SecurityRelevance: []string{"CC6.8"},
		},
		{
			ResourceType:      "aws_iam_role",
			ResourceName:      "role1",
			SecurityRelevance: []string{"CC6.1", "CC6.3"},
		},
		{
			ResourceType:      "aws_ec2_instance",
			ResourceName:      "instance1",
			SecurityRelevance: []string{},
		},
		{
			ResourceType:      "aws_kms_key",
			ResourceName:      "key1",
			SecurityRelevance: []string{"CC6.8"},
		},
	}

	tests := map[string]struct {
		controls []string
		expected []string
	}{
		"no filters returns all": {
			controls: nil,
			expected: []string{"bucket1", "role1", "instance1", "key1"},
		},
		"control code filter": {
			controls: []string{"CC6.8"},
			expected: []string{"bucket1", "key1"},
		},
		"keyword filter": {
			controls: []string{"encryption"},
			expected: []string{"bucket1", "key1"},
		},
		"multiple filters": {
			controls: []string{"CC6.1", "iam"},
			expected: []string{"role1"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			filtered := FilterResultsBySecurityControls(results, tt.controls)

			if len(filtered) != len(tt.expected) {
				t.Fatalf("expected %d results, got %d", len(tt.expected), len(filtered))
			}

			gotNames := make(map[string]struct{}, len(filtered))
			for _, result := range filtered {
				gotNames[result.ResourceName] = struct{}{}
			}

			for _, expectedName := range tt.expected {
				if _, ok := gotNames[expectedName]; !ok {
					t.Errorf("expected result for resource %s", expectedName)
				}
			}
		})
	}
}

func TestGenerateTerraformSecurityReportCSV(t *testing.T) {
	analysis := EncryptionAnalysis{
		EncryptedResources:   []string{"aws_s3_bucket.app_logs"},
		UnencryptedResources: []string{"aws_s3_bucket.audit"},
		EncryptionMethods: map[string][]string{
			"AES256": {"aws_s3_bucket.app_logs"},
		},
		TotalResources:  2,
		EncryptionScore: 0.5,
	}

	report, err := GenerateTerraformSecurityReport(analysis, "csv")
	if err != nil {
		t.Fatalf("unexpected error generating csv report: %v", err)
	}

	if strings.Contains(report, "not implemented") {
		t.Fatal("expected concrete csv report, found placeholder content")
	}

	if !strings.Contains(report, "encryption,security_score,0.50") {
		t.Errorf("expected csv to include security score, got %q", report)
	}

	if !strings.Contains(report, "encryption,unencrypted_resources,aws_s3_bucket.audit") {
		t.Errorf("expected csv to list unencrypted resources, got %q", report)
	}
}

func TestGenerateTerraformSecurityReportMarkdown(t *testing.T) {
	analysis := NetworkSecurityAnalysis{
		SecurityGroups:        []string{"aws_security_group.public"},
		OpenToInternet:        []string{"aws_security_group.public"},
		RestrictedAccess:      []string{"aws_security_group.private"},
		TotalNetworkResources: 2,
		SecurityScore:         0.5,
	}

	report, err := GenerateTerraformSecurityReport(analysis, "markdown")
	if err != nil {
		t.Fatalf("unexpected error generating markdown report: %v", err)
	}

	if strings.Contains(report, "not implemented") {
		t.Fatal("expected markdown renderer to produce report content, got placeholder")
	}

	if !strings.Contains(report, "## Network Security") {
		t.Errorf("expected markdown to contain network section, got %q", report)
	}

	if !strings.Contains(report, "aws_security_group.public") {
		t.Errorf("expected markdown to list open security group, got %q", report)
	}
}
