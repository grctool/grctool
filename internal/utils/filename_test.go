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

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilenameGenerator_GenerateFilename(t *testing.T) {
	fg := NewFilenameGenerator()

	tests := []struct {
		name        string
		referenceID string
		numericID   string
		itemName    string
		extension   string
		expected    string
	}{
		{
			name:        "Policy with simple name",
			referenceID: "P136",
			numericID:   "94623",
			itemName:    "Data Retention and Disposal",
			extension:   "json",
			expected:    "P136_94623_Data_Retention_and_Disposal.json",
		},
		{
			name:        "Control with complex name",
			referenceID: "AC1",
			numericID:   "778771",
			itemName:    "Access Provisioning and Approval - Infrastructure",
			extension:   ".json",
			expected:    "AC1_778771_Access_Provisioning_and_Approval_Infrastructure.json",
		},
		{
			name:        "Evidence task with special characters",
			referenceID: "ET1",
			numericID:   "327992",
			itemName:    "Access Control / Registration & De-registration Process Document",
			extension:   "md",
			expected:    "ET1_327992_Access_Control_Registration_and_De-registration_Process_Document.md",
		},
		{
			name:        "Name with parentheses",
			referenceID: "P200",
			numericID:   "123456",
			itemName:    "Security Policy (Draft)",
			extension:   "json",
			expected:    "P200_123456_Security_Policy_Draft.json",
		},
		{
			name:        "Name with percent and at symbols",
			referenceID: "ET50",
			numericID:   "789012",
			itemName:    "Performance @ 100% Load",
			extension:   "md",
			expected:    "ET50_789012_Performance_at_100percent_Load.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fg.GenerateFilename(tt.referenceID, tt.numericID, tt.itemName, tt.extension)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilenameGenerator_SanitizeName(t *testing.T) {
	fg := NewFilenameGenerator()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple name with spaces",
			input:    "Data Retention and Disposal",
			expected: "Data_Retention_and_Disposal",
		},
		{
			name:     "Name with slashes",
			input:    "Access Control / Registration",
			expected: "Access_Control_Registration",
		},
		{
			name:     "Name with hyphens and spaces",
			input:    "Access Control - Registration and De-registration",
			expected: "Access_Control_Registration_and_De-registration",
		},
		{
			name:     "Name with special characters",
			input:    "Security & Compliance (2023) #1",
			expected: "Security_and_Compliance_2023_1",
		},
		{
			name:     "Name with quotes",
			input:    `Policy "Draft" Version`,
			expected: "Policy_Draft_Version",
		},
		{
			name:     "Name with multiple spaces",
			input:    "Policy    with     spaces",
			expected: "Policy_with_spaces",
		},
		{
			name:     "Very long name gets truncated",
			input:    "This is an extremely long policy name that exceeds the maximum allowed length and should be truncated to fit within the limit set by the system",
			expected: "This_is_an_extremely_long_policy_name_that_exceeds_the_maximum_allowed_length_and_should_be_truncate",
		},
		{
			name:     "Name with percent and at",
			input:    "Performance @ 100%",
			expected: "Performance_at_100percent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fg.SanitizeName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilenameGenerator_ParseFilename(t *testing.T) {
	fg := NewFilenameGenerator()

	tests := []struct {
		name          string
		filename      string
		expectedRefID string
		expectedNumID string
		expectedName  string
		expectedExt   string
		expectError   bool
	}{
		{
			name:          "Valid policy filename",
			filename:      "P136_94623_Data_Retention_and_Disposal.json",
			expectedRefID: "P136",
			expectedNumID: "94623",
			expectedName:  "Data_Retention_and_Disposal",
			expectedExt:   ".json",
			expectError:   false,
		},
		{
			name:          "Valid control filename",
			filename:      "AC1_778771_Access_Provisioning_and_Approval.md",
			expectedRefID: "AC1",
			expectedNumID: "778771",
			expectedName:  "Access_Provisioning_and_Approval",
			expectedExt:   ".md",
			expectError:   false,
		},
		{
			name:          "Name with multiple underscores",
			filename:      "ET1_327992_Access_Control_Registration_and_De_registration.json",
			expectedRefID: "ET1",
			expectedNumID: "327992",
			expectedName:  "Access_Control_Registration_and_De_registration",
			expectedExt:   ".json",
			expectError:   false,
		},
		{
			name:        "Invalid format - missing parts",
			filename:    "P136_Data.json",
			expectError: true,
		},
		{
			name:        "Invalid format - no extension",
			filename:    "P136_94623_Data_Retention",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refID, numID, name, ext, err := fg.ParseFilename(tt.filename)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedRefID, refID)
				assert.Equal(t, tt.expectedNumID, numID)
				assert.Equal(t, tt.expectedName, name)
				assert.Equal(t, tt.expectedExt, ext)
			}
		})
	}
}

func TestFilenameGenerator_IsValidFilename(t *testing.T) {
	fg := NewFilenameGenerator()

	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{
			name:     "Valid policy JSON",
			filename: "P136_94623_Data_Retention_and_Disposal.json",
			expected: true,
		},
		{
			name:     "Valid control MD",
			filename: "AC1_778771_Access_Provisioning.md",
			expected: true,
		},
		{
			name:     "Valid evidence task",
			filename: "ET123_456789_Some_Task_Name.json",
			expected: true,
		},
		{
			name:     "Invalid - wrong extension",
			filename: "P136_94623_Data_Retention.txt",
			expected: false,
		},
		{
			name:     "Invalid - missing numeric ID",
			filename: "P136_Data_Retention.json",
			expected: false,
		},
		{
			name:     "Invalid - lowercase reference",
			filename: "p136_94623_Data_Retention.json",
			expected: false,
		},
		{
			name:     "Invalid - special chars in name",
			filename: "P136_94623_Data-Retention.json",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fg.IsValidFilename(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}
