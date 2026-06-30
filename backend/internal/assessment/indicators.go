package assessment

import (
	"crypto/sha256"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/safe-ai/excel-brushing-tool/internal/upload"
)

// ColumnDetail holds per-column completeness information.
type ColumnDetail struct {
	Name              string  `json:"name"`
	CompletenessRatio float64 `json:"completeness_ratio"`
}

// CalculateRowCompleteness computes the average row completeness score (0-100).
// For each row: non-empty cells / total columns → average all rows × 100.
// Returns 0 if there are no data rows.
func CalculateRowCompleteness(data *upload.SheetData) float64 {
	if len(data.Rows) == 0 || data.ColCount == 0 {
		return 0
	}

	totalRatio := 0.0
	for _, row := range data.Rows {
		nonEmpty := 0
		for i := 0; i < data.ColCount; i++ {
			if i < len(row) && !row[i].IsEmpty && !isCellInvalidValue(row[i].Raw) {
				nonEmpty++
			}
		}
		totalRatio += float64(nonEmpty) / float64(data.ColCount)
	}

	return (totalRatio / float64(len(data.Rows))) * 100
}

// CalculateColumnCompleteness computes the average column completeness score (0-100)
// and returns per-column detail ratios.
// For each column: non-empty values / total rows → average all columns × 100.
// Returns 0 if there are no data rows.
func CalculateColumnCompleteness(data *upload.SheetData) (float64, []ColumnDetail) {
	if len(data.Rows) == 0 || data.ColCount == 0 {
		return 0, nil
	}

	details := make([]ColumnDetail, data.ColCount)
	totalRatio := 0.0

	for col := 0; col < data.ColCount; col++ {
		nonEmpty := 0
		for _, row := range data.Rows {
			if col < len(row) && !row[col].IsEmpty && !isCellInvalidValue(row[col].Raw) {
				nonEmpty++
			}
		}
		ratio := float64(nonEmpty) / float64(len(data.Rows))
		totalRatio += ratio

		name := ""
		if col < len(data.Headers) {
			name = data.Headers[col]
		}
		details[col] = ColumnDetail{
			Name:              name,
			CompletenessRatio: ratio,
		}
	}

	return (totalRatio / float64(data.ColCount)) * 100, details
}

// CalculateFormatConsistency computes the format consistency score (0-100).
// For each column with ≥1 non-empty value:
//   - Detect format type for each value (priority: date > numeric > boolean > text)
//   - Find dominant format type (tie-break by highest priority)
//   - Column score = dominant count / non-empty count
//
// Average all valid columns × 100. Columns with 0 non-empty values are excluded.
func CalculateFormatConsistency(data *upload.SheetData) float64 {
	if data.ColCount == 0 {
		return 100
	}

	validCols := 0
	totalScore := 0.0

	for col := 0; col < data.ColCount; col++ {
		// Collect non-empty values in this column
		formatCounts := map[FormatType]int{}
		nonEmptyCount := 0

		for _, row := range data.Rows {
			if col < len(row) && !row[col].IsEmpty {
				nonEmptyCount++
				ft := DetectFormatType(row[col].Raw)
				formatCounts[ft]++
			}
		}

		if nonEmptyCount == 0 {
			continue // exclude from average
		}

		validCols++

		// Find dominant format type (tie-break by highest priority = lowest FormatType value)
		dominantCount := 0
		for ft := FormatDate; ft <= FormatText; ft++ {
			if formatCounts[ft] > dominantCount {
				dominantCount = formatCounts[ft]
			} else if formatCounts[ft] == dominantCount && formatCounts[ft] > 0 {
				// tie-break: higher priority (lower ft value) wins
				// Since we iterate from highest priority, the first match is the winner
				// Actually we already set dominantCount from the higher priority, so no change needed
			}
		}

		colScore := float64(dominantCount) / float64(nonEmptyCount)
		totalScore += colScore
	}

	if validCols == 0 {
		return 100 // no data to evaluate
	}

	return (totalScore / float64(validCols)) * 100
}

