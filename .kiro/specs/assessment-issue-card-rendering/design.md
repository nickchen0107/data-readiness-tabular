# Assessment Issue Card Rendering Bugfix Design

## Overview

AssessmentPage.tsx 中問題卡片（Issue Card）的展開內容區存在 4 個渲染 Bug，分別影響描述文字排版、多表格分組顯示、跨表高亮、以及表頭列（row_number==1）高亮的 DOM 元素選擇。修復策略為：在同一檔案 `AssessmentPage.tsx` 內調整渲染邏輯，確保每個 Label group 獨立成表、表頭行正確使用 `<th>` 並可高亮、描述文字改用列表或調寬排版。

## Glossary

- **Bug_Condition (C)**: 觸發渲染 Bug 的輸入組合 — 含 `\n` 的描述、多 Label 分組、row_number==1 有 highlights 等
- **Property (P)**: 修正後的期望行為 — 描述清楚換行、每組獨立表格、表頭 th 可高亮
- **Preservation**: 不受影響的既有行為 — 單行描述、單表格、data row 高亮、合併儲存格、收合狀態
- **IssueExample**: 後端回傳的範例資料結構，含 label、headers、row_number、cells、highlights、merges
- **Label Group**: 以 `label` 欄位分組的 IssueExample 集合，代表不同表格/工作表
- **Header Row**: `row_number == 1` 的 IssueExample，代表該表格的表頭列

## Bug Details

### Bug Condition

4 個 Bug 的觸發條件可歸納為一個統一的 Bug Condition 函式：

**Formal Specification:**
```
FUNCTION isBugCondition(issue, example, group)
  INPUT: issue of type Issue, example of type IssueExample, group of type LabelGroup
  OUTPUT: boolean (true if ANY of the 4 bugs is triggered)
  
  // Bug 1: Description with newlines in narrow container
  condition1 := issue.description CONTAINS '\n'
  
  // Bug 2: Multiple label groups rendered as single table
  condition2 := countDistinctLabels(issue.examples) > 1
  
  // Bug 3: Second+ group lacks independent headers and highlight
  condition3 := group.index > 0 AND group.items[0].row_number == 1
  
  // Bug 4: Header row highlights applied to td instead of th
  condition4 := example.row_number == 1 AND example.highlights.length > 0
  
  RETURN condition1 OR condition2 OR condition3 OR condition4
END FUNCTION
```

### Examples

- **Bug 1**: `issue.description = "第1行缺少姓名\n第5行缺少電話\n第8行日期格式錯誤"` → 目前 `pre-line` 在窄容器內導致文字排版破碎；期望每行清楚顯示
- **Bug 2**: `issue.examples` 含 label="表格一" (3 rows) + label="表格二" (2 rows) → 目前全部合併為一張 table；期望分開為 2 張獨立 table（最多 3 張）
- **Bug 3**: 第二個 group 的第一筆 example `row_number=1` → 目前該行被當作 tbody data row 顯示；期望作為新 table 的 thead header row
- **Bug 4**: `example = {row_number: 1, cells: ["姓名","電話","地址"], highlights: [2]}` → 目前第 2 格的紅框標記在 `<td>` 上；期望標記在 `<th>` 上
- **正常情況**: `issue.description = "單行描述"` + 單一 label group + row_number > 1 → 維持現有行為不變

## Expected Behavior

### Preservation Requirements

**Unchanged Behaviors:**
- 不含 `\n` 的描述仍以單段落 inline text 方式顯示
- 只有一個 label group（或無 label）時仍顯示單一 table
- 所有 `row_number > 1` 的 data row 高亮行為維持紅框 td 不變
- 無 highlights 的 cells 維持預設樣式
- 合併儲存格（merges）維持 colspan + 藍色樣式
- 收合狀態下的 card header（標題 / severity badge / description / affected count）不變

**Scope:**
所有不觸發上述 4 個 Bug Condition 的輸入組合應完全不受此修復影響。這包括：
- 單行描述的 Issue cards
- 只有一個 label 的表格渲染
- row_number > 1 且有 highlights 的資料列
- 滑鼠互動（展開收合、hover）

## Hypothesized Root Cause

Based on the bug description and code analysis:

1. **Bug 1 — `white-space: pre-line` 在窄容器**: 描述區使用 `whiteSpace: 'pre-line'` 但外層容器受 flex/minWidth 限制，當文字含多個 `\n` 時，每行獨立但寬度不足造成二次斷行。
   - 根因：應改用結構化列表而非依賴 `pre-line` 呈現多行描述

