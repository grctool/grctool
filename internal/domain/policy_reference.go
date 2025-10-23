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
	"fmt"
	"sort"
)

// PolicyReferenceProcessor handles the assignment of reference IDs to policies
type PolicyReferenceProcessor struct{}

// NewPolicyReferenceProcessor creates a new policy reference processor
func NewPolicyReferenceProcessor() *PolicyReferenceProcessor {
	return &PolicyReferenceProcessor{}
}

// ProcessPolicyReferences assigns reference IDs to policies based on master_policy_id
// or auto-assigns sequential IDs for policies without master_policy_id
func (p *PolicyReferenceProcessor) ProcessPolicyReferences(policies []Policy) []Policy {
	processedPolicies := make([]Policy, len(policies))
	copy(processedPolicies, policies)

	// Track used reference IDs to avoid duplicates
	usedRefs := make(map[string]bool)
	policiesWithoutRefs := make([]*Policy, 0)

	// First pass: assign reference IDs based on master_policy_id
	for i := range processedPolicies {
		policy := &processedPolicies[i]

		if policy.MasterPolicyID != "" {
			// Use master policy ID as the reference (e.g., "P1", "P36")
			refID := fmt.Sprintf("P%s", policy.MasterPolicyID)

			// Check for duplicates (shouldn't happen but be safe)
			if !usedRefs[refID] {
				policy.ReferenceID = refID
				usedRefs[refID] = true
			} else {
				// Fallback for duplicates
				policiesWithoutRefs = append(policiesWithoutRefs, policy)
			}
		} else {
			// No master policy ID, will auto-assign later
			policiesWithoutRefs = append(policiesWithoutRefs, policy)
		}
	}

	// Second pass: auto-assign sequential IDs for policies without master_policy_id
	if len(policiesWithoutRefs) > 0 {
		// Sort policies without refs by name for consistent assignment
		sort.Slice(policiesWithoutRefs, func(i, j int) bool {
			return policiesWithoutRefs[i].Name < policiesWithoutRefs[j].Name
		})

		// Find the next available P number
		nextPNumber := p.findNextAvailablePNumber(usedRefs)

		for _, policy := range policiesWithoutRefs {
			refID := fmt.Sprintf("P%d", nextPNumber)
			policy.ReferenceID = refID
			usedRefs[refID] = true
			nextPNumber++
		}
	}

	return processedPolicies
}

// findNextAvailablePNumber finds the next available P number that's not already used
func (p *PolicyReferenceProcessor) findNextAvailablePNumber(usedRefs map[string]bool) int {
	// Start from P1000 to avoid conflicts with master policy IDs
	// Master policy IDs are typically low numbers (1-100)
	startNum := 1000

	for {
		refID := fmt.Sprintf("P%d", startNum)
		if !usedRefs[refID] {
			return startNum
		}
		startNum++
	}
}

// GetMasterPolicyReferenceMap returns a mapping of master policy IDs to reference IDs
// This can be useful for understanding the reference assignment pattern
func (p *PolicyReferenceProcessor) GetMasterPolicyReferenceMap(policies []Policy) map[string]string {
	mapping := make(map[string]string)

	for _, policy := range policies {
		if policy.MasterPolicyID != "" && policy.ReferenceID != "" {
			mapping[policy.MasterPolicyID] = policy.ReferenceID
		}
	}

	return mapping
}

// ValidateReferenceIDs checks for duplicate or invalid reference IDs
func (p *PolicyReferenceProcessor) ValidateReferenceIDs(policies []Policy) []string {
	var errors []string
	usedRefs := make(map[string][]string) // refID -> policy IDs using it

	for _, policy := range policies {
		if policy.ReferenceID == "" {
			errors = append(errors, fmt.Sprintf("Policy %s (%s) has no reference ID", policy.ID, policy.Name))
			continue
		}

		// Check for valid format (P followed by numbers)
		if len(policy.ReferenceID) < 2 || policy.ReferenceID[0] != 'P' {
			errors = append(errors, fmt.Sprintf("Policy %s (%s) has invalid reference ID format: %s", policy.ID, policy.Name, policy.ReferenceID))
		}

		// Check for duplicates
		usedRefs[policy.ReferenceID] = append(usedRefs[policy.ReferenceID], policy.ID)
	}

	// Report duplicates
	for refID, policyIDs := range usedRefs {
		if len(policyIDs) > 1 {
			errors = append(errors, fmt.Sprintf("Duplicate reference ID %s used by policies: %v", refID, policyIDs))
		}
	}

	return errors
}

// GetReferenceIDStats returns statistics about reference ID assignment
func (p *PolicyReferenceProcessor) GetReferenceIDStats(policies []Policy) PolicyReferenceStats {
	stats := PolicyReferenceStats{
		Total:              len(policies),
		WithMasterPolicyID: 0,
		AutoAssigned:       0,
		ByCategory:         make(map[string]int),
	}

	for _, policy := range policies {
		if policy.MasterPolicyID != "" {
			stats.WithMasterPolicyID++
		} else {
			stats.AutoAssigned++
		}

		if policy.Category != "" {
			stats.ByCategory[policy.Category]++
		}
	}

	return stats
}

// PolicyReferenceStats provides statistics about policy reference assignment
type PolicyReferenceStats struct {
	Total              int            `json:"total"`
	WithMasterPolicyID int            `json:"with_master_policy_id"`
	AutoAssigned       int            `json:"auto_assigned"`
	ByCategory         map[string]int `json:"by_category"`
}
