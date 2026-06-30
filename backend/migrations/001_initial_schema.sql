-- Migration: 001_initial_schema
-- Description: Create initial database schema for SAFE-AI Excel 梳理小工具
-- Created: 2025-01-01

-- ============================================================
-- users table
-- ============================================================
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ============================================================
-- uploads table
-- ============================================================
CREATE TABLE IF NOT EXISTS uploads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    filename VARCHAR(512) NOT NULL,
    file_path VARCHAR(1024) NOT NULL,
    file_size BIGINT NOT NULL,
    row_count INT,
    col_count INT,
    selected_sheet VARCHAR(255),
    sheet_names JSONB,
    merged_cells JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ============================================================
-- assessments table
-- ============================================================
CREATE TABLE IF NOT EXISTS assessments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    upload_id UUID NOT NULL REFERENCES uploads(id),
    total_score FLOAT,
    row_completeness FLOAT,
    column_completeness FLOAT,
    format_consistency FLOAT,
    duplicate_similar FLOAT,
    table_structure FLOAT,
    ai_query_readiness FLOAT,
    weights_snapshot JSONB NOT NULL,
    status VARCHAR(20) NOT NULL,
    issues JSONB,
    column_details JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ============================================================
-- cleaning_sessions table
-- ============================================================
CREATE TABLE IF NOT EXISTS cleaning_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assessment_id UUID NOT NULL REFERENCES assessments(id),
    user_id UUID NOT NULL REFERENCES users(id),
    rules_applied JSONB,
    rows_before INT,
    rows_after INT,
    score_before FLOAT,
    score_after FLOAT,
    cleaning_log JSONB,
    refined_file_path VARCHAR(1024),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ============================================================
-- evidence_records table
-- ============================================================
CREATE TABLE IF NOT EXISTS evidence_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cleaning_session_id UUID NOT NULL REFERENCES cleaning_sessions(id),
    dataset_hash VARCHAR(64) NOT NULL,
    log_hash VARCHAR(64) NOT NULL,
    report_hash VARCHAR(64) NOT NULL,
    record_id VARCHAR(255),
    transaction_hash VARCHAR(255),
    signature_status VARCHAR(20) DEFAULT 'pending',
    verification_url VARCHAR(1024),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ============================================================
-- system_settings table
-- ============================================================
CREATE TABLE IF NOT EXISTS system_settings (
    key VARCHAR(255) PRIMARY KEY,
    value JSONB NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_by UUID REFERENCES users(id)
);

-- ============================================================
-- login_attempts table (for rate limiting)
-- ============================================================
CREATE TABLE IF NOT EXISTS login_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL,
    success BOOLEAN NOT NULL DEFAULT false,
    attempted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ============================================================
-- Indexes
-- ============================================================

-- Index for login_attempts rate limiting query
CREATE INDEX IF NOT EXISTS idx_login_attempts_email_time ON login_attempts(email, attempted_at);

-- Index for uploads by user
CREATE INDEX IF NOT EXISTS idx_uploads_user_id ON uploads(user_id);

-- Index for assessments by upload
CREATE INDEX IF NOT EXISTS idx_assessments_upload_id ON assessments(upload_id);

-- ============================================================
-- Default data
-- ============================================================

-- Insert default weight settings (20/20/15/10/15/20)
INSERT INTO system_settings (key, value) VALUES (
    'assessment_weights',
    '{"row_completeness": 0.20, "column_completeness": 0.20, "format_consistency": 0.15, "duplicate_similar": 0.10, "table_structure": 0.15, "ai_query_readiness": 0.20}'
) ON CONFLICT (key) DO NOTHING;
