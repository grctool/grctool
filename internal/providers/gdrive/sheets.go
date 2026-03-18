// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package gdrive

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/domain"
)

// SheetData represents a spreadsheet's content as rows and columns.
type SheetData struct {
	Title   string     `json:"title"`
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
}

// ControlMatrixBuilder builds a SheetData from a slice of controls.
type ControlMatrixBuilder struct{}

// controlMatrixHeaders are the column headers for the control matrix sheet.
var controlMatrixHeaders = []string{
	"Reference ID", "Name", "Status", "Risk Level",
	"Framework Codes", "Category", "Implemented Date", "Tested Date",
}

// evidenceTaskHeaders are the column headers for the evidence task sheet.
var evidenceTaskHeaders = []string{
	"Reference ID", "Name", "Status", "Priority",
	"Collection Interval", "Framework", "Category",
	"Last Collected", "Next Due",
}

// BuildControlMatrix creates a SheetData with controls as rows.
// Columns: Reference ID, Name, Status, Risk Level, Framework Codes, Category, Implemented Date, Tested Date
func (b *ControlMatrixBuilder) BuildControlMatrix(controls []domain.Control) *SheetData {
	rows := make([][]string, 0, len(controls))
	for _, c := range controls {
		frameworkCodes := formatFrameworkCodes(c.FrameworkCodes)
		rows = append(rows, []string{
			c.ReferenceID,
			c.Name,
			c.Status,
			c.RiskLevel,
			frameworkCodes,
			c.Category,
			formatDate(c.ImplementedDate),
			formatDate(c.TestedDate),
		})
	}
	return &SheetData{
		Title:   "Control Matrix",
		Headers: controlMatrixHeaders,
		Rows:    rows,
	}
}

// BuildEvidenceTaskSheet creates a SheetData for evidence tasks.
// Columns: Reference ID, Name, Status, Priority, Collection Interval, Framework, Category, Last Collected, Next Due
func (b *ControlMatrixBuilder) BuildEvidenceTaskSheet(tasks []domain.EvidenceTask) *SheetData {
	rows := make([][]string, 0, len(tasks))
	for _, t := range tasks {
		rows = append(rows, []string{
			t.ReferenceID,
			t.Name,
			t.Status,
			t.Priority,
			t.CollectionInterval,
			t.Framework,
			t.Category,
			formatDate(t.LastCollected),
			formatDate(t.NextDue),
		})
	}
	return &SheetData{
		Title:   "Evidence Tasks",
		Headers: evidenceTaskHeaders,
		Rows:    rows,
	}
}

// ParseControlMatrix converts a SheetData back to a slice of controls.
// Used for bidirectional sync: edits in Sheets -> control updates.
func (b *ControlMatrixBuilder) ParseControlMatrix(sheet *SheetData) ([]domain.Control, error) {
	colIndex, err := buildColumnIndex(sheet.Headers, controlMatrixHeaders)
	if err != nil {
		return nil, err
	}

	controls := make([]domain.Control, 0, len(sheet.Rows))
	for i, row := range sheet.Rows {
		c := domain.Control{
			ReferenceID:    getCell(row, colIndex["Reference ID"]),
			Name:           getCell(row, colIndex["Name"]),
			Status:         getCell(row, colIndex["Status"]),
			RiskLevel:      getCell(row, colIndex["Risk Level"]),
			Category:       getCell(row, colIndex["Category"]),
			FrameworkCodes: parseFrameworkCodes(getCell(row, colIndex["Framework Codes"])),
		}

		implDate, err := parseDate(getCell(row, colIndex["Implemented Date"]))
		if err != nil {
			return nil, fmt.Errorf("row %d: invalid Implemented Date: %w", i+1, err)
		}
		c.ImplementedDate = implDate

		testDate, err := parseDate(getCell(row, colIndex["Tested Date"]))
		if err != nil {
			return nil, fmt.Errorf("row %d: invalid Tested Date: %w", i+1, err)
		}
		c.TestedDate = testDate

		controls = append(controls, c)
	}
	return controls, nil
}

