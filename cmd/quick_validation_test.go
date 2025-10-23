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

package cmd

import (
	"testing"

	"github.com/grctool/grctool/internal/adapters"
	tugboatmodels "github.com/grctool/grctool/internal/tugboat/models"
)

// TestDataMappingImprovements verifies our adapter fixes work correctly
func TestDataMappingImprovements(t *testing.T) {
	adapter := adapters.NewTugboatToDomain()

	t.Run("Policy_Content_Extraction", func(t *testing.T) {
		// Simulate API response with summary and details
		apiPolicy := tugboatmodels.PolicyDetails{
			Summary: "This is a policy summary with important security guidelines.",
			Details: "<h1>Detailed Policy Content</h1><p>This policy establishes comprehensive security requirements...</p>",
		}
		apiPolicy.ID = tugboatmodels.IntOrString("12345")
		apiPolicy.Name = "Test Security Policy"

		// Convert using our fixed adapter
		domainPolicy := adapter.ConvertPolicy(apiPolicy)

		// Verify content extraction
		if domainPolicy.Description == "" {
			t.Error("Policy description should be mapped from summary field")
		}
		if domainPolicy.Content == "" {
			t.Error("Policy content should be mapped from details field")
		}
		if len(domainPolicy.Content) < 50 {
			t.Error("Policy content should contain substantial content from details field")
		}

		t.Logf("✅ Policy content extraction working: Description=%d chars, Content=%d chars",
			len(domainPolicy.Description), len(domainPolicy.Content))
	})

	t.Run("Evidence_Task_Guidance_Extraction", func(t *testing.T) {
		// Simulate API response with master_content guidance
		apiTask := tugboatmodels.EvidenceTaskDetails{
			MasterContent: &tugboatmodels.MasterContent{
				Guidance: "**Evidence Collection Guidelines**\n1. Collect access logs\n2. Verify authentication mechanisms\n3. Document access control procedures",
			},
		}
		apiTask.ID = 98765
		apiTask.Name = "Access Control Evidence"
		apiTask.Description = "Collect evidence for access control implementation"

		// Convert using our fixed adapter
		domainTask := adapter.ConvertEvidenceTask(apiTask)

		// Verify guidance extraction
		if domainTask.Guidance == "" {
			t.Error("Evidence task guidance should be extracted from master_content")
		}
		if len(domainTask.Guidance) < 20 {
			t.Error("Evidence task guidance should contain substantial guidance content")
		}

		t.Logf("✅ Evidence task guidance extraction working: Guidance=%d chars",
			len(domainTask.Guidance))
	})

	t.Run("Type_Safety_Improvements", func(t *testing.T) {
		// Test that our interface{} fixes handle different data types
		apiTask := tugboatmodels.EvidenceTaskDetails{
			AecStatus: "na", // String type
			Assignees: []tugboatmodels.EvidenceAssignee{
				{
					ID:   57888, // Numeric ID
					Name: "Test User",
				},
			},
		}
		apiTask.ID = 11111
		apiTask.Name = "Type Safety Test"

		// This should not panic or fail due to type mismatches
		domainTask := adapter.ConvertEvidenceTask(apiTask)

		if domainTask.AecStatus == nil {
			t.Error("AecStatus should be converted from string")
		}
		if domainTask.AecStatus.Status != "na" {
			t.Error("AecStatus should preserve string value")
		}
		if len(domainTask.Assignees) == 0 {
			t.Error("Assignees should be converted despite numeric ID")
		}

		t.Logf("✅ Type safety improvements working: AecStatus=%s, Assignees=%d",
			domainTask.AecStatus.Status, len(domainTask.Assignees))
	})
}
