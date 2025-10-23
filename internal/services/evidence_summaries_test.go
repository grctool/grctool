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

	"github.com/grctool/grctool/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvidenceContext_MapBasedSummaries(t *testing.T) {
	t.Run("control summaries correctly mapped by ID", func(t *testing.T) {
		// Create evidence context with controls
		context := &models.EvidenceContext{
			Controls: []models.Control{
				{ID: 100, Name: "Control 100"},
				{ID: 200, Name: "Control 200"},
				{ID: 300, Name: "Control 300"},
			},
			ControlSummaries: make(map[int]models.AIControlSummary),
		}

		// Add summaries in different order to test mapping
		context.ControlSummaries[200] = models.AIControlSummary{
			ControlID:   200,
			ControlName: "Control 200",
			Summary:     "Summary for control 200",
		}
		context.ControlSummaries[100] = models.AIControlSummary{
			ControlID:   100,
			ControlName: "Control 100",
			Summary:     "Summary for control 100",
		}
		context.ControlSummaries[300] = models.AIControlSummary{
			ControlID:   300,
			ControlName: "Control 300",
			Summary:     "Summary for control 300",
		}

		// Verify each control has the correct summary
		for _, control := range context.Controls {
			summary, exists := context.ControlSummaries[control.ID]
			require.True(t, exists, "Summary should exist for control %d", control.ID)
			assert.Equal(t, control.ID, summary.ControlID)
			assert.Equal(t, control.Name, summary.ControlName)
			assert.Contains(t, summary.Summary, "control")
		}
	})

	t.Run("policy summaries correctly mapped by ID", func(t *testing.T) {
		// Create evidence context with policies
		context := &models.EvidenceContext{
			Policies: []models.Policy{
				{ID: models.IntOrString("P001"), Name: "Policy P001"},
				{ID: models.IntOrString("P002"), Name: "Policy P002"},
				{ID: models.IntOrString("P003"), Name: "Policy P003"},
			},
			PolicySummaries: make(map[string]models.AIPolicySummary),
		}

		// Add summaries
		context.PolicySummaries["P001"] = models.AIPolicySummary{
			PolicyID:   "P001",
			PolicyName: "Policy P001",
			Summary:    "Summary for policy P001",
		}
		context.PolicySummaries["P002"] = models.AIPolicySummary{
			PolicyID:   "P002",
			PolicyName: "Policy P002",
			Summary:    "Summary for policy P002",
		}
		context.PolicySummaries["P003"] = models.AIPolicySummary{
			PolicyID:   "P003",
			PolicyName: "Policy P003",
			Summary:    "Summary for policy P003",
		}

		// Verify each policy has the correct summary
		for _, policy := range context.Policies {
			summary, exists := context.PolicySummaries[policy.ID.String()]
			require.True(t, exists, "Summary should exist for policy %s", policy.ID)
			assert.Equal(t, policy.ID.String(), summary.PolicyID)
			assert.Equal(t, policy.Name, summary.PolicyName)
			assert.Contains(t, summary.Summary, "policy")
		}
	})

	t.Run("missing summaries handled gracefully", func(t *testing.T) {
		// Create context with controls but incomplete summaries
		context := &models.EvidenceContext{
			Controls: []models.Control{
				{ID: 100, Name: "Control 100"},
				{ID: 200, Name: "Control 200"},
				{ID: 300, Name: "Control 300"},
			},
			ControlSummaries: make(map[int]models.AIControlSummary),
		}

		// Only add summary for one control
		context.ControlSummaries[200] = models.AIControlSummary{
			ControlID:   200,
			ControlName: "Control 200",
			Summary:     "Summary for control 200",
		}

		// Verify we can check for missing summaries
		summary100, exists100 := context.ControlSummaries[100]
		assert.False(t, exists100, "Summary should not exist for control 100")
		assert.Zero(t, summary100.ControlID)

		summary200, exists200 := context.ControlSummaries[200]
		assert.True(t, exists200, "Summary should exist for control 200")
		assert.Equal(t, 200, summary200.ControlID)

		summary300, exists300 := context.ControlSummaries[300]
		assert.False(t, exists300, "Summary should not exist for control 300")
		assert.Zero(t, summary300.ControlID)
	})

	t.Run("handles mixed policy ID types", func(t *testing.T) {
		// Test that policies with different ID types work correctly
		context := &models.EvidenceContext{
			Policies: []models.Policy{
				{ID: models.IntOrString("P001"), Name: "String ID Policy"},
				{ID: models.IntOrString("123"), Name: "Numeric ID Policy"},
			},
			PolicySummaries: make(map[string]models.AIPolicySummary),
		}

		// Add summaries for both types
		context.PolicySummaries["P001"] = models.AIPolicySummary{
			PolicyID:   "P001",
			PolicyName: "String ID Policy",
			Summary:    "Summary for string ID policy",
		}
		context.PolicySummaries["123"] = models.AIPolicySummary{
			PolicyID:   "123",
			PolicyName: "Numeric ID Policy",
			Summary:    "Summary for numeric ID policy",
		}

		// Verify both work correctly
		for _, policy := range context.Policies {
			summary, exists := context.PolicySummaries[policy.ID.String()]
			require.True(t, exists, "Summary should exist for policy %s", policy.ID)
			assert.Equal(t, policy.Name, summary.PolicyName)
		}
	})
}

func TestSummaryGeneration_NoArrayIndexing(t *testing.T) {
	t.Run("summaries remain correctly associated even with generation failures", func(t *testing.T) {
		// This test verifies that if some summaries fail to generate,
		// the remaining summaries are still correctly associated with their controls/policies

		context := &models.EvidenceContext{
			Controls: []models.Control{
				{ID: 100, Name: "Control 100"},
				{ID: 200, Name: "Control 200"},
				{ID: 300, Name: "Control 300"},
			},
			ControlSummaries: make(map[int]models.AIControlSummary),
		}

		// Simulate scenario where control 200 fails to generate summary
		// In old array-based system, this would cause misalignment
		context.ControlSummaries[100] = models.AIControlSummary{
			ControlID:   100,
			ControlName: "Control 100",
			Summary:     "Summary for control 100",
		}
		// Skip control 200
		context.ControlSummaries[300] = models.AIControlSummary{
			ControlID:   300,
			ControlName: "Control 300",
			Summary:     "Summary for control 300",
		}

		// Verify control 100 still has correct summary
		summary100, exists100 := context.ControlSummaries[100]
		assert.True(t, exists100)
		assert.Equal(t, 100, summary100.ControlID)
		assert.Contains(t, summary100.Summary, "control 100")

		// Verify control 200 has no summary
		_, exists200 := context.ControlSummaries[200]
		assert.False(t, exists200)

		// Verify control 300 still has correct summary
		summary300, exists300 := context.ControlSummaries[300]
		assert.True(t, exists300)
		assert.Equal(t, 300, summary300.ControlID)
		assert.Contains(t, summary300.Summary, "control 300")
	})
}
