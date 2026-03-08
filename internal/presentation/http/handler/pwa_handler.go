package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/omanjaya/patra/pkg/response"
)

type PWAHandler struct{}

func NewPWAHandler() *PWAHandler {
	return &PWAHandler{}
}

// ValidatePWA checks if the request originates from a PWA (standalone) context.
// GET /pwa/validate
func (h *PWAHandler) ValidatePWA(c *gin.Context) {
	isPWA := false

	// Method 1: Check custom header set by the PWA client
	if c.GetHeader("X-PWA-Mode") == "standalone" {
		isPWA = true
	}

	// Method 2: Check Sec-Fetch-Dest / display-mode hints
	if secFetchDest := c.GetHeader("Sec-Fetch-Dest"); secFetchDest == "empty" {
		// Additional check: standalone PWAs typically do not include a Referer with the origin domain in the same way browsers do
		if c.GetHeader("X-Requested-With") != "" {
			isPWA = true
		}
	}

	// Method 3: Check the display_mode query parameter (client can append ?display_mode=standalone)
	if c.Query("display_mode") == "standalone" {
		isPWA = true
	}

	// Method 4: Check User-Agent for common PWA / WebView indicators
	ua := strings.ToLower(c.GetHeader("User-Agent"))
	pwaIndicators := []string{"pwa", "standalone", "wv", "webview"}
	for _, indicator := range pwaIndicators {
		if strings.Contains(ua, indicator) {
			isPWA = true
			break
		}
	}

	c.JSON(http.StatusOK, response.Response{
		Success: true,
		Message: "OK",
		Data: gin.H{
			"is_pwa": isPWA,
		},
	})
}
