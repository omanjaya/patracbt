package exam

import (
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/omanjaya/patra/internal/application/dto"
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/pagination"
)

type ExamScheduleUseCase struct {
	repo        repository.ExamScheduleRepository
	sessionRepo repository.ExamSessionRepository
}

func NewExamScheduleUseCase(repo repository.ExamScheduleRepository, sessionRepo ...repository.ExamSessionRepository) *ExamScheduleUseCase {
	uc := &ExamScheduleUseCase{repo: repo}
	if len(sessionRepo) > 0 {
		uc.sessionRepo = sessionRepo[0]
	}
	return uc
}

func (uc *ExamScheduleUseCase) List(filter repository.ExamScheduleFilter, p pagination.Params) ([]*entity.ExamSchedule, int64, error) {
	return uc.repo.List(filter, p)
}

func (uc *ExamScheduleUseCase) GetByID(id uint) (*entity.ExamSchedule, error) {
	s, err := uc.repo.FindByID(id)
	if err != nil {
		return nil, errors.New("jadwal ujian tidak ditemukan")
	}
	return s, nil
}

func (uc *ExamScheduleUseCase) Create(req dto.CreateExamScheduleRequest, createdBy uint) (*entity.ExamSchedule, error) {
	if req.EndTime.Before(req.StartTime) {
		return nil, errors.New("waktu selesai harus setelah waktu mulai")
	}

	allowSeeResult := true
	if req.AllowSeeResult != nil {
		allowSeeResult = *req.AllowSeeResult
	}
	maxViolations := req.MaxViolations
	if maxViolations <= 0 {
		maxViolations = 3
	}

	latePolicy := req.LatePolicy
	if latePolicy == "" {
		latePolicy = "allow_full_time"
	}
	detectCheating := true
	if req.DetectCheating != nil {
		detectCheating = *req.DetectCheating
	}
	showScoreAfter := req.ShowScoreAfter
	if showScoreAfter == "" {
		showScoreAfter = "immediately"
	}

	s := &entity.ExamSchedule{
		Name:                 req.Name,
		Token:                generateToken(),
		StartTime:            req.StartTime,
		EndTime:              req.EndTime,
		DurationMinutes:      req.DurationMinutes,
		Status:               entity.ExamStatusDraft,
		AllowSeeResult:       allowSeeResult,
		MaxViolations:        maxViolations,
		RandomizeQuestions:   req.RandomizeQuestions,
		RandomizeOptions:     req.RandomizeOptions,
		NextExamScheduleID:   req.NextExamScheduleID,
		LatePolicy:           latePolicy,
		MinWorkingTime:       req.MinWorkingTime,
		DetectCheating:       detectCheating,
		CheatingLimit:        req.CheatingLimit,
		ShowScoreAfter:       showScoreAfter,
		CreatedBy:            createdBy,
	}

	if err := uc.repo.Create(s); err != nil {
		return nil, err
	}

	if err := uc.syncRelations(s.ID, req.QuestionBanks, req.RombelIDs, req.TagIDs); err != nil {
		return nil, err
	}
	if err := uc.syncUsers(s.ID, req.IncludeUsers, req.ExcludeUsers); err != nil {
		return nil, err
	}

	return uc.repo.FindByID(s.ID)
}

func (uc *ExamScheduleUseCase) Update(id uint, req dto.UpdateExamScheduleRequest) (*entity.ExamSchedule, error) {
	s, err := uc.repo.FindByID(id)
	if err != nil {
		return nil, errors.New("jadwal ujian tidak ditemukan")
	}
	if s.Status != entity.ExamStatusDraft {
		return nil, errors.New("tidak dapat mengedit jadwal yang sudah dipublikasi atau aktif")
	}
	if req.EndTime.Before(req.StartTime) {
		return nil, errors.New("waktu selesai harus setelah waktu mulai")
	}

	allowSeeResult := true
	if req.AllowSeeResult != nil {
		allowSeeResult = *req.AllowSeeResult
	}

	detectCheating := true
	if req.DetectCheating != nil {
		detectCheating = *req.DetectCheating
	}

	s.Name = req.Name
	s.StartTime = req.StartTime
	s.EndTime = req.EndTime
	s.DurationMinutes = req.DurationMinutes
	s.AllowSeeResult = allowSeeResult
	s.MaxViolations = req.MaxViolations
	s.RandomizeQuestions = req.RandomizeQuestions
	s.RandomizeOptions = req.RandomizeOptions
	s.NextExamScheduleID = req.NextExamScheduleID
	if req.LatePolicy != "" {
		s.LatePolicy = req.LatePolicy
	}
	s.MinWorkingTime = req.MinWorkingTime
	s.DetectCheating = detectCheating
	s.CheatingLimit = req.CheatingLimit
	if req.ShowScoreAfter != "" {
		s.ShowScoreAfter = req.ShowScoreAfter
	}

	if err := uc.repo.Update(s); err != nil {
		return nil, err
	}
	if err := uc.syncRelations(id, req.QuestionBanks, req.RombelIDs, req.TagIDs); err != nil {
		return nil, err
	}
	if err := uc.syncUsers(id, req.IncludeUsers, req.ExcludeUsers); err != nil {
		return nil, err
	}
	return uc.repo.FindByID(id)
}

