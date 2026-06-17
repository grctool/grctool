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

package provenance

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/utils"
)

// Service handles provenance capture for evidence collection
type Service struct {
	toolVersion string
}

// NewService creates a new provenance service
func NewService(toolVersion string) *Service {
	return &Service{
		toolVersion: toolVersion,
	}
}

// CaptureProvenance captures the current provenance context for a given path
// If the path is not in a git repository, it returns partial provenance with only basic info
func (s *Service) CaptureProvenance(contextPath string) (*models.Provenance, error) {
	if contextPath == "" {
		return nil, fmt.Errorf("context path is empty")
	}

	// Make sure the path is absolute
	absPath, err := filepath.Abs(contextPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	provenance := &models.Provenance{
		CollectedAt: time.Now().UTC(),
		ToolVersion: s.toolVersion,
	}

	// Get hostname
	hostname, err := os.Hostname()
	if err == nil {
		provenance.CollectorHost = hostname
	}

	// Check if we're in a git repository
	if !utils.IsGitRepository(absPath) {
		// Not a git repo - return partial provenance
		return provenance, nil
	}

	// Capture git information
	if err := s.captureGitInfo(absPath, provenance); err != nil {
		// Log error but continue with partial provenance
		// We don't want to fail evidence collection just because git info is unavailable
		return provenance, nil
	}

	return provenance, nil
}

// captureGitInfo captures git-specific provenance information
func (s *Service) captureGitInfo(repoPath string, provenance *models.Provenance) error {
	// Get commit SHA (full hash)
	commitSHA, err := utils.GetCommitHash(repoPath)
	if err != nil {
		return fmt.Errorf("failed to get commit hash: %w", err)
	}
	provenance.GitCommitSHA = commitSHA

	// Get current branch
	branch, err := utils.GetCurrentBranch(repoPath)
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}
	provenance.GitBranch = branch

	// Get tags pointing at HEAD (optional - don't fail if none exist)
	tags, err := utils.GetTagsPointingAtHEAD(repoPath)
	if err == nil && len(tags) > 0 {
		provenance.GitTags = tags
	}

	// Get remote URL (optional - don't fail if not configured)
	remoteURL, err := utils.GetRemoteURL(repoPath)
	if err == nil && remoteURL != "" {
		provenance.GitRemoteURL = remoteURL
	}

	// Store working directory as relative path from git root
	gitRoot, err := s.getGitRoot(repoPath)
	if err == nil {
		relPath, err := filepath.Rel(gitRoot, repoPath)
		if err == nil {
			provenance.WorkingDir = relPath
		}
	}

	return nil
}

// getGitRoot finds the git repository root for a given path
func (s *Service) getGitRoot(repoPath string) (string, error) {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return "", err
	}

	// Walk up the directory tree to find .git
	current := absPath
	for {
		gitDir := filepath.Join(current, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached root without finding .git
			return "", fmt.Errorf("not in a git repository")
		}
		current = parent
	}
}

// CaptureForDataDir captures provenance for the GRCTool data directory
// This is useful when the evidence is being collected from non-git sources
func (s *Service) CaptureForDataDir(dataDir string) (*models.Provenance, error) {
	return s.CaptureProvenance(dataDir)
}

// CaptureForSourceFile captures provenance for a specific source file
// This includes the file's git context and working directory
func (s *Service) CaptureForSourceFile(filePath string) (*models.Provenance, error) {
	fileDir := filepath.Dir(filePath)
	return s.CaptureProvenance(fileDir)
}

// ValidateProvenance checks if provenance contains minimal required information
func (s *Service) ValidateProvenance(prov *models.Provenance) error {
	if prov == nil {
		return fmt.Errorf("provenance is nil")
	}

	if prov.CollectedAt.IsZero() {
		return fmt.Errorf("collection timestamp is missing")
	}

	// Git information is optional, so we only validate if it's present
	if prov.GitCommitSHA != "" {
		if len(prov.GitCommitSHA) != 40 {
			return fmt.Errorf("invalid git commit SHA (expected 40 characters, got %d)", len(prov.GitCommitSHA))
		}
	}

	return nil
}
