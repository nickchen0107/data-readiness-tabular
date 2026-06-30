# 匯出模組錯誤修復技術設計

## Overview

SAFE-AI Excel 梳理小工具的匯出模組（`backend/internal/export`）存在三個缺陷需要修復：

1. **Excel 檔名固定問題** — `handler.go` 中 `DownloadExcel` 方法硬編碼 `refined.xlsx` 作為下載檔名，未使用原始上傳檔案名稱
2. **PDF 中文亂碼問題** — `pdf.go` 中字型不存在時靜默降級至 Helvetica，導致中文亂碼而非回報錯誤
3. **清理日誌格式問題** — `log.go` 使用 `json.MarshalIndent` 輸出原始 JSON，非人類可讀格式

修復策略為：最小化變更範圍，僅修改必要檔案，並確保既有功能（Excel 內容、PDF 完整性、日誌資訊完整性）不受影響。

## Glossary

- **Bug_Condition (C)**: 觸發各缺陷的輸入條件
- **Property (P)**: 修復後各情境下的期望行為
- **Preservation**: 修復不得影響的既有行為（Excel 內容完整性、PDF 正常產出、日誌資訊完整性）
- **CleaningSession**: `cleaning/model.go` 中代表一次清理操作的資料結構
- **LogEntry**: `cleaning/model.go` 中代表單一清理操作的日誌紀錄結構
- **OriginalFilename**: 需新增至 `CleaningSession` 的欄位，記錄使用者上傳的原始檔案名稱
- **GeneratePDF**: `pdf.go` 中產生 PDF 報告的函式
- **GenerateLog**: `log.go` 中產生清理日誌的函式
- **DownloadExcel**: `handler.go` 中處理 Excel 下載的 HTTP handler

## Bug Details

### Bug Condition

三個 bug 各有不同觸發條件：

**Bug 1 — Excel 檔名固定**：所有 Excel 匯出請求都受影響（handler 硬編碼檔名）

**Bug 2 — PDF 中文亂碼**：當設定的字型檔案不存在時觸發（`pdf.go` 第 41-49 行靜默降級）

**Bug 3 — cleaning.log 格式**：所有 log 匯出請求都受影響（`log.go` 使用 `json.MarshalIndent`）

**Formal Specification:**
```
FUNCTION isBugCondition(input)
  INPUT: input of type ExportRequest
  OUTPUT: boolean
  
  SWITCH input.exportType:
    CASE "xlsx":
      // Bug 1: 所有 Excel 匯出都使用固定檔名
      RETURN TRUE
    CASE "pdf":
      // Bug 2: 字型檔案不存在時觸發
      RETURN NOT fileExists(input.config.Report.FontPath)
    CASE "log":
      // Bug 3: 所有 log 匯出都輸出原始 JSON
      RETURN TRUE
  END SWITCH
END FUNCTION
```

### Examples

- 使用者上傳「客戶名單.xlsx」→ 梳理完成後匯出 → 檔名為 `refined.xlsx`（期望：`客戶名單_refined.xlsx`）
- Docker 容器中 `/app/assets/fonts/NotoSansTC-Regular.ttf` 不存在 → PDF 產生成功但中文全為亂碼（期望：回傳錯誤）
- 使用者匯出 cleaning.log → 得到 `[{"operation_type":"dedup","affected_rows":[5,12],...}]`（期望：`[2024-03-15 14:23:01] 移除重複列：第 5, 12 列`）
- 歷史 session 無 OriginalFilename → 匯出 Excel → 檔名為 `refined.xlsx`（回退行為，正確）

## Expected Behavior

### Preservation Requirements

**Unchanged Behaviors:**
- Excel 檔案內容（欄位標題、資料列、欄位寬度、樣式）必須維持不變
- PDF 報告在字型存在時必須繼續正常產出完整內容（分數摘要、六項指標表格、問題摘要、梳理前後對比）
- cleaning.log 必須包含所有 LogEntry 資訊（OperationType、AffectedRows、Timestamp、OperatorID、Details）
- JWT 所有權驗證邏輯不受影響
- 匯出快取機制不受影響（已快取檔案直接回傳）
- PDF 與 Excel 的實際生成邏輯（`GenerateExcel`、`GeneratePDF`）在字型存在情境下不受影響

**Scope:**
所有不涉及以下三點的輸入/操作均不受此修復影響：
- Excel 匯出時的 Content-Disposition 標頭生成
- PDF 產生時的字型載入邏輯
- Log 匯出時的格式化邏輯

