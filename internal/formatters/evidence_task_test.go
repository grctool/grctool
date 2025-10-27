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

package formatters

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/interpolation"
)

func TestNewEvidenceTaskFormatter(t *testing.T) {
	formatter := NewEvidenceTaskFormatter()
	if formatter == nil {
		t.Fatal("Expected formatter to be created, got nil")
	}
	if formatter.baseFormatter == nil {
		t.Fatal("Expected base formatter to be created, got nil")
	}
}

func TestNewEvidenceTaskFormatterWithInterpolation(t *testing.T) {
	config := interpolation.InterpolatorConfig{
		Variables: map[string]string{
			"company.name": "Test Company",
		},
		Enabled:           true,
		OnMissingVariable: interpolation.MissingVariableIgnore,
	}
	interpolator := interpolation.NewStandardInterpolator(config)
	formatter := NewEvidenceTaskFormatterWithInterpolation(interpolator)

	if formatter == nil {
		t.Fatal("Expected formatter to be created, got nil")
	}
	if formatter.baseFormatter == nil {
		t.Fatal("Expected base formatter to be created, got nil")
	}
}

func TestEvidenceTaskFormatter_ToMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		task     *domain.EvidenceTask
		validate func(t *testing.T, markdown string)
	}{
		{
			name: "Complete evidence task with all fields",
			task: createCompleteEvidenceTask(),
			validate: func(t *testing.T, markdown string) {
				// Check header
				if !strings.Contains(markdown, "# Evidence Task 327992") {
					t.Error("Missing evidence task header")
				}
				// Check metadata box
				if !strings.Contains(markdown, "Task ID: 327992") {
					t.Error("Missing task ID in metadata box")
				}
				if !strings.Contains(markdown, "Framework: SOC2") {
					t.Error("Missing framework in metadata box")
				}
				if !strings.Contains(markdown, "Priority: high") {
					t.Error("Missing priority in metadata box")
				}
				// Check sections
				if !strings.Contains(markdown, "## Access Control Registration Document") {
					t.Error("Missing task name section")
				}
				if !strings.Contains(markdown, "### Description") {
					t.Error("Missing description section")
				}
				if !strings.Contains(markdown, "### Collection Guidance") {
					t.Error("Missing guidance section")
				}
				if !strings.Contains(markdown, "### Collection Requirements") {
					t.Error("Missing collection requirements section")
				}
				if !strings.Contains(markdown, "### Basic Information") {
					t.Error("Missing basic information section")
				}
				if !strings.Contains(markdown, "### Collection Timeline") {
					t.Error("Missing collection timeline section")
				}
				if !strings.Contains(markdown, "### Associated Items") {
					t.Error("Missing associated items section")
				}
				if !strings.Contains(markdown, "### Assignees") {
					t.Error("Missing assignees section")
				}
				if !strings.Contains(markdown, "### Related Controls") {
					t.Error("Missing related controls section")
				}
				// Check footer
				if !strings.Contains(markdown, "*Generated on") {
					t.Error("Missing footer")
				}
			},
		},
		{
			name: "Minimal evidence task with basic fields only",
			task: createMinimalEvidenceTask(),
			validate: func(t *testing.T, markdown string) {
				// Check header
				if !strings.Contains(markdown, "# Evidence Task 123") {
					t.Error("Missing evidence task header")
				}
				// Check basic fields
				if !strings.Contains(markdown, "## Basic Evidence Task") {
					t.Error("Missing task name")
				}
				if !strings.Contains(markdown, "Task ID: 123") {
					t.Error("Missing task ID")
				}
				// Should not contain empty sections
				if strings.Contains(markdown, "### Associated Items") {
					t.Error("Should not contain associated items section for minimal task")
				}
				if strings.Contains(markdown, "### Assignees") {
					t.Error("Should not contain assignees section for minimal task")
				}
			},
		},
		{
			name: "Evidence task with HTML content requiring cleaning",
			task: createEvidenceTaskWithHTML(),
			validate: func(t *testing.T, markdown string) {
				// Check that HTML was converted to markdown
				if strings.Contains(markdown, "<p>") || strings.Contains(markdown, "</p>") {
					t.Error("HTML tags should be converted to markdown")
				}
				if !strings.Contains(markdown, "**bold text**") {
					t.Error("Missing converted bold text")
				}
				if !strings.Contains(markdown, "*italic text*") {
					t.Error("Missing converted italic text")
				}
			},
		},
		{
			name: "Evidence task with AEC status",
			task: createEvidenceTaskWithAEC(),
			validate: func(t *testing.T, markdown string) {
				if !strings.Contains(markdown, "### Automated Evidence Collection") {
					t.Error("Missing AEC section")
				}
				if !strings.Contains(markdown, "| **Status** | enabled |") {
					t.Error("Missing AEC status")
				}
			},
		},
		{
			name: "Evidence task with sensitive data flag",
			task: createSensitiveEvidenceTask(),
			validate: func(t *testing.T, markdown string) {
				if !strings.Contains(markdown, "⚠️ **Sensitive Data**") {
					t.Error("Missing sensitive data warning")
				}
				if !strings.Contains(markdown, "| **Sensitive Data** | Yes |") {
					t.Error("Missing sensitive data in metadata table")
				}
			},
		},
		{
			name: "Evidence task with subtasks",
			task: createEvidenceTaskWithSubtasks(),
			validate: func(t *testing.T, markdown string) {
				if !strings.Contains(markdown, "### Subtask Information") {
					t.Error("Missing subtask information section")
				}
				if !strings.Contains(markdown, "| **Total Subtasks** | 5 |") {
					t.Error("Missing total subtasks")
				}
				if !strings.Contains(markdown, "| **Completed Subtasks** | 3 |") {
					t.Error("Missing completed subtasks")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewEvidenceTaskFormatter()
			markdown := formatter.ToMarkdown(tt.task)

			if markdown == "" {
				t.Fatal("Expected non-empty markdown output")
			}

			tt.validate(t, markdown)
		})
	}
}

