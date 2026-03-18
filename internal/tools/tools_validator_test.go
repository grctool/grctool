// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// ValidateTaskReference
// ---------------------------------------------------------------------------

func TestValidateTaskReference(t *testing.T) {
	t.Parallel()

	// Use a temp dir so the registry load silently fails (no data)
	// which is fine — we test fallback calculation path.
	v := NewValidator(t.TempDir())

	tests := map[string]struct {
		input       string
		wantValid   bool
		wantNorm    string // expected normalized value (empty = don't check)
		wantErrMsg  string // substring in first error message, if invalid
		wantWarning bool
	}{
		"empty string": {
			input:      "",
			wantValid:  false,
			wantErrMsg: "required",
		},
		"plain numeric ID": {
			input:     "328001",
			wantValid: true,
			wantNorm:  "328001",
		},
		"ET-1 dash format": {
			input:       "ET-1",
			wantValid:   true,
			wantNorm:    "327992",
			wantWarning: true,
		},
		"ET-101 dash format": {
			input:       "ET-101",
			wantValid:   true,
			wantNorm:    "328092",
			wantWarning: true,
		},
		"et-101 lowercase": {
			input:       "et-101",
			wantValid:   true,
			wantNorm:    "328092",
			wantWarning: true,
		},
		"ET 101 space format": {
			input:       "ET 101",
			wantValid:   true,
			wantNorm:    "328092",
			wantWarning: true,
		},
		"ET101 no separator": {
			input:       "ET101",
			wantValid:   true,
			wantNorm:    "328092",
			wantWarning: true,
		},
		"direct numeric in 32xxxx range": {
			input:     "327992",
			wantValid: true,
			wantNorm:  "327992",
		},
		"invalid format - random text": {
			input:      "foobar",
			wantValid:  false,
			wantErrMsg: "invalid task reference format",
		},
		"whitespace only": {
			input:      "   ",
			wantValid:  false,
			wantErrMsg: "invalid task reference format",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := v.ValidateTaskReference(tc.input)
			require.NoError(t, err, "ValidateTaskReference should not return an error")
			require.NotNil(t, result)

			assert.Equal(t, tc.wantValid, result.Valid, "valid flag mismatch")

			if tc.wantValid {
				assert.Empty(t, result.Errors, "expected no errors for valid input")
				if tc.wantNorm != "" {
					assert.Equal(t, tc.wantNorm, result.Normalized["task_ref"])
				}
				if tc.wantWarning {
					assert.NotEmpty(t, result.Warnings, "expected a warning for format conversion")
				}
			} else {
				assert.NotEmpty(t, result.Errors, "expected errors for invalid input")
				if tc.wantErrMsg != "" {
					assert.Contains(t, result.Errors[0].Message, tc.wantErrMsg)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ValidatePathSafety
// ---------------------------------------------------------------------------

func TestValidatePathSafety(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	v := NewValidator(dataDir)

	tests := map[string]struct {
		path      string
		wantValid bool
		wantRule  string // expected rule in first error
	}{
		"empty path": {
			path:      "",
			wantValid: false,
			wantRule:  "required",
		},
		"simple relative path": {
			path:      "docs/policies/test.json",
			wantValid: true,
		},
		"path traversal with double dots": {
			path:      "../../../etc/passwd",
			wantValid: false,
			wantRule:  "no_traversal",
		},
		"absolute path": {
			path:      "/etc/passwd",
			wantValid: false,
			wantRule:  "relative_only",
		},
		"dangerous .ssh path": {
			path:      "something/.ssh/keys",
			wantValid: false,
			wantRule:  "no_dangerous_paths",
		},
		"dangerous .env path": {
			path:      ".env",
			wantValid: false,
			wantRule:  "no_dangerous_paths",
		},
		"dangerous .git path": {
			path:      "repo/.git/config",
			wantValid: false,
			wantRule:  "no_dangerous_paths",
		},
		"valid nested path": {
			path:      "evidence/ET-0001/report.md",
			wantValid: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := v.ValidatePathSafety(tc.path)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, tc.wantValid, result.Valid)

			if !tc.wantValid && tc.wantRule != "" {
				require.NotEmpty(t, result.Errors)
				assert.Equal(t, tc.wantRule, result.Errors[0].Rule)
			}

			if tc.wantValid {
				assert.NotEmpty(t, result.Normalized["path"], "valid path should be normalized")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// NormalizeReferenceID — Evidence
// ---------------------------------------------------------------------------

func TestNormalizeReferenceID_Evidence(t *testing.T) {
	t.Parallel()

	v := NewValidator(t.TempDir())

	tests := map[string]struct {
		input   string
		want    string
		wantErr bool
	}{
		"ET-1":         {input: "ET-1", want: "ET-0001"},
		"ET-101":       {input: "ET-101", want: "ET-0101"},
		"ET-0047":      {input: "ET-0047", want: "ET-0047"},
		"et-47":        {input: "et-47", want: "ET-0047"},
		"ET 47":        {input: "ET 47", want: "ET-0047"},
		"ET47":         {input: "ET47", want: "ET-0047"},
		"plain numeric": {input: "47", want: "ET-0047"},
		"large number":  {input: "ET-9999", want: "ET-9999"},
		"invalid":       {input: "FOOBAR", wantErr: true},
		"negative":      {input: "-5", wantErr: true},
		"zero":          {input: "0", wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := v.NormalizeReferenceID(tc.input, "evidence")
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.want, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// NormalizeReferenceID — Policy
// ---------------------------------------------------------------------------

func TestNormalizeReferenceID_Policy(t *testing.T) {
	t.Parallel()

	v := NewValidator(t.TempDir())

	tests := map[string]struct {
		input   string
		want    string
		wantErr bool
	}{
		"POL-1":          {input: "POL-1", want: "POL-0001"},
		"POL-0042":       {input: "POL-0042", want: "POL-0042"},
		"pol-42":         {input: "pol-42", want: "POL-0042"},
		"POL 42":         {input: "POL 42", want: "POL-0042"},
		"POL42":          {input: "POL42", want: "POL-0042"},
		"plain numeric":  {input: "42", want: "POL-0042"},
		"invalid string": {input: "FOOBAR", wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := v.NormalizeReferenceID(tc.input, "policy")
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.want, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// NormalizeReferenceID — Control
// ---------------------------------------------------------------------------

func TestNormalizeReferenceID_Control(t *testing.T) {
	t.Parallel()

	v := NewValidator(t.TempDir())

	tests := map[string]struct {
		input   string
		want    string
		wantErr bool
	}{
		"CC-6.1 with dot":   {input: "CC-6.1", want: "CC-06_1"},
		"CC-06.1":           {input: "CC-06.1", want: "CC-06_1"},
		"AC-1 no subsection": {input: "AC-1", want: "AC-01"},
		"AC1 no dash":       {input: "AC1", want: "AC-01"},
		"ac-1 lowercase":    {input: "ac-1", want: "AC-01"},
		"SO-19":             {input: "SO-19", want: "SO-19"},
		"CC 6.1 with space": {input: "CC 6.1", want: "CC-06_1"},
		"CC 6 with space":   {input: "CC 6", want: "CC-06"},
		"CC6.1 no dash dot": {input: "CC6.1", want: "CC-06_1"},
		"CC6 no dash no dot": {input: "CC6", want: "CC-06"},
		"plain numeric":     {input: "42", wantErr: true},
		"invalid prefix":    {input: "ZZ-1", wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := v.NormalizeReferenceID(tc.input, "control")
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.want, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// NormalizeReferenceID — unsupported doc type
// ---------------------------------------------------------------------------

func TestNormalizeReferenceID_UnsupportedDocType(t *testing.T) {
	t.Parallel()

	v := NewValidator(t.TempDir())
	_, err := v.NormalizeReferenceID("test", "unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported document type")
}

// ---------------------------------------------------------------------------
// ValidateParameters
// ---------------------------------------------------------------------------

func TestValidateParameters(t *testing.T) {
	t.Parallel()

	v := NewValidator(t.TempDir())

	t.Run("required parameter missing", func(t *testing.T) {
		t.Parallel()

		rules := map[string]ValidationRule{
			"name": {Required: true, Type: "string"},
		}
		result, err := v.ValidateParameters(map[string]interface{}{}, rules)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Equal(t, "required", result.Errors[0].Rule)
	})

	t.Run("required parameter present and valid", func(t *testing.T) {
		t.Parallel()

		rules := map[string]ValidationRule{
			"name": {Required: true, Type: "string"},
		}
		result, err := v.ValidateParameters(map[string]interface{}{"name": "hello"}, rules)
		require.NoError(t, err)
		assert.True(t, result.Valid)
	})

	t.Run("wrong type - int expected", func(t *testing.T) {
		t.Parallel()

		rules := map[string]ValidationRule{
			"count": {Required: true, Type: "int"},
		}
		result, err := v.ValidateParameters(map[string]interface{}{"count": "not-a-number"}, rules)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Equal(t, "type", result.Errors[0].Rule)
	})

	t.Run("wrong type - bool expected", func(t *testing.T) {
		t.Parallel()

		rules := map[string]ValidationRule{
			"active": {Required: true, Type: "bool"},
		}
		result, err := v.ValidateParameters(map[string]interface{}{"active": "not-a-bool"}, rules)
		require.NoError(t, err)
		assert.False(t, result.Valid)
	})

	t.Run("min length violation", func(t *testing.T) {
		t.Parallel()

		rules := map[string]ValidationRule{
			"name": {Required: true, Type: "string", MinLength: 5},
		}
		result, err := v.ValidateParameters(map[string]interface{}{"name": "ab"}, rules)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Equal(t, "min_length", result.Errors[0].Rule)
	})

	t.Run("max length violation", func(t *testing.T) {
		t.Parallel()

		rules := map[string]ValidationRule{
			"name": {Required: true, Type: "string", MaxLength: 3},
		}
		result, err := v.ValidateParameters(map[string]interface{}{"name": "toolong"}, rules)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Equal(t, "max_length", result.Errors[0].Rule)
	})

	t.Run("pattern mismatch", func(t *testing.T) {
		t.Parallel()

		rules := map[string]ValidationRule{
			"code": {Required: true, Type: "string", Pattern: `^[A-Z]{2}-\d+$`},
		}
		result, err := v.ValidateParameters(map[string]interface{}{"code": "bad"}, rules)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Equal(t, "pattern", result.Errors[0].Rule)
	})

	t.Run("pattern match", func(t *testing.T) {
		t.Parallel()

		rules := map[string]ValidationRule{
			"code": {Required: true, Type: "string", Pattern: `^[A-Z]{2}-\d+$`},
		}
		result, err := v.ValidateParameters(map[string]interface{}{"code": "CC-01"}, rules)
		require.NoError(t, err)
		assert.True(t, result.Valid)
	})

	t.Run("allowed values - valid", func(t *testing.T) {
		t.Parallel()

		rules := map[string]ValidationRule{
			"level": {Required: true, Type: "string", AllowedValues: []string{"basic", "standard", "comprehensive"}},
		}
		result, err := v.ValidateParameters(map[string]interface{}{"level": "standard"}, rules)
		require.NoError(t, err)
		assert.True(t, result.Valid)
	})

	t.Run("allowed values - invalid", func(t *testing.T) {
		t.Parallel()

		rules := map[string]ValidationRule{
			"level": {Required: true, Type: "string", AllowedValues: []string{"basic", "standard"}},
		}
		result, err := v.ValidateParameters(map[string]interface{}{"level": "extreme"}, rules)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Equal(t, "allowed_values", result.Errors[0].Rule)
	})

	t.Run("optional parameter absent is OK", func(t *testing.T) {
		t.Parallel()

		rules := map[string]ValidationRule{
			"opt": {Required: false, Type: "string"},
		}
		result, err := v.ValidateParameters(map[string]interface{}{}, rules)
		require.NoError(t, err)
		assert.True(t, result.Valid)
	})

	t.Run("path safety validation", func(t *testing.T) {
		t.Parallel()

		rules := map[string]ValidationRule{
			"file": {Required: true, Type: "path", PathSafety: true},
		}
		result, err := v.ValidateParameters(map[string]interface{}{"file": "../../etc/passwd"}, rules)
		require.NoError(t, err)
		assert.False(t, result.Valid)
	})
}

// ---------------------------------------------------------------------------
// ValidationError.Error()
// ---------------------------------------------------------------------------

func TestValidationError_Error(t *testing.T) {
	t.Parallel()

	ve := ValidationError{
		Field:   "task_ref",
		Value:   "",
		Rule:    "required",
		Message: "task reference is required",
	}
	assert.Contains(t, ve.Error(), "task_ref")
	assert.Contains(t, ve.Error(), "task reference is required")
}
