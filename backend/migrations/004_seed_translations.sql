-- Migration: 004_seed_translations
-- Description: Seed default translation key-value pairs for zh-TW and en locales
-- Created: 2025-06-25
-- Requirements: 11.1, 11.2, 11.3, 11.4, 11.5, 11.6, 12.4

-- ============================================================
-- Stepper labels (步驟列)
-- ============================================================
INSERT INTO translations (locale, key, value) VALUES
  ('zh-TW', 'stepper.landing', '進入'),
  ('zh-TW', 'stepper.upload', '上傳'),
  ('zh-TW', 'stepper.assess', '評估'),
  ('zh-TW', 'stepper.route', '分流'),
  ('zh-TW', 'stepper.clean', '梳理'),
  ('zh-TW', 'stepper.export', '產出'),
  ('zh-TW', 'stepper.evidence', '存證'),
  ('zh-TW', 'stepper.qa', '問答')
ON CONFLICT (locale, key) DO NOTHING;

INSERT INTO translations (locale, key, value) VALUES
  ('en', 'stepper.landing', 'Landing'),
  ('en', 'stepper.upload', 'Upload'),
  ('en', 'stepper.assess', 'Assess'),
  ('en', 'stepper.route', 'Route'),
  ('en', 'stepper.clean', 'Clean'),
  ('en', 'stepper.export', 'Export'),
  ('en', 'stepper.evidence', 'Evidence'),
  ('en', 'stepper.qa', 'QA')
ON CONFLICT (locale, key) DO NOTHING;

-- ============================================================
-- Page titles (頁面標題)
-- ============================================================
INSERT INTO translations (locale, key, value) VALUES
  ('zh-TW', 'page.upload.title', '上傳檔案'),
  ('zh-TW', 'page.upload.desc', '拖曳或點擊選取 Excel / CSV 檔案，系統將解析後進行品質評估'),
  ('zh-TW', 'page.assessment.title', '品質評估結果'),
  ('zh-TW', 'page.cleaning.title', '資料梳理'),
  ('zh-TW', 'page.cleaning.desc', '選擇梳理規則並執行批次資料清理'),
  ('zh-TW', 'page.export.title', '梳理成果總覽'),
  ('zh-TW', 'page.export.download_title', '產出下載'),
  ('zh-TW', 'page.login.title', '登入'),
  ('zh-TW', 'page.login.desc', '請輸入您的帳號密碼以登入系統'),
  ('zh-TW', 'page.register.title', '註冊帳號'),
  ('zh-TW', 'page.register.desc', '建立新帳號以使用 SAFE-AI 資料梳理平台')
ON CONFLICT (locale, key) DO NOTHING;

INSERT INTO translations (locale, key, value) VALUES
  ('en', 'page.upload.title', 'Upload File'),
  ('en', 'page.upload.desc', 'Drag and drop or click to select an Excel / CSV file for quality assessment'),
  ('en', 'page.assessment.title', 'Quality Assessment Results'),
  ('en', 'page.cleaning.title', 'Data Cleaning'),
  ('en', 'page.cleaning.desc', 'Select cleaning rules and run batch data cleaning'),
  ('en', 'page.export.title', 'Cleaning Results Overview'),
  ('en', 'page.export.download_title', 'Export Downloads'),
  ('en', 'page.login.title', 'Login'),
  ('en', 'page.login.desc', 'Enter your credentials to access the system'),
  ('en', 'page.register.title', 'Register'),
  ('en', 'page.register.desc', 'Create a new account to use the SAFE-AI Data Brushing Platform')
ON CONFLICT (locale, key) DO NOTHING;

-- ============================================================
-- Button labels (按鈕標籤)
-- ============================================================
INSERT INTO translations (locale, key, value) VALUES
  ('zh-TW', 'btn.start_assess', '開始評估'),
  ('zh-TW', 'btn.next_step', '下一步'),
  ('zh-TW', 'btn.back_upload', '返回上傳'),
  ('zh-TW', 'btn.back_cleaning', '返回梳理步驟'),
  ('zh-TW', 'btn.reupload', '重新上傳檔案'),
  ('zh-TW', 'btn.logout', '登出'),
  ('zh-TW', 'btn.login', '登入'),
  ('zh-TW', 'btn.register', '註冊'),
  ('zh-TW', 'btn.run_clean', '執行梳理'),
  ('zh-TW', 'btn.download_xlsx', '下載梳理後資料'),
  ('zh-TW', 'btn.download_pdf', '下載品質報告'),
  ('zh-TW', 'btn.download_log', '下載梳理紀錄'),
  ('zh-TW', 'btn.go_register', '前往註冊'),
  ('zh-TW', 'btn.go_login', '前往登入'),
  ('zh-TW', 'btn.save', '儲存'),
  ('zh-TW', 'btn.cancel', '取消'),
  ('zh-TW', 'btn.confirm', '確認')
