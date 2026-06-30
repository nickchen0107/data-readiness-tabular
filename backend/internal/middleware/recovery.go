package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/safe-ai/excel-brushing-tool/pkg/response"
)

// Recovery panic 恢復中介軟體
// 捕捉 panic 後回傳 HTTP 500 安全錯誤回應，並記錄堆疊追蹤
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 記錄堆疊追蹤供除錯用
				log.Printf("[PANIC RECOVERY] %v\n%s", err, debug.Stack())

				// 回傳安全的錯誤回應，不暴露內部細節
				c.AbortWithStatusJSON(http.StatusInternalServerError, response.NewInternalError())
			}
		}()

		c.Next()
	}
}
