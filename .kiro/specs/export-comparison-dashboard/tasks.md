# Implementation Plan: Export Comparison Dashboard

## Overview

將匯出頁面（ExportPage）重新設計為比較儀表板，包含後端比較 API（`GET /api/compare/:session_id`）和前端可視化元件。實作順序：後端 comparison 套件 → 前端工具函式 → 前端元件重寫。

## Tasks

- [x] 1. 建立後端 comparison 套件（模型與服務層）
  - [x] 1.1 建立 `internal/comparison/model.go` — 定義 ComparisonResponse、SessionSummary、AssessmentSummary 結構體
    - 定義 `ComparisonResponse` struct 包含 Session、OriginalAssess、PostCleanAssess 三個欄位
    - 定義 `SessionSummary` struct 包含 ID、RowsBefore、RowsAfter、ScoreBefore、ScoreAfter、RulesApplied、CleaningLog、CreatedAt
    - 定義 `AssessmentSummary` struct 包含 ID、TotalScore、Status、六項指標分數、Issues、RowDistribution
    - JSON tag 使用 snake_case 與設計文件一致
    - _Requirements: 8.1, 8.2, 8.3_

  - [x] 1.2 建立 `internal/comparison/service.go` — 實作 GetComparison 業務邏輯
    - 建立 `Service` struct 持有 `*cleaning.Repository` 和 `*assessment.Repository` 以及 `*assessment.Service`
    - 實作 `NewService` 建構函式
    - 實作 `GetComparison(ctx, sessionID, userID)` 方法：
      - 呼叫 `cleanRepo.GetByIDAndUser` 取得 session（含 ownership 驗證）
      - 呼叫 `assessRepo.GetByID(session.AssessmentID)` 取得原始評估
      - 使用 assessment.Service 對 `session.RefinedFilePath` 重新執行評估以取得梳理後分數
      - 組裝 `ComparisonResponse` 並回傳
    - 若 session 不存在或非該使用者擁有，回傳 `cleaning.ErrSessionNotFound`
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5_

  - [x] 1.3 建立 `internal/comparison/handler.go` — 實作 HTTP 端點 `GET /api/compare/:session_id`
    - 建立 `Handler` struct 持有 `*Service`
    - 實作 `NewHandler` 建構函式
    - 實作 `GetComparison(c *gin.Context)` 方法：
      - 從 URL 取得 session_id 參數並解析 UUID
      - 從 JWT context 取得 user_id
      - 呼叫 `service.GetComparison`
      - 成功回傳 HTTP 200 + JSON ComparisonResponse
      - Session 不存在回傳 HTTP 404 `{"error": {"code": "NOT_FOUND", "message": "梳理記錄不存在"}}`
      - 內部錯誤回傳 HTTP 500
    - _Requirements: 8.1, 8.4, 8.5_

  - [x] 1.4 在 `cmd/server/main.go` 註冊 comparison 路由
    - import `internal/comparison` 套件
    - 建立 `comparisonSvc` 傳入 cleanRepo、assessRepo、assessSvc
    - 建立 `comparisonHandler`
    - 在 protected group 註冊 `GET /compare/:id` 路由
    - _Requirements: 8.1_

- [x] 2. Checkpoint — 後端 comparison API 驗證
  - 確認 `go build ./...` 編譯通過
  - 確認 comparison 套件的結構與 handler 連接正確
  - Ensure all tests pass, ask the user if questions arise.

