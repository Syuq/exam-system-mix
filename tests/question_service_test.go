package tests

import (
	"exam-system/models"
	"exam-system/services"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupQuestionTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema
	db.AutoMigrate(&models.User{}, &models.Question{}, &models.Exam{}, &models.ExamQuestion{})

	return db
}

func TestQuestionService_CreateQuestion(t *testing.T) {
	db := setupQuestionTestDB()
	logger := logrus.New()
	questionService := services.NewQuestionService(db, logger)

	// Create a test user
	testUser := models.User{
		ID:       1,
		Email:    "admin@example.com",
		Username: "admin",
		Role:     models.RoleAdmin,
		IsActive: true,
	}
	db.Create(&testUser)

	t.Run("successful question creation", func(t *testing.T) {
		req := services.CreateQuestionRequest{
			Title:   "Test Question",
			Content: "What is the capital of France?",
			Type:    models.MultipleChoice,
			Difficulty: models.Easy,
			Options: []models.Option{
				{ID: "a", Text: "London", IsCorrect: false},
				{ID: "b", Text: "Berlin", IsCorrect: false},
				{ID: "c", Text: "Paris", IsCorrect: true},
				{ID: "d", Text: "Madrid", IsCorrect: false},
			},
			Tags:        []string{"geography", "capitals"},
			Points:      1,
			TimeLimit:   60,
			Explanation: "Paris is the capital of France.",
		}

		question, err := questionService.CreateQuestion(req, testUser.ID)

		assert.NoError(t, err)
		assert.NotNil(t, question)
		assert.Equal(t, req.Title, question.Title)
		assert.Equal(t, req.Content, question.Content)
		assert.Equal(t, req.Type, question.Type)
		assert.Equal(t, req.Difficulty, question.Difficulty)
		assert.Equal(t, len(req.Options), len(question.Options))
		assert.Equal(t, req.Tags, []string(question.Tags))
		assert.Equal(t, req.Points, question.Points)
		assert.Equal(t, req.TimeLimit, question.TimeLimit)
		assert.Equal(t, req.Explanation, question.Explanation)
		assert.True(t, question.IsActive)
		assert.Equal(t, testUser.ID, question.CreatedBy)
	})

	t.Run("invalid options - no correct answer", func(t *testing.T) {
		req := services.CreateQuestionRequest{
			Title:   "Invalid Question",
			Content: "Test question with no correct answer",
			Type:    models.MultipleChoice,
			Difficulty: models.Easy,
			Options: []models.Option{
				{ID: "a", Text: "Option A", IsCorrect: false},
				{ID: "b", Text: "Option B", IsCorrect: false},
			},
			Tags:      []string{"test"},
			Points:    1,
			TimeLimit: 60,
		}

		question, err := questionService.CreateQuestion(req, testUser.ID)

		assert.Error(t, err)
		assert.Nil(t, question)
		assert.Contains(t, err.Error(), "at least one correct answer")
	})

	t.Run("invalid options - insufficient options", func(t *testing.T) {
		req := services.CreateQuestionRequest{
			Title:   "Invalid Question",
			Content: "Test question with only one option",
			Type:    models.MultipleChoice,
			Difficulty: models.Easy,
			Options: []models.Option{
				{ID: "a", Text: "Only Option", IsCorrect: true},
			},
			Tags:      []string{"test"},
			Points:    1,
			TimeLimit: 60,
		}

		question, err := questionService.CreateQuestion(req, testUser.ID)

		assert.Error(t, err)
		assert.Nil(t, question)
		assert.Contains(t, err.Error(), "at least 2 options")
	})

	t.Run("true/false question validation", func(t *testing.T) {
		req := services.CreateQuestionRequest{
			Title:   "True/False Question",
			Content: "Go is a compiled language",
			Type:    models.TrueFalse,
			Difficulty: models.Easy,
			Options: []models.Option{
				{ID: "true", Text: "True", IsCorrect: true},
				{ID: "false", Text: "False", IsCorrect: false},
			},
			Tags:      []string{"programming", "go"},
			Points:    1,
			TimeLimit: 30,
		}

		question, err := questionService.CreateQuestion(req, testUser.ID)

		assert.NoError(t, err)
		assert.NotNil(t, question)
		assert.Equal(t, models.TrueFalse, question.Type)
		assert.Equal(t, 2, len(question.Options))
	})

	t.Run("invalid true/false question - too many options", func(t *testing.T) {
		req := services.CreateQuestionRequest{
			Title:   "Invalid True/False Question",
			Content: "Test question",
			Type:    models.TrueFalse,
			Difficulty: models.Easy,
			Options: []models.Option{
				{ID: "true", Text: "True", IsCorrect: true},
				{ID: "false", Text: "False", IsCorrect: false},
				{ID: "maybe", Text: "Maybe", IsCorrect: false},
			},
			Tags:      []string{"test"},
			Points:    1,
			TimeLimit: 30,
		}

		question, err := questionService.CreateQuestion(req, testUser.ID)

		assert.Error(t, err)
		assert.Nil(t, question)
		assert.Contains(t, err.Error(), "exactly 2 options")
	})
}

