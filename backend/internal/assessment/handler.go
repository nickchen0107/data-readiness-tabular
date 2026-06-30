package assessment

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/safe-ai/excel-brushing-tool/internal/upload"
	"github.com/safe-ai/excel-brushing-tool/pkg/response"
)

// Handler 處理 assessment 相關的 HTTP 請求
type Handler struct {
	service *Service
}

// NewHandler 建立新的 assessment Handler
func NewHandler(svc *Service) *Handler {
	return &Handler{service: svc}
}

// RunAssessment 處理 POST /api/assess
// 請求 body: {"upload_id": "uuid", "sheet_name": "Sheet1"}
func (h *Handler) RunAssessment(c *gin.Context) {
	var req RunAssessmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendValidationError(c, "請提供 upload_id 和 sheet_name")
		return
	}

	assessment, err := h.service.RunAssessment(c.Request.Context(), req.UploadID, req.SheetName)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, assessment)
}

// GetAssessment 處理 GET /api/assess/:id
// 回傳完整的評估結果
func (h *Handler) GetAssessment(c *gin.Context) {
	assessmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendValidationError(c, "無效的評估 ID")
		return
	}

	assessment, err := h.service.GetAssessment(c.Request.Context(), assessmentID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, assessment)
}

// GetIssues 處理 GET /api/assess/:id/issues
// 回傳問題列表
func (h *Handler) GetIssues(c *gin.Context) {
	assessmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendValidationError(c, "無效的評估 ID")
		return
	}

	issues, err := h.service.GetIssues(c.Request.Context(), assessmentID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"issues": issues,
	})
}

// GetLatest 處理 GET /api/assess/latest
// 回傳當前使用者的最新評估結果
func (h *Handler) GetLatest(c *gin.Context) {
	assessment, err := h.service.GetLatest(c.Request.Context())
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, assessment)
}

// handleError 處理評估相關錯誤的 HTTP 回應
func (h *Handler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrAssessmentNotFound):
		response.SendNotFoundError(c, "評估記錄不存在")
	case errors.Is(err, upload.ErrUploadNotFound):
		response.SendNotFoundError(c, "上傳記錄不存在")
	case errors.Is(err, upload.ErrUnsupportedFormat):
		response.SendValidationError(c, "不支援的檔案格式")
	default:
		response.SendInternalError(c)
	}
}
