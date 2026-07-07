package assessment

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/safe-ai/excel-brushing-tool/internal/upload"
)

// Issue represents a detected data quality problem.
type Issue struct {
	Title        string          `json:"title"`         // 大分類標題
	TitleEn      string          `json:"title_en"`      // English title
	Severity     string          `json:"severity"`      // "High", "Medium", "Low"
	AffectedRows int             `json:"affected_rows"`
	Unit         string          `json:"unit"`          // "列受影響", "列", "組", "處"
	Description  string          `json:"description"`   // 具體問題描述
	DescriptionEn string         `json:"description_en"` // English description (simplified)
	Examples     []IssueExample  `json:"examples"`      // 實際資料片段（Excel 截圖風格）
	Indicator    string          `json:"indicator"`
}

// IssueExample 代表一個問題的實際資料截圖
// 包含欄位標頭、列號、cell 值，以及哪些 cell 有問題
type IssueExample struct {
	Label        string      `json:"label,omitempty"`         // 分組標籤（同一 issue 內的子問題名稱）
	Headers      []string    `json:"headers"`                 // 顯示的欄位名
	RowNumber    int         `json:"row_number"`              // 原始資料中的列號（1-based）
	Cells        []string    `json:"cells"`                   // 該列各欄位的實際值
	Highlights   []int       `json:"highlights"`              // 有問題的 cell index（紅色高亮）
	Merges       []CellMerge `json:"merges,omitempty"`        // 合併儲存格資訊
	FormatLabels []string    `json:"format_labels,omitempty"` // 格式標籤（格式混用時顯示每欄偵測到的格式）
}

// CellMerge represents a merged cell span in the example display
type CellMerge struct {
	StartCol int `json:"start_col"` // starting cell index in the Cells array
	Span     int `json:"span"`      // number of columns this cell spans (colspan)
}

// maxExamples is the maximum number of examples per issue to avoid bloating the response.
const maxExamples = 5

// DetectIssues generates a problem list based on indicator scores and detected problems.
// Descriptions list specific problems found (column names, formats, etc.)
func DetectIssues(data *upload.SheetData, scores IndicatorScores) []Issue {
	var issues []Issue

	totalRows := len(data.Rows)

	// Row Completeness issues — list which rows have most gaps
	if scores.RowCompleteness < 60 {
		incompleteRows := countIncompleteRows(data)
		examples := buildRowCompletenessExamples(data)
		issues = append(issues, Issue{
			Title:        "資料大量缺漏",
			Severity:     "High",
			AffectedRows: incompleteRows,
			Unit:         "列受影響",
			Description:  fmt.Sprintf("共 %d 列存在空值欄位，平均每列填寫率僅 %.0f%%", incompleteRows, scores.RowCompleteness),
			Examples:     examples,
			Indicator:    "row_completeness",
		})
	} else if scores.RowCompleteness < 80 {
		incompleteRows := countIncompleteRows(data)
		examples := buildRowCompletenessExamples(data)
		issues = append(issues, Issue{
			Title:        "部分資料缺漏",
			Severity:     "Medium",
			AffectedRows: incompleteRows,
			Unit:         "列受影響",
			Description:  fmt.Sprintf("共 %d 列存在部分空值，建議補齊資料以提升品質", incompleteRows),
			Examples:     examples,
			Indicator:    "row_completeness",
		})
	}

	// Column Completeness issues — list specific low-completeness columns
	if scores.ColumnCompleteness < 80 {
		lowCols := findLowCompletenessColumnNames(data, 0.7)
		affectedRows := countRowsWithMissingInColumns(data, 0.7)
		sev := "Medium"
		title := "部分資料缺漏"
		threshold := 0.7
		if scores.ColumnCompleteness < 60 {
			sev = "High"
			title = "資料大量缺漏"
			lowCols = findLowCompletenessColumnNames(data, 0.5)
			affectedRows = countRowsWithMissingInColumns(data, 0.5)
			threshold = 0.5
		}
		if len(lowCols) > 0 {
			// Build per-column missing rates for description
			var colDescs []string
			for col := 0; col < data.ColCount && len(colDescs) < 3; col++ {
				nonEmpty := 0
				for _, row := range data.Rows {
					if col < len(row) && !row[col].IsEmpty {
						nonEmpty++
					}
				}
				ratio := float64(nonEmpty) / float64(len(data.Rows))
				if ratio < threshold {
					missingPct := int((1 - ratio) * 100)
					colDescs = append(colDescs, fmt.Sprintf("「%s」(%d%%)", getColumnName(data, col), missingPct))
				}
			}
			desc := "以下欄位嚴重缺漏：" + strings.Join(colDescs, "、")
			if len(lowCols) > 3 {
				desc += fmt.Sprintf("⋯等 %d 欄", len(lowCols))
			}
			examples := buildColumnCompletenessExamples(data, threshold)
			issues = append(issues, Issue{
				Title:        title,
				Severity:     sev,
				AffectedRows: affectedRows,
				Unit:         "列受影響",
				Description:  desc,
				Examples:     examples,
				Indicator:    "column_completeness",
			})
		}
	}

	// Format Consistency issues — list columns with mixed formats
	if scores.FormatConsistency < 80 {
		mixedCols := findMixedFormatColumns(data)
		affectedRows := countRowsWithFormatIssues(data)
		sev := "Medium"
		title := "格式混用"
		if scores.FormatConsistency < 60 {
			sev = "High"
			title = "日期格式不一致"
		}
		if len(mixedCols) > 0 {
			examples := buildFormatConsistencyExamples(data)
			issues = append(issues, Issue{
				Title:        title,
				Severity:     sev,
				AffectedRows: affectedRows,
				Unit:         "列",
				Description:  "以下欄位格式不一致（同欄位中混合了不同格式）：\n" + strings.Join(mixedCols, "\n"),
				Examples:     examples,
				Indicator:    "format_consistency",
			})
		}
	}

	// Duplicate/Similar issues
	if scores.DuplicateSimilar < 80 {
		affectedRows := totalRows - int(scores.DuplicateSimilar*float64(totalRows)/100)
		if affectedRows < 0 {
			affectedRows = 0
		}
		if affectedRows > totalRows {
			affectedRows = totalRows
		}
		sev := "Medium"
		if scores.DuplicateSimilar < 60 {
			sev = "High"
		}
		examples := buildDuplicateExamples(data)
		desc := fmt.Sprintf("偵測到約 %d 列重複或近似資料，建議清理以避免統計偏差", affectedRows)
		if len(examples) == 0 {
			desc = "偵測到欄位中有近似值（如拼寫相近的名稱），建議統一命名以避免混淆"
		}
		issues = append(issues, Issue{
			Title:        "疑似重複資料",
			Severity:     sev,
			AffectedRows: affectedRows,
			Unit:         "組",
			Description:  desc,
			Examples:     examples,
			Indicator:    "duplicate_similar",
		})
	}

	// Company name variant detection — only if a company/client column exists
	companyCol := findCompanyColumn(data)
	if companyCol >= 0 {
		variantGroups := detectNameVariants(data, companyCol)
		if len(variantGroups) > 0 {
			totalVariants := 0
			for _, g := range variantGroups {
				totalVariants += g.totalRows - g.canonicalCount
			}
			if totalVariants > 0 {
				examples := buildNameVariantExamples(data, companyCol, variantGroups)
				issues = append(issues, Issue{
					Title:        "客戶名稱不一致",
					Severity:     "Medium",
					AffectedRows: totalVariants,
					Unit:         "列",
					Description:  fmt.Sprintf("偵測到 %d 組客戶名稱有多種寫法，建議統一", len(variantGroups)),
					Examples:     examples,
					Indicator:    "name_variants",
				})
			}
		}
	}

	// Table Structure issues — single issue with sub-problem labels in examples
	// Only include problems that can actually produce examples
	if scores.TableStructure < 100 {
		structProblems := detectStructureProblems(data)
		sev := "Low"
		if scores.TableStructure < 60 {
			sev = "High"
		} else if scores.TableStructure < 80 {
			sev = "Medium"
		}
		var structExamples []IssueExample
		var confirmedProblems []string
		for _, problem := range structProblems {
			subExamples := buildSingleStructureExamples(data, problem)
			if len(subExamples) == 0 {
				continue // 無法展示的問題不列入描述
			}
			confirmedProblems = append(confirmedProblems, problem)
			for i := range subExamples {
				subExamples[i].Label = problem
			}
			structExamples = append(structExamples, subExamples...)
		}
		if len(confirmedProblems) > 0 {
			issues = append(issues, Issue{
				Title:        "表格結構問題",
				Severity:     sev,
				AffectedRows: len(confirmedProblems),
				Unit:         "處",
				Description:  "偵測到" + strings.Join(confirmedProblems, "、"),
				Examples:     structExamples,
				Indicator:    "table_structure",
			})
		}
	}



	// Orphan Total Rows detection — independent of keyword subtotal detection
	orphanIssues := DetectOrphanTotalRows(data)
	issues = append(issues, orphanIssues...)

	// Cell Reference Placeholder detection
	placeholderIssues := DetectCellReferencePlaceholders(data)
	issues = append(issues, placeholderIssues...)

	// Column Type Mismatch detection
	typeMismatchIssues := DetectColumnTypeMismatch(data)
	issues = append(issues, typeMismatchIssues...)

	// Empty Header detection
	emptyHeaderIssues := DetectEmptyHeaders(data)
	issues = append(issues, emptyHeaderIssues...)

	// Inline Remark Detection
	inlineRemarkIssues := DetectInlineRemarks(data)
	issues = append(issues, inlineRemarkIssues...)

	// Strikethrough Formatting Detection
	strikethroughIssues := DetectStrikethroughFormatting(data)
	issues = append(issues, strikethroughIssues...)

	// AI Query Readiness → shown as "AI 應用完備度" with plain language (always last)
	if scores.AIQueryReadiness < 80 {
		sev := "Medium"
		if scores.AIQueryReadiness < 60 {
			sev = "High"
		}
		examples := buildStructuralReadinessExamples(data, scores)
		issues = append(issues, Issue{
			Title:        "AI 應用完備度",
			Severity:     sev,
			AffectedRows: totalRows,
			Unit:         "列",
			Description:  "部分結構化條件未滿足，可能影響 AI 分析的精準度與完整性",
			Examples:     examples,
			Indicator:    "ai_query_readiness",
		})
	}

	// Post-processing: merge issues with the same Title
	issues = mergeIssuesByTitle(issues)

	// Post-processing: add English titles and descriptions
	issues = addEnglishTranslations(issues)

	return issues
}

// countIncompleteRows returns the number of rows that have at least one empty cell.
func countIncompleteRows(data *upload.SheetData) int {
	count := 0
	for _, row := range data.Rows {
		for i := 0; i < data.ColCount; i++ {
			if i >= len(row) || row[i].IsEmpty {
				count++
				break
			}
		}
	}
	return count
}

// findLowCompletenessColumnNames returns column names with completeness below threshold.
func findLowCompletenessColumnNames(data *upload.SheetData, threshold float64) []string {
	if len(data.Rows) == 0 {
		return nil
	}
	var names []string
	for col := 0; col < data.ColCount; col++ {
		nonEmpty := 0
		for _, row := range data.Rows {
			if col < len(row) && !row[col].IsEmpty {
				nonEmpty++
			}
		}
		ratio := float64(nonEmpty) / float64(len(data.Rows))
		if ratio < threshold {
			names = append(names, getColumnName(data, col))
		}
	}
	return names
}

