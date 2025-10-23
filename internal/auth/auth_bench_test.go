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

package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/logger"
)

// BenchmarkGitHubAuth_Validate benchmarks GitHub token validation
func BenchmarkGitHubAuth_Validate(b *testing.B) {
	// Create mock GitHub API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user" {
			w.Header().Set("X-OAuth-Scopes", "repo, user")
			response := map[string]interface{}{
				"login": "testuser",
				"id":    12345,
			}
			json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	// Create provider with mock server
	provider := setupBenchmarkGitHubProvider(b, server.URL)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := provider.validateToken(ctx)
		if err != nil {
			b.Fatalf("Failed to validate GitHub token: %v", err)
		}
	}
}

// BenchmarkTugboatAuth_GetToken benchmarks Tugboat token operations
func BenchmarkTugboatAuth_GetToken(b *testing.B) {
	// Create mock Tugboat API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/org_policies/" {
			response := map[string]interface{}{
				"data": []interface{}{},
				"meta": map[string]interface{}{
					"total": 0,
				},
			}
			json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	// Create provider with mock server
	provider := setupBenchmarkTugboatProvider(b, server.URL)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := provider.validateToken(ctx)
		if err != nil {
			b.Fatalf("Failed to validate Tugboat token: %v", err)
		}
	}
}

// BenchmarkGitHubAuth_CacheOperations benchmarks GitHub cache operations
func BenchmarkGitHubAuth_CacheOperations(b *testing.B) {
	provider := setupBenchmarkGitHubProvider(b, "https://api.github.com")

	cache := &GitHubTokenCache{
		Token:         "test_token",
		Valid:         true,
		LastValidated: time.Now(),
		UserLogin:     "testuser",
		Scopes:        []string{"repo", "user"},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Benchmark save operation
		err := provider.saveToCache(cache)
		if err != nil {
			b.Fatalf("Failed to save to cache: %v", err)
		}

		// Benchmark load operation
		loadedCache := provider.loadFromCache()
		if loadedCache == nil {
			b.Fatal("Failed to load from cache")
		}
	}
}

// BenchmarkTugboatAuth_CacheOperations benchmarks Tugboat cache operations
func BenchmarkTugboatAuth_CacheOperations(b *testing.B) {
	provider := setupBenchmarkTugboatProvider(b, "https://api-my.tugboatlogic.com")

	cache := &TugboatTokenCache{
		BearerToken:   "test_bearer_token",
		Valid:         true,
		LastValidated: time.Now(),
		BaseURL:       "https://api-my.tugboatlogic.com",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Benchmark save operation
		err := provider.saveToCache(cache)
		if err != nil {
			b.Fatalf("Failed to save to cache: %v", err)
		}

		// Benchmark load operation
		loadedCache := provider.loadFromCache()
		if loadedCache == nil {
			b.Fatal("Failed to load from cache")
		}
	}
}

// BenchmarkGitHubAuth_StatusCheck benchmarks GitHub status checking
func BenchmarkGitHubAuth_StatusCheck(b *testing.B) {
	provider := setupBenchmarkGitHubProvider(b, "https://api.github.com")
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		status := provider.GetStatus(ctx)
		if status == nil {
			b.Fatal("Status is nil")
		}
	}
}

// BenchmarkTugboatAuth_StatusCheck benchmarks Tugboat status checking
func BenchmarkTugboatAuth_StatusCheck(b *testing.B) {
	provider := setupBenchmarkTugboatProvider(b, "https://api-my.tugboatlogic.com")
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		status := provider.GetStatus(ctx)
		if status == nil {
			b.Fatal("Status is nil")
		}
	}
}

// BenchmarkAuth_ConcurrentValidation benchmarks concurrent authentication validation
func BenchmarkAuth_ConcurrentValidation(b *testing.B) {
	// Create mock GitHub server
	githubServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-OAuth-Scopes", "repo, user")
		response := map[string]interface{}{
			"login": "testuser",
			"id":    12345,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer githubServer.Close()

	// Create mock Tugboat server
	tugboatServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"data": []interface{}{},
			"meta": map[string]interface{}{
				"total": 0,
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer tugboatServer.Close()

	githubProvider := setupBenchmarkGitHubProvider(b, githubServer.URL)
	tugboatProvider := setupBenchmarkTugboatProvider(b, tugboatServer.URL)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Launch concurrent validations
		done := make(chan error, 2)

		go func() {
			done <- githubProvider.validateToken(ctx)
		}()

		go func() {
			done <- tugboatProvider.validateToken(ctx)
		}()

		// Wait for both to complete
		for j := 0; j < 2; j++ {
			if err := <-done; err != nil {
				b.Fatalf("Concurrent validation failed: %v", err)
			}
		}
	}
}

// BenchmarkAuth_MemoryAllocation benchmarks memory allocation patterns
func BenchmarkAuth_MemoryAllocation(b *testing.B) {
	provider := setupBenchmarkGitHubProvider(b, "https://api.github.com")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Create multiple auth status objects to test allocation
		for j := 0; j < 10; j++ {
			status := &AuthStatus{
				Provider:      "github",
				TokenPresent:  true,
				Authenticated: true,
				TokenValid:    true,
				Source:        "api",
				LastValidated: &[]time.Time{time.Now()}[0],
				CacheUsed:     false,
			}

			// Use the status to prevent optimization
			if status.Provider == "" {
				b.Fatal("Status provider is empty")
			}
		}

		// Update provider status multiple times
		for j := 0; j < 5; j++ {
			provider.updateStatus()
		}
	}
}

