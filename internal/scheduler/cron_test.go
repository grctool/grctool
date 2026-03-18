// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"testing"
	"time"
)

// --- parseCron tests ---

func TestParseCron_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		expr string
	}{
		{"every minute", "* * * * *"},
		{"hourly at :00", "0 * * * *"},
		{"daily at 9am", "0 9 * * *"},
		{"weekly monday 9am", "0 9 * * 1"},
		{"monthly first day", "0 0 1 * *"},
		{"specific month", "0 0 1 6 *"},
		{"step minutes", "*/5 * * * *"},
		{"step hours", "0 */2 * * *"},
		{"range minutes", "1-5 * * * *"},
		{"list minutes", "0,15,30,45 * * * *"},
		{"range with step", "1-10/2 * * * *"},
		{"complex", "*/15 9-17 * 1,4,7,10 1-5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			expr, err := parseCron(tt.expr)
			if err != nil {
				t.Fatalf("parseCron(%q) unexpected error: %v", tt.expr, err)
			}
			if expr == nil {
				t.Fatal("expected non-nil expression")
			}
		})
	}
}

func TestParseCron_Invalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		expr    string
		wantErr string
	}{
		{"empty", "", "must have 5 fields"},
		{"one field", "0", "must have 5 fields"},
		{"four fields", "0 * * *", "must have 5 fields"},
		{"six fields", "0 * * * * *", "must have 5 fields"},
		{"invalid minute value", "60 * * * *", "out of range"},
		{"negative minute", "-1 * * * *", "invalid"},
		{"invalid hour", "0 24 * * *", "out of range"},
		{"invalid day-of-month", "0 0 32 * *", "out of range"},
		{"invalid month", "0 0 * 13 *", "out of range"},
		{"invalid day-of-week", "0 0 * * 7", "out of range"},
		{"zero month", "0 0 * 0 *", "out of range"},
		{"zero day", "0 0 0 * *", "out of range"},
		{"invalid step", "*/0 * * * *", "invalid step"},
		{"negative step", "*/-1 * * * *", "invalid step"},
		{"non-numeric step", "*/abc * * * *", "invalid step"},
		{"alpha in minute", "abc * * * *", "invalid value"},
		{"alpha in hour", "0 abc * * *", "invalid"},
		{"alpha in day", "0 0 abc * *", "invalid"},
		{"alpha in month", "0 0 * abc *", "invalid"},
		{"alpha in dow", "0 0 * * abc", "invalid"},
		{"invalid range reversed", "5-1 * * * *", "out of bounds"},
		{"range below min", "0 0 0-5 * *", "out of bounds"},
		{"range above max", "0 0 * 1-13 *", "out of bounds"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseCron(tt.expr)
			if err == nil {
				t.Fatalf("parseCron(%q) expected error containing %q, got nil", tt.expr, tt.wantErr)
			}
		})
	}
}

// --- parseField tests ---

func TestParseField_Wildcard(t *testing.T) {
	t.Parallel()

	vals, err := parseField("*", 0, 59)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vals) != 60 {
		t.Errorf("expected 60 values for minute wildcard, got %d", len(vals))
	}
	if vals[0] != 0 || vals[59] != 59 {
		t.Errorf("expected range 0-59, got %d-%d", vals[0], vals[len(vals)-1])
	}
}

func TestParseField_SingleValue(t *testing.T) {
	t.Parallel()

	vals, err := parseField("30", 0, 59)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vals) != 1 || vals[0] != 30 {
		t.Errorf("expected [30], got %v", vals)
	}
}

func TestParseField_SingleValueOutOfRange(t *testing.T) {
	t.Parallel()

	_, err := parseField("60", 0, 59)
	if err == nil {
		t.Fatal("expected error for value out of range")
	}
}

func TestParseField_Range(t *testing.T) {
	t.Parallel()

	vals, err := parseField("1-5", 0, 59)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []int{1, 2, 3, 4, 5}
	if len(vals) != len(expected) {
		t.Fatalf("expected %d values, got %d", len(expected), len(vals))
	}
	for i, v := range expected {
		if vals[i] != v {
			t.Errorf("vals[%d] = %d, want %d", i, vals[i], v)
		}
	}
}

func TestParseField_List(t *testing.T) {
	t.Parallel()

	vals, err := parseField("1,3,5", 0, 59)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []int{1, 3, 5}
	if len(vals) != len(expected) {
		t.Fatalf("expected %d values, got %d: %v", len(expected), len(vals), vals)
	}
	for i, v := range expected {
		if vals[i] != v {
			t.Errorf("vals[%d] = %d, want %d", i, vals[i], v)
		}
	}
}

