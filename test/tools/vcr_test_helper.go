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
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/vcr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// VCRTestConfig provides configuration for VCR-enabled tests
type VCRTestConfig struct {
	Mode          vcr.Mode
	CassetteName  string
	CassetteDir   string
	RedactHeaders []string
	RedactParams  []string
	SanitizeData  bool
	TestLayer     string // "unit", "integration", "functional", "e2e"
	RecordReal    bool
	BaseURL       string
	ExpectedCalls int
}

// DefaultVCRConfig returns a default VCR test configuration
func DefaultVCRConfig() *VCRTestConfig {
	return &VCRTestConfig{
		Mode:          vcr.ModePlayback,
		SanitizeData:  true,
		TestLayer:     "integration",
		RecordReal:    false,
		ExpectedCalls: 1,
		RedactHeaders: []string{"authorization", "cookie", "x-api-key", "token", "x-auth-token"},
		RedactParams:  []string{"api_key", "token", "password", "secret", "access_token"},
	}
}

// SetupVCRTest creates a VCR-enabled HTTP client for testing
func SetupVCRTest(t *testing.T, testConfig *VCRTestConfig) (*http.Client, *vcr.VCR) {
	if testConfig == nil {
		testConfig = DefaultVCRConfig()
	}

	// Use specified cassette directory or default to project vcr_cassettes
	cassetteDir := testConfig.CassetteDir
	if cassetteDir == "" {
		// Default to project vcr_cassettes directory
		cassetteDir = filepath.Join("..", "..", "test", "vcr_cassettes")
	}

	// Configure VCR
	vcrConfig := &vcr.Config{
		Enabled:         true,
		Mode:            testConfig.Mode,
		CassetteDir:     cassetteDir,
		SanitizeHeaders: testConfig.SanitizeData,
		SanitizeParams:  testConfig.SanitizeData,
		RedactHeaders:   []string{"authorization", "cookie", "x-api-key", "token", "x-auth-token"},
		RedactParams:    []string{"api_key", "token", "password", "secret", "access_token"},
		MatchMethod:     true,
		MatchURI:        true,
		MatchQuery:      false,
		MatchHeaders:    false,
		MatchBody:       false,
	}

	// Create VCR instance
	vcrInstance := vcr.New(vcrConfig)

	// Create HTTP client with VCR transport
	client := &http.Client{
		Transport: vcrInstance,
	}

	return client, vcrInstance
}

// Layer-specific VCR setup functions
func SetupUnitVCR(t *testing.T, cassetteName string) (*http.Client, *vcr.VCR) {
	config := &VCRTestConfig{
		Mode:          vcr.ModePlayback,
		CassetteName:  cassetteName,
		TestLayer:     "unit",
		SanitizeData:  true,
		RedactHeaders: []string{"authorization", "cookie", "x-api-key", "token", "x-auth-token"},
		RedactParams:  []string{"api_key", "token", "password", "secret", "access_token"},
	}
	return SetupLayerVCR(t, config)
}

func SetupIntegrationVCR(t *testing.T, cassetteName string) (*http.Client, *vcr.VCR) {
	config := &VCRTestConfig{
		Mode:          getVCRModeFromEnv(),
		CassetteName:  cassetteName,
		TestLayer:     "integration",
		SanitizeData:  true,
		RedactHeaders: []string{"authorization", "cookie", "x-api-key", "token", "x-auth-token"},
		RedactParams:  []string{"api_key", "token", "password", "secret", "access_token"},
	}
	return SetupLayerVCR(t, config)
}

