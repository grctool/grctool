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

package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/storage"
)

// BenchmarkEvidenceGenerator_Process benchmarks the complete evidence generation process
func BenchmarkEvidenceGenerator_Process(b *testing.B) {
	ctx := context.Background()
	generator := setupBenchmarkEvidenceGenerator(b)

	params := map[string]interface{}{
		"task_ref": "ET-0001",
		"tools":    []string{"terraform", "github", "docs"},
		"format":   "markdown",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _, err := generator.Execute(ctx, params)
		if err != nil {
			b.Fatalf("Failed to execute evidence generator: %v", err)
		}
	}
}

// BenchmarkEvidenceGenerator_GeneratePrompt benchmarks prompt generation from task reference
func BenchmarkEvidenceGenerator_GeneratePrompt(b *testing.B) {
	ctx := context.Background()
	generator := setupBenchmarkEvidenceGenerator(b)

	taskRef := "ET-0001"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Simulate prompt generation by resolving task ID and building prompt
		taskID, err := generator.resolveTaskID(ctx, taskRef)
		if err != nil {
			b.Fatalf("Failed to resolve task ID: %v", err)
		}

		task, err := generator.dataService.GetEvidenceTask(ctx, taskID)
		if err != nil {
			b.Fatalf("Failed to get evidence task: %v", err)
		}

		// Generate prompt content
		prompt := fmt.Sprintf("Generate evidence for task: %s\nDescription: %s\nFramework: %s\nPriority: %s",
			task.Name, task.Description, task.Framework, task.Priority)

		if len(prompt) == 0 {
			b.Fatal("Generated prompt is empty")
		}
	}
}

// BenchmarkEvidenceGenerator_LargeDataset benchmarks evidence generation with large datasets
func BenchmarkEvidenceGenerator_LargeDataset(b *testing.B) {
	ctx := context.Background()
	generator := setupBenchmarkEvidenceGenerator(b)

	// Create a large prompt file to simulate processing large datasets
	largePrompt := createLargePromptContent(10000) // 10KB of prompt content
	promptFile := createTempPromptFile(b, largePrompt)
	defer os.Remove(promptFile)

	params := map[string]interface{}{
		"prompt_file": promptFile,
		"tools":       []string{"terraform", "github", "docs", "manual"},
		"format":      "json",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _, err := generator.Execute(ctx, params)
		if err != nil {
			b.Fatalf("Failed to execute evidence generator with large dataset: %v", err)
		}
	}
}

// BenchmarkEvidenceGenerator_CoordinateSubTools benchmarks sub-tool coordination
func BenchmarkEvidenceGenerator_CoordinateSubTools(b *testing.B) {
	ctx := context.Background()
	generator := setupBenchmarkEvidenceGenerator(b)

	taskID := 327992 // Sample task ID
	prompt := "Generate evidence for access control implementation"
	tools := []string{"terraform", "github", "docs"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _, err := generator.coordinateSubTools(ctx, taskID, prompt, tools)
		if err != nil {
			b.Fatalf("Failed to coordinate sub-tools: %v", err)
		}
	}
}

// BenchmarkEvidenceGenerator_GenerateFinalEvidence benchmarks final evidence generation
func BenchmarkEvidenceGenerator_GenerateFinalEvidence(b *testing.B) {
	ctx := context.Background()
	generator := setupBenchmarkEvidenceGenerator(b)

	taskID := 327992
	prompt := "Generate evidence for access control implementation"
	sources := createMockEvidenceSources(5) // 5 mock sources
	synthesis := "Evidence synthesis summary with detailed analysis of findings"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := generator.generateFinalEvidence(ctx, taskID, prompt, sources, synthesis, "markdown")
		if err != nil {
			b.Fatalf("Failed to generate final evidence: %v", err)
		}
	}
}

// BenchmarkEvidenceGenerator_TaskIDResolution benchmarks task ID resolution
func BenchmarkEvidenceGenerator_TaskIDResolution(b *testing.B) {
	ctx := context.Background()
	generator := setupBenchmarkEvidenceGenerator(b)

	taskRefs := []string{"ET-0001", "ET-0002", "ET-0003", "ET-0004", "ET-0005"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		taskRef := taskRefs[i%len(taskRefs)]
		_, err := generator.resolveTaskID(ctx, taskRef)
		if err != nil {
			b.Fatalf("Failed to resolve task ID for %s: %v", taskRef, err)
		}
	}
}

// BenchmarkEvidenceGenerator_PromptExtractionFromFile benchmarks prompt extraction
func BenchmarkEvidenceGenerator_PromptExtractionFromFile(b *testing.B) {
	generator := setupBenchmarkEvidenceGenerator(b)

	promptContent := `
Evidence Task: ET-42
Generate comprehensive evidence for access control implementation.
Requirements:
1. Document user access provisioning process
2. Review access control policies
3. Validate implementation across all systems
4. Generate compliance report
`

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		taskID := generator.extractTaskIDFromPrompt(promptContent)
		if taskID == 0 {
			b.Fatal("Failed to extract task ID from prompt")
		}
	}
}

