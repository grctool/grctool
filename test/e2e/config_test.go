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
	"log"
	"os"
	"strings"
)

// E2E test configuration and environment setup
func init() {
	setupE2ELogging()
	validateE2EEnvironment()
}

// setupE2ELogging configures logging for E2E tests
func setupE2ELogging() {
	// Set log level for E2E tests
	logLevel := os.Getenv("E2E_LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	// Set grctool log level
	os.Setenv("GRCTOOL_LOG_LEVEL", logLevel)

	log.Printf("E2E test logging configured (level: %s)", logLevel)
}

// validateE2EEnvironment validates required environment variables and warns about missing ones
func validateE2EEnvironment() {
	log.Println("=== E2E Test Environment Validation ===")

	// Required environment variables for different test categories
	envVars := map[string]struct {
		required    bool
		description string
		testType    string
	}{
		// GitHub authentication
		"GITHUB_TOKEN": {
			required:    false,
			description: "Personal access token with repo permissions",
			testType:    "GitHub API tests",
		},
		"TEST_GITHUB_REPO": {
			required:    false,
			description: "Repository to test against (e.g., 'octocat/Hello-World')",
			testType:    "GitHub repository tests",
		},

		// Tugboat authentication (checked dynamically)
		"TUGBOAT_BASE_URL": {
			required:    false,
			description: "Tugboat API base URL",
			testType:    "Tugboat API tests",
		},
		"TUGBOAT_SESSION_COOKIE": {
			required:    false,
			description: "Tugboat session cookie for authentication",
			testType:    "Tugboat API tests",
		},
		"TUGBOAT_CSRF_TOKEN": {
			required:    false,
			description: "Tugboat CSRF token for authentication",
			testType:    "Tugboat API tests",
		},

		// Test enablement flags
		"TEST_MULTIPLE_REPOS": {
			required:    false,
			description: "Enable multiple repository testing",
			testType:    "Multi-repository tests",
		},
		"LARGE_REPO_TEST": {
			required:    false,
			description: "Enable large repository performance testing",
			testType:    "Performance tests",
		},
		"TEST_RATE_LIMITS": {
			required:    false,
			description: "Enable rate limit testing",
			testType:    "Rate limit tests",
		},
		"TEST_TIMEOUTS": {
			required:    false,
			description: "Enable timeout testing",
			testType:    "Timeout tests",
		},
		"TEST_CONCURRENCY": {
			required:    false,
			description: "Enable concurrency testing",
			testType:    "Concurrency tests",
		},
		"TEST_BULK_SYNC": {
			required:    false,
			description: "Enable bulk sync testing",
			testType:    "Bulk operation tests",
		},
		"TEST_MEMORY_USAGE": {
			required:    false,
			description: "Enable memory usage testing",
			testType:    "Memory usage tests",
		},
		"TEST_RATE_LIMIT_PERFORMANCE": {
			required:    false,
			description: "Enable rate limit performance testing",
			testType:    "Rate limit performance tests",
		},
		"TEST_TOOL_SCALABILITY": {
			required:    false,
			description: "Enable tool scalability testing",
			testType:    "Tool scalability tests",
		},
		"TEST_DATA_PROCESSING_SPEED": {
			required:    false,
			description: "Enable data processing speed testing",
			testType:    "Data processing tests",
		},
		"TEST_QUARTERLY_REVIEW": {
			required:    false,
			description: "Enable quarterly review workflow testing",
			testType:    "Quarterly review tests",
		},
		"TEST_DATA_CONSISTENCY": {
			required:    false,
			description: "Enable data consistency testing",
			testType:    "Data consistency tests",
		},
		"TEST_DATA_INTEGRITY": {
			required:    false,
			description: "Enable data integrity testing",
			testType:    "Data integrity tests",
		},

		// Optional test configuration
		"LARGE_TEST_REPO": {
			required:    false,
			description: "Large repository for performance testing (default: kubernetes/kubernetes)",
			testType:    "Performance tests",
		},
		"TEST_TERRAFORM_PATH": {
			required:    false,
			description: "Path to terraform files for testing",
			testType:    "Terraform tests",
		},
	}

	var missingRequired []string
	var availableOptional []string
	var enabledFlags []string

	for envVar, info := range envVars {
		value := os.Getenv(envVar)
		if value != "" {
			if strings.HasPrefix(envVar, "TEST_") && !strings.Contains(envVar, "REPO") && !strings.Contains(envVar, "PATH") {
				enabledFlags = append(enabledFlags, envVar)
			} else {
				availableOptional = append(availableOptional, envVar)
			}
			log.Printf("✓ %s: configured for %s", envVar, info.testType)
		} else {
			if info.required {
				missingRequired = append(missingRequired, envVar)
				log.Printf("✗ %s: REQUIRED for %s - %s", envVar, info.testType, info.description)
			} else {
				log.Printf("- %s: not set, %s will be skipped", envVar, info.testType)
			}
		}
	}

	// Summary
	log.Println()
	log.Println("=== E2E Test Environment Summary ===")

	if len(missingRequired) > 0 {
		log.Printf("❌ Missing required environment variables: %v", missingRequired)
		log.Println("   Some E2E tests will fail without required authentication.")
	}

	if len(availableOptional) > 0 {
		log.Printf("✅ Available authentication: %v", availableOptional)
	}

	if len(enabledFlags) > 0 {
		log.Printf("🚀 Enabled test categories: %v", enabledFlags)
	} else {
		log.Println("ℹ️  No optional test categories enabled. Set TEST_* environment variables to enable additional tests.")
	}

	// Special checks
	validateGitHubAuthentication()
	validateTugboatAuthentication()
	validateTestDataPaths()

	log.Println("=== E2E Environment Validation Complete ===")
	log.Println()
}

// validateGitHubAuthentication validates GitHub authentication setup
func validateGitHubAuthentication() {
	githubToken := os.Getenv("GITHUB_TOKEN")
	testRepo := os.Getenv("TEST_GITHUB_REPO")

	if githubToken != "" {
		log.Printf("GitHub Authentication: ✅ Token available (length: %d)", len(githubToken))

		if testRepo != "" {
			log.Printf("GitHub Repository: ✅ Test repository configured: %s", testRepo)
		} else {
			log.Printf("GitHub Repository: ⚠️  Using default repository (octocat/Hello-World)")
		}
	} else {
		log.Printf("GitHub Authentication: ❌ No token available - GitHub tests will be skipped")
		log.Println("  Set GITHUB_TOKEN environment variable with a personal access token")
		log.Println("  Token needs 'repo' permissions for full functionality")
	}
}

// validateTugboatAuthentication validates Tugboat authentication setup
func validateTugboatAuthentication() {
	baseURL := os.Getenv("TUGBOAT_BASE_URL")
	sessionCookie := os.Getenv("TUGBOAT_SESSION_COOKIE")
	csrfToken := os.Getenv("TUGBOAT_CSRF_TOKEN")

	envAuth := baseURL != "" && sessionCookie != "" && csrfToken != ""

	if envAuth {
		log.Printf("Tugboat Authentication: ✅ Environment variables configured")
	} else {
		log.Printf("Tugboat Authentication: ⚠️  Environment variables not set, checking grctool auth...")
		log.Println("  Use 'grctool auth login' or set TUGBOAT_* environment variables")
		log.Println("  Some Tugboat tests may be skipped without authentication")
	}
}

// validateTestDataPaths validates test data paths and configuration
func validateTestDataPaths() {
	terraformPath := os.Getenv("TEST_TERRAFORM_PATH")
	largeRepo := os.Getenv("LARGE_TEST_REPO")

	if terraformPath != "" {
		if _, err := os.Stat(terraformPath); err == nil {
			log.Printf("Terraform Test Data: ✅ Path exists: %s", terraformPath)
		} else {
			log.Printf("Terraform Test Data: ⚠️  Path not found: %s", terraformPath)
		}
	} else {
		log.Printf("Terraform Test Data: ℹ️  Using default test terraform directory")
	}

	if largeRepo != "" {
		log.Printf("Large Repository Test: ✅ Custom repository: %s", largeRepo)
	} else {
		log.Printf("Large Repository Test: ℹ️  Using default large repository for performance tests")
	}
}

// getTestConfiguration returns the current test configuration
func getTestConfiguration() map[string]interface{} {
	config := make(map[string]interface{})

	// Authentication status
	config["github_auth"] = os.Getenv("GITHUB_TOKEN") != ""
	config["tugboat_env_auth"] = os.Getenv("TUGBOAT_BASE_URL") != "" &&
		os.Getenv("TUGBOAT_SESSION_COOKIE") != "" &&
		os.Getenv("TUGBOAT_CSRF_TOKEN") != ""

	// Test repositories
	config["test_github_repo"] = getEnvWithDefault("TEST_GITHUB_REPO", "octocat/Hello-World")
	config["large_test_repo"] = getEnvWithDefault("LARGE_TEST_REPO", "kubernetes/kubernetes")

	// Test data paths
	config["terraform_path"] = getEnvWithDefault("TEST_TERRAFORM_PATH", "../../test_terraform")

	// Test enablement flags
	testFlags := []string{
		"TEST_MULTIPLE_REPOS", "LARGE_REPO_TEST", "TEST_RATE_LIMITS",
		"TEST_TIMEOUTS", "TEST_CONCURRENCY", "TEST_BULK_SYNC",
		"TEST_MEMORY_USAGE", "TEST_RATE_LIMIT_PERFORMANCE",
		"TEST_TOOL_SCALABILITY", "TEST_DATA_PROCESSING_SPEED",
		"TEST_QUARTERLY_REVIEW", "TEST_DATA_CONSISTENCY", "TEST_DATA_INTEGRITY",
	}

	for _, flag := range testFlags {
		config[strings.ToLower(flag)] = os.Getenv(flag) != ""
	}

	return config
}

// getEnvWithDefault returns environment variable value or default if not set
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// isTestEnabled checks if a specific test category is enabled
func isTestEnabled(testFlag string) bool {
	return os.Getenv(testFlag) != ""
}

// hasGitHubAuth checks if GitHub authentication is available
func hasGitHubAuth() bool {
	return os.Getenv("GITHUB_TOKEN") != ""
}

// hasTugboatEnvAuth checks if Tugboat environment authentication is available
func hasTugboatEnvAuth() bool {
	return os.Getenv("TUGBOAT_BASE_URL") != "" &&
		os.Getenv("TUGBOAT_SESSION_COOKIE") != "" &&
		os.Getenv("TUGBOAT_CSRF_TOKEN") != ""
}

// printTestSummary prints a summary of which tests will run
func printTestSummary() {
	log.Println("=== E2E Test Execution Summary ===")

	config := getTestConfiguration()

	// Authentication-based tests
	if config["github_auth"].(bool) {
		log.Println("✅ GitHub API tests will run")
	} else {
		log.Println("❌ GitHub API tests will be skipped (no GITHUB_TOKEN)")
	}

	if config["tugboat_env_auth"].(bool) {
		log.Println("✅ Tugboat API tests will run (env auth)")
	} else {
		log.Println("⚠️  Tugboat API tests depend on grctool auth status")
	}

	// Performance tests
	performanceTests := []string{
		"large_repo_test", "test_concurrency", "test_bulk_sync",
		"test_memory_usage", "test_tool_scalability", "test_data_processing_speed",
	}

	enabledPerformanceTests := 0
	for _, test := range performanceTests {
		if config[test].(bool) {
			enabledPerformanceTests++
		}
	}

	if enabledPerformanceTests > 0 {
		log.Printf("✅ %d performance test categories enabled", enabledPerformanceTests)
	} else {
		log.Println("ℹ️  No performance tests enabled (set TEST_* flags to enable)")
	}

	// Workflow tests
	workflowTests := []string{
		"test_quarterly_review", "test_data_consistency", "test_data_integrity",
	}

	enabledWorkflowTests := 0
	for _, test := range workflowTests {
		if config[test].(bool) {
			enabledWorkflowTests++
		}
	}

	if enabledWorkflowTests > 0 {
		log.Printf("✅ %d workflow test categories enabled", enabledWorkflowTests)
	} else {
		log.Println("ℹ️  No workflow tests enabled (set TEST_* flags to enable)")
	}

	log.Println("=================================")
}
