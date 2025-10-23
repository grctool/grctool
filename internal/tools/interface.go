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
	"context"

	"github.com/grctool/grctool/internal/models"
)

// Tool defines the interface for evidence collection tools
type Tool interface {
	// GetClaudeToolDefinition returns the tool definition for Claude
	GetClaudeToolDefinition() models.ClaudeTool

	// Execute runs the tool with the given parameters
	// Returns: result string, evidence source (if applicable), error
	Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error)

	// Name returns the tool name
	Name() string

	// Description returns the tool description
	Description() string
}
