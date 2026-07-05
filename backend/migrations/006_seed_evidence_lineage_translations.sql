-- Migration: 006_seed_evidence_lineage_translations
-- Description: Add evidence lineage i18n translations
-- Created: 2026-07-06

INSERT INTO translations (locale, key, value) VALUES
  ('zh-TW', 'evidence.record_id', '存證編號'),
  ('zh-TW', 'evidence.transaction_id', '交易 ID（鏈上證明）'),
  ('zh-TW', 'evidence.lineage_title', '資料溯源 (Data Lineage)'),
  ('zh-TW', 'evidence.lineage_desc', '記錄資料處理前後的雜湊與 IPFS 存證位址，確保資料可追溯'),
  ('zh-TW', 'evidence.artifact_raw_dataset', '原始資料（梳理後資料集）'),
  ('zh-TW', 'evidence.artifact_processed_report', '處理後報告（PDF 品質報告）'),
  ('zh-TW', 'evidence.artifact_cleaning_log', '清洗日誌'),
  ('zh-TW', 'evidence.flag_no_sensitive', '鏈上無敏感資料'),
  ('zh-TW', 'evidence.flag_verifiable', '完整性可驗證'),
  ('zh-TW', 'evidence.flag_immutable', '不可竄改'),
  ('en', 'evidence.record_id', 'Record ID'),
  ('en', 'evidence.transaction_id', 'Transaction ID (On-chain Proof)'),
  ('en', 'evidence.lineage_title', 'Data Lineage'),
  ('en', 'evidence.lineage_desc', 'Records pre/post-processing hashes and IPFS addresses for full data traceability'),
  ('en', 'evidence.artifact_raw_dataset', 'Raw Dataset (Cleaned Data)'),
  ('en', 'evidence.artifact_processed_report', 'Processed Report (PDF Quality Report)'),
  ('en', 'evidence.artifact_cleaning_log', 'Cleaning Log'),
  ('en', 'evidence.flag_no_sensitive', 'No sensitive data on-chain'),
  ('en', 'evidence.flag_verifiable', 'Integrity verifiable'),
  ('en', 'evidence.flag_immutable', 'Immutable record')
ON CONFLICT (locale, key) DO UPDATE SET value = EXCLUDED.value;
