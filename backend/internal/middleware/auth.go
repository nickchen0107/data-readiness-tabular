package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/safe-ai/excel-brushing-tool/internal/auth"
	"github.com/safe-ai/excel-brushing-tool/pkg/response"
)

// JWTAuth JWT 認證中介軟體
// 驗證 Authorization header 中的 Bearer token，檢查 blacklist，並將 user_id 設定至 context
func JWTAuth(secret string, blacklist *auth.TokenBlacklist) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 取得 Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.SendAuthError(c, "未提供認證令牌")
			c.Abort()
			return
		}

		// 2. 解析 Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.SendAuthError(c, "無效的認證令牌")
			c.Abort()
			return
		}
		tokenString := parts[1]

		// 3. 檢查 token 是否在黑名單中
		if blacklist.IsBlacklisted(tokenString) {
			response.SendAuthError(c, "token 已失效")
			c.Abort()
			return
		}

		// 4. 驗證 token 並取得 userID
		userID, err := auth.ValidateToken(tokenString, secret)
		if err != nil {
			switch err {
			case auth.ErrExpiredToken:
				c.JSON(http.StatusUnauthorized, response.NewAuthError("token 已過期"))
				c.Abort()
				return
			default:
				c.JSON(http.StatusUnauthorized, response.NewAuthError("無效的認證令牌"))
				c.Abort()
				return
			}
		}

		// 5. 設定 user_id 與 raw token 至 context
		c.Set("user_id", userID)
		c.Set("token", tokenString)

		// 6. 繼續處理
		c.Next()
	}
}
