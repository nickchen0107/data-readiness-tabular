# Implementation Plan: SAFE-AI Excel 梳理小工具

## Overview

實作一個 Docker-based、前後端分離的 MVP 資料品質評估與梳理平台。採用 React + TypeScript 前端、Go (Gin) 後端、PostgreSQL 資料庫。按模組逐步建構，每個步驟以可運行的增量交付為目標。

## Tasks

- [x] 1. Project Infrastructure Setup
  - Docker Compose 環境、Go 專案初始化、React 專案初始化、PostgreSQL 設定、Nginx 反向代理配置
  - _Requirements: 18.1, 18.2, 18.3_

  - [x] 1.1 Create Docker Compose configuration
    - 建立 `docker-compose.yml`，定義 frontend (Nginx)、backend (Go)、db (PostgreSQL 16) 三個 service
    - 配置 volumes: `upload_data` (Excel 檔案) 和 `pg_data` (資料庫)
    - 設定 environment variables: DATABASE_URL, JWT_SECRET, GEMINI_API_KEY, BLOCKCHAIN_API_URL
    - 配置 service dependencies: frontend → backend → db
    - _Requirements: 18.1, 18.2_

  - [x] 1.2 Initialize Go backend project
    - 建立 `/backend/` 目錄結構：`cmd/server/main.go`, `internal/`, `pkg/`, `migrations/`
    - 初始化 `go.mod`，加入核心 dependencies: gin, sqlx/pgx, uuid, bcrypt, jwt-go, excelize, rapid, testify
    - 建立基本 Gin server 啟動檔（port 8080），含 health check endpoint `GET /api/health`
    - 建立 Dockerfile (multi-stage build)
    - _Requirements: 18.1, 19.1_

  - [x] 1.3 Initialize React frontend project
    - 建立 `/frontend/` 目錄，使用 Vite + React 18 + TypeScript 模板
    - 安裝核心 dependencies: react-router-dom, axios, recharts, @tanstack/react-table, rc-slider
    - 建立基本 App.tsx 含 React Router 骨架
    - 建立 Dockerfile (multi-stage: build with Node, serve with Nginx)
    - _Requirements: 18.1, 20.1_

  - [x] 1.4 Configure Nginx reverse proxy
    - 建立 `frontend/nginx.conf`：靜態檔 serve、`/api/*` proxy 至 backend:8080、SPA fallback `try_files`
    - 設定 `client_max_body_size 55m` (略大於 50MB 上傳限制)
    - 配置 CORS headers 與 CSP security headers
    - _Requirements: 18.3_

  - [x] 1.5 Create PostgreSQL database migration
    - 建立 `/backend/migrations/001_initial_schema.sql`
    - 包含所有 tables: users, uploads, assessments, cleaning_sessions, evidence_records, system_settings, login_attempts
    - 插入 system_settings 預設權重值 (20/20/15/10/15/20)
    - 建立 migration runner (golang-migrate 或手動 SQL execution)
    - _Requirements: 18.1_

  - [x] 1.6 Set up shared backend infrastructure
    - 建立統一 error response 結構 (`pkg/response/error.go`): code, message, details
    - 建立 database connection pool with retry (3 attempts, exponential backoff)
    - 建立 config package 讀取環境變數
    - 建立 CORS middleware、Recovery middleware、Request logging middleware
    - _Requirements: 19.2, 19.3, 19.4, 19.5, 20.2_

  - [x]* 1.7 Write property test for API error response structure
    - **Property 29: API error responses are structured and safe**
    - 驗證所有 error response 為 valid JSON，含 error.code + error.message，message 為繁體中文，500 不含 stack trace
    - **Validates: Requirements 19.2, 19.3, 19.4, 19.5, 20.2**

