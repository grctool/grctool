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
	"time"

	"github.com/grctool/grctool/internal/models"
)

// SubmitEvidence submits evidence content for a task
// NOTE: This is a placeholder implementation until we have confirmed Tugboat API endpoints
func (c *Client) SubmitEvidence(ctx context.Context, orgID string, taskID int, req *models.SubmitEvidenceRequest) (*models.SubmitEvidenceResponse, error) {
	// Construct endpoint
	endpoint := fmt.Sprintf("/api/org_evidence/%d/submissions/?org=%s", taskID, orgID)

	// Make POST request
	var response models.SubmitEvidenceResponse
	if err := c.post(ctx, endpoint, req, &response); err != nil {
		return nil, fmt.Errorf("failed to submit evidence: %w", err)
	}

	return &response, nil
}

// UpdateTaskCompletionStatus marks a task as completed
func (c *Client) UpdateTaskCompletionStatus(ctx context.Context, orgID string, taskID int, completed bool, lastCollected time.Time) error {
	// Construct endpoint
	endpoint := fmt.Sprintf("/api/org_evidence/%d/?org=%s", taskID, orgID)

	// Prepare update payload
	payload := map[string]interface{}{
		"completed":      completed,
		"last_collected": lastCollected.Format(time.RFC3339),
	}

	// Make PATCH request
	if err := c.patch(ctx, endpoint, payload, nil); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	return nil
}

// GetSubmissionStatus retrieves submission status
func (c *Client) GetSubmissionStatus(ctx context.Context, orgID string, taskID int, submissionID string) (*models.SubmissionStatusResponse, error) {
	// Construct endpoint
	endpoint := fmt.Sprintf("/api/org_evidence/%d/submissions/%s/?org=%s", taskID, submissionID, orgID)

	// Make GET request
	var response models.SubmissionStatusResponse
	if err := c.get(ctx, endpoint, &response); err != nil {
		return nil, fmt.Errorf("failed to get submission status: %w", err)
	}

	return &response, nil
}

// ListTaskSubmissions lists all submissions for a task
func (c *Client) ListTaskSubmissions(ctx context.Context, orgID string, taskID int) (*models.SubmissionListResponse, error) {
	// Construct endpoint
	endpoint := fmt.Sprintf("/api/org_evidence/%d/submissions/?org=%s", taskID, orgID)

	// Make GET request
	var response models.SubmissionListResponse
	if err := c.get(ctx, endpoint, &response); err != nil {
		return nil, fmt.Errorf("failed to list submissions: %w", err)
	}

	return &response, nil
}

// UploadEvidenceFile uploads an evidence file
// NOTE: This is a placeholder - actual implementation will need multipart/form-data
func (c *Client) UploadEvidenceFile(ctx context.Context, orgID string, taskID int, filename string, content []byte) (*models.FileUploadResponse, error) {
	// Construct endpoint
	endpoint := fmt.Sprintf("/api/org_evidence/%d/files/?org=%s", taskID, orgID)

	// For now, this is a simple POST
	// Real implementation would use multipart/form-data
	payload := map[string]interface{}{
		"filename": filename,
		"content":  content,
	}

	var response models.FileUploadResponse
	if err := c.post(ctx, endpoint, payload, &response); err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	return &response, nil
}

// DeleteSubmission deletes a submission
func (c *Client) DeleteSubmission(ctx context.Context, orgID string, taskID int, submissionID string) error {
	// Construct endpoint
	endpoint := fmt.Sprintf("/api/org_evidence/%d/submissions/%s/?org=%s", taskID, submissionID, orgID)

	// Make DELETE request
	if err := c.delete(ctx, endpoint); err != nil {
		return fmt.Errorf("failed to delete submission: %w", err)
	}

	return nil
}
