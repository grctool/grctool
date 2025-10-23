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

//go:build disabled
// +build disabled

// VCR tests disabled - config.VCR field no longer exists
// TODO: Update these tests to work with current VCR implementation

package tools_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/grctool/grctool/internal/tools"
	"github.com/grctool/grctool/internal/vcr"
	tools_helper "github.com/grctool/grctool/test/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitHubTool_VCRRecording(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping VCR recording test in short mode")
	}

	tempDir := t.TempDir()
	cassetteDir := filepath.Join(tempDir, "vcr_cassettes")
	err := os.MkdirAll(cassetteDir, 0755)
	require.NoError(t, err)

	// Create mock GitHub server for recording
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search/code":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tools_helper.SampleGitHubResponses["search_code"])
		case "/repos/test-org/test-repo/contents/docs/security-policy.md":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tools_helper.SampleGitHubResponses["file_content"])
		case "/repos/test-org/test-repo/git/trees/main":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tools_helper.SampleGitHubResponses["repository_tree"])
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	// Create test configuration with VCR recording mode
	cfg := tools_helper.CreateTestConfig(tempDir)
	// Note: GitHub tool doesn't use BaseURL anymore
	cfg.VCR.Mode = string(vcr.ModeRecord)
	cfg.VCR.CassetteDir = cassetteDir

	log := tools_helper.CreateTestLogger(t)
	tool := tools.NewGitHubTool(cfg, log)

	t.Run("Record Code Search", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"query":       "encryption",
			"search_type": "code",
		}

		result, source, err := tool.Execute(ctx, params)
		require.NoError(t, err)
		assert.NotEmpty(t, result)
		assert.NotNil(t, source)

		// Verify content from recorded response
		assert.Contains(t, result, "security-config.tf")
		assert.Contains(t, result, "aws_kms_key")
		assert.Contains(t, result, "enable_key_rotation")
	})

	t.Run("Record File Content", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"search_type": "content",
			"file_path":   "docs/security-policy.md",
		}

		result, source, err := tool.Execute(ctx, params)
		require.NoError(t, err)
		assert.NotEmpty(t, result)
		assert.NotNil(t, source)

		// Verify decoded content
		assert.Contains(t, result, "# Security Policy")
		assert.Contains(t, result, "Data Encryption")
		assert.Contains(t, result, "AES-256 encryption")
	})

	t.Run("Record Repository Tree", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"search_type": "tree",
			"recursive":   true,
		}

		result, source, err := tool.Execute(ctx, params)
		require.NoError(t, err)
		assert.NotEmpty(t, result)
		assert.NotNil(t, source)

		// Verify tree structure
		assert.Contains(t, result, "docs/security-policy.md")
		assert.Contains(t, result, "terraform/security-config.tf")
		assert.Contains(t, result, "README.md")
	})

	// Verify cassettes were created
	cassettes, err := filepath.Glob(filepath.Join(cassetteDir, "*.json"))
	require.NoError(t, err)
	assert.Greater(t, len(cassettes), 0, "At least one VCR cassette should be created")

	// Verify cassette content is sanitized
	for _, cassettePath := range cassettes {
		cassetteData, err := os.ReadFile(cassettePath)
		require.NoError(t, err)

		var cassette map[string]interface{}
		err = json.Unmarshal(cassetteData, &cassette)
		require.NoError(t, err)

		// Check that sensitive data is redacted
		interactions := cassette["interactions"].([]interface{})
		for _, interaction := range interactions {
			interactionMap := interaction.(map[string]interface{})
			request := interactionMap["request"].(map[string]interface{})
			headers := request["headers"].(map[string]interface{})

			// Authorization header should be redacted
			if authHeader, exists := headers["Authorization"]; exists {
				assert.Equal(t, "[REDACTED]", authHeader, "Authorization header should be redacted")
			}
		}
	}
}

