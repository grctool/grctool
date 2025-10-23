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
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/domain"
)

func TestEvidenceTaskFormatterWithControlDetails(t *testing.T) {
	// Create a sample evidence task with control relationships
	evidenceTask := &domain.EvidenceTask{
		ID:                 327992,
		ReferenceID:        "ET1",
		Name:               "Access Control / Registration and De-registration Process Document",
		Description:        "Provide user registration and de-registration process document/Access Control policy or procedure reviewed by management.",
		Framework:          "SOC2",
		Priority:           "High",
		Status:             "In Progress",
		CollectionInterval: "Annual",
		Controls:           []string{"778771", "778805"}, // Control IDs this evidence task relates to
		CreatedAt:          time.Date(2023, 1, 15, 10, 0, 0, 0, time.UTC),
		UpdatedAt:          time.Date(2024, 6, 30, 14, 30, 0, 0, time.UTC),
	}

	// Create related controls
	controls := []domain.Control{
		{
			ID:          778771,
			ReferenceID: "AC1",
			Name:        "Access Provisioning and Approval",
			Description: "Access to in-scope system components requires documented access request and approval from management prior to access provisioning.",
			Category:    "Access Control",
			Status:      "Implemented",
			Framework:   "SOC2",
			MasterContent: &domain.ControlMasterContent{
				Guidance: "Establish processes to control the unique identities of users and services.",
			},
		},
		{
			ID:          778805,
			ReferenceID: "AA1",
			Name:        "Unique IDs and Strong Passwords - Infrastructure",
			Description: "Unique user IDs and strong passwords are required to gain access to infrastructure supporting the application.",
			Category:    "Access Authentication",
			Status:      "Implemented",
			Framework:   "SOC2",
			MasterContent: &domain.ControlMasterContent{
				Guidance: "Enforce the use of individual user IDs and strong passwords to maintain accountability.",
			},
		},
	}

	// Create formatter and set control details
	formatter := NewEvidenceTaskFormatter()
	formatter.SetRelatedControls(controls)

	t.Run("ToMarkdownWithControlDetails", func(t *testing.T) {
		markdown := formatter.ToMarkdown(evidenceTask)

		// Verify the markdown contains control details
		if !strings.Contains(markdown, "### Related Controls") {
			t.Error("Expected markdown to contain 'Related Controls' section")
		}

		// Check that detailed control information is included
		if !strings.Contains(markdown, "AC1 - Access Provisioning and Approval") {
			t.Error("Expected control AC1 details to be included")
		}

		if !strings.Contains(markdown, "AA1 - Unique IDs and Strong Passwords") {
			t.Error("Expected control AA1 details to be included")
		}

		// Check control descriptions are included
		if !strings.Contains(markdown, "Access to in-scope system components") {
			t.Error("Expected control AC1 description to be included")
		}

		if !strings.Contains(markdown, "Unique user IDs and strong passwords") {
			t.Error("Expected control AA1 description to be included")
		}

		// Check control metadata
		if !strings.Contains(markdown, "**Category**: Access Control") {
			t.Error("Expected control category to be included")
		}

		if !strings.Contains(markdown, "**Status**: Implemented") {
			t.Error("Expected control status to be included")
		}

		t.Logf("Generated markdown with control details:\n%s", markdown)
	})

	t.Run("ToDocumentMarkdownWithControlDetails", func(t *testing.T) {
		markdown := formatter.ToDocumentMarkdown(evidenceTask)

		// Verify the document contains comprehensive control information
		if !strings.Contains(markdown, "### Related Controls") {
			t.Error("Expected document to contain 'Related Controls' section")
		}

		// Check that comprehensive control information is included
		if !strings.Contains(markdown, "#### AC1 - Access Provisioning and Approval") {
			t.Error("Expected control AC1 comprehensive details to be included")
		}

		// Check control guidance is included
		if !strings.Contains(markdown, "**Implementation Guidance**") {
			t.Error("Expected implementation guidance section")
		}

		if !strings.Contains(markdown, "Establish processes to control") {
			t.Error("Expected control guidance content to be included")
		}

		// Check control metadata table
		if !strings.Contains(markdown, "**Control Details**:") {
			t.Error("Expected control details section")
		}

		if !strings.Contains(markdown, "- **Category**: Access Control") {
			t.Error("Expected control category in details")
		}

		if !strings.Contains(markdown, "- **Implementation Status**: Implemented") {
			t.Error("Expected control status in details")
		}

		t.Logf("Generated document with control details:\n%s", markdown)
	})

	t.Run("FallbackToControlIDsWhenNoDetails", func(t *testing.T) {
		// Create formatter without control details
		formatterNoControls := NewEvidenceTaskFormatter()

		// Create evidence task with controls that don't have details
		taskWithUnknownControls := &domain.EvidenceTask{
			ID:       12345,
			Name:     "Test Task",
			Controls: []string{"999999", "888888"}, // Unknown control IDs
		}

		markdown := formatterNoControls.ToMarkdown(taskWithUnknownControls)

		// Should fall back to showing just the control IDs
		if !strings.Contains(markdown, "Control ID: 999999") {
			t.Error("Expected fallback to control ID when details not available")
		}

		if !strings.Contains(markdown, "Control ID: 888888") {
			t.Error("Expected fallback to control ID when details not available")
		}
	})
}

func TestEvidenceTaskFormatterControlRelationshipExamples(t *testing.T) {
	// This test demonstrates the actual relationships we discovered

	// Example: Control AA1 has multiple evidence tasks
	relatedEvidenceTasks := []string{
		"327992", // Access Control / Registration and De-registration Process Document
		"327993", // Population 2 - List of Employees and Contractors
		"327994", // User Access Approval List to Application, Infrastructure and Service
		"327995", // Population 15 - List of Users with Access to Application and Infrastructure
		"327996", // Configuration for Application Password
		"327997", // Enabled Multi-Factor Authentication
	}

	control := domain.Control{
		ID:          778805,
		ReferenceID: "AA1",
		Name:        "Unique IDs and Strong Passwords - Infrastructure",
		Description: "Unique user IDs and strong passwords are required to gain access to infrastructure supporting the application.",
		Category:    "Access Authentication",
		Status:      "Implemented",
	}

	formatter := NewEvidenceTaskFormatter()
	formatter.SetRelatedControls([]domain.Control{control})

	// Test each evidence task shows the control relationship
	for i, taskID := range relatedEvidenceTasks {
		evidenceTask := &domain.EvidenceTask{
			ID:       327992 + i, // Incremental IDs
			Name:     fmt.Sprintf("Evidence Task %d", i+1),
			Controls: []string{"778805"}, // All relate to AA1 control
		}

		markdown := formatter.ToMarkdown(evidenceTask)

		if !strings.Contains(markdown, "AA1 - Unique IDs and Strong Passwords") {
			t.Errorf("Evidence task %s should show relationship to control AA1", taskID)
		}

		if !strings.Contains(markdown, "Access Authentication") {
			t.Errorf("Evidence task %s should show control category", taskID)
		}
	}

	t.Logf("Successfully tested %d evidence tasks with control relationships", len(relatedEvidenceTasks))
}
