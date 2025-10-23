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

package tools

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/forms/v1"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// GoogleWorkspaceTool provides Google Workspace evidence collection capabilities
type GoogleWorkspaceTool struct {
	config *config.Config
	logger logger.Logger
}

// NewGoogleWorkspaceTool creates a new Google Workspace tool
func NewGoogleWorkspaceTool(cfg *config.Config, log logger.Logger) Tool {
	return &GoogleWorkspaceTool{
		config: cfg,
		logger: log,
	}
}

// Name returns the tool name
func (gwt *GoogleWorkspaceTool) Name() string {
	return "google-workspace"
}

// Description returns the tool description
func (gwt *GoogleWorkspaceTool) Description() string {
	return "Extract evidence from Google Workspace documents including Drive, Docs, Sheets, and Forms"
}

// GetClaudeToolDefinition returns the tool definition for Claude
func (gwt *GoogleWorkspaceTool) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        gwt.Name(),
		Description: gwt.Description(),
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"document_id": map[string]interface{}{
					"type":        "string",
					"description": "Google document ID (from URL or share link)",
				},
				"document_type": map[string]interface{}{
					"type":        "string",
					"description": "Type of Google document: drive, docs, sheets, forms",
					"enum":        []string{"drive", "docs", "sheets", "forms"},
					"default":     "drive",
				},
				"extraction_rules": map[string]interface{}{
					"type":        "object",
					"description": "Rules for extracting content",
					"properties": map[string]interface{}{
						"include_metadata": map[string]interface{}{
							"type":        "boolean",
							"description": "Include document metadata (created, modified, editors)",
							"default":     true,
						},
						"include_revisions": map[string]interface{}{
							"type":        "boolean",
							"description": "Include revision history",
							"default":     false,
						},
						"sheet_range": map[string]interface{}{
							"type":        "string",
							"description": "For sheets: range to extract (e.g., 'A1:D10', 'Sheet1!A:Z')",
						},
						"search_query": map[string]interface{}{
							"type":        "string",
							"description": "Search query for Drive folder content",
						},
						"max_results": map[string]interface{}{
							"type":        "integer",
							"description": "Maximum number of results to return",
							"minimum":     1,
							"maximum":     100,
							"default":     20,
						},
					},
				},
				"credentials_path": map[string]interface{}{
					"type":        "string",
					"description": "Path to Google service account credentials JSON file",
				},
			},
			"required": []string{"document_id"},
		},
	}
}

