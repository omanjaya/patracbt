package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/omanjaya/patra/pkg/jwt"
	"github.com/omanjaya/patra/pkg/response"
)

func AuthMiddleware(accessSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			response.Unauthorized(c, "Token tidak ditemukan")
			c.Abort()
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(c, "Format token tidak valid")
			c.Abort()
			return
		}

		claims, err := jwt.ValidateToken(parts[1], accessSecret)
		if err != nil {
			response.Unauthorized(c, "Token tidak valid atau expired")
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			response.Forbidden(c, "Akses ditolak")
			c.Abort()
			return
		}

		roleStr, ok := role.(string)
		if !ok {
			response.Forbidden(c, "Akses ditolak")
			c.Abort()
			return
		}

		for _, allowed := range allowedRoles {
			if roleStr == allowed {
				c.Next()
				return
			}
		}

		response.Forbidden(c, "Anda tidak memiliki akses untuk fitur ini")
		c.Abort()
	}
}
