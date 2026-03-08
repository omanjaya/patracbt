package exam

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/omanjaya/patra/internal/application/dto"
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/pkg/hashid"
	"github.com/omanjaya/patra/pkg/logger"
	"github.com/omanjaya/patra/pkg/pagination"
	"github.com/omanjaya/patra/pkg/types"
)

func (uc *ExamSessionUseCase) StartExam(userID uint, req dto.StartExamRequest) (*StartExamResult, error) {
	schedule, err := uc.scheduleRepo.FindByID(req.ExamScheduleID)
	if err != nil {
		return nil, errors.New("jadwal ujian tidak ditemukan")
	}

	// Token check
	if schedule.Token != req.Token {
		return nil, errors.New("token ujian tidak valid")
	}

	// Status check
	now := time.Now()
	if schedule.Status != entity.ExamStatusPublished && schedule.Status != entity.ExamStatusActive {
		return nil, errors.New("ujian belum tersedia")
	}
	if now.Before(schedule.StartTime) {
		return nil, errors.New("ujian belum dimulai")
	}
	if now.After(schedule.EndTime) {
		return nil, errors.New("ujian sudah berakhir")
	}

	// Eligibility check
	ok, err := uc.isEligible(userID, schedule)
	if err != nil || !ok {
		return nil, errors.New("anda tidak terdaftar untuk ujian ini")
	}

	// Check existing session
	existing, err := uc.sessionRepo.FindByUserAndSchedule(userID, schedule.ID)
	if err == nil {
		// Session exists
		if existing.Status == entity.SessionStatusFinished || existing.Status == entity.SessionStatusTerminated {
			return nil, errors.New("anda sudah menyelesaikan ujian ini")
		}
		// Check if session end time has passed before allowing resume
		if existing.EndTime != nil && existing.EndTime.Before(time.Now()) {
			// Auto-finish the expired session
			finishedAt := time.Now()
			existing.Status = entity.SessionStatusFinished
			existing.FinishedAt = &finishedAt
			if updateErr := uc.sessionRepo.Update(existing); updateErr != nil {
				logger.Log.Errorf("StartExam: failed to auto-finish expired session #%d: %v", existing.ID, updateErr)
			}
			return nil, errors.New("waktu ujian telah habis")
		}
		// Resume ongoing session
		return uc.loadSessionResult(existing)
	}

	// BUG-01 fix: jika error bukan "not found", propagate
	if !isNotFoundError(err) {
		return nil, err
	}

	// Load questions from assigned banks
	questions, err := uc.loadQuestionsForSchedule(schedule)
	if err != nil {
		return nil, err
	}
	if len(questions) == 0 {
		return nil, errors.New("tidak ada soal yang tersedia untuk ujian ini")
	}

	// Shuffle if enabled — with stimulus grouping
	if schedule.RandomizeQuestions {
		questions = uc.shuffleWithStimulusGrouping(questions)
	}

	// Build question order
	order := make([]uint, len(questions))
	for i, q := range questions {
		order[i] = q.ID
	}
	orderJSON, err := json.Marshal(order)
	if err != nil {
		return nil, fmt.Errorf("gagal marshal question order: %w", err)
	}

	// Build option order if shuffle options enabled
	var optionOrderJSON []byte
	if schedule.RandomizeOptions {
		optionOrder := uc.buildOptionOrder(questions)
		optionOrderJSON, err = json.Marshal(optionOrder)
		if err != nil {
			return nil, fmt.Errorf("gagal marshal option order: %w", err)
		}
	}

	// Late policy: compute effective duration
	effectiveDuration := schedule.DurationMinutes
	if schedule.LatePolicy == "deduct_time" {
		minutesLate := int(math.Ceil(now.Sub(schedule.StartTime).Minutes()))
		if minutesLate > 0 {
			effectiveDuration = schedule.DurationMinutes - minutesLate
			if effectiveDuration <= 0 {
				return nil, errors.New("waktu ujian sudah habis karena keterlambatan")
			}
		}
	}

	// Compute end time
	endTime := now.Add(time.Duration(effectiveDuration) * time.Minute)
	if endTime.After(schedule.EndTime) {
		endTime = schedule.EndTime
	}

	session := &entity.ExamSession{
		ExamScheduleID: schedule.ID,
		UserID:         userID,
		Status:         entity.SessionStatusOngoing,
		StartTime:      &now,
		EndTime:        &endTime,
		QuestionOrder:  types.JSON(orderJSON),
		OptionOrder:    types.JSON(optionOrderJSON),
	}

	if err := uc.sessionRepo.Create(session); err != nil {
		// BUG-01 fix: handle race condition — jika unique constraint violated, resume session yang ada
		if isDuplicateKeyError(err) {
			if existing, lookupErr := uc.sessionRepo.FindByUserAndSchedule(userID, schedule.ID); lookupErr == nil {
				if existing.Status == entity.SessionStatusFinished || existing.Status == entity.SessionStatusTerminated {
					return nil, errors.New("anda sudah menyelesaikan ujian ini")
				}
				return uc.loadSessionResult(existing)
			}
		}
		return nil, err
	}

	// Cache session in Redis for write-behind pattern
	duration := time.Duration(effectiveDuration)*time.Minute + 30*time.Minute
	if cacheErr := uc.examCache.CacheSession(session, duration); cacheErr != nil {
		logger.Log.Warnf("StartExam: failed to cache session #%d: %v", session.ID, cacheErr)
	}

	safeQuestions := toSafeQuestions(questions)

	// Apply option order for new session
	if len(session.OptionOrder) > 0 {
		safeQuestions = uc.applyOptionOrder(safeQuestions, session.OptionOrder)
	}

	return &StartExamResult{
		Session:   session,
		HashID:    hashid.Encode(session.ID),
		Questions: safeQuestions,
		Answers:   []entity.ExamAnswer{},
	}, nil
}

