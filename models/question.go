package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

type QuestionType string

const (
	MultipleChoice QuestionType = "multiple_choice"
	TrueFalse      QuestionType = "true_false"
)

type QuestionDifficulty string

const (
	Easy   QuestionDifficulty = "easy"
	Medium QuestionDifficulty = "medium"
	Hard   QuestionDifficulty = "hard"
)

type Option struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	IsCorrect bool   `json:"is_correct"`
}

type Options []Option

func (o Options) Value() (driver.Value, error) {
	return json.Marshal(o)
}

func (o *Options) Scan(value interface{}) error {
	if value == nil {
		*o = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(bytes, o)
}

type StringArray []string

func (s StringArray) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(bytes, s)
}

type Question struct {
	ID          uint               `json:"id" gorm:"primaryKey"`
	Title       string             `json:"title" gorm:"not null"`
	Content     string             `json:"content" gorm:"type:text;not null"`
	Type        QuestionType       `json:"type" gorm:"default:'multiple_choice'"`
	Difficulty  QuestionDifficulty `json:"difficulty" gorm:"default:'medium'"`
	Options     Options            `json:"options" gorm:"type:jsonb"`
	Tags        StringArray        `json:"tags" gorm:"type:jsonb"`
	Points      int                `json:"points" gorm:"default:1"`
	TimeLimit   int                `json:"time_limit" gorm:"default:60"` // in seconds
	Explanation string             `json:"explanation" gorm:"type:text"`
	IsActive    bool               `json:"is_active" gorm:"default:true"`
	CreatedBy   uint               `json:"created_by"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
	DeletedAt   gorm.DeletedAt     `json:"-" gorm:"index"`

	// Relationships
	Creator       User           `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
	ExamQuestions []ExamQuestion `json:"exam_questions,omitempty" gorm:"foreignKey:QuestionID"`
}

type QuestionResponse struct {
	ID          uint               `json:"id"`
	Title       string             `json:"title"`
	Content     string             `json:"content"`
	Type        QuestionType       `json:"type"`
	Difficulty  QuestionDifficulty `json:"difficulty"`
	Options     []OptionResponse   `json:"options"`
	Tags        []string           `json:"tags"`
	Points      int                `json:"points"`
	TimeLimit   int                `json:"time_limit"`
	Explanation string             `json:"explanation,omitempty"`
	IsActive    bool               `json:"is_active"`
	CreatedBy   uint               `json:"created_by"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

type OptionResponse struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	// IsCorrect is omitted for security reasons when serving to users
}

func (q *Question) ToResponse(includeCorrectAnswers bool) QuestionResponse {
	options := make([]OptionResponse, len(q.Options))
	for i, opt := range q.Options {
		options[i] = OptionResponse{
			ID:   opt.ID,
			Text: opt.Text,
		}
	}

	response := QuestionResponse{
		ID:         q.ID,
		Title:      q.Title,
		Content:    q.Content,
		Type:       q.Type,
		Difficulty: q.Difficulty,
		Options:    options,
		Tags:       []string(q.Tags),
		Points:     q.Points,
		TimeLimit:  q.TimeLimit,
		IsActive:   q.IsActive,
		CreatedBy:  q.CreatedBy,
		CreatedAt:  q.CreatedAt,
		UpdatedAt:  q.UpdatedAt,
	}

	if includeCorrectAnswers {
		response.Explanation = q.Explanation
	}

	return response
}

func (q *Question) GetCorrectAnswers() []string {
	var correctAnswers []string
	for _, opt := range q.Options {
		if opt.IsCorrect {
			correctAnswers = append(correctAnswers, opt.ID)
		}
	}
	return correctAnswers
}

func (q *Question) ValidateAnswer(selectedOptions []string) bool {
	correctAnswers := q.GetCorrectAnswers()
	
	if len(selectedOptions) != len(correctAnswers) {
		return false
	}

	correctMap := make(map[string]bool)
	for _, correct := range correctAnswers {
		correctMap[correct] = true
	}

	for _, selected := range selectedOptions {
		if !correctMap[selected] {
			return false
		}
	}

	return true
}

