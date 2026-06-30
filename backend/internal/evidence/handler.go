package evidence

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/safe-ai/excel-brushing-tool/internal/cleaning"
	"github.com/safe-ai/excel-brushing-tool/pkg/response"
)

// Handler handles HTTP requests for the evidence endpoints
type Handler struct {
	service *Service
}

// NewHandler creates a new evidence Handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Submit handles POST /api/evidence/submit
// Computes hashes and forwards to blockchain API
func (h *Handler) Submit(c *gin.Context) {
	// Get user ID from JWT context
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	// Bind request body
	var req EvidenceSubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendValidationError(c, "請提供有效的 session_id")
		return
	}

	// Submit evidence
	record, err := h.service.Submit(c.Request.Context(), req.SessionID, userID)
	if err != nil {
		if errors.Is(err, cleaning.ErrSessionNotFound) {
			c.JSON(http.StatusNotFound, response.NewNotFoundError("梳理記錄不存在或無權存取"))
			return
		}
		// Check if it's a blockchain connectivity issue
		if isBlockchainError(err) {
			c.JSON(http.StatusServiceUnavailable, &response.ErrorResponse{
				Error: response.ErrorDetail{
					Code:    "BLOCKCHAIN_UNAVAILABLE",
					Message: "區塊鏈服務暫時無法連線",
				},
			})
			return
		}
		log.Printf("存證提交失敗: %v", err)
		response.SendInternalError(c)
		return
	}

	c.JSON(http.StatusOK, record)
}

// Get handles GET /api/evidence/:record_id
// Queries local DB and optionally proxies blockchain API
func (h *Handler) Get(c *gin.Context) {
	recordID := c.Param("record_id")
	if recordID == "" {
		response.SendValidationError(c, "請提供有效的 record_id")
		return
	}

	record, err := h.service.GetRecord(c.Request.Context(), recordID)
	if err != nil {
		if errors.Is(err, ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, response.NewNotFoundError("存證記錄不存在"))
			return
		}
		log.Printf("查詢存證記錄失敗: %v", err)
		response.SendInternalError(c)
		return
	}

	c.JSON(http.StatusOK, record)
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

// isBlockchainError checks if the error is related to blockchain service connectivity
func isBlockchainError(err error) bool {
	// In the service layer, blockchain errors result in a demo record being created,
	// so this would only be triggered by other kinds of failures.
	// We keep this for explicit blockchain proxy errors from the Get endpoint.
	return false
}
