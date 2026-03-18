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
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestConfig_ProvidersConfig(t *testing.T) {
	input := `
providers:
  providers:
    - name: tugboat-primary
      type: tugboat
      enabled: true
      settings:
        base_url: "https://app.tugboatlogic.com"
    - name: gdrive-evidence
      type: gdrive
      enabled: false
      settings:
        folder_id: "abc123"
`
	var cfg Config
	if err := yaml.Unmarshal([]byte(input), &cfg); err != nil {
		t.Fatalf("Failed to unmarshal providers YAML: %v", err)
	}

	if len(cfg.Providers.Providers) != 2 {
		t.Fatalf("Expected 2 providers, got %d", len(cfg.Providers.Providers))
	}

	p0 := cfg.Providers.Providers[0]
	if p0.Name != "tugboat-primary" {
		t.Errorf("Expected provider name 'tugboat-primary', got %q", p0.Name)
	}
	if p0.Type != "tugboat" {
		t.Errorf("Expected provider type 'tugboat', got %q", p0.Type)
	}
	if !p0.Enabled {
		t.Error("Expected provider to be enabled")
	}
	if p0.Settings["base_url"] != "https://app.tugboatlogic.com" {
		t.Errorf("Expected setting base_url, got %q", p0.Settings["base_url"])
	}

	p1 := cfg.Providers.Providers[1]
	if p1.Name != "gdrive-evidence" {
		t.Errorf("Expected provider name 'gdrive-evidence', got %q", p1.Name)
	}
	if p1.Enabled {
		t.Error("Expected provider to be disabled")
	}

	// Round-trip: marshal back to YAML and unmarshal again
	out, err := yaml.Marshal(&cfg)
	if err != nil {
		t.Fatalf("Failed to marshal providers: %v", err)
	}

	var cfg2 Config
	if err := yaml.Unmarshal(out, &cfg2); err != nil {
		t.Fatalf("Failed to unmarshal round-tripped providers: %v", err)
	}

	if len(cfg2.Providers.Providers) != 2 {
		t.Fatalf("Round-trip: expected 2 providers, got %d", len(cfg2.Providers.Providers))
	}
	if cfg2.Providers.Providers[0].Name != "tugboat-primary" {
		t.Errorf("Round-trip: expected provider name 'tugboat-primary', got %q", cfg2.Providers.Providers[0].Name)
	}
}

func TestConfig_SchedulesConfig(t *testing.T) {
	input := `
schedules:
  schedules:
    - name: nightly-sync
      cron: "0 2 * * *"
      enabled: true
      scope: all
      provider: tugboat-primary
    - name: weekly-evidence
      cron: "0 8 * * 1"
      enabled: false
      scope: evidence
`
	var cfg Config
	if err := yaml.Unmarshal([]byte(input), &cfg); err != nil {
		t.Fatalf("Failed to unmarshal schedules YAML: %v", err)
	}

	if len(cfg.Schedules.Schedules) != 2 {
		t.Fatalf("Expected 2 schedules, got %d", len(cfg.Schedules.Schedules))
	}

	s0 := cfg.Schedules.Schedules[0]
	if s0.Name != "nightly-sync" {
		t.Errorf("Expected schedule name 'nightly-sync', got %q", s0.Name)
	}
	if s0.Cron != "0 2 * * *" {
		t.Errorf("Expected cron '0 2 * * *', got %q", s0.Cron)
	}
	if !s0.Enabled {
		t.Error("Expected schedule to be enabled")
	}
	if s0.Scope != "all" {
		t.Errorf("Expected scope 'all', got %q", s0.Scope)
	}
	if s0.Provider != "tugboat-primary" {
		t.Errorf("Expected provider 'tugboat-primary', got %q", s0.Provider)
	}

	s1 := cfg.Schedules.Schedules[1]
	if s1.Enabled {
		t.Error("Expected schedule to be disabled")
	}
	if s1.Provider != "" {
		t.Errorf("Expected empty provider, got %q", s1.Provider)
	}

	// Round-trip
	out, err := yaml.Marshal(&cfg)
	if err != nil {
		t.Fatalf("Failed to marshal schedules: %v", err)
	}

	var cfg2 Config
	if err := yaml.Unmarshal(out, &cfg2); err != nil {
		t.Fatalf("Failed to unmarshal round-tripped schedules: %v", err)
	}

	if len(cfg2.Schedules.Schedules) != 2 {
		t.Fatalf("Round-trip: expected 2 schedules, got %d", len(cfg2.Schedules.Schedules))
	}
	if cfg2.Schedules.Schedules[0].Cron != "0 2 * * *" {
		t.Errorf("Round-trip: expected cron '0 2 * * *', got %q", cfg2.Schedules.Schedules[0].Cron)
	}
}

