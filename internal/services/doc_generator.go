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

//go:embed templates/evidence-workflow.tmpl
var evidenceWorkflowTemplate string

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
	data := TemplateData{
		Timestamp: time.Now().Format("2006-01-02 15:04:05 MST"),
		Version:   version,
		DataDir:   cfg.Storage.DataDir,
		Config:    cfg,
	}

	// Generate each documentation file
	generators := []struct {
		filename     string
		templateText string
	}{
		{"evidence-workflow.md", evidenceWorkflowTemplate},
		// TODO: Add other templates once extracted
		// {"directory-structure.md", directoryStructureTemplate},
		// {"tool-capabilities.md", toolCapabilitiesTemplate},
		// {"status-commands.md", statusCommandsTemplate},
		// {"submission-process.md", submissionProcessTemplate},
		// {"bulk-operations.md", bulkOperationsTemplate},
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
		defer f.Close()

		// Execute template
		if err := tmpl.Execute(f, data); err != nil {
			return fmt.Errorf("failed to execute template %s: %w", gen.filename, err)
		}
	}

	// For now, generate the other docs using the old method (temporary fallback)
	if err := generateRemainingDocs(cfg, version, docsDir); err != nil {
		return fmt.Errorf("failed to generate remaining docs: %w", err)
	}

	return nil
}

// generateRemainingDocs generates docs not yet converted to templates
// TODO: Remove this once all templates are extracted
func generateRemainingDocs(cfg *config.Config, version string, docsDir string) error {
	// Import the old generators temporarily
	oldGenerators := []struct {
		filename string
		generate func(*config.Config, string) (string, error)
	}{
		{"directory-structure.md", generateDirectoryStructureDocs},
		{"tool-capabilities.md", generateToolCapabilitiesDocs},
		{"status-commands.md", generateStatusCommandsDocs},
		{"submission-process.md", generateSubmissionProcessDocs},
		{"bulk-operations.md", generateBulkOperationsDocs},
	}

	for _, gen := range oldGenerators {
		content, err := gen.generate(cfg, version)
		if err != nil {
			return fmt.Errorf("failed to generate %s: %w", gen.filename, err)
		}

		docPath := filepath.Join(docsDir, gen.filename)
		if err := os.WriteFile(docPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", gen.filename, err)
		}
	}

	return nil
}
