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

package interpolation

import (
	"strings"
	"testing"
)

func TestNewStandardInterpolator(t *testing.T) {
	config := InterpolatorConfig{
		Variables: map[string]string{
			"test.var": "test value",
		},
		Enabled:           true,
		OnMissingVariable: MissingVariableIgnore,
	}

	interpolator := NewStandardInterpolator(config)

	if interpolator == nil {
		t.Fatal("Expected interpolator to be created")
	}

	if !interpolator.enabled {
		t.Error("Expected interpolator to be enabled")
	}

	if len(interpolator.variables) != 1 {
		t.Errorf("Expected 1 variable, got %d", len(interpolator.variables))
	}
}

func TestInterpolateTemplateVariables(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		vars     map[string]string
		expected string
	}{
		{
			name:     "Single template variable",
			input:    "Welcome to {{organization.name}}!",
			vars:     map[string]string{"organization.name": "ACME Corp"},
			expected: "Welcome to ACME Corp!",
		},
		{
			name:     "Multiple template variables",
			input:    "{{organization.name}} contact: {{support.email}}",
			vars:     map[string]string{"organization.name": "ACME Corp", "support.email": "support@acme.com"},
			expected: "ACME Corp contact: support@acme.com",
		},
		{
			name:     "Template variable with extra spaces",
			input:    "{{ organization.name }} is great!",
			vars:     map[string]string{"organization.name": "ACME Corp"},
			expected: "ACME Corp is great!",
		},
		{
			name:     "No variables",
			input:    "This is just plain text",
			vars:     map[string]string{},
			expected: "This is just plain text",
		},
		{
			name:     "Missing variable (ignore mode)",
			input:    "Welcome to {{unknown.variable}}!",
			vars:     map[string]string{"organization.name": "ACME Corp"},
			expected: "Welcome to {{unknown.variable}}!",
		},
		{
			name:     "Empty input",
			input:    "",
			vars:     map[string]string{"organization.name": "ACME Corp"},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := InterpolatorConfig{
				Variables:         tt.vars,
				Enabled:           true,
				OnMissingVariable: MissingVariableIgnore,
			}
			interpolator := NewStandardInterpolator(config)

			result, err := interpolator.Interpolate(tt.input)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestInterpolateWithContext(t *testing.T) {
	config := InterpolatorConfig{
		Variables: map[string]string{
			"organization.name": "Acme Corp",
			"support.email":     "support@acme.com",
		},
		Enabled:           true,
		OnMissingVariable: MissingVariableIgnore,
	}

	interpolator := NewStandardInterpolator(config)

	input := "Welcome to {{organization.name}}! Contact us at {{support.email}}"
	result, substitutions, err := interpolator.InterpolateWithContext(input)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedResult := "Welcome to Acme Corp! Contact us at support@acme.com"
	if result != expectedResult {
		t.Errorf("Expected %q, got %q", expectedResult, result)
	}

	if len(substitutions) == 0 {
		t.Error("Expected substitutions to be tracked")
	}
}

func TestMissingVariableHandling(t *testing.T) {
	input := "Welcome to {{organization.name}} and {{unknown.variable}}"
	vars := map[string]string{
		"organization.name": "ACME Corp",
	}

	tests := []struct {
		name           string
		action         MissingVariableAction
		expectedResult string
		expectError    bool
	}{
		{
			name:           "Ignore missing variable",
			action:         MissingVariableIgnore,
			expectedResult: "Welcome to ACME Corp and {{unknown.variable}}",
			expectError:    false,
		},
		{
			name:           "Warn on missing variable",
			action:         MissingVariableWarn,
			expectedResult: "Welcome to ACME Corp and {{unknown.variable}}",
			expectError:    false,
		},
		{
			name:        "Error on missing variable",
			action:      MissingVariableError,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := InterpolatorConfig{
				Variables:         vars,
				Enabled:           true,
				OnMissingVariable: tt.action,
			}
			interpolator := NewStandardInterpolator(config)

			result, err := interpolator.Interpolate(input)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error for missing variable")
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if result != tt.expectedResult {
					t.Errorf("Expected %q, got %q", tt.expectedResult, result)
				}
			}
		})
	}
}

func TestInterpolatorDisabled(t *testing.T) {
	config := InterpolatorConfig{
		Variables: map[string]string{
			"organization.name": "ACME Corp",
		},
		Enabled:           false, // Disabled
		OnMissingVariable: MissingVariableIgnore,
	}

	interpolator := NewStandardInterpolator(config)
	input := "Welcome to {{organization.name}}!"

	result, err := interpolator.Interpolate(input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should return unchanged when disabled
	if result != input {
		t.Errorf("Expected unchanged input when disabled, got %q", result)
	}
}

func TestVariableManagement(t *testing.T) {
	config := InterpolatorConfig{
		Variables: map[string]string{
			"var1": "value1",
			"var2": "value2",
		},
		Enabled:           true,
		OnMissingVariable: MissingVariableIgnore,
	}

	interpolator := NewStandardInterpolator(config)

	// Test GetVariables returns a copy
	variables := interpolator.GetVariables()
	if len(variables) != 2 {
		t.Errorf("Expected 2 variables, got %d", len(variables))
	}

	// Modify the returned map - should not affect interpolator
	variables["var3"] = "value3"

	// Original should be unchanged
	originalVars := interpolator.GetVariables()
	if len(originalVars) != 2 {
		t.Error("Original variables should be unchanged")
	}
}

func TestFindVariableValue(t *testing.T) {
	config := InterpolatorConfig{
		Variables: map[string]string{
			"organization.name": "ACME Corp",
			"support.email":     "support@acme.com",
		},
		Enabled:           true,
		OnMissingVariable: MissingVariableIgnore,
	}

	interpolator := NewStandardInterpolator(config)

	tests := []struct {
		name     string
		varName  string
		expected string
		found    bool
	}{
		{
			name:     "Exact match",
			varName:  "organization.name",
			expected: "ACME Corp",
			found:    true,
		},
		{
			name:     "No match",
			varName:  "unknown.variable",
			expected: "",
			found:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, found := interpolator.findVariableValue(tt.varName)

			if found != tt.found {
				t.Errorf("Expected found=%v, got found=%v", tt.found, found)
			}

			if value != tt.expected {
				t.Errorf("Expected value=%q, got value=%q", tt.expected, value)
			}
		})
	}
}

