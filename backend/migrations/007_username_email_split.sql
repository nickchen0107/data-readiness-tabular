-- 007: Split email field into username + email
-- email column currently stores the login username (e.g. "nick", "test")
-- Rename it to username, then add a proper email column

-- Step 1: Rename email → username
ALTER TABLE users RENAME COLUMN email TO username;

-- Step 2: Rename the unique index
ALTER INDEX users_email_key RENAME TO users_username_key;

-- Step 3: Add email column (nullable, optional)
ALTER TABLE users ADD COLUMN email VARCHAR(255) DEFAULT NULL;

-- Step 4: Also rename in login_attempts
ALTER TABLE login_attempts RENAME COLUMN email TO username;
