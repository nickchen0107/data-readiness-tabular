package admin

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/safe-ai/excel-brushing-tool/internal/assessment"
	"github.com/safe-ai/excel-brushing-tool/internal/auth"
	"github.com/safe-ai/excel-brushing-tool/internal/quota"
	"github.com/safe-ai/excel-brushing-tool/internal/translation"
	"github.com/safe-ai/excel-brushing-tool/pkg/response"
)

// Handler 處理管理後台相關的 HTTP 請求
type Handler struct {
	authRepo    *auth.Repository
	quotaSvc    *quota.Service
	quotaRepo   *quota.Repository
	transSvc    *translation.Service
	transRepo   *translation.Repository
	assessRepo  *assessment.Repository
}

// NewHandler 建立新的 admin Handler
func NewHandler(
	authRepo *auth.Repository,
	quotaSvc *quota.Service,
	quotaRepo *quota.Repository,
	transSvc *translation.Service,
	transRepo *translation.Repository,
	assessRepo *assessment.Repository,
) *Handler {
	return &Handler{
		authRepo:   authRepo,
		quotaSvc:   quotaSvc,
		quotaRepo:  quotaRepo,
		transSvc:   transSvc,
		transRepo:  transRepo,
		assessRepo: assessRepo,
	}
}

// ListUsers 處理 GET /api/admin/users?page=1&page_size=20
// 回傳使用者列表，包含配額資訊
func (h *Handler) ListUsers(c *gin.Context) {
	page, pageSize := parsePagination(c)
	offset := (page - 1) * pageSize

	users, total, err := h.authRepo.ListAll(c.Request.Context(), offset, pageSize)
	if err != nil {
		response.SendInternalError(c)
		return
	}

	type userItem struct {
		ID        uuid.UUID `json:"id"`
		Email     string    `json:"email"`
		Role      string    `json:"role"`
		UsedCount int       `json:"used_count"`
		Remaining int       `json:"remaining"`
	}

	items := make([]userItem, 0, len(users))
	for _, u := range users {
		info, err := h.quotaSvc.GetUserQuotaInfo(c.Request.Context(), u.ID)
		var usedCount, remaining int
		if err == nil && info != nil {
			usedCount = info.UsedCount
			remaining = info.Remaining
		}
		items = append(items, userItem{
			ID:        u.ID,
			Email:     u.Email,
			Role:      u.Role,
			UsedCount: usedCount,
			Remaining: remaining,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"users":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetQuotaSettings 處理 GET /api/admin/quota
// 回傳當前配額設定
func (h *Handler) GetQuotaSettings(c *gin.Context) {
	settings, err := h.quotaRepo.GetSettings(c.Request.Context())
	if err != nil {
		response.SendInternalError(c)
		return
	}

	c.JSON(http.StatusOK, settings)
}

// updateQuotaRequest 更新配額設定的請求結構
type updateQuotaRequest struct {
	MaxAssessments int    `json:"max_assessments"`
	ResetPeriod    string `json:"reset_period"`
}

// UpdateQuotaSettings 處理 PUT /api/admin/quota
// 更新配額設定
func (h *Handler) UpdateQuotaSettings(c *gin.Context) {
	var req updateQuotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendValidationError(c, "配額設定無效：max_assessments 必須為正整數，reset_period 必須為 daily 或 weekly")
		return
	}

	// 驗證
	if req.MaxAssessments < 1 {
		response.SendValidationError(c, "配額設定無效：max_assessments 必須為正整數，reset_period 必須為 daily 或 weekly")
		return
	}
	if req.ResetPeriod != "daily" && req.ResetPeriod != "weekly" {
		response.SendValidationError(c, "配額設定無效：max_assessments 必須為正整數，reset_period 必須為 daily 或 weekly")
		return
	}

	if err := h.quotaRepo.UpdateSettings(c.Request.Context(), req.MaxAssessments, req.ResetPeriod); err != nil {
		response.SendInternalError(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "配額設定已更新",
	})
}

// ListTranslations 處理 GET /api/admin/translations?locale=zh-TW&search=keyword&page=1&page_size=20
// 回傳翻譯列表（可搜尋、分頁）
func (h *Handler) ListTranslations(c *gin.Context) {
	locale := c.DefaultQuery("locale", "zh-TW")
	search := c.DefaultQuery("search", "")
	page, pageSize := parsePagination(c)

	translations, total, err := h.transSvc.Search(c.Request.Context(), locale, search, page, pageSize)
	if err != nil {
		response.SendInternalError(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"translations": translations,
		"total":        total,
		"page":         page,
		"page_size":    pageSize,
	})
}

// updateTranslationRequest 更新翻譯值的請求結構
type updateTranslationRequest struct {
	Value string `json:"value"`
}

// UpdateTranslation 處理 PUT /api/admin/translations/:id
// 更新翻譯內容
func (h *Handler) UpdateTranslation(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendValidationError(c, "無效的翻譯 ID")
		return
	}

	var req updateTranslationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendValidationError(c, "請提供翻譯值")
		return
	}

	if err := h.transSvc.Update(c.Request.Context(), id, req.Value); err != nil {
		if errors.Is(err, translation.ErrTranslationNotFound) {
			response.SendNotFoundError(c, "翻譯項目不存在")
			return
		}
		response.SendInternalError(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "翻譯已更新",
	})
}

// ListAssessments 處理 GET /api/admin/assessments?user_id=uuid&page=1&page_size=20
// user_id 為可選參數，若不提供則列出所有使用者的評估記錄
func (h *Handler) ListAssessments(c *gin.Context) {
	page, pageSize := parsePagination(c)
	offset := (page - 1) * pageSize

	userIDStr := c.Query("user_id")

	type assessmentItem struct {
		ID         uuid.UUID `json:"id"`
		Filename   string    `json:"filename"`
		TotalScore float64   `json:"total_score"`
		Status     string    `json:"status"`
		CreatedAt  string    `json:"created_at"`
	}

	var assessments []assessment.Assessment
	var total int
	var err error

	if userIDStr != "" {
		userID, parseErr := uuid.Parse(userIDStr)
		if parseErr != nil {
			response.SendValidationError(c, "無效的 user_id")
			return
		}
		assessments, total, err = h.assessRepo.ListByUserID(c.Request.Context(), userID, offset, pageSize)
	} else {
		assessments, total, err = h.assessRepo.ListAll(c.Request.Context(), offset, pageSize)
	}

	if err != nil {
		response.SendInternalError(c)
		return
	}

	items := make([]assessmentItem, 0, len(assessments))
	for _, a := range assessments {
		items = append(items, assessmentItem{
			ID:         a.ID,
			Filename:   a.Filename,
			TotalScore: a.TotalScore,
			Status:     a.Status,
			CreatedAt:  a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"assessments": items,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
	})
}

// parsePagination 從 query 參數解析分頁資訊
func parsePagination(c *gin.Context) (page, pageSize int) {
	page = 1
	pageSize = 20

	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 && v <= 100 {
			pageSize = v
		}
	}
	return
}
