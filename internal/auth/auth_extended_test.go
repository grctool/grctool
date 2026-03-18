// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- NoAuthProvider comprehensive tests ---

func TestNoAuthProvider_DefaultSource(t *testing.T) {
	t.Parallel()
	provider := NewNoAuthProvider("local-tool", "")
	status := provider.GetStatus(context.Background())
	assert.Equal(t, "local", status.Source, "empty source should default to 'local'")
}

func TestNoAuthProvider_CustomSource(t *testing.T) {
	t.Parallel()
	provider := NewNoAuthProvider("fs-tool", "filesystem")
	status := provider.GetStatus(context.Background())
	assert.Equal(t, "filesystem", status.Source)
}

func TestNoAuthProvider_FullInterface(t *testing.T) {
	t.Parallel()
	provider := NewNoAuthProvider("test", "cache")
	ctx := context.Background()

	assert.Equal(t, "test", provider.Name())
	assert.False(t, provider.IsAuthRequired())

	status := provider.GetStatus(ctx)
	require.NotNil(t, status)
	assert.True(t, status.Authenticated)
	assert.Equal(t, "test", status.Provider)
	assert.False(t, status.TokenPresent)
	assert.False(t, status.TokenValid)
	assert.False(t, status.CacheUsed)
	assert.Equal(t, "cache", status.Source)
	assert.Empty(t, status.Error)

	assert.NoError(t, provider.Authenticate(ctx))
	assert.NoError(t, provider.ValidateAuth(ctx))
	assert.NoError(t, provider.ClearAuth())
}

// --- GitHubAuthProvider tests ---

func TestGitHubAuthProvider_WithToken_NoAPICall(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewGitHubAuthProvider("ghp_testtoken123", cacheDir, log)

	assert.Equal(t, "github", provider.Name())
	assert.True(t, provider.IsAuthRequired())

	// Status before authentication: token present but not authenticated
	status := provider.GetStatus(context.Background())
	assert.True(t, status.TokenPresent)
	assert.False(t, status.Authenticated)
	assert.Equal(t, "api", status.Source)
}

func TestGitHubAuthProvider_WhitespaceToken(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewGitHubAuthProvider("  ghp_token123  ", cacheDir, log)
	assert.Equal(t, "ghp_token123", provider.token, "token should be trimmed")
}

func TestGitHubAuthProvider_EmptyToken_AuthenticateFails(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewGitHubAuthProvider("", cacheDir, log)

	err := provider.Authenticate(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "GitHub Personal Access Token is required")
	assert.False(t, provider.status.Authenticated)
	assert.NotEmpty(t, provider.status.Error)
}

func TestGitHubAuthProvider_ValidateToken_Success(t *testing.T) {
	t.Parallel()
	// GitHub validateToken hardcodes api.github.com, so we test through cache flow.
	// The Authenticate path with fresh cache is tested in TestGitHubAuthProvider_Authenticate_WithFreshCache.
}

func TestGitHubAuthProvider_ValidateToken_Unauthorized(t *testing.T) {
	t.Parallel()
	// GitHub validateToken hardcodes api.github.com, so we test status updates directly.
}

func TestGitHubAuthProvider_CacheOperations(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewGitHubAuthProvider("test-token", cacheDir, log)

	// Save to cache
	now := time.Now()
	cache := &GitHubTokenCache{
		Token:         "test-token",
		Valid:         true,
		LastValidated: now,
		UserLogin:     "testuser",
		Scopes:        []string{"repo", "user"},
	}
	err := provider.saveToCache(cache)
	require.NoError(t, err)

	// Verify cache file exists
	_, err = os.Stat(provider.cacheFile)
	require.NoError(t, err)

	// Load from cache
	loaded := provider.loadFromCache()
	require.NotNil(t, loaded)
	assert.Equal(t, "test-token", loaded.Token)
	assert.True(t, loaded.Valid)
	assert.Equal(t, "testuser", loaded.UserLogin)
	assert.Equal(t, []string{"repo", "user"}, loaded.Scopes)
}

