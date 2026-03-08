package postgres

import (
	"time"

	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/pkg/pagination"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ExamSessionRepo struct {
	db *gorm.DB
}

func NewExamSessionRepository(db *gorm.DB) *ExamSessionRepo {
	return &ExamSessionRepo{db: db}
}

func (r *ExamSessionRepo) Create(s *entity.ExamSession) error {
	return r.db.Create(s).Error
}

func (r *ExamSessionRepo) FindByID(id uint) (*entity.ExamSession, error) {
	var s entity.ExamSession
	err := r.db.Where("id = ?", id).
		Preload("ExamSchedule").
		Preload("User").
		First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *ExamSessionRepo) FindByIDBasic(id uint) (*entity.ExamSession, error) {
	var s entity.ExamSession
	err := r.db.Where("id = ?", id).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *ExamSessionRepo) FindByUserAndSchedule(userID, scheduleID uint) (*entity.ExamSession, error) {
	var s entity.ExamSession
	err := r.db.Where("user_id = ? AND exam_schedule_id = ?", userID, scheduleID).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *ExamSessionRepo) Update(s *entity.ExamSession) error {
	return r.db.Model(s).Updates(map[string]any{
		"status":          s.Status,
		"start_time":      s.StartTime,
		"end_time":        s.EndTime,
		"finished_at":     s.FinishedAt,
		"score":           s.Score,
		"max_score":       s.MaxScore,
		"violation_count": s.ViolationCount,
		"extra_time":      s.ExtraTime,
		"section_index":   s.SectionIndex,
		"question_order":  s.QuestionOrder,
	}).Error
}

func (r *ExamSessionRepo) AtomicFinish(id uint) (int64, error) {
	result := r.db.Model(&entity.ExamSession{}).
		Where("id = ? AND status = ?", id, entity.SessionStatusOngoing).
		Update("status", entity.SessionStatusFinished)
	return result.RowsAffected, result.Error
}

func (r *ExamSessionRepo) UpdateStatus(id uint, status string, finishedAt *time.Time) error {
	updates := map[string]any{"status": status, "finished_at": finishedAt}
	return r.db.Model(&entity.ExamSession{}).Where("id = ?", id).Updates(updates).Error
}

func (r *ExamSessionRepo) UpdateScore(id uint, score, maxScore float64) error {
	return r.db.Model(&entity.ExamSession{}).Where("id = ?", id).
		Updates(map[string]any{"score": score, "max_score": maxScore}).Error
}

func (r *ExamSessionRepo) IncrementViolation(id uint) error {
	return r.db.Model(&entity.ExamSession{}).Where("id = ?", id).
		UpdateColumn("violation_count", gorm.Expr("violation_count + 1")).Error
}

func (r *ExamSessionRepo) UpdateExtraTime(id uint, extraTime int) error {
	return r.db.Model(&entity.ExamSession{}).Where("id = ?", id).
		Update("extra_time", extraTime).Error
}

func (r *ExamSessionRepo) ListBySchedule(scheduleID uint, p pagination.Params) ([]*entity.ExamSession, int64, error) {
	var total int64
	r.db.Model(&entity.ExamSession{}).Where("exam_schedule_id = ?", scheduleID).Count(&total)

	var sessions []*entity.ExamSession
	err := r.db.Where("exam_schedule_id = ?", scheduleID).
		Preload("User").
		Order("created_at DESC").
		Offset(p.Offset()).Limit(p.PerPage).
		Find(&sessions).Error
	return sessions, total, err
}

func (r *ExamSessionRepo) ListByUser(userID uint) ([]*entity.ExamSession, error) {
	var sessions []*entity.ExamSession
	err := r.db.Where("user_id = ?", userID).
		Preload("ExamSchedule").
		Order("created_at DESC").
		Find(&sessions).Error
	return sessions, err
}

func (r *ExamSessionRepo) UpsertAnswer(answer *entity.ExamAnswer) error {
	now := time.Now()
	answer.AnsweredAt = &now
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "exam_session_id"}, {Name: "question_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"answer", "is_flagged", "answered_at", "updated_at"}),
	}).Create(answer).Error
}

