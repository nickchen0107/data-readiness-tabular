# Requirements Document

## Introduction

將匯出/產出頁面（ExportPage）重新設計為「比較儀表板」，以視覺化方式呈現梳理前後的品質變化。儀表板包含總分改善顯示、六項指標進度條（含改善延伸色段）、雙層雷達圖、已解決與尚待解決問題列表，以及原有的下載功能。整體版面風格與評估頁面（AssessmentPage）保持一致。

## Glossary

- **Comparison_Dashboard**: 比較儀表板，重新設計後的匯出頁面，同時呈現梳理前後評估數據的對比視圖
- **Original_Assessment**: 梳理前的品質評估結果，包含六項指標分數、總分、問題列表
- **Post_Cleaning_Assessment**: 梳理後的品質評估結果，由後端對清理後資料重新執行評估產生
- **Indicator_Progress_Bar**: 指標進度條，以水平條狀圖呈現單項指標分數，使用雙色段（原始分數 + 改善增量）
- **Dual_Layer_Radar_Chart**: 雙層雷達圖，在同一張雷達圖上以兩個不同顏色的多邊形分別呈現梳理前後的六項指標
- **Resolved_Issues**: 已修正的問題，指梳理過程中成功解決的資料品質問題
- **Remaining_Issues**: 尚待解決的問題，指梳理後仍存在的資料品質問題
- **Improvement_Delta**: 改善增量，梳理後分數與梳理前分數的差值
- **Score_Display_Format**: 分數顯示格式，以「原始分數 (+改善增量)」的形式呈現，例如 "28.8 (+55.6)"
- **CleaningSession**: 梳理記錄，後端儲存的清理操作完整資訊，包含 score_before、score_after、cleaning_log 等

## Requirements

### Requirement 1: 比較儀表板資料載入

**User Story:** 作為使用者，我希望在匯出頁面自動載入梳理前後的完整評估資料，以便檢視品質改善成果。

#### Acceptance Criteria

1. WHEN the Comparison_Dashboard is loaded, THE Dashboard SHALL fetch the latest CleaningSession data including score_before, score_after, rows_before, and rows_after
2. WHEN the Comparison_Dashboard is loaded, THE Dashboard SHALL fetch the Original_Assessment record containing six indicator scores and the issues list
3. WHEN the Comparison_Dashboard is loaded, THE Dashboard SHALL fetch the Post_Cleaning_Assessment record containing six indicator scores and the issues list
4. IF the CleaningSession data is unavailable, THEN THE Dashboard SHALL display an error message and provide a navigation link back to the cleaning step
5. WHILE the data is loading, THE Dashboard SHALL display a loading skeleton matching the dashboard layout structure

### Requirement 2: 總分改善顯示

**User Story:** 作為使用者，我希望一眼看到總分以及改善幅度，以便快速掌握梳理的整體效果。

#### Acceptance Criteria

1. THE Comparison_Dashboard SHALL display the total score using the Score_Display_Format showing the original score followed by the Improvement_Delta in parentheses (e.g., "28.8 (+55.6)")
2. WHEN the Improvement_Delta is positive, THE Dashboard SHALL display the delta value in green color
3. WHEN the Improvement_Delta is zero, THE Dashboard SHALL display "(+0.0)" in neutral gray color
4. THE Comparison_Dashboard SHALL display the post-cleaning total score (score_after) as the primary prominent number alongside the Score_Display_Format
5. THE Comparison_Dashboard SHALL display the status grade (ready/conditional/not_ready) based on the Post_Cleaning_Assessment result

### Requirement 3: 六項指標進度條

**User Story:** 作為使用者，我希望透過進度條清楚看到每項指標的原始分數與改善幅度，以便了解哪些指標改善最多。

#### Acceptance Criteria

1. THE Comparison_Dashboard SHALL display six Indicator_Progress_Bar components for: 列完整度、欄完整度、格式一致性、資料唯一性、表格結構、AI 問答可用性
2. THE Indicator_Progress_Bar SHALL render the Original_Assessment score as a base color segment starting from the left edge
3. THE Indicator_Progress_Bar SHALL render the Improvement_Delta as a visually distinct second color segment extending from the end of the base segment
4. THE Indicator_Progress_Bar SHALL use a different hue or lighter shade for the improvement segment to distinguish it from the original score segment
5. THE Indicator_Progress_Bar SHALL display the numeric original score and the improvement delta as text labels (e.g., "45.2 → 78.6 (+33.4)")
6. WHEN the Improvement_Delta for an indicator is zero, THE Indicator_Progress_Bar SHALL display only the original score segment without an improvement extension
7. THE Indicator_Progress_Bar SHALL use a 0-100 scale with the total bar width representing 100 points

