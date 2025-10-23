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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/storage"
)

// NameGeneratorTool uses Claude to generate concise names for documents
type NameGeneratorTool struct {
	config    *config.Config
	logger    logger.Logger
	dataStore interfaces.LocalDataStore
	validator *Validator
}

// NewNameGeneratorTool creates a new NameGeneratorTool
func NewNameGeneratorTool(cfg *config.Config, log logger.Logger) Tool {
	// Initialize local data store for offline-first operation
	paths := cfg.Storage.Paths.WithDefaults().ResolveRelativeTo(cfg.Storage.DataDir)
	localDataStore, err := storage.NewLocalDataStore(cfg.Storage.DataDir, paths)
	if err != nil {
		log.Error("Failed to initialize local data store for name generator tool",
			logger.Field{Key: "error", Value: err})
		return nil
	}

	// Initialize validator for reference ID validation
	validator := NewValidator(cfg.Storage.DataDir)

	return &NameGeneratorTool{
		config:    cfg,
		logger:    log,
		dataStore: localDataStore,
		validator: validator,
	}
}

// Name returns the tool name
func (ngt *NameGeneratorTool) Name() string {
	return "name-generator"
}

// Description returns the tool description
func (ngt *NameGeneratorTool) Description() string {
	return "Generates concise, filesystem-friendly names for evidence tasks, controls, and policies using Claude AI"
}

// GetClaudeToolDefinition returns the tool definition for Claude
func (ngt *NameGeneratorTool) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        ngt.Name(),
		Description: ngt.Description(),
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"document_type": map[string]interface{}{
					"type":        "string",
					"description": "Type of document: 'evidence', 'control', or 'policy'",
					"enum":        []string{"evidence", "control", "policy"},
				},
				"reference_id": map[string]interface{}{
					"type":        "string",
					"description": "Document reference ID (e.g., 'ET-101', 'AC-1', 'POL-001')",
					"examples":    []string{"ET-101", "AC-1", "POL-001", "CC-1.1", "SO-19"},
				},
				"batch_mode": map[string]interface{}{
					"type":        "boolean",
					"description": "Process all documents of the specified type (default: false)",
					"default":     false,
				},
			},
			"required": []string{"document_type"},
		},
	}
}

// Execute generates names for documents
func (ngt *NameGeneratorTool) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	ngt.logger.Debug("Executing name generation", logger.Field{Key: "params", Value: params})

	// Extract document type
	docTypeRaw, ok := params["document_type"]
	if !ok {
		return "", nil, fmt.Errorf("document_type parameter is required")
	}

	docType, ok := docTypeRaw.(string)
	if !ok {
		return "", nil, fmt.Errorf("document_type must be a string")
	}

	// Validate document type
	if docType != "evidence" && docType != "control" && docType != "policy" {
		return "", nil, fmt.Errorf("document_type must be 'evidence', 'control', or 'policy'")
	}

	// Check for batch mode
	batchMode := false
	if batchRaw, exists := params["batch_mode"]; exists {
		if batch, ok := batchRaw.(bool); ok {
			batchMode = batch
		}
	}

	if batchMode {
		return ngt.processBatch(ctx, docType)
	}

	// Single document processing
	refIDRaw, ok := params["reference_id"]
	if !ok {
		return "", nil, fmt.Errorf("reference_id parameter is required for single document processing")
	}

	refID, ok := refIDRaw.(string)
	if !ok {
		return "", nil, fmt.Errorf("reference_id must be a string")
	}

	return ngt.processSingle(ctx, docType, refID)
}

// processSingle generates a name for a single document
func (ngt *NameGeneratorTool) processSingle(ctx context.Context, docType, refID string) (string, *models.EvidenceSource, error) {
	// Normalize reference ID using new standardized format
	normalizedRefID, err := ngt.validator.NormalizeReferenceID(refID, docType)
	if err != nil {
		return "", nil, fmt.Errorf("invalid reference ID '%s': %w", refID, err)
	}

	// For backwards compatibility with evidence tasks, also validate using the old method
	var validationResult *ValidationResult
	if docType == "evidence" {
		validationResult, err = ngt.validator.ValidateTaskReference(refID)
		if err != nil || !validationResult.Valid {
			return "", nil, fmt.Errorf("invalid evidence task reference '%s': %w", refID, err)
		}
	} else {
		// For non-evidence documents, create a simple validation result
		validationResult = &ValidationResult{
			Valid:      true,
			Errors:     []ValidationError{},
			Warnings:   []ValidationError{},
			Normalized: map[string]string{"ref_id": normalizedRefID},
		}
	}

	// Get document details
	docDetails, err := ngt.getDocumentDetails(docType, refID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get document details: %w", err)
	}

	// Generate short name using Claude
	shortName, err := ngt.generateShortName(ctx, docDetails)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate short name: %w", err)
	}

	// Format result
	result := map[string]interface{}{
		"document_type":           docType,
		"original_reference_id":   refID,
		"normalized_reference_id": normalizedRefID,
		"current_name":            docDetails.CurrentName,
		"generated_name":          shortName,
		"validation":              validationResult,
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	// Create evidence source
	evidenceSource := &models.EvidenceSource{
		Type:      "name_generator",
		Resource:  fmt.Sprintf("%s:%s", docType, refID),
		Content:   string(resultJSON),
		Relevance: 1.0,
	}

	return string(resultJSON), evidenceSource, nil
}

