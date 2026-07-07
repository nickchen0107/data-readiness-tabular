-- 008: Clear and reseed translations
-- The DB had incorrect translations from earlier seeds.
-- This migration clears all and lets the backend re-seed from the
-- correct JSON fallback files on next startup.
DELETE FROM translations;
