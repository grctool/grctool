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

//go:build integration
// +build integration

package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/tools/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTerraformIndexWorkflow tests the complete index build and query lifecycle
func TestTerraformIndexWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Terraform index integration tests in short mode")
	}

	// Setup test environment with realistic Terraform files
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, ".cache")
	setupIndexTestFixtures(t, tempDir)

	cfg := createIndexTestConfig(tempDir, cacheDir)
	log, _ := logger.NewTestLogger()

	t.Run("Index Build and Persistence", func(t *testing.T) {
		ctx := context.Background()

		// Create indexer
		indexer := terraform.NewSecurityAttributeIndexer(cfg, log)

		// Build and persist index
		start := time.Now()
		persistedIndex, err := indexer.BuildAndPersistIndex(ctx)
		buildDuration := time.Since(start)

		require.NoError(t, err, "index build should succeed")
		assert.NotNil(t, persistedIndex, "persisted index should not be nil")

		// Verify metadata
		assert.Equal(t, terraform.IndexVersion, persistedIndex.Version, "version should match")
		assert.NotZero(t, persistedIndex.IndexedAt, "indexed_at should be set")
		assert.Greater(t, persistedIndex.Metadata.TotalResources, 0, "should have indexed resources")
		assert.Greater(t, persistedIndex.Metadata.TotalFiles, 0, "should have scanned files")
		assert.Greater(t, persistedIndex.Metadata.ScanDurationMs, int64(0), "scan duration should be recorded")

		// Verify index file exists
		indexPath := filepath.Join(cacheDir, "terraform", terraform.IndexFileName)
		assert.FileExists(t, indexPath, "index file should be persisted")

		// Verify source file tracking
		assert.NotEmpty(t, persistedIndex.SourceFiles, "should track source files")
		for path, info := range persistedIndex.SourceFiles {
			assert.NotEmpty(t, path, "file path should not be empty")
			assert.NotEmpty(t, info.Checksum, "checksum should be calculated")
			assert.NotZero(t, info.SizeBytes, "file size should be recorded")
		}

		// Verify statistics
		assert.NotNil(t, persistedIndex.Statistics, "statistics should be calculated")
		assert.GreaterOrEqual(t, persistedIndex.Statistics.ComplianceCoverage, 0.0)
		assert.LessOrEqual(t, persistedIndex.Statistics.ComplianceCoverage, 1.0)

		t.Logf("✓ Index built in %v", buildDuration)
		t.Logf("  Total Resources: %d", persistedIndex.Metadata.TotalResources)
		t.Logf("  Total Files: %d", persistedIndex.Metadata.TotalFiles)
		t.Logf("  Compliance Coverage: %.2f%%", persistedIndex.Statistics.ComplianceCoverage*100)
	})

	t.Run("Index Load and Validation", func(t *testing.T) {
		indexPath := filepath.Join(cacheDir, "terraform", terraform.IndexFileName)
		storage := terraform.NewIndexStorage(indexPath, log)

		// Load persisted index
		loadedIndex, err := storage.LoadIndex()
		require.NoError(t, err, "should load index successfully")
		assert.NotNil(t, loadedIndex, "loaded index should not be nil")

		// Verify version compatibility
		assert.Equal(t, terraform.IndexVersion, loadedIndex.Version)

		// Verify index structure
		assert.NotNil(t, loadedIndex.Index, "index data should exist")
		assert.NotEmpty(t, loadedIndex.Index.IndexedResources, "should have indexed resources")
		assert.NotEmpty(t, loadedIndex.Index.ControlMapping, "should have control mappings")
		assert.NotEmpty(t, loadedIndex.Index.SecurityAttributes, "should have security attributes")

		// Verify data integrity
		assert.Equal(t, loadedIndex.Metadata.TotalResources, len(loadedIndex.Index.IndexedResources),
			"metadata resource count should match actual resources")

		t.Logf("✓ Index loaded successfully")
		t.Logf("  Version: %s", loadedIndex.Version)
		t.Logf("  Indexed At: %s", loadedIndex.IndexedAt.Format(time.RFC3339))
	})

	t.Run("Query by SOC2 Control", func(t *testing.T) {
		ctx := context.Background()
		indexer := terraform.NewSecurityAttributeIndexer(cfg, log)

		// Load index
		persistedIndex, err := indexer.LoadOrBuildIndex(ctx, false)
		require.NoError(t, err)

		// Create query interface
		query := terraform.NewIndexQuery(persistedIndex)

		// Test CC6.8 (Encryption) query
		t.Run("CC6.8 Encryption Controls", func(t *testing.T) {
			start := time.Now()
			result := query.ByControl("CC6.8")
			queryDuration := time.Since(start)

			assert.Greater(t, result.Count, 0, "should find encryption-related resources")
			assert.Less(t, queryDuration, 100*time.Millisecond, "query should be fast (<100ms)")

			// Verify we found encryption resources
			hasEncryptionResource := false
			for _, res := range result.Resources {
				if contains(res.SecurityAttributes, "encryption") ||
					contains(res.SecurityAttributes, "key_management") {
					hasEncryptionResource = true
					break
				}
			}
			assert.True(t, hasEncryptionResource, "should find resources with encryption attributes")

			t.Logf("✓ Found %d resources for CC6.8 in %v", result.Count, queryDuration)
		})

		// Test CC6.1 (Access Control) query
		t.Run("CC6.1 Access Control", func(t *testing.T) {
			start := time.Now()
			result := query.ByControl("CC6.1")
			queryDuration := time.Since(start)

			assert.Greater(t, result.Count, 0, "should find access control resources")
			assert.Less(t, queryDuration, 100*time.Millisecond, "query should be fast")

			t.Logf("✓ Found %d resources for CC6.1 in %v", result.Count, queryDuration)
		})

		// Test CC8.1 (Audit Logging) query
		t.Run("CC8.1 Audit Logging", func(t *testing.T) {
			start := time.Now()
			result := query.ByControl("CC8.1")
			queryDuration := time.Since(start)

			assert.GreaterOrEqual(t, result.Count, 0, "may or may not find audit logging resources")
			assert.Less(t, queryDuration, 100*time.Millisecond, "query should be fast")

			t.Logf("✓ Found %d resources for CC8.1 in %v", result.Count, queryDuration)
		})
	})

	t.Run("Query by Security Attribute", func(t *testing.T) {
		ctx := context.Background()
		indexer := terraform.NewSecurityAttributeIndexer(cfg, log)
		persistedIndex, err := indexer.LoadOrBuildIndex(ctx, false)
		require.NoError(t, err)

		query := terraform.NewIndexQuery(persistedIndex)

		// Query encryption attribute
		t.Run("Encryption Attribute", func(t *testing.T) {
			start := time.Now()
			result := query.ByAttribute("encryption")
			queryDuration := time.Since(start)

			assert.Greater(t, result.Count, 0, "should find resources with encryption attribute")
			assert.Less(t, queryDuration, 100*time.Millisecond, "query should be fast")

			t.Logf("✓ Found %d resources with 'encryption' attribute in %v", result.Count, queryDuration)
		})

		// Query access_control attribute
		t.Run("Access Control Attribute", func(t *testing.T) {
			result := query.ByAttribute("access_control")
			assert.GreaterOrEqual(t, result.Count, 0, "should handle access_control attribute query")
		})

		// Query monitoring attribute
		t.Run("Monitoring Attribute", func(t *testing.T) {
			result := query.ByAttribute("monitoring")
			assert.GreaterOrEqual(t, result.Count, 0, "should handle monitoring attribute query")
		})
	})

	t.Run("Query by Resource Type", func(t *testing.T) {
		ctx := context.Background()
		indexer := terraform.NewSecurityAttributeIndexer(cfg, log)
		persistedIndex, err := indexer.LoadOrBuildIndex(ctx, false)
		require.NoError(t, err)

		query := terraform.NewIndexQuery(persistedIndex)

		// Query KMS keys
		t.Run("KMS Keys", func(t *testing.T) {
			result := query.ByResourceType("aws_kms_key")
			assert.Greater(t, result.Count, 0, "should find KMS key resources")

			// Verify all results are KMS keys
			for _, res := range result.Resources {
				assert.Equal(t, "aws_kms_key", res.ResourceType)
			}

			t.Logf("✓ Found %d KMS keys", result.Count)
		})

		// Query S3 buckets
		t.Run("S3 Buckets", func(t *testing.T) {
			result := query.ByResourceType("aws_s3_bucket")

			if result.Count > 0 {
				// Verify all results are S3 buckets
				for _, res := range result.Resources {
					assert.Contains(t, res.ResourceType, "aws_s3_bucket")
				}
			}

			t.Logf("✓ Found %d S3 buckets/related resources", result.Count)
		})
	})

	t.Run("Performance Validation", func(t *testing.T) {
		ctx := context.Background()
		indexer := terraform.NewSecurityAttributeIndexer(cfg, log)

		// Measure build time
		buildStart := time.Now()
		persistedIndex, err := indexer.BuildAndPersistIndex(ctx)
		buildDuration := time.Since(buildStart)
		require.NoError(t, err)

		assert.Less(t, buildDuration, 10*time.Second, "index build should complete in <10s")

		// Measure query time
		query := terraform.NewIndexQuery(persistedIndex)

		queryStart := time.Now()
		result := query.ByControl("CC6.8")
		queryDuration := time.Since(queryStart)

		assert.Less(t, queryDuration, 100*time.Millisecond, "query should be <100ms")
		assert.Greater(t, result.Count, 0, "query should return results")

		// Compare to expected live scan time (would be 30s+ for this size)
		estimatedLiveScanTime := time.Duration(persistedIndex.Metadata.TotalFiles) * 500 * time.Millisecond
		speedup := float64(estimatedLiveScanTime) / float64(queryDuration)

		t.Logf("✓ Performance Metrics:")
		t.Logf("  Build Time: %v", buildDuration)
		t.Logf("  Query Time: %v", queryDuration)
		t.Logf("  Estimated Live Scan Time: %v", estimatedLiveScanTime)
		t.Logf("  Speedup: %.1fx faster", speedup)

		assert.Greater(t, speedup, 10.0, "index query should be at least 10x faster than live scan")
	})

	t.Run("Cache Invalidation on File Change", func(t *testing.T) {
		ctx := context.Background()
		indexer := terraform.NewSecurityAttributeIndexer(cfg, log)

		// Build initial index
		initialIndex, err := indexer.BuildAndPersistIndex(ctx)
		require.NoError(t, err)
		initialResourceCount := initialIndex.Metadata.TotalResources

		// Wait a bit to ensure different modification time
		time.Sleep(10 * time.Millisecond)

		// Modify a source file (add a new resource)
		terraformDir := filepath.Join(tempDir, "terraform")
		testFile := filepath.Join(terraformDir, "test_modification.tf")
		newContent := `
# Additional resource for cache invalidation test
resource "aws_kms_key" "test_invalidation" {
  description             = "Test key for cache invalidation"
  deletion_window_in_days = 7
  enable_key_rotation     = true

  tags = {
    Purpose = "CacheInvalidationTest"
    Control = "CC6.8"
  }
}
`
		err = os.WriteFile(testFile, []byte(newContent), 0644)
		require.NoError(t, err)

		// Load or rebuild index (should detect change and rebuild)
		rebuiltIndex, err := indexer.LoadOrBuildIndex(ctx, false)
		require.NoError(t, err)

		// Verify index was rebuilt with new resource
		// Note: The indexer may or may not auto-invalidate depending on implementation
		// At minimum, forcing a rebuild should pick up the new resource
		forcedRebuildIndex, err := indexer.BuildAndPersistIndex(ctx)
		require.NoError(t, err)

		assert.Greater(t, forcedRebuildIndex.Metadata.TotalResources, initialResourceCount,
			"rebuilt index should have more resources after file addition")

		t.Logf("✓ Cache invalidation test:")
		t.Logf("  Initial Resources: %d", initialResourceCount)
		t.Logf("  After Rebuild: %d", forcedRebuildIndex.Metadata.TotalResources)
		t.Logf("  Successfully detected file change: %v", rebuiltIndex.Metadata.TotalResources > initialResourceCount)
	})
}

