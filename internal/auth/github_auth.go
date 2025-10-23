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
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/logger"
)

// GitHubAuthProvider handles GitHub Personal Access Token authentication
type GitHubAuthProvider struct {
	token     string
	cacheDir  string
	logger    logger.Logger
	status    *AuthStatus
	cacheFile string
}

// GitHubTokenCache represents cached GitHub token information
type GitHubTokenCache struct {
	Token         string     `json:"token"`
	Valid         bool       `json:"valid"`
	LastValidated time.Time  `json:"last_validated"`
	UserLogin     string     `json:"user_login,omitempty"`
	Scopes        []string   `json:"scopes,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
}

// NewGitHubAuthProvider creates a new GitHub authentication provider
func NewGitHubAuthProvider(token string, cacheDir string, log logger.Logger) *GitHubAuthProvider {
	provider := &GitHubAuthProvider{
		token:     strings.TrimSpace(token),
		cacheDir:  cacheDir,
		logger:    log.WithComponent("github-auth"),
		cacheFile: filepath.Join(cacheDir, "github_auth.json"),
	}

	provider.status = &AuthStatus{
		Provider:      "github",
		TokenPresent:  provider.token != "",
		Authenticated: false,
		TokenValid:    false,
		Source:        "api",
	}

	return provider
}

// Name returns the provider name
func (p *GitHubAuthProvider) Name() string {
	return "github"
}

// IsAuthRequired returns true if authentication is required
func (p *GitHubAuthProvider) IsAuthRequired() bool {
	return true
}

// GetStatus returns the current authentication status
func (p *GitHubAuthProvider) GetStatus(ctx context.Context) *AuthStatus {
	// Update status based on current state
	p.updateStatus()
	return p.status
}

// Authenticate performs GitHub token validation
func (p *GitHubAuthProvider) Authenticate(ctx context.Context) error {
	if p.token == "" {
		p.status.Error = "GitHub Personal Access Token not provided"
		p.status.Authenticated = false
		p.logger.Debug("GitHub authentication failed: no token provided")
		return fmt.Errorf("GitHub Personal Access Token is required")
	}

	// First try to load from cache
	if cached := p.loadFromCache(); cached != nil {
		// Check if cached token is still valid (not older than 1 hour)
		if time.Since(cached.LastValidated) < 1*time.Hour && cached.Valid {
			p.status.Authenticated = true
			p.status.TokenValid = true
			p.status.LastValidated = &cached.LastValidated
			p.status.ExpiresAt = cached.ExpiresAt
			p.status.CacheUsed = true
			p.status.Error = ""
			p.logger.Debug("Using cached GitHub authentication")
			return nil
		}
	}

	// Validate token with GitHub API
	if err := p.validateToken(ctx); err != nil {
		p.status.Error = err.Error()
		p.status.Authenticated = false
		p.status.TokenValid = false
		p.logger.Error("GitHub token validation failed",
			logger.Field{Key: "error", Value: err})
		return err
	}

	p.status.Authenticated = true
	p.status.TokenValid = true
	p.status.Error = ""
	now := time.Now()
	p.status.LastValidated = &now
	p.status.CacheUsed = false

	p.logger.Info("GitHub authentication successful")
	return nil
}

// ValidateAuth validates the current authentication state
func (p *GitHubAuthProvider) ValidateAuth(ctx context.Context) error {
	return p.validateToken(ctx)
}

// ClearAuth clears any cached authentication data
func (p *GitHubAuthProvider) ClearAuth() error {
	if err := os.Remove(p.cacheFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clear GitHub auth cache: %w", err)
	}

	p.status.Authenticated = false
	p.status.TokenValid = false
	p.status.LastValidated = nil
	p.status.CacheUsed = false
	p.status.Error = ""

	p.logger.Debug("GitHub auth cache cleared")
	return nil
}

// validateToken validates the GitHub Personal Access Token
func (p *GitHubAuthProvider) validateToken(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return fmt.Errorf("failed to create GitHub API request: %w", err)
	}

	req.Header.Set("Authorization", "token "+p.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "grctool/1.0.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("GitHub API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("GitHub token is invalid or expired")
	}

	if resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("GitHub token lacks required permissions")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	// Parse user info for caching
	var userInfo struct {
		Login string `json:"login"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		p.logger.Warn("Failed to parse GitHub user info",
			logger.Field{Key: "error", Value: err})
		// Don't fail validation just because we can't parse user info
	}

	// Check token scopes from response headers
	scopes := strings.Split(resp.Header.Get("X-OAuth-Scopes"), ", ")

	// Save to cache
	cache := &GitHubTokenCache{
		Token:         p.token,
		Valid:         true,
		LastValidated: time.Now(),
		UserLogin:     userInfo.Login,
		Scopes:        scopes,
	}

	if err := p.saveToCache(cache); err != nil {
		p.logger.Warn("Failed to cache GitHub auth info",
			logger.Field{Key: "error", Value: err})
		// Don't fail validation just because caching failed
	}

	p.logger.Info("GitHub token validated successfully",
		logger.String("user", userInfo.Login),
		logger.Field{Key: "scopes", Value: scopes})

	return nil
}

// loadFromCache loads cached authentication information
func (p *GitHubAuthProvider) loadFromCache() *GitHubTokenCache {
	if _, err := os.Stat(p.cacheFile); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(p.cacheFile)
	if err != nil {
		p.logger.Debug("Failed to read GitHub auth cache",
			logger.Field{Key: "error", Value: err})
		return nil
	}

	var cache GitHubTokenCache
	if err := json.Unmarshal(data, &cache); err != nil {
		p.logger.Debug("Failed to parse GitHub auth cache",
			logger.Field{Key: "error", Value: err})
		return nil
	}

	// Verify the cached token matches the current token
	if cache.Token != p.token {
		p.logger.Debug("Cached GitHub token doesn't match current token")
		return nil
	}

	return &cache
}

// saveToCache saves authentication information to cache
func (p *GitHubAuthProvider) saveToCache(cache *GitHubTokenCache) error {
	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(p.cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache data: %w", err)
	}

	if err := os.WriteFile(p.cacheFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// updateStatus updates the status based on current state
func (p *GitHubAuthProvider) updateStatus() {
	p.status.TokenPresent = p.token != ""

	// If no token, definitely not authenticated
	if !p.status.TokenPresent {
		p.status.Authenticated = false
		p.status.TokenValid = false
		p.status.Source = "local"
		return
	}

	// Check if we have recent validation
	if p.status.LastValidated != nil {
		validationAge := time.Since(*p.status.LastValidated)
		// Consider token valid if validated within the last hour
		if validationAge < 1*time.Hour && p.status.TokenValid {
			p.status.Authenticated = true
			return
		}
	}

	// If we have a token but no recent validation, status is unknown
	// The tool should call Authenticate() to validate
	if p.status.TokenPresent && p.status.LastValidated == nil {
		p.status.Authenticated = false
		p.status.Source = "api"
	}
}