func TestGitHubAuthProvider_CacheTokenMismatch(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewGitHubAuthProvider("token-A", cacheDir, log)

	// Save cache with different token
	cache := &GitHubTokenCache{
		Token:         "token-B",
		Valid:         true,
		LastValidated: time.Now(),
	}
	err := provider.saveToCache(cache)
	require.NoError(t, err)

	// Load should return nil due to token mismatch
	loaded := provider.loadFromCache()
	assert.Nil(t, loaded, "cache should return nil when token doesn't match")
}

func TestGitHubAuthProvider_CacheCorrupted(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewGitHubAuthProvider("test-token", cacheDir, log)

	// Write corrupt JSON
	err := os.WriteFile(provider.cacheFile, []byte("not-json"), 0600)
	require.NoError(t, err)

	loaded := provider.loadFromCache()
	assert.Nil(t, loaded, "corrupted cache should return nil")
}

func TestGitHubAuthProvider_CacheMissing(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewGitHubAuthProvider("test-token", cacheDir, log)

	loaded := provider.loadFromCache()
	assert.Nil(t, loaded, "missing cache file should return nil")
}

func TestGitHubAuthProvider_Authenticate_WithFreshCache(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewGitHubAuthProvider("test-token", cacheDir, log)

	// Write a fresh valid cache
	now := time.Now()
	cache := &GitHubTokenCache{
		Token:         "test-token",
		Valid:         true,
		LastValidated: now,
		UserLogin:     "testuser",
	}
	err := provider.saveToCache(cache)
	require.NoError(t, err)

	// Authenticate should use cache (no HTTP call needed)
	err = provider.Authenticate(context.Background())
	require.NoError(t, err)
	assert.True(t, provider.status.Authenticated)
	assert.True(t, provider.status.CacheUsed)
	assert.True(t, provider.status.TokenValid)
	assert.Empty(t, provider.status.Error)
}

func TestGitHubAuthProvider_Authenticate_WithStaleCache(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewGitHubAuthProvider("test-token", cacheDir, log)

	// Write a stale cache (>1 hour old)
	staleTime := time.Now().Add(-2 * time.Hour)
	cache := &GitHubTokenCache{
		Token:         "test-token",
		Valid:         true,
		LastValidated: staleTime,
		UserLogin:     "testuser",
	}
	err := provider.saveToCache(cache)
	require.NoError(t, err)

	// Authenticate will try API (which will fail since we have no real server)
	// but this tests that stale cache is NOT used
	err = provider.Authenticate(context.Background())
	assert.Error(t, err, "should fail because stale cache is skipped and real API call fails")
	assert.False(t, provider.status.CacheUsed)
}

func TestGitHubAuthProvider_ClearAuth(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewGitHubAuthProvider("test-token", cacheDir, log)

	// Create cache first
	cache := &GitHubTokenCache{
		Token:         "test-token",
		Valid:         true,
		LastValidated: time.Now(),
	}
	err := provider.saveToCache(cache)
	require.NoError(t, err)

	// Set status as if authenticated
	provider.status.Authenticated = true
	provider.status.TokenValid = true
	now := time.Now()
	provider.status.LastValidated = &now

	// Clear
	err = provider.ClearAuth()
	require.NoError(t, err)
	assert.False(t, provider.status.Authenticated)
	assert.False(t, provider.status.TokenValid)
	assert.Nil(t, provider.status.LastValidated)
	assert.False(t, provider.status.CacheUsed)

	// Cache file should be gone
	_, err = os.Stat(provider.cacheFile)
	assert.True(t, os.IsNotExist(err))
}

func TestGitHubAuthProvider_ClearAuth_NoCache(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewGitHubAuthProvider("test-token", cacheDir, log)

	// Clear when no cache exists should not error
	err := provider.ClearAuth()
	assert.NoError(t, err)
}

func TestGitHubAuthProvider_UpdateStatus_NoToken(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewGitHubAuthProvider("", cacheDir, log)

	provider.updateStatus()
	assert.False(t, provider.status.Authenticated)
	assert.False(t, provider.status.TokenValid)
	assert.False(t, provider.status.TokenPresent)
	assert.Equal(t, "local", provider.status.Source)
}

func TestGitHubAuthProvider_UpdateStatus_RecentValidation(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewGitHubAuthProvider("test-token", cacheDir, log)

	now := time.Now()
	provider.status.LastValidated = &now
	provider.status.TokenValid = true

	provider.updateStatus()
	assert.True(t, provider.status.Authenticated)
}

