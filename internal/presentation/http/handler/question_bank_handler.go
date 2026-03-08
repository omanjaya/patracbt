package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/omanjaya/patra/internal/application/dto"
	questionuc "github.com/omanjaya/patra/internal/application/usecase/question"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/ginhelper"
	"github.com/omanjaya/patra/pkg/pagination"
	"github.com/omanjaya/patra/pkg/response"
)

type QuestionBankHandler struct {
	uc *questionuc.QuestionBankUseCase
}

func NewQuestionBankHandler(uc *questionuc.QuestionBankUseCase) *QuestionBankHandler {
	return &QuestionBankHandler{uc: uc}
}

func (h *QuestionBankHandler) List(c *gin.Context) {
	p := pagination.FromQuery(c)
	filter := repository.QuestionBankFilter{
		Search: c.Query("search"),
	}
	if s := c.Query("subject_id"); s != "" {
		if id, err := strconv.ParseUint(s, 10, 64); err == nil {
			uid := uint(id)
			filter.SubjectID = &uid
		}
	}

	banks, total, err := h.uc.List(filter, p)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	ginhelper.RespondPaginated(c, banks, p, total)
}

func (h *QuestionBankHandler) GetByID(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	bank, err := h.uc.GetByID(id)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.Success(c, bank)
}

func (h *QuestionBankHandler) Create(c *gin.Context) {
	var req dto.CreateQuestionBankRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")
	bank, err := h.uc.Create(req, userID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Created(c, bank)
}

func (h *QuestionBankHandler) Update(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateQuestionBankRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	bank, err := h.uc.Update(id, req)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.Success(c, bank)
}

func (h *QuestionBankHandler) Delete(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	if err := h.uc.Delete(id); err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "Bank soal dihapus"})
}

func (h *QuestionBankHandler) BulkDelete(c *gin.Context) {
	var body struct {
		IDs []uint `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.uc.BulkDelete(body.IDs); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "Bank soal dihapus"})
}

func (h *QuestionBankHandler) Clone(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	userID := c.GetUint("user_id")
	bank, err := h.uc.Clone(id, userID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, bank)
}

func (h *QuestionBankHandler) ToggleStatus(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	if err := h.uc.ToggleStatus(id); err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "Status bank soal diperbarui"})
}
