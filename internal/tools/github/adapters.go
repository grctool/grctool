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

package github

import (
	"context"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/tools/types"
)

// Tool interface adapters for consolidated GitHub tools
// These adapters maintain backward compatibility with the existing Tool interface
// while using the new consolidated GitHub client architecture.

// GitHubAdapter provides the main GitHub tool interface
type GitHubAdapter struct {
	tool types.LegacyTool
}

// NewGitHubAdapter creates a GitHub tool adapter (legacy "github" tool)
func NewGitHubAdapter(cfg *config.Config, log logger.Logger) types.LegacyTool {
	return &GitHubAdapter{
		tool: NewGitHubTool(cfg, log),
	}
}

// Name returns the tool name
func (ga *GitHubAdapter) Name() string {
	return ga.tool.Name()
}

// Description returns the tool description
func (ga *GitHubAdapter) Description() string {
	return ga.tool.Description()
}

// GetClaudeToolDefinition returns the tool definition for Claude
func (ga *GitHubAdapter) GetClaudeToolDefinition() models.ClaudeTool {
	return ga.tool.GetClaudeToolDefinition()
}

// Execute runs the GitHub tool
func (ga *GitHubAdapter) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	return ga.tool.Execute(ctx, params)
}

// GitHubEnhancedAdapter provides the enhanced GitHub search interface
type GitHubEnhancedAdapter struct {
	tool types.LegacyTool
}

// NewGitHubEnhancedAdapter creates an enhanced GitHub search tool adapter
func NewGitHubEnhancedAdapter(cfg *config.Config, log logger.Logger) types.LegacyTool {
	return &GitHubEnhancedAdapter{
		tool: NewGitHubEnhancedTool(cfg, log),
	}
}

// Name returns the tool name
func (gea *GitHubEnhancedAdapter) Name() string {
	return gea.tool.Name()
}

// Description returns the tool description
func (gea *GitHubEnhancedAdapter) Description() string {
	return gea.tool.Description()
}

// GetClaudeToolDefinition returns the tool definition for Claude
func (gea *GitHubEnhancedAdapter) GetClaudeToolDefinition() models.ClaudeTool {
	return gea.tool.GetClaudeToolDefinition()
}

// Execute runs the enhanced GitHub search tool
func (gea *GitHubEnhancedAdapter) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	return gea.tool.Execute(ctx, params)
}

// GitHubPermissionsAdapter provides the permissions analysis interface
type GitHubPermissionsAdapter struct {
	tool types.LegacyTool
}

// NewGitHubPermissionsAdapter creates a permissions analysis tool adapter
func NewGitHubPermissionsAdapter(cfg *config.Config, log logger.Logger) types.LegacyTool {
	return &GitHubPermissionsAdapter{
		tool: NewGitHubPermissionsTool(cfg, log),
	}
}

// Name returns the tool name
func (gpa *GitHubPermissionsAdapter) Name() string {
	return gpa.tool.Name()
}

// Description returns the tool description
func (gpa *GitHubPermissionsAdapter) Description() string {
	return gpa.tool.Description()
}

// GetClaudeToolDefinition returns the tool definition for Claude
func (gpa *GitHubPermissionsAdapter) GetClaudeToolDefinition() models.ClaudeTool {
	return gpa.tool.GetClaudeToolDefinition()
}

// Execute runs the permissions analysis tool
func (gpa *GitHubPermissionsAdapter) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	return gpa.tool.Execute(ctx, params)
}

// GitHubDeploymentAccessAdapter provides the deployment access analysis interface
type GitHubDeploymentAccessAdapter struct {
	tool types.LegacyTool
}

// NewGitHubDeploymentAccessAdapter creates a deployment access analysis tool adapter
func NewGitHubDeploymentAccessAdapter(cfg *config.Config, log logger.Logger) types.LegacyTool {
	return &GitHubDeploymentAccessAdapter{
		tool: NewGitHubDeploymentAccessTool(cfg, log),
	}
}

// Name returns the tool name
func (gdaa *GitHubDeploymentAccessAdapter) Name() string {
	return gdaa.tool.Name()
}

// Description returns the tool description
func (gdaa *GitHubDeploymentAccessAdapter) Description() string {
	return gdaa.tool.Description()
}

// GetClaudeToolDefinition returns the tool definition for Claude
func (gdaa *GitHubDeploymentAccessAdapter) GetClaudeToolDefinition() models.ClaudeTool {
	return gdaa.tool.GetClaudeToolDefinition()
}

// Execute runs the deployment access analysis tool
func (gdaa *GitHubDeploymentAccessAdapter) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	return gdaa.tool.Execute(ctx, params)
}

// GitHubSecurityFeaturesAdapter provides the security features analysis interface
type GitHubSecurityFeaturesAdapter struct {
	tool types.LegacyTool
}

