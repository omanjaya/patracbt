package ginhelper

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/omanjaya/patra/pkg/pagination"
	"github.com/omanjaya/patra/pkg/response"
)

// ParseID extracts and validates a uint ID from a URL parameter.
// Returns 0 and sends a 400 response if invalid.
func ParseID(c *gin.Context, param string) (uint, bool) {
	id, err := strconv.ParseUint(c.Param(param), 10, 64)
	if err != nil {
		response.BadRequest(c, "ID tidak valid")
		return 0, false
	}
	return uint(id), true
}

// RespondPaginated sends a paginated success response with standard meta.
func RespondPaginated(c *gin.Context, data interface{}, p pagination.Params, total int64) {
	response.SuccessWithMeta(c, data, &response.Meta{
		Page:       p.Page,
		PerPage:    p.PerPage,
		Total:      total,
		TotalPages: pagination.TotalPages(total, p.PerPage),
	})
}
