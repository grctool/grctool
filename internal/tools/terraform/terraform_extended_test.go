// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package terraform

import (
	"regexp"
	"strings"
	"testing"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestAnalyzer(t *testing.T) *Analyzer {
	t.Helper()
	log, err := logger.NewTestLogger()
	require.NoError(t, err)
	return &Analyzer{
		config: &config.TerraformToolConfig{
			Enabled:         true,
			ScanPaths:       []string{},
			IncludePatterns: []string{"*.tf"},
			ExcludePatterns: []string{".terraform/**"},
		},
		logger: log,
	}
}

// ---- matchesPatterns tests ----

func TestAnalyzer_MatchesPatterns(t *testing.T) {
	t.Parallel()
	a := newTestAnalyzer(t)

	tests := []struct {
		name     string
		filePath string
		patterns []string
		expected bool
	}{
		{"matches .tf extension", "main.tf", []string{"*.tf"}, true},
		{"matches .tfvars extension", "vars.tfvars", []string{"*.tf", "*.tfvars"}, true},
		{"no match", "readme.md", []string{"*.tf"}, false},
		{"empty patterns", "main.tf", []string{}, false},
		{"full path match", "modules/vpc/main.tf", []string{"modules/*/main.tf"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := a.matchesPatterns(tt.filePath, tt.patterns)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---- isResourceTypeOfInterest tests ----

func TestAnalyzer_IsResourceTypeOfInterest(t *testing.T) {
	t.Parallel()
	a := newTestAnalyzer(t)

	tests := []struct {
		name          string
		resourceType  string
		resourceTypes []string
		expected      bool
	}{
		{"all resources when empty", "aws_iam_role", []string{}, true},
		{"exact match", "aws_iam_role", []string{"aws_iam_role"}, true},
		{"no match", "aws_s3_bucket", []string{"aws_iam_role"}, false},
		{"glob match", "aws_iam_role", []string{"aws_iam_*"}, true},
		{"glob no match", "aws_s3_bucket", []string{"aws_iam_*"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := a.isResourceTypeOfInterest(tt.resourceType, tt.resourceTypes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---- getSecurityRelevance tests ----

func TestAnalyzer_GetSecurityRelevance(t *testing.T) {
	t.Parallel()
	a := newTestAnalyzer(t)

	tests := []struct {
		name         string
		resourceType string
		expectAny    bool
		contains     string
	}{
		{"IAM role", "aws_iam_role", true, "CC6.1"},
		{"security group", "aws_security_group", true, "CC6.6"},
		{"KMS key", "aws_kms_key", true, "CC6.8"},
		{"CloudTrail", "aws_cloudtrail", true, "CC7.2"},
		{"autoscaling group", "aws_autoscaling_group", true, "SO2"},
		{"unknown resource", "custom_unknown_resource", false, ""},
		{"generic IAM resource", "custom_iam_thing", true, "CC6.1"},
		{"generic network resource", "custom_network_thing", true, "CC6.6"},
		{"generic encrypt resource", "custom_encrypt_thing", true, "CC6.8"},
		{"generic log resource", "custom_log_thing", true, "CC7.2"},
		{"generic autoscaling resource", "custom_autoscaling_thing", true, "SO2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := a.getSecurityRelevance(tt.resourceType)
			if tt.expectAny {
				assert.NotEmpty(t, result)
				if tt.contains != "" {
					assert.Contains(t, result, tt.contains)
				}
			} else {
				assert.Empty(t, result)
			}
		})
	}
}

// ---- parseResourceConfiguration tests ----

func TestAnalyzer_ParseResourceConfiguration(t *testing.T) {
	t.Parallel()
	a := newTestAnalyzer(t)
	varPattern := regexp.MustCompile(`^\s*(\w+)\s*=`)

	t.Run("simple key-value", func(t *testing.T) {
		t.Parallel()
		resource := &models.TerraformScanResult{
			Configuration: make(map[string]interface{}),
		}
		a.parseResourceConfiguration(`  enabled = true`, resource, varPattern)
		assert.Equal(t, "true", resource.Configuration["enabled"])
	})

	t.Run("quoted value", func(t *testing.T) {
		t.Parallel()
		resource := &models.TerraformScanResult{
			Configuration: make(map[string]interface{}),
		}
		a.parseResourceConfiguration(`  name = "my-bucket"`, resource, varPattern)
		assert.Equal(t, "my-bucket", resource.Configuration["name"])
	})

	t.Run("security-relevant line", func(t *testing.T) {
		t.Parallel()
		resource := &models.TerraformScanResult{
			Configuration: make(map[string]interface{}),
		}
		a.parseResourceConfiguration(`  encryption = "AES256"`, resource, varPattern)
		assert.Contains(t, resource.Configuration, "security_config")
	})

	t.Run("skip comments", func(t *testing.T) {
		t.Parallel()
		resource := &models.TerraformScanResult{
			Configuration: make(map[string]interface{}),
		}
		a.parseResourceConfiguration(`# this is a comment`, resource, varPattern)
		assert.Empty(t, resource.Configuration)
	})

	t.Run("skip empty lines", func(t *testing.T) {
		t.Parallel()
		resource := &models.TerraformScanResult{
			Configuration: make(map[string]interface{}),
		}
		a.parseResourceConfiguration(``, resource, varPattern)
		assert.Empty(t, resource.Configuration)
	})

	t.Run("value with trailing comment", func(t *testing.T) {
		t.Parallel()
		resource := &models.TerraformScanResult{
			Configuration: make(map[string]interface{}),
		}
		a.parseResourceConfiguration(`  port = 443 # HTTPS port`, resource, varPattern)
		assert.Equal(t, "443", resource.Configuration["port"])
	})

	t.Run("appends to existing security_config", func(t *testing.T) {
		t.Parallel()
		resource := &models.TerraformScanResult{
			Configuration: map[string]interface{}{
				"security_config": "  policy = existing",
			},
		}
		a.parseResourceConfiguration(`  access_control = "strict"`, resource, varPattern)
		secConfig, ok := resource.Configuration["security_config"].(string)
		require.True(t, ok)
		assert.Contains(t, secConfig, "existing")
		assert.Contains(t, secConfig, "access_control")
	})
}

// ---- escapeCSV tests ----

func TestAnalyzer_EscapeCSV(t *testing.T) {
	t.Parallel()
	a := newTestAnalyzer(t)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple value", "hello", "hello"},
		{"value with comma", "a,b", `"a,b"`},
		{"value with newline", "a\nb", `"a` + "\n" + `b"`},
		{"value with quotes", `say "hi"`, `"say ""hi"""`},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := a.escapeCSV(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---- calculateRelevance tests ----

func TestAnalyzer_CalculateRelevance(t *testing.T) {
	t.Parallel()
	a := newTestAnalyzer(t)

	t.Run("empty results", func(t *testing.T) {
		t.Parallel()
		result := a.calculateRelevance(nil)
		assert.Equal(t, 0.0, result)
	})

	t.Run("single result no high value", func(t *testing.T) {
		t.Parallel()
		results := []models.TerraformScanResult{
			{ResourceType: "aws_instance"},
		}
		result := a.calculateRelevance(results)
		assert.Equal(t, 0.5, result)
	})

	t.Run("many results with high value types", func(t *testing.T) {
		t.Parallel()
		var results []models.TerraformScanResult
		for i := 0; i < 12; i++ {
			results = append(results, models.TerraformScanResult{ResourceType: "aws_iam_role"})
		}
		result := a.calculateRelevance(results)
		assert.Equal(t, 1.0, result) // capped at 1.0
	})

	t.Run("5 results with 2 high value", func(t *testing.T) {
		t.Parallel()
		results := []models.TerraformScanResult{
			{ResourceType: "aws_iam_role"},
			{ResourceType: "aws_s3_bucket"},
			{ResourceType: "aws_instance"},
			{ResourceType: "aws_instance"},
			{ResourceType: "aws_instance"},
		}
		result := a.calculateRelevance(results)
		assert.True(t, result >= 0.7) // 0.5 + 0.1 (5 results) + 0.2 (2 high-value)
	})

	t.Run("1 high value type", func(t *testing.T) {
		t.Parallel()
		results := []models.TerraformScanResult{
			{ResourceType: "aws_kms_key"},
		}
		result := a.calculateRelevance(results)
		assert.Equal(t, 0.6, result) // 0.5 + 0.1 (1 high-value)
	})
}

// ---- GenerateEvidenceReport tests ----

func TestAnalyzer_GenerateEvidenceReport_Empty(t *testing.T) {
	t.Parallel()
	a := newTestAnalyzer(t)

	result, err := a.GenerateEvidenceReport(nil, "csv")
	require.NoError(t, err)
	assert.Contains(t, result, "No Terraform resources found")
}

func TestAnalyzer_GenerateEvidenceReport_CSV(t *testing.T) {
	t.Parallel()
	a := newTestAnalyzer(t)

	results := []models.TerraformScanResult{
		{
			ResourceType:      "aws_iam_role",
			ResourceName:      "admin_role",
			FilePath:          "iam/roles.tf",
			LineStart:         10,
			LineEnd:            25,
			SecurityRelevance: []string{"CC6.1", "CC6.3"},
			Configuration: map[string]interface{}{
				"assume_role_policy": "admin-policy",
			},
		},
	}

	result, err := a.GenerateEvidenceReport(results, "csv")
	require.NoError(t, err)
	assert.Contains(t, result, "Resource Type,Resource Name,File Path")
	assert.Contains(t, result, "aws_iam_role")
	assert.Contains(t, result, "admin_role")
	assert.Contains(t, result, "iam/roles.tf")
}

func TestAnalyzer_GenerateEvidenceReport_Markdown(t *testing.T) {
	t.Parallel()
	a := newTestAnalyzer(t)

	results := []models.TerraformScanResult{
		{
			ResourceType:      "aws_security_group",
			ResourceName:      "web_sg",
			FilePath:          "vpc/security.tf",
			LineStart:         1,
			LineEnd:            20,
			SecurityRelevance: []string{"CC6.6"},
			Configuration: map[string]interface{}{
				"ingress": "0.0.0.0/0",
			},
		},
	}

	result, err := a.GenerateEvidenceReport(results, "markdown")
	require.NoError(t, err)
	assert.Contains(t, result, "# Terraform Security Configuration Evidence")
	assert.Contains(t, result, "## aws_security_group")
	assert.Contains(t, result, "### web_sg")
	assert.Contains(t, result, "`CC6.6`")
	assert.Contains(t, result, "**ingress:** `0.0.0.0/0`")
}

func TestAnalyzer_GenerateEvidenceReport_UnsupportedFormat(t *testing.T) {
	t.Parallel()
	a := newTestAnalyzer(t)

	results := []models.TerraformScanResult{{ResourceType: "aws_instance"}}
	_, err := a.GenerateEvidenceReport(results, "xml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format")
}

// ---- ExtractSecurityConfiguration tests ----

func TestAnalyzer_ExtractSecurityConfiguration(t *testing.T) {
	t.Parallel()
	a := newTestAnalyzer(t)

	results := []models.TerraformScanResult{
		{ResourceType: "aws_iam_role", ResourceName: "admin"},
		{ResourceType: "aws_iam_role", ResourceName: "readonly"},
		{ResourceType: "aws_s3_bucket", ResourceName: "logs"},
	}

	secConfig := a.ExtractSecurityConfiguration(results)
	assert.NotNil(t, secConfig)
	assert.Contains(t, secConfig, "_summary")
	summary, ok := secConfig["_summary"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, 3, summary["total_resources"])
	assert.Equal(t, 2, summary["resource_types_count"])
	// Check grouped resources
	assert.Contains(t, secConfig, "aws_iam_role")
	assert.Contains(t, secConfig, "aws_s3_bucket")
}

func TestAnalyzer_ExtractSecurityConfiguration_Empty(t *testing.T) {
	t.Parallel()
	a := newTestAnalyzer(t)

	secConfig := a.ExtractSecurityConfiguration(nil)
	assert.NotNil(t, secConfig)
	summary, ok := secConfig["_summary"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, 0, summary["total_resources"])
}

// ---- BaseAnalysisStrategy tests ----

func TestBaseAnalysisStrategy_Analyze(t *testing.T) {
	t.Parallel()
	bas := &BaseAnalysisStrategy{}

	results := []models.TerraformScanResult{
		{ResourceType: "aws_instance", ResourceName: "web"},
	}

	analyzed, err := bas.Analyze(results)
	require.NoError(t, err)
	assert.Equal(t, results, analyzed)
}

// ---- CSV report with _content field (should be excluded) ----

func TestAnalyzer_GenerateCSVReport_ExcludesContent(t *testing.T) {
	t.Parallel()
	a := newTestAnalyzer(t)

	results := []models.TerraformScanResult{
		{
			ResourceType: "aws_instance",
			ResourceName: "web",
			FilePath:     "main.tf",
			LineStart:    1,
			LineEnd:      10,
			Configuration: map[string]interface{}{
				"_content":      "resource aws_instance web { ... }",
				"instance_type": "t3.micro",
			},
		},
	}

	csv := a.generateCSVReport(results)
	assert.NotContains(t, csv, "_content")
	assert.Contains(t, csv, "instance_type")
}

// ---- Markdown report excludes _content ----

func TestAnalyzer_GenerateMarkdownReport_ExcludesContent(t *testing.T) {
	t.Parallel()
	a := newTestAnalyzer(t)

	results := []models.TerraformScanResult{
		{
			ResourceType: "aws_instance",
			ResourceName: "web",
			FilePath:     "main.tf",
			LineStart:    1,
			LineEnd:      10,
			Configuration: map[string]interface{}{
				"_content":      "resource aws_instance web { ... }",
				"instance_type": "t3.micro",
			},
		},
	}

	md := a.generateMarkdownReport(results)
	lines := strings.Split(md, "\n")
	for _, line := range lines {
		assert.NotContains(t, line, "**_content:**")
	}
	assert.Contains(t, md, "**instance_type:**")
}

// ---- GenerateEvidenceReport sorted output ----

func TestAnalyzer_GenerateEvidenceReport_SortedOutput(t *testing.T) {
	t.Parallel()
	a := newTestAnalyzer(t)

	results := []models.TerraformScanResult{
		{ResourceType: "aws_s3_bucket", ResourceName: "logs", FilePath: "s3.tf", LineStart: 1, LineEnd: 5, Configuration: map[string]interface{}{}},
		{ResourceType: "aws_iam_role", ResourceName: "admin", FilePath: "iam.tf", LineStart: 1, LineEnd: 5, Configuration: map[string]interface{}{}},
		{ResourceType: "aws_iam_role", ResourceName: "readonly", FilePath: "iam.tf", LineStart: 10, LineEnd: 20, Configuration: map[string]interface{}{}},
	}

	csv, err := a.GenerateEvidenceReport(results, "csv")
	require.NoError(t, err)

	lines := strings.Split(csv, "\n")
	// Header is first, then aws_iam_role should come before aws_s3_bucket
	assert.Contains(t, lines[1], "aws_iam_role")
	assert.Contains(t, lines[1], "admin")
}
