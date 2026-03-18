// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"testing"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newAuthTestConfig() *config.Config {
	return &config.Config{
		Auth: config.AuthConfig{
			GitHub: config.GitHubAuthConfig{
				Token: "ghp_testtoken123",
			},
			CacheDir: "/tmp/test-auth-cache",
		},
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				GitHub: config.GitHubToolConfig{
					APIToken: "ghp_fallback456",
				},
			},
		},
		Tugboat: config.TugboatConfig{
			BearerToken: "tb_testtoken789",
			BaseURL:     "https://test.tugboatlogic.com",
		},
	}
}

func TestNewSharedAuthProviders_GitHub(t *testing.T) {
	t.Parallel()
	log, err := logger.NewTestLogger()
	require.NoError(t, err)
	cfg := newAuthTestConfig()

	providers := NewSharedAuthProviders(cfg, log)

	require.NotNil(t, providers)
	require.NotNil(t, providers.GitHub)
	assert.Equal(t, "github", providers.GitHub.Name())
	assert.True(t, providers.GitHub.IsAuthRequired())

	// Token is present (we passed a non-empty token)
	status := providers.GitHub.GetStatus(context.Background())
	assert.True(t, status.TokenPresent, "GitHub token should be present")
}

func TestNewSharedAuthProviders_Tugboat(t *testing.T) {
	t.Parallel()
	log, err := logger.NewTestLogger()
	require.NoError(t, err)
	cfg := newAuthTestConfig()

	providers := NewSharedAuthProviders(cfg, log)

	require.NotNil(t, providers.Tugboat)
	assert.Equal(t, "tugboat", providers.Tugboat.Name())
}

func TestNewSharedAuthProviders_GitHubFallbackToken(t *testing.T) {
	t.Parallel()
	log, err := logger.NewTestLogger()
	require.NoError(t, err)

	// Auth.GitHub.Token is empty, falls back to Evidence.Tools.GitHub.APIToken
	cfg := newAuthTestConfig()
	cfg.Auth.GitHub.Token = ""

	providers := NewSharedAuthProviders(cfg, log)

	require.NotNil(t, providers.GitHub)
	status := providers.GitHub.GetStatus(context.Background())
	assert.True(t, status.TokenPresent, "should use fallback token from tools config")
}

func TestNewSharedAuthProviders_NoGitHubToken(t *testing.T) {
	t.Parallel()
	log, err := logger.NewTestLogger()
	require.NoError(t, err)

	cfg := newAuthTestConfig()
	cfg.Auth.GitHub.Token = ""
	cfg.Evidence.Tools.GitHub.APIToken = ""

	providers := NewSharedAuthProviders(cfg, log)

	require.NotNil(t, providers.GitHub)
	// Provider exists but token is empty
	status := providers.GitHub.GetStatus(context.Background())
	assert.False(t, status.TokenPresent, "no token available")
}

func TestGetSharedAuthProviders_BeforeInit(t *testing.T) {
	// Save and restore the global
	saved := sharedAuth
	sharedAuth = nil
	defer func() { sharedAuth = saved }()

	assert.Nil(t, GetSharedAuthProviders(), "should be nil before initialization")
}

func TestGetSharedAuthProviders_AfterInit(t *testing.T) {
	// Save and restore the global
	saved := sharedAuth
	defer func() { sharedAuth = saved }()

	log, err := logger.NewTestLogger()
	require.NoError(t, err)

	sharedAuth = NewSharedAuthProviders(newAuthTestConfig(), log)
	got := GetSharedAuthProviders()
	require.NotNil(t, got)
	assert.NotNil(t, got.GitHub)
	assert.NotNil(t, got.Tugboat)
}
