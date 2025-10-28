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

package services

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/storage"
)

// EvidenceEvaluatorService evaluates evidence against task requirements
type EvidenceEvaluatorService struct {
	evidenceDir string
	storage     *storage.Storage
	scanner     EvidenceScanner
	logger      logger.Logger
}

// NewEvidenceEvaluatorService creates a new evidence evaluator service
func NewEvidenceEvaluatorService(evidenceDir string, storage *storage.Storage, scanner EvidenceScanner, log logger.Logger) *EvidenceEvaluatorService {
	return &EvidenceEvaluatorService{
		evidenceDir: evidenceDir,
		storage:     storage,
		scanner:     scanner,
		logger:      log.WithComponent("evidence_evaluator"),
	}
}

// EvaluateWindow evaluates all evidence in a window (across all subfolders)
func (s *EvidenceEvaluatorService) EvaluateWindow(ctx context.Context, taskRef string, window string) (*models.EvaluationResult, error) {
	s.logger.Info("Evaluating evidence window",
		logger.String("task_ref", taskRef),
		logger.String("window", window))

	// Get task details
	task, err := s.getTaskDetails(taskRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get task details: %w", err)
	}

	// Scan the window
	windowState, err := s.scanner.ScanWindow(ctx, taskRef, window)
	if err != nil {
		return nil, fmt.Errorf("failed to scan window: %w", err)
	}

	// Create evaluation result
	result := models.NewEvaluationResult(taskRef, task.ID, window, "all")
	result.FileCount = windowState.FileCount
	result.TotalBytes = windowState.TotalBytes

	// Evaluate each dimension
	s.evaluateCompleteness(task, windowState, result)
	s.evaluateRequirementsMatch(task, windowState, result)
	s.evaluateQuality(task, windowState, result)
	s.evaluateControlAlignment(task, windowState, result)

	// Calculate overall score and determine status
	result.CalculateOverallScore()
	result.DetermineStatus()

	// Generate recommendations
	s.generateRecommendations(task, result)

	s.logger.Info("Evaluation complete",
		logger.String("task_ref", taskRef),
		logger.String("window", window),
		logger.Field{Key: "score", Value: result.OverallScore},
		logger.String("status", string(result.OverallStatus)))

	return result, nil
}

// EvaluateSubfolder evaluates evidence in a specific subfolder (.submitted/archive)
func (s *EvidenceEvaluatorService) EvaluateSubfolder(ctx context.Context, taskRef string, window string, subfolder string) (*models.EvaluationResult, error) {
	s.logger.Info("Evaluating evidence subfolder",
		logger.String("task_ref", taskRef),
		logger.String("window", window),
		logger.String("subfolder", subfolder))

	// Get task details
	task, err := s.getTaskDetails(taskRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get task details: %w", err)
	}

	// Get files from subfolder
	files, err := s.storage.GetEvidenceFilesFromSubfolder(taskRef, window, subfolder)
	if err != nil {
		return nil, fmt.Errorf("failed to get evidence files: %w", err)
	}

	// Create window state from files
	windowState := &models.WindowState{
		Window:    window,
		FileCount: len(files),
		Files:     make([]models.FileState, len(files)),
	}

	var totalBytes int64
	for i, file := range files {
		windowState.Files[i] = models.FileState{
			Filename:  file.Filename,
			SizeBytes: file.SizeBytes,
			Checksum:  file.ChecksumSHA256,
		}
		totalBytes += file.SizeBytes
	}
	windowState.TotalBytes = totalBytes

	// Create evaluation result
	result := models.NewEvaluationResult(taskRef, task.ID, window, subfolder)
	result.FileCount = windowState.FileCount
	result.TotalBytes = windowState.TotalBytes

	// Evaluate each dimension
	s.evaluateCompleteness(task, windowState, result)
	s.evaluateRequirementsMatch(task, windowState, result)
	s.evaluateQuality(task, windowState, result)
	s.evaluateControlAlignment(task, windowState, result)

	// Calculate overall score and determine status
	result.CalculateOverallScore()
	result.DetermineStatus()

	// Generate recommendations
	s.generateRecommendations(task, result)

	s.logger.Info("Evaluation complete",
		logger.String("task_ref", taskRef),
		logger.String("window", window),
		logger.String("subfolder", subfolder),
		logger.Field{Key: "score", Value: result.OverallScore},
		logger.String("status", string(result.OverallStatus)))

	return result, nil
}

