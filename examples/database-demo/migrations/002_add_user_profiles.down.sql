-- Migration rollback: Add user profiles
-- Version: 002
-- Description: Remove user profiles and extended user fields

-- Drop views
DROP VIEW IF EXISTS user_profiles_complete;

-- Drop tables
DROP TABLE IF EXISTS user_sessions;
DROP TABLE IF EXISTS user_profiles;

-- Remove columns from users table
ALTER TABLE users DROP COLUMN IF EXISTS timezone;
ALTER TABLE users DROP COLUMN IF EXISTS birth_date;
ALTER TABLE users DROP COLUMN IF EXISTS location;
ALTER TABLE users DROP COLUMN IF EXISTS website_url;
ALTER TABLE users DROP COLUMN IF EXISTS avatar_url;
ALTER TABLE users DROP COLUMN IF EXISTS bio;