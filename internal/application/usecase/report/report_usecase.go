package report

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/internal/domain/service"
	"github.com/omanjaya/patra/pkg/pagination"
	"github.com/omanjaya/patra/pkg/types"
)

// ─── Response types ────────────────────────────────────────────

type SessionRow struct {
	SessionID      uint       `json:"session_id"`
	UserID         uint       `json:"user_id"`
	UserName       string     `json:"user_name"`
	Username       string     `json:"username"`
	Score          float64    `json:"score"`
	MaxScore       float64    `json:"max_score"`
	Percent        float64    `json:"percent"`
	Status         string     `json:"status"`
	AnsweredCount  int        `json:"answered_count"`
	TotalQuestions int        `json:"total_questions"`
	ViolationCount int        `json:"violation_count"`
	Duration       int        `json:"duration_seconds"`
	StartTime      *time.Time `json:"start_time"`
	FinishedAt     *time.Time `json:"finished_at"`
}

type ScheduleReport struct {
	ScheduleID uint         `json:"schedule_id"`
	Name       string       `json:"schedule_name"`
	Sessions   []SessionRow `json:"sessions"`
	Stats      ReportStats  `json:"stats"`
}

type ReportStats struct {
	Total     int     `json:"total"`
	Finished  int     `json:"finished"`
	Mean      float64 `json:"mean"`
	Median    float64 `json:"median"`
	Mode      float64 `json:"mode"`
	StdDev    float64 `json:"std_dev"`
	Highest   float64 `json:"highest"`
	Lowest    float64 `json:"lowest"`
}

type AnswerDetail struct {
	QuestionID   uint       `json:"question_id"`
	QuestionType string     `json:"question_type"`
	Body         string     `json:"body"`
	Score        float64    `json:"score"`
	Options      types.JSON `json:"options"`
	CorrectAnswer types.JSON `json:"correct_answer"`
	UserAnswer   types.JSON `json:"user_answer"`
	EarnedScore  float64    `json:"earned_score"`
	IsFlagged    bool       `json:"is_flagged"`
	IsCorrect    bool       `json:"is_correct"`
}

type PersonalReport struct {
	Session entity.ExamSession `json:"session"`
	Answers []AnswerDetail     `json:"answers"`
}

type QuestionAnalysis struct {
	QuestionID          uint    `json:"question_id"`
	Body                string  `json:"body"`
	QuestionType        string  `json:"question_type"`
	DifficultyIndex     float64 `json:"difficulty_index"` // p-value: correct/total
	DiscriminationIndex float64 `json:"discrimination_index"` // D-value
	Quality             string  `json:"quality"` // "Baik", "Cukup", "Revisi", "Buang"
}

type ExamAnalysis struct {
	ScheduleID uint               `json:"schedule_id"`
	Questions  []QuestionAnalysis `json:"questions"`
	Stats      ReportStats        `json:"stats"`
}

// ─── UseCase ──────────────────────────────────────────────────

type ReportUseCase struct {
	sessionRepo  repository.ExamSessionRepository
	scheduleRepo repository.ExamScheduleRepository
	questionRepo repository.QuestionRepository
	calculator   *service.ScoreCalculator
}

func NewReportUseCase(
	sessionRepo repository.ExamSessionRepository,
	scheduleRepo repository.ExamScheduleRepository,
	questionRepo repository.QuestionRepository,
) *ReportUseCase {
	return &ReportUseCase{
		sessionRepo:  sessionRepo,
		scheduleRepo: scheduleRepo,
		questionRepo: questionRepo,
		calculator:   service.NewScoreCalculator(),
	}
}

