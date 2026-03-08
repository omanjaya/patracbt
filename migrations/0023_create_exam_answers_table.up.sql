CREATE TABLE exam_answers (
    id              BIGSERIAL PRIMARY KEY,
    exam_session_id BIGINT NOT NULL REFERENCES exam_sessions(id) ON DELETE CASCADE,
    question_id     BIGINT NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    answer          JSONB,
    is_flagged      BOOLEAN NOT NULL DEFAULT FALSE,
    answered_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Composite unique: one answer per question per session (BUG-04 fix, enables ON CONFLICT)
CREATE UNIQUE INDEX idx_answer_session_question ON exam_answers (exam_session_id, question_id);

-- Standalone index for fast session lookups
CREATE INDEX idx_exam_answers_session_id ON exam_answers (exam_session_id);
