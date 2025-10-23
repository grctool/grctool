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

package testdata

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type TestDataManager struct {
	baseDir string
}

func NewTestDataManager(t *testing.T) *TestDataManager {
	// Get the current working directory and construct the absolute path
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Look for grctool root directory
	for !strings.HasSuffix(wd, "grctool") && wd != "/" {
		wd = filepath.Dir(wd)
	}

	baseDir := filepath.Join(wd, "test", "testdata")
	return &TestDataManager{baseDir: baseDir}
}

// Load test fixtures by category
func (m *TestDataManager) LoadTerraformFixture(name string) []byte {
	return m.loadFile(filepath.Join("terraform", name+".tf"))
}

func (m *TestDataManager) LoadGitHubFixture(name string) []byte {
	return m.loadFile(filepath.Join("github", name+".json"))
}

func (m *TestDataManager) LoadConfigFixture(name string) []byte {
	return m.loadFile(filepath.Join("configs", name+".yaml"))
}

func (m *TestDataManager) LoadEvidenceFixture(name string) []byte {
	return m.loadFile(filepath.Join("evidence", name+".json"))
}

// Load file contents
func (m *TestDataManager) loadFile(relativePath string) []byte {
	fullPath := filepath.Join(m.baseDir, relativePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		panic("Failed to load test fixture: " + fullPath + " - " + err.Error())
	}
	return content
}

// Create temporary test environments
func (m *TestDataManager) CreateTempTerraformProject(t *testing.T, fixtures []string) string {
	tempDir := t.TempDir()
	for _, fixture := range fixtures {
		content := m.LoadTerraformFixture(fixture)
		filePath := filepath.Join(tempDir, fixture+".tf")
		writeFile(t, filePath, content)
	}
	return tempDir
}

func (m *TestDataManager) CreateTempConfig(t *testing.T, configName string) string {
	tempDir := t.TempDir()
	content := m.LoadConfigFixture(configName)
	configPath := filepath.Join(tempDir, ".grctool.yaml")
	writeFile(t, configPath, content)
	return configPath
}

func (m *TestDataManager) CreateTempGitHubProject(t *testing.T, fixtures []string) string {
	tempDir := t.TempDir()
	for _, fixture := range fixtures {
		content := m.LoadGitHubFixture(fixture)
		filePath := filepath.Join(tempDir, fixture+".json")
		writeFile(t, filePath, content)
	}
	return tempDir
}

// Create complete test workspace with all fixture types
func (m *TestDataManager) CreateCompleteTestWorkspace(t *testing.T, config TestWorkspaceConfig) string {
	tempDir := t.TempDir()

	// Create terraform files
	if len(config.TerraformFixtures) > 0 {
		tfDir := filepath.Join(tempDir, "terraform")
		require.NoError(t, os.MkdirAll(tfDir, 0755))
		for _, fixture := range config.TerraformFixtures {
			content := m.LoadTerraformFixture(fixture)
			filePath := filepath.Join(tfDir, fixture+".tf")
			writeFile(t, filePath, content)
		}
	}

	// Create GitHub files
	if len(config.GitHubFixtures) > 0 {
		ghDir := filepath.Join(tempDir, ".github")
		require.NoError(t, os.MkdirAll(ghDir, 0755))
		for _, fixture := range config.GitHubFixtures {
			content := m.LoadGitHubFixture(fixture)
			filePath := filepath.Join(ghDir, fixture+".json")
			writeFile(t, filePath, content)
		}
	}

	// Create evidence files
	if len(config.EvidenceFixtures) > 0 {
		evidenceDir := filepath.Join(tempDir, "evidence")
		require.NoError(t, os.MkdirAll(evidenceDir, 0755))
		for _, fixture := range config.EvidenceFixtures {
			content := m.LoadEvidenceFixture(fixture)
			filePath := filepath.Join(evidenceDir, fixture+".json")
			writeFile(t, filePath, content)
		}
	}

	// Create config file
	if config.ConfigFixture != "" {
		content := m.LoadConfigFixture(config.ConfigFixture)
		configPath := filepath.Join(tempDir, ".grctool.yaml")
		writeFile(t, configPath, content)
	}

	return tempDir
}

type TestWorkspaceConfig struct {
	TerraformFixtures []string
	GitHubFixtures    []string
	EvidenceFixtures  []string
	ConfigFixture     string
}

// Helper function to write files safely
func writeFile(t *testing.T, filePath string, content []byte) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	err := os.MkdirAll(dir, 0755)
	require.NoError(t, err)

	// Write the file
	err = os.WriteFile(filePath, content, 0644)
	require.NoError(t, err)
}

// Quick access methods for common test scenarios
func (m *TestDataManager) CreateMinimalTestEnv(t *testing.T) string {
	return m.CreateCompleteTestWorkspace(t, TestWorkspaceConfig{
		TerraformFixtures: []string{"basic_s3", "encryption"},
		ConfigFixture:     "minimal",
	})
}

func (m *TestDataManager) CreateCompleteTestEnv(t *testing.T) string {
	return m.CreateCompleteTestWorkspace(t, TestWorkspaceConfig{
		TerraformFixtures: []string{"multi_az", "encryption", "iam_roles", "network_security"},
		GitHubFixtures:    []string{"repository", "collaborators", "workflows"},
		EvidenceFixtures:  []string{"et96_sample", "et103_sample"},
		ConfigFixture:     "complete",
	})
}

func (m *TestDataManager) CreateInvalidTestEnv(t *testing.T) string {
	return m.CreateCompleteTestWorkspace(t, TestWorkspaceConfig{
		TerraformFixtures: []string{"invalid_syntax"},
		ConfigFixture:     "invalid",
	})
}
