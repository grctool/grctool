// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package gdrive

import (
	"testing"
	"time"

	"github.com/grctool/grctool/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func datePtr(y int, m time.Month, d int) *time.Time {
	t := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	return &t
}

func sampleControls() []domain.Control {
	return []domain.Control{
		{
			ReferenceID:     "CC-06.1",
			Name:            "Logical Access Security",
			Status:          "implemented",
			RiskLevel:       "medium",
			FrameworkCodes:  []domain.FrameworkCode{{Code: "CC6.1"}},
			Category:        "Common Criteria",
			ImplementedDate: datePtr(2025, 3, 15),
			TestedDate:      datePtr(2025, 6, 1),
		},
		{
			ReferenceID: "CC-06.3",
			Name:        "Access Removal",
			Status:      "implemented",
			RiskLevel:   "low",
			FrameworkCodes: []domain.FrameworkCode{
				{Code: "CC6.3"},
				{Code: "A1.2"},
			},
			Category:        "Common Criteria",
			ImplementedDate: datePtr(2025, 1, 10),
		},
		{
			ReferenceID: "SO-19",
			Name:        "Vulnerability Management",
			Status:      "in_progress",
			RiskLevel:   "high",
			Category:    "Security Operations",
		},
	}
}

func sampleEvidenceTasks() []domain.EvidenceTask {
	return []domain.EvidenceTask{
		{
			ReferenceID:        "ET-0047",
			Name:               "GitHub Repository Access Controls",
			Status:             "pending",
			Priority:           "high",
			CollectionInterval: "quarter",
			Framework:          "SOC2",
			Category:           "Infrastructure",
			LastCollected:      datePtr(2025, 1, 15),
			NextDue:            datePtr(2025, 4, 15),
		},
		{
			ReferenceID:        "ET-0001",
			Name:               "Access Registration",
			Status:             "completed",
			Priority:           "medium",
			CollectionInterval: "month",
			Framework:          "SOC2",
			Category:           "Personnel",
		},
	}
}

// --- Control Matrix Tests ---

func TestBuildControlMatrix_Basic(t *testing.T) {
	t.Parallel()
	b := &ControlMatrixBuilder{}
	controls := sampleControls()
	sheet := b.BuildControlMatrix(controls)

	assert.Equal(t, "Control Matrix", sheet.Title)
	assert.Equal(t, controlMatrixHeaders, sheet.Headers)
	require.Len(t, sheet.Rows, 3)

	// Verify first row field mapping.
	row := sheet.Rows[0]
	assert.Equal(t, "CC-06.1", row[0])            // Reference ID
	assert.Equal(t, "Logical Access Security", row[1]) // Name
	assert.Equal(t, "implemented", row[2])         // Status
	assert.Equal(t, "medium", row[3])              // Risk Level
	assert.Equal(t, "CC6.1", row[4])               // Framework Codes
	assert.Equal(t, "Common Criteria", row[5])     // Category
	assert.Equal(t, "2025-03-15", row[6])          // Implemented Date
	assert.Equal(t, "2025-06-01", row[7])          // Tested Date
}

func TestBuildControlMatrix_Empty(t *testing.T) {
	t.Parallel()
	b := &ControlMatrixBuilder{}
	sheet := b.BuildControlMatrix(nil)

	assert.Equal(t, controlMatrixHeaders, sheet.Headers)
	assert.Empty(t, sheet.Rows)
}

func TestBuildControlMatrix_FrameworkCodes(t *testing.T) {
	t.Parallel()
	b := &ControlMatrixBuilder{}
	controls := sampleControls()
	sheet := b.BuildControlMatrix(controls)

	// Second control has two framework codes.
	assert.Equal(t, "CC6.3, A1.2", sheet.Rows[1][4])
}

func TestBuildControlMatrix_MissingFields(t *testing.T) {
	t.Parallel()
	b := &ControlMatrixBuilder{}
	controls := sampleControls()
	sheet := b.BuildControlMatrix(controls)

	// Third control has no framework codes, no dates.
	row := sheet.Rows[2]
	assert.Equal(t, "SO-19", row[0])
	assert.Equal(t, "", row[4]) // Framework Codes — empty, not nil
	assert.Equal(t, "", row[6]) // Implemented Date
	assert.Equal(t, "", row[7]) // Tested Date
}

// --- Evidence Task Sheet Tests ---

func TestBuildEvidenceTaskSheet_Basic(t *testing.T) {
	t.Parallel()
	b := &ControlMatrixBuilder{}
	tasks := sampleEvidenceTasks()
	sheet := b.BuildEvidenceTaskSheet(tasks)

	assert.Equal(t, "Evidence Tasks", sheet.Title)
	assert.Equal(t, evidenceTaskHeaders, sheet.Headers)
	require.Len(t, sheet.Rows, 2)

	row := sheet.Rows[0]
	assert.Equal(t, "ET-0047", row[0])
	assert.Equal(t, "GitHub Repository Access Controls", row[1])
	assert.Equal(t, "pending", row[2])
	assert.Equal(t, "high", row[3])
	assert.Equal(t, "quarter", row[4])
	assert.Equal(t, "SOC2", row[5])
	assert.Equal(t, "Infrastructure", row[6])
	assert.Equal(t, "2025-01-15", row[7])
	assert.Equal(t, "2025-04-15", row[8])
}

