package question

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/omanjaya/patra/internal/application/dto"
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/audio"
	"github.com/omanjaya/patra/pkg/pagination"
	"github.com/omanjaya/patra/pkg/sanitizer"
	"github.com/omanjaya/patra/pkg/types"
)

type QuestionUseCase struct {
	questionRepo repository.QuestionRepository
	bankRepo     repository.QuestionBankRepository
}

func NewQuestionUseCase(questionRepo repository.QuestionRepository, bankRepo repository.QuestionBankRepository) *QuestionUseCase {
	return &QuestionUseCase{questionRepo: questionRepo, bankRepo: bankRepo}
}

func (uc *QuestionUseCase) List(bankID uint, p pagination.Params) ([]*entity.Question, int64, error) {
	return uc.questionRepo.ListByBank(bankID, p)
}

func (uc *QuestionUseCase) GetByID(id uint) (*entity.Question, error) {
	q, err := uc.questionRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("soal tidak ditemukan")
	}
	return q, nil
}

func (uc *QuestionUseCase) Create(bankID uint, req dto.CreateQuestionRequest) (*entity.Question, error) {
	if _, err := uc.bankRepo.FindByID(bankID); err != nil {
		return nil, errors.New("bank soal tidak ditemukan")
	}
	if uc.bankRepo.IsBankUsedInSchedule(bankID) {
		return nil, errors.New("tidak dapat mengubah soal karena bank soal sedang digunakan")
	}

	if req.StimulusID != nil && *req.StimulusID > 0 {
		stimulus, err := uc.questionRepo.FindStimulusByID(*req.StimulusID)
		if err != nil {
			return nil, fmt.Errorf("stimulus tidak ditemukan")
		}
		if stimulus.QuestionBankID != bankID {
			return nil, fmt.Errorf("stimulus tidak termasuk dalam bank soal ini")
		}
	}

	difficulty := req.Difficulty
	if difficulty == "" {
		difficulty = entity.DifficultyMedium
	}
	score := req.Score
	if score <= 0 {
		score = 1
	}

	audioLimit := req.AudioLimit
	if audioLimit == 0 && req.AudioPath != nil {
		audioLimit = 2
	}

	q := &entity.Question{
		QuestionBankID: bankID,
		StimulusID:     req.StimulusID,
		QuestionType:   req.QuestionType,
		Body:           sanitizer.SanitizeHTML(req.Body),
		Score:          score,
		Difficulty:     difficulty,
		Options:        sanitizeOptions(req.Options),
		CorrectAnswer:  types.JSON(req.CorrectAnswer),
		OrderIndex:     req.OrderIndex,
		AudioPath:      req.AudioPath,
		AudioLimit:     audioLimit,
		BloomLevel:     req.BloomLevel,
		TopicCode:      req.TopicCode,
	}
	if err := uc.questionRepo.Create(q); err != nil {
		return nil, err
	}
	return q, nil
}

func (uc *QuestionUseCase) Update(id uint, req dto.UpdateQuestionRequest) (*entity.Question, error) {
	q, err := uc.questionRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("soal tidak ditemukan")
	}
	if uc.bankRepo.IsBankUsedInSchedule(q.QuestionBankID) {
		return nil, errors.New("tidak dapat mengubah soal karena bank soal sedang digunakan")
	}

	if req.QuestionType != "" {
		q.QuestionType = req.QuestionType
	}
	if req.Body != "" {
		q.Body = sanitizer.SanitizeHTML(req.Body)
	}
	if req.Score > 0 {
		q.Score = req.Score
	}
	if req.Difficulty != "" {
		q.Difficulty = req.Difficulty
	}
	if req.BloomLevel >= 0 {
		q.BloomLevel = req.BloomLevel
	}
	q.TopicCode = req.TopicCode
	if req.Options != nil {
		q.Options = sanitizeOptions(req.Options)
	}
	if req.CorrectAnswer != nil {
		q.CorrectAnswer = types.JSON(req.CorrectAnswer)
	}
	q.StimulusID = req.StimulusID
	q.OrderIndex = req.OrderIndex

	// Handle audio removal
	if req.RemoveAudio {
		if q.AudioPath != nil {
			audio.DeleteFile(*q.AudioPath)
		}
		q.AudioPath = nil
		q.AudioLimit = 0
	}

	// Handle new audio upload
	if req.AudioPath != nil {
		// Delete old audio file if exists
		if q.AudioPath != nil {
			audio.DeleteFile(*q.AudioPath)
		}
		q.AudioPath = req.AudioPath
		q.AudioLimit = req.AudioLimit
		if q.AudioLimit == 0 {
			q.AudioLimit = 2
		}
	}

	// Update audio limit even without new file
	if !req.RemoveAudio && req.AudioPath == nil && req.AudioLimit > 0 && q.AudioPath != nil {
		q.AudioLimit = req.AudioLimit
	}

	if err := uc.questionRepo.Update(q); err != nil {
		return nil, err
	}
	return q, nil
}

