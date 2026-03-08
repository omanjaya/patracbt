CREATE TABLE user_rombels (
    user_id   BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rombel_id BIGINT NOT NULL REFERENCES rombels(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, rombel_id)
);
