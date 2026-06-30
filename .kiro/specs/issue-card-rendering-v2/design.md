# Issue Card Rendering V2 Bugfix Design

## Overview

本次修復涵蓋品質評估頁面 Issue Card 的四項顯示缺陷：(1) 多行描述文字的第一行應為段落而非列表項；(2) 多表格結構的空白間隔列不應標紅；(3) 空白標題欄的高亮應使用醒目琥珀色背景；(4) 格式混用卡片應按欄分組顯示並附加格式標籤。修復策略為最小化改動，每個 bug 獨立修正且不影響其他 issue card 的既有行為。

## Glossary

- **Bug_Condition (C)**: 觸發 bug 的輸入條件 — 涵蓋四種情境（描述含換行、空白間隔列被標紅、空白標題欄高亮不明顯、格式混用僅取單欄）
- **Property (P)**: 修正後的期望行為 — 描述正確渲染、間隔列無標紅、標題欄琥珀色背景、格式混用分欄分組帶標籤
- **Preservation**: 不受修正影響的既有行為 — 無換行描述、其他結構問題、非高亮標題欄、非格式混用 issue 的顯示
- **`buildFormatConsistencyExamples`**: `issues.go` 中建構格式混用 issue 範例的函式，目前僅取第一個混用欄
- **`buildSingleStructureExamples`**: `issues.go` 中建構結構問題範例的函式，含"多表格混在同一 sheet" case
- **`IssueExample`**: `model.go` 中代表 issue 範例資料的 struct，包含 headers/cells/highlights/label
- **`FormatType`**: `format_detector.go` 中的格式分類 enum（Date / Numeric / Boolean / Text）
- **`FormatTypeLabel`**: 新增的 helper 函式，將 `FormatType` 轉為中文顯示標籤（"日期"/"數字"/"布林"/"文字"）

## Bug Details

### Bug Condition

本次修復涵蓋四種獨立的 bug condition：

**Bug 1 — 描述第一行被渲染為列表項**：當 issue description 包含 `\n` 字元時，前端把所有行（含第一行介紹語句）都渲染為 `<li>` 項目。

**Bug 2 — 多表格空白間隔列標紅**：當偵測到"多表格混在同一 sheet"問題時，backend 在兩表格間插入一列 label="（空白列）"的 example 且 highlights 包含所有欄位 index。

**Bug 3 — 空白標題欄紅框不明顯**：當 header row（row_number=1）有 highlighted 欄位時，前端用紅色邊框呈現，但灰色 `<th>` 背景上紅框不夠醒目。

**Bug 4 — 格式混用僅取單欄**：`buildFormatConsistencyExamples` 只取第一個 mixed column，所有 example 共享相同列號，其他欄位在這些列中可能格式一致，無法看出混用。

**Formal Specification:**

```
FUNCTION isBugCondition_Bug1(input)
  INPUT: input of type IssueDescription
  OUTPUT: boolean
  RETURN input.text CONTAINS "\n"
END FUNCTION

FUNCTION isBugCondition_Bug2(input)
  INPUT: input of type IssueExample
  OUTPUT: boolean
  RETURN input.label = "（空白列）"
         AND input.highlights IS NOT EMPTY
END FUNCTION

FUNCTION isBugCondition_Bug3(input)
  INPUT: input of type {example: IssueExample, cellIndex: int}
  OUTPUT: boolean
  RETURN input.example.row_number = 1
         AND input.cellIndex IN input.example.highlights
END FUNCTION

FUNCTION isBugCondition_Bug4(input)
  INPUT: input of type SheetData
  OUTPUT: boolean
  RETURN countMixedFormatColumns(input) >= 1
         AND input.indicator = "format_consistency"
END FUNCTION
```

### Examples

- **Bug 1**: Description = "以下欄位格式不一致：\nTracking No.\nAmount" → 目前三行全為 `<li>`；期望第一行為 `<p>`，後兩行為 `<li>`
- **Bug 2**: "多表格混在同一 sheet" 產生 label="（空白列）" example，highlights=[0,1,2,3,4,5] → 所有空 cell 被紅框標記；期望 highlights=nil
- **Bug 3**: "空白標題欄" header row 的 (空白) 欄位有 `1.5px solid red` border on grey `<th>` → 不夠醒目；期望 amber/orange 背景色
- **Bug 4**: 兩個 mixed columns (col A: 80% numeric, col B: 70% date) 但只取 col A 的 rows → col B 在這些 rows 中全是 date 看不出 mix；期望分別選列

## Expected Behavior

### Preservation Requirements

