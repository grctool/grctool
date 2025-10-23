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

//go:build !e2e && !functional

package helpers

import (
	"testing"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/stretchr/testify/require"
)

// Unit test helpers - fast, no external dependencies
func CreateTestLogger(t *testing.T) logger.Logger {
	log, err := logger.New(&logger.Config{
		Level:  logger.ErrorLevel, // Quiet during tests
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err)
	return log
}

func CreateStubConfig(t *testing.T) *config.Config {
	return &config.Config{
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				GitHub: config.GitHubToolConfig{
					Enabled:    true,
					Repository: "test/repo",
					APIToken:   "stub-token",
				},
				Terraform: config.TerraformToolConfig{
					Enabled:   true,
					ScanPaths: []string{"test_data"},
				},
			},
		},
		Storage: config.StorageConfig{
			DataDir: t.TempDir(),
		},
	}
}

// Fast unit test setup with minimal dependencies
func SetupUnitTest(t *testing.T) (*config.Config, logger.Logger) {
	cfg := CreateStubConfig(t)
	log := CreateTestLogger(t)
	return cfg, log
}

// Create stub implementations for unit testing
func CreateStubHTTPResponse(statusCode int, body string) *MockHTTPResponse {
	return &MockHTTPResponse{
		StatusCode: statusCode,
		Body:       body,
		Headers:    make(map[string]string),
	}
}

type MockHTTPResponse struct {
	StatusCode int
	Body       string
	Headers    map[string]string
}

func (m *MockHTTPResponse) AddHeader(key, value string) {
	m.Headers[key] = value
}

// Validation helpers for unit tests
func RequireNoError(t *testing.T, err error, msgAndArgs ...interface{}) {
	require.NoError(t, err, msgAndArgs...)
}

func RequireEqual(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	require.Equal(t, expected, actual, msgAndArgs...)
}

func RequireContains(t *testing.T, haystack, needle string, msgAndArgs ...interface{}) {
	require.Contains(t, haystack, needle, msgAndArgs...)
}
