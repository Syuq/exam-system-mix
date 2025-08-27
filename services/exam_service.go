package services

import (
	"exam-system/models"
	"exam-system/utils"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type ExamService struct {
	db          *gorm.DB
	redisClient *utils.RedisClient
	logger      *logrus.Logger
}

type CreateExamRequest struct {
	Title       string                `json:"title" binding:"required"`
	Description string                `json:"description"`
	Duration    int                   `json:"duration" binding:"required,min=1"` // in minutes
	PassScore   int                   `json:"pass_score" binding:"min=0,max=100"`
	StartTime   time.Time             `json:"start_time"`
	EndTime     time.Time             `json:"end_time"`
	Questions   []ExamQuestionRequest `json:"questions" binding:"required,min=1"`
}

type ExamQuestionRequest struct {
	QuestionID uint `json:"question_id" binding:"required"`
	Points     int  `json:"points" binding:"min=1"`
	Order      int  `json:"order" binding:"min=1"`
}

type UpdateExamRequest struct {
	Title       string                `json:"title" binding:"required"`
	Description string                `json:"description"`
	Duration    int                   `json:"duration" binding:"required,min=1"`
	PassScore   int                   `json:"pass_score" binding:"min=0,max=100"`
	StartTime   *time.Time            `json:"start_time"`
	EndTime     *time.Time            `json:"end_time"`
	Status      models.ExamStatus     `json:"status" binding:"required"`
	Questions   []ExamQuestionRequest `json:"questions" binding:"required,min=1"`
}

type AssignExamRequest struct {
	UserIDs     []uint     `json:"user_ids" binding:"required,min=1"`
	ExpiresAt   *time.Time `json:"expires_at"`
	MaxAttempts int        `json:"max_attempts" binding:"min=1"`
}

type StartExamResponse struct {
	UserExam  models.UserExamResponse   `json:"user_exam"`
	Questions []models.QuestionResponse `json:"questions"`
	TimeLeft  int                       `json:"time_left"` // in seconds
}

type SubmitExamRequest struct {
	Answers []SubmitAnswerRequest `json:"answers" binding:"required"`
}

type SubmitAnswerRequest struct {
	QuestionID      uint     `json:"question_id" binding:"required"`
	SelectedOptions []string `json:"selected_options" binding:"required"`
	TimeSpent       int      `json:"time_spent" binding:"min=0"` // in seconds
}

type ExamListResponse struct {
	Exams      []models.ExamResponse `json:"exams"`
	Total      int64                 `json:"total"`
	Page       int                   `json:"page"`
	PageSize   int                   `json:"page_size"`
	TotalPages int                   `json:"total_pages"`
}

func NewExamService(db *gorm.DB, redisClient *utils.RedisClient, logger *logrus.Logger) *ExamService {
	return &ExamService{
		db:          db,
		redisClient: redisClient,
		logger:      logger,
	}
}

func (s *ExamService) CreateExam(req CreateExamRequest, createdBy uint) (*models.Exam, error) {
	// Validate questions exist
	questionIDs := make([]uint, len(req.Questions))
	for i, q := range req.Questions {
		questionIDs[i] = q.QuestionID
	}

	var questionCount int64
	if err := s.db.Model(&models.Question{}).Where("id IN ? AND is_active = ?", questionIDs, true).Count(&questionCount).Error; err != nil {
		s.logger.WithError(err).Error("Failed to validate questions")
		return nil, fmt.Errorf("failed to validate questions")
	}

	if int(questionCount) != len(questionIDs) {
		return nil, fmt.Errorf("some questions are invalid or inactive")
	}

	// Calculate total points
	totalPoints := 0
	for _, q := range req.Questions {
		totalPoints += q.Points
	}

	// Create exam
	exam := models.Exam{
		Title:       req.Title,
		Description: req.Description,
		Duration:    req.Duration,
		TotalPoints: totalPoints,
		PassScore:   req.PassScore,
		Status:      models.ExamDraft,
		StartTime:   &req.StartTime,
		EndTime:     &req.EndTime,
		IsActive:    true,
		CreatedBy:   createdBy,
	}

	// Start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Create(&exam).Error; err != nil {
		tx.Rollback()
		s.logger.WithError(err).Error("Failed to create exam")
		return nil, fmt.Errorf("failed to create exam")
	}

	// Create exam questions
	for _, q := range req.Questions {
		examQuestion := models.ExamQuestion{
			ExamID:     exam.ID,
			QuestionID: q.QuestionID,
			Order:      q.Order,
			Points:     q.Points,
		}

		if err := tx.Create(&examQuestion).Error; err != nil {
			tx.Rollback()
			s.logger.WithError(err).Error("Failed to create exam question")
			return nil, fmt.Errorf("failed to create exam")
		}
	}

	if err := tx.Commit().Error; err != nil {
		s.logger.WithError(err).Error("Failed to commit exam creation")
		return nil, fmt.Errorf("failed to create exam")
	}

	s.logger.WithFields(logrus.Fields{
		"exam_id":    exam.ID,
		"title":      exam.Title,
		"created_by": createdBy,
	}).Info("Exam created successfully")

	return &exam, nil
}