// ParseEvidenceTaskSheet converts a SheetData back to evidence tasks.
func (b *ControlMatrixBuilder) ParseEvidenceTaskSheet(sheet *SheetData) ([]domain.EvidenceTask, error) {
	colIndex, err := buildColumnIndex(sheet.Headers, evidenceTaskHeaders)
	if err != nil {
		return nil, err
	}

	tasks := make([]domain.EvidenceTask, 0, len(sheet.Rows))
	for i, row := range sheet.Rows {
		t := domain.EvidenceTask{
			ReferenceID:        getCell(row, colIndex["Reference ID"]),
			Name:               getCell(row, colIndex["Name"]),
			Status:             getCell(row, colIndex["Status"]),
			Priority:           getCell(row, colIndex["Priority"]),
			CollectionInterval: getCell(row, colIndex["Collection Interval"]),
			Framework:          getCell(row, colIndex["Framework"]),
			Category:           getCell(row, colIndex["Category"]),
		}

		lc, err := parseDate(getCell(row, colIndex["Last Collected"]))
		if err != nil {
			return nil, fmt.Errorf("row %d: invalid Last Collected: %w", i+1, err)
		}
		t.LastCollected = lc

		nd, err := parseDate(getCell(row, colIndex["Next Due"]))
		if err != nil {
			return nil, fmt.Errorf("row %d: invalid Next Due: %w", i+1, err)
		}
		t.NextDue = nd

		tasks = append(tasks, t)
	}
	return tasks, nil
}

// ToCSV exports SheetData as CSV string (for local use without Sheets API).
func (s *SheetData) ToCSV() string {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	// Write header row.
	_ = w.Write(s.Headers)
	// Write data rows.
	for _, row := range s.Rows {
		// Pad or truncate row to match header length.
		normalized := normalizeRow(row, len(s.Headers))
		_ = w.Write(normalized)
	}
	w.Flush()
	return buf.String()
}

// SheetDataFromCSV parses CSV into SheetData.
func SheetDataFromCSV(csvData string) (*SheetData, error) {
	r := csv.NewReader(strings.NewReader(csvData))
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parsing CSV: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("CSV is empty: no header row")
	}

	sheet := &SheetData{
		Headers: records[0],
	}
	if len(records) > 1 {
		sheet.Rows = records[1:]
	}
	return sheet, nil
}

// --- helpers ---

// formatDate formats a *time.Time as "2006-01-02" or returns "" for nil/zero.
func formatDate(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

// parseDate parses a "2006-01-02" string into *time.Time. Returns nil for empty strings.
func parseDate(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// formatFrameworkCodes joins FrameworkCode.Code values with ", ".
func formatFrameworkCodes(codes []domain.FrameworkCode) string {
	if len(codes) == 0 {
		return ""
	}
	parts := make([]string, len(codes))
	for i, fc := range codes {
		parts[i] = fc.Code
	}
	return strings.Join(parts, ", ")
}

// parseFrameworkCodes splits a comma-separated string into FrameworkCode structs.
func parseFrameworkCodes(s string) []domain.FrameworkCode {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	codes := make([]domain.FrameworkCode, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			codes = append(codes, domain.FrameworkCode{Code: p})
		}
	}
	return codes
}

// buildColumnIndex maps required header names to their column indices.
// It returns an error if any required header is missing.
func buildColumnIndex(actual, required []string) (map[string]int, error) {
	index := make(map[string]int, len(actual))
	for i, h := range actual {
		index[h] = i
	}
	for _, rh := range required {
		if _, ok := index[rh]; !ok {
			return nil, fmt.Errorf("missing required header: %q", rh)
		}
	}
	return index, nil
}

// getCell safely retrieves a cell value from a row by column index.
// Returns "" if the index is out of range.
func getCell(row []string, col int) string {
	if col < 0 || col >= len(row) {
		return ""
	}
	return row[col]
}

// normalizeRow pads or truncates a row to the desired length.
func normalizeRow(row []string, length int) []string {
	if len(row) == length {
		return row
	}
	out := make([]string, length)
	copy(out, row)
	return out
}
