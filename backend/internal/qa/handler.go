package qa

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/safe-ai/excel-brushing-tool/internal/cleaning"
	"github.com/safe-ai/excel-brushing-tool/pkg/response"
)

// Handler handles HTTP requests for the QA endpoints
type Handler struct {
	service *Service
}

// NewHandler creates a new QA Handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Ask handles POST /api/qa/ask
// Processes a question with consent check, guardrail, and dual Gemini calls
func (h *Handler) Ask(c *gin.Context) {
	// Get user ID from JWT context
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	// Bind request body
	var req QARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendValidationError(c, "請提供有效的 session_id 和 question")
		return
	}

	// Process question
	result, err := h.service.Ask(c.Request.Context(), req, userID)
	if err != nil {
		// Handle consent error
		if errors.Is(err, ErrConsentRequired) {
			c.JSON(http.StatusForbidden, &response.ErrorResponse{
				Error: response.ErrorDetail{
					Code:    "CONSENT_REQUIRED",
					Message: "請先同意資料保護聲明",
				},
			})
			return
		}

		// Handle data insufficiency error
		var dataErr *DataInsufficiencyError
		if errors.As(err, &dataErr) {
			c.JSON(http.StatusOK, gin.H{
				"original_answer": dataErr.Explanation,
				"cleaned_answer":  dataErr.Explanation,
				"data_insufficient": true,
			})
			return
		}

		// Handle session not found
		if errors.Is(err, cleaning.ErrSessionNotFound) {
			c.JSON(http.StatusNotFound, response.NewNotFoundError("梳理記錄不存在或無權存取"))
			return
		}

		log.Printf("QA 問答失敗: %v", err)
		response.SendInternalError(c)
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetSuggestions handles GET /api/qa/suggestions/:assess_id
// Returns 3 suggested questions based on the assessment's column names
func (h *Handler) GetSuggestions(c *gin.Context) {
	// Parse assessment ID
	assessIDStr := c.Param("assess_id")
	assessID, err := uuid.Parse(assessIDStr)
	if err != nil {
		response.SendValidationError(c, "無效的評估 ID")
		return
	}

	suggestions, err := h.service.GetSuggestions(c.Request.Context(), assessID)
	if err != nil {
		log.Printf("取得建議問題失敗: %v", err)
		response.SendInternalError(c)
		return
	}

	c.JSON(http.StatusOK, SuggestionsResponse{
		Suggestions: suggestions,
	})
}

// getUserID extracts the user ID from the JWT context
func (h *Handler) getUserID(c *gin.Context) (uuid.UUID, bool) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		response.SendAuthError(c, "未提供認證令牌")
		return uuid.Nil, false
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		response.SendAuthError(c, "無效的使用者身份")
		return uuid.Nil, false
	}
	return userID, true
}
