package tests

import (
	"exam-system/config"
	"exam-system/models"
	"exam-system/services"
	"exam-system/utils"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// MockRedisClient is a mock implementation of RedisClient
type MockRedisClient struct {
	mock.Mock
}

func (m *MockRedisClient) Set(key string, value interface{}, expiration time.Duration) error {
	args := m.Called(key, value, expiration)
	return args.Error(0)
}

func (m *MockRedisClient) Get(key string) (string, error) {
	args := m.Called(key)
	return args.String(0), args.Error(1)
}

func (m *MockRedisClient) Del(key string) error {
	args := m.Called(key)
	return args.Error(0)
}

func (m *MockRedisClient) SetJSON(key string, value interface{}, expiration time.Duration) error {
	args := m.Called(key, value, expiration)
	return args.Error(0)
}

func (m *MockRedisClient) GetJSON(key string, dest interface{}) error {
	args := m.Called(key, dest)
	return args.Error(1)
}

func (m *MockRedisClient) IsRateLimited(key string, limit int, window time.Duration) (bool, error) {
	args := m.Called(key, limit, window)
	return args.Bool(0), args.Error(1)
}

func setupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema
	db.AutoMigrate(&models.User{})

	return db
}

func setupTestConfig() {
	config.AppConfig = &config.Config{
		JWT: config.JWTConfig{
			Secret:        "test-secret",
			AccessExpiry:  15 * time.Minute,
			RefreshExpiry: 7 * 24 * time.Hour,
		},
	}
}

func TestAuthService_Register(t *testing.T) {
	setupTestConfig()
	db := setupTestDB()
	mockRedis := &MockRedisClient{}
	logger := logrus.New()

	authService := services.NewAuthService(db, mockRedis, logger)

	t.Run("successful registration", func(t *testing.T) {
		req := services.RegisterRequest{
			Email:     "test@example.com",
			Username:  "testuser",
			Password:  "password123",
			FirstName: "Test",
			LastName:  "User",
		}

		user, err := authService.Register(req)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, req.Email, user.Email)
		assert.Equal(t, req.Username, user.Username)
		assert.Equal(t, req.FirstName, user.FirstName)
		assert.Equal(t, req.LastName, user.LastName)
		assert.Equal(t, models.RoleUser, user.Role)
		assert.True(t, user.IsActive)
		assert.NotEmpty(t, user.Password)
		assert.NotEqual(t, req.Password, user.Password) // Password should be hashed
	})

	t.Run("duplicate email registration", func(t *testing.T) {
		req := services.RegisterRequest{
			Email:     "test@example.com", // Same email as above
			Username:  "testuser2",
			Password:  "password123",
			FirstName: "Test",
			LastName:  "User2",
		}

		user, err := authService.Register(req)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("duplicate username registration", func(t *testing.T) {
		req := services.RegisterRequest{
			Email:     "test2@example.com",
			Username:  "testuser", // Same username as first test
			Password:  "password123",
			FirstName: "Test",
			LastName:  "User2",
		}

		user, err := authService.Register(req)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "already exists")
	})
}

