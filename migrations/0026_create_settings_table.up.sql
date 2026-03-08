CREATE TABLE settings (
    id         BIGSERIAL PRIMARY KEY,
    key        VARCHAR(255) NOT NULL,
    value      TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_settings_key ON settings (key);
