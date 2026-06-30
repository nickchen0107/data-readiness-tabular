package assessment

import (
	"fmt"
	"testing"

	"github.com/safe-ai/excel-brushing-tool/internal/upload"
	"github.com/stretchr/testify/assert"
)

// --- Helper functions ---

func cell(raw string, isEmpty bool) upload.CellValue {
	return upload.CellValue{Raw: raw, IsEmpty: isEmpty}
}

func nonEmpty(raw string) upload.CellValue {
	return upload.CellValue{Raw: raw, IsEmpty: false}
}

func empty() upload.CellValue {
	return upload.CellValue{Raw: "", IsEmpty: true}
}

func makeSheetData(headers []string, rows [][]upload.CellValue) *upload.SheetData {
	colCount := len(headers)
	return &upload.SheetData{
		Headers:  headers,
		Rows:     rows,
		RowCount: len(rows),
		ColCount: colCount,
	}
}

// --- Row Completeness Tests ---

func TestCalculateRowCompleteness_AllFilled(t *testing.T) {
	data := makeSheetData(
		[]string{"A", "B", "C"},
		[][]upload.CellValue{
			{nonEmpty("1"), nonEmpty("2"), nonEmpty("3")},
			{nonEmpty("4"), nonEmpty("5"), nonEmpty("6")},
		},
	)
	score := CalculateRowCompleteness(data)
	assert.InDelta(t, 100.0, score, 0.001)
}

func TestCalculateRowCompleteness_HalfFilled(t *testing.T) {
	data := makeSheetData(
		[]string{"A", "B"},
		[][]upload.CellValue{
			{nonEmpty("1"), empty()},
			{nonEmpty("2"), empty()},
		},
	)
	score := CalculateRowCompleteness(data)
	assert.InDelta(t, 50.0, score, 0.001)
}

func TestCalculateRowCompleteness_NoRows(t *testing.T) {
	data := makeSheetData([]string{"A", "B"}, nil)
	score := CalculateRowCompleteness(data)
	assert.Equal(t, 0.0, score)
}

func TestCalculateRowCompleteness_MixedRows(t *testing.T) {
	data := makeSheetData(
		[]string{"A", "B", "C", "D"},
		[][]upload.CellValue{
			{nonEmpty("1"), nonEmpty("2"), nonEmpty("3"), nonEmpty("4")}, // 4/4 = 1.0
			{nonEmpty("1"), empty(), empty(), empty()},                   // 1/4 = 0.25
		},
	)
	// Average: (1.0 + 0.25) / 2 = 0.625 → × 100 = 62.5
	score := CalculateRowCompleteness(data)
	assert.InDelta(t, 62.5, score, 0.001)
}

// --- Column Completeness Tests ---

func TestCalculateColumnCompleteness_AllFilled(t *testing.T) {
	data := makeSheetData(
		[]string{"A", "B"},
		[][]upload.CellValue{
			{nonEmpty("1"), nonEmpty("2")},
			{nonEmpty("3"), nonEmpty("4")},
		},
	)
	score, details := CalculateColumnCompleteness(data)
	assert.InDelta(t, 100.0, score, 0.001)
	assert.Len(t, details, 2)
	assert.InDelta(t, 1.0, details[0].CompletenessRatio, 0.001)
	assert.InDelta(t, 1.0, details[1].CompletenessRatio, 0.001)
}

func TestCalculateColumnCompleteness_PartiallyFilled(t *testing.T) {
	data := makeSheetData(
		[]string{"Name", "Age"},
		[][]upload.CellValue{
			{nonEmpty("Alice"), nonEmpty("30")},
			{nonEmpty("Bob"), empty()},
			{empty(), nonEmpty("25")},
		},
	)
	score, details := CalculateColumnCompleteness(data)
	// Col 0: 2/3, Col 1: 2/3 → average = 2/3 → ×100 = 66.67
	assert.InDelta(t, 66.667, score, 0.01)
	assert.InDelta(t, 2.0/3.0, details[0].CompletenessRatio, 0.001)
	assert.InDelta(t, 2.0/3.0, details[1].CompletenessRatio, 0.001)
}

func TestCalculateColumnCompleteness_NoRows(t *testing.T) {
	data := makeSheetData([]string{"A", "B"}, nil)
	score, details := CalculateColumnCompleteness(data)
	assert.Equal(t, 0.0, score)
	assert.Nil(t, details)
}