func (s *ExamService) GetExams(page, pageSize int, userID uint, isAdmin bool) (*ExamListResponse, error) {
	var exams []models.Exam
	var total int64

	query := s.db.Model(&models.Exam{}).Preload("Creator")

	if !isAdmin {
		// For regular users, only show exams assigned to them
		query = query.Joins("JOIN user_exams ON user_exams.exam_id = exams.id").
			Where("user_exams.user_id = ?", userID)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		s.logger.WithError(err).Error("Failed to count exams")
		return nil, fmt.Errorf("failed to get exams")
	}

	// Get paginated results
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&exams).Error; err != nil {
		s.logger.WithError(err).Error("Failed to get exams")
		return nil, fmt.Errorf("failed to get exams")
	}

	// Convert to response format
	examResponses := make([]models.ExamResponse, len(exams))
	for i, exam := range exams {
		var userExam *models.UserExam
		if !isAdmin {
			// Get user exam info for regular users
			s.db.Where("user_id = ? AND exam_id = ?", userID, exam.ID).First(&userExam)
		}
		examResponses[i] = exam.ToResponse(false, userExam)
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return &ExamListResponse{
		Exams:      examResponses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *ExamService) GetExam(examID uint, userID uint, isAdmin bool) (*models.Exam, *models.UserExam, error) {
	var exam models.Exam
	query := s.db.Preload("Creator").Preload("ExamQuestions.Question")

	if err := query.Where("id = ?", examID).First(&exam).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil, fmt.Errorf("exam not found")
		}
		s.logger.WithError(err).Error("Failed to get exam")
		return nil, nil, fmt.Errorf("failed to get exam")
	}

	var userExam *models.UserExam
	if !isAdmin {
		// Check if user has access to this exam
		var ue models.UserExam
		if err := s.db.Where("user_id = ? AND exam_id = ?", userID, examID).First(&ue).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, nil, fmt.Errorf("exam not assigned to user")
			}
			s.logger.WithError(err).Error("Failed to get user exam")
			return nil, nil, fmt.Errorf("failed to get exam")
		}
		userExam = &ue
	}

	return &exam, userExam, nil
}