func TestQuestionService_GetQuestions(t *testing.T) {
	db := setupQuestionTestDB()
	logger := logrus.New()
	questionService := services.NewQuestionService(db, logger)

	// Create test user
	testUser := models.User{
		ID:       1,
		Email:    "admin@example.com",
		Username: "admin",
		Role:     models.RoleAdmin,
		IsActive: true,
	}
	db.Create(&testUser)

	// Create test questions
	questions := []models.Question{
		{
			Title:      "Geography Question",
			Content:    "What is the capital of France?",
			Type:       models.MultipleChoice,
			Difficulty: models.Easy,
			Options: models.Options{
				{ID: "a", Text: "London", IsCorrect: false},
				{ID: "b", Text: "Paris", IsCorrect: true},
			},
			Tags:      models.StringArray{"geography", "capitals"},
			Points:    1,
			TimeLimit: 60,
			IsActive:  true,
			CreatedBy: testUser.ID,
		},
		{
			Title:      "Programming Question",
			Content:    "What is Go?",
			Type:       models.MultipleChoice,
			Difficulty: models.Medium,
			Options: models.Options{
				{ID: "a", Text: "Compiled language", IsCorrect: true},
				{ID: "b", Text: "Interpreted language", IsCorrect: false},
			},
			Tags:      models.StringArray{"programming", "go"},
			Points:    2,
			TimeLimit: 90,
			IsActive:  true,
			CreatedBy: testUser.ID,
		},
		{
			Title:      "Inactive Question",
			Content:    "This question is inactive",
			Type:       models.MultipleChoice,
			Difficulty: models.Hard,
			Options: models.Options{
				{ID: "a", Text: "Option A", IsCorrect: true},
				{ID: "b", Text: "Option B", IsCorrect: false},
			},
			Tags:      models.StringArray{"test"},
			Points:    3,
			TimeLimit: 120,
			IsActive:  false,
			CreatedBy: testUser.ID,
		},
	}

	for _, question := range questions {
		db.Create(&question)
	}

	t.Run("get all questions", func(t *testing.T) {
		filter := services.QuestionFilter{}
		result, err := questionService.GetQuestions(1, 10, filter)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(3), result.Total) // All questions including inactive
		assert.Equal(t, 3, len(result.Questions))
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 10, result.PageSize)
		assert.Equal(t, 1, result.TotalPages)
	})

	t.Run("filter by tags", func(t *testing.T) {
		filter := services.QuestionFilter{
			Tags: []string{"geography"},
		}
		result, err := questionService.GetQuestions(1, 10, filter)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(1), result.Total)
		assert.Equal(t, 1, len(result.Questions))
		assert.Equal(t, "Geography Question", result.Questions[0].Title)
	})

	t.Run("filter by difficulty", func(t *testing.T) {
		filter := services.QuestionFilter{
			Difficulty: models.Easy,
		}
		result, err := questionService.GetQuestions(1, 10, filter)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(1), result.Total)
		assert.Equal(t, 1, len(result.Questions))
		assert.Equal(t, "Geography Question", result.Questions[0].Title)
	})

	t.Run("filter by type", func(t *testing.T) {
		filter := services.QuestionFilter{
			Type: models.MultipleChoice,
		}
		result, err := questionService.GetQuestions(1, 10, filter)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(3), result.Total) // All test questions are multiple choice
		assert.Equal(t, 3, len(result.Questions))
	})

	t.Run("filter by active status", func(t *testing.T) {
		isActive := true
		filter := services.QuestionFilter{
			IsActive: &isActive,
		}
		result, err := questionService.GetQuestions(1, 10, filter)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(2), result.Total) // Only active questions
		assert.Equal(t, 2, len(result.Questions))
	})

	t.Run("search by title", func(t *testing.T) {
		filter := services.QuestionFilter{
			Search: "Geography",
		}
		result, err := questionService.GetQuestions(1, 10, filter)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(1), result.Total)
		assert.Equal(t, 1, len(result.Questions))
		assert.Equal(t, "Geography Question", result.Questions[0].Title)
	})

	t.Run("pagination", func(t *testing.T) {
		filter := services.QuestionFilter{}
		result, err := questionService.GetQuestions(1, 2, filter)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(3), result.Total)
		assert.Equal(t, 2, len(result.Questions))
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 2, result.PageSize)
		assert.Equal(t, 2, result.TotalPages)
	})
}

