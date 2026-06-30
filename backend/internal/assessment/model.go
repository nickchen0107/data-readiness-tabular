package assessment

import (
	"time"

	"github.com/google/uuid"
)

// Assessment 評估結果完整記錄
type Assessment struct {
	ID                 uuid.UUID          `json:"id"`
	UploadID           uuid.UUID          `json:"upload_id"`
	Filename           string             `json:"filename,omitempty"`
	TotalRows          int                `json:"total_rows,omitempty"`
	TotalScore         float64            `json:"total_score"`
	RowCompleteness    float64            `json:"row_completeness"`
	ColumnCompleteness float64            `json:"column_completeness"`
	FormatConsistency  float64            `json:"format_consistency"`
	DuplicateSimilar   float64            `json:"duplicate_similar"`
	TableStructure     float64            `json:"table_structure"`
	AIQueryReadiness   float64            `json:"ai_query_readiness"`
	WeightsSnapshot    Weights            `json:"weights_snapshot"`
	Status             string             `json:"status"` // "ready", "conditional", "not_ready"
	Issues             []Issue            `json:"issues"`
	ColumnDetails      []ColumnDetail     `json:"column_details"`
	RowDistribution    RowDistribution    `json:"row_distribution"`
	CreatedAt          time.Time          `json:"created_at"`
}

// RowDistribution 每列 readiness 等級分佈統計
type RowDistribution struct {
	High   int `json:"high"`   // 列非空率 >= 80%
	Medium int `json:"medium"` // 列非空率 50%-79%
	Low    int `json:"low"`    // 列非空率 < 50%
}

// RunAssessmentRequest 執行評估的 HTTP 請求
type RunAssessmentRequest struct {
	UploadID  uuid.UUID `json:"upload_id" binding:"required"`
	SheetName string    `json:"sheet_name" binding:"required"`
}
