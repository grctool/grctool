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

//go:build !e2e && !functional

package helpers

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var updateGoldenFiles = flag.Bool("update", false, "update golden files")

// GoldenFile helps manage golden file tests for comparing actual output with expected output
type GoldenFile struct {
	baseDir  string
	testName string
}

// NewGoldenFile creates a new GoldenFile instance
// baseDir should be the path to the testdata directory (e.g., "testdata" or "../../testdata")
// testName is typically t.Name() to create test-specific golden files
func NewGoldenFile(baseDir, testName string) *GoldenFile {
	return &GoldenFile{
		baseDir:  baseDir,
		testName: testName,
	}
}

// Assert compares actual output with the golden file
// If UPDATE_GOLDEN environment variable is set or -update flag is provided,
// it will update the golden file instead of comparing
func (g *GoldenFile) Assert(t *testing.T, actual []byte, filename string) {
	t.Helper()

	goldenPath := filepath.Join(g.baseDir, filename)

	// Check if we should update golden files
	shouldUpdate := *updateGoldenFiles || os.Getenv("UPDATE_GOLDEN") != ""

	if shouldUpdate {
		g.updateGoldenFile(t, actual, goldenPath)
		return
	}

	g.compareWithGoldenFile(t, actual, goldenPath)
}

// AssertString is a convenience method for string comparisons
func (g *GoldenFile) AssertString(t *testing.T, actual string, filename string) {
	g.Assert(t, []byte(actual), filename)
}

// AssertJSON compares JSON output, ensuring consistent formatting
func (g *GoldenFile) AssertJSON(t *testing.T, actual []byte, filename string) {
	// For JSON, we might want to pretty-print or normalize formatting
	g.Assert(t, actual, filename)
}

// updateGoldenFile writes the actual output to the golden file
func (g *GoldenFile) updateGoldenFile(t *testing.T, actual []byte, goldenPath string) {
	t.Helper()

	// Ensure directory exists
	dir := filepath.Dir(goldenPath)
	err := os.MkdirAll(dir, 0755)
	require.NoError(t, err, "failed to create golden file directory: %s", dir)

	// Write the golden file
	err = os.WriteFile(goldenPath, actual, 0644)
	require.NoError(t, err, "failed to write golden file: %s", goldenPath)

	t.Logf("Updated golden file: %s", goldenPath)
}

// compareWithGoldenFile compares actual output with the golden file
func (g *GoldenFile) compareWithGoldenFile(t *testing.T, actual []byte, goldenPath string) {
	t.Helper()

	// Read the golden file
	expected, err := os.ReadFile(goldenPath)
	if os.IsNotExist(err) {
		t.Fatalf("Golden file does not exist: %s\n"+
			"Run with -update flag or set UPDATE_GOLDEN=1 to create it:\n"+
			"  go test -update\n"+
			"  UPDATE_GOLDEN=1 go test", goldenPath)
	}
	require.NoError(t, err, "failed to read golden file: %s", goldenPath)

	// Compare the content
	if !assert.Equal(t, string(expected), string(actual)) {
		t.Logf("Golden file mismatch for: %s", goldenPath)
		t.Logf("To update the golden file, run:")
		t.Logf("  go test -update")
		t.Logf("  or set UPDATE_GOLDEN=1")
	}
}

// GetGoldenPath returns the full path to a golden file
func (g *GoldenFile) GetGoldenPath(filename string) string {
	return filepath.Join(g.baseDir, filename)
}

// Exists checks if a golden file exists
func (g *GoldenFile) Exists(filename string) bool {
	goldenPath := g.GetGoldenPath(filename)
	_, err := os.Stat(goldenPath)
	return !os.IsNotExist(err)
}

// Remove deletes a golden file (useful for cleanup in tests)
func (g *GoldenFile) Remove(t *testing.T, filename string) {
	t.Helper()
	goldenPath := g.GetGoldenPath(filename)
	err := os.Remove(goldenPath)
	if err != nil && !os.IsNotExist(err) {
		t.Logf("Warning: failed to remove golden file %s: %v", goldenPath, err)
	}
}

// SetupGoldenFileTest is a convenience function for setting up golden file tests
// It returns a GoldenFile instance configured for the test
func SetupGoldenFileTest(t *testing.T, testdataDir string) *GoldenFile {
	t.Helper()
	return NewGoldenFile(testdataDir, t.Name())
}

// Example usage:
/*
func TestSomething(t *testing.T) {
	golden := helpers.SetupGoldenFileTest(t, "testdata")

	// Your test logic here
	actual := generateSomeOutput()

	// Compare with golden file
	golden.AssertString(t, actual, "something.txt")
}

// To update golden files:
// go test -update
// or
// UPDATE_GOLDEN=1 go test
*/
