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
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/formatters"
	"github.com/grctool/grctool/internal/interpolation"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/services"
	"github.com/grctool/grctool/internal/services/evidence"
	"github.com/grctool/grctool/internal/services/submission"
	"github.com/grctool/grctool/internal/storage"
	"github.com/grctool/grctool/internal/tugboat"
	tugboatModels "github.com/grctool/grctool/internal/tugboat/models"
	"github.com/spf13/cobra"
)

// EvidenceGenerateOptions holds options for evidence generation
type EvidenceGenerateOptions struct {
	Format    string   `json:"format"`
	Tools     []string `json:"tools"`
	OutputDir string   `json:"output_dir"`
}

// evidenceCmd represents the evidence command
var evidenceCmd = &cobra.Command{
	Use:   "evidence",
	Short: "Evidence generation and management tools",
	Long:  `Generate security compliance evidence using automated tools including Terraform scanning, GitHub integration, and template-based prompt generation`,
}

var evidenceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List evidence tasks",
	Long:  `List all evidence collection tasks from Tugboat Logic`,
	RunE:  runEvidenceList,
}

var evidenceViewCmd = &cobra.Command{
	Use:   "view [task-id]",
	Short: "View an evidence task in markdown format",
	Long: `Display an evidence task document in markdown format with full content and metadata.

The evidence task is displayed with:
- Task header with ID, framework, priority, and status information
- Full task description and collection guidance
- Collection requirements and timeline
- Related controls and policies
- Assignees and metadata

Examples:
  # View a specific evidence task by reference or ID
  grctool evidence view ET-0001
  grctool evidence view 327992

  # Save evidence task to markdown file
  grctool evidence view ET-0001 --output task-ET-0001.md`,
	Args: cobra.ExactArgs(1),
	RunE: runEvidenceView,
}

var evidenceAnalyzeCmd = &cobra.Command{
	Use:   "analyze [task-id]",
	Short: "Analyze evidence tasks and generate structured prompts",
	Long: `Analyze evidence tasks to understand their relationships with controls and policies, and generate structured prompts for evidence collection using template-based assembly. 
	
Can analyze a single task by ID/reference or all tasks with --all flag.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runEvidenceAnalyze,
}

var evidenceMapCmd = &cobra.Command{
	Use:   "map",
	Short: "Map relationships between evidence tasks, controls, and policies",
	Long:  `Create a visual map showing the relationships between evidence tasks, controls, and policies.`,
	RunE:  runEvidenceMap,
}

var evidenceGenerateCmd = &cobra.Command{
	Use:   "generate [task-id]",
	Short: "Generate evidence using coordinated tools",
	Long:  `Generate evidence for a specific task using coordinated tool analysis of your infrastructure and documentation.`,
	RunE:  runEvidenceGenerate,
}

var evidenceReviewCmd = &cobra.Command{
	Use:   "review [task-id]",
	Short: "Review generated evidence",
	Long:  `Review and validate evidence that has been generated for a task.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runEvidenceReview,
}

