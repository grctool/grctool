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
	"fmt"
	"testing"

	"github.com/grctool/grctool/internal/testhelpers"
)

// testMappings returns a standard set of mappings for tests.
func testMappings() []TaskToolMapping {
	return []TaskToolMapping{
		{TaskRef: "ET-0001", Tools: []string{"github-permissions", "github-security-features"}, Schedule: "daily"},
		{TaskRef: "ET-0002", Tools: []string{"terraform-security-analyzer"}, Schedule: "weekly"},
		{TaskRef: "ET-0003", Tools: []string{"google-workspace", "docs-reader"}, Schedule: "daily"},
	}
}

// successExecutor always succeeds.
func successExecutor(_ context.Context, _ string, _ map[string]interface{}) (string, error) {
	return "evidence collected", nil
}

// failingToolExecutor fails for a specific tool name.
func failingToolExecutor(failTool string) func(context.Context, string, map[string]interface{}) (string, error) {
	return func(_ context.Context, toolName string, _ map[string]interface{}) (string, error) {
		if toolName == failTool {
			return "", fmt.Errorf("tool failed")
		}
		return "evidence collected", nil
	}
}

func TestOrchestrator_BuildPlan_AllTasks(t *testing.T) {
	o := NewOrchestrator(testMappings(), testhelpers.NewStubLogger())

	plan := o.BuildPlan(nil)

	if len(plan.Tasks) != 3 {
		t.Fatalf("expected 3 planned tasks, got %d", len(plan.Tasks))
	}

	// Verify order is preserved.
	if plan.Tasks[0].TaskRef != "ET-0001" {
		t.Errorf("expected first task ET-0001, got %s", plan.Tasks[0].TaskRef)
	}
	if plan.Tasks[1].TaskRef != "ET-0002" {
		t.Errorf("expected second task ET-0002, got %s", plan.Tasks[1].TaskRef)
	}
	if plan.Tasks[2].TaskRef != "ET-0003" {
		t.Errorf("expected third task ET-0003, got %s", plan.Tasks[2].TaskRef)
	}

	// Verify tools are ordered within a task.
	if len(plan.Tasks[0].Tools) != 2 {
		t.Fatalf("expected 2 tools for ET-0001, got %d", len(plan.Tasks[0].Tools))
	}
	if plan.Tasks[0].Tools[0].ToolName != "github-permissions" {
		t.Errorf("expected first tool github-permissions, got %s", plan.Tasks[0].Tools[0].ToolName)
	}
	if plan.Tasks[0].Tools[0].Order != 0 {
		t.Errorf("expected order 0, got %d", plan.Tasks[0].Tools[0].Order)
	}
	if plan.Tasks[0].Tools[1].Order != 1 {
		t.Errorf("expected order 1, got %d", plan.Tasks[0].Tools[1].Order)
	}
}

func TestOrchestrator_BuildPlan_FilteredTasks(t *testing.T) {
	o := NewOrchestrator(testMappings(), testhelpers.NewStubLogger())

	plan := o.BuildPlan([]string{"ET-0002"})

	if len(plan.Tasks) != 1 {
		t.Fatalf("expected 1 planned task, got %d", len(plan.Tasks))
	}
	if plan.Tasks[0].TaskRef != "ET-0002" {
		t.Errorf("expected task ET-0002, got %s", plan.Tasks[0].TaskRef)
	}
}

func TestOrchestrator_BuildPlan_UnknownTask(t *testing.T) {
	o := NewOrchestrator(testMappings(), testhelpers.NewStubLogger())

	plan := o.BuildPlan([]string{"ET-9999"})

	if len(plan.Tasks) != 0 {
		t.Fatalf("expected 0 planned tasks for unknown ref, got %d", len(plan.Tasks))
	}
}

func TestOrchestrator_Execute_AllSucceed(t *testing.T) {
	o := NewOrchestrator(testMappings(), testhelpers.NewStubLogger())
	plan := o.BuildPlan(nil)

	summary := o.Execute(context.Background(), plan, successExecutor)

	// 3 tasks: ET-0001 (2 tools) + ET-0002 (1 tool) + ET-0003 (2 tools) = 5 tools
	if summary.TotalTasks != 3 {
		t.Errorf("expected 3 total tasks, got %d", summary.TotalTasks)
	}
	if summary.TotalTools != 5 {
		t.Errorf("expected 5 total tools, got %d", summary.TotalTools)
	}
	if summary.Succeeded != 5 {
		t.Errorf("expected 5 succeeded, got %d", summary.Succeeded)
	}
	if summary.Failed != 0 {
		t.Errorf("expected 0 failed, got %d", summary.Failed)
	}
	if summary.Skipped != 0 {
		t.Errorf("expected 0 skipped, got %d", summary.Skipped)
	}
	if len(summary.Results) != 5 {
		t.Fatalf("expected 5 results, got %d", len(summary.Results))
	}

	for _, r := range summary.Results {
		if !r.Success {
			t.Errorf("expected all results to succeed, %s/%s failed", r.TaskRef, r.ToolName)
		}
		if r.Output != "evidence collected" {
			t.Errorf("expected output 'evidence collected', got %q", r.Output)
		}
	}
}

