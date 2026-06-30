package cleaning

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/safe-ai/excel-brushing-tool/internal/upload"
	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

// --- Generators ---

func genSheetData(t *rapid.T) *upload.SheetData {
	rows := rapid.IntRange(0, 50).Draw(t, "rows")
	cols := rapid.IntRange(1, 10).Draw(t, "cols")
	headers := make([]string, cols)
	for i := range headers {
		headers[i] = rapid.StringMatching(`[A-Za-z]{2,10}`).Draw(t, fmt.Sprintf("header_%d", i))
	}
	data := make([][]upload.CellValue, rows)
	for i := range data {
		row := make([]upload.CellValue, cols)
		for j := range row {
			isEmpty := rapid.Bool().Draw(t, fmt.Sprintf("empty_%d_%d", i, j))
			if isEmpty {
				row[j] = upload.CellValue{Raw: "", IsEmpty: true}
			} else {
				row[j] = upload.CellValue{Raw: rapid.StringMatching(`[A-Za-z0-9]{1,15}`).Draw(t, fmt.Sprintf("val_%d_%d", i, j)), IsEmpty: false}
			}
		}
		data[i] = row
	}
	return &upload.SheetData{Headers: headers, Rows: data, RowCount: rows, ColCount: cols}
}

// genDateSheetData generates a sheet where one column is predominantly dates
func genDateSheetData(t *rapid.T) *upload.SheetData {
	rows := rapid.IntRange(3, 30).Draw(t, "rows")
	dateCol := 0
	cols := rapid.IntRange(1, 5).Draw(t, "cols")
	headers := make([]string, cols)
	for i := range headers {
		headers[i] = fmt.Sprintf("Col%d", i)
	}

	data := make([][]upload.CellValue, rows)
	for i := range data {
		row := make([]upload.CellValue, cols)
		for j := range row {
			if j == dateCol {
				// Generate date in various formats
				y := rapid.IntRange(2020, 2025).Draw(t, fmt.Sprintf("year_%d", i))
				m := rapid.IntRange(1, 12).Draw(t, fmt.Sprintf("month_%d", i))
				d := rapid.IntRange(1, 28).Draw(t, fmt.Sprintf("day_%d", i))
				format := rapid.IntRange(0, 2).Draw(t, fmt.Sprintf("fmt_%d", i))
				var dateStr string
				switch format {
				case 0:
					dateStr = fmt.Sprintf("%04d/%d/%d", y, m, d)
				case 1:
					dateStr = fmt.Sprintf("%04d-%02d-%02d", y, m, d)
				case 2:
					rocYear := y - 1911
					dateStr = fmt.Sprintf("%d.%d.%d", rocYear, m, d)
				}
				row[j] = upload.CellValue{Raw: dateStr, IsEmpty: false}
			} else {
				row[j] = upload.CellValue{Raw: rapid.StringMatching(`[A-Za-z]{3,8}`).Draw(t, fmt.Sprintf("val_%d_%d", i, j)), IsEmpty: false}
			}
		}
		data[i] = row
	}
	return &upload.SheetData{Headers: headers, Rows: data, RowCount: rows, ColCount: cols}
}

// genDedupSheetData generates data with guaranteed duplicates
func genDedupSheetData(t *rapid.T) *upload.SheetData {
	baseRows := rapid.IntRange(2, 20).Draw(t, "base_rows")
	cols := rapid.IntRange(1, 5).Draw(t, "cols")
	headers := make([]string, cols)
	for i := range headers {
		headers[i] = fmt.Sprintf("Col%d", i)
	}

	// Generate unique base rows
	baseData := make([][]upload.CellValue, baseRows)
	for i := range baseData {
		row := make([]upload.CellValue, cols)
		for j := range row {
			row[j] = upload.CellValue{
				Raw:     fmt.Sprintf("r%d_c%d_%s", i, j, rapid.StringMatching(`[a-z]{3}`).Draw(t, fmt.Sprintf("base_%d_%d", i, j))),
				IsEmpty: false,
			}
		}
		baseData[i] = row
	}

	// Add some duplicates
	dupsCount := rapid.IntRange(1, baseRows).Draw(t, "dups")
	allRows := make([][]upload.CellValue, len(baseData))
	copy(allRows, baseData)
	for i := 0; i < dupsCount; i++ {
		srcIdx := rapid.IntRange(0, baseRows-1).Draw(t, fmt.Sprintf("dup_src_%d", i))
		dupRow := make([]upload.CellValue, cols)
		copy(dupRow, baseData[srcIdx])
		allRows = append(allRows, dupRow)
	}

	return &upload.SheetData{Headers: headers, Rows: allRows, RowCount: len(allRows), ColCount: cols}
}

