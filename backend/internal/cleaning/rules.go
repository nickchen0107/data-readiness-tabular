package cleaning

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/safe-ai/excel-brushing-tool/internal/upload"
)

// Supported date patterns for parsing
var (
	dateISOSlash = regexp.MustCompile(`^(\d{4})[/](\d{1,2})[/](\d{1,2})$`)
	dateISODash  = regexp.MustCompile(`^(\d{4})[-](\d{1,2})[-](\d{1,2})$`)
	dateROC      = regexp.MustCompile(`^(\d{2,3})\.(\d{1,2})\.(\d{1,2})$`)
	// MM-DD-YY or MM/DD/YY (2-digit year, US format)
	dateMDY2 = regexp.MustCompile(`^(\d{1,2})[-/](\d{1,2})[-/](\d{2})$`)
)

// DateNormalize converts all date values to YYYY/MM/DD format.
// Detects date columns and parses: yyyy/MM/dd, yyyy-MM-dd, ROC yyy.M.d
func DateNormalize(data *upload.SheetData, log *[]LogEntry, operatorID string) {
	if data == nil || len(data.Rows) == 0 {
		return
	}

	// For each column, check if it's predominantly a date column
	for col := 0; col < data.ColCount; col++ {
		if !isDateColumn(data, col) {
			continue
		}

		var affectedRows []int
		for rowIdx, row := range data.Rows {
			if col >= len(row) || row[col].IsEmpty {
				continue
			}

			normalized, ok := normalizeDate(row[col].Raw)
			if ok && normalized != row[col].Raw {
				data.Rows[rowIdx][col].Raw = normalized
				affectedRows = append(affectedRows, rowIdx)
			}
		}

		if len(affectedRows) > 0 {
			*log = append(*log, LogEntry{
				OperationType: "date_normalize",
				AffectedRows:  affectedRows,
				Timestamp:     time.Now(),
				OperatorID:    operatorID,
				Details:       fmt.Sprintf("統一欄位 %d 的日期格式為 YYYY/MM/DD", col),
			})
		}
	}
}

// Dedup removes exact duplicate rows, keeping the first occurrence.
// Uses SHA-256 full-row hash and maintains original order.
func Dedup(data *upload.SheetData, log *[]LogEntry, operatorID string) {
	if data == nil || len(data.Rows) == 0 {
		return
	}

	seen := make(map[string]bool)
	var kept [][]upload.CellValue
	var removedRows []int

	for i, row := range data.Rows {
		h := hashRow(row, data.ColCount)
		if seen[h] {
			removedRows = append(removedRows, i)
		} else {
			seen[h] = true
			kept = append(kept, row)
		}
	}

	if len(removedRows) > 0 {
		data.Rows = kept
		data.RowCount = len(kept)

		*log = append(*log, LogEntry{
			OperationType: "dedup",
			AffectedRows:  removedRows,
			Timestamp:     time.Now(),
			OperatorID:    operatorID,
			Details:       fmt.Sprintf("移除 %d 筆完全重複的資料列", len(removedRows)),
		})
	}
}

