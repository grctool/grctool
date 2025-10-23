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
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/transport"
)

// BrowserAuth handles browser-based authentication flow.
// Currently only Safari is supported for automatic cookie extraction on macOS.
type BrowserAuth struct {
	BaseURL     string
	Timeout     time.Duration
	BrowserType string // Browser type - only "safari" is supported
}

// AuthCredentials contains the captured authentication data
type AuthCredentials struct {
	CookieHeader string    `json:"cookie_header"`
	BearerToken  string    `json:"bearer_token,omitempty"`
	OrgID        string    `json:"org_id,omitempty"` // Organization ID extracted from API
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	CapturedAt   time.Time `json:"captured_at"`
}

// AuthResult contains the authentication result
type AuthResult struct {
	Credentials *AuthCredentials
	Error       error
}

// NewBrowserAuth creates a new browser authentication handler
func NewBrowserAuth(baseURL string) *BrowserAuth {
	return &BrowserAuth{
		BaseURL:     baseURL,
		Timeout:     5 * time.Minute,
		BrowserType: "safari", // Default to Safari
	}
}

// Login opens Safari for authentication and captures the cookies.
// This delegates to the Safari-specific implementation which uses AppleScript
// to automatically extract cookies from Safari on macOS.
func (b *BrowserAuth) Login(ctx context.Context) (*AuthCredentials, error) {
	// Only Safari is supported
	if strings.ToLower(b.BrowserType) != "safari" && b.BrowserType != "" {
		return nil, fmt.Errorf("only Safari browser is supported, got: %s", b.BrowserType)
	}

	safariAuth := NewSafariAuth(b.BaseURL)
	safariAuth.Timeout = b.Timeout
	return safariAuth.Login(ctx)
}

// ValidateCredentials tests if the captured credentials are valid
// and extracts the organization ID from the API response
func ValidateCredentials(ctx context.Context, baseURL string, creds *AuthCredentials) error {
	// Create a test request to validate the credentials
	// baseURL might already include /api path, so clean it up
	testURL := strings.TrimSuffix(baseURL, "/")
	if !strings.HasSuffix(testURL, "/api") {
		testURL = testURL + "/api"
	}
	// Use /api/policy/ without org parameter - it should return policies for authenticated user
	validationURL := testURL + "/policy/?page=1&page_size=1"
	req, err := http.NewRequestWithContext(ctx, "GET", validationURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}

	// Set headers to match what Tugboat expects
	if creds.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+creds.BearerToken)
	} else {
		// If no bearer token, still try with cookies
		req.Header.Set("Cookie", creds.CookieHeader)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "grctool/1.0.0")

	// Make the request with logging
	httpTransport := transport.NewLoggingTransport(http.DefaultTransport, logger.WithComponent("auth-validation"))
	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: httpTransport,
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("validation request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check if we got a successful response
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("credentials are invalid or expired")
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("validation failed with status %d", resp.StatusCode)
	}

	// Parse response body to extract org_id
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// Don't fail validation just because we can't read the body
		logger.Warn("Failed to read validation response body", logger.Error(err))
		return nil
	}

	// Parse JSON response to extract org_id from policy results
	// The response structure matches /api/policy/ endpoint
	var apiResponse struct {
		Results []struct {
			OrgID interface{} `json:"org_id"` // Can be int, string, or null
		} `json:"results"`
	}

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		// Don't fail validation, just log the error
		logger.Warn("Failed to parse validation response", logger.Error(err))
		return nil
	}

	// Extract org_id from first result
	if len(apiResponse.Results) > 0 {
		orgID := apiResponse.Results[0].OrgID

		// Handle different org_id types (int, string, or null)
		switch v := orgID.(type) {
		case float64: // JSON numbers are float64
			if v > 0 {
				creds.OrgID = fmt.Sprintf("%.0f", v)
				logger.Debug("Extracted org_id from API", logger.String("org_id", creds.OrgID))
			}
		case string:
			if v != "" {
				creds.OrgID = v
				logger.Debug("Extracted org_id from API", logger.String("org_id", creds.OrgID))
			}
		case int:
			if v > 0 {
				creds.OrgID = fmt.Sprintf("%d", v)
				logger.Debug("Extracted org_id from API", logger.String("org_id", creds.OrgID))
			}
		default:
			logger.Warn("No valid org_id found in validation response", logger.String("org_id_type", fmt.Sprintf("%T", orgID)))
		}
	} else {
		logger.Warn("No policies found in validation response")
	}

	return nil
}
