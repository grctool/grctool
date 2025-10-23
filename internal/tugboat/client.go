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

package tugboat

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/transport"
	"github.com/grctool/grctool/internal/vcr"
)

// Client represents the Tugboat Logic API client
type Client struct {
	baseURL      string
	cookieHeader string
	httpClient   *http.Client
	rateLimiter  *time.Ticker
	logger       logger.Logger
	vcrConfig    *vcr.Config
}

// APIResponse represents a generic API response wrapper
type APIResponse struct {
	Data  interface{} `json:"data"`
	Error *APIError   `json:"error,omitempty"`
	Meta  *APIMeta    `json:"meta,omitempty"`
}

// APIError represents an API error response
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// APIMeta represents API response metadata
type APIMeta struct {
	Page       int `json:"page,omitempty"`
	PerPage    int `json:"per_page,omitempty"`
	Total      int `json:"total,omitempty"`
	TotalPages int `json:"total_pages,omitempty"`
}

// NewClient creates a new Tugboat Logic API client
func NewClient(cfg *config.TugboatConfig, vcrConfig *vcr.Config) *Client {
	// Start with default transport
	httpTransport := http.DefaultTransport

	// Create logger for the client
	log := logger.WithComponent("tugboat")
	if log == nil {
		// If global logger isn't initialized, create a test logger
		log, _ = logger.NewTestLogger()
	}

	// Wrap with logging if enabled
	if cfg.LogAPIRequests || cfg.LogAPIResponses {
		httpTransport = transport.NewLoggingTransport(httpTransport, log)
	}

	// Wrap with VCR if enabled (VCR should be outermost to record actual requests)
	if vcrConfig != nil && vcrConfig.Enabled {
		httpTransport = vcr.New(vcrConfig)
	}

	client := &Client{
		baseURL:      cfg.BaseURL,
		cookieHeader: cfg.CookieHeader,
		httpClient: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: httpTransport,
		},
		logger:    log,
		vcrConfig: vcrConfig,
	}

	// Set up rate limiting if specified
	if cfg.RateLimit > 0 {
		client.rateLimiter = time.NewTicker(time.Second / time.Duration(cfg.RateLimit))
	}

	return client
}

// extractBearerToken extracts the bearer token from the cookie header
func (c *Client) extractBearerToken() (string, error) {
	if c.cookieHeader == "" {
		return "", fmt.Errorf("no cookie header provided")
	}

	// Parse cookies to find the token cookie
	cookies := strings.Split(c.cookieHeader, "; ")
	for _, cookie := range cookies {
		if strings.HasPrefix(cookie, "token=") {
			tokenValue := strings.TrimPrefix(cookie, "token=")

			// First try to decode as base64
			decodedBytes, err := base64.StdEncoding.DecodeString(tokenValue)
			if err != nil {
				// If base64 decode fails, try URL unescape
				decodedToken, err := url.QueryUnescape(tokenValue)
				if err != nil {
					return "", fmt.Errorf("failed to decode token: %w", err)
				}
				decodedBytes = []byte(decodedToken)
			}

			// Parse the JSON token to extract access_token
			var tokenData map[string]interface{}
			if err := json.Unmarshal(decodedBytes, &tokenData); err != nil {
				return "", fmt.Errorf("failed to parse token JSON: %w", err)
			}

			accessToken, ok := tokenData["access_token"].(string)
			if !ok {
				return "", fmt.Errorf("access_token not found in token")
			}

			return accessToken, nil
		}
	}

	return "", fmt.Errorf("token cookie not found")
}

// Close cleans up client resources
func (c *Client) Close() {
	if c.rateLimiter != nil {
		c.rateLimiter.Stop()
	}
}

