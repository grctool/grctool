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

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/naming"
	"github.com/grctool/grctool/internal/services"
	"github.com/grctool/grctool/internal/storage"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// storageAdapter adapts storage.Storage to services.Storage interface
type storageAdapterEvaluate struct {
	storage *storage.Storage
}

func (sa *storageAdapterEvaluate) GetEvidenceTask(ctx context.Context, taskID int) (*domain.EvidenceTask, error) {
	idStr := fmt.Sprintf("%d", taskID)
	return sa.storage.GetEvidenceTask(idStr)
}

var evidenceEvaluateCmd = &cobra.Command{
	Use:   "evaluate [task-ref]",
	Short: "Evaluate evidence against task requirements",
	Long: `Evaluate evidence comprehensively across four dimensions:

Evaluation Dimensions:
  1. Completeness (30%) - Required files and metadata present
  2. Requirements Match (30%) - Evidence addresses task requirements
  3. Quality (20%) - File formats, naming, and structure
  4. Control Alignment (20%) - Evidence addresses related controls

The evaluation produces:
  - Overall score (0-100) and pass/fail status
  - Individual dimension scores
  - Specific issues with severity levels
  - Recommendations for improvement

Results are displayed in the terminal AND saved to .validation/validation.yaml

Examples:
  # Evaluate all evidence in a window (root + .submitted + archive)
  grctool evidence evaluate ET-0001 --window 2025-Q4

  # Evaluate specific subfolder only
  grctool evidence evaluate ET-0001 --window 2025-Q4 --subfolder .submitted

  # Evaluate all tasks
  grctool evidence evaluate --all`,
	Args: cobra.MaximumNArgs(1),
	RunE: runEvidenceEvaluate,
}

func init() {
	evidenceEvaluateCmd.Flags().String("window", "", "specific window to evaluate (e.g., 2025-Q4)")
	evidenceEvaluateCmd.Flags().String("subfolder", "", "evaluate specific subfolder only (.submitted/archive)")
	evidenceEvaluateCmd.Flags().Bool("all", false, "evaluate all evidence tasks")
	evidenceEvaluateCmd.Flags().StringP("output", "o", "", "output results to JSON file")
	evidenceEvaluateCmd.Flags().Bool("save-validation", true, "save results to .validation/validation.yaml")
	evidenceEvaluateCmd.Flags().Bool("verbose", false, "show detailed evaluation information")
}

func runEvidenceEvaluate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get flags
	window, _ := cmd.Flags().GetString("window")
	subfolder, _ := cmd.Flags().GetString("subfolder")
	all, _ := cmd.Flags().GetBool("all")
	outputFile, _ := cmd.Flags().GetString("output")
	saveValidation, _ := cmd.Flags().GetBool("save-validation")
	verbose, _ := cmd.Flags().GetBool("verbose")

	// Validate arguments
	if !all && len(args) == 0 {
		return fmt.Errorf("task reference required (or use --all flag)")
	}

	if !all && window == "" {
		return fmt.Errorf("--window flag required when evaluating specific task")
	}

	if all && len(args) > 0 {
		return fmt.Errorf("cannot specify task reference with --all flag")
	}

	if subfolder != "" && subfolder != naming.SubfolderSubmitted && subfolder != naming.SubfolderArchive {
		return fmt.Errorf("invalid subfolder: must be %s or %s", naming.SubfolderSubmitted, naming.SubfolderArchive)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create services
	evidenceDir := filepath.Join(cfg.Storage.DataDir, "evidence")

	// Initialize logger
	consoleLoggerCfg := cfg.Logging.Loggers["console"]
	log, err := logger.New((&consoleLoggerCfg).ToLoggerConfig())
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Initialize storage
	store, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Wrap storage with adapter
	storageAdapter := &storageAdapterEvaluate{storage: store}

	scanner := services.NewEvidenceScanner(evidenceDir, storageAdapter, log)
	evaluatorService := services.NewEvidenceEvaluatorService(evidenceDir, store, scanner, log)

	if all {
		// Evaluate all tasks
		return evaluateAllTasks(ctx, scanner, evaluatorService, store, saveValidation, verbose, outputFile)
	} else {
		// Evaluate specific task
		taskRef := args[0]
		return evaluateTask(ctx, evaluatorService, store, taskRef, window, subfolder, saveValidation, verbose, outputFile)
	}
}

