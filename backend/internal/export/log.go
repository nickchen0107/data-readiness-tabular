package export

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/safe-ai/excel-brushing-tool/internal/cleaning"
)

// GenerateLog writes the cleaning log as a human-readable text file.
// Returns the file path of the generated log file.
func GenerateLog(session *cleaning.CleaningSession, outputDir string) (string, error) {
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("建立輸出目錄失敗: %w", err)
	}

	// Format each log entry as a human-readable line
	lines := make([]string, 0, len(session.CleaningLog))
	for _, entry := range session.CleaningLog {
		lines = append(lines, formatLogEntry(entry))
	}

	// Join all lines and write to file
	content := strings.Join(lines, "\n")

	filename := fmt.Sprintf("cleaning_%s.log", session.ID.String()[:8])
	filePath := filepath.Join(outputDir, filename)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("寫入清理日誌檔案失敗: %w", err)
	}

	return filePath, nil
}

// formatLogEntry formats a single LogEntry as a human-readable line.
// Format: [2006-01-02 15:04:05] {label}{rows}{details}
func formatLogEntry(entry cleaning.LogEntry) string {
	timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")
	label := operationTypeLabel(entry.OperationType)

	var rows string
	if len(entry.AffectedRows) > 0 {
		rowStrs := make([]string, len(entry.AffectedRows))
		for i, r := range entry.AffectedRows {
			rowStrs[i] = strconv.Itoa(r)
		}
		rows = "：第 " + strings.Join(rowStrs, ", ") + " 列"
	}

	var details string
	if entry.Details != "" {
		details = " — " + entry.Details
	}

	return fmt.Sprintf("[%s] %s%s%s", timestamp, label, rows, details)
}

// operationTypeLabel maps operation type strings to Chinese labels.
func operationTypeLabel(opType string) string {
	switch opType {
	case "dedup":
		return "移除重複列"
	case "date_normalize":
		return "統一日期格式"
	case "name_normalize":
		return "客戶名正規化"
	case "subtotal_remove":
		return "移除小計列"
	case "delete_row":
		return "刪除指定列"
	case "fill_na":
		return "填補空值"
	case "empty_col_remove":
		return "移除空白欄位"
	case "keep_block":
		return "保留資料區塊"
	default:
		return opType
	}
}
