package comparison

import (
	"time"

	"github.com/google/uuid"
	"github.com/safe-ai/excel-brushing-tool/internal/assessment"
	"github.com/safe-ai/excel-brushing-tool/internal/cleaning"
)

// ComparisonResponse is the API response containing full before/after comparison data.
type ComparisonResponse struct {
	Session         SessionSummary    `json:"session"`
	OriginalAssess  AssessmentSummary `json:"original_assessment"`
	PostCleanAssess AssessmentSummary `json:"post_clean_assessment"`
}

// SessionSummary contains cleaning session metadata.
type SessionSummary struct {
	ID               uuid.UUID          `json:"id"`
	RowsBefore       int                `json:"rows_before"`
	RowsAfter        int                `json:"rows_after"`
	ScoreBefore      float64            `json:"score_before"`
	ScoreAfter       float64            `json:"score_after"`
	RulesApplied     []string           `json:"rules_applied"`
	CleaningLog      []cleaning.LogEntry `json:"cleaning_log"`
	OriginalFilename string             `json:"original_filename"`
	CreatedAt        time.Time          `json:"created_at"`
}

// AssessmentSummary contains the indicator scores and issues for one assessment.
type AssessmentSummary struct {
	ID                 uuid.UUID                  `json:"id"`
	TotalScore         float64                    `json:"total_score"`
	Status             string                     `json:"status"`
	RowCompleteness    float64                    `json:"row_completeness"`
	ColumnCompleteness float64                    `json:"column_completeness"`
	FormatConsistency  float64                    `json:"format_consistency"`
	DuplicateSimilar   float64                    `json:"duplicate_similar"`
	TableStructure     float64                    `json:"table_structure"`
	AIQueryReadiness   float64                    `json:"ai_query_readiness"`
	Issues             []assessment.Issue         `json:"issues"`
	RowDistribution    assessment.RowDistribution `json:"row_distribution"`
}