func SetupLayerVCR(t *testing.T, config *VCRTestConfig) (*http.Client, *vcr.VCR) {
	if config == nil {
		config = DefaultVCRConfig()
	}

	// Use test-specific cassette directory if not provided
	cassetteDir := config.CassetteDir
	if cassetteDir == "" {
		cassetteDir = filepath.Join(t.TempDir(), "vcr_cassettes")
	}

	// Configure VCR
	vcrConfig := &vcr.Config{
		Enabled:         true,
		Mode:            config.Mode,
		CassetteDir:     cassetteDir,
		SanitizeHeaders: config.SanitizeData,
		SanitizeParams:  config.SanitizeData,
		RedactHeaders:   config.RedactHeaders,
		RedactParams:    config.RedactParams,
		MatchMethod:     true,
		MatchURI:        true,
		MatchQuery:      false,
		MatchHeaders:    false,
		MatchBody:       false,
	}

	// Create VCR instance
	vcrInstance := vcr.New(vcrConfig)

	// Create HTTP client with VCR transport
	client := &http.Client{
		Transport: vcrInstance,
	}

	return client, vcrInstance
}

func getVCRModeFromEnv() vcr.Mode {
	switch os.Getenv("VCR_MODE") {
	case "record":
		return vcr.ModeRecord
	case "record_once":
		return vcr.ModeRecordOnce
	default:
		return vcr.ModePlayback
	}
}

// Cassette management helpers
func CleanCassettes(t *testing.T, pattern string) {
	matches, err := filepath.Glob(pattern)
	require.NoError(t, err, "Failed to find cassette files with pattern: %s", pattern)

	for _, match := range matches {
		err := os.Remove(match)
		require.NoError(t, err, "Failed to remove cassette file: %s", match)
		t.Logf("Cleaned cassette: %s", match)
	}
}

func ValidateCassette(t *testing.T, cassetteName string) {
	_, err := os.Stat(cassetteName)
	assert.NoError(t, err, "Cassette should exist: %s", cassetteName)

	// Check that cassette has content
	info, err := os.Stat(cassetteName)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0), "Cassette should not be empty")
}

// CreateTestLogger creates a logger for testing
func CreateTestLogger(t *testing.T) logger.Logger {
	log, err := logger.New(&logger.Config{
		Level:  logger.ErrorLevel,
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err)
	return log
}

// CreateTestConfig creates a base configuration for tool testing
func CreateTestConfig(tempDir string) *config.Config {
	return &config.Config{
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				GitHub: config.GitHubToolConfig{
					Enabled:    true,
					APIToken:   "test-token-redacted",
					Repository: "test-org/test-repo",
				},
				Terraform: config.TerraformToolConfig{
					Enabled:         true,
					ScanPaths:       []string{tempDir},
					IncludePatterns: []string{"*.tf", "*.tfvars"},
					ExcludePatterns: []string{".terraform/"},
				},
			},
		},
		Storage: config.StorageConfig{
			DataDir: tempDir,
		},
	}
}

