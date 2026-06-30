package assessment

import (
	"fmt"
	"testing"

	"github.com/safe-ai/excel-brushing-tool/internal/upload"
	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

// =============================================================================
// Task 1: Bug Condition Exploration Tests
// These tests MUST FAIL on unfixed code to confirm bugs exist.
// =============================================================================

// TestPBT_BugCondition_IssueCard_GapRow tests Bug 2: gap rows in "多表格混在同一 sheet"
// should have Highlights == nil. On unfixed code, the gap row gets all-column highlights.
//
// **Validates: Requirements 1.2, 2.2**
func TestPBT_BugCondition_IssueCard_GapRow(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate multi-block SheetData: 2 data blocks separated by 2+ empty rows
		cols := rapid.IntRange(2, 6).Draw(t, "cols")
		block1Rows := rapid.IntRange(3, 8).Draw(t, "block1_rows")
		gapRows := rapid.IntRange(2, 4).Draw(t, "gap_rows")
		block2Rows := rapid.IntRange(3, 8).Draw(t, "block2_rows")

		headers := make([]string, cols)
		for i := range headers {
			headers[i] = fmt.Sprintf("Col%d", i+1)
		}

		var rows [][]upload.CellValue

		// Block 1: non-empty data rows
		for r := 0; r < block1Rows; r++ {
			row := make([]upload.CellValue, cols)
			for c := range row {
				val := rapid.StringMatching(`[A-Za-z]{3,8}`).Draw(t, fmt.Sprintf("b1_r%d_c%d", r, c))
				row[c] = upload.CellValue{Raw: val, IsEmpty: false}
			}
			rows = append(rows, row)
		}

		// Gap: 2+ consecutive empty rows
		for r := 0; r < gapRows; r++ {
			row := make([]upload.CellValue, cols)
			for c := range row {
				row[c] = upload.CellValue{Raw: "", IsEmpty: true}
			}
			rows = append(rows, row)
		}

		// Block 2: non-empty data rows
		for r := 0; r < block2Rows; r++ {
			row := make([]upload.CellValue, cols)
			for c := range row {
				val := rapid.StringMatching(`[A-Za-z]{3,8}`).Draw(t, fmt.Sprintf("b2_r%d_c%d", r, c))
				row[c] = upload.CellValue{Raw: val, IsEmpty: false}
			}
			rows = append(rows, row)
		}

		data := &upload.SheetData{
			Headers:        headers,
			Rows:           rows,
			RowCount:       len(rows),
			ColCount:       cols,
			HeaderRowIndex: 0,
		}

		// Verify multi-table is detected
		if !hasMultipleTables(data) {
			t.Skip("multi-table not detected, skip")
		}

		// Call buildSingleStructureExamples for "多表格混在同一 sheet"
		examples := buildSingleStructureExamples(data, "多表格混在同一 sheet")

		// Find gap rows (label = "（空白列）")
		for _, ex := range examples {
			if ex.Label == "（空白列）" {
				// Bug condition: gap rows should NOT be highlighted
				assert.Nil(t, ex.Highlights,
					"Gap row (label='（空白列）') should have Highlights == nil, but got %v", ex.Highlights)
			}
		}
	})
}

