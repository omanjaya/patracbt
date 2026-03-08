package exam

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/internal/domain/service"
	"github.com/omanjaya/patra/internal/infrastructure/cache"
	"github.com/omanjaya/patra/pkg/hashid"
	"github.com/omanjaya/patra/pkg/logger"
	"github.com/omanjaya/patra/pkg/pagination"
	"github.com/omanjaya/patra/pkg/types"
	"gorm.io/gorm"
)

// SafeQuestion strips correct_answer for delivery to peserta
type SafeQuestion struct {
	ID             uint        `json:"id"`
	QuestionBankID uint        `json:"question_bank_id"`
	StimulusID     *uint       `json:"stimulus_id"`
	QuestionType   string      `json:"question_type"`
	Body           string      `json:"body"`
	Score          float64     `json:"score"`
	Difficulty     string      `json:"difficulty"`
	Options        types.JSON  `json:"options"`
	OrderIndex     int         `json:"order_index"`
}

type StartExamResult struct {
	Session   *entity.ExamSession `json:"session"`
	HashID    string              `json:"hash_id"`
	Questions []SafeQuestion      `json:"questions"`
	Answers   []entity.ExamAnswer `json:"answers"`
}

type ExamSessionUseCase struct {
	sessionRepo  repository.ExamSessionRepository
	scheduleRepo repository.ExamScheduleRepository
	questionRepo repository.QuestionRepository
	calculator   *service.ScoreCalculator
	examCache    *cache.ExamCache
	flusher      *cache.AnswerFlusher
}

func NewExamSessionUseCase(
	sessionRepo repository.ExamSessionRepository,
	scheduleRepo repository.ExamScheduleRepository,
	questionRepo repository.QuestionRepository,
	examCache *cache.ExamCache,
	flusher *cache.AnswerFlusher,
) *ExamSessionUseCase {
	return &ExamSessionUseCase{
		sessionRepo:  sessionRepo,
		scheduleRepo: scheduleRepo,
		questionRepo: questionRepo,
		calculator:   service.NewScoreCalculator(),
		examCache:    examCache,
		flusher:      flusher,
	}
}

func (uc *ExamSessionUseCase) GetAvailableExams(userID uint) ([]*entity.ExamSchedule, error) {
	// Load all published/active schedules
	p := pagination.Params{Page: 1, PerPage: 100}
	schedules, _, err := uc.scheduleRepo.List(repository.ExamScheduleFilter{Status: entity.ExamStatusPublished}, p)
	if err != nil {
		return nil, err
	}
	active, _, err := uc.scheduleRepo.List(repository.ExamScheduleFilter{Status: entity.ExamStatusActive}, p)
	if err != nil {
		return nil, err
	}
	schedules = append(schedules, active...)

	// Fetch user's rombel and tag memberships once to avoid N+1 per schedule
	userRombelIDs, err := uc.sessionRepo.GetUserRombelIDs(userID)
	if err != nil {
		return nil, err
	}
	userTagIDs, err := uc.sessionRepo.GetUserTagIDs(userID)
	if err != nil {
		return nil, err
	}
	rombelSet := make(map[uint]struct{}, len(userRombelIDs))
	for _, id := range userRombelIDs {
		rombelSet[id] = struct{}{}
	}
	tagSet := make(map[uint]struct{}, len(userTagIDs))
	for _, id := range userTagIDs {
		tagSet[id] = struct{}{}
	}

	var eligible []*entity.ExamSchedule
	for _, s := range schedules {
		if uc.isEligibleFromSets(s, userID, rombelSet, tagSet) {
			eligible = append(eligible, s)
		}
	}
	return eligible, nil
}

// GetMyHistory returns the student's finished/terminated sessions with schedule info.
func (uc *ExamSessionUseCase) GetMyHistory(userID uint) ([]*entity.ExamSession, error) {
	sessions, err := uc.sessionRepo.ListByUser(userID)
	if err != nil {
		return nil, err
	}
	var finished []*entity.ExamSession
	for _, s := range sessions {
		if s.Status == entity.SessionStatusFinished || s.Status == entity.SessionStatusTerminated {
			finished = append(finished, s)
		}
	}
	return finished, nil
}

// isEligibleFromSets checks eligibility using pre-fetched membership sets (no extra DB queries).
func (uc *ExamSessionUseCase) isEligibleFromSets(schedule *entity.ExamSchedule, userID uint, rombelSet, tagSet map[uint]struct{}) bool {
	// Check individual user whitelist/blacklist first
	if len(schedule.Users) > 0 {
		var hasIncludeList bool
		for _, u := range schedule.Users {
			if u.Type == "exclude" && u.UserID == userID {
				return false // user is explicitly blocked
			}
			if u.Type == "include" {
				hasIncludeList = true
			}
		}
		if hasIncludeList {
			for _, u := range schedule.Users {
				if u.Type == "include" && u.UserID == userID {
					return true
				}
			}
			return false // include list exists but user is not in it
		}
	}

	// No restrictions = everyone eligible
	if len(schedule.Rombels) == 0 && len(schedule.Tags) == 0 {
		return true
	}
	for _, r := range schedule.Rombels {
		if _, ok := rombelSet[r.RombelID]; ok {
			return true
		}
	}
	for _, t := range schedule.Tags {
		if _, ok := tagSet[t.TagID]; ok {
			return true
		}
	}
	return false
}

func (uc *ExamSessionUseCase) LoadSession(sessionID, userID uint) (*StartExamResult, error) {
	session, err := uc.sessionRepo.FindByID(sessionID)
	if err != nil {
		return nil, errors.New("sesi tidak ditemukan")
	}
	if session.UserID != userID {
		return nil, errors.New("akses ditolak")
	}
	return uc.loadSessionResult(session)
}