**Unchanged Behaviors:**
- 無 `\n` 的 issue description 繼續以 `pre-line` 模式顯示為段落
- "合併儲存格"、"小計列"等其他結構問題的 example highlight 邏輯不變
- 非 highlighted 的 header cells 維持標準灰色 `<th>` 背景
- 非 format_consistency 的 issue examples 不含 `format_labels` 欄位
- 未展開的 issue card header（title + severity + description + affected count）佈局不變
- 滑鼠點擊展開/收合 card 行為不變
- `buildDuplicateExamples`、`buildRowCompletenessExamples` 等其他 example builder 不受影響

**Scope:**
所有不屬於上述四種 bug condition 的輸入應完全不受影響。包括：
- Description 不含 `\n` 的 issue 卡片
- 非"多表格"的結構問題 examples
- 非 row_number=1 或非 highlighted 的 header cells
- 非 format_consistency indicator 的 issue examples

## Hypothesized Root Cause

Based on the bug descriptions and code analysis:

1. **Bug 1 — Missing first-line differentiation**: `AssessmentPage.tsx` line ~340 使用 `issue.description.split('\n').map(line => <li>)` 將所有行一視同仁渲染為列表項。缺少邏輯將第一行（介紹語句）與後續行（欄位名稱列表）區分。

2. **Bug 2 — Gap row highlights all cells**: `buildSingleStructureExamples` "多表格混在同一 sheet" case 在 issues.go ~1379 明確建構 `gapHighlights := make([]int, len(allCols))` 並填入所有 index，再設為 example 的 Highlights。這是故意標注間隔但視覺效果不佳。

3. **Bug 3 — Red border on grey background low contrast**: `AssessmentPage.tsx` 對 header row highlighted cells 使用 `border: 1.5px solid var(--rose)` + `background: rgba(220,38,38,0.06)` — 在灰色 `#f3f4f6` 上紅框可見度低，因為整個 `<th>` 本身就有 border。

4. **Bug 4 — Single target column selection**: `buildFormatConsistencyExamples` 在 issues.go ~676 執行 `targetCol := mixedCols[0]` 只取第一個 mixed column，後續所有 dominant/mismatch row 選擇都基於這單一欄位。其他 mixed columns 的混用狀態在被選中的行內可能看不出來。

## Correctness Properties

Property 1: Bug Condition — Description First Line as Paragraph

_For any_ issue where the description contains `\n` characters (isBugCondition_Bug1 returns true), the fixed rendering function SHALL render the first line (before the first `\n`) as a paragraph element (`<p>` or `<div>`), and only subsequent lines as `<li>` list items.

**Validates: Requirements 2.1**

Property 2: Bug Condition — Gap Row Not Highlighted

_For any_ "多表格混在同一 sheet" structure example where the gap row has label="（空白列）" (isBugCondition_Bug2 returns true), the fixed `buildSingleStructureExamples` function SHALL produce the gap row with `Highlights: nil` (no red highlighting on empty cells).

**Validates: Requirements 2.2**

Property 3: Bug Condition — Empty Header Amber Background

_For any_ header cell where row_number=1 and the cell index is in the highlights array (isBugCondition_Bug3 returns true), the fixed frontend rendering SHALL use an amber/orange background color (`#fbbf24` or similar) instead of red border styling, making the problem location clearly distinguishable from the grey `<th>` background.

**Validates: Requirements 2.3**

Property 4: Bug Condition — Per-Column Format Groups with Labels

_For any_ sheet where format_consistency issue is detected with N mixed columns (isBugCondition_Bug4 returns true, N ≥ 1), the fixed `buildFormatConsistencyExamples` function SHALL produce min(N, 5) distinct label groups, each independently selecting rows showing contrast for that specific column, and each example SHALL include a `FormatLabels` field with the detected format type string (e.g., "數字", "文字", "日期") for the affected column's cells.

**Validates: Requirements 2.4**

Property 5: Preservation — Non-Newline Description Rendering

_For any_ issue where the description does NOT contain `\n` (isBugCondition_Bug1 returns false), the fixed rendering function SHALL produce exactly the same output as the original function, preserving `pre-line` paragraph display.

**Validates: Requirements 3.1**

Property 6: Preservation — Other Structure Problems Unchanged

_For any_ structure problem that is NOT "多表格混在同一 sheet" (isBugCondition_Bug2 input is not applicable), the fixed `buildSingleStructureExamples` function SHALL produce the same result as the original function, preserving all existing highlight behavior.

**Validates: Requirements 3.2**

Property 7: Preservation — Non-Highlighted Header Cells Unchanged

_For any_ header cell where the cell index is NOT in the highlights array (isBugCondition_Bug3 returns false), the fixed rendering SHALL produce the same standard grey `<th>` styling as before.

**Validates: Requirements 3.3**

Property 8: Preservation — Non-Format-Consistency Issues Unchanged

_For any_ issue that is NOT format_consistency (isBugCondition_Bug4 does not apply), the fixed code SHALL produce examples without `format_labels` field and without per-column label grouping, identical to the original behavior.

**Validates: Requirements 3.4, 3.5**

