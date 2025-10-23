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

package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/storage"
)

// ServiceImpl implements the validation Service interface
type ServiceImpl struct {
	validator DataValidator
	config    *config.Config
	logger    logger.Logger
}

// NewService creates a new validation service implementation
func NewService(storage *storage.Storage, cfg *config.Config, log logger.Logger) (Service, error) {
	validator := NewDataValidator(storage, cfg.Storage.DataDir)

	return &ServiceImpl{
		validator: validator,
		config:    cfg,
		logger:    log,
	}, nil
}

// ValidateAll performs comprehensive data validation
func (s *ServiceImpl) ValidateAll(ctx context.Context) (*ValidationReport, error) {
	return s.validator.ValidateAll(ctx)
}

// ValidatePolicies validates policy data
func (s *ServiceImpl) ValidatePolicies(ctx context.Context) (*PolicyValidation, error) {
	validation, _, err := s.validator.ValidatePolicies(ctx)
	return validation, err
}

// ValidateControls validates control data
func (s *ServiceImpl) ValidateControls(ctx context.Context) (*ControlValidation, error) {
	validation, _, err := s.validator.ValidateControls(ctx)
	return validation, err
}

// ValidateEvidenceTasks validates evidence task data
func (s *ServiceImpl) ValidateEvidenceTasks(ctx context.Context) (*EvidenceValidation, error) {
	validation, _, err := s.validator.ValidateEvidenceTasks(ctx)
	return validation, err
}

// ValidateRelationships validates relationships between entities
func (s *ServiceImpl) ValidateRelationships(ctx context.Context) (*RelationshipValidation, error) {
	validation, _, err := s.validator.ValidateRelationships(ctx)
	return validation, err
}

// GenerateReport generates a validation report with options
func (s *ServiceImpl) GenerateReport(ctx context.Context, options ValidationOptions) (*ValidationReport, error) {
	report, err := s.ValidateAll(ctx)
	if err != nil {
		return nil, err
	}

	// Apply options filters
	if !options.IncludeInfo {
		filteredIssues := []ValidationIssue{}
		for _, issue := range report.Issues {
			if issue.Type != "info" {
				filteredIssues = append(filteredIssues, issue)
			}
		}
		report.Issues = filteredIssues
	}

	return report, nil
}

