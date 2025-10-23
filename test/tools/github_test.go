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

//go:build !e2e

package tools_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/tools"
	// "github.com/grctool/grctool/internal/vcr" // Disabled - VCR config removed
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock GitHub API responses
var mockSearchResponse = map[string]interface{}{
	"total_count": 2,
	"items": []interface{}{
		map[string]interface{}{
			"name":     "security-policy.md",
			"path":     "docs/security-policy.md",
			"html_url": "https://github.com/test/repo/blob/main/docs/security-policy.md",
			"text_matches": []interface{}{
				map[string]interface{}{
					"fragment": "Our security policy includes encryption at rest",
				},
			},
		},
		map[string]interface{}{
			"name":     "terraform/main.tf",
			"path":     "terraform/main.tf",
			"html_url": "https://github.com/test/repo/blob/main/terraform/main.tf",
			"text_matches": []interface{}{
				map[string]interface{}{
					"fragment": "resource \"aws_kms_key\" \"example\"",
				},
			},
		},
	},
}

var mockContentResponse = map[string]interface{}{
	"name":     "security-policy.md",
	"path":     "docs/security-policy.md",
	"sha":      "abc123",
	"size":     1234,
	"content":  "IyBTZWN1cml0eSBQb2xpY3kKCk91ciBzZWN1cml0eSBwb2xpY3kgaW5jbHVkZXMgZW5jcnlwdGlvbiBhdCByZXN0Lg==", // base64 "# Security Policy\n\nOur security policy includes encryption at rest."
	"encoding": "base64",
	"html_url": "https://github.com/test/repo/blob/main/docs/security-policy.md",
}

var mockTreeResponse = map[string]interface{}{
	"sha": "main",
	"tree": []interface{}{
		map[string]interface{}{
			"path": "docs",
			"type": "tree",
			"sha":  "tree1",
		},
		map[string]interface{}{
			"path": "terraform",
			"type": "tree",
			"sha":  "tree2",
		},
		map[string]interface{}{
			"path": "docs/security-policy.md",
			"type": "blob",
			"sha":  "abc123",
		},
		map[string]interface{}{
			"path": "terraform/main.tf",
			"type": "blob",
			"sha":  "def456",
		},
	},
}

func TestGitHubTool_BasicProperties(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				GitHub: config.GitHubToolConfig{
					Enabled:    true,
					APIToken:   "test-token",
					Repository: "test-org/test-repo",
				},
			},
		},
		Storage: config.StorageConfig{
			DataDir: "/tmp/grctool-test",
		},
	}

	log, err := logger.New(&logger.Config{
		Level:  logger.ErrorLevel,
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err)

	tool := tools.NewGitHubTool(cfg, log)

	t.Run("Tool Properties", func(t *testing.T) {
		assert.Equal(t, "github_searcher", tool.Name())
		assert.Contains(t, tool.Description(), "GitHub repository")

		definition := tool.GetClaudeToolDefinition()
		assert.Equal(t, "github_searcher", definition.Name)
		assert.NotNil(t, definition.InputSchema)

		// Check schema properties
		schema := definition.InputSchema
		properties := schema["properties"].(map[string]interface{})
		assert.Contains(t, properties, "query")
		assert.Contains(t, properties, "search_type")
		assert.Contains(t, properties, "file_extensions")
		assert.Contains(t, properties, "paths")
	})
}

