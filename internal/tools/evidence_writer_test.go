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

package tools

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/naming"
	"github.com/stretchr/testify/assert"
)

func TestCalculateEvidenceWindow(t *testing.T) {
	// Table-driven tests using maps (AGENTS.md preferred pattern)
	tests := map[string]struct {
		interval string
		date     time.Time
		want     string
	}{
		"annual window 2025": {
			interval: "year",
			date:     time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
			want:     "2025",
		},
		"annual with 'annually' variant": {
			interval: "annually",
			date:     time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
			want:     "2025",
		},
		"quarterly Q1": {
			interval: "quarter",
			date:     time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
			want:     "2025-Q1",
		},
		"quarterly Q2": {
			interval: "quarter",
			date:     time.Date(2025, 5, 15, 0, 0, 0, 0, time.UTC),
			want:     "2025-Q2",
		},
		"quarterly Q3": {
			interval: "quarterly",
			date:     time.Date(2025, 8, 10, 0, 0, 0, 0, time.UTC),
			want:     "2025-Q3",
		},
		"quarterly Q4": {
			interval: "quarter",
			date:     time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC),
			want:     "2025-Q4",
		},
		"monthly January": {
			interval: "month",
			date:     time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC),
			want:     "2025-01",
		},
		"monthly December": {
			interval: "monthly",
			date:     time.Date(2025, 12, 25, 0, 0, 0, 0, time.UTC),
			want:     "2025-12",
		},
		"semi-annual H1 early": {
			interval: "six_month",
			date:     time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
			want:     "2025-H1",
		},
		"semi-annual H1 boundary": {
			interval: "semi-annual",
			date:     time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC),
			want:     "2025-H1",
		},
		"semi-annual H2 early": {
			interval: "semiannual",
			date:     time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
			want:     "2025-H2",
		},
		"semi-annual H2 late": {
			interval: "six_month",
			date:     time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC),
			want:     "2025-H2",
		},
		"unknown interval defaults to annual": {
			interval: "unknown",
			date:     time.Date(2025, 5, 15, 0, 0, 0, 0, time.UTC),
			want:     "2025",
		},
		"empty interval defaults to annual": {
			interval: "",
			date:     time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
			want:     "2025",
		},
		"case insensitive": {
			interval: "QUARTER",
			date:     time.Date(2025, 4, 15, 0, 0, 0, 0, time.UTC),
			want:     "2025-Q2",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := CalculateEvidenceWindow(tt.interval, tt.date)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGenerateEvidenceFilename(t *testing.T) {
	tests := map[string]struct {
		index int
		title string
		want  string
	}{
		"simple title": {
			index: 1,
			title: "Access Control Policy",
			want:  "01_access_control_policy",
		},
		"with special characters": {
			index: 2,
			title: "User@Provisioning#Data! & Config",
			want:  "02_userprovisioningdata_config",
		},
		"double digit index": {
			index: 15,
			title: "Test Evidence",
			want:  "15_test_evidence",
		},
		"hyphens and underscores": {
			index: 3,
			title: "Terraform-IAM_Config",
			want:  "03_terraform_iam_config",
		},
		"multiple spaces": {
			index: 4,
			title: "Multiple   Spaces   Here",
			want:  "04_multiple_spaces_here",
		},
		"trailing and leading spaces": {
			index: 5,
			title: "  Padded Title  ",
			want:  "05_padded_title",
		},
		"numbers in title": {
			index: 6,
			title: "Q1 2025 Report v2.1",
			want:  "06_q1_2025_report_v21",
		},
		"parentheses and brackets": {
			index: 7,
			title: "Policy Document (Final) [Approved]",
			want:  "07_policy_document_final_approved",
		},
		"empty title": {
			index: 8,
			title: "",
			want:  "08_evidence",
		},
		"only special characters": {
			index: 9,
			title: "!@#$%^&*()",
			want:  "09_evidence",
		},
		"consecutive underscores": {
			index: 10,
			title: "Test___Multiple___Underscores",
			want:  "10_test_multiple_underscores",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := GenerateEvidenceFilename(tt.index, tt.title)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCalculateCompleteness(t *testing.T) {
	tests := map[string]struct {
		entries []EvidenceEntry
		want    float64
	}{
		"no evidence": {
			entries: []EvidenceEntry{},
			want:    0.0,
		},
		"single complete evidence": {
			entries: []EvidenceEntry{{Status: "complete"}},
			want:    1.0,
		},
		"single partial evidence": {
			entries: []EvidenceEntry{{Status: "partial"}},
			want:    0.5,
		},
		"single pending evidence": {
			entries: []EvidenceEntry{{Status: "pending"}},
			want:    0.0,
		},
		"mixed complete and partial": {
			entries: []EvidenceEntry{
				{Status: "complete"},
				{Status: "partial"},
				{Status: "complete"},
			},
			want: 5.0 / 6.0, // (1.0 + 0.5 + 1.0) / 3 = 2.5 / 3
		},
		"all complete": {
			entries: []EvidenceEntry{
				{Status: "complete"},
				{Status: "complete"},
			},
			want: 1.0,
		},
		"all partial": {
			entries: []EvidenceEntry{
				{Status: "partial"},
				{Status: "partial"},
			},
			want: 0.5,
		},
		"all pending": {
			entries: []EvidenceEntry{
				{Status: "pending"},
				{Status: "pending"},
				{Status: "pending"},
			},
			want: 0.0,
		},
		"mixed all statuses": {
			entries: []EvidenceEntry{
				{Status: "complete"}, // 1.0
				{Status: "partial"},  // 0.5
				{Status: "pending"},  // 0.0
				{Status: "complete"}, // 1.0
			},
			want: 2.5 / 4.0, // 0.625
		},
		"unknown status treated as pending": {
			entries: []EvidenceEntry{
				{Status: "complete"},
				{Status: "unknown"},
				{Status: "invalid"},
			},
			want: 1.0 / 3.0, // Only complete counts
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := CalculateCompleteness(tt.entries)
			assert.InDelta(t, tt.want, got, 0.001, "Completeness calculation should be accurate")
		})
	}
}

func TestFormatCompleteness(t *testing.T) {
	tests := map[string]struct {
		score float64
		want  string
	}{
		"zero percent": {
			score: 0.0,
			want:  "0%",
		},
		"fifty percent": {
			score: 0.5,
			want:  "50%",
		},
		"hundred percent": {
			score: 1.0,
			want:  "100%",
		},
		"partial percentage": {
			score: 0.75,
			want:  "75%",
		},
		"low percentage": {
			score: 0.123,
			want:  "12%",
		},
		"high percentage": {
			score: 0.987,
			want:  "98%",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := FormatCompleteness(tt.score)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetCompletenessStatus(t *testing.T) {
	tests := map[string]struct {
		score float64
		want  string
	}{
		"not started": {
			score: 0.0,
			want:  "Not Started",
		},
		"barely started": {
			score: 0.1,
			want:  "Started",
		},
		"half way": {
			score: 0.5,
			want:  "In Progress",
		},
		"mostly done": {
			score: 0.8,
			want:  "In Progress",
		},
		"nearly complete": {
			score: 0.89,
			want:  "In Progress",
		},
		"complete threshold": {
			score: 0.9,
			want:  "Complete",
		},
		"fully complete": {
			score: 1.0,
			want:  "Complete",
		},
		"over complete": {
			score: 1.1,
			want:  "Complete",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := GetCompletenessStatus(tt.score)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSanitizeTaskName(t *testing.T) {
	tests := map[string]struct {
		name string
		want string
	}{
		"simple name": {
			name: "Access Control",
			want: "Access_Control",
		},
		"with special characters": {
			name: "User@Registration#Process!",
			want: "User_Registration_Process",
		},
		"with parentheses and brackets": {
			name: "Policy Document (Final) [Approved]",
			want: "Policy_Document_(Final)_[Approved]",
		},
		"multiple spaces": {
			name: "Multiple   Spaces   Here",
			want: "Multiple_Spaces_Here",
		},
		"trailing and leading spaces": {
			name: "  Padded Name  ",
			want: "Padded_Name",
		},
		"underscores and hyphens": {
			name: "Already_Good-Name",
			want: "Already_Good-Name",
		},
		"empty name": {
			name: "",
			want: "",
		},
		"only special characters": {
			name: "!@#$%^&*",
			want: "",
		},
		"very long name": {
			name: "This is a very long evidence task name that exceeds the typical filesystem limitations and should be truncated to ensure compatibility with all operating systems",
			want: "This_is_a_very_long_evidence_task_name_that_exceeds_the_typical_filesystem_limitations_and_should_be",
		},
		"consecutive underscores": {
			name: "Test___Multiple___Underscores",
			want: "Test_Multiple_Underscores",
		},
		"unicode characters": {
			name: "Policy Café Review — Final",
			want: "Policy_Caf_Review_Final",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := naming.SanitizeTaskName(tt.name)
			assert.Equal(t, tt.want, got)

			// Additional checks for filesystem safety
			assert.LessOrEqual(t, len(got), 100, "Sanitized name should not exceed 100 characters")
			assert.NotEqual(t, "_", got, "Sanitized name should not be just underscores")
		})
	}
}

func TestCalculateFileChecksum(t *testing.T) {
	tests := map[string]struct {
		content  string
		expected string // Expected checksum (sha256: + hex)
	}{
		"hello world": {
			content:  "Hello, World!",
			expected: "sha256:dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f",
		},
		"empty file": {
			content:  "",
			expected: "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		"single line": {
			content:  "test",
			expected: "sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
		},
		"multiline": {
			content:  "line1\nline2\nline3",
			expected: "sha256:6bb6a5ad9b9c43a7cb535e636578716b64ac42edea814a4cad102ba404946837",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create a temporary test file
			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "test.txt")

			err := os.WriteFile(testFile, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Calculate checksum
			checksum, err := calculateFileChecksum(testFile)
			if err != nil {
				t.Fatalf("calculateFileChecksum failed: %v", err)
			}

			// Verify checksum matches expected value
			assert.Equal(t, tt.expected, checksum)

			// Verify checksum format
			assert.NotEmpty(t, checksum, "Checksum should not be empty")
			assert.True(t, len(checksum) > 7, "Checksum should have content after prefix")
			assert.Equal(t, "sha256:", checksum[:7], "Checksum should start with 'sha256:'")

			// Verify checksum length (sha256: + 64 hex chars)
			expectedLength := len("sha256:") + 64
			assert.Equal(t, expectedLength, len(checksum), "Checksum should be exactly 71 characters")
		})
	}
}

func TestCalculateFileChecksumConsistency(t *testing.T) {
	// Create a temporary test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	testContent := "Consistency test content"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Calculate checksum multiple times
	checksum1, err := calculateFileChecksum(testFile)
	if err != nil {
		t.Fatalf("calculateFileChecksum failed on first call: %v", err)
	}

	checksum2, err := calculateFileChecksum(testFile)
	if err != nil {
		t.Fatalf("calculateFileChecksum failed on second call: %v", err)
	}

	checksum3, err := calculateFileChecksum(testFile)
	if err != nil {
		t.Fatalf("calculateFileChecksum failed on third call: %v", err)
	}

	// Verify all checksums are identical
	assert.Equal(t, checksum1, checksum2, "Checksums should match for same file")
	assert.Equal(t, checksum2, checksum3, "Checksums should match for same file")
	assert.Equal(t, checksum1, checksum3, "Checksums should match for same file")
}

func TestCalculateFileChecksumNonexistentFile(t *testing.T) {
	// Test with a nonexistent file
	_, err := calculateFileChecksum("/nonexistent/file/path/that/does/not/exist.txt")
	assert.Error(t, err, "Expected error for nonexistent file")
	assert.Contains(t, err.Error(), "reading file for checksum", "Error should mention reading file")
}

func TestCalculateFileChecksumUniqueness(t *testing.T) {
	// Create two temporary files with different content
	tempDir := t.TempDir()
	testFile1 := filepath.Join(tempDir, "test1.txt")
	testFile2 := filepath.Join(tempDir, "test2.txt")

	err := os.WriteFile(testFile1, []byte("content1"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file 1: %v", err)
	}

	err = os.WriteFile(testFile2, []byte("content2"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file 2: %v", err)
	}

	// Calculate checksums
	checksum1, err := calculateFileChecksum(testFile1)
	if err != nil {
		t.Fatalf("calculateFileChecksum failed for file 1: %v", err)
	}

	checksum2, err := calculateFileChecksum(testFile2)
	if err != nil {
		t.Fatalf("calculateFileChecksum failed for file 2: %v", err)
	}

	// Verify checksums are different
	assert.NotEqual(t, checksum1, checksum2, "Different file contents should produce different checksums")
}
