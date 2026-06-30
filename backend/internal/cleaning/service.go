package cleaning

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/safe-ai/excel-brushing-tool/internal/assessment"
	"github.com/safe-ai/excel-brushing-tool/internal/upload"
	"github.com/safe-ai/excel-brushing-tool/pkg/config"
)

// Service handles cleaning business logic
type Service struct {
	repo         *Repository
	assessRepo   *assessment.Repository
	settingsRepo *assessment.SettingsRepository
	uploadRepo   *upload.Repository
	cfg          *config.Config
}

// NewService creates a new cleaning Service
func NewService(
	repo *Repository,
	assessRepo *assessment.Repository,
	settingsRepo *assessment.SettingsRepository,
	uploadRepo *upload.Repository,
	cfg *config.Config,
) *Service {
	return &Service{
		repo:         repo,
		assessRepo:   assessRepo,
		settingsRepo: settingsRepo,
		uploadRepo:   uploadRepo,
		cfg:          cfg,
	}
}

// ApplyRules executes cleaning rules and row operations on the assessment data
func (s *Service) ApplyRules(ctx context.Context, userID uuid.UUID, req CleanRequest) (*CleaningSession, error) {
	// 1. Get assessment record
	assess, err := s.assessRepo.GetByID(ctx, req.AssessmentID)
	if err != nil {
		return nil, fmt.Errorf("無法取得評估記錄: %w", err)
	}

	// 2. Get upload record → load SheetData
	up, err := s.uploadRepo.GetByID(ctx, assess.UploadID)
	if err != nil {
		return nil, fmt.Errorf("無法取得上傳記錄: %w", err)
	}

	ext := strings.TrimPrefix(filepath.Ext(up.Filename), ".")
	ext = strings.ToLower(ext)

	sheetName := ""
	if up.SelectedSheet != nil {
		sheetName = *up.SelectedSheet
	} else if len(up.SheetNames) > 0 {
		sheetName = up.SheetNames[0]
	}

	data, err := upload.LoadSheetData(up.FilePath, sheetName, ext)
	if err != nil {
		return nil, fmt.Errorf("無法載入工作表資料: %w", err)
	}

	rowsBefore := data.RowCount
	scoreBefore := assess.TotalScore
	operatorID := userID.String()

	// 3. Apply batch rules in a fixed logical order:
	// First: structural removals (multi-table, subtotal) before row-level operations
	var cleaningLog []LogEntry

	// Phase 1: Multi-table block removal (must be before empty_row_remove)
	for _, rule := range req.Rules {
		if rule == "multi_table_keep_main" {
			if req.KeepBlockIndex >= 0 {
				MultiTableKeepSpecific(data, req.KeepBlockIndex, &cleaningLog, operatorID)
			} else {
				MultiTableKeepMain(data, &cleaningLog, operatorID)
			}
		}
	}

	// Phase 2: All other rules
	for _, rule := range req.Rules {
		switch rule {
		case "date_normalize":
			DateNormalize(data, &cleaningLog, operatorID)
		case "dedup":
			Dedup(data, &cleaningLog, operatorID)
		case "name_normalize":
			NameNormalize(data, &cleaningLog, operatorID)
		case "subtotal_remove":
			SubtotalRemove(data, &cleaningLog, operatorID)
		case "newline_remove":
			NewlineRemove(data, &cleaningLog, operatorID)
		case "bracket_note_remove":
			BracketNoteRemove(data, &cleaningLog, operatorID)
		case "empty_row_remove":
			EmptyRowRemove(data, &cleaningLog, operatorID)
		case "empty_col_remove":
			if len(req.RemoveColumns) > 0 {
				EmptyColRemoveSpecific(data, req.RemoveColumns, &cleaningLog, operatorID)
			} else {
				EmptyColRemove(data, &cleaningLog, operatorID)
			}
		// multi_table_keep_main already handled in Phase 1
		}
	}

	// 4. Apply row operations (sort by descending index for delete to avoid index shifting)
	if len(req.RowOps) > 0 {
		// Separate fill_na and delete operations
		var fillOps, deleteOps []RowOp
		for _, op := range req.RowOps {
			switch op.Action {
			case "fill_na":
				fillOps = append(fillOps, op)
			case "delete":
				deleteOps = append(deleteOps, op)
			}
		}

		// Apply fill_na first (doesn't change indices)
		for _, op := range fillOps {
			if err := FillNA(data, op.RowIndex, &cleaningLog, operatorID); err != nil {
				// Log the error but continue with other operations
				continue
			}
		}

		// Apply deletes in descending order to preserve indices
		sort.Slice(deleteOps, func(i, j int) bool {
			return deleteOps[i].RowIndex > deleteOps[j].RowIndex
		})
		for _, op := range deleteOps {
			if err := DeleteRow(data, op.RowIndex, &cleaningLog, operatorID); err != nil {
				continue
			}
		}
	}

	// 5. Re-run assessment on cleaned data → score_after
	weights, err := s.settingsRepo.GetWeights(ctx)
	if err != nil {
		return nil, fmt.Errorf("無法取得權重設定: %w", err)
	}

	rowComp := assessment.CalculateRowCompleteness(data)
	colComp, _ := assessment.CalculateColumnCompleteness(data)
	formatCon := assessment.CalculateFormatConsistency(data)
	dupSim := assessment.CalculateDuplicateSimilar(data)
	tableStr := assessment.CalculateTableStructure(data)
	aiReady := assessment.CalculateAIQueryReadiness(data)

	indicators := assessment.IndicatorScores{
		RowCompleteness:    rowComp,
		ColumnCompleteness: colComp,
		FormatConsistency:  formatCon,
		DuplicateSimilar:   dupSim,
		TableStructure:     tableStr,
		AIQueryReadiness:   aiReady,
	}

	scoreAfter, _, err := assessment.CalculateTotalScore(indicators, weights)
	if err != nil {
		return nil, fmt.Errorf("無法計算清理後分數: %w", err)
	}

	// 6. Save refined file to volume
	sessionID := uuid.New()
	refinedPath := s.generateRefinedPath(sessionID)

	if err := s.saveRefinedData(data, refinedPath); err != nil {
		return nil, fmt.Errorf("無法儲存清理後檔案: %w", err)
	}

	// 7. Build and save session
	session := &CleaningSession{
		ID:              sessionID,
		AssessmentID:    req.AssessmentID,
		UserID:          userID,
		RulesApplied:    req.Rules,
		RowsBefore:      rowsBefore,
		RowsAfter:       data.RowCount,
		ScoreBefore:     scoreBefore,
		ScoreAfter:      scoreAfter,
		CleaningLog:     cleaningLog,
		RefinedFilePath: refinedPath,
	}

	if err := s.repo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("無法儲存梳理記錄: %w", err)
	}

	return session, nil
}

