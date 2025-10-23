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

package interpolation

import (
	"fmt"
	"regexp"
	"strings"
)

// Interpolator defines the interface for variable interpolation
type Interpolator interface {
	// Interpolate replaces variables in the input text with configured values
	Interpolate(text string) (string, error)

	// InterpolateWithContext replaces variables and provides context about substitutions made
	InterpolateWithContext(text string) (result string, substitutions map[string]string, err error)
}

// StandardInterpolator implements the Interpolator interface
type StandardInterpolator struct {
	variables         map[string]string
	enabled           bool
	onMissingVariable MissingVariableAction
}

// MissingVariableAction defines how to handle missing variables
type MissingVariableAction int

const (
	// MissingVariableIgnore leaves missing variables unchanged
	MissingVariableIgnore MissingVariableAction = iota
	// MissingVariableWarn logs a warning but continues processing
	MissingVariableWarn
	// MissingVariableError returns an error when variables are missing
	MissingVariableError
)

// InterpolatorConfig holds configuration for the interpolator
type InterpolatorConfig struct {
	Variables         map[string]string
	Enabled           bool
	OnMissingVariable MissingVariableAction
}

// NewStandardInterpolator creates a new StandardInterpolator with the given configuration
func NewStandardInterpolator(config InterpolatorConfig) *StandardInterpolator {
	// Create a copy of the variables map to avoid external modifications
	variables := make(map[string]string)
	for k, v := range config.Variables {
		variables[k] = v
	}

	return &StandardInterpolator{
		variables:         variables,
		enabled:           config.Enabled,
		onMissingVariable: config.OnMissingVariable,
	}
}

// Interpolate replaces variables in the input text with configured values
func (si *StandardInterpolator) Interpolate(text string) (string, error) {
	result, _, err := si.InterpolateWithContext(text)
	return result, err
}

// InterpolateWithContext replaces variables and provides context about substitutions made
func (si *StandardInterpolator) InterpolateWithContext(text string) (string, map[string]string, error) {
	if !si.enabled {
		return text, make(map[string]string), nil
	}

	// Handle template variables {{variable.name}}
	result, substitutions, err := si.interpolateTemplateVariables(text)
	if err != nil {
		return text, nil, err
	}

	// Handle bracket variables [Variable Name]
	result, bracketSubstitutions, err := si.interpolateBracketVariables(result)
	if err != nil {
		return text, nil, err
	}

	// Merge substitutions
	for k, v := range bracketSubstitutions {
		substitutions[k] = v
	}

	return result, substitutions, nil
}

// interpolateTemplateVariables handles {{variable.name}} format
func (si *StandardInterpolator) interpolateTemplateVariables(text string) (string, map[string]string, error) {
	// Regex to match {{variable.name}} patterns
	templateRegex := regexp.MustCompile(`\{\{([^}]+)\}\}`)
	substitutions := make(map[string]string)

	result := templateRegex.ReplaceAllStringFunc(text, func(match string) string {
		// Extract the variable name (remove {{ and }})
		varName := strings.TrimSpace(match[2 : len(match)-2])

		// Look up the variable value
		if value, exists := si.variables[varName]; exists {
			substitutions[match] = value
			return value
		}

		// Handle missing variable based on configuration
		switch si.onMissingVariable {
		case MissingVariableIgnore:
			return match // Leave unchanged
		case MissingVariableWarn:
			// In a real implementation, this would log a warning
			// For now, we'll just leave it unchanged
			return match
		case MissingVariableError:
			// We can't return an error from this func, so we'll handle it outside
			substitutions["__ERROR__"] = fmt.Sprintf("missing variable: %s", varName)
			return match
		default:
			return match
		}
	})

	// Check if there was an error during substitution
	if errorMsg, hasError := substitutions["__ERROR__"]; hasError {
		delete(substitutions, "__ERROR__")
		return text, nil, fmt.Errorf("%s", errorMsg)
	}

	return result, substitutions, nil
}

// interpolateBracketVariables handles [Variable Name] format
func (si *StandardInterpolator) interpolateBracketVariables(text string) (string, map[string]string, error) {
	// Regex to match [Variable Name] patterns
	bracketRegex := regexp.MustCompile(`\[([^\]]+)\]`)
	substitutions := make(map[string]string)

	result := bracketRegex.ReplaceAllStringFunc(text, func(match string) string {
		// Extract the variable name (remove [ and ])
		varName := strings.TrimSpace(match[1 : len(match)-1])

		// Look up the variable value
		if value, exists := si.variables[varName]; exists {
			substitutions[match] = value
			return value
		}

		// Handle missing variable based on configuration
		switch si.onMissingVariable {
		case MissingVariableIgnore:
			return match // Leave unchanged
		case MissingVariableWarn:
			// In a real implementation, this would log a warning
			// For now, we'll just leave it unchanged
			return match
		case MissingVariableError:
			// We can't return an error from this func, so we'll handle it outside
			substitutions["__ERROR__"] = fmt.Sprintf("missing variable: %s", varName)
			return match
		default:
			return match
		}
	})

	// Check if there was an error during substitution
	if errorMsg, hasError := substitutions["__ERROR__"]; hasError {
		delete(substitutions, "__ERROR__")
		return text, nil, fmt.Errorf("%s", errorMsg)
	}

	return result, substitutions, nil
}

// findVariableValue looks up a variable value with exact matching
func (si *StandardInterpolator) findVariableValue(varName string) (string, bool) {
	value, exists := si.variables[varName]
	return value, exists
}

// GetVariables returns a copy of the configured variables
func (si *StandardInterpolator) GetVariables() map[string]string {
	variables := make(map[string]string)
	for k, v := range si.variables {
		variables[k] = v
	}
	return variables
}

// SetVariable adds or updates a variable
func (si *StandardInterpolator) SetVariable(key, value string) {
	si.variables[key] = value
}

// RemoveVariable removes a variable
func (si *StandardInterpolator) RemoveVariable(key string) {
	delete(si.variables, key)
}