func (uc *ExamSessionUseCase) loadSessionResult(session *entity.ExamSession) (*StartExamResult, error) {
	// Parse question order
	var order []uint
	if err := json.Unmarshal(session.QuestionOrder, &order); err != nil || len(order) == 0 {
		return nil, errors.New("urutan soal tidak valid")
	}

	// Batch-load all questions in a single query to avoid N+1
	fetched, err := uc.questionRepo.FindByIDs(order)
	if err != nil {
		return nil, err
	}
	// Build map for O(1) lookup
	qMap := make(map[uint]*entity.Question, len(fetched))
	for _, q := range fetched {
		qMap[q.ID] = q
	}
	// Preserve the original order
	questions := make([]*entity.Question, 0, len(order))
	for _, qID := range order {
		if q, ok := qMap[qID]; ok {
			questions = append(questions, q)
		}
	}

	answers, _ := uc.sessionRepo.GetAllAnswers(session.ID)
	safeQuestions := toSafeQuestions(questions)

	// Apply option order if stored
	if len(session.OptionOrder) > 0 {
		safeQuestions = uc.applyOptionOrder(safeQuestions, session.OptionOrder)
	}

	return &StartExamResult{
		Session:   session,
		HashID:    hashid.Encode(session.ID),
		Questions: safeQuestions,
		Answers:   answers,
	}, nil
}

func (uc *ExamSessionUseCase) GetSessionsBySchedule(scheduleID uint, p pagination.Params) ([]*entity.ExamSession, int64, error) {
	return uc.sessionRepo.ListBySchedule(scheduleID, p)
}

// GetSessionByID returns a session (for handler-level WS emit)
func (uc *ExamSessionUseCase) GetSessionByID(sessionID uint) (*entity.ExamSession, error) {
	return uc.sessionRepo.FindByID(sessionID)
}

// GetAnswerStats returns (answered, total) for a session
// BUG-03 fix: hitung hanya jawaban yang benar-benar terisi (non-empty)
func (uc *ExamSessionUseCase) GetAnswerStats(sessionID uint) (int, int) {
	answered, err := uc.sessionRepo.CountNonEmptyAnswers(sessionID)
	if err != nil {
		// fallback ke count semua jika query gagal
		answers, _ := uc.sessionRepo.GetAllAnswers(sessionID)
		answered = len(answers)
	}
	session, err := uc.sessionRepo.FindByID(sessionID)
	if err != nil {
		return answered, 0
	}
	var order []uint
	if err := json.Unmarshal(session.QuestionOrder, &order); err != nil {
		logger.Log.Errorf("GetAnswerStats: invalid question order for session %d: %v", sessionID, err)
		return answered, 0
	}
	return answered, len(order)
}

// GetAnswerStatsWithSession returns session + (answered, total) dalam satu method
// BUG-15 fix: mengurangi jumlah DB query dari 3 menjadi 2 di SaveAnswer handler
func (uc *ExamSessionUseCase) GetAnswerStatsWithSession(sessionID uint) (*entity.ExamSession, int, int) {
	session, err := uc.sessionRepo.FindByID(sessionID)
	if err != nil {
		return nil, 0, 0
	}
	answered, err := uc.sessionRepo.CountNonEmptyAnswers(sessionID)
	if err != nil {
		answers, _ := uc.sessionRepo.GetAllAnswers(sessionID)
		answered = len(answers)
	}
	var order []uint
	if err := json.Unmarshal(session.QuestionOrder, &order); err != nil {
		logger.Log.Errorf("GetAnswerStatsWithSession: invalid question order for session %d: %v", sessionID, err)
		return session, answered, 0
	}
	return session, answered, len(order)
}

// GetOngoingByUser returns the active session for a user (for single session check).
func (uc *ExamSessionUseCase) GetOngoingByUser(userID uint) (*entity.ExamSession, error) {
	return uc.sessionRepo.FindOngoingByUser(userID)
}

// ListOngoingBySchedule returns all ongoing sessions for a schedule.
func (uc *ExamSessionUseCase) ListOngoingBySchedule(scheduleID uint) ([]*entity.ExamSession, error) {
	return uc.sessionRepo.ListOngoingBySchedule(scheduleID)
}

// ListNotStartedBySchedule returns all not-started sessions for a schedule.
func (uc *ExamSessionUseCase) ListNotStartedBySchedule(scheduleID uint) ([]*entity.ExamSession, error) {
	return uc.sessionRepo.ListNotStartedBySchedule(scheduleID)
}

// ─── package-level helpers ─────────────────────────────────────

// isNotFoundError checks if error is a gorm not-found error
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, gorm.ErrRecordNotFound)
}

// isDuplicateKeyError checks if error is a PostgreSQL unique constraint violation
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "duplicate key") ||
		strings.Contains(errStr, "unique constraint") ||
		strings.Contains(errStr, "23505")
}

func toSafeQuestions(questions []*entity.Question) []SafeQuestion {
	safe := make([]SafeQuestion, len(questions))
	for i, q := range questions {
		safe[i] = SafeQuestion{
			ID:             q.ID,
			QuestionBankID: q.QuestionBankID,
			StimulusID:     q.StimulusID,
			QuestionType:   q.QuestionType,
			Body:           q.Body,
			Score:          q.Score,
			Difficulty:     q.Difficulty,
			Options:        q.Options,
			OrderIndex:     q.OrderIndex,
		}
	}
	return safe
}