// NameNormalize normalizes company names by removing suffix variants.
// Groups by normalized name and unifies to the longest variant.
func NameNormalize(data *upload.SheetData, log *[]LogEntry, operatorID string) {
	if data == nil || len(data.Rows) == 0 {
		return
	}

	suffixes := []string{
		"股份有限公司",
		"有限公司",
		"公司",
		"Company",
		"Corp.",
		"Inc.",
		"Ltd.",
		"Co.",
	}

	// For each text column, attempt name normalization
	for col := 0; col < data.ColCount; col++ {
		if !isTextColumn(data, col) {
			continue
		}

		// Build groups: normalized_name → list of (original_value, row_indices)
		type valueInfo struct {
			value    string
			rowIdxes []int
		}
		groups := make(map[string]*[]valueInfo)

		for rowIdx, row := range data.Rows {
			if col >= len(row) || row[col].IsEmpty {
				continue
			}
			val := strings.TrimSpace(row[col].Raw)
			normalized := removeSuffixes(val, suffixes)
			normalizedKey := strings.ToLower(normalized)

			if _, exists := groups[normalizedKey]; !exists {
				groups[normalizedKey] = &[]valueInfo{}
			}

			// Check if this value variant already exists
			found := false
			for i, vi := range *groups[normalizedKey] {
				if vi.value == val {
					(*groups[normalizedKey])[i].rowIdxes = append((*groups[normalizedKey])[i].rowIdxes, rowIdx)
					found = true
					break
				}
			}
			if !found {
				*groups[normalizedKey] = append(*groups[normalizedKey], valueInfo{
					value:    val,
					rowIdxes: []int{rowIdx},
				})
			}
		}

		// For each group with multiple variants, unify to longest
		var affectedRows []int
		for _, variants := range groups {
			if len(*variants) <= 1 {
				continue
			}

			// Find most frequent variant (canonical) — reflects actual usage habit
			canonical := ""
			maxCount := 0
			for _, vi := range *variants {
				if len(vi.rowIdxes) > maxCount {
					maxCount = len(vi.rowIdxes)
					canonical = vi.value
				}
			}

			// Replace all other variants with canonical
			for _, vi := range *variants {
				if vi.value == canonical {
					continue
				}
				for _, rowIdx := range vi.rowIdxes {
					data.Rows[rowIdx][col].Raw = canonical
					affectedRows = append(affectedRows, rowIdx)
				}
			}
		}

		if len(affectedRows) > 0 {
			*log = append(*log, LogEntry{
				OperationType: "name_normalize",
				AffectedRows:  affectedRows,
				Timestamp:     time.Now(),
				OperatorID:    operatorID,
				Details:       fmt.Sprintf("正規化欄位 %d 的公司名稱，統一為最長變體", col),
			})
		}
	}
}

// SubtotalRemove removes rows containing subtotal keywords.
// Keywords: "小計", "合計", "total", "subtotal" (case-insensitive)
func SubtotalRemove(data *upload.SheetData, log *[]LogEntry, operatorID string) {
	if data == nil || len(data.Rows) == 0 {
		return
	}

	keywords := []string{"小計", "合計", "total", "subtotal"}

	var kept [][]upload.CellValue
	var removedRows []int

	for i, row := range data.Rows {
		if isSubtotalRow(row, keywords) {
			removedRows = append(removedRows, i)
		} else {
			kept = append(kept, row)
		}
	}

	if len(removedRows) > 0 {
		data.Rows = kept
		data.RowCount = len(kept)

		*log = append(*log, LogEntry{
			OperationType: "subtotal_remove",
			AffectedRows:  removedRows,
			Timestamp:     time.Now(),
			OperatorID:    operatorID,
			Details:       fmt.Sprintf("移除 %d 筆小計/合計列", len(removedRows)),
		})
	}
}

// --- Helper functions ---

// isDateColumn determines if a column is predominantly date-formatted.
// Returns true if > 50% of non-empty values parse as dates.
func isDateColumn(data *upload.SheetData, col int) bool {
	dateCount := 0
	nonEmptyCount := 0

	for _, row := range data.Rows {
		if col >= len(row) || row[col].IsEmpty {
			continue
		}
		nonEmptyCount++
		if _, ok := normalizeDate(row[col].Raw); ok {
			dateCount++
		}
	}

	if nonEmptyCount == 0 {
		return false
	}
	return float64(dateCount)/float64(nonEmptyCount) > 0.5
}

