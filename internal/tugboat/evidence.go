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
	"strconv"

	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/tugboat/models"
)

// EvidenceTaskListOptions represents options for listing evidence tasks
type EvidenceTaskListOptions struct {
	Framework  string   `json:"framework,omitempty"`   // Filter by framework (SOC2, ISO27001, etc.)
	Status     []string `json:"status,omitempty"`      // Filter by status
	Priority   []string `json:"priority,omitempty"`    // Filter by priority
	AssignedTo string   `json:"assigned_to,omitempty"` // Filter by assignee
	Page       int      `json:"page,omitempty"`        // Page number for pagination
	PageSize   int      `json:"page_size,omitempty"`   // Items per page
	Org        string   `json:"org,omitempty"`         // Organization ID
	Embeds     []string `json:"embeds,omitempty"`      // Include related data
	Ordering   string   `json:"ordering,omitempty"`    // Sort order
}

// EvidenceTaskListResponse represents the response structure from Tugboat Logic API
type EvidenceTaskListResponse struct {
	MaxPageSize int                   `json:"max_page_size"`
	PageSize    int                   `json:"page_size"`
	NumPages    int                   `json:"num_pages"`
	PageNumber  int                   `json:"page_number"`
	Count       int                   `json:"count"`
	Next        *int                  `json:"next"`     // Can be null or integer
	Previous    *int                  `json:"previous"` // Can be null or integer
	Results     []models.EvidenceTask `json:"results"`
}

// GetEvidenceTasks retrieves evidence tasks from Tugboat Logic with options
func (c *Client) GetEvidenceTasks(ctx context.Context, opts *EvidenceTaskListOptions) ([]models.EvidenceTask, error) {
	endpoint := "/api/org_evidence/"

	// Build query parameters
	if opts != nil {
		params := make([]string, 0)

		// Start with just org and basic pagination
		if opts.Org != "" {
			params = append(params, "org="+opts.Org)
		}

		// Add pagination
		if opts.Page > 0 {
			params = append(params, "page="+strconv.Itoa(opts.Page))
		} else {
			params = append(params, "page=1")
		}

		if opts.PageSize > 0 {
			params = append(params, "page_size="+strconv.Itoa(opts.PageSize))
		} else {
			params = append(params, "page_size=25")
		}

		// Add embeds if specified
		if len(opts.Embeds) > 0 {
			for _, embed := range opts.Embeds {
				params = append(params, "embeds="+embed)
			}
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

	var response EvidenceTaskListResponse
	if err := c.get(ctx, endpoint, &response); err != nil {
		return nil, fmt.Errorf("failed to get evidence tasks: %w", err)
	}

	// Log evidence response for debugging
	c.logger.Debug("Parsed evidence response",
		logger.Int("count", response.Count),
		logger.Int("results_length", len(response.Results)))

	return response.Results, nil
}

// GetAllEvidenceTasks retrieves all evidence tasks by handling pagination automatically
func (c *Client) GetAllEvidenceTasks(ctx context.Context, org string, framework string) ([]models.EvidenceTask, error) {
	var allTasks []models.EvidenceTask
	page := 1
	pageSize := 25 // Use the same page size as the original URL

	for {
		opts := &EvidenceTaskListOptions{
			Org:       org,
			Framework: framework,
			Page:      page,
			PageSize:  pageSize,
			// Include embeds to get assignee details and other important fields
			Embeds: []string{
				"assignees",
				"tags",
				"aec_status",
			},
		}

		tasks, err := c.GetEvidenceTasks(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to get evidence tasks page %d: %w", page, err)
		}

		allTasks = append(allTasks, tasks...)

		// If we got fewer tasks than requested, we've reached the end
		if len(tasks) < pageSize {
			break
		}

		page++
	}

	return allTasks, nil
}

// GetEvidenceTaskDetails retrieves a specific evidence task by ID with full details including embeds
func (c *Client) GetEvidenceTaskDetails(ctx context.Context, taskID string) (*models.EvidenceTaskDetails, error) {
	endpoint := fmt.Sprintf("/api/org_evidence/%s/?embeds=master_content&embeds=org_scope&embeds=tags&embeds=open_incident_count&embeds=jira_issues&embeds=assignees&embeds=supported_integrations&embeds=aec_status&embeds=subtask_metadata&embeds=last_reminded_at&embeds=supported_internal_aec&embeds=org_controls", taskID)

	var taskDetails models.EvidenceTaskDetails
	if err := c.get(ctx, endpoint, &taskDetails); err != nil {
		return nil, fmt.Errorf("failed to get evidence task details %s: %w", taskID, err)
	}

	return &taskDetails, nil
}

// GetEvidenceTasksByControl retrieves evidence tasks filtered by control ID
func (c *Client) GetEvidenceTasksByControl(ctx context.Context, controlID string, org string) ([]models.EvidenceTask, error) {
	endpoint := fmt.Sprintf("/api/org_evidence/?control_id=%s&org=%s&embeds=org_controls", controlID, org)

	var response EvidenceTaskListResponse
	if err := c.get(ctx, endpoint, &response); err != nil {
		return nil, fmt.Errorf("failed to get evidence tasks for control %s: %w", controlID, err)
	}

	return response.Results, nil
}