// ─── start helpers ─────────────────────────────────────────────

func (uc *ExamSessionUseCase) isEligible(userID uint, schedule *entity.ExamSchedule) (bool, error) {
	// Check individual user whitelist/blacklist first
	if len(schedule.Users) > 0 {
		var hasIncludeList bool
		for _, u := range schedule.Users {
			if u.Type == "exclude" && u.UserID == userID {
				return false, nil // user is explicitly blocked
			}
			if u.Type == "include" {
				hasIncludeList = true
			}
		}
		if hasIncludeList {
			// Include list exists: user must be in it
			found := false
			for _, u := range schedule.Users {
				if u.Type == "include" && u.UserID == userID {
					found = true
					break
				}
			}
			if !found {
				return false, nil
			}
			// User is in include list, grant access regardless of rombel/tag
			return true, nil
		}
	}

	// No restrictions = everyone eligible
	if len(schedule.Rombels) == 0 && len(schedule.Tags) == 0 {
		return true, nil
	}

	// BUG-06 fix: propagate error dari rombel check
	rombelIDs := make([]uint, len(schedule.Rombels))
	for i, r := range schedule.Rombels {
		rombelIDs[i] = r.RombelID
	}
	if len(rombelIDs) > 0 {
		inRombel, err := uc.sessionRepo.UserInRombels(userID, rombelIDs)
		if err != nil {
			return false, err
		}
		if inRombel {
			return true, nil
		}
	}

	tagIDs := make([]uint, len(schedule.Tags))
	for i, t := range schedule.Tags {
		tagIDs[i] = t.TagID
	}
	if len(tagIDs) > 0 {
		return uc.sessionRepo.UserHasTags(userID, tagIDs)
	}
	return false, nil
}

func (uc *ExamSessionUseCase) loadQuestionsForSchedule(schedule *entity.ExamSchedule) ([]*entity.Question, error) {
	var all []*entity.Question
	p := pagination.Params{Page: 1, PerPage: 9999}

	for _, bankRef := range schedule.QuestionBanks {
		questions, _, err := uc.questionRepo.ListByBank(bankRef.QuestionBankID, p)
		if err != nil {
			continue
		}
		if bankRef.QuestionCount > 0 && len(questions) > bankRef.QuestionCount {
			// Randomly pick QuestionCount questions
			rand.Shuffle(len(questions), func(i, j int) { questions[i], questions[j] = questions[j], questions[i] })
			questions = questions[:bankRef.QuestionCount]
		}
		all = append(all, questions...)
	}
	return all, nil
}