// normalizeDate attempts to parse a date string and return YYYY/MM/DD format.
func normalizeDate(s string) (string, bool) {
	trimmed := strings.TrimSpace(s)

	// Try yyyy/MM/dd or yyyy/M/d
	if m := dateISOSlash.FindStringSubmatch(trimmed); m != nil {
		year, _ := strconv.Atoi(m[1])
		month, _ := strconv.Atoi(m[2])
		day, _ := strconv.Atoi(m[3])
		if isValidDate(year, month, day) {
			return fmt.Sprintf("%04d/%02d/%02d", year, month, day), true
		}
	}

	// Try yyyy-MM-dd
	if m := dateISODash.FindStringSubmatch(trimmed); m != nil {
		year, _ := strconv.Atoi(m[1])
		month, _ := strconv.Atoi(m[2])
		day, _ := strconv.Atoi(m[3])
		if isValidDate(year, month, day) {
			return fmt.Sprintf("%04d/%02d/%02d", year, month, day), true
		}
	}

	// Try ROC date: yyy.M.d (ROC year + 1911 = AD year)
	if m := dateROC.FindStringSubmatch(trimmed); m != nil {
		rocYear, _ := strconv.Atoi(m[1])
		month, _ := strconv.Atoi(m[2])
		day, _ := strconv.Atoi(m[3])
		adYear := rocYear + 1911
		if isValidDate(adYear, month, day) {
			return fmt.Sprintf("%04d/%02d/%02d", adYear, month, day), true
		}
	}

	// Try MM-DD-YY or MM/DD/YY (2-digit year, assume 2000s for 00-49, 1900s for 50-99)
	if m := dateMDY2.FindStringSubmatch(trimmed); m != nil {
		month, _ := strconv.Atoi(m[1])
		day, _ := strconv.Atoi(m[2])
		shortYear, _ := strconv.Atoi(m[3])
		year := shortYear
		if shortYear < 50 {
			year = 2000 + shortYear
		} else {
			year = 1900 + shortYear
		}
		if isValidDate(year, month, day) {
			return fmt.Sprintf("%04d/%02d/%02d", year, month, day), true
		}
	}

	return "", false
}

// isValidDate checks if year/month/day form a valid date.
func isValidDate(year, month, day int) bool {
	if month < 1 || month > 12 || day < 1 {
		return false
	}
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	return t.Year() == year && t.Month() == time.Month(month) && t.Day() == day
}

// hashRow creates a SHA-256 hash of a row for deduplication.
func hashRow(row []upload.CellValue, colCount int) string {
	h := sha256.New()
	for i := 0; i < colCount; i++ {
		if i < len(row) {
			h.Write([]byte(row[i].Raw))
		}
		h.Write([]byte("|"))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// isTextColumn checks if a column is predominantly text (not numeric/date).
func isTextColumn(data *upload.SheetData, col int) bool {
	textCount := 0
	nonEmptyCount := 0

	for _, row := range data.Rows {
		if col >= len(row) || row[col].IsEmpty {
			continue
		}
		nonEmptyCount++
		val := row[col].Raw
		// If it's not parseable as a number and not a date, it's text
		if !isNumericString(val) {
			textCount++
		}
	}

	if nonEmptyCount == 0 {
		return false
	}
	return float64(textCount)/float64(nonEmptyCount) > 0.5
}

// isNumericString checks if a string is a number.
func isNumericString(s string) bool {
	cleaned := strings.ReplaceAll(strings.TrimSpace(s), ",", "")
	_, err := strconv.ParseFloat(cleaned, 64)
	return err == nil
}

// removeSuffixes removes known company suffixes from a name.
func removeSuffixes(name string, suffixes []string) string {
	result := strings.TrimSpace(name)
	for _, suffix := range suffixes {
		if strings.HasSuffix(result, suffix) {
			result = strings.TrimSpace(strings.TrimSuffix(result, suffix))
			break
		}
	}
	return result
}

// isSubtotalRow checks if any cell in the row contains subtotal keywords.
func isSubtotalRow(row []upload.CellValue, keywords []string) bool {
	for _, cell := range row {
		if cell.IsEmpty {
			continue
		}
		lower := strings.ToLower(strings.TrimSpace(cell.Raw))
		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				return true
			}
		}
	}
	return false
}

// NewlineRemove replaces all newline characters in cells with spaces.
func NewlineRemove(data *upload.SheetData, log *[]LogEntry, operatorID string) {
	if data == nil || len(data.Rows) == 0 {
		return
	}

	var affectedRows []int
	for rowIdx, row := range data.Rows {
		modified := false
		for col := 0; col < data.ColCount; col++ {
			if col >= len(row) || row[col].IsEmpty {
				continue
			}
			if strings.Contains(row[col].Raw, "\n") {
				data.Rows[rowIdx][col].Raw = strings.ReplaceAll(row[col].Raw, "\n", " ")
				modified = true
			}
		}
		if modified {
			affectedRows = append(affectedRows, rowIdx)
		}
	}

	if len(affectedRows) > 0 {
		*log = append(*log, LogEntry{
			OperationType: "newline_remove",
			AffectedRows:  affectedRows,
			Timestamp:     time.Now(),
			OperatorID:    operatorID,
			Details:       fmt.Sprintf("移除 %d 列中的儲存格內換行", len(affectedRows)),
		})
	}
}