// Execute runs the Google Workspace tool with the given parameters
func (gwt *GoogleWorkspaceTool) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	gwt.logger.Debug("Executing Google Workspace tool", logger.Field{Key: "params", Value: params})

	// Extract parameters
	documentID, ok := params["document_id"].(string)
	if !ok || documentID == "" {
		return "", nil, fmt.Errorf("document_id parameter is required")
	}

	documentType := "drive"
	if dt, ok := params["document_type"].(string); ok && dt != "" {
		documentType = dt
	}

	// Extract extraction rules
	extractionRules := ExtractionRules{
		IncludeMetadata:  true,
		IncludeRevisions: false,
		MaxResults:       20,
	}

	if rules, ok := params["extraction_rules"].(map[string]interface{}); ok {
		if includeMetadata, ok := rules["include_metadata"].(bool); ok {
			extractionRules.IncludeMetadata = includeMetadata
		}
		if includeRevisions, ok := rules["include_revisions"].(bool); ok {
			extractionRules.IncludeRevisions = includeRevisions
		}
		if sheetRange, ok := rules["sheet_range"].(string); ok {
			extractionRules.SheetRange = sheetRange
		}
		if searchQuery, ok := rules["search_query"].(string); ok {
			extractionRules.SearchQuery = searchQuery
		}
		if maxResults, ok := rules["max_results"].(int); ok {
			extractionRules.MaxResults = maxResults
		}
	}

	// Get credentials path
	credentialsPath := ""
	if cp, ok := params["credentials_path"].(string); ok && cp != "" {
		credentialsPath = cp
	} else {
		// Try common locations for service account credentials
		commonPaths := []string{
			"google-credentials.json",
			"service-account.json",
			filepath.Join(os.Getenv("HOME"), ".config", "gcloud", "application_default_credentials.json"),
			os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
		}

		for _, path := range commonPaths {
			if path != "" {
				if _, err := os.Stat(path); err == nil {
					credentialsPath = path
					break
				}
			}
		}
	}

	if credentialsPath == "" {
		return "", nil, fmt.Errorf("google credentials not found. Set credentials_path parameter or GOOGLE_APPLICATION_CREDENTIALS environment variable")
	}

	// Initialize Google API client
	client, err := gwt.initializeGoogleClient(ctx, credentialsPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to initialize Google client: %w", err)
	}

	// Execute based on document type
	var result *GoogleWorkspaceResult
	switch documentType {
	case "drive":
		result, err = gwt.extractFromDrive(ctx, client, documentID, extractionRules)
	case "docs":
		result, err = gwt.extractFromDocs(ctx, client, documentID, extractionRules)
	case "sheets":
		result, err = gwt.extractFromSheets(ctx, client, documentID, extractionRules)
	case "forms":
		result, err = gwt.extractFromForms(ctx, client, documentID, extractionRules)
	default:
		return "", nil, fmt.Errorf("unsupported document type: %s", documentType)
	}

	if err != nil {
		return "", nil, fmt.Errorf("failed to extract from %s: %w", documentType, err)
	}

	// Generate report
	report := gwt.generateReport(result, documentType, extractionRules)

	// Calculate relevance
	relevance := gwt.calculateRelevance(result)

	// Create evidence source
	source := &models.EvidenceSource{
		Type:        "google-workspace",
		Resource:    fmt.Sprintf("%s document: %s", documentType, result.DocumentName),
		Content:     report,
		Relevance:   relevance,
		ExtractedAt: time.Now(),
		Metadata: map[string]interface{}{
			"document_id":   documentID,
			"document_type": documentType,
			"document_name": result.DocumentName,
			"owner":         result.Owner,
			"created_at":    result.CreatedAt,
			"modified_at":   result.ModifiedAt,
			"editors":       result.Editors,
			"content_size":  len(result.Content),
		},
	}

	return report, source, nil
}

// initializeGoogleClient initializes the Google API client with service account credentials
func (gwt *GoogleWorkspaceTool) initializeGoogleClient(ctx context.Context, credentialsPath string) (*http.Client, error) {
	gwt.logger.Debug("Initializing Google client", logger.String("credentials_path", credentialsPath))

	// Read service account key file
	credentialsData, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	// Create OAuth2 config for service account
	config, err := google.JWTConfigFromJSON(credentialsData,
		drive.DriveReadonlyScope,
		docs.DocumentsReadonlyScope,
		sheets.SpreadsheetsReadonlyScope,
		forms.FormsResponsesReadonlyScope,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create JWT config: %w", err)
	}

	// Create HTTP client
	client := config.Client(ctx)
	return client, nil
}

// extractFromDrive extracts content from Google Drive (file or folder)
func (gwt *GoogleWorkspaceTool) extractFromDrive(ctx context.Context, client *http.Client, documentID string, rules ExtractionRules) (*GoogleWorkspaceResult, error) {
	driveService, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Drive service: %w", err)
	}

	// Get file information
	file, err := driveService.Files.Get(documentID).Fields("id,name,mimeType,owners,createdTime,modifiedTime,size,parents").Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get file information: %w", err)
	}

	result := &GoogleWorkspaceResult{
		DocumentID:   documentID,
		DocumentName: file.Name,
		DocumentType: gwt.getMimeTypeCategory(file.MimeType),
		CreatedAt:    gwt.parseGoogleTime(file.CreatedTime),
		ModifiedAt:   gwt.parseGoogleTime(file.ModifiedTime),
		MimeType:     file.MimeType,
	}

	if len(file.Owners) > 0 {
		result.Owner = file.Owners[0].DisplayName
	}

	// Handle different file types
	if strings.HasPrefix(file.MimeType, "application/vnd.google-apps.folder") {
		// It's a folder - list contents
		return gwt.extractFolderContents(ctx, driveService, documentID, rules, result)
	} else {
		// It's a file - get content based on type
		return gwt.extractFileContent(ctx, client, documentID, file, rules, result)
	}
}