// CalculateDuplicateSimilar computes the data uniqueness score (0-100).
// Based on ISO/IEC 25024:2015 (SQuaRE) methodology:
//   - Completeness factor: ratio of non-empty cells to total cells
//   - Uniqueness factor: ratio of non-duplicated rows to total rows
//   - Score = Completeness × Uniqueness × 100
//
// The multiplicative composition naturally produces non-linear behavior:
// both factors must be high for the score to be high. If either is poor,
// the score drops significantly (e.g., 38% fill × 90% unique = 34.2).
//
// Near-duplicate detection via Levenshtein distance is included as a
// secondary signal within the uniqueness factor.
func CalculateDuplicateSimilar(data *upload.SheetData) float64 {
	if len(data.Rows) == 0 {
		return 100
	}

	totalRows := len(data.Rows)
	totalCells := totalRows * data.ColCount
	if totalCells == 0 {
		return 100
	}

	// Factor 1: Completeness (ISO 25024 — non-empty cells / total cells)
	emptyCellCount := 0
	for _, row := range data.Rows {
		for i := 0; i < data.ColCount; i++ {
			if i >= len(row) || row[i].IsEmpty {
				emptyCellCount++
			}
		}
	}
	completeness := 1.0 - float64(emptyCellCount)/float64(totalCells)

	// Factor 2: Uniqueness (ISO 25024 — non-duplicated rows / total rows)
	// Step 2a: Exact row-level duplicates
	rowHashes := make(map[string]int)
	emptyRowCount := 0
	for _, row := range data.Rows {
		emptyCount := 0
		for i := 0; i < data.ColCount; i++ {
			if i >= len(row) || row[i].IsEmpty {
				emptyCount++
			}
		}
		if emptyCount == data.ColCount {
			emptyRowCount++
			continue
		}
		h := hashRow(row, data.ColCount)
		rowHashes[h]++
	}

	exactDuplicateCount := 0
	if emptyRowCount > 1 {
		exactDuplicateCount += emptyRowCount - 1
	}
	for _, count := range rowHashes {
		if count > 1 {
			exactDuplicateCount += count - 1
		}
	}

	// Step 2b: Near-duplicate detection via eligible text columns
	eligibleCols := selectEligibleColumns(data)
	nearDuplicateGroups := 0

	for _, col := range eligibleCols {
		uniqueValues := collectUniqueValues(data, col)

		var longValues []string
		for _, v := range uniqueValues {
			if len([]rune(v)) > 3 {
				longValues = append(longValues, v)
			}
		}

		limit := len(longValues)
		if limit > 200 {
			limit = 200
		}
		for i := 0; i < limit; i++ {
			for j := i + 1; j < limit; j++ {
				if levenshteinDistance(longValues[i], longValues[j]) <= 2 {
					nearDuplicateGroups++
				}
			}
		}
	}

	// Combine exact + near duplicates (near-duplicates weighted at 0.5)
	effectiveNear := nearDuplicateGroups
	if effectiveNear > totalRows {
		effectiveNear = totalRows
	}
	duplicateRatio := (float64(exactDuplicateCount) + float64(effectiveNear)*0.5) / float64(totalRows)
	uniqueness := math.Max(0, 1.0-duplicateRatio)

	// ISO 25024 composite: Completeness × Uniqueness × 100
	score := completeness * uniqueness * 100
	return math.Round(score*10) / 10
}

// CalculateTableStructure computes the table structure quality score (0-100).
// Start at 100, deduct for various structural issues (each at most once).
// Floor at 0.
func CalculateTableStructure(data *upload.SheetData) float64 {
	score := 100.0

	// Check merged cells
	if len(data.MergedCells) > 0 {
		score -= 20
	}

	// Check multi-layer headers (>1 row in first 5 where all non-empty are text, no repeats)
	if hasMultiLayerHeaders(data) {
		score -= 20
	}

	// Check subtotal rows
	if hasSubtotalRows(data) {
		score -= 15
	}

	// Check multiple tables (≥2 consecutive empty rows separating data blocks)
	if hasMultipleTables(data) {
		score -= 25
	}

	// Check notes in data (text col stddev > mean × 3, mean > 0)
	if hasNotesInData(data) {
		score -= 10
	}

	// Check cells with newlines (multi-info crammed in one cell)
	if hasNewlinesInCells(data) {
		score -= 5
	}

	// Check for strikethrough formatted cells (data marked for deletion but not removed)
	if len(data.StrikethroughCells) > 0 {
		score -= 10
	}

	return math.Max(0, score)
}

