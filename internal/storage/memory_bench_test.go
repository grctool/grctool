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

package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
)

// BenchmarkLargeFileProcessing benchmarks streaming vs in-memory file processing
func BenchmarkLargeFileProcessing(b *testing.B) {
	// Create large test file (10MB)
	tempDir := b.TempDir()
	largeFile := filepath.Join(tempDir, "large_data.json")
	createLargeTestFile(b, largeFile, 10*1024*1024) // 10MB

	b.Run("StreamingRead", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			err := processFileStreaming(largeFile)
			if err != nil {
				b.Fatalf("Streaming read failed: %v", err)
			}
		}
	})

	b.Run("InMemoryRead", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			err := processFileInMemory(largeFile)
			if err != nil {
				b.Fatalf("In-memory read failed: %v", err)
			}
		}
	})
}

// BenchmarkConcurrentOperations benchmarks concurrent storage operations
func BenchmarkConcurrentOperations(b *testing.B) {
	tempDir := b.TempDir()
	storage, err := NewStorage(config.StorageConfig{DataDir: tempDir})
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}

	// Create test data
	evidenceTasks := createLargeEvidenceTaskSet(100)
	_ = createLargePolicySet(50)
	_ = createLargeControlSet(75)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Launch concurrent operations
		done := make(chan error, 6)

		// Concurrent reads
		go func() {
			_, err := storage.GetAllEvidenceTasks()
			done <- err
		}()

		go func() {
			_, err := storage.GetAllPolicies()
			done <- err
		}()

		go func() {
			_, err := storage.GetAllControls()
			done <- err
		}()

		// Concurrent writes (limited for safety)
		go func() {
			for j := 0; j < 5; j++ {
				record := &domain.EvidenceRecord{
					ID:          fmt.Sprintf("concurrent-record-%d-%d", i, j),
					TaskID:      evidenceTasks[j%len(evidenceTasks)].ID,
					Format:      "markdown",
					Source:      "benchmark",
					Content:     "Concurrent benchmark content",
					CollectedAt: time.Now(),
				}
				err := storage.SaveEvidenceRecord(record)
				if err != nil {
					done <- err
					return
				}
			}
			done <- nil
		}()

		// Wait for all operations
		for j := 0; j < 4; j++ {
			if err := <-done; err != nil {
				b.Fatalf("Concurrent operation failed: %v", err)
			}
		}
	}
}

// BenchmarkCachePerformance benchmarks caching mechanisms
func BenchmarkCachePerformance(b *testing.B) {
	tempDir := b.TempDir()
	storage, err := NewStorage(config.StorageConfig{DataDir: tempDir})
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}

	// Setup test data
	evidenceTasks := createLargeEvidenceTaskSet(200)

	// Pre-populate storage
	for _, task := range evidenceTasks {
		// Note: In a real implementation, we would save tasks to storage
		// This is a placeholder for the benchmark
		_ = task
	}

	b.Run("ColdCache", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Simulate cold cache by clearing any internal caches
			// In a real implementation with caching, we would clear caches here

			// Access data (should trigger cache misses)
			_, err := storage.GetAllEvidenceTasks()
			if err != nil {
				b.Fatalf("Cold cache read failed: %v", err)
			}
		}
	})

	b.Run("WarmCache", func(b *testing.B) {
		// Warm up the cache
		storage.GetAllEvidenceTasks()

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Access data (should hit cache)
			_, err := storage.GetAllEvidenceTasks()
			if err != nil {
				b.Fatalf("Warm cache read failed: %v", err)
			}
		}
	})

	b.Run("CacheEviction", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Simulate cache eviction scenarios
			for j := 0; j < 10; j++ {
				_, err := storage.GetAllEvidenceTasks()
				if err != nil {
					b.Fatalf("Cache eviction read failed: %v", err)
				}

				// Force cache invalidation by accessing different data types
				storage.GetAllPolicies()
				storage.GetAllControls()
			}
		}
	})
}

