package export

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/xuri/excelize/v2"
	"pgregory.net/rapid"

	"github.com/safe-ai/excel-brushing-tool/internal/assessment"
	"github.com/safe-ai/excel-brushing-tool/internal/cleaning"
	"github.com/safe-ai/excel-brushing-tool/pkg/config"
)

// --- Generators ---

// genHeaders generates a random slice of non-empty header strings (1-10 columns)
func genHeaders(t *rapid.T) []string {
	count := rapid.IntRange(1, 10).Draw(t, "numCols")
	headers := make([]string, count)
	for i := range headers {
		headers[i] = rapid.StringMatching(`[A-Za-z\x{4e00}-\x{9fff}]{1,8}`).Draw(t, fmt.Sprintf("header_%d", i))
	}
	return headers
}

// genRows generates random row data matching given column count (1-30 rows)
// Values are always non-empty to avoid ambiguity with excelize's trailing cell handling
func genRows(t *rapid.T, numCols int) [][]string {
	numRows := rapid.IntRange(1, 30).Draw(t, "numRows")
	rows := make([][]string, numRows)
	for i := range rows {
		row := make([]string, numCols)
		for j := range row {
			row[j] = rapid.StringMatching(`[A-Za-z0-9\x{4e00}-\x{9fff}]{1,12}`).Draw(t, fmt.Sprintf("cell_%d_%d", i, j))
		}
		rows[i] = row
	}
	return rows
}

// genPreservationSession generates a random CleaningSession with a valid cleaning log
func genPreservationSession(t *rapid.T) *cleaning.CleaningSession {
	numEntries := rapid.IntRange(1, 10).Draw(t, "numLogEntries")
	log := make([]cleaning.LogEntry, numEntries)

	opTypes := []string{"dedup", "date_normalize", "name_normalize", "subtotal_remove", "delete_row", "fill_na"}

	for i := range log {
		opIdx := rapid.IntRange(0, len(opTypes)-1).Draw(t, fmt.Sprintf("opType_%d", i))
		numAffected := rapid.IntRange(0, 20).Draw(t, fmt.Sprintf("numAffected_%d", i))
		affected := make([]int, numAffected)
		for j := range affected {
			affected[j] = rapid.IntRange(1, 1000).Draw(t, fmt.Sprintf("row_%d_%d", i, j))
		}

		log[i] = cleaning.LogEntry{
			OperationType: opTypes[opIdx],
			AffectedRows:  affected,
			Timestamp:     time.Date(2024, time.Month(rapid.IntRange(1, 12).Draw(t, fmt.Sprintf("month_%d", i))), rapid.IntRange(1, 28).Draw(t, fmt.Sprintf("day_%d", i)), rapid.IntRange(0, 23).Draw(t, fmt.Sprintf("hour_%d", i)), rapid.IntRange(0, 59).Draw(t, fmt.Sprintf("min_%d", i)), rapid.IntRange(0, 59).Draw(t, fmt.Sprintf("sec_%d", i)), 0, time.UTC),
			OperatorID:    rapid.StringMatching(`[a-z0-9\-]{5,20}`).Draw(t, fmt.Sprintf("operator_%d", i)),
			Details:       rapid.StringMatching(`[A-Za-z0-9 \x{4e00}-\x{9fff}]{0,30}`).Draw(t, fmt.Sprintf("details_%d", i)),
		}
	}

	return &cleaning.CleaningSession{
		ID:           uuid.New(),
		AssessmentID: uuid.New(),
		UserID:       uuid.New(),
		RulesApplied: []string{"dedup", "date_normalize"},
		RowsBefore:   100,
		RowsAfter:    90,
		ScoreBefore:  65.0,
		ScoreAfter:   82.0,
		CleaningLog:  log,
		CreatedAt:    time.Now(),
	}
}