func TestOrchestrator_Execute_ToolFails(t *testing.T) {
	o := NewOrchestrator(testMappings(), testhelpers.NewStubLogger())

	// Only plan ET-0001 which has 2 tools: github-permissions, github-security-features
	plan := o.BuildPlan([]string{"ET-0001"})

	// Fail the first tool — the second should be skipped.
	executor := failingToolExecutor("github-permissions")
	summary := o.Execute(context.Background(), plan, executor)

	if summary.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", summary.Failed)
	}
	if summary.Skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", summary.Skipped)
	}
	if summary.Succeeded != 0 {
		t.Errorf("expected 0 succeeded, got %d", summary.Succeeded)
	}

	// First result should be the failure.
	if summary.Results[0].ToolName != "github-permissions" {
		t.Errorf("expected first result for github-permissions, got %s", summary.Results[0].ToolName)
	}
	if summary.Results[0].Success {
		t.Error("expected first result to fail")
	}

	// Second result should be skipped.
	if summary.Results[1].ToolName != "github-security-features" {
		t.Errorf("expected second result for github-security-features, got %s", summary.Results[1].ToolName)
	}
	if summary.Results[1].Success {
		t.Error("expected second result to not succeed (skipped)")
	}
	if summary.Results[1].Error != "skipped: prior tool failed" {
		t.Errorf("expected skip error message, got %q", summary.Results[1].Error)
	}
}

func TestOrchestrator_Execute_MultipleTasksPartialFailure(t *testing.T) {
	mappings := []TaskToolMapping{
		{TaskRef: "ET-0001", Tools: []string{"failing-tool", "other-tool"}},
		{TaskRef: "ET-0002", Tools: []string{"good-tool"}},
	}
	o := NewOrchestrator(mappings, testhelpers.NewStubLogger())
	plan := o.BuildPlan(nil)

	executor := func(_ context.Context, toolName string, _ map[string]interface{}) (string, error) {
		if toolName == "failing-tool" {
			return "", fmt.Errorf("tool failed")
		}
		return "evidence collected", nil
	}

	summary := o.Execute(context.Background(), plan, executor)

	// ET-0001: failing-tool fails, other-tool skipped. ET-0002: good-tool succeeds.
	if summary.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", summary.Failed)
	}
	if summary.Skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", summary.Skipped)
	}
	if summary.Succeeded != 1 {
		t.Errorf("expected 1 succeeded, got %d", summary.Succeeded)
	}

	// Verify the successful result is for ET-0002.
	var found bool
	for _, r := range summary.Results {
		if r.TaskRef == "ET-0002" && r.ToolName == "good-tool" && r.Success {
			found = true
		}
	}
	if !found {
		t.Error("expected ET-0002/good-tool to succeed")
	}
}

func TestOrchestrator_Execute_ContextCancelled(t *testing.T) {
	mappings := []TaskToolMapping{
		{TaskRef: "ET-0001", Tools: []string{"tool-a"}},
		{TaskRef: "ET-0002", Tools: []string{"tool-b"}},
	}
	o := NewOrchestrator(mappings, testhelpers.NewStubLogger())
	plan := o.BuildPlan(nil)

	// Cancel context before execution.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	summary := o.Execute(ctx, plan, successExecutor)

	// All tools should be skipped due to cancelled context.
	if summary.Skipped != 2 {
		t.Errorf("expected 2 skipped, got %d", summary.Skipped)
	}
	if summary.Succeeded != 0 {
		t.Errorf("expected 0 succeeded, got %d", summary.Succeeded)
	}
	if summary.Failed != 0 {
		t.Errorf("expected 0 failed, got %d", summary.Failed)
	}
}

func TestOrchestrator_Execute_EmptyPlan(t *testing.T) {
	o := NewOrchestrator(testMappings(), testhelpers.NewStubLogger())
	plan := &CollectionPlan{}

	summary := o.Execute(context.Background(), plan, successExecutor)

	if summary.TotalTasks != 0 {
		t.Errorf("expected 0 total tasks, got %d", summary.TotalTasks)
	}
	if summary.TotalTools != 0 {
		t.Errorf("expected 0 total tools, got %d", summary.TotalTools)
	}
	if summary.Succeeded != 0 {
		t.Errorf("expected 0 succeeded, got %d", summary.Succeeded)
	}
	if len(summary.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(summary.Results))
	}
	if summary.Duration <= 0 {
		t.Error("expected positive duration")
	}
}