// countRowsWithMissingInColumns counts rows that have at least one missing value
// in the low-completeness columns.
func countRowsWithMissingInColumns(data *upload.SheetData, threshold float64) int {
	if len(data.Rows) == 0 {
		return 0
	}
	// Find low-completeness column indices
	var lowCols []int
	for col := 0; col < data.ColCount; col++ {
		nonEmpty := 0
		for _, row := range data.Rows {
			if col < len(row) && !row[col].IsEmpty {
				nonEmpty++
			}
		}
		ratio := float64(nonEmpty) / float64(len(data.Rows))
		if ratio < threshold {
			lowCols = append(lowCols, col)
		}
	}
	// Count rows affected
	count := 0
	for _, row := range data.Rows {
		for _, col := range lowCols {
			if col >= len(row) || row[col].IsEmpty {
				count++
				break
			}
		}
	}
	return count
}

// findMixedFormatColumns returns column names where format consistency is low.
func findMixedFormatColumns(data *upload.SheetData) []string {
	var names []string
	for col := 0; col < data.ColCount; col++ {
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
			continue
		}
		// Find dominant count
		dominantCount := 0
		for _, c := range formatCounts {
			if c > dominantCount {
				dominantCount = c
			}
		}
		consistency := float64(dominantCount) / float64(nonEmptyCount)
		if consistency < 0.8 {
			names = append(names, getColumnName(data, col))
		}
	}
	return names
}

// countRowsWithFormatIssues counts rows that have at least one cell not matching dominant format.
func countRowsWithFormatIssues(data *upload.SheetData) int {
	count := 0
	for _, row := range data.Rows {
		hasIssue := false
		for col := 0; col < data.ColCount && !hasIssue; col++ {
			if col >= len(row) || row[col].IsEmpty {
				continue
			}
			// Simple check: if the value looks like it could be multiple formats
			ft := DetectFormatType(row[col].Raw)
			if ft == FormatText {
				// Text in a column that might be numeric/date could indicate an issue
				hasIssue = true
			}
		}
		if hasIssue {
			count++
		}
	}
	// Cap at total rows
	if count > len(data.Rows) {
		count = len(data.Rows)
	}
	return count
}

// getColumnName returns a display name for the given column index.
// Uses the header value if available and non-empty, otherwise falls back to "第N欄".
func getColumnName(data *upload.SheetData, col int) string {
	if col < len(data.Headers) {
		name := strings.TrimSpace(data.Headers[col])
		if name != "" {
			// Replace newlines in header names (Excel headers can contain line breaks)
			name = strings.ReplaceAll(name, "\n", " ")
			name = strings.ReplaceAll(name, "\r", "")
			return name
		}
	}
	// Fallback
	return fmt.Sprintf("第%d欄", col+1)
}

// detectStructureProblems returns human-readable descriptions of structure issues.
func detectStructureProblems(data *upload.SheetData) []string {
	var problems []string
	if len(data.MergedCells) > 0 {
		problems = append(problems, "合併儲存格")
	}
	if hasMultiLayerHeaders(data) {
		problems = append(problems, "多層標題")
	}
	if hasSubtotalRows(data) {
		problems = append(problems, "小計/合計列")
	}
	if hasMultipleTables(data) {
		problems = append(problems, "多表格混在同一 sheet")
	}
	// Note: "備註混入資料欄" is handled by DetectInlineRemarks as an independent card
	if hasNewlinesInCells(data) {
		problems = append(problems, "儲存格含換行")
	}
	return problems
}

// --- Example builder helpers ---

// limitExamples truncates a slice to at most maxExamples items.
func limitExamples(items []IssueExample) []IssueExample {
	if len(items) <= maxExamples {
		return items
	}
	return items[:maxExamples]
}

// getDisplayHeaders returns header names for the given column indices.
func getDisplayHeaders(data *upload.SheetData, relevantCols []int) []string {
	headers := make([]string, len(relevantCols))
	for i, col := range relevantCols {
		headers[i] = getColumnName(data, col)
	}
	return headers
}

// getRowCells extracts cell values for the given column indices from a row.
// Each value is truncated to 40 characters.
func getRowCells(row []upload.CellValue, cols []int) []string {
	cells := make([]string, len(cols))
	for i, col := range cols {
		if col < len(row) && !row[col].IsEmpty {
			val := row[col].Raw
			if len([]rune(val)) > 40 {
				val = string([]rune(val)[:40]) + "…"
			}
			cells[i] = val
		} else {
			cells[i] = ""
		}
	}
	return cells
}

// selectDisplayColumns picks at most 6 columns to display.
// Includes a context column (first non-date column) plus the problem columns, sorted.
// Problem columns are always included; context columns are trimmed if over limit.
func selectDisplayColumns(data *upload.SheetData, problemCols []int) []int {
	const maxDisplayCols = 6
	chosen := make(map[int]bool)

	// Always include all problem columns first (these are the most important)
	for _, c := range problemCols {
		if c < data.ColCount {
			chosen[c] = true
		}
	}

	// Add context column only if space allows
	if len(chosen) < maxDisplayCols {
		contextCol := 0
		if isDateColumn(data, 0) && data.ColCount > 1 {
			for c := 1; c < data.ColCount; c++ {
				if !isDateColumn(data, c) {
					contextCol = c
					break
				}
			}
		}
		chosen[contextCol] = true
	}

	// Convert to sorted slice
	var result []int
	for c := range chosen {
		result = append(result, c)
	}
	sort.Ints(result)

	// Truncate to maxDisplayCols — keep problem cols, drop context cols
	if len(result) > maxDisplayCols {
		problemSet := make(map[int]bool)
		for _, c := range problemCols {
			problemSet[c] = true
		}
		if len(problemCols) >= maxDisplayCols {
			// Too many problem cols — just take first maxDisplayCols from sorted problem cols
			var sortedProblems []int
			for _, c := range problemCols {
				if c < data.ColCount {
					sortedProblems = append(sortedProblems, c)
				}
			}
			sort.Ints(sortedProblems)
			if len(sortedProblems) > maxDisplayCols {
				sortedProblems = sortedProblems[:maxDisplayCols]
			}
			result = sortedProblems
		} else {
			// Keep all problem cols, fill remaining with non-problem cols
			var kept []int
			var extra []int
			for _, c := range result {
				if problemSet[c] {
					kept = append(kept, c)
				} else {
					extra = append(extra, c)
				}
			}
			remaining := maxDisplayCols - len(kept)
			if remaining > len(extra) {
				remaining = len(extra)
			}
			kept = append(kept, extra[:remaining]...)
			sort.Ints(kept)
			result = kept
		}
	}
	return result
}

// buildRowCompletenessExamples shows rows with empty cells, highlighting the empty ones.
// Displays at most 6 columns.
func buildRowCompletenessExamples(data *upload.SheetData) []IssueExample {
	var examples []IssueExample
	for rowIdx, row := range data.Rows {
		if len(examples) >= maxExamples {
			break
		}
		// Find empty cols in this row
		var emptyCols []int
		for col := 0; col < data.ColCount; col++ {
			if col >= len(row) || row[col].IsEmpty {
				emptyCols = append(emptyCols, col)
			}
		}
		if len(emptyCols) == 0 {
			continue
		}
		// Pick display columns: include some empty cols + col 0 for context
		displayCols := selectDisplayColumns(data, emptyCols)
		headers := getDisplayHeaders(data, displayCols)
		cells := getRowCells(row, displayCols)
		// Determine which display indices are highlighted (the empty ones)
		var highlights []int
		for i, col := range displayCols {
			if col >= len(row) || row[col].IsEmpty {
				highlights = append(highlights, i)
			}
		}
		examples = append(examples, IssueExample{
			Headers:    headers,
			RowNumber:  rowIdx + data.HeaderRowIndex + 2,
			Cells:      cells,
			Highlights: highlights,
		})
	}
	return limitExamples(examples)
}

// buildColumnCompletenessExamples shows rows with missing values in low-completeness columns.
// Uses selectDisplayColumns to pick which cols to show.
func buildColumnCompletenessExamples(data *upload.SheetData, threshold float64) []IssueExample {
	if len(data.Rows) == 0 {
		return nil
	}
	// Find low-completeness column indices
	var lowCols []int
	for col := 0; col < data.ColCount; col++ {
		nonEmpty := 0
		for _, row := range data.Rows {
			if col < len(row) && !row[col].IsEmpty {
				nonEmpty++
			}
		}
		ratio := float64(nonEmpty) / float64(len(data.Rows))
		if ratio < threshold {
			lowCols = append(lowCols, col)
		}
	}
	if len(lowCols) == 0 {
		return nil
	}
	displayCols := selectDisplayColumns(data, lowCols)
	headers := getDisplayHeaders(data, displayCols)

	var examples []IssueExample
	for rowIdx, row := range data.Rows {
		if len(examples) >= maxExamples {
			break
		}
		// Check if this row has missing values in any low-completeness column
		hasMissing := false
		for _, col := range lowCols {
			if col >= len(row) || row[col].IsEmpty {
				hasMissing = true
				break
			}
		}
		if !hasMissing {
			continue
		}
		cells := getRowCells(row, displayCols)
		// Highlight cells that are empty AND in a low-completeness column
		var highlights []int
		for i, col := range displayCols {
			if col >= len(row) || row[col].IsEmpty {
				// Check if this col is one of the low-completeness cols
				for _, lc := range lowCols {
					if col == lc {
						highlights = append(highlights, i)
						break
					}
				}
			}
		}
		examples = append(examples, IssueExample{
			Headers:    headers,
			RowNumber:  rowIdx + data.HeaderRowIndex + 2,
			Cells:      cells,
			Highlights: highlights,
		})
	}
	return limitExamples(examples)
}