func TestEvidenceTaskFormatter_ToSummaryMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		task     *domain.EvidenceTask
		validate func(t *testing.T, markdown string)
	}{
		{
			name: "Complete evidence task summary",
			task: createCompleteEvidenceTask(),
			validate: func(t *testing.T, markdown string) {
				if !strings.Contains(markdown, "## Access Control Registration Document") {
					t.Error("Missing task name in summary")
				}
				if !strings.Contains(markdown, "**ID:** 327992") {
					t.Error("Missing task ID in summary")
				}
				if !strings.Contains(markdown, "**Priority:** high") {
					t.Error("Missing priority in summary")
				}
				if !strings.Contains(markdown, "**Collection Interval**: year") {
					t.Error("Missing collection interval in summary")
				}
				if !strings.Contains(markdown, "**Next Due**: 2024-12-31") {
					t.Error("Missing next due date in summary")
				}
				if !strings.Contains(markdown, "**Associated:** 2 controls, 1 policies") {
					t.Error("Missing associations in summary")
				}
			},
		},
		{
			name: "Minimal evidence task summary",
			task: createMinimalEvidenceTask(),
			validate: func(t *testing.T, markdown string) {
				if !strings.Contains(markdown, "## Basic Evidence Task") {
					t.Error("Missing task name in summary")
				}
				if !strings.Contains(markdown, "**ID:** 123") {
					t.Error("Missing task ID in summary")
				}
				// Should not contain association info for minimal task
				if strings.Contains(markdown, "**Associated:**") {
					t.Error("Should not contain associations for minimal task")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewEvidenceTaskFormatter()
			markdown := formatter.ToSummaryMarkdown(tt.task)

			if markdown == "" {
				t.Fatal("Expected non-empty markdown output")
			}

			tt.validate(t, markdown)
		})
	}
}

