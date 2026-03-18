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

package scheduler

import (
	"context"
	"time"

	"github.com/grctool/grctool/internal/logger"
)

// TaskToolMapping maps an evidence task to the tools that collect its evidence.
type TaskToolMapping struct {
	TaskRef  string   `json:"task_ref" yaml:"task_ref"`                          // e.g., "ET-0047"
	Tools    []string `json:"tools" yaml:"tools"`                                // tool names, in order
	Schedule string   `json:"schedule,omitempty" yaml:"schedule,omitempty"`       // schedule name
}

// CollectionPlan is an ordered list of tool runs for a collection cycle.
type CollectionPlan struct {
	Tasks []PlannedTask `json:"tasks" yaml:"tasks"`
}

// PlannedTask is one evidence task to collect.
type PlannedTask struct {
	TaskRef string           `json:"task_ref" yaml:"task_ref"`
	Tools   []PlannedToolRun `json:"tools" yaml:"tools"`
}

// PlannedToolRun is one tool execution within a task.
type PlannedToolRun struct {
	ToolName string                 `json:"tool_name" yaml:"tool_name"`
	Params   map[string]interface{} `json:"params,omitempty" yaml:"params,omitempty"`
	Order    int                    `json:"order" yaml:"order"`
}

// CollectionResult captures the outcome of a collection run.
type CollectionResult struct {
	TaskRef   string        `json:"task_ref" yaml:"task_ref"`
	ToolName  string        `json:"tool_name" yaml:"tool_name"`
	Success   bool          `json:"success" yaml:"success"`
	Output    string        `json:"output,omitempty" yaml:"output,omitempty"`
	Error     string        `json:"error,omitempty" yaml:"error,omitempty"`
	Duration  time.Duration `json:"duration" yaml:"duration"`
	StartedAt time.Time     `json:"started_at" yaml:"started_at"`
}

// CollectionSummary summarizes a full collection run.
type CollectionSummary struct {
	TotalTasks int                `json:"total_tasks"`
	TotalTools int                `json:"total_tools"`
	Succeeded  int                `json:"succeeded"`
	Failed     int                `json:"failed"`
	Skipped    int                `json:"skipped"`
	Results    []CollectionResult `json:"results"`
	Duration   time.Duration      `json:"duration"`
}

// Orchestrator coordinates evidence collection across tasks and tools.
type Orchestrator struct {
	mappings []TaskToolMapping
	logger   logger.Logger
}

// NewOrchestrator creates an orchestrator from task-tool mappings.
func NewOrchestrator(mappings []TaskToolMapping, log logger.Logger) *Orchestrator {
	return &Orchestrator{
		mappings: mappings,
		logger:   log,
	}
}

// BuildPlan creates a collection plan for the given task refs.
// If taskRefs is empty, plans for all mapped tasks.
func (o *Orchestrator) BuildPlan(taskRefs []string) *CollectionPlan {
	plan := &CollectionPlan{}

	if len(taskRefs) == 0 {
		// Plan all mapped tasks.
		for _, m := range o.mappings {
			plan.Tasks = append(plan.Tasks, buildPlannedTask(m))
		}
		return plan
	}

	// Build a lookup set for requested task refs.
	requested := make(map[string]struct{}, len(taskRefs))
	for _, ref := range taskRefs {
		requested[ref] = struct{}{}
	}

	for _, m := range o.mappings {
		if _, ok := requested[m.TaskRef]; ok {
			plan.Tasks = append(plan.Tasks, buildPlannedTask(m))
		}
	}

	return plan
}

// buildPlannedTask converts a mapping into a PlannedTask.
func buildPlannedTask(m TaskToolMapping) PlannedTask {
	pt := PlannedTask{
		TaskRef: m.TaskRef,
	}
	for i, toolName := range m.Tools {
		pt.Tools = append(pt.Tools, PlannedToolRun{
			ToolName: toolName,
			Order:    i,
		})
	}
	return pt
}

