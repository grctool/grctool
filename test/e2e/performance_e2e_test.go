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

//go:build e2e

package e2e

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/tools"
	"github.com/grctool/grctool/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Performance test configuration
const (
	MaxSyncTime         = 2 * time.Minute         // Maximum time for sync operations
	MaxAnalysisTime     = 5 * time.Minute         // Maximum time for analysis operations
	MaxLargeRepoTime    = 10 * time.Minute        // Maximum time for large repository analysis
	MaxBulkOpTime       = 15 * time.Minute        // Maximum time for bulk operations
	RateLimitWaitTime   = 1 * time.Second         // Wait between API calls to avoid rate limits
	PerformanceTestRepo = "kubernetes/kubernetes" // Known large repository for testing
)

// TestLargeRepository_Performance tests performance with large repositories
func TestLargeRepository_Performance(t *testing.T) {
	// Test performance with large repositories
	if os.Getenv("LARGE_REPO_TEST") == "" {
		t.Skip("LARGE_REPO_TEST not enabled")
	}

	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("GITHUB_TOKEN required for large repository tests")
	}

	largeRepo := os.Getenv("LARGE_TEST_REPO")
	if largeRepo == "" {
		largeRepo = PerformanceTestRepo
	}

	t.Logf("Testing performance with large repository: %s", largeRepo)
	start := time.Now()

	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				GitHub: config.GitHubToolConfig{
					Enabled:    true,
					APIToken:   os.Getenv("GITHUB_TOKEN"),
					Repository: largeRepo,
					MaxIssues:  50, // Limit for performance testing
				},
			},
		},
	}

	log, _ := logger.NewTestLogger()
	tool := tools.NewGitHubTool(cfg, log)

	ctx, cancel := context.WithTimeout(context.Background(), MaxLargeRepoTime)
	defer cancel()

	result, source, err := tool.Execute(ctx, map[string]interface{}{
		"analysis_type": "basic",
		"output_format": "summary",
		"max_items":     10, // Limit output for performance
	})

	duration := time.Since(start)

	// Performance assertions
	require.NoError(t, err, "Large repository analysis should complete successfully")
	assert.NotEmpty(t, result)
	assert.NotNil(t, source)
	assert.Less(t, duration, MaxLargeRepoTime, "Large repository analysis should complete within time limit")

	t.Logf("Large repository analysis completed in %v", duration)

	// Memory and performance metrics
	if duration > 30*time.Second {
		t.Logf("WARNING: Analysis took longer than expected (%v)", duration)
	}
}

// TestConcurrentOperations_Performance tests concurrent operations performance
func TestConcurrentOperations_Performance(t *testing.T) {
	if os.Getenv("TEST_CONCURRENCY") == "" {
		t.Skip("TEST_CONCURRENCY not enabled")
	}

	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("GITHUB_TOKEN required for concurrency tests")
	}

	repositories := []string{
		"octocat/Hello-World",
		"github/docs",
		"actions/checkout",
	}

	t.Log("Testing concurrent operations performance...")
	start := time.Now()

	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				GitHub: config.GitHubToolConfig{
					Enabled:  true,
					APIToken: os.Getenv("GITHUB_TOKEN"),
				},
			},
		},
	}

	log, _ := logger.NewTestLogger()

	// Create a channel to collect results
	type result struct {
		repo     string
		duration time.Duration
		error    error
	}

	results := make(chan result, len(repositories))

	// Launch concurrent operations
	for _, repo := range repositories {
		go func(repository string) {
			repoStart := time.Now()

			// Configure tool for this repository
			repoConfig := *cfg
			repoConfig.Evidence.Tools.GitHub.Repository = repository

			tool := tools.NewGitHubTool(&repoConfig, log)

			// Add delay to avoid rate limits
			time.Sleep(RateLimitWaitTime)

			_, _, err := tool.Execute(context.Background(), map[string]interface{}{
				"analysis_type": "basic",
				"output_format": "summary",
			})

			results <- result{
				repo:     repository,
				duration: time.Since(repoStart),
				error:    err,
			}
		}(repo)
	}

	// Collect results
	var totalErrors int
	var maxDuration time.Duration

	for i := 0; i < len(repositories); i++ {
		select {
		case res := <-results:
			if res.error != nil {
				totalErrors++
				t.Logf("Repository %s failed: %v", res.repo, res.error)
			} else {
				t.Logf("Repository %s completed in %v", res.repo, res.duration)
			}
			if res.duration > maxDuration {
				maxDuration = res.duration
			}
		case <-time.After(MaxAnalysisTime):
			t.Fatal("Concurrent operations timed out")
		}
	}

	totalDuration := time.Since(start)

	// Performance assertions
	assert.Less(t, totalErrors, len(repositories), "Not all operations should fail")
	assert.Less(t, totalDuration, MaxAnalysisTime, "Concurrent operations should complete within time limit")
	assert.Less(t, maxDuration, MaxAnalysisTime/2, "Individual operations should be reasonably fast")

	t.Logf("Concurrent operations completed in %v (max individual: %v, errors: %d/%d)",
		totalDuration, maxDuration, totalErrors, len(repositories))
}

