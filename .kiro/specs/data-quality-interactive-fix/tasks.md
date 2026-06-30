# Implementation Plan: Data Quality Interactive Fix

## Overview

擴充 SAFE-AI Excel 梳理小工具的 Assessment Engine（新增 5 種語意層級問題偵測）與 Cleaning Engine（互動式儲存格修正），並在前端新增 CellEditor 元件。實作採用增量式推進：先完成偵測函式，再整合分數計算，接著建立互動式修正後端，最後實作前端 UI。

## Tasks

- [x] 1. 新增偵測函式基礎設施
  - [x] 1.1 實作 DetectCellReferencePlaceholders 偵測函式
    - 在 `backend/internal/assessment/issues.go` 新增 `DetectCellReferencePlaceholders(data *upload.SheetData) []Issue`
    - 使用 regex `^同[A-Z]+\d+$` 比對每個 cell
    - 回傳 Issue（Title: "儲存格引用佔位符", Severity: "High", indicator: "cell_reference_placeholder"）
    - 若偵測到的 cell 位於 numeric 欄位，Description 加註 AI 計算不可行
    - 提供 up to 5 examples（使用現有 `limitExamples`, `selectDisplayColumns` helper）
    - 在 `DetectIssues()` 中呼叫此函式並 append 結果
    - _Requirements: 1.1, 1.2, 1.3, 1.4_

  - [x] 1.2 實作 DetectColumnTypeMismatch 偵測函式
    - 在 `backend/internal/assessment/issues.go` 新增 `DetectColumnTypeMismatch(data *upload.SheetData) []Issue`
    - 欄位型別推斷邏輯：去除空白、移除貨幣符號（NT$, USD, $, ¥, €）、移除千分位逗號、嘗試 ParseFloat
    - 超過 70% non-empty cells 為 numeric → 欄位為 "numeric" type
    - 標記所有非 numeric、非 empty 的 cell
    - Severity: >10% mismatch → "High"，≤10% → "Medium"
    - 在 `DetectIssues()` 中呼叫此函式並 append 結果
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5_

  - [x] 1.3 實作 DetectInlineRemarks 偵測函式
    - 在 `backend/internal/assessment/issues.go` 新增 `DetectInlineRemarks(data *upload.SheetData) []Issue`
    - 結構欄位判定：>60% non-empty cells 符合 alphanumeric identifier pattern
    - 標記含括號內容（半形 `()` 或全形 `（）`）且括號內含中文字 OR 長度 > 5 的 cell
    - 排除結構性括號（版本號如 "(v2)"、單字碼如 "(A)"）
    - Severity: "Medium", indicator: "inline_remark"
    - 在 `DetectIssues()` 中呼叫此函式並 append 結果
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5_

  - [x] 1.4 實作 DetectOrphanTotalRows 偵測函式
    - 在 `backend/internal/assessment/issues.go` 新增 `DetectOrphanTotalRows(data *upload.SheetData) []Issue`
    - 偵測條件：(a) 主資料塊後有 2+ 連續空列, (b) 該列至多 2 個非空 cell, (c) 至少一個為 numeric
    - 回傳結果併入 "表格結構問題" issue，sub-label: "孤立合計列"
    - 在 `DetectIssues()` 中呼叫（可整合到既有 TableStructure issue 或獨立 append）
    - _Requirements: 4.1, 4.2, 4.3, 4.4_

  - [x] 1.5 實作 DetectEmptyHeaders 偵測函式
    - 在 `backend/internal/assessment/issues.go` 新增 `DetectEmptyHeaders(data *upload.SheetData) []Issue`
    - 檢查每個 header：null/empty/whitespace-only → 標記
    - Severity: "Low", indicator: "empty_header"
    - Description 列出受影響的欄位位置（如 "第1欄", "第3欄"）
    - 在 `DetectIssues()` 中呼叫此函式並 append 結果
    - _Requirements: 5.1, 5.2, 5.3, 5.4_