// BracketNoteRemove removes Chinese bracket notes from cells.
// E.g. "PI-20190227(IQC檢測CPU)" → "PI-20190227"
func BracketNoteRemove(data *upload.SheetData, log *[]LogEntry, operatorID string) {
	if data == nil || len(data.Rows) == 0 {
		return
	}

	var affectedRows []int
	for rowIdx, row := range data.Rows {
		modified := false
		for col := 0; col < data.ColCount; col++ {
			if col >= len(row) || row[col].IsEmpty {
				continue
			}
			cleaned := removeChineseBracketNotes(row[col].Raw)
			if cleaned != row[col].Raw {
				data.Rows[rowIdx][col].Raw = cleaned
				data.Rows[rowIdx][col].IsEmpty = strings.TrimSpace(cleaned) == ""
				modified = true
			}
		}
		if modified {
			affectedRows = append(affectedRows, rowIdx)
		}
	}

	if len(affectedRows) > 0 {
		*log = append(*log, LogEntry{
			OperationType: "bracket_note_remove",
			AffectedRows:  affectedRows,
			Timestamp:     time.Now(),
			OperatorID:    operatorID,
			Details:       fmt.Sprintf("移除 %d 列中的中文括號備註", len(affectedRows)),
		})
	}
}

// removeChineseBracketNotes removes all parenthesized content that contains Chinese characters.
func removeChineseBracketNotes(s string) string {
	result := s
	// Process both half-width and full-width brackets
	for {
		changed := false
		// Half-width brackets
		if idx := strings.Index(result, "("); idx >= 0 {
			closeIdx := strings.Index(result[idx:], ")")
			if closeIdx > 0 {
				inner := result[idx+1 : idx+closeIdx]
				if containsChinese(inner) {
					result = result[:idx] + result[idx+closeIdx+1:]
					changed = true
				}
			}
		}
		// Full-width brackets
		if idx := strings.Index(result, "（"); idx >= 0 {
			closeIdx := strings.Index(result[idx:], "）")
			if closeIdx > 0 {
				inner := result[idx+len("（") : idx+closeIdx]
				if containsChinese(inner) {
					result = result[:idx] + result[idx+closeIdx+len("）"):]
					changed = true
				}
			}
		}
		if !changed {
			break
		}
	}
	return strings.TrimSpace(result)
}

