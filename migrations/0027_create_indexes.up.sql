-- =============================================================
-- Performance indexes for high-concurrency exam scenarios
-- Optimized for 500+ concurrent users during exam sessions
-- =============================================================

-- exam_sessions: supervision dashboard (filter by schedule + status)
CREATE INDEX idx_ses_schedule_status ON exam_sessions (exam_schedule_id, status);

-- exam_sessions: per-user session lookup (filter by user + status)
CREATE INDEX idx_ses_user_status ON exam_sessions (user_id, status);

-- exam_sessions: sorting by start_time (leaderboard, monitoring)
CREATE INDEX idx_ses_schedule_starttime ON exam_sessions (exam_schedule_id, start_time);

-- exam_sessions: schedule-only lookups
CREATE INDEX idx_ses_schedule_id ON exam_sessions (exam_schedule_id);

-- exam_schedules: status filter for listing
CREATE INDEX idx_exam_schedules_status ON exam_schedules (status);

-- exam_schedule_rooms: room-based supervision token lookups
CREATE TABLE IF NOT EXISTS exam_schedule_rooms (
    exam_schedule_id   BIGINT NOT NULL REFERENCES exam_schedules(id) ON DELETE CASCADE,
    room_id            BIGINT NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    supervision_token  VARCHAR(6) NOT NULL,
    PRIMARY KEY (exam_schedule_id, room_id)
);

-- violation_logs: tracking cheating events per session
CREATE TABLE IF NOT EXISTS violation_logs (
    id              BIGSERIAL PRIMARY KEY,
    exam_session_id BIGINT NOT NULL REFERENCES exam_sessions(id) ON DELETE CASCADE,
    violation_type  VARCHAR(100) NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_violation_logs_session_id ON violation_logs (exam_session_id);

-- user_profiles: add deferred FK constraints to rombels and rooms (tables created after user_profiles)
ALTER TABLE user_profiles ADD CONSTRAINT fk_user_profiles_rombel FOREIGN KEY (rombel_id) REFERENCES rombels(id) ON DELETE SET NULL;
ALTER TABLE user_profiles ADD CONSTRAINT fk_user_profiles_room FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE SET NULL;

-- questions: composite index for bank + order queries
CREATE INDEX idx_questions_bank_order ON questions (question_bank_id, order_index);

-- questions: type-based filtering within a bank
CREATE INDEX idx_questions_bank_type ON questions (question_bank_id, question_type);

-- user lookup by role (admin dashboard, filtering)
CREATE INDEX idx_users_role ON users (role);

-- user lookup by active status
CREATE INDEX idx_users_is_active ON users (is_active);
