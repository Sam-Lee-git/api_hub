package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	jwtpkg "github.com/youorg/ai-proxy-platform/backend/pkg/jwt"
)

// JWTAuth validates the JWT access token and attaches userID + role to context.
func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing Authorization header"})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == authHeader {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			return
		}

		claims, err := jwtpkg.Verify(tokenStr, secret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		c.Set(CtxUserID, claims.UserID)
		c.Set(CtxUserRole, claims.Role)
		c.Next()
	}
}