// processBatch generates names for all documents of a type
func (ngt *NameGeneratorTool) processBatch(ctx context.Context, docType string) (string, *models.EvidenceSource, error) {
	// This would implement batch processing of all documents
	// For now, return a placeholder
	return fmt.Sprintf("Batch processing for %s documents not yet implemented", docType), nil, nil
}

// DocumentDetails holds information about a document
type DocumentDetails struct {
	Type        string
	ReferenceID string
	CurrentName string
	Description string
	Category    string
	Framework   string
}

// getDocumentDetails retrieves details about a document
func (ngt *NameGeneratorTool) getDocumentDetails(docType, refID string) (*DocumentDetails, error) {
	switch docType {
	case "evidence":
		return ngt.getEvidenceTaskDetails(refID)
	case "control":
		return ngt.getControlDetails(refID)
	case "policy":
		return ngt.getPolicyDetails(refID)
	default:
		return nil, fmt.Errorf("unsupported document type: %s", docType)
	}
}

// getEvidenceTaskDetails retrieves evidence task details
func (ngt *NameGeneratorTool) getEvidenceTaskDetails(refID string) (*DocumentDetails, error) {
	// Parse reference to get numeric ID (using existing validator logic)
	validationResult, err := ngt.validator.ValidateTaskReference(refID)
	if err != nil || !validationResult.Valid {
		return nil, fmt.Errorf("invalid task reference: %s", refID)
	}

	// Get normalized ID for lookup
	normalizedRef := validationResult.Normalized["task_ref"]

	// Try to get task from data store
	task, err := ngt.dataStore.GetEvidenceTask(normalizedRef)
	if err != nil || task == nil {
		return nil, fmt.Errorf("evidence task not found: %s", refID)
	}

	return &DocumentDetails{
		Type:        "evidence",
		ReferenceID: refID,
		CurrentName: task.Name,
		Description: task.Description,
		Category:    task.Category,
		Framework:   task.Framework,
	}, nil
}

// getControlDetails retrieves control details
func (ngt *NameGeneratorTool) getControlDetails(refID string) (*DocumentDetails, error) {
	// This would implement control detail retrieval
	// For now, return a placeholder
	return &DocumentDetails{
		Type:        "control",
		ReferenceID: refID,
		CurrentName: fmt.Sprintf("Control %s", refID),
		Description: "Control description would be loaded from file",
	}, nil
}

// getPolicyDetails retrieves policy details
func (ngt *NameGeneratorTool) getPolicyDetails(refID string) (*DocumentDetails, error) {
	// This would implement policy detail retrieval
	// For now, return a placeholder
	return &DocumentDetails{
		Type:        "policy",
		ReferenceID: refID,
		CurrentName: fmt.Sprintf("Policy %s", refID),
		Description: "Policy description would be loaded from file",
	}, nil
}

// generateShortName uses Claude to generate a concise name
func (ngt *NameGeneratorTool) generateShortName(ctx context.Context, details *DocumentDetails) (string, error) {
	// This would implement the actual Claude API call using prompt-as-data
	// For now, generate a simple name based on the current name

	// Basic name generation logic as placeholder
	name := strings.ToLower(details.CurrentName)

	// Remove common words
	commonWords := []string{"population", "list", "of", "the", "and", "a", "an", "for", "in", "on", "at", "to", "from", "with", "by"}
	words := strings.Fields(name)
	var filteredWords []string

	for _, word := range words {
		isCommon := false
		for _, common := range commonWords {
			if word == common {
				isCommon = true
				break
			}
		}
		if !isCommon && len(word) > 2 {
			filteredWords = append(filteredWords, word)
		}
	}

	// Take first 4 words and join with underscores
	if len(filteredWords) > 4 {
		filteredWords = filteredWords[:4]
	}

	result := strings.Join(filteredWords, "_")

	// Clean up the result
	result = strings.ReplaceAll(result, "-", "_")
	result = strings.ReplaceAll(result, " ", "_")

	// Limit to 40 characters
	if len(result) > 40 {
		result = result[:40]
	}

	// Remove trailing underscores
	result = strings.TrimRight(result, "_")

	return result, nil
}