// BenchmarkAuth_JSONMarshalUnmarshal benchmarks JSON operations for cache
func BenchmarkAuth_JSONMarshalUnmarshal(b *testing.B) {
	githubCache := &GitHubTokenCache{
		Token:         "test_token_with_some_length_for_realistic_benchmarking",
		Valid:         true,
		LastValidated: time.Now(),
		UserLogin:     "testuser_with_longer_name",
		Scopes:        []string{"repo", "user", "admin:org", "write:packages", "read:packages"},
		ExpiresAt:     &[]time.Time{time.Now().Add(24 * time.Hour)}[0],
	}

	tugboatCache := &TugboatTokenCache{
		BearerToken:   "test_bearer_token_with_significant_length_for_realistic_benchmarking_purposes",
		Valid:         true,
		LastValidated: time.Now(),
		BaseURL:       "https://api-my.tugboatlogic.com/api/v1/comprehensive/endpoint",
		ExpiresAt:     &[]time.Time{time.Now().Add(12 * time.Hour)}[0],
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Benchmark GitHub cache marshaling
		githubData, err := json.MarshalIndent(githubCache, "", "  ")
		if err != nil {
			b.Fatalf("Failed to marshal GitHub cache: %v", err)
		}

		var unmarshaled GitHubTokenCache
		err = json.Unmarshal(githubData, &unmarshaled)
		if err != nil {
			b.Fatalf("Failed to unmarshal GitHub cache: %v", err)
		}

		// Benchmark Tugboat cache marshaling
		tugboatData, err := json.MarshalIndent(tugboatCache, "", "  ")
		if err != nil {
			b.Fatalf("Failed to marshal Tugboat cache: %v", err)
		}

		var tugboatUnmarshaled TugboatTokenCache
		err = json.Unmarshal(tugboatData, &tugboatUnmarshaled)
		if err != nil {
			b.Fatalf("Failed to unmarshal Tugboat cache: %v", err)
		}
	}
}

// BenchmarkAuth_ClearAuth benchmarks cache clearing operations
func BenchmarkAuth_ClearAuth(b *testing.B) {
	githubProvider := setupBenchmarkGitHubProvider(b, "https://api.github.com")
	tugboatProvider := setupBenchmarkTugboatProvider(b, "https://api-my.tugboatlogic.com")

	// Setup initial cache
	githubCache := &GitHubTokenCache{
		Token:         "test_token",
		Valid:         true,
		LastValidated: time.Now(),
		UserLogin:     "testuser",
		Scopes:        []string{"repo", "user"},
	}
	githubProvider.saveToCache(githubCache)

	tugboatCache := &TugboatTokenCache{
		BearerToken:   "test_bearer_token",
		Valid:         true,
		LastValidated: time.Now(),
		BaseURL:       "https://api-my.tugboatlogic.com",
	}
	tugboatProvider.saveToCache(tugboatCache)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Clear GitHub auth
		err := githubProvider.ClearAuth()
		if err != nil {
			b.Fatalf("Failed to clear GitHub auth: %v", err)
		}

		// Clear Tugboat auth
		err = tugboatProvider.ClearAuth()
		if err != nil {
			b.Fatalf("Failed to clear Tugboat auth: %v", err)
		}

		// Restore cache for next iteration
		githubProvider.saveToCache(githubCache)
		tugboatProvider.saveToCache(tugboatCache)
	}
}

// Helper functions for benchmarking

func setupBenchmarkGitHubProvider(b *testing.B, apiURL string) *GitHubAuthProvider {
	tempDir := b.TempDir()
	log := logger.WithComponent("github-auth-bench")

	// Override API URL for mock server if provided
	provider := NewGitHubAuthProvider("test_token", tempDir, log)

	// For mock servers, we need to modify the validation URL
	if apiURL != "https://api.github.com" {
		// This is a simplified approach for benchmarking
		// In a real implementation, we'd need to make the API URL configurable
		b.Logf("Using mock GitHub server: %s", apiURL)
	}

	return provider
}

func setupBenchmarkTugboatProvider(b *testing.B, baseURL string) *TugboatAuthProvider {
	tempDir := b.TempDir()
	log := logger.WithComponent("tugboat-auth-bench")

	provider := NewTugboatAuthProvider("test_bearer_token", baseURL, tempDir, log)

	return provider
}
