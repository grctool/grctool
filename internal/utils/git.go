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

// GetCurrentBranch retrieves the current git branch name
// Returns the branch name, or "HEAD" if in detached HEAD state
func GetCurrentBranch(repoPath string) (string, error) {
	if repoPath == "" {
		return "", fmt.Errorf("repository path is empty")
	}

	// Execute git rev-parse --abbrev-ref HEAD
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git command failed: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("failed to execute git command: %w", err)
	}

	branch := strings.TrimSpace(string(output))
	if branch == "" {
		return "", fmt.Errorf("git returned empty branch name")
	}

	return branch, nil
}

// GetTagsPointingAtHEAD retrieves all tags pointing to the current commit
// Returns an empty slice if no tags point to HEAD
func GetTagsPointingAtHEAD(repoPath string) ([]string, error) {
	if repoPath == "" {
		return nil, fmt.Errorf("repository path is empty")
	}

	// Execute git tag --points-at HEAD
	cmd := exec.Command("git", "tag", "--points-at", "HEAD")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("git command failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to execute git command: %w", err)
	}

	// Split by newlines and filter empty strings
	tags := []string{}
	for _, tag := range strings.Split(string(output), "\n") {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			tags = append(tags, tag)
		}
	}

	return tags, nil
}

// GetRemoteURL retrieves the URL of the origin remote
// Returns the sanitized remote URL (credentials removed if present)
func GetRemoteURL(repoPath string) (string, error) {
	if repoPath == "" {
		return "", fmt.Errorf("repository path is empty")
	}

	// Execute git remote get-url origin
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		// Remote might not exist, that's ok - return empty string
		return "", nil
	}

	url := strings.TrimSpace(string(output))

	// Sanitize URL to remove credentials
	url = sanitizeGitURL(url)

	return url, nil
}

// sanitizeGitURL removes credentials from git URLs
// Supports both HTTPS (https://user:pass@host/repo) and SSH (git@host:repo) formats
func sanitizeGitURL(url string) string {
	// Remove credentials from HTTPS URLs
	// Pattern: https://user:pass@github.com/... -> https://github.com/...
	if strings.Contains(url, "@") && strings.HasPrefix(url, "http") {
		parts := strings.SplitN(url, "@", 2)
		if len(parts) == 2 {
			// Extract protocol
			protocolEnd := strings.Index(parts[0], "://")
			if protocolEnd >= 0 {
				protocol := parts[0][:protocolEnd+3]
				url = protocol + parts[1]
			}
		}
	}

	return url
}

// IsGitRepository checks if the given path is inside a git repository
func IsGitRepository(repoPath string) bool {
	if repoPath == "" {
		return false
	}

	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = repoPath

	err := cmd.Run()
	return err == nil
}
