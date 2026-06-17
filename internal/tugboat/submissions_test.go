// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

//go:build integration
// +build integration

package tugboat

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/tugboat/models"
	"github.com/grctool/grctool/internal/vcr"
	"github.com/spf13/viper"
)

// TestClient_EvidenceAttachmentOperations tests evidence attachment retrieval using VCR
// These tests verify that the grctool properly downloads evidence from Tugboat Logic
func TestClient_EvidenceAttachmentOperations(t *testing.T) {
	ctx := context.Background()
	client := setupSubmissionsTestClient(t, "evidence_attachments")
	defer client.Close()

	// ET-0050: Reports of User Access Reviews - has automated file attachments
	t.Run("GetEvidenceAttachments_AccessReviews_ET0050", func(t *testing.T) {
		taskID := 328041 // ET-0050 task ID

		attachments, err := client.GetEvidenceAttachmentsByTask(ctx, taskID)
		if err != nil {
			t.Fatalf("Failed to get evidence attachments: %v", err)
		}

		if len(attachments) == 0 {
			t.Error("Expected at least one attachment for ET-0050")
		}

		// Count file attachments (these are "automated" type with xlsx files)
		fileCount := 0
		for _, att := range attachments {
			if att.Attachment != nil {
				fileCount++
				t.Logf("✅ File attachment %d: %s (type=%s)",
					att.ID,
					att.Attachment.OriginalFilename,
					att.Type)
			}
		}

		if fileCount == 0 {
			t.Error("Expected file attachments for ET-0050 (access reviews)")
		}

		t.Logf("✅ Found %d total attachments, %d with files", len(attachments), fileCount)
	})

	// ET-0041: Risk Assessment Report - has automated file attachments
	t.Run("GetEvidenceAttachments_RiskAssessment_ET0041", func(t *testing.T) {
		taskID := 328032 // ET-0041 task ID

		attachments, err := client.GetEvidenceAttachmentsByTask(ctx, taskID)
		if err != nil {
			t.Fatalf("Failed to get evidence attachments: %v", err)
		}

		if len(attachments) == 0 {
			t.Error("Expected at least one attachment for ET-0041")
		}

		// Count file attachments
		fileCount := 0
		for _, att := range attachments {
			if att.Attachment != nil {
				fileCount++
				t.Logf("✅ File attachment %d: %s (type=%s)",
					att.ID,
					att.Attachment.OriginalFilename,
					att.Type)
			}
		}

		if fileCount == 0 {
			t.Error("Expected file attachments for ET-0041 (risk assessments)")
		}

		t.Logf("✅ Found %d total attachments, %d with files", len(attachments), fileCount)
	})

	// Test that attachments have expected file types
	t.Run("VerifyAttachmentFileTypes", func(t *testing.T) {
		taskID := 328041 // ET-0050

		attachments, err := client.GetEvidenceAttachmentsByTask(ctx, taskID)
		if err != nil {
			t.Fatalf("Failed to get evidence attachments: %v", err)
		}

		xlsxCount := 0
		for _, att := range attachments {
			if att.Attachment != nil {
				// Check for expected Excel MIME type
				if att.Attachment.MimeType == "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
					xlsxCount++
				}
			}
		}

		if xlsxCount == 0 {
			t.Error("Expected xlsx files for access reviews")
		}

		t.Logf("✅ Found %d xlsx files", xlsxCount)
	})
}

// TestClient_AttachmentContent verifies attachment metadata is properly populated
func TestClient_AttachmentContent(t *testing.T) {
	ctx := context.Background()
	client := setupSubmissionsTestClient(t, "attachment_content")
	defer client.Close()

	t.Run("VerifyAttachmentFields", func(t *testing.T) {
		taskID := 328041 // ET-0050

		attachments, err := client.GetEvidenceAttachmentsByTask(ctx, taskID)
		if err != nil {
			t.Fatalf("Failed to get evidence attachments: %v", err)
		}

		// Find first attachment with file
		var fileAttachment *models.EvidenceAttachment
		for i := range attachments {
			att := &attachments[i]
			if att.Attachment != nil {
				fileAttachment = att
				break
			}
		}

		if fileAttachment == nil {
			t.Fatal("No file attachment found")
		}

		// Verify expected fields are populated
		if fileAttachment.ID == 0 {
			t.Error("Expected attachment to have ID")
		}
		if fileAttachment.Collected == "" {
			t.Error("Expected attachment to have Collected date")
		}
		if fileAttachment.Attachment.OriginalFilename == "" {
			t.Error("Expected attachment to have filename")
		}
		if fileAttachment.Attachment.MimeType == "" {
			t.Error("Expected attachment to have MIME type")
		}

		t.Logf("✅ Attachment %d has all expected fields", fileAttachment.ID)
		t.Logf("   Filename: %s", fileAttachment.Attachment.OriginalFilename)
		t.Logf("   MIME Type: %s", fileAttachment.Attachment.MimeType)
		t.Logf("   Collected: %s", fileAttachment.Collected)
		t.Logf("   Type: %s", fileAttachment.Type)
		t.Logf("   Integration: %s/%s", fileAttachment.IntegrationType, fileAttachment.IntegrationSubtype)
	})
}