// CalculateAIQueryReadiness computes the AI query readiness score (0-100).
// 5 sub-conditions, each +20 (max 100).
// Returns 0 if there are no data rows.
func CalculateAIQueryReadiness(data *upload.SheetData) float64 {
	if len(data.Rows) == 0 {
		return 0
	}

	score := 0.0

	// Sub-condition 1: Identifier column (unique ratio > 80%)
	if hasIdentifierColumn(data) {
		score += 20
	}

	// Sub-condition 2: Time column (date parse > 60% of first min(100, N) rows)
	if hasTimeColumn(data) {
		score += 20
	}

	// Sub-condition 3: Category column (unique count < 20% rows AND > 1)
	if hasCategoryColumn(data) {
		score += 20
	}

	// Sub-condition 4: Numeric column (>80% non-empty parseable as number)
	if hasNumericColumn(data) {
		score += 20
	}

	// Sub-condition 5: Column name quality (all non-empty, non-duplicate, length > 1)
	if hasGoodColumnNames(data) {
		score += 20
	}

	return score
}

// --- Helper functions ---

// isCellInvalidValue returns true if a cell's value is a placeholder reference (同XX)
// that should be treated as equivalent to an empty cell for scoring purposes.
func isCellInvalidValue(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	return cellReferencePlaceholderRe.MatchString(trimmed)
}

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

// selectEligibleColumns selects text columns with 5% < cardinality < 80%, max 5 left-to-right.
func selectEligibleColumns(data *upload.SheetData) []int {
	if len(data.Rows) == 0 {
		return nil
	}

	totalRows := len(data.Rows)
	var eligible []int

	for col := 0; col < data.ColCount && len(eligible) < 5; col++ {
		// Check if column is predominantly text type
		textCount := 0
		nonEmptyCount := 0
		uniqueVals := make(map[string]bool)

		for _, row := range data.Rows {
			if col < len(row) && !row[col].IsEmpty {
				nonEmptyCount++
				val := row[col].Raw
				uniqueVals[val] = true
				if DetectFormatType(val) == FormatText {
					textCount++
				}
			}
		}

		if nonEmptyCount == 0 {
			continue
		}

		// Must be predominantly text (skip only if >70% numeric/date)
		if float64(textCount)/float64(nonEmptyCount) < 0.3 {
			continue
		}

		// Skip date-type columns — repeated dates are normal, not duplicates
		dateCount := 0
		for _, row := range data.Rows {
			if col < len(row) && !row[col].IsEmpty {
				if DetectFormatType(row[col].Raw) == FormatDate {
					dateCount++
				}
			}
		}
		if nonEmptyCount > 0 && float64(dateCount)/float64(nonEmptyCount) > 0.5 {
			continue // predominantly date column — skip
		}

		// Also detect date-like patterns not covered by DetectFormatType
		// (e.g. "02-27-19", "3/15/20", "2020.03.15")
		dateLikeCount := 0
		for _, row := range data.Rows {
			if col < len(row) && !row[col].IsEmpty {
				val := strings.TrimSpace(row[col].Raw)
				if looksLikeDatePattern(val) {
					dateLikeCount++
				}
			}
		}
		if nonEmptyCount > 0 && float64(dateLikeCount)/float64(nonEmptyCount) > 0.5 {
			continue // date-like column — skip
		}

		// Skip key/grouping columns — if any single value repeats > 5% of rows,
		// it's likely a relational key (like PO number) where repeats are normal
		valueCounts := make(map[string]int)
		for _, row := range data.Rows {
			if col < len(row) && !row[col].IsEmpty {
				valueCounts[row[col].Raw]++
			}
		}
		maxCount := 0
		for _, count := range valueCounts {
			if count > maxCount {
				maxCount = count
			}
		}
		if maxCount > 0 && float64(maxCount)/float64(totalRows) > 0.05 {
			continue // key/grouping column — repeats are intentional
		}

		// Skip category columns (≤10 unique values = classification data, not duplicates)
		if len(uniqueVals) <= 10 {
			continue
		}

		// Skip ID-like columns (uniqueness > 80% = likely identifiers, not real duplicates)
		uniqueRatio := float64(len(uniqueVals)) / float64(nonEmptyCount)
		if uniqueRatio > 0.8 {
			continue
		}

		// Check cardinality: > 1% and < 95% (relaxed for real-world data)
		cardinality := float64(len(uniqueVals)) / float64(totalRows)
		if cardinality > 0.01 && cardinality < 0.95 {
			eligible = append(eligible, col)
		}
	}

	return eligible
}

