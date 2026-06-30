# Requirements Document

## Introduction

SAFE-AI Excel 梳理小工具是一個 Docker-based、前後端分離（React + Go）、模組化的 MVP 工具平台。使用者可上傳 Excel 檔案，經由 AI Data Readiness 評估（六項 deterministic 指標）與資料梳理流程後，產出品質提升的資料集與美觀 PDF 報告，並透過 Gemini LLM 問答對比展現梳理前後的差異。系統同時提供 Evidence 存證 API 接口供後續區塊鏈串接。

## Glossary

- **System**：SAFE-AI Excel 梳理小工具後端服務（Go RESTful API）
- **Frontend**：SAFE-AI Excel 梳理小工具前端應用程式（React + TypeScript + Nginx）
- **Upload_Service**：負責檔案上傳、格式驗證與儲存的後端模組
- **Assessment_Engine**：負責六項 AI Data Readiness 指標計算的後端模組
- **Cleaning_Engine**：負責資料梳理批次規則執行的後端模組
- **Export_Service**：負責產出 refined.xlsx、report.pdf 與 cleaning.log 的後端模組
- **Evidence_Service**：負責計算 hash 並呼叫外部區塊鏈 API 的後端模組
- **QA_Service**：負責串接 Gemini API 執行前後對比問答的後端模組
- **Auth_Service**：負責使用者註冊、登入、登出與 JWT 管理的後端模組
- **Readiness_Score**：六項指標加權計算後的 0-100 總分
- **Cleaning_Log**：JSON 格式的資料梳理操作紀錄，包含操作類型、影響列、時間戳與操作者
- **Levenshtein_Distance**：計算兩字串間最少編輯次數（插入、刪除、替換）的演算法
- **User**：已註冊並通過身份驗證的系統使用者

## Requirements

### Requirement 1: 使用者註冊

**User Story:** As a User, I want to register an account with email and password, so that I can access the system features.

#### Acceptance Criteria

1. WHEN a User submits a registration request with a valid email and a password between 8 and 72 characters (inclusive), THE Auth_Service SHALL create a new account, store the password as a bcrypt hash, and return a success response containing the created account's email.
2. IF a User submits a registration request with an email that already exists in the database, THEN THE Auth_Service SHALL return an error indicating the email is already registered without creating a duplicate account.
3. IF a User submits a registration request with an invalid email format, THEN THE Auth_Service SHALL return a validation error without creating an account.
4. IF a User submits a registration request with a password shorter than 8 characters or longer than 72 characters, THEN THE Auth_Service SHALL return a validation error indicating the password length requirement without creating an account.

### Requirement 2: 使用者登入與登出

**User Story:** As a User, I want to log in and log out of the system, so that I can securely access my data.

#### Acceptance Criteria

1. WHEN a User submits valid email and password credentials, THE Auth_Service SHALL return a JWT token with a configurable expiration time (default 24 hours) for subsequent authenticated requests.
2. WHEN a User submits incorrect credentials, THE Auth_Service SHALL return an authentication error without revealing which field is incorrect.
3. WHEN a User sends a logout request with a valid JWT token, THE Auth_Service SHALL invalidate the token so that subsequent requests using the same token are rejected with HTTP 401.
4. IF a request is made with an expired JWT token, THEN THE Auth_Service SHALL return HTTP 401 indicating the token has expired.
5. IF more than 5 failed login attempts occur for the same email within a 15-minute window, THEN THE Auth_Service SHALL temporarily block login attempts for that email and return an error indicating the account is temporarily locked.

### Requirement 3: 檔案上傳

**User Story:** As a User, I want to upload Excel or CSV files, so that I can assess and clean my data.

#### Acceptance Criteria