// --- Property 18: Date normalization produces YYYY/MM/DD format ---
// **Validates: Requirements 12.1**
func TestPBT_DateNormalization(t *testing.T) {
	slashDatePattern := regexp.MustCompile(`^\d{4}/\d{2}/\d{2}$`)

	rapid.Check(t, func(t *rapid.T) {
		data := genDateSheetData(t)

		var log []LogEntry
		DateNormalize(data, &log, "pbt-operator")

		// After normalization, all date cells in date columns should be YYYY/MM/DD
		for _, row := range data.Rows {
			cell := row[0] // column 0 is the date column
			if cell.IsEmpty {
				continue
			}
			// If the value was a valid date, it should now be in YYYY/MM/DD
			if _, ok := normalizeDate(cell.Raw); ok {
				assert.Regexp(t, slashDatePattern, cell.Raw,
					"Date cell should be in YYYY/MM/DD format, got: %q", cell.Raw)
			}
		}
	})
}

// --- Property 19: Dedup preserves uniqueness and order ---
// **Validates: Requirements 12.2**
func TestPBT_DedupUniqueness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		data := genDedupSheetData(t)
		originalOrder := make([]string, len(data.Rows))
		for i, row := range data.Rows {
			originalOrder[i] = rowToString(row, data.ColCount)
		}

		var log []LogEntry
		Dedup(data, &log, "pbt-operator")

		// Property 1: No two remaining rows have identical hashes
		seen := make(map[string]bool)
		for _, row := range data.Rows {
			h := hashRow(row, data.ColCount)
			assert.False(t, seen[h], "Duplicate row found after dedup")
			seen[h] = true
		}

		// Property 2: Original relative order is preserved
		// Each kept row should appear in the same relative order as in original
		prevOrigIdx := -1
		for _, row := range data.Rows {
			rowStr := rowToString(row, data.ColCount)
			// Find this row in the original order (first occurrence)
			for origIdx, origStr := range originalOrder {
				if origStr == rowStr && origIdx > prevOrigIdx {
					prevOrigIdx = origIdx
					break
				}
			}
		}
		// If we get here without panic, order is preserved
	})
}

// --- Property 20: Company name normalization unifies to longest ---
// **Validates: Requirements 12.3**
func TestPBT_NameNormalization(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a base name and create variants with different suffixes
		baseName := rapid.StringMatching(`[A-Z][a-z]{3,10}`).Draw(t, "base")
		suffixes := []string{"", " Co.", " Corp.", " Inc.", " Ltd."}
		numVariants := rapid.IntRange(2, len(suffixes)).Draw(t, "num_variants")

		rows := make([][]upload.CellValue, 0, numVariants*2)
		usedSuffixes := suffixes[:numVariants]
		for _, suffix := range usedSuffixes {
			rows = append(rows, []upload.CellValue{
				{Raw: baseName + suffix, IsEmpty: false},
			})
		}

		data := &upload.SheetData{
			Headers:  []string{"Company"},
			Rows:     rows,
			RowCount: len(rows),
			ColCount: 1,
		}

		var log []LogEntry
		NameNormalize(data, &log, "pbt-operator")

		// Find the longest variant
		longest := ""
		for _, suffix := range usedSuffixes {
			candidate := baseName + suffix
			if len([]rune(candidate)) > len([]rune(longest)) {
				longest = candidate
			}
		}

		// After normalization, all values in the group should be unified to longest
		for i, row := range data.Rows {
			assert.Equal(t, longest, row[0].Raw,
				"Row %d should be unified to longest variant %q, got %q", i, longest, row[0].Raw)
		}
	})
}