func collectUniqueValues(data *upload.SheetData, col int) []string {
	seen := make(map[string]bool)
	var values []string

	for _, row := range data.Rows {
		if col < len(row) && !row[col].IsEmpty {
			val := row[col].Raw
			if !seen[val] {
				seen[val] = true
				values = append(values, val)
			}
		}
	}
	return values
}

// looksLikeDatePattern checks if a value looks like a date pattern not covered by DetectFormatType.
// Matches patterns like "02-27-19", "2020/3/15", "03.15.2020", "3-15-20".
func looksLikeDatePattern(val string) bool {
	if len(val) < 6 || len(val) > 12 {
		return false
	}
	digits := 0
	separators := 0
	for _, r := range val {
		if r >= '0' && r <= '9' {
			digits++
		} else if r == '-' || r == '/' || r == '.' {
			separators++
		}
	}
	return digits >= 4 && separators >= 2
}

// hasMultiLayerHeaders checks if there are multiple header-like rows in the first 5 rows.
// A row qualifies as header-like if all non-empty cells are text type and no repeated values.
func hasMultiLayerHeaders(data *upload.SheetData) bool {
	limit := 5
	if limit > len(data.Rows) {
		limit = len(data.Rows)
	}

	headerLikeRows := 0
	for i := 0; i < limit; i++ {
		if isHeaderLikeRow(data.Rows[i], data.ColCount) {
			headerLikeRows++
		}
	}

	return headerLikeRows > 1
}

func isHeaderLikeRow(row []upload.CellValue, colCount int) bool {
	seen := make(map[string]bool)
	hasNonEmpty := false

	for i := 0; i < colCount; i++ {
		if i >= len(row) || row[i].IsEmpty {
			continue
		}
		hasNonEmpty = true
		val := strings.TrimSpace(row[i].Raw)

		// Must be text type
		if DetectFormatType(val) != FormatText {
			return false
		}

		// No repeated values
		lower := strings.ToLower(val)
		if seen[lower] {
			return false
		}
		seen[lower] = true
	}

	return hasNonEmpty
}

// hasSubtotalRows checks if any cell contains subtotal keywords.
func hasSubtotalRows(data *upload.SheetData) bool {
	keywords := []string{"小計", "合計", "total", "subtotal"}
	for _, row := range data.Rows {
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
	}
	return false
}

// hasMultipleTables checks for ≥2 consecutive empty rows separating data blocks.
func hasMultipleTables(data *upload.SheetData) bool {
	consecutiveEmpty := 0
	foundDataBefore := false

	for _, row := range data.Rows {
		if isRowEmpty(row, data.ColCount) {
			consecutiveEmpty++
			if consecutiveEmpty >= 2 && foundDataBefore {
				// Check if there's data after these empty rows
				return true
			}
		} else {
			if consecutiveEmpty >= 2 && foundDataBefore {
				// Found data after empty block → multiple tables
				return true
			}
			foundDataBefore = true
			consecutiveEmpty = 0
		}
	}
	return false
}

func isRowEmpty(row []upload.CellValue, colCount int) bool {
	for i := 0; i < colCount && i < len(row); i++ {
		if !row[i].IsEmpty {
			return false
		}
	}
	return true
}

