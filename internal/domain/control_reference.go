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
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ControlReferenceProcessor handles extraction and assignment of control reference IDs
type ControlReferenceProcessor struct {
	// Regex pattern to match alphanumeric control reference prefixes like AC1, AA2, OM2, etc.
	referencePattern *regexp.Regexp
}

// NewControlReferenceProcessor creates a new processor for control reference IDs
func NewControlReferenceProcessor() *ControlReferenceProcessor {
	// Pattern matches: start of string, 1-3 letters, 1-3 digits, followed by space and dash
	pattern := regexp.MustCompile(`^([A-Z]{1,3}\d{1,3})\s*-\s*(.+)$`)
	return &ControlReferenceProcessor{
		referencePattern: pattern,
	}
}

// ExtractReferenceID extracts the reference ID from a control name if present
// Returns: referenceID, cleanName, hasPrefix
func (p *ControlReferenceProcessor) ExtractReferenceID(controlName string) (string, string, bool) {
	matches := p.referencePattern.FindStringSubmatch(strings.TrimSpace(controlName))
	if len(matches) == 3 {
		referenceID := strings.TrimSpace(matches[1])
		cleanName := strings.TrimSpace(matches[2])
		return referenceID, cleanName, true
	}
	return "", controlName, false
}

// ProcessControlReferences processes a slice of controls to assign reference IDs
// For controls with prefixed reference IDs (like AC1, AA2), extracts them
// For controls without prefixes, assigns C + sequential number in alphanumeric order
func (p *ControlReferenceProcessor) ProcessControlReferences(controls []Control) []Control {
	processedControls := make([]Control, len(controls))
	copy(processedControls, controls)

	// Track existing reference IDs and controls without reference IDs
	existingRefs := make(map[string]bool)
	controlsWithoutRefs := make([]*Control, 0)

	// First pass: extract existing reference IDs and clean names
	for i := range processedControls {
		control := &processedControls[i]
		refID, cleanName, hasPrefix := p.ExtractReferenceID(control.Name)

		if hasPrefix {
			control.ReferenceID = refID
			control.Name = cleanName
			existingRefs[refID] = true
		} else {
			controlsWithoutRefs = append(controlsWithoutRefs, control)
		}
	}

	// Second pass: assign C-prefixed reference IDs to controls without them
	if len(controlsWithoutRefs) > 0 {
		// Sort controls without references by name for consistent ordering
		sort.Slice(controlsWithoutRefs, func(i, j int) bool {
			return strings.ToLower(controlsWithoutRefs[i].Name) < strings.ToLower(controlsWithoutRefs[j].Name)
		})

		// Find the next available C number
		nextCNumber := p.findNextCNumber(existingRefs)

		// Assign C-prefixed reference IDs
		for _, control := range controlsWithoutRefs {
			refID := fmt.Sprintf("C%d", nextCNumber)
			// Ensure we don't conflict with existing reference IDs
			for existingRefs[refID] {
				nextCNumber++
				refID = fmt.Sprintf("C%d", nextCNumber)
			}

			control.ReferenceID = refID
			existingRefs[refID] = true
			nextCNumber++
		}
	}

	return processedControls
}

// findNextCNumber finds the next available C number by checking existing reference IDs
func (p *ControlReferenceProcessor) findNextCNumber(existingRefs map[string]bool) int {
	nextNumber := 1

	// Check for existing C-prefixed reference IDs to find the highest number
	for ref := range existingRefs {
		if strings.HasPrefix(ref, "C") && len(ref) > 1 {
			if num, err := strconv.Atoi(ref[1:]); err == nil && num >= nextNumber {
				nextNumber = num + 1
			}
		}
	}

	return nextNumber
}
