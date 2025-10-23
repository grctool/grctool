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
	"fmt"
	"os/exec"
	"strings"
)

// GetCommitHash retrieves the current git commit hash from a repository
// Returns the full SHA hash of HEAD, or an error if unable to retrieve
func GetCommitHash(repoPath string) (string, error) {
	if repoPath == "" {
		return "", fmt.Errorf("repository path is empty")
	}

	// Execute git rev-parse HEAD in the specified directory
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		// Check if it's a git error
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git command failed: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("failed to execute git command: %w", err)
	}

	// Trim whitespace and return the hash
	hash := strings.TrimSpace(string(output))
	if hash == "" {
		return "", fmt.Errorf("git returned empty hash")
	}

	return hash, nil
}

// GetShortCommitHash retrieves the short version of the current git commit hash
// Returns the first 7 characters of the SHA hash, or an error if unable to retrieve
func GetShortCommitHash(repoPath string) (string, error) {
	if repoPath == "" {
		return "", fmt.Errorf("repository path is empty")
	}

	// Execute git rev-parse --short HEAD in the specified directory
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		// Check if it's a git error
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git command failed: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("failed to execute git command: %w", err)
	}

	// Trim whitespace and return the short hash
	hash := strings.TrimSpace(string(output))
	if hash == "" {
		return "", fmt.Errorf("git returned empty hash")
	}

	return hash, nil
}
