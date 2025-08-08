package services

import (
	"exam-system/models"
	"fmt"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	db     *gorm.DB
	logger *logrus.Logger
}

type UpdateProfileRequest struct {
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
	Username  string `json:"username" binding:"required,min=3,max=50"`
}

type UpdateUserRequest struct {
	FirstName string          `json:"first_name" binding:"required"`
	LastName  string          `json:"last_name" binding:"required"`
	Username  string          `json:"username" binding:"required,min=3,max=50"`
	Email     string          `json:"email" binding:"required,email"`
	Role      models.UserRole `json:"role" binding:"required"`
	IsActive  bool            `json:"is_active"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

type UserListResponse struct {
	Users      []models.UserResponse `json:"users"`
	Total      int64                 `json:"total"`
	Page       int                   `json:"page"`
	PageSize   int                   `json:"page_size"`
	TotalPages int                   `json:"total_pages"`
}

func NewUserService(db *gorm.DB, logger *logrus.Logger) *UserService {
	return &UserService{
		db:     db,
		logger: logger,
	}
}

func (s *UserService) GetProfile(userID uint) (*models.User, error) {
	var user models.User
	if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		s.logger.WithError(err).Error("Failed to get user profile")
		return nil, fmt.Errorf("failed to get user profile")
	}

	return &user, nil
}

func (s *UserService) UpdateProfile(userID uint, req UpdateProfileRequest) (*models.User, error) {
	var user models.User
	if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		s.logger.WithError(err).Error("Failed to find user")
		return nil, fmt.Errorf("failed to update profile")
	}

	// Check if username is already taken by another user
	var existingUser models.User
	if err := s.db.Where("username = ? AND id != ?", req.Username, userID).First(&existingUser).Error; err == nil {
		return nil, fmt.Errorf("username already taken")
	}

	// Update user fields
	user.FirstName = req.FirstName
	user.LastName = req.LastName
	user.Username = req.Username

	if err := s.db.Save(&user).Error; err != nil {
		s.logger.WithError(err).Error("Failed to update user profile")
		return nil, fmt.Errorf("failed to update profile")
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":  user.ID,
		"username": user.Username,
	}).Info("User profile updated successfully")

	return &user, nil
}

func (s *UserService) ChangePassword(userID uint, req ChangePasswordRequest) error {
	var user models.User
	if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("user not found")
		}
		s.logger.WithError(err).Error("Failed to find user")
		return fmt.Errorf("failed to change password")
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword)); err != nil {
		return fmt.Errorf("current password is incorrect")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		s.logger.WithError(err).Error("Failed to hash new password")
		return fmt.Errorf("failed to change password")
	}

	// Update password
	user.Password = string(hashedPassword)
	if err := s.db.Save(&user).Error; err != nil {
		s.logger.WithError(err).Error("Failed to update password")
		return fmt.Errorf("failed to change password")
	}

	s.logger.WithField("user_id", user.ID).Info("Password changed successfully")
	return nil
}

func (s *UserService) GetUsers(page, pageSize int, search string) (*UserListResponse, error) {
	var users []models.User
	var total int64

	query := s.db.Model(&models.User{})

	// Add search filter if provided
	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where("email ILIKE ? OR username ILIKE ? OR first_name ILIKE ? OR last_name ILIKE ?",
			searchPattern, searchPattern, searchPattern, searchPattern)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		s.logger.WithError(err).Error("Failed to count users")
		return nil, fmt.Errorf("failed to get users")
	}

	// Get paginated results
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&users).Error; err != nil {
		s.logger.WithError(err).Error("Failed to get users")
		return nil, fmt.Errorf("failed to get users")
	}

	// Convert to response format
	userResponses := make([]models.UserResponse, len(users))
	for i, user := range users {
		userResponses[i] = user.ToResponse()
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return &UserListResponse{
		Users:      userResponses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *UserService) GetUser(userID uint) (*models.User, error) {
	var user models.User
	if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		s.logger.WithError(err).Error("Failed to get user")
		return nil, fmt.Errorf("failed to get user")
	}

	return &user, nil
}

func (s *UserService) UpdateUser(userID uint, req UpdateUserRequest) (*models.User, error) {
	var user models.User
	if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		s.logger.WithError(err).Error("Failed to find user")
		return nil, fmt.Errorf("failed to update user")
	}

	// Check if email is already taken by another user
	var existingUser models.User
	if err := s.db.Where("email = ? AND id != ?", req.Email, userID).First(&existingUser).Error; err == nil {
		return nil, fmt.Errorf("email already taken")
	}

	// Check if username is already taken by another user
	if err := s.db.Where("username = ? AND id != ?", req.Username, userID).First(&existingUser).Error; err == nil {
		return nil, fmt.Errorf("username already taken")
	}

	// Update user fields
	user.FirstName = req.FirstName
	user.LastName = req.LastName
	user.Username = req.Username
	user.Email = req.Email
	user.Role = req.Role
	user.IsActive = req.IsActive

	if err := s.db.Save(&user).Error; err != nil {
		s.logger.WithError(err).Error("Failed to update user")
		return nil, fmt.Errorf("failed to update user")
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":  user.ID,
		"email":    user.Email,
		"username": user.Username,
		"role":     user.Role,
	}).Info("User updated successfully")

	return &user, nil
}

func (s *UserService) DeleteUser(userID uint) error {
	var user models.User
	if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("user not found")
		}
		s.logger.WithError(err).Error("Failed to find user")
		return fmt.Errorf("failed to delete user")
	}

	// Soft delete the user
	if err := s.db.Delete(&user).Error; err != nil {
		s.logger.WithError(err).Error("Failed to delete user")
		return fmt.Errorf("failed to delete user")
	}

	s.logger.WithFields(logrus.Fields{
		"user_id": user.ID,
		"email":   user.Email,
	}).Info("User deleted successfully")

	return nil
}

func (s *UserService) ActivateUser(userID uint) error {
	if err := s.db.Model(&models.User{}).Where("id = ?", userID).Update("is_active", true).Error; err != nil {
		s.logger.WithError(err).Error("Failed to activate user")
		return fmt.Errorf("failed to activate user")
	}

	s.logger.WithField("user_id", userID).Info("User activated successfully")
	return nil
}

func (s *UserService) DeactivateUser(userID uint) error {
	if err := s.db.Model(&models.User{}).Where("id = ?", userID).Update("is_active", false).Error; err != nil {
		s.logger.WithError(err).Error("Failed to deactivate user")
		return fmt.Errorf("failed to deactivate user")
	}

	s.logger.WithField("user_id", userID).Info("User deactivated successfully")
	return nil
}