// containsChinese checks if a string contains Chinese characters.
func containsChinese(s string) bool {
	for _, r := range s {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}

// EmptyRowRemove removes rows where all cells are empty.
func EmptyRowRemove(data *upload.SheetData, log *[]LogEntry, operatorID string) {
	if data == nil || len(data.Rows) == 0 {
		return
	}

	var kept [][]upload.CellValue
	var removedRows []int

	for i, row := range data.Rows {
		allEmpty := true
		for col := 0; col < data.ColCount; col++ {
			if col < len(row) && !row[col].IsEmpty {
				allEmpty = false
				break
			}
		}
		if allEmpty {
			removedRows = append(removedRows, i)
		} else {
			kept = append(kept, row)
		}
	}

	if len(removedRows) > 0 {
		data.Rows = kept
		data.RowCount = len(kept)

		*log = append(*log, LogEntry{
			OperationType: "empty_row_remove",
			AffectedRows:  removedRows,
			Timestamp:     time.Now(),
			OperatorID:    operatorID,
			Details:       fmt.Sprintf("移除 %d 筆全空資料列", len(removedRows)),
		})
	}
}

// MultiTableKeepMain keeps only the largest continuous data block.
// Removes rows in other blocks separated by ≥2 consecutive empty rows.
func MultiTableKeepMain(data *upload.SheetData, log *[]LogEntry, operatorID string) {
	if data == nil || len(data.Rows) == 0 {
		return
	}

	// Find all data blocks (separated by ≥2 consecutive empty rows)
	type block struct {
		startIdx int
		endIdx   int // exclusive
		size     int
	}

	var blocks []block
	currentStart := -1
	consecutiveEmpty := 0

	for i, row := range data.Rows {
		empty := true
		for col := 0; col < data.ColCount && col < len(row); col++ {
			if !row[col].IsEmpty {
				empty = false
				break
			}
		}

		if empty {
			consecutiveEmpty++
			if consecutiveEmpty >= 2 && currentStart >= 0 {
				// End of a block
				blocks = append(blocks, block{startIdx: currentStart, endIdx: i - consecutiveEmpty + 1, size: (i - consecutiveEmpty + 1) - currentStart})
				currentStart = -1
			}
		} else {
			if currentStart < 0 {
				currentStart = i
			}
			consecutiveEmpty = 0
		}
	}
	// Don't forget the last block
	if currentStart >= 0 {
		blocks = append(blocks, block{startIdx: currentStart, endIdx: len(data.Rows), size: len(data.Rows) - currentStart})
	}

	if len(blocks) <= 1 {
		return // only one block, nothing to remove
	}

	// Find the largest block
	largestIdx := 0
	for i, b := range blocks {
		if b.size > blocks[largestIdx].size {
			largestIdx = i
		}
	}

	// Keep only the largest block
	largest := blocks[largestIdx]
	removedCount := len(data.Rows) - largest.size
	data.Rows = data.Rows[largest.startIdx:largest.endIdx]
	data.RowCount = len(data.Rows)

	*log = append(*log, LogEntry{
		OperationType: "multi_table_keep_main",
		AffectedRows:  []int{}, // many rows removed
		Timestamp:     time.Now(),
		OperatorID:    operatorID,
		Details:       fmt.Sprintf("保留最大資料區塊（%d 列），移除其他 %d 列", largest.size, removedCount),
	})
}

// EmptyColRemove removes columns where >80% of cells are empty.
// This directly improves Row/Column Completeness scores.
func EmptyColRemove(data *upload.SheetData, log *[]LogEntry, operatorID string) {
	if data == nil || len(data.Rows) == 0 {
		return
	}

	threshold := 0.8 // remove columns with >80% empty
	var keepCols []int
	var removedCols []string

	for col := 0; col < data.ColCount; col++ {
		emptyCount := 0
		for _, row := range data.Rows {
			if col >= len(row) || row[col].IsEmpty {
				emptyCount++
			}
		}
		emptyRate := float64(emptyCount) / float64(len(data.Rows))
		if emptyRate > threshold {
			removedCols = append(removedCols, getColName(data, col))
		} else {
			keepCols = append(keepCols, col)
		}
	}

	if len(removedCols) == 0 {
		return // nothing to remove
	}

	// Rebuild headers
	newHeaders := make([]string, len(keepCols))
	for i, col := range keepCols {
		if col < len(data.Headers) {
			newHeaders[i] = data.Headers[col]
		}
	}

	// Rebuild rows
	for rowIdx, row := range data.Rows {
		newRow := make([]upload.CellValue, len(keepCols))
		for i, col := range keepCols {
			if col < len(row) {
				newRow[i] = row[col]
			} else {
				newRow[i] = upload.CellValue{Raw: "", IsEmpty: true}
			}
		}
		data.Rows[rowIdx] = newRow
	}

	data.Headers = newHeaders
	data.ColCount = len(keepCols)

	*log = append(*log, LogEntry{
		OperationType: "empty_col_remove",
		AffectedRows:  []int{},
		Timestamp:     time.Now(),
		OperatorID:    operatorID,
		Details:       fmt.Sprintf("移除 %d 個高度空缺欄位：%s", len(removedCols), strings.Join(removedCols, "、")),
	})
}

// EmptyColRemoveSpecific removes only the specified column indices.
// Used when the user has explicitly chosen which columns to remove via the preview panel.
func EmptyColRemoveSpecific(data *upload.SheetData, colIndices []int, log *[]LogEntry, operatorID string) {
	if data == nil || len(data.Rows) == 0 || len(colIndices) == 0 {
		return
	}

	// Build a set of columns to remove
	removeSet := make(map[int]bool)
	for _, idx := range colIndices {
		removeSet[idx] = true
	}

	// Build list of columns to keep
	var keepCols []int
	var removedCols []string
	for col := 0; col < data.ColCount; col++ {
		if removeSet[col] {
			removedCols = append(removedCols, getColName(data, col))
		} else {
			keepCols = append(keepCols, col)
		}
	}

	if len(removedCols) == 0 {
		return
	}

	// Rebuild headers
	newHeaders := make([]string, len(keepCols))
	for i, col := range keepCols {
		if col < len(data.Headers) {
			newHeaders[i] = data.Headers[col]
		}
	}

	// Rebuild rows
	for rowIdx, row := range data.Rows {
		newRow := make([]upload.CellValue, len(keepCols))
		for i, col := range keepCols {
			if col < len(row) {
				newRow[i] = row[col]
			} else {
				newRow[i] = upload.CellValue{Raw: "", IsEmpty: true}
			}
		}
		data.Rows[rowIdx] = newRow
	}

	data.Headers = newHeaders
	data.ColCount = len(keepCols)

	*log = append(*log, LogEntry{
		OperationType: "empty_col_remove",
		AffectedRows:  []int{},
		Timestamp:     time.Now(),
		OperatorID:    operatorID,
		Details:       fmt.Sprintf("移除 %d 個使用者選定的空缺欄位：%s", len(removedCols), strings.Join(removedCols, "、")),
	})
}

// MultiTableKeepSpecific keeps only the block at the specified index.
// Used when the user has explicitly chosen which block to keep via the preview panel.
func MultiTableKeepSpecific(data *upload.SheetData, keepIdx int, log *[]LogEntry, operatorID string) {
	if data == nil || len(data.Rows) == 0 {
		return
	}

	// Find all data blocks (same logic as MultiTableKeepMain)
	type block struct {
		startIdx int
		endIdx   int // exclusive
		size     int
	}

	var blocks []block
	currentStart := -1
	consecutiveEmpty := 0

	for i, row := range data.Rows {
		empty := true
		for col := 0; col < data.ColCount && col < len(row); col++ {
			if !row[col].IsEmpty {
				empty = false
				break
			}
		}

		if empty {
			consecutiveEmpty++
			if consecutiveEmpty >= 2 && currentStart >= 0 {
				blocks = append(blocks, block{startIdx: currentStart, endIdx: i - consecutiveEmpty + 1, size: (i - consecutiveEmpty + 1) - currentStart})
				currentStart = -1
			}
		} else {
			if currentStart < 0 {
				currentStart = i
			}
			consecutiveEmpty = 0
		}
	}
	if currentStart >= 0 {
		blocks = append(blocks, block{startIdx: currentStart, endIdx: len(data.Rows), size: len(data.Rows) - currentStart})
	}

	if len(blocks) <= 1 {
		return
	}

	// Validate keepIdx
	if keepIdx < 0 || keepIdx >= len(blocks) {
		// Fallback to largest block
		keepIdx = 0
		for i, b := range blocks {
			if b.size > blocks[keepIdx].size {
				keepIdx = i
			}
		}
	}

	// Keep only the selected block
	selected := blocks[keepIdx]
	removedCount := len(data.Rows) - selected.size
	data.Rows = data.Rows[selected.startIdx:selected.endIdx]
	data.RowCount = len(data.Rows)

	*log = append(*log, LogEntry{
		OperationType: "multi_table_keep_main",
		AffectedRows:  []int{},
		Timestamp:     time.Now(),
		OperatorID:    operatorID,
		Details:       fmt.Sprintf("保留使用者選定的資料區塊（%d 列），移除其他 %d 列", selected.size, removedCount),
	})
}

// getColName returns a display name for the given column index.
func getColName(data *upload.SheetData, col int) string {
	if col < len(data.Headers) {
		name := strings.TrimSpace(data.Headers[col])
		if name != "" {
			return name
		}
	}
	return fmt.Sprintf("第%d欄", col+1)
}
