package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/omanjaya/patra/internal/application/usecase/master"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/ginhelper"
	"github.com/omanjaya/patra/pkg/pagination"
	"github.com/omanjaya/patra/pkg/response"
)

type PermissionHandler struct {
	uc *master.PermissionUseCase
}

func NewPermissionHandler(uc *master.PermissionUseCase) *PermissionHandler {
	return &PermissionHandler{uc: uc}
}

// GET /admin/permissions
func (h *PermissionHandler) List(c *gin.Context) {
	p := pagination.FromQuery(c)
	filter := repository.PermissionListFilter{
		Search:    c.Query("search"),
		GroupName: c.Query("group_name"),
	}
	perms, total, err := h.uc.List(filter, p)
	if err != nil {
		response.InternalError(c, "Gagal mengambil data permission")
		return
	}

	// Collect unique groups from full dataset for convenience
	groups, _ := h.uc.ListGroups()

	type PermissionMeta struct {
		Page       int      `json:"page"`
		PerPage    int      `json:"per_page"`
		Total      int64    `json:"total"`
		TotalPages int      `json:"total_pages"`
		Groups     []string `json:"groups"`
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "OK",
		"data":    perms,
		"meta": PermissionMeta{
			Page:       p.Page,
			PerPage:    p.PerPage,
			Total:      total,
			TotalPages: pagination.TotalPages(total, p.PerPage),
			Groups:     groups,
		},
	})
}

// GET /admin/permissions/all
func (h *PermissionHandler) ListAll(c *gin.Context) {
	perms, err := h.uc.ListAll()
	if err != nil {
		response.InternalError(c, "Gagal mengambil data permission")
		return
	}
	response.Success(c, perms)
}

// GET /admin/permissions/groups
func (h *PermissionHandler) ListGroups(c *gin.Context) {
	groups, err := h.uc.ListGroups()
	if err != nil {
		response.InternalError(c, "Gagal mengambil grup permission")
		return
	}
	response.Success(c, groups)
}

// POST /admin/permissions
func (h *PermissionHandler) Create(c *gin.Context) {
	var req struct {
		Name        string  `json:"name" binding:"required"`
		GroupName   string  `json:"group_name" binding:"required"`
		Description *string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	p, err := h.uc.Create(req.Name, req.GroupName, req.Description)
	if err != nil {
		response.InternalError(c, "Gagal membuat permission")
		return
	}
	response.Created(c, p)
}

// PUT /admin/permissions/:id
func (h *PermissionHandler) Update(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	var req struct {
		Name        string  `json:"name" binding:"required"`
		GroupName   string  `json:"group_name" binding:"required"`
		Description *string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	p, err := h.uc.Update(id, req.Name, req.GroupName, req.Description)
	if err != nil {
		if errors.Is(err, master.ErrPermissionNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalError(c, "Gagal memperbarui permission")
		return
	}
	response.Success(c, p)
}

// DELETE /admin/permissions/:id
func (h *PermissionHandler) Delete(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	if err := h.uc.Delete(id); err != nil {
		if errors.Is(err, master.ErrPermissionNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalError(c, "Gagal menghapus permission")
		return
	}
	response.Success(c, nil)
}

// GET /admin/user-permissions
func (h *PermissionHandler) ListUsersWithPermissions(c *gin.Context) {
	p := pagination.FromQuery(c)
	filter := repository.UserPermissionListFilter{
		Search: c.Query("search"),
	}
	if pid := c.Query("permission_id"); pid != "" {
		id, err := strconv.ParseUint(pid, 10, 64)
		if err == nil {
			uid := uint(id)
			filter.PermissionID = &uid
		}
	}
	if nid := c.Query("no_permission_id"); nid != "" {
		id, err := strconv.ParseUint(nid, 10, 64)
		if err == nil {
			uid := uint(id)
			filter.NoPermissionID = &uid
		}
	}

	users, total, err := h.uc.ListUsersWithPermissions(filter, p)
	if err != nil {
		response.InternalError(c, "Gagal mengambil data user-permission")
		return
	}

	// Map to frontend-expected shape
	type permItem struct {
		ID        uint   `json:"id"`
		Name      string `json:"name"`
		GroupName string `json:"group_name"`
	}
	type userRow struct {
		ID       uint       `json:"id"`
		Name     string     `json:"name"`
		Username string     `json:"username"`
		NIS      *string    `json:"nis"`
		Rombel   *string    `json:"rombel"`
		Status   string     `json:"status"`
		Tags     []permItem `json:"tags"`
	}

	rows := make([]userRow, len(users))
	for i, u := range users {
		tags := make([]permItem, len(u.Permissions))
		for j, perm := range u.Permissions {
			tags[j] = permItem{ID: perm.ID, Name: perm.Name, GroupName: perm.GroupName}
		}
		rows[i] = userRow{
			ID:       u.ID,
			Name:     u.Name,
			Username: u.Username,
			NIS:      u.NIS,
			Rombel:   u.Rombel,
			Status:   "active",
			Tags:     tags,
		}
	}

	ginhelper.RespondPaginated(c, rows, p, total)
}

// POST /admin/user-permissions/assign
func (h *PermissionHandler) AssignPermissionToUsers(c *gin.Context) {
	var req struct {
		UserIDs      []uint `json:"user_ids" binding:"required"`
		PermissionID uint   `json:"permission_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	if err := h.uc.AssignToUsers(req.PermissionID, req.UserIDs); err != nil {
		response.InternalError(c, "Gagal menetapkan permission ke user")
		return
	}
	response.Success(c, nil)
}

// POST /admin/user-permissions/remove
func (h *PermissionHandler) RemovePermissionFromUsers(c *gin.Context) {
	var req struct {
		UserIDs      []uint `json:"user_ids" binding:"required"`
		PermissionID uint   `json:"permission_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	if err := h.uc.RemoveFromUsers(req.PermissionID, req.UserIDs); err != nil {
		response.InternalError(c, "Gagal menghapus permission dari user")
		return
	}
	response.Success(c, nil)
}
