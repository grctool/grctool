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
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestIndexer creates a minimal indexer for testing
func createTestIndexer(t *testing.T) *SecurityAttributeIndexer {
	log, _ := logger.NewTestLogger()
	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				Terraform: config.TerraformToolConfig{
					ScanPaths: []string{"/test"},
				},
			},
		},
		Storage: config.StorageConfig{
			CacheDir: t.TempDir(),
		},
	}
	return NewSecurityAttributeIndexer(cfg, log)
}

// TestCalculateIndexStats tests index statistics calculation
func TestCalculateIndexStats(t *testing.T) {
	tests := []struct {
		name  string
		index *SecurityIndex
		want  *IndexStatistics
	}{
		{
			name: "calculate stats for simple index",
			index: &SecurityIndex{
				ComplianceCoverage: 0.85,
				SecurityAttributes: map[string]SecurityAttributeDetails{
					"encryption": {
						AttributeName: "encryption",
						ResourceCount: 5,
					},
					"access_control": {
						AttributeName: "access_control",
						ResourceCount: 3,
					},
				},
				EnvironmentStats: map[string]EnvironmentStats{
					"production": {
						ResourceCount: 10,
					},
					"staging": {
						ResourceCount: 5,
					},
				},
				ControlMapping: map[string][]IndexedResource{
					"CC-06.1": {
						{ComplianceStatus: "compliant"},
						{ComplianceStatus: "compliant"},
						{ComplianceStatus: "non_compliant"},
					},
					"CC-07.1": {
						{ComplianceStatus: "compliant"},
					},
				},
			},
			want: &IndexStatistics{
				ComplianceCoverage: 0.85,
				AttributeDistribution: map[string]int{
					"encryption":     5,
					"access_control": 3,
				},
				EnvironmentDistribution: map[string]int{
					"production": 10,
					"staging":    5,
				},
				ControlCoverage: map[string]ControlCoverageStats{
					"CC-06.1": {
						TotalResources:     3,
						CompliantResources: 2,
						ComplianceRate:     0.6666666666666666,
					},
					"CC-07.1": {
						TotalResources:     1,
						CompliantResources: 1,
						ComplianceRate:     1.0,
					},
				},
			},
		},
		{
			name: "empty index returns zero stats",
			index: &SecurityIndex{
				ComplianceCoverage: 0.0,
				SecurityAttributes: map[string]SecurityAttributeDetails{},
				EnvironmentStats:   map[string]EnvironmentStats{},
				ControlMapping:     map[string][]IndexedResource{},
			},
			want: &IndexStatistics{
				ComplianceCoverage:      0.0,
				AttributeDistribution:   map[string]int{},
				EnvironmentDistribution: map[string]int{},
				ControlCoverage:         map[string]ControlCoverageStats{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indexer := createTestIndexer(t)
			stats := indexer.calculateIndexStats(tt.index)

			assert.Equal(t, tt.want.ComplianceCoverage, stats.ComplianceCoverage)
			assert.Equal(t, tt.want.AttributeDistribution, stats.AttributeDistribution)
			assert.Equal(t, tt.want.EnvironmentDistribution, stats.EnvironmentDistribution)

			// Check control coverage
			for control, expected := range tt.want.ControlCoverage {
				actual, ok := stats.ControlCoverage[control]
				require.True(t, ok, "control %s should exist in stats", control)
				assert.Equal(t, expected.TotalResources, actual.TotalResources)
				assert.Equal(t, expected.CompliantResources, actual.CompliantResources)
				assert.InDelta(t, expected.ComplianceRate, actual.ComplianceRate, 0.0001)
			}
		})
	}
}

// TestExtractSecurityAttributes tests security attribute extraction
func TestExtractSecurityAttributes(t *testing.T) {
	tests := []struct {
		name         string
		resource     models.TerraformScanResult
		wantContains []string
		wantNotEmpty bool
	}{
		{
			name: "KMS resource has encryption attributes",
			resource: models.TerraformScanResult{
				ResourceType: "aws_kms_key",
			},
			wantContains: []string{"encryption"},
			wantNotEmpty: true,
		},
		{
			name: "CloudTrail has monitoring attributes",
			resource: models.TerraformScanResult{
				ResourceType: "aws_cloudtrail",
			},
			wantContains: []string{"monitoring"},
			wantNotEmpty: true,
		},
		{
			name: "IAM resource has access_control attributes",
			resource: models.TerraformScanResult{
				ResourceType: "aws_iam_role",
			},
			wantContains: []string{"access_control"},
			wantNotEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indexer := createTestIndexer(t)
			attrs := indexer.extractSecurityAttributes(tt.resource)

			if tt.wantNotEmpty {
				assert.NotEmpty(t, attrs, "should extract security attributes")
			}
			for _, want := range tt.wantContains {
				assert.Contains(t, attrs, want, "should contain attribute %s", want)
			}
		})
	}
}

// TestCalculateResourceRiskLevel tests risk level calculation returns valid levels
func TestCalculateResourceRiskLevel(t *testing.T) {
	indexer := createTestIndexer(t)

	resource := models.TerraformScanResult{
		ResourceType: "aws_s3_bucket",
		Configuration: map[string]interface{}{
			"acl": "private",
		},
	}

	riskLevel := indexer.calculateResourceRiskLevel(resource)

	// Should return a valid risk level
	assert.Contains(t, []string{"low", "medium", "high"}, riskLevel, "should return valid risk level")
}

