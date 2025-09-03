package tests

import (
	"exam-system/models"
	"exam-system/services"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupResultTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema
	db.AutoMigrate(&models.User{}, &models.Question{}, &models.Exam{}, &models.ExamQuestion{}, &models.UserExam{}, &models.Result{})

	return db
}

func createTestResult(db *gorm.DB, userID, examID, userExamID uint, score float64, passed bool) models.Result {
	result := models.Result{
		UserID:      userID,
		ExamID:      examID,
		UserExamID:  userExamID,
		Score:       score,
		TotalPoints: int(score * 10 / 100), // Assuming max 10 points
		MaxPoints:   10,
		Passed:      passed,
		Answers: models.Answers{
			{
				QuestionID:      1,
				SelectedOptions: []string{"b"},
				IsCorrect:       true,
				Points:          2,
				TimeSpent:       30,
			},
		},
		StartTime: time.Now().Add(-time.Hour),
		EndTime:   time.Now(),
		Duration:  3600, // 1 hour in seconds
	}
	db.Create(&result)
	return result
}

func TestResultService_GetResults(t *testing.T) {
	db := setupResultTestDB()
	logger := logrus.New()

	resultService := services.NewResultService(db, logger)

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
		TotalPoints: 10,
		PassScore:   70,
		Status:      models.ExamActive,
		IsActive:    true,
		CreatedBy:   admin.ID,
	}
	db.Create(&exam)

	userExam := models.UserExam{
		UserID:      user.ID,
		ExamID:      exam.ID,
		Status:      models.UserExamCompleted,
		MaxAttempts: 1,
	}
	db.Create(&userExam)

	// Create test results
	result1 := createTestResult(db, user.ID, exam.ID, userExam.ID, 85.0, true)
	result2 := createTestResult(db, user.ID, exam.ID, userExam.ID, 65.0, false)

	t.Run("admin gets all results", func(t *testing.T) {
		response, err := resultService.GetResults(1, 10, admin.ID, nil, true)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, int64(2), response.Total)
		assert.Len(t, response.Results, 2)
	})

	t.Run("user gets own results only", func(t *testing.T) {
		response, err := resultService.GetResults(1, 10, user.ID, nil, false)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, int64(2), response.Total)
		assert.Len(t, response.Results, 2)
		
		// Verify all results belong to the user
		for _, result := range response.Results {
			assert.Equal(t, user.ID, result.UserID)
		}
	})

	t.Run("filter results by exam", func(t *testing.T) {
		response, err := resultService.GetResults(1, 10, user.ID, &exam.ID, false)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, int64(2), response.Total)
		assert.Len(t, response.Results, 2)
		
		// Verify all results belong to the specified exam
		for _, result := range response.Results {
			assert.Equal(t, exam.ID, result.ExamID)
		}
	})

	t.Run("pagination works correctly", func(t *testing.T) {
		response, err := resultService.GetResults(1, 1, user.ID, nil, false)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, int64(2), response.Total)
		assert.Len(t, response.Results, 1)
		assert.Equal(t, 2, response.TotalPages)
	})

	// Clean up
	_ = result1
	_ = result2
}

