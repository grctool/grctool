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
	"context"
	"io"

	"github.com/grctool/grctool/internal/models"
)

// GitHubAPIProvider defines the interface for GitHub API operations
type GitHubAPIProvider interface {
	GetRepositoryCollaborators(ctx context.Context, owner, repo string) ([]models.GitHubCollaborator, error)
	GetRepositoryTeams(ctx context.Context, owner, repo string) ([]models.GitHubTeam, error)
	GetRepositoryBranches(ctx context.Context, owner, repo string) ([]models.GitHubBranch, error)
	GetDeploymentEnvironments(ctx context.Context, owner, repo string) ([]models.GitHubEnvironment, error)
	GetRepositorySecurity(ctx context.Context, owner, repo string) (*models.GitHubSecuritySettings, error)
	GetOrganizationMembers(ctx context.Context, org string) ([]models.GitHubOrgMember, error)
}

// FileReader defines the interface for file operations
type FileReader interface {
	Open(path string) (io.ReadCloser, error)
	Walk(root string, walkFn func(path string, info FileInfo, err error) error) error
	Glob(pattern string) ([]string, error)
}

// FileInfo represents file information
type FileInfo interface {
	Name() string
	Size() int64
	IsDir() bool
	ModTime() interface{} // Using interface{} to avoid time package dependency
}

// TerraformFileAnalyzer defines the interface for Terraform file analysis operations
type TerraformFileAnalyzer interface {
	ScanFile(filePath string, content io.Reader, resourceTypes []string) ([]models.TerraformScanResult, error)
	MatchesPatterns(filePath string, patterns []string) bool
}
