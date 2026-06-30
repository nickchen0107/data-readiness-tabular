# Implementation Plan

## Overview

Fix four issue card rendering bugs: (1) description first line as paragraph, (2) gap row highlights removal, (3) empty header amber background, (4) per-column format groups with labels. Uses exploratory bugfix workflow: write tests before fix to confirm bugs, preserve existing behavior, then implement fixes.

## Tasks

- [x] 1. Write bug condition exploration tests (backend)
  - **Property 1: Bug Condition** - Gap Row Highlights & Format Consistency Grouping
  - **CRITICAL**: This test MUST FAIL on unfixed code — failure confirms the bugs exist
  - **DO NOT attempt to fix the test or the code when it fails**
  - **NOTE**: This test encodes the expected behavior — it will validate the fix when it passes after implementation
  - **GOAL**: Surface counterexamples that demonstrate Bug 2 and Bug 4 exist in the backend
  - **Scoped PBT Approach**: Use `pgregory.net/rapid` to generate SheetData inputs
  - **Bug 2 — Gap Row Not Highlighted**:
    - Generate multi-block SheetData (≥2 data blocks separated by empty rows)
    - Call `buildSingleStructureExamples(data, "多表格混在同一 sheet")`
    - Assert gap rows (label="（空白列）") have `Highlights == nil`
    - `isBugCondition`: example.Label == "（空白列）" AND example.Highlights != nil
    - Expected: gap row Highlights is nil (will FAIL on unfixed code because highlights contain all column indices)
  - **Bug 4 — Per-Column Format Groups**:
    - Generate SheetData with ≥2 columns having mixed formats (e.g., col A: 80% numeric + 20% text, col B: 70% date + 30% text)
    - Call `buildFormatConsistencyExamples(data)`
    - Assert number of distinct Label groups == min(countMixedFormatColumns, 5)
    - Assert each example has non-nil `FormatLabels` with valid format strings
    - `isBugCondition`: countMixedFormatColumns(data) >= 1 AND indicator == "format_consistency"
    - Expected: multiple label groups (will FAIL on unfixed code because only first mixed column is used)
  - Run tests on UNFIXED code
  - **EXPECTED OUTCOME**: Tests FAIL (this is correct — proves bugs exist)
  - Document counterexamples found (e.g., "gap row Highlights=[0,1,2,3,4]" and "only 1 label group for 3 mixed columns")
  - Mark task complete when tests are written, run, and failures are documented
  - Test file: `backend/internal/assessment/pbt_issue_card_test.go`
  - Test command: `cd backend && go test ./internal/assessment/ -v -run TestPBT_BugCondition_IssueCard`
  - _Requirements: 1.2, 1.4, 2.2, 2.4_

- [x] 2. Write bug condition exploration tests (frontend)
  - **Property 1: Bug Condition** - Description Rendering & Header Styling
  - **CRITICAL**: This test MUST FAIL on unfixed code — failure confirms the bugs exist
  - **DO NOT attempt to fix the test or the code when it fails**
  - **GOAL**: Surface counterexamples that demonstrate Bug 1 and Bug 3 exist in the frontend
  - **Scoped PBT Approach**: Use `fast-check` to generate descriptions with `\n` and header highlight scenarios
  - **Bug 1 — Description First Line as Paragraph**:
    - Generate descriptions containing `\n` (e.g., "以下欄位格式不一致：\nCol A\nCol B")
    - Render the issue card description section
    - Assert the first line is rendered as a `<div>` paragraph, NOT as a `<li>`
    - `isBugCondition`: description.includes("\n")
    - Expected: first line is paragraph (will FAIL on unfixed code — all lines are `<li>`)
  - **Bug 3 — Empty Header Amber Background**:
    - Generate IssueExample with row_number=1, highlights containing cell indices
    - Render the header row `<th>` cells
    - Assert highlighted header cells have amber background (not red border on grey)
    - `isBugCondition`: example.row_number === 1 AND cellIndex in highlights
    - Expected: amber background (will FAIL on unfixed code — uses red border)
  - Run tests on UNFIXED code
  - **EXPECTED OUTCOME**: Tests FAIL (this is correct — proves bugs exist)
  - Document counterexamples found
  - Test file: `frontend/src/pages/AssessmentPage.test.tsx`
  - Test command: `cd frontend && npx vitest --run src/pages/AssessmentPage.test.tsx`
  - _Requirements: 1.1, 1.3, 2.1, 2.3_