// hasNotesInData checks if any text column has stddev > mean × 3 (mean > 0).
func hasNotesInData(data *upload.SheetData) bool {
	if len(data.Rows) == 0 {
		return false
	}

	for col := 0; col < data.ColCount; col++ {
		// Check if column is predominantly text
		textCount := 0
		nonEmptyCount := 0
		var lengths []float64

		for _, row := range data.Rows {
			if col < len(row) && !row[col].IsEmpty {
				nonEmptyCount++
				val := row[col].Raw
				if DetectFormatType(val) == FormatText {
					textCount++
					lengths = append(lengths, float64(len([]rune(val))))
				}
			}
		}

		if nonEmptyCount == 0 || float64(textCount)/float64(nonEmptyCount) < 0.5 {
			continue
		}

		if len(lengths) == 0 {
			continue
		}

		// Calculate mean and stddev of text lengths
		mean := 0.0
		for _, l := range lengths {
			mean += l
		}
		mean /= float64(len(lengths))

		if mean <= 0 {
			continue
		}

		variance := 0.0
		for _, l := range lengths {
			diff := l - mean
			variance += diff * diff
		}
		variance /= float64(len(lengths))
		stddev := math.Sqrt(variance)

		if stddev > mean*3 {
			return true
		}
	}

	// Also check for notes mixed into numeric columns
	// (e.g. "500(先匯)" in a predominantly numeric column)
	for col := 0; col < data.ColCount; col++ {
		numericCount := 0
		nonEmptyCount := 0
		hasNote := false

		for _, row := range data.Rows {
			if col < len(row) && !row[col].IsEmpty {
				nonEmptyCount++
				val := row[col].Raw
				if isParseableAsNumber(val) {
					numericCount++
				} else if hasChineseBracketNote(val) {
					hasNote = true
				}
			}
		}

		if nonEmptyCount > 0 && float64(numericCount)/float64(nonEmptyCount) > 0.6 && hasNote {
			return true
		}
	}

	// Check for cells with comments (red triangle)
	if len(data.CommentCells) > 0 {
		return true
	}

	// Check for cells with strikethrough formatting
	if len(data.StrikethroughCells) > 0 {
		return true
	}

	// Check for bracket notes in any column (e.g. "PI-20190227(IQC檢測CPU)")
	for col := 0; col < data.ColCount; col++ {
		noteCount := 0
		nonEmptyCount := 0
		for _, row := range data.Rows {
			if col < len(row) && !row[col].IsEmpty {
				nonEmptyCount++
				if hasChineseBracketNote(row[col].Raw) {
					noteCount++
				}
			}
		}
		// If a column has > 10% cells with Chinese bracket notes AND it's not ALL (like a formula column)
		if nonEmptyCount > 10 && float64(noteCount)/float64(nonEmptyCount) > 0.1 && float64(noteCount)/float64(nonEmptyCount) < 0.8 {
			return true
		}
	}

	return false
}

// hasNewlinesInCells checks if any column has >10% cells containing newline characters.
func hasNewlinesInCells(data *upload.SheetData) bool {
	for col := 0; col < data.ColCount; col++ {
		newlineCount := 0
		nonEmptyCount := 0
		for _, row := range data.Rows {
			if col < len(row) && !row[col].IsEmpty {
				nonEmptyCount++
				if strings.Contains(row[col].Raw, "\n") {
					newlineCount++
				}
			}
		}
		if nonEmptyCount > 5 && float64(newlineCount)/float64(nonEmptyCount) > 0.1 {
			return true
		}
	}
	return false
}

// hasChineseBracketNote checks if a value contains brackets with Chinese content inside.
// This indicates user-added notes rather than standard notation like (P/N: xxx).
func hasChineseBracketNote(val string) bool {
	// Check for half-width or full-width opening brackets
	if idx := strings.IndexAny(val, "(（"); idx >= 0 {
		// Find the matching closing bracket
		closeIdx := strings.IndexAny(val[idx+1:], ")）")
		if closeIdx > 0 {
			inner := val[idx+1 : idx+1+closeIdx]
			// Check if inner content has Chinese characters
			for _, r := range inner {
				if r >= 0x4E00 && r <= 0x9FFF {
					return true
				}
			}
		}
	}
	return false
}

// hasIdentifierColumn checks if ≥1 col has unique ratio > 80% (non-empty values).
func hasIdentifierColumn(data *upload.SheetData) bool {
	for col := 0; col < data.ColCount; col++ {
		uniqueVals := make(map[string]bool)
		nonEmptyCount := 0

		for _, row := range data.Rows {
			if col < len(row) && !row[col].IsEmpty {
				nonEmptyCount++
				uniqueVals[row[col].Raw] = true
			}
		}

		if nonEmptyCount > 0 {
			uniqueRatio := float64(len(uniqueVals)) / float64(nonEmptyCount)
			if uniqueRatio > 0.8 {
				return true
			}
		}
	}
	return false
}

