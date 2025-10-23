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

func TestControlReferenceProcessor_ExtractReferenceID(t *testing.T) {
	processor := NewControlReferenceProcessor()

	tests := []struct {
		name              string
		controlName       string
		expectedRefID     string
		expectedCleanName string
		expectedHasPrefix bool
	}{
		{
			name:              "AC prefix with number",
			controlName:       "AC1 - Access Provisioning and Approval",
			expectedRefID:     "AC1",
			expectedCleanName: "Access Provisioning and Approval",
			expectedHasPrefix: true,
		},
		{
			name:              "AA prefix with number",
			controlName:       "AA2 - Unique IDs and Strong Passwords - Applications",
			expectedRefID:     "AA2",
			expectedCleanName: "Unique IDs and Strong Passwords - Applications",
			expectedHasPrefix: true,
		},
		{
			name:              "OM prefix with number",
			controlName:       "OM2 - Owners are involved in day-to-day operations",
			expectedRefID:     "OM2",
			expectedCleanName: "Owners are involved in day-to-day operations",
			expectedHasPrefix: true,
		},
		{
			name:              "Multi-letter prefix",
			controlName:       "CSM15 - Change Set Management",
			expectedRefID:     "CSM15",
			expectedCleanName: "Change Set Management",
			expectedHasPrefix: true,
		},
		{
			name:              "No prefix - regular control name",
			controlName:       "Security Awareness Training",
			expectedRefID:     "",
			expectedCleanName: "Security Awareness Training",
			expectedHasPrefix: false,
		},
		{
			name:              "No prefix - descriptive name",
			controlName:       "Owners are involved in day-to-day operations",
			expectedRefID:     "",
			expectedCleanName: "Owners are involved in day-to-day operations",
			expectedHasPrefix: false,
		},
		{
			name:              "Extra spaces around dash",
			controlName:       "AC1  -  Access Provisioning and Approval",
			expectedRefID:     "AC1",
			expectedCleanName: "Access Provisioning and Approval",
			expectedHasPrefix: true,
		},
		{
			name:              "Single letter prefix",
			controlName:       "A1 - Simple Control",
			expectedRefID:     "A1",
			expectedCleanName: "Simple Control",
			expectedHasPrefix: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refID, cleanName, hasPrefix := processor.ExtractReferenceID(tt.controlName)

			if refID != tt.expectedRefID {
				t.Errorf("ExtractReferenceID() refID = %v, expected %v", refID, tt.expectedRefID)
			}
			if cleanName != tt.expectedCleanName {
				t.Errorf("ExtractReferenceID() cleanName = %v, expected %v", cleanName, tt.expectedCleanName)
			}
			if hasPrefix != tt.expectedHasPrefix {
				t.Errorf("ExtractReferenceID() hasPrefix = %v, expected %v", hasPrefix, tt.expectedHasPrefix)
			}
		})
	}
}

func TestControlReferenceProcessor_ProcessControlReferences(t *testing.T) {
	processor := NewControlReferenceProcessor()

	// Create test controls
	controls := []Control{
		{
			ID:   1,
			Name: "AC1 - Access Provisioning and Approval",
		},
		{
			ID:   2,
			Name: "AA2 - Unique IDs and Strong Passwords - Applications",
		},
		{
			ID:   3,
			Name: "Security Awareness Training",
		},
		{
			ID:   4,
			Name: "Backup and Recovery Procedures",
		},
		{
			ID:   5,
			Name: "OM2 - Owners are involved in day-to-day operations",
		},
	}

	// Process the controls
	processedControls := processor.ProcessControlReferences(controls)

	// Verify we have the same number of controls
	if len(processedControls) != len(controls) {
		t.Errorf("ProcessControlReferences() returned %d controls, expected %d", len(processedControls), len(controls))
	}

	// Check that controls with existing reference IDs are processed correctly
	expectedResults := map[int]struct {
		refID string
		name  string
	}{
		1: {"AC1", "Access Provisioning and Approval"},
		2: {"AA2", "Unique IDs and Strong Passwords - Applications"},
		3: {"C2", "Security Awareness Training"},    // "Security Awareness Training" alphabetically after "Backup and Recovery Procedures"
		4: {"C1", "Backup and Recovery Procedures"}, // "Backup and Recovery Procedures" alphabetically first
		5: {"OM2", "Owners are involved in day-to-day operations"},
	}

	for _, control := range processedControls {
		expected, exists := expectedResults[control.ID]
		if !exists {
			t.Errorf("Unexpected control ID: %d", control.ID)
			continue
		}

		if control.ReferenceID != expected.refID {
			t.Errorf("Control %d: ReferenceID = %v, expected %v", control.ID, control.ReferenceID, expected.refID)
		}
		if control.Name != expected.name {
			t.Errorf("Control %d: Name = %v, expected %v", control.ID, control.Name, expected.name)
		}
	}
}

func TestControlReferenceProcessor_findNextCNumber(t *testing.T) {
	processor := NewControlReferenceProcessor()

	tests := []struct {
		name             string
		existingRefs     map[string]bool
		expectedNextCNum int
	}{
		{
			name:             "No existing C references",
			existingRefs:     map[string]bool{"AC1": true, "AA2": true},
			expectedNextCNum: 1,
		},
		{
			name:             "C1 exists",
			existingRefs:     map[string]bool{"AC1": true, "C1": true},
			expectedNextCNum: 2,
		},
		{
			name:             "C1 and C3 exist",
			existingRefs:     map[string]bool{"C1": true, "C3": true, "AC1": true},
			expectedNextCNum: 4,
		},
		{
			name:             "C1, C2, C5 exist",
			existingRefs:     map[string]bool{"C1": true, "C2": true, "C5": true},
			expectedNextCNum: 6,
		},
		{
			name:             "Empty references",
			existingRefs:     map[string]bool{},
			expectedNextCNum: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCNum := processor.findNextCNumber(tt.existingRefs)
			if nextCNum != tt.expectedNextCNum {
				t.Errorf("findNextCNumber() = %v, expected %v", nextCNum, tt.expectedNextCNum)
			}
		})
	}
}
