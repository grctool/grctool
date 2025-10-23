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
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"gopkg.in/yaml.v3"
)

// GoogleEvidenceMappingsLoader handles loading and caching of Google Workspace evidence mappings
type GoogleEvidenceMappingsLoader struct {
	config     *config.Config
	logger     logger.Logger
	cache      *GoogleEvidenceMappings
	cacheMutex sync.RWMutex
	lastLoaded time.Time
}

// NewGoogleEvidenceMappingsLoader creates a new mappings loader
func NewGoogleEvidenceMappingsLoader(cfg *config.Config, log logger.Logger) *GoogleEvidenceMappingsLoader {
	return &GoogleEvidenceMappingsLoader{
		config: cfg,
		logger: log,
	}
}

// GoogleEvidenceMappings represents the complete evidence mapping configuration
type GoogleEvidenceMappings struct {
	GoogleWorkspace  GoogleWorkspaceConfig      `yaml:"google_workspace"`
	EvidenceMappings map[string]EvidenceMapping `yaml:"evidence_mappings"`
	Metadata         MappingMetadata            `yaml:"metadata"`
	CacheSettings    CacheSettings              `yaml:"cache_settings"`
}

// GoogleWorkspaceConfig holds global Google Workspace configuration
type GoogleWorkspaceConfig struct {
	DefaultExtractionRules GoogleExtractionRules `yaml:"default_extraction_rules"`
	Auth                   AuthConfig            `yaml:"auth"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	CredentialsPath string   `yaml:"credentials_path"`
	Scopes          []string `yaml:"scopes"`
}

// EvidenceMapping represents a mapping between an evidence task and Google Workspace documents
type EvidenceMapping struct {
	TaskRef     string           `yaml:"task_ref"`
	Description string           `yaml:"description"`
	SourceType  string           `yaml:"source_type"`
	Priority    string           `yaml:"priority"`
	Documents   []DocumentConfig `yaml:"documents"`
}

// DocumentConfig represents configuration for a specific Google Workspace document
type DocumentConfig struct {
	DocumentID      string                `yaml:"document_id"`
	DocumentName    string                `yaml:"document_name"`
	DocumentType    string                `yaml:"document_type"`
	ExtractionRules GoogleExtractionRules `yaml:"extraction_rules"`
	Validation      GoogleValidationRules `yaml:"validation"`
}

// GoogleExtractionRules defines how content should be extracted from documents
type GoogleExtractionRules struct {
	// Basic extraction settings
	IncludeMetadata  bool `yaml:"include_metadata"`
	IncludeRevisions bool `yaml:"include_revisions"`
	MaxResults       int  `yaml:"max_results"`

	// Document-specific settings
	SheetRange  string   `yaml:"sheet_range,omitempty"`
	SearchQuery string   `yaml:"search_query,omitempty"`
	FileTypes   []string `yaml:"file_types,omitempty"`

	// Content filtering
	ContentFilters []ContentFilter `yaml:"content_filters,omitempty"`
	FolderFilters  *FolderFilters  `yaml:"folder_filters,omitempty"`

	// Structured data extraction (for sheets)
	ColumnMapping  map[string]string `yaml:"column_mapping,omitempty"`
	DataValidation *DataValidation   `yaml:"data_validation,omitempty"`
	RowFilters     *RowFilters       `yaml:"row_filters,omitempty"`
	Aggregations   []Aggregation     `yaml:"aggregations,omitempty"`

	// Forms-specific settings
	IncludeResponses   bool                `yaml:"include_responses,omitempty"`
	FormAnalysis       *FormAnalysis       `yaml:"form_analysis,omitempty"`
	ResponseProcessing *ResponseProcessing `yaml:"response_processing,omitempty"`

	// Access review specific settings
	AccessReviewRules *AccessReviewRules `yaml:"access_review_rules,omitempty"`

	// Required metadata fields
	RequiredMetadata []string `yaml:"required_metadata,omitempty"`
}

// ContentFilter defines patterns to extract specific content sections
type ContentFilter struct {
	Section string `yaml:"section"`
	Pattern string `yaml:"pattern"`
}

// FolderFilters defines how to filter folder contents
type FolderFilters struct {
	IncludePatterns []string    `yaml:"include_patterns"`
	ExcludePatterns []string    `yaml:"exclude_patterns"`
	DateFilter      *DateFilter `yaml:"date_filter,omitempty"`
}

// DateFilter defines date-based filtering
type DateFilter struct {
	Field  string `yaml:"field"`
	After  string `yaml:"after,omitempty"`
	Before string `yaml:"before,omitempty"`
}

// DataValidation defines validation rules for structured data
type DataValidation struct {
	RequiredColumns []string            `yaml:"required_columns"`
	DateColumns     []string            `yaml:"date_columns"`
	NumericColumns  []string            `yaml:"numeric_columns"`
	DateFormat      string              `yaml:"date_format,omitempty"`
	EnumColumns     map[string][]string `yaml:"enum_columns,omitempty"`
}

// RowFilters defines how to filter spreadsheet rows
type RowFilters struct {
	StatusColumn  string     `yaml:"status_column,omitempty"`
	IncludeStatus []string   `yaml:"include_status,omitempty"`
	DateRange     *DateRange `yaml:"date_range,omitempty"`
}

// DateRange defines a date range filter
type DateRange struct {
	Column string `yaml:"column"`
	From   string `yaml:"from"`
	To     string `yaml:"to"`
}

// Aggregation defines data aggregation rules
type Aggregation struct {
	Type    string `yaml:"type"`
	GroupBy string `yaml:"group_by"`
}

// FormAnalysis defines analysis rules for Google Forms
type FormAnalysis struct {
	RequiredQuestions     []string               `yaml:"required_questions"`
	GoogleValidationRules []GoogleValidationRule `yaml:"validation_rules"`
}

// GoogleValidationRule defines a validation rule for form fields
type GoogleValidationRule struct {
	Field        string     `yaml:"field"`
	RequiredText string     `yaml:"required_text,omitempty"`
	Format       string     `yaml:"format,omitempty"`
	Range        *DateRange `yaml:"range,omitempty"`
}

// ResponseProcessing defines how to process form responses
type ResponseProcessing struct {
	MaxResponses          int      `yaml:"max_responses"`
	IncludeMetadata       bool     `yaml:"include_metadata"`
	AnonymizePersonalData bool     `yaml:"anonymize_personal_data"`
	ExtractFields         []string `yaml:"extract_fields"`
}

// AccessReviewRules defines rules specific to access reviews
type AccessReviewRules struct {
	CertificationRequirements []CertificationRequirement `yaml:"certification_requirements"`
	RiskIndicators            []RiskIndicator            `yaml:"risk_indicators"`
}

// CertificationRequirement defines requirements for access certification
type CertificationRequirement struct {
	Field          string   `yaml:"field"`
	RequiredValues []string `yaml:"required_values,omitempty"`
	CannotBeEmpty  bool     `yaml:"cannot_be_empty,omitempty"`
	WithinPeriod   int      `yaml:"within_period,omitempty"` // days
}

// RiskIndicator defines conditions that indicate risk
type RiskIndicator struct {
	Condition string `yaml:"condition"`
	Flag      string `yaml:"flag"`
}

// GoogleValidationRules defines validation requirements for documents
type GoogleValidationRules struct {
	MinContentLength          int        `yaml:"min_content_length,omitempty"`
	RequiredKeywords          []string   `yaml:"required_keywords,omitempty"`
	DateRange                 *DateRange `yaml:"date_range,omitempty"`
	Frequency                 string     `yaml:"frequency,omitempty"`
	MinRows                   int        `yaml:"min_rows,omitempty"`
	RequiredHeaders           []string   `yaml:"required_headers,omitempty"`
	MinResponses              int        `yaml:"min_responses,omitempty"`
	ResponseCompleteness      float64    `yaml:"response_completeness,omitempty"`
	CertificationCompleteness float64    `yaml:"certification_completeness,omitempty"`
	ReviewerCoverage          float64    `yaml:"reviewer_coverage,omitempty"`
	RequiredTrainingModules   []string   `yaml:"required_training_modules,omitempty"`
}

// MappingMetadata holds metadata about the mappings configuration
type MappingMetadata struct {
	Version              string                           `yaml:"version"`
	CreatedDate          string                           `yaml:"created_date"`
	UpdatedDate          string                           `yaml:"updated_date"`
	CreatedBy            string                           `yaml:"created_by"`
	RefreshSchedule      map[string]string                `yaml:"refresh_schedule"`
	ComplianceFrameworks map[string][]map[string][]string `yaml:"compliance_frameworks"`
}

// CacheSettings defines caching configuration
type CacheSettings struct {
	EnableContentCache    bool       `yaml:"enable_content_cache"`
	CacheDuration         string     `yaml:"cache_duration"`
	EnableIncrementalSync bool       `yaml:"enable_incremental_sync"`
	RateLimits            RateLimits `yaml:"rate_limits"`
}

// RateLimits defines API rate limiting settings
type RateLimits struct {
	RequestsPerMinute  int `yaml:"requests_per_minute"`
	ConcurrentRequests int `yaml:"concurrent_requests"`
}

// LoadMappings loads the evidence mappings from the YAML configuration file
func (loader *GoogleEvidenceMappingsLoader) LoadMappings() (*GoogleEvidenceMappings, error) {
	loader.cacheMutex.RLock()
	// Return cached version if available and recent
	if loader.cache != nil && time.Since(loader.lastLoaded) < 5*time.Minute {
		defer loader.cacheMutex.RUnlock()
		return loader.cache, nil
	}
	loader.cacheMutex.RUnlock()

	// Determine the configuration file path
	configPath := loader.getMappingsFilePath()

	loader.logger.Debug("Loading Google evidence mappings", logger.String("path", configPath))

	// Read the YAML file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read mappings file %s: %w", configPath, err)
	}

	// Parse the YAML
	var mappings GoogleEvidenceMappings
	if err := yaml.Unmarshal(data, &mappings); err != nil {
		return nil, fmt.Errorf("failed to parse mappings YAML: %w", err)
	}

	// Validate the mappings
	if err := loader.validateMappings(&mappings); err != nil {
		return nil, fmt.Errorf("invalid mappings configuration: %w", err)
	}

	// Apply defaults
	loader.applyDefaults(&mappings)

	// Cache the results
	loader.cacheMutex.Lock()
	loader.cache = &mappings
	loader.lastLoaded = time.Now()
	loader.cacheMutex.Unlock()

	loader.logger.Info("Loaded Google evidence mappings",
		logger.Int("mapping_count", len(mappings.EvidenceMappings)),
		logger.String("version", mappings.Metadata.Version))

	return &mappings, nil
}

// GetMappingForTask returns the evidence mapping for a specific task reference
func (loader *GoogleEvidenceMappingsLoader) GetMappingForTask(taskRef string) (*EvidenceMapping, error) {
	mappings, err := loader.LoadMappings()
	if err != nil {
		return nil, err
	}

	mapping, exists := mappings.EvidenceMappings[taskRef]
	if !exists {
		return nil, fmt.Errorf("no mapping found for task reference: %s", taskRef)
	}

	return &mapping, nil
}

// GetAllMappings returns all evidence mappings
func (loader *GoogleEvidenceMappingsLoader) GetAllMappings() (map[string]EvidenceMapping, error) {
	mappings, err := loader.LoadMappings()
	if err != nil {
		return nil, err
	}

	return mappings.EvidenceMappings, nil
}

// GetMappingsByPriority returns mappings filtered by priority
func (loader *GoogleEvidenceMappingsLoader) GetMappingsByPriority(priority string) (map[string]EvidenceMapping, error) {
	allMappings, err := loader.GetAllMappings()
	if err != nil {
		return nil, err
	}

	filtered := make(map[string]EvidenceMapping)
	for taskRef, mapping := range allMappings {
		if mapping.Priority == priority {
			filtered[taskRef] = mapping
		}
	}

	return filtered, nil
}

// GetMappingsBySourceType returns mappings filtered by source type
func (loader *GoogleEvidenceMappingsLoader) GetMappingsBySourceType(sourceType string) (map[string]EvidenceMapping, error) {
	allMappings, err := loader.GetAllMappings()
	if err != nil {
		return nil, err
	}

	filtered := make(map[string]EvidenceMapping)
	for taskRef, mapping := range allMappings {
		if mapping.SourceType == sourceType {
			filtered[taskRef] = mapping
		}
	}

	return filtered, nil
}

// ValidateDocumentAccess validates that the specified documents are accessible
func (loader *GoogleEvidenceMappingsLoader) ValidateDocumentAccess(mapping *EvidenceMapping) error {
	loader.logger.Debug("Validating document access",
		logger.String("task_ref", mapping.TaskRef),
		logger.Int("document_count", len(mapping.Documents)))

	for _, doc := range mapping.Documents {
		if doc.DocumentID == "" {
			return fmt.Errorf("document_id is required for mapping %s", mapping.TaskRef)
		}

		if doc.DocumentType == "" {
			return fmt.Errorf("document_type is required for document %s in mapping %s",
				doc.DocumentID, mapping.TaskRef)
		}

		// Validate document type
		validTypes := []string{"docs", "sheets", "forms", "drive"}
		validType := false
		for _, vt := range validTypes {
			if doc.DocumentType == vt {
				validType = true
				break
			}
		}
		if !validType {
			return fmt.Errorf("invalid document_type '%s' for document %s in mapping %s",
				doc.DocumentType, doc.DocumentID, mapping.TaskRef)
		}
	}

	return nil
}

// TransformToGoogleAPIParams transforms mapping rules into parameters for Google API calls
func (loader *GoogleEvidenceMappingsLoader) TransformToGoogleAPIParams(mapping *EvidenceMapping, documentIndex int) (map[string]interface{}, error) {
	if documentIndex >= len(mapping.Documents) {
		return nil, fmt.Errorf("document index %d out of range for mapping %s", documentIndex, mapping.TaskRef)
	}

	doc := mapping.Documents[documentIndex]
	params := make(map[string]interface{})

	// Basic parameters
	params["document_id"] = doc.DocumentID
	params["document_type"] = doc.DocumentType

	// Extraction rules
	extractionRules := make(map[string]interface{})
	extractionRules["include_metadata"] = doc.ExtractionRules.IncludeMetadata
	extractionRules["include_revisions"] = doc.ExtractionRules.IncludeRevisions

	if doc.ExtractionRules.MaxResults > 0 {
		extractionRules["max_results"] = doc.ExtractionRules.MaxResults
	}

	if doc.ExtractionRules.SheetRange != "" {
		extractionRules["sheet_range"] = doc.ExtractionRules.SheetRange
	}

	if doc.ExtractionRules.SearchQuery != "" {
		extractionRules["search_query"] = doc.ExtractionRules.SearchQuery
	}

	params["extraction_rules"] = extractionRules

	// Add credentials path if available
	mappings, err := loader.LoadMappings()
	if err == nil && mappings.GoogleWorkspace.Auth.CredentialsPath != "" {
		params["credentials_path"] = mappings.GoogleWorkspace.Auth.CredentialsPath
	}

	return params, nil
}

// GetRefreshSchedule returns the refresh schedule for a specific task
func (loader *GoogleEvidenceMappingsLoader) GetRefreshSchedule(taskRef string) (string, error) {
	mappings, err := loader.LoadMappings()
	if err != nil {
		return "", err
	}

	if schedule, exists := mappings.Metadata.RefreshSchedule[taskRef]; exists {
		return schedule, nil
	}

	return "monthly", nil // default
}

// getMappingsFilePath determines the path to the mappings configuration file
func (loader *GoogleEvidenceMappingsLoader) getMappingsFilePath() string {
	// Check if a custom path is configured
	if loader.config != nil && loader.config.Storage.DataDir != "" {
		// Try in the data directory first
		dataPath := filepath.Join(loader.config.Storage.DataDir, "google_evidence_mappings.yaml")
		if _, err := os.Stat(dataPath); err == nil {
			return dataPath
		}
	}

	// Try in the current directory (grctool root)
	currentPath := "google_evidence_mappings.yaml"
	if _, err := os.Stat(currentPath); err == nil {
		return currentPath
	}

	// Try in a configs directory
	configPath := filepath.Join("configs", "google_evidence_mappings.yaml")
	if _, err := os.Stat(configPath); err == nil {
		return configPath
	}

	// Default to current directory
	return currentPath
}

// validateMappings validates the loaded mappings configuration
func (loader *GoogleEvidenceMappingsLoader) validateMappings(mappings *GoogleEvidenceMappings) error {
	if len(mappings.EvidenceMappings) == 0 {
		return fmt.Errorf("no evidence mappings defined")
	}

	for taskRef, mapping := range mappings.EvidenceMappings {
		if mapping.TaskRef != taskRef {
			return fmt.Errorf("task_ref mismatch for mapping %s", taskRef)
		}

		if mapping.SourceType == "" {
			return fmt.Errorf("source_type is required for mapping %s", taskRef)
		}

		if len(mapping.Documents) == 0 {
			return fmt.Errorf("at least one document is required for mapping %s", taskRef)
		}

		// Validate each document
		for i, doc := range mapping.Documents {
			if doc.DocumentID == "" {
				return fmt.Errorf("document_id is required for document %d in mapping %s", i, taskRef)
			}

			if doc.DocumentType == "" {
				return fmt.Errorf("document_type is required for document %d in mapping %s", i, taskRef)
			}
		}
	}

	return nil
}

// applyDefaults applies default values to mappings configuration
func (loader *GoogleEvidenceMappingsLoader) applyDefaults(mappings *GoogleEvidenceMappings) {
	// Apply default extraction rules where not specified
	for taskRef, mapping := range mappings.EvidenceMappings {
		for i := range mapping.Documents {
			doc := &mapping.Documents[i]

			// Apply default extraction rules if not set
			if doc.ExtractionRules.MaxResults == 0 {
				if mappings.GoogleWorkspace.DefaultExtractionRules.MaxResults > 0 {
					doc.ExtractionRules.MaxResults = mappings.GoogleWorkspace.DefaultExtractionRules.MaxResults
				} else {
					doc.ExtractionRules.MaxResults = 20 // fallback default
				}
			}

			// Set default metadata inclusion if not specified
			if !doc.ExtractionRules.IncludeMetadata && mappings.GoogleWorkspace.DefaultExtractionRules.IncludeMetadata {
				doc.ExtractionRules.IncludeMetadata = true
			}
		}

		// Update the mapping in place
		mappings.EvidenceMappings[taskRef] = mapping
	}

	// Apply default cache settings
	if mappings.CacheSettings.CacheDuration == "" {
		mappings.CacheSettings.CacheDuration = "24h"
	}

	if mappings.CacheSettings.RateLimits.RequestsPerMinute == 0 {
		mappings.CacheSettings.RateLimits.RequestsPerMinute = 60
	}

	if mappings.CacheSettings.RateLimits.ConcurrentRequests == 0 {
		mappings.CacheSettings.RateLimits.ConcurrentRequests = 5
	}
}

// ClearCache clears the cached mappings, forcing a reload on next access
func (loader *GoogleEvidenceMappingsLoader) ClearCache() {
	loader.cacheMutex.Lock()
	defer loader.cacheMutex.Unlock()

	loader.cache = nil
	loader.lastLoaded = time.Time{}

	loader.logger.Debug("Cleared Google evidence mappings cache")
}

// GetSupportedTaskRefs returns a list of all supported task references
func (loader *GoogleEvidenceMappingsLoader) GetSupportedTaskRefs() ([]string, error) {
	mappings, err := loader.LoadMappings()
	if err != nil {
		return nil, err
	}

	var taskRefs []string
	for taskRef := range mappings.EvidenceMappings {
		taskRefs = append(taskRefs, taskRef)
	}

	return taskRefs, nil
}

// GetComplianceFrameworkMappings returns mappings for a specific compliance framework
func (loader *GoogleEvidenceMappingsLoader) GetComplianceFrameworkMappings(framework string) (map[string][]string, error) {
	mappings, err := loader.LoadMappings()
	if err != nil {
		return nil, err
	}

	if mappings.Metadata.ComplianceFrameworks == nil {
		return nil, fmt.Errorf("no compliance framework mappings available")
	}

	frameworkMappings, exists := mappings.Metadata.ComplianceFrameworks[framework]
	if !exists {
		return nil, fmt.Errorf("compliance framework '%s' not found", framework)
	}

	// Flatten the mapping structure
	result := make(map[string][]string)
	for _, mapping := range frameworkMappings {
		for taskRef, controls := range mapping {
			result[taskRef] = controls
		}
	}

	return result, nil
}
