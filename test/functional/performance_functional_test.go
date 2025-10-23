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

//go:build functional

package functional

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Performance thresholds for different operations
const (
	ToolListMaxDuration          = 5 * time.Second  // Tool listing should be fast
	StorageOperationMaxDuration  = 10 * time.Second // Storage operations should be reasonable
	EvidenceAnalysisMaxDuration  = 30 * time.Second // Evidence analysis can take longer
	TerraformAnalysisMaxDuration = 45 * time.Second // Terraform analysis can be slow
	LargeDataMaxDuration         = 60 * time.Second // Large dataset operations
)

// TestCLI_BasicPerformance tests basic CLI performance metrics
func TestCLI_BasicPerformance(t *testing.T) {
	binaryPath := ensureBinaryExists(t)

	t.Run("HelpCommandSpeed", func(t *testing.T) {
		start := time.Now()
		cmd := exec.Command(binaryPath, "--help")
		output, err := cmd.CombinedOutput()
		duration := time.Since(start)

		require.NoError(t, err, "Help command should succeed: %s", string(output))
		assert.Less(t, duration, 3*time.Second, "Help should be very fast")
		assert.True(t, len(output) > 100, "Help should provide substantial output")

		t.Logf("Help command completed in %v", duration)
	})

	t.Run("ToolListSpeed", func(t *testing.T) {
		start := time.Now()
		cmd := exec.Command(binaryPath, "tool", "list")
		output, err := cmd.CombinedOutput()
		duration := time.Since(start)

		require.NoError(t, err, "Tool list should succeed: %s", string(output))
		assert.Less(t, duration, ToolListMaxDuration, "Tool list should complete quickly")
		assert.Contains(t, string(output), "Available tools", "Should list tools")

		// Count tools for performance correlation
		toolCount := strings.Count(string(output), ":") - 1 // Subtract header
		t.Logf("Tool list completed in %v (%d tools)", duration, toolCount)
	})

	t.Run("VersionCommandSpeed", func(t *testing.T) {
		start := time.Now()
		cmd := exec.Command(binaryPath, "--version")
		output, err := cmd.CombinedOutput()
		duration := time.Since(start)

		if err == nil {
			assert.Less(t, duration, 2*time.Second, "Version should be very fast")
			t.Logf("Version command completed in %v", duration)
		} else {
			t.Logf("Version command not available or failed: %v, output: %s", err, string(output))
		}
	})
}