// TestClient_SPAURLDownload tests downloading content from Tugboat SPA URLs via API
// These tests verify that policy/control/evidence task URLs are properly fetched via API
func TestClient_SPAURLDownload(t *testing.T) {
	ctx := context.Background()
	client := setupSubmissionsTestClient(t, "spa_url_download")
	defer client.Close()

	// Test downloading policy content via API
	t.Run("DownloadPolicyContent", func(t *testing.T) {
		destDir := t.TempDir()
		policyID := "94641" // Acceptable Use policy

		result, err := client.downloadPolicyContent(ctx, policyID, destDir, "url_test")
		if err != nil {
			t.Fatalf("Failed to download policy content: %v", err)
		}

		if result.Filename == "" {
			t.Error("Expected filename to be set")
		}

		if result.BytesWritten == 0 {
			t.Error("Expected content to be written")
		}

		if result.ContentType != "text/markdown" {
			t.Errorf("Expected content type 'text/markdown', got %q", result.ContentType)
		}

		t.Logf("✅ Downloaded policy as %s (%d bytes)", result.Filename, result.BytesWritten)

		// Verify file exists and has content
		content, err := os.ReadFile(filepath.Join(destDir, result.Filename))
		if err != nil {
			t.Fatalf("Failed to read downloaded file: %v", err)
		}

		// Verify markdown structure
		contentStr := string(content)
		if !strings.Contains(contentStr, "# ") {
			t.Error("Expected markdown heading in content")
		}
		if !strings.Contains(contentStr, "**Status:**") {
			t.Error("Expected status field in content")
		}

		t.Logf("✅ Policy content verified (%d chars)", len(contentStr))
	})

	// Test downloading control content via API
	t.Run("DownloadControlContent", func(t *testing.T) {
		destDir := t.TempDir()
		controlID := "778771" // Access Provisioning control

		result, err := client.downloadControlContent(ctx, controlID, destDir, "url_test")
		if err != nil {
			t.Fatalf("Failed to download control content: %v", err)
		}

		if result.Filename == "" {
			t.Error("Expected filename to be set")
		}

		if result.BytesWritten == 0 {
			t.Error("Expected content to be written")
		}

		t.Logf("✅ Downloaded control as %s (%d bytes)", result.Filename, result.BytesWritten)

		// Verify file exists and has content
		content, err := os.ReadFile(filepath.Join(destDir, result.Filename))
		if err != nil {
			t.Fatalf("Failed to read downloaded file: %v", err)
		}

		contentStr := string(content)
		if !strings.Contains(contentStr, "# ") {
			t.Error("Expected markdown heading in content")
		}

		t.Logf("✅ Control content verified (%d chars)", len(contentStr))
	})

	// Test downloading evidence task content via API
	t.Run("DownloadEvidenceTaskContent", func(t *testing.T) {
		destDir := t.TempDir()
		taskID := "327992" // ET-0001 task

		result, err := client.downloadEvidenceTaskContent(ctx, taskID, destDir, "url_test")
		if err != nil {
			t.Fatalf("Failed to download evidence task content: %v", err)
		}

		if result.Filename == "" {
			t.Error("Expected filename to be set")
		}

		if result.BytesWritten == 0 {
			t.Error("Expected content to be written")
		}

		t.Logf("✅ Downloaded evidence task as %s (%d bytes)", result.Filename, result.BytesWritten)

		// Verify file exists and has content
		content, err := os.ReadFile(filepath.Join(destDir, result.Filename))
		if err != nil {
			t.Fatalf("Failed to read downloaded file: %v", err)
		}

		contentStr := string(content)
		if !strings.Contains(contentStr, "# ") {
			t.Error("Expected markdown heading in content")
		}
		if !strings.Contains(contentStr, "## Description") || !strings.Contains(contentStr, "## Guidance") {
			t.Log("Note: Evidence task may not have description/guidance sections")
		}

		t.Logf("✅ Evidence task content verified (%d chars)", len(contentStr))
	})

	// Test full SPA URL handling
	t.Run("HandleTugboatSPAURL_Policy", func(t *testing.T) {
		destDir := t.TempDir()
		policyURL := "https://my.tugboatlogic.com/org/13888/policies/94641"

		result, err := client.handleTugboatSPAURL(ctx, policyURL, destDir, "spa_test")
		if err != nil {
			t.Fatalf("Failed to handle SPA URL: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result for policy URL, got nil")
		}

		if result.BytesWritten == 0 {
			t.Error("Expected content to be downloaded")
		}

		t.Logf("✅ SPA URL handled: %s (%d bytes)", result.Filename, result.BytesWritten)
	})
}

