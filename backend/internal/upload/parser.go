package upload

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// 解析相關的錯誤
var (
	ErrFileCorrupted    = errors.New("檔案已損壞或無法讀取")
	ErrTooManyRows      = errors.New("資料列數超過 100,000 列上限")
	ErrUnsupportedFormat = errors.New("不支援的檔案格式，僅支援 xlsx 和 csv")
)

// ParseXLSX 解析 xlsx 檔案，回傳 metadata
func ParseXLSX(filePath string) (*ParseResult, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, ErrFileCorrupted
	}
	defer f.Close()

	// 取得所有工作表名稱
	sheetNames := f.GetSheetList()
	if len(sheetNames) == 0 {
		return nil, ErrFileCorrupted
	}

	// 使用第一個工作表計算 row/col count
	firstSheet := sheetNames[0]
	rows, err := f.GetRows(firstSheet)
	if err != nil {
		return nil, ErrFileCorrupted
	}

	rowCount := len(rows)
	colCount := 0
	for _, row := range rows {
		if len(row) > colCount {
			colCount = len(row)
		}
	}

	// 偵測合併儲存格
	mergedCells := detectMergedCellsXLSX(f, firstSheet)

	return &ParseResult{
		SheetNames:  sheetNames,
		MergedCells: mergedCells,
		RowCount:    rowCount,
		ColCount:    colCount,
	}, nil
}

// ParseCSV 解析 CSV 檔案（支援 UTF-8 BOM），回傳 metadata
func ParseCSV(filePath string) (*ParseResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, ErrFileCorrupted
	}

	// 偵測並跳過 UTF-8 BOM (0xEF, 0xBB, 0xBF)
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		data = data[3:]
	}

	reader := csv.NewReader(bytes.NewReader(data))
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1 // 允許不一致的欄位數

	rowCount := 0
	colCount := 0
	maxRows := 100001 // 讀取上限 + 1 以檢測超限

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, ErrFileCorrupted
		}

		rowCount++
		if rowCount > maxRows {
			return nil, ErrTooManyRows
		}
		if len(record) > colCount {
			colCount = len(record)
		}
	}

	return &ParseResult{
		SheetNames:  []string{"Sheet1"}, // CSV 只有一個 "sheet"
		MergedCells: nil,
		RowCount:    rowCount,
		ColCount:    colCount,
	}, nil
}

// LoadSheetData 載入指定工作表的完整資料，供 Assessment Engine 使用
func LoadSheetData(filePath, sheetName, format string) (*SheetData, error) {
	switch strings.ToLower(format) {
	case "xlsx":
		return loadSheetDataXLSX(filePath, sheetName)
	case "csv":
		return loadSheetDataCSV(filePath)
	default:
		return nil, ErrUnsupportedFormat
	}
}

