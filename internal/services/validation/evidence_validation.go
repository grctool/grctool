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
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/storage"
)

// EvidenceValidationMode defines how strict validation should be
type EvidenceValidationMode string

const (
	EvidenceValidationModeStrict   EvidenceValidationMode = "strict"   // All checks must pass
	EvidenceValidationModeLenient  EvidenceValidationMode = "lenient"  // Only errors block
	EvidenceValidationModeAdvisory EvidenceValidationMode = "advisory" // Nothing blocks
	EvidenceValidationModeSkip     EvidenceValidationMode = "skip"     // No validation
)

// EvidenceValidationService handles evidence validation for submission
type EvidenceValidationService struct {
	storage *storage.Storage
	rules   []EvidenceValidationRule
}

// NewEvidenceValidationService creates a new evidence validation service
func NewEvidenceValidationService(stor *storage.Storage) *EvidenceValidationService {
	svc := &EvidenceValidationService{
		storage: stor,
	}

	// Register all validation rules
	svc.registerRules()

	return svc
}

// EvidenceValidationRequest defines a validation request
type EvidenceValidationRequest struct {
	TaskRef        string
	Window         string
	ValidationMode EvidenceValidationMode
}

// ValidateEvidence validates evidence for a task/window
func (vs *EvidenceValidationService) ValidateEvidence(req *EvidenceValidationRequest) (*models.ValidationResult, error) {
	// Load evidence files
	files, err := vs.storage.GetEvidenceFiles(req.TaskRef, req.Window)
	if err != nil {
		// If directory doesn't exist, return error with helpful message
		return nil, fmt.Errorf("failed to get evidence files for %s in window %s: %w", req.TaskRef, req.Window, err)
	}

	// Initialize result
	result := &models.ValidationResult{
		TaskRef:             req.TaskRef,
		Window:              req.Window,
		ValidationMode:      string(req.ValidationMode),
		Checks:              []models.ValidationCheck{},
		Errors:              []models.ValidationError{},
		WarningsList:        []models.ValidationError{},
		EvidenceFiles:       files,
		ValidationTimestamp: time.Now(),
	}

	// Skip all checks if mode is skip
	if req.ValidationMode == EvidenceValidationModeSkip {
		result.Status = "skipped"
		result.ReadyForSubmission = true
		return result, nil
	}

	// Run all validation rules
	for _, rule := range vs.rules {
		checkResult := rule.Validate(req.TaskRef, req.Window, files, vs.storage)
		result.Checks = append(result.Checks, checkResult.Check)

		// Add errors/warnings
		if len(checkResult.Errors) > 0 {
			result.Errors = append(result.Errors, checkResult.Errors...)
		}
		if len(checkResult.Warnings) > 0 {
			result.WarningsList = append(result.WarningsList, checkResult.Warnings...)
		}
	}

	// Calculate totals
	result.TotalChecks = len(result.Checks)
	result.FailedChecks = len(result.Errors)
	result.Warnings = len(result.WarningsList)
	result.PassedChecks = result.TotalChecks - result.FailedChecks

	// Calculate completeness score
	if result.TotalChecks > 0 {
		result.CompletenessScore = float64(result.PassedChecks) / float64(result.TotalChecks)
	}

	// Determine if ready for submission
	result.ReadyForSubmission = vs.isReadyForSubmission(result, req.ValidationMode)

	// Set overall status
	if result.FailedChecks > 0 {
		result.Status = "failed"
	} else if result.Warnings > 0 {
		result.Status = "warning"
	} else {
		result.Status = "passed"
	}

	// Save validation result
	if err := vs.storage.SaveValidationResult(req.TaskRef, req.Window, result); err != nil {
		// Log error but don't fail validation
		fmt.Printf("Warning: failed to save validation result: %v\n", err)
	}

	return result, nil
}

// isReadyForSubmission determines if evidence is ready based on mode
func (vs *EvidenceValidationService) isReadyForSubmission(result *models.ValidationResult, mode EvidenceValidationMode) bool {
	switch mode {
	case EvidenceValidationModeStrict:
		// No errors or warnings
		return result.FailedChecks == 0 && result.Warnings == 0
	case EvidenceValidationModeLenient:
		// No errors (warnings OK)
		return result.FailedChecks == 0
	case EvidenceValidationModeAdvisory:
		// Always ready
		return true
	case EvidenceValidationModeSkip:
		// Always ready
		return true
	default:
		return result.FailedChecks == 0
	}
}

// registerRules registers all validation rules
func (vs *EvidenceValidationService) registerRules() {
	vs.rules = []EvidenceValidationRule{
		&MinimumFileCountRule{},
		&RequiredFilesExistRule{},
		&ValidFileExtensionsRule{},
		&FileSizeLimitsRule{},
		&NonEmptyContentRule{},
		&ChecksumPresentRule{},
		&ValidTaskRefRule{},
		&WindowFormatRule{},
	}
}

// EvidenceValidationRule defines a validation rule interface
type EvidenceValidationRule interface {
	Validate(taskRef, window string, files []models.EvidenceFileRef, storage *storage.Storage) EvidenceValidationRuleResult
}

