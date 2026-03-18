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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScheduleCmd_HasSubcommands(t *testing.T) {
	subcommands := scheduleCmd.Commands()
	names := make(map[string]bool)
	for _, c := range subcommands {
		names[c.Name()] = true
	}

	assert.True(t, names["list"], "schedule should have 'list' subcommand")
	assert.True(t, names["status"], "schedule should have 'status' subcommand")
	assert.True(t, names["run"], "schedule should have 'run' subcommand")
}

func TestScheduleRunCmd_DryRunFlag(t *testing.T) {
	dryRun := scheduleRunCmd.Flags().Lookup("dry-run")
	require.NotNil(t, dryRun, "--dry-run flag should exist")
	assert.Equal(t, "false", dryRun.DefValue)

	force := scheduleRunCmd.Flags().Lookup("force")
	require.NotNil(t, force, "--force flag should exist")
	assert.Equal(t, "false", force.DefValue)
}

func TestScheduleRunCmd_Args(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "no args is valid",
			args:        []string{},
			expectError: false,
		},
		{
			name:        "one arg is valid",
			args:        []string{"nightly-sync"},
			expectError: false,
		},
		{
			name:        "two args is invalid",
			args:        []string{"nightly-sync", "extra"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := scheduleRunCmd.Args(scheduleRunCmd, tt.args)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestScheduleCmd_ParentUse(t *testing.T) {
	assert.Equal(t, "schedule", scheduleCmd.Use)
	assert.NotEmpty(t, scheduleCmd.Short)
}

func TestScheduleListCmd_Use(t *testing.T) {
	assert.Equal(t, "list", scheduleListCmd.Use)
}

func TestScheduleStatusCmd_Use(t *testing.T) {
	assert.Equal(t, "status", scheduleStatusCmd.Use)
}
