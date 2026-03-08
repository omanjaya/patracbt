package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/omanjaya/patra/internal/infrastructure/persistence/postgres"
	"github.com/omanjaya/patra/pkg/pagination"
	"github.com/omanjaya/patra/pkg/response"
)

type AuditLogHandler struct {
	repo *postgres.AuditLogRepo
}

func NewAuditLogHandler(repo *postgres.AuditLogRepo) *AuditLogHandler {
	return &AuditLogHandler{repo: repo}
}

// GET /admin/audit-logs
func (h *AuditLogHandler) List(c *gin.Context) {
	p := pagination.FromQuery(c)

	logs, total, err := h.repo.ListAll(p.Page, p.PerPage)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"data":        logs,
		"total":       total,
		"page":        p.Page,
		"per_page":    p.PerPage,
		"total_pages": pagination.TotalPages(total, p.PerPage),
	})
}