func TestEvidenceTaskFormatter_ToDocumentMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		task     *domain.EvidenceTask
		validate func(t *testing.T, markdown string)
	}{
		{
			name: "Complete evidence task document",
			task: createCompleteEvidenceTask(),
			validate: func(t *testing.T, markdown string) {
				// Check document header with reference
				if !strings.Contains(markdown, "# ET1 - Access Control Registration Document") {
					t.Error("Missing document header with reference")
				}
				// Check main sections
				if !strings.Contains(markdown, "## Description") {
					t.Error("Missing description section")
				}
				if !strings.Contains(markdown, "## Collection Guidance") {
					t.Error("Missing collection guidance section")
				}
				if !strings.Contains(markdown, "## Collection Requirements") {
					t.Error("Missing collection requirements section")
				}
				if !strings.Contains(markdown, "## Collection Strategy") {
					t.Error("Missing collection strategy section")
				}
				// Check metadata section
				if !strings.Contains(markdown, "## Document Information") {
					t.Error("Missing document information section")
				}
				if !strings.Contains(markdown, "| **Document Reference** | ET1 |") {
					t.Error("Missing document reference in metadata")
				}
				if !strings.Contains(markdown, "| **Task ID** | 327992 |") {
					t.Error("Missing task ID in metadata")
				}
			},
		},
		{
			name: "Evidence task with different reference pattern",
			task: &domain.EvidenceTask{
				ID:                 456,
				Name:               "Network Security Monitoring",
				Description:        "Monitor network security events",
				CollectionInterval: "daily",
				Priority:           "medium",
				CreatedAt:          time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
				UpdatedAt:          time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC),
			},
			validate: func(t *testing.T, markdown string) {
				// Should generate a valid reference for network security monitoring
				if !strings.Contains(markdown, "Network Security Monitoring") {
					t.Error("Missing task name in document")
				}
				if !strings.Contains(markdown, "## Document Information") {
					t.Error("Missing document information section")
				}
				// Check that a reference was generated
				if !strings.Contains(markdown, "**Document Reference**") {
					t.Error("Missing document reference in metadata")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewEvidenceTaskFormatter()

			// Initialize reference mapping for test task
			formatter.InitializeReferenceMapping([]domain.EvidenceTask{*tt.task})

			markdown := formatter.ToDocumentMarkdown(tt.task)

			if markdown == "" {
				t.Fatal("Expected non-empty markdown output")
			}

			tt.validate(t, markdown)
		})
	}
}

func TestEvidenceTaskFormatter_GenerateReference(t *testing.T) {
	formatter := NewEvidenceTaskFormatter()

	// Create test tasks with different IDs
	testCases := []struct {
		name     string
		task     *domain.EvidenceTask
		expected string
	}{
		{
			name:     "First task gets ET1",
			task:     &domain.EvidenceTask{ID: 100, Name: "Access Control Registration"},
			expected: "ET1",
		},
		{
			name:     "Second task gets ET2",
			task:     &domain.EvidenceTask{ID: 200, Name: "Password Policy Review"},
			expected: "ET2",
		},
		{
			name:     "Third task gets ET3",
			task:     &domain.EvidenceTask{ID: 300, Name: "MFA Setup"},
			expected: "ET3",
		},
	}

	// Create test tasks for initialization
	tasks := []domain.EvidenceTask{
		{ID: 100, Name: "Access Control Registration"},
		{ID: 200, Name: "Password Policy Review"},
		{ID: 300, Name: "MFA Setup"},
	}

	// Initialize reference mapping
	formatter.InitializeReferenceMapping(tasks)

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.generateReference(tt.task)
			if result != tt.expected {
				t.Errorf("generateReference() = %q, expected %q", result, tt.expected)
			}
			// Test that result is a valid reference (letters and numbers only)
			validRef := regexp.MustCompile(`^ET\d+$`)
			if !validRef.MatchString(result) {
				t.Errorf("generateReference() = %q, should be valid ET reference format", result)
			}
		})
	}
}

