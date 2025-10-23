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
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/grctool/grctool/internal/registry"
)

// ValidationError represents a validation error with detailed context
type ValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

// Error implements the error interface
func (v ValidationError) Error() string {
	return fmt.Sprintf("validation error on field '%s': %s", v.Field, v.Message)
}

// ValidationResult contains the results of input validation
type ValidationResult struct {
	Valid      bool              `json:"valid"`
	Errors     []ValidationError `json:"errors,omitempty"`
	Warnings   []ValidationError `json:"warnings,omitempty"`
	Normalized map[string]string `json:"normalized,omitempty"`
}

// Validator provides input validation and path safety checks
type Validator struct {
	dataDir  string
	registry *registry.EvidenceTaskRegistry
}

// NewValidator creates a new validator with the specified data directory
func NewValidator(dataDir string) *Validator {
	// Initialize evidence task registry
	taskRegistry := registry.NewEvidenceTaskRegistry(dataDir)
	// Load registry - ignore errors for now to maintain backward compatibility
	_ = taskRegistry.LoadRegistry()

	return &Validator{
		dataDir:  dataDir,
		registry: taskRegistry,
	}
}

// ValidateTaskReference validates and normalizes task references
func (v *Validator) ValidateTaskReference(taskRef string) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:      true,
		Errors:     []ValidationError{},
		Warnings:   []ValidationError{},
		Normalized: make(map[string]string),
	}

	if taskRef == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "task_ref",
			Value:   taskRef,
			Rule:    "required",
			Message: "task reference is required",
		})
		return result, nil
	}

	// Normalize task reference
	normalized, warning, err := v.normalizeTaskReference(taskRef)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "task_ref",
			Value:   taskRef,
			Rule:    "format",
			Message: err.Error(),
		})
		return result, nil
	}

	result.Normalized["task_ref"] = normalized

	if warning != "" {
		result.Warnings = append(result.Warnings, ValidationError{
			Field:   "task_ref",
			Value:   taskRef,
			Rule:    "format",
			Message: warning,
		})
	}

	return result, nil
}

// normalizeTaskReference converts various task reference formats to numeric ID
func (v *Validator) normalizeTaskReference(taskRef string) (string, string, error) {
	trimmed := strings.TrimSpace(taskRef)

	// If it's already a numeric ID, return as-is
	if _, err := strconv.Atoi(trimmed); err == nil {
		return trimmed, "", nil
	}

	// Try registry lookup first for ET references
	if v.registry != nil {
		// Handle all ET reference formats and normalize them for registry lookup
		normalizedRef := v.normalizeETReference(trimmed)
		if normalizedRef != "" {
			if taskID, found := v.registry.GetTaskID(normalizedRef); found {
				return strconv.Itoa(taskID), fmt.Sprintf("resolved %s to task ID %d from registry", trimmed, taskID), nil
			}
		}
	}

	// Fallback to calculation for backward compatibility
	return v.calculateTaskID(trimmed)
}

// normalizeETReference normalizes ET reference formats for registry lookup
func (v *Validator) normalizeETReference(taskRef string) string {
	upper := strings.ToUpper(taskRef)

	// Handle ET-XXX format (with dash) -> ETXXX
	if matched, _ := regexp.MatchString(`^ET-\d+$`, upper); matched {
		return strings.Replace(upper, "-", "", 1)
	}

	// Handle ET XXX format (with space) -> ETXXX
	if matched, _ := regexp.MatchString(`^ET\s+\d+$`, upper); matched {
		parts := strings.Fields(upper)
		if len(parts) == 2 {
			return "ET" + parts[1]
		}
	}

	// Handle ETXXX format (already normalized)
	if matched, _ := regexp.MatchString(`^ET\d+$`, upper); matched {
		return upper
	}

	return ""
}

// NormalizeReferenceID normalizes reference IDs to standardized format
// Evidence: ET-0001, ET-0104 (4-digit zero-padded)
// Policy: POL-0001, POL-0002 (4-digit zero-padded)
// Control: AC-01, CC-01_1, SO-19 (2-digit zero-padded, underscore for subsections)
func (v *Validator) NormalizeReferenceID(refID string, docType string) (string, error) {
	trimmed := strings.TrimSpace(refID)

	switch docType {
	case "evidence":
		return v.normalizeEvidenceReference(trimmed)
	case "policy":
		return v.normalizePolicyReference(trimmed)
	case "control":
		return v.normalizeControlReference(trimmed)
	default:
		return "", fmt.Errorf("unsupported document type: %s", docType)
	}
}

