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
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/formatters"
	"github.com/grctool/grctool/internal/interpolation"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/naming"
	"github.com/grctool/grctool/internal/services"
	"github.com/grctool/grctool/internal/services/evidence"
	"github.com/grctool/grctool/internal/services/submission"
	"github.com/grctool/grctool/internal/storage"
	"github.com/grctool/grctool/internal/tools"
	"github.com/grctool/grctool/internal/tugboat"
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

func init() {
	rootCmd.AddCommand(evidenceCmd)

	evidenceCmd.AddCommand(evidenceListCmd)
	evidenceCmd.AddCommand(evidenceViewCmd)
	evidenceCmd.AddCommand(evidenceMapCmd)
	evidenceCmd.AddCommand(evidenceGenerateCmd)
	evidenceCmd.AddCommand(evidenceReviewCmd)
	evidenceCmd.AddCommand(evidenceSubmitCmd)
	evidenceCmd.AddCommand(evidenceEvaluateCmd)

	// Register completion functions for task ID arguments
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
	evidenceEvaluateCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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

	// Evidence generate flags
	evidenceGenerateCmd.Flags().Bool("all", false, "generate evidence for all pending tasks")
	evidenceGenerateCmd.Flags().StringSlice("tools", []string{}, "tools to use for evidence collection (auto-detect if empty)")
	evidenceGenerateCmd.Flags().String("format", "csv", "output format (csv, markdown)")
	evidenceGenerateCmd.Flags().String("output-dir", "", "directory to save generated evidence")
	evidenceGenerateCmd.Flags().String("window", "", "evidence collection window (e.g., 2025-Q4)")
	evidenceGenerateCmd.Flags().Bool("context-only", false, "only generate context document, don't prompt for generation")
	evidenceGenerateCmd.Flags().Bool("with-tool-data", false, "execute applicable tools and collect data during context generation")

	// Evidence review flags
	evidenceReviewCmd.Flags().String("window", "", "evidence collection window (e.g., 2025-Q4)")
	evidenceReviewCmd.Flags().Bool("show-reasoning", true, "show AI reasoning process")
	evidenceReviewCmd.Flags().Bool("show-sources", true, "show evidence sources")

	// Evidence submit flags
	evidenceSubmitCmd.Flags().String("window", "", "evidence collection window (e.g., 2025-Q4)")
	evidenceSubmitCmd.Flags().String("notes", "", "submission notes for auditors")
	evidenceSubmitCmd.Flags().Bool("skip-validation", false, "skip evidence validation checks")
	evidenceSubmitCmd.Flags().Bool("dry-run", false, "preview submission without uploading to Tugboat")
	evidenceSubmitCmd.MarkFlagRequired("window")
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
		return fmt.Errorf("evidence task not found: %s", taskIDOrRef)
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
		return processBulkEvidenceGeneration(cmd, evidenceService, options, ctx)
	}

	// Require task ID if not using --all
	if len(args) == 0 {
		return fmt.Errorf("task ID is required (or use --all flag)")
	}

	taskRef := args[0]
	window, _ := cmd.Flags().GetString("window")
	contextOnly, _ := cmd.Flags().GetBool("context-only")

	// Default window to current quarter if not specified
	if window == "" {
		window = getCurrentQuarter()
	}

	// Process single task
	return processSingleTaskGeneration(cmd, taskRef, window, contextOnly, options)
}

// executeApplicableTools executes applicable tools and saves their output
func executeApplicableTools(task *domain.EvidenceTask, toolNames []string, outputDir string, cfg *config.Config) error {
	if len(toolNames) == 0 {
		return nil // No tools to execute
	}

	for _, toolName := range toolNames {
		tool, err := tools.GetTool(toolName)
		if err != nil {
			// Skip unknown tools - don't fail the entire operation
			continue
		}

		// Execute tool
		ctx := context.Background()
		request := createToolRequestForEvidence(task, toolName, cfg)

		result, _, err := tool.Execute(ctx, request)
		if err != nil {
			// Log error but continue with other tools
			// Don't fail the entire operation for one tool failure
			continue
		}

		// Save tool output as JSON
		outputFile := filepath.Join(outputDir, fmt.Sprintf("%s.json", toolName))
		data := []byte(result)

		if err := os.WriteFile(outputFile, data, 0644); err != nil {
			continue
		}
	}

	return nil
}

// createToolRequestForEvidence creates a tool request based on task and tool type
func createToolRequestForEvidence(task *domain.EvidenceTask, toolName string, cfg *config.Config) map[string]interface{} {
	// Create a basic request structure for the tool
	// Different tools may need different request structures
	// This creates a generic request that most tools can accept

	baseRequest := map[string]interface{}{
		"task_ref":    task.ReferenceID,
		"task_name":   task.Name,
		"description": task.Description,
	}

	// Tool-specific request customization
	switch toolName {
	case "terraform-security-indexer":
		baseRequest["query_type"] = "control_mapping"
	case "terraform-security-analyzer":
		baseRequest["security_domain"] = "all"
	case "github-permissions":
		if cfg.Evidence.Tools.GitHub.Repository != "" {
			baseRequest["repository"] = cfg.Evidence.Tools.GitHub.Repository
		}
	case "github-security-features":
		if cfg.Evidence.Tools.GitHub.Repository != "" {
			baseRequest["repository"] = cfg.Evidence.Tools.GitHub.Repository
		}
	case "github-workflow-analyzer":
		if cfg.Evidence.Tools.GitHub.Repository != "" {
			baseRequest["repository"] = cfg.Evidence.Tools.GitHub.Repository
		}
	}

	return baseRequest
}

func processSingleTaskGeneration(cmd *cobra.Command, taskRef string, window string, contextOnly bool, options evidence.BulkGenerationOptions) error {
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

	// Load evidence task
	cmd.Printf("Loading task %s...\n", taskRef)
	task, err := storage.GetEvidenceTask(taskRef)
	if err != nil {
		return fmt.Errorf("evidence task not found: %s", taskRef)
	}

	// Check if this is a Tugboat-managed task (AEC enabled + Hybrid collection)
	if isTugboatManagedTask(task) {
		displayTugboatManagedMessage(cmd, task)
		return nil // Skip assembly context generation
	}

	// Generate comprehensive assembly context
	assemblyContext, err := generateAssemblyContext(task, window, options.Tools, cfg, storage)
	if err != nil {
		return fmt.Errorf("failed to generate assembly context: %w", err)
	}

	// Save comprehensive assembly materials to root directory
	assemblyPaths, err := saveAssemblyContext(task, window, assemblyContext, cfg.Storage.DataDir)
	if err != nil {
		return fmt.Errorf("failed to save assembly context: %w", err)
	}

	// Execute tools if requested
	withToolData, _ := cmd.Flags().GetBool("with-tool-data")
	if withToolData && len(assemblyContext.ApplicableTools) > 0 {
		cmd.Printf("🔧 Executing %d applicable tool(s)...\n", len(assemblyContext.ApplicableTools))
		if err := executeApplicableTools(task, assemblyContext.ApplicableTools, assemblyPaths.ToolDataDir, cfg); err != nil {
			// Log warning but continue
			cmd.Printf("⚠️  Warning: Some tools failed to execute: %v\n", err)
		} else {
			cmd.Printf("✅ Tool data collected in: %s\n", assemblyPaths.ToolDataDir)
		}
	}

	// Output success with new structure
	cmd.Printf("✅ Assembly context created for %s: %s\n\n", task.ReferenceID, task.Name)
	cmd.Printf("📄 Assembly prompt: %s\n", assemblyPaths.PromptFile)
	cmd.Printf("📋 Claude instructions: %s\n", assemblyPaths.InstructionsFile)
	cmd.Printf("📝 Evidence template: %s\n", assemblyPaths.TemplateFile)

	if !contextOnly {
		cmd.Println("\nNext steps:")
		cmd.Println("  1. Ask Claude: 'Help me generate evidence for " + task.ReferenceID + "'")
		cmd.Println("  2. Claude will read the assembly prompt and guide you through:")
		cmd.Println("     - Running applicable tools")
		cmd.Println("     - Using evidence-generator for synthesis")
		cmd.Println("     - Creating comprehensive report")
	}

	return nil
}