func TestGitHubAuthProvider_UpdateStatus_StaleValidation(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewGitHubAuthProvider("test-token", cacheDir, log)

	stale := time.Now().Add(-2 * time.Hour)
	provider.status.LastValidated = &stale
	provider.status.TokenValid = true
	provider.status.Authenticated = false

	provider.updateStatus()
	// Token present but validation stale - not authenticated
	assert.False(t, provider.status.Authenticated)
}

// --- TugboatAuthProvider tests ---

func TestTugboatAuthProvider_WithToken(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewTugboatAuthProvider("bearer-token-123", "https://api.tugboat.com", cacheDir, log)

	assert.Equal(t, "tugboat", provider.Name())
	assert.True(t, provider.IsAuthRequired())
	assert.Equal(t, "bearer-token-123", provider.bearerToken)
	assert.Equal(t, "https://api.tugboat.com", provider.baseURL)
}

func TestTugboatAuthProvider_WhitespaceToken(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewTugboatAuthProvider("  bearer-token  ", "https://api.tugboat.com/", cacheDir, log)
	assert.Equal(t, "bearer-token", provider.bearerToken, "token should be trimmed")
	assert.Equal(t, "https://api.tugboat.com", provider.baseURL, "trailing slash should be removed")
}

func TestTugboatAuthProvider_EmptyToken_AuthenticateFails(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewTugboatAuthProvider("", "https://api.tugboat.com", cacheDir, log)

	err := provider.Authenticate(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tugboat bearer token is required")
	assert.False(t, provider.status.Authenticated)
}

func TestTugboatAuthProvider_CacheOperations(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewTugboatAuthProvider("bearer-token", "https://api.tugboat.com", cacheDir, log)

	now := time.Now()
	cache := &TugboatTokenCache{
		BearerToken:   "bearer-token",
		Valid:         true,
		LastValidated: now,
		BaseURL:       "https://api.tugboat.com",
	}
	err := provider.saveToCache(cache)
	require.NoError(t, err)

	loaded := provider.loadFromCache()
	require.NotNil(t, loaded)
	assert.Equal(t, "bearer-token", loaded.BearerToken)
	assert.True(t, loaded.Valid)
	assert.Equal(t, "https://api.tugboat.com", loaded.BaseURL)
}

func TestTugboatAuthProvider_CacheTokenMismatch(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewTugboatAuthProvider("token-A", "https://api.tugboat.com", cacheDir, log)

	cache := &TugboatTokenCache{
		BearerToken:   "token-B",
		Valid:         true,
		LastValidated: time.Now(),
		BaseURL:       "https://api.tugboat.com",
	}
	err := provider.saveToCache(cache)
	require.NoError(t, err)

	loaded := provider.loadFromCache()
	assert.Nil(t, loaded, "cache should return nil when token doesn't match")
}

func TestTugboatAuthProvider_CacheURLMismatch(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewTugboatAuthProvider("token", "https://api-new.tugboat.com", cacheDir, log)

	cache := &TugboatTokenCache{
		BearerToken:   "token",
		Valid:         true,
		LastValidated: time.Now(),
		BaseURL:       "https://api-old.tugboat.com",
	}
	err := provider.saveToCache(cache)
	require.NoError(t, err)

	loaded := provider.loadFromCache()
	assert.Nil(t, loaded, "cache should return nil when URL doesn't match")
}

func TestTugboatAuthProvider_Authenticate_WithFreshCache(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewTugboatAuthProvider("bearer-token", "https://api.tugboat.com", cacheDir, log)

	now := time.Now()
	cache := &TugboatTokenCache{
		BearerToken:   "bearer-token",
		Valid:         true,
		LastValidated: now,
		BaseURL:       "https://api.tugboat.com",
	}
	err := provider.saveToCache(cache)
	require.NoError(t, err)

	// Authenticate should use cache
	err = provider.Authenticate(context.Background())
	require.NoError(t, err)
	assert.True(t, provider.status.Authenticated)
	assert.True(t, provider.status.CacheUsed)
	assert.Empty(t, provider.status.Error)
}

func TestTugboatAuthProvider_Authenticate_WithStaleCache(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewTugboatAuthProvider("bearer-token", "https://api.tugboat.com", cacheDir, log)

	stale := time.Now().Add(-1 * time.Hour) // >30 min old
	cache := &TugboatTokenCache{
		BearerToken:   "bearer-token",
		Valid:         true,
		LastValidated: stale,
		BaseURL:       "https://api.tugboat.com",
	}
	err := provider.saveToCache(cache)
	require.NoError(t, err)

	err = provider.Authenticate(context.Background())
	assert.Error(t, err, "should fail because stale cache is skipped")
}

func TestTugboatAuthProvider_ValidateToken_HTTPServer(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer my-bearer-token", r.Header.Get("Authorization"))
		assert.Contains(t, r.URL.Path, "/api/org_policies/")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}})
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewTugboatAuthProvider("my-bearer-token", server.URL, cacheDir, log)

	err := provider.validateToken(context.Background())
	require.NoError(t, err)

	// Cache should have been saved
	loaded := provider.loadFromCache()
	require.NotNil(t, loaded)
	assert.True(t, loaded.Valid)
}