func TestOrchestrator_Execute_NilPlan(t *testing.T) {
	o := NewOrchestrator(testMappings(), testhelpers.NewStubLogger())

	summary := o.Execute(context.Background(), nil, successExecutor)

	if summary.TotalTasks != 0 {
		t.Errorf("expected 0 total tasks, got %d", summary.TotalTasks)
	}
}

func TestOrchestrator_GetMappingsForSchedule(t *testing.T) {
	o := NewOrchestrator(testMappings(), testhelpers.NewStubLogger())

	daily := o.GetMappingsForSchedule("daily")
	if len(daily) != 2 {
		t.Fatalf("expected 2 daily mappings, got %d", len(daily))
	}
	if daily[0].TaskRef != "ET-0001" {
		t.Errorf("expected first daily mapping ET-0001, got %s", daily[0].TaskRef)
	}
	if daily[1].TaskRef != "ET-0003" {
		t.Errorf("expected second daily mapping ET-0003, got %s", daily[1].TaskRef)
	}

	weekly := o.GetMappingsForSchedule("weekly")
	if len(weekly) != 1 {
		t.Fatalf("expected 1 weekly mapping, got %d", len(weekly))
	}

	none := o.GetMappingsForSchedule("nonexistent")
	if len(none) != 0 {
		t.Errorf("expected 0 mappings for nonexistent schedule, got %d", len(none))
	}
}

func TestOrchestrator_GetMappingForTask(t *testing.T) {
	o := NewOrchestrator(testMappings(), testhelpers.NewStubLogger())

	m, ok := o.GetMappingForTask("ET-0002")
	if !ok {
		t.Fatal("expected to find mapping for ET-0002")
	}
	if m.TaskRef != "ET-0002" {
		t.Errorf("expected task ref ET-0002, got %s", m.TaskRef)
	}
	if len(m.Tools) != 1 || m.Tools[0] != "terraform-security-analyzer" {
		t.Errorf("unexpected tools: %v", m.Tools)
	}

	_, ok = o.GetMappingForTask("ET-9999")
	if ok {
		t.Error("expected not to find mapping for ET-9999")
	}
}

func TestCollectionSummary_Counts(t *testing.T) {
	// Build a scenario with known counts and verify the summary.
	mappings := []TaskToolMapping{
		{TaskRef: "ET-0001", Tools: []string{"tool-a", "tool-b"}},
		{TaskRef: "ET-0002", Tools: []string{"failing-tool", "tool-c"}},
		{TaskRef: "ET-0003", Tools: []string{"tool-d"}},
	}
	o := NewOrchestrator(mappings, testhelpers.NewStubLogger())
	plan := o.BuildPlan(nil)

	executor := func(_ context.Context, toolName string, _ map[string]interface{}) (string, error) {
		if toolName == "failing-tool" {
			return "", fmt.Errorf("tool failed")
		}
		return "ok", nil
	}

	summary := o.Execute(context.Background(), plan, executor)

	// ET-0001: tool-a OK, tool-b OK (2 succeeded)
	// ET-0002: failing-tool FAIL, tool-c SKIPPED (1 failed, 1 skipped)
	// ET-0003: tool-d OK (1 succeeded)
	expectedTotal := 3 + 2 // 3 tasks, but TotalTools = 5
	if summary.TotalTasks != 3 {
		t.Errorf("expected 3 total tasks, got %d", summary.TotalTasks)
	}
	if summary.TotalTools != expectedTotal {
		t.Errorf("expected %d total tools, got %d", expectedTotal, summary.TotalTools)
	}
	if summary.Succeeded != 3 {
		t.Errorf("expected 3 succeeded, got %d", summary.Succeeded)
	}
	if summary.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", summary.Failed)
	}
	if summary.Skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", summary.Skipped)
	}

	// Verify invariant: succeeded + failed + skipped == total results.
	totalResults := summary.Succeeded + summary.Failed + summary.Skipped
	if totalResults != len(summary.Results) {
		t.Errorf("count mismatch: succeeded(%d) + failed(%d) + skipped(%d) = %d, but results has %d entries",
			summary.Succeeded, summary.Failed, summary.Skipped, totalResults, len(summary.Results))
	}
	if totalResults != summary.TotalTools {
		t.Errorf("count mismatch: succeeded+failed+skipped = %d, but TotalTools = %d",
			totalResults, summary.TotalTools)
	}
}