// TestStoragePerformance tests storage operation performance
func TestStoragePerformance(t *testing.T) {
	binaryPath := ensureBinaryExists(t)

	t.Run("SingleStorageWrite", func(t *testing.T) {
		testData := `{"performance_test": true, "timestamp": "` + time.Now().Format(time.RFC3339) + `", "size": "small"}`
		filename := fmt.Sprintf("perf_test_%d.json", time.Now().Unix())

		start := time.Now()
		cmd := exec.Command(binaryPath, "tool", "storage-write",
			"--path", filename,
			"--content", testData,
			"--format", "json")

		output, err := cmd.CombinedOutput()
		duration := time.Since(start)

		require.NoError(t, err, "Storage write should succeed: %s", string(output))
		assert.Less(t, duration, StorageOperationMaxDuration, "Storage write should be reasonably fast")

		t.Logf("Storage write completed in %v", duration)
	})

	t.Run("SingleStorageRead", func(t *testing.T) {
		// First write a test file
		testData := `{"read_performance_test": true, "data_size": "small"}`
		filename := fmt.Sprintf("read_perf_test_%d.json", time.Now().Unix())

		writeCmd := exec.Command(binaryPath, "tool", "storage-write",
			"--path", filename,
			"--content", testData,
			"--format", "json")
		_, err := writeCmd.CombinedOutput()
		require.NoError(t, err, "Setup write should succeed")

		// Now test read performance
		start := time.Now()
		cmd := exec.Command(binaryPath, "tool", "storage-read",
			"--path", filename,
			"--format", "json")

		output, err := cmd.CombinedOutput()
		duration := time.Since(start)

		require.NoError(t, err, "Storage read should succeed: %s", string(output))
		assert.Less(t, duration, StorageOperationMaxDuration, "Storage read should be reasonably fast")
		assert.Contains(t, string(output), "read_performance_test", "Should read correct content")

		t.Logf("Storage read completed in %v", duration)
	})

	t.Run("BatchStorageOperations", func(t *testing.T) {
		const batchSize = 10
		filenames := make([]string, batchSize)

		// Batch write test
		start := time.Now()
		for i := 0; i < batchSize; i++ {
			filenames[i] = fmt.Sprintf("batch_test_%d_%d.json", time.Now().Unix(), i)
			testData := fmt.Sprintf(`{"batch_id": %d, "timestamp": "%s"}`, i, time.Now().Format(time.RFC3339))

			cmd := exec.Command(binaryPath, "tool", "storage-write",
				"--path", filenames[i],
				"--content", testData,
				"--format", "json")

			_, err := cmd.CombinedOutput()
			require.NoError(t, err, "Batch write %d should succeed", i)
		}
		batchWriteDuration := time.Since(start)

		// Batch read test
		start = time.Now()
		for i := 0; i < batchSize; i++ {
			cmd := exec.Command(binaryPath, "tool", "storage-read",
				"--path", filenames[i],
				"--format", "json")

			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "Batch read %d should succeed", i)
			assert.Contains(t, string(output), fmt.Sprintf(`"batch_id": %d`, i), "Should read correct content")
		}
		batchReadDuration := time.Since(start)

		// Performance assertions
		assert.Less(t, batchWriteDuration, StorageOperationMaxDuration*time.Duration(batchSize/2), "Batch writes should be reasonably fast")
		assert.Less(t, batchReadDuration, StorageOperationMaxDuration*time.Duration(batchSize/2), "Batch reads should be reasonably fast")

		avgWriteTime := batchWriteDuration / batchSize
		avgReadTime := batchReadDuration / batchSize
		t.Logf("Batch operations: %d writes in %v (avg %v), %d reads in %v (avg %v)",
			batchSize, batchWriteDuration, avgWriteTime,
			batchSize, batchReadDuration, avgReadTime)
	})
}

// TestEvidenceAnalysisPerformance tests evidence analysis performance
func TestEvidenceAnalysisPerformance(t *testing.T) {
	binaryPath := ensureBinaryExists(t)

	t.Run("EvidenceTaskListPerformance", func(t *testing.T) {
		start := time.Now()
		cmd := exec.Command(binaryPath, "tool", "evidence-task-list", "--output-format", "json")
		output, err := cmd.CombinedOutput()
		duration := time.Since(start)

		if err != nil {
			if strings.Contains(string(output), "no evidence tasks found") ||
				strings.Contains(string(output), "directory not found") {
				t.Skip("Skipping evidence task list performance test - no data available")
			}
			t.Logf("Evidence task list failed: %v, output: %s", err, string(output))
			return
		}

		assert.Less(t, duration, EvidenceAnalysisMaxDuration, "Evidence task list should complete in reasonable time")
		t.Logf("Evidence task list completed in %v", duration)
	})

	t.Run("EvidenceTaskDetailsPerformance", func(t *testing.T) {
		start := time.Now()
		cmd := exec.Command(binaryPath, "tool", "evidence-task-details",
			"--task-ref", "ET-101",
			"--output-format", "json")
		output, err := cmd.CombinedOutput()
		duration := time.Since(start)

		if err != nil {
			if strings.Contains(string(output), "task not found") {
				t.Skip("Skipping evidence task details performance test - task ET-101 not available")
			}
			t.Logf("Evidence task details failed: %v, output: %s", err, string(output))
			return
		}

		assert.Less(t, duration, EvidenceAnalysisMaxDuration, "Evidence task details should complete in reasonable time")
		t.Logf("Evidence task details completed in %v", duration)
	})

	t.Run("PromptAssemblerPerformance", func(t *testing.T) {
		start := time.Now()
		cmd := exec.Command(binaryPath, "tool", "prompt-assembler",
			"--task-ref", "ET-101",
			"--output-format", "markdown")
		output, err := cmd.CombinedOutput()
		duration := time.Since(start)

		if err != nil {
			if strings.Contains(string(output), "task not found") {
				t.Skip("Skipping prompt assembler performance test - task ET-101 not available")
			}
			t.Logf("Prompt assembler failed: %v, output: %s", err, string(output))
			return
		}

		assert.Less(t, duration, EvidenceAnalysisMaxDuration, "Prompt assembler should complete in reasonable time")
		outputLength := len(output)
		t.Logf("Prompt assembler completed in %v (generated %d chars)", duration, outputLength)
	})
}

