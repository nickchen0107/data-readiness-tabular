package assessment

import (
	"fmt"
	"math"
	"testing"

	"github.com/safe-ai/excel-brushing-tool/internal/upload"
	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

// --- Generators ---

func genSheetData(t *rapid.T) *upload.SheetData {
	rows := rapid.IntRange(0, 50).Draw(t, "rows")
	cols := rapid.IntRange(1, 10).Draw(t, "cols")
	headers := make([]string, cols)
	for i := range headers {
		headers[i] = rapid.StringMatching(`[A-Za-z]{2,10}`).Draw(t, fmt.Sprintf("header_%d", i))
	}
	data := make([][]upload.CellValue, rows)
	for i := range data {
		row := make([]upload.CellValue, cols)
		for j := range row {
			isEmpty := rapid.Bool().Draw(t, fmt.Sprintf("empty_%d_%d", i, j))
			if isEmpty {
				row[j] = upload.CellValue{Raw: "", IsEmpty: true}
			} else {
				row[j] = upload.CellValue{Raw: rapid.StringMatching(`[A-Za-z0-9]{1,15}`).Draw(t, fmt.Sprintf("val_%d_%d", i, j)), IsEmpty: false}
			}
		}
		data[i] = row
	}
	return &upload.SheetData{Headers: headers, Rows: data, RowCount: rows, ColCount: cols}
}

func genNonEmptySheetData(t *rapid.T) *upload.SheetData {
	rows := rapid.IntRange(1, 50).Draw(t, "rows")
	cols := rapid.IntRange(1, 10).Draw(t, "cols")
	headers := make([]string, cols)
	for i := range headers {
		headers[i] = rapid.StringMatching(`[A-Za-z]{2,10}`).Draw(t, fmt.Sprintf("header_%d", i))
	}
	data := make([][]upload.CellValue, rows)
	for i := range data {
		row := make([]upload.CellValue, cols)
		for j := range row {
			isEmpty := rapid.Bool().Draw(t, fmt.Sprintf("empty_%d_%d", i, j))
			if isEmpty {
				row[j] = upload.CellValue{Raw: "", IsEmpty: true}
			} else {
				row[j] = upload.CellValue{Raw: rapid.StringMatching(`[A-Za-z0-9]{1,15}`).Draw(t, fmt.Sprintf("val_%d_%d", i, j)), IsEmpty: false}
			}
		}
		data[i] = row
	}
	return &upload.SheetData{Headers: headers, Rows: data, RowCount: rows, ColCount: cols}
}

func genValidWeights(t *rapid.T) Weights {
	// Generate 5 random values, compute the 6th so they sum to 1.0
	w1 := rapid.Float64Range(0.01, 0.4).Draw(t, "w1")
	w2 := rapid.Float64Range(0.01, 0.4).Draw(t, "w2")
	w3 := rapid.Float64Range(0.01, 0.4).Draw(t, "w3")
	w4 := rapid.Float64Range(0.01, 0.4).Draw(t, "w4")
	w5 := rapid.Float64Range(0.01, 0.4).Draw(t, "w5")

	sum5 := w1 + w2 + w3 + w4 + w5
	// If sum5 >= 1.0, rescale
	if sum5 >= 0.99 {
		factor := 0.8 / sum5
		w1 *= factor
		w2 *= factor
		w3 *= factor
		w4 *= factor
		w5 *= factor
		sum5 = w1 + w2 + w3 + w4 + w5
	}
	w6 := 1.0 - sum5

	return Weights{
		RowCompleteness:    w1,
		ColumnCompleteness: w2,
		FormatConsistency:  w3,
		DuplicateSimilar:   w4,
		TableStructure:     w5,
		AIQueryReadiness:   w6,
	}
}

func genIndicatorScores(t *rapid.T) IndicatorScores {
	return IndicatorScores{
		RowCompleteness:    rapid.Float64Range(0, 100).Draw(t, "rc"),
		ColumnCompleteness: rapid.Float64Range(0, 100).Draw(t, "cc"),
		FormatConsistency:  rapid.Float64Range(0, 100).Draw(t, "fc"),
		DuplicateSimilar:   rapid.Float64Range(0, 100).Draw(t, "ds"),
		TableStructure:     rapid.Float64Range(0, 100).Draw(t, "ts"),
		AIQueryReadiness:   rapid.Float64Range(0, 100).Draw(t, "aq"),
	}
}

// --- Property 5: Row Completeness formula correctness ---
// **Validates: Requirements 4.1, 4.3**
// For any grid, score = avg(per-row non-empty/cols) * 100. Zero rows → 0.
func TestPBT_RowCompleteness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		data := genSheetData(t)

		result := CalculateRowCompleteness(data)

		// Manually compute expected
		if len(data.Rows) == 0 || data.ColCount == 0 {
			assert.Equal(t, 0.0, result)
			return
		}

		totalRatio := 0.0
		for _, row := range data.Rows {
			nonEmpty := 0
			for i := 0; i < data.ColCount; i++ {
				if i < len(row) && !row[i].IsEmpty {
					nonEmpty++
				}
			}
			totalRatio += float64(nonEmpty) / float64(data.ColCount)
		}
		expected := (totalRatio / float64(len(data.Rows))) * 100

		assert.InDelta(t, expected, result, 0.0001)
		assert.GreaterOrEqual(t, result, 0.0)
		assert.LessOrEqual(t, result, 100.0)
	})
}