- [x] 2. Authentication Module
  - 使用者註冊、登入、登出、JWT middleware、rate limiting
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 2.1, 2.2, 2.3, 2.4, 2.5_

  - [x] 2.1 Implement User model and repository
    - 建立 `/backend/internal/auth/model.go`: User struct, RegisterRequest, LoginRequest, TokenResponse
    - 建立 `/backend/internal/auth/repository.go`: CreateUser, GetByEmail, GetByID (sqlx queries)
    - 實作 bcrypt hash (cost ≥ 12) 與 password validation (8-72 chars)
    - _Requirements: 1.1, 1.4_

  - [x] 2.2 Implement Register endpoint
    - 建立 `/backend/internal/auth/service.go`: Register function
    - Email format validation (RFC 5322 簡化)、password length check
    - 重複 email 檢查 → 409 Conflict
    - Handler: `POST /api/auth/register` → 回傳 `{id, email, created_at}`
    - _Requirements: 1.1, 1.2, 1.3, 1.4_

  - [x]* 2.3 Write property tests for registration
    - **Property 1: Registration accepts valid inputs and persists correctly**
    - **Property 2: Registration rejects duplicate emails**
    - **Property 3: Registration rejects invalid inputs**
    - **Validates: Requirements 1.1, 1.2, 1.3, 1.4**

  - [x] 2.4 Implement Login and JWT issuance
    - 建立 JWT token generation (HS256, configurable expiry default 24h)
    - 驗證 email + bcrypt compare → issue token with user_id in payload
    - Handler: `POST /api/auth/login` → 回傳 `{token, expires_at}`
    - 錯誤回傳不揭露哪個欄位錯誤
    - _Requirements: 2.1, 2.2_

  - [x]* 2.5 Write property test for JWT issuance
    - **Property 4: Login produces valid JWT for valid credentials**
    - **Validates: Requirements 2.1**

  - [x] 2.6 Implement Logout and JWT middleware
    - 建立 token blacklist (DB table or in-memory with TTL)
    - `POST /api/auth/logout` → 加入 blacklist
    - JWT middleware: validate token, check blacklist, extract user_id, check expiry
    - `GET /api/auth/me` → 回傳當前使用者資訊
    - _Requirements: 2.3, 2.4_

  - [x] 2.7 Implement rate limiting for login
    - 建立 login_attempts repository: 記錄每次登入嘗試
    - 實作 CheckRateLimit: 同一 email 15 分鐘內 >5 次失敗 → 暫時封鎖
    - 整合至 login handler
    - _Requirements: 2.5_

- [x] 3. Checkpoint - Infrastructure & Auth
  - Ensure all tests pass, ask the user if questions arise.

- [x] 4. File Upload Module
  - 檔案上傳、格式驗證、xlsx/csv 解析、sheet 選擇、metadata 擷取
  - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 3.7, 3.8_

  - [x] 4.1 Implement Upload endpoint and file validation
    - 建立 `/backend/internal/upload/service.go` 與 handler
    - `POST /api/upload` multipart/form-data 接收
    - 檔案格式驗證 (magic bytes for xlsx, extension for csv)
    - 檔案大小驗證 (≤50MB)
    - UUID-based 儲存路徑 → Docker volume
    - 儲存 metadata 至 DB (filename, file_size, user_id, created_at)
    - _Requirements: 3.1, 3.2, 3.3_

  - [x] 4.2 Implement xlsx parsing with excelize
    - 使用 excelize library 解析 xlsx
    - 擷取 sheet names list
    - 偵測 merged cells locations
    - 取 formula cells 的 computed values
    - 計算 row_count, col_count
    - 多 sheet 回傳 list；單 sheet 自動選擇
    - _Requirements: 3.4, 3.5, 3.6_

  - [x] 4.3 Implement CSV parsing with encoding support
    - 支援 UTF-8 和 UTF-8 with BOM
    - 偵測 encoding (BOM detection)
    - 解析 CSV 為 SheetData 結構
    - 驗證 row count ≤ 100,000
    - Corruption detection (malformed CSV → 400 error)
    - _Requirements: 3.7, 3.8, 3.2_

  - [x] 4.4 Implement sheet selection API
    - `GET /api/upload/{id}/sheets` → 回傳 sheet name list
    - `POST /api/upload/{id}/select-sheet` → 記錄 selected_sheet
    - 建立 SheetData loader: 從儲存的檔案讀取指定 sheet 為 SheetData struct
    - _Requirements: 3.4_

