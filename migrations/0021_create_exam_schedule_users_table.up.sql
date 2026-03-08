CREATE TABLE exam_schedule_users (
    id               BIGSERIAL PRIMARY KEY,
    exam_schedule_id BIGINT NOT NULL REFERENCES exam_schedules(id) ON DELETE CASCADE,
    user_id          BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type             VARCHAR(10) NOT NULL DEFAULT 'include'
);

CREATE INDEX idx_esu_exam_schedule_id ON exam_schedule_users (exam_schedule_id);
CREATE INDEX idx_esu_user_id ON exam_schedule_users (user_id);
