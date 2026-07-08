package export

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/safe-ai/excel-brushing-tool/internal/cleaning"
	"github.com/safe-ai/excel-brushing-tool/pkg/response"
)

// Handler handles HTTP requests for the export endpoints
type Handler struct {
	service *Service
}

// NewHandler creates a new export Handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GenerateExcelFilename derives the download filename based on the session's OriginalFilename.
// If OriginalFilename is non-empty, it strips the extension and appends "_refined.xlsx".
// If OriginalFilename is empty (legacy data), it falls back to "refined.xlsx".
func GenerateExcelFilename(originalFilename string) string {
	if originalFilename == "" {
		return "refined.xlsx"
	}
	base := filepath.Base(originalFilename)
	for _, ext := range []string{".xlsx", ".csv", ".xls", ".tsv"} {
		if strings.HasSuffix(strings.ToLower(base), ext) {
			base = strings.TrimSuffix(base, base[len(base)-len(ext):])
			break
		}
	}
	return base + "_refined.xlsx"
}

// GeneratePDFFilename derives the download filename for the PDF report.
// If OriginalFilename is non-empty, it strips the extension and appends "_report.pdf".
// If OriginalFilename is empty (legacy data), it falls back to "report.pdf".
func GeneratePDFFilename(originalFilename string) string {
	if originalFilename == "" {
		return "report.pdf"
	}
	base := filepath.Base(originalFilename)
	for _, ext := range []string{".xlsx", ".csv", ".xls", ".tsv"} {
		if strings.HasSuffix(strings.ToLower(base), ext) {
			base = strings.TrimSuffix(base, base[len(base)-len(ext):])
			break
		}
	}
	return base + "_report.pdf"
}

// GenerateLogFilename derives the download filename for the cleaning log.
// If OriginalFilename is non-empty, it strips the extension and appends "_cleaning.log".
// If OriginalFilename is empty (legacy data), it falls back to "cleaning.log".
func GenerateLogFilename(originalFilename string) string {
	if originalFilename == "" {
		return "cleaning.log"
	}
	base := filepath.Base(originalFilename)
	for _, ext := range []string{".xlsx", ".csv", ".xls", ".tsv"} {
		if strings.HasSuffix(strings.ToLower(base), ext) {
			base = strings.TrimSuffix(base, base[len(base)-len(ext):])
			break
		}
	}
	return base + "_cleaning.log"
}

// DownloadExcel handles GET /api/export/:id/xlsx
// Generates (if not cached) and serves the refined.xlsx file
func (h *Handler) DownloadExcel(c *gin.Context) {
	session, ok := h.getVerifiedSession(c)
	if !ok {
		return
	}

	filePath, err := h.service.GenerateExcelFile(c.Request.Context(), session)
	if err != nil {
		log.Printf("產生 Excel 失敗: %v", err)
		response.SendInternalError(c)
		return
	}

	// Dynamic filename based on original uploaded filename
	filename := GenerateExcelFilename(session.OriginalFilename)

	// RFC 5987 Content-Disposition: ASCII fallback + UTF-8 encoded filename
	disposition := fmt.Sprintf(`attachment; filename="refined.xlsx"; filename*=UTF-8''%s`, url.PathEscape(filename))
	c.Header("Content-Disposition", disposition)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.File(filePath)
}

// DownloadPDF handles GET /api/export/:id/pdf
// Generates and serves the report.pdf file
func (h *Handler) DownloadPDF(c *gin.Context) {
	session, ok := h.getVerifiedSession(c)
	if !ok {
		return
	}

	// Determine locale from query param (default: zh-TW)
	locale := c.DefaultQuery("locale", "zh-TW")

	filePath, err := h.service.GeneratePDFFile(c.Request.Context(), session, locale)
	if err != nil {
		log.Printf("產生 PDF 失敗: %v", err)
		response.SendInternalError(c)
		return
	}

	// Dynamic filename based on original uploaded filename
	filename := GeneratePDFFilename(session.OriginalFilename)
	disposition := fmt.Sprintf(`attachment; filename="report.pdf"; filename*=UTF-8''%s`, url.PathEscape(filename))
	c.Header("Content-Disposition", disposition)
	c.Header("Content-Type", "application/pdf")
	c.File(filePath)
}

// DownloadLog handles GET /api/export/:id/log
// Generates and serves the cleaning.log file
func (h *Handler) DownloadLog(c *gin.Context) {
	session, ok := h.getVerifiedSession(c)
	if !ok {
		return
	}

	// Determine locale from query param (default: zh-TW)
	locale := c.DefaultQuery("locale", "zh-TW")

	filePath, err := h.service.GenerateLogFile(c.Request.Context(), session, locale)
	if err != nil {
		log.Printf("產生清理日誌失敗: %v", err)
		response.SendInternalError(c)
		return
	}

	// Dynamic filename based on original uploaded filename
	filename := GenerateLogFilename(session.OriginalFilename)
	disposition := fmt.Sprintf(`attachment; filename="cleaning.log"; filename*=UTF-8''%s`, url.PathEscape(filename))
	c.Header("Content-Disposition", disposition)
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.File(filePath)
}

// getVerifiedSession extracts session ID from URL param, verifies user ownership,
// and returns the session. Returns false if any step fails (response already sent).
func (h *Handler) getVerifiedSession(c *gin.Context) (*cleaning.CleaningSession, bool) {
	// Parse session ID from URL
	idStr := c.Param("id")
	sessionID, err := uuid.Parse(idStr)
	if err != nil {
		response.SendValidationError(c, "無效的 session ID")
		return nil, false
	}

	// Get user ID from JWT context
	userIDVal, exists := c.Get("user_id")
	if !exists {
		response.SendAuthError(c, "未提供認證令牌")
		return nil, false
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		response.SendAuthError(c, "無效的使用者身份")
		return nil, false
	}

	// Verify ownership
	session, err := h.service.GetSessionWithOwnership(c.Request.Context(), sessionID, userID)
	if err != nil {
		if err == cleaning.ErrSessionNotFound {
			c.JSON(http.StatusNotFound, response.NewNotFoundError("梳理記錄不存在或無權存取"))
			return nil, false
		}
		log.Printf("取得梳理記錄失敗: %v", err)
		response.SendInternalError(c)
		return nil, false
	}

	return session, true
}


