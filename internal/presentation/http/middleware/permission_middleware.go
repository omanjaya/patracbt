package middleware

import (
	"log"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/omanjaya/patra/internal/domain/entity"
	"gorm.io/gorm"
)

// validPermissionRe ensures permission strings contain only lowercase letters,
// digits, underscores, and hyphens. This prevents potential injection in SQL
// IN clauses.
var validPermissionRe = regexp.MustCompile(`^[a-z0-9_-]+$`)

// PermissionMiddleware checks if the authenticated user has at least one of the
// specified permissions.  Permissions can come from two sources:
//   - role_permissions  (role → permissions via the roles.guard_name ↔ users.role link)
//   - user_permissions  (direct per-user grant)
//
// Users whose role is "admin" always pass (Super Admin bypass).
func PermissionMiddleware(db *gorm.DB, permissions ...string) gin.HandlerFunc {
	// Validate permission strings at registration time (startup) to prevent
	// potential injection in the SQL IN clause.
	for _, p := range permissions {
		if !validPermissionRe.MatchString(p) {
			log.Fatalf("PermissionMiddleware: invalid permission string %q — must match ^[a-z0-9_-]+$", p)
		}
	}

	return func(c *gin.Context) {
		// Admin bypass — admin always has full access
		role, _ := c.Get("role")
		if role == entity.RoleAdmin {
			c.Next()
			return
		}

		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Unauthorized"})
			c.Abort()
			return
		}

		// Permission check via UNION of two sources:
		//
		// 1) Role-based permissions: look up the user's role (from users.role),
		//    find the matching roles row by guard_name, then join through
		//    role_permissions to get all permission names granted to that role.
		//
		// 2) Direct user permissions: join user_permissions to permissions
		//    for grants assigned directly to this specific user (overrides/extras).
		//
		// The UNION deduplicates so a permission present in both paths is counted
		// once. We then count how many of the requested permission names appear;
		// if count > 0 the user has at least one required permission.
		var count int64
		db.Raw(`
			SELECT COUNT(*) FROM (
				SELECT p.name FROM permissions p
				JOIN role_permissions rp ON rp.permission_id = p.id
				JOIN roles r ON r.id = rp.role_id
				WHERE r.guard_name = (SELECT role FROM users WHERE id = ? AND deleted_at IS NULL)
				  AND p.name IN ?
				  AND p.deleted_at IS NULL
				UNION
				SELECT p.name FROM permissions p
				JOIN user_permissions up ON up.permission_id = p.id
				WHERE up.user_id = ?
				  AND p.name IN ?
				  AND p.deleted_at IS NULL
			) combined
		`, userID, permissions, userID, permissions).Count(&count)

		if count == 0 {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "Anda tidak memiliki izin untuk mengakses fitur ini",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