// buildFormatConsistencyExamples shows rows from mixed-format columns where different
// formats coexist, making the "mixing" visible. It iterates over each mixed column
// independently (up to 5 columns), picking 1-2 rows matching the dominant format
// (no highlight) and 2-3 rows that DON'T match (highlighted) for each column group.
func buildFormatConsistencyExamples(data *upload.SheetData) []IssueExample {
	// First find columns with mixed formats and their dominant type
	type colInfo struct {
		col          int
		dominantType FormatType
	}
	var mixedCols []colInfo
	for col := 0; col < data.ColCount; col++ {
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
			continue
		}
		dominantCount := 0
		var dominantType FormatType
		for ft, c := range formatCounts {
			if c > dominantCount {
				dominantCount = c
				dominantType = ft
			}
		}
		consistency := float64(dominantCount) / float64(nonEmptyCount)
		if consistency < 0.8 {
			mixedCols = append(mixedCols, colInfo{col: col, dominantType: dominantType})
		}
	}
	if len(mixedCols) == 0 {
		return nil
	}

	// Cap at 5 columns
	maxCols := 5
	if len(mixedCols) < maxCols {
		maxCols = len(mixedCols)
	}

	// Calculate per-column budget to fit within maxExamples
	perColBudget := maxExamples / maxCols
	if perColBudget < 2 {
		perColBudget = 2 // at least 1 dominant + 1 mismatch per column
	}

	var allExamples []IssueExample

	// Process each mixed column independently
	for ci := 0; ci < maxCols; ci++ {
		targetCol := mixedCols[ci]
		colLabel := getColumnName(data, targetCol.col)

		// Select display columns focused on this column
		displayCols := selectDisplayColumns(data, []int{targetCol.col})
		headers := getDisplayHeaders(data, displayCols)

		// Find the display index of the target column within displayCols
		targetDisplayIdx := -1
		for di, dc := range displayCols {
			if dc == targetCol.col {
				targetDisplayIdx = di
				break
			}
		}

		// Collect rows matching dominant format and rows NOT matching for THIS column
		var dominantRows []int
		var mismatchRows []int

		for rowIdx, row := range data.Rows {
			col := targetCol.col
			if col >= len(row) || row[col].IsEmpty {
				continue
			}
			ft := DetectFormatType(row[col].Raw)
			if ft == targetCol.dominantType {
				dominantRows = append(dominantRows, rowIdx)
			} else {
				mismatchRows = append(mismatchRows, rowIdx)
			}
		}

		// Build examples for this column using the per-column budget
		// Split budget: ~1/3 dominant + ~2/3 mismatch (at least 1 of each)
		dominantLimit := 1
		mismatchLimit := perColBudget - dominantLimit
		if mismatchLimit < 1 {
			mismatchLimit = 1
		}
		if len(dominantRows) < dominantLimit {
			dominantLimit = len(dominantRows)
		}
		if len(mismatchRows) < mismatchLimit {
			mismatchLimit = len(mismatchRows)
		}
		for i := 0; i < dominantLimit; i++ {
			rowIdx := dominantRows[i]
			row := data.Rows[rowIdx]
			cells := getRowCells(row, displayCols)

			// Build FormatLabels: label only the target column's cell
			formatLabels := make([]string, len(displayCols))
			for di, dc := range displayCols {
				if dc == targetCol.col && dc < len(row) && !row[dc].IsEmpty {
					formatLabels[di] = FormatTypeLabel(DetectFormatType(row[dc].Raw))
				}
			}

			allExamples = append(allExamples, IssueExample{
				Label:        colLabel,
				Headers:      headers,
				RowNumber:    rowIdx + data.HeaderRowIndex + 2,
				Cells:        cells,
				Highlights:   nil, // dominant format — no highlight
				FormatLabels: formatLabels,
			})
		}

		for i := 0; i < mismatchLimit; i++ {
			rowIdx := mismatchRows[i]
			row := data.Rows[rowIdx]
			cells := getRowCells(row, displayCols)

			// Build FormatLabels: label only the target column's cell
			formatLabels := make([]string, len(displayCols))
			for di, dc := range displayCols {
				if dc == targetCol.col && dc < len(row) && !row[dc].IsEmpty {
					formatLabels[di] = FormatTypeLabel(DetectFormatType(row[dc].Raw))
				}
			}

			// Highlight only the target column's display index (mismatch)
			var highlights []int
			if targetDisplayIdx >= 0 {
				highlights = []int{targetDisplayIdx}
			}

			allExamples = append(allExamples, IssueExample{
				Label:        colLabel,
				Headers:      headers,
				RowNumber:    rowIdx + data.HeaderRowIndex + 2,
				Cells:        cells,
				Highlights:   highlights,
				FormatLabels: formatLabels,
			})
		}
	}

	return limitExamples(allExamples)
}

// buildDuplicateExamples shows pairs of duplicate or near-duplicate rows.
// First tries exact duplicates (hash match). If none found, falls back to
// near-duplicate detection via Levenshtein distance on eligible columns.
// Displays up to 5 examples (pairs of rows).
func buildDuplicateExamples(data *upload.SheetData) []IssueExample {
	// Step 1: Try exact duplicates
	examples := buildExactDuplicateExamples(data)
	if len(examples) > 0 {
		return examples
	}

	// Step 2: Fall back to near-duplicate detection
	return buildNearDuplicateExamples(data)
}

// buildExactDuplicateExamples finds rows with identical content via hash.
// Skips mostly-empty rows (>70% cells empty) to match indicator logic.
func buildExactDuplicateExamples(data *upload.SheetData) []IssueExample {
	hashToRows := make(map[string][]int)
	var hashOrder []string
	for rowIdx, row := range data.Rows {
		// Skip mostly-empty rows (>70% cells empty)
		emptyCount := 0
		for i := 0; i < data.ColCount; i++ {
			if i >= len(row) || row[i].IsEmpty {
				emptyCount++
			}
		}
		if data.ColCount > 0 && float64(emptyCount)/float64(data.ColCount) > 0.7 {
			continue
		}
		h := hashRowForExamples(row, data.ColCount)
		if _, exists := hashToRows[h]; !exists {
			hashOrder = append(hashOrder, h)
		}
		hashToRows[h] = append(hashToRows[h], rowIdx)
	}

	// Pick display columns (all columns up to 6)
	var allCols []int
	limit := data.ColCount
	if limit > 6 {
		limit = 6
	}
	for i := 0; i < limit; i++ {
		allCols = append(allCols, i)
	}
	headers := getDisplayHeaders(data, allCols)

	var examples []IssueExample
	for _, h := range hashOrder {
		if len(examples) >= maxExamples {
			break
		}
		indices := hashToRows[h]
		if len(indices) < 2 {
			continue
		}
		// Show exactly 2 rows from this group (a pair)
		for _, idx := range indices[:2] {
			if len(examples) >= maxExamples {
				break
			}
			row := data.Rows[idx]
			cells := getRowCells(row, allCols)
			// Highlight all cells to indicate they match
			highlights := make([]int, len(allCols))
			for i := range allCols {
				highlights[i] = i
			}
			examples = append(examples, IssueExample{
				Headers:    headers,
				RowNumber:  idx + data.HeaderRowIndex + 2,
				Cells:      cells,
				Highlights: highlights,
			})
		}
	}
	return limitExamples(examples)
}

// nearDupPair holds a near-duplicate pair found in a specific column.
type nearDupPair struct {
	col    int
	value1 string
	value2 string
}

// buildNearDuplicateExamples finds near-duplicate values (Levenshtein ≤ 2) in eligible columns
// and shows the rows containing those values side by side.
func buildNearDuplicateExamples(data *upload.SheetData) []IssueExample {
	eligibleCols := selectEligibleColumns(data)
	if len(eligibleCols) == 0 {
		return nil
	}

	// Find near-duplicate pairs across eligible columns
	var pairs []nearDupPair
	for _, col := range eligibleCols {
		if len(pairs) >= maxExamples {
			break
		}
		uniqueValues := collectUniqueValues(data, col)

		// Only compare values with length > 3 to avoid false positives
		var longValues []string
		for _, v := range uniqueValues {
			if len([]rune(v)) > 3 {
				longValues = append(longValues, v)
			}
		}

		// Limit to first 200 values to avoid O(n²) explosion
		limit := len(longValues)
		if limit > 200 {
			limit = 200
		}
		for i := 0; i < limit && len(pairs) < maxExamples; i++ {
			for j := i + 1; j < limit && len(pairs) < maxExamples; j++ {
				if levenshteinDistance(longValues[i], longValues[j]) <= 2 {
					pairs = append(pairs, nearDupPair{
						col:    col,
						value1: longValues[i],
						value2: longValues[j],
					})
				}
			}
		}
	}

	if len(pairs) == 0 {
		return nil
	}

	var examples []IssueExample
	for _, pair := range pairs {
		if len(examples) >= maxExamples {
			break
		}

		// Find a row containing value1 and a row containing value2
		row1Idx := -1
		row2Idx := -1
		for rowIdx, row := range data.Rows {
			if pair.col < len(row) && !row[pair.col].IsEmpty {
				if row[pair.col].Raw == pair.value1 && row1Idx == -1 {
					row1Idx = rowIdx
				} else if row[pair.col].Raw == pair.value2 && row2Idx == -1 {
					row2Idx = rowIdx
				}
			}
			if row1Idx >= 0 && row2Idx >= 0 {
				break
			}
		}
		if row1Idx < 0 || row2Idx < 0 {
			continue
		}

		// Build display columns: include the near-duplicate column + col 0 for context
		displayCols := selectDisplayColumns(data, []int{pair.col})
		headers := getDisplayHeaders(data, displayCols)

		// Find the display index of the near-duplicate column for highlighting
		var highlights []int
		for i, col := range displayCols {
			if col == pair.col {
				highlights = append(highlights, i)
				break
			}
		}

		// Show both rows, highlighting only the near-duplicate column
		cells1 := getRowCells(data.Rows[row1Idx], displayCols)
		examples = append(examples, IssueExample{
			Headers:    headers,
			RowNumber:  row1Idx + data.HeaderRowIndex + 2,
			Cells:      cells1,
			Highlights: highlights,
		})

		if len(examples) >= maxExamples {
			break
		}

		cells2 := getRowCells(data.Rows[row2Idx], displayCols)
		examples = append(examples, IssueExample{
			Headers:    headers,
			RowNumber:  row2Idx + data.HeaderRowIndex + 2,
			Cells:      cells2,
			Highlights: highlights,
		})
	}

	return limitExamples(examples)
}

