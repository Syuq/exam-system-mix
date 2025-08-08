package models

import (
	"exam-system/config"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitDB() (*gorm.DB, error) {
	cfg := config.AppConfig.Database

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		cfg.Host, cfg.User, cfg.Password, cfg.Name, cfg.Port, cfg.SSLMode)

	var gormLogger logger.Interface
	if config.AppConfig.Server.GinMode == "debug" {
		gormLogger = logger.Default.LogMode(logger.Info)
	} else {
		gormLogger = logger.Default.LogMode(logger.Silent)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying sql.DB to configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	return db, nil
}

func RunMigrations(db *gorm.DB) error {
	// Auto migrate all models
	err := db.AutoMigrate(
		&User{},
		&Question{},
		&Exam{},
		&ExamQuestion{},
		&UserExam{},
		&Result{},
	)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Add indexes for better performance
	if err := addIndexes(db); err != nil {
		return fmt.Errorf("failed to add indexes: %w", err)
	}

	return nil
}

func addIndexes(db *gorm.DB) error {
	// User indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_users_role ON users(role)")

	// Question indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_questions_tags ON questions USING GIN(tags)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_questions_difficulty ON questions(difficulty)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_questions_type ON questions(type)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_questions_is_active ON questions(is_active)")

	// Exam indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_exams_status ON exams(status)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_exams_start_time ON exams(start_time)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_exams_end_time ON exams(end_time)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_exams_created_by ON exams(created_by)")

	// ExamQuestion indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_exam_questions_exam_id ON exam_questions(exam_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_exam_questions_question_id ON exam_questions(question_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_exam_questions_order ON exam_questions(exam_id, \"order\")")

	// UserExam indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_user_exams_user_id ON user_exams(user_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_user_exams_exam_id ON user_exams(exam_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_user_exams_status ON user_exams(status)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_user_exams_started_at ON user_exams(started_at)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_user_exams_expires_at ON user_exams(expires_at)")

	// Result indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_results_user_id ON results(user_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_results_exam_id ON results(exam_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_results_user_exam_id ON results(user_exam_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_results_score ON results(score)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_results_passed ON results(passed)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_results_created_at ON results(created_at)")

	return nil
}

