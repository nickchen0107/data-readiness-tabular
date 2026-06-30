package qa

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

// --- Generators ---

func genHeaders(t *rapid.T) []string {
	count := rapid.IntRange(1, 10).Draw(t, "num_headers")
	headers := make([]string, count)
	for i := range headers {
		headers[i] = rapid.StringMatching(`[A-Z][a-z]{2,10}`).Draw(t, fmt.Sprintf("header_%d", i))
	}
	return headers
}

func genRows(t *rapid.T, colCount int) [][]string {
	rowCount := rapid.IntRange(1, 50).Draw(t, "num_rows")
	rows := make([][]string, rowCount)
	for i := range rows {
		row := make([]string, colCount)
		for j := range row {
			isEmpty := rapid.Bool().Draw(t, fmt.Sprintf("empty_%d_%d", i, j))
			if isEmpty {
				row[j] = ""
			} else {
				row[j] = rapid.StringMatching(`[A-Za-z0-9]{1,15}`).Draw(t, fmt.Sprintf("val_%d_%d", i, j))
			}
		}
		rows[i] = row
	}
	return rows
}

// genHighMissingRows generates rows where the specified column has > threshold missing rate
func genHighMissingRows(t *rapid.T, colCount int, targetCol int, threshold float64) [][]string {
	rowCount := rapid.IntRange(10, 50).Draw(t, "num_rows")
	rows := make([][]string, rowCount)
	// Ensure > threshold fraction of target column is missing
	missingCount := int(float64(rowCount)*threshold) + 1
	if missingCount > rowCount {
		missingCount = rowCount
	}

	for i := range rows {
		row := make([]string, colCount)
		for j := range row {
			if j == targetCol && i < missingCount {
				row[j] = "" // empty
			} else {
				row[j] = rapid.StringMatching(`[A-Za-z0-9]{1,10}`).Draw(t, fmt.Sprintf("val_%d_%d", i, j))
			}
		}
		rows[i] = row
	}
	return rows
}

// --- Property 26: Data insufficiency guardrail blocks when > threshold ---
// **Validates: Requirements 16.2**
func TestPBT_DataInsufficiencyGuardrail(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		headers := genHeaders(t)
		colCount := len(headers)
		threshold := 0.5

		// Pick a target column to make insufficient
		targetCol := rapid.IntRange(0, colCount-1).Draw(t, "target_col")

		// Generate rows with high missing rate in target column
		rows := genHighMissingRows(t, colCount, targetCol, threshold)

		// Ask a question that references the target column name
		question := fmt.Sprintf("請分析%s的趨勢", headers[targetCol])

		result := CheckDataInsufficiency(headers, rows, question, threshold)

		// Calculate actual missing rate for verification
		missingCount := 0
		for _, row := range rows {
			if targetCol >= len(row) || strings.TrimSpace(row[targetCol]) == "" {
				missingCount++
			}
		}
		actualMissingRate := float64(missingCount) / float64(len(rows))

		if actualMissingRate > threshold {
			assert.True(t, result.Blocked,
				"Should block when missing rate %.2f > threshold %.2f", actualMissingRate, threshold)
			assert.NotEmpty(t, result.Explanation)
			assert.Contains(t, result.Explanation, "資料不足")
		}
	})
}

// --- Property 27: Suggested questions reference column names ---
// **Validates: Requirements 16.3**
func TestPBT_SuggestedQuestionsReferenceColumns(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		headers := genHeaders(t)

		suggestions := GenerateSuggestions(headers)

		// Must return exactly 3 suggestions
		assert.Len(t, suggestions, 3, "Must generate exactly 3 suggestions")

		// Each suggestion must reference at least one column name
		for i, suggestion := range suggestions {
			found := false
			for _, header := range headers {
				if strings.Contains(suggestion, header) {
					found = true
					break
				}
			}
			assert.True(t, found,
				"Suggestion %d (%q) must reference at least one column name from %v", i, suggestion, headers)
		}
	})
}