// FormatReport formats a validation report in the specified format
func (s *ServiceImpl) FormatReport(report *ValidationReport, format string) (string, error) {
	switch strings.ToLower(format) {
	case "json":
		jsonData, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal JSON: %w", err)
		}
		return string(jsonData), nil
	case "text", "":
		return s.formatTextReport(report), nil
	case "markdown":
		return s.formatMarkdownReport(report), nil
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

// SaveReport saves a validation report to file
func (s *ServiceImpl) SaveReport(report *ValidationReport, outputPath string, format string) error {
	content, err := s.FormatReport(report, format)
	if err != nil {
		return err
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create report file: %w", err)
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("failed to write report content: %w", err)
	}

	return nil
}

// formatTextReport formats report as plain text
func (s *ServiceImpl) formatTextReport(report *ValidationReport) string {
	var result strings.Builder

	result.WriteString("=== DATA VALIDATION REPORT ===\n\n")

	// Summary
	result.WriteString("SUMMARY:\n")
	result.WriteString(fmt.Sprintf("  Overall Status: %s\n", strings.ToUpper(report.Summary.Status)))
	result.WriteString(fmt.Sprintf("  Overall Score: %.1f%%\n", report.Summary.OverallScore))
	result.WriteString(fmt.Sprintf("  Total Policies: %d\n", report.Summary.TotalPolicies))
	result.WriteString(fmt.Sprintf("  Total Controls: %d\n", report.Summary.TotalControls))
	result.WriteString(fmt.Sprintf("  Total Evidence Tasks: %d\n", report.Summary.TotalEvidenceTasks))
	result.WriteString(fmt.Sprintf("  Critical Issues: %d\n", report.Summary.CriticalIssues))
	result.WriteString(fmt.Sprintf("  Warnings: %d\n\n", report.Summary.Warnings))

	// Detailed sections
	result.WriteString("POLICY VALIDATION:\n")
	result.WriteString(fmt.Sprintf("  Content Completeness: %.1f%%\n", report.Policies.ContentCompleteness))
	result.WriteString(fmt.Sprintf("  Policies with Content: %d\n", report.Policies.WithContent))
	result.WriteString(fmt.Sprintf("  Policies with Substantial Content: %d\n", report.Policies.WithSubstantialContent))
	result.WriteString(fmt.Sprintf("  Missing Content: %d\n\n", report.Policies.MissingContent))

	result.WriteString("CONTROL VALIDATION:\n")
	result.WriteString(fmt.Sprintf("  Linkage Completeness: %.1f%%\n", report.Controls.LinkageCompleteness))
	result.WriteString(fmt.Sprintf("  Controls with Policy Links: %d\n", report.Controls.WithPolicyLinks))
	result.WriteString(fmt.Sprintf("  Missing Policy Links: %d\n\n", report.Controls.MissingPolicyLinks))

	result.WriteString("EVIDENCE VALIDATION:\n")
	result.WriteString(fmt.Sprintf("  Content Completeness: %.1f%%\n", report.Evidence.ContentCompleteness))
	result.WriteString(fmt.Sprintf("  Tasks with Guidance: %d\n", report.Evidence.WithGuidance))
	result.WriteString(fmt.Sprintf("  Missing Guidance: %d\n\n", report.Evidence.MissingGuidance))

	// Issues
	if len(report.Issues) > 0 {
		result.WriteString("VALIDATION ISSUES:\n")
		for _, issue := range report.Issues {
			result.WriteString(fmt.Sprintf("  [%s] %s: %s\n",
				strings.ToUpper(issue.Type), issue.Category, issue.Description))
		}
	}

	return result.String()
}

// formatMarkdownReport formats report as markdown
func (s *ServiceImpl) formatMarkdownReport(report *ValidationReport) string {
	var result strings.Builder

	result.WriteString("# Data Validation Report\n\n")

	// Summary
	result.WriteString("## Summary\n\n")
	result.WriteString(fmt.Sprintf("- **Overall Status:** %s\n", strings.ToUpper(report.Summary.Status)))
	result.WriteString(fmt.Sprintf("- **Overall Score:** %.1f%%\n", report.Summary.OverallScore))
	result.WriteString(fmt.Sprintf("- **Total Policies:** %d\n", report.Summary.TotalPolicies))
	result.WriteString(fmt.Sprintf("- **Total Controls:** %d\n", report.Summary.TotalControls))
	result.WriteString(fmt.Sprintf("- **Total Evidence Tasks:** %d\n", report.Summary.TotalEvidenceTasks))
	result.WriteString(fmt.Sprintf("- **Critical Issues:** %d\n", report.Summary.CriticalIssues))
	result.WriteString(fmt.Sprintf("- **Warnings:** %d\n\n", report.Summary.Warnings))

	// Detailed sections
	result.WriteString("## Policy Validation\n\n")
	result.WriteString(fmt.Sprintf("- Content Completeness: %.1f%%\n", report.Policies.ContentCompleteness))
	result.WriteString(fmt.Sprintf("- Policies with Content: %d\n", report.Policies.WithContent))
	result.WriteString(fmt.Sprintf("- Policies with Substantial Content: %d\n", report.Policies.WithSubstantialContent))
	result.WriteString(fmt.Sprintf("- Missing Content: %d\n\n", report.Policies.MissingContent))

	result.WriteString("## Control Validation\n\n")
	result.WriteString(fmt.Sprintf("- Linkage Completeness: %.1f%%\n", report.Controls.LinkageCompleteness))
	result.WriteString(fmt.Sprintf("- Controls with Policy Links: %d\n", report.Controls.WithPolicyLinks))
	result.WriteString(fmt.Sprintf("- Missing Policy Links: %d\n\n", report.Controls.MissingPolicyLinks))

	result.WriteString("## Evidence Validation\n\n")
	result.WriteString(fmt.Sprintf("- Content Completeness: %.1f%%\n", report.Evidence.ContentCompleteness))
	result.WriteString(fmt.Sprintf("- Tasks with Guidance: %d\n", report.Evidence.WithGuidance))
	result.WriteString(fmt.Sprintf("- Missing Guidance: %d\n\n", report.Evidence.MissingGuidance))

	// Issues
	if len(report.Issues) > 0 {
		result.WriteString("## Validation Issues\n\n")
		for _, issue := range report.Issues {
			result.WriteString(fmt.Sprintf("- **%s [%s]:** %s\n",
				strings.ToUpper(issue.Type), issue.Category, issue.Description))
		}
	}

	return result.String()
}

// DataValidatorImpl implements the DataValidator interface
type DataValidatorImpl struct {
	storage *storage.Storage
	dataDir string
}

// NewDataValidator creates a new data validator
func NewDataValidator(storage *storage.Storage, dataDir string) DataValidator {
	return &DataValidatorImpl{
		storage: storage,
		dataDir: dataDir,
	}
}

// ValidateAll performs comprehensive data validation
func (v *DataValidatorImpl) ValidateAll(ctx context.Context) (*ValidationReport, error) {
	report := &ValidationReport{
		Issues: []ValidationIssue{},
	}

	// Validate policies
	policyValidation, policyIssues, err := v.ValidatePolicies(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to validate policies: %w", err)
	}
	report.Policies = *policyValidation
	report.Issues = append(report.Issues, policyIssues...)

	// Validate controls
	controlValidation, controlIssues, err := v.ValidateControls(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to validate controls: %w", err)
	}
	report.Controls = *controlValidation
	report.Issues = append(report.Issues, controlIssues...)

	// Validate evidence tasks
	evidenceValidation, evidenceIssues, err := v.ValidateEvidenceTasks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to validate evidence: %w", err)
	}
	report.Evidence = *evidenceValidation
	report.Issues = append(report.Issues, evidenceIssues...)

	// Validate relationships
	relationshipValidation, relationshipIssues, err := v.ValidateRelationships(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to validate relationships: %w", err)
	}
	report.Relationships = *relationshipValidation
	report.Issues = append(report.Issues, relationshipIssues...)

	// Generate summary
	report.Summary = v.generateSummary(report)

	return report, nil
}