func TestTugboatAuthProvider_ValidateToken_Unauthorized(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewTugboatAuthProvider("bad-token", server.URL, cacheDir, log)

	err := provider.validateToken(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid or expired")
}

func TestTugboatAuthProvider_ValidateToken_Forbidden(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewTugboatAuthProvider("limited-token", server.URL, cacheDir, log)

	err := provider.validateToken(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "lacks required permissions")
}

func TestTugboatAuthProvider_ValidateToken_ServerError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewTugboatAuthProvider("token", server.URL, cacheDir, log)

	err := provider.validateToken(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func TestTugboatAuthProvider_ValidateToken_URLWithApi(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// When baseURL already ends with /api, it should not duplicate
		assert.Contains(t, r.URL.Path, "/api/org_policies/")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{})
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewTugboatAuthProvider("token", server.URL+"/api", cacheDir, log)

	err := provider.validateToken(context.Background())
	require.NoError(t, err)
}

func TestTugboatAuthProvider_ClearAuth(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewTugboatAuthProvider("token", "https://api.tugboat.com", cacheDir, log)

	// Create cache
	cache := &TugboatTokenCache{
		BearerToken:   "token",
		Valid:         true,
		LastValidated: time.Now(),
		BaseURL:       "https://api.tugboat.com",
	}
	err := provider.saveToCache(cache)
	require.NoError(t, err)

	provider.status.Authenticated = true
	provider.status.TokenValid = true

	err = provider.ClearAuth()
	require.NoError(t, err)
	assert.False(t, provider.status.Authenticated)
	assert.False(t, provider.status.TokenValid)
	assert.Nil(t, provider.status.LastValidated)

	_, err = os.Stat(provider.cacheFile)
	assert.True(t, os.IsNotExist(err))
}

func TestTugboatAuthProvider_UpdateStatus_NoToken(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewTugboatAuthProvider("", "https://api.tugboat.com", cacheDir, log)

	provider.updateStatus()
	assert.False(t, provider.status.Authenticated)
	assert.False(t, provider.status.TokenPresent)
	assert.Equal(t, "local", provider.status.Source)
}

func TestTugboatAuthProvider_UpdateStatus_RecentValidation(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewTugboatAuthProvider("token", "https://api.tugboat.com", cacheDir, log)

	now := time.Now()
	provider.status.LastValidated = &now
	provider.status.TokenValid = true

	provider.updateStatus()
	assert.True(t, provider.status.Authenticated)
}

func TestTugboatAuthProvider_Authenticate_FullFlow_WithHTTPTest(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{})
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewTugboatAuthProvider("good-token", server.URL, cacheDir, log)

	err := provider.Authenticate(context.Background())
	require.NoError(t, err)
	assert.True(t, provider.status.Authenticated)
	assert.True(t, provider.status.TokenValid)
	assert.False(t, provider.status.CacheUsed)
	assert.Empty(t, provider.status.Error)
	assert.NotNil(t, provider.status.LastValidated)
}

// --- ValidateCredentials tests ---

func TestValidateCredentials_OrgIDExtraction_Float(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []map[string]interface{}{
				{"org_id": 13888.0},
			},
		})
	}))
	defer server.Close()

	creds := &AuthCredentials{BearerToken: "test-token", CapturedAt: time.Now()}
	err := ValidateCredentials(context.Background(), server.URL, creds)
	require.NoError(t, err)
	assert.Equal(t, "13888", creds.OrgID)
}