// evaluateCompleteness evaluates if all required evidence is present
func (s *EvidenceEvaluatorService) evaluateCompleteness(task *domain.EvidenceTask, window *models.WindowState, result *models.EvaluationResult) {
	score := 0.0
	maxScore := 100.0

	// Check if any files exist (baseline)
	if window.FileCount == 0 {
		result.Completeness.Score = 0
		result.Completeness.Status = "fail"
		result.Completeness.Details = "No evidence files found"
		result.AddIssue(models.IssueCritical, "completeness", "No evidence files present", "", "Upload or generate evidence files")
		return
	}

	// Files exist - start with base score
	score = 40.0

	// Check for metadata presence (40 points total)
	if window.HasGenerationMeta {
		score += 15.0
		result.Completeness.Details += "Generation metadata present. "
	} else {
		result.AddIssue(models.IssueMedium, "completeness", "Missing generation metadata", ".generation/metadata.yaml", "Track how evidence was generated")
	}

	// Check file count reasonable (20 points)
	expectedFileCount := s.estimateExpectedFileCount(task)
	if window.FileCount >= expectedFileCount {
		score += 20.0
		result.Completeness.Details += fmt.Sprintf("File count adequate (%d files). ", window.FileCount)
	} else {
		score += float64(window.FileCount) / float64(expectedFileCount) * 20.0
		result.AddIssue(models.IssueMedium, "completeness",
			fmt.Sprintf("File count below expected (found %d, expected ~%d)", window.FileCount, expectedFileCount),
			"", "Ensure all required evidence is collected")
	}

	// Check total size reasonable (15 points)
	if window.TotalBytes > 1024 { // At least 1KB of content
		score += 15.0
	} else {
		score += 5.0
		result.AddIssue(models.IssueLow, "completeness",
			fmt.Sprintf("Evidence files very small (%d bytes total)", window.TotalBytes),
			"", "Ensure evidence contains sufficient detail")
	}

	// Check for recent files (10 points)
	if window.NewestFile != nil {
		score += 10.0
		result.Completeness.Details += "Evidence recently updated. "
	}

	result.Completeness.Score = score
	result.Completeness.MaxScore = maxScore

	if score >= 80 {
		result.Completeness.Status = "pass"
	} else if score >= 50 {
		result.Completeness.Status = "warning"
	} else {
		result.Completeness.Status = "fail"
	}
}

// evaluateRequirementsMatch evaluates if evidence matches task requirements
func (s *EvidenceEvaluatorService) evaluateRequirementsMatch(task *domain.EvidenceTask, window *models.WindowState, result *models.EvaluationResult) {
	score := 0.0
	maxScore := 100.0

	// Extract keywords from task description and guidance
	requiredKeywords := s.extractRequiredKeywords(task)

	// Check filenames for relevant keywords (40 points)
	matchCount := 0
	for _, file := range window.Files {
		filename := strings.ToLower(file.Filename)
		for _, keyword := range requiredKeywords {
			if strings.Contains(filename, keyword) {
				matchCount++
				break
			}
		}
	}

	if len(requiredKeywords) > 0 {
		keywordScore := (float64(matchCount) / float64(len(window.Files))) * 40.0
		score += keywordScore
		result.RequirementsMatch.Details = fmt.Sprintf("Filename relevance: %d/%d files match keywords. ", matchCount, len(window.Files))
	} else {
		// No specific keywords - give benefit of doubt
		score += 30.0
		result.RequirementsMatch.Details = "No specific keywords required. "
	}

	// Check if collection guidance is addressed (30 points)
	if task.Guidance != "" {
		// Look for evidence of following guidance
		// This is a heuristic - would need AI analysis for deep evaluation
		score += 20.0
		result.RequirementsMatch.Details += "Collection guidance present. "
	} else {
		score += 30.0
	}

	// Check for appropriate file types (30 points)
	expectedFormats := s.extractExpectedFormats(task)
	formatScore := s.checkFileFormats(window.Files, expectedFormats)
	score += formatScore * 30.0
	result.RequirementsMatch.Details += fmt.Sprintf("File format match: %.0f%%. ", formatScore*100)

	if formatScore < 0.5 {
		result.AddIssue(models.IssueMedium, "requirements",
			"Evidence file formats may not match expected types",
			"", fmt.Sprintf("Consider using formats: %s", strings.Join(expectedFormats, ", ")))
	}

	result.RequirementsMatch.Score = score
	result.RequirementsMatch.MaxScore = maxScore

	if score >= 80 {
		result.RequirementsMatch.Status = "pass"
	} else if score >= 50 {
		result.RequirementsMatch.Status = "warning"
	} else {
		result.RequirementsMatch.Status = "fail"
	}
}