// TestPBT_BugCondition_IssueCard_FormatGroups tests Bug 4: format consistency examples
// should have multiple distinct Label groups (one per mixed column, up to 5) and
// each example should have non-nil FormatLabels.
//
// **Validates: Requirements 1.4, 2.4**
func TestPBT_BugCondition_IssueCard_FormatGroups(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate SheetData with 3+ columns having mixed formats
		cols := rapid.IntRange(3, 6).Draw(t, "cols")
		rowCount := rapid.IntRange(10, 30).Draw(t, "row_count")

		headers := make([]string, cols)
		for i := range headers {
			headers[i] = fmt.Sprintf("Field%d", i+1)
		}

		rows := make([][]upload.CellValue, rowCount)
		// Ensure at least 2 columns have mixed formats:
		// - Column 0: 70% numeric, 30% text
		// - Column 1: 60% date, 40% numeric
		// - Column 2: 75% text, 25% numeric
		// Other columns: uniform text (no mix)
		for r := 0; r < rowCount; r++ {
			row := make([]upload.CellValue, cols)

			// Column 0: 70% numeric, 30% text
			if r < rowCount*7/10 {
				num := rapid.IntRange(1, 9999).Draw(t, fmt.Sprintf("c0_num_%d", r))
				row[0] = upload.CellValue{Raw: fmt.Sprintf("%d", num), IsEmpty: false}
			} else {
				txt := rapid.StringMatching(`[A-Z][a-z]{3,7}`).Draw(t, fmt.Sprintf("c0_txt_%d", r))
				row[0] = upload.CellValue{Raw: txt, IsEmpty: false}
			}

			// Column 1: 60% date, 40% numeric
			if r < rowCount*6/10 {
				m := rapid.IntRange(1, 12).Draw(t, fmt.Sprintf("c1_m_%d", r))
				d := rapid.IntRange(1, 28).Draw(t, fmt.Sprintf("c1_d_%d", r))
				row[1] = upload.CellValue{Raw: fmt.Sprintf("2024-%02d-%02d", m, d), IsEmpty: false}
			} else {
				num := rapid.IntRange(100, 999).Draw(t, fmt.Sprintf("c1_num_%d", r))
				row[1] = upload.CellValue{Raw: fmt.Sprintf("%d", num), IsEmpty: false}
			}

			// Column 2: 75% text, 25% numeric
			if r < rowCount*75/100 {
				txt := rapid.StringMatching(`[A-Z][a-z]{4,8}`).Draw(t, fmt.Sprintf("c2_txt_%d", r))
				row[2] = upload.CellValue{Raw: txt, IsEmpty: false}
			} else {
				num := rapid.IntRange(1, 500).Draw(t, fmt.Sprintf("c2_num_%d", r))
				row[2] = upload.CellValue{Raw: fmt.Sprintf("%d", num), IsEmpty: false}
			}

			// Remaining columns: uniform text
			for c := 3; c < cols; c++ {
				txt := rapid.StringMatching(`[a-z]{5,10}`).Draw(t, fmt.Sprintf("c%d_r%d", c, r))
				row[c] = upload.CellValue{Raw: txt, IsEmpty: false}
			}

			rows[r] = row
		}

		data := &upload.SheetData{
			Headers:        headers,
			Rows:           rows,
			RowCount:       rowCount,
			ColCount:       cols,
			HeaderRowIndex: 0,
		}

		// Verify we actually have mixed format columns
		mixedCols := findMixedFormatColumns(data)
		if len(mixedCols) < 2 {
			t.Skip("not enough mixed format columns detected")
		}

		// Call buildFormatConsistencyExamples
		examples := buildFormatConsistencyExamples(data)
		if len(examples) == 0 {
			t.Fatal("expected format consistency examples but got none")
		}

		// Bug 4 assertion: there should be multiple distinct Label values
		// (one per mixed column, up to 5)
		labelSet := make(map[string]bool)
		for _, ex := range examples {
			if ex.Label != "" {
				labelSet[ex.Label] = true
			}
		}

		expectedGroups := len(mixedCols)
		if expectedGroups > 5 {
			expectedGroups = 5
		}
		assert.GreaterOrEqual(t, len(labelSet), 2,
			"Expected at least 2 distinct Label groups for %d mixed columns, got %d labels: %v",
			len(mixedCols), len(labelSet), labelSet)

		// Bug 4 assertion: each example should have non-nil FormatLabels
		for _, ex := range examples {
			assert.NotNil(t, ex.FormatLabels,
				"Expected non-nil FormatLabels on format consistency example (row %d), got nil",
				ex.RowNumber)
		}
	})
}

// =============================================================================
// Task 3: Preservation Property Tests
// These tests MUST PASS on unfixed code to confirm baseline behavior.
// =============================================================================

// TestPBT_Preservation_IssueCard_StructureHighlights tests that for structure
// problems OTHER than "多表格混在同一 sheet" (e.g. "合併儲存格"), examples retain
// their existing highlight behavior (non-nil highlights on relevant cells).
//
// **Validates: Requirements 3.2**
func TestPBT_Preservation_IssueCard_StructureHighlights(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate SheetData with merged cells to trigger "合併儲存格"
		cols := rapid.IntRange(4, 8).Draw(t, "cols")
		rowCount := rapid.IntRange(5, 20).Draw(t, "rows")

		headers := make([]string, cols)
		for i := range headers {
			headers[i] = fmt.Sprintf("Header%d", i+1)
		}

		rows := make([][]upload.CellValue, rowCount)
		for r := range rows {
			row := make([]upload.CellValue, cols)
			for c := range row {
				val := rapid.StringMatching(`[A-Za-z]{3,8}`).Draw(t, fmt.Sprintf("v_%d_%d", r, c))
				row[c] = upload.CellValue{Raw: val, IsEmpty: false}
			}
			rows[r] = row
		}

		// Add merged cells to trigger the "合併儲存格" problem.
		// MergedRange uses 1-based coordinates (as excelize returns).
		// Span at least 2 columns so the detection code picks it up.
		mergeRow := rapid.IntRange(0, rowCount-1).Draw(t, "merge_row")
		mergeStartCol := rapid.IntRange(0, cols-3).Draw(t, "merge_start_col")
		mergeEndCol := rapid.IntRange(mergeStartCol+1, cols-1).Draw(t, "merge_end_col")

		// Convert to 1-based for the MergedRange (excelize convention)
		headerRowIndex := 0
		sheetRow1Based := mergeRow + headerRowIndex + 2 // data row to sheet row (1-based)

		data := &upload.SheetData{
			Headers:        headers,
			Rows:           rows,
			RowCount:       rowCount,
			ColCount:       cols,
			HeaderRowIndex: headerRowIndex,
			MergedCells: []upload.MergedRange{
				{
					StartRow: sheetRow1Based,
					EndRow:   sheetRow1Based,
					StartCol: mergeStartCol + 1, // 1-based
					EndCol:   mergeEndCol + 1,   // 1-based
				},
			},
		}

		// Verify "合併儲存格" is detected
		problems := detectStructureProblems(data)
		hasMerge := false
		for _, p := range problems {
			if p == "合併儲存格" {
				hasMerge = true
				break
			}
		}
		if !hasMerge {
			t.Skip("合併儲存格 not detected, skip")
		}

		// Call buildSingleStructureExamples for "合併儲存格"
		examples := buildSingleStructureExamples(data, "合併儲存格")

		// Preservation: merged cell examples should have non-nil Highlights
		// (highlighting the merged range cells in red). This behavior must be
		// preserved even after the gap row fix.
		if len(examples) > 0 {
			hasHighlights := false
			for _, ex := range examples {
				if len(ex.Highlights) > 0 {
					hasHighlights = true
					break
				}
			}
			assert.True(t, hasHighlights,
				"合併儲存格 examples should have non-empty Highlights to show merged cell positions")
		}
	})
}