1. WHEN a User uploads a file in xlsx or csv format with size not exceeding 50MB and row count not exceeding 100,000 rows, THE Upload_Service SHALL accept the file, store it persistently, and record metadata (filename, file size, upload timestamp, user ID) in the database.
2. WHEN a User uploads a file exceeding 50MB or 100,000 rows, THE Upload_Service SHALL reject the upload and return an error specifying which limit was exceeded.
3. WHEN a User uploads a file in an unsupported format, THE Upload_Service SHALL reject the upload and return an error listing the supported formats (xlsx, csv).
4. WHEN an uploaded xlsx file contains multiple sheets, THE Upload_Service SHALL return the list of sheet names and allow the User to select one sheet for assessment. WHEN an xlsx file contains exactly one sheet, THE Upload_Service SHALL auto-select that sheet without prompting.
5. WHEN an uploaded xlsx file contains formula cells, THE Upload_Service SHALL extract the computed values rather than the formula expressions.
6. WHEN an uploaded xlsx file contains merged cells, THE Upload_Service SHALL detect and record the merged cell locations without performing automatic unmerging.
7. WHEN an uploaded csv file is provided, THE Upload_Service SHALL support UTF-8 and UTF-8 with BOM encodings.
8. IF an uploaded file cannot be parsed due to corruption or invalid structure, THEN THE Upload_Service SHALL reject the upload and return an error indicating the file is corrupted or unreadable.

### Requirement 4: AI Data Readiness 評估 — 列完整度

**User Story:** As a User, I want to know how complete each row is in my dataset, so that I can understand data gaps at the row level.

#### Acceptance Criteria

1. WHEN an assessment is triggered, THE Assessment_Engine SHALL calculate Row Completeness by: for each data row (excluding header rows), computing the ratio of non-empty cells to total columns, then averaging these ratios across all data rows and multiplying by 100 to produce a score from 0 to 100. A cell is considered empty if it is null, contains only whitespace characters, or has no value.
2. THE Assessment_Engine SHALL apply a default weight of 20% to the Row Completeness score in the overall Readiness_Score calculation.
3. IF the selected sheet contains 0 data rows after excluding header rows, THEN THE Assessment_Engine SHALL assign a Row Completeness score of 0.

### Requirement 5: AI Data Readiness 評估 — 欄位完整度

**User Story:** As a User, I want to know how complete each column is in my dataset, so that I can identify columns with excessive missing values.

#### Acceptance Criteria

1. WHEN an assessment is triggered, THE Assessment_Engine SHALL calculate Column Completeness by: for each column, computing the ratio of non-empty values to total data rows (excluding header rows), then averaging these ratios across all columns and multiplying by 100 to produce a score from 0 to 100. A cell is considered empty if it is null, contains only whitespace characters, or has no value.
2. THE Assessment_Engine SHALL apply a default weight of 20% to the Column Completeness score in the overall Readiness_Score calculation.
3. WHEN the assessment is complete, THE Assessment_Engine SHALL include per-column completeness ratios in the assessment output to enable identification of specific low-completeness columns.
4. IF the selected sheet contains 0 data rows after excluding header rows, THEN THE Assessment_Engine SHALL assign a Column Completeness score of 0.

### Requirement 6: AI Data Readiness 評估 — 格式一致性

**User Story:** As a User, I want to identify format inconsistencies within columns, so that I can standardize my data for reliable analysis.

#### Acceptance Criteria

1. WHEN an assessment is triggered, THE Assessment_Engine SHALL detect the format type of each non-empty value using these categories in priority order: date (yyyy/MM/dd, yyyy-MM-dd, ROC calendar yyy.M.d), numeric (integers, thousands-separated, decimals), boolean (true/false, 是/否, Y/N), and text (all others). IF a value matches multiple categories, THEN THE Assessment_Engine SHALL assign the first matching category in the listed priority order.
2. WHEN an assessment is triggered, THE Assessment_Engine SHALL identify the dominant format type for each column as the format type with the highest count among non-empty values. IF two or more format types share the highest count, THEN THE Assessment_Engine SHALL select the one with the highest priority according to the category order defined in criterion 1.
3. WHEN an assessment is triggered, THE Assessment_Engine SHALL calculate Format Consistency for each column as the ratio of values matching the dominant format type to total non-empty values, then average across all columns that contain at least one non-empty value, and express the result as a score from 0 to 100.
4. IF a column contains zero non-empty values, THEN THE Assessment_Engine SHALL exclude that column from the Format Consistency average calculation.
5. THE Assessment_Engine SHALL apply a default weight of 15% to the Format Consistency score in the overall Readiness_Score calculation.

### Requirement 7: AI Data Readiness 評估 — 重複與近似

**User Story:** As a User, I want to detect duplicate and near-duplicate rows, so that I can eliminate redundant data.

#### Acceptance Criteria

