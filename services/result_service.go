package services

import (
	"exam-system/models"
	"fmt"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type ResultService struct {
	db     *gorm.DB
	logger *logrus.Logger
}

type ResultListResponse struct {
	Results    []models.ResultResponse `json:"results"`
	Total      int64                   `json:"total"`
	Page       int                     `json:"page"`
	PageSize   int                     `json:"page_size"`
	TotalPages int                     `json:"total_pages"`
}

type StatisticsResponse struct {
	ExamStatistics     []models.ExamStatistics     `json:"exam_statistics"`
	UserStatistics     []models.UserStatistics     `json:"user_statistics"`
	QuestionStatistics []models.QuestionStatistics `json:"question_statistics"`
	OverallStats       OverallStatistics           `json:"overall_stats"`
}

type OverallStatistics struct {
	TotalExams       int     `json:"total_exams"`
	TotalUsers       int     `json:"total_users"`
	TotalAttempts    int     `json:"total_attempts"`
	AverageScore     float64 `json:"average_score"`
	PassRate         float64 `json:"pass_rate"`
	TotalTimeSpent   int     `json:"total_time_spent"` // in seconds
	AverageDuration  int     `json:"average_duration"` // in seconds
}

func NewResultService(db *gorm.DB, logger *logrus.Logger) *ResultService {
	return &ResultService{
		db:     db,
		logger: logger,
	}
}

func (s *ResultService) GetResults(page, pageSize int, userID uint, examID *uint, isAdmin bool) (*ResultListResponse, error) {
	var results []models.Result
	var total int64

	query := s.db.Model(&models.Result{}).
		Preload("User").
		Preload("Exam")

	// Filter by user if not admin
	if !isAdmin {
		query = query.Where("user_id = ?", userID)
	}

	// Filter by exam if specified
	if examID != nil {
		query = query.Where("exam_id = ?", *examID)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		s.logger.WithError(err).Error("Failed to count results")
		return nil, fmt.Errorf("failed to get results")
	}

	// Get paginated results
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&results).Error; err != nil {
		s.logger.WithError(err).Error("Failed to get results")
		return nil, fmt.Errorf("failed to get results")
	}

	// Convert to response format
	resultResponses := make([]models.ResultResponse, len(results))
	for i, result := range results {
		resultResponses[i] = result.ToResponse(true, isAdmin) // Include answers, correct answers for admin
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return &ResultListResponse{
		Results:    resultResponses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *ResultService) GetResult(resultID uint, userID uint, isAdmin bool) (*models.Result, error) {
	var result models.Result
	query := s.db.Preload("User").Preload("Exam").Preload("UserExam")

	if !isAdmin {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.Where("id = ?", resultID).First(&result).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("result not found")
		}
		s.logger.WithError(err).Error("Failed to get result")
		return nil, fmt.Errorf("failed to get result")
	}

	// Load question details for answers
	if err := s.loadQuestionDetailsForResult(&result); err != nil {
		s.logger.WithError(err).Warn("Failed to load question details for result")
	}

	return &result, nil
}

func (s *ResultService) GetUserResults(userID uint, page, pageSize int) (*ResultListResponse, error) {
	return s.GetResults(page, pageSize, userID, nil, false)
}

func (s *ResultService) GetExamResults(examID uint, page, pageSize int) (*ResultListResponse, error) {
	return s.GetResults(page, pageSize, 0, &examID, true)
}

func (s *ResultService) GetStatistics() (*StatisticsResponse, error) {
	// Get exam statistics
	examStats, err := s.getExamStatistics()
	if err != nil {
		s.logger.WithError(err).Error("Failed to get exam statistics")
		return nil, fmt.Errorf("failed to get statistics")
	}

	// Get user statistics
	userStats, err := s.getUserStatistics()
	if err != nil {
		s.logger.WithError(err).Error("Failed to get user statistics")
		return nil, fmt.Errorf("failed to get statistics")
	}

	// Get question statistics
	questionStats, err := s.getQuestionStatistics()
	if err != nil {
		s.logger.WithError(err).Error("Failed to get question statistics")
		return nil, fmt.Errorf("failed to get statistics")
	}

	// Get overall statistics
	overallStats, err := s.getOverallStatistics()
	if err != nil {
		s.logger.WithError(err).Error("Failed to get overall statistics")
		return nil, fmt.Errorf("failed to get statistics")
	}

	return &StatisticsResponse{
		ExamStatistics:     examStats,
		UserStatistics:     userStats,
		QuestionStatistics: questionStats,
		OverallStats:       overallStats,
	}, nil
}

func (s *ResultService) getExamStatistics() ([]models.ExamStatistics, error) {
	var stats []models.ExamStatistics

	rows, err := s.db.Raw(`
		SELECT 
			e.id as exam_id,
			e.title as exam_title,
			COUNT(r.id) as total_attempts,
			COUNT(CASE WHEN r.passed = true THEN 1 END) as passed_attempts,
			COUNT(CASE WHEN r.passed = false THEN 1 END) as failed_attempts,
			COALESCE(AVG(CASE WHEN r.passed = true THEN 1.0 ELSE 0.0 END) * 100, 0) as pass_rate,
			COALESCE(AVG(r.score), 0) as average_score,
			COALESCE(MAX(r.score), 0) as highest_score,
			COALESCE(MIN(r.score), 0) as lowest_score,
			COALESCE(AVG(r.duration), 0) as average_duration
		FROM exams e
		LEFT JOIN results r ON e.id = r.exam_id
		WHERE e.deleted_at IS NULL
		GROUP BY e.id, e.title
		ORDER BY total_attempts DESC
	`).Rows()

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var stat models.ExamStatistics
		err := rows.Scan(
			&stat.ExamID,
			&stat.ExamTitle,
			&stat.TotalAttempts,
			&stat.PassedAttempts,
			&stat.FailedAttempts,
			&stat.PassRate,
			&stat.AverageScore,
			&stat.HighestScore,
			&stat.LowestScore,
			&stat.AverageDuration,
		)
		if err != nil {
			continue
		}
		stats = append(stats, stat)
	}

	return stats, nil
}

