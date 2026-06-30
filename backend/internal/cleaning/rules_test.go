package cleaning

import (
	"testing"

	"github.com/safe-ai/excel-brushing-tool/internal/upload"
	"github.com/stretchr/testify/assert"
)

// --- DateNormalize tests ---

func TestDateNormalize_SlashFormat(t *testing.T) {
	data := &upload.SheetData{
		Headers:  []string{"日期", "名稱"},
		ColCount: 2,
		Rows: [][]upload.CellValue{
			{{Raw: "2024/01/15", IsEmpty: false}, {Raw: "Alice", IsEmpty: false}},
			{{Raw: "2024/12/05", IsEmpty: false}, {Raw: "Bob", IsEmpty: false}},
			{{Raw: "2023/3/9", IsEmpty: false}, {Raw: "Carol", IsEmpty: false}},
		},
		RowCount: 3,
	}

	var log []LogEntry
	DateNormalize(data, &log, "test-user")

	assert.Equal(t, "2024/01/15", data.Rows[0][0].Raw)
	assert.Equal(t, "2024/12/05", data.Rows[1][0].Raw)
	assert.Equal(t, "2023/03/09", data.Rows[2][0].Raw)
	assert.NotEmpty(t, log)
	assert.Equal(t, "date_normalize", log[0].OperationType)
}

func TestDateNormalize_ROCFormat(t *testing.T) {
	data := &upload.SheetData{
		Headers:  []string{"日期"},
		ColCount: 1,
		Rows: [][]upload.CellValue{
			{{Raw: "113.1.5", IsEmpty: false}},
			{{Raw: "112.12.31", IsEmpty: false}},
			{{Raw: "111.6.15", IsEmpty: false}},
		},
		RowCount: 3,
	}

	var log []LogEntry
	DateNormalize(data, &log, "test-user")

	assert.Equal(t, "2024/01/05", data.Rows[0][0].Raw)
	assert.Equal(t, "2023/12/31", data.Rows[1][0].Raw)
	assert.Equal(t, "2022/06/15", data.Rows[2][0].Raw)
	assert.Len(t, log, 1)
}

func TestDateNormalize_AlreadyNormalized(t *testing.T) {
	data := &upload.SheetData{
		Headers:  []string{"日期"},
		ColCount: 1,
		Rows: [][]upload.CellValue{
			{{Raw: "2024/01/15", IsEmpty: false}},
			{{Raw: "2024/12/05", IsEmpty: false}},
		},
		RowCount: 2,
	}

	var log []LogEntry
	DateNormalize(data, &log, "test-user")

	// Already in correct format — no log entries expected
	assert.Equal(t, "2024/01/15", data.Rows[0][0].Raw)
	assert.Equal(t, "2024/12/05", data.Rows[1][0].Raw)
	assert.Empty(t, log)
}

// --- Dedup tests ---

func TestDedup_RemovesDuplicates(t *testing.T) {
	data := &upload.SheetData{
		Headers:  []string{"A", "B"},
		ColCount: 2,
		Rows: [][]upload.CellValue{
			{{Raw: "a", IsEmpty: false}, {Raw: "1", IsEmpty: false}},
			{{Raw: "b", IsEmpty: false}, {Raw: "2", IsEmpty: false}},
			{{Raw: "a", IsEmpty: false}, {Raw: "1", IsEmpty: false}}, // duplicate of row 0
			{{Raw: "c", IsEmpty: false}, {Raw: "3", IsEmpty: false}},
			{{Raw: "b", IsEmpty: false}, {Raw: "2", IsEmpty: false}}, // duplicate of row 1
		},
		RowCount: 5,
	}

	var log []LogEntry
	Dedup(data, &log, "test-user")

	assert.Equal(t, 3, data.RowCount)
	assert.Equal(t, 3, len(data.Rows))
	assert.Equal(t, "a", data.Rows[0][0].Raw)
	assert.Equal(t, "b", data.Rows[1][0].Raw)
	assert.Equal(t, "c", data.Rows[2][0].Raw)
	assert.Len(t, log, 1)
	assert.Equal(t, "dedup", log[0].OperationType)
	assert.ElementsMatch(t, []int{2, 4}, log[0].AffectedRows)
}

