package upload

import (
	"time"

	"github.com/google/uuid"
)

// Upload 上傳檔案的 metadata
type Upload struct {
	ID            uuid.UUID    `json:"id"`
	UserID        uuid.UUID    `json:"user_id"`
	Filename      string       `json:"filename"`
	FilePath      string       `json:"-"`
	FileSize      int64        `json:"file_size"`
	RowCount      int          `json:"row_count"`
	ColCount      int          `json:"col_count"`
	SelectedSheet *string      `json:"selected_sheet"`
	SheetNames    []string     `json:"sheet_names"`
	MergedCells   []MergedRange `json:"merged_cells"`
	CreatedAt     time.Time    `json:"created_at"`
}

// MergedRange 合併儲存格的範圍資訊
type MergedRange struct {
	StartRow int `json:"start_row"`
	EndRow   int `json:"end_row"`
	StartCol int `json:"start_col"`
	EndCol   int `json:"end_col"`
}

// SheetData 解析後的試算表資料，供 Assessment Engine 使用
type SheetData struct {
	Headers            []string
	Rows               [][]CellValue
	MergedCells        []MergedRange
	RowCount           int        // data rows (不含 header)
	ColCount           int
	TotalSheetRows     int        // sheet 的原始總列數（含所有列）
	HeaderRowIndex     int        // 被偵測為 header 的列 index (0-based)
	RawFirstRows       [][]string // 原始前 5 列（含 header 前的列，供 debug）
	CommentCells       []CellLocation // cells that have comments (red triangle)
	StrikethroughCells []CellLocation // cells with strikethrough formatting
}

// CellLocation identifies a specific cell position
type CellLocation struct {
	Row int // 0-based data row index
	Col int // 0-based column index
}

// CellValue 單一儲存格的值
type CellValue struct {
	Raw     string
	IsEmpty bool
}

// ParseResult 檔案解析結果
type ParseResult struct {
	SheetNames  []string
	MergedCells []MergedRange
	RowCount    int
	ColCount    int
}

// SelectSheetRequest 選取工作表的請求
type SelectSheetRequest struct {
	SheetName string `json:"sheet_name" binding:"required"`
}
