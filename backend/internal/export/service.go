package export

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"

	"github.com/safe-ai/excel-brushing-tool/internal/assessment"
	"github.com/safe-ai/excel-brushing-tool/internal/cleaning"
	"github.com/safe-ai/excel-brushing-tool/pkg/config"
)

// Service handles export operations for cleaning sessions
type Service struct {
	cleanRepo *cleaning.Repository
	assessRepo *assessment.Repository
	cfg       *config.Config
}

// NewService creates a new export Service
func NewService(cleanRepo *cleaning.Repository, assessRepo *assessment.Repository, cfg *config.Config) *Service {
	return &Service{
		cleanRepo:  cleanRepo,
		assessRepo: assessRepo,
		cfg:        cfg,
	}
}

// GetSessionWithOwnership retrieves a cleaning session verifying user ownership
func (s *Service) GetSessionWithOwnership(ctx context.Context, sessionID, userID uuid.UUID) (*cleaning.CleaningSession, error) {
	session, err := s.cleanRepo.GetByIDAndUser(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}
	return session, nil
}

// GenerateExcelFile generates or retrieves the refined xlsx file for a session.
// Returns the file path to the generated/cached xlsx file.
func (s *Service) GenerateExcelFile(ctx context.Context, session *cleaning.CleaningSession) (string, error) {
	outputDir := s.getOutputDir(session.ID)

	// Check if xlsx file already exists (cached)
	cachedPath := filepath.Join(outputDir, fmt.Sprintf("refined_%s.xlsx", session.ID.String()[:8]))
	if _, err := os.Stat(cachedPath); err == nil {
		return cachedPath, nil
	}

	// Need to generate xlsx from the refined data (stored as CSV by the cleaning service)
	if session.RefinedFilePath != "" {
		if _, err := os.Stat(session.RefinedFilePath); err == nil {
			// Read the CSV file and convert to xlsx
			headers, rows, err := readCSVData(session.RefinedFilePath)
			if err != nil {
				return "", fmt.Errorf("讀取精煉檔案失敗: %w", err)
			}

			// Ensure output dir exists
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				return "", fmt.Errorf("建立輸出目錄失敗: %w", err)
			}

			return GenerateExcel(session, headers, rows, outputDir)
		}
	}

	return "", fmt.Errorf("無法產生 Excel 檔案：找不到精煉資料")
}

// GeneratePDFFile generates the PDF report for a session.
// Returns the file path to the generated PDF.
func (s *Service) GeneratePDFFile(ctx context.Context, session *cleaning.CleaningSession, locale string) (string, error) {
	outputDir := s.getOutputDir(session.ID)

	// Check if cached (include locale in cache key)
	cachedPath := filepath.Join(outputDir, fmt.Sprintf("report_%s_%s.pdf", session.ID.String()[:8], locale))
	if _, err := os.Stat(cachedPath); err == nil {
		return cachedPath, nil
	}

	// Get assessment for the report data
	assess, err := s.assessRepo.GetByID(ctx, session.AssessmentID)
	if err != nil {
		return "", fmt.Errorf("取得評估記錄失敗: %w", err)
	}

	reportData := &PDFReportData{
		Session:    session,
		Assessment: assess,
		Issues:     assess.Issues,
		Locale:     locale,
	}

	return GeneratePDF(reportData, s.cfg, outputDir)
}

// GenerateLogFile generates the cleaning log JSON file for a session.
// Returns the file path to the generated log file.
func (s *Service) GenerateLogFile(ctx context.Context, session *cleaning.CleaningSession, locale string) (string, error) {
	outputDir := s.getOutputDir(session.ID)

	// Check if cached (include locale in cache key)
	cachedPath := filepath.Join(outputDir, fmt.Sprintf("cleaning_%s_%s.log", session.ID.String()[:8], locale))
	if _, err := os.Stat(cachedPath); err == nil {
		return cachedPath, nil
	}

	return GenerateLog(session, outputDir, locale)
}

// getOutputDir returns the output directory for a session's export files
func (s *Service) getOutputDir(sessionID uuid.UUID) string {
	return filepath.Join(s.cfg.UploadDir, "exports", sessionID.String())
}

// readCSVData reads headers and rows from a CSV file
func readCSVData(filePath string) ([]string, [][]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	allRows, err := reader.ReadAll()
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

// readExcelData reads headers and rows from an existing xlsx file
func readExcelData(filePath string) ([]string, [][]string, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	sheetName := f.GetSheetName(0)
	allRows, err := f.GetRows(sheetName)
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
