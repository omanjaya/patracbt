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

type RoomHandler struct {
	uc *master.RoomUseCase
}

func NewRoomHandler(uc *master.RoomUseCase) *RoomHandler {
	return &RoomHandler{uc: uc}
}

func (h *RoomHandler) List(c *gin.Context) {
	p := pagination.FromQuery(c)
	search := c.Query("search")

	rooms, total, err := h.uc.List(search, p)
	if err != nil {
		response.InternalError(c, "Gagal mengambil data ruangan")
		return
	}

	ginhelper.RespondPaginated(c, rooms, p, total)
}

func (h *RoomHandler) Create(c *gin.Context) {
	var req dto.CreateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	room, err := h.uc.Create(req)
	if err != nil {
		response.InternalError(c, "Gagal membuat ruangan")
		return
	}
	response.Created(c, room)
}

func (h *RoomHandler) Update(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	room, err := h.uc.Update(id, req)
	if err != nil {
		if errors.Is(err, master.ErrRoomNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalError(c, "Gagal memperbarui ruangan")
		return
	}
	response.Success(c, room)
}

func (h *RoomHandler) Delete(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	if err := h.uc.Delete(id); err != nil {
		if errors.Is(err, master.ErrRoomNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		if errors.Is(err, master.ErrRoomHasStudents) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalError(c, "Gagal menghapus ruangan")
		return
	}
	response.Success(c, nil)
}

func (h *RoomHandler) BulkDelete(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	result, err := h.uc.BulkDelete(req.IDs)
	if err != nil {
		response.InternalError(c, "Gagal menghapus ruangan")
		return
	}
	response.Success(c, result)
}

func (h *RoomHandler) AssignUsers(c *gin.Context) {
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
		if errors.Is(err, master.ErrRoomNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalError(c, "Gagal mengaitkan user ke ruangan")
		return
	}
	response.Success(c, nil)
}

func (h *RoomHandler) RemoveUsers(c *gin.Context) {
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
		if errors.Is(err, master.ErrRoomNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalError(c, "Gagal menghapus user dari ruangan")
		return
	}
	response.Success(c, nil)
}

func (h *RoomHandler) GetUsers(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	p := pagination.FromQuery(c)

	users, total, err := h.uc.GetUsers(id, p)
	if err != nil {
		if errors.Is(err, master.ErrRoomNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalError(c, "Gagal mengambil data user ruangan")
		return
	}

	ginhelper.RespondPaginated(c, users, p, total)
}
