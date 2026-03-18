// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package accountablehq

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Compile-time interface assertion.
var _ AccountableHQClient = (*HTTPClient)(nil)

// HTTPClient implements AccountableHQClient using real HTTP calls.
// Endpoint paths are configurable to support API discovery validation.
type HTTPClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// HTTPClientConfig holds configuration for the HTTP client.
type HTTPClientConfig struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

// NewHTTPClient creates a real HTTP client for AccountableHQ.
func NewHTTPClient(cfg HTTPClientConfig) *HTTPClient {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &HTTPClient{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:  cfg.APIKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// apiResponse wraps the AccountableHQ API response envelope.
type apiResponse struct {
	Data json.RawMessage `json:"data"`
	Meta *apiMeta        `json:"meta,omitempty"`
}

type apiMeta struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}

func (c *HTTPClient) TestConnection(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/v1/policies?per_page=1", nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("connection test: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("authentication failed (401)")
	}
	if resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("access denied (403)")
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error: %d", resp.StatusCode)
	}
	return nil
}

func (c *HTTPClient) ListPolicies(ctx context.Context, page, pageSize int) ([]AHQPolicy, int, error) {
	if pageSize == 0 {
		pageSize = 25
	}
	url := fmt.Sprintf("%s/api/v1/policies?page=%d&per_page=%d", c.baseURL, page+1, pageSize) // API is 1-indexed

	body, meta, err := c.doGet(ctx, url)
	if err != nil {
		return nil, 0, err
	}

	var policies []AHQPolicy
	if err := json.Unmarshal(body, &policies); err != nil {
		return nil, 0, fmt.Errorf("parse policies: %w", err)
	}

	total := len(policies)
	if meta != nil {
		total = meta.Total
	}
	return policies, total, nil
}

func (c *HTTPClient) GetPolicy(ctx context.Context, id string) (*AHQPolicy, error) {
	url := fmt.Sprintf("%s/api/v1/policies/%s", c.baseURL, id)

	body, _, err := c.doGet(ctx, url)
	if err != nil {
		return nil, err
	}

	var policy AHQPolicy
	if err := json.Unmarshal(body, &policy); err != nil {
		return nil, fmt.Errorf("parse policy: %w", err)
	}
	return &policy, nil
}

func (c *HTTPClient) CreatePolicy(ctx context.Context, policy *AHQPolicy) (string, error) {
	url := fmt.Sprintf("%s/api/v1/policies", c.baseURL)

	data, err := json.Marshal(policy)
	if err != nil {
		return "", fmt.Errorf("marshal policy: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(data)))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("create policy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create policy failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("parse create response: %w", err)
	}
	return result.Data.ID, nil
}

func (c *HTTPClient) UpdatePolicy(ctx context.Context, id string, policy *AHQPolicy) error {
	url := fmt.Sprintf("%s/api/v1/policies/%s", c.baseURL, id)

	data, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("marshal policy: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, strings.NewReader(string(data)))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("update policy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update policy failed (%d): %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func (c *HTTPClient) DeletePolicy(ctx context.Context, id string) error {
	url := fmt.Sprintf("%s/api/v1/policies/%s", c.baseURL, id)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete policy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete policy failed (%d): %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func (c *HTTPClient) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "grctool/1.0")
}

func (c *HTTPClient) doGet(ctx context.Context, url string) (json.RawMessage, *apiMeta, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("build request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil, fmt.Errorf("not found (404)")
	}
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read response: %w", err)
	}

	// Try envelope format first
	var envelope apiResponse
	if err := json.Unmarshal(body, &envelope); err == nil && len(envelope.Data) > 0 {
		return envelope.Data, envelope.Meta, nil
	}

	// Fall back to raw body (no envelope)
	return body, nil, nil
}