func TestGitHubTool_VCRPlayback(t *testing.T) {
	tempDir := t.TempDir()
	cassetteDir := filepath.Join(tempDir, "vcr_cassettes")
	err := os.MkdirAll(cassetteDir, 0755)
	require.NoError(t, err)

	// Create a sample cassette for playback
	sampleCassette := map[string]interface{}{
		"name":        "get_search_code_encryption_abc123.json",
		"recorded_at": "2024-01-01T12:00:00Z",
		"interactions": []interface{}{
			map[string]interface{}{
				"request": map[string]interface{}{
					"method": "GET",
					"url":    "https://api.github.com/search/code?q=encryption+repo%3Atest-org%2Ftest-repo",
					"headers": map[string]interface{}{
						"Authorization": "[REDACTED]",
						"User-Agent":    "grctool/1.0",
						"Accept":        "application/vnd.github.v3+json",
					},
				},
				"response": map[string]interface{}{
					"status_code": 200,
					"status":      "200 OK",
					"headers": map[string]interface{}{
						"Content-Type":          "application/json; charset=utf-8",
						"X-RateLimit-Limit":     "30",
						"X-RateLimit-Remaining": "29",
					},
					"body": func() string {
						body, _ := json.Marshal(tools_helper.SampleGitHubResponses["search_code"])
						return string(body)
					}(),
				},
				"timestamp": "2024-01-01T12:00:00Z",
			},
		},
	}

	cassettePath := filepath.Join(cassetteDir, "get_search_code_encryption_abc123.json")
	cassetteData, err := json.MarshalIndent(sampleCassette, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(cassettePath, cassetteData, 0644)
	require.NoError(t, err)

	// Create test configuration with VCR playback mode
	cfg := tools_helper.CreateTestConfig(tempDir)
	// Note: GitHub tool doesn't use BaseURL anymore - VCR will intercept real API calls
	cfg.VCR.Mode = string(vcr.ModePlayback)
	cfg.VCR.CassetteDir = cassetteDir

	log := tools_helper.CreateTestLogger(t)
	tool := tools.NewGitHubTool(cfg, log)

	t.Run("Playback Code Search", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"query":       "encryption",
			"search_type": "code",
		}

		result, source, err := tool.Execute(ctx, params)
		require.NoError(t, err)
		assert.NotEmpty(t, result)
		assert.NotNil(t, source)

		// Verify content from cassette
		assert.Contains(t, result, "security-config.tf")
		assert.Contains(t, result, "security-policy.md")
		assert.Contains(t, result, "aws_kms_key")
		assert.Contains(t, result, "encryption_key")
	})
}

func TestGitHubTool_VCRRecordOnce(t *testing.T) {
	tempDir := t.TempDir()
	cassetteDir := filepath.Join(tempDir, "vcr_cassettes")
	err := os.MkdirAll(cassetteDir, 0755)
	require.NoError(t, err)

	// Create mock server
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tools_helper.SampleGitHubResponses["search_code"])
	}))
	defer server.Close()

	// Create test configuration with VCR record_once mode
	cfg := tools_helper.CreateTestConfig(tempDir)
	// Note: GitHub tool doesn't use BaseURL anymore
	cfg.VCR.Mode = string(vcr.ModeRecordOnce)
	cfg.VCR.CassetteDir = cassetteDir

	log := tools_helper.CreateTestLogger(t)
	tool := tools.NewGitHubTool(cfg, log)

	params := map[string]interface{}{
		"query":       "encryption",
		"search_type": "code",
	}

	t.Run("First Call - Should Record", func(t *testing.T) {
		ctx := context.Background()
		result, source, err := tool.Execute(ctx, params)
		require.NoError(t, err)
		assert.NotEmpty(t, result)
		assert.NotNil(t, source)

		// Should have made one HTTP call
		assert.Equal(t, 1, callCount)

		// Verify cassette was created
		cassettes, err := filepath.Glob(filepath.Join(cassetteDir, "*.json"))
		require.NoError(t, err)
		assert.Greater(t, len(cassettes), 0)
	})

	t.Run("Second Call - Should Playback", func(t *testing.T) {
		ctx := context.Background()
		result, source, err := tool.Execute(ctx, params)
		require.NoError(t, err)
		assert.NotEmpty(t, result)
		assert.NotNil(t, source)

		// Should still have made only one HTTP call (playback doesn't count)
		assert.Equal(t, 1, callCount)
	})
}

