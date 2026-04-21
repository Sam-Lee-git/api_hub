package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/youorg/ai-proxy-platform/backend/internal/db/cache"
)

// RateLimit limits requests per user per minute using the cache as a counter.
func RateLimit(c cache.Client, maxPerMinute int) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID, ok := ctx.Get(CtxUserID)
		if !ok {
			ctx.Next()
			return
		}

		reqCtx := ctx.Request.Context()
		minute := time.Now().Format("200601021504") // yyyyMMddHHmm
		key := fmt.Sprintf("ratelimit:%v:%s", userID, minute)

		count, err := c.Incr(reqCtx, key)
		if err != nil {
			ctx.Next()
			return
		}
		if count == 1 {
			c.Expire(reqCtx, key, 70*time.Second)
		}

		if count > int64(maxPerMinute) {
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
				"code":  "RATE_LIMIT_EXCEEDED",
			})
			return
		}

		ctx.Next()
	}
}
