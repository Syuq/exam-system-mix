package tests

import (
	"exam-system/config"
	"exam-system/models"
	"exam-system/services"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupExamTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema
	db.AutoMigrate(&models.User{}, &models.Question{}, &models.Exam{}, &models.ExamQuestion{}, &models.UserExam{}, &models.Result{})

	return db
}

func createTestUser(db *gorm.DB, role models.UserRole) models.User {
	user := models.User{
		Email:     "test@example.com",
		Username:  "testuser",
		Password:  "hashedpassword",
		FirstName: "Test",
		LastName:  "User",
		Role:      role,
		IsActive:  true,
	}
	db.Create(&user)
	return user
}

func createTestQuestion(db *gorm.DB, createdBy uint) models.Question {
	question := models.Question{
		Title:      "Test Question",
		Content:    "What is 2 + 2?",
		Type:       models.MultipleChoice,
		Difficulty: models.Easy,
		Options: models.Options{
			{ID: "a", Text: "3", IsCorrect: false},
			{ID: "b", Text: "4", IsCorrect: true},
			{ID: "c", Text: "5", IsCorrect: false},
		},
		Tags:        models.StringArray{"math", "basic"},
		Points:      1,
		TimeLimit:   60,
		Explanation: "2 + 2 = 4",
		IsActive:    true,
		CreatedBy:   createdBy,
	}
	db.Create(&question)
	return question
}

func TestExamService_CreateExam(t *testing.T) {
	setupTestConfig()
	db := setupExamTestDB()
	mockRedis := &MockRedisClient{}
	logger := logrus.New()

	examService := services.NewExamService(db, mockRedis, logger)

	// Create test user and question
	admin := createTestUser(db, models.RoleAdmin)
	question := createTestQuestion(db, admin.ID)

	t.Run("successful exam creation", func(t *testing.T) {
		req := services.CreateExamRequest{
			Title:       "Test Exam",
			Description: "A test exam",
			Duration:    60,
			PassScore:   70,
			StartTime:   time.Now().Add(time.Hour),
			EndTime:     time.Now().Add(24 * time.Hour),
			Questions: []services.ExamQuestionRequest{
				{
					QuestionID: question.ID,
					Points:     2,
					Order:      1,
				},
			},
		}

		exam, err := examService.CreateExam(req, admin.ID)

		assert.NoError(t, err)
		assert.NotNil(t, exam)
		assert.Equal(t, req.Title, exam.Title)
		assert.Equal(t, req.Description, exam.Description)
		assert.Equal(t, req.Duration, exam.Duration)
		assert.Equal(t, req.PassScore, exam.PassScore)
		assert.Equal(t, 2, exam.TotalPoints) // Points from the question
		assert.Equal(t, models.ExamDraft, exam.Status)
		assert.Equal(t, admin.ID, exam.CreatedBy)
	})

	t.Run("exam creation with invalid question", func(t *testing.T) {
		req := services.CreateExamRequest{
			Title:       "Test Exam 2",
			Description: "Another test exam",
			Duration:    60,
			PassScore:   70,
			StartTime:   time.Now().Add(time.Hour),
			EndTime:     time.Now().Add(24 * time.Hour),
			Questions: []services.ExamQuestionRequest{
				{
					QuestionID: 999, // Non-existent question
					Points:     2,
					Order:      1,
				},
			},
		}

		exam, err := examService.CreateExam(req, admin.ID)

		assert.Error(t, err)
		assert.Nil(t, exam)
		assert.Contains(t, err.Error(), "invalid or inactive")
	})
}

func TestExamService_GetExams(t *testing.T) {
	setupTestConfig()
	db := setupExamTestDB()
	mockRedis := &MockRedisClient{}
	logger := logrus.New()

	examService := services.NewExamService(db, mockRedis, logger)

	// Create test data
	admin := createTestUser(db, models.RoleAdmin)
	user := models.User{
		Email:     "user@example.com",
		Username:  "regularuser",
		Password:  "hashedpassword",
		FirstName: "Regular",
		LastName:  "User",
		Role:      models.RoleUser,
		IsActive:  true,
	}
	db.Create(&user)

	question := createTestQuestion(db, admin.ID)

	// Create test exam
	exam := models.Exam{
		Title:       "Test Exam",
		Description: "A test exam",
		Duration:    60,
		TotalPoints: 2,
		PassScore:   70,
		Status:      models.ExamActive,
		IsActive:    true,
		CreatedBy:   admin.ID,
	}
	db.Create(&exam)

	// Create exam question
	examQuestion := models.ExamQuestion{
		ExamID:     exam.ID,
		QuestionID: question.ID,
		Order:      1,
		Points:     2,
	}
	db.Create(&examQuestion)

	// Create user exam assignment
	userExam := models.UserExam{
		UserID:      user.ID,
		ExamID:      exam.ID,
		Status:      models.UserExamAssigned,
		MaxAttempts: 1,
	}
	db.Create(&userExam)

	t.Run("admin gets all exams", func(t *testing.T) {
		response, err := examService.GetExams(1, 10, admin.ID, true)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, int64(1), response.Total)
		assert.Len(t, response.Exams, 1)
		assert.Equal(t, exam.Title, response.Exams[0].Title)
	})

	t.Run("regular user gets assigned exams only", func(t *testing.T) {
		response, err := examService.GetExams(1, 10, user.ID, false)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, int64(1), response.Total)
		assert.Len(t, response.Exams, 1)
		assert.Equal(t, exam.Title, response.Exams[0].Title)
	})
}