// SampleGitHubResponses provides mock GitHub API responses for testing
var SampleGitHubResponses = map[string]interface{}{
	"search_code": map[string]interface{}{
		"total_count":        3,
		"incomplete_results": false,
		"items": []interface{}{
			map[string]interface{}{
				"name":     "security-config.tf",
				"path":     "terraform/security-config.tf",
				"sha":      "abc123def456",
				"url":      "https://api.github.com/repositories/123456/contents/terraform/security-config.tf",
				"git_url":  "https://api.github.com/repositories/123456/git/blobs/abc123def456",
				"html_url": "https://github.com/test-org/test-repo/blob/main/terraform/security-config.tf",
				"repository": map[string]interface{}{
					"id":        123456,
					"name":      "test-repo",
					"full_name": "test-org/test-repo",
					"private":   false,
					"owner": map[string]interface{}{
						"login": "test-org",
						"type":  "Organization",
					},
				},
				"score": 1.0,
				"text_matches": []interface{}{
					map[string]interface{}{
						"object_url":  "https://api.github.com/repositories/123456/contents/terraform/security-config.tf",
						"object_type": "FileMatch",
						"property":    "content",
						"fragment":    "resource \"aws_kms_key\" \"encryption_key\" {\n  description = \"KMS key for data encryption\"\n  enable_key_rotation = true\n}",
						"matches": []interface{}{
							map[string]interface{}{
								"text":    "encryption",
								"indices": []int{45, 55},
							},
						},
					},
				},
			},
			map[string]interface{}{
				"name":     "security-policy.md",
				"path":     "docs/security-policy.md",
				"sha":      "def456ghi789",
				"url":      "https://api.github.com/repositories/123456/contents/docs/security-policy.md",
				"git_url":  "https://api.github.com/repositories/123456/git/blobs/def456ghi789",
				"html_url": "https://github.com/test-org/test-repo/blob/main/docs/security-policy.md",
				"repository": map[string]interface{}{
					"id":        123456,
					"name":      "test-repo",
					"full_name": "test-org/test-repo",
					"private":   false,
					"owner": map[string]interface{}{
						"login": "test-org",
						"type":  "Organization",
					},
				},
				"score": 0.8,
				"text_matches": []interface{}{
					map[string]interface{}{
						"object_url":  "https://api.github.com/repositories/123456/contents/docs/security-policy.md",
						"object_type": "FileMatch",
						"property":    "content",
						"fragment":    "## Data Encryption\n\nAll sensitive data is encrypted at rest using AES-256 encryption.",
						"matches": []interface{}{
							map[string]interface{}{
								"text":    "encryption",
								"indices": []int{8, 18},
							},
							map[string]interface{}{
								"text":    "encrypted",
								"indices": []int{45, 54},
							},
						},
					},
				},
			},
		},
	},
	"file_content": map[string]interface{}{
		"type":         "file",
		"encoding":     "base64",
		"size":         1234,
		"name":         "security-policy.md",
		"path":         "docs/security-policy.md",
		"content":      "IyBTZWN1cml0eSBQb2xpY3kKCiMjIERhdGEgRW5jcnlwdGlvbgoKQWxsIHNlbnNpdGl2ZSBkYXRhIGlzIGVuY3J5cHRlZCBhdCByZXN0IHVzaW5nIEFFUy0yNTYgZW5jcnlwdGlvbi4KRGF0YSBpbiB0cmFuc2l0IGlzIHByb3RlY3RlZCB1c2luZyBUTFMgMS4zLgoKIyMgQWNjZXNzIENvbnRyb2wKCk11bHRpLWZhY3RvciBhdXRoZW50aWNhdGlvbiBpcyByZXF1aXJlZCBmb3IgYWxsIGFkbWluaXN0cmF0aXZlIGFjY2Vzcy4KUm9sZS1iYXNlZCBhY2Nlc3MgY29udHJvbCAoUkJBQykgaXMgaW1wbGVtZW50ZWQgYWNyb3NzIGFsbCBzeXN0ZW1zLg==",
		"sha":          "def456ghi789",
		"url":          "https://api.github.com/repositories/123456/contents/docs/security-policy.md",
		"git_url":      "https://api.github.com/repositories/123456/git/blobs/def456ghi789",
		"html_url":     "https://github.com/test-org/test-repo/blob/main/docs/security-policy.md",
		"download_url": "https://raw.githubusercontent.com/test-org/test-repo/main/docs/security-policy.md",
		"_links": map[string]interface{}{
			"git":  "https://api.github.com/repositories/123456/git/blobs/def456ghi789",
			"self": "https://api.github.com/repositories/123456/contents/docs/security-policy.md",
			"html": "https://github.com/test-org/test-repo/blob/main/docs/security-policy.md",
		},
	},
	"repository_tree": map[string]interface{}{
		"sha":       "main",
		"url":       "https://api.github.com/repositories/123456/git/trees/main",
		"truncated": false,
		"tree": []interface{}{
			map[string]interface{}{
				"path": ".github",
				"mode": "040000",
				"type": "tree",
				"sha":  "tree1",
				"url":  "https://api.github.com/repositories/123456/git/trees/tree1",
			},
			map[string]interface{}{
				"path": "docs",
				"mode": "040000",
				"type": "tree",
				"sha":  "tree2",
				"url":  "https://api.github.com/repositories/123456/git/trees/tree2",
			},
			map[string]interface{}{
				"path": "terraform",
				"mode": "040000",
				"type": "tree",
				"sha":  "tree3",
				"url":  "https://api.github.com/repositories/123456/git/trees/tree3",
			},
			map[string]interface{}{
				"path": "README.md",
				"mode": "100644",
				"type": "blob",
				"sha":  "readme123",
				"size": 1500,
				"url":  "https://api.github.com/repositories/123456/git/blobs/readme123",
			},
			map[string]interface{}{
				"path": "docs/security-policy.md",
				"mode": "100644",
				"type": "blob",
				"sha":  "def456ghi789",
				"size": 1234,
				"url":  "https://api.github.com/repositories/123456/git/blobs/def456ghi789",
			},
			map[string]interface{}{
				"path": "terraform/security-config.tf",
				"mode": "100644",
				"type": "blob",
				"sha":  "abc123def456",
				"size": 2048,
				"url":  "https://api.github.com/repositories/123456/git/blobs/abc123def456",
			},
		},
	},
	"rate_limit_error": map[string]interface{}{
		"message":           "API rate limit exceeded for user ID 123.",
		"documentation_url": "https://docs.github.com/rest/overview/resources-in-the-rest-api#rate-limiting",
	},
	"not_found_error": map[string]interface{}{
		"message":           "Not Found",
		"documentation_url": "https://docs.github.com/rest",
	},
}

