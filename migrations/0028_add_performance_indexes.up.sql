CREATE INDEX IF NOT EXISTS idx_exam_answers_question_id ON exam_answers (question_id);
CREATE INDEX IF NOT EXISTS idx_exam_sessions_schedule_status_user ON exam_sessions (exam_schedule_id, status, user_id);