// GetScheduleReport returns all session results for a schedule.
func (uc *ReportUseCase) GetScheduleReport(scheduleID uint) (*ScheduleReport, error) {
	schedule, err := uc.scheduleRepo.FindByID(scheduleID)
	if err != nil {
		return nil, err
	}

	// Fetch all sessions in batches of 100 to avoid loading huge result sets at once
	var sessions []*entity.ExamSession
	for page := 1; ; page++ {
		p := pagination.Params{Page: page, PerPage: 100}
		batch, _, err := uc.sessionRepo.ListBySchedule(scheduleID, p)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, batch...)
		if len(batch) < 100 {
			break
		}
	}

	// Single query to fetch all answers for this schedule — avoids N+1
	allAnswers, err := uc.sessionRepo.GetAllAnswersBySchedule(scheduleID)
	if err != nil {
		return nil, err
	}

	rows := make([]SessionRow, 0, len(sessions))
	for _, s := range sessions {
		answers := allAnswers[s.ID]

		var order []uint
		_ = json.Unmarshal(s.QuestionOrder, &order)

		dur := 0
		if s.StartTime != nil && s.FinishedAt != nil {
			dur = int(s.FinishedAt.Sub(*s.StartTime).Seconds())
		}

		pct := 0.0
		if s.MaxScore > 0 {
			pct = (s.Score / s.MaxScore) * 100
		}

		userName := "Unknown"
		username := ""
		if s.User.ID != 0 {
			userName = s.User.Name
			username = s.User.Username
		}

		rows = append(rows, SessionRow{
			SessionID:      s.ID,
			UserID:         s.UserID,
			UserName:       userName,
			Username:       username,
			Score:          s.Score,
			MaxScore:       s.MaxScore,
			Percent:        pct,
			Status:         s.Status,
			AnsweredCount:  len(answers),
			TotalQuestions: len(order),
			ViolationCount: s.ViolationCount,
			Duration:       dur,
			StartTime:      s.StartTime,
			FinishedAt:     s.FinishedAt,
		})
	}

	stats := computeStats(rows)
	return &ScheduleReport{
		ScheduleID: schedule.ID,
		Name:       schedule.Name,
		Sessions:   rows,
		Stats:      stats,
	}, nil
}

// GetPersonalReport returns detailed answer analysis for a session.
func (uc *ReportUseCase) GetPersonalReport(sessionID uint) (*PersonalReport, error) {
	session, err := uc.sessionRepo.FindByID(sessionID)
	if err != nil {
		return nil, err
	}

	var order []uint
	_ = json.Unmarshal(session.QuestionOrder, &order)

	answers, err := uc.sessionRepo.GetAllAnswers(sessionID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil jawaban: %w", err)
	}
	answerMap := make(map[uint]entity.ExamAnswer)
	for _, a := range answers {
		answerMap[a.QuestionID] = a
	}

	// Batch-load all questions in a single query — avoids N+1
	fetched, err := uc.questionRepo.FindByIDs(order)
	if err != nil {
		return nil, err
	}
	qMap := make(map[uint]*entity.Question, len(fetched))
	for _, q := range fetched {
		qMap[q.ID] = q
	}

	details := make([]AnswerDetail, 0, len(order))
	for _, qID := range order {
		q, ok := qMap[qID]
		if !ok {
			continue
		}

		d := AnswerDetail{
			QuestionID:    q.ID,
			QuestionType:  q.QuestionType,
			Body:          q.Body,
			Score:         q.Score,
			Options:       q.Options,
			CorrectAnswer: q.CorrectAnswer,
		}

		if a, ok := answerMap[qID]; ok {
			d.UserAnswer = a.Answer
			d.IsFlagged = a.IsFlagged
			earned := uc.calculator.Calculate(q, a.Answer)
			d.EarnedScore = earned
			d.IsCorrect = earned >= q.Score*0.5 // >50% of score = correct enough
		}

		details = append(details, d)
	}

	return &PersonalReport{Session: *session, Answers: details}, nil
}

