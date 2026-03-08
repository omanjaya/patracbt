-- Composite index for exam_answers lookups by session + question
CREATE INDEX IF NOT EXISTS idx_exam_answers_session_question ON exam_answers (session_id, question_id);

-- Composite index for finding a user's session in a schedule
CREATE INDEX IF NOT EXISTS idx_exam_sessions_user_schedule ON exam_sessions (user_id, exam_schedule_id);
