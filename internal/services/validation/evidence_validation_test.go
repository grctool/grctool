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

package validation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvidenceValidationService_ValidateEvidence(t *testing.T) {
	tmpDir := t.TempDir()

	// Create storage
	cfg := config.StorageConfig{
		DataDir: tmpDir,
		Paths:   config.StoragePaths{}.WithDefaults(),
	}
	stor, err := storage.NewStorage(cfg)
	require.NoError(t, err)

	// Create validation service
	validator := NewEvidenceValidationService(stor)

	// Create test evidence directory with files
	evidenceDir := filepath.Join(tmpDir, "evidence", "ET-0001", "2025-Q4")
	require.NoError(t, os.MkdirAll(evidenceDir, 0755))

	// Create test files
	testFile1 := filepath.Join(evidenceDir, "01_terraform_roles.md")
	testFile2 := filepath.Join(evidenceDir, "02_github_perms.md")
	err = os.WriteFile(testFile1, []byte("# Test Evidence 1\n\nContent here"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(testFile2, []byte("# Test Evidence 2\n\nMore content"), 0644)
	require.NoError(t, err)

	tests := []struct {
		name              string
		taskRef           string
		window            string
		mode              EvidenceValidationMode
		expectedStatus    string
		expectedReady     bool
		expectedPassedMin int
	}{
		{
			name:              "strict mode with valid evidence",
			taskRef:           "ET-0001",
			window:            "2025-Q4",
			mode:              EvidenceValidationModeStrict,
			expectedStatus:    "passed",
			expectedReady:     true,
			expectedPassedMin: 6, // Most checks should pass
		},
		{
			name:              "lenient mode with valid evidence",
			taskRef:           "ET-0001",
			window:            "2025-Q4",
			mode:              EvidenceValidationModeLenient,
			expectedStatus:    "passed",
			expectedReady:     true,
			expectedPassedMin: 6,
		},
		{
			name:              "advisory mode always ready",
			taskRef:           "ET-0001",
			window:            "2025-Q4",
			mode:              EvidenceValidationModeAdvisory,
			expectedReady:     true,
			expectedPassedMin: 0,
		},
		{
			name:              "skip mode bypasses validation",
			taskRef:           "ET-0001",
			window:            "2025-Q4",
			mode:              EvidenceValidationModeSkip,
			expectedStatus:    "skipped",
			expectedReady:     true,
			expectedPassedMin: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &EvidenceValidationRequest{
				TaskRef:        tt.taskRef,
				Window:         tt.window,
				ValidationMode: tt.mode,
			}

			result, err := validator.ValidateEvidence(req)
			require.NoError(t, err)
			assert.NotNil(t, result)

			if tt.expectedStatus != "" {
				assert.Equal(t, tt.expectedStatus, result.Status)
			}
			assert.Equal(t, tt.expectedReady, result.ReadyForSubmission)
			assert.GreaterOrEqual(t, result.PassedChecks, tt.expectedPassedMin)
			assert.Equal(t, tt.taskRef, result.TaskRef)
			assert.Equal(t, tt.window, result.Window)
		})
	}
}

func TestEvidenceValidationService_NoEvidence(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.StorageConfig{
		DataDir: tmpDir,
		Paths:   config.StoragePaths{}.WithDefaults(),
	}
	stor, err := storage.NewStorage(cfg)
	require.NoError(t, err)

	validator := NewEvidenceValidationService(stor)

	// Try to validate non-existent evidence
	req := &EvidenceValidationRequest{
		TaskRef:        "ET-9999",
		Window:         "2025-Q4",
		ValidationMode: EvidenceValidationModeStrict,
	}

	_, err = validator.ValidateEvidence(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get evidence files")
}

func TestValidationRules_MinimumFileCount(t *testing.T) {
	rule := &MinimumFileCountRule{}

	tests := []struct {
		name           string
		fileCount      int
		expectedStatus string
		expectedError  bool
	}{
		{
			name:           "no files - error",
			fileCount:      0,
			expectedStatus: "failed",
			expectedError:  true,
		},
		{
			name:           "one file - warning",
			fileCount:      1,
			expectedStatus: "warning",
			expectedError:  false,
		},
		{
			name:           "multiple files - passed",
			fileCount:      3,
			expectedStatus: "passed",
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := make([]models.EvidenceFileRef, tt.fileCount)
			for i := 0; i < tt.fileCount; i++ {
				files[i] = models.EvidenceFileRef{
					Filename:  "test.md",
					SizeBytes: 1024,
				}
			}

			result := rule.Validate("ET-0001", "2025-Q4", files, nil)
			assert.Equal(t, tt.expectedStatus, result.Check.Status)

			if tt.expectedError {
				assert.NotEmpty(t, result.Errors)
			} else {
				assert.Empty(t, result.Errors)
			}
		})
	}
}

func TestValidationRules_FileSizeLimits(t *testing.T) {
	rule := &FileSizeLimitsRule{}

	tests := []struct {
		name           string
		fileSize       int64
		expectedStatus string
		expectedError  bool
	}{
		{
			name:           "small file - passed",
			fileSize:       1024,
			expectedStatus: "passed",
			expectedError:  false,
		},
		{
			name:           "large but valid file - passed",
			fileSize:       10 * 1024 * 1024, // 10MB
			expectedStatus: "passed",
			expectedError:  false,
		},
		{
			name:           "oversized file - failed",
			fileSize:       100 * 1024 * 1024, // 100MB > 50MB limit
			expectedStatus: "failed",
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := []models.EvidenceFileRef{
				{
					Filename:  "test.md",
					SizeBytes: tt.fileSize,
				},
			}

			result := rule.Validate("ET-0001", "2025-Q4", files, nil)
			assert.Equal(t, tt.expectedStatus, result.Check.Status)

			if tt.expectedError {
				assert.NotEmpty(t, result.Errors)
			}
		})
	}
}

