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

package tugboat

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildWebURL(t *testing.T) {
	tests := []struct {
		name       string
		apiBaseURL string
		want       string
	}{
		{
			name:       "converts api-my.tugboatlogic.com to my.tugboatlogic.com",
			apiBaseURL: "https://api-my.tugboatlogic.com",
			want:       "https://my.tugboatlogic.com",
		},
		{
			name:       "removes /api suffix",
			apiBaseURL: "https://api-my.tugboatlogic.com/api",
			want:       "https://my.tugboatlogic.com",
		},
		{
			name:       "removes trailing slash",
			apiBaseURL: "https://api-my.tugboatlogic.com/",
			want:       "https://my.tugboatlogic.com",
		},
		{
			name:       "handles app.tugboatlogic.com unchanged",
			apiBaseURL: "https://app.tugboatlogic.com",
			want:       "https://app.tugboatlogic.com",
		},
		{
			name:       "handles app.tugboatlogic.com with /api",
			apiBaseURL: "https://app.tugboatlogic.com/api",
			want:       "https://app.tugboatlogic.com",
		},
		{
			name:       "handles complex path with /api",
			apiBaseURL: "https://api-my.tugboatlogic.com/api/",
			want:       "https://my.tugboatlogic.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildWebURL(tt.apiBaseURL)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildEvidenceTaskURL(t *testing.T) {
	tests := []struct {
		name       string
		apiBaseURL string
		orgID      int
		taskID     int
		want       string
	}{
		{
			name:       "builds evidence task URL with api-my domain",
			apiBaseURL: "https://api-my.tugboatlogic.com",
			orgID:      13888,
			taskID:     327992,
			want:       "https://my.tugboatlogic.com/org/13888/evidence/tasks/327992",
		},
		{
			name:       "builds evidence task URL with app domain",
			apiBaseURL: "https://app.tugboatlogic.com",
			orgID:      13888,
			taskID:     12345,
			want:       "https://app.tugboatlogic.com/org/13888/evidence/tasks/12345",
		},
		{
			name:       "handles /api suffix",
			apiBaseURL: "https://api-my.tugboatlogic.com/api",
			orgID:      13888,
			taskID:     100,
			want:       "https://my.tugboatlogic.com/org/13888/evidence/tasks/100",
		},
		{
			name:       "handles single digit task ID",
			apiBaseURL: "https://api-my.tugboatlogic.com",
			orgID:      999,
			taskID:     1,
			want:       "https://my.tugboatlogic.com/org/999/evidence/tasks/1",
		},
		{
			name:       "handles different org ID",
			apiBaseURL: "https://api-my.tugboatlogic.com",
			orgID:      54321,
			taskID:     328027,
			want:       "https://my.tugboatlogic.com/org/54321/evidence/tasks/328027",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildEvidenceTaskURL(tt.apiBaseURL, tt.orgID, tt.taskID)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildControlURL(t *testing.T) {
	tests := []struct {
		name       string
		apiBaseURL string
		controlID  int
		want       string
	}{
		{
			name:       "builds control URL",
			apiBaseURL: "https://api-my.tugboatlogic.com",
			controlID:  778805,
			want:       "https://my.tugboatlogic.com/control/778805",
		},
		{
			name:       "handles /api suffix",
			apiBaseURL: "https://api-my.tugboatlogic.com/api",
			controlID:  778771,
			want:       "https://my.tugboatlogic.com/control/778771",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildControlURL(tt.apiBaseURL, tt.controlID)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildPolicyURL(t *testing.T) {
	tests := []struct {
		name       string
		apiBaseURL string
		policyID   int
		want       string
	}{
		{
			name:       "builds policy URL",
			apiBaseURL: "https://api-my.tugboatlogic.com",
			policyID:   12345,
			want:       "https://my.tugboatlogic.com/policy/12345",
		},
		{
			name:       "handles /api suffix",
			apiBaseURL: "https://api-my.tugboatlogic.com/api",
			policyID:   67890,
			want:       "https://my.tugboatlogic.com/policy/67890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildPolicyURL(tt.apiBaseURL, tt.policyID)
			assert.Equal(t, tt.want, got)
		})
	}
}
