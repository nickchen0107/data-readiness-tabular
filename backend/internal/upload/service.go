package upload

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/safe-ai/excel-brushing-tool/pkg/config"
)

// Service 相關錯誤
var (
	ErrFileTooLarge     = errors.New("檔案大小超過 50MB 上限")
	ErrInvalidFormat    = errors.New("不支援的檔案格式，僅支援 xlsx 和 csv")
	ErrSheetNotFound    = errors.New("指定的工作表不存在")
)

// Service 處理上傳業務邏輯
type Service struct {
	repo *Repository
	cfg  *config.Config
}

// NewService 建立新的 upload Service
func NewService(repo *Repository, cfg *config.Config) *Service {
	return &Service{repo: repo, cfg: cfg}
}

// Upload 處理檔案上傳
// 驗證格式、大小，儲存檔案，解析 metadata，寫入資料庫
func (s *Service) Upload(ctx context.Context, userID uuid.UUID, file io.Reader, filename string, fileSize int64) (*Upload, error) {
	// 1. 驗證檔案格式
	ext := getFileExtension(filename)
	if !s.isAllowedFormat(ext) {
		return nil, ErrInvalidFormat
	}

	// 2. 驗證檔案大小
	maxSize := int64(s.cfg.Upload.MaxFileSizeMB) * 1024 * 1024
	if fileSize > maxSize {
		return nil, ErrFileTooLarge
	}

	// 3. 產生 UUID-based 儲存路徑
	uploadID := uuid.New()
	storagePath := s.generateStoragePath(uploadID, ext)

	// 4. 確保目錄存在
	dir := filepath.Dir(storagePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("無法建立上傳目錄: %w", err)
	}

	// 5. 儲存檔案
	outFile, err := os.Create(storagePath)
	if err != nil {
		return nil, fmt.Errorf("無法建立檔案: %w", err)
	}
	defer outFile.Close()

	written, err := io.Copy(outFile, file)
	if err != nil {
		os.Remove(storagePath)
		return nil, fmt.Errorf("無法儲存檔案: %w", err)
	}

	// 再次確認實際寫入大小
	if written > maxSize {
		os.Remove(storagePath)
		return nil, ErrFileTooLarge
	}

	// 6. 解析檔案取得 metadata
	var parseResult *ParseResult
	switch ext {
	case "xlsx":
		parseResult, err = ParseXLSX(storagePath)
	case "csv":
		parseResult, err = ParseCSV(storagePath)
	default:
		os.Remove(storagePath)
		return nil, ErrInvalidFormat
	}
	if err != nil {
		os.Remove(storagePath)
		return nil, err
	}

	// 7. 檢查列數上限
	maxRows := s.cfg.Upload.MaxRowCount
	if maxRows > 0 && parseResult.RowCount > maxRows {
		os.Remove(storagePath)
		return nil, ErrTooManyRows
	}

	// 8. 如果只有一個工作表，自動選取
	var selectedSheet *string
	if len(parseResult.SheetNames) == 1 {
		selectedSheet = &parseResult.SheetNames[0]
	}

	// 9. 建立上傳記錄
	upload := &Upload{
		ID:            uploadID,
		UserID:        userID,
		Filename:      filename,
		FilePath:      storagePath,
		FileSize:      written,
		RowCount:      parseResult.RowCount,
		ColCount:      parseResult.ColCount,
		SelectedSheet: selectedSheet,
		SheetNames:    parseResult.SheetNames,
		MergedCells:   parseResult.MergedCells,
	}

	if err := s.repo.Create(ctx, upload); err != nil {
		os.Remove(storagePath)
		return nil, fmt.Errorf("無法儲存上傳記錄: %w", err)
	}

	return upload, nil
}

// GetSheets 取得上傳檔案的工作表列表
func (s *Service) GetSheets(ctx context.Context, uploadID, userID uuid.UUID) ([]string, error) {
	upload, err := s.repo.GetByIDAndUser(ctx, uploadID, userID)
	if err != nil {
		return nil, err
	}
	return upload.SheetNames, nil
}

// SelectSheet 選取工作表
func (s *Service) SelectSheet(ctx context.Context, uploadID, userID uuid.UUID, sheetName string) error {
	upload, err := s.repo.GetByIDAndUser(ctx, uploadID, userID)
	if err != nil {
		return err
	}

	// 驗證 sheetName 存在於 sheet_names 列表中
	found := false
	for _, name := range upload.SheetNames {
		if name == sheetName {
			found = true
			break
		}
	}
	if !found {
		return ErrSheetNotFound
	}

	return s.repo.UpdateSelectedSheet(ctx, uploadID, sheetName)
}

// GetUpload 取得上傳記錄（含所有權檢查）
func (s *Service) GetUpload(ctx context.Context, uploadID, userID uuid.UUID) (*Upload, error) {
	return s.repo.GetByIDAndUser(ctx, uploadID, userID)
}

// isAllowedFormat 檢查檔案格式是否允許
func (s *Service) isAllowedFormat(ext string) bool {
	allowedFormats := s.cfg.Upload.AllowedFormats
	if len(allowedFormats) == 0 {
		// 預設允許格式
		allowedFormats = []string{"xlsx", "csv"}
	}
	for _, allowed := range allowedFormats {
		if strings.EqualFold(ext, allowed) {
			return true
		}
	}
	return false
}

// generateStoragePath 產生 UUID-based 的檔案儲存路徑
func (s *Service) generateStoragePath(id uuid.UUID, ext string) string {
	// 使用 UUID 前 2 字元作為子目錄以分散檔案
	idStr := id.String()
	subDir := idStr[:2]
	return filepath.Join(s.cfg.UploadDir, subDir, idStr+"."+ext)
}