func (s *ResultService) getUserStatistics() ([]models.UserStatistics, error) {
	var stats []models.UserStatistics

	rows, err := s.db.Raw(`
		SELECT 
			u.id as user_id,
			u.username,
			COUNT(r.id) as total_exams,
			COUNT(CASE WHEN r.passed = true THEN 1 END) as passed_exams,
			COUNT(CASE WHEN r.passed = false THEN 1 END) as failed_exams,
			COALESCE(AVG(CASE WHEN r.passed = true THEN 1.0 ELSE 0.0 END) * 100, 0) as pass_rate,
			COALESCE(AVG(r.score), 0) as average_score,
			COALESCE(MAX(r.score), 0) as highest_score,
			COALESCE(MIN(r.score), 0) as lowest_score,
			COALESCE(SUM(r.duration), 0) as total_time_spent
		FROM users u
		LEFT JOIN results r ON u.id = r.user_id
		WHERE u.deleted_at IS NULL AND u.role = 'user'
		GROUP BY u.id, u.username
		HAVING COUNT(r.id) > 0
		ORDER BY total_exams DESC
		LIMIT 50
	`).Rows()

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var stat models.UserStatistics
		err := rows.Scan(
			&stat.UserID,
			&stat.Username,
			&stat.TotalExams,
			&stat.PassedExams,
			&stat.FailedExams,
			&stat.PassRate,
			&stat.AverageScore,
			&stat.HighestScore,
			&stat.LowestScore,
			&stat.TotalTimeSpent,
		)
		if err != nil {
			continue
		}
		stats = append(stats, stat)
	}

	return stats, nil
}

func (s *ResultService) getQuestionStatistics() ([]models.QuestionStatistics, error) {
	var stats []models.QuestionStatistics

	rows, err := s.db.Raw(`
		SELECT 
			q.id as question_id,
			q.title as question_title,
			COUNT(a.question_id) as total_attempts,
			COUNT(CASE WHEN a.is_correct = true THEN 1 END) as correct_attempts,
			COUNT(CASE WHEN a.is_correct = false THEN 1 END) as wrong_attempts,
			COALESCE(AVG(CASE WHEN a.is_correct = true THEN 1.0 ELSE 0.0 END) * 100, 0) as success_rate,
			COALESCE(AVG(a.time_spent), 0) as average_time_spent
		FROM questions q
		JOIN (
			SELECT 
				(jsonb_array_elements(r.answers)->>'question_id')::int as question_id,
				(jsonb_array_elements(r.answers)->>'is_correct')::boolean as is_correct,
				(jsonb_array_elements(r.answers)->>'time_spent')::int as time_spent
			FROM results r
		) a ON q.id = a.question_id
		WHERE q.deleted_at IS NULL
		GROUP BY q.id, q.title
		ORDER BY total_attempts DESC
		LIMIT 100
	`).Rows()

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var stat models.QuestionStatistics
		err := rows.Scan(
			&stat.QuestionID,
			&stat.QuestionTitle,
			&stat.TotalAttempts,
			&stat.CorrectAttempts,
			&stat.WrongAttempts,
			&stat.SuccessRate,
			&stat.AverageTimeSpent,
		)
		if err != nil {
			continue
		}
		stats = append(stats, stat)
	}

	return stats, nil
}

func (s *ResultService) getOverallStatistics() (OverallStatistics, error) {
	var stats OverallStatistics

	// Get overall statistics
	row := s.db.Raw(`
		SELECT 
			(SELECT COUNT(*) FROM exams WHERE deleted_at IS NULL) as total_exams,
			(SELECT COUNT(*) FROM users WHERE deleted_at IS NULL AND role = 'user') as total_users,
			COUNT(r.id) as total_attempts,
			COALESCE(AVG(r.score), 0) as average_score,
			COALESCE(AVG(CASE WHEN r.passed = true THEN 1.0 ELSE 0.0 END) * 100, 0) as pass_rate,
			COALESCE(SUM(r.duration), 0) as total_time_spent,
			COALESCE(AVG(r.duration), 0) as average_duration
		FROM results r
	`).Row()

	err := row.Scan(
		&stats.TotalExams,
		&stats.TotalUsers,
		&stats.TotalAttempts,
		&stats.AverageScore,
		&stats.PassRate,
		&stats.TotalTimeSpent,
		&stats.AverageDuration,
	)

	return stats, err
}

func (s *ResultService) loadQuestionDetailsForResult(result *models.Result) error {
	// Get all question IDs from answers
	questionIDs := make([]uint, len(result.Answers))
	for i, answer := range result.Answers {
		questionIDs[i] = answer.QuestionID
	}

	// Load questions
	var questions []models.Question
	if err := s.db.Where("id IN ?", questionIDs).Find(&questions).Error; err != nil {
		return err
	}

	// Create question map for quick lookup
	questionMap := make(map[uint]models.Question)
	for _, q := range questions {
		questionMap[q.ID] = q
	}

	// Update result answers with question details and correct answers
	for i, answer := range result.Answers {
		if question, exists := questionMap[answer.QuestionID]; exists {
			// This would be used in the response conversion
			// We store the question reference for later use
			_ = question
		}
		result.Answers[i] = answer
	}

	return nil
}

