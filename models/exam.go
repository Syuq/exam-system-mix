package models

import (
	"time"

	"gorm.io/gorm"
)

type ExamStatus string

const (
	ExamDraft     ExamStatus = "draft"
	ExamActive    ExamStatus = "active"
	ExamCompleted ExamStatus = "completed"
	ExamArchived  ExamStatus = "archived"
)

type Exam struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Title       string         `json:"title" gorm:"not null"`
	Description string         `json:"description" gorm:"type:text"`
	Duration    int            `json:"duration" gorm:"not null"` // in minutes
	TotalPoints int            `json:"total_points" gorm:"default:0"`
	PassScore   int            `json:"pass_score" gorm:"default:60"` // percentage
	Status      ExamStatus     `json:"status" gorm:"default:'draft'"`
	StartTime   *time.Time     `json:"start_time"`
	EndTime     *time.Time     `json:"end_time"`
	IsActive    bool           `json:"is_active" gorm:"default:true"`
	CreatedBy   uint           `json:"created_by"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// Relationships
	Creator       User           `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
	ExamQuestions []ExamQuestion `json:"exam_questions,omitempty" gorm:"foreignKey:ExamID"`
	UserExams     []UserExam     `json:"user_exams,omitempty" gorm:"foreignKey:ExamID"`
	Results       []Result       `json:"results,omitempty" gorm:"foreignKey:ExamID"`
}

type ExamQuestion struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	ExamID     uint      `json:"exam_id" gorm:"not null"`
	QuestionID uint      `json:"question_id" gorm:"not null"`
	Order      int       `json:"order" gorm:"not null"`
	Points     int       `json:"points" gorm:"default:1"`
	CreatedAt  time.Time `json:"created_at"`

	// Relationships
	Exam     Exam     `json:"exam,omitempty" gorm:"foreignKey:ExamID"`
	Question Question `json:"question,omitempty" gorm:"foreignKey:QuestionID"`
}

type UserExamStatus string

const (
	UserExamAssigned  UserExamStatus = "assigned"
	UserExamStarted   UserExamStatus = "started"
	UserExamCompleted UserExamStatus = "completed"
	UserExamExpired   UserExamStatus = "expired"
)

type UserExam struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	UserID      uint           `json:"user_id" gorm:"not null"`
	ExamID      uint           `json:"exam_id" gorm:"not null"`
	Status      UserExamStatus `json:"status" gorm:"default:'assigned'"`
	StartedAt   *time.Time     `json:"started_at"`
	CompletedAt *time.Time     `json:"completed_at"`
	ExpiresAt   *time.Time     `json:"expires_at"`
	AttemptCount int           `json:"attempt_count" gorm:"default:0"`
	MaxAttempts int            `json:"max_attempts" gorm:"default:1"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`

	// Relationships
	User   User   `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Exam   Exam   `json:"exam,omitempty" gorm:"foreignKey:ExamID"`
	Result Result `json:"result,omitempty" gorm:"foreignKey:UserExamID"`
}

type ExamResponse struct {
	ID          uint                   `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Duration    int                    `json:"duration"`
	TotalPoints int                    `json:"total_points"`
	PassScore   int                    `json:"pass_score"`
	Status      ExamStatus             `json:"status"`
	StartTime   *time.Time             `json:"start_time"`
	EndTime     *time.Time             `json:"end_time"`
	IsActive    bool                   `json:"is_active"`
	CreatedBy   uint                   `json:"created_by"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Questions   []QuestionResponse     `json:"questions,omitempty"`
	UserExam    *UserExamResponse      `json:"user_exam,omitempty"`
}

type UserExamResponse struct {
	ID           uint           `json:"id"`
	Status       UserExamStatus `json:"status"`
	StartedAt    *time.Time     `json:"started_at"`
	CompletedAt  *time.Time     `json:"completed_at"`
	ExpiresAt    *time.Time     `json:"expires_at"`
	AttemptCount int            `json:"attempt_count"`
	MaxAttempts  int            `json:"max_attempts"`
	TimeLeft     *int           `json:"time_left,omitempty"` // in seconds
}

func (e *Exam) ToResponse(includeQuestions bool, userExam *UserExam) ExamResponse {
	response := ExamResponse{
		ID:          e.ID,
		Title:       e.Title,
		Description: e.Description,
		Duration:    e.Duration,
		TotalPoints: e.TotalPoints,
		PassScore:   e.PassScore,
		Status:      e.Status,
		StartTime:   e.StartTime,
		EndTime:     e.EndTime,
		IsActive:    e.IsActive,
		CreatedBy:   e.CreatedBy,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}

	if includeQuestions {
		questions := make([]QuestionResponse, len(e.ExamQuestions))
		for i, eq := range e.ExamQuestions {
			questions[i] = eq.Question.ToResponse(false) // Don't include correct answers
		}
		response.Questions = questions
	}

	if userExam != nil {
		userExamResp := &UserExamResponse{
			ID:           userExam.ID,
			Status:       userExam.Status,
			StartedAt:    userExam.StartedAt,
			CompletedAt:  userExam.CompletedAt,
			ExpiresAt:    userExam.ExpiresAt,
			AttemptCount: userExam.AttemptCount,
			MaxAttempts:  userExam.MaxAttempts,
		}

		// Calculate time left if exam is started
		if userExam.StartedAt != nil && userExam.Status == UserExamStarted {
			examDuration := time.Duration(e.Duration) * time.Minute
			elapsed := time.Since(*userExam.StartedAt)
			timeLeft := int((examDuration - elapsed).Seconds())
			if timeLeft < 0 {
				timeLeft = 0
			}
			userExamResp.TimeLeft = &timeLeft
		}

		response.UserExam = userExamResp
	}

	return response
}

func (ue *UserExam) IsExpired() bool {
	if ue.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*ue.ExpiresAt)
}

func (ue *UserExam) CanStart() bool {
	return ue.Status == UserExamAssigned && !ue.IsExpired()
}

func (ue *UserExam) CanSubmit() bool {
	return ue.Status == UserExamStarted && !ue.IsExpired()
}