func (s *ExamService) UpdateExam(examID uint, req UpdateExamRequest) (*models.Exam, error) {
	var exam models.Exam
	if err := s.db.Where("id = ?", examID).First(&exam).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("exam not found")
		}
		s.logger.WithError(err).Error("Failed to find exam")
		return nil, fmt.Errorf("failed to update exam")
	}

	// Check if exam can be updated
	if exam.Status == models.ExamCompleted {
		return nil, fmt.Errorf("cannot update completed exam")
	}

	// Validate questions exist
	questionIDs := make([]uint, len(req.Questions))
	for i, q := range req.Questions {
		questionIDs[i] = q.QuestionID
	}

	var questionCount int64
	if err := s.db.Model(&models.Question{}).Where("id IN ? AND is_active = ?", questionIDs, true).Count(&questionCount).Error; err != nil {
		s.logger.WithError(err).Error("Failed to validate questions")
		return nil, fmt.Errorf("failed to validate questions")
	}

	if int(questionCount) != len(questionIDs) {
		return nil, fmt.Errorf("some questions are invalid or inactive")
	}

	// Calculate total points
	totalPoints := 0
	for _, q := range req.Questions {
		totalPoints += q.Points
	}

	// Start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update exam
	exam.Title = req.Title
	exam.Description = req.Description
	exam.Duration = req.Duration
	exam.TotalPoints = totalPoints
	exam.PassScore = req.PassScore
	exam.Status = req.Status
	exam.StartTime = req.StartTime
	exam.EndTime = req.EndTime

	if err := tx.Save(&exam).Error; err != nil {
		tx.Rollback()
		s.logger.WithError(err).Error("Failed to update exam")
		return nil, fmt.Errorf("failed to update exam")
	}

	// Delete existing exam questions
	if err := tx.Where("exam_id = ?", examID).Delete(&models.ExamQuestion{}).Error; err != nil {
		tx.Rollback()
		s.logger.WithError(err).Error("Failed to delete existing exam questions")
		return nil, fmt.Errorf("failed to update exam")
	}

	// Create new exam questions
	for _, q := range req.Questions {
		examQuestion := models.ExamQuestion{
			ExamID:     exam.ID,
			QuestionID: q.QuestionID,
			Order:      q.Order,
			Points:     q.Points,
		}

		if err := tx.Create(&examQuestion).Error; err != nil {
			tx.Rollback()
			s.logger.WithError(err).Error("Failed to create exam question")
			return nil, fmt.Errorf("failed to update exam")
		}
	}

	if err := tx.Commit().Error; err != nil {
		s.logger.WithError(err).Error("Failed to commit exam update")
		return nil, fmt.Errorf("failed to update exam")
	}

	s.logger.WithFields(logrus.Fields{
		"exam_id": exam.ID,
		"title":   exam.Title,
	}).Info("Exam updated successfully")

	return &exam, nil
}

func (s *ExamService) DeleteExam(examID uint) error {
	var exam models.Exam
	if err := s.db.Where("id = ?", examID).First(&exam).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("exam not found")
		}
		s.logger.WithError(err).Error("Failed to find exam")
		return fmt.Errorf("failed to delete exam")
	}

	// Check if exam has any completed attempts
	var resultCount int64
	if err := s.db.Model(&models.Result{}).Where("exam_id = ?", examID).Count(&resultCount).Error; err != nil {
		s.logger.WithError(err).Error("Failed to check exam results")
		return fmt.Errorf("failed to delete exam")
	}

	if resultCount > 0 {
		return fmt.Errorf("cannot delete exam with existing results")
	}

	// Soft delete the exam
	if err := s.db.Delete(&exam).Error; err != nil {
		s.logger.WithError(err).Error("Failed to delete exam")
		return fmt.Errorf("failed to delete exam")
	}

	s.logger.WithFields(logrus.Fields{
		"exam_id": exam.ID,
		"title":   exam.Title,
	}).Info("Exam deleted successfully")

	return nil
}

func (s *ExamService) AssignExam(examID uint, req AssignExamRequest) error {
	// Validate exam exists and is active
	var exam models.Exam
	if err := s.db.Where("id = ? AND is_active = ?", examID, true).First(&exam).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("exam not found or inactive")
		}
		s.logger.WithError(err).Error("Failed to find exam")
		return fmt.Errorf("failed to assign exam")
	}

	// Validate users exist
	var userCount int64
	if err := s.db.Model(&models.User{}).Where("id IN ? AND is_active = ?", req.UserIDs, true).Count(&userCount).Error; err != nil {
		s.logger.WithError(err).Error("Failed to validate users")
		return fmt.Errorf("failed to assign exam")
	}

	if int(userCount) != len(req.UserIDs) {
		return fmt.Errorf("some users are invalid or inactive")
	}

	// Create user exam assignments
	for _, userID := range req.UserIDs {
		userExam := models.UserExam{
			UserID:      userID,
			ExamID:      examID,
			Status:      models.UserExamAssigned,
			ExpiresAt:   req.ExpiresAt,
			MaxAttempts: req.MaxAttempts,
		}

		// Use ON CONFLICT to handle duplicates
		if err := s.db.Create(&userExam).Error; err != nil {
			// If it's a duplicate key error, update the existing record
			if err := s.db.Where("user_id = ? AND exam_id = ?", userID, examID).Updates(&userExam).Error; err != nil {
				s.logger.WithError(err).Error("Failed to assign exam to user")
				continue
			}
		}
	}

	s.logger.WithFields(logrus.Fields{
		"exam_id":  examID,
		"user_ids": req.UserIDs,
	}).Info("Exam assigned successfully")

	return nil
}

