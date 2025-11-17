// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package services

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/grctool/grctool/internal/config"
)

// Embed all documentation templates
//
//go:embed templates/directory-structure.tmpl
var directoryStructureTemplate string

//go:embed templates/evidence-workflow.tmpl
var evidenceWorkflowTemplate string

//go:embed templates/tool-capabilities.tmpl
var toolCapabilitiesTemplate string

//go:embed templates/status-commands.tmpl
var statusCommandsTemplate string

//go:embed templates/submission-process.tmpl
var submissionProcessTemplate string

//go:embed templates/bulk-operations.tmpl
var bulkOperationsTemplate string

// TemplateData holds data for template rendering
type TemplateData struct {
	Timestamp string
	Version   string
	DataDir   string
	Config    *config.Config
}

// GenerateAgentDocs generates all agent documentation files using templates
func GenerateAgentDocs(cfg *config.Config, version string) error {
	// Create docs directory
	docsDir := filepath.Join(".grctool", "docs")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		return fmt.Errorf("failed to create docs directory: %w", err)
	}

	// Prepare template data
	dataDir := cfg.Storage.DataDir
	if dataDir == "" {
		dataDir = "./data"
	}

	data := TemplateData{
		Timestamp: time.Now().Format("2006-01-02 15:04:05 MST"),
		Version:   version,
		DataDir:   dataDir,
		Config:    cfg,
	}

	// Generate each documentation file from templates
	generators := []struct {
		filename     string
		templateText string
	}{
		{"directory-structure.md", directoryStructureTemplate},
		{"evidence-workflow.md", evidenceWorkflowTemplate},
		{"tool-capabilities.md", toolCapabilitiesTemplate},
		{"status-commands.md", statusCommandsTemplate},
		{"submission-process.md", submissionProcessTemplate},
		{"bulk-operations.md", bulkOperationsTemplate},
	}

	for _, gen := range generators {
		// Parse template
		tmpl, err := template.New(gen.filename).Parse(gen.templateText)
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", gen.filename, err)
		}

		// Create output file
		docPath := filepath.Join(docsDir, gen.filename)
		f, err := os.Create(docPath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", gen.filename, err)
		}

		// Execute template
		if err := tmpl.Execute(f, data); err != nil {
			f.Close()
			return fmt.Errorf("failed to execute template %s: %w", gen.filename, err)
		}

		if err := f.Close(); err != nil {
			return fmt.Errorf("failed to close file %s: %w", gen.filename, err)
		}
	}

	return nil
}
