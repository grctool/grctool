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

// TugboatAuthProvider handles Tugboat Logic Bearer Token authentication
type TugboatAuthProvider struct {
	bearerToken string
	baseURL     string
	cacheDir    string
	logger      logger.Logger
	status      *AuthStatus
	cacheFile   string
}

// TugboatTokenCache represents cached Tugboat token information
type TugboatTokenCache struct {
	BearerToken   string     `json:"bearer_token"`
	Valid         bool       `json:"valid"`
	LastValidated time.Time  `json:"last_validated"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	BaseURL       string     `json:"base_url"`
}

// NewTugboatAuthProvider creates a new Tugboat authentication provider
func NewTugboatAuthProvider(bearerToken string, baseURL string, cacheDir string, log logger.Logger) *TugboatAuthProvider {
	provider := &TugboatAuthProvider{
		bearerToken: strings.TrimSpace(bearerToken),
		baseURL:     strings.TrimSuffix(baseURL, "/"),
		cacheDir:    cacheDir,
		logger:      log.WithComponent("tugboat-auth"),
		cacheFile:   filepath.Join(cacheDir, "tugboat_auth.json"),
	}

	provider.status = &AuthStatus{
		Provider:      "tugboat",
		TokenPresent:  provider.bearerToken != "",
		Authenticated: false,
		TokenValid:    false,
		Source:        "api",
	}

	return provider
}

// Name returns the provider name
func (p *TugboatAuthProvider) Name() string {
	return "tugboat"
}

// IsAuthRequired returns true as Tugboat requires authentication
func (p *TugboatAuthProvider) IsAuthRequired() bool {
	return true
}

// GetStatus returns the current authentication status
func (p *TugboatAuthProvider) GetStatus(ctx context.Context) *AuthStatus {
	p.updateStatus()
	return p.status
}

// Authenticate performs Tugboat bearer token validation
func (p *TugboatAuthProvider) Authenticate(ctx context.Context) error {
	if p.bearerToken == "" {
		p.status.Error = "Tugboat Bearer Token not provided"
		p.status.Authenticated = false
		p.logger.Debug("Tugboat authentication failed: no bearer token provided")
		return fmt.Errorf("tugboat bearer token is required")
	}

	// First try to load from cache
	if cached := p.loadFromCache(); cached != nil {
		// Check if cached token is still valid (not older than 30 minutes)
		if time.Since(cached.LastValidated) < 30*time.Minute && cached.Valid {
			p.status.Authenticated = true
			p.status.TokenValid = true
			p.status.LastValidated = &cached.LastValidated
			p.status.ExpiresAt = cached.ExpiresAt
			p.status.CacheUsed = true
			p.status.Error = ""
			p.logger.Debug("Using cached Tugboat authentication")
			return nil
		}
	}

	// Validate token with Tugboat API
	if err := p.validateToken(ctx); err != nil {
		p.status.Error = err.Error()
		p.status.Authenticated = false
		p.status.TokenValid = false
		p.logger.Error("Tugboat token validation failed",
			logger.Field{Key: "error", Value: err})
		return err
	}

	p.status.Authenticated = true
	p.status.TokenValid = true
	p.status.Error = ""
	now := time.Now()
	p.status.LastValidated = &now
	p.status.CacheUsed = false

	p.logger.Info("Tugboat authentication successful")
	return nil
}

// ValidateAuth validates the current authentication state
func (p *TugboatAuthProvider) ValidateAuth(ctx context.Context) error {
	return p.validateToken(ctx)
}

// ClearAuth clears any cached authentication data
func (p *TugboatAuthProvider) ClearAuth() error {
	if err := os.Remove(p.cacheFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clear Tugboat auth cache: %w", err)
	}

	p.status.Authenticated = false
	p.status.TokenValid = false
	p.status.LastValidated = nil
	p.status.CacheUsed = false
	p.status.Error = ""

	p.logger.Debug("Tugboat auth cache cleared")
	return nil
}

// validateToken validates the Tugboat Bearer Token
func (p *TugboatAuthProvider) validateToken(ctx context.Context) error {
	// Construct the validation URL - try a simple API endpoint
	apiURL := p.baseURL
	if !strings.HasSuffix(apiURL, "/api") {
		apiURL = apiURL + "/api"
	}
	validationURL := apiURL + "/org_policies/?page=1&page_size=1"

	req, err := http.NewRequestWithContext(ctx, "GET", validationURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create Tugboat API request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.bearerToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "grctool/1.0.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("tugboat API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("tugboat bearer token is invalid or expired")
	}

	if resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("tugboat bearer token lacks required permissions")
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("tugboat API returned status %d", resp.StatusCode)
	}

	// Save to cache
	cache := &TugboatTokenCache{
		BearerToken:   p.bearerToken,
		Valid:         true,
		LastValidated: time.Now(),
		BaseURL:       p.baseURL,
	}

	if err := p.saveToCache(cache); err != nil {
		p.logger.Warn("Failed to cache Tugboat auth info",
			logger.Field{Key: "error", Value: err})
		// Don't fail validation just because caching failed
	}

	p.logger.Info("Tugboat bearer token validated successfully")
	return nil
}

// loadFromCache loads cached authentication information
func (p *TugboatAuthProvider) loadFromCache() *TugboatTokenCache {
	if _, err := os.Stat(p.cacheFile); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(p.cacheFile)
	if err != nil {
		p.logger.Debug("Failed to read Tugboat auth cache",
			logger.Field{Key: "error", Value: err})
		return nil
	}

	var cache TugboatTokenCache
	if err := json.Unmarshal(data, &cache); err != nil {
		p.logger.Debug("Failed to parse Tugboat auth cache",
			logger.Field{Key: "error", Value: err})
		return nil
	}

	// Verify the cached token matches the current token and URL
	if cache.BearerToken != p.bearerToken || cache.BaseURL != p.baseURL {
		p.logger.Debug("Cached Tugboat credentials don't match current configuration")
		return nil
	}

	return &cache
}

// saveToCache saves authentication information to cache
func (p *TugboatAuthProvider) saveToCache(cache *TugboatTokenCache) error {
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
func (p *TugboatAuthProvider) updateStatus() {
	p.status.TokenPresent = p.bearerToken != ""

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
		// Consider token valid if validated within the last 30 minutes
		if validationAge < 30*time.Minute && p.status.TokenValid {
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