func TestDedup_NoDuplicates(t *testing.T) {
	data := &upload.SheetData{
		Headers:  []string{"A"},
		ColCount: 1,
		Rows: [][]upload.CellValue{
			{{Raw: "x", IsEmpty: false}},
			{{Raw: "y", IsEmpty: false}},
			{{Raw: "z", IsEmpty: false}},
		},
		RowCount: 3,
	}

	var log []LogEntry
	Dedup(data, &log, "test-user")

	assert.Equal(t, 3, data.RowCount)
	assert.Empty(t, log)
}

// --- NameNormalize tests ---

func TestNameNormalize_ChineseSuffixes(t *testing.T) {
	data := &upload.SheetData{
		Headers:  []string{"公司名稱"},
		ColCount: 1,
		Rows: [][]upload.CellValue{
			{{Raw: "台灣科技股份有限公司", IsEmpty: false}},
			{{Raw: "台灣科技有限公司", IsEmpty: false}},
			{{Raw: "台灣科技", IsEmpty: false}},
			{{Raw: "其他企業", IsEmpty: false}},
		},
		RowCount: 4,
	}

	var log []LogEntry
	NameNormalize(data, &log, "test-user")

	// All variants of 台灣科技 should be unified to the longest: "台灣科技股份有限公司"
	assert.Equal(t, "台灣科技股份有限公司", data.Rows[0][0].Raw)
	assert.Equal(t, "台灣科技股份有限公司", data.Rows[1][0].Raw)
	assert.Equal(t, "台灣科技股份有限公司", data.Rows[2][0].Raw)
	assert.Equal(t, "其他企業", data.Rows[3][0].Raw) // unchanged
	assert.NotEmpty(t, log)
	assert.Equal(t, "name_normalize", log[0].OperationType)
}

func TestNameNormalize_EnglishSuffixes(t *testing.T) {
	data := &upload.SheetData{
		Headers:  []string{"Company"},
		ColCount: 1,
		Rows: [][]upload.CellValue{
			{{Raw: "Acme Corp.", IsEmpty: false}},
			{{Raw: "Acme Co.", IsEmpty: false}},
			{{Raw: "Acme", IsEmpty: false}},
		},
		RowCount: 3,
	}

	var log []LogEntry
	NameNormalize(data, &log, "test-user")

	// All should unify to longest variant "Acme Corp."
	assert.Equal(t, "Acme Corp.", data.Rows[0][0].Raw)
	assert.Equal(t, "Acme Corp.", data.Rows[1][0].Raw)
	assert.Equal(t, "Acme Corp.", data.Rows[2][0].Raw)
	assert.NotEmpty(t, log)
}

// --- SubtotalRemove tests ---

func TestSubtotalRemove_RemovesSubtotalRows(t *testing.T) {
	data := &upload.SheetData{
		Headers:  []string{"項目", "金額"},
		ColCount: 2,
		Rows: [][]upload.CellValue{
			{{Raw: "商品A", IsEmpty: false}, {Raw: "100", IsEmpty: false}},
			{{Raw: "商品B", IsEmpty: false}, {Raw: "200", IsEmpty: false}},
			{{Raw: "小計", IsEmpty: false}, {Raw: "300", IsEmpty: false}},
			{{Raw: "商品C", IsEmpty: false}, {Raw: "150", IsEmpty: false}},
			{{Raw: "合計", IsEmpty: false}, {Raw: "450", IsEmpty: false}},
		},
		RowCount: 5,
	}

	var log []LogEntry
	SubtotalRemove(data, &log, "test-user")

	assert.Equal(t, 3, data.RowCount)
	assert.Equal(t, "商品A", data.Rows[0][0].Raw)
	assert.Equal(t, "商品B", data.Rows[1][0].Raw)
	assert.Equal(t, "商品C", data.Rows[2][0].Raw)
	assert.Len(t, log, 1)
	assert.Equal(t, "subtotal_remove", log[0].OperationType)
	assert.ElementsMatch(t, []int{2, 4}, log[0].AffectedRows)
}

func TestSubtotalRemove_CaseInsensitive(t *testing.T) {
	data := &upload.SheetData{
		Headers:  []string{"Item", "Amount"},
		ColCount: 2,
		Rows: [][]upload.CellValue{
			{{Raw: "Item A", IsEmpty: false}, {Raw: "100", IsEmpty: false}},
			{{Raw: "TOTAL", IsEmpty: false}, {Raw: "100", IsEmpty: false}},
			{{Raw: "Item B", IsEmpty: false}, {Raw: "200", IsEmpty: false}},
			{{Raw: "Subtotal", IsEmpty: false}, {Raw: "200", IsEmpty: false}},
		},
		RowCount: 4,
	}

	var log []LogEntry
	SubtotalRemove(data, &log, "test-user")

	assert.Equal(t, 2, data.RowCount)
	assert.Equal(t, "Item A", data.Rows[0][0].Raw)
	assert.Equal(t, "Item B", data.Rows[1][0].Raw)
}