func (r *ExamSessionRepo) BatchUpsertAnswers(answers []*entity.ExamAnswer) error {
	if len(answers) == 0 {
		return nil
	}
	now := time.Now()
	for _, a := range answers {
		a.AnsweredAt = &now
	}
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "exam_session_id"}, {Name: "question_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"answer", "is_flagged", "answered_at", "updated_at"}),
	}).CreateInBatches(answers, 50).Error
}

func (r *ExamSessionRepo) GetAnswer(sessionID, questionID uint) (*entity.ExamAnswer, error) {
	var a entity.ExamAnswer
	err := r.db.Where("exam_session_id = ? AND question_id = ?", sessionID, questionID).First(&a).Error
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *ExamSessionRepo) GetAllAnswers(sessionID uint) ([]entity.ExamAnswer, error) {
	var answers []entity.ExamAnswer
	err := r.db.Where("exam_session_id = ?", sessionID).Find(&answers).Error
	return answers, err
}

// GetAllAnswersBySchedule fetches all answers for all sessions of a schedule in a single query.
// Returns map[sessionID][]ExamAnswer to avoid N+1 when iterating sessions.
func (r *ExamSessionRepo) GetAllAnswersBySchedule(scheduleID uint) (map[uint][]entity.ExamAnswer, error) {
	var answers []entity.ExamAnswer
	err := r.db.
		Joins("JOIN exam_sessions ON exam_sessions.id = exam_answers.exam_session_id").
		Where("exam_sessions.exam_schedule_id = ?", scheduleID).
		Find(&answers).Error
	if err != nil {
		return nil, err
	}
	result := make(map[uint][]entity.ExamAnswer)
	for _, a := range answers {
		result[a.ExamSessionID] = append(result[a.ExamSessionID], a)
	}
	return result, nil
}

// BUG-02 fix: hapus semua jawaban untuk sesi tertentu (digunakan saat ResetSession)
func (r *ExamSessionRepo) DeleteAnswersBySession(sessionID uint) error {
	return r.db.Where("exam_session_id = ?", sessionID).Delete(&entity.ExamAnswer{}).Error
}

// BUG-03 fix: hitung jawaban non-empty (Answer tidak null dan bukan literal "null")
func (r *ExamSessionRepo) CountNonEmptyAnswers(sessionID uint) (int, error) {
	var count int64
	err := r.db.Model(&entity.ExamAnswer{}).
		Where("exam_session_id = ? AND answer IS NOT NULL AND answer != 'null'::jsonb AND answer != '{}'::jsonb", sessionID).
		Count(&count).Error
	return int(count), err
}

func (r *ExamSessionRepo) LogViolation(log *entity.ViolationLog) error {
	return r.db.Create(log).Error
}

func (r *ExamSessionRepo) CountViolations(sessionID uint) (int, error) {
	var count int64
	err := r.db.Model(&entity.ViolationLog{}).Where("exam_session_id = ?", sessionID).Count(&count).Error
	return int(count), err
}

func (r *ExamSessionRepo) UserInRombels(userID uint, rombelIDs []uint) (bool, error) {
	if len(rombelIDs) == 0 {
		return false, nil
	}
	var count int64
	err := r.db.Table("user_rombels").
		Where("user_id = ? AND rombel_id IN ?", userID, rombelIDs).
		Count(&count).Error
	return count > 0, err
}

func (r *ExamSessionRepo) UserHasTags(userID uint, tagIDs []uint) (bool, error) {
	if len(tagIDs) == 0 {
		return false, nil
	}
	var count int64
	err := r.db.Table("user_tags").
		Where("user_id = ? AND tag_id IN ?", userID, tagIDs).
		Count(&count).Error
	return count > 0, err
}

// GetUserRombelIDs returns all rombel IDs that a user belongs to.
func (r *ExamSessionRepo) GetUserRombelIDs(userID uint) ([]uint, error) {
	var ids []uint
	err := r.db.Table("user_rombels").
		Where("user_id = ?", userID).
		Pluck("rombel_id", &ids).Error
	return ids, err
}

// GetUserTagIDs returns all tag IDs that a user belongs to.
func (r *ExamSessionRepo) GetUserTagIDs(userID uint) ([]uint, error) {
	var ids []uint
	err := r.db.Table("user_tags").
		Where("user_id = ?", userID).
		Pluck("tag_id", &ids).Error
	return ids, err
}

