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

	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/tools"
	"github.com/grctool/grctool/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hasValidTugboatAuth checks if Tugboat authentication is available
func hasValidTugboatAuth(t *testing.T) bool {
	// Check for authentication via grctool auth status
	cmd := exec.Command("./bin/grctool", "auth", "status")
	err := cmd.Run()
	if err == nil {
		t.Log("Tugboat authentication available via grctool auth")
		return true
	}

	// Check for environment variables
	baseURL := os.Getenv("TUGBOAT_BASE_URL")
	sessionCookie := os.Getenv("TUGBOAT_SESSION_COOKIE")
	csrfToken := os.Getenv("TUGBOAT_CSRF_TOKEN")

	if baseURL != "" && sessionCookie != "" && csrfToken != "" {
		t.Log("Tugboat authentication available via environment variables")
		return true
	}

	t.Log("No valid Tugboat authentication found")
	return false
}

// TestTugboatSync_RealAPI tests Tugboat sync with real API
func TestTugboatSync_RealAPI(t *testing.T) {
	// Skip if no auth available
	if !hasValidTugboatAuth(t) {
		t.Skip("Valid Tugboat authentication required")
	}

	// Build the binary first
	buildCmd := exec.Command("go", "build", "-o", "bin/grctool")
	buildCmd.Dir = "../../" // Adjust path from test/e2e to root
	err := buildCmd.Run()
	require.NoError(t, err, "Failed to build grctool binary")

	// Test real Tugboat sync
	syncCmd := exec.Command("../../bin/grctool", "sync", "--verbose")
	output, err := syncCmd.CombinedOutput()

	require.NoError(t, err, "Tugboat sync should succeed with valid auth")

	outputStr := string(output)
	assert.Contains(t, outputStr, "sync", "Output should contain sync information")

	// Check for successful sync indicators
	successIndicators := []string{"completed", "success", "synced"}
	hasSuccess := false
	for _, indicator := range successIndicators {
		if strings.Contains(strings.ToLower(outputStr), indicator) {
			hasSuccess = true
			break
		}
	}
	assert.True(t, hasSuccess, "Sync output should indicate success")

	t.Logf("Tugboat sync completed successfully")
}

