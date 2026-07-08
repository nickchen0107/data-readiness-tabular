package export

import (
	"fmt"
	"html"
	"math"
	"strings"

	"github.com/safe-ai/excel-brushing-tool/internal/assessment"
)

// dynamicTranslations maps Chinese dynamic strings to English
var dynamicTranslations = map[string]string{
	"小計/合計列":          "Subtotal/Total Rows",
	"合併儲存格":           "Merged Cells",
	"多表格混在同一 sheet":    "Multiple Tables in One Sheet",
	"儲存格含換行":          "Cells Contain Newlines",
	"孤立合計列":           "Orphan Total Row",
	"(空白)":            "(blank)",
	"表格結構":            "Table Structure",
	"含有合併儲存格或小計列等":    "Contains merged cells or subtotal rows",
	"待改善項目":           "Items to Improve",
	"目前狀態":            "Current Status",
	"欄位名稱":            "Column Names",
	"部分欄位名稱為空或過短":     "Some column names are empty or too short",
	"每欄資料量":           "Column Fill Rate",
	"格式一致性":           "Format Consistency",
	"部分欄位格式混合使用":      "Mixed formats in some columns",
	"資料唯一性":           "Data Uniqueness",
	"存在相似或重複的資料":      "Similar or duplicate data exists",
	"（空白列）":           "(blank row)",
	"列受影響":            "rows affected",
	"列":              "rows",
	"組":              "groups",
	"處":              "issues",
	"欄":              "columns",
}

// translateDynamic translates Chinese dynamic strings to English when isEn=true
func translateDynamic(text string, isEn bool) string {
	if !isEn || text == "" {
		return text
	}
	if translated, ok := dynamicTranslations[text]; ok {
		return translated
	}
	return text
}

// buildReportHTML generates a self-contained HTML report page
func buildReportHTML(data *PDFReportData, isEn bool) string {
	var sb strings.Builder

	sb.WriteString(`<!DOCTYPE html><html><head><meta charset="utf-8">`)
	sb.WriteString(`<style>`)
	sb.WriteString(reportCSS())
	sb.WriteString(`</style></head><body>`)

	// Cover
	sb.WriteString(`<div class="cover">`)
	sb.WriteString(`<h1>SAFE-AI</h1>`)
	if isEn {
		sb.WriteString(`<h2>Data Quality Report</h2>`)
	} else {
		sb.WriteString(`<h2>資料梳理報告</h2>`)
	}
	if data.Session.OriginalFilename != "" {
		label := "Source File"
		if !isEn {
			label = "來源檔案"
		}
		sb.WriteString(fmt.Sprintf(`<p class="meta">%s: %s</p>`, label, html.EscapeString(data.Session.OriginalFilename)))
	}
	sb.WriteString(fmt.Sprintf(`<p class="meta">Generated: %s</p>`, data.Session.CreatedAt.Format("2006-01-02 15:04:05")))
	sb.WriteString(`</div>`)

	// Section 1: Original Assessment
	sb.WriteString(`<div class="page-break"></div>`)
	sectionTitle := "1. Original Assessment"
	if !isEn {
		sectionTitle = "1. 原始評估"
	}
	sb.WriteString(fmt.Sprintf(`<h2 class="section-title">%s</h2>`, sectionTitle))
	writeScoreBlock(&sb, data.Assessment.TotalScore, data.Assessment.Status, isEn)
	writeIndicatorTable(&sb, data.Assessment, isEn)
	// Radar chart for original assessment (single layer)
	if data.PostAssessment != nil {
		writeRadarChart(&sb, data.Assessment, data.PostAssessment, isEn)
	} else {
		writeRadarChartSingle(&sb, data.Assessment, isEn)
	}
	issuesTitle := "Issues Detected"
	if !isEn {
		issuesTitle = "偵測到的問題"
	}
	writeIssuesSection(&sb, data.Issues, isEn, issuesTitle)

	// Section 2: Cleaning Operations
	sb.WriteString(`<div class="page-break"></div>`)
	sectionTitle = "2. Cleaning Operations"
	if !isEn {
		sectionTitle = "2. 梳理操作"
	}
	sb.WriteString(fmt.Sprintf(`<h2 class="section-title">%s</h2>`, sectionTitle))
	writeRulesSection(&sb, data.Session.RulesApplied, isEn)
	writeComparisonTable(&sb, data.Session.RowsBefore, data.Session.RowsAfter, data.Session.ScoreBefore, data.Session.ScoreAfter, isEn)

	// Section 3: Post-Cleaning Assessment
	if data.PostAssessment != nil {
		sb.WriteString(`<div class="page-break"></div>`)
		sectionTitle = "3. Post-Cleaning Assessment"
		if !isEn {
			sectionTitle = "3. 梳理後評估"
		}
		sb.WriteString(fmt.Sprintf(`<h2 class="section-title">%s</h2>`, sectionTitle))
		writeScoreBlock(&sb, data.PostAssessment.TotalScore, data.PostAssessment.Status, isEn)

		// Indicator comparison table with delta
		writeIndicatorComparisonTable(&sb, data.Assessment, data.PostAssessment, isEn)

		// Progress bars with before/after overlay
		writeProgressBars(&sb, data.Assessment, data.PostAssessment, isEn)

		// SVG Radar chart
		writeRadarChart(&sb, data.Assessment, data.PostAssessment, isEn)
	}

	// Section 4: Remaining Issues
	if data.PostAssessment != nil && len(data.PostAssessment.Issues) > 0 {
		sb.WriteString(`<div class="page-break"></div>`)
		sectionTitle = "4. Remaining Issues (Manual Processing Required)"
		if !isEn {
			sectionTitle = "4. 待手動處理的問題"
		}
		sb.WriteString(fmt.Sprintf(`<h2 class="section-title">%s</h2>`, sectionTitle))
		note := "These issues could not be resolved automatically and require manual intervention."
		if !isEn {
			note = "以下問題無法透過自動化規則處理，需要人工介入。"
		}
		sb.WriteString(fmt.Sprintf(`<p class="note">%s</p>`, note))
		writeIssuesSection(&sb, data.PostAssessment.Issues, isEn, "")
	}

	sb.WriteString(`</body></html>`)
	return sb.String()
}

