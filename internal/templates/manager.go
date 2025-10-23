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
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"strings"
	"sync"
	"text/template"

	"github.com/grctool/grctool/internal/logger"
)

//go:embed prompts/*.tmpl
var promptsFS embed.FS

// Manager handles template loading and execution
type Manager struct {
	templates map[string]*template.Template
	mu        sync.RWMutex
	logger    logger.Logger
}

// Singleton pattern for template manager
var (
	singleton     *Manager
	singletonOnce sync.Once
)

// GetSingleton returns the singleton template manager instance
func GetSingleton(log logger.Logger) (*Manager, error) {
	var err error
	singletonOnce.Do(func() {
		singleton = &Manager{
			templates: make(map[string]*template.Template),
			logger:    log,
		}

		// Load all templates on initialization
		if loadErr := singleton.loadTemplates(); loadErr != nil {
			err = fmt.Errorf("failed to load templates: %w", loadErr)
			singleton = nil
		}
	})

	if singleton == nil && err != nil {
		return nil, err
	}

	return singleton, nil
}

// NewManager creates a new template manager
func NewManager(log logger.Logger) (*Manager, error) {
	m := &Manager{
		templates: make(map[string]*template.Template),
		logger:    log,
	}

	// Load all templates on initialization
	if err := m.loadTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	return m, nil
}

// loadTemplates loads all templates from the embedded filesystem
func (m *Manager) loadTemplates() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Walk through the prompts directory
	err := fs.WalkDir(promptsFS, "prompts", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Only process .tmpl files
		if !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		// Read template content
		content, err := promptsFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", path, err)
		}

		// Parse template with custom functions
		tmpl, err := template.New(path).Funcs(m.getTemplateFuncs()).Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", path, err)
		}

		// Store template with a simplified name (without directory prefix)
		name := strings.TrimPrefix(path, "prompts/")
		name = strings.TrimSuffix(name, ".tmpl")
		m.templates[name] = tmpl

		m.logger.Debug("Loaded template",
			logger.Field{Key: "name", Value: name},
			logger.Field{Key: "path", Value: path},
		)

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk templates directory: %w", err)
	}

	m.logger.Info("Templates loaded successfully",
		logger.Field{Key: "count", Value: len(m.templates)},
	)

	return nil
}

// Execute executes a named template with the given data
func (m *Manager) Execute(name string, data interface{}) (string, error) {
	m.mu.RLock()
	tmpl, exists := m.templates[name]
	m.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("template not found: %s", name)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}

	return buf.String(), nil
}

// getTemplateFuncs returns custom template functions
func (m *Manager) getTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		// String manipulation functions
		"truncate": func(max int, s string) string {
			if len(s) <= max {
				return s
			}
			return s[:max] + "..."
		},
		"join":      strings.Join,
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"toLower":   strings.ToLower,
		"toUpper":   strings.ToUpper,

		// Conditional functions
		"default": func(def, val interface{}) interface{} {
			if val == nil || val == "" {
				return def
			}
			return val
		},

		// List functions
		"first": func(list []string) string {
			if len(list) > 0 {
				return list[0]
			}
			return ""
		},
		"last": func(list []string) string {
			if len(list) > 0 {
				return list[len(list)-1]
			}
			return ""
		},

		// Formatting functions
		"formatList": func(items []string) string {
			if len(items) == 0 {
				return ""
			}
			var result strings.Builder
			for _, item := range items {
				result.WriteString("• ")
				result.WriteString(item)
				result.WriteString("\n")
			}
			return strings.TrimSpace(result.String())
		},

		// Type checking
		"isNil": func(v interface{}) bool {
			return v == nil
		},
		"notNil": func(v interface{}) bool {
			return v != nil
		},

		// Math functions
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"mul": func(a, b int) int {
			return a * b
		},
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},

		// Slice functions
		"slice": func() []string {
			return []string{}
		},
		"append": func(slice []string, elem string) []string {
			return append(slice, elem)
		},

		// Pointer dereferencing
		"deref": func(ptr *string) string {
			if ptr == nil {
				return ""
			}
			return *ptr
		},
		"hasValue": func(ptr *string) bool {
			return ptr != nil && *ptr != ""
		},
	}
}

// ReloadTemplates reloads all templates (useful for development)
func (m *Manager) ReloadTemplates() error {
	m.logger.Info("Reloading templates")
	return m.loadTemplates()
}
