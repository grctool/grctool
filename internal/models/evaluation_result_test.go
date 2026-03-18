package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEvaluationResult(t *testing.T) {
	t.Parallel()
	er := NewEvaluationResult("ET-0001", "1", "2025-Q4", "")

	assert.Equal(t, "ET-0001", er.TaskRef)
	assert.Equal(t, "1", er.TaskID)
	assert.Equal(t, "2025-Q4", er.Window)
	assert.Equal(t, "", er.Subfolder)
	assert.Equal(t, float64(0), er.OverallScore)
	assert.Equal(t, EvaluationFail, er.OverallStatus)
	assert.Equal(t, 70.0, er.PassThreshold)
	assert.NotNil(t, er.Issues)
	assert.Empty(t, er.Issues)
	assert.NotNil(t, er.Recommendations)
	assert.Empty(t, er.Recommendations)
	assert.NotZero(t, er.EvaluatedAt)
	assert.Equal(t, "grctool-evaluator", er.EvaluatedBy)

	// Check dimension weights sum to 1.0
	totalWeight := er.Completeness.Weight + er.RequirementsMatch.Weight +
		er.QualityScore.Weight + er.ControlAlignment.Weight
	assert.InDelta(t, 1.0, totalWeight, 0.001)
}

func TestNewEvaluationResult_WithSubfolder(t *testing.T) {
	t.Parallel()
	er := NewEvaluationResult("ET-0002", "2", "2025-Q4", ".submitted")
	assert.Equal(t, ".submitted", er.Subfolder)
}

func TestEvaluationResult_CalculateOverallScore(t *testing.T) {
	t.Parallel()

	t.Run("weighted average", func(t *testing.T) {
		t.Parallel()
		er := NewEvaluationResult("ET-0001", "1", "2025-Q4", "")
		er.Completeness.Score = 80
		er.RequirementsMatch.Score = 90
		er.QualityScore.Score = 70
		er.ControlAlignment.Score = 60

		er.CalculateOverallScore()

		// weights: 0.30, 0.30, 0.20, 0.20
		// (80*0.30 + 90*0.30 + 70*0.20 + 60*0.20) / 1.0 = 24 + 27 + 14 + 12 = 77
		assert.InDelta(t, 77.0, er.OverallScore, 0.01)
	})

	t.Run("all zeros", func(t *testing.T) {
		t.Parallel()
		er := NewEvaluationResult("ET-0001", "1", "2025-Q4", "")
		er.CalculateOverallScore()
		assert.Equal(t, float64(0), er.OverallScore)
	})

	t.Run("all perfect scores", func(t *testing.T) {
		t.Parallel()
		er := NewEvaluationResult("ET-0001", "1", "2025-Q4", "")
		er.Completeness.Score = 100
		er.RequirementsMatch.Score = 100
		er.QualityScore.Score = 100
		er.ControlAlignment.Score = 100

		er.CalculateOverallScore()
		assert.InDelta(t, 100.0, er.OverallScore, 0.01)
	})

	t.Run("zero weights returns zero", func(t *testing.T) {
		t.Parallel()
		er := &EvaluationResult{
			Completeness:      DimensionScore{Score: 80, Weight: 0},
			RequirementsMatch: DimensionScore{Score: 90, Weight: 0},
			QualityScore:      DimensionScore{Score: 70, Weight: 0},
			ControlAlignment:  DimensionScore{Score: 60, Weight: 0},
		}
		er.CalculateOverallScore()
		assert.Equal(t, float64(0), er.OverallScore)
	})
}

