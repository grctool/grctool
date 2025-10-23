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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/appcontext"
)

// SafariAuth handles automatic cookie capture using Safari on macOS.
//
// This implementation uses AppleScript to:
// 1. Open Safari to the Tugboat Logic login page
// 2. Wait for user to complete authentication (supports Touch ID, Face ID, 1Password)
// 3. Automatically extract cookies using document.cookie JavaScript via AppleScript
// 4. Parse the base64-encoded JSON token cookie to extract the bearer token
// 5. Fall back to manual cookie extraction if automatic extraction fails
//
// Advantages over other browsers:
// - No profile corruption (Safari doesn't use profiles like Chrome)
// - Native macOS integration with Keychain and biometric auth
// - Preserves all extensions and bookmarks
// - Can extract bearer token from non-httpOnly cookies automatically
//
// Requirements:
// - macOS only (uses AppleScript)
// - Safari Developer settings: "Allow JavaScript from Apple Events" enabled for automatic extraction
type SafariAuth struct {
	BaseURL string
	Timeout time.Duration
}

// NewSafariAuth creates a new Safari-based authentication handler
func NewSafariAuth(baseURL string) *SafariAuth {
	return &SafariAuth{
		BaseURL: baseURL,
		Timeout: 5 * time.Minute,
	}
}

// Login launches Safari and uses AppleScript to automate cookie capture.
//
// Flow:
// 1. Checks macOS compatibility
// 2. Tests Safari JavaScript automation permissions
// 3. Opens Safari to Tugboat Logic login page
// 4. Waits for user authentication
// 5. Extracts cookies via AppleScript (document.cookie)
// 6. Parses bearer token from base64-encoded JSON token cookie
// 7. Retries if cookies are found but no valid token (up to 3 attempts)
// 8. Falls back to manual extraction if automatic fails
//
// Returns AuthCredentials with both cookie header and extracted bearer token.
func (s *SafariAuth) Login(ctx context.Context) (*AuthCredentials, error) {
	if runtime.GOOS != "darwin" {
		return nil, fmt.Errorf("safari automation is only supported on macOS")
	}

	appcontext.Printf(ctx, "🦎 Starting automatic cookie capture with Safari...\n")
	appcontext.Printf(ctx, "🍎 Safari will open automatically with your existing cookies and extensions!\n")
	appcontext.Printf(ctx, "🔑 Use Touch ID, Face ID, or Keychain for seamless authentication\n\n")

	// Check if Safari JavaScript automation is enabled
	if err := s.checkSafariAutomation(); err != nil {
		appcontext.Printf(ctx, "⚠️  %v\n\n", err)
	}

	// Convert API URL to web login URL
	loginURL := s.buildWebLoginURL()

	appcontext.Printf(ctx, "🦎 Opening Safari to: %s\n", loginURL)
	appcontext.Printf(ctx, "⏳ Waiting for login... (timeout: %v)\n\n", s.Timeout)

	// Open Safari with the login URL
	cmd := exec.Command("open", "-a", "Safari", loginURL)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to open Safari: %w", err)
	}

	appcontext.Printf(ctx, "📋 Please complete the following steps:\n")
	appcontext.Printf(ctx, "   1. Log in to Tugboat Logic in Safari (use Touch ID/Face ID if available)\n")
	appcontext.Printf(ctx, "   2. After successful login, we'll automatically extract cookies\n\n")

	// Wait a moment for Safari to load
	time.Sleep(3 * time.Second)

	// Use AppleScript to extract cookies from Safari with retry logic
	appcontext.Printf(ctx, "🔄 Attempting to extract cookies using AppleScript...\n")

	cookies, err := s.extractCookiesFromSafariWithRetry(ctx)
	if err != nil {
		return nil, fmt.Errorf("automatic cookie extraction failed: %w", err)
	}

	appcontext.Printf(ctx, "✅ Authentication successful with Safari!\n")
	appcontext.Printf(ctx, "🍪 Extracted %d characters of cookie data\n", len(cookies.CookieHeader))
	if cookies.BearerToken != "" {
		appcontext.Printf(ctx, "🔑 Bearer token extracted successfully\n")
	}

	return cookies, nil
}

// checkSafariAutomation verifies if Safari JavaScript automation is enabled
func (s *SafariAuth) checkSafariAutomation() error {
	// Try a simple JavaScript test
	testScript := `tell application "Safari" to do JavaScript "1+1" in current tab of front window`
	cmd := exec.Command("osascript", "-e", testScript)
	output, err := cmd.CombinedOutput()

	if err != nil && strings.Contains(string(output), "Allow JavaScript from Apple Events") {
		return fmt.Errorf("safari automation requires enabling 'Allow JavaScript from Apple Events' in Developer settings")
	}

	return nil
}

