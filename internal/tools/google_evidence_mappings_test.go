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

//go:build e2e

package tools

import (
	"testing"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
)

func TestGoogleEvidenceMappingsLoader(t *testing.T) {
	// Create a test logger
	log, _ := logger.NewZerologLogger(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "text",
		Output: "stdout",
	})

	// Create a test config
	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: "../../", // Point to grctool root where the YAML file is
		},
	}

	// Create the loader
	loader := NewGoogleEvidenceMappingsLoader(cfg, log)

	t.Run("LoadMappings", func(t *testing.T) {
		mappings, err := loader.LoadMappings()
		if err != nil {
			t.Fatalf("Failed to load mappings: %v", err)
		}

		if mappings == nil {
			t.Fatal("Mappings is nil")
		}

		if len(mappings.EvidenceMappings) == 0 {
			t.Fatal("No evidence mappings loaded")
		}

		// Check that we have the expected high-priority mappings
		expectedTasks := []string{"ET54", "ET64", "ET74", "ET29", "ET37", "ET50"}
		for _, taskRef := range expectedTasks {
			if _, exists := mappings.EvidenceMappings[taskRef]; !exists {
				t.Errorf("Expected mapping for task %s not found", taskRef)
			}
		}
	})

	t.Run("GetMappingForTask", func(t *testing.T) {
		// Test getting a specific mapping
		mapping, err := loader.GetMappingForTask("ET54")
		if err != nil {
			t.Fatalf("Failed to get mapping for ET54: %v", err)
		}

		if mapping.TaskRef != "ET54" {
			t.Errorf("Expected task ref ET54, got %s", mapping.TaskRef)
		}

		if mapping.SourceType != "google_docs" {
			t.Errorf("Expected source type google_docs, got %s", mapping.SourceType)
		}

		if len(mapping.Documents) == 0 {
			t.Error("No documents found in ET54 mapping")
		}
	})

	t.Run("GetMappingsByPriority", func(t *testing.T) {
		highPriorityMappings, err := loader.GetMappingsByPriority("high")
		if err != nil {
			t.Fatalf("Failed to get high priority mappings: %v", err)
		}

		if len(highPriorityMappings) == 0 {
			t.Error("No high priority mappings found")
		}

		// Check that all returned mappings have high priority
		for _, mapping := range highPriorityMappings {
			if mapping.Priority != "high" {
				t.Errorf("Expected high priority, got %s for task %s", mapping.Priority, mapping.TaskRef)
			}
		}
	})

	t.Run("GetMappingsBySourceType", func(t *testing.T) {
		sheetsMappings, err := loader.GetMappingsBySourceType("google_sheets")
		if err != nil {
			t.Fatalf("Failed to get sheets mappings: %v", err)
		}

		if len(sheetsMappings) == 0 {
			t.Error("No Google Sheets mappings found")
		}

		// Check that all returned mappings are for sheets
		for _, mapping := range sheetsMappings {
			if mapping.SourceType != "google_sheets" {
				t.Errorf("Expected google_sheets source type, got %s for task %s", mapping.SourceType, mapping.TaskRef)
			}
		}
	})

	t.Run("ValidateDocumentAccess", func(t *testing.T) {
		mapping, err := loader.GetMappingForTask("ET74")
		if err != nil {
			t.Fatalf("Failed to get mapping for ET74: %v", err)
		}

		err = loader.ValidateDocumentAccess(mapping)
		if err != nil {
			t.Errorf("Document access validation failed: %v", err)
		}
	})

	t.Run("TransformToGoogleAPIParams", func(t *testing.T) {
		mapping, err := loader.GetMappingForTask("ET74")
		if err != nil {
			t.Fatalf("Failed to get mapping for ET74: %v", err)
		}

		params, err := loader.TransformToGoogleAPIParams(mapping, 0)
		if err != nil {
			t.Fatalf("Failed to transform to API params: %v", err)
		}

		if params["document_id"] == nil {
			t.Error("document_id not found in API params")
		}

		if params["document_type"] == nil {
			t.Error("document_type not found in API params")
		}

		if params["extraction_rules"] == nil {
			t.Error("extraction_rules not found in API params")
		}
	})

	t.Run("GetRefreshSchedule", func(t *testing.T) {
		schedule, err := loader.GetRefreshSchedule("ET54")
		if err != nil {
			t.Fatalf("Failed to get refresh schedule for ET54: %v", err)
		}

		if schedule == "" {
			t.Error("Empty refresh schedule returned")
		}

		// Test default schedule for non-existent task
		defaultSchedule, err := loader.GetRefreshSchedule("NONEXISTENT")
		if err != nil {
			t.Fatalf("Failed to get default refresh schedule: %v", err)
		}

		if defaultSchedule != "monthly" {
			t.Errorf("Expected default schedule 'monthly', got %s", defaultSchedule)
		}
	})

	t.Run("GetSupportedTaskRefs", func(t *testing.T) {
		taskRefs, err := loader.GetSupportedTaskRefs()
		if err != nil {
			t.Fatalf("Failed to get supported task refs: %v", err)
		}

		if len(taskRefs) == 0 {
			t.Error("No supported task refs returned")
		}

		// Check that all expected tasks are included
		expectedTasks := map[string]bool{
			"ET54": false, "ET64": false, "ET74": false,
			"ET29": false, "ET37": false, "ET50": false,
		}

		for _, taskRef := range taskRefs {
			if _, exists := expectedTasks[taskRef]; exists {
				expectedTasks[taskRef] = true
			}
		}

		for taskRef, found := range expectedTasks {
			if !found {
				t.Errorf("Expected task ref %s not found in supported list", taskRef)
			}
		}
	})

	t.Run("GetComplianceFrameworkMappings", func(t *testing.T) {
		soc2Mappings, err := loader.GetComplianceFrameworkMappings("SOC2")
		if err != nil {
			t.Fatalf("Failed to get SOC2 mappings: %v", err)
		}

		if len(soc2Mappings) == 0 {
			t.Error("No SOC2 mappings found")
		}

		// Check that ET54 maps to control environment controls
		if controls, exists := soc2Mappings["ET54"]; exists {
			if len(controls) == 0 {
				t.Error("No controls found for ET54 in SOC2 mapping")
			}
		} else {
			t.Error("ET54 not found in SOC2 mappings")
		}
	})

	t.Run("ClearCache", func(t *testing.T) {
		// Load mappings to populate cache
		_, err := loader.LoadMappings()
		if err != nil {
			t.Fatalf("Failed to load mappings: %v", err)
		}

		// Clear cache
		loader.ClearCache()

		// Verify cache is cleared by checking internal state
		loader.cacheMutex.RLock()
		cacheEmpty := loader.cache == nil
		loader.cacheMutex.RUnlock()

		if !cacheEmpty {
			t.Error("Cache was not cleared properly")
		}
	})
}

