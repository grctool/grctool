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
	"fmt"
	"strconv"
	"strings"
	"time"
)

// cronExpr is a parsed 5-field cron expression.
// Fields: minute (0-59), hour (0-23), day-of-month (1-31), month (1-12), day-of-week (0-6, 0=Sunday).
type cronExpr struct {
	minutes    []int
	hours      []int
	daysOfMonth []int
	months     []int
	daysOfWeek []int
}

// parseCron parses a standard 5-field cron expression.
// Supported syntax per field:
//   - "*"   — every value
//   - "*/N" — every N-th value
//   - "N"   — specific value
//   - "N,M" — list of specific values
//   - "N-M" — range of values
func parseCron(expr string) (*cronExpr, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return nil, fmt.Errorf("cron expression must have 5 fields, got %d: %q", len(fields), expr)
	}

	minutes, err := parseField(fields[0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("invalid minute field %q: %w", fields[0], err)
	}
	hours, err := parseField(fields[1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("invalid hour field %q: %w", fields[1], err)
	}
	daysOfMonth, err := parseField(fields[2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("invalid day-of-month field %q: %w", fields[2], err)
	}
	months, err := parseField(fields[3], 1, 12)
	if err != nil {
		return nil, fmt.Errorf("invalid month field %q: %w", fields[3], err)
	}
	daysOfWeek, err := parseField(fields[4], 0, 6)
	if err != nil {
		return nil, fmt.Errorf("invalid day-of-week field %q: %w", fields[4], err)
	}

	return &cronExpr{
		minutes:     minutes,
		hours:       hours,
		daysOfMonth: daysOfMonth,
		months:      months,
		daysOfWeek:  daysOfWeek,
	}, nil
}

// parseField parses a single cron field into a sorted list of matching values.
func parseField(field string, min, max int) ([]int, error) {
	// Handle comma-separated lists (e.g., "1,4,7,10").
	if strings.Contains(field, ",") {
		var result []int
		for _, part := range strings.Split(field, ",") {
			vals, err := parseField(part, min, max)
			if err != nil {
				return nil, err
			}
			result = append(result, vals...)
		}
		return dedupSort(result, min, max), nil
	}

	// Handle step values (e.g., "*/5", "1-10/2").
	if strings.Contains(field, "/") {
		parts := strings.SplitN(field, "/", 2)
		step, err := strconv.Atoi(parts[1])
		if err != nil || step <= 0 {
			return nil, fmt.Errorf("invalid step value: %q", parts[1])
		}

		rangeMin, rangeMax := min, max
		if parts[0] != "*" {
			rangeMin, rangeMax, err = parseRange(parts[0], min, max)
			if err != nil {
				return nil, err
			}
		}

		var result []int
		for v := rangeMin; v <= rangeMax; v += step {
			result = append(result, v)
		}
		return result, nil
	}

	// Handle wildcard.
	if field == "*" {
		result := make([]int, 0, max-min+1)
		for v := min; v <= max; v++ {
			result = append(result, v)
		}
		return result, nil
	}

	// Handle range (e.g., "1-5").
	if strings.Contains(field, "-") {
		lo, hi, err := parseRange(field, min, max)
		if err != nil {
			return nil, err
		}
		result := make([]int, 0, hi-lo+1)
		for v := lo; v <= hi; v++ {
			result = append(result, v)
		}
		return result, nil
	}

	// Single value.
	v, err := strconv.Atoi(field)
	if err != nil {
		return nil, fmt.Errorf("invalid value: %q", field)
	}
	if v < min || v > max {
		return nil, fmt.Errorf("value %d out of range [%d, %d]", v, min, max)
	}
	return []int{v}, nil
}

// parseRange parses "N-M" into (N, M).
func parseRange(field string, min, max int) (int, int, error) {
	parts := strings.SplitN(field, "-", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid range: %q", field)
	}
	lo, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid range start: %q", parts[0])
	}
	hi, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid range end: %q", parts[1])
	}
	if lo < min || hi > max || lo > hi {
		return 0, 0, fmt.Errorf("range %d-%d out of bounds [%d, %d]", lo, hi, min, max)
	}
	return lo, hi, nil
}

// dedupSort removes duplicates and returns values in range, sorted.
func dedupSort(vals []int, min, max int) []int {
	seen := make(map[int]bool, len(vals))
	var result []int
	// Iterate in order from min to max to produce sorted output.
	for _, v := range vals {
		seen[v] = true
	}
	for v := min; v <= max; v++ {
		if seen[v] {
			result = append(result, v)
		}
	}
	return result
}

// contains checks if a slice contains a value.
func contains(vals []int, v int) bool {
	for _, val := range vals {
		if val == v {
			return true
		}
	}
	return false
}

// Next returns the first time matching the cron expression that is strictly after t.
// It searches up to 4 years ahead to avoid infinite loops on impossible expressions.
func (c *cronExpr) Next(t time.Time) time.Time {
	// Start from the next minute after t.
	candidate := t.Truncate(time.Minute).Add(time.Minute)
	// Safety limit: do not search more than ~4 years.
	limit := t.Add(4 * 365 * 24 * time.Hour)

	for candidate.Before(limit) {
		if !contains(c.months, int(candidate.Month())) {
			// Advance to next matching month.
			candidate = advanceMonth(candidate, c.months)
			continue
		}

		if !contains(c.daysOfMonth, candidate.Day()) || !contains(c.daysOfWeek, int(candidate.Weekday())) {
			candidate = candidate.Add(24 * time.Hour)
			candidate = time.Date(candidate.Year(), candidate.Month(), candidate.Day(), 0, 0, 0, 0, candidate.Location())
			continue
		}

		if !contains(c.hours, candidate.Hour()) {
			candidate = candidate.Add(time.Hour)
			candidate = time.Date(candidate.Year(), candidate.Month(), candidate.Day(), candidate.Hour(), 0, 0, 0, candidate.Location())
			continue
		}

		if !contains(c.minutes, candidate.Minute()) {
			candidate = candidate.Add(time.Minute)
			continue
		}

		return candidate
	}

	// Should not happen for valid expressions; return the limit as fallback.
	return limit
}

// advanceMonth moves to the first day of the next matching month.
func advanceMonth(t time.Time, months []int) time.Time {
	year := t.Year()
	month := int(t.Month())

	for i := 0; i < 48; i++ { // 4 years of months
		month++
		if month > 12 {
			month = 1
			year++
		}
		if contains(months, month) {
			return time.Date(year, time.Month(month), 1, 0, 0, 0, 0, t.Location())
		}
	}

	// Fallback (should not happen for valid expressions).
	return t.Add(4 * 365 * 24 * time.Hour)
}
