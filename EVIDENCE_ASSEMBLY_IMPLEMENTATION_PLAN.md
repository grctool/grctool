# Evidence Assembly Integration - Implementation Plan

**Date**: 2025-10-27
**Status**: Ready for Implementation
**Objective**: Integrate prompt-assembler and evidence-generator into bulk evidence generation workflow

---

## Problem Statement

**Current behavior** (`grctool evidence generate --all`):
- Creates minimal 43-line `.context/generation-context.md` files
- No comprehensive prompts for Claude
- No integration with proven assembly workflow
- Claude gets insufficient context to write good evidence reports

**Proven workflow that WORKS**:
- `prompt-assembler` creates comprehensive prompts with controls, policies, examples
- `evidence-generator` coordinates tools and creates structured reports
- Results in 500+ line comprehensive evidence (like ET-0008)

**Gap**: Bulk generation doesn't use the proven assembly workflow

---

## Solution Overview

**Make assembly prompts the DEFAULT** - No backward compatibility needed.

When user runs:
```bash
grctool evidence generate --all --window 2025-Q4
```

**New behavior**:
1. ✅ Load all pending tasks
2. ✅ For each task, generate COMPREHENSIVE context using `prompt-assembler`
3. ✅ Create Claude-ready assembly instructions
4. ✅ Optionally run tools and collect data (with flag)
5. ✅ Prepare evidence-generator workflow materials

---

## Architecture Changes

### Current Flow
```
grctool evidence generate ET-XXXX
    └─> generateEvidenceContext()
        └─> formatContextAsMarkdown()
            └─> saveEvidenceContext()
                └─> .context/generation-context.md (minimal)
```

### New Flow
```
grctool evidence generate ET-XXXX
    └─> generateEvidenceContext()
        └─> callPromptAssembler()  # NEW
            └─> Run: grctool tool prompt-assembler --task-ref ET-XXXX
            └─> Get comprehensive prompt
        └─> createClaudeInstructions()  # NEW
        └─> saveEvidenceAssembly()  # NEW
            ├─> .context/assembly-prompt.md (comprehensive)
            ├─> .context/claude-instructions.md
            └─> .context/evidence-template.md
```

---

## Implementation Tasks

### Phase 1: Core Integration

#### Task 1.1: Modify Evidence Context Generation
**File**: `cmd/evidence.go`

**Function to modify**: `processSingleTaskGeneration()`

**Changes**:
```go
func processSingleTaskGeneration(cmd *cobra.Command, taskRef string, window string, contextOnly bool, options evidence.BulkGenerationOptions) error {
    // ... existing code to load task ...

    // OLD: Generate basic context
    // context, err := generateEvidenceContext(task, window, options.Tools, cfg, storage)

    // NEW: Generate comprehensive assembly context
    assemblyContext, err := generateAssemblyContext(task, window, options.Tools, cfg, storage)
    if err != nil {
        return fmt.Errorf("failed to generate assembly context: %w", err)
    }

    // Save comprehensive assembly materials
    assemblyPaths, err := saveAssemblyContext(task, window, assemblyContext, cfg.Storage.DataDir)
    if err != nil {
        return fmt.Errorf("failed to save assembly context: %w", err)
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
```

#### Task 1.2: Create New Assembly Context Generator
**File**: `cmd/evidence.go` (add new function)

**New function**:
```go
// AssemblyContext holds all materials for evidence assembly
type AssemblyContext struct {
    Task              *domain.EvidenceTask
    Window            string
    ComprehensivePrompt string  // From prompt-assembler
    ClaudeInstructions  string  // How to use materials
    EvidenceTemplate    string  // Structure guide
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
    applicableTools := identifyApplicableTools(task, tools)

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
```

#### Task 1.3: Execute Prompt Assembler Programmatically
**File**: `cmd/evidence.go` (add new function)

```go
func executePromptAssembler(task *domain.EvidenceTask, cfg *config.Config, storage *storage.Storage) (*PromptAssemblerOutput, error) {
    // Get prompt-assembler tool from registry
    toolRegistry := tools.GetRegistry()
    promptTool := toolRegistry.GetTool("prompt-assembler")
    if promptTool == nil {
        return nil, fmt.Errorf("prompt-assembler tool not found")
    }

    // Prepare request
    request := &tools.PromptAssemblerRequest{
        TaskRef:         task.ReferenceID,
        ContextLevel:    "comprehensive",  // Always comprehensive
        IncludeExamples: true,
        OutputFormat:    "markdown",
    }

    // Execute tool
    ctx := context.Background()
    result, err := promptTool.Execute(ctx, request)
    if err != nil {
        return nil, err
    }

    // Extract prompt from result
    output := &PromptAssemblerOutput{
        Prompt: result.Data["prompt"].(string),
    }

    return output, nil
}

type PromptAssemblerOutput struct {
    Prompt string
}
```

