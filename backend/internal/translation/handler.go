package translation

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// supportedLocales 支援的語系
var supportedLocales = map[string]bool{
	"zh-TW": true,
	"en":    true,
}

// Handler 處理翻譯公開 API
type Handler struct {
	service *Service
}

// NewHandler 建立新的 translation Handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetTranslations 公開端點 GET /api/translations/:locale
// 回傳指定語系的所有翻譯 key-value map
func (h *Handler) GetTranslations(c *gin.Context) {
	locale := c.Param("locale")

	if !supportedLocales[locale] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "不支援的語系，僅支援 zh-TW 與 en",
		})
		return
	}

	translations, err := h.service.GetByLocale(c.Request.Context(), locale)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "取得翻譯失敗",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"translations": translations,
	})
}
