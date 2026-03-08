CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    name          VARCHAR(255) NOT NULL,
    username      VARCHAR(255) NOT NULL,
    email         VARCHAR(255),
    password      VARCHAR(255) NOT NULL,
    role          VARCHAR(50) NOT NULL DEFAULT 'peserta',
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    avatar_path   VARCHAR(255),
    login_token   VARCHAR(255),
    last_login_at TIMESTAMPTZ,
    deleted_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_users_username ON users (username);
CREATE UNIQUE INDEX idx_users_email ON users (email);
CREATE INDEX idx_users_login_token ON users (login_token);
CREATE INDEX idx_users_deleted_at ON users (deleted_at);