// evaluateQuality evaluates evidence quality
func (s *EvidenceEvaluatorService) evaluateQuality(task *domain.EvidenceTask, window *models.WindowState, result *models.EvaluationResult) {
	score := 0.0
	maxScore := 100.0

	if window.FileCount == 0 {
		result.QualityScore.Score = 0
		result.QualityScore.Status = "fail"
		return
	}

	// Check file naming conventions (25 points)
	properlyNamed := 0
	for _, file := range window.Files {
		if s.hasProperNaming(file.Filename) {
			properlyNamed++
		}
	}
	namingScore := (float64(properlyNamed) / float64(len(window.Files))) * 25.0
	score += namingScore
	result.QualityScore.Details = fmt.Sprintf("Naming conventions: %d/%d files. ", properlyNamed, len(window.Files))

	if namingScore < 15 {
		result.AddIssue(models.IssueLow, "quality",
			"Some files don't follow naming conventions",
			"", "Use descriptive lowercase names with underscores")
	}

	// Check file sizes reasonable (25 points)
	reasonableSizes := 0
	for _, file := range window.Files {
		if file.SizeBytes > 100 && file.SizeBytes < 100*1024*1024 { // 100 bytes to 100MB
			reasonableSizes++
		}
	}
	sizeScore := (float64(reasonableSizes) / float64(len(window.Files))) * 25.0
	score += sizeScore

	// Check for structured formats (25 points)
	structuredCount := 0
	for _, file := range window.Files {
		ext := strings.ToLower(filepath.Ext(file.Filename))
		if ext == ".csv" || ext == ".json" || ext == ".yaml" || ext == ".xlsx" {
			structuredCount++
		}
	}
	if structuredCount > 0 {
		structScore := (float64(structuredCount) / float64(len(window.Files))) * 25.0
		score += structScore
		result.QualityScore.Details += fmt.Sprintf("Structured formats: %d/%d files. ", structuredCount, len(window.Files))
	} else {
		score += 10.0 // Give some points for having any files
		result.AddIssue(models.IssueLow, "quality",
			"No structured data formats (CSV, JSON, YAML)",
			"", "Consider using structured formats for better auditability")
	}

	// Check for documentation (25 points)
	hasDocumentation := false
	for _, file := range window.Files {
		ext := strings.ToLower(filepath.Ext(file.Filename))
		if ext == ".md" || ext == ".txt" || ext == ".pdf" {
			hasDocumentation = true
			break
		}
	}
	if hasDocumentation {
		score += 25.0
		result.QualityScore.Details += "Documentation present. "
	} else {
		score += 10.0
		result.AddIssue(models.IssueLow, "quality",
			"No documentation files found",
			"", "Consider adding a summary document or README")
	}

	result.QualityScore.Score = score
	result.QualityScore.MaxScore = maxScore

	if score >= 80 {
		result.QualityScore.Status = "pass"
	} else if score >= 50 {
		result.QualityScore.Status = "warning"
	} else {
		result.QualityScore.Status = "fail"
	}
}

// evaluateControlAlignment evaluates how well evidence addresses related controls
func (s *EvidenceEvaluatorService) evaluateControlAlignment(task *domain.EvidenceTask, window *models.WindowState, result *models.EvaluationResult) {
	score := 0.0
	maxScore := 100.0

	// Check if task has related controls
	if len(task.RelatedControls) == 0 {
		// No controls specified - give benefit of doubt
		score = 70.0
		result.ControlAlignment.Status = "pass"
		result.ControlAlignment.Details = "No specific controls to evaluate against. "
		result.ControlAlignment.Score = score
		result.ControlAlignment.MaxScore = maxScore
		return
	}

	// Base score for having evidence (30 points)
	if window.FileCount > 0 {
		score = 30.0
	}

	// Check if evidence addresses control domains (40 points)
	controlKeywords := s.extractControlKeywords(task.RelatedControls)
	coverage := s.calculateKeywordCoverage(window.Files, controlKeywords)
	score += coverage * 40.0
	result.ControlAlignment.Details = fmt.Sprintf("Control keyword coverage: %.0f%%. ", coverage*100)

	if coverage < 0.3 {
		result.AddIssue(models.IssueHigh, "control_alignment",
			"Evidence may not adequately address related controls",
			"", "Ensure evidence demonstrates control implementation")
	}

	// Check if multiple controls are addressed (30 points)
	if len(task.RelatedControls) > 0 {
		// Give full points if we have files - detailed analysis would require AI
		score += 30.0
		result.ControlAlignment.Details += fmt.Sprintf("Addresses %d control(s). ", len(task.RelatedControls))
	}

	result.ControlAlignment.Score = score
	result.ControlAlignment.MaxScore = maxScore

	if score >= 80 {
		result.ControlAlignment.Status = "pass"
	} else if score >= 50 {
		result.ControlAlignment.Status = "warning"
	} else {
		result.ControlAlignment.Status = "fail"
	}
}

// Helper methods

func (s *EvidenceEvaluatorService) getTaskDetails(taskRef string) (*domain.EvidenceTask, error) {
	// Extract task ID from reference (ET-0001 -> 1)
	var taskID int
	if _, err := fmt.Sscanf(taskRef, "ET-%d", &taskID); err != nil {
		return nil, fmt.Errorf("invalid task reference format: %s", taskRef)
	}

	return s.storage.GetEvidenceTask(fmt.Sprintf("%d", taskID))
}

