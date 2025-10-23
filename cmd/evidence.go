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
	"fmt"
	"os"
	"path/filepath"
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
	"github.com/grctool/grctool/internal/storage"
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

func init() {
	rootCmd.AddCommand(evidenceCmd)

	evidenceCmd.AddCommand(evidenceListCmd)
	evidenceCmd.AddCommand(evidenceViewCmd)
	evidenceCmd.AddCommand(evidenceAnalyzeCmd)
	evidenceCmd.AddCommand(evidenceMapCmd)
	evidenceCmd.AddCommand(evidenceGenerateCmd)
	evidenceCmd.AddCommand(evidenceReviewCmd)
	evidenceCmd.AddCommand(evidenceSubmitCmd)

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

	// Initialize service
	evidenceService, err := initializeEvidenceService()
	if err != nil {
		return err
	}

	taskID, err := evidenceService.ResolveTaskID(ctx, args[0])
	if err != nil {
		return err
	}

	// Implementation delegated to service layer
	cmd.Printf("Submitting evidence for task %d\n", taskID)
	cmd.Println("Evidence submission functionality has been moved to the service layer.")
	cmd.Printf("Task %d submission process would be handled by the evidence service.\n", taskID)

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