func TestParseField_ListWithDuplicates(t *testing.T) {
	t.Parallel()

	vals, err := parseField("1,3,1,5,3", 0, 59)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// dedupSort should remove duplicates and sort.
	expected := []int{1, 3, 5}
	if len(vals) != len(expected) {
		t.Fatalf("expected %d values after dedup, got %d: %v", len(expected), len(vals), vals)
	}
	for i, v := range expected {
		if vals[i] != v {
			t.Errorf("vals[%d] = %d, want %d", i, vals[i], v)
		}
	}
}

func TestParseField_Step(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		field    string
		min, max int
		expected []int
	}{
		{"every 5 minutes", "*/5", 0, 59, []int{0, 5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55}},
		{"every 15 minutes", "*/15", 0, 59, []int{0, 15, 30, 45}},
		{"every 2 hours", "*/2", 0, 23, []int{0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22}},
		{"every 3 months", "*/3", 1, 12, []int{1, 4, 7, 10}},
		{"range with step", "1-10/3", 0, 59, []int{1, 4, 7, 10}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			vals, err := parseField(tt.field, tt.min, tt.max)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(vals) != len(tt.expected) {
				t.Fatalf("expected %d values, got %d: %v", len(tt.expected), len(vals), vals)
			}
			for i, v := range tt.expected {
				if vals[i] != v {
					t.Errorf("vals[%d] = %d, want %d", i, vals[i], v)
				}
			}
		})
	}
}

func TestParseField_ListInvalid(t *testing.T) {
	t.Parallel()

	_, err := parseField("1,abc,5", 0, 59)
	if err == nil {
		t.Fatal("expected error for invalid list element")
	}
}

// --- parseRange tests ---

func TestParseRange_Valid(t *testing.T) {
	t.Parallel()

	lo, hi, err := parseRange("5-10", 0, 59)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lo != 5 || hi != 10 {
		t.Errorf("expected 5-10, got %d-%d", lo, hi)
	}
}

func TestParseRange_Boundary(t *testing.T) {
	t.Parallel()

	lo, hi, err := parseRange("0-59", 0, 59)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lo != 0 || hi != 59 {
		t.Errorf("expected 0-59, got %d-%d", lo, hi)
	}
}

func TestParseRange_Invalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		field string
		min   int
		max   int
	}{
		{"reversed", "10-5", 0, 59},
		{"below min", "0-5", 1, 31},
		{"above max", "1-60", 0, 59},
		{"non-numeric start", "a-5", 0, 59},
		{"non-numeric end", "1-b", 0, 59},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := parseRange(tt.field, tt.min, tt.max)
			if err == nil {
				t.Fatalf("expected error for range %q [%d,%d]", tt.field, tt.min, tt.max)
			}
		})
	}
}

// --- dedupSort tests ---