- [x] 3. Write preservation property tests (backend — BEFORE implementing fix)
  - **Property 2: Preservation** - Non-Bug Backend Behavior Unchanged
  - **IMPORTANT**: Follow observation-first methodology
  - **Observe on UNFIXED code**:
    - Call `buildSingleStructureExamples(data, "合併儲存格")` → observe highlight behavior (non-gap rows highlighted as before)
    - Call `buildSingleStructureExamples(data, "小計列")` → observe examples unchanged
    - Generate SheetData with all columns at ≥80% format consistency → observe `buildFormatConsistencyExamples` returns empty/nil
    - Generate examples for non-format-consistency indicators → observe no `FormatLabels` field present
  - **Write property-based tests capturing observed behavior**:
    - Property: for all structure problems != "多表格混在同一 sheet", `buildSingleStructureExamples` output is identical to current behavior (highlights preserved on non-gap rows)
    - Property: for all SheetData where no column has mixed formats, `buildFormatConsistencyExamples` returns nil/empty
    - Property: for all non-format-consistency issue examples, `FormatLabels` field is nil
  - Verify tests PASS on UNFIXED code
  - **EXPECTED OUTCOME**: Tests PASS (confirms baseline behavior to preserve)
  - Test file: `backend/internal/assessment/pbt_issue_card_test.go`
  - Test command: `cd backend && go test ./internal/assessment/ -v -run TestPBT_Preservation_IssueCard`
  - _Requirements: 3.2, 3.4, 3.5_