### Requirement 4: 雙層雷達圖

**User Story:** 作為使用者，我希望在雷達圖上同時看到梳理前後的六項指標分佈，以便一目了然比較整體形狀的變化。

#### Acceptance Criteria

1. THE Dual_Layer_Radar_Chart SHALL render two polygon layers on the same radar chart using the six indicator axes
2. THE Dual_Layer_Radar_Chart SHALL render the Original_Assessment layer using a semi-transparent fill with a distinct border color (e.g., gray or light blue)
3. THE Dual_Layer_Radar_Chart SHALL render the Post_Cleaning_Assessment layer using a semi-transparent fill with a different distinct border color (e.g., green or accent color)
4. THE Dual_Layer_Radar_Chart SHALL display a legend identifying which layer represents "梳理前" (before) and which represents "梳理後" (after)
5. THE Dual_Layer_Radar_Chart SHALL label each of the six axes with the indicator name in Traditional Chinese
6. THE Dual_Layer_Radar_Chart SHALL use a 0-100 scale for the radial axis

### Requirement 5: 問題解決狀態列表

**User Story:** 作為使用者，我希望清楚看到哪些問題已修正、哪些仍待解決，以便了解資料目前的品質狀態。

#### Acceptance Criteria

1. THE Comparison_Dashboard SHALL display a "已修正的問題" section listing all Resolved_Issues
2. THE Comparison_Dashboard SHALL display a "尚待解決的問題" section listing all Remaining_Issues
3. THE Comparison_Dashboard SHALL determine Resolved_Issues by comparing the Original_Assessment issues list against the Post_Cleaning_Assessment issues list, identifying issues present in the original but absent in the post-cleaning assessment
4. THE Comparison_Dashboard SHALL determine Remaining_Issues as all issues present in the Post_Cleaning_Assessment issues list
5. WHEN the Resolved_Issues list is empty, THE Dashboard SHALL display a message indicating no issues were resolved
6. WHEN the Remaining_Issues list is empty, THE Dashboard SHALL display a message indicating all issues have been resolved
7. THE issue list items SHALL display the issue title, severity badge, and affected row count consistent with the AssessmentPage issue card style

### Requirement 6: 版面風格一致性

**User Story:** 作為使用者，我希望比較儀表板的視覺設計與評估頁面風格統一，以便獲得一致的使用體驗。

#### Acceptance Criteria

1. THE Comparison_Dashboard SHALL use the same card container style as AssessmentPage including border-radius of 14px, paper background, and line border
2. THE Comparison_Dashboard SHALL use the same step header pattern showing "STEP 5" label, page title, and descriptive subtitle
3. THE Comparison_Dashboard SHALL use consistent font sizes, spacing values, and color variables (--accent, --green, --ink-soft, --ink-faint) as defined in the AssessmentPage
4. THE Comparison_Dashboard SHALL arrange the score display and radar chart in the upper section, indicator progress bars in the middle section, and issue lists in the lower section
5. THE Comparison_Dashboard SHALL be responsive and maintain readability on viewport widths between 800px and 1440px

### Requirement 7: 下載功能保留

**User Story:** 作為使用者，我希望仍能從比較儀表板下載梳理後資料、品質報告及操作紀錄，以便進行後續作業。

#### Acceptance Criteria

1. THE Comparison_Dashboard SHALL display download buttons for: 梳理後資料 (refined.xlsx), 品質報告 (report.pdf), 梳理紀錄 (cleaning.log)
2. WHEN a download button is clicked, THE Dashboard SHALL initiate the file download from the existing export API endpoint (/api/export/:id/:type)
3. WHILE a download is in progress, THE Dashboard SHALL disable the corresponding download button and display a loading indicator
4. IF a download fails, THEN THE Dashboard SHALL display an error notification without disrupting the dashboard view
5. THE download section SHALL be placed in the lower portion of the dashboard below the issue lists

### Requirement 8: 後端比較資料 API

**User Story:** 作為前端開發者，我希望有一個整合的 API 端點提供梳理前後的完整比較資料，以便前端一次取得所有需要的數據。

#### Acceptance Criteria

1. THE Backend SHALL provide a comparison data API endpoint that returns the Original_Assessment and Post_Cleaning_Assessment together in a single response
2. THE comparison API response SHALL include both assessments' six indicator scores, total scores, status grades, and issues lists
3. THE comparison API response SHALL include the CleaningSession metadata: rows_before, rows_after, score_before, score_after, rules_applied, and cleaning_log summary
4. WHEN the requested CleaningSession does not exist, THE Backend SHALL return HTTP 404 with a descriptive error message
5. THE comparison API endpoint SHALL verify user ownership of the CleaningSession before returning data