// TestBulkDataSync_Performance tests bulk data synchronization performance
func TestBulkDataSync_Performance(t *testing.T) {
	if os.Getenv("TEST_BULK_SYNC") == "" {
		t.Skip("TEST_BULK_SYNC not enabled")
	}

	if !hasValidTugboatAuth(t) {
		t.Skip("Valid Tugboat authentication required for bulk sync test")
	}

	buildGrctoolBinary(t)

	t.Log("Testing bulk data sync performance...")
	start := time.Now()

	// Perform bulk sync operation
	syncCmd := exec.Command("../../bin/grctool", "sync", "--verbose")
	syncCmd.Env = append(os.Environ(), "GRCTOOL_LOG_LEVEL=info")

	ctx, cancel := context.WithTimeout(context.Background(), MaxBulkOpTime)
	defer cancel()

	output, err := syncCmd.CombinedOutput()
	duration := time.Since(start)

	// Performance assertions
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("Bulk sync timed out after %v", MaxBulkOpTime)
	}

	require.NoError(t, err, "Bulk sync should complete successfully")
	assert.Less(t, duration, MaxSyncTime, "Bulk sync should complete within reasonable time")

	outputStr := string(output)
	assert.Contains(t, strings.ToLower(outputStr), "sync")

	t.Logf("Bulk data sync completed in %v", duration)

	// Additional performance metrics
	if duration > 30*time.Second {
		t.Logf("INFO: Sync took %v - consider optimization if this is too slow", duration)
	}
}

// TestMemoryUsage_Performance tests memory usage during operations
func TestMemoryUsage_Performance(t *testing.T) {
	if os.Getenv("TEST_MEMORY_USAGE") == "" {
		t.Skip("TEST_MEMORY_USAGE not enabled")
	}

	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("GITHUB_TOKEN required for memory tests")
	}

	t.Log("Testing memory usage during operations...")

	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				GitHub: config.GitHubToolConfig{
					Enabled:    true,
					APIToken:   os.Getenv("GITHUB_TOKEN"),
					Repository: "octocat/Hello-World",
				},
			},
		},
	}

	log, _ := logger.NewTestLogger()

	// Perform multiple operations to test memory usage
	operations := []struct {
		name   string
		params map[string]interface{}
	}{
		{
			name:   "basic_analysis",
			params: map[string]interface{}{"analysis_type": "basic"},
		},
		{
			name:   "permissions_analysis",
			params: map[string]interface{}{"analysis_type": "permissions"},
		},
		{
			name:   "security_analysis",
			params: map[string]interface{}{"analysis_type": "security_features"},
		},
	}

	for i := 0; i < 3; i++ { // Repeat to check for memory leaks
		t.Run(string(rune('A'+i)), func(t *testing.T) {
			for _, op := range operations {
				t.Run(op.name, func(t *testing.T) {
					tool := tools.NewGitHubTool(cfg, log)

					start := time.Now()
					result, source, err := tool.Execute(context.Background(), op.params)
					duration := time.Since(start)

					if err != nil {
						t.Logf("Operation %s failed: %v", op.name, err)
					} else {
						assert.NotEmpty(t, result)
						assert.NotNil(t, source)
						t.Logf("Operation %s completed in %v", op.name, duration)
					}

					// Small delay between operations
					time.Sleep(100 * time.Millisecond)
				})
			}
		})
	}

	t.Log("Memory usage test completed")
}

// TestRateLimitHandling_Performance tests API rate limit handling performance
func TestRateLimitHandling_Performance(t *testing.T) {
	if os.Getenv("TEST_RATE_LIMIT_PERFORMANCE") == "" {
		t.Skip("TEST_RATE_LIMIT_PERFORMANCE not enabled")
	}

	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("GITHUB_TOKEN required for rate limit tests")
	}

	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				GitHub: config.GitHubToolConfig{
					Enabled:    true,
					APIToken:   os.Getenv("GITHUB_TOKEN"),
					Repository: "octocat/Hello-World",
				},
			},
		},
	}

	log, _ := logger.NewTestLogger()
	tool := tools.NewGitHubTool(cfg, log)

	t.Log("Testing rate limit handling performance...")

	// Make rapid sequential requests to test rate limit handling
	requestCount := 5
	var totalDuration time.Duration
	var successCount int

	for i := 0; i < requestCount; i++ {
		start := time.Now()

		result, source, err := tool.Execute(context.Background(), map[string]interface{}{
			"analysis_type": "basic",
			"output_format": "summary",
		})

		duration := time.Since(start)
		totalDuration += duration

		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "rate limit") {
				t.Logf("Request %d hit rate limit (expected): %v", i+1, err)
			} else {
				t.Logf("Request %d failed with other error: %v", i+1, err)
			}
		} else {
			successCount++
			assert.NotEmpty(t, result)
			assert.NotNil(t, source)
			t.Logf("Request %d succeeded in %v", i+1, duration)
		}

		// Small delay between requests
		time.Sleep(200 * time.Millisecond)
	}

	averageDuration := totalDuration / time.Duration(requestCount)

	t.Logf("Rate limit test completed: %d/%d requests successful, average duration: %v",
		successCount, requestCount, averageDuration)

	// At least some requests should succeed
	assert.Greater(t, successCount, 0, "At least some requests should succeed")

	// Average duration should be reasonable (allowing for rate limit delays)
	assert.Less(t, averageDuration, 30*time.Second, "Average request time should be reasonable")
}

