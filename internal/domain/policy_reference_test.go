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

package domain

import (
	"testing"
)

func TestPolicyReferenceProcessor_ProcessPolicyReferences(t *testing.T) {
	processor := NewPolicyReferenceProcessor()

	tests := []struct {
		name     string
		policies []Policy
		expected map[string]string // policy name -> expected reference ID
	}{
		{
			name: "policies with master_policy_id",
			policies: []Policy{
				{ID: "1", Name: "Acceptable Use", MasterPolicyID: "1"},
				{ID: "2", Name: "Access Control", MasterPolicyID: "36"},
				{ID: "3", Name: "Backup Policy", MasterPolicyID: "23"},
			},
			expected: map[string]string{
				"Acceptable Use": "P1",
				"Access Control": "P36",
				"Backup Policy":  "P23",
			},
		},
		{
			name: "policies without master_policy_id get auto-assigned",
			policies: []Policy{
				{ID: "1", Name: "Custom Policy A"},
				{ID: "2", Name: "Custom Policy B"},
			},
			expected: map[string]string{
				"Custom Policy A": "P1000", // Auto-assigned starting from 1000
				"Custom Policy B": "P1001",
			},
		},
		{
			name: "mixed policies with and without master_policy_id",
			policies: []Policy{
				{ID: "1", Name: "Acceptable Use", MasterPolicyID: "1"},
				{ID: "2", Name: "Custom Policy", MasterPolicyID: ""},
				{ID: "3", Name: "Access Control", MasterPolicyID: "36"},
			},
			expected: map[string]string{
				"Acceptable Use": "P1",
				"Access Control": "P36",
				"Custom Policy":  "P1000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.ProcessPolicyReferences(tt.policies)

			// Verify all policies got reference IDs
			if len(result) != len(tt.policies) {
				t.Errorf("Expected %d policies, got %d", len(tt.policies), len(result))
			}

			// Check specific reference ID assignments
			for _, policy := range result {
				expectedRef, exists := tt.expected[policy.Name]
				if !exists {
					t.Errorf("Unexpected policy in result: %s", policy.Name)
					continue
				}

				if policy.ReferenceID != expectedRef {
					t.Errorf("Policy %s: expected reference ID %s, got %s",
						policy.Name, expectedRef, policy.ReferenceID)
				}
			}
		})
	}
}

func TestPolicyReferenceProcessor_ValidateReferenceIDs(t *testing.T) {
	processor := NewPolicyReferenceProcessor()

	tests := []struct {
		name           string
		policies       []Policy
		expectedErrors int
	}{
		{
			name: "valid reference IDs",
			policies: []Policy{
				{ID: "1", Name: "Policy A", ReferenceID: "P1"},
				{ID: "2", Name: "Policy B", ReferenceID: "P36"},
			},
			expectedErrors: 0,
		},
		{
			name: "missing reference ID",
			policies: []Policy{
				{ID: "1", Name: "Policy A", ReferenceID: ""},
				{ID: "2", Name: "Policy B", ReferenceID: "P36"},
			},
			expectedErrors: 1,
		},
		{
			name: "invalid reference ID format",
			policies: []Policy{
				{ID: "1", Name: "Policy A", ReferenceID: "X1"},
				{ID: "2", Name: "Policy B", ReferenceID: "P36"},
			},
			expectedErrors: 1,
		},
		{
			name: "duplicate reference IDs",
			policies: []Policy{
				{ID: "1", Name: "Policy A", ReferenceID: "P1"},
				{ID: "2", Name: "Policy B", ReferenceID: "P1"},
			},
			expectedErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := processor.ValidateReferenceIDs(tt.policies)

			if len(errors) != tt.expectedErrors {
				t.Errorf("Expected %d errors, got %d: %v", tt.expectedErrors, len(errors), errors)
			}
		})
	}
}

func TestPolicyReferenceProcessor_GetMasterPolicyReferenceMap(t *testing.T) {
	processor := NewPolicyReferenceProcessor()

	policies := []Policy{
		{ID: "1", Name: "Acceptable Use", MasterPolicyID: "1", ReferenceID: "P1"},
		{ID: "2", Name: "Access Control", MasterPolicyID: "36", ReferenceID: "P36"},
		{ID: "3", Name: "Custom Policy", MasterPolicyID: "", ReferenceID: "P1000"},
	}

	mapping := processor.GetMasterPolicyReferenceMap(policies)

	expected := map[string]string{
		"1":  "P1",
		"36": "P36",
	}

	if len(mapping) != len(expected) {
		t.Errorf("Expected %d mappings, got %d", len(expected), len(mapping))
	}

	for masterID, expectedRef := range expected {
		if actualRef, exists := mapping[masterID]; !exists {
			t.Errorf("Missing mapping for master policy ID %s", masterID)
		} else if actualRef != expectedRef {
			t.Errorf("Master policy ID %s: expected reference %s, got %s",
				masterID, expectedRef, actualRef)
		}
	}
}

func TestPolicyReferenceProcessor_GetReferenceIDStats(t *testing.T) {
	processor := NewPolicyReferenceProcessor()

	policies := []Policy{
		{ID: "1", Name: "Acceptable Use", MasterPolicyID: "1", Category: "Organization"},
		{ID: "2", Name: "Access Control", MasterPolicyID: "36", Category: "Access Control"},
		{ID: "3", Name: "Custom Policy", MasterPolicyID: "", Category: "Organization"},
	}

	stats := processor.GetReferenceIDStats(policies)

	if stats.Total != 3 {
		t.Errorf("Expected total 3, got %d", stats.Total)
	}

	if stats.WithMasterPolicyID != 2 {
		t.Errorf("Expected 2 with master policy ID, got %d", stats.WithMasterPolicyID)
	}

	if stats.AutoAssigned != 1 {
		t.Errorf("Expected 1 auto-assigned, got %d", stats.AutoAssigned)
	}

	if stats.ByCategory["Organization"] != 2 {
		t.Errorf("Expected 2 Organization policies, got %d", stats.ByCategory["Organization"])
	}

	if stats.ByCategory["Access Control"] != 1 {
		t.Errorf("Expected 1 Access Control policy, got %d", stats.ByCategory["Access Control"])
	}
}