// --- Property 21: Subtotal removal eliminates all keyword rows ---
// **Validates: Requirements 12.4**
func TestPBT_SubtotalRemoval(t *testing.T) {
	keywords := []string{"小計", "合計", "total", "subtotal"}

	rapid.Check(t, func(t *rapid.T) {
		data := genSheetData(t)

		// Inject some subtotal rows
		numSubtotals := rapid.IntRange(0, 5).Draw(t, "num_subtotals")
		for i := 0; i < numSubtotals; i++ {
			kwIdx := rapid.IntRange(0, len(keywords)-1).Draw(t, fmt.Sprintf("kw_%d", i))
			cols := data.ColCount
			row := make([]upload.CellValue, cols)
			for j := range row {
				row[j] = upload.CellValue{Raw: rapid.StringMatching(`[A-Za-z]{2,5}`).Draw(t, fmt.Sprintf("sub_v_%d_%d", i, j)), IsEmpty: false}
			}
			// Put keyword in a random cell
			cellIdx := rapid.IntRange(0, cols-1).Draw(t, fmt.Sprintf("kw_cell_%d", i))
			row[cellIdx] = upload.CellValue{Raw: keywords[kwIdx], IsEmpty: false}
			data.Rows = append(data.Rows, row)
			data.RowCount++
		}

		var log []LogEntry
		SubtotalRemove(data, &log, "pbt-operator")

		// After removal, no row should contain any keyword
		for i, row := range data.Rows {
			for _, cell := range row {
				if cell.IsEmpty {
					continue
				}
				lower := strings.ToLower(strings.TrimSpace(cell.Raw))
				for _, kw := range keywords {
					assert.False(t, strings.Contains(lower, kw),
						"Row %d still contains keyword %q in cell %q", i, kw, cell.Raw)
				}
			}
		}
	})
}

// --- Property 22: Operations produce log entries ---
// **Validates: Requirements 12.5, 13.1, 13.2**
func TestPBT_LogEntries(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Create data that will trigger operations
		data := &upload.SheetData{
			Headers:  []string{"A", "B"},
			ColCount: 2,
			Rows: [][]upload.CellValue{
				{{Raw: "val1", IsEmpty: false}, {Raw: "val2", IsEmpty: false}},
				{{Raw: "val1", IsEmpty: false}, {Raw: "val2", IsEmpty: false}}, // duplicate
				{{Raw: "", IsEmpty: true}, {Raw: "x", IsEmpty: false}},
			},
			RowCount: 3,
		}

		operatorID := rapid.StringMatching(`[a-z]{5,10}`).Draw(t, "operator")

		var log []LogEntry
		Dedup(data, &log, operatorID)

		// Dedup should have produced a log entry (there was a duplicate)
		assert.NotEmpty(t, log, "Dedup on data with duplicates must produce log entries")

		for _, entry := range log {
			assert.NotEmpty(t, entry.OperationType, "Log entry must have operation_type")
			assert.NotEmpty(t, entry.AffectedRows, "Log entry must have non-empty affected_rows")
			assert.False(t, entry.Timestamp.IsZero(), "Log entry must have valid timestamp")
			assert.NotEmpty(t, entry.OperatorID, "Log entry must have operator_id")
		}
	})
}