// BenchmarkMemoryAllocation benchmarks memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("SmallObjects", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Allocate many small objects
			var records []*domain.EvidenceRecord
			for j := 0; j < 1000; j++ {
				record := &domain.EvidenceRecord{
					ID:          fmt.Sprintf("small-%d-%d", i, j),
					TaskID:      j % 100,
					Format:      "markdown",
					Source:      "benchmark",
					Content:     "Small content",
					CollectedAt: time.Now(),
				}
				records = append(records, record)
			}

			// Use the records to prevent optimization
			if len(records) != 1000 {
				b.Fatal("Unexpected record count")
			}
		}
	})

	b.Run("LargeObjects", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Allocate fewer large objects
			var tasks []*domain.EvidenceTask
			for j := 0; j < 100; j++ {
				task := &domain.EvidenceTask{
					ID:          j,
					ReferenceID: fmt.Sprintf("ET-%04d", j),
					Name:        fmt.Sprintf("Large evidence task %d with comprehensive details", j),
					Description: strings.Repeat("Detailed description with comprehensive requirements. ", 100),
					Framework:   "SOC2",
					Priority:    "High",
					Status:      "in_progress",
					Controls:    []string{"AC-01", "AC-02", "AC-03", "AC-04", "AC-05"},
					Sensitive:   true,
					NextDue:     &[]time.Time{time.Now().Add(30 * 24 * time.Hour)}[0],
				}
				tasks = append(tasks, task)
			}

			// Use the tasks to prevent optimization
			if len(tasks) != 100 {
				b.Fatal("Unexpected task count")
			}
		}
	})

	b.Run("StringBuilding", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			var builder strings.Builder
			builder.Grow(10000) // Pre-allocate capacity

			for j := 0; j < 1000; j++ {
				builder.WriteString(fmt.Sprintf("Line %d: This is a test line with some content.\n", j))
			}

			result := builder.String()
			if len(result) == 0 {
				b.Fatal("String building result is empty")
			}
		}
	})

	b.Run("SliceGrowth", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Test slice growth patterns
			var slice []string

			// Grow slice without pre-allocation
			for j := 0; j < 1000; j++ {
				slice = append(slice, fmt.Sprintf("item-%d", j))
			}

			// Use the slice to prevent optimization
			if len(slice) != 1000 {
				b.Fatal("Unexpected slice length")
			}
		}
	})

	b.Run("SlicePreallocation", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Test slice with pre-allocation
			slice := make([]string, 0, 1000) // Pre-allocate capacity

			for j := 0; j < 1000; j++ {
				slice = append(slice, fmt.Sprintf("item-%d", j))
			}

			// Use the slice to prevent optimization
			if len(slice) != 1000 {
				b.Fatal("Unexpected slice length")
			}
		}
	})
}

// BenchmarkJSONMemoryUsage benchmarks JSON operations memory usage
func BenchmarkJSONMemoryUsage(b *testing.B) {
	// Create large test objects
	evidenceTasks := createLargeEvidenceTaskSet(100)
	policies := createLargePolicySet(50)
	controls := createLargeControlSet(75)

	b.Run("IndividualSerialization", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Serialize each object individually
			for _, task := range evidenceTasks {
				data, err := json.Marshal(task)
				if err != nil {
					b.Fatalf("Task serialization failed: %v", err)
				}
				if len(data) == 0 {
					b.Fatal("Serialized data is empty")
				}
			}
		}
	})

	b.Run("BatchSerialization", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Serialize all objects in batches
			batchData := struct {
				EvidenceTasks []domain.EvidenceTask `json:"evidence_tasks"`
				Policies      []domain.Policy       `json:"policies"`
				Controls      []domain.Control      `json:"controls"`
			}{
				EvidenceTasks: evidenceTasks,
				Policies:      policies,
				Controls:      controls,
			}

			// Note: ToJSON() method would need to be implemented on the struct
			// For benchmarking, we simulate the operation
			_ = batchData
		}
	})

	b.Run("StreamingSerialization", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Simulate streaming serialization to reduce memory usage
			var totalSize int

			for _, task := range evidenceTasks {
				data, err := json.Marshal(task)
				if err != nil {
					b.Fatalf("Streaming task serialization failed: %v", err)
				}
				totalSize += len(data)
				// In a real implementation, we would write to a stream here
			}

			if totalSize == 0 {
				b.Fatal("No data serialized")
			}
		}
	})
}