// PreviewRemovals analyzes the assessment data and returns items that could be removed.
// This allows the user to select which columns/blocks to actually remove.
func (s *Service) PreviewRemovals(ctx context.Context, assessmentID uuid.UUID) (*RemovalPreview, error) {
	// 1. Get assessment record
	assess, err := s.assessRepo.GetByID(ctx, assessmentID)
	if err != nil {
		return nil, fmt.Errorf("無法取得評估記錄: %w", err)
	}

	// 2. Get upload record → load SheetData
	up, err := s.uploadRepo.GetByID(ctx, assess.UploadID)
	if err != nil {
		return nil, fmt.Errorf("無法取得上傳記錄: %w", err)
	}

	ext := strings.TrimPrefix(filepath.Ext(up.Filename), ".")
	ext = strings.ToLower(ext)

	sheetName := ""
	if up.SelectedSheet != nil {
		sheetName = *up.SelectedSheet
	} else if len(up.SheetNames) > 0 {
		sheetName = up.SheetNames[0]
	}

	data, err := upload.LoadSheetData(up.FilePath, sheetName, ext)
	if err != nil {
		return nil, fmt.Errorf("無法載入工作表資料: %w", err)
	}

	result := &RemovalPreview{
		AllColumns:   []ColumnRemovalItem{},
		EmptyColumns: []ColumnRemovalItem{},
		DataBlocks:   []DataBlockItem{},
		SampleRows:   []SampleRow{},
		Headers:      []string{},
	}

	// Build headers list
	headers := make([]string, data.ColCount)
	for i := 0; i < data.ColCount; i++ {
		if i < len(data.Headers) {
			h := strings.ReplaceAll(data.Headers[i], "\n", " ")
			headers[i] = h
		}
	}
	result.Headers = headers

	// Build sample rows (first 15 data rows)
	maxSample := 15
	if maxSample > len(data.Rows) {
		maxSample = len(data.Rows)
	}
	for i := 0; i < maxSample; i++ {
		cells := make([]string, data.ColCount)
		for col := 0; col < data.ColCount; col++ {
			if col < len(data.Rows[i]) && !data.Rows[i][col].IsEmpty {
				val := data.Rows[i][col].Raw
				val = strings.ReplaceAll(val, "\n", " ")
				runes := []rune(val)
				if len(runes) > 20 {
					val = string(runes[:20]) + "…"
				}
				cells[col] = val
			}
		}
		result.SampleRows = append(result.SampleRows, SampleRow{
			RowNumber: i + data.HeaderRowIndex + 2, // 1-based Excel row
			Cells:     cells,
		})
	}

	// 3. Build ALL columns list and find columns with >80% empty
	threshold := 0.8
	for col := 0; col < data.ColCount; col++ {
		emptyCount := 0
		for _, row := range data.Rows {
			if col >= len(row) || row[col].IsEmpty {
				emptyCount++
			}
		}
		emptyRate := float64(emptyCount) / float64(len(data.Rows))
		colName := getColName(data, col)

		result.AllColumns = append(result.AllColumns, ColumnRemovalItem{
			ColIndex:  col,
			ColName:   colName,
			EmptyRate: emptyRate,
		})

		if emptyRate > threshold {
			result.EmptyColumns = append(result.EmptyColumns, ColumnRemovalItem{
				ColIndex:  col,
				ColName:   colName,
				EmptyRate: emptyRate,
			})
		}
	}

	// 4. Find data blocks separated by ≥2 consecutive empty rows
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

	// Only include data blocks if there are multiple
	if len(blocks) > 1 {
		largestIdx := 0
		for i, b := range blocks {
			if b.size > blocks[largestIdx].size {
				largestIdx = i
			}
		}

		for i, b := range blocks {
			// Build preview from first row of block
			preview := ""
			if b.startIdx < len(data.Rows) {
				row := data.Rows[b.startIdx]
				var parts []string
				for col := 0; col < data.ColCount && col < len(row); col++ {
					if !row[col].IsEmpty {
						parts = append(parts, row[col].Raw)
					}
					if len(parts) >= 3 {
						break
					}
				}
				preview = strings.Join(parts, " | ")
				if len(preview) > 60 {
					preview = preview[:57] + "..."
				}
			}

			// Build sample rows for this block (first 3 rows)
			blockSampleMax := 10
			if b.size < blockSampleMax {
				blockSampleMax = b.size
			}
			var blockSampleRows []SampleRow
			for j := 0; j < blockSampleMax; j++ {
				rowIdx := b.startIdx + j
				if rowIdx >= len(data.Rows) {
					break
				}
				cells := make([]string, data.ColCount)
				for col := 0; col < data.ColCount; col++ {
					if col < len(data.Rows[rowIdx]) && !data.Rows[rowIdx][col].IsEmpty {
						val := data.Rows[rowIdx][col].Raw
						val = strings.ReplaceAll(val, "\n", " ")
						runes := []rune(val)
						if len(runes) > 20 {
							val = string(runes[:20]) + "…"
						}
						cells[col] = val
					}
				}
				blockSampleRows = append(blockSampleRows, SampleRow{
					RowNumber: rowIdx + data.HeaderRowIndex + 2,
					Cells:     cells,
				})
			}

			result.DataBlocks = append(result.DataBlocks, DataBlockItem{
				StartRow:   b.startIdx + 1, // 1-based
				EndRow:     b.endIdx,        // 1-based (endIdx is exclusive 0-based, so it equals 1-based last row)
				RowCount:   b.size,
				IsMain:     i == largestIdx,
				Preview:    preview,
				SampleRows: blockSampleRows,
			})
		}
	}

	return result, nil
}

