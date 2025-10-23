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

package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/storage"
)

// BenchmarkDataProcessing benchmarks data processing operations
func BenchmarkDataProcessing(b *testing.B) {
	ctx := context.Background()
	dataService := setupBenchmarkDataService(b)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Benchmark evidence task retrieval
		tasks, err := dataService.GetAllEvidenceTasks(ctx)
		if err != nil {
			b.Fatalf("Failed to get evidence tasks: %v", err)
		}

		// Benchmark filtering operations
		filter := domain.EvidenceFilter{
			Status:   []string{"open", "in_progress"},
			Priority: []string{"high", "medium"},
		}
		_, err = dataService.FilterEvidenceTasks(ctx, filter)
		if err != nil {
			b.Fatalf("Failed to filter evidence tasks: %v", err)
		}

		// Use the tasks to prevent optimization
		_ = len(tasks)
	}
}

// BenchmarkJSONSerialization benchmarks JSON serialization operations
func BenchmarkJSONSerialization(b *testing.B) {
	// Create large domain objects for serialization benchmarking
	evidenceTask := createLargeEvidenceTask()
	policy := createLargePolicy()
	control := createLargeControl()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Benchmark evidence task serialization
		taskData, err := json.Marshal(evidenceTask)
		if err != nil {
			b.Fatalf("Failed to marshal evidence task: %v", err)
		}

		var unmarshaled domain.EvidenceTask
		err = json.Unmarshal(taskData, &unmarshaled)
		if err != nil {
			b.Fatalf("Failed to unmarshal evidence task: %v", err)
		}

		// Benchmark policy serialization
		policyData, err := json.Marshal(policy)
		if err != nil {
			b.Fatalf("Failed to marshal policy: %v", err)
		}

		var unmarshaledPolicy domain.Policy
		err = json.Unmarshal(policyData, &unmarshaledPolicy)
		if err != nil {
			b.Fatalf("Failed to unmarshal policy: %v", err)
		}

		// Benchmark control serialization
		controlData, err := json.Marshal(control)
		if err != nil {
			b.Fatalf("Failed to marshal control: %v", err)
		}

		var unmarshaledControl domain.Control
		err = json.Unmarshal(controlData, &unmarshaledControl)
		if err != nil {
			b.Fatalf("Failed to unmarshal control: %v", err)
		}
	}
}

// BenchmarkDataService_GetEvidenceTask benchmarks individual evidence task retrieval
func BenchmarkDataService_GetEvidenceTask(b *testing.B) {
	ctx := context.Background()
	dataService := setupBenchmarkDataService(b)

	taskIDs := []int{327992, 327993, 327994, 327995, 327996}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		taskID := taskIDs[i%len(taskIDs)]
		_, err := dataService.GetEvidenceTask(ctx, taskID)
		if err != nil {
			b.Fatalf("Failed to get evidence task %d: %v", taskID, err)
		}
	}
}

// BenchmarkDataService_GetAllControls benchmarks control retrieval
func BenchmarkDataService_GetAllControls(b *testing.B) {
	ctx := context.Background()
	dataService := setupBenchmarkDataService(b)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		controls, err := dataService.GetAllControls(ctx)
		if err != nil {
			b.Fatalf("Failed to get all controls: %v", err)
		}

		// Use the controls to prevent optimization
		_ = len(controls)
	}
}

// BenchmarkDataService_GetAllPolicies benchmarks policy retrieval
func BenchmarkDataService_GetAllPolicies(b *testing.B) {
	ctx := context.Background()
	dataService := setupBenchmarkDataService(b)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		policies, err := dataService.GetAllPolicies(ctx)
		if err != nil {
			b.Fatalf("Failed to get all policies: %v", err)
		}

		// Use the policies to prevent optimization
		_ = len(policies)
	}
}

