package dto

import "encoding/json"

type CreateQuestionBankRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=255"`
	SubjectID   *uint  `json:"subject_id"`
	Description string `json:"description" binding:"omitempty,max=1000"`
}

type UpdateQuestionBankRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=255"`
	SubjectID   *uint  `json:"subject_id"`
	Description string `json:"description" binding:"omitempty,max=1000"`
}

type QuestionBankResponse struct {
	ID            uint   `json:"id"`
	Name          string `json:"name"`
	SubjectID     *uint  `json:"subject_id"`
	Description   string `json:"description"`
	Status        string `json:"status"`
	CreatedBy     uint   `json:"created_by"`
	QuestionCount int    `json:"question_count"`
	IsLocked      bool   `json:"is_locked"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type CreateStimulusRequest struct {
	Content string `json:"content" binding:"required"`
}

type UpdateStimulusRequest struct {
	Content string `json:"content" binding:"required"`
}

type CreateQuestionRequest struct {
	StimulusID    *uint           `json:"stimulus_id"`
	QuestionType  string          `json:"question_type" binding:"required,oneof=pg pgk benar_salah menjodohkan isian matrix esai"`
	Body          string          `json:"body" binding:"required,min=1"`
	Score         float64         `json:"score" binding:"omitempty,min=0"`
	Difficulty    string          `json:"difficulty" binding:"omitempty,oneof=mudah sedang sulit"`
	Options       json.RawMessage `json:"options"`
	CorrectAnswer json.RawMessage `json:"correct_answer"`
	OrderIndex    int             `json:"order_index" binding:"omitempty,min=0"`
	AudioPath     *string         `json:"audio_path"`
	AudioLimit    int             `json:"audio_limit" binding:"omitempty,min=0"`
	BloomLevel    int             `json:"bloom_level" binding:"omitempty,min=0,max=6"`
	TopicCode     string          `json:"topic_code" binding:"omitempty,max=50"`
}

type UpdateQuestionRequest struct {
	StimulusID    *uint           `json:"stimulus_id"`
	QuestionType  string          `json:"question_type" binding:"omitempty,oneof=pg pgk benar_salah menjodohkan isian matrix esai"`
	Body          string          `json:"body"`
	Score         float64         `json:"score" binding:"omitempty,min=0"`
	Difficulty    string          `json:"difficulty" binding:"omitempty,oneof=mudah sedang sulit"`
	Options       json.RawMessage `json:"options"`
	CorrectAnswer json.RawMessage `json:"correct_answer"`
	OrderIndex    int             `json:"order_index" binding:"omitempty,min=0"`
	AudioPath     *string         `json:"audio_path"`
	AudioLimit    int             `json:"audio_limit" binding:"omitempty,min=0"`
	RemoveAudio   bool            `json:"remove_audio"`
	BloomLevel    int             `json:"bloom_level" binding:"omitempty,min=0,max=6"`
	TopicCode     string          `json:"topic_code" binding:"omitempty,max=50"`
}

type ReorderItem struct {
	ID         uint `json:"id" binding:"required"`
	OrderIndex int  `json:"order_index"`
}

type ReorderRequest struct {
	Items []ReorderItem `json:"items" binding:"required"`
}
