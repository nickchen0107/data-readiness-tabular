package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/safe-ai/excel-brushing-tool/pkg/response"
)

// AdminAuth 管理員權限中介軟體
// 檢查 gin context 中的 user_role 是否為 "admin"，若不是則回傳 403。
// 必須放在 JWTAuth 中介軟體之後使用。
func AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("user_role")
		if !exists || role != "admin" {
			c.JSON(http.StatusForbidden, &response.ErrorResponse{
				Error: response.ErrorDetail{
					Code:    "FORBIDDEN",
					Message: "權限不足",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
