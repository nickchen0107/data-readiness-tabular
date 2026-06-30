package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger 請求日誌中介軟體
// 記錄每個請求的 HTTP 方法、路徑、狀態碼與處理時間
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 處理請求
		c.Next()

		// 計算處理時間
		duration := time.Since(start)

		log.Printf("[HTTP] %s %s | %d | %v",
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			duration,
		)
	}
}