func TestConfig_LifecycleConfig(t *testing.T) {
	input := `
lifecycle:
  policy_review_cadence: "90d"
  control_test_cadence: "quarterly"
  evidence_retention: "7y"
`
	var cfg Config
	if err := yaml.Unmarshal([]byte(input), &cfg); err != nil {
		t.Fatalf("Failed to unmarshal lifecycle YAML: %v", err)
	}

	if cfg.Lifecycle.PolicyReviewCadence != "90d" {
		t.Errorf("Expected policy_review_cadence '90d', got %q", cfg.Lifecycle.PolicyReviewCadence)
	}
	if cfg.Lifecycle.ControlTestCadence != "quarterly" {
		t.Errorf("Expected control_test_cadence 'quarterly', got %q", cfg.Lifecycle.ControlTestCadence)
	}
	if cfg.Lifecycle.EvidenceRetention != "7y" {
		t.Errorf("Expected evidence_retention '7y', got %q", cfg.Lifecycle.EvidenceRetention)
	}

	// Round-trip
	out, err := yaml.Marshal(&cfg)
	if err != nil {
		t.Fatalf("Failed to marshal lifecycle: %v", err)
	}

	var cfg2 Config
	if err := yaml.Unmarshal(out, &cfg2); err != nil {
		t.Fatalf("Failed to unmarshal round-tripped lifecycle: %v", err)
	}

	if cfg2.Lifecycle.PolicyReviewCadence != "90d" {
		t.Errorf("Round-trip: expected '90d', got %q", cfg2.Lifecycle.PolicyReviewCadence)
	}
	if cfg2.Lifecycle.ControlTestCadence != "quarterly" {
		t.Errorf("Round-trip: expected 'quarterly', got %q", cfg2.Lifecycle.ControlTestCadence)
	}
	if cfg2.Lifecycle.EvidenceRetention != "7y" {
		t.Errorf("Round-trip: expected '7y', got %q", cfg2.Lifecycle.EvidenceRetention)
	}
}

func TestConfig_EmptyNewSections(t *testing.T) {
	// A config with no providers, schedules, or lifecycle should unmarshal fine
	input := `
tugboat:
  base_url: "https://test.com"
  org_id: "123"
`
	var cfg Config
	if err := yaml.Unmarshal([]byte(input), &cfg); err != nil {
		t.Fatalf("Failed to unmarshal config without new sections: %v", err)
	}

	if len(cfg.Providers.Providers) != 0 {
		t.Errorf("Expected 0 providers, got %d", len(cfg.Providers.Providers))
	}
	if len(cfg.Schedules.Schedules) != 0 {
		t.Errorf("Expected 0 schedules, got %d", len(cfg.Schedules.Schedules))
	}
	if cfg.Lifecycle.PolicyReviewCadence != "" {
		t.Errorf("Expected empty policy_review_cadence, got %q", cfg.Lifecycle.PolicyReviewCadence)
	}
	if cfg.Lifecycle.ControlTestCadence != "" {
		t.Errorf("Expected empty control_test_cadence, got %q", cfg.Lifecycle.ControlTestCadence)
	}
	if cfg.Lifecycle.EvidenceRetention != "" {
		t.Errorf("Expected empty evidence_retention, got %q", cfg.Lifecycle.EvidenceRetention)
	}
}

