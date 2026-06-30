package upload

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCSV_Simple(t *testing.T) {
	// 建立臨時 CSV 檔案
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "test.csv")
	content := "Name,Age,City\nAlice,30,Taipei\nBob,25,Kaohsiung\n"
	err := os.WriteFile(csvPath, []byte(content), 0644)
	require.NoError(t, err)

	result, err := ParseCSV(csvPath)
	require.NoError(t, err)

	assert.Equal(t, 2, result.RowCount) // 2 data rows (excluding header)
	assert.Equal(t, 3, result.ColCount)
	assert.Equal(t, []string{"Sheet1"}, result.SheetNames)
	assert.Nil(t, result.MergedCells)
}

func TestParseCSV_UTF8BOM(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "bom.csv")
	// UTF-8 BOM + content
	bom := []byte{0xEF, 0xBB, 0xBF}
	content := append(bom, []byte("Col1,Col2\nA,B\n")...)
	err := os.WriteFile(csvPath, content, 0644)
	require.NoError(t, err)

	result, err := ParseCSV(csvPath)
	require.NoError(t, err)

	assert.Equal(t, 1, result.RowCount) // 1 data row (excluding header)
	assert.Equal(t, 2, result.ColCount)
}

func TestParseCSV_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "empty.csv")
	err := os.WriteFile(csvPath, []byte(""), 0644)
	require.NoError(t, err)

	result, err := ParseCSV(csvPath)
	require.NoError(t, err)

	assert.Equal(t, 0, result.RowCount)
	assert.Equal(t, 0, result.ColCount)
}

func TestParseCSV_CorruptedFile(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "missing.csv")

	_, err := ParseCSV(csvPath)
	assert.ErrorIs(t, err, ErrFileCorrupted)
}

func TestLoadSheetDataCSV(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "data.csv")
	content := "Name,Age,City\nAlice,30,Taipei\nBob,,Kaohsiung\n"
	err := os.WriteFile(csvPath, []byte(content), 0644)
	require.NoError(t, err)

	data, err := LoadSheetData(csvPath, "Sheet1", "csv")
	require.NoError(t, err)

	assert.Equal(t, []string{"Name", "Age", "City"}, data.Headers)
	assert.Equal(t, 2, data.RowCount)
	assert.Equal(t, 3, data.ColCount)

	// 第一列：Alice, 30, Taipei — 都非空
	assert.False(t, data.Rows[0][0].IsEmpty)
	assert.Equal(t, "Alice", data.Rows[0][0].Raw)

	// 第二列：Bob, (empty), Kaohsiung
	assert.False(t, data.Rows[1][0].IsEmpty)
	assert.True(t, data.Rows[1][1].IsEmpty)
	assert.False(t, data.Rows[1][2].IsEmpty)
}

func TestLoadSheetDataCSV_WithBOM(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "bom_data.csv")
	bom := []byte{0xEF, 0xBB, 0xBF}
	content := append(bom, []byte("ID,Value\n1,Hello\n2,World\n")...)
	err := os.WriteFile(csvPath, content, 0644)
	require.NoError(t, err)

	data, err := LoadSheetData(csvPath, "Sheet1", "csv")
	require.NoError(t, err)

	assert.Equal(t, []string{"ID", "Value"}, data.Headers)
	assert.Equal(t, 2, data.RowCount)
}

func TestIsEmptyValue(t *testing.T) {
	tests := []struct {
		val      string
		expected bool
	}{
		{"", true},
		{" ", true},
		{"  \t  ", true},
		{"\n", true},
		{"hello", false},
		{"0", false},
		{" x ", false},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, isEmptyValue(tt.val), "isEmptyValue(%q)", tt.val)
	}
}

func TestGetFileExtension(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"test.xlsx", "xlsx"},
		{"test.CSV", "csv"},
		{"my.file.xlsx", "xlsx"},
		{"noext", ""},
		{"report.PDF", "pdf"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, getFileExtension(tt.filename), "getFileExtension(%q)", tt.filename)
	}
}

func TestLoadSheetData_UnsupportedFormat(t *testing.T) {
	_, err := LoadSheetData("/tmp/test.pdf", "Sheet1", "pdf")
	assert.ErrorIs(t, err, ErrUnsupportedFormat)
}
