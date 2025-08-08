package tests

import (
	"exam-system/config"
	"exam-system/models"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestConfig sets up configuration for testing
func TestConfig() {
	config.AppConfig = &config.Config{
		Server: config.ServerConfig{
			Port:    "8080",
			GinMode: "test",
		},
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     "5432",
			User:     "test",
			Password: "test",
			Name:     "test_db",
			SSLMode:  "disable",
		},
		Redis: config.RedisConfig{
			Host:     "localhost",
			Port:     "6379",
			Password: "",
			DB:       1, // Use different DB for tests
		},
		JWT: config.JWTConfig{
			Secret:        "test-secret-key",
			AccessExpiry:  15 * time.Minute,
			RefreshExpiry: 7 * 24 * time.Hour,
		},
		RateLimit: config.RateLimitConfig{
			LoginLimit:  5,
			SubmitLimit: 10,
			Window:      1 * time.Minute,
		},
		Logging: config.LoggingConfig{
			Level:  "error", // Reduce log noise in tests
			Format: "text",
		},
	}
}

// SetupTestDatabase creates an in-memory SQLite database for testing
func SetupTestDatabase() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: nil, // Disable logging for tests
	})
	if err != nil {
		panic("failed to connect to test database")
	}

	// Run migrations
	err = db.AutoMigrate(
		&models.User{},
		&models.Question{},
		&models.Exam{},
		&models.ExamQuestion{},
		&models.UserExam{},
		&models.Result{},
	)
	if err != nil {
		panic("failed to migrate test database")
	}

	return db
}

// CleanupTestDatabase cleans up the test database
func CleanupTestDatabase(db *gorm.DB) {
	// Drop all tables
	db.Migrator().DropTable(
		&models.Result{},
		&models.UserExam{},
		&models.ExamQuestion{},
		&models.Exam{},
		&models.Question{},
		&models.User{},
	)
}

// CreateTestUser creates a test user for testing purposes
func CreateTestUser(db *gorm.DB, role models.UserRole) *models.User {
	user := &models.User{
		Email:     "test@example.com",
		Username:  "testuser",
		Password:  "hashedpassword",
		FirstName: "Test",
		LastName:  "User",
		Role:      role,
		IsActive:  true,
	}

	if role == models.RoleAdmin {
		user.Email = "admin@example.com"
		user.Username = "admin"
		user.FirstName = "Admin"
		user.LastName = "User"
	}

	db.Create(user)
	return user
}

// CreateTestQuestion creates a test question for testing purposes
func CreateTestQuestion(db *gorm.DB, createdBy uint) *models.Question {
	question := &models.Question{
		Title:   "Test Question",
		Content: "What is the capital of France?",
		Type:    models.MultipleChoice,
		Difficulty: models.Easy,
		Options: models.Options{
			{ID: "a", Text: "London", IsCorrect: false},
			{ID: "b", Text: "Berlin", IsCorrect: false},
			{ID: "c", Text: "Paris", IsCorrect: true},
			{ID: "d", Text: "Madrid", IsCorrect: false},
		},
		Tags:        models.StringArray{"geography", "test"},
		Points:      1,
		TimeLimit:   60,
		Explanation: "Paris is the capital of France.",
		IsActive:    true,
		CreatedBy:   createdBy,
	}

	db.Create(question)
	return question
}

// CreateTestExam creates a test exam for testing purposes
func CreateTestExam(db *gorm.DB, createdBy uint, questions []models.Question) *models.Exam {
	now := time.Now()
	startTime := now.Add(time.Hour)
	endTime := now.Add(24 * time.Hour)

	exam := &models.Exam{
		Title:       "Test Exam",
		Description: "A test exam for testing purposes",
		Duration:    30,
		TotalPoints: len(questions),
		PassScore:   60,
		Status:      models.ExamActive,
		StartTime:   &startTime,
		EndTime:     &endTime,
		IsActive:    true,
		CreatedBy:   createdBy,
	}

	db.Create(exam)

	// Add questions to exam
	for i, question := range questions {
		examQuestion := &models.ExamQuestion{
			ExamID:     exam.ID,
			QuestionID: question.ID,
			Order:      i + 1,
			Points:     question.Points,
		}
		db.Create(examQuestion)
	}

	return exam
}

// CreateTestUserExam creates a test user exam assignment
func CreateTestUserExam(db *gorm.DB, userID, examID uint) *models.UserExam {
	userExam := &models.UserExam{
		UserID:      userID,
		ExamID:      examID,
		Status:      models.UserExamAssigned,
		MaxAttempts: 1,
	}

	db.Create(userExam)
	return userExam
}