func TestEvidenceTaskFormatter_GetDocumentFilename(t *testing.T) {
	formatter := NewEvidenceTaskFormatter()

	tests := []struct {
		name string
		task *domain.EvidenceTask
	}{
		{
			name: "Standard evidence task",
			task: &domain.EvidenceTask{
				ID:          123,
				ReferenceID: "ET123",
				Name:        "Access Control Review",
			},
		},
		{
			name: "Evidence task with special characters",
			task: &domain.EvidenceTask{
				ID:          456,
				ReferenceID: "ET456",
				Name:        "Security Assessment: Network/Firewall Analysis",
			},
		},
		{
			name: "Evidence task with quotes and colons",
			task: &domain.EvidenceTask{
				ID:          789,
				ReferenceID: "ET789",
				Name:        "Policy Review \"Access Control\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.GetDocumentFilename(tt.task)

			// Check that filename has correct unified structure: ReferenceID_NumericID_Name.md
			if !strings.HasSuffix(result, ".md") {
				t.Error("Filename should end with .md")
			}

			// Check that it follows the unified pattern
			parts := strings.Split(strings.TrimSuffix(result, ".md"), "_")
			if len(parts) < 3 {
				t.Errorf("Filename should have at least 3 parts separated by underscores: %s", result)
			}

			// First part should be reference ID
			if parts[0] != tt.task.ReferenceID {
				t.Errorf("First part should be reference ID %s, got %s", tt.task.ReferenceID, parts[0])
			}

			// Second part should be numeric ID
			if parts[1] != strconv.Itoa(tt.task.ID) {
				t.Errorf("Second part should be numeric ID %d, got %s", tt.task.ID, parts[1])
			}

			if result == "" {
				t.Error("Filename should not be empty")
			}

			// Check that special characters were cleaned
			invalidChars := []string{":", "\"", "/", "?", "*", "<", ">", "|", "\\", " "}
			for _, char := range invalidChars {
				if strings.Contains(result, char) {
					t.Errorf("Filename contains invalid character %q: %s", char, result)
				}
			}
		})
	}
}

func TestEvidenceTaskFormatter_WithInterpolation(t *testing.T) {
	config := interpolation.InterpolatorConfig{
		Variables: map[string]string{
			"company.name":   "Test Company Inc",
			"company.domain": "test.com",
			"security.team":  "Security Team",
		},
		Enabled:           true,
		OnMissingVariable: interpolation.MissingVariableIgnore,
	}
	interpolator := interpolation.NewStandardInterpolator(config)
	formatter := NewEvidenceTaskFormatterWithInterpolation(interpolator)

	task := &domain.EvidenceTask{
		ID:                 999,
		Name:               "{{company.name}} Access Control Policy",
		Description:        "Review access control policy for {{company.domain}}",
		Guidance:           "Contact {{security.team}} for implementation details",
		Framework:          "SOC2",
		Priority:           "high",
		Status:             "pending",
		CollectionInterval: "quarterly",
		CreatedAt:          time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:          time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
	}

	t.Run("ToMarkdown with interpolation", func(t *testing.T) {
		markdown := formatter.ToMarkdown(task)

		// Check that variables were interpolated
		if !strings.Contains(markdown, "Test Company Inc Access Control Policy") {
			t.Error("Company name variable not interpolated in title")
		}
		if !strings.Contains(markdown, "access control policy for test.com") {
			t.Error("Company domain variable not interpolated in description")
		}
		if !strings.Contains(markdown, "Contact Security Team for implementation") {
			t.Error("Security team variable not interpolated in guidance")
		}
		// Check that original template syntax is not present
		if strings.Contains(markdown, "{{company.name}}") {
			t.Error("Template syntax should be replaced with actual values")
		}
	})

	t.Run("ToDocumentMarkdown with interpolation", func(t *testing.T) {
		markdown := formatter.ToDocumentMarkdown(task)

		// Check that variables were interpolated in document format
		if !strings.Contains(markdown, "Test Company Inc Access Control Policy") {
			t.Errorf("Company name variable not interpolated in document. Got: %s", markdown[:300])
		}
	})
}