- [x] 2. Checkpoint — 偵測函式驗證
  - Ensure all tests pass, ask the user if questions arise.

- [x] 3. 分數整合與 PBT 測試（偵測層）
  - [x] 3.1 修改 indicators.go 整合新偵測結果至分數計算
    - 新增/修改 `CalculateFormatConsistencyWithIssues(data, placeholderCells, mismatchCells int) float64`
    - 新增/修改 `CalculateTableStructureWithIssues(data, orphanTotalDetected, inlineRemarkDense bool) float64`
    - 新增/修改 `CalculateAIQueryReadinessWithIssues(data, emptyHeaderDetected bool) float64`
    - 更新 `CalculateIndicatorScores()` 呼叫端使用新函式
    - 確保 placeholder cell 計為 format mismatch、orphan total 不與 keyword subtotal 重複扣分、empty header 扣 20 分
    - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5_

  - [ ]* 3.2 撰寫 PBT — Property 1: Cell Reference Placeholder Detection Completeness
    - **Property 1: Cell Reference Placeholder Detection Completeness**
    - **Validates: Requirements 1.1, 1.4**
    - 在 `backend/internal/assessment/pbt_interactive_test.go` 撰寫
    - Generator：生成隨機字串（含/不含 `^同[A-Z]+\d+$` 模式）
    - Assert：符合模式的 cell 被 flag，不符合的不被 flag

  - [ ]* 3.3 撰寫 PBT — Property 2: Column Type Inference and Mismatch Flagging
    - **Property 2: Column Type Inference and Mismatch Flagging**
    - **Validates: Requirements 2.1, 2.2, 2.4, 2.5**
    - Generator：生成欄位（numeric ratio 0-100%）
    - Assert：>70% numeric 時 non-numeric non-empty cells 被 flag；≤70% 不 flag；empty 不 flag

  - [ ]* 3.4 撰寫 PBT — Property 3: Column Type Mismatch Severity Threshold
    - **Property 3: Column Type Mismatch Severity Threshold**
    - **Validates: Requirements 2.3**
    - Generator：生成 numeric 欄位（mismatch ratio 圍繞 10% 邊界）
    - Assert：>10% mismatch → "High"，≤10% → "Medium"

  - [ ]* 3.5 撰寫 PBT — Property 4: Inline Remark Detection Precision
    - **Property 4: Inline Remark Detection Precision**
    - **Validates: Requirements 3.1, 3.2, 3.4, 3.5**
    - Generator：生成結構欄位 cell（含各種括號內容）
    - Assert：含中文或長度>5 的括號被 flag，結構性括號不被 flag

  - [ ]* 3.6 撰寫 PBT — Property 5: Orphan Total Row Detection
    - **Property 5: Orphan Total Row Detection**
    - **Validates: Requirements 4.1, 4.4**
    - Generator：生成尾部資料（empty gap + sparse rows）
    - Assert：符合三條件的列被偵測，不符合的不被偵測

  - [ ]* 3.7 撰寫 PBT — Property 6: Empty Header Detection
    - **Property 6: Empty Header Detection**
    - **Validates: Requirements 5.1, 5.4**
    - Generator：生成 header array（含 null/empty/whitespace/valid）
    - Assert：empty/whitespace header 被 flag，non-empty 不被 flag

  - [ ]* 3.8 撰寫 PBT — Property 12: Placeholder Cells Reduce Format Consistency Score
    - **Property 12: Placeholder Cells Reduce Format Consistency Score**
    - **Validates: Requirements 9.1**
    - Generator：生成 numeric 欄位含 placeholder cells
    - Assert：含 placeholder 的分數低於全為 numeric 的分數

  - [ ]* 3.9 撰寫 PBT — Property 13: Empty Header Reduces AI Query Readiness
    - **Property 13: Empty Header Reduces AI Query Readiness**
    - **Validates: Requirements 9.4**
    - Generator：生成含空白 header 的 SheetData
    - Assert：AI Query Readiness 比全 valid header 低 20 分

  - [ ]* 3.10 撰寫 PBT — Property 14: Orphan Total Deduction Non-Duplication
    - **Property 14: Orphan Total Deduction Non-Duplication**
    - **Validates: Requirements 9.3**
    - Generator：生成含/不含 keyword subtotal 的資料 + orphan total
    - Assert：keyword subtotal 存在時不再扣 orphan total 的 -15

