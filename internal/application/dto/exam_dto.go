package dto

import (
	"encoding/json"
	"time"
)

// Exam Schedule DTOs

type CreateExamScheduleRequest struct {
	Name                 string    `json:"name" binding:"required,min=1,max=255"`
	StartTime            time.Time `json:"start_time" binding:"required"`
	EndTime              time.Time `json:"end_time" binding:"required"`
	DurationMinutes      int       `json:"duration_minutes" binding:"required,min=1,max=1440"`
	AllowSeeResult       *bool     `json:"allow_see_result"`
	MaxViolations        int       `json:"max_violations" binding:"omitempty,min=0"`
	RandomizeQuestions   bool      `json:"randomize_questions"`
	RandomizeOptions     bool      `json:"randomize_options"`
	NextExamScheduleID   *uint     `json:"next_exam_schedule_id"`
	LatePolicy           string    `json:"late_policy" binding:"omitempty,oneof=deduct_time allow_full_time"`
	MinWorkingTime       int       `json:"min_working_time" binding:"omitempty,min=0"`
	DetectCheating       *bool     `json:"detect_cheating"`
	CheatingLimit        int       `json:"cheating_limit" binding:"omitempty,min=0"`
	ShowScoreAfter       string    `json:"show_score_after" binding:"omitempty,oneof=immediately after_end_time manual"`
	QuestionBanks        []ExamScheduleBankInput `json:"question_banks"`
	RombelIDs            []uint  `json:"rombel_ids"`
	TagIDs               []uint  `json:"tag_ids"`
	IncludeUsers         []uint  `json:"include_users"` // whitelist user IDs
	ExcludeUsers         []uint  `json:"exclude_users"` // blacklist user IDs
}

type ExamScheduleBankInput struct {
	QuestionBankID uint    `json:"question_bank_id" binding:"required,min=1"`
	QuestionCount  int     `json:"question_count" binding:"omitempty,min=0"` // 0 = all
	Weight         float64 `json:"weight" binding:"omitempty,min=0"`         // default 1
}

type UpdateExamScheduleRequest struct {
	Name                 string    `json:"name" binding:"required,min=1,max=255"`
	StartTime            time.Time `json:"start_time" binding:"required"`
	EndTime              time.Time `json:"end_time" binding:"required"`
	DurationMinutes      int       `json:"duration_minutes" binding:"required,min=1,max=1440"`
	AllowSeeResult       *bool     `json:"allow_see_result"`
	MaxViolations        int       `json:"max_violations" binding:"omitempty,min=0"`
	RandomizeQuestions   bool      `json:"randomize_questions"`
	RandomizeOptions     bool      `json:"randomize_options"`
	NextExamScheduleID   *uint     `json:"next_exam_schedule_id"`
	LatePolicy           string    `json:"late_policy" binding:"omitempty,oneof=deduct_time allow_full_time"`
	MinWorkingTime       int       `json:"min_working_time" binding:"omitempty,min=0"`
	DetectCheating       *bool     `json:"detect_cheating"`
	CheatingLimit        int       `json:"cheating_limit" binding:"omitempty,min=0"`
	ShowScoreAfter       string    `json:"show_score_after" binding:"omitempty,oneof=immediately after_end_time manual"`
	QuestionBanks        []ExamScheduleBankInput `json:"question_banks"`
	RombelIDs            []uint  `json:"rombel_ids"`
	TagIDs               []uint  `json:"tag_ids"`
	IncludeUsers         []uint  `json:"include_users"` // whitelist user IDs
	ExcludeUsers         []uint  `json:"exclude_users"` // blacklist user IDs
}

// ExamScheduleResponse is the response DTO for exam schedule details.
type ExamScheduleResponse struct {
	ID                   uint                   `json:"id"`
	Name                 string                 `json:"name"`
	Token                string                 `json:"token,omitempty"`
	StartTime            time.Time              `json:"start_time"`
	EndTime              time.Time              `json:"end_time"`
	DurationMinutes      int                    `json:"duration_minutes"`
	Status               string                 `json:"status"`
	AllowSeeResult       bool                   `json:"allow_see_result"`
	MaxViolations        int                    `json:"max_violations"`
	RandomizeQuestions   bool                   `json:"randomize_questions"`
	RandomizeOptions     bool                   `json:"randomize_options"`
	LatePolicy           string                 `json:"late_policy"`
	MinWorkingTime       int                    `json:"min_working_time"`
	DetectCheating       bool                   `json:"detect_cheating"`
	CheatingLimit        int                    `json:"cheating_limit"`
	ShowScoreAfter       string                 `json:"show_score_after"`
	NextExamScheduleID   *uint                  `json:"next_exam_schedule_id"`
	CreatedAt            time.Time              `json:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
}

// ExamSessionResponse is the response DTO for exam session details.
type ExamSessionResponse struct {
	ID             uint                `json:"id"`
	ExamScheduleID uint                `json:"exam_schedule_id"`
	UserID         uint                `json:"user_id"`
	Status         string              `json:"status"`
	StartTime      *time.Time          `json:"start_time"`
	EndTime        *time.Time          `json:"end_time"`
	FinishedAt     *time.Time          `json:"finished_at"`
	QuestionOrder  json.RawMessage     `json:"question_order"`
	OptionOrder    json.RawMessage     `json:"option_order,omitempty"`
	Score          float64             `json:"score"`
	MaxScore       float64             `json:"max_score"`
	ViolationCount int                 `json:"violation_count"`
	ExtraTime      int                 `json:"extra_time"`
	SectionIndex   int                 `json:"section_index"`
	MinWorkingTime int                 `json:"min_working_time"`
	ShowScoreAfter string              `json:"show_score_after"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
}

// Exam Session DTOs

type StartExamRequest struct {
	ExamScheduleID uint   `json:"exam_schedule_id" binding:"required"`
	Token          string `json:"token" binding:"required"`
}

type SaveAnswerRequest struct {
	QuestionID uint            `json:"question_id" binding:"required"`
	Answer     json.RawMessage `json:"answer"`
	IsFlagged  bool            `json:"is_flagged"`
}

type LogViolationRequest struct {
	ViolationType string `json:"violation_type" binding:"required,max=100"`
	Description   string `json:"description" binding:"omitempty,max=500"`
}

type ToggleFlagRequest struct {
	QuestionID uint `json:"question_id" binding:"required"`
	IsFlagged  bool `json:"is_flagged"`
}

type BeaconSyncRequest struct {
	Answers []SaveAnswerRequest `json:"answers" binding:"required"`
}