// buildSingleStructureExamples shows actual problematic rows for a single structure issue type.
// Returns up to maxExamples examples for the given problem.
func buildSingleStructureExamples(data *upload.SheetData, problem string) []IssueExample {
	var allCols []int
	limit := data.ColCount
	if limit > 6 {
		limit = 6
	}
	for i := 0; i < limit; i++ {
		allCols = append(allCols, i)
	}
	headers := getDisplayHeaders(data, allCols)

	var examples []IssueExample

	switch problem {
	case "合併儲存格":
		// Sort merged cells by row number ascending
		type sortedMerge struct {
			mr      upload.MergedRange
			sortKey int
		}
		var sorted []sortedMerge
		for _, mr := range data.MergedCells {
			// Only include merges that span multiple columns (visually meaningful)
			colSpan := mr.EndCol - mr.StartCol + 1
			if colSpan < 2 {
				continue
			}
			sorted = append(sorted, sortedMerge{mr: mr, sortKey: mr.StartRow})
		}
		// Sort ascending
		for i := 0; i < len(sorted)-1; i++ {
			for j := i + 1; j < len(sorted); j++ {
				if sorted[i].sortKey > sorted[j].sortKey {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}
			}
		}

		for _, sm := range sorted {
			if len(examples) >= maxExamples {
				break
			}
			mr := sm.mr
			// Coordinates are 1-based from excelize
			mrStartCol := mr.StartCol - 1 // convert to 0-based
			mrEndCol := mr.EndCol - 1     // convert to 0-based
			mrSheetRow0 := mr.StartRow - 1 // 0-based sheet row

			// Convert to data row index
			dataRowIdx := mrSheetRow0 - data.HeaderRowIndex - 1
			if dataRowIdx < 0 || dataRowIdx >= len(data.Rows) {
				continue
			}

			row := data.Rows[dataRowIdx]
			cells := getRowCells(row, allCols)

			// Find colspan in display columns
			var merges []CellMerge
			for i, col := range allCols {
				if col >= mrStartCol && col <= mrEndCol {
					span := 0
					for j := i; j < len(allCols) && allCols[j] <= mrEndCol; j++ {
						span++
					}
					if span > 1 {
						merges = append(merges, CellMerge{StartCol: i, Span: span})
					}
					break
				}
			}

			// Highlight merged cells
			var highlights []int
			for i, col := range allCols {
				if col >= mrStartCol && col <= mrEndCol {
					highlights = append(highlights, i)
				}
			}

			examples = append(examples, IssueExample{
				Headers:    headers,
				RowNumber:  mr.StartRow, // 1-based Excel row
				Cells:      cells,
				Highlights: highlights,
				Merges:     merges,
			})
		}

	case "多層標題":
		// Show pre-header rows from RawFirstRows + the header row as context
		for i := 0; i < data.HeaderRowIndex && len(examples) < maxExamples; i++ {
			if i < len(data.RawFirstRows) {
				rawRow := data.RawFirstRows[i]
				cells := make([]string, len(allCols))
				for ci, col := range allCols {
					if col < len(rawRow) {
						val := rawRow[col]
						if len([]rune(val)) > 40 {
							val = string([]rune(val)[:40]) + "…"
						}
						cells[ci] = val
					}
				}
				highlights := make([]int, len(allCols))
				for hi := range allCols {
					highlights[hi] = hi
				}
				examples = append(examples, IssueExample{
					Headers:    headers,
					RowNumber:  i + 1,
					Cells:      cells,
					Highlights: highlights,
				})
			}
		}
		// Also show the header row itself as context (if space remains)
		if len(examples) < maxExamples && data.HeaderRowIndex < len(data.RawFirstRows) {
			rawRow := data.RawFirstRows[data.HeaderRowIndex]
			cells := make([]string, len(allCols))
			for ci, col := range allCols {
				if col < len(rawRow) {
					val := rawRow[col]
					if len([]rune(val)) > 40 {
						val = string([]rune(val)[:40]) + "…"
					}
					cells[ci] = val
				}
			}
			examples = append(examples, IssueExample{
				Headers:    headers,
				RowNumber:  data.HeaderRowIndex + 1,
				Cells:      cells,
				Highlights: nil, // header row is context, not highlighted
			})
		}

	case "備註混入資料欄":
		noteCol, noteRowIdx := findNoteInDataRow(data)
		// Use consistent display columns for all note examples
		displayCols := allCols
		if noteCol >= 0 {
			displayCols = selectDisplayColumns(data, []int{noteCol})
		}
		noteHeaders := getDisplayHeaders(data, displayCols)

		if noteCol >= 0 && noteRowIdx >= 0 {
			row := data.Rows[noteRowIdx]
			cells := getRowCells(row, displayCols)

			// Highlight the column with abnormally long text
			var highlights []int
			for i, col := range displayCols {
				if col == noteCol {
					highlights = append(highlights, i)
					break
				}
			}

			// Check if this row also has merged cells
			sheetRow1Based := noteRowIdx + data.HeaderRowIndex + 2
			var merges []CellMerge
			for _, mr := range data.MergedCells {
				if mr.StartRow == sheetRow1Based {
					mrStartCol := mr.StartCol - 1
					mrEndCol := mr.EndCol - 1
					colSpan := mrEndCol - mrStartCol + 1
					if colSpan < 2 {
						continue
					}
					for i, col := range displayCols {
						if col >= mrStartCol && col <= mrEndCol {
							span := 0
							for j := i; j < len(displayCols) && displayCols[j] <= mrEndCol; j++ {
								span++
							}
							if span > 1 {
								merges = append(merges, CellMerge{StartCol: i, Span: span})
							}
							break
						}
					}
				}
			}

			examples = append(examples, IssueExample{
				Headers:    noteHeaders,
				RowNumber:  sheetRow1Based,
				Cells:      cells,
				Highlights: highlights,
				Merges:     merges,
			})

			// Find additional examples with Chinese bracket notes in displayed columns only
			shown := make(map[int]bool)
			shown[noteRowIdx] = true
			for rowIdx, row := range data.Rows {
				if shown[rowIdx] || len(examples) >= maxExamples {
					continue
				}
				var addHighlights []int
				for i, col := range displayCols {
					if col < len(row) && !row[col].IsEmpty && hasChineseBracketNote(row[col].Raw) {
						addHighlights = append(addHighlights, i)
					}
				}
				if len(addHighlights) > 0 {
					addCells := getRowCells(row, displayCols)
					examples = append(examples, IssueExample{
						Headers:    noteHeaders,
						RowNumber:  rowIdx + data.HeaderRowIndex + 2,
						Cells:      addCells,
						Highlights: addHighlights,
					})
					shown[rowIdx] = true
				}
			}
		}

		// Also show cells with comments (red triangle) if no text-based note was found
		if noteCol < 0 && len(data.CommentCells) > 0 && len(examples) < maxExamples {
			// Find first comment cell that's not in a date column
			for _, cl := range data.CommentCells {
				if isDateColumn(data, cl.Col) {
					continue
				}
				displayCols := selectDisplayColumns(data, []int{cl.Col})
				noteHeaders := getDisplayHeaders(data, displayCols)
				row := data.Rows[cl.Row]
				cells := getRowCells(row, displayCols)
				var highlights []int
				for i, col := range displayCols {
					if col == cl.Col {
						highlights = append(highlights, i)
						break
					}
				}
				examples = append(examples, IssueExample{
					Label:      "儲存格含有批註 (紅色三角形)",
					Headers:    noteHeaders,
					RowNumber:  cl.Row + data.HeaderRowIndex + 2,
					Cells:      cells,
					Highlights: highlights,
				})
				break
			}
		}

		// Also show cells with strikethrough formatting (only if no other notes found)
		if noteCol < 0 && len(data.StrikethroughCells) > 0 && len(examples) < maxExamples {
			// Find first strikethrough cell that's not in a date column
			for _, cl := range data.StrikethroughCells {
				if isDateColumn(data, cl.Col) {
					continue
				}
				displayCols := selectDisplayColumns(data, []int{cl.Col})
				noteHeaders := getDisplayHeaders(data, displayCols)
				row := data.Rows[cl.Row]
				cells := getRowCells(row, displayCols)
				var highlights []int
				for i, col := range displayCols {
					if col == cl.Col {
						highlights = append(highlights, i)
						break
					}
				}
				examples = append(examples, IssueExample{
					Label:      "儲存格含有刪除線格式",
					Headers:    noteHeaders,
					RowNumber:  cl.Row + data.HeaderRowIndex + 2,
					Cells:      cells,
					Highlights: highlights,
				})
				break
			}
		}

	case "儲存格含換行":
		// First, collect ALL columns that have newline issues
		var newlineCols []int
		for col := 0; col < data.ColCount; col++ {
			for _, row := range data.Rows {
				if col < len(row) && !row[col].IsEmpty && strings.Contains(row[col].Raw, "\n") {
					newlineCols = append(newlineCols, col)
					break
				}
			}
		}
		if len(newlineCols) == 0 {
			break
		}

		// Use consistent display columns for all examples
		displayCols := selectDisplayColumns(data, newlineCols)
		exHeaders := getDisplayHeaders(data, displayCols)

		// Find rows with newlines in any of these columns
		for rowIdx, row := range data.Rows {
			if len(examples) >= maxExamples {
				break
			}
			var highlights []int
			hasNewline := false
			for i, col := range displayCols {
				if col < len(row) && !row[col].IsEmpty && strings.Contains(row[col].Raw, "\n") {
					highlights = append(highlights, i)
					hasNewline = true
				}
			}
			if hasNewline {
				cells := getRowCells(row, displayCols)
				examples = append(examples, IssueExample{
					Headers:    exHeaders,
					RowNumber:  rowIdx + data.HeaderRowIndex + 2,
					Cells:      cells,
					Highlights: highlights,
				})
			}
		}

	case "小計/合計列":
		// Find columns that contain subtotal keywords
		var keywordCols []int
		keywordColSet := make(map[int]bool)
		keywords := []string{"小計", "合計", "總計", "subtotal", "total"}
		for _, row := range data.Rows {
			for colIdx, cell := range row {
				if cell.IsEmpty {
					continue
				}
				lower := strings.ToLower(strings.TrimSpace(cell.Raw))
				for _, kw := range keywords {
					if strings.Contains(lower, kw) && !keywordColSet[colIdx] {
						keywordColSet[colIdx] = true
						keywordCols = append(keywordCols, colIdx)
					}
				}
			}
		}
		displayCols := selectDisplayColumns(data, keywordCols)
		exHeaders := getDisplayHeaders(data, displayCols)

		for rowIdx, row := range data.Rows {
			if len(examples) >= maxExamples {
				break
			}
			if isSubtotalRow(row) {
				cells := getRowCells(row, displayCols)
				// Highlight the cell(s) containing the keyword
				var highlights []int
				for i, col := range displayCols {
					if col < len(row) && !row[col].IsEmpty {
						lower := strings.ToLower(strings.TrimSpace(row[col].Raw))
						for _, kw := range keywords {
							if strings.Contains(lower, kw) {
								highlights = append(highlights, i)
								break
							}
						}
					}
				}
				examples = append(examples, IssueExample{
					Headers:    exHeaders,
					RowNumber:  rowIdx + data.HeaderRowIndex + 2,
					Cells:      cells,
					Highlights: highlights,
				})
			}
		}

	case "多表格混在同一 sheet":
		// Find data blocks separated by 2+ consecutive empty rows.
		// Show each block's first 3 data rows with a label, and one empty gap row between.
		type dataBlock struct {
			startIdx int // first row index of the block (0-based into data.Rows)
			endIdx   int // last row index of the block (inclusive)
		}

		var blocks []dataBlock
		inBlock := false
		consecutiveEmpty := 0
		var currentBlockStart int

		for rowIdx, row := range data.Rows {
			empty := isRowEmpty(row, data.ColCount)
			if empty {
				consecutiveEmpty++
				if inBlock && consecutiveEmpty >= 2 {
					// End current block at the row before the empty streak
					blocks = append(blocks, dataBlock{startIdx: currentBlockStart, endIdx: rowIdx - consecutiveEmpty})
					inBlock = false
				}
			} else {
				if !inBlock {
					currentBlockStart = rowIdx
					inBlock = true
				}
				consecutiveEmpty = 0
			}
		}
		// Close last block if still open
		if inBlock {
			blocks = append(blocks, dataBlock{startIdx: currentBlockStart, endIdx: len(data.Rows) - 1})
		}

		if len(blocks) >= 2 {
			// Determine which columns have data across blocks
			var dataCols []int
			dataColSet := make(map[int]bool)
			for _, blk := range blocks {
				for ri := blk.startIdx; ri <= blk.endIdx && ri < len(data.Rows); ri++ {
					for col := 0; col < data.ColCount; col++ {
						if col < len(data.Rows[ri]) && !data.Rows[ri][col].IsEmpty && !dataColSet[col] {
							dataColSet[col] = true
							dataCols = append(dataCols, col)
						}
					}
				}
			}
			multiDisplayCols := selectDisplayColumns(data, dataCols)
			multiHeaders := getDisplayHeaders(data, multiDisplayCols)

			// Show at most 2 blocks
			maxBlocks := 2
			if len(blocks) < maxBlocks {
				maxBlocks = len(blocks)
			}

			for bi := 0; bi < maxBlocks; bi++ {
				blk := blocks[bi]
				// Excel row numbers (1-based)
				blkStartRow := blk.startIdx + data.HeaderRowIndex + 2
				blkEndRow := blk.endIdx + data.HeaderRowIndex + 2
				label := fmt.Sprintf("表格%s（第 %d~%d 列）", []string{"一", "二"}[bi], blkStartRow, blkEndRow)

				// Collect first 3 non-empty rows from this block
				shown := 0
				for ri := blk.startIdx; ri <= blk.endIdx && shown < 3; ri++ {
					if isRowEmpty(data.Rows[ri], data.ColCount) {
						continue
					}
					cells := getRowCells(data.Rows[ri], multiDisplayCols)
					// Highlight all cells to show this is a separate table block
					rowHighlights := make([]int, len(multiDisplayCols))
					for i := range multiDisplayCols {
						rowHighlights[i] = i
					}
					examples = append(examples, IssueExample{
						Label:      label,
						Headers:    multiHeaders,
						RowNumber:  ri + data.HeaderRowIndex + 2,
						Cells:      cells,
						Highlights: rowHighlights,
					})
					shown++
				}

				// After first block, insert one empty gap row (no highlights — context only)
				if bi == 0 && maxBlocks > 1 {
					gapRowIdx := blk.endIdx + 1
					if gapRowIdx < len(data.Rows) {
						emptyCells := make([]string, len(multiDisplayCols))
						examples = append(examples, IssueExample{
							Label:      "（空白列）",
							Headers:    multiHeaders,
							RowNumber:  gapRowIdx + data.HeaderRowIndex + 2,
							Cells:      emptyCells,
							Highlights: nil,
						})
					}
				}
			}
		}
	}

	// Sort examples by row number for consistent display
	sort.Slice(examples, func(i, j int) bool {
		return examples[i].RowNumber < examples[j].RowNumber
	})
	return limitStructureExamples(examples)
}