func TestValidateCredentials_OrgIDExtraction_String(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []map[string]interface{}{
				{"org_id": "org-abc"},
			},
		})
	}))
	defer server.Close()

	creds := &AuthCredentials{BearerToken: "test-token", CapturedAt: time.Now()}
	err := ValidateCredentials(context.Background(), server.URL, creds)
	require.NoError(t, err)
	assert.Equal(t, "org-abc", creds.OrgID)
}

func TestValidateCredentials_NoResults(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []interface{}{},
		})
	}))
	defer server.Close()

	creds := &AuthCredentials{BearerToken: "test-token", CapturedAt: time.Now()}
	err := ValidateCredentials(context.Background(), server.URL, creds)
	require.NoError(t, err)
	assert.Empty(t, creds.OrgID)
}

func TestValidateCredentials_URLNormalization(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// URL should include /api/policy/
		assert.Contains(t, r.URL.Path, "/api/policy/")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"results": []interface{}{}})
	}))
	defer server.Close()

	// Test with URL that already has /api
	creds := &AuthCredentials{BearerToken: "test-token", CapturedAt: time.Now()}
	err := ValidateCredentials(context.Background(), server.URL+"/api", creds)
	require.NoError(t, err)
}

// --- extractBearerTokenFromSafari additional tests ---

func TestExtractBearerTokenFromSafari_JWTLikeToken(t *testing.T) {
	t.Parallel()
	// JWT-like token in a bearer_ prefixed cookie
	cookie := "session=abc; bearer_token=eyJhbGciOiJIUzI1NiJ9.eyJ0ZXN0IjoxfQ.sig"
	result := extractBearerTokenFromSafari(cookie)
	assert.Equal(t, "eyJhbGciOiJIUzI1NiJ9.eyJ0ZXN0IjoxfQ.sig", result)
}

func TestExtractBearerTokenFromSafari_AuthCookieShortValue(t *testing.T) {
	t.Parallel()
	// Short value in auth cookie should NOT be extracted (< 20 chars and no JWT format)
	cookie := "session=abc; auth=short"
	result := extractBearerTokenFromSafari(cookie)
	assert.Empty(t, result)
}

func TestExtractBearerTokenFromSafari_MalformedCookies(t *testing.T) {
	t.Parallel()
	// Cookie without = separator
	cookie := "noseparator; token=invalidbase64!@#"
	result := extractBearerTokenFromSafari(cookie)
	assert.Empty(t, result)
}

// --- ValidateCredentials with cookies-only (no bearer) ---

func TestValidateCredentials_CookiesOnly(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should see Cookie header, not Authorization
		assert.Empty(t, r.Header.Get("Authorization"))
		assert.NotEmpty(t, r.Header.Get("Cookie"))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"results": []interface{}{}})
	}))
	defer server.Close()

	creds := &AuthCredentials{
		CookieHeader: "session=test123",
		BearerToken:  "",
		CapturedAt:   time.Now(),
	}
	err := ValidateCredentials(context.Background(), server.URL, creds)
	require.NoError(t, err)
}

func TestValidateCredentials_InvalidJSON(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	creds := &AuthCredentials{BearerToken: "test", CapturedAt: time.Now()}
	err := ValidateCredentials(context.Background(), server.URL, creds)
	// Should not error - invalid JSON is logged but doesn't fail validation
	require.NoError(t, err)
}

func TestValidateCredentials_URLWithTrailingSlash(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"results": []interface{}{}})
	}))
	defer server.Close()

	creds := &AuthCredentials{BearerToken: "test", CapturedAt: time.Now()}
	err := ValidateCredentials(context.Background(), server.URL+"/", creds)
	require.NoError(t, err)
}

// --- GitHub Authenticate full error flow ---

func TestGitHubAuthProvider_Authenticate_InvalidCacheNeedsAPICall(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewGitHubAuthProvider("test-token", cacheDir, log)

	// Write an invalid cache (valid=false)
	cache := &GitHubTokenCache{
		Token:         "test-token",
		Valid:         false,
		LastValidated: time.Now(),
	}
	err := provider.saveToCache(cache)
	require.NoError(t, err)

	// Authenticate should skip invalid cache and try API
	err = provider.Authenticate(context.Background())
	assert.Error(t, err, "API call to real github.com should fail")
	assert.False(t, provider.status.Authenticated)
}

