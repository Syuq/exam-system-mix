package handlers

import (
	"exam-system/middleware"
	"exam-system/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type UserHandler struct {
	userService *services.UserService
	logger      *logrus.Logger
}

func NewUserHandler(userService *services.UserService, logger *logrus.Logger) *UserHandler {
	return &UserHandler{
		userService: userService,
		logger:      logger,
	}
}

// GetProfile returns the current user's profile
// @Summary Get user profile
// @Description Get the profile of the currently authenticated user
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "User profile"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/users/profile [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		middleware.StructuredErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	user, err := h.userService.GetProfile(userID)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"user_id":    userID,
			"request_id": middleware.GetRequestID(c),
		}).WithError(err).Error("Failed to get user profile")

		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "PROFILE_FETCH_FAILED", "Failed to get user profile", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user.ToResponse(),
	})
}

// UpdateProfile updates the current user's profile
// @Summary Update user profile
// @Description Update the profile of the currently authenticated user
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body services.UpdateProfileRequest true "Profile update data"
// @Success 200 {object} map[string]interface{} "Profile updated successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 409 {object} map[string]interface{} "Username already taken"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/users/profile [put]
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		middleware.StructuredErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	var req services.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.StructuredErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request data", err.Error())
		return
	}

	user, err := h.userService.UpdateProfile(userID, req)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"user_id":    userID,
			"username":   req.Username,
			"request_id": middleware.GetRequestID(c),
		}).WithError(err).Error("Failed to update user profile")

		if err.Error() == "username already taken" {
			middleware.StructuredErrorResponse(c, http.StatusConflict, "USERNAME_TAKEN", "Username is already taken", nil)
			return
		}

		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "PROFILE_UPDATE_FAILED", "Failed to update user profile", nil)
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":    userID,
		"username":   user.Username,
		"request_id": middleware.GetRequestID(c),
	}).Info("User profile updated successfully")

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile updated successfully",
		"user":    user.ToResponse(),
	})
}

// ChangePassword changes the current user's password
// @Summary Change user password
// @Description Change the password of the currently authenticated user
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body services.ChangePasswordRequest true "Password change data"
// @Success 200 {object} map[string]interface{} "Password changed successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized or incorrect current password"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/users/change-password [post]
func (h *UserHandler) ChangePassword(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		middleware.StructuredErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	var req services.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.StructuredErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request data", err.Error())
		return
	}

	if err := h.userService.ChangePassword(userID, req); err != nil {
		h.logger.WithFields(logrus.Fields{
			"user_id":    userID,
			"request_id": middleware.GetRequestID(c),
		}).WithError(err).Error("Failed to change user password")

		if err.Error() == "current password is incorrect" {
			middleware.StructuredErrorResponse(c, http.StatusUnauthorized, "INCORRECT_PASSWORD", "Current password is incorrect", nil)
			return
		}

		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "PASSWORD_CHANGE_FAILED", "Failed to change password", nil)
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":    userID,
		"request_id": middleware.GetRequestID(c),
	}).Info("User password changed successfully")

	c.JSON(http.StatusOK, gin.H{
		"message": "Password changed successfully",
	})
}

// GetUsers returns a paginated list of users (admin only)
// @Summary Get users list
// @Description Get a paginated list of all users (admin only)
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Param search query string false "Search term"
// @Success 200 {object} services.UserListResponse "Users list"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/users [get]
func (h *UserHandler) GetUsers(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	search := c.Query("search")

	// Validate pagination parameters
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	users, err := h.userService.GetUsers(page, pageSize, search)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"page":       page,
			"page_size":  pageSize,
			"search":     search,
			"request_id": middleware.GetRequestID(c),
		}).WithError(err).Error("Failed to get users")

		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "USERS_FETCH_FAILED", "Failed to get users", nil)
		return
	}

	c.JSON(http.StatusOK, users)
}

// GetUser returns a specific user by ID (admin only)
// @Summary Get user by ID
// @Description Get a specific user by their ID (admin only)
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} map[string]interface{} "User details"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		middleware.StructuredErrorResponse(c, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID", nil)
		return
	}

	user, err := h.userService.GetUser(uint(userID))
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"target_user_id": userID,
			"request_id":     middleware.GetRequestID(c),
		}).WithError(err).Error("Failed to get user")

		if err.Error() == "user not found" {
			middleware.StructuredErrorResponse(c, http.StatusNotFound, "USER_NOT_FOUND", "User not found", nil)
			return
		}

		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "USER_FETCH_FAILED", "Failed to get user", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user.ToResponse(),
	})
}

// UpdateUser updates a specific user (admin only)
// @Summary Update user
// @Description Update a specific user (admin only)
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Param request body services.UpdateUserRequest true "User update data"
// @Success 200 {object} map[string]interface{} "User updated successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 409 {object} map[string]interface{} "Email or username already taken"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/users/{id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		middleware.StructuredErrorResponse(c, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID", nil)
		return
	}

	var req services.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.StructuredErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request data", err.Error())
		return
	}

	user, err := h.userService.UpdateUser(uint(userID), req)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"target_user_id": userID,
			"email":          req.Email,
			"username":       req.Username,
			"request_id":     middleware.GetRequestID(c),
		}).WithError(err).Error("Failed to update user")

		if err.Error() == "user not found" {
			middleware.StructuredErrorResponse(c, http.StatusNotFound, "USER_NOT_FOUND", "User not found", nil)
			return
		}

		if err.Error() == "email already taken" || err.Error() == "username already taken" {
			middleware.StructuredErrorResponse(c, http.StatusConflict, "DUPLICATE_DATA", err.Error(), nil)
			return
		}

		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "USER_UPDATE_FAILED", "Failed to update user", nil)
		return
	}

	h.logger.WithFields(logrus.Fields{
		"target_user_id": userID,
		"email":          user.Email,
		"username":       user.Username,
		"request_id":     middleware.GetRequestID(c),
	}).Info("User updated successfully")

	c.JSON(http.StatusOK, gin.H{
		"message": "User updated successfully",
		"user":    user.ToResponse(),
	})
}

// DeleteUser deletes a specific user (admin only)
// @Summary Delete user
// @Description Delete a specific user (admin only)
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} map[string]interface{} "User deleted successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		middleware.StructuredErrorResponse(c, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID", nil)
		return
	}

	if err := h.userService.DeleteUser(uint(userID)); err != nil {
		h.logger.WithFields(logrus.Fields{
			"target_user_id": userID,
			"request_id":     middleware.GetRequestID(c),
		}).WithError(err).Error("Failed to delete user")

		if err.Error() == "user not found" {
			middleware.StructuredErrorResponse(c, http.StatusNotFound, "USER_NOT_FOUND", "User not found", nil)
			return
		}

		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "USER_DELETE_FAILED", "Failed to delete user", nil)
		return
	}

	h.logger.WithFields(logrus.Fields{
		"target_user_id": userID,
		"request_id":     middleware.GetRequestID(c),
	}).Info("User deleted successfully")

	c.JSON(http.StatusOK, gin.H{
		"message": "User deleted successfully",
	})
}