## Hypothesized Root Cause

Based on the bug description and source code analysis, the root causes are:

1. **Bug 1 — Excel 檔名硬編碼**
   - `handler.go` 第 42 行：`filename := "refined.xlsx"` 直接硬編碼
   - `CleaningSession` struct 缺少 `OriginalFilename` 欄位
   - 上傳流程未保存原始檔名至 session
   - 即使有原始檔名，handler 也未使用它

2. **Bug 2 — PDF 字型靜默降級**
   - `pdf.go` 第 41-49 行：使用 `if/else` 靜默設定 `hasChinese = false`
   - 未回傳任何錯誤，`setFont` 閉包在 `hasChinese = false` 時使用 Helvetica
   - 後續所有中文字串寫入 PDF 時無法正確渲染
   - 函式回傳成功，caller 無法得知問題

3. **Bug 3 — Log 使用 JSON 格式**
   - `log.go` 第 20 行：`json.MarshalIndent(session.CleaningLog, "", "  ")`
   - 直接序列化整個 `[]LogEntry` struct 為 JSON
   - 未提供任何格式化邏輯將 struct 欄位轉換為人類可讀描述
   - `LogEntry` 的 `OperationType` 為程式化字串（如 `dedup`）

## Correctness Properties

Property 1: Bug Condition - Excel 檔名延伸原始檔名

_For any_ 匯出請求，當 session 記錄了 OriginalFilename 時，修復後的 DownloadExcel handler SHALL 在 Content-Disposition 標頭中使用 `{原始檔名去除副檔名}_refined.xlsx` 格式；當 OriginalFilename 為空時，SHALL 回退使用 `refined.xlsx`。

**Validates: Requirements 2.1, 2.2**

Property 2: Bug Condition - PDF 字型不存在時回傳錯誤

_For any_ PDF 產生請求，當設定的字型檔案路徑不存在時，修復後的 GeneratePDF 函式 SHALL 回傳包含「字型」關鍵字的 error，且不產生任何 PDF 檔案。

**Validates: Requirements 2.3, 2.4**

Property 3: Bug Condition - Log 輸出人類可讀格式

_For any_ log 匯出請求，修復後的 GenerateLog 函式 SHALL 輸出逐行格式，每行符合 `[YYYY-MM-DD HH:MM:SS] {操作描述}` 模式，不包含 JSON 欄位名稱如 `operation_type` 或 `affected_rows`。

**Validates: Requirements 2.5, 2.6**

Property 4: Preservation - Excel 內容完整性

_For any_ Excel 匯出請求，修復後生成的 .xlsx 檔案 SHALL 包含與修復前完全相同的欄位標題、資料列及欄位寬度，僅 Content-Disposition 標頭檔名不同。

**Validates: Requirements 3.1, 3.5**

Property 5: Preservation - PDF 正常情境完整性

_For any_ PDF 產生請求，當字型檔案存在時，修復後的 GeneratePDF 函式 SHALL 產出與修復前完全相同的 PDF 報告內容。

**Validates: Requirements 3.2, 3.5**

Property 6: Preservation - Log 資訊完整性

_For any_ log 匯出請求，修復後的輸出 SHALL 包含每一筆 LogEntry 的所有資訊（Timestamp、OperationType 的人類可讀描述、AffectedRows 列號、Details），且不遺漏任何紀錄。

**Validates: Requirements 3.3**

## Fix Implementation

### Changes Required

Assuming our root cause analysis is correct:

**File**: `backend/internal/cleaning/model.go`

**Change**: 新增 `OriginalFilename` 欄位至 `CleaningSession` struct

**Specific Changes**:
1. **新增欄位**: 在 `CleaningSession` struct 中新增 `OriginalFilename string` 欄位（含 json tag 及 db tag）

---

**File**: `backend/internal/export/handler.go`

**Function**: `DownloadExcel`

**Specific Changes**:
1. **動態檔名生成**: 將 `filename := "refined.xlsx"` 改為依據 `session.OriginalFilename` 動態產生
2. **檔名邏輯**: 若 `session.OriginalFilename` 非空，去除副檔名後加上 `_refined.xlsx` 後綴；否則回退使用 `refined.xlsx`
3. **安全處理**: 使用 `filepath.Base` 防止路徑穿越，使用 `strings.TrimSuffix` 去除原始副檔名
4. **UTF-8 編碼**: Content-Disposition 使用 RFC 5987 `filename*=UTF-8''` 編碼以支援中文檔名