// BUG-10 fix: validasi transisi status — hanya izinkan transisi yang legal
// draft → published → active → finished (dan finished tidak bisa kembali)
var validStatusTransitions = map[string][]string{
	entity.ExamStatusDraft:     {entity.ExamStatusPublished},
	entity.ExamStatusPublished: {entity.ExamStatusActive, entity.ExamStatusDraft},
	entity.ExamStatusActive:    {entity.ExamStatusFinished},
	entity.ExamStatusFinished:  {}, // tidak ada transisi dari finished
}

func (uc *ExamScheduleUseCase) UpdateStatus(id uint, status string) error {
	s, err := uc.repo.FindByID(id)
	if err != nil {
		return errors.New("jadwal ujian tidak ditemukan")
	}

	allowed, ok := validStatusTransitions[s.Status]
	if !ok {
		return errors.New("status saat ini tidak dikenali")
	}
	isAllowed := false
	for _, a := range allowed {
		if a == status {
			isAllowed = true
			break
		}
	}
	if !isAllowed {
		return errors.New("transisi status tidak diizinkan: " + s.Status + " → " + status)
	}

	s.Status = status
	return uc.repo.Update(s)
}

func (uc *ExamScheduleUseCase) Delete(id uint) error {
	if _, err := uc.repo.FindByID(id); err != nil {
		return errors.New("jadwal ujian tidak ditemukan")
	}
	// Check for ongoing sessions before deleting
	if uc.sessionRepo != nil {
		ongoingCount, err := uc.sessionRepo.CountByScheduleAndStatus(id, "ongoing")
		if err == nil && ongoingCount > 0 {
			return errors.New("tidak dapat menghapus jadwal dengan sesi yang sedang berlangsung")
		}
	}
	return uc.repo.Delete(id)
}

func (uc *ExamScheduleUseCase) Restore(id uint) error {
	return uc.repo.Restore(id)
}

func (uc *ExamScheduleUseCase) ForceDelete(id uint) error {
	return uc.repo.ForceDelete(id)
}

func (uc *ExamScheduleUseCase) ListTrashed(filter repository.ExamScheduleFilter, p pagination.Params) ([]*entity.ExamSchedule, int64, error) {
	return uc.repo.ListTrashed(filter, p)
}

func (uc *ExamScheduleUseCase) syncRelations(scheduleID uint, banks []dto.ExamScheduleBankInput, rombelIDs, tagIDs []uint) error {
	bankEntities := make([]entity.ExamScheduleQuestionBank, len(banks))
	for i, b := range banks {
		weight := b.Weight
		if weight <= 0 {
			weight = 1
		}
		bankEntities[i] = entity.ExamScheduleQuestionBank{
			ExamScheduleID: scheduleID,
			QuestionBankID: b.QuestionBankID,
			QuestionCount:  b.QuestionCount,
			Weight:         weight,
		}
	}
	if err := uc.repo.SetQuestionBanks(scheduleID, bankEntities); err != nil {
		return err
	}
	if err := uc.repo.SetRombels(scheduleID, rombelIDs); err != nil {
		return err
	}
	return uc.repo.SetTags(scheduleID, tagIDs)
}

