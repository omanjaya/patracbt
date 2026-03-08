CREATE TABLE questions (
    id               BIGSERIAL PRIMARY KEY,
    question_bank_id BIGINT NOT NULL REFERENCES question_banks(id) ON DELETE CASCADE,
    stimulus_id      BIGINT REFERENCES stimuli(id) ON DELETE SET NULL,
    question_type    VARCHAR(50) NOT NULL,
    body             TEXT NOT NULL,
    score            DOUBLE PRECISION NOT NULL DEFAULT 1,
    difficulty       VARCHAR(50) NOT NULL DEFAULT 'medium',
    options          JSONB,
    correct_answer   JSONB,
    audio_path       VARCHAR(255),
    audio_limit      INTEGER NOT NULL DEFAULT 2,
    bloom_level      INTEGER NOT NULL DEFAULT 0,
    topic_code       VARCHAR(100) NOT NULL DEFAULT '',
    order_index      INTEGER NOT NULL DEFAULT 0,
    deleted_at       TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_questions_question_bank_id ON questions (question_bank_id);
CREATE INDEX idx_questions_stimulus_id ON questions (stimulus_id);
CREATE INDEX idx_questions_deleted_at ON questions (deleted_at);