// FindOngoingByUser returns the active exam session for a user (if any).
func (r *ExamSessionRepo) FindOngoingByUser(userID uint) (*entity.ExamSession, error) {
	var s entity.ExamSession
	err := r.db.Where("user_id = ? AND status = ?", userID, "ongoing").
		Preload("ExamSchedule").
		First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// FindExpiredOngoing returns sessions that are ongoing but time has expired.
func (r *ExamSessionRepo) FindExpiredOngoing() ([]*entity.ExamSession, error) {
	var sessions []*entity.ExamSession
	err := r.db.Raw(`
		SELECT es.* FROM exam_sessions es
		JOIN exam_schedules sc ON sc.id = es.exam_schedule_id
		WHERE es.status = 'ongoing'
		  AND es.start_time IS NOT NULL
		  AND es.start_time + ((sc.duration_minutes + es.extra_time) * INTERVAL '1 minute') <= NOW()
		LIMIT 1000
	`).Scan(&sessions).Error
	return sessions, err
}

// ListFinishedBySchedule returns all finished sessions for a schedule (for ledger export).
func (r *ExamSessionRepo) ListFinishedBySchedule(scheduleID uint) ([]*entity.ExamSession, error) {
	var sessions []*entity.ExamSession
	err := r.db.Where("exam_schedule_id = ? AND status = ?", scheduleID, "finished").
		Preload("User").
		Preload("Answers").
		Order("score DESC").
		Find(&sessions).Error
	return sessions, err
}

func (r *ExamSessionRepo) ListOngoingBySchedule(scheduleID uint) ([]*entity.ExamSession, error) {
	var sessions []*entity.ExamSession
	err := r.db.Where("exam_schedule_id = ? AND status = ?", scheduleID, entity.SessionStatusOngoing).
		Preload("User").
		Find(&sessions).Error
	return sessions, err
}

func (r *ExamSessionRepo) ListNotStartedBySchedule(scheduleID uint) ([]*entity.ExamSession, error) {
	// Users in the schedule's rombels/tags that have no session
	// Simplified: return sessions with status not_started
	var sessions []*entity.ExamSession
	err := r.db.Where("exam_schedule_id = ? AND status = ?", scheduleID, entity.SessionStatusNotStarted).
		Preload("User").
		Find(&sessions).Error
	return sessions, err
}

func (r *ExamSessionRepo) Delete(id uint) error {
	return r.db.Delete(&entity.ExamSession{}, id).Error
}

func (r *ExamSessionRepo) CountByScheduleAndStatus(scheduleID uint, status string) (int64, error) {
	var count int64
	err := r.db.Model(&entity.ExamSession{}).
		Where("exam_schedule_id = ? AND status = ?", scheduleID, status).
		Count(&count).Error
	return count, err
}

func (r *ExamSessionRepo) CreateRegradeLog(log *entity.RegradeLog) error {
	return r.db.Create(log).Error
}

func (r *ExamSessionRepo) ListRegradeLogs(scheduleID uint) ([]entity.RegradeLog, error) {
	var logs []entity.RegradeLog
	err := r.db.Where("exam_schedule_id = ?", scheduleID).
		Order("created_at DESC").
		Find(&logs).Error
	return logs, err
}

// GetUserRombelNames returns a map of userID to rombel names for the given user IDs.
func (r *ExamSessionRepo) GetUserRombelNames(userIDs []uint) (map[uint][]string, error) {
	if len(userIDs) == 0 {
		return map[uint][]string{}, nil
	}
	type row struct {
		UserID     uint   `gorm:"column:user_id"`
		RombelName string `gorm:"column:name"`
	}
	var rows []row
	err := r.db.Table("user_rombels").
		Select("user_rombels.user_id, rombels.name").
		Joins("JOIN rombels ON rombels.id = user_rombels.rombel_id").
		Where("user_rombels.user_id IN ? AND rombels.deleted_at IS NULL", userIDs).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[uint][]string)
	for _, r := range rows {
		result[r.UserID] = append(result[r.UserID], r.RombelName)
	}
	return result, nil
}
