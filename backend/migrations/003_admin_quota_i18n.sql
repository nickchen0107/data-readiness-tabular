-- Migration: 003_admin_quota_i18n
-- Description: Add role, quota, and translations support
-- Created: 2025-06-25

-- ============================================================
-- 1. Add role column to users table
-- ============================================================
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS role VARCHAR(10) NOT NULL DEFAULT 'user';

-- Add CHECK constraint (use DO block for idempotency)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'users_role_check'
    ) THEN
        ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('admin', 'user'));
    END IF;
END$$;

-- ============================================================
-- 2. Add last_quota_reset column to users table
-- ============================================================
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS last_quota_reset TIMESTAMP WITH TIME ZONE DEFAULT NOW();

-- ============================================================
-- 3. Create quota_settings table
-- ============================================================
CREATE TABLE IF NOT EXISTS quota_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    max_assessments INTEGER NOT NULL DEFAULT 5 CHECK (max_assessments >= 1),
    reset_period VARCHAR(10) NOT NULL DEFAULT 'daily' CHECK (reset_period IN ('daily', 'weekly')),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Insert default quota settings (skip if already exists)
INSERT INTO quota_settings (max_assessments, reset_period)
SELECT 5, 'daily'
WHERE NOT EXISTS (SELECT 1 FROM quota_settings LIMIT 1);

-- ============================================================
-- 4. Create translations table
-- ============================================================
CREATE TABLE IF NOT EXISTS translations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    locale VARCHAR(10) NOT NULL,
    key VARCHAR(255) NOT NULL,
    value TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Unique index on (locale, key)
CREATE UNIQUE INDEX IF NOT EXISTS idx_translations_locale_key ON translations(locale, key);

-- Index for locale-based lookups
CREATE INDEX IF NOT EXISTS idx_translations_locale ON translations(locale);

-- ============================================================
-- DOWN (manual rollback):
-- DROP INDEX IF EXISTS idx_translations_locale;
-- DROP INDEX IF EXISTS idx_translations_locale_key;
-- DROP TABLE IF EXISTS translations;
-- DROP TABLE IF EXISTS quota_settings;
-- ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
-- ALTER TABLE users DROP COLUMN IF EXISTS last_quota_reset;
-- ALTER TABLE users DROP COLUMN IF EXISTS role;
-- ============================================================
