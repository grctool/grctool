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

package models

import (
	"time"
)

// Procedure represents a security procedure from Tugboat Logic
type Procedure struct {
	ID          string    `json:"id"`
	PolicyID    string    `json:"policy_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Steps       []Step    `json:"steps"`
	Frequency   string    `json:"frequency"` // daily, weekly, monthly, quarterly, annually
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Step represents a step within a procedure
type Step struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Order       int    `json:"order"`
	Required    bool   `json:"required"`
	Status      string `json:"status"` // pending, in_progress, completed, skipped
}

// ProcedureSummary represents a summary view of procedures
type ProcedureSummary struct {
	Total       int            `json:"total"`
	ByFrequency map[string]int `json:"by_frequency"`
	ByStatus    map[string]int `json:"by_status"`
	LastSync    time.Time      `json:"last_sync"`
}
