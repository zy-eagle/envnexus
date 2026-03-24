package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// PermissionChecker is implemented by the RBAC service.
type PermissionChecker interface {
	HasPermission(ctx context.Context, userID, permission string) (bool, error)
}

// RequirePermission returns a middleware that checks if the authenticated user
// has the given permission. It reads user_id from the JWT context set by JWTAuth.
func RequirePermission(checker PermissionChecker, permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "unauthorized", "message": "not authenticated"},
			})
			return
		}

		ok, err := checker.HasPermission(c.Request.Context(), userID.(string), permission)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{"code": "internal_error", "message": "permission check failed"},
			})
			return
		}

		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":    "forbidden",
					"message": "you do not have permission to perform this action",
					"required": permission,
				},
			})
			return
		}

		c.Next()
	}
}
