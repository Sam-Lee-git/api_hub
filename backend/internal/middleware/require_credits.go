package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/youorg/ai-proxy-platform/backend/internal/service"
)

// RequireCredits rejects requests with 402 if the user has zero or negative balance.
// The balance check uses Redis cache (30s TTL) to avoid DB hits on every call.
func RequireCredits(creditSvc *service.CreditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := c.Get(CtxUserID)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		has, err := creditSvc.HasCredits(c.Request.Context(), userID.(int64))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		if !has {
			c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
				"error": "Insufficient credits. Please top up your account.",
				"code":  "INSUFFICIENT_CREDITS",
			})
			return
		}

		c.Next()
	}
}
