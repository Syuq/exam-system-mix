package services

import (
	"exam-system/models"
	"fmt"
	"math/rand"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type QuestionService struct {
	db     *gorm.DB
	logger *logrus.Logger
}

type CreateQuestionRequest struct {
	Title       string                     `json:"title" binding:"required"`
	Content     string                     `json:"content" binding:"required"`
	Type        models.QuestionType        `json:"type" binding:"required"`
	Difficulty  models.QuestionDifficulty  `json:"difficulty" binding:"required"`
	Options     []models.Option            `json:"options" binding:"required,min=2"`
	Tags        []string                   `json:"tags" binding:"required,min=1"`
	Points      int                        `json:"points" binding:"min=1"`
	TimeLimit   int                        `json:"time_limit" binding:"min=10"`
	Explanation string                     `json:"explanation"`
}

type UpdateQuestionRequest struct {
	Title       string                     `json:"title" binding:"required"`
	Content     string                     `json:"content" binding:"required"`
	Type        models.QuestionType        `json:"type" binding:"required"`
	Difficulty  models.QuestionDifficulty  `json:"difficulty" binding:"required"`
	Options     []models.Option            `json:"options" binding:"required,min=2"`
	Tags        []string                   `json:"tags" binding:"required,min=1"`
	Points      int                        `json:"points" binding:"min=1"`
	TimeLimit   int                        `json:"time_limit" binding:"min=10"`
	Explanation string                     `json:"explanation"`
	IsActive    bool                       `json:"is_active"`
}

type QuestionListResponse struct {
	Questions  []models.QuestionResponse `json:"questions"`
	Total      int64                     `json:"total"`
	Page       int                       `json:"page"`
	PageSize   int                       `json:"page_size"`
	TotalPages int                       `json:"total_pages"`
}

type QuestionFilter struct {
	Tags       []string                   `json:"tags"`
	Difficulty models.QuestionDifficulty  `json:"difficulty"`
	Type       models.QuestionType        `json:"type"`
	Search     string                     `json:"search"`
	IsActive   *bool                      `json:"is_active"`
}

func NewQuestionService(db *gorm.DB, logger *logrus.Logger) *QuestionService {
	return &QuestionService{
		db:     db,
		logger: logger,
	}
}

func (s *QuestionService) CreateQuestion(req CreateQuestionRequest, createdBy uint) (*models.Question, error) {
	// Validate options
	if err := s.validateOptions(req.Options, req.Type); err != nil {
		return nil, err
	}

	question := models.Question{
		Title:       req.Title,
		Content:     req.Content,
		Type:        req.Type,
		Difficulty:  req.Difficulty,
		Options:     models.Options(req.Options),
		Tags:        models.StringArray(req.Tags),
		Points:      req.Points,
		TimeLimit:   req.TimeLimit,
		Explanation: req.Explanation,
		IsActive:    true,
		CreatedBy:   createdBy,
	}

	if err := s.db.Create(&question).Error; err != nil {
		s.logger.WithError(err).Error("Failed to create question")
		return nil, fmt.Errorf("failed to create question")
	}

	s.logger.WithFields(logrus.Fields{
		"question_id": question.ID,
		"title":       question.Title,
		"created_by":  createdBy,
	}).Info("Question created successfully")

	return &question, nil
}

func (s *QuestionService) GetQuestions(page, pageSize int, filter QuestionFilter) (*QuestionListResponse, error) {
	var questions []models.Question
	var total int64

	query := s.db.Model(&models.Question{}).Preload("Creator")

	// Apply filters
	if len(filter.Tags) > 0 {
		// Use PostgreSQL JSONB contains operator
		for _, tag := range filter.Tags {
			query = query.Where("tags @> ?", fmt.Sprintf(`["%s"]`, tag))
		}
	}

	if filter.Difficulty != "" {
		query = query.Where("difficulty = ?", filter.Difficulty)
	}

	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}

	if filter.Search != "" {
		searchPattern := "%" + filter.Search + "%"
		query = query.Where("title ILIKE ? OR content ILIKE ?", searchPattern, searchPattern)
	}

	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		s.logger.WithError(err).Error("Failed to count questions")
		return nil, fmt.Errorf("failed to get questions")
	}

	// Get paginated results
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&questions).Error; err != nil {
		s.logger.WithError(err).Error("Failed to get questions")
		return nil, fmt.Errorf("failed to get questions")
	}

	// Convert to response format
	questionResponses := make([]models.QuestionResponse, len(questions))
	for i, question := range questions {
		questionResponses[i] = question.ToResponse(true) // Include correct answers for admin
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return &QuestionListResponse{
		Questions:  questionResponses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *QuestionService) GetQuestion(questionID uint, includeCorrectAnswers bool) (*models.Question, error) {
	var question models.Question
	if err := s.db.Preload("Creator").Where("id = ?", questionID).First(&question).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("question not found")
		}
		s.logger.WithError(err).Error("Failed to get question")
		return nil, fmt.Errorf("failed to get question")
	}

	return &question, nil
}