// extractFromDocs extracts content from Google Docs
func (gwt *GoogleWorkspaceTool) extractFromDocs(ctx context.Context, client *http.Client, documentID string, rules ExtractionRules) (*GoogleWorkspaceResult, error) {
	docsService, err := docs.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Docs service: %w", err)
	}

	// Get document
	doc, err := docsService.Documents.Get(documentID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	result := &GoogleWorkspaceResult{
		DocumentID:   documentID,
		DocumentName: doc.Title,
		DocumentType: "docs",
		CreatedAt:    time.Now(), // Docs API doesn't provide creation time directly
		ModifiedAt:   time.Now(), // Would need Drive API for accurate timestamps
	}

	// Extract text content
	var content strings.Builder
	for _, element := range doc.Body.Content {
		if element.Paragraph != nil {
			for _, textElement := range element.Paragraph.Elements {
				if textElement.TextRun != nil {
					content.WriteString(textElement.TextRun.Content)
				}
			}
		}
	}

	result.Content = content.String()

	// Extract metadata if requested
	if rules.IncludeMetadata {
		result.Metadata = map[string]interface{}{
			"revision_id": doc.RevisionId,
			"title":       doc.Title,
		}
	}

	return result, nil
}

// extractFromSheets extracts content from Google Sheets
func (gwt *GoogleWorkspaceTool) extractFromSheets(ctx context.Context, client *http.Client, documentID string, rules ExtractionRules) (*GoogleWorkspaceResult, error) {
	sheetsService, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Sheets service: %w", err)
	}

	// Get spreadsheet metadata
	spreadsheet, err := sheetsService.Spreadsheets.Get(documentID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get spreadsheet: %w", err)
	}

	result := &GoogleWorkspaceResult{
		DocumentID:   documentID,
		DocumentName: spreadsheet.Properties.Title,
		DocumentType: "sheets",
		CreatedAt:    time.Now(), // Would need Drive API for accurate timestamps
		ModifiedAt:   time.Now(),
	}

	// Extract sheet data
	sheetRange := rules.SheetRange
	if sheetRange == "" {
		// Default to first sheet, all data
		if len(spreadsheet.Sheets) > 0 {
			sheetName := spreadsheet.Sheets[0].Properties.Title
			sheetRange = fmt.Sprintf("%s!A:Z", sheetName)
		} else {
			sheetRange = "A:Z"
		}
	}

	valueRange, err := sheetsService.Spreadsheets.Values.Get(documentID, sheetRange).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get sheet values: %w", err)
	}

	// Convert sheet data to formatted content
	var content strings.Builder
	content.WriteString(fmt.Sprintf("Spreadsheet: %s\n", spreadsheet.Properties.Title))
	content.WriteString(fmt.Sprintf("Range: %s\n", sheetRange))
	content.WriteString(fmt.Sprintf("Rows: %d\n\n", len(valueRange.Values)))

	// Format as CSV-like content
	for i, row := range valueRange.Values {
		if i == 0 {
			content.WriteString("Headers: ")
		} else {
			content.WriteString(fmt.Sprintf("Row %d: ", i))
		}

		var rowValues []string
		for _, cell := range row {
			if cellStr, ok := cell.(string); ok {
				rowValues = append(rowValues, cellStr)
			}
		}
		content.WriteString(strings.Join(rowValues, " | "))
		content.WriteString("\n")
	}

	result.Content = content.String()
	result.SheetData = valueRange.Values

	// Extract metadata if requested
	if rules.IncludeMetadata {
		result.Metadata = map[string]interface{}{
			"sheet_count": len(spreadsheet.Sheets),
			"range":       sheetRange,
			"row_count":   len(valueRange.Values),
		}
	}

	return result, nil
}