func TestEvidenceTaskFormatter_EdgeCases(t *testing.T) {
	formatter := NewEvidenceTaskFormatter()

	t.Run("Empty evidence task", func(t *testing.T) {
		task := &domain.EvidenceTask{}
		markdown := formatter.ToMarkdown(task)

		if markdown == "" {
			t.Error("Should return non-empty markdown even for empty task")
		}
		if !strings.Contains(markdown, "# Evidence Task 0") {
			t.Error("Should contain header even for empty task")
		}
	})

	t.Run("Evidence task with nil pointers", func(t *testing.T) {
		task := &domain.EvidenceTask{
			ID:               123,
			Name:             "Test Task",
			LastCollected:    nil,
			NextDue:          nil,
			LastViewedAt:     nil,
			LastDownloadedAt: nil,
			LastReferencedAt: nil,
			MasterContent:    nil,
			Associations:     nil,
			AecStatus:        nil,
			SubtaskMetadata:  nil,
		}

		// Should not panic
		markdown := formatter.ToMarkdown(task)
		if markdown == "" {
			t.Error("Should return non-empty markdown")
		}
	})

	t.Run("Evidence task with zero times", func(t *testing.T) {
		zeroTime := time.Time{}
		task := &domain.EvidenceTask{
			ID:            123,
			Name:          "Test Task",
			LastCollected: &zeroTime,
			NextDue:       &zeroTime,
			CreatedAt:     zeroTime,
			UpdatedAt:     zeroTime,
		}

		markdown := formatter.ToMarkdown(task)
		// Should not include zero times in output
		if strings.Contains(markdown, "0001-01-01") {
			t.Error("Should not include zero times in output")
		}
	})
}

// Test helper functions

func createCompleteEvidenceTask() *domain.EvidenceTask {
	lastCollected := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	nextDue := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	assignedAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	lastViewed := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)

	return &domain.EvidenceTask{
		ID:                 327992,
		Name:               "Access Control Registration Document",
		Description:        "Provide user registration and de-registration process document/Access Control policy or procedure reviewed by management.",
		Guidance:           "The documented access control policies and procedures should demonstrate account management processes.",
		CollectionInterval: "year",
		Priority:           "high",
		Framework:          "SOC2",
		Status:             "pending",
		Completed:          false,
		LastCollected:      &lastCollected,
		NextDue:            &nextDue,
		DueDaysBefore:      30,
		AdHoc:              false,
		Sensitive:          false,
		CreatedAt:          time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:          time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
		Controls:           []string{"AC-1", "AC-2"},
		Assignees: []domain.Person{
			{
				ID:         "123",
				Name:       "John Doe",
				Email:      "john.doe@example.com",
				AssignedAt: &assignedAt,
			},
		},
		Tags: []domain.Tag{
			{Name: "compliance", Color: "#0066cc"},
			{Name: "quarterly", Color: "#00cc66"},
		},
		ViewCount:    15,
		LastViewedAt: &lastViewed,
		MasterContent: &domain.EvidenceTaskMasterContent{
			Guidance:    "The documented access control policies and procedures should demonstrate account management processes.",
			Description: "Access control registration and de-registration procedures",
			Help:        "Review organizational policies and procedures for user account lifecycle management.",
		},
		Associations: &domain.EvidenceTaskAssociations{
			Controls:   2,
			Policies:   1,
			Procedures: 0,
		},
		SupportedIntegrations: []domain.Integration{
			{
				ID:          "github",
				Name:        "GitHub",
				Type:        "code_repository",
				Description: "Collect user access data from GitHub",
				Enabled:     true,
			},
		},
	}
}

func createMinimalEvidenceTask() *domain.EvidenceTask {
	return &domain.EvidenceTask{
		ID:                 123,
		Name:               "Basic Evidence Task",
		Description:        "A simple evidence collection task",
		CollectionInterval: "monthly",
		Priority:           "medium",
		Status:             "active",
		CreatedAt:          time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:          time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
	}
}