func writeScoreBlock(sb *strings.Builder, score float64, status string, isEn bool) {
	color := getGradeColorHex(status)
	label := getGradeLabel(status, !isEn)
	sb.WriteString(fmt.Sprintf(`<div class="score-block"><span class="score" style="color:%s">%.1f</span>`, color, score))
	sb.WriteString(fmt.Sprintf(`<span class="grade" style="color:%s">%s</span></div>`, color, label))
}

func writeIndicatorTable(sb *strings.Builder, a *assessment.Assessment, isEn bool) {
	type ind struct {
		name  string
		score float64
	}
	indicators := []ind{
		{"Row Completeness", a.RowCompleteness},
		{"Column Completeness", a.ColumnCompleteness},
		{"Format Consistency", a.FormatConsistency},
		{"Duplicate/Similar", a.DuplicateSimilar},
		{"Table Structure", a.TableStructure},
		{"AI Query Readiness", a.AIQueryReadiness},
	}
	if !isEn {
		indicators[0].name = "列完整度"
		indicators[1].name = "欄完整度"
		indicators[2].name = "格式一致性"
		indicators[3].name = "重複/相似"
		indicators[4].name = "表格結構"
		indicators[5].name = "AI查詢準備度"
	}

	hdr1, hdr2 := "Indicator", "Score"
	if !isEn {
		hdr1, hdr2 = "指標", "分數"
	}

	sb.WriteString(`<table class="ind-table"><thead><tr>`)
	sb.WriteString(fmt.Sprintf(`<th>%s</th><th style="width:200px">%s</th><th style="width:50px">%s</th>`, hdr1, "", hdr2))
	sb.WriteString(`</tr></thead><tbody>`)
	for i, ind := range indicators {
		cls := ""
		if i%2 == 0 {
			cls = ` class="alt"`
		}
		barColor := "#2b6cb0"
		if ind.score >= 80 {
			barColor = "#15803d"
		} else if ind.score < 60 {
			barColor = "#b42318"
		}
		bar := fmt.Sprintf(`<div style="background:#e5e7eb;border-radius:3px;height:8px;width:100%%"><div style="background:%s;border-radius:3px;height:8px;width:%.0f%%"></div></div>`, barColor, ind.score)
		sb.WriteString(fmt.Sprintf(`<tr%s><td class="left">%s</td><td>%s</td><td>%.1f</td></tr>`, cls, ind.name, bar, ind.score))
	}
	sb.WriteString(`</tbody></table>`)
}