// GetLatest returns the most recent cleaning session for a user
func (s *Service) GetLatest(ctx context.Context, userID uuid.UUID) (*CleaningSession, error) {
	return s.repo.GetLatestByUser(ctx, userID)
}

// GetPreview returns a preview of the cleaned data
func (s *Service) GetPreview(ctx context.Context, sessionID, userID uuid.UUID) (*PreviewResult, error) {
	session, err := s.repo.GetByIDAndUser(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}

	// Load the refined CSV file
	data, err := upload.LoadSheetData(session.RefinedFilePath, "Sheet1", "csv")
	if err != nil {
		return nil, fmt.Errorf("無法載入清理後資料: %w", err)
	}

	// Convert to preview format
	var rows [][]string
	for _, row := range data.Rows {
		var strRow []string
		for _, cell := range row {
			strRow = append(strRow, cell.Raw)
		}
		rows = append(rows, strRow)
	}

	return &PreviewResult{
		Headers:  data.Headers,
		Rows:     rows,
		RowCount: data.RowCount,
		ColCount: data.ColCount,
	}, nil
}

// GetLog returns the cleaning log entries for a session
func (s *Service) GetLog(ctx context.Context, sessionID, userID uuid.UUID) ([]LogEntry, error) {
	session, err := s.repo.GetByIDAndUser(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}

	if session.CleaningLog == nil {
		return []LogEntry{}, nil
	}
	return session.CleaningLog, nil
}

