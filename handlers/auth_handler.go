package handlers

import (
	"exam-system/middleware"
	"exam-system/services"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type AuthHandler struct {
	authService *services.AuthService
	logger      *logrus.Logger
}

func NewAuthHandler(authService *services.AuthService, logger *logrus.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger,
	}
}

// Register handles user registration
// @Summary Register a new user
// @Description Create a new user account
// @Tags auth
// @Accept json
// @Produce json
// @Param request body services.RegisterRequest true "Registration details"
// @Success 201 {object} map[string]interface{} "User created successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 409 {object} map[string]interface{} "User already exists"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req services.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.StructuredErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request data", err.Error())
		return
	}

	user, err := h.authService.Register(req)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"email":      req.Email,
			"username":   req.Username,
			"request_id": middleware.GetRequestID(c),
		}).WithError(err).Error("Failed to register user")

		if err.Error() == "user with email or username already exists" {
			middleware.StructuredErrorResponse(c, http.StatusConflict, "USER_EXISTS", "User with this email or username already exists", nil)
			return
		}

		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "REGISTRATION_FAILED", "Failed to register user", nil)
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":    user.ID,
		"email":      user.Email,
		"username":   user.Username,
		"request_id": middleware.GetRequestID(c),
	}).Info("User registered successfully")

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully",
		"user":    user.ToResponse(),
	})
}

// Login handles user authentication
// @Summary Login user
// @Description Authenticate user and return JWT tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param request body services.LoginRequest true "Login credentials"
// @Success 200 {object} map[string]interface{} "Login successful"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Invalid credentials"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.StructuredErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request data", err.Error())
		return
	}

	user, tokenResponse, err := h.authService.Login(req)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"email":      req.Email,
			"request_id": middleware.GetRequestID(c),
		}).WithError(err).Warn("Failed login attempt")

		if err.Error() == "invalid credentials" {
			middleware.StructuredErrorResponse(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password", nil)
			return
		}

		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "LOGIN_FAILED", "Failed to authenticate user", nil)
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":    user.ID,
		"email":      user.Email,
		"request_id": middleware.GetRequestID(c),
	}).Info("User logged in successfully")

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"user":    user.ToResponse(),
		"tokens":  tokenResponse,
	})
}

// RefreshToken handles token refresh
// @Summary Refresh access token
// @Description Get a new access token using refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body services.RefreshTokenRequest true "Refresh token"
// @Success 200 {object} map[string]interface{} "Token refreshed successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Invalid refresh token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req services.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.StructuredErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request data", err.Error())
		return
	}

	tokenResponse, err := h.authService.RefreshToken(req)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"request_id": middleware.GetRequestID(c),
		}).WithError(err).Warn("Failed to refresh token")

		middleware.StructuredErrorResponse(c, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "Invalid or expired refresh token", nil)
		return
	}

	h.logger.WithFields(logrus.Fields{
		"request_id": middleware.GetRequestID(c),
	}).Info("Token refreshed successfully")

	c.JSON(http.StatusOK, gin.H{
		"message": "Token refreshed successfully",
		"tokens":  tokenResponse,
	})
}

// Logout handles user logout
// @Summary Logout user
// @Description Invalidate user's refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Logout successful"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		middleware.StructuredErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	if err := h.authService.Logout(userID); err != nil {
		h.logger.WithFields(logrus.Fields{
			"user_id":    userID,
			"request_id": middleware.GetRequestID(c),
		}).WithError(err).Error("Failed to logout user")

		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "LOGOUT_FAILED", "Failed to logout user", nil)
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":    userID,
		"request_id": middleware.GetRequestID(c),
	}).Info("User logged out successfully")

	c.JSON(http.StatusOK, gin.H{
		"message": "Logout successful",
	})
}

// GetProfile returns current user profile (convenience endpoint)
// @Summary Get current user profile
// @Description Get the profile of the currently authenticated user
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "User profile"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/auth/profile [get]
func (h *AuthHandler) GetProfile(c *gin.Context) {
	claims, exists := middleware.GetClaims(c)
	if !exists {
		middleware.StructuredErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":       claims.UserID,
			"email":    claims.Email,
			"username": claims.Username,
			"role":     claims.Role,
		},
	})
}

