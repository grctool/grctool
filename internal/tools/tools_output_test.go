// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// NewSuccessOutput / NewErrorOutput
// ---------------------------------------------------------------------------

func TestNewSuccessOutput(t *testing.T) {
	t.Parallel()

	meta := NewToolMeta("corr-123", "ET-0047", "test-tool", 150*time.Millisecond)
	output := NewSuccessOutput(map[string]string{"key": "value"}, meta)

	assert.True(t, output.OK)
	assert.NotNil(t, output.Data)
	assert.Nil(t, output.Error)
	assert.Equal(t, "corr-123", output.Meta.CorrelationID)
	assert.Equal(t, "ET-0047", output.Meta.TaskRef)
	assert.Equal(t, "test-tool", output.Meta.Tool)
	assert.Equal(t, int64(150), output.Meta.DurationMS)
	assert.Equal(t, "1.0.0", output.Meta.Version)
}

func TestNewErrorOutput(t *testing.T) {
	t.Parallel()

	toolErr := NewToolError(ErrorCodeValidation, "bad input", "corr-456", map[string]interface{}{
		"field": "task_ref",
	})
	meta := NewToolMeta("corr-456", "", "test-tool", 10*time.Millisecond)
	output := NewErrorOutput(toolErr, meta)

	assert.False(t, output.OK)
	assert.Nil(t, output.Data)
	require.NotNil(t, output.Error)
	assert.Equal(t, ErrorCodeValidation, output.Error.Code)
	assert.Equal(t, "bad input", output.Error.Message)
	assert.Equal(t, "corr-456", output.Error.CorrelationID)
	assert.Equal(t, "task_ref", output.Error.Details["field"])
}

// ---------------------------------------------------------------------------
// NewToolMeta / NewToolMetaWithAuth
// ---------------------------------------------------------------------------

func TestNewToolMeta(t *testing.T) {
	t.Parallel()

	meta := NewToolMeta("abc-123", "ET-0001", "github-permissions", 500*time.Millisecond)

	assert.Equal(t, "abc-123", meta.CorrelationID)
	assert.Equal(t, "ET-0001", meta.TaskRef)
	assert.Equal(t, "github-permissions", meta.Tool)
	assert.Equal(t, int64(500), meta.DurationMS)
	assert.Equal(t, "1.0.0", meta.Version)
	assert.Nil(t, meta.AuthStatus)
	assert.WithinDuration(t, time.Now(), meta.Timestamp, 2*time.Second)
}

func TestNewToolMetaWithAuth(t *testing.T) {
	t.Parallel()

	authStatus := &AuthStatus{
		Authenticated: true,
		Provider:      "github",
		CacheUsed:     true,
		TokenPresent:  true,
	}
	meta := NewToolMetaWithAuth("corr-1", "ET-0001", "github-tool", 200*time.Millisecond, authStatus, "api")

	require.NotNil(t, meta.AuthStatus)
	assert.True(t, meta.AuthStatus.Authenticated)
	assert.Equal(t, "github", meta.AuthStatus.Provider)
	assert.True(t, meta.AuthStatus.CacheUsed)
	assert.Equal(t, "api", meta.DataSource)
}

// ---------------------------------------------------------------------------
// NewToolError
// ---------------------------------------------------------------------------

func TestNewToolError(t *testing.T) {
	t.Parallel()

	err := NewToolError(ErrorCodeExecution, "something broke", "corr-x", nil)
	assert.Equal(t, ErrorCodeExecution, err.Code)
	assert.Equal(t, "something broke", err.Message)
	assert.Equal(t, "corr-x", err.CorrelationID)
	assert.Nil(t, err.Details)
	assert.WithinDuration(t, time.Now(), err.Timestamp, 2*time.Second)
}

// ---------------------------------------------------------------------------
// GenerateCorrelationID
// ---------------------------------------------------------------------------

func TestGenerateCorrelationID(t *testing.T) {
	t.Parallel()

	id1 := GenerateCorrelationID()
	id2 := GenerateCorrelationID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2, "should generate unique IDs")
	// UUID v4 format: 8-4-4-4-12
	assert.Regexp(t, `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`, id1)
}

// ---------------------------------------------------------------------------
// Error codes are distinct constants
// ---------------------------------------------------------------------------

func TestErrorCodes(t *testing.T) {
	t.Parallel()

	codes := []string{
		ErrorCodeValidation,
		ErrorCodeExecution,
		ErrorCodePathSafety,
		ErrorCodeTaskNotFound,
		ErrorCodeUnauthorized,
		ErrorCodeInternal,
		ErrorCodeConfigError,
		ErrorCodeServiceError,
		ErrorCodeNetworkError,
	}

	seen := make(map[string]bool)
	for _, code := range codes {
		assert.NotEmpty(t, code)
		assert.False(t, seen[code], "duplicate error code: %s", code)
		seen[code] = true
	}
}

// ---------------------------------------------------------------------------
// RedactSensitiveData
// ---------------------------------------------------------------------------

