package export

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-pdf/fpdf"

	"github.com/safe-ai/excel-brushing-tool/internal/assessment"
	"github.com/safe-ai/excel-brushing-tool/internal/cleaning"
	"github.com/safe-ai/excel-brushing-tool/pkg/config"
)

// PDFReportData aggregates all data needed for the PDF report
type PDFReportData struct {
	Session        *cleaning.CleaningSession
	Assessment     *assessment.Assessment   // Original assessment (Step 3)
	Issues         []assessment.Issue        // Original issues
	PostAssessment *assessment.Assessment    // Post-cleaning assessment (Step 5 result), may be nil
	Locale         string                    // "en" or "zh-TW"
}

// isEnglish returns true if locale is English
func (d *PDFReportData) isEnglish() bool {
	return d.Locale == "en"
}

// GeneratePDF creates a branded PDF report for a cleaning session.
// Returns the file path of the generated PDF.
func GeneratePDF(data *PDFReportData, cfg *config.Config, outputDir string) (string, error) {
	// Check font existence BEFORE creating PDF — font missing is an error
	if cfg.Report.FontPath == "" {
		return "", fmt.Errorf("中文字型檔案未設定，無法產生 PDF 報告")
	}
	if _, err := os.Stat(cfg.Report.FontPath); err != nil {
		return "", fmt.Errorf("中文字型檔案未安裝，無法產生 PDF 報告: %s", cfg.Report.FontPath)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("建立輸出目錄失敗: %w", err)
	}

	isEn := data.isEnglish()

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 15)

	// Parse brand colors from config
	primaryR, primaryG, primaryB := parseHexColor(cfg.Report.Colors.Primary)
	accentR, accentG, accentB := parseHexColor(cfg.Report.Colors.Accent)
	greenR, greenG, greenB := parseHexColor(cfg.Report.Colors.Green)

	// Load Chinese font — guaranteed to exist after the check above
	pdf.AddUTF8Font("NotoSansTC", "", cfg.Report.FontPath)
	if cfg.Report.FontBoldPath != "" {
		if _, err := os.Stat(cfg.Report.FontBoldPath); err == nil {
			pdf.AddUTF8Font("NotoSansTC", "B", cfg.Report.FontBoldPath)
		}
	}

	setFont := func(style string, size float64) {
		pdf.SetFont("NotoSansTC", style, size)
	}

	// ─── Page 1: Title Page ───
	pdf.AddPage()
	pdf.SetFillColor(int(primaryR), int(primaryG), int(primaryB))
	pdf.Rect(0, 0, 210, 297, "F")

	pdf.SetTextColor(255, 255, 255)
	setFont("B", 28)
	pdf.SetY(80)
	pdf.CellFormat(190, 15, "S.A.F.E.-AI", "", 1, "C", false, 0, "")
	pdf.Ln(5)
	setFont("B", 20)
	if isEn {
		pdf.CellFormat(190, 12, "Data Quality Report", "", 1, "C", false, 0, "")
	} else {
		pdf.CellFormat(190, 12, "\xe8\xb3\x87\xe6\x96\x99\xe6\xa2\xb3\xe7\x90\x86\xe5\xa0\xb1\xe5\x91\x8a", "", 1, "C", false, 0, "")
	}

	// Show original filename
	pdf.Ln(15)
	setFont("", 12)
	if data.Session.OriginalFilename != "" {
		if isEn {
			pdf.CellFormat(190, 8, "Source File: "+data.Session.OriginalFilename, "", 1, "C", false, 0, "")
		} else {
			pdf.CellFormat(190, 8, "\xe4\xbe\x86\xe6\xba\x90\xe6\xaa\x94\xe6\xa1\x88: "+data.Session.OriginalFilename, "", 1, "C", false, 0, "")
		}
	}

	pdf.Ln(8)
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	pdf.CellFormat(190, 8, "Generated: "+timestamp, "", 1, "C", false, 0, "")

	// ─── Page 2: Score Summary ───
	pdf.AddPage()
	pdf.SetFillColor(255, 255, 255)
	pdf.SetTextColor(int(primaryR), int(primaryG), int(primaryB))

	setFont("B", 16)
	if isEn {
		pdf.CellFormat(190, 10, "Score Summary", "", 1, "L", false, 0, "")
	} else {
		pdf.CellFormat(190, 10, "\xe8\xa9\x95\xe5\x88\x86\xe6\x91\x98\xe8\xa6\x81", "", 1, "L", false, 0, "")
	}
	pdf.Ln(10)

	// Large score number
	score := data.Assessment.TotalScore
	setFont("B", 48)

	// Color based on grade
	grade := data.Assessment.Status
	switch grade {
	case "ready":
		pdf.SetTextColor(int(greenR), int(greenG), int(greenB))
	case "conditional":
		pdf.SetTextColor(180, 83, 9) // amber
	default:
		pdf.SetTextColor(180, 35, 24) // rose
	}
	pdf.CellFormat(190, 25, fmt.Sprintf("%.1f", score), "", 1, "C", false, 0, "")

	// Grade badge
	pdf.Ln(5)
	setFont("B", 14)
	gradeLabel := getGradeLabel(grade, !isEn)
	pdf.CellFormat(190, 10, gradeLabel, "", 1, "C", false, 0, "")

	// ─── Section: Six Indicators Table ───
	pdf.Ln(15)
	pdf.SetTextColor(int(primaryR), int(primaryG), int(primaryB))
	setFont("B", 14)
	if isEn {
		pdf.CellFormat(190, 10, "Six Indicators", "", 1, "L", false, 0, "")
	} else {
		pdf.CellFormat(190, 10, "\xe5\x85\xad\xe9\xa0\x85\xe6\x8c\x87\xe6\xa8\x99", "", 1, "L", false, 0, "")
	}
	pdf.Ln(5)

	// Table header
	pdf.SetFillColor(int(accentR), int(accentG), int(accentB))
	pdf.SetTextColor(255, 255, 255)
	setFont("B", 10)
	colWidths := []float64{80, 40, 40}
	var headers []string
	if isEn {
		headers = []string{"Indicator", "Score", "Weight"}
	} else {
		headers = []string{"\xe6\x8c\x87\xe6\xa8\x99\xe5\x90\x8d\xe7\xa8\xb1", "\xe5\x88\x86\xe6\x95\xb8", "\xe6\xac\x8a\xe9\x87\x8d"}
	}
	for i, h := range headers {
		pdf.CellFormat(colWidths[i], 8, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	// Table rows
	pdf.SetFillColor(245, 245, 245)
	pdf.SetTextColor(int(primaryR), int(primaryG), int(primaryB))
	setFont("", 10)

	indicators := getIndicatorRows(data.Assessment, !isEn)
	for i, row := range indicators {
		fill := i%2 == 0
		for j, val := range row {
			pdf.CellFormat(colWidths[j], 7, val, "1", 0, "C", fill, 0, "")
		}
		pdf.Ln(-1)
	}

	// ─── Section: Problem Summary ───
	pdf.Ln(10)
	setFont("B", 14)
	if isEn {
		pdf.CellFormat(190, 10, "Issues Summary", "", 1, "L", false, 0, "")
	} else {
		pdf.CellFormat(190, 10, "\xe5\x95\x8f\xe9\xa1\x8c\xe6\x91\x98\xe8\xa6\x81", "", 1, "L", false, 0, "")
	}
	pdf.Ln(5)

	if len(data.Issues) > 0 {
		// Use multi-cell approach for long descriptions to avoid overflow
		pdf.SetFillColor(int(accentR), int(accentG), int(accentB))
		pdf.SetTextColor(255, 255, 255)
		setFont("B", 10)
		probWidths := []float64{25, 135}
		var probHeaders []string
		if isEn {
			probHeaders = []string{"Severity", "Description"}
		} else {
			probHeaders = []string{"\xe5\x9a\xb4\xe9\x87\x8d\xe5\xba\xa6", "\xe8\xaa\xaa\xe6\x98\x8e"}
		}
		for i, h := range probHeaders {
			pdf.CellFormat(probWidths[i], 8, h, "1", 0, "C", true, 0, "")
		}
		pdf.Ln(-1)

		pdf.SetTextColor(int(primaryR), int(primaryG), int(primaryB))
		setFont("", 9)
		for i, issue := range data.Issues {
			if i >= 20 {
				break
			}
			fill := i%2 == 0
			pdf.SetFillColor(245, 245, 245)

			// Use the English description if available and locale is English
			desc := issue.Description
			if isEn && issue.DescriptionEn != "" {
				desc = issue.DescriptionEn
			}

			// Calculate row height based on description length
			descTrunc := truncateString(desc, 100)
			lineHeight := float64(7)
			// Use MultiCell for long text
			x := pdf.GetX()
			y := pdf.GetY()
			pdf.CellFormat(probWidths[0], lineHeight, issue.Severity, "1", 0, "C", fill, 0, "")
			pdf.SetXY(x+probWidths[0], y)
			pdf.CellFormat(probWidths[1], lineHeight, descTrunc, "1", 0, "L", fill, 0, "")
			pdf.Ln(-1)
		}
	} else {
		setFont("", 10)
		if isEn {
			pdf.CellFormat(190, 8, "No issues found", "", 1, "L", false, 0, "")
		} else {
			pdf.CellFormat(190, 8, "\xe7\x84\xa1\xe5\x95\x8f\xe9\xa1\x8c", "", 1, "L", false, 0, "")
		}
	}

	// ─── Page 3: Before/After + Rules ───
	pdf.AddPage()
	pdf.SetTextColor(int(primaryR), int(primaryG), int(primaryB))
	setFont("B", 14)
	if isEn {
		pdf.CellFormat(190, 10, "Cleaning Operations (Step 5)", "", 1, "L", false, 0, "")
	} else {
		pdf.CellFormat(190, 10, "\xe6\xa2\xb3\xe7\x90\x86\xe6\x93\x8d\xe4\xbd\x9c (Step 5)", "", 1, "L", false, 0, "")
	}
	pdf.Ln(5)

	// ─── Section: Cleaning Rules Applied ───
	setFont("B", 12)
	if isEn {
		pdf.CellFormat(190, 8, "Rules Applied", "", 1, "L", false, 0, "")
	} else {
		pdf.CellFormat(190, 8, "\xe5\xb7\xb2\xe5\xa5\x97\xe7\x94\xa8\xe6\xa2\xb3\xe7\x90\x86\xe8\xa6\x8f\xe5\x89\x87", "", 1, "L", false, 0, "")
	}
	pdf.Ln(3)

	setFont("", 11)
	for i, rule := range data.Session.RulesApplied {
		ruleLabel := getRuleLabel(rule, !isEn)
		pdf.CellFormat(190, 7, fmt.Sprintf("%d. %s", i+1, ruleLabel), "", 1, "L", false, 0, "")
	}

	if len(data.Session.RulesApplied) == 0 {
		if isEn {
			pdf.CellFormat(190, 7, "No rules applied", "", 1, "L", false, 0, "")
		} else {
			pdf.CellFormat(190, 7, "\xe6\x9c\xaa\xe5\xa5\x97\xe7\x94\xa8\xe4\xbb\xbb\xe4\xbd\x95\xe8\xa6\x8f\xe5\x89\x87", "", 1, "L", false, 0, "")
		}
	}

	// Before/After comparison table
	pdf.Ln(10)
	setFont("B", 12)
	if isEn {
		pdf.CellFormat(190, 8, "Before / After Comparison", "", 1, "L", false, 0, "")
	} else {
		pdf.CellFormat(190, 8, "\xe6\xa2\xb3\xe7\x90\x86\xe5\x89\x8d\xe5\xbe\x8c\xe5\xb0\x8d\xe6\xaf\x94", "", 1, "L", false, 0, "")
	}
	pdf.Ln(3)

	setFont("B", 10)
	pdf.SetFillColor(int(accentR), int(accentG), int(accentB))
	pdf.SetTextColor(255, 255, 255)
	compWidths := []float64{60, 60, 60}
	var compHeaders []string
	if isEn {
		compHeaders = []string{"Metric", "Before", "After"}
	} else {
		compHeaders = []string{"\xe6\x8c\x87\xe6\xa8\x99", "\xe6\xa2\xb3\xe7\x90\x86\xe5\x89\x8d", "\xe6\xa2\xb3\xe7\x90\x86\xe5\xbe\x8c"}
	}
	for i, h := range compHeaders {
		pdf.CellFormat(compWidths[i], 8, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetTextColor(int(primaryR), int(primaryG), int(primaryB))
	setFont("", 10)
	pdf.SetFillColor(245, 245, 245)

	var rowsLabel, scoreLabel string
	if isEn {
		rowsLabel = "Data Rows"
		scoreLabel = "Score"
	} else {
		rowsLabel = "\xe8\xb3\x87\xe6\x96\x99\xe5\x88\x97\xe6\x95\xb8"
		scoreLabel = "\xe8\xa9\x95\xe5\x88\x86"
	}
	pdf.CellFormat(compWidths[0], 7, rowsLabel, "1", 0, "C", true, 0, "")
	pdf.CellFormat(compWidths[1], 7, strconv.Itoa(data.Session.RowsBefore), "1", 0, "C", true, 0, "")
	pdf.CellFormat(compWidths[2], 7, strconv.Itoa(data.Session.RowsAfter), "1", 0, "C", true, 0, "")
	pdf.Ln(-1)

	pdf.CellFormat(compWidths[0], 7, scoreLabel, "1", 0, "C", false, 0, "")
	pdf.CellFormat(compWidths[1], 7, fmt.Sprintf("%.1f", data.Session.ScoreBefore), "1", 0, "C", false, 0, "")
	pdf.CellFormat(compWidths[2], 7, fmt.Sprintf("%.1f", data.Session.ScoreAfter), "1", 0, "C", false, 0, "")
	pdf.Ln(-1)

	// ─── Page 4: Post-Cleaning Assessment ───
	if data.PostAssessment != nil {
		pdf.AddPage()
		pdf.SetTextColor(int(primaryR), int(primaryG), int(primaryB))
		setFont("B", 14)
		if isEn {
			pdf.CellFormat(190, 10, "Post-Cleaning Assessment", "", 1, "L", false, 0, "")
		} else {
			pdf.CellFormat(190, 10, "\xe6\xa2\xb3\xe7\x90\x86\xe5\xbe\x8c\xe8\xa9\x95\xe4\xbc\xb0\xe7\xb5\x90\xe6\x9e\x9c", "", 1, "L", false, 0, "")
		}
		pdf.Ln(8)

		// Post-clean score
		postScore := data.PostAssessment.TotalScore
		setFont("B", 36)
		postGrade := data.PostAssessment.Status
		switch postGrade {
		case "ready":
			pdf.SetTextColor(int(greenR), int(greenG), int(greenB))
		case "conditional":
			pdf.SetTextColor(180, 83, 9)
		default:
			pdf.SetTextColor(180, 35, 24)
		}
		pdf.CellFormat(190, 20, fmt.Sprintf("%.1f", postScore), "", 1, "C", false, 0, "")
		setFont("B", 12)
		pdf.CellFormat(190, 8, getGradeLabel(postGrade, !isEn), "", 1, "C", false, 0, "")
		pdf.Ln(10)

		// Post-clean six indicators
		pdf.SetTextColor(int(primaryR), int(primaryG), int(primaryB))
		setFont("B", 12)
		if isEn {
			pdf.CellFormat(190, 8, "Six Indicators (After Cleaning)", "", 1, "L", false, 0, "")
		} else {
			pdf.CellFormat(190, 8, "\xe5\x85\xad\xe9\xa0\x85\xe6\x8c\x87\xe6\xa8\x99\xef\xbc\x88\xe6\xa2\xb3\xe7\x90\x86\xe5\xbe\x8c\xef\xbc\x89", "", 1, "L", false, 0, "")
		}
		pdf.Ln(3)

		pdf.SetFillColor(int(accentR), int(accentG), int(accentB))
		pdf.SetTextColor(255, 255, 255)
		setFont("B", 10)
		for i, h := range headers {
			pdf.CellFormat(colWidths[i], 8, h, "1", 0, "C", true, 0, "")
		}
		pdf.Ln(-1)

		pdf.SetFillColor(245, 245, 245)
		pdf.SetTextColor(int(primaryR), int(primaryG), int(primaryB))
		setFont("", 10)

		postIndicators := getIndicatorRows(data.PostAssessment, !isEn)
		for i, row := range postIndicators {
			fill := i%2 == 0
			for j, val := range row {
				pdf.CellFormat(colWidths[j], 7, val, "1", 0, "C", fill, 0, "")
			}
			pdf.Ln(-1)
		}
	}

	// ─── Page 5: Remaining Issues (Manual Processing Required) ───
	if data.PostAssessment != nil && len(data.PostAssessment.Issues) > 0 {
		pdf.AddPage()
		pdf.SetTextColor(int(primaryR), int(primaryG), int(primaryB))
		setFont("B", 14)
		if isEn {
			pdf.CellFormat(190, 10, "Remaining Issues (Manual Processing Required)", "", 1, "L", false, 0, "")
		} else {
			pdf.CellFormat(190, 10, "\xe5\xbe\x85\xe6\x89\x8b\xe5\x8b\x95\xe8\x99\x95\xe7\x90\x86\xe7\x9a\x84\xe5\x95\x8f\xe9\xa1\x8c", "", 1, "L", false, 0, "")
		}
		pdf.Ln(3)
		setFont("", 10)
		if isEn {
			pdf.SetTextColor(100, 100, 100)
			pdf.CellFormat(190, 7, "These issues could not be resolved automatically and require manual intervention.", "", 1, "L", false, 0, "")
		} else {
			pdf.SetTextColor(100, 100, 100)
			pdf.CellFormat(190, 7, "\xe4\xbb\xa5\xe4\xb8\x8b\xe5\x95\x8f\xe9\xa1\x8c\xe7\x84\xa1\xe6\xb3\x95\xe9\x80\x8f\xe9\x81\x8e\xe8\x87\xaa\xe5\x8b\x95\xe5\x8c\x96\xe8\xa6\x8f\xe5\x89\x87\xe8\x99\x95\xe7\x90\x86\xef\xbc\x8c\xe9\x9c\x80\xe8\xa6\x81\xe4\xba\xba\xe5\xb7\xa5\xe4\xbb\x8b\xe5\x85\xa5\xe3\x80\x82", "", 1, "L", false, 0, "")
		}
		pdf.Ln(5)

		pdf.SetFillColor(int(accentR), int(accentG), int(accentB))
		pdf.SetTextColor(255, 255, 255)
		setFont("B", 10)
		remainWidths := []float64{25, 135}
		var remainHeaders []string
		if isEn {
			remainHeaders = []string{"Severity", "Description"}
		} else {
			remainHeaders = []string{"\xe5\x9a\xb4\xe9\x87\x8d\xe5\xba\xa6", "\xe8\xaa\xaa\xe6\x98\x8e"}
		}
		for i, h := range remainHeaders {
			pdf.CellFormat(remainWidths[i], 8, h, "1", 0, "C", true, 0, "")
		}
		pdf.Ln(-1)

		pdf.SetTextColor(int(primaryR), int(primaryG), int(primaryB))
		setFont("", 9)
		for i, issue := range data.PostAssessment.Issues {
			if i >= 20 {
				break
			}
			fill := i%2 == 0
			pdf.SetFillColor(245, 245, 245)
			desc := issue.Description
			if isEn && issue.DescriptionEn != "" {
				desc = issue.DescriptionEn
			}
			descTrunc := truncateString(desc, 100)
			pdf.CellFormat(remainWidths[0], 7, issue.Severity, "1", 0, "C", fill, 0, "")
			pdf.CellFormat(remainWidths[1], 7, descTrunc, "1", 0, "L", fill, 0, "")
			pdf.Ln(-1)
		}
	}

	// Save PDF with locale in filename for caching
	filename := fmt.Sprintf("report_%s_%s.pdf", data.Session.ID.String()[:8], data.Locale)
	filePath := filepath.Join(outputDir, filename)
	if err := pdf.OutputFileAndClose(filePath); err != nil {
		return "", fmt.Errorf("儲存 PDF 檔案失敗: %w", err)
	}

	return filePath, nil
}

// parseHexColor converts a hex color string like "#1a1f2e" to RGB values
func parseHexColor(hex string) (uint8, uint8, uint8) {
	if len(hex) == 7 && hex[0] == '#' {
		r, _ := strconv.ParseUint(hex[1:3], 16, 8)
		g, _ := strconv.ParseUint(hex[3:5], 16, 8)
		b, _ := strconv.ParseUint(hex[5:7], 16, 8)
		return uint8(r), uint8(g), uint8(b)
	}
	// Default to dark if parsing fails
	return 26, 31, 46
}

// getGradeLabel returns the display label for a grade
func getGradeLabel(grade string, hasChinese bool) string {
	if hasChinese {
		switch grade {
		case "ready":
			return "Ready (\xe5\xb7\xb2\xe5\xb0\xb1\xe7\xb7\x92)"
		case "conditional":
			return "Conditional (\xe6\x9c\x89\xe6\xa2\x9d\xe4\xbb\xb6\xe5\xb0\xb1\xe7\xb7\x92)"
		default:
			return "Not Ready (\xe6\x9c\xaa\xe5\xb0\xb1\xe7\xb7\x92)"
		}
	}
	switch grade {
	case "ready":
		return "Ready"
	case "conditional":
		return "Conditional"
	default:
		return "Not Ready"
	}
}

// getIndicatorRows returns the indicator table data
func getIndicatorRows(a *assessment.Assessment, hasChinese bool) [][]string {
	type indicator struct {
		name   string
		score  float64
		weight float64
	}

	indicators := []indicator{
		{"Row Completeness", a.RowCompleteness, a.WeightsSnapshot.RowCompleteness},
		{"Column Completeness", a.ColumnCompleteness, a.WeightsSnapshot.ColumnCompleteness},
		{"Format Consistency", a.FormatConsistency, a.WeightsSnapshot.FormatConsistency},
		{"Duplicate/Similar", a.DuplicateSimilar, a.WeightsSnapshot.DuplicateSimilar},
		{"Table Structure", a.TableStructure, a.WeightsSnapshot.TableStructure},
		{"AI Query Readiness", a.AIQueryReadiness, a.WeightsSnapshot.AIQueryReadiness},
	}

	if hasChinese {
		indicators[0].name = "\xe5\x88\x97\xe5\xae\x8c\xe6\x95\xb4\xe5\xba\xa6"
		indicators[1].name = "\xe6\xac\x84\xe5\xae\x8c\xe6\x95\xb4\xe5\xba\xa6"
		indicators[2].name = "\xe6\xa0\xbc\xe5\xbc\x8f\xe4\xb8\x80\xe8\x87\xb4\xe6\x80\xa7"
		indicators[3].name = "\xe9\x87\x8d\xe8\xa4\x87/\xe7\x9b\xb8\xe4\xbc\xbc"
		indicators[4].name = "\xe8\xa1\xa8\xe6\xa0\xbc\xe7\xb5\x90\xe6\xa7\x8b"
		indicators[5].name = "AI\xe6\x9f\xa5\xe8\xa9\xa2\xe6\xba\x96\xe5\x82\x99\xe5\xba\xa6"
	}

	var rows [][]string
	for _, ind := range indicators {
		rows = append(rows, []string{
			ind.name,
			fmt.Sprintf("%.1f", ind.score),
			fmt.Sprintf("%.0f%%", ind.weight*100),
		})
	}
	return rows
}

// getRuleLabel returns a human-readable label for a cleaning rule
func getRuleLabel(rule string, hasChinese bool) string {
	if hasChinese {
		switch rule {
		case "date_normalize":
			return "\xe7\xb5\xb1\xe4\xb8\x80\xe6\x97\xa5\xe6\x9c\x9f\xe6\xa0\xbc\xe5\xbc\x8f (date_normalize)"
		case "dedup":
			return "\xe7\xa7\xbb\xe9\x99\xa4\xe9\x87\x8d\xe8\xa4\x87\xe5\x88\x97 (dedup)"
		case "name_normalize":
			return "\xe5\xae\xa2\xe6\x88\xb6\xe5\x90\x8d\xe6\xad\xa3\xe8\xa6\x8f\xe5\x8c\x96 (name_normalize)"
		case "subtotal_remove":
			return "\xe7\xa7\xbb\xe9\x99\xa4\xe5\xb0\x8f\xe8\xa8\x88\xe5\x88\x97 (subtotal_remove)"
		case "fill_na":
			return "\xe5\xa1\xab\xe8\xa3\x9c\xe7\xa9\xba\xe5\x80\xbc (fill_na)"
		case "empty_col_remove":
			return "\xe7\xa7\xbb\xe9\x99\xa4\xe7\xa9\xba\xe7\x99\xbd\xe6\xac\x84\xe4\xbd\x8d (empty_col_remove)"
		case "keep_block":
			return "\xe4\xbf\x9d\xe7\x95\x99\xe8\xb3\x87\xe6\x96\x99\xe5\x8d\x80\xe5\xa1\x8a (keep_block)"
		default:
			return rule
		}
	}
	switch rule {
	case "date_normalize":
		return "Date Normalization"
	case "dedup":
		return "Remove Duplicates"
	case "name_normalize":
		return "Name Normalization"
	case "subtotal_remove":
		return "Remove Subtotal Rows"
	case "fill_na":
		return "Fill Empty Values"
	case "empty_col_remove":
		return "Remove Empty Columns"
	case "keep_block":
		return "Keep Data Block"
	default:
		return rule
	}
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}