func TestExamService_StartExam(t *testing.T) {
	setupTestConfig()
	db := setupExamTestDB()
	mockRedis := &MockRedisClient{}
	logger := logrus.New()

	examService := services.NewExamService(db, mockRedis, logger)

	// Create test data
	admin := createTestUser(db, models.RoleAdmin)
	user := models.User{
		Email:     "user@example.com",
		Username:  "regularuser",
		Password:  "hashedpassword",
		FirstName: "Regular",
		LastName:  "User",
		Role:      models.RoleUser,
		IsActive:  true,
	}
	db.Create(&user)

	question := createTestQuestion(db, admin.ID)

	exam := models.Exam{
		Title:       "Test Exam",
		Description: "A test exam",
		Duration:    60,
		TotalPoints: 2,
		PassScore:   70,
		Status:      models.ExamActive,
		IsActive:    true,
		CreatedBy:   admin.ID,
	}
	db.Create(&exam)

	examQuestion := models.ExamQuestion{
		ExamID:     exam.ID,
		QuestionID: question.ID,
		Order:      1,
		Points:     2,
	}
	db.Create(&examQuestion)

	userExam := models.UserExam{
		UserID:      user.ID,
		ExamID:      exam.ID,
		Status:      models.UserExamAssigned,
		MaxAttempts: 1,
	}
	db.Create(&userExam)

	t.Run("successful exam start", func(t *testing.T) {
		mockRedis.On("SetJSON", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(nil)

		response, err := examService.StartExam(exam.ID, user.ID)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, models.UserExamStarted, response.UserExam.Status)
		assert.NotNil(t, response.UserExam.StartedAt)
		assert.Equal(t, 1, response.UserExam.AttemptCount)
		assert.Len(t, response.Questions, 1)
		assert.Equal(t, question.Title, response.Questions[0].Title)
		assert.Greater(t, response.TimeLeft, 0)

		mockRedis.AssertExpectations(t)
	})

	t.Run("exam not assigned to user", func(t *testing.T) {
		otherUser := models.User{
			Email:     "other@example.com",
			Username:  "otheruser",
			Password:  "hashedpassword",
			FirstName: "Other",
			LastName:  "User",
			Role:      models.RoleUser,
			IsActive:  true,
		}
		db.Create(&otherUser)

		response, err := examService.StartExam(exam.ID, otherUser.ID)

		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "not assigned")
	})
}

func TestExamService_AssignExam(t *testing.T) {
	setupTestConfig()
	db := setupExamTestDB()
	mockRedis := &MockRedisClient{}
	logger := logrus.New()

	examService := services.NewExamService(db, mockRedis, logger)

	// Create test data
	admin := createTestUser(db, models.RoleAdmin)
	user := models.User{
		Email:     "user@example.com",
		Username:  "regularuser",
		Password:  "hashedpassword",
		FirstName: "Regular",
		LastName:  "User",
		Role:      models.RoleUser,
		IsActive:  true,
	}
	db.Create(&user)

	exam := models.Exam{
		Title:       "Test Exam",
		Description: "A test exam",
		Duration:    60,
		TotalPoints: 2,
		PassScore:   70,
		Status:      models.ExamActive,
		IsActive:    true,
		CreatedBy:   admin.ID,
	}
	db.Create(&exam)

	t.Run("successful exam assignment", func(t *testing.T) {
		expiresAt := time.Now().Add(24 * time.Hour)
		req := services.AssignExamRequest{
			UserIDs:     []uint{user.ID},
			ExpiresAt:   &expiresAt,
			MaxAttempts: 2,
		}

		err := examService.AssignExam(exam.ID, req)

		assert.NoError(t, err)

		// Verify assignment was created
		var userExam models.UserExam
		err = db.Where("user_id = ? AND exam_id = ?", user.ID, exam.ID).First(&userExam).Error
		assert.NoError(t, err)
		assert.Equal(t, models.UserExamAssigned, userExam.Status)
		assert.Equal(t, 2, userExam.MaxAttempts)
	})

	t.Run("assign exam to non-existent user", func(t *testing.T) {
		req := services.AssignExamRequest{
			UserIDs:     []uint{999}, // Non-existent user
			MaxAttempts: 1,
		}

		err := examService.AssignExam(exam.ID, req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid or inactive")
	})

	t.Run("assign non-existent exam", func(t *testing.T) {
		req := services.AssignExamRequest{
			UserIDs:     []uint{user.ID},
			MaxAttempts: 1,
		}

		err := examService.AssignExam(999, req) // Non-existent exam

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