func evaluateTask(ctx context.Context, evaluator *services.EvidenceEvaluatorService, storage *storage.Storage,
	taskRef, window, subfolder string, saveValidation, verbose bool, outputFile string) error {

	fmt.Printf("Evaluating evidence for %s / %s", taskRef, window)
	if subfolder != "" {
		fmt.Printf(" / %s", subfolder)
	}
	fmt.Println()

	var result *models.EvaluationResult
	var err error

	// Evaluate
	if subfolder != "" {
		result, err = evaluator.EvaluateSubfolder(ctx, taskRef, window, subfolder)
	} else {
		result, err = evaluator.EvaluateWindow(ctx, taskRef, window)
	}

	if err != nil {
		return fmt.Errorf("evaluation failed: %w", err)
	}

	// Display results
	displayEvaluationResult(result, verbose)

	// Save validation metadata if requested
	if saveValidation && subfolder != "" {
		if err := saveValidationMetadata(storage, result); err != nil {
			fmt.Printf("\n⚠ Warning: Failed to save validation metadata: %v\n", err)
		} else {
			fmt.Printf("\n✓ Validation metadata saved to %s/%s/.validation/validation.yaml\n",
				result.Window, result.Subfolder)
		}
	}

	// Save to output file if requested
	if outputFile != "" {
		if err := saveEvaluationResultToFile(result, outputFile); err != nil {
			return fmt.Errorf("failed to save results: %w", err)
		}
		fmt.Printf("✓ Results saved to: %s\n", outputFile)
	}

	return nil
}

func evaluateAllTasks(ctx context.Context, scanner services.EvidenceScanner, evaluator *services.EvidenceEvaluatorService,
	storage *storage.Storage, saveValidation, verbose bool, outputFile string) error {

	fmt.Println("Evaluating all evidence tasks...")

	// Scan all evidence
	taskStates, err := scanner.ScanAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to scan evidence: %w", err)
	}

	results := []*models.EvaluationResult{}
	errors := []string{}

	// Evaluate each task's windows
	for taskRef, taskState := range taskStates {
		for window := range taskState.Windows {
			// Check context cancellation
			if ctx.Err() != nil {
				return ctx.Err()
			}

			fmt.Printf("Evaluating %s / %s...\n", taskRef, window)

			result, err := evaluator.EvaluateWindow(ctx, taskRef, window)
			if err != nil {
				errMsg := fmt.Sprintf("%s/%s: %v", taskRef, window, err)
				errors = append(errors, errMsg)
				fmt.Printf("  ✗ Failed: %v\n\n", err)
				continue
			}

			results = append(results, result)

			// Show brief result
			status := getStatusEmoji(result.OverallStatus)
			fmt.Printf("  %s Score: %.1f/100 (%s)\n\n", status, result.OverallScore, result.OverallStatus)
		}
	}

	// Display summary
	fmt.Println("=== Evaluation Summary ===")
	displayEvaluationSummary(results, errors)

	// Save all results if output file specified
	if outputFile != "" {
		if err := saveAllEvaluationResults(results, outputFile); err != nil {
			return fmt.Errorf("failed to save results: %w", err)
		}
		fmt.Printf("\n✓ All results saved to: %s\n", outputFile)
	}

	return nil
}

