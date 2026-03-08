CREATE TABLE question_banks (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    subject_id  BIGINT REFERENCES subjects(id) ON DELETE SET NULL,
    description TEXT NOT NULL DEFAULT '',
    status      VARCHAR(50) NOT NULL DEFAULT 'active',
    created_by  BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    deleted_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_question_banks_subject_id ON question_banks (subject_id);
CREATE INDEX idx_question_banks_created_by ON question_banks (created_by);
CREATE INDEX idx_question_banks_deleted_at ON question_banks (deleted_at);
