# Requirements Document

## Introduction

本功能為 SAFE-AI Excel 梳理小工具新增三大模組：(1) 管理者角色與管理後台、(2) 使用者評估配額管理與流程狀態保存、(3) 多語系（i18n）支援含管理者翻譯編輯器。透過角色區分與配額限制，實現使用量控管；透過 i18n 架構與後台翻譯編輯器，讓平台可彈性切換繁體中文與英文介面。

## Glossary

- **System**：SAFE-AI Excel 梳理小工具後端服務
- **Frontend**：SAFE-AI Excel 梳理小工具前端 React 應用程式
- **Admin**：具有管理者角色 (role='admin') 的使用者
- **User**：具有一般使用者角色 (role='user') 的使用者
- **Quota**：每位使用者在一個重置週期內可執行的最大評估次數
- **Reset_Period**：配額自動回補的時間週期，可為「每日」或「每週」
- **Assessment**：一次完整的資料品質評估操作，消耗 1 次配額
- **Stepper**：前端頁面頂部的步驟導航列，顯示目前流程進度
- **Translation_Key**：i18n 翻譯系統中用來對應翻譯文字的唯一識別碼
- **Locale**：語言地區設定，支援 zh-TW（繁體中文）與 en（英文）

## Requirements

### Requirement 1: 角色系統

**User Story:** 身為系統管理者，我希望使用者具有角色區分，以便控制不同使用者的權限範圍。

#### Acceptance Criteria

1. THE System SHALL 為每位使用者指定一個角色，角色值為 'admin' 或 'user'
2. WHEN 新使用者註冊時，THE System SHALL 將其角色預設為 'user'
3. THE System SHALL 在 JWT token 的 payload 中包含使用者的 role 欄位
4. WHEN 使用者的角色為 'admin' 時，THE Frontend SHALL 在導航中顯示管理後台入口連結

### Requirement 2: 管理後台存取控制

**User Story:** 身為系統管理者，我希望只有 Admin 可以存取管理後台，以確保一般使用者無法操作管理功能。

#### Acceptance Criteria

1. WHEN 角色為 'admin' 的使用者存取 /admin 路由時，THE Frontend SHALL 顯示管理後台頁面
2. WHEN 角色為 'user' 的使用者嘗試存取 /admin 路由時，THE Frontend SHALL 將其導向首頁並顯示「無權限存取」訊息
3. WHEN 角色為 'user' 的使用者呼叫管理相關 API 時，THE System SHALL 回傳 HTTP 403 狀態碼並附帶「權限不足」錯誤訊息
4. THE System SHALL 在每個管理 API 端點驗證請求者的角色為 'admin'

### Requirement 3: 使用者管理列表

**User Story:** 身為 Admin，我希望在管理後台看到所有使用者的資訊與配額使用狀況，以便監控系統使用情形。

#### Acceptance Criteria

1. WHEN Admin 進入使用者管理頁面時，THE System SHALL 回傳所有使用者的列表，包含 email、已使用配額數量、剩餘配額數量
2. THE System SHALL 以分頁方式提供使用者列表，每頁預設 20 筆
3. WHEN Admin 查看使用者列表時，THE Frontend SHALL 顯示每位使用者的 email、已使用配額與剩餘配額

### Requirement 4: 配額設定

**User Story:** 身為 Admin，我希望能設定全域配額上限與重置週期，以便靈活控管使用者的評估次數。

#### Acceptance Criteria

1. THE System SHALL 維護一組全域配額設定，包含最大評估次數 (max_assessments) 與重置週期 (reset_period)
2. WHEN Admin 透過管理後台修改配額設定時，THE System SHALL 儲存新的 max_assessments 值（正整數）與 reset_period 值（'daily' 或 'weekly'）
3. WHEN 配額設定不存在時，THE System SHALL 使用預設值：max_assessments = 5，reset_period = 'daily'
4. IF Admin 提交的 max_assessments 小於 1 或 reset_period 非 'daily'/'weekly'，THEN THE System SHALL 回傳 HTTP 400 狀態碼並附帶驗證錯誤訊息

