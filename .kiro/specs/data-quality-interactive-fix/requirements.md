# Requirements Document

## Introduction

本功能為 SAFE-AI Excel 梳理小工具新增進階資料品質偵測與互動式修正能力。現有的 Assessment（Step 3）與 Cleaning（Step 5）流程僅處理結構性問題（合併儲存格、小計列、重複列等），但實際業務資料中存在更深層的語意問題：儲存格引用佔位符（「同OOO」）、欄位型別不一致、行內備註混入結構欄位、孤立合計列、以及空白標題欄。這些問題嚴重影響 AI 對資料的計算與理解能力。

本功能擴充偵測引擎（新增 5 種 issue type）並提供對應的互動式修正 UI（Cell Editor），讓使用者能逐筆檢視問題並選擇修正方式，而非僅依賴全自動批次規則。

## Glossary

- **Assessment_Engine**：負責六項 AI Data Readiness 指標計算與問題偵測的後端模組
- **Cleaning_Engine**：負責資料梳理批次規則與互動式修正執行的後端模組
- **Frontend**：SAFE-AI Excel 梳理小工具前端應用程式（React + TypeScript）
- **Cell_Reference_Placeholder**：儲存格中以「同」開頭後接 Excel 欄位座標的值（如「同AH2」、「同AI6」），表示「與該儲存格相同值」的人工標記
- **Column_Type**：系統根據欄位中多數內容推斷出的資料型別（numeric、date、text、boolean）
- **Inline_Remark**：結構化欄位中以括號包裹的附註文字，例如 PI 單號後的中文說明
- **Orphan_Total_Row**：位於資料末端、前方有多列空行、僅含單一數值的疑似合計列
- **Cell_Editor**：互動式儲存格編輯元件，讓使用者逐筆檢視問題儲存格並選擇修正方式
- **User**：已登入的系統使用者
- **SheetData**：後端用於表示解析後試算表資料的結構（含 Headers、Rows、ColCount 等）
- **Issue**：Assessment_Engine 偵測到的資料品質問題，包含標題、嚴重度、影響列數與範例
- **RowOp**：Cleaning_Engine 支援的單列操作指令結構

## Requirements

### Requirement 1: 儲存格引用佔位符偵測（Cell Reference Placeholder Detection）

**User Story:** As a User, I want the system to detect cells containing "同" followed by a cell reference (like 同AH2), so that I can identify values that AI cannot compute on.

#### Acceptance Criteria

1. WHEN an assessment is triggered, THE Assessment_Engine SHALL scan all cells and flag those matching the regex pattern `^同[A-Z]+\d+$` (value starts with "同" followed by one or more uppercase letters and one or more digits, with no other content) as a "cell_reference_placeholder" issue.
2. WHEN Cell_Reference_Placeholder issues are detected, THE Assessment_Engine SHALL report the issue with severity "High", include the affected row count, and provide up to 5 examples showing the flagged cells with their column headers and row numbers.
3. WHEN Cell_Reference_Placeholder issues are detected in a column that the Assessment_Engine infers as numeric (per Requirement 2 column type inference), THE Assessment_Engine SHALL include a description indicating that AI computation on this column is impossible due to non-numeric references.
4. IF no cells match the Cell_Reference_Placeholder pattern, THEN THE Assessment_Engine SHALL not include this issue type in the assessment results.

### Requirement 2: 欄位型別不一致偵測（Column Type Inconsistency Detection）

**User Story:** As a User, I want the system to detect columns where the majority of values are numeric but some cells contain text, so that I can fix mixed-type columns that break AI arithmetic.

#### Acceptance Criteria

1. WHEN an assessment is triggered, THE Assessment_Engine SHALL infer the Column_Type for each column by classifying each non-empty cell as numeric (parseable as integer, decimal, or thousands-separated number after trimming whitespace and removing currency symbols NT$, USD, $, ¥, €) or non-numeric, and assigning "numeric" type when more than 70% of non-empty cells are numeric.
2. WHEN a column is inferred as numeric type, THE Assessment_Engine SHALL flag each non-numeric cell in that column as a "column_type_mismatch" issue.
3. WHEN column_type_mismatch issues are detected, THE Assessment_Engine SHALL report the issue with severity "High" if more than 10% of cells in the column are mismatched or "Medium" if 10% or fewer are mismatched, include the total count of affected cells across all numeric columns, and provide up to 5 examples showing the mismatched cells highlighted within their row context.
4. THE Assessment_Engine SHALL not flag cells that are empty as type mismatches.
5. IF no columns qualify as numeric type or no type mismatches exist, THEN THE Assessment_Engine SHALL not include this issue type in the assessment results.