func writeRulesSection(sb *strings.Builder, rules []string, isEn bool) {
	title := "Rules Applied"
	if !isEn {
		title = "已套用規則"
	}
	sb.WriteString(fmt.Sprintf(`<h3>%s</h3>`, title))
	if len(rules) == 0 {
		noRules := "No rules applied"
		if !isEn {
			noRules = "未套用任何規則"
		}
		sb.WriteString(fmt.Sprintf(`<p class="note">%s</p>`, noRules))
		return
	}
	sb.WriteString(`<ol class="rules">`)
	for _, rule := range rules {
		sb.WriteString(fmt.Sprintf(`<li>%s</li>`, getRuleLabel(rule, !isEn)))
	}
	sb.WriteString(`</ol>`)
}

func writeComparisonTable(sb *strings.Builder, rowsBefore, rowsAfter int, scoreBefore, scoreAfter float64, isEn bool) {
	h1, h2, h3 := "Metric", "Before", "After"
	r1, r2 := "Data Rows", "Score"
	if !isEn {
		h1, h2, h3 = "指標", "梳理前", "梳理後"
		r1, r2 = "資料列數", "評分"
	}
	sb.WriteString(`<table class="comp-table"><thead><tr>`)
	sb.WriteString(fmt.Sprintf(`<th>%s</th><th>%s</th><th>%s</th>`, h1, h2, h3))
	sb.WriteString(`</tr></thead><tbody>`)
	sb.WriteString(fmt.Sprintf(`<tr><td>%s</td><td>%d</td><td>%d</td></tr>`, r1, rowsBefore, rowsAfter))
	sb.WriteString(fmt.Sprintf(`<tr class="alt"><td>%s</td><td>%.1f</td><td class="green">%.1f</td></tr>`, r2, scoreBefore, scoreAfter))
	sb.WriteString(`</tbody></table>`)
}

func writeIssuesSection(sb *strings.Builder, issues []assessment.Issue, isEn bool, title string) {
	if title != "" {
		sb.WriteString(fmt.Sprintf(`<h3>%s</h3>`, title))
	}
	if len(issues) == 0 {
		noIssues := "No issues found."
		if !isEn {
			noIssues = "無問題。"
		}
		sb.WriteString(fmt.Sprintf(`<p class="note">%s</p>`, noIssues))
		return
	}

	for _, issue := range issues {
		sb.WriteString(`<div class="issue-card">`)
		// Header
		sevClass := "sev-low"
		if issue.Severity == "High" {
			sevClass = "sev-high"
		} else if issue.Severity == "Medium" {
			sevClass = "sev-med"
		}
		issueTitle := issue.Title
		if isEn && issue.TitleEn != "" {
			issueTitle = issue.TitleEn
		}
		sb.WriteString(fmt.Sprintf(`<div class="issue-header"><span class="%s">%s</span>`, sevClass, issue.Severity))
		sb.WriteString(fmt.Sprintf(`<span class="issue-title">%s</span>`, html.EscapeString(issueTitle)))
		rowsLabel := "rows"
		if !isEn {
			rowsLabel = "列"
		}
		sb.WriteString(fmt.Sprintf(`<span class="affected">%d %s</span></div>`, issue.AffectedRows, rowsLabel))

		// Description
		desc := issue.Description
		if isEn && issue.DescriptionEn != "" {
			desc = issue.DescriptionEn
		} else if isEn {
			desc = translateDynamic(desc, true)
		}
		if desc != "" {
			sb.WriteString(fmt.Sprintf(`<div class="issue-desc">%s</div>`, html.EscapeString(desc)))
		}

		// Examples (Excel-style tables)
		if len(issue.Examples) > 0 {
			writeExamples(sb, issue.Examples, isEn)
		}

		sb.WriteString(`</div>`)
	}
}