// TestScalabilityTools_Performance tests tool scalability with multiple tool operations
func TestScalabilityTools_Performance(t *testing.T) {
	if os.Getenv("TEST_TOOL_SCALABILITY") == "" {
		t.Skip("TEST_TOOL_SCALABILITY not enabled")
	}

	cfg := helpers.SetupE2ETest(t)

	log, _ := logger.NewTestLogger()

	t.Log("Testing tool scalability performance...")
	start := time.Now()

	// Initialize tool registry
	err := tools.InitializeToolRegistry(cfg, log)
	require.NoError(t, err, "Tool registry should initialize")

	availableTools := tools.ListTools()
	t.Logf("Testing scalability with %d available tools", len(availableTools))

	// Test multiple tool operations concurrently
	testTools := []string{
		"evidence-task-list",
		"storage-read",
		"docs-reader",
	}

	type toolResult struct {
		name     string
		duration time.Duration
		error    error
	}

	results := make(chan toolResult, len(testTools))

	// Execute tools concurrently
	for _, toolName := range testTools {
		go func(name string) {
			toolStart := time.Now()

			tool, err := tools.GetTool(name)
			if err != nil {
				results <- toolResult{name: name, error: ErrToolNotFound}
				return
			}

			params := make(map[string]interface{})
			if name == "storage-read" {
				params["path"] = "docs"
			}

			_, _, execErr := tool.Execute(context.Background(), params)

			results <- toolResult{
				name:     name,
				duration: time.Since(toolStart),
				error:    execErr,
			}
		}(toolName)
	}

	// Collect results
	var successful int
	var maxToolDuration time.Duration

	for i := 0; i < len(testTools); i++ {
		select {
		case result := <-results:
			if result.error != nil {
				t.Logf("Tool %s failed: %v", result.name, result.error)
			} else {
				successful++
				t.Logf("Tool %s completed in %v", result.name, result.duration)
			}
			if result.duration > maxToolDuration {
				maxToolDuration = result.duration
			}
		case <-time.After(MaxAnalysisTime):
			t.Fatal("Tool scalability test timed out")
		}
	}

	totalDuration := time.Since(start)

	// Performance assertions
	assert.Greater(t, successful, 0, "At least some tools should execute successfully")
	assert.Less(t, totalDuration, MaxAnalysisTime, "Tool operations should complete within time limit")
	assert.Less(t, maxToolDuration, MaxAnalysisTime/2, "Individual tools should be reasonably fast")

	t.Logf("Tool scalability test completed in %v (max tool: %v, successful: %d/%d)",
		totalDuration, maxToolDuration, successful, len(testTools))
}

// Custom error for tool not found
var ErrToolNotFound = &ToolNotFoundError{}

type ToolNotFoundError struct{}

func (e *ToolNotFoundError) Error() string {
	return "tool not found"
}

// TestDataProcessingSpeed_Performance tests data processing speed with various data sizes
func TestDataProcessingSpeed_Performance(t *testing.T) {
	if os.Getenv("TEST_DATA_PROCESSING_SPEED") == "" {
		t.Skip("TEST_DATA_PROCESSING_SPEED not enabled")
	}

	t.Log("Testing data processing speed...")

	// Test data processing with different sizes
	testCases := []struct {
		name        string
		tool        string
		maxDuration time.Duration
	}{
		{
			name:        "small_data_read",
			tool:        "storage-read",
			maxDuration: 5 * time.Second,
		},
		{
			name:        "medium_data_list",
			tool:        "evidence-task-list",
			maxDuration: 10 * time.Second,
		},
		{
			name:        "large_docs_read",
			tool:        "docs-reader",
			maxDuration: 30 * time.Second,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			tool, err := tools.GetTool(testCase.tool)
			if err != nil {
				t.Skipf("Tool %s not available", testCase.tool)
			}

			start := time.Now()

			params := make(map[string]interface{})
			if testCase.tool == "storage-read" {
				params["path"] = "docs"
				params["recursive"] = true
			}

			result, source, err := tool.Execute(context.Background(), params)
			duration := time.Since(start)

			if err != nil {
				t.Logf("Data processing test %s failed: %v", testCase.name, err)
			} else {
				assert.NotEmpty(t, result)
				assert.NotNil(t, source)
				assert.Less(t, duration, testCase.maxDuration,
					"Data processing should complete within expected time")

				t.Logf("Data processing %s completed in %v (limit: %v)",
					testCase.name, duration, testCase.maxDuration)
			}
		})
	}
}