// TestDataFiles provides sample files for testing tools
var TestDataFiles = map[string]string{
	"terraform/main.tf": `
# Main Terraform configuration
terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# S3 Bucket with encryption
resource "aws_s3_bucket" "secure_bucket" {
  bucket = "example-secure-bucket"
}

resource "aws_s3_bucket_encryption" "secure_bucket" {
  bucket = aws_s3_bucket.secure_bucket.id

  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        kms_master_key_id = aws_kms_key.s3_key.arn
        sse_algorithm     = "aws:kms"
      }
    }
  }
}

resource "aws_kms_key" "s3_key" {
  description             = "KMS key for S3 bucket encryption"
  deletion_window_in_days = 7
  enable_key_rotation     = true
}
`,
	"docs/security-policy.md": `# Security Policy

## Data Encryption

All sensitive data is encrypted at rest using AES-256 encryption.
Data in transit is protected using TLS 1.3.

## Access Control

Multi-factor authentication is required for all administrative access.
Role-based access control (RBAC) is implemented across all systems.

## Compliance

We maintain SOC 2 Type II compliance and undergo annual security audits.
Vulnerability assessments are performed quarterly.

## Incident Response

Our incident response team is available 24/7 for security incidents.
All incidents are logged and tracked for resolution.
`,
	"docs/privacy-policy.md": `# Privacy Policy

## Data Collection

We collect minimal necessary data and follow privacy by design principles.
Personal data is processed only for legitimate business purposes.

## Data Retention

Personal data is retained only as long as necessary for business purposes.
Data is automatically purged after retention periods expire.

## User Rights

Users have the right to access, modify, and delete their personal data.
Privacy requests are processed within 30 days.
`,
	"config/application.yaml": `# Application configuration
server:
  port: 8080
  ssl:
    enabled: true
    key-store: /etc/ssl/app.jks
    key-store-password: ${SSL_KEYSTORE_PASSWORD}

security:
  oauth2:
    client:
      registration:
        github:
          client-id: ${GITHUB_CLIENT_ID}
          client-secret: ${GITHUB_CLIENT_SECRET}
          
database:
  url: ${DATABASE_URL}
  username: ${DATABASE_USERNAME}
  password: ${DATABASE_PASSWORD}
  encryption:
    enabled: true
    algorithm: AES-256
`,
}

