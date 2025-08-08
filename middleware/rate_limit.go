package middleware

import (
	"exam-system/utils"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func RateLimitMiddleware(redisClient *utils.RedisClient, limit int, window time.Duration, keyPrefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get client identifier (IP address or user ID if authenticated)
		var identifier string
		
		// Try to get user ID from context first
		if userID, exists := c.Get(UserIDKey); exists {
			identifier = fmt.Sprintf("user_%d", userID.(uint))
		} else {
			// Fall back to IP address
			identifier = c.ClientIP()
		}

		// Create rate limit key
		key := fmt.Sprintf("rate_limit:%s:%s", keyPrefix, identifier)

		// Check rate limit
		isLimited, err := redisClient.IsRateLimited(key, limit, window)
		if err != nil {
			// Log error but don't block request if Redis is down
			c.Header("X-RateLimit-Error", "Rate limit check failed")
			c.Next()
			return
		}

		if isLimited {
			// Get TTL for reset time
			ttl, _ := redisClient.TTL(key)
			resetTime := time.Now().Add(ttl).Unix()

			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime))
			c.Header("Retry-After", fmt.Sprintf("%d", int(ttl.Seconds())))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"code":    "RATE_LIMIT_EXCEEDED",
				"message": fmt.Sprintf("Too many requests. Limit: %d per %v", limit, window),
				"retry_after": int(ttl.Seconds()),
			})
			c.Abort()
			return
		}

		// Add rate limit headers
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		
		c.Next()
	}
}

// LoginRateLimitMiddleware creates rate limiting specifically for login endpoints
func LoginRateLimitMiddleware(redisClient *utils.RedisClient, limit int, window time.Duration) gin.HandlerFunc {
	return RateLimitMiddleware(redisClient, limit, window, "login")
}

// SubmitRateLimitMiddleware creates rate limiting specifically for exam submission endpoints
func SubmitRateLimitMiddleware(redisClient *utils.RedisClient, limit int, window time.Duration) gin.HandlerFunc {
	return RateLimitMiddleware(redisClient, limit, window, "submit")
}

// APIRateLimitMiddleware creates general API rate limiting
func APIRateLimitMiddleware(redisClient *utils.RedisClient, limit int, window time.Duration) gin.HandlerFunc {
	return RateLimitMiddleware(redisClient, limit, window, "api")
}

// IPBasedRateLimitMiddleware creates rate limiting based only on IP address
func IPBasedRateLimitMiddleware(redisClient *utils.RedisClient, limit int, window time.Duration, keyPrefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Use IP address as identifier
		identifier := c.ClientIP()
		key := fmt.Sprintf("rate_limit:%s:%s", keyPrefix, identifier)

		// Check rate limit
		isLimited, err := redisClient.IsRateLimited(key, limit, window)
		if err != nil {
			// Log error but don't block request if Redis is down
			c.Header("X-RateLimit-Error", "Rate limit check failed")
			c.Next()
			return
		}

		if isLimited {
			// Get TTL for reset time
			ttl, _ := redisClient.TTL(key)
			resetTime := time.Now().Add(ttl).Unix()

			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime))
			c.Header("Retry-After", fmt.Sprintf("%d", int(ttl.Seconds())))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"code":    "RATE_LIMIT_EXCEEDED",
				"message": fmt.Sprintf("Too many requests from this IP. Limit: %d per %v", limit, window),
				"retry_after": int(ttl.Seconds()),
			})
			c.Abort()
			return
		}

		// Add rate limit headers
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		
		c.Next()
	}
}