func TestValidationRules_NonEmptyContent(t *testing.T) {
	rule := &NonEmptyContentRule{}

	tests := []struct {
		name           string
		fileSize       int64
		expectedStatus string
	}{
		{
			name:           "empty file",
			fileSize:       0,
			expectedStatus: "failed",
		},
		{
			name:           "file with content",
			fileSize:       1024,
			expectedStatus: "passed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := []models.EvidenceFileRef{
				{
					Filename:  "test.md",
					SizeBytes: tt.fileSize,
				},
			}

			result := rule.Validate("ET-0001", "2025-Q4", files, nil)
			assert.Equal(t, tt.expectedStatus, result.Check.Status)
		})
	}
}

func TestValidationRules_ValidTaskRef(t *testing.T) {
	rule := &ValidTaskRefRule{}

	tests := []struct {
		name           string
		taskRef        string
		expectedStatus string
	}{
		{
			name:           "valid task ref",
			taskRef:        "ET-0001",
			expectedStatus: "passed",
		},
		{
			name:           "valid task ref with large number",
			taskRef:        "ET-9999",
			expectedStatus: "passed",
		},
		{
			name:           "invalid - missing ET prefix",
			taskRef:        "0001",
			expectedStatus: "failed",
		},
		{
			name:           "invalid - lowercase",
			taskRef:        "et-0001",
			expectedStatus: "failed",
		},
		{
			name:           "invalid - too short",
			taskRef:        "ET-",
			expectedStatus: "failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rule.Validate(tt.taskRef, "2025-Q4", nil, nil)
			assert.Equal(t, tt.expectedStatus, result.Check.Status)
		})
	}
}

func TestValidationRules_WindowFormat(t *testing.T) {
	rule := &WindowFormatRule{}

	tests := []struct {
		name           string
		window         string
		expectedStatus string
	}{
		{
			name:           "valid quarterly format",
			window:         "2025-Q4",
			expectedStatus: "passed",
		},
		{
			name:           "valid date format",
			window:         "2025-10-22",
			expectedStatus: "passed",
		},
		{
			name:           "invalid format",
			window:         "Q4-2025",
			expectedStatus: "warning",
		},
		{
			name:           "invalid format - just year",
			window:         "2025",
			expectedStatus: "warning",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rule.Validate("ET-0001", tt.window, nil, nil)
			assert.Equal(t, tt.expectedStatus, result.Check.Status)
		})
	}
}

func TestValidationRules_ValidFileExtensions(t *testing.T) {
	rule := &ValidFileExtensionsRule{}

	tests := []struct {
		name           string
		filename       string
		expectedStatus string
	}{
		{
			name:           "markdown file",
			filename:       "test.md",
			expectedStatus: "passed",
		},
		{
			name:           "csv file",
			filename:       "data.csv",
			expectedStatus: "passed",
		},
		{
			name:           "json file",
			filename:       "config.json",
			expectedStatus: "passed",
		},
		{
			name:           "pdf file",
			filename:       "document.pdf",
			expectedStatus: "passed",
		},
		{
			name:           "unexpected extension",
			filename:       "script.exe",
			expectedStatus: "warning",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := []models.EvidenceFileRef{
				{
					Filename:  tt.filename,
					SizeBytes: 1024,
				},
			}

			result := rule.Validate("ET-0001", "2025-Q4", files, nil)
			assert.Equal(t, tt.expectedStatus, result.Check.Status)
		})
	}
}
