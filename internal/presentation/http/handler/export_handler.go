package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	reportuc "github.com/omanjaya/patra/internal/application/usecase/report"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/response"
)

type ExportHandler struct {
	sessionRepo  repository.ExamSessionRepository
	scheduleRepo repository.ExamScheduleRepository
	questionRepo repository.QuestionRepository
}

func NewExportHandler(
	sessionRepo repository.ExamSessionRepository,
	scheduleRepo repository.ExamScheduleRepository,
	questionRepo repository.QuestionRepository,
) *ExportHandler {
	return &ExportHandler{
		sessionRepo:  sessionRepo,
		scheduleRepo: scheduleRepo,
		questionRepo: questionRepo,
	}
}

// GET /reports/:scheduleId/export?multi_sheet=true
func (h *ExportHandler) LedgerExcel(c *gin.Context) {
	scheduleID, err := strconv.ParseUint(c.Param("scheduleId"), 10, 64)
	if err != nil {
		response.BadRequest(c, "ID tidak valid")
		return
	}

	multiSheet := c.Query("multi_sheet") == "true"

	var (
		data     []byte
		filename string
	)

	if multiSheet {
		data, filename, err = reportuc.ExportLedgerMultiSheet(uint(scheduleID), h.sessionRepo, h.scheduleRepo, h.questionRepo)
	} else {
		data, filename, err = reportuc.ExportLedger(uint(scheduleID), h.sessionRepo, h.scheduleRepo, h.questionRepo)
	}

	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

// UnfinishedExcel exports a list of students who haven't finished the exam.
// GET /reports/:scheduleId/unfinished/export
func (h *ExportHandler) UnfinishedExcel(c *gin.Context) {
	scheduleID, err := strconv.ParseUint(c.Param("scheduleId"), 10, 64)
	if err != nil {
		response.BadRequest(c, "ID tidak valid")
		return
	}

	data, filename, err := reportuc.ExportUnfinished(uint(scheduleID), h.sessionRepo, h.scheduleRepo)
	if err != nil {
		response.BadRequest(c, fmt.Sprintf("Gagal export: %s", err.Error()))
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}
