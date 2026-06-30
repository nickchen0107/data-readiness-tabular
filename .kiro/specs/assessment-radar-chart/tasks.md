# Implementation Plan: Assessment Radar Chart

## Overview

為 AssessmentPage 評估指標區段新增雷達圖、指標資訊 Tooltip、以及指標重新命名。所有變更集中在 `frontend/src/pages/AssessmentPage.tsx` 單一檔案，另將 tooltip 定位純函式抽出至獨立檔案以便測試。不需安裝新套件（recharts 已安裝），不需修改後端。

## Tasks

- [ ] 1. 重新命名指標與建立雷達圖佈局
  - [ ] 1.1 重新命名「重複/近似」指標並建立 flex 佈局
    - 在 `frontend/src/pages/AssessmentPage.tsx` 中修改 `indicators` 陣列：將 `{ name: '重複/近似', nameEn: 'Duplicate/Similar' }` 改為 `{ name: '資料唯一性', nameEn: 'Data Uniqueness' }`
    - 將現有指標進度條區塊包裹為 `display: flex` 佈局：左側為進度條（flex: 1），右側為雷達圖容器（固定寬度 280px, flexShrink: 0）
    - 確認 score 值仍綁定 `assessment.duplicate_similar` 欄位，color 保持 `var(--amber)`
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 1.6_

  - [ ] 1.2 實作 RadarChart 元件渲染
    - 在右側容器中加入 recharts 元件：`ResponsiveContainer`（width="100%", height={260}）包裹 `RadarChart`（cx="50%", cy="50%", outerRadius="80%"）
    - 加入 `PolarGrid`、`PolarAngleAxis`（dataKey="subject" 顯示中文名）、`PolarRadiusAxis`（domain=[0,100], tick={false}）
    - 加入 `Radar`（dataKey="score", fill="var(--accent)", fillOpacity={0.35}, stroke="var(--accent)", strokeWidth={2}）
    - 建立 `radarData` 衍生陣列：`indicators.map(ind => ({ subject: ind.name, score: ind.score }))`
    - import 必要的 recharts 元件：`RadarChart, Radar, PolarGrid, PolarAngleAxis, PolarRadiusAxis, ResponsiveContainer`
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 1.7_

- [ ] 2. 實作指標 Info Tooltip
  - [ ] 2.1 新增 tooltip 狀態與內容映射
    - 新增 `useState<string | null>(null)` 管理 `hoveredIndicator` 狀態
    - 新增 `indicatorInfo` 物件（Record<string, { desc: string; calc: string }>），包含六項指標的說明與計算方式文字（依據 Requirements 2.7 定義的內容）
    - _Requirements: 2.4, 2.5, 2.7_

  - [ ] 2.2 實作 ⓘ 圖示與 tooltip 彈出框
    - 在每個 indicator bar row 的中文名稱後方加入 ⓘ `<span>`，設定 `onMouseEnter` / `onMouseLeave` 事件切換 `hoveredIndicator`
    - 每個 row 容器設為 `position: relative`
    - 當 `hoveredIndicator === ind.name` 時渲染 tooltip `<div>`（position: absolute, 含 desc 與 calc 兩行內容）
    - tooltip 樣式：background var(--panel), border 1px solid var(--line), borderRadius 8, padding 10px 14px, fontSize 12, boxShadow, zIndex 10, minWidth 220, maxWidth 300
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6_

  - [ ] 2.3 抽取 computeTooltipPosition 純函式
    - 建立 `frontend/src/pages/computeTooltipPosition.ts` 匯出純函式
    - 函式簽名：`computeTooltipPosition(iconRect: { top: number; left: number; width: number; height: number }, tooltipSize: { width: number; height: number }, viewport: { width: number; height: number }): { top: number; left: number }`
    - 實作 viewport 邊界檢查邏輯：右溢出翻轉至左側、底部溢出上移、頂部溢出 clamp 至 8px、左溢出 clamp 至 8px
    - 在 AssessmentPage.tsx 中 import 並可選用於動態定位
    - _Requirements: 2.6_

- [ ] 3. Checkpoint — 功能完整驗證
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 4. 測試
  - [ ]* 4.1 撰寫 Property-Based Test — Property 1: Tooltip viewport containment
    - **Property 1: Tooltip viewport containment**
    - **Validates: Requirements 2.6**
    - 在 `frontend/src/pages/computeTooltipPosition.pbt.test.ts` 撰寫
    - 使用 fast-check 生成隨機 iconRect（top/left/width/height 皆為正數且在合理 viewport 範圍內）、tooltipSize（width/height 正數）、viewport（width/height 正數且 ≥ tooltipSize）
    - Assert：回傳的 top ≥ 0, left ≥ 0, top + tooltipSize.height ≤ viewport.height, left + tooltipSize.width ≤ viewport.width
    - 最少 100 iterations
    - Tag: `Feature: assessment-radar-chart, Property 1: Tooltip viewport containment`

  - [ ]* 4.2 撰寫 Unit Test — 雷達圖與 tooltip 渲染
    - **Validates: Requirements 1.1, 1.3, 2.1, 3.1, 3.3, 4.1**
    - 在 `frontend/src/pages/AssessmentPage.test.tsx` 撰寫
    - 測試案例：
      - 渲染後存在 RadarChart 相關 SVG 元素
      - 軸標籤包含六項中文指標名（含「資料唯一性」，不含「重複/近似」）
      - 每個 indicator row 含 ⓘ 圖示
      - hover ⓘ 後 tooltip 出現正確內容
      - mouseLeave 後 tooltip 消失
      - 六個進度條仍正常渲染

- [ ] 5. Final checkpoint — 全部測試通過
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties from the design document
- 所有變更集中在 `frontend/src/pages/AssessmentPage.tsx`，tooltip 純函式抽至 `frontend/src/pages/computeTooltipPosition.ts` 以便測試
- recharts 已安裝於專案中，無需新增依賴
- 測試基礎設施已就緒（vitest + @testing-library/react + fast-check）
- 本專案在 Docker 中運行，不需全域安裝任何套件

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1"] },
    { "id": 1, "tasks": ["1.2", "2.1"] },
    { "id": 2, "tasks": ["2.2", "2.3"] },
    { "id": 3, "tasks": ["4.1", "4.2"] }
  ]
}
```
