package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/omanjaya/patra/internal/application/dto"
	"github.com/omanjaya/patra/internal/application/usecase/master"
	"github.com/omanjaya/patra/pkg/ginhelper"
	"github.com/omanjaya/patra/pkg/pagination"
	"github.com/omanjaya/patra/pkg/response"
)

type SubjectHandler struct {
	uc *master.SubjectUseCase
}

func NewSubjectHandler(uc *master.SubjectUseCase) *SubjectHandler {
	return &SubjectHandler{uc: uc}
}

func (h *SubjectHandler) List(c *gin.Context) {
	p := pagination.FromQuery(c)
	search := c.Query("search")

	subjects, total, err := h.uc.List(search, p)
	if err != nil {
		response.InternalError(c, "Gagal mengambil data mata pelajaran")
		return
	}

	ginhelper.RespondPaginated(c, subjects, p, total)
}

func (h *SubjectHandler) ListAll(c *gin.Context) {
	subjects, err := h.uc.ListAll()
	if err != nil {
		response.InternalError(c, "Gagal mengambil data mata pelajaran")
		return
	}
	response.Success(c, subjects)
}

func (h *SubjectHandler) Create(c *gin.Context) {
	var req dto.CreateSubjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	subject, err := h.uc.Create(req)
	if err != nil {
		response.InternalError(c, "Gagal membuat mata pelajaran")
		return
	}
	response.Created(c, subject)
}

func (h *SubjectHandler) Update(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateSubjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	subject, err := h.uc.Update(id, req)
	if err != nil {
		if errors.Is(err, master.ErrSubjectNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalError(c, "Gagal memperbarui mata pelajaran")
		return
	}
	response.Success(c, subject)
}

func (h *SubjectHandler) Delete(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	if err := h.uc.Delete(id); err != nil {
		if errors.Is(err, master.ErrSubjectNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalError(c, "Gagal menghapus mata pelajaran")
		return
	}
	response.Success(c, nil)
}

func (h *SubjectHandler) BulkDelete(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	if err := h.uc.BulkDelete(req.IDs); err != nil {
		response.InternalError(c, "Gagal menghapus mata pelajaran")
		return
	}
	response.Success(c, gin.H{"deleted": len(req.IDs)})
}
