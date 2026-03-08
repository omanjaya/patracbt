-- Additional performance indexes for exam queries
CREATE INDEX IF NOT EXISTS idx_exam_sessions_user_status ON exam_sessions (user_id, status);
CREATE INDEX IF NOT EXISTS idx_exam_sessions_schedule_status ON exam_sessions (exam_schedule_id, status);
CREATE INDEX IF NOT EXISTS idx_exam_sessions_finished_at ON exam_sessions (finished_at);
CREATE INDEX IF NOT EXISTS idx_exam_answers_created_at ON exam_answers (created_at);
CREATE INDEX IF NOT EXISTS idx_exam_schedules_created_at ON exam_schedules (created_at);