func TestEvaluationResult_DetermineStatus(t *testing.T) {
	t.Parallel()

	t.Run("critical issue forces fail", func(t *testing.T) {
		t.Parallel()
		er := NewEvaluationResult("ET-0001", "1", "2025-Q4", "")
		er.OverallScore = 95 // Would pass normally
		er.AddIssue(IssueCritical, "completeness", "Missing required file", "", "Add file")
		er.DetermineStatus()
		assert.Equal(t, EvaluationFail, er.OverallStatus)
	})

	t.Run("score above threshold passes", func(t *testing.T) {
		t.Parallel()
		er := NewEvaluationResult("ET-0001", "1", "2025-Q4", "")
		er.OverallScore = 85
		er.DetermineStatus()
		assert.Equal(t, EvaluationPass, er.OverallStatus)
	})

	t.Run("score at threshold passes", func(t *testing.T) {
		t.Parallel()
		er := NewEvaluationResult("ET-0001", "1", "2025-Q4", "")
		er.OverallScore = 70
		er.DetermineStatus()
		assert.Equal(t, EvaluationPass, er.OverallStatus)
	})

	t.Run("score between 60pct and threshold is warning", func(t *testing.T) {
		t.Parallel()
		er := NewEvaluationResult("ET-0001", "1", "2025-Q4", "")
		// PassThreshold = 70, so warning range is 42..69.99
		er.OverallScore = 50
		er.DetermineStatus()
		assert.Equal(t, EvaluationWarning, er.OverallStatus)
	})

	t.Run("score below 60pct of threshold is fail", func(t *testing.T) {
		t.Parallel()
		er := NewEvaluationResult("ET-0001", "1", "2025-Q4", "")
		// PassThreshold = 70, 60% of 70 = 42
		er.OverallScore = 30
		er.DetermineStatus()
		assert.Equal(t, EvaluationFail, er.OverallStatus)
	})

	t.Run("high severity downgrades pass to warning", func(t *testing.T) {
		t.Parallel()
		er := NewEvaluationResult("ET-0001", "1", "2025-Q4", "")
		er.OverallScore = 85
		er.AddIssue(IssueHigh, "quality", "Poor formatting", "", "Reformat")
		er.DetermineStatus()
		assert.Equal(t, EvaluationWarning, er.OverallStatus)
	})

	t.Run("medium severity does not downgrade pass", func(t *testing.T) {
		t.Parallel()
		er := NewEvaluationResult("ET-0001", "1", "2025-Q4", "")
		er.OverallScore = 85
		er.AddIssue(IssueMedium, "quality", "Minor issue", "", "Fix later")
		er.DetermineStatus()
		assert.Equal(t, EvaluationPass, er.OverallStatus)
	})

	t.Run("critical checked before score", func(t *testing.T) {
		t.Parallel()
		er := NewEvaluationResult("ET-0001", "1", "2025-Q4", "")
		er.OverallScore = 100
		er.AddIssue(IssueCritical, "completeness", "Fatal error", "", "")
		er.DetermineStatus()
		assert.Equal(t, EvaluationFail, er.OverallStatus)
	})
}

func TestEvaluationResult_AddIssue(t *testing.T) {
	t.Parallel()
	er := NewEvaluationResult("ET-0001", "1", "2025-Q4", "")
	assert.Empty(t, er.Issues)

	er.AddIssue(IssueHigh, "completeness", "Missing file", "evidence.md", "Add file")
	require.Len(t, er.Issues, 1)
	assert.Equal(t, IssueHigh, er.Issues[0].Severity)
	assert.Equal(t, "completeness", er.Issues[0].Category)
	assert.Equal(t, "Missing file", er.Issues[0].Message)
	assert.Equal(t, "evidence.md", er.Issues[0].Location)
	assert.Equal(t, "Add file", er.Issues[0].Suggestion)

	er.AddIssue(IssueLow, "quality", "Minor", "", "")
	assert.Len(t, er.Issues, 2)
}

func TestEvaluationResult_AddRecommendation(t *testing.T) {
	t.Parallel()
	er := NewEvaluationResult("ET-0001", "1", "2025-Q4", "")
	assert.Empty(t, er.Recommendations)

	er.AddRecommendation("Add more detail")
	require.Len(t, er.Recommendations, 1)
	assert.Equal(t, "Add more detail", er.Recommendations[0])

	er.AddRecommendation("Include screenshots")
	assert.Len(t, er.Recommendations, 2)
}

func TestEvaluationResult_GetCriticalIssueCount(t *testing.T) {
	t.Parallel()
	er := NewEvaluationResult("ET-0001", "1", "2025-Q4", "")
	assert.Equal(t, 0, er.GetCriticalIssueCount())

	er.AddIssue(IssueCritical, "a", "msg1", "", "")
	er.AddIssue(IssueHigh, "b", "msg2", "", "")
	er.AddIssue(IssueCritical, "c", "msg3", "", "")
	er.AddIssue(IssueLow, "d", "msg4", "", "")
	assert.Equal(t, 2, er.GetCriticalIssueCount())
}