ON CONFLICT (locale, key) DO NOTHING;

INSERT INTO translations (locale, key, value) VALUES
  ('en', 'btn.start_assess', 'Start Assessment'),
  ('en', 'btn.next_step', 'Next'),
  ('en', 'btn.back_upload', 'Back to Upload'),
  ('en', 'btn.back_cleaning', 'Back to Cleaning'),
  ('en', 'btn.reupload', 'Re-upload File'),
  ('en', 'btn.logout', 'Logout'),
  ('en', 'btn.login', 'Login'),
  ('en', 'btn.register', 'Register'),
  ('en', 'btn.run_clean', 'Run Cleaning'),
  ('en', 'btn.download_xlsx', 'Download Cleaned Data'),
  ('en', 'btn.download_pdf', 'Download Quality Report'),
  ('en', 'btn.download_log', 'Download Cleaning Log'),
  ('en', 'btn.go_register', 'Go to Register'),
  ('en', 'btn.go_login', 'Go to Login'),
  ('en', 'btn.save', 'Save'),
  ('en', 'btn.cancel', 'Cancel'),
  ('en', 'btn.confirm', 'Confirm')
ON CONFLICT (locale, key) DO NOTHING;

-- ============================================================
-- Indicator names (指標名稱)
-- ============================================================
INSERT INTO translations (locale, key, value) VALUES
  ('zh-TW', 'indicator.row_completeness', '列完整度'),
  ('zh-TW', 'indicator.column_completeness', '欄完整度'),
  ('zh-TW', 'indicator.format_consistency', '格式一致性'),
  ('zh-TW', 'indicator.data_uniqueness', '資料唯一性'),
  ('zh-TW', 'indicator.table_structure', '表格結構'),
  ('zh-TW', 'indicator.ai_query_readiness', 'AI 問答可用性')
ON CONFLICT (locale, key) DO NOTHING;

INSERT INTO translations (locale, key, value) VALUES
  ('en', 'indicator.row_completeness', 'Row Completeness'),
  ('en', 'indicator.column_completeness', 'Column Completeness'),
  ('en', 'indicator.format_consistency', 'Format Consistency'),
  ('en', 'indicator.data_uniqueness', 'Data Uniqueness'),
  ('en', 'indicator.table_structure', 'Table Structure'),
  ('en', 'indicator.ai_query_readiness', 'AI Query Readiness')
ON CONFLICT (locale, key) DO NOTHING;

-- ============================================================
-- Indicator descriptions (指標說明)
-- ============================================================
INSERT INTO translations (locale, key, value) VALUES
  ('zh-TW', 'indicator.row_completeness.desc', '衡量每列資料的填寫比例'),
  ('zh-TW', 'indicator.column_completeness.desc', '衡量每欄資料的填寫比例'),
  ('zh-TW', 'indicator.format_consistency.desc', '衡量每欄內資料格式的統一程度'),
  ('zh-TW', 'indicator.data_uniqueness.desc', '依據 ISO/IEC 25024 衡量資料的完整度與唯一性'),
  ('zh-TW', 'indicator.table_structure.desc', '衡量表格結構是否乾淨規整'),
  ('zh-TW', 'indicator.ai_query_readiness.desc', '衡量資料結構是否具備 AI/ML 所需的 schema 品質')
ON CONFLICT (locale, key) DO NOTHING;

INSERT INTO translations (locale, key, value) VALUES
  ('en', 'indicator.row_completeness.desc', 'Measures the fill rate of each row'),
  ('en', 'indicator.column_completeness.desc', 'Measures the fill rate of each column'),
  ('en', 'indicator.format_consistency.desc', 'Measures format uniformity within each column'),
  ('en', 'indicator.data_uniqueness.desc', 'Measures data completeness and uniqueness per ISO/IEC 25024'),
  ('en', 'indicator.table_structure.desc', 'Measures whether the table structure is clean and well-organized'),
  ('en', 'indicator.ai_query_readiness.desc', 'Measures whether data structure meets AI/ML schema quality requirements')
ON CONFLICT (locale, key) DO NOTHING;

-- ============================================================
-- Status labels (狀態標籤)
-- ============================================================
INSERT INTO translations (locale, key, value) VALUES
  ('zh-TW', 'status.ready', 'AI Ready'),
  ('zh-TW', 'status.conditional', '有條件通過'),
  ('zh-TW', 'status.not_ready', '未就緒'),
  ('zh-TW', 'status.upload_complete', '上傳完成'),
  ('zh-TW', 'status.cleaning_complete', '梳理完成')