func TestResultService_GetResult(t *testing.T) {
	db := setupResultTestDB()
	logger := logrus.New()

	resultService := services.NewResultService(db, logger)

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
		TotalPoints: 10,
		PassScore:   70,
		Status:      models.ExamActive,
		IsActive:    true,
		CreatedBy:   admin.ID,
	}
	db.Create(&exam)

	userExam := models.UserExam{
		UserID:      user.ID,
		ExamID:      exam.ID,
		Status:      models.UserExamCompleted,
		MaxAttempts: 1,
	}
	db.Create(&userExam)

	result := createTestResult(db, user.ID, exam.ID, userExam.ID, 85.0, true)

	t.Run("admin can get any result", func(t *testing.T) {
		fetchedResult, err := resultService.GetResult(result.ID, admin.ID, true)

		assert.NoError(t, err)
		assert.NotNil(t, fetchedResult)
		assert.Equal(t, result.ID, fetchedResult.ID)
		assert.Equal(t, result.Score, fetchedResult.Score)
		assert.Equal(t, result.Passed, fetchedResult.Passed)
	})

	t.Run("user can get own result", func(t *testing.T) {
		fetchedResult, err := resultService.GetResult(result.ID, user.ID, false)

		assert.NoError(t, err)
		assert.NotNil(t, fetchedResult)
		assert.Equal(t, result.ID, fetchedResult.ID)
		assert.Equal(t, result.UserID, fetchedResult.UserID)
	})

	t.Run("user cannot get other user's result", func(t *testing.T) {
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

		fetchedResult, err := resultService.GetResult(result.ID, otherUser.ID, false)

		assert.Error(t, err)
		assert.Nil(t, fetchedResult)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("non-existent result", func(t *testing.T) {
		fetchedResult, err := resultService.GetResult(999, admin.ID, true)

		assert.Error(t, err)
		assert.Nil(t, fetchedResult)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestResultService_GetStatistics(t *testing.T) {
	db := setupResultTestDB()
	logger := logrus.New()

	resultService := services.NewResultService(db, logger)

	// Create test data
	admin := createTestUser(db, models.RoleAdmin)
	user1 := models.User{
		Email:     "user1@example.com",
		Username:  "user1",
		Password:  "hashedpassword",
		FirstName: "User",
		LastName:  "One",
		Role:      models.RoleUser,
		IsActive:  true,
	}
	db.Create(&user1)

	user2 := models.User{
		Email:     "user2@example.com",
		Username:  "user2",
		Password:  "hashedpassword",
		FirstName: "User",
		LastName:  "Two",
		Role:      models.RoleUser,
		IsActive:  true,
	}
	db.Create(&user2)

	exam1 := models.Exam{
		Title:       "Math Exam",
		Description: "A math exam",
		Duration:    60,
		TotalPoints: 10,
		PassScore:   70,
		Status:      models.ExamActive,
		IsActive:    true,
		CreatedBy:   admin.ID,
	}
	db.Create(&exam1)

	exam2 := models.Exam{
		Title:       "Science Exam",
		Description: "A science exam",
		Duration:    90,
		TotalPoints: 15,
		PassScore:   75,
		Status:      models.ExamActive,
		IsActive:    true,
		CreatedBy:   admin.ID,
	}
	db.Create(&exam2)

	// Create user exams
	userExam1 := models.UserExam{
		UserID:      user1.ID,
		ExamID:      exam1.ID,
		Status:      models.UserExamCompleted,
		MaxAttempts: 1,
	}
	db.Create(&userExam1)

	userExam2 := models.UserExam{
		UserID:      user2.ID,
		ExamID:      exam1.ID,
		Status:      models.UserExamCompleted,
		MaxAttempts: 1,
	}
	db.Create(&userExam2)

	userExam3 := models.UserExam{
		UserID:      user1.ID,
		ExamID:      exam2.ID,
		Status:      models.UserExamCompleted,
		MaxAttempts: 1,
	}
	db.Create(&userExam3)

	// Create test results
	createTestResult(db, user1.ID, exam1.ID, userExam1.ID, 85.0, true)  // Pass
	createTestResult(db, user2.ID, exam1.ID, userExam2.ID, 65.0, false) // Fail
	createTestResult(db, user1.ID, exam2.ID, userExam3.ID, 90.0, true)  // Pass

	t.Run("get comprehensive statistics", func(t *testing.T) {
		stats, err := resultService.GetStatistics()

		assert.NoError(t, err)
		assert.NotNil(t, stats)
		
		// Check overall stats
		assert.Equal(t, 2, stats.OverallStats.TotalExams)
		assert.Equal(t, 2, stats.OverallStats.TotalUsers)
		assert.Equal(t, 3, stats.OverallStats.TotalAttempts)
		assert.Greater(t, stats.OverallStats.AverageScore, 0.0)
		assert.Greater(t, stats.OverallStats.PassRate, 0.0)
		
		// Check that we have exam statistics
		assert.NotEmpty(t, stats.ExamStatistics)
		
		// Check that we have user statistics
		assert.NotEmpty(t, stats.UserStatistics)
		
		// Note: Question statistics might be empty due to the complex JSON query
		// In a real implementation, you might want to simplify this or use a different approach
	})
}

func TestResultService_GetUserResults(t *testing.T) {
	db := setupResultTestDB()
	logger := logrus.New()

	resultService := services.NewResultService(db, logger)

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
		TotalPoints: 10,
		PassScore:   70,
		Status:      models.ExamActive,
		IsActive:    true,
		CreatedBy:   admin.ID,
	}
	db.Create(&exam)

	userExam := models.UserExam{
		UserID:      user.ID,
		ExamID:      exam.ID,
		Status:      models.UserExamCompleted,
		MaxAttempts: 1,
	}
	db.Create(&userExam)

	createTestResult(db, user.ID, exam.ID, userExam.ID, 85.0, true)

	t.Run("get user results", func(t *testing.T) {
		response, err := resultService.GetUserResults(user.ID, 1, 10)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, int64(1), response.Total)
		assert.Len(t, response.Results, 1)
		assert.Equal(t, user.ID, response.Results[0].UserID)
	})
}

func TestResultService_GetExamResults(t *testing.T) {
	db := setupResultTestDB()
	logger := logrus.New()

	resultService := services.NewResultService(db, logger)

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
		TotalPoints: 10,
		PassScore:   70,
		Status:      models.ExamActive,
		IsActive:    true,
		CreatedBy:   admin.ID,
	}
	db.Create(&exam)

	userExam := models.UserExam{
		UserID:      user.ID,
		ExamID:      exam.ID,
		Status:      models.UserExamCompleted,
		MaxAttempts: 1,
	}
	db.Create(&userExam)

	createTestResult(db, user.ID, exam.ID, userExam.ID, 85.0, true)

	t.Run("get exam results", func(t *testing.T) {
		response, err := resultService.GetExamResults(exam.ID, 1, 10)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, int64(1), response.Total)
		assert.Len(t, response.Results, 1)
		assert.Equal(t, exam.ID, response.Results[0].ExamID)
	})
}