// maxStructureExamples is the maximum number of examples for structural issues (higher than default).
const maxStructureExamples = 10

// limitStructureExamples truncates a slice to at most maxStructureExamples items.
func limitStructureExamples(items []IssueExample) []IssueExample {
	if len(items) <= maxStructureExamples {
		return items
	}
	return items[:maxStructureExamples]
}

// buildStructuralReadinessExamples produces a checklist of failed AI readiness conditions.
// Only shows items that FAILED (no green checkmarks for passing items).
func buildStructuralReadinessExamples(data *upload.SheetData, scores IndicatorScores) []IssueExample {
	headers := []string{"待改善項目", "目前狀態"}
	var examples []IssueExample

	// 1. Check column names quality
	if !hasGoodColumnNames(data) {
		examples = append(examples, IssueExample{
			Headers:    headers,
			RowNumber:  0,
			Cells:      []string{"欄位名稱", "部分欄位名稱為空或過短"},
			Highlights: []int{1},
		})
	}

	// 2. Check column fill rate — any column < 50% filled
	if len(data.Rows) > 0 {
		for col := 0; col < data.ColCount; col++ {
			nonEmpty := 0
			for _, row := range data.Rows {
				if col < len(row) && !row[col].IsEmpty {
					nonEmpty++
				}
			}
			ratio := float64(nonEmpty) / float64(len(data.Rows))
			if ratio < 0.5 {
				colName := getColumnName(data, col)
				examples = append(examples, IssueExample{
					Headers:    headers,
					RowNumber:  0,
					Cells:      []string{"每欄資料量", fmt.Sprintf("欄位「%s」超過一半為空", colName)},
					Highlights: []int{1},
				})
				break // show only first bad column
			}
		}
	}

	// 3. Format consistency
	if scores.FormatConsistency < 80 {
		examples = append(examples, IssueExample{
			Headers:    headers,
			RowNumber:  0,
			Cells:      []string{"格式一致性", "部分欄位格式混合使用"},
			Highlights: []int{1},
		})
	}

	// 4. Duplicate/similar uniqueness
	if scores.DuplicateSimilar < 80 {
		examples = append(examples, IssueExample{
			Headers:    headers,
			RowNumber:  0,
			Cells:      []string{"資料唯一性", "存在相似或重複的資料"},
			Highlights: []int{1},
		})
	}

	// 5. Table structure
	if scores.TableStructure < 100 {
		examples = append(examples, IssueExample{
			Headers:    headers,
			RowNumber:  0,
			Cells:      []string{"表格結構", "含有合併儲存格或小計列等"},
			Highlights: []int{1},
		})
	}

	if len(examples) == 0 {
		return nil
	}
	return examples
}

// findNoteInDataRow finds the text column with highest length variance and returns
// findNoteInDataRow finds a cell with note-like content.
// Priority: 1) cell with \n AND Chinese bracket, 2) cell with Chinese bracket, 3) cell with \n.
// Returns (colIdx, rowIdx) or (-1, -1) if not found.
func findNoteInDataRow(data *upload.SheetData) (int, int) {
	if len(data.Rows) == 0 {
		return -1, -1
	}

	// Priority 1: Find a cell with newline AND Chinese bracket note (strongest signal)
	for col := 0; col < data.ColCount; col++ {
		if isDateColumn(data, col) {
			continue
		}
		for rowIdx, row := range data.Rows {
			if col < len(row) && !row[col].IsEmpty {
				val := row[col].Raw
				if strings.Contains(val, "\n") && hasChineseBracketNote(val) {
					return col, rowIdx
				}
			}
		}
	}

	// Priority 2: Find a cell with Chinese bracket note
	for col := 0; col < data.ColCount; col++ {
		if isDateColumn(data, col) {
			continue
		}
		for rowIdx, row := range data.Rows {
			if col < len(row) && !row[col].IsEmpty {
				if hasChineseBracketNote(row[col].Raw) {
					return col, rowIdx
				}
			}
		}
	}

	// Priority 3: Find a cell with newline (skip date-like columns)
	for col := 0; col < data.ColCount; col++ {
		if isDateColumn(data, col) {
			continue
		}

		for rowIdx, row := range data.Rows {
			if col < len(row) && !row[col].IsEmpty {
				if strings.Contains(row[col].Raw, "\n") {
					return col, rowIdx
				}
			}
		}
	}

	// Priority 4: Fallback to numeric column with notes
	return findNoteInNumericCol(data)
}

// isDateColumn checks if a column is predominantly date values (>30%).
func isDateColumn(data *upload.SheetData, col int) bool {
	dateCount := 0
	nonEmpty := 0
	for _, row := range data.Rows {
		if col < len(row) && !row[col].IsEmpty {
			nonEmpty++
			val := strings.TrimSpace(row[col].Raw)
			if DetectFormatType(val) == FormatDate || looksLikeDatePattern(val) {
				dateCount++
			}
		}
	}
	return nonEmpty > 0 && float64(dateCount)/float64(nonEmpty) > 0.3
}

// findNoteInNumericCol finds a numeric-dominant column where some values contain
// notes mixed in (like "500(先匯)", "700(p2p)"). Returns col index and row index,
// or -1, -1 if not found.
func findNoteInNumericCol(data *upload.SheetData) (int, int) {
	if len(data.Rows) == 0 {
		return -1, -1
	}

	for col := 0; col < data.ColCount; col++ {
		numericCount := 0
		nonEmptyCount := 0
		noteRowIdx := -1

		for rowIdx, row := range data.Rows {
			if col >= len(row) || row[col].IsEmpty {
				continue
			}
			nonEmptyCount++
			val := row[col].Raw
			if isParseableAsNumber(val) {
				numericCount++
			} else if hasNotePattern(val) && noteRowIdx == -1 {
				noteRowIdx = rowIdx
			}
		}

		// Column should be predominantly numeric (>60% parseable as number)
		// but have at least one value with note pattern
		if nonEmptyCount > 0 && float64(numericCount)/float64(nonEmptyCount) > 0.6 && noteRowIdx >= 0 {
			return col, noteRowIdx
		}
	}
	return -1, -1
}

// hasNotePattern checks if a value looks like a number with a note appended.
// Patterns: contains parentheses (both half-width and full-width), or
// a number followed by Chinese/letter text.
func hasNotePattern(val string) bool {
	// Only flag brackets that contain Chinese characters (user notes)
	// Not technical notation like (P/N: xxx) or (HATTELAND)
	if idx := strings.IndexAny(val, "(（"); idx >= 0 {
		closeIdx := strings.IndexAny(val[idx+1:], ")）")
		if closeIdx > 0 {
			inner := val[idx+1 : idx+1+closeIdx]
			for _, r := range inner {
				if r >= 0x4E00 && r <= 0x9FFF {
					return true
				}
			}
		}
	}
	return false
}

// mergeIssuesByTitle merges issues that share the same Title.
// When merging: keep the higher severity, sum AffectedRows, combine descriptions with \n,
// concatenate examples (limited to maxExamples total), set Indicator to "completeness",
// keep the first issue's Unit.
func mergeIssuesByTitle(issues []Issue) []Issue {
	if len(issues) == 0 {
		return issues
	}

	type mergeEntry struct {
		issue *Issue
		order int // preserve original order
	}

	seen := make(map[string]*mergeEntry)
	var order []string

	for i := range issues {
		title := issues[i].Title
		if entry, exists := seen[title]; exists {
			// Merge into existing
			// Use the larger AffectedRows (don't sum — they may overlap)
			if issues[i].AffectedRows > entry.issue.AffectedRows {
				entry.issue.AffectedRows = issues[i].AffectedRows
			}
			entry.issue.Description += "\n" + issues[i].Description
			// Keep higher severity
			entry.issue.Severity = higherSeverity(entry.issue.Severity, issues[i].Severity)
			// Concatenate examples up to maxExamples
			remaining := maxExamples - len(entry.issue.Examples)
			if remaining > 0 {
				toAdd := issues[i].Examples
				if len(toAdd) > remaining {
					toAdd = toAdd[:remaining]
				}
				entry.issue.Examples = append(entry.issue.Examples, toAdd...)
			}
			// Set indicator to "completeness" (combined)
			entry.issue.Indicator = "completeness"
		} else {
			// First occurrence — make a copy
			issueCopy := issues[i]
			seen[title] = &mergeEntry{issue: &issueCopy, order: len(order)}
			order = append(order, title)
		}
	}

	// Rebuild slice in original order
	result := make([]Issue, 0, len(order))
	for _, title := range order {
		result = append(result, *seen[title].issue)
	}
	return result
}

// addEnglishTranslations adds English title and description to issues
func addEnglishTranslations(issues []Issue) []Issue {
	titleMap := map[string]string{
		"資料大量缺漏":         "Significant Data Gaps",
		"部分資料缺漏":         "Partial Data Gaps",
		"格式混用":           "Mixed Formats",
		"日期格式不一致":        "Inconsistent Date Formats",
		"疑似重複資料":         "Suspected Duplicate Data",
		"客戶名稱不一致":        "Inconsistent Client Names",
		"表格結構問題":         "Table Structure Issues",
		"AI 應用完備度":       "AI Application Readiness",
		"「同XX」引用未填入實際值":  "Cell References Not Filled",
		"空白標題欄":          "Empty Header Columns",
		"行內備註混入資料欄":      "Inline Remarks in Data Columns",
		"欄位型別不一致":        "Inconsistent Column Types",
		"儲存格含刪除線格式":      "Cells with Strikethrough Format",
	}
	unitMap := map[string]string{
		"列受影響": "rows affected",
		"列":    "rows",
		"組":    "groups",
		"處":    "issues",
		"欄":    "columns",
	}
	for i := range issues {
		if en, ok := titleMap[issues[i].Title]; ok {
			issues[i].TitleEn = en
		} else {
			issues[i].TitleEn = issues[i].Title
		}
		if en, ok := unitMap[issues[i].Unit]; ok {
			issues[i].DescriptionEn = en
		}
		// Generate English description from Chinese description patterns
		issues[i].DescriptionEn = translateDescription(issues[i].Description, issues[i].Indicator)
	}
	return issues
}

