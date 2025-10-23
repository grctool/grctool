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

package tugboat

import (
	"context"
	"fmt"

	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/tugboat/models"
)

// PolicyListOptions represents options for listing policies
type PolicyListOptions struct {
	Framework string `json:"framework,omitempty"` // Filter by framework (SOC2, ISO27001, etc.)
	Status    string `json:"status,omitempty"`    // Filter by status
	Page      int    `json:"page,omitempty"`      // Page number for pagination
	PageSize  int    `json:"page_size,omitempty"` // Items per page (using page_size like evidence API)
	Org       string `json:"org,omitempty"`       // Organization ID
}

// PolicyListResponse represents the response for listing policies
type PolicyListResponse struct {
	MaxPageSize int             `json:"max_page_size"`
	PageSize    int             `json:"page_size"`
	NumPages    int             `json:"num_pages"`
	PageNumber  int             `json:"page_number"`
	Count       int             `json:"count"`
	Next        *int            `json:"next"`     // Can be null or integer
	Previous    *int            `json:"previous"` // Can be null or integer
	Results     []models.Policy `json:"results"`  // The actual policies are in 'results', not 'policies'
}

// GetPolicies retrieves all policies from Tugboat Logic
func (c *Client) GetPolicies(ctx context.Context, opts *PolicyListOptions) ([]models.Policy, error) {
	endpoint := "/api/policy/"

	// Build query parameters - matching the exact working endpoint
	if opts != nil {
		params := make([]string, 0)

		// Add embeds (exactly as specified in the working endpoint)
		params = append(params, "embeds=tags")
		params = append(params, "embeds=assignees")
		params = append(params, "embeds=current_version_status")
		params = append(params, "embeds=read_time")
		params = append(params, "embeds=reviewers")

		// Add organization parameter
		if opts.Org != "" {
			params = append(params, "org="+opts.Org)
		}

		// Add pagination
		if opts.Page > 0 {
			params = append(params, fmt.Sprintf("page=%d", opts.Page))
		} else {
			params = append(params, "page=1")
		}

		if opts.PageSize > 0 {
			params = append(params, fmt.Sprintf("page_size=%d", opts.PageSize))
		} else {
			params = append(params, "page_size=25")
		}

		// Add framework filter
		if opts.Framework != "" {
			params = append(params, "framework="+opts.Framework)
		}

		// Add status filter
		if opts.Status != "" {
			params = append(params, "status="+opts.Status)
		}

		// Build query string
		if len(params) > 0 {
			endpoint += "?"
			for i, param := range params {
				if i > 0 {
					endpoint += "&"
				}
				endpoint += param
			}
		}
	}

	var response PolicyListResponse
	if err := c.get(ctx, endpoint, &response); err != nil {
		return nil, fmt.Errorf("failed to get policies: %w", err)
	}

	// Log policy response for debugging
	c.logger.Debug("Parsed policy response",
		logger.Int("count", response.Count),
		logger.Int("results_length", len(response.Results)))
	if len(response.Results) > 0 {
		c.logger.Debug("First policy details",
			logger.String("id", response.Results[0].ID.String()),
			logger.String("name", response.Results[0].Name))
	}

	return response.Results, nil
}

// GetAllPolicies retrieves all policies by handling pagination automatically
func (c *Client) GetAllPolicies(ctx context.Context, org string, framework string) ([]models.Policy, error) {
	var allPolicies []models.Policy
	page := 1
	pageSize := 25 // Use consistent page size

	for {
		opts := &PolicyListOptions{
			Org:       org,
			Framework: framework,
			Page:      page,
			PageSize:  pageSize,
		}

		policies, err := c.GetPolicies(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to get policies page %d: %w", page, err)
		}

		allPolicies = append(allPolicies, policies...)

		// If we got fewer policies than requested, we've reached the end
		if len(policies) < pageSize {
			break
		}

		page++
	}

	return allPolicies, nil
}

// GetPolicyDetails retrieves a specific policy by ID with full details including embeds
func (c *Client) GetPolicyDetails(ctx context.Context, policyID string) (*models.PolicyDetails, error) {
	endpoint := fmt.Sprintf("/api/policy/%s/?embeds=current_version&embeds=tags&embeds=association_counts&embeds=latest_version&embeds=usage&embeds=deprecation_notes&embeds=assignees&embeds=reviewers", policyID)

	var policyDetails models.PolicyDetails
	if err := c.get(ctx, endpoint, &policyDetails); err != nil {
		return nil, fmt.Errorf("failed to get policy details %s: %w", policyID, err)
	}

	return &policyDetails, nil
}
