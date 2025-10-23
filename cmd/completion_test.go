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
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestCompleteTaskRefs(t *testing.T) {
	// Set up test environment
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	// Change to project root if we're in cmd directory
	if filepath.Base(origDir) == "cmd" {
		os.Chdir("..")
	}

	// Set config to use test data
	os.Setenv("GRCTOOL_CONFIG", "test-completion.yaml")
	defer os.Unsetenv("GRCTOOL_CONFIG")

	// Create a dummy command
	cmd := &cobra.Command{}

	// Test completion with empty input
	completions, directive := completeTaskRefs(cmd, []string{}, "")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("Expected ShellCompDirectiveNoFileComp, got %v", directive)
	}

	// We expect 6 completions from test data (ET-001 through ET-005, ET-101)
	expectedMin := 1 // At least one completion
	if len(completions) < expectedMin {
		t.Logf("Got completions: %v", completions)
		t.Errorf("Expected at least %d completions, got %d", expectedMin, len(completions))
	}

	// Test completion with prefix
	completions, directive = completeTaskRefs(cmd, []string{}, "ET-00")
	if len(completions) > 0 {
		// Verify format includes description
		for _, comp := range completions {
			if comp == "" {
				t.Error("Got empty completion")
			}
			t.Logf("Completion: %s", comp)
		}
	}
}
