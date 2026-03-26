package middlewares

import (
	"fmt"
	"net/http"
	"time"

	"github.com/06babyshark06/JQKStudy/services/api-gateway/redis"
	"github.com/06babyshark06/JQKStudy/shared/env"
	"github.com/gin-gonic/gin"
)

func RateLimiter() gin.HandlerFunc {
	// Load config
	limit := env.GetInt("RATE_LIMIT_MAX", 100)
	window := time.Duration(env.GetInt("RATE_LIMIT_WINDOW_SECONDS", 60)) * time.Second

	return func(c *gin.Context) {
		// Identify user (by User ID if authenticated, else by IP)
		var key string
		userID, exists := c.Get("userID")
		if exists {
			key = fmt.Sprintf("user:%v", userID)
		} else {
			key = fmt.Sprintf("ip:%s", c.ClientIP())
		}

		count, ok, err := redis.CheckRateLimit(key, limit, window)
		if err != nil {
			// In case of Redis error, we usually allow the request to pass 
			// but log the error (Fail-open strategy)
			fmt.Printf("Rate limit error: %v\n", err)
			c.Next()
			return
		}

		// Add headers for visibility
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", max(0, limit-count)))

		if !ok {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Too Many Requests",
				"message": fmt.Sprintf("You have exceeded the rate limit of %d requests per %v. Please try again later.", limit, window),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
