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

package services

import (
	"testing"

	"github.com/grctool/grctool/internal/naming"
)

// Test helper functions
func TestExtractTaskIDFromRef(t *testing.T) {
	tests := []struct {
		name     string
		taskRef  string
		expected int
	}{
		{"Basic task ref", "ET-0001", 1},
		{"Large task ID", "ET-0047", 47},
		{"Padded task ID", "ET-0104", 104},
		{"Invalid format", "ET-XYZ", 0},
		{"Missing prefix", "0001", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTaskIDFromRef(tt.taskRef)
			if result != tt.expected {
				t.Errorf("extractTaskIDFromRef(%s) = %d, want %d", tt.taskRef, result, tt.expected)
			}
		})
	}
}

func TestExtractTaskName(t *testing.T) {
	tests := []struct {
		name     string
		dirname  string
		expected string
	}{
		{"Simple name", "TaskName_ET-0001_328001", "TaskName"},
		{"Multi-word name", "GitHub_Access_ET-0047_328047", "GitHub Access"},
		{"Complex name", "Terraform_Security_Audit_ET-0104_328104", "Terraform Security Audit"},
		{"Invalid format - just ref", "ET-0001", ""},
		{"Invalid format - no ref", "TaskName", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, _ := naming.ParseEvidenceTaskDirName(tt.dirname)
			if result != tt.expected {
				t.Errorf("ParseEvidenceTaskDirName(%s) = %q, want %q", tt.dirname, result, tt.expected)
			}
		})
	}
}

func TestIsWindowDirectory(t *testing.T) {
	tests := []struct {
		name     string
		dirname  string
		expected bool
	}{
		// Valid window patterns
		{"Year only", "2025", true},
		{"Quarterly Q1", "2025-Q1", true},
		{"Quarterly Q4", "2025-Q4", true},
		{"Monthly Jan", "2025-01", true},
		{"Monthly Dec", "2025-12", true},
		{"Half-year H1", "2025-H1", true},
		{"Half-year H2", "2025-H2", true},

		// Invalid patterns
		{"Invalid quarter", "2025-Q5", false},
		{"Invalid half", "2025-H3", false},
		{"Invalid month", "2025-13", false},
		{"Missing dash", "2025Q4", false},
		{"Random text", "evidence", false},
		{"Hidden dir", ".context", false},
		{"Task ref", "ET-0001", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isWindowDirectory(tt.dirname)
			if result != tt.expected {
				t.Errorf("isWindowDirectory(%s) = %v, want %v", tt.dirname, result, tt.expected)
			}
		})
	}
}