// --- Format Consistency Tests ---

func TestCalculateFormatConsistency_AllSameFormat(t *testing.T) {
	data := makeSheetData(
		[]string{"ID", "Date"},
		[][]upload.CellValue{
			{nonEmpty("123"), nonEmpty("2024-01-01")},
			{nonEmpty("456"), nonEmpty("2024-02-15")},
			{nonEmpty("789"), nonEmpty("2024-03-20")},
		},
	)
	score := CalculateFormatConsistency(data)
	assert.InDelta(t, 100.0, score, 0.001)
}

func TestCalculateFormatConsistency_MixedFormats(t *testing.T) {
	data := makeSheetData(
		[]string{"Values"},
		[][]upload.CellValue{
			{nonEmpty("123")},
			{nonEmpty("abc")},
			{nonEmpty("456")},
			{nonEmpty("def")},
		},
	)
	// 2 numeric, 2 text → dominant = 2/4 = 0.5 → ×100 = 50.0
	score := CalculateFormatConsistency(data)
	assert.InDelta(t, 50.0, score, 0.001)
}

func TestCalculateFormatConsistency_AllEmptyColumn(t *testing.T) {
	data := makeSheetData(
		[]string{"A", "B"},
		[][]upload.CellValue{
			{nonEmpty("test"), empty()},
			{nonEmpty("test2"), empty()},
		},
	)
	// Col A: all text → 1.0; Col B: all empty → excluded
	// Average = 1.0 → ×100 = 100.0
	score := CalculateFormatConsistency(data)
	assert.InDelta(t, 100.0, score, 0.001)
}

// --- Format Detector Tests ---

func TestDetectFormatType_Dates(t *testing.T) {
	tests := []struct {
		input    string
		expected FormatType
	}{
		{"2024-01-15", FormatDate},
		{"2024/01/15", FormatDate},
		{"113.1.5", FormatDate},
		{"112.12.31", FormatDate},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, DetectFormatType(tt.input))
		})
	}
}

func TestDetectFormatType_Numeric(t *testing.T) {
	tests := []struct {
		input    string
		expected FormatType
	}{
		{"123", FormatNumeric},
		{"-456", FormatNumeric},
		{"1,234", FormatNumeric},
		{"1,234,567", FormatNumeric},
		{"3.14", FormatNumeric},
		{".5", FormatNumeric},
		{"1,234.56", FormatNumeric},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, DetectFormatType(tt.input))
		})
	}
}

func TestDetectFormatType_Boolean(t *testing.T) {
	tests := []struct {
		input    string
		expected FormatType
	}{
		{"true", FormatBoolean},
		{"FALSE", FormatBoolean},
		{"是", FormatBoolean},
		{"否", FormatBoolean},
		{"Y", FormatBoolean},
		{"n", FormatBoolean},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, DetectFormatType(tt.input))
		})
	}
}

func TestDetectFormatType_Text(t *testing.T) {
	tests := []struct {
		input    string
		expected FormatType
	}{
		{"hello world", FormatText},
		{"ABC公司", FormatText},
		{"mixed 123 text", FormatText},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, DetectFormatType(tt.input))
		})
	}
}

// --- Duplicate/Similar Tests ---

func TestCalculateDuplicateSimilar_NoDuplicates(t *testing.T) {
	data := makeSheetData(
		[]string{"Name", "Age"},
		[][]upload.CellValue{
			{nonEmpty("Alice"), nonEmpty("30")},
			{nonEmpty("Bob"), nonEmpty("25")},
			{nonEmpty("Carol"), nonEmpty("35")},
		},
	)
	score := CalculateDuplicateSimilar(data)
	assert.InDelta(t, 100.0, score, 0.001)
}

func TestCalculateDuplicateSimilar_AllDuplicates(t *testing.T) {
	data := makeSheetData(
		[]string{"Name", "Age"},
		[][]upload.CellValue{
			{nonEmpty("Alice"), nonEmpty("30")},
			{nonEmpty("Alice"), nonEmpty("30")},
			{nonEmpty("Alice"), nonEmpty("30")},
		},
	)
	// 2 exact duplicates (excess beyond first occurrence)
	// penalty = 2/3 → score = max(0, (1 - 2/3) × 100) = 33.33
	score := CalculateDuplicateSimilar(data)
	assert.InDelta(t, 33.333, score, 0.5)
}