// TestCalculateComplianceStatus tests compliance status calculation returns valid status
func TestCalculateComplianceStatus(t *testing.T) {
	indexer := createTestIndexer(t)

	resource := models.TerraformScanResult{
		ResourceType: "aws_s3_bucket",
		Configuration: map[string]interface{}{
			"bucket": "test",
		},
	}

	status := indexer.calculateComplianceStatus(resource)

	// Should return a valid compliance status
	assert.Contains(t, []string{"compliant", "non_compliant", "partial"}, status, "should return valid compliance status")
}

// TestExtractEnvironmentFromPath tests environment extraction from file paths
func TestExtractEnvironmentFromPath(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		want     string
	}{
		{
			name:     "production environment",
			filePath: "/terraform/production/main.tf",
			want:     "prod",
		},
		{
			name:     "staging environment",
			filePath: "/terraform/staging/vpc.tf",
			want:     "staging",
		},
		{
			name:     "development environment",
			filePath: "/infra/dev/resources.tf",
			want:     "dev",
		},
		{
			name:     "prod short form",
			filePath: "/terraform/prod/main.tf",
			want:     "prod",
		},
		{
			name:     "unknown environment",
			filePath: "/terraform/main.tf",
			want:     "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indexer := createTestIndexer(t)
			env := indexer.extractEnvironmentFromPath(tt.filePath)
			assert.Equal(t, tt.want, env)
		})
	}
}

// TestIndexResource tests resource indexing produces valid output
func TestIndexResource(t *testing.T) {
	indexer := createTestIndexer(t)
	query := SecurityIndexQuery{
		IncludeMetadata: true,
	}

	resource := models.TerraformScanResult{
		ResourceType: "aws_s3_bucket",
		ResourceName: "test-bucket",
		FilePath:     "/terraform/production/s3.tf",
		Configuration: map[string]interface{}{
			"bucket": "my-test-bucket",
		},
		SecurityRelevance: []string{"CC-06.1", "CC-06.6"},
	}

	indexed := indexer.indexResource(resource, query)

	// Verify basic fields are populated
	assert.Equal(t, "aws_s3_bucket", indexed.ResourceType)
	assert.NotEmpty(t, indexed.ResourceID, "ResourceID should be populated")
	assert.Equal(t, "/terraform/production/s3.tf", indexed.FilePath)
	assert.NotEmpty(t, indexed.Environment, "Environment should be extracted from path")
	assert.ElementsMatch(t, []string{"CC-06.1", "CC-06.6"}, indexed.ControlRelevance)
	assert.Contains(t, []string{"low", "medium", "high"}, indexed.RiskLevel, "RiskLevel should be valid")
	assert.Contains(t, []string{"compliant", "non_compliant", "partial"}, indexed.ComplianceStatus, "ComplianceStatus should be valid")
	assert.NotNil(t, indexed.Configuration)
}

// TestUpdateIndexMappings tests index mapping updates
func TestUpdateIndexMappings(t *testing.T) {
	indexer := createTestIndexer(t)

	index := &SecurityIndex{
		IndexedResources:   []IndexedResource{},
		SecurityAttributes: make(map[string]SecurityAttributeDetails),
		ControlMapping:     make(map[string][]IndexedResource),
		FrameworkMapping:   make(map[string][]IndexedResource),
		RiskDistribution:   make(map[string]int),
		EnvironmentStats:   make(map[string]EnvironmentStats),
	}

	resource := IndexedResource{
		ResourceType:       "aws_s3_bucket",
		ResourceID:         "test-bucket",
		FilePath:           "/test/main.tf",
		Environment:        "production",
		SecurityAttributes: []string{"encryption", "versioning"},
		ControlRelevance:   []string{"CC-06.1", "CC-06.6"},
		RiskLevel:          "low",
		LastModified:       time.Now(),
	}

	indexer.updateIndexMappings(index, resource)

	// Check control mapping
	assert.Contains(t, index.ControlMapping, "CC-06.1")
	assert.Contains(t, index.ControlMapping, "CC-06.6")
	assert.Len(t, index.ControlMapping["CC-06.1"], 1)

	// Check security attributes
	assert.Contains(t, index.SecurityAttributes, "encryption")
	assert.Contains(t, index.SecurityAttributes, "versioning")
	assert.Equal(t, 1, index.SecurityAttributes["encryption"].ResourceCount)

	// Check risk distribution
	assert.Equal(t, 1, index.RiskDistribution["low"])

	// Check environment stats
	assert.Contains(t, index.EnvironmentStats, "production")
	assert.Equal(t, 1, index.EnvironmentStats["production"].ResourceCount)
}

// TestIsSecurityRelevantConfig tests security relevance detection
func TestIsSecurityRelevantConfig(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value interface{}
		want  bool
	}{
		{
			name:  "encryption key is security relevant",
			key:   "encryption",
			value: true,
			want:  true,
		},
		{
			name:  "kms_key is security relevant",
			key:   "kms_key_id",
			value: "arn:aws:kms:...",
			want:  true,
		},
		{
			name:  "iam_role is security relevant",
			key:   "iam_role_arn",
			value: "arn:aws:iam:...",
			want:  true,
		},
		{
			name:  "bucket name is not security relevant",
			key:   "bucket",
			value: "my-bucket",
			want:  false,
		},
		{
			name:  "tags are not security relevant",
			key:   "tags",
			value: map[string]interface{}{"Name": "test"},
			want:  false,
		},
		{
			name:  "logging is security relevant",
			key:   "logging",
			value: map[string]interface{}{"enabled": true},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indexer := createTestIndexer(t)
			result := indexer.isSecurityRelevantConfig(tt.key, tt.value)
			assert.Equal(t, tt.want, result)
		})
	}
}
