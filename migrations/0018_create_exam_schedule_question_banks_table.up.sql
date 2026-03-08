CREATE TABLE exam_schedule_question_banks (
    id               BIGSERIAL PRIMARY KEY,
    exam_schedule_id BIGINT NOT NULL REFERENCES exam_schedules(id) ON DELETE CASCADE,
    question_bank_id BIGINT NOT NULL REFERENCES question_banks(id) ON DELETE CASCADE,
    question_count   INTEGER NOT NULL DEFAULT 0,
    weight           DOUBLE PRECISION NOT NULL DEFAULT 1
);

CREATE INDEX idx_esqb_exam_schedule_id ON exam_schedule_question_banks (exam_schedule_id);
CREATE INDEX idx_esqb_question_bank_id ON exam_schedule_question_banks (question_bank_id);