ON CONFLICT (locale, key) DO NOTHING;

INSERT INTO translations (locale, key, value) VALUES
  ('en', 'status.ready', 'AI Ready'),
  ('en', 'status.conditional', 'Conditional'),
  ('en', 'status.not_ready', 'Not Ready'),
  ('en', 'status.upload_complete', 'Upload Complete'),
  ('en', 'status.cleaning_complete', 'Cleaning Complete')
ON CONFLICT (locale, key) DO NOTHING;

-- ============================================================
-- Error messages (錯誤訊息)
-- ============================================================
INSERT INTO translations (locale, key, value) VALUES
  ('zh-TW', 'error.invalid_id', '無效的評估 ID'),
  ('zh-TW', 'error.assessment_not_found', '評估記錄不存在，請先上傳檔案'),
  ('zh-TW', 'error.load_assessment_failed', '載入評估結果失敗'),
  ('zh-TW', 'error.upload_failed', '上傳失敗，請稍後再試'),
  ('zh-TW', 'error.login_failed', '登入失敗，請檢查帳號密碼'),
  ('zh-TW', 'error.register_failed', '註冊失敗，請稍後再試'),
  ('zh-TW', 'error.password_mismatch', '兩次輸入的密碼不一致'),
  ('zh-TW', 'error.password_length', '密碼長度需介於 8 至 72 字元之間'),
  ('zh-TW', 'error.file_format', '僅支援 .xlsx 或 .csv 格式檔案'),
  ('zh-TW', 'error.cleaning_failed', '梳理執行失敗'),
  ('zh-TW', 'error.cleaning_not_found', '梳理記錄不存在，請先執行資料梳理'),
  ('zh-TW', 'error.no_cleaning_record', '找不到梳理記錄，請先執行資料梳理'),
  ('zh-TW', 'error.load_comparison_failed', '載入比較資料失敗'),
  ('zh-TW', 'error.download_failed', '下載失敗，請稍後再試'),
  ('zh-TW', 'error.cannot_start_assessment', '無法啟動評估'),
  ('zh-TW', 'error.preview_failed', '無法取得預覽資料'),
  ('zh-TW', 'error.quota_exceeded', '評估次數已用盡，請聯繫管理員'),
  ('zh-TW', 'error.forbidden', '權限不足'),
  ('zh-TW', 'error.no_access', '無權限存取'),
  ('zh-TW', 'error.invalid_token', '無效的認證令牌'),
  ('zh-TW', 'error.invalid_locale', '不支援的語系，僅支援 zh-TW 與 en'),
  ('zh-TW', 'error.translation_not_found', '翻譯項目不存在'),
  ('zh-TW', 'error.quota_validation', '配額設定無效：max_assessments 必須為正整數，reset_period 必須為 daily 或 weekly')
ON CONFLICT (locale, key) DO NOTHING;

INSERT INTO translations (locale, key, value) VALUES
  ('en', 'error.invalid_id', 'Invalid assessment ID'),
  ('en', 'error.assessment_not_found', 'Assessment record not found. Please upload a file first.'),
  ('en', 'error.load_assessment_failed', 'Failed to load assessment results'),
  ('en', 'error.upload_failed', 'Upload failed. Please try again later.'),
  ('en', 'error.login_failed', 'Login failed. Please check your credentials.'),
  ('en', 'error.register_failed', 'Registration failed. Please try again later.'),
  ('en', 'error.password_mismatch', 'Passwords do not match'),
  ('en', 'error.password_length', 'Password must be between 8 and 72 characters'),
  ('en', 'error.file_format', 'Only .xlsx and .csv files are supported'),
  ('en', 'error.cleaning_failed', 'Cleaning execution failed'),
  ('en', 'error.cleaning_not_found', 'Cleaning record not found. Please run data cleaning first.'),
  ('en', 'error.no_cleaning_record', 'No cleaning record found. Please run data cleaning first.'),
  ('en', 'error.load_comparison_failed', 'Failed to load comparison data'),
  ('en', 'error.download_failed', 'Download failed. Please try again later.'),
  ('en', 'error.cannot_start_assessment', 'Unable to start assessment'),
  ('en', 'error.preview_failed', 'Unable to load preview data'),
  ('en', 'error.quota_exceeded', 'Assessment quota exhausted. Please contact your administrator.'),
  ('en', 'error.forbidden', 'Access denied'),
  ('en', 'error.no_access', 'No permission to access'),
  ('en', 'error.invalid_token', 'Invalid authentication token'),
  ('en', 'error.invalid_locale', 'Unsupported locale. Only zh-TW and en are supported.'),
  ('en', 'error.translation_not_found', 'Translation entry not found'),
  ('en', 'error.quota_validation', 'Invalid quota settings: max_assessments must be a positive integer, reset_period must be daily or weekly')
