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

package config

import (
	"testing"
)

func TestInterpolationConfig_GetFlatVariables(t *testing.T) {
	tests := []struct {
		name      string
		variables map[string]interface{}
		expected  map[string]string
	}{
		{
			name: "Simple flat variables",
			variables: map[string]interface{}{
				"name":  "Test Organization",
				"email": "test@example.com",
			},
			expected: map[string]string{
				"name":  "Test Organization",
				"email": "test@example.com",
			},
		},
		{
			name: "Nested variables",
			variables: map[string]interface{}{
				"organization": map[string]interface{}{
					"name":    "Test Corp",
					"website": "https://test.com",
				},
				"contact": map[string]interface{}{
					"email": "support@test.com",
					"phone": "+1-555-1234",
				},
			},
			expected: map[string]string{
				"organization.name":    "Test Corp",
				"organization.website": "https://test.com",
				"contact.email":        "support@test.com",
				"contact.phone":        "+1-555-1234",
			},
		},
		{
			name: "Mixed flat and nested",
			variables: map[string]interface{}{
				"Company Name": "Test Company",
				"organization": map[string]interface{}{
					"name": "Test Org",
				},
				"simple": "value",
			},
			expected: map[string]string{
				"Company Name":      "Test Company",
				"organization.name": "Test Org",
				"simple":            "value",
			},
		},
		{
			name: "Deep nesting",
			variables: map[string]interface{}{
				"org": map[string]interface{}{
					"dept": map[string]interface{}{
						"team": map[string]interface{}{
							"lead": "John Doe",
						},
					},
				},
			},
			expected: map[string]string{
				"org.dept.team.lead": "John Doe",
			},
		},
		{
			name: "YAML-style map[interface{}]interface{}",
			variables: map[string]interface{}{
				"yaml_nested": map[interface{}]interface{}{
					"key1": "value1",
					"key2": map[interface{}]interface{}{
						"subkey": "subvalue",
					},
				},
			},
			expected: map[string]string{
				"yaml_nested.key1":        "value1",
				"yaml_nested.key2.subkey": "subvalue",
			},
		},
		{
			name: "Non-string values converted",
			variables: map[string]interface{}{
				"number":  42,
				"boolean": true,
				"float":   3.14,
			},
			expected: map[string]string{
				"number":  "42",
				"boolean": "true",
				"float":   "3.14",
			},
		},
		{
			name:      "Empty variables",
			variables: map[string]interface{}{},
			expected:  map[string]string{},
		},
		{
			name:      "Nil variables",
			variables: nil,
			expected:  map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ic := &InterpolationConfig{
				Enabled:   true,
				Variables: tt.variables,
			}

			result := ic.GetFlatVariables()

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d variables, got %d", len(tt.expected), len(result))
			}

			for key, expectedValue := range tt.expected {
				if actualValue, exists := result[key]; !exists {
					t.Errorf("Expected key %q not found in result", key)
				} else if actualValue != expectedValue {
					t.Errorf("For key %q: expected %q, got %q", key, expectedValue, actualValue)
				}
			}

			// Check for unexpected keys
			for key := range result {
				if _, expected := tt.expected[key]; !expected {
					t.Errorf("Unexpected key %q found in result with value %q", key, result[key])
				}
			}
		})
	}
}

func TestInterpolationConfig_flattenVariables(t *testing.T) {
	ic := &InterpolationConfig{}
	result := make(map[string]string)

	// Test with nil variables
	ic.flattenVariables("", nil, result)
	if len(result) != 0 {
		t.Error("Expected empty result for nil variables")
	}

	// Test prefix handling
	result = make(map[string]string)
	variables := map[string]interface{}{
		"key": "value",
	}
	ic.flattenVariables("prefix", variables, result)

	if result["prefix.key"] != "value" {
		t.Errorf("Expected 'prefix.key' = 'value', got %q", result["prefix.key"])
	}
}

func TestValidateInterpolationVariables(t *testing.T) {
	tests := []struct {
		name        string
		variables   map[string]interface{}
		expectError bool
	}{
		{
			name: "No circular references",
			variables: map[string]interface{}{
				"org": map[string]interface{}{
					"name": "Test Corp",
				},
				"welcome": "Welcome to {{org.name}}",
			},
			expectError: false,
		},
		{
			name: "Simple circular reference",
			variables: map[string]interface{}{
				"a": "{{b}}",
				"b": "{{a}}",
			},
			expectError: true,
		},
		{
			name: "Complex circular reference",
			variables: map[string]interface{}{
				"a": "{{b}} is great",
				"b": "{{c}} systems",
				"c": "{{a}} technology",
			},
			expectError: true,
		},
		{
			name: "Self reference not detected in simple case",
			variables: map[string]interface{}{
				"recursive": "This is {{recursive}}",
			},
			expectError: false, // Current implementation doesn't detect this case
		},
		{
			name: "No references",
			variables: map[string]interface{}{
				"simple": "Just text",
				"number": "42",
			},
			expectError: false,
		},
		{
			name: "Bracket format circular reference",
			variables: map[string]interface{}{
				"x": "[Y]",
				"Y": "[x]",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Interpolation: InterpolationConfig{
					Enabled:   true,
					Variables: tt.variables,
				},
			}

			err := config.validateInterpolationVariables()

			if tt.expectError && err == nil {
				t.Error("Expected error for circular reference, but got none")
			} else if !tt.expectError && err != nil {
				t.Errorf("Expected no error, but got: %v", err)
			}
		})
	}
}

func TestConfig_hasCircularReference(t *testing.T) {
	config := &Config{}
	flatVars := map[string]string{
		"a": "{{b}}",
		"b": "{{c}}",
		"c": "no reference",
	}
	visited := make(map[string]bool)
	recursionStack := make(map[string]bool)

	// Should not have circular reference
	if config.hasCircularReference("a", flatVars, visited, recursionStack) {
		t.Error("Expected no circular reference")
	}

	// Test actual circular reference
	flatVars["c"] = "{{a}}"
	visited = make(map[string]bool)
	recursionStack = make(map[string]bool)

	if !config.hasCircularReference("a", flatVars, visited, recursionStack) {
		t.Error("Expected circular reference to be detected")
	}
}

func TestConfig_Validate_InterpolationDefaults(t *testing.T) {
	config := &Config{
		Tugboat: TugboatConfig{
			BaseURL: "https://test.com",
			OrgID:   "123",
		},
		Interpolation: InterpolationConfig{
			Enabled: true,
			// Variables is nil, should get defaults
		},
	}

	err := config.Validate()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check that defaults were set
	flatVars := config.Interpolation.GetFlatVariables()
	if len(flatVars) == 0 {
		t.Error("Expected default variables to be set")
	}

	if flatVars["organization.name"] == "" {
		t.Error("Expected default organization.name to be set")
	}

	if flatVars["Organization Name"] == "" {
		t.Error("Expected default 'Organization Name' to be set")
	}
}

func TestConfig_Validate_InterpolationWithExistingVariables(t *testing.T) {
	config := &Config{
		Tugboat: TugboatConfig{
			BaseURL: "https://test.com",
			OrgID:   "123",
		},
		Interpolation: InterpolationConfig{
			Enabled: true,
			Variables: map[string]interface{}{
				"organization": map[string]interface{}{
					"name": "Custom Corp",
				},
			},
		},
	}

	err := config.Validate()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check that custom variables are preserved
	flatVars := config.Interpolation.GetFlatVariables()
	if flatVars["organization.name"] != "Custom Corp" {
		t.Errorf("Expected custom organization name, got: %s", flatVars["organization.name"])
	}
}