// generateRefinedPath generates the storage path for refined data
func (s *Service) generateRefinedPath(sessionID uuid.UUID) string {
	idStr := sessionID.String()
	subDir := idStr[:2]
	return filepath.Join(s.cfg.UploadDir, "refined", subDir, idStr+".csv")
}

// ApplyInteractiveEdits executes user's interactive cell edits in strict order:
// replace → remark_split → header_rename → delete_row (descending index)
func (s *Service) ApplyInteractiveEdits(ctx context.Context, userID uuid.UUID, req InteractiveFixRequest) (*InteractiveFixResponse, error) {
	// 1. Load assessment to get upload/sheet data
	assess, err := s.assessRepo.GetByID(ctx, req.AssessmentID)
	if err != nil {
		return nil, fmt.Errorf("無法取得評估記錄: %w", err)
	}

	// 2. Load upload record and SheetData
	up, err := s.uploadRepo.GetByID(ctx, assess.UploadID)
	if err != nil {
		return nil, fmt.Errorf("無法取得上傳記錄: %w", err)
	}

	ext := strings.TrimPrefix(filepath.Ext(up.Filename), ".")
	ext = strings.ToLower(ext)

	sheetName := ""
	if up.SelectedSheet != nil {
		sheetName = *up.SelectedSheet
	} else if len(up.SheetNames) > 0 {
		sheetName = up.SheetNames[0]
	}

	data, err := upload.LoadSheetData(up.FilePath, sheetName, ext)
	if err != nil {
		return nil, fmt.Errorf("無法載入工作表資料: %w", err)
	}

	operatorID := userID.String()
	var logEntries []LogEntry
	var warnings []string
	rowsAffected := 0

	// 3. Categorize edits by action type
	var replaceEdits, remarkSplitEdits, headerRenameEdits, deleteRowEdits []CellEdit
	for _, edit := range req.Edits {
		switch edit.Action {
		case "replace":
			replaceEdits = append(replaceEdits, edit)
		case "remark_split":
			remarkSplitEdits = append(remarkSplitEdits, edit)
		case "header_rename":
			headerRenameEdits = append(headerRenameEdits, edit)
		case "delete_row":
			deleteRowEdits = append(deleteRowEdits, edit)
		case "keep":
			// Do nothing
		}
	}

	// 3a. Apply replace edits first
	for _, edit := range replaceEdits {
		if edit.RowIndex < 0 || edit.RowIndex >= len(data.Rows) ||
			edit.ColIndex < 0 || edit.ColIndex >= data.ColCount {
			warnings = append(warnings, fmt.Sprintf("跳過: replace 指令超出範圍 (row=%d, col=%d)", edit.RowIndex, edit.ColIndex))
			continue
		}
		// Ensure the row has enough columns
		for len(data.Rows[edit.RowIndex]) <= edit.ColIndex {
			data.Rows[edit.RowIndex] = append(data.Rows[edit.RowIndex], upload.CellValue{IsEmpty: true})
		}
		oldValue := data.Rows[edit.RowIndex][edit.ColIndex].Raw
		data.Rows[edit.RowIndex][edit.ColIndex] = upload.CellValue{
			Raw:     edit.Value,
			IsEmpty: edit.Value == "",
		}
		logEntries = append(logEntries, LogEntry{
			OperationType: "cell_edit",
			AffectedRows:  []int{edit.RowIndex},
			Timestamp:     time.Now(),
			OperatorID:    operatorID,
			Details:       fmt.Sprintf("儲存格 [%d,%d] 值由 \"%s\" 修改為 \"%s\"", edit.RowIndex, edit.ColIndex, oldValue, edit.Value),
		})
		rowsAffected++
	}

	// 3b. Apply remark_split edits second
	for _, edit := range remarkSplitEdits {
		if edit.RowIndex < 0 || edit.RowIndex >= len(data.Rows) ||
			edit.ColIndex < 0 || edit.ColIndex >= data.ColCount {
			warnings = append(warnings, fmt.Sprintf("跳過: remark_split 指令超出範圍 (row=%d, col=%d)", edit.RowIndex, edit.ColIndex))
			continue
		}
		if edit.ColIndex >= len(data.Rows[edit.RowIndex]) || data.Rows[edit.RowIndex][edit.ColIndex].IsEmpty {
			warnings = append(warnings, fmt.Sprintf("跳過: remark_split 目標儲存格為空 (row=%d, col=%d)", edit.RowIndex, edit.ColIndex))
			continue
		}

		cellVal := data.Rows[edit.RowIndex][edit.ColIndex].Raw
		structural, remark := extractLastRemark(cellVal)
		if remark == "" {
			// No remark to extract, skip
			warnings = append(warnings, fmt.Sprintf("跳過: remark_split 未偵測到括號備註 (row=%d, col=%d)", edit.RowIndex, edit.ColIndex))
			continue
		}

		// Find or create 備註 column
		remarkColIdx := -1
		for i, h := range data.Headers {
			if h == "備註" {
				remarkColIdx = i
				break
			}
		}
		if remarkColIdx < 0 {
			// Append 備註 as last column
			remarkColIdx = data.ColCount
			data.Headers = append(data.Headers, "備註")
			data.ColCount++
			// Extend all rows to have the new column
			for i := range data.Rows {
				data.Rows[i] = append(data.Rows[i], upload.CellValue{IsEmpty: true})
			}
		}

		// Update original cell with structural value
		data.Rows[edit.RowIndex][edit.ColIndex] = upload.CellValue{
			Raw:     structural,
			IsEmpty: structural == "",
		}
		// Place remark in 備註 column
		for len(data.Rows[edit.RowIndex]) <= remarkColIdx {
			data.Rows[edit.RowIndex] = append(data.Rows[edit.RowIndex], upload.CellValue{IsEmpty: true})
		}
		data.Rows[edit.RowIndex][remarkColIdx] = upload.CellValue{
			Raw:     remark,
			IsEmpty: false,
		}

		logEntries = append(logEntries, LogEntry{
			OperationType: "remark_split",
			AffectedRows:  []int{edit.RowIndex},
			Timestamp:     time.Now(),
			OperatorID:    operatorID,
			Details:       fmt.Sprintf("儲存格 [%d,%d] 備註分離: \"%s\" → 結構值=\"%s\", 備註=\"%s\"", edit.RowIndex, edit.ColIndex, cellVal, structural, remark),
		})
		rowsAffected++
	}

	// 3c. Apply header_rename edits third
	for _, edit := range headerRenameEdits {
		if edit.ColIndex < 0 || edit.ColIndex >= data.ColCount {
			warnings = append(warnings, fmt.Sprintf("跳過: header_rename 指令超出範圍 (col=%d)", edit.ColIndex))
			continue
		}
		oldHeader := data.Headers[edit.ColIndex]
		data.Headers[edit.ColIndex] = edit.Value

		logEntries = append(logEntries, LogEntry{
			OperationType: "header_rename",
			AffectedRows:  []int{},
			Timestamp:     time.Now(),
			OperatorID:    operatorID,
			Details:       fmt.Sprintf("欄位 %d 標題由 \"%s\" 重新命名為 \"%s\"", edit.ColIndex, oldHeader, edit.Value),
		})
		rowsAffected++
	}

	// 3d. Apply delete_row edits last (descending index order)
	// Collect unique valid indices
	deleteIndices := make(map[int]bool)
	for _, edit := range deleteRowEdits {
		if edit.RowIndex < 0 || edit.RowIndex >= len(data.Rows) {
			warnings = append(warnings, fmt.Sprintf("跳過: delete_row 指令超出範圍 (row=%d)", edit.RowIndex))
			continue
		}
		deleteIndices[edit.RowIndex] = true
	}

	if len(deleteIndices) > 0 {
		// Sort descending to avoid index shifting
		sortedIndices := make([]int, 0, len(deleteIndices))
		for idx := range deleteIndices {
			sortedIndices = append(sortedIndices, idx)
		}
		sort.Sort(sort.Reverse(sort.IntSlice(sortedIndices)))

		for _, idx := range sortedIndices {
			if idx < len(data.Rows) {
				data.Rows = append(data.Rows[:idx], data.Rows[idx+1:]...)
				logEntries = append(logEntries, LogEntry{
					OperationType: "delete_row",
					AffectedRows:  []int{idx},
					Timestamp:     time.Now(),
					OperatorID:    operatorID,
					Details:       fmt.Sprintf("刪除第 %d 列", idx),
				})
			}
		}
		data.RowCount = len(data.Rows)
		rowsAffected += len(sortedIndices)
	}

	// 6. Save the modified data
	sessionID := uuid.New()
	refinedPath := s.generateRefinedPath(sessionID)

	if err := s.saveRefinedData(data, refinedPath); err != nil {
		return nil, fmt.Errorf("無法儲存修正結果: %w", err)
	}

	// Save session to DB
	session := &CleaningSession{
		ID:           sessionID,
		AssessmentID: req.AssessmentID,
		UserID:       userID,
		RulesApplied: []string{"interactive_edit"},
		RowsBefore:   assess.TotalRows,
		RowsAfter:    data.RowCount,
		ScoreBefore:  assess.TotalScore,
		ScoreAfter:   assess.TotalScore, // score unchanged until re-assessment
		CleaningLog:  logEntries,
		RefinedFilePath: refinedPath,
	}

	if err := s.repo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("無法儲存梳理記錄: %w", err)
	}

	return &InteractiveFixResponse{
		Success:      true,
		RowsAffected: rowsAffected,
		Warnings:     warnings,
		LogEntries:   logEntries,
	}, nil
}

