# Implementation Plan

## Overview

修復匯出模組（export package）三個缺陷：Excel 檔名固定、PDF 中文亂碼靜默降級、cleaning.log 原始 JSON 格式。
採用 Bug Condition 方法論：先撰寫測試確認 bug 存在，再撰寫 preservation 測試捕捉基線行為，接著實施修復並驗證。

## Tasks

- [x] 1. 撰寫 Bug Condition 探索測試（修復前）
  - **Property 1: Bug Condition** - 匯出模組三缺陷確認
  - **CRITICAL**: 此測試必須在實施修復前撰寫並執行，預期 FAIL
  - **DO NOT attempt to fix the test or the code when it fails**
  - **NOTE**: 此測試編碼了期望行為 — 修復後通過即驗證修復成功
  - **GOAL**: 產生反例（counterexamples）以證明 bug 存在
  - **Scoped PBT Approach**: 針對三個具體 bug 條件各設計 property
  - Bug 1 — Excel 檔名：建立含 OriginalFilename 的 CleaningSession，呼叫 DownloadExcel，斷言 Content-Disposition 包含 `{原始檔名}_refined.xlsx`（isBugCondition: exportType = "xlsx"）
  - Bug 2 — PDF 字型缺失：設定不存在的 FontPath，呼叫 GeneratePDF，斷言回傳 error 且 error message 包含「字型」（isBugCondition: NOT fileExists(config.Report.FontPath)）
  - Bug 3 — Log 格式：呼叫 GenerateLog，斷言每行匹配 `^\[\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\] .+` 且不包含 `"operation_type"` 或 `"affected_rows"` 字串（isBugCondition: exportType = "log"）
  - 在 `backend/internal/export/export_bugfix_test.go` 撰寫 property-based test（使用 `testing/quick` 或 `gopter`）
  - 在未修復程式碼上執行測試
  - **EXPECTED OUTCOME**: 測試 FAIL（此為正確結果 — 證明 bug 存在）
  - 記錄發現的 counterexamples（例：OriginalFilename="客戶名單.xlsx" 但 Content-Disposition 為 "refined.xlsx"）
  - 任務完成標記：測試已撰寫、已執行、失敗已記錄
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5_

- [x] 2. 撰寫 Preservation Property 測試（修復前）
  - **Property 2: Preservation** - 匯出模組既有行為保全
  - **IMPORTANT**: 遵循觀察優先方法論（observation-first methodology）
  - 觀察：在未修復程式碼上呼叫 GenerateExcel，確認產出的 xlsx 包含正確的 headers、rows、column widths
  - 觀察：在未修復程式碼上呼叫 GeneratePDF（字型存在），確認產出完整 PDF 報告
  - 觀察：在未修復程式碼上呼叫 GenerateLog，確認輸出包含所有 LogEntry 的 OperationType、AffectedRows、Timestamp
  - 撰寫 property-based test：
    - Property 2a: 對所有隨機生成的 (headers, rows) 組合，GenerateExcel 產出的 xlsx 內容與輸入一致
    - Property 2b: 對所有有效 config（字型存在），GeneratePDF 回傳 nil error 且產出檔案存在
    - Property 2c: 對所有隨機生成的 CleaningSession（含多種 OperationType、不同長度 AffectedRows），GenerateLog 輸出包含每筆 entry 的所有資訊
  - 在 `backend/internal/export/export_preservation_test.go` 撰寫測試
  - 在未修復程式碼上執行測試
  - **EXPECTED OUTCOME**: 測試 PASS（確認基線行為已捕捉）
  - 任務完成標記：測試已撰寫、已執行、且在未修復程式碼上通過
  - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5_

