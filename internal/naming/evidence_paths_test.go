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

package naming

import (
	"strings"
	"testing"
)

func TestGetEvidenceTaskDirName(t *testing.T) {
	tests := []struct {
		name      string
		taskName  string
		taskRef   string
		tugboatID string
		want      string
	}{
		{
			name:      "simple task name",
			taskName:  "GitHub Access Controls",
			taskRef:   "ET-0001",
			tugboatID: "328031",
			want:      "GitHub_Access_Controls_ET-0001_328031",
		},
		{
			name:      "task name with special characters",
			taskName:  "Test/Report: Access & Permissions",
			taskRef:   "ET-0042",
			tugboatID: "328042",
			want:      "Test_Report_Access_Permissions_ET-0042_328042",
		},
		{
			name:      "task name with multiple spaces",
			taskName:  "User   Access   Review",
			taskRef:   "ET-0100",
			tugboatID: "328100",
			want:      "User_Access_Review_ET-0100_328100",
		},
		{
			name:      "task name with parentheses and brackets",
			taskName:  "Policy Review (Q1) [Draft]",
			taskRef:   "ET-0005",
			tugboatID: "328005",
			want:      "Policy_Review_(Q1)_[Draft]_ET-0005_328005",
		},
		{
			name:      "very long task name",
			taskName:  "This is a very long task name that should be truncated to fit within the maximum allowed length for filesystem compatibility and readability purposes",
			taskRef:   "ET-0200",
			tugboatID: "328200",
			want:      "This_is_a_very_long_task_name_that_should_be_truncated_to_fit_within_the_maximum_allowed_length_for_ET-0200_328200",
		},
		{
			name:      "task name with leading and trailing spaces",
			taskName:  "  Trimmed Task  ",
			taskRef:   "ET-0003",
			tugboatID: "328003",
			want:      "Trimmed_Task_ET-0003_328003",
		},
		{
			name:      "empty task name",
			taskName:  "",
			taskRef:   "ET-0099",
			tugboatID: "328099",
			want:      "_ET-0099_328099",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetEvidenceTaskDirName(tt.taskName, tt.taskRef, tt.tugboatID)
			if got != tt.want {
				t.Errorf("GetEvidenceTaskDirName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseEvidenceTaskDirName(t *testing.T) {
	tests := []struct {
		name          string
		dirName       string
		wantName      string
		wantRef       string
		wantTugboatID string
	}{
		{
			name:          "valid directory name",
			dirName:       "GitHub_Access_Controls_ET-0001_328031",
			wantName:      "GitHub Access Controls",
			wantRef:       "ET-0001",
			wantTugboatID: "328031",
		},
		{
			name:          "directory with parentheses",
			dirName:       "Policy_Review_(Q1)_[Draft]_ET-0005_328005",
			wantName:      "Policy Review (Q1) [Draft]",
			wantRef:       "ET-0005",
			wantTugboatID: "328005",
		},
		{
			name:          "invalid format - old format",
			dirName:       "ET-0001_GitHub_Access_Controls",
			wantName:      "",
			wantRef:       "",
			wantTugboatID: "",
		},
		{
			name:          "invalid format - wrong reference pattern",
			dirName:       "INVALID_Task_Name",
			wantName:      "",
			wantRef:       "",
			wantTugboatID: "",
		},
		{
			name:          "directory with hyphenated name",
			dirName:       "Test-Task-Name_ET-0042_328042",
			wantName:      "Test-Task-Name",
			wantRef:       "ET-0042",
			wantTugboatID: "328042",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotRef, gotTugboatID := ParseEvidenceTaskDirName(tt.dirName)
			if gotName != tt.wantName {
				t.Errorf("ParseEvidenceTaskDirName() name = %v, want %v", gotName, tt.wantName)
			}
			if gotRef != tt.wantRef {
				t.Errorf("ParseEvidenceTaskDirName() ref = %v, want %v", gotRef, tt.wantRef)
			}
			if gotTugboatID != tt.wantTugboatID {
				t.Errorf("ParseEvidenceTaskDirName() tugboatID = %v, want %v", gotTugboatID, tt.wantTugboatID)
			}
		})
	}
}

func TestSanitizeTaskName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "spaces to underscores",
			input: "GitHub Access Controls",
			want:  "GitHub_Access_Controls",
		},
		{
			name:  "remove special characters",
			input: "Test/Report: Access & Permissions",
			want:  "Test_Report_Access_Permissions",
		},
		{
			name:  "multiple consecutive spaces",
			input: "User   Access   Review",
			want:  "User_Access_Review",
		},
		{
			name:  "preserve parentheses and brackets",
			input: "Policy Review (Q1) [Draft]",
			want:  "Policy_Review_(Q1)_[Draft]",
		},
		{
			name:  "preserve hyphens",
			input: "Test-Task-Name",
			want:  "Test-Task-Name",
		},
		{
			name:  "trim leading and trailing underscores",
			input: "  Trimmed Task  ",
			want:  "Trimmed_Task",
		},
		{
			name:  "remove filesystem-unsafe characters",
			input: `Task<>:"|?*Name`,
			want:  "Task_Name",
		},
		{
			name:  "truncate long names",
			input: strings.Repeat("A", 150),
			want:  strings.Repeat("A", MaxTaskNameLength),
		},
		{
			name:  "truncate and trim trailing underscore",
			input: strings.Repeat("A", 98) + " B",
			want:  strings.Repeat("A", 98) + "_B",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only special characters",
			input: "@#$%^&*",
			want:  "", // All special chars become underscores, then trimmed
		},
		{
			name:  "unicode characters",
			input: "Task名前Name",
			want:  "Task_Name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeTaskName(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeTaskName() = %v, want %v", got, tt.want)
			}

			// Verify length constraint
			if len(got) > MaxTaskNameLength {
				t.Errorf("SanitizeTaskName() length = %d, exceeds max %d", len(got), MaxTaskNameLength)
			}

			// Verify no trailing underscores
			if len(got) > 0 && strings.HasSuffix(got, "_") {
				t.Errorf("SanitizeTaskName() has trailing underscore: %v", got)
			}

			// Verify no consecutive underscores
			if strings.Contains(got, "__") {
				t.Errorf("SanitizeTaskName() has consecutive underscores: %v", got)
			}
		})
	}
}