- [x] 3. 建立前端工具函式
  - [x] 3.1 建立 `src/utils/scoreFormat.ts` — 分數格式化純函式
    - 實作 `formatScoreWithDelta(before: number, after: number): string`：回傳 `"X.X (+Y.Y)"` 格式
    - 實作 `formatIndicatorLabel(before: number, after: number): string`：回傳 `"X.X → Y.Y (+Z.Z)"` 格式
    - 實作 `getDeltaColor(delta: number): string`：正數回傳 green，零或負回傳 gray
    - 所有數值取小數點後一位
    - _Requirements: 2.1, 2.2, 2.3, 3.5_

  - [ ]* 3.2 撰寫 Property Test — scoreFormat 函式 (Property 1: Score display format correctness)
    - **Property 1: Score display format correctness**
    - **Validates: Requirements 2.1**
    - 在 `src/utils/scoreFormat.test.ts` 使用 fast-check
    - 生成任意 (before: 0–100, after: 0–100) 數對
    - 驗證 `formatScoreWithDelta` 輸出匹配 `/^\d+\.\d \(\+\-?\d+\.\d\)$/`
    - 驗證數值正確：解析輸出中的數字並比對輸入

  - [ ]* 3.3 撰寫 Property Test — formatIndicatorLabel 函式 (Property 3: Indicator label format correctness)
    - **Property 3: Indicator label format correctness**
    - **Validates: Requirements 3.5**
    - 在 `src/utils/scoreFormat.test.ts` 使用 fast-check
    - 生成任意 (before: 0–100, after: 0–100) 數對
    - 驗證 `formatIndicatorLabel` 輸出匹配 `"X.X → Y.Y (+Z.Z)"` 格式
    - 驗證 Z.Z = Y.Y - X.X（容忍浮點誤差 0.1 以內）

  - [x] 3.4 建立 `src/utils/issueDiff.ts` — 問題差異計算純函式
    - 定義 `Issue` interface：`{ title: string; severity: string; affected_rows: number }`
    - 實作 `getResolvedIssues(original: Issue[], postCleaning: Issue[]): Issue[]`：回傳 original 中 title 不在 postCleaning 中的問題
    - 實作 `getRemainingIssues(postCleaning: Issue[]): Issue[]`：直接回傳 postCleaning 列表
    - _Requirements: 5.3, 5.4_

  - [ ]* 3.5 撰寫 Property Test — issueDiff 函式 (Property 4: Resolved issues set difference)
    - **Property 4: Resolved issues set difference**
    - **Validates: Requirements 5.3**
    - 在 `src/utils/issueDiff.test.ts` 使用 fast-check
    - 生成任意兩個 Issue[] 列表
    - 驗證 `getResolvedIssues(original, post)` 中每個 issue 的 title 都存在於 original 但不存在於 post
    - 驗證 resolved + remaining 的 title 集合覆蓋 original 的所有 title

- [x] 4. Checkpoint — 前端工具函式驗證
  - 執行 `npx vitest --run src/utils/` 確認所有測試通過
  - 確認 TypeScript 編譯無錯誤
  - Ensure all tests pass, ask the user if questions arise.

- [x] 5. 重寫前端 ExportPage 為 ComparisonDashboard
  - [x] 5.1 重寫 `src/pages/ExportPage.tsx` — 主要資料取得與架構
    - 保留檔案位置 `src/pages/ExportPage.tsx`（路由不變）
    - 定義 `ComparisonData` TypeScript interface 對應後端 ComparisonResponse
    - 使用 useEffect + apiClient 呼叫 `GET /api/compare/${sessionId}`（sessionId 從 `/clean/latest` 取得或使用 URL params）
    - 實作 loading 狀態：顯示 skeleton 佔位元件（匹配儀表板版面結構）
    - 實作 error 狀態：顯示錯誤卡片 + 返回清理步驟的連結
    - 保留 STEP 5 header 風格（mono 字體、accent 色、uppercase label）
    - 整體卡片容器使用 `borderRadius: 14px`、`background: var(--paper)`、`border: 1px solid var(--line)`
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 6.1, 6.2, 6.3_

  - [x] 5.2 實作總分改善顯示區塊
    - 顯示 `score_after` 作為主要大數字
    - 使用 `formatScoreWithDelta(score_before, score_after)` 顯示改善格式
    - delta 正值顯示綠色（`var(--green)`），零值顯示灰色（`var(--ink-faint)`）
    - 顯示 post-cleaning 狀態等級 badge（ready/conditional/not_ready）
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5_

  - [x] 5.3 實作六項指標進度條（IndicatorProgressBar 內聯元件）
    - 建立 `IndicatorProgressBar` 元件，props：`{ label: string; before: number; after: number }`
    - 渲染基底色段：寬度 = `before%`，顏色 `var(--accent)`
    - 渲染改善色段：寬度 = `(after - before)%`，起始於 `before%`，使用較淺色調
    - delta 為零時不渲染改善色段
    - 使用 `formatIndicatorLabel` 顯示文字標籤
    - 進度條總寬度代表 0-100 分
    - 六項指標按順序渲染：列完整度、欄完整度、格式一致性、資料唯一性、表格結構、AI 問答可用性
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 3.7_

  - [ ]* 5.4 撰寫 Property Test — IndicatorProgressBar 色段定位 (Property 2)
    - **Property 2: Indicator progress bar segment positioning**
    - **Validates: Requirements 3.2, 3.3, 3.7**
    - 在 `src/pages/ExportPage.test.ts` 使用 fast-check
    - 生成任意 (before: 0–100, after: before–100) 數對
    - 驗證 base segment width = before, improvement segment width = after - before
    - 驗證 base + improvement <= 100

  - [x] 5.5 實作雙層雷達圖（DualRadarChart 內聯元件）
    - 使用 recharts `RadarChart`、`PolarGrid`、`PolarAngleAxis`、`PolarRadiusAxis`
    - 兩個 `Radar` 層：梳理前（stroke `#94a3b8`、fill opacity 0.2）、梳理後（stroke `var(--green)`、fill opacity 0.3）
    - 使用 `Legend` 元件顯示圖例（「梳理前」/「梳理後」）
    - 六軸標籤使用繁體中文指標名稱
    - radialAxis domain 設定為 [0, 100]
    - 資料結構：`[{ name: "列完整度", before: X, after: Y }, ...]`
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5, 4.6_

  - [x] 5.6 實作問題解決狀態列表（IssueList 內聯元件）
    - 使用 `getResolvedIssues` 計算已修正問題列表
    - 使用 `getRemainingIssues` 取得尚待解決問題列表
    - 渲染「已修正的問題」區塊：列出每個 issue 的 title、severity badge、affected_rows
    - 渲染「尚待解決的問題」區塊：同上格式
    - 空列表時顯示對應的空狀態訊息
    - 風格與 AssessmentPage issue card 一致
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7_

  - [x] 5.7 實作下載功能區塊（DownloadSection 內聯元件）
    - 保留原有三個下載按鈕：梳理後資料（xlsx）、品質報告（pdf）、梳理紀錄（log）
    - 呼叫既有 `/api/export/:id/:type` 端點下載
    - 下載中禁用按鈕並顯示 loading 指示器
    - 下載失敗顯示 toast 通知，不影響儀表板其餘部分
    - 放置於問題列表下方
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5_

  - [x] 5.8 實作版面配置與響應式設計
    - 上方區塊：總分顯示 + 雷達圖（並排或堆疊）
    - 中間區塊：六項指標進度條
    - 下方區塊：問題列表 + 下載按鈕
    - 使用一致的 CSS 變數：`--accent`、`--green`、`--ink-soft`、`--ink-faint`
    - 確保 800px–1440px viewport 寬度下可讀性
    - 保留 footer 導航至下一步（存證作業）
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5_