// makeRequest makes an HTTP request to the Tugboat Logic API
func (c *Client) makeRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
	// Rate limiting
	if c.rateLimiter != nil {
		select {
		case <-c.rateLimiter.C:
			// Continue
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Prepare request body
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	// Create request
	url := c.baseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	// Skip bearer token extraction in VCR playback mode
	isVCRPlayback := c.vcrConfig != nil && c.vcrConfig.Enabled && c.vcrConfig.Mode == vcr.ModePlayback
	if isVCRPlayback {
		// In playback mode, set a dummy token for request matching
		req.Header.Set("Authorization", "Bearer vcr-playback-mode")
	} else if c.cookieHeader != "" {
		// Try to extract bearer token, but don't fail if it's not in the expected format
		// (this allows unit tests with mock servers to use any cookie format)
		var bearerToken string
		bearerToken, err = c.extractBearerToken()
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+bearerToken)
		} else {
			// Fallback: set Cookie header directly for test scenarios
			// where cookie isn't in JWT format
			req.Header.Set("Cookie", c.cookieHeader)
		}
	}
	// If no cookie header and not VCR playback, skip auth header (for unit tests)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "grctool/1.0.0")

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// handleResponse processes the HTTP response and handles common error cases
func (c *Client) handleResponse(resp *http.Response, result interface{}) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Log raw API response for debugging if debug logging is enabled
	if c.logger != nil && strings.Contains(resp.Request.URL.Path, "policy") {
		bodyLen := len(body)
		previewLen := 1000
		if bodyLen < previewLen {
			previewLen = bodyLen
		}
		c.logger.Debug("Raw API response body",
			logger.String("url", resp.Request.URL.Path),
			logger.String("body_preview", string(body[:previewLen])))
	}

	// Check for HTTP errors
	if resp.StatusCode >= 400 {
		var apiResp APIResponse
		if err := json.Unmarshal(body, &apiResp); err != nil {
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
		}
		if apiResp.Error != nil {
			return fmt.Errorf("API error %s: %s", apiResp.Error.Code, apiResp.Error.Message)
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Parse successful response
	if result != nil {
		// Try to unmarshal directly first
		if err := json.Unmarshal(body, result); err != nil {
			// If that fails, try as wrapped APIResponse
			var apiResp APIResponse
			if err := json.Unmarshal(body, &apiResp); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}
			// Extract data from wrapped response
			if apiResp.Data != nil {
				dataBytes, err := json.Marshal(apiResp.Data)
				if err != nil {
					return fmt.Errorf("failed to marshal response data: %w", err)
				}
				if err := json.Unmarshal(dataBytes, result); err != nil {
					return fmt.Errorf("failed to unmarshal response data: %w", err)
				}
			}
		}
	}

	return nil
}

// TestConnection tests the connection to Tugboat Logic API
func (c *Client) TestConnection(ctx context.Context) error {
	// Test with the actual evidence endpoint to verify authentication
	resp, err := c.makeRequest(ctx, "GET", "/api/org_evidence/?page=1&page_size=1&org=13888", nil)
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("authentication failed - invalid token")
	}
	if resp.StatusCode == 403 {
		return fmt.Errorf("access denied - insufficient permissions")
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("connection test failed with status: %d", resp.StatusCode)
	}

	return nil
}

// GET request helper
func (c *Client) get(ctx context.Context, endpoint string, result interface{}) error {
	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return err
	}
	return c.handleResponse(resp, result)
}

// POST request helper
func (c *Client) post(ctx context.Context, endpoint string, body interface{}, result interface{}) error {
	resp, err := c.makeRequest(ctx, "POST", endpoint, body)
	if err != nil {
		return err
	}
	return c.handleResponse(resp, result)
}

// PUT request helper
func (c *Client) put(ctx context.Context, endpoint string, body interface{}, result interface{}) error {
	resp, err := c.makeRequest(ctx, "PUT", endpoint, body)
	if err != nil {
		return err
	}
	return c.handleResponse(resp, result)
}

// PATCH request helper
func (c *Client) patch(ctx context.Context, endpoint string, body interface{}, result interface{}) error {
	resp, err := c.makeRequest(ctx, "PATCH", endpoint, body)
	if err != nil {
		return err
	}
	return c.handleResponse(resp, result)
}

// DELETE request helper
func (c *Client) delete(ctx context.Context, endpoint string) error {
	resp, err := c.makeRequest(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return err
	}
	return c.handleResponse(resp, nil)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