#### Task 1.4: Generate Claude Instructions
**File**: `cmd/evidence.go` (add new function)

```go
func generateClaudeInstructions(task *domain.EvidenceTask, window string) string {
    return fmt.Sprintf(`# Claude Code Instructions: %s

## Your Mission

Help the user generate comprehensive evidence for **%s** (%s).

## What You Have

1. **Assembly Prompt** (assembly-prompt.md)
   - Comprehensive context from prompt-assembler
   - Related controls and policies
   - Example evidence structure
   - All requirements for this task

2. **Evidence Template** (evidence-template.md)
   - Pre-structured report outline
   - Section headers and prompts
   - Based on proven evidence patterns

3. **Tool Data** (tool_outputs/ directory, if available)
   - Pre-collected data from automated tools
   - Ready for synthesis into report

## Workflow

### Step 1: Review Assembly Prompt
Read assembly-prompt.md to understand:
- What evidence is needed
- Which controls it satisfies
- What policies are relevant
- What tools can help

### Step 2: Collect Tool Data (if not already collected)
Run applicable tools identified in the prompt:
` + "```bash\n" + `
grctool tool <tool-name> --repository <repo> > tool_outputs/<tool-name>.json
` + "```\n" + `

### Step 3: Use Evidence Generator
Synthesize tool outputs into comprehensive report:
` + "```bash\n" + `
grctool tool evidence-generator \
  --task-ref %s \
  --prompt-file .context/assembly-prompt.md \
  --tools terraform,github,docs \
  --format markdown \
  --output-dir .
` + "```\n" + `

### Step 4: Refine Report
- Review generated evidence
- Fill in analysis sections
- Add executive summary
- Ensure all requirements met

### Step 5: Save Evidence
` + "```bash\n" + `
grctool tool evidence-writer \
  --task-ref %s \
  --window %s \
  --title "Comprehensive Evidence Report" \
  --file evidence-report.md
` + "```\n" + `

## Expected Output

A comprehensive evidence document (500+ lines) with:
- Executive summary
- Control mapping
- Policy foundations
- Technical evidence/data
- Compliance review/analysis
- Auditor notes
- Quality assurance section

## Example

See ET-0008 Workstation Firewall evidence for reference structure.

---

**Need help?** Ask the user questions to understand what data they have available!
`, task.ReferenceID, task.ReferenceID, task.Name, task.ReferenceID, task.ReferenceID, window)
}
```

#### Task 1.5: Select Evidence Templates
**File**: `cmd/evidence.go` (add new function)

```go
func selectEvidenceTemplate(task *domain.EvidenceTask) string {
    // Determine template based on task category
    category := strings.ToLower(task.Category)

    templates := map[string]string{
        "infrastructure": getInfrastructureTemplate(),
        "personnel":      getPersonnelTemplate(),
        "process":        getProcessTemplate(),
        "compliance":     getComplianceTemplate(),
        "monitoring":     getMonitoringTemplate(),
        "data":           getDataTemplate(),
    }

    template, ok := templates[category]
    if !ok {
        // Default comprehensive template
        return getDefaultTemplate()
    }

    return template
}

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

// Add category-specific templates
func getInfrastructureTemplate() string {
    // Infrastructure-specific template with sections for:
    // - Architecture diagrams
    // - Configuration evidence
    // - Security controls
    // - Monitoring/logging
    return "..." // Full template
}

// ... more template functions ...
```

#### Task 1.6: Save Assembly Context
**File**: `cmd/evidence.go` (add new function)