func TestQuestionService_UpdateQuestion(t *testing.T) {
	db := setupQuestionTestDB()
	logger := logrus.New()
	questionService := services.NewQuestionService(db, logger)

	// Create test user
	testUser := models.User{
		ID:       1,
		Email:    "admin@example.com",
		Username: "admin",
		Role:     models.RoleAdmin,
		IsActive: true,
	}
	db.Create(&testUser)

	// Create test question
	question := models.Question{
		Title:      "Original Question",
		Content:    "Original content",
		Type:       models.MultipleChoice,
		Difficulty: models.Easy,
		Options: models.Options{
			{ID: "a", Text: "Option A", IsCorrect: true},
			{ID: "b", Text: "Option B", IsCorrect: false},
		},
		Tags:      models.StringArray{"original"},
		Points:    1,
		TimeLimit: 60,
		IsActive:  true,
		CreatedBy: testUser.ID,
	}
	db.Create(&question)

	t.Run("successful question update", func(t *testing.T) {
		req := services.UpdateQuestionRequest{
			Title:   "Updated Question",
			Content: "Updated content",
			Type:    models.MultipleChoice,
			Difficulty: models.Medium,
			Options: []models.Option{
				{ID: "a", Text: "Updated Option A", IsCorrect: true},
				{ID: "b", Text: "Updated Option B", IsCorrect: false},
				{ID: "c", Text: "New Option C", IsCorrect: false},
			},
			Tags:        []string{"updated", "test"},
			Points:      2,
			TimeLimit:   90,
			Explanation: "Updated explanation",
			IsActive:    true,
		}

		updatedQuestion, err := questionService.UpdateQuestion(question.ID, req)

		assert.NoError(t, err)
		assert.NotNil(t, updatedQuestion)
		assert.Equal(t, req.Title, updatedQuestion.Title)
		assert.Equal(t, req.Content, updatedQuestion.Content)
		assert.Equal(t, req.Difficulty, updatedQuestion.Difficulty)
		assert.Equal(t, len(req.Options), len(updatedQuestion.Options))
		assert.Equal(t, req.Tags, []string(updatedQuestion.Tags))
		assert.Equal(t, req.Points, updatedQuestion.Points)
		assert.Equal(t, req.TimeLimit, updatedQuestion.TimeLimit)
		assert.Equal(t, req.Explanation, updatedQuestion.Explanation)
		assert.Equal(t, req.IsActive, updatedQuestion.IsActive)
	})

	t.Run("question not found", func(t *testing.T) {
		req := services.UpdateQuestionRequest{
			Title:   "Updated Question",
			Content: "Updated content",
			Type:    models.MultipleChoice,
			Difficulty: models.Medium,
			Options: []models.Option{
				{ID: "a", Text: "Option A", IsCorrect: true},
				{ID: "b", Text: "Option B", IsCorrect: false},
			},
			Tags:      []string{"test"},
			Points:    1,
			TimeLimit: 60,
			IsActive:  true,
		}

		updatedQuestion, err := questionService.UpdateQuestion(999, req)

		assert.Error(t, err)
		assert.Nil(t, updatedQuestion)
		assert.Contains(t, err.Error(), "question not found")
	})
}

func TestQuestionService_DeleteQuestion(t *testing.T) {
	db := setupQuestionTestDB()
	logger := logrus.New()
	questionService := services.NewQuestionService(db, logger)

	// Create test user
	testUser := models.User{
		ID:       1,
		Email:    "admin@example.com",
		Username: "admin",
		Role:     models.RoleAdmin,
		IsActive: true,
	}
	db.Create(&testUser)

	// Create test question
	question := models.Question{
		Title:      "Test Question",
		Content:    "Test content",
		Type:       models.MultipleChoice,
		Difficulty: models.Easy,
		Options: models.Options{
			{ID: "a", Text: "Option A", IsCorrect: true},
			{ID: "b", Text: "Option B", IsCorrect: false},
		},
		Tags:      models.StringArray{"test"},
		Points:    1,
		TimeLimit: 60,
		IsActive:  true,
		CreatedBy: testUser.ID,
	}
	db.Create(&question)

	t.Run("successful question deletion", func(t *testing.T) {
		err := questionService.DeleteQuestion(question.ID)

		assert.NoError(t, err)

		// Verify question is soft deleted
		var deletedQuestion models.Question
		err = db.Unscoped().Where("id = ?", question.ID).First(&deletedQuestion).Error
		assert.NoError(t, err)
		assert.NotNil(t, deletedQuestion.DeletedAt)
	})

	t.Run("question not found", func(t *testing.T) {
		err := questionService.DeleteQuestion(999)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "question not found")
	})
}

