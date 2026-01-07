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
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
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

// URLDownloadResult contains the result of downloading a URL
type URLDownloadResult struct {
	Filename    string // Suggested filename (from Content-Disposition or URL)
	ContentType string // MIME type from Content-Type header
	BytesWritten int64 // Number of bytes written
}

// DownloadFromURL downloads content from an arbitrary URL to the specified directory
// It returns the actual filename used and metadata about the download
// For Tugboat SPA URLs (policies, controls, evidence), it fetches content via API
func (c *Client) DownloadFromURL(ctx context.Context, sourceURL string, destDir string, fallbackName string) (*URLDownloadResult, error) {
	c.logger.Debug("Downloading from URL",
		logger.String("url", sourceURL),
		logger.String("dest_dir", destDir))

	// Check if this is a Tugboat SPA URL that we should handle via API
	if result, err := c.handleTugboatSPAURL(ctx, sourceURL, destDir, fallbackName); result != nil || err != nil {
		return result, err
	}

	// Standard URL download for non-SPA URLs
	return c.downloadDirectURL(ctx, sourceURL, destDir, fallbackName)
}

// handleTugboatSPAURL checks if the URL is a Tugboat SPA URL and fetches content via API
// Returns nil, nil if this is not a Tugboat SPA URL
func (c *Client) handleTugboatSPAURL(ctx context.Context, sourceURL string, destDir string, fallbackName string) (*URLDownloadResult, error) {
	// Parse the URL to extract resource type and ID
	// Patterns: /org/{org_id}/policies/{id}, /org/{org_id}/controls/{id}, /org/{org_id}/evidence/tasks/{id}

	u, err := url.Parse(sourceURL)
	if err != nil {
		return nil, nil // Not a valid URL, let standard handler deal with it
	}

	// Only handle tugboatlogic.com URLs
	if !strings.Contains(u.Host, "tugboatlogic.com") {
		return nil, nil
	}

	// Match URL patterns
	policyPattern := regexp.MustCompile(`/org/\d+/policies/(\d+)`)
	controlPattern := regexp.MustCompile(`/org/\d+/controls/(\d+)`)
	evidencePattern := regexp.MustCompile(`/org/\d+/evidence/tasks/(\d+)`)

	path := u.Path

	if matches := policyPattern.FindStringSubmatch(path); len(matches) == 2 {
		return c.downloadPolicyContent(ctx, matches[1], destDir, fallbackName)
	}

	if matches := controlPattern.FindStringSubmatch(path); len(matches) == 2 {
		return c.downloadControlContent(ctx, matches[1], destDir, fallbackName)
	}

	if matches := evidencePattern.FindStringSubmatch(path); len(matches) == 2 {
		return c.downloadEvidenceTaskContent(ctx, matches[1], destDir, fallbackName)
	}

	// Not a recognized SPA pattern, let standard handler deal with it
	return nil, nil
}

// downloadPolicyContent fetches policy content via API and saves as markdown
func (c *Client) downloadPolicyContent(ctx context.Context, policyID string, destDir string, fallbackName string) (*URLDownloadResult, error) {
	c.logger.Debug("Fetching policy via API", logger.String("policy_id", policyID))

	// Fetch policy with full details using existing method
	policy, err := c.GetPolicyDetails(ctx, policyID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch policy %s: %w", policyID, err)
	}

	// Build markdown content
	content := buildPolicyMarkdownFromDetails(policy)

	// Determine filename
	name := "policy"
	if policy.Name != "" {
		name = sanitizeFilename(policy.Name)
	}
	filename := fmt.Sprintf("%s_%s.md", fallbackName, name)

	// Write file
	destPath := filepath.Join(destDir, filename)
	if err := os.WriteFile(destPath, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write policy file: %w", err)
	}

	return &URLDownloadResult{
		Filename:     filename,
		ContentType:  "text/markdown",
		BytesWritten: int64(len(content)),
	}, nil
}