- [x] 4. Write preservation property tests (frontend — BEFORE implementing fix)
  - **Property 2: Preservation** - Non-Bug Frontend Behavior Unchanged
  - **IMPORTANT**: Follow observation-first methodology
  - **Observe on UNFIXED code**:
    - Render issue with description NOT containing `\n` → observe `<div>` paragraph with pre-line whitespace
    - Render header row with non-highlighted cells → observe standard grey `<th>` background (#f3f4f6)
    - Render non-format-consistency issue examples → observe cells without format labels
  - **Write property-based tests capturing observed behavior**:
    - Property: for all descriptions without `\n`, rendering produces a single `<div>` paragraph (no `<ul>/<li>`)
    - Property: for all header cells NOT in highlights, styling uses standard grey background without amber/warning
    - Property: for all non-format-consistency examples, no format label elements rendered below cells
  - Verify tests PASS on UNFIXED code
  - **EXPECTED OUTCOME**: Tests PASS (confirms baseline behavior to preserve)
  - Test file: `frontend/src/pages/AssessmentPage.test.tsx`
  - Test command: `cd frontend && npx vitest --run src/pages/AssessmentPage.test.tsx`
  - _Requirements: 3.1, 3.3, 3.4, 3.6_

- [x] 5. Fix for Issue Card Rendering Bugs

  - [x] 5.1 Backend Bug 2 — Remove gap row highlights in "多表格混在同一 sheet"
    - In `backend/internal/assessment/issues.go`, locate the "多表格混在同一 sheet" case in `buildSingleStructureExamples`
    - Remove the `gapHighlights` slice construction that fills all column indices
    - Set `Highlights: nil` on the gap row IssueExample
    - _Bug_Condition: isBugCondition(X) where X.label = "（空白列）" AND X.highlights IS NOT EMPTY_
    - _Expected_Behavior: gap row Highlights = nil (no red highlighting on empty cells)_
    - _Preservation: Other structure problems (合併儲存格, 小計列) highlight behavior unchanged_
    - _Requirements: 1.2, 2.2, 3.2_

  - [x] 5.2 Backend Bug 4 — Add FormatLabels field to IssueExample
    - In `backend/internal/assessment/model.go`, add `FormatLabels []string \`json:"format_labels,omitempty"\`` to the `IssueExample` struct (after Merges field)
    - _Requirements: 2.4_

  - [x] 5.3 Backend Bug 4 — Add FormatTypeLabel helper function
    - In `backend/internal/assessment/format_detector.go`, add:
      ```go
      func FormatTypeLabel(ft FormatType) string {
          switch ft {
          case FormatDate:
              return "日期"
          case FormatNumeric:
              return "數字"
          case FormatBoolean:
              return "布林"
          default:
              return "文字"
          }
      }
      ```
    - _Requirements: 2.4_

  - [x] 5.4 Backend Bug 4 — Rewrite buildFormatConsistencyExamples for per-column groups
    - In `backend/internal/assessment/issues.go`, replace single-column logic with multi-column iteration:
      - Iterate over `mixedCols` (up to 5 columns)
      - For EACH column: independently find dominant format rows and mismatch format rows
      - Select 1-2 dominant rows + 2-3 mismatch rows per column
      - Set `Label` to the column header name (e.g., "Tracking No. 寄出快遞單號")
      - Build `FormatLabels` array: for each cell in the display columns, if cell's column is the current target column, set label to `FormatTypeLabel(DetectFormatType(cell))`; otherwise empty string
    - _Bug_Condition: countMixedFormatColumns(data) >= 1 AND indicator = "format_consistency"_
    - _Expected_Behavior: min(N, 5) distinct label groups, each with dominant+mismatch rows and FormatLabels_
    - _Preservation: Non-format-consistency issues produce examples without format_labels field_
    - _Requirements: 1.4, 2.4, 3.4, 3.5_

  - [x] 5.5 Frontend Bug 1 — Description first line as paragraph
    - In `frontend/src/pages/AssessmentPage.tsx`, modify the description rendering block (where `issue.description.includes('\n')` is checked):
      - Extract `firstLine = description.split('\n')[0]`
      - Extract `remainingLines = description.split('\n').slice(1).filter(Boolean)`
      - Render `firstLine` as a `<div>` paragraph with same styling as non-newline descriptions
      - Render `remainingLines` as `<ul><li>` items
    - _Bug_Condition: issue.description contains "\n"_
    - _Expected_Behavior: first line rendered as paragraph, remaining lines as list items_
    - _Preservation: Descriptions without "\n" continue to render as plain paragraph with pre-line_
    - _Requirements: 1.1, 2.1, 3.1_

  - [x] 5.6 Frontend Bug 3 — Empty header highlighted cells use amber background
    - In `frontend/src/pages/AssessmentPage.tsx`, modify the `<th>` rendering for highlighted header cells:
      - Change `background` from `rgba(220, 38, 38, 0.06)` to `rgba(245, 158, 11, 0.15)` (amber)
      - Change `border` from `1.5px solid var(--rose, #dc2626)` to `1.5px solid #f59e0b` (amber)
      - Change `color` from `var(--rose, #dc2626)` to `#92400e` (dark amber)
    - _Bug_Condition: example.row_number === 1 AND cellIndex in highlights_
    - _Expected_Behavior: amber/orange background clearly distinguishable from grey <th> background_
    - _Preservation: Non-highlighted header cells maintain standard grey <th> background_
    - _Requirements: 1.3, 2.3, 3.3_

  - [x] 5.7 Frontend Bug 4 — Add format_labels to IssueExample interface and render
    - In `frontend/src/pages/AssessmentPage.tsx`:
      - Add `format_labels?: string[]` to the `IssueExample` interface
      - In the data cell rendering loop (`ex.cells.map`), when `ex.format_labels?.[k]` is non-empty:
        - Render a small pill/tag below the cell value showing the format type (e.g., `<span style={{fontSize: 10, background: 'rgba(99,102,241,0.1)', borderRadius: 3, padding: '1px 4px'}}>日期</span>`)
    - _Bug_Condition: indicator === "format_consistency" AND example has format_labels_
    - _Expected_Behavior: format labels displayed below cells for affected column_
    - _Preservation: Non-format-consistency examples render without format labels_
    - _Requirements: 2.4, 3.4_

  - [x] 5.8 Verify bug condition exploration tests now pass (backend)
    - **Property 1: Expected Behavior** - Gap Row & Format Consistency Fixed
    - **IMPORTANT**: Re-run the SAME tests from task 1 — do NOT write new tests
    - The tests from task 1 encode the expected behavior
    - When these tests pass, it confirms the expected behavior is satisfied
    - Run: `cd backend && go test ./internal/assessment/ -v -run TestPBT_BugCondition_IssueCard`
    - **EXPECTED OUTCOME**: Tests PASS (confirms bugs are fixed)
    - _Requirements: 2.2, 2.4_

  - [x] 5.9 Verify bug condition exploration tests now pass (frontend)
    - **Property 1: Expected Behavior** - Description Rendering & Header Styling Fixed
    - **IMPORTANT**: Re-run the SAME tests from task 2 — do NOT write new tests
    - Run: `cd frontend && npx vitest --run src/pages/AssessmentPage.test.tsx`
    - **EXPECTED OUTCOME**: Tests PASS (confirms bugs are fixed)
    - _Requirements: 2.1, 2.3_

  - [x] 5.10 Verify preservation tests still pass (backend)
    - **Property 2: Preservation** - Backend Non-Bug Behavior Unchanged
    - **IMPORTANT**: Re-run the SAME tests from task 3 — do NOT write new tests
    - Run: `cd backend && go test ./internal/assessment/ -v -run TestPBT_Preservation_IssueCard`
    - **EXPECTED OUTCOME**: Tests PASS (confirms no regressions)
    - Confirm all backend preservation tests still pass after fix

  - [x] 5.11 Verify preservation tests still pass (frontend)
    - **Property 2: Preservation** - Frontend Non-Bug Behavior Unchanged
    - **IMPORTANT**: Re-run the SAME tests from task 4 — do NOT write new tests
    - Run: `cd frontend && npx vitest --run src/pages/AssessmentPage.test.tsx`
    - **EXPECTED OUTCOME**: Tests PASS (confirms no regressions)
    - Confirm all frontend preservation tests still pass after fix

- [x] 6. Checkpoint — Ensure all tests pass
  - Run full backend test suite: `cd backend && go test ./internal/assessment/ -v`
  - Run full frontend test suite: `cd frontend && npx vitest --run`
  - Ensure all tests pass, ask the user if questions arise
  - Verify no regressions in existing `pbt_test.go` and `indicators_test.go`

## Task Dependency Graph

```json
{
  "waves": [
    {"tasks": ["1", "2", "3", "4"]},
    {"tasks": ["5.1", "5.2", "5.3"]},
    {"tasks": ["5.4", "5.5", "5.6", "5.7"]},
    {"tasks": ["5.8", "5.9", "5.10", "5.11"]},
    {"tasks": ["6"]}
  ]
}
```

## Notes

- Backend tests use `pgregory.net/rapid` for property-based testing (already in go.mod)
- Frontend tests use `vitest` + `@testing-library/react` + `fast-check` (already installed)
- All tests run inside Docker; do NOT install packages globally
- Backend test command: `cd backend && go test ./internal/assessment/ -v -run TestName`
- Frontend test command: `cd frontend && npx vitest --run`
- Bug 2 and Bug 4 are backend-only; Bug 1 and Bug 3 are frontend-only; Bug 4 has both backend + frontend parts