1. WHEN an assessment is triggered, THE Assessment_Engine SHALL detect exact duplicate rows by comparing full-row hash values, where exact_duplicate_count is the total number of excess duplicate rows (total duplicates minus one retained instance per unique row).
2. WHEN an assessment is triggered, THE Assessment_Engine SHALL detect near-duplicate values using Levenshtein_Distance with a threshold of 2 or fewer edits, applied only to text columns where the unique value count is between 5% and 80% of total rows. THE Assessment_Engine SHALL select eligible columns in left-to-right positional order, limited to a maximum of 5 columns. near_duplicate_group_count is the total number of distinct value-pair groups across all selected columns where Levenshtein_Distance between two unique values is ≤ 2.
3. WHEN an assessment is triggered, THE Assessment_Engine SHALL calculate the Duplicate/Similar score as: (1 - (exact_duplicate_count + near_duplicate_group_count × 0.5) / total_rows) × 100, with a minimum score of 0.
4. THE Assessment_Engine SHALL apply a default weight of 10% to the Duplicate/Similar score in the overall Readiness_Score calculation.
5. IF the dataset contains 0 data rows, THEN THE Assessment_Engine SHALL assign a Duplicate/Similar score of 100.

### Requirement 8: AI Data Readiness 評估 — 表格結構品質

**User Story:** As a User, I want to assess the structural quality of my spreadsheet, so that I can ensure it is machine-parsable.

#### Acceptance Criteria

1. WHEN an assessment is triggered, THE Assessment_Engine SHALL calculate Table Structure Quality starting from a base score of 100 and applying per-type deductions (each deduction applied at most once regardless of how many instances are found): merged cells detected (-20), multi-layer headers detected (-20), subtotal/total rows detected (-15), multiple tables in one sheet detected (-25), and notes mixed into data columns detected (-10).
2. WHEN the Assessment_Engine detects more than one row in the first 5 rows where all non-empty cells contain text values and no cell value is repeated within the same row, THE Assessment_Engine SHALL classify it as multi-layer headers.
3. WHEN the Assessment_Engine detects rows where any cell value contains the substrings "小計", "合計", "total", or "subtotal" (case-insensitive), THE Assessment_Engine SHALL classify them as subtotal/total rows.
4. WHEN the Assessment_Engine detects 2 or more consecutive rows where every cell is empty separating non-empty data blocks, THE Assessment_Engine SHALL classify it as multiple tables in one sheet.
5. WHEN the standard deviation of text length in a text-type column exceeds 3 times the column mean text length, and the column mean text length is greater than 0, THE Assessment_Engine SHALL classify it as notes mixed into data columns.
6. THE Assessment_Engine SHALL ensure the Table Structure Quality score has a minimum value of 0.
7. THE Assessment_Engine SHALL apply a default weight of 15% to the Table Structure Quality score in the overall Readiness_Score calculation.
8. WHEN the Assessment_Engine evaluates merged cells, THE Assessment_Engine SHALL use the merged cell locations recorded by the Upload_Service during file upload.

### Requirement 9: AI Data Readiness 評估 — AI 問答可用性

**User Story:** As a User, I want to know if my dataset has the structural elements needed for effective LLM-based question answering, so that I can predict AI analysis reliability.

#### Acceptance Criteria

1. WHEN an assessment is triggered, THE Assessment_Engine SHALL calculate AI Query Readiness by summing points for five sub-conditions (each worth 20 points, maximum total 100): identifier column exists (at least one column with unique value ratio > 80% after removing empty values), time column exists (at least one column where date parsing using the system-defined date formats succeeds for > 60% of the first min(100, total_rows) rows), category column exists (at least one column with unique value count < 20% of total rows and > 1), numeric column exists (at least one column where > 80% of non-empty values are parseable as numbers including integers, thousands-separated numbers, and decimals), and column name quality (all column names are non-empty after trimming whitespace, non-duplicate, and longer than 1 character).
2. THE Assessment_Engine SHALL apply a default weight of 20% to the AI Query Readiness score in the overall Readiness_Score calculation.
3. IF the dataset contains 0 data rows (excluding header), THEN THE Assessment_Engine SHALL assign an AI Query Readiness score of 0.
4. WHEN evaluating the column name quality sub-condition, THE Assessment_Engine SHALL treat column names that consist solely of whitespace characters as empty and fail the sub-condition.

### Requirement 10: 總分計算與分級

**User Story:** As a User, I want to see an overall readiness score and grade, so that I can quickly understand my data quality status.

#### Acceptance Criteria