// extractFromForms extracts content from Google Forms
func (gwt *GoogleWorkspaceTool) extractFromForms(ctx context.Context, client *http.Client, documentID string, rules ExtractionRules) (*GoogleWorkspaceResult, error) {
	formsService, err := forms.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Forms service: %w", err)
	}

	// Get form metadata
	form, err := formsService.Forms.Get(documentID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get form: %w", err)
	}

	result := &GoogleWorkspaceResult{
		DocumentID:   documentID,
		DocumentName: form.Info.Title,
		DocumentType: "forms",
		CreatedAt:    time.Now(), // Would need Drive API for accurate timestamps
		ModifiedAt:   time.Now(),
	}

	// Extract form structure
	var content strings.Builder
	content.WriteString(fmt.Sprintf("Form: %s\n", form.Info.Title))
	if form.Info.Description != "" {
		content.WriteString(fmt.Sprintf("Description: %s\n", form.Info.Description))
	}
	content.WriteString("\nQuestions:\n")

	for i, item := range form.Items {
		if item.Title != "" {
			content.WriteString(fmt.Sprintf("%d. %s\n", i+1, item.Title))
		}
		if item.Description != "" {
			content.WriteString(fmt.Sprintf("   Description: %s\n", item.Description))
		}

		// Add question type info
		if item.QuestionItem != nil {
			questionType := gwt.getQuestionType(item.QuestionItem)
			content.WriteString(fmt.Sprintf("   Type: %s\n", questionType))
		}
		content.WriteString("\n")
	}

	// Try to get responses if available
	responses, err := formsService.Forms.Responses.List(documentID).Do()
	if err == nil && len(responses.Responses) > 0 {
		content.WriteString(fmt.Sprintf("\nResponses: %d total\n", len(responses.Responses)))

		// Show summary of recent responses (limited for privacy)
		maxResponsesToShow := 5
		if len(responses.Responses) < maxResponsesToShow {
			maxResponsesToShow = len(responses.Responses)
		}

		for i := 0; i < maxResponsesToShow; i++ {
			response := responses.Responses[i]
			content.WriteString(fmt.Sprintf("Response %d (ID: %s):\n", i+1, response.ResponseId))
			content.WriteString(fmt.Sprintf("  Submitted: %s\n", response.CreateTime))
			content.WriteString(fmt.Sprintf("  Last Updated: %s\n", response.LastSubmittedTime))
		}
	}

	result.Content = content.String()

	// Extract metadata if requested
	if rules.IncludeMetadata {
		result.Metadata = map[string]interface{}{
			"question_count":      len(form.Items),
			"response_count":      len(responses.Responses),
			"accepting_responses": form.Settings.QuizSettings != nil,
		}
	}

	return result, nil
}

// extractFolderContents extracts contents of a Google Drive folder
func (gwt *GoogleWorkspaceTool) extractFolderContents(ctx context.Context, driveService *drive.Service, folderID string, rules ExtractionRules, result *GoogleWorkspaceResult) (*GoogleWorkspaceResult, error) {
	query := fmt.Sprintf("'%s' in parents and trashed=false", folderID)
	if rules.SearchQuery != "" {
		query += fmt.Sprintf(" and fullText contains '%s'", rules.SearchQuery)
	}

	fileList, err := driveService.Files.List().
		Q(query).
		Fields("files(id,name,mimeType,owners,createdTime,modifiedTime,size)").
		PageSize(int64(rules.MaxResults)).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list folder contents: %w", err)
	}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("Folder: %s\n", result.DocumentName))
	content.WriteString(fmt.Sprintf("Files found: %d\n\n", len(fileList.Files)))

	var folderContents []FolderItem
	for _, file := range fileList.Files {
		item := FolderItem{
			ID:       file.Id,
			Name:     file.Name,
			MimeType: file.MimeType,
			Type:     gwt.getMimeTypeCategory(file.MimeType),
			Size:     file.Size,
			Created:  gwt.parseGoogleTime(file.CreatedTime),
			Modified: gwt.parseGoogleTime(file.ModifiedTime),
		}

		if len(file.Owners) > 0 {
			item.Owner = file.Owners[0].DisplayName
		}

		folderContents = append(folderContents, item)

		content.WriteString(fmt.Sprintf("- %s (%s)\n", file.Name, item.Type))
		content.WriteString(fmt.Sprintf("  ID: %s\n", file.Id))
		content.WriteString(fmt.Sprintf("  Modified: %s\n", item.Modified.Format("2006-01-02 15:04:05")))
		if item.Owner != "" {
			content.WriteString(fmt.Sprintf("  Owner: %s\n", item.Owner))
		}
		content.WriteString("\n")
	}

	result.Content = content.String()
	result.FolderContents = folderContents

	return result, nil
}

