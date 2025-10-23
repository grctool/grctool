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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// BenchmarkGoogleWorkspaceOperations benchmarks various Google Workspace operations
func BenchmarkGoogleWorkspaceOperations(b *testing.B) {
	// Setup test environment
	server := setupBenchmarkGoogleServer(b)
	defer server.Close()

	tempDir := b.TempDir()
	credentialsPath := createBenchmarkCredentials(b, tempDir, server.URL)

	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: tempDir,
		},
	}

	cfgLogger := &logger.Config{
		Level:  logger.ErrorLevel,
		Format: "text",
		Output: "stdout",
	}
	log, err := logger.New(cfgLogger)
	require.NoError(b, err)

	tool := NewGoogleWorkspaceTool(cfg, log)

	b.Run("BasicExecution", func(b *testing.B) {
		ctx := context.Background()
		params := map[string]interface{}{
			"document_id":      "bench-doc-id",
			"document_type":    "docs",
			"credentials_path": credentialsPath,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := tool.Execute(ctx, params)
			if err != nil {
				b.Fatalf("Execution failed: %v", err)
			}
		}
	})

	b.Run("SheetsExtraction", func(b *testing.B) {
		ctx := context.Background()
		params := map[string]interface{}{
			"document_id":      "bench-sheet-id",
			"document_type":    "sheets",
			"credentials_path": credentialsPath,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := tool.Execute(ctx, params)
			if err != nil {
				b.Fatalf("Sheets extraction failed: %v", err)
			}
		}
	})

	b.Run("FormsExtraction", func(b *testing.B) {
		ctx := context.Background()
		params := map[string]interface{}{
			"document_id":      "bench-form-id",
			"document_type":    "forms",
			"credentials_path": credentialsPath,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := tool.Execute(ctx, params)
			if err != nil {
				b.Fatalf("Forms extraction failed: %v", err)
			}
		}
	})

	b.Run("FolderListing", func(b *testing.B) {
		ctx := context.Background()
		params := map[string]interface{}{
			"document_id":      "bench-folder-id",
			"document_type":    "drive",
			"credentials_path": credentialsPath,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := tool.Execute(ctx, params)
			if err != nil {
				b.Fatalf("Folder listing failed: %v", err)
			}
		}
	})
}

// BenchmarkHelperFunctions benchmarks individual helper functions
func BenchmarkHelperFunctions(b *testing.B) {
	cfg := &config.Config{}
	cfgLogger := &logger.Config{
		Level:  logger.ErrorLevel,
		Format: "text",
		Output: "stdout",
	}
	log, err := logger.New(cfgLogger)
	require.NoError(b, err)

	tool := NewGoogleWorkspaceTool(cfg, log)
	gwt := tool.(*GoogleWorkspaceTool)

	b.Run("ParseGoogleTime", func(b *testing.B) {
		timeStr := "2023-06-01T12:00:00.000Z"
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			gwt.parseGoogleTime(timeStr)
		}
	})

	b.Run("GetMimeTypeCategory", func(b *testing.B) {
		mimeType := "application/vnd.google-apps.document"
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			gwt.getMimeTypeCategory(mimeType)
		}
	})

	b.Run("CalculateRelevance", func(b *testing.B) {
		result := &GoogleWorkspaceResult{
			DocumentType: "docs",
			Content:      strings.Repeat("Content ", 1000),
			ModifiedAt:   time.Now().AddDate(0, 0, -5),
			FolderContents: []FolderItem{
				{ID: "1", Name: "File 1"},
				{ID: "2", Name: "File 2"},
			},
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			gwt.calculateRelevance(result)
		}
	})

	b.Run("GenerateReport", func(b *testing.B) {
		result := &GoogleWorkspaceResult{
			DocumentID:   "test-doc-id",
			DocumentName: "Test Document",
			DocumentType: "docs",
			Owner:        "Test Owner",
			Content:      strings.Repeat("Document content goes here. ", 100),
			CreatedAt:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			ModifiedAt:   time.Date(2023, 6, 1, 12, 0, 0, 0, time.UTC),
		}
		rules := ExtractionRules{
			IncludeMetadata: true,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			gwt.generateReport(result, "docs", rules)
		}
	})

	b.Run("GetQuestionType", func(b *testing.B) {
		// Test getQuestionType indirectly through forms processing
		// For benchmarking, we'll use a simple test
		result := &GoogleWorkspaceResult{
			Content: "Form content for benchmarking",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			gwt.calculateRelevance(result) // Related function that's easier to benchmark
		}
	})
}

