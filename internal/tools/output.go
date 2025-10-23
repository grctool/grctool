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
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ToolOutput represents the standardized JSON envelope for all tool outputs
type ToolOutput struct {
	OK    bool        `json:"ok"`
	Data  interface{} `json:"data,omitempty"`
	Error *ToolError  `json:"error,omitempty"`
	Meta  ToolMeta    `json:"meta"`
}

// ToolError provides detailed error information with correlation tracking
type ToolError struct {
	Code          string                 `json:"code"`
	Message       string                 `json:"message"`
	Details       map[string]interface{} `json:"details,omitempty"`
	CorrelationID string                 `json:"correlation_id"`
	Timestamp     time.Time              `json:"timestamp"`
}

// ToolMeta contains metadata about the operation
type ToolMeta struct {
	CorrelationID string      `json:"correlation_id"`
	Timestamp     time.Time   `json:"timestamp"`
	DurationMS    int64       `json:"duration_ms"`
	TaskRef       string      `json:"task_ref,omitempty"`
	Tool          string      `json:"tool,omitempty"`
	Version       string      `json:"version,omitempty"`
	AuthStatus    *AuthStatus `json:"auth_status,omitempty"`
	DataSource    string      `json:"data_source,omitempty"`
}

// AuthStatus contains authentication information for tools that require it
type AuthStatus struct {
	Authenticated bool       `json:"authenticated"`
	Provider      string     `json:"provider"`
	CacheUsed     bool       `json:"cache_used"`
	TokenPresent  bool       `json:"token_present,omitempty"`
	LastValidated *time.Time `json:"last_validated,omitempty"`
	Error         string     `json:"error,omitempty"`
}

// Common error codes
const (
	ErrorCodeValidation   = "VALIDATION_ERROR"
	ErrorCodeExecution    = "EXECUTION_ERROR"
	ErrorCodePathSafety   = "PATH_SAFETY_ERROR"
	ErrorCodeTaskNotFound = "TASK_NOT_FOUND"
	ErrorCodeUnauthorized = "UNAUTHORIZED"
	ErrorCodeInternal     = "INTERNAL_ERROR"
	ErrorCodeConfigError  = "CONFIG_ERROR"
	ErrorCodeServiceError = "SERVICE_ERROR"
	ErrorCodeNetworkError = "NETWORK_ERROR"
)

// NewSuccessOutput creates a successful tool output
func NewSuccessOutput(data interface{}, meta ToolMeta) *ToolOutput {
	return &ToolOutput{
		OK:   true,
		Data: data,
		Meta: meta,
	}
}

// NewErrorOutput creates an error tool output
func NewErrorOutput(err *ToolError, meta ToolMeta) *ToolOutput {
	return &ToolOutput{
		OK:    false,
		Error: err,
		Meta:  meta,
	}
}

// NewToolError creates a new tool error with correlation tracking
func NewToolError(code, message, correlationID string, details map[string]interface{}) *ToolError {
	return &ToolError{
		Code:          code,
		Message:       message,
		Details:       details,
		CorrelationID: correlationID,
		Timestamp:     time.Now(),
	}
}

// NewToolMeta creates metadata for an operation
func NewToolMeta(correlationID, taskRef, tool string, duration time.Duration) ToolMeta {
	return ToolMeta{
		CorrelationID: correlationID,
		Timestamp:     time.Now(),
		DurationMS:    duration.Milliseconds(),
		TaskRef:       taskRef,
		Tool:          tool,
		Version:       "1.0.0", // Can be configurable
	}
}

// NewToolMetaWithAuth creates metadata for an operation with authentication status
func NewToolMetaWithAuth(correlationID, taskRef, tool string, duration time.Duration, authStatus *AuthStatus, dataSource string) ToolMeta {
	return ToolMeta{
		CorrelationID: correlationID,
		Timestamp:     time.Now(),
		DurationMS:    duration.Milliseconds(),
		TaskRef:       taskRef,
		Tool:          tool,
		Version:       "1.0.0", // Can be configurable
		AuthStatus:    authStatus,
		DataSource:    dataSource,
	}
}

// GenerateCorrelationID generates a new correlation ID for request tracking
func GenerateCorrelationID() string {
	return uuid.New().String()
}

// RedactSensitiveData removes sensitive information from data structures
func RedactSensitiveData(data interface{}) interface{} {
	return redactValue(data)
}

// sensitiveKeys are field names that should be redacted
var sensitiveKeys = []string{
	"api_key", "apikey", "api-key",
	"token", "access_token", "refresh_token", "bearer_token",
	"password", "passwd", "pwd",
	"secret", "client_secret",
	"cookie", "session", "session_id",
	"authorization", "auth",
	"key", "private_key", "public_key",
	"credential", "credentials",
	"x-api-key", "x-auth-token",
}

// redactValue recursively redacts sensitive data from any value
func redactValue(v interface{}) interface{} {
	switch value := v.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, val := range value {
			if isSensitiveKey(k) {
				result[k] = "[REDACTED]"
			} else {
				result[k] = redactValue(val)
			}
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(value))
		for i, val := range value {
			result[i] = redactValue(val)
		}
		return result
	case map[interface{}]interface{}:
		result := make(map[interface{}]interface{})
		for k, val := range value {
			if keyStr, ok := k.(string); ok && isSensitiveKey(keyStr) {
				result[k] = "[REDACTED]"
			} else {
				result[k] = redactValue(val)
			}
		}
		return result
	default:
		return value
	}
}

// isSensitiveKey checks if a key name indicates sensitive data
func isSensitiveKey(key string) bool {
	keyLower := strings.ToLower(key)
	for _, sensitiveKey := range sensitiveKeys {
		if strings.Contains(keyLower, sensitiveKey) {
			return true
		}
	}
	return false
}

// FormatJSON formats a ToolOutput as pretty-printed JSON
func FormatJSON(output *ToolOutput) ([]byte, error) {
	// Apply redaction before formatting
	redactedOutput := &ToolOutput{
		OK:    output.OK,
		Data:  RedactSensitiveData(output.Data),
		Error: output.Error, // Errors shouldn't contain sensitive data
		Meta:  output.Meta,
	}

	return json.MarshalIndent(redactedOutput, "", "  ")
}

// FormatCompactJSON formats a ToolOutput as compact JSON
func FormatCompactJSON(output *ToolOutput) ([]byte, error) {
	// Apply redaction before formatting
	redactedOutput := &ToolOutput{
		OK:    output.OK,
		Data:  RedactSensitiveData(output.Data),
		Error: output.Error, // Errors shouldn't contain sensitive data
		Meta:  output.Meta,
	}

	return json.Marshal(redactedOutput)
}

// OutputWriter interface for writing tool outputs
type OutputWriter interface {
	WriteOutput(output *ToolOutput, quiet bool) error
}

// JSONOutputWriter writes tool outputs as JSON
type JSONOutputWriter struct{}

// WriteOutput writes the tool output as JSON to stdout
func (w *JSONOutputWriter) WriteOutput(output *ToolOutput, quiet bool) error {
	var jsonData []byte
	var err error

	if quiet {
		jsonData, err = FormatCompactJSON(output)
	} else {
		jsonData, err = FormatJSON(output)
	}

	if err != nil {
		return fmt.Errorf("failed to format JSON output: %w", err)
	}

	fmt.Println(string(jsonData))
	return nil
}

// NewJSONOutputWriter creates a new JSON output writer
func NewJSONOutputWriter() OutputWriter {
	return &JSONOutputWriter{}
}
