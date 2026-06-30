package assessment

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/safe-ai/excel-brushing-tool/internal/upload"
)

// LogAssessmentResult writes assessment details to a log file for debugging and audit.
func LogAssessmentResult(assessID uuid.UUID, data *upload.SheetData, a *Assessment) {
	logDir := "/app/uploads/logs"
	os.MkdirAll(logDir, 0755)

	filename := filepath.Join(logDir, fmt.Sprintf("assess_%s_%s.json",
		assessID.String()[:8], time.Now().Format("20060102_150405")))

	logData := map[string]interface{}{
		"assessment_id":      assessID.String(),
		"timestamp":          time.Now().Format(time.RFC3339),
		"filename":           a.Filename,
		"total_sheet_rows":   a.TotalRows,
		"data_rows":          data.RowCount,
		"col_count":          data.ColCount,
		"header_row_index":   data.HeaderRowIndex,
		"headers":            data.Headers,
		"scores": map[string]float64{
			"total":               a.TotalScore,
			"row_completeness":    a.RowCompleteness,
			"column_completeness": a.ColumnCompleteness,
			"format_consistency":  a.FormatConsistency,
			"duplicate_similar":   a.DuplicateSimilar,
			"table_structure":     a.TableStructure,
			"ai_query_readiness":  a.AIQueryReadiness,
		},
		"row_distribution": a.RowDistribution,
		"issues_count":     len(a.Issues),
	}

	// 記錄前 5 列原始資料（含 header 前的列）供 debug
	logData["raw_first_rows"] = data.RawFirstRows

	// 記錄合併儲存格資訊供 debug
	logData["merged_cells_count"] = len(data.MergedCells)
	if len(data.MergedCells) > 0 {
		mcSample := data.MergedCells
		if len(mcSample) > 5 {
			mcSample = mcSample[:5]
		}
		logData["merged_cells_sample"] = mcSample
	}

	jsonBytes, err := json.MarshalIndent(logData, "", "  ")
	if err != nil {
		log.Printf("無法序列化評估日誌: %v", err)
		return
	}

	if err := os.WriteFile(filename, jsonBytes, 0644); err != nil {
		log.Printf("無法寫入評估日誌: %v", err)
		return
	}
	log.Printf("評估日誌已儲存: %s", filename)
}
