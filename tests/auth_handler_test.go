package tests

import (
	"bytes"
	"encoding/json"
	"exam-system/handlers"
	"exam-system/models"
	"exam-system/services"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAuthService is a mock implementation of AuthService
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Register(req services.RegisterRequest) (*models.User, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthService) Login(req services.LoginRequest) (*models.User, *services.TokenResponse, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).(*models.User), args.Get(1).(*services.TokenResponse), args.Error(2)
}

func (m *MockAuthService) RefreshToken(req services.RefreshTokenRequest) (*services.TokenResponse, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.TokenResponse), args.Error(1)
}

func (m *MockAuthService) Logout(userID uint) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockAuthService) ValidateToken(tokenString string) (*services.Claims, error) {
	args := m.Called(tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.Claims), args.Error(1)
}

func setupGinTest() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

func TestAuthHandler_Register(t *testing.T) {
	router := setupGinTest()
	mockAuthService := &MockAuthService{}
	logger := logrus.New()
	authHandler := handlers.NewAuthHandler(mockAuthService, logger)

	router.POST("/register", authHandler.Register)

	t.Run("successful registration", func(t *testing.T) {
		req := services.RegisterRequest{
			Email:     "test@example.com",
			Username:  "testuser",
			Password:  "password123",
			FirstName: "Test",
			LastName:  "User",
		}

		expectedUser := &models.User{
			ID:        1,
			Email:     req.Email,
			Username:  req.Username,
			FirstName: req.FirstName,
			LastName:  req.LastName,
			Role:      models.RoleUser,
			IsActive:  true,
		}

		mockAuthService.On("Register", req).Return(expectedUser, nil)

		reqBody, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(reqBody))
		httpReq.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "User registered successfully", response["message"])
		assert.NotNil(t, response["user"])

		mockAuthService.AssertExpectations(t)
	})

	t.Run("invalid request data", func(t *testing.T) {
		invalidReq := map[string]interface{}{
			"email": "invalid-email", // Invalid email format
		}

		reqBody, _ := json.Marshal(invalidReq)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(reqBody))
		httpReq.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "INVALID_REQUEST", response["code"])
	})

	t.Run("user already exists", func(t *testing.T) {
		req := services.RegisterRequest{
			Email:     "existing@example.com",
			Username:  "existing",
			Password:  "password123",
			FirstName: "Existing",
			LastName:  "User",
		}

		mockAuthService.On("Register", req).Return(nil, assert.AnError)

		reqBody, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(reqBody))
		httpReq.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		mockAuthService.AssertExpectations(t)
	})
}

func TestAuthHandler_Login(t *testing.T) {
	router := setupGinTest()
	mockAuthService := &MockAuthService{}
	logger := logrus.New()
	authHandler := handlers.NewAuthHandler(mockAuthService, logger)

	router.POST("/login", authHandler.Login)

	t.Run("successful login", func(t *testing.T) {
		req := services.LoginRequest{
			Email:    "test@example.com",
			Password: "password123",
		}

		expectedUser := &models.User{
			ID:       1,
			Email:    req.Email,
			Username: "testuser",
			Role:     models.RoleUser,
			IsActive: true,
		}

		expectedTokens := &services.TokenResponse{
			AccessToken:  "access.token.here",
			RefreshToken: "refresh.token.here",
			ExpiresIn:    900,
			TokenType:    "Bearer",
		}

		mockAuthService.On("Login", req).Return(expectedUser, expectedTokens, nil)

		reqBody, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(reqBody))
		httpReq.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Login successful", response["message"])
		assert.NotNil(t, response["user"])
		assert.NotNil(t, response["tokens"])

		mockAuthService.AssertExpectations(t)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		req := services.LoginRequest{
			Email:    "test@example.com",
			Password: "wrongpassword",
		}

		mockAuthService.On("Login", req).Return(nil, nil, assert.AnError)

		reqBody, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(reqBody))
		httpReq.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		mockAuthService.AssertExpectations(t)
	})

	t.Run("invalid request format", func(t *testing.T) {
		invalidReq := map[string]interface{}{
			"email": "", // Empty email
		}

		reqBody, _ := json.Marshal(invalidReq)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(reqBody))
		httpReq.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "INVALID_REQUEST", response["code"])
	})
}

func TestAuthHandler_RefreshToken(t *testing.T) {
	router := setupGinTest()
	mockAuthService := &MockAuthService{}
	logger := logrus.New()
	authHandler := handlers.NewAuthHandler(mockAuthService, logger)

	router.POST("/refresh", authHandler.RefreshToken)

	t.Run("successful token refresh", func(t *testing.T) {
		req := services.RefreshTokenRequest{
			RefreshToken: "valid.refresh.token",
		}

		expectedTokens := &services.TokenResponse{
			AccessToken:  "new.access.token",
			RefreshToken: "new.refresh.token",
			ExpiresIn:    900,
			TokenType:    "Bearer",
		}

		mockAuthService.On("RefreshToken", req).Return(expectedTokens, nil)

		reqBody, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/refresh", bytes.NewBuffer(reqBody))
		httpReq.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Token refreshed successfully", response["message"])
		assert.NotNil(t, response["tokens"])

		mockAuthService.AssertExpectations(t)
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		req := services.RefreshTokenRequest{
			RefreshToken: "invalid.refresh.token",
		}

		mockAuthService.On("RefreshToken", req).Return(nil, assert.AnError)

		reqBody, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/refresh", bytes.NewBuffer(reqBody))
		httpReq.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "INVALID_REFRESH_TOKEN", response["code"])

		mockAuthService.AssertExpectations(t)
	})

	t.Run("missing refresh token", func(t *testing.T) {
		invalidReq := map[string]interface{}{
			"refresh_token": "", // Empty refresh token
		}

		reqBody, _ := json.Marshal(invalidReq)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/refresh", bytes.NewBuffer(reqBody))
		httpReq.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "INVALID_REQUEST", response["code"])
	})
}