func processBulkEvidenceGeneration(cmd *cobra.Command, evidenceService interface{}, options evidence.BulkGenerationOptions, ctx context.Context) error {
	window, _ := cmd.Flags().GetString("window")
	contextOnly, _ := cmd.Flags().GetBool("context-only")

	// Default window to current quarter if not specified
	if window == "" {
		window = getCurrentQuarter()
	}

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

	// Type assert evidenceService to get the correct interface
	svc, ok := evidenceService.(evidence.Service)
	if !ok {
		return fmt.Errorf("invalid evidence service type")
	}

	cmd.Println("Loading pending evidence tasks...")

	// Get all pending tasks (not completed)
	filter := domain.EvidenceFilter{}
	allTasks, err := svc.ListEvidenceTasks(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to list evidence tasks: %w", err)
	}

	// Filter for non-completed tasks
	var pendingTasks []domain.EvidenceTask
	for _, task := range allTasks {
		if !task.Completed && strings.ToLower(task.Status) != "completed" {
			pendingTasks = append(pendingTasks, task)
		}
	}

	if len(pendingTasks) == 0 {
		cmd.Println("No pending evidence tasks found.")
		return nil
	}

	cmd.Printf("Found %d pending task(s)\n\n", len(pendingTasks))
	cmd.Println("Generating comprehensive assembly contexts:")

	// Track results
	var successCount, failureCount int
	var failedTasks []string

	// Process each task with assembly context
	for i, task := range pendingTasks {
		taskNum := i + 1
		cmd.Printf("  [%d/%d] %s - %s", taskNum, len(pendingTasks), task.ReferenceID, task.Name)

		// Generate COMPREHENSIVE assembly context (not minimal)
		assemblyContext, err := generateAssemblyContext(&task, window, options.Tools, cfg, storage)
		if err != nil {
			cmd.Printf(" ⚠️  Failed: %v\n", err)
			failureCount++
			failedTasks = append(failedTasks, fmt.Sprintf("%s (%s)", task.ReferenceID, err.Error()))
			continue
		}

		// Save assembly materials
		_, err = saveAssemblyContext(&task, window, assemblyContext, cfg.Storage.DataDir)
		if err != nil {
			cmd.Printf(" ⚠️  Failed to save: %v\n", err)
			failureCount++
			failedTasks = append(failedTasks, fmt.Sprintf("%s (%s)", task.ReferenceID, err.Error()))
			continue
		}

		cmd.Printf(" ✅\n")
		successCount++
	}

	// Display summary
	cmd.Print("\n" + strings.Repeat("=", 60) + "\n")
	cmd.Print("Assembly Context Generation Complete\n")
	cmd.Printf("  ✅ Successful: %d tasks\n", successCount)
	if failureCount > 0 {
		cmd.Printf("  ⚠️  Failed: %d tasks\n", failureCount)
		cmd.Println("\nFailed tasks:")
		for _, task := range failedTasks {
			cmd.Printf("    - %s\n", task)
		}
	}

	if !contextOnly {
		cmd.Println("\nNext steps:")
		cmd.Println("  1. Ask Claude Code to help with evidence generation")
		cmd.Println("  2. Claude will read assembly prompts and guide you through:")
		cmd.Println("     - Running applicable tools")
		cmd.Println("     - Using evidence-generator for synthesis")
		cmd.Println("     - Creating comprehensive reports")
		cmd.Println("\n  Example: 'Help me generate all pending evidence'")
	}

	return nil
}