func TestGitHubTool_SearchCode(t *testing.T) {
	// Create mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search/code":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockSearchResponse)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	// Create test configuration with mock server
	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				GitHub: config.GitHubToolConfig{
					Enabled:    true,
					APIToken:   "test-token",
					Repository: "test-org/test-repo",
				},
			},
		},
		Storage: config.StorageConfig{
			DataDir: "/tmp/grctool-test",
		},
	}

	log, err := logger.New(&logger.Config{
		Level:  logger.ErrorLevel,
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err)

	tool := tools.NewGitHubTool(cfg, log)

	t.Run("Search Code", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"query":       "encryption",
			"search_type": "code",
		}

		result, source, err := tool.Execute(ctx, params)
		require.NoError(t, err)
		assert.NotEmpty(t, result)
		assert.NotNil(t, source)
		assert.Equal(t, "github_searcher", source.Type)

		// Check that result contains expected content
		assert.Contains(t, result, "security-policy.md")
		assert.Contains(t, result, "terraform/main.tf")
		assert.Contains(t, result, "encryption at rest")
		assert.Contains(t, result, "aws_kms_key")
	})

	t.Run("Search with File Extensions", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"query":           "resource",
			"search_type":     "code",
			"file_extensions": []interface{}{"tf", "tfvars"},
		}

		result, source, err := tool.Execute(ctx, params)
		require.NoError(t, err)
		assert.NotEmpty(t, result)
		assert.NotNil(t, source)
	})

	t.Run("Search with Paths", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"query":       "security",
			"search_type": "code",
			"paths":       []interface{}{"docs/", "policies/"},
		}

		result, source, err := tool.Execute(ctx, params)
		require.NoError(t, err)
		assert.NotEmpty(t, result)
		assert.NotNil(t, source)
	})
}

func TestGitHubTool_GetContent(t *testing.T) {
	// Create mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/test-org/test-repo/contents/docs/security-policy.md":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockContentResponse)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	// Create test configuration with mock server
	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				GitHub: config.GitHubToolConfig{
					Enabled:    true,
					APIToken:   "test-token",
					Repository: "test-org/test-repo",
				},
			},
		},
		Storage: config.StorageConfig{
			DataDir: "/tmp/grctool-test",
		},
	}

	log, err := logger.New(&logger.Config{
		Level:  logger.ErrorLevel,
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err)

	tool := tools.NewGitHubTool(cfg, log)

	t.Run("Get File Content", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"search_type": "content",
			"file_path":   "docs/security-policy.md",
		}

		result, source, err := tool.Execute(ctx, params)
		require.NoError(t, err)
		assert.NotEmpty(t, result)
		assert.NotNil(t, source)

		// Check that result contains decoded content
		assert.Contains(t, result, "# Security Policy")
		assert.Contains(t, result, "encryption at rest")
	})
}

func TestGitHubTool_ListFiles(t *testing.T) {
	// Create mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/test-org/test-repo/git/trees/main":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockTreeResponse)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	// Create test configuration with mock server
	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				GitHub: config.GitHubToolConfig{
					Enabled:    true,
					APIToken:   "test-token",
					Repository: "test-org/test-repo",
				},
			},
		},
		Storage: config.StorageConfig{
			DataDir: "/tmp/grctool-test",
		},
	}

	log, err := logger.New(&logger.Config{
		Level:  logger.ErrorLevel,
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err)

	tool := tools.NewGitHubTool(cfg, log)

	t.Run("List Repository Files", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"search_type": "tree",
			"recursive":   true,
		}

		result, source, err := tool.Execute(ctx, params)
		require.NoError(t, err)
		assert.NotEmpty(t, result)
		assert.NotNil(t, source)

		// Check that result contains expected files
		assert.Contains(t, result, "docs/security-policy.md")
		assert.Contains(t, result, "terraform/main.tf")
	})

	t.Run("List Files with Path Filter", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"search_type":  "tree",
			"recursive":    true,
			"path_pattern": "docs/",
		}

		result, source, err := tool.Execute(ctx, params)
		require.NoError(t, err)
		assert.NotEmpty(t, result)
		assert.NotNil(t, source)
	})
}

func TestGitHubTool_WithVCR(t *testing.T) {
	t.Skip("VCR tests disabled - config.VCR field no longer exists. TODO: Update to current VCR implementation")

	// Create temporary directory for VCR cassettes
	tempDir := t.TempDir()
	_ = filepath.Join(tempDir, "vcr_cassettes") // cassetteDir unused after VCR removal

	// Create test configuration with VCR enabled
	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				GitHub: config.GitHubToolConfig{
					Enabled:    true,
					APIToken:   "test-token",
					Repository: "test-org/test-repo",
				},
			},
		},
		Storage: config.StorageConfig{
			DataDir: tempDir,
		},
	}

	log, err := logger.New(&logger.Config{
		Level:  logger.ErrorLevel,
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err)

	tool := tools.NewGitHubTool(cfg, log)

	t.Run("VCR Recording Mode", func(t *testing.T) {
		// This test would record real API calls if run against a real GitHub repo
		// For testing purposes, we'll just verify the tool can be configured with VCR
		assert.Equal(t, "github_searcher", tool.Name())
		// In a real scenario, this would make HTTP calls that get recorded
	})
}

