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
	"testing"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/logger"
	"github.com/stretchr/testify/assert"
)

func TestEvidenceTaskListTool_enrichTasksWithURLs(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		orgID    string
		tasks    []domain.EvidenceTask
		wantURLs []string
	}{
		{
			name:    "adds URLs to tasks with valid IDs and org ID",
			baseURL: "https://api-my.tugboatlogic.com",
			orgID:   "13888",
			tasks: []domain.EvidenceTask{
				{ID: 327992, OrgID: 13888, ReferenceID: "ET-001", Name: "Test Task 1"},
				{ID: 12345, OrgID: 13888, ReferenceID: "ET-002", Name: "Test Task 2"},
			},
			wantURLs: []string{
				"https://my.tugboatlogic.com/org/13888/evidence/tasks/327992",
				"https://my.tugboatlogic.com/org/13888/evidence/tasks/12345",
			},
		},
		{
			name:     "handles empty task list",
			baseURL:  "https://api-my.tugboatlogic.com",
			orgID:    "13888",
			tasks:    []domain.EvidenceTask{},
			wantURLs: []string{},
		},
		{
			name:    "skips tasks with zero ID",
			baseURL: "https://api-my.tugboatlogic.com",
			orgID:   "13888",
			tasks: []domain.EvidenceTask{
				{ID: 0, OrgID: 13888, ReferenceID: "ET-001", Name: "Test Task"},
			},
			wantURLs: []string{""},
		},
		{
			name:    "handles app.tugboatlogic.com base URL",
			baseURL: "https://app.tugboatlogic.com",
			orgID:   "13888",
			tasks: []domain.EvidenceTask{
				{ID: 100, OrgID: 13888, ReferenceID: "ET-001", Name: "Test Task"},
			},
			wantURLs: []string{
				"https://app.tugboatlogic.com/org/13888/evidence/tasks/100",
			},
		},
		{
			name:     "handles empty base URL gracefully",
			baseURL:  "",
			orgID:    "13888",
			tasks:    []domain.EvidenceTask{{ID: 100, OrgID: 13888, ReferenceID: "ET-001"}},
			wantURLs: []string{""},
		},
		{
			name:    "uses config org ID when task org ID is missing",
			baseURL: "https://api-my.tugboatlogic.com",
			orgID:   "54321",
			tasks: []domain.EvidenceTask{
				{ID: 100, OrgID: 0, ReferenceID: "ET-001", Name: "Test Task"},
			},
			wantURLs: []string{
				"https://my.tugboatlogic.com/org/54321/evidence/tasks/100",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create tool with test config
			cfg := &config.Config{
				Tugboat: config.TugboatConfig{
					BaseURL: tt.baseURL,
					OrgID:   tt.orgID,
				},
			}
			logCfg := logger.Config{
				Level:  logger.ErrorLevel,
				Format: "text",
				Output: "stdout",
			}
			log, err := logger.New(&logCfg)
			assert.NoError(t, err)

			tool := &EvidenceTaskListTool{
				config: cfg,
				logger: log,
			}

			// Enrich tasks with URLs
			tool.enrichTasksWithURLs(tt.tasks)

			// Verify URLs were added correctly
			assert.Equal(t, len(tt.wantURLs), len(tt.tasks))
			for i, task := range tt.tasks {
				assert.Equal(t, tt.wantURLs[i], task.TugboatURL,
					"Task %d URL mismatch", i)
			}
		})
	}
}