func runEvidenceReview(cmd *cobra.Command, args []string) error {
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

	// Get task by reference or ID
	taskRef := args[0]
	task, err := storage.GetEvidenceTask(taskRef)
	if err != nil {
		return fmt.Errorf("evidence task not found: %s", taskRef)
	}

	// Get window from flags or use current quarter
	window, _ := cmd.Flags().GetString("window")
	if window == "" {
		window = getCurrentQuarter()
	}

	// Display review header
	displayReviewHeader(cmd, task, window)

	// 1. Display evidence files summary
	files, err := storage.GetEvidenceFiles(task.ReferenceID, window)
	hasFiles := err == nil && len(files) > 0
	displayEvidenceFiles(cmd, files, err)

	// 2. Display validation status if available
	validationResult, validationErr := storage.LoadValidationResult(task.ReferenceID, window)
	hasValidation := validationErr == nil && validationResult != nil
	displayValidationStatus(cmd, validationResult, validationErr)

	// 3. Display requirements checklist
	displayRequirementsChecklist(cmd, task, hasFiles)

	// 4. Display control alignment
	displayControlAlignment(cmd, task, storage)

	// 5. Display submission recommendation
	alreadySubmitted, _ := storage.CheckAlreadySubmitted(task.ReferenceID, window)
	displaySubmissionRecommendation(cmd, task, window, hasFiles, hasValidation, validationResult, alreadySubmitted)

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

	// Check if files already exist in .submitted/ folder (prevents resubmission)
	alreadySubmitted, err := storage.CheckAlreadySubmitted(taskRef, window)
	if err != nil {
		return fmt.Errorf("failed to check submission status: %w", err)
	}
	if alreadySubmitted {
		cmd.Printf("⚠️  Evidence for %s/%s has already been submitted\n", taskRef, window)
		cmd.Println("Files are in .submitted/ folder. To resubmit:")
		cmd.Println("  1. Move files from .submitted/ back to root directory")
		cmd.Println("  2. Run submit command again")
		return nil
	}

	// Preview files (from root directory)
	files, err := storage.GetEvidenceFiles(taskRef, window)
	if err != nil {
		return fmt.Errorf("failed to get evidence files: %w", err)
	}

	cmd.Printf("📁 Evidence directory: data/evidence/%s/%s (root)\n", taskRef, window)
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

		// NEW HYBRID APPROACH: Move files to .submitted/ after successful upload
		cmd.Println("\n📦 Moving files to .submitted/ folder...")
		if err := storage.MoveEvidenceFilesToSubmitted(taskRef, window, files); err != nil {
			cmd.Printf("⚠️  Warning: Failed to move files to .submitted/: %v\n", err)
			cmd.Println("Files were uploaded successfully but remain in root directory")
		} else {
			cmd.Printf("✅ Files moved to .submitted/ (prevents resubmission)\n")
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

func displayEvidenceMap(cmd *cobra.Command, mapResult *evidence.EvidenceMapResult) error {
	cmd.Println("Mapping Evidence Task Relationships")

	if len(mapResult.Tasks) == 0 {
		cmd.Println("No evidence tasks found.")
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
	cmd.Println("   • Use 'grctool evidence generate <task-id>' to create evidence and assembly context")

	return nil
}

// Evidence context generation helpers

// EvidenceGenerationContext holds all context needed for evidence generation
type EvidenceGenerationContext struct {
	Task             *domain.EvidenceTask
	RelatedControls  []domain.Control
	ApplicableTools  []string
	ExistingEvidence []string
	SourceLocations  map[string]string
	PreviousWindows  []string
}

// AssemblyContext holds all materials for evidence assembly
type AssemblyContext struct {
	Task                *domain.EvidenceTask
	Window              string
	ComprehensivePrompt string                 // From prompt-assembler
	ClaudeInstructions  string                 // How to use materials
	EvidenceTemplate    string                 // Structure guide
	ApplicableTools     []string
	ToolData            map[string]interface{} // If --with-tool-data
}

// AssemblyPaths holds file paths for saved assembly materials
type AssemblyPaths struct {
	PromptFile       string
	InstructionsFile string
	TemplateFile     string
	ToolDataDir      string
}

// PromptAssemblerOutput holds the result from prompt-assembler tool
type PromptAssemblerOutput struct {
	Prompt string
	Data   map[string]interface{}
}

func getCurrentQuarter() string {
	now := time.Now()
	quarter := (int(now.Month())-1)/3 + 1
	return fmt.Sprintf("%d-Q%d", now.Year(), quarter)
}

func generateEvidenceContext(task *domain.EvidenceTask, window string, requestedTools []string, cfg *config.Config, storage *storage.Storage) (*EvidenceGenerationContext, error) {
	context := &EvidenceGenerationContext{
		Task:            task,
		SourceLocations: make(map[string]string),
	}

	// Identify applicable tools based on task description and name
	if len(requestedTools) > 0 {
		context.ApplicableTools = requestedTools
	} else {
		context.ApplicableTools = identifyApplicableTools(task)
	}

	// Scan for existing evidence
	evidenceDir := filepath.Join(cfg.Storage.DataDir, "evidence")
	taskDirName := naming.GetEvidenceTaskDirName(task.ReferenceID, task.Name)
	taskEvidenceDir := filepath.Join(evidenceDir, taskDirName)

	// Check for existing evidence windows
	if _, err := os.Stat(taskEvidenceDir); err == nil {
		entries, err := os.ReadDir(taskEvidenceDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() && entry.Name() != ".context" {
					context.PreviousWindows = append(context.PreviousWindows, entry.Name())

					// Check for files in this window
					windowDir := filepath.Join(taskEvidenceDir, entry.Name())
					files, err := os.ReadDir(windowDir)
					if err == nil {
						for _, file := range files {
							if !file.IsDir() && !strings.HasPrefix(file.Name(), ".") {
								context.ExistingEvidence = append(context.ExistingEvidence,
									fmt.Sprintf("%s/%s", entry.Name(), file.Name()))
							}
						}
					}
				}
			}
		}
	}

	// Add source locations from config
	if len(cfg.Evidence.Tools.Terraform.ScanPaths) > 0 {
		context.SourceLocations["Terraform"] = fmt.Sprintf("%d path(s) configured", len(cfg.Evidence.Tools.Terraform.ScanPaths))
	} else if cfg.Evidence.Terraform.AtmosPath != "" || cfg.Evidence.Terraform.RepoPath != "" {
		context.SourceLocations["Terraform"] = "Configured"
	}
	if cfg.Evidence.Tools.GitHub.Enabled && cfg.Evidence.Tools.GitHub.Repository != "" {
		context.SourceLocations["GitHub"] = cfg.Evidence.Tools.GitHub.Repository
	}
	if cfg.Evidence.Tools.GoogleDocs.Enabled && cfg.Evidence.Tools.GoogleDocs.CredentialsFile != "" {
		context.SourceLocations["Google Workspace"] = "Configured"
	}

	// Load related controls (if available)
	// This would require extending storage to get controls by task
	// For now, we'll use the control IDs from the task itself
	if len(task.Controls) > 0 {
		for _, controlID := range task.Controls {
			control, err := storage.GetControl(controlID)
			if err == nil {
				context.RelatedControls = append(context.RelatedControls, *control)
			}
		}
	}

	return context, nil
}

func identifyApplicableTools(task *domain.EvidenceTask) []string {
	var tools []string

	// Combine task name and description for keyword matching
	searchText := strings.ToLower(task.Name + " " + task.Description)

	// Tool patterns - map keywords to tool names
	toolPatterns := map[string][]string{
		"github-permissions":          {"github", "repository", "access control", "permissions", "team"},
		"github-repo-analyzer":        {"github", "repository", "branch protection", "security"},
		"github-workflow-analyzer":    {"github", "ci/cd", "workflow", "pipeline", "actions"},
		"terraform-security-indexer":  {"terraform", "infrastructure", "iac", "security"},
		"terraform-security-analyzer": {"terraform", "cloud", "aws", "gcp", "azure"},
		"google-workspace":            {"google", "drive", "docs", "sheets", "forms", "workspace"},
		"atmos-stack-analyzer":        {"atmos", "stack", "environment"},
	}

	// Check each tool pattern
	for tool, keywords := range toolPatterns {
		for _, keyword := range keywords {
			if strings.Contains(searchText, keyword) {
				tools = append(tools, tool)
				break
			}
		}
	}

	return tools
}

func formatContextAsMarkdown(context *EvidenceGenerationContext, task *domain.EvidenceTask, window string) string {
	var md strings.Builder

	md.WriteString(fmt.Sprintf("# Evidence Generation Context: %s\n\n", task.ReferenceID))

	// Task Details
	md.WriteString("## Task Details\n\n")
	md.WriteString(fmt.Sprintf("- **Reference**: %s\n", task.ReferenceID))
	md.WriteString(fmt.Sprintf("- **Name**: %s\n", task.Name))
	md.WriteString(fmt.Sprintf("- **Framework**: %s\n", task.Framework))
	md.WriteString(fmt.Sprintf("- **Priority**: %s\n", task.Priority))
	md.WriteString(fmt.Sprintf("- **Collection Interval**: %s\n", task.CollectionInterval))
	if task.NextDue != nil {
		md.WriteString(fmt.Sprintf("- **Due Date**: %s\n", task.NextDue.Format("2006-01-02")))
	}
	md.WriteString("\n")

	// Description
	if task.Description != "" {
		md.WriteString("## Description\n\n")
		md.WriteString(task.Description)
		md.WriteString("\n\n")
	}

	// Related Controls
	if len(context.RelatedControls) > 0 {
		md.WriteString("## Related Controls\n\n")
		for _, control := range context.RelatedControls {
			md.WriteString(fmt.Sprintf("- **%s**: %s\n", control.ReferenceID, control.Name))
			if control.Description != "" {
				md.WriteString(fmt.Sprintf("  %s\n", control.Description))
			}
		}
		md.WriteString("\n")
	} else if len(task.Controls) > 0 {
		md.WriteString("## Related Controls\n\n")
		for _, controlID := range task.Controls {
			md.WriteString(fmt.Sprintf("- %s\n", controlID))
		}
		md.WriteString("\n")
	}

	// Applicable Tools
	md.WriteString("## Applicable Tools\n\n")
	if len(context.ApplicableTools) > 0 {
		for i, tool := range context.ApplicableTools {
			md.WriteString(fmt.Sprintf("%d. **%s**\n", i+1, tool))
		}
	} else {
		md.WriteString("*No automated tools identified. This may require manual evidence collection.*\n")
	}
	md.WriteString("\n")

	// Required Evidence
	md.WriteString("## Required Evidence\n\n")
	md.WriteString(fmt.Sprintf("- **Evidence Window**: %s\n", window))
	md.WriteString("- **Format**: CSV, JSON, PDF, or Markdown\n")
	md.WriteString("- **Purpose**: Demonstrate compliance with related controls\n")
	md.WriteString("\n")

	// Available Source Data
	if len(context.SourceLocations) > 0 {
		md.WriteString("## Available Source Data\n\n")
		for source, location := range context.SourceLocations {
			md.WriteString(fmt.Sprintf("- **%s**: %s\n", source, location))
		}
		md.WriteString("\n")
	}

	// Previous Evidence
	if len(context.ExistingEvidence) > 0 {
		md.WriteString("## Previous Evidence\n\n")
		if len(context.PreviousWindows) > 0 {
			md.WriteString(fmt.Sprintf("Found evidence in %d window(s):\n\n", len(context.PreviousWindows)))
		}
		for _, evidenceFile := range context.ExistingEvidence {
			md.WriteString(fmt.Sprintf("- %s\n", evidenceFile))
		}
		md.WriteString("\n")
	}

	// Suggested Workflow
	md.WriteString("## Suggested Workflow\n\n")
	if len(context.ApplicableTools) > 0 {
		for i, tool := range context.ApplicableTools {
			md.WriteString(fmt.Sprintf("%d. Run: `grctool tool %s`\n", i+1, tool))
		}
		md.WriteString(fmt.Sprintf("%d. Synthesize results into evidence document\n", len(context.ApplicableTools)+1))
		md.WriteString(fmt.Sprintf("%d. Save using: `grctool tool evidence-writer --task-ref %s --title \"Evidence Report\" --file evidence.csv`\n",
			len(context.ApplicableTools)+2, task.ReferenceID))
	} else {
		md.WriteString("1. Manually collect required evidence\n")
		md.WriteString("2. Create evidence file (CSV, PDF, etc.)\n")
		md.WriteString(fmt.Sprintf("3. Save using: `grctool tool evidence-writer --task-ref %s --title \"Evidence Report\" --file evidence.pdf`\n",
			task.ReferenceID))
	}
	md.WriteString("\n")

	// Task URL
	if task.TugboatURL != "" {
		md.WriteString("## Additional Information\n\n")
		md.WriteString(fmt.Sprintf("- **Tugboat Task**: %s\n", task.TugboatURL))
		md.WriteString("\n")
	}

	return md.String()
}

func saveEvidenceContext(task *domain.EvidenceTask, window string, contextMarkdown string, dataDir string) (string, error) {
	// Create evidence directory structure
	evidenceDir := filepath.Join(dataDir, "evidence")
	taskDirName := naming.GetEvidenceTaskDirName(task.ReferenceID, task.Name)
	windowDir := filepath.Join(evidenceDir, taskDirName, window)
	contextDir := filepath.Join(windowDir, ".context")

	// Create .context directory
	if err := os.MkdirAll(contextDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create context directory: %w", err)
	}

	// Write context file
	contextPath := filepath.Join(contextDir, "generation-context.md")
	if err := os.WriteFile(contextPath, []byte(contextMarkdown), 0644); err != nil {
		return "", fmt.Errorf("failed to write context file: %w", err)
	}

	return contextPath, nil
}

func generateClaudeInstructions(task *domain.EvidenceTask, window string) string {
	return fmt.Sprintf(`# Claude Code Instructions: %s

## Your Mission

Help the user generate evidence for **%s** (%s).

## What You Have

1. **Assembly Prompt** (.context/assembly-prompt.md)
   - Comprehensive context from prompt-assembler
   - Related controls and policies
   - Example evidence structure
   - All requirements for this task

2. **Evidence Template** (.context/evidence-template.md)
   - Pre-structured report outline
   - Section headers and prompts
   - Based on proven evidence patterns

3. **Tool Data** (.context/tool_outputs/ directory, if available)
   - Pre-collected data from automated tools
   - Ready for synthesis into evidence

## Dual Output Approach

You will generate TWO outputs:

### 1. Simple Evidence File (%s_Evidence.md - root directory)
**Purpose**: Clean, auditor-friendly evidence document
**Structure**:
- Header with task description and collection date
- Collection tasks broken down from task description/guidance
- Each task shows: Evidence statement → Inline snippet (quoted) → Source reference
- Flat file structure (no subfolders) ready for Tugboat upload

**Format Example**:
`+"```"+`markdown
## Collection Task 1: Document access provisioning process

**Evidence:** Policy requires manager approval for all access requests

**Source:** `+"`"+`POL-0001-access-control.md`+"`"+`
**Original Path:** `+"`"+`docs/policies/POL-0001-access-control.md`+"`"+`
**Last Modified:** 2025-09-15
**Section:** 3.2 (lines 45-52)

> All access requests must be:
> 1. Submitted via standardized request form
> 2. Approved by direct manager
> 3. Reviewed by security team

---
`+"```"+`

### 2. Narrative Background (.context/narrative-background.md)
**Purpose**: Detailed context and explanations (NOT uploaded to Tugboat)
**Contents**:
- Detailed analysis and reasoning
- Executive summaries
- Compliance interpretations
- Background information for internal use

## Workflow

### Step 1: Review Assembly Prompt
Read .context/assembly-prompt.md to understand:
- What evidence is needed
- Which controls it satisfies
- What policies/sources are relevant
- What tools can help

### Step 2: Breakdown Collection Tasks
Analyze task description and guidance to identify discrete collection tasks:
- What specific items need to be verified?
- What documentation needs to be reviewed?
- What technical evidence needs to be collected?

### Step 3: Collect Tool Data & Sources
Run applicable tools and gather source materials:
`+"```bash\n"+`
grctool tool <tool-name> --repository <repo> > .context/tool_outputs/<tool-name>.json
`+"```\n"+`

### Step 4: Generate Simple Evidence File
Create %s_Evidence.md (root directory) with:
- Collection tasks derived from description/guidance
- Evidence snippets with inline quotes
- Source file references with relative paths and dates
- Copy all referenced source files flat to root directory

### Step 5: Generate Narrative Background
Create .context/narrative-background.md with:
- Detailed analysis and interpretation
- Executive summary
- Compliance reasoning
- Additional context

## Source File Handling

**IMPORTANT**: Copy all referenced source files to root directory (flat, no subdirectories):
- Policy documents → POL-XXXX-name.md (from docs/policies/markdown/)
- Control files → AC1-778771.md (from docs/controls/markdown/)
- Infrastructure configs → main.tf, deploy.yml (original source files)
- Application configs → config.yaml, .env.example

**File Selection Rules**:
- ✅ **DO include**: Markdown documentation (.md), infrastructure source files (.tf, .yml, .yaml, .toml, .hcl)
- ❌ **NEVER include**: JSON files (.json) - not auditor-friendly
- 📊 **Tool outputs**: Analyze JSON internally, summarize findings in narrative, DON'T copy JSON to root

**Path References**: Use relative paths from data directory:
- ✅ `+"`"+`docs/policies/markdown/POL-0001-access-control.md`+"`"+`
- ✅ `+"`"+`docs/controls/markdown/AC1-778771.md`+"`"+`
- ❌ `+"`"+`/Users/erik/Projects/7thsense-ops/isms/docs/policies/POL-0001-access-control.md`+"`"+`

## Expected Outputs

**Root Directory**: Ready for Tugboat upload
- %s_Evidence.md (simple, task-focused)
- POL-XXXX-*.md (policy source files in markdown)
- AC*-*.md (control source files in markdown)
- Infrastructure/config files (.tf, .yml, .yaml - original source files only, NO JSON)

**.context/**: Internal context only
- narrative-background.md (detailed analysis)
- assembly-prompt.md
- claude-instructions.md
- evidence-template.md

---

**Need help?** Review the assembly prompt first, then ask questions about available data sources!
`, task.ReferenceID, task.Name, task.ReferenceID, task.ReferenceID, task.ReferenceID, task.ReferenceID)
}

// isTugboatManagedTask checks if a task is managed by Tugboat (AEC enabled + Hybrid collection)
func isTugboatManagedTask(task *domain.EvidenceTask) bool {
	// Check if AEC (Automated Evidence Collection) is enabled
	if task.AecStatus != nil && task.AecStatus.Status == "enabled" {
		// Check if collection type is Hybrid (indicating Tugboat collects it)
		collectionType := task.GetCollectionType()
		if collectionType == "Hybrid" {
			return true
		}
	}
	return false
}

// displayTugboatManagedMessage shows guidance for Tugboat-managed evidence tasks
func displayTugboatManagedMessage(cmd *cobra.Command, task *domain.EvidenceTask) {
	cmd.Println()
	cmd.Println("═══════════════════════════════════════════════════════════════════")
	cmd.Printf("📋 Task: %s - %s\n", task.ReferenceID, task.Name)
	cmd.Println("═══════════════════════════════════════════════════════════════════")
	cmd.Println()
	cmd.Println("ℹ️  This evidence is collected directly in Tugboat Logic.")
	cmd.Println()
	cmd.Println("This task has Automated Evidence Collection (AEC) enabled and uses")
	cmd.Println("Tugboat's data collection system. Evidence should be provided through")
	cmd.Println("Tugboat's interface rather than using grctool automation.")
	cmd.Println()

	// Show what data is needed
	if task.Description != "" || task.Guidance != "" {
		cmd.Println("📝 What's needed:")
		cmd.Println()
		if task.Description != "" {
			cmd.Printf("   %s\n", task.Description)
			cmd.Println()
		}
		if task.Guidance != "" {
			cmd.Println("   Guidance:")
			cmd.Printf("   %s\n", task.Guidance)
			cmd.Println()
		}
	}

	cmd.Println("🔗 How to provide this evidence:")
	cmd.Println()
	cmd.Println("   1. Log into Tugboat Logic web interface")

	if task.TugboatURL != "" {
		cmd.Printf("   2. Navigate to this task: %s\n", task.TugboatURL)
	} else {
		cmd.Printf("   2. Navigate to evidence task %s\n", task.ReferenceID)
	}

	cmd.Println("   3. Use Tugboat's data upload or integration features to provide:")
	cmd.Println()

	// Category-specific guidance
	category := task.GetCategory()
	switch category {
	case "Personnel":
		cmd.Println("      • Upload CSV or Excel file with employee/contractor data")
		cmd.Println("      • Required fields: Name, Start Date, Title, Department")
		cmd.Println("      • Include termination dates for separated personnel")
		cmd.Println("      • Use your HRIS system export or generate from HR database")
	case "Process":
		cmd.Println("      • Upload policy documents, procedures, or reports")
		cmd.Println("      • Provide signed acknowledgments or approvals")
		cmd.Println("      • Include meeting minutes or review documentation")
	case "Infrastructure":
		cmd.Println("      • Upload system configuration files or screenshots")
		cmd.Println("      • Provide scan results or monitoring reports")
		cmd.Println("      • Include audit logs or access control lists")
	default:
		cmd.Println("      • Follow the task guidance above for required data format")
		cmd.Println("      • Upload supporting documents or evidence files")
		cmd.Println("      • Ensure all required fields are included")
	}

	cmd.Println()

	// Show related controls for context
	if len(task.RelatedControls) > 0 {
		cmd.Println("📊 Related controls:")
		cmd.Println()
		for i, ctrl := range task.RelatedControls {
			if i < 5 { // Limit to first 5 controls
				cmd.Printf("   • %s: %s\n", ctrl.ReferenceID, ctrl.Name)
			}
		}
		if len(task.RelatedControls) > 5 {
			cmd.Printf("   ... and %d more\n", len(task.RelatedControls)-5)
		}
		cmd.Println()
	}

	cmd.Println("═══════════════════════════════════════════════════════════════════")
	cmd.Println()
	cmd.Println("💡 Note: grctool automation is designed for infrastructure-sourced")
	cmd.Println("   evidence (GitHub, Terraform, docs). For HRIS and manual uploads,")
	cmd.Println("   use Tugboat Logic's web interface directly.")
	cmd.Println()
}

// generateAssemblyContext generates a comprehensive assembly context for evidence collection
func generateAssemblyContext(task *domain.EvidenceTask, window string, tools []string, cfg *config.Config, storage *storage.Storage) (*AssemblyContext, error) {
	// 1. Call prompt-assembler tool to get comprehensive prompt
	promptOutput, err := executePromptAssembler(task, cfg, storage)
	if err != nil {
		return nil, fmt.Errorf("prompt-assembler failed: %w", err)
	}

	// 2. Generate Claude-specific instructions
	claudeInstructions := generateClaudeInstructions(task, window)

	// 3. Select/generate evidence template based on task category
	evidenceTemplate := selectEvidenceTemplate(task)

	// 4. Identify applicable tools (from prompt or config)
	applicableTools := identifyApplicableToolsForAssembly(task, tools)

	return &AssemblyContext{
		Task:                task,
		Window:              window,
		ComprehensivePrompt: promptOutput.Prompt,
		ClaudeInstructions:  claudeInstructions,
		EvidenceTemplate:    evidenceTemplate,
		ApplicableTools:     applicableTools,
		ToolData:            make(map[string]interface{}),
	}, nil
}

// executePromptAssembler calls the prompt-assembler tool to generate a comprehensive prompt
func executePromptAssembler(task *domain.EvidenceTask, cfg *config.Config, storage *storage.Storage) (*PromptAssemblerOutput, error) {
	// Get prompt-assembler tool from registry
	tool, err := tools.GetTool("prompt-assembler")
	if err != nil {
		return nil, fmt.Errorf("prompt-assembler tool not found: %w", err)
	}

	// Prepare request parameters
	params := map[string]interface{}{
		"task_ref":         task.ReferenceID,
		"context_level":    "comprehensive", // Always comprehensive
		"include_examples": true,
		"output_format":    "markdown",
		"save_to_file":     true,
	}

	// Execute tool
	ctx := context.Background()
	result, _, err := tool.Execute(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute prompt-assembler: %w", err)
	}

	// Parse the JSON result
	var responseData struct {
		Success      bool   `json:"success"`
		PromptText   string `json:"prompt_text"`
		FilePath     string `json:"file_path"`
		PromptMetadata map[string]interface{} `json:"prompt_metadata"`
	}

	// Try to parse as JSON first
	if err := parseJSONResult(result, &responseData); err == nil && responseData.Success {
		// Extract prompt from parsed response
		output := &PromptAssemblerOutput{
			Prompt: responseData.PromptText,
			Data: map[string]interface{}{
				"file_path": responseData.FilePath,
				"metadata":  responseData.PromptMetadata,
			},
		}
		return output, nil
	}

	// Fallback: treat the entire result as the prompt text
	output := &PromptAssemblerOutput{
		Prompt: result,
		Data:   make(map[string]interface{}),
	}

	return output, nil
}

// identifyApplicableToolsForAssembly identifies applicable tools for evidence assembly
func identifyApplicableToolsForAssembly(task *domain.EvidenceTask, tools []string) []string {
	// If tools explicitly specified, use those
	if len(tools) > 0 {
		return tools
	}

	// Otherwise, infer from task metadata or description
	applicableTools := []string{}

	// Check task name and description for tool hints
	taskText := strings.ToLower(task.Name + " " + task.Description)

	// Infrastructure tools
	if strings.Contains(taskText, "terraform") || strings.Contains(taskText, "infrastructure") {
		applicableTools = append(applicableTools, "terraform-security-indexer", "terraform-security-analyzer")
	}

	// GitHub tools
	if strings.Contains(taskText, "github") || strings.Contains(taskText, "repository") || strings.Contains(taskText, "code review") {
		applicableTools = append(applicableTools, "github-permissions", "github-security-features", "github-workflow-analyzer")
	}

	// Google Workspace tools
	if strings.Contains(taskText, "google") || strings.Contains(taskText, "workspace") || strings.Contains(taskText, "drive") {
		applicableTools = append(applicableTools, "google-workspace")
	}

	// Documentation tools
	if strings.Contains(taskText, "documentation") || strings.Contains(taskText, "policy") {
		applicableTools = append(applicableTools, "docs-reader")
	}

	return applicableTools
}

// selectEvidenceTemplate selects an appropriate evidence template based on task category
func selectEvidenceTemplate(task *domain.EvidenceTask) string {
	// Get task category
	category := task.GetCategory()

	// Select template based on category
	switch category {
	case "Infrastructure":
		return generateInfrastructureTemplate()
	case "Personnel":
		return generatePersonnelTemplate()
	case "Process":
		return generateProcessTemplate()
	case "Compliance":
		return generateComplianceTemplate()
	case "Monitoring":
		return generateMonitoringTemplate()
	case "Data":
		return generateDataTemplate()
	default:
		return generateGenericTemplate()
	}
}

// Template generation functions (these provide structure guides for evidence)

func generateGenericTemplate() string {
	return `# Evidence Report Template

## Executive Summary
[Brief overview of compliance status and key findings]

## Control Mapping
[List of controls this evidence satisfies]

## Policy Foundations
[Related policies and governance documents]

## Technical Evidence
[Specific configurations, settings, and technical implementations]

## Compliance Analysis
[Analysis of how the evidence demonstrates compliance]

## Auditor Notes
[Additional context for auditors]

## Quality Assurance
[Verification steps and validation]
`
}

func generateInfrastructureTemplate() string {
	return `# Infrastructure Evidence Report

## Executive Summary
[Brief overview of infrastructure security posture]

## Control Mapping
[Controls satisfied by this infrastructure evidence]

## Policy Foundations
[Infrastructure security policies and standards]

## Infrastructure Configuration
### Cloud Resources
[Cloud infrastructure details]

### Network Security
[Network configurations and security controls]

### Access Controls
[IAM policies, roles, and permissions]

### Monitoring & Logging
[CloudTrail, logging configurations]

## Security Analysis
[Security posture assessment]

## Compliance Review
[Compliance status against requirements]

## Auditor Notes
[Additional context for infrastructure audit]
`
}

func generatePersonnelTemplate() string {
	return `# Personnel Evidence Report

## Executive Summary
[Overview of personnel security controls]

## Control Mapping
[Personnel-related controls]

## Policy Foundations
[HR policies, acceptable use policies]

## Personnel Security Controls
### Access Management
[User provisioning and deprovisioning]

### Training & Awareness
[Security training programs]

### Background Checks
[Background verification processes]

## Compliance Analysis
[Personnel control effectiveness]

## Auditor Notes
[Additional personnel security context]
`
}

func generateProcessTemplate() string {
	return `# Process Evidence Report

## Executive Summary
[Overview of process controls]

## Control Mapping
[Process-related controls]

## Policy Foundations
[Process policies and procedures]

## Process Documentation
### Standard Operating Procedures
[SOPs and process documentation]

### Change Management
[Change control processes]

### Incident Response
[Incident handling procedures]

## Compliance Analysis
[Process control effectiveness]

## Auditor Notes
[Additional process context]
`
}

func generateComplianceTemplate() string {
	return `# Compliance Evidence Report

## Executive Summary
[Overview of compliance program]

## Control Mapping
[Compliance controls]

## Policy Foundations
[Compliance policies and frameworks]

## Compliance Program
### Framework Alignment
[Framework mappings (SOC2, ISO27001, etc.)]

### Risk Management
[Risk assessment and treatment]

### Audit & Review
[Audit processes and findings]

## Compliance Analysis
[Compliance posture assessment]

## Auditor Notes
[Additional compliance context]
`
}

func generateMonitoringTemplate() string {
	return `# Monitoring Evidence Report

## Executive Summary
[Overview of monitoring capabilities]

## Control Mapping
[Monitoring-related controls]

## Policy Foundations
[Monitoring policies and standards]

## Monitoring Infrastructure
### Log Collection
[Logging systems and aggregation]

### Alerting & Detection
[Alert rules and detection mechanisms]

### Incident Detection
[Security monitoring and SIEM]

## Compliance Analysis
[Monitoring effectiveness]

## Auditor Notes
[Additional monitoring context]
`
}

func generateDataTemplate() string {
	return `# Data Security Evidence Report

## Executive Summary
[Overview of data security controls]

## Control Mapping
[Data security controls]

## Policy Foundations
[Data protection and privacy policies]

## Data Security Controls
### Data Classification
[Data classification scheme]

### Encryption
[Encryption at rest and in transit]

### Access Controls
[Data access restrictions]

### Data Lifecycle
[Data retention and disposal]

## Compliance Analysis
[Data security effectiveness]

## Auditor Notes
[Additional data security context]
`
}

// parseJSONResult attempts to parse a JSON string into the provided struct
func parseJSONResult(jsonStr string, target interface{}) error {
	return json.Unmarshal([]byte(jsonStr), target)
}

// ============================================================================
// Comprehensive Evidence Template System
// ============================================================================
// These comprehensive templates provide detailed, auditor-ready structure for
// evidence documents. They include all necessary sections for compliance review.

// getDefaultTemplate returns a comprehensive evidence template suitable for all evidence types
func getDefaultTemplate() string {
	return `# Evidence Report: {{TASK_REF}} - {{TASK_NAME}}

**Evidence Reference:** {{TASK_REF}}
**Tugboat ID:** {{TUGBOAT_ID}}
**Collection Window:** {{WINDOW}}
**Collection Date:** {{DATE}}
**Period Covered:** {{PERIOD}}

---

## Executive Summary

[Provide 3-5 paragraph summary of evidence, key findings, and compliance status]

---

## Control Mapping

This evidence supports the following controls:

| Control ID | Control Name | Framework | Compliance Status |
|------------|--------------|-----------|-------------------|
| {{CONTROL_ID}} | {{CONTROL_NAME}} | {{FRAMEWORK}} | ✅ Compliant |

---

## Policy Foundation

### Primary Policy: {{POLICY_REF}} - {{POLICY_NAME}}

**Relevant Policy Sections:**

[Include key policy excerpts that establish requirements]

---

## Evidence Collection Method

[Describe how evidence was collected - automated tools, manual review, etc.]

### Tools Used:
- {{TOOL_NAME}}: {{TOOL_PURPOSE}}

---

## Technical Evidence

[Present data/findings from tools and manual collection]

### {{SECTION_NAME}}

[Data, screenshots, configurations, etc.]

---

## Compliance Analysis

[Interpret the evidence in context of control requirements]

### Control Objective

[What the control requires]

### How This Evidence Addresses the Control

[Map evidence to requirements]

---

## Quality Assurance

**Review Checklist:**
- [ ] All required evidence collected
- [ ] Evidence dated within collection window
- [ ] Control mappings verified
- [ ] Technical accuracy confirmed
- [ ] Auditor-ready formatting

---

## Auditor Notes

### Control Operating Effectiveness

**Design:** [Effective/Ineffective + justification]
**Operating Effectiveness:** [Effective/Ineffective + justification]

---

**Evidence Collection:** {{COLLECTION_TYPE}}
**Next Collection Date:** {{NEXT_DATE}}
**Control Status:** ✅ Operating Effectively
`
}

// getInfrastructureTemplate returns a specialized template for infrastructure evidence
func getInfrastructureTemplate() string {
	return `# Infrastructure Evidence Report: {{TASK_REF}} - {{TASK_NAME}}

**Evidence Reference:** {{TASK_REF}}
**Tugboat ID:** {{TUGBOAT_ID}}
**Collection Window:** {{WINDOW}}
**Collection Date:** {{DATE}}
**Period Covered:** {{PERIOD}}

---

## Executive Summary

[Summary of infrastructure configuration and security controls found during this evidence collection period]

### Key Findings
- [Finding 1: Summary of main infrastructure security control]
- [Finding 2: Summary of configuration state]
- [Finding 3: Summary of compliance posture]

---

## Infrastructure Overview

### Architecture
[High-level architecture description of the infrastructure being evidenced]

### Components Analyzed
[List of infrastructure components reviewed during this collection period]

- Component 1: [Description]
- Component 2: [Description]
- Component 3: [Description]

---

## Control Mapping

This evidence supports the following infrastructure security controls:

| Control ID | Control Name | Framework | Compliance Status |
|------------|--------------|-----------|-------------------|
| {{CONTROL_ID}} | {{CONTROL_NAME}} | {{FRAMEWORK}} | ✅ Compliant |

---

## Policy Foundation

### Primary Policy: {{POLICY_REF}} - {{POLICY_NAME}}

**Relevant Policy Requirements:**

[Include key policy excerpts that establish infrastructure security requirements]

---

## Configuration Evidence

### Security Controls

**Firewall Configurations:**
[Firewall rules, network segmentation, zone configurations]

**Encryption Settings:**
[Encryption at rest, in transit, key management]

**Authentication & Authorization:**
[IAM configurations, role assignments, permission boundaries]

### Access Controls

**Network Access:**
[VPN, bastion hosts, security groups, NACLs]

**Administrative Access:**
[Privileged access management, MFA requirements, session recordings]

### Monitoring & Logging

**Log Collection:**
[Centralized logging, retention policies, log sources]

**Alerting:**
[Alert configurations, incident response triggers, escalation paths]

**Audit Trails:**
[Configuration change tracking, access logs, API activity]

---

## Terraform Analysis

### Resources Analyzed
[List of Terraform resources reviewed, organized by service/component]

### Security Findings

**Resource Type:** [e.g., aws_s3_bucket]
- **Total Count:** [Number of resources]
- **Security Configuration:** [Details of security settings]
- **Compliance Status:** [Assessment against requirements]

**Resource Type:** [e.g., aws_security_group]
- **Total Count:** [Number of resources]
- **Security Configuration:** [Details of security settings]
- **Compliance Status:** [Assessment against requirements]

---

## Infrastructure as Code

### Terraform Modules Used
[List of modules and their security configurations]

### Version Control
[Git repository, branch protection, code review process]

### Deployment Controls
[CI/CD pipeline security, approval processes, rollback procedures]

---

## Compliance Assessment

### Control Objective
[What the infrastructure control requires]

### Evidence Analysis
[How the infrastructure configuration meets the control requirements]

### Gaps Identified
[Any gaps or areas needing remediation]

### Remediation Status
[Status of any identified gaps]

---

## Quality Assurance

**Technical Review Checklist:**
- [ ] Infrastructure configurations reviewed
- [ ] Security controls verified operational
- [ ] Terraform state validated
- [ ] Access controls audited
- [ ] Monitoring/logging confirmed active
- [ ] Documentation complete

---

## Auditor Notes

### Infrastructure Security Assessment

**Design Effectiveness:** [Effective/Ineffective + justification]
- Security controls are [properly/improperly] designed to meet requirements
- Infrastructure architecture [does/does not] implement defense in depth
- Configurations [align/do not align] with industry best practices

**Operating Effectiveness:** [Effective/Ineffective + justification]
- Security controls are [consistently/inconsistently] applied
- Monitoring and logging [are/are not] capturing required events
- Access controls [are/are not] functioning as designed

---

**Evidence Collection:** {{COLLECTION_TYPE}}
**Next Collection Date:** {{NEXT_DATE}}
**Infrastructure Status:** ✅ Operating Effectively
`
}

// getPersonnelTemplate returns a specialized template for personnel/HR evidence
func getPersonnelTemplate() string {
	return `# Personnel Evidence Report: {{TASK_REF}} - {{TASK_NAME}}

**Evidence Reference:** {{TASK_REF}}
**Tugboat ID:** {{TUGBOAT_ID}}
**Collection Window:** {{WINDOW}}
**Collection Date:** {{DATE}}
**Period Covered:** {{PERIOD}}

---

## Executive Summary

[Summary of personnel-related evidence collected during this period]

### Key Findings
- [Finding 1: Personnel count and role distribution]
- [Finding 2: Training completion status]
- [Finding 3: Access management compliance]

---

## Personnel Scope

### Roles Covered
[List of roles/positions analyzed during this collection period]

- **Role 1:** [Count] employees
- **Role 2:** [Count] employees
- **Role 3:** [Count] contractors

### Time Period
**Collection Window:** {{WINDOW}}
**Personnel Count:** [Total number of personnel in scope]

---

## Control Mapping

This evidence supports the following personnel security controls:

| Control ID | Control Name | Framework | Compliance Status |
|------------|--------------|-----------|-------------------|
| {{CONTROL_ID}} | {{CONTROL_NAME}} | {{FRAMEWORK}} | ✅ Compliant |

---

## Policy Foundation

### Primary Policy: {{POLICY_REF}} - {{POLICY_NAME}}

**Relevant Policy Requirements:**

[Include key policy excerpts that establish personnel security requirements]

---

## Onboarding Evidence

### New Hires
**Period:** {{WINDOW}}
**Total New Hires:** [Count]

**Onboarding Process Completion:**
- Background checks: [Count] completed
- Security training: [Count] completed
- Access provisioning: [Count] completed
- Policy acknowledgments: [Count] signed

### Documentation
[List of onboarding documentation reviewed]

---

## Training Evidence

### Security Awareness Training

**Training Programs:**
- **Program 1:** [Name and description]
  - Assigned: [Count] personnel
  - Completed: [Count] personnel
  - Completion Rate: [Percentage]

- **Program 2:** [Name and description]
  - Assigned: [Count] personnel
  - Completed: [Count] personnel
  - Completion Rate: [Percentage]

### Completion Records
[Summary of training completion data for the period]

### Role-Specific Training
[Specialized training for specific roles]

---

## Access Control Evidence

### User Access

**Active Users:**
- **Production Access:** [Count] users
- **Administrative Access:** [Count] users
- **Read-Only Access:** [Count] users

**Access Reviews:**
- **Review Date:** [Date]
- **Accounts Reviewed:** [Count]
- **Accounts Modified:** [Count]
- **Accounts Terminated:** [Count]

### Permissions

**Permission Assignments:**
[Summary of permission grants and role assignments]

**Privileged Access:**
[List of users with privileged access and justification]

**Multi-Factor Authentication:**
- Users with MFA enabled: [Count/Percentage]
- Users pending MFA: [Count]

---

## Offboarding Evidence

### Terminated Personnel
**Period:** {{WINDOW}}
**Total Terminations:** [Count]

**Offboarding Process Completion:**
- Access revocation: [Count] completed
- Asset return: [Count] completed
- Exit interviews: [Count] completed
- Final acknowledgments: [Count] signed

### Access Termination Timeline
[Summary of access termination timing relative to last day of employment]

---

## Background Checks

### Background Screening Results
**Period:** {{WINDOW}}
**Total Screenings:** [Count]

**Screening Components:**
- Criminal history checks: [Count]
- Employment verification: [Count]
- Education verification: [Count]
- Reference checks: [Count]

**Results:** [Summary of findings]

---

## Compliance Assessment

### Control Objective
[What the personnel control requires]

### Evidence Analysis
[How personnel processes meet the control requirements]

### Compliance Metrics
- Onboarding compliance rate: [Percentage]
- Training completion rate: [Percentage]
- Access review completion: [Percentage]
- Offboarding compliance rate: [Percentage]

---

## Quality Assurance

**Personnel Review Checklist:**
- [ ] All new hires properly onboarded
- [ ] Training requirements met
- [ ] Access controls reviewed and validated
- [ ] Terminations processed completely
- [ ] Background checks completed
- [ ] Documentation complete and signed

---

## Auditor Notes

### Personnel Security Assessment

**Design Effectiveness:** [Effective/Ineffective + justification]
- Personnel security processes are [properly/improperly] designed
- Onboarding/offboarding procedures [are/are not] comprehensive
- Training programs [do/do not] address security awareness

**Operating Effectiveness:** [Effective/Ineffective + justification]
- Personnel processes [are/are not] consistently followed
- Access controls [are/are not] properly maintained
- Documentation [is/is not] complete and timely

---

**Evidence Collection:** {{COLLECTION_TYPE}}
**Next Collection Date:** {{NEXT_DATE}}
**Personnel Security Status:** ✅ Operating Effectively
`
}

// getProcessTemplate returns a specialized template for process/procedure evidence
func getProcessTemplate() string {
	return `# Process Evidence Report: {{TASK_REF}} - {{TASK_NAME}}

**Evidence Reference:** {{TASK_REF}}
**Tugboat ID:** {{TUGBOAT_ID}}
**Collection Window:** {{WINDOW}}
**Collection Date:** {{DATE}}
**Period Covered:** {{PERIOD}}

---

## Executive Summary

[Summary of process evidence collected during this period]

### Key Findings
- [Finding 1: Process execution status]
- [Finding 2: Compliance with documented procedures]
- [Finding 3: Process effectiveness assessment]

---

## Process Overview

### Process Description
[Detailed description of the process being evidenced]

**Process Owner:** [Name/Role]
**Process Stakeholders:** [List of stakeholders]

### Frequency
**Execution Schedule:** [How often process executes]
**Collection Window:** {{WINDOW}}
**Total Executions This Period:** [Count]

---

## Control Mapping

This evidence supports the following process controls:

| Control ID | Control Name | Framework | Compliance Status |
|------------|--------------|-----------|-------------------|
| {{CONTROL_ID}} | {{CONTROL_NAME}} | {{FRAMEWORK}} | ✅ Compliant |

---

## Policy Foundation

### Primary Policy: {{POLICY_REF}} - {{POLICY_NAME}}

**Relevant Policy Requirements:**

[Include key policy excerpts that establish process requirements]

---

## Process Documentation

### Procedures
[Links to procedures/documentation that define this process]

- **Procedure 1:** [Name and location]
  - Version: [Version number]
  - Last Updated: [Date]
  - Approved By: [Name]

- **Procedure 2:** [Name and location]
  - Version: [Version number]
  - Last Updated: [Date]
  - Approved By: [Name]

### Workflow
[Process workflow description with key steps]

1. **Step 1:** [Description]
   - Responsible: [Role]
   - Inputs: [Required inputs]
   - Outputs: [Expected outputs]

2. **Step 2:** [Description]
   - Responsible: [Role]
   - Inputs: [Required inputs]
   - Outputs: [Expected outputs]

3. **Step 3:** [Description]
   - Responsible: [Role]
   - Inputs: [Required inputs]
   - Outputs: [Expected outputs]

---

## Process Execution Evidence

### Execution Records
[Evidence of process execution during the collection period]

**Period:** {{WINDOW}}
**Total Executions:** [Count]
**Successful Executions:** [Count]
**Failed/Incomplete Executions:** [Count]

### Sample Executions

**Execution 1:**
- Date: [Date]
- Executor: [Name]
- Result: [Success/Failure]
- Notes: [Any relevant notes]

**Execution 2:**
- Date: [Date]
- Executor: [Name]
- Result: [Success/Failure]
- Notes: [Any relevant notes]

**Execution 3:**
- Date: [Date]
- Executor: [Name]
- Result: [Success/Failure]
- Notes: [Any relevant notes]

---

## Process Artifacts

### Input Documentation
[Documentation/data used as inputs to the process]

### Output Documentation
[Documentation/data produced by the process]

### Supporting Evidence
[Additional artifacts that support process execution]

- Meeting minutes
- Approval emails
- Decision logs
- Change records
- Incident tickets

---

## Process Metrics

### Performance Indicators
[Metrics that demonstrate process effectiveness]

**Metric 1:** [Name]
- Target: [Target value]
- Actual: [Actual value]
- Status: [Met/Not Met]

**Metric 2:** [Name]
- Target: [Target value]
- Actual: [Actual value]
- Status: [Met/Not Met]

### Trend Analysis
[Analysis of process performance over time]

---

## Change Management

### Process Changes
[Any changes to the process during this period]

**Change 1:**
- Date: [Date]
- Description: [Change description]
- Reason: [Justification]
- Approved By: [Name]

---

## Compliance Assessment

### Control Objective
[What the process control requires]

### Evidence Analysis
[How process execution meets the control requirements]

**Process Adherence:**
[Assessment of adherence to documented procedures]

**Exception Handling:**
[How exceptions were identified and handled]

**Continuous Improvement:**
[Evidence of process improvement activities]

---

## Quality Assurance

**Process Review Checklist:**
- [ ] Process documentation current and approved
- [ ] Process executed per documented procedures
- [ ] Required approvals obtained
- [ ] Artifacts collected and retained
- [ ] Metrics tracked and reviewed
- [ ] Exceptions properly documented and resolved

---

## Auditor Notes

### Process Control Assessment

**Design Effectiveness:** [Effective/Ineffective + justification]
- Process design [is/is not] adequate to meet control objectives
- Procedures [are/are not] comprehensive and clear
- Roles and responsibilities [are/are not] properly defined

**Operating Effectiveness:** [Effective/Ineffective + justification]
- Process [is/is not] consistently executed as documented
- Required approvals [are/are not] obtained
- Documentation [is/is not] complete and timely
- Metrics [are/are not] tracked and reviewed

---

**Evidence Collection:** {{COLLECTION_TYPE}}
**Next Collection Date:** {{NEXT_DATE}}
**Process Status:** ✅ Operating Effectively
`
}

// getComplianceTemplate returns a template for compliance-specific evidence
func getComplianceTemplate() string {
	// For now, use the default template
	// Can be customized later with compliance-specific sections
	return getDefaultTemplate()
}

// getMonitoringTemplate returns a template for monitoring/logging evidence
func getMonitoringTemplate() string {
	// For now, use the default template
	// Can be customized later with monitoring-specific sections
	return getDefaultTemplate()
}

// getDataTemplate returns a template for data management evidence
func getDataTemplate() string {
	// For now, use the default template
	// Can be customized later with data-specific sections
	return getDefaultTemplate()
}

// applyTemplateVariables replaces template placeholders with actual values
func applyTemplateVariables(template string, task *domain.EvidenceTask, window string) string {
	replacements := map[string]string{
		"{{TASK_REF}}":   task.ReferenceID,
		"{{TASK_NAME}}":  task.Name,
		"{{TUGBOAT_ID}}": fmt.Sprintf("%d", task.ID),
		"{{WINDOW}}":     window,
		"{{DATE}}":       time.Now().Format("2006-01-02"),
		"{{PERIOD}}":     calculatePeriod(window),
	}

	result := template
	for key, value := range replacements {
		result = strings.ReplaceAll(result, key, value)
	}

	return result
}

// calculatePeriod converts a window identifier into a human-readable period description
func calculatePeriod(window string) string {
	// Parse window like "2025-Q4" and return period description
	if strings.Contains(window, "Q") {
		return fmt.Sprintf("Quarterly period %s", window)
	}
	return window
}

// saveAssemblyContext persists all assembly materials to disk
func saveAssemblyContext(task *domain.EvidenceTask, window string, ctx *AssemblyContext, dataDir string) (*AssemblyPaths, error) {
	// Determine evidence directory path
	evidenceDir := filepath.Join(dataDir, "evidence")
	taskDirName := naming.GetEvidenceTaskDirName(task.ReferenceID, task.Name)
	windowDir := filepath.Join(evidenceDir, taskDirName, window)
	contextDir := filepath.Join(windowDir, ".context")

	// Create directories (hybrid approach - working files go to root)
	if err := os.MkdirAll(contextDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create context directory: %w", err)
	}
	// Ensure window root exists for evidence files
	if err := os.MkdirAll(windowDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create window directory: %w", err)
	}

	assemblyPaths := &AssemblyPaths{
		PromptFile:       filepath.Join(contextDir, "assembly-prompt.md"),
		InstructionsFile: filepath.Join(contextDir, "claude-instructions.md"),
		TemplateFile:     filepath.Join(contextDir, "evidence-template.md"),
		ToolDataDir:      filepath.Join(contextDir, "tool_outputs"),
	}

	// Save assembly prompt
	if err := os.WriteFile(assemblyPaths.PromptFile, []byte(ctx.ComprehensivePrompt), 0644); err != nil {
		return nil, fmt.Errorf("failed to save assembly prompt: %w", err)
	}

	// Save Claude instructions
	if err := os.WriteFile(assemblyPaths.InstructionsFile, []byte(ctx.ClaudeInstructions), 0644); err != nil {
		return nil, fmt.Errorf("failed to save instructions: %w", err)
	}

	// Save evidence template (with variables applied)
	templateContent := applyTemplateVariables(ctx.EvidenceTemplate, task, window)
	if err := os.WriteFile(assemblyPaths.TemplateFile, []byte(templateContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to save template: %w", err)
	}

	// Create tool outputs directory
	if err := os.MkdirAll(assemblyPaths.ToolDataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create tool outputs directory: %w", err)
	}

	return assemblyPaths, nil
}

// ============================================================================
// Evidence Review Display Functions
// ============================================================================

func displayReviewHeader(cmd *cobra.Command, task *domain.EvidenceTask, window string) {
	separator := strings.Repeat("=", 67)
	cmd.Println(separator)
	cmd.Printf("Evidence Review: %s (%s)\n", task.ReferenceID, window)
	cmd.Println(task.Name)
	cmd.Println(separator)
	cmd.Println()
}

func displayEvidenceFiles(cmd *cobra.Command, files []models.EvidenceFileRef, err error) {
	cmd.Println("📁 EVIDENCE FILES (Root Directory)")
	cmd.Println(strings.Repeat("─", 67))

	if err != nil || len(files) == 0 {
		cmd.Println("  ⚠️  No evidence files found in root directory")
		cmd.Println("     Run 'grctool evidence generate' first")
		cmd.Println()
		return
	}

	var totalSize int64
	for _, file := range files {
		// Format size
		sizeStr := formatFileSize(file.SizeBytes)

		// Display file
		cmd.Printf("  ✓ %-40s (%8s)\n",
			truncateFileName(file.Filename, 40),
			sizeStr)
		totalSize += file.SizeBytes
	}

	cmd.Printf("\nTotal: %d files (%s)\n\n", len(files), formatFileSize(totalSize))
}

func displayValidationStatus(cmd *cobra.Command, result *models.ValidationResult, err error) {
	cmd.Println("📊 EVALUATION STATUS")
	cmd.Println(strings.Repeat("─", 67))

	if err != nil || result == nil {
		cmd.Println("  ⚠️  No evaluation results found")
		cmd.Println("     Run 'grctool evidence evaluate' for detailed scoring")
		cmd.Println()
		return
	}

	// Overall score and status
	statusIcon := getStatusIcon(result.Status)
	cmd.Printf("Overall Score: %.1f/100 %s %s\n\n",
		result.CompletenessScore,
		statusIcon,
		strings.ToUpper(result.Status))

	// Dimension scores (if available from checks)
	cmd.Println("Dimension Scores:")
	displayDimensionScores(cmd, result)

	// Issues summary
	if len(result.Errors) > 0 || len(result.WarningsList) > 0 {
		cmd.Printf("\nIssues Found: %d errors, %d warnings\n",
			len(result.Errors), len(result.WarningsList))

		// Display errors
		for _, err := range result.Errors {
			cmd.Printf("  ❌ [%s] %s\n", strings.ToUpper(err.Severity), err.Message)
		}

		// Display warnings
		for _, warn := range result.WarningsList {
			cmd.Printf("  ⚠️  [%s] %s\n", strings.ToUpper(warn.Severity), warn.Message)
		}
	} else {
		cmd.Println("\nNo issues found ✓")
	}

	cmd.Println()
}

func displayDimensionScores(cmd *cobra.Command, result *models.ValidationResult) {
	// Parse dimension scores from checks if available
	dimensions := make(map[string]float64)
	dimensionMax := make(map[string]float64)

	// Default dimensions
	dimensionMax["Completeness"] = 30.0
	dimensionMax["Requirements Match"] = 30.0
	dimensionMax["Quality"] = 20.0
	dimensionMax["Control Alignment"] = 20.0

	// Try to extract from checks
	for _, check := range result.Checks {
		// Parse check name for dimension
		checkLower := strings.ToLower(check.Name)
		if strings.Contains(checkLower, "completeness") {
			if check.Status == "passed" {
				dimensions["Completeness"] += 10.0
			}
		} else if strings.Contains(checkLower, "requirement") {
			if check.Status == "passed" {
				dimensions["Requirements Match"] += 10.0
			}
		} else if strings.Contains(checkLower, "quality") || strings.Contains(checkLower, "format") {
			if check.Status == "passed" {
				dimensions["Quality"] += 10.0
			}
		} else if strings.Contains(checkLower, "control") {
			if check.Status == "passed" {
				dimensions["Control Alignment"] += 10.0
			}
		}
	}

	// If no checks, estimate from overall score
	if len(dimensions) == 0 && result.CompletenessScore > 0 {
		// Distribute overall score proportionally
		dimensions["Completeness"] = result.CompletenessScore * 0.30
		dimensions["Requirements Match"] = result.CompletenessScore * 0.30
		dimensions["Quality"] = result.CompletenessScore * 0.20
		dimensions["Control Alignment"] = result.CompletenessScore * 0.20
	}

	// Display dimensions
	for _, dimName := range []string{"Completeness", "Requirements Match", "Quality", "Control Alignment"} {
		score := dimensions[dimName]
		maxScore := dimensionMax[dimName]
		percentage := 0.0
		if maxScore > 0 {
			percentage = (score / maxScore) * 100
		}

		status := getScoreStatus(percentage)
		cmd.Printf("  %-20s %5.1f/%.0f  (%3.0f%%)  %s\n",
			dimName,
			score,
			maxScore,
			percentage,
			status)
	}
}

func displayRequirementsChecklist(cmd *cobra.Command, task *domain.EvidenceTask, hasFiles bool) {
	cmd.Println("📋 REQUIREMENTS CHECKLIST")
	cmd.Println(strings.Repeat("─", 67))
	cmd.Println("This evidence should demonstrate:")

	// Parse requirements from task description and guidance
	requirements := extractRequirements(task)

	if len(requirements) == 0 {
		cmd.Println("  • Compliance with related controls")
		cmd.Println("  • Evidence of implemented security measures")
		if hasFiles {
			cmd.Println("  ✓ Evidence files present")
		} else {
			cmd.Println("  ⚠️  No evidence files found")
		}
	} else {
		for _, req := range requirements {
			icon := "⚠️ "
			if hasFiles {
				icon = "✓"
			}
			cmd.Printf("  %s %s\n", icon, req)
		}
	}

	cmd.Println()
}

func displayControlAlignment(cmd *cobra.Command, task *domain.EvidenceTask, storage *storage.Storage) {
	cmd.Println("🎯 CONTROL ALIGNMENT")
	cmd.Println(strings.Repeat("─", 67))

	if len(task.Controls) == 0 {
		cmd.Println("  No controls mapped to this evidence task")
		cmd.Println()
		return
	}

	cmd.Println("Supports controls:")

	// Load and display related controls
	displayCount := len(task.Controls)
	if displayCount > 5 {
		displayCount = 5
	}

	for i := 0; i < displayCount; i++ {
		controlID := task.Controls[i]
		control, err := storage.GetControl(controlID)
		if err == nil {
			cmd.Printf("  • %s - %s\n", control.ReferenceID, control.Name)
		} else {
			cmd.Printf("  • %s\n", controlID)
		}
	}

	if len(task.Controls) > 5 {
		cmd.Printf("  ... and %d more\n", len(task.Controls)-5)
	}

	cmd.Println()
}

func displaySubmissionRecommendation(cmd *cobra.Command, task *domain.EvidenceTask, window string, hasFiles bool, hasValidation bool, result *models.ValidationResult, alreadySubmitted bool) {
	// Check if ready
	isReady := hasFiles && (!hasValidation || (result != nil && result.ReadyForSubmission))

	if alreadySubmitted {
		cmd.Println("📦 STATUS: ALREADY SUBMITTED")
		cmd.Println(strings.Repeat("─", 67))
		cmd.Println("Evidence for this window has already been submitted to Tugboat.")
		cmd.Println("Files are in .submitted/ folder.")
		cmd.Println()
		cmd.Println("To resubmit:")
		cmd.Println("  1. Move files from .submitted/ back to root directory")
		cmd.Println("  2. Run: grctool evidence submit " + task.ReferenceID + " --window " + window)
		cmd.Println(strings.Repeat("=", 67))
		return
	}

	if isReady {
		cmd.Println("✅ RECOMMENDATION: READY TO SUBMIT")
	} else {
		cmd.Println("⚠️  RECOMMENDATION: NOT READY FOR SUBMISSION")
	}
	cmd.Println(strings.Repeat("─", 67))

	if isReady {
		cmd.Println("This evidence is ready for submission to Tugboat Logic.")
		cmd.Println()
		cmd.Println("Reasons:")
		cmd.Println("  ✓ All required files present")
		if hasValidation && result != nil {
			cmd.Printf("  ✓ Meets quality threshold (%.0f/100)\n", result.CompletenessScore)
		}
		cmd.Println("  ✓ Addresses key requirements")
		cmd.Println()
		cmd.Println("Next steps:")
		cmd.Println("  1. Review files one more time if desired")
		cmd.Printf("  2. Run: grctool evidence submit %s --window %s\n", task.ReferenceID, window)
		cmd.Println("  3. Files will be automatically moved to .submitted/ after upload")
	} else {
		cmd.Println("This evidence needs additional work before submission.")
		cmd.Println()
		cmd.Println("Reasons:")
		if !hasFiles {
			cmd.Println("  ❌ No evidence files found")
		}
		if hasValidation && result != nil && !result.ReadyForSubmission {
			cmd.Printf("  ❌ Does not meet quality threshold (%.0f/100 < 70)\n", result.CompletenessScore)
			if len(result.Errors) > 0 {
				cmd.Printf("  ❌ %d error(s) found\n", len(result.Errors))
			}
		}
		cmd.Println()
		cmd.Println("Next steps:")
		if !hasFiles {
			cmd.Printf("  1. Run: grctool evidence generate %s --window %s\n", task.ReferenceID, window)
		} else {
			cmd.Println("  1. Address validation errors")
			cmd.Printf("  2. Run: grctool evidence evaluate %s --window %s\n", task.ReferenceID, window)
			cmd.Println("  3. Review results and resubmit")
		}
	}

	cmd.Println(strings.Repeat("=", 67))
}

// Helper functions for formatting

func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func getStatusIcon(status string) string {
	switch strings.ToLower(status) {
	case "passed", "pass":
		return "✓"
	case "failed", "fail":
		return "✗"
	case "warning":
		return "⚠️"
	default:
		return "○"
	}
}

func getScoreStatus(percentage float64) string {
	if percentage >= 90 {
		return "✓ pass"
	} else if percentage >= 70 {
		return "✓ pass"
	} else if percentage >= 50 {
		return "⚠ warning"
	}
	return "✗ fail"
}

func truncateFileName(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func extractRequirements(task *domain.EvidenceTask) []string {
	requirements := []string{}

	// Parse from description
	desc := strings.ToLower(task.Description + " " + task.Guidance)

	// Common requirement patterns
	if strings.Contains(desc, "access") {
		requirements = append(requirements, "Access control documentation")
	}
	if strings.Contains(desc, "approval") {
		requirements = append(requirements, "Approval workflow evidence")
	}
	if strings.Contains(desc, "configuration") {
		requirements = append(requirements, "System configuration details")
	}
	if strings.Contains(desc, "log") {
		requirements = append(requirements, "Audit log examples")
	}
	if strings.Contains(desc, "policy") {
		requirements = append(requirements, "Policy documentation")
	}
	if strings.Contains(desc, "procedure") {
		requirements = append(requirements, "Procedural documentation")
	}
	if strings.Contains(desc, "training") {
		requirements = append(requirements, "Training records")
	}
	if strings.Contains(desc, "review") {
		requirements = append(requirements, "Review documentation")
	}

	return requirements
}