func TestCalculateDuplicateSimilar_NoRows(t *testing.T) {
	data := makeSheetData([]string{"A"}, nil)
	score := CalculateDuplicateSimilar(data)
	assert.Equal(t, 100.0, score)
}

// --- Table Structure Tests ---

func TestCalculateTableStructure_PerfectTable(t *testing.T) {
	data := makeSheetData(
		[]string{"ID", "Name", "Value"},
		[][]upload.CellValue{
			{nonEmpty("1"), nonEmpty("Alice"), nonEmpty("100")},
			{nonEmpty("2"), nonEmpty("Bob"), nonEmpty("200")},
		},
	)
	score := CalculateTableStructure(data)
	assert.Equal(t, 100.0, score)
}

func TestCalculateTableStructure_WithMergedCells(t *testing.T) {
	data := &upload.SheetData{
		Headers:  []string{"A", "B"},
		Rows:     [][]upload.CellValue{{nonEmpty("1"), nonEmpty("2")}},
		RowCount: 1,
		ColCount: 2,
		MergedCells: []upload.MergedRange{
			{StartRow: 0, EndRow: 1, StartCol: 0, EndCol: 1},
		},
	}
	score := CalculateTableStructure(data)
	assert.Equal(t, 80.0, score)
}

func TestCalculateTableStructure_WithSubtotalRow(t *testing.T) {
	data := makeSheetData(
		[]string{"Item", "Amount"},
		[][]upload.CellValue{
			{nonEmpty("Product A"), nonEmpty("100")},
			{nonEmpty("Product B"), nonEmpty("200")},
			{nonEmpty("合計"), nonEmpty("300")},
		},
	)
	score := CalculateTableStructure(data)
	assert.Equal(t, 85.0, score)
}

func TestCalculateTableStructure_WithMultipleTables(t *testing.T) {
	data := makeSheetData(
		[]string{"A", "B"},
		[][]upload.CellValue{
			{nonEmpty("1"), nonEmpty("2")},
			{empty(), empty()},
			{empty(), empty()},
			{nonEmpty("3"), nonEmpty("4")},
		},
	)
	score := CalculateTableStructure(data)
	assert.Equal(t, 75.0, score)
}

// --- AI Query Readiness Tests ---

func TestCalculateAIQueryReadiness_FullScore(t *testing.T) {
	// Build data that satisfies all 5 sub-conditions
	// Need unique count < 20% of total rows AND > 1 for category
	// With 20 rows: 20% = 4. Need uniqueCount < 4 AND > 1
	rows := make([][]upload.CellValue, 20)
	for i := 0; i < 20; i++ {
		cat := "TypeA"
		if i%10 == 0 {
			cat = "TypeB"
		}
		rows[i] = []upload.CellValue{
			nonEmpty(fmt.Sprintf("ID-%d", i)),             // identifier (unique)
			nonEmpty(fmt.Sprintf("2024-01-%02d", i%28+1)), // time
			nonEmpty(cat),                                  // category (2 unique < 20% of 20 = 4)
			nonEmpty(fmt.Sprintf("%d", i*100)),             // numeric
		}
	}

	data := makeSheetData(
		[]string{"ID", "Date", "Category", "Amount"},
		rows,
	)
	score := CalculateAIQueryReadiness(data)
	assert.Equal(t, 100.0, score)
}

func TestCalculateAIQueryReadiness_NoRows(t *testing.T) {
	data := makeSheetData([]string{"A", "B"}, nil)
	score := CalculateAIQueryReadiness(data)
	assert.Equal(t, 0.0, score)
}