func TestAuthService_Login(t *testing.T) {
	setupTestConfig()
	db := setupTestDB()
	mockRedis := &MockRedisClient{}
	logger := logrus.New()

	authService := services.NewAuthService(db, mockRedis, logger)

	// Create a test user
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	testUser := models.User{
		Email:     "test@example.com",
		Username:  "testuser",
		Password:  string(hashedPassword),
		FirstName: "Test",
		LastName:  "User",
		Role:      models.RoleUser,
		IsActive:  true,
	}
	db.Create(&testUser)

	t.Run("successful login", func(t *testing.T) {
		mockRedis.On("Set", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(nil)

		req := services.LoginRequest{
			Email:    "test@example.com",
			Password: "password123",
		}

		user, tokenResponse, err := authService.Login(req)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.NotNil(t, tokenResponse)
		assert.Equal(t, testUser.Email, user.Email)
		assert.NotEmpty(t, tokenResponse.AccessToken)
		assert.NotEmpty(t, tokenResponse.RefreshToken)
		assert.Equal(t, "Bearer", tokenResponse.TokenType)
		assert.Greater(t, tokenResponse.ExpiresIn, int64(0))

		mockRedis.AssertExpectations(t)
	})

	t.Run("invalid email", func(t *testing.T) {
		req := services.LoginRequest{
			Email:    "nonexistent@example.com",
			Password: "password123",
		}

		user, tokenResponse, err := authService.Login(req)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Nil(t, tokenResponse)
		assert.Contains(t, err.Error(), "invalid credentials")
	})

	t.Run("invalid password", func(t *testing.T) {
		req := services.LoginRequest{
			Email:    "test@example.com",
			Password: "wrongpassword",
		}

		user, tokenResponse, err := authService.Login(req)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Nil(t, tokenResponse)
		assert.Contains(t, err.Error(), "invalid credentials")
	})

	t.Run("inactive user", func(t *testing.T) {
		// Create inactive user
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		inactiveUser := models.User{
			Email:     "inactive@example.com",
			Username:  "inactive",
			Password:  string(hashedPassword),
			FirstName: "Inactive",
			LastName:  "User",
			Role:      models.RoleUser,
			IsActive:  false,
		}
		db.Create(&inactiveUser)

		req := services.LoginRequest{
			Email:    "inactive@example.com",
			Password: "password123",
		}

		user, tokenResponse, err := authService.Login(req)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Nil(t, tokenResponse)
		assert.Contains(t, err.Error(), "invalid credentials")
	})
}

func TestAuthService_ValidateToken(t *testing.T) {
	setupTestConfig()
	db := setupTestDB()
	mockRedis := &MockRedisClient{}
	logger := logrus.New()

	authService := services.NewAuthService(db, mockRedis, logger)

	// Create a test user and generate tokens
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	testUser := models.User{
		ID:        1,
		Email:     "test@example.com",
		Username:  "testuser",
		Password:  string(hashedPassword),
		FirstName: "Test",
		LastName:  "User",
		Role:      models.RoleUser,
		IsActive:  true,
	}
	db.Create(&testUser)

	mockRedis.On("Set", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(nil)

	req := services.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	_, tokenResponse, err := authService.Login(req)
	assert.NoError(t, err)

	t.Run("valid token", func(t *testing.T) {
		claims, err := authService.ValidateToken(tokenResponse.AccessToken)

		assert.NoError(t, err)
		assert.NotNil(t, claims)
		assert.Equal(t, testUser.ID, claims.UserID)
		assert.Equal(t, testUser.Email, claims.Email)
		assert.Equal(t, testUser.Username, claims.Username)
		assert.Equal(t, testUser.Role, claims.Role)
	})

	t.Run("invalid token", func(t *testing.T) {
		claims, err := authService.ValidateToken("invalid.token.here")

		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("empty token", func(t *testing.T) {
		claims, err := authService.ValidateToken("")

		assert.Error(t, err)
		assert.Nil(t, claims)
	})
}

func TestAuthService_Logout(t *testing.T) {
	setupTestConfig()
	db := setupTestDB()
	mockRedis := &MockRedisClient{}
	logger := logrus.New()

	authService := services.NewAuthService(db, mockRedis, logger)

	t.Run("successful logout", func(t *testing.T) {
		userID := uint(1)
		mockRedis.On("Del", "refresh_token:1").Return(nil)

		err := authService.Logout(userID)

		assert.NoError(t, err)
		mockRedis.AssertExpectations(t)
	})

	t.Run("redis error during logout", func(t *testing.T) {
		userID := uint(2)
		mockRedis.On("Del", "refresh_token:2").Return(assert.AnError)

		err := authService.Logout(userID)

		assert.Error(t, err)
		mockRedis.AssertExpectations(t)
	})
}

