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
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBrowserAuth(t *testing.T) {
	auth := NewBrowserAuth("https://example.com")

	assert.Equal(t, "https://example.com", auth.BaseURL)
	assert.Equal(t, 5*time.Minute, auth.Timeout)
	assert.Equal(t, "safari", auth.BrowserType)
}

func TestExtractBearerTokenFromSafari(t *testing.T) {
	tests := []struct {
		name     string
		cookie   string
		expected string
	}{
		{
			name:     "Tugboat token cookie (base64 encoded JSON)",
			cookie:   "session=abc123; token=eyJhY2Nlc3NfdG9rZW4iOiJ3YmlzN0o5eFV1dFhvOWRiVFBoeEppVGU2a0dUTlEiLCJleHBpcmVzIjoiMjAyNS0wNy0wNFQxMDo0NTozMS45ODYwNTZaIiwic2NvcGUiOiJyZWFkIHdyaXRlIiwib3JnIjoxMzg4OCwicmVmcmVzaF90b2tlbiI6bnVsbCwidG9rZW5fdHlwZSI6IkJlYXJlciJ9; other=value",
			expected: "wbis7J9xUutXo9dbTPhxJiTe6kGTNQ",
		},
		{
			name:     "Bearer token in auth_token",
			cookie:   "session=abc123; bearer_auth=Bearer1234567890abcdefghijklmnop; other=value",
			expected: "Bearer1234567890abcdefghijklmnop",
		},
		{
			name:     "No token found",
			cookie:   "session=abc123; user=john; theme=dark",
			expected: "",
		},
		{
			name:     "Empty cookie",
			cookie:   "",
			expected: "",
		},
		{
			name:     "Cookies without valid token (common bug scenario)",
			cookie:   "session=abc123; csrftoken=xyz789; user_preferences=dark_mode; last_activity=2025-01-01",
			expected: "",
		},
		{
			name:     "Token cookie with invalid base64 (should return empty)",
			cookie:   "session=abc123; token=invalidbase64!@#; other=value",
			expected: "",
		},
		{
			name:     "Token cookie with valid base64 but invalid JSON",
			cookie:   "session=abc123; token=bm90IGpzb24=; other=value", // base64 for "not json"
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractBearerTokenFromSafari(tt.cookie)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateCredentials(t *testing.T) {
	tests := []struct {
		name        string
		serverFunc  func(w http.ResponseWriter, r *http.Request)
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid credentials",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				// Check that authorization header is present (bearer token preferred)
				auth := r.Header.Get("Authorization")
				cookie := r.Header.Get("Cookie")
				if auth == "" && cookie == "" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				// Verify correct headers are sent
				if auth != "Bearer test-bearer-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"data": []map[string]interface{}{
						{"id": 1, "name": "Test Policy"},
					},
				})
			},
			expectError: false,
		},
		{
			name: "Invalid credentials",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "Invalid authentication",
				})
			},
			expectError: true,
			errorMsg:    "credentials are invalid or expired",
		},
		{
			name: "Server error",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectError: true,
			errorMsg:    "validation failed with status 500",
		},
		{
			name: "Missing bearer token (cookies only)",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				// Check if only cookie header is present (no Authorization header)
				auth := r.Header.Get("Authorization")
				cookie := r.Header.Get("Cookie")
				if auth == "" && cookie != "" {
					// Simulate Tugboat Logic rejecting cookie-only requests
					w.WriteHeader(http.StatusUnauthorized)
					_ = json.NewEncoder(w).Encode(map[string]string{
						"error": "Bearer token required",
					})
					return
				}
				w.WriteHeader(http.StatusOK)
			},
			expectError: true,
			errorMsg:    "credentials are invalid or expired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(tt.serverFunc))
			defer server.Close()

			// Create test credentials based on test case
			var creds *AuthCredentials
			if tt.name == "Missing bearer token (cookies only)" {
				// Test credentials with cookies but no bearer token
				creds = &AuthCredentials{
					CookieHeader: "session=test123; csrftoken=xyz789; user=john",
					BearerToken:  "", // No bearer token - this is the bug scenario
					CapturedAt:   time.Now(),
				}
			} else {
				// Default test credentials with bearer token
				creds = &AuthCredentials{
					CookieHeader: "session=test123; token=abc",
					BearerToken:  "test-bearer-token",
					CapturedAt:   time.Now(),
				}
			}

			// Test validation
			ctx := context.Background()
			err := ValidateCredentials(ctx, server.URL, creds)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBrowserAuth_Login_NonMacOS(t *testing.T) {
	// This test will only run on non-macOS systems
	if runtime.GOOS == "darwin" {
		t.Skip("Skipping non-macOS test on macOS")
	}

	auth := NewBrowserAuth("https://example.com")
	ctx := context.Background()

	_, err := auth.Login(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "safari automation is only supported on macOS")
}

func TestBrowserAuth_Login_UnsupportedBrowser(t *testing.T) {
	auth := NewBrowserAuth("https://example.com")
	auth.BrowserType = "chrome" // Not supported anymore

	ctx := context.Background()
	_, err := auth.Login(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "only Safari browser is supported")
}
