package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/omanjaya/patra/pkg/jwt"
	"gorm.io/gorm"
)

// SingleSession ensures only one active session per user.
// Uses a minimal SELECT (login_token only) instead of a full FindByID to reduce DB overhead.
// Rejects the request if login_token is "REVOKED".
func SingleSession(db *gorm.DB, accessSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.Next()
			return
		}
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 {
			c.Next()
			return
		}

		claims, err := jwt.ValidateToken(parts[1], accessSecret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Token tidak valid"})
			c.Abort()
			return
		}

		// Minimal query: only fetch login_token to avoid loading full user + profile.
		var loginToken *string
		result := db.Raw("SELECT login_token FROM users WHERE id = ? AND deleted_at IS NULL", claims.UserID).
			Scan(&loginToken)
		if result.Error != nil || result.RowsAffected == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "User tidak ditemukan"})
			c.Abort()
			return
		}

		// Compare JWT's login_token claim with the DB's login_token
		// If tokens don't match, the session was invalidated by a new login
		jwtLoginToken := claims.LoginToken
		if loginToken == nil || *loginToken == "REVOKED" || *loginToken != jwtLoginToken {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Sesi Anda telah berakhir. Silakan login kembali.",
				"code":    "SESSION_INVALIDATED",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