func createEvidenceTaskWithHTML() *domain.EvidenceTask {
	return &domain.EvidenceTask{
		ID:                 456,
		Name:               "HTML Content Task",
		Description:        "<p>This contains <strong>bold text</strong> and <em>italic text</em>.</p>",
		Guidance:           "<div>Some <b>guidance</b> with <a href='#'>links</a></div>",
		CollectionInterval: "weekly",
		CreatedAt:          time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:          time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		MasterContent: &domain.EvidenceTaskMasterContent{
			Help: "<ul><li>Item 1</li><li>Item 2</li></ul>",
		},
	}
}

func createEvidenceTaskWithAEC() *domain.EvidenceTask {
	lastExecuted := time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC)
	nextScheduled := time.Date(2024, 7, 1, 10, 0, 0, 0, time.UTC)

	return &domain.EvidenceTask{
		ID:                 789,
		Name:               "Automated Evidence Task",
		Description:        "Task with automated evidence collection",
		CollectionInterval: "daily",
		CreatedAt:          time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:          time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		AecStatus: &domain.AecStatus{
			ID:             "aec-123",
			Status:         "enabled",
			LastExecuted:   &lastExecuted,
			NextScheduled:  &nextScheduled,
			SuccessfulRuns: 10,
			FailedRuns:     1,
			ErrorMessage:   "Connection timeout",
		},
	}
}

func createSensitiveEvidenceTask() *domain.EvidenceTask {
	return &domain.EvidenceTask{
		ID:                 101,
		Name:               "Sensitive Data Task",
		Description:        "Task involving sensitive data collection",
		CollectionInterval: "quarterly",
		Sensitive:          true,
		CreatedAt:          time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:          time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
	}
}

func createEvidenceTaskWithSubtasks() *domain.EvidenceTask {
	return &domain.EvidenceTask{
		ID:                 202,
		Name:               "Task with Subtasks",
		Description:        "Task that has multiple subtasks",
		CollectionInterval: "monthly",
		CreatedAt:          time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:          time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		SubtaskMetadata: &domain.SubtaskMetadata{
			TotalSubtasks:     5,
			CompletedSubtasks: 3,
			PendingSubtasks:   1,
			OverdueSubtasks:   1,
		},
	}
}

// Tests for new functionality added for issue fixes