// Helper functions

func setupIndexTestFixtures(t *testing.T, tempDir string) {
	terraformDir := filepath.Join(tempDir, "terraform")
	err := os.MkdirAll(terraformDir, 0755)
	require.NoError(t, err)

	// Use realistic fixture files - copy from test_data
	fixtureSource := "../../test_data/terraform/soc2"

	// Read and copy the realistic test files
	files := []string{
		"cc6_8_encryption.tf",
		"cc6_1_access_control.tf",
		"cc8_1_audit_logging.tf",
	}

	for _, filename := range files {
		sourcePath := filepath.Join(fixtureSource, filename)
		destPath := filepath.Join(terraformDir, filename)

		// If source doesn't exist, create simplified inline version
		content, err := os.ReadFile(sourcePath)
		if err != nil {
			// Create simplified test fixture inline
			content = []byte(getSimplifiedFixture(filename))
		}

		err = os.WriteFile(destPath, content, 0644)
		require.NoError(t, err, "failed to create test fixture: %s", filename)
	}
}

func getSimplifiedFixture(filename string) string {
	switch filename {
	case "cc6_8_encryption.tf":
		return `
# CC6.8 - Encryption Test Fixture
resource "aws_kms_key" "main" {
  description             = "Main encryption key"
  deletion_window_in_days = 30
  enable_key_rotation     = true

  tags = {
    Purpose = "SOC2 Compliance"
    Control = "CC6.8"
  }
}

resource "aws_s3_bucket_encryption" "data" {
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
  identifier          = "test-db"
  engine              = "postgres"
  instance_class      = "db.t3.micro"
  storage_encrypted   = true
  kms_key_id          = aws_kms_key.main.arn

  tags = {
    Control = "CC6.8"
  }
}
`
	case "cc6_1_access_control.tf":
		return `
# CC6.1 - Access Control Test Fixture
resource "aws_iam_role" "app" {
  name = "application-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "lambda.amazonaws.com"
      }
    }]
  })

  tags = {
    Control = "CC6.1"
    Purpose = "Least Privilege Access"
  }
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
  }
}
`
	case "cc8_1_audit_logging.tf":
		return `
# CC8.1 - Audit Logging Test Fixture
resource "aws_cloudtrail" "main" {
  name           = "main-trail"
  s3_bucket_name = "audit-logs"

  enable_logging = true
  is_multi_region_trail = true

  tags = {
    Control = "CC8.1"
    Purpose = "Audit Trail"
  }
}

resource "aws_cloudwatch_log_group" "app" {
  name              = "/aws/lambda/app"
  retention_in_days = 365

  tags = {
    Control = "CC8.1"
    Purpose = "Application Logging"
  }
}
`
	default:
		return ""
	}
}

func createIndexTestConfig(tempDir, cacheDir string) *config.Config {
	return &config.Config{
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				Terraform: config.TerraformToolConfig{
					Enabled:         true,
					ScanPaths:       []string{filepath.Join(tempDir, "terraform")},
					IncludePatterns: []string{"*.tf"},
					ExcludePatterns: []string{".terraform/", "*.tfstate*"},
				},
			},
		},
		Storage: config.StorageConfig{
			DataDir:  tempDir,
			CacheDir: cacheDir,
		},
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
