-- Prevent duplicate question bank assignments to same schedule
CREATE UNIQUE INDEX IF NOT EXISTS idx_esqb_schedule_bank
ON exam_schedule_question_banks (exam_schedule_id, question_bank_id);

-- Prevent duplicate user assignments to same schedule
CREATE UNIQUE INDEX IF NOT EXISTS idx_esu_schedule_user_type
ON exam_schedule_users (exam_schedule_id, user_id, type);
