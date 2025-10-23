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

//go:build integration
// +build integration

package terraform

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
)

// Benchmark fixtures paths
const (
	fixturesBaseDir = "../../../test/fixtures/terraform"
	smallFixtures   = "basic"  // 10-20 files
	mediumFixtures  = "medium" // 50-100 files
	largeFixtures   = "large"  // 200+ files
)

// setupBenchmarkIndexer creates an indexer for benchmark testing
func setupBenchmarkIndexer(b *testing.B, fixturesPath string) *SecurityAttributeIndexer {
	b.Helper()

	log, _ := logger.NewTestLogger()

	// Resolve absolute path to fixtures
	absFixturePath, err := filepath.Abs(filepath.Join(fixturesBaseDir, fixturesPath))
	if err != nil {
		b.Fatalf("Failed to resolve fixture path: %v", err)
	}

	// Verify fixtures directory exists
	if _, err := os.Stat(absFixturePath); os.IsNotExist(err) {
		b.Skipf("Fixtures directory does not exist: %s", absFixturePath)
	}

	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				Terraform: config.TerraformToolConfig{
					Enabled:         true,
					ScanPaths:       []string{absFixturePath},
					IncludePatterns: []string{"**/*.tf"},
					ExcludePatterns: []string{"**/.terraform/**"},
				},
			},
		},
		Storage: config.StorageConfig{
			CacheDir: b.TempDir(),
		},
	}

	return NewSecurityAttributeIndexer(cfg, log)
}

// ============================================================================
// Index Build Benchmarks
// ============================================================================

// BenchmarkIndexBuild_Small benchmarks index building with small repositories (10-20 files)
func BenchmarkIndexBuild_Small(b *testing.B) {
	indexer := setupBenchmarkIndexer(b, smallFixtures)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := indexer.BuildAndPersistIndex(ctx)
		if err != nil {
			b.Fatalf("Failed to build index: %v", err)
		}
	}
}

// BenchmarkIndexBuild_Medium benchmarks index building with medium repositories (50-100 files)
func BenchmarkIndexBuild_Medium(b *testing.B) {
	indexer := setupBenchmarkIndexer(b, mediumFixtures)
	ctx := context.Background()

	// Check if medium fixtures exist, skip if not
	mediumPath := filepath.Join(fixturesBaseDir, mediumFixtures)
	if _, err := os.Stat(mediumPath); os.IsNotExist(err) {
		b.Skip("Medium fixtures not available")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := indexer.BuildAndPersistIndex(ctx)
		if err != nil {
			b.Fatalf("Failed to build index: %v", err)
		}
	}
}

// BenchmarkIndexBuild_Large benchmarks index building with large repositories (200+ files)
func BenchmarkIndexBuild_Large(b *testing.B) {
	indexer := setupBenchmarkIndexer(b, largeFixtures)
	ctx := context.Background()

	// Check if large fixtures exist, skip if not
	largePath := filepath.Join(fixturesBaseDir, largeFixtures)
	if _, err := os.Stat(largePath); os.IsNotExist(err) {
		b.Skip("Large fixtures not available")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := indexer.BuildAndPersistIndex(ctx)
		if err != nil {
			b.Fatalf("Failed to build index: %v", err)
		}
	}
}

// ============================================================================
// Index Persistence Benchmarks
// ============================================================================