---

**File**: `backend/internal/export/pdf.go`

**Function**: `GeneratePDF`

**Specific Changes**:
1. **字型檢查提前**: 在建立 `fpdf.New()` 之前，先檢查 `cfg.Report.FontPath` 是否存在
2. **回傳明確錯誤**: 若字型檔案不存在，回傳 `fmt.Errorf("中文字型檔案未安裝，無法產生 PDF 報告: %s", cfg.Report.FontPath)` 
3. **移除靜默降級**: 刪除 `hasChinese` 分支邏輯，字型不存在即視為錯誤
4. **保持向後相容**: 若 `cfg.Report.FontPath` 為空字串（未設定），也回傳錯誤提示

---

**File**: `backend/internal/export/log.go`

**Function**: `GenerateLog`

**Specific Changes**:
1. **替換 JSON 序列化**: 移除 `json.MarshalIndent` 呼叫
2. **逐行格式化**: 遍歷 `session.CleaningLog`，將每筆 `LogEntry` 轉換為 `[timestamp] 操作描述` 格式
3. **新增 formatLogEntry 函式**: 將 `LogEntry` 轉為人類可讀描述
4. **OperationType 中文映射**: 建立 operationType → 中文描述的映射（dedup→移除重複列、date_normalize→統一日期格式、name_normalize→客戶名正規化、subtotal_remove→移除小計列）
5. **AffectedRows 格式化**: 將 `[]int` 以逗號分隔嵌入描述中（如「第 5, 12, 23 列」）

---

**File**: `backend/internal/export/handler.go`

**Function**: `DownloadLog`

**Specific Changes**:
1. **Content-Type 修正**: 將 `application/json` 改為 `text/plain; charset=utf-8` 以反映新格式

---

**File**: `backend/internal/cleaning/repository.go`（及相關 migration）

**Specific Changes**:
1. **資料庫 migration**: 為 `cleaning_sessions` 表新增 `original_filename` 欄位（VARCHAR, nullable）
2. **Repository 更新**: 確保 SELECT/INSERT 語句包含新欄位

## Testing Strategy

### Validation Approach

測試策略採用兩階段方法：首先在未修復程式碼上撰寫測試確認 bug 行為，接著驗證修復後的正確性與既有行為保全。

### Exploratory Bug Condition Checking

**Goal**: 在實施修復前，撰寫測試展示 bug 的存在，確認根因分析正確。

**Test Plan**: 撰寫單元測試直接呼叫 handler/函式，觀察未修復程式碼的行為。

**Test Cases**:
1. **Excel 檔名測試**: 建立含有 OriginalFilename 的 session，呼叫 DownloadExcel，驗證 Content-Disposition 仍為 `refined.xlsx`（未修復程式碼上失敗）
2. **PDF 字型缺失測試**: 設定不存在的字型路徑，呼叫 GeneratePDF，驗證函式回傳 nil error 且產生 PDF（未修復程式碼上失敗——應回傳 error 但實際回傳 nil）
3. **Log 格式測試**: 呼叫 GenerateLog，驗證輸出是否包含 `[` timestamp `]` 行格式（未修復程式碼上失敗——輸出為 JSON）
4. **Log JSON 欄位測試**: 呼叫 GenerateLog，驗證輸出不包含 `"operation_type"` 字串（未修復程式碼上失敗）

**Expected Counterexamples**:
- DownloadExcel 始終使用 `refined.xlsx`，忽略 OriginalFilename
- GeneratePDF 在字型不存在時回傳 nil error，產出含亂碼的 PDF
- GenerateLog 輸出包含 `"operation_type"`、`"affected_rows"` 等 JSON key

### Fix Checking

**Goal**: 驗證所有觸發 bug 條件的輸入，修復後函式產生期望行為。