// loadSheetDataXLSX 從 xlsx 檔案載入指定工作表資料
func loadSheetDataXLSX(filePath, sheetName string) (*SheetData, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, ErrFileCorrupted
	}
	defer f.Close()

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("無法讀取工作表 %q: %w", sheetName, err)
	}

	// Fill down merged cells — merged ranges have values only in top-left cell
	fillDownMergedCells(f, sheetName, rows)

	if len(rows) == 0 {
		return &SheetData{
			Headers:        nil,
			Rows:           nil,
			MergedCells:    detectMergedCellsXLSX(f, sheetName),
			RowCount:       0,
			ColCount:       0,
			TotalSheetRows: 0,
			HeaderRowIndex: 0,
		}, nil
	}

	// 計算最大欄數
	colCount := 0
	for _, row := range rows {
		if len(row) > colCount {
			colCount = len(row)
		}
	}

	// 偵測隱藏欄位，建立可見欄位索引列表
	visibleCols := make([]int, 0, colCount)
	for col := 0; col < colCount; col++ {
		colName, _ := excelize.ColumnNumberToName(col + 1)
		visible, _ := f.GetColVisible(sheetName, colName)
		if visible {
			visibleCols = append(visibleCols, col)
		}
	}
	// 如果所有欄位都可見（或 API 回傳異常），使用全部欄位
	if len(visibleCols) == 0 {
		for col := 0; col < colCount; col++ {
			visibleCols = append(visibleCols, col)
		}
	}
	effectiveColCount := len(visibleCols)

	// 自動偵測表頭位置（掃描前 10 列，只用可見欄位）
	headerRowIdx := detectHeaderRow(rows, colCount)

	// 取得 headers — 只取可見欄位，使用 GetCellValue 逐格讀取
	headers := make([]string, effectiveColCount)
	for i, col := range visibleCols {
		cellName, _ := excelize.CoordinatesToCellName(col+1, headerRowIdx+1)
		val, _ := f.GetCellValue(sheetName, cellName)
		headers[i] = val
	}

	// 資料列（表頭之後的所有列，只取可見欄位）
	// 使用 GetCellValue 逐格讀取以正確處理日期格式（GetRows 不套用數字格式）
	var dataRows [][]CellValue
	for i := headerRowIdx + 1; i < len(rows); i++ {
		row := make([]CellValue, effectiveColCount)
		for j, col := range visibleCols {
			cellName, _ := excelize.CoordinatesToCellName(col+1, i+1)
			val, _ := f.GetCellValue(sheetName, cellName)
			row[j] = CellValue{
				Raw:     val,
				IsEmpty: isEmptyValue(val),
			}
		}
		dataRows = append(dataRows, row)
	}

	mergedCells := detectMergedCellsXLSX(f, sheetName)

	// Remove trailing empty rows (Excel "ghost rows" that have no actual data)
	dataRows = trimTrailingEmptyRows(dataRows, effectiveColCount)

	// Detect cells with comments (red triangle indicator)
	commentCells := detectCommentCells(f, sheetName, headerRowIdx, effectiveColCount, len(dataRows), visibleCols)

	// Detect cells with strikethrough formatting
	strikethroughCells := detectStrikethroughCells(f, sheetName, headerRowIdx, effectiveColCount, len(dataRows), visibleCols)

	// TotalSheetRows = header rows + data rows (after trimming)
	totalSheetRows := headerRowIdx + 1 + len(dataRows)

	return &SheetData{
		Headers:            headers,
		Rows:               dataRows,
		MergedCells:        mergedCells,
		RowCount:           len(dataRows),
		ColCount:           effectiveColCount,
		TotalSheetRows:     totalSheetRows,
		HeaderRowIndex:     headerRowIdx,
		RawFirstRows:       extractRawFirstRowsFormatted(f, sheetName, rows, colCount, 5),
		CommentCells:       commentCells,
		StrikethroughCells: strikethroughCells,
	}, nil
}

// detectCommentCells finds all cells that have comments (red triangle indicator)
func detectCommentCells(f *excelize.File, sheetName string, headerRowIdx, colCount, dataRowCount int, visibleCols []int) []CellLocation {
	var cells []CellLocation
	comments, err := f.GetComments(sheetName)
	if err != nil || len(comments) == 0 {
		return nil
	}
	for _, comment := range comments {
		// Parse comment cell reference (e.g. "A5")
		col, row, err := excelize.CellNameToCoordinates(comment.Cell)
		if err != nil {
			continue
		}
		// Convert 1-based to 0-based, then to data row index
		sheetRow0 := row - 1
		sheetCol0 := col - 1
		dataRowIdx := sheetRow0 - headerRowIdx - 1
		if dataRowIdx < 0 || dataRowIdx >= dataRowCount {
			continue
		}
		// Map sheet column to visible column index
		for visIdx, visCol := range visibleCols {
			if visCol == sheetCol0 {
				cells = append(cells, CellLocation{Row: dataRowIdx, Col: visIdx})
				break
			}
		}
	}
	return cells
}