// GetExamAnalysis returns per-question difficulty and discrimination analysis.
func (uc *ReportUseCase) GetExamAnalysis(scheduleID uint) (*ExamAnalysis, error) {
	// Fetch all sessions in batches of 100
	var sessions []*entity.ExamSession
	for page := 1; ; page++ {
		p := pagination.Params{Page: page, PerPage: 100}
		batch, _, err := uc.sessionRepo.ListBySchedule(scheduleID, p)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, batch...)
		if len(batch) < 100 {
			break
		}
	}

	// BUG-08 fix: gabungkan (union) semua question ID dari SEMUA session yang finished
	// agar soal acak (RandomizeQuestions=true) tetap teranalisis semua
	var questionOrder []uint
	questionSet := make(map[uint]struct{})
	for _, s := range sessions {
		if s.Status == entity.SessionStatusFinished && len(s.QuestionOrder) > 2 {
			var order []uint
			if err := json.Unmarshal(s.QuestionOrder, &order); err == nil {
				for _, qID := range order {
					questionSet[qID] = struct{}{}
				}
			}
		}
	}
	// Ubah set ke slice
	for qID := range questionSet {
		questionOrder = append(questionOrder, qID)
	}

	if len(questionOrder) == 0 {
		// Get from schedule banks
		schedule, _ := uc.scheduleRepo.FindByID(scheduleID)
		if schedule != nil {
			for _, bankRef := range schedule.QuestionBanks {
				qs, _, _ := uc.questionRepo.ListByBank(bankRef.QuestionBankID, pagination.Params{Page: 1, PerPage: 100})
				for _, q := range qs {
					questionOrder = append(questionOrder, q.ID)
				}
			}
		}
	}

	// Build score map: sessionID → score (percent)
	finishedSessions := make([]*entity.ExamSession, 0)
	for _, s := range sessions {
		if s.Status == entity.SessionStatusFinished {
			finishedSessions = append(finishedSessions, s)
		}
	}

	// Batch-load all answers for this schedule in a single query — avoids N+1
	allAnswers, err := uc.sessionRepo.GetAllAnswersBySchedule(scheduleID)
	if err != nil {
		return nil, err
	}

	// Batch-load all questions in a single query — avoids N+1
	allQuestions, err := uc.questionRepo.FindByIDs(questionOrder)
	if err != nil {
		return nil, err
	}
	qMap := make(map[uint]*entity.Question, len(allQuestions))
	for _, q := range allQuestions {
		qMap[q.ID] = q
	}

	// For each question, calculate correct count and scores
	type qStat struct {
		total      int
		correct    int
		correctMap []int     // per-session: 1 if correct, 0 if not (for discrimination)
		scores     []float64 // per-session overall score percent (for sorting by ability)
	}
	qStats := make(map[uint]*qStat)
	sessionScores := make(map[uint]float64) // sessionID → percent score

	for _, s := range finishedSessions {
		pct := 0.0
		if s.MaxScore > 0 {
			pct = s.Score / s.MaxScore
		}
		sessionScores[s.ID] = pct
	}

	for _, s := range finishedSessions {
		answers := allAnswers[s.ID]
		ansMap := make(map[uint]entity.ExamAnswer, len(answers))
		for _, a := range answers {
			ansMap[a.QuestionID] = a
		}

		for _, qID := range questionOrder {
			q, ok := qMap[qID]
			if !ok {
				continue
			}
			if _, ok := qStats[qID]; !ok {
				qStats[qID] = &qStat{}
			}
			qStats[qID].total++
			isCorrect := 0
			if a, ok := ansMap[qID]; ok {
				earned := uc.calculator.Calculate(q, a.Answer)
				if earned >= q.Score*0.5 {
					qStats[qID].correct++
					isCorrect = 1
				}
			}
			qStats[qID].correctMap = append(qStats[qID].correctMap, isCorrect)
			qStats[qID].scores = append(qStats[qID].scores, sessionScores[s.ID])
		}
	}

	// Build analysis using already-loaded qMap (no extra queries)
	analyses := make([]QuestionAnalysis, 0, len(questionOrder))
	for _, qID := range questionOrder {
		q, ok := qMap[qID]
		if !ok {
			continue
		}
		stat := qStats[qID]
		p := 0.0
		d := 0.0
		if stat != nil && stat.total > 0 {
			p = float64(stat.correct) / float64(stat.total)
			d = computeDiscrimination(stat.scores, stat.correctMap, stat.total)
		}

		qa := QuestionAnalysis{
			QuestionID:          q.ID,
			Body:                q.Body,
			QuestionType:        q.QuestionType,
			DifficultyIndex:     math.Round(p*1000) / 1000,
			DiscriminationIndex: math.Round(d*1000) / 1000,
			Quality:             qualityLabel(p, d),
		}
		analyses = append(analyses, qa)
	}

	// Get rows for stats
	rows := make([]SessionRow, 0, len(finishedSessions))
	for _, s := range finishedSessions {
		pct := 0.0
		if s.MaxScore > 0 {
			pct = s.Score / s.MaxScore * 100
		}
		rows = append(rows, SessionRow{Score: pct})
	}

	return &ExamAnalysis{
		ScheduleID: scheduleID,
		Questions:  analyses,
		Stats:      computeStats(rows),
	}, nil
}