// extractFileContent extracts content from a specific file
func (gwt *GoogleWorkspaceTool) extractFileContent(ctx context.Context, client *http.Client, documentID string, file *drive.File, rules ExtractionRules, result *GoogleWorkspaceResult) (*GoogleWorkspaceResult, error) {
	// Dispatch to appropriate extraction method based on mime type
	switch {
	case strings.Contains(file.MimeType, "document"):
		return gwt.extractFromDocs(ctx, client, documentID, rules)
	case strings.Contains(file.MimeType, "spreadsheet"):
		return gwt.extractFromSheets(ctx, client, documentID, rules)
	case strings.Contains(file.MimeType, "form"):
		return gwt.extractFromForms(ctx, client, documentID, rules)
	default:
		// For other file types, just return metadata
		result.Content = fmt.Sprintf("File: %s\nType: %s\nSize: %d bytes\n", file.Name, file.MimeType, file.Size)
		return result, nil
	}
}

// Helper functions

func (gwt *GoogleWorkspaceTool) getMimeTypeCategory(mimeType string) string {
	switch {
	case strings.Contains(mimeType, "folder"):
		return "folder"
	case strings.Contains(mimeType, "document"):
		return "document"
	case strings.Contains(mimeType, "spreadsheet"):
		return "spreadsheet"
	case strings.Contains(mimeType, "presentation"):
		return "presentation"
	case strings.Contains(mimeType, "form"):
		return "form"
	case strings.Contains(mimeType, "drawing"):
		return "drawing"
	default:
		return "file"
	}
}

func (gwt *GoogleWorkspaceTool) parseGoogleTime(timeStr string) time.Time {
	if timeStr == "" {
		return time.Time{}
	}

	// Google API returns RFC3339 format
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		gwt.logger.Warn("Failed to parse time", logger.String("time", timeStr), logger.Field{Key: "error", Value: err})
		return time.Time{}
	}
	return t
}

func (gwt *GoogleWorkspaceTool) getQuestionType(questionItem *forms.QuestionItem) string {
	if questionItem.Question == nil {
		return "unknown"
	}

	question := questionItem.Question
	switch {
	case question.ChoiceQuestion != nil:
		return "choice"
	case question.TextQuestion != nil:
		return "text"
	case question.ScaleQuestion != nil:
		return "scale"
	case question.DateQuestion != nil:
		return "date"
	case question.TimeQuestion != nil:
		return "time"
	case question.FileUploadQuestion != nil:
		return "file_upload"
	default:
		return "other"
	}
}

func (gwt *GoogleWorkspaceTool) calculateRelevance(result *GoogleWorkspaceResult) float64 {
	relevance := 0.5 // Base relevance

	// Increase relevance based on content size
	contentLength := len(result.Content)
	if contentLength > 5000 {
		relevance += 0.3
	} else if contentLength > 1000 {
		relevance += 0.2
	} else if contentLength > 100 {
		relevance += 0.1
	}

	// Increase relevance for recent modifications
	if !result.ModifiedAt.IsZero() {
		daysSinceModified := time.Since(result.ModifiedAt).Hours() / 24
		if daysSinceModified <= 7 {
			relevance += 0.2
		} else if daysSinceModified <= 30 {
			relevance += 0.1
		}
	}

	// Increase relevance for certain document types
	switch result.DocumentType {
	case "sheets":
		relevance += 0.1 // Structured data often more relevant
	case "forms":
		relevance += 0.15 // Forms responses are usually highly relevant
	}

	// Increase relevance if it's a folder with multiple items
	if len(result.FolderContents) > 0 {
		relevance += float64(len(result.FolderContents)) * 0.01
	}

	// Cap at 1.0
	if relevance > 1.0 {
		relevance = 1.0
	}

	return relevance
}