func TestRedactSensitiveData(t *testing.T) {
	t.Parallel()

	t.Run("redacts top-level sensitive keys", func(t *testing.T) {
		t.Parallel()

		data := map[string]interface{}{
			"api_key": "secret123",
			"name":    "visible",
			"token":   "tok-abc",
		}
		redacted := RedactSensitiveData(data).(map[string]interface{})

		assert.Equal(t, "[REDACTED]", redacted["api_key"])
		assert.Equal(t, "visible", redacted["name"])
		assert.Equal(t, "[REDACTED]", redacted["token"])
	})

	t.Run("redacts nested map keys", func(t *testing.T) {
		t.Parallel()

		data := map[string]interface{}{
			"config": map[string]interface{}{
				"password":   "hunter2",
				"host":       "localhost",
				"secret_key": "s3cr3t",
			},
		}
		redacted := RedactSensitiveData(data).(map[string]interface{})
		nested := redacted["config"].(map[string]interface{})

		assert.Equal(t, "[REDACTED]", nested["password"])
		assert.Equal(t, "localhost", nested["host"])
		assert.Equal(t, "[REDACTED]", nested["secret_key"])
	})

	t.Run("redacts inside arrays", func(t *testing.T) {
		t.Parallel()

		data := []interface{}{
			map[string]interface{}{
				"authorization": "Bearer xyz",
				"url":           "https://example.com",
			},
		}
		redacted := RedactSensitiveData(data).([]interface{})
		first := redacted[0].(map[string]interface{})

		assert.Equal(t, "[REDACTED]", first["authorization"])
		assert.Equal(t, "https://example.com", first["url"])
	})

	t.Run("preserves non-map non-slice values", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "hello", RedactSensitiveData("hello"))
		assert.Equal(t, 42, RedactSensitiveData(42))
		assert.Equal(t, true, RedactSensitiveData(true))
		assert.Nil(t, RedactSensitiveData(nil))
	})

	t.Run("case insensitive key matching", func(t *testing.T) {
		t.Parallel()

		data := map[string]interface{}{
			"API_KEY":       "key1",
			"Access_Token":  "tok1",
			"X-Auth-Token":  "tok2",
			"Client_Secret": "sec1",
		}
		redacted := RedactSensitiveData(data).(map[string]interface{})

		for key := range data {
			assert.Equal(t, "[REDACTED]", redacted[key], "key %s should be redacted", key)
		}
	})

	t.Run("handles map[interface{}]interface{}", func(t *testing.T) {
		t.Parallel()

		data := map[interface{}]interface{}{
			"token":   "secret",
			"visible": "value",
			42:        "numeric-key",
		}
		redacted := RedactSensitiveData(data).(map[interface{}]interface{})

		assert.Equal(t, "[REDACTED]", redacted["token"])
		assert.Equal(t, "value", redacted["visible"])
		assert.Equal(t, "numeric-key", redacted[42])
	})
}

// ---------------------------------------------------------------------------
// isSensitiveKey
// ---------------------------------------------------------------------------

func TestIsSensitiveKey(t *testing.T) {
	t.Parallel()

	sensitiveExamples := []string{
		"api_key", "API_KEY", "apiKey",
		"token", "access_token", "refresh_token", "bearer_token",
		"password", "passwd", "pwd",
		"secret", "client_secret",
		"cookie", "session", "session_id",
		"authorization", "auth",
		"key", "private_key", "public_key",
		"credential", "credentials",
		"x-api-key", "x-auth-token",
	}

	for _, key := range sensitiveExamples {
		assert.True(t, isSensitiveKey(key), "should detect %q as sensitive", key)
	}

	safeExamples := []string{"name", "host", "port", "url", "description", "status"}
	for _, key := range safeExamples {
		assert.False(t, isSensitiveKey(key), "should not detect %q as sensitive", key)
	}
}

// ---------------------------------------------------------------------------
// FormatJSON / FormatCompactJSON
// ---------------------------------------------------------------------------

func TestFormatJSON(t *testing.T) {
	t.Parallel()

	meta := ToolMeta{
		CorrelationID: "test-corr",
		Timestamp:     time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
		DurationMS:    42,
		Tool:          "test-tool",
		Version:       "1.0.0",
	}
	output := NewSuccessOutput(map[string]interface{}{
		"result":  "pass",
		"api_key": "should-be-redacted",
	}, meta)

	jsonBytes, err := FormatJSON(output)
	require.NoError(t, err)

	// Parse back and verify
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonBytes, &parsed))

	assert.Equal(t, true, parsed["ok"])

	data := parsed["data"].(map[string]interface{})
	assert.Equal(t, "pass", data["result"])
	assert.Equal(t, "[REDACTED]", data["api_key"], "sensitive data should be redacted")

	// Pretty-printed JSON should contain newlines
	assert.Contains(t, string(jsonBytes), "\n")
}

func TestFormatCompactJSON(t *testing.T) {
	t.Parallel()

	meta := ToolMeta{
		CorrelationID: "test-corr",
		Timestamp:     time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
		DurationMS:    10,
		Tool:          "test-tool",
		Version:       "1.0.0",
	}
	output := NewSuccessOutput("simple", meta)

	jsonBytes, err := FormatCompactJSON(output)
	require.NoError(t, err)

	// Compact JSON should NOT contain indentation newlines (just one line)
	// (it may technically contain no newlines at all)
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonBytes, &parsed))
	assert.Equal(t, true, parsed["ok"])
}

// ---------------------------------------------------------------------------
// ToolOutput JSON serialization round-trip
// ---------------------------------------------------------------------------

func TestToolOutput_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	original := &ToolOutput{
		OK:   true,
		Data: map[string]interface{}{"count": float64(42)},
		Meta: ToolMeta{
			CorrelationID: "round-trip-test",
			DurationMS:    100,
			Tool:          "test",
			Version:       "1.0.0",
		},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var restored ToolOutput
	require.NoError(t, json.Unmarshal(data, &restored))

	assert.Equal(t, original.OK, restored.OK)
	assert.Equal(t, original.Meta.CorrelationID, restored.Meta.CorrelationID)
	assert.Equal(t, original.Meta.DurationMS, restored.Meta.DurationMS)
}

// ---------------------------------------------------------------------------
// NewJSONOutputWriter
// ---------------------------------------------------------------------------

func TestNewJSONOutputWriter(t *testing.T) {
	t.Parallel()

	writer := NewJSONOutputWriter()
	assert.NotNil(t, writer)
}
