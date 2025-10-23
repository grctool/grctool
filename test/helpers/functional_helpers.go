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

package helpers

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Functional test helpers - CLI testing
func BuildTestBinary(t *testing.T) string {
	t.Helper()

	cmd := exec.Command("go", "build", "-o", "bin/grctool", ".")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build test binary: %v\nStderr: %s", err, stderr.String())
	}

	return "./bin/grctool"
}

func RunCLICommand(t *testing.T, args ...string) ([]byte, error) {
	t.Helper()

	binaryPath := BuildTestBinary(t)
	cmd := exec.Command(binaryPath, args...)
	return cmd.Output()
}

func RunCLICommandWithInput(t *testing.T, input string, args ...string) ([]byte, error) {
	t.Helper()

	binaryPath := BuildTestBinary(t)
	cmd := exec.Command(binaryPath, args...)
	cmd.Stdin = strings.NewReader(input)
	return cmd.Output()
}

func RunCLICommandWithEnv(t *testing.T, env map[string]string, args ...string) ([]byte, error) {
	t.Helper()

	binaryPath := BuildTestBinary(t)
	cmd := exec.Command(binaryPath, args...)

	// Set environment variables
	for key, value := range env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}

	return cmd.Output()
}

// CLI command result helpers
type CLIResult struct {
	Output   []byte
	ExitCode int
	Error    error
}

func RunCLICommandFull(t *testing.T, args ...string) *CLIResult {
	t.Helper()

	binaryPath := BuildTestBinary(t)
	cmd := exec.Command(binaryPath, args...)

	output, err := cmd.Output()
	exitCode := 0

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
	}

	return &CLIResult{
		Output:   output,
		ExitCode: exitCode,
		Error:    err,
	}
}

// Assertion helpers for functional tests
func AssertCLISuccess(t *testing.T, result *CLIResult) {
	t.Helper()
	require.Equal(t, 0, result.ExitCode, "CLI command should succeed")
	require.NoError(t, result.Error, "CLI command should not error")
}

func AssertCLIFailure(t *testing.T, result *CLIResult, expectedExitCode int) {
	t.Helper()
	require.Equal(t, expectedExitCode, result.ExitCode, "CLI command should fail with expected exit code")
}

func AssertCLIOutputContains(t *testing.T, result *CLIResult, expected string) {
	t.Helper()
	output := string(result.Output)
	require.Contains(t, output, expected, "CLI output should contain expected text")
}

func AssertCLIOutputNotContains(t *testing.T, result *CLIResult, unexpected string) {
	t.Helper()
	output := string(result.Output)
	require.NotContains(t, output, unexpected, "CLI output should not contain unexpected text")
}

// Common CLI test scenarios
func TestCLIVersion(t *testing.T) {
	result := RunCLICommandFull(t, "version")
	AssertCLISuccess(t, result)
	AssertCLIOutputContains(t, result, "Version:")
}

func TestCLIHelp(t *testing.T) {
	result := RunCLICommandFull(t, "help")
	AssertCLISuccess(t, result)
	AssertCLIOutputContains(t, result, "Available Commands:")
}

func TestCLIInvalidCommand(t *testing.T) {
	result := RunCLICommandFull(t, "invalid-command")
	AssertCLIFailure(t, result, 1)
	AssertCLIOutputContains(t, result, "unknown command")
}

// File system helpers for functional tests
func SetupTempWorkspace(t *testing.T) string {
	tempDir := t.TempDir()

	// Create a basic .grctool.yaml config
	configContent := `data_dir: .
log_level: info
evidence:
  tools:
    github:
      enabled: true
      repository: test/repo
    terraform:
      enabled: true
      scan_paths:
        - terraform/
`

	configPath := tempDir + "/.grctool.yaml"
	err := exec.Command("bash", "-c", "echo '"+configContent+"' > "+configPath).Run()
	require.NoError(t, err)

	return tempDir
}

// Environment variable helpers
func WithTestEnv(env map[string]string, fn func()) {
	// Save current environment
	oldEnv := make(map[string]string)
	for key := range env {
		oldEnv[key] = exec.Command("bash", "-c", "echo $"+key).String()
	}

	// Set test environment
	for key, value := range env {
		exec.Command("bash", "-c", "export "+key+"="+value).Run()
	}

	// Run function
	defer func() {
		// Restore environment
		for key, value := range oldEnv {
			if value == "" {
				exec.Command("bash", "-c", "unset "+key).Run()
			} else {
				exec.Command("bash", "-c", "export "+key+"="+value).Run()
			}
		}
	}()

	fn()
}
