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

package domain

import (
	"encoding/json"
	"time"
)

// Person represents a user in the system
type Person struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Email      string     `json:"email"`
	Role       string     `json:"role,omitempty"`
	AssignedAt *time.Time `json:"assigned_at,omitempty"`
}

// Tag represents a tag that can be applied to various entities
type Tag struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

// UnmarshalJSON implements custom unmarshaling for Tag to support both string and object formats
func (t *Tag) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as a string first (for local JSON files)
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		t.Name = str
		t.ID = ""
		t.Color = ""
		return nil
	}

	// If string unmarshaling fails, try as structured object (for API data)
	type tagAlias Tag // Prevent infinite recursion
	var alias tagAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	*t = Tag(alias)
	return nil
}

// FrameworkCode represents a compliance framework code
type FrameworkCode struct {
	ID        string `json:"id"`
	Code      string `json:"code"`
	Framework string `json:"framework"`
	Name      string `json:"name"`
}

// OrgScope represents the organizational scope for compliance items
type OrgScope struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

// AuditProject represents an audit project
type AuditProject struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date"`
	Description string    `json:"description"`
}

// JiraIssue represents a Jira issue
type JiraIssue struct {
	ID        string    `json:"id"`
	Key       string    `json:"key"`
	Summary   string    `json:"summary"`
	Status    string    `json:"status"`
	Priority  string    `json:"priority"`
	IssueType string    `json:"issue_type"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Assignee  string    `json:"assignee"`
	Reporter  string    `json:"reporter"`
}

// Relationship represents relationships between domain entities
type Relationship struct {
	SourceType string `json:"source_type"` // policy, control, evidence_task
	SourceID   string `json:"source_id"`
	TargetType string `json:"target_type"` // policy, control, evidence_task
	TargetID   string `json:"target_id"`
	Type       string `json:"type"` // implements, verifies, requires, etc.
}
