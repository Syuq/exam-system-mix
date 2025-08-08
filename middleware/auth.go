package middleware

import (
	"exam-system/models"
	"exam-system/services"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	UserIDKey   = "user_id"
	UserKey     = "user"
	ClaimsKey   = "claims"
	IsAdminKey  = "is_admin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Authorization header required",
				"code":    "MISSING_AUTH_HEADER",
				"message": "Please provide a valid authorization token",
			})
			c.Abort()
			return
		}

		// Check if token has Bearer prefix
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid authorization header format",
				"code":    "INVALID_AUTH_FORMAT",
				"message": "Authorization header must be in format: Bearer <token>",
			})
			c.Abort()
			return
		}

		token := tokenParts[1]

		// Create a temporary auth service to validate token
		// In a real application, you might want to inject this dependency
		authService := &services.AuthService{}
		claims, err := authService.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid or expired token",
				"code":    "INVALID_TOKEN",
				"message": "Please login again to get a valid token",
			})
			c.Abort()
			return
		}

		// Set user information in context
		c.Set(UserIDKey, claims.UserID)
		c.Set(ClaimsKey, claims)
		c.Set(IsAdminKey, claims.Role == models.RoleAdmin)

		c.Next()
	}
}

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdmin, exists := c.Get(IsAdminKey)
		if !exists || !isAdmin.(bool) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "Admin access required",
				"code":    "INSUFFICIENT_PERMISSIONS",
				"message": "This endpoint requires administrator privileges",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetUserID extracts user ID from context
func GetUserID(c *gin.Context) (uint, bool) {
	userID, exists := c.Get(UserIDKey)
	if !exists {
		return 0, false
	}
	return userID.(uint), true
}

// GetClaims extracts JWT claims from context
func GetClaims(c *gin.Context) (*services.Claims, bool) {
	claims, exists := c.Get(ClaimsKey)
	if !exists {
		return nil, false
	}
	return claims.(*services.Claims), true
}

// IsAdmin checks if current user is admin
func IsAdmin(c *gin.Context) bool {
	isAdmin, exists := c.Get(IsAdminKey)
	if !exists {
		return false
	}
	return isAdmin.(bool)
}

// RequireAuth is a helper function to check authentication in handlers
func RequireAuth(c *gin.Context) (uint, bool) {
	userID, exists := GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Authentication required",
			"code":    "AUTH_REQUIRED",
			"message": "Please login to access this resource",
		})
		return 0, false
	}
	return userID, true
}

// RequireAdmin is a helper function to check admin privileges in handlers
func RequireAdmin(c *gin.Context) bool {
	if !IsAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Admin access required",
			"code":    "ADMIN_REQUIRED",
			"message": "This resource requires administrator privileges",
		})
		return false
	}
	return true
}

