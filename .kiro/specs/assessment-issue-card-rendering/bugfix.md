# Bugfix Requirements Document

## Introduction

AssessmentPage.tsx 中的問題卡片（Issue Card）渲染存在 4 個前端 Bug，影響用戶查看品質評估問題的體驗。問題涵蓋：描述文字排版斷裂、格式混用問題只顯示單一表格、多表結構中第二張表的行未正確高亮與缺少表頭樣式、以及表頭列（RowNumber==1）的高亮錯誤地標記在資料列而非表頭列。

## Bug Analysis

### Current Behavior (Defect)

1.1 WHEN issue description text contains `\n` newline characters THEN the system renders the text with `white-space: pre-line` in a container with insufficient width, causing lines to wrap poorly and producing a visually broken layout

1.2 WHEN format consistency issues have examples with multiple distinct Labels (e.g. "表格一", "表格二") THEN the system renders all groups within a single visual table container without clear visual separation, making it appear as one combined table instead of separate independent tables

1.3 WHEN examples span multiple Label groups (representing different tables/sheets) THEN the system uses the first group's headers for all subsequent groups, and the first row of a new Label group is not rendered with header styling (th) nor are its rows highlighted with red borders

1.4 WHEN an example has `row_number == 1` (indicating a header row) and has highlight indices THEN the system applies the red highlight border/styling to tbody td cells instead of the thead th cells, causing the wrong row to be visually marked

### Expected Behavior (Correct)

2.1 WHEN issue description text contains `\n` newline characters THEN the system SHALL render each line as a separate list item (or use adequate container width) so that multiline descriptions display clearly without awkward wrapping

2.2 WHEN format consistency issues have examples with multiple distinct Labels THEN the system SHALL render each Label group as a separate independent table element with its own border and heading, showing a maximum of 3 table groups

2.3 WHEN examples span multiple Label groups THEN the system SHALL use each group's own first row as the header row (with th styling) when it has `row_number == 1`, and SHALL apply red highlight borders to rows within each group independently

2.4 WHEN an example has `row_number == 1` and has highlight indices THEN the system SHALL apply the red highlight border/styling to the corresponding th elements in the thead row, not to td cells in the tbody

### Unchanged Behavior (Regression Prevention)

3.1 WHEN issue descriptions do NOT contain `\n` newline characters THEN the system SHALL CONTINUE TO render the description as a single paragraph with normal text flow

3.2 WHEN format consistency issues have only one Label group (or no Label) THEN the system SHALL CONTINUE TO render a single table with the existing header/data row layout

3.3 WHEN all examples have `row_number > 1` (data rows only) THEN the system SHALL CONTINUE TO apply red highlight borders to td cells in the tbody as currently implemented

3.4 WHEN examples have no highlights (empty highlights array) THEN the system SHALL CONTINUE TO render cells with default styling without any red borders

3.5 WHEN examples contain merge information THEN the system SHALL CONTINUE TO render merged cells with colspan and blue merge styling as currently implemented

3.6 WHEN issue cards are collapsed (not expanded) THEN the system SHALL CONTINUE TO display the header section with title, severity badge, description, and affected count without any change