### Requirement 5: 配額消耗與執行限制

**User Story:** 身為系統管理者，我希望每次評估消耗一次配額，且配額用盡時阻止使用者繼續評估，以實現使用量控管。

#### Acceptance Criteria

1. WHEN 使用者成功啟動一次評估時，THE System SHALL 將該使用者在當前重置週期內的已使用配額數量增加 1
2. WHEN 使用者的已使用配額等於 max_assessments 時，THE System SHALL 拒絕新的評估請求並回傳 HTTP 403 狀態碼與錯誤訊息「評估次數已用盡，請聯繫管理員」
3. WHILE 使用者的配額已耗盡，THE Frontend SHALL 將「重新上傳檔案」按鈕設為 disabled 狀態
4. WHILE 使用者的配額已耗盡，THE Frontend SHALL 在使用者 hover 該 disabled 按鈕時顯示 tooltip「評估次數已用盡，請聯繫管理員」

### Requirement 6: 配額自動回補

**User Story:** 身為系統管理者，我希望配額在每個重置週期開始時自動回補至上限，以免手動重置。

#### Acceptance Criteria

1. WHEN reset_period 設定為 'daily' 且系統時間超過當日午夜 (00:00 UTC+8) 時，THE System SHALL 將所有使用者的已使用配額數量重置為 0
2. WHEN reset_period 設定為 'weekly' 且系統時間超過每週一午夜 (00:00 UTC+8) 時，THE System SHALL 將所有使用者的已使用配額數量重置為 0
3. THE System SHALL 記錄每位使用者的 last_quota_reset 時間戳，用以判斷是否需要回補
4. WHEN 使用者在重置時間點後首次呼叫需要配額的 API 時，THE System SHALL 先檢查並執行回補邏輯再判斷配額

### Requirement 7: 評估記錄查詢

**User Story:** 身為 Admin，我希望能查看每位使用者的評估歷史記錄，以便追蹤使用情形與結果。

#### Acceptance Criteria

1. WHEN Admin 查詢特定使用者的評估記錄時，THE System SHALL 回傳該使用者所有評估的列表，包含時間戳、檔案名稱、與評估分數
2. THE System SHALL 以時間戳降序排列評估記錄
3. THE System SHALL 以分頁方式提供評估記錄，每頁預設 20 筆

### Requirement 8: 流程步驟狀態保存

**User Story:** 身為使用者，我希望已完成的步驟能保存狀態，以便我回顧或重做特定步驟而不遺失進度。

#### Acceptance Criteria

1. THE Frontend SHALL 為每個流程步驟（上傳、評估、分流、梳理、產出、存證、問答）維護完成狀態
2. WHEN 使用者導航回已完成的「上傳」步驟時，THE Frontend SHALL 以唯讀模式顯示已上傳的檔案資訊（檔案名稱、列數、欄數），並提供「重新上傳檔案」按鈕
3. WHEN 使用者導航回已完成的「評估」步驟時，THE Frontend SHALL 顯示最近一次的評估結果
4. WHEN 使用者導航回已完成的「梳理」步驟時，THE Frontend SHALL 以唯讀模式顯示梳理結果，並提供「重新梳理」選項

### Requirement 9: Stepper 導航限制

**User Story:** 身為使用者，我希望尚未達到的步驟無法點擊，以避免跳過必要步驟導致錯誤。

#### Acceptance Criteria

1. WHILE 某個步驟尚未達到（步驟索引大於目前進度），THE Frontend SHALL 將該步驟在 Stepper 中以灰色顯示且禁止點擊
2. WHEN 使用者點擊尚未達到的步驟時，THE Frontend SHALL 顯示提示訊息「請先完成前面的步驟」
3. THE Frontend SHALL 允許使用者點擊已完成的步驟及目前進行中的步驟

### Requirement 10: 多語系架構

**User Story:** 身為使用者，我希望能切換介面語言，以便在繁體中文與英文之間選擇最適合的語言。