func TestCalculateAIQueryReadiness_OnlyColumnNames(t *testing.T) {
	// Only good column names → 20 points
	data := makeSheetData(
		[]string{"ID", "Name", "Value"},
		[][]upload.CellValue{
			{nonEmpty("abc"), nonEmpty("abc"), nonEmpty("abc")},
			{nonEmpty("abc"), nonEmpty("abc"), nonEmpty("abc")},
			{nonEmpty("abc"), nonEmpty("abc"), nonEmpty("abc")},
			{nonEmpty("abc"), nonEmpty("abc"), nonEmpty("abc")},
			{nonEmpty("abc"), nonEmpty("abc"), nonEmpty("abc")},
			{nonEmpty("abc"), nonEmpty("abc"), nonEmpty("abc")},
		},
	)
	// No identifier (all same), no time, category has 1 unique (not >1), no numeric
	// But column names are good → +20
	score := CalculateAIQueryReadiness(data)
	// identifier: unique ratio = 1/6 = 0.167 ≤ 0.8 → no
	// time: no dates → no
	// category: unique count = 1, need >1 → no
	// numeric: "abc" not numeric → no
	// column names: all valid → +20
	assert.Equal(t, 20.0, score)
}

// --- Scoring Tests ---

func TestCalculateTotalScore_DefaultWeights(t *testing.T) {
	indicators := IndicatorScores{
		RowCompleteness:    100,
		ColumnCompleteness: 100,
		FormatConsistency:  100,
		DuplicateSimilar:   100,
		TableStructure:     100,
		AIQueryReadiness:   100,
	}
	total, grade, err := CalculateTotalScore(indicators, DefaultWeights())
	assert.NoError(t, err)
	assert.InDelta(t, 100.0, total, 0.1)
	assert.Equal(t, "ready", grade)
}

func TestCalculateTotalScore_Conditional(t *testing.T) {
	indicators := IndicatorScores{
		RowCompleteness:    70,
		ColumnCompleteness: 70,
		FormatConsistency:  70,
		DuplicateSimilar:   70,
		TableStructure:     70,
		AIQueryReadiness:   70,
	}
	total, grade, err := CalculateTotalScore(indicators, DefaultWeights())
	assert.NoError(t, err)
	assert.InDelta(t, 70.0, total, 0.1)
	assert.Equal(t, "conditional", grade)
}

func TestCalculateTotalScore_NotReady(t *testing.T) {
	indicators := IndicatorScores{
		RowCompleteness:    30,
		ColumnCompleteness: 40,
		FormatConsistency:  50,
		DuplicateSimilar:   20,
		TableStructure:     30,
		AIQueryReadiness:   40,
	}
	total, grade, err := CalculateTotalScore(indicators, DefaultWeights())
	assert.NoError(t, err)
	assert.Less(t, total, 60.0)
	assert.Equal(t, "not_ready", grade)
}

func TestCalculateTotalScore_InvalidWeights(t *testing.T) {
	indicators := IndicatorScores{}
	weights := Weights{
		RowCompleteness:    0.5,
		ColumnCompleteness: 0.5,
		FormatConsistency:  0.5,
		DuplicateSimilar:   0.5,
		TableStructure:     0.5,
		AIQueryReadiness:   0.5,
	}
	_, _, err := CalculateTotalScore(indicators, weights)
	assert.Error(t, err)
}

func TestWeights_IsValid(t *testing.T) {
	assert.True(t, DefaultWeights().IsValid())

	invalid := Weights{
		RowCompleteness: 0.5,
	}
	assert.False(t, invalid.IsValid())
}

// --- Levenshtein Tests ---

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		a, b     string
		expected int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"kitten", "sitting", 3},
		{"hello", "hello", 0},
		{"hello", "hallo", 1},
		{"台北市", "台北", 1},
		{"Alice", "Alicee", 1},
	}
	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			assert.Equal(t, tt.expected, levenshteinDistance(tt.a, tt.b))
		})
	}
}

// --- Issues Detection Tests ---

func TestDetectIssues_HighSeverity(t *testing.T) {
	data := makeSheetData(
		[]string{"A", "B"},
		[][]upload.CellValue{
			{nonEmpty("1"), empty()},
			{empty(), nonEmpty("2")},
		},
	)
	scores := IndicatorScores{
		RowCompleteness:    40,
		ColumnCompleteness: 40,
		FormatConsistency:  40,
		DuplicateSimilar:   40,
		TableStructure:     40,
		AIQueryReadiness:   40,
	}
	issues := DetectIssues(data, scores)
	assert.NotEmpty(t, issues)
	// Should have High severity issues
	highCount := 0
	for _, issue := range issues {
		if issue.Severity == "High" {
			highCount++
		}
	}
	assert.Greater(t, highCount, 0)
}