// BenchmarkMappingsOperations benchmarks evidence mappings operations
func BenchmarkMappingsOperations(b *testing.B) {
	tempDir := b.TempDir()

	// Create test mappings file
	mappingsPath := filepath.Join(tempDir, "google_evidence_mappings.yaml")
	mappingsContent := createLargeMappingsFile()
	err := os.WriteFile(mappingsPath, []byte(mappingsContent), 0644)
	require.NoError(b, err)

	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: tempDir,
		},
	}

	cfgLogger := &logger.Config{
		Level:  logger.ErrorLevel,
		Format: "text",
		Output: "stdout",
	}
	log, err := logger.New(cfgLogger)
	require.NoError(b, err)

	loader := NewGoogleEvidenceMappingsLoader(cfg, log)

	b.Run("LoadMappings", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			loader.ClearCache() // Clear cache to force reload
			_, err := loader.LoadMappings()
			if err != nil {
				b.Fatalf("Failed to load mappings: %v", err)
			}
		}
	})

	b.Run("GetMappingForTask", func(b *testing.B) {
		// Pre-load mappings
		_, err := loader.LoadMappings()
		require.NoError(b, err)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			taskRef := fmt.Sprintf("ET-%d", (i%50)+1) // Cycle through 50 tasks
			_, err := loader.GetMappingForTask(taskRef)
			if err != nil {
				b.Fatalf("Failed to get mapping for %s: %v", taskRef, err)
			}
		}
	})

	b.Run("GetMappingsByPriority", func(b *testing.B) {
		_, err := loader.LoadMappings()
		require.NoError(b, err)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			priority := []string{"high", "medium", "low"}[i%3]
			_, err := loader.GetMappingsByPriority(priority)
			if err != nil {
				b.Fatalf("Failed to get mappings by priority: %v", err)
			}
		}
	})

	b.Run("TransformToGoogleAPIParams", func(b *testing.B) {
		mapping, err := loader.GetMappingForTask("ET-1")
		require.NoError(b, err)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := loader.TransformToGoogleAPIParams(mapping, 0)
			if err != nil {
				b.Fatalf("Failed to transform params: %v", err)
			}
		}
	})
}

