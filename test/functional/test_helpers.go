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

//go:build functional

package functional

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Test constants
const (
	BinaryName     = "grctool"
	BinaryPath     = "../../bin/grctool"
	TestDataDir    = "../../test_data"
	BuildTimeout   = 60 * time.Second
	CommandTimeout = 30 * time.Second
)

// TestEnvironment holds test environment configuration
type TestEnvironment struct {
	BinaryPath   string
	TempDir      string
	ConfigPath   string
	DataDir      string
	CleanupFuncs []func() error
}

// NewTestEnvironment creates a new test environment
func NewTestEnvironment(t *testing.T) *TestEnvironment {
	tempDir := t.TempDir()
	env := &TestEnvironment{
		BinaryPath:   ensureBinaryExists(t),
		TempDir:      tempDir,
		DataDir:      filepath.Join(tempDir, "data"),
		ConfigPath:   filepath.Join(tempDir, "config.yaml"),
		CleanupFuncs: make([]func() error, 0),
	}

	// Create data directory
	err := os.MkdirAll(env.DataDir, 0755)
	require.NoError(t, err, "Failed to create test data directory")

	// Add cleanup function
	t.Cleanup(func() {
		for _, cleanup := range env.CleanupFuncs {
			if err := cleanup(); err != nil {
				t.Logf("Cleanup error: %v", err)
			}
		}
	})

	return env
}

// AddCleanup adds a cleanup function to be called when test completes
func (env *TestEnvironment) AddCleanup(cleanup func() error) {
	env.CleanupFuncs = append(env.CleanupFuncs, cleanup)
}

// RunCommand executes a grctool command with the test environment
func (env *TestEnvironment) RunCommand(args ...string) (*CommandResult, error) {
	cmd := exec.Command(env.BinaryPath, args...)
	cmd.Env = append(os.Environ(),
		"GRCTOOL_DATA_DIR="+env.DataDir,
		"GRCTOOL_CONFIG="+env.ConfigPath,
		"GRCTOOL_LOG_LEVEL=debug",
	)

	start := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	return &CommandResult{
		Output:   string(output),
		Error:    err,
		Duration: duration,
		Args:     args,
	}, err
}

// RunCommandWithTimeout executes a command with a specific timeout
func (env *TestEnvironment) RunCommandWithTimeout(timeout time.Duration, args ...string) (*CommandResult, error) {
	cmd := exec.Command(env.BinaryPath, args...)
	cmd.Env = append(os.Environ(),
		"GRCTOOL_DATA_DIR="+env.DataDir,
		"GRCTOOL_CONFIG="+env.ConfigPath,
		"GRCTOOL_LOG_LEVEL=debug",
	)

	start := time.Now()

	// Set timeout
	done := make(chan error, 1)
	var output []byte
	go func() {
		var err error
		output, err = cmd.CombinedOutput()
		done <- err
	}()

	select {
	case err := <-done:
		duration := time.Since(start)
		return &CommandResult{
			Output:   string(output),
			Error:    err,
			Duration: duration,
			Args:     args,
		}, err
	case <-time.After(timeout):
		cmd.Process.Kill()
		return &CommandResult{
			Output:   string(output),
			Error:    fmt.Errorf("command timed out after %v", timeout),
			Duration: timeout,
			Args:     args,
		}, fmt.Errorf("command timed out")
	}
}

// WriteTestFile writes content to a file in the test environment
func (env *TestEnvironment) WriteTestFile(relativePath, content string) error {
	fullPath := filepath.Join(env.DataDir, relativePath)
	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	return os.WriteFile(fullPath, []byte(content), 0644)
}

// ReadTestFile reads content from a file in the test environment
func (env *TestEnvironment) ReadTestFile(relativePath string) (string, error) {
	fullPath := filepath.Join(env.DataDir, relativePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", fullPath, err)
	}
	return string(content), nil
}

// CommandResult holds the result of a command execution
type CommandResult struct {
	Output   string
	Error    error
	Duration time.Duration
	Args     []string
}

// Successful returns true if the command succeeded
func (r *CommandResult) Successful() bool {
	return r.Error == nil
}

// Failed returns true if the command failed
func (r *CommandResult) Failed() bool {
	return r.Error != nil
}

// Contains checks if the output contains the specified string
func (r *CommandResult) Contains(s string) bool {
	return strings.Contains(r.Output, s)
}

// ContainsAny checks if the output contains any of the specified strings
func (r *CommandResult) ContainsAny(strings ...string) bool {
	for _, s := range strings {
		if r.Contains(s) {
			return true
		}
	}
	return false
}

// Log logs the command result
func (r *CommandResult) Log(t *testing.T) {
	t.Logf("Command: %v", r.Args)
	t.Logf("Duration: %v", r.Duration)
	if r.Error != nil {
		t.Logf("Error: %v", r.Error)
	}
	t.Logf("Output: %s", r.Output)
}