// hasTimeColumn checks if ≥1 col where date parse > 60% of first min(100, N) rows.
func hasTimeColumn(data *upload.SheetData) bool {
	sampleSize := len(data.Rows)
	if sampleSize > 100 {
		sampleSize = 100
	}

	for col := 0; col < data.ColCount; col++ {
		dateCount := 0
		for i := 0; i < sampleSize; i++ {
			row := data.Rows[i]
			if col < len(row) && !row[col].IsEmpty {
				if DetectFormatType(row[col].Raw) == FormatDate {
					dateCount++
				}
			}
		}
		if float64(dateCount)/float64(sampleSize) > 0.6 {
			return true
		}
	}
	return false
}

// hasCategoryColumn checks if ≥1 col with unique count < 20% rows AND > 1.
func hasCategoryColumn(data *upload.SheetData) bool {
	totalRows := len(data.Rows)
	for col := 0; col < data.ColCount; col++ {
		uniqueVals := make(map[string]bool)

		for _, row := range data.Rows {
			if col < len(row) && !row[col].IsEmpty {
				uniqueVals[row[col].Raw] = true
			}
		}

		uniqueCount := len(uniqueVals)
		if uniqueCount > 1 && float64(uniqueCount) < float64(totalRows)*0.2 {
			return true
		}
	}
	return false
}

// hasNumericColumn checks if ≥1 col where >80% non-empty values are parseable as number.
func hasNumericColumn(data *upload.SheetData) bool {
	for col := 0; col < data.ColCount; col++ {
		numericCount := 0
		nonEmptyCount := 0

		for _, row := range data.Rows {
			if col < len(row) && !row[col].IsEmpty {
				nonEmptyCount++
				if isParseableAsNumber(row[col].Raw) {
					numericCount++
				}
			}
		}

		if nonEmptyCount > 0 && float64(numericCount)/float64(nonEmptyCount) > 0.8 {
			return true
		}
	}
	return false
}

// hasGoodColumnNames checks if all column names are non-empty (trimmed), non-duplicate, length > 1.
func hasGoodColumnNames(data *upload.SheetData) bool {
	if len(data.Headers) == 0 {
		return false
	}

	nameSet := make(map[string]bool)
	for _, header := range data.Headers {
		trimmed := strings.TrimSpace(header)
		if trimmed == "" || len([]rune(trimmed)) <= 1 {
			return false
		}
		lower := strings.ToLower(trimmed)
		if nameSet[lower] {
			return false
		}
		nameSet[lower] = true
	}
	return true
}

// isParseableAsNumber checks if a string can be parsed as a number.
func isParseableAsNumber(s string) bool {
	trimmed := strings.TrimSpace(s)
	// Remove thousands separators for parsing
	cleaned := strings.ReplaceAll(trimmed, ",", "")
	_, err := strconv.ParseFloat(cleaned, 64)
	return err == nil
}

// CalculateFormatConsistencyWithIssues computes the format consistency score (0-100),
// integrating additional placeholder and type mismatch counts.
// Placeholder cells in numeric columns are treated as severe format violations.
func CalculateFormatConsistencyWithIssues(data *upload.SheetData, placeholderCells int, mismatchCells int) float64 {
	if data.ColCount == 0 {
		return 100
	}

	// If no extra issues, fall back to the base calculation
	if placeholderCells == 0 && mismatchCells == 0 {
		return CalculateFormatConsistency(data)
	}

	// Start with base format consistency, then apply a direct penalty
	baseScore := CalculateFormatConsistency(data)

	// Count total non-empty cells
	totalNonEmpty := 0
	for col := 0; col < data.ColCount; col++ {
		for _, row := range data.Rows {
			if col < len(row) && !row[col].IsEmpty {
				totalNonEmpty++
			}
		}
	}

	if totalNonEmpty == 0 {
		return baseScore
	}

	// Apply aggressive penalty:
	// Placeholder cells in numeric columns are 5x weight (completely corrupt data)
	// Type mismatches are 3x weight (wrong type in numeric column)
	effectiveProblemCells := float64(placeholderCells)*5.0 + float64(mismatchCells)*3.0
	penaltyRatio := effectiveProblemCells / float64(totalNonEmpty)

	// No cap — if data is severely corrupt, score should drop to near 0
	if penaltyRatio > 1.0 {
		penaltyRatio = 1.0
	}

	return math.Max(0, baseScore*(1.0-penaltyRatio))
}