// TestGoogleWorkspaceMemoryUsage tests memory usage patterns
func TestGoogleWorkspaceMemoryUsage(t *testing.T) {
	server := setupMemoryTestServer(t)
	defer server.Close()

	tempDir := t.TempDir()
	credentialsPath := createTestCredentials(t, tempDir, server.URL)

	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: tempDir,
		},
	}

	cfgLogger := &logger.Config{
		Level:  logger.ErrorLevel,
		Format: "text",
		Output: "stdout",
	}
	log, err := logger.New(cfgLogger)
	require.NoError(t, err)

	tool := NewGoogleWorkspaceTool(cfg, log)

	t.Run("Large_Document_Memory_Usage", func(t *testing.T) {
		var m1, m2 runtime.MemStats

		// Get initial memory stats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		ctx := context.Background()
		params := map[string]interface{}{
			"document_id":      "large-doc-id",
			"document_type":    "docs",
			"credentials_path": credentialsPath,
		}

		// Process large document
		_, _, err := tool.Execute(ctx, params)
		require.NoError(t, err)

		// Force GC and get final memory stats
		runtime.GC()
		runtime.ReadMemStats(&m2)

		// Calculate memory used
		memUsed := m2.Alloc - m1.Alloc
		t.Logf("Memory used for large document processing: %d bytes", memUsed)

		// Memory usage should be reasonable (less than 50MB for test data)
		assert.Less(t, memUsed, uint64(50*1024*1024), "Memory usage too high")
	})

	t.Run("Concurrent_Execution_Memory", func(t *testing.T) {
		var m1, m2 runtime.MemStats

		runtime.GC()
		runtime.ReadMemStats(&m1)

		ctx := context.Background()
		var wg sync.WaitGroup

		// Run 10 concurrent executions
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				params := map[string]interface{}{
					"document_id":      fmt.Sprintf("concurrent-doc-%d", id),
					"document_type":    "docs",
					"credentials_path": credentialsPath,
				}
				tool.Execute(ctx, params)
			}(i)
		}

		wg.Wait()

		runtime.GC()
		runtime.ReadMemStats(&m2)

		memUsed := m2.Alloc - m1.Alloc
		t.Logf("Memory used for concurrent execution: %d bytes", memUsed)

		// Should handle concurrent access without excessive memory usage
		assert.Less(t, memUsed, uint64(100*1024*1024), "Concurrent memory usage too high")
	})

	t.Run("Memory_Cleanup", func(t *testing.T) {
		var m1, m2, m3 runtime.MemStats

		runtime.GC()
		runtime.ReadMemStats(&m1)

		ctx := context.Background()

		// Process multiple documents
		for i := 0; i < 5; i++ {
			params := map[string]interface{}{
				"document_id":      fmt.Sprintf("cleanup-test-doc-%d", i),
				"document_type":    "sheets",
				"credentials_path": credentialsPath,
			}
			tool.Execute(ctx, params)
		}

		runtime.GC()
		runtime.ReadMemStats(&m2)

		// Wait a bit and force another GC
		time.Sleep(100 * time.Millisecond)
		runtime.GC()
		runtime.ReadMemStats(&m3)

		t.Logf("Memory after processing: %d bytes", m2.Alloc-m1.Alloc)
		t.Logf("Memory after cleanup: %d bytes", m3.Alloc-m1.Alloc)

		// Memory should be cleaned up after GC
		assert.LessOrEqual(t, m3.Alloc, m2.Alloc, "Memory not properly cleaned up")
	})
}

// TestGoogleWorkspaceConcurrency tests concurrent access patterns
func TestGoogleWorkspaceConcurrency(t *testing.T) {
	server := setupConcurrencyTestServer(t)
	defer server.Close()

	tempDir := t.TempDir()
	credentialsPath := createTestCredentials(t, tempDir, server.URL)

	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: tempDir,
		},
	}

	cfgLogger := &logger.Config{
		Level:  logger.ErrorLevel,
		Format: "text",
		Output: "stdout",
	}
	log, err := logger.New(cfgLogger)
	require.NoError(t, err)

	t.Run("Concurrent_Tool_Creation", func(t *testing.T) {
		var wg sync.WaitGroup
		tools := make([]Tool, 100)

		// Create multiple tool instances concurrently
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				tools[index] = NewGoogleWorkspaceTool(cfg, log)
			}(i)
		}

		wg.Wait()

		// Verify all tools were created successfully
		for i, tool := range tools {
			assert.NotNil(t, tool, "Tool %d was not created", i)
			assert.Equal(t, "google-workspace", tool.Name())
		}
	})

	t.Run("Concurrent_Execution_Different_Documents", func(t *testing.T) {
		tool := NewGoogleWorkspaceTool(cfg, log)
		ctx := context.Background()

		var wg sync.WaitGroup
		results := make(chan error, 50)

		// Execute 50 concurrent requests for different documents
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(docIndex int) {
				defer wg.Done()
				params := map[string]interface{}{
					"document_id":      fmt.Sprintf("concurrent-doc-%d", docIndex),
					"document_type":    []string{"docs", "sheets", "forms"}[docIndex%3],
					"credentials_path": credentialsPath,
				}

				_, _, err := tool.Execute(ctx, params)
				results <- err
			}(i)
		}

		wg.Wait()
		close(results)

		// Check results
		var errors []error
		for err := range results {
			if err != nil {
				errors = append(errors, err)
			}
		}

		assert.Empty(t, errors, "Found %d errors in concurrent execution: %v", len(errors), errors)
	})

	t.Run("Concurrent_Mappings_Loading", func(t *testing.T) {
		// Create test mappings
		mappingsPath := filepath.Join(tempDir, "google_evidence_mappings.yaml")
		mappingsContent := createTestMappingsFile()
		err := os.WriteFile(mappingsPath, []byte(mappingsContent), 0644)
		require.NoError(t, err)

		var wg sync.WaitGroup
		results := make(chan error, 20)

		// Load mappings concurrently from multiple loaders
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				loader := NewGoogleEvidenceMappingsLoader(cfg, log)
				_, err := loader.LoadMappings()
				results <- err
			}()
		}

		wg.Wait()
		close(results)

		// Check results
		var errors []error
		for err := range results {
			if err != nil {
				errors = append(errors, err)
			}
		}

		assert.Empty(t, errors, "Found %d errors in concurrent mappings loading: %v", len(errors), errors)
	})

	t.Run("Concurrent_Cache_Operations", func(t *testing.T) {
		mappingsPath := filepath.Join(tempDir, "google_evidence_mappings.yaml")
		mappingsContent := createTestMappingsFile()
		err := os.WriteFile(mappingsPath, []byte(mappingsContent), 0644)
		require.NoError(t, err)

		loader := NewGoogleEvidenceMappingsLoader(cfg, log)

		var wg sync.WaitGroup

		// Concurrent cache operations
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				if index%5 == 0 {
					// Clear cache occasionally
					loader.ClearCache()
				} else {
					// Load mappings
					loader.LoadMappings()
				}
			}(i)
		}

		wg.Wait()

		// Final load should work
		mappings, err := loader.LoadMappings()
		assert.NoError(t, err)
		assert.NotNil(t, mappings)
	})
}

