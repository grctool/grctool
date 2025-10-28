// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"testing"
)

func TestValidateCollectorURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "Valid URL",
			url:     "https://openapi.tugboatlogic.com/api/v0/evidence/collector/805/",
			wantErr: false,
		},
		{
			name:    "Valid URL with different collector ID",
			url:     "https://openapi.tugboatlogic.com/api/v0/evidence/collector/123456/",
			wantErr: false,
		},
		{
			name:    "Missing trailing slash",
			url:     "https://openapi.tugboatlogic.com/api/v0/evidence/collector/805",
			wantErr: true,
		},
		{
			name:    "HTTP not HTTPS",
			url:     "http://openapi.tugboatlogic.com/api/v0/evidence/collector/805/",
			wantErr: true,
		},
		{
			name:    "Wrong domain",
			url:     "https://example.com/api/v0/evidence/collector/805/",
			wantErr: true,
		},
		{
			name:    "Invalid collector ID (non-numeric)",
			url:     "https://openapi.tugboatlogic.com/api/v0/evidence/collector/abc/",
			wantErr: true,
		},
		{
			name:    "Wrong API path",
			url:     "https://openapi.tugboatlogic.com/api/v1/evidence/collector/805/",
			wantErr: true,
		},
		{
			name:    "Missing collector ID",
			url:     "https://openapi.tugboatlogic.com/api/v0/evidence/collector/",
			wantErr: true,
		},
		{
			name:    "Empty URL",
			url:     "",
			wantErr: true,
		},
		{
			name:    "Invalid URL format",
			url:     "not-a-url",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCollectorURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCollectorURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateCollectorURL_ErrorMessages(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		wantErrContain string
	}{
		{
			name:           "HTTP shows scheme error",
			url:            "http://openapi.tugboatlogic.com/api/v0/evidence/collector/805/",
			wantErrContain: "must use HTTPS",
		},
		{
			name:           "Wrong domain shows pattern error",
			url:            "https://example.com/api/v0/evidence/collector/805/",
			wantErrContain: "does not match",
		},
		{
			name:           "Missing slash shows pattern error",
			url:            "https://openapi.tugboatlogic.com/api/v0/evidence/collector/805",
			wantErrContain: "does not match",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCollectorURL(tt.url)
			if err == nil {
				t.Errorf("validateCollectorURL() expected error containing %q, got nil", tt.wantErrContain)
				return
			}
			if !containsString(err.Error(), tt.wantErrContain) {
				t.Errorf("validateCollectorURL() error = %v, want error containing %q", err, tt.wantErrContain)
			}
		})
	}
}

// Helper function to check if string contains substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
