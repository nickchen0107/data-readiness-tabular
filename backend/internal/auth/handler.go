package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/safe-ai/excel-brushing-tool/pkg/response"
)

// Handler 處理 auth 相關的 HTTP 請求
type Handler struct {
	service     *Service
	blacklist   *TokenBlacklist
	rateLimiter *RateLimiter
}

// NewHandler 建立新的 auth Handler
func NewHandler(svc *Service) *Handler {
	return &Handler{service: svc}
}

// SetBlacklist 設定 token blacklist（由 main.go 注入）
func (h *Handler) SetBlacklist(bl *TokenBlacklist) {
	h.blacklist = bl
}

// SetRateLimiter 設定登入速率限制器（由 main.go 注入）
func (h *Handler) SetRateLimiter(rl *RateLimiter) {
	h.rateLimiter = rl
}

// Register 處理使用者註冊 POST /api/auth/register
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendValidationError(c, "請提供有效的帳號和密碼")
		return
	}

	user, err := h.service.Register(c.Request.Context(), req)
	if err != nil {
		h.handleRegisterError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         user.ID,
		"email":      user.Email,
		"created_at": user.CreatedAt,
	})
}

// Login 處理使用者登入 POST /api/auth/login
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendValidationError(c, "請提供有效的帳號和密碼")
		return
	}

	// 檢查速率限制
	if h.rateLimiter != nil {
		blocked, err := h.rateLimiter.IsBlocked(c.Request.Context(), req.Email)
		if err == nil && blocked {
			c.JSON(http.StatusTooManyRequests, response.ErrorResponse{
				Error: response.ErrorDetail{
					Code:    "RATE_LIMIT",
					Message: "帳號已暫時鎖定，請稍後再試",
				},
			})
			return
		}
	}

	tokenResp, err := h.service.Login(c.Request.Context(), req)
	if err != nil {
		// 記錄失敗嘗試
		if h.rateLimiter != nil {
			_ = h.rateLimiter.RecordAttempt(c.Request.Context(), req.Email, false)
		}
		h.handleLoginError(c, err)
		return
	}

	// 記錄成功嘗試
	if h.rateLimiter != nil {
		_ = h.rateLimiter.RecordAttempt(c.Request.Context(), req.Email, true)
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      tokenResp.Token,
		"expires_at": tokenResp.ExpiresAt,
	})
}

// Logout 處理使用者登出 POST /api/auth/logout
func (h *Handler) Logout(c *gin.Context) {
	// 從 context 取得 token（由 JWT middleware 設定）
	tokenVal, exists := c.Get("token")
	if !exists {
		response.SendAuthError(c, "未提供認證令牌")
		return
	}
	token, ok := tokenVal.(string)
	if !ok || token == "" {
		response.SendAuthError(c, "無效的認證令牌")
		return
	}

	// 將 token 加入黑名單（設定過期時間為 24 小時後，與 JWT 預設過期一致）
	if h.blacklist != nil {
		h.blacklist.Add(token, h.service.getTokenExpiry())
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "已成功登出",
	})
}

// GetMe 回傳當前使用者資訊 GET /api/auth/me
func (h *Handler) GetMe(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		response.SendAuthError(c, "未認證")
		return
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		response.SendAuthError(c, "無效的使用者身份")
		return
	}

	user, err := h.service.repo.GetByID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			response.SendNotFoundError(c, "使用者不存在")
			return
		}
		response.SendInternalError(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         user.ID,
		"email":      user.Email,
		"created_at": user.CreatedAt,
	})
}

// handleRegisterError 處理註冊錯誤對應的 HTTP 回應
func (h *Handler) handleRegisterError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidEmail):
		response.SendValidationError(c, err.Error())
	case errors.Is(err, ErrPasswordTooShort):
		response.SendValidationError(c, err.Error())
	case errors.Is(err, ErrPasswordTooLong):
		response.SendValidationError(c, err.Error())
	case errors.Is(err, ErrEmailAlreadyExists):
		c.JSON(http.StatusConflict, response.ErrorResponse{
			Error: response.ErrorDetail{
				Code:    "CONFLICT",
				Message: err.Error(),
			},
		})
	default:
		response.SendInternalError(c)
	}
}

// handleLoginError 處理登入錯誤對應的 HTTP 回應
func (h *Handler) handleLoginError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrUserNotFound), errors.Is(err, ErrInvalidCredentials):
		// 不揭露是帳號不存在還是密碼錯誤
		response.SendAuthError(c, "帳號或密碼錯誤")
	default:
		response.SendInternalError(c)
	}
}
