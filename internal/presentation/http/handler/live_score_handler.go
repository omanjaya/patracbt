package handler

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/internal/domain/service"
	"github.com/omanjaya/patra/pkg/ginhelper"
	"github.com/omanjaya/patra/pkg/pagination"
	"github.com/omanjaya/patra/pkg/response"
)

type LiveScoreHandler struct {
	sessionRepo  repository.ExamSessionRepository
	scheduleRepo repository.ExamScheduleRepository
	questionRepo repository.QuestionRepository
	calculator   *service.ScoreCalculator
}

func NewLiveScoreHandler(
	sessionRepo repository.ExamSessionRepository,
	scheduleRepo repository.ExamScheduleRepository,
	questionRepo repository.QuestionRepository,
) *LiveScoreHandler {
	return &LiveScoreHandler{
		sessionRepo:  sessionRepo,
		scheduleRepo: scheduleRepo,
		questionRepo: questionRepo,
		calculator:   service.NewScoreCalculator(),
	}
}

// ── Response DTOs ────────────────────────────────────────────

type LiveScoreRow struct {
	SessionID      uint    `json:"session_id"`
	UserID         uint    `json:"user_id"`
	NIS            string  `json:"nis"`
	Name           string  `json:"name"`
	Rombel         string  `json:"rombel"`
	TotalQuestions int     `json:"total_questions"`
	Answered       int     `json:"answered"`
	Correct        int     `json:"correct"`
	Wrong          int     `json:"wrong"`
	Unanswered     int     `json:"unanswered"`
	Score          float64 `json:"score"`
	MaxScore       float64 `json:"max_score"`
	Percent        float64 `json:"percent"`
	Status         string  `json:"status"`
	ViolationCount int     `json:"violation_count"`
	UpdatedAt      string  `json:"updated_at"`
}

type LiveScoreData struct {
	ScheduleID   uint           `json:"schedule_id"`
	ScheduleName string         `json:"schedule_name"`
	SubjectName  string         `json:"subject_name"`
	StartTime    string         `json:"start_time"`
	EndTime      string         `json:"end_time"`
	Rombels      []RombelInfo   `json:"rombels"`
	Students     []LiveScoreRow `json:"students"`
	Summary      LiveSummary    `json:"summary"`
	Timestamp    string         `json:"timestamp"`
}

type RombelInfo struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type LiveSummary struct {
	TotalParticipants int     `json:"total_participants"`
	Ongoing           int     `json:"ongoing"`
	Finished          int     `json:"finished"`
	NotStarted        int     `json:"not_started"`
	AverageScore      float64 `json:"average_score"`
	HighestScore      float64 `json:"highest_score"`
}

// GetLiveData returns the full live score leaderboard for a schedule.
// GET /admin/live-score/:scheduleId
func (h *LiveScoreHandler) GetLiveData(c *gin.Context) {
	scheduleID, ok := ginhelper.ParseID(c, "scheduleId")
	if !ok {
		return
	}

	// Optional rombel filter (comma-separated)
	rombelFilter := parseUintSlice(c.Query("rombel_ids"))

	data, err := h.buildLiveData(scheduleID, rombelFilter)
	if err != nil {
		response.NotFound(c, "Jadwal ujian tidak ditemukan")
		return
	}

	response.Success(c, data)
}

// GetUpdate returns only sessions updated since the given timestamp.
// GET /admin/live-score/:scheduleId/update?since=2026-03-08T10:00:00Z
func (h *LiveScoreHandler) GetUpdate(c *gin.Context) {
	scheduleID, ok := ginhelper.ParseID(c, "scheduleId")
	if !ok {
		return
	}

	sinceStr := c.Query("since")
	rombelFilter := parseUintSlice(c.Query("rombel_ids"))

	data, err := h.buildLiveData(scheduleID, rombelFilter)
	if err != nil {
		response.NotFound(c, "Jadwal ujian tidak ditemukan")
		return
	}

	// If "since" is provided, filter to only changed rows
	if sinceStr != "" {
		since, parseErr := time.Parse(time.RFC3339, sinceStr)
		if parseErr == nil {
			filtered := make([]LiveScoreRow, 0)
			for _, row := range data.Students {
				rowTime, _ := time.Parse(time.RFC3339, row.UpdatedAt)
				if rowTime.After(since) {
					filtered = append(filtered, row)
				}
			}
			data.Students = filtered
		}
	}

	response.Success(c, data)
}