func TestGitHubTool_BasicErrorHandling(t *testing.T) {
	// Create mock HTTP server that returns errors
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search/code":
			http.Error(w, "API rate limit exceeded", http.StatusTooManyRequests)
		case "/repos/test-org/test-repo/contents/nonexistent.md":
			http.Error(w, "Not Found", http.StatusNotFound)
		default:
			http.Error(w, "Server Error", http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	// Create test configuration with mock server
	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				GitHub: config.GitHubToolConfig{
					Enabled:    true,
					APIToken:   "test-token",
					Repository: "test-org/test-repo",
				},
			},
		},
		Storage: config.StorageConfig{
			DataDir: "/tmp/grctool-test",
		},
	}

	log, err := logger.New(&logger.Config{
		Level:  logger.ErrorLevel,
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err)

	tool := tools.NewGitHubTool(cfg, log)

	t.Run("Rate Limit Error", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"query":       "test",
			"search_type": "code",
		}

		_, _, err := tool.Execute(ctx, params)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rate limit")
	})

	t.Run("File Not Found Error", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"search_type": "content",
			"file_path":   "nonexistent.md",
		}

		_, _, err := tool.Execute(ctx, params)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "404")
	})

	t.Run("Invalid Parameters", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"search_type": "invalid",
		}

		_, _, err := tool.Execute(ctx, params)
		assert.Error(t, err)
	})

	t.Run("Missing Required Parameters", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"search_type": "code",
			// missing query
		}

		_, _, err := tool.Execute(ctx, params)
		assert.Error(t, err)
	})
}

func TestGitHubTool_ParameterValidation(t *testing.T) {
	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				GitHub: config.GitHubToolConfig{
					Enabled:    true,
					APIToken:   "test-token",
					Repository: "test-org/test-repo",
				},
			},
		},
		Storage: config.StorageConfig{
			DataDir: "/tmp/grctool-test",
		},
	}

	log, err := logger.New(&logger.Config{
		Level:  logger.ErrorLevel,
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err)

	tool := tools.NewGitHubTool(cfg, log)

	testCases := []struct {
		name        string
		params      map[string]interface{}
		expectError bool
		errorText   string
	}{
		{
			name: "Valid code search parameters",
			params: map[string]interface{}{
				"query":       "test",
				"search_type": "code",
			},
			expectError: false,
		},
		{
			name: "Valid content parameters",
			params: map[string]interface{}{
				"search_type": "content",
				"file_path":   "README.md",
			},
			expectError: false,
		},
		{
			name: "Valid tree parameters",
			params: map[string]interface{}{
				"search_type": "tree",
				"recursive":   true,
			},
			expectError: false,
		},
		{
			name: "Invalid search type",
			params: map[string]interface{}{
				"search_type": "invalid_type",
			},
			expectError: true,
			errorText:   "invalid search_type",
		},
		{
			name: "Missing query for code search",
			params: map[string]interface{}{
				"search_type": "code",
			},
			expectError: true,
			errorText:   "query is required",
		},
		{
			name: "Missing file_path for content",
			params: map[string]interface{}{
				"search_type": "content",
			},
			expectError: true,
			errorText:   "file_path is required",
		},
		{
			name:        "Empty parameters",
			params:      map[string]interface{}{},
			expectError: true,
			errorText:   "search_type is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			_, _, err := tool.Execute(ctx, tc.params)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorText != "" {
					assert.Contains(t, err.Error(), tc.errorText)
				}
			} else {
				// For valid parameters, we expect network errors since we're not mocking
				// but the parameter validation should pass
				if err != nil {
					// Network errors are OK for parameter validation tests
					assert.NotContains(t, err.Error(), "invalid")
					assert.NotContains(t, err.Error(), "required")
				}
			}
		})
	}
}