- [x] 3. 修復匯出模組三缺陷

  - [x] 3.1 新增 OriginalFilename 欄位至 CleaningSession
    - 在 `backend/internal/cleaning/model.go` 的 `CleaningSession` struct 新增 `OriginalFilename string` 欄位
    - JSON tag: `json:"original_filename"`
    - DB tag: `db:"original_filename"`
    - _Bug_Condition: isBugCondition(input) where exportType = "xlsx"_
    - _Requirements: 2.1, 2.2_

  - [x] 3.2 資料庫 migration — 新增 original_filename 欄位
    - 建立 migration 檔案為 `cleaning_sessions` 表新增 `original_filename VARCHAR NULL` 欄位
    - 歷史資料 OriginalFilename 為 NULL，對應回退行為
    - _Requirements: 2.2_

  - [x] 3.3 更新 repository.go — SELECT/INSERT 包含新欄位
    - 在 `backend/internal/cleaning/repository.go` 的 SELECT 語句加入 `original_filename`
    - 在 INSERT 語句加入 `original_filename` 參數
    - 確保 Scan 正確處理 NULL 值（使用 `sql.NullString` 或指標）
    - _Requirements: 2.1, 2.2_

  - [x] 3.4 修復 handler.go DownloadExcel — 動態檔名 + RFC 5987 編碼
    - 將 `filename := "refined.xlsx"` 改為依據 `session.OriginalFilename` 動態產生
    - 若 OriginalFilename 非空：使用 `filepath.Base` 防止路徑穿越，`strings.TrimSuffix` 去除 `.xlsx`/`.csv` 副檔名，拼接 `_refined.xlsx`
    - 若 OriginalFilename 為空（歷史資料回退）：使用 `refined.xlsx`
    - Content-Disposition 使用 RFC 5987 UTF-8 編碼：`filename*=UTF-8''` + `url.PathEscape(filename)` 以支援中文檔名
    - 同時保留 ASCII `filename` 參數作為不支援 RFC 5987 的客戶端回退
    - _Bug_Condition: isBugCondition(input) where exportType = "xlsx", ALL xlsx exports affected_
    - _Expected_Behavior: Content-Disposition contains "{stripExtension(OriginalFilename)}_refined.xlsx" or fallback "refined.xlsx"_
    - _Preservation: Excel 檔案內容（headers, rows, widths）不受影響_
    - _Requirements: 2.1, 2.2, 3.1_

  - [x] 3.5 修復 pdf.go GeneratePDF — 字型不存在時回傳錯誤
    - 在 `GeneratePDF` 函式開頭（建立 fpdf.New 之前），檢查 `cfg.Report.FontPath`
    - 若 FontPath 為空字串或檔案不存在：回傳 `fmt.Errorf("中文字型檔案未安裝，無法產生 PDF 報告: %s", cfg.Report.FontPath)`
    - 移除 `hasChinese` 靜默降級邏輯，字型不存在即為錯誤
    - 保留字型存在時的完整 PDF 產生邏輯不變
    - _Bug_Condition: isBugCondition(input) where NOT fileExists(config.Report.FontPath)_
    - _Expected_Behavior: GeneratePDF returns non-nil error containing "字型"_
    - _Preservation: 字型存在時 PDF 正常產出，內容不變_
    - _Requirements: 2.3, 2.4, 3.2_

  - [x] 3.6 重寫 log.go GenerateLog — 人類可讀格式
    - 移除 `json.MarshalIndent` 呼叫
    - 遍歷 `session.CleaningLog`，對每筆 LogEntry 呼叫新增的 `formatLogEntry` 函式
    - 新增 `formatLogEntry(entry LogEntry) string` 函式：
      - 格式：`[2006-01-02 15:04:05] {中文操作描述}`
      - 操作描述包含 AffectedRows（逗號分隔），如「移除重複列：第 5, 12, 23 列」
    - 新增 `operationTypeLabel(opType string) string` 函式：
      - `dedup` → `移除重複列`
      - `date_normalize` → `統一日期格式`
      - `name_normalize` → `客戶名正規化`
      - `subtotal_remove` → `移除小計列`
      - 未知類型 → 直接使用原始字串
    - AffectedRows 為空時省略列號部分，僅顯示操作描述
    - Details 非空時附加於描述末尾
    - 以 `\n` 連接所有行並寫入檔案
    - _Bug_Condition: isBugCondition(input) where exportType = "log", ALL log exports affected_
    - _Expected_Behavior: 每行匹配 ^\[\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\] .+, 不含 JSON 欄位名_
    - _Preservation: 所有 LogEntry 資訊（Timestamp, OperationType, AffectedRows, Details）均保留於輸出中_
    - _Requirements: 2.5, 2.6, 3.3_

  - [x] 3.7 修復 handler.go DownloadLog — Content-Type 修正
    - 將 `c.Header("Content-Type", "application/json")` 改為 `c.Header("Content-Type", "text/plain; charset=utf-8")`
    - _Requirements: 2.5_

  - [x] 3.8 驗證 Bug Condition 探索測試通過
    - **Property 1: Expected Behavior** - 匯出模組三缺陷已修復
    - **IMPORTANT**: 重新執行任務 1 的同一測試 — 不要撰寫新測試
    - 任務 1 的測試編碼了期望行為，通過即代表修復成功
    - 執行 `go test ./internal/export/ -run TestBugCondition -v`
    - **EXPECTED OUTCOME**: 測試 PASS（確認 bug 已修復）
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6_

  - [x] 3.9 驗證 Preservation 測試仍然通過
    - **Property 2: Preservation** - 匯出模組既有行為未受影響
    - **IMPORTANT**: 重新執行任務 2 的同一測試 — 不要撰寫新測試
    - 執行 `go test ./internal/export/ -run TestPreservation -v`
    - **EXPECTED OUTCOME**: 測試 PASS（確認無回歸）
    - 確認所有 preservation 測試在修復後仍然通過
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5_

- [x] 4. Checkpoint — 確認所有測試通過
  - 執行完整測試套件：`go test ./internal/export/ -v`
  - 確認 Bug Condition 測試全部 PASS
  - 確認 Preservation 測試全部 PASS
  - 確認無編譯錯誤
  - 若有問題，詢問使用者確認方向

## Task Dependency Graph

```json
{
  "waves": [
    {"tasks": ["1", "2"]},
    {"tasks": ["3.1", "3.5", "3.6"]},
    {"tasks": ["3.2", "3.3", "3.7"]},
    {"tasks": ["3.4"]},
    {"tasks": ["3.8"]},
    {"tasks": ["3.9"]},
    {"tasks": ["4"]}
  ]
}
```

## Notes

- 本修復使用 Go 標準 `testing/quick` 或社群 `pgregory.net/rapid` 進行 property-based testing
- PDF 修復移除靜默降級邏輯，字型不存在時 handler 層會回傳 HTTP 500 含明確錯誤訊息
- Log 格式變更為 breaking change — 下游如有消費 JSON 格式的程式需同步更新
- OriginalFilename 為 nullable column，歷史資料自動使用回退檔名 `refined.xlsx`
