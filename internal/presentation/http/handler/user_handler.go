package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/omanjaya/patra/internal/application/dto"
	"github.com/omanjaya/patra/internal/domain/entity"
	useruc "github.com/omanjaya/patra/internal/application/usecase/user"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/ginhelper"
	"github.com/omanjaya/patra/pkg/pagination"
	"github.com/omanjaya/patra/pkg/response"
	"gorm.io/gorm"
)

type UserHandler struct {
	uc *useruc.UserUseCase
	db *gorm.DB
}

func NewUserHandler(uc *useruc.UserUseCase, db *gorm.DB) *UserHandler {
	return &UserHandler{uc: uc, db: db}
}

func (h *UserHandler) List(c *gin.Context) {
	p := pagination.FromQuery(c)
	filter := repository.UserListFilter{
		Search: c.Query("search"),
		Role:   c.Query("role"),
	}
	if rid := c.Query("rombel_id"); rid != "" {
		id, err := strconv.ParseUint(rid, 10, 64)
		if err == nil {
			uid := uint(id)
			filter.RombelID = &uid
		}
	}

	users, total, err := h.uc.List(filter, p)
	if err != nil {
		response.InternalError(c, "Gagal mengambil data user")
		return
	}

	ginhelper.RespondPaginated(c, users, p, total)
}

func (h *UserHandler) Create(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	user, err := h.uc.Create(req)
	if err != nil {
		if errors.Is(err, useruc.ErrUsernameTaken) {
			response.ValidationError(c, gin.H{"username": err.Error()})
			return
		}
		response.InternalError(c, "Gagal membuat user")
		return
	}
	response.Created(c, user)
}

func (h *UserHandler) Update(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	user, err := h.uc.Update(id, req)
	if err != nil {
		if errors.Is(err, useruc.ErrUserNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalError(c, "Gagal memperbarui user")
		return
	}
	response.Success(c, user)
}

func (h *UserHandler) Delete(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	if err := h.uc.Delete(id); err != nil {
		if errors.Is(err, useruc.ErrUserNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalError(c, "Gagal menghapus user")
		return
	}
	response.Success(c, nil)
}

func (h *UserHandler) ListTrashed(c *gin.Context) {
	p := pagination.FromQuery(c)
	filter := repository.UserListFilter{
		Search: c.Query("search"),
		Role:   c.Query("role"),
	}
	users, total, err := h.uc.ListTrashed(filter, p)
	if err != nil {
		response.InternalError(c, "Gagal mengambil data user")
		return
	}
	ginhelper.RespondPaginated(c, users, p, total)
}

func (h *UserHandler) Restore(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	if err := h.uc.Restore(id); err != nil {
		response.InternalError(c, "Gagal memulihkan user")
		return
	}
	response.Success(c, nil)
}

func (h *UserHandler) ForceDelete(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	if err := h.uc.ForceDelete(id); err != nil {
		response.InternalError(c, "Gagal menghapus permanen user")
		return
	}
	response.Success(c, nil)
}

func (h *UserHandler) BulkAction(c *gin.Context) {
	var body struct {
		Action string `json:"action" binding:"required"`
		IDs    []uint `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	var err error
	switch body.Action {
	case "delete":
		err = h.uc.BulkDelete(body.IDs)
	case "restore":
		err = h.uc.BulkRestore(body.IDs)
	case "force_delete":
		err = h.uc.BulkForceDelete(body.IDs)
	default:
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Aksi tidak valid")
		return
	}

	if err != nil {
		response.InternalError(c, "Gagal melakukan aksi massal")
		return
	}
	response.Success(c, nil)
}

// POST /admin/users/import — multipart Excel file
func (h *UserHandler) ImportExcel(c *gin.Context) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, "file wajib diupload")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		response.InternalError(c, "Gagal membaca file")
		return
	}

	result, err := h.uc.ImportExcel(data)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, result)
}

// GET /admin/users/import/template
func (h *UserHandler) DownloadTemplate(c *gin.Context) {
	csvContent := "name,username,password,role,email,nis,nip,class,major,phone\r\n" +
		"Budi Santoso,budisantoso,password123,peserta,budi@sekolah.id,12345,,XII IPA 1,IPA,081234567890\r\n" +
		"Siti Rahayu,sitirahayu,password123,guru,siti@sekolah.id,,198801012010,,,081298765432\r\n"

	c.Header("Content-Disposition", `attachment; filename="template-import-user.csv"`)
	c.Header("Content-Type", "text/csv")
	c.String(http.StatusOK, csvContent)
}

// SearchPeserta returns a list of peserta users for selection dropdowns.
// GET /admin/users/search-peserta?q=keyword&rombel_id=X&limit=20
func (h *UserHandler) SearchPeserta(c *gin.Context) {
	q := c.Query("q")
	rombelIDStr := c.Query("rombel_id")
	limitStr := c.DefaultQuery("limit", "20")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	type PesertaResult struct {
		ID         uint    `json:"id"`
		Name       string  `json:"name"`
		Username   string  `json:"username"`
		NIS        *string `json:"nis"`
		RombelName *string `json:"rombel_name"`
	}

	query := h.db.Table("users").
		Select("users.id, users.name, users.username, up.nis, r.name as rombel_name").
		Joins("LEFT JOIN user_profiles up ON up.user_id = users.id").
		Joins("LEFT JOIN user_rombels ur ON ur.user_id = users.id").
		Joins("LEFT JOIN rombels r ON r.id = ur.rombel_id AND r.deleted_at IS NULL").
		Where("users.role = ?", entity.RolePeserta).
		Where("users.deleted_at IS NULL").
		Where("users.is_active = ?", true)

	if q != "" {
		search := "%" + q + "%"
		query = query.Where("(users.name ILIKE ? OR users.username ILIKE ? OR up.nis ILIKE ?)", search, search, search)
	}

	if rombelIDStr != "" {
		rombelID, err := strconv.ParseUint(rombelIDStr, 10, 64)
		if err == nil {
			query = query.Where("ur.rombel_id = ?", uint(rombelID))
		}
	}

	results := make([]PesertaResult, 0)
	if err := query.Limit(limit).Order("users.name ASC").Find(&results).Error; err != nil {
		response.InternalError(c, "Gagal mencari peserta")
		return
	}

	response.Success(c, results)
}
