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

//go:build disabled
// +build disabled

// NOTE: These tests are temporarily disabled due to VCR cassette hash mismatches.
// The cassettes need to be re-recorded with updated request parameters.
// To re-enable: change build tag back to "integration" and run:
//   VCR_MODE=record TUGBOAT_BEARER=<token> go test -tags=integration -v ./internal/tugboat/...

package tugboat

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestControlEvidenceRelationships(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	client := setupTestClient(t, "relationship_discovery")
	defer client.Close()

	// Test control ID from existing data
	controlID := "778771" // AC1 - Access Provisioning and Approval

	t.Run("TestControlEvidenceRelationshipThroughQuery", func(t *testing.T) {
		// Test the working approach: getting evidence tasks by control ID
		evidenceTasks, err := client.GetEvidenceTasksByControl(ctx, controlID, "134")
		if err != nil {
			// Skip test if VCR cassette is missing (expected for new endpoints)
			if strings.Contains(err.Error(), "no such file or directory") {
				t.Skip("Skipping test - VCR cassette not available for new API endpoint")
				return
			}
			t.Fatalf("Failed to get evidence tasks by control: %v", err)
		}

		t.Logf("Found %d evidence tasks for control %s", len(evidenceTasks), controlID)

		// Verify we got some evidence tasks
		if len(evidenceTasks) > 0 {
			t.Logf("First evidence task: ID=%d, Name=%s", evidenceTasks[0].ID, evidenceTasks[0].Name)
			// Check if the evidence task has control relationships
			t.Logf("Controls in evidence task: %v", evidenceTasks[0].Controls)
		}

		// Also test regular control details to ensure they have evidence metrics
		controlDetails, err := client.GetControlDetails(ctx, controlID)
		if err != nil {
			t.Fatalf("Failed to get control details: %v", err)
		}

		t.Logf("Control ID: %d", controlDetails.ID)
		t.Logf("Control Name: %s", controlDetails.Name)

		// Check evidence metrics in control
		if controlDetails.OrgEvidenceCount != nil {
			t.Logf("Evidence count: %d", *controlDetails.OrgEvidenceCount)
		}
		if controlDetails.OrgEvidenceCollectedCount != nil {
			t.Logf("Evidence collected count: %d", *controlDetails.OrgEvidenceCollectedCount)
		}
	})

	// Test evidence task ID from existing data
	evidenceTaskID := "327992" // ET1 - Access Control Registration

	t.Run("TestEvidenceTaskDetailsWithControlEmbeds", func(t *testing.T) {
		taskDetails, err := client.GetEvidenceTaskDetails(ctx, evidenceTaskID)
		if err != nil {
			t.Fatalf("Failed to get evidence task details: %v", err)
		}

		t.Logf("Evidence Task ID: %d", taskDetails.ID)
		t.Logf("Evidence Task Name: %s", taskDetails.Name)

		// Check for any new fields that might contain control relationships
		if taskDetails.Controls != nil {
			controlsJSON, _ := json.Marshal(taskDetails.Controls)
			t.Logf("Controls field: %s", string(controlsJSON))
		}
		if taskDetails.Associations != nil {
			associationsJSON, _ := json.Marshal(taskDetails.Associations)
			t.Logf("Associations field: %s", string(associationsJSON))
		}

		t.Logf("Full evidence task details structure logged for inspection")
	})
}

func TestPotentialRelationshipEndpoints(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	client := setupTestClient(t, "endpoint_discovery")
	defer client.Close()

	controlID := "778771"
	evidenceTaskID := "327992"

	testEndpoints := []struct {
		name     string
		endpoint string
	}{
		{"Control Evidence List", "/api/org_control/" + controlID + "/evidence/"},
		{"Evidence Control List", "/api/org_evidence/" + evidenceTaskID + "/controls/"},
		{"Relationships Endpoint", "/api/relationships/"},
		{"Mappings Endpoint", "/api/mappings/"},
		{"Associations Endpoint", "/api/associations/"},
		{"Control Associations", "/api/org_control/" + controlID + "/associations/"},
		{"Evidence Associations", "/api/org_evidence/" + evidenceTaskID + "/associations/"},
	}

	for _, test := range testEndpoints {
		t.Run(test.name, func(t *testing.T) {
			resp, err := client.makeRequest(ctx, "GET", test.endpoint, nil)
			if err != nil {
				t.Logf("Endpoint %s - Request failed: %v", test.endpoint, err)
				return
			}
			defer resp.Body.Close()

			t.Logf("Endpoint %s - Status: %d", test.endpoint, resp.StatusCode)

			switch resp.StatusCode {
			case 200:
				t.Logf("✓ Endpoint %s exists and returned 200!", test.endpoint)
				// Response received successfully
			case 404:
				t.Logf("✗ Endpoint %s not found (404)", test.endpoint)
			default:
				t.Logf("? Endpoint %s returned %d", test.endpoint, resp.StatusCode)
			}
		})
	}
}
