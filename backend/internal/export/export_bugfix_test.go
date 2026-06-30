package export

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"

	"github.com/safe-ai/excel-brushing-tool/internal/assessment"
	"github.com/safe-ai/excel-brushing-tool/internal/cleaning"
	"github.com/safe-ai/excel-brushing-tool/pkg/config"
)

// =============================================================================
// Bug Condition Exploration Tests (Task 1)
// =============================================================================
// These tests encode the EXPECTED (correct) behavior.
// They are expected to FAIL on the unfixed code, proving the bugs exist.
// After the fix is applied, these tests should PASS.
// =============================================================================

// --- Generators ---

// genOriginalFilename generates realistic filenames including Chinese characters
func genOriginalFilename(t *rapid.T) string {
	// Generate filenames with mix of Chinese and ASCII characters
	choice := rapid.IntRange(0, 3).Draw(t, "filename_choice")
	switch choice {
	case 0:
		// Chinese filename
		prefixes := []string{"客戶名單", "銷售報表", "庫存資料", "員工清冊", "訂單明細"}
		idx := rapid.IntRange(0, len(prefixes)-1).Draw(t, "prefix_idx")
		return prefixes[idx] + ".xlsx"
	case 1:
		// ASCII filename
		name := rapid.StringMatching(`[A-Za-z0-9_]{3,20}`).Draw(t, "ascii_name")
		return name + ".xlsx"
	case 2:
		// Mixed filename
		name := rapid.StringMatching(`[A-Za-z]{2,8}`).Draw(t, "mixed_name")
		return name + "_報表.xlsx"
	default:
		// CSV filename
		name := rapid.StringMatching(`[A-Za-z0-9]{3,15}`).Draw(t, "csv_name")
		return name + ".csv"
	}
}

// genCleaningSession generates a CleaningSession with OriginalFilename set
func genCleaningSession(t *rapid.T) *cleaning.CleaningSession {
	numEntries := rapid.IntRange(1, 10).Draw(t, "num_entries")
	logEntries := make([]cleaning.LogEntry, numEntries)

	opTypes := []string{"dedup", "date_normalize", "name_normalize", "subtotal_remove", "delete_row", "fill_na"}

	for i := 0; i < numEntries; i++ {
		opIdx := rapid.IntRange(0, len(opTypes)-1).Draw(t, fmt.Sprintf("op_%d", i))
		numRows := rapid.IntRange(0, 20).Draw(t, fmt.Sprintf("num_rows_%d", i))
		affectedRows := make([]int, numRows)
		for j := 0; j < numRows; j++ {
			affectedRows[j] = rapid.IntRange(1, 1000).Draw(t, fmt.Sprintf("row_%d_%d", i, j))
		}

		logEntries[i] = cleaning.LogEntry{
			OperationType: opTypes[opIdx],
			AffectedRows:  affectedRows,
			Timestamp:     time.Date(2024, time.Month(rapid.IntRange(1, 12).Draw(t, fmt.Sprintf("month_%d", i))), rapid.IntRange(1, 28).Draw(t, fmt.Sprintf("day_%d", i)), rapid.IntRange(0, 23).Draw(t, fmt.Sprintf("hour_%d", i)), rapid.IntRange(0, 59).Draw(t, fmt.Sprintf("min_%d", i)), rapid.IntRange(0, 59).Draw(t, fmt.Sprintf("sec_%d", i)), 0, time.UTC),
			OperatorID:    rapid.StringMatching(`[a-z0-9]{8}`).Draw(t, fmt.Sprintf("operator_%d", i)),
			Details:       rapid.StringMatching(`[A-Za-z0-9 ]{0,30}`).Draw(t, fmt.Sprintf("details_%d", i)),
		}
	}

	return &cleaning.CleaningSession{
		ID:           uuid.New(),
		AssessmentID: uuid.New(),
		UserID:       uuid.New(),
		RulesApplied: []string{"dedup", "date_normalize"},
		RowsBefore:   100,
		RowsAfter:    90,
		ScoreBefore:  60.0,
		ScoreAfter:   85.0,
		CleaningLog:  logEntries,
		CreatedAt:    time.Now(),
	}
}

// --- Bug 1: Excel Filename Property Test ---
// **Validates: Requirements 1.1, 1.2, 2.1, 2.2**
//
// Property: When a CleaningSession has an OriginalFilename set, the DownloadExcel
// handler SHOULD produce a Content-Disposition header containing
// "{stripExtension(OriginalFilename)}_refined.xlsx".
//
// On unfixed code, this test will FAIL because the handler always uses "refined.xlsx".
func TestBugCondition_ExcelFilename(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		originalFilename := genOriginalFilename(rt)

		// The expected filename after fix: strip extension, add _refined.xlsx
		baseName := strings.TrimSuffix(originalFilename, ".xlsx")
		baseName = strings.TrimSuffix(baseName, ".csv")
		expectedFilename := baseName + "_refined.xlsx"

		// Call the actual GenerateExcelFilename helper function
		actualFilename := GenerateExcelFilename(originalFilename)

		// Assert the expected behavior (this WILL FAIL on unfixed code)
		assert.Equal(t, expectedFilename, actualFilename,
			"Bug 1 confirmed: Content-Disposition should contain %q (based on OriginalFilename=%q) but got %q",
			expectedFilename, originalFilename, actualFilename)
	})
}

