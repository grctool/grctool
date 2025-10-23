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

package templates

import (
	"strings"
	"testing"
	"text/template"

	"github.com/grctool/grctool/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateFunctions(t *testing.T) {
	cfg := &logger.Config{
		Level:  logger.InfoLevel,
		Format: "text",
		Output: "stdout",
	}
	log, err := logger.New(cfg)
	require.NoError(t, err)
	m := &Manager{logger: log}
	funcs := m.getTemplateFuncs()

	t.Run("truncate", func(t *testing.T) {
		truncateFn := funcs["truncate"].(func(int, string) string)
		assert.Equal(t, "Hello", truncateFn(10, "Hello"))
		assert.Equal(t, "Hello W...", truncateFn(7, "Hello World"))
		assert.Equal(t, "", truncateFn(5, ""))
	})

	t.Run("default", func(t *testing.T) {
		defaultFn := funcs["default"].(func(interface{}, interface{}) interface{})
		assert.Equal(t, "default", defaultFn("default", nil))
		assert.Equal(t, "default", defaultFn("default", ""))
		assert.Equal(t, "value", defaultFn("default", "value"))
		assert.Equal(t, 42, defaultFn("default", 42))
	})

	t.Run("formatList", func(t *testing.T) {
		formatListFn := funcs["formatList"].(func([]string) string)
		assert.Equal(t, "", formatListFn([]string{}))
		assert.Equal(t, "• Item 1", formatListFn([]string{"Item 1"}))

		result := formatListFn([]string{"Item 1", "Item 2", "Item 3"})
		expected := "• Item 1\n• Item 2\n• Item 3"
		assert.Equal(t, expected, result)
	})

	t.Run("first and last", func(t *testing.T) {
		firstFn := funcs["first"].(func([]string) string)
		lastFn := funcs["last"].(func([]string) string)

		assert.Equal(t, "", firstFn([]string{}))
		assert.Equal(t, "", lastFn([]string{}))

		items := []string{"first", "middle", "last"}
		assert.Equal(t, "first", firstFn(items))
		assert.Equal(t, "last", lastFn(items))
	})

	t.Run("isNil and notNil", func(t *testing.T) {
		isNilFn := funcs["isNil"].(func(interface{}) bool)
		notNilFn := funcs["notNil"].(func(interface{}) bool)

		assert.True(t, isNilFn(nil))
		assert.False(t, isNilFn("value"))
		assert.False(t, isNilFn(0))

		assert.False(t, notNilFn(nil))
		assert.True(t, notNilFn("value"))
		assert.True(t, notNilFn(0))
	})
}

func TestManagerCreation(t *testing.T) {
	cfg := &logger.Config{
		Level:  logger.InfoLevel,
		Format: "text",
		Output: "stdout",
	}
	log, err := logger.New(cfg)
	require.NoError(t, err)

	// Since we don't have any templates yet, this will fail
	// We'll update this test after we create the template files
	m, err := NewManager(log)

	// For now, we expect this to succeed even with no templates
	// The embed.FS will just be empty
	require.NoError(t, err)
	assert.NotNil(t, m)
	assert.NotNil(t, m.templates)
	assert.NotNil(t, m.logger)
}

func TestExecuteTemplate(t *testing.T) {
	cfg := &logger.Config{
		Level:  logger.InfoLevel,
		Format: "text",
		Output: "stdout",
	}
	log, err := logger.New(cfg)
	require.NoError(t, err)
	m := &Manager{
		templates: make(map[string]*template.Template),
		logger:    log,
	}

	// Create a simple test template
	tmplContent := `Hello {{.Name}}! You have {{len .Items}} items.`
	tmpl, err := template.New("test").Funcs(m.getTemplateFuncs()).Parse(tmplContent)
	require.NoError(t, err)

	m.templates["test"] = tmpl

	// Test execution
	data := struct {
		Name  string
		Items []string
	}{
		Name:  "World",
		Items: []string{"item1", "item2", "item3"},
	}

	result, err := m.Execute("test", data)
	require.NoError(t, err)
	assert.Equal(t, "Hello World! You have 3 items.", result)

	// Test non-existent template
	_, err = m.Execute("nonexistent", data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template not found")
}

func TestComplexTemplate(t *testing.T) {
	cfg := &logger.Config{
		Level:  logger.InfoLevel,
		Format: "text",
		Output: "stdout",
	}
	log, err := logger.New(cfg)
	require.NoError(t, err)
	m := &Manager{
		templates: make(map[string]*template.Template),
		logger:    log,
	}

	// Create a more complex template using custom functions
	tmplContent := `
{{- if .Title}}
# {{.Title}}
{{- end}}

{{- if .Description}}
{{.Description | truncate 50}}
{{- end}}

{{- if .Items}}
## Items:
{{formatList .Items}}
{{- end}}

Default Value: {{default "No value provided" .Value}}
First Item: {{first .Items}}
`

	tmpl, err := template.New("complex").Funcs(m.getTemplateFuncs()).Parse(tmplContent)
	require.NoError(t, err)

	m.templates["complex"] = tmpl

	// Test with full data
	data := struct {
		Title       string
		Description string
		Items       []string
		Value       string
	}{
		Title:       "Test Document",
		Description: "This is a very long description that should be truncated to fifty characters",
		Items:       []string{"First item", "Second item", "Third item"},
		Value:       "Provided value",
	}

	result, err := m.Execute("complex", data)
	require.NoError(t, err)

	assert.Contains(t, result, "# Test Document")
	assert.Contains(t, result, "This is a very long description that should be tru...")
	assert.Contains(t, result, "• First item")
	assert.Contains(t, result, "• Second item")
	assert.Contains(t, result, "• Third item")
	assert.Contains(t, result, "Default Value: Provided value")
	assert.Contains(t, result, "First Item: First item")

	// Test with partial data
	partialData := struct {
		Title       string
		Description string
		Items       []string
		Value       string
	}{
		Title: "Partial Document",
		Items: []string{},
	}

	result, err = m.Execute("complex", partialData)
	require.NoError(t, err)

	assert.Contains(t, result, "# Partial Document")
	assert.NotContains(t, result, "## Items:")
	assert.Contains(t, result, "Default Value: No value provided")
	assert.Contains(t, result, "First Item:")

	// Verify the output is clean (no empty sections)
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		// Check that we don't have orphaned headers
		if strings.TrimSpace(line) == "## Items:" {
			t.Error("Found orphaned Items header")
		}
	}
}