func (gwt *GoogleWorkspaceTool) generateReport(result *GoogleWorkspaceResult, documentType string, rules ExtractionRules) string {
	var report strings.Builder

	report.WriteString("# Google Workspace Evidence Report\n\n")
	report.WriteString(fmt.Sprintf("**Document Type**: %s\n", documentType))
	report.WriteString(fmt.Sprintf("**Document Name**: %s\n", result.DocumentName))
	report.WriteString(fmt.Sprintf("**Document ID**: %s\n", result.DocumentID))
	report.WriteString(fmt.Sprintf("**Generated**: %s\n\n", time.Now().Format(time.RFC3339)))

	// Document metadata
	if rules.IncludeMetadata {
		report.WriteString("## Document Metadata\n\n")
		report.WriteString(fmt.Sprintf("- **Owner**: %s\n", result.Owner))
		if !result.CreatedAt.IsZero() {
			report.WriteString(fmt.Sprintf("- **Created**: %s\n", result.CreatedAt.Format("2006-01-02 15:04:05")))
		}
		if !result.ModifiedAt.IsZero() {
			report.WriteString(fmt.Sprintf("- **Last Modified**: %s\n", result.ModifiedAt.Format("2006-01-02 15:04:05")))
		}
		if result.MimeType != "" {
			report.WriteString(fmt.Sprintf("- **MIME Type**: %s\n", result.MimeType))
		}
		if len(result.Editors) > 0 {
			report.WriteString(fmt.Sprintf("- **Editors**: %s\n", strings.Join(result.Editors, ", ")))
		}
		report.WriteString("\n")
	}

	// Folder contents (if applicable)
	if len(result.FolderContents) > 0 {
		report.WriteString("## Folder Contents\n\n")

		// Sort by modification date (newest first)
		sort.Slice(result.FolderContents, func(i, j int) bool {
			return result.FolderContents[i].Modified.After(result.FolderContents[j].Modified)
		})

		for _, item := range result.FolderContents {
			report.WriteString(fmt.Sprintf("### %s\n\n", item.Name))
			report.WriteString(fmt.Sprintf("- **Type**: %s\n", item.Type))
			report.WriteString(fmt.Sprintf("- **ID**: %s\n", item.ID))
			report.WriteString(fmt.Sprintf("- **Owner**: %s\n", item.Owner))
			report.WriteString(fmt.Sprintf("- **Modified**: %s\n", item.Modified.Format("2006-01-02 15:04:05")))
			if item.Size > 0 {
				report.WriteString(fmt.Sprintf("- **Size**: %d bytes\n", item.Size))
			}
			report.WriteString("\n")
		}
	}

	// Sheet data summary (if applicable)
	if len(result.SheetData) > 0 {
		report.WriteString("## Sheet Data Summary\n\n")
		report.WriteString(fmt.Sprintf("- **Total Rows**: %d\n", len(result.SheetData)))
		if len(result.SheetData) > 0 {
			report.WriteString(fmt.Sprintf("- **Columns**: %d\n", len(result.SheetData[0])))
		}
		report.WriteString("\n")
	}

	// Document content
	report.WriteString("## Document Content\n\n")
	if result.Content != "" {
		report.WriteString("```\n")
		report.WriteString(result.Content)
		report.WriteString("\n```\n\n")
	} else {
		report.WriteString("*No content extracted*\n\n")
	}

	// Additional metadata
	if len(result.Metadata) > 0 {
		report.WriteString("## Additional Metadata\n\n")
		for key, value := range result.Metadata {
			report.WriteString(fmt.Sprintf("- **%s**: %v\n", key, value))
		}
		report.WriteString("\n")
	}

	return report.String()
}

// Data structures

// ExtractionRules defines rules for extracting content from Google Workspace
type ExtractionRules struct {
	IncludeMetadata  bool   `json:"include_metadata"`
	IncludeRevisions bool   `json:"include_revisions"`
	SheetRange       string `json:"sheet_range,omitempty"`
	SearchQuery      string `json:"search_query,omitempty"`
	MaxResults       int    `json:"max_results"`
}