- [x] 5. Assessment Engine
  - 六項 deterministic 指標計算、總分、分級、問題偵測
  - _Requirements: 4.1, 5.1, 6.1, 7.1, 8.1, 9.1, 10.1, 10.5, 10.6, 10.7, 10.8_

  - [x] 5.1 Implement Row Completeness indicator
    - 建立 `/backend/internal/assessment/indicators.go`
    - 實作 `CalculateRowCompleteness(data *SheetData) float64`
    - 邏輯：每列 non-empty cells / total cols → 平均 → ×100
    - 0 data rows → return 0
    - 空值判定：null, whitespace-only, no value
    - _Requirements: 4.1, 4.3_

  - [x]* 5.2 Write property test for Row Completeness
    - **Property 5: Row Completeness formula correctness**
    - 使用 rapid 產生隨機 grid，驗證計算公式正確性
    - **Validates: Requirements 4.1, 4.3**

  - [x] 5.3 Implement Column Completeness indicator
    - 實作 `CalculateColumnCompleteness(data *SheetData) (float64, []ColumnDetail)`
    - 邏輯：每欄 non-empty / total rows → 平均 → ×100
    - 同時輸出 per-column 完整度 ratios
    - 0 data rows → return 0
    - _Requirements: 5.1, 5.3, 5.4_

  - [x]* 5.4 Write property test for Column Completeness
    - **Property 6: Column Completeness formula correctness**
    - **Validates: Requirements 5.1, 5.3, 5.4**

  - [x] 5.5 Implement Format Consistency indicator
    - 實作 format type detection (priority: date > numeric > boolean > text)
    - 實作 `CalculateFormatConsistency(data *SheetData) float64`
    - 每欄：dominant type count / non-empty count → 平均所有 valid columns → ×100
    - 空欄位排除；tie-break by priority
    - _Requirements: 6.1, 6.2, 6.3, 6.4_

  - [x]* 5.6 Write property tests for Format Consistency
    - **Property 7: Format type detection priority**
    - **Property 8: Format Consistency calculation**
    - **Validates: Requirements 6.1, 6.2, 6.3, 6.4**

  - [x] 5.7 Implement Duplicate/Similar indicator
    - 實作 exact duplicate detection via full-row SHA-256 hash
    - 實作 eligible column selection (text type, 5%-80% cardinality, max 5 cols left-to-right)
    - 實作 Levenshtein distance calculation (threshold ≤ 2)
    - 計算 score = max(0, (1 - (exact + near × 0.5) / total_rows) × 100)
    - 0 data rows → return 100
    - _Requirements: 7.1, 7.2, 7.3, 7.5_

  - [x]* 5.8 Write property test for Duplicate/Similar
    - **Property 9: Duplicate/Similar score formula**
    - **Validates: Requirements 7.1, 7.2, 7.3, 7.5**

  - [x] 5.9 Implement Table Structure Quality indicator
    - 起始分數 100，逐項扣分：merged cells (-20), multi-layer headers (-20), subtotal rows (-15), multiple tables (-25), notes in data (-10)
    - 每項最多扣一次，最低 0
    - Multi-layer header detection: first 5 rows, >1 row all-text no-repeat
    - Subtotal detection: "小計", "合計", "total", "subtotal" (case-insensitive)
    - Multiple tables: ≥2 consecutive empty rows
    - Notes detection: text col stddev > mean × 3
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 8.6, 8.8_

  - [x]* 5.10 Write property tests for Table Structure Quality
    - **Property 10: Table Structure Quality deductions with floor**
    - **Property 11: Multi-layer header detection**
    - **Property 12: Subtotal row detection**
    - **Property 13: Multiple tables detection**
    - **Validates: Requirements 8.1, 8.2, 8.3, 8.4, 8.6**

  - [x] 5.11 Implement AI Query Readiness indicator
    - 五項子條件加分制 (每項 +20，max 100)：
      - Identifier column: unique ratio > 80%
      - Time column: date parse > 60% of first min(100, N) rows
      - Category column: unique count < 20% rows AND > 1
      - Numeric column: >80% parseable as number
      - Column name quality: non-empty, non-duplicate, length > 1
    - 0 data rows → return 0
    - _Requirements: 9.1, 9.3, 9.4_

  - [x]* 5.12 Write property test for AI Query Readiness
    - **Property 14: AI Query Readiness sub-condition scoring**
    - **Validates: Requirements 9.1, 9.3, 9.4**

  - [x] 5.13 Implement total score calculation and grading
    - Weighted sum: 六項 × 各自權重 → round to 1 decimal
    - Grading: ≥80 Ready, ≥60 Conditional, <60 Not Ready
    - Issue detection: 掃描各指標結果產出問題清單 (severity, affected_rows, description)
    - Weight validation: sum ≠ 100% → reject
    - 任一指標失敗 → halt, return error naming failed indicator
    - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5, 10.6, 10.7, 10.8_

  - [x]* 5.14 Write property tests for total score and grading
    - **Property 15: Weighted score calculation and grading**
    - **Property 16: Assessment determinism**
    - **Property 17: Invalid weight sum rejection**
    - **Validates: Requirements 10.1, 10.2, 10.3, 10.4, 10.5, 10.8**

  - [x] 5.15 Implement Assessment API endpoints
    - `POST /api/assess` → {upload_id, sheet_name} → run full assessment → return assessment_id
    - `GET /api/assess/{id}` → return full Assessment result
    - `GET /api/assess/{id}/issues` → return issues list
    - 儲存 assessment 至 DB including weights_snapshot
    - _Requirements: 10.1, 10.6_