1. WHEN all six indicator scores are calculated, THE Assessment_Engine SHALL compute the Readiness_Score as the weighted sum of the six indicators using the configured weights, round the result to one decimal place, and produce a value between 0.0 and 100.0.
2. IF the Readiness_Score is greater than or equal to 80.0, THEN THE Assessment_Engine SHALL classify it as "Ready".
3. IF the Readiness_Score is greater than or equal to 60.0 and less than 80.0, THEN THE Assessment_Engine SHALL classify it as "Conditionally Ready".
4. IF the Readiness_Score is less than 60.0, THEN THE Assessment_Engine SHALL classify it as "Not Ready".
5. WHEN the same file and sheet are assessed multiple times with the same weight configuration, THE Assessment_Engine SHALL produce identical scores.
6. WHEN an assessment is complete, THE Assessment_Engine SHALL output a problem list where each entry contains a severity level (High, Medium, or Low), the affected row count, and a recommended action description for the detected issue.
7. IF any one of the six indicator calculations fails due to an error, THEN THE Assessment_Engine SHALL halt the overall score computation and return an error indicating which indicator failed, without producing a partial Readiness_Score.
8. IF the configured weights do not sum to 100%, THEN THE Assessment_Engine SHALL reject the assessment request and return an error indicating the weight configuration is invalid.

### Requirement 11: 分流決策

**User Story:** As a User, I want to choose between cleaning my data as-is or re-uploading an improved version, so that I can proceed with the workflow that best fits my situation.

#### Acceptance Criteria

1. WHEN an assessment is complete, THE Frontend SHALL present two pathway options: "以現況梳理" (proceed to Cleaning_Engine) and "補齊後重新上傳" (return to Upload_Service).
2. WHEN the Readiness_Score is classified as "Not Ready" and the User selects "以現況梳理", THE Frontend SHALL display a risk warning but allow the User to proceed.
3. WHEN the User selects "補齊後重新上傳", THE System SHALL allow re-upload and re-run the full assessment process.

### Requirement 12: 以現況梳理 — 批次規則

**User Story:** As a User, I want to apply batch cleaning rules to my data, so that I can fix common data quality issues efficiently.

#### Acceptance Criteria

1. WHEN the User selects the "統一日期格式" rule, THE Cleaning_Engine SHALL convert all detected date values to yyyy-MM-dd format.
2. WHEN the User selects the "移除重複列" rule, THE Cleaning_Engine SHALL remove exact duplicate rows and retain only the first occurrence.
3. WHEN the User selects the "客戶名正規化" rule, THE Cleaning_Engine SHALL remove company suffix variants (Co., Company, 公司, 股份有限公司) for comparison and unify matching names to the longest variant.
4. WHEN the User selects the "移除小計列" rule, THE Cleaning_Engine SHALL remove rows containing the keywords "小計", "合計", "total", or "subtotal".
5. WHEN any cleaning rule is applied, THE Cleaning_Engine SHALL record the operation in the Cleaning_Log with operation type, affected row numbers, timestamp, and operator ID.

### Requirement 13: 以現況梳理 — 單列處理

**User Story:** As a User, I want to handle individual problematic rows, so that I can make targeted corrections to my data.

#### Acceptance Criteria

1. WHEN the User selects "填入 N/A" for a specific row, THE Cleaning_Engine SHALL fill all empty cells in that row with the text "N/A" and record the operation in the Cleaning_Log.
2. WHEN the User selects "刪除該列" for a specific row, THE Cleaning_Engine SHALL remove the row from the dataset and record the operation in the Cleaning_Log.

### Requirement 14: 產出檔案

**User Story:** As a User, I want to download the cleaned dataset, a quality report, and the cleaning log, so that I have complete documentation of the data improvement process.

#### Acceptance Criteria

1. WHEN data cleaning is complete, THE Export_Service SHALL generate a refined.xlsx file containing the cleaned dataset.
2. WHEN data cleaning is complete, THE Export_Service SHALL generate a report.pdf file containing: the Readiness_Score, six indicator scores with descriptions, problem summary, before-and-after comparison statistics, and cleaning rule summary, formatted with brand colors, clear layout, ring score charts, and bar charts.
3. WHEN data cleaning is complete, THE Export_Service SHALL generate a cleaning.log file in JSON format containing the complete cleaning operation history.
4. THE Export_Service SHALL make all three files available for download via dedicated API endpoints.

