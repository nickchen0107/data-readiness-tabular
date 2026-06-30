package qa

import (
	"fmt"
	"strings"
)

// CheckDataInsufficiency checks if the data has too many missing values
// for the columns referenced in the question.
// Returns a GuardrailResult indicating whether the LLM call should be blocked.
func CheckDataInsufficiency(headers []string, rows [][]string, question string, threshold float64) GuardrailResult {
	if len(rows) == 0 {
		return GuardrailResult{
			Blocked:     true,
			Explanation: "資料不足：資料集中沒有任何資料列",
		}
	}

	// Find columns mentioned in the question
	mentionedCols := findMentionedColumns(headers, question)

	// If no specific columns mentioned, check all columns
	if len(mentionedCols) == 0 {
		mentionedCols = make([]int, len(headers))
		for i := range headers {
			mentionedCols[i] = i
		}
	}

	// Check missing rate for mentioned columns
	var insufficientCols []string
	for _, colIdx := range mentionedCols {
		if colIdx >= len(headers) {
			continue
		}
		missingCount := 0
		for _, row := range rows {
			if colIdx >= len(row) || strings.TrimSpace(row[colIdx]) == "" {
				missingCount++
			}
		}
		missingRate := float64(missingCount) / float64(len(rows))
		if missingRate > threshold {
			insufficientCols = append(insufficientCols, fmt.Sprintf("%s (缺漏率 %.0f%%)", headers[colIdx], missingRate*100))
		}
	}

	if len(insufficientCols) > 0 {
		explanation := fmt.Sprintf("資料不足：以下欄位缺漏率超過 %.0f%% 門檻值 — %s",
			threshold*100, strings.Join(insufficientCols, "、"))
		return GuardrailResult{
			Blocked:     true,
			Explanation: explanation,
		}
	}

	return GuardrailResult{Blocked: false}
}

// findMentionedColumns returns column indices that are mentioned in the question
func findMentionedColumns(headers []string, question string) []int {
	var mentioned []int
	questionLower := strings.ToLower(question)

	for i, header := range headers {
		headerLower := strings.ToLower(strings.TrimSpace(header))
		if headerLower == "" {
			continue
		}
		if strings.Contains(questionLower, headerLower) {
			mentioned = append(mentioned, i)
		}
	}

	return mentioned
}