## Fix Implementation

### Changes Required

Assuming our root cause analysis is correct:

**File**: `frontend/src/pages/AssessmentPage.tsx`

**Bug 1 Fix — Description rendering:**

1. **Split first line from rest**: In the description rendering block (~line 340), when `issue.description.includes('\n')` is true:
   - Extract `firstLine = description.split('\n')[0]`
   - Extract `remainingLines = description.split('\n').slice(1).filter(Boolean)`
   - Render `firstLine` as a `<div>` paragraph
   - Render `remainingLines` as `<ul><li>` items

**Bug 3 Fix — Header cell styling:**

2. **Amber background for highlighted headers**: In the `<th>` rendering block for header row cells, when `isHighlighted` is true:
   - Change `background` from `rgba(220, 38, 38, 0.06)` to `rgba(245, 158, 11, 0.15)` (amber)
   - Change `border` from `1.5px solid var(--rose)` to `1.5px solid #f59e0b` (amber border)
   - Change `color` from `var(--rose)` to `#92400e` (dark amber text)
   - Keep `fontWeight: 600`

---

**File**: `backend/internal/assessment/model.go`

**Bug 4 Fix — Add FormatLabels field:**

3. **Extend IssueExample struct**: Add a new field:
   ```go
   FormatLabels []string `json:"format_labels,omitempty"`
   ```

---

**File**: `backend/internal/assessment/format_detector.go`

**Bug 4 Fix — Add FormatTypeLabel helper:**

4. **Add label conversion function**:
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

---

**File**: `backend/internal/assessment/issues.go`

**Bug 2 Fix — Remove gap row highlights:**

5. **Set gap row Highlights to nil**: In `buildSingleStructureExamples`, case "多表格混在同一 sheet", change the gap row construction at ~line 1379:
   - Remove `gapHighlights` slice construction
   - Set `Highlights: nil` on the gap row IssueExample

**Bug 4 Fix — Per-column format example groups:**

6. **Rewrite `buildFormatConsistencyExamples`**: Replace single-column logic with multi-column iteration:
   - Iterate over `mixedCols` (up to 5)
   - For EACH column: independently find dominant rows and mismatch rows
   - Select 1-2 dominant rows + 2-3 mismatch rows per column
   - Set `Label` to the column header name (e.g., "Tracking No. 寄出快遞單號")
   - Build `FormatLabels` array: for each cell in `displayCols`, if the cell's column is the current target column, set its label to `FormatTypeLabel(DetectFormatType(cell))`; otherwise leave empty string

---

**File**: `frontend/src/pages/AssessmentPage.tsx`

**Bug 4 Fix — Render format labels:**

7. **Add `format_labels` to IssueExample interface**: Add optional field:
   ```typescript
   format_labels?: string[]
   ```

8. **Render format label below cell**: In the data cell rendering loop, if `ex.format_labels` exists and `ex.format_labels[k]` is non-empty:
   - Render a small tag below the cell value showing the format type (e.g., `<span>` with small font, pill styling)

## Testing Strategy

### Validation Approach

The testing strategy follows a two-phase approach: first, surface counterexamples that demonstrate the bugs on unfixed code, then verify the fixes work correctly and preserve existing behavior.

### Exploratory Bug Condition Checking

**Goal**: Surface counterexamples that demonstrate the bugs BEFORE implementing the fix. Confirm or refute the root cause analysis. If we refute, we will need to re-hypothesize.

**Test Plan**: Write unit tests and property-based tests that exercise each bug condition. Run on UNFIXED code to observe failures.

**Test Cases**:
1. **Bug 1 Test**: Render a description containing "\n" and assert first line is NOT a `<li>` element (will fail on unfixed code — all lines are `<li>`)
2. **Bug 2 Test**: Call `buildSingleStructureExamples(data, "多表格混在同一 sheet")` with multi-block data and assert gap row has nil highlights (will fail — highlights contain all indices)
3. **Bug 3 Test**: Render header row with highlighted cells and assert background is amber (will fail — background is red-tinted)
4. **Bug 4 Test**: Call `buildFormatConsistencyExamples` with 3 mixed columns and assert 3 distinct label groups exist (will fail — only 1 target column used)

**Expected Counterexamples**:
- Bug 1: All lines rendered as `<li>` including introductory text
- Bug 2: Gap row `Highlights` = `[0,1,2,3,...]` (all column indices)
- Bug 3: Header cell `background` = `rgba(220,38,38,0.06)` (barely visible red on grey)
- Bug 4: Single group of examples, only one column's contrast shown

### Fix Checking

**Goal**: Verify that for all inputs where each bug condition holds, the fixed function produces the expected behavior.

