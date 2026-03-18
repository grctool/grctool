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

package testhelpers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/grctool/grctool/internal/domain"
)

// testdataDir returns the absolute path to the testdata directory adjacent to
// this source file. It uses runtime.Caller so it works regardless of the
// working directory the test binary runs in.
func testdataDir() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "testdata")
}

// LoadFixture reads a file from the testdata/ directory and returns the raw bytes.
func LoadFixture(path string) ([]byte, error) {
	fullPath := filepath.Join(testdataDir(), path)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("loading fixture %s: %w", path, err)
	}
	return data, nil
}

// LoadPolicyFixture loads and unmarshals a Policy JSON fixture from
// testdata/policies/<name>.json.
func LoadPolicyFixture(name string) (*domain.Policy, error) {
	data, err := LoadFixture(filepath.Join("policies", name+".json"))
	if err != nil {
		return nil, err
	}
	var p domain.Policy
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("unmarshaling policy fixture %s: %w", name, err)
	}
	return &p, nil
}

// LoadControlFixture loads and unmarshals a Control JSON fixture from
// testdata/controls/<name>.json.
func LoadControlFixture(name string) (*domain.Control, error) {
	data, err := LoadFixture(filepath.Join("controls", name+".json"))
	if err != nil {
		return nil, err
	}
	var c domain.Control
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("unmarshaling control fixture %s: %w", name, err)
	}
	return &c, nil
}

// LoadEvidenceTaskFixture loads and unmarshals an EvidenceTask JSON fixture
// from testdata/evidence_tasks/<name>.json.
func LoadEvidenceTaskFixture(name string) (*domain.EvidenceTask, error) {
	data, err := LoadFixture(filepath.Join("evidence_tasks", name+".json"))
	if err != nil {
		return nil, err
	}
	var t domain.EvidenceTask
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("unmarshaling evidence task fixture %s: %w", name, err)
	}
	return &t, nil
}

// ---------------------------------------------------------------------------
// In-memory sample builders (no file I/O required)
// ---------------------------------------------------------------------------

// SamplePolicy returns a valid test Policy with realistic compliance data.
func SamplePolicy() *domain.Policy {
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	return &domain.Policy{
		ID:          "12345",
		ReferenceID: "POL-0001",
		Name:        "Access Control Policy",
		Description: "Defines the organization's approach to managing logical and physical access to information assets.",
		Framework:   "SOC2",
		Status:      "active",
		Category:    "Security",
		Content:     "All access to production systems must be authorized, authenticated, and logged.",
		CreatedAt:   now,
		UpdatedAt:   now,
		VersionNum:  1,
		ControlCount: 5,
	}
}

// SampleControl returns a valid test Control with realistic compliance data.
func SampleControl() *domain.Control {
	return &domain.Control{
		ID:          "1001",
		ReferenceID: "CC-06.1",
		Name:        "Logical Access Security",
		Description: "The entity implements logical access security software, infrastructure, and architectures over protected information assets.",
		Category:    "Common Criteria",
		Framework:   "SOC2",
		Status:      "implemented",
		Risk:        "medium",
		RiskLevel:   "medium",
	}
}

// SampleEvidenceTask returns a valid test EvidenceTask with realistic compliance data.
func SampleEvidenceTask() *domain.EvidenceTask {
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	return &domain.EvidenceTask{
		ID:                 "327992",
		ReferenceID:        "ET-0047",
		Name:               "GitHub Repository Access Controls",
		Description:        "Provide evidence of repository access controls including team permissions, branch protections, and code review requirements.",
		Guidance:           "Extract team membership, repository permissions matrix, and branch protection rules from GitHub.",
		CollectionInterval: "quarter",
		Priority:           "high",
		Framework:          "SOC2",
		Status:             "pending",
		Category:           "Infrastructure",
		ComplexityLevel:    "Moderate",
		CollectionType:     "Automated",
		CreatedAt:          now,
		UpdatedAt:          now,
		Controls:           []string{"CC-06.1", "CC-06.3"},
	}
}

// SampleEvidenceRecord returns a valid test EvidenceRecord.
func SampleEvidenceRecord() *domain.EvidenceRecord {
	return &domain.EvidenceRecord{
		ID:          "rec-001",
		TaskID:      "327992",
		Title:       "GitHub Repository Access Controls - Q1 2025",
		Description: "Automated evidence collection of GitHub repository access controls.",
		Content:     "Repository: org/main-app\nTeams with access: engineering (write), security (admin), devops (maintain)",
		Format:      "markdown",
		Source:      "github",
		CollectedAt: time.Date(2025, 1, 20, 14, 30, 0, 0, time.UTC),
		CollectedBy: "grctool",
		Metadata: map[string]interface{}{
			"tool":       "github-permissions",
			"repository": "org/main-app",
		},
	}
}