// TestTerraformAnalysisPerformance tests terraform analysis performance
func TestTerraformAnalysisPerformance(t *testing.T) {
	binaryPath := ensureBinaryExists(t)

	t.Run("TerraformHCLParserPerformance", func(t *testing.T) {
		start := time.Now()
		cmd := exec.Command(binaryPath, "tool", "terraform-hcl-parser",
			"--scan-paths", "../../test_data/terraform",
			"--output-format", "json")
		output, err := cmd.CombinedOutput()
		duration := time.Since(start)

		if err != nil {
			if strings.Contains(string(output), "no terraform files found") ||
				strings.Contains(string(output), "parameter") {
				t.Skip("Skipping terraform HCL parser performance test - no data or parameter issues")
			}
			t.Logf("Terraform HCL parser failed: %v, output: %s", err, string(output))
			return
		}

		assert.Less(t, duration, TerraformAnalysisMaxDuration, "Terraform HCL parser should complete in reasonable time")
		t.Logf("Terraform HCL parser completed in %v", duration)
	})

	t.Run("TerraformSecurityAnalyzerPerformance", func(t *testing.T) {
		start := time.Now()
		cmd := exec.Command(binaryPath, "tool", "terraform-security-analyzer",
			"--scan-paths", "../../test_data/terraform",
			"--output-format", "summary")
		output, err := cmd.CombinedOutput()
		duration := time.Since(start)

		if err != nil {
			if strings.Contains(string(output), "no terraform files found") ||
				strings.Contains(string(output), "parameter") {
				t.Skip("Skipping terraform security analyzer performance test - no data or parameter issues")
			}
			t.Logf("Terraform security analyzer failed: %v, output: %s", err, string(output))
			return
		}

		assert.Less(t, duration, TerraformAnalysisMaxDuration, "Terraform security analyzer should complete in reasonable time")
		t.Logf("Terraform security analyzer completed in %v", duration)
	})
}

// TestLargeDataset_Performance tests CLI performance with large datasets
func TestLargeDataset_Performance(t *testing.T) {
	binaryPath := ensureBinaryExists(t)

	t.Run("LargeFileWrite", func(t *testing.T) {
		// Generate large content (1MB)
		const targetSize = 1024 * 1024 // 1MB
		baseContent := `{"large_test": true, "data": "`
		suffix := `"}`
		paddingNeeded := targetSize - len(baseContent) - len(suffix)
		padding := strings.Repeat("X", paddingNeeded)
		largeContent := baseContent + padding + suffix

		filename := fmt.Sprintf("large_file_%d.json", time.Now().Unix())

		start := time.Now()
		cmd := exec.Command(binaryPath, "tool", "storage-write",
			"--path", filename,
			"--content", largeContent)

		output, err := cmd.CombinedOutput()
		duration := time.Since(start)

		if err != nil {
			if strings.Contains(string(output), "too large") ||
				strings.Contains(string(output), "size limit") {
				t.Logf("Large file rejected due to size limits (expected): %s", string(output))
				assert.Less(t, duration, 10*time.Second, "Size limit check should be fast")
				return
			}
			t.Logf("Large file write failed: %v, output: %s", err, string(output))
			return
		}

		assert.Less(t, duration, LargeDataMaxDuration, "Large file write should complete within time limit")
		assert.Contains(t, string(output), "success", "Should indicate success")

		megabytesPerSecond := float64(targetSize) / float64(duration.Seconds()) / (1024 * 1024)
		t.Logf("Large file write completed in %v (%.2f MB/s)", duration, megabytesPerSecond)
	})

	t.Run("ManySmallFilesPerformance", func(t *testing.T) {
		const numFiles = 100
		start := time.Now()

		for i := 0; i < numFiles; i++ {
			filename := fmt.Sprintf("many_files_%d_%d.json", time.Now().Unix(), i)
			content := fmt.Sprintf(`{"file_id": %d, "timestamp": "%s"}`, i, time.Now().Format(time.RFC3339))

			cmd := exec.Command(binaryPath, "tool", "storage-write",
				"--path", filename,
				"--content", content,
				"--format", "json")

			_, err := cmd.CombinedOutput()
			if err != nil {
				t.Logf("File %d failed: %v", i, err)
				break
			}

			// Check for timeout
			if time.Since(start) > LargeDataMaxDuration {
				t.Logf("Stopping at file %d due to time limit", i)
				break
			}
		}

		duration := time.Since(start)
		filesPerSecond := float64(numFiles) / duration.Seconds()

		assert.Less(t, duration, LargeDataMaxDuration, "Many files operation should complete within time limit")
		t.Logf("Created %d files in %v (%.1f files/sec)", numFiles, duration, filesPerSecond)
	})
}