```go
func saveAssemblyContext(task *domain.EvidenceTask, window string, ctx *AssemblyContext, dataDir string) (*AssemblyPaths, error) {
    // Determine evidence directory path
    paths := ctx.Task.GetPaths()
    evidenceDir := filepath.Join(dataDir, "evidence", paths.DataDir)
    windowDir := filepath.Join(evidenceDir, window)
    contextDir := filepath.Join(windowDir, ".context")

    // Create context directory
    if err := os.MkdirAll(contextDir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create context directory: %w", err)
    }

    paths := &AssemblyPaths{
        PromptFile:       filepath.Join(contextDir, "assembly-prompt.md"),
        InstructionsFile: filepath.Join(contextDir, "claude-instructions.md"),
        TemplateFile:     filepath.Join(contextDir, "evidence-template.md"),
        ToolDataDir:      filepath.Join(contextDir, "tool_outputs"),
    }

    // Save assembly prompt
    if err := os.WriteFile(paths.PromptFile, []byte(ctx.ComprehensivePrompt), 0644); err != nil {
        return nil, fmt.Errorf("failed to save assembly prompt: %w", err)
    }

    // Save Claude instructions
    if err := os.WriteFile(paths.InstructionsFile, []byte(ctx.ClaudeInstructions), 0644); err != nil {
        return nil, fmt.Errorf("failed to save instructions: %w", err)
    }

    // Save evidence template
    templateContent := applyTemplateVariables(ctx.EvidenceTemplate, task, window)
    if err := os.WriteFile(paths.TemplateFile, []byte(templateContent), 0644); err != nil {
        return nil, fmt.Errorf("failed to save template: %w", err)
    }

    // Create tool outputs directory
    if err := os.MkdirAll(paths.ToolDataDir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create tool outputs directory: %w", err)
    }

    return paths, nil
}

func applyTemplateVariables(template string, task *domain.EvidenceTask, window string) string {
    replacements := map[string]string{
        "{{TASK_REF}}":     task.ReferenceID,
        "{{TASK_NAME}}":    task.Name,
        "{{TUGBOAT_ID}}":   fmt.Sprintf("%d", task.TugboatID),
        "{{WINDOW}}":       window,
        "{{DATE}}":         time.Now().Format("2006-01-02"),
        "{{PERIOD}}":       calculatePeriod(window),
    }

    result := template
    for key, value := range replacements {
        result = strings.ReplaceAll(result, key, value)
    }

    return result
}
```

### Phase 2: Bulk Generation Integration

#### Task 2.1: Update Bulk Generation to Use Assembly
**File**: `cmd/evidence.go`

**Function to modify**: `processBulkEvidenceGeneration()`

**Changes**:
```go
func processBulkEvidenceGeneration(cmd *cobra.Command, evidenceService interface{}, options evidence.BulkGenerationOptions, ctx context.Context) error {
    // ... existing code to load pending tasks ...

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

    // Final summary
    cmd.Printf("\n" + strings.Repeat("=", 60) + "\n")
    cmd.Printf("Assembly Context Generation Complete\n")
    cmd.Printf("  ✅ Successful: %d tasks\n", successCount)
    if failureCount > 0 {
        cmd.Printf("  ⚠️  Failed: %d tasks\n", failureCount)
        cmd.Println("\nFailed tasks:")
        for _, task := range failedTasks {
            cmd.Printf("    - %s\n", task)
        }
    }

    cmd.Println("\nNext steps:")
    cmd.Println("  1. Ask Claude Code to help with evidence generation")
    cmd.Println("  2. Claude will read assembly prompts and guide you through:")
    cmd.Println("     - Running applicable tools")
    cmd.Println("     - Using evidence-generator for synthesis")
    cmd.Println("     - Creating comprehensive reports")
    cmd.Println("\n  Example: 'Help me generate all pending evidence'")

    return nil
}
```

### Phase 3: Add Optional Tool Execution

#### Task 3.1: Add --with-tool-data Flag
**File**: `cmd/evidence.go`

**In init():**
```go
evidenceGenerateCmd.Flags().Bool("with-tool-data", false,
    "execute applicable tools and collect data during context generation")
```

#### Task 3.2: Implement Tool Execution
**File**: `cmd/evidence.go` (add new function)

```go
func executeApplicableTools(task *domain.EvidenceTask, tools []string, outputDir string, cfg *config.Config) error {
    toolRegistry := tools.GetRegistry()

    for _, toolName := range tools {
        tool := toolRegistry.GetTool(toolName)
        if tool == nil {
            continue // Skip unknown tools
        }

        // Execute tool
        ctx := context.Background()
        request := createToolRequest(task, toolName, cfg)

        result, err := tool.Execute(ctx, request)
        if err != nil {
            // Log error but continue with other tools
            continue
        }

        // Save tool output
        outputFile := filepath.Join(outputDir, fmt.Sprintf("%s.json", toolName))
        data, _ := json.MarshalIndent(result, "", "  ")
        os.WriteFile(outputFile, data, 0644)
    }

    return nil
}
```

### Phase 4: Documentation Updates

#### Task 4.1: Update evidence-workflow.md
**File**: `.grctool/docs/evidence-workflow.md`