func TestEvaluationResult_GetHighIssueCount(t *testing.T) {
	t.Parallel()
	er := NewEvaluationResult("ET-0001", "1", "2025-Q4", "")
	assert.Equal(t, 0, er.GetHighIssueCount())

	er.AddIssue(IssueHigh, "a", "msg1", "", "")
	er.AddIssue(IssueCritical, "b", "msg2", "", "")
	er.AddIssue(IssueHigh, "c", "msg3", "", "")
	er.AddIssue(IssueHigh, "d", "msg4", "", "")
	assert.Equal(t, 3, er.GetHighIssueCount())
}

func TestEvaluationResult_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	er := NewEvaluationResult("ET-0001", "1", "2025-Q4", ".submitted")
	er.Completeness.Score = 80
	er.RequirementsMatch.Score = 90
	er.QualityScore.Score = 70
	er.ControlAlignment.Score = 60
	er.CalculateOverallScore()
	er.AddIssue(IssueHigh, "quality", "Needs improvement", "file.md", "Fix it")
	er.AddRecommendation("Use automation")
	er.MissingRequirements = []string{"MFA evidence"}
	er.DetermineStatus()

	data, err := json.Marshal(er)
	require.NoError(t, err)

	var decoded EvaluationResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, er.TaskRef, decoded.TaskRef)
	assert.Equal(t, er.OverallScore, decoded.OverallScore)
	assert.Equal(t, er.OverallStatus, decoded.OverallStatus)
	assert.Equal(t, er.PassThreshold, decoded.PassThreshold)
	assert.Equal(t, er.Completeness.Score, decoded.Completeness.Score)
	assert.Equal(t, er.Completeness.Weight, decoded.Completeness.Weight)
	assert.Len(t, decoded.Issues, 1)
	assert.Len(t, decoded.Recommendations, 1)
	assert.Len(t, decoded.MissingRequirements, 1)
}

func TestDimensionScore_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	ds := DimensionScore{
		Score:       85.5,
		MaxScore:    100,
		Weight:      0.30,
		Status:      "pass",
		Description: "Evidence completeness",
		Details:     "All files present",
	}

	data, err := json.Marshal(ds)
	require.NoError(t, err)

	var decoded DimensionScore
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, ds.Score, decoded.Score)
	assert.Equal(t, ds.MaxScore, decoded.MaxScore)
	assert.Equal(t, ds.Weight, decoded.Weight)
	assert.Equal(t, ds.Status, decoded.Status)
	assert.Equal(t, ds.Description, decoded.Description)
	assert.Equal(t, ds.Details, decoded.Details)
}

func TestEvaluationIssue_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	issue := EvaluationIssue{
		Severity:   IssueCritical,
		Category:   "completeness",
		Message:    "Missing required file",
		Location:   "evidence/ET-0001/",
		Suggestion: "Generate the evidence file",
	}

	data, err := json.Marshal(issue)
	require.NoError(t, err)

	var decoded EvaluationIssue
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, issue.Severity, decoded.Severity)
	assert.Equal(t, issue.Category, decoded.Category)
	assert.Equal(t, issue.Message, decoded.Message)
	assert.Equal(t, issue.Location, decoded.Location)
	assert.Equal(t, issue.Suggestion, decoded.Suggestion)
}

func TestEvaluationStatus_Constants(t *testing.T) {
	t.Parallel()
	assert.Equal(t, EvaluationStatus("pass"), EvaluationPass)
	assert.Equal(t, EvaluationStatus("warning"), EvaluationWarning)
	assert.Equal(t, EvaluationStatus("fail"), EvaluationFail)
}

func TestIssueSeverity_Constants(t *testing.T) {
	t.Parallel()
	assert.Equal(t, IssueSeverity("critical"), IssueCritical)
	assert.Equal(t, IssueSeverity("high"), IssueHigh)
	assert.Equal(t, IssueSeverity("medium"), IssueMedium)
	assert.Equal(t, IssueSeverity("low"), IssueLow)
	assert.Equal(t, IssueSeverity("info"), IssueInfo)
}