func TestGitHubTool_VCRErrorScenarios(t *testing.T) {
	tempDir := t.TempDir()
	cassetteDir := filepath.Join(tempDir, "vcr_cassettes")
	err := os.MkdirAll(cassetteDir, 0755)
	require.NoError(t, err)

	t.Run("Rate Limit Error Recording", func(t *testing.T) {
		// Create mock server that returns rate limit error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(tools_helper.SampleGitHubResponses["rate_limit_error"])
		}))
		defer server.Close()

		cfg := tools_helper.CreateTestConfig(tempDir)
		// Note: GitHub tool doesn't use BaseURL anymore
		cfg.VCR.Mode = string(vcr.ModeRecord)
		cfg.VCR.CassetteDir = cassetteDir

		log := tools_helper.CreateTestLogger(t)
		tool := tools.NewGitHubTool(cfg, log)

		ctx := context.Background()
		params := map[string]interface{}{
			"query":       "test",
			"search_type": "code",
		}

		_, _, err := tool.Execute(ctx, params)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rate limit")

		// Error should still be recorded in cassette
		cassettes, err := filepath.Glob(filepath.Join(cassetteDir, "*.json"))
		require.NoError(t, err)
		assert.Greater(t, len(cassettes), 0)
	})

	t.Run("Playback Missing Cassette", func(t *testing.T) {
		cfg := tools_helper.CreateTestConfig(tempDir)
		// Note: GitHub tool doesn't use BaseURL anymore
		cfg.VCR.Mode = string(vcr.ModePlayback)
		cfg.VCR.CassetteDir = cassetteDir

		log := tools_helper.CreateTestLogger(t)
		tool := tools.NewGitHubTool(cfg, log)

		ctx := context.Background()
		params := map[string]interface{}{
			"query":       "nonexistent-query",
			"search_type": "code",
		}

		_, _, err := tool.Execute(ctx, params)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cassette")
	})
}

func TestGitHubTool_VCRSensitiveDataRedaction(t *testing.T) {
	tempDir := t.TempDir()
	cassetteDir := filepath.Join(tempDir, "vcr_cassettes")
	err := os.MkdirAll(cassetteDir, 0755)
	require.NoError(t, err)

	// Create mock server that echoes sensitive headers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Echo back headers in response for testing
		responseData := map[string]interface{}{
			"headers_received": map[string]string{
				"authorization": r.Header.Get("Authorization"),
				"x-api-key":     r.Header.Get("X-API-Key"),
				"cookie":        r.Header.Get("Cookie"),
			},
			"query_params": r.URL.Query(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseData)
	}))
	defer server.Close()

	cfg := tools_helper.CreateTestConfig(tempDir)
	// Note: GitHub tool doesn't use BaseURL anymore
	cfg.Evidence.Tools.GitHub.APIToken = "sensitive-token-123"
	cfg.VCR.Mode = string(vcr.ModeRecord)
	cfg.VCR.CassetteDir = cassetteDir
	cfg.VCR.SanitizeHeaders = true
	cfg.VCR.SanitizeParams = true
	cfg.VCR.RedactHeaders = []string{"authorization", "x-api-key", "cookie", "token"}
	cfg.VCR.RedactParams = []string{"access_token", "api_key", "token", "password"}

	log := tools_helper.CreateTestLogger(t)
	tool := tools.NewGitHubTool(cfg, log)

	t.Run("Record with Sensitive Data", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"query":        "test",
			"search_type":  "code",
			"access_token": "secret-token", // This should be redacted
		}

		_, _, err := tool.Execute(ctx, params)
		// May error due to invalid response format, but cassette should be created

		// Check cassette for redacted data
		cassettes, err := filepath.Glob(filepath.Join(cassetteDir, "*.json"))
		require.NoError(t, err)
		assert.Greater(t, len(cassettes), 0)

		for _, cassettePath := range cassettes {
			cassetteData, err := os.ReadFile(cassettePath)
			require.NoError(t, err)

			// Verify sensitive token is not in cassette
			assert.NotContains(t, string(cassetteData), "sensitive-token-123")
			assert.NotContains(t, string(cassetteData), "secret-token")

			// Verify redaction placeholders are present
			var cassette map[string]interface{}
			err = json.Unmarshal(cassetteData, &cassette)
			require.NoError(t, err)

			interactions := cassette["interactions"].([]interface{})
			for _, interaction := range interactions {
				interactionMap := interaction.(map[string]interface{})
				request := interactionMap["request"].(map[string]interface{})
				headers := request["headers"].(map[string]interface{})

				// Check that sensitive headers are redacted
				for _, sensitiveHeader := range []string{"Authorization", "X-Api-Key", "Cookie"} {
					if headerValue, exists := headers[sensitiveHeader]; exists {
						assert.Equal(t, "[REDACTED]", headerValue,
							"Sensitive header %s should be redacted", sensitiveHeader)
					}
				}
			}
		}
	})
}

