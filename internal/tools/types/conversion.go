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

package types

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// ConvertLegacyParamsToRequest converts legacy map[string]interface{} params to typed request
func ConvertLegacyParamsToRequest(params map[string]interface{}, req Request) error {
	// Use JSON marshaling/unmarshaling for conversion
	jsonBytes, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal params: %w", err)
	}

	if err := json.Unmarshal(jsonBytes, req); err != nil {
		return fmt.Errorf("failed to unmarshal to request type: %w", err)
	}

	return nil
}

// ConvertRequestToLegacyParams converts typed request to legacy map[string]interface{}
func ConvertRequestToLegacyParams(req Request) (map[string]interface{}, error) {
	jsonBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var params map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &params); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to legacy params: %w", err)
	}

	return params, nil
}

// ValidateAndConvertParams validates parameters and converts them to the appropriate request type
func ValidateAndConvertParams(toolName string, params map[string]interface{}) (Request, error) {
	// Create request instance for tool
	req, err := DefaultRequestMatcher.CreateRequestForTool(toolName)
	if err != nil {
		// If no typed request is registered, return a generic error
		return nil, fmt.Errorf("tool %s does not have a registered request type", toolName)
	}

	// Convert legacy params to typed request
	if err := ConvertLegacyParamsToRequest(params, req); err != nil {
		return nil, fmt.Errorf("failed to convert parameters for tool %s: %w", toolName, err)
	}

	// Validate the request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("request validation failed for tool %s: %w", toolName, err)
	}

	return req, nil
}

// ExtractRequestFromInterface attempts to extract a typed request from interface{}
func ExtractRequestFromInterface(input interface{}, targetType Request) error {
	// Handle different input types
	switch v := input.(type) {
	case map[string]interface{}:
		return ConvertLegacyParamsToRequest(v, targetType)
	case string:
		// Try to parse as JSON
		var params map[string]interface{}
		if err := json.Unmarshal([]byte(v), &params); err != nil {
			return fmt.Errorf("failed to parse JSON string: %w", err)
		}
		return ConvertLegacyParamsToRequest(params, targetType)
	default:
		// Try direct JSON marshaling/unmarshaling
		jsonBytes, err := json.Marshal(input)
		if err != nil {
			return fmt.Errorf("failed to marshal input: %w", err)
		}

		if err := json.Unmarshal(jsonBytes, targetType); err != nil {
			return fmt.Errorf("failed to unmarshal to target type: %w", err)
		}
	}

	return nil
}

// GetRequestTypeForTool returns the Go type of the request for a given tool name
func GetRequestTypeForTool(toolName string) (reflect.Type, error) {
	req, err := DefaultRequestMatcher.CreateRequestForTool(toolName)
	if err != nil {
		return nil, err
	}
	return reflect.TypeOf(req).Elem(), nil
}

// ListRegisteredRequestTypes returns a list of all registered request types
func ListRegisteredRequestTypes() []string {
	var types []string
	for toolName := range DefaultRequestMatcher.toolRequestTypes {
		types = append(types, toolName)
	}
	return types
}

// GetRequestFieldInfo returns information about the fields in a request type
func GetRequestFieldInfo(toolName string) (map[string]FieldInfo, error) {
	reqType, err := GetRequestTypeForTool(toolName)
	if err != nil {
		return nil, err
	}

	fields := make(map[string]FieldInfo)
	for i := 0; i < reqType.NumField(); i++ {
		field := reqType.Field(i)

		// Get JSON tag
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			jsonTag = strings.ToLower(field.Name)
		} else {
			// Remove options like ",omitempty"
			jsonTag = strings.Split(jsonTag, ",")[0]
		}

		// Get validation tag
		validationTag := field.Tag.Get("validate")

		fields[jsonTag] = FieldInfo{
			Name:        field.Name,
			Type:        field.Type.String(),
			JSONTag:     jsonTag,
			Validation:  validationTag,
			Required:    strings.Contains(validationTag, "required"),
			Description: getFieldDescription(field),
		}
	}

	return fields, nil
}

// FieldInfo contains metadata about a request field
type FieldInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	JSONTag     string `json:"json_tag"`
	Validation  string `json:"validation"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

// getFieldDescription extracts description from field tags or comments
func getFieldDescription(field reflect.StructField) string {
	// Look for a description tag
	if desc := field.Tag.Get("description"); desc != "" {
		return desc
	}

	// Look for a comment tag
	if comment := field.Tag.Get("comment"); comment != "" {
		return comment
	}

	// Generate a basic description from the field name
	name := field.Name
	// Convert CamelCase to space-separated words
	var result []string
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, " ")
		}
		result = append(result, string(r))
	}

	return fmt.Sprintf("The %s parameter", strings.Join(result, ""))
}
