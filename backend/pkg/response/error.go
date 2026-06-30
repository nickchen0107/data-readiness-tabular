package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse 統一的 API 錯誤回應結構
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail 錯誤詳細資訊
type ErrorDetail struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// NewValidationError 建立驗證錯誤回應 (HTTP 400)
func NewValidationError(message string) *ErrorResponse {
	return &ErrorResponse{
		Error: ErrorDetail{
			Code:    "VALIDATION_ERROR",
			Message: message,
		},
	}
}

// NewAuthError 建立認證錯誤回應 (HTTP 401)
func NewAuthError(message string) *ErrorResponse {
	return &ErrorResponse{
		Error: ErrorDetail{
			Code:    "AUTH_ERROR",
			Message: message,
		},
	}
}

// NewNotFoundError 建立資源不存在錯誤回應 (HTTP 404)
func NewNotFoundError(message string) *ErrorResponse {
	return &ErrorResponse{
		Error: ErrorDetail{
			Code:    "NOT_FOUND",
			Message: message,
		},
	}
}

// NewInternalError 建立內部錯誤回應 (HTTP 500)
// 永不暴露內部細節，僅回傳通用訊息
func NewInternalError() *ErrorResponse {
	return &ErrorResponse{
		Error: ErrorDetail{
			Code:    "INTERNAL_ERROR",
			Message: "系統內部錯誤，請稍後重試",
		},
	}
}

// SendValidationError 回傳 HTTP 400 驗證錯誤
func SendValidationError(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, NewValidationError(message))
}

// SendAuthError 回傳 HTTP 401 認證錯誤
func SendAuthError(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, NewAuthError(message))
}

// SendNotFoundError 回傳 HTTP 404 資源不存在錯誤
func SendNotFoundError(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, NewNotFoundError(message))
}

// SendInternalError 回傳 HTTP 500 內部錯誤
func SendInternalError(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, NewInternalError())
}