// normalizeEvidenceReference normalizes evidence task references to ET-#### format
func (v *Validator) normalizeEvidenceReference(refID string) (string, error) {
	upper := strings.ToUpper(refID)

	// Handle ET-XXX format (with dash)
	if matched, _ := regexp.MatchString(`^ET-\d+$`, upper); matched {
		numStr := strings.TrimPrefix(upper, "ET-")
		num, err := strconv.Atoi(numStr)
		if err != nil {
			return "", fmt.Errorf("invalid numeric part: %s", numStr)
		}
		return fmt.Sprintf("ET-%04d", num), nil
	}

	// Handle ET XXX format (with space)
	if matched, _ := regexp.MatchString(`^ET\s+\d+$`, upper); matched {
		parts := strings.Fields(upper)
		if len(parts) == 2 {
			num, err := strconv.Atoi(parts[1])
			if err != nil {
				return "", fmt.Errorf("invalid numeric part: %s", parts[1])
			}
			return fmt.Sprintf("ET-%04d", num), nil
		}
	}

	// Handle ETXXX format (no separator)
	if matched, _ := regexp.MatchString(`^ET\d+$`, upper); matched {
		numStr := strings.TrimPrefix(upper, "ET")
		num, err := strconv.Atoi(numStr)
		if err != nil {
			return "", fmt.Errorf("invalid numeric part: %s", numStr)
		}
		return fmt.Sprintf("ET-%04d", num), nil
	}

	// Handle plain numeric (assume it's an evidence task ID)
	if num, err := strconv.Atoi(upper); err == nil && num > 0 {
		return fmt.Sprintf("ET-%04d", num), nil
	}

	return "", fmt.Errorf("invalid evidence reference format: %s", refID)
}

// normalizePolicyReference normalizes policy references to POL-#### format
func (v *Validator) normalizePolicyReference(refID string) (string, error) {
	upper := strings.ToUpper(refID)

	// Handle POL-XXX format (with dash)
	if matched, _ := regexp.MatchString(`^POL-\d+$`, upper); matched {
		numStr := strings.TrimPrefix(upper, "POL-")
		num, err := strconv.Atoi(numStr)
		if err != nil {
			return "", fmt.Errorf("invalid numeric part: %s", numStr)
		}
		return fmt.Sprintf("POL-%04d", num), nil
	}

	// Handle POL XXX format (with space)
	if matched, _ := regexp.MatchString(`^POL\s+\d+$`, upper); matched {
		parts := strings.Fields(upper)
		if len(parts) == 2 {
			num, err := strconv.Atoi(parts[1])
			if err != nil {
				return "", fmt.Errorf("invalid numeric part: %s", parts[1])
			}
			return fmt.Sprintf("POL-%04d", num), nil
		}
	}

	// Handle POLXXX format (no separator)
	if matched, _ := regexp.MatchString(`^POL\d+$`, upper); matched {
		numStr := strings.TrimPrefix(upper, "POL")
		num, err := strconv.Atoi(numStr)
		if err != nil {
			return "", fmt.Errorf("invalid numeric part: %s", numStr)
		}
		return fmt.Sprintf("POL-%04d", num), nil
	}

	// Handle plain numeric (assume it's a policy ID)
	if num, err := strconv.Atoi(upper); err == nil && num > 0 {
		return fmt.Sprintf("POL-%04d", num), nil
	}

	return "", fmt.Errorf("invalid policy reference format: %s", refID)
}

