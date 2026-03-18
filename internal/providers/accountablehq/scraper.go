// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package accountablehq

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

// ScraperClient implements AccountableHQClient by scraping the web UI.
// This is the fallback when no REST API is available. It uses standard
// net/http with cookie-based session management.
//
// The actual HTML parsing (colly, chromedp, or goquery) will be added
// once the AccountableHQ web stack is investigated (grct-9qj.1).
// For now, this provides the session management and request scaffolding.
type ScraperClient struct {
	baseURL    string
	httpClient *http.Client
	session    *ScraperSession
}

// ScraperSession tracks login state.
type ScraperSession struct {
	LoggedIn  bool
	ExpiresAt time.Time
	Username  string
}

// ScraperConfig holds configuration for the scraper.
type ScraperConfig struct {
	BaseURL  string
	Username string
	Password string
	Timeout  time.Duration
}

// NewScraperClient creates a scraper-based AccountableHQ client.
func NewScraperClient(cfg ScraperConfig) (*ScraperClient, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &ScraperClient{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
			Jar:     jar,
		},
		session: &ScraperSession{},
	}, nil
}

// Login authenticates with AccountableHQ's web login form.
// This is a placeholder — the actual login flow depends on the web stack.
func (c *ScraperClient) Login(ctx context.Context, username, password string) error {
	// Phase 1: POST to login endpoint with credentials
	// Phase 2: Follow redirects, capture session cookie
	// Phase 3: Verify session by loading a protected page

	loginURL := c.baseURL + "/login"
	form := url.Values{
		"username": {username},
		"password": {password},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", loginURL,
		strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("build login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("login request: %w", err)
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body) // drain body

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("login failed: %d", resp.StatusCode)
	}

	c.session.LoggedIn = true
	c.session.Username = username
	c.session.ExpiresAt = time.Now().Add(1 * time.Hour)
	return nil
}

// IsLoggedIn checks if the session is still valid.
func (c *ScraperClient) IsLoggedIn() bool {
	return c.session.LoggedIn && time.Now().Before(c.session.ExpiresAt)
}

// doGet performs an authenticated GET request.
func (c *ScraperClient) doGet(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/html,application/json")
	return c.httpClient.Do(req)
}