- [x] 4. Checkpoint — 偵測層完整驗證
  - Ensure all tests pass, ask the user if questions arise.

- [x] 5. 互動式修正後端
  - [x] 5.1 新增 InteractiveFixRequest/CellEdit/InteractiveFixResponse 結構
    - 在 `backend/internal/cleaning/model.go` 新增三個 struct
    - `InteractiveFixRequest`: AssessmentID (uuid), Edits ([]CellEdit)
    - `CellEdit`: RowIndex, ColIndex, Action (oneof=replace/keep/delete_row/remark_split/header_rename), Value
    - `InteractiveFixResponse`: Success, RowsAffected, Warnings, LogEntries
    - _Requirements: 8.1, 8.2_

  - [x] 5.2 實作 ApplyInteractiveEdits 方法
    - 在 `backend/internal/cleaning/service.go` 新增 `ApplyInteractiveEdits(ctx, userID, req) (*InteractiveFixResponse, error)`
    - 載入 assessment → upload → SheetData
    - 按順序處理：replace → remark_split → header_rename → delete_row（降序索引）
    - replace: 更新指定 cell 值，記錄 LogEntry（operation_type: "cell_edit"）
    - remark_split: 提取括號內容到「備註」欄，記錄 LogEntry（operation_type: "remark_split"）
    - header_rename: 更新 header，記錄 LogEntry（operation_type: "header_rename"）
    - delete_row: 降序刪除，記錄 LogEntry（operation_type: "delete_row"）
    - 超出範圍的 edit 跳過並加入 warnings
    - 儲存修正後資料、更新 cleaning session
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5, 7.6_

  - [x] 5.3 實作 ApplyInteractiveFix handler 端點
    - 在 `backend/internal/cleaning/handler.go` 新增 `ApplyInteractiveFix(c *gin.Context)`
    - 驗證 JWT user_id
    - 綁定 InteractiveFixRequest JSON body
    - 驗證 edits 非空（空則回傳 400）
    - 驗證 action 值合法性、replace/header_rename 必須有 value
    - 呼叫 service.ApplyInteractiveEdits
    - 回傳 InteractiveFixResponse（200）或對應錯誤碼（400/404/500）
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 8.6_

  - [x] 5.4 註冊路由 POST /api/clean/interactive
    - 在 router 設定檔（`cmd/server/main.go` 或路由配置處）註冊新端點
    - 路由需通過 JWT auth middleware
    - _Requirements: 8.1_

