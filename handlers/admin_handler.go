package handlers

import (
	"exam-system/middleware"
	"exam-system/models"
	"exam-system/services"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// SeedData seeds the database with initial data (admin only)
// @Summary Seed database
// @Description Seed the database with initial users, questions, and exams (admin only)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Database seeded successfully"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/admin/seed [post]
func SeedData(c *gin.Context) {
	// This would typically be injected, but for simplicity we'll get it from context
	// In a real application, you'd inject the database connection
	db, exists := c.Get("db")
	if !exists {
		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "DB_NOT_AVAILABLE", "Database connection not available", nil)
		return
	}

	database := db.(*gorm.DB)

	// Seed users
	if err := seedUsers(database); err != nil {
		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "SEED_USERS_FAILED", "Failed to seed users", err.Error())
		return
	}

	// Seed questions
	if err := seedQuestions(database); err != nil {
		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "SEED_QUESTIONS_FAILED", "Failed to seed questions", err.Error())
		return
	}

	// Seed exams
	if err := seedExams(database); err != nil {
		middleware.StructuredErrorResponse(c, http.StatusInternalServerError, "SEED_EXAMS_FAILED", "Failed to seed exams", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Database seeded successfully",
		"seeded": gin.H{
			"users":     "admin and test users created",
			"questions": "sample questions created",
			"exams":     "sample exams created",
		},
	})
}

// GetLogs returns application logs (admin only)
// @Summary Get application logs
// @Description Get recent application logs (admin only)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param lines query int false "Number of log lines to return" default(100)
// @Param level query string false "Log level filter (error, warn, info, debug)"
// @Success 200 {object} map[string]interface{} "Application logs"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/admin/logs [get]
func GetLogs(c *gin.Context) {
	// In a real application, you would read from log files or a logging service
	// For this example, we'll return a mock response
	c.JSON(http.StatusOK, gin.H{
		"message": "Log retrieval not implemented in this demo",
		"note":    "In production, this would read from log files or a centralized logging service",
		"logs":    []string{},
	})
}

