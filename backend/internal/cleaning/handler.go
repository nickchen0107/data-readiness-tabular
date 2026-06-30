package cleaning

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/safe-ai/excel-brushing-tool/internal/assessment"
)

// Handler handles HTTP requests for cleaning operations
type Handler struct {
	service *Service
}

// NewHandler creates a new cleaning Handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// PreviewRemovals handles POST /api/clean/preview-removals
// Returns lists of columns/blocks that COULD be removed, letting the user choose
func (h *Handler) PreviewRemovals(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "未授權的請求",
			},
		})
		return
	}

	// Validate user_id (same pattern as other handlers)
	_, ok := userIDStr.(uuid.UUID)
	if !ok {
		idStr, ok := userIDStr.(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "無效的使用者 ID",
				},
			})
			return
		}
		if _, err := uuid.Parse(idStr); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "無效的使用者 ID",
				},
			})
			return
		}
	}

	var req struct {
		AssessmentID uuid.UUID `json:"assessment_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "請提供 assessment_id",
			},
		})
		return
	}

	result, err := h.service.PreviewRemovals(c.Request.Context(), req.AssessmentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "PREVIEW_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ApplyRules handles POST /api/clean/apply
func (h *Handler) ApplyRules(c *gin.Context) {
	// Get user ID from JWT context
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "未授權的請求",
			},
		})
		return
	}

	userID, ok := userIDStr.(uuid.UUID)
	if !ok {
		// Try parsing from string
		idStr, ok := userIDStr.(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "無效的使用者 ID",
				},
			})
			return
		}
		var err error
		userID, err = uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "無效的使用者 ID",
				},
			})
			return
		}
	}

	var req CleanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "請求格式錯誤：" + err.Error(),
			},
		})
		return
	}

	session, err := h.service.ApplyRules(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "CLEANING_ERROR",
				"message": "梳理作業失敗：" + err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id":   session.ID,
		"rows_before":  session.RowsBefore,
		"rows_after":   session.RowsAfter,
		"score_before": session.ScoreBefore,
		"score_after":  session.ScoreAfter,
	})
}

// GetLatest handles GET /api/clean/latest
func (h *Handler) GetLatest(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": "UNAUTHORIZED", "message": "未授權的請求"},
		})
		return
	}

	userID, ok := userIDStr.(uuid.UUID)
	if !ok {
		idStr, ok := userIDStr.(string)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{"code": "BAD_REQUEST", "message": "無效的使用者 ID"},
			})
			return
		}
		var err error
		userID, err = uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{"code": "BAD_REQUEST", "message": "無效的使用者 ID"},
			})
			return
		}
	}

	session, err := h.service.GetLatest(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{"code": "NOT_FOUND", "message": "沒有找到梳理記錄"},
		})
		return
	}

	c.JSON(http.StatusOK, session)
}

// GetPreview handles GET /api/clean/:id/preview
func (h *Handler) GetPreview(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "未授權的請求",
			},
		})
		return
	}

	userID, ok := userIDStr.(uuid.UUID)
	if !ok {
		idStr, ok := userIDStr.(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "無效的使用者 ID",
				},
			})
			return
		}
		var err error
		userID, err = uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "無效的使用者 ID",
				},
			})
			return
		}
	}

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "無效的 session ID",
			},
		})
		return
	}

	preview, err := h.service.GetPreview(c.Request.Context(), sessionID, userID)
	if err != nil {
		if err == ErrSessionNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"code":    "NOT_FOUND",
					"message": "梳理記錄不存在",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "PREVIEW_ERROR",
				"message": "無法取得預覽資料：" + err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, preview)
}

// GetLog handles GET /api/clean/:id/log
func (h *Handler) GetLog(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "未授權的請求",
			},
		})
		return
	}

	userID, ok := userIDStr.(uuid.UUID)
	if !ok {
		idStr, ok := userIDStr.(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "無效的使用者 ID",
				},
			})
			return
		}
		var err error
		userID, err = uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "無效的使用者 ID",
				},
			})
			return
		}
	}

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "無效的 session ID",
			},
		})
		return
	}

	logEntries, err := h.service.GetLog(c.Request.Context(), sessionID, userID)
	if err != nil {
		if err == ErrSessionNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"code":    "NOT_FOUND",
					"message": "梳理記錄不存在",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "LOG_ERROR",
				"message": "無法取得梳理日誌：" + err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"log": logEntries,
	})
}

// ApplyInteractiveFix handles POST /api/clean/interactive
func (h *Handler) ApplyInteractiveFix(c *gin.Context) {
	// Extract JWT user_id from context
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "未授權的請求",
			},
		})
		return
	}

	userID, ok := userIDStr.(uuid.UUID)
	if !ok {
		idStr, ok := userIDStr.(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "無效的使用者 ID",
				},
			})
			return
		}
		var err error
		userID, err = uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "無效的使用者 ID",
				},
			})
			return
		}
	}

	// Bind InteractiveFixRequest JSON body
	var req InteractiveFixRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "請求格式錯誤：" + err.Error(),
			},
		})
		return
	}

	// Validate: edits must not be empty
	if len(req.Edits) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "請提供至少一筆修正",
			},
		})
		return
	}

	// Validate: action values are valid, replace/header_rename must have non-empty value
	validActions := map[string]bool{
		"replace":       true,
		"keep":          true,
		"delete_row":    true,
		"remark_split":  true,
		"header_rename": true,
	}
	for _, edit := range req.Edits {
		if !validActions[edit.Action] {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":    "VALIDATION_ERROR",
					"message": "無效的操作類型：" + edit.Action,
				},
			})
			return
		}
		if (edit.Action == "replace" || edit.Action == "header_rename") && strings.TrimSpace(edit.Value) == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":    "VALIDATION_ERROR",
					"message": "該操作需要提供新值",
				},
			})
			return
		}
	}

	// Call service.ApplyInteractiveEdits
	resp, err := h.service.ApplyInteractiveEdits(c.Request.Context(), userID, req)
	if err != nil {
		// Check if assessment not found
		if err.Error() == assessment.ErrAssessmentNotFound.Error() ||
			strings.Contains(err.Error(), "評估記錄不存在") {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"code":    "NOT_FOUND",
					"message": "評估記錄不存在",
				},
			})
			return
		}

		// Check for data loading / validation errors
		if strings.Contains(err.Error(), "無法載入工作表資料") ||
			strings.Contains(err.Error(), "無法取得上傳記錄") {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":    "PROCESSING_ERROR",
					"message": "無法載入工作表資料",
				},
			})
			return
		}

		// Generic processing error
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "PROCESSING_ERROR",
				"message": "互動式修正處理失敗：" + err.Error(),
			},
		})
		return
	}

	// Return 200 with InteractiveFixResponse on success
	c.JSON(http.StatusOK, resp)
}