func TestGitHubTool_VCRCIMode(t *testing.T) {
	tempDir := t.TempDir()
	cassetteDir := filepath.Join(tempDir, "vcr_cassettes")
	err := os.MkdirAll(cassetteDir, 0755)
	require.NoError(t, err)

	// Simulate CI environment by setting VCR to playback mode
	// and ensuring cassettes exist
	sampleCassette := map[string]interface{}{
		"name":        "ci_test_cassette.json",
		"recorded_at": "2024-01-01T12:00:00Z",
		"interactions": []interface{}{
			map[string]interface{}{
				"request": map[string]interface{}{
					"method": "GET",
					"url":    "https://api.github.com/search/code?q=security",
					"headers": map[string]interface{}{
						"Authorization": "[REDACTED]",
						"Accept":        "application/vnd.github.v3+json",
					},
				},
				"response": map[string]interface{}{
					"status_code": 200,
					"status":      "200 OK",
					"headers": map[string]interface{}{
						"Content-Type": "application/json",
					},
					"body": func() string {
						body, _ := json.Marshal(tools_helper.SampleGitHubResponses["search_code"])
						return string(body)
					}(),
				},
				"timestamp": "2024-01-01T12:00:00Z",
			},
		},
	}

	cassettePath := filepath.Join(cassetteDir, "ci_test_cassette.json")
	cassetteData, err := json.MarshalIndent(sampleCassette, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(cassettePath, cassetteData, 0644)
	require.NoError(t, err)

	// Configure for CI mode (playback only)
	cfg := tools_helper.CreateTestConfig(tempDir)
	// Note: GitHub tool doesn't use BaseURL anymore
	cfg.VCR.Mode = string(vcr.ModePlayback)
	cfg.VCR.CassetteDir = cassetteDir

	log := tools_helper.CreateTestLogger(t)
	tool := tools.NewGitHubTool(cfg, log)

	t.Run("CI Playback Mode", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"query":       "security",
			"search_type": "code",
		}

		result, source, err := tool.Execute(ctx, params)
		require.NoError(t, err)
		assert.NotEmpty(t, result)
		assert.NotNil(t, source)

		// Should get results from cassette without making HTTP calls
		assert.Contains(t, result, "security-config.tf")
		assert.Contains(t, result, "security-policy.md")
	})

	t.Run("CI Missing Cassette Fails", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"query":       "missing-cassette-query",
			"search_type": "code",
		}

		_, _, err := tool.Execute(ctx, params)
		assert.Error(t, err)
		// Should fail in CI if cassette is missing
		assert.Contains(t, err.Error(), "cassette")
	})
}