// ValidatePolicies validates policy data completeness and quality
func (v *DataValidatorImpl) ValidatePolicies(ctx context.Context) (*PolicyValidation, []ValidationIssue, error) {
	validation := &PolicyValidation{}
	var issues []ValidationIssue

	// Get all policies
	policies, err := v.storage.GetAllPolicies()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get policies: %w", err)
	}

	totalContentLength := 0
	for _, policy := range policies {
		// Check for missing description
		if policy.Description == "" {
			validation.MissingDescription++
			issues = append(issues, ValidationIssue{
				Type:        "warning",
				Category:    "policy",
				ID:          policy.ID,
				Description: fmt.Sprintf("Policy '%s' has no description", policy.Name),
			})
		}

		// Check policy content
		contentLength := len(policy.Description)
		totalContentLength += contentLength

		if contentLength == 0 {
			validation.MissingContent++
		} else {
			validation.WithContent++
			lines := len(strings.Split(policy.Description, "\n"))
			if lines >= 10 {
				validation.WithSubstantialContent++
			}
			if lines >= 100 {
				validation.WithExtensiveContent++
			}
		}
	}

	if len(policies) > 0 {
		validation.AverageContentLength = float64(totalContentLength) / float64(len(policies))
		validation.ContentCompleteness = float64(validation.WithContent) / float64(len(policies)) * 100
	}

	return validation, issues, nil
}