#### Acceptance Criteria

1. THE Frontend SHALL 支援繁體中文 (zh-TW) 與英文 (en) 兩種語系
2. THE Frontend SHALL 以 zh-TW 作為預設語系
3. WHEN 使用者點擊 Header 中的語言切換按鈕時，THE Frontend SHALL 立即切換所有介面文字為所選語言
4. THE Frontend SHALL 將使用者的語言偏好儲存於 localStorage
5. WHEN 頁面載入時，THE Frontend SHALL 從 localStorage 讀取語言偏好，若存在則使用該設定

### Requirement 11: 翻譯內容涵蓋範圍

**User Story:** 身為使用者，我希望所有介面文字皆支援中英文切換，以確保完整的多語系體驗。

#### Acceptance Criteria

1. THE Frontend SHALL 將所有硬編碼中文字串替換為 Translation_Key 引用，涵蓋：頁面標題、按鈕標籤、表單標籤、錯誤訊息
2. THE Frontend SHALL 使用 Translation_Key 顯示指標名稱（例如 列完整度 / Row Completeness）
3. THE System SHALL 為後端產生的字串（問題標題、問題描述）提供中英文版本
4. THE Frontend SHALL 使用 Translation_Key 顯示狀態標籤（Ready / 就緒、Conditional / 有條件通過、Not Ready / 未就緒）
5. THE Frontend SHALL 使用 Translation_Key 顯示 Stepper 步驟標籤
6. THE Frontend SHALL 保留專有名詞原文不翻譯：「SAFE-AI」、「S.A.F.E.-AI」、「AI Readiness」

### Requirement 12: 翻譯資料來源

**User Story:** 身為系統管理者，我希望翻譯內容從後端 API 載入，以便透過管理後台即時更新翻譯而無需重新部署。

#### Acceptance Criteria

1. THE System SHALL 提供 API 端點回傳指定語系的所有翻譯鍵值對
2. WHEN Frontend 啟動時，THE Frontend SHALL 從後端 API 載入翻譯資料
3. IF 翻譯 API 不可用，THEN THE Frontend SHALL 使用前端內建的預設翻譯檔案作為 fallback
4. THE System SHALL 將翻譯鍵值對儲存於資料庫中

### Requirement 13: 管理後台翻譯編輯器

**User Story:** 身為 Admin，我希望在管理後台直接編輯翻譯內容，以便即時修改介面文字而不需修改程式碼。

#### Acceptance Criteria

1. WHEN Admin 進入翻譯編輯器頁面時，THE Frontend SHALL 以鍵值對列表形式顯示所有翻譯項目，可依語系篩選
2. WHEN Admin 修改某個 Translation_Key 的翻譯值並儲存時，THE System SHALL 將更新後的值寫入資料庫
3. WHEN Admin 儲存翻譯修改時，THE System SHALL 回傳成功回應並使前端翻譯快取失效
4. THE Frontend SHALL 提供搜尋功能，讓 Admin 可依 Translation_Key 或翻譯值查找項目

### Requirement 14: 資料庫結構變更

**User Story:** 身為開發者，我需要資料庫結構支援角色、配額與翻譯功能，以便後端服務正確儲存與查詢相關資料。

#### Acceptance Criteria

1. THE System SHALL 在 users 資料表新增 role 欄位，型別為 VARCHAR，預設值為 'user'，允許值為 'admin' 和 'user'
2. THE System SHALL 建立 quota_settings 資料表，包含 id、max_assessments (INTEGER)、reset_period (VARCHAR)、updated_at 欄位
3. THE System SHALL 在 users 資料表新增 last_quota_reset 欄位，型別為 TIMESTAMP WITH TIME ZONE
4. THE System SHALL 建立 translations 資料表，包含 id、locale (VARCHAR)、key (VARCHAR)、value (TEXT)、updated_at 欄位
5. THE System SHALL 為 translations 資料表的 (locale, key) 組合建立唯一索引
6. THE System SHALL 透過資料庫遷移 (migration) 腳本執行上述結構變更