func (s *EvidenceEvaluatorService) estimateExpectedFileCount(task *domain.EvidenceTask) int {
	// Heuristic: estimate based on task complexity
	baseCount := 1

	// More controls = more files expected
	if len(task.RelatedControls) > 3 {
		baseCount = 3
	} else if len(task.RelatedControls) > 1 {
		baseCount = 2
	}

	// Complex descriptions suggest more files
	if len(task.Description) > 500 {
		baseCount++
	}

	return baseCount
}

func (s *EvidenceEvaluatorService) extractRequiredKeywords(task *domain.EvidenceTask) []string {
	// Extract relevant keywords from task name and description
	text := strings.ToLower(task.Name + " " + task.Description)

	keywords := []string{}
	commonWords := []string{"github", "terraform", "access", "permissions", "security",
		"policy", "control", "users", "roles", "audit", "log", "review", "deployment"}

	for _, word := range commonWords {
		if strings.Contains(text, word) {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

func (s *EvidenceEvaluatorService) extractExpectedFormats(task *domain.EvidenceTask) []string {
	// Determine expected file formats based on task description
	formats := []string{"csv", "json", "md"} // Default formats

	text := strings.ToLower(task.Description)
	if strings.Contains(text, "screenshot") || strings.Contains(text, "image") {
		formats = append(formats, "png", "jpg")
	}
	if strings.Contains(text, "report") || strings.Contains(text, "document") {
		formats = append(formats, "pdf", "docx")
	}
	if strings.Contains(text, "spreadsheet") || strings.Contains(text, "table") {
		formats = append(formats, "xlsx", "csv")
	}

	return formats
}

func (s *EvidenceEvaluatorService) checkFileFormats(files []models.FileState, expectedFormats []string) float64 {
	if len(files) == 0 {
		return 0.0
	}

	matchCount := 0
	for _, file := range files {
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(file.Filename), "."))
		for _, format := range expectedFormats {
			if ext == format {
				matchCount++
				break
			}
		}
	}

	return float64(matchCount) / float64(len(files))
}

func (s *EvidenceEvaluatorService) hasProperNaming(filename string) bool {
	// Check if filename follows conventions:
	// - lowercase
	// - uses underscores
	// - descriptive (more than 5 chars before extension)

	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Check length
	if len(name) < 5 {
		return false
	}

	// Check if lowercase (allow numbers and underscores)
	if strings.ToLower(name) != name {
		return false
	}

	// Prefer underscores over spaces
	if strings.Contains(name, " ") {
		return false
	}

	return true
}

func (s *EvidenceEvaluatorService) extractControlKeywords(controls []domain.Control) []string {
	keywords := make(map[string]bool)

	for _, control := range controls {
		// Extract from control name
		name := strings.ToLower(control.Name)
		words := strings.Fields(name)
		for _, word := range words {
			if len(word) > 4 { // Skip short words
				keywords[word] = true
			}
		}
	}

	result := make([]string, 0, len(keywords))
	for k := range keywords {
		result = append(result, k)
	}
	return result
}

func (s *EvidenceEvaluatorService) calculateKeywordCoverage(files []models.FileState, keywords []string) float64 {
	if len(keywords) == 0 {
		return 1.0 // No keywords to match
	}

	matchedKeywords := make(map[string]bool)

	for _, file := range files {
		filename := strings.ToLower(file.Filename)
		for _, keyword := range keywords {
			if strings.Contains(filename, keyword) {
				matchedKeywords[keyword] = true
			}
		}
	}

	return float64(len(matchedKeywords)) / float64(len(keywords))
}

func (s *EvidenceEvaluatorService) generateRecommendations(task *domain.EvidenceTask, result *models.EvaluationResult) {
	// Generate recommendations based on scores and issues

	if result.Completeness.Score < 70 {
		result.AddRecommendation("Add more evidence files to improve completeness")
	}

	if result.RequirementsMatch.Score < 70 {
		result.AddRecommendation("Review task requirements and ensure evidence addresses all requirements")
	}

	if result.QualityScore.Score < 70 {
		result.AddRecommendation("Improve evidence quality by using structured formats and better naming")
	}

	if result.ControlAlignment.Score < 70 {
		result.AddRecommendation(fmt.Sprintf("Ensure evidence demonstrates implementation of all %d related controls", len(task.RelatedControls)))
	}

	if result.FileCount < 2 {
		result.AddRecommendation("Consider adding supporting documentation or additional evidence files")
	}

	if result.GetCriticalIssueCount() > 0 {
		result.AddRecommendation("Address all critical issues before submission")
	} else if result.GetHighIssueCount() > 0 {
		result.AddRecommendation("Review and address high-priority issues")
	}
}