// GoogleWorkspaceResult represents the extracted data from Google Workspace
type GoogleWorkspaceResult struct {
	DocumentID     string                 `json:"document_id"`
	DocumentName   string                 `json:"document_name"`
	DocumentType   string                 `json:"document_type"`
	Owner          string                 `json:"owner"`
	Editors        []string               `json:"editors"`
	CreatedAt      time.Time              `json:"created_at"`
	ModifiedAt     time.Time              `json:"modified_at"`
	MimeType       string                 `json:"mime_type"`
	Content        string                 `json:"content"`
	SheetData      [][]interface{}        `json:"sheet_data,omitempty"`
	FolderContents []FolderItem           `json:"folder_contents,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// FolderItem represents an item within a Google Drive folder
type FolderItem struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Type     string    `json:"type"`
	MimeType string    `json:"mime_type"`
	Owner    string    `json:"owner"`
	Size     int64     `json:"size"`
	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`
}

// NewGoogleWorkspaceToolWithMappings creates a Google Workspace tool with evidence mapping support
func NewGoogleWorkspaceToolWithMappings(cfg *config.Config, log logger.Logger) *GoogleWorkspaceToolWithMappings {
	return &GoogleWorkspaceToolWithMappings{
		GoogleWorkspaceTool: &GoogleWorkspaceTool{
			config: cfg,
			logger: log,
		},
		mappingsLoader: NewGoogleEvidenceMappingsLoader(cfg, log),
	}
}

// GoogleWorkspaceToolWithMappings extends the base Google Workspace tool with evidence mapping capabilities
type GoogleWorkspaceToolWithMappings struct {
	*GoogleWorkspaceTool
	mappingsLoader *GoogleEvidenceMappingsLoader
}

// ExecuteForEvidenceTask executes the Google Workspace tool using predefined evidence mappings
func (gwt *GoogleWorkspaceToolWithMappings) ExecuteForEvidenceTask(ctx context.Context, taskRef string) ([]string, []*models.EvidenceSource, error) {
	gwt.logger.Info("Executing Google Workspace tool for evidence task", logger.String("task_ref", taskRef))

	// Get the mapping for this task
	mapping, err := gwt.mappingsLoader.GetMappingForTask(taskRef)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get mapping for task %s: %w", taskRef, err)
	}

	// Validate document access
	if err := gwt.mappingsLoader.ValidateDocumentAccess(mapping); err != nil {
		return nil, nil, fmt.Errorf("document access validation failed: %w", err)
	}

	var reports []string
	var sources []*models.EvidenceSource

	// Execute for each document in the mapping
	for i, doc := range mapping.Documents {
		gwt.logger.Debug("Processing document",
			logger.String("task_ref", taskRef),
			logger.String("document_id", doc.DocumentID),
			logger.String("document_type", doc.DocumentType),
			logger.Int("document_index", i))

		// Transform mapping to API parameters
		params, err := gwt.mappingsLoader.TransformToGoogleAPIParams(mapping, i)
		if err != nil {
			gwt.logger.Warn("Failed to transform mapping to API params",
				logger.String("task_ref", taskRef),
				logger.String("document_id", doc.DocumentID),
				logger.Field{Key: "error", Value: err})
			continue
		}

		// Execute the tool
		report, source, err := gwt.Execute(ctx, params)
		if err != nil {
			gwt.logger.Warn("Failed to execute tool for document",
				logger.String("task_ref", taskRef),
				logger.String("document_id", doc.DocumentID),
				logger.Field{Key: "error", Value: err})
			continue
		}

		// Enhance the source with mapping metadata
		if source != nil {
			source.Metadata["task_ref"] = taskRef
			source.Metadata["mapping_priority"] = mapping.Priority
			source.Metadata["mapping_description"] = mapping.Description
			source.Metadata["document_name"] = doc.DocumentName

			// Add validation status
			validationStatus := gwt.validateDocumentContent(doc, source)
			source.Metadata["validation_status"] = validationStatus

			// Adjust relevance based on validation
			if validationStatus["passed"].(bool) {
				boosted := source.Relevance * 1.1
				if boosted > 1.0 {
					source.Relevance = 1.0
				} else {
					source.Relevance = boosted
				}
			} else {
				reduced := source.Relevance * 0.8
				if reduced < 0.1 {
					source.Relevance = 0.1
				} else {
					source.Relevance = reduced
				}
			}
		}

		reports = append(reports, report)
		sources = append(sources, source)

		gwt.logger.Debug("Successfully processed document",
			logger.String("task_ref", taskRef),
			logger.String("document_id", doc.DocumentID),
			logger.Field{Key: "relevance", Value: source.Relevance})
	}

	if len(reports) == 0 {
		return nil, nil, fmt.Errorf("no documents could be processed for task %s", taskRef)
	}

	gwt.logger.Info("Completed Google Workspace evidence collection",
		logger.String("task_ref", taskRef),
		logger.Int("documents_processed", len(reports)),
		logger.Int("sources_generated", len(sources)))

	return reports, sources, nil
}

