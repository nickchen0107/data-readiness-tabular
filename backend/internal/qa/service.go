package qa

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"

	"github.com/safe-ai/excel-brushing-tool/internal/assessment"
	"github.com/safe-ai/excel-brushing-tool/internal/cleaning"
	"github.com/safe-ai/excel-brushing-tool/internal/upload"
	"github.com/safe-ai/excel-brushing-tool/pkg/config"
)

// Service handles QA operations
type Service struct {
	geminiClient *GeminiClient
	cleanRepo    *cleaning.Repository
	assessRepo   *assessment.Repository
	uploadRepo   *upload.Repository
	cfg          *config.Config
}

// NewService creates a new QA Service
func NewService(
	geminiClient *GeminiClient,
	cleanRepo *cleaning.Repository,
	assessRepo *assessment.Repository,
	uploadRepo *upload.Repository,
	cfg *config.Config,
) *Service {
	return &Service{
		geminiClient: geminiClient,
		cleanRepo:    cleanRepo,
		assessRepo:   assessRepo,
		uploadRepo:   uploadRepo,
		cfg:          cfg,
	}
}

// Ask processes a QA question with consent check, guardrail, and dual Gemini calls
func (s *Service) Ask(ctx context.Context, req QARequest, userID uuid.UUID) (*QAResponse, error) {
	// 1. Check consent
	if !req.Consent {
		return nil, ErrConsentRequired
	}

	// 2. Get cleaning session with ownership verification
	session, err := s.cleanRepo.GetByIDAndUser(ctx, req.SessionID, userID)
	if err != nil {
		return nil, fmt.Errorf("取得梳理記錄失敗: %w", err)
	}

	// 3. Get original data from assessment
	assess, err := s.assessRepo.GetByID(ctx, session.AssessmentID)
	if err != nil {
		return nil, fmt.Errorf("取得評估記錄失敗: %w", err)
	}

	// Get the upload for original file path
	uploadRecord, err := s.uploadRepo.GetByID(ctx, assess.UploadID)
	if err != nil {
		return nil, fmt.Errorf("取得上傳記錄失敗: %w", err)
	}

	// 4. Read original data
	originalHeaders, originalRows, err := readExcelAsCSV(uploadRecord.FilePath, uploadRecord.SelectedSheet)
	if err != nil {
		return nil, fmt.Errorf("讀取原始檔案失敗: %w", err)
	}

	// 5. Check guardrail (data insufficiency) on original data
	threshold := s.cfg.LLM.DataInsufficiencyThreshold
	guardrailResult := CheckDataInsufficiency(originalHeaders, originalRows, req.Question, threshold)
	if guardrailResult.Blocked {
		return nil, &DataInsufficiencyError{Explanation: guardrailResult.Explanation}
	}

	// 6. Read cleaned data
	var cleanedHeaders []string
	var cleanedRows [][]string
	if session.RefinedFilePath != "" {
		cleanedHeaders, cleanedRows, err = readExcelAsCSV(session.RefinedFilePath, nil)
		if err != nil {
			return nil, fmt.Errorf("讀取精煉檔案失敗: %w", err)
		}
	} else {
		// Fallback to original if no refined file
		cleanedHeaders = originalHeaders
		cleanedRows = originalRows
	}

	// 7. Prepare CSV data snippets (max rows from config)
	maxRows := s.cfg.LLM.Prompt.MaxDataRows
	originalCSV := toCSVSnippet(originalHeaders, originalRows, maxRows)
	cleanedCSV := toCSVSnippet(cleanedHeaders, cleanedRows, maxRows)

	// 8. Call Gemini with original data
	originalAnswer, err := s.geminiClient.Ask(ctx, originalCSV, req.Question)
	if err != nil {
		return nil, fmt.Errorf("呼叫 Gemini (原始資料) 失敗: %w", err)
	}

	// 9. Call Gemini with cleaned data
	cleanedAnswer, err := s.geminiClient.Ask(ctx, cleanedCSV, req.Question)
	if err != nil {
		return nil, fmt.Errorf("呼叫 Gemini (梳理資料) 失敗: %w", err)
	}

	return &QAResponse{
		OriginalAnswer: originalAnswer,
		CleanedAnswer:  cleanedAnswer,
	}, nil
}

// GetSuggestions returns 3 suggested questions based on assessment columns
func (s *Service) GetSuggestions(ctx context.Context, assessID uuid.UUID) ([]string, error) {
	assess, err := s.assessRepo.GetByID(ctx, assessID)
	if err != nil {
		return nil, fmt.Errorf("取得評估記錄失敗: %w", err)
	}

	// Get the upload to read headers
	uploadRecord, err := s.uploadRepo.GetByID(ctx, assess.UploadID)
	if err != nil {
		return nil, fmt.Errorf("取得上傳記錄失敗: %w", err)
	}

	headers, _, err := readExcelAsCSV(uploadRecord.FilePath, uploadRecord.SelectedSheet)
	if err != nil {
		return nil, fmt.Errorf("讀取檔案標題失敗: %w", err)
	}

	return GenerateSuggestions(headers), nil
}

// readExcelAsCSV reads an Excel file and returns headers and rows as string arrays
func readExcelAsCSV(filePath string, sheetName *string) ([]string, [][]string, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	var sheet string
	if sheetName != nil && *sheetName != "" {
		sheet = *sheetName
	} else {
		sheet = f.GetSheetName(0)
	}

	allRows, err := f.GetRows(sheet)
	if err != nil {
		return nil, nil, err
	}

	if len(allRows) == 0 {
		return nil, nil, fmt.Errorf("檔案為空")
	}

	headers := allRows[0]
	var dataRows [][]string
	if len(allRows) > 1 {
		dataRows = allRows[1:]
	}

	return headers, dataRows, nil
}

// toCSVSnippet converts headers and rows to a CSV-formatted string, limited to maxRows
func toCSVSnippet(headers []string, rows [][]string, maxRows int) string {
	var sb strings.Builder

	// Write header
	sb.WriteString(strings.Join(headers, ","))
	sb.WriteString("\n")

	// Write data rows (limited)
	limit := len(rows)
	if maxRows > 0 && limit > maxRows {
		limit = maxRows
	}

	for i := 0; i < limit; i++ {
		sb.WriteString(strings.Join(rows[i], ","))
		sb.WriteString("\n")
	}

	return sb.String()
}

// Errors
var (
	ErrConsentRequired = fmt.Errorf("consent_required")
)

// DataInsufficiencyError indicates data is insufficient for the question
type DataInsufficiencyError struct {
	Explanation string
}

func (e *DataInsufficiencyError) Error() string {
	return e.Explanation
}
