package upload

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/safe-ai/excel-brushing-tool/pkg/response"
)

// Handler 處理 upload 相關的 HTTP 請求
type Handler struct {
	service *Service
}

// NewHandler 建立新的 upload Handler
func NewHandler(svc *Service) *Handler {
	return &Handler{service: svc}
}

// Upload 處理檔案上傳 POST /api/upload
func (h *Handler) Upload(c *gin.Context) {
	// 從 context 取得 user_id（由 JWT middleware 設定）
	userID, err := h.getUserID(c)
	if err != nil {
		response.SendAuthError(c, "未認證")
		return
	}

	// 取得上傳檔案
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.SendValidationError(c, "請上傳檔案")
		return
	}

	// 開啟檔案
	file, err := fileHeader.Open()
	if err != nil {
		response.SendValidationError(c, "無法讀取上傳檔案")
		return
	}
	defer file.Close()

	// 呼叫 service 處理上傳
	upload, err := h.service.Upload(c.Request.Context(), userID, file, fileHeader.Filename, fileHeader.Size)
	if err != nil {
		h.handleUploadError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":             upload.ID,
		"filename":       upload.Filename,
		"file_size":      upload.FileSize,
		"row_count":      upload.RowCount,
		"col_count":      upload.ColCount,
		"sheet_names":    upload.SheetNames,
		"selected_sheet": upload.SelectedSheet,
		"created_at":     upload.CreatedAt,
	})
}

// GetSheets 取得工作表列表 GET /api/upload/:id/sheets
func (h *Handler) GetSheets(c *gin.Context) {
	userID, err := h.getUserID(c)
	if err != nil {
		response.SendAuthError(c, "未認證")
		return
	}

	uploadID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendValidationError(c, "無效的上傳 ID")
		return
	}

	sheets, err := h.service.GetSheets(c.Request.Context(), uploadID, userID)
	if err != nil {
		if errors.Is(err, ErrUploadNotFound) {
			response.SendNotFoundError(c, "上傳記錄不存在")
			return
		}
		response.SendInternalError(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sheets": sheets,
	})
}

// SelectSheet 選取工作表 POST /api/upload/:id/select-sheet
func (h *Handler) SelectSheet(c *gin.Context) {
	userID, err := h.getUserID(c)
	if err != nil {
		response.SendAuthError(c, "未認證")
		return
	}

	uploadID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendValidationError(c, "無效的上傳 ID")
		return
	}

	var req SelectSheetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendValidationError(c, "請提供工作表名稱")
		return
	}

	err = h.service.SelectSheet(c.Request.Context(), uploadID, userID, req.SheetName)
	if err != nil {
		switch {
		case errors.Is(err, ErrUploadNotFound):
			response.SendNotFoundError(c, "上傳記錄不存在")
		case errors.Is(err, ErrSheetNotFound):
			response.SendValidationError(c, "指定的工作表不存在")
		default:
			response.SendInternalError(c)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "已選取工作表",
		"selected_sheet": req.SheetName,
	})
}

// getUserID 從 gin context 取得 user_id
func (h *Handler) getUserID(c *gin.Context) (uuid.UUID, error) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, errors.New("未認證")
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.New("無效的使用者身份")
	}
	return userID, nil
}

// handleUploadError 處理上傳錯誤的 HTTP 回應
func (h *Handler) handleUploadError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidFormat):
		response.SendValidationError(c, "不支援的檔案格式，僅支援 xlsx 和 csv")
	case errors.Is(err, ErrFileTooLarge):
		response.SendValidationError(c, "檔案大小超過 50MB 上限")
	case errors.Is(err, ErrTooManyRows):
		response.SendValidationError(c, "資料列數超過 100,000 列上限")
	case errors.Is(err, ErrFileCorrupted):
		response.SendValidationError(c, "檔案已損壞或無法讀取")
	default:
		response.SendInternalError(c)
	}
}
