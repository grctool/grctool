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

//go:build integration
// +build integration

package tugboat

import (
	"context"
	"encoding/json"
	"testing"
)

func TestFinalRelationshipImplementation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	client := setupTestClient(t, "final_relationship_test")
	defer client.Close()

	// Test evidence task with control relationships
	evidenceTaskID := "327992" // ET1 - Access Control Registration

	t.Run("EvidenceTaskWithControls", func(t *testing.T) {
		taskDetails, err := client.GetEvidenceTaskDetails(ctx, evidenceTaskID)
		if err != nil {
			t.Fatalf("Failed to get evidence task details: %v", err)
		}

		t.Logf("Evidence Task: %s", taskDetails.Name)

		// Check if the org_controls embed worked
		if taskDetails.Controls != nil {
			controlsJSON, _ := json.Marshal(taskDetails.Controls)
			t.Logf("✓ Controls field populated: %s", string(controlsJSON))
		} else {
			t.Logf("✗ Controls field is null")
		}
	})

	// Test control with evidence metrics
	controlID := "778805" // AA1 - Known to have evidence

	t.Run("ControlWithEvidenceMetrics", func(t *testing.T) {
		controlDetails, err := client.GetControlDetails(ctx, controlID)
		if err != nil {
			t.Fatalf("Failed to get control details: %v", err)
		}

		t.Logf("Control: %s", controlDetails.Name)

		if controlDetails.OrgEvidenceCount != nil {
			t.Logf("✓ Evidence count: %d", *controlDetails.OrgEvidenceCount)
		} else {
			t.Logf("✗ Evidence count field is null")
		}

		if controlDetails.OrgEvidenceCollectedCount != nil {
			t.Logf("✓ Evidence collected count: %d", *controlDetails.OrgEvidenceCollectedCount)
		}

		if controlDetails.OrgEvidenceLastCollected != nil {
			t.Logf("✓ Evidence last collected: %s", *controlDetails.OrgEvidenceLastCollected)
		}
	})

	// Test getting evidence tasks by control
	t.Run("EvidenceTasksByControl", func(t *testing.T) {
		evidenceTasks, err := client.GetEvidenceTasksByControl(ctx, controlID, "13888")
		if err != nil {
			t.Fatalf("Failed to get evidence tasks by control: %v", err)
		}

		t.Logf("✓ Found %d evidence tasks for control %s", len(evidenceTasks), controlID)

		for i, task := range evidenceTasks {
			t.Logf("  Evidence Task %d: %s (ID: %d)", i+1, task.Name, task.ID)
		}
	})
}