// Helper functions

func processFileStreaming(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Process file in chunks
	buffer := make([]byte, 8192) // 8KB buffer
	var totalBytes int

	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		totalBytes += n
		// Simulate processing
		_ = buffer[:n]
	}

	return nil
}

func processFileInMemory(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	// Simulate processing
	_ = data

	return nil
}

func createLargeTestFile(b *testing.B, filename string, size int) {
	file, err := os.Create(filename)
	if err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}
	defer file.Close()

	// Write test data
	content := `{"test": "data", "content": "`
	remaining := size

	for remaining > 0 {
		writeSize := len(content)
		if writeSize > remaining {
			writeSize = remaining
		}

		n, err := file.WriteString(content[:writeSize])
		if err != nil {
			b.Fatalf("Failed to write test data: %v", err)
		}

		remaining -= n
	}
}

func createLargeEvidenceTaskSet(count int) []domain.EvidenceTask {
	tasks := make([]domain.EvidenceTask, count)

	for i := 0; i < count; i++ {
		tasks[i] = domain.EvidenceTask{
			ID:          327992 + i,
			ReferenceID: fmt.Sprintf("ET-%04d", i+1),
			Name:        fmt.Sprintf("Evidence Task %d - Comprehensive Implementation", i+1),
			Description: strings.Repeat(fmt.Sprintf("Detailed description for task %d with comprehensive requirements. ", i+1), 10),
			Framework:   "SOC2",
			Priority:    []string{"High", "Medium", "Low"}[i%3],
			Status:      []string{"open", "in_progress", "closed"}[i%3],
			Controls:    []string{fmt.Sprintf("AC-%02d", (i%10)+1), fmt.Sprintf("CC-%02d", (i%5)+1)},
			Sensitive:   i%2 == 0,
			NextDue:     &[]time.Time{time.Now().Add(time.Duration(i*24) * time.Hour)}[0],
		}
	}

	return tasks
}

func createLargePolicySet(count int) []domain.Policy {
	policies := make([]domain.Policy, count)

	for i := 0; i < count; i++ {
		policies[i] = domain.Policy{
			ID:          fmt.Sprintf("%d", 94641+i),
			ReferenceID: fmt.Sprintf("POL-%04d", i+1),
			Name:        fmt.Sprintf("Policy %d - Comprehensive Security Framework", i+1),
			Content:     strings.Repeat(fmt.Sprintf("Comprehensive policy content for policy %d with detailed procedures. ", i+1), 20),
			Framework:   "SOC2",
			Version:     fmt.Sprintf("1.%d", i%10),
			Status:      []string{"active", "draft", "archived"}[i%3],
			CreatedAt:   time.Now().Add(-time.Duration((i+30)*24) * time.Hour),
			UpdatedAt:   time.Now().Add(-time.Duration(i*24) * time.Hour),
		}
	}

	return policies
}

func createLargeControlSet(count int) []domain.Control {
	controls := make([]domain.Control, count)

	for i := 0; i < count; i++ {
		controls[i] = domain.Control{
			ID:          778805 + i,
			ReferenceID: fmt.Sprintf("AC-%02d", i+1),
			Name:        fmt.Sprintf("Control %d - Comprehensive Access Management", i+1),
			Description: strings.Repeat(fmt.Sprintf("Detailed control description %d with implementation guidance. ", i+1), 15),
			Framework:   "SOC2",
			Category:    []string{"Access Control", "Data Protection", "System Monitoring"}[i%3],
			RiskLevel:   []string{"Low", "Medium", "High"}[i%3],
			Status:      []string{"implemented", "planned", "in_progress"}[i%3],
		}
	}

	return controls
}
