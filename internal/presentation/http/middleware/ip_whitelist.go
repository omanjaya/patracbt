package middleware

import (
	"net"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/response"
)

var (
	ipWhitelistEnabledCache    *bool
	ipWhitelistIPsCache        *string
	ipWhitelistCachedAt        time.Time
	ipWhitelistMu              sync.RWMutex
	ipWhitelistCacheTTL        = 2 * time.Minute
)

// IPWhitelist blocks requests from IPs not in the whitelist when ip_whitelist_enabled = "1".
// Admin role is always allowed regardless of IP.
func IPWhitelist(settingRepo repository.SettingRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Admin is always allowed
		role, _ := c.Get("role")
		if role == entity.RoleAdmin {
			c.Next()
			return
		}

		enabled, ips := getIPWhitelistSettings(settingRepo)
		if !enabled {
			c.Next()
			return
		}

		clientIP := c.ClientIP()
		if isIPAllowed(clientIP, ips) {
			c.Next()
			return
		}

		response.Forbidden(c, "Akses dari IP Anda tidak diizinkan. Hubungi administrator.")
		c.Abort()
	}
}

func getIPWhitelistSettings(settingRepo repository.SettingRepository) (bool, string) {
	ipWhitelistMu.RLock()
	if ipWhitelistEnabledCache != nil && time.Since(ipWhitelistCachedAt) < ipWhitelistCacheTTL {
		enabled := *ipWhitelistEnabledCache
		ips := ""
		if ipWhitelistIPsCache != nil {
			ips = *ipWhitelistIPsCache
		}
		ipWhitelistMu.RUnlock()
		return enabled, ips
	}
	ipWhitelistMu.RUnlock()

	ipWhitelistMu.Lock()
	defer ipWhitelistMu.Unlock()

	// Double-check after acquiring write lock
	if ipWhitelistEnabledCache != nil && time.Since(ipWhitelistCachedAt) < ipWhitelistCacheTTL {
		enabled := *ipWhitelistEnabledCache
		ips := ""
		if ipWhitelistIPsCache != nil {
			ips = *ipWhitelistIPsCache
		}
		return enabled, ips
	}

	// Fetch enabled setting
	setting, err := settingRepo.GetByKey("ip_whitelist_enabled")
	enabled := err == nil && setting != nil && setting.Value != nil && *setting.Value == "1"
	ipWhitelistEnabledCache = &enabled

	// Fetch IPs setting
	ips := ""
	if enabled {
		ipsSetting, err := settingRepo.GetByKey("ip_whitelist_ips")
		if err == nil && ipsSetting != nil && ipsSetting.Value != nil {
			ips = *ipsSetting.Value
		}
	}
	ipWhitelistIPsCache = &ips
	ipWhitelistCachedAt = time.Now()

	return enabled, ips
}

func isIPAllowed(clientIP string, whitelist string) bool {
	if whitelist == "" {
		return false
	}

	ip := net.ParseIP(clientIP)
	if ip == nil {
		return false
	}

	entries := strings.Split(whitelist, ",")
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		// Check if it's a CIDR notation
		if strings.Contains(entry, "/") {
			_, network, err := net.ParseCIDR(entry)
			if err != nil {
				continue
			}
			if network.Contains(ip) {
				return true
			}
		} else {
			// Single IP comparison
			if entry == clientIP {
				return true
			}
		}
	}

	return false
}