### Requirement 3: 行內備註偵測（Inline Remark Detection）

**User Story:** As a User, I want the system to detect cells in structured columns that contain parenthesized remarks, so that I can separate remarks from actual data values to improve record matching.

#### Acceptance Criteria

1. WHEN an assessment is triggered, THE Assessment_Engine SHALL identify structured columns by checking if more than 60% of non-empty values in a column match a consistent structural pattern (containing alphanumeric identifiers like order numbers, PI numbers, or codes without parenthesized content).
2. WHEN a structured column is identified, THE Assessment_Engine SHALL flag cells that contain parenthesized content (using half-width `()` or full-width `（）`) where the parenthesized text contains Chinese characters or is longer than 5 characters, as an "inline_remark" issue.
3. WHEN inline_remark issues are detected, THE Assessment_Engine SHALL report the issue with severity "Medium", include the count of affected cells, and provide up to 5 examples showing the cell value with the remark portion highlighted.
4. THE Assessment_Engine SHALL not flag cells where the parenthesized content is part of the structural pattern itself (for example, version numbers like "(v2)" or single-digit codes like "(A)").
5. IF no structured columns contain inline remarks, THEN THE Assessment_Engine SHALL not include this issue type in the assessment results.

### Requirement 4: 孤立合計列偵測（Orphan Total Row Detection）

**User Story:** As a User, I want the system to detect isolated numeric values at the bottom of data that appear to be subtotals, so that I can remove them before AI treats them as separate transactions.

#### Acceptance Criteria

1. WHEN an assessment is triggered, THE Assessment_Engine SHALL detect Orphan_Total_Rows by identifying rows where: (a) the row appears after 2 or more consecutive empty rows following the main data block, (b) the row contains at most 2 non-empty cells, and (c) at least one non-empty cell contains a numeric value (parseable as a number after removing thousands separators and decimal points).
2. WHEN Orphan_Total_Row issues are detected, THE Assessment_Engine SHALL report them as part of the existing "表格結構問題" issue category with sub-label "孤立合計列", include the row numbers of detected orphan totals, and provide examples showing the row content.
3. THE Assessment_Engine SHALL apply this detection independently of the existing subtotal keyword detection (which uses "小計", "合計", "total", "subtotal"), as orphan totals typically lack keyword markers.
4. IF no Orphan_Total_Rows are found, THEN THE Assessment_Engine SHALL not include this sub-issue in the structure results.

### Requirement 5: 空白標題欄偵測（Empty Header Cell Detection）

**User Story:** As a User, I want the system to detect columns with empty or blank header names, so that I can provide meaningful column names for AI comprehension.

#### Acceptance Criteria

1. WHEN an assessment is triggered, THE Assessment_Engine SHALL check each column header in the header row and flag columns where the header is empty (null, empty string, or contains only whitespace characters) as an "empty_header" issue.
2. WHEN empty_header issues are detected, THE Assessment_Engine SHALL report the issue with severity "Low", include the count of affected columns, and list the column positions (e.g., "第1欄", "第3欄") that have empty headers.
3. THE Assessment_Engine SHALL include this issue independently of the existing AI Query Readiness "column name quality" sub-condition, providing specific actionable detail about which columns need naming.
4. IF all column headers are non-empty after trimming whitespace, THEN THE Assessment_Engine SHALL not include this issue type in the assessment results.

### Requirement 6: 互動式儲存格編輯元件（Interactive Cell Editor UI）

**User Story:** As a User, I want an interactive editing interface during the cleaning step that shows flagged cells and lets me choose how to fix each one, so that I can make informed corrections rather than relying solely on automatic rules.

#### Acceptance Criteria

1. WHEN the cleaning step displays issues that require interactive fixes (cell_reference_placeholder, column_type_mismatch, inline_remark, or empty_header), THE Frontend SHALL render a Cell_Editor component that lists all flagged cells grouped by issue type.
2. THE Cell_Editor SHALL display each flagged cell with: the column name, the row number (1-based Excel row), the current cell value, and the issue type description.
3. THE Cell_Editor SHALL provide the following action options for each flagged cell: "輸入新值" (type a replacement value), "保留原值" (keep the original value unchanged), and "刪除該列" (delete the entire row containing this cell).
4. WHEN the issue type is "inline_remark", THE Cell_Editor SHALL additionally provide a "分離備註" action that splits the parenthesized remark into a separate "備註" column while keeping the structural value in the original cell.
5. WHEN the issue type is "empty_header", THE Cell_Editor SHALL display an inline text input for the user to type a new column header name.
6. WHEN the User confirms all Cell_Editor changes, THE Frontend SHALL submit the changes as a batch to the Cleaning_Engine API endpoint.

