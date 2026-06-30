package cleaning

import (
	"errors"
	"fmt"
	"time"

	"github.com/safe-ai/excel-brushing-tool/internal/upload"
)

// Row operation errors
var (
	ErrInvalidRowIndex = errors.New("無效的列索引")
	ErrRowOutOfBounds  = errors.New("列索引超出資料範圍")
)

// FillNA fills all empty cells in the specified row with "N/A".
func FillNA(data *upload.SheetData, rowIndex int, log *[]LogEntry, operatorID string) error {
	if data == nil {
		return ErrInvalidRowIndex
	}
	if rowIndex < 0 || rowIndex >= len(data.Rows) {
		return ErrRowOutOfBounds
	}

	row := data.Rows[rowIndex]
	filled := false

	for col := 0; col < data.ColCount; col++ {
		if col >= len(row) {
			// Extend the row if needed
			for len(data.Rows[rowIndex]) <= col {
				data.Rows[rowIndex] = append(data.Rows[rowIndex], upload.CellValue{Raw: "", IsEmpty: true})
			}
			data.Rows[rowIndex][col] = upload.CellValue{Raw: "N/A", IsEmpty: false}
			filled = true
		} else if row[col].IsEmpty {
			data.Rows[rowIndex][col] = upload.CellValue{Raw: "N/A", IsEmpty: false}
			filled = true
		}
	}

	if filled {
		*log = append(*log, LogEntry{
			OperationType: "fill_na",
			AffectedRows:  []int{rowIndex},
			Timestamp:     time.Now(),
			OperatorID:    operatorID,
			Details:       fmt.Sprintf("將第 %d 列的空白儲存格填入 N/A", rowIndex),
		})
	}

	return nil
}

// DeleteRow removes the row at the specified index.
func DeleteRow(data *upload.SheetData, rowIndex int, log *[]LogEntry, operatorID string) error {
	if data == nil {
		return ErrInvalidRowIndex
	}
	if rowIndex < 0 || rowIndex >= len(data.Rows) {
		return ErrRowOutOfBounds
	}

	// Remove the row
	data.Rows = append(data.Rows[:rowIndex], data.Rows[rowIndex+1:]...)
	data.RowCount = len(data.Rows)

	*log = append(*log, LogEntry{
		OperationType: "delete_row",
		AffectedRows:  []int{rowIndex},
		Timestamp:     time.Now(),
		OperatorID:    operatorID,
		Details:       fmt.Sprintf("刪除第 %d 列", rowIndex),
	})

	return nil
}