// detectStrikethroughCells finds cells with strikethrough font formatting
func detectStrikethroughCells(f *excelize.File, sheetName string, headerRowIdx, colCount, dataRowCount int, visibleCols []int) []CellLocation {
	var cells []CellLocation
	// Sample first 100 data rows to avoid performance issues
	maxRows := dataRowCount
	if maxRows > 100 {
		maxRows = 100
	}
	for dataRowIdx := 0; dataRowIdx < maxRows; dataRowIdx++ {
		sheetRow := headerRowIdx + 1 + dataRowIdx + 1 // 1-based for excelize
		for visIdx, visCol := range visibleCols {
			cellName, _ := excelize.CoordinatesToCellName(visCol+1, sheetRow)
			styleID, err := f.GetCellStyle(sheetName, cellName)
			if err != nil || styleID == 0 {
				continue
			}
			style, err := f.GetStyle(styleID)
			if err != nil {
				continue
			}
			if style != nil && style.Font != nil && style.Font.Strike {
				cells = append(cells, CellLocation{Row: dataRowIdx, Col: visIdx})
			}
		}
	}
	return cells
}

// loadSheetDataCSV 從 CSV 檔案載入資料
func loadSheetDataCSV(filePath string) (*SheetData, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, ErrFileCorrupted
	}

	// 跳過 UTF-8 BOM
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		data = data[3:]
	}

	reader := csv.NewReader(bytes.NewReader(data))
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	records, err := reader.ReadAll()
	if err != nil {
		return nil, ErrFileCorrupted
	}

	if len(records) == 0 {
		return &SheetData{
			Headers:        nil,
			Rows:           nil,
			MergedCells:    nil,
			RowCount:       0,
			ColCount:       0,
			TotalSheetRows: 0,
			HeaderRowIndex: 0,
		}, nil
	}

	// 第一列為 headers
	headers := records[0]

	// 計算最大欄數
	colCount := len(headers)
	for _, record := range records {
		if len(record) > colCount {
			colCount = len(record)
		}
	}

	// 填充 headers 至最大欄數
	for len(headers) < colCount {
		headers = append(headers, "")
	}

	// 資料列
	var dataRows [][]CellValue
	for i := 1; i < len(records); i++ {
		row := make([]CellValue, colCount)
		for j := 0; j < colCount; j++ {
			if j < len(records[i]) {
				val := records[i][j]
				row[j] = CellValue{
					Raw:     val,
					IsEmpty: isEmptyValue(val),
				}
			} else {
				row[j] = CellValue{Raw: "", IsEmpty: true}
			}
		}
		dataRows = append(dataRows, row)
	}

	return &SheetData{
		Headers:        headers,
		Rows:           dataRows,
		MergedCells:    nil,
		RowCount:       len(dataRows),
		ColCount:       colCount,
		TotalSheetRows: len(records),
		HeaderRowIndex: 0,
	}, nil
}

// detectMergedCellsXLSX 偵測 xlsx 檔案中的合併儲存格
func detectMergedCellsXLSX(f *excelize.File, sheetName string) []MergedRange {
	mergedCells, err := f.GetMergeCells(sheetName)
	if err != nil {
		return nil
	}

	var ranges []MergedRange
	for _, mc := range mergedCells {
		startCell := mc.GetStartAxis()
		endCell := mc.GetEndAxis()

		startCol, startRow, err1 := excelize.CellNameToCoordinates(startCell)
		endCol, endRow, err2 := excelize.CellNameToCoordinates(endCell)
		if err1 != nil || err2 != nil {
			continue
		}

		ranges = append(ranges, MergedRange{
			StartRow: startRow,
			EndRow:   endRow,
			StartCol: startCol,
			EndCol:   endCol,
		})
	}
	return ranges
}

// isEmptyValue 判斷儲存格值是否為空（null、僅空白字元、或無值）
func isEmptyValue(val string) bool {
	return strings.TrimSpace(val) == ""
}

