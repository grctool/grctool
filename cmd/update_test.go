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
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int
	}{
		{
			name:     "equal versions",
			v1:       "1.0.0",
			v2:       "1.0.0",
			expected: 0,
		},
		{
			name:     "v1 less than v2",
			v1:       "1.0.0",
			v2:       "1.0.1",
			expected: -1,
		},
		{
			name:     "v1 greater than v2",
			v1:       "1.1.0",
			v2:       "1.0.1",
			expected: 1,
		},
		{
			name:     "major version difference",
			v1:       "2.0.0",
			v2:       "1.9.9",
			expected: 1,
		},
		{
			name:     "with v prefix",
			v1:       "v1.0.0",
			v2:       "v1.0.1",
			expected: -1,
		},
		{
			name:     "different lengths",
			v1:       "1.0",
			v2:       "1.0.0",
			expected: 0,
		},
		{
			name:     "pre-release version",
			v1:       "1.0.0-beta",
			v2:       "1.0.0",
			expected: 0, // Simple comparison ignores pre-release
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareVersions(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("compareVersions(%q, %q) = %d, want %d", tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}

func TestUpdateCommand(t *testing.T) {
	// Test that commands are registered
	if updateCmd == nil {
		t.Fatal("updateCmd is nil")
	}

	if checkCmd == nil {
		t.Fatal("checkCmd is nil")
	}

	if installCmd == nil {
		t.Fatal("installCmd is nil")
	}

	// Verify command structure
	if updateCmd.Use != "update" {
		t.Errorf("updateCmd.Use = %q, want %q", updateCmd.Use, "update")
	}

	// Verify subcommands are added
	var hasCheck, hasInstall bool
	for _, cmd := range updateCmd.Commands() {
		switch cmd.Use {
		case "check":
			hasCheck = true
		case "install":
			hasInstall = true
		}
	}

	if !hasCheck {
		t.Error("update command missing 'check' subcommand")
	}

	if !hasInstall {
		t.Error("update command missing 'install' subcommand")
	}
}

func TestInstallCommandFlags(t *testing.T) {
	// Test that install command has expected flags
	yesFlag := installCmd.Flags().Lookup("yes")
	if yesFlag == nil {
		t.Error("install command missing --yes flag")
	}

	systemFlag := installCmd.Flags().Lookup("system")
	if systemFlag == nil {
		t.Error("install command missing --system flag")
	}
}