// translateDescription converts Chinese issue descriptions to English equivalents
func translateDescription(desc string, indicator string) string {
	// Pattern-based translation for common description formats
	switch indicator {
	case "row_completeness", "column_completeness", "completeness":
		return "Multiple rows contain empty cells. Consider filling missing data to improve quality."
	case "format_consistency":
		return "Inconsistent data formats detected within columns (mixed date formats, number formats, etc.)"
	case "duplicate_similar":
		return "Suspected duplicate or near-duplicate rows detected."
	case "name_variants":
		return "Multiple spelling variations detected for the same entity. Consider normalizing."
	case "table_structure":
		return "Table structure issues detected (merged cells, subtotal rows, multi-table layout, etc.)"
	case "ai_query_readiness":
		return "Some structural requirements for AI analysis are not met."
	case "cell_reference_placeholder":
		return "Cells contain references like 'same as XX' instead of actual values. AI cannot interpret these."
	case "empty_header":
		return "Some header columns are blank. This affects data schema quality."
	case "inline_remark":
		return "Parenthetical remarks found mixed into structured data columns. Consider separating."
	case "column_type_mismatch":
		return "Mixed data types within columns (e.g. numbers and text in the same column)."
	case "strikethrough_formatting":
		return "Cells with strikethrough formatting detected. These may represent cancelled or invalid data."
	default:
		return desc
	}
}

// higherSeverity returns the higher of two severity levels.
func higherSeverity(a, b string) string {
	rank := map[string]int{"High": 3, "Medium": 2, "Low": 1}
	if rank[a] >= rank[b] {
		return a
	}
	return b
}

// hashRowForExamples creates a simple hash string for duplicate detection in examples.
func hashRowForExamples(row []upload.CellValue, colCount int) string {
	var parts []string
	for i := 0; i < colCount; i++ {
		if i < len(row) {
			parts = append(parts, row[i].Raw)
		} else {
			parts = append(parts, "")
		}
	}
	return strings.Join(parts, "|")
}

// isSubtotalRow checks if a row looks like a subtotal/total row.
func isSubtotalRow(row []upload.CellValue) bool {
	keywords := []string{"小計", "合計", "總計", "subtotal", "total"}
	for _, cell := range row {
		if cell.IsEmpty {
			continue
		}
		lower := strings.ToLower(cell.Raw)
		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				return true
			}
		}
	}
	return false
}

// findCompanyColumn finds the first column whose header matches company/client keywords.
// Returns column index or -1 if not found.
func findCompanyColumn(data *upload.SheetData) int {
	keywords := []string{"客戶", "公司", "廠商", "client", "company", "vendor", "supplier", "customer"}
	for col, header := range data.Headers {
		lower := strings.ToLower(header)
		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				return col
			}
		}
	}
	return -1
}

// nameVariantGroup represents a group of name variants that should be unified
type nameVariantGroup struct {
	normalizedKey  string
	variants       []string // different spellings
	canonicalCount int      // count of the most common variant
	totalRows      int      // total rows across all variants
}

// detectNameVariants finds company name variants using suffix removal + grouping.
func detectNameVariants(data *upload.SheetData, col int) []nameVariantGroup {
	suffixes := []string{"股份有限公司", "有限公司", "公司", "Company", "Corp.", "Inc.", "Ltd.", "Co."}

	// Group by normalized name
	type variantInfo struct {
		value string
		count int
	}
	groups := make(map[string][]variantInfo)

	for _, row := range data.Rows {
		if col >= len(row) || row[col].IsEmpty {
			continue
		}
		val := strings.TrimSpace(row[col].Raw)
		// Remove newlines for comparison
		val = strings.ReplaceAll(val, "\n", " ")
		if val == "" {
			continue
		}

		normalized := removeSuffixesForComparison(val, suffixes)
		normalizedKey := strings.ToLower(strings.TrimSpace(normalized))

		// Find or add variant
		found := false
		for i, vi := range groups[normalizedKey] {
			if vi.value == val {
				groups[normalizedKey][i].count++
				found = true
				break
			}
		}
		if !found {
			groups[normalizedKey] = append(groups[normalizedKey], variantInfo{value: val, count: 1})
		}
	}

	// Filter: only keep groups with >1 variant
	var result []nameVariantGroup
	for key, variants := range groups {
		if len(variants) <= 1 {
			continue
		}
		g := nameVariantGroup{normalizedKey: key}
		maxCount := 0
		for _, vi := range variants {
			g.variants = append(g.variants, vi.value)
			g.totalRows += vi.count
			if vi.count > maxCount {
				maxCount = vi.count
			}
		}
		g.canonicalCount = maxCount
		result = append(result, g)
	}

	return result
}

// removeSuffixesForComparison removes known company suffixes for grouping comparison.
func removeSuffixesForComparison(name string, suffixes []string) string {
	result := strings.TrimSpace(name)
	for _, suffix := range suffixes {
		if strings.HasSuffix(result, suffix) {
			result = strings.TrimSpace(strings.TrimSuffix(result, suffix))
			break
		}
	}
	return result
}

// buildNameVariantExamples shows the variant groups as examples.
// Each example shows the company column with different variants highlighted.
func buildNameVariantExamples(data *upload.SheetData, companyCol int, groups []nameVariantGroup) []IssueExample {
	displayCols := selectDisplayColumns(data, []int{companyCol})
	headers := getDisplayHeaders(data, displayCols)

	var examples []IssueExample
	for _, g := range groups {
		if len(examples) >= maxExamples {
			break
		}
		// Find a row for each variant (show first 2 variants of this group)
		shown := 0
		for rowIdx, row := range data.Rows {
			if shown >= 2 || len(examples) >= maxExamples {
				break
			}
			if companyCol >= len(row) || row[companyCol].IsEmpty {
				continue
			}
			val := strings.TrimSpace(strings.ReplaceAll(row[companyCol].Raw, "\n", " "))
			// Check if this row has one of the non-canonical variants
			for _, variant := range g.variants {
				if val == variant {
					cells := getRowCells(row, displayCols)
					var highlights []int
					for i, col := range displayCols {
						if col == companyCol {
							highlights = append(highlights, i)
							break
						}
					}
					examples = append(examples, IssueExample{
						Headers:    headers,
						RowNumber:  rowIdx + data.HeaderRowIndex + 2,
						Cells:      cells,
						Highlights: highlights,
					})
					shown++
					break
				}
			}
		}
	}

	return examples
}

// cellReferencePlaceholderRe matches cells containing "同" followed by a cell reference.
// Allows optional whitespace, mixed case letters (e.g., 同AH2, 同 ah2, 同aH2).
var cellReferencePlaceholderRe = regexp.MustCompile(`^同\s*[A-Za-z]+\d+$`)

// DetectCellReferencePlaceholders detects cells containing "同" followed by a cell reference
// (e.g., 同AH2, 同AI6). These are human-entered placeholders meaning "same as cell XX"
// and cannot be used by AI for numeric computation.
func DetectCellReferencePlaceholders(data *upload.SheetData) []Issue {
	type flaggedCell struct {
		rowIdx int
		colIdx int
	}

	var flagged []flaggedCell
	for rowIdx, row := range data.Rows {
		for colIdx := 0; colIdx < data.ColCount; colIdx++ {
			if colIdx >= len(row) || row[colIdx].IsEmpty {
				continue
			}
			val := strings.TrimSpace(row[colIdx].Raw)
			if cellReferencePlaceholderRe.MatchString(val) {
				flagged = append(flagged, flaggedCell{rowIdx: rowIdx, colIdx: colIdx})
			}
		}
	}

	if len(flagged) == 0 {
		return nil
	}

	// Determine which columns contain flagged cells
	affectedColSet := make(map[int]bool)
	for _, f := range flagged {
		affectedColSet[f.colIdx] = true
	}

	// Determine affected rows (unique row indices)
	affectedRowSet := make(map[int]bool)
	for _, f := range flagged {
		affectedRowSet[f.rowIdx] = true
	}

	// Check if any flagged cell is in a numeric column (>70% numeric)
	inNumericCol := false
	for col := range affectedColSet {
		if isColumnNumericForPlaceholder(data, col) {
			inNumericCol = true
			break
		}
	}

	// Build description — use plain language that non-technical users can understand
	desc := fmt.Sprintf("有 %d 個儲存格寫著「同XX」（例如「同AH2」），意思是「跟上面某格一樣」，但 AI 無法理解這種寫法", len(flagged))
	if inNumericCol {
		desc += "。這些出現在金額欄位中，導致 AI 無法計算正確的合計金額"
	}

	// Build examples (up to 5), grouped by row so all flagged cells in the same row are highlighted
	var problemCols []int
	for col := range affectedColSet {
		problemCols = append(problemCols, col)
	}
	displayCols := selectDisplayColumns(data, problemCols)
	headers := getDisplayHeaders(data, displayCols)

	// Group flagged cells by row
	rowToFlaggedCols := make(map[int][]int)
	var rowOrder []int
	for _, f := range flagged {
		if _, exists := rowToFlaggedCols[f.rowIdx]; !exists {
			rowOrder = append(rowOrder, f.rowIdx)
		}
		rowToFlaggedCols[f.rowIdx] = append(rowToFlaggedCols[f.rowIdx], f.colIdx)
	}

	var examples []IssueExample
	for _, rowIdx := range rowOrder {
		if len(examples) >= maxExamples {
			break
		}
		row := data.Rows[rowIdx]
		cells := getRowCells(row, displayCols)

		// Highlight ALL flagged columns in this row
		flaggedColsInRow := rowToFlaggedCols[rowIdx]
		var highlights []int
		for i, col := range displayCols {
			for _, fc := range flaggedColsInRow {
				if col == fc {
					highlights = append(highlights, i)
					break
				}
			}
		}

		examples = append(examples, IssueExample{
			Headers:    headers,
			RowNumber:  rowIdx + data.HeaderRowIndex + 2,
			Cells:      cells,
			Highlights: highlights,
		})
	}

	return []Issue{
		{
			Title:        "「同XX」引用未填入實際值",
			Severity:     "High",
			AffectedRows: len(affectedRowSet),
			Unit:         "列受影響",
			Description:  desc,
			Examples:     limitExamples(examples),
			Indicator:    "cell_reference_placeholder",
		},
	}
}

// isColumnNumericForPlaceholder checks if a column is predominantly numeric (>70% of non-empty cells).
// Used by DetectCellReferencePlaceholders to determine if the placeholder is in a numeric column.
func isColumnNumericForPlaceholder(data *upload.SheetData, col int) bool {
	numericCount := 0
	nonEmptyCount := 0
	for _, row := range data.Rows {
		if col >= len(row) || row[col].IsEmpty {
			continue
		}
		nonEmptyCount++
		if isParseableAsNumber(row[col].Raw) {
			numericCount++
		}
	}
	if nonEmptyCount == 0 {
		return false
	}
	return float64(numericCount)/float64(nonEmptyCount) > 0.70
}