// validateDocumentContent validates extracted content against mapping requirements
func (gwt *GoogleWorkspaceToolWithMappings) validateDocumentContent(doc DocumentConfig, source *models.EvidenceSource) map[string]interface{} {
	validation := map[string]interface{}{
		"passed":   true,
		"errors":   []string{},
		"warnings": []string{},
	}

	errors := []string{}
	warnings := []string{}

	// Check minimum content length
	if doc.Validation.MinContentLength > 0 && len(source.Content) < doc.Validation.MinContentLength {
		errors = append(errors, fmt.Sprintf("Content length %d is below minimum %d",
			len(source.Content), doc.Validation.MinContentLength))
	}

	// Check required keywords
	if len(doc.Validation.RequiredKeywords) > 0 {
		contentLower := strings.ToLower(source.Content)
		for _, keyword := range doc.Validation.RequiredKeywords {
			if !strings.Contains(contentLower, strings.ToLower(keyword)) {
				warnings = append(warnings, fmt.Sprintf("Required keyword '%s' not found in content", keyword))
			}
		}
	}

	// Check date range validation if applicable
	if doc.Validation.DateRange != nil {
		if extractedAt, ok := source.Metadata["modified_at"].(time.Time); ok {
			if doc.Validation.DateRange.From != "" {
				if fromDate, err := time.Parse("2006-01-02", doc.Validation.DateRange.From); err == nil {
					if extractedAt.Before(fromDate) {
						warnings = append(warnings, fmt.Sprintf("Document date %s is before required range start %s",
							extractedAt.Format("2006-01-02"), doc.Validation.DateRange.From))
					}
				}
			}
			if doc.Validation.DateRange.To != "" {
				if toDate, err := time.Parse("2006-01-02", doc.Validation.DateRange.To); err == nil {
					if extractedAt.After(toDate) {
						warnings = append(warnings, fmt.Sprintf("Document date %s is after required range end %s",
							extractedAt.Format("2006-01-02"), doc.Validation.DateRange.To))
					}
				}
			}
		}
	}

	// Set validation results
	validation["errors"] = errors
	validation["warnings"] = warnings
	validation["passed"] = len(errors) == 0

	return validation
}

// GetSupportedEvidenceTasks returns a list of evidence tasks supported by this tool
func (gwt *GoogleWorkspaceToolWithMappings) GetSupportedEvidenceTasks() ([]string, error) {
	return gwt.mappingsLoader.GetSupportedTaskRefs()
}

// GetTaskMappingInfo returns detailed information about a task's mapping configuration
func (gwt *GoogleWorkspaceToolWithMappings) GetTaskMappingInfo(taskRef string) (*EvidenceMapping, error) {
	return gwt.mappingsLoader.GetMappingForTask(taskRef)
}

// RefreshMappings clears the mappings cache and forces a reload
func (gwt *GoogleWorkspaceToolWithMappings) RefreshMappings() {
	gwt.mappingsLoader.ClearCache()
	gwt.logger.Info("Refreshed Google Workspace evidence mappings cache")
}
