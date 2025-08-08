package services

import (
	"exam-system/config"
	"exam-system/models"
	"exam-system/utils"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	db          *gorm.DB
	redisClient *utils.RedisClient
	logger      *logrus.Logger
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type RegisterRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Username  string `json:"username" binding:"required,min=3,max=50"`
	Password  string `json:"password" binding:"required,min=6"`
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type Claims struct {
	UserID   uint             `json:"user_id"`
	Email    string           `json:"email"`
	Username string           `json:"username"`
	Role     models.UserRole  `json:"role"`
	jwt.RegisteredClaims
}

func NewAuthService(db *gorm.DB, redisClient *utils.RedisClient, logger *logrus.Logger) *AuthService {
	return &AuthService{
		db:          db,
		redisClient: redisClient,
		logger:      logger,
	}
}

func (s *AuthService) Register(req RegisterRequest) (*models.User, error) {
	// Check if user already exists
	var existingUser models.User
	if err := s.db.Where("email = ? OR username = ?", req.Email, req.Username).First(&existingUser).Error; err == nil {
		return nil, fmt.Errorf("user with email or username already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.WithError(err).Error("Failed to hash password")
		return nil, fmt.Errorf("failed to hash password")
	}

	// Create user
	user := models.User{
		Email:     req.Email,
		Username:  req.Username,
		Password:  string(hashedPassword),
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Role:      models.RoleUser,
		IsActive:  true,
	}

	if err := s.db.Create(&user).Error; err != nil {
		s.logger.WithError(err).Error("Failed to create user")
		return nil, fmt.Errorf("failed to create user")
	}

	s.logger.WithFields(logrus.Fields{
		"user_id": user.ID,
		"email":   user.Email,
	}).Info("User registered successfully")

	return &user, nil
}

func (s *AuthService) Login(req LoginRequest) (*models.User, *TokenResponse, error) {
	// Find user by email
	var user models.User
	if err := s.db.Where("email = ? AND is_active = ?", req.Email, true).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil, fmt.Errorf("invalid credentials")
		}
		s.logger.WithError(err).Error("Failed to find user")
		return nil, nil, fmt.Errorf("failed to authenticate user")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, nil, fmt.Errorf("invalid credentials")
	}

	// Generate tokens
	tokenResponse, err := s.generateTokens(&user)
	if err != nil {
		return nil, nil, err
	}

	s.logger.WithFields(logrus.Fields{
		"user_id": user.ID,
		"email":   user.Email,
	}).Info("User logged in successfully")

	return &user, tokenResponse, nil
}

func (s *AuthService) RefreshToken(req RefreshTokenRequest) (*TokenResponse, error) {
	// Parse refresh token
	token, err := jwt.ParseWithClaims(req.RefreshToken, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.AppConfig.JWT.Secret), nil
	})

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid refresh token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Check if refresh token exists in Redis
	key := fmt.Sprintf("refresh_token:%d", claims.UserID)
	storedToken, err := s.redisClient.Get(key)
	if err != nil || storedToken != req.RefreshToken {
		return nil, fmt.Errorf("refresh token not found or expired")
	}

	// Get user from database
	var user models.User
	if err := s.db.Where("id = ? AND is_active = ?", claims.UserID, true).First(&user).Error; err != nil {
		return nil, fmt.Errorf("user not found or inactive")
	}

	// Generate new tokens
	tokenResponse, err := s.generateTokens(&user)
	if err != nil {
		return nil, err
	}

	s.logger.WithFields(logrus.Fields{
		"user_id": user.ID,
		"email":   user.Email,
	}).Info("Token refreshed successfully")

	return tokenResponse, nil
}

func (s *AuthService) Logout(userID uint) error {
	// Remove refresh token from Redis
	key := fmt.Sprintf("refresh_token:%d", userID)
	if err := s.redisClient.Del(key); err != nil {
		s.logger.WithError(err).Error("Failed to remove refresh token from Redis")
		return fmt.Errorf("failed to logout")
	}

	s.logger.WithField("user_id", userID).Info("User logged out successfully")
	return nil
}

func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(config.AppConfig.JWT.Secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func (s *AuthService) generateTokens(user *models.User) (*TokenResponse, error) {
	now := time.Now()
	accessExpiry := now.Add(config.AppConfig.JWT.AccessExpiry)
	refreshExpiry := now.Add(config.AppConfig.JWT.RefreshExpiry)

	// Create access token claims
	accessClaims := Claims{
		UserID:   user.ID,
		Email:    user.Email,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Subject:   fmt.Sprintf("%d", user.ID),
		},
	}

	// Create refresh token claims
	refreshClaims := Claims{
		UserID:   user.ID,
		Email:    user.Email,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExpiry),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Subject:   fmt.Sprintf("%d", user.ID),
		},
	}

	// Generate access token
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(config.AppConfig.JWT.Secret))
	if err != nil {
		s.logger.WithError(err).Error("Failed to sign access token")
		return nil, fmt.Errorf("failed to generate access token")
	}

	// Generate refresh token
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(config.AppConfig.JWT.Secret))
	if err != nil {
		s.logger.WithError(err).Error("Failed to sign refresh token")
		return nil, fmt.Errorf("failed to generate refresh token")
	}

	// Store refresh token in Redis
	key := fmt.Sprintf("refresh_token:%d", user.ID)
	if err := s.redisClient.Set(key, refreshTokenString, config.AppConfig.JWT.RefreshExpiry); err != nil {
		s.logger.WithError(err).Error("Failed to store refresh token in Redis")
		return nil, fmt.Errorf("failed to store refresh token")
	}

	return &TokenResponse{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresIn:    int64(config.AppConfig.JWT.AccessExpiry.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