// normalizeControlReference normalizes control references to <PREFIX>-##[_#] format
func (v *Validator) normalizeControlReference(refID string) (string, error) {
	upper := strings.ToUpper(refID)

	// Define control prefixes
	controlPrefixes := []string{"AA", "AC", "AT", "CC", "CM", "CR", "DP", "DS", "HR", "IM", "OM", "RM", "SO", "SS", "VM", "WS"}

	for _, prefix := range controlPrefixes {
		// Handle PREFIX-XX.X format (with dash and dot)
		pattern := fmt.Sprintf(`^%s-\d+\.\d+$`, prefix)
		if matched, _ := regexp.MatchString(pattern, upper); matched {
			parts := strings.Split(strings.TrimPrefix(upper, prefix+"-"), ".")
			if len(parts) == 2 {
				main, err := strconv.Atoi(parts[0])
				if err != nil {
					return "", fmt.Errorf("invalid main number: %s", parts[0])
				}
				sub, err := strconv.Atoi(parts[1])
				if err != nil {
					return "", fmt.Errorf("invalid sub number: %s", parts[1])
				}
				return fmt.Sprintf("%s-%02d_%d", prefix, main, sub), nil
			}
		}

		// Handle PREFIX-XX format (with dash, no subsection)
		pattern = fmt.Sprintf(`^%s-\d+$`, prefix)
		if matched, _ := regexp.MatchString(pattern, upper); matched {
			numStr := strings.TrimPrefix(upper, prefix+"-")
			num, err := strconv.Atoi(numStr)
			if err != nil {
				return "", fmt.Errorf("invalid numeric part: %s", numStr)
			}
			return fmt.Sprintf("%s-%02d", prefix, num), nil
		}

		// Handle PREFIXXX.X format (no dash, with dot)
		pattern = fmt.Sprintf(`^%s\d+\.\d+$`, prefix)
		if matched, _ := regexp.MatchString(pattern, upper); matched {
			parts := strings.Split(strings.TrimPrefix(upper, prefix), ".")
			if len(parts) == 2 {
				main, err := strconv.Atoi(parts[0])
				if err != nil {
					return "", fmt.Errorf("invalid main number: %s", parts[0])
				}
				sub, err := strconv.Atoi(parts[1])
				if err != nil {
					return "", fmt.Errorf("invalid sub number: %s", parts[1])
				}
				return fmt.Sprintf("%s-%02d_%d", prefix, main, sub), nil
			}
		}

		// Handle PREFIXXX format (no dash, no subsection)
		pattern = fmt.Sprintf(`^%s\d+$`, prefix)
		if matched, _ := regexp.MatchString(pattern, upper); matched {
			numStr := strings.TrimPrefix(upper, prefix)
			num, err := strconv.Atoi(numStr)
			if err != nil {
				return "", fmt.Errorf("invalid numeric part: %s", numStr)
			}
			return fmt.Sprintf("%s-%02d", prefix, num), nil
		}

		// Handle PREFIX XX.X format (with space and dot)
		pattern = fmt.Sprintf(`^%s\s+\d+\.\d+$`, prefix)
		if matched, _ := regexp.MatchString(pattern, upper); matched {
			remaining := strings.TrimPrefix(upper, prefix)
			remaining = strings.TrimSpace(remaining)
			parts := strings.Split(remaining, ".")
			if len(parts) == 2 {
				main, err := strconv.Atoi(parts[0])
				if err != nil {
					return "", fmt.Errorf("invalid main number: %s", parts[0])
				}
				sub, err := strconv.Atoi(parts[1])
				if err != nil {
					return "", fmt.Errorf("invalid sub number: %s", parts[1])
				}
				return fmt.Sprintf("%s-%02d_%d", prefix, main, sub), nil
			}
		}

		// Handle PREFIX XX format (with space, no subsection)
		pattern = fmt.Sprintf(`^%s\s+\d+$`, prefix)
		if matched, _ := regexp.MatchString(pattern, upper); matched {
			remaining := strings.TrimPrefix(upper, prefix)
			remaining = strings.TrimSpace(remaining)
			num, err := strconv.Atoi(remaining)
			if err != nil {
				return "", fmt.Errorf("invalid numeric part: %s", remaining)
			}
			return fmt.Sprintf("%s-%02d", prefix, num), nil
		}
	}

	// Handle plain numeric (assume it's a control ID)
	if num, err := strconv.Atoi(upper); err == nil && num > 0 {
		// Without more context, we can't determine the prefix, so return error
		return "", fmt.Errorf("control reference requires prefix (e.g., AC-1, CC-1.1): %s", refID)
	}

	return "", fmt.Errorf("invalid control reference format: %s", refID)
}