// trimTrailingEmptyRows removes trailing rows where all cells are empty.
// This handles Excel "ghost rows" that have formatting but no actual data.
func trimTrailingEmptyRows(rows [][]CellValue, colCount int) [][]CellValue {
	lastNonEmpty := len(rows) - 1
	for lastNonEmpty >= 0 {
		allEmpty := true
		for i := 0; i < colCount && i < len(rows[lastNonEmpty]); i++ {
			if !rows[lastNonEmpty][i].IsEmpty {
				allEmpty = false
				break
			}
		}
		if !allEmpty {
			break
		}
		lastNonEmpty--
	}
	return rows[:lastNonEmpty+1]
}

// parseMergedCellRange 解析合併儲存格範圍字串（如 "A1:C3"）
// 此函數用於內部輔助，將 Excel 欄位名稱轉為數字座標
func parseMergedCellRange(cellRange string) (startCol, startRow, endCol, endRow int, err error) {
	parts := strings.Split(cellRange, ":")
	if len(parts) != 2 {
		return 0, 0, 0, 0, fmt.Errorf("無效的範圍: %s", cellRange)
	}

	startCol, startRow, err = excelize.CellNameToCoordinates(parts[0])
	if err != nil {
		return 0, 0, 0, 0, err
	}

	endCol, endRow, err = excelize.CellNameToCoordinates(parts[1])
	if err != nil {
		return 0, 0, 0, 0, err
	}

	return startCol, startRow, endCol, endRow, nil
}

// getFileExtension 取得檔案副檔名（小寫、不含點）
func getFileExtension(filename string) string {
	parts := strings.Split(filename, ".")
	if len(parts) < 2 {
		return ""
	}
	return strings.ToLower(parts[len(parts)-1])
}

// formatFileSize 格式化檔案大小顯示
func formatFileSize(size int64) string {
	if size < 1024 {
		return strconv.FormatInt(size, 10) + " B"
	}
	if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
}

// detectHeaderRow scans the first N rows to find the most likely header row.
// A good header row has: all cells non-empty, all values are text (not numeric/date),
// no repeated values, and values are relatively short.
// Returns the 0-based index of the detected header row.
func detectHeaderRow(rows [][]string, colCount int) int {
	maxScan := 10
	if maxScan > len(rows) {
		maxScan = len(rows)
	}

	bestRow := 0
	bestScore := -1

	for i := 0; i < maxScan; i++ {
		row := rows[i]
		score := headerRowScore(row, colCount)
		if score > bestScore {
			bestScore = score
			bestRow = i
		}
	}

	return bestRow
}

// headerRowScore scores a row on how likely it is to be a header.
// Higher score = more likely to be a header row.
func headerRowScore(row []string, colCount int) int {
	if len(row) == 0 {
		return -1
	}

	score := 0
	nonEmptyCount := 0
	allText := true
	seen := make(map[string]bool)
	hasDuplicate := false
	totalLen := 0

	for i := 0; i < colCount && i < len(row); i++ {
		val := strings.TrimSpace(row[i])
		if val == "" {
			continue
		}
		nonEmptyCount++
		totalLen += len([]rune(val))

		// Check if value looks like text (not a number or date)
		if isNumericLike(val) || isDateLike(val) {
			allText = false
		}

		// Check for duplicates
		lower := strings.ToLower(val)
		if seen[lower] {
			hasDuplicate = true
		}
		seen[lower] = true
	}

	if nonEmptyCount == 0 {
		return -1
	}

	// Scoring criteria:
	// +30: high fill rate (>= 80% of columns filled)
	fillRate := float64(nonEmptyCount) / float64(colCount)
	if fillRate >= 0.8 {
		score += 30
	} else if fillRate >= 0.5 {
		score += 15
	}

	// +20: all values are text (not numbers/dates)
	if allText {
		score += 20
	}

	// +20: no duplicate values
	if !hasDuplicate {
		score += 20
	}

	// +15: average value length is short (typical for headers: 2-15 chars)
	avgLen := float64(totalLen) / float64(nonEmptyCount)
	if avgLen >= 2 && avgLen <= 20 {
		score += 15
	}

	// +10: values contain no newlines or very long strings
	hasLong := false
	for i := 0; i < colCount && i < len(row); i++ {
		if len([]rune(row[i])) > 50 || strings.Contains(row[i], "\n") {
			hasLong = true
			break
		}
	}
	if !hasLong {
		score += 10
	}

	return score
}