func (uc *ExamScheduleUseCase) syncUsers(scheduleID uint, includeUserIDs, excludeUserIDs []uint) error {
	var users []entity.ExamScheduleUser
	for _, uid := range includeUserIDs {
		users = append(users, entity.ExamScheduleUser{
			ExamScheduleID: scheduleID,
			UserID:         uid,
			Type:           "include",
		})
	}
	for _, uid := range excludeUserIDs {
		users = append(users, entity.ExamScheduleUser{
			ExamScheduleID: scheduleID,
			UserID:         uid,
			Type:           "exclude",
		})
	}
	return uc.repo.SetUsers(scheduleID, users)
}

func generateToken() string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 6)
	rand.Read(b)
	for i := range b {
		b[i] = chars[int(b[i])%len(chars)]
	}
	return fmt.Sprintf("%s-%s", string(b[:3]), string(b[3:]))
}

func (uc *ExamScheduleUseCase) Clone(scheduleID uint, clonedBy uint) (*entity.ExamSchedule, error) {
	orig, err := uc.repo.FindByID(scheduleID)
	if err != nil {
		return nil, errors.New("jadwal ujian tidak ditemukan")
	}

	// Create new schedule with same config but draft status and cleared times
	cloned := &entity.ExamSchedule{
		Name:                 "[Clone] " + orig.Name,
		Token:                generateToken(),
		DurationMinutes:      orig.DurationMinutes,
		Status:               entity.ExamStatusDraft,
		AllowSeeResult:       orig.AllowSeeResult,
		MaxViolations:        orig.MaxViolations,
		RandomizeQuestions:   orig.RandomizeQuestions,
		RandomizeOptions:     orig.RandomizeOptions,
		NextExamScheduleID:   orig.NextExamScheduleID,
		LatePolicy:           orig.LatePolicy,
		MinWorkingTime:       orig.MinWorkingTime,
		DetectCheating:       orig.DetectCheating,
		CheatingLimit:        orig.CheatingLimit,
		ShowScoreAfter:       orig.ShowScoreAfter,
		CreatedBy:            clonedBy,
	}

	if err := uc.repo.Create(cloned); err != nil {
		return nil, err
	}

	// Copy question banks
	banks := make([]entity.ExamScheduleQuestionBank, len(orig.QuestionBanks))
	for i, b := range orig.QuestionBanks {
		banks[i] = entity.ExamScheduleQuestionBank{
			ExamScheduleID: cloned.ID,
			QuestionBankID: b.QuestionBankID,
			QuestionCount:  b.QuestionCount,
			Weight:         b.Weight,
		}
	}
	if err := uc.repo.SetQuestionBanks(cloned.ID, banks); err != nil {
		return nil, err
	}

	// Copy rombels
	rombelIDs := make([]uint, len(orig.Rombels))
	for i, r := range orig.Rombels {
		rombelIDs[i] = r.RombelID
	}
	if err := uc.repo.SetRombels(cloned.ID, rombelIDs); err != nil {
		return nil, err
	}

	// Copy tags
	tagIDs := make([]uint, len(orig.Tags))
	for i, t := range orig.Tags {
		tagIDs[i] = t.TagID
	}
	if err := uc.repo.SetTags(cloned.ID, tagIDs); err != nil {
		return nil, err
	}

	// Copy users (include/exclude)
	users := make([]entity.ExamScheduleUser, len(orig.Users))
	for i, u := range orig.Users {
		users[i] = entity.ExamScheduleUser{
			ExamScheduleID: cloned.ID,
			UserID:         u.UserID,
			Type:           u.Type,
		}
	}
	if err := uc.repo.SetUsers(cloned.ID, users); err != nil {
		return nil, err
	}

	return uc.repo.FindByID(cloned.ID)
}

func (uc *ExamScheduleUseCase) SaveSupervisionTokens(scheduleID uint, globalToken string, roomTokensMap map[uint]string) error {
	s, err := uc.repo.FindByID(scheduleID)
	if err != nil {
		return errors.New("jadwal tidak ditemukan")
	}

	// Update global token if provided
	if globalToken != "" {
		s.SupervisionToken = globalToken
		if err := uc.repo.UpdateSupervisionToken(scheduleID, globalToken); err != nil {
			return err
		}
	}

	// Make Room token relations
	var roomTokens []entity.ExamScheduleRoom
	for roomID, token := range roomTokensMap {
		roomTokens = append(roomTokens, entity.ExamScheduleRoom{
			ExamScheduleID:   scheduleID,
			RoomID:           roomID,
			SupervisionToken: token,
		})
	}

	return uc.repo.SetExamRooms(scheduleID, roomTokens)
}
