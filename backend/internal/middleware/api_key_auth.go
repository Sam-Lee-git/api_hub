package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/youorg/ai-proxy-platform/backend/internal/repository"
	"github.com/youorg/ai-proxy-platform/backend/pkg/crypto"
)

const (
	CtxUserID    = "userID"
	CtxAPIKeyID  = "apiKeyID"
	CtxUserRole  = "userRole"
)

// APIKeyAuth validates the platform API key (sk-...) and attaches userID + apiKeyID to context.
// Caches the key-to-user mapping in Redis for 5 minutes.
func APIKeyAuth(keyRepo repository.APIKeyRepository, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing Authorization header"})
			return
		}

		key := strings.TrimPrefix(authHeader, "Bearer ")
		if key == authHeader || !strings.HasPrefix(key, "sk-") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid API key format"})
			return
		}

		keyHash := crypto.SHA256Hex(key)
		cacheKey := fmt.Sprintf("apikey:%s", keyHash)

		// Try Redis cache first
		ctx := c.Request.Context()
		cached, err := rdb.HGetAll(ctx, cacheKey).Result()
		if err == nil && len(cached) > 0 {
			var userID, apiKeyID int64
			fmt.Sscanf(cached["userID"], "%d", &userID)
			fmt.Sscanf(cached["apiKeyID"], "%d", &apiKeyID)
			c.Set(CtxUserID, userID)
			c.Set(CtxAPIKeyID, apiKeyID)
			c.Next()
			return
		}

		// Cache miss: query DB
		apiKey, err := keyRepo.FindByHash(ctx, keyHash)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		if apiKey == nil || !apiKey.IsActive() {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or revoked API key"})
			return
		}

		// Cache for 5 minutes
		rdb.HSet(ctx, cacheKey,
			"userID", fmt.Sprintf("%d", apiKey.UserID),
			"apiKeyID", fmt.Sprintf("%d", apiKey.ID),
		)
		rdb.Expire(ctx, cacheKey, 5*time.Minute)

		c.Set(CtxUserID, apiKey.UserID)
		c.Set(CtxAPIKeyID, apiKey.ID)
		c.Next()
	}
}