ON CONFLICT (locale, key) DO NOTHING;

-- ============================================================
-- Issue titles (問題標題)
-- ============================================================
INSERT INTO translations (locale, key, value) VALUES
  ('zh-TW', 'issue.data_gap', '資料大量缺漏'),
  ('zh-TW', 'issue.format_mix', '格式混用'),
  ('zh-TW', 'issue.table_structure', '表格結構問題'),
  ('zh-TW', 'issue.duplicate_rows', '重複資料列'),
  ('zh-TW', 'issue.merged_cells', '合併儲存格'),
  ('zh-TW', 'issue.subtotal_rows', '含小計/合計列'),
  ('zh-TW', 'issue.multi_table', '多表格結構'),
  ('zh-TW', 'issue.empty_columns', '高度空缺欄位'),
  ('zh-TW', 'issue.name_variants', '名稱不一致'),
  ('zh-TW', 'issue.newline_in_cell', '儲存格內換行'),
  ('zh-TW', 'issue.bracket_notes', '中文括號備註')
ON CONFLICT (locale, key) DO NOTHING;

INSERT INTO translations (locale, key, value) VALUES
  ('en', 'issue.data_gap', 'Significant Data Gaps'),
  ('en', 'issue.format_mix', 'Mixed Formats'),
  ('en', 'issue.table_structure', 'Table Structure Issues'),
  ('en', 'issue.duplicate_rows', 'Duplicate Rows'),
  ('en', 'issue.merged_cells', 'Merged Cells'),
  ('en', 'issue.subtotal_rows', 'Subtotal/Total Rows'),
  ('en', 'issue.multi_table', 'Multiple Table Sections'),
  ('en', 'issue.empty_columns', 'Highly Empty Columns'),
  ('en', 'issue.name_variants', 'Inconsistent Naming'),
  ('en', 'issue.newline_in_cell', 'Newlines in Cells'),
  ('en', 'issue.bracket_notes', 'Parenthetical Notes')
ON CONFLICT (locale, key) DO NOTHING;

-- ============================================================
-- Common UI strings (共用介面文字)
-- ============================================================
INSERT INTO translations (locale, key, value) VALUES
  ('zh-TW', 'common.loading', '載入中'),
  ('zh-TW', 'common.downloading', '下載中'),
  ('zh-TW', 'common.rows_affected', '列受影響'),
  ('zh-TW', 'common.minutes', '分'),
  ('zh-TW', 'common.score', '分'),
  ('zh-TW', 'common.rows', '列'),
  ('zh-TW', 'common.columns', '欄'),
  ('zh-TW', 'common.upload_progress', '上傳中...'),
  ('zh-TW', 'common.assessing', '評估中，正在分析資料品質...'),
  ('zh-TW', 'common.cleaning_progress', '梳理執行中...'),
  ('zh-TW', 'common.loading_assessment', '載入評估結果中...'),
  ('zh-TW', 'common.loading_comparison', '載入比較資料中...'),
  ('zh-TW', 'common.login_progress', '登入中...'),
  ('zh-TW', 'common.register_progress', '註冊中...'),
  ('zh-TW', 'common.running', '執行中...'),
  ('zh-TW', 'common.show_first_n', '僅顯示前 5 筆，更多問題列請至梳理步驟查看'),
  ('zh-TW', 'common.select_sheet', '選擇工作表：'),
  ('zh-TW', 'common.ready_for_assess', '選定工作表後將進行 AI Readiness 品質評估'),
  ('zh-TW', 'common.after_clean_hint', '梳理完成，可進入下一步產出檔案'),
  ('zh-TW', 'common.select_rules_hint', '選擇規則後點擊執行'),
  ('zh-TW', 'common.total_score', '總分')
ON CONFLICT (locale, key) DO NOTHING;

INSERT INTO translations (locale, key, value) VALUES
  ('en', 'common.loading', 'Loading'),
  ('en', 'common.downloading', 'Downloading'),
  ('en', 'common.rows_affected', 'rows affected'),
  ('en', 'common.minutes', 'min'),
  ('en', 'common.score', 'pts'),
  ('en', 'common.rows', 'rows'),
  ('en', 'common.columns', 'columns'),
  ('en', 'common.upload_progress', 'Uploading...'),
  ('en', 'common.assessing', 'Assessing data quality...'),
  ('en', 'common.cleaning_progress', 'Running cleaning...'),
  ('en', 'common.loading_assessment', 'Loading assessment results...'),
  ('en', 'common.loading_comparison', 'Loading comparison data...'),
  ('en', 'common.login_progress', 'Logging in...'),
  ('en', 'common.register_progress', 'Registering...'),
  ('en', 'common.running', 'Running...'),
  ('en', 'common.show_first_n', 'Showing first 5 entries only. See more in the Cleaning step.'),
  ('en', 'common.select_sheet', 'Select worksheet:'),
  ('en', 'common.ready_for_assess', 'AI Readiness quality assessment will begin after selecting a worksheet'),
  ('en', 'common.after_clean_hint', 'Cleaning complete. Proceed to export.'),
  ('en', 'common.select_rules_hint', 'Select rules then click Run'),
  ('en', 'common.total_score', 'Total Score')
