// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package github

import (
	"testing"

	"github.com/grctool/grctool/internal/auth"
	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testCfg(t *testing.T) *config.Config {
	t.Helper()
	return &config.Config{
		Storage: config.StorageConfig{DataDir: t.TempDir()},
		Auth: config.AuthConfig{
			GitHub:   config.GitHubAuthConfig{Token: "test-token"},
			CacheDir: t.TempDir(),
		},
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				GitHub: config.GitHubToolConfig{Repository: "org/repo"},
			},
		},
	}
}

func TestGetOrCreateClient_NoShared(t *testing.T) {
	ResetSharedClient()
	defer ResetSharedClient()

	log := testhelpers.NewStubLogger()
	cfg := testCfg(t)

	client := GetOrCreateClient(cfg, log)
	require.NotNil(t, client)
	assert.Equal(t, "https://api.github.com", client.baseURL)
	client.Close()
}

func TestGetOrCreateClient_WithShared(t *testing.T) {
	ResetSharedClient()
	defer ResetSharedClient()

	log := testhelpers.NewStubLogger()
	cfg := testCfg(t)

	// Set a shared client
	shared := NewGitHubClient(cfg, log)
	SetSharedClient(shared)

	// GetOrCreateClient should return the shared one
	got := GetOrCreateClient(cfg, log)
	assert.Same(t, shared, got)
	shared.Close()
}

func TestInitSharedClient_UsesInjectedAuth(t *testing.T) {
	ResetSharedClient()
	defer ResetSharedClient()

	log := testhelpers.NewStubLogger()
	cfg := testCfg(t)

	injected := auth.NewGitHubAuthProvider("injected-token", t.TempDir(), log)
	client := InitSharedClient(cfg, log, injected)

	require.NotNil(t, client)
	assert.Same(t, injected, client.authProvider)

	// Second call should be a no-op (returns same client)
	client2 := InitSharedClient(cfg, log, auth.NewGitHubAuthProvider("other", t.TempDir(), log))
	assert.Same(t, client, client2, "second InitSharedClient should be no-op")
	client.Close()
}

func TestSetSharedClient_Overrides(t *testing.T) {
	ResetSharedClient()
	defer ResetSharedClient()

	log := testhelpers.NewStubLogger()
	cfg := testCfg(t)

	c1 := NewGitHubClient(cfg, log)
	c2 := NewGitHubClient(cfg, log)

	SetSharedClient(c1)
	assert.Same(t, c1, GetOrCreateClient(cfg, log))

	SetSharedClient(c2)
	assert.Same(t, c2, GetOrCreateClient(cfg, log))

	c1.Close()
	c2.Close()
}

func TestResetSharedClient(t *testing.T) {
	ResetSharedClient()
	defer ResetSharedClient()

	log := testhelpers.NewStubLogger()
	cfg := testCfg(t)

	SetSharedClient(NewGitHubClient(cfg, log))
	assert.NotNil(t, sharedClient)

	ResetSharedClient()
	assert.Nil(t, sharedClient)
}