// extractLastRemark extracts the last parenthesized remark from a cell value.
// It looks for the last occurrence of full-width （） or half-width () brackets
// where the inner content contains Chinese characters or has length > 5.
// Returns (structural_value, remark_content).
func extractLastRemark(val string) (string, string) {
	// Try full-width brackets first, then half-width
	type bracketPair struct {
		open  string
		close string
	}
	pairs := []bracketPair{
		{"（", "）"},
		{"(", ")"},
	}

	for _, bp := range pairs {
		// Find the last opening bracket
		lastOpen := strings.LastIndex(val, bp.open)
		if lastOpen < 0 {
			continue
		}

		afterOpen := val[lastOpen+len(bp.open):]
		closeIdx := strings.Index(afterOpen, bp.close)
		if closeIdx < 0 {
			continue
		}

		innerText := afterOpen[:closeIdx]

		// Check if this qualifies as a remark (Chinese chars or length > 5)
		if containsChinese(innerText) || utf8.RuneCountInString(innerText) > 5 {
			structural := strings.TrimSpace(val[:lastOpen])
			return structural, innerText
		}
	}

	return val, ""
}

// saveRefinedData saves the cleaned SheetData as a CSV file
func (s *Service) saveRefinedData(data *upload.SheetData, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	// Write headers
	if err := writer.Write(data.Headers); err != nil {
		return err
	}

	// Write data rows
	for _, row := range data.Rows {
		record := make([]string, data.ColCount)
		for i := 0; i < data.ColCount; i++ {
			if i < len(row) {
				record[i] = row[i].Raw
			}
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}