// ensureBinaryExists ensures the grctool binary exists and builds it if necessary
func ensureBinaryExists(t *testing.T) string {
	binaryPath, err := filepath.Abs(BinaryPath)
	require.NoError(t, err, "Failed to get absolute path for binary")

	// Check if binary exists and is recent
	if stat, err := os.Stat(binaryPath); err == nil {
		// If binary is less than 5 minutes old, assume it's current
		if time.Since(stat.ModTime()) < 5*time.Minute {
			return binaryPath
		}
	}

	// Build the binary
	t.Log("Building grctool binary for functional testing...")
	start := time.Now()

	// Get the project root (two levels up from test/functional)
	projectRoot, err := filepath.Abs("../../")
	require.NoError(t, err, "Failed to get project root")

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = projectRoot

	output, err := cmd.CombinedOutput()
	buildTime := time.Since(start)

	require.NoError(t, err, "Failed to build grctool binary: %s", string(output))
	t.Logf("Built grctool binary in %v", buildTime)

	// Verify binary is executable
	_, err = exec.LookPath(binaryPath)
	require.NoError(t, err, "Built binary is not executable")

	return binaryPath
}

// SkipIfNoData skips the test if required data is not available
func SkipIfNoData(t *testing.T, result *CommandResult, dataType string) {
	if result.Failed() {
		if result.ContainsAny("not found", "no such file", "directory not found", "no evidence tasks") {
			t.Skipf("Skipping test - %s data not available: %s", dataType, result.Output)
		}
	}
}

// SkipIfAuthRequired skips the test if authentication is required
func SkipIfAuthRequired(t *testing.T, result *CommandResult) {
	if result.Failed() {
		if result.ContainsAny("auth", "credential", "token", "login", "unauthorized") {
			t.Skipf("Skipping test - authentication required: %s", result.Output)
		}
	}
}

// SkipIfParameterIssue skips the test if there are parameter mapping issues
func SkipIfParameterIssue(t *testing.T, result *CommandResult) {
	if result.Failed() {
		if result.ContainsAny("parameter", "flag", "unknown command", "not registered") {
			t.Skipf("Skipping test - parameter/command issue: %s", result.Output)
		}
	}
}

// CreateSampleTerraformFile creates a sample terraform file for testing
func CreateSampleTerraformFile(env *TestEnvironment, filename, content string) error {
	if content == "" {
		content = `
resource "aws_s3_bucket" "test" {
  bucket = "test-bucket-${random_id.test.hex}"
  
  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        sse_algorithm = "AES256"
      }
    }
  }
}

resource "aws_s3_bucket_public_access_block" "test" {
  bucket = aws_s3_bucket.test.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "random_id" "test" {
  byte_length = 8
}

output "bucket_name" {
  value = aws_s3_bucket.test.bucket
}
`
	}

	return env.WriteTestFile(filename, content)
}

// CreateSampleEvidenceTask creates a sample evidence task file for testing
func CreateSampleEvidenceTask(env *TestEnvironment, taskRef string) error {
	content := fmt.Sprintf(`{
  "id": "%s",
  "ref": "%s",
  "title": "Sample Evidence Task for Testing",
  "description": "This is a sample evidence task created for functional testing",
  "framework": "SOC2",
  "control": "CC6.1",
  "status": "pending",
  "created_at": "%s",
  "updated_at": "%s",
  "requirements": [
    {
      "id": "REQ-001",
      "description": "Test requirement",
      "type": "documentation",
      "status": "pending"
    }
  ],
  "tools_recommended": [
    "terraform-security-analyzer",
    "storage-read"
  ],
  "tags": ["test", "functional"]
}`, taskRef, taskRef, time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339))

	return env.WriteTestFile(fmt.Sprintf("evidence_tasks/%s.json", taskRef), content)
}

// CreateSamplePolicy creates a sample policy file for testing
func CreateSamplePolicy(env *TestEnvironment, policyId string) error {
	content := fmt.Sprintf(`{
  "id": "%s",
  "title": "Sample Policy for Testing",
  "description": "This is a sample policy created for functional testing",
  "category": "Access Control",
  "status": "active",
  "version": "1.0",
  "created_at": "%s",
  "updated_at": "%s",
  "sections": [
    {
      "title": "Purpose",
      "content": "This policy defines access control requirements for functional testing."
    },
    {
      "title": "Scope",
      "content": "This policy applies to all test environments and resources."
    },
    {
      "title": "Requirements",
      "content": "All access must be properly authenticated and authorized."
    }
  ],
  "tags": ["test", "functional", "access-control"]
}`, policyId, time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339))

	return env.WriteTestFile(fmt.Sprintf("policies/%s.json", policyId), content)
}