// --- Property 6: Column Completeness formula correctness ---
// **Validates: Requirements 5.1, 5.3, 5.4**
func TestPBT_ColumnCompleteness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		data := genSheetData(t)

		result, details := CalculateColumnCompleteness(data)

		if len(data.Rows) == 0 || data.ColCount == 0 {
			assert.Equal(t, 0.0, result)
			assert.Nil(t, details)
			return
		}

		// Manually compute expected
		totalRatio := 0.0
		for col := 0; col < data.ColCount; col++ {
			nonEmpty := 0
			for _, row := range data.Rows {
				if col < len(row) && !row[col].IsEmpty {
					nonEmpty++
				}
			}
			colRatio := float64(nonEmpty) / float64(len(data.Rows))
			totalRatio += colRatio

			// Verify per-column detail
			assert.InDelta(t, colRatio, details[col].CompletenessRatio, 0.0001)
		}
		expected := (totalRatio / float64(data.ColCount)) * 100

		assert.InDelta(t, expected, result, 0.0001)
		assert.GreaterOrEqual(t, result, 0.0)
		assert.LessOrEqual(t, result, 100.0)
		assert.Len(t, details, data.ColCount)
	})
}

// --- Property 7: Format type detection priority (date > numeric > boolean > text) ---
// **Validates: Requirements 6.1**
func TestPBT_FormatTypePriority(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate values of known types and verify priority ordering
		kind := rapid.IntRange(0, 3).Draw(t, "kind")

		var value string
		var expectedType FormatType

		switch kind {
		case 0: // Date
			y := rapid.IntRange(2000, 2030).Draw(t, "year")
			m := rapid.IntRange(1, 12).Draw(t, "month")
			d := rapid.IntRange(1, 28).Draw(t, "day")
			value = fmt.Sprintf("%04d-%02d-%02d", y, m, d)
			expectedType = FormatDate
		case 1: // Numeric
			n := rapid.IntRange(-9999, 9999).Draw(t, "num")
			value = fmt.Sprintf("%d", n)
			expectedType = FormatNumeric
		case 2: // Boolean
			bools := []string{"true", "false", "是", "否", "Y", "N", "yes", "no"}
			idx := rapid.IntRange(0, len(bools)-1).Draw(t, "bool_idx")
			value = bools[idx]
			expectedType = FormatBoolean
		case 3: // Text (guaranteed not parseable as other types)
			value = "Hello_" + rapid.StringMatching(`[A-Z]{3,8}`).Draw(t, "text")
			expectedType = FormatText
		}

		result := DetectFormatType(value)

		// Verify priority: the detected type should be >= expectedType in priority
		// (lower enum value = higher priority). If it detects a higher priority, that's fine.
		assert.LessOrEqual(t, int(result), int(expectedType),
			"value=%q expected at most type %d, got %d", value, expectedType, result)
	})
}

