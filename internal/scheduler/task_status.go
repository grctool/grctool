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
	"time"
)

// TaskDueCategory classifies how urgent an evidence task's due date is.
type TaskDueCategory string

const (
	TaskOverdue      TaskDueCategory = "overdue"
	TaskDueThisWeek  TaskDueCategory = "due_this_week"
	TaskDueThisMonth TaskDueCategory = "due_this_month"
	TaskUpcoming     TaskDueCategory = "upcoming"
	TaskNoSchedule   TaskDueCategory = "no_schedule"
)

// TaskDueDetail captures a single evidence task's due status.
type TaskDueDetail struct {
	TaskRef   string          `json:"task_ref"`
	TaskName  string          `json:"task_name"`
	Category  TaskDueCategory `json:"category"`
	DueDate   *time.Time      `json:"due_date,omitempty"`
	DaysUntil int             `json:"days_until"` // negative = overdue
}

// TaskDueGrouping groups evidence tasks by due urgency.
type TaskDueGrouping struct {
	Overdue      []TaskDueDetail `json:"overdue"`
	DueThisWeek  []TaskDueDetail `json:"due_this_week"`
	DueThisMonth []TaskDueDetail `json:"due_this_month"`
	Upcoming     []TaskDueDetail `json:"upcoming"`
}

// ClassifyTaskDue determines which due category a task falls into.
func ClassifyTaskDue(dueDate *time.Time, now time.Time) (TaskDueCategory, int) {
	if dueDate == nil {
		return TaskNoSchedule, 0
	}

	hours := dueDate.Sub(now).Hours()
	days := int(hours / 24)

	if now.After(*dueDate) {
		overdueDays := int(now.Sub(*dueDate).Hours() / 24)
		return TaskOverdue, -overdueDays
	}
	if days <= 7 {
		return TaskDueThisWeek, days
	}
	if days <= 30 {
		return TaskDueThisMonth, days
	}
	return TaskUpcoming, days
}

// GroupTasksByDue classifies a slice of task details into due groupings.
func GroupTasksByDue(details []TaskDueDetail) *TaskDueGrouping {
	g := &TaskDueGrouping{}
	for _, d := range details {
		switch d.Category {
		case TaskOverdue:
			g.Overdue = append(g.Overdue, d)
		case TaskDueThisWeek:
			g.DueThisWeek = append(g.DueThisWeek, d)
		case TaskDueThisMonth:
			g.DueThisMonth = append(g.DueThisMonth, d)
		case TaskUpcoming:
			g.Upcoming = append(g.Upcoming, d)
		}
	}
	return g
}
