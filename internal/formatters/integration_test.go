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

package formatters

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/interpolation"
)

// TestInterpolationWithRealPolicyData tests the interpolation system with actual policy data
func TestInterpolationWithRealPolicyData(t *testing.T) {
	// Skip this test if we're not in the expected directory structure
	policyDir := "../../../docs/policies"
	if _, err := os.Stat(policyDir); os.IsNotExist(err) {
		t.Skip("Skipping integration test - policy directory not found")
		return
	}

	// Set up interpolation configuration
	config := interpolation.InterpolatorConfig{
		Variables: map[string]string{
			"organization.name": "Seventh Sense",
			"Organization Name": "Seventh Sense",
			"support.email":     "support@example.com",
			"security.email":    "security@example.com",
			"ceo.name":          "Test CEO",
			"cto.name":          "Test CTO",
		},
		Enabled:           true,
		OnMissingVariable: interpolation.MissingVariableIgnore,
	}

	interpolator := interpolation.NewStandardInterpolator(config)
	formatter := NewPolicyFormatterWithInterpolation(interpolator)

	// Find a policy file that contains template variables
	var testPolicyFile string
	err := filepath.Walk(policyDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".json") {
			// Read the file to check if it contains template variables
			data, err := os.ReadFile(path)
			if err != nil {
				return nil // Continue searching
			}

			// Check if the content contains template variables
			if strings.Contains(string(data), "{{organization.name}}") {
				testPolicyFile = path
				return filepath.SkipAll // Found one, stop searching
			}
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Error searching for policy files: %v", err)
	}

	if testPolicyFile == "" {
		t.Skip("No policy files with template variables found - skipping integration test")
		return
	}

	t.Logf("Testing with policy file: %s", testPolicyFile)

	// Load the policy data
	data, err := os.ReadFile(testPolicyFile)
	if err != nil {
		t.Fatalf("Failed to read policy file: %v", err)
	}

	var policy domain.Policy
	if err := json.Unmarshal(data, &policy); err != nil {
		t.Fatalf("Failed to parse policy JSON: %v", err)
	}

	// Generate markdown with interpolation
	markdown := formatter.ToMarkdown(&policy)

	// Verify that template variables were replaced
	if strings.Contains(markdown, "{{organization.name}}") {
		t.Error("Template variables should be replaced in the output")
	}

	// Verify that our organization name appears in the output
	if !strings.Contains(markdown, "Seventh Sense") {
		t.Error("Expected 'Seventh Sense' to appear in interpolated output")
	}

	t.Logf("✅ Successfully interpolated policy: %s", policy.Name)
	t.Logf("✅ Generated markdown length: %d characters", len(markdown))

	// Print a sample of the output for manual verification (first 500 chars)
	sample := markdown
	if len(sample) > 500 {
		sample = sample[:500] + "..."
	}
	t.Logf("Sample output:\n%s", sample)
}