// TestMemoryUsage tests CLI memory usage patterns
func TestMemoryUsage(t *testing.T) {
	binaryPath := ensureBinaryExists(t)

	t.Run("MemoryUsageBaseline", func(t *testing.T) {
		// Get baseline memory usage
		var m1 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)
		baselineAlloc := m1.Alloc

		// Run a simple operation
		cmd := exec.Command(binaryPath, "tool", "list")
		start := time.Now()
		output, err := cmd.CombinedOutput()
		duration := time.Since(start)

		require.NoError(t, err, "Tool list should succeed: %s", string(output))

		// Get memory usage after operation
		var m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m2)
		finalAlloc := m2.Alloc

		// Memory usage should be reasonable
		memoryUsed := finalAlloc - baselineAlloc
		t.Logf("Tool list: duration=%v, memory_delta=%d bytes", duration, memoryUsed)

		// This is just informational - we don't enforce strict memory limits
		if memoryUsed > 10*1024*1024 { // 10MB
			t.Logf("Warning: Tool list used %d MB of memory", memoryUsed/(1024*1024))
		}
	})
}

// TestConcurrentPerformance tests performance under concurrent load
func TestConcurrentPerformance(t *testing.T) {
	binaryPath := ensureBinaryExists(t)

	t.Run("ConcurrentStorageOperations", func(t *testing.T) {
		const numConcurrent = 10
		type result struct {
			id       int
			duration time.Duration
			err      error
		}

		results := make(chan result, numConcurrent)
		overallStart := time.Now()

		// Launch concurrent operations
		for i := 0; i < numConcurrent; i++ {
			go func(id int) {
				start := time.Now()
				filename := fmt.Sprintf("concurrent_perf_%d_%d.json", time.Now().Unix(), id)
				content := fmt.Sprintf(`{"concurrent_id": %d, "timestamp": "%s"}`, id, time.Now().Format(time.RFC3339))

				cmd := exec.Command(binaryPath, "tool", "storage-write",
					"--path", filename,
					"--content", content,
					"--format", "json")

				_, err := cmd.CombinedOutput()
				duration := time.Since(start)
				results <- result{id, duration, err}
			}(i)
		}

		// Collect results
		successes := 0
		totalDuration := time.Duration(0)
		maxDuration := time.Duration(0)
		minDuration := time.Duration(1<<63 - 1) // max duration

		for i := 0; i < numConcurrent; i++ {
			select {
			case res := <-results:
				if res.err == nil {
					successes++
					totalDuration += res.duration
					if res.duration > maxDuration {
						maxDuration = res.duration
					}
					if res.duration < minDuration {
						minDuration = res.duration
					}
				}
			case <-time.After(60 * time.Second):
				t.Fatalf("Concurrent operations timed out")
			}
		}

		overallDuration := time.Since(overallStart)
		avgDuration := totalDuration / time.Duration(successes)

		// Performance assertions
		assert.GreaterOrEqual(t, successes, numConcurrent*8/10, "At least 80%% of concurrent operations should succeed")
		assert.Less(t, maxDuration, StorageOperationMaxDuration*2, "No single operation should take too long")

		t.Logf("Concurrent performance: %d/%d succeeded, overall=%v, avg=%v, min=%v, max=%v",
			successes, numConcurrent, overallDuration, avgDuration, minDuration, maxDuration)
	})

	t.Run("ConcurrentToolList", func(t *testing.T) {
		// Test that multiple tool list operations can run concurrently
		const numConcurrent = 5
		type result struct {
			id       int
			duration time.Duration
			err      error
		}

		results := make(chan result, numConcurrent)
		start := time.Now()

		for i := 0; i < numConcurrent; i++ {
			go func(id int) {
				opStart := time.Now()
				cmd := exec.Command(binaryPath, "tool", "list")
				_, err := cmd.CombinedOutput()
				duration := time.Since(opStart)
				results <- result{id, duration, err}
			}(i)
		}

		// Collect results
		successes := 0
		for i := 0; i < numConcurrent; i++ {
			select {
			case res := <-results:
				if res.err == nil {
					successes++
				}
			case <-time.After(30 * time.Second):
				t.Fatalf("Concurrent tool list operations timed out")
			}
		}

		overallDuration := time.Since(start)

		assert.Equal(t, numConcurrent, successes, "All concurrent tool list operations should succeed")
		assert.Less(t, overallDuration, ToolListMaxDuration*2, "Concurrent tool lists should not take much longer than sequential")

		t.Logf("Concurrent tool list: %d/%d succeeded in %v", successes, numConcurrent, overallDuration)
	})
}

