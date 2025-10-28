package naming

import (
	"strings"
	"testing"
)

func TestGetEvidenceTaskDirName(t *testing.T) {
	tests := []struct {
		name     string
		taskRef  string
		taskName string
		want     string
	}{
		{
			name:     "simple task name",
			taskRef:  "ET-0001",
			taskName: "GitHub Access Controls",
			want:     "ET-0001_GitHub_Access_Controls",
		},
		{
			name:     "task name with special characters",
			taskRef:  "ET-0042",
			taskName: "Test/Report: Access & Permissions",
			want:     "ET-0042_Test_Report_Access_Permissions",
		},
		{
			name:     "task name with multiple spaces",
			taskRef:  "ET-0100",
			taskName: "User   Access   Review",
			want:     "ET-0100_User_Access_Review",
		},
		{
			name:     "task name with parentheses and brackets",
			taskRef:  "ET-0005",
			taskName: "Policy Review (Q1) [Draft]",
			want:     "ET-0005_Policy_Review_(Q1)_[Draft]",
		},
		{
			name:     "very long task name",
			taskRef:  "ET-0200",
			taskName: "This is a very long task name that should be truncated to fit within the maximum allowed length for filesystem compatibility and readability purposes",
			want:     "ET-0200_This_is_a_very_long_task_name_that_should_be_truncated_to_fit_within_the_maximum_allowed_length_for",
		},
		{
			name:     "task name with leading and trailing spaces",
			taskRef:  "ET-0003",
			taskName: "  Trimmed Task  ",
			want:     "ET-0003_Trimmed_Task",
		},
		{
			name:     "empty task name",
			taskRef:  "ET-0099",
			taskName: "",
			want:     "ET-0099_",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetEvidenceTaskDirName(tt.taskRef, tt.taskName)
			if got != tt.want {
				t.Errorf("GetEvidenceTaskDirName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseEvidenceTaskDirName(t *testing.T) {
	tests := []struct {
		name     string
		dirName  string
		wantRef  string
		wantName string
	}{
		{
			name:     "valid directory name",
			dirName:  "ET-0001_GitHub_Access_Controls",
			wantRef:  "ET-0001",
			wantName: "GitHub Access Controls",
		},
		{
			name:     "directory with parentheses",
			dirName:  "ET-0005_Policy_Review_(Q1)_[Draft]",
			wantRef:  "ET-0005",
			wantName: "Policy Review (Q1) [Draft]",
		},
		{
			name:     "invalid format - no underscore",
			dirName:  "ET-0001",
			wantRef:  "",
			wantName: "",
		},
		{
			name:     "invalid format - wrong reference pattern",
			dirName:  "INVALID_Task_Name",
			wantRef:  "",
			wantName: "",
		},
		{
			name:     "directory with hyphenated name",
			dirName:  "ET-0042_Test-Task-Name",
			wantRef:  "ET-0042",
			wantName: "Test-Task-Name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRef, gotName := ParseEvidenceTaskDirName(tt.dirName)
			if gotRef != tt.wantRef {
				t.Errorf("ParseEvidenceTaskDirName() ref = %v, want %v", gotRef, tt.wantRef)
			}
			if gotName != tt.wantName {
				t.Errorf("ParseEvidenceTaskDirName() name = %v, want %v", gotName, tt.wantName)
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
			name:    "prefix match with underscore",
			dirName: "ET-0001_GitHub_Access_Controls",
			taskRef: "ET-0001",
			want:    true,
		},
		{
			name:    "no match",
			dirName: "ET-0002_Task_Name",
			taskRef: "ET-0001",
			want:    false,
		},
		{
			name:    "partial reference match without underscore",
			dirName: "ET-0001234",
			taskRef: "ET-0001",
			want:    false,
		},
		{
			name:    "different reference",
			dirName: "ET-0100_Something",
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
			dirName: "ET-0001_GitHub_Access_Controls",
			want:    "ET-0001",
		},
		{
			name:    "extract from reference-only directory",
			dirName: "ET-0001",
			want:    "ET-0001",
		},
		{
			name:    "extract with parentheses",
			dirName: "ET-0042_Policy_(Q1)",
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
			name:    "wrong reference pattern",
			dirName: "ET-999_Task",
			want:    "",
		},
		{
			name:    "four digit reference",
			dirName: "ET-9999_Final_Task",
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
		taskRef  string
		taskName string
	}{
		{"ET-0001", "GitHub Access Controls"},
		{"ET-0042", "Policy Review (Q1)"},
		{"ET-0100", "Test Task with Special Characters!@#"},
	}

	for _, tt := range tests {
		t.Run(tt.taskRef, func(t *testing.T) {
			// Create directory name
			dirName := GetEvidenceTaskDirName(tt.taskRef, tt.taskName)

			// Parse it back
			gotRef, gotName := ParseEvidenceTaskDirName(dirName)

			if gotRef != tt.taskRef {
				t.Errorf("Round trip failed: ref = %v, want %v", gotRef, tt.taskRef)
			}

			// The parsed name should match the sanitized version
			expectedName := strings.ReplaceAll(SanitizeTaskName(tt.taskName), "_", " ")
			if gotName != expectedName {
				t.Errorf("Round trip failed: name = %v, want %v", gotName, expectedName)
			}
		})
	}
}