ON CONFLICT (locale, key) DO NOTHING;

-- ============================================================
-- Form labels (表單標籤)
-- ============================================================
INSERT INTO translations (locale, key, value) VALUES
  ('zh-TW', 'form.email', '帳號'),
  ('zh-TW', 'form.password', '密碼'),
  ('zh-TW', 'form.confirm_password', '確認密碼'),
  ('zh-TW', 'form.email_placeholder', '請輸入帳號'),
  ('zh-TW', 'form.password_placeholder', '請輸入密碼'),
  ('zh-TW', 'form.register_email_placeholder', '請輸入帳號（3 字元以上）'),
  ('zh-TW', 'form.register_password_placeholder', '請輸入密碼（8-72 字元）'),
  ('zh-TW', 'form.confirm_password_placeholder', '請再次輸入密碼'),
  ('zh-TW', 'form.has_account', '已有帳號？'),
  ('zh-TW', 'form.no_account', '還沒有帳號？')
ON CONFLICT (locale, key) DO NOTHING;

INSERT INTO translations (locale, key, value) VALUES
  ('en', 'form.email', 'Email'),
  ('en', 'form.password', 'Password'),
  ('en', 'form.confirm_password', 'Confirm Password'),
  ('en', 'form.email_placeholder', 'Enter your email'),
  ('en', 'form.password_placeholder', 'Enter your password'),
  ('en', 'form.register_email_placeholder', 'Enter your email (3+ characters)'),
  ('en', 'form.register_password_placeholder', 'Enter password (8-72 characters)'),
  ('en', 'form.confirm_password_placeholder', 'Re-enter your password'),
  ('en', 'form.has_account', 'Already have an account?'),
  ('en', 'form.no_account', 'Don''t have an account?')
ON CONFLICT (locale, key) DO NOTHING;

-- ============================================================
-- Upload page specific (上傳頁面)
-- ============================================================
INSERT INTO translations (locale, key, value) VALUES
  ('zh-TW', 'upload.drop_title', '拖曳檔案至此處'),
  ('zh-TW', 'upload.drop_desc', '或點擊選取檔案（支援 .xlsx、.csv，上限 50MB）'),
  ('zh-TW', 'upload.file_info', '{{rows}} 列 × {{cols}} 欄')
ON CONFLICT (locale, key) DO NOTHING;

INSERT INTO translations (locale, key, value) VALUES
  ('en', 'upload.drop_title', 'Drop file here'),
  ('en', 'upload.drop_desc', 'Or click to select a file (.xlsx, .csv, max 50MB)'),
  ('en', 'upload.file_info', '{{rows}} rows × {{cols}} columns')
ON CONFLICT (locale, key) DO NOTHING;

-- ============================================================
-- Cleaning rules (梳理規則)
-- ============================================================
INSERT INTO translations (locale, key, value) VALUES
  ('zh-TW', 'rule.date_normalize', '統一日期格式'),
  ('zh-TW', 'rule.date_normalize.desc', '將各種日期寫法統一為 yyyy-MM-dd'),
  ('zh-TW', 'rule.dedup', '移除重複列'),
  ('zh-TW', 'rule.dedup.desc', '刪除完全相同的資料列'),
  ('zh-TW', 'rule.name_normalize', '客戶名正規化'),
  ('zh-TW', 'rule.name_normalize.desc', '統一公司名稱的不同寫法為最常用版本'),
  ('zh-TW', 'rule.subtotal_remove', '移除小計列'),
  ('zh-TW', 'rule.subtotal_remove.desc', '刪除含「小計」「合計」的非資料列'),
  ('zh-TW', 'rule.newline_remove', '移除儲存格內換行'),
  ('zh-TW', 'rule.newline_remove.desc', '將儲存格內的換行符號替換為空格'),
  ('zh-TW', 'rule.bracket_note_remove', '移除中文括號備註'),
  ('zh-TW', 'rule.bracket_note_remove.desc', '刪除儲存格內的中文括號備註內容'),
  ('zh-TW', 'rule.empty_row_remove', '移除全空列'),
  ('zh-TW', 'rule.empty_row_remove.desc', '刪除所有欄位都為空的資料列'),
  ('zh-TW', 'rule.multi_table_keep_main', '移除多餘資料區塊'),
  ('zh-TW', 'rule.multi_table_keep_main.desc', '保留最大的連續資料區塊，移除其他被空白列隔開的段落'),
  ('zh-TW', 'rule.empty_col_remove', '移除高度空缺欄位'),
  ('zh-TW', 'rule.empty_col_remove.desc', '移除超過 80% 為空值的欄位，提升資料完整度')
