package middleware

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	RequestIDKey = "request_id"
)

type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func LoggingMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate request ID
		requestID := uuid.New().String()
		c.Set(RequestIDKey, requestID)

		// Add request ID to response headers
		c.Header("X-Request-ID", requestID)

		// Start timer
		start := time.Now()

		// Read request body
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// Create response writer wrapper to capture response body
		responseBody := &bytes.Buffer{}
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:          responseBody,
		}
		c.Writer = writer

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get user ID if available
		var userID interface{}
		if uid, exists := c.Get(UserIDKey); exists {
			userID = uid
		}

		// Prepare log fields
		fields := logrus.Fields{
			"request_id":     requestID,
			"method":         c.Request.Method,
			"path":           c.Request.URL.Path,
			"query":          c.Request.URL.RawQuery,
			"status":         c.Writer.Status(),
			"latency":        latency.Milliseconds(),
			"latency_human":  latency.String(),
			"client_ip":      c.ClientIP(),
			"user_agent":     c.Request.UserAgent(),
			"referer":        c.Request.Referer(),
			"content_length": c.Request.ContentLength,
			"response_size":  c.Writer.Size(),
		}

		if userID != nil {
			fields["user_id"] = userID
		}

		// Add request body for non-GET requests (but exclude sensitive data)
		if c.Request.Method != "GET" && len(requestBody) > 0 && len(requestBody) < 1024 {
			// Don't log request body for auth endpoints to avoid logging passwords
			if !isAuthEndpoint(c.Request.URL.Path) {
				fields["request_body"] = string(requestBody)
			}
		}

		// Add response body for errors or if response is small
		if c.Writer.Status() >= 400 || responseBody.Len() < 512 {
			fields["response_body"] = responseBody.String()
		}

		// Log based on status code
		entry := logger.WithFields(fields)

		switch {
		case c.Writer.Status() >= 500:
			entry.Error("Server error")
		case c.Writer.Status() >= 400:
			entry.Warn("Client error")
		case c.Writer.Status() >= 300:
			entry.Info("Redirection")
		default:
			entry.Info("Request completed")
		}

		// Log slow requests
		if latency > 1*time.Second {
			logger.WithFields(fields).Warn("Slow request detected")
		}
	}
}

// RequestIDMiddleware adds request ID to context without full logging
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := uuid.New().String()
		c.Set(RequestIDKey, requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// GetRequestID extracts request ID from context
func GetRequestID(c *gin.Context) string {
	if requestID, exists := c.Get(RequestIDKey); exists {
		return requestID.(string)
	}
	return ""
}

// isAuthEndpoint checks if the path is an authentication endpoint
func isAuthEndpoint(path string) bool {
	authPaths := []string{
		"/api/v1/auth/login",
		"/api/v1/auth/register",
		"/api/v1/auth/refresh",
	}

	for _, authPath := range authPaths {
		if path == authPath {
			return true
		}
	}
	return false
}

// ErrorLoggingMiddleware logs errors that occur during request processing
func ErrorLoggingMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Log any errors that occurred
		if len(c.Errors) > 0 {
			requestID := GetRequestID(c)
			
			for _, err := range c.Errors {
				logger.WithFields(logrus.Fields{
					"request_id": requestID,
					"method":     c.Request.Method,
					"path":       c.Request.URL.Path,
					"error_type": err.Type,
					"error":      err.Error(),
				}).Error("Request error occurred")
			}
		}
	}
}

// StructuredErrorResponse creates a consistent error response format
func StructuredErrorResponse(c *gin.Context, statusCode int, errorCode string, message string, details interface{}) {
	requestID := GetRequestID(c)
	
	response := gin.H{
		"error":      message,
		"code":       errorCode,
		"request_id": requestID,
		"timestamp":  time.Now().Unix(),
	}

	if details != nil {
		response["details"] = details
	}

	c.JSON(statusCode, response)
}