// --- Bug 2: PDF Font Missing Property Test ---
// **Validates: Requirements 1.3, 1.4, 2.3, 2.4**
//
// Property: When the configured FontPath does not exist on disk, GeneratePDF
// SHOULD return an error containing "字型" (font), rather than silently producing
// a garbled PDF.
//
// On unfixed code, this test will FAIL because GeneratePDF silently falls back to Helvetica.
func TestBugCondition_PDFFontMissing(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Generate a non-existent font path
		nonExistentPath := fmt.Sprintf("/tmp/nonexistent_fonts_%s/NotoSansTC-Regular.ttf",
			rapid.StringMatching(`[a-z0-9]{8}`).Draw(rt, "path_suffix"))

		cfg := &config.Config{
			Report: config.ReportConfig{
				FontPath:     nonExistentPath,
				FontBoldPath: nonExistentPath,
				Colors: config.ReportColors{
					Primary: "#1a1f2e",
					Accent:  "#2b6cb0",
					Green:   "#15803d",
					Amber:   "#b45309",
					Rose:    "#b42318",
				},
			},
		}

		session := &cleaning.CleaningSession{
			ID:           uuid.New(),
			AssessmentID: uuid.New(),
			UserID:       uuid.New(),
			RulesApplied: []string{"dedup"},
			RowsBefore:   50,
			RowsAfter:    45,
			ScoreBefore:  65.0,
			ScoreAfter:   80.0,
			CleaningLog:  []cleaning.LogEntry{},
			CreatedAt:    time.Now(),
		}

		reportData := &PDFReportData{
			Session: session,
			Assessment: &assessment.Assessment{
				ID:         uuid.New(),
				TotalScore: 80.0,
				Status:     "ready",
				Issues:     []assessment.Issue{},
				WeightsSnapshot: assessment.Weights{
					RowCompleteness:    0.20,
					ColumnCompleteness: 0.20,
					FormatConsistency:  0.15,
					DuplicateSimilar:   0.10,
					TableStructure:     0.15,
					AIQueryReadiness:   0.20,
				},
			},
			Issues: []assessment.Issue{},
		}

		// Create a temp output directory
		outputDir, err := os.MkdirTemp("", "pdf_test_*")
		assert.NoError(t, err)
		defer os.RemoveAll(outputDir)

		// Call GeneratePDF with non-existent font path
		_, err = GeneratePDF(reportData, cfg, outputDir)

		// EXPECTED BEHAVIOR (after fix): should return error containing "字型"
		// CURRENT BEHAVIOR (unfixed): returns nil error, silently produces garbled PDF
		assert.Error(t, err,
			"Bug 2 confirmed: GeneratePDF should return error when font file does not exist at %q", nonExistentPath)
		if err != nil {
			assert.Contains(t, err.Error(), "字型",
				"Bug 2 confirmed: error message should contain '字型' (font), got: %v", err)
		}
	})
}

// --- Bug 3: Log Format Property Test ---
// **Validates: Requirements 1.5, 2.5, 2.6**
//
// Property: GenerateLog output SHOULD be in human-readable line format where each
// line matches `[YYYY-MM-DD HH:MM:SS] description` pattern and does NOT contain
// raw JSON field names like "operation_type" or "affected_rows".
//
// On unfixed code, this test will FAIL because GenerateLog outputs raw JSON.
func TestBugCondition_LogFormat(t *testing.T) {
	linePattern := regexp.MustCompile(`^\[\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\] .+`)

	rapid.Check(t, func(rt *rapid.T) {
		session := genCleaningSession(rt)

		// Create temp output directory
		outputDir, err := os.MkdirTemp("", "log_test_*")
		assert.NoError(t, err)
		defer os.RemoveAll(outputDir)

		// Call GenerateLog
		filePath, err := GenerateLog(session, outputDir)
		assert.NoError(t, err, "GenerateLog should not return error")

		// Read the generated file
		content, err := os.ReadFile(filePath)
		assert.NoError(t, err, "Should be able to read generated log file")

		contentStr := string(content)

		// EXPECTED BEHAVIOR (after fix):
		// - Each non-empty line matches [YYYY-MM-DD HH:MM:SS] description
		// - No raw JSON field names present
		//
		// CURRENT BEHAVIOR (unfixed):
		// - Output is JSON with "operation_type", "affected_rows" etc.

		// Check that output does NOT contain JSON field names
		assert.NotContains(t, contentStr, `"operation_type"`,
			"Bug 3 confirmed: log output should not contain raw JSON field 'operation_type'")
		assert.NotContains(t, contentStr, `"affected_rows"`,
			"Bug 3 confirmed: log output should not contain raw JSON field 'affected_rows'")

		// Check that each non-empty line matches the human-readable pattern
		lines := strings.Split(contentStr, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			assert.Regexp(t, linePattern, trimmed,
				"Bug 3 confirmed: each log line should match [YYYY-MM-DD HH:MM:SS] pattern, got: %q", trimmed)
		}
	})
}