2. **Bug 2 — 分組邏輯只影響 label 標題，未產生獨立 table**: 目前程式碼在 `groups.map()` 中，每個 group 確實各自渲染一個 `<table>`，但問題在於 format mixing 類型的 issue 可能所有 examples 的 label 都相同或程式將它們視為同一 group。
   - 根因：需確認分組邏輯是否正確處理後端回傳的 label 欄位；且需限制最多顯示 3 個 table

3. **Bug 3 — 第二張表缺少獨立 header**: 目前每個 group 共用 `group.items[0].headers` 作為 `<thead>`，但所有 items 都渲染在 `<tbody>` 中。如果 group 的第一筆 `row_number == 1`，它應該作為 thead 的資料來源而非出現在 tbody。
   - 根因：渲染邏輯未區分 `row_number == 1` 的行，一律放入 tbody `<tr>`

4. **Bug 4 — header row 高亮標記在 td**: 與 Bug 3 相關。當 `row_number == 1` 的行被放在 tbody 時，其 cells 自然以 `<td>` 呈現。即使移到 thead，現有的 highlight 邏輯也未套用到 `<th>` 上。
   - 根因：highlight 邏輯只在 tbody cells 渲染處判斷，thead 的 `<th>` 沒有對應的 highlight 邏輯

## Correctness Properties

Property 1: Bug Condition - Multi-line Description Renders Clearly

_For any_ Issue where the description contains `\n` characters, the fixed rendering function SHALL display each line segment as a distinct visual element (list item or block) without awkward mid-word wrapping caused by container width constraints.

**Validates: Requirements 2.1**

Property 2: Bug Condition - Multiple Label Groups Render as Separate Tables

_For any_ Issue whose examples contain more than one distinct Label value, the fixed rendering function SHALL produce separate independent `<table>` elements per Label group (up to a maximum of 3), each with its own border, header row, and data rows.

**Validates: Requirements 2.2, 2.3**

Property 3: Bug Condition - Header Row Uses th with Highlights

_For any_ IssueExample where `row_number == 1` and `highlights` is non-empty, the fixed rendering function SHALL render highlighted cells as `<th>` elements with red border/background styling, not as `<td>` elements in tbody.

**Validates: Requirements 2.3, 2.4**

Property 4: Preservation - Existing Single-Group Rendering Unchanged

_For any_ Issue where the description contains no `\n`, examples have a single Label group, and all examples have `row_number > 1`, the fixed rendering function SHALL produce the same DOM structure and styling as the original function, preserving all existing behaviors.

**Validates: Requirements 3.1, 3.2, 3.3, 3.4, 3.5, 3.6**

## Fix Implementation

### Changes Required

Assuming our root cause analysis is correct:

**File**: `frontend/src/pages/AssessmentPage.tsx`

**Scope**: Issue card 展開區的渲染邏輯（約第 200-320 行）

**Specific Changes**:

1. **Bug 1 — 描述文字排版**: 
   - 檢測 `issue.description` 是否含 `\n`
   - 若含 `\n`，將文字 split 為陣列，以 `<ul><li>` 或 `<div>` per line 渲染，取代單一 `<div>` + `pre-line`
   - 若不含 `\n`，維持原有單段落渲染

2. **Bug 2 — 多 Label Group 最多 3 個獨立 table**:
   - 在 groups 切片時加上 `.slice(0, 3)` 限制最多 3 個
   - 確認現有分組邏輯正確運作（按 label 相鄰分組）
   - 每個 group 的 `<div>` wrapper 保持獨立的 border + borderRadius

3. **Bug 3 — 每個 Group 獨立 Header 邏輯**:
   - 在每個 group 渲染時，檢查 `group.items[0].row_number == 1`
   - 若為 header row：用其 `cells` 作為 `<thead><tr><th>` 內容（取代硬編碼的 `headers` 欄位）
   - 將該 row 從 `<tbody>` 渲染中排除
   - 若 group 第一行 `row_number > 1`：維持使用 `headers` 欄位作為表頭

4. **Bug 4 — Header row 的 highlights 套用到 th**:
   - 在新的 thead 渲染邏輯中，讀取 header row example 的 `highlights` 陣列
   - 對 highlights 中指定的 column index，為對應的 `<th>` 套用紅框 + 紅色背景樣式
   - 保留非 highlighted th 的原有灰色樣式

5. **Merge 處理整合**:
   - thead 中的 th 也需要支援 `merges` 邏輯（colspan）
   - 確保 highlight + merge 同時存在時正確處理

## Testing Strategy

### Validation Approach

