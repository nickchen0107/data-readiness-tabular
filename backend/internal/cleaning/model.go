package cleaning

import (
	"time"

	"github.com/google/uuid"
)

// CleaningSession represents a complete cleaning operation with results
type CleaningSession struct {
	ID              uuid.UUID  `json:"id"`
	AssessmentID    uuid.UUID  `json:"assessment_id"`
	UserID          uuid.UUID  `json:"user_id"`
	RulesApplied    []string   `json:"rules_applied"`
	RowsBefore      int        `json:"rows_before"`
	RowsAfter       int        `json:"rows_after"`
	ScoreBefore     float64    `json:"score_before"`
	ScoreAfter      float64    `json:"score_after"`
	CleaningLog     []LogEntry `json:"cleaning_log"`
	RefinedFilePath   string     `json:"-"`
	OriginalFilename string     `json:"original_filename" db:"original_filename"`
	CreatedAt        time.Time  `json:"created_at"`
}

// LogEntry records a single cleaning operation for audit trail
type LogEntry struct {
	OperationType string    `json:"operation_type"`
	AffectedRows  []int     `json:"affected_rows"`
	Timestamp     time.Time `json:"timestamp"`
	OperatorID    string    `json:"operator_id"`
	Details       string    `json:"details"`
}

// CleanRequest is the input for applying cleaning rules
type CleanRequest struct {
	AssessmentID   uuid.UUID `json:"assessment_id" binding:"required"`
	Rules          []string  `json:"rules"`            // ["date_normalize", "dedup", "name_normalize", "subtotal_remove"]
	RowOps         []RowOp   `json:"row_ops"`          // individual row operations
	RemoveColumns  []int     `json:"remove_columns"`   // specific column indices to remove (used with empty_col_remove)
	KeepBlockIndex int       `json:"keep_block_index"` // which data block to keep (-1 = auto)
}

// RowOp represents an individual row-level operation
type RowOp struct {
	RowIndex int    `json:"row_index"`
	Action   string `json:"action"` // "fill_na" | "delete"
}

// PreviewResult contains a preview of the cleaned data
type PreviewResult struct {
	Headers  []string   `json:"headers"`
	Rows     [][]string `json:"rows"`
	RowCount int        `json:"row_count"`
	ColCount int        `json:"col_count"`
}

// RemovalPreview contains items that could be removed, letting the user choose
type RemovalPreview struct {
	AllColumns   []ColumnRemovalItem `json:"all_columns"`   // ALL columns with their empty rates
	EmptyColumns []ColumnRemovalItem `json:"empty_columns"` // only columns above threshold
	DataBlocks   []DataBlockItem     `json:"data_blocks"`
	SampleRows   []SampleRow         `json:"sample_rows"`   // first 15 data rows for display
	Headers      []string            `json:"headers"`       // all column headers
}

// SampleRow represents a single data row for preview display
type SampleRow struct {
	RowNumber int      `json:"row_number"` // 1-based Excel row
	Cells     []string `json:"cells"`      // cell values (truncated to 20 chars)
}

// ColumnRemovalItem represents a column candidate for removal
type ColumnRemovalItem struct {
	ColIndex  int     `json:"col_index"`
	ColName   string  `json:"col_name"`
	EmptyRate float64 `json:"empty_rate"` // 0-1
}

// DataBlockItem represents a contiguous data block in the sheet
type DataBlockItem struct {
	StartRow   int         `json:"start_row"`   // 1-based display row
	EndRow     int         `json:"end_row"`     // 1-based display row
	RowCount   int         `json:"row_count"`
	IsMain     bool        `json:"is_main"`     // true = largest block (recommended to keep)
	Preview    string      `json:"preview"`     // first row content preview
	SampleRows []SampleRow `json:"sample_rows"` // first 3 rows of this block
}

// InteractiveFixRequest 互動式修正 API 請求
type InteractiveFixRequest struct {
	AssessmentID uuid.UUID  `json:"assessment_id" binding:"required"`
	Edits        []CellEdit `json:"edits" binding:"required,min=1"`
}

// CellEdit 單筆儲存格修正指令
type CellEdit struct {
	RowIndex int    `json:"row_index"`
	ColIndex int    `json:"col_index"`
	Action   string `json:"action" binding:"required,oneof=replace keep delete_row remark_split header_rename"`
	Value    string `json:"value"`
}

// InteractiveFixResponse 互動式修正 API 回應
type InteractiveFixResponse struct {
	Success      bool       `json:"success"`
	RowsAffected int        `json:"rows_affected"`
	Warnings     []string   `json:"warnings"`
	LogEntries   []LogEntry `json:"log_entries"`
}
