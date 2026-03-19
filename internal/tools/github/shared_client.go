// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package github

import (
	"sync"

	"github.com/grctool/grctool/internal/auth"
	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
)

var (
	sharedClient     *GitHubClient
	sharedClientOnce sync.Once
)

// SetSharedClient sets the shared GitHub client for all tools in this package.
// Call this during initialization (e.g., from registry_init.go) to inject a
// client with a centrally-managed auth provider. If not called, tools fall
// back to creating their own client via NewGitHubClient.
func SetSharedClient(client *GitHubClient) {
	sharedClient = client
}

// GetOrCreateClient returns the shared client if set, otherwise creates a new one.
// This is the single entry point that all tool constructors should use.
func GetOrCreateClient(cfg *config.Config, log logger.Logger) *GitHubClient {
	if sharedClient != nil {
		return sharedClient
	}
	return NewGitHubClient(cfg, log)
}

// InitSharedClient creates a shared GitHub client with an injected auth provider
// and sets it for reuse. Returns the client. Safe to call multiple times —
// subsequent calls are no-ops.
func InitSharedClient(cfg *config.Config, log logger.Logger, authProvider auth.AuthProvider) *GitHubClient {
	sharedClientOnce.Do(func() {
		sharedClient = NewGitHubClientWithAuth(cfg, log, authProvider)
	})
	return sharedClient
}

// ResetSharedClient clears the shared client (for testing only).
func ResetSharedClient() {
	sharedClient = nil
	sharedClientOnce = sync.Once{}
}