- [x] 6. Checkpoint - Assessment Engine
  - Ensure all tests pass, ask the user if questions arise.

- [x] 7. Cleaning Engine
  - 4 批次規則 + 單列操作 + cleaning log
  - _Requirements: 12.1, 12.2, 12.3, 12.4, 12.5, 13.1, 13.2_

  - [x] 7.1 Implement date normalization rule
    - 偵測 date-type columns (reuse format detection from assessment)
    - 解析所有支援格式 (yyyy/MM/dd, yyyy-MM-dd, ROC yyy.M.d)
    - 統一輸出 yyyy-MM-dd
    - 每個修改記錄至 cleaning log
    - _Requirements: 12.1_

  - [x]* 7.2 Write property test for date normalization
    - **Property 18: Date normalization rule**
    - **Validates: Requirements 12.1**

  - [x] 7.3 Implement deduplication rule
    - Full-row SHA-256 hash 比對
    - 保留第一筆出現的 row，移除後續重複
    - 維持 original relative order
    - 記錄至 cleaning log
    - _Requirements: 12.2_

  - [x]* 7.4 Write property test for deduplication
    - **Property 19: Deduplication rule preserves uniqueness**
    - **Validates: Requirements 12.2**

  - [x] 7.5 Implement company name normalization rule
    - 定義 suffix list: Co., Company, 公司, 股份有限公司, etc.
    - 對 eligible text columns: 移除 suffix → grouping → 統一為最長版本
    - 記錄每個修改至 cleaning log
    - _Requirements: 12.3_

  - [x]* 7.6 Write property test for company name normalization
    - **Property 20: Company name normalization**
    - **Validates: Requirements 12.3**

  - [x] 7.7 Implement subtotal row removal rule
    - 偵測含 "小計", "合計", "total", "subtotal" (case-insensitive) 的 rows
    - 移除所有匹配 rows
    - 記錄至 cleaning log
    - _Requirements: 12.4_

  - [x]* 7.8 Write property test for subtotal removal
    - **Property 21: Subtotal row removal**
    - **Validates: Requirements 12.4**

  - [x] 7.9 Implement row operations (Fill N/A, Delete row)
    - Fill N/A: 指定 row 所有 empty cells → "N/A"，non-empty 不變
    - Delete row: 移除指定 row，dataset 行數 -1
    - 兩者皆記錄至 cleaning log
    - _Requirements: 13.1, 13.2_

  - [x]* 7.10 Write property tests for row operations
    - **Property 22: Cleaning operations produce log entries**
    - **Property 23: Fill N/A preserves non-empty and fills empty**
    - **Property 24: Row deletion shrinks dataset**
    - **Validates: Requirements 12.5, 13.1, 13.2**

  - [x] 7.11 Implement Cleaning API endpoints
    - `POST /api/clean/apply` → {assessment_id, rules: [...], row_ops: [...]} → 執行規則 → return session
    - `GET /api/clean/{id}/preview` → 回傳 cleaned data preview
    - `GET /api/clean/{id}/log` → 回傳 cleaning log entries
    - 建立 cleaning_sessions record，含 rows_before, rows_after, score_before
    - Re-assess: 梳理後重跑 assessment engine → score_after
    - 儲存 refined file 至 docker volume
    - _Requirements: 12.5_

- [x] 8. Checkpoint - Cleaning Engine
  - Ensure all tests pass, ask the user if questions arise.