// downloadControlContent fetches control content via API and saves as markdown
func (c *Client) downloadControlContent(ctx context.Context, controlID string, destDir string, fallbackName string) (*URLDownloadResult, error) {
	c.logger.Debug("Fetching control via API", logger.String("control_id", controlID))

	// Fetch control with full details using existing method
	control, err := c.GetControlDetails(ctx, controlID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch control %s: %w", controlID, err)
	}

	// Build markdown content
	content := buildControlMarkdownFromDetails(control)

	// Determine filename
	name := "control"
	if control.Name != "" {
		name = sanitizeFilename(control.Name)
	}
	filename := fmt.Sprintf("%s_%s.md", fallbackName, name)

	// Write file
	destPath := filepath.Join(destDir, filename)
	if err := os.WriteFile(destPath, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write control file: %w", err)
	}

	return &URLDownloadResult{
		Filename:     filename,
		ContentType:  "text/markdown",
		BytesWritten: int64(len(content)),
	}, nil
}

// downloadEvidenceTaskContent fetches evidence task content via API and saves as markdown
func (c *Client) downloadEvidenceTaskContent(ctx context.Context, taskID string, destDir string, fallbackName string) (*URLDownloadResult, error) {
	c.logger.Debug("Fetching evidence task via API", logger.String("task_id", taskID))

	// Fetch evidence task with full details using existing method
	task, err := c.GetEvidenceTaskDetails(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch evidence task %s: %w", taskID, err)
	}

	// Build markdown content
	content := buildEvidenceTaskMarkdownFromDetails(task)

	// Determine filename
	name := "evidence_task"
	if task.Name != "" {
		name = sanitizeFilename(task.Name)
	}
	filename := fmt.Sprintf("%s_%s.md", fallbackName, name)

	// Write file
	destPath := filepath.Join(destDir, filename)
	if err := os.WriteFile(destPath, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write evidence task file: %w", err)
	}

	return &URLDownloadResult{
		Filename:     filename,
		ContentType:  "text/markdown",
		BytesWritten: int64(len(content)),
	}, nil
}

// downloadDirectURL handles standard URL downloads (non-SPA)
func (c *Client) downloadDirectURL(ctx context.Context, sourceURL string, destDir string, fallbackName string) (*URLDownloadResult, error) {
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", sourceURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Use authenticated client for Tugboat URLs, standard client for external URLs
	var resp *http.Response
	if strings.Contains(sourceURL, "tugboatlogic.com") {
		// Add authentication for Tugboat URLs
		if c.cookieHeader != "" {
			req.Header.Set("Cookie", c.cookieHeader)
		}
		resp, err = c.httpClient.Do(req)
	} else {
		resp, err = http.DefaultClient.Do(req)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Determine filename
	filename := extractFilenameFromResponse(resp, sourceURL, fallbackName)

	// Create destination path
	destPath := filepath.Join(destDir, filename)

	// Create destination file
	outFile, err := os.Create(destPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer outFile.Close()

	// Copy content
	written, err := io.Copy(outFile, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to write content: %w", err)
	}

	result := &URLDownloadResult{
		Filename:     filename,
		ContentType:  resp.Header.Get("Content-Type"),
		BytesWritten: written,
	}

	c.logger.Debug("URL download completed",
		logger.String("filename", filename),
		logger.Int("bytes_written", int(written)))

	return result, nil
}

// buildPolicyMarkdownFromDetails creates markdown content from a PolicyDetails struct
func buildPolicyMarkdownFromDetails(policy *models.PolicyDetails) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n\n", policy.Name))

	if policy.Status != "" {
		sb.WriteString(fmt.Sprintf("**Status:** %s\n\n", policy.Status))
	}

	if policy.Category != "" {
		sb.WriteString(fmt.Sprintf("**Category:** %s\n\n", policy.Category))
	}

	sb.WriteString("---\n\n")

	// Try to get content from various sources
	hasContent := false

	if policy.Summary != "" {
		sb.WriteString("## Summary\n\n")
		sb.WriteString(policy.Summary)
		sb.WriteString("\n\n")
		hasContent = true
	}

	if policy.Details != "" {
		sb.WriteString("## Policy Details\n\n")
		sb.WriteString(policy.Details)
		sb.WriteString("\n\n")
		hasContent = true
	}

	if policy.Description != "" {
		sb.WriteString("## Description\n\n")
		sb.WriteString(policy.Description)
		sb.WriteString("\n\n")
		hasContent = true
	}

	// Get content from current version if available
	if policy.CurrentVersion != nil && policy.CurrentVersion.Content != "" {
		sb.WriteString("## Content\n\n")
		sb.WriteString(policy.CurrentVersion.Content)
		sb.WriteString("\n\n")
		hasContent = true
	}

	if !hasContent {
		sb.WriteString("*No policy content available via API.*\n")
	}

	return sb.String()
}

// buildControlMarkdownFromDetails creates markdown content from a ControlDetails struct
func buildControlMarkdownFromDetails(control *models.ControlDetails) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n\n", control.Name))

	if control.Category != "" {
		sb.WriteString(fmt.Sprintf("**Category:** %s\n\n", control.Category))
	}

	if control.Status != "" {
		sb.WriteString(fmt.Sprintf("**Status:** %s\n\n", control.Status))
	}

	sb.WriteString("---\n\n")

	hasContent := false

	// Body contains the main control description
	if control.Body != "" {
		sb.WriteString("## Description\n\n")
		sb.WriteString(control.Body)
		sb.WriteString("\n\n")
		hasContent = true
	}

	// Help contains guidance
	if control.Help != "" {
		sb.WriteString("## Guidance\n\n")
		sb.WriteString(control.Help)
		sb.WriteString("\n\n")
		hasContent = true
	}

	// MasterContent may have additional guidance
	if control.MasterContent != nil {
		if control.MasterContent.Guidance != "" && control.MasterContent.Guidance != control.Help {
			sb.WriteString("## Additional Guidance\n\n")
			sb.WriteString(control.MasterContent.Guidance)
			sb.WriteString("\n\n")
			hasContent = true
		}
	}

	if !hasContent {
		sb.WriteString("*No control content available via API.*\n")
	}

	return sb.String()
}

