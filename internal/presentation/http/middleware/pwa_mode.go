package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
)

var (
	pwaModeCache    *bool
	pwaModeCachedAt time.Time
	pwaModeMu       sync.RWMutex
	pwaModeCacheTTL = 2 * time.Minute
)

// EnforcePWA checks if PWA mode is required and blocks non-PWA access for peserta.
// PWA apps should send X-PWA-Mode: true header or ?pwa=1 query param.
func EnforcePWA(settingRepo repository.SettingRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check setting: enforce_pwa_mode
		enforced := getPWAModeEnforced(settingRepo)
		if !enforced {
			c.Next()
			return
		}

		// Only enforce for peserta
		role, _ := c.Get("role")
		if role != entity.RolePeserta {
			c.Next()
			return
		}

		// Check for PWA indicator in headers or query params
		isPWA := c.GetHeader("X-PWA-Mode") == "true" || c.Query("pwa") == "1"

		if !isPWA {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "Akses hanya diizinkan melalui aplikasi PWA",
				"code":    "PWA_REQUIRED",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func getPWAModeEnforced(settingRepo repository.SettingRepository) bool {
	pwaModeMu.RLock()
	if pwaModeCache != nil && time.Since(pwaModeCachedAt) < pwaModeCacheTTL {
		val := *pwaModeCache
		pwaModeMu.RUnlock()
		return val
	}
	pwaModeMu.RUnlock()

	pwaModeMu.Lock()
	defer pwaModeMu.Unlock()

	// Double-check after acquiring write lock
	if pwaModeCache != nil && time.Since(pwaModeCachedAt) < pwaModeCacheTTL {
		return *pwaModeCache
	}

	setting, _ := settingRepo.GetByKey("enforce_pwa_mode")
	enforced := setting != nil && setting.Value != nil && *setting.Value == "1"
	pwaModeCache = &enforced
	pwaModeCachedAt = time.Now()
	return enforced
}