// TestClient_TextSubmissions tests handling of text-only submissions
func TestClient_TextSubmissions(t *testing.T) {
	ctx := context.Background()
	client := setupSubmissionsTestClient(t, "text_submissions")
	defer client.Close()

	// ET-0057 has text-only submissions alongside file attachments
	t.Run("GetTextSubmissions_ET0057", func(t *testing.T) {
		taskID := 328048 // ET-0057 - Backup Schedule

		attachments, err := client.GetEvidenceAttachmentsByTask(ctx, taskID)
		if err != nil {
			t.Fatalf("Failed to get evidence attachments: %v", err)
		}

		if len(attachments) == 0 {
			t.Error("Expected at least one attachment for ET-0057")
		}

		// Count different types
		fileCount := 0
		textCount := 0
		urlCount := 0

		for _, att := range attachments {
			if att.Attachment != nil {
				fileCount++
			} else if att.Type == "url" && att.URL != "" {
				urlCount++
			} else if att.Notes != "" {
				textCount++
			}
		}

		t.Logf("✅ Found %d total attachments: %d files, %d URLs, %d text-only",
			len(attachments), fileCount, urlCount, textCount)

		// ET-0057 is known to have text submissions
		if textCount == 0 {
			t.Log("Note: No text-only submissions found in current window")
		}
	})
}

// setupSubmissionsTestClient creates a VCR-enabled client for submissions tests
func setupSubmissionsTestClient(t *testing.T, testScenario string) *Client {
	cassetteDir := filepath.Join("testdata", "vcr_cassettes")

	if err := os.MkdirAll(cassetteDir, 0755); err != nil {
		t.Fatalf("Failed to create cassette directory: %v", err)
	}

	vcrConfig := &vcr.Config{
		Enabled:         true,
		Mode:            vcr.ModePlayback,
		CassetteDir:     cassetteDir,
		SanitizeHeaders: true,
		SanitizeParams:  true,
		RedactHeaders:   []string{"authorization", "cookie", "x-api-key", "token"},
		RedactParams:    []string{"api_key", "token", "password", "secret"},
		MatchMethod:     true,
		MatchURI:        true,
		MatchQuery:      false,
		MatchHeaders:    false,
		MatchBody:       false,
	}

	// Allow override for recording
	if os.Getenv("VCR_MODE") == "record" {
		vcrConfig.Mode = vcr.ModeRecord
	} else if os.Getenv("VCR_MODE") == "record_once" {
		vcrConfig.Mode = vcr.ModeRecordOnce
	}

	// Initialize viper for config - search from project root
	viper.Reset()
	viper.SetConfigName(".grctool")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")           // Current directory
	viper.AddConfigPath("../..")       // internal/tugboat -> grctool root
	viper.AddConfigPath("../../..")    // For nested test runs
	viper.AddConfigPath("$HOME")       // Home directory fallback

	// Also try explicit read from known location
	_ = viper.ReadInConfig()

	// Load configuration
	var tugboatConfig *config.TugboatConfig
	cfg, err := config.Load()
	if err != nil {
		// Fallback for tests
		tugboatConfig = &config.TugboatConfig{
			BaseURL:   "https://api-my.tugboatlogic.com",
			Timeout:   30 * time.Second,
			RateLimit: 10,
		}
		if vcrConfig.Mode == vcr.ModeRecord {
			t.Fatalf("Failed to load config for recording: %v", err)
		}
	} else {
		tugboatConfig = &cfg.Tugboat
	}

	// Create client with VCR config
	client := NewClient(tugboatConfig, vcrConfig)

	return client
}