// ValidateControls validates control data completeness and relationships
func (v *DataValidatorImpl) ValidateControls(ctx context.Context) (*ControlValidation, []ValidationIssue, error) {
	validation := &ControlValidation{}
	var issues []ValidationIssue

	// Get all controls
	controls, err := v.storage.GetAllControls()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get controls: %w", err)
	}

	for _, control := range controls {
		// Check for missing description
		if control.Description == "" {
			validation.MissingDescription++
			issues = append(issues, ValidationIssue{
				Type:        "warning",
				Category:    "control",
				ID:          fmt.Sprintf("%d", control.ID),
				Description: fmt.Sprintf("Control '%s' has no description", control.Name),
			})
		} else {
			validation.WithDescription++
		}

		// Check for policy links via Associations
		// Check if the control has associated policies through its Associations field
		hasPolicyLinks := false
		if control.Associations != nil && control.Associations.Policies > 0 {
			hasPolicyLinks = true
		}

		if hasPolicyLinks {
			validation.WithPolicyLinks++
		} else {
			validation.MissingPolicyLinks++
			issues = append(issues, ValidationIssue{
				Type:        "warning",
				Category:    "control",
				ID:          fmt.Sprintf("%d", control.ID),
				Description: fmt.Sprintf("Control '%s' has no policy links", control.Name),
			})
		}
	}

	if len(controls) > 0 {
		validation.LinkageCompleteness = float64(validation.WithPolicyLinks) / float64(len(controls)) * 100
	}

	return validation, issues, nil
}

// ValidateEvidenceTasks validates evidence task data completeness
func (v *DataValidatorImpl) ValidateEvidenceTasks(ctx context.Context) (*EvidenceValidation, []ValidationIssue, error) {
	validation := &EvidenceValidation{}
	var issues []ValidationIssue

	// Get all evidence tasks
	evidenceTasks, err := v.storage.GetAllEvidenceTasks()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get evidence tasks: %w", err)
	}

	for _, task := range evidenceTasks {
		// Check for missing description
		if task.Description == "" {
			validation.MissingDescription++
			issues = append(issues, ValidationIssue{
				Type:        "warning",
				Category:    "evidence",
				ID:          fmt.Sprintf("%d", task.ID),
				Description: fmt.Sprintf("Evidence task '%s' has no description", task.Name),
			})
		} else {
			validation.WithDescription++
		}

		// Check for guidance
		if task.Guidance == "" {
			validation.MissingGuidance++
		} else {
			validation.WithGuidance++
		}

		// Check for control links (simplified)
		validation.WithControlLinks++
	}

	if len(evidenceTasks) > 0 {
		validation.ContentCompleteness = float64(validation.WithDescription+validation.WithGuidance) / float64(len(evidenceTasks)*2) * 100
	}

	return validation, issues, nil
}

// ValidateRelationships validates relationships between entities
func (v *DataValidatorImpl) ValidateRelationships(ctx context.Context) (*RelationshipValidation, []ValidationIssue, error) {
	validation := &RelationshipValidation{}
	var issues []ValidationIssue

	// This is a simplified implementation
	// In practice, you would validate actual relationship consistency
	validation.ControlsWithPolicies = 0
	validation.EvidenceWithControls = 0
	validation.BrokenControlReferences = 0
	validation.BrokenPolicyReferences = 0

	return validation, issues, nil
}

// generateSummary generates the summary section of the validation report
func (v *DataValidatorImpl) generateSummary(report *ValidationReport) ValidationSummary {
	summary := ValidationSummary{}

	// Count issues by type
	criticalCount := 0
	warningCount := 0

	for _, issue := range report.Issues {
		switch issue.Type {
		case "critical":
			criticalCount++
		case "warning":
			warningCount++
		}
	}

	summary.CriticalIssues = criticalCount
	summary.Warnings = warningCount

	// Calculate overall score (simplified)
	totalChecks := 0
	passedChecks := 0

	// Policy checks
	totalChecks += report.Policies.WithContent + report.Policies.MissingContent
	passedChecks += report.Policies.WithContent

	// Control checks
	totalChecks += report.Controls.WithDescription + report.Controls.MissingDescription
	passedChecks += report.Controls.WithDescription

	// Evidence checks
	totalChecks += report.Evidence.WithDescription + report.Evidence.MissingDescription
	passedChecks += report.Evidence.WithDescription

	if totalChecks > 0 {
		summary.OverallScore = float64(passedChecks) / float64(totalChecks) * 100
	}

	// Determine status
	if criticalCount > 0 {
		summary.Status = "fail"
	} else if warningCount > 0 {
		summary.Status = "warning"
	} else {
		summary.Status = "pass"
	}

	return summary
}