// BenchmarkIndexSerialization benchmarks index serialization to disk
func BenchmarkIndexSerialization(b *testing.B) {
	indexer := setupBenchmarkIndexer(b, smallFixtures)
	ctx := context.Background()

	// Build index once
	persistedIndex, err := indexer.BuildAndPersistIndex(ctx)
	if err != nil {
		b.Fatalf("Failed to build initial index: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := indexer.indexStorage.SaveIndex(persistedIndex)
		if err != nil {
			b.Fatalf("Failed to save index: %v", err)
		}
	}
}

// BenchmarkIndexDeserialization benchmarks index loading from disk
func BenchmarkIndexDeserialization(b *testing.B) {
	indexer := setupBenchmarkIndexer(b, smallFixtures)
	ctx := context.Background()

	// Build and persist index once
	_, err := indexer.BuildAndPersistIndex(ctx)
	if err != nil {
		b.Fatalf("Failed to build initial index: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := indexer.indexStorage.LoadIndex()
		if err != nil {
			b.Fatalf("Failed to load index: %v", err)
		}
	}
}

// BenchmarkIndexLoadOrBuild_CacheHit benchmarks cache hit performance
func BenchmarkIndexLoadOrBuild_CacheHit(b *testing.B) {
	indexer := setupBenchmarkIndexer(b, smallFixtures)
	ctx := context.Background()

	// Build index once to populate cache
	_, err := indexer.BuildAndPersistIndex(ctx)
	if err != nil {
		b.Fatalf("Failed to build initial index: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := indexer.LoadOrBuildIndex(ctx, false)
		if err != nil {
			b.Fatalf("Failed to load or build index: %v", err)
		}
	}
}

// BenchmarkIndexLoadOrBuild_CacheMiss benchmarks cache miss performance (force rebuild)
func BenchmarkIndexLoadOrBuild_CacheMiss(b *testing.B) {
	indexer := setupBenchmarkIndexer(b, smallFixtures)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := indexer.LoadOrBuildIndex(ctx, true)
		if err != nil {
			b.Fatalf("Failed to load or build index: %v", err)
		}
	}
}

// ============================================================================
// Query Benchmarks
// ============================================================================

// BenchmarkQueryByControl benchmarks querying by SOC2 control codes
func BenchmarkQueryByControl(b *testing.B) {
	indexer := setupBenchmarkIndexer(b, smallFixtures)
	ctx := context.Background()

	// Build index once
	persistedIndex, err := indexer.BuildAndPersistIndex(ctx)
	if err != nil {
		b.Fatalf("Failed to build index: %v", err)
	}

	query := SecurityIndexQuery{
		ControlCodes:    []string{"CC6.1", "CC6.6", "CC6.8"},
		IncludeMetadata: false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = indexer.filterIndex(persistedIndex.Index, "by_control", query)
	}
}

// BenchmarkQueryByAttribute benchmarks querying by security attributes
func BenchmarkQueryByAttribute(b *testing.B) {
	indexer := setupBenchmarkIndexer(b, smallFixtures)
	ctx := context.Background()

	// Build index once
	persistedIndex, err := indexer.BuildAndPersistIndex(ctx)
	if err != nil {
		b.Fatalf("Failed to build index: %v", err)
	}

	query := SecurityIndexQuery{
		SecurityAttributes: []string{"encryption", "access_control", "monitoring"},
		IncludeMetadata:    false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = indexer.filterIndex(persistedIndex.Index, "by_attribute", query)
	}
}

// BenchmarkQueryByEnvironment benchmarks querying by environment
func BenchmarkQueryByEnvironment(b *testing.B) {
	indexer := setupBenchmarkIndexer(b, smallFixtures)
	ctx := context.Background()

	// Build index once
	persistedIndex, err := indexer.BuildAndPersistIndex(ctx)
	if err != nil {
		b.Fatalf("Failed to build index: %v", err)
	}

	query := SecurityIndexQuery{
		Environments:    []string{"prod", "production"},
		IncludeMetadata: false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = indexer.filterIndex(persistedIndex.Index, "by_environment", query)
	}
}

// BenchmarkQueryByRiskLevel benchmarks querying by risk level
func BenchmarkQueryByRiskLevel(b *testing.B) {
	indexer := setupBenchmarkIndexer(b, smallFixtures)
	ctx := context.Background()

	// Build index once
	persistedIndex, err := indexer.BuildAndPersistIndex(ctx)
	if err != nil {
		b.Fatalf("Failed to build index: %v", err)
	}

	query := SecurityIndexQuery{
		RiskLevels:      []string{"high", "medium"},
		IncludeMetadata: false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = indexer.filterIndex(persistedIndex.Index, "by_risk_level", query)
	}
}

// BenchmarkQueryByResourceType benchmarks querying by resource type
func BenchmarkQueryByResourceType(b *testing.B) {
	indexer := setupBenchmarkIndexer(b, smallFixtures)
	ctx := context.Background()

	// Build index once
	persistedIndex, err := indexer.BuildAndPersistIndex(ctx)
	if err != nil {
		b.Fatalf("Failed to build index: %v", err)
	}

	query := SecurityIndexQuery{
		ResourceTypes:   []string{"aws_s3_bucket", "aws_kms_key", "aws_iam_role"},
		IncludeMetadata: false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = indexer.filterIndex(persistedIndex.Index, "by_resource_type", query)
	}
}

// BenchmarkQueryComplex benchmarks complex query with multiple filters
func BenchmarkQueryComplex(b *testing.B) {
	indexer := setupBenchmarkIndexer(b, smallFixtures)
	ctx := context.Background()

	// Build index once
	persistedIndex, err := indexer.BuildAndPersistIndex(ctx)
	if err != nil {
		b.Fatalf("Failed to build index: %v", err)
	}

	query := SecurityIndexQuery{
		ControlCodes:       []string{"CC6.1", "CC6.6", "CC6.8"},
		SecurityAttributes: []string{"encryption", "access_control"},
		Environments:       []string{"prod", "production"},
		RiskLevels:         []string{"high"},
		IncludeMetadata:    true,
	}

	b.ResetTimer()
	var result []IndexedResource
	for i := 0; i < b.N; i++ {
		filtered := indexer.filterIndex(persistedIndex.Index, "by_control", query)

		// Apply additional filters to simulate complex query
		var finalResults []IndexedResource
		for _, resource := range filtered.IndexedResources {
			// Check if resource matches all criteria
			matchesEnv := false
			for _, env := range query.Environments {
				if resource.Environment == env {
					matchesEnv = true
					break
				}
			}

			matchesRisk := false
			for _, risk := range query.RiskLevels {
				if resource.RiskLevel == risk {
					matchesRisk = true
					break
				}
			}

			if matchesEnv && matchesRisk {
				finalResults = append(finalResults, resource)
			}
		}
		result = finalResults
	}
	// Prevent compiler optimization
	_ = result
}

// ============================================================================
// IndexQuery Benchmarks (using the IndexQuery interface)
// ============================================================================

// BenchmarkIndexQuery_ByControl benchmarks IndexQuery.ByControl method
func BenchmarkIndexQuery_ByControl(b *testing.B) {
	indexer := setupBenchmarkIndexer(b, smallFixtures)
	ctx := context.Background()

	// Build index once
	persistedIndex, err := indexer.BuildAndPersistIndex(ctx)
	if err != nil {
		b.Fatalf("Failed to build index: %v", err)
	}

	iq := NewIndexQuery(persistedIndex)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = iq.ByControl("CC6.1", "CC6.6", "CC6.8")
	}
}

// BenchmarkIndexQuery_ByAttribute benchmarks IndexQuery.ByAttribute method
func BenchmarkIndexQuery_ByAttribute(b *testing.B) {
	indexer := setupBenchmarkIndexer(b, smallFixtures)
	ctx := context.Background()

	// Build index once
	persistedIndex, err := indexer.BuildAndPersistIndex(ctx)
	if err != nil {
		b.Fatalf("Failed to build index: %v", err)
	}

	iq := NewIndexQuery(persistedIndex)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = iq.ByAttribute("encryption", "access_control", "monitoring")
	}
}

