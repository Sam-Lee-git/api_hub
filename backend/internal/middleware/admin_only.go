package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// AdminOnly rejects requests from non-admin users.
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get(CtxUserRole)
		if role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}
		c.Next()
	}
}