func TestDetectIssues_NoIssues(t *testing.T) {
	data := makeSheetData(
		[]string{"A", "B"},
		[][]upload.CellValue{
			{nonEmpty("1"), nonEmpty("2")},
		},
	)
	scores := IndicatorScores{
		RowCompleteness:    100,
		ColumnCompleteness: 100,
		FormatConsistency:  100,
		DuplicateSimilar:   100,
		TableStructure:     100,
		AIQueryReadiness:   100,
	}
	issues := DetectIssues(data, scores)
	assert.Empty(t, issues)
}

// --- Edge Cases ---

// Verify score is bounded [0, 100] for all indicators
func TestIndicatorScoresBounded(t *testing.T) {
	data := makeSheetData(
		[]string{"A", "B", "C"},
		[][]upload.CellValue{
			{nonEmpty("1"), empty(), nonEmpty("2024-01-01")},
			{empty(), nonEmpty("text"), empty()},
			{nonEmpty("3"), nonEmpty("text"), nonEmpty("true")},
		},
	)

	rowScore := CalculateRowCompleteness(data)
	assert.GreaterOrEqual(t, rowScore, 0.0)
	assert.LessOrEqual(t, rowScore, 100.0)

	colScore, _ := CalculateColumnCompleteness(data)
	assert.GreaterOrEqual(t, colScore, 0.0)
	assert.LessOrEqual(t, colScore, 100.0)

	fmtScore := CalculateFormatConsistency(data)
	assert.GreaterOrEqual(t, fmtScore, 0.0)
	assert.LessOrEqual(t, fmtScore, 100.0)

	dupScore := CalculateDuplicateSimilar(data)
	assert.GreaterOrEqual(t, dupScore, 0.0)
	assert.LessOrEqual(t, dupScore, 100.0)

	structScore := CalculateTableStructure(data)
	assert.GreaterOrEqual(t, structScore, 0.0)
	assert.LessOrEqual(t, structScore, 100.0)

	aiScore := CalculateAIQueryReadiness(data)
	assert.GreaterOrEqual(t, aiScore, 0.0)
	assert.LessOrEqual(t, aiScore, 100.0)
}


func TestRowCompleteness_ShorterRowsThanColCount(t *testing.T) {
	data := &upload.SheetData{
		Headers:  []string{"A", "B", "C", "D"},
		Rows:     [][]upload.CellValue{{nonEmpty("1"), nonEmpty("2")}}, // only 2 cells for 4 cols
		RowCount: 1,
		ColCount: 4,
	}
	score := CalculateRowCompleteness(data)
	// 2/4 = 0.5 → × 100 = 50.0
	assert.InDelta(t, 50.0, score, 0.001)
}

func TestFormatConsistency_TieBreakByPriority(t *testing.T) {
	// A value like "113.1.5" matches ROC date format (priority higher than numeric)
	ft := DetectFormatType("113.1.5")
	assert.Equal(t, FormatDate, ft)
}

func TestCalculateTableStructure_MaxDeductions(t *testing.T) {
	// Build data with ALL deductions
	data := &upload.SheetData{
		Headers: []string{"Header1", "Header2"},
		Rows: [][]upload.CellValue{
			// Row 0: header-like (text, no repeats)
			{nonEmpty("Category"), nonEmpty("Description")},
			// Row 1: another header-like
			{nonEmpty("SubCat"), nonEmpty("Detail")},
			// Row 2: data with subtotal keyword
			{nonEmpty("合計"), nonEmpty("1000")},
			// Empty rows for multiple tables
			{empty(), empty()},
			{empty(), empty()},
			// Data after empty rows
			{nonEmpty("more data"), nonEmpty("values")},
		},
		RowCount: 6,
		ColCount: 2,
		MergedCells: []upload.MergedRange{
			{StartRow: 0, EndRow: 0, StartCol: 0, EndCol: 1},
		},
	}
	score := CalculateTableStructure(data)
	// merged: -20, multi-header: -20, subtotal: -15, multiple tables: -25 = 100-80=20
	// notes: text lengths likely won't trigger
	assert.GreaterOrEqual(t, score, 0.0)
	assert.LessOrEqual(t, score, 100.0)
	// Expected: at most 100 - 80 = 20 (merged + multi-header + subtotal + multiple tables)
	assert.LessOrEqual(t, score, 25.0)
}