func writeExamples(sb *strings.Builder, examples []assessment.IssueExample, isEn bool) {
	// Group by label
	type group struct {
		label string
		items []assessment.IssueExample
	}
	var groups []group
	groupMap := map[string]int{}
	for _, ex := range examples {
		label := ex.Label
		if label == "" {
			label = "__default"
		}
		if idx, ok := groupMap[label]; ok {
			groups[idx].items = append(groups[idx].items, ex)
		} else {
			groupMap[label] = len(groups)
			groups = append(groups, group{label: label, items: []assessment.IssueExample{ex}})
		}
	}

	// Render max 3 groups
	maxGroups := 3
	if len(groups) < maxGroups {
		maxGroups = len(groups)
	}

	for g := 0; g < maxGroups; g++ {
		grp := groups[g]
		if grp.label != "__default" {
			translatedLabel := translateDynamic(grp.label, isEn)
			sb.WriteString(fmt.Sprintf(`<div class="group-label">%s</div>`, html.EscapeString(translatedLabel)))
		}

		sb.WriteString(`<table class="excel-table"><thead><tr><th class="row-num">#</th>`)
		if len(grp.items) > 0 && len(grp.items[0].Headers) > 0 {
			for _, h := range grp.items[0].Headers {
				translatedH := translateDynamic(h, isEn)
				sb.WriteString(fmt.Sprintf(`<th>%s</th>`, html.EscapeString(translatedH)))
			}
		}
		sb.WriteString(`</tr></thead><tbody>`)

		maxRows := 5
		if len(grp.items) < maxRows {
			maxRows = len(grp.items)
		}
		for i := 0; i < maxRows; i++ {
			ex := grp.items[i]
			sb.WriteString(`<tr>`)
			sb.WriteString(fmt.Sprintf(`<td class="row-num">%d</td>`, ex.RowNumber))
			for colIdx, cell := range ex.Cells {
				isHL := false
				for _, h := range ex.Highlights {
					if h == colIdx {
						isHL = true
						break
					}
				}
				cls := ""
				if isHL {
					cls = ` class="hl"`
				}
				cellVal := translateDynamic(cell, isEn)
				if cellVal == "" {
					cellVal = "—"
				}
				sb.WriteString(fmt.Sprintf(`<td%s>%s</td>`, cls, html.EscapeString(cellVal)))
			}
			sb.WriteString(`</tr>`)
		}
		sb.WriteString(`</tbody></table>`)
	}
}

func getGradeColorHex(grade string) string {
	switch grade {
	case "ready":
		return "#15803d"
	case "conditional":
		return "#b45309"
	default:
		return "#b42318"
	}
}

func writeIndicatorComparisonTable(sb *strings.Builder, before, after *assessment.Assessment, isEn bool) {
	type ind struct {
		name        string
		scoreBefore float64
		scoreAfter  float64
	}
	indicators := []ind{
		{"Row Completeness", before.RowCompleteness, after.RowCompleteness},
		{"Column Completeness", before.ColumnCompleteness, after.ColumnCompleteness},
		{"Format Consistency", before.FormatConsistency, after.FormatConsistency},
		{"Duplicate/Similar", before.DuplicateSimilar, after.DuplicateSimilar},
		{"Table Structure", before.TableStructure, after.TableStructure},
		{"AI Query Readiness", before.AIQueryReadiness, after.AIQueryReadiness},
	}
	if !isEn {
		indicators[0].name = "列完整度"
		indicators[1].name = "欄完整度"
		indicators[2].name = "格式一致性"
		indicators[3].name = "重複/相似"
		indicators[4].name = "表格結構"
		indicators[5].name = "AI查詢準備度"
	}

	h1, h2, h3, h4 := "Indicator", "Before", "After", "Change"
	if !isEn {
		h1, h2, h3, h4 = "指標", "梳理前", "梳理後", "改善"
	}

	sb.WriteString(`<table class="ind-table"><thead><tr>`)
	sb.WriteString(fmt.Sprintf(`<th>%s</th><th>%s</th><th>%s</th><th>%s</th>`, h1, h2, h3, h4))
	sb.WriteString(`</tr></thead><tbody>`)
	for i, ind := range indicators {
		cls := ""
		if i%2 == 0 {
			cls = ` class="alt"`
		}
		delta := ind.scoreAfter - ind.scoreBefore
		deltaStr := fmt.Sprintf("+%.1f", delta)
		deltaColor := "#15803d"
		if delta <= 0 {
			deltaStr = fmt.Sprintf("%.1f", delta)
			deltaColor = "#666"
		}
		sb.WriteString(fmt.Sprintf(`<tr%s><td class="left">%s</td><td>%.1f</td><td>%.1f</td><td style="color:%s;font-weight:600">%s</td></tr>`,
			cls, ind.name, ind.scoreBefore, ind.scoreAfter, deltaColor, deltaStr))
	}
	sb.WriteString(`</tbody></table>`)
}

