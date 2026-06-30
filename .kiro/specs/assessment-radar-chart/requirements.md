# Requirements Document

## Introduction

本功能為 SAFE-AI Excel 梳理小工具的評估指標區段進行三項 UI/UX 增強：(1) 新增雷達圖以視覺化呈現六項指標的整體輪廓；(2) 每項指標標題旁新增資訊圖示，懸停顯示說明與計算方式；(3) 將「重複/近似」指標重新命名為語意更直觀的名稱，使高分明確代表正面意義。此為純前端變更，無需修改後端。

## Glossary

- **Assessment_Page**: 評估結果頁面，顯示上傳檔案的品質分析結果
- **Radar_Chart**: 雷達圖（蜘蛛圖），使用 recharts 的 RadarChart 元件繪製六軸多邊形圖表
- **Indicator_Panel**: 指標面板，顯示六項品質指標的分數與視覺化
- **Info_Tooltip**: 資訊工具提示，滑鼠懸停於 ⓘ 圖示時顯示的浮動說明框
- **Indicator**: 品質指標，包含中文名、英文名、分數（0-100）及顏色的資料結構
- **Uniqueness_Indicator**: 資料唯一性指標，原名「重複/近似」，衡量資料中重複或近似列的比例，高分代表資料唯一性佳

## Requirements

### Requirement 1: Radar Chart Display

**User Story:** As a data analyst, I want to see a radar chart of all six indicators, so that I can quickly grasp the overall quality profile of my data at a glance.

#### Acceptance Criteria

1. WHEN the Assessment_Page renders indicator scores, THE Indicator_Panel SHALL display a Radar_Chart with six axes corresponding to the six quality indicators
2. THE Radar_Chart SHALL use the recharts RadarChart component with PolarGrid, PolarAngleAxis, and PolarRadiusAxis sub-components
3. THE Radar_Chart SHALL display each axis label using the Chinese indicator name
4. THE Radar_Chart SHALL render the score polygon with a filled area using a semi-transparent accent color
5. THE Radar_Chart SHALL set the radial axis scale from 0 to 100
6. THE Radar_Chart SHALL be positioned to the RIGHT of the progress bar list in a side-by-side (flex row) layout, with indicator bars on the left and the Radar_Chart on the right
7. WHILE indicator scores are all zero, THE Radar_Chart SHALL render an empty polygon at the center point

### Requirement 2: Indicator Info Icon with Tooltip

**User Story:** As a user unfamiliar with data quality metrics, I want to see an explanation of each indicator when I hover over an info icon, so that I can understand what each score means and how it is calculated.

#### Acceptance Criteria

1. THE Indicator_Panel SHALL display an ⓘ icon adjacent to each indicator name
2. WHEN the user hovers over an ⓘ icon, THE Info_Tooltip SHALL appear within 100ms showing two sections: a brief description and a simplified calculation explanation
3. WHEN the user moves the cursor away from the ⓘ icon, THE Info_Tooltip SHALL disappear within 100ms
4. THE Info_Tooltip SHALL contain a description line explaining what the indicator measures in plain language
5. THE Info_Tooltip SHALL contain a calculation line explaining how the score is derived in simplified terms
6. THE Info_Tooltip SHALL position itself to avoid overflow beyond the viewport boundaries
7. THE Info_Tooltip SHALL use the following content for each indicator:
   - 列完整度: 衡量每列資料的填寫比例；計算方式為每列非空格數÷總欄數的平均值×100
   - 欄完整度: 衡量每欄資料的填寫比例；計算方式為每欄非空值數÷總列數的平均值×100
   - 格式一致性: 衡量每欄內資料格式的統一程度；計算方式為每欄主要格式佔比的平均值×100
   - 資料唯一性: 衡量資料中重複或近似列的稀少程度；計算方式為 (1 − 重複列比例) × 100
   - 表格結構: 衡量表格結構是否乾淨規整；依據合併儲存格、多層表頭、小計列等問題各扣分
   - AI 問答可用性: 衡量資料是否適合 AI 查詢分析；依據是否含 ID 欄、時間欄、分類欄、數值欄、欄名品質各加 20 分

### Requirement 3: Rename Duplicate/Similar Indicator

**User Story:** As a user, I want the indicator name to clearly convey that a high score is positive, so that I do not misinterpret a high "Duplicate/Similar" score as meaning many duplicates.

#### Acceptance Criteria

1. THE Indicator_Panel SHALL display the indicator previously named "重複/近似" as "資料唯一性" with English name "Data Uniqueness"
2. THE Radar_Chart SHALL use "資料唯一性" as the axis label for the uniqueness indicator
3. THE Assessment_Page SHALL apply the rename consistently across all locations where the indicator name appears in the frontend
4. THE renamed indicator SHALL retain the same score value, color assignment, and data binding to the backend field `duplicate_similar`

### Requirement 4: Preserve Existing Bar Display

**User Story:** As a user, I want to retain the detailed per-indicator progress bar with numeric scores, so that I can still see precise values alongside the radar chart overview.

#### Acceptance Criteria

1. THE Indicator_Panel SHALL continue to display the existing progress bar row for each indicator below or alongside the Radar_Chart
2. THE progress bar display SHALL show the indicator name, English sub-label, colored bar proportional to score, and numeric score out of 100
3. WHEN any indicator score changes, THE progress bar and Radar_Chart SHALL both reflect the updated value consistently