ON CONFLICT (locale, key) DO NOTHING;

INSERT INTO translations (locale, key, value) VALUES
  ('en', 'rule.date_normalize', 'Normalize Dates'),
  ('en', 'rule.date_normalize.desc', 'Unify various date formats to yyyy-MM-dd'),
  ('en', 'rule.dedup', 'Remove Duplicates'),
  ('en', 'rule.dedup.desc', 'Delete identical duplicate rows'),
  ('en', 'rule.name_normalize', 'Normalize Names'),
  ('en', 'rule.name_normalize.desc', 'Unify different spellings of company names to the most common version'),
  ('en', 'rule.subtotal_remove', 'Remove Subtotals'),
  ('en', 'rule.subtotal_remove.desc', 'Delete non-data rows containing subtotals or totals'),
  ('en', 'rule.newline_remove', 'Remove Cell Newlines'),
  ('en', 'rule.newline_remove.desc', 'Replace newline characters in cells with spaces'),
  ('en', 'rule.bracket_note_remove', 'Remove Parenthetical Notes'),
  ('en', 'rule.bracket_note_remove.desc', 'Remove bracketed annotation content from cells'),
  ('en', 'rule.empty_row_remove', 'Remove Empty Rows'),
  ('en', 'rule.empty_row_remove.desc', 'Delete rows where all columns are empty'),
  ('en', 'rule.multi_table_keep_main', 'Remove Extra Data Blocks'),
  ('en', 'rule.multi_table_keep_main.desc', 'Keep the largest contiguous data block, remove sections separated by blank rows'),
  ('en', 'rule.empty_col_remove', 'Remove Highly Empty Columns'),
  ('en', 'rule.empty_col_remove.desc', 'Remove columns with over 80% empty values to improve data completeness')
ON CONFLICT (locale, key) DO NOTHING;

-- ============================================================
-- Export page (產出頁面)
-- ============================================================
INSERT INTO translations (locale, key, value) VALUES
  ('zh-TW', 'export.refined_data', '梳理後資料'),
  ('zh-TW', 'export.refined_data.desc', '清理完成的 Excel 檔案'),
  ('zh-TW', 'export.quality_report', '品質報告'),
  ('zh-TW', 'export.quality_report.desc', '含圖表的品質評估報告'),
  ('zh-TW', 'export.cleaning_log', '梳理紀錄'),
  ('zh-TW', 'export.cleaning_log.desc', '所有操作的文字紀錄'),
  ('zh-TW', 'export.indicator_improvement', '六項指標改善'),
  ('zh-TW', 'export.issue_status', '問題解決狀態'),
  ('zh-TW', 'export.resolved_issues', '已修正的問題'),
  ('zh-TW', 'export.remaining_issues', '尚待解決的問題'),
  ('zh-TW', 'export.all_resolved', '所有問題已全部修正 🎉'),
  ('zh-TW', 'export.no_resolved', '本次梳理未解決任何問題'),
  ('zh-TW', 'export.fixed_label', '已修正'),
  ('zh-TW', 'export.before_clean', '梳理前'),
  ('zh-TW', 'export.after_clean', '梳理後')
ON CONFLICT (locale, key) DO NOTHING;

INSERT INTO translations (locale, key, value) VALUES
  ('en', 'export.refined_data', 'Cleaned Data'),
  ('en', 'export.refined_data.desc', 'The cleaned Excel file'),
  ('en', 'export.quality_report', 'Quality Report'),
  ('en', 'export.quality_report.desc', 'Quality assessment report with charts'),
  ('en', 'export.cleaning_log', 'Cleaning Log'),
  ('en', 'export.cleaning_log.desc', 'Text log of all operations'),
  ('en', 'export.indicator_improvement', 'Indicator Improvements'),
  ('en', 'export.issue_status', 'Issue Resolution Status'),
  ('en', 'export.resolved_issues', 'Resolved Issues'),
  ('en', 'export.remaining_issues', 'Remaining Issues'),
  ('en', 'export.all_resolved', 'All issues have been resolved 🎉'),
  ('en', 'export.no_resolved', 'No issues were resolved in this cleaning pass'),
  ('en', 'export.fixed_label', 'Fixed'),
  ('en', 'export.before_clean', 'Before Cleaning'),
  ('en', 'export.after_clean', 'After Cleaning')