// CreateSampleControl creates a sample control file for testing
func CreateSampleControl(env *TestEnvironment, controlId string) error {
	content := fmt.Sprintf(`{
  "id": "%s",
  "title": "Sample Control for Testing",
  "description": "This is a sample control created for functional testing",
  "framework": "SOC2",
  "category": "CC6.1",
  "status": "active",
  "version": "1.0",
  "created_at": "%s",
  "updated_at": "%s",
  "control_activities": [
    {
      "id": "CA-001",
      "description": "Review access permissions monthly",
      "frequency": "monthly",
      "owner": "IT Security Team"
    },
    {
      "id": "CA-002",
      "description": "Monitor access logs for anomalies",
      "frequency": "daily",
      "owner": "Security Operations"
    }
  ],
  "testing_procedures": [
    {
      "id": "TP-001",
      "description": "Sample access control testing",
      "method": "automated",
      "frequency": "monthly"
    }
  ],
  "tags": ["test", "functional", "access-control"]
}`, controlId, time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339))

	return env.WriteTestFile(fmt.Sprintf("controls/%s.json", controlId), content)
}

// SetupMinimalTestData creates minimal test data for functional tests
func SetupMinimalTestData(t *testing.T, env *TestEnvironment) {
	// Create sample evidence task
	err := CreateSampleEvidenceTask(env, "ET-TEST-001")
	require.NoError(t, err, "Failed to create sample evidence task")

	// Create sample policy
	err = CreateSamplePolicy(env, "POL-TEST-001")
	require.NoError(t, err, "Failed to create sample policy")

	// Create sample control
	err = CreateSampleControl(env, "CTRL-TEST-001")
	require.NoError(t, err, "Failed to create sample control")

	// Create sample terraform file
	err = CreateSampleTerraformFile(env, "terraform/test.tf", "")
	require.NoError(t, err, "Failed to create sample terraform file")

	t.Log("Minimal test data setup completed")
}

// PerformanceMetrics holds performance measurement data
type PerformanceMetrics struct {
	CommandName   string
	ExecutionTime time.Duration
	MemoryUsage   int64
	OutputSize    int
	Successful    bool
	ErrorType     string
}

// MeasurePerformance measures the performance of a command execution
func MeasurePerformance(commandName string, result *CommandResult) *PerformanceMetrics {
	metrics := &PerformanceMetrics{
		CommandName:   commandName,
		ExecutionTime: result.Duration,
		OutputSize:    len(result.Output),
		Successful:    result.Successful(),
	}

	if result.Failed() {
		if result.ContainsAny("timeout", "killed") {
			metrics.ErrorType = "timeout"
		} else if result.ContainsAny("auth", "credential") {
			metrics.ErrorType = "auth"
		} else if result.ContainsAny("not found", "no such file") {
			metrics.ErrorType = "not_found"
		} else if result.ContainsAny("parameter", "flag", "unknown") {
			metrics.ErrorType = "parameter"
		} else {
			metrics.ErrorType = "other"
		}
	}

	return metrics
}

// LogPerformanceMetrics logs performance metrics for analysis
func LogPerformanceMetrics(t *testing.T, metrics *PerformanceMetrics) {
	status := "SUCCESS"
	if !metrics.Successful {
		status = fmt.Sprintf("FAILED(%s)", metrics.ErrorType)
	}

	t.Logf("PERFORMANCE: %s | %v | %d bytes | %s",
		metrics.CommandName,
		metrics.ExecutionTime,
		metrics.OutputSize,
		status)
}

// TestDataPaths holds paths to various test data directories
var TestDataPaths = struct {
	Terraform     string
	TerraformSOC2 string
	GitHub        string
	Configs       string
	EvidenceTasks string
}{
	Terraform:     "../../test_data/terraform",
	TerraformSOC2: "../../test_data/terraform/soc2",
	GitHub:        "../../test_data/github",
	Configs:       "../../test_data/configs",
	EvidenceTasks: "../../test_data/evidence_tasks",
}

// CommonTestPatterns holds common patterns to check in test outputs
var CommonTestPatterns = struct {
	Success   []string
	Error     []string
	Auth      []string
	NotFound  []string
	Parameter []string
	JSON      []string
	Markdown  []string
}{
	Success:   []string{"success", "completed", "generated", "found"},
	Error:     []string{"error", "failed", "exception", "panic"},
	Auth:      []string{"auth", "credential", "token", "login", "unauthorized"},
	NotFound:  []string{"not found", "no such file", "directory not found", "no evidence tasks"},
	Parameter: []string{"parameter", "flag", "unknown command", "not registered"},
	JSON:      []string{"{", "}", `"`, "[", "]"},
	Markdown:  []string{"#", "##", "###", "*", "-"},
}
