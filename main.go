package main

import (
	"fmt"
	"math"
	"net/http"
	"rate-limiter/limiter"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	// Initialising to 5 reqs/sec, Burst = 10
	rl := limiter.NewRateLimiter(rdb, 5, 10)
	r := gin.Default()

	rateLimitMiddleware := func(c *gin.Context) {
		ip := c.ClientIP()

		allowed, remaining, resetTime, retryMs, err := rl.Allow(ip)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			c.Abort()
			return
		}

		c.Header("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime, 10))

		if !allowed {
			//Ensure at least 1ms safety
			if retryMs <= 0 {
				retryMs = 1
			}

			waitSecs := int(math.Ceil(float64(retryMs) / 1000.0))
			c.Header("Retry-After", strconv.Itoa(waitSecs))

			var retryStr string
			if retryMs >= 1000 {
				//Convert to seconds with 2 decimal places (e.g. :- 1250ms -> "1.25s")
				retryStr = fmt.Sprintf("%.2fs", float64(retryMs)/1000.0)
			} else {
				//Keep in milliseconds (e.g. 200ms -> "200ms")
				retryStr = fmt.Sprintf("%dms", retryMs)
			}

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"retry_after": retryStr,
			})
			c.Abort()
			return
		}

		c.Next()
	}

	r.GET("/ping", rateLimitMiddleware, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	r.Run(":8080")
}
