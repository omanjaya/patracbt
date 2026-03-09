package handler

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/omanjaya/patra/internal/application/dto"
	"github.com/omanjaya/patra/internal/application/usecase/master"
	"github.com/omanjaya/patra/pkg/ginhelper"
	"github.com/omanjaya/patra/pkg/pagination"
	"github.com/omanjaya/patra/pkg/response"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type TagHandler struct {
	uc *master.TagUseCase
	db *gorm.DB
}

func NewTagHandler(uc *master.TagUseCase, db *gorm.DB) *TagHandler {
	return &TagHandler{uc: uc, db: db}
}

func (h *TagHandler) List(c *gin.Context) {
	p := pagination.FromQuery(c)
	search := c.Query("search")

	tags, total, err := h.uc.List(search, p)
	if err != nil {
		response.InternalError(c, "Gagal mengambil data tag")
		return
	}

	ginhelper.RespondPaginated(c, tags, p, total)
}

func (h *TagHandler) ListAll(c *gin.Context) {
	tags, err := h.uc.ListAll()
	if err != nil {
		response.InternalError(c, "Gagal mengambil data tag")
		return
	}
	response.Success(c, tags)
}

func (h *TagHandler) Create(c *gin.Context) {
	var req dto.CreateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	tag, err := h.uc.Create(req)
	if err != nil {
		response.InternalError(c, "Gagal membuat tag")
		return
	}
	response.Created(c, tag)
}

func (h *TagHandler) Update(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	tag, err := h.uc.Update(id, req)
	if err != nil {
		if errors.Is(err, master.ErrTagNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalError(c, "Gagal memperbarui tag")
		return
	}
	response.Success(c, tag)
}

func (h *TagHandler) Delete(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	if err := h.uc.Delete(id); err != nil {
		if errors.Is(err, master.ErrTagNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		if errors.Is(err, master.ErrTagInUse) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalError(c, "Gagal menghapus tag")
		return
	}
	response.Success(c, nil)
}

func (h *TagHandler) BulkDelete(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	result, err := h.uc.BulkDelete(req.IDs)
	if err != nil {
		response.InternalError(c, "Gagal menghapus tag")
		return
	}
	response.Success(c, result)
}

func (h *TagHandler) AssignUsers(c *gin.Context) {
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
		response.InternalError(c, "Gagal mengaitkan user ke tag")
		return
	}
	response.Success(c, nil)
}

func (h *TagHandler) RemoveUsers(c *gin.Context) {
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
		if errors.Is(err, master.ErrTagNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalError(c, "Gagal menghapus user dari tag")
		return
	}
	response.Success(c, nil)
}

// ImportUserTags handles Excel import for tag assignments.
// POST /admin/tags/import-users
func (h *TagHandler) ImportUserTags(c *gin.Context) {
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

	result, err := master.ImportUserTags(data, h.db)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, result)
}

// ExportTemplate generates an Excel template for tag import.
// GET /admin/tags/export-template
func (h *TagHandler) ExportTemplate(c *gin.Context) {
	// Get all tags
	tags, err := h.uc.ListAll()
	if err != nil {
		response.InternalError(c, "Gagal mengambil data tag")
		return
	}

	f := excelize.NewFile()
	sheet := "Sheet1"

	// Header row
	_ = f.SetCellValue(sheet, "A1", "NIS")
	for i, tag := range tags {
		col, _ := excelize.ColumnNumberToName(i + 2)
		_ = f.SetCellValue(sheet, fmt.Sprintf("%s1", col), tag.Name)
	}

	// Style header
	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E2E8F0"}, Pattern: 1},
	})
	lastCol, _ := excelize.ColumnNumberToName(len(tags) + 1)
	_ = f.SetCellStyle(sheet, "A1", fmt.Sprintf("%s1", lastCol), style)

	// Example row
	_ = f.SetCellValue(sheet, "A2", "12345")
	for i := range tags {
		col, _ := excelize.ColumnNumberToName(i + 2)
		_ = f.SetCellValue(sheet, fmt.Sprintf("%s2", col), "Ya")
	}

	// Write to buffer
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		response.InternalError(c, "Gagal membuat template")
		return
	}

	c.Header("Content-Disposition", `attachment; filename="template-import-tag.xlsx"`)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}