### Requirement 15: Evidence 存證 API

**User Story:** As a User, I want to submit evidence records of my data processing to a blockchain service, so that the integrity of my data cleaning process can be verified.

#### Acceptance Criteria

1. WHEN the User triggers evidence submission, THE Evidence_Service SHALL compute SHA-256 hashes of the refined.xlsx, cleaning.log, and report.pdf files.
2. WHEN hashes are computed, THE Evidence_Service SHALL call the external blockchain API via POST /api/evidence/submit with dataset_hash, cleaning_log_hash, report_hash, timestamp, tool_version, rule_version, operator_id, and metadata (original_filename, original_rows, refined_rows, readiness_before, readiness_after).
3. WHEN the blockchain API returns a response, THE Evidence_Service SHALL store the record_id and signature_status and display the evidence record to the User.
4. WHEN the User queries an evidence record, THE Evidence_Service SHALL call GET /api/evidence/{record_id} on the external blockchain API and return the result.
5. IF the external blockchain API is unavailable, THEN THE Evidence_Service SHALL return a clear error message indicating the blockchain service is unreachable without crashing the application.

### Requirement 16: 前後對比問答

**User Story:** As a User, I want to ask questions about my data before and after cleaning and see side-by-side answers, so that I can understand the impact of the cleaning process.

#### Acceptance Criteria

1. WHEN the User submits a question, THE QA_Service SHALL send the question along with structured CSV data snippets (original and cleaned) to the Gemini API and return the complete response (non-streaming).
2. WHEN a question involves a column with a missing value rate exceeding 50% in the original dataset, THE QA_Service SHALL return "資料不足" with an explanation of which data is missing, without calling the Gemini API.
3. WHEN a cleaning session is complete, THE QA_Service SHALL generate 3 suggested questions based on column names in the dataset.
4. THE Frontend SHALL display original-data answers and cleaned-data answers in a left-right side-by-side layout.
5. WHEN the User has not yet agreed to the data protection consent prompt, THE QA_Service SHALL block question submission and THE Frontend SHALL display a consent notice explaining that data will be sent to an external model.

### Requirement 17: 權重設定

**User Story:** As a User, I want to adjust the weights of the six assessment indicators, so that I can customize the scoring to match my priorities.

#### Acceptance Criteria

1. THE Frontend SHALL provide six slider controls for adjusting indicator weights, with a constraint that the sum of all weights equals 100%.
2. WHEN the User saves updated weights, THE System SHALL persist the new weights in the database and apply them to all subsequent assessments.
3. WHEN weights are changed, THE System SHALL preserve historical assessment reports with their original weight snapshots, ensuring past reports remain unaffected.

### Requirement 18: Docker 容器化部署

**User Story:** As a developer, I want all services to start with a single Docker Compose command, so that deployment is simple and reproducible.

#### Acceptance Criteria

1. THE System SHALL provide a Docker Compose configuration that starts all services (frontend, backend, database) with a single `docker compose up` command.
2. THE System SHALL persist uploaded Excel files on a Docker volume so that files survive container restarts.
3. THE System SHALL configure Nginx to serve the React frontend static files and reverse proxy /api/* requests to the Go backend.

### Requirement 19: RESTful API 設計

**User Story:** As a developer, I want the backend to follow RESTful API conventions, so that the API is predictable and easy to integrate with.

#### Acceptance Criteria

1. THE System SHALL expose all functionality through RESTful HTTP endpoints using appropriate HTTP methods (GET for retrieval, POST for creation, PUT for updates).
2. IF a request fails validation, THEN THE System SHALL return HTTP 400 with a JSON error body describing the validation failure.
3. IF a request targets a non-existent resource, THEN THE System SHALL return HTTP 404 with a JSON error body.
4. IF an unauthenticated request is made to a protected endpoint, THEN THE System SHALL return HTTP 401 with a JSON error body.
5. IF an internal error occurs, THEN THE System SHALL return HTTP 500 with a JSON error body that does not expose internal implementation details.

### Requirement 20: 繁體中文介面

**User Story:** As a User, I want the entire interface to be in Traditional Chinese, so that I can use the tool in my native language.

#### Acceptance Criteria

1. THE Frontend SHALL render all user-facing text, labels, error messages, and notifications in Traditional Chinese (繁體中文).
2. THE System SHALL return all API error messages and status descriptions in Traditional Chinese.
