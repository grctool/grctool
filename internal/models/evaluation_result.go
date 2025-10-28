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

import "time"

// EvaluationResult represents a comprehensive evaluation of evidence against task requirements
type EvaluationResult struct {
	// Identification
	TaskRef  string `json:"task_ref" yaml:"task_ref"`
	TaskID   int    `json:"task_id" yaml:"task_id"`
	Window   string `json:"window" yaml:"window"`
	Subfolder string `json:"subfolder,omitempty" yaml:"subfolder,omitempty"` // .submitted or archive

	// Overall evaluation
	OverallScore  float64          `json:"overall_score" yaml:"overall_score"`     // 0-100
	OverallStatus EvaluationStatus `json:"overall_status" yaml:"overall_status"`   // pass, warning, fail
	PassThreshold float64          `json:"pass_threshold" yaml:"pass_threshold"`   // Score needed to pass (default 70)

	// Dimension scores
	Completeness        DimensionScore `json:"completeness" yaml:"completeness"`
	RequirementsMatch   DimensionScore `json:"requirements_match" yaml:"requirements_match"`
	QualityScore        DimensionScore `json:"quality_score" yaml:"quality_score"`
	ControlAlignment    DimensionScore `json:"control_alignment" yaml:"control_alignment"`

	// Issues and recommendations
	Issues           []EvaluationIssue  `json:"issues" yaml:"issues"`
	Recommendations  []string           `json:"recommendations" yaml:"recommendations"`
	MissingRequirements []string        `json:"missing_requirements,omitempty" yaml:"missing_requirements,omitempty"`

	// Metadata
	EvaluatedAt time.Time `json:"evaluated_at" yaml:"evaluated_at"`
	EvaluatedBy string    `json:"evaluated_by" yaml:"evaluated_by"` // system, user, etc.
	FileCount   int       `json:"file_count" yaml:"file_count"`
	TotalBytes  int64     `json:"total_bytes" yaml:"total_bytes"`
}

// DimensionScore represents a score for one evaluation dimension
type DimensionScore struct {
	Score       float64 `json:"score" yaml:"score"`             // 0-100
	MaxScore    float64 `json:"max_score" yaml:"max_score"`     // Usually 100
	Weight      float64 `json:"weight" yaml:"weight"`           // Contribution to overall score
	Status      string  `json:"status" yaml:"status"`           // pass, warning, fail
	Description string  `json:"description" yaml:"description"` // What this dimension measures
	Details     string  `json:"details,omitempty" yaml:"details,omitempty"` // Specific findings
}

// EvaluationIssue represents a specific issue found during evaluation
type EvaluationIssue struct {
	Severity    IssueSeverity `json:"severity" yaml:"severity"`       // critical, high, medium, low, info
	Category    string        `json:"category" yaml:"category"`       // completeness, quality, requirements, control_alignment
	Message     string        `json:"message" yaml:"message"`         // Human-readable description
	Location    string        `json:"location,omitempty" yaml:"location,omitempty"` // File or section where issue was found
	Suggestion  string        `json:"suggestion,omitempty" yaml:"suggestion,omitempty"` // How to fix
}

// EvaluationStatus represents the overall status of an evaluation
type EvaluationStatus string

const (
	EvaluationPass    EvaluationStatus = "pass"    // Evidence meets requirements
	EvaluationWarning EvaluationStatus = "warning" // Evidence acceptable but has issues
	EvaluationFail    EvaluationStatus = "fail"    // Evidence does not meet requirements
)

// IssueSeverity represents the severity of an evaluation issue
type IssueSeverity string

const (
	IssueCritical IssueSeverity = "critical" // Must be fixed
	IssueHigh     IssueSeverity = "high"     // Should be fixed
	IssueMedium   IssueSeverity = "medium"   // Consider fixing
	IssueLow      IssueSeverity = "low"      // Minor issue
	IssueInfo     IssueSeverity = "info"     // Informational only
)

