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

package tugboat

import (
	"context"
	"fmt"
	"testing"
)

func TestControlEvidenceMapping(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	client := setupTestClient(t, "control_evidence_mapping")
	defer client.Close()

	// Test control that shows evidence count > 0 in VCR data
	controlID := "778805" // AA1 - Has org_evidence_count: 2

	t.Run("GetControlWithEvidenceMetrics", func(t *testing.T) {
		controlDetails, err := client.GetControlDetails(ctx, controlID)
		if err != nil {
			t.Fatalf("Failed to get control details: %v", err)
		}

		t.Logf("Control ID: %d", controlDetails.ID)
		t.Logf("Control Name: %s", controlDetails.Name)

		// Check if OrgEvidenceMetrics field contains the evidence information
		if controlDetails.OrgEvidenceMetrics != nil {
			t.Logf("Org Evidence Metrics: %+v", controlDetails.OrgEvidenceMetrics)
		} else {
			t.Logf("No OrgEvidenceMetrics found")
		}

		// Check associations for evidence count
		if controlDetails.Associations != nil {
			t.Logf("Evidence count in associations: %d", controlDetails.Associations.Evidence)
		}
	})

	// Test potential endpoint patterns for getting evidence by control
	potentialEndpoints := []string{
		fmt.Sprintf("/api/org_control/%s/org_evidence/", controlID),
		fmt.Sprintf("/api/org_control/%s/evidence/", controlID),
		fmt.Sprintf("/api/org_evidence/?control_id=%s", controlID),
		fmt.Sprintf("/api/org_evidence/?org_control=%s", controlID),
		fmt.Sprintf("/api/org_evidence/?control=%s", controlID),
		fmt.Sprintf("/api/org_evidence/?controls=%s", controlID),
		"/api/org_evidence/?embeds=org_control",
		"/api/org_evidence/?embeds=org_controls",
		"/api/org_evidence/?embeds=control",
		"/api/org_evidence/?embeds=controls",
	}

	for i, endpoint := range potentialEndpoints {
		t.Run(fmt.Sprintf("TestEndpoint_%d", i), func(t *testing.T) {
			resp, err := client.makeRequest(ctx, "GET", endpoint, nil)
			if err != nil {
				t.Logf("Endpoint %s - Request failed: %v", endpoint, err)
				return
			}
			defer resp.Body.Close()

			t.Logf("Endpoint %s - Status: %d", endpoint, resp.StatusCode)

			switch resp.StatusCode {
			case 200:
				t.Logf("✓ Endpoint %s exists and returned 200!", endpoint)
				// Response received successfully
			case 404:
				t.Logf("✗ Endpoint %s not found (404)", endpoint)
			default:
				t.Logf("? Endpoint %s returned %d", endpoint, resp.StatusCode)
			}
		})
	}
}
