package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/youorg/ai-proxy-platform/backend/internal/db/cache"
	"github.com/youorg/ai-proxy-platform/backend/internal/repository"
	"github.com/youorg/ai-proxy-platform/backend/pkg/crypto"
)

const (
	CtxUserID   = "userID"
	CtxAPIKeyID = "apiKeyID"
	CtxUserRole = "userRole"
)

// APIKeyAuth validates the platform API key (sk-...) and attaches userID + apiKeyID to context.
// Caches the key-to-user mapping for 5 minutes as "userID:apiKeyID".
func APIKeyAuth(keyRepo repository.APIKeyRepository, c cache.Client) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing Authorization header"})
			return
		}

		key := strings.TrimPrefix(authHeader, "Bearer ")
		if key == authHeader || !strings.HasPrefix(key, "sk-") {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid API key format"})
			return
		}

		keyHash := crypto.SHA256Hex(key)
		cacheKey := fmt.Sprintf("apikey:%s", keyHash)
		reqCtx := ctx.Request.Context()

		if val, err := c.Get(reqCtx, cacheKey); err == nil {
			var userID, apiKeyID int64
			fmt.Sscanf(val, "%d:%d", &userID, &apiKeyID)
			ctx.Set(CtxUserID, userID)
			ctx.Set(CtxAPIKeyID, apiKeyID)
			ctx.Next()
			return
		}

		apiKey, err := keyRepo.FindByHash(reqCtx, keyHash)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		if apiKey == nil || !apiKey.IsActive() {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or revoked API key"})
			return
		}

		c.Set(reqCtx, cacheKey, fmt.Sprintf("%d:%d", apiKey.UserID, apiKey.ID), 5*time.Minute)

		ctx.Set(CtxUserID, apiKey.UserID)
		ctx.Set(CtxAPIKeyID, apiKey.ID)
		ctx.Next()
	}
}
