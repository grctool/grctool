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

package utils

import (
	"os"
	"testing"
)

func TestGetCommitHash(t *testing.T) {
	// Test with current directory (should be a git repo)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Navigate up to the project root (which should be a git repo)
	projectRoot := cwd + "/../.."

	t.Run("Valid git repository", func(t *testing.T) {
		hash, err := GetCommitHash(projectRoot)
		if err != nil {
			t.Skipf("Skipping test - not in git repository or git not available: %v", err)
		}

		if hash == "" {
			t.Error("Expected non-empty git hash")
		}

		// Git hash should be 40 characters (full SHA)
		if len(hash) != 40 {
			t.Errorf("Expected git hash length 40, got %d: %s", len(hash), hash)
		}
	})

	t.Run("Empty path", func(t *testing.T) {
		_, err := GetCommitHash("")
		if err == nil {
			t.Error("Expected error for empty path")
		}
	})

	t.Run("Invalid path", func(t *testing.T) {
		_, err := GetCommitHash("/nonexistent/path")
		if err == nil {
			t.Error("Expected error for invalid path")
		}
	})
}

func TestGetShortCommitHash(t *testing.T) {
	// Test with current directory (should be a git repo)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Navigate up to the project root (which should be a git repo)
	projectRoot := cwd + "/../.."

	t.Run("Valid git repository", func(t *testing.T) {
		hash, err := GetShortCommitHash(projectRoot)
		if err != nil {
			t.Skipf("Skipping test - not in git repository or git not available: %v", err)
		}

		if hash == "" {
			t.Error("Expected non-empty git hash")
		}

		// Short git hash should be 7 characters
		if len(hash) != 7 {
			t.Errorf("Expected short git hash length 7, got %d: %s", len(hash), hash)
		}
	})

	t.Run("Empty path", func(t *testing.T) {
		_, err := GetShortCommitHash("")
		if err == nil {
			t.Error("Expected error for empty path")
		}
	})

	t.Run("Invalid path", func(t *testing.T) {
		_, err := GetShortCommitHash("/nonexistent/path")
		if err == nil {
			t.Error("Expected error for invalid path")
		}
	})
}
