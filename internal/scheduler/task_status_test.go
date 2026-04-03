// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClassifyTaskDue_Nil(t *testing.T) {
	t.Parallel()
	cat, days := ClassifyTaskDue(nil, time.Now())
	assert.Equal(t, TaskNoSchedule, cat)
	assert.Equal(t, 0, days)
}

func TestClassifyTaskDue_Overdue(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	due := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	cat, days := ClassifyTaskDue(&due, now)
	assert.Equal(t, TaskOverdue, cat)
	assert.Equal(t, -5, days)
}

func TestClassifyTaskDue_DueThisWeek(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	due := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	cat, days := ClassifyTaskDue(&due, now)
	assert.Equal(t, TaskDueThisWeek, cat)
	assert.Equal(t, 4, days)
}

func TestClassifyTaskDue_DueThisMonth(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	due := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	cat, days := ClassifyTaskDue(&due, now)
	assert.Equal(t, TaskDueThisMonth, cat)
	assert.Equal(t, 15, days)
}

func TestClassifyTaskDue_Upcoming(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	due := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	cat, days := ClassifyTaskDue(&due, now)
	assert.Equal(t, TaskUpcoming, cat)
	assert.True(t, days > 30)
}

func TestClassifyTaskDue_DueToday(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	due := time.Date(2026, 4, 10, 18, 0, 0, 0, time.UTC) // later today
	cat, days := ClassifyTaskDue(&due, now)
	assert.Equal(t, TaskDueThisWeek, cat)
	assert.Equal(t, 0, days)
}

func TestClassifyTaskDue_DueExactly7Days(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	due := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)
	cat, _ := ClassifyTaskDue(&due, now)
	assert.Equal(t, TaskDueThisWeek, cat)
}

func TestClassifyTaskDue_DueExactly30Days(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	due := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	cat, _ := ClassifyTaskDue(&due, now)
	assert.Equal(t, TaskDueThisMonth, cat)
}

func TestGroupTasksByDue(t *testing.T) {
	t.Parallel()
	details := []TaskDueDetail{
		{TaskRef: "ET-0001", Category: TaskOverdue, DaysUntil: -3},
		{TaskRef: "ET-0002", Category: TaskOverdue, DaysUntil: -1},
		{TaskRef: "ET-0003", Category: TaskDueThisWeek, DaysUntil: 2},
		{TaskRef: "ET-0004", Category: TaskDueThisMonth, DaysUntil: 15},
		{TaskRef: "ET-0005", Category: TaskUpcoming, DaysUntil: 60},
	}

	g := GroupTasksByDue(details)
	assert.Len(t, g.Overdue, 2)
	assert.Len(t, g.DueThisWeek, 1)
	assert.Len(t, g.DueThisMonth, 1)
	assert.Len(t, g.Upcoming, 1)
}

func TestGroupTasksByDue_Empty(t *testing.T) {
	t.Parallel()
	g := GroupTasksByDue(nil)
	assert.Empty(t, g.Overdue)
	assert.Empty(t, g.DueThisWeek)
	assert.Empty(t, g.DueThisMonth)
	assert.Empty(t, g.Upcoming)
}

func TestGroupTasksByDue_AllSameCategory(t *testing.T) {
	t.Parallel()
	details := []TaskDueDetail{
		{TaskRef: "ET-0001", Category: TaskOverdue},
		{TaskRef: "ET-0002", Category: TaskOverdue},
	}
	g := GroupTasksByDue(details)
	assert.Len(t, g.Overdue, 2)
	assert.Empty(t, g.DueThisWeek)
	assert.Empty(t, g.DueThisMonth)
	assert.Empty(t, g.Upcoming)
}