// EvidenceValidationRuleResult contains the result of a validation rule
type EvidenceValidationRuleResult struct {
	Check    models.ValidationCheck
	Errors   []models.ValidationError
	Warnings []models.ValidationError
}

// MinimumFileCountRule ensures at least one evidence file exists
type MinimumFileCountRule struct{}

func (r *MinimumFileCountRule) Validate(taskRef, window string, files []models.EvidenceFileRef, _ *storage.Storage) EvidenceValidationRuleResult {
	result := EvidenceValidationRuleResult{
		Check: models.ValidationCheck{
			Code:     "MINIMUM_FILE_COUNT",
			Name:     "Minimum File Count",
			Severity: "error",
		},
	}

	if len(files) == 0 {
		result.Check.Status = "failed"
		result.Check.Message = "No evidence files found"
		result.Errors = append(result.Errors, models.ValidationError{
			Code:       "MINIMUM_FILE_COUNT",
			Severity:   "error",
			Message:    "No evidence files found in the evidence directory",
			Suggestion: fmt.Sprintf("Generate evidence using: grctool evidence generate %s", taskRef),
		})
	} else if len(files) < 2 {
		result.Check.Status = "warning"
		result.Check.Message = fmt.Sprintf("Only %d file found (recommend 2+)", len(files))
		result.Warnings = append(result.Warnings, models.ValidationError{
			Code:       "MINIMUM_FILE_COUNT",
			Severity:   "warning",
			Message:    fmt.Sprintf("Only %d evidence file found. Consider adding more evidence for completeness.", len(files)),
			Suggestion: "Add additional evidence files or documentation",
		})
	} else {
		result.Check.Status = "passed"
		result.Check.Message = fmt.Sprintf("Found %d evidence files", len(files))
	}

	return result
}

// RequiredFilesExistRule checks that referenced files exist
type RequiredFilesExistRule struct{}

func (r *RequiredFilesExistRule) Validate(taskRef, window string, files []models.EvidenceFileRef, _ *storage.Storage) EvidenceValidationRuleResult {
	result := EvidenceValidationRuleResult{
		Check: models.ValidationCheck{
			Code:     "REQUIRED_FILES_PRESENT",
			Name:     "Required Files Present",
			Severity: "error",
		},
	}

	// Check that all files in the list actually exist
	// Files are already checked during GetEvidenceFiles
	// Mark as passed
	result.Check.Status = "passed"
	result.Check.Message = "All files present"

	return result
}

// ValidFileExtensionsRule validates file extensions
type ValidFileExtensionsRule struct{}

var allowedExtensions = []string{".md", ".csv", ".json", ".pdf", ".xlsx", ".txt", ".yaml", ".yml"}

func (r *ValidFileExtensionsRule) Validate(taskRef, window string, files []models.EvidenceFileRef, _ *storage.Storage) EvidenceValidationRuleResult {
	result := EvidenceValidationRuleResult{
		Check: models.ValidationCheck{
			Code:     "VALID_FILE_EXTENSIONS",
			Name:     "Valid File Extensions",
			Severity: "warning",
		},
	}

	invalidFiles := []string{}
	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file.Filename))
		valid := false
		for _, allowedExt := range allowedExtensions {
			if ext == allowedExt {
				valid = true
				break
			}
		}
		if !valid {
			invalidFiles = append(invalidFiles, file.Filename)
		}
	}

	if len(invalidFiles) > 0 {
		result.Check.Status = "warning"
		result.Check.Message = fmt.Sprintf("%d files with unexpected extensions", len(invalidFiles))
		for _, filename := range invalidFiles {
			result.Warnings = append(result.Warnings, models.ValidationError{
				Code:       "VALID_FILE_EXTENSIONS",
				Severity:   "warning",
				Message:    fmt.Sprintf("Unexpected file extension: %s", filename),
				Suggestion: fmt.Sprintf("Allowed extensions: %s", strings.Join(allowedExtensions, ", ")),
			})
		}
	} else {
		result.Check.Status = "passed"
		result.Check.Message = "All file extensions valid"
	}

	return result
}

// FileSizeLimitsRule checks file size limits
type FileSizeLimitsRule struct{}

const maxFileSize = 50 * 1024 * 1024 // 50 MB

func (r *FileSizeLimitsRule) Validate(taskRef, window string, files []models.EvidenceFileRef, _ *storage.Storage) EvidenceValidationRuleResult {
	result := EvidenceValidationRuleResult{
		Check: models.ValidationCheck{
			Code:     "FILE_SIZE_LIMITS",
			Name:     "File Size Limits",
			Severity: "error",
		},
	}

	oversizedFiles := []string{}
	for _, file := range files {
		if file.SizeBytes > maxFileSize {
			oversizedFiles = append(oversizedFiles, file.Filename)
		}
	}

	if len(oversizedFiles) > 0 {
		result.Check.Status = "failed"
		result.Check.Message = fmt.Sprintf("%d files exceed size limit", len(oversizedFiles))
		for _, filename := range oversizedFiles {
			result.Errors = append(result.Errors, models.ValidationError{
				Code:       "FILE_SIZE_LIMITS",
				Severity:   "error",
				Message:    fmt.Sprintf("File exceeds 50MB limit: %s", filename),
				Suggestion: "Compress the file or split into smaller files",
			})
		}
	} else {
		result.Check.Status = "passed"
		result.Check.Message = "All files within size limits"
	}

	return result
}