func writeRadarChart(sb *strings.Builder, before, after *assessment.Assessment, isEn bool) {
	// Simple SVG radar chart
	labels := []string{"Row", "Col", "Format", "Dup", "Structure", "AI"}
	if !isEn {
		labels = []string{"列", "欄", "格式", "重複", "結構", "AI"}
	}
	beforeScores := []float64{before.RowCompleteness, before.ColumnCompleteness, before.FormatConsistency, before.DuplicateSimilar, before.TableStructure, before.AIQueryReadiness}
	afterScores := []float64{after.RowCompleteness, after.ColumnCompleteness, after.FormatConsistency, after.DuplicateSimilar, after.TableStructure, after.AIQueryReadiness}

	cx, cy, r := 150.0, 140.0, 100.0
	n := len(labels)

	sb.WriteString(`<div style="text-align:center;margin:16px 0">`)
	sb.WriteString(fmt.Sprintf(`<svg width="300" height="300" viewBox="0 0 300 300" style="font-family:sans-serif;font-size:10px">`))

	// Grid circles
	for _, pct := range []float64{0.25, 0.5, 0.75, 1.0} {
		sb.WriteString(fmt.Sprintf(`<circle cx="%.0f" cy="%.0f" r="%.0f" fill="none" stroke="#e5e7eb" stroke-width="0.5"/>`, cx, cy, r*pct))
	}

	// Axis lines + labels
	pointAt := func(i int) (float64, float64) {
		angle := float64(i)*2*3.14159/float64(n) - 3.14159/2
		x := cx + r*cosApprox(angle)
		y := cy + r*sinApprox(angle)
		return x, y
	}
	for i, label := range labels {
		x, y := pointAt(i)
		sb.WriteString(fmt.Sprintf(`<line x1="%.0f" y1="%.0f" x2="%.1f" y2="%.1f" stroke="#e5e7eb" stroke-width="0.5"/>`, cx, cy, x, y))
		// Label position (slightly outside)
		lx := cx + (r+15)*cosApprox(float64(i)*2*3.14159/float64(n)-3.14159/2)
		ly := cy + (r+15)*sinApprox(float64(i)*2*3.14159/float64(n)-3.14159/2)
		sb.WriteString(fmt.Sprintf(`<text x="%.1f" y="%.1f" text-anchor="middle" fill="#666">%s</text>`, lx, ly+4, label))
	}

	// Before polygon
	sb.WriteString(`<polygon points="`)
	for i, score := range beforeScores {
		pct := score / 100.0
		angle := float64(i)*2*3.14159/float64(n) - 3.14159/2
		x := cx + r*pct*cosApprox(angle)
		y := cy + r*pct*sinApprox(angle)
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(fmt.Sprintf("%.1f,%.1f", x, y))
	}
	sb.WriteString(`" fill="rgba(148,163,184,0.2)" stroke="#94a3b8" stroke-width="1.5"/>`)

	// After polygon
	sb.WriteString(`<polygon points="`)
	for i, score := range afterScores {
		pct := score / 100.0
		angle := float64(i)*2*3.14159/float64(n) - 3.14159/2
		x := cx + r*pct*cosApprox(angle)
		y := cy + r*pct*sinApprox(angle)
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(fmt.Sprintf("%.1f,%.1f", x, y))
	}
	sb.WriteString(`" fill="rgba(21,128,61,0.15)" stroke="#15803d" stroke-width="1.5"/>`)

	// Legend
	sb.WriteString(`<rect x="200" y="270" width="12" height="12" fill="rgba(148,163,184,0.4)"/>`)
	beforeLabel := "Before"
	afterLabel := "After"
	if !isEn {
		beforeLabel = "梳理前"
		afterLabel = "梳理後"
	}
	sb.WriteString(fmt.Sprintf(`<text x="216" y="280" fill="#666">%s</text>`, beforeLabel))
	sb.WriteString(`<rect x="260" y="270" width="12" height="12" fill="rgba(21,128,61,0.3)"/>`)
	sb.WriteString(fmt.Sprintf(`<text x="276" y="280" fill="#666">%s</text>`, afterLabel))

	sb.WriteString(`</svg></div>`)
}

// Simple trig functions for radar chart
func cosApprox(angle float64) float64 {
	return math.Cos(angle)
}

func sinApprox(angle float64) float64 {
	return math.Sin(angle)
}

