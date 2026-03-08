CREATE TABLE regrade_logs (
    id               BIGSERIAL PRIMARY KEY,
    exam_schedule_id BIGINT NOT NULL REFERENCES exam_schedules(id) ON DELETE CASCADE,
    requested_by     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    sessions_count   INTEGER NOT NULL DEFAULT 0,
    score_changes    JSONB,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_regrade_logs_exam_schedule_id ON regrade_logs (exam_schedule_id);