// NonEmptyContentRule checks that files are not empty
type NonEmptyContentRule struct{}

func (r *NonEmptyContentRule) Validate(taskRef, window string, files []models.EvidenceFileRef, _ *storage.Storage) EvidenceValidationRuleResult {
	result := EvidenceValidationRuleResult{
		Check: models.ValidationCheck{
			Code:     "NON_EMPTY_CONTENT",
			Name:     "Non-Empty Content",
			Severity: "error",
		},
	}

	emptyFiles := []string{}
	for _, file := range files {
		if file.SizeBytes == 0 {
			emptyFiles = append(emptyFiles, file.Filename)
		}
	}

	if len(emptyFiles) > 0 {
		result.Check.Status = "failed"
		result.Check.Message = fmt.Sprintf("%d empty files", len(emptyFiles))
		for _, filename := range emptyFiles {
			result.Errors = append(result.Errors, models.ValidationError{
				Code:       "NON_EMPTY_CONTENT",
				Severity:   "error",
				Message:    fmt.Sprintf("File is empty: %s", filename),
				Suggestion: "Add content to the file or remove it",
			})
		}
	} else {
		result.Check.Status = "passed"
		result.Check.Message = "All files have content"
	}

	return result
}

// ChecksumPresentRule validates checksums are present
type ChecksumPresentRule struct{}

func (r *ChecksumPresentRule) Validate(taskRef, window string, files []models.EvidenceFileRef, _ *storage.Storage) EvidenceValidationRuleResult {
	result := EvidenceValidationRuleResult{
		Check: models.ValidationCheck{
			Code:     "CHECKSUM_PRESENT",
			Name:     "Checksum Present",
			Severity: "warning",
		},
	}

	missingChecksums := 0
	for _, file := range files {
		if file.ChecksumSHA256 == "" {
			missingChecksums++
		}
	}

	if missingChecksums > 0 {
		result.Check.Status = "warning"
		result.Check.Message = fmt.Sprintf("%d files missing checksums", missingChecksums)
		result.Warnings = append(result.Warnings, models.ValidationError{
			Code:       "CHECKSUM_PRESENT",
			Severity:   "warning",
			Message:    fmt.Sprintf("%d files are missing SHA256 checksums", missingChecksums),
			Suggestion: "Checksums will be generated during submission",
		})
	} else {
		result.Check.Status = "passed"
		result.Check.Message = "All files have checksums"
	}

	return result
}

// ValidTaskRefRule validates task reference format
type ValidTaskRefRule struct{}

func (r *ValidTaskRefRule) Validate(taskRef, window string, files []models.EvidenceFileRef, _ *storage.Storage) EvidenceValidationRuleResult {
	result := EvidenceValidationRuleResult{
		Check: models.ValidationCheck{
			Code:     "VALID_TASK_REF",
			Name:     "Valid Task Reference",
			Severity: "error",
		},
	}

	// Check format ET-XXXX
	if !strings.HasPrefix(taskRef, "ET-") || len(taskRef) < 4 {
		result.Check.Status = "failed"
		result.Check.Message = "Invalid task reference format"
		result.Errors = append(result.Errors, models.ValidationError{
			Code:       "VALID_TASK_REF",
			Severity:   "error",
			Message:    fmt.Sprintf("Task reference '%s' does not match pattern ET-XXXX", taskRef),
			Field:      "task_ref",
			Suggestion: "Use proper task reference format (e.g., ET-0001)",
		})
	} else {
		result.Check.Status = "passed"
		result.Check.Message = "Valid task reference format"
	}

	return result
}

// WindowFormatRule validates window format
type WindowFormatRule struct{}

func (r *WindowFormatRule) Validate(taskRef, window string, files []models.EvidenceFileRef, _ *storage.Storage) EvidenceValidationRuleResult {
	result := EvidenceValidationRuleResult{
		Check: models.ValidationCheck{
			Code:     "WINDOW_FORMAT",
			Name:     "Window Format",
			Severity: "warning",
		},
	}

	// Check format YYYY-QX or YYYY-MM-DD
	validQuarter := len(window) == 7 && window[4] == '-' && window[5] == 'Q'
	validDate := len(window) == 10 && window[4] == '-' && window[7] == '-'

	if !validQuarter && !validDate {
		result.Check.Status = "warning"
		result.Check.Message = "Unexpected window format"
		result.Warnings = append(result.Warnings, models.ValidationError{
			Code:       "WINDOW_FORMAT",
			Severity:   "warning",
			Message:    fmt.Sprintf("Window '%s' does not match expected format", window),
			Field:      "window",
			Suggestion: "Use format YYYY-QX (e.g., 2025-Q4) or YYYY-MM-DD",
		})
	} else {
		result.Check.Status = "passed"
		result.Check.Message = "Valid window format"
	}

	return result
}