// --- Row operations tests ---

func TestFillNA_FillsEmptyCells(t *testing.T) {
	data := &upload.SheetData{
		Headers:  []string{"A", "B", "C"},
		ColCount: 3,
		Rows: [][]upload.CellValue{
			{{Raw: "val", IsEmpty: false}, {Raw: "", IsEmpty: true}, {Raw: "val2", IsEmpty: false}},
		},
		RowCount: 1,
	}

	var log []LogEntry
	err := FillNA(data, 0, &log, "test-user")

	assert.NoError(t, err)
	assert.Equal(t, "val", data.Rows[0][0].Raw)
	assert.Equal(t, "N/A", data.Rows[0][1].Raw)
	assert.False(t, data.Rows[0][1].IsEmpty)
	assert.Equal(t, "val2", data.Rows[0][2].Raw)
	assert.Len(t, log, 1)
	assert.Equal(t, "fill_na", log[0].OperationType)
}

func TestFillNA_OutOfBounds(t *testing.T) {
	data := &upload.SheetData{
		Headers:  []string{"A"},
		ColCount: 1,
		Rows: [][]upload.CellValue{
			{{Raw: "val", IsEmpty: false}},
		},
		RowCount: 1,
	}

	var log []LogEntry
	err := FillNA(data, 5, &log, "test-user")

	assert.ErrorIs(t, err, ErrRowOutOfBounds)
	assert.Empty(t, log)
}

func TestDeleteRow_RemovesRow(t *testing.T) {
	data := &upload.SheetData{
		Headers:  []string{"A"},
		ColCount: 1,
		Rows: [][]upload.CellValue{
			{{Raw: "row0", IsEmpty: false}},
			{{Raw: "row1", IsEmpty: false}},
			{{Raw: "row2", IsEmpty: false}},
		},
		RowCount: 3,
	}

	var log []LogEntry
	err := DeleteRow(data, 1, &log, "test-user")

	assert.NoError(t, err)
	assert.Equal(t, 2, data.RowCount)
	assert.Equal(t, "row0", data.Rows[0][0].Raw)
	assert.Equal(t, "row2", data.Rows[1][0].Raw)
	assert.Len(t, log, 1)
	assert.Equal(t, "delete_row", log[0].OperationType)
}

func TestDeleteRow_OutOfBounds(t *testing.T) {
	data := &upload.SheetData{
		Headers:  []string{"A"},
		ColCount: 1,
		Rows: [][]upload.CellValue{
			{{Raw: "val", IsEmpty: false}},
		},
		RowCount: 1,
	}

	var log []LogEntry
	err := DeleteRow(data, -1, &log, "test-user")

	assert.ErrorIs(t, err, ErrRowOutOfBounds)
	assert.Empty(t, log)
}

// --- Helper function tests ---

func TestNormalizeDate_Various(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		ok       bool
	}{
		{"2024/01/15", "2024/01/15", true},
		{"2024/1/5", "2024/01/05", true},
		{"2024-01-15", "2024/01/15", true},
		{"113.1.5", "2024/01/05", true},
		{"112.12.31", "2023/12/31", true},
		{"02-27-19", "2019/02/27", true},
		{"12/31/20", "2020/12/31", true},
		{"01-06-20", "2020/01/06", true},
		{"not-a-date", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		result, ok := normalizeDate(tt.input)
		assert.Equal(t, tt.ok, ok, "input: %q", tt.input)
		if ok {
			assert.Equal(t, tt.expected, result, "input: %q", tt.input)
		}
	}
}

func TestRemoveSuffixes(t *testing.T) {
	suffixes := []string{"股份有限公司", "有限公司", "公司", "Corp.", "Inc.", "Ltd.", "Co."}

	tests := []struct {
		input    string
		expected string
	}{
		{"台灣科技股份有限公司", "台灣科技"},
		{"台灣科技有限公司", "台灣科技"},
		{"台灣科技公司", "台灣科技"},
		{"Acme Corp.", "Acme"},
		{"Acme Inc.", "Acme"},
		{"Plain Name", "Plain Name"},
	}

	for _, tt := range tests {
		result := removeSuffixes(tt.input, suffixes)
		assert.Equal(t, tt.expected, result, "input: %q", tt.input)
	}
}