- [x] 9. Export Module
  - refined.xlsx 產出、PDF 報告產生、cleaning.log 打包、下載 endpoints
  - _Requirements: 14.1, 14.2, 14.3, 14.4_

  - [x] 9.1 Implement refined.xlsx generation
    - 使用 excelize 建立新 xlsx file
    - 寫入 cleaned data (headers + rows)
    - 儲存至 docker volume，path 記錄至 cleaning_session
    - _Requirements: 14.1_

  - [x] 9.2 Implement PDF report generation
    - 使用 Go PDF library (e.g., gofpdf 或 maroto)
    - 報告內容：Readiness_Score, 六項指標分數, 問題摘要, 前後對比統計, 梳理規則摘要
    - 含 ring score chart (可用 SVG embedded 或 Go 繪圖)
    - 品牌色系排版
    - 中文字體支援 (embedded font)
    - _Requirements: 14.2_

  - [x] 9.3 Implement cleaning.log JSON packaging
    - 從 cleaning_session 取出 cleaning_log JSONB
    - 格式化為 indented JSON file
    - 儲存至 volume
    - _Requirements: 14.3_

  - [x] 9.4 Implement download endpoints
    - `GET /api/export/{id}/xlsx` → Content-Disposition: attachment, application/vnd.openxmlformats
    - `GET /api/export/{id}/pdf` → Content-Disposition: attachment, application/pdf
    - `GET /api/export/{id}/log` → Content-Disposition: attachment, application/json
    - Ownership verification: 確認當前 user 為 session owner
    - _Requirements: 14.4_

- [x] 10. Evidence Module
  - SHA-256 hashing、blockchain API proxy、status tracking
  - _Requirements: 15.1, 15.2, 15.3, 15.4, 15.5_

  - [x] 10.1 Implement SHA-256 hash computation
    - 計算 refined.xlsx, cleaning.log, report.pdf 三個檔案的 SHA-256 hex digest
    - 建立 `/backend/internal/evidence/service.go`
    - _Requirements: 15.1_

  - [x]* 10.2 Write property test for SHA-256 computation
    - **Property 25: SHA-256 hash correctness**
    - **Validates: Requirements 15.1**

  - [x] 10.3 Implement blockchain API proxy
    - `POST /api/evidence/submit` → compute hashes → forward to external blockchain API
    - Request payload: dataset_hash, cleaning_log_hash, report_hash, timestamp, tool_version, rule_version, operator_id, metadata
    - 儲存回傳的 record_id, signature_status 至 evidence_records table
    - 超時 (10s) → 503, 儲存為 "pending"
    - Blockchain unavailable → 清楚 error message, 不 crash
    - _Requirements: 15.2, 15.3, 15.5_

  - [x] 10.4 Implement evidence query endpoint
    - `GET /api/evidence/{record_id}` → proxy to external blockchain GET API → return result
    - 同時從 local DB 取得 record 補充資訊
    - _Requirements: 15.4_

- [x] 11. QA Module (Gemini Integration)
  - Gemini API 串接、data insufficiency guardrail、suggested questions、consent flow
  - _Requirements: 16.1, 16.2, 16.3, 16.5_

  - [x] 11.1 Implement QA Service and Gemini client
    - 建立 `/backend/internal/qa/service.go`
    - Gemini API REST client (structured prompt with CSV data)
    - Prompt template: system instruction + CSV data snippet + user question
    - 完整回應 (non-streaming), timeout 30s, retry once on 5xx
    - _Requirements: 16.1_

  - [x] 11.2 Implement data insufficiency guardrail
    - 檢查問題涉及的 column 缺漏率
    - 若 column missing rate > 50% → return "資料不足" + explanation, 不呼叫 Gemini
    - _Requirements: 16.2_

  - [x]* 11.3 Write property test for data insufficiency guardrail
    - **Property 26: Data insufficiency guardrail**
    - **Validates: Requirements 16.2**

  - [x] 11.4 Implement suggested questions generation
    - 根據 dataset column names 產出 3 個建議問題
    - 每個問題必須 reference 至少一個實際 column name
    - `GET /api/qa/suggestions/{assess_id}` endpoint
    - _Requirements: 16.3_

  - [x]* 11.5 Write property test for suggested questions
    - **Property 27: Suggested questions reference column names**
    - **Validates: Requirements 16.3**

  - [x] 11.6 Implement consent enforcement and QA endpoint
    - `POST /api/qa/ask` → 檢查 consent flag → 若未 consent 則 block
    - 執行 QA: 分別用 original data 和 cleaned data 呼叫 Gemini → 回傳 side-by-side answers
    - _Requirements: 16.5, 16.1_

- [x] 12. Checkpoint - Export, Evidence & QA
  - Ensure all tests pass, ask the user if questions arise.