// BenchmarkIndexQuery_ByEvidenceTask benchmarks IndexQuery.ByEvidenceTask method
func BenchmarkIndexQuery_ByEvidenceTask(b *testing.B) {
	indexer := setupBenchmarkIndexer(b, smallFixtures)
	ctx := context.Background()

	// Build index once
	persistedIndex, err := indexer.BuildAndPersistIndex(ctx)
	if err != nil {
		b.Fatalf("Failed to build index: %v", err)
	}

	iq := NewIndexQuery(persistedIndex)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = iq.ByEvidenceTask("ET-001")
	}
}

// ============================================================================
// Report Generation Benchmarks
// ============================================================================

// BenchmarkReportGeneration_DetailedJSON benchmarks detailed JSON report generation
func BenchmarkReportGeneration_DetailedJSON(b *testing.B) {
	indexer := setupBenchmarkIndexer(b, smallFixtures)
	ctx := context.Background()

	// Build index once
	persistedIndex, err := indexer.BuildAndPersistIndex(ctx)
	if err != nil {
		b.Fatalf("Failed to build index: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := indexer.generateDetailedJSONReport(persistedIndex.Index)
		if err != nil {
			b.Fatalf("Failed to generate report: %v", err)
		}
	}
}

// BenchmarkReportGeneration_SummaryTable benchmarks summary table report generation
func BenchmarkReportGeneration_SummaryTable(b *testing.B) {
	indexer := setupBenchmarkIndexer(b, smallFixtures)
	ctx := context.Background()

	// Build index once
	persistedIndex, err := indexer.BuildAndPersistIndex(ctx)
	if err != nil {
		b.Fatalf("Failed to build index: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := indexer.generateSummaryTableReport(persistedIndex.Index)
		if err != nil {
			b.Fatalf("Failed to generate report: %v", err)
		}
	}
}

// BenchmarkReportGeneration_SecurityMatrix benchmarks security matrix report generation
func BenchmarkReportGeneration_SecurityMatrix(b *testing.B) {
	indexer := setupBenchmarkIndexer(b, smallFixtures)
	ctx := context.Background()

	// Build index once
	persistedIndex, err := indexer.BuildAndPersistIndex(ctx)
	if err != nil {
		b.Fatalf("Failed to build index: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := indexer.generateSecurityMatrixReport(persistedIndex.Index)
		if err != nil {
			b.Fatalf("Failed to generate report: %v", err)
		}
	}
}

// ============================================================================
// Aggregate Benchmarks (End-to-End)
// ============================================================================

// BenchmarkEndToEnd_ColdStart benchmarks complete cold start (no cache)
func BenchmarkEndToEnd_ColdStart(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		indexer := setupBenchmarkIndexer(b, smallFixtures)
		b.StartTimer()

		params := map[string]interface{}{
			"query_type":       "by_control",
			"control_codes":    []interface{}{"CC6.1", "CC6.6"},
			"output_format":    "detailed_json",
			"include_metadata": true,
			"skip_cache":       true,
		}

		_, _, err := indexer.Execute(ctx, params)
		if err != nil {
			b.Fatalf("Failed to execute: %v", err)
		}
	}
}

// BenchmarkEndToEnd_WarmCache benchmarks complete execution with warm cache
func BenchmarkEndToEnd_WarmCache(b *testing.B) {
	indexer := setupBenchmarkIndexer(b, smallFixtures)
	ctx := context.Background()

	// Warm up cache
	_, err := indexer.BuildAndPersistIndex(ctx)
	if err != nil {
		b.Fatalf("Failed to build initial index: %v", err)
	}

	params := map[string]interface{}{
		"query_type":       "by_control",
		"control_codes":    []interface{}{"CC6.1", "CC6.6"},
		"output_format":    "detailed_json",
		"include_metadata": true,
		"skip_cache":       false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := indexer.Execute(ctx, params)
		if err != nil {
			b.Fatalf("Failed to execute: %v", err)
		}
	}
}

// ============================================================================
// Memory Benchmarks
// ============================================================================

// BenchmarkMemory_IndexBuild benchmarks memory usage during index build
func BenchmarkMemory_IndexBuild(b *testing.B) {
	indexer := setupBenchmarkIndexer(b, smallFixtures)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := indexer.BuildAndPersistIndex(ctx)
		if err != nil {
			b.Fatalf("Failed to build index: %v", err)
		}
	}
}

// BenchmarkMemory_QueryExecution benchmarks memory usage during query execution
func BenchmarkMemory_QueryExecution(b *testing.B) {
	indexer := setupBenchmarkIndexer(b, smallFixtures)
	ctx := context.Background()

	// Build index once
	persistedIndex, err := indexer.BuildAndPersistIndex(ctx)
	if err != nil {
		b.Fatalf("Failed to build index: %v", err)
	}

	query := SecurityIndexQuery{
		ControlCodes:       []string{"CC6.1", "CC6.6", "CC6.8"},
		SecurityAttributes: []string{"encryption"},
		IncludeMetadata:    true,
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = indexer.filterIndex(persistedIndex.Index, "by_control", query)
	}
}
