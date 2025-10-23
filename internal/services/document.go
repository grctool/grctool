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

package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/markdown"
)

// DocumentService provides shared document generation utilities
type DocumentService struct {
	baseDir     string
	mdFormatter *markdown.Formatter
	paths       config.StoragePaths // Configured paths for documents
}

// NewDocumentService creates a new document service
func NewDocumentService(cfg *config.Config) *DocumentService {
	// Apply defaults and resolve paths
	paths := cfg.Storage.Paths.WithDefaults().ResolveRelativeTo(cfg.Storage.DataDir)

	return &DocumentService{
		baseDir:     cfg.Storage.DataDir,
		mdFormatter: markdown.NewFormatter(markdown.DefaultConfig()),
		paths:       paths,
	}
}

// DocumentType represents the type of document being generated
type DocumentType string

const (
	PolicyDocument         DocumentType = "policy_documents"
	ControlDocument        DocumentType = "control_documents"
	EvidenceTaskDocument   DocumentType = "evidence_task_documents"
	EvidencePromptDocument DocumentType = "evidence_prompts"
)

// getDocumentPath returns the configured path for a document type
func (ds *DocumentService) getDocumentPath(docType DocumentType) string {
	switch docType {
	case PolicyDocument:
		return ds.paths.PoliciesMarkdown
	case ControlDocument:
		return ds.paths.ControlsMarkdown
	case EvidenceTaskDocument:
		return ds.paths.EvidenceTasksMarkdown
	case EvidencePromptDocument:
		return ds.paths.EvidencePrompts
	default:
		// Fallback to old behavior if unknown type
		return filepath.Join(ds.baseDir, string(docType))
	}
}

// GenerateDocument creates a document file with the given content
func (ds *DocumentService) GenerateDocument(docType DocumentType, filename, content string) error {
	// Get configured path for this document type
	documentsDir := ds.getDocumentPath(docType)
	if err := os.MkdirAll(documentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create %s directory: %w", docType, err)
	}

	// Format markdown content if it's a markdown file
	if strings.HasSuffix(filename, ".md") {
		content = ds.mdFormatter.FormatDocument(content)
	}

	// Write the document to file
	filePath := filepath.Join(documentsDir, filename)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write document to %s: %w", filePath, err)
	}

	return nil
}

// EnsureDocumentDirectory creates the document directory if it doesn't exist
func (ds *DocumentService) EnsureDocumentDirectory(docType DocumentType) error {
	documentsDir := ds.getDocumentPath(docType)
	if err := os.MkdirAll(documentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create %s directory: %w", docType, err)
	}
	return nil
}

// GetDocumentPath returns the full path for a document
func (ds *DocumentService) GetDocumentPath(docType DocumentType, filename string) string {
	return filepath.Join(ds.getDocumentPath(docType), filename)
}
