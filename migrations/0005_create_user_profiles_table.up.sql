CREATE TABLE user_profiles (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    nis        VARCHAR(255),
    nip        VARCHAR(255),
    class      VARCHAR(255),
    major      VARCHAR(255),
    year       SMALLINT,
    phone      VARCHAR(255),
    address    TEXT,
    rombel_id  BIGINT,
    room_id    BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_user_profiles_user_id ON user_profiles (user_id);
CREATE INDEX idx_user_profiles_rombel_id ON user_profiles (rombel_id);
CREATE INDEX idx_user_profiles_room_id ON user_profiles (room_id);