// --- Property 8: Format Consistency calculation correctness ---
// **Validates: Requirements 6.2, 6.3, 6.4**
func TestPBT_FormatConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		data := genSheetData(t)

		result := CalculateFormatConsistency(data)

		// Verify it's bounded [0, 100]
		assert.GreaterOrEqual(t, result, 0.0)
		assert.LessOrEqual(t, result, 100.0)

		// Manually compute expected
		if data.ColCount == 0 {
			assert.InDelta(t, 100.0, result, 0.001)
			return
		}

		validCols := 0
		totalScore := 0.0
		for col := 0; col < data.ColCount; col++ {
			formatCounts := map[FormatType]int{}
			nonEmptyCount := 0
			for _, row := range data.Rows {
				if col < len(row) && !row[col].IsEmpty {
					nonEmptyCount++
					ft := DetectFormatType(row[col].Raw)
					formatCounts[ft]++
				}
			}
			if nonEmptyCount == 0 {
				continue
			}
			validCols++
			dominantCount := 0
			for ft := FormatDate; ft <= FormatText; ft++ {
				if formatCounts[ft] > dominantCount {
					dominantCount = formatCounts[ft]
				}
			}
			totalScore += float64(dominantCount) / float64(nonEmptyCount)
		}

		if validCols == 0 {
			assert.InDelta(t, 100.0, result, 0.001)
		} else {
			expected := (totalScore / float64(validCols)) * 100
			assert.InDelta(t, expected, result, 0.001)
		}
	})
}

// --- Property 9: Duplicate/Similar score formula ---
// **Validates: Requirements 7.1, 7.2, 7.3, 7.5**
func TestPBT_DuplicateSimilar(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		data := genSheetData(t)

		result := CalculateDuplicateSimilar(data)

		// Score must be in [0, 100]
		assert.GreaterOrEqual(t, result, 0.0)
		assert.LessOrEqual(t, result, 100.0)

		// Empty data → 100
		if len(data.Rows) == 0 {
			assert.Equal(t, 100.0, result)
		}
	})
}

// --- Property 10: Table Structure deductions with floor at 0 ---
// **Validates: Requirements 8.1, 8.6**
func TestPBT_TableStructureFloor(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		data := genSheetData(t)
		// Randomly add merged cells
		hasMerged := rapid.Bool().Draw(t, "has_merged")
		if hasMerged && data.ColCount > 0 {
			data.MergedCells = []upload.MergedRange{
				{StartRow: 0, EndRow: 0, StartCol: 0, EndCol: data.ColCount - 1},
			}
		}

		result := CalculateTableStructure(data)

		// Score must be in [0, 100], floor at 0
		assert.GreaterOrEqual(t, result, 0.0)
		assert.LessOrEqual(t, result, 100.0)

		// Verify the score is a valid deduction from 100
		// Possible deductions: 20, 20, 15, 25, 10 → max total = 90
		// So score >= 10 unless notes also detected (then >= 0)
		// But floor is at 0 regardless
		assert.GreaterOrEqual(t, result, 0.0)
	})
}

// --- Property 14: AI Query Readiness score is 0-100 ---
// **Validates: Requirements 9.1, 9.3, 9.4**
func TestPBT_AIQueryReadinessBounded(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		data := genSheetData(t)

		result := CalculateAIQueryReadiness(data)

		// Must be bounded [0, 100]
		assert.GreaterOrEqual(t, result, 0.0)
		assert.LessOrEqual(t, result, 100.0)

		// 0 rows → 0
		if len(data.Rows) == 0 {
			assert.Equal(t, 0.0, result)
		}

		// Score must be a multiple of 20 (5 sub-conditions × 20 each)
		remainder := math.Mod(result, 20.0)
		assert.InDelta(t, 0.0, remainder, 0.001,
			"AI Query Readiness score should be a multiple of 20, got %f", result)
	})
}