- [x] 13. Weight Settings
  - 權重 API、持久化、前端 sliders
  - _Requirements: 17.1, 17.2, 17.3_

  - [x] 13.1 Implement Settings API
    - `GET /api/settings/weights` → 從 system_settings 讀取當前權重 → return JSON
    - `PUT /api/settings/weights` → validate sum = 100% → persist to DB → return updated
    - 權重變更不影響歷史 assessment (已 snapshot)
    - _Requirements: 17.1, 17.2, 17.3_

  - [x]* 13.2 Write property test for historical assessment immutability
    - **Property 28: Historical assessment immutability**
    - **Validates: Requirements 17.3**

- [x] 14. Frontend Implementation
  - 所有頁面 / views: login, upload, assessment, cleaning, export, evidence, QA, settings
  - 參考 `SAFE-AI_Excel梳理小工具_演示原型.html` 作為 UI 設計指引
  - _Requirements: 11.1, 11.2, 11.3, 16.4, 17.1, 20.1_

  - [x] 14.1 Implement shared layout, routing, and auth context
    - App Router 配置: Landing, Login, Register, Dashboard (protected)
    - AuthContext: JWT token storage (localStorage), login/logout state, axios interceptor for Authorization header
    - Protected route wrapper (redirect to login if no token)
    - Shared Layout: sidebar navigation, header with user info + logout
    - API service layer (`/frontend/src/api/`) with axios instance and error handling
    - 所有 UI text 為繁體中文
    - _Requirements: 20.1_

  - [x] 14.2 Implement Login and Register pages
    - Login form: email + password + submit + error display + rate limit message
    - Register form: email + password + confirm password + validation feedback
    - 成功 login → 儲存 token → redirect to dashboard
    - 參考原型 HTML 的登入頁面風格
    - _Requirements: 1.1, 2.1, 20.1_

  - [x] 14.3 Implement Upload view
    - Drag & drop file upload zone + file picker button
    - 上傳進度指示
    - 檔案驗證回饋 (format, size)
    - Sheet 選擇 modal (多 sheet 時)
    - 上傳成功 → 顯示 metadata (filename, rows, cols, sheets)
    - _Requirements: 3.1, 3.4, 20.1_

  - [x] 14.4 Implement Assessment view
    - Ring chart: 總分 (Recharts PieChart)
    - 六項指標 bar chart
    - 分級 badge (Ready / Conditional / Not Ready)
    - 問題清單 table (severity, affected rows, recommendation)
    - Per-column completeness details (expandable)
    - _Requirements: 10.1, 10.6, 20.1_

  - [x] 14.5 Implement Routing Decision view
    - 兩張卡片："以現況梳理" 和 "補齊後重新上傳"
    - Not Ready 時顯示風險警告 (允許繼續)
    - "補齊後重新上傳" → redirect to upload view
    - _Requirements: 11.1, 11.2, 11.3_

  - [x] 14.6 Implement Cleaning view
    - Data table (TanStack Table): 顯示問題列 highlight
    - 批次規則 panel: 4 checkboxes (日期, 重複, 客戶名, 小計) + Apply button
    - 單列操作: 每列 hover 出現 "填入 N/A" / "刪除該列" buttons
    - Cleaning progress / result display
    - Before/after statistics comparison
    - _Requirements: 12.1, 12.2, 12.3, 12.4, 13.1, 13.2, 20.1_

  - [x] 14.7 Implement Export view
    - 前後統計對比 cards (rows before/after, score before/after)
    - 三個 download buttons: refined.xlsx, report.pdf, cleaning.log
    - 下載進度指示
    - _Requirements: 14.4, 20.1_

  - [x] 14.8 Implement Evidence view
    - Evidence Record 卡片: hash values, timestamp, record_id, signature_status
    - Submit evidence button → call API → show result
    - Demo Mode indicator 當 blockchain unavailable
    - 標示 "No sensitive data on-chain" / "Integrity verifiable"
    - _Requirements: 15.3, 15.5, 20.1_

  - [x] 14.9 Implement QA Comparison view
    - Consent modal (首次使用時): 說明資料將送至外部模型
    - 建議問題 3 個 quick-select buttons
    - 自由輸入 text input + submit
    - 左右並排 layout: 原始資料回答 (left) vs 梳理後回答 (right)
    - "資料不足" 時顯示說明 card
    - Loading state 顯示
    - _Requirements: 16.1, 16.4, 16.5, 20.1_

  - [x] 14.10 Implement Weight Settings view
    - 六個 slider controls (rc-slider)
    - 即時計算並顯示 sum (需 = 100%)
    - Sum ≠ 100% 時 disable save button + warning
    - Save button → PUT /api/settings/weights
    - 當前值載入 from GET /api/settings/weights
    - _Requirements: 17.1, 17.2, 20.1_