// BenchmarkDataService_FilterOperations benchmarks complex filtering operations
func BenchmarkDataService_FilterOperations(b *testing.B) {
	ctx := context.Background()
	dataService := setupBenchmarkDataService(b)

	// Create complex filters for benchmarking
	filters := []domain.EvidenceFilter{
		{
			Status:    []string{"open", "in_progress"},
			Priority:  []string{"high"},
			Framework: "SOC2",
			Sensitive: &[]bool{true}[0],
		},
		{
			Status:          []string{"closed", "pending"},
			Priority:        []string{"medium", "low"},
			Category:        []string{"technical", "administrative"},
			ComplexityLevel: []string{"high", "medium"},
		},
		{
			AecStatus:      []string{"approved", "pending"},
			CollectionType: []string{"manual", "automated"},
			DueBefore:      &[]time.Time{time.Now().Add(30 * 24 * time.Hour)}[0],
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		filter := filters[i%len(filters)]
		_, err := dataService.FilterEvidenceTasks(ctx, filter)
		if err != nil {
			b.Fatalf("Failed to filter evidence tasks: %v", err)
		}
	}
}

// BenchmarkDataService_Relationships benchmarks relationship retrieval
func BenchmarkDataService_Relationships(b *testing.B) {
	ctx := context.Background()
	dataService := setupBenchmarkDataService(b)

	sourceTypes := []string{"evidence_task", "control", "policy"}
	sourceIDs := []string{"327992", "778805", "94641"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		sourceType := sourceTypes[i%len(sourceTypes)]
		sourceID := sourceIDs[i%len(sourceIDs)]

		_, err := dataService.GetRelationships(ctx, sourceType, sourceID)
		if err != nil {
			b.Fatalf("Failed to get relationships for %s:%s: %v", sourceType, sourceID, err)
		}
	}
}

// BenchmarkDataService_EvidenceRecords benchmarks evidence record operations
func BenchmarkDataService_EvidenceRecords(b *testing.B) {
	ctx := context.Background()
	dataService := setupBenchmarkDataService(b)

	evidenceRecord := createLargeEvidenceRecord()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Benchmark save operation
		err := dataService.SaveEvidenceRecord(ctx, evidenceRecord)
		if err != nil {
			b.Fatalf("Failed to save evidence record: %v", err)
		}

		// Benchmark retrieval operation
		_, err = dataService.GetEvidenceRecords(ctx, evidenceRecord.TaskID)
		if err != nil {
			b.Fatalf("Failed to get evidence records: %v", err)
		}
	}
}

// BenchmarkDataService_ConcurrentOperations benchmarks concurrent data operations
func BenchmarkDataService_ConcurrentOperations(b *testing.B) {
	ctx := context.Background()
	dataService := setupBenchmarkDataService(b)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Launch concurrent operations
		done := make(chan error, 3)

		go func() {
			_, err := dataService.GetAllEvidenceTasks(ctx)
			done <- err
		}()

		go func() {
			_, err := dataService.GetAllControls(ctx)
			done <- err
		}()

		go func() {
			_, err := dataService.GetAllPolicies(ctx)
			done <- err
		}()

		// Wait for all operations to complete
		for j := 0; j < 3; j++ {
			if err := <-done; err != nil {
				b.Fatalf("Concurrent operation failed: %v", err)
			}
		}
	}
}

// BenchmarkDataService_MemoryIntensive benchmarks memory-intensive operations
func BenchmarkDataService_MemoryIntensive(b *testing.B) {
	ctx := context.Background()
	dataService := setupBenchmarkDataService(b)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Get all data and perform operations that require memory allocation
		tasks, err := dataService.GetAllEvidenceTasks(ctx)
		if err != nil {
			b.Fatalf("Failed to get evidence tasks: %v", err)
		}

		controls, err := dataService.GetAllControls(ctx)
		if err != nil {
			b.Fatalf("Failed to get controls: %v", err)
		}

		policies, err := dataService.GetAllPolicies(ctx)
		if err != nil {
			b.Fatalf("Failed to get policies: %v", err)
		}

		// Perform memory-intensive operations
		combinedData := make(map[string]interface{})
		combinedData["tasks"] = tasks
		combinedData["controls"] = controls
		combinedData["policies"] = policies

		// Serialize to JSON (memory intensive)
		_, err = json.Marshal(combinedData)
		if err != nil {
			b.Fatalf("Failed to marshal combined data: %v", err)
		}

		// Create large string operations
		var builder strings.Builder
		for _, task := range tasks {
			builder.WriteString(fmt.Sprintf("Task: %s - %s\n", task.ReferenceID, task.Name))
		}
		for _, control := range controls {
			builder.WriteString(fmt.Sprintf("Control: %s - %s\n", control.ReferenceID, control.Name))
		}
		for _, policy := range policies {
			builder.WriteString(fmt.Sprintf("Policy: %s - %s\n", policy.ReferenceID, policy.Name))
		}

		result := builder.String()
		if len(result) == 0 {
			b.Fatal("Combined string is empty")
		}
	}
}