// calculateTaskID provides fallback calculation method
func (v *Validator) calculateTaskID(taskRef string) (string, string, error) {
	trimmed := strings.TrimSpace(taskRef)

	// Handle ET-XXX format (with dash)
	if matched, _ := regexp.MatchString(`^ET-\d+$`, strings.ToUpper(trimmed)); matched {
		etNum := strings.TrimPrefix(strings.ToUpper(trimmed), "ET-")
		etNumInt, _ := strconv.Atoi(etNum)

		// Convert ET-1 to 327992, ET-2 to 327993, etc.
		taskID := 327991 + etNumInt
		return strconv.Itoa(taskID), fmt.Sprintf("calculated ET-%s to task ID %d (fallback)", etNum, taskID), nil
	}

	// Handle ET XXX format (with space)
	if matched, _ := regexp.MatchString(`^ET\s+\d+$`, strings.ToUpper(trimmed)); matched {
		parts := strings.Fields(strings.ToUpper(trimmed))
		if len(parts) == 2 {
			etNum := parts[1]
			etNumInt, _ := strconv.Atoi(etNum)

			// Convert ET 1 to 327992, ET 2 to 327993, etc.
			taskID := 327991 + etNumInt
			return strconv.Itoa(taskID), fmt.Sprintf("calculated ET %s to task ID %d (fallback)", etNum, taskID), nil
		}
	}

	// Handle ETXXX format (no separator)
	if matched, _ := regexp.MatchString(`^ET\d+$`, strings.ToUpper(trimmed)); matched {
		etNum := strings.TrimPrefix(strings.ToUpper(trimmed), "ET")
		etNumInt, _ := strconv.Atoi(etNum)

		// Convert ET1 to 327992, ET2 to 327993, etc.
		taskID := 327991 + etNumInt
		return strconv.Itoa(taskID), fmt.Sprintf("calculated ET%s to task ID %d (fallback)", etNum, taskID), nil
	}

	// Handle direct numeric task IDs in various ranges
	if matched, _ := regexp.MatchString(`^(32\d{4}|65\d{4}|82\d{4}|86\d{4})$`, trimmed); matched {
		return trimmed, "", nil
	}

	return "", "", fmt.Errorf("invalid task reference format: '%s'. Expected formats: ET-101, ET 101, ET101, 328001, or plain numeric ID", taskRef)
}

// ValidatePathSafety ensures paths are safe and within allowed boundaries
func (v *Validator) ValidatePathSafety(path string) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:      true,
		Errors:     []ValidationError{},
		Warnings:   []ValidationError{},
		Normalized: make(map[string]string),
	}

	if path == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "path",
			Value:   path,
			Rule:    "required",
			Message: "path is required",
		})
		return result, nil
	}

	// Clean and normalize the path
	cleanPath := filepath.Clean(path)
	result.Normalized["path"] = cleanPath

	// Check for path traversal attempts
	if strings.Contains(cleanPath, "..") {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "path",
			Value:   path,
			Rule:    "no_traversal",
			Message: "path traversal not allowed (contains '..')",
		})
		return result, nil
	}

	// Check for absolute paths when relative expected
	if filepath.IsAbs(cleanPath) {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "path",
			Value:   path,
			Rule:    "relative_only",
			Message: "absolute paths not allowed, use relative paths under data directory",
		})
		return result, nil
	}

	// Ensure path would be under data directory
	if v.dataDir != "" {
		fullPath := filepath.Join(v.dataDir, cleanPath)
		absDataDir, err := filepath.Abs(v.dataDir)
		if err != nil {
			return result, fmt.Errorf("failed to resolve data directory: %w", err)
		}

		absFullPath, err := filepath.Abs(fullPath)
		if err != nil {
			return result, fmt.Errorf("failed to resolve full path: %w", err)
		}

		// Check if the resolved path is under data directory
		if !strings.HasPrefix(absFullPath, absDataDir+string(filepath.Separator)) &&
			absFullPath != absDataDir {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   "path",
				Value:   path,
				Rule:    "under_data_dir",
				Message: fmt.Sprintf("path must be under data directory: %s", absDataDir),
			})
			return result, nil
		}
	}

	// Check for potentially dangerous paths
	dangerousPaths := []string{
		"/etc", "/proc", "/sys", "/dev",
		"etc/", "proc/", "sys/", "dev/",
		".ssh", ".git", ".env",
	}

	lowerPath := strings.ToLower(cleanPath)
	for _, dangerous := range dangerousPaths {
		if strings.Contains(lowerPath, dangerous) {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   "path",
				Value:   path,
				Rule:    "no_dangerous_paths",
				Message: fmt.Sprintf("path contains potentially dangerous component: %s", dangerous),
			})
			return result, nil
		}
	}

	return result, nil
}