func TestInterpolateTemplateVariables_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		vars     map[string]string
		expected string
	}{
		{
			name:     "Template with spaces",
			input:    "{{ organization.name }}",
			vars:     map[string]string{"organization.name": "ACME Corp"},
			expected: "ACME Corp",
		},
		{
			name:     "Multiple templates same variable",
			input:    "{{name}} and {{name}} are the same",
			vars:     map[string]string{"name": "John"},
			expected: "John and John are the same",
		},
		{
			name:     "Template with dots and underscores",
			input:    "{{org.sub_dept.team_lead}}",
			vars:     map[string]string{"org.sub_dept.team_lead": "Jane Doe"},
			expected: "Jane Doe",
		},
		{
			name:     "Malformed templates",
			input:    "{{unclosed or }incomplete}",
			vars:     map[string]string{},
			expected: "{{unclosed or }incomplete}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := InterpolatorConfig{
				Variables:         tt.vars,
				Enabled:           true,
				OnMissingVariable: MissingVariableIgnore,
			}
			interpolator := NewStandardInterpolator(config)

			result, err := interpolator.Interpolate(tt.input)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestMissingVariableActions_Comprehensive(t *testing.T) {
	input := "Welcome to {{company.name}} for all your needs!"
	vars := map[string]string{
		// "company.name" is intentionally missing
	}

	tests := []struct {
		name           string
		action         MissingVariableAction
		expectedResult string
		expectError    bool
	}{
		{
			name:           "Ignore missing variables",
			action:         MissingVariableIgnore,
			expectedResult: "Welcome to {{company.name}} for all your needs!",
			expectError:    false,
		},
		{
			name:           "Warn on missing variables",
			action:         MissingVariableWarn,
			expectedResult: "Welcome to {{company.name}} for all your needs!",
			expectError:    false, // Warnings don't cause errors
		},
		{
			name:        "Error on missing variables",
			action:      MissingVariableError,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := InterpolatorConfig{
				Variables:         vars,
				Enabled:           true,
				OnMissingVariable: tt.action,
			}
			interpolator := NewStandardInterpolator(config)

			result, err := interpolator.Interpolate(input)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error for missing variable, but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if result != tt.expectedResult {
					t.Errorf("Expected %q, got %q", tt.expectedResult, result)
				}
			}
		})
	}
}

func TestInterpolationWithContext_DetailedSubstitutions(t *testing.T) {
	input := "{{org.name}} policy with {{contact.email}}"
	vars := map[string]string{
		"org.name":      "ACME Corp",
		"contact.email": "info@acme.com",
	}

	config := InterpolatorConfig{
		Variables:         vars,
		Enabled:           true,
		OnMissingVariable: MissingVariableIgnore,
	}
	interpolator := NewStandardInterpolator(config)

	result, substitutions, err := interpolator.InterpolateWithContext(input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedResult := "ACME Corp policy with info@acme.com"
	if result != expectedResult {
		t.Errorf("Expected %q, got %q", expectedResult, result)
	}

	// Just verify that substitutions were made
	if len(substitutions) == 0 {
		t.Error("Expected some substitutions to be tracked")
	}

	// Verify the variables that should have been substituted exist in the result
	if !strings.Contains(result, "ACME Corp") {
		t.Error("Expected 'ACME Corp' in result")
	}
	if !strings.Contains(result, "info@acme.com") {
		t.Error("Expected 'info@acme.com' in result")
	}
}

func TestComplexPolicyContent(t *testing.T) {
	config := InterpolatorConfig{
		Variables: map[string]string{
			"organization.name": "Seventh Sense",
			"support.email":     "support@example.com",
			"security.email":    "security@example.com",
			"company.phone":     "+1 (555) 123-4567",
		},
		Enabled:           true,
		OnMissingVariable: MissingVariableIgnore,
	}

	interpolator := NewStandardInterpolator(config)

	// Simulate a complex policy document
	input := `# {{organization.name}} Security Policy

## 1.0 Purpose
This policy establishes security guidelines for {{organization.name}}.

## 2.0 Contact Information
- Support: {{support.email}}
- Security Team: {{security.email}}
- Phone: {{company.phone}}

## 3.0 Scope
This policy applies to all {{organization.name}} systems and personnel.`

	result, err := interpolator.Interpolate(input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify key substitutions were made
	if !strings.Contains(result, "Seventh Sense Security Policy") {
		t.Error("Expected organization name in title")
	}
	if !strings.Contains(result, "support@example.com") {
		t.Error("Expected support email substitution")
	}
	if !strings.Contains(result, "security@example.com") {
		t.Error("Expected security email substitution")
	}
	if !strings.Contains(result, "+1 (555) 123-4567") {
		t.Error("Expected phone number substitution")
	}

	// Verify the original template variables are gone
	if strings.Contains(result, "{{organization.name}}") {
		t.Error("Template variables should be replaced")
	}
}