func TestMatchesTaskRef(t *testing.T) {
	tests := []struct {
		name    string
		dirName string
		taskRef string
		want    bool
	}{
		{
			name:    "exact match",
			dirName: "ET-0001",
			taskRef: "ET-0001",
			want:    true,
		},
		{
			name:    "new format match",
			dirName: "GitHub_Access_Controls_ET-0001_328031",
			taskRef: "ET-0001",
			want:    true,
		},
		{
			name:    "no match",
			dirName: "Task_Name_ET-0002_328002",
			taskRef: "ET-0001",
			want:    false,
		},
		{
			name:    "partial reference match without proper format",
			dirName: "ET-0001234",
			taskRef: "ET-0001",
			want:    false,
		},
		{
			name:    "different reference",
			dirName: "Something_ET-0100_328100",
			taskRef: "ET-0001",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchesTaskRef(tt.dirName, tt.taskRef)
			if got != tt.want {
				t.Errorf("MatchesTaskRef() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractTaskRef(t *testing.T) {
	tests := []struct {
		name    string
		dirName string
		want    string
	}{
		{
			name:    "extract from full directory name",
			dirName: "GitHub_Access_Controls_ET-0001_328031",
			want:    "ET-0001",
		},
		{
			name:    "extract from reference-only directory",
			dirName: "ET-0001",
			want:    "ET-0001",
		},
		{
			name:    "extract with parentheses",
			dirName: "Policy_(Q1)_ET-0042_328042",
			want:    "ET-0042",
		},
		{
			name:    "invalid format",
			dirName: "INVALID_DIR",
			want:    "",
		},
		{
			name:    "empty string",
			dirName: "",
			want:    "",
		},
		{
			name:    "wrong reference pattern - three digits",
			dirName: "Task_ET-999_328999",
			want:    "",
		},
		{
			name:    "four digit reference",
			dirName: "Final_Task_ET-9999_329999",
			want:    "ET-9999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractTaskRef(tt.dirName)
			if got != tt.want {
				t.Errorf("ExtractTaskRef() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	// Test that we can create a directory name and parse it back
	tests := []struct {
		taskName  string
		taskRef   string
		tugboatID string
	}{
		{"GitHub Access Controls", "ET-0001", "328031"},
		{"Policy Review (Q1)", "ET-0042", "328042"},
		{"Test Task with Special Characters!@#", "ET-0100", "328100"},
	}

	for _, tt := range tests {
		t.Run(tt.taskRef, func(t *testing.T) {
			// Create directory name
			dirName := GetEvidenceTaskDirName(tt.taskName, tt.taskRef, tt.tugboatID)

			// Parse it back
			gotName, gotRef, gotTugboatID := ParseEvidenceTaskDirName(dirName)

			if gotRef != tt.taskRef {
				t.Errorf("Round trip failed: ref = %v, want %v", gotRef, tt.taskRef)
			}

			if gotTugboatID != tt.tugboatID {
				t.Errorf("Round trip failed: tugboatID = %v, want %v", gotTugboatID, tt.tugboatID)
			}

			// The parsed name should match the sanitized version
			expectedName := strings.ReplaceAll(SanitizeTaskName(tt.taskName), "_", " ")
			if gotName != expectedName {
				t.Errorf("Round trip failed: name = %v, want %v", gotName, expectedName)
			}
		})
	}
}