func TestEvidenceTaskFormatter_InitializeReferenceMapping(t *testing.T) {
	formatter := NewEvidenceTaskFormatter()

	// Test with various task orders to ensure deterministic sorting
	testCases := []struct {
		name  string
		tasks []domain.EvidenceTask
	}{
		{
			name: "Tasks in ascending ID order",
			tasks: []domain.EvidenceTask{
				{ID: 100, Name: "First Task"},
				{ID: 200, Name: "Second Task"},
				{ID: 300, Name: "Third Task"},
			},
		},
		{
			name: "Tasks in descending ID order",
			tasks: []domain.EvidenceTask{
				{ID: 300, Name: "Third Task"},
				{ID: 200, Name: "Second Task"},
				{ID: 100, Name: "First Task"},
			},
		},
		{
			name: "Tasks in random order",
			tasks: []domain.EvidenceTask{
				{ID: 200, Name: "Second Task"},
				{ID: 100, Name: "First Task"},
				{ID: 300, Name: "Third Task"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			formatter.InitializeReferenceMapping(tc.tasks)

			// Check that all tasks get sequential references regardless of input order
			expectedRefs := map[int]string{
				100: "ET1", // Lowest ID gets ET1
				200: "ET2", // Second lowest ID gets ET2
				300: "ET3", // Highest ID gets ET3
			}

			for taskID, expectedRef := range expectedRefs {
				if ref, exists := formatter.referenceMapping[taskID]; !exists {
					t.Errorf("Expected mapping for task ID %d", taskID)
				} else if ref != expectedRef {
					t.Errorf("Task ID %d: expected %q, got %q", taskID, expectedRef, ref)
				}
			}
		})
	}

	// Test empty input
	t.Run("Empty task list", func(t *testing.T) {
		formatter.InitializeReferenceMapping([]domain.EvidenceTask{})
		if len(formatter.referenceMapping) != 0 {
			t.Errorf("Expected empty mapping for empty input, got %d entries", len(formatter.referenceMapping))
		}
	})

	// Test with realistic evidence task IDs (like those from Tugboat)
	t.Run("Realistic task IDs", func(t *testing.T) {
		tasks := []domain.EvidenceTask{
			{ID: 327995, Name: "Task A"},
			{ID: 327992, Name: "Task B"}, // Lower ID should get ET1
			{ID: 328010, Name: "Task C"},
		}

		formatter.InitializeReferenceMapping(tasks)

		// Should be sorted by ID and assigned sequentially
		expectedMappings := map[int]string{
			327992: "ET1", // Lowest ID
			327995: "ET2", // Middle ID
			328010: "ET3", // Highest ID
		}

		for taskID, expectedRef := range expectedMappings {
			if ref := formatter.referenceMapping[taskID]; ref != expectedRef {
				t.Errorf("Task ID %d: expected %q, got %q", taskID, expectedRef, ref)
			}
		}
	})
}

func TestEvidenceTaskFormatter_ToDocumentMarkdownWithContext(t *testing.T) {
	config := interpolation.InterpolatorConfig{
		Variables: map[string]string{
			"organization.name": "Test Organization",
		},
		Enabled:           true,
		OnMissingVariable: interpolation.MissingVariableIgnore,
	}
	interpolator := interpolation.NewStandardInterpolator(config)
	formatter := NewEvidenceTaskFormatterWithInterpolation(interpolator)

	task := &domain.EvidenceTask{
		ID:          123,
		Name:        "Access Control Documentation",
		Description: "Test access control task",
		Controls:    []string{"1", "2"},
	}

	controls := []domain.Control{
		{
			ID:          1,
			ReferenceID: "AC1",
			Name:        "Access Provisioning",
			Description: "Control access to systems",
			Category:    "Access Control",
			Status:      "implemented",
		},
		{
			ID:          2,
			ReferenceID: "AC2",
			Name:        "Access Revocation",
			Description: "Remove access when needed",
			Category:    "Access Control",
			Status:      "implemented",
		},
	}

	policies := []domain.Policy{
		{
			ID:          "policy1",
			Name:        "Access Control Policy",
			Description: "Policy for access control",
			Status:      "published",
		},
	}

	t.Run("Document with controls and policies context", func(t *testing.T) {
		result := formatter.ToDocumentMarkdownWithContext(task, controls, policies)

		// Check that related controls section is included
		if !strings.Contains(result, "## Related Controls") {
			t.Error("Expected 'Related Controls' section in output")
		}

		// Check that specific controls are mentioned
		if !strings.Contains(result, "AC1 - Access Provisioning") {
			t.Error("Expected control AC1 details in output")
		}

		if !strings.Contains(result, "AC2 - Access Revocation") {
			t.Error("Expected control AC2 details in output")
		}

		// Check that related policies section is included
		if !strings.Contains(result, "## Related Policies") {
			t.Error("Expected 'Related Policies' section in output")
		}

		// Check that policy is mentioned
		if !strings.Contains(result, "Access Control Policy") {
			t.Error("Expected policy name in output")
		}
	})

	t.Run("Document without context falls back to basic", func(t *testing.T) {
		result := formatter.ToDocumentMarkdownWithContext(task, []domain.Control{}, []domain.Policy{})

		// Should still contain basic task information
		if !strings.Contains(result, "Access Control Documentation") {
			t.Error("Expected task name in basic output")
		}

		// When no controls are provided for a task that has control IDs,
		// we should get the basic ToDocumentMarkdown result which includes
		// a basic "Related Controls" section showing the control IDs
		// This is expected behavior since the task has Controls: []string{"1", "2"}
	})
}

func TestEvidenceTaskFormatter_AddEnhancedControlSection(t *testing.T) {
	formatter := NewEvidenceTaskFormatter()

	task := &domain.EvidenceTask{
		ID:       123,
		Name:     "Test Task",
		Controls: []string{"1", "999"}, // 999 doesn't exist to test missing control
	}

	controls := []domain.Control{
		{
			ID:          1,
			ReferenceID: "AC1",
			Name:        "Test Control",
			Description: "Test control description",
			Category:    "Test Category",
			Status:      "implemented",
		},
	}

	var result strings.Builder
	formatter.addEnhancedControlSection(&result, task, controls)

	output := result.String()

	// Check that section is created
	if !strings.Contains(output, "## Related Controls") {
		t.Error("Expected 'Related Controls' section header")
	}

	// Check that existing control is included
	if !strings.Contains(output, "AC1 - Test Control") {
		t.Error("Expected control with reference ID and name")
	}

	if !strings.Contains(output, "Test control description") {
		t.Error("Expected control description")
	}

	// Check that missing control is handled gracefully
	if !strings.Contains(output, "Control ID: 999") {
		t.Error("Expected missing control to be noted")
	}

	if !strings.Contains(output, "Control details not available") {
		t.Error("Expected missing control message")
	}
}

func TestEvidenceTaskFormatter_AddEnhancedPolicySection(t *testing.T) {
	formatter := NewEvidenceTaskFormatter()

	task := &domain.EvidenceTask{
		ID:          123,
		Name:        "Access Control Task",
		Description: "Task related to access control",
		Framework:   "SOC2",
	}

	policies := []domain.Policy{
		{
			ID:          "policy1",
			Name:        "Access Control Policy",
			Description: "Policy for access control",
			Status:      "published",
			Framework:   "SOC2",
		},
		{
			ID:          "policy2",
			Name:        "Data Security Policy",
			Description: "Policy for data security",
			Status:      "published",
			Framework:   "SOC2",
		},
		{
			ID:          "policy3",
			Name:        "Network Security Policy",
			Description: "Policy for network security",
			Status:      "published",
			Framework:   "ISO27001", // Different framework
		},
	}

	var result strings.Builder
	formatter.addEnhancedPolicySection(&result, task, policies)

	output := result.String()

	// Check that section is created
	if !strings.Contains(output, "## Related Policies") {
		t.Error("Expected 'Related Policies' section header")
	}

	// Check that framework-matching policy is included
	if !strings.Contains(output, "Access Control Policy") {
		t.Error("Expected framework-matching policy")
	}

	// Check that other framework policy might be included too
	if !strings.Contains(output, "Data Security Policy") {
		t.Error("Expected other framework-matching policy")
	}
}

func TestEvidenceTaskFormatter_EnhanceDocumentWithContext(t *testing.T) {
	formatter := NewEvidenceTaskFormatter()

	baseContent := `# ET1 - Test Task

## Description
Test description

---

## Document Information
| Field | Value |`

	task := &domain.EvidenceTask{
		ID:       123,
		Name:     "Test Task",
		Controls: []string{"1"},
	}

	controls := []domain.Control{
		{
			ID:          1,
			ReferenceID: "AC1",
			Name:        "Test Control",
		},
	}

	policies := []domain.Policy{
		{
			ID:   "policy1",
			Name: "Test Policy",
		},
	}

	result := formatter.enhanceDocumentWithContext(baseContent, task, controls, policies)

	// Check that controls section was inserted before metadata
	controlsIndex := strings.Index(result, "## Related Controls")
	metadataIndex := strings.Index(result, "## Document Information")

	if controlsIndex == -1 {
		t.Error("Expected Related Controls section to be added")
	}

	if metadataIndex == -1 {
		t.Error("Expected Document Information section to be preserved")
	}

	if controlsIndex > metadataIndex {
		t.Error("Expected Related Controls section to come before Document Information")
	}

	// Test with malformed content (no metadata section)
	malformedContent := "# Test\nSome content"
	malformedResult := formatter.enhanceDocumentWithContext(malformedContent, task, controls, policies)

	// Should return original content when can't parse
	if malformedResult != malformedContent {
		t.Error("Expected malformed content to be returned unchanged")
	}
}
