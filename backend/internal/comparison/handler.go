package comparison

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/safe-ai/excel-brushing-tool/internal/cleaning"
	"github.com/safe-ai/excel-brushing-tool/pkg/response"
)

// ErrSessionNotFound is returned when a cleaning session does not exist
// or does not belong to the requesting user.
var ErrSessionNotFound = errors.New("session not found")

// Handler handles comparison API HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new comparison Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetComparison handles GET /api/compare/:id
// Returns the full before/after comparison data for a cleaning session.
func (h *Handler) GetComparison(c *gin.Context) {
	// Parse session ID from URL parameter
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendValidationError(c, "無效的 session ID")
		return
	}

	// Get user ID from JWT context
	userIDVal, exists := c.Get("user_id")
	if !exists {
		response.SendAuthError(c, "未提供認證令牌")
		return
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		response.SendAuthError(c, "無效的使用者身份")
		return
	}

	// Call service to get comparison data
	result, err := h.service.GetComparison(c.Request.Context(), sessionID, userID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) || errors.Is(err, cleaning.ErrSessionNotFound) {
			c.JSON(http.StatusNotFound, response.NewNotFoundError("梳理記錄不存在"))
			return
		}
		c.JSON(http.StatusInternalServerError, &response.ErrorResponse{
			Error: response.ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, result)
}