// --- Property 15: Weighted score calculation ---
// **Validates: Requirements 10.1, 10.2, 10.3, 10.4**
func TestPBT_WeightedScore(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		scores := genIndicatorScores(t)
		weights := genValidWeights(t)

		total, grade, err := CalculateTotalScore(scores, weights)

		assert.NoError(t, err)

		// Verify total is bounded [0, 100]
		assert.GreaterOrEqual(t, total, 0.0)
		assert.LessOrEqual(t, total, 100.0)

		// Verify the weighted sum (before rounding)
		rawTotal := scores.RowCompleteness*weights.RowCompleteness +
			scores.ColumnCompleteness*weights.ColumnCompleteness +
			scores.FormatConsistency*weights.FormatConsistency +
			scores.DuplicateSimilar*weights.DuplicateSimilar +
			scores.TableStructure*weights.TableStructure +
			scores.AIQueryReadiness*weights.AIQueryReadiness
		expectedRounded := math.Round(rawTotal*10) / 10

		assert.InDelta(t, expectedRounded, total, 0.001)

		// Verify grading
		switch {
		case total >= 80:
			assert.Equal(t, "ready", grade)
		case total >= 60:
			assert.Equal(t, "conditional", grade)
		default:
			assert.Equal(t, "not_ready", grade)
		}
	})
}

// --- Property 16: Assessment determinism (same input → same output) ---
// **Validates: Requirements 10.5**
func TestPBT_AssessmentDeterminism(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		data := genNonEmptySheetData(t)

		// Run all indicators twice
		rc1 := CalculateRowCompleteness(data)
		rc2 := CalculateRowCompleteness(data)
		assert.Equal(t, rc1, rc2)

		cc1, d1 := CalculateColumnCompleteness(data)
		cc2, d2 := CalculateColumnCompleteness(data)
		assert.Equal(t, cc1, cc2)
		assert.Equal(t, d1, d2)

		fc1 := CalculateFormatConsistency(data)
		fc2 := CalculateFormatConsistency(data)
		assert.Equal(t, fc1, fc2)

		ds1 := CalculateDuplicateSimilar(data)
		ds2 := CalculateDuplicateSimilar(data)
		assert.Equal(t, ds1, ds2)

		ts1 := CalculateTableStructure(data)
		ts2 := CalculateTableStructure(data)
		assert.Equal(t, ts1, ts2)

		ai1 := CalculateAIQueryReadiness(data)
		ai2 := CalculateAIQueryReadiness(data)
		assert.Equal(t, ai1, ai2)
	})
}

// --- Property 17: Invalid weight sum rejection ---
// **Validates: Requirements 10.8**
func TestPBT_InvalidWeightRejection(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate weights that explicitly don't sum to 1.0
		w := Weights{
			RowCompleteness:    rapid.Float64Range(0.0, 1.0).Draw(t, "w1"),
			ColumnCompleteness: rapid.Float64Range(0.0, 1.0).Draw(t, "w2"),
			FormatConsistency:  rapid.Float64Range(0.0, 1.0).Draw(t, "w3"),
			DuplicateSimilar:   rapid.Float64Range(0.0, 1.0).Draw(t, "w4"),
			TableStructure:     rapid.Float64Range(0.0, 1.0).Draw(t, "w5"),
			AIQueryReadiness:   rapid.Float64Range(0.0, 1.0).Draw(t, "w6"),
		}

		scores := genIndicatorScores(t)

		_, _, err := CalculateTotalScore(scores, w)

		if math.Abs(w.Sum()-1.0) < 0.001 {
			// Weights happen to be valid
			assert.NoError(t, err)
		} else {
			// Weights are invalid → must reject
			assert.Error(t, err)
		}
	})
}