### Requirement 7: 互動式修正後端處理（Interactive Fix Backend Processing）

**User Story:** As a User, I want the backend to apply my interactive cell edits correctly and record them in the cleaning log, so that my manual corrections are traceable.

#### Acceptance Criteria

1. WHEN the Cleaning_Engine receives a batch of interactive cell edits, THE Cleaning_Engine SHALL apply each edit in the following order: cell value replacements first, remark splits second, header renames third, and row deletions last (processing deletion indices in descending order to avoid index shifting).
2. WHEN a "輸入新值" edit is applied, THE Cleaning_Engine SHALL update the specified cell (identified by row index and column index) with the new value and record the operation in the Cleaning_Log with operation_type "cell_edit", the affected row index, and the old and new values in the details field.
3. WHEN a "分離備註" edit is applied, THE Cleaning_Engine SHALL extract the parenthesized content from the cell, place the extracted remark in a new "備註" column (appended as the last column if it does not yet exist), update the original cell to contain only the structural value (without the parenthesized content), and record the operation in the Cleaning_Log with operation_type "remark_split".
4. WHEN an "empty_header" rename is applied, THE Cleaning_Engine SHALL update the column header at the specified column index with the user-provided name and record the operation in the Cleaning_Log with operation_type "header_rename".
5. WHEN a "刪除該列" edit is applied, THE Cleaning_Engine SHALL remove the specified row from the dataset and record the operation in the Cleaning_Log with operation_type "delete_row".
6. IF a cell edit references an invalid row index or column index (out of bounds), THEN THE Cleaning_Engine SHALL skip that edit, continue processing remaining edits, and include the skipped edit details in the API response as a warning.

### Requirement 8: 互動式修正 API 端點（Interactive Fix API Endpoint）

**User Story:** As a developer, I want a dedicated API endpoint for submitting interactive cell edits, so that the frontend can send batch corrections to the backend.

#### Acceptance Criteria

1. THE System SHALL expose a POST /api/clean/interactive endpoint that accepts a JSON body containing: assessment_id (UUID, required), and edits (array of edit objects, required).
2. Each edit object in the edits array SHALL contain: row_index (integer, 0-based data row index), col_index (integer, 0-based column index), action (string, one of "replace", "keep", "delete_row", "remark_split", "header_rename"), and value (string, required when action is "replace" or "header_rename", ignored otherwise).
3. WHEN the endpoint receives a valid request, THE Cleaning_Engine SHALL process the edits per Requirement 7 and return a JSON response containing: success (boolean), rows_affected (integer), warnings (array of skipped edit descriptions), and the updated cleaning_log entries.
4. IF the assessment_id does not correspond to an existing assessment or cleaning session, THEN THE System SHALL return HTTP 404.
5. IF the edits array is empty, THEN THE System SHALL return HTTP 400 with an error message indicating no edits were provided.
6. IF the request body fails validation (missing required fields, invalid action values), THEN THE System SHALL return HTTP 400 with a descriptive validation error.

### Requirement 9: 問題嚴重度對 AI Readiness 影響（Issue Severity Impact on AI Readiness）

**User Story:** As a User, I want the new issue types to contribute to the overall AI Readiness Score, so that the score accurately reflects problems that affect AI computation.

#### Acceptance Criteria

1. WHEN Cell_Reference_Placeholder issues are detected in numeric columns, THE Assessment_Engine SHALL reduce the Format Consistency indicator by counting each placeholder cell as a format mismatch in its column.
2. WHEN column_type_mismatch issues are detected, THE Assessment_Engine SHALL incorporate mismatched cells into the existing Format Consistency calculation by treating them as cells not matching the dominant format type.
3. WHEN Orphan_Total_Row issues are detected, THE Assessment_Engine SHALL apply the existing Table Structure deduction for subtotal/total rows (-15) if no keyword-based subtotal rows were already detected, to avoid double-counting.
4. WHEN empty_header issues are detected, THE Assessment_Engine SHALL fail the "column name quality" sub-condition in the AI Query Readiness indicator (deducting 20 points from that indicator).
5. WHEN inline_remark issues are detected in more than 20% of cells in a structured column, THE Assessment_Engine SHALL apply the existing "備註混入資料欄" deduction (-10) to the Table Structure indicator if the deduction was not already applied.

