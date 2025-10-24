// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package provenance

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/models"
)

func TestNewService(t *testing.T) {
	version := "1.0.0-test"
	svc := NewService(version)

	if svc == nil {
		t.Fatal("NewService() returned nil")
	}

	if svc.toolVersion != version {
		t.Errorf("NewService() toolVersion = %v, want %v", svc.toolVersion, version)
	}
}

func TestCaptureProvenance_NonGitDirectory(t *testing.T) {
	// Create a temporary non-git directory that is definitely outside any git repo
	// We'll create it in /tmp which should not be in a git repo
	tmpDir := t.TempDir()

	// Verify we're really outside a git repo by checking if .git exists in parents
	if findGitRoot(tmpDir) != "" {
		t.Skip("Temp directory is inside a git repository, skipping non-git test")
	}

	svc := NewService("1.0.0-test")
	prov, err := svc.CaptureProvenance(tmpDir)

	if err != nil {
		t.Fatalf("CaptureProvenance() error = %v, want nil (should handle non-git gracefully)", err)
	}

	if prov == nil {
		t.Fatal("CaptureProvenance() returned nil provenance")
	}

	// Should have basic info
	if prov.ToolVersion != "1.0.0-test" {
		t.Errorf("Provenance.ToolVersion = %v, want %v", prov.ToolVersion, "1.0.0-test")
	}

	if prov.CollectedAt.IsZero() {
		t.Error("Provenance.CollectedAt is zero")
	}

	if prov.CollectorHost == "" {
		t.Error("Provenance.CollectorHost is empty")
	}

	// Git fields should be empty for non-git directory
	if prov.GitCommitSHA != "" {
		t.Errorf("Provenance.GitCommitSHA = %v, want empty for non-git dir", prov.GitCommitSHA)
	}

	if prov.GitBranch != "" {
		t.Errorf("Provenance.GitBranch = %v, want empty for non-git dir", prov.GitBranch)
	}
}

func TestCaptureProvenance_GitDirectory(t *testing.T) {
	// This test requires a git repository - skip if not in one
	cwd, err := os.Getwd()
	if err != nil {
		t.Skip("Cannot get current directory")
	}

	// Try to find git root by walking up
	gitDir := findGitRoot(cwd)
	if gitDir == "" {
		t.Skip("Not in a git repository, skipping git-specific test")
	}

	svc := NewService("1.0.0-test")
	prov, err := svc.CaptureProvenance(gitDir)

	if err != nil {
		t.Fatalf("CaptureProvenance() error = %v", err)
	}

	if prov == nil {
		t.Fatal("CaptureProvenance() returned nil provenance")
	}

	// Should have git info
	if prov.GitCommitSHA == "" {
		t.Error("Provenance.GitCommitSHA is empty for git directory")
	}

	if len(prov.GitCommitSHA) != 40 {
		t.Errorf("Provenance.GitCommitSHA length = %d, want 40", len(prov.GitCommitSHA))
	}

	if prov.GitBranch == "" {
		t.Error("Provenance.GitBranch is empty for git directory")
	}

	// Remote URL and tags are optional, so don't fail if empty
	t.Logf("Git commit: %s", prov.GitCommitSHA)
	t.Logf("Git branch: %s", prov.GitBranch)
	t.Logf("Git remote: %s", prov.GitRemoteURL)
	t.Logf("Git tags: %v", prov.GitTags)
}

func TestValidateProvenance(t *testing.T) {
	svc := NewService("1.0.0-test")

	tests := []struct {
		name    string
		prov    *models.Provenance
		wantErr bool
	}{
		{
			name:    "nil provenance",
			prov:    nil,
			wantErr: true,
		},
		{
			name: "missing collection timestamp",
			prov: &models.Provenance{
				ToolVersion: "1.0.0",
			},
			wantErr: true,
		},
		{
			name: "valid provenance without git",
			prov: &models.Provenance{
				CollectedAt: time.Now(),
				ToolVersion: "1.0.0",
			},
			wantErr: false,
		},
		{
			name: "valid provenance with git",
			prov: &models.Provenance{
				CollectedAt:  time.Now(),
				ToolVersion:  "1.0.0",
				GitCommitSHA: "1234567890abcdef1234567890abcdef12345678",
				GitBranch:    "main",
			},
			wantErr: false,
		},
		{
			name: "invalid git commit SHA (too short)",
			prov: &models.Provenance{
				CollectedAt:  time.Now(),
				ToolVersion:  "1.0.0",
				GitCommitSHA: "1234567",
				GitBranch:    "main",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.ValidateProvenance(tt.prov)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProvenance() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCaptureForDataDir(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService("1.0.0-test")

	prov, err := svc.CaptureForDataDir(tmpDir)
	if err != nil {
		t.Fatalf("CaptureForDataDir() error = %v", err)
	}

	if prov == nil {
		t.Fatal("CaptureForDataDir() returned nil")
	}

	if prov.ToolVersion != "1.0.0-test" {
		t.Errorf("ToolVersion = %v, want %v", prov.ToolVersion, "1.0.0-test")
	}
}

func TestCaptureForSourceFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.tf")

	// Create test file
	if err := os.WriteFile(testFile, []byte("# test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	svc := NewService("1.0.0-test")
	prov, err := svc.CaptureForSourceFile(testFile)

	if err != nil {
		t.Fatalf("CaptureForSourceFile() error = %v", err)
	}

	if prov == nil {
		t.Fatal("CaptureForSourceFile() returned nil")
	}

	if prov.CollectedAt.IsZero() {
		t.Error("CollectedAt is zero")
	}
}

// Helper function to find git root
func findGitRoot(startPath string) string {
	current := startPath
	for {
		gitDir := filepath.Join(current, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			return current
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached filesystem root
			return ""
		}
		current = parent
	}
}
