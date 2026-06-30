# Bugfix Requirements Document

## Introduction

Issue card 在品質評估頁面的展示存在四項視覺/資訊不足的問題，導致使用者無法正確理解系統偵測到的資料品質問題。本次修復涵蓋：描述文字的換行顯示（格式混用卡片）、多表格結構問題的紅框標示位置、空白標題欄紅框可見度不足、以及格式混用卡片中缺少偵測到的格式類型標注。

## Bug Analysis

### Current Behavior (Defect)

1.1 WHEN issue description contains `\n` characters (e.g. 格式混用 description "以下欄位格式不一致（同欄位中混合了不同格式）：\nTracking No. 寄出快遞單號\nAmount 金額") THEN the system renders ALL lines as `<li>` bullet items including the introductory sentence, making the intro text look like a list item instead of a paragraph

1.2 WHEN the "多表格混在同一 sheet" structure problem is detected THEN the backend highlights ALL cells of the empty gap row between two tables (red border), causing the user to see red-framed empty cells instead of seeing which rows constitute the two separate tables

1.3 WHEN "空白標題欄" is detected and the header row (row_number === 1) has empty columns highlighted THEN the red border on the grey `<th>` header cells does not visually stand out enough, making users perceive the red frame as being on data rows rather than the header row

1.4 WHEN "格式混用" issue examples are generated THEN the backend puts ALL mixed-format columns into a SINGLE table with shared rows, causing some columns to only show one format type within the selected rows (because the rows were chosen to demonstrate column A's mixing but column B happens to have only one format in those same rows), making it impossible for the user to see the format mixing for every affected column

### Expected Behavior (Correct)

2.1 WHEN issue description contains `\n` characters THEN the system SHALL render the first line (before the first `\n`) as a normal paragraph, and subsequent lines as `<li>` list items — OR alternatively render all content as a single line with "、" separating column names

2.2 WHEN the "多表格混在同一 sheet" structure problem is detected THEN the system SHALL NOT highlight the empty gap row in red; instead, it SHALL visually indicate the boundaries of each data block (e.g., highlight the first row of each block, or use distinct label groups with colored backgrounds to distinguish table 一 from table 二)

2.3 WHEN "空白標題欄" is detected and header columns are empty THEN the system SHALL use a highly visible styling on those header cells — such as an amber/orange background color or a warning icon (⚠️) next to the "(空白)" text — to clearly distinguish the problem location from normal data rows

2.4 WHEN "格式混用" issue is detected with multiple mixed-format columns THEN the backend SHALL generate SEPARATE example groups (using distinct Labels like "Tracking No. 寄出快遞單號", "Amount 金額") for EACH affected column (up to a maximum of 5 columns). Each group SHALL independently select rows that clearly demonstrate the format mixing contrast for THAT specific column — showing 1-2 rows of the dominant format (no highlight) and 2-3 rows of the mismatched format (highlighted). Additionally, each cell in the affected column SHALL display its detected format type label (e.g., "數字", "文字", "日期") so users can understand the contrast.

### Unchanged Behavior (Regression Prevention)

3.1 WHEN issue description does NOT contain `\n` characters THEN the system SHALL CONTINUE TO render description as a plain paragraph with `pre-line` whitespace handling

3.2 WHEN table structure issues other than "多表格混在同一 sheet" are displayed (e.g., "合併儲存格", "小計列") THEN the system SHALL CONTINUE TO show their existing example formatting and highlight behavior unchanged

3.3 WHEN header row (row_number === 1) columns are NOT in the highlights array THEN the system SHALL CONTINUE TO render them with the standard grey `<th>` background without any warning styling

3.4 WHEN issue examples for non-format-consistency issues are displayed (e.g., 資料缺漏, 重複資料) THEN the system SHALL CONTINUE TO display cells without format type labels

3.5 WHEN the format consistency issue has rows that match the dominant format (no highlight) within each per-column group THEN the system SHALL CONTINUE TO display those rows without red highlighting, and the format label SHALL serve as informational context only (not as a warning)

3.6 WHEN issue cards are collapsed (not expanded) THEN the system SHALL CONTINUE TO show the same header layout with title, severity badge, description, and affected count

---

## Bug Condition Derivations

### Bug 1: Description Line Break Rendering

```pascal
FUNCTION isBugCondition(X)
  INPUT: X of type IssueDescription
  OUTPUT: boolean

  RETURN X.text CONTAINS "\n"
END FUNCTION
```

```pascal
// Property: Fix Checking — First line rendered as paragraph
FOR ALL X WHERE isBugCondition(X) DO
  rendered ← renderDescription'(X)
  firstLine ← X.text SPLIT("\n")[0]
  remainingLines ← X.text SPLIT("\n")[1:]
  ASSERT firstLine IS_RENDERED_AS paragraph
  ASSERT remainingLines ARE_RENDERED_AS list_items
END FOR
```

```pascal
// Property: Preservation Checking
FOR ALL X WHERE NOT isBugCondition(X) DO
  ASSERT renderDescription(X) = renderDescription'(X)
END FOR
```

### Bug 2: Multi-Table Gap Row Highlighting

```pascal
FUNCTION isBugCondition(X)
  INPUT: X of type StructureExample
  OUTPUT: boolean

  RETURN X.label = "（空白列）" AND X.highlights IS_NOT_EMPTY
END FUNCTION
```

```pascal
// Property: Fix Checking — Gap row not highlighted
FOR ALL X WHERE isBugCondition(X) DO
  examples ← buildSingleStructureExamples'(data, "多表格混在同一 sheet")
  gapRows ← examples WHERE label = "（空白列）"
  ASSERT FOR_EACH gapRow IN gapRows: gapRow.highlights = nil OR gapRow NOT IN examples
END FOR
```

```pascal
// Property: Preservation Checking
FOR ALL X WHERE NOT isBugCondition(X) DO
  ASSERT buildSingleStructureExamples(data, problem) = buildSingleStructureExamples'(data, problem)
END FOR
```

### Bug 3: Empty Header Visibility

```pascal
FUNCTION isBugCondition(X)
  INPUT: X of type HeaderCell
  OUTPUT: boolean

  RETURN X.row_number = 1 AND X.index IN X.parentExample.highlights
END FUNCTION
```

```pascal
// Property: Fix Checking — Highlighted headers use distinct visible styling
FOR ALL X WHERE isBugCondition(X) DO
  style ← getHeaderStyle'(X)
  ASSERT style.background = amber_or_warning_color OR style.icon = "⚠️"
  ASSERT style IS visually_distinguishable FROM normal_data_cell_highlight
END FOR
```

```pascal
// Property: Preservation Checking
FOR ALL X WHERE NOT isBugCondition(X) DO
  ASSERT getHeaderStyle(X) = getHeaderStyle'(X)
END FOR
```

### Bug 4: Format Mixing Per-Column Separation

```pascal
FUNCTION isBugCondition(X)
  INPUT: X of type IssueExampleSet
  OUTPUT: boolean

  RETURN X.indicator = "format_consistency" AND countMixedColumns(X.data) >= 1
END FUNCTION
```

```pascal
// Property: Fix Checking — Each mixed column gets its own labeled group with format labels
FOR ALL X WHERE isBugCondition(X) DO
  result ← buildFormatConsistencyExamples'(data)
  mixedCols ← findMixedFormatColumns(data)
  displayedCols ← min(len(mixedCols), 5)
  
  // Each column gets its own Label group
  ASSERT countDistinctLabels(result) = displayedCols
  
  // Within each group, both dominant and mismatch formats are visible
  FOR EACH group IN result.groups DO
    dominantRows ← group.rows WHERE highlights = nil
    mismatchRows ← group.rows WHERE highlights IS_NOT_EMPTY
    ASSERT len(dominantRows) >= 1
    ASSERT len(mismatchRows) >= 1
    // Format labels present
    ASSERT FOR_EACH row IN group.rows:
      row HAS format_labels field
      AND format_labels[targetCol] IN {"數字", "文字", "日期", "布林"}
  END FOR
END FOR
```

```pascal
// Property: Preservation Checking
FOR ALL X WHERE NOT isBugCondition(X) DO
  ASSERT buildExamples(X) = buildExamples'(X)
  // Non-format-consistency issues do not include format_labels or per-column grouping
END FOR
```
