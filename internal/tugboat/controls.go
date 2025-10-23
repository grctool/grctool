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
	"encoding/json"
	"fmt"
	"io"

	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/tugboat/models"
)

// ControlListOptions represents options for listing controls
type ControlListOptions struct {
	Framework string   `json:"framework,omitempty"` // Filter by framework (SOC2, ISO27001, etc.)
	Status    string   `json:"status,omitempty"`    // Filter by status
	Page      int      `json:"page,omitempty"`      // Page number for pagination
	PageSize  int      `json:"page_size,omitempty"` // Items per page
	Org       string   `json:"org,omitempty"`       // Organization ID
	Embeds    []string `json:"embeds,omitempty"`    // Include related data
}

// ControlListResponse represents the response structure from Tugboat Logic API
type ControlListResponse struct {
	MaxPageSize int              `json:"max_page_size"`
	PageSize    int              `json:"page_size"`
	NumPages    int              `json:"num_pages"`
	PageNumber  int              `json:"page_number"`
	Count       int              `json:"count"`
	Next        *int             `json:"next"`     // Can be null or integer
	Previous    *int             `json:"previous"` // Can be null or integer
	Results     []models.Control `json:"results"`  // The actual controls are in 'results'
}

// GetControls retrieves controls from Tugboat Logic with options
func (c *Client) GetControls(ctx context.Context, opts *ControlListOptions) ([]models.Control, error) {
	endpoint := "/api/org_control/"

	// Build query parameters - matching the exact working endpoint
	if opts != nil {
		params := make([]string, 0)

		// Add embeds (exactly as specified in the working endpoint)
		params = append(params, "embeds=framework_codes")
		params = append(params, "embeds=org_scope")
		params = append(params, "embeds=assignees")
		params = append(params, "embeds=tags")
		params = append(params, "embeds=audit_projects")
		params = append(params, "embeds=jira_issues")

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

	var response ControlListResponse
	if err := c.get(ctx, endpoint, &response); err != nil {
		return nil, fmt.Errorf("failed to get controls: %w", err)
	}

	// Log control response for debugging
	c.logger.Debug("Parsed control response",
		logger.Int("count", response.Count),
		logger.Int("results_length", len(response.Results)))

	return response.Results, nil
}

// GetAllControls retrieves all controls by handling pagination automatically
func (c *Client) GetAllControls(ctx context.Context, org string, framework string) ([]models.Control, error) {
	var allControls []models.Control
	page := 1
	pageSize := 25 // Use consistent page size

	for {
		opts := &ControlListOptions{
			Org:       org,
			Framework: framework,
			Page:      page,
			PageSize:  pageSize,
		}

		controls, err := c.GetControls(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to get controls page %d: %w", page, err)
		}

		allControls = append(allControls, controls...)

		// If we got fewer controls than requested, we've reached the end
		if len(controls) < pageSize {
			break
		}

		page++
	}

	return allControls, nil
}

// GetControlDetails retrieves a specific control by ID with full details including embeds
func (c *Client) GetControlDetails(ctx context.Context, controlID string) (*models.ControlDetails, error) {
	endpoint := fmt.Sprintf("/api/org_control/%s/?embeds=org_scope&embeds=associations&embeds=audit_projects&embeds=jira_issues&embeds=usage&embeds=master_content&embeds=assignees&embeds=tags&embeds=deprecation_notes&embeds=recommended_evidence_count&embeds=framework_codes&embeds=org_evidence_metrics&embeds=open_incident_count", controlID)

	// Make the HTTP request directly to handle response parsing manually
	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to make request for control details %s: %w", controlID, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body for control details %s: %w", controlID, err)
	}

	// Check for HTTP errors
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Try to parse as a single object first
	var controlDetails models.ControlDetails
	if err := json.Unmarshal(body, &controlDetails); err == nil {
		return &controlDetails, nil
	}

	// If that fails, try to parse as an array (some APIs return arrays even for single objects)
	var controlDetailsArray []models.ControlDetails
	if err := json.Unmarshal(body, &controlDetailsArray); err == nil {
		if len(controlDetailsArray) == 0 {
			return nil, fmt.Errorf("control details %s not found - empty response", controlID)
		}
		return &controlDetailsArray[0], nil
	}

	// If both attempts fail, return an error with some debug info
	return nil, fmt.Errorf("failed to parse control details response for %s - body: %s", controlID, string(body[:min(200, len(body))]))
}

// GetControlDetailsWithEvidenceEmbeds tries to retrieve control details with evidence-related embeds
func (c *Client) GetControlDetailsWithEvidenceEmbeds(ctx context.Context, controlID string) (*models.ControlDetails, error) {
	// Test various evidence-related embed parameters
	endpoint := fmt.Sprintf("/api/org_control/%s/?embeds=org_scope&embeds=associations&embeds=audit_projects&embeds=jira_issues&embeds=usage&embeds=master_content&embeds=assignees&embeds=tags&embeds=deprecation_notes&embeds=recommended_evidence_count&embeds=framework_codes&embeds=org_evidence_metrics&embeds=open_incident_count&embeds=evidence_tasks&embeds=org_evidence&embeds=evidence&embeds=related_evidence", controlID)

	// Make the HTTP request directly to handle response parsing manually
	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to make request for control details with evidence embeds %s: %w", controlID, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body for control details with evidence embeds %s: %w", controlID, err)
	}

	// Check for HTTP errors
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Try to parse as a single object first
	var controlDetails models.ControlDetails
	if err := json.Unmarshal(body, &controlDetails); err == nil {
		return &controlDetails, nil
	}

	// If that fails, try to parse as an array (some APIs return arrays even for single objects)
	var controlDetailsArray []models.ControlDetails
	if err := json.Unmarshal(body, &controlDetailsArray); err == nil {
		if len(controlDetailsArray) == 0 {
			return nil, fmt.Errorf("control details with evidence embeds %s not found - empty response", controlID)
		}
		return &controlDetailsArray[0], nil
	}

	// If both attempts fail, return an error with some debug info
	return nil, fmt.Errorf("failed to parse control details with evidence embeds response for %s - body: %s", controlID, string(body[:min(200, len(body))]))
}
