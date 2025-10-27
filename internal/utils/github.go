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

// GetGitHubTokenFromCLI attempts to retrieve a GitHub token from the gh CLI
// Returns the token string if successful, or an error if gh CLI is not authenticated
// or not installed.
//
// This function executes: gh auth token
// Which returns the authentication token for the currently logged in user.
//
// Usage:
//
//	token, err := GetGitHubTokenFromCLI()
//	if err != nil {
//	    // gh CLI not available or not authenticated
//	    return err
//	}
//	// Use token for GitHub API calls
func GetGitHubTokenFromCLI() (string, error) {
	// Execute gh auth token command
	cmd := exec.Command("gh", "auth", "token")

	output, err := cmd.Output()
	if err != nil {
		// Check if it's an exec error with stderr
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "not logged in") || strings.Contains(stderr, "No token") {
				return "", fmt.Errorf("gh CLI is not authenticated. Run: gh auth login")
			}
			return "", fmt.Errorf("gh CLI error: %s", stderr)
		}
		// Likely gh CLI is not installed
		return "", fmt.Errorf("gh CLI not found. Install with: brew install gh (or visit https://cli.github.com)")
	}

	// Trim whitespace and return the token
	token := strings.TrimSpace(string(output))
	if token == "" {
		return "", fmt.Errorf("gh CLI returned empty token")
	}

	return token, nil
}

// IsGitHubCLIAvailable checks if the gh CLI is installed and authenticated
// Returns true if gh CLI can provide a token, false otherwise
func IsGitHubCLIAvailable() bool {
	_, err := GetGitHubTokenFromCLI()
	return err == nil
}