// writeRadarChartSingle writes a single-layer radar for original assessment only
func writeRadarChartSingle(sb *strings.Builder, a *assessment.Assessment, isEn bool) {
	labels := []string{"Row", "Col", "Format", "Dup", "Structure", "AI"}
	if !isEn {
		labels = []string{"列", "欄", "格式", "重複", "結構", "AI"}
	}
	scores := []float64{a.RowCompleteness, a.ColumnCompleteness, a.FormatConsistency, a.DuplicateSimilar, a.TableStructure, a.AIQueryReadiness}

	cx, cy, r := 150.0, 140.0, 100.0
	n := len(labels)

	sb.WriteString(`<div style="text-align:center;margin:16px 0">`)
	sb.WriteString(`<svg width="300" height="280" viewBox="0 0 300 280" style="font-family:sans-serif;font-size:10px">`)

	for _, pct := range []float64{0.25, 0.5, 0.75, 1.0} {
		sb.WriteString(fmt.Sprintf(`<circle cx="%.0f" cy="%.0f" r="%.0f" fill="none" stroke="#e5e7eb" stroke-width="0.5"/>`, cx, cy, r*pct))
	}

	for i, label := range labels {
		angle := float64(i)*2*math.Pi/float64(n) - math.Pi/2
		x := cx + r*math.Cos(angle)
		y := cy + r*math.Sin(angle)
		sb.WriteString(fmt.Sprintf(`<line x1="%.0f" y1="%.0f" x2="%.1f" y2="%.1f" stroke="#e5e7eb" stroke-width="0.5"/>`, cx, cy, x, y))
		lx := cx + (r+15)*math.Cos(angle)
		ly := cy + (r+15)*math.Sin(angle)
		sb.WriteString(fmt.Sprintf(`<text x="%.1f" y="%.1f" text-anchor="middle" fill="#666">%s</text>`, lx, ly+4, label))
	}

	sb.WriteString(`<polygon points="`)
	for i, score := range scores {
		pct := score / 100.0
		angle := float64(i)*2*math.Pi/float64(n) - math.Pi/2
		x := cx + r*pct*math.Cos(angle)
		y := cy + r*pct*math.Sin(angle)
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(fmt.Sprintf("%.1f,%.1f", x, y))
	}
	sb.WriteString(`" fill="rgba(43,108,176,0.2)" stroke="#2b6cb0" stroke-width="1.5"/>`)
	sb.WriteString(`</svg></div>`)
}

// writeProgressBars writes before/after progress bars matching the frontend style
func writeProgressBars(sb *strings.Builder, before, after *assessment.Assessment, isEn bool) {
	type ind struct {
		name   string
		nameEn string
		before float64
		after  float64
	}
	indicators := []ind{
		{"列完整度", "Row Completeness", before.RowCompleteness, after.RowCompleteness},
		{"欄完整度", "Column Completeness", before.ColumnCompleteness, after.ColumnCompleteness},
		{"格式一致性", "Format Consistency", before.FormatConsistency, after.FormatConsistency},
		{"重複/相似", "Duplicate/Similar", before.DuplicateSimilar, after.DuplicateSimilar},
		{"表格結構", "Table Structure", before.TableStructure, after.TableStructure},
		{"AI查詢準備度", "AI Query Readiness", before.AIQueryReadiness, after.AIQueryReadiness},
	}

	title := "Indicator Improvement"
	if !isEn {
		title = "指標改善幅度"
	}
	sb.WriteString(fmt.Sprintf(`<h3 style="margin-top:16px">%s</h3>`, title))

	for _, ind := range indicators {
		name := ind.nameEn
		if !isEn {
			name = ind.name
		}
		delta := ind.after - ind.before
		baseWidth := math.Max(0, math.Min(100, ind.before))
		improvementWidth := 0.0
		if delta > 0 {
			improvementWidth = math.Min(delta, 100-baseWidth)
		}

		deltaStr := ""
		if delta > 0 {
			deltaStr = fmt.Sprintf(`<span style="color:#15803d;font-size:10px;margin-left:4px">(+%.1f)</span>`, delta)
		}

		sb.WriteString(`<div style="display:flex;align-items:center;gap:10px;padding:8px 0;border-bottom:1px solid #f3f4f6">`)
		sb.WriteString(fmt.Sprintf(`<div style="width:130px"><div style="font-size:12px;font-weight:600">%s</div></div>`, name))
		sb.WriteString(fmt.Sprintf(`<div style="flex:1;height:7px;border-radius:4px;background:#e5e7eb;position:relative;overflow:hidden">`))
		sb.WriteString(fmt.Sprintf(`<div style="position:absolute;left:0;top:0;height:100%%;width:%.0f%%;background:#2b6cb0;border-radius:4px"></div>`, baseWidth))
		if improvementWidth > 0 {
			sb.WriteString(fmt.Sprintf(`<div style="position:absolute;left:%.0f%%;top:0;height:100%%;width:%.0f%%;background:rgba(34,197,94,0.6);border-radius:0 4px 4px 0"></div>`, baseWidth, improvementWidth))
		}
		sb.WriteString(`</div>`)
		sb.WriteString(fmt.Sprintf(`<div style="width:100px;text-align:right;font-size:11px;font-family:monospace">%.1f / 100%s</div>`, ind.after, deltaStr))
		sb.WriteString(`</div>`)
	}
}