**Update Step 2 section** to reflect new assembly context:
```markdown
### Step 2: Generate Assembly Context

**Purpose**: Create comprehensive assembly context with prompts and templates
**Command**: `grctool evidence generate ET-XXXX --window {window}`

...What Happens:
- Creates `.context/assembly-prompt.md` via prompt-assembler
- Generates `.context/claude-instructions.md`
- Provides `.context/evidence-template.md`
- Identifies applicable tools
...
```

#### Task 4.2: Update bulk-operations.md
**File**: `.grctool/docs/bulk-operations.md`

**Update Step 3** to show new assembly context generation.

#### Task 4.3: Update CLAUDE.md in isms project
**File**: `/Users/erik/Projects/7thsense-ops/isms/CLAUDE.md`

Add section on assembly workflow.

---

## File Structure After Implementation

```
evidence/ET-0008_Workstation_Firewall_Settings/
└── 2025-Q4/
    ├── .context/
    │   ├── assembly-prompt.md           # Comprehensive prompt (NEW)
    │   ├── claude-instructions.md       # How to use materials (NEW)
    │   ├── evidence-template.md         # Pre-structured template (NEW)
    │   └── tool_outputs/                # Optional tool data (NEW)
    │       ├── github-permissions.json
    │       └── terraform-security.json
    ├── .generation/
    │   └── metadata.json
    └── (evidence files written by user with Claude's help)
```

---

## Testing Plan

### Test 1: Single Task Assembly
```bash
grctool evidence generate ET-0008 --window 2025-Q4
```

**Expected**:
- `.context/assembly-prompt.md` created (comprehensive, 200+ lines)
- `.context/claude-instructions.md` created
- `.context/evidence-template.md` created
- All files contain task-specific content

### Test 2: Bulk Assembly Generation
```bash
grctool evidence generate --all --window 2025-Q4
```

**Expected**:
- All pending tasks get assembly contexts
- Progress shown for each task
- Summary reports success/failures

### Test 3: With Tool Data
```bash
grctool evidence generate ET-0008 --window 2025-Q4 --with-tool-data
```

**Expected**:
- Assembly context created
- Applicable tools executed
- Tool outputs saved to `tool_outputs/`

### Test 4: Claude Integration
```bash
# After generating context
claude

# In Claude session
> "Help me generate evidence for ET-0008"
```

**Expected**:
- Claude reads assembly-prompt.md
- Claude reads claude-instructions.md
- Claude guides through evidence-generator usage
- Results in comprehensive 500+ line report

---

## Migration Notes

**No backward compatibility** - this replaces minimal context generation entirely.

**Affected commands**:
- `grctool evidence generate <task-ref>` - now creates assembly context
- `grctool evidence generate --all` - now creates assembly contexts for all

**Removed**:
- `generateEvidenceContext()` - replaced with `generateAssemblyContext()`
- `formatContextAsMarkdown()` - replaced with prompt-assembler integration
- Minimal generation-context.md files

---

## Success Criteria

✅ `grctool evidence generate --all` creates comprehensive assembly contexts
✅ Claude has all materials needed to write 500+ line evidence reports
✅ Assembly contexts include prompts, instructions, and templates
✅ Workflow integrates existing prompt-assembler and evidence-generator tools
✅ Documentation updated to reflect new assembly-first approach
✅ Tests pass for single task and bulk generation

---

## Implementation Checklist

### Phase 1: Core Integration
- [ ] Task 1.1: Modify processSingleTaskGeneration()
- [ ] Task 1.2: Create generateAssemblyContext()
- [ ] Task 1.3: Create executePromptAssembler()
- [ ] Task 1.4: Create generateClaudeInstructions()
- [ ] Task 1.5: Create evidence templates
- [ ] Task 1.6: Create saveAssemblyContext()

### Phase 2: Bulk Generation
- [ ] Task 2.1: Update processBulkEvidenceGeneration()

### Phase 3: Tool Execution
- [ ] Task 3.1: Add --with-tool-data flag
- [ ] Task 3.2: Implement executeApplicableTools()

### Phase 4: Documentation
- [ ] Task 4.1: Update evidence-workflow.md
- [ ] Task 4.2: Update bulk-operations.md
- [ ] Task 4.3: Update CLAUDE.md

### Testing
- [ ] Test 1: Single task assembly
- [ ] Test 2: Bulk assembly generation
- [ ] Test 3: With tool data
- [ ] Test 4: Claude integration

---

**Ready to implement in next session.**
