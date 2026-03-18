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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	// Policy initial state is "draft". Trying to transition directly to "published"
	// should fail since draft -> published is not a valid transition.
	buf := new(bytes.Buffer)
	cmd := lifecycleTransitionCmd
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := runLifecycleTransition(cmd, []string{"policy", "POL-0001", "published"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot transition")
	assert.Contains(t, err.Error(), "draft")
	assert.Contains(t, err.Error(), "published")
}

func TestLifecycleTransitionCmd_ValidTransition(t *testing.T) {
	// Policy: draft -> review is valid.
	buf := new(bytes.Buffer)
	cmd := lifecycleTransitionCmd
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := runLifecycleTransition(cmd, []string{"policy", "POL-0001", "review"})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Transitioning")
	assert.Contains(t, buf.String(), "draft")
	assert.Contains(t, buf.String(), "review")
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
