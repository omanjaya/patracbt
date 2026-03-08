-- Drop performance indexes and extra tables in reverse order
DROP INDEX IF EXISTS idx_users_is_active;
DROP INDEX IF EXISTS idx_users_role;
DROP INDEX IF EXISTS idx_questions_bank_type;
DROP INDEX IF EXISTS idx_questions_bank_order;
ALTER TABLE user_profiles DROP CONSTRAINT IF EXISTS fk_user_profiles_room;
ALTER TABLE user_profiles DROP CONSTRAINT IF EXISTS fk_user_profiles_rombel;
DROP INDEX IF EXISTS idx_violation_logs_session_id;
DROP TABLE IF EXISTS violation_logs;
DROP TABLE IF EXISTS exam_schedule_rooms;
DROP INDEX IF EXISTS idx_exam_schedules_status;
DROP INDEX IF EXISTS idx_ses_schedule_id;
DROP INDEX IF EXISTS idx_ses_schedule_starttime;
DROP INDEX IF EXISTS idx_ses_user_status;
DROP INDEX IF EXISTS idx_ses_schedule_status;