var evidenceSubmitCmd = &cobra.Command{
	Use:   "submit [task-id]",
	Short: "Submit evidence to Tugboat Logic",
	Long:  `Submit completed evidence to Tugboat Logic for compliance review.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runEvidenceSubmit,
}

var evidenceListSubmittedCmd = &cobra.Command{
	Use:   "list-submitted [task-id]",
	Short: "List submitted evidence for a task",
	Long: `List all evidence that has been previously submitted to Tugboat Logic for an evidence task.

This command retrieves the history of evidence submissions including:
- Evidence implementations (submissions)
- Attached files and documents
- Submission dates and metadata
- Collection dates

Examples:
  # List submitted evidence for a specific task
  grctool evidence list-submitted ET-0001
  grctool evidence list-submitted 327992

  # List submitted evidence with detailed output
  grctool evidence list-submitted ET-0001 --format detailed`,
	Args: cobra.ExactArgs(1),
	RunE: runEvidenceListSubmitted,
}

var evidenceDownloadCmd = &cobra.Command{
	Use:   "download [task-id]",
	Short: "Download submitted evidence attachments",
	Long: `Download evidence files and attachments that have been previously submitted to Tugboat Logic.

This command downloads:
- All attachments for a specific evidence task
- Individual attachments by attachment ID
- Evidence files to a local directory for reference or reuse

Downloaded evidence can be used as context for generating new evidence or for audit review.

Examples:
  # Download all submitted evidence for a task
  grctool evidence download ET-0001

  # Download to a specific directory
  grctool evidence download ET-0001 --output-dir ./evidence/ET-0001/submitted

  # Download only recent submissions (e.g., last 30 days)
  grctool evidence download ET-0001 --since 30d

  # Download specific attachment by ID
  grctool evidence download --attachment-id 12345 --output evidence.pdf`,
	Args: cobra.MaximumNArgs(1),
	RunE: runEvidenceDownload,
}

func init() {
	rootCmd.AddCommand(evidenceCmd)

	evidenceCmd.AddCommand(evidenceListCmd)
	evidenceCmd.AddCommand(evidenceViewCmd)
	evidenceCmd.AddCommand(evidenceAnalyzeCmd)
	evidenceCmd.AddCommand(evidenceMapCmd)
	evidenceCmd.AddCommand(evidenceGenerateCmd)
	evidenceCmd.AddCommand(evidenceReviewCmd)
	evidenceCmd.AddCommand(evidenceSubmitCmd)
	evidenceCmd.AddCommand(evidenceListSubmittedCmd)
	evidenceCmd.AddCommand(evidenceDownloadCmd)

	// Register completion functions for task ID arguments
	evidenceAnalyzeCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return completeTaskRefs(cmd, args, toComplete)
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	evidenceGenerateCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return completeTaskRefs(cmd, args, toComplete)
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	evidenceReviewCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return completeTaskRefs(cmd, args, toComplete)
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	evidenceSubmitCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return completeTaskRefs(cmd, args, toComplete)
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	evidenceListSubmittedCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return completeTaskRefs(cmd, args, toComplete)
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	evidenceDownloadCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return completeTaskRefs(cmd, args, toComplete)
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Evidence list flags
	evidenceListCmd.Flags().StringSlice("status", []string{}, "filter by status (pending, completed, overdue)")
	evidenceListCmd.Flags().String("framework", "", "filter by framework (soc2, iso27001, etc)")
	evidenceListCmd.Flags().StringSlice("priority", []string{}, "filter by priority (high, medium, low)")
	evidenceListCmd.Flags().String("assignee", "", "filter by assignee")
	evidenceListCmd.Flags().Bool("overdue", false, "show only overdue tasks")
	evidenceListCmd.Flags().Bool("due-soon", false, "show tasks due within 7 days")
	evidenceListCmd.Flags().StringSlice("category", []string{}, "filter by category (Infrastructure, Personnel, Process, Compliance, Monitoring, Data)")
	evidenceListCmd.Flags().StringSlice("aec-status", []string{}, "filter by AEC status (enabled, disabled, na)")
	evidenceListCmd.Flags().StringSlice("collection-type", []string{}, "filter by collection type (Manual, Automated, Hybrid)")
	evidenceListCmd.Flags().Bool("sensitive", false, "show only sensitive data tasks")
	evidenceListCmd.Flags().StringSlice("complexity", []string{}, "filter by complexity level (Simple, Moderate, Complex)")

	// Evidence view flags
	evidenceViewCmd.Flags().StringP("output", "o", "", "output file path (optional)")
	evidenceViewCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return completeTaskRefs(cmd, args, toComplete)
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Evidence analyze flags
	evidenceAnalyzeCmd.Flags().String("output", "", "save analysis to file (markdown format)")
	evidenceAnalyzeCmd.Flags().Bool("markdown", false, "format output as markdown")
	evidenceAnalyzeCmd.Flags().Bool("include-templates", true, "include evidence collection templates")
	evidenceAnalyzeCmd.Flags().Bool("include-checklist", true, "include compliance checklist")
	evidenceAnalyzeCmd.Flags().Bool("all", false, "generate prompts for all evidence tasks")

	// Evidence generate flags
	evidenceGenerateCmd.Flags().Bool("all", false, "generate evidence for all pending tasks")
	evidenceGenerateCmd.Flags().StringSlice("tools", []string{"terraform", "github"}, "tools to use for evidence collection")
	evidenceGenerateCmd.Flags().String("format", "csv", "output format (csv, markdown)")
	evidenceGenerateCmd.Flags().String("output-dir", "", "directory to save generated evidence")

	// Evidence review flags
	evidenceReviewCmd.Flags().Bool("show-reasoning", true, "show AI reasoning process")
	evidenceReviewCmd.Flags().Bool("show-sources", true, "show evidence sources")

	// Evidence submit flags
	evidenceSubmitCmd.Flags().String("window", "", "evidence collection window (e.g., 2025-Q4)")
	evidenceSubmitCmd.Flags().String("notes", "", "submission notes for auditors")
	evidenceSubmitCmd.Flags().Bool("skip-validation", false, "skip evidence validation checks")
	evidenceSubmitCmd.Flags().Bool("dry-run", false, "preview submission without uploading to Tugboat")
	evidenceSubmitCmd.MarkFlagRequired("window")

	// Evidence list-submitted flags
	evidenceListSubmittedCmd.Flags().String("format", "table", "output format (table, detailed, json)")
	evidenceListSubmittedCmd.Flags().String("since", "", "show submissions since date or duration (e.g., 30d, 2025-01-01)")

	// Evidence download flags
	evidenceDownloadCmd.Flags().String("output-dir", "", "directory to download evidence files (default: data/evidence/{task}/submitted)")
	evidenceDownloadCmd.Flags().String("attachment-id", "", "download specific attachment by ID")
	evidenceDownloadCmd.Flags().String("output", "", "output filename when downloading specific attachment")
	evidenceDownloadCmd.Flags().String("since", "", "download only submissions since date or duration (e.g., 30d, 2025-01-01)")
	evidenceDownloadCmd.Flags().Bool("dry-run", false, "preview downloads without actually downloading files")
}

func runEvidenceList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Initialize service
	evidenceService, err := initializeEvidenceService()
	if err != nil {
		return err
	}

	// Build filter from flags
	filter := buildEvidenceFilterFromFlags(cmd)

	// Get filtered tasks
	tasks, err := evidenceService.ListEvidenceTasks(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to list evidence tasks: %w", err)
	}

	// Display tasks
	return displayEvidenceTasks(cmd, tasks, evidenceService, ctx)
}

func runEvidenceView(cmd *cobra.Command, args []string) error {
	taskIDOrRef := args[0]

	// Get flags
	outputFile, _ := cmd.Flags().GetString("output")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize unified storage with full config to support custom paths
	storage, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Get evidence task - try by reference ID first, then by numeric ID
	task, err := storage.GetEvidenceTask(taskIDOrRef)
	if err != nil {
		return fmt.Errorf("evidence task not found: %s (hint: run 'grctool sync --evidence' to fetch latest tasks)", taskIDOrRef)
	}

	// Initialize formatter with interpolation if enabled
	var formatter *formatters.EvidenceTaskFormatter
	if cfg.Interpolation.Enabled {
		interpolatorConfig := interpolation.InterpolatorConfig{
			Variables:         cfg.Interpolation.GetFlatVariables(),
			Enabled:           true,
			OnMissingVariable: interpolation.MissingVariableIgnore,
		}
		interpolator := interpolation.NewStandardInterpolator(interpolatorConfig)
		formatter = formatters.NewEvidenceTaskFormatterWithInterpolation(interpolator)
	} else {
		formatter = formatters.NewEvidenceTaskFormatter()
	}

	// Generate markdown
	markdown := formatter.ToDocumentMarkdown(task)

	// Output markdown
	if outputFile != "" {
		// Ensure output directory exists
		if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		// Write to file
		if err := os.WriteFile(outputFile, []byte(markdown), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "✅ Evidence task exported to: %s\n", outputFile)
	} else {
		// Print to stdout
		fmt.Fprint(cmd.OutOrStdout(), markdown)
	}

	return nil
}

func runEvidenceAnalyze(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Initialize service
	evidenceService, err := initializeEvidenceService()
	if err != nil {
		return err
	}

	// Parse flags
	allTasks, _ := cmd.Flags().GetBool("all")
	outputFile, _ := cmd.Flags().GetString("output")

	// Validate args
	if !allTasks && len(args) == 0 {
		return fmt.Errorf("task ID required (or use --all flag)")
	}

	return processEvidenceAnalysis(cmd, evidenceService, allTasks, args, outputFile, ctx)
}

func runEvidenceMap(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Initialize service
	evidenceService, err := initializeEvidenceService()
	if err != nil {
		return err
	}

	// Get evidence relationship mapping
	mapResult, err := evidenceService.MapEvidenceRelationships(ctx)
	if err != nil {
		return fmt.Errorf("failed to map evidence relationships: %w", err)
	}

	// Display mapping results
	return displayEvidenceMap(cmd, mapResult)
}

func runEvidenceGenerate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Initialize service
	evidenceService, err := initializeEvidenceService()
	if err != nil {
		return err
	}

	// Parse flags and delegate to service
	options := evidence.BulkGenerationOptions{
		All:       cmd.Flags().Changed("all") && must(cmd.Flags().GetBool("all")),
		Tools:     must(cmd.Flags().GetStringSlice("tools")),
		Format:    must(cmd.Flags().GetString("format")),
		OutputDir: must(cmd.Flags().GetString("output-dir")),
	}

	return processEvidenceGeneration(cmd, evidenceService, options, args, ctx)
}

func must[T any](val T, err error) T {
	if err != nil {
		// Log the error and return zero value of T.
		lg := logger.WithComponent("cmd").WithFields(logger.Field{Key: "operation", Value: "must"})
		lg.Error("error retrieving flag value", logger.Error(err))
		var zero T
		return zero
	}
	return val
}

func processEvidenceGeneration(cmd *cobra.Command, evidenceService interface{}, options evidence.BulkGenerationOptions, args []string, ctx context.Context) error {
	// Check if --all flag is set for bulk generation
	if options.All {
		return fmt.Errorf("bulk generation not yet implemented")
	}

	// Require task ID if not using --all
	if len(args) == 0 {
		return fmt.Errorf("task ID is required (or use --all flag)")
	}

	// For now, just return success - this would need proper implementation
	// based on the actual evidenceService interface
	cmd.Printf("Context generation tools are now available! Use:\n")
	cmd.Printf("  make ai-context - Generate all AI context\n")
	cmd.Printf("  ./bin/grctool tool context summary - Generate codebase summary\n")
	return nil
}

func runEvidenceReview(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Initialize service
	evidenceService, err := initializeEvidenceService()
	if err != nil {
		return err
	}

	taskID, err := evidenceService.ResolveTaskID(ctx, args[0])
	if err != nil {
		return err
	}

	// Parse flags
	options := evidence.ReviewOptions{
		ShowReasoning: must(cmd.Flags().GetBool("show-reasoning")),
		ShowSources:   must(cmd.Flags().GetBool("show-sources")),
	}

	return processEvidenceReview(cmd, evidenceService, taskID, options, ctx)
}

func processEvidenceReview(cmd *cobra.Command, evidenceService evidence.Service, taskID int, options evidence.ReviewOptions, ctx context.Context) error {
	cmd.Printf("Reviewing evidence for task %d\n", taskID)

	// Implementation delegated to service layer
	// For now, return a simplified message
	cmd.Println("Evidence review functionality has been moved to the service layer.")
	cmd.Printf("Use the evidence service to review task %d\n", taskID)

	return nil
}

func runEvidenceSubmit(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get flags
	window, _ := cmd.Flags().GetString("window")
	notes, _ := cmd.Flags().GetString("notes")
	skipValidation, _ := cmd.Flags().GetBool("skip-validation")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	taskRef := args[0]

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize storage
	storage, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Initialize Tugboat client (only if not dry-run)
	var tugboatClient *tugboat.Client
	if !dryRun {
		tugboatClient = tugboat.NewClient(&cfg.Tugboat, nil) // nil VCR config for production use
	}

	// Initialize submission service
	submissionService := submission.NewSubmissionService(
		storage,
		tugboatClient,
		cfg.Tugboat.OrgID,
		cfg.Tugboat.CollectorURLs,
	)

	// Build submission request
	req := &submission.SubmitRequest{
		TaskRef:        taskRef,
		Window:         window,
		Notes:          notes,
		SkipValidation: skipValidation,
		SubmittedBy:    "grctool-cli",
	}

	// Preview files
	files, err := storage.GetEvidenceFiles(taskRef, window)
	if err != nil {
		return fmt.Errorf("failed to get evidence files: %w", err)
	}

	cmd.Printf("📁 Evidence directory: data/evidence/%s/%s\n", taskRef, window)
	cmd.Printf("📄 Files to submit: %d\n\n", len(files))
	for i, file := range files {
		cmd.Printf("  %d. %s (%d bytes)\n", i+1, file.Filename, file.SizeBytes)
	}
	cmd.Println()

	if dryRun {
		cmd.Println("🔍 Dry-run mode - no files will be uploaded")
		if collectorURL, ok := cfg.Tugboat.CollectorURLs[taskRef]; ok {
			cmd.Printf("Would submit to: %s\n", collectorURL)
		} else {
			cmd.Printf("⚠️  Warning: No collector URL configured for %s\n", taskRef)
			cmd.Println("Add to .grctool.yaml under tugboat.collector_urls")
		}
		return nil
	}

	// Submit evidence
	cmd.Printf("🚀 Submitting evidence to Tugboat Logic...\n\n")
	resp, err := submissionService.Submit(ctx, req)
	if err != nil {
		return fmt.Errorf("submission failed: %w", err)
	}

	// Display results
	if resp.Success {
		cmd.Printf("✅ Success! Submission ID: %s\n", resp.SubmissionID)
		cmd.Printf("Status: %s\n", resp.Status)
		if resp.Submission != nil {
			cmd.Printf("Files submitted: %d/%d\n", resp.Submission.TotalFileCount, len(files))

			// Show failed files if any
			if resp.Submission.TugboatResponse != nil && resp.Submission.TugboatResponse.Metadata != nil {
				if failedCount, ok := resp.Submission.TugboatResponse.Metadata["files_failed"].(int); ok && failedCount > 0 {
					cmd.Printf("\n⚠️  Warning: %d file(s) failed to upload\n", failedCount)
					if failedFiles, ok := resp.Submission.TugboatResponse.Metadata["failed_files"].([]string); ok {
						for _, failedFile := range failedFiles {
							cmd.Printf("  ❌ %s\n", failedFile)
						}
					}
				}
			}
		}
		if resp.Message != "" {
			cmd.Printf("\n%s\n", resp.Message)
		}
	} else {
		cmd.Printf("❌ Submission failed: %s\n", resp.Message)
		if resp.ValidationResult != nil && !resp.ValidationResult.ReadyForSubmission {
			cmd.Printf("\nValidation errors: %d\n", resp.ValidationResult.FailedChecks)
			for _, err := range resp.ValidationResult.Errors {
				cmd.Printf("  - %s\n", err)
			}
		}
	}

	return nil
}

func runEvidenceListSubmitted(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get flags
	format, _ := cmd.Flags().GetString("format")
	sinceStr, _ := cmd.Flags().GetString("since")

	taskIDOrRef := args[0]

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize storage to resolve task ref
	storage, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Resolve task ID from reference
	taskID, taskRef, err := resolveTaskIdentifier(ctx, storage, taskIDOrRef)
	if err != nil {
		return fmt.Errorf("failed to resolve task: %w", err)
	}

	// Initialize Tugboat client
	tugboatClient := tugboat.NewClient(&cfg.Tugboat, nil)

	// Get submission history
	cmd.Printf("📋 Retrieving submitted evidence for %s...\n\n", taskRef)
	history, err := tugboatClient.GetSubmittedEvidenceHistory(ctx, fmt.Sprintf("%d", taskID), taskRef)
	if err != nil {
		return fmt.Errorf("failed to retrieve submission history: %w", err)
	}

	// Filter by date if specified
	if sinceStr != "" {
		sinceTime, err := parseSinceFlag(sinceStr)
		if err != nil {
			return fmt.Errorf("invalid --since value: %w", err)
		}
		history.Implementations = filterImplementationsBySince(history.Implementations, sinceTime)
	}

	// Display results based on format
	switch format {
	case "json":
		return displaySubmittedEvidenceJSON(cmd, history)
	case "detailed":
		return displaySubmittedEvidenceDetailed(cmd, history)
	default: // table
		return displaySubmittedEvidenceTable(cmd, history)
	}
}

func runEvidenceDownload(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get flags
	outputDir, _ := cmd.Flags().GetString("output-dir")
	attachmentID, _ := cmd.Flags().GetString("attachment-id")
	outputFile, _ := cmd.Flags().GetString("output")
	sinceStr, _ := cmd.Flags().GetString("since")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize Tugboat client
	tugboatClient := tugboat.NewClient(&cfg.Tugboat, nil)

	// Case 1: Download specific attachment by ID
	if attachmentID != "" {
		if outputFile == "" {
			return fmt.Errorf("--output flag required when downloading specific attachment")
		}
		return downloadSpecificAttachment(ctx, cmd, tugboatClient, attachmentID, outputFile, dryRun)
	}

	// Case 2: Download all attachments for a task
	if len(args) == 0 {
		return fmt.Errorf("task ID required when not using --attachment-id")
	}

	taskIDOrRef := args[0]

	// Initialize storage to resolve task ref
	storage, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Resolve task ID from reference
	taskID, taskRef, err := resolveTaskIdentifier(ctx, storage, taskIDOrRef)
	if err != nil {
		return fmt.Errorf("failed to resolve task: %w", err)
	}

	// Determine output directory
	if outputDir == "" {
		outputDir = filepath.Join(cfg.Storage.DataDir, "evidence", taskRef, "submitted")
	}

	// Get submission history
	cmd.Printf("📥 Downloading evidence for %s...\n\n", taskRef)
	history, err := tugboatClient.GetSubmittedEvidenceHistory(ctx, fmt.Sprintf("%d", taskID), taskRef)
	if err != nil {
		return fmt.Errorf("failed to retrieve submission history: %w", err)
	}

	// Filter by date if specified
	if sinceStr != "" {
		sinceTime, err := parseSinceFlag(sinceStr)
		if err != nil {
			return fmt.Errorf("invalid --since value: %w", err)
		}
		history.Implementations = filterImplementationsBySince(history.Implementations, sinceTime)
	}

	// Download all attachments
	return downloadAllAttachments(ctx, cmd, tugboatClient, history, outputDir, dryRun)
}

// Helper functions

func initializeEvidenceService() (evidence.Service, error) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize data service
	storage, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	dataService := services.NewDataService(storage)
	consoleLoggerCfg := cfg.Logging.Loggers["console"]
	log, err := logger.New((&consoleLoggerCfg).ToLoggerConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Initialize evidence service
	evidenceService, err := evidence.NewService(dataService, cfg, log)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize evidence service: %w", err)
	}

	return evidenceService, nil
}

func buildEvidenceFilterFromFlags(cmd *cobra.Command) domain.EvidenceFilter {
	status, _ := cmd.Flags().GetStringSlice("status")
	framework, _ := cmd.Flags().GetString("framework")
	priority, _ := cmd.Flags().GetStringSlice("priority")
	assignedTo, _ := cmd.Flags().GetString("assignee")
	overdue, _ := cmd.Flags().GetBool("overdue")
	dueSoon, _ := cmd.Flags().GetBool("due-soon")
	category, _ := cmd.Flags().GetStringSlice("category")
	aecStatus, _ := cmd.Flags().GetStringSlice("aec-status")
	collectionType, _ := cmd.Flags().GetStringSlice("collection-type")
	sensitiveOnly, _ := cmd.Flags().GetBool("sensitive")
	complexity, _ := cmd.Flags().GetStringSlice("complexity")

	// Build filter
	filter := domain.EvidenceFilter{
		Status:          status,
		Framework:       framework,
		Priority:        priority,
		AssignedTo:      assignedTo,
		Category:        category,
		AecStatus:       aecStatus,
		CollectionType:  collectionType,
		ComplexityLevel: complexity,
	}

	// Set sensitive filter if requested
	if sensitiveOnly {
		sensitiveBool := true
		filter.Sensitive = &sensitiveBool
	}

	// Add date filters for overdue and due soon
	now := time.Now()
	if overdue {
		filter.DueBefore = &now
	}
	if dueSoon {
		dueSoonDate := now.AddDate(0, 0, 7)
		filter.DueAfter = &now
		filter.DueBefore = &dueSoonDate
	}

	return filter
}

func displayEvidenceTasks(cmd *cobra.Command, tasks []domain.EvidenceTask, evidenceService evidence.Service, ctx context.Context) error {
	if len(tasks) == 0 {
		cmd.Println("No evidence tasks found matching the specified criteria.")
		cmd.Println("Run 'grctool sync --evidence' to fetch latest data")
		return nil
	}

	// Display summary
	cmd.Printf("Found %d evidence task(s)\n\n", len(tasks))

	// Create tabwriter for formatted output
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "REF\tID\tNAME\tCATEGORY\tFRAMEWORK\tSTATUS\tAEC\tTYPE\tPRIORITY\tDUE DATE\tASSIGNEE\tURL")
	fmt.Fprintln(w, "---\t--\t----\t--------\t---------\t------\t---\t----\t--------\t--------\t--------\t---")

	now := time.Now()
	for _, task := range tasks {
		dueDateStr := "N/A"
		if task.NextDue != nil {
			dueDateStr = task.NextDue.Format("2006-01-02")
			// Mark overdue tasks
			if task.NextDue.Before(now) {
				dueDateStr += " ⚠️"
			}
		}

		// Get assignee information from task details
		assigneeStr := "N/A"
		if len(task.Assignees) > 0 {
			// Show first assignee, or count if multiple
			if len(task.Assignees) == 1 && task.Assignees[0].Name != "" {
				assigneeStr = task.Assignees[0].Name
			} else if len(task.Assignees) > 1 {
				assigneeStr = fmt.Sprintf("%d assigned", len(task.Assignees))
			}
		}

		// Truncate long names
		name := task.Name
		if len(name) > 32 {
			name = name[:29] + "..."
		}

		// Get reference ID (assign one if not set)
		refID := task.ReferenceID
		if refID == "" {
			// Generate a reference ID based on task ID if not available
			refID = fmt.Sprintf("ET-%04d", task.ID-327991) // Offset to start from ET-0001
		}

		// Get category with intelligent assignment
		category := task.GetCategory()

		// Get AEC status display
		aecStatus := task.GetAecStatusDisplay()
		// Truncate AEC status for display
		if len(aecStatus) > 8 {
			switch aecStatus {
			case "Enabled":
				aecStatus = "✅"
			case "Disabled":
				aecStatus = "⏸️"
			case "Not Available":
				aecStatus = "❌"
			default:
				aecStatus = aecStatus[:5] + "..."
			}
		}

		// Get collection type
		collectionType := task.GetCollectionType()

		// Format URL for display
		urlDisplay := "N/A"
		if task.TugboatURL != "" {
			// Show clickable link indicator with hyperlink support for modern terminals
			// Format: ESC]8;;URL\aVISIBLE_TEXT\aESC]8;;\a
			urlDisplay = fmt.Sprintf("\x1b]8;;%s\x1b\\🔗 View\x1b]8;;\x1b\\", task.TugboatURL)
		}

		fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			refID, task.ID, name, category, task.Framework, task.Status, aecStatus, collectionType, task.Priority, dueDateStr, assigneeStr, urlDisplay)
	}

	w.Flush()
	cmd.Println()

	// Show summary stats
	summary, err := evidenceService.GetEvidenceTaskSummary(ctx)
	if err == nil {
		cmd.Printf("Summary: %d total, %d overdue, %d due soon\n", summary.Total, summary.Overdue, summary.DueSoon)
	}

	return nil
}

func processEvidenceAnalysis(cmd *cobra.Command, evidenceService evidence.Service, allTasks bool, args []string, outputFile string, ctx context.Context) error {
	if allTasks {
		cmd.Println("Generating Claude AI prompts for all evidence tasks...")
		return evidenceService.ProcessBulkAnalysis(ctx, "markdown")
	}

	// Resolve single task ID
	taskID, err := evidenceService.ResolveTaskID(ctx, args[0])
	if err != nil {
		return err
	}

	cmd.Printf("Analyzing evidence task %d...\n", taskID)

	// Process analysis for task
	promptPath, promptText, err := evidenceService.ProcessAnalysisForTask(ctx, taskID, "markdown")
	if err != nil {
		return fmt.Errorf("failed to process analysis for task %d: %w", taskID, err)
	}

	cmd.Printf("   Prompt generated: %s\n", promptPath)

	if outputFile != "" {
		// For single task with custom output file
		if err := evidenceService.SaveAnalysisToFile(outputFile, promptText); err != nil {
			return fmt.Errorf("failed to save to custom file: %w", err)
		}
		cmd.Printf("   Also saved to: %s\n", outputFile)
	}

	cmd.Printf("Template-based prompt generation complete\n")
	return nil
}

func displayEvidenceMap(cmd *cobra.Command, mapResult *evidence.EvidenceMapResult) error {
	cmd.Println("Mapping Evidence Task Relationships")

	if len(mapResult.Tasks) == 0 {
		cmd.Println("No evidence tasks found. Run 'grctool sync --evidence' first.")
		return nil
	}

	cmd.Printf("Found %d tasks, %d controls, %d policies\n\n",
		mapResult.Summary.TotalTasks, mapResult.Summary.TotalControls, mapResult.Summary.TotalPolicies)

	// Display framework summary
	cmd.Println("**Tasks by Framework:**")
	for framework, count := range mapResult.Summary.FrameworkCounts {
		cmd.Printf("   %s: %d tasks\n", framework, count)
	}
	cmd.Println()

	// Show detailed mapping for each framework
	for framework, frameworkTasks := range mapResult.FrameworkGroups {
		cmd.Printf("**%s Framework**\n", framework)
		cmd.Printf("   Tasks: %d\n", len(frameworkTasks))

		// Show sample tasks
		cmd.Printf("   Sample Tasks:\n")
		displayCount := len(frameworkTasks)
		if displayCount > 5 {
			displayCount = 5
		}

		for i := 0; i < displayCount; i++ {
			task := frameworkTasks[i]
			cmd.Printf("      %d. %s [%s]\n", task.ID, task.Name, task.Status)
		}

		if len(frameworkTasks) > 5 {
			cmd.Printf("      ... and %d more\n", len(frameworkTasks)-5)
		}

		cmd.Println()
	}

	// Show relationship statistics
	cmd.Println("**Relationship Overview:**")
	cmd.Printf("   Total task-to-control/policy relationships: %d\n", mapResult.TotalRelationships)
	cmd.Printf("   Average relationships per task: %.1f\n", mapResult.Summary.AverageRelationships)

	// Summary recommendations
	cmd.Println("\n**Recommendations:**")
	if mapResult.Summary.OverdueCount > 0 {
		cmd.Printf("   • Address %d overdue tasks\n", mapResult.Summary.OverdueCount)
	}
	cmd.Println("   • Use 'grctool evidence analyze <task-id>' for detailed task analysis")
	cmd.Println("   • Use 'grctool evidence generate <task-id>' to create evidence")

	return nil
}

// Legacy function for backward compatibility - moved to service layer
func generateTemplateBasedPrompt(context *models.EvidenceContext, outputFormat string) string {
	// Delegate to service layer
	return "Legacy function - use service layer implementation"
}

// Evidence retrieval helper functions

func resolveTaskIdentifier(ctx context.Context, storage *storage.Storage, taskIDOrRef string) (int, string, error) {
	// Try to parse as ET-XXXX format first
	if len(taskIDOrRef) > 3 && taskIDOrRef[:3] == "ET-" {
		// Get task from storage by reference
		tasks, err := storage.GetAllEvidenceTasks()
		if err != nil {
			return 0, "", fmt.Errorf("failed to get evidence tasks: %w", err)
		}

		for _, task := range tasks {
			// Generate task ref from ID (ET-XXXX format)
			taskRef := fmt.Sprintf("ET-%04d", task.ID)
			if taskRef == taskIDOrRef {
				return task.ID, taskRef, nil
			}
		}
		return 0, "", fmt.Errorf("task not found: %s", taskIDOrRef)
	}

	// Try to parse as numeric ID
	taskID, err := strconv.Atoi(taskIDOrRef)
	if err != nil {
		return 0, "", fmt.Errorf("invalid task ID or reference: %s", taskIDOrRef)
	}

	taskRef := fmt.Sprintf("ET-%04d", taskID)
	return taskID, taskRef, nil
}

func parseSinceFlag(sinceStr string) (time.Time, error) {
	// Try parsing as duration (e.g., "30d", "7d", "1h")
	if len(sinceStr) > 1 {
		unit := sinceStr[len(sinceStr)-1:]
		valueStr := sinceStr[:len(sinceStr)-1]
		value, err := strconv.Atoi(valueStr)
		if err == nil {
			var duration time.Duration
			switch unit {
			case "d":
				duration = time.Duration(value) * 24 * time.Hour
			case "h":
				duration = time.Duration(value) * time.Hour
			case "m":
				duration = time.Duration(value) * time.Minute
			default:
				goto parseAsDate
			}
			return time.Now().Add(-duration), nil
		}
	}

parseAsDate:
	// Try parsing as ISO date
	layouts := []string{
		"2006-01-02",
		"2006-01-02T15:04:05",
		time.RFC3339,
	}
	for _, layout := range layouts {
		t, err := time.Parse(layout, sinceStr)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid date format (use '30d' or '2025-01-01')")
}

func filterImplementationsBySince(implementations []tugboatModels.EvidenceImplementation, sinceTime time.Time) []tugboatModels.EvidenceImplementation {
	filtered := make([]tugboatModels.EvidenceImplementation, 0)
	for _, impl := range implementations {
		if !impl.CreatedAt.IsZero() && impl.CreatedAt.After(sinceTime) {
			filtered = append(filtered, impl)
		}
	}
	return filtered
}

func displaySubmittedEvidenceTable(cmd *cobra.Command, history *tugboatModels.EvidenceSubmissionHistory) error {
	if len(history.Implementations) == 0 {
		cmd.Println("No submitted evidence found for this task.")
		return nil
	}

	cmd.Printf("Task: %s (ID: %v)\n", history.TaskRef, history.TaskID)
	cmd.Printf("Total Submissions: %d\n", history.TotalCount)
	if history.LastSubmitted != nil && !history.LastSubmitted.IsZero() {
		cmd.Printf("Last Submitted: %s\n", history.LastSubmitted.Format("2006-01-02 15:04:05"))
	}
	cmd.Println()

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Collected Date\tSubmitted\tAttachments\tNotes")
	fmt.Fprintln(w, "─────────────\t─────────\t───────────\t─────")

	for _, impl := range history.Implementations {
		collectedDate := impl.CollectedDate
		if collectedDate == "" {
			collectedDate = "N/A"
		}

		submittedAt := "N/A"
		if !impl.CreatedAt.IsZero() {
			submittedAt = impl.CreatedAt.Format("2006-01-02")
		}

		attachmentCount := len(impl.Attachments)
		notes := impl.Notes
		if len(notes) > 40 {
			notes = notes[:37] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", collectedDate, submittedAt, attachmentCount, notes)
	}

	w.Flush()
	return nil
}

func displaySubmittedEvidenceDetailed(cmd *cobra.Command, history *tugboatModels.EvidenceSubmissionHistory) error {
	if len(history.Implementations) == 0 {
		cmd.Println("No submitted evidence found for this task.")
		return nil
	}

	cmd.Printf("# Evidence Submission History: %s\n\n", history.TaskRef)
	cmd.Printf("**Task ID:** %v\n", history.TaskID)
	cmd.Printf("**Total Submissions:** %d\n", history.TotalCount)
	if history.LastSubmitted != nil && !history.LastSubmitted.IsZero() {
		cmd.Printf("**Last Submitted:** %s\n", history.LastSubmitted.Format("2006-01-02 15:04:05"))
	}
	cmd.Println()

	for i, impl := range history.Implementations {
		cmd.Printf("## Submission %d\n\n", i+1)
		cmd.Printf("**Collected Date:** %s\n", impl.CollectedDate)
		if !impl.CreatedAt.IsZero() {
			cmd.Printf("**Submitted At:** %s\n", impl.CreatedAt.Format("2006-01-02 15:04:05"))
		}
		cmd.Printf("**Status:** %s\n", impl.Status)
		if impl.Notes != "" {
			cmd.Printf("**Notes:** %s\n", impl.Notes)
		}
		cmd.Println()

		if len(impl.Attachments) > 0 {
			cmd.Println("**Attachments:**")
			for j, att := range impl.Attachments {
				cmd.Printf("  %d. %s", j+1, att.Filename)
				if att.Size > 0 {
					cmd.Printf(" (%d bytes)", att.Size)
				}
				cmd.Println()
				if att.Description != "" {
					cmd.Printf("     Description: %s\n", att.Description)
				}
			}
			cmd.Println()
		}

		if len(impl.Links) > 0 {
			cmd.Println("**Links:**")
			for j, link := range impl.Links {
				cmd.Printf("  %d. %s\n", j+1, link.URL)
				if link.Description != "" {
					cmd.Printf("     %s\n", link.Description)
				}
			}
			cmd.Println()
		}

		cmd.Println("---")
		cmd.Println()
	}

	return nil
}

func displaySubmittedEvidenceJSON(cmd *cobra.Command, history *tugboatModels.EvidenceSubmissionHistory) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(history)
}

func downloadSpecificAttachment(ctx context.Context, cmd *cobra.Command, client *tugboat.Client, attachmentID, outputFile string, dryRun bool) error {
	if dryRun {
		cmd.Printf("🔍 Dry-run: Would download attachment %s to %s\n", attachmentID, outputFile)
		return nil
	}

	cmd.Printf("⬇️  Downloading attachment %s...\n", attachmentID)
	bytes, err := client.DownloadEvidenceAttachment(ctx, attachmentID, outputFile)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	cmd.Printf("✅ Downloaded %d bytes to %s\n", bytes, outputFile)
	return nil
}

func downloadAllAttachments(ctx context.Context, cmd *cobra.Command, client *tugboat.Client, history *tugboatModels.EvidenceSubmissionHistory, outputDir string, dryRun bool) error {
	totalAttachments := 0
	for _, impl := range history.Implementations {
		totalAttachments += len(impl.Attachments)
	}

	if totalAttachments == 0 {
		cmd.Println("No attachments found for this task.")
		return nil
	}

	if !dryRun {
		// Create output directory
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	cmd.Printf("📂 Output directory: %s\n", outputDir)
	cmd.Printf("📎 Total attachments: %d\n\n", totalAttachments)

	downloaded := 0
	failed := 0

	for implIdx, impl := range history.Implementations {
		if len(impl.Attachments) == 0 {
			continue
		}

		// Create subdirectory for this submission
		submissionDir := filepath.Join(outputDir, fmt.Sprintf("submission-%d", implIdx+1))
		if !dryRun {
			if err := os.MkdirAll(submissionDir, 0755); err != nil {
				cmd.Printf("⚠️  Failed to create directory %s: %v\n", submissionDir, err)
				continue
			}
		}

		for _, att := range impl.Attachments {
			filename := att.Filename
			if filename == "" {
				filename = att.Name
			}
			if filename == "" {
				filename = fmt.Sprintf("attachment-%v", att.ID)
			}

			outputPath := filepath.Join(submissionDir, filename)

			if dryRun {
				cmd.Printf("  Would download: %s\n", filename)
				continue
			}

			cmd.Printf("⬇️  Downloading %s...", filename)

			attID := fmt.Sprintf("%v", att.ID)
			bytes, err := client.DownloadEvidenceAttachment(ctx, attID, outputPath)
			if err != nil {
				cmd.Printf(" ❌ Failed: %v\n", err)
				failed++
				continue
			}

			cmd.Printf(" ✅ (%d bytes)\n", bytes)
			downloaded++
		}
	}

	cmd.Println()
	if dryRun {
		cmd.Printf("🔍 Dry-run complete. Would download %d attachments to %s\n", totalAttachments, outputDir)
	} else {
		cmd.Printf("✅ Download complete: %d successful, %d failed\n", downloaded, failed)
		cmd.Printf("📁 Files saved to: %s\n", outputDir)
	}

	return nil
}