func TestGoogleEvidenceMappingsValidation(t *testing.T) {
	log, _ := logger.NewZerologLogger(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "text",
		Output: "stdout",
	})

	cfg := &config.Config{}
	loader := NewGoogleEvidenceMappingsLoader(cfg, log)

	t.Run("ValidateEmptyMappings", func(t *testing.T) {
		mappings := &GoogleEvidenceMappings{
			EvidenceMappings: make(map[string]EvidenceMapping),
		}

		err := loader.validateMappings(mappings)
		if err == nil {
			t.Error("Expected validation error for empty mappings")
		}
	})

	t.Run("ValidateTaskRefMismatch", func(t *testing.T) {
		mappings := &GoogleEvidenceMappings{
			EvidenceMappings: map[string]EvidenceMapping{
				"ET54": {
					TaskRef:    "ET55", // Mismatch
					SourceType: "google_docs",
					Documents: []DocumentConfig{
						{
							DocumentID:   "test-doc",
							DocumentType: "docs",
						},
					},
				},
			},
		}

		err := loader.validateMappings(mappings)
		if err == nil {
			t.Error("Expected validation error for task ref mismatch")
		}
	})

	t.Run("ValidateMissingSourceType", func(t *testing.T) {
		mappings := &GoogleEvidenceMappings{
			EvidenceMappings: map[string]EvidenceMapping{
				"ET54": {
					TaskRef: "ET54",
					// SourceType missing
					Documents: []DocumentConfig{
						{
							DocumentID:   "test-doc",
							DocumentType: "docs",
						},
					},
				},
			},
		}

		err := loader.validateMappings(mappings)
		if err == nil {
			t.Error("Expected validation error for missing source type")
		}
	})

	t.Run("ValidateMissingDocuments", func(t *testing.T) {
		mappings := &GoogleEvidenceMappings{
			EvidenceMappings: map[string]EvidenceMapping{
				"ET54": {
					TaskRef:    "ET54",
					SourceType: "google_docs",
					Documents:  []DocumentConfig{}, // Empty
				},
			},
		}

		err := loader.validateMappings(mappings)
		if err == nil {
			t.Error("Expected validation error for missing documents")
		}
	})
}