func (s *ExamService) StartExam(examID uint, userID uint) (*StartExamResponse, error) {
	// Get user exam
	var userExam models.UserExam
	if err := s.db.Where("user_id = ? AND exam_id = ?", userID, examID).First(&userExam).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("exam not assigned to user")
		}
		s.logger.WithError(err).Error("Failed to get user exam")
		return nil, fmt.Errorf("failed to start exam")
	}

	// Check if user can start the exam
	if !userExam.CanStart() {
		return nil, fmt.Errorf("exam cannot be started")
	}

	// Get exam with questions
	var exam models.Exam
	if err := s.db.Preload("ExamQuestions.Question").Where("id = ?", examID).First(&exam).Error; err != nil {
		s.logger.WithError(err).Error("Failed to get exam")
		return nil, fmt.Errorf("failed to start exam")
	}

	// Update user exam status
	now := time.Now()
	userExam.Status = models.UserExamStarted
	userExam.StartedAt = &now
	userExam.AttemptCount++

	if err := s.db.Save(&userExam).Error; err != nil {
		s.logger.WithError(err).Error("Failed to update user exam")
		return nil, fmt.Errorf("failed to start exam")
	}

	// Prepare questions (without correct answers)
	questions := make([]models.QuestionResponse, len(exam.ExamQuestions))
	for i, eq := range exam.ExamQuestions {
		questions[i] = eq.Question.ToResponse(false)
	}

	// Calculate time left
	examDuration := time.Duration(exam.Duration) * time.Minute
	timeLeft := int(examDuration.Seconds())

	// Store exam session in Redis for timer validation
	sessionKey := fmt.Sprintf("exam_session:%d:%d", userID, examID)
	sessionData := map[string]interface{}{
		"started_at": now.Unix(),
		"duration":   exam.Duration * 60, // in seconds
	}
	if err := s.redisClient.SetJSON(sessionKey, sessionData, examDuration); err != nil {
		s.logger.WithError(err).Warn("Failed to store exam session in Redis")
	}

	// Convert UserExam to UserExamResponse
	userExamResponse := s.convertUserExamToResponse(&userExam, &exam)

	response := &StartExamResponse{
		UserExam:  *userExamResponse,
		Questions: questions,
		TimeLeft:  timeLeft,
	}

	s.logger.WithFields(logrus.Fields{
		"exam_id": examID,
		"user_id": userID,
	}).Info("Exam started successfully")

	return response, nil
}

func (s *ExamService) SubmitExam(examID uint, userID uint, req SubmitExamRequest) (*models.Result, error) {
	// Get user exam
	var userExam models.UserExam
	if err := s.db.Where("user_id = ? AND exam_id = ?", userID, examID).First(&userExam).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("exam not assigned to user")
		}
		s.logger.WithError(err).Error("Failed to get user exam")
		return nil, fmt.Errorf("failed to submit exam")
	}

	// Check if user can submit the exam
	if !userExam.CanSubmit() {
		return nil, fmt.Errorf("exam cannot be submitted")
	}

	// Validate timer using Redis session
	sessionKey := fmt.Sprintf("exam_session:%d:%d", userID, examID)
	var sessionData map[string]interface{}
	if err := s.redisClient.GetJSON(sessionKey, &sessionData); err == nil {
		startedAt := int64(sessionData["started_at"].(float64))
		duration := int64(sessionData["duration"].(float64))
		elapsed := time.Now().Unix() - startedAt

		if elapsed > duration {
			// Auto-submit due to time expiry
			s.logger.WithFields(logrus.Fields{
				"exam_id":  examID,
				"user_id":  userID,
				"elapsed":  elapsed,
				"duration": duration,
			}).Warn("Exam auto-submitted due to time expiry")
		}
	}

	// Get exam with questions
	var exam models.Exam
	if err := s.db.Preload("ExamQuestions.Question").Where("id = ?", examID).First(&exam).Error; err != nil {
		s.logger.WithError(err).Error("Failed to get exam")
		return nil, fmt.Errorf("failed to submit exam")
	}

	// Process answers and calculate score
	result, err := s.processExamSubmission(&exam, &userExam, req.Answers)
	if err != nil {
		return nil, err
	}

	// Update user exam status
	now := time.Now()
	userExam.Status = models.UserExamCompleted
	userExam.CompletedAt = &now

	if err := s.db.Save(&userExam).Error; err != nil {
		s.logger.WithError(err).Error("Failed to update user exam")
		return nil, fmt.Errorf("failed to submit exam")
	}

	// Clean up Redis session
	s.redisClient.Del(sessionKey)

	s.logger.WithFields(logrus.Fields{
		"exam_id": examID,
		"user_id": userID,
		"score":   result.Score,
		"passed":  result.Passed,
	}).Info("Exam submitted successfully")

	return result, nil
}

