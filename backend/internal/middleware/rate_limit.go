package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimit limits requests per user per minute using Redis counter.
func RateLimit(rdb *redis.Client, maxPerMinute int) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := c.Get(CtxUserID)
		if !ok {
			c.Next()
			return
		}

		ctx := c.Request.Context()
		now := time.Now()
		minute := now.Format("200601021504") // yyyyMMddHHmm
		key := fmt.Sprintf("ratelimit:%v:%s", userID, minute)

		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			c.Next()
			return
		}
		if count == 1 {
			rdb.Expire(ctx, key, 70*time.Second)
		}

		if count > int64(maxPerMinute) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
				"code":  "RATE_LIMIT_EXCEEDED",
			})
			return
		}

		c.Next()
	}
}