ON CONFLICT (locale, key) DO NOTHING;

-- ============================================================
-- Assessment page (評估頁面)
-- ============================================================
INSERT INTO translations (locale, key, value) VALUES
  ('zh-TW', 'assessment.six_indicators', '六項評估指標'),
  ('zh-TW', 'assessment.issue_list', '問題清單'),
  ('zh-TW', 'assessment.ai_ready_hint', '可直接進入 AI 應用'),
  ('zh-TW', 'assessment.conditional_hint', '建議先梳理再進入 AI 應用'),
  ('zh-TW', 'assessment.not_ready_hint', '目前不建議直接進入 AI 應用'),
  ('zh-TW', 'assessment.score_after_clean', '梳理後品質'),
  ('zh-TW', 'assessment.data_rows', '資料列數'),
  ('zh-TW', 'assessment.batch_rules', '批次規則'),
  ('zh-TW', 'assessment.confirm_removal', '確認要移除的項目'),
  ('zh-TW', 'assessment.empty_col_hint', '高度空缺欄位（紅色欄位建議移除，點擊表頭切換）'),
  ('zh-TW', 'assessment.data_blocks_hint', '資料區塊（選擇要保留的區塊，其他將被移除）'),
  ('zh-TW', 'assessment.recommend_keep', '推薦保留')
ON CONFLICT (locale, key) DO NOTHING;

INSERT INTO translations (locale, key, value) VALUES
  ('en', 'assessment.six_indicators', 'Six Quality Indicators'),
  ('en', 'assessment.issue_list', 'Issue List'),
  ('en', 'assessment.ai_ready_hint', 'Ready for AI applications'),
  ('en', 'assessment.conditional_hint', 'Recommended to clean before AI applications'),
  ('en', 'assessment.not_ready_hint', 'Not recommended for direct AI application use'),
  ('en', 'assessment.score_after_clean', 'Post-cleaning quality'),
  ('en', 'assessment.data_rows', 'Data rows'),
  ('en', 'assessment.batch_rules', 'Batch Rules'),
  ('en', 'assessment.confirm_removal', 'Confirm items to remove'),
  ('en', 'assessment.empty_col_hint', 'Highly empty columns (red columns recommended for removal, click header to toggle)'),
  ('en', 'assessment.data_blocks_hint', 'Data blocks (select block to keep, others will be removed)'),
  ('en', 'assessment.recommend_keep', 'Recommended')
ON CONFLICT (locale, key) DO NOTHING;

-- ============================================================
-- Header & navigation (標頭與導航)
-- ============================================================
INSERT INTO translations (locale, key, value) VALUES
  ('zh-TW', 'header.platform_name', 'SAFE-AI 資料梳理平台'),
  ('zh-TW', 'header.tool_version', 'Excel 梳理小工具 v0.1'),
  ('zh-TW', 'header.admin', '管理後台'),
  ('zh-TW', 'nav.stepper_disabled', '請先完成前面的步驟')
ON CONFLICT (locale, key) DO NOTHING;

INSERT INTO translations (locale, key, value) VALUES
  ('en', 'header.platform_name', 'SAFE-AI Data Brushing Platform'),
  ('en', 'header.tool_version', 'Excel Brushing Tool v0.1'),
  ('en', 'header.admin', 'Admin Panel'),
  ('en', 'nav.stepper_disabled', 'Please complete the previous steps first')
ON CONFLICT (locale, key) DO NOTHING;

-- ============================================================
-- Admin pages (管理後台)
-- ============================================================
INSERT INTO translations (locale, key, value) VALUES
  ('zh-TW', 'admin.users', '使用者管理'),
  ('zh-TW', 'admin.quota', '配額設定'),
  ('zh-TW', 'admin.translations', '翻譯編輯器'),
  ('zh-TW', 'admin.records', '評估記錄'),
  ('zh-TW', 'admin.max_assessments', '最大評估次數'),
  ('zh-TW', 'admin.reset_period', '重置週期'),
  ('zh-TW', 'admin.daily', '每日'),
  ('zh-TW', 'admin.weekly', '每週'),
  ('zh-TW', 'admin.used_quota', '已使用配額'),
  ('zh-TW', 'admin.remaining_quota', '剩餘配額'),
  ('zh-TW', 'admin.search_placeholder', '搜尋翻譯鍵或值...'),
  ('zh-TW', 'admin.locale_filter', '語系篩選'),
  ('zh-TW', 'admin.save_success', '儲存成功'),
  ('zh-TW', 'admin.save_failed', '儲存失敗')
