// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"testing"

	"github.com/grctool/grctool/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// GrctoolRunTool — constructor and metadata
// ---------------------------------------------------------------------------

func TestGrctoolRunTool_NewAndMetadata(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t.TempDir())
	log := testhelpers.NewStubLogger()
	tool := NewGrctoolRunTool(cfg, log)

	require.NotNil(t, tool)
	assert.Equal(t, "grctool-run", tool.Name())
	assert.NotEmpty(t, tool.Description())
	assert.Equal(t, "1.0.0", tool.Version())
	assert.NotEmpty(t, tool.Category())

	def := tool.GetClaudeToolDefinition()
	assert.Equal(t, "grctool-run", def.Name)
}

// ---------------------------------------------------------------------------
// isCommandAllowed
// ---------------------------------------------------------------------------

func TestGrctoolRunTool_IsCommandAllowed(t *testing.T) {
	t.Parallel()

	tool := NewGrctoolRunTool(newTestConfig(t.TempDir()), testhelpers.NewStubLogger())

	assert.True(t, tool.isCommandAllowed("sync"))
	assert.True(t, tool.isCommandAllowed("evidence"))
	assert.True(t, tool.isCommandAllowed("policy"))
	assert.True(t, tool.isCommandAllowed("control"))
	assert.True(t, tool.isCommandAllowed("config"))
	assert.False(t, tool.isCommandAllowed("rm"))
	assert.False(t, tool.isCommandAllowed(""))
}

// ---------------------------------------------------------------------------
// getCommandTimeout
// ---------------------------------------------------------------------------

func TestGrctoolRunTool_GetCommandTimeout(t *testing.T) {
	t.Parallel()

	tool := NewGrctoolRunTool(newTestConfig(t.TempDir()), testhelpers.NewStubLogger())

	assert.Equal(t, 300, tool.getCommandTimeout("sync"))
	assert.Equal(t, 600, tool.getCommandTimeout("evidence"))
	assert.Equal(t, 60, tool.getCommandTimeout("unknown")) // default
}

// ---------------------------------------------------------------------------
// GetAllowedCommands
// ---------------------------------------------------------------------------

func TestGrctoolRunTool_GetAllowedCommands(t *testing.T) {
	t.Parallel()

	tool := NewGrctoolRunTool(newTestConfig(t.TempDir()), testhelpers.NewStubLogger())
	commands := tool.GetAllowedCommands()

	assert.NotEmpty(t, commands)
	assert.Contains(t, commands, "sync")
	assert.Contains(t, commands, "evidence")
}

// ---------------------------------------------------------------------------
// validateArguments
// ---------------------------------------------------------------------------

func TestGrctoolRunTool_ValidateArguments(t *testing.T) {
	t.Parallel()

	tool := NewGrctoolRunTool(newTestConfig(t.TempDir()), testhelpers.NewStubLogger())

	t.Run("allowed flags pass", func(t *testing.T) {
		t.Parallel()
		cmdConfig := tool.allowedCommands["sync"]
		err := tool.validateArguments("sync", []string{"--policies", "--force"}, cmdConfig)
		assert.NoError(t, err)
	})

	t.Run("disallowed flags fail", func(t *testing.T) {
		t.Parallel()
		cmdConfig := tool.allowedCommands["sync"]
		err := tool.validateArguments("sync", []string{"--delete-everything"}, cmdConfig)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "flag not allowed")
	})

	t.Run("non-flag arguments allowed", func(t *testing.T) {
		t.Parallel()
		cmdConfig := tool.allowedCommands["evidence"]
		err := tool.validateArguments("evidence", []string{"list", "--status", "pending"}, cmdConfig)
		assert.NoError(t, err)
	})

	t.Run("flag=value format", func(t *testing.T) {
		t.Parallel()
		cmdConfig := tool.allowedCommands["sync"]
		err := tool.validateArguments("sync", []string{"--output=json"}, cmdConfig)
		assert.NoError(t, err)
	})

	t.Run("empty args allowed", func(t *testing.T) {
		t.Parallel()
		cmdConfig := tool.allowedCommands["config"]
		err := tool.validateArguments("config", []string{}, cmdConfig)
		assert.NoError(t, err)
	})
}

// ---------------------------------------------------------------------------
// checkAuthentication
// ---------------------------------------------------------------------------

func TestGrctoolRunTool_CheckAuthentication(t *testing.T) {
	t.Parallel()

	t.Run("no auth configured", func(t *testing.T) {
		t.Parallel()
		tool := NewGrctoolRunTool(newTestConfig(t.TempDir()), testhelpers.NewStubLogger())
		err := tool.checkAuthentication()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not configured")
	})

	t.Run("cookie header configured", func(t *testing.T) {
		t.Parallel()
		cfg := newTestConfig(t.TempDir())
		cfg.Tugboat.CookieHeader = "session=abc123"
		tool := NewGrctoolRunTool(cfg, testhelpers.NewStubLogger())
		err := tool.checkAuthentication()
		assert.NoError(t, err)
	})

	t.Run("bearer token configured", func(t *testing.T) {
		t.Parallel()
		cfg := newTestConfig(t.TempDir())
		cfg.Tugboat.BearerToken = "tok-xyz"
		tool := NewGrctoolRunTool(cfg, testhelpers.NewStubLogger())
		err := tool.checkAuthentication()
		assert.NoError(t, err)
	})
}

// ---------------------------------------------------------------------------
// getExitCode
// ---------------------------------------------------------------------------

func TestGetExitCode(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 0, getExitCode(nil))
}