測試策略分兩階段：第一階段在未修復程式碼上跑測試確認 Bug 重現，第二階段修復後驗證正確性與 preservation。

### Exploratory Bug Condition Checking

**Goal**: 在未修復的程式碼上撰寫測試 surface counterexamples，確認 root cause 分析正確。

**Test Plan**: 使用 React Testing Library + Vitest 渲染 Issue Card 的展開區域，檢查 DOM 結構是否符合預期。在未修復程式碼上，這些測試應該失敗。

**Test Cases**:
1. **Multi-line Description Test**: 渲染含 `\n` 的 description，檢查是否有 `<li>` 或獨立 block 元素（will fail on unfixed code — 目前只有一個 pre-line div）
2. **Multiple Label Groups Test**: 傳入 2 個不同 label 的 examples，檢查是否存在 2 個獨立 `<table>` 元素（will fail if grouping logic is broken）
3. **Header Row th Test**: 傳入 `row_number=1` + highlights 的 example，檢查 `<th>` 元素是否有紅框樣式（will fail — 目前 highlights 只在 td）
4. **Second Group Header Test**: 傳入兩個 group，第二個 group 首行 row_number=1，檢查第二張 table 的 thead 是否使用該行 cells（will fail on unfixed code）

**Expected Counterexamples**:
- Description 以單一 div 渲染，無結構化 list items
- 多 label 但只有一個 table 元素
- `<th>` 沒有紅框樣式而 `<td>` 有
- 第二 group 的 header 使用了第一 group 的 headers 欄位

### Fix Checking

**Goal**: 驗證對所有觸發 Bug Condition 的輸入，修復後函式產生正確行為。

**Pseudocode:**
```
FOR ALL input WHERE isBugCondition(input) DO
  dom := render_IssueCard_fixed(input)
  IF input.description CONTAINS '\n' THEN
    ASSERT dom HAS structured_line_elements(count = split('\n').length)
  IF countDistinctLabels(input.examples) > 1 THEN
    ASSERT dom HAS separate_tables(count = min(distinctLabels, 3))
  IF example.row_number == 1 AND example.highlights.length > 0 THEN
    ASSERT dom.thead.th[highlighted_indices] HAS red_border_style
  ASSERT expectedBehavior(dom)
END FOR
```

### Preservation Checking

**Goal**: 驗證對所有不觸發 Bug Condition 的輸入，修復後函式產生與原始函式相同的 DOM 結構。

**Pseudocode:**
```
FOR ALL input WHERE NOT isBugCondition(input) DO
  ASSERT render_IssueCard_original(input) = render_IssueCard_fixed(input)
END FOR
```

**Testing Approach**: Property-based testing 特別適合 preservation checking，因為：
- 可自動產生大量隨機的 Issue + IssueExample 組合
- 可捕捉手動測試遺漏的 edge cases（如空 label、0 highlights、超長文字）
- 對 non-buggy inputs 的行為一致性提供強保證

**Test Plan**: 先在未修復程式碼上記錄正常輸入的渲染結果（snapshot），修復後確認這些 snapshot 不變。

**Test Cases**:
1. **Single-line Description Preservation**: 單行描述仍以段落呈現，無 list 元素
2. **Single Label Group Preservation**: 只有一個 label 時仍為單一 table，結構不變
3. **Data Row Highlight Preservation**: `row_number > 1` 的高亮維持 td 紅框
4. **Merge Styling Preservation**: 合併儲存格維持 colspan + 藍色樣式
5. **Collapse State Preservation**: 收合卡片的 header 區域 DOM 結構不變

### Unit Tests

- 測試 description 含 `\n` 時渲染為 list items 的邏輯
- 測試 label 分組函式對各種 label 組合的正確性
- 測試 `row_number == 1` 的行是否正確判定為 header row
- 測試 highlight 索引是否正確套用到 th 與 td
- 測試 groups 最多 3 個的截斷邏輯

### Property-Based Tests

- 產生隨機 Issue.description（含/不含 `\n`），驗證渲染結構正確切換
- 產生隨機數量的 label groups（1~5 個），驗證最多渲染 3 個 table 且各自獨立
- 產生隨機 row_number + highlights 組合，驗證高亮永遠標記在正確的 DOM 元素（th vs td）
- 產生不觸發 Bug Condition 的隨機輸入，驗證 DOM 結構與修復前一致

### Integration Tests

- 完整 AssessmentPage 渲染含多 Bug Condition 的 assessment data，驗證 UI 整體一致性
- 展開/收合互動後 DOM 結構正確
- 多個 issue cards 同時展開時各自渲染獨立，互不影響
