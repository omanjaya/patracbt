package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/omanjaya/patra/internal/application/usecase/master"
	"github.com/omanjaya/patra/pkg/ginhelper"
	"github.com/omanjaya/patra/pkg/pagination"
	"github.com/omanjaya/patra/pkg/response"
)

type RoleHandler struct {
	uc *master.RoleUseCase
}

func NewRoleHandler(uc *master.RoleUseCase) *RoleHandler {
	return &RoleHandler{uc: uc}
}

// GET /admin/roles
func (h *RoleHandler) List(c *gin.Context) {
	p := pagination.FromQuery(c)
	search := c.Query("search")

	roles, total, err := h.uc.List(search, p)
	if err != nil {
		response.InternalError(c, "Gagal mengambil data role")
		return
	}

	ginhelper.RespondPaginated(c, roles, p, total)
}

// POST /admin/roles
func (h *RoleHandler) Create(c *gin.Context) {
	var req struct {
		Name      string `json:"name" binding:"required"`
		GuardName string `json:"guard_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	if req.GuardName == "" {
		req.GuardName = "web"
	}
	role, err := h.uc.Create(req.Name, req.GuardName)
	if err != nil {
		response.InternalError(c, "Gagal membuat role")
		return
	}
	response.Created(c, role)
}

// PUT /admin/roles/:id
func (h *RoleHandler) Update(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	var req struct {
		Name      string `json:"name"`
		GuardName string `json:"guard_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	role, err := h.uc.Update(id, req.Name, req.GuardName)
	if err != nil {
		response.InternalError(c, "Gagal memperbarui role")
		return
	}
	response.Success(c, role)
}

// DELETE /admin/roles/:id
func (h *RoleHandler) Delete(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	if err := h.uc.Delete(id); err != nil {
		response.InternalError(c, "Gagal menghapus role")
		return
	}
	response.Success(c, nil)
}

// GET /admin/roles/:id/permissions
func (h *RoleHandler) GetPermissions(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	permissions, err := h.uc.GetRolePermissions(id)
	if err != nil {
		response.InternalError(c, "Gagal mendapatkan izin role")
		return
	}
	response.Success(c, permissions)
}

// POST /admin/roles/:id/permissions
func (h *RoleHandler) AssignPermissions(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	var req struct {
		PermissionIDs []uint `json:"permission_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	if err := h.uc.AssignPermissions(id, req.PermissionIDs); err != nil {
		response.InternalError(c, "Gagal menetapkan izin role")
		return
	}
	response.Success(c, gin.H{"message": "Izin role berhasil diperbarui"})
}
