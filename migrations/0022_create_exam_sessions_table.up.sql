CREATE TABLE exam_sessions (
    id               BIGSERIAL PRIMARY KEY,
    exam_schedule_id BIGINT NOT NULL REFERENCES exam_schedules(id) ON DELETE CASCADE,
    user_id          BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status           VARCHAR(50) NOT NULL DEFAULT 'not_started',
    start_time       TIMESTAMPTZ,
    end_time         TIMESTAMPTZ,
    finished_at      TIMESTAMPTZ,
    question_order   JSONB,
    option_order     JSONB,
    score            DOUBLE PRECISION NOT NULL DEFAULT 0,
    max_score        DOUBLE PRECISION NOT NULL DEFAULT 0,
    violation_count  INTEGER NOT NULL DEFAULT 0,
    extra_time       INTEGER NOT NULL DEFAULT 0,
    section_index    INTEGER NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Composite unique: prevent duplicate sessions per user per schedule (BUG-01 fix)
CREATE UNIQUE INDEX idx_session_user_schedule ON exam_sessions (exam_schedule_id, user_id);