// shuffleWithStimulusGrouping groups questions by stimulus, shuffles groups and within groups.
func (uc *ExamSessionUseCase) shuffleWithStimulusGrouping(questions []*entity.Question) []*entity.Question {
	// Group questions: stimulus-linked share a group, non-stimulus are individual
	type group struct {
		stimulusID *uint
		questions  []*entity.Question
	}
	groupMap := make(map[uint]*group)     // stimulus_id -> group
	var groups []*group
	var noStimulus []*group

	for _, q := range questions {
		if q.StimulusID != nil {
			if g, ok := groupMap[*q.StimulusID]; ok {
				g.questions = append(g.questions, q)
			} else {
				g = &group{stimulusID: q.StimulusID, questions: []*entity.Question{q}}
				groupMap[*q.StimulusID] = g
				groups = append(groups, g)
			}
		} else {
			g := &group{stimulusID: nil, questions: []*entity.Question{q}}
			noStimulus = append(noStimulus, g)
		}
	}

	allGroups := append(groups, noStimulus...)

	// Shuffle groups
	rand.Shuffle(len(allGroups), func(i, j int) {
		allGroups[i], allGroups[j] = allGroups[j], allGroups[i]
	})

	// Shuffle questions within each group
	for _, g := range allGroups {
		if len(g.questions) > 1 {
			rand.Shuffle(len(g.questions), func(i, j int) {
				g.questions[i], g.questions[j] = g.questions[j], g.questions[i]
			})
		}
	}

	// Flatten
	result := make([]*entity.Question, 0, len(questions))
	for _, g := range allGroups {
		result = append(result, g.questions...)
	}
	return result
}

// buildOptionOrder creates shuffled option indices for each question based on type.
func (uc *ExamSessionUseCase) buildOptionOrder(questions []*entity.Question) map[string][]int {
	optionOrder := make(map[string][]int)

	for _, q := range questions {
		key := fmt.Sprintf("%d", q.ID)

		switch q.QuestionType {
		case entity.QuestionTypePG, entity.QuestionTypePGK, entity.QuestionTypeBenarSalah:
			// Parse options to get count
			var options []json.RawMessage
			if err := json.Unmarshal(q.Options, &options); err != nil {
				continue
			}
			indices := makeIndices(len(options))
			rand.Shuffle(len(indices), func(i, j int) {
				indices[i], indices[j] = indices[j], indices[i]
			})
			optionOrder[key] = indices

		case entity.QuestionTypeMenjodohkan:
			// Shuffle answer column only; parse to get count of pairs
			var pairs []json.RawMessage
			if err := json.Unmarshal(q.Options, &pairs); err != nil {
				continue
			}
			indices := makeIndices(len(pairs))
			rand.Shuffle(len(indices), func(i, j int) {
				indices[i], indices[j] = indices[j], indices[i]
			})
			optionOrder[key] = indices

		case entity.QuestionTypeMatrix:
			// Shuffle row indices
			var rows []json.RawMessage
			if err := json.Unmarshal(q.Options, &rows); err != nil {
				continue
			}
			indices := makeIndices(len(rows))
			rand.Shuffle(len(indices), func(i, j int) {
				indices[i], indices[j] = indices[j], indices[i]
			})
			optionOrder[key] = indices

		// isian_singkat, esai: no options to shuffle
		}
	}
	return optionOrder
}

// applyOptionOrder reorders SafeQuestion options according to stored shuffle mapping.
func (uc *ExamSessionUseCase) applyOptionOrder(questions []SafeQuestion, optionOrderRaw types.JSON) []SafeQuestion {
	var optionOrder map[string][]int
	if err := json.Unmarshal(optionOrderRaw, &optionOrder); err != nil {
		return questions
	}

	for i, q := range questions {
		key := fmt.Sprintf("%d", q.ID)
		indices, ok := optionOrder[key]
		if !ok || len(indices) == 0 {
			continue
		}

		var options []json.RawMessage
		if err := json.Unmarshal(q.Options, &options); err != nil || len(options) == 0 {
			continue
		}

		// Reorder options according to stored indices
		reordered := make([]json.RawMessage, len(indices))
		for newIdx, origIdx := range indices {
			if origIdx < len(options) {
				reordered[newIdx] = options[origIdx]
			}
		}

		reorderedJSON, err := json.Marshal(reordered)
		if err != nil {
			continue
		}
		questions[i].Options = types.JSON(reorderedJSON)
	}
	return questions
}

// makeIndices creates a slice [0, 1, 2, ..., n-1].
func makeIndices(n int) []int {
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}
	return indices
}