ON CONFLICT (locale, key) DO NOTHING;

INSERT INTO translations (locale, key, value) VALUES
  ('en', 'admin.users', 'User Management'),
  ('en', 'admin.quota', 'Quota Settings'),
  ('en', 'admin.translations', 'Translation Editor'),
  ('en', 'admin.records', 'Assessment Records'),
  ('en', 'admin.max_assessments', 'Max Assessments'),
  ('en', 'admin.reset_period', 'Reset Period'),
  ('en', 'admin.daily', 'Daily'),
  ('en', 'admin.weekly', 'Weekly'),
  ('en', 'admin.used_quota', 'Used Quota'),
  ('en', 'admin.remaining_quota', 'Remaining Quota'),
  ('en', 'admin.search_placeholder', 'Search translation key or value...'),
  ('en', 'admin.locale_filter', 'Locale Filter'),
  ('en', 'admin.save_success', 'Saved successfully'),
  ('en', 'admin.save_failed', 'Save failed')
ON CONFLICT (locale, key) DO NOTHING;

-- ============================================================
-- Severity labels (嚴重程度標籤)
-- ============================================================
INSERT INTO translations (locale, key, value) VALUES
  ('zh-TW', 'severity.high', '嚴重'),
  ('zh-TW', 'severity.medium', '中等'),
  ('zh-TW', 'severity.low', '輕微')
ON CONFLICT (locale, key) DO NOTHING;

INSERT INTO translations (locale, key, value) VALUES
  ('en', 'severity.high', 'High'),
  ('en', 'severity.medium', 'Medium'),
  ('en', 'severity.low', 'Low')
ON CONFLICT (locale, key) DO NOTHING;

-- ============================================================
-- Cleaning results (梳理結果)
-- ============================================================
INSERT INTO translations (locale, key, value) VALUES
  ('zh-TW', 'clean.complete', '梳理完成'),
  ('zh-TW', 'clean.row_count', '資料列數'),
  ('zh-TW', 'clean.quality_score', '品質分數'),
  ('zh-TW', 'clean.remove_label', '移除'),
  ('zh-TW', 'clean.keep_label', '保留'),
  ('zh-TW', 'clean.empty_rate', '空值'),
  ('zh-TW', 'clean.merged_cell_label', '合併儲存格')
ON CONFLICT (locale, key) DO NOTHING;

INSERT INTO translations (locale, key, value) VALUES
  ('en', 'clean.complete', 'Cleaning Complete'),
  ('en', 'clean.row_count', 'Row Count'),
  ('en', 'clean.quality_score', 'Quality Score'),
  ('en', 'clean.remove_label', 'Remove'),
  ('en', 'clean.keep_label', 'Keep'),
  ('en', 'clean.empty_rate', 'Empty'),
  ('en', 'clean.merged_cell_label', 'Merged Cell')
ON CONFLICT (locale, key) DO NOTHING;

-- ============================================================
-- Row distribution labels (列分佈標籤)
-- ============================================================
INSERT INTO translations (locale, key, value) VALUES
  ('zh-TW', 'distribution.high_readiness', 'High readiness'),
  ('zh-TW', 'distribution.medium', 'Medium'),
  ('zh-TW', 'distribution.low', 'Low')
ON CONFLICT (locale, key) DO NOTHING;

INSERT INTO translations (locale, key, value) VALUES
  ('en', 'distribution.high_readiness', 'High readiness'),
  ('en', 'distribution.medium', 'Medium'),
  ('en', 'distribution.low', 'Low')
ON CONFLICT (locale, key) DO NOTHING;

-- ============================================================
-- Misc / Tooltips
-- ============================================================
INSERT INTO translations (locale, key, value) VALUES
  ('zh-TW', 'tooltip.quota_exhausted', '評估次數已用盡，請聯繫管理員'),
  ('zh-TW', 'misc.ten_thousand', '萬'),
  ('zh-TW', 'misc.calculation', '計算'),
  ('zh-TW', 'misc.reference', '依據')
ON CONFLICT (locale, key) DO NOTHING;

INSERT INTO translations (locale, key, value) VALUES
  ('en', 'tooltip.quota_exhausted', 'Assessment quota exhausted. Please contact your administrator.'),
  ('en', 'misc.ten_thousand', '0k'),
  ('en', 'misc.calculation', 'Calculation'),
  ('en', 'misc.reference', 'Reference')
ON CONFLICT (locale, key) DO NOTHING;

-- ============================================================
-- END OF SEED
-- Total: ~100 key-value pairs per locale (200+ rows total)
-- ============================================================