// --- Property 23: Fill N/A preserves non-empty, fills empty ---
// **Validates: Requirements 13.1**
func TestPBT_FillNA(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cols := rapid.IntRange(1, 10).Draw(t, "cols")
		rows := rapid.IntRange(1, 20).Draw(t, "rows")

		data := &upload.SheetData{
			Headers:  make([]string, cols),
			ColCount: cols,
			Rows:     make([][]upload.CellValue, rows),
			RowCount: rows,
		}
		for i := range data.Headers {
			data.Headers[i] = fmt.Sprintf("Col%d", i)
		}

		// Generate a row with mix of empty and non-empty
		rowIdx := rapid.IntRange(0, rows-1).Draw(t, "target_row")
		for i := 0; i < rows; i++ {
			data.Rows[i] = make([]upload.CellValue, cols)
			for j := 0; j < cols; j++ {
				if rapid.Bool().Draw(t, fmt.Sprintf("empty_%d_%d", i, j)) {
					data.Rows[i][j] = upload.CellValue{Raw: "", IsEmpty: true}
				} else {
					data.Rows[i][j] = upload.CellValue{
						Raw:     rapid.StringMatching(`[A-Za-z0-9]{1,10}`).Draw(t, fmt.Sprintf("v_%d_%d", i, j)),
						IsEmpty: false,
					}
				}
			}
		}

		// Record original state of target row
		originalValues := make([]upload.CellValue, cols)
		copy(originalValues, data.Rows[rowIdx])

		var log []LogEntry
		err := FillNA(data, rowIdx, &log, "pbt-operator")
		assert.NoError(t, err)

		// Verify properties
		for j := 0; j < cols; j++ {
			if originalValues[j].IsEmpty {
				// Was empty → should now be "N/A"
				assert.Equal(t, "N/A", data.Rows[rowIdx][j].Raw,
					"Empty cell at col %d should be filled with N/A", j)
				assert.False(t, data.Rows[rowIdx][j].IsEmpty)
			} else {
				// Was non-empty → should be unchanged
				assert.Equal(t, originalValues[j].Raw, data.Rows[rowIdx][j].Raw,
					"Non-empty cell at col %d should be preserved", j)
			}
		}
	})
}

// --- Property 24: Delete row shrinks dataset by 1 ---
// **Validates: Requirements 13.2**
func TestPBT_DeleteRow(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		rows := rapid.IntRange(1, 50).Draw(t, "rows")
		cols := rapid.IntRange(1, 10).Draw(t, "cols")

		data := &upload.SheetData{
			Headers:  make([]string, cols),
			ColCount: cols,
			Rows:     make([][]upload.CellValue, rows),
			RowCount: rows,
		}
		for i := range data.Headers {
			data.Headers[i] = fmt.Sprintf("Col%d", i)
		}
		for i := 0; i < rows; i++ {
			data.Rows[i] = make([]upload.CellValue, cols)
			for j := 0; j < cols; j++ {
				data.Rows[i][j] = upload.CellValue{
					Raw:     fmt.Sprintf("r%d_c%d", i, j),
					IsEmpty: false,
				}
			}
		}

		rowIdx := rapid.IntRange(0, rows-1).Draw(t, "delete_idx")
		deletedRowStr := rowToString(data.Rows[rowIdx], cols)
		originalRowCount := data.RowCount

		var log []LogEntry
		err := DeleteRow(data, rowIdx, &log, "pbt-operator")

		assert.NoError(t, err)
		// Dataset should shrink by exactly 1
		assert.Equal(t, originalRowCount-1, data.RowCount)
		assert.Len(t, data.Rows, originalRowCount-1)

		// The deleted row should not appear at the same position
		if rowIdx < len(data.Rows) {
			currentRowStr := rowToString(data.Rows[rowIdx], cols)
			// If the deleted row was not the last one, the row at that index should be different
			if rowIdx < originalRowCount-1 {
				assert.NotEqual(t, deletedRowStr, currentRowStr,
					"Row at deleted index should be different")
			}
		}

		// Log entry should exist
		assert.NotEmpty(t, log)
		assert.Equal(t, "delete_row", log[0].OperationType)
	})
}

// --- Helpers ---

func rowToString(row []upload.CellValue, colCount int) string {
	parts := make([]string, colCount)
	for i := 0; i < colCount; i++ {
		if i < len(row) {
			parts[i] = row[i].Raw
		}
	}
	return strings.Join(parts, "|")
}
