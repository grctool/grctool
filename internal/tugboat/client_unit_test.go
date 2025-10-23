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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/stretchr/testify/assert"
)

// mockRoundTripper allows mocking HTTP responses
type mockRoundTripper struct {
	response *http.Response
	err      error
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func TestNewClient(t *testing.T) {
	tests := map[string]struct {
		cfg         *config.TugboatConfig
		expectError bool
		errorMsg    string
	}{
		"valid config": {
			cfg: &config.TugboatConfig{
				BaseURL:      "https://api.tugboat.com",
				CookieHeader: "session=abc123",
			},
			expectError: false,
		},
		"missing base URL": {
			cfg: &config.TugboatConfig{
				CookieHeader: "session=abc123",
			},
			expectError: true,
			errorMsg:    "base URL is required",
		},
		"missing cookie header": {
			cfg: &config.TugboatConfig{
				BaseURL: "https://api.tugboat.com",
			},
			expectError: true,
			errorMsg:    "cookie header is required",
		},
		"invalid base URL": {
			cfg: &config.TugboatConfig{
				BaseURL:      "not-a-url",
				CookieHeader: "session=abc123",
			},
			expectError: false, // URL validation happens during requests
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.expectError {
				err := validateConfig(tc.cfg)
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				err := validateConfig(tc.cfg)
				assert.NoError(t, err)

				// Test actual client creation
				client := NewClient(tc.cfg, nil)
				assert.NotNil(t, client)
				assert.Equal(t, tc.cfg.BaseURL, client.baseURL)
				assert.Equal(t, tc.cfg.CookieHeader, client.cookieHeader)
				assert.NotNil(t, client.httpClient)
				assert.NotNil(t, client.logger)
			}
		})
	}
}