// RegradeSummary holds the result of a regrade operation.
type RegradeSummary struct {
	TotalSessions int                  `json:"total_sessions"`
	ScoreChanges  []entity.ScoreChange `json:"score_changes"`
}

// RegradeSchedule recalculates all session scores for a schedule and logs changes.
func (uc *ReportUseCase) RegradeSchedule(scheduleID, requestedBy uint) (*RegradeSummary, error) {
	// Fetch all sessions in batches of 100
	var sessions []*entity.ExamSession
	for page := 1; ; page++ {
		p := pagination.Params{Page: page, PerPage: 100}
		batch, _, err := uc.sessionRepo.ListBySchedule(scheduleID, p)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, batch...)
		if len(batch) < 100 {
			break
		}
	}

	// Single query for all answers — avoids N+1
	allAnswers, err := uc.sessionRepo.GetAllAnswersBySchedule(scheduleID)
	if err != nil {
		return nil, err
	}

	// Collect all question IDs across finished/terminated sessions
	questionSet := make(map[uint]struct{})
	for _, s := range sessions {
		if s.Status != entity.SessionStatusFinished && s.Status != entity.SessionStatusTerminated {
			continue
		}
		var order []uint
		if err := json.Unmarshal(s.QuestionOrder, &order); err == nil {
			for _, qID := range order {
				questionSet[qID] = struct{}{}
			}
		}
	}
	allQIDs := make([]uint, 0, len(questionSet))
	for qID := range questionSet {
		allQIDs = append(allQIDs, qID)
	}

	// Batch-load all questions in a single query — avoids N+1
	allQuestions, err := uc.questionRepo.FindByIDs(allQIDs)
	if err != nil {
		return nil, err
	}
	qMap := make(map[uint]*entity.Question, len(allQuestions))
	for _, q := range allQuestions {
		qMap[q.ID] = q
	}

	summary := &RegradeSummary{}
	for _, s := range sessions {
		if s.Status != entity.SessionStatusFinished && s.Status != entity.SessionStatusTerminated {
			continue
		}

		var order []uint
		if err := json.Unmarshal(s.QuestionOrder, &order); err != nil {
			continue
		}

		answers := allAnswers[s.ID]
		ansMap := make(map[uint]entity.ExamAnswer, len(answers))
		for _, a := range answers {
			ansMap[a.QuestionID] = a
		}

		originalScore := s.Score
		var score, maxScore float64
		for _, qID := range order {
			q, ok := qMap[qID]
			if !ok {
				continue
			}
			maxScore += q.Score
			if a, ok := ansMap[qID]; ok {
				earned := uc.calculator.Calculate(q, a.Answer)
				// Also check manual_score for essay questions
				if q.QuestionType == entity.QuestionTypeEsai {
					var ansData map[string]any
					if err := json.Unmarshal(a.Answer, &ansData); err == nil {
						if ms, ok := ansData["manual_score"].(float64); ok {
							earned = ms
						}
					}
				}
				score += earned
			}
		}

		s.Score = score
		s.MaxScore = maxScore
		if err := uc.sessionRepo.Update(s); err == nil {
			summary.TotalSessions++
			if originalScore != score {
				summary.ScoreChanges = append(summary.ScoreChanges, entity.ScoreChange{
					SessionID: s.ID,
					OldScore:  originalScore,
					NewScore:  score,
				})
			}
		}
	}

	// Create RegradeLog entry
	changesJSON, _ := json.Marshal(summary.ScoreChanges)
	regradeLog := &entity.RegradeLog{
		ExamScheduleID: scheduleID,
		RequestedBy:    requestedBy,
		SessionsCount:  summary.TotalSessions,
		ScoreChanges:   changesJSON,
	}
	_ = uc.sessionRepo.CreateRegradeLog(regradeLog)

	// Update schedule's LastGradedAt
	schedule, err := uc.scheduleRepo.FindByID(scheduleID)
	if err == nil && schedule != nil {
		now := time.Now()
		schedule.LastGradedAt = &now
		_ = uc.scheduleRepo.Update(schedule)
	}

	return summary, nil
}

