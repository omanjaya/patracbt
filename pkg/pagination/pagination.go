package pagination

import (
	"math"
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	DefaultPerPage = 20
	MaxPerPage     = 100
)

type Params struct {
	Page    int
	PerPage int
}

// Normalize enforces valid bounds on pagination parameters.
// Page minimum is 1, PerPage is clamped between 1 and MaxPerPage.
func (p *Params) Normalize() {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PerPage > MaxPerPage {
		p.PerPage = MaxPerPage
	}
	if p.PerPage < 1 {
		p.PerPage = DefaultPerPage
	}
}

func FromQuery(c *gin.Context) Params {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	p := Params{Page: page, PerPage: perPage}
	p.Normalize()
	return p
}

func (p Params) Offset() int {
	return (p.Page - 1) * p.PerPage
}

func TotalPages(total int64, perPage int) int {
	return int(math.Ceil(float64(total) / float64(perPage)))
}
