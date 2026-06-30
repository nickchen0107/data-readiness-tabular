-- Migration: 002_add_original_filename
-- Description: Add original_filename column to cleaning_sessions table
-- Created: 2025-01-15

-- ============================================================
-- UP: Add original_filename column
-- ============================================================
-- Stores the user's original uploaded filename so the export can
-- produce a download name like "{original}_refined.xlsx".
-- Historical sessions will have NULL, which triggers the fallback
-- behavior of using "refined.xlsx" as the download filename.
ALTER TABLE cleaning_sessions
    ADD COLUMN IF NOT EXISTS original_filename VARCHAR(255) NULL;

-- ============================================================
-- DOWN (manual rollback):
-- ALTER TABLE cleaning_sessions DROP COLUMN IF EXISTS original_filename;
-- ============================================================