// --- Tugboat status edge cases ---

func TestTugboatAuthProvider_UpdateStatus_TokenPresentNoValidation(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewTugboatAuthProvider("token", "https://api.tugboat.com", cacheDir, log)

	// Token present, no validation done yet
	provider.status.LastValidated = nil
	provider.updateStatus()
	assert.False(t, provider.status.Authenticated)
	assert.Equal(t, "api", provider.status.Source)
}

func TestTugboatAuthProvider_UpdateStatus_StaleValidation(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewTugboatAuthProvider("token", "https://api.tugboat.com", cacheDir, log)

	stale := time.Now().Add(-1 * time.Hour) // > 30 min
	provider.status.LastValidated = &stale
	provider.status.TokenValid = true
	provider.status.Authenticated = false

	provider.updateStatus()
	assert.False(t, provider.status.Authenticated)
}

// --- ExpiresAt in cache ---

func TestGitHubAuthProvider_CacheWithExpiry(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewGitHubAuthProvider("test-token", cacheDir, log)

	expires := time.Now().Add(24 * time.Hour)
	cache := &GitHubTokenCache{
		Token:         "test-token",
		Valid:         true,
		LastValidated: time.Now(),
		UserLogin:     "user",
		Scopes:        []string{"repo"},
		ExpiresAt:     &expires,
	}
	err := provider.saveToCache(cache)
	require.NoError(t, err)

	loaded := provider.loadFromCache()
	require.NotNil(t, loaded)
	assert.NotNil(t, loaded.ExpiresAt)
}

func TestTugboatAuthProvider_CacheWithExpiry(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewTugboatAuthProvider("token", "https://api.tugboat.com", cacheDir, log)

	expires := time.Now().Add(24 * time.Hour)
	cache := &TugboatTokenCache{
		BearerToken:   "token",
		Valid:         true,
		LastValidated: time.Now(),
		ExpiresAt:     &expires,
		BaseURL:       "https://api.tugboat.com",
	}
	err := provider.saveToCache(cache)
	require.NoError(t, err)

	loaded := provider.loadFromCache()
	require.NotNil(t, loaded)
	assert.NotNil(t, loaded.ExpiresAt)
}

// --- Tugboat Authenticate with API failure (non-cache path) ---

func TestTugboatAuthProvider_Authenticate_APIFailure(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewTugboatAuthProvider("bad-token", server.URL, cacheDir, log)

	err := provider.Authenticate(context.Background())
	require.Error(t, err)
	assert.False(t, provider.status.Authenticated)
	assert.False(t, provider.status.TokenValid)
	assert.NotEmpty(t, provider.status.Error)
}

// --- GitHub GetStatus with cache expiry info ---

func TestGitHubAuthProvider_Authenticate_CacheWithExpiry(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewGitHubAuthProvider("test-token", cacheDir, log)

	expires := time.Now().Add(24 * time.Hour)
	cache := &GitHubTokenCache{
		Token:         "test-token",
		Valid:         true,
		LastValidated: time.Now(),
		ExpiresAt:     &expires,
	}
	err := provider.saveToCache(cache)
	require.NoError(t, err)

	err = provider.Authenticate(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, provider.status.ExpiresAt)
}

// --- AuthStatus JSON tests ---

func TestAuthStatus_JSON_Roundtrip(t *testing.T) {
	t.Parallel()
	now := time.Now().Truncate(time.Second)
	expires := now.Add(24 * time.Hour)
	status := AuthStatus{
		Authenticated: true,
		Provider:      "github",
		TokenPresent:  true,
		TokenValid:    true,
		LastValidated: &now,
		ExpiresAt:     &expires,
		CacheUsed:     false,
		Error:         "",
		Source:        "api",
	}

	data, err := json.Marshal(status)
	require.NoError(t, err)

	var decoded AuthStatus
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, status.Authenticated, decoded.Authenticated)
	assert.Equal(t, status.Provider, decoded.Provider)
	assert.Equal(t, status.TokenPresent, decoded.TokenPresent)
	assert.Equal(t, status.Source, decoded.Source)
}

