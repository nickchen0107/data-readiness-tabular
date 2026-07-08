package assessment

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/safe-ai/excel-brushing-tool/internal/upload"
)

// Service 處理評估相關的業務邏輯
type Service struct {
	uploadRepo   *upload.Repository
	assessRepo   *Repository
	settingsRepo *SettingsRepository
}

// NewService 建立新的 assessment Service
func NewService(uploadRepo *upload.Repository, assessRepo *Repository, settingsRepo *SettingsRepository) *Service {
	return &Service{
		uploadRepo:   uploadRepo,
		assessRepo:   assessRepo,
		settingsRepo: settingsRepo,
	}
}

// RunAssessment 載入檔案資料，執行所有 6 項指標計算，計算總分，偵測問題，並儲存至資料庫
func (s *Service) RunAssessment(ctx context.Context, uploadID uuid.UUID, sheetName string) (*Assessment, error) {
	// 1. 取得上傳記錄
	up, err := s.uploadRepo.GetByID(ctx, uploadID)
	if err != nil {
		return nil, fmt.Errorf("無法取得上傳記錄: %w", err)
	}

	// 2. 從檔案副檔名判斷格式
	ext := strings.TrimPrefix(filepath.Ext(up.Filename), ".")
	ext = strings.ToLower(ext)

	// 取得檔案名稱（保留副檔名）
	filename := up.Filename

	// 3. 載入 SheetData
	data, err := upload.LoadSheetData(up.FilePath, sheetName, ext)
	if err != nil {
		return nil, fmt.Errorf("無法載入工作表資料: %w", err)
	}

	// 4. 取得當前權重設定
	weights, err := s.settingsRepo.GetWeights(ctx)
	if err != nil {
		return nil, fmt.Errorf("無法取得權重設定: %w", err)
	}

	// 5. 偵測新問題類型以取得分數整合所需的參數
	placeholderCells := CountPlaceholderCells(data)
	mismatchCells := CountTypeMismatchCells(data)
	orphanTotalDetected := HasOrphanTotalRows(data)
	inlineRemarkDense := IsInlineRemarkDense(data)
	emptyHeaderDetected := HasEmptyHeaders(data)

	// 6. 計算所有 6 項指標（整合新偵測結果）
	rowComp := CalculateRowCompleteness(data)
	colComp, columnDetails := CalculateColumnCompleteness(data)
	formatCon := CalculateFormatConsistencyWithIssues(data, placeholderCells, mismatchCells)
	dupSim := CalculateDuplicateSimilar(data)
	tableStr := CalculateTableStructureWithIssues(data, orphanTotalDetected, inlineRemarkDense)
	aiReady := CalculateAIQueryReadinessWithIssues(data, emptyHeaderDetected)

	indicators := IndicatorScores{
		RowCompleteness:    rowComp,
		ColumnCompleteness: colComp,
		FormatConsistency:  formatCon,
		DuplicateSimilar:   dupSim,
		TableStructure:     tableStr,
		AIQueryReadiness:   aiReady,
	}

	// 7. 計算每列 readiness 分佈 (needed for total score)
	rowDist := CalculateRowDistribution(data)

	// 8. 計算總分 + 等級 (incorporating row readiness)
	totalScore, grade, err := CalculateTotalScoreWithReadiness(indicators, weights, rowDist)
	if err != nil {
		return nil, fmt.Errorf("無法計算總分: %w", err)
	}

	// 9. 偵測問題
	issues := DetectIssues(data, indicators)

	// 10. 組裝 Assessment 記錄
	assessment := &Assessment{
		ID:                 uuid.New(),
		UploadID:           uploadID,
		Filename:           filename,
		TotalRows:          data.TotalSheetRows, // sheet 的全部列數
		TotalScore:         totalScore,
		RowCompleteness:    rowComp,
		ColumnCompleteness: colComp,
		FormatConsistency:  formatCon,
		DuplicateSimilar:   dupSim,
		TableStructure:     tableStr,
		AIQueryReadiness:   aiReady,
		WeightsSnapshot:    weights,
		Status:             grade,
		Issues:             issues,
		ColumnDetails:      columnDetails,
		RowDistribution:    rowDist,
	}

	// 11. 儲存至資料庫
	if err := s.assessRepo.Create(ctx, assessment); err != nil {
		return nil, fmt.Errorf("無法儲存評估記錄: %w", err)
	}

	// Log assessment result for debugging
	LogAssessmentResult(assessment.ID, data, assessment)

	return assessment, nil
}

// GetLatest 取得最新的評估記錄
func (s *Service) GetLatest(ctx context.Context) (*Assessment, error) {
	a, err := s.assessRepo.GetLatest(ctx)
	if err != nil {
		return nil, err
	}
	if a.Filename == "" {
		up, err := s.uploadRepo.GetByID(ctx, a.UploadID)
		if err == nil {
			a.Filename = up.Filename
		}
	}
	return a, nil
}

// GetAssessment 取得評估記錄
func (s *Service) GetAssessment(ctx context.Context, assessmentID uuid.UUID) (*Assessment, error) {
	a, err := s.assessRepo.GetByID(ctx, assessmentID)
	if err != nil {
		return nil, err
	}
	// Populate filename from upload record if not stored
	if a.Filename == "" {
		up, err := s.uploadRepo.GetByID(ctx, a.UploadID)
		if err == nil {
			a.Filename = up.Filename
		}
	}
	return a, nil
}

// GetIssues 取得評估問題列表
func (s *Service) GetIssues(ctx context.Context, assessmentID uuid.UUID) ([]Issue, error) {
	a, err := s.assessRepo.GetByID(ctx, assessmentID)
	if err != nil {
		return nil, err
	}
	if a.Issues == nil {
		return []Issue{}, nil
	}
	return a.Issues, nil
}
