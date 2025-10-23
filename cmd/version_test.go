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
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestVersionCommand(t *testing.T) {
	// Set test version information
	SetVersionInfo("1.0.0", "2024-01-01T12:00:00Z", "abc123")

	tests := []struct {
		name     string
		short    bool
		wantText []string
	}{
		{
			name:     "full version output",
			short:    false,
			wantText: []string{"grctool version 1.0.0", "Build Time: 2024-01-01T12:00:00Z", "Git Commit: abc123", "Go Version:", "OS/Arch:"},
		},
		{
			name:     "short version output",
			short:    true,
			wantText: []string{"1.0.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new version command for each test
			testCmd := &cobra.Command{
				Use:   "version",
				Short: "Display version information",
				Run: func(cmd *cobra.Command, args []string) {
					short, _ := cmd.Flags().GetBool("short")

					if short {
						cmd.Println(version)
					} else {
						cmd.Printf("grctool version %s\n", version)
						cmd.Printf("  Build Time: %s\n", buildTime)
						cmd.Printf("  Git Commit: %s\n", gitCommit)
						cmd.Printf("  Go Version: %s\n", runtime.Version())
						cmd.Printf("  OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
					}
				},
			}
			testCmd.Flags().BoolP("short", "s", false, "Print only the version number")

			// Capture output
			buf := new(bytes.Buffer)
			testCmd.SetOut(buf)
			testCmd.SetErr(buf)
			testCmd.SetArgs([]string{})
			if tt.short {
				testCmd.Flags().Set("short", "true")
			}

			// Execute command
			testCmd.Run(testCmd, []string{})

			output := buf.String()

			// Check that all expected text is present
			for _, want := range tt.wantText {
				if !strings.Contains(output, want) {
					t.Errorf("output missing expected text %q\nGot:\n%s", want, output)
				}
			}
		})
	}
}

func TestSetVersionInfo(t *testing.T) {
	// Test that SetVersionInfo properly updates the package variables
	testVersion := "2.0.0"
	testBuildTime := "2024-06-15T10:30:00Z"
	testGitCommit := "def456"

	SetVersionInfo(testVersion, testBuildTime, testGitCommit)

	if version != testVersion {
		t.Errorf("version = %q, want %q", version, testVersion)
	}
	if buildTime != testBuildTime {
		t.Errorf("buildTime = %q, want %q", buildTime, testBuildTime)
	}
	if gitCommit != testGitCommit {
		t.Errorf("gitCommit = %q, want %q", gitCommit, testGitCommit)
	}
}
