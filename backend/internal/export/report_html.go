package export

import (
	"fmt"
	"html"
	"strings"

	"github.com/safe-ai/excel-brushing-tool/internal/assessment"
)

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
		writeIndicatorTable(&sb, data.PostAssessment, isEn)
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
	sb.WriteString(fmt.Sprintf(`<th>%s</th><th>%s</th>`, hdr1, hdr2))
	sb.WriteString(`</tr></thead><tbody>`)
	for i, ind := range indicators {
		cls := ""
		if i%2 == 0 {
			cls = ` class="alt"`
		}
		sb.WriteString(fmt.Sprintf(`<tr%s><td class="left">%s</td><td>%.1f</td></tr>`, cls, ind.name, ind.score))
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
		}
		if desc != "" {
			sb.WriteString(fmt.Sprintf(`<div class="issue-desc">%s</div>`, html.EscapeString(desc)))
		}

		// Examples (Excel-style tables)
		if len(issue.Examples) > 0 {
			writeExamples(sb, issue.Examples)
		}

		sb.WriteString(`</div>`)
	}
}

func writeExamples(sb *strings.Builder, examples []assessment.IssueExample) {
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
			sb.WriteString(fmt.Sprintf(`<div class="group-label">%s</div>`, html.EscapeString(grp.label)))
		}

		sb.WriteString(`<table class="excel-table"><thead><tr><th class="row-num">#</th>`)
		if len(grp.items) > 0 && len(grp.items[0].Headers) > 0 {
			for _, h := range grp.items[0].Headers {
				sb.WriteString(fmt.Sprintf(`<th>%s</th>`, html.EscapeString(h)))
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
				cellVal := cell
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
