-- Drop existing email index if it exists and recreate as partial unique
DROP INDEX IF EXISTS idx_users_email;
CREATE UNIQUE INDEX idx_users_email_unique ON users (email) WHERE email IS NOT NULL AND deleted_at IS NULL;