// ValidateParameters validates a map of parameters
func (v *Validator) ValidateParameters(params map[string]interface{}, rules map[string]ValidationRule) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:      true,
		Errors:     []ValidationError{},
		Warnings:   []ValidationError{},
		Normalized: make(map[string]string),
	}

	// Check required parameters
	for field, rule := range rules {
		value, exists := params[field]

		if rule.Required && !exists {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   field,
				Value:   "",
				Rule:    "required",
				Message: fmt.Sprintf("required parameter '%s' is missing", field),
			})
			continue
		}

		if !exists {
			continue // Skip validation for optional missing parameters
		}

		// Validate parameter value
		if err := v.validateParameterValue(field, value, rule, result); err != nil {
			return result, err
		}
	}

	return result, nil
}

// ValidationRule defines validation rules for parameters
type ValidationRule struct {
	Required      bool     `json:"required"`
	Type          string   `json:"type"` // "string", "int", "bool", "path"
	MinLength     int      `json:"min_length,omitempty"`
	MaxLength     int      `json:"max_length,omitempty"`
	Pattern       string   `json:"pattern,omitempty"`
	AllowedValues []string `json:"allowed_values,omitempty"`
	PathSafety    bool     `json:"path_safety,omitempty"`
}

// validateParameterValue validates a single parameter value
func (v *Validator) validateParameterValue(field string, value interface{}, rule ValidationRule, result *ValidationResult) error {
	valueStr := fmt.Sprintf("%v", value)

	// Type validation
	switch rule.Type {
	case "string":
		if _, ok := value.(string); !ok {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   field,
				Value:   valueStr,
				Rule:    "type",
				Message: "must be a string",
			})
		}
	case "int":
		if _, err := strconv.Atoi(valueStr); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   field,
				Value:   valueStr,
				Rule:    "type",
				Message: "must be an integer",
			})
		}
	case "bool":
		if _, err := strconv.ParseBool(valueStr); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   field,
				Value:   valueStr,
				Rule:    "type",
				Message: "must be a boolean",
			})
		}
	case "path":
		if rule.PathSafety {
			pathResult, err := v.ValidatePathSafety(valueStr)
			if err != nil {
				return err
			}
			if !pathResult.Valid {
				result.Valid = false
				result.Errors = append(result.Errors, pathResult.Errors...)
			}
			if pathResult.Normalized["path"] != "" {
				result.Normalized[field] = pathResult.Normalized["path"]
			}
		}
	}

	// Length validation for strings
	if rule.MinLength > 0 && len(valueStr) < rule.MinLength {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   field,
			Value:   valueStr,
			Rule:    "min_length",
			Message: fmt.Sprintf("must be at least %d characters long", rule.MinLength),
		})
	}

	if rule.MaxLength > 0 && len(valueStr) > rule.MaxLength {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   field,
			Value:   valueStr,
			Rule:    "max_length",
			Message: fmt.Sprintf("must be no more than %d characters long", rule.MaxLength),
		})
	}

	// Pattern validation
	if rule.Pattern != "" {
		matched, err := regexp.MatchString(rule.Pattern, valueStr)
		if err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
		if !matched {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   field,
				Value:   valueStr,
				Rule:    "pattern",
				Message: fmt.Sprintf("does not match required pattern: %s", rule.Pattern),
			})
		}
	}

	// Allowed values validation
	if len(rule.AllowedValues) > 0 {
		found := false
		for _, allowed := range rule.AllowedValues {
			if valueStr == allowed {
				found = true
				break
			}
		}
		if !found {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   field,
				Value:   valueStr,
				Rule:    "allowed_values",
				Message: fmt.Sprintf("must be one of: %s", strings.Join(rule.AllowedValues, ", ")),
			})
		}
	}

	return nil
}
