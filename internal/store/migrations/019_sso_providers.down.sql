-- Remove SSO-related columns from users
DROP INDEX IF EXISTS idx_users_sso;
ALTER TABLE users DROP COLUMN IF EXISTS sso_subject_id;
ALTER TABLE users DROP COLUMN IF EXISTS sso_provider_id;

-- Drop SSO sessions table
DROP TABLE IF EXISTS sso_sessions;

-- Drop SSO providers table
DROP TABLE IF EXISTS sso_providers;
