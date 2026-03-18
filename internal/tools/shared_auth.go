// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"github.com/grctool/grctool/internal/auth"
	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
)

// SharedAuthProviders holds pre-constructed auth providers for reuse across
// tools. Construct once via NewSharedAuthProviders and pass to tool
// constructors instead of having each tool create its own.
type SharedAuthProviders struct {
	GitHub  auth.AuthProvider
	Tugboat auth.AuthProvider
}

// NewSharedAuthProviders creates auth providers from the application config.
// The GitHub token is resolved from config (which already handles env vars
// and gh CLI fallback). The Tugboat bearer token comes from config.
func NewSharedAuthProviders(cfg *config.Config, log logger.Logger) *SharedAuthProviders {
	providers := &SharedAuthProviders{}

	// GitHub auth — resolve token from config's auth or tools section
	githubToken := cfg.Auth.GitHub.Token
	if githubToken == "" {
		githubToken = cfg.Evidence.Tools.GitHub.APIToken
	}
	providers.GitHub = auth.NewGitHubAuthProvider(githubToken, cfg.Auth.CacheDir, log)

	// Tugboat auth — resolve bearer token from config
	bearerToken := cfg.Tugboat.BearerToken
	providers.Tugboat = auth.NewTugboatAuthProvider(bearerToken, cfg.Tugboat.BaseURL, cfg.Auth.CacheDir, log)

	return providers
}