// NewGitHubSecurityFeaturesAdapter creates a security features analysis tool adapter
func NewGitHubSecurityFeaturesAdapter(cfg *config.Config, log logger.Logger) types.LegacyTool {
	return &GitHubSecurityFeaturesAdapter{
		tool: NewGitHubSecurityFeaturesTool(cfg, log),
	}
}

// Name returns the tool name
func (gsfa *GitHubSecurityFeaturesAdapter) Name() string {
	return gsfa.tool.Name()
}

// Description returns the tool description
func (gsfa *GitHubSecurityFeaturesAdapter) Description() string {
	return gsfa.tool.Description()
}

// GetClaudeToolDefinition returns the tool definition for Claude
func (gsfa *GitHubSecurityFeaturesAdapter) GetClaudeToolDefinition() models.ClaudeTool {
	return gsfa.tool.GetClaudeToolDefinition()
}

// Execute runs the security features analysis tool
func (gsfa *GitHubSecurityFeaturesAdapter) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	return gsfa.tool.Execute(ctx, params)
}

// GitHubWorkflowAnalyzerAdapter provides the workflow analysis interface
type GitHubWorkflowAnalyzerAdapter struct {
	tool types.LegacyTool
}

// NewGitHubWorkflowAnalyzerAdapter creates a workflow analysis tool adapter
func NewGitHubWorkflowAnalyzerAdapter(cfg *config.Config, log logger.Logger) types.LegacyTool {
	return &GitHubWorkflowAnalyzerAdapter{
		tool: NewGitHubWorkflowAnalyzer(cfg, log),
	}
}

// Name returns the tool name
func (gwaa *GitHubWorkflowAnalyzerAdapter) Name() string {
	return gwaa.tool.Name()
}

// Description returns the tool description
func (gwaa *GitHubWorkflowAnalyzerAdapter) Description() string {
	return gwaa.tool.Description()
}

// GetClaudeToolDefinition returns the tool definition for Claude
func (gwaa *GitHubWorkflowAnalyzerAdapter) GetClaudeToolDefinition() models.ClaudeTool {
	return gwaa.tool.GetClaudeToolDefinition()
}

// Execute runs the workflow analysis tool
func (gwaa *GitHubWorkflowAnalyzerAdapter) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	return gwaa.tool.Execute(ctx, params)
}

// GitHubReviewAnalyzerAdapter provides the review analysis interface
type GitHubReviewAnalyzerAdapter struct {
	tool types.LegacyTool
}

// NewGitHubReviewAnalyzerAdapter creates a review analysis tool adapter
func NewGitHubReviewAnalyzerAdapter(cfg *config.Config, log logger.Logger) types.LegacyTool {
	return &GitHubReviewAnalyzerAdapter{
		tool: NewGitHubReviewAnalyzer(cfg, log),
	}
}

// Name returns the tool name
func (graa *GitHubReviewAnalyzerAdapter) Name() string {
	return graa.tool.Name()
}

// Description returns the tool description
func (graa *GitHubReviewAnalyzerAdapter) Description() string {
	return graa.tool.Description()
}

// GetClaudeToolDefinition returns the tool definition for Claude
func (graa *GitHubReviewAnalyzerAdapter) GetClaudeToolDefinition() models.ClaudeTool {
	return graa.tool.GetClaudeToolDefinition()
}

// Execute runs the review analysis tool
func (graa *GitHubReviewAnalyzerAdapter) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	return graa.tool.Execute(ctx, params)
}

// Legacy tool name mapping for backward compatibility
var LegacyToolMappings = map[string]func(*config.Config, logger.Logger) types.LegacyTool{
	// Main tools
	"github":          NewGitHubAdapter,
	"github-enhanced": NewGitHubEnhancedAdapter,
	"github_searcher": NewGitHubAdapter, // Legacy alias

	// Permission tools
	"github-permissions":       NewGitHubPermissionsAdapter,
	"github-deployment-access": NewGitHubDeploymentAccessAdapter,
	"github-security-features": NewGitHubSecurityFeaturesAdapter,

	// Analysis tools
	"github-workflow-analyzer": NewGitHubWorkflowAnalyzerAdapter,
	"github-review-analyzer":   NewGitHubReviewAnalyzerAdapter,
}

// GetGitHubTool returns a GitHub tool by name using the legacy mapping
func GetGitHubTool(name string, cfg *config.Config, log logger.Logger) types.LegacyTool {
	if factory, exists := LegacyToolMappings[name]; exists {
		return factory(cfg, log)
	}
	return nil
}

// GetAllGitHubTools returns all available GitHub tools
func GetAllGitHubTools(cfg *config.Config, log logger.Logger) map[string]types.LegacyTool {
	tools := make(map[string]types.LegacyTool)
	for name, factory := range LegacyToolMappings {
		tools[name] = factory(cfg, log)
	}
	return tools
}
