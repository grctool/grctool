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
	"fmt"
	"strings"
)

// BuildWebURL converts a Tugboat API base URL to the corresponding web UI URL.
// This handles the conversion from API endpoints (api-my.tugboatlogic.com) to
// web UI endpoints (my.tugboatlogic.com).
//
// Examples:
//   - "https://api-my.tugboatlogic.com" -> "https://my.tugboatlogic.com"
//   - "https://api-my.tugboatlogic.com/api" -> "https://my.tugboatlogic.com"
//   - "https://app.tugboatlogic.com" -> "https://app.tugboatlogic.com"
func BuildWebURL(apiBaseURL string) string {
	webURL := strings.Replace(apiBaseURL, "api-my.tugboatlogic.com", "my.tugboatlogic.com", 1)
	// Remove trailing slashes and /api suffix (in any order)
	webURL = strings.TrimSuffix(webURL, "/")
	webURL = strings.TrimSuffix(webURL, "/api")
	webURL = strings.TrimSuffix(webURL, "/")
	return webURL
}

// BuildEvidenceTaskURL creates a web URL for a specific evidence task.
// The orgID and taskID should be the Tugboat organization and evidence task IDs (integers).
//
// Example:
//   - BuildEvidenceTaskURL("https://api-my.tugboatlogic.com", 13888, 327992)
//     returns "https://my.tugboatlogic.com/org/13888/evidence/tasks/327992"
func BuildEvidenceTaskURL(apiBaseURL string, orgID int, taskID int) string {
	webURL := BuildWebURL(apiBaseURL)
	return fmt.Sprintf("%s/org/%d/evidence/tasks/%d", webURL, orgID, taskID)
}

// BuildControlURL creates a web URL for a specific control.
// The controlID should be the Tugboat control ID (integer).
//
// Example:
//   - BuildControlURL("https://api-my.tugboatlogic.com", 778805)
//     returns "https://my.tugboatlogic.com/control/778805"
func BuildControlURL(apiBaseURL string, controlID int) string {
	webURL := BuildWebURL(apiBaseURL)
	return fmt.Sprintf("%s/control/%d", webURL, controlID)
}

// BuildPolicyURL creates a web URL for a specific policy.
// The policyID should be the Tugboat policy ID (integer).
//
// Example:
//   - BuildPolicyURL("https://api-my.tugboatlogic.com", 12345)
//     returns "https://my.tugboatlogic.com/policy/12345"
func BuildPolicyURL(apiBaseURL string, policyID int) string {
	webURL := BuildWebURL(apiBaseURL)
	return fmt.Sprintf("%s/policy/%d", webURL, policyID)
}