// --- Property 2a: Excel content preservation ---
// For all random (headers, rows) combinations, GenerateExcel output xlsx content matches input.
// **Validates: Requirements 3.1**
func TestPreservation_ExcelContentMatchesInput(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		headers := genHeaders(t)
		rows := genRows(t, len(headers))

		session := &cleaning.CleaningSession{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
		}

		// Create temp directory
		tmpDir, err := os.MkdirTemp("", "excel_preservation_*")
		assert.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Generate Excel
		filePath, err := GenerateExcel(session, headers, rows, tmpDir)
		assert.NoError(t, err)
		assert.NotEmpty(t, filePath)

		// Verify file exists
		_, err = os.Stat(filePath)
		assert.NoError(t, err)

		// Read back the xlsx and verify content
		f, err := excelize.OpenFile(filePath)
		assert.NoError(t, err)
		defer f.Close()

		sheetName := f.GetSheetName(0)
		allRows, err := f.GetRows(sheetName)
		assert.NoError(t, err)

		// Verify headers (first row)
		assert.GreaterOrEqual(t, len(allRows), 1, "Excel must have at least the header row")
		for i, h := range headers {
			if i < len(allRows[0]) {
				assert.Equal(t, h, allRows[0][i], "Header mismatch at column %d", i)
			}
		}

		// Verify data rows
		assert.Equal(t, len(rows)+1, len(allRows), "Row count mismatch (header + data)")
		for rowIdx, row := range rows {
			excelRow := allRows[rowIdx+1]
			for colIdx, val := range row {
				if colIdx < len(excelRow) {
					assert.Equal(t, val, excelRow[colIdx],
						"Cell mismatch at row %d, col %d", rowIdx, colIdx)
				} else if val != "" {
					// If excel row is shorter, the value should have been empty
					t.Fatalf("Expected value %q at row %d col %d but excel row is shorter", val, rowIdx, colIdx)
				}
			}
		}

		// Verify column widths are set (between 10 and 50 as per implementation)
		for colIdx := range headers {
			colName, _ := excelize.ColumnNumberToName(colIdx + 1)
			width, err := f.GetColWidth(sheetName, colName)
			assert.NoError(t, err)
			assert.GreaterOrEqual(t, width, 10.0, "Column %d width should be >= 10", colIdx)
			assert.LessOrEqual(t, width, 50.0, "Column %d width should be <= 50", colIdx)
		}
	})
}