**Pseudocode:**
```
// Bug 1
FOR ALL description WHERE description CONTAINS "\n" DO
  rendered := renderDescription_fixed(description)
  lines := description SPLIT "\n"
  ASSERT rendered.firstElement IS paragraph CONTAINING lines[0]
  ASSERT rendered.listItems EQUALS lines[1:]
END FOR

// Bug 2
FOR ALL data WHERE hasMultipleDataBlocks(data) DO
  examples := buildSingleStructureExamples_fixed(data, "多表格混在同一 sheet")
  gapRows := examples WHERE label = "（空白列）"
  ASSERT FOR_EACH gap IN gapRows: gap.Highlights = nil
END FOR

// Bug 3
FOR ALL (example, cellIdx) WHERE example.row_number=1 AND cellIdx IN example.highlights DO
  style := getHeaderStyle_fixed(example, cellIdx)
  ASSERT style.background CONTAINS amber_component
  ASSERT style.border CONTAINS amber_color
END FOR

// Bug 4
FOR ALL data WHERE countMixedFormatColumns(data) >= 1 DO
  examples := buildFormatConsistencyExamples_fixed(data)
  groups := groupByLabel(examples)
  ASSERT len(groups) = min(countMixedFormatColumns(data), 5)
  FOR EACH group IN groups DO
    ASSERT hasRows(group, withHighlights=false) >= 1   // dominant
    ASSERT hasRows(group, withHighlights=true) >= 1    // mismatch
    ASSERT FOR_EACH row IN group: row.FormatLabels IS NOT nil
  END FOR
END FOR
```

### Preservation Checking

**Goal**: Verify that for all inputs where the bug conditions do NOT hold, the fixed functions produce the same result as the original functions.

**Pseudocode:**
```
// Bug 1 preservation
FOR ALL description WHERE description NOT CONTAINS "\n" DO
  ASSERT renderDescription_original(description) = renderDescription_fixed(description)
END FOR

// Bug 2 preservation
FOR ALL (data, problem) WHERE problem != "多表格混在同一 sheet" DO
  ASSERT buildSingleStructureExamples_original(data, problem) = buildSingleStructureExamples_fixed(data, problem)
END FOR

// Bug 3 preservation
FOR ALL (example, cellIdx) WHERE NOT (example.row_number=1 AND cellIdx IN example.highlights) DO
  ASSERT getHeaderStyle_original(example, cellIdx) = getHeaderStyle_fixed(example, cellIdx)
END FOR

// Bug 4 preservation
FOR ALL data WHERE indicator != "format_consistency" DO
  ASSERT buildExamples_original(data) = buildExamples_fixed(data)
  ASSERT examples DO NOT contain format_labels field
END FOR
```

**Testing Approach**: Property-based testing is recommended for preservation checking because:
- It generates many SheetData configurations automatically across the input domain
- It catches edge cases (e.g., single-column sheets, all-empty data, 0 mixed columns) that manual tests miss
- It provides strong guarantees that non-buggy paths remain unchanged

**Test Plan**: Observe behavior on UNFIXED code first for non-bug inputs, then write property-based tests capturing that behavior.

**Test Cases**:
1. **Description Preservation**: Generate random descriptions without `\n` and verify rendering is unchanged
2. **Structure Problem Preservation**: Generate structure examples for "合併儲存格", "小計列" etc. and verify highlights unchanged
3. **Header Non-Highlight Preservation**: Generate header rows with non-highlighted cells and verify standard grey styling
4. **Non-Format Issue Preservation**: Generate examples for duplicate/completeness issues and verify no format_labels present

### Unit Tests

- Test `buildFormatConsistencyExamples` with 1, 2, 3, 5, and 6 mixed columns → verify correct number of label groups (capped at 5)
- Test `buildSingleStructureExamples` "多表格混在同一 sheet" → verify gap row has nil highlights
- Test `FormatTypeLabel` returns correct Chinese labels for all FormatType values
- Test description rendering with "\n" → first line is paragraph, rest are list items
- Test description rendering without "\n" → unchanged paragraph with pre-line

### Property-Based Tests

- Generate random SheetData with varying column mixes → verify `buildFormatConsistencyExamples` always produces ≤ 5 groups, each group has both dominant and mismatch rows, and all format_labels are valid strings
- Generate random multi-block SheetData → verify `buildSingleStructureExamples` gap row never has non-nil highlights
- Generate random format data where consistency ≥ 80% for all columns → verify function returns nil (no mixed columns detected, preservation)
- Generate random SheetData with no multi-table structure → verify `buildSingleStructureExamples` output is unchanged vs original

### Integration Tests

- Full assessment flow with a file containing mixed-format columns → verify API response has per-column label groups with format_labels
- Full assessment flow with multi-table file → verify API response gap row has no highlights
- Frontend render test: mount AssessmentPage with mock data containing multiline description → verify DOM structure (paragraph + list)
- Frontend render test: mount AssessmentPage with header row highlights → verify amber background CSS applied