func reportCSS() string {
	return `
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "Noto Sans TC", sans-serif; font-size: 12px; line-height: 1.6; color: #1a1f2e; padding: 30px 40px; max-width: 780px; margin: 0 auto; }
.cover { text-align: center; padding: 120px 0 60px; }
.cover h1 { font-size: 36px; font-weight: 700; margin-bottom: 8px; }
.cover h2 { font-size: 20px; font-weight: 400; color: #555; }
.cover .meta { margin-top: 12px; font-size: 12px; color: #888; }
.page-break { page-break-before: always; padding-top: 20px; }
.section-title { font-size: 16px; font-weight: 700; border-bottom: 2px solid #2b6cb0; padding-bottom: 5px; margin-bottom: 14px; margin-top: 10px; }
h3 { font-size: 13px; font-weight: 600; margin: 12px 0 6px; }
.note { font-size: 11px; color: #666; margin-bottom: 10px; }
.score-block { text-align: center; margin: 16px 0 20px; }
.score { font-size: 42px; font-weight: 700; display: block; }
.grade { font-size: 13px; font-weight: 600; }
table { width: 100%; border-collapse: collapse; margin-bottom: 16px; }
.ind-table th, .comp-table th { background: #2b6cb0; color: #fff; padding: 6px 10px; text-align: center; font-size: 11px; }
.ind-table td, .comp-table td { padding: 5px 10px; text-align: center; border: 1px solid #e5e7eb; font-size: 11px; }
.ind-table td.left { text-align: left; }
tr.alt td, .ind-table tr.alt td { background: #f9fafb; }
.green { color: #15803d; font-weight: 600; }
.rules { padding-left: 20px; margin-bottom: 14px; }
.rules li { margin-bottom: 3px; font-size: 12px; }
.issue-card { border: 1px solid #e5e7eb; border-radius: 6px; margin-bottom: 10px; overflow: hidden; page-break-inside: avoid; }
.issue-header { padding: 8px 12px; background: #f9fafb; border-bottom: 1px solid #e5e7eb; display: flex; align-items: center; gap: 8px; font-size: 12px; }
.issue-title { font-weight: 600; flex: 1; }
.affected { font-size: 10px; color: #666; }
.sev-high { font-size: 10px; font-weight: 600; padding: 2px 6px; border-radius: 3px; background: #fef2f2; color: #dc2626; }
.sev-med { font-size: 10px; font-weight: 600; padding: 2px 6px; border-radius: 3px; background: #fffbeb; color: #d97706; }
.sev-low { font-size: 10px; font-weight: 600; padding: 2px 6px; border-radius: 3px; background: #f0fdf4; color: #16a34a; }
.issue-desc { padding: 6px 12px; font-size: 11px; color: #555; }
.group-label { font-size: 11px; font-weight: 600; color: #2b6cb0; margin: 6px 12px 3px; }
.excel-table { margin: 4px 12px 10px; width: calc(100% - 24px); font-size: 10px; }
.excel-table th { background: #f3f4f6; padding: 3px 6px; font-weight: 600; border: 1px solid #e5e7eb; font-size: 10px; }
.excel-table td { padding: 3px 6px; border: 1px solid #e5e7eb; max-width: 120px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.excel-table td.hl { background: rgba(220,38,38,0.06); border: 1.5px solid #dc2626; color: #dc2626; }
.excel-table .row-num { width: 28px; text-align: center; color: #999; font-size: 9px; }
`
}