func displayEvaluationResult(result *models.EvaluationResult, verbose bool) {
	// Overall score and status
	status := getStatusEmoji(result.OverallStatus)
	fmt.Printf("%s Overall Score: %.1f/100 (%s)\n\n", status, result.OverallScore, strings.ToUpper(string(result.OverallStatus)))

	// Dimension scores
	fmt.Println("Dimension Scores:")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "  Dimension\tScore\tWeight\tStatus\n")
	fmt.Fprintf(w, "  ---------\t-----\t------\t------\n")

	displayDimension(w, "Completeness", result.Completeness)
	displayDimension(w, "Requirements Match", result.RequirementsMatch)
	displayDimension(w, "Quality", result.QualityScore)
	displayDimension(w, "Control Alignment", result.ControlAlignment)

	w.Flush()
	fmt.Println()

	// Evidence summary
	fmt.Printf("Evidence Files: %d files (%.2f KB total)\n\n",
		result.FileCount, float64(result.TotalBytes)/1024)

	// Issues
	if len(result.Issues) > 0 {
		fmt.Println("Issues Found:")
		for _, issue := range result.Issues {
			severityEmoji := getSeverityEmoji(issue.Severity)
			fmt.Printf("  %s [%s] %s\n", severityEmoji, strings.ToUpper(string(issue.Severity)), issue.Message)
			if issue.Location != "" {
				fmt.Printf("      Location: %s\n", issue.Location)
			}
			if issue.Suggestion != "" {
				fmt.Printf("      Suggestion: %s\n", issue.Suggestion)
			}
		}
		fmt.Println()
	} else {
		fmt.Println("✓ No issues found")
	}

	// Recommendations
	if len(result.Recommendations) > 0 {
		fmt.Println("Recommendations:")
		for _, rec := range result.Recommendations {
			fmt.Printf("  • %s\n", rec)
		}
		fmt.Println()
	}

	// Verbose details
	if verbose {
		fmt.Println("=== Detailed Evaluation ===")
		fmt.Printf("\nCompleteness: %s\n", result.Completeness.Details)
		fmt.Printf("Requirements Match: %s\n", result.RequirementsMatch.Details)
		fmt.Printf("Quality: %s\n", result.QualityScore.Details)
		fmt.Printf("Control Alignment: %s\n", result.ControlAlignment.Details)
		fmt.Println()
	}
}

func displayDimension(w *tabwriter.Writer, name string, dimension models.DimensionScore) {
	status := getDimensionStatusEmoji(dimension.Status)
	fmt.Fprintf(w, "  %s\t%.1f/%.0f\t%.0f%%\t%s %s\n",
		name, dimension.Score, dimension.MaxScore, dimension.Weight*100, status, dimension.Status)
}

func displayEvaluationSummary(results []*models.EvaluationResult, errors []string) {
	if len(results) == 0 && len(errors) == 0 {
		fmt.Println("No evaluations performed")
		return
	}

	// Count by status
	statusCounts := make(map[models.EvaluationStatus]int)
	totalScore := 0.0
	for _, result := range results {
		statusCounts[result.OverallStatus]++
		totalScore += result.OverallScore
	}

	fmt.Printf("Total evaluations: %d\n", len(results))
	if len(results) > 0 {
		fmt.Printf("Average score: %.1f/100\n", totalScore/float64(len(results)))
	}
	fmt.Println()

	fmt.Println("Status breakdown:")
	fmt.Printf("  ✓ Pass: %d\n", statusCounts[models.EvaluationPass])
	fmt.Printf("  ⚠ Warning: %d\n", statusCounts[models.EvaluationWarning])
	fmt.Printf("  ✗ Fail: %d\n", statusCounts[models.EvaluationFail])

	if len(errors) > 0 {
		fmt.Printf("\nErrors: %d\n", len(errors))
		for _, err := range errors {
			fmt.Printf("  ✗ %s\n", err)
		}
	}
}

func saveValidationMetadata(storage *storage.Storage, result *models.EvaluationResult) error {
	// Convert EvaluationResult to ValidationResult format
	validationResult := &models.ValidationResult{
		TaskRef:             result.TaskRef,
		Window:              result.Window,
		Status:              string(result.OverallStatus),
		ValidationMode:      "comprehensive",
		CompletenessScore:   result.OverallScore / 100.0,
		TotalChecks:         4, // Four evaluation dimensions
		PassedChecks:        countPassedDimensions(result),
		FailedChecks:        countFailedDimensions(result),
		Warnings:            countWarnings(result),
		Errors:              convertIssuesToValidationErrors(result.Issues),
		WarningsList:        []models.ValidationError{},
		Checks:              buildValidationChecks(result),
		EvidenceFiles:       []models.EvidenceFileRef{},
		ReadyForSubmission:  result.OverallStatus == models.EvaluationPass,
		ValidationTimestamp: result.EvaluatedAt,
	}

	return storage.SaveValidationResultToSubfolder(result.TaskRef, result.Window, validationResult, result.Subfolder)
}