// ── Internal helpers ─────────────────────────────────────────

func (h *LiveScoreHandler) buildLiveData(scheduleID uint, rombelFilter []uint) (*LiveScoreData, error) {
	schedule, err := h.scheduleRepo.FindByID(scheduleID)
	if err != nil {
		return nil, err
	}

	// Collect rombel info from schedule
	rombels := make([]RombelInfo, 0)
	for _, sr := range schedule.Rombels {
		rombels = append(rombels, RombelInfo{
			ID:   sr.RombelID,
			Name: sr.Rombel.Name,
		})
	}

	// Collect all questions from question banks (loaded once, avoids N+1)
	questionMap := make(map[uint]*entity.Question)
	totalQuestions := 0
	for _, sqb := range schedule.QuestionBanks {
		questions, qErr := h.questionRepo.ListAllByBank(sqb.QuestionBankID)
		if qErr != nil {
			continue
		}
		for _, q := range questions {
			questionMap[q.ID] = q
		}
		totalQuestions += len(questions)
	}

	// Load all sessions (paginated to avoid huge loads)
	var sessions []*entity.ExamSession
	for page := 1; ; page++ {
		p := pagination.Params{Page: page, PerPage: 100}
		batch, _, bErr := h.sessionRepo.ListBySchedule(scheduleID, p)
		if bErr != nil {
			return nil, bErr
		}
		sessions = append(sessions, batch...)
		if len(batch) < 100 {
			break
		}
	}

	// Load all answers for the schedule in one query (avoids N+1)
	allAnswers, err := h.sessionRepo.GetAllAnswersBySchedule(scheduleID)
	if err != nil {
		return nil, err
	}

	// Get rombel names for all user IDs
	userIDs := make([]uint, 0, len(sessions))
	for _, s := range sessions {
		userIDs = append(userIDs, s.UserID)
	}
	userRombelMap, _ := h.sessionRepo.GetUserRombelNames(userIDs)

	// Build rows
	rows := make([]LiveScoreRow, 0, len(sessions))
	var sumScore float64
	var highestScore float64
	ongoing, finished, notStarted := 0, 0, 0

	for _, s := range sessions {
		// Rombel filter
		rombelName := "-"
		if names, ok := userRombelMap[s.UserID]; ok && len(names) > 0 {
			rombelName = strings.Join(names, ", ")
		}

		if len(rombelFilter) > 0 {
			// Check if user's rombel is in filter
			userRombelIDs, _ := h.sessionRepo.GetUserRombelIDs(s.UserID)
			matched := false
			for _, rid := range userRombelIDs {
				for _, fid := range rombelFilter {
					if rid == fid {
						matched = true
						break
					}
				}
				if matched {
					break
				}
			}
			if !matched {
				continue
			}
		}

		// Get question order from session
		var questionOrder []uint
		_ = json.Unmarshal(s.QuestionOrder, &questionOrder)
		sessionTotalQ := len(questionOrder)
		if sessionTotalQ == 0 {
			sessionTotalQ = totalQuestions
		}

		// Calculate score from answers
		answers := allAnswers[s.ID]
		answeredCount := 0
		correctCount := 0
		wrongCount := 0
		earnedScore := 0.0
		maxScore := 0.0

		// Build answer map
		answerMap := make(map[uint]*entity.ExamAnswer)
		for i := range answers {
			answerMap[answers[i].QuestionID] = &answers[i]
		}

		// Iterate through session's question order for accurate calculation
		questionsToCheck := questionOrder
		if len(questionsToCheck) == 0 {
			// Fallback: use all questions from question map
			for qid := range questionMap {
				questionsToCheck = append(questionsToCheck, qid)
			}
		}

		for _, qid := range questionsToCheck {
			q, exists := questionMap[qid]
			if !exists {
				continue
			}

			qScore := q.Score
			if qScore <= 0 {
				qScore = 1
			}
			maxScore += qScore

			ans, hasAns := answerMap[qid]
			if !hasAns || ans.Answer == nil || len(ans.Answer) == 0 || string(ans.Answer) == "null" {
				continue
			}

			answeredCount++

			// Calculate score using score calculator
			earned := h.calculator.Calculate(q, ans.Answer)
			if earned < 0 {
				earned = 0
			}
			if earned > qScore {
				earned = qScore
			}
			earnedScore += earned

			if earned >= qScore {
				correctCount++
			} else {
				wrongCount++
			}
		}

		percent := 0.0
		if maxScore > 0 {
			percent = (earnedScore / maxScore) * 100
		}

		// For finished sessions, prefer the stored score if it looks valid
		displayScore := earnedScore
		displayMax := maxScore
		displayPercent := percent
		if (s.Status == entity.SessionStatusFinished || s.Status == entity.SessionStatusTerminated) && s.MaxScore > 0 {
			displayScore = s.Score
			displayMax = s.MaxScore
			displayPercent = 0
			if displayMax > 0 {
				displayPercent = (displayScore / displayMax) * 100
			}
		}

		// Status counters
		switch s.Status {
		case entity.SessionStatusOngoing:
			ongoing++
		case entity.SessionStatusFinished, entity.SessionStatusTerminated:
			finished++
		case entity.SessionStatusNotStarted:
			notStarted++
		}

		userName := "Unknown"
		username := ""
		nis := "-"
		if s.User.ID != 0 {
			userName = s.User.Name
			username = s.User.Username
			if s.User.Profile != nil && s.User.Profile.NIS != nil {
				nis = *s.User.Profile.NIS
			} else {
				nis = username
			}
		}

		row := LiveScoreRow{
			SessionID:      s.ID,
			UserID:         s.UserID,
			NIS:            nis,
			Name:           userName,
			Rombel:         rombelName,
			TotalQuestions: sessionTotalQ,
			Answered:       answeredCount,
			Correct:        correctCount,
			Wrong:          wrongCount,
			Unanswered:     sessionTotalQ - answeredCount,
			Score:          roundTo(displayScore, 2),
			MaxScore:       roundTo(displayMax, 2),
			Percent:        roundTo(displayPercent, 2),
			Status:         s.Status,
			ViolationCount: s.ViolationCount,
			UpdatedAt:      s.UpdatedAt.Format(time.RFC3339),
		}
		rows = append(rows, row)
		sumScore += displayPercent
		if displayPercent > highestScore {
			highestScore = displayPercent
		}
	}

	// Sort by percent descending (leaderboard)
	sortRows(rows)

	avg := 0.0
	if len(rows) > 0 {
		avg = sumScore / float64(len(rows))
	}

	// Subject name from first question bank
	subjectName := ""
	if len(schedule.QuestionBanks) > 0 && schedule.QuestionBanks[0].QuestionBank.Subject != nil {
		subjectName = schedule.QuestionBanks[0].QuestionBank.Subject.Name
	}

	return &LiveScoreData{
		ScheduleID:   schedule.ID,
		ScheduleName: schedule.Name,
		SubjectName:  subjectName,
		StartTime:    schedule.StartTime.Format(time.RFC3339),
		EndTime:      schedule.EndTime.Format(time.RFC3339),
		Rombels:      rombels,
		Students:     rows,
		Summary: LiveSummary{
			TotalParticipants: len(rows),
			Ongoing:           ongoing,
			Finished:          finished,
			NotStarted:        notStarted,
			AverageScore:      roundTo(avg, 2),
			HighestScore:      roundTo(highestScore, 2),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}, nil
}

func sortRows(rows []LiveScoreRow) {
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0 && rows[j].Percent > rows[j-1].Percent; j-- {
			rows[j], rows[j-1] = rows[j-1], rows[j]
		}
	}
}

func roundTo(val float64, decimals int) float64 {
	pow := 1.0
	for i := 0; i < decimals; i++ {
		pow *= 10
	}
	return float64(int(val*pow+0.5)) / pow
}

func parseUintSlice(s string) []uint {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]uint, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		v, err := strconv.ParseUint(p, 10, 64)
		if err == nil {
			result = append(result, uint(v))
		}
	}
	return result
}