// CalculateTableStructureWithIssues computes the table structure quality score (0-100),
// integrating orphan total and inline remark detection results.
// - If orphanTotalDetected AND no keyword subtotals already detected, apply -15
// - If inlineRemarkDense (>20% in structured col) AND notes deduction not already applied, apply -10
// Avoids double-counting deductions.
func CalculateTableStructureWithIssues(data *upload.SheetData, orphanTotalDetected bool, inlineRemarkDense bool) float64 {
	score := 100.0

	// Check merged cells
	if len(data.MergedCells) > 0 {
		score -= 20
	}

	// Check multi-layer headers
	if hasMultiLayerHeaders(data) {
		score -= 20
	}

	// Check subtotal rows (keyword-based)
	hasKeywordSubtotal := hasSubtotalRows(data)
	if hasKeywordSubtotal {
		score -= 15
	}

	// Orphan total: apply -20 only if no keyword subtotals already detected (avoid double-counting)
	if orphanTotalDetected && !hasKeywordSubtotal {
		score -= 20
	}

	// Check multiple tables
	if hasMultipleTables(data) {
		score -= 25
	}

	// Check notes in data
	hasNotes := hasNotesInData(data)
	if hasNotes {
		score -= 10
	}

	// Inline remark dense: apply -15 only if notes deduction not already applied (avoid double-counting)
	if inlineRemarkDense && !hasNotes {
		score -= 15
	}

	// Check cells with newlines
	if hasNewlinesInCells(data) {
		score -= 5
	}

	// Check for strikethrough formatted cells (data marked for deletion but not removed)
	if len(data.StrikethroughCells) > 0 {
		score -= 10
	}

	return math.Max(0, score)
}

// CalculateAIQueryReadinessWithIssues computes the AI query readiness score (0-100),
// integrating the empty header detection result and data corruption penalties.
// If emptyHeaderDetected is true, the "column name quality" sub-condition automatically fails (-20).
// Additionally applies penalties for placeholder cells and inline remarks that make AI querying unreliable.
func CalculateAIQueryReadinessWithIssues(data *upload.SheetData, emptyHeaderDetected bool) float64 {
	if len(data.Rows) == 0 {
		return 0
	}

	score := 0.0

	// Sub-condition 1: Identifier column
	if hasIdentifierColumn(data) {
		score += 20
	}

	// Sub-condition 2: Time column
	if hasTimeColumn(data) {
		score += 20
	}

	// Sub-condition 3: Category column
	if hasCategoryColumn(data) {
		score += 20
	}

	// Sub-condition 4: Numeric column
	if hasNumericColumn(data) {
		score += 20
	}

	// Sub-condition 5: Column name quality
	// If emptyHeaderDetected, this sub-condition fails regardless of other checks
	if !emptyHeaderDetected && hasGoodColumnNames(data) {
		score += 20
	}

	// Additional penalties for data corruption issues that directly affect AI query reliability
	// Placeholder cells ("同XX") — if >5% of total cells, deduct up to 30 points
	placeholderCount := CountPlaceholderCells(data)
	totalCells := data.ColCount * len(data.Rows)
	if totalCells > 0 && placeholderCount > 0 {
		placeholderRatio := float64(placeholderCount) / float64(totalCells)
		// Deduct 30 points scaled by ratio (cap at 30)
		penalty := math.Min(30, placeholderRatio*600) // 5% → 30 points
		score -= penalty
	}

	// Inline remarks in structured columns — if dense, deduct 15
	if IsInlineRemarkDense(data) {
		score -= 15
	}

	return math.Max(0, score)
}

// CountPlaceholderCells counts the number of cells matching the cell reference placeholder pattern.
// Used to pass detection results to CalculateFormatConsistencyWithIssues.
func CountPlaceholderCells(data *upload.SheetData) int {
	count := 0
	for _, row := range data.Rows {
		for colIdx := 0; colIdx < data.ColCount; colIdx++ {
			if colIdx >= len(row) || row[colIdx].IsEmpty {
				continue
			}
			val := strings.TrimSpace(row[colIdx].Raw)
			if cellReferencePlaceholderRe.MatchString(val) {
				count++
			}
		}
	}
	return count
}