// KeyChange represents a question whose answer key was changed after last grading.
type KeyChange struct {
	QuestionID     uint      `json:"question_id"`
	QuestionNumber int       `json:"question_number"`
	Body           string    `json:"body"`
	QuestionType   string    `json:"question_type"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// GetKeyChanges returns questions whose answer keys changed after the schedule's last grading.
func (uc *ReportUseCase) GetKeyChanges(scheduleID uint) ([]KeyChange, error) {
	schedule, err := uc.scheduleRepo.FindByID(scheduleID)
	if err != nil {
		return nil, err
	}

	if schedule.LastGradedAt == nil {
		return []KeyChange{}, nil
	}

	// Load all questions for all banks in the schedule using paginated batches
	var allQuestions []*entity.Question
	for _, bankRef := range schedule.QuestionBanks {
		for page := 1; ; page++ {
			p := pagination.Params{Page: page, PerPage: 100}
			batch, _, err := uc.questionRepo.ListByBank(bankRef.QuestionBankID, p)
			if err != nil {
				break
			}
			allQuestions = append(allQuestions, batch...)
			if len(batch) < 100 {
				break
			}
		}
	}

	changes := make([]KeyChange, 0)
	for i, q := range allQuestions {
		if q.UpdatedAt.After(*schedule.LastGradedAt) {
			changes = append(changes, KeyChange{
				QuestionID:     q.ID,
				QuestionNumber: i + 1,
				Body:           q.Body,
				QuestionType:   q.QuestionType,
				UpdatedAt:      q.UpdatedAt,
			})
		}
	}

	return changes, nil
}

// ListRegradeLogs returns all regrade log entries for a schedule.
func (uc *ReportUseCase) ListRegradeLogs(scheduleID uint) ([]entity.RegradeLog, error) {
	return uc.sessionRepo.ListRegradeLogs(scheduleID)
}

// SetEssayScore manually sets score for a specific answer (AI or manual grading).
// BUG-17 fix: recalculate session total score setelah grading
func (uc *ReportUseCase) SetEssayScore(sessionID, questionID uint, score float64) error {
	answer, err := uc.sessionRepo.GetAnswer(sessionID, questionID)
	if err != nil {
		return err
	}

	// Encode score into answer JSON
	var current map[string]any
	_ = json.Unmarshal(answer.Answer, &current)
	if current == nil {
		current = map[string]any{}
	}
	current["manual_score"] = score
	updated, _ := json.Marshal(current)
	answer.Answer = types.JSON(updated)

	if err := uc.sessionRepo.UpsertAnswer(answer); err != nil {
		return err
	}

	// BUG-17 fix: recalculate total session score
	return uc.recalcSessionScore(sessionID)
}

// recalcSessionScore recalculates and persists the total score for a session.
func (uc *ReportUseCase) recalcSessionScore(sessionID uint) error {
	session, err := uc.sessionRepo.FindByID(sessionID)
	if err != nil {
		return err
	}

	var order []uint
	if err := json.Unmarshal(session.QuestionOrder, &order); err != nil || len(order) == 0 {
		return nil // tidak ada soal, skip
	}

	answers, err := uc.sessionRepo.GetAllAnswers(sessionID)
	if err != nil {
		return fmt.Errorf("gagal mengambil jawaban: %w", err)
	}
	ansMap := make(map[uint]entity.ExamAnswer)
	for _, a := range answers {
		ansMap[a.QuestionID] = a
	}

	// Batch-load all questions in a single query — avoids N+1
	fetched, err := uc.questionRepo.FindByIDs(order)
	if err != nil {
		return err
	}
	qMap := make(map[uint]*entity.Question, len(fetched))
	for _, q := range fetched {
		qMap[q.ID] = q
	}

	var totalScore, maxScore float64
	for _, qID := range order {
		q, ok := qMap[qID]
		if !ok {
			continue
		}
		maxScore += q.Score
		if a, ok := ansMap[qID]; ok {
			earned := uc.calculator.Calculate(q, a.Answer)
			// Juga cek manual_score untuk esai
			if q.QuestionType == entity.QuestionTypeEsai {
				var ansData map[string]any
				if err := json.Unmarshal(a.Answer, &ansData); err == nil {
					if ms, ok := ansData["manual_score"].(float64); ok {
						earned = ms
					}
				}
			}
			totalScore += earned
		}
	}

	session.Score = totalScore
	session.MaxScore = maxScore
	return uc.sessionRepo.Update(session)
}

// ─── Helpers ──────────────────────────────────────────────────

// BUG-18 fix: hanya hitung row yang secara eksplisit berstatus finished (bukan status kosong "")
func computeStats(rows []SessionRow) ReportStats {
	if len(rows) == 0 {
		return ReportStats{}
	}
	finished := 0
	scores := []float64{}
	for _, r := range rows {
		if r.Status == entity.SessionStatusFinished {
			finished++
			scores = append(scores, r.Score)
		}
	}
	if len(scores) == 0 {
		return ReportStats{Total: len(rows), Finished: finished}
	}

	sort.Float64s(scores)
	sum := 0.0
	for _, s := range scores {
		sum += s
	}
	mean := sum / float64(len(scores))

	var median float64
	n := len(scores)
	if n%2 == 0 {
		median = (scores[n/2-1] + scores[n/2]) / 2
	} else {
		median = scores[n/2]
	}

	variance := 0.0
	for _, s := range scores {
		d := s - mean
		variance += d * d
	}
	stdDev := math.Sqrt(variance / float64(len(scores)))

	// Mode: most frequent score (rounded to 2 decimals for grouping)
	freqMap := make(map[float64]int)
	for _, s := range scores {
		rounded := math.Round(s*100) / 100
		freqMap[rounded]++
	}
	mode := scores[0]
	maxFreq := 0
	for val, freq := range freqMap {
		if freq > maxFreq {
			maxFreq = freq
			mode = val
		}
	}

	return ReportStats{
		Total:    len(rows),
		Finished: finished,
		Mean:     math.Round(mean*100) / 100,
		Median:   math.Round(median*100) / 100,
		Mode:     mode,
		StdDev:   math.Round(stdDev*100) / 100,
		Highest:  scores[len(scores)-1],
		Lowest:   scores[0],
	}
}

// BUG-09 fix: discrimination index menggunakan proporsi benar di upper vs lower 27%
// scores = overall test score per student (for ranking by ability)
// correctFlags = per-student 1/0 whether they got THIS question correct
// D = (proportion correct in top 27%) - (proportion correct in bottom 27%)
func computeDiscrimination(scores []float64, correctFlags []int, total int) float64 {
	if total < 6 || len(scores) != total || len(correctFlags) != total {
		return 0
	}

	// Pair each student's overall score with their correctness on this question
	type pair struct {
		score   float64
		correct int
	}
	pairs := make([]pair, total)
	for i := 0; i < total; i++ {
		pairs[i] = pair{score: scores[i], correct: correctFlags[i]}
	}

	// Sort by overall score ascending
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].score < pairs[j].score
	})

	k := int(math.Ceil(float64(total) * 0.27))
	if k == 0 {
		return 0
	}

	// Upper group = top 27% by overall score
	upperCorrect := 0
	for _, p := range pairs[total-k:] {
		upperCorrect += p.correct
	}

	// Lower group = bottom 27% by overall score
	lowerCorrect := 0
	for _, p := range pairs[:k] {
		lowerCorrect += p.correct
	}

	// D = proportion correct in upper - proportion correct in lower
	return float64(upperCorrect)/float64(k) - float64(lowerCorrect)/float64(k)
}

func qualityLabel(p, d float64) string {
	if p < 0.2 || p > 0.8 {
		if d < 0.2 {
			return "Buang"
		}
		return "Revisi"
	}
	if d >= 0.4 {
		return "Baik Sekali"
	}
	if d >= 0.3 {
		return "Baik"
	}
	if d >= 0.2 {
		return "Cukup"
	}
	return "Revisi"
}
