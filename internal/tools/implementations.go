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

package tools

import (
	// TEMP: context commented out due to GitHub tools being disabled
	// "context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/grctool/grctool/internal/models"
)

// Concrete implementations of the interfaces

// OSFileReader implements FileReader using the OS file system
type OSFileReader struct{}

// NewOSFileReader creates a new OS file reader
func NewOSFileReader() FileReader {
	return &OSFileReader{}
}

// Open opens a file for reading
func (r *OSFileReader) Open(path string) (io.ReadCloser, error) {
	return os.Open(path)
}

// Walk walks a directory tree
func (r *OSFileReader) Walk(root string, walkFn func(path string, info FileInfo, err error) error) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		var fileInfo FileInfo
		if info != nil {
			fileInfo = &OSFileInfo{info: info}
		}
		return walkFn(path, fileInfo, err)
	})
}

// Glob returns file paths matching a pattern
func (r *OSFileReader) Glob(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}

// OSFileInfo implements FileInfo using os.FileInfo
type OSFileInfo struct {
	info os.FileInfo
}

// Name returns the file name
func (f *OSFileInfo) Name() string {
	return f.info.Name()
}

// Size returns the file size
func (f *OSFileInfo) Size() int64 {
	return f.info.Size()
}

// IsDir returns whether this is a directory
func (f *OSFileInfo) IsDir() bool {
	return f.info.IsDir()
}

// ModTime returns the modification time
func (f *OSFileInfo) ModTime() interface{} {
	return f.info.ModTime()
}

// DefaultTerraformFileAnalyzer implements TerraformFileAnalyzer
type DefaultTerraformFileAnalyzer struct{}

// NewTerraformFileAnalyzer creates a new Terraform file analyzer
func NewTerraformFileAnalyzer() TerraformFileAnalyzer {
	return &DefaultTerraformFileAnalyzer{}
}

// ScanFile scans a Terraform file for resources
func (a *DefaultTerraformFileAnalyzer) ScanFile(filePath string, content io.Reader, resourceTypes []string) ([]models.TerraformScanResult, error) {
	return ParseTerraformContent(content, filePath, resourceTypes)
}

// MatchesPatterns checks if a file path matches any of the given patterns
func (a *DefaultTerraformFileAnalyzer) MatchesPatterns(filePath string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, filepath.Base(filePath))
		if err != nil {
			continue
		}
		if matched {
			return true
		}

		// Also check if the full path matches (for patterns with directories)
		if contains(pattern, "/") {
			matched, err = filepath.Match(pattern, filePath)
			if err == nil && matched {
				return true
			}
		}
	}
	return false
}

// HTTPGitHubAPIClient wraps the existing GitHubAPIClient to implement the interface
// TEMP: Commented out due to interface mismatch - GitHub tools are disabled anyway
/*
type HTTPGitHubAPIClient struct {
	client *GitHubAPIClient
}

// NewHTTPGitHubAPIClient creates a new HTTP GitHub API client
func NewHTTPGitHubAPIClient(client *GitHubAPIClient) GitHubAPIClient {
	return &HTTPGitHubAPIClient{client: client}
}
*/

// TEMP: Commented out GitHub API client methods - GitHub tools are disabled
/*
// GetRepositoryCollaborators gets all repository collaborators with their permissions
func (c *HTTPGitHubAPIClient) GetRepositoryCollaborators(ctx context.Context, owner, repo string) ([]models.GitHubCollaborator, error) {
	return c.client.GetRepositoryCollaborators(ctx, owner, repo)
}

// GetRepositoryTeams gets all teams with access to the repository
func (c *HTTPGitHubAPIClient) GetRepositoryTeams(ctx context.Context, owner, repo string) ([]models.GitHubTeam, error) {
	return c.client.GetRepositoryTeams(ctx, owner, repo)
}

// GetRepositoryBranches gets all branches in the repository
func (c *HTTPGitHubAPIClient) GetRepositoryBranches(ctx context.Context, owner, repo string) ([]models.GitHubBranch, error) {
	return c.client.GetRepositoryBranches(ctx, owner, repo)
}
*/

/*
// GetDeploymentEnvironments gets all deployment environments
func (c *HTTPGitHubAPIClient) GetDeploymentEnvironments(ctx context.Context, owner, repo string) ([]models.GitHubEnvironment, error) {
	return c.client.GetDeploymentEnvironments(ctx, owner, repo)
}

// GetRepositorySecurity gets repository security settings
func (c *HTTPGitHubAPIClient) GetRepositorySecurity(ctx context.Context, owner, repo string) (*models.GitHubSecuritySettings, error) {
	return c.client.GetRepositorySecurity(ctx, owner, repo)
}

// GetOrganizationMembers gets all organization members
func (c *HTTPGitHubAPIClient) GetOrganizationMembers(ctx context.Context, org string) ([]models.GitHubOrgMember, error) {
	return c.client.GetOrganizationMembers(ctx, org)
}
*/

// Helper functions

// contains checks if a string contains a substring (using strings.Contains)
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