func seedUsers(db *gorm.DB) error {
	// Check if admin user already exists
	var adminUser models.User
	if err := db.Where("email = ?", "admin@example.com").First(&adminUser).Error; err == nil {
		return nil // Admin already exists
	}

	// Create admin user
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	admin := models.User{
		Email:     "admin@example.com",
		Username:  "admin",
		Password:  string(hashedPassword),
		FirstName: "System",
		LastName:  "Administrator",
		Role:      models.RoleAdmin,
		IsActive:  true,
	}

	if err := db.Create(&admin).Error; err != nil {
		return err
	}

	// Create test users
	testUsers := []models.User{
		{
			Email:     "john.doe@example.com",
			Username:  "johndoe",
			Password:  string(hashedPassword), // Same password for demo
			FirstName: "John",
			LastName:  "Doe",
			Role:      models.RoleUser,
			IsActive:  true,
		},
		{
			Email:     "jane.smith@example.com",
			Username:  "janesmith",
			Password:  string(hashedPassword),
			FirstName: "Jane",
			LastName:  "Smith",
			Role:      models.RoleUser,
			IsActive:  true,
		},
		{
			Email:     "bob.wilson@example.com",
			Username:  "bobwilson",
			Password:  string(hashedPassword),
			FirstName: "Bob",
			LastName:  "Wilson",
			Role:      models.RoleUser,
			IsActive:  true,
		},
	}

	for _, user := range testUsers {
		var existingUser models.User
		if err := db.Where("email = ?", user.Email).First(&existingUser).Error; err != nil {
			if err := db.Create(&user).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

func seedQuestions(db *gorm.DB) error {
	// Check if questions already exist
	var count int64
	db.Model(&models.Question{}).Count(&count)
	if count > 0 {
		return nil // Questions already exist
	}

	// Get admin user ID
	var admin models.User
	if err := db.Where("role = ?", models.RoleAdmin).First(&admin).Error; err != nil {
		return err
	}

	questions := []models.Question{
		{
			Title:   "What is Go?",
			Content: "Go is a programming language developed by Google. What type of language is it?",
			Type:    models.MultipleChoice,
			Difficulty: models.Easy,
			Options: models.Options{
				{ID: "a", Text: "Interpreted language", IsCorrect: false},
				{ID: "b", Text: "Compiled language", IsCorrect: true},
				{ID: "c", Text: "Scripting language", IsCorrect: false},
				{ID: "d", Text: "Markup language", IsCorrect: false},
			},
			Tags:        models.StringArray{"programming", "go", "basics"},
			Points:      1,
			TimeLimit:   60,
			Explanation: "Go is a compiled programming language developed by Google.",
			IsActive:    true,
			CreatedBy:   admin.ID,
		},
		{
			Title:   "HTTP Status Codes",
			Content: "Which HTTP status code indicates a successful request?",
			Type:    models.MultipleChoice,
			Difficulty: models.Easy,
			Options: models.Options{
				{ID: "a", Text: "404", IsCorrect: false},
				{ID: "b", Text: "500", IsCorrect: false},
				{ID: "c", Text: "200", IsCorrect: true},
				{ID: "d", Text: "301", IsCorrect: false},
			},
			Tags:        models.StringArray{"http", "web", "basics"},
			Points:      1,
			TimeLimit:   45,
			Explanation: "HTTP status code 200 indicates a successful request.",
			IsActive:    true,
			CreatedBy:   admin.ID,
		},
		{
			Title:   "Database Normalization",
			Content: "What is the primary goal of database normalization?",
			Type:    models.MultipleChoice,
			Difficulty: models.Medium,
			Options: models.Options{
				{ID: "a", Text: "Increase data redundancy", IsCorrect: false},
				{ID: "b", Text: "Reduce data redundancy", IsCorrect: true},
				{ID: "c", Text: "Increase storage space", IsCorrect: false},
				{ID: "d", Text: "Decrease query performance", IsCorrect: false},
			},
			Tags:        models.StringArray{"database", "normalization", "design"},
			Points:      2,
			TimeLimit:   90,
			Explanation: "Database normalization aims to reduce data redundancy and improve data integrity.",
			IsActive:    true,
			CreatedBy:   admin.ID,
		},
		{
			Title:   "REST API",
			Content: "Is REST a protocol?",
			Type:    models.TrueFalse,
			Difficulty: models.Medium,
			Options: models.Options{
				{ID: "true", Text: "True", IsCorrect: false},
				{ID: "false", Text: "False", IsCorrect: true},
			},
			Tags:        models.StringArray{"rest", "api", "web"},
			Points:      1,
			TimeLimit:   30,
			Explanation: "REST is an architectural style, not a protocol. HTTP is the protocol commonly used with REST.",
			IsActive:    true,
			CreatedBy:   admin.ID,
		},
		{
			Title:   "Algorithm Complexity",
			Content: "What is the time complexity of binary search?",
			Type:    models.MultipleChoice,
			Difficulty: models.Hard,
			Options: models.Options{
				{ID: "a", Text: "O(n)", IsCorrect: false},
				{ID: "b", Text: "O(log n)", IsCorrect: true},
				{ID: "c", Text: "O(n²)", IsCorrect: false},
				{ID: "d", Text: "O(1)", IsCorrect: false},
			},
			Tags:        models.StringArray{"algorithms", "complexity", "search"},
			Points:      3,
			TimeLimit:   120,
			Explanation: "Binary search has O(log n) time complexity as it divides the search space in half with each iteration.",
			IsActive:    true,
			CreatedBy:   admin.ID,
		},
	}

	for _, question := range questions {
		if err := db.Create(&question).Error; err != nil {
			return err
		}
	}

	return nil
}

func seedExams(db *gorm.DB) error {
	// Check if exams already exist
	var count int64
	db.Model(&models.Exam{}).Count(&count)
	if count > 0 {
		return nil // Exams already exist
	}

	// Get admin user ID
	var admin models.User
	if err := db.Where("role = ?", models.RoleAdmin).First(&admin).Error; err != nil {
		return err
	}

	// Get questions
	var questions []models.Question
	if err := db.Find(&questions).Error; err != nil {
		return err
	}

	if len(questions) == 0 {
		return nil // No questions to create exams with
	}

	// Create sample exams
	now := time.Now()
	startTime := now.Add(time.Hour)
	endTime := now.Add(24 * time.Hour)

	exams := []models.Exam{
		{
			Title:       "Basic Programming Quiz",
			Description: "A basic quiz covering fundamental programming concepts",
			Duration:    30, // 30 minutes
			TotalPoints: 5,
			PassScore:   60,
			Status:      models.ExamActive,
			StartTime:   &startTime,
			EndTime:     &endTime,
			IsActive:    true,
			CreatedBy:   admin.ID,
		},
		{
			Title:       "Web Development Assessment",
			Description: "Assessment covering web development fundamentals",
			Duration:    45, // 45 minutes
			TotalPoints: 8,
			PassScore:   70,
			Status:      models.ExamDraft,
			StartTime:   &startTime,
			EndTime:     &endTime,
			IsActive:    true,
			CreatedBy:   admin.ID,
		},
	}

	for i, exam := range exams {
		if err := db.Create(&exam).Error; err != nil {
			return err
		}

		// Add questions to exam
		questionsToAdd := questions
		if len(questions) > 3 {
			questionsToAdd = questions[:3] // Limit to first 3 questions
		}

		for j, question := range questionsToAdd {
			examQuestion := models.ExamQuestion{
				ExamID:     exam.ID,
				QuestionID: question.ID,
				Order:      j + 1,
				Points:     question.Points,
			}

			if err := db.Create(&examQuestion).Error; err != nil {
				return err
			}
		}

		// Update total points
		var totalPoints int
		db.Model(&models.ExamQuestion{}).Where("exam_id = ?", exam.ID).Select("COALESCE(SUM(points), 0)").Scan(&totalPoints)
		db.Model(&exam).Update("total_points", totalPoints)
	}

	return nil
}