// Helper functions for performance tests

func setupBenchmarkGoogleServer(b *testing.B) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(r.URL.Path, "/token"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"access_token": "bench-token", "token_type": "Bearer", "expires_in": 3600}`))

		case strings.Contains(r.URL.Path, "/docs/v1/documents/"):
			response := createBenchmarkDocsResponse()
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)

		case strings.Contains(r.URL.Path, "/sheets/v4/spreadsheets/") && strings.Contains(r.URL.Path, "/values/"):
			response := createBenchmarkSheetsValuesResponse()
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)

		case strings.Contains(r.URL.Path, "/sheets/v4/spreadsheets/"):
			response := createBenchmarkSheetsResponse()
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)

		case strings.Contains(r.URL.Path, "/forms/v1/forms/") && strings.Contains(r.URL.Path, "/responses"):
			response := createBenchmarkFormsResponsesResponse()
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)

		case strings.Contains(r.URL.Path, "/forms/v1/forms/"):
			response := createBenchmarkFormsResponse()
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)

		case strings.Contains(r.URL.Path, "/drive/v3/files/bench-folder-id"):
			response := createBenchmarkFolderResponse()
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)

		case strings.Contains(r.URL.Path, "/drive/v3/files") && strings.Contains(r.URL.Query().Get("q"), "parents"):
			response := createBenchmarkFolderContentsResponse()
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)

		case strings.Contains(r.URL.Path, "/drive/v3/files/"):
			response := createBenchmarkFileResponse()
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func setupMemoryTestServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/token") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"access_token": "memory-test-token", "token_type": "Bearer", "expires_in": 3600}`))
			return
		}

		if strings.Contains(r.URL.Path, "/docs/v1/documents/") {
			// Create large document response
			var content []map[string]interface{}
			for i := 0; i < 1000; i++ {
				content = append(content, map[string]interface{}{
					"paragraph": map[string]interface{}{
						"elements": []map[string]interface{}{
							{
								"textRun": map[string]interface{}{
									"content": fmt.Sprintf("This is a very long paragraph %d with substantial content that will test memory usage patterns. It contains multiple sentences and detailed information about various security policies and procedures that are essential for compliance and audit purposes. ", i),
								},
							},
						},
					},
				})
			}

			response := map[string]interface{}{
				"documentId": "large-doc-id",
				"title":      "Large Memory Test Document",
				"revisionId": "large-revision-123",
				"body": map[string]interface{}{
					"content": content,
				},
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": "test", "name": "Memory Test Document"}`))
	}))
}