// Execute runs a collection plan using the provided tool executor.
// The executor func matches the pattern of tools.ExecuteTool (without the
// EvidenceSource return value) to keep the orchestrator decoupled from the
// tool registry.
//
// Execution rules:
//   - Tasks run sequentially in plan order.
//   - Tools within a task run sequentially in order.
//   - If a tool fails, remaining tools for that task are skipped.
//   - Execution continues to the next task regardless of failures.
//   - Context cancellation stops execution immediately.
func (o *Orchestrator) Execute(
	ctx context.Context,
	plan *CollectionPlan,
	executor func(ctx context.Context, toolName string, params map[string]interface{}) (string, error),
) *CollectionSummary {
	start := time.Now()
	summary := &CollectionSummary{}

	if plan == nil {
		summary.Duration = time.Since(start)
		return summary
	}

	summary.TotalTasks = len(plan.Tasks)

	for _, task := range plan.Tasks {
		summary.TotalTools += len(task.Tools)
	}

	for _, task := range plan.Tasks {
		// Check context before starting each task.
		if ctx.Err() != nil {
			o.logger.Warn("context cancelled, stopping collection",
				logger.String("reason", ctx.Err().Error()),
			)
			// Skip remaining tools for this and all subsequent tasks.
			for _, tool := range task.Tools {
				summary.Results = append(summary.Results, CollectionResult{
					TaskRef:   task.TaskRef,
					ToolName:  tool.ToolName,
					Success:   false,
					Error:     ctx.Err().Error(),
					StartedAt: time.Now(),
				})
				summary.Skipped++
			}
			// Count remaining tasks' tools as skipped too.
			continue
		}

		o.executeTask(ctx, task, executor, summary)
	}

	summary.Duration = time.Since(start)
	return summary
}

// executeTask runs all tools for a single task, recording results into summary.
func (o *Orchestrator) executeTask(
	ctx context.Context,
	task PlannedTask,
	executor func(ctx context.Context, toolName string, params map[string]interface{}) (string, error),
	summary *CollectionSummary,
) {
	taskFailed := false

	for _, tool := range task.Tools {
		// Check context before each tool run.
		if ctx.Err() != nil {
			summary.Results = append(summary.Results, CollectionResult{
				TaskRef:   task.TaskRef,
				ToolName:  tool.ToolName,
				Success:   false,
				Error:     ctx.Err().Error(),
				StartedAt: time.Now(),
			})
			summary.Skipped++
			continue
		}

		// If a previous tool in this task failed, skip remaining tools.
		if taskFailed {
			o.logger.Info("skipping tool due to prior failure in task",
				logger.String("task_ref", task.TaskRef),
				logger.String("tool", tool.ToolName),
			)
			summary.Results = append(summary.Results, CollectionResult{
				TaskRef:   task.TaskRef,
				ToolName:  tool.ToolName,
				Success:   false,
				Error:     "skipped: prior tool failed",
				StartedAt: time.Now(),
			})
			summary.Skipped++
			continue
		}

		toolStart := time.Now()
		o.logger.Info("executing tool",
			logger.String("task_ref", task.TaskRef),
			logger.String("tool", tool.ToolName),
		)

		output, err := executor(ctx, tool.ToolName, tool.Params)
		duration := time.Since(toolStart)

		result := CollectionResult{
			TaskRef:   task.TaskRef,
			ToolName:  tool.ToolName,
			Duration:  duration,
			StartedAt: toolStart,
		}

		if err != nil {
			result.Success = false
			result.Error = err.Error()
			taskFailed = true
			summary.Failed++
			o.logger.Error("tool execution failed",
				logger.String("task_ref", task.TaskRef),
				logger.String("tool", tool.ToolName),
				logger.String("error", err.Error()),
			)
		} else {
			result.Success = true
			result.Output = output
			summary.Succeeded++
			o.logger.Info("tool execution succeeded",
				logger.String("task_ref", task.TaskRef),
				logger.String("tool", tool.ToolName),
			)
		}

		summary.Results = append(summary.Results, result)
	}
}

// GetMappingsForSchedule returns mappings associated with a schedule name.
func (o *Orchestrator) GetMappingsForSchedule(scheduleName string) []TaskToolMapping {
	var result []TaskToolMapping
	for _, m := range o.mappings {
		if m.Schedule == scheduleName {
			result = append(result, m)
		}
	}
	return result
}

// GetMappingForTask returns the mapping for a specific task ref.
func (o *Orchestrator) GetMappingForTask(taskRef string) (*TaskToolMapping, bool) {
	for i := range o.mappings {
		if o.mappings[i].TaskRef == taskRef {
			return &o.mappings[i], true
		}
	}
	return nil, false
}