// --- Property 2b: PDF generation succeeds when font exists ---
// For all valid configs (font exists), GeneratePDF returns nil error and output file exists.
// **Validates: Requirements 3.2**
func TestPreservation_PDFGeneratesWhenFontExists(t *testing.T) {
	// fpdf.New("P","mm","A4","") internally sets fontpath to "." which causes
	// path.Join(".", absolutePath) to strip the leading "/".
	// Therefore we must use a relative font path from the test working directory
	// (which is the package directory: backend/internal/export/).

	// Look for a system font we can copy into testdata/
	systemFontPaths := []string{
		"/System/Library/Fonts/Geneva.ttf",
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf",
	}

	var srcFont string
	for _, fp := range systemFontPaths {
		if _, err := os.Stat(fp); err == nil {
			srcFont = fp
			break
		}
	}

	if srcFont == "" {
		t.Skip("No suitable font file found for PDF preservation test; skipping")
	}

	// Place font in testdata/fonts/ relative to the test CWD (package directory)
	testFontDir := filepath.Join("testdata", "fonts")
	if err := os.MkdirAll(testFontDir, 0755); err != nil {
		t.Fatal(err)
	}
	testFontPath := filepath.Join(testFontDir, "TestFont.ttf")

	// Copy font only if not already there
	if _, err := os.Stat(testFontPath); err != nil {
		fontData, err := os.ReadFile(srcFont)
		if err != nil {
			t.Skip("Cannot read font file:", err)
		}
		if err := os.WriteFile(testFontPath, fontData, 0644); err != nil {
			t.Skip("Cannot write font file:", err)
		}
	}
	// Cleanup after test
	defer os.RemoveAll("testdata")

	// Create a temp directory for PDF outputs
	tmpDir, err := os.MkdirTemp("", "pdf_preservation_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	rapid.Check(t, func(rt *rapid.T) {
		cfg := &config.Config{
			Report: config.ReportConfig{
				Colors: config.ReportColors{
					Primary: "#1a1f2e",
					Accent:  "#2b6cb0",
					Green:   "#15803d",
					Amber:   "#b45309",
					Rose:    "#b42318",
				},
				FontPath:     testFontPath, // relative path works with fpdf
				FontBoldPath: testFontPath, // use same font for bold to simplify test
			},
		}

		session := &cleaning.CleaningSession{
			ID:           uuid.New(),
			AssessmentID: uuid.New(),
			UserID:       uuid.New(),
			RulesApplied: []string{"dedup"},
			RowsBefore:   rapid.IntRange(10, 1000).Draw(rt, "rowsBefore"),
			RowsAfter:    rapid.IntRange(5, 500).Draw(rt, "rowsAfter"),
			ScoreBefore:  float64(rapid.IntRange(30, 90).Draw(rt, "scoreBefore")),
			ScoreAfter:   float64(rapid.IntRange(50, 100).Draw(rt, "scoreAfter")),
			CleaningLog:  []cleaning.LogEntry{},
			CreatedAt:    time.Now(),
		}

		assess := &assessment.Assessment{
			ID:                 session.AssessmentID,
			TotalScore:         session.ScoreAfter,
			RowCompleteness:    float64(rapid.IntRange(50, 100).Draw(rt, "rc")),
			ColumnCompleteness: float64(rapid.IntRange(50, 100).Draw(rt, "cc")),
			FormatConsistency:  float64(rapid.IntRange(50, 100).Draw(rt, "fc")),
			DuplicateSimilar:   float64(rapid.IntRange(50, 100).Draw(rt, "ds")),
			TableStructure:     float64(rapid.IntRange(50, 100).Draw(rt, "ts")),
			AIQueryReadiness:   float64(rapid.IntRange(50, 100).Draw(rt, "aqr")),
			WeightsSnapshot: assessment.Weights{
				RowCompleteness:    0.20,
				ColumnCompleteness: 0.20,
				FormatConsistency:  0.15,
				DuplicateSimilar:   0.10,
				TableStructure:     0.15,
				AIQueryReadiness:   0.20,
			},
			Status: rapid.SampledFrom([]string{"ready", "conditional", "not_ready"}).Draw(rt, "status"),
			Issues: []assessment.Issue{},
		}

		data := &PDFReportData{
			Session:    session,
			Assessment: assess,
			Issues:     assess.Issues,
		}

		outputDir := filepath.Join(tmpDir, uuid.New().String())

		pdfPath, err := GeneratePDF(data, cfg, outputDir)
		assert.NoError(t, err, "GeneratePDF should not return error when font exists")
		assert.NotEmpty(t, pdfPath)

		if pdfPath != "" {
			// Verify file exists and is non-empty
			info, err := os.Stat(pdfPath)
			assert.NoError(t, err, "PDF file should exist")
			if err == nil {
				assert.Greater(t, info.Size(), int64(0), "PDF file should be non-empty")
			}
		}
	})
}

// --- Property 2c: Log output contains all entry info ---
// For all random CleaningSessions (various OperationType, different AffectedRows lengths),
// GenerateLog output contains all entry info.
// **Validates: Requirements 3.3**
func TestPreservation_LogContainsAllEntryInfo(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		session := genPreservationSession(rt)

		// Create temp directory
		tmpDir, err := os.MkdirTemp("", "log_preservation_*")
		assert.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Generate log
		logPath, err := GenerateLog(session, tmpDir)
		assert.NoError(t, err)
		assert.NotEmpty(t, logPath)

		// Read log content
		content, err := os.ReadFile(logPath)
		assert.NoError(t, err)
		logContent := string(content)

		// Verify each LogEntry's information is present in the output
		for i, entry := range session.CleaningLog {
			// Check OperationType is present via its human-readable label OR raw string
			label := operationTypeLabel(entry.OperationType)
			assert.True(t, strings.Contains(logContent, label),
				"Log entry %d: OperationType label %q (from %q) not found in output", i, label, entry.OperationType)

			// Check Timestamp info is present (year-month-day at minimum)
			timestampStr := entry.Timestamp.Format("2006-01-02")
			assert.True(t, strings.Contains(logContent, timestampStr),
				"Log entry %d: Timestamp date %q not found in output", i, timestampStr)

			// Check AffectedRows - each row number should appear in the output
			for _, rowNum := range entry.AffectedRows {
				rowStr := fmt.Sprintf("%d", rowNum)
				assert.True(t, strings.Contains(logContent, rowStr),
					"Log entry %d: AffectedRow %d not found in output", i, rowNum)
			}
		}
	})
}