// CalculateOverallScore calculates the weighted overall score from dimension scores
func (er *EvaluationResult) CalculateOverallScore() {
	totalWeight := er.Completeness.Weight + er.RequirementsMatch.Weight +
		er.QualityScore.Weight + er.ControlAlignment.Weight

	if totalWeight == 0 {
		er.OverallScore = 0
		return
	}

	weightedSum := (er.Completeness.Score * er.Completeness.Weight) +
		(er.RequirementsMatch.Score * er.RequirementsMatch.Weight) +
		(er.QualityScore.Score * er.QualityScore.Weight) +
		(er.ControlAlignment.Score * er.ControlAlignment.Weight)

	er.OverallScore = weightedSum / totalWeight
}

// DetermineStatus determines the overall status based on score and issues
func (er *EvaluationResult) DetermineStatus() {
	// Check for critical issues
	for _, issue := range er.Issues {
		if issue.Severity == IssueCritical {
			er.OverallStatus = EvaluationFail
			return
		}
	}

	// Determine based on score
	if er.OverallScore >= er.PassThreshold {
		er.OverallStatus = EvaluationPass
	} else if er.OverallScore >= (er.PassThreshold * 0.6) {
		// Warning if score is between 60% and 100% of pass threshold
		er.OverallStatus = EvaluationWarning
	} else {
		er.OverallStatus = EvaluationFail
	}

	// Downgrade to warning if there are high severity issues
	if er.OverallStatus == EvaluationPass {
		for _, issue := range er.Issues {
			if issue.Severity == IssueHigh {
				er.OverallStatus = EvaluationWarning
				break
			}
		}
	}
}

// AddIssue adds an issue to the evaluation result
func (er *EvaluationResult) AddIssue(severity IssueSeverity, category, message, location, suggestion string) {
	er.Issues = append(er.Issues, EvaluationIssue{
		Severity:   severity,
		Category:   category,
		Message:    message,
		Location:   location,
		Suggestion: suggestion,
	})
}

// AddRecommendation adds a recommendation to the evaluation result
func (er *EvaluationResult) AddRecommendation(recommendation string) {
	er.Recommendations = append(er.Recommendations, recommendation)
}

// GetCriticalIssueCount returns the number of critical issues
func (er *EvaluationResult) GetCriticalIssueCount() int {
	count := 0
	for _, issue := range er.Issues {
		if issue.Severity == IssueCritical {
			count++
		}
	}
	return count
}

// GetHighIssueCount returns the number of high severity issues
func (er *EvaluationResult) GetHighIssueCount() int {
	count := 0
	for _, issue := range er.Issues {
		if issue.Severity == IssueHigh {
			count++
		}
	}
	return count
}

// NewEvaluationResult creates a new evaluation result with default values
func NewEvaluationResult(taskRef string, taskID int, window string, subfolder string) *EvaluationResult {
	return &EvaluationResult{
		TaskRef:       taskRef,
		TaskID:        taskID,
		Window:        window,
		Subfolder:     subfolder,
		OverallScore:  0,
		OverallStatus: EvaluationFail,
		PassThreshold: 70.0, // Default 70% to pass
		Completeness: DimensionScore{
			MaxScore:    100,
			Weight:      0.30, // 30% of overall score
			Description: "Evidence completeness and required files present",
		},
		RequirementsMatch: DimensionScore{
			MaxScore:    100,
			Weight:      0.30, // 30% of overall score
			Description: "Evidence matches task requirements and guidance",
		},
		QualityScore: DimensionScore{
			MaxScore:    100,
			Weight:      0.20, // 20% of overall score
			Description: "Evidence quality and presentation",
		},
		ControlAlignment: DimensionScore{
			MaxScore:    100,
			Weight:      0.20, // 20% of overall score
			Description: "Evidence addresses related controls appropriately",
		},
		Issues:              []EvaluationIssue{},
		Recommendations:     []string{},
		MissingRequirements: []string{},
		EvaluatedAt:         time.Now(),
		EvaluatedBy:         "grctool-evaluator",
	}
}