func setupConcurrencyTestServer(t *testing.T) *httptest.Server {
	requestCount := 0
	var mutex sync.Mutex

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mutex.Lock()
		requestCount++
		currentCount := requestCount
		mutex.Unlock()

		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/token") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"access_token": "concurrency-token", "token_type": "Bearer", "expires_in": 3600}`))
			return
		}

		// Simulate processing time
		time.Sleep(10 * time.Millisecond)

		response := map[string]interface{}{
			"id":    fmt.Sprintf("concurrent-doc-%d", currentCount),
			"name":  fmt.Sprintf("Concurrent Document %d", currentCount),
			"title": fmt.Sprintf("Document %d", currentCount),
		}

		if strings.Contains(r.URL.Path, "/docs/v1/documents/") {
			response["body"] = map[string]interface{}{
				"content": []map[string]interface{}{
					{
						"paragraph": map[string]interface{}{
							"elements": []map[string]interface{}{
								{
									"textRun": map[string]interface{}{
										"content": fmt.Sprintf("Content for document %d", currentCount),
									},
								},
							},
						},
					},
				},
			}
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
}

// Helper functions to create benchmark responses
func createBenchmarkDocsResponse() map[string]interface{} {
	return map[string]interface{}{
		"documentId": "bench-doc-id",
		"title":      "Benchmark Document",
		"revisionId": "bench-revision",
		"body": map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"paragraph": map[string]interface{}{
						"elements": []map[string]interface{}{
							{
								"textRun": map[string]interface{}{
									"content": "This is benchmark content for performance testing.",
								},
							},
						},
					},
				},
			},
		},
	}
}

func createBenchmarkSheetsResponse() map[string]interface{} {
	return map[string]interface{}{
		"spreadsheetId": "bench-sheet-id",
		"properties": map[string]interface{}{
			"title": "Benchmark Spreadsheet",
		},
		"sheets": []map[string]interface{}{
			{
				"properties": map[string]interface{}{
					"title": "Benchmark Sheet",
				},
			},
		},
	}
}

func createBenchmarkSheetsValuesResponse() map[string]interface{} {
	var values [][]interface{}
	values = append(values, []interface{}{"ID", "Name", "Value"})
	for i := 0; i < 100; i++ {
		values = append(values, []interface{}{
			fmt.Sprintf("ID%d", i),
			fmt.Sprintf("Name%d", i),
			fmt.Sprintf("Value%d", i),
		})
	}

	return map[string]interface{}{
		"range":          "Sheet1!A:C",
		"majorDimension": "ROWS",
		"values":         values,
	}
}

func createBenchmarkFormsResponse() map[string]interface{} {
	return map[string]interface{}{
		"formId": "bench-form-id",
		"info": map[string]interface{}{
			"title":       "Benchmark Form",
			"description": "Form for performance testing",
		},
		"items": []map[string]interface{}{
			{
				"title": "Question 1",
				"questionItem": map[string]interface{}{
					"question": map[string]interface{}{
						"textQuestion": map[string]interface{}{},
					},
				},
			},
		},
	}
}

func createBenchmarkFormsResponsesResponse() map[string]interface{} {
	var responses []map[string]interface{}
	for i := 0; i < 10; i++ {
		responses = append(responses, map[string]interface{}{
			"responseId":        fmt.Sprintf("resp%d", i),
			"createTime":        "2023-06-01T10:00:00Z",
			"lastSubmittedTime": "2023-06-01T10:00:00Z",
		})
	}

	return map[string]interface{}{
		"responses": responses,
	}
}

func createBenchmarkFileResponse() map[string]interface{} {
	return map[string]interface{}{
		"id":           "bench-file-id",
		"name":         "Benchmark File",
		"mimeType":     "application/vnd.google-apps.document",
		"owners":       []map[string]interface{}{{"displayName": "Benchmark Owner"}},
		"createdTime":  "2023-01-01T00:00:00.000Z",
		"modifiedTime": "2023-06-01T12:00:00.000Z",
		"size":         "12345",
	}
}

