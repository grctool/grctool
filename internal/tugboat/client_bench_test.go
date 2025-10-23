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

package tugboat

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/vcr"
)

// BenchmarkTugboatClient_Sync benchmarks the complete sync operation
func BenchmarkTugboatClient_Sync(b *testing.B) {
	ctx := context.Background()
	client := setupBenchmarkClient(b, "sync_benchmark")
	defer client.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Benchmark full sync operations
		_, err := client.GetAllPolicies(ctx, "13888", "")
		if err != nil {
			b.Fatalf("Failed to get all policies: %v", err)
		}

		_, err = client.GetAllControls(ctx, "13888", "")
		if err != nil {
			b.Fatalf("Failed to get all controls: %v", err)
		}

		_, err = client.GetAllEvidenceTasks(ctx, "13888", "")
		if err != nil {
			b.Fatalf("Failed to get all evidence tasks: %v", err)
		}
	}
}

// BenchmarkTugboatClient_FetchEvidenceTasks benchmarks evidence task fetching
func BenchmarkTugboatClient_FetchEvidenceTasks(b *testing.B) {
	ctx := context.Background()
	client := setupBenchmarkClient(b, "evidence_benchmark")
	defer client.Close()

	opts := &EvidenceTaskListOptions{
		Org:      "13888",
		Page:     1,
		PageSize: 50,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := client.GetEvidenceTasks(ctx, opts)
		if err != nil {
			b.Fatalf("Failed to get evidence tasks: %v", err)
		}
	}
}

// BenchmarkTugboatClient_FetchPolicies benchmarks policy fetching
func BenchmarkTugboatClient_FetchPolicies(b *testing.B) {
	ctx := context.Background()
	client := setupBenchmarkClient(b, "policies_benchmark")
	defer client.Close()

	opts := &PolicyListOptions{
		Org:      "13888",
		Page:     1,
		PageSize: 50,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := client.GetPolicies(ctx, opts)
		if err != nil {
			b.Fatalf("Failed to get policies: %v", err)
		}
	}
}

// BenchmarkTugboatClient_FetchEvidenceTaskDetails benchmarks detailed evidence task fetching
func BenchmarkTugboatClient_FetchEvidenceTaskDetails(b *testing.B) {
	ctx := context.Background()
	client := setupBenchmarkClient(b, "evidence_details_benchmark")
	defer client.Close()

	taskID := "327992" // Known task ID from test data

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := client.GetEvidenceTaskDetails(ctx, taskID)
		if err != nil {
			b.Fatalf("Failed to get evidence task details: %v", err)
		}
	}
}

// BenchmarkTugboatClient_FetchPolicyDetails benchmarks detailed policy fetching
func BenchmarkTugboatClient_FetchPolicyDetails(b *testing.B) {
	ctx := context.Background()
	client := setupBenchmarkClient(b, "policy_details_benchmark")
	defer client.Close()

	policyID := "94641" // Known policy ID from test data

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := client.GetPolicyDetails(ctx, policyID)
		if err != nil {
			b.Fatalf("Failed to get policy details: %v", err)
		}
	}
}

// BenchmarkTugboatClient_TestConnection benchmarks connection testing
func BenchmarkTugboatClient_TestConnection(b *testing.B) {
	ctx := context.Background()
	client := setupBenchmarkClient(b, "connection_benchmark")
	defer client.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := client.TestConnection(ctx)
		if err != nil {
			b.Fatalf("Failed to test connection: %v", err)
		}
	}
}

// BenchmarkTugboatClient_PaginatedFetch benchmarks paginated data fetching
func BenchmarkTugboatClient_PaginatedFetch(b *testing.B) {
	ctx := context.Background()
	client := setupBenchmarkClient(b, "pagination_benchmark")
	defer client.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Fetch first page of each resource type
		opts := &EvidenceTaskListOptions{
			Org:      "13888",
			Page:     1,
			PageSize: 10,
		}
		_, err := client.GetEvidenceTasks(ctx, opts)
		if err != nil {
			b.Fatalf("Failed to get evidence tasks page: %v", err)
		}

		policyOpts := &PolicyListOptions{
			Org:      "13888",
			Page:     1,
			PageSize: 10,
		}
		_, err = client.GetPolicies(ctx, policyOpts)
		if err != nil {
			b.Fatalf("Failed to get policies page: %v", err)
		}

		controlOpts := &ControlListOptions{
			Org:      "13888",
			Page:     1,
			PageSize: 10,
		}
		_, err = client.GetControls(ctx, controlOpts)
		if err != nil {
			b.Fatalf("Failed to get controls page: %v", err)
		}
	}
}

// BenchmarkTugboatClient_ConcurrentRequests benchmarks concurrent API requests
func BenchmarkTugboatClient_ConcurrentRequests(b *testing.B) {
	ctx := context.Background()
	client := setupBenchmarkClient(b, "concurrent_benchmark")
	defer client.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Create channels for coordination
		done := make(chan error, 3)

		// Launch concurrent requests
		go func() {
			_, err := client.GetAllPolicies(ctx, "13888", "")
			done <- err
		}()

		go func() {
			_, err := client.GetAllControls(ctx, "13888", "")
			done <- err
		}()

		go func() {
			_, err := client.GetAllEvidenceTasks(ctx, "13888", "")
			done <- err
		}()

		// Wait for all requests to complete
		for j := 0; j < 3; j++ {
			if err := <-done; err != nil {
				b.Fatalf("Concurrent request failed: %v", err)
			}
		}
	}
}

// BenchmarkTugboatClient_TokenExtraction benchmarks bearer token extraction
func BenchmarkTugboatClient_TokenExtraction(b *testing.B) {
	client := setupBenchmarkClient(b, "token_benchmark")
	defer client.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := client.extractBearerToken()
		if err != nil {
			b.Fatalf("Failed to extract bearer token: %v", err)
		}
	}
}

// setupBenchmarkClient creates a test client optimized for benchmarking
func setupBenchmarkClient(b *testing.B, testScenario string) *Client {
	// Use the testdata directory for cassettes (relative to this test file)
	cassetteDir := filepath.Join("testdata", "vcr_cassettes")

	// Ensure the testdata directory exists
	if err := os.MkdirAll(cassetteDir, 0755); err != nil {
		b.Fatalf("Failed to create cassette directory: %v", err)
	}

	// Configure VCR for benchmarking - use playback mode to ensure consistent results
	vcrConfig := &vcr.Config{
		Enabled:         true,
		Mode:            vcr.ModePlayback, // Always use playback for consistent benchmark results
		CassetteDir:     cassetteDir,
		SanitizeHeaders: true,
		SanitizeParams:  true,
		RedactHeaders:   []string{"authorization", "cookie", "x-api-key", "token"},
		RedactParams:    []string{"api_key", "token", "password", "secret"},
		MatchMethod:     true,
		MatchURI:        true,
		MatchQuery:      false,
		MatchHeaders:    false,
		MatchBody:       false,
	}

	// Fallback tugboat config for benchmarking
	tugboatConfig := &config.TugboatConfig{
		BaseURL:      "https://api-my.tugboatlogic.com",
		CookieHeader: os.Getenv("TUGBOAT_COOKIE_HEADER"),
		Timeout:      30 * time.Second,
		RateLimit:    10,
	}

	// If no cookie header available, use a dummy one since VCR will use recorded responses
	if tugboatConfig.CookieHeader == "" {
		tugboatConfig.CookieHeader = "token=dummy_token_for_benchmark"
	}

	// Create client
	client := NewClient(tugboatConfig, vcrConfig)

	return client
}