// buildEvidenceTaskMarkdownFromDetails creates markdown content from an EvidenceTaskDetails struct
func buildEvidenceTaskMarkdownFromDetails(task *models.EvidenceTaskDetails) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n\n", task.Name))

	if task.Status != "" {
		sb.WriteString(fmt.Sprintf("**Status:** %s\n\n", task.Status))
	}

	if task.CollectionInterval != "" {
		sb.WriteString(fmt.Sprintf("**Collection Interval:** %s\n\n", task.CollectionInterval))
	}

	sb.WriteString("---\n\n")

	hasContent := false

	if task.Description != "" {
		sb.WriteString("## Description\n\n")
		sb.WriteString(task.Description)
		sb.WriteString("\n\n")
		hasContent = true
	}

	// MasterContent contains guidance
	if task.MasterContent != nil && task.MasterContent.Guidance != "" {
		sb.WriteString("## Guidance\n\n")
		sb.WriteString(task.MasterContent.Guidance)
		sb.WriteString("\n\n")
		hasContent = true
	}

	if !hasContent {
		sb.WriteString("*No evidence task content available via API.*\n")
	}

	return sb.String()
}

// extractFilenameFromResponse determines the best filename for a downloaded file
func extractFilenameFromResponse(resp *http.Response, sourceURL string, fallbackName string) string {
	// Try Content-Disposition header first
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if _, params, err := mime.ParseMediaType(cd); err == nil {
			if filename := params["filename"]; filename != "" {
				return sanitizeFilename(filename)
			}
		}
	}

	// Try to extract from URL path
	if u, err := url.Parse(sourceURL); err == nil {
		if path := u.Path; path != "" {
			parts := strings.Split(path, "/")
			if len(parts) > 0 {
				lastPart := parts[len(parts)-1]
				if lastPart != "" && strings.Contains(lastPart, ".") {
					return sanitizeFilename(lastPart)
				}
			}
		}
	}

	// Use fallback with extension based on content type
	ext := getExtensionFromContentType(resp.Header.Get("Content-Type"))
	return fallbackName + ext
}

// getExtensionFromContentType returns a file extension for common MIME types
func getExtensionFromContentType(contentType string) string {
	// Parse the content type (may include charset)
	mediaType, _, _ := mime.ParseMediaType(contentType)

	switch mediaType {
	case "text/html":
		return ".html"
	case "application/pdf":
		return ".pdf"
	case "application/json":
		return ".json"
	case "text/plain":
		return ".txt"
	case "text/csv":
		return ".csv"
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return ".xlsx"
	case "application/vnd.ms-excel":
		return ".xls"
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	default:
		return ".html" // Default to HTML for web pages
	}
}

// sanitizeFilename removes or replaces invalid filename characters
func sanitizeFilename(name string) string {
	// Replace path separators and other problematic characters
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(name)
}
