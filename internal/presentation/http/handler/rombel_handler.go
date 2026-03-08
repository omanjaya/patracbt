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

type RombelHandler struct {
	uc *master.RombelUseCase
}

func NewRombelHandler(uc *master.RombelUseCase) *RombelHandler {
	return &RombelHandler{uc: uc}
}

func (h *RombelHandler) List(c *gin.Context) {
	p := pagination.FromQuery(c)
	search := c.Query("search")

	rombels, total, err := h.uc.List(search, p)
	if err != nil {
		response.InternalError(c, "Gagal mengambil data rombel")
		return
	}

	ginhelper.RespondPaginated(c, rombels, p, total)
}

func (h *RombelHandler) Create(c *gin.Context) {
	var req dto.CreateRombelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	rombel, err := h.uc.Create(req)
	if err != nil {
		response.InternalError(c, "Gagal membuat rombel")
		return
	}

	response.Created(c, rombel)
}

func (h *RombelHandler) Update(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateRombelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	rombel, err := h.uc.Update(id, req)
	if err != nil {
		if errors.Is(err, master.ErrRombelNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalError(c, "Gagal memperbarui rombel")
		return
	}

	response.Success(c, rombel)
}

func (h *RombelHandler) Delete(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	if err := h.uc.Delete(id); err != nil {
		if errors.Is(err, master.ErrRombelNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalError(c, "Gagal menghapus rombel")
		return
	}
	response.Success(c, nil)
}

func (h *RombelHandler) BulkDelete(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	if err := h.uc.BulkDelete(req.IDs); err != nil {
		response.InternalError(c, "Gagal menghapus rombel")
		return
	}
	response.Success(c, gin.H{"deleted": len(req.IDs)})
}

func (h *RombelHandler) AssignUsers(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	var req dto.AssignUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	if err := h.uc.AssignUsers(id, req); err != nil {
		response.InternalError(c, "Gagal mengaitkan user ke rombel")
		return
	}
	response.Success(c, nil)
}

func (h *RombelHandler) RemoveUsers(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	var req dto.AssignUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	if err := h.uc.RemoveUsers(id, req); err != nil {
		response.InternalError(c, "Gagal melepas user dari rombel")
		return
	}
	response.Success(c, nil)
}