func (s *ExamService) processExamSubmission(exam *models.Exam, userExam *models.UserExam, submittedAnswers []SubmitAnswerRequest) (*models.Result, error) {
	// Create answer map for quick lookup
	answerMap := make(map[uint]SubmitAnswerRequest)
	for _, answer := range submittedAnswers {
		answerMap[answer.QuestionID] = answer
	}

	var answers []models.Answer
	totalPoints := 0
	earnedPoints := 0

	// Process each question
	for _, eq := range exam.ExamQuestions {
		question := eq.Question
		totalPoints += eq.Points

		answer := models.Answer{
			QuestionID:      question.ID,
			SelectedOptions: []string{},
			IsCorrect:       false,
			Points:          0,
			TimeSpent:       0,
		}

		// Check if user provided an answer
		if submittedAnswer, exists := answerMap[question.ID]; exists {
			answer.SelectedOptions = submittedAnswer.SelectedOptions
			answer.TimeSpent = submittedAnswer.TimeSpent

			// Validate answer - you'll need to implement this method in your Question model
			if s.validateQuestionAnswer(&question, submittedAnswer.SelectedOptions) {
				answer.IsCorrect = true
				answer.Points = eq.Points
				earnedPoints += eq.Points
			}
		}

		answers = append(answers, answer)
	}

	// Calculate score percentage
	score := float64(earnedPoints) / float64(totalPoints) * 100
	passed := score >= float64(exam.PassScore)

	// Calculate duration
	duration := int(time.Since(*userExam.StartedAt).Seconds())

	// Create result
	result := models.Result{
		UserID:      userExam.UserID,
		ExamID:      exam.ID,
		UserExamID:  userExam.ID,
		Score:       score,
		TotalPoints: earnedPoints,
		MaxPoints:   totalPoints,
		Passed:      passed,
		Answers:     models.Answers(answers),
		StartTime:   *userExam.StartedAt,
		EndTime:     time.Now(),
		Duration:    duration,
	}

	if err := s.db.Create(&result).Error; err != nil {
		s.logger.WithError(err).Error("Failed to create result")
		return nil, fmt.Errorf("failed to save exam result")
	}

	return &result, nil
}

// Helper method to validate question answer
// This assumes you have correct answer data in your Question model
func (s *ExamService) validateQuestionAnswer(question *models.Question, selectedOptions []string) bool {
	// You'll need to implement this based on your Question model structure
	// This is a placeholder - replace with your actual validation logic
	// For example, if you have a CorrectAnswers field in your Question model:
	// return question.ValidateAnswer(selectedOptions)

	// Placeholder implementation - replace with actual logic
	return len(selectedOptions) > 0
}

// Helper method to convert UserExam to UserExamResponse
func (s *ExamService) convertUserExamToResponse(userExam *models.UserExam, exam *models.Exam) *models.UserExamResponse {
	response := &models.UserExamResponse{
		ID:           userExam.ID,
		Status:       userExam.Status,
		StartedAt:    userExam.StartedAt,
		CompletedAt:  userExam.CompletedAt,
		ExpiresAt:    userExam.ExpiresAt,
		AttemptCount: userExam.AttemptCount,
		MaxAttempts:  userExam.MaxAttempts,
	}

	// Calculate time left if exam is started
	if userExam.StartedAt != nil && userExam.Status == models.UserExamStarted {
		examDuration := time.Duration(exam.Duration) * time.Minute
		elapsed := time.Since(*userExam.StartedAt)
		timeLeft := int((examDuration - elapsed).Seconds())
		if timeLeft < 0 {
			timeLeft = 0
		}
		response.TimeLeft = &timeLeft
	}

	return response
}