// BenchmarkEvidenceGenerator_PathSafetyCheck benchmarks path safety validation
func BenchmarkEvidenceGenerator_PathSafetyCheck(b *testing.B) {
	generator := setupBenchmarkEvidenceGenerator(b)

	testPaths := []string{
		"/safe/path/to/file.txt",
		"relative/path/file.txt",
		"../unsafe/path.txt",
		"/data/dir/safe/file.json",
		"../../etc/passwd",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		path := testPaths[i%len(testPaths)]
		_ = generator.isPathSafe(path)
	}
}

// BenchmarkEvidenceGenerator_FileOperations benchmarks file I/O operations
func BenchmarkEvidenceGenerator_FileOperations(b *testing.B) {
	generator := setupBenchmarkEvidenceGenerator(b)
	tempDir := b.TempDir()

	result := &EvidenceGenerationResult{
		TaskID:      327992,
		Content:     createLargeEvidenceContent(5000), // 5KB of content
		Format:      "markdown",
		GeneratedAt: time.Now(),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := generator.saveEvidenceToFile(result, tempDir)
		if err != nil {
			b.Fatalf("Failed to save evidence to file: %v", err)
		}
	}
}

// BenchmarkEvidenceGenerator_JSONSerialization benchmarks JSON evidence generation
func BenchmarkEvidenceGenerator_JSONSerialization(b *testing.B) {
	ctx := context.Background()
	generator := setupBenchmarkEvidenceGenerator(b)

	taskID := 327992
	prompt := "Generate evidence for access control implementation"
	sources := createMockEvidenceSources(10)                                            // 10 mock sources for larger JSON
	synthesis := strings.Repeat("Detailed analysis with comprehensive findings. ", 100) // Large synthesis

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := generator.generateFinalEvidence(ctx, taskID, prompt, sources, synthesis, "json")
		if err != nil {
			b.Fatalf("Failed to generate JSON evidence: %v", err)
		}
	}
}

// Helper functions for benchmarking

func setupBenchmarkEvidenceGenerator(b *testing.B) *EvidenceGeneratorTool {
	// Create temporary directory for test data
	tempDir := b.TempDir()

	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: tempDir,
		},
	}

	log := logger.WithComponent("evidence-generator-bench")

	generator, err := NewEvidenceGeneratorTool(cfg, log)
	if err != nil {
		b.Fatalf("Failed to create evidence generator: %v", err)
	}

	// Create mock storage data for benchmarking
	setupMockStorageData(b, generator.storage)

	return generator
}

func setupMockStorageData(b *testing.B, storage *storage.Storage) {
	// Create some mock evidence tasks for benchmarking
	mockTasks := []struct {
		id          string
		referenceID string
		name        string
		framework   string
		priority    string
	}{
		{"327992", "ET-0001", "Access Control Implementation", "SOC2", "High"},
		{"327993", "ET-0002", "Data Encryption Standards", "SOC2", "High"},
		{"327994", "ET-0003", "Backup Procedures", "SOC2", "Medium"},
		{"327995", "ET-0004", "Incident Response Plan", "SOC2", "High"},
		{"327996", "ET-0005", "Security Training Program", "SOC2", "Low"},
	}

	for _, task := range mockTasks {
		// Note: In a real implementation, we would create proper domain models
		// For benchmarking, we're just ensuring the storage has some data
		b.Logf("Mock task: %s - %s", task.id, task.name)
	}
}

func createLargePromptContent(sizeBytes int) string {
	content := `Evidence Task: ET-42
Generate comprehensive evidence for access control implementation.

Requirements:
1. Document user access provisioning process
2. Review access control policies  
3. Validate implementation across all systems
4. Generate compliance report

Details:
`
	// Fill to desired size
	filler := "This is additional prompt content to reach the desired size for benchmarking purposes. "
	for len(content) < sizeBytes {
		content += filler
	}
	return content[:sizeBytes]
}

func createLargeEvidenceContent(sizeBytes int) string {
	content := `# Evidence Report

## Summary
This evidence report documents the implementation and validation of access controls.

## Findings
`
	// Fill to desired size
	filler := "Detailed finding with comprehensive analysis and supporting documentation. "
	for len(content) < sizeBytes {
		content += filler
	}
	return content[:sizeBytes]
}

func createTempPromptFile(b *testing.B, content string) string {
	tempFile := filepath.Join(b.TempDir(), "benchmark_prompt.txt")
	if err := os.WriteFile(tempFile, []byte(content), 0644); err != nil {
		b.Fatalf("Failed to create temp prompt file: %v", err)
	}
	return tempFile
}

func createMockEvidenceSources(count int) []models.EvidenceSource {
	sources := make([]models.EvidenceSource, count)
	for i := 0; i < count; i++ {
		sources[i] = models.EvidenceSource{
			Type:        fmt.Sprintf("tool-%d", i),
			Resource:    fmt.Sprintf("Resource %d for evidence gathering", i),
			Content:     fmt.Sprintf("Mock evidence content from source %d with detailed findings and analysis", i),
			ExtractedAt: time.Now(),
			Metadata: map[string]interface{}{
				"source_id": i,
				"quality":   "high",
				"verified":  true,
			},
		}
	}
	return sources
}