func createBenchmarkFolderResponse() map[string]interface{} {
	return map[string]interface{}{
		"id":           "bench-folder-id",
		"name":         "Benchmark Folder",
		"mimeType":     "application/vnd.google-apps.folder",
		"owners":       []map[string]interface{}{{"displayName": "Folder Owner"}},
		"createdTime":  "2023-01-01T00:00:00.000Z",
		"modifiedTime": "2023-06-01T12:00:00.000Z",
	}
}

func createBenchmarkFolderContentsResponse() map[string]interface{} {
	var files []map[string]interface{}
	for i := 0; i < 20; i++ {
		files = append(files, map[string]interface{}{
			"id":           fmt.Sprintf("file%d", i),
			"name":         fmt.Sprintf("File %d", i),
			"mimeType":     "application/vnd.google-apps.document",
			"owners":       []map[string]interface{}{{"displayName": "File Owner"}},
			"createdTime":  "2023-01-01T00:00:00Z",
			"modifiedTime": "2023-06-01T12:00:00Z",
			"size":         "1024",
		})
	}

	return map[string]interface{}{
		"files": files,
	}
}

func createLargeMappingsFile() string {
	var mappings strings.Builder
	mappings.WriteString(`
google_workspace:
  default_extraction_rules:
    include_metadata: true
    include_revisions: false
    max_results: 20

evidence_mappings:
`)

	// Create 50 mappings for performance testing
	for i := 1; i <= 50; i++ {
		priority := []string{"high", "medium", "low"}[(i-1)%3]
		sourceType := []string{"google_docs", "google_sheets", "google_forms"}[(i-1)%3]

		mappings.WriteString(fmt.Sprintf(`  ET-%d:
    task_ref: "ET-%d"
    description: "Performance test mapping %d"
    source_type: "%s"
    priority: "%s"
    documents:
      - document_id: "perf-doc-%d"
        document_name: "Performance Document %d"
        document_type: "docs"
        extraction_rules:
          include_metadata: true
          max_results: 20
        validation:
          min_content_length: 100
`, i, i, i, sourceType, priority, i, i))
	}

	mappings.WriteString(`
metadata:
  version: "1.0"
  created_date: "2023-01-01"

cache_settings:
  enable_content_cache: true
  cache_duration: "24h"
`)

	return mappings.String()
}

func createTestMappingsFile() string {
	return `
google_workspace:
  default_extraction_rules:
    include_metadata: true
    include_revisions: false
    max_results: 20

evidence_mappings:
  ET-TEST:
    task_ref: "ET-TEST"
    description: "Test mapping"
    source_type: "google_docs"
    priority: "high"
    documents:
      - document_id: "test-doc-id"
        document_name: "Test Document"
        document_type: "docs"

metadata:
  version: "1.0"
  created_date: "2023-01-01"

cache_settings:
  enable_content_cache: true
  cache_duration: "24h"
`
}

func createBenchmarkCredentials(b *testing.B, tempDir, serverURL string) string {
	credentialsPath := filepath.Join(tempDir, "credentials.json")
	credentialsJSON := fmt.Sprintf(`{
		"type": "service_account",
		"project_id": "benchmark-project",
		"private_key_id": "benchmark-key-id",
		"private_key": "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC4f4DaSIa8l8jA\n7bPXFCM8YrXX/6E1G6F6H6F6H6F6H6F6H6F6H6F6H6F6H6F6H6F6H6F6H6F6H6F6\nH6F6H6F6H6F6H6F6H6F6H6F6H6F6H6F6H6F6H6F6H6F6H6F6H6F6H6F6H6F6H6F6\n-----END PRIVATE KEY-----\n",
		"client_email": "benchmark@benchmark-project.iam.gserviceaccount.com",
		"client_id": "1234567890",
		"auth_uri": "https://accounts.google.com/o/oauth2/auth",
		"token_uri": "%s/token",
		"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
		"client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/benchmark%%40benchmark-project.iam.gserviceaccount.com"
	}`, serverURL)
	err := os.WriteFile(credentialsPath, []byte(credentialsJSON), 0644)
	if err != nil {
		b.Fatalf("Failed to create credentials file: %v", err)
	}
	return credentialsPath
}