- [x] 6. Checkpoint — 前端整合驗證
  - 確認 TypeScript 編譯無錯誤（`npx tsc --noEmit`）
  - 確認無 unused imports/variables（strict mode）
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 7. 後端 Property Tests（Go rapid）
  - [ ]* 7.1 撰寫 Property Test — Comparison API response completeness (Property 5)
    - **Property 5: Comparison API response completeness**
    - **Validates: Requirements 8.1, 8.2, 8.3**
    - 在 `internal/comparison/comparison_pbt_test.go` 使用 `pgregory.net/rapid`
    - 生成隨機 CleaningSession + Assessment 資料
    - 驗證 ComparisonResponse 包含所有必要欄位（六項指標、兩個 total_score、兩個 status、兩個 issues、session metadata）

  - [ ]* 7.2 撰寫 Property Test — User ownership authorization (Property 6)
    - **Property 6: User ownership authorization**
    - **Validates: Requirements 8.5**
    - 在 `internal/comparison/comparison_pbt_test.go` 使用 `pgregory.net/rapid`
    - 生成隨機 session（user_id = A）+ 請求者 user_id = B（A ≠ B）
    - 驗證 GetComparison 回傳 ErrSessionNotFound

- [x] 8. Final Checkpoint — 全部測試通過
  - 後端：`go build ./...` 編譯通過
  - 後端：`go test ./internal/comparison/ -v` 全部通過
  - 前端：`npx tsc --noEmit` 無錯誤
  - 前端：`npx vitest --run` 全部通過
  - Docker 重建：`docker compose up -d --build backend frontend`
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties from the design document
- Unit tests validate specific examples and edge cases
- 後端使用 `pgregory.net/rapid` 進行 property-based testing，前端使用 `fast-check`
- ExportPage.tsx 為重寫（非新建），路由維持不變
- 比較 API 需對 refined_file_path 重新執行評估以取得梳理後分數（非讀取儲存值）
- recharts Legend 元件已可用，無需安裝新套件
- Docker 容器需在變更後重建：`docker compose up -d --build backend frontend`

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1"] },
    { "id": 1, "tasks": ["1.2", "1.3"] },
    { "id": 2, "tasks": ["1.4", "3.1", "3.4"] },
    { "id": 3, "tasks": ["3.2", "3.3", "3.5"] },
    { "id": 4, "tasks": ["5.1"] },
    { "id": 5, "tasks": ["5.2", "5.3", "5.5"] },
    { "id": 6, "tasks": ["5.4", "5.6", "5.7"] },
    { "id": 7, "tasks": ["5.8"] },
    { "id": 8, "tasks": ["7.1", "7.2"] }
  ]
}
```
