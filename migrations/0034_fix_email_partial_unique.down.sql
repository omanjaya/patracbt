DROP INDEX IF EXISTS idx_users_email_unique;
CREATE INDEX idx_users_email ON users (email);