// CountTypeMismatchCells counts the number of type mismatch cells across all numeric columns.
// A column is "numeric" type if >70% of non-empty cells are numeric.
// Used to pass detection results to CalculateFormatConsistencyWithIssues.
func CountTypeMismatchCells(data *upload.SheetData) int {
	count := 0
	for col := 0; col < data.ColCount; col++ {
		numericCount := 0
		nonEmptyCount := 0

		for _, row := range data.Rows {
			if col >= len(row) || row[col].IsEmpty {
				continue
			}
			nonEmptyCount++
			if isCellNumericForTypeMismatch(row[col].Raw) {
				numericCount++
			}
		}

		if nonEmptyCount == 0 {
			continue
		}

		// Column is "numeric" type if >70% are numeric
		if float64(numericCount)/float64(nonEmptyCount) <= 0.70 {
			continue
		}

		// Count non-numeric, non-empty cells
		for _, row := range data.Rows {
			if col >= len(row) || row[col].IsEmpty {
				continue
			}
			if !isCellNumericForTypeMismatch(row[col].Raw) {
				count++
			}
		}
	}
	return count
}

// HasOrphanTotalRows returns true if orphan total rows are detected in the data.
// Used to pass detection results to CalculateTableStructureWithIssues.
func HasOrphanTotalRows(data *upload.SheetData) bool {
	issues := DetectOrphanTotalRows(data)
	return len(issues) > 0
}

// IsInlineRemarkDense returns true if any structured column has >20% cells with inline remarks.
// Used to pass detection results to CalculateTableStructureWithIssues.
func IsInlineRemarkDense(data *upload.SheetData) bool {
	for col := 0; col < data.ColCount; col++ {
		if !isStructuredColumn(data, col) {
			continue
		}

		nonEmpty := 0
		remarkCount := 0
		for _, row := range data.Rows {
			if col >= len(row) || row[col].IsEmpty {
				continue
			}
			nonEmpty++
			if hasInlineRemark(row[col].Raw) {
				remarkCount++
			}
		}

		if nonEmpty > 0 && float64(remarkCount)/float64(nonEmpty) > 0.20 {
			return true
		}
	}
	return false
}

// HasEmptyHeaders returns true if any column header is empty or whitespace-only.
// Used to pass detection results to CalculateAIQueryReadinessWithIssues.
func HasEmptyHeaders(data *upload.SheetData) bool {
	for _, header := range data.Headers {
		if strings.TrimSpace(header) == "" {
			return true
		}
	}
	return false
}

// CalculateRowDistribution classifies ALL sheet rows into High/Medium/Low readiness
// based on their non-empty cell ratio:
//   - High: >= 80% non-empty
//   - Medium: 50%-79% non-empty
//   - Low: < 50% non-empty
// Includes pre-header rows, header row, and data rows so that sum = TotalSheetRows.
func CalculateRowDistribution(data *upload.SheetData) RowDistribution {
	dist := RowDistribution{}
	if data.ColCount == 0 {
		return dist
	}

	classifyRow := func(cells []string) {
		nonEmpty := 0
		for _, c := range cells {
			if strings.TrimSpace(c) != "" {
				nonEmpty++
			}
		}
		ratio := float64(nonEmpty) / float64(data.ColCount)
		switch {
		case ratio >= 0.8:
			dist.High++
		case ratio >= 0.5:
			dist.Medium++
		default:
			dist.Low++
		}
	}

	// 1. Pre-header rows + header row (from RawFirstRows up to and including HeaderRowIndex)
	//    These are rows BEFORE the data that LoadSheetData excluded from data.Rows
	preDataCount := data.HeaderRowIndex + 1 // number of rows before data starts
	for i := 0; i < preDataCount; i++ {
		if i < len(data.RawFirstRows) {
			classifyRow(data.RawFirstRows[i])
		} else {
			// If raw data not available, count header row separately
			if i == data.HeaderRowIndex && len(data.Headers) > 0 {
				classifyRow(data.Headers)
			} else {
				dist.Low++ // unknown row → default to low
			}
		}
	}

	// 2. Data rows (from data.Rows)
	for _, row := range data.Rows {
		nonEmpty := 0
		for i := 0; i < data.ColCount; i++ {
			if i < len(row) && !row[i].IsEmpty {
				nonEmpty++
			}
		}
		ratio := float64(nonEmpty) / float64(data.ColCount)
		switch {
		case ratio >= 0.8:
			dist.High++
		case ratio >= 0.5:
			dist.Medium++
		default:
			dist.Low++
		}
	}

	return dist
}
