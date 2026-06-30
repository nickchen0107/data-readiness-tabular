package export

import (
	"fmt"
	"path/filepath"

	"github.com/xuri/excelize/v2"

	"github.com/safe-ai/excel-brushing-tool/internal/cleaning"
)

// GenerateExcel creates a refined.xlsx file from cleaned session data.
// It writes headers with bold styling, all data rows, and sets reasonable column widths.
// Returns the file path of the generated Excel file.
func GenerateExcel(session *cleaning.CleaningSession, headers []string, rows [][]string, outputDir string) (string, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Sheet1"

	// Create bold style for headers
	boldStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
		},
	})
	if err != nil {
		return "", fmt.Errorf("建立樣式失敗: %w", err)
	}

	// Write headers in the first row
	for colIdx, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 1)
		f.SetCellValue(sheetName, cell, header)
		f.SetCellStyle(sheetName, cell, cell, boldStyle)
	}

	// Write data rows
	for rowIdx, row := range rows {
		for colIdx, val := range row {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
			f.SetCellValue(sheetName, cell, val)
		}
	}

	// Auto-fit column widths with reasonable defaults
	for colIdx, header := range headers {
		colName, _ := excelize.ColumnNumberToName(colIdx + 1)
		// Estimate width: max of header length and a sample of data
		maxWidth := float64(len([]rune(header)))
		sampleSize := 100
		if sampleSize > len(rows) {
			sampleSize = len(rows)
		}
		for i := 0; i < sampleSize; i++ {
			if colIdx < len(rows[i]) {
				cellLen := float64(len([]rune(rows[i][colIdx])))
				if cellLen > maxWidth {
					maxWidth = cellLen
				}
			}
		}
		// Clamp width between 10 and 50
		width := maxWidth * 1.2
		if width < 10 {
			width = 10
		}
		if width > 50 {
			width = 50
		}
		f.SetColWidth(sheetName, colName, colName, width)
	}

	// Save to the volume path
	filename := fmt.Sprintf("refined_%s.xlsx", session.ID.String()[:8])
	filePath := filepath.Join(outputDir, filename)
	if err := f.SaveAs(filePath); err != nil {
		return "", fmt.Errorf("儲存 Excel 檔案失敗: %w", err)
	}

	return filePath, nil
}

// GenerateExcelFromPath reads the refined file at session.RefinedFilePath
// and returns the path. If the file already exists, it returns it directly.
func GenerateExcelFromPath(session *cleaning.CleaningSession) (string, error) {
	if session.RefinedFilePath == "" {
		return "", fmt.Errorf("梳理記錄沒有精煉檔案路徑")
	}
	return session.RefinedFilePath, nil
}

// GenerateExcelForSession generates xlsx from stored refined file data.
// If the refined file already exists on disk, returns its path directly.
// Otherwise, it generates a new one from the provided data.
func GenerateExcelForSession(session *cleaning.CleaningSession, headers []string, rows [][]string, outputDir string) (string, error) {
	// If refined file already exists, use it
	if session.RefinedFilePath != "" {
		return session.RefinedFilePath, nil
	}

	// Generate new file
	sessionDir := filepath.Join(outputDir, session.ID.String())
	return GenerateExcel(session, headers, rows, sessionDir)
}