- [x] 15. Final Checkpoint
  - Ensure all tests pass, ask the user if questions arise.
  - 驗證 `docker compose up` 能成功啟動所有服務
  - 驗證 Nginx 正確 serve frontend 並 proxy API requests

- [x] 16. Issue 清單 UX 增強
  - 後端 Issue struct 加 Examples 欄位、描述分號改換行、前端問題清單改為可展開式 Accordion
  - _Requirements: 10.6_

  - [x] 16.1 Add Examples field to Issue struct and fix description format
    - 在 `Issue` struct 新增 `Examples []string` 欄位（JSON tag: `examples`）
    - 修改 `DetectIssues` 各段邏輯，填入具體資料範例（如：缺漏欄位的實際空值列號、格式不一致的實際值樣本、重複列的實際內容摘要）
    - 將 description 中的分號（；）分隔改為換行分隔（`\n`）
    - 確保後端測試可通過
    - _Requirements: 10.6_

  - [x] 16.2 Make issue list collapsible accordion with examples display
    - 將 AssessmentPage 的問題清單從扁平卡片改為可展開的 Accordion 式
    - 收合狀態顯示：severity badge + title + affected count
    - 展開狀態額外顯示：description（換行渲染）+ examples 列表（具體資料問題示例）
    - 加入展開/收合動畫與 chevron icon
    - 確保前端可正常 build
    - _Requirements: 10.6, 20.1_
    - _Depends on: 16.1_

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties from the design document (29 properties total)
- Unit tests validate specific examples and edge cases
- 參考 `SAFE-AI_Excel梳理小工具_演示原型.html` 作為前端 UI 設計指引
- 所有 API response messages 和前端 UI 均使用繁體中文
- Go PBT library: `pgregory.net/rapid`, assertions: `github.com/stretchr/testify`

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1", "1.2", "1.3"] },
    { "id": 1, "tasks": ["1.4", "1.5", "1.6"] },
    { "id": 2, "tasks": ["1.7", "2.1"] },
    { "id": 3, "tasks": ["2.2", "2.4"] },
    { "id": 4, "tasks": ["2.3", "2.5", "2.6", "2.7"] },
    { "id": 5, "tasks": ["4.1"] },
    { "id": 6, "tasks": ["4.2", "4.3"] },
    { "id": 7, "tasks": ["4.4"] },
    { "id": 8, "tasks": ["5.1", "5.3", "5.5"] },
    { "id": 9, "tasks": ["5.2", "5.4", "5.6", "5.7"] },
    { "id": 10, "tasks": ["5.8", "5.9", "5.11"] },
    { "id": 11, "tasks": ["5.10", "5.12", "5.13"] },
    { "id": 12, "tasks": ["5.14", "5.15"] },
    { "id": 13, "tasks": ["7.1", "7.3", "7.5", "7.7"] },
    { "id": 14, "tasks": ["7.2", "7.4", "7.6", "7.8", "7.9"] },
    { "id": 15, "tasks": ["7.10", "7.11"] },
    { "id": 16, "tasks": ["9.1", "9.3"] },
    { "id": 17, "tasks": ["9.2", "9.4"] },
    { "id": 18, "tasks": ["10.1"] },
    { "id": 19, "tasks": ["10.2", "10.3", "10.4"] },
    { "id": 20, "tasks": ["11.1", "11.4"] },
    { "id": 21, "tasks": ["11.2", "11.3", "11.5", "11.6"] },
    { "id": 22, "tasks": ["13.1"] },
    { "id": 23, "tasks": ["13.2"] },
    { "id": 24, "tasks": ["14.1"] },
    { "id": 25, "tasks": ["14.2", "14.3"] },
    { "id": 26, "tasks": ["14.4", "14.5", "14.6"] },
    { "id": 27, "tasks": ["14.7", "14.8", "14.9", "14.10"] },
    { "id": 28, "tasks": ["16.1"] },
    { "id": 29, "tasks": ["16.2"] }
  ]
}
```