func (s *QuestionService) UpdateQuestion(questionID uint, req UpdateQuestionRequest) (*models.Question, error) {
	var question models.Question
	if err := s.db.Where("id = ?", questionID).First(&question).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("question not found")
		}
		s.logger.WithError(err).Error("Failed to find question")
		return nil, fmt.Errorf("failed to update question")
	}

	// Validate options
	if err := s.validateOptions(req.Options, req.Type); err != nil {
		return nil, err
	}

	// Update question fields
	question.Title = req.Title
	question.Content = req.Content
	question.Type = req.Type
	question.Difficulty = req.Difficulty
	question.Options = models.Options(req.Options)
	question.Tags = models.StringArray(req.Tags)
	question.Points = req.Points
	question.TimeLimit = req.TimeLimit
	question.Explanation = req.Explanation
	question.IsActive = req.IsActive

	if err := s.db.Save(&question).Error; err != nil {
		s.logger.WithError(err).Error("Failed to update question")
		return nil, fmt.Errorf("failed to update question")
	}

	s.logger.WithFields(logrus.Fields{
		"question_id": question.ID,
		"title":       question.Title,
	}).Info("Question updated successfully")

	return &question, nil
}

func (s *QuestionService) DeleteQuestion(questionID uint) error {
	var question models.Question
	if err := s.db.Where("id = ?", questionID).First(&question).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("question not found")
		}
		s.logger.WithError(err).Error("Failed to find question")
		return fmt.Errorf("failed to delete question")
	}

	// Check if question is used in any active exams
	var examQuestionCount int64
	if err := s.db.Model(&models.ExamQuestion{}).
		Joins("JOIN exams ON exams.id = exam_questions.exam_id").
		Where("exam_questions.question_id = ? AND exams.status IN ?", questionID, []string{"active", "draft"}).
		Count(&examQuestionCount).Error; err != nil {
		s.logger.WithError(err).Error("Failed to check question usage")
		return fmt.Errorf("failed to delete question")
	}

	if examQuestionCount > 0 {
		return fmt.Errorf("cannot delete question as it is used in active or draft exams")
	}

	// Soft delete the question
	if err := s.db.Delete(&question).Error; err != nil {
		s.logger.WithError(err).Error("Failed to delete question")
		return fmt.Errorf("failed to delete question")
	}

	s.logger.WithFields(logrus.Fields{
		"question_id": question.ID,
		"title":       question.Title,
	}).Info("Question deleted successfully")

	return nil
}

func (s *QuestionService) GetRandomQuestionsByTags(tags []string, count int, difficulty models.QuestionDifficulty) ([]models.Question, error) {
	var questions []models.Question

	query := s.db.Where("is_active = ?", true)

	// Filter by tags if provided
	if len(tags) > 0 {
		for _, tag := range tags {
			query = query.Where("tags @> ?", fmt.Sprintf(`["%s"]`, tag))
		}
	}

	// Filter by difficulty if provided
	if difficulty != "" {
		query = query.Where("difficulty = ?", difficulty)
	}

	if err := query.Find(&questions).Error; err != nil {
		s.logger.WithError(err).Error("Failed to get questions by tags")
		return nil, fmt.Errorf("failed to get questions")
	}

	// Shuffle and select random questions
	if len(questions) > count {
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(questions), func(i, j int) {
			questions[i], questions[j] = questions[j], questions[i]
		})
		questions = questions[:count]
	}

	return questions, nil
}

func (s *QuestionService) GetAllTags() ([]string, error) {
	var tags []string

	rows, err := s.db.Raw(`
		SELECT DISTINCT jsonb_array_elements_text(tags) as tag 
		FROM questions 
		WHERE is_active = true AND deleted_at IS NULL
		ORDER BY tag
	`).Rows()
	if err != nil {
		s.logger.WithError(err).Error("Failed to get tags")
		return nil, fmt.Errorf("failed to get tags")
	}
	defer rows.Close()

	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			continue
		}
		tags = append(tags, tag)
	}

	return tags, nil
}

func (s *QuestionService) validateOptions(options []models.Option, questionType models.QuestionType) error {
	if len(options) < 2 {
		return fmt.Errorf("question must have at least 2 options")
	}

	correctCount := 0
	for _, option := range options {
		if option.Text == "" {
			return fmt.Errorf("option text cannot be empty")
		}
		if option.ID == "" {
			return fmt.Errorf("option ID cannot be empty")
		}
		if option.IsCorrect {
			correctCount++
		}
	}

	if correctCount == 0 {
		return fmt.Errorf("question must have at least one correct answer")
	}

	if questionType == models.TrueFalse {
		if len(options) != 2 {
			return fmt.Errorf("true/false questions must have exactly 2 options")
		}
		if correctCount != 1 {
			return fmt.Errorf("true/false questions must have exactly one correct answer")
		}
	}

	return nil
}