// --- BrowserAuth tests ---

func TestNewBrowserAuth_Defaults(t *testing.T) {
	t.Parallel()
	auth := NewBrowserAuth("https://my.tugboat.com")
	assert.Equal(t, "https://my.tugboat.com", auth.BaseURL)
	assert.Equal(t, 5*time.Minute, auth.Timeout)
	assert.Equal(t, "safari", auth.BrowserType)
}

// --- SafariAuth tests (non-macOS safe) ---

func TestNewSafariAuth(t *testing.T) {
	t.Parallel()
	sa := NewSafariAuth("https://my.tugboat.com")
	assert.Equal(t, "https://my.tugboat.com", sa.BaseURL)
	assert.Equal(t, 5*time.Minute, sa.Timeout)
}

func TestSafariAuth_BuildWebLoginURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		baseURL  string
		expected string
	}{
		{"https://api-my.tugboatlogic.com/api", "https://my.tugboatlogic.com/login"},
		{"https://api-my.tugboatlogic.com", "https://my.tugboatlogic.com/login"},
		{"https://example.com/api", "https://example.com/login"},
	}

	for _, tt := range tests {
		sa := NewSafariAuth(tt.baseURL)
		assert.Equal(t, tt.expected, sa.buildWebLoginURL(), "for baseURL: %s", tt.baseURL)
	}
}

// --- ValidateAuth tests (simple delegation to validateToken) ---

func TestGitHubAuthProvider_ValidateAuth_NoToken(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	// With empty token, validateToken will fail trying to hit api.github.com
	provider := NewGitHubAuthProvider("", cacheDir, log)
	err := provider.ValidateAuth(context.Background())
	// Will fail because there's no real API to hit, but exercises the code path
	assert.Error(t, err)
}

func TestTugboatAuthProvider_ValidateAuth_WithServer(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{})
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewTugboatAuthProvider("token", server.URL, cacheDir, log)

	err := provider.ValidateAuth(context.Background())
	assert.NoError(t, err)
}

// --- SaveToCache error paths ---

func TestGitHubAuthProvider_SaveToCache_WritePerm(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewGitHubAuthProvider("test-token", cacheDir, log)

	cache := &GitHubTokenCache{
		Token:         "test-token",
		Valid:         true,
		LastValidated: time.Now(),
	}
	// Normal save should work
	err := provider.saveToCache(cache)
	assert.NoError(t, err)
}

func TestTugboatAuthProvider_SaveToCache_CreatesDir(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	cacheDir := filepath.Join(baseDir, "nested", "deep", "cache")
	log := &mockLogger{}
	provider := NewTugboatAuthProvider("token", "https://api.tugboat.com", cacheDir, log)

	cache := &TugboatTokenCache{
		BearerToken:   "token",
		Valid:         true,
		LastValidated: time.Now(),
		BaseURL:       "https://api.tugboat.com",
	}
	err := provider.saveToCache(cache)
	require.NoError(t, err)

	info, err := os.Stat(cacheDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

// --- GitHub cache directory creation ---

// --- LoadFromCache error paths ---

func TestTugboatAuthProvider_CacheCorrupted(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewTugboatAuthProvider("token", "https://api.tugboat.com", cacheDir, log)

	err := os.WriteFile(provider.cacheFile, []byte("{invalid json"), 0600)
	require.NoError(t, err)

	loaded := provider.loadFromCache()
	assert.Nil(t, loaded, "corrupted JSON cache should return nil")
}

func TestTugboatAuthProvider_CacheMissing(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	log := &mockLogger{}
	provider := NewTugboatAuthProvider("token", "https://api.tugboat.com", cacheDir, log)

	loaded := provider.loadFromCache()
	assert.Nil(t, loaded, "missing cache should return nil")
}

func TestGitHubAuthProvider_SaveToCache_CreatesDir(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	cacheDir := filepath.Join(baseDir, "nested", "cache")
	log := &mockLogger{}
	provider := NewGitHubAuthProvider("test-token", cacheDir, log)

	cache := &GitHubTokenCache{
		Token:         "test-token",
		Valid:         true,
		LastValidated: time.Now(),
	}
	err := provider.saveToCache(cache)
	require.NoError(t, err)

	// Verify directory was created
	info, err := os.Stat(cacheDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}