// DetectEmptyHeaders detects columns with empty or whitespace-only header names.
// Flags columns where the header is null, empty string, or contains only whitespace after trimming.
func DetectEmptyHeaders(data *upload.SheetData) []Issue {
	var emptyPositions []string
	var emptyColIndices []int
	count := 0

	for col := 0; col < data.ColCount; col++ {
		header := ""
		if col < len(data.Headers) {
			header = data.Headers[col]
		}
		if strings.TrimSpace(header) == "" {
			count++
			emptyPositions = append(emptyPositions, fmt.Sprintf("第%d欄", col+1))
			emptyColIndices = append(emptyColIndices, col)
		}
	}

	if count == 0 {
		return nil
	}

	desc := fmt.Sprintf("共 %d 個欄位標題為空白：%s。AI 無法判斷該欄位代表的意義", count, strings.Join(emptyPositions, "、"))

	// Build example showing the header row with empty headers highlighted
	// Show all column headers as a "row" to visualize the issue
	maxDisplayCols := 8
	displayCols := make([]int, 0, maxDisplayCols)
	// Include empty columns and their neighbors for context
	for _, emptyCol := range emptyColIndices {
		if len(displayCols) >= maxDisplayCols {
			break
		}
		// Add the column before (if exists and not already added)
		if emptyCol > 0 && (len(displayCols) == 0 || displayCols[len(displayCols)-1] != emptyCol-1) {
			displayCols = append(displayCols, emptyCol-1)
		}
		if len(displayCols) >= maxDisplayCols {
			break
		}
		displayCols = append(displayCols, emptyCol)
		// Add column after (if exists)
		if emptyCol+1 < data.ColCount && len(displayCols) < maxDisplayCols {
			displayCols = append(displayCols, emptyCol+1)
		}
	}

	// Ensure at least 6 columns are shown for better context
	for col := 0; col < data.ColCount && len(displayCols) < 6; col++ {
		alreadyHas := false
		for _, dc := range displayCols {
			if dc == col {
				alreadyHas = true
				break
			}
		}
		if !alreadyHas {
			displayCols = append(displayCols, col)
		}
	}
	// Sort displayCols
	sort.Ints(displayCols)

	// Build header row as cells
	headerCells := make([]string, len(displayCols))
	// Use actual header values as the table headers
	headerLabels := make([]string, len(displayCols))
	var highlights []int
	for i, col := range displayCols {
		if col < len(data.Headers) {
			h := strings.TrimSpace(data.Headers[col])
			if h == "" {
				headerLabels[i] = "(空白)"
				headerCells[i] = "(空白)"
				highlights = append(highlights, i)
			} else {
				headerLabels[i] = h
				headerCells[i] = h
			}
		} else {
			headerLabels[i] = "(空白)"
			headerCells[i] = "(空白)"
			highlights = append(highlights, i)
		}
	}

	examples := []IssueExample{
		{
			Headers:    headerLabels,
			RowNumber:  1, // header row is row 1
			Cells:      headerCells,
			Highlights: highlights,
		},
	}

	// Also show first 2 data rows for context
	for ri := 0; ri < 2 && ri < len(data.Rows); ri++ {
		rowCells := make([]string, len(displayCols))
		for i, col := range displayCols {
			if col < len(data.Rows[ri]) && !data.Rows[ri][col].IsEmpty {
				val := data.Rows[ri][col].Raw
				runes := []rune(val)
				if len(runes) > 15 {
					val = string(runes[:15]) + "…"
				}
				rowCells[i] = val
			}
		}
		examples = append(examples, IssueExample{
			Headers:    headerLabels,
			RowNumber:  data.HeaderRowIndex + 2 + ri,
			Cells:      rowCells,
			Highlights: nil,
		})
	}

	return []Issue{
		{
			Title:        "空白標題欄",
			Severity:     "Medium",
			AffectedRows: count,
			Unit:         "欄",
			Description:  desc,
			Examples:     examples,
			Indicator:    "empty_header",
		},
	}
}

// alphanumericIdentifierPattern matches code/ID-like values: letters, digits, dashes, underscores.
// Does NOT match values containing parenthesized content.
var alphanumericIdentifierPattern = regexp.MustCompile(`^[A-Za-z0-9\-_]+[A-Za-z0-9\-_.]*$`)

// DetectInlineRemarks detects cells in structured columns that contain parenthesized remarks.
// A structured column is one where >60% of non-empty cells match an alphanumeric identifier pattern.
// Flagged cells contain half-width () or full-width （） where the inner text has Chinese chars or length > 5.
// Structural parentheses (single char codes, version numbers, purely numeric) are excluded.
func DetectInlineRemarks(data *upload.SheetData) []Issue {
	if len(data.Rows) == 0 {
		return nil
	}

	type flaggedCell struct {
		rowIdx int
		colIdx int
	}

	var flaggedCells []flaggedCell

	for col := 0; col < data.ColCount; col++ {
		if !isStructuredColumn(data, col) {
			continue
		}

		// Check each cell in this structured column for inline remarks
		for rowIdx, row := range data.Rows {
			if col >= len(row) || row[col].IsEmpty {
				continue
			}
			val := row[col].Raw
			if hasInlineRemark(val) {
				flaggedCells = append(flaggedCells, flaggedCell{rowIdx: rowIdx, colIdx: col})
			}
		}
	}

	if len(flaggedCells) == 0 {
		return nil
	}

	// Build examples
	var problemCols []int
	colSet := make(map[int]bool)
	for _, fc := range flaggedCells {
		if !colSet[fc.colIdx] {
			colSet[fc.colIdx] = true
			problemCols = append(problemCols, fc.colIdx)
		}
	}

	displayCols := selectDisplayColumns(data, problemCols)
	headers := getDisplayHeaders(data, displayCols)

	var examples []IssueExample
	for _, fc := range flaggedCells {
		if len(examples) >= maxExamples {
			break
		}
		row := data.Rows[fc.rowIdx]
		cells := getRowCells(row, displayCols)
		var highlights []int
		for i, col := range displayCols {
			if col == fc.colIdx {
				highlights = append(highlights, i)
				break
			}
		}
		examples = append(examples, IssueExample{
			Headers:    headers,
			RowNumber:  fc.rowIdx + data.HeaderRowIndex + 2,
			Cells:      cells,
			Highlights: highlights,
		})
	}

	return []Issue{
		{
			Title:        "行內備註混入資料欄",
			Severity:     "Medium",
			AffectedRows: len(flaggedCells),
			Unit:         "處",
			Description:  fmt.Sprintf("偵測到 %d 個儲存格在結構化欄位中混入括號備註，建議分離備註以利 AI 比對", len(flaggedCells)),
			Examples:     examples,
			Indicator:    "inline_remark",
		},
	}
}

// isStructuredColumn checks if >60% of non-empty cells in the column match
// an alphanumeric identifier pattern (letters, digits, dashes, underscores —
// code/ID-like values without parenthesized content).
func isStructuredColumn(data *upload.SheetData, col int) bool {
	nonEmpty := 0
	matchCount := 0

	for _, row := range data.Rows {
		if col >= len(row) || row[col].IsEmpty {
			continue
		}
		nonEmpty++
		val := strings.TrimSpace(row[col].Raw)
		if val == "" {
			continue
		}
		// Check if value matches the identifier pattern (no parenthesized content)
		if alphanumericIdentifierPattern.MatchString(val) {
			matchCount++
		}
	}

	if nonEmpty == 0 {
		return false
	}
	return float64(matchCount)/float64(nonEmpty) > 0.6
}

// hasInlineRemark checks if a cell value contains parenthesized content
// that qualifies as an inline remark (contains Chinese chars or length > 5),
// excluding structural parentheses (single char codes, version numbers, purely numeric).
func hasInlineRemark(val string) bool {
	// Search for parenthesized content: half-width () or full-width （）
	// Check all occurrences of parenthesized content
	remaining := val
	for {
		openIdx := strings.IndexAny(remaining, "(（")
		if openIdx < 0 {
			break
		}

		// Determine which type of bracket was found
		var closeChars string
		r := []rune(remaining[openIdx:])[0]
		if r == '(' {
			closeChars = ")"
		} else {
			closeChars = "）"
		}

		inner := remaining[openIdx+len(string(r)):]
		closeIdx := strings.Index(inner, closeChars)
		if closeIdx < 0 {
			break
		}

		innerText := inner[:closeIdx]

		// Check if this qualifies as an inline remark (not structural)
		if isInlineRemarkContent(innerText) {
			return true
		}

		// Move past this parenthesized section
		remaining = inner[closeIdx+len(closeChars):]
	}
	return false
}

// isInlineRemarkContent determines if parenthesized content is a remark vs structural.
// Returns true if the content should be flagged as a remark.
// Structural patterns (NOT flagged):
//   - Single character codes like "A", "B" (length == 1)
//   - Version numbers like "v2", "V3"
//   - Purely numeric content like "123", "45"
//
// Remark patterns (flagged):
//   - Contains Chinese characters
//   - Length > 5 (regardless of content)
func isInlineRemarkContent(inner string) bool {
	trimmed := strings.TrimSpace(inner)
	if trimmed == "" {
		return false
	}

	runes := []rune(trimmed)

	// Check for Chinese characters — always flag
	for _, r := range runes {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}

	// Structural exclusions (not flagged even if length > 5):
	// Single character: "(A)", "(B)"
	if len(runes) == 1 {
		return false
	}

	// Version numbers: "(v2)", "(V3)", "(v12)"
	if len(runes) >= 2 && (runes[0] == 'v' || runes[0] == 'V') {
		allDigits := true
		for _, r := range runes[1:] {
			if r < '0' || r > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			return false
		}
	}

	// Purely numeric: "(123)", "(45)"
	allNumeric := true
	for _, r := range runes {
		if (r < '0' || r > '9') && r != '.' && r != ',' {
			allNumeric = false
			break
		}
	}
	if allNumeric {
		return false
	}

	// Length > 5 — flag it
	if len(runes) > 5 {
		return true
	}

	return false
}