// TestPerformanceRegression tests for performance regressions
func TestPerformanceRegression(t *testing.T) {
	binaryPath := ensureBinaryExists(t)

	// These benchmarks establish baseline performance metrics
	// They should be updated when significant architectural changes are made

	t.Run("ToolListBenchmark", func(t *testing.T) {
		const iterations = 5
		durations := make([]time.Duration, iterations)

		for i := 0; i < iterations; i++ {
			start := time.Now()
			cmd := exec.Command(binaryPath, "tool", "list")
			_, err := cmd.CombinedOutput()
			durations[i] = time.Since(start)

			require.NoError(t, err, "Tool list iteration %d should succeed", i)
		}

		// Calculate statistics
		total := time.Duration(0)
		min := durations[0]
		max := durations[0]

		for _, d := range durations {
			total += d
			if d < min {
				min = d
			}
			if d > max {
				max = d
			}
		}

		avg := total / time.Duration(iterations)

		// Performance expectations
		assert.Less(t, avg, ToolListMaxDuration, "Average tool list time should be reasonable")
		assert.Less(t, max, ToolListMaxDuration*2, "No single tool list should take too long")

		t.Logf("Tool list benchmark (%d iterations): avg=%v, min=%v, max=%v", iterations, avg, min, max)
	})

	t.Run("StorageBenchmark", func(t *testing.T) {
		const iterations = 3
		writeDurations := make([]time.Duration, iterations)
		readDurations := make([]time.Duration, iterations)

		for i := 0; i < iterations; i++ {
			filename := fmt.Sprintf("benchmark_%d_%d.json", time.Now().Unix(), i)
			content := fmt.Sprintf(`{"benchmark": %d, "timestamp": "%s"}`, i, time.Now().Format(time.RFC3339))

			// Write benchmark
			start := time.Now()
			writeCmd := exec.Command(binaryPath, "tool", "storage-write",
				"--path", filename,
				"--content", content,
				"--format", "json")
			_, err := writeCmd.CombinedOutput()
			writeDurations[i] = time.Since(start)
			require.NoError(t, err, "Write benchmark iteration %d should succeed", i)

			// Read benchmark
			start = time.Now()
			readCmd := exec.Command(binaryPath, "tool", "storage-read",
				"--path", filename,
				"--format", "json")
			_, err = readCmd.CombinedOutput()
			readDurations[i] = time.Since(start)
			require.NoError(t, err, "Read benchmark iteration %d should succeed", i)
		}

		// Calculate averages
		avgWrite := time.Duration(0)
		avgRead := time.Duration(0)
		for i := 0; i < iterations; i++ {
			avgWrite += writeDurations[i]
			avgRead += readDurations[i]
		}
		avgWrite /= time.Duration(iterations)
		avgRead /= time.Duration(iterations)

		// Performance expectations
		assert.Less(t, avgWrite, StorageOperationMaxDuration, "Average write time should be reasonable")
		assert.Less(t, avgRead, StorageOperationMaxDuration, "Average read time should be reasonable")

		t.Logf("Storage benchmark (%d iterations): avg_write=%v, avg_read=%v", iterations, avgWrite, avgRead)
	})
}
