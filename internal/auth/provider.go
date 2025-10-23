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
	"time"
)

// AuthProvider defines the interface for tool-specific authentication providers
type AuthProvider interface {
	// Name returns the provider name (e.g., "github", "tugboat")
	Name() string

	// IsAuthRequired returns true if authentication is required for this provider
	IsAuthRequired() bool

	// GetStatus returns the current authentication status
	GetStatus(ctx context.Context) *AuthStatus

	// Authenticate performs the authentication process
	Authenticate(ctx context.Context) error

	// ValidateAuth validates the current authentication state
	ValidateAuth(ctx context.Context) error

	// ClearAuth clears any cached authentication data
	ClearAuth() error
}

// AuthStatus represents the current authentication state
type AuthStatus struct {
	// Authenticated indicates if the provider is successfully authenticated
	Authenticated bool `json:"authenticated"`

	// Provider is the name of the auth provider
	Provider string `json:"provider"`

	// TokenPresent indicates if a token/credential is configured (but may be invalid)
	TokenPresent bool `json:"token_present"`

	// TokenValid indicates if the token has been validated recently
	TokenValid bool `json:"token_valid"`

	// LastValidated is when the token was last validated
	LastValidated *time.Time `json:"last_validated,omitempty"`

	// ExpiresAt is when the authentication expires, if known
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	// CacheUsed indicates if cached credentials were used
	CacheUsed bool `json:"cache_used"`

	// Error contains any authentication error message
	Error string `json:"error,omitempty"`

	// Source indicates the data source being used (local, cache, api)
	Source string `json:"source"`
}

// NoAuthProvider is a provider for tools that don't require authentication
type NoAuthProvider struct {
	providerName string
	dataSource   string
}

// NewNoAuthProvider creates a provider for offline/local tools
func NewNoAuthProvider(name string, source string) AuthProvider {
	if source == "" {
		source = "local"
	}
	return &NoAuthProvider{
		providerName: name,
		dataSource:   source,
	}
}

// Name returns the provider name
func (p *NoAuthProvider) Name() string {
	return p.providerName
}

// IsAuthRequired always returns false for no-auth providers
func (p *NoAuthProvider) IsAuthRequired() bool {
	return false
}

// GetStatus returns authenticated status for local operations
func (p *NoAuthProvider) GetStatus(ctx context.Context) *AuthStatus {
	return &AuthStatus{
		Authenticated: true,
		Provider:      p.providerName,
		TokenPresent:  false,
		TokenValid:    false,
		CacheUsed:     false,
		Source:        p.dataSource,
	}
}

// Authenticate is a no-op for local providers
func (p *NoAuthProvider) Authenticate(ctx context.Context) error {
	return nil
}

// ValidateAuth is a no-op for local providers
func (p *NoAuthProvider) ValidateAuth(ctx context.Context) error {
	return nil
}

// ClearAuth is a no-op for local providers
func (p *NoAuthProvider) ClearAuth() error {
	return nil
}