// TestPBT_Preservation_IssueCard_NoFormatIssueWhenConsistent tests that when ALL
// columns have ≥80% format consistency, no format consistency issue is generated.
//
// **Validates: Requirements 3.5**
func TestPBT_Preservation_IssueCard_NoFormatIssueWhenConsistent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate SheetData where all columns are highly consistent (all numeric)
		cols := rapid.IntRange(2, 6).Draw(t, "cols")
		rowCount := rapid.IntRange(10, 30).Draw(t, "rows")

		headers := make([]string, cols)
		for i := range headers {
			headers[i] = fmt.Sprintf("NumCol%d", i+1)
		}

		rows := make([][]upload.CellValue, rowCount)
		for r := range rows {
			row := make([]upload.CellValue, cols)
			for c := range row {
				// All numeric — 100% consistent
				num := rapid.IntRange(1, 99999).Draw(t, fmt.Sprintf("n_%d_%d", r, c))
				row[c] = upload.CellValue{Raw: fmt.Sprintf("%d", num), IsEmpty: false}
			}
			rows[r] = row
		}

		data := &upload.SheetData{
			Headers:        headers,
			Rows:           rows,
			RowCount:       rowCount,
			ColCount:       cols,
			HeaderRowIndex: 0,
		}

		// Verify: no mixed format columns should be detected
		mixedCols := findMixedFormatColumns(data)
		assert.Empty(t, mixedCols,
			"Expected no mixed format columns when all data is numeric, got: %v", mixedCols)

		// Verify: buildFormatConsistencyExamples returns nil
		examples := buildFormatConsistencyExamples(data)
		assert.Nil(t, examples,
			"Expected nil format consistency examples when all columns are consistent, got %d examples", len(examples))
	})
}

// TestPBT_Preservation_IssueCard_NoFormatLabelsOnOtherIssues tests that for
// non-format-consistency issues (e.g., row_completeness), examples do NOT have
// FormatLabels field set.
//
// **Validates: Requirements 3.4**
func TestPBT_Preservation_IssueCard_NoFormatLabelsOnOtherIssues(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate SheetData with some empty cells to trigger row_completeness issues
		cols := rapid.IntRange(3, 8).Draw(t, "cols")
		rowCount := rapid.IntRange(10, 30).Draw(t, "rows")

		headers := make([]string, cols)
		for i := range headers {
			headers[i] = fmt.Sprintf("Col%d", i+1)
		}

		rows := make([][]upload.CellValue, rowCount)
		for r := range rows {
			row := make([]upload.CellValue, cols)
			for c := range row {
				// Make ~40% of cells empty to trigger completeness issues
				isEmpty := rapid.Float64Range(0, 1).Draw(t, fmt.Sprintf("e_%d_%d", r, c)) < 0.4
				if isEmpty {
					row[c] = upload.CellValue{Raw: "", IsEmpty: true}
				} else {
					val := rapid.StringMatching(`[A-Za-z]{3,8}`).Draw(t, fmt.Sprintf("v_%d_%d", r, c))
					row[c] = upload.CellValue{Raw: val, IsEmpty: false}
				}
			}
			rows[r] = row
		}

		data := &upload.SheetData{
			Headers:        headers,
			Rows:           rows,
			RowCount:       rowCount,
			ColCount:       cols,
			HeaderRowIndex: 0,
		}

		// Calculate scores to trigger issues
		scores := IndicatorScores{
			RowCompleteness:    30, // Force low to trigger row completeness issue
			ColumnCompleteness: 90,
			FormatConsistency:  90,
			DuplicateSimilar:   90,
			TableStructure:     100,
			AIQueryReadiness:   90,
		}

		issues := DetectIssues(data, scores)

		// Find non-format-consistency issues
		for _, issue := range issues {
			if issue.Indicator == "format_consistency" {
				continue
			}
			// For all other issues, FormatLabels should be nil on every example
			for _, ex := range issue.Examples {
				assert.Nil(t, ex.FormatLabels,
					"Non-format-consistency issue '%s' (indicator: %s) example at row %d should not have FormatLabels, got: %v",
					issue.Title, issue.Indicator, ex.RowNumber, ex.FormatLabels)
			}
		}
	})
}
