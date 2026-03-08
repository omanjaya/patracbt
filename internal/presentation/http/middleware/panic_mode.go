package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/response"
)

var (
	panicModeCache    *bool
	panicModeCachedAt time.Time
	panicModeMu       sync.RWMutex
	panicModeCacheTTL = 2 * time.Minute
)

// PanicMode blocks peserta routes when panic_mode_active = "1".
func PanicMode(settingRepo repository.SettingRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role != entity.RolePeserta {
			c.Next()
			return
		}

		active := getPanicModeActive(settingRepo)
		if active {
			response.Error(c, http.StatusServiceUnavailable, "PANIC_MODE", "Sistem ujian sedang dalam mode darurat. Hubungi pengawas.")
			c.Abort()
			return
		}
		c.Next()
	}
}

func getPanicModeActive(settingRepo repository.SettingRepository) bool {
	panicModeMu.RLock()
	if panicModeCache != nil && time.Since(panicModeCachedAt) < panicModeCacheTTL {
		val := *panicModeCache
		panicModeMu.RUnlock()
		return val
	}
	panicModeMu.RUnlock()

	panicModeMu.Lock()
	defer panicModeMu.Unlock()

	// Double-check after acquiring write lock
	if panicModeCache != nil && time.Since(panicModeCachedAt) < panicModeCacheTTL {
		return *panicModeCache
	}

	setting, err := settingRepo.GetByKey("panic_mode_active")
	active := err == nil && setting != nil && setting.Value != nil && *setting.Value == "1"
	panicModeCache = &active
	panicModeCachedAt = time.Now()
	return active
}