- [x] 6. Checkpoint — 後端互動式修正驗證
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 7. 互動式修正 PBT 測試
  - [ ]* 7.1 撰寫 PBT — Property 7: Interactive Edit Application Order Invariant
    - **Property 7: Interactive Edit Application Order Invariant**
    - **Validates: Requirements 7.1**
    - 在 `backend/internal/cleaning/pbt_interactive_test.go` 撰寫
    - Generator：生成含混合 action 的 edit batch（隨機排列）
    - Assert：不論輸入順序，最終狀態等同 canonical order（replace → split → rename → delete desc）

  - [ ]* 7.2 撰寫 PBT — Property 8: Cell Replacement Round-Trip
    - **Property 8: Cell Replacement Round-Trip**
    - **Validates: Requirements 7.2**
    - Generator：生成有效 (row_index, col_index, value) tuple
    - Assert：apply 後 cell 值等於 replacement value，log 含正確 entry

  - [ ]* 7.3 撰寫 PBT — Property 9: Remark Split Preservation
    - **Property 9: Remark Split Preservation**
    - **Validates: Requirements 7.3**
    - Generator：生成含括號的 cell 值（中文/長度>5）
    - Assert：split 後 structural value + 括號 + remark = 原始值；備註欄存在且正確

  - [ ]* 7.4 撰寫 PBT — Property 10: Row Deletion Index Safety
    - **Property 10: Row Deletion Index Safety**
    - **Validates: Requirements 7.5**
    - Generator：生成多筆 delete 指令（含重複索引）
    - Assert：降序刪除後 row count = original - distinct valid deletions；剩餘 rows 內容不變

  - [ ]* 7.5 撰寫 PBT — Property 11: Invalid Edit Graceful Handling
    - **Property 11: Invalid Edit Graceful Handling**
    - **Validates: Requirements 7.6**
    - Generator：生成含 out-of-bounds 的 edit batch
    - Assert：無效 edits 被跳過，有效 edits 正確 apply，warnings 列出所有 skipped

- [x] 8. Checkpoint — 後端完整驗證
  - Ensure all tests pass, ask the user if questions arise.

- [x] 9. 前端 CellEditor 元件
  - [x] 9.1 新增 API client 方法 — interactive fix
    - 在 `frontend/src/api/client.ts` 新增 `submitInteractiveEdits(req: InteractiveFixRequest): Promise<InteractiveFixResponse>`
    - 定義 TypeScript interfaces：`FlaggedCell`, `CellEditAction`, `InteractiveFixRequest`, `InteractiveFixResponse`
    - _Requirements: 6.6, 8.1_

  - [x] 9.2 實作 CellEditor 元件
    - 在 `frontend/src/components/CellEditor.tsx` 建立元件
    - Props: assessmentId, flaggedCells, onComplete
    - 按 issue_type 分群顯示 flagged cells
    - 每個 cell 顯示：欄位名稱、列號、目前值、issue type 描述
    - 提供 action 選項："輸入新值"、"保留原值"、"刪除該列"
    - inline_remark 額外顯示「分離備註」按鈕
    - empty_header 顯示 inline text input
    - 確認按鈕觸發批次提交
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 6.6_

  - [x] 9.3 整合 CellEditor 至 CleaningPage
    - 修改 `frontend/src/pages/CleaningPage.tsx`
    - 偵測到互動式修正問題時（cell_reference_placeholder, column_type_mismatch, inline_remark, empty_header），render CellEditor
    - 將 assessment issues 轉換為 FlaggedCell[] 格式
    - 處理 onComplete callback：刷新頁面狀態或顯示成功訊息
    - _Requirements: 6.1, 6.6_

- [x] 10. Final checkpoint — 完整驗證
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties from the design document
- Unit tests validate specific examples and edge cases
- 本功能使用 `pgregory.net/rapid` 進行 property-based testing（專案既有依賴）
- 偵測函式遵循現有 `issues.go` 中的模式：接收 `*upload.SheetData`，回傳 `[]Issue`
- 互動式修正與現有 `ApplyRules` 平行運作，共用 `LogEntry` 審計機制
- 前端 CellEditor 嵌入 CleaningPage（Step 5），位於批次規則之後

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1", "1.2", "1.3", "1.4", "1.5"] },
    { "id": 1, "tasks": ["3.1", "5.1"] },
    { "id": 2, "tasks": ["3.2", "3.3", "3.4", "3.5", "3.6", "3.7", "5.2"] },
    { "id": 3, "tasks": ["3.8", "3.9", "3.10", "5.3"] },
    { "id": 4, "tasks": ["5.4", "7.1", "7.2", "7.3", "7.4", "7.5"] },
    { "id": 5, "tasks": ["9.1"] },
    { "id": 6, "tasks": ["9.2"] },
    { "id": 7, "tasks": ["9.3"] }
  ]
}
```
