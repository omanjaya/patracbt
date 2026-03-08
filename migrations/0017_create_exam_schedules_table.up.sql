CREATE TABLE exam_schedules (
    id                     BIGSERIAL PRIMARY KEY,
    name                   VARCHAR(255) NOT NULL,
    token                  VARCHAR(255) NOT NULL,
    supervision_token      VARCHAR(255) NOT NULL DEFAULT '',
    start_time             TIMESTAMPTZ NOT NULL,
    end_time               TIMESTAMPTZ NOT NULL,
    duration_minutes       INTEGER NOT NULL DEFAULT 60,
    status                 VARCHAR(50) NOT NULL DEFAULT 'draft',
    allow_see_result       BOOLEAN NOT NULL DEFAULT TRUE,
    max_violations         INTEGER NOT NULL DEFAULT 3,
    randomize_questions    BOOLEAN NOT NULL DEFAULT FALSE,
    randomize_options      BOOLEAN NOT NULL DEFAULT FALSE,
    next_exam_schedule_id  BIGINT REFERENCES exam_schedules(id) ON DELETE SET NULL,
    late_policy            VARCHAR(50) NOT NULL DEFAULT 'allow_full_time',
    min_working_time       INTEGER NOT NULL DEFAULT 0,
    detect_cheating        BOOLEAN NOT NULL DEFAULT TRUE,
    cheating_limit         INTEGER NOT NULL DEFAULT 0,
    show_score_after       VARCHAR(50) NOT NULL DEFAULT 'immediately',
    last_graded_at         TIMESTAMPTZ,
    created_by             BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    deleted_at             TIMESTAMPTZ,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_exam_schedules_token ON exam_schedules (token);
CREATE INDEX idx_exam_schedules_supervision_token ON exam_schedules (supervision_token);
CREATE INDEX idx_exam_schedules_next_id ON exam_schedules (next_exam_schedule_id);
CREATE INDEX idx_exam_schedules_created_by ON exam_schedules (created_by);
CREATE INDEX idx_exam_schedules_deleted_at ON exam_schedules (deleted_at);
