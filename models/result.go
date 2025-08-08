package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

type Answer struct {
	QuestionID      uint     `json:"question_id"`
	SelectedOptions []string `json:"selected_options"`
	IsCorrect       bool     `json:"is_correct"`
	Points          int      `json:"points"`
	TimeSpent       int      `json:"time_spent"` // in seconds
}

type Answers []Answer

func (a Answers) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *Answers) Scan(value interface{}) error {
	if value == nil {
		*a = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(bytes, a)
}

type Result struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	UserID       uint           `json:"user_id" gorm:"not null"`
	ExamID       uint           `json:"exam_id" gorm:"not null"`
	UserExamID   uint           `json:"user_exam_id" gorm:"not null"`
	Score        float64        `json:"score" gorm:"not null"`        // percentage score
	TotalPoints  int            `json:"total_points" gorm:"not null"` // points earned
	MaxPoints    int            `json:"max_points" gorm:"not null"`   // maximum possible points
	Passed       bool           `json:"passed" gorm:"default:false"`
	Answers      Answers        `json:"answers" gorm:"type:jsonb"`
	StartTime    time.Time      `json:"start_time" gorm:"not null"`
	EndTime      time.Time      `json:"end_time" gorm:"not null"`
	Duration     int            `json:"duration" gorm:"not null"` // in seconds
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`

	// Relationships
	User     User     `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Exam     Exam     `json:"exam,omitempty" gorm:"foreignKey:ExamID"`
	UserExam UserExam `json:"user_exam,omitempty" gorm:"foreignKey:UserExamID"`
}

type ResultResponse struct {
	ID          uint              `json:"id"`
	UserID      uint              `json:"user_id"`
	ExamID      uint              `json:"exam_id"`
	ExamTitle   string            `json:"exam_title"`
	Score       float64           `json:"score"`
	TotalPoints int               `json:"total_points"`
	MaxPoints   int               `json:"max_points"`
	Passed      bool              `json:"passed"`
	StartTime   time.Time         `json:"start_time"`
	EndTime     time.Time         `json:"end_time"`
	Duration    int               `json:"duration"`
	CreatedAt   time.Time         `json:"created_at"`
	Answers     []AnswerResponse  `json:"answers,omitempty"`
	User        *UserResponse     `json:"user,omitempty"`
}

type AnswerResponse struct {
	QuestionID      uint                `json:"question_id"`
	Question        *QuestionResponse   `json:"question,omitempty"`
	SelectedOptions []string            `json:"selected_options"`
	CorrectOptions  []string            `json:"correct_options,omitempty"`
	IsCorrect       bool                `json:"is_correct"`
	Points          int                 `json:"points"`
	TimeSpent       int                 `json:"time_spent"`
}

func (r *Result) ToResponse(includeAnswers bool, includeCorrectAnswers bool) ResultResponse {
	response := ResultResponse{
		ID:          r.ID,
		UserID:      r.UserID,
		ExamID:      r.ExamID,
		Score:       r.Score,
		TotalPoints: r.TotalPoints,
		MaxPoints:   r.MaxPoints,
		Passed:      r.Passed,
		StartTime:   r.StartTime,
		EndTime:     r.EndTime,
		Duration:    r.Duration,
		CreatedAt:   r.CreatedAt,
	}

	if r.Exam.Title != "" {
		response.ExamTitle = r.Exam.Title
	}

	if r.User.ID != 0 {
		userResp := r.User.ToResponse()
		response.User = &userResp
	}

	if includeAnswers {
		answers := make([]AnswerResponse, len(r.Answers))
		for i, ans := range r.Answers {
			answerResp := AnswerResponse{
				QuestionID:      ans.QuestionID,
				SelectedOptions: ans.SelectedOptions,
				IsCorrect:       ans.IsCorrect,
				Points:          ans.Points,
				TimeSpent:       ans.TimeSpent,
			}

			if includeCorrectAnswers {
				// This would need to be populated from the question data
				// For now, we'll leave it empty and populate it in the service layer
			}

			answers[i] = answerResp
		}
		response.Answers = answers
	}

	return response
}

type ExamStatistics struct {
	ExamID          uint    `json:"exam_id"`
	ExamTitle       string  `json:"exam_title"`
	TotalAttempts   int     `json:"total_attempts"`
	PassedAttempts  int     `json:"passed_attempts"`
	FailedAttempts  int     `json:"failed_attempts"`
	PassRate        float64 `json:"pass_rate"`
	AverageScore    float64 `json:"average_score"`
	HighestScore    float64 `json:"highest_score"`
	LowestScore     float64 `json:"lowest_score"`
	AverageDuration int     `json:"average_duration"` // in seconds
}

type UserStatistics struct {
	UserID          uint    `json:"user_id"`
	Username        string  `json:"username"`
	TotalExams      int     `json:"total_exams"`
	PassedExams     int     `json:"passed_exams"`
	FailedExams     int     `json:"failed_exams"`
	PassRate        float64 `json:"pass_rate"`
	AverageScore    float64 `json:"average_score"`
	HighestScore    float64 `json:"highest_score"`
	LowestScore     float64 `json:"lowest_score"`
	TotalTimeSpent  int     `json:"total_time_spent"` // in seconds
}

type QuestionStatistics struct {
	QuestionID      uint    `json:"question_id"`
	QuestionTitle   string  `json:"question_title"`
	TotalAttempts   int     `json:"total_attempts"`
	CorrectAttempts int     `json:"correct_attempts"`
	WrongAttempts   int     `json:"wrong_attempts"`
	SuccessRate     float64 `json:"success_rate"`
	AverageTimeSpent int    `json:"average_time_spent"` // in seconds
}

