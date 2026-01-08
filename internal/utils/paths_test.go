// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"path/filepath"
	"testing"
)

func TestValidateRelativePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid relative path",
			path:    "evidence/ET-0001/2025-Q4/file.md",
			wantErr: false,
		},
		{
			name:    "valid nested relative path",
			path:    "data/docs/policies/POL-0001.json",
			wantErr: false,
		},
		{
			name:    "absolute unix path",
			path:    "/home/user/evidence/file.md",
			wantErr: true,
			errMsg:  "path is absolute",
		},
		{
			name:    "absolute /Users path",
			path:    "/Users/john/Documents/evidence.md",
			wantErr: true,
			errMsg:  "path is absolute",
		},
		{
			name:    "absolute /pool path",
			path:    "/pool0/erik/Projects/file.tf",
			wantErr: true,
			errMsg:  "path is absolute",
		},
		{
			name:    "windows absolute path",
			path:    "C:\\Users\\john\\evidence.md",
			wantErr: true,
			errMsg:  "absolute prefix",
		},
		{
			name:    "parent directory traversal",
			path:    "../../../etc/passwd",
			wantErr: true,
			errMsg:  "contains parent directory references",
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
			errMsg:  "path is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRelativePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRelativePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateRelativePath() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

func TestRelativizePathFromDataDir(t *testing.T) {
	tests := []struct {
		name    string
		absPath string
		dataDir string
		want    string
		wantErr bool
	}{
		{
			name:    "path inside data dir",
			absPath: "/tmp/grctool-test/data/evidence/ET-0001/file.md",
			dataDir: "/tmp/grctool-test/data",
			want:    "evidence/ET-0001/file.md",
			wantErr: false,
		},
		{
			name:    "path at data dir root",
			absPath: "/tmp/grctool-test/data/file.md",
			dataDir: "/tmp/grctool-test/data",
			want:    "file.md",
			wantErr: false,
		},
		{
			name:    "path outside data dir",
			absPath: "/tmp/other-location/file.md",
			dataDir: "/tmp/grctool-test/data",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RelativizePathFromDataDir(tt.absPath, tt.dataDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("RelativizePathFromDataDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("RelativizePathFromDataDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSanitizePathForEvidence(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		dataDir string
		wantErr bool
	}{
		{
			name:    "already relative path",
			path:    "evidence/ET-0001/file.md",
			dataDir: "/tmp/data",
			wantErr: false,
		},
		{
			name:    "absolute path inside data dir",
			path:    "/tmp/data/evidence/ET-0001/file.md",
			dataDir: "/tmp/data",
			wantErr: false,
		},
		{
			name:    "absolute path outside data dir",
			path:    "/home/user/file.md",
			dataDir: "/tmp/data",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := SanitizePathForEvidence(tt.path, tt.dataDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("SanitizePathForEvidence() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsPathWithinDirectory(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		directory string
		want      bool
	}{
		{
			name:      "path inside directory",
			path:      "/tmp/data/evidence/file.md",
			directory: "/tmp/data",
			want:      true,
		},
		{
			name:      "path outside directory",
			path:      "/home/user/file.md",
			directory: "/tmp/data",
			want:      false,
		},
		{
			name:      "path at directory root",
			path:      "/tmp/data",
			directory: "/tmp/data",
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IsPathWithinDirectory(tt.path, tt.directory)
			if err != nil {
				t.Errorf("IsPathWithinDirectory() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("IsPathWithinDirectory() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizePathSeparators(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "unix path unchanged",
			path: "evidence/ET-0001/file.md",
			want: "evidence/ET-0001/file.md",
		},
		{
			name: "windows path converted",
			path: filepath.Join("evidence", "ET-0001", "file.md"),
			want: "evidence/ET-0001/file.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizePathSeparators(tt.path)
			if got != tt.want {
				t.Errorf("NormalizePathSeparators() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJoinRelativePaths(t *testing.T) {
	tests := []struct {
		name       string
		components []string
		want       string
		wantErr    bool
	}{
		{
			name:       "valid relative components",
			components: []string{"evidence", "ET-0001", "file.md"},
			want:       "evidence/ET-0001/file.md",
			wantErr:    false,
		},
		{
			name:       "absolute component",
			components: []string{"/tmp", "evidence", "file.md"},
			want:       "",
			wantErr:    true,
		},
		{
			name:       "single component",
			components: []string{"file.md"},
			want:       "file.md",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := JoinRelativePaths(tt.components...)
			if (err != nil) != tt.wantErr {
				t.Errorf("JoinRelativePaths() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("JoinRelativePaths() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
