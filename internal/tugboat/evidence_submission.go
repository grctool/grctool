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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"
)

// SubmitEvidenceRequest represents a request to submit evidence via Custom Evidence Integration API
type SubmitEvidenceRequest struct {
	CollectorURL  string    // Full collector URL (e.g., https://openapi.tugboatlogic.com/api/v0/evidence/collector/805/)
	FilePath      string    // Path to the evidence file to upload
	CollectedDate time.Time // Date the evidence was collected
	ContentType   string    // MIME type of the file (e.g., "text/csv", "application/json")
}

// SubmitEvidenceResponse represents the response from submitting evidence
type SubmitEvidenceResponse struct {
	Success    bool      `json:"success"`
	Message    string    `json:"message,omitempty"`
	ReceivedAt time.Time `json:"received_at"`
}

// SubmitEvidence submits evidence using the Tugboat Logic Custom Evidence Integration API
// Uses HTTP Basic Auth + X-API-KEY header with multipart/form-data upload
// API Documentation: https://support.tugboatlogic.com/hc/en-us/articles/360049620392
func (c *Client) SubmitEvidence(ctx context.Context, req *SubmitEvidenceRequest) (*SubmitEvidenceResponse, error) {
	// Validate request
	if req.CollectorURL == "" {
		return nil, fmt.Errorf("collector URL is required")
	}
	if req.FilePath == "" {
		return nil, fmt.Errorf("file path is required")
	}

	// Open the evidence file
	file, err := os.Open(req.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", req.FilePath, err)
	}
	defer file.Close()

	// Get file info for validation
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Validate file size (max 20MB as per Tugboat API docs)
	const maxFileSize = 20 * 1024 * 1024 // 20MB
	if fileInfo.Size() > maxFileSize {
		return nil, fmt.Errorf("file size %d bytes exceeds maximum allowed size of %d bytes (20MB)", fileInfo.Size(), maxFileSize)
	}

	// Create multipart form-data body
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add collected date field (ISO8601 format: yyyy-mm-dd)
	collectedDate := req.CollectedDate.Format("2006-01-02")
	if err := writer.WriteField("collected", collectedDate); err != nil {
		return nil, fmt.Errorf("failed to write collected field: %w", err)
	}

	// Add file field
	filename := filepath.Base(req.FilePath)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	// Copy file content to multipart form
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

	// Close the multipart writer to finalize the body
	contentType := writer.FormDataContentType()
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Make request using Custom Evidence Integration API authentication
	resp, err := c.makeEvidenceRequest(ctx, "POST", req.CollectorURL, body, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to submit evidence: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("evidence submission failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response (Tugboat might return empty body on success or JSON)
	result := &SubmitEvidenceResponse{
		Success:    true,
		ReceivedAt: time.Now(),
	}

	// Try to parse JSON response if present
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			// If JSON parsing fails, just use the default success response
			result.Message = string(respBody)
		}
	}

	return result, nil
}

// ValidateFileType validates that the file extension is supported by Tugboat
// Supported: txt, csv, odt, ods, xls, json, pdf, png, gif, jpg, jpeg
// Not supported: html, htm, js, exe, php5, pht, phtml, shtml, asa, cer, asax, swf, xap
func ValidateFileType(filename string) error {
	ext := filepath.Ext(filename)
	if ext == "" {
		return fmt.Errorf("file has no extension")
	}
	ext = ext[1:] // Remove leading dot

	// Check unsupported extensions first
	unsupported := map[string]bool{
		"html": true, "htm": true, "js": true, "exe": true,
		"php5": true, "pht": true, "phtml": true, "shtml": true,
		"asa": true, "cer": true, "asax": true, "swf": true, "xap": true,
	}
	if unsupported[ext] {
		return fmt.Errorf("file extension .%s is not supported by Tugboat", ext)
	}

	// Check supported extensions
	supported := map[string]bool{
		"txt": true, "csv": true, "odt": true, "ods": true,
		"xls": true, "json": true, "pdf": true, "png": true,
		"gif": true, "jpg": true, "jpeg": true,
	}
	if !supported[ext] {
		return fmt.Errorf("file extension .%s is not in the list of supported types", ext)
	}

	return nil
}