func TestBuildEvidenceTaskSheet_Empty(t *testing.T) {
	t.Parallel()
	b := &ControlMatrixBuilder{}
	sheet := b.BuildEvidenceTaskSheet(nil)

	assert.Equal(t, evidenceTaskHeaders, sheet.Headers)
	assert.Empty(t, sheet.Rows)
}

// --- Parse (Round-trip) Tests ---

func TestParseControlMatrix_RoundTrip(t *testing.T) {
	t.Parallel()
	b := &ControlMatrixBuilder{}
	original := sampleControls()
	sheet := b.BuildControlMatrix(original)

	parsed, err := b.ParseControlMatrix(sheet)
	require.NoError(t, err)
	require.Len(t, parsed, len(original))

	for i, orig := range original {
		got := parsed[i]
		assert.Equal(t, orig.ReferenceID, got.ReferenceID, "row %d ReferenceID", i)
		assert.Equal(t, orig.Name, got.Name, "row %d Name", i)
		assert.Equal(t, orig.Status, got.Status, "row %d Status", i)
		assert.Equal(t, orig.RiskLevel, got.RiskLevel, "row %d RiskLevel", i)
		assert.Equal(t, orig.Category, got.Category, "row %d Category", i)

		// Compare framework codes by Code field only (parse doesn't recover ID/Framework/Name).
		assert.Len(t, got.FrameworkCodes, len(orig.FrameworkCodes), "row %d FrameworkCodes count", i)
		for j, fc := range orig.FrameworkCodes {
			assert.Equal(t, fc.Code, got.FrameworkCodes[j].Code, "row %d FC %d Code", i, j)
		}

		// Compare dates.
		assert.Equal(t, formatDate(orig.ImplementedDate), formatDate(got.ImplementedDate), "row %d ImplementedDate", i)
		assert.Equal(t, formatDate(orig.TestedDate), formatDate(got.TestedDate), "row %d TestedDate", i)
	}
}

func TestParseControlMatrix_InvalidHeaders(t *testing.T) {
	t.Parallel()
	b := &ControlMatrixBuilder{}
	sheet := &SheetData{
		Headers: []string{"Wrong", "Headers"},
		Rows:    [][]string{{"a", "b"}},
	}
	_, err := b.ParseControlMatrix(sheet)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required header")
}

func TestParseEvidenceTaskSheet_RoundTrip(t *testing.T) {
	t.Parallel()
	b := &ControlMatrixBuilder{}
	original := sampleEvidenceTasks()
	sheet := b.BuildEvidenceTaskSheet(original)

	parsed, err := b.ParseEvidenceTaskSheet(sheet)
	require.NoError(t, err)
	require.Len(t, parsed, len(original))

	for i, orig := range original {
		got := parsed[i]
		assert.Equal(t, orig.ReferenceID, got.ReferenceID, "row %d ReferenceID", i)
		assert.Equal(t, orig.Name, got.Name, "row %d Name", i)
		assert.Equal(t, orig.Status, got.Status, "row %d Status", i)
		assert.Equal(t, orig.Priority, got.Priority, "row %d Priority", i)
		assert.Equal(t, orig.CollectionInterval, got.CollectionInterval, "row %d CollectionInterval", i)
		assert.Equal(t, orig.Framework, got.Framework, "row %d Framework", i)
		assert.Equal(t, orig.Category, got.Category, "row %d Category", i)
		assert.Equal(t, formatDate(orig.LastCollected), formatDate(got.LastCollected), "row %d LastCollected", i)
		assert.Equal(t, formatDate(orig.NextDue), formatDate(got.NextDue), "row %d NextDue", i)
	}
}

// --- CSV Tests ---

func TestSheetData_ToCSV(t *testing.T) {
	t.Parallel()
	b := &ControlMatrixBuilder{}
	sheet := b.BuildControlMatrix(sampleControls())
	csvStr := sheet.ToCSV()

	assert.Contains(t, csvStr, "Reference ID,Name,Status,Risk Level,Framework Codes,Category,Implemented Date,Tested Date")
	assert.Contains(t, csvStr, "CC-06.1,Logical Access Security,implemented,medium,CC6.1,Common Criteria,2025-03-15,2025-06-01")
	assert.Contains(t, csvStr, "SO-19,Vulnerability Management,in_progress,high,,Security Operations,,")
}

func TestSheetDataFromCSV_RoundTrip(t *testing.T) {
	t.Parallel()
	b := &ControlMatrixBuilder{}
	original := b.BuildControlMatrix(sampleControls())
	csvStr := original.ToCSV()

	parsed, err := SheetDataFromCSV(csvStr)
	require.NoError(t, err)
	assert.Equal(t, original.Headers, parsed.Headers)
	require.Len(t, parsed.Rows, len(original.Rows))

	// Re-export and compare CSV strings for exact round-trip.
	csvStr2 := parsed.ToCSV()
	assert.Equal(t, csvStr, csvStr2)
}

func TestSheetDataFromCSV_Empty(t *testing.T) {
	t.Parallel()
	_, err := SheetDataFromCSV("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestSheetDataFromCSV_HeaderOnly(t *testing.T) {
	t.Parallel()
	csvStr := "Reference ID,Name,Status\n"
	sheet, err := SheetDataFromCSV(csvStr)
	require.NoError(t, err)
	assert.Equal(t, []string{"Reference ID", "Name", "Status"}, sheet.Headers)
	assert.Empty(t, sheet.Rows)
}