**Pseudocode:**
```
// Bug 1: Excel 檔名
FOR ALL session WHERE session.OriginalFilename ≠ "" DO
  result := DownloadExcel'(session)
  expected := stripExtension(session.OriginalFilename) + "_refined.xlsx"
  ASSERT result.ContentDisposition contains expected
END FOR

FOR ALL session WHERE session.OriginalFilename = "" DO
  result := DownloadExcel'(session)
  ASSERT result.ContentDisposition contains "refined.xlsx"
END FOR

// Bug 2: PDF 字型錯誤
FOR ALL config WHERE NOT fileExists(config.Report.FontPath) DO
  result := GeneratePDF'(data, config, outputDir)
  ASSERT result.error ≠ nil
  ASSERT result.error.message contains "字型"
END FOR

// Bug 3: Log 格式
FOR ALL session WITH non-empty CleaningLog DO
  result := GenerateLog'(session, outputDir)
  lines := splitLines(readFile(result.filePath))
  FOR EACH line IN lines WHERE line ≠ "" DO
    ASSERT line matches "^\[\\d{4}-\\d{2}-\\d{2} \\d{2}:\\d{2}:\\d{2}\] .+"
  END FOR
  ASSERT fileContent does NOT contain "operation_type"
  ASSERT fileContent does NOT contain "affected_rows"
END FOR
```

### Preservation Checking

**Goal**: 驗證所有不觸發 bug 條件的輸入，修復後函式與原始函式行為一致。

**Pseudocode:**
```
// Preservation 1: Excel 內容不變
FOR ALL session DO
  excelContent_F  := GenerateExcel(session, headers, rows, dir)
  excelContent_F' := GenerateExcel'(session, headers, rows, dir)
  ASSERT readXlsxData(excelContent_F) = readXlsxData(excelContent_F')
END FOR

// Preservation 2: PDF 字型存在時行為不變
FOR ALL config WHERE fileExists(config.Report.FontPath) DO
  ASSERT GeneratePDF(data, config, dir) = GeneratePDF'(data, config, dir)
END FOR

// Preservation 3: Log 包含所有資訊
FOR ALL session DO
  logContent := readFile(GenerateLog'(session, dir))
  FOR EACH entry IN session.CleaningLog DO
    ASSERT logContent contains formattedTimestamp(entry.Timestamp)
    ASSERT logContent contains humanReadableOp(entry.OperationType)
    FOR EACH row IN entry.AffectedRows DO
      ASSERT logContent contains toString(row)
    END FOR
  END FOR
END FOR
```

**Testing Approach**: Property-based testing is recommended for preservation checking because:
- 可自動生成大量隨機 CleaningSession / LogEntry 組合
- 能捕捉手動測試難以覆蓋的邊界案例（空 AffectedRows、特殊字元檔名等）
- 提供強保證：所有非 bug 輸入行為不變

**Test Plan**: 先在未修復程式碼上觀察正常行為，再撰寫 property-based 測試確認修復後行為一致。

**Test Cases**:
1. **Excel 內容保全**: 驗證修復前後 GenerateExcel 產出的 xlsx 內容完全一致
2. **PDF 正常產出保全**: 字型存在時驗證 GeneratePDF 修復前後行為一致
3. **Log 資訊完整保全**: 驗證每筆 LogEntry 的 Timestamp、OperationType、AffectedRows、Details 都出現在輸出中
4. **JWT 驗證保全**: 驗證未授權請求仍回傳 401/403
5. **快取機制保全**: 驗證已存在的匯出檔案直接回傳不重新生成

### Unit Tests

- `handler_test.go`: 測試 DownloadExcel 對不同 OriginalFilename 的 Content-Disposition 輸出
- `pdf_test.go`: 測試字型不存在時 GeneratePDF 回傳 error
- `pdf_test.go`: 測試字型存在時 GeneratePDF 正常產出
- `log_test.go`: 測試 GenerateLog 輸出格式正確性
- `log_test.go`: 測試各 OperationType 的中文映射正確性
- `log_test.go`: 測試 AffectedRows 為空陣列時的處理

### Property-Based Tests

- 隨機生成 OriginalFilename（含中文、特殊字元、空白）→ 驗證檔名格式正確且安全
- 隨機生成 LogEntry 陣列（不同 OperationType、不同長度 AffectedRows）→ 驗證每行格式符合 pattern
- 隨機生成 CleaningSession → 驗證 log 輸出包含所有 entry 資訊（preservation）
- 隨機生成設定 → 驗證字型路徑不存在時必回傳 error、存在時必回傳 nil error

### Integration Tests

- 完整匯出流程：上傳檔案 → 梳理 → 匯出 Excel → 驗證檔名含原始名
- PDF 匯出流程：正常環境匯出 PDF → 驗證包含完整報告內容
- Log 匯出流程：完成多步驟梳理 → 匯出 log → 驗證格式可讀且資訊完整
- 錯誤處理流程：字型缺失時匯出 PDF → 驗證 HTTP 回應為 500 含明確錯誤訊息