func (uc *QuestionUseCase) Delete(id uint) error {
	q, err := uc.questionRepo.FindByID(id)
	if err != nil {
		return errors.New("soal tidak ditemukan")
	}
	if uc.bankRepo.IsBankUsedInSchedule(q.QuestionBankID) {
		return errors.New("tidak dapat mengubah soal karena bank soal sedang digunakan")
	}
	return uc.questionRepo.Delete(id)
}

func (uc *QuestionUseCase) Reorder(bankID uint, items []dto.ReorderItem) error {
	return uc.questionRepo.Reorder(bankID, items)
}

func (uc *QuestionUseCase) BulkDelete(ids []uint) error {
	return uc.questionRepo.BulkDelete(ids)
}

func (uc *QuestionUseCase) MoveToBank(ids []uint, targetBankID uint) error {
	if _, err := uc.bankRepo.FindByID(targetBankID); err != nil {
		return errors.New("bank soal tujuan tidak ditemukan")
	}
	return uc.questionRepo.MoveToBank(ids, targetBankID)
}

func (uc *QuestionUseCase) CopyToBank(ids []uint, targetBankID uint) error {
	if _, err := uc.bankRepo.FindByID(targetBankID); err != nil {
		return errors.New("bank soal tujuan tidak ditemukan")
	}
	return uc.questionRepo.CopyToBank(ids, targetBankID)
}

func (uc *QuestionUseCase) ListAllByBank(bankID uint) ([]*entity.Question, error) {
	return uc.questionRepo.ListAllByBank(bankID)
}

func (uc *QuestionUseCase) ListIDsByBank(bankID uint, search string) ([]uint, error) {
	return uc.questionRepo.ListIDsByBank(bankID, search)
}

// Stimulus

func (uc *QuestionUseCase) ListStimuli(bankID uint) ([]*entity.Stimulus, error) {
	return uc.questionRepo.ListStimuliByBank(bankID)
}

func (uc *QuestionUseCase) CreateStimulus(bankID uint, req dto.CreateStimulusRequest) (*entity.Stimulus, error) {
	s := &entity.Stimulus{
		QuestionBankID: bankID,
		Content:        sanitizer.SanitizeHTML(req.Content),
	}
	if err := uc.questionRepo.CreateStimulus(s); err != nil {
		return nil, err
	}
	return s, nil
}

func (uc *QuestionUseCase) UpdateStimulus(id uint, req dto.UpdateStimulusRequest) (*entity.Stimulus, error) {
	s, err := uc.questionRepo.FindStimulusByID(id)
	if err != nil {
		return nil, errors.New("stimulus tidak ditemukan")
	}
	s.Content = sanitizer.SanitizeHTML(req.Content)
	if err := uc.questionRepo.UpdateStimulus(s); err != nil {
		return nil, err
	}
	return s, nil
}

func (uc *QuestionUseCase) DeleteStimulus(id uint) error {
	return uc.questionRepo.DeleteStimulus(id)
}

// sanitizeOptions sanitizes HTML in option text fields within a JSON array.
func sanitizeOptions(raw json.RawMessage) types.JSON {
	if raw == nil {
		return nil
	}
	var opts []map[string]interface{}
	if err := json.Unmarshal(raw, &opts); err != nil {
		// Not a JSON array of objects, sanitize as raw string
		return types.JSON(sanitizer.SanitizeHTML(string(raw)))
	}
	for i, opt := range opts {
		if text, ok := opt["text"].(string); ok {
			opts[i]["text"] = sanitizer.SanitizeHTML(text)
		}
	}
	b, _ := json.Marshal(opts)
	return types.JSON(b)
}