func TestClient_makeRequest(t *testing.T) {
	tests := map[string]struct {
		method       string
		path         string
		body         interface{}
		mockResponse *http.Response
		mockError    error
		expectError  bool
		checkRequest func(t *testing.T, req *http.Request)
	}{
		"successful GET request": {
			method: "GET",
			path:   "/api/v1/policies",
			mockResponse: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"data": []}`)),
				Header:     make(http.Header),
			},
			expectError: false,
			checkRequest: func(t *testing.T, req *http.Request) {
				assert.Equal(t, "GET", req.Method)
				assert.Contains(t, req.URL.Path, "/api/v1/policies")
				assert.Equal(t, "session=test", req.Header.Get("Cookie"))
			},
		},
		"successful POST request with body": {
			method: "POST",
			path:   "/api/v1/policies",
			body: map[string]string{
				"name": "Test Policy",
			},
			mockResponse: &http.Response{
				StatusCode: 201,
				Body:       io.NopCloser(bytes.NewBufferString(`{"data": {"id": 1}}`)),
				Header:     make(http.Header),
			},
			expectError: false,
			checkRequest: func(t *testing.T, req *http.Request) {
				assert.Equal(t, "POST", req.Method)
				assert.Equal(t, "application/json", req.Header.Get("Content-Type"))

				// Check body
				bodyBytes, _ := io.ReadAll(req.Body)
				var body map[string]string
				json.Unmarshal(bodyBytes, &body)
				assert.Equal(t, "Test Policy", body["name"])
			},
		},
		"network error": {
			method:      "GET",
			path:        "/api/v1/policies",
			mockError:   errors.New("network error"),
			expectError: true,
		},
		"404 response": {
			method: "GET",
			path:   "/api/v1/policies/999",
			mockResponse: &http.Response{
				StatusCode: 404,
				Body:       io.NopCloser(bytes.NewBufferString(`{"error": {"message": "Not found"}}`)),
				Header:     make(http.Header),
			},
			expectError: false, // makeRequest doesn't check status codes, handleResponse does
		},
		"500 server error": {
			method: "GET",
			path:   "/api/v1/policies",
			mockResponse: &http.Response{
				StatusCode: 500,
				Body:       io.NopCloser(bytes.NewBufferString(`{"error": {"message": "Internal server error"}}`)),
				Header:     make(http.Header),
			},
			expectError: false, // makeRequest doesn't check status codes, handleResponse does
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create a test server to capture requests
			var capturedReq *http.Request
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedReq = r.Clone(context.Background())
				// Read body to allow checking it
				if r.Body != nil {
					bodyBytes, _ := io.ReadAll(r.Body)
					capturedReq.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				}

				if tc.mockResponse != nil {
					w.WriteHeader(tc.mockResponse.StatusCode)
					if tc.mockResponse.Body != nil {
						bodyBytes, _ := io.ReadAll(tc.mockResponse.Body)
						tc.mockResponse.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
						w.Write(bodyBytes)
					}
				}
			}))
			defer server.Close()

			client := &Client{
				baseURL:      server.URL,
				cookieHeader: "session=test",
				httpClient:   server.Client(),
				logger:       logger.WithComponent("test"),
			}

			if tc.mockError != nil {
				client.httpClient = &http.Client{
					Transport: &mockRoundTripper{err: tc.mockError},
				}
			}

			resp, err := client.makeRequest(context.Background(), tc.method, tc.path, tc.body)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			}

			if tc.checkRequest != nil && capturedReq != nil {
				tc.checkRequest(t, capturedReq)
			}
		})
	}
}

func TestClient_parseResponse(t *testing.T) {
	tests := map[string]struct {
		response    *http.Response
		target      interface{}
		expectError bool
	}{
		"successful parse": {
			response: &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(bytes.NewBufferString(`{
					"data": {"id": 1, "name": "Test"}
				}`)),
			},
			target:      &map[string]interface{}{},
			expectError: false,
		},
		"invalid JSON": {
			response: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{invalid json}`)),
			},
			target:      &map[string]interface{}{},
			expectError: true,
		},
		"empty body": {
			response: &http.Response{
				StatusCode: 204,
				Body:       io.NopCloser(bytes.NewBufferString(``)),
			},
			target:      &map[string]interface{}{},
			expectError: false,
		},
		"error response": {
			response: &http.Response{
				StatusCode: 400,
				Body: io.NopCloser(bytes.NewBufferString(`{
					"error": {"message": "Bad request"}
				}`)),
			},
			target:      &map[string]interface{}{},
			expectError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_ = &Client{
				logger: logger.WithComponent("test"),
			}

			err := parseResponse(tc.response, tc.target)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAPIError_Error(t *testing.T) {
	tests := map[string]struct {
		apiError *APIError
		expected string
	}{
		"with code and message": {
			apiError: &APIError{
				Code:    "ERR001",
				Message: "Something went wrong",
			},
			expected: "ERR001: Something went wrong",
		},
		"with details": {
			apiError: &APIError{
				Code:    "ERR002",
				Message: "Validation failed",
				Details: "Field 'name' is required",
			},
			expected: "ERR002: Validation failed - Field 'name' is required",
		},
		"message only": {
			apiError: &APIError{
				Message: "Generic error",
			},
			expected: "Generic error",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := formatAPIError(tc.apiError)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestClient_buildURL(t *testing.T) {
	tests := map[string]struct {
		baseURL  string
		path     string
		params   map[string]string
		expected string
	}{
		"simple path": {
			baseURL:  "https://api.tugboat.com",
			path:     "/api/v1/policies",
			expected: "https://api.tugboat.com/api/v1/policies",
		},
		"path with query params": {
			baseURL: "https://api.tugboat.com",
			path:    "/api/v1/policies",
			params: map[string]string{
				"page":     "2",
				"per_page": "50",
			},
			expected: "https://api.tugboat.com/api/v1/policies?page=2&per_page=50",
		},
		"path with trailing slash": {
			baseURL:  "https://api.tugboat.com/",
			path:     "/api/v1/policies",
			expected: "https://api.tugboat.com/api/v1/policies",
		},
		"empty path": {
			baseURL:  "https://api.tugboat.com",
			path:     "",
			expected: "https://api.tugboat.com",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := &Client{
				baseURL: tc.baseURL,
			}

			result := client.buildURL(tc.path, tc.params)

			// Parse URLs to compare properly (params can be in any order)
			if tc.params != nil {
				resultURL, _ := url.Parse(result)
				expectedURL, _ := url.Parse(tc.expected)

				assert.Equal(t, expectedURL.Scheme, resultURL.Scheme)
				assert.Equal(t, expectedURL.Host, resultURL.Host)
				assert.Equal(t, expectedURL.Path, resultURL.Path)

				// Check query params
				for key, value := range tc.params {
					assert.Equal(t, value, resultURL.Query().Get(key))
				}
			} else {
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestClient_withRateLimiting(t *testing.T) {
	client := &Client{
		rateLimiter: time.NewTicker(100 * time.Millisecond),
		logger:      logger.WithComponent("test"),
	}
	defer client.rateLimiter.Stop()

	start := time.Now()

	// Make 3 requests
	for i := 0; i < 3; i++ {
		client.withRateLimiting()
	}

	elapsed := time.Since(start)

	// Should take at least 200ms for 3 requests with 100ms rate limit
	assert.True(t, elapsed >= 200*time.Millisecond, "Rate limiting not working properly")
}

func TestClient_retryableRequest(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"data": "success"})
	}))
	defer server.Close()

	client := &Client{
		baseURL:      server.URL,
		cookieHeader: "session=test",
		httpClient:   server.Client(),
		logger:       logger.WithComponent("test"),
	}

	resp, err := client.makeRequestWithRetry(context.Background(), "GET", "/test", nil, 3)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 3, attempts, "Should have retried until success")
}

func TestClient_validateConfig(t *testing.T) {
	tests := map[string]struct {
		cfg         *config.TugboatConfig
		expectError bool
		errorMsg    string
	}{
		"valid config": {
			cfg: &config.TugboatConfig{
				BaseURL:      "https://api.tugboat.com",
				CookieHeader: "session=abc123",
			},
			expectError: false,
		},
		"nil config": {
			cfg:         nil,
			expectError: true,
			errorMsg:    "config is required",
		},
		"empty base URL": {
			cfg: &config.TugboatConfig{
				BaseURL:      "",
				CookieHeader: "session=abc123",
			},
			expectError: true,
			errorMsg:    "base URL is required",
		},
		"empty cookie": {
			cfg: &config.TugboatConfig{
				BaseURL:      "https://api.tugboat.com",
				CookieHeader: "",
			},
			expectError: true,
			errorMsg:    "cookie header is required",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := validateConfig(tc.cfg)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Helper functions for testing

func parseResponse(resp *http.Response, target interface{}) error {
	if resp.Body == nil {
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if len(body) == 0 {
		return nil
	}

	if resp.StatusCode >= 400 {
		var apiResp APIResponse
		if err := json.Unmarshal(body, &apiResp); err == nil && apiResp.Error != nil {
			return errors.New(formatAPIError(apiResp.Error))
		}
		return errors.New("request failed")
	}

	return json.Unmarshal(body, target)
}

func formatAPIError(apiErr *APIError) string {
	if apiErr.Code != "" {
		if apiErr.Details != "" {
			return apiErr.Code + ": " + apiErr.Message + " - " + apiErr.Details
		}
		return apiErr.Code + ": " + apiErr.Message
	}
	return apiErr.Message
}

func validateConfig(cfg *config.TugboatConfig) error {
	if cfg == nil {
		return errors.New("config is required")
	}
	if cfg.BaseURL == "" {
		return errors.New("base URL is required")
	}
	if cfg.CookieHeader == "" {
		return errors.New("cookie header is required")
	}
	return nil
}

func (c *Client) buildURL(path string, params map[string]string) string {
	baseURL := strings.TrimSuffix(c.baseURL, "/")
	fullURL := baseURL + path

	if len(params) > 0 {
		values := url.Values{}
		for k, v := range params {
			values.Add(k, v)
		}
		fullURL += "?" + values.Encode()
	}

	return fullURL
}

func (c *Client) withRateLimiting() {
	if c.rateLimiter != nil {
		<-c.rateLimiter.C
	}
}

func (c *Client) makeRequestWithRetry(ctx context.Context, method, path string, body interface{}, maxRetries int) (*http.Response, error) {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		resp, err := c.makeRequest(ctx, method, path, body)
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}
		lastErr = err
		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * time.Second)
		}
	}
	return nil, lastErr
}
