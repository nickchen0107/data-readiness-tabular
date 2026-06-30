package settings

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/safe-ai/excel-brushing-tool/internal/assessment"
	"github.com/safe-ai/excel-brushing-tool/pkg/response"
)

// Handler handles HTTP requests for the settings endpoints
type Handler struct {
	pool *pgxpool.Pool
}

// NewHandler creates a new settings Handler
func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

// WeightsRequest is the request body for updating weights
type WeightsRequest struct {
	RowCompleteness    float64 `json:"row_completeness"`
	ColumnCompleteness float64 `json:"column_completeness"`
	FormatConsistency  float64 `json:"format_consistency"`
	DuplicateSimilar   float64 `json:"duplicate_similar"`
	TableStructure     float64 `json:"table_structure"`
	AIQueryReadiness   float64 `json:"ai_query_readiness"`
}

// WeightsResponse is the response for weights endpoints
type WeightsResponse struct {
	RowCompleteness    float64 `json:"row_completeness"`
	ColumnCompleteness float64 `json:"column_completeness"`
	FormatConsistency  float64 `json:"format_consistency"`
	DuplicateSimilar   float64 `json:"duplicate_similar"`
	TableStructure     float64 `json:"table_structure"`
	AIQueryReadiness   float64 `json:"ai_query_readiness"`
}

// GetWeights handles GET /api/settings/weights
// Returns the current assessment weights from system_settings
func (h *Handler) GetWeights(c *gin.Context) {
	weights, err := h.loadWeights(c.Request.Context())
	if err != nil {
		log.Printf("讀取權重設定失敗: %v", err)
		response.SendInternalError(c)
		return
	}

	c.JSON(http.StatusOK, WeightsResponse{
		RowCompleteness:    weights.RowCompleteness,
		ColumnCompleteness: weights.ColumnCompleteness,
		FormatConsistency:  weights.FormatConsistency,
		DuplicateSimilar:   weights.DuplicateSimilar,
		TableStructure:     weights.TableStructure,
		AIQueryReadiness:   weights.AIQueryReadiness,
	})
}

// UpdateWeights handles PUT /api/settings/weights
// Validates the sum = 100% (1.0) and persists the new weights
func (h *Handler) UpdateWeights(c *gin.Context) {
	// Get user ID from JWT context
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	var req WeightsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendValidationError(c, "請提供有效的權重設定")
		return
	}

	// Validate sum = 1.0 (100%)
	sum := req.RowCompleteness + req.ColumnCompleteness + req.FormatConsistency +
		req.DuplicateSimilar + req.TableStructure + req.AIQueryReadiness

	if math.Abs(sum-1.0) >= 0.001 {
		c.JSON(http.StatusBadRequest, &response.ErrorResponse{
			Error: response.ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "六項權重總和必須等於 100%",
			},
		})
		return
	}

	// Validate no negative weights
	if req.RowCompleteness < 0 || req.ColumnCompleteness < 0 || req.FormatConsistency < 0 ||
		req.DuplicateSimilar < 0 || req.TableStructure < 0 || req.AIQueryReadiness < 0 {
		response.SendValidationError(c, "權重數值不可為負數")
		return
	}

	// Persist to system_settings
	weights := assessment.Weights{
		RowCompleteness:    req.RowCompleteness,
		ColumnCompleteness: req.ColumnCompleteness,
		FormatConsistency:  req.FormatConsistency,
		DuplicateSimilar:   req.DuplicateSimilar,
		TableStructure:     req.TableStructure,
		AIQueryReadiness:   req.AIQueryReadiness,
	}

	if err := h.saveWeights(c.Request.Context(), weights, userID); err != nil {
		log.Printf("儲存權重設定失敗: %v", err)
		response.SendInternalError(c)
		return
	}

	c.JSON(http.StatusOK, WeightsResponse{
		RowCompleteness:    weights.RowCompleteness,
		ColumnCompleteness: weights.ColumnCompleteness,
		FormatConsistency:  weights.FormatConsistency,
		DuplicateSimilar:   weights.DuplicateSimilar,
		TableStructure:     weights.TableStructure,
		AIQueryReadiness:   weights.AIQueryReadiness,
	})
}

// loadWeights reads weights from system_settings table
func (h *Handler) loadWeights(ctx context.Context) (assessment.Weights, error) {
	var valueJSON []byte

	err := h.pool.QueryRow(ctx,
		`SELECT value FROM system_settings WHERE key = 'assessment_weights'`,
	).Scan(&valueJSON)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return assessment.DefaultWeights(), nil
		}
		return assessment.Weights{}, err
	}

	var w assessment.Weights
	if err := json.Unmarshal(valueJSON, &w); err != nil {
		return assessment.DefaultWeights(), nil
	}

	if !w.IsValid() {
		return assessment.DefaultWeights(), nil
	}

	return w, nil
}

// saveWeights persists weights to system_settings table using UPSERT
func (h *Handler) saveWeights(ctx context.Context, weights assessment.Weights, userID uuid.UUID) error {
	valueJSON, err := json.Marshal(weights)
	if err != nil {
		return err
	}

	_, err = h.pool.Exec(ctx,
		`INSERT INTO system_settings (key, value, updated_at, updated_by)
		 VALUES ('assessment_weights', $1, NOW(), $2)
		 ON CONFLICT (key) DO UPDATE SET value = $1, updated_at = NOW(), updated_by = $2`,
		valueJSON, userID,
	)
	return err
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
