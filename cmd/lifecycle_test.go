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

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/storage"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupLifecycleTestConfig creates a temp directory with a minimal config and
// a test policy entity, then configures viper to load from it.
func setupLifecycleTestConfig(t *testing.T) string {
	t.Helper()
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")

	// Write minimal config.
	configContent := `
tugboat:
  base_url: "https://api-test.example.com"
  org_id: "test"
  timeout: "30s"
  rate_limit: 10
  cookie_header: "test-cookie"

storage:
  data_dir: "` + dataDir + `"
`
	configFile := filepath.Join(tempDir, ".grctool.yaml")
	require.NoError(t, os.WriteFile(configFile, []byte(configContent), 0o644))

	viper.Reset()
	viper.SetConfigFile(configFile)
	require.NoError(t, viper.ReadInConfig())

	// Create a test policy entity through the storage layer so filenames are correct.
	store, err := storage.NewStorage(config.StorageConfig{DataDir: dataDir})
	require.NoError(t, err)

	policy := &domain.Policy{
		ID:          "12345",
		ReferenceID: "POL-0001",
		Name:        "Test Policy",
	}
	require.NoError(t, store.SavePolicy(policy))

	return tempDir
}

func TestLifecycleCmd_HasSubcommands(t *testing.T) {
	subcommands := lifecycleCmd.Commands()
	names := make(map[string]bool)
	for _, c := range subcommands {
		names[c.Name()] = true
	}

	assert.True(t, names["status"], "lifecycle should have 'status' subcommand")
	assert.True(t, names["transition"], "lifecycle should have 'transition' subcommand")
}

func TestLifecycleTransitionCmd_Args(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "no args is invalid",
			args:        []string{},
			expectError: true,
		},
		{
			name:        "one arg is invalid",
			args:        []string{"policy"},
			expectError: true,
		},
		{
			name:        "two args is invalid",
			args:        []string{"policy", "POL-0001"},
			expectError: true,
		},
		{
			name:        "three args is valid",
			args:        []string{"policy", "POL-0001", "review"},
			expectError: false,
		},
		{
			name:        "four args is invalid",
			args:        []string{"policy", "POL-0001", "review", "extra"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := lifecycleTransitionCmd.Args(lifecycleTransitionCmd, tt.args)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLifecycleStatusCmd_Args(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "no args is valid (shows all)",
			args:        []string{},
			expectError: false,
		},
		{
			name:        "one arg is valid",
			args:        []string{"policy"},
			expectError: false,
		},
		{
			name:        "two args is invalid",
			args:        []string{"policy", "extra"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := lifecycleStatusCmd.Args(lifecycleStatusCmd, tt.args)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLifecycleTransitionCmd_InvalidEntityType(t *testing.T) {
	// Execute the transition command with an unknown entity type.
	buf := new(bytes.Buffer)
	cmd := lifecycleTransitionCmd
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := runLifecycleTransition(cmd, []string{"unknown_type", "ID-001", "draft"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown entity type")
	assert.Contains(t, err.Error(), "unknown_type")
}

func TestLifecycleTransitionCmd_InvalidTransition(t *testing.T) {
	setupLifecycleTestConfig(t)

	buf := new(bytes.Buffer)
	cmd := lifecycleTransitionCmd
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	// Policy initial state is "draft". Trying to transition directly to "published"
	// should fail since draft -> published is not a valid transition.
	err := runLifecycleTransition(cmd, []string{"policy", "POL-0001", "published"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot transition")
	assert.Contains(t, err.Error(), "draft")
	assert.Contains(t, err.Error(), "published")
}

func TestLifecycleTransitionCmd_ValidTransition(t *testing.T) {
	setupLifecycleTestConfig(t)

	buf := new(bytes.Buffer)
	cmd := lifecycleTransitionCmd
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	// Policy: draft -> review is valid.
	err := runLifecycleTransition(cmd, []string{"policy", "POL-0001", "review"})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Transitioned")
	assert.Contains(t, buf.String(), "draft")
	assert.Contains(t, buf.String(), "review")
}

func TestLifecycleTransitionCmd_PersistsState(t *testing.T) {
	setupLifecycleTestConfig(t)

	buf := new(bytes.Buffer)
	cmd := lifecycleTransitionCmd
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	// Transition draft -> review
	err := runLifecycleTransition(cmd, []string{"policy", "POL-0001", "review"})
	require.NoError(t, err)

	// Now transition review -> approved (should work because state was persisted)
	buf.Reset()
	err = runLifecycleTransition(cmd, []string{"policy", "POL-0001", "approved"})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "review")
	assert.Contains(t, buf.String(), "approved")

	// Transitioning from draft should fail now (we're at approved, not draft)
	buf.Reset()
	err = runLifecycleTransition(cmd, []string{"policy", "POL-0001", "draft"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot transition")
}

func TestLifecycleTransitionCmd_InvalidState(t *testing.T) {
	// "nonexistent" is not a valid state for any entity type.
	buf := new(bytes.Buffer)
	cmd := lifecycleTransitionCmd
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := runLifecycleTransition(cmd, []string{"policy", "POL-0001", "nonexistent"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid state")
}

func TestLifecycleStatusCmd_ValidEntityType(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := lifecycleStatusCmd
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := runLifecycleStatus(cmd, []string{"policy"})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "policy Lifecycle")
	assert.Contains(t, buf.String(), "draft")
}

func TestLifecycleStatusCmd_InvalidEntityType(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := lifecycleStatusCmd
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := runLifecycleStatus(cmd, []string{"bogus"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown entity type")
}

func TestLifecycleStatusCmd_AllEntityTypes(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := lifecycleStatusCmd
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := runLifecycleStatus(cmd, []string{})
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "policy Lifecycle")
	assert.Contains(t, output, "control Lifecycle")
	assert.Contains(t, output, "evidence_task Lifecycle")
}