// CreateTestDataFiles creates test files in the specified directory
func CreateTestDataFiles(t *testing.T, baseDir string, files map[string]string) {
	for relativePath, content := range files {
		fullPath := filepath.Join(baseDir, relativePath)

		// Create directory if it doesn't exist
		dir := filepath.Dir(fullPath)
		err := os.MkdirAll(dir, 0755)
		require.NoError(t, err)

		// Create the file
		err = os.WriteFile(fullPath, []byte(content), 0644)
		require.NoError(t, err)
	}
}

// AssertVCRCassetteCreated verifies that a VCR cassette was created during recording
func AssertVCRCassetteCreated(t *testing.T, cassetteDir, cassetteName string) {
	cassettePath := filepath.Join(cassetteDir, cassetteName)
	_, err := os.Stat(cassettePath)
	assert.NoError(t, err, "VCR cassette should be created: %s", cassettePath)
}

// AssertVCRCassetteUsed verifies that a VCR cassette was loaded during playback
func AssertVCRCassetteUsed(t *testing.T, vcrInstance *vcr.VCR) {
	stats := vcrInstance.Stats()

	// Check that cassettes were loaded
	loadedCassettes, ok := stats["loaded_cassettes"].(int)
	assert.True(t, ok, "loaded_cassettes should be an integer")
	assert.Greater(t, loadedCassettes, 0, "At least one cassette should be loaded")

	// Check that interactions were replayed
	totalInteractions, ok := stats["total_interactions"].(int)
	assert.True(t, ok, "total_interactions should be an integer")
	assert.Greater(t, totalInteractions, 0, "At least one interaction should be replayed")
}

// ValidateVCRCassette checks if a VCR cassette exists and is properly formatted
func ValidateVCRCassette(t *testing.T, cassetteDir, cassetteName string) {
	cassettePath := filepath.Join(cassetteDir, cassetteName)

	// Check if cassette file exists
	_, err := os.Stat(cassettePath)
	assert.NoError(t, err, "VCR cassette should exist: %s", cassettePath)

	// Check cassette file is not empty
	info, err := os.Stat(cassettePath)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(100), "VCR cassette should not be empty")
}

// RecordVCRCassette sets up VCR in record mode for capturing real API interactions
func RecordVCRCassette(t *testing.T, cassetteName string) (*http.Client, *vcr.VCR) {
	if testing.Short() {
		t.Skip("Skipping VCR recording in short mode")
	}

	testConfig := &VCRTestConfig{
		Mode:         vcr.ModeRecord,
		CassetteName: cassetteName,
		SanitizeData: true,
		RecordReal:   true,
	}

	cassetteDir := filepath.Join("..", "..", "test", "vcr_cassettes")

	// Ensure cassette directory exists
	err := os.MkdirAll(cassetteDir, 0755)
	require.NoError(t, err)

	vcrConfig := &vcr.Config{
		Enabled:         true,
		Mode:            testConfig.Mode,
		CassetteDir:     cassetteDir,
		SanitizeHeaders: true,
		SanitizeParams:  true,
		RedactHeaders: []string{
			"authorization", "cookie", "x-api-key", "token", "x-auth-token",
			"x-github-token", "authorization-header", "bearer",
		},
		RedactParams: []string{
			"api_key", "token", "password", "secret", "access_token",
			"client_secret", "private_key", "auth_token",
		},
		MatchMethod:  true,
		MatchURI:     true,
		MatchQuery:   true,
		MatchHeaders: false,
		MatchBody:    false,
	}

	vcrInstance := vcr.New(vcrConfig)

	client := &http.Client{
		Transport: vcrInstance,
		Timeout:   60 * time.Second, // Longer timeout for recording
	}

	return client, vcrInstance
}
