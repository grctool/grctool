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
	"io"
	"os"
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

// GetEvidenceImplementations retrieves all evidence implementations (submissions) for an evidence task
// These are the actual evidence files and data that have been submitted for the task
func (c *Client) GetEvidenceImplementations(ctx context.Context, evidenceTaskID string) ([]models.EvidenceImplementation, error) {
	// Try multiple possible API endpoint patterns based on OneTrust/Tugboat API structure
	// Pattern 1: Direct evidence task implementations endpoint
	endpoint := fmt.Sprintf("/api/org_evidence/%s/implementations/", evidenceTaskID)

	var implementations []models.EvidenceImplementation
	err := c.get(ctx, endpoint, &implementations)
	if err == nil {
		return implementations, nil
	}

	// Pattern 2: Evidence task implementations as embedded data
	endpoint = fmt.Sprintf("/api/org_evidence/%s/?embeds=implementations", evidenceTaskID)

	var taskWithImpls struct {
		Implementations []models.EvidenceImplementation `json:"implementations"`
	}
	err = c.get(ctx, endpoint, &taskWithImpls)
	if err == nil {
		return taskWithImpls.Implementations, nil
	}

	// Pattern 3: Via evidence-task-implementations endpoint (OneTrust GRC API)
	endpoint = fmt.Sprintf("/api/controls/v1/evidence-task-implementations/?evidenceTaskId=%s", evidenceTaskID)

	var implResponse struct {
		Data []models.EvidenceImplementation `json:"data"`
	}
	err = c.get(ctx, endpoint, &implResponse)
	if err == nil {
		return implResponse.Data, nil
	}

	return nil, fmt.Errorf("failed to get evidence implementations for task %s (tried multiple endpoints): %w", evidenceTaskID, err)
}

// GetEvidenceAttachments retrieves all attachments for a specific evidence implementation
func (c *Client) GetEvidenceAttachments(ctx context.Context, implementationID string) ([]models.EvidenceAttachment, error) {
	endpoint := fmt.Sprintf("/api/controls/v1/evidence-task-implementations/%s/attachments/", implementationID)

	var attachments []models.EvidenceAttachment
	if err := c.get(ctx, endpoint, &attachments); err != nil {
		return nil, fmt.Errorf("failed to get attachments for implementation %s: %w", implementationID, err)
	}

	return attachments, nil
}

// DownloadEvidenceAttachment downloads a specific evidence attachment to a file
// Returns the number of bytes downloaded
func (c *Client) DownloadEvidenceAttachment(ctx context.Context, attachmentID string, filepath string) (int64, error) {
	// Try the OneTrust GRC API endpoint pattern first
	endpoint := fmt.Sprintf("/api/controls/v1/attachments/%s/download/", attachmentID)

	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to download attachment %s: %w", attachmentID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("failed to download attachment %s: HTTP %d", attachmentID, resp.StatusCode)
	}

	// Create the output file
	outFile, err := os.Create(filepath)
	if err != nil {
		return 0, fmt.Errorf("failed to create file %s: %w", filepath, err)
	}
	defer outFile.Close()

	// Copy the response body to the file
	bytesWritten, err := io.Copy(outFile, resp.Body)
	if err != nil {
		return bytesWritten, fmt.Errorf("failed to write attachment to file: %w", err)
	}

	return bytesWritten, nil
}

// GetSubmittedEvidenceHistory retrieves the full submission history for an evidence task
// This provides a consolidated view of all evidence implementations and attachments
func (c *Client) GetSubmittedEvidenceHistory(ctx context.Context, taskID string, taskRef string) (*models.EvidenceSubmissionHistory, error) {
	// Get all implementations for this task
	implementations, err := c.GetEvidenceImplementations(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get submission history for task %s: %w", taskRef, err)
	}

	history := &models.EvidenceSubmissionHistory{
		TaskID:          taskID,
		TaskRef:         taskRef,
		Implementations: implementations,
		TotalCount:      len(implementations),
	}

	// Find the most recent submission date
	for _, impl := range implementations {
		if !impl.CreatedAt.IsZero() {
			if history.LastSubmitted == nil {
				createdAt := impl.CreatedAt
				history.LastSubmitted = &createdAt
			} else if impl.CreatedAt.After(history.LastSubmitted.Time) {
				createdAt := impl.CreatedAt
				history.LastSubmitted = &createdAt
			}
		}
	}

	return history, nil
}