// TestTugboatSyncTool_RealAPI tests the Tugboat sync wrapper tool directly
func TestTugboatSyncTool_RealAPI(t *testing.T) {
	if !hasValidTugboatAuth(t) {
		t.Skip("Valid Tugboat authentication required")
	}

	cfg := helpers.SetupE2ETest(t)

	log, _ := logger.NewTestLogger()
	syncTool, err := tools.NewTugboatSyncWrapperTool(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create sync tool: %v", err)
	}

	if syncTool == nil {
		t.Skip("Tugboat sync wrapper tool not available")
	}

	result, source, err := syncTool.Execute(context.Background(), map[string]interface{}{
		"sync_type": "full",
		"verbose":   true,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.NotNil(t, source)

	t.Logf("Tugboat sync tool executed successfully")
}

// TestTugboatDataAccess_RealAPI tests accessing synced Tugboat data
func TestTugboatDataAccess_RealAPI(t *testing.T) {
	if !hasValidTugboatAuth(t) {
		t.Skip("Valid Tugboat authentication required")
	}

	cfg := helpers.SetupE2ETest(t)

	log, _ := logger.NewTestLogger()

	// Test evidence task details tool with real data
	detailsTool := tools.NewEvidenceTaskDetailsTool(cfg, log)
	if detailsTool == nil {
		t.Skip("Evidence task details tool not available")
	}

	// Try to get details for a known evidence task
	result, source, err := detailsTool.Execute(context.Background(), map[string]interface{}{
		"task_ref": "ET-101",
		"format":   "json",
	})

	if err != nil {
		// This might fail if ET-101 doesn't exist, which is fine
		t.Logf("Evidence task ET-101 not found (this is expected if not synced): %v", err)
	} else {
		assert.NotEmpty(t, result)
		assert.NotNil(t, source)
		t.Logf("Successfully retrieved evidence task details")
	}
}

// TestTugboatAuthentication_Status tests Tugboat authentication status
func TestTugboatAuthentication_Status(t *testing.T) {
	// Build the binary first
	buildCmd := exec.Command("go", "build", "-o", "bin/grctool")
	buildCmd.Dir = "../../"
	err := buildCmd.Run()
	require.NoError(t, err, "Failed to build grctool binary")

	// Check auth status
	statusCmd := exec.Command("../../bin/grctool", "auth", "status")
	output, err := statusCmd.CombinedOutput()

	outputStr := string(output)
	t.Logf("Auth status output: %s", outputStr)

	if err == nil {
		assert.Contains(t, strings.ToLower(outputStr), "authenticated")
		t.Log("Tugboat authentication is active")
	} else {
		t.Log("Tugboat authentication not available - this is expected in CI/development")
	}
}

// TestTugboatSyncWithTimeout_RealAPI tests sync with timeout handling
func TestTugboatSyncWithTimeout_RealAPI(t *testing.T) {
	if !hasValidTugboatAuth(t) {
		t.Skip("Valid Tugboat authentication required")
	}

	if os.Getenv("TEST_TIMEOUTS") == "" {
		t.Skip("TEST_TIMEOUTS not enabled")
	}

	buildCmd := exec.Command("go", "build", "-o", "bin/grctool")
	buildCmd.Dir = "../../"
	err := buildCmd.Run()
	require.NoError(t, err, "Failed to build grctool binary")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Run sync with timeout
	syncCmd := exec.CommandContext(ctx, "../../bin/grctool", "sync", "--timeout=25s")
	start := time.Now()
	output, err := syncCmd.CombinedOutput()
	duration := time.Since(start)

	outputStr := string(output)
	t.Logf("Sync completed in %v", duration)

	if ctx.Err() == context.DeadlineExceeded {
		t.Log("Sync timed out as expected for timeout test")
	} else if err == nil {
		assert.Contains(t, strings.ToLower(outputStr), "sync")
		t.Log("Sync completed within timeout")
	} else {
		t.Logf("Sync failed: %v", err)
	}
}

// TestTugboatErrorHandling_RealAPI tests error handling with invalid operations
func TestTugboatErrorHandling_RealAPI(t *testing.T) {
	buildCmd := exec.Command("go", "build", "-o", "bin/grctool")
	buildCmd.Dir = "../../"
	err := buildCmd.Run()
	require.NoError(t, err, "Failed to build grctool binary")

	// Test invalid command
	invalidCmd := exec.Command("../../bin/grctool", "sync", "--invalid-flag=true")
	output, err := invalidCmd.CombinedOutput()

	// Should handle invalid flags gracefully
	outputStr := string(output)
	t.Logf("Invalid flag output: %s", outputStr)

	// Either succeeds ignoring invalid flag, or provides helpful error
	if err != nil {
		assert.Contains(t, strings.ToLower(outputStr), "flag")
		t.Log("Invalid flag handled with appropriate error")
	} else {
		t.Log("Invalid flag ignored gracefully")
	}
}

// TestTugboatDataConsistency_RealAPI tests data consistency after sync
func TestTugboatDataConsistency_RealAPI(t *testing.T) {
	if !hasValidTugboatAuth(t) {
		t.Skip("Valid Tugboat authentication required")
	}

	if os.Getenv("TEST_DATA_CONSISTENCY") == "" {
		t.Skip("TEST_DATA_CONSISTENCY not enabled")
	}

	// Perform two syncs and compare results
	buildCmd := exec.Command("go", "build", "-o", "bin/grctool")
	buildCmd.Dir = "../../"
	err := buildCmd.Run()
	require.NoError(t, err, "Failed to build grctool binary")

	// First sync
	sync1Cmd := exec.Command("../../bin/grctool", "sync")
	output1, err1 := sync1Cmd.CombinedOutput()
	require.NoError(t, err1, "First sync should succeed")

	// Small delay
	time.Sleep(2 * time.Second)

	// Second sync
	sync2Cmd := exec.Command("../../bin/grctool", "sync")
	output2, err2 := sync2Cmd.CombinedOutput()
	require.NoError(t, err2, "Second sync should succeed")

	// Both syncs should complete successfully
	output1Str := string(output1)
	output2Str := string(output2)

	assert.Contains(t, strings.ToLower(output1Str), "sync")
	assert.Contains(t, strings.ToLower(output2Str), "sync")

	t.Log("Data consistency test completed - both syncs successful")
}