func countPassedDimensions(result *models.EvaluationResult) int {
	count := 0
	if result.Completeness.Status == "pass" {
		count++
	}
	if result.RequirementsMatch.Status == "pass" {
		count++
	}
	if result.QualityScore.Status == "pass" {
		count++
	}
	if result.ControlAlignment.Status == "pass" {
		count++
	}
	return count
}

func countFailedDimensions(result *models.EvaluationResult) int {
	count := 0
	if result.Completeness.Status == "fail" {
		count++
	}
	if result.RequirementsMatch.Status == "fail" {
		count++
	}
	if result.QualityScore.Status == "fail" {
		count++
	}
	if result.ControlAlignment.Status == "fail" {
		count++
	}
	return count
}

func countWarnings(result *models.EvaluationResult) int {
	count := 0
	for _, issue := range result.Issues {
		if issue.Severity == models.IssueHigh || issue.Severity == models.IssueMedium {
			count++
		}
	}
	return count
}

func convertIssuesToValidationErrors(issues []models.EvaluationIssue) []models.ValidationError {
	errors := []models.ValidationError{}
	for _, issue := range issues {
		if issue.Severity == models.IssueCritical || issue.Severity == models.IssueHigh {
			errors = append(errors, models.ValidationError{
				Code:     fmt.Sprintf("%s_error", issue.Category),
				Message:  issue.Message,
				Field:    issue.Location,
				Severity: string(issue.Severity),
			})
		}
	}
	return errors
}

func buildValidationChecks(result *models.EvaluationResult) []models.ValidationCheck {
	checks := []models.ValidationCheck{
		{
			Code:     "completeness",
			Name:     "Completeness",
			Status:   result.Completeness.Status,
			Severity: getSeverityFromStatus(result.Completeness.Status),
			Message:  result.Completeness.Details,
		},
		{
			Code:     "requirements_match",
			Name:     "Requirements Match",
			Status:   result.RequirementsMatch.Status,
			Severity: getSeverityFromStatus(result.RequirementsMatch.Status),
			Message:  result.RequirementsMatch.Details,
		},
		{
			Code:     "quality",
			Name:     "Quality",
			Status:   result.QualityScore.Status,
			Severity: getSeverityFromStatus(result.QualityScore.Status),
			Message:  result.QualityScore.Details,
		},
		{
			Code:     "control_alignment",
			Name:     "Control Alignment",
			Status:   result.ControlAlignment.Status,
			Severity: getSeverityFromStatus(result.ControlAlignment.Status),
			Message:  result.ControlAlignment.Details,
		},
	}
	return checks
}

func getSeverityFromStatus(status string) string {
	switch status {
	case "fail":
		return "error"
	case "warning":
		return "warning"
	default:
		return "info"
	}
}

func saveEvaluationResultToFile(result *models.EvaluationResult, filename string) error {
	// Determine format from extension
	ext := filepath.Ext(filename)

	var data []byte
	var err error

	switch ext {
	case ".json":
		data, err = json.MarshalIndent(result, "", "  ")
	case ".yaml", ".yml":
		data, err = yaml.Marshal(result)
	default:
		// Default to JSON
		data, err = json.MarshalIndent(result, "", "  ")
	}

	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func saveAllEvaluationResults(results []*models.EvaluationResult, filename string) error {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Helper functions for display

func getStatusEmoji(status models.EvaluationStatus) string {
	switch status {
	case models.EvaluationPass:
		return "✓"
	case models.EvaluationWarning:
		return "⚠"
	case models.EvaluationFail:
		return "✗"
	default:
		return "?"
	}
}

func getDimensionStatusEmoji(status string) string {
	switch status {
	case "pass":
		return "✓"
	case "warning":
		return "⚠"
	case "fail":
		return "✗"
	default:
		return "?"
	}
}

func getSeverityEmoji(severity models.IssueSeverity) string {
	switch severity {
	case models.IssueCritical:
		return "🔴"
	case models.IssueHigh:
		return "🟠"
	case models.IssueMedium:
		return "🟡"
	case models.IssueLow:
		return "🟢"
	case models.IssueInfo:
		return "ℹ️"
	default:
		return "•"
	}
}