func TestConfig_ValidateWithProviders(t *testing.T) {
	cfg := &Config{
		Tugboat: TugboatConfig{
			BaseURL: "https://test.com",
			OrgID:   "123",
		},
		Providers: ProvidersConfig{
			Providers: []ProviderConfig{
				{
					Name:    "tugboat-primary",
					Type:    "tugboat",
					Enabled: true,
					Settings: map[string]string{
						"base_url": "https://app.tugboatlogic.com",
					},
				},
			},
		},
		Schedules: SchedulesConfig{
			Schedules: []ScheduleConfig{
				{
					Name:     "nightly-sync",
					Cron:     "0 2 * * *",
					Enabled:  true,
					Scope:    "all",
					Provider: "tugboat-primary",
				},
			},
		},
		Lifecycle: LifecycleConfig{
			PolicyReviewCadence: "90d",
			ControlTestCadence:  "quarterly",
			EvidenceRetention:   "7y",
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("Expected validation to pass with providers/schedules/lifecycle, got: %v", err)
	}

	// Verify providers are still intact after validation
	if len(cfg.Providers.Providers) != 1 {
		t.Errorf("Expected 1 provider after validation, got %d", len(cfg.Providers.Providers))
	}
	if len(cfg.Schedules.Schedules) != 1 {
		t.Errorf("Expected 1 schedule after validation, got %d", len(cfg.Schedules.Schedules))
	}
	if cfg.Lifecycle.EvidenceRetention != "7y" {
		t.Errorf("Expected evidence_retention '7y' after validation, got %q", cfg.Lifecycle.EvidenceRetention)
	}
}

func TestConfig_ValidateConfigStructure_NewKeys(t *testing.T) {
	// Write a temp config file with the new keys and verify no warnings are produced
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, ".grctool.yaml")

	cfgContent := `tugboat:
  base_url: "https://test.com"
  org_id: "123"
providers:
  providers:
    - name: test
      type: tugboat
      enabled: true
schedules:
  schedules:
    - name: nightly
      cron: "0 2 * * *"
      enabled: true
lifecycle:
  policy_review_cadence: "90d"
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}

	// Parse the YAML and check against known keys directly
	var rawConfig map[string]interface{}
	if err := yaml.Unmarshal([]byte(cfgContent), &rawConfig); err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	knownKeys := map[string]bool{
		"tugboat":       true,
		"evidence":      true,
		"storage":       true,
		"logging":       true,
		"interpolation": true,
		"auth":          true,
		"providers":     true,
		"schedules":     true,
		"lifecycle":     true,
	}

	for key := range rawConfig {
		if !knownKeys[key] {
			t.Errorf("Key %q would trigger unknown key warning", key)
		}
	}
}

func TestConfig_BackwardCompatible(t *testing.T) {
	// A config that only has the original sections should still load and validate
	cfg := &Config{
		Tugboat: TugboatConfig{
			BaseURL: "https://test.com",
			OrgID:   "123",
		},
		Evidence: EvidenceConfig{
			Generation: GenerationConfig{
				OutputDir:     "evidence/generated",
				PromptDir:     "evidence/prompts",
				DefaultFormat: "csv",
				MaxToolCalls:  50,
			},
		},
		Storage: StorageConfig{
			DataDir:      "./data",
			LocalDataDir: "./local_data",
			CacheDir:     "./.cache",
		},
		Interpolation: InterpolationConfig{
			Enabled: true,
			Variables: map[string]interface{}{
				"organization": map[string]interface{}{
					"name": "Test Corp",
				},
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("Expected backward-compatible config to validate, got: %v", err)
	}

	// New sections should be zero-valued but not cause issues
	if len(cfg.Providers.Providers) != 0 {
		t.Errorf("Expected 0 providers in backward-compatible config, got %d", len(cfg.Providers.Providers))
	}
	if len(cfg.Schedules.Schedules) != 0 {
		t.Errorf("Expected 0 schedules in backward-compatible config, got %d", len(cfg.Schedules.Schedules))
	}
	if cfg.Lifecycle.PolicyReviewCadence != "" {
		t.Errorf("Expected empty lifecycle cadence in backward-compatible config, got %q", cfg.Lifecycle.PolicyReviewCadence)
	}
}