// isNumericLike checks if a string looks like a number
func isNumericLike(s string) bool {
	cleaned := strings.ReplaceAll(strings.TrimSpace(s), ",", "")
	if cleaned == "" {
		return false
	}
	_, err := strconv.ParseFloat(cleaned, 64)
	return err == nil
}

// isDateLike checks if a string looks like a date
func isDateLike(s string) bool {
	s = strings.TrimSpace(s)
	// Common date patterns
	datePatterns := []string{
		`^\d{4}[/-]\d{1,2}[/-]\d{1,2}$`,
		`^\d{2,3}\.\d{1,2}\.\d{1,2}$`,
		`^\d{1,2}[/-]\d{1,2}[/-]\d{2,4}$`,
	}
	for _, pattern := range datePatterns {
		if matched, _ := regexp.MatchString(pattern, s); matched {
			return true
		}
	}
	return false
}

// extractRawFirstRows extracts the first N rows as raw string arrays for debugging.
func extractRawFirstRows(rows [][]string, colCount int, n int) [][]string {
	if n > len(rows) {
		n = len(rows)
	}
	result := make([][]string, n)
	for i := 0; i < n; i++ {
		row := make([]string, colCount)
		for j := 0; j < colCount && j < len(rows[i]); j++ {
			row[j] = rows[i][j]
		}
		result[i] = row
	}
	return result
}

// extractRawFirstRowsFormatted extracts the first N rows using GetCellValue for correct date formatting.
func extractRawFirstRowsFormatted(f *excelize.File, sheetName string, rows [][]string, colCount int, n int) [][]string {
	if n > len(rows) {
		n = len(rows)
	}
	result := make([][]string, n)
	for i := 0; i < n; i++ {
		row := make([]string, colCount)
		for j := 0; j < colCount; j++ {
			cellName, _ := excelize.CoordinatesToCellName(j+1, i+1)
			val, _ := f.GetCellValue(sheetName, cellName)
			row[j] = val
		}
		result[i] = row
	}
	return result
}

// fillDownMergedCells fills empty cells within merged ranges with the top-left cell's value.
// Merged cells in Excel only store the value in the top-left cell; other cells in the range
// appear empty when read via GetRows. This function restores the values for downstream processing.
func fillDownMergedCells(f *excelize.File, sheetName string, rows [][]string) {
	mergedCells, err := f.GetMergeCells(sheetName)
	if err != nil || len(mergedCells) == 0 {
		return
	}

	for _, mc := range mergedCells {
		// Get the value of the merged cell (stored in start cell)
		value := mc.GetCellValue()
		if value == "" {
			continue
		}

		// Parse the range
		startCell := mc.GetStartAxis()
		endCell := mc.GetEndAxis()
		startCol, startRow, err1 := excelize.CellNameToCoordinates(startCell)
		endCol, endRow, err2 := excelize.CellNameToCoordinates(endCell)
		if err1 != nil || err2 != nil {
			continue
		}

		// Fill all cells in the range (coordinates are 1-based, rows slice is 0-based)
		for row := startRow; row <= endRow; row++ {
			rowIdx := row - 1 // convert to 0-based
			if rowIdx < 0 || rowIdx >= len(rows) {
				continue
			}
			for col := startCol; col <= endCol; col++ {
				colIdx := col - 1 // convert to 0-based
				// Extend row if needed
				for len(rows[rowIdx]) <= colIdx {
					rows[rowIdx] = append(rows[rowIdx], "")
				}
				// Fill if empty
				if rows[rowIdx][colIdx] == "" {
					rows[rowIdx][colIdx] = value
				}
			}
		}
	}
}