// Helper functions for benchmarking

func setupBenchmarkDataService(b *testing.B) DataService {
	// Create temporary directory for test storage
	tempDir := b.TempDir()

	// Initialize storage
	storage, err := storage.NewStorage(config.StorageConfig{DataDir: tempDir})
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}

	// Create data service
	dataService := NewDataService(storage)

	// Setup mock data for benchmarking
	setupMockDataForBenchmark(b, storage)

	return dataService
}

func setupMockDataForBenchmark(b *testing.B, storage *storage.Storage) {
	// Note: In a real implementation, we would populate the storage with mock data
	// For benchmarking purposes, we're creating placeholders
	b.Log("Setting up mock data for benchmarking")
}

func createLargeEvidenceTask() *domain.EvidenceTask {
	longDescription := strings.Repeat("This is a detailed description of the evidence task with comprehensive requirements and implementation details. ", 20)

	return &domain.EvidenceTask{
		ID:          327992,
		ReferenceID: "ET-0001",
		Name:        "Comprehensive Access Control Implementation Evidence Collection",
		Description: longDescription,
		Framework:   "SOC2",
		Priority:    "High",
		Status:      "in_progress",
		Controls:    []string{"AC-01", "AC-02", "AC-03", "AC-04", "AC-05"},
		Sensitive:   true,
		NextDue:     &[]time.Time{time.Now().Add(30 * 24 * time.Hour)}[0],
		AecStatus: &domain.AecStatus{
			Status:        "approved",
			LastExecuted:  &[]time.Time{time.Now().Add(-7 * 24 * time.Hour)}[0],
			NextScheduled: &[]time.Time{time.Now().Add(30 * 24 * time.Hour)}[0],
		},
	}
}

func createLargePolicy() *domain.Policy {
	longContent := strings.Repeat("This is comprehensive policy content with detailed procedures, requirements, and implementation guidelines. ", 50)

	return &domain.Policy{
		ID:          "94641",
		ReferenceID: "POL-0001",
		Name:        "Comprehensive Information Security Policy with Detailed Procedures",
		Content:     longContent,
		Framework:   "SOC2",
		Version:     "2.1",
		Status:      "active",
		CreatedAt:   time.Now().Add(-30 * 24 * time.Hour),
		UpdatedAt:   time.Now().Add(-7 * 24 * time.Hour),
	}
}

func createLargeControl() *domain.Control {
	longDescription := strings.Repeat("This is a detailed control description with comprehensive implementation guidance and testing procedures. ", 30)

	return &domain.Control{
		ID:              778805,
		ReferenceID:     "AC-01",
		Name:            "Comprehensive Access Control Policy and Procedures Implementation",
		Description:     longDescription,
		Framework:       "SOC2",
		Category:        "Access Control",
		Status:          "implemented",
		ImplementedDate: &[]time.Time{time.Now().Add(-90 * 24 * time.Hour)}[0],
		TestedDate:      &[]time.Time{time.Now().Add(-30 * 24 * time.Hour)}[0],
	}
}

func createLargeEvidenceRecord() *domain.EvidenceRecord {
	largeContent := strings.Repeat("This is comprehensive evidence content with detailed findings, analysis, and supporting documentation. ", 100)

	return &domain.EvidenceRecord{
		ID:          "evidence-record-benchmark-001",
		TaskID:      327992,
		Format:      "markdown",
		Source:      "system-scan",
		Content:     largeContent,
		CollectedBy: "Automated Scanner",
		CollectedAt: time.Now(),
		Metadata: map[string]interface{}{
			"scan_type":      "comprehensive",
			"findings_count": 25,
			"severity":       "high",
			"validated_by":   "Security Analyst",
			"status":         "validated",
		},
	}
}
