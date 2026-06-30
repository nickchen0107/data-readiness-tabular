package comparison

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/safe-ai/excel-brushing-tool/internal/assessment"
	"github.com/safe-ai/excel-brushing-tool/internal/cleaning"
	"github.com/safe-ai/excel-brushing-tool/internal/upload"
)

// Service handles comparison business logic.
type Service struct {
	cleanRepo  *cleaning.Repository
	assessRepo *assessment.Repository
	assessSvc  *assessment.Service
}

// NewService creates a new comparison Service.
func NewService(cleanRepo *cleaning.Repository, assessRepo *assessment.Repository, assessSvc *assessment.Service) *Service {
	return &Service{
		cleanRepo:  cleanRepo,
		assessRepo: assessRepo,
		assessSvc:  assessSvc,
	}
}

// GetComparison retrieves the full comparison data for a cleaning session.
// It verifies ownership via userID, loads the original assessment, and runs
// a fresh assessment on the refined file to produce post-cleaning scores.
func (s *Service) GetComparison(ctx context.Context, sessionID, userID uuid.UUID) (*ComparisonResponse, error) {
	// 1. Get cleaning session with ownership verification
	session, err := s.cleanRepo.GetByIDAndUser(ctx, sessionID, userID)
	if err != nil {
		// cleaning.ErrSessionNotFound covers both "not found" and "wrong user"
		return nil, err
	}

	// 2. Get original assessment
	originalAssess, err := s.assessRepo.GetByID(ctx, session.AssessmentID)
	if err != nil {
		return nil, fmt.Errorf("無法取得原始評估記錄: %w", err)
	}

	// 3. Run assessment on the refined file to get post-cleaning scores
	postAssess, err := s.runPostCleaningAssessment(session)
	if err != nil {
		return nil, fmt.Errorf("無法計算梳理後評估: %w", err)
	}

	// 4. Assemble response
	resp := &ComparisonResponse{
		Session: SessionSummary{
			ID:           session.ID,
			RowsBefore:   session.RowsBefore,
			RowsAfter:    session.RowsAfter,
			ScoreBefore:  session.ScoreBefore,
			ScoreAfter:   session.ScoreAfter,
			RulesApplied: session.RulesApplied,
			CleaningLog:  session.CleaningLog,
			CreatedAt:    session.CreatedAt,
		},
		OriginalAssess: assessmentToSummary(originalAssess),
		PostCleanAssess: postAssess,
	}

	return resp, nil
}

// runPostCleaningAssessment loads the refined file and computes all indicators.
func (s *Service) runPostCleaningAssessment(session *cleaning.CleaningSession) (AssessmentSummary, error) {
	// Load refined CSV file
	data, err := upload.LoadSheetData(session.RefinedFilePath, "Sheet1", "csv")
	if err != nil {
		return AssessmentSummary{}, fmt.Errorf("無法載入清理後資料: %w", err)
	}

	// Calculate all 6 indicators
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

	// Calculate row distribution
	rowDist := assessment.CalculateRowDistribution(data)

	// Calculate total score and grade using default weights
	// (we use DefaultWeights here since we don't have context for DB lookup;
	// this matches the on-demand design rationale)
	weights := assessment.DefaultWeights()
	totalScore, grade, err := assessment.CalculateTotalScoreWithReadiness(indicators, weights, rowDist)
	if err != nil {
		// Fallback to simple calculation
		totalScore, grade, err = assessment.CalculateTotalScore(indicators, weights)
		if err != nil {
			return AssessmentSummary{}, fmt.Errorf("無法計算總分: %w", err)
		}
	}

	// Detect issues
	issues := assessment.DetectIssues(data, indicators)

	return AssessmentSummary{
		ID:                 session.ID, // Use session ID as synthetic assessment ID
		TotalScore:         totalScore,
		Status:             grade,
		RowCompleteness:    rowComp,
		ColumnCompleteness: colComp,
		FormatConsistency:  formatCon,
		DuplicateSimilar:   dupSim,
		TableStructure:     tableStr,
		AIQueryReadiness:   aiReady,
		Issues:             issues,
		RowDistribution:    rowDist,
	}, nil
}

// assessmentToSummary converts a full Assessment to an AssessmentSummary.
func assessmentToSummary(a *assessment.Assessment) AssessmentSummary {
	issues := a.Issues
	if issues == nil {
		issues = []assessment.Issue{}
	}
	return AssessmentSummary{
		ID:                 a.ID,
		TotalScore:         a.TotalScore,
		Status:             a.Status,
		RowCompleteness:    a.RowCompleteness,
		ColumnCompleteness: a.ColumnCompleteness,
		FormatConsistency:  a.FormatConsistency,
		DuplicateSimilar:   a.DuplicateSimilar,
		TableStructure:     a.TableStructure,
		AIQueryReadiness:   a.AIQueryReadiness,
		Issues:             issues,
		RowDistribution:    a.RowDistribution,
	}
}
