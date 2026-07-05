-- Migration: 005_t3_integration
-- Description: Add T3 TrustChain integration tables and update evidence_records
-- Created: 2026-07-06

-- ============================================================
-- T3 使用者帳號對應表
-- ============================================================
CREATE TABLE IF NOT EXISTS t3_user_mapping (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    local_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    t3_username VARCHAR(255) NOT NULL,
    t3_password_encrypted TEXT NOT NULL,
    t3_token TEXT,
    t3_token_expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(local_user_id)
);

CREATE INDEX IF NOT EXISTS idx_t3_user_mapping_local_user ON t3_user_mapping(local_user_id);

-- ============================================================
-- 更新 evidence_records 加入 T3 相關欄位
-- ============================================================
ALTER TABLE evidence_records
    ADD COLUMN IF NOT EXISTS t3_file_id VARCHAR(255),
    ADD COLUMN IF NOT EXISTS t3_cid VARCHAR(255),
    ADD COLUMN IF NOT EXISTS t3_token_id VARCHAR(255),
    ADD COLUMN IF NOT EXISTS t3_tx_id VARCHAR(255),
    ADD COLUMN IF NOT EXISTS t3_minted_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN IF NOT EXISTS t3_metadata JSONB;