func TestDedupSort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		vals     []int
		min, max int
		expected []int
	}{
		{"no duplicates", []int{3, 1, 5}, 0, 10, []int{1, 3, 5}},
		{"with duplicates", []int{3, 1, 3, 5, 1}, 0, 10, []int{1, 3, 5}},
		{"single value", []int{7}, 0, 10, []int{7}},
		{"empty", []int{}, 0, 10, nil},
		{"all same", []int{5, 5, 5}, 0, 10, []int{5}},
		{"filters out of range", []int{0, 5, 15}, 0, 10, []int{0, 5}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := dedupSort(tt.vals, tt.min, tt.max)
			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d values, got %d: %v", len(tt.expected), len(result), result)
			}
			for i := range tt.expected {
				if result[i] != tt.expected[i] {
					t.Errorf("result[%d] = %d, want %d", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

// --- contains tests ---

func TestContains(t *testing.T) {
	t.Parallel()

	if !contains([]int{1, 2, 3}, 2) {
		t.Error("expected contains([1,2,3], 2) = true")
	}
	if contains([]int{1, 2, 3}, 4) {
		t.Error("expected contains([1,2,3], 4) = false")
	}
	if contains([]int{}, 1) {
		t.Error("expected contains([], 1) = false")
	}
	if contains(nil, 0) {
		t.Error("expected contains(nil, 0) = false")
	}
}

// --- Next() edge case tests ---

func TestNext_MidnightRollover(t *testing.T) {
	t.Parallel()

	// "30 23 * * *" = 23:30 daily
	after := time.Date(2026, 3, 18, 23, 45, 0, 0, time.UTC)
	expr, err := parseCron("30 23 * * *")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	next := expr.Next(after)
	expected := time.Date(2026, 3, 19, 23, 30, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}

func TestNext_MonthEnd(t *testing.T) {
	t.Parallel()

	// "0 0 1 * *" = midnight on the 1st of every month
	after := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)
	expr, err := parseCron("0 0 1 * *")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	next := expr.Next(after)
	expected := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}

func TestNext_LeapYear(t *testing.T) {
	t.Parallel()

	// "0 0 29 2 *" = Feb 29, midnight
	// 2028 is a leap year.
	after := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	expr, err := parseCron("0 0 29 2 *")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	next := expr.Next(after)
	// Feb 29 2028 is a Saturday (weekday 6), but * matches all days of week.
	expected := time.Date(2028, 2, 29, 0, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}

func TestNext_DayOfWeek_MondaysAt9(t *testing.T) {
	t.Parallel()

	// "0 9 * * 1" = Mondays at 9am
	// 2026-03-18 is a Wednesday.
	after := time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC)
	expr, err := parseCron("0 9 * * 1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	next := expr.Next(after)
	// Next Monday is 2026-03-23.
	expected := time.Date(2026, 3, 23, 9, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}

func TestNext_DayOfWeek_Weekdays(t *testing.T) {
	t.Parallel()

	// "0 9 * * 1-5" = weekdays at 9am
	// 2026-03-20 is a Friday.
	after := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	expr, err := parseCron("0 9 * * 1-5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	next := expr.Next(after)
	// Next weekday is Monday 2026-03-23.
	expected := time.Date(2026, 3, 23, 9, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}

func TestNext_SpecificMonth(t *testing.T) {
	t.Parallel()

	// "0 0 1 6 *" = June 1st midnight
	after := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	expr, err := parseCron("0 0 1 6 *")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	next := expr.Next(after)
	// Next June 1st is 2027, but only if that day matches dow.
	// * for dow means any day matches.
	expected := time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}

func TestNext_AdvanceMonth(t *testing.T) {
	t.Parallel()

	// "0 0 1 1 *" = January 1st midnight
	after := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	expr, err := parseCron("0 0 1 1 *")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	next := expr.Next(after)
	expected := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}

func TestNext_EveryFiveMinutes(t *testing.T) {
	t.Parallel()

	// "*/5 * * * *"
	after := time.Date(2026, 3, 18, 10, 7, 0, 0, time.UTC)
	expr, err := parseCron("*/5 * * * *")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	next := expr.Next(after)
	expected := time.Date(2026, 3, 18, 10, 10, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}

func TestNext_QuarterlyMonths(t *testing.T) {
	t.Parallel()

	// "0 0 1 1,4,7,10 *" = quarterly on the 1st
	after := time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)
	expr, err := parseCron("0 0 1 1,4,7,10 *")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	next := expr.Next(after)
	expected := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}

func TestNext_SundayAt0(t *testing.T) {
	t.Parallel()

	// "0 0 * * 0" = Sundays at midnight
	// 2026-03-18 is Wednesday.
	after := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	expr, err := parseCron("0 0 * * 0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	next := expr.Next(after)
	// Next Sunday is 2026-03-22.
	expected := time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}

// --- advanceMonth tests ---

func TestAdvanceMonth_Basic(t *testing.T) {
	t.Parallel()

	// From March, advance to June if months are [6, 12].
	result := advanceMonth(
		time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC),
		[]int{6, 12},
	)
	expected := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestAdvanceMonth_YearWrap(t *testing.T) {
	t.Parallel()

	// From November, advance to January (months [1]).
	result := advanceMonth(
		time.Date(2026, 11, 15, 10, 0, 0, 0, time.UTC),
		[]int{1},
	)
	expected := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestAdvanceMonth_SameMonth(t *testing.T) {
	t.Parallel()

	// From March, advance when months = [3]. Should go to next year's March.
	result := advanceMonth(
		time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC),
		[]int{3},
	)
	// advanceMonth starts from month+1, so it wraps to March next year.
	expected := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestAdvanceMonth_December(t *testing.T) {
	t.Parallel()

	// From December, advance to January (months all).
	result := advanceMonth(
		time.Date(2026, 12, 15, 10, 0, 0, 0, time.UTC),
		[]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
	)
	expected := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

// --- Integration tests for parsed expressions ---

func TestParsedExpressionFields(t *testing.T) {
	t.Parallel()

	expr, err := parseCron("*/15 9-17 1,15 * 1-5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Minutes: 0, 15, 30, 45
	if len(expr.minutes) != 4 {
		t.Errorf("expected 4 minutes, got %d: %v", len(expr.minutes), expr.minutes)
	}

	// Hours: 9, 10, 11, 12, 13, 14, 15, 16, 17
	if len(expr.hours) != 9 {
		t.Errorf("expected 9 hours, got %d: %v", len(expr.hours), expr.hours)
	}

	// Days of month: 1, 15
	if len(expr.daysOfMonth) != 2 {
		t.Errorf("expected 2 days, got %d: %v", len(expr.daysOfMonth), expr.daysOfMonth)
	}

	// Months: all 12
	if len(expr.months) != 12 {
		t.Errorf("expected 12 months, got %d", len(expr.months))
	}

	// Days of week: 1-5 (Mon-Fri)
	if len(expr.daysOfWeek) != 5 {
		t.Errorf("expected 5 weekdays, got %d: %v", len(expr.daysOfWeek), expr.daysOfWeek)
	}
}