// TestInterpolationSystemEndToEnd demonstrates the complete workflow
func TestInterpolationSystemEndToEnd(t *testing.T) {
	// Create a policy that mimics the structure of real Tugboat policies
	policy := &domain.Policy{
		ID:      "TEST-001",
		Name:    "{{organization.name}} Information Security Policy",
		Status:  "published",
		Summary: "<p>This policy establishes [Organization Name]'s approach to information security.</p>",
		Content: `<h1 style="text-align: center;"><strong>{{organization.name}} Information Security Policy</strong></h1>
<h2><strong>1.0 Purpose</strong></h2>
<p>The purpose of this policy is to ensure the security of information systems at {{organization.name}}.</p>
<p>This policy applies to all [Organization Name] employees, contractors, and vendors.</p>
<h2><strong>2.0 Scope</strong></h2>
<p>All {{organization.name}} employees must comply with this policy. Questions should be directed to {{support.email}}.</p>
<h2><strong>3.0 Responsibilities</strong></h2>
<p>The Chief Information Security Officer (CISO) at [Organization Name] is responsible for maintaining this policy.</p>
<ul>
<li>{{organization.name}} IT department shall implement technical controls</li>
<li>All employees shall report security incidents to {{security.email}}</li>
<li>Management shall review this policy annually</li>
</ul>`,
	}

	// Test different interpolation configurations
	testCases := []struct {
		name      string
		config    interpolation.InterpolatorConfig
		checks    []string // Strings that should appear in output
		notChecks []string // Strings that should NOT appear in output
	}{
		{
			name: "Standard interpolation enabled",
			config: interpolation.InterpolatorConfig{
				Variables: map[string]string{
					"organization.name": "Acme Corporation",
					"Organization Name": "Acme Corporation",
					"support.email":     "help@acme.com",
					"security.email":    "security@acme.com",
				},
				Enabled:           true,
				OnMissingVariable: interpolation.MissingVariableIgnore,
			},
			checks: []string{
				"Acme Corporation Information Security Policy",
				"security of information systems at Acme Corporation",
				"All Acme Corporation employees must comply",
				"directed to help@acme.com",
				"incidents to security@acme.com",
			},
			notChecks: []string{
				"{{organization.name}}",
				"[Organization Name]",
			},
		},
		{
			name: "Interpolation disabled",
			config: interpolation.InterpolatorConfig{
				Variables: map[string]string{
					"organization.name": "Should Not Appear",
				},
				Enabled:           false,
				OnMissingVariable: interpolation.MissingVariableIgnore,
			},
			checks: []string{
				"{{organization.name}} Information Security Policy",
				"[Organization Name]'s approach",
			},
			notChecks: []string{
				"Should Not Appear",
				"Acme Corporation",
			},
		},
		{
			name: "Partial variable configuration",
			config: interpolation.InterpolatorConfig{
				Variables: map[string]string{
					"organization.name": "Partial Corp",
					// Missing Organization Name, support.email, security.email
				},
				Enabled:           true,
				OnMissingVariable: interpolation.MissingVariableIgnore,
			},
			checks: []string{
				"Partial Corp Information Security Policy",
				"information systems at Partial Corp",
			},
			notChecks: []string{
				"{{organization.name}}",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			interpolator := interpolation.NewStandardInterpolator(tc.config)
			formatter := NewPolicyFormatterWithInterpolation(interpolator)

			markdown := formatter.ToMarkdown(policy)

			// Check for expected strings, normalizing whitespace for wrapped text
			normalizedOutput := strings.ReplaceAll(markdown, "\n", " ")
			normalizedOutput = strings.Join(strings.Fields(normalizedOutput), " ")

			for _, check := range tc.checks {
				normalizedCheck := strings.Join(strings.Fields(check), " ")
				if !strings.Contains(normalizedOutput, normalizedCheck) {
					t.Errorf("Expected to find %q in output", check)
				}
			}

			// Check for strings that should NOT be present
			for _, notCheck := range tc.notChecks {
				if strings.Contains(markdown, notCheck) {
					t.Errorf("Expected NOT to find %q in output", notCheck)
				}
			}
		})
	}
}

// TestInterpolationPerformance tests the performance impact of interpolation
func TestInterpolationPerformance(t *testing.T) {
	// Create a large policy content to test performance
	largeContent := strings.Repeat(`<p>{{organization.name}} policy section with [Organization Name] references and {{support.email}} contact info. `, 1000)

	policy := &domain.Policy{
		ID:      "PERF-001",
		Name:    "{{organization.name}} Large Policy Document",
		Content: largeContent,
		Status:  "published",
	}

	config := interpolation.InterpolatorConfig{
		Variables: map[string]string{
			"organization.name": "Performance Test Corp",
			"Organization Name": "Performance Test Corp",
			"support.email":     "support@perftest.com",
		},
		Enabled:           true,
		OnMissingVariable: interpolation.MissingVariableIgnore,
	}

	interpolator := interpolation.NewStandardInterpolator(config)
	formatter := NewPolicyFormatterWithInterpolation(interpolator)

	// Test performance (should complete reasonably quickly)
	markdown := formatter.ToMarkdown(policy)

	// Verify interpolation worked on large content
	expectedReplacements := 1000 * 3 // 3 variables per repeated section
	actualReplacements := strings.Count(markdown, "Performance Test Corp") + strings.Count(markdown, "support@perftest.com")

	if actualReplacements < expectedReplacements {
		t.Errorf("Expected at least %d variable replacements, got %d", expectedReplacements, actualReplacements)
	}

	t.Logf("✅ Performance test completed with %d variable replacements", actualReplacements)
	t.Logf("✅ Output size: %d characters", len(markdown))
}