// DetectOrphanTotalRows detects isolated numeric rows at the bottom of data
// that appear after 2+ consecutive empty rows following the main data block.
// Detection conditions:
//   (a) the row appears after 2 or more consecutive empty rows following the main data block
//   (b) the row contains at most 2 non-empty cells
//   (c) at least one non-empty cell is numeric (parseable after removing thousands separators and decimal points)
//
// "Main data block" = the last row before the first empty row gap of 2+ consecutive empty rows.
// This detection is independent of existing subtotal keyword detection.
func DetectOrphanTotalRows(data *upload.SheetData) []Issue {
	if len(data.Rows) == 0 {
		return nil
	}

	// Find the end of the main data block: the first occurrence of 2+ consecutive empty rows.
	mainBlockEnd := -1 // index of the last data row before the gap
	consecutiveEmpty := 0
	for rowIdx, row := range data.Rows {
		if isRowEmpty(row, data.ColCount) {
			if consecutiveEmpty == 0 && rowIdx > 0 {
				// Mark potential end of main data block
				mainBlockEnd = rowIdx - 1
			}
			consecutiveEmpty++
		} else {
			if consecutiveEmpty >= 2 {
				// We found the gap — mainBlockEnd is already set
				break
			}
			// Reset: no gap found yet
			consecutiveEmpty = 0
			mainBlockEnd = -1
		}
	}

	// If we never found a gap of 2+ empty rows, no orphan totals
	if consecutiveEmpty < 2 && mainBlockEnd == -1 {
		return nil
	}
	// If mainBlockEnd is still -1 (e.g., file starts with empty rows), set it to -1 meaning no data block
	if mainBlockEnd < 0 {
		mainBlockEnd = 0
	}

	// Find the start of the post-gap area: skip past the first gap of 2+ empty rows
	postGapStart := -1
	consecutiveEmpty = 0
	for rowIdx, row := range data.Rows {
		if isRowEmpty(row, data.ColCount) {
			consecutiveEmpty++
		} else {
			if consecutiveEmpty >= 2 && rowIdx > mainBlockEnd {
				postGapStart = rowIdx
				break
			}
			consecutiveEmpty = 0
		}
	}

	if postGapStart < 0 {
		return nil
	}

	// Scan rows from postGapStart onward for orphan total candidates
	var orphanRowNumbers []int
	var orphanExamples []IssueExample

	for rowIdx := postGapStart; rowIdx < len(data.Rows); rowIdx++ {
		row := data.Rows[rowIdx]

		// Skip empty rows in the post-gap area
		if isRowEmpty(row, data.ColCount) {
			continue
		}

		// Condition (b): at most 2 non-empty cells
		nonEmptyCells := 0
		for col := 0; col < data.ColCount && col < len(row); col++ {
			if !row[col].IsEmpty {
				nonEmptyCells++
			}
		}
		if nonEmptyCells > 2 {
			continue
		}

		// Condition (c): at least one non-empty cell is numeric
		hasNumeric := false
		for col := 0; col < data.ColCount && col < len(row); col++ {
			if col < len(row) && !row[col].IsEmpty {
				if isOrphanNumeric(row[col].Raw) {
					hasNumeric = true
					break
				}
			}
		}
		if !hasNumeric {
			continue
		}

		// This row qualifies as an orphan total row
		excelRowNumber := rowIdx + data.HeaderRowIndex + 2 // 1-based Excel row
		orphanRowNumbers = append(orphanRowNumbers, excelRowNumber)

		// Build example
		if len(orphanExamples) < maxExamples {
			var allCols []int
			limit := data.ColCount
			if limit > 6 {
				limit = 6
			}
			for i := 0; i < limit; i++ {
				allCols = append(allCols, i)
			}
			headers := getDisplayHeaders(data, allCols)
			cells := getRowCells(row, allCols)

			// Highlight non-empty cells
			var highlights []int
			for i, col := range allCols {
				if col < len(row) && !row[col].IsEmpty {
					highlights = append(highlights, i)
				}
			}

			orphanExamples = append(orphanExamples, IssueExample{
				Label:      "孤立合計列",
				Headers:    headers,
				RowNumber:  excelRowNumber,
				Cells:      cells,
				Highlights: highlights,
			})
		}
	}

	if len(orphanRowNumbers) == 0 {
		return nil
	}

	// Build row number description
	var rowDesc string
	if len(orphanRowNumbers) == 1 {
		rowDesc = fmt.Sprintf("第 %d 列", orphanRowNumbers[0])
	} else {
		parts := make([]string, len(orphanRowNumbers))
		for i, rn := range orphanRowNumbers {
			parts[i] = fmt.Sprintf("%d", rn)
		}
		rowDesc = fmt.Sprintf("第 %s 列", strings.Join(parts, "、"))
	}

	return []Issue{{
		Title:        "表格結構問題",
		Severity:     "Medium",
		AffectedRows: len(orphanRowNumbers),
		Unit:         "處",
		Description:  fmt.Sprintf("偵測到孤立合計列：%s（位於資料區塊後方的獨立數值列）", rowDesc),
		Examples:     limitExamples(orphanExamples),
		Indicator:    "table_structure",
	}}
}

// isOrphanNumeric checks if a value is numeric for orphan total detection.
// It removes thousands separators (commas) and decimal points before parsing.
func isOrphanNumeric(s string) bool {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return false
	}
	// Remove thousands separators (commas)
	cleaned := strings.ReplaceAll(trimmed, ",", "")
	// Try parsing directly (handles decimals)
	_, err := strconv.ParseFloat(cleaned, 64)
	return err == nil
}

// DetectColumnTypeMismatch detects columns where the majority of values are numeric
// but some cells contain non-numeric text. A column is inferred as "numeric" type when
// more than 70% of its non-empty cells are numeric (after trimming whitespace, removing
// currency symbols NT$, USD, $, ¥, €, removing thousands commas, and attempting ParseFloat).
// Empty cells are never flagged. Severity is "High" if >10% of non-empty cells are mismatched,
// "Medium" otherwise. Provides up to 5 examples.
func DetectColumnTypeMismatch(data *upload.SheetData) []Issue {
	if len(data.Rows) == 0 {
		return nil
	}

	type flaggedCell struct {
		rowIdx int
		colIdx int
	}

	var allFlagged []flaggedCell
	totalMismatch := 0

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

		// Column is "numeric" type if >70% of non-empty cells are numeric
		if float64(numericCount)/float64(nonEmptyCount) <= 0.70 {
			continue
		}

		// Flag all non-numeric, non-empty cells in this numeric column
		for rowIdx, row := range data.Rows {
			if col >= len(row) || row[col].IsEmpty {
				continue
			}
			if !isCellNumericForTypeMismatch(row[col].Raw) {
				allFlagged = append(allFlagged, flaggedCell{rowIdx: rowIdx, colIdx: col})
				totalMismatch++
			}
		}
	}

	if totalMismatch == 0 {
		return nil
	}

	// Determine severity: calculate mismatch percentage across all numeric columns
	// Use per-column logic: if any column has >10% mismatch → "High"
	severity := "Medium"
	for col := 0; col < data.ColCount; col++ {
		numericCount := 0
		nonEmptyCount := 0
		mismatchCount := 0

		for _, row := range data.Rows {
			if col >= len(row) || row[col].IsEmpty {
				continue
			}
			nonEmptyCount++
			if isCellNumericForTypeMismatch(row[col].Raw) {
				numericCount++
			} else {
				mismatchCount++
			}
		}

		if nonEmptyCount == 0 {
			continue
		}
		// Only consider numeric columns
		if float64(numericCount)/float64(nonEmptyCount) <= 0.70 {
			continue
		}
		// Check mismatch ratio in this column
		if nonEmptyCount > 0 && float64(mismatchCount)/float64(nonEmptyCount) > 0.10 {
			severity = "High"
			break
		}
	}

	// Determine affected rows (unique row indices)
	affectedRowSet := make(map[int]bool)
	for _, f := range allFlagged {
		affectedRowSet[f.rowIdx] = true
	}

	// Determine affected columns for examples
	colSet := make(map[int]bool)
	var problemCols []int
	for _, f := range allFlagged {
		if !colSet[f.colIdx] {
			colSet[f.colIdx] = true
			problemCols = append(problemCols, f.colIdx)
		}
	}

	// Build description
	var colNames []string
	for _, col := range problemCols {
		colNames = append(colNames, getColumnName(data, col))
	}
	desc := "以下數值欄位中出現了非數字的值，AI 無法正確進行計算：\n" + strings.Join(colNames, "\n")

	// Build examples (up to 8)
	displayCols := selectDisplayColumns(data, problemCols)
	headers := getDisplayHeaders(data, displayCols)

	// Group flagged cells by row
	rowToFlaggedCols := make(map[int][]int)
	var rowOrder []int
	for _, f := range allFlagged {
		if _, exists := rowToFlaggedCols[f.rowIdx]; !exists {
			rowOrder = append(rowOrder, f.rowIdx)
		}
		rowToFlaggedCols[f.rowIdx] = append(rowToFlaggedCols[f.rowIdx], f.colIdx)
	}

	// Sort rowOrder by row index ascending for consistent display
	sort.Slice(rowOrder, func(i, j int) bool { return rowOrder[i] < rowOrder[j] })

	typeMismatchMaxExamples := 8
	var examples []IssueExample
	for _, rowIdx := range rowOrder {
		if len(examples) >= typeMismatchMaxExamples {
			break
		}
		row := data.Rows[rowIdx]
		cells := getRowCells(row, displayCols)

		// Highlight all flagged cells in this row
		flaggedColsInRow := rowToFlaggedCols[rowIdx]
		var highlights []int
		for i, col := range displayCols {
			for _, fc := range flaggedColsInRow {
				if col == fc {
					highlights = append(highlights, i)
					break
				}
			}
		}

		examples = append(examples, IssueExample{
			Headers:    headers,
			RowNumber:  rowIdx + data.HeaderRowIndex + 2,
			Cells:      cells,
			Highlights: highlights,
		})
	}

	return []Issue{
		{
			Title:        "欄位型別不一致",
			Severity:     severity,
			AffectedRows: len(affectedRowSet),
			Unit:         "列受影響",
			Description:  desc,
			Examples:     examples,
			Indicator:    "column_type_mismatch",
		},
	}
}

// isCellNumericForTypeMismatch determines if a cell value is numeric using the
// column type inference logic: trim whitespace, remove currency symbols (NT$, USD, $, ¥, €),
// remove thousands commas, then attempt strconv.ParseFloat.
func isCellNumericForTypeMismatch(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return false
	}

	// Remove currency symbols
	cleaned := trimmed
	// Remove multi-char currency prefixes first (order matters: NT$ before $)
	for _, prefix := range []string{"NT$", "USD"} {
		if strings.HasPrefix(cleaned, prefix) {
			cleaned = strings.TrimSpace(cleaned[len(prefix):])
			break
		}
	}
	// Remove single-char currency symbols (prefix or suffix)
	cleaned = strings.TrimLeft(cleaned, "$¥€")
	cleaned = strings.TrimRight(cleaned, "$¥€")
	cleaned = strings.TrimSpace(cleaned)

	// Remove thousands commas
	cleaned = strings.ReplaceAll(cleaned, ",", "")

	if cleaned == "" {
		return false
	}

	_, err := strconv.ParseFloat(cleaned, 64)
	return err == nil
}





// DetectStrikethroughFormatting detects cells with strikethrough formatting.
// These may represent data that was marked for deletion but not actually removed.
func DetectStrikethroughFormatting(data *upload.SheetData) []Issue {
	if len(data.StrikethroughCells) == 0 {
		return nil
	}

	// Determine unique affected rows
	affectedRowSet := make(map[int]bool)
	for _, cl := range data.StrikethroughCells {
		affectedRowSet[cl.Row] = true
	}

	desc := fmt.Sprintf("有 %d 個儲存格使用刪除線標記，可能為待刪除但未移除的資料，建議確認是否應刪除或保留", len(data.StrikethroughCells))

	// Determine which columns have strikethrough cells for display
	var stCols []int
	stColSet := make(map[int]bool)
	for _, cl := range data.StrikethroughCells {
		if !stColSet[cl.Col] {
			stColSet[cl.Col] = true
			stCols = append(stCols, cl.Col)
		}
	}
	displayCols := selectDisplayColumns(data, stCols)
	headers := getDisplayHeaders(data, displayCols)

	// Group strikethrough cells by row
	rowToCols := make(map[int][]int)
	var rowOrder []int
	for _, cl := range data.StrikethroughCells {
		if _, exists := rowToCols[cl.Row]; !exists {
			rowOrder = append(rowOrder, cl.Row)
		}
		rowToCols[cl.Row] = append(rowToCols[cl.Row], cl.Col)
	}

	// Build examples (up to 5 rows)
	var examples []IssueExample
	for _, rowIdx := range rowOrder {
		if len(examples) >= maxExamples {
			break
		}
		if rowIdx < 0 || rowIdx >= len(data.Rows) {
			continue
		}
		row := data.Rows[rowIdx]
		cells := getRowCells(row, displayCols)

		// Highlight the strikethrough cell positions
		stColsInRow := rowToCols[rowIdx]
		var highlights []int
		for i, col := range displayCols {
			for _, sc := range stColsInRow {
				if col == sc {
					highlights = append(highlights, i)
					break
				}
			}
		}

		examples = append(examples, IssueExample{
			Headers:    headers,
			RowNumber:  rowIdx + data.HeaderRowIndex + 2,
			Cells:      cells,
			Highlights: highlights,
		})
	}

	return []Issue{
		{
			Title:        "儲存格含刪除線格式",
			Severity:     "Medium",
			AffectedRows: len(affectedRowSet),
			Unit:         "列受影響",
			Description:  desc,
			Examples:     limitExamples(examples),
			Indicator:    "strikethrough_formatting",
		},
	}
}