// extractCookiesFromSafariWithRetry attempts to extract cookies with retry logic for token validation
func (s *SafariAuth) extractCookiesFromSafariWithRetry(ctx context.Context) (*AuthCredentials, error) {
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		appcontext.Printf(ctx, "🔍 Extraction attempt %d/%d...\n", attempt, maxRetries)

		cookies, err := s.extractCookiesFromSafari(ctx)
		if err != nil {
			// If we can't extract cookies at all, return the error
			return nil, err
		}

		// Check if we got a valid bearer token
		if cookies.BearerToken != "" {
			appcontext.Printf(ctx, "✅ Valid bearer token found on attempt %d\n", attempt)
			return cookies, nil
		}

		// We have cookies but no valid token
		if attempt < maxRetries {
			appcontext.Printf(ctx, "⚠️  Cookies extracted but no valid bearer token found\n")
			appcontext.Printf(ctx, "💡 This often happens when the token cookie isn't set immediately after login\n")
			appcontext.Printf(ctx, "📋 Please try one of the following in Safari:\n")
			appcontext.Printf(ctx, "   1. Refresh the page (⌘R)\n")
			appcontext.Printf(ctx, "   2. Navigate to another page in Tugboat Logic\n")
			appcontext.Printf(ctx, "   3. Wait a moment for the session to fully initialize\n")
			appcontext.Printf(ctx, "⏳ Retrying in 5 seconds... (attempt %d/%d)\n\n", attempt+1, maxRetries)

			time.Sleep(5 * time.Second)
		} else {
			// Final attempt failed
			appcontext.Printf(ctx, "❌ No valid bearer token found after %d attempts\n", maxRetries)
			appcontext.Printf(ctx, "🍪 Cookies were captured but they don't contain the authentication token\n")
			appcontext.Printf(ctx, "💡 Try running 'grctool auth login' again - this often works on the second attempt\n")
			return nil, fmt.Errorf("no valid bearer token found in cookies after %d attempts", maxRetries)
		}
	}

	// This should never be reached, but just in case
	return nil, fmt.Errorf("failed to extract valid credentials after retries")
}

// extractCookiesFromSafari uses AppleScript to get cookies from Safari
func (s *SafariAuth) extractCookiesFromSafari(ctx context.Context) (*AuthCredentials, error) {
	// AppleScript to get cookies from Safari
	// Note: document.cookie only returns non-httpOnly cookies
	script := `
tell application "Safari"
	-- Get the current tab's URL to verify we're on the right site
	set currentURL to URL of current tab of front window

	-- Check if we're on tugboatlogic.com
	if currentURL contains "tugboatlogic.com" then
		-- Use JavaScript to get document.cookie
		set cookieString to do JavaScript "document.cookie" in current tab of front window
		return cookieString
	else
		error "Not on tugboatlogic.com"
	end if
end tell`

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if it's the JavaScript permission error
		outputStr := string(output)
		if strings.Contains(outputStr, "Allow JavaScript from Apple Events") {
			appcontext.Printf(ctx, "⚠️  Safari requires permission to execute JavaScript via AppleScript\n")
			appcontext.Printf(ctx, "📋 To enable automatic cookie extraction:\n")
			appcontext.Printf(ctx, "   1. Open Safari → Settings (⌘,)\n")
			appcontext.Printf(ctx, "   2. Go to the Advanced tab\n")
			appcontext.Printf(ctx, "   3. Check 'Show features for web developers'\n")
			appcontext.Printf(ctx, "   4. Go to the Developer tab (now visible)\n")
			appcontext.Printf(ctx, "   5. Check 'Allow JavaScript from Apple Events'\n")
			appcontext.Printf(ctx, "   6. Run this command again\n\n")
			return nil, fmt.Errorf("safari JavaScript permission required")
		}
		return nil, fmt.Errorf("AppleScript failed: %w (output: %s)", err, outputStr)
	}

	cookieHeader := strings.TrimSpace(string(output))
	if cookieHeader == "" {
		return nil, fmt.Errorf("no cookies found - please ensure you're logged into Tugboat Logic in Safari")
	}

	// Extract bearer token from cookie if present
	bearerToken := extractBearerTokenFromSafari(cookieHeader)

	// Create credentials (even if no token found - let caller handle validation)
	credentials := &AuthCredentials{
		CookieHeader: cookieHeader,
		BearerToken:  bearerToken,
		CapturedAt:   time.Now(),
		ExpiresAt:    time.Now().Add(24 * time.Hour), // Default 24h expiry
	}

	return credentials, nil
}

// buildWebLoginURL converts API URL to web login URL
func (s *SafariAuth) buildWebLoginURL() string {
	webURL := strings.Replace(s.BaseURL, "api-my.tugboatlogic.com", "my.tugboatlogic.com", 1)
	webURL = strings.Replace(webURL, "/api", "", 1)
	return webURL + "/login"
}

// extractBearerTokenFromSafari attempts to extract a bearer token from the cookie string.
//
// Tugboat Logic stores the bearer token in a base64-encoded JSON cookie named "token".
// The JSON structure contains: {"access_token": "...", "token_type": "Bearer", ...}
// This function decodes the base64, parses the JSON, and extracts access_token.
//
// Falls back to checking other common token cookie names if the main token cookie
// is not found or cannot be decoded.
func extractBearerTokenFromSafari(cookieStr string) string {
	// Look for token patterns in the cookie
	cookies := strings.Split(cookieStr, ";")
	for _, cookie := range cookies {
		cookie = strings.TrimSpace(cookie)
		parts := strings.SplitN(cookie, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Check for the "token" cookie that Tugboat uses
		if key == "token" {
			// The token cookie contains base64-encoded JSON with the actual bearer token
			decoded, err := base64.StdEncoding.DecodeString(value)
			if err != nil {
				// If base64 decode fails, this is not a valid Tugboat token cookie
				// Don't return the raw value as it's likely a session cookie, not a bearer token
				continue
			}

			// Parse the JSON to extract the access_token
			var tokenData struct {
				AccessToken string `json:"access_token"`
				TokenType   string `json:"token_type"`
			}
			if err := json.Unmarshal(decoded, &tokenData); err == nil && tokenData.AccessToken != "" {
				return tokenData.AccessToken
			}
			// If JSON parsing fails or no access_token, continue looking for other tokens
		}

		// Fallback: Check for other common token cookie names
		if strings.Contains(strings.ToLower(key), "bearer") ||
			strings.Contains(strings.ToLower(key), "auth") {
			// Attempt to decode if it looks like a JWT or bearer token
			if strings.Count(value, ".") == 2 || len(value) > 20 {
				return value
			}
		}
	}

	return ""
}
