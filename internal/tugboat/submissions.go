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
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/tugboat/models"
)

// GetEvidenceAttachments retrieves evidence attachments (submissions) from Tugboat Logic with options
func (c *Client) GetEvidenceAttachments(ctx context.Context, opts *models.EvidenceAttachmentListOptions) ([]models.EvidenceAttachment, error) {
	endpoint := "/api/org_evidence_attachment/"

	// Build query parameters
	if opts != nil {
		params := make([]string, 0)

		// Add org_evidence filter (evidence task ID)
		if opts.OrgEvidence > 0 {
			params = append(params, "org_evidence="+strconv.Itoa(opts.OrgEvidence))
		}

		// Add observation period filter
		if opts.ObservationPeriod != "" {
			params = append(params, "observation_period="+opts.ObservationPeriod)
		}

		// Add ordering
		if opts.Ordering != "" {
			params = append(params, "ordering="+opts.Ordering)
		} else {
			// Default: newest first
			params = append(params, "ordering=-collected,-created")
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
		} else {
			// Default embeds to get complete data
			params = append(params, "embeds=attachment")
			params = append(params, "embeds=org_members")
			params = append(params, "embeds=collected_in_window")
			params = append(params, "embeds=org_evidence_subtask")
		}

		// Add type filter
		if opts.Type != "" {
			params = append(params, "type="+opts.Type)
		}

		// Add deleted filter
		if opts.Deleted != nil {
			params = append(params, "deleted="+strconv.FormatBool(*opts.Deleted))
		}

		// Build query string
		if len(params) > 0 {
			endpoint += "?" + strings.Join(params, "&")
		}
	}

	var response models.EvidenceAttachmentListResponse
	if err := c.get(ctx, endpoint, &response); err != nil {
		return nil, fmt.Errorf("failed to get evidence attachments: %w", err)
	}

	// Log response for debugging
	c.logger.Debug("Parsed evidence attachments response",
		logger.Int("count", response.Count),
		logger.Int("results_length", len(response.Results)))

	return response.Results, nil
}

// GetAllEvidenceAttachments retrieves all evidence attachments for a task by handling pagination automatically
func (c *Client) GetAllEvidenceAttachments(ctx context.Context, taskID int, observationPeriod string) ([]models.EvidenceAttachment, error) {
	var allAttachments []models.EvidenceAttachment
	page := 1
	pageSize := 100 // Use larger page size for efficiency (max is 200)

	for {
		opts := &models.EvidenceAttachmentListOptions{
			OrgEvidence:       taskID,
			ObservationPeriod: observationPeriod,
			Ordering:          "-collected,-created", // Newest first
			Page:              page,
			PageSize:          pageSize,
			// Include all relevant embeds
			Embeds: []string{
				"attachment",
				"org_members",
				"collected_in_window",
				"org_evidence_subtask",
			},
		}

		attachments, err := c.GetEvidenceAttachments(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to get evidence attachments page %d: %w", page, err)
		}

		allAttachments = append(allAttachments, attachments...)

		// If we got fewer attachments than requested, we've reached the end
		if len(attachments) < pageSize {
			break
		}

		page++
	}

	return allAttachments, nil
}

// GetEvidenceAttachmentsByTask retrieves all evidence attachments for a specific evidence task
// This is a convenience method that wraps GetAllEvidenceAttachments with a default observation period
func (c *Client) GetEvidenceAttachmentsByTask(ctx context.Context, taskID int) ([]models.EvidenceAttachment, error) {
	// Use a wide date range to get all historical attachments
	// This matches the pattern seen in the Tugboat UI
	observationPeriod := "2013-01-01,2025-12-31"
	return c.GetAllEvidenceAttachments(ctx, taskID, observationPeriod)
}

// GetEvidenceAttachmentsByTaskAndWindow retrieves evidence attachments for a specific task and time window
func (c *Client) GetEvidenceAttachmentsByTaskAndWindow(ctx context.Context, taskID int, startDate, endDate string) ([]models.EvidenceAttachment, error) {
	observationPeriod := fmt.Sprintf("%s,%s", startDate, endDate)
	return c.GetAllEvidenceAttachments(ctx, taskID, observationPeriod)
}

// GetAttachmentDownloadURL retrieves the download URL for a specific attachment
// The endpoint returns a signed S3 URL that is temporary and will expire
func (c *Client) GetAttachmentDownloadURL(ctx context.Context, attachmentID int) (*models.AttachmentDownloadResponse, error) {
	endpoint := fmt.Sprintf("/api/org_evidence_attachment/%d/download/", attachmentID)

	var response models.AttachmentDownloadResponse
	if err := c.get(ctx, endpoint, &response); err != nil {
		return nil, fmt.Errorf("failed to get attachment download URL for ID %d: %w", attachmentID, err)
	}

	return &response, nil
}

// DownloadAttachment downloads an attachment file to the specified destination path
// This method handles the full download workflow:
// 1. Get the signed S3 URL from Tugboat API
// 2. Download the file from S3
// 3. Save to the destination path
func (c *Client) DownloadAttachment(ctx context.Context, attachmentID int, destPath string) error {
	// Get the download URL
	downloadInfo, err := c.GetAttachmentDownloadURL(ctx, attachmentID)
	if err != nil {
		return fmt.Errorf("failed to get download URL: %w", err)
	}

	// Download from the S3 URL
	c.logger.Debug("Downloading attachment from S3",
		logger.Int("attachment_id", attachmentID),
		logger.String("filename", downloadInfo.OriginalFilename),
		logger.String("dest_path", destPath))

	// Create HTTP request to download from S3
	req, err := http.NewRequestWithContext(ctx, "GET", downloadInfo.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}

	// Use standard HTTP client for S3 download (no auth needed, URL is signed)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create destination file
	outFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer outFile.Close()

	// Copy file content
	written, err := io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file content: %w", err)
	}

	c.logger.Debug("Attachment downloaded successfully",
		logger.Int("attachment_id", attachmentID),
		logger.Int("bytes_written", int(written)))

	return nil
}
